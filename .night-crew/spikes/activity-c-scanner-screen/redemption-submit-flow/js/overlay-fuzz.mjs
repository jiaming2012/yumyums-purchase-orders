// overlay-fuzz.mjs — lockstep equivalence: hand-rolled (flat) vs the
// overlay-region variant. Same instrument as lockstep-fuzz.mjs.
import { createHandrolled } from './machine-handrolled.mjs';
import { createXstateOverlay } from './machine-xstate-overlay.mjs';

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
const EVENTS = [
  ['SCAN', {}], ['QR_DECODED', { code: 'X' }], ['QR_DECODED', { code: 'Y' }],
  ['RESOLVED', { kind: 'offerReady' }], ['RESOLVED', { kind: 'offerReady', requiresOnline: true }],
  ['RESOLVED', { kind: 'unknownCode' }], ['RESOLVED', { kind: 'spentLocally' }], ['RESOLVED', { kind: 'expiredLocally' }],
  ['ORDER_OK', {}], ['SUBMIT', {}], ['OVERRIDE_REQUEST', {}], ['OVERRIDE_CONFIRM', {}], ['OVERRIDE_CANCEL', {}],
  ['DISMISS', {}], ['RETRY', {}], ['NEXT_CUSTOMER', {}],
  ['SRV_REDEEMED', {}], ['SRV_ALREADY_USED', {}], ['SRV_EXPIRED', {}], ['SRV_NOT_FOUND', {}], ['SRV_ERROR', {}],
  ['CONN_DOWN', {}], ['CONN_UP', {}], ['PROBE_TIMEOUT', {}], ['RESUBSCRIBED', {}],
];
const observe = (m, fx) => JSON.stringify({ c: m.conn(), s: m.scan(), f: m.flags(), fx });

let divergent = 0;
const shapes = new Map();
for (let w = 0; w < WALKS; w++) {
  const rnd = mulberry32(SEED + w);
  const canOverride = rnd() < 0.5;
  const fxA = [], fxB = [];
  const A = createHandrolled({ canOverride }, fxA);
  const B = createXstateOverlay({ canOverride }, fxB);
  const trace = [`(canOverride=${canOverride})`];
  for (let s = 0; s < STEPS; s++) {
    const [type, payload] = EVENTS[Math.floor(rnd() * EVENTS.length)];
    trace.push(type + (payload.code ? `:${payload.code}` : payload.kind ? `:${payload.kind}${payload.requiresOnline ? '!' : ''}` : ''));
    A.send(type, payload); B.send(type, payload);
    const a = observe(A, fxA), b = observe(B, fxB);
    if (a !== b) {
      divergent++;
      const key = `${type} → flat:${A.scan()}/${A.conn()} vs overlay:${B.scan()}/${B.conn()}`;
      if (!shapes.has(key)) shapes.set(key, { walk: w, trace: trace.join(' '), a, b });
      break;
    }
  }
}
console.log(`# overlay lockstep fuzz: ${WALKS} walks x ${STEPS} steps, seed ${SEED}`);
if (divergent === 0) { console.log('NO DIVERGENCE — the overlay variant is observably the same machine'); process.exit(0); }
console.log(`${divergent} divergent walk(s), ${shapes.size} distinct shape(s):`);
for (const [key, ex] of shapes) {
  console.log(`\n  ✗ ${key}\n    walk ${ex.walk}: ${ex.trace}\n    flat:    ${ex.a}\n    overlay: ${ex.b}`);
}
process.exit(1);
