#!/usr/bin/env bash
# marketing/sync/harness/recovery-clear-run.sh — the B-439 validation gate for
# card sync-coordinates-provisioning (run 20260907; B-345 precedent: the
# runnable script IS the verdict).
#
# THIS GATE IS THE OWED SPIKE-04 RE-RUN (goal ledger Comebacks): spike 04
# (`recovery-edge-for-b439`) re-executed against the SHIPPED clear — the
# refusal-run.sh precedent, wrapper-free against the shipped code. The latch
# must go `unresolved()=false` in BOTH recovery shapes (with-docs and
# recovery-EMPTY) where the spike measured it stuck `true`, and must STILL
# latch during both error phases. See recovery-clear-harness.mjs.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the shipped clear takes the latch back to false in both recovery
#           shapes AND the latch still latches during both error phases.
#   exit 1  a phase failed (stuck latch = B-439 unrepaired; a non-latching
#           error phase = the clear fails OPEN).
#   exit 2  could not run.
#
# RED evidence for this gate: the pre-fix run of this same script (exit 1 at
# phase C1 — .night-crew/runs/2026-09-07-autonomous/c1-red-recovery-clear.log)
# and the unmodified spike script's post-fix exit 1 at ITS ghost-check
# ("…then B-439 is already fixed…" — c1-spike04-rerun-postfix.log).
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in marketing/sync/replicas.js marketing/sync/pull-replication.js \
         marketing/sync/clock.js; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this gate tests the SHIPPED artifact, not a copy"
  printf '#   %-40s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── device JWT (throwaway secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here; the link is gitignored).
[ -e "$SCRIPT_DIR/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/node_modules"

echo
echo "── the post-ready error → recovery clear, on the SHIPPED surface ──"
set +e
DB_CID="$DB_CID" node "$SCRIPT_DIR/recovery-clear-harness.mjs" "$TOKEN"
NODE_EXIT=$?
set -e

if [ "$NODE_EXIT" -eq 0 ]; then
  printf '\n✅ VERDICT: GREEN — B-439 is closed on the SHIPPED source: the latch clears on the successful-pull edge in BOTH recovery shapes and still latches during every error phase.\n'
else
  printf '\n🛑 VERDICT: RED — a phase failed (node exit %s); see the log above.\n' "$NODE_EXIT"
fi
exit "$NODE_EXIT"
