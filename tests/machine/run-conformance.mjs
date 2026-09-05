// tests/machine/run-conformance.mjs — wires the PRODUCTION strict machine
// (marketing/submit-machine.js) on the VENDORED XState UMD build
// (lib/xstate.umd.min.js — the exact shipped artifact, loaded through the UMD
// wrapper's CommonJS branch) and runs the 18-sequence §19.4 conformance suite
// in THROW mode: an undeclared (state,event) pair kills the actor, and any
// tripwire hit fails the run even if every sequence "passed" (no pass-by-death
// at the suite level either).
//
// Usage: node tests/machine/run-conformance.mjs   (exit 0 = gate green)

import { createRequire } from 'node:module';
import { runConformance } from './conformance.mjs';

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

const trips = [];
const ok = runConformance(
  'strict-overlay-xstate (production marketing/submit-machine.js on vendored xstate.umd.min.js)',
  (input, effects) => createSubmitMachine(X, input, effects, { mode: 'throw', onTrip: (t) => trips.push(t) }),
);

if (trips.length > 0) {
  console.log(`# TRIPWIRE: ${trips.length} undeclared-pair hit(s) during the suite:`);
  for (const t of trips.slice(0, 20)) console.log(`    ${t.event} at scan:${t.scan}/conn:${t.conn}/overlay:${t.overlay}`);
}

const probe = createSubmitMachine(X, {}, [], { mode: 'throw' });
const pairs = probe.declaredPairs();
console.log(`# declared pairs: ${pairs.alphabetPairs} across ${pairs.states} states (alphabet ${pairs.alphabet})`);

process.exit(ok && trips.length === 0 ? 0 : 1);
