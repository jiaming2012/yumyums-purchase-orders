#!/usr/bin/env bash
# 01-pull-bounded-checkpoint.sh — spike: the Activity B pull premise on the
# BUILT Activity A artifacts. An RxDB client (replicateRxCollection, custom
# pull handler → PostgREST as a device JWT):
#   (a) initial sync lands the in-window codes and NOT a code outside the §5.3
#       two-day window (the bounded negative);
#   (b) pull resumption is observably checkpointed on updated_at (request log
#       enumerated — the post-initial request carries a non-epoch cursor);
#   (c) the done_when core: a code burned by the committed redeem() RPC as
#       device-a surfaces as redeemed in device-b's RUNNING local replica via
#       Realtime nudge → pull tick — no client restart, no manual reSync.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (a)+(b)+(c) held.   exit 1  a leg failed.   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only; never :5433, never :5434, no hosted project.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Card 1's gate, not this spike's premise"

echo
echo "── per-run rows: in-window-expired, OUT-of-window-expired, live target ──"
IN1="$(newuuid)"; OUT1="$(newuuid)"; TARGET="$(newuuid)"
psqlq <<SQL >/dev/null || cannot_run "per-run seeding failed"
insert into public.codes (id, token_hash, campaign_id, expires_at) values
 ('$IN1',  'spike-pull-in-$IN1',   'a0000000-0000-4000-8000-000000000001', now() - interval '1 day'),
 ('$OUT1', 'spike-pull-out-$OUT1', 'a0000000-0000-4000-8000-000000000001', now() - interval '5 days'),
 ('$TARGET','spike-pull-tgt-$TARGET','a0000000-0000-4000-8000-000000000001', now() + interval '1 day');
SQL
echo "  in-window(expired-1d)=$IN1"
echo "  out-of-window(expired-5d)=$OUT1"
echo "  live-target=$TARGET"

echo
echo "── device JWTs (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"
TOKEN_B="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-b -role authenticated -ttl 30m)" || cannot_run "mint device-b failed"

# Node module resolution: the spike js dir borrows the proven QA rxdb
# node_modules via symlink (walk-up resolution; nothing installed here).
[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── the three legs (node, RxDB memory storage — device-b is the replica) ──"
node "$SCRIPT_DIR/js/pull-bounded-checkpoint.mjs" \
  "$TOKEN_A" "$TOKEN_B" "$TARGET" "$IN1" "$OUT1" \
  || fail "a leg red — see the node log above"

psqlq <<<"delete from public.codes where id in ('$IN1','$OUT1','$TARGET');" >/dev/null 2>&1 || true
printf '\n✅ VERDICT: GREEN — bounded, checkpointed pull holds on the built schema; a real redeem() propagates to a second device live.\n'
