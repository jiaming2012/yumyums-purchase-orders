#!/usr/bin/env bash
# 01-wrong-shapes-reproduce-lock-is-unpinned-backstop-blanks.sh — spike for card K2
# (dish-merge-shapes-and-backstop-tests, B-484 + B-485 + B-487, triage T-66). Three legs in a
# THROWAWAY worktree off `dev`, Go database hq_test_spike_k2_go on :5434 (TestMain migrates):
#   (a)+(c)  a throwaway _test.go beside repository_test.go: the two wrong merge shapes
#            reproduce (missing source → 200, non-uuid target → 500) and deleting a dish a
#            campaign and a code name leaves both rows with item_id NULL (the backstop works)
#   (b)      MUTATION: " FOR SHARE" removed from the target read; the existing
#            TestMergeMenuItem_MissingTargetIsRefused still PASSES → the lock is unpinned
# NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a)+(c) pass, control pass, mutated (b) pass — all three premises hold
#   exit 1  a premise is wrong (a shape already fixed; the lock IS pinned; the backstop fails)
#   exit 2  could not run
set -euo pipefail
SPIKE_TAG=k2
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/../_golib.sh"
preflight
worktree_up
REPO="$WT/backend/internal/recipes/repository.go"
T_GUARD='TestMergeMenuItem_MissingTargetIsRefused'

leg "(a)+(c) — shapes and backstop, throwaway test on a fresh $GO_DB"
cp "$SCRIPT_DIR/zz_spike_k2_test.go" "$WT/backend/internal/recipes/zz_spike_k2_test.go"
fresh_go_db; migrate_go_db
RC_AC="$(go_run ac ./internal/recipes "TestSpikeK2ShapesAndBackstop")"; summary ac
grep -E "SPIKE-K2-[ac]:|\(a[12]\)|\(c\)" "$LOG/ac.log" | sed 's/^/#   /' || true
echo "#   exit=$RC_AC pass=$(passes ac) fail=$(fails ac) skip=$(skips ac)"
built ac || cannot_run "the spike test did not compile — see $LOG/ac.log"
[ "$(skips ac)" = 0 ] || cannot_run "the spike test SKIPPED (DB_TEST_URL not honoured?) — see $LOG/ac.log"
rm -f "$WT/backend/internal/recipes/zz_spike_k2_test.go"
[ "$RC_AC" = 0 ] && [ "$(passes ac)" = 1 ] && [ "$(fails ac)" = 0 ] || fail "a shape or the backstop premise is wrong — see the SPIKE-K2 lines above and $LOG/ac.log"

leg "(b) control — $T_GUARD on the unmutated worktree"
fresh_go_db; migrate_go_db
RC_B0="$(go_run b0 ./internal/recipes "$T_GUARD")"; summary b0
echo "#   control: exit=$RC_B0 pass=$(passes b0) fail=$(fails b0)"
[ "$RC_B0" = 0 ] && [ "$(fails b0)" = 0 ] || cannot_run "control is not green — see $LOG/b0.log"

leg "(b) mutation — remove ' FOR SHARE' from the target read (worktree only)"
grep -q 'WHERE id = \$1 FOR SHARE' "$REPO" || cannot_run "the target read 'WHERE id = \$1 FOR SHARE' is not in $REPO — the guard moved; re-read the card"
# Only the SQL line — the phrase also appears in a comment above it.
sed -i 's/WHERE id = \$1 FOR SHARE/WHERE id = $1/' "$REPO"
git -C "$WT" diff --stat -- backend/internal/recipes/repository.go | sed 's/^/#   /'
[ "$(git -C "$WT" diff --numstat -- backend/internal/recipes/repository.go | awk '{print $1"/"$2}')" = "1/1" ] || cannot_run "mutation did not apply as exactly one changed line"
fresh_go_db; migrate_go_db
RC_B1="$(go_run b1 ./internal/recipes "$T_GUARD")"; summary b1
echo "#   mutated: exit=$RC_B1 pass=$(passes b1) fail=$(fails b1)"
built b1 || cannot_run "the mutated tree did not compile — see $LOG/b1.log"

leg "restore"
git -C "$WT" checkout -- backend/internal/recipes/repository.go
[ -z "$(git -C "$WT" status --porcelain)" ] || cannot_run "worktree not clean after restore"
main_tree_untouched
if [ "$RC_B1" != 0 ] || [ "$(fails b1)" != 0 ]; then
  fail "$(fails b1) test(s) RED with FOR SHARE removed — the lock IS pinned; B-484's premise is wrong (see $LOG/b1.log)"
fi

echo
echo "✅ GREEN on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): missing source → 200, non-uuid target → 500 (B-485);"
echo "   with FOR SHARE removed the missing-target test still passes (B-484, unpinned); a dish deletion blanks"
echo "   both referencing rows (B-487's backstop works — the card pins it by behaviour)."
