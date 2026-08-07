#!/usr/bin/env bash
# spike-e-reconnect.sh — SPIKE E. Night-crew card E `spike-e-reconnect-catchup`.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   GREEN. A replicating RxDB client that was DISCONNECTED while rows
#            changed recovered EVERYTHING on reconnect via checkpoint pull —
#            including a row UPDATED IN PLACE that the client already held.
#
#   exit 1   RED — ran, and catch-up demonstrably MISSES dark-window changes.
#            🛑 A RED VERDICT IS A SUCCESSFUL SPIKE AND IS THE DELIVERABLE. It
#            means the Activity 3 build cards need an explicit resync step, and
#            finding that out costs one night now versus a crew member's phone
#            sleeping through a write in production. Record it with the script's
#            output and STOP. Debugging the harness is legitimate. Rewriting the
#            goal so it passes is not.
#
#   exit 2   COULD NOT RUN. Docker down, the substrate would not come up, the
#            scratch Postgres never became healthy, HQ's migrator failed, login
#            failed, /saveResponse did not return 204, the relay never projected
#            a dark-window row at all, the sever leaked, or the reconnected
#            client is provably dead. 🛑 THIS IS NOT A VERDICT. It says nothing
#            about catch-up and must never be reported as red.
#
#   exit 3   A verdict was reached BUT spike A's shared substrate could not be
#            restored. Rows left in hq_sync_checklists / hq_grant_projection WILL
#            red backend/internal/sync's TestJWTBridgeRLS, whose service_role
#            CONTROL asserts an EXACT full-table row set. Repair before trusting
#            anything.
#
#   exit 64  usage error.
#
# There is deliberately no "warn and continue" anywhere in this file and no
# advisory leg. A step that cannot decide is a FAILURE. That is spike A's rule
# (env-up.sh:18-27), carried through B, C and D unchanged.
# ═══════════════════════════════════════════════════════════════════════════
#
# ───────────────────────────────────────────────────────────────────────────
# THE QUESTION, AND WHY IT IS NOT ALREADY ANSWERED
#
# Spike C proved the round trip — one write through HQ's REAL write path reaches
# a RUNNING RxDB client. Spike D proved the live Realtime filter is honoured.
# NEITHER EVER SEVERED A CLIENT. Every Activity 3 card assumes a phone that was
# asleep, backgrounded or out of signal while rows changed comes back and catches
# up, and nothing in this repo has ever measured that.
#
# The sharp end is the UPDATE case. `sync-rxdb/client.js:874` filters the
# checklists collection on `submitted_at=gte.<iso>` — an advancing watermark keyed
# on a BUSINESS timestamp. HQ never re-stamps `submitted_at`: migration 0011 gives
# it `DEFAULT now()` on insert, and the only UPDATEs HQ issues against
# checklist_submissions (repository.go:1186 approve, :1232 reject) set
# status/reviewed_by/reviewed_at and leave submitted_at alone. That is the exact
# unreliable-watermark shape that disqualified `answered_at` for spike C's polling
# relay candidate. A row updated without its watermark advancing is invisible to a
# watermark-keyed catch-up, and it is invisible SILENTLY.
#
# So this card writes, while the client is dark, an INSERT *and* an UPDATE to a
# row the client already holds, and measures what comes back.
# ───────────────────────────────────────────────────────────────────────────
#
# ⚠ WHAT THIS CARD REUSES, AND HOW.
#   Spike C's harness END TO END, and spike D's substrate discipline, both
#   READ-ONLY: env-up.sh, docker-compose.hq-real.yml, sql/spike-c-relay-trigger.sql,
#   backend/cmd/spikec-relay, mintjwt/ and rxdb/spike-env.js are consumed exactly
#   as they are, not edited, so spikes A-D's four GREEN verdicts keep reproducing
#   byte-for-byte. What this card needs that they do not have is added as a
#   SIBLING (this file, rxdb/spike-e-reconnect.js) — the same addition-not-edit
#   rule spike D applied when it added rtprobe/ beside rtwatch/.
#
#   🛑 THE RELAY IS NOT THE MECHANISM UNDER TEST HERE. Spike C already proved it.
#   It is armed on BOTH paths of this card. The thing this card removes for its
#   red is the CHECKPOINT PULL.
#
# ⚠ CONTAINERS — the standing rule, absolute.
#   The HQ Postgres is a FRESH SCRATCH CONTAINER under its own compose project
#   `spike-e-hq`, on a DOCKER-ASSIGNED EPHEMERAL host port.
#     * NEVER :5433. That cluster is PRODUCTION AND DEV BOTH — it serves
#       https://hq.yumyums.kitchen — and a probe against it destroyed the prod
#       database on 2026-08-06 (B-141/B-143, ledger decision 155).
#     * NEVER :5434 (`yumyums-test-pg`). NEVER 5432 (`infra-postgres-1`).
#   The refusal below is a RUNTIME CHECK on the port Docker actually assigned,
#   not a comment.
#
# ⚠ SPIKE A'S STACK IS CONSUMED, NOT MODIFIED — RECONCILE ONLY.
#   🛑 THIS SCRIPT HAS NO --fresh / --fresh-substrate FLAG, DELIBERATELY. Spike C
#   carries one; it is omitted here so it cannot be typed by accident. `--fresh`
#   does `down --volumes` on the `spike-supabase` project and would eat another
#   session's running substrate. B-159 names this exact footgun.
#
# ⚠ THE SUBSTRATE IS RESTORED, AND THE RESTORE IS VERIFIED, ON EVERY PATH.
#   B-148's standard. Snapshot the exact id set of hq_sync_checklists and the
#   exact pair set of hq_grant_projection BEFORE anything is written; remove what
#   was added; ASSERT the sets are identical again. That assertion runs on green,
#   on red and on abort, so the recovery path is rehearsed every single run.
#
# USAGE
#   .night-crew/qa/spike-supabase/spike-e-reconnect.sh
#   .night-crew/qa/spike-supabase/spike-e-reconnect.sh --no-pull   # RED-FIRST: realtime-only recovery
#   .night-crew/qa/spike-supabase/spike-e-reconnect.sh --keep      # leave the scratch pg + server up
#
#   or via the repo Taskfile:  task spike:reconnect   /   task spike:reconnect:red

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

HQ_COMPOSE="$SPIKE_DIR/docker-compose.hq-real.yml"   # spike C's, read-only
HQ_PROJECT="spike-e-hq"                              # this card's OWN project
HQ_DB_USER="hq"
HQ_DB_NAME="hq_real"
HQ_DB_PASS="9a1e5c7b3d206f84e5b90c1a7d43f68e"        # throwaway; see the compose banner

SYNC_TABLE="hq_sync_checklists"
PROJ_TABLE="hq_grant_projection"
APP_SLUG="operations"
SUPERADMIN_EMAIL="jamal@yumyums.kitchen"
SUPERADMIN_PASSWORD="test123"                        # backend/config/superadmins.yaml dev_password

# Deliberately NOT 8089 (dev server), NOT 8199 (Playwright default), NOT 8471
# (spike C's) and NOT 4823 (this card's Playwright gate port). A spike that
# silently attached to a dev server would be measuring the wrong database.
HQ_API_PORT="${SPIKE_E_API_PORT:-8472}"

# The bound each arrival gets. Same generosity as spike C, and for the same
# reason: the question is whether catch-up HAPPENS, and a tight bound would let a
# slow-but-working mechanism read as "does not exist".
DEADLINE_MS="${SPIKE_E_DEADLINE_MS:-20000}"

KEEP=0
NO_PULL=0
for arg in "$@"; do
  case "$arg" in
    --keep)     KEEP=1 ;;
    --no-pull)  NO_PULL=1 ;;
    *) echo "usage: $(basename "$0") [--keep] [--no-pull]" >&2; exit 64 ;;
  esac
done

# Go is not on a non-interactive shell's PATH on this box and this script needs it
# three times over (HQ's server, the relay, the JWT minter). Without this they die
# `go: not found` (exit 127) and it LOOKS like a substrate failure when it is a
# PATH failure. Same list env-up.sh / spike-b / spike-c / spike-d use.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
# 🛑 TWO failure verbs, and the distinction is the card's requirement.
#   cannot_run  -> exit 2. Infrastructure/setup. NOT a verdict.
#   red         -> exit 1. Ran, catch-up disproven. A successful spike.
cannot_run() { printf '\n🛑 COULD NOT RUN (not a verdict) — %s\n' "$1" >&2; exit 2; }
red()        { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }

# Anchor compose the way env-up.sh does, so the project directory is identical
# whether this runs from the main checkout or a worktree.
ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
HQDC=(docker compose -p "$HQ_PROJECT" --project-directory "$ANCHOR" -f "$HQ_COMPOSE")

RUN_ID="${SPIKE_E_RUN_ID:-e$(date -u +%Y%m%d%H%M%S)}"
export SPIKE_E_RUN_ID="$RUN_ID"
WORK="$(mktemp -d -t spike-e-XXXXXX)"

printf '# spike-e-reconnect.sh — sever a replicating RxDB client, write while dark, reconnect, measure catch-up\n'
printf '# repo    %s\n' "$REPO_ROOT"
printf '# anchor  %s\n' "$ANCHOR"
printf '# run     %s\n' "$RUN_ID"
printf '# mode    %s\n' "$([ "$NO_PULL" = 1 ] && echo 'NO-PULL (red-first capture: realtime-only recovery, checkpoint pull absent)' || echo 'checkpoint pull ARMED')"
printf '# started %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown. Registered BEFORE anything is created, so an abort between `up` and
# the first assertion still cleans up. It preserves the run's exit status — that
# status is the verdict — with exactly ONE exception: a FAILED substrate restore
# forces exit 3, because a run that leaves spike A's shared tables dirty has
# broken a committed Go suite whether or not its own assertions passed.
# --------------------------------------------------------------------------
RELAY_PID=""
SERVER_PID=""
BASELINE_SYNC=""
BASELINE_PROJ=""
SERVICE_TOKEN=""
REST_BASE=""
HQ_USER_ID=""

teardown() {
  local rc=$?
  set +e

  if [ -n "$RELAY_PID" ]; then kill "$RELAY_PID" 2>/dev/null; wait "$RELAY_PID" 2>/dev/null; fi
  if [ -n "$SERVER_PID" ]; then kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null; fi

  # ---- (a) restore spike A's SHARED substrate, and VERIFY the restore --------
  # 🛑 NOT TIDINESS. hq_sync_checklists is shared and
  # backend/internal/sync/jwtbridge_rls_test.go's service_role CONTROL asserts an
  # EXACT full-table row set. Rows left behind by a spike red a committed Go suite
  # — that has already happened once (spike B's first G2 run, four subtests).
  if [ -n "$SERVICE_TOKEN" ] && [ -n "$REST_BASE" ] && [ -n "$BASELINE_SYNC" ]; then
    printf '\n── teardown (1/2): restoring spike A'"'"'s shared substrate ──\n'
    # Everything projected into the shared table is prefixed `spikec-` by
    # backend/internal/sync/spikec_relay.go (consumed unchanged — this card adds
    # no relay of its own). Deleting by prefix cannot touch spike A's fixture rows
    # or spike B's, and the verification below PROVES that rather than asserting it.
    curl -sS -X DELETE -H "Authorization: Bearer $SERVICE_TOKEN" \
      "$REST_BASE/$SYNC_TABLE?id=like.spikec-*" >/dev/null 2>&1
    if [ -n "$HQ_USER_ID" ]; then
      curl -sS -X DELETE -H "Authorization: Bearer $SERVICE_TOKEN" \
        "$REST_BASE/$PROJ_TABLE?user_id=eq.$HQ_USER_ID&app_slug=eq.$APP_SLUG" >/dev/null 2>&1
    fi

    local after_sync after_proj
    after_sync="$(curl -sS -H "Authorization: Bearer $SERVICE_TOKEN" \
      "$REST_BASE/$SYNC_TABLE?select=id&order=id" 2>/dev/null)"
    after_proj="$(curl -sS -H "Authorization: Bearer $SERVICE_TOKEN" \
      "$REST_BASE/$PROJ_TABLE?select=user_id,app_slug&order=user_id,app_slug" 2>/dev/null)"

    if [ "$after_sync" = "$BASELINE_SYNC" ] && [ "$after_proj" = "$BASELINE_PROJ" ]; then
      printf '  VERIFIED: %s and %s are byte-identical to the pre-run baseline.\n' "$SYNC_TABLE" "$PROJ_TABLE"
    else
      printf '  🛑 THE SUBSTRATE RESTORE FAILED OR WAS INCOMPLETE.\n'
      printf '     %s baseline: %s\n' "$SYNC_TABLE" "$BASELINE_SYNC"
      printf '     %s after   : %s\n' "$SYNC_TABLE" "$after_sync"
      printf '     %s baseline: %s\n' "$PROJ_TABLE" "$BASELINE_PROJ"
      printf '     %s after   : %s\n' "$PROJ_TABLE" "$after_proj"
      printf '     These rows WILL red backend/internal/sync TestJWTBridgeRLS. Repair before\n'
      printf '     trusting any verdict from this run.\n'
      rc=3
    fi
  else
    printf '\n── teardown (1/2): nothing was written to the substrate; nothing to restore ──\n'
  fi

  # ---- (b) destroy the scratch HQ Postgres ---------------------------------
  if [ "$KEEP" = "1" ]; then
    printf '\n(--keep) scratch HQ Postgres LEFT RUNNING: docker compose -p %s ... down --volumes\n' "$HQ_PROJECT"
  else
    printf '\n── teardown (2/2): destroying the scratch HQ Postgres (project %s) ──\n' "$HQ_PROJECT"
    "${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || \
      printf '  ⚠ teardown of project %s did not complete cleanly — check `docker ps -a`\n' "$HQ_PROJECT"
    printf '  torn down.\n'
  fi
  rm -rf "$WORK"

  # 🛑 `exit`, NOT `return`. Inside an EXIT trap a `return` cannot change the
  #    script's status, so a failed substrate restore would be swallowed.
  exit $rc
}
trap teardown EXIT

# --------------------------------------------------------------------------
step "preflight — required tooling"
# --------------------------------------------------------------------------
for bin in docker curl node go; do
  command -v "$bin" >/dev/null 2>&1 || cannot_run "required tool not on PATH: $bin"
  printf '  %-6s %s\n' "$bin" "$(command -v "$bin")"
done
docker compose version >/dev/null 2>&1 || cannot_run "Compose v2 unavailable — needs 'docker compose'"
docker info >/dev/null 2>&1 || cannot_run "the Docker daemon is not reachable — 'docker info' failed"
[ -f "$HQ_COMPOSE" ] || cannot_run "compose file missing: $HQ_COMPOSE"
[ -f "$SPIKE_DIR/rxdb/spike-e-reconnect.js" ] || cannot_run "spike asset missing: rxdb/spike-e-reconnect.js"
[ -f "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" ] || cannot_run "spike asset missing: sql/spike-c-relay-trigger.sql (spike C's, consumed read-only)"

# 🛑 ISOLATION ASSERTION, NOT A COMMENT.
if grep -Eq '^[[:space:]]*-[[:space:]]*"?(5432|5433|5434):' "$HQ_COMPOSE"; then
  cannot_run "docker-compose.hq-real.yml publishes a FIXED host port in the 5432-5434 range. \
:5433 is the PRODUCTION cluster (a probe there destroyed the prod DB on 2026-08-06) and \
:5434 is yumyums-test-pg. This spike must use a Docker-assigned ephemeral port."
fi
echo "  isolation: hq-real compose publishes no fixed host port (Docker-assigned only)"

# --------------------------------------------------------------------------
step "substrate — spike A's Supabase + RxDB environment (RECONCILE, never destroy)"
# --------------------------------------------------------------------------
# 🛑 No arguments. This script cannot pass --fresh; the flag does not exist here.
echo "  delegating to env-up.sh (reconcile, no flags); its exit status gates this leg"
"$SPIKE_DIR/env-up.sh" \
  || cannot_run "the substrate did not come up — env-up.sh returned non-zero. Its own output above names the leg."

[ -d "$SPIKE_DIR/rxdb/node_modules" ] || cannot_run "rxdb/node_modules is still missing after env-up.sh — the RxDB harness cannot run"

REST_PORT="$(docker compose -p spike-supabase --project-directory "$ANCHOR" \
  -f "$REPO_ROOT/docker-compose.supabase.yml" port rest 3000 2>/dev/null | sed 's/.*://')"
case "$REST_PORT" in
  ''|*[!0-9]*) cannot_run "could not resolve spike A's PostgREST host port (got '$REST_PORT')" ;;
esac
REST_BASE="http://127.0.0.1:$REST_PORT"
echo "  PostgREST: $REST_BASE"

# Read the throwaway JWT secret rather than re-typing it, so a rotation in spike
# A's compose cannot leave this script signing with a stale secret and reporting a
# catch-up failure that is really a 401.
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"

SERVICE_TOKEN="$(cd "$SPIKE_DIR" && go run ./mintjwt -secret "$JWT_SECRET" -sub "spike-e-relay" -role service_role -ttl 60m)" \
  || cannot_run "minting the service_role token failed"
[ -n "$SERVICE_TOKEN" ] || cannot_run "the service_role token came back empty"
echo "  service_role token minted (role=service_role, ttl=60m)"

# --------------------------------------------------------------------------
step "baseline snapshot of spike A's SHARED tables (so the restore can be VERIFIED)"
# --------------------------------------------------------------------------
BASELINE_SYNC="$(curl -sS -H "Authorization: Bearer $SERVICE_TOKEN" "$REST_BASE/$SYNC_TABLE?select=id&order=id")" \
  || cannot_run "could not read $SYNC_TABLE through PostgREST with the service token"
BASELINE_PROJ="$(curl -sS -H "Authorization: Bearer $SERVICE_TOKEN" "$REST_BASE/$PROJ_TABLE?select=user_id,app_slug&order=user_id,app_slug")" \
  || cannot_run "could not read $PROJ_TABLE through PostgREST with the service token"
case "$BASELINE_SYNC" in
  '['*) ;;
  *) cannot_run "the $SYNC_TABLE baseline is not a JSON array — PostgREST said: $BASELINE_SYNC" ;;
esac
printf '  %s baseline: %s\n' "$SYNC_TABLE" "$BASELINE_SYNC"
printf '  %s baseline: %s\n' "$PROJ_TABLE" "$BASELINE_PROJ"
# A pre-existing spikec- row would make the teardown verification pass vacuously
# and could make a dark-window change look pre-arrived.
case "$BASELINE_SYNC" in
  *spikec-*) cannot_run "the substrate already holds spikec- rows from an aborted run. Remove them first: curl -X DELETE -H 'Authorization: Bearer <service token>' '$REST_BASE/$SYNC_TABLE?id=like.spikec-*'" ;;
esac

# --------------------------------------------------------------------------
step "HQ Postgres — fresh scratch container (project $HQ_PROJECT), EMPTY"
# --------------------------------------------------------------------------
"${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
"${HQDC[@]}" up -d || cannot_run "'docker compose up -d' failed for project $HQ_PROJECT"
"${HQDC[@]}" ps

HQ_CID="$("${HQDC[@]}" ps -q hqreal || true)"
[ -n "$HQ_CID" ] || cannot_run "the hqreal service has no container — 'docker compose -p $HQ_PROJECT ps -a' will say why"

deadline=$(( $(date +%s) + 120 ))
while :; do
  state="$(docker inspect -f '{{.State.Status}}' "$HQ_CID" 2>/dev/null || echo missing)"
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$HQ_CID" 2>/dev/null || echo none)"
  [ "$health" = "healthy" ] && { echo "  hqreal: $state/$health"; break; }
  case "$state" in
    exited|dead|missing) cannot_run "hqreal container is '$state' — 'docker logs $HQ_CID' will say why" ;;
  esac
  [ "$(date +%s)" -lt "$deadline" ] || cannot_run "hqreal never became healthy within 120s (last state=$state health=$health)"
  sleep 2
done

HQ_PORT="$("${HQDC[@]}" port hqreal 5432 2>/dev/null | sed 's/.*://' || true)"
case "$HQ_PORT" in
  ''|*[!0-9]*) cannot_run "could not resolve the Docker-assigned host port for hqreal:5432 (got '$HQ_PORT')" ;;
esac
case "$HQ_PORT" in
  5432|5433|5434) cannot_run "Docker assigned host port $HQ_PORT to the scratch Postgres, which collides with a protected port (5433 = PRODUCTION cluster, 5434 = yumyums-test-pg). Refusing to continue. Re-run; the ephemeral range will pick another." ;;
esac
HQ_DSN="postgres://$HQ_DB_USER:$HQ_DB_PASS@127.0.0.1:$HQ_PORT/$HQ_DB_NAME?sslmode=disable"
printf '  container %s  ·  host port %s (Docker-assigned, ephemeral)\n' "${HQ_CID:0:12}" "$HQ_PORT"

srcpsql() { docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -t -A -c "$1"; }

# --------------------------------------------------------------------------
step "HQ's REAL schema — applied by HQ's OWN binary, not by a transcription"
# --------------------------------------------------------------------------
echo "  building HQ's server binary"
( cd "$REPO_ROOT/backend" && go build -o "$WORK/hq-server" ./cmd/server ) \
  || cannot_run "'go build ./cmd/server' failed — this is a build failure, not a catch-up finding"

MIG_COUNT="$(find "$REPO_ROOT/backend/internal/db/migrations" -name '*.sql' | wc -l | tr -d ' ')"
echo "  booting it against the scratch Postgres (it will apply $MIG_COUNT migrations)"

API="http://127.0.0.1:$HQ_API_PORT"

# 🛑 REFUSE TO REUSE A SERVER WE DID NOT START. Spike C's measured failure: an
# orphaned server from a previous run answered the health poll in milliseconds
# while THIS run's migrator was three migrations in. Both halves of the fix are
# here — the pre-flight refusal, and starting the binary DIRECTLY (below) so $! is
# the server itself and teardown's kill actually lands.
if curl -fsS --max-time 2 "$API/api/v1/health" >/dev/null 2>&1; then
  cannot_run "something is ALREADY serving $API/api/v1/health. This spike must never attach to a server it did not start — that server has a different database. Kill it (pkill -f hq-server) or set SPIKE_E_API_PORT to a free port."
fi

# STATIC_DIR is set for the SAME reason playwright.config.js sets it: main.go
# computes `secureCookie := os.Getenv("STATIC_DIR") == ""`, so leaving it unset
# yields Secure cookies curl and fetch never send back over http and every
# authenticated call 401s for a reason unrelated to this card. The blanked
# credential vars mirror playwright.config.js's webServer command verbatim — the
# root Taskfile's `dotenv: ['backend/.env']` injects LIVE Mercury/Anthropic/Cliq/
# SMTP credentials into anything launched from the checkout.
# 🛑 `env ... binary &`, NEVER `( ... ) &`. A subshell makes $! the SUBSHELL's pid;
# teardown's `kill` then reaps the subshell and leaves an orphan bound to the port.
env PORT="$HQ_API_PORT" DB_URL="$HQ_DSN" STATIC_DIR="$REPO_ROOT" \
  SUPERADMIN_CONFIG="$REPO_ROOT/backend/config/superadmins.yaml" \
  TEMPLATE_CONFIG="$REPO_ROOT/backend/config/templates.yaml" \
  TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
  MERCURY_API_KEY= ANTHROPIC_API_KEY= \
  ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
  SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= \
  "$WORK/hq-server" > "$WORK/hq-server.log" 2>&1 &
SERVER_PID=$!

deadline=$(( $(date +%s) + 90 ))
until curl -fsS "$API/api/v1/health" >/dev/null 2>&1; do
  kill -0 "$SERVER_PID" 2>/dev/null || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ's server exited before it became healthy (migrations or boot failed — log above)"; }
  [ "$(date +%s)" -lt "$deadline" ] || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ's server never answered /api/v1/health within 90s"; }
  sleep 1
done
echo "  health: $(curl -fsS "$API/api/v1/health")"

APPLIED="$(srcpsql "select count(*) from goose_db_version" 2>/dev/null || echo 0)"
TABLES="$(srcpsql "select count(*) from information_schema.tables where table_schema='public'")"
printf '  goose versions applied: %s  ·  public tables: %s\n' "$APPLIED" "$TABLES"
[ "$TABLES" -gt 30 ] || cannot_run "only $TABLES public tables exist — HQ's migrator did not run. See $WORK/hq-server.log"
srcpsql "select 1 from information_schema.tables where table_schema='public' and table_name='submission_responses'" | grep -q 1 \
  || cannot_run "submission_responses does not exist — the real write path has nowhere to write"

# --------------------------------------------------------------------------
step "the write path's real prerequisites — a real session and FOUR real fields"
# --------------------------------------------------------------------------
HQ_USER_ID="$(srcpsql "select id from users where email='$SUPERADMIN_EMAIL'")"
[ -n "$HQ_USER_ID" ] || cannot_run "the superadmin from config/superadmins.yaml was not upserted into users"
echo "  hq user: $HQ_USER_ID ($SUPERADMIN_EMAIL)"

# A REAL login through the REAL endpoint. Not a hand-inserted session row: the
# card says the REAL write path, and the session is part of it.
LOGIN_CODE="$(curl -sS -o "$WORK/login.json" -w '%{http_code}' \
  -c "$WORK/cookies.txt" -H 'Content-Type: application/json' \
  -d "{\"email\":\"$SUPERADMIN_EMAIL\",\"password\":\"$SUPERADMIN_PASSWORD\"}" \
  "$API/api/v1/auth/login")"
[ "$LOGIN_CODE" = "200" ] || { cat "$WORK/login.json"; cannot_run "POST /api/v1/auth/login returned HTTP $LOGIN_CODE"; }
SESSION="$(awk '$6=="hq_session"{print $7}' "$WORK/cookies.txt" | tail -1)"
[ -n "$SESSION" ] || cannot_run "login succeeded but set no hq_session cookie"
echo "  session: real hq_session cookie from POST /api/v1/auth/login (HTTP 200)"

# FOUR distinct fields, because the card needs four distinguishable writes:
#   A  dark-window INSERT       B  pre-sever save, then dark-window UPDATE
#   C  dark-window INSERT       D  post-reconnect liveness control
# `text` preferred so the sentinel is a natural value for the field's own type;
# saveResponse does not type-check the value, so any four are acceptable and
# falling back rather than failing keeps the spike honest about what it measures.
mapfile -t FIELDS < <(srcpsql "select id from checklist_fields where type='text' order by id limit 4")
if [ "${#FIELDS[@]}" -lt 4 ]; then
  mapfile -t FIELDS < <(srcpsql "select id from checklist_fields order by id limit 4")
fi
[ "${#FIELDS[@]}" -ge 4 ] || cannot_run "fewer than 4 checklist_fields rows exist (${#FIELDS[@]}) — config/templates.yaml did not seed enough to distinguish two INSERTs, an UPDATE and a liveness control"
FIELD_A="${FIELDS[0]}"; FIELD_B="${FIELDS[1]}"; FIELD_C="${FIELDS[2]}"; FIELD_D="${FIELDS[3]}"
printf '  field A (dark INSERT)   %s — %s\n' "$FIELD_A" "$(srcpsql "select label||' ['||type||']' from checklist_fields where id='$FIELD_A'")"
printf '  field B (dark UPDATE)   %s — %s\n' "$FIELD_B" "$(srcpsql "select label||' ['||type||']' from checklist_fields where id='$FIELD_B'")"
printf '  field C (dark INSERT)   %s — %s\n' "$FIELD_C" "$(srcpsql "select label||' ['||type||']' from checklist_fields where id='$FIELD_C'")"
printf '  field D (liveness)      %s — %s\n' "$FIELD_D" "$(srcpsql "select label||' ['||type||']' from checklist_fields where id='$FIELD_D'")"

# --------------------------------------------------------------------------
step "WATERMARK CENSUS — what the real write path does to its timestamps"
# --------------------------------------------------------------------------
# The card asks for the submitted_at semantics as a FINDING, measured, not
# assumed. This is the schema-side half, taken from the database HQ's OWN migrator
# just built; the row-level half is measured end-to-end by the client leg (step
# 10 of rxdb/spike-e-reconnect.js) on rows this run actually wrote.
printf '  checklist_submissions.submitted_at : %s\n' \
  "$(srcpsql "select coalesce(column_default,'(none)') from information_schema.columns where table_name='checklist_submissions' and column_name='submitted_at'")"
printf '  submission_responses.answered_at   : %s\n' \
  "$(srcpsql "select coalesce(column_default,'(none)') from information_schema.columns where table_name='submission_responses' and column_name='answered_at'")"
SUBMIT_TRG="$(srcpsql "select count(*) from pg_trigger t join pg_class c on c.oid=t.tgrelid where c.relname='checklist_submissions' and not t.tgisinternal")"
RESP_TRG="$(srcpsql "select count(*) from pg_trigger t join pg_class c on c.oid=t.tgrelid where c.relname='submission_responses' and not t.tgisinternal")"
printf '  user triggers on checklist_submissions: %s   ·  on submission_responses: %s\n' "$SUBMIT_TRG" "$RESP_TRG"
printf '  => nothing re-stamps submitted_at on UPDATE. Its only writer is the INSERT default;\n'
printf '     repository.go:1186 (approve) and :1232 (reject) set status/reviewed_by/reviewed_at\n'
printf '     and leave submitted_at alone. answered_at IS re-stamped, explicitly, by\n'
printf '     repository.go:829 (ON CONFLICT ... DO UPDATE SET value = EXCLUDED.value, answered_at = now()).\n'

# --------------------------------------------------------------------------
step "substrate prerequisite — the live grant projection for this user"
# --------------------------------------------------------------------------
# hq_sync_checklists' RLS has TWO axes: owner_id vs the token's sub (identity) and
# app_slug vs hq_grant_projection (live entitlement). Without a projection row the
# relay's write would land and the RxDB read would still see nothing — a distinct
# and much nastier failure than "never arrived".
PROJ_CODE="$(curl -sS -o "$WORK/proj.json" -w '%{http_code}' -X POST \
  -H "Authorization: Bearer $SERVICE_TOKEN" -H 'Content-Type: application/json' \
  -H 'Prefer: resolution=merge-duplicates,return=minimal' \
  -d "[{\"user_id\":\"$HQ_USER_ID\",\"app_slug\":\"$APP_SLUG\"}]" \
  "$REST_BASE/$PROJ_TABLE")"
case "$PROJ_CODE" in
  2*) echo "  $PROJ_TABLE += ($HQ_USER_ID, $APP_SLUG) -> HTTP $PROJ_CODE" ;;
  *)  cat "$WORK/proj.json"; cannot_run "seeding $PROJ_TABLE returned HTTP $PROJ_CODE" ;;
esac

# --------------------------------------------------------------------------
step "the CARRIER — spike C's LISTEN/NOTIFY relay, armed on BOTH paths"
# --------------------------------------------------------------------------
# 🛑 The relay is NOT what this card removes for its red. Spike C already proved
# it. It is armed identically on the green and the red run so the only difference
# between them is whether the CHECKPOINT PULL happens on reconnect.
echo "  applying sql/spike-c-relay-trigger.sql (spike C's, unchanged) to the scratch HQ Postgres"
# ON_ERROR_STOP=1 is load-bearing: without it psql reports success having skipped a
# failed statement — the schema-shaped silent no-op.
docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -q -f - \
  < "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" \
  || cannot_run "applying sql/spike-c-relay-trigger.sql failed"
srcpsql "select 1 from pg_trigger where tgname='spike_c_relay_notify'" | grep -q 1 \
  || cannot_run "the trigger was applied but pg_trigger does not list spike_c_relay_notify"
echo "  trigger present: spike_c_relay_notify AFTER INSERT OR UPDATE ON submission_responses"

echo "  building and starting the relay (backend/cmd/spikec-relay, unchanged)"
( cd "$REPO_ROOT/backend" && go build -o "$WORK/spikec-relay" ./cmd/spikec-relay ) \
  || cannot_run "'go build ./cmd/spikec-relay' failed"
env SPIKE_C_HQ_DSN="$HQ_DSN" \
  SPIKE_C_REST_BASE="$REST_BASE" \
  SPIKE_C_SERVICE_TOKEN="$SERVICE_TOKEN" \
  SPIKE_C_SYNC_TABLE="$SYNC_TABLE" \
  SPIKE_C_APP_SLUG="$APP_SLUG" \
  "$WORK/spikec-relay" > "$WORK/relay.log" 2>&1 &
RELAY_PID=$!

# 🛑 Wait for the relay's OWN readiness line, not for "the process exists". A relay
# that has not finished issuing LISTEN drops the notifications it was started for.
deadline=$(( $(date +%s) + 30 ))
until grep -q SPIKE_C_RELAY_READY "$WORK/relay.log" 2>/dev/null; do
  kill -0 "$RELAY_PID" 2>/dev/null || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay exited before announcing readiness"; }
  [ "$(date +%s)" -lt "$deadline" ] || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay never announced readiness within 30s"; }
  sleep 1
done
echo "  relay READY (LISTEN on spike_c_relay established, pid $RELAY_PID)"

# --------------------------------------------------------------------------
step "THE CYCLE — subscribe · observe · SEVER · write while dark · reconnect · measure"
# --------------------------------------------------------------------------
set +e
( cd "$SPIKE_DIR/rxdb" && \
  SPIKE_E_API_BASE="$API" \
  SPIKE_E_SESSION="$SESSION" \
  SPIKE_E_USER_ID="$HQ_USER_ID" \
  SPIKE_E_FIELD_A="$FIELD_A" \
  SPIKE_E_FIELD_B="$FIELD_B" \
  SPIKE_E_FIELD_C="$FIELD_C" \
  SPIKE_E_FIELD_D="$FIELD_D" \
  SPIKE_E_SYNC_TABLE="$SYNC_TABLE" \
  SPIKE_E_DEADLINE_MS="$DEADLINE_MS" \
  SPIKE_E_RUN_ID="$RUN_ID" \
  SPIKE_E_NO_PULL="$NO_PULL" \
  node spike-e-reconnect.js )
CLIENT_RC=$?
set -e

# --------------------------------------------------------------------------
step "corroboration in HQ's OWN Postgres — the writes really were an INSERT and an UPDATE"
# --------------------------------------------------------------------------
# Without this the client leg's "UPDATE to an already-held row" rests on the
# substrate id staying the same. This proves it at the SOURCE: HQ's upsert
# (repository.go:826, ON CONFLICT (field_id, answered_by) WHERE submission_id IS
# NULL) must have left field B with exactly ONE row, whose answered_at moved.
B_ROWS="$(srcpsql "select count(*) from submission_responses where field_id='$FIELD_B' and submission_id is null")"
A_ROWS="$(srcpsql "select count(*) from submission_responses where field_id='$FIELD_A' and submission_id is null")"
C_ROWS="$(srcpsql "select count(*) from submission_responses where field_id='$FIELD_C' and submission_id is null")"
printf '  submission_responses draft rows — field A: %s · field B: %s · field C: %s\n' "$A_ROWS" "$B_ROWS" "$C_ROWS"
printf '  field B row: %s\n' "$(srcpsql "select id||'  answered_at='||answered_at||'  value='||value::text from submission_responses where field_id='$FIELD_B' and submission_id is null")"
if [ -f "$WORK/relay.log" ]; then
  echo "  --- relay.log (tail) ---"
  tail -n 30 "$WORK/relay.log" | sed 's/^/  | /'
fi

case "$CLIENT_RC" in
  0)
    [ "$B_ROWS" = "1" ] || red "the client leg reported full recovery but field B has $B_ROWS draft rows in HQ's Postgres — the second write was not an UPDATE in place, so the mandatory UPDATE case was NOT exercised and this green would be vacuous"
    ;;
  1)
    red "a replicating RxDB client that was severed while rows changed did NOT recover everything on reconnect. $([ "$NO_PULL" = 1 ] && echo 'This is the expected RED-FIRST capture: the checkpoint pull leg was disabled and recovery was realtime-only.' || echo 'Checkpoint pull was ARMED and still missed dark-window changes — the build cards need an explicit resync step.')"
    ;;
  2)  cannot_run "the client leg could not run — see its own output above. This is NOT a verdict." ;;
  *)  cannot_run "the client leg exited $CLIENT_RC, which is outside its documented contract (0/1/2)" ;;
esac

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — reconnect catch-up recovers everything.\n'
printf '   A RUNNING RxDB client was OBSERVED replicating, then SEVERED (replication\n'
printf '   cancelled, channels removed, Realtime socket disconnected). While it was\n'
printf '   dark, 3 changes went through POST /api/v1/workflow/saveResponse — 2 INSERTs\n'
printf '   and 1 UPDATE to a row the client ALREADY HELD. Dark-window silence was\n'
printf '   VERIFIED (the collection was byte-identical while the substrate moved on).\n'
printf '   On reconnect, checkpoint pull delivered ALL THREE within %s ms, and the\n' "$DEADLINE_MS"
printf '   UPDATE case was exercised and recovered IN PLACE (same primary key, new body,\n'
printf '   corroborated by field B holding exactly 1 draft row in HQ Postgres).\n'
printf '   hq postgres: project %s, container %s, host port %s (ephemeral)\n' "$HQ_PROJECT" "${HQ_CID:0:12}" "$HQ_PORT"
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
