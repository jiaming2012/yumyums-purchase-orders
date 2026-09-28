#!/usr/bin/env bash
# 01-coordinates-through-the-door.sh — spike: the shipped pull replicas work
# through the HQ door on the session cookie alone, and the client bearer is
# inert there.
#
# The provisioning card writes {restUrl, bearer} into SYNC_KEY and startSync
# runs. The premise nobody has RUN: with restUrl = <origin>/sync/rest, the
# shipped buildPullUrl `${restUrl}/${table}` composes with the proxy's prefix
# strip, the proxy substitutes a per-request token minted for the SESSION user
# (discarding whatever bearer the client holds), RLS answers rows (not a
# silent 200-with-nothing), and the policy source reaches attached/size>0 —
# done_when clause 2's premise, nothing stubbed.
#
# Auth matrix, enumerated as a set (B-216):
#   (a) valid cookie + garbage bearer      → 200 AND rows (substitution works)
#   (b) no cookie + VALID minted bearer    → 401 (the door never honors a client bearer)
#   (c) no cookie + no bearer              → 401
#   (d) control: garbage bearer DIRECT to PostgREST → non-200 (PostgREST does
#       validate, so (a)'s 200 can only be the proxy's substitution)
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the door works and the matrix agreed.
#   exit 1  a measurement disagreed.   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh (spike-supabase compose only) +
# lib-hq.sh (spike-owned hq_test_spike_prov on the TEST-ONLY :5434).

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
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — the gate of earlier cards, not this spike's premise"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in backend/internal/sync/proxy.go backend/internal/sync/jwtbridge_handler.go \
         marketing/sync/replicas.js marketing/sync/pull-replication.js marketing/scan-page.js; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this spike tests the SHIPPED artifact, not a copy"
  printf '#   %-46s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── HQ door up (real server, real middleware, spike-owned test DB) ──"
resolve_substrate_rest
hq_db_reset
hq_up
hq_login
echo "#   session cookie   : hq_session=${COOKIE:0:8}… (the credential under test)"

echo
echo "── a VALID bridge token for matrix row (b) (minted by the shipped endpoint) ──"
VALID_TOKEN="$(curl -s -X POST "$HQ_ORIGIN/api/v1/sync/token" -H "Cookie: hq_session=$COOKIE" \
  | node -e 'let d="";process.stdin.on("data",(c)=>d+=c).on("end",()=>{const j=JSON.parse(d);if(!j.token)process.exit(1);process.stdout.write(j.token)})')" \
  || fail "POST /api/v1/sync/token with the cookie did not answer a token — spike 02's territory, but row (b) needs one"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── the matrix + all three shipped replicas, through the door ──"
node "$SCRIPT_DIR/js/door-pull.mjs" "$HQ_ORIGIN" "$COOKIE" "$VALID_TOKEN" "$REST_DIRECT" \
  || fail "a measurement disagreed — see the node log above"

printf '\n✅ VERDICT: GREEN — the session cookie is the credential; the client bearer is inert at the door; all three shipped replicas deliver rows and the policy source attaches for real.\n'
