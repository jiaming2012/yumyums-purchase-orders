#!/usr/bin/env bash
# 03-machines-equivalent.sh — spike: the two candidates are OBSERVABLY THE SAME
# MACHINE outside the suite. The 18-sequence suite proves the decided
# behaviors; this drives both machines through seeded random event walks in
# lockstep (js/lockstep-fuzz.mjs) and fails on any divergence in region
# states, gate flags, or emitted effects. Without this, the "which engine"
# fork silently becomes a "which behavior" fork — which is exactly what the
# missing-states deep dive caught: the pre-fix pair diverged in 114/5000 walks
# (PROBE_TIMEOUT while stale) plus session-lifecycle gaps the suite's first
# ten sequences never reached.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  no divergence across all walks on both seeds.
#   exit 1  a divergence — the candidates disagree somewhere the suite
#           doesn't pin; the disagreement IS the finding (reproducible from
#           the printed walk seed).
#   exit 2  no node.
#
# Deterministic (mulberry32, fixed seeds). Two seeds so a single lucky seed
# cannot green a real divergence. No dependencies for the hand-rolled side;
# xstate must already be installed (01-machine-xstate.sh does that).

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
command -v node >/dev/null 2>&1 || cannot_run "node not on PATH"
[ -d "$SCRIPT_DIR/js/node_modules/xstate" ] || cannot_run "xstate not installed — run 01-machine-xstate.sh first"

echo "── lockstep fuzz, seed 1 ──"
FUZZ_WALKS=20000 FUZZ_STEPS=20 FUZZ_SEED=20260904 node "$SCRIPT_DIR/js/lockstep-fuzz.mjs" \
  || fail "the candidates diverge (seed 20260904) — see the shapes above"

echo
echo "── lockstep fuzz, seed 2 ──"
FUZZ_WALKS=20000 FUZZ_STEPS=20 FUZZ_SEED=41 node "$SCRIPT_DIR/js/lockstep-fuzz.mjs" \
  || fail "the candidates diverge (seed 41) — see the shapes above"

printf '\n✅ VERDICT: GREEN — 40,000 walks x 20 steps, 2 seeds, 25-event alphabet: the two machines are observably equivalent. The fork is purely an engine choice.\n'
