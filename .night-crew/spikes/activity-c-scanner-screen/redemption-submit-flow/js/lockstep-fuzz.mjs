// lockstep-fuzz.mjs — drives BOTH machine candidates through the SAME seeded
// random event walks and reports every observable divergence (region states,
// gate flags, emitted effects). The conformance suite proves the decided
// behaviors; this proves no divergence lurks in the state space OUTSIDE the
// suite — the two candidates must be observably the same machine, or the
// "which engine" fork silently becomes a "which behavior" fork.
//
// Deterministic: seeded PRNG (mulberry32), so any reported divergence is
// reproducible from its walk seed. exit 0 = no divergence across all walks;
// exit 1 = divergences found (each distinct shape printed once, with trace).

import { createHandrolled } from './machine-handrolled.mjs';
import { createXstate } from './machine-xstate.mjs';

const WALKS = Number(process.env.FUZZ_WALKS || 5000);
const STEPS = Number(process.env.FUZZ_STEPS || 14);
const SEED = Number(process.env.FUZZ_SEED || 20260904);

function mulberry32(a) {
  return function () {
    a |= 0; a = (a + 0x6D2B79F5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

// The full event alphabet with canonical payloads — connectivity, scan flow,
// resolution kinds (both campaign classes), gate/override, session lifecycle,
// server verdicts.
const EVENTS = [
  ['SCAN', {}],
  ['QR_DECODED', { code: 'X' }],
  ['QR_DECODED', { code: 'Y' }],
  ['RESOLVED', { kind: 'offerReady' }],
  ['RESOLVED', { kind: 'offerReady', requiresOnline: true }],
  ['RESOLVED', { kind: 'unknownCode' }],
  ['RESOLVED', { kind: 'spentLocally' }],
  ['RESOLVED', { kind: 'expiredLocally' }],
  ['ORDER_OK', {}],
  ['SUBMIT', {}],
  ['OVERRIDE_REQUEST', {}],
  ['OVERRIDE_CONFIRM', {}],
  ['OVERRIDE_CANCEL', {}],
  ['DISMISS', {}],
  ['RETRY', {}],
  ['NEXT_CUSTOMER', {}],
  ['SRV_REDEEMED', {}],
  ['SRV_ALREADY_USED', {}],
  ['SRV_EXPIRED', {}],
  ['SRV_NOT_FOUND', {}],
  ['SRV_ERROR', {}],
  ['CONN_DOWN', {}],
  ['CONN_UP', {}],
  ['PROBE_TIMEOUT', {}],
  ['RESUBSCRIBED', {}],
];

const observe = (m, fx) => JSON.stringify({
  conn: m.conn(), scan: m.scan(), flags: m.flags(), effects: fx,
});

let divergent = 0;
const shapes = new Map(); // distinct divergence shapes → first example

for (let w = 0; w < WALKS; w++) {
  const rnd = mulberry32(SEED + w);
  const canOverride = rnd() < 0.5;
  const fxA = [], fxB = [];
  const A = createHandrolled({ canOverride }, fxA);
  const B = createXstate({ canOverride }, fxB);
  const trace = [`(canOverride=${canOverride})`];

  for (let s = 0; s < STEPS; s++) {
    const [type, payload] = EVENTS[Math.floor(rnd() * EVENTS.length)];
    trace.push(type + (payload.code ? `:${payload.code}` : payload.kind ? `:${payload.kind}${payload.requiresOnline ? '!' : ''}` : ''));
    A.send(type, payload);
    B.send(type, payload);
    const a = observe(A, fxA), b = observe(B, fxB);
    if (a !== b) {
      divergent++;
      const key = `${type} → hand:${A.scan()}/${A.conn()} vs xstate:${B.scan()}/${B.conn()}`;
      if (!shapes.has(key)) shapes.set(key, { walk: w, trace: trace.join(' '), a, b });
      break; // first divergence per walk; later state is garbage
    }
  }
}

console.log(`# lockstep fuzz: ${WALKS} walks x ${STEPS} steps, seed ${SEED}, alphabet ${EVENTS.length} events`);
if (divergent === 0) {
  console.log('NO DIVERGENCE — the two machines are observably equivalent over the fuzzed space');
  process.exit(0);
}
console.log(`${divergent} divergent walk(s), ${shapes.size} distinct shape(s):`);
for (const [key, ex] of shapes) {
  console.log(`\n  ✗ ${key}`);
  console.log(`    walk ${ex.walk}: ${ex.trace}`);
  console.log(`    hand-rolled: ${ex.a}`);
  console.log(`    xstate:      ${ex.b}`);
}
process.exit(1);
