#!/usr/bin/env bash
# 01-photo-pick-during-held-lookup-leaves-no-submit.sh — spike for card J1
# (photo-scan-guard-and-lookup-arms, B-475, ledger T-64 decision 205): the card's [PS-01]
# red-first baseline. On today's `dev`, through the SHIPPED page (provisioning, resolver,
# lookup, submit machine all real; only the sync door is mocked at the network layer):
#   with never-seen code A's server lookup HELD, picking a PHOTO of locally-held code B
#   raises the F6 "Finish the current customer first" prompt, and when A's lookup is
#   released as a live row A's offer card renders with NO #ms-order — the stuck state.
#   Control: the same scan with no photo picked renders A's offer WITH #ms-order.
# The spec lives beside this script (spike-ps.spec.js) and is copied into a THROWAWAY
# worktree's tests/ directory for the run; the main tree is never touched.
# Coordinates: TEST_DB_NAME=hq_test_spike_j1_20261005, TEST_PORT=8321, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  premise reproduced (spec 1 ok) AND control healthy (spec 2 ok)
#   exit 1  the stuck state did NOT reproduce, or the control is itself broken — "🛑 VERDICT: RED"
#   exit 2  could not run (precondition missing; nothing was measured)
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
preflight
worktree_up

leg "spec — copy spike-ps.spec.js into the worktree's tests/"
cp "$SCRIPT_DIR/spike-ps.spec.js" "$WT/tests/spike-ps.spec.js"
[ -f "$WT/tests/fixtures/qr-fixture-1.png" ] || cannot_run "tests/fixtures/qr-fixture-1.png missing in the worktree"
echo "#   $WT/tests/spike-ps.spec.js (2 specs: premise + control); photo = tests/fixtures/qr-fixture-1.png"

leg "run — premise + control through the shipped page"
RC="$(pw_run ps tests/spike-ps.spec.js)"
OUT="$(pw_outcomes ps 2>/dev/null || true)"
grep -E "SPIKE-PS-1 after release|passed|failed|flaky|Error|Timed out" "$LOG/ps.log" | tail -6 | sed 's/^/#   /'
exitline "playwright tests/spike-ps.spec.js" "$RC" "0"
[ -s "$LOG/ps.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/ps.log"
[ "$(wc -l <<<"$OUT")" = "2" ] || cannot_run "expected exactly 2 specs, got: $(tr '\n' ' ' <<<"$OUT")"
has "$OUT" "[PS-SPIKE-2]=true" || fail "CONTROL is red — a held-then-released scan with no photo does not render #ms-order; the premise cannot be read against a broken control (see $LOG/ps.log)"
has "$OUT" "[PS-SPIKE-1]=true" || fail "the stuck state did NOT reproduce — after the photo pick A rendered WITH a submit control, or the F6 prompt never appeared (see $LOG/ps.log)"
[ "$RC" = "0" ] || fail "playwright exited $RC with both specs ok=true"

main_tree_untouched
echo
echo "✅ GREEN — B-475 reproduced by execution on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): a photo picked while"
echo "   the lookup waits raises the F6 prompt and leaves A's offer with NO #ms-order; the control renders #ms-order."
echo "   Card J1's [PS-01] red-first baseline stands; the fix is the camera path's busy guard on onFilePicked."
