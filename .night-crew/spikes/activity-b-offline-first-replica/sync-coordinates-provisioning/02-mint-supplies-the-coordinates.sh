#!/usr/bin/env bash
# 02-mint-supplies-the-coordinates.sh — spike: everything provisioning must
# write is derivable at PAGE INIT from the session alone; degradations fail
# closed; a session outlives any bearer concern.
#
# Legs, enumerated as a set (B-216):
#   1. cookie → POST /api/v1/sync/token → 200 {token, expires_at, sub, role,
#      grants}; sub == the logged-in user's id (the deviceId coordinate);
#      exp − iat == 900s (DefaultTokenTTL, measured not read)
#   2. the minted token is a REAL substrate credential: accepted DIRECT by
#      PostgREST (even though spike 01 proves it inert at the door)
#   3. no cookie → 401 (no anonymous mint)
#   4. degraded deploy (HQ_SYNC_JWT_SECRET + HQ_SYNC_REST_URL unset):
#      /sync/token → 503 sync_bridge_not_configured; /sync/rest → refuses
#      (status enumerated) — fail closed, never fail open
#   5. longevity: the session row's expires_at IS NULL, and after aging
#      created_at by 30 days the SAME cookie still mints — a provisioned
#      device holds as long as its session row exists
#   6. the SYNC_KEY writer set in the tree is exactly the declaration —
#      nothing conflicts with the writer the card adds
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0 all legs agreed.  exit 1 a measurement disagreed.  exit 2 could not run.

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
echo "── HQ door up ──"
resolve_substrate_rest
hq_db_reset
hq_up
hq_login

echo
echo "── leg 1: the mint envelope, with only the cookie ──"
ENVELOPE="$(curl -s -w '\n%{http_code}' -X POST "$HQ_ORIGIN/api/v1/sync/token" -H "Cookie: hq_session=$COOKIE")"
MINT_STATUS="$(printf '%s' "$ENVELOPE" | tail -1)"
MINT_BODY="$(printf '%s' "$ENVELOPE" | sed '$d')"
[ "$MINT_STATUS" = "200" ] || fail "mint with cookie answered HTTP $MINT_STATUS: $MINT_BODY"

USER_ID="$(printf "select id from users where email = '%s';" "$ADMIN_EMAIL" | hq_psql)"
[ -n "$USER_ID" ] || cannot_run "could not read the superadmin's user id from the spike DB"

TOKEN="$(printf '%s' "$MINT_BODY" | USER_ID="$USER_ID" node -e '
let d = "";
process.stdin.on("data", (c) => d += c).on("end", () => {
  const j = JSON.parse(d);
  const missing = ["token","expires_at","sub","role","grants"].filter((k) => !(k in j));
  if (missing.length) { console.error(`envelope missing: ${missing.join(",")}`); process.exit(1); }
  const [, payloadB64] = j.token.split(".");
  const p = JSON.parse(Buffer.from(payloadB64, "base64url").toString());
  const ttl = p.exp - p.iat;
  console.error(`  envelope: sub=${j.sub} role=${j.role} grants=[${j.grants}]`);
  console.error(`  claims:   exp−iat=${ttl}s  role=${p.role}  sub=${p.sub}`);
  if (ttl !== 900) { console.error(`exp−iat=${ttl}, expected 900 (DefaultTokenTTL)`); process.exit(1); }
  if (j.sub !== process.env.USER_ID) { console.error(`sub=${j.sub} != users.id=${process.env.USER_ID} — the deviceId coordinate would be wrong`); process.exit(1); }
  if (p.role !== "authenticated") { console.error(`role=${p.role}, RLS grants are to authenticated`); process.exit(1); }
  process.stdout.write(j.token);
})' )" || fail "the mint envelope disagreed — see above"

echo
echo "── leg 2: the minted token is a real substrate credential (direct PostgREST) ──"
DIRECT_STATUS="$(curl -s -o /dev/null -w '%{http_code}' "$REST_DIRECT/campaigns?select=id&limit=1" -H "Authorization: Bearer $TOKEN")"
echo "  direct GET /campaigns with the minted token → HTTP $DIRECT_STATUS"
[ "$DIRECT_STATUS" = "200" ] || fail "PostgREST refused the minted token (HTTP $DIRECT_STATUS) — the coordinate is not a real credential"

echo
echo "── leg 3: no cookie → no mint ──"
ANON_STATUS="$(curl -s -o /dev/null -w '%{http_code}' -X POST "$HQ_ORIGIN/api/v1/sync/token")"
echo "  anonymous mint → HTTP $ANON_STATUS"
[ "$ANON_STATUS" = "401" ] || fail "anonymous mint answered $ANON_STATUS, expected 401"

echo
echo "── leg 4: the degraded deploy fails CLOSED (both rooms) ──"
hq_up_degraded
hq_login "$HQ2_ORIGIN"; COOKIE2="$COOKIE"
D_MINT="$(curl -s -w '\n%{http_code}' -X POST "$HQ2_ORIGIN/api/v1/sync/token" -H "Cookie: hq_session=$COOKIE2")"
D_MINT_STATUS="$(printf '%s' "$D_MINT" | tail -1)"; D_MINT_BODY="$(printf '%s' "$D_MINT" | sed '$d')"
echo "  mint on degraded → HTTP $D_MINT_STATUS  $D_MINT_BODY"
[ "$D_MINT_STATUS" = "503" ] || fail "mint on a secret-less deploy answered $D_MINT_STATUS, expected 503"
printf '%s' "$D_MINT_BODY" | grep -q 'sync_bridge_not_configured' || fail "503 without the named error"
D_DOOR_STATUS="$(curl -s -o /dev/null -w '%{http_code}' "$HQ2_ORIGIN/sync/rest/campaigns?select=id&limit=1" -H "Cookie: hq_session=$COOKIE2")"
echo "  /sync/rest on degraded → HTTP $D_DOOR_STATUS (enumerated — the card's provisioning must survive this without breaking the scanner)"
case "$D_DOOR_STATUS" in
  2*) fail "the unconfigured door answered $D_DOOR_STATUS — fail OPEN" ;;
esac
hq_login   # restore $COOKIE for the main instance

echo
echo "── leg 5: longevity — the session row, aged, still mints ──"
NULL_CHECK="$(printf "select (expires_at is null) from sessions s join users u on u.id = s.user_id where u.email = '%s' order by s.created_at desc limit 1;" "$ADMIN_EMAIL" | hq_psql)"
echo "  newest session expires_at IS NULL → $NULL_CHECK"
[ "$NULL_CHECK" = "t" ] || fail "sessions are minted WITH an expiry — the provisioning-holds-forever premise is false, the card owes refresh handling"
printf "update sessions set created_at = created_at - interval '30 days' where user_id = '%s';" "$USER_ID" | hq_psql >/dev/null
AGED_STATUS="$(curl -s -o /dev/null -w '%{http_code}' -X POST "$HQ_ORIGIN/api/v1/sync/token" -H "Cookie: hq_session=$COOKIE")"
echo "  mint with the 30-day-aged session → HTTP $AGED_STATUS"
[ "$AGED_STATUS" = "200" ] || fail "an aged session stopped minting (HTTP $AGED_STATUS) — provisioning would silently die at that age"

echo
echo "── leg 6: the SYNC_KEY writer set (enumerated, whole tree) ──"
MATCHES="$(grep -rn "hq_marketing_sync_v1" "$REPO_ROOT" \
  --include='*.js' --include='*.html' --include='*.go' \
  --exclude-dir=node_modules --exclude-dir=.night-crew --exclude-dir=vendor || true)"
printf '%s\n' "$MATCHES" | sed 's/^/  /'
COUNT="$(printf '%s\n' "$MATCHES" | grep -c . || true)"
[ "$COUNT" = "1" ] || fail "expected exactly 1 occurrence (the scan-page.js declaration), found $COUNT — a second writer/reader has appeared and the card must reconcile with it"

printf '\n✅ VERDICT: GREEN — page init has everything provisioning needs (cookie → envelope with sub); TTL is the constant; degradations fail closed; an aged session still mints; the card adds the only SYNC_KEY writer.\n'
