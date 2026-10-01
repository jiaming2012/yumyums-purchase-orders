#!/usr/bin/env bash
# 03-scan-attempts-service-read.sh — spike: the H3 mirror can READ Supabase
# scan_attempts with the service role (RLS grants devices INSERT only and no
# SELECT — proven at Activity A; the server is the only reader) using a keyset
# order (scanned_at,id) PostgREST accepts. Against the LOCAL substrate only.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  service-role GET → 200 + JSON array (keyset order accepted) AND an
#           authenticated device GET → 401/403 (the RLS asymmetry still holds).
#   exit 1  either half disagrees.   exit 2  could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
DC=(docker compose -p spike-supabase --project-directory "$REPO_ROOT" -f "$REPO_ROOT/docker-compose.supabase.yml")
REST_PORT="$("${DC[@]}" port rest 3000 2>/dev/null | awk -F: '{print $NF}')"
[ -n "$REST_PORT" ] || cannot_run "spike-supabase rest not up (env-up.sh first)"
REST="http://127.0.0.1:$REST_PORT"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
SVC="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub hq-server -role service_role -ttl 10m)" || cannot_run "mint service_role failed"
DEV="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 10m)" || cannot_run "mint device failed"
Q="$REST/scan_attempts?select=id,scanned_at,status,offline_override,unverified_code,pos_order_number,pos_business_date&order=scanned_at.asc,id.asc&limit=50"
C1="$(curl -sS -o /tmp/spike-h3-svc.$$ -w '%{http_code}' "$Q" -H "Authorization: Bearer $SVC")"
echo "# service_role GET → HTTP $C1, body starts: $(head -c 120 /tmp/spike-h3-svc.$$)"
[ "$C1" = "200" ] || fail "service_role read answered HTTP $C1"
head -c 1 /tmp/spike-h3-svc.$$ | grep -q '\[' || fail "service_role body is not a JSON array"
C2="$(curl -sS -o /dev/null -w '%{http_code}' "$Q" -H "Authorization: Bearer $DEV")"
echo "# authenticated device GET → HTTP $C2 (expected 401/403: devices are push-only)"
rm -f /tmp/spike-h3-svc.$$
case "$C2" in 401|403) ;; *) fail "a device JWT could READ scan_attempts (HTTP $C2) — the push-only RLS asymmetry no longer holds";; esac
echo "✅ GREEN — server reads with the service role in keyset order; devices still cannot"
