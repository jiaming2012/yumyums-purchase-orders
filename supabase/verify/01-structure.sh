#!/usr/bin/env bash
# 01-structure.sh — Card 1 gate: the in-repo migration applies clean TWICE
# (fresh and warm) and every structural claim is enumerable BY NAME — the
# spike's proven assertion set PLUS the #5 `marketing_settings` surface and the
# TEST seed fixtures. Adapted from the spike's 01-schema-applies.sh (read-only
# source); the spike applied a drop-first fixture, this applies the ADDITIVE
# in-repo migration under supabase/migrations/.
#
# 🛑 THE VERDICT IS THIS SCRIPT'S EXIT STATUS, NEVER ITS PROSE.
#   exit 0   default mode: bare → apply #1 (fresh) → all assertions → apply #2
#            (warm) → all assertions again → the #5 config-survives-reapply leg.
#            --assert-only: every named object is present RIGHT NOW (no apply).
#   exit 1   an assertion failed. Against a bare substrate, --assert-only
#            exiting 1 is the red-first evidence — the probe does NOT invert
#            its meaning to "expect failure"; red is red.
#   exit 2   could not run (substrate would not come up, tooling missing).
#
# Enumerated, not sampled (B-216): assertions print the enumerating query's
# full result, so a reader sees the set, not a count standing in for one.
#
# USAGE
#   01-structure.sh                # the full fresh+warm gate
#   01-structure.sh --assert-only  # assert current state only (red-first probe)

set -euo pipefail
# shellcheck source=lib.sh
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"

MODE="full"
case "${1:-}" in
  --assert-only) MODE="assert" ;;
  "") ;;
  *) echo "usage: $(basename "$0") [--assert-only]" >&2; exit 64 ;;
esac

substrate_up

assert_all() {
  echo
  echo "── assertions, each enumerated by name ──"

  echo "· tables in public (the 3 arbiter tables + the #5 settings table):"
  TABLES="$(echo "select tablename from pg_tables where schemaname='public' and tablename in ('campaigns','codes','scan_attempts','marketing_settings') order by 1;" | psqlq)"
  echo "$TABLES" | sed 's/^/    /'
  [ "$(echo "$TABLES" | grep -c .)" = 4 ] || fail "expected 4 tables (campaigns, codes, marketing_settings, scan_attempts), got: $TABLES"

  echo "· scan_attempts.unverified_code (F2) is a boolean:"
  COL="$(echo "select column_name||':'||data_type from information_schema.columns where table_schema='public' and table_name='scan_attempts' and column_name='unverified_code';" | psqlq)"
  echo "    ${COL:-<absent>}"
  [ "$COL" = "unverified_code:boolean" ] || fail "unverified_code boolean missing from scan_attempts"

  echo "· codes never stores a raw token (no token/raw_token column):"
  RAWCOL="$(echo "select count(*) from information_schema.columns where table_schema='public' and table_name='codes' and column_name in ('token','raw_token');" | psqlq)"
  echo "    raw-token-shaped columns on codes: $RAWCOL"
  [ "$RAWCOL" = 0 ] || fail "codes carries a raw-token-shaped column — token_hash is the only identity the arbiter may hold"

  echo "· codes indexes (unique token_hash + the updated_at checkpoint key):"
  IDX="$(echo "select indexname||' :: '||indexdef from pg_indexes where schemaname='public' and tablename='codes' order by 1;" | psqlq)"
  echo "$IDX" | sed 's/^/    /'
  echo "$IDX" | grep -q 'UNIQUE.*token_hash' || fail "no unique index on codes.token_hash"
  echo "$IDX" | grep -q '(updated_at)'       || fail "no index on codes(updated_at) — the replication checkpoint key"

  echo "· scan_attempts join-key index (pos_business_date, pos_order_number):"
  JIDX="$(echo "select indexname from pg_indexes where schemaname='public' and tablename='scan_attempts' and indexdef like '%pos_business_date%pos_order_number%';" | psqlq)"
  echo "    ${JIDX:-<absent>}"
  [ -n "$JIDX" ] || fail "no index on scan_attempts(pos_business_date, pos_order_number)"

  echo "· RLS enabled (relrowsecurity) on all four:"
  # a concatenated boolean casts to 'true'/'false', not psql's bare-column 't'/'f'
  # (the spike's one corrected defect — do not regress it)
  RLS="$(echo "select c.relname||':'||c.relrowsecurity from pg_class c join pg_namespace n on n.oid=c.relnamespace where n.nspname='public' and c.relname in ('campaigns','codes','scan_attempts','marketing_settings') order by 1;" | psqlq)"
  echo "$RLS" | sed 's/^/    /'
  [ "$(echo "$RLS" | grep -c ':true$')" = 4 ] || fail "RLS is not enabled on all four tables: $RLS"

  echo "· policies by name (and marketing_settings has NONE — server-side only):"
  POL="$(echo "select tablename||'.'||policyname from pg_policies where schemaname='public' and tablename in ('campaigns','codes','scan_attempts','marketing_settings') order by 1;" | psqlq)"
  echo "$POL" | sed 's/^/    /'
  for p in campaigns.campaigns_select_device codes.codes_select_device scan_attempts.scan_attempts_insert_own; do
    echo "$POL" | grep -qx "$p" || fail "policy missing: $p"
  done
  echo "$POL" | grep -q '^marketing_settings\.' && fail "marketing_settings has a policy — it must stay server-side only (RLS on, no policies, no client grants)"

  echo "· marketing_settings client grants (must be zero for anon/authenticated):"
  MSG="$(echo "select count(*) from information_schema.role_table_grants where table_schema='public' and table_name='marketing_settings' and grantee in ('anon','authenticated');" | psqlq)"
  echo "    grants to anon/authenticated: $MSG"
  [ "$MSG" = 0 ] || fail "marketing_settings is granted to a client role — the threshold is consulted server-side at campaign creation, never read by devices"

  echo "· marketing_settings shape (#5): threshold column + singleton guard by name:"
  MSCOL="$(echo "select column_name||':'||data_type from information_schema.columns where table_schema='public' and table_name='marketing_settings' and column_name='requires_online_threshold_cents';" | psqlq)"
  echo "    ${MSCOL:-<absent>}"
  [ "$MSCOL" = "requires_online_threshold_cents:integer" ] || fail "requires_online_threshold_cents integer missing from marketing_settings"
  MSCON="$(echo "select conname from pg_constraint where conrelid='public.marketing_settings'::regclass and conname='marketing_settings_singleton';" | psqlq)"
  echo "    constraint: ${MSCON:-<absent>}"
  [ "$MSCON" = "marketing_settings_singleton" ] || fail "marketing_settings_singleton check constraint missing — the table must be structurally single-row"

  echo "· marketing_settings is seeded, single-row, threshold 2000 (\$20):"
  MSROW="$(echo "select count(*)||':'||coalesce(min(requires_online_threshold_cents)::text,'-') from public.marketing_settings;" | psqlq)"
  echo "    rows:threshold = $MSROW"
  [ "$MSROW" = "1:2000" ] || fail "expected exactly one settings row with threshold 2000, got $MSROW"

  echo "· TEST campaigns seeded (one requires_online=false, one true — derived from face value vs the threshold):"
  CAMPS="$(echo "select id||' '||requires_online||' '||name from public.campaigns order by id;" | psqlq)"
  echo "$CAMPS" | sed 's/^/    /'
  echo "$CAMPS" | grep -q '^a0000000-0000-4000-8000-000000000001 false' || fail "TEST low-value campaign (requires_online=false) missing"
  echo "$CAMPS" | grep -q '^a0000000-0000-4000-8000-000000000002 true'  || fail "TEST high-value campaign (requires_online=true) missing"
  [ "$(echo "$CAMPS" | grep -c .)" = 2 ] || fail "expected exactly the 2 TEST campaigns (idempotent seed), got: $CAMPS"

  echo "· TEST codes seeded (5 fixed fixtures for Card 2 + the RLS/Realtime legs):"
  CODES="$(echo "select id||' '||case when redeemed_at is null then 'unredeemed' else 'redeemed' end||' '||case when expires_at > now() then 'active' else 'expired' end from public.codes order by id;" | psqlq)"
  echo "$CODES" | sed 's/^/    /'
  for c in c0000000-0000-4000-8000-000000000001 c0000000-0000-4000-8000-000000000002 c0000000-0000-4000-8000-000000000003 c0000000-0000-4000-8000-000000000004 c0000000-0000-4000-8000-000000000005; do
    echo "$CODES" | grep -q "^$c " || fail "seeded TEST code missing: $c"
  done
  [ "$(echo "$CODES" | grep -c .)" = 5 ] || fail "expected exactly the 5 TEST codes (idempotent seed), got: $CODES"
  echo "$CODES" | grep -q '^c0000000-0000-4000-8000-000000000003 unredeemed expired' || fail "code …0003 must be the expired fixture (Card 2's expired leg)"
  echo "$CODES" | grep -q '^c0000000-0000-4000-8000-000000000004 redeemed'           || fail "code …0004 must be the pre-redeemed fixture (Card 2's already_used leg)"

  echo "· supabase_realtime publication membership (§7.1):"
  PUB="$(echo "select schemaname||'.'||tablename from pg_publication_tables where pubname='supabase_realtime' order by 1;" | psqlq)"
  echo "$PUB" | sed 's/^/    /'
  echo "$PUB" | grep -qx 'public.codes' || fail "public.codes is not in the supabase_realtime publication"
}

if [ "$MODE" = "assert" ]; then
  assert_all
  printf '\n✅ VERDICT: GREEN — every structural claim present by name (assert-only; nothing applied)\n'
  exit 0
fi

echo
echo "── staging fresh state: dropping this card's objects only (throwaway substrate) ──"
reset_bare
echo "  bare."

echo
echo "── apply #1 (fresh: objects do not exist, the migration creates them) ──"
apply_all || fail "the migration did not apply clean on a bare substrate (fresh apply)"
assert_all

echo
echo "── apply #2 (warm: 'applies clean' must survive a re-run on its own output) ──"
apply_all || fail "the migration did not apply clean on a second run (warm apply)"
assert_all

echo
echo "── #5 leg: an operator-changed threshold SURVIVES a migration re-apply ──"
echo "update public.marketing_settings set requires_online_threshold_cents = 2500 where id = 1;" | psqlq >/dev/null
apply_all >/dev/null || fail "the migration did not apply clean over an operator-modified settings row"
MSAFTER="$(echo "select requires_online_threshold_cents from public.marketing_settings where id=1;" | psqlq)"
echo "    threshold after operator-set 2500 + re-apply: $MSAFTER"
[ "$MSAFTER" = 2500 ] || fail "re-applying the migration clobbered the operator's threshold ($MSAFTER) — #5 requires 'configurable without a migration'"
echo "update public.marketing_settings set requires_online_threshold_cents = 2000 where id = 1;" | psqlq >/dev/null
echo "    restored to the seeded default 2000"

printf '\n✅ VERDICT: GREEN — migration applies clean twice (fresh + warm), every structural claim present by name, #5 config survives re-apply\n'
exit 0
