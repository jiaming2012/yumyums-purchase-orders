#!/usr/bin/env bash
# 01-device-reads-one-code-by-token-hash.sh — spike for B-468 (Activity I card I2,
# scan-time verify): an ONLINE phone can look up the token_hash it already computed
# on the PostgREST connection it already holds. Proves, against the LOCAL
# spike-supabase substrate with a DEVICE (`authenticated`) JWT, read-only:
#   (a) GET /codes?token_hash=eq.<seeded>&select=id,campaign_id,expires_at,redeemed_at,redeemed_by
#       → 200, exactly 1 row, campaign_id non-null
#   (b) same GET for a random 64-hex hash → 200 and `[]` (unknown = empty, not an error)
#   (c) the seeded REDEEMED row (…-0004) is readable with redeemed_at set, so
#       "already used" is visible at scan time (skipped, not red, if the seed has none)
#   (d) every call completes inside the 3.5 s connectivity-probe budget
#   (e) observation only: HEAD + Prefer: count=exact + limit=1 — what PostgREST answers
# Seed rows come from supabase/seed.sql (fixed UUIDs c0000000-…-0001…0005).
# B-460: another spike may reset_bare+apply_all the shared substrate for a few seconds;
# if leg (a) answers 0 rows we wait 15 s and retry ONCE before calling it red.
# 🛑 exit 0 = proven; exit 1 = red; exit 2 = could not run (substrate down — we never start it).
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"
SEED="$REPO_ROOT/supabase/seed.sql"
TMP="$(mktemp -d)"; trap 'rm -rf "$TMP"' EXIT
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
BUDGET_S=3.5
SELECT='id,campaign_id,expires_at,redeemed_at,redeemed_by'

# --- substrate: find PostgREST; never start it, never touch :5433/:5434 ---------------
DC=(docker compose -p spike-supabase --project-directory "$REPO_ROOT" -f "$REPO_ROOT/docker-compose.supabase.yml")
REST_PORT="$("${DC[@]}" port rest 3000 2>/dev/null | awk -F: '{print $NF}')"
[ -n "$REST_PORT" ] || cannot_run "spike-supabase rest not up (this spike does not start it)"
BASE="http://127.0.0.1:$REST_PORT"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "JWT_SECRET not found in docker-compose.supabase.yml"
DEV="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 10m)" || cannot_run "mint device failed"
echo "# PostgREST at $BASE, device JWT minted (role authenticated, sub device-a)"

# --- seed: pull the token_hash values out of supabase/seed.sql -------------------------
[ -f "$SEED" ] || cannot_run "seed.sql missing at $SEED"
# The codes insert lists each row as (id, token_hash, campaign_id, expires_at, redeemed_at, redeemed_by);
# node parses the tuple list so we assert against what the seed actually says, not a copy.
node -e '
const s=require("fs").readFileSync(process.argv[1],"utf8");
const m=s.match(/insert into public\.codes[^;]*?values([\s\S]*?)on conflict/i); if(!m){console.error("codes insert not found");process.exit(3)}
const rows=[]; const re=/\(\s*\x27([^\x27]+)\x27\s*,\s*\x27([0-9a-f]{64})\x27\s*,\s*\x27([^\x27]+)\x27\s*,\s*\x27([^\x27]+)\x27\s*,\s*(null|\x27[^\x27]*\x27)\s*,\s*(null|\x27[^\x27]*\x27)\s*\)/gi;
let x; while((x=re.exec(m[1]))) rows.push({id:x[1],token_hash:x[2],campaign_id:x[3],expires_at:x[4],redeemed_at:x[5]==="null"?null:x[5].slice(1,-1),redeemed_by:x[6]==="null"?null:x[6].slice(1,-1)});
require("fs").writeFileSync(process.argv[2],JSON.stringify(rows));
console.log("# seed.sql: "+rows.length+" codes rows; redeemed: "+rows.filter(r=>r.redeemed_at).map(r=>r.id).join(",")||"none");
' "$SEED" "$TMP/seed.json" || cannot_run "could not parse codes rows from seed.sql"
SEED_UNRED_HASH="$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1]));const u=r.find(x=>!x.redeemed_at);process.stdout.write(u?u.token_hash:"")' "$TMP/seed.json")"
SEED_UNRED_ID="$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1]));const u=r.find(x=>!x.redeemed_at);process.stdout.write(u?u.id:"")' "$TMP/seed.json")"
SEED_RED_HASH="$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1]));const u=r.find(x=>x.redeemed_at);process.stdout.write(u?u.token_hash:"")' "$TMP/seed.json")"
SEED_RED_ID="$(node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1]));const u=r.find(x=>x.redeemed_at);process.stdout.write(u?u.id:"")' "$TMP/seed.json")"
[ -n "$SEED_UNRED_HASH" ] || cannot_run "seed.sql has no unredeemed codes row to read"

# --- helper: GET one hash, record http code + body + time ------------------------------
# get <hash> <outfile>  → prints "HTTP TIME" ; body in outfile
get() {
  curl -sS -o "$2" -w '%{http_code} %{time_total}' --max-time 10 \
    "$BASE/codes?token_hash=eq.$1&select=$SELECT" \
    -H "Authorization: Bearer $DEV" -H 'Accept: application/json'
}
rows() { node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"));process.stdout.write(Array.isArray(r)?String(r.length):"-1")' "$1"; }
under_budget() { awk -v t="$1" -v b="$BUDGET_S" 'BEGIN{exit !(t+0 < b+0)}'; }
TIMES=()

# --- (a) seeded unredeemed hash → 200, 1 row, campaign_id non-null -------------------
RETRIED=no
read -r A_CODE A_TIME <<<"$(get "$SEED_UNRED_HASH" "$TMP/a.json")"
A_ROWS="$(rows "$TMP/a.json")"
if [ "$A_CODE" = "200" ] && [ "$A_ROWS" = "0" ]; then
  echo "# (a) answered 0 rows — B-460 shared substrate may be mid-reset by another spike; waiting 15 s and retrying once"
  sleep 15; RETRIED=yes
  read -r A_CODE A_TIME <<<"$(get "$SEED_UNRED_HASH" "$TMP/a.json")"
  A_ROWS="$(rows "$TMP/a.json")"
fi
echo "# (a) GET /codes?token_hash=eq.$SEED_UNRED_HASH&select=$SELECT → HTTP $A_CODE rows=$A_ROWS time=${A_TIME}s retried=$RETRIED"
cat "$TMP/a.json"; echo
[ "$A_CODE" = "200" ] || fail "(a) seeded hash answered HTTP $A_CODE"
[ "$A_ROWS" = "1" ] || fail "(a) seeded hash answered $A_ROWS rows, expected exactly 1 (retried=$RETRIED)"
node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"))[0];const need=["id","campaign_id","expires_at","redeemed_at","redeemed_by"];for(const k of need)if(!(k in r)){console.log("missing column "+k);process.exit(1)};if(!r.campaign_id){console.log("campaign_id null");process.exit(1)};if(r.id!==process.argv[2]){console.log("id "+r.id+" != seed "+process.argv[2]);process.exit(1)};console.log("# (a) row id="+r.id+" campaign_id="+r.campaign_id+" redeemed_at="+r.redeemed_at+" — all 5 columns present")' "$TMP/a.json" "$SEED_UNRED_ID" || fail "(a) row shape wrong"
TIMES+=("a:$A_TIME")

# --- (b) random 64-hex hash → 200 and [] ----------------------------------------------
RAND_HASH="$(head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n')"
read -r B_CODE B_TIME <<<"$(get "$RAND_HASH" "$TMP/b.json")"
B_ROWS="$(rows "$TMP/b.json")"
echo "# (b) GET /codes?token_hash=eq.$RAND_HASH → HTTP $B_CODE rows=$B_ROWS body=$(cat "$TMP/b.json") time=${B_TIME}s"
[ "$B_CODE" = "200" ] || fail "(b) unknown hash answered HTTP $B_CODE (expected 200 + [])"
[ "$B_ROWS" = "0" ] || fail "(b) unknown hash answered $B_ROWS rows, expected []"
TIMES+=("b:$B_TIME")

# --- (c) seeded REDEEMED hash → 200, 1 row, redeemed_at set ---------------------------
if [ -n "$SEED_RED_HASH" ]; then
  read -r C_CODE C_TIME <<<"$(get "$SEED_RED_HASH" "$TMP/c.json")"
  C_ROWS="$(rows "$TMP/c.json")"
  echo "# (c) GET /codes?token_hash=eq.$SEED_RED_HASH (seed $SEED_RED_ID, redeemed) → HTTP $C_CODE rows=$C_ROWS time=${C_TIME}s"
  cat "$TMP/c.json"; echo
  [ "$C_CODE" = "200" ] || fail "(c) redeemed hash answered HTTP $C_CODE"
  [ "$C_ROWS" = "1" ] || fail "(c) redeemed hash answered $C_ROWS rows, expected 1 (RLS hiding redeemed rows?)"
  node -e 'const r=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"))[0];if(!r.redeemed_at){console.log("redeemed_at is null — already-used not visible");process.exit(1)};console.log("# (c) redeemed_at="+r.redeemed_at+" redeemed_by="+r.redeemed_by+" — already-used IS visible at scan time")' "$TMP/c.json" || fail "(c) redeemed row readable but redeemed_at not set"
  TIMES+=("c:$C_TIME")
else
  echo "# (c) SKIPPED — seed.sql holds no redeemed codes row (not a failure)"
fi

# --- (d) every call inside the 3.5 s probe budget ------------------------------------
for t in "${TIMES[@]}"; do
  leg="${t%%:*}"; sec="${t#*:}"
  under_budget "$sec" || fail "(d) leg $leg took ${sec}s, over the ${BUDGET_S}s probe budget"
done
echo "# (d) times: ${TIMES[*]} — all < ${BUDGET_S}s"

# --- (e) observation only: count without body (HEAD + Prefer: count=exact, limit=1) ---
E_HDR="$(curl -sS -I --max-time 10 "$BASE/codes?token_hash=eq.$SEED_UNRED_HASH&select=id&limit=1" \
  -H "Authorization: Bearer $DEV" -H 'Prefer: count=exact' 2>&1 | tr -d '\r')" || true
E_CODE="$(printf '%s\n' "$E_HDR" | awk 'NR==1{print $2}')"
E_RANGE="$(printf '%s\n' "$E_HDR" | grep -i '^content-range:' | head -1)"
echo "# (e) HEAD /codes?token_hash=eq.<seeded>&select=id&limit=1 + Prefer: count=exact → HTTP ${E_CODE:-?}; ${E_RANGE:-no Content-Range header}"
E2_HDR="$(curl -sS -I --max-time 10 "$BASE/codes?token_hash=eq.$RAND_HASH&select=id&limit=1" \
  -H "Authorization: Bearer $DEV" -H 'Prefer: count=exact' 2>&1 | tr -d '\r')" || true
echo "# (e) same HEAD for the random hash → HTTP $(printf '%s\n' "$E2_HDR" | awk 'NR==1{print $2}'); $(printf '%s\n' "$E2_HDR" | grep -i '^content-range:' | head -1 || echo 'no Content-Range header')"

echo "✅ GREEN — the device role reads exactly one codes row by token_hash (with campaign_id, expires_at, redeemed_at, redeemed_by), an unknown hash answers 200 [], a redeemed row is readable, all inside the 3.5 s probe budget; B-468 scan-time verify is one PostgREST GET on the connection the phone already holds"
