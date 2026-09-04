// machine-xstate-overlay.mjs — the third candidate, from the operator's
// history-state challenge (ledger Addendum 3): instead of modeling the
// finish-first prompt as a scan state (flat) or as a sibling-of-a-compound
// with history, make it a third parallel OVERLAY region. The scan region then
// NEVER MOVES when the prompt appears — there is nothing to remember and no
// go-back problem at all. DISMISS is `overlay: prompt → none`.
//
// What the shape costs instead (the relocated friction, kept honest):
//   - a modal must block the controls beneath it, so every user-driven scan
//     transition is gated on `overlay === none` — injected mechanically by
//     gateOn(); NEXT_CUSTOMER and RESUBSCRIBED deliberately bypass the gate
//     (reset and the P-KR4 auto-resume work THROUGH the prompt — which also
//     dissolves the flat model's two "reconnected while prompting" special
//     branches);
//   - the overlay's open guard needs the scan region's state, and the gate
//     needs the overlay's — TWO cross-region mirrors (`sc`, `ov`) where the
//     flat model needed one.
//
// The observable adapter is unchanged: scan() reports what the SCREEN shows —
// the prompt when the overlay is up, the (unmoved) scan state otherwise. The
// 18-sequence suite and the lockstep fuzzer decide whether this is a drop-in
// re-architecture of the same behavior.

import { createMachine, createActor, assign } from 'xstate';

const INTERRUPTIBLE = [
  'resolving', 'offerReady', 'readyToSubmit', 'unknownCode', 'spentLocally',
  'expiredLocally', 'blockedOffline', 'overridePending', 'redeemed',
  'alreadyUsed', 'expired', 'notFound', 'error',
];

const RESET_CTX = {
  sessionCode: null, resolutionKind: null, requiresOnline: false,
  overrideAvailable: false, unverifiedWarning: false, gateReturn: null,
};

export function createXstateOverlay(input, effects) {
  const emit = (e) => effects.push(e);

  // Gate injection: wrap every transition's guard with `ov === 'none'`.
  const overlayGate = (transitions) => {
    const wrap = (t) => {
      const def = typeof t === 'string' ? { target: t } : { ...t };
      const g = def.guard;
      def.guard = g
        ? (args) => args.context.ov === 'none' && g(args)
        : ({ context }) => context.ov === 'none';
      return def;
    };
    return Array.isArray(transitions) ? transitions.map(wrap) : [wrap(transitions)];
  };
  const gateOn = (on, bypass = ['NEXT_CUSTOMER', 'RESUBSCRIBED']) =>
    Object.fromEntries(Object.entries(on).map(([ev, t]) =>
      (bypass.includes(ev) ? [ev, t] : [ev, overlayGate(t)])));

  // Every scan child stamps itself into the `sc` mirror (the overlay's open
  // guard cannot see the sibling region — same mirror pattern as `conn`).
  const st = (name, on) => [name, { entry: assign({ sc: name }), on: gateOn(on) }];

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

  const machine = createMachine({
    id: 'scannerOverlay',
    type: 'parallel',
    context: {
      canOverride: false, ...input,
      conn: 'online', ov: 'none', sc: 'idle', ...RESET_CTX,
    },
    states: {
      connectivity: {
        initial: 'online',
        states: {
          online:  { entry: assign({ conn: 'online' }),  on: { CONN_DOWN: 'offline', PROBE_TIMEOUT: 'offline' } },
          offline: { entry: assign({ conn: 'offline' }), on: { CONN_UP: 'stale' } },
          stale:   { entry: assign({ conn: 'stale' }),   on: { RESUBSCRIBED: 'online', CONN_DOWN: 'offline', PROBE_TIMEOUT: 'offline' } },
        },
      },
      overlay: {
        initial: 'none',
        states: {
          none: {
            entry: assign({ ov: 'none' }),
            on: {
              QR_DECODED: [
                { guard: ({ context, event }) => context.sessionCode !== null && event.code === context.sessionCode }, // same code: no-op / re-shows
                { guard: ({ context }) => context.sessionCode !== null && INTERRUPTIBLE.includes(context.sc), target: 'prompt' },
              ],
            },
          },
          prompt: {
            entry: assign({ ov: 'prompt' }),
            on: {
              DISMISS: 'none',        // the whole go-back problem, solved: close the overlay
              NEXT_CUSTOMER: 'none',  // scan region resets in the same broadcast
              QR_DECODED: [
                { guard: ({ context, event }) => event.code === context.sessionCode },
                {}, // a third code changes nothing — still finish the current customer
              ],
            },
          },
        },
      },
      scan: {
        initial: 'idle',
        states: Object.fromEntries([
          st('idle', { SCAN: 'scanning' }),
          st('scanning', {
            QR_DECODED: { target: 'resolving', actions: assign({ sessionCode: ({ event }) => event.code }) },
          }),
          st('resolving', {
            RESOLVED: [
              { guard: ({ event }) => event.kind === 'offerReady', target: 'offerReady', actions: resolvedKind },
              { guard: ({ event }) => event.kind === 'unknownCode', target: 'unknownCode', actions: resolvedKind },
              { guard: ({ context, event }) => event.kind === 'spentLocally' && context.conn === 'online', target: 'offerReady', actions: resolvedKind },
              { guard: ({ event }) => event.kind === 'spentLocally', target: 'spentLocally', actions: resolvedKind },
              { guard: ({ event }) => event.kind === 'expiredLocally', target: 'expiredLocally', actions: resolvedKind },
            ],
            ...resettable,
          }),
          st('offerReady', { ORDER_OK: 'readyToSubmit', ...resettable }),
          st('readyToSubmit', { ...submitOrGate('readyToSubmit'), ...resettable }),
          st('unknownCode', { ...submitOrGate('unknownCode'), ...resettable }),
          st('blockedOffline', {
            OVERRIDE_REQUEST: {
              guard: ({ context }) => context.overrideAvailable,
              target: 'overrideConfirm',
              actions: assign({ unverifiedWarning: ({ context }) => context.resolutionKind === 'unknownCode' }),
            },
            // P-KR4 resume — bypasses the overlay gate on purpose: the gate
            // re-arms UNDER the prompt, so a later DISMISS lands on a live
            // submit (this replaces the flat model's two special branches).
            RESUBSCRIBED: [
              { guard: ({ context }) => context.conn === 'stale' && context.gateReturn === 'readyToSubmit', target: 'readyToSubmit' },
              { guard: ({ context }) => context.conn === 'stale' && context.gateReturn === 'unknownCode', target: 'unknownCode' },
            ],
            ...resettable,
          }),
          st('overrideConfirm', {
            OVERRIDE_CONFIRM: {
              target: 'overridePending',
              actions: ({ context }) => emit({
                type: 'writeAttempt',
                offline_override: true,
                unverified_code: context.resolutionKind === 'unknownCode',
              }),
            },
            OVERRIDE_CANCEL: 'blockedOffline',
          }),
          st('overridePending', { ...resettable }),
          st('submitting', {
            SRV_REDEEMED: 'redeemed', SRV_ALREADY_USED: 'alreadyUsed',
            SRV_EXPIRED: 'expired', SRV_NOT_FOUND: 'notFound', SRV_ERROR: 'error',
          }),
          st('redeemed', { ...resettable }),
          st('alreadyUsed', { ...resettable }),
          st('spentLocally', { ...resettable }),
          st('expiredLocally', { ...resettable }),
          st('expired', { ...resettable }),
          st('notFound', { ...resettable }),
          st('error', { RETRY: 'readyToSubmit', ...resettable }),
        ]),
      },
    },
  });

  const actor = createActor(machine);
  actor.start();

  return {
    send: (type, payload = {}) => actor.send({ type, ...payload }),
    conn: () => actor.getSnapshot().value.connectivity,
    // the screen shows the prompt while the overlay is up; the scan region
    // beneath it never moved — that is the whole point of the shape
    scan: () => {
      const v = actor.getSnapshot().value;
      return v.overlay === 'prompt' ? 'promptFinishCurrent' : v.scan;
    },
    flags: () => {
      const c = actor.getSnapshot().context;
      return { overrideAvailable: c.overrideAvailable, unverifiedWarning: c.unverifiedWarning };
    },
  };
}
