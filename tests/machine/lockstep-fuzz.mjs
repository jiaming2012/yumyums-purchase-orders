// tests/machine/lockstep-fuzz.mjs — THE CARD'S GATE, fuzz half (card
// redemption-submit-flow, run 20260905). Drives the PRODUCTION strict machine
// (marketing/submit-machine.js, throw mode, vendored lib/xstate.umd.min.js)
// and the spike's proven hand-rolled reference (./reference-machine.mjs)
// through the SAME seeded random walks and fails on any observable divergence
// (region states, gate flags, emitted effects) — the strict declarations must
// not have changed behavior anywhere in the fuzzed space.
//
// THE PER-STEP LIVENESS ASSERTION (the slate's addition — pass-by-death is
// impossible): after EVERY event the strict actor must still be alive and
// trip-free. The spike's Addendum 2 proved a dead actor frozen at its
// pre-trip state satisfies "state must not change" assertions vacuously; a
// corpse cannot satisfy THIS assertion, so an undeclared (state,event) pair
// anywhere in the fuzzed space reds the gate with its walk seed + trace.
//
// Deterministic: mulberry32, two seeds so one lucky seed cannot green a real
// divergence. Reproduce any failure from the printed seed/walk.
//
// Usage: node tests/machine/lockstep-fuzz.mjs      (exit 0 = gate green)
//   env: FUZZ_WALKS (default 20000, per seed) FUZZ_STEPS (20) FUZZ_SEEDS

import { createRequire } from 'node:module';
import { createHandrolled } from './reference-machine.mjs';

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

const WALKS = Number(process.env.FUZZ_WALKS || 20000);
const STEPS = Number(process.env.FUZZ_STEPS || 20);
const SEEDS = (process.env.FUZZ_SEEDS || '20260904,20260905').split(',').map(Number);

function mulberry32(a) {
  return function () {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// The spike's canonical 25-entry alphabet — connectivity, scan flow, both
// campaign classes, gate/override, session lifecycle, server verdicts.
const EVENTS = [
  ['SCAN', {}], ['QR_DECODED', { code: 'X' }], ['QR_DECODED', { code: 'Y' }],
  ['RESOLVED', { kind: 'offerReady' }], ['RESOLVED', { kind: 'offerReady', requiresOnline: true }],
  ['RESOLVED', { kind: 'unknownCode' }], ['RESOLVED', { kind: 'spentLocally' }], ['RESOLVED', { kind: 'expiredLocally' }],
  ['ORDER_OK', {}], ['SUBMIT', {}], ['OVERRIDE_REQUEST', {}], ['OVERRIDE_CONFIRM', {}], ['OVERRIDE_CANCEL', {}],
  ['DISMISS', {}], ['RETRY', {}], ['NEXT_CUSTOMER', {}],
  ['SRV_REDEEMED', {}], ['SRV_ALREADY_USED', {}], ['SRV_EXPIRED', {}], ['SRV_NOT_FOUND', {}], ['SRV_ERROR', {}],
  ['CONN_DOWN', {}], ['CONN_UP', {}], ['PROBE_TIMEOUT', {}], ['RESUBSCRIBED', {}],
];

const observe = (m, fx) => JSON.stringify({ conn: m.conn(), scan: m.scan(), flags: m.flags(), effects: fx });

let divergent = 0;
let deaths = 0;
let trippedTotal = 0;
const shapes = new Map();
let totalWalks = 0;

for (const SEED of SEEDS) {
  for (let w = 0; w < WALKS; w++) {
    totalWalks += 1;
    const rnd = mulberry32(SEED + w);
    const canOverride = rnd() < 0.5;
    const fxA = [];
    const fxB = [];
    const trips = [];
    const A = createHandrolled({ canOverride }, fxA);
    const B = createSubmitMachine(X, { canOverride }, fxB, { mode: 'throw', onTrip: (t) => trips.push(t) });
    const trace = [`(seed=${SEED} walk=${w} canOverride=${canOverride})`];

    for (let s = 0; s < STEPS; s++) {
      const [type, payload] = EVENTS[Math.floor(rnd() * EVENTS.length)];
      trace.push(type + (payload.code ? `:${payload.code}` : payload.kind ? `:${payload.kind}${payload.requiresOnline ? '!' : ''}` : ''));
      A.send(type, payload);
      B.send(type, payload);

      // ── PER-STEP LIVENESS ──
      if (!B.alive() || trips.length > 0) {
        deaths += 1;
        trippedTotal += trips.length;
        const t0 = trips[0];
        const key = `DEAD after ${type}${t0 ? ` (trip: ${t0.event} at scan:${t0.scan}/conn:${t0.conn}/overlay:${t0.overlay})` : ` (${B.lastError()})`}`;
        if (!shapes.has(key)) shapes.set(key, { trace: trace.join(' ') });
        break;
      }
      const a = observe(A, fxA);
      const b = observe(B, fxB);
      if (a !== b) {
        divergent += 1;
        const key = `${type} → hand:${A.scan()}/${A.conn()} vs strict:${B.scan()}/${B.conn()}`;
        if (!shapes.has(key)) shapes.set(key, { trace: trace.join(' '), a, b });
        break;
      }
    }
  }
}

console.log(`# lockstep fuzz: ${totalWalks} walks x ${STEPS} steps, seeds [${SEEDS.join(', ')}], alphabet ${EVENTS.length} entries`);
console.log(`# per-step liveness armed: deaths=${deaths} (tripwire hits ${trippedTotal}), divergences=${divergent}`);
if (deaths === 0 && divergent === 0) {
  console.log('NO DIVERGENCE, NO DEATHS — the strict machine is observably the proven reference, alive at every step');
  process.exit(0);
}
for (const [key, ex] of shapes) {
  console.log(`\n  FAIL ${key}`);
  console.log(`    ${ex.trace}`);
  if (ex.a) {
    console.log(`    hand-rolled: ${ex.a}`);
    console.log(`    strict:      ${ex.b}`);
  }
}
process.exit(1);
