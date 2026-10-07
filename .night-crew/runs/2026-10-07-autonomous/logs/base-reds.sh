#!/usr/bin/env bash
# base-reds leg, run 20261007 — full Go suite then full Playwright suite on the UNMODIFIED
# dev base, in a detached worktree, both under /tmp/hq-full-suite.lock. :5434 only.
# Same harness as run 20261006's base leg (Playwright's own webServer, config retries = 1).
set -u
export PATH="/usr/local/go/bin:$PATH"
export PGPASSWORD=hqtest
REPO=/home/jcole/projects/hq
WT=/home/jcole/projects/hq-worktrees/base-20261007
LOGS=$REPO/.night-crew/runs/2026-10-07-autonomous/logs
GO_DB=hq_test_go_base
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
PORT_PW=8271
PORT_MIG=8371

[ -d "$WT" ] || git -C "$REPO" worktree add --detach "$WT" dev >/dev/null
SHA=$(git -C "$WT" rev-parse HEAD)
ln -sfn $REPO/node_modules "$WT/node_modules"
ln -sfn $REPO/marketing/sync/harness/node_modules "$WT/marketing/sync/harness/node_modules"
ln -sfn $REPO/.night-crew/qa/spike-supabase/rxdb/node_modules "$WT/.night-crew/qa/spike-supabase/rxdb/node_modules"

# ---- Go ----
{
  echo "# base-go leg, run 20261007. SHA $SHA. start $(date -Iseconds)"
  echo "# cmd: flock -o /tmp/hq-full-suite.lock go test -p 1 -count=1 -v ./...  (DB_TEST_URL=$GO_URL, migrated by one server boot)"
} > $LOGS/base-go.log
"${PG[@]}" -c "drop database if exists $GO_DB with (force)" -c "create database $GO_DB" >> $LOGS/base-go.log 2>&1
( cd "$WT/backend" && go build -o /tmp/hq-base-20261007-server ./cmd/server/ ) >> $LOGS/base-go.log 2>&1
echo "EXIT_SERVERBUILD=$?" >> $LOGS/base-go.log
( cd "$WT/backend" && exec env PORT=$PORT_MIG DB_URL="$GO_URL" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml \
    TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= \
    ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= \
    STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= /tmp/hq-base-20261007-server ) > $LOGS/base-migrate-server.log 2>&1 &
SP=$!
for i in $(seq 1 90); do grep -q "database migrations applied successfully" $LOGS/base-migrate-server.log && break; sleep 1; done
grep -q "database migrations applied successfully" $LOGS/base-migrate-server.log && echo "# migrated" >> $LOGS/base-go.log || echo "# MIGRATION NOT CONFIRMED" >> $LOGS/base-go.log
kill $SP 2>/dev/null; wait $SP 2>/dev/null
echo "# waiting for lock $(date -Iseconds)" >> $LOGS/base-go.log
( cd "$WT/backend" && env -u HQ_RLS_TEST_DB -u HQ_SYNC_SUBSTRATE_OPTIONAL -u HQ_SYNC_GATE_CHILD DB_TEST_URL="$GO_URL" \
    flock -o /tmp/hq-full-suite.lock go test -p 1 -count=1 -v ./... ) >> $LOGS/base-go.log 2>&1
echo "EXIT_TEST=$?" >> $LOGS/base-go.log
echo "# end $(date -Iseconds)" >> $LOGS/base-go.log

# ---- Playwright ----
{
  echo "# base-pw leg, run 20261007. SHA $SHA. start $(date -Iseconds)"
  echo "# cmd: npx bddgen; warm build; flock -o /tmp/hq-full-suite.lock npx playwright test — 1 worker, config retries (1). TEST_PORT=$PORT_PW TEST_DB_NAME=hq_test_e2e_base_20261007 DB_PORT=5434 DB_USER=hqtest"
} > $LOGS/base-pw.log
cd "$WT"
export DB_HOST=localhost DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest TEST_PORT=$PORT_PW TEST_DB_NAME=hq_test_e2e_base_20261007
npx bddgen >> $LOGS/base-pw.log 2>&1; echo "EXIT_BDDGEN=$?" >> $LOGS/base-pw.log
( cd backend && go build -o /dev/null ./cmd/server/ ) >> $LOGS/base-pw.log 2>&1; echo "EXIT_SERVERBUILD=$?" >> $LOGS/base-pw.log
echo "# waiting for lock $(date -Iseconds)" >> $LOGS/base-pw.log
flock -o /tmp/hq-full-suite.lock npx playwright test --reporter=list >> $LOGS/base-pw.log 2>&1
echo "EXIT_PW=$?" >> $LOGS/base-pw.log
echo "# end $(date -Iseconds)" >> $LOGS/base-pw.log
git -C "$WT" status --porcelain > $LOGS/base-pw-tree-after.txt
git -C "$WT" checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/ .night-crew/runs/2026-10-02-autonomous/logs/h5/states/ 2>/dev/null
echo BASE_REDS_DONE
