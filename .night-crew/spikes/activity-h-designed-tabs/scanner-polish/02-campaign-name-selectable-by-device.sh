#!/usr/bin/env bash
# 02-campaign-name-selectable-by-device.sh — spike for B-447: adding `name` to
# the campaigns pull selection is a column the DEVICE role may read — the
# Activity A RLS grants SELECT on public.campaigns to authenticated with no
# column list, so `select=id,name,requires_online,updated_at` answers 200 with
# `name` populated, from the LOCAL substrate. (If the grant were column-scoped
# this would be a schema card, not a pull-selection change.)
# 🛑 exit 0 = 200 and every row has a non-null name; exit 1 = not; exit 2 = could not run.
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
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
DEV="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 10m)" || cannot_run "mint device failed"
C="$(curl -sS -o /tmp/spike-h6.$$ -w '%{http_code}' "http://127.0.0.1:$REST_PORT/campaigns?select=id,name,requires_online,updated_at&order=updated_at.asc" -H "Authorization: Bearer $DEV")"
echo "# device GET campaigns(select=id,name,requires_online,updated_at) → HTTP $C"; head -c 300 /tmp/spike-h6.$$; echo
[ "$C" = "200" ] || { rm -f /tmp/spike-h6.$$; fail "device read answered HTTP $C"; }
node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8")); if(!Array.isArray(r)||r.length===0){console.log("no rows");process.exit(1)}; if(r.some(x=>!x.name)){console.log("a row has null name");process.exit(1)}; console.log("# rows="+r.length+" all with name")' /tmp/spike-h6.$$ || { rm -f /tmp/spike-h6.$$; fail "name not populated on every row"; }
rm -f /tmp/spike-h6.$$
echo "✅ GREEN — the device role can select campaigns.name; B-447 is a pull-selection + UI change, not a schema card"
