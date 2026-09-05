#!/usr/bin/env bash
# 02-refusal-arms-on-real-data.sh — spike: the card's done_when, first half.
#
# Card 6 built the §8 refusal behind an INJECTABLE policy seam
# (submit-flow.js setCampaignPolicy) and pinned it with a seam-injected e2e:
# conformance seq 10 hands the machine `requiresOnline: true` as a literal. That
# proves the guard, not the arming. This spike closes the gap the card exists to
# close — the policy comes from a REAL row that a REAL replica pulled off the
# built schema, and the machine under test is the shipped
# marketing/submit-machine.js running on the shipped lib/xstate.umd.min.js (the
# same wiring tests/machine/run-conformance.mjs gates).
#
# Four runs — {HIGH code, LOW code} × {policy source = replica, policy source =
# none}:
#   replica + HIGH (campaign …0002, requires_online=true)  → override REFUSED
#   replica + LOW  (campaign …0001, requires_online=false) → override OFFERED
#   none    + HIGH → override OFFERED   ← today's shipped behaviour
#   none    + LOW  → override OFFERED
#
# The bottom two are the leg that makes this a proof rather than a restatement:
# they DEMONSTRATE "the refusal is unreachable today" instead of asserting it.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the refusal arms on the real flag and is provably dead without it.
#   exit 1  a run disagreed.   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase compose project only; never :5433, never :5434, no hosted
# project.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this spike's premise"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in marketing/submit-machine.js lib/xstate.umd.min.js marketing/sync/pull-replication.js; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this spike tests the SHIPPED artifact, not a copy"
  printf '#   %-42s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── device JWT (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── four runs of the shipped machine, policy read from the real replica ──"
node "$SCRIPT_DIR/js/refusal-arms.mjs" "$TOKEN" \
  || fail "a run disagreed — see the node log above"

printf '\n✅ VERDICT: GREEN — the §8 refusal arms on the replicated flag, and is provably dead without it.\n'
