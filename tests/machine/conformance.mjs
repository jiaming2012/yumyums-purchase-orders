// tests/machine/conformance.mjs — THE CARD'S GATE, sequence half (card
// redemption-submit-flow, run 20260905). The 18 sequences transcribed from the
// §19.4 acceptance criteria, inherited VERBATIM in behavior from the spike
// (.night-crew/spikes/activity-c-scanner-screen/redemption-submit-flow/js/
// conformance.mjs — the spike's throwaway scripts are not load-bearing; this
// repo-owned copy is what the extraction record says the card must inherit).
// Machine-agnostic on purpose: the machine choice can never silently change
// behavior. Run via ./run-conformance.mjs against the PRODUCTION
// marketing/submit-machine.js on the VENDORED lib/xstate.umd.min.js.
//
// conformance.mjs — the SHARED behavioral contract both machine candidates
// must pass, transcribed from the §19.4 acceptance criteria (F1/F2/F3/F6, the
// §8 requires_online branch, and stale-routes-like-offline). Machine-agnostic:
// a candidate supplies createMachine(input, effects) returning
//   { send(type, payload?), conn() -> string, scan() -> string,
//     flags() -> {overrideAvailable, unverifiedWarning} }
//
// Event protocol (from §19.3): SCAN, QR_DECODED{code}, RESOLVED{kind,
// requiresOnline?}, ORDER_OK, SUBMIT, OVERRIDE_REQUEST, OVERRIDE_CONFIRM,
// SRV_REDEEMED, SRV_ALREADY_USED, CONN_DOWN, CONN_UP, RESUBSCRIBED.

const seqs = [];
const seq = (name, fn) => seqs.push({ name, fn });

function expectEq(actual, want, what) {
  if (actual !== want) throw new Error(`${what}: got ${JSON.stringify(actual)}, want ${JSON.stringify(want)}`);
}

// -- 1 -----------------------------------------------------------------------
seq('F1-parallel — connectivity change never resets scan progress', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'offerReady' });
  expectEq(m.scan(), 'offerReady', 'scan before drop');
  expectEq(m.conn(), 'online', 'conn before drop');
  m.send('CONN_DOWN');
  expectEq(m.conn(), 'offline', 'conn after drop');
  expectEq(m.scan(), 'offerReady', 'scan after drop (must NOT reset)');
});

// -- 2 -----------------------------------------------------------------------
seq('F1-gate — offline SUBMIT routes to the gate, never the server', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'offerReady' });
  m.send('ORDER_OK'); m.send('CONN_DOWN'); m.send('SUBMIT');
  expectEq(m.scan(), 'blockedOffline', 'scan after offline SUBMIT');
  expectEq(effects.filter((e) => e.type === 'submitToServer').length, 0, 'server submits while offline');
});

// -- 3 -----------------------------------------------------------------------
seq('F1-stale — stale routes SUBMIT exactly like offline', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'offerReady' });
  m.send('ORDER_OK'); m.send('CONN_DOWN'); m.send('CONN_UP');
  expectEq(m.conn(), 'stale', 'conn after reconnect without refetch');
  m.send('SUBMIT');
  expectEq(m.scan(), 'blockedOffline', 'scan after stale SUBMIT');
  expectEq(effects.filter((e) => e.type === 'submitToServer').length, 0, 'server submits while stale');
});

// -- 4 -----------------------------------------------------------------------
seq('F2-no-perm — offline unknownCode without permission is blocked, no override', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('CONN_DOWN');
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'unknownCode' });
  expectEq(m.scan(), 'unknownCode', 'resolution');
  m.send('SUBMIT');
  expectEq(m.scan(), 'blockedOffline', 'blocked');
  expectEq(m.flags().overrideAvailable, false, 'override offered without permission');
  m.send('OVERRIDE_REQUEST');
  expectEq(m.scan(), 'blockedOffline', 'override path without permission must be a no-op');
});

// -- 5 -----------------------------------------------------------------------
seq('F2-with-perm — override on unknownCode warns unverifiable + writes both flags', (create) => {
  const effects = [];
  const m = create({ canOverride: true }, effects);
  m.send('CONN_DOWN');
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'unknownCode' });
  m.send('SUBMIT');
  expectEq(m.scan(), 'blockedOffline', 'gate first');
  expectEq(m.flags().overrideAvailable, true, 'override offered with permission');
  m.send('OVERRIDE_REQUEST');
  expectEq(m.scan(), 'overrideConfirm', 'confirmation step');
  expectEq(m.flags().unverifiedWarning, true, 'confirmation must state neither offer nor prior use verifiable');
  m.send('OVERRIDE_CONFIRM');
  expectEq(m.scan(), 'overridePending', 'queued for sync arbitration');
  const w = effects.filter((e) => e.type === 'writeAttempt');
  expectEq(w.length, 1, 'attempt writes');
  expectEq(w[0].offline_override, true, 'offline_override flag');
  expectEq(w[0].unverified_code, true, 'unverified_code flag (F2)');
});

// -- 6 -----------------------------------------------------------------------
seq('F3-offline — locally-spent rejects immediately, no submit path', (create) => {
  const effects = [];
  const m = create({ canOverride: true }, effects);
  m.send('CONN_DOWN');
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'spentLocally' });
  expectEq(m.scan(), 'spentLocally', 'advisory reject');
  m.send('SUBMIT');
  expectEq(m.scan(), 'spentLocally', 'SUBMIT from the reject must not move');
  expectEq(effects.length, 0, 'effects on an offline spent reject');
});

// -- 7 -----------------------------------------------------------------------
seq('F3-online — the server wins: local flag does not reject, SRV_ALREADY_USED does', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'spentLocally' });
  expectEq(m.scan(), 'offerReady', 'online: do NOT reject on the stale local flag');
  m.send('ORDER_OK'); m.send('SUBMIT');
  expectEq(m.scan(), 'submitting', 'online submit proceeds');
  expectEq(effects.filter((e) => e.type === 'submitToServer').length, 1, 'server submit effect');
  m.send('SRV_ALREADY_USED');
  expectEq(m.scan(), 'alreadyUsed', 'server verdict is authoritative');
});

// -- 8 -----------------------------------------------------------------------
seq('F6-same-code — in-session re-scan is a no-op; after terminal it re-shows', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'offerReady' });
  const effectsBefore = effects.length;
  m.send('QR_DECODED', { code: 'X' });
  expectEq(m.scan(), 'offerReady', 'mid-session re-scan must not move the state');
  expectEq(effects.length, effectsBefore, 'mid-session re-scan must not emit');
  m.send('ORDER_OK'); m.send('SUBMIT'); m.send('SRV_REDEEMED');
  expectEq(m.scan(), 'redeemed', 'terminal');
  m.send('QR_DECODED', { code: 'X' });
  expectEq(m.scan(), 'redeemed', 'post-terminal re-scan re-shows the result');
});

// -- 9 -----------------------------------------------------------------------
seq('F6-different-code — a second customer mid-session prompts to finish first', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' }); m.send('RESOLVED', { kind: 'offerReady' });
  m.send('QR_DECODED', { code: 'Y' });
  expectEq(m.scan(), 'promptFinishCurrent', 'different code mid-session prompts');
});

// -- 10 ----------------------------------------------------------------------
seq('§8-high-value — requires_online campaign refuses override even with permission', (create) => {
  const effects = [];
  const m = create({ canOverride: true }, effects);
  m.send('SCAN'); m.send('QR_DECODED', { code: 'X' });
  m.send('RESOLVED', { kind: 'offerReady', requiresOnline: true });
  m.send('ORDER_OK'); m.send('CONN_DOWN'); m.send('SUBMIT');
  expectEq(m.scan(), 'blockedOffline', 'blocked');
  expectEq(m.flags().overrideAvailable, false, 'override on a requires_online campaign');
  m.send('OVERRIDE_REQUEST');
  expectEq(m.scan(), 'blockedOffline', 'even a manager cannot force it (§8)');
});

// ---------------------------------------------------------------------------
// Sequences 11–18 came out of the missing-states deep dive (operator-directed,
// same sitting): the first ten never send NEXT_CUSTOMER, never recover from
// the offline gate, and never scan mid-submit — and the two candidates
// silently diverged in exactly that unpinned territory (lockstep-fuzz.mjs
// found the connectivity half; code diff found the session-lifecycle half).
// These pin the session lifecycle, the P-KR4 live recovery, and the
// mid-submit protection.
// ---------------------------------------------------------------------------

const play = (m, steps) => { for (const [t, p] of steps) m.send(t, p || {}); };
const TO_OFFER = [['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'offerReady' }]];

// -- 11 ----------------------------------------------------------------------
seq('Reset — every parked or terminal state clears fully for the next customer', (create) => {
  const cases = {
    redeemed:       { drive: [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_REDEEMED']] },
    alreadyUsed:    { drive: [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_ALREADY_USED']] },
    expired:        { drive: [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_EXPIRED']] },
    notFound:       { drive: [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_NOT_FOUND']] },
    error:          { drive: [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_ERROR']] },
    spentLocally:   { drive: [['CONN_DOWN'], ['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'spentLocally' }]] },
    expiredLocally: { drive: [['CONN_DOWN'], ['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'expiredLocally' }]] },
    blockedOffline: { drive: [...TO_OFFER, ['ORDER_OK'], ['CONN_DOWN'], ['SUBMIT']] },
    overridePending: {
      canOverride: true,
      drive: [['CONN_DOWN'], ['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'unknownCode' }],
              ['SUBMIT'], ['OVERRIDE_REQUEST'], ['OVERRIDE_CONFIRM']],
    },
  };
  for (const [want, c] of Object.entries(cases)) {
    const effects = [];
    const m = create({ canOverride: c.canOverride ?? false }, effects);
    play(m, c.drive);
    expectEq(m.scan(), want, `drive to ${want}`);
    m.send('NEXT_CUSTOMER');
    expectEq(m.scan(), 'idle', `NEXT_CUSTOMER from ${want} must reset to idle`);
    expectEq(m.flags().overrideAvailable, false, `overrideAvailable leaked past reset from ${want}`);
    expectEq(m.flags().unverifiedWarning, false, `unverifiedWarning leaked past reset from ${want}`);
    m.send('SCAN'); m.send('QR_DECODED', { code: 'Y' });
    expectEq(m.scan(), 'resolving', `fresh session on a new code after ${want} (stale session must not prompt)`);
  }
});

// -- 12 ----------------------------------------------------------------------
seq('P-KR4 — the gate resumes submit on its own when reachability returns', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, [...TO_OFFER, ['ORDER_OK'], ['CONN_DOWN'], ['SUBMIT']]);
  expectEq(m.scan(), 'blockedOffline', 'gated');
  m.send('CONN_UP');
  expectEq(m.conn(), 'stale', 'reconnected but not refetched');
  expectEq(m.scan(), 'blockedOffline', 'stale must NOT resume the gate');
  m.send('RESUBSCRIBED');
  expectEq(m.conn(), 'online', 'refetched');
  expectEq(m.scan(), 'readyToSubmit', 'gate must resume the pre-gate state without manual refresh');
  m.send('SUBMIT');
  expectEq(m.scan(), 'submitting', 'resumed submit reaches the server');
  expectEq(effects.filter((e) => e.type === 'submitToServer').length, 1, 'server submits');
});

// -- 13 ----------------------------------------------------------------------
seq('P-KR4 — an unknown-code gate resumes too, and the server then decides', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, [['CONN_DOWN'], ['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'unknownCode' }], ['SUBMIT']]);
  expectEq(m.scan(), 'blockedOffline', 'gated');
  play(m, [['CONN_UP'], ['RESUBSCRIBED']]);
  expectEq(m.scan(), 'unknownCode', 'resumes the pre-gate resolution');
  m.send('SUBMIT');
  expectEq(m.scan(), 'submitting', 'online unknown code submits — the server decides');
  m.send('SRV_NOT_FOUND');
  expectEq(m.scan(), 'notFound', 'server verdict lands');
});

// -- 14 ----------------------------------------------------------------------
seq('F6 — the finish-first prompt can be dismissed (progress kept) or cleared', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, [...TO_OFFER, ['QR_DECODED', { code: 'Y' }]]);
  expectEq(m.scan(), 'promptFinishCurrent', 'different code prompts');
  m.send('DISMISS');
  expectEq(m.scan(), 'offerReady', 'dismiss returns to the interrupted state — progress preserved');
  m.send('QR_DECODED', { code: 'X' });
  expectEq(m.scan(), 'offerReady', 'session X still owns the screen (same-code no-op intact)');
  m.send('QR_DECODED', { code: 'Y' });
  expectEq(m.scan(), 'promptFinishCurrent', 'prompts again');
  m.send('NEXT_CUSTOMER');
  expectEq(m.scan(), 'idle', 'clearing from the prompt resets');
  play(m, [['SCAN'], ['QR_DECODED', { code: 'Y' }]]);
  expectEq(m.scan(), 'resolving', 'the new customer scans fresh');
});

// -- 15 ----------------------------------------------------------------------
seq('§13 double-entry — no path to submit without the order number', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, TO_OFFER);
  m.send('SUBMIT');
  expectEq(m.scan(), 'offerReady', 'SUBMIT before ORDER_OK must be a no-op');
  expectEq(effects.length, 0, 'no effect may fire without the order number');
});

// -- 16 ----------------------------------------------------------------------
seq('Hardening — a dead probe while stale reads offline; a cancelled override returns to the gate', (create) => {
  const effects = [];
  const m = create({ canOverride: true }, effects);
  play(m, [['CONN_DOWN'], ['CONN_UP']]);
  expectEq(m.conn(), 'stale', 'reconnected, not refetched');
  m.send('PROBE_TIMEOUT');
  expectEq(m.conn(), 'offline', 'a dead probe while stale must read offline, not linger stale');
  play(m, [['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'unknownCode' }], ['SUBMIT'], ['OVERRIDE_REQUEST']]);
  expectEq(m.scan(), 'overrideConfirm', 'confirming');
  m.send('OVERRIDE_CANCEL');
  expectEq(m.scan(), 'blockedOffline', 'cancel returns to the gate, nothing written');
  expectEq(effects.filter((e) => e.type === 'writeAttempt').length, 0, 'attempt writes after a cancel');
});

// -- 17 ----------------------------------------------------------------------
seq('Mid-submit protection — a scan cannot yank an in-flight verdict', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, [...TO_OFFER, ['ORDER_OK'], ['SUBMIT']]);
  expectEq(m.scan(), 'submitting', 'in flight');
  m.send('QR_DECODED', { code: 'Y' });
  expectEq(m.scan(), 'submitting', 'a different code mid-submit must be ignored, not prompt');
  m.send('QR_DECODED', { code: 'X' });
  expectEq(m.scan(), 'submitting', 'same-code re-scan mid-submit is a no-op');
  m.send('SRV_REDEEMED');
  expectEq(m.scan(), 'redeemed', 'the verdict still lands');
});

// -- 18 ----------------------------------------------------------------------
seq('Failures are retryable — error offers RETRY back to submit', (create) => {
  const effects = [];
  const m = create({ canOverride: false }, effects);
  play(m, [...TO_OFFER, ['ORDER_OK'], ['SUBMIT'], ['SRV_ERROR']]);
  expectEq(m.scan(), 'error', 'transport/server failure card');
  m.send('RETRY');
  expectEq(m.scan(), 'readyToSubmit', 'retry re-arms the submit with the session intact');
  m.send('SUBMIT');
  expectEq(m.scan(), 'submitting', 'second attempt reaches the server');
  m.send('SRV_REDEEMED');
  expectEq(m.scan(), 'redeemed', 'recovered');
});

// ---------------------------------------------------------------------------
export function runConformance(candidateName, createMachine) {
  let failed = 0;
  console.log(`# conformance run — candidate: ${candidateName} (${seqs.length} sequences)`);
  for (const s of seqs) {
    try {
      s.fn(createMachine);
      console.log(`  PASS  ${s.name}`);
    } catch (e) {
      failed++;
      console.log(`  FAIL  ${s.name}\n        ${e.message}`);
    }
  }
  console.log(failed === 0
    ? `# ${candidateName}: ALL ${seqs.length} SEQUENCES HELD`
    : `# ${candidateName}: ${failed}/${seqs.length} sequences FAILED`);
  return failed === 0;
}
