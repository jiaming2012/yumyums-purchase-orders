#!/usr/bin/env bash
# K2 mutation legs on the FINAL tree worktree. Each mutation is applied with sed, the named test run,
# and the file restored with git checkout. :5434 only.
set -u
export PATH="/usr/local/go/bin:$PATH"
export PGPASSWORD=hqtest
W=/home/jcole/projects/hq-worktrees/triage-adv-20261007
L=/home/jcole/projects/hq/.night-crew/runs/2026-10-07-autonomous/logs/triage-adversarial
URL=$(cat $L/k2-db-url.txt)
cd $W/backend
run() { env -u HQ_RLS_TEST_DB DB_TEST_URL="$URL" go test "$@" 2>&1; echo "EXIT=$?"; }
summ() { echo "   -> PASS=$(grep -c '^--- PASS' $1) FAIL=$(grep -c '^--- FAIL' $1) SKIP=$(grep -c '^--- SKIP' $1) $(grep -E '^(ok|FAIL)\s' $1 | head -1) $(grep EXIT= $1)"; }

echo "## M1: remove ' FOR SHARE' from the target read; lock test x3"
sed -i 's/ FOR SHARE`/`/' internal/recipes/repository.go
git diff --stat -- internal/recipes/repository.go
{ echo "# M1 diff:"; git diff -- internal/recipes/repository.go; run -count=3 -run 'TestMergeMenuItem_LockHoldsAgainstConcurrentTargetDelete$' -v ./internal/recipes/; } > $L/k2-m1-no-for-share.log
git checkout -- internal/recipes/repository.go
summ $L/k2-m1-no-for-share.log
grep -m3 'repository_test.go:' $L/k2-m1-no-for-share.log | cut -c1-200

echo "## M2: handler arm swap: source_not_found answers 400 instead of 404; MissingSourceIs404"
python3 - <<'EOF'
import re,io
p='internal/recipes/handler.go'; s=open(p).read()
s=s.replace('writeError(w, http.StatusNotFound, "source_not_found")','writeError(w, http.StatusBadRequest, "source_not_found")',1)
open(p,'w').write(s)
EOF
{ echo "# M2 diff:"; git diff -- internal/recipes/handler.go; run -count=1 -run 'TestMergeMenuItem_MissingSourceIs404$' -v ./internal/recipes/; } > $L/k2-m2-handler-arm-400.log
git checkout -- internal/recipes/handler.go
summ $L/k2-m2-handler-arm-400.log
grep -m3 'repository_test.go:' $L/k2-m2-handler-arm-400.log | cut -c1-200

echo "## M3: remove the source-existence refusal (fall through on no rows); MissingSourceIs404"
python3 - <<'EOF'
p='internal/recipes/repository.go'; s=open(p).read()
s=s.replace('\t\t\treturn 0, ErrMergeSourceNotFound\n','\t\t\t_ = sourceExists // ADV MUTATION: no refusal\n',1)
open(p,'w').write(s)
EOF
{ echo "# M3 diff:"; git diff -- internal/recipes/repository.go; run -count=1 -run 'TestMergeMenuItem_MissingSourceIs404$' -v ./internal/recipes/; } > $L/k2-m3-no-source-check.log
git checkout -- internal/recipes/repository.go
summ $L/k2-m3-no-source-check.log
grep -m3 'repository_test.go:' $L/k2-m3-no-source-check.log | cut -c1-200

echo "## M4: migration 0086 SET NULL -> NO ACTION on the two item_id FKs; fresh DB; backstop test"
DBM=hq_test_triage_adv_20261007_go_k2m
URLM="postgres://hqtest:hqtest@localhost:5434/$DBM?sslmode=disable"
psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA -c "drop database if exists $DBM with (force)" -c "create database $DBM"
MIG=$(ls internal/db/migrations/0086*.sql)
sed -i -E 's/(REFERENCES menu_items\(id\)\s+ON DELETE) SET NULL/\1 NO ACTION/' "$MIG"
{ echo "# M4 diff:"; git diff -- "$MIG"; echo "# control first: the mutated migration on the fresh DB, marketing backstop test"; env -u HQ_RLS_TEST_DB DB_TEST_URL="$URLM" go test -count=1 -run 'TestMigration0086DishDeleteBlanksCampaignAndCode$' -v ./internal/marketing/ 2>&1; echo "EXIT=$?"; } > $L/k2-m4-0086-no-action.log
git checkout -- "$MIG"
summ $L/k2-m4-0086-no-action.log
grep -m3 'erasure_test.go:' $L/k2-m4-0086-no-action.log | cut -c1-220
echo "## M4 control: unmutated 0086 on another fresh DB"
DBC=hq_test_triage_adv_20261007_go_k2c
URLC="postgres://hqtest:hqtest@localhost:5434/$DBC?sslmode=disable"
psql -X -h localhost -p 5434 -U hqtest -d postgres -v ON_ERROR_STOP=1 -qtA -c "drop database if exists $DBC with (force)" -c "create database $DBC"
{ env -u HQ_RLS_TEST_DB DB_TEST_URL="$URLC" go test -count=1 -run 'TestMigration0086DishDeleteBlanksCampaignAndCode$' -v ./internal/marketing/ 2>&1; echo "EXIT=$?"; } > $L/k2-m4-control.log
summ $L/k2-m4-control.log
git status --porcelain
