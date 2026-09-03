#!/usr/bin/env bash
# 01-schema-applies.sh — spike: the §4 schema applies clean on a Supabase-shaped
# Postgres, twice, and every structural claim is enumerable by name.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   schema applied twice (fresh + warm = "applies clean"), and every
#            assertion below found its named object: 3 tables, the F2 column,
#            the unique token_hash, the updated_at checkpoint index, RLS enabled
#            on all 3, the 3 policies BY NAME, codes in supabase_realtime.
#   exit 1   an assertion failed — the premise is not proven.
#   exit 2   could not run (substrate would not come up, tooling missing).
#
# Enumerated, not sampled (B-216): the assertions print the enumerating query's
# full result, so a reader can see the set, not a count standing in for one.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
QA="$REPO_ROOT/.night-crew/qa/spike-supabase"

fail()       { printf '\n🛑 VERDICT: RED — %s\n' "$1" >&2; exit 1; }
cannot_run() { printf '\n⚠ COULD-NOT-RUN — %s\n' "$1" >&2; exit 2; }

# ---------------------------------------------------------------------------
# Prod-safety coordinates, read-only, before any write (decision 155's habit):
# this script touches ONLY the throwaway `spike-supabase` compose project.
# Never :5433 (dev/prod cluster), never :5434 (test cluster), no hosted
# Supabase project exists or is contacted.
# ---------------------------------------------------------------------------
echo "# target coordinates (read-only statement before any write):"
echo "#   compose project : spike-supabase (throwaway, committed throwaway creds)"
echo "#   compose file    : $REPO_ROOT/docker-compose.supabase.yml"
echo "#   NOT :5433, NOT :5434, NOT any hosted supabase.com project"

[ -f "$SCRIPT_DIR/sql/qr-schema.sql" ] || cannot_run "fixture missing: $SCRIPT_DIR/sql/qr-schema.sql"

echo
echo "── substrate up (env-up.sh, idempotent reconcile) ──"
"$QA/env-up.sh" || cannot_run "env-up.sh did not reach GREEN — the substrate is a precondition, not this spike's premise"

# Anchor compose exactly the way env-up.sh does (path-stability fix).
ANCHOR=""
if command -v git >/dev/null 2>&1; then
  _cd="$(cd -- "$REPO_ROOT" && git rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)"
  [ -n "$_cd" ] && ANCHOR="$(cd -- "$_cd/.." && pwd)"
fi
[ -n "$ANCHOR" ] || ANCHOR="$REPO_ROOT"
DC=(docker compose -p spike-supabase --project-directory "$ANCHOR" -f "$REPO_ROOT/docker-compose.supabase.yml")
DB_CID="$("${DC[@]}" ps -q db)"
[ -n "$DB_CID" ] || cannot_run "no db container after env-up GREEN"
echo "#   db container    : $DB_CID"

psqlq() { docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -qtA; }

apply() {
  docker exec -i "$DB_CID" psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -q -f - \
    < "$SCRIPT_DIR/sql/qr-schema.sql"
}

echo
echo "── apply #1 (fresh: the fixture drops and recreates) ──"
apply || fail "the §4 schema did not apply clean (first apply)"

echo "── apply #2 (warm: 'applies clean' must survive a re-run) ──"
apply || fail "the §4 schema did not apply clean on a second run"

echo
echo "── assertions, each enumerated by name ──"

echo "· tables in public:"
TABLES="$(echo "select tablename from pg_tables where schemaname='public' and tablename in ('campaigns','codes','scan_attempts') order by 1;" | psqlq)"
echo "$TABLES" | sed 's/^/    /'
[ "$(echo "$TABLES" | grep -c .)" = 3 ] || fail "expected 3 tables (campaigns, codes, scan_attempts), got: $TABLES"

echo "· scan_attempts.unverified_code (F2) is a boolean:"
COL="$(echo "select column_name||':'||data_type from information_schema.columns where table_schema='public' and table_name='scan_attempts' and column_name='unverified_code';" | psqlq)"
echo "    ${COL:-<absent>}"
[ "$COL" = "unverified_code:boolean" ] || fail "unverified_code boolean missing from scan_attempts"

echo "· codes indexes (unique token_hash + the updated_at checkpoint key):"
IDX="$(echo "select indexname||' :: '||indexdef from pg_indexes where schemaname='public' and tablename='codes' order by 1;" | psqlq)"
echo "$IDX" | sed 's/^/    /'
echo "$IDX" | grep -q 'UNIQUE.*token_hash' || fail "no unique index on codes.token_hash"
echo "$IDX" | grep -q '(updated_at)'       || fail "no index on codes(updated_at) — the replication checkpoint key"

echo "· scan_attempts join-key index (pos_business_date, pos_order_number):"
JIDX="$(echo "select indexname from pg_indexes where schemaname='public' and tablename='scan_attempts' and indexdef like '%pos_business_date%pos_order_number%';" | psqlq)"
echo "    ${JIDX:-<absent>}"
[ -n "$JIDX" ] || fail "no index on scan_attempts(pos_business_date, pos_order_number)"

echo "· RLS enabled (relrowsecurity) on all three:"
RLS="$(echo "select c.relname||':'||c.relrowsecurity from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relname in ('campaigns','codes','scan_attempts') order by 1;" | psqlq)"
echo "$RLS" | sed 's/^/    /'
# a concatenated boolean casts to 'true'/'false', not psql's bare-column 't'/'f'
[ "$(echo "$RLS" | grep -c ':true$')" = 3 ] || fail "RLS is not enabled on all three tables: $RLS"

echo "· policies by name:"
POL="$(echo "select tablename||'.'||policyname from pg_policies where schemaname='public' and tablename in ('campaigns','codes','scan_attempts') order by 1;" | psqlq)"
echo "$POL" | sed 's/^/    /'
for p in campaigns.campaigns_select_device codes.codes_select_device scan_attempts.scan_attempts_insert_own; do
  echo "$POL" | grep -qx "$p" || fail "policy missing: $p"
done

echo "· supabase_realtime publication membership (§7.1):"
PUB="$(echo "select schemaname||'.'||tablename from pg_publication_tables where pubname='supabase_realtime' order by 1;" | psqlq)"
echo "$PUB" | sed 's/^/    /'
echo "$PUB" | grep -qx 'public.codes' || fail "public.codes is not in the supabase_realtime publication"

printf '\n✅ VERDICT: GREEN — §4 schema applies clean (twice) and every structural claim is present by name\n'
exit 0
