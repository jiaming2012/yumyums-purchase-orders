#!/usr/bin/env bash
# 01-refused-pick-is-silent-and-post-boot-policy-patch-is-inert.sh — spike for card K4
# (scanner-refusal-seam-and-pick-feedback, B-482 + B-483, triage T-66). Through the SHIPPED page
# (sync door mocked at the network layer):
#   premise 1 (B-483)  a photo picked while a lookup is held is refused with NO feedback
#   control            an offline scan of a held low-value code, unpatched — the kind to compare
#   premise 2 (B-482)  policyFor patched to throw AFTER boot → the same kind as the control; the
#                      throw never reaches the page (capture-once), so only a boot-time seam can
# Coordinates: TEST_DB_NAME=hq_test_spike_k4_20261007, TEST_PORT=8343, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  both premises reproduce and the control ran
#   exit 1  a premise did not reproduce — "🛑 VERDICT: RED" (if premise 2 is red, the post-boot
#           patch DID reach the refusal, and the card is just the test, no seam)
#   exit 2  could not run
set -euo pipefail
SPIKE_TAG=k4; PW_PORT=8343
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/../_pwlib.sh"
preflight
worktree_up

leg "spec — copy spike-k4.spec.js into the worktree's tests/"
cp "$SCRIPT_DIR/spike-k4.spec.js" "$WT/tests/spike-k4.spec.js"
[ -f "$WT/tests/fixtures/qr-fixture-1.png" ] || cannot_run "tests/fixtures/qr-fixture-1.png missing in the worktree"
echo "#   $WT/tests/spike-k4.spec.js (3 specs: premise B-483, control, premise B-482)"

leg "run — through the shipped page"
RC="$(pw_run k4 tests/spike-k4.spec.js)"
OUT="$(pw_outcomes k4 2>/dev/null || true)"
grep -o 'SPIKE-K4-[0-9]: {.*}' "$LOG/k4.log" | sed 's/^/#   /'
grep -E "passed|failed|flaky|Timed out" "$LOG/k4.log" | tail -3 | sed 's/^/#   /'
exitline "playwright tests/spike-k4.spec.js" "$RC" "0"
[ -s "$LOG/k4.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/k4.log"
[ "$(wc -l <<<"$OUT")" = "3" ] || cannot_run "expected exactly 3 specs, got: $(tr '\n' ' ' <<<"$OUT")"
has "$OUT" "[K4-SPIKE-3]=true" || cannot_run "the CONTROL did not run to a readable kind — see $LOG/k4.log"
has "$OUT" "[K4-SPIKE-1]=true" || fail "B-483 did NOT reproduce — the refused pick showed feedback, or the F6 prompt appeared (see $LOG/k4.log)"
has "$OUT" "[K4-SPIKE-2]=true" || fail "B-482's premise is WRONG — patching policyFor after boot reached the refusal; the card needs no seam, only the test (see $LOG/k4.log)"
[ "$RC" = "0" ] || fail "playwright exited $RC with all three specs ok=true"

main_tree_untouched
echo
echo "✅ GREEN on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): a refused photo pick is silent (B-483), and a post-boot"
echo "   policyFor patch never reaches the page (B-482) — the card adds one feedback line and a boot-time test seam."
