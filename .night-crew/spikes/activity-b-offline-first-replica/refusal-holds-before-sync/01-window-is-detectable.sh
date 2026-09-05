#!/usr/bin/env bash
# 01-window-is-detectable.sh — spike: the pre-sync window is observable.
#
# Every candidate fix shape for B-432 (gate override on campaigns-replica
# readiness / fail closed for unresolved campaigns) presumes the client can
# tell, at policy-lookup time, three states apart on the SHIPPED
# startCampaignsReplica handle:
#   ready      — initial replication complete, the policy Map is truth
#   in-flight  — started, nothing delivered yet, no error
#   erroring   — the pull handler is throwing (HTTP 4xx/5xx — the B-432 window)
#
# scan-page.js already consumes awaitInitialReplication() (display-only
# SCAN_STATE.synced), but nobody has ever RUN the erroring leg: does
# awaitInitialReplication() hang while error$ emits? does a late subscriber
# see a replay? does recovery resolve the SAME handle without a restart? and
# is the shipped createCampaignPolicySource Map actually populated at the
# moment readiness fires (subscription race)?
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the three states are distinguishable on the shipped surface.
#   exit 1  a measurement disagreed.   exit 2  could not run.
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
for f in marketing/sync/replicas.js marketing/sync/pull-replication.js vendor/rxdb.bundle.js; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this spike tests the SHIPPED artifact, not a copy"
  printf '#   %-42s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── vendored-bundle surface (the browser must carry the same signals) ──"
VENDOR_MISSING=0
for name in 'error\$' 'awaitInitialReplication' 'awaitInSync' 'active\$'; do
  if grep -q "$name" "$REPO_ROOT/vendor/rxdb.bundle.js"; then
    printf '#   vendor/rxdb.bundle.js carries %s\n' "$name"
  else
    printf '#   vendor/rxdb.bundle.js MISSING %s\n' "$name"
    VENDOR_MISSING=1
  fi
done
[ "$VENDOR_MISSING" = 0 ] || fail "the vendored bundle lacks a signal the fix would gate on — the card owes a vendor-surface widening BEFORE the seam work"

echo
echo "── device JWT (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "── the erroring → recovered window, measured on the shipped handle ──"
node "$SCRIPT_DIR/js/window-signals.mjs" "$TOKEN" \
  || fail "a measurement disagreed — see the node log above"

printf '\n✅ VERDICT: GREEN — ready / in-flight / erroring are distinguishable on the shipped replica handle, and recovery needs no restart.\n'
