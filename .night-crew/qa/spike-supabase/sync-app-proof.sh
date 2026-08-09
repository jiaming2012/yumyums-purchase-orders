#!/usr/bin/env bash
# sync-app-proof.sh — PROVE the sync capability is USABLE IN THE APP, red-first.
# Card `sync-live-in-dev-app-proof` (Activity 5, roadmap leg 3, run 20260810).
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE. (GATE ON THE
#    SCRIPT, never on `task sync:app-proof` — go-task returns its own 201 on a
#    failing command, so red and could-not-run are indistinguishable at the
#    `task` boundary. B-163.)
#
#   exit 0   GREEN. The REAL production workflows.html — flag hq_sync_read=on,
#            NO page.route stub — driven in a REAL Chromium against the REAL HQ
#            server whose /sync/* proxy is wired (the 4 HQ_SYNC_* vars, Card 1's
#            leg 2) to the PERSISTENT Supabase substrate (Card 1's leg 1),
#            surfaced ONE field (written through HQ's REAL /saveResponse path) in
#            the app's own RxDB-served dev surface (#sync-one-row ->
#            data-state="served") within the bound, under the REAL per-user RLS
#            policy resolved through the substrate's FDW to HQ's live views.
#            AND — the RED-FIRST — the SAME served-asserting spec, run with the
#            carrier (the leg-1 relay stand-in) WITHHELD, did NOT reach `served`
#            (the app stayed `waiting`), so the green is not vacuous.
#
#            🛑 THE CARD'S DONE_WHEN, EXACTLY: "run the spec with the relay UP
#            (pass) and DOWN (fail)". This script does precisely that: it runs the
#            ONE served-asserting spec (app-proof/workflows-live.spec.js) TWICE —
#            carrier DOWN first (must FAIL, spec exit != 0), then carrier UP (must
#            PASS, spec exit 0). Both spec exit codes are captured and printed.
#
#   exit 1   RED — ran, and the integration is DISPROVEN. All setup went green
#            (proxy door open, FDW resolving, row carried into the substrate) and
#            the real browser still did not reach `served` within the bound.
#            🛑 A RED VERDICT IS A SUCCESSFUL PROOF of a defect: record it with
#            the captured console/network diagnostics and STOP.
#
#   exit 2   COULD NOT RUN. Setup/infra — Docker down, substrate would not
#            reconcile, scratch HQ never healthy, migrator failed, login failed,
#            /saveResponse did not 204, the /sync proxy answered 503 (leg-2 config
#            not effective), the FDW would not connect, OR the red-first control
#            PASSED (the app reached `served` with NO carrier — a vacuous
#            assertion the green cannot be trusted over). 🛑 NOT A VERDICT.
#
#   exit 3   A verdict was reached BUT the substrate could not be restored — the
#            FDW server options or the submission_responses row set were not put
#            back. These WILL red backend/internal/sync's RLS suites.
#
#   exit 64  usage error.
#
# ───────────────────────────────────────────────────────────────────────────
# WHY THIS IS THE CARD, AND WHAT IT PROMOTES
#
# This card promotes spike F (spike-f-browser-live.sh, which proved leg 3 GREEN)
# into a repo-owned, card-owned RED-FIRST harness. It reuses the PROVEN pieces —
# the app-proof/ spec+config (promoted from browser-live/), the scratch-HQ compose,
# the vendored RxDB, the repo-root Playwright, and Card 1's SHIPPED FDW SQL
# (sql/persistent-dev-fdw-pointing.sql). The novel claim it stands on is the READ
# PATH THROUGH THE PRODUCTION SURFACE against the substrate Card 1 makes persistent:
#     real browser -> same-origin /sync/* proxy -> substrate (PostgREST + Realtime)
#     -> real per-user RLS via FDW -> RxDB -> workflows.html renders it.
#
# ───────────────────────────────────────────────────────────────────────────
# B-164 — THE :5433 GUARD, AND THE SPIKE-F MODEL
#
# The card drives the app against the PERSISTENT substrate (Card 1's deliverable,
# stood up by `task sync:dev:up`). But an UNATTENDED run must NEVER point the FDW
# at the operator's live dev HQ (:5433 = dev AND prod; a probe there destroyed the
# prod database on 2026-08-06). So — exactly like sync-dev-proof.sh — this harness
# uses the SPIKE-F MODEL: a FRESH scratch HQ Postgres booted by HQ's own binary
# (real migrations) on a Docker-ASSIGNED EPHEMERAL port, the substrate consumed in
# RECONCILE mode (env-up.sh — never destroyed), and a snapshot->verify->restore of
# the ONE substrate mutation it makes (repoint the FDW server hq_pg at the scratch
# HQ, restored in teardown). The MECHANISM is byte-identical to what sync-dev-up.sh
# wires for the operator's live dev HQ; only the HQ coordinate differs (ephemeral
# scratch here, live dev there). NEVER 5432/5433/5434, asserted before + after.
#
# The carrier that lands the written row in the substrate is a STAND-IN for Card
# 1's persistent relay (LISTEN/NOTIFY -> PostgREST, proven GREEN in spikes C/D/E
# and re-proven by sync-dev-proof.sh). Withholding it IS "the relay is stopped".
#
# USAGE
#   .night-crew/qa/spike-supabase/sync-app-proof.sh
#   .night-crew/qa/spike-supabase/sync-app-proof.sh --keep
#   .night-crew/qa/spike-supabase/sync-app-proof.sh --fresh-substrate
#
#   or via the repo Taskfile:  task sync:app-proof   (GATE ON THE SCRIPT, not task)

set -euo pipefail

SPIKE_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

HQ_COMPOSE="$SPIKE_DIR/docker-compose.hq-real.yml"     # reuse spike-c/f's scratch-HQ compose
HQ_PROJECT="sync-app-proof-hq"                          # its OWN project — disjoint from spike-c/f/sync-dev-proof
HQ_DB_USER="hq"
HQ_DB_NAME="hq_real"
HQ_DB_PASS="9a1e5c7b3d206f84e5b90c1a7d43f68e"          # throwaway; docker-compose.hq-real.yml banner

SYNC_TABLE="submission_responses"                      # the PRODUCTION table `responses` reads (same as spike F)
FDW_SERVER="hq_pg"
FDW_ROLE="hq_sync_fdw"
FDW_ROLE_PASS="b2-rowvis-suite-throwaway"              # MUST match the substrate's existing user mapping
REALTIME_HOST="realtime-dev.localhost"
SUPERADMIN_EMAIL="jamal@yumyums.kitchen"
SUPERADMIN_PASSWORD="test123"                          # backend/config/superadmins.yaml dev_password

HQ_API_PORT="${HQ_APP_PROOF_API_PORT:-8474}"           # NOT 8471/8472 (spike C/F), NOT 8473 (sync-dev-proof), NOT 8089/8199 (dev/PW)
DEADLINE_MS="${HQ_APP_PROOF_DEADLINE_MS:-20000}"

KEEP=0
SUBSTRATE_ARGS=()
for arg in "$@"; do
  case "$arg" in
    --keep)             KEEP=1 ;;
    --fresh-substrate)  SUBSTRATE_ARGS=(--fresh) ;;
    *) echo "usage: $(basename "$0") [--keep] [--fresh-substrate]" >&2; exit 64 ;;
  esac
done

# Go is not on a non-interactive shell's PATH on this box; this script needs it
# for HQ's server build. Without it: `go: not found` (exit 127) LOOKS like a
# substrate failure. Same list env-up.sh / the spikes use.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
cannot_run() { printf '\n🛑 COULD NOT RUN (not a verdict) — %s\n' "$1" >&2; exit 2; }
red()        { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }

# Anchor compose the way env-up.sh does, so the project directory is identical
# from the main checkout or a worktree.
ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
HQDC=(docker compose -p "$HQ_PROJECT" --project-directory "$ANCHOR" -f "$HQ_COMPOSE")
SUBDC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")

RUN_ID="${HQ_APP_PROOF_RUN_ID:-ap$(date -u +%Y%m%d%H%M%S)}"
WORK="$(mktemp -d -t sync-app-proof-XXXXXX)"
SENTINEL="appproof-$RUN_ID-$(date +%s%N)"

printf '# sync-app-proof.sh — real workflows.html -> /sync proxy -> persistent substrate -> RxDB\n'
printf '# repo    %s\n# run     %s\n# started %s\n' "$REPO_ROOT" "$RUN_ID" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown — registered BEFORE anything is created. Restores the substrate.
# --------------------------------------------------------------------------
SERVER_PID=""
SUB_DB_CID=""
FDW_REPOINTED=0
ORIG_FDW_HOST=""; ORIG_FDW_PORT=""; ORIG_FDW_DBNAME=""
BASELINE_RESP=""

subpsql() { docker exec -i "$SUB_DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -tAc "$1"; }

teardown() {
  local rc=$?
  set +e

  if [ -n "$SERVER_PID" ]; then kill "$SERVER_PID" 2>/dev/null; wait "$SERVER_PID" 2>/dev/null; fi

  # ---- (a) restore the substrate: remove carried rows, restore the FDW server --
  if [ -n "$SUB_DB_CID" ]; then
    printf '\n── teardown (1/3): removing carried appproof- rows from %s ──\n' "$SYNC_TABLE"
    subpsql "delete from public.$SYNC_TABLE where id like 'appproof-%'" >/dev/null 2>&1
    if [ -n "$BASELINE_RESP" ]; then
      local after
      after="$(subpsql "select coalesce(string_agg(id, ',' order by id), '') from public.$SYNC_TABLE" 2>/dev/null)"
      if [ "$after" = "$BASELINE_RESP" ]; then
        printf '  VERIFIED: %s id-set is byte-identical to the pre-run baseline.\n' "$SYNC_TABLE"
      else
        printf '  🛑 %s DID NOT RETURN TO BASELINE.\n     baseline: %s\n     after   : %s\n' "$SYNC_TABLE" "$BASELINE_RESP" "$after"
        rc=3
      fi
    fi

    if [ "$FDW_REPOINTED" = "1" ] && [ -n "$ORIG_FDW_HOST" ]; then
      printf '\n── teardown (2/3): restoring FDW server %s -> %s:%s/%s ──\n' "$FDW_SERVER" "$ORIG_FDW_HOST" "$ORIG_FDW_PORT" "$ORIG_FDW_DBNAME"
      subpsql "alter server $FDW_SERVER options (set host '$ORIG_FDW_HOST', set port '$ORIG_FDW_PORT', set dbname '$ORIG_FDW_DBNAME')" >/dev/null 2>&1 \
        || { printf '  🛑 could NOT restore the FDW server options — repair by hand.\n'; rc=3; }
      subpsql "select postgres_fdw_disconnect_all()" >/dev/null 2>&1
      printf '  FDW options now: %s\n' "$(subpsql "select string_agg(o,'|' order by o) from pg_foreign_server, unnest(srvoptions) o where srvname='$FDW_SERVER'" 2>/dev/null)"
    fi
  else
    printf '\n── teardown (1/3): substrate was never reached; nothing to restore ──\n'
  fi

  # ---- (b) destroy the scratch HQ Postgres ---------------------------------
  if [ "$KEEP" = "1" ]; then
    printf '\n(--keep) scratch HQ Postgres LEFT RUNNING (project %s)\n' "$HQ_PROJECT"
  else
    printf '\n── teardown (3/3): destroying the scratch HQ Postgres (project %s) ──\n' "$HQ_PROJECT"
    "${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || \
      printf '  ⚠ teardown of project %s did not complete cleanly — check `docker ps -a`\n' "$HQ_PROJECT"
    printf '  torn down.\n'
  fi
  rm -rf "$WORK"

  # `exit`, NOT `return` — inside an EXIT trap a `return` cannot change status.
  exit $rc
}
trap teardown EXIT

# --------------------------------------------------------------------------
step "preflight — required tooling"
# --------------------------------------------------------------------------
for bin in docker curl node go npx; do
  command -v "$bin" >/dev/null 2>&1 || cannot_run "required tool not on PATH: $bin"
  printf '  %-6s %s\n' "$bin" "$(command -v "$bin")"
done
docker compose version >/dev/null 2>&1 || cannot_run "Compose v2 unavailable — needs 'docker compose'"
docker info >/dev/null 2>&1 || cannot_run "the Docker daemon is not reachable — 'docker info' failed"
[ -f "$HQ_COMPOSE" ] || cannot_run "compose file missing: $HQ_COMPOSE"
[ -f "$SPIKE_DIR/app-proof/workflows-live.spec.js" ] || cannot_run "card asset missing: app-proof/workflows-live.spec.js"
[ -f "$SPIKE_DIR/app-proof/playwright.app-proof.config.js" ] || cannot_run "card asset missing: app-proof/playwright.app-proof.config.js"
[ -f "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" ] || cannot_run "Card 1 asset missing: sql/persistent-dev-fdw-pointing.sql"
[ -f "$REPO_ROOT/workflows.html" ] || cannot_run "workflows.html missing at repo root — the server would have nothing to serve"
node -e "require.resolve('@playwright/test')" 2>/dev/null || cannot_run "@playwright/test does not resolve from the repo root — run 'npm ci' at the repo root first"
# 🛑 Resolve the Playwright CLI DETERMINISTICALLY, not via bare `npx playwright`.
# A bare `npx playwright test` resolves `playwright` off PATH, and this box carries
# a FOREIGN playwright (a Python one at /Users/jamal/miniconda3/bin/playwright) with
# NO `test` subcommand. When node_modules/.bin/playwright is missing (npm ci can
# skip bin-linking on its "Exit handler never called" internal error — measured on
# this worktree 2026-08-09), npx falls through to that foreign binary and prints
# `error: unknown command 'test'` for BOTH the red-first and the armed run — a
# could-not-run that MASQUERADES as a red-first pass + armed red. Prefer the repo's
# own CLI (@playwright/test/cli.js) so the verdict cannot be corrupted by PATH.
PW_CLI="$REPO_ROOT/node_modules/@playwright/test/cli.js"
[ -f "$PW_CLI" ] || PW_CLI="$REPO_ROOT/node_modules/playwright/cli.js"
[ -f "$PW_CLI" ] || cannot_run "the Playwright CLI (node_modules/@playwright/test/cli.js) is not present — run 'npm ci' (and if it printed 'Exit handler never called', 'npm rebuild @playwright/test') at the repo root first"
node "$PW_CLI" test --version >/dev/null 2>&1 || cannot_run "the resolved Playwright CLI ($PW_CLI) has no 'test' subcommand — a foreign playwright shadowed it or the install is incomplete"
echo "  playwright cli: node $PW_CLI ($(node "$PW_CLI" test --version 2>/dev/null))"
if grep -Eq '^[[:space:]]*-[[:space:]]*"?(5432|5433|5434):' "$HQ_COMPOSE"; then
  cannot_run "docker-compose.hq-real.yml publishes a FIXED host port in 5432-5434. This proof must use a Docker-assigned ephemeral port."
fi
echo "  isolation: hq-real compose publishes no fixed host port (Docker-assigned only)"

# --------------------------------------------------------------------------
step "substrate — Card 1's persistent Supabase substrate (RECONCILE, never destroy)"
# --------------------------------------------------------------------------
"$SPIKE_DIR/env-up.sh" "${SUBSTRATE_ARGS[@]}" \
  || cannot_run "the substrate did not come up — env-up.sh returned non-zero (its output names the leg)."

SUB_DB_CID="$("${SUBDC[@]}" ps -q db 2>/dev/null)"
[ -n "$SUB_DB_CID" ] || cannot_run "could not resolve the substrate db container (docker compose -p spike-supabase ps -q db)"

REST_PORT="$("${SUBDC[@]}" port rest 3000 2>/dev/null | sed 's/.*://')"
RT_PORT="$("${SUBDC[@]}" port realtime 4000 2>/dev/null | sed 's/.*://')"
case "$REST_PORT" in ''|*[!0-9]*) cannot_run "could not resolve the substrate PostgREST host port (got '$REST_PORT')" ;; esac
case "$RT_PORT" in ''|*[!0-9]*) cannot_run "could not resolve the substrate Realtime host port (got '$RT_PORT')" ;; esac
REST_BASE="http://127.0.0.1:$REST_PORT"
RT_BASE="http://127.0.0.1:$RT_PORT"
printf '  PostgREST %s  ·  Realtime %s\n' "$REST_BASE" "$RT_BASE"

JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"
echo "  substrate JWT secret read from compose (the /sync proxy signs with the same one)"

# --------------------------------------------------------------------------
step "substrate baseline + FDW snapshot (so the restore can be VERIFIED, and the red-first is not confounded)"
# --------------------------------------------------------------------------
BASELINE_RESP="$(subpsql "select coalesce(string_agg(id, ',' order by id), '') from public.$SYNC_TABLE")" \
  || cannot_run "could not read the $SYNC_TABLE baseline from the substrate"
case "$BASELINE_RESP" in
  *appproof-*) cannot_run "the substrate already holds appproof- rows from an aborted run. Remove them: docker exec $SUB_DB_CID psql -U supabase_admin -d postgres -c \"delete from public.$SYNC_TABLE where id like 'appproof-%'\"" ;;
esac
printf '  %s baseline holds %s row(s)\n' "$SYNC_TABLE" "$(subpsql "select count(*) from public.$SYNC_TABLE")"

# Snapshot the FDW server options BEFORE repointing, so teardown restores exactly.
FDW_OPTS="$(subpsql "select string_agg(o,'|' order by o) from pg_foreign_server, unnest(srvoptions) o where srvname='$FDW_SERVER'")" \
  || cannot_run "could not read the $FDW_SERVER FDW options"
[ -n "$FDW_OPTS" ] || cannot_run "the FDW server $FDW_SERVER has no options — is 0002_hq_fdw.sql applied to the substrate?"
ORIG_FDW_HOST="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^host=//p')"
ORIG_FDW_PORT="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^port=//p')"
ORIG_FDW_DBNAME="$(printf '%s' "$FDW_OPTS" | tr '|' '\n' | sed -n 's/^dbname=//p')"
[ -n "$ORIG_FDW_HOST" ] && [ -n "$ORIG_FDW_PORT" ] && [ -n "$ORIG_FDW_DBNAME" ] \
  || cannot_run "could not parse host/port/dbname out of the FDW options: $FDW_OPTS"
printf '  FDW %s currently -> %s:%s/%s (snapshotted for restore)\n' "$FDW_SERVER" "$ORIG_FDW_HOST" "$ORIG_FDW_PORT" "$ORIG_FDW_DBNAME"

# --------------------------------------------------------------------------
step "HQ Postgres — fresh scratch container (project $HQ_PROJECT), EMPTY, ephemeral port"
# --------------------------------------------------------------------------
"${HQDC[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
"${HQDC[@]}" up -d || cannot_run "'docker compose up -d' failed for project $HQ_PROJECT"
HQ_CID="$("${HQDC[@]}" ps -q hqreal || true)"
[ -n "$HQ_CID" ] || cannot_run "the hqreal service has no container — 'docker compose -p $HQ_PROJECT ps -a' will say why"

deadline=$(( $(date +%s) + 120 ))
while :; do
  state="$(docker inspect -f '{{.State.Status}}' "$HQ_CID" 2>/dev/null || echo missing)"
  health="$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}none{{end}}' "$HQ_CID" 2>/dev/null || echo none)"
  [ "$health" = "healthy" ] && { echo "  hqreal: $state/$health"; break; }
  case "$state" in exited|dead|missing) cannot_run "hqreal container is '$state' — 'docker logs $HQ_CID' will say why" ;; esac
  [ "$(date +%s)" -lt "$deadline" ] || cannot_run "hqreal never became healthy within 120s (last state=$state health=$health)"
  sleep 2
done

HQ_PORT="$("${HQDC[@]}" port hqreal 5432 2>/dev/null | sed 's/.*://' || true)"
case "$HQ_PORT" in ''|*[!0-9]*) cannot_run "could not resolve the Docker-assigned host port for hqreal:5432 (got '$HQ_PORT')" ;; esac
case "$HQ_PORT" in 5432|5433|5434) cannot_run "Docker assigned protected host port $HQ_PORT to the scratch Postgres. Re-run; the ephemeral range will pick another." ;; esac
HQ_DSN="postgres://$HQ_DB_USER:$HQ_DB_PASS@127.0.0.1:$HQ_PORT/$HQ_DB_NAME?sslmode=disable"
printf '  container %s  ·  host port %s (Docker-assigned, ephemeral, not 5432/5433/5434)\n' "${HQ_CID:0:12}" "$HQ_PORT"
srcpsql() { docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -t -A -c "$1"; }

# --------------------------------------------------------------------------
step "HQ's REAL schema — applied by HQ's OWN binary, and the /sync proxy WIRED (leg 2, Card 1's 4 vars)"
# --------------------------------------------------------------------------
echo "  building HQ's server binary"
( cd "$REPO_ROOT/backend" && go build -o "$WORK/hq-server" ./cmd/server ) \
  || cannot_run "'go build ./cmd/server' failed — a build failure, not a mechanism finding"

API="http://127.0.0.1:$HQ_API_PORT"
if curl -fsS --max-time 2 "$API/api/v1/health" >/dev/null 2>&1; then
  cannot_run "something is ALREADY serving $API/api/v1/health — this proof must never attach to a server it did not start. Kill it or set HQ_APP_PROOF_API_PORT."
fi

# 🛑 The four HQ_SYNC_* vars (Card 1's leg 2) open the /sync proxy door.
# STATIC_DIR=$REPO_ROOT so the server serves the REAL workflows.html + sync-rxdb +
# vendor from this tree. The blanked credential vars mirror playwright.config.js's
# webServer (the root Taskfile's dotenv injects live Mercury/Anthropic/Cliq/SMTP
# creds otherwise). `env ... binary &`, NEVER `( ... ) &`.
env PORT="$HQ_API_PORT" DB_URL="$HQ_DSN" STATIC_DIR="$REPO_ROOT" \
  SUPERADMIN_CONFIG="$REPO_ROOT/backend/config/superadmins.yaml" \
  TEMPLATE_CONFIG="$REPO_ROOT/backend/config/templates.yaml" \
  TOAST_SYNC_INTERVAL=0 E2E_DISABLE_SCHEDULERS=1 \
  HQ_SYNC_REST_URL="$REST_BASE" \
  HQ_SYNC_REALTIME_URL="$RT_BASE" \
  HQ_SYNC_JWT_SECRET="$JWT_SECRET" \
  HQ_SYNC_REALTIME_HOST="$REALTIME_HOST" \
  MERCURY_API_KEY= ANTHROPIC_API_KEY= \
  ZOHO_CLIQ_CLIENT_ID= ZOHO_CLIQ_CLIENT_SECRET= ZOHO_CLIQ_REFRESH_TOKEN= \
  SMTP_ADDR= SMTP_USERNAME= SMTP_PASSWORD= \
  "$WORK/hq-server" > "$WORK/hq-server.log" 2>&1 &
SERVER_PID=$!

deadline=$(( $(date +%s) + 90 ))
until curl -fsS "$API/api/v1/health" >/dev/null 2>&1; do
  kill -0 "$SERVER_PID" 2>/dev/null || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ's server exited before health (migrations or boot failed)"; }
  [ "$(date +%s)" -lt "$deadline" ] || { echo "--- hq-server.log ---"; cat "$WORK/hq-server.log"; cannot_run "HQ's server never answered /api/v1/health within 90s"; }
  sleep 1
done
echo "  health: $(curl -fsS "$API/api/v1/health")"
TABLES="$(srcpsql "select count(*) from information_schema.tables where table_schema='public'")"
[ "$TABLES" -gt 30 ] || cannot_run "only $TABLES public tables — HQ's migrator did not run. See $WORK/hq-server.log"
srcpsql "select 1 from pg_views where viewname='hq_sync_field_templates'" | grep -q 1 \
  || cannot_run "hq_sync_field_templates view absent in the scratch HQ — migration 0073 did not run; the FDW would have nothing to resolve"

# --------------------------------------------------------------------------
step "wire the production RLS: repoint the FDW at THIS scratch HQ (Card 1's SHIPPED SQL, both halves)"
# --------------------------------------------------------------------------
# The scratch HQ carries 0073/0074's source views + the hq_sync_fdw NOLOGIN role.
# Apply Card 1's shipped persistent-dev-fdw-pointing.sql: HALF A (HQ side) gives
# hq_sync_fdw LOGIN with the password the substrate's user mapping already uses;
# HALF B (substrate side) repoints hq_pg at the scratch HQ. This is the SAME file
# sync:dev:up applies to the operator's live dev HQ — only the coordinate differs.
docker exec -i "$HQ_CID" psql -U "$HQ_DB_USER" -d "$HQ_DB_NAME" -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="host.docker.internal" -v hq_port="$HQ_PORT" -v hq_dbname="$HQ_DB_NAME" \
  -f - < "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || cannot_run "applying the FDW SQL (HALF A) to the scratch HQ failed"
subpsql "select 1" >/dev/null
docker exec -i "$SUB_DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q \
  -v fdw_password="$FDW_ROLE_PASS" -v hq_host="host.docker.internal" -v hq_port="$HQ_PORT" -v hq_dbname="$HQ_DB_NAME" \
  -f - < "$SPIKE_DIR/sql/persistent-dev-fdw-pointing.sql" \
  || cannot_run "applying the FDW SQL (HALF B) to the substrate failed"
FDW_REPOINTED=1
subpsql "select postgres_fdw_disconnect_all()" >/dev/null 2>&1
printf '  FDW %s -> host.docker.internal:%s/%s (via Card 1 shipped SQL)\n' "$FDW_SERVER" "$HQ_PORT" "$HQ_DB_NAME"

# The FDW MUST connect now, or the production RLS cannot resolve and the browser
# would red for a wiring reason, not a mechanism one. A connect failure here is
# setup (cannot_run); a successful connect returning 0 for our field is a finding.
FT_COUNT="$(subpsql "select count(*) from public.hq_field_templates" 2>"$WORK/fdw.err")" \
  || { echo "--- fdw.err ---"; cat "$WORK/fdw.err"; cannot_run "the FDW could not connect to the scratch HQ (see error above) — wiring, not mechanism"; }
echo "  FDW resolves: hq_field_templates has $FT_COUNT row(s) through the bridge"

# --------------------------------------------------------------------------
step "the write path's real prerequisites — a real session and a real field"
# --------------------------------------------------------------------------
HQ_USER_ID="$(srcpsql "select id from users where email='$SUPERADMIN_EMAIL'")"
[ -n "$HQ_USER_ID" ] || cannot_run "the superadmin from config/superadmins.yaml was not upserted into users"
echo "  hq user: $HQ_USER_ID ($SUPERADMIN_EMAIL)"

LOGIN_CODE="$(curl -sS -o "$WORK/login.json" -w '%{http_code}' -c "$WORK/cookies.txt" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$SUPERADMIN_EMAIL\",\"password\":\"$SUPERADMIN_PASSWORD\"}" "$API/api/v1/auth/login")"
[ "$LOGIN_CODE" = "200" ] || { cat "$WORK/login.json"; cannot_run "POST /api/v1/auth/login returned HTTP $LOGIN_CODE"; }
SESSION="$(awk '$6=="hq_session"{print $7}' "$WORK/cookies.txt" | tail -1)"
[ -n "$SESSION" ] || cannot_run "login succeeded but set no hq_session cookie"
echo "  session: real hq_session cookie from POST /api/v1/auth/login (HTTP 200)"

FIELD_ID="$(srcpsql "select id from checklist_fields where type='text' order by id limit 1")"
[ -n "$FIELD_ID" ] || FIELD_ID="$(srcpsql "select id from checklist_fields order by id limit 1")"
[ -n "$FIELD_ID" ] || cannot_run "no checklist_fields rows — config/templates.yaml did not seed"
TEMPLATE_ID="$(srcpsql "select template_id from hq_sync_field_templates where field_id='$FIELD_ID' limit 1")"
[ -n "$TEMPLATE_ID" ] || cannot_run "field $FIELD_ID resolves to no template via hq_sync_field_templates"
CHECKLIST_ID="$(srcpsql "select gen_random_uuid()")"   # a draft has no submission; any id satisfies the URL contract
echo "  field:    $FIELD_ID"
echo "  template: $TEMPLATE_ID"

# --------------------------------------------------------------------------
step "leg-2 door check — the /sync proxy must NOT answer 503 (Card 1's 4 vars are set)"
# --------------------------------------------------------------------------
PROXY_CODE="$(curl -sS -o "$WORK/proxy.out" -w '%{http_code}' -b "$WORK/cookies.txt" "$API/sync/rest/")"
case "$PROXY_CODE" in
  503) cat "$WORK/proxy.out"; cannot_run "the /sync proxy answered 503 — HQ_SYNC_* did not open the door (leg-2 config not effective)" ;;
  200) echo "  GET /sync/rest/ -> 200 (proxy door open, PostgREST reachable through it)" ;;
  *)   cat "$WORK/proxy.out"; cannot_run "GET /sync/rest/ returned HTTP $PROXY_CODE (expected 200) — proxy/substrate wiring incomplete" ;;
esac

# --------------------------------------------------------------------------
step "write ONE field through HQ's REAL /saveResponse path"
# --------------------------------------------------------------------------
SAVE_CODE="$(curl -sS -o "$WORK/save.out" -w '%{http_code}' -b "$WORK/cookies.txt" \
  -H 'Content-Type: application/json' \
  -d "{\"field_id\":\"$FIELD_ID\",\"value\":\"$SENTINEL\"}" "$API/api/v1/workflow/saveResponse")"
case "$SAVE_CODE" in 200|204) echo "  POST /api/v1/workflow/saveResponse -> $SAVE_CODE" ;; *) cat "$WORK/save.out"; cannot_run "POST /saveResponse returned HTTP $SAVE_CODE" ;; esac
HQ_VALUE="$(srcpsql "select value::text from submission_responses where field_id='$FIELD_ID' and answered_by='$HQ_USER_ID' and submission_id is null order by answered_at desc limit 1")"
[ -n "$HQ_VALUE" ] || cannot_run "the write did not land as a draft submission_responses row in HQ Postgres"
case "$HQ_VALUE" in *"$SENTINEL"*) : ;; *) cannot_run "the landed row's value does not carry the sentinel (got $HQ_VALUE)" ;; esac
echo "  HQ Postgres holds the draft row, value=$HQ_VALUE"

# The browser-drive closure. The SAME served-asserting spec every time; the only
# thing that changes between the red-first and armed runs is whether the carrier
# has landed the row in the substrate. Returns the spec's own exit code.
run_spec() {
  ( cd "$REPO_ROOT" && \
    HQ_APP_PROOF_HQ_URL="$API" HQ_APP_PROOF_SESSION="$SESSION" \
    HQ_APP_PROOF_CHECKLIST_ID="$CHECKLIST_ID" HQ_APP_PROOF_TEMPLATE_ID="$TEMPLATE_ID" \
    HQ_APP_PROOF_FIELD_ID="$FIELD_ID" HQ_APP_PROOF_USER_ID="$HQ_USER_ID" \
    HQ_APP_PROOF_SENTINEL="$SENTINEL" HQ_APP_PROOF_DEADLINE_MS="$DEADLINE_MS" \
    node "$PW_CLI" test -c "$SPIKE_DIR/app-proof/playwright.app-proof.config.js" )
}

# --------------------------------------------------------------------------
step "RED-FIRST — run the served-asserting spec with the relay/carrier DOWN (must FAIL)"
# --------------------------------------------------------------------------
printf '  the carrier (Card 1 relay stand-in) has NOT run. Everything else is real and\n'
printf '  live: the page really opens replication through the proxy. The SAME served-\n'
printf '  asserting spec must FAIL — the app stays `waiting`, never reaches `served`.\n'
set +e; run_spec; RED_RC=$?; set -e
printf '  RED-FIRST spec exit code: %s\n' "$RED_RC"
if [ "$RED_RC" -eq 0 ]; then
  cannot_run "the red-first control PASSED (exit 0) with NO row in the substrate — the app reached `served` with no carrier, so the armed assertion is VACUOUS. This is a could-not-run, never a green."
fi
echo "  ✅ red-first: the served-asserting spec FAILED (exit $RED_RC) with the relay/carrier down — the app did NOT reach served. Non-vacuous."

# --------------------------------------------------------------------------
step "THE CARRIER — project the exact /saveResponse row into the substrate submission_responses"
# --------------------------------------------------------------------------
# 🛑 A STAND-IN for Card 1's persistent relay (LISTEN/NOTIFY -> PostgREST), which
# spikes C/D/E + sync-dev-proof.sh already proved GREEN. Its only job is to land the
# real written row in the substrate so the browser read can be measured. Landing it
# = "the relay is UP".
subpsql "insert into public.$SYNC_TABLE (id, submission_id, field_id, value, answered_by) values ('appproof-$RUN_ID', null, '$FIELD_ID', '$HQ_VALUE'::jsonb, '$HQ_USER_ID')" >/dev/null \
  || cannot_run "the carrier could not write the row into the substrate $SYNC_TABLE"
echo "  carried: appproof-$RUN_ID -> $SYNC_TABLE (field_id=$FIELD_ID, value=$HQ_VALUE, answered_by=$HQ_USER_ID)"

# --------------------------------------------------------------------------
step "ARMED — run the SAME served-asserting spec with the relay/carrier UP (must PASS — THE VERDICT)"
# --------------------------------------------------------------------------
set +e; run_spec; ARMED_RC=$?; set -e
printf '  ARMED spec exit code: %s\n' "$ARMED_RC"
case "$ARMED_RC" in
  0) : ;;
  *) red "the real workflows.html dev surface did NOT reach data-state=served with the sentinel within ${DEADLINE_MS} ms, even though the row is in the substrate and the proxy door is open. See the playwright output above (console + /sync response codes) for whether the read stalled at the proxy, the RLS, or RxDB." ;;
esac

printf '\n══════════════════════════════════════════════════════════\n'
printf '✅ VERDICT: GREEN — the app shows the round trip (leg 3, in the app).\n'
printf '   real Chromium -> workflows.html?hq_sync_read=on (no page.route stub)\n'
printf '   -> same-origin /sync/* proxy -> persistent substrate (PostgREST + Realtime)\n'
printf '   -> per-user RLS via FDW -> RxDB -> #sync-one-row served, carrying the\n'
printf '   sentinel written through POST /api/v1/workflow/saveResponse, within %s ms.\n' "$DEADLINE_MS"
printf '   RED-FIRST: the SAME served-asserting spec with the relay DOWN exited %s\n' "$RED_RC"
printf '              (did NOT reach served); with the relay UP it exited 0. Non-vacuous.\n'
printf '   The 4 HQ_SYNC_* vars + FDW pointing are Card 1'"'"'s deliverable, consumed here.\n'
printf '   finished %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '══════════════════════════════════════════════════════════\n'
exit 0
