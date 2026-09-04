// machine-handrolled.mjs — the no-dependency candidate: a parallel-region
// scanner machine in plain ES, written to HQ conventions (vanilla, no build,
// no framework). Two orthogonal regions ({connectivity, scan}) receive every
// event; cross-region guards read the sibling region directly. This file IS
// the owned-code price of the hand-rolled door — measured by the spike.
//
// Revised in the missing-states deep dive (same sitting): session lifecycle
// (NEXT_CUSTOMER from every parked state, full context clear), the P-KR4 gate
// resume on reachability return, prompt dismiss-with-memory, mid-submit scan
// protection, and error RETRY. Suite: js/conformance.mjs (18 sequences).

// States a different-code scan may interrupt with the finish-first prompt
// (F6). Deliberately NOT here: scanning (that decode starts the session),
// submitting (an in-flight verdict must never be yanked — seq 17) and
// overrideConfirm (mid-confirmation scans are ignored).
const INTERRUPTIBLE = new Set([
  'resolving', 'offerReady', 'readyToSubmit', 'unknownCode', 'spentLocally',
  'expiredLocally', 'blockedOffline', 'overridePending', 'redeemed',
  'alreadyUsed', 'expired', 'notFound', 'error',
]);

// States NEXT_CUSTOMER resets from — everywhere a session can park.
// Deliberately NOT here: idle, scanning (no session yet), submitting (verdict
// in flight) and overrideConfirm (cancel first, then move on).
const RESETTABLE = new Set([...INTERRUPTIBLE, 'promptFinishCurrent']);

export function createHandrolled(input, effects) {
  const ctx = {
    canOverride: false, ...input,
    resolutionKind: null, requiresOnline: false, sessionCode: null,
    overrideAvailable: false, unverifiedWarning: false,
    gateReturn: null,     // pre-gate state, resumed when reachability returns (P-KR4)
    promptReturn: null,   // interrupted state, resumed on DISMISS (F6)
  };
  const state = { connectivity: 'online', scan: 'idle' };
  const online = () => state.connectivity === 'online';
  const emit = (e) => effects.push(e);

  function resetSession() {
    ctx.sessionCode = null; ctx.resolutionKind = null; ctx.requiresOnline = false;
    ctx.overrideAvailable = false; ctx.unverifiedWarning = false;
    ctx.gateReturn = null; ctx.promptReturn = null;
    state.scan = 'idle';
  }

  // Region A — connectivity (§19.1): online | probing | offline | syncing |
  // stale. The spike models the states the conformance contract exercises;
  // probing and syncing join in the card (refinements of the same region).
  function connectivity(t) {
    const s = state.connectivity;
    if (t === 'CONN_DOWN' || t === 'PROBE_TIMEOUT') state.connectivity = 'offline';
    else if (t === 'CONN_UP' && s === 'offline') state.connectivity = 'stale'; // online-but-not-refetched (§7)
    else if (t === 'RESUBSCRIBED' && s === 'stale') state.connectivity = 'online';
  }

  // Region B — scan / redemption flow.
  function scan(t, p = {}) {
    const s = state.scan;

    // Session lifecycle — cross-cutting.
    if (t === 'NEXT_CUSTOMER' && RESETTABLE.has(s)) { resetSession(); return; }

    // F6 — session dedupe, cross-cutting: any mid-session decode.
    if (t === 'QR_DECODED' && ctx.sessionCode !== null && s !== 'scanning') {
      if (p.code === ctx.sessionCode) return;          // no-op / re-shows current
      if (s === 'promptFinishCurrent') return;          // a third code changes nothing
      if (!INTERRUPTIBLE.has(s)) return;                // submitting/overrideConfirm: protected (seq 17)
      ctx.promptReturn = s;
      state.scan = 'promptFinishCurrent';               // finish this customer first
      return;
    }

    switch (s) {
      case 'idle':
        if (t === 'SCAN') state.scan = 'scanning';
        break;
      case 'scanning':
        if (t === 'QR_DECODED') { ctx.sessionCode = p.code; state.scan = 'resolving'; }
        break;
      case 'resolving':
        if (t === 'RESOLVED') {
          ctx.resolutionKind = p.kind;
          ctx.requiresOnline = !!p.requiresOnline;
          if (p.kind === 'offerReady') state.scan = 'offerReady';
          else if (p.kind === 'unknownCode') state.scan = 'unknownCode';
          else if (p.kind === 'spentLocally') state.scan = online() ? 'offerReady' : 'spentLocally'; // F3
          else if (p.kind === 'expiredLocally') state.scan = 'expiredLocally';
        }
        break;
      case 'offerReady':
        if (t === 'ORDER_OK') state.scan = 'readyToSubmit';
        break;
      case 'readyToSubmit':
      case 'unknownCode':
        if (t === 'SUBMIT') {
          if (online()) { emit({ type: 'submitToServer' }); state.scan = 'submitting'; }
          else {
            // F1: not-online (offline OR stale) routes to the gate; §8 decides
            // whether the override is even offered. Remember where to resume.
            ctx.overrideAvailable = ctx.canOverride && !ctx.requiresOnline;
            ctx.gateReturn = s;
            state.scan = 'blockedOffline';
          }
        }
        break;
      case 'blockedOffline':
        if (t === 'OVERRIDE_REQUEST' && ctx.overrideAvailable) {
          ctx.unverifiedWarning = ctx.resolutionKind === 'unknownCode'; // F2 wording
          state.scan = 'overrideConfirm';
        } else if (t === 'RESUBSCRIBED' && online()) {
          // P-KR4: reachability returned (connectivity ran first this event) —
          // the submit control re-arms on its own, no manual refresh.
          state.scan = ctx.gateReturn;
        }
        break;
      case 'overrideConfirm':
        if (t === 'OVERRIDE_CONFIRM') {
          emit({
            type: 'writeAttempt',
            offline_override: true,
            unverified_code: ctx.resolutionKind === 'unknownCode', // F2
          });
          state.scan = 'overridePending';
        } else if (t === 'OVERRIDE_CANCEL') state.scan = 'blockedOffline';
        break;
      case 'submitting':
        if (t === 'SRV_REDEEMED') state.scan = 'redeemed';
        else if (t === 'SRV_ALREADY_USED') state.scan = 'alreadyUsed';
        else if (t === 'SRV_EXPIRED') state.scan = 'expired';
        else if (t === 'SRV_NOT_FOUND') state.scan = 'notFound';
        else if (t === 'SRV_ERROR') state.scan = 'error';
        break;
      case 'error':
        if (t === 'RETRY') state.scan = 'readyToSubmit'; // failures are retryable; session intact
        break;
      case 'promptFinishCurrent':
        if (t === 'DISMISS') {
          // Back to the interrupted state — unless it was the gate and
          // reachability returned while prompting; then resume past it.
          state.scan = (ctx.promptReturn === 'blockedOffline' && online())
            ? ctx.gateReturn
            : ctx.promptReturn;
        }
        break;
      default:
        break;
    }
  }

  return {
    send(t, p) { connectivity(t, p); scan(t, p); },  // broadcast — both regions see every event
    conn: () => state.connectivity,
    scan: () => state.scan,
    flags: () => ({ overrideAvailable: ctx.overrideAvailable, unverifiedWarning: ctx.unverifiedWarning }),
  };
}
