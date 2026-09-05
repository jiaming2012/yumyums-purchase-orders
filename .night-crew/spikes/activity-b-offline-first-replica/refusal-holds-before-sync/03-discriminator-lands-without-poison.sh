#!/usr/bin/env bash
# 03-discriminator-lands-without-poison.sh — spike: the done_when's second
# half, measured per layer.
#
# The card must make "campaigns replica failed/unready" distinguishable from
# "genuinely unknown campaign" IN THE ATTEMPT RECORD. Today a
# `policy_unresolved` discriminator would be dropped at TWO shipped layers
# (enqueueAttempt's destructured field list; the push handler's whitelisted
# landing body) — and if the card extends the landing body before the server
# column exists, PostgREST's unknown-column refusal is the same throw-retry
# head-of-line poison class F-2 measured (12 redeem attempts, 0 landings).
#
# Legs (PRE → spike-local DDL → POST → psql readback):
#   PRE  1. shipped enqueueAttempt: the extra field's fate, measured
#        2. direct insert on the shipped SCAN_ATTEMPTS_SCHEMA under ajv: measured
#        3. the card's future landing body (F-2 land-unverified + the field)
#           against the PRE-migration server: status enumerated (the poison leg)
#   DDL  alter table scan_attempts add policy_unresolved boolean not null
#        default false — INSIDE the spike, never the committed migrations;
#        reported as the migration the card owes. notify pgrst reload.
#   POST 4. extended body lands 201 (discriminated) + control lands 201 (false)
#        5. shipped handler drains a queue holding one extended-local-row
#           unverified attempt + one legitimate redeem on a FRESHLY-MINTED
#           live code (sibling build-fact 5) — no throw, no head-of-line
#        6. psql readback: the rows are distinguishable, the F-2 check
#           constraint holds, no new terminal status
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0  the discriminator lands, distinguishable, poison-free (DDL-first).
#   exit 1  a measurement disagreed.   exit 2  could not run.
#
# Substrate discipline: supabase/verify/lib.sh — the throwaway LOCAL
# spike-supabase compose project only; never :5433, never :5434, no hosted
# project.

set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"

# shellcheck source=/dev/null
. "$REPO_ROOT/supabase/verify/lib.sh"

substrate_up
echo
echo "── built schema: reset_bare + apply_all (committed migrations + seed) ──"
reset_bare || cannot_run "reset_bare failed"
apply_all  || cannot_run "the committed migrations/seed did not apply — Activity A/B's gate, not this spike's premise"

echo
echo "── the shipped artifacts under test (identity pinned, not assumed) ──"
for f in marketing/sync/push-replication.js supabase/migrations/20260906000200_scan_attempts_unverified_landing.sql; do
  [ -f "$REPO_ROOT/$f" ] || cannot_run "$f missing — this spike tests the SHIPPED artifact, not a copy"
  printf '#   %-64s sha256 %s\n' "$f" "$(shasum -a 256 "$REPO_ROOT/$f" | cut -c1-16)…"
done

echo
echo "── device JWT (secret from the compose file) ──"
JWT_SECRET="$(grep -m1 -oE 'JWT_SECRET: *[0-9a-f]{32,}' "$REPO_ROOT/docker-compose.supabase.yml" | awk '{print $2}')"
[ -n "$JWT_SECRET" ] || cannot_run "could not read JWT_SECRET"
TOKEN="$(cd "$QA" && go run ./mintjwt -secret "$JWT_SECRET" -sub device-a -role authenticated -ttl 30m)" || cannot_run "mint device-a failed"

[ -e "$SCRIPT_DIR/js/node_modules" ] || ln -s "$QA/rxdb/node_modules" "$SCRIPT_DIR/js/node_modules"

echo
echo "════ PRE legs (server does NOT have the column yet) ════"
node "$SCRIPT_DIR/js/discriminator.mjs" "$TOKEN" pre \
  || fail "a PRE measurement disagreed — see the node log above"

echo
echo "════ the spike-local DDL — the migration the card owes ════"
echo "#   alter table public.scan_attempts add column policy_unresolved boolean not null default false"
psqlq <<'SQL' >/dev/null || cannot_run "spike-local DDL failed"
alter table public.scan_attempts add column if not exists policy_unresolved boolean not null default false;
notify pgrst, 'reload schema';
SQL
sleep 2

echo
echo "── a FRESH live code for the drain leg (sibling build-fact 5: never reuse a seeded code) ──"
LIVE_CODE_ID="$(psqlq <<'SQL'
insert into public.codes (id, token_hash, campaign_id, expires_at)
values (gen_random_uuid(), md5(random()::text) || md5(random()::text),
        'a0000000-0000-4000-8000-000000000001', now() + interval '7 days')
returning id;
SQL
)" || cannot_run "could not mint a fresh live code"
echo "#   live code: $LIVE_CODE_ID"

echo
echo "════ POST legs (column exists; PostgREST schema reloaded) ════"
node "$SCRIPT_DIR/js/discriminator.mjs" "$TOKEN" post "$LIVE_CODE_ID" \
  || fail "a POST measurement disagreed — see the node log above"

echo
echo "── psql readback: distinguishability + the F-2 constraint (server truth, not client claims) ──"
ROWS="$(psqlq <<'SQL'
select coalesce(token_hash, code_id::text), status, offline_override, unverified_code, policy_unresolved
from public.scan_attempts
order by scanned_at;
SQL
)"
echo "$ROWS" | sed 's/^/    /'

echo "$ROWS" | grep -q "^$(printf 'a1%.0s' {1..32})|accepted|t|t|t$" \
  || fail "the discriminated override did not read back policy_unresolved=t"
echo "$ROWS" | grep -q "^$(printf 'b2%.0s' {1..32})|accepted|t|t|f$" \
  || fail "the control (genuinely-unknown campaign) did not read back policy_unresolved=f"
echo "$ROWS" | grep -q "^$LIVE_CODE_ID|accepted|f|f|f$" \
  || fail "the legitimate redeem behind the discriminated attempt did not land accepted"

STATUSES="$(psqlq <<<'select distinct status from public.scan_attempts;')"
for s in $STATUSES; do
  case "$s" in pending|accepted|rejected) ;; *) fail "a NEW terminal status appeared: $s — the §9/§19 taxonomy was supposed to hold" ;; esac
done

CON="$(psqlq <<<"select count(*) from pg_constraint where conname='scan_attempts_names_a_code';")"
[ "$CON" = "1" ] || fail "the F-2 check constraint scan_attempts_names_a_code is gone after the DDL"

printf '\n✅ VERDICT: GREEN — the discriminator lands and reads back distinguishable, the F-2 constraint holds, no new status, and the DDL-first order is poison-free.\n'
