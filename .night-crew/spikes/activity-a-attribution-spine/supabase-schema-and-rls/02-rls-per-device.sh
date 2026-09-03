#!/usr/bin/env bash
# 02-rls-per-device.sh — spike: the §7.2 RLS design discriminates through the real
# API surface (PostgREST + role-claim JWTs), positively AND negatively.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   all six legs held:
#              (1) a device JWT (role=authenticated) reads the seeded code row
#              (2) an anonymous request cannot see it
#              (3) a device inserts a scan_attempt AS ITSELF              → accepted
#              (4) the same device inserting AS ANOTHER device_id          → refused
#              (5) the device role cannot SELECT scan_attempts (push-only)
#              (6) the own-insert really landed (verified server-side)
#   exit 1   a leg failed — the RLS premise is not proven.
#   exit 2   could not run.
#
# Positive and negative legs both matter: a policy that lets everything through
# passes (1)/(3) and fails (2)/(4)/(5) — the negatives are what make this
# falsifiable rather than decorative.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

# Go for mintjwt — non-interactive shells do not carry it (run-mechanics rule).
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH
command -v go >/dev/null 2>&1 || cannot_run "go is not on PATH — mintjwt cannot mint the device tokens"

echo "# target coordinates (read-only statement before any write):"
echo "#   compose project : spike-supabase (throwaway). NOT :5433, NOT :5434, no hosted project."

echo
echo "── substrate up ──"
"$QA/env-up.sh" || cannot_run "env-up.sh did not reach GREEN"

ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
DC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")
DB_CID="$("${DC[@]}" ps -q db)";  [ -n "$DB_CID" ] || cannot_run "no db container"
REST_PORT="$("${DC[@]}" port rest 3000 | awk -F: '{print $NF}')"
case "$REST_PORT" in ''|*[!0-9]*|0) cannot_run "could not resolve PostgREST host port" ;; esac
REST="http://127.0.0.1:$REST_PORT"
echo "#   db container=$DB_CID rest=$REST"

psqlq()  { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA; }

echo
echo "── schema + seed (server-side, as supabase_admin) ──"
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
  < "$SCRIPT_DIR/sql/qr-schema.sql" || cannot_run "qr-schema.sql would not apply (spike 01 owns that premise)"

CAMPAIGN='11111111-1111-1111-1111-111111111111'
CODE='22222222-2222-2222-2222-222222222222'
psqlq <<SQL || cannot_run "seeding failed"
insert into public.campaigns (id, name, face_value) values ('$CAMPAIGN', 'spike-rls', 2.00);
insert into public.codes (id, token_hash, campaign_id, expires_at)
  values ('$CODE', 'spike-rls-hash-1', '$CAMPAIGN', now() + interval '1 day');
SQL
echo "  seeded campaign=$CAMPAIGN code=$CODE"

echo
echo "── device tokens (HS256, secret read from the compose file, never re-typed) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" \
  || cannot_run "minting device-a token failed"
echo "  minted role=authenticated sub=device-a"

# curl helper: prints "HTTPCODE\nBODY"
req() { curl -s -m 10 -w '\n%{http_code}' "$@"; }

echo
echo "── leg 1: device-a reads the code replica through PostgREST ──"
R="$(req "$REST/codes?select=id,token_hash,expires_at" -H "Authorization: Bearer $TOKEN_A")"
CODE_HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  HTTP $CODE_HTTP  $BODY"
[ "$CODE_HTTP" = 200 ] || fail "device read expected 200, got $CODE_HTTP"
echo "$BODY" | grep -q "$CODE" || fail "device read did not return the seeded code row"

echo "── leg 2: anonymous request cannot see it ──"
R="$(req "$REST/codes?select=id")"
CODE_HTTP="$(echo "$R" | tail -1)"; BODY="$(echo "$R" | sed '$d')"
echo "  HTTP $CODE_HTTP  $(echo "$BODY" | head -c 200)"
if [ "$CODE_HTTP" = 200 ] && echo "$BODY" | grep -q "$CODE"; then
  fail "an anonymous request can read the codes replica"
fi

echo "── leg 3: device-a inserts a scan_attempt AS ITSELF ──"
R="$(req -X POST "$REST/scan_attempts" \
      -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" -H "Prefer: return=minimal" \
      -d "{\"id\":\"33333333-3333-3333-3333-333333333333\",\"code_id\":\"$CODE\",\"device_id\":\"device-a\",\"scanned_at\":\"2026-09-03T18:00:00Z\",\"pos_business_date\":\"2026-09-03\"}")"
CODE_HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $CODE_HTTP"
[ "$CODE_HTTP" = 201 ] || fail "own-device insert expected 201, got $CODE_HTTP: $(echo "$R" | sed '$d')"

echo "── leg 4: device-a inserting AS device-b is refused (with-check on jwt sub) ──"
R="$(req -X POST "$REST/scan_attempts" \
      -H "Authorization: Bearer $TOKEN_A" -H "Content-Type: application/json" -H "Prefer: return=minimal" \
      -d "{\"id\":\"44444444-4444-4444-4444-444444444444\",\"code_id\":\"$CODE\",\"device_id\":\"device-b\",\"scanned_at\":\"2026-09-03T18:00:00Z\",\"pos_business_date\":\"2026-09-03\"}")"
CODE_HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $CODE_HTTP  $(echo "$R" | sed '$d' | head -c 200)"
case "$CODE_HTTP" in 4*) : ;; *) fail "spoofed-device insert expected a 4xx refusal, got $CODE_HTTP" ;; esac

echo "── leg 5: the device role cannot SELECT scan_attempts (push-only, §4) ──"
R="$(req "$REST/scan_attempts?select=id" -H "Authorization: Bearer $TOKEN_A")"
CODE_HTTP="$(echo "$R" | tail -1)"
echo "  HTTP $CODE_HTTP  $(echo "$R" | sed '$d' | head -c 200)"
case "$CODE_HTTP" in 4*) : ;; *) fail "scan_attempts SELECT expected a 4xx refusal, got $CODE_HTTP" ;; esac

echo "── leg 6: the own-insert really landed (server-side count) ──"
N="$(echo "select count(*) from public.scan_attempts where device_id='device-a';" | psqlq)"
NB="$(echo "select count(*) from public.scan_attempts where device_id='device-b';" | psqlq)"
echo "  device-a rows=$N  device-b rows=$NB"
[ "$N" = 1 ]  || fail "expected exactly 1 device-a attempt row, found $N"
[ "$NB" = 0 ] || fail "the refused spoof insert landed anyway ($NB rows)"

printf '\n✅ VERDICT: GREEN — RLS discriminates per device through PostgREST, positively and negatively\n'
exit 0
