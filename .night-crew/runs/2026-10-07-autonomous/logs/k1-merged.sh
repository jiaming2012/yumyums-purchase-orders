#!/usr/bin/env bash
# Card 1's FULL Playwright suite on the merged tree (under the lock), then the measurement leg:
# three Inventory tests 10x alone each (--retries=0), then the inventory|recipes seam 5x (--retries=0).
set -u
export PATH="/usr/local/go/bin:$PATH"
REPO=/home/jcole/projects/hq
SHA=$1
WT=/home/jcole/projects/hq-worktrees/k1-merged-20261007
LOGS=$REPO/.night-crew/runs/2026-10-07-autonomous/logs
[ -d "$WT" ] || git -C "$REPO" worktree add --detach "$WT" "$SHA" >/dev/null
ln -sfn $REPO/node_modules "$WT/node_modules"
ln -sfn $REPO/marketing/sync/harness/node_modules "$WT/marketing/sync/harness/node_modules"
ln -sfn $REPO/.night-crew/qa/spike-supabase/rxdb/node_modules "$WT/.night-crew/qa/spike-supabase/rxdb/node_modules"
cd "$WT"
export DB_HOST=localhost DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest TEST_PORT=8272 TEST_DB_NAME=hq_test_e2e_k1merged_20261007
restore() { git checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/ .night-crew/runs/2026-10-02-autonomous/logs/h5/states/ sw.js version.json 2>/dev/null; }
F=$LOGS/k1-full-pw.log
{ echo "# Card 1 full suite on merged tree $(git rev-parse HEAD). start $(date -Iseconds)"; echo "# cmd: flock -o /tmp/hq-full-suite.lock npx playwright test --reporter=list — 1 worker, config retries (1). TEST_PORT=$TEST_PORT TEST_DB_NAME=$TEST_DB_NAME :5434"; } > $F
npx bddgen >> $F 2>&1; echo "EXIT_BDDGEN=$?" >> $F
( cd backend && go build -o /dev/null ./cmd/server/ ) >> $F 2>&1; echo "EXIT_SERVERBUILD=$?" >> $F
echo "# waiting for lock $(date -Iseconds)" >> $F
flock -o /tmp/hq-full-suite.lock bash -c 'echo "# lock acquired $(date -Iseconds)"; npx playwright test --reporter=list' >> $F 2>&1
echo "EXIT_PW=$?" >> $F; echo "# end $(date -Iseconds)" >> $F
git status --porcelain > $LOGS/k1-full-pw-tree-after.txt; restore
M=$LOGS/k1-measure.log
echo "# Card 1 measurement leg on merged tree $(git rev-parse HEAD). start $(date -Iseconds). --retries=0 throughout" > $M
export TEST_PORT=8273 TEST_DB_NAME=hq_test_e2e_k1meas_20261007
for T in "Setup item editor shows alias chips" "create item without group shows alert" "creating item opens edit form with store location dropdown"; do
  for i in 1 2 3 4 5 6 7 8 9 10; do
    npx playwright test tests/inventory.spec.js -g "$T" --retries=0 --reporter=line > $LOGS/k1-measure-last.log 2>&1; rc=$?
    echo "ALONE | $T | run $i | EXIT=$rc | $(grep -E '^\s+[0-9]+ (passed|failed)' $LOGS/k1-measure-last.log | tr -s ' ' | tr '\n' ';')" >> $M
    [ $rc -ne 0 ] && { echo "---- failing output ($T run $i)" >> $M; grep -v WebServer $LOGS/k1-measure-last.log | tail -40 >> $M; }
    restore
  done
done
for i in 1 2 3 4 5; do
  npx playwright test "inventory|recipes" --retries=0 --reporter=list > $LOGS/k1-measure-seam-$i.log 2>&1; rc=$?
  echo "SEAM inventory|recipes | run $i | EXIT=$rc | $(grep -E '^\s+[0-9]+ (passed|failed|skipped|flaky)' $LOGS/k1-measure-seam-$i.log | tr -s ' ' | tr '\n' ';')" >> $M
  restore
done
echo "# end $(date -Iseconds)" >> $M
echo K1_MERGED_DONE
