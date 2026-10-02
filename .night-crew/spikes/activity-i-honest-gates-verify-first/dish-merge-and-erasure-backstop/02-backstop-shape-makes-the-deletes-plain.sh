#!/usr/bin/env bash
# 02-backstop-shape-makes-the-deletes-plain.sh — spike for card I3 (decision 194, ledger
# T-62): the candidate backstop (migration 0086 draft) makes the same deletes plain:
#   ON DELETE SET NULL on campaigns_admin.item_id, qr_codes.item_id, subscribers.source_short;
#   ON DELETE CASCADE on subscriber_events.subscriber_id (a timeline row cannot be blanked).
# Then: the merge sequence succeeds and the campaign + code survive with item_id NULL; the
# subscriber delete takes its events with it; deleting the code blanks the (re-seeded)
# subscriber's source_short.
# BEYOND THE LEDGER: qr_scans.short → qr_codes(short) is NOT NULL and has no ON DELETE, so
# the four ALTERs above still leave a scanned code undeletable. This script proves that
# (expects 23503 qr_scans_short_fkey after the four ALTERs), then applies a fifth candidate
# — ON DELETE CASCADE on qr_scans.short — and proves the code delete goes through with it.
# Constraint names are read from pg_constraint, not guessed; the ALTERs that worked are
# printed verbatim at the end as the 0086 draft.
# DB: localhost:5434 only, scratch db hq_test_spike_i3_20261003, created + migrated by
# running backend/cmd/server once (embedded goose), dropped on exit. See lib.sh.
# 🛑 exit 0 = every assertion holds; exit 1 = any delete still blocked or a row in the
# wrong state afterwards; exit 2 = could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
. "$SCRIPT_DIR/lib.sh"
T0=$(date +%s)

spike_db_create
spike_db_migrate
spike_seed

# --- the four ledger ALTERs, names from pg_constraint ------------------------------
FK_CAMP=$(fk_name campaigns_admin item_id);      [ -n "$FK_CAMP" ] || fail "no FK on campaigns_admin.item_id"
FK_CODE=$(fk_name qr_codes item_id);             [ -n "$FK_CODE" ] || fail "no FK on qr_codes.item_id"
FK_SUB=$(fk_name subscribers source_short);      [ -n "$FK_SUB" ]  || fail "no FK on subscribers.source_short"
FK_EV=$(fk_name subscriber_events subscriber_id);[ -n "$FK_EV" ]   || fail "no FK on subscriber_events.subscriber_id"
FK_SCAN=$(fk_name qr_scans short);               [ -n "$FK_SCAN" ] || fail "no FK on qr_scans.short"
echo "# constraint names: $FK_CAMP $FK_CODE $FK_SUB $FK_EV (beyond ledger: $FK_SCAN)"

ALTERS_LEDGER=$(cat <<SQL
ALTER TABLE campaigns_admin   DROP CONSTRAINT $FK_CAMP, ADD CONSTRAINT $FK_CAMP FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
ALTER TABLE qr_codes          DROP CONSTRAINT $FK_CODE, ADD CONSTRAINT $FK_CODE FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
ALTER TABLE subscribers       DROP CONSTRAINT $FK_SUB,  ADD CONSTRAINT $FK_SUB  FOREIGN KEY (source_short)  REFERENCES qr_codes(short) ON DELETE SET NULL;
ALTER TABLE subscriber_events DROP CONSTRAINT $FK_EV,   ADD CONSTRAINT $FK_EV   FOREIGN KEY (subscriber_id) REFERENCES subscribers(id) ON DELETE CASCADE;
SQL
)
ALTER_SCANS="ALTER TABLE qr_scans          DROP CONSTRAINT $FK_SCAN, ADD CONSTRAINT $FK_SCAN FOREIGN KEY (short)         REFERENCES qr_codes(short) ON DELETE CASCADE;"
pq -c "$ALTERS_LEDGER" || fail "the four ledger ALTERs did not apply"
echo "# applied the four ledger ALTERs; FK delete rules now:"
pq -c "select conrelid::regclass as tbl, conname, confdeltype from pg_constraint where conname in ('$FK_CAMP','$FK_CODE','$FK_SUB','$FK_EV','$FK_SCAN') order by 1"

# (a) the merge sequence, verbatim from recipes.MergeMenuItem
try_sql "(a) merge A→B: UPDATE recipes; DELETE menu_items A" \
  "update recipes set menu_item_id = '$DISH_B', updated_at = now() where menu_item_id = '$DISH_A';
   delete from menu_items where id = '$DISH_A';"
[ "$TRY_STATE" = OK ] || fail "(a) merge still blocked: $TRY_STATE $TRY_CONSTRAINT"
[ "$(pqa -c "select count(*) from menu_items where id='$DISH_A'")" = 0 ] || fail "(a) dish A still present"
[ "$(pqa -c "select count(*) filter (where item_id is null) || '/' || count(*) from campaigns_admin where id='$CAMP_ID'")" = 1/1 ] || fail "(a) campaign missing or item_id not NULL"
[ "$(pqa -c "select count(*) filter (where item_id is null) || '/' || count(*) from qr_codes where short='$SHORT'")" = 1/1 ] || fail "(a) code missing or item_id not NULL"
echo "   campaign $CAMP_ID and code $SHORT survive with item_id NULL"

# (b) erase the subscriber; events go with it
try_sql "(b) DELETE subscribers $SUB_ID" "delete from subscribers where id = '$SUB_ID';"
[ "$TRY_STATE" = OK ] || fail "(b) subscriber delete still blocked: $TRY_STATE $TRY_CONSTRAINT"
[ "$(pqa -c "select count(*) from subscribers where id='$SUB_ID'")" = 0 ] || fail "(b) subscriber still present"
[ "$(pqa -c "select count(*) from subscriber_events where subscriber_id='$SUB_ID'")" = 0 ] || fail "(b) subscriber_events rows survived the subscriber"
echo "   subscriber and its events gone"

# (c) re-seed the subscriber, delete the code: source_short blanks
spike_seed_subscriber
try_sql "(c) DELETE qr_codes $SHORT (re-seeded subscriber first-touch, no scans)" "delete from qr_codes where short = '$SHORT';"
[ "$TRY_STATE" = OK ] || fail "(c) code delete still blocked: $TRY_STATE $TRY_CONSTRAINT"
[ "$(pqa -c "select count(*) from qr_codes where short='$SHORT'")" = 0 ] || fail "(c) code still present"
[ "$(pqa -c "select count(*) filter (where source_short is null) || '/' || count(*) from subscribers where id='$SUB_ID'")" = 1/1 ] || fail "(c) subscriber missing or source_short not NULL"
echo "   subscriber survives with source_short NULL"

# (d) beyond the ledger: a scanned code is still undeletable after the four ALTERs
pq -c "insert into qr_codes (short, campaign_id, channel, created_by) values ('SPKE24', '$CAMP_ID', 'flyer', '$U_ID');
       insert into qr_scans (short) values ('SPKE24');
       update subscribers set source_short = 'SPKE24' where id = '$SUB_ID';" >/dev/null
try_sql "(d1) DELETE qr_codes SPKE24 (one scan) with only the four ledger ALTERs" "delete from qr_codes where short = 'SPKE24';"
[ "$TRY_STATE" = 23503 ] && [ "$TRY_CONSTRAINT" = "$FK_SCAN" ] || fail "(d1) expected 23503 $FK_SCAN — premise about qr_scans changed: $TRY_STATE $TRY_CONSTRAINT"
pq -c "$ALTER_SCANS" || fail "the qr_scans ALTER did not apply"
try_sql "(d2) DELETE qr_codes SPKE24 after ON DELETE CASCADE on qr_scans.short" "delete from qr_codes where short = 'SPKE24';"
[ "$TRY_STATE" = OK ] || fail "(d2) code delete still blocked: $TRY_STATE $TRY_CONSTRAINT"
[ "$(pqa -c "select count(*) from qr_scans where short='SPKE24'")" = 0 ] || fail "(d2) qr_scans rows survived the code"
[ "$(pqa -c "select count(*) filter (where source_short is null) || '/' || count(*) from subscribers where id='$SUB_ID'")" = 1/1 ] || fail "(d2) subscriber source_short not blanked"
echo "   scans gone with the code, subscriber source_short NULL"

echo "# wall-clock: $(( $(date +%s) - T0 ))s"
echo "# ===== migration 0086 draft — the ALTERs that worked ====="
echo "$ALTERS_LEDGER"
echo "-- beyond the ledger (needed for erasing a code that has been scanned; qr_scans.short is NOT NULL so CASCADE is the only blank-on-delete shape):"
echo "$ALTER_SCANS"
echo "✅ GREEN — SET NULL ×3 + CASCADE ×1 make the merge, the subscriber erasure and the code erasure plain; CORRECTION: a scanned code also needs qr_scans.short ON DELETE CASCADE (or the card accepts that scanned codes are deactivated, not deleted)"
