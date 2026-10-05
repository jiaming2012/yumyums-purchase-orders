#!/usr/bin/env bash
# 02-blanking-step-removed-leaves-tests-green.sh — spike for card J2 (B-480, ledger T-64):
# the test gap. With the three-line `UPDATE qr_scans … SET subscriber_id = NULL` removed from
# backend/internal/db/migrations/0086_merge_repoint_and_erasure_backstop.sql, the four erasure
# tests and the three migration round-trips in internal/marketing are claimed to stay green on a
# FRESH database (the migrations are embedded in the binary — a fresh database is what makes the
# mutated file the one that runs). Mutation in a THROWAWAY worktree off `dev`; Go database
# hq_test_spike_j2_go on :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  control PASS and mutated PASS — no test pins the blanking step (gap reproduced)
#   exit 1  a test went RED under the mutation — the step IS pinned; B-480's premise is wrong
#   exit 2  could not run
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"
spike_preflight
git -C "$REPO_ROOT" rev-parse --verify -q dev >/dev/null || cannot_run "no local branch 'dev'"
MAIN_SHA="$(git -C "$REPO_ROOT" rev-parse HEAD)"
LOG="$WORK"
TESTS='TestMigration0086ErasureBackstopDownAndUpRoundTrip|TestSubscriberDeleteCascadesTimelineAndBlanksScans|TestCodeDeleteBlanksFirstTouch|TestScannedCodeDeleteIsRefused|TestMigration0083DownAndUpRoundTrip|TestMigration0085SubscribersDownAndUpRoundTrip'
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
MIG="backend/internal/db/migrations/0086_merge_repoint_and_erasure_backstop.sql"

leg "worktree"
if [ -d "$WT" ]; then git -C "$REPO_ROOT" worktree remove --force "$WT" 2>/dev/null || rm -rf "$WT"; git -C "$REPO_ROOT" worktree prune; fi
git -C "$REPO_ROOT" worktree add --detach "$WT" dev >/dev/null
echo "#   $WT @ $(git -C "$WT" rev-parse --short HEAD) (dev)"

fresh_go_db() {
  padmin -c "drop database if exists $GO_DB with (force)" >/dev/null
  padmin -c "create database $GO_DB" >/dev/null
}
go_run() { # $1=label → echoes exit; log in $LOG/$1.log
  local rc=0
  ( cd "$WT/backend" && DB_TEST_URL="$GO_URL" go test ./internal/marketing -run "$TESTS" -count=1 -v ) >"$LOG/$1.log" 2>&1 || rc=$?
  echo "$rc"
}
summary() { grep -E "^(--- (PASS|FAIL|SKIP)|ok|FAIL|PASS)" "$LOG/$1.log" | sed 's/^/#   /'; }
passes()  { grep -c "^--- PASS" "$LOG/$1.log" || true; }
fails()   { grep -c "^--- FAIL" "$LOG/$1.log" || true; }
skips()   { grep -c "^--- SKIP" "$LOG/$1.log" || true; }

leg "control — the six tests on the UNMUTATED worktree, fresh $GO_DB"
fresh_go_db
RC0="$(go_run control)"; summary control
echo "#   control: exit=$RC0 pass=$(passes control) fail=$(fails control) skip=$(skips control)"
[ "$(skips control)" = 0 ] || cannot_run "a test SKIPPED on the control run (DB_TEST_URL not honoured?) — see $LOG/control.log"
[ "$RC0" = 0 ] && [ "$(passes control)" -ge 6 ] && [ "$(fails control)" = 0 ] || cannot_run "control is not green (exit=$RC0) — see $LOG/control.log"

leg "mutate — remove the three-line UPDATE from $MIG (worktree only)"
perl -0pi -e 's/UPDATE qr_scans s SET subscriber_id = NULL\n WHERE s\.subscriber_id IS NOT NULL\n   AND NOT EXISTS \(SELECT 1 FROM subscribers b WHERE b\.id = s\.subscriber_id\);\n//' "$WT/$MIG"
git -C "$WT" diff --stat -- "$MIG" | sed 's/^/#   /'
[ "$(git -C "$WT" diff --numstat -- "$MIG" | awk '{print $1"/"$2}')" = "0/3" ] || cannot_run "mutation did not apply as exactly 3 removed lines"
# (`grep -c` exits 1 on a zero count — under pipefail that read as a failure on the first run; hence the `|| true`)
echo "#   remaining \"SET subscriber_id = NULL\" occurrences in the Up: $(grep -c "SET subscriber_id = NULL" "$WT/$MIG" || true)"

leg "mutated — the same six tests, FRESH $GO_DB so the mutated 0086 is what migrates"
fresh_go_db
RC1="$(go_run mutated)"; summary mutated
echo "#   mutated: exit=$RC1 pass=$(passes mutated) fail=$(fails mutated) skip=$(skips mutated)"
if grep -q "^\[build failed\]\|cannot find\|undefined:" "$LOG/mutated.log"; then cannot_run "the mutated tree did not compile; see $LOG/mutated.log"; fi
[ "$(skips mutated)" = 0 ] || cannot_run "a test SKIPPED on the mutated run — see $LOG/mutated.log"

leg "restore"
git -C "$WT" checkout -- "$MIG"
[ -z "$(git -C "$WT" status --porcelain)" ] || cannot_run "worktree not clean after restore"
[ "$(git -C "$REPO_ROOT" rev-parse HEAD)" = "$MAIN_SHA" ] || fail "main tree HEAD moved during the spike"
[ -z "$(git -C "$REPO_ROOT" status --porcelain -- backend)" ] || fail "main tree backend/ is dirty — a mutation escaped the worktree"

echo "# wall-clock: $(( $(date +%s) - T0 ))s"
if [ "$RC1" != 0 ] || [ "$(fails mutated)" != 0 ]; then
  fail "$(fails mutated) test(s) RED with the blanking UPDATE removed — the step IS pinned; B-480's premise is wrong (see $LOG/mutated.log)"
fi
echo "✅ GREEN — B-480's gap reproduced on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): with 0086's blanking UPDATE removed,"
echo "   all $(passes mutated) tests (erasure ×4, round-trips ×3 incl. 0086's own) still PASS on a fresh database. No test pins the step."
