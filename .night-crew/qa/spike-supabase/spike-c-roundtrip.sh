#!/usr/bin/env bash
# spike-c-roundtrip.sh — SPIKE C. Night-crew card C `spike-c-round-trip`.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   GREEN. One row written through HQ's REAL write path
#            (POST /api/v1/workflow/saveResponse, real session cookie, real auth
#            middleware, real grant gate, real repository SQL, against a Postgres
#            carrying HQ's REAL migrations) reached a RUNNING RxDB client's
#            collection within the stated bound, carrying HQ's real user uuid and
#            the exact field_id and value the write path was given.
#
#   exit 1   RED — ran, and the mechanism is DISPROVEN. The row did not arrive
#            within the bound, or arrived carrying the wrong thing.
#            🛑 A RED VERDICT IS A SUCCESSFUL SPIKE. This card exists to find out
#            whether the round trip closes, not to make it close. If a leg reds:
#            record it with the captured output and STOP. Debugging the harness
#            is legitimate. Rewriting the goal so it passes is not.
#
#   exit 2   COULD NOT RUN. A setup/infrastructure failure — Docker down, the
#            substrate would not come up, the scratch Postgres never became
#            healthy, HQ's migrator failed, login failed, /saveResponse did not
#            return 204. 🛑 THIS IS NOT A VERDICT. It says nothing about the
#            mechanism and must never be reported as red.
#
#   exit 3   A verdict was reached BUT spike A's shared substrate could not be
#            restored. Migrated rows are still in hq_sync_checklists /
#            hq_grant_projection and WILL red backend/internal/sync's
#            TestJWTBridgeRLS, whose service_role CONTROL asserts an EXACT
#            full-table row set. Repair before trusting anything.
#
#   exit 64  usage error.
#
# There is deliberately no "warn and continue" anywhere in this file and no
# advisory leg. A step that cannot decide is a FAILURE. That is spike A's rule
# (env-up.sh:18-27) carried forward through spike B unchanged; if you are about
# to add `|| true` to an assertion you are about to destroy the only thing this
# script is for.
# ═══════════════════════════════════════════════════════════════════════════
#
# ───────────────────────────────────────────────────────────────────────────
# THE MECHANISM QUESTION, AND HOW THE CANDIDATE WAS CHOSEN
#
# The card's job is to establish whether the HQ-Postgres -> substrate -> RxDB-read
# path exists AT ALL, and BY WHAT MECHANISM. Decision 126 measured the "rows flow
# back from the substrate" premise FALSE on night nine of nine: RxDB replicates
# from a SECOND, DIFFERENT Postgres and the FDW bridge is one-directional and
# carries PERMISSIONS, NOT DATA. So the inventory came first.
#
# WHAT ALREADY EXISTS IN THE TREE (inventoried before anything was written):
#   * backend/internal/sync/listener.go — a live pgxlisten.Listener on
#     `ops_channel`, re-reading the op by id and fanning out to WebSocket
#     clients. HQ ALREADY RUNS A LISTEN/NOTIFY RELAY. It relays to browsers,
#     not to a substrate.
#   * backend/internal/sync/ops.go:204,304 — `pg_notify('ops_channel', ...)`
#     fired INSIDE the writing transaction.
#   * backend/internal/sync/hub.go — the fan-out target.
#   * migrations 0073/0074 — `sync_fdw_*` VIEWS. This is decision 126's bridge:
#     one-directional, permissions only.
#   * spike B's service-identity lane — measured viable, no schema change
#     (`service_role` already has rolbypassrls=t and full table grants).
#
# THE FOUR CANDIDATES, AND THEIR DISPOSITION:
#
#   (1) LISTEN/NOTIFY relay  ← CHOSEN AND PROVEN BY THIS SCRIPT
#       HQ Postgres --NOTIFY--> Go relay --PostgREST(service identity)--> substrate.
#       Chosen because every part of it already exists in the tree: the NOTIFY
#       idiom, the pgxlisten dependency (a DIRECT dependency of backend/go.mod
#       already), and spike B's proven service lane. It also leaves the write
#       path completely untouched, which is decision 126's shape.
#
#   (2) Polling relay
#       Same relay, watermark on answered_at instead of NOTIFY. Strictly weaker
#       on latency and it needs a reliable watermark; `answered_at` is
#       `now()`-stamped and collides within a transaction. It is the FALLBACK if
#       (1) reds, not the first thing to try — and it shares (1)'s entire
#       transform, so proving (1) proves most of (2).
#
#   (3) Logical replication  — DISPROVEN ON SHAPE, no run needed.
#       Postgres logical replication copies rows into an IDENTICALLY SHAPED
#       table. HQ's `submission_responses` (uuid keys, JSONB value, FK to
#       checklist_fields) and the sync contract (`text` PK, `_deleted`,
#       `_modified`, flat body) are not the same shape and cannot be made so
#       without the transform that spike B wrote in SQL. Logical replication has
#       no transform stage. It would move HQ's tables verbatim into the
#       substrate, which is a different (and much larger) proposal than
#       decision 126's.
#
#   (4) PostgREST forward-writer inside the HQ handler
#       Dual-write from `saveResponse` at request time. REJECTED, and the reason
#       is a product one, not a technical one: it puts the substrate on the
#       critical path of a crew member's checkbox. A substrate outage would red
#       the write. Decision 126's shape is that writes keep owning writes.
#
# 🛑 Choosing among these is the spike's OWN call and that finding is the
#    deliverable — it is explicitly NOT a park (see the card's PARK note).
# ───────────────────────────────────────────────────────────────────────────
#
# ⚠ CONTAINERS — the standing rule, absolute.
#   The HQ Postgres is a FRESH SCRATCH CONTAINER of its own, under the compose
#   project `spike-c-hq`, on a DOCKER-ASSIGNED EPHEMERAL host port.
#     * NEVER :5433. That cluster is PRODUCTION AND DEV BOTH — it serves
#       https://hq.yumyums.kitchen — and a probe against it destroyed the prod
#       database on 2026-08-06 (B-141/B-143, ledger decision 155). No command in
#       this file resolves any database name against :5433.
#     * NEVER :5434 (`yumyums-test-pg`) and none of its databases — the card's
#       isolation rule names it explicitly.
#     * NEVER 5432 (`infra-postgres-1`).
#   It is created at the top and DESTROYED at the bottom, so the script is
#   re-runnable from nothing. `--keep` suspends only the teardown.
#
# ⚠ SPIKE A'S STACK IS CONSUMED, NOT MODIFIED.
#   env-up.sh is called in its default RECONCILE mode — idempotent, brings up
#   what is missing, destroys nothing. Never `--fresh` by default: that does
#   `down --volumes` on the `spike-supabase` project and would eat another
#   session's running substrate. Pass --fresh-substrate to opt in deliberately.
#
# ⚠ THE SUBSTRATE IS RESTORED, AND THE RESTORE IS VERIFIED (B-148 residual).
#   B-148's residual finding is that spike B's harness RED path was never
#   re-rehearsed, so its recovery path is unproven. This script therefore does
#   NOT reuse rxdb/hq-reset.js. It snapshots the exact id set of
#   hq_sync_checklists and the exact pair set of hq_grant_projection BEFORE it
#   writes anything, removes what it added, and then ASSERTS the sets are
#   identical again. That assertion runs on every path — green, red and aborted
#   — so the recovery path is rehearsed every single run, including this one.
#
# USAGE
#   .night-crew/qa/spike-supabase/spike-c-roundtrip.sh
#   .night-crew/qa/spike-supabase/spike-c-roundtrip.sh --no-relay          # RED-FIRST: reproduce the pre-mechanism red
#   .night-crew/qa/spike-supabase/spike-c-roundtrip.sh --keep              # leave the scratch pg + server up
#   .night-crew/qa/spike-supabase/spike-c-roundtrip.sh --fresh-substrate   # also rebuild spike A's stack from nothing
#
#   or via the repo Taskfile:  task spike:roundtrip

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

HQ_COMPOSE="$SPIKE_DIR/docker-compose.hq-real.yml"
HQ_PROJECT="spike-c-hq"
HQ_DB_USER="hq"
HQ_DB_NAME="hq_real"
HQ_DB_PASS="9a1e5c7b3d206f84e5b90c1a7d43f68e"   # throwaway; see the compose banner

SYNC_TABLE="hq_sync_checklists"
PROJ_TABLE="hq_grant_projection"
APP_SLUG="operations"
SUPERADMIN_EMAIL="jamal@yumyums.kitchen"
SUPERADMIN_PASSWORD="test123"                    # backend/config/superadmins.yaml dev_password

# The HQ API port. Deliberately NOT 8089 (dev server), NOT 8199 (Playwright's
# default), NOT 4471 (this card's own Playwright gate port). A spike that
# silently attached to a dev server would be measuring the wrong database.
HQ_API_PORT="${SPIKE_C_API_PORT:-8471}"

# The bound. 20s is generous for a LISTEN/NOTIFY hop plus a PostgREST write plus
# a Realtime push — spike A measured pull convergence in the low hundreds of ms —
# and being generous is right: the question is whether the path EXISTS, and a
# tight bound would let a slow-but-working mechanism read as "does not exist".
DEADLINE_MS="${SPIKE_C_DEADLINE_MS:-20000}"

KEEP=0
NO_RELAY=0
SUBSTRATE_ARGS=()
for arg in "$@"; do
  case "$arg" in
    --keep)             KEEP=1 ;;
    --no-relay)         NO_RELAY=1 ;;
    --fresh-substrate)  SUBSTRATE_ARGS=(--fresh) ;;
    *) echo "usage: $(basename "$0") [--keep] [--no-relay] [--fresh-substrate]" >&2; exit 64 ;;
  esac
done

# Go is not on a non-interactive shell's PATH on this box, and this script needs
# it three times over (HQ's server, the relay, the JWT minter). Without this they
# die `go: not found` (exit 127) and it LOOKS like a substrate failure when it is
# a PATH failure. Same list env-up.sh and spike-b-migration.sh use.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
# 🛑 TWO failure verbs, and the distinction is the card's requirement.
#   cannot_run  -> exit 2. Infrastructure/setup. NOT a verdict.
#   red         -> exit 1. Ran, mechanism disproven. A successful spike.
cannot_run() { printf '\n🛑 COULD NOT RUN (not a verdict) — %s\n' "$1" >&2; exit 2; }
red()        { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }

# Anchor compose the way env-up.sh does, so the project directory is identical
# whether this runs from the main checkout or a worktree. This compose file has
# no bind mounts (deliberately — see its banner).
ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
HQDC=(docker compose -p "$HQ_PROJECT" --project-directory "$ANCHOR" -f "$HQ_COMPOSE")

RUN_ID="${SPIKE_C_RUN_ID:-c$(date -u +%Y%m%d%H%M%S)}"
export SPIKE_C_RUN_ID="$RUN_ID"
WORK="$(mktemp -d -t spike-c-XXXXXX)"

printf '# spike-c-roundtrip.sh — /saveResponse -> HQ Postgres -> relay -> substrate -> RxDB\n'
printf '# repo    %s\n' "$REPO_ROOT"
printf '# anchor  %s\n' "$ANCHOR"
printf '# run     %s\n' "$RUN_ID"
printf '# mode    %s\n' "$([ "$NO_RELAY" = 1 ] && echo 'NO-RELAY (red-first capture: the mechanism is deliberately absent)' || echo 'relay armed')"
printf '# started %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown. Registered BEFORE anything is created, so an abort between `up` and
# the first assertion still cleans up. It preserves the run's exit status —
# that status is the verdict — with exactly ONE exception: a FAILED substrate
# restore forces exit 3, because a run that leaves spike A's shared tables dirty
# has broken a committed Go suite whether or not its own assertions passed.
# --------------------------------------------------------------------------
RELAY_PID=""
SERVER_PID=""
BASELINE_SYNC=""
BASELINE_PROJ=""
SERVICE_TOKEN=""
REST_BASE=""

teardown() {
  local rc=$?
  set +e

  if [ -n "$RELAY_PID" ]; then
    kill "$RELAY_PID" 2>/dev/null
    wait "$RELAY_PID" 2>/dev/null
  fi
  if [ -n "$SERVER_PID" ]; then
    kill "$SERVER_PID" 2>/dev/null
    wait "$SERVER_PID" 2>/dev/null
  fi

  # ---- (a) restore spike A's SHARED substrate, and VERIFY the restore --------
  # 🛑 NOT TIDINESS. hq_sync_checklists is shared and
  # backend/internal/sync/jwtbridge_rls_test.go's service_role CONTROL asserts
  # an EXACT full-table row set. Rows left behind by a spike RED that committed
  # Go suite once already (spike B's own first G2 run, four subtests of
  # TestJWTBridgeRLS).
  if [ -n "$SERVICE_TOKEN" ] && [ -n "$REST_BASE" ] && [ -n "$BASELINE_SYNC" ]; then
    printf '\n── teardown (1/2): restoring spike A'"'"'s shared substrate ──\n'
    # Everything this spike writes into the shared table is prefixed `spikec-`
    # by backend/internal/sync/spikec_relay.go. Deleting by prefix cannot touch
    # spike A's fixture rows or spike B's, and the verification below is what
    # proves that claim rather than asserting it.
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
  #    script's status, so a failed substrate restore would be swallowed and a
  #    run that left spike A's tables dirty would still exit 0 — the silent
  #    no-op class this whole cycle exists to retire.
  exit $rc
}
HQ_USER_ID=""
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
[ -f "$SPIKE_DIR/rxdb/spike-c-read.js" ] || cannot_run "spike asset missing: rxdb/spike-c-read.js"
[ -f "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" ] || cannot_run "spike asset missing: sql/spike-c-relay-trigger.sql"
# NOTE: rxdb/node_modules is deliberately NOT asserted here. env-up.sh is what
# installs it, and it runs in the very next step — asserting it in preflight
# makes a first run on a clean machine fail for a reason the next step would
# have fixed. It is asserted after the substrate leg instead.

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
echo "  delegating to env-up.sh (${SUBSTRATE_ARGS[*]:-reconcile}); its exit status gates this leg"
"$SPIKE_DIR/env-up.sh" "${SUBSTRATE_ARGS[@]}" \
  || cannot_run "the substrate did not come up — env-up.sh returned non-zero. Its own output above names the leg."

[ -d "$SPIKE_DIR/rxdb/node_modules" ] || cannot_run "rxdb/node_modules is still missing after env-up.sh — the RxDB harness cannot run"

REST_PORT="$(docker compose -p spike-supabase --project-directory "$ANCHOR" \
  -f "$REPO_ROOT/docker-compose.supabase.yml" port rest 3000 2>/dev/null | sed 's/.*://')"
case "$REST_PORT" in
  ''|*[!0-9]*) cannot_run "could not resolve spike A's PostgREST host port (got '$REST_PORT')" ;;
esac
REST_BASE="http://127.0.0.1:$REST_PORT"
echo "  PostgREST: $REST_BASE"

# The throwaway JWT secret, committed on purpose in docker-compose.supabase.yml
# and mirrored in rxdb/spike-env.js. Read from spike A's compose file rather than
# re-typed, so a rotation there cannot leave this script signing with a stale
# secret and reporting a mechanism failure that is really a 401.
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"

# service_role, not a per-user token. Spike B measured that a bulk/relay lane
# CANNOT run on per-user tokens: hq_sync_checklists_insert's WITH CHECK refuses a
# row whose owner holds no live grant on its app.
SERVICE_TOKEN="$(cd "$SPIKE_DIR" && go run ./mintjwt -secret "$JWT_SECRET" -sub "spike-c-relay" -role service_role -ttl 60m)" \
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
# and could make the round trip look instantaneous.
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
# 🛑 This is the leg that separates spike C from spike B. Spike B transcribed 8
# tables by hand; this boots `backend/cmd/server`, whose `db.Migrate(pool)` runs
# every migration in backend/internal/db/migrations against the empty container.
# `/saveResponse` runs through real handler + real middleware + real repository
# SQL, so it must execute against the schema that code was written for.
echo "  building HQ's server binary"
( cd "$REPO_ROOT/backend" && go build -o "$WORK/hq-server" ./cmd/server ) \
  || cannot_run "'go build ./cmd/server' failed — this is a build failure, not a mechanism finding"

MIG_COUNT="$(find "$REPO_ROOT/backend/internal/db/migrations" -name '*.sql' | wc -l | tr -d ' ')"
echo "  booting it against the scratch Postgres (it will apply $MIG_COUNT migrations)"

# STATIC_DIR is set for the SAME reason playwright.config.js sets it: main.go
# computes `secureCookie := os.Getenv("STATIC_DIR") == ""`, so leaving it unset
# yields Secure cookies that curl and fetch will never send back over http, and
# every authenticated call would 401 for a reason that has nothing to do with
# this card. The blanked credential vars mirror playwright.config.js's webServer
# command verbatim (cross-contamination audit 2026-07-21): the root Taskfile's
# `dotenv: ['backend/.env']` injects LIVE Mercury/Anthropic/Cliq/SMTP credentials
# into anything launched from the checkout, and the alert queue is NOT gated by
# E2E_DISABLE_SCHEDULERS.
(
  cd "$REPO_ROOT/backend" && \
  PORT="$HQ_API_PORT" DB_URL="$HQ_DSN" STATIC_DIR=../ \
  SUPERADMIN_CONFIG=config/superadmins.yaml TEMPLATE_CONFIG=config/templates.yaml \
  TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
  MERCURY_API_KEY= ANTHROPIC_API_KEY= \
  ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
  SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= \
  "$WORK/hq-server" > "$WORK/hq-server.log" 2>&1
) &
SERVER_PID=$!

API="http://127.0.0.1:$HQ_API_PORT"
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
step "the write path's real prerequisites — a real session and a real field"
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

# A real field from the real seeded template. `text` preferred so the sentinel is
# a natural value for the field's own type; any field is acceptable because
# saveResponse does not type-check the value, and falling back rather than
# failing keeps the spike honest about what it is measuring.
FIELD_ID="$(srcpsql "select id from checklist_fields where type='text' order by id limit 1")"
[ -n "$FIELD_ID" ] || FIELD_ID="$(srcpsql "select id from checklist_fields order by id limit 1")"
[ -n "$FIELD_ID" ] || cannot_run "no checklist_fields rows — config/templates.yaml did not seed"
FIELD_LABEL="$(srcpsql "select label||' ['||type||']' from checklist_fields where id='$FIELD_ID'")"
echo "  field:   $FIELD_ID — $FIELD_LABEL"

# --------------------------------------------------------------------------
step "substrate prerequisite — the live grant projection for this user"
# --------------------------------------------------------------------------
# hq_sync_checklists' RLS has TWO axes: owner_id vs the token's sub (identity),
# and app_slug vs hq_grant_projection (live entitlement). Without a projection
# row the relay's write would land and the RxDB read would still see nothing —
# a distinct and much nastier failure than "never arrived". Writing it here (and
# removing it in teardown) keeps the two axes separable.
#
# 🛑 In production this table is written by the jwt-bridge/grant projection, not
# by a spike. This is a fixture, and it is the ONE substrate prerequisite this
# card creates.
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
step "THE MECHANISM — LISTEN/NOTIFY relay"
# --------------------------------------------------------------------------
if [ "$NO_RELAY" = "1" ]; then
  printf '  🛑 --no-relay: the trigger is NOT applied and the relay is NOT started.\n'
  printf '     This is the RED-FIRST capture. Everything on both sides of the gap is\n'
  printf '     real and live — HQ writes really land in HQ Postgres, and the RxDB client\n'
  printf '     really replicates from the substrate — and NOTHING bridges them.\n'
  printf '     The round trip below must FAIL, and must fail with exit 1, not exit 2.\n'
else
  echo "  applying sql/spike-c-relay-trigger.sql to the scratch HQ Postgres"
  # ON_ERROR_STOP=1 is load-bearing: without it psql reports success having
  # skipped a failed statement — the schema-shaped silent no-op.
  docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -q -f - \
    < "$SPIKE_DIR/sql/spike-c-relay-trigger.sql" \
    || cannot_run "applying sql/spike-c-relay-trigger.sql failed"
  srcpsql "select 1 from pg_trigger where tgname='spike_c_relay_notify'" | grep -q 1 \
    || cannot_run "the trigger was applied but pg_trigger does not list spike_c_relay_notify"
  echo "  trigger present: spike_c_relay_notify AFTER INSERT OR UPDATE ON submission_responses"

  echo "  building and starting the relay (backend/cmd/spikec-relay)"
  ( cd "$REPO_ROOT/backend" && go build -o "$WORK/spikec-relay" ./cmd/spikec-relay ) \
    || cannot_run "'go build ./cmd/spikec-relay' failed"
  (
    SPIKE_C_HQ_DSN="$HQ_DSN" \
    SPIKE_C_REST_BASE="$REST_BASE" \
    SPIKE_C_SERVICE_TOKEN="$SERVICE_TOKEN" \
    SPIKE_C_SYNC_TABLE="$SYNC_TABLE" \
    SPIKE_C_APP_SLUG="$APP_SLUG" \
    "$WORK/spikec-relay" > "$WORK/relay.log" 2>&1
  ) &
  RELAY_PID=$!

  # 🛑 Wait for the relay's OWN readiness line, not for "the process exists".
  # This spike makes exactly ONE write; a relay that has not finished issuing
  # LISTEN drops the only NOTIFY there will ever be, and the round trip would
  # red for a reason that is not the mechanism.
  deadline=$(( $(date +%s) + 30 ))
  until grep -q SPIKE_C_RELAY_READY "$WORK/relay.log" 2>/dev/null; do
    kill -0 "$RELAY_PID" 2>/dev/null || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay exited before announcing readiness"; }
    [ "$(date +%s)" -lt "$deadline" ] || { echo "--- relay.log ---"; cat "$WORK/relay.log"; cannot_run "the relay never announced readiness within 30s"; }
    sleep 1
  done
  echo "  relay READY (LISTEN on spike_c_relay established, pid $RELAY_PID)"
fi

# --------------------------------------------------------------------------
step "ROUND TRIP — /saveResponse -> ... -> a RUNNING RxDB client"
# --------------------------------------------------------------------------
SENTINEL="spikec-$RUN_ID-$(date +%s%N)"
set +e
( cd "$SPIKE_DIR/rxdb" && \
  SPIKE_C_API_BASE="$API" \
  SPIKE_C_SESSION="$SESSION" \
  SPIKE_C_FIELD_ID="$FIELD_ID" \
  SPIKE_C_USER_ID="$HQ_USER_ID" \
  SPIKE_C_SENTINEL="$SENTINEL" \
  SPIKE_C_SYNC_TABLE="$SYNC_TABLE" \
  SPIKE_C_DEADLINE_MS="$DEADLINE_MS" \
  SPIKE_C_RUN_ID="$RUN_ID" \
  node spike-c-read.js )
READ_RC=$?
set -e

# The row must ALSO really be in HQ's Postgres — otherwise a red could mean the
# write never happened rather than that nothing carried it. This check runs on
# BOTH paths, and on the red path it is what proves the red is the MECHANISM.
IN_HQ="$(srcpsql "select count(*) from submission_responses where value::text like '%$SENTINEL%'")"
printf '\n  HQ Postgres holds %s row(s) carrying the sentinel (the write path itself)\n' "$IN_HQ"
if [ -f "$WORK/relay.log" ]; then
  echo "  --- relay.log ---"
  sed 's/^/  | /' "$WORK/relay.log"
fi

case "$READ_RC" in
  0)
    [ "$IN_HQ" = "1" ] || red "the RxDB read passed but HQ's own Postgres holds $IN_HQ sentinel rows — the round trip cannot be green if the source row is not there"
    ;;
  1)
    [ "$IN_HQ" = "1" ] || cannot_run "the round trip failed AND the write never landed in HQ Postgres ($IN_HQ rows) — that is a harness failure, not a mechanism finding"
    red "the row written through POST /api/v1/workflow/saveResponse landed in HQ's Postgres and did NOT reach the RxDB-served read within ${DEADLINE_MS} ms. $([ "$NO_RELAY" = 1 ] && echo 'This is the expected RED-FIRST capture: no relay was running.' || echo 'The LISTEN/NOTIFY relay did not close the round trip — see relay.log above.')"
    ;;
  2)  cannot_run "the round-trip leg could not run — see its own output above. This is NOT a verdict." ;;
  *)  cannot_run "the round-trip leg exited $READ_RC, which is outside its documented contract (0/1/2)" ;;
esac

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — the round trip closes.\n'
printf '   POST /api/v1/workflow/saveResponse (real session, real auth, real grant\n'
printf '   gate, real repository SQL, HQ%s real migrations) -> HQ Postgres\n' "'"
printf '   -> NOTIFY spike_c_relay -> Go relay -> PostgREST (service identity)\n'
printf '   -> %s -> a RUNNING RxDB client, within %s ms.\n' "$SYNC_TABLE" "$DEADLINE_MS"
printf '   MECHANISM PROVEN: LISTEN/NOTIFY relay. See this file'"'"'s header for the\n'
printf '   three candidates that were not chosen and why.\n'
printf '   hq postgres: project %s, container %s, host port %s (ephemeral)\n' "$HQ_PROJECT" "${HQ_CID:0:12}" "$HQ_PORT"
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
