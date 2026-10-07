#!/usr/bin/env bash
# triage-adversarial Go leg, run 20261007. Usage: go-leg.sh <worktree> <dbname> <migrate-port> <label>
# Same harness as logs/final-suite.sh: fresh DB on :5434, migrated by one server boot, then
# go test -p 1 -count=1 -v ./... with DB_TEST_URL set. :5434 ONLY.
set -u
export PATH="/usr/local/go/bin:$PATH"
export PGPASSWORD=hqtest
WT=$1; GO_DB=$2; PORT_MIG=$3; LABEL=$4
LOGS=/home/jcole/projects/hq/.night-crew/runs/2026-10-07-autonomous/logs/triage-adversarial
GO_URL="postgres://hqtest:hqtest@localhost:5434/$GO_DB?sslmode=disable"
PG=(psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA)
SHA=$(git -C "$WT" rev-parse HEAD)
LOG=$LOGS/go-$LABEL.log
{
  echo "# triage-adversarial go leg ($LABEL). SHA $SHA. start $(date -Iseconds)"
  echo "# cmd: go test -p 1 -count=1 -v ./...  (DB_TEST_URL=$GO_URL, migrated by one server boot on :$PORT_MIG)"
} > "$LOG"
"${PG[@]}" -c "drop database if exists $GO_DB with (force)" -c "create database $GO_DB" >> "$LOG" 2>&1
( cd "$WT/backend" && go build -o /tmp/hq-triage-adv-$LABEL-server ./cmd/server/ ) >> "$LOG" 2>&1
echo "EXIT_SERVERBUILD=$?" >> "$LOG"
( cd "$WT/backend" && exec env PORT=$PORT_MIG DB_URL="$GO_URL" STATIC_DIR=../ SUPERADMIN_CONFIG=config/superadmins.yaml \
    TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 MERCURY_API_KEY= ANTHROPIC_API_KEY= ZOHO_CLIQ_CLIENT_ID= \
    ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= STORAGE_KEY= \
    STORAGE_SECRET= STORAGE_BUCKET= STORAGE_REGION= STORAGE_ENDPOINT= /tmp/hq-triage-adv-$LABEL-server ) > $LOGS/go-$LABEL-migrate-server.log 2>&1 &
SP=$!
for i in $(seq 1 90); do grep -q "database migrations applied successfully" $LOGS/go-$LABEL-migrate-server.log && break; sleep 1; done
grep -q "database migrations applied successfully" $LOGS/go-$LABEL-migrate-server.log && echo "# migrated" >> "$LOG" || echo "# MIGRATION NOT CONFIRMED" >> "$LOG"
kill $SP 2>/dev/null; wait $SP 2>/dev/null
( cd "$WT/backend" && env -u HQ_RLS_TEST_DB -u HQ_SYNC_SUBSTRATE_OPTIONAL -u HQ_SYNC_GATE_CHILD DB_TEST_URL="$GO_URL" \
    go test -p 1 -count=1 -v ./... ) >> "$LOG" 2>&1
echo "EXIT_TEST=$?" >> "$LOG"
echo "# end $(date -Iseconds)" >> "$LOG"
echo "# counts: PASS=$(grep -c '^--- PASS' "$LOG") FAIL=$(grep -c '^--- FAIL' "$LOG") SKIP=$(grep -c '^--- SKIP' "$LOG") pkgs_ok=$(grep -c '^ok ' "$LOG") pkgs_fail=$(grep -c '^FAIL' "$LOG") notests=$(grep -c 'no test files' "$LOG")" >> "$LOG"
echo DONE_$LABEL
