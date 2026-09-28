#!/usr/bin/env bash
# marketing/sync/harness/push-run.sh — the standalone gate for card
# scan-attempts-push-conflict (run 20260905; B-345 precedent: the runnable
# script IS the verdict). A SIBLING of Card 2's run.sh — that landed gate stays
# byte-identical and independently re-runnable.
#
# Drives the PRODUCTION marketing/sync push module (green) — and a deliberately
# defective inline naive probe (red) — against the LOCAL spike-supabase
# substrate with the BUILT Activity A migrations + seed applied.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  every leg held (green) — or, in the red mode, the probe failed AS
#           EXPECTED (the defect demonstrably reds the assertion).
#   exit 1  a leg failed (green) — or the red probe PASSED (the assertion does
#           not catch the defect class it exists for).
#   exit 2  could not run.
#
# USAGE
#   push-run.sh            # green — the PRIMARY gate: production module, all legs
#   push-run.sh red-gap1   # naive redeem-then-land probe under an injected
#                          #   landing failure: the WINNER's UI mis-flips
#                          #   (GAP-1's land-fails-after-redeem window, before
#                          #   the fix)
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

MODE="${1:-green}"
case "$MODE" in green|red-gap1) ;; *) echo "usage: push-run.sh [green|red-gap1]" >&2; exit 64 ;; esac

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this card's premise"

echo
echo "── per-run rows ──"
TARGET="$(newuuid)"; W1="$(newuuid)"; W2="$(newuuid)"
psqlq <<SQL >/dev/null || cannot_run "per-run seeding failed"
insert into public.codes (id, token_hash, campaign_id, expires_at) values
 ('$TARGET','c3-hash-race-$TARGET','a0000000-0000-4000-8000-000000000001', now() + interval '1 day'),
 ('$W1',    'c3-hash-w1-$W1',      'a0000000-0000-4000-8000-000000000001', now() + interval '1 day'),
 ('$W2',    'c3-hash-w2-$W2',      'a0000000-0000-4000-8000-000000000001', now() + interval '1 day');
SQL
echo "  race target code        = $TARGET"
echo "  land-fail window code   = $W1  (GAP-1 belt 1)"
echo "  lost-response code      = $W2  (GAP-1 belt 2)"

echo
echo "── device JWTs (throwaway secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"
TOKEN_B="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-b -role authenticated -ttl 30m)" || cannot_run "mint device-b failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here; the link is gitignored).
[ -e "$SCRIPT_DIR/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/node_modules"

echo
echo "── mode: $MODE ──"
set +e
C3_JWT_A="$TOKEN_A" C3_JWT_B="$TOKEN_B" \
C3_TARGET="$TARGET" C3_W1="$W1" C3_W2="$W2" \
node "$SCRIPT_DIR/push-harness.mjs" "$MODE"
NODE_EXIT=$?
set -e

if [ "$MODE" = "green" ] && [ "$NODE_EXIT" -eq 0 ]; then
  echo
  echo "── server-side enumeration (supabase_admin via psql — devices provably cannot read this) ──"
  ROWS="$(psqlq <<<"select device_id||'|'||status||'|'||coalesce(reason,'-') from public.scan_attempts where code_id = '$TARGET' order by device_id;")"
  echo "  race code attempts:"
  echo "$ROWS" | sed 's/^/    /'
  N="$(echo "$ROWS" | grep -c . || true)"
  [ "$N" = "2" ] || { NODE_EXIT=1; echo "🛑 expected exactly 2 server attempt rows for the race code, found $N"; }
  ACC="$(echo "$ROWS" | grep -c '|accepted|' || true)"
  REJ="$(echo "$ROWS" | grep -c '|rejected|already_used' || true)"
  [ "$ACC" = "1" ] || { NODE_EXIT=1; echo "🛑 expected exactly 1 accepted attempt, found $ACC"; }
  [ "$REJ" = "1" ] || { NODE_EXIT=1; echo "🛑 expected exactly 1 rejected/already_used attempt, found $REJ"; }
  WINNER="$(psqlq <<<"select redeemed_by from public.codes where id = '$TARGET';")"
  ACC_DEV="$(echo "$ROWS" | grep '|accepted|' | cut -d'|' -f1 || true)"
  echo "  codes.redeemed_by = $WINNER (accepted attempt's device: $ACC_DEV)"
  [ "$WINNER" = "$ACC_DEV" ] || { NODE_EXIT=1; echo "🛑 codes.redeemed_by does not match the accepted attempt's device"; }

  W1ROW="$(psqlq <<<"select device_id||'|'||status||'|'||coalesce(reason,'-') from public.scan_attempts where code_id = '$W1';")"
  echo "  W1 (land-fail) attempt: $W1ROW"
  [ "$W1ROW" = "device-a|accepted|-" ] || { NODE_EXIT=1; echo "🛑 W1's landed row is not device-a|accepted (the retried landing must carry the WINNING outcome)"; }
  W2ROW="$(psqlq <<<"select device_id||'|'||status||'|'||coalesce(reason,'-') from public.scan_attempts where code_id = '$W2';")"
  echo "  W2 (lost-response) attempt: $W2ROW"
  [ "$W2ROW" = "device-b|accepted|-" ] || { NODE_EXIT=1; echo "🛑 W2's landed row is not device-b|accepted (own-device already_used must land as accepted)"; }
fi

# Per-run rows out (throwaway substrate hygiene; failure here is not a verdict).
psqlq <<<"delete from public.scan_attempts where code_id in ('$TARGET','$W1','$W2'); delete from public.codes where id in ('$TARGET','$W1','$W2');" >/dev/null 2>&1 || true

if [ "$MODE" = "green" ]; then
  if [ "$NODE_EXIT" -eq 0 ]; then
    printf '\n✅ VERDICT: GREEN — offline queue, concurrent push, exactly-one-winner, pull-replica-sourced loser flip, both GAP-1 windows survived, write-only RLS holds.\n'
  else
    printf '\n🛑 VERDICT: RED — a leg failed (exit %s); see the log above.\n' "$NODE_EXIT"
  fi
else
  if [ "$NODE_EXIT" -ne 0 ]; then
    printf '\n🔴 RED-FIRST DEMONSTRATED (%s): the naive probe reds the assertion (node exit %s). This non-zero exit IS the evidence.\n' "$MODE" "$NODE_EXIT"
  else
    printf '\n🛑 RED PROBE PASSED (%s): the assertion does NOT catch the defect class — the harness is not evidence. Fix the harness.\n' "$MODE"
  fi
fi
exit "$NODE_EXIT"
