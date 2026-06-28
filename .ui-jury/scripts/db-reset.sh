#!/bin/bash
# scripts/ui-jury/db-reset.sh — ui-jury db_reset hook for hq.
#
# Wipes inventory + receipt data only (purchase_line_items, purchase_events,
# pending_purchases). Preserves users (needed for the routes.yaml setup login)
# and workflow templates (needed for /workflows.html to render anything
# meaningful). Idempotent — safe to re-invoke between fixture groups.
#
# Wraps `task backend:db-reset-inventory` (backend/Taskfile.yml:205-210).
# Do NOT use `task backend:db-reset` — that drops the docker volume, losing
# users and templates needed by other parts of the run.

set -euo pipefail

# Always run from repo root regardless of CWD at invocation time. The orchestrator
# already cwd's to project root before invoking, but this makes the script
# safe to run by hand for smoke-testing.
cd "$(dirname "$0")/../.."

exec task backend:db-reset-inventory
