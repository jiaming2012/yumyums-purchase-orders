#!/usr/bin/env bash
# marketing/sync/harness/run.sh — the standalone gate for card rxdb-pull-replica
# (run 20260905; B-345 precedent: the runnable script IS the verdict).
#
# Drives the PRODUCTION marketing/sync modules (green) — and two deliberately
# defective inline probes (red) — against the LOCAL spike-supabase substrate
# with the BUILT Activity A migrations + seed applied.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  every leg held (green) — or, in a red mode, the probe failed AS
#           EXPECTED (the defect demonstrably reds the assertion).
#   exit 1  a leg failed (green) — or a red probe PASSED (the assertion does
#           not catch the defect class it exists for).
#   exit 2  could not run.
#
# USAGE
#   run.sh            # green — the PRIMARY gate: production modules, all legs
#   run.sh red-gap1   # naive gt-cursor probe: same-updated_at batch boundary
#                     #   silently misses rows (GAP-1 before the fix)
#   run.sh red-window # unbounded probe: out-of-window rows land
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

MODE="${1:-green}"
case "$MODE" in green|red-gap1|red-window) ;; *) echo "usage: run.sh [green|red-gap1|red-window]" >&2; exit 64 ;; esac

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this card's premise"

echo
echo "── per-run rows ──"
IN1="$(newuuid)"; OUT1="$(newuuid)"; TARGET="$(newuuid)"; OFFER="$(newuuid)"
G1="$(newuuid)"; G2="$(newuuid)"; G3="$(newuuid)"; G4="$(newuuid)"; G5="$(newuuid)"
TARGET_HASH="c2-hash-target-$TARGET"
OFFER_HASH="c2-hash-customer-$OFFER"
psqlq <<SQL >/dev/null || cannot_run "per-run seeding failed"
insert into public.codes (id, token_hash, campaign_id, expires_at) values
 ('$IN1',   'c2-hash-in-$IN1',   'a0000000-0000-4000-8000-000000000001', now() - interval '1 day'),
 ('$OUT1',  'c2-hash-out-$OUT1', 'a0000000-0000-4000-8000-000000000001', now() - interval '5 days'),
 ('$TARGET','$TARGET_HASH',      'a0000000-0000-4000-8000-000000000001', now() + interval '1 day'),
 ('$OFFER', '$OFFER_HASH',       'a0000000-0000-4000-8000-000000000001', now() + interval '3 days');
insert into public.codes (id, token_hash, campaign_id, expires_at) values
 ('$G1', 'c2-hash-gap-$G1', 'a0000000-0000-4000-8000-000000000001', now() + interval '2 days'),
 ('$G2', 'c2-hash-gap-$G2', 'a0000000-0000-4000-8000-000000000001', now() + interval '2 days'),
 ('$G3', 'c2-hash-gap-$G3', 'a0000000-0000-4000-8000-000000000001', now() + interval '2 days'),
 ('$G4', 'c2-hash-gap-$G4', 'a0000000-0000-4000-8000-000000000001', now() + interval '2 days'),
 ('$G5', 'c2-hash-gap-$G5', 'a0000000-0000-4000-8000-000000000001', now() + interval '2 days');
-- GAP-1's subject: ONE statement stamps all five with an IDENTICAL updated_at,
-- strictly later than every other row — the same-timestamp tie group a batch
-- boundary must fall inside (harness batchSize=2 < 5 guarantees it).
update public.codes set updated_at = now()
 where id in ('$G1','$G2','$G3','$G4','$G5');
SQL
echo "  in-window(expired-1d)  = $IN1"
echo "  out-of-window(exp-5d)  = $OUT1"
echo "  live burn target       = $TARGET"
echo "  synced customer offer  = $OFFER  (hash: $OFFER_HASH)"
echo "  GAP-1 tie group (5)    = $G1 $G2 $G3 $G4 $G5"

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
C2_JWT_A="$TOKEN_A" C2_JWT_B="$TOKEN_B" \
C2_IN1="$IN1" C2_OUT1="$OUT1" \
C2_TARGET="$TARGET" C2_TARGET_HASH="$TARGET_HASH" \
C2_OFFER="$OFFER" C2_OFFER_HASH="$OFFER_HASH" \
C2_GAP_IDS="$G1,$G2,$G3,$G4,$G5" \
node "$SCRIPT_DIR/harness.mjs" "$MODE"
NODE_EXIT=$?
set -e

# Per-run rows out (throwaway substrate hygiene; failure here is not a verdict).
psqlq <<<"delete from public.codes where id in ('$IN1','$OUT1','$TARGET','$OFFER','$G1','$G2','$G3','$G4','$G5');" >/dev/null 2>&1 || true

if [ "$MODE" = "green" ]; then
  if [ "$NODE_EXIT" -eq 0 ]; then
    printf '\n✅ VERDICT: GREEN — both bounded, keyset-checkpointed replicas hold on the built schema; redeem() propagates live; offers resolve offline; GAP-1 boundary walked.\n'
  else
    printf '\n🛑 VERDICT: RED — a leg failed (node exit %s); see the log above.\n' "$NODE_EXIT"
  fi
else
  # Red modes propagate the node exit code so the captured log carries the
  # honest EXIT — a demonstrated red is EXIT=1. A red probe that PASSES its
  # assertion (EXIT=0) means the assertion cannot catch the defect class: that
  # is a harness defect, and the line below says so in the capture.
  if [ "$NODE_EXIT" -ne 0 ]; then
    printf '\n🔴 RED-FIRST DEMONSTRATED (%s): the defective probe reds the assertion (node exit %s). This non-zero exit IS the evidence.\n' "$MODE" "$NODE_EXIT"
  else
    printf '\n🛑 RED PROBE PASSED (%s): the assertion does NOT catch the defect class — the harness is not evidence. Fix the harness.\n' "$MODE"
  fi
fi
exit "$NODE_EXIT"
