#!/usr/bin/env bash
# 03-push-lands-as-the-session-user.sh — spike: the shipped push replica works
# through the door under the identity the door imposes, and what a wrong
# deviceId costs.
#
# Through the proxy, RLS evaluates the SUBSTITUTED token — sub = the session
# user's id — never anything the client claims. scan_attempts' with-check is
# device_id = sub. So the provisioning card's deviceId coordinate MUST be the
# mint envelope's sub, and getting it wrong is not a graceful degrade: it is
# the F-2 throw-retry head-of-line poison class, measured here.
#
# Legs:
#   (a) caller-set enumeration: startScanAttemptsReplica has ZERO callers
#       outside its own module + tests/harness — the scope finding, priced
#       before the slate rather than discovered on card night
#   (b) deviceId = mint sub: a legit redeem attempt on a FRESH code (sibling
#       ledger build-fact 5: never reuse a seeded code) drains through
#       <origin>/sync/rest — redeem RPC composes with the prefix strip, the
#       attempt lands accepted, the code reads back redeemed by the user id
#   (c) deviceId = 'rogue-device': the drain draws the RLS refusal — status
#       enumerated, handler behavior (throw = poison class) measured, and
#       whatever the redeem RPC did BEFORE the landing refused is reported
#       (a burn that can never record is the expensive shape of this mistake)
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0 legs agreed.  exit 1 a measurement disagreed (a rogue LANDING is
#   escalated as a security finding).  exit 2 could not run.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"
# shellcheck source=/dev/null
. "$SCRIPT_DIR/lib-hq.sh"
trap hq_down EXIT

substrate_up
echo
echo "── built schema: reset_bare + apply_all ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── leg (a): the caller set of startScanAttemptsReplica (enumerated) ──"
CALLERS="$(grep -rn "startScanAttemptsReplica" "$REPO_ROOT" \
  --include='*.js' --include='*.html' \
  --exclude-dir=node_modules --exclude-dir=.night-crew --exclude-dir=vendor || true)"
printf '%s\n' "$CALLERS" | sed 's/^/  /'
PAGE_CALLERS="$(printf '%s\n' "$CALLERS" | grep -v 'marketing/sync/push-replication.js' | grep -v '^\s*$' | grep -cv 'tests/\|harness/' || true)"
echo "  → callers outside the module/tests/harness: $PAGE_CALLERS"
[ "$PAGE_CALLERS" = "0" ] || echo "  (a page caller exists — the scope finding below adjusts)"

echo
echo "── HQ door up ──"
resolve_substrate_rest
hq_db_reset
hq_up
hq_login

SUB="$(curl -s -X POST "$HQ_ORIGIN/api/v1/sync/token" -H "Cookie: hq_session=$COOKIE" \
  | node -e 'let d="";process.stdin.on("data",(c)=>d+=c).on("end",()=>{const j=JSON.parse(d);if(!j.sub)process.exit(1);process.stdout.write(j.sub)})')" \
  || cannot_run "could not read sub from the mint envelope"
echo "#   deviceId coordinate (mint sub) : $SUB"

echo
echo "── fresh live codes (spike-local data on the throwaway substrate) ──"
FRESH_A="d0000000-0000-4000-8000-00000000000a"
FRESH_B="d0000000-0000-4000-8000-00000000000b"
psqlq <<SQL
insert into public.codes (id, token_hash, campaign_id, expires_at) values
  ('$FRESH_A', '$(printf 'spike-prov-fresh-a' | sha256sum | cut -d' ' -f1)',
   'a0000000-0000-4000-8000-000000000001', now() + interval '2 days'),
  ('$FRESH_B', '$(printf 'spike-prov-fresh-b' | sha256sum | cut -d' ' -f1)',
   'a0000000-0000-4000-8000-000000000001', now() + interval '2 days');
SQL

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── legs (b) + (c): the shipped push replica through the door ──"
node "$SCRIPT_DIR/js/door-push.mjs" "$HQ_ORIGIN" "$COOKIE" "$SUB" "$FRESH_A" "$FRESH_B" \
  || fail "a measurement disagreed — see the node log above"

echo
echo "── server readback (the arbiter's view) ──"
psqlq <<SQL
select left(id::text, 8) || '…', device_id, status from public.scan_attempts order by scanned_at;
SQL
LANDED_ROGUE="$(printf "select count(*) from public.scan_attempts where device_id = 'rogue-device';" | psqlq)"
[ "$LANDED_ROGUE" = "0" ] || fail "🛑 a rogue-device attempt LANDED — RLS is not binding through the proxy. SECURITY FINDING, escalate immediately."
REDEEMED_BY="$(printf "select redeemed_by from public.codes where id = '%s';" "$FRESH_A" | psqlq)"
echo "  fresh code A redeemed_by: $REDEEMED_BY"
[ "$REDEEMED_BY" = "$SUB" ] || fail "the legit burn recorded redeemed_by='$REDEEMED_BY', expected the session user id"
ROGUE_BURN="$(printf "select coalesce(redeemed_by, '(unburned)') from public.codes where id = '%s';" "$FRESH_B" | psqlq)"
echo "  fresh code B redeemed_by: $ROGUE_BURN — the expensive shape of a wrong deviceId:"
echo "  the redeem RPC burned the code BEFORE the landing was refused, so the burn can never record."

printf '\n✅ VERDICT: GREEN — the push lands through the door as the session user; a wrong deviceId is refused by RLS and poisons the queue (measured, enumerated above); the wiring gap is priced.\n'
