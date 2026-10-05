#!/usr/bin/env bash
# 01-missing-target-deletes-dish-and-blank-step-is-load-bearing.sh — spike for card J2
# (merge-target-and-blanking-guards; B-479 + B-480, ledger T-64). On a freshly migrated schema:
#   (a) B-479: MergeMenuItem's exact statement sequence with target = a uuid naming NO dish and
#       source = an unattached dish runs to OK — the dish and its daily_menu_sales row are gone;
#   (b) the same sequence for a dish a campaign names fails 23503 campaigns_admin_item_id_fkey
#       (the 500-and-rollback the review saw — the asymmetry that hides the bug);
#   (c) B-480: with qr_scans_subscriber_id_fkey dropped and one qr_scans row whose subscriber_id
#       names nobody, 0086's ADD CONSTRAINT fails 23503; after 0086's UPDATE … SET subscriber_id
#       = NULL the same ADD CONSTRAINT succeeds and the row survives, blanked.
# DB: localhost:5434 only, scratch hq_test_spike_j2_20261005, migrated by running
# backend/cmd/server once (embedded goose), dropped on exit. See lib.sh.
# 🛑 exit 0 = all three legs behave as stated; exit 1 = any leg differs; exit 2 = could not run.
set -euo pipefail
. "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)/lib.sh"
spike_preflight
spike_db_create
spike_db_migrate
spike_seed

leg "(a) B-479 — merge unattached dish A into $BOGUS (names no dish)"
[ "$(pqa -c "select count(*) from menu_items where id='$BOGUS'")" = 0 ] || cannot_run "the bogus target exists"
try_sql "(a) MergeMenuItem sequence A→BOGUS" "$(merge_sql "$BOGUS" "$DISH_A")"
A_STATE=$TRY_STATE
A_DISH="$(pqa -c "select count(*) from menu_items where id='$DISH_A'")"
A_SALES="$(pqa -c "select count(*) from daily_menu_sales where menu_item_id='$DISH_A'")"
echo "#   after: dish A rows=$A_DISH, its daily_menu_sales rows=$A_SALES (seeded 1)"

leg "(b) B-479's asymmetry — merge dish C (a campaign names it) into $BOGUS"
try_sql "(b) MergeMenuItem sequence C→BOGUS" "$(merge_sql "$BOGUS" "$DISH_C")"
B_STATE=$TRY_STATE; B_CON=$TRY_CONSTRAINT
C_DISH="$(pqa -c "select count(*) from menu_items where id='$DISH_C'")"
echo "#   after: dish C rows=$C_DISH"

leg "(c) B-480 — the blanking UPDATE in 0086 is load-bearing at deploy"
pq -c "alter table qr_scans drop constraint qr_scans_subscriber_id_fkey" >/dev/null
DANGLING=77777777-7777-4777-8777-777777777777
pq -c "insert into qr_scans (short, subscriber_id) values ('$SHORT', '$DANGLING')" >/dev/null
[ "$(pqa -c "select count(*) from subscribers where id='$DANGLING'")" = 0 ] || cannot_run "the dangling id names a subscriber"
ADD_FK="alter table qr_scans add constraint qr_scans_subscriber_id_fkey foreign key (subscriber_id) references subscribers(id) on delete set null;"
try_sql "(c1) ADD CONSTRAINT with one dangling subscriber_id, NO blanking step" "$ADD_FK"
C1_STATE=$TRY_STATE; C1_CON=$TRY_CONSTRAINT
try_sql "(c2) 0086's UPDATE … SET subscriber_id = NULL, then the same ADD CONSTRAINT" \
  "update qr_scans s set subscriber_id = null where s.subscriber_id is not null and not exists (select 1 from subscribers b where b.id = s.subscriber_id);
   $ADD_FK"
C2_STATE=$TRY_STATE
C_ROWS="$(pqa -c "select count(*) from qr_scans where short='$SHORT'")"
C_NULL="$(pqa -c "select count(*) from qr_scans where short='$SHORT' and subscriber_id is null")"
echo "#   after: qr_scans rows for $SHORT=$C_ROWS, of which subscriber_id IS NULL=$C_NULL"

echo "# wall-clock: $(( $(date +%s) - T0 ))s"
[ "$A_STATE" = OK ] || fail "(a) expected the merge into a missing target to run to OK, got $A_STATE"
[ "$A_DISH" = 0 ] && [ "$A_SALES" = 0 ] || fail "(a) dish A survived (rows=$A_DISH sales=$A_SALES) — the merge was refused or did not cascade"
[ "$B_STATE" = 23503 ] && [ "$B_CON" = campaigns_admin_item_id_fkey ] || fail "(b) expected 23503 campaigns_admin_item_id_fkey, got $B_STATE $B_CON"
[ "$C_DISH" = 1 ] || fail "(b) dish C was deleted although the merge was refused"
[ "$C1_STATE" = 23503 ] && [ "$C1_CON" = qr_scans_subscriber_id_fkey ] || fail "(c1) expected ADD CONSTRAINT to fail 23503 qr_scans_subscriber_id_fkey on a dangling id, got $C1_STATE $C1_CON"
[ "$C2_STATE" = OK ] || fail "(c2) expected the UPDATE + ADD CONSTRAINT to succeed, got $C2_STATE"
[ "$C_ROWS" = 1 ] && [ "$C_NULL" = 1 ] || fail "(c2) expected the scan row to survive blanked (rows=$C_ROWS null=$C_NULL)"
echo "✅ GREEN — (a) B-479: an unattached dish merged into a missing target is deleted with its sales row (200, not 404);"
echo "   (b) a dish a campaign names is refused 23503 (the review's 500) — the asymmetry; (c) B-480: without 0086's UPDATE"
echo "   the ADD CONSTRAINT fails 23503 on one dangling id, with it the row survives blanked. Both premises hold."
