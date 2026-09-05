#!/usr/bin/env bash
# 02-refusal-holds-during-window.sh — spike: the card's done_when, first half,
# at the policy seam.
#
# B-432: with codes + offers replicated and the campaigns replica NOT yet
# delivered (first sync, or a campaigns pull that 4xx/5xx's while codes
# succeeds), the shipped policyFor coerces null → false and a known,
# entitlement-bearing requires_online=true code is offline-overridable.
#
# Five runs of the SHIPPED machine (mode 'throw', canOverride: true), policy
# fed from real replicas over the built schema, campaigns behind a 503 gate:
#   prototype + window + HIGH known code  → override REFUSED   (the fix)
#   prototype + window + unknown code     → override OFFERED, F2 warning (166 survives)
#   shipped   + window + HIGH known code  → override OFFERED   ← B-432, demonstrated
#   prototype + ready  + HIGH             → override REFUSED   (normal §8)
#   prototype + ready  + LOW              → override OFFERED   (no over-refusal)
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the refusal holds through the window and nothing else moved.
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
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A/B's gate, not this spike's premise"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in marketing/submit-machine.js lib/xstate.umd.min.js marketing/sync/replicas.js marketing/sync/pull-replication.js; do
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
echo "── five runs of the shipped machine across the window ──"
node "$SCRIPT_DIR/js/refusal-holds.mjs" "$TOKEN" \
  || fail "a run disagreed — see the node log above"

printf '\n✅ VERDICT: GREEN — fail-closed at the seam holds the refusal through the window, F2 survives, and readiness lifts it.\n'
