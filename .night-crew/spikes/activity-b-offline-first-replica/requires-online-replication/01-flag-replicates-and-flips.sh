#!/usr/bin/env bash
# 01-flag-replicates-and-flips.sh — spike: WHICH mechanism carries the §8
# `requires_online` flag to an offline device, and — the leg that decides the
# design — carries a CHANGE to it.
#
# The roadmap card names two candidates without choosing ("a campaigns replica,
# or embed the flag in the codes pull"). Landing the flag on initial sync is the
# easy half. A mechanism that lands it once and then goes permanently stale arms
# the §8 refusal WRONG — a campaign the operator makes online-only keeps being
# overridable on every tablet — which is worse than today's honest unknown→false.
#
# Legs, all enumerated (B-216 — a finding is a set, not a sample):
#   (1) what `authenticated` can actually read on public.campaigns;
#   (2) supabase_realtime publication membership (the nudge source);
#   (3) can the SHIPPED makePullHandler serve `campaigns` unchanged?
#   (4) mechanism A (campaigns replica) and mechanism B (codes pull + PostgREST
#       FK embed) both to steady state, then a flag FLIP observed on each —
#       first with the writer NOT stamping updated_at, then stamping it.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  a mechanism carries the flag AND the flip, and the loser is
#           demonstrated to go stale.   exit 1  neither carries the flip.
#   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase compose project only; never :5433, never :5434, no hosted
# project. Extra DDL a mechanism needs is applied INSIDE this spike, never to
# the committed migrations; what it applies is reported as a migration the card
# owes.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A's gate, not this spike's premise"

echo
echo "── enumerated: supabase_realtime publication membership (public schema) ──"
echo "#   (a campaigns replica with no publication row gets no Realtime frame —"
echo "#    it can only be nudged by another table's channel or a re-SUBSCRIBE)"
psqlq <<'SQL' | sed 's/^/#   in publication: /'
select tablename from pg_publication_tables
 where pubname = 'supabase_realtime' and schemaname = 'public'
 order by tablename;
SQL

echo
echo "── enumerated: authenticated's grants on public.campaigns ──"
psqlq <<'SQL' | sed 's/^/#   grant: /'
select privilege_type from information_schema.role_table_grants
 where table_schema='public' and table_name='campaigns' and grantee='authenticated'
 order by privilege_type;
SQL

echo
echo "── enumerated: does anything advance campaigns.updated_at on UPDATE? ──"
psqlq <<'SQL' | sed 's/^/#   trigger: /'
select coalesce(string_agg(tgname, ', '), '(none — updated_at is default-only, never touched on UPDATE)')
  from pg_trigger t join pg_class c on c.oid = t.tgrelid
 where c.relname = 'campaigns' and not t.tgisinternal;
SQL

echo
echo "── device JWT (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

# Node module resolution: borrow the proven QA rxdb node_modules via symlink
# (walk-up resolution; nothing installed here).
[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

# The flip target: campaign …0001 is seeded requires_online=false and owns four
# of the five seeded codes. Flipping it to TRUE is the operator making a
# campaign online-only — the direction whose staleness LEAKS MONEY (a device
# that never hears about it keeps offering the offline override).
FLIP_CAMPAIGN='a0000000-0000-4000-8000-000000000001'

echo
echo "── the mechanisms, side by side (node, RxDB memory storage) ──"
echo "#   flip target: campaign $FLIP_CAMPAIGN (seeded requires_online=false → true)"
node "$SCRIPT_DIR/js/flag-replicates.mjs" "$TOKEN" "$FLIP_CAMPAIGN" "$DB_CID" \
  || fail "no mechanism carried the flag flip — see the node log above"

printf '\n✅ VERDICT: GREEN — a mechanism carries the flag AND a flip; the loser is demonstrated stale.\n'
