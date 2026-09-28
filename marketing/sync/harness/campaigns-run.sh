#!/usr/bin/env bash
# marketing/sync/harness/campaigns-run.sh — the standalone campaigns-replica
# gate for card requires-online-replication (run 20260906; B-345 precedent:
# the runnable script IS the verdict). See campaigns-harness.mjs for the legs:
# optional bound (campaigns unbounded, codes still bounded), the production
# campaigns replica, the §8 refusal on the PRODUCTION policy source (spike
# 02's four-run matrix, mode 'throw'), and the trigger-stamped downgrade flip
# re-delivered on the next RESYNC.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  every leg held.   exit 1  a leg failed.   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase project only (RECONCILE mode, never --fresh); never :5433,
# never :5434, no hosted project. lib.sh prints resolved coordinates read-only
# before any write.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply"

echo
echo "── enumerated: campaigns replication surface (the committed 20260906000100 migration) ──"
psqlq <<'SQL' | sed 's/^/#   /'
select 'supabase_realtime member: ' || coalesce(string_agg(tablename, ', ' order by tablename), '(none)')
  from pg_publication_tables
 where pubname='supabase_realtime' and schemaname='public';
select 'campaigns touch trigger : ' || coalesce(string_agg(tgname, ', '), '(none)')
  from pg_trigger t join pg_class c on c.oid = t.tgrelid
 where c.relname = 'campaigns' and not t.tgisinternal;
SQL
psqlq <<'SQL' | grep -qx 'campaigns' || fail "public.campaigns is NOT in the supabase_realtime publication — a campaigns replica gets no Realtime nudge (spike build-fact 3)"
select tablename from pg_publication_tables
 where pubname='supabase_realtime' and schemaname='public' and tablename='campaigns';
SQL

echo
echo "── device JWT (throwaway secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN_A="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here; the link is gitignored).
[ -e "$SCRIPT_DIR/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/node_modules"

echo
set +e
C8_JWT="$TOKEN_A" C8_DB_CID="$DB_CID" node "$SCRIPT_DIR/campaigns-harness.mjs"
NODE_EXIT=$?
set -e

if [ "$NODE_EXIT" -eq 0 ]; then
  printf '\n✅ VERDICT: GREEN — the campaigns replica carries the flag AND a downgrade flip; the §8 refusal arms on real data; codes keep their bound.\n'
else
  printf '\n🛑 VERDICT: RED — a leg failed (node exit %s); see the log above.\n' "$NODE_EXIT"
fi
exit "$NODE_EXIT"
