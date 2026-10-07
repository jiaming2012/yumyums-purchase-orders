#!/usr/bin/env bash
# triage-adversarial Playwright leg, run 20261007.
# Usage: pw-leg.sh <worktree> <port> <dbname> <logname> [playwright args...]
# Same shape as Taskfile test:* legs and logs/final-suite.sh: Playwright's own webServer,
# DB on :5434 ONLY (role hqtest). Each call drops+recreates its own TEST_DB_NAME via reset-e2e-db.js.
set -u
export PATH="/usr/local/go/bin:$PATH"
WT=$1; PORT=$2; DBN=$3; LOGNAME=$4; shift 4
LOGS=/home/jcole/projects/hq/.night-crew/runs/2026-10-07-autonomous/logs/triage-adversarial
LOG=$LOGS/$LOGNAME.log
cd "$WT"
export DB_HOST=localhost DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest TEST_PORT=$PORT TEST_DB_NAME=$DBN
export TEST_OUTPUT_DIR=/tmp/claude-1000/-home-jcole-projects-hq/d35260ba-2f67-4dc1-9824-435e818e0e51/scratchpad/pw-out-$LOGNAME
{
  echo "# triage-adversarial pw leg $LOGNAME. WT=$WT SHA=$(git rev-parse HEAD) start $(date -Iseconds)"
  echo "# env: TEST_PORT=$PORT TEST_DB_NAME=$DBN DB_PORT=5434 DB_USER=hqtest"
  echo "# cmd: npx playwright test --reporter=list $*"
  echo "# git status --porcelain BEFORE: $(git status --porcelain | wc -l) lines"
} > "$LOG"
npx playwright test --reporter=list "$@" >> "$LOG" 2>&1
echo "EXIT_PW=$?" >> "$LOG"
echo "# git status --porcelain AFTER:" >> "$LOG"
git status --porcelain >> "$LOG"
echo "# porcelain_lines=$(git status --porcelain | wc -l)" >> "$LOG"
echo "# end $(date -Iseconds)" >> "$LOG"
grep -E '^\s+[0-9]+ (passed|failed|flaky|skipped)|^EXIT_PW|porcelain_lines' "$LOG"
