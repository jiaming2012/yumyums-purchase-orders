#!/usr/bin/env bash
# 02-rls-six-legs.sh — Card 1 gate: the RLS design discriminates through the
# REAL API surface (PostgREST + role-claim JWTs), positively AND negatively —
# the spike's proven six legs, driven against the IN-REPO migration + seed
# instead of the spike fixture. Adapted from the spike's 02-rls-per-device.sh
# (read-only source).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all six legs held:
#              (1) a device JWT (role=authenticated) reads the seeded code row
#              (2) an anonymous request cannot see it
#              (3) a device inserts a scan_attempt AS ITSELF              → accepted
#              (4) the same device inserting AS ANOTHER device_id        → refused
#              (5) the device role cannot SELECT scan_attempts (push-only)
#              (6) the own-insert really landed, the spoof did not (server-side)
#   exit 1   a leg failed — the RLS premise does not hold on the built migration.
#   exit 2   could not run.
#
# Positive and negative legs both matter: a policy that lets everything through
# passes (1)/(3) and fails (2)/(4)/(5) — the negatives make this falsifiable.
#
# Re-runnable on a warm substrate: the device identity is per-run
# (device-<runid>), so leg 6's exact counts never collide with a previous run.

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

command -v go >/dev/null 2>&1 || cannot_run "go is not on PATH — mintjwt cannot mint the device tokens"

substrate_up

REST_PORT="$("${DC[@]}" port rest 3000 | awk -F: '{print $NF}')"
case "$REST_PORT" in ''|*[!0-9]*|0) cannot_run "could not resolve PostgREST host port" ;; esac
REST="http://127.0.0.1:$REST_PORT"
echo "#   rest            : $REST"

echo
echo "── migration + seed (idempotent, server-side as supabase_admin) ──"
apply_all || cannot_run "the migration would not apply (01-structure.sh owns that premise)"

# The seeded fixtures this harness reads (contract values from supabase/seed.sql):
CODE='c0000000-0000-4000-8000-000000000001'   # LOW campaign, active, unredeemed

RUNID="$(date +%s)-$$"
DEVICE="device-$RUNID"
SPOOF="device-spoof-$RUNID"

echo
echo "── device token (HS256, secret read from the compose file, never re-typed) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub "$DEVICE" -role authenticated -ttl 30m)" \
  || cannot_run "minting the device token failed"
echo "  minted role=authenticated sub=$DEVICE"

# curl helper: prints "BODY\nHTTPCODE"
req() { curl -s -m 10 -w '\n%{http_code}' "$@"; }

echo
echo "── leg 1: the device reads the seeded code replica through PostgREST ──"
R="$(req "$REST/codes?select=id,token_hash,expires_at&id=eq.$CODE" -H "Authorization: Bearer $TOKEN")"
HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  HTTP $HTTP  $BODY"
[ "$HTTP" = 200 ] || fail "device read expected 200, got $HTTP"
echo "$BODY" | grep -q "$CODE" || fail "device read did not return the seeded code row"

echo "── leg 2: an anonymous request cannot see it ──"
R="$(req "$REST/codes?select=id")"
HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  HTTP $HTTP  $(echo "$BODY" | head -c 200)"
if [ "$HTTP" = 200 ] && echo "$BODY" | grep -q "$CODE"; then
  fail "an anonymous request can read the codes replica"
fi

echo "── leg 3: the device inserts a scan_attempt AS ITSELF ──"
OWN_ID="$(newuuid)"
R="$(req -X POST "$REST/scan_attempts" \
      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -H "Prefer: return=minimal" \
      -d "{\"id\":\"$OWN_ID\",\"code_id\":\"$CODE\",\"device_id\":\"$DEVICE\",\"scanned_at\":\"2026-09-04T18:00:00Z\",\"pos_business_date\":\"2026-09-04\"}")"
HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $HTTP  (attempt id $OWN_ID)"
[ "$HTTP" = 201 ] || fail "own-device insert expected 201, got $HTTP: $(echo "$R" | sed '$d')"

echo "── leg 4: the device inserting AS ANOTHER device_id is refused (with-check on jwt sub) ──"
SPOOF_ID="$(newuuid)"
R="$(req -X POST "$REST/scan_attempts" \
      -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -H "Prefer: return=minimal" \
      -d "{\"id\":\"$SPOOF_ID\",\"code_id\":\"$CODE\",\"device_id\":\"$SPOOF\",\"scanned_at\":\"2026-09-04T18:00:00Z\",\"pos_business_date\":\"2026-09-04\"}")"
HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $HTTP  $(echo "$R" | sed '$d' | head -c 200)"
case "$HTTP" in 4*) : ;; *) fail "spoofed-device insert expected a 4xx refusal, got $HTTP" ;; esac

echo "── leg 5: the device role cannot SELECT scan_attempts (push-only, structural) ──"
R="$(req "$REST/scan_attempts?select=id" -H "Authorization: Bearer $TOKEN")"
HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $HTTP  $(echo "$R" | sed '$d' | head -c 200)"
case "$HTTP" in 4*) : ;; *) fail "scan_attempts SELECT expected a 4xx refusal, got $HTTP" ;; esac

echo "── leg 6: the own-insert really landed, the spoof really did not (server-side) ──"
N="$(echo "select count(*) from public.scan_attempts where device_id='$DEVICE';" | psqlq)"
NB="$(echo "select count(*) from public.scan_attempts where device_id='$SPOOF';" | psqlq)"
echo "  $DEVICE rows=$N  $SPOOF rows=$NB"
[ "$N" = 1 ]  || fail "expected exactly 1 own-device attempt row, found $N"
[ "$NB" = 0 ] || fail "the refused spoof insert landed anyway ($NB rows)"

printf '\n✅ VERDICT: GREEN — RLS discriminates per device through PostgREST against the in-repo migration, positively and negatively\n'
exit 0
