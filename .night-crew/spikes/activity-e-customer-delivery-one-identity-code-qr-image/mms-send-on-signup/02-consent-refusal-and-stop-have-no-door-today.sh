#!/usr/bin/env bash
# 02-consent-refusal-and-stop-have-no-door-today.sh — spike for card E2 (mms-send-on-signup,
# D-KR2): the red-first BASELINE for the two compliance tests the card writes. On a fresh
# database with the committed Fluent Forms fixture imported: a phone-bearing subscriber WITHOUT
# sms consent exists (refusal subject), one WITH exists (send subject), zero code_sent events,
# no inbound-SMS door (404), and the refusal subject is not opted out. A throwaway _test.go is
# copied beside subscribers_test.go in a THROWAWAY worktree; Go database hq_test_spike_e2_go on
# :5434 (TestMain migrates). NEVER :5433. No provider is called.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the baseline holds (both subjects present, 0 code_sent, 404 door, not opted out)
#   exit 1  a premise is wrong (a door already exists, a subject is missing, a send was recorded)
#   exit 2  could not run
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"
preflight
worktree_up
MKT="$WT/backend/internal/marketing"

leg "spec — copy the throwaway test beside subscribers_test.go"
cp "$SCRIPT_DIR/zz_spike_e2_test.go" "$MKT/zz_spike_e2_test.go"
echo "#   $MKT/zz_spike_e2_test.go"

leg "run — TestSpikeE2ConsentAndStopBaseline on a fresh $GO_DB"
fresh_go_db
RC="$(go_run e2 ./internal/marketing "TestSpikeE2ConsentAndStopBaseline")"; summary e2
grep -E "SPIKE-E2-2|\(a\)|\(b\)|\(c\)|\(d\)|\(e\)" "$LOG/e2.log" | sed 's/^/#   /' || true
echo "#   exit=$RC pass=$(passes e2) fail=$(fails e2) skip=$(skips e2)"
built e2 || cannot_run "the spike test did not compile — see $LOG/e2.log"
[ "$(skips e2)" = 0 ] || cannot_run "the spike test SKIPPED (DB_TEST_URL not honoured?) — see $LOG/e2.log"
rm -f "$MKT/zz_spike_e2_test.go"
[ -z "$(git -C "$WT" status --porcelain)" ] || cannot_run "worktree not clean after restore"
main_tree_untouched
[ "$RC" = 0 ] && [ "$(passes e2)" = 1 ] && [ "$(fails e2)" = 0 ] || fail "a baseline premise is wrong — see the (a)–(e) lines above and $LOG/e2.log"

echo
echo "✅ GREEN on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): the fixture carries a refusal subject (+17735550117, no sms consent)"
echo "   and a send subject (+17735559930); 0 code_sent; POST /sms/inbound is 404; nobody is opted out. D-KR2's reds are real."
