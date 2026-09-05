// tests/machine/strictness.mjs — the no-silent-no-ops contract, asserted (card
// redemption-submit-flow, run 20260905; the operator's strictness call,
// verbatim in the slate): every (state,event) pair is a DECLARED decision; an
// UNDECLARED pair THROWS in the test build (mode 'throw' — this is why the
// fuzz carries a per-step liveness assertion) and in the production build
// (mode 'model', the page default) raises the modeled, visible, retryable
// unexpectedEvent state. The three Addendum-2 reachable-benign pairs are
// asserted DECLARED by name — each would brick the screen under naive
// throwing strictness.
//
// Usage: node tests/machine/strictness.mjs   (exit 0 = gate green)

import { createRequire } from 'node:module';
const require = createRequire(import.meta.url);

let X;
let createSubmitMachine;
try {
  X = require('../../lib/xstate.umd.min.js');
  ({ createSubmitMachine } = await import('../../marketing/submit-machine.js'));
} catch (e) {
  console.log(`# RED — the production machine is absent from this tree: ${e.message}`);
  process.exit(1);
}


let failed = 0;
const check = (name, fn) => {
  try { fn(); console.log(`  PASS  ${name}`); }
  catch (e) { failed++; console.log(`  FAIL  ${name}\n        ${e.message}`); }
};
const eq = (a, w, what) => { if (a !== w) throw new Error(`${what}: got ${JSON.stringify(a)}, want ${JSON.stringify(w)}`); };
const drive = (m, steps) => { for (const [t, p] of steps) m.send(t, p || {}); };
const TO_OFFER = [['SCAN'], ['QR_DECODED', { code: 'X' }], ['RESOLVED', { kind: 'offerReady' }]];

console.log('# strictness assertions — the no-silent-no-ops contract');

check('TEST BUILD: an undeclared (state,event) pair THROWS and the trip names the pair', () => {
  const trips = [];
  const m = createSubmitMachine(X, {}, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  m.send('BOGUS_EVENT');
  eq(trips.length >= 1, true, 'tripwire fired');
  eq(trips[0].event, 'BOGUS_EVENT', 'trip names the event');
  eq(m.alive(), false, 'the throwing tripwire kills the actor (why the liveness assertion exists)');
});

check('PROD BUILD: the same undeclared pair raises the modeled unexpectedEvent state — alive, visible', () => {
  const trips = [];
  const m = createSubmitMachine(X, {}, [], { mode: 'model', onTrip: (t) => trips.push(t) });
  drive(m, TO_OFFER);
  m.send('BOGUS_EVENT');
  eq(m.alive(), true, 'never a dead actor at the window');
  eq(m.scan(), 'unexpectedEvent', 'modeled, visible error state');
  eq(trips.length >= 1, true, 'logged (onTrip fired)');
  eq(m.ctx().unexpectedInfo.event, 'BOGUS_EVENT', 'the card can name the event');
  eq(m.ctx().unexpectedInfo.scan, 'offerReady', 'the card can name the interrupted state');
});

check('PROD BUILD: unexpectedEvent is RETRYABLE back to the interrupted state (UI-R6)', () => {
  const m = createSubmitMachine(X, {}, [], { mode: 'model' });
  drive(m, [...TO_OFFER, ['ORDER_OK']]);
  eq(m.scan(), 'readyToSubmit', 'pre-trip state');
  m.send('BOGUS_EVENT');
  eq(m.scan(), 'unexpectedEvent', 'tripped');
  m.send('RETRY');
  eq(m.scan(), 'readyToSubmit', 'RETRY returns to the interrupted state, session intact');
  m.send('SUBMIT');
  eq(m.scan(), 'submitting', 'the resumed session still submits');
});

check('PROD BUILD: unexpectedEvent resets cleanly for the next customer', () => {
  const m = createSubmitMachine(X, {}, [], { mode: 'model' });
  drive(m, TO_OFFER);
  m.send('BOGUS_EVENT');
  m.send('NEXT_CUSTOMER');
  eq(m.scan(), 'idle', 'reset');
  eq(m.ctx().unexpectedInfo, null, 'trip info cleared');
  drive(m, [['SCAN'], ['QR_DECODED', { code: 'Y' }]]);
  eq(m.scan(), 'resolving', 'fresh session (no stale prompt)');
});

check('PROD BUILD: a trip closes the finish-first prompt so the error card is visible', () => {
  const m = createSubmitMachine(X, {}, [], { mode: 'model' });
  drive(m, [...TO_OFFER, ['QR_DECODED', { code: 'Y' }]]);
  eq(m.scan(), 'promptFinishCurrent', 'prompt open');
  m.send('BOGUS_EVENT');
  eq(m.scan(), 'unexpectedEvent', 'error card supersedes the modal');
  m.send('RETRY');
  eq(m.scan(), 'offerReady', 'return lands on the state the prompt had interrupted');
});

check('DECLARED reachable-benign #1: PROBE_TIMEOUT while already offline is a no-op, not a brick', () => {
  const trips = [];
  const m = createSubmitMachine(X, {}, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  m.send('CONN_DOWN');
  m.send('PROBE_TIMEOUT'); m.send('PROBE_TIMEOUT'); m.send('PROBE_TIMEOUT');
  eq(m.alive(), true, 'the probe loop timing out while offline must not kill the actor');
  eq(trips.length, 0, 'declared, not tripped');
  eq(m.conn(), 'offline', 'still offline');
});

check('DECLARED reachable-benign #2: a spurious RESUBSCRIBED while online is a no-op', () => {
  const trips = [];
  const m = createSubmitMachine(X, {}, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  m.send('RESUBSCRIBED');
  eq(m.alive(), true, 'alive');
  eq(trips.length, 0, 'declared, not tripped');
  eq(m.conn(), 'online', 'still online');
});

check('DECLARED reachable-benign #3: a late RESOLVED after NEXT_CUSTOMER is a no-op', () => {
  const trips = [];
  const m = createSubmitMachine(X, {}, [], { mode: 'throw', onTrip: (t) => trips.push(t) });
  drive(m, [['SCAN'], ['QR_DECODED', { code: 'X' }], ['NEXT_CUSTOMER']]);
  m.send('RESOLVED', { kind: 'offerReady' });
  eq(m.alive(), true, 'alive');
  eq(trips.length, 0, 'declared, not tripped');
  eq(m.scan(), 'idle', 'still idle');
});

check('The declared-pair accounting is real: every alphabet event is a declared decision in every state', () => {
  const m = createSubmitMachine(X, {}, [], { mode: 'throw' });
  const p = m.declaredPairs();
  console.log(`        declared alphabet pairs: ${p.alphabetPairs} across ${p.states} states (alphabet ${p.alphabet})`);
  eq(p.alphabetPairs, p.states * p.alphabet, 'full coverage: states x alphabet — nothing left implicit');
});

console.log(failed === 0 ? '# strictness: ALL ASSERTIONS HELD' : `# strictness: ${failed} FAILED`);
process.exit(failed === 0 ? 0 : 1);
