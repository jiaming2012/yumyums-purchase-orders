#!/usr/bin/env bash
# 01-scorecard-reads-hand-run-records.sh — spike: the card's central premise —
# "emit the per-run scorecard files the CLI ALREADY reads" — is true for the
# installed binary: a hand-authored `<run-id>.jsonl` in the committed location
# renders record-backed team rows, no CLI change required.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all three legs held:
#            (1) baseline — on the REAL repo, `night-crew scorecard` shows the
#                Q-KR4 red ("No runs to show."), and the committed scorecard dir
#                holds no per-run file (enumerated)
#            (2) a scratch repo with ONE hand-written
#                .night-crew/knowledge/scorecard/<run-id>.jsonl (the shape the
#                CLI's own union read documents: kind=run-scorecard + kind=team)
#                renders all four roles' team rows and no "No runs to show."
#            (3) the real repo's scorecard state is byte-untouched
#   exit 1   a leg failed — if (2) fails, that IS the card's OR-arm finding
#            (the CLI can't read any target-side file) and belongs clone-side.
#   exit 2   could not run.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
SC_DIR="$REPO_ROOT/.night-crew/knowledge/scorecard"
RUN_ID="hq-20260903-hand"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

command -v night-crew >/dev/null 2>&1 || cannot_run "night-crew CLI not on PATH"

echo "# target coordinates (read-only on the real repo; all writes go to a scratch repo):"
echo "#   real scorecard dir : $SC_DIR"
echo "#   binary             : $(command -v night-crew) ($(night-crew version 2>/dev/null | head -1 || echo '?'))"
STATE_BEFORE="$(ls -la "$SC_DIR" 2>/dev/null | shasum -a 256 | awk '{print $1}')"

echo
echo "── leg 1: the Q-KR4 red, on the real repo ──"
echo "  committed per-run files (enumerated):"
ls "$SC_DIR" 2>/dev/null | sed 's/^/    /'
RUN_FILES="$(ls "$SC_DIR" 2>/dev/null | grep -v '^milestones\.jsonl$' | grep -c '\.jsonl$' || true)"
echo "  per-run .jsonl count (excluding the milestones marker log): $RUN_FILES"
[ "$RUN_FILES" = 0 ] || fail "per-run scorecard files already exist — the red this card retires is gone; re-baseline before slating"
SC_OUT="$(night-crew scorecard -repo "$REPO_ROOT" 2>&1 || true)"
echo "$SC_OUT" | head -4 | sed 's/^/  | /'
echo "$SC_OUT" | grep -q "No runs to show" || fail "expected the 'No runs to show.' red on the real repo; the baseline moved"

echo
echo "── leg 2: a hand-written per-run file renders team rows (scratch repo) ──"
WORK="$(mktemp -d)"; trap 'rm -rf "$WORK"' EXIT
SCRATCH="$WORK/repo"
mkdir -p "$SCRATCH/.night-crew/knowledge/scorecard"
cat > "$SCRATCH/.night-crew/knowledge/scorecard/$RUN_ID.jsonl" <<JSONL
{"schema":1,"kind":"run-scorecard","run_id":"$RUN_ID","ts":"2026-09-03T06:00:00Z","work_orders":4,"points_committed":8,"points_completed":8,"velocity":1,"first_pass_rate":1,"merges_standing":4,"regressions":0}
{"schema":1,"kind":"team","run_id":"$RUN_ID","ts":"2026-09-03T06:00:00Z","team":"product","rating":80,"points_completed":2,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"$RUN_ID","ts":"2026-09-03T06:00:00Z","team":"delivery","rating":80,"points_completed":2,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"$RUN_ID","ts":"2026-09-03T06:00:00Z","team":"engineering","rating":80,"points_completed":2,"value_per_token":0.01}
{"schema":1,"kind":"team","run_id":"$RUN_ID","ts":"2026-09-03T06:00:00Z","team":"qa","rating":80,"points_completed":2,"value_per_token":0.01}
JSONL
echo "  wrote $SCRATCH/.night-crew/knowledge/scorecard/$RUN_ID.jsonl (1 run-scorecard + 4 team records)"
set +e
S2_OUT="$(night-crew scorecard -repo "$SCRATCH" 2>&1)"
S2_RC=$?
set -e
echo "  exit=$S2_RC — output:"
echo "$S2_OUT" | sed 's/^/  | /'
[ "$S2_RC" -eq 0 ] || fail "scorecard exited $S2_RC on the scratch repo"
echo "$S2_OUT" | grep -q "No runs to show" && fail "the hand-written run file was not read — 'No runs to show.' persists (this is the OR-arm finding: record it clone-side)"
for team in product delivery engineering qa; do
  echo "$S2_OUT" | grep -q "$team" || fail "team '$team' does not render from its hand-written record (OR-arm finding — record it clone-side)"
done
echo "$S2_OUT" | grep -q "$RUN_ID" || fail "the run id never renders — the union read did not pick the file up as a run"
echo "  ✓ all four roles render from one hand-authored committed file"

echo
echo "── leg 3: the real repo's scorecard state is untouched ──"
STATE_AFTER="$(ls -la "$SC_DIR" 2>/dev/null | shasum -a 256 | awk '{print $1}')"
[ "$STATE_BEFORE" = "$STATE_AFTER" ] || fail "the real scorecard dir changed — stop and look"
echo "  ✓ unchanged"

printf '\n✅ VERDICT: GREEN — the installed CLI reads a hand-authored per-run scorecard file; the card is target-side work, no CLI change needed for rendering\n'
exit 0
