#!/usr/bin/env bash
# 01-push-lost-race-flips.sh — spike: the losing device's round trip. Two RxDB
# device clients each hold a locally-queued pending attempt for the SAME code
# (the §8 offline double-accept); both push concurrently through the committed
# redeem(); exactly one local row ends accepted, the loser's row flips to
# rejected/already_used carrying the winning device + time; both attempt rows
# land server-side under RLS; the write-back does not loop; and the device
# role still cannot SELECT scan_attempts (push-only holds at the API surface).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  all legs held.   exit 1  a leg failed.   exit 2  could not run.
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

TARGET="$(newuuid)"
psqlq <<SQL >/dev/null || cannot_run "per-run seeding failed"
insert into public.codes (id, token_hash, campaign_id, expires_at)
values ('$TARGET', 'spike-push-$TARGET', 'a0000000-0000-4000-8000-000000000001', now() + interval '1 day');
SQL
echo "  race target code=$TARGET"

echo
echo "── device JWTs ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"
TOKEN_B="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-b -role authenticated -ttl 30m)" || cannot_run "mint device-b failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── the client legs (node — two RxDB devices, concurrent push) ──"
node "$SCRIPT_DIR/js/push-lost-race.mjs" "$TOKEN_A" "$TOKEN_B" "$TARGET" \
  || fail "a client leg red — see the node log above"

echo
echo "── server-side enumeration (supabase_admin via psql — devices provably cannot read this) ──"
ROWS="$(psqlq <<<"select device_id||'|'||status||'|'||coalesce(reason,'-')||'|'||offline_override from public.scan_attempts where code_id = '$TARGET' order by device_id;")"
echo "$ROWS" | sed 's/^/  /'
N="$(echo "$ROWS" | grep -c . || true)"
[ "$N" = "2" ] || fail "expected exactly 2 server attempt rows for the code, found $N"
echo "$ROWS" | grep -q '^device-a|' || fail "device-a's attempt row missing server-side"
echo "$ROWS" | grep -q '^device-b|' || fail "device-b's attempt row missing server-side"
ACC="$(echo "$ROWS" | grep -c '|accepted|' || true)"
REJ="$(echo "$ROWS" | grep -c '|rejected|already_used|' || true)"
[ "$ACC" = "1" ] || fail "expected exactly 1 accepted attempt, found $ACC"
[ "$REJ" = "1" ] || fail "expected exactly 1 rejected/already_used attempt, found $REJ"

WINNER="$(psqlq <<<"select redeemed_by from public.codes where id = '$TARGET';")"
echo "  codes.redeemed_by = $WINNER"
ACC_DEV="$(echo "$ROWS" | grep '|accepted|' | cut -d'|' -f1)"
[ "$WINNER" = "$ACC_DEV" ] || fail "codes.redeemed_by ($WINNER) does not match the accepted attempt's device ($ACC_DEV)"

psqlq <<<"delete from public.scan_attempts where code_id = '$TARGET'; delete from public.codes where id = '$TARGET';" >/dev/null 2>&1 || true
printf '\n✅ VERDICT: GREEN — the lost race flips on the losing device with winner info; push-only RLS holds; the write-back terminates.\n'
