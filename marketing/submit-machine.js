// marketing/submit-machine.js — the strict scanner statechart (card
// redemption-submit-flow, run 20260905; design docs/qr-offline-redemption-handoff.md
// §8/§13/§19.1/§19.4 F1/F2/F3/F6, P-KR4; spike
// .night-crew/knowledge/spikes/activity-c-scanner-screen/redemption-submit-flow.md).
//
// ENGINE (operator-resolved at slate sign-off, overriding the spike extraction's
// hand-rolled recommendation): XState v5, vendored as lib/xstate.umd.min.js —
// the OVERLAY-REGION shape the spike's Addendum 3 proved: three parallel regions
// {connectivity, overlay, scan}. The finish-first prompt is the overlay region,
// so the scan region NEVER moves when the prompt appears — there is no go-back
// problem; DISMISS closes the overlay and the interrupted state is simply still
// there (F1's progress-preservation principle).
//
// STRICTNESS (the operator's call, verbatim in the slate): every (state, event)
// pair over the 20-event alphabet is a DECLARED decision. Each state carries its
// real transitions plus an enumerated `ignores:` list compiled to explicit no-op
// transitions — states declare even the events they deliberately ignore. An
// UNDECLARED pair falls to a per-region '*' wildcard that:
//   * mode 'throw' (every Node gate / test build): records the trip and THROWS —
//     the actor dies and the suite/fuzz's per-step liveness assertion reds;
//   * mode 'model' (the page's production default): captures {event, scan, conn,
//     overlay}, raises the internal UNEXPECTED event, and the scan region enters
//     the modeled, visible, retryable `unexpectedEvent` error state — loud,
//     logged (onTrip → console.error → log.js beacon), RETRY returns to the
//     interrupted state, NEXT_CUSTOMER resets. Never a dead actor at the window.
// Guard-fail fall-throughs inside a DECLARED pair are declared no-op tails ({}) —
// strictness is pair-level; payload validity is the page mapping's job (an
// unmapped resolver kind never reaches this machine).
//
// Three reachable-benign pairs are DECLARED ignores by name (spike Addendum 2 —
// each would brick the screen under throwing strictness): PROBE_TIMEOUT while
// already offline (the probe loop keeps timing out), RESUBSCRIBED while already
// online (Realtime re-subscribes spuriously), and a late RESOLVED landing after
// NEXT_CUSTOMER reset.
//
// DEPENDENCY-INJECTED like the marketing family — this module imports nothing.
// The XState primitives arrive as the first parameter (`window.XState` in the
// browser; Node `require()` of the SAME vendored UMD file in the gates), so the
// gates run against the exact shipped artifact.
//
// The behavior pin is tests/machine/: conformance.mjs (18 sequences, §19.4) +
// lockstep-fuzz.mjs (vs the spike's proven hand-rolled reference, with per-step
// liveness). Refactor only against them.

// ── the §19.3 event alphabet (the spike's, unchanged — 20 events) ───────────
export const SCAN_EVENTS = [
  'SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK', 'SUBMIT',
  'OVERRIDE_REQUEST', 'OVERRIDE_CONFIRM', 'OVERRIDE_CANCEL',
  'DISMISS', 'RETRY', 'NEXT_CUSTOMER',
  'SRV_REDEEMED', 'SRV_ALREADY_USED', 'SRV_EXPIRED', 'SRV_NOT_FOUND', 'SRV_ERROR',
];
export const CONN_EVENTS = ['CONN_DOWN', 'CONN_UP', 'PROBE_TIMEOUT', 'RESUBSCRIBED'];
export const EVENT_ALPHABET = [...SCAN_EVENTS, ...CONN_EVENTS];

// Internal only — raised by the model-mode wildcard, never sent from outside.
const UNEXPECTED = 'UNEXPECTED';

const SRV_VERDICTS = ['SRV_REDEEMED', 'SRV_ALREADY_USED', 'SRV_EXPIRED', 'SRV_NOT_FOUND', 'SRV_ERROR'];
const OVERRIDE_EVENTS = ['OVERRIDE_REQUEST', 'OVERRIDE_CONFIRM', 'OVERRIDE_CANCEL'];

// States a different-code scan may interrupt with the finish-first prompt (F6).
// Deliberately NOT here: scanning (that decode starts the session), submitting
// (an in-flight verdict must never be yanked — seq 17) and overrideConfirm
// (mid-confirmation scans are ignored). unexpectedEvent IS interruptible — it
// parks like a terminal card.
const INTERRUPTIBLE = [
  'resolving', 'offerReady', 'readyToSubmit', 'unknownCode', 'spentLocally',
  'expiredLocally', 'blockedOffline', 'overridePending', 'redeemed',
  'alreadyUsed', 'expired', 'notFound', 'error', 'unexpectedEvent',
];

const RESET_CTX = {
  sessionCode: null, resolutionKind: null, requiresOnline: false,
  overrideAvailable: false, unverifiedWarning: false, gateReturn: null,
  unexpectedInfo: null, unexpectedReturn: null,
};

export const SCAN_STATE_NAMES = [
  'idle', 'scanning', 'resolving', 'offerReady', 'readyToSubmit', 'unknownCode',
  'blockedOffline', 'overrideConfirm', 'overridePending', 'submitting',
  'redeemed', 'alreadyUsed', 'spentLocally', 'expiredLocally', 'expired',
  'notFound', 'error', 'unexpectedEvent',
];

/**
 * @param {object} X        {createMachine, createActor, assign, raise} — the
 *                          vendored XState v5 namespace.
 * @param {object} input    {canOverride} — the marketing-offline-override
 *                          entitlement (#12), decided at creation.
 * @param {object} effects  the effect sink: anything with .push(e). Emitted:
 *                          {type:'submitToServer'} and {type:'writeAttempt',
 *                          offline_override, unverified_code}.
 * @param {object} [opts]   {mode: 'model'|'throw' (default 'model'),
 *                          onTrip: (info)=>void}
 */
export function createSubmitMachine(X, input = {}, effects = [], opts = {}) {
  const { createMachine, createActor, assign, raise } = X;
  const mode = opts.mode || 'model';
  const onTrip = typeof opts.onTrip === 'function' ? opts.onTrip : null;
  const emit = (e) => effects.push(e);

  let declaredAlphabetPairs = 0;
  let stateCount = 0;

  // ── compiler ──────────────────────────────────────────────────────────────
  // Merge real transitions + the enumerated ignores into one `on` map, count
  // the declared pairs, and reject contradictions (an event listed as both a
  // transition and an ignore is a design error, loudly).
  const toArray = (t) => (Array.isArray(t) ? t : [t]);
  const norm = (t) => (typeof t === 'string' ? { target: t } : { ...t });

  function buildOn(stateName, on, ignores) {
    const out = {};
    for (const [ev, t] of Object.entries(on)) out[ev] = toArray(t).map(norm);
    for (const ev of ignores) {
      if (out[ev]) throw new Error(`submit-machine: ${stateName} declares ${ev} as BOTH transition and ignore`);
      out[ev] = [{}]; // declared no-op — the event is deliberately ignored here
    }
    for (const ev of Object.keys(out)) {
      if (EVENT_ALPHABET.includes(ev)) declaredAlphabetPairs += 1;
    }
    return out;
  }

  // Scan-region compiler: stamps the `sc` mirror, injects the UNEXPECTED
  // handler, applies the overlay gate (a modal blocks the controls beneath it —
  // every user-driven transition is guarded on ov === 'none', with a declared
  // no-op tail for the gated-out case), and appends a declared no-op tail to
  // any guard chain whose last entry is guarded (probe-proven: a guard-failed
  // explicit key otherwise falls through to the region wildcard).
  const GATE_BYPASS = ['NEXT_CUSTOMER', 'RESUBSCRIBED', UNEXPECTED];
  function scanState(name, on, ignores) {
    stateCount += 1;
    const merged = buildOn(name, on, ignores);
    if (name !== 'unexpectedEvent') {
      merged[UNEXPECTED] = [{ target: 'unexpectedEvent' }];
    }
    for (const [ev, list] of Object.entries(merged)) {
      let wrapped = list;
      if (!GATE_BYPASS.includes(ev)) {
        wrapped = list.map((def) => {
          const d = { ...def };
          // A declared no-op ({}) stays unguarded: ignoring an event does not
          // depend on whether the prompt is open.
          if (!d.target && !d.actions && !d.guard) return d;
          const g = d.guard;
          d.guard = g
            ? (args) => args.context.ov === 'none' && g(args)
            : ({ context }) => context.ov === 'none';
          return d;
        });
      }
      const last = wrapped[wrapped.length - 1];
      if (last && last.guard) wrapped = [...wrapped, {}]; // declared no-op tail
      merged[ev] = wrapped;
    }
    return [name, { entry: assign({ sc: name }), on: merged }];
  }

  function plainState(mirror, name, on, ignores) {
    stateCount += 1;
    return [name, { entry: assign({ [mirror]: name }), on: buildOn(name, on, ignores) }];
  }

  // ── the wildcard tripwire ────────────────────────────────────────────────
  const tripInfo = ({ context, event }) => ({
    event: event.type, scan: context.sc, conn: context.conn, overlay: context.ov,
  });
  const wildcard = mode === 'throw'
    ? {
      actions: [(args) => {
        const info = tripInfo(args);
        if (onTrip) onTrip(info);
        throw new Error(`submit-machine: UNDECLARED (state,event) pair — ${info.event} at scan:${info.scan}/conn:${info.conn}/overlay:${info.overlay}`);
      }],
    }
    : {
      actions: [
        (args) => { if (onTrip) onTrip(tripInfo(args)); },
        assign((args) => ({
          unexpectedInfo: tripInfo(args),
          unexpectedReturn: args.context.sc,
        })),
        raise({ type: UNEXPECTED }),
      ],
    };

  // ── shared scan-region pieces (the spike's, verbatim in behavior) ────────
  const resettable = { NEXT_CUSTOMER: { target: 'idle', actions: assign(RESET_CTX) } };
  const resolvedKind = assign({
    resolutionKind: ({ event }) => event.kind,
    requiresOnline: ({ event }) => !!event.requiresOnline,
  });
  const submitOrGate = (self) => ({
    SUBMIT: [
      { guard: ({ context }) => context.conn === 'online', target: 'submitting', actions: () => emit({ type: 'submitToServer' }) },
      {
        target: 'blockedOffline',
        actions: assign({
          overrideAvailable: ({ context }) => context.canOverride && !context.requiresOnline,
          gateReturn: self,
        }),
      },
    ],
  });

  // Ignore groups, named for the audit trail.
  const LATE_VERDICTS = SRV_VERDICTS;         // verdicts landing after a reset/park
  const SIBLING_CONN = CONN_EVENTS;           // the connectivity region's concern

  const machine = createMachine({
    id: 'submitFlow',
    type: 'parallel',
    context: {
      canOverride: false, ...input,
      conn: 'online', ov: 'none', sc: 'idle', ...RESET_CTX,
    },
    states: {
      connectivity: {
        initial: 'online',
        on: { '*': wildcard },
        states: Object.fromEntries([
          plainState('conn', 'online',
            { CONN_DOWN: 'offline', PROBE_TIMEOUT: 'offline' },
            ['CONN_UP', 'RESUBSCRIBED', // spurious re-subscribe while online — reachable-benign (Addendum 2)
              UNEXPECTED, ...SCAN_EVENTS]),
          plainState('conn', 'offline',
            { CONN_UP: 'stale' },
            ['CONN_DOWN',
              'PROBE_TIMEOUT', // the probe loop keeps timing out while offline — THE Addendum-2 brick, declared
              'RESUBSCRIBED',  // a subscribe race before the probe notices — conservative: stay offline until CONN_UP
              UNEXPECTED, ...SCAN_EVENTS]),
          plainState('conn', 'stale',
            { RESUBSCRIBED: 'online', CONN_DOWN: 'offline', PROBE_TIMEOUT: 'offline' },
            ['CONN_UP', UNEXPECTED, ...SCAN_EVENTS]),
        ]),
      },
      overlay: {
        initial: 'none',
        on: { '*': wildcard },
        states: Object.fromEntries([
          plainState('ov', 'none', {
            QR_DECODED: [
              { guard: ({ context, event }) => context.sessionCode !== null && event.code === context.sessionCode }, // same code: no-op / re-shows
              { guard: ({ context }) => context.sessionCode !== null && INTERRUPTIBLE.includes(context.sc), target: 'prompt' },
              {}, // fresh scan (scan region owns it) or a protected state (mid-submit / mid-confirmation) — declared no-op
            ],
            [UNEXPECTED]: {}, // stays closed; the scan region shows the error card
          }, [...SCAN_EVENTS.filter((e) => e !== 'QR_DECODED'), ...SIBLING_CONN]),
          plainState('ov', 'prompt', {
            DISMISS: 'none',        // the whole go-back problem, solved: close the overlay
            NEXT_CUSTOMER: 'none',  // scan region resets in the same broadcast
            QR_DECODED: [
              { guard: ({ context, event }) => event.code === context.sessionCode },
              {}, // a third code changes nothing — still finish the current customer
            ],
            [UNEXPECTED]: 'none',   // an unexpected trip supersedes the modal — the error card must be visible
          }, [...SCAN_EVENTS.filter((e) => !['DISMISS', 'NEXT_CUSTOMER', 'QR_DECODED'].includes(e)), ...SIBLING_CONN]),
        ]),
      },
      scan: {
        initial: 'idle',
        on: { '*': wildcard },
        states: Object.fromEntries([
          scanState('idle',
            { SCAN: 'scanning' },
            ['QR_DECODED', // late decode after reset — the page sends SCAN first
              'RESOLVED',  // late resolution after NEXT_CUSTOMER — reachable-benign (Addendum 2), declared
              'ORDER_OK', 'SUBMIT', ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY',
              'NEXT_CUSTOMER', // already idle
              ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('scanning',
            { QR_DECODED: { target: 'resolving', actions: assign({ sessionCode: ({ event }) => event.code }) } },
            ['SCAN', 'RESOLVED', 'ORDER_OK', 'SUBMIT', ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY',
              'NEXT_CUSTOMER', // no session yet
              ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('resolving', {
            RESOLVED: [
              { guard: ({ event }) => event.kind === 'offerReady', target: 'offerReady', actions: resolvedKind },
              { guard: ({ event }) => event.kind === 'unknownCode', target: 'unknownCode', actions: resolvedKind },
              { guard: ({ context, event }) => event.kind === 'spentLocally' && context.conn === 'online', target: 'offerReady', actions: resolvedKind }, // F3 online: server decides at submit
              { guard: ({ event }) => event.kind === 'spentLocally', target: 'spentLocally', actions: resolvedKind },
              { guard: ({ event }) => event.kind === 'expiredLocally', target: 'expiredLocally', actions: resolvedKind },
              // unmapped kind: declared no-op via the mechanical tail — the page
              // mapping is the validity boundary, not this machine
            ],
            ...resettable,
          }, ['SCAN',
            'QR_DECODED', // the overlay region owns the F6 reaction
            'ORDER_OK', 'SUBMIT', ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY',
            ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('offerReady',
            { ORDER_OK: 'readyToSubmit', ...resettable },
            ['SCAN', 'QR_DECODED', 'RESOLVED',
              'SUBMIT', // §13 double-entry: no order number, no submit (seq 15)
              ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY', ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('readyToSubmit',
            { ...submitOrGate('readyToSubmit'), ...resettable },
            ['SCAN', 'QR_DECODED', 'RESOLVED',
              'ORDER_OK', // a re-validated order number re-fires harmlessly
              ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY', ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('unknownCode',
            { ...submitOrGate('unknownCode'), ...resettable },
            ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK',
              ...OVERRIDE_EVENTS, // the override lives behind the gate, not here
              'DISMISS', 'RETRY', ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('blockedOffline', {
            OVERRIDE_REQUEST: {
              guard: ({ context }) => context.overrideAvailable,
              target: 'overrideConfirm',
              actions: assign({ unverifiedWarning: ({ context }) => context.resolutionKind === 'unknownCode' }),
            },
            // P-KR4 resume — gate-bypassed on purpose: the submit control
            // re-arms even UNDER the prompt, so a later DISMISS lands on a live
            // submit. Guards read the PRE-event conn mirror: 'stale' here means
            // "this same RESUBSCRIBED is taking connectivity to online".
            RESUBSCRIBED: [
              { guard: ({ context }) => context.conn === 'stale' && context.gateReturn === 'readyToSubmit', target: 'readyToSubmit' },
              { guard: ({ context }) => context.conn === 'stale' && context.gateReturn === 'unknownCode', target: 'unknownCode' },
            ],
            ...resettable,
          }, ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK',
            'SUBMIT', // re-tapping the gate changes nothing — reachability does
            'OVERRIDE_CONFIRM', 'OVERRIDE_CANCEL', 'DISMISS', 'RETRY',
            ...LATE_VERDICTS, 'CONN_DOWN', 'CONN_UP', 'PROBE_TIMEOUT']),
          scanState('overrideConfirm', {
            OVERRIDE_CONFIRM: {
              target: 'overridePending',
              actions: ({ context }) => emit({
                type: 'writeAttempt',
                offline_override: true,
                unverified_code: context.resolutionKind === 'unknownCode', // F2
              }),
            },
            OVERRIDE_CANCEL: 'blockedOffline',
          }, ['SCAN',
            'QR_DECODED', // mid-confirmation scans are ignored (deep-dive call)
            'RESOLVED', 'ORDER_OK', 'SUBMIT', 'OVERRIDE_REQUEST', 'DISMISS', 'RETRY',
            'NEXT_CUSTOMER', // cancel first, then move on (reference semantics)
            ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('overridePending',
            { ...resettable }, // terminal-class "queued — will verify on reconnect"
            ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK', 'SUBMIT',
              ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY', ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('submitting', {
            SRV_REDEEMED: 'redeemed', SRV_ALREADY_USED: 'alreadyUsed',
            SRV_EXPIRED: 'expired', SRV_NOT_FOUND: 'notFound', SRV_ERROR: 'error',
          }, ['SCAN',
            'QR_DECODED', // mid-submit protection: a scan cannot yank the in-flight verdict (seq 17)
            'RESOLVED', 'ORDER_OK',
            'SUBMIT', // double-tap while in flight
            ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY',
            'NEXT_CUSTOMER', // verdict in flight (reference semantics)
            ...SIBLING_CONN]),
          ...['redeemed', 'alreadyUsed', 'spentLocally', 'expiredLocally', 'expired', 'notFound'].map((name) =>
            scanState(name, { ...resettable },
              ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK', 'SUBMIT',
                ...OVERRIDE_EVENTS, 'DISMISS', 'RETRY', ...LATE_VERDICTS, ...SIBLING_CONN])),
          scanState('error',
            { RETRY: 'readyToSubmit', ...resettable }, // failures are retryable, session intact (UI-R6)
            ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK', 'SUBMIT',
              ...OVERRIDE_EVENTS, 'DISMISS', ...LATE_VERDICTS, ...SIBLING_CONN]),
          scanState('unexpectedEvent', {
            // RETRY returns to the interrupted state (the spike's generated-
            // branch precedent); with nothing captured, a clean reset.
            RETRY: [
              ...SCAN_STATE_NAMES.filter((n) => n !== 'unexpectedEvent').map((n) => ({
                guard: ({ context }) => context.unexpectedReturn === n, target: n,
              })),
              { target: 'idle', actions: assign(RESET_CTX) },
            ],
            ...resettable,
          }, ['SCAN', 'QR_DECODED', 'RESOLVED', 'ORDER_OK', 'SUBMIT',
            ...OVERRIDE_EVENTS, 'DISMISS', ...LATE_VERDICTS, ...SIBLING_CONN,
            UNEXPECTED]), // already showing the error card
        ]),
      },
    },
  });

  const actor = createActor(machine);
  let deadError = null;
  const changeListeners = new Set();
  actor.subscribe({
    next: (snap) => { for (const fn of changeListeners) { try { fn(snap); } catch (e) { /* listener errors never kill the actor */ } } },
    error: (err) => { deadError = err; },
  });
  actor.start();

  return {
    send: (type, payload = {}) => actor.send({ type, ...payload }),
    conn: () => actor.getSnapshot().value.connectivity,
    // the screen shows the prompt while the overlay is up; the scan region
    // beneath it never moved — that is the point of the overlay shape
    scan: () => {
      const v = actor.getSnapshot().value;
      return v.overlay === 'prompt' ? 'promptFinishCurrent' : v.scan;
    },
    flags: () => {
      const c = actor.getSnapshot().context;
      return { overrideAvailable: c.overrideAvailable, unverifiedWarning: c.unverifiedWarning };
    },
    ctx: () => actor.getSnapshot().context,
    alive: () => deadError === null && actor.getSnapshot().status === 'active',
    lastError: () => deadError,
    onChange: (fn) => { changeListeners.add(fn); return () => changeListeners.delete(fn); },
    declaredPairs: () => ({ alphabetPairs: declaredAlphabetPairs, states: stateCount, alphabet: EVENT_ALPHABET.length }),
  };
}
