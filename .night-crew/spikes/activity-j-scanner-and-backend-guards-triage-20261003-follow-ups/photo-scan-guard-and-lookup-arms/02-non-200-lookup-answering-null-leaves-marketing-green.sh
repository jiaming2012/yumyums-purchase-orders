#!/usr/bin/env bash
# 02-non-200-lookup-answering-null-leaves-marketing-green.sh — spike for card J1
# (photo-scan-guard-and-lookup-arms, B-476, ledger T-64): one of the four scan-time-check arms
# the review found ungated. In marketing/scanner.js createServerLookup, a non-200 answer
# REJECTS ("could not ask" → unknownCode with verified:false, "couldn't check the server").
# Mutated to `return null` ("the server does not know it" → plain unknownCode, no warning),
# the WHOLE of tests/marketing.spec.js is claimed to stay green. Mutation in a THROWAWAY
# worktree; TEST_DB_NAME=hq_test_spike_j1_20261005, TEST_PORT=8321, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the mutated tree runs the full marketing spec: exit 0, 0 failed, ≥ 60 passed (gap reproduced)
#   exit 1  a spec went red under the mutation — the arm IS gated; B-476's premise is wrong for it
#   exit 2  could not run
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
preflight
worktree_up

leg "mutate — createServerLookup: non-200 → return null (was: throw) in $WT/marketing/scanner.js"
perl -0pi -e 's/(      if \(res\.status !== 200\) \{\n)        throw new Error\(`\[marketing-scan\] server lookup answered HTTP \$\{res\.status\}`\);\n(      \})/$1        return null; \/\/ SPIKE J1 MUTATION (B-476 arm 3)\n$2/' "$WT/marketing/scanner.js"
git -C "$WT" diff --stat -- marketing/scanner.js | sed 's/^/#   /'
git -C "$WT" diff -U0 -- marketing/scanner.js | grep -E '^[-+]\s' | sed 's/^/#   /'
[ "$(git -C "$WT" diff --numstat -- marketing/scanner.js | awk '{print $1"/"$2}')" = "1/1" ] || cannot_run "mutation did not apply as exactly one changed line"

leg "run — the WHOLE of tests/marketing.spec.js against the mutated lookup (--retries=0)"
RC="$(pw_run m tests/marketing.spec.js)"
read -r EXPECTED UNEXPECTED SKIPPED FLAKY <<<"$(pw_stats m 2>/dev/null || echo "0 0 0 0")"
grep -E "passed|failed|flaky|skipped|Timed out" "$LOG/m.log" | tail -4 | sed 's/^/#   /'
exitline "playwright tests/marketing.spec.js (mutated)" "$RC" "0"
echo "#   stats: expected=$EXPECTED unexpected=$UNEXPECTED skipped=$SKIPPED flaky=$FLAKY"
[ -s "$LOG/m.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/m.log"
[ "$EXPECTED" -ge 60 ] || cannot_run "only $EXPECTED specs passed — the run did not cover the file (webServer wedge? B-477 rxdb?); see $LOG/m.log"
if [ "$UNEXPECTED" != "0" ] || [ "$RC" != "0" ]; then
  pw_outcomes m | grep -F '=false' | sed 's/^/#   RED: /'
  fail "$UNEXPECTED spec(s) red under the non-200→null mutation — this arm IS gated; B-476's premise is wrong for it"
fi

leg "restore — git checkout -- marketing/scanner.js (worktree only)"
git -C "$WT" checkout -- marketing/scanner.js
[ -z "$(git -C "$WT" status --porcelain -- marketing)" ] || cannot_run "worktree marketing/ not clean after restore"
main_tree_untouched
echo
echo "✅ GREEN — B-476 arm 3 reproduced on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): with a non-200 lookup answering"
echo "   null instead of rejecting, tests/marketing.spec.js is $EXPECTED passed / 0 failed. No spec reds; the arm has no gate."
