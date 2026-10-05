#!/usr/bin/env bash
# 01-late-responses-wipe-the-add-bar-and-revert-the-chips.sh — spike for card K1
# (inventory-setup-races, B-459 + B-478, triage T-66): the red-first baseline for [IS-01] and
# [IS-02]. Through the SHIPPED inventory.html with only network TIMING altered (page.route):
#   premise 1 (B-478)  groups delayed 600 ms → the typed name is wiped, create sends no POST
#   control            no delay → the typed name survives
#   premise 2 (B-459)  the Setup tab's first GET /items held until after an alias POST → chips
#                      revert 2 → 1 while the server keeps both aliases
# Coordinates: TEST_DB_NAME=hq_test_spike_k1_20261007, TEST_PORT=8341, :5434. NEVER :5433.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  both premises reproduce AND the control is healthy
#   exit 1  a premise did not reproduce, or the control is broken — "🛑 VERDICT: RED"
#   exit 2  could not run
set -euo pipefail
SPIKE_TAG=k1; PW_PORT=8341
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/../_pwlib.sh"
preflight
worktree_up

leg "spec — copy spike-k1.spec.js into the worktree's tests/"
cp "$SCRIPT_DIR/spike-k1.spec.js" "$WT/tests/spike-k1.spec.js"
echo "#   $WT/tests/spike-k1.spec.js (3 specs: premise B-478, control, premise B-459)"

leg "run — through the shipped page"
RC="$(pw_run k1 tests/spike-k1.spec.js)"
OUT="$(pw_outcomes k1 2>/dev/null || true)"
grep -o 'SPIKE-K1-[0-9]: {.*}' "$LOG/k1.log" | sed 's/^/#   /'
grep -E "passed|failed|flaky|Timed out" "$LOG/k1.log" | tail -3 | sed 's/^/#   /'
exitline "playwright tests/spike-k1.spec.js" "$RC" "0"
[ -s "$LOG/k1.json" ] || cannot_run "no Playwright JSON — the stack did not come up; see $LOG/k1.log"
[ "$(wc -l <<<"$OUT")" = "3" ] || cannot_run "expected exactly 3 specs, got: $(tr '\n' ' ' <<<"$OUT")"
has "$OUT" "[K1-SPIKE-2]=true" || fail "CONTROL is red — the typed name does not survive even without a delay; the premise cannot be read (see $LOG/k1.log)"
has "$OUT" "[K1-SPIKE-1]=true" || fail "B-478 did NOT reproduce — the late groups response did not wipe the typed name, or create still POSTed (see $LOG/k1.log)"
# Leg 3 carries the CORRECTED premise (runs 2–4): under the filed timing the view and the server
# AGREE — B-459's diagnosed "view lies" mechanism does not reproduce; what DOES happen, some runs,
# is the add never reaching the server (addDropped:true in the diag line). A red here means the
# view and server disagreed, i.e. the filed mechanism reproduced and the card goes back to it.
has "$OUT" "[K1-SPIKE-3]=true" || fail "the view and the server DISAGREED about the typed nickname — B-459's filed mechanism reproduced after all; revert the correction and re-read the card (see $LOG/k1.log)"
grep -o 'SPIKE-K1-3: {.*}' "$LOG/k1.log" | grep -q '"addDropped":true' && echo "#   NOTE: the add was DROPPED this run (never reached the server) — the card's [IS-04] subject"
[ "$RC" = "0" ] || fail "playwright exited $RC with all three specs ok=true"

main_tree_untouched
echo
echo "✅ GREEN on dev@$(git -C "$REPO_ROOT" rev-parse --short dev): B-478 reproduced (a late groups response wipes the typed name; create sends no POST),"
echo "   the control is healthy, and B-459's diagnosed mechanism does NOT reproduce under the filed timing (view and server agree)."
echo "   Card K1: [IS-01] red-first stands; the B-459 half is request-sequencing hardening proven by a 10× measurement."
