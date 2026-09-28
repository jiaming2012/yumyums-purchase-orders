#!/usr/bin/env bash
# 04-recovery-edge-for-b439.sh — spike: the B-439 fix has a real edge to key on.
#
# done_when clause 4: after a pull failure + full recovery, unresolved()
# returns to false. The shipped latch clears lastError exactly once, at
# first-ready (replicas.js), and the sibling spike measured only the PRE-ready
# window. Never run: after ready, a 503 blip, then recovery — which SHIPPED
# observable marks the successful post-ready pull cycle so the latch can clear?
#
# Candidates, enumerated as a set (B-216): error$ silence (not an event),
# active$ transitions, remoteEvents$ emissions, awaitInSync() re-resolution,
# clock.captures (increments on every HTTP-200 pull — the §5.1 seam scan-page
# already injects). Each measured in BOTH phases (fires on success; does not
# fire / is distinguishable while erroring) and in BOTH recovery shapes:
# recovery-with-docs and recovery-EMPTY (zero new rows — the worst case).
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  at least one shipped signal marks the successful post-ready cycle
#           in BOTH shapes.
#   exit 1  no signal survives the recovery-empty shape — the fix must move
#           to the pull-handler seam (a different card shape, priced).
#   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — direct PostgREST with a
# mintjwt device token (the sibling harness pattern); no HQ server needed for
# a client-observable question.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── device JWT (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── the post-ready error → recovery edge, measured on the shipped handle ──"
DB_CID="$DB_CID" node "$SCRIPT_DIR/js/recovery-edge.mjs" "$TOKEN" \
  || fail "no shipped signal survives the recovery-empty shape — see the node log; the card's fix moves to the pull-handler seam"

printf '\n✅ VERDICT: GREEN — the recovery edge is observable on shipped surface in both shapes; the B-439 clear has something real to key on.\n'
