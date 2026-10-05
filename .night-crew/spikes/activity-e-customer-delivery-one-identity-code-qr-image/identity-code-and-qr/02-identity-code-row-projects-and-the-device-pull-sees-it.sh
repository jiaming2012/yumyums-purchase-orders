#!/usr/bin/env bash
# 02-identity-code-row-projects-and-the-device-pull-sees-it.sh — spike for card E1
# (identity-code-and-qr, D-KR3, §10): an identity-code row written to the arbiter's `codes`
# table the way projection.go writes campaigns (service_role upsert over PostgREST,
# merge-duplicates) is visible to a DEVICE through the tablet's own offers-pull query under the
# shipped RLS, the upsert is idempotent, and the row is removed on exit. Runs against the LOCAL
# spike-supabase substrate only — the committed TEST campaign a0…0001 is the campaign it names.
# Not :5433, not :5434, not a hosted project.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  device sees nothing before; upsert lands; device pull contains the hash; re-upsert
#           leaves one row; delete restores the "nothing" state.
#   exit 1  PostgREST refused, the device cannot see the row, or two rows appeared.
#   exit 2  could not run (substrate down, mintjwt failed, fixture campaign missing).
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
leg()        { printf '\n── %s ──\n' "$1"; }
export PATH="/usr/local/go/bin:$PATH"
T0=$(date +%s)

echo "# target: compose project spike-supabase (LOCAL throwaway), file $REPO_ROOT/docker-compose.supabase.yml — NOT :5433, NOT :5434, NOT hosted"
DC=(docker compose -p spike-supabase --project-directory "$REPO_ROOT" -f "$REPO_ROOT/docker-compose.supabase.yml")
REST_PORT="$("${DC[@]}" port rest 3000 2>/dev/null | awk -F: '{print $NF}')"
[ -n "$REST_PORT" ] || cannot_run "spike-supabase rest service not up (run .night-crew/qa/spike-supabase/env-up.sh first)"
REST="http://127.0.0.1:$REST_PORT"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "no JWT_SECRET in docker-compose.supabase.yml"
SVC="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub hq-server -role service_role -ttl 10m)" || cannot_run "mint service_role jwt failed"
DEV="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub spike-e1-device -role authenticated -ttl 10m)" || cannot_run "mint device jwt failed"

CAMPAIGN="a0000000-0000-4000-8000-000000000001"
ID="e1000000-0000-4000-8000-000000000001"
TOKEN="spike-e1-identity-token-$(date +%s)"
HASH="$(printf %s "$TOKEN" | sha256sum | cut -c1-64)"
COLS="id,token_hash,campaign_id,expires_at,redeemed_at,redeemed_by,updated_at,_deleted"   # pull-replication.js's selection
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
# The tablet's offers pull (pull-replication.js): the column list, expires_at > window, keyset order, a limit.
device_pull() { curl -sS "$REST/codes?select=$COLS&expires_at=gt.$(printf %s "$NOW" | sed 's/:/%3A/g')&order=updated_at.asc,id.asc&limit=500" -H "Authorization: Bearer $DEV"; }
device_by_hash() { curl -sS "$REST/codes?select=id,token_hash&token_hash=eq.$HASH" -H "Authorization: Bearer $DEV"; }
cleanup() {
  local rc=$?
  set +e
  curl -sS -o /dev/null -X DELETE "$REST/codes?id=eq.$ID" -H "Authorization: Bearer $SVC"
  printf '# cleanup: DELETE codes id=%s as service_role; wall-clock %ss\n' "$ID" "$(( $(date +%s) - T0 ))"
  exit "$rc"
}
trap cleanup EXIT

leg "preflight — the fixture campaign the identity code will name"
C="$(curl -sS "$REST/campaigns?id=eq.$CAMPAIGN&select=id,name,requires_online" -H "Authorization: Bearer $SVC")"
echo "#   $C"
grep -q '"name"' <<<"$C" || cannot_run "fixture campaign $CAMPAIGN not present — supabase/seed.sql not applied"
echo "#   token (never stored): $TOKEN"
echo "#   token_hash          : $HASH"
# Reconcile a stale row from an aborted run before measuring "nothing".
curl -sS -o /dev/null -X DELETE "$REST/codes?id=eq.$ID" -H "Authorization: Bearer $SVC"

leg "(a) BEFORE — the device's filtered pull for this hash"
B="$(device_by_hash)"; echo "#   $B"
[ "$B" = "[]" ] || fail "device already sees a row for the spike hash before any write: $B"

leg "(b) UPSERT as service_role — the projection path (Prefer: resolution=merge-duplicates)"
BODY="[{\"id\":\"$ID\",\"token_hash\":\"$HASH\",\"campaign_id\":\"$CAMPAIGN\",\"expires_at\":\"2028-01-01T00:00:00Z\"}]"
CODE="$(curl -sS -o /tmp/spike-e1-up.$$ -w '%{http_code}' -X POST "$REST/codes" \
  -H "Authorization: Bearer $SVC" -H "Content-Type: application/json" \
  -H "Prefer: resolution=merge-duplicates,return=representation" -d "$BODY")"
echo "#   upsert HTTP $CODE: $(cat /tmp/spike-e1-up.$$)"; rm -f /tmp/spike-e1-up.$$
[ "$CODE" = "201" ] || [ "$CODE" = "200" ] || fail "upsert answered HTTP $CODE"

leg "(c) AFTER — the tablet's own offers-pull query as an authenticated device"
P="$(device_pull)"
N_TOTAL="$(node -e 'console.log(JSON.parse(require("fs").readFileSync(0,"utf8")).length)' <<<"$P")"
echo "#   pull returned $N_TOTAL live row(s); contains spike hash: $(grep -c "$HASH" <<<"$P" || true)"
grep -q "$HASH" <<<"$P" || fail "the device's offers pull does not contain the projected identity-code row — RLS or the window filter hides it"
A="$(device_by_hash)"; echo "#   by hash: $A"
grep -q "\"$ID\"" <<<"$A" || fail "device by-hash lookup does not return the row"

leg "(d) IDEMPOTENT re-upsert — a re-import must not mint a second row"
CODE2="$(curl -sS -o /dev/null -w '%{http_code}' -X POST "$REST/codes" \
  -H "Authorization: Bearer $SVC" -H "Content-Type: application/json" \
  -H "Prefer: resolution=merge-duplicates" -d "$BODY")"
echo "#   re-upsert HTTP $CODE2"
[ "$CODE2" = "201" ] || [ "$CODE2" = "200" ] || fail "re-upsert answered HTTP $CODE2"
N_HASH="$(node -e 'console.log(JSON.parse(require("fs").readFileSync(0,"utf8")).length)' <<<"$(device_by_hash)")"
echo "#   rows for the hash after re-upsert: $N_HASH"
[ "$N_HASH" = "1" ] || fail "expected exactly 1 row for the hash after re-upsert, got $N_HASH"

leg "(e) DELETE as service_role — reconcile"
curl -sS -o /dev/null -w '#   delete HTTP %{http_code}\n' -X DELETE "$REST/codes?id=eq.$ID" -H "Authorization: Bearer $SVC"
Z="$(device_by_hash)"; echo "#   device by hash after delete: $Z"
[ "$Z" = "[]" ] || fail "row survived the delete: $Z"

echo
echo "✅ GREEN — an identity-code row upserted by the projection path is visible to a device through the shipped"
echo "   offers-pull filter under codes_select_device; re-upsert is idempotent (1 row); deleted on exit."
