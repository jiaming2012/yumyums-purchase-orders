#!/usr/bin/env bash
# 02-required-campaign-id-needs-a-replica-migration.sh — spike for card I1's decision-191
# rider "add campaign_id to MARKETING_REPLICA_SCHEMA.required": is that a one-line edit or
# a replica-schema migration? Against the vendored RxDB (the harness's own imports), one
# node process, same storage object across opens:
#   (1) open at the SHIPPED schema (version 0), insert one row with campaign_id, close (not remove)
#   (1b) reopen at the same schema → the row is still there (proves the reopen is real)
#   (2) reopen with `required` + campaign_id at the SAME version 0 → RxDB must REFUSE
#       (schema-hash mismatch, DB6) — the one-line edit bricks every device holding rows
#   (3) reopen at version 1 + required + migrationStrategies {1: d => d} → row present
#   (4) at version 1, a row WITHOUT campaign_id is refused on insert → [SP-03b]'s row can
#       no longer exist on the device
# Storage: rxdb's memory storage — its state lives in a module-level map that close()
# keeps and only remove() deletes, so (1b) is a genuine close/reopen in-process. No
# fake-indexeddb is vendored anywhere, so the Dexie path (B-441) is NOT exercised here;
# the DB6 check and the migration run in rx-database.js / the migration plugin, above
# the storage layer, so the verdict is storage-agnostic by construction.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  (2) throws AND (3) migrates the row AND (4) refuses — the rider is a migration
#   exit 1  otherwise — prints "🛑 VERDICT: RED"
#   exit 2  could not run (no vendored rxdb, or the storage does not survive close/reopen)
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA_NM="$REPO_ROOT/.night-crew/qa/spike-supabase/rxdb/node_modules"
fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }
export PATH="/usr/local/go/bin:$PATH"
command -v node >/dev/null || cannot_run "node not on PATH"
[ -e "$QA_NM/rxdb/package.json" ] || cannot_run "no vendored rxdb at $QA_NM (npm ci in .night-crew/qa/spike-supabase/rxdb)"

# Node resolves `rxdb` by walking up from the .mjs; nothing is installed here, so borrow
# the proven QA node_modules via a gitignored symlink — exactly what campaigns-run.sh
# does for the harness — and remove it afterwards if this run created it.
MADE_LINK=0
if [ ! -e "$SCRIPT_DIR/node_modules" ]; then ln -s "$QA_NM" "$SCRIPT_DIR/node_modules"; MADE_LINK=1; fi
trap '[ "$MADE_LINK" = 1 ] && rm -f "$SCRIPT_DIR/node_modules"' EXIT

echo "# rxdb $(node -p 'require("'"$QA_NM"'/rxdb/package.json").version') (vendored, the harness's import); schema from marketing/sync/replicas.js"
T0=$(date +%s)
rc=0
node "$SCRIPT_DIR/02-required-campaign-id-needs-a-replica-migration.mjs" || rc=$?
echo "# wall-clock $(( $(date +%s) - T0 ))s"
case "$rc" in
  0) echo "✅ GREEN — required+campaign_id at v0 is refused on a store holding rows; v1 + migrationStrategies carries the row; a campaign_id-less row cannot be inserted at v1. The rider is a replica-schema migration (three-part shape), and [SP-03b]'s row retires with it." ;;
  1) fail "a leg contradicted the ledger — see the lines above" ;;
  2) cannot_run "see the lines above" ;;
  *) cannot_run "node exited $rc" ;;
esac
