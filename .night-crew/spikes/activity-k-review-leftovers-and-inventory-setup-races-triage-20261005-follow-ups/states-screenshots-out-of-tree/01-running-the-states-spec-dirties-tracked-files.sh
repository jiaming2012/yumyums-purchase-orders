#!/usr/bin/env bash
# 01-running-the-states-spec-dirties-tracked-files.sh — spike for card K3
# (states-screenshots-out-of-tree, B-486, triage T-66): in a fresh worktree of `dev`, running
# tests/states-marketing-stats.spec.js ALONE (its own exit code is not the premise) leaves
# tracked PNGs under .night-crew/runs/2026-10-02-autonomous/logs/h4/states/ MODIFIED; the ignored
# home the fix points the default at (test-screenshots/) already exists in .gitignore; and the
# spec's SHOT_DIR is a hard-coded tracked path with no env override.
# Coordinates: TEST_DB_NAME=hq_test_spike_k3_20261007, TEST_PORT=8342, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the run dirtied tracked files AND test-screenshots/ is ignored AND SHOT_DIR has no override
#   exit 1  the tree stayed clean (B-486's premise is wrong), or the ignore convention is missing
#   exit 2  could not run
set -euo pipefail
SPIKE_TAG=k3; PW_PORT=8342
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/../_pwlib.sh"
preflight
worktree_up
STATES_DIR=".night-crew/runs/2026-10-02-autonomous/logs/h4/states"
SPEC="tests/states-marketing-stats.spec.js"

leg "(c) static — the spec's SHOT_DIR constant"
grep -n -A2 "^const SHOT_DIR" "$WT/$SPEC" | sed 's/^/#   /'
grep -q "^const SHOT_DIR = path.join(__dirname, '..', '.night-crew', 'runs'" "$WT/$SPEC" || cannot_run "SHOT_DIR is not the hard-coded runs path any more — re-read the card"
if grep -q "STATES_SHOT_DIR" "$WT/$SPEC"; then fail "the spec already honours STATES_SHOT_DIR — B-486's premise is wrong"; fi
echo "#   SHOT_DIR is hard-coded under $STATES_DIR; no STATES_SHOT_DIR override"
N_TRACKED="$(git -C "$WT" ls-files -- "$STATES_DIR" | wc -l)"
echo "#   tracked PNGs under $STATES_DIR: $N_TRACKED"
[ "$N_TRACKED" -gt 0 ] || cannot_run "no tracked files under $STATES_DIR — nothing to dirty"

leg "(b) — the ignored home exists"
if git -C "$WT" check-ignore -q "test-screenshots/anything.png"; then
  echo "#   test-screenshots/ is gitignored (the .gitignore convention the fix uses)"
else
  fail "test-screenshots/ is NOT ignored — the fix's default home does not exist in .gitignore"
fi

leg "(a) — run the states spec alone on the spike stack (exit code informational)"
[ -z "$(git -C "$WT" status --porcelain -- "$STATES_DIR")" ] || cannot_run "worktree already dirty under $STATES_DIR before the run"
RC="$(pw_run k3 "$SPEC")"
grep -E "passed|failed|flaky|Timed out|Error:" "$LOG/k3.log" | tail -4 | sed 's/^/#   /' || true
exitline "playwright $SPEC" "$RC" "any"
[ -s "$LOG/k3.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/k3.log"
DIRTY="$(git -C "$WT" status --porcelain -- "$STATES_DIR" || true)"
N_DIRTY="$(printf %s "$DIRTY" | grep -c . || true)"
echo "#   modified tracked files after the run: $N_DIRTY"
printf %s "$DIRTY" | head -8 | sed 's/^/#     /'
N_UNTRACKED="$(printf %s "$DIRTY" | grep -c '^??' || true)"
[ "$N_DIRTY" -gt 0 ] || fail "the tree stayed CLEAN after running the states spec — B-486's premise is wrong (see $LOG/k3.log)"

leg "restore"
git -C "$WT" checkout -- "$STATES_DIR"; git -C "$WT" clean -fdq -- "$STATES_DIR"
[ -z "$(git -C "$WT" status --porcelain -- "$STATES_DIR")" ] || cannot_run "could not restore the worktree"
main_tree_untouched
echo
echo "✅ GREEN — running $SPEC alone modified $N_DIRTY tracked file(s) ($N_UNTRACKED new) under $STATES_DIR on dev@$(git -C "$REPO_ROOT" rev-parse --short dev);"
echo "   test-screenshots/ is already ignored and the spec has no STATES_SHOT_DIR override. B-486 reproduced."
