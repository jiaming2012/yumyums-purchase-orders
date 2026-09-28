// strict-experiment.mjs — operator-directed observation: rewrite the XState
// candidate with the v5 migration guide's strict-mode replacement (a '*'
// wildcard whose action throws) and observe what additionally fails.
//
// Phase 0 probes the selection semantics the blast radius depends on (does a
// wildcard fire when an explicit key's guards all fail? does a child's
// unhandled/guard-failed event reach the parent's explicit key before the
// parent's wildcard?). Then three tiers, each run against the full
// 18-sequence conformance suite:
//   tier A — naive tripwire (region-root wildcards, nothing whitelisted)
//   tier B — sibling-region alphabets whitelisted as explicit {} no-ops
//   tier C — the suite-asserted contract ignores whitelisted too
// and finally a lockstep fuzz of tier C vs the hand-rolled machine, with
// every tripwire hit recorded as (scan-state × event) for enumeration.

import { createMachine, createActor } from 'xstate';
import { runConformance } from './conformance.mjs';
import { createXstate } from './machine-xstate.mjs';
import { createHandrolled } from './machine-handrolled.mjs';

// ---------------------------------------------------------------------------
// Phase 0 — selection-semantics probes
// ---------------------------------------------------------------------------
console.log('════ phase 0 · selection-semantics probes ════');
{
  // (a) same state: explicit key whose guards all fail + a wildcard.
  const log = [];
  const m = createMachine({
    initial: 'a',
    states: {
      a: {
        on: {
          E: { guard: () => false, target: 'b' },
          '*': { actions: () => log.push('wildcard fired') },
        },
      },
      b: {},
    },
  });
  const actor = createActor(m); actor.start(); actor.send({ type: 'E' }); actor.stop();
  console.log(`  (a) guard-failed explicit key + same-state wildcard → ${log.length ? log[0] : 'wildcard did NOT fire'}`);
}
{
  // (b) child unhandled → parent explicit {} vs parent wildcard.
  const log = [];
  const m = createMachine({
    initial: 'p',
    states: {
      p: {
        initial: 'c',
        on: { E: { actions: () => log.push('parent explicit') }, '*': { actions: () => log.push('parent wildcard') } },
        states: { c: {} },
      },
    },
  });
  const actor = createActor(m); actor.start(); actor.send({ type: 'E' }); actor.stop();
  console.log(`  (b) child-unhandled event at parent → ${log.join(', ') || 'nothing fired'}`);
}
{
  // (c) child explicit key with failing guard → parent explicit {}?
  const log = [];
  const m = createMachine({
    initial: 'p',
    states: {
      p: {
        initial: 'c',
        on: { E: { actions: () => log.push('parent explicit') }, '*': { actions: () => log.push('parent wildcard') } },
        states: { c: { on: { E: { guard: () => false, target: 'c' } } } },
      },
    },
  });
  const actor = createActor(m); actor.start(); actor.send({ type: 'E' }); actor.stop();
  console.log(`  (c) child guard-failed event at parent → ${log.join(', ') || 'nothing fired'}`);
}

// ---------------------------------------------------------------------------
// The three tiers against the full suite
// ---------------------------------------------------------------------------
function tier(name, opts) {
  console.log(`\n════ ${name} ════`);
  const trips = [];
  const ok = runConformance(`xstate-strict(${name})`, (input, effects) =>
    createXstate(input, effects, { ...opts, trips }));
  const distinct = [...new Set(trips)];
  console.log(`  tripwire hits: ${trips.length} total, ${distinct.length} distinct`);
  for (const t of distinct.slice(0, 12)) console.log(`    ✗ ${t}`);
  if (distinct.length > 12) console.log(`    … and ${distinct.length - 12} more`);
  return ok;
}

tier('tier A · naive tripwire', { strict: true });
tier('tier B · + foreign alphabet whitelisted', { strict: true, ignoreForeign: true });
const tierCOk = tier('tier C · + contract ignores whitelisted', { strict: true, ignoreForeign: true, ignoreContracts: true });

// ---------------------------------------------------------------------------
// Tier C under fuzz — enumerate every silent ignore the tripwire can find
// ---------------------------------------------------------------------------
console.log('\n════ tier C · lockstep fuzz vs hand-rolled (3000 × 14, seed 20260904) ════');
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

const tripSites = new Map(); // "scanState × event" → count
let divergentWalks = 0, trippedWalks = 0;
for (let w = 0; w < 3000; w++) {
  const rnd = mulberry32(20260904 + w);
  const canOverride = rnd() < 0.5;
  const fxA = [], fxB = [], trips = [];
  const A = createHandrolled({ canOverride }, fxA);
  const B = createXstate({ canOverride }, fxB, { strict: true, ignoreForeign: true, ignoreContracts: true, trips });
  for (let s = 0; s < 14; s++) {
    const [type, payload] = EVENTS[Math.floor(rnd() * EVENTS.length)];
    const preScan = B.scan(), preConn = B.conn(), preTrips = trips.length;
    A.send(type, payload);
    B.send(type, payload);
    if (trips.length > preTrips) {
      trippedWalks++;
      const site = `${preScan} (conn ${preConn}) × ${type}`;
      tripSites.set(site, (tripSites.get(site) || 0) + 1);
      break; // strict actor is dead — the walk cannot continue
    }
    const a = JSON.stringify({ c: A.conn(), s: A.scan(), f: A.flags(), fx: fxA });
    const b = JSON.stringify({ c: B.conn(), s: B.scan(), f: B.flags(), fx: fxB });
    if (a !== b) { divergentWalks++; break; }
  }
}
console.log(`  walks: 3000 · killed by tripwire: ${trippedWalks} · diverged without a trip: ${divergentWalks}`);
console.log(`  distinct trip sites (state × event) — the machine's complete residual silent-ignore map:`);
const sorted = [...tripSites.entries()].sort((x, y) => y[1] - x[1]);
for (const [site, n] of sorted) console.log(`    ${String(n).padStart(5)} × ${site}`);

console.log(`\n════ summary ════`);
console.log(`  tier C suite: ${tierCOk ? 'ALL SEQUENCES HELD' : 'FAILURES (see above)'}`);
console.log(`  residual ignore sites found by the tripwire: ${sorted.length}`);
process.exit(0);
