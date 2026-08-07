#!/usr/bin/env bash
# spike-d-realtime.sh — SPIKE D. Night-crew card D `spike-d-realtime-live`.
# Closes B-62. The 4th of D-KR1's four spike verdicts.
#
# ═══════════════════════════════════════════════════════════════════════════
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#
#   exit 0   GREEN. A LIVE self-hosted Realtime server HONOURS the replication
#            filter `sync-rxdb/client.js` emits: the in-scope row ARRIVED on
#            every filtered channel, the out-of-scope row DID NOT, and the same
#            out-of-scope row arrived on the unfiltered controls — so the
#            suppression is attributable to the filter and to nothing else.
#
#   exit 1   RED — ran, and the mechanism is DISPROVEN. The filter was ignored
#            (the out-of-scope row arrived anyway), or rejected (the filtered
#            subscription could not be established while the unfiltered ones
#            could), or over-broad-in-the-other-direction (the IN-SCOPE row was
#            dropped too).
#            🛑 A RED VERDICT IS THE CARD'S PAYOFF, NOT A FAILURE. B-62 exists
#            because a filter that is syntactically perfect and semantically
#            ignored passes the entire Playwright suite while B-42's unscoped
#            live leg stays open behind three files that say it is closed. If a
#            leg reds: record it with the captured output and STOP. Debugging
#            the harness is legitimate. Rewriting the goal so it passes is not.
#
#   exit 2   COULD NOT RUN. Setup/infrastructure failure — Docker down, spike
#            A's substrate would not reconcile, the fixture would not apply, the
#            websocket would not connect, or Realtime delivered NOTHING AT ALL
#            (in which case there is no filter question to answer).
#            🛑 THIS IS NOT A VERDICT. It says nothing about the mechanism and
#            must never be reported as red.
#
#   exit 3   A verdict was reached BUT spike A's shared substrate could not be
#            restored — this run's three tables are still present and/or the
#            `supabase_realtime` publication no longer matches its pre-run
#            membership. Repair before trusting anything.
#
#   exit 64  usage error.
#
# There is deliberately no "warn and continue" anywhere in this file and no
# advisory leg. A step that cannot decide is a FAILURE. That is spike A's rule
# (env-up.sh:18-27), carried through spikes B and C unchanged; if you are about
# to add `|| true` to an assertion you are about to destroy the only thing this
# script is for.
# ═══════════════════════════════════════════════════════════════════════════
#
# ───────────────────────────────────────────────────────────────────────────
# THE QUESTION, AND WHY THE CONFIG-LEVEL PROOF IS NOT AN ANSWER
#
# Card S1a (run 20260803) applied B-42 option (i): a single `column=op.value`
# clause injected into the vendored plugin's hard-coded `postgres_changes`
# binding, through a shim on `client.channel`. `tests/sync-rxdb-client.spec.js`
# [SCOPE-04] proves the clause is correct, is ONE clause, reaches the binding
# config under the right channel name, and is absent on `responses`.
#
# 🛑 EVERY ONE OF THOSE ASSERTIONS IS ABOUT THE OBJECT HANDED TO THE LIBRARY.
#
# Nothing anywhere connects to `spike-supabase-realtime-1`, publishes an
# out-of-scope row, and observes that it does not arrive. A filter that is
# syntactically perfect and SEMANTICALLY IGNORED — wrong column for the
# publication's REPLICA IDENTITY FULL payload, an operator Realtime's parser
# refuses, a version that silently drops unknown keys — would pass the whole
# suite. And because the filter is a NARROWING, an ignored filter leaves the
# system behaving EXACTLY as it did before the card. That is B-62.
#
# ───────────────────────────────────────────────────────────────────────────
# THE ASSERTION SET, AND WHY EACH LEG IS LOAD-BEARING
#
# Five channels on ONE websocket, so no difference between them can be a
# difference in timing:
#
#   f-gte   spike_d_submissions  filter = realtimeFilterFor('checklists', LIST)
#                                      = `submitted_at=gte.<since>`
#   f-eq    spike_d_submissions  filter = realtimeFilterFor('checklists', FILL)
#                                      = `id=eq.<in-scope id>`
#   f-in    spike_d_templates    filter = realtimeFilterFor('templates', LIST)
#                                      = `id=in.(<in-scope id>,<absent id>)`
#   u-sub   spike_d_submissions  NO filter  — same-table control
#   u-resp  spike_d_responses    NO filter  — the `responses` control B-62 names
#
# Those are ALL THREE clause shapes HQ emits (`gte`, `eq`, `in`) plus the one
# collection HQ deliberately leaves unfiltered. 🛑 THE FILTER STRINGS ARE NEVER
# TYPED INTO THIS SCRIPT — they are read at run time out of
# `sync-rxdb/client.js`'s own exported `realtimeFilterFor()`, so this spike
# cannot quietly go on proving a string production no longer emits, and the
# absence of a filter on `responses` is READ FROM THE CODE rather than assumed.
#
#   1. POSITIVE ARRIVAL — MANDATORY, and the card says so twice.
#      An ignored filter and a dead subscription look identical from the
#      suppression leg alone. Unless the in-scope row is OBSERVED ARRIVING on
#      every filtered channel, the negative leg is not evidence of anything.
#
#   2. SUPPRESSION — the out-of-scope row must not arrive on f-gte / f-eq /
#      f-in.
#
#   3. SAME-TABLE UNFILTERED CONTROL (u-sub) — the identical out-of-scope row
#      must ARRIVE here. Without it, "Realtime suppressed the row" and "the row
#      never reached Realtime" are indistinguishable, and the spike would be
#      measuring its own fixture.
#
#   4. THE `responses` CONTROL (u-resp) — BOTH rows must arrive. That is what
#      turns "we chose not to filter this collection" from a comment in
#      client.js into a measurement.
#
#   5. STRUCTURAL PARITY — the clone tables' column list, `relreplident` and
#      publication membership are compared against HQ's REAL sync tables and the
#      run is refused unless they match. See sql/spike-d-fixture.sql for why the
#      measurement runs on clones at all.
#
# `--no-filter` runs the identical assertion set with the clause NOT attached to
# the binding. That is the RED-FIRST capture (gate RF): the pre-B-42 world,
# which B-62 warns is indistinguishable from an ignored filter — so if this
# script cannot tell those two apart it is not evidence, and the red proves it
# can. Legs 1, 3 and 4 stay green there; leg 2 reds. Exit 1.
#
# ───────────────────────────────────────────────────────────────────────────
# ⚠ CONTAINERS — the standing rule, absolute.
#   This script creates NO container. It consumes spike A's stack in RECONCILE
#   mode (env-up.sh's default — idempotent, brings up what is missing, destroys
#   nothing). Never `--fresh` by default: that does `down --volumes` on the
#   `spike-supabase` project and would eat another session's substrate.
#     * NEVER :5433. That cluster is PRODUCTION AND DEV BOTH — it serves
#       https://hq.yumyums.kitchen — and a probe against it destroyed the
#       production database on 2026-08-06 (B-141/B-143, ledger decision 155).
#     * NEVER :5434 (`yumyums-test-pg`) and none of its databases.
#     * NEVER :5432 (`infra-postgres-1`).
#   The isolation check below is an EXECUTABLE REFUSAL, not a comment: the
#   resolved published port of spike A's `db` container is compared against all
#   three and the run aborts on a match.
#
# ⚠ NO SHARED TABLE IS WRITTEN. Not one. This card creates three tables of its
#   own (`spike_d_*`), writes only to those, and drops them at teardown — so it
#   structurally CANNOT leave rows in `hq_sync_checklists` /
#   `hq_grant_projection` / the four real sync tables, which is the failure that
#   red four subtests of `TestJWTBridgeRLS` on spike B's first G2 run.
#
# ⚠ THE RESTORE IS VERIFIED (B-148's residual). B-148's residual finding is that
#   spike B's harness recovery path was never re-rehearsed after its fix, so
#   this card does not borrow it. It snapshots spike A's `spike_notes` id set
#   AND the exact `supabase_realtime` publication membership before creating
#   anything, and asserts both are byte-identical after teardown. A failed
#   restore forces exit 3 even on a green verdict.
# ───────────────────────────────────────────────────────────────────────────

set -uo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
SPIKE_DIR="$SCRIPT_DIR"
REPO_ROOT="$(cd -- "$SPIKE_DIR/../../.." && pwd)"

KEEP=0
NO_FILTER=0
WINDOW="15s"
SUBSTRATE_ARGS=()
while [ $# -gt 0 ]; do
  case "$1" in
    --keep)             KEEP=1 ;;
    --no-filter)        NO_FILTER=1 ;;
    --fresh-substrate)  SUBSTRATE_ARGS=(--fresh) ;;
    --window)           shift; WINDOW="${1:-15s}" ;;
    *) echo "usage: $(basename "$0") [--keep] [--no-filter] [--fresh-substrate] [--window 15s]" >&2; exit 64 ;;
  esac
  shift
done

# Go is not on a non-interactive shell's PATH on this box, and this script needs
# it twice (mintjwt, rtprobe). Without this they die `go: not found` (exit 127)
# and it LOOKS like a substrate failure when it is a PATH failure. Same list
# env-up.sh, spike-b-migration.sh and spike-c-roundtrip.sh use.
for _godir in "${GOROOT:-}/bin" "${GOPATH:-$HOME/go}/bin" /usr/local/go/bin /usr/lib/go/bin "$HOME/.local/go/bin"; do
  case "$_godir" in ""|"/bin") continue ;; esac
  [ -d "$_godir" ] && case ":$PATH:" in *":$_godir:"*) ;; *) PATH="$_godir:$PATH" ;; esac
done
export PATH

STEP=0
step() { STEP=$((STEP + 1)); printf '\n══ %d. %s ═══════════════════════════════\n' "$STEP" "$1"; }
# 🛑 THREE failure verbs, and the distinctions are the card's requirement.
#   cannot_run -> exit 2. Infrastructure/setup. NOT a verdict.
#   red        -> exit 1. Ran, mechanism disproven. A successful spike.
#   (exit 3 is reached only from teardown, on a failed restore.)
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

RUN_ID="${SPIKE_D_RUN_ID:-d$(date -u +%Y%m%d%H%M%S)}"
WORK="$(mktemp -d -t spike-d-XXXXXX)"

TPL_IN="spiked-${RUN_ID}-tpl-in"
TPL_OUT="spiked-${RUN_ID}-tpl-out"
TPL_ABSENT="spiked-${RUN_ID}-tpl-absent"
SUB_IN="spiked-${RUN_ID}-sub-in"
SUB_OUT="spiked-${RUN_ID}-sub-out"
RESP_IN="spiked-${RUN_ID}-resp-in"
RESP_OUT="spiked-${RUN_ID}-resp-out"

printf '# spike-d-realtime.sh — does a LIVE Realtime server honour HQ'"'"'s replication filter?\n'
printf '# repo    %s\n' "$REPO_ROOT"
printf '# anchor  %s\n' "$ANCHOR"
printf '# run     %s\n' "$RUN_ID"
printf '# mode    %s\n' "$([ "$NO_FILTER" = 1 ] && echo 'NO-FILTER (red-first capture: the clause is deliberately NOT attached)' || echo 'filters armed')"
printf '# window  %s\n' "$WINDOW"
printf '# started %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# --------------------------------------------------------------------------
# Teardown. Registered BEFORE anything is created, so an abort between the
# fixture and the first assertion still cleans up. It preserves the run's exit
# status — that status is the verdict — with exactly ONE exception: a FAILED
# substrate restore forces exit 3.
# --------------------------------------------------------------------------
PROBE_PID=""
DB_CID=""
BASELINE_PUB=""
BASELINE_NOTES=""
FIXTURE_APPLIED=0

dbq() { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -t -A -c "$1"; }

teardown() {
  local rc=$?
  set +e

  if [ -n "$PROBE_PID" ]; then
    kill "$PROBE_PID" 2>/dev/null
    wait "$PROBE_PID" 2>/dev/null
  fi

  if [ "$FIXTURE_APPLIED" = "1" ] && [ -n "$DB_CID" ]; then
    if [ "$KEEP" = "1" ]; then
      printf '\n── teardown: (--keep) spike_d_* tables LEFT IN PLACE. They are still in the\n'
      printf '   supabase_realtime publication. Drop them with:\n'
      printf '     docker exec -i %s psql -U supabase_admin -d postgres -f - < %s\n' \
        "$DB_CID" "$SPIKE_DIR/sql/spike-d-teardown.sql"
    else
      printf '\n── teardown: dropping this run'"'"'s three tables and verifying the restore ──\n'
      docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
        < "$SPIKE_DIR/sql/spike-d-teardown.sql" >/dev/null 2>&1

      local after_pub after_notes leftover
      after_pub="$(dbq "select schemaname||'.'||tablename from pg_publication_tables where pubname='supabase_realtime' order by 1" 2>/dev/null)"
      after_notes="$(dbq "select id from public.spike_notes order by id" 2>/dev/null)"
      leftover="$(dbq "select count(*) from information_schema.tables where table_schema='public' and table_name like 'spike\_d\_%'" 2>/dev/null)"

      if [ "$after_pub" = "$BASELINE_PUB" ] && [ "$after_notes" = "$BASELINE_NOTES" ] && [ "$leftover" = "0" ]; then
        printf '  VERIFIED: publication membership and spike A'"'"'s spike_notes are byte-identical\n'
        printf '            to the pre-run baseline, and no spike_d_* table remains.\n'
      else
        printf '  🛑 THE SUBSTRATE RESTORE FAILED OR WAS INCOMPLETE.\n'
        printf '     publication baseline: %s\n' "$(echo "$BASELINE_PUB" | tr '\n' ' ')"
        printf '     publication after   : %s\n' "$(echo "$after_pub" | tr '\n' ' ')"
        printf '     spike_notes baseline: %s\n' "$(echo "$BASELINE_NOTES" | tr '\n' ' ')"
        printf '     spike_notes after   : %s\n' "$(echo "$after_notes" | tr '\n' ' ')"
        printf '     spike_d_* tables remaining: %s\n' "$leftover"
        printf '     Repair before trusting any verdict from this run.\n'
        rc=3
      fi
    fi
  else
    printf '\n── teardown: nothing was created; nothing to restore ──\n'
  fi

  if [ -f "$WORK/rtprobe.log" ]; then
    cp "$WORK/rtprobe.log" "${SPIKE_D_PROBE_LOG:-$WORK/rtprobe.kept.log}" 2>/dev/null
  fi
  [ "$KEEP" = "1" ] || rm -rf "$WORK"

  # 🛑 `exit`, NOT `return`. Inside an EXIT trap a `return` cannot change the
  #    script's status, so a failed restore would be swallowed.
  exit $rc
}
trap teardown EXIT

# --------------------------------------------------------------------------
step "preflight — required tooling"
# --------------------------------------------------------------------------
for bin in docker node go; do
  command -v "$bin" >/dev/null 2>&1 || cannot_run "required tool not on PATH: $bin"
  printf '  %-6s %s\n' "$bin" "$(command -v "$bin")"
done
docker compose version >/dev/null 2>&1 || cannot_run "Compose v2 unavailable — needs 'docker compose'"
docker info >/dev/null 2>&1 || cannot_run "the Docker daemon is not reachable — 'docker info' failed"
for f in sql/spike-d-fixture.sql sql/spike-d-teardown.sql rtprobe/main.go mintjwt/main.go; do
  [ -f "$SPIKE_DIR/$f" ] || cannot_run "spike asset missing: $f"
done
[ -f "$REPO_ROOT/sync-rxdb/client.js" ] || cannot_run "sync-rxdb/client.js is missing — there is no filter to drive"

# --------------------------------------------------------------------------
step "substrate — spike A's Supabase environment (RECONCILE, never destroy)"
# --------------------------------------------------------------------------
echo "  delegating to env-up.sh (${SUBSTRATE_ARGS[*]:-reconcile}); its exit status gates this leg"
"$SPIKE_DIR/env-up.sh" "${SUBSTRATE_ARGS[@]}" \
  || cannot_run "the substrate did not come up — env-up.sh returned non-zero. Its own output above names the leg."

SBDC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")

RT_PORT="$("${SBDC[@]}" port realtime 4000 2>/dev/null | sed 's/.*://')"
case "$RT_PORT" in
  ''|*[!0-9]*) cannot_run "could not resolve spike A's Realtime host port (got '$RT_PORT')" ;;
esac
echo "  Realtime: 127.0.0.1:$RT_PORT (tenant vhost realtime-dev.localhost)"

DB_PORT="$("${SBDC[@]}" port db 5432 2>/dev/null | sed 's/.*://')"
case "$DB_PORT" in
  ''|*[!0-9]*) cannot_run "could not resolve spike A's Postgres host port (got '$DB_PORT')" ;;
esac
# 🛑 ISOLATION REFUSAL, NOT A COMMENT. :5433 is production AND dev.
case "$DB_PORT" in
  5432|5433|5434)
    cannot_run "spike A's db resolved to host port :$DB_PORT. :5433 is the PRODUCTION cluster \
(a probe there destroyed the prod DB on 2026-08-06), :5434 is yumyums-test-pg, :5432 is \
infra-postgres-1. This spike must run against the spike stack's own ephemeral port." ;;
esac
echo "  Postgres: 127.0.0.1:$DB_PORT — not 5432/5433/5434 (asserted, not assumed)"

DB_CID="$("${SBDC[@]}" ps -q db 2>/dev/null | head -1)"
[ -n "$DB_CID" ] || cannot_run "could not resolve the spike-supabase db container id"
dbq "select 1" >/dev/null 2>&1 || cannot_run "cannot run psql as supabase_admin inside $DB_CID"
echo "  db container: $DB_CID"

# --------------------------------------------------------------------------
step "baseline — snapshot what must be byte-identical after teardown"
# --------------------------------------------------------------------------
BASELINE_PUB="$(dbq "select schemaname||'.'||tablename from pg_publication_tables where pubname='supabase_realtime' order by 1")"
BASELINE_NOTES="$(dbq "select id from public.spike_notes order by id")"
[ -n "$BASELINE_PUB" ] || cannot_run "the supabase_realtime publication is empty — the substrate is not in the state env-up.sh leaves it in"
printf '  publication (%s tables): %s\n' "$(echo "$BASELINE_PUB" | wc -l)" "$(echo "$BASELINE_PUB" | tr '\n' ' ')"
printf '  spike_notes ids: %s\n' "$(echo "$BASELINE_NOTES" | tr '\n' ' ')"

if echo "$BASELINE_PUB" | grep -q 'public\.spike_d_'; then
  cannot_run "a previous spike D run left spike_d_* tables in the publication. Drop them first: \
docker exec -i $DB_CID psql -U supabase_admin -d postgres -f - < $SPIKE_DIR/sql/spike-d-teardown.sql"
fi

# --------------------------------------------------------------------------
step "fixture — three clones of HQ's real sync tables, and a parity CHECK"
# --------------------------------------------------------------------------
# ON_ERROR_STOP=1 is load-bearing: without it psql reports success having
# skipped every failing statement.
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
  < "$SPIKE_DIR/sql/spike-d-fixture.sql" \
  || cannot_run "sql/spike-d-fixture.sql did not apply cleanly"
FIXTURE_APPLIED=1
echo "  applied sql/spike-d-fixture.sql"

# 🛑 THE CLONE IS CHECKED AGAINST THE REAL TABLE, NOT ASSERTED TO MATCH IT.
parity_fail=0
for pair in "spike_d_templates:checklist_templates" \
            "spike_d_submissions:checklist_submissions" \
            "spike_d_responses:submission_responses"; do
  clone="${pair%%:*}"; real="${pair##*:}"
  cols_clone="$(dbq "select column_name||':'||data_type from information_schema.columns where table_schema='public' and table_name='$clone' order by ordinal_position")"
  cols_real="$(dbq  "select column_name||':'||data_type from information_schema.columns where table_schema='public' and table_name='$real'  order by ordinal_position")"
  ri_clone="$(dbq "select relreplident::text from pg_class where relname='$clone' and relnamespace='public'::regnamespace")"
  ri_real="$(dbq  "select relreplident::text from pg_class where relname='$real'  and relnamespace='public'::regnamespace")"
  in_pub="$(dbq "select count(*) from pg_publication_tables where pubname='supabase_realtime' and schemaname='public' and tablename='$clone'")"
  if [ "$cols_clone" = "$cols_real" ] && [ "$ri_clone" = "$ri_real" ] && [ "$in_pub" = "1" ]; then
    printf '  PARITY %-20s == %-22s cols=%s relreplident=%s in_publication=yes\n' \
      "$clone" "$real" "$(echo "$cols_clone" | wc -l)" "$ri_clone"
  else
    printf '  🛑 PARITY FAILED %s vs %s (relreplident %s vs %s, in_publication=%s)\n' \
      "$clone" "$real" "$ri_clone" "$ri_real" "$in_pub"
    diff <(echo "$cols_real") <(echo "$cols_clone") | sed 's/^/      /'
    parity_fail=1
  fi
done
[ "$parity_fail" = "0" ] || cannot_run "a clone does not match the real sync table it stands in for — the measurement would not transfer"

# --------------------------------------------------------------------------
step "the filters — read out of sync-rxdb/client.js, never typed here"
# --------------------------------------------------------------------------
SINCE="$(date -u -d '1 day ago' +%Y-%m-%dT%H:%M:%SZ 2>/dev/null)"
[ -n "$SINCE" ] || cannot_run "could not compute the LIST scope's 'since' floor (GNU date required)"

node --input-type=module -e "
import { pathToFileURL } from 'node:url';
const m = await import(pathToFileURL('$REPO_ROOT/sync-rxdb/client.js').href);
const LIST = { mode:'list', userId:'spike-d', since:'$SINCE', templateIds:['$TPL_IN','$TPL_ABSENT'] };
const FILL = { mode:'fill', userId:'spike-d', checklistId:'$SUB_IN', templateId:'$TPL_IN' };
const p = (k,v) => console.log(k + '=' + (v === null || v === undefined ? '<null>' : v));
p('FILTER_GTE',  m.realtimeFilterFor('checklists', LIST));
p('FILTER_EQ',   m.realtimeFilterFor('checklists', FILL));
p('FILTER_IN',   m.realtimeFilterFor('templates',  LIST));
p('FILTER_RESP_LIST', m.realtimeFilterFor('responses', LIST));
p('FILTER_RESP_FILL', m.realtimeFilterFor('responses', FILL));
" > "$WORK/filters.txt" 2> "$WORK/filters.err"
[ -s "$WORK/filters.txt" ] || cannot_run "could not read the filters out of sync-rxdb/client.js: $(cat "$WORK/filters.err")"

getf() { sed -n "s/^$1=//p" "$WORK/filters.txt"; }
FILTER_GTE="$(getf FILTER_GTE)"
FILTER_EQ="$(getf FILTER_EQ)"
FILTER_IN="$(getf FILTER_IN)"
FILTER_RESP_LIST="$(getf FILTER_RESP_LIST)"
FILTER_RESP_FILL="$(getf FILTER_RESP_FILL)"

printf '  checklists/LIST -> %s\n' "$FILTER_GTE"
printf '  checklists/FILL -> %s\n' "$FILTER_EQ"
printf '  templates/LIST  -> %s\n' "$FILTER_IN"
printf '  responses/LIST  -> %s\n' "$FILTER_RESP_LIST"
printf '  responses/FILL  -> %s\n' "$FILTER_RESP_FILL"

for v in "$FILTER_GTE" "$FILTER_EQ" "$FILTER_IN"; do
  [ -n "$v" ] && [ "$v" != "<null>" ] || cannot_run "sync-rxdb/client.js no longer emits a filter for one of the three filtered collections — the card's premise has moved"
done
# 🛑 The `responses` control is only a control if the ABSENCE is HQ's, not ours.
if [ "$FILTER_RESP_LIST" != "<null>" ] || [ "$FILTER_RESP_FILL" != "<null>" ]; then
  cannot_run "sync-rxdb/client.js now emits a filter for 'responses' (list=$FILTER_RESP_LIST fill=$FILTER_RESP_FILL). \
The negative control in this spike assumes it does not; measuring it would be measuring a stale premise."
fi
echo "  responses is unfiltered IN THE CODE — the control is HQ's choice, not this script's"

# --------------------------------------------------------------------------
step "token — service_role, minted with spike A's own JWT_SECRET"
# --------------------------------------------------------------------------
# Read from spike A's compose file rather than re-typed, so a rotation there
# cannot leave this script signing with a stale secret and reporting a mechanism
# failure that is really a 403.
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET out of docker-compose.supabase.yml"

# service_role (rolbypassrls=t) is chosen so RLS is a CONSTANT across all five
# channels — see sql/spike-d-fixture.sql. The unfiltered controls are what prove
# RLS is not doing the discriminating.
TOKEN="$(cd "$SPIKE_DIR" && go run ./mintjwt -secret "$JWT_SECRET" -sub "spike-d-probe" -role service_role -ttl 30m)" \
  || cannot_run "minting the service_role token failed"
[ -n "$TOKEN" ] || cannot_run "the service_role token came back empty"
echo "  token minted (role=service_role, sub=spike-d-probe, ttl=30m)"

# --------------------------------------------------------------------------
step "subscribe — five channels, one socket"
# --------------------------------------------------------------------------
if [ "$NO_FILTER" = 1 ]; then
  BIND_GTE=""; BIND_EQ=""; BIND_IN=""
  echo "  🛑 --no-filter: the clause is NOT attached to any binding (red-first capture)"
else
  BIND_GTE="$FILTER_GTE"; BIND_EQ="$FILTER_EQ"; BIND_IN="$FILTER_IN"
fi

# 🛑 BUILD, then run the BINARY. Not `go run … &`. Card C's harness lost a
# server to exactly this shape: `go run` is a supervisor process, so the pid the
# shell holds is not the pid doing the work, and teardown's `kill` leaves an
# orphan holding the socket. A built binary makes $! the process under test.
(cd "$SPIKE_DIR" && go build -o "$WORK/rtprobe" ./rtprobe) \
  || cannot_run "rtprobe would not build"

"$WORK/rtprobe" \
  -addr "127.0.0.1:$RT_PORT" -host realtime-dev.localhost -token "$TOKEN" \
  -window "$WINDOW" \
  -bind "f-gte|public|spike_d_submissions|$BIND_GTE" \
  -bind "f-eq|public|spike_d_submissions|$BIND_EQ" \
  -bind "f-in|public|spike_d_templates|$BIND_IN" \
  -bind "u-sub|public|spike_d_submissions|" \
  -bind "u-resp|public|spike_d_responses|" \
  > "$WORK/rtprobe.log" 2>&1 &
PROBE_PID=$!

# 🛑 The READY handshake. Nothing is inserted until every subscription exists,
# or a row could be written into a gap and its non-arrival would mean nothing.
for _ in $(seq 1 90); do
  grep -q '^RTP READY' "$WORK/rtprobe.log" && break
  kill -0 "$PROBE_PID" 2>/dev/null || break
  sleep 1
done
if ! grep -q '^RTP READY' "$WORK/rtprobe.log"; then
  sed 's/^/  | /' "$WORK/rtprobe.log"
  cannot_run "rtprobe never reached READY — the websocket did not connect or the joins never resolved"
fi
grep -E '^RTP (CONNECT|BIND|JOIN-OK|JOIN-ERR|SYS|SYS-ERR|STATE|READY)' "$WORK/rtprobe.log" | sed 's/^/  /'

# --------------------------------------------------------------------------
step "write — two rows per table, as the substrate superuser"
# --------------------------------------------------------------------------
docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q <<SQL \
  || cannot_run "the spike rows would not insert"
insert into public.spike_d_templates (id, name) values
  ('$TPL_IN',  'spike D in-scope template'),
  ('$TPL_OUT', 'spike D out-of-scope template');

insert into public.spike_d_submissions (id, template_id, template_snapshot, submitted_by, submitted_at) values
  ('$SUB_IN',  '$TPL_IN', '{}'::jsonb, 'spike-d', now()),
  ('$SUB_OUT', '$TPL_IN', '{}'::jsonb, 'spike-d', now() - interval '30 days');

insert into public.spike_d_responses (id, submission_id, field_id, value, answered_by, answered_at) values
  ('$RESP_IN',  '$SUB_IN',  'spike-d-field', '"in"'::jsonb,  'spike-d', now()),
  ('$RESP_OUT', '$SUB_OUT', 'spike-d-field', '"out"'::jsonb, 'spike-d', now() - interval '30 days');
SQL
printf '  templates   IN=%s  OUT=%s\n' "$TPL_IN" "$TPL_OUT"
printf '  submissions IN=%s (submitted_at=now)  OUT=%s (submitted_at=now-30d)\n' "$SUB_IN" "$SUB_OUT"
printf '  responses   %s / %s\n' "$RESP_IN" "$RESP_OUT"
printf '  the LIST floor is %s — IN is above it, OUT is 29 days below it\n' "$SINCE"

# --------------------------------------------------------------------------
step "observe — waiting out rtprobe's window"
# --------------------------------------------------------------------------
wait "$PROBE_PID"; PROBE_RC=$?
PROBE_PID=""
echo "  rtprobe exit=$PROBE_RC"
grep -E '^RTP (EVENT|COUNT|DONE)' "$WORK/rtprobe.log" | sed 's/^/  /'
[ "$PROBE_RC" = "0" ] || cannot_run "rtprobe exited $PROBE_RC (3 = could not run). Its log is above."

saw()   { grep -qE "^RTP EVENT label=$1 .* id=$2\$" "$WORK/rtprobe.log"; }
state() { sed -n "s/^RTP STATE label=$1 state=//p" "$WORK/rtprobe.log" | head -1; }
mark()  { if [ "$1" = "1" ]; then printf '  ✅ %s\n' "$2"; else printf '  ❌ %s\n' "$2"; fi; }

# --------------------------------------------------------------------------
step "leg 0 — every subscription was actually ESTABLISHED"
# --------------------------------------------------------------------------
# 🛑 A join reply of "ok" is not a subscription; the failure arrives afterwards
# as a `system` frame. See rtprobe's header.
FAIL=0
for l in u-sub u-resp; do
  s="$(state "$l")"
  [ "$s" = "SUBSCRIBED" ] || cannot_run "the UNFILTERED control channel '$l' is $s — Realtime itself is not usable, so there is no filter question to answer"
  printf '  ✅ control %-7s %s\n' "$l" "$s"
done
for l in f-gte f-eq f-in; do
  s="$(state "$l")"
  if [ "$s" != "SUBSCRIBED" ]; then
    printf '  ❌ filtered %-7s %s\n' "$l" "$s"
    grep -E "^RTP (SYS-ERR|JOIN-ERR) label=$l" "$WORK/rtprobe.log" | sed 's/^/      /'
    red "the filtered channel '$l' could not be established ($s) while both UNFILTERED controls could. \
The substrate REJECTS the clause sync-rxdb/client.js emits — B-42 option (i) is not usable as written."
  fi
  printf '  ✅ filtered %-7s %s\n' "$l" "$s"
done

# --------------------------------------------------------------------------
step "leg 1 — POSITIVE ARRIVAL (mandatory; without it the rest is vacuous)"
# --------------------------------------------------------------------------
POS=1
saw u-sub  "$SUB_IN"  && mark 1 "u-sub  received the in-scope submission"  || { mark 0 "u-sub  did NOT receive the in-scope submission"; POS=0; }
saw u-resp "$RESP_IN" && mark 1 "u-resp received the in-scope response"    || { mark 0 "u-resp did NOT receive the in-scope response";   POS=0; }
if [ "$POS" = "0" ]; then
  cannot_run "the UNFILTERED controls received nothing — Realtime is not delivering row changes at all on this stack, \
so nothing here is evidence about the filter. Not a verdict."
fi

FPOS=1
saw f-gte "$SUB_IN" && mark 1 "f-gte  received the in-scope submission (submitted_at >= floor)" || { mark 0 "f-gte  did NOT receive the in-scope submission"; FPOS=0; }
saw f-eq  "$SUB_IN" && mark 1 "f-eq   received the in-scope submission (id = eq)"                || { mark 0 "f-eq   did NOT receive the in-scope submission"; FPOS=0; }
saw f-in  "$TPL_IN" && mark 1 "f-in   received the in-scope template (id in (...))"              || { mark 0 "f-in   did NOT receive the in-scope template";   FPOS=0; }
if [ "$FPOS" = "0" ]; then
  red "a filtered channel dropped the IN-SCOPE row while the unfiltered control on the same table received it. \
The filter is not merely ignored — it is WRONG, and it would make the live leg blind to rows the pull admits."
fi

# --------------------------------------------------------------------------
step "leg 2 — SAME-TABLE UNFILTERED CONTROL (the out-of-scope row DID reach Realtime)"
# --------------------------------------------------------------------------
CTL=1
saw u-sub  "$SUB_OUT"  && mark 1 "u-sub  received the out-of-scope submission" || { mark 0 "u-sub  did NOT receive the out-of-scope submission"; CTL=0; }
saw u-resp "$RESP_OUT" && mark 1 "u-resp received the out-of-scope response"   || { mark 0 "u-resp did NOT receive the out-of-scope response";   CTL=0; }
saw u-sub  "$TPL_IN" && { mark 0 "u-sub received a TEMPLATE row — the channels are cross-talking"; CTL=0; } || true
if [ "$CTL" = "0" ]; then
  cannot_run "the out-of-scope row never reached Realtime on the UNFILTERED channel, so its absence on the \
filtered channels proves nothing about the filter. The fixture, not the filter, is what this run measured. Not a verdict."
fi
echo "  → any suppression observed below is therefore attributable to the FILTER and to nothing else"

# --------------------------------------------------------------------------
step "leg 3 — the 'responses' NEGATIVE CONTROL (filter deliberately absent)"
# --------------------------------------------------------------------------
# Both rows must arrive. This is what turns client.js's "responses gets nothing"
# from a comment into a measurement.
RC3=1
saw u-resp "$RESP_IN"  && mark 1 "responses: in-scope row arrived"     || { mark 0 "responses: in-scope row missing";     RC3=0; }
saw u-resp "$RESP_OUT" && mark 1 "responses: out-of-scope row arrived" || { mark 0 "responses: out-of-scope row missing"; RC3=0; }
[ "$RC3" = "1" ] || red "the 'responses' collection is UNFILTERED by construction and yet did not receive both rows. \
client.js's stated residual (B-42 stays open on responses) does not match what the substrate does."

# --------------------------------------------------------------------------
step "leg 4 — SUPPRESSION (the question the card exists to answer)"
# --------------------------------------------------------------------------
SUP=1
saw f-gte "$SUB_OUT" && { mark 0 "f-gte  RECEIVED the out-of-scope submission — submitted_at=gte was IGNORED"; SUP=0; } || mark 1 "f-gte  suppressed the out-of-scope submission"
saw f-eq  "$SUB_OUT" && { mark 0 "f-eq   RECEIVED the out-of-scope submission — id=eq was IGNORED";           SUP=0; } || mark 1 "f-eq   suppressed the out-of-scope submission"
saw f-in  "$TPL_OUT" && { mark 0 "f-in   RECEIVED the out-of-scope template — id=in.(...) was IGNORED";       SUP=0; } || mark 1 "f-in   suppressed the out-of-scope template"

if [ "$SUP" = "0" ]; then
  red "the live Realtime server DID NOT HONOUR the replication filter. The out-of-scope row arrived on a \
channel whose binding carried the clause. This is B-62 exactly: B-42's unscoped live leg is STILL OPEN, and \
sync-rxdb/client.js, tests/sync-rxdb-client.spec.js [SCOPE-04] and B-42's own entry all say otherwise."
fi

# --------------------------------------------------------------------------
step "VERDICT"
# --------------------------------------------------------------------------
cat <<EOF
  GREEN.

  A live self-hosted Realtime server (supabase/realtime, spike A's stack)
  HONOURS the replication filter sync-rxdb/client.js emits, in all three clause
  shapes HQ produces:

    submitted_at=gte.<iso>   in-scope arrived, out-of-scope suppressed
    id=eq.<id>               in-scope arrived, out-of-scope suppressed
    id=in.(<id>,<id>)        in-scope arrived, out-of-scope suppressed

  and the collection HQ deliberately leaves unfiltered ('responses') received
  BOTH rows — so that residual is now measured, not merely stated.

  The suppression is attributable to the filter: the identical out-of-scope row
  arrived on an unfiltered subscription to the SAME TABLE over the SAME SOCKET
  in the SAME window.

  B-62 is answered. [SCOPE-04]'s config-level assertions now sit on top of a
  live-substrate proof rather than an assumption.
EOF
exit 0
