#!/usr/bin/env bash
# marketing/sync/harness/clock-run.sh — the standalone gate for card
# clock-offset-on-sync (run 20260905; B-345 precedent: the runnable script IS
# the verdict). A SIBLING of run.sh (Card 2) and push-run.sh (Card 3) —
# deliberately not an edit to them; the landed gates stay re-runnable.
#
# Drives the PRODUCTION marketing/sync modules with the sync clock injected
# (green) — and a deliberately naive inline probe (red) — against the LOCAL
# spike-supabase substrate with the BUILT Activity A migrations + seed applied.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  every leg held (green) — or, in red mode, the probe failed AS
#           EXPECTED (the defect demonstrably reds the assertion).
#   exit 1  a leg failed (green) — or the red probe PASSED (the assertion does
#           not catch the defect class it exists for).
#   exit 2  could not run.
#
# USAGE
#   clock-run.sh           # green — the PRIMARY gate: production clock, all legs
#   clock-run.sh red-skew  # naive deviceNow comparison under ±2d skew:
#                          #   behind-skew ACCEPTS the dead code (§5.1's hole),
#                          #   ahead-skew falsely REJECTS the valid code
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

MODE="${1:-green}"
case "$MODE" in green|red-skew) ;; *) echo "usage: clock-run.sh [green|red-skew]" >&2; exit 64 ;; esac

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this card's premise"

echo
echo "── per-run rows ──"
DEAD="$(newuuid)"; VALID="$(newuuid)"
DEAD_HASH="c4-hash-dead-$DEAD"
VALID_HASH="c4-hash-valid-$VALID"
# DEAD: expired ~1 day ago, INSIDE the §5.3 2-day codes window — the exact
#   dangerous row a rolled-back clock resurrects (§5.1).
# VALID: expires in ~1 day — INSIDE a 2-days-ahead clock's false-rejection
#   band (an unadjusted fast clock reads it as already expired).
psqlq <<SQL >/dev/null || cannot_run "per-run seeding failed"
insert into public.codes (id, token_hash, campaign_id, expires_at) values
 ('$DEAD',  '$DEAD_HASH',  'a0000000-0000-4000-8000-000000000001', now() - interval '1 day'),
 ('$VALID', '$VALID_HASH', 'a0000000-0000-4000-8000-000000000001', now() + interval '1 day');
SQL
echo "  dead (expired -1d, in-window) = $DEAD"
echo "  valid (expires +1d)           = $VALID  (hash: $VALID_HASH)"

echo
echo "── device JWT (throwaway secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here; the link is gitignored).
[ -e "$SCRIPT_DIR/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/node_modules"

echo
echo "── mode: $MODE ──"
set +e
C4_JWT="$TOKEN" \
C4_DEAD="$DEAD" C4_DEAD_HASH="$DEAD_HASH" \
C4_VALID="$VALID" C4_VALID_HASH="$VALID_HASH" \
node "$SCRIPT_DIR/clock-harness.mjs" "$MODE"
NODE_EXIT=$?
set -e

# Per-run rows out (throwaway substrate hygiene; failure here is not a verdict).
psqlq <<<"delete from public.codes where id in ('$DEAD','$VALID');" >/dev/null 2>&1 || true

if [ "$MODE" = "green" ]; then
  if [ "$NODE_EXIT" -eq 0 ]; then
    printf '\n✅ VERDICT: GREEN — offset captured from the pull Date header on every successful pull; offline expiry rejects the dead code and accepts the valid one under BOTH ±2d skews; window bounds follow the adjusted clock.\n'
  else
    printf '\n🛑 VERDICT: RED — a leg failed (node exit %s); see the log above.\n' "$NODE_EXIT"
  fi
else
  # Red mode propagates the node exit code so the captured log carries the
  # honest EXIT — a demonstrated red is EXIT=1. A red probe that PASSES its
  # assertion (EXIT=0) means the assertion cannot catch the defect class: that
  # is a harness defect, and the line below says so in the capture.
  if [ "$NODE_EXIT" -ne 0 ]; then
    printf '\n🔴 RED-FIRST DEMONSTRATED (%s): the naive deviceNow comparison reds the assertion (node exit %s). This non-zero exit IS the evidence.\n' "$MODE" "$NODE_EXIT"
  else
    printf '\n🛑 RED PROBE PASSED (%s): the assertion does NOT catch the defect class — the harness is not evidence. Fix the harness.\n' "$MODE"
  fi
fi
exit "$NODE_EXIT"
