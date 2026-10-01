#!/usr/bin/env bash
# 02-projection-upsert-over-postgrest.sh — spike: the HQ Go handler can write
# the four tablet columns of a campaign into Supabase `campaigns` as an UPSERT
# over PostgREST with a service-role JWT (Prefer: resolution=merge-duplicates),
# and read it back — the projection path decision 187 rests on. Runs against
# the committed LOCAL spike-supabase substrate only (reconcile mode); the row
# it writes is the TEST fixture campaign a0…0001 and its name is restored.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  upsert 201/200 + read-back shows the new name, then restored.
#   exit 1  PostgREST refused or read-back disagrees.   exit 2  could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
echo "# target: compose project spike-supabase (LOCAL throwaway), file $REPO_ROOT/docker-compose.supabase.yml — NOT :5433, NOT :5434, NOT hosted"
DC=(docker compose -p spike-supabase --project-directory "$REPO_ROOT" -f "$REPO_ROOT/docker-compose.supabase.yml")
REST_PORT="$("${DC[@]}" port rest 3000 2>/dev/null | awk -F: '{print $NF}')"
[ -n "$REST_PORT" ] || cannot_run "spike-supabase rest service not up (run .night-crew/qa/spike-supabase/env-up.sh first)"
REST="http://127.0.0.1:$REST_PORT"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
SVC="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub hq-server -role service_role -ttl 10m)" || cannot_run "mint service_role jwt failed"
ID="a0000000-0000-4000-8000-000000000001"
ORIG="$(curl -sS "$REST/campaigns?id=eq.$ID&select=name,face_value,requires_online" -H "Authorization: Bearer $SVC")"
echo "# before: $ORIG"
echo "$ORIG" | grep -q '"name"' || cannot_run "fixture campaign $ID not present — seed.sql not applied"
NEW="SPIKE H1 projection $(date +%s)"
CODE="$(curl -sS -o /tmp/spike-h1-up.$$ -w '%{http_code}' -X POST "$REST/campaigns" \
  -H "Authorization: Bearer $SVC" -H "Content-Type: application/json" \
  -H "Prefer: resolution=merge-duplicates,return=representation" \
  -d "[{\"id\":\"$ID\",\"name\":\"$NEW\",\"face_value\":2.00,\"requires_online\":false}]")"
echo "# upsert HTTP $CODE: $(cat /tmp/spike-h1-up.$$)"; rm -f /tmp/spike-h1-up.$$
[ "$CODE" = "201" ] || [ "$CODE" = "200" ] || fail "upsert answered HTTP $CODE"
BACK="$(curl -sS "$REST/campaigns?id=eq.$ID&select=name" -H "Authorization: Bearer $SVC")"
echo "# after:  $BACK"
echo "$BACK" | grep -qF "$NEW" || fail "read-back does not show the projected name"
# restore (reconcile, never destroy)
ORIGNAME="$(printf %s "$ORIG" | sed -n 's/.*"name":"\([^"]*\)".*/\1/p')"
curl -sS -o /dev/null -X PATCH "$REST/campaigns?id=eq.$ID" -H "Authorization: Bearer $SVC" -H "Content-Type: application/json" -d "{\"name\":\"$ORIGNAME\"}"
echo "# restored name: $ORIGNAME"
echo "✅ GREEN — service-role upsert over PostgREST lands and reads back"
