#!/usr/bin/env bash
# 01-merge-and-erasure-blocked-by-fks.sh — spike for card I3 (decision 194, ledger T-62):
# on a freshly migrated schema (goose through 0085) the three deletes the card is about
# are blocked by plain FKs with no ON DELETE:
#   (a) recipes.MergeMenuItem's exact sequence (UPDATE recipes → DELETE FROM menu_items A)
#       fails 23503 once a campaign (campaigns_admin.item_id) or a code (qr_codes.item_id)
#       references dish A;
#   (b) DELETE FROM subscribers for a subscriber with one event fails 23503
#       (subscriber_events.subscriber_id);
#   (c) DELETE FROM qr_codes for a code a subscriber first-touched fails 23503
#       (subscribers.source_short);
#   (d) BEYOND THE LEDGER — qr_scans.short → qr_codes(short) (0083) ALSO blocks deleting a
#       code once it has been scanned; proven on a second code with one scan and no
#       subscriber, so the constraint name is unambiguous.
# DB: localhost:5434 only, scratch db hq_test_spike_i3_20261003, created + migrated by
# running backend/cmd/server once (embedded goose), dropped on exit. See lib.sh.
# 🛑 exit 0 = all four deletes fail 23503 on the named constraints; exit 1 = any delete
# succeeded or failed with a different SQLSTATE/constraint; exit 2 = could not run.
set -euo pipefail
SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
. "$SCRIPT_DIR/lib.sh"
T0=$(date +%s)

spike_db_create
spike_db_migrate
spike_seed

echo "# FKs onto the four tables as migrated (confdeltype a = NO ACTION):"
pq -c "select conrelid::regclass as tbl, conname, confrelid::regclass as refs, confdeltype
       from pg_constraint where contype='f' and confrelid in ('menu_items'::regclass,'qr_codes'::regclass,'subscribers'::regclass) order by 1,2"

# (a) the merge, verbatim from backend/internal/recipes/repository.go MergeMenuItem
try_sql "(a) merge A→B: UPDATE recipes; DELETE menu_items A" \
  "update recipes set menu_item_id = '$DISH_B', updated_at = now() where menu_item_id = '$DISH_A';
   delete from menu_items where id = '$DISH_A';"
A_STATE=$TRY_STATE; A_CON=$TRY_CONSTRAINT
[ "$(pqa -c "select count(*) from menu_items where id='$DISH_A'")" = 1 ] || fail "(a) dish A was deleted — the merge was not blocked"

# (b) erase the subscriber
try_sql "(b) DELETE subscribers $SUB_ID" "delete from subscribers where id = '$SUB_ID';"
B_STATE=$TRY_STATE; B_CON=$TRY_CONSTRAINT

# (c) erase the code a subscriber first-touched
try_sql "(c) DELETE qr_codes $SHORT (subscriber first-touch)" "delete from qr_codes where short = '$SHORT';"
C_STATE=$TRY_STATE; C_CON=$TRY_CONSTRAINT

# (d) beyond the ledger: a second code with a scan and no subscriber
pq -c "insert into qr_codes (short, campaign_id, channel, created_by) values ('SPKE24', '$CAMP_ID', 'flyer', '$U_ID');
       insert into qr_scans (short) values ('SPKE24');" >/dev/null
try_sql "(d) DELETE qr_codes SPKE24 (one qr_scans row, no subscriber)" "delete from qr_codes where short = 'SPKE24';"
D_STATE=$TRY_STATE; D_CON=$TRY_CONSTRAINT

echo "# wall-clock: $(( $(date +%s) - T0 ))s"
[ "$A_STATE" = 23503 ] || fail "(a) expected 23503, got $A_STATE"
case "$A_CON" in campaigns_admin_item_id_fkey|qr_codes_item_id_fkey) ;; *) fail "(a) expected campaigns_admin/qr_codes item_id fkey, got $A_CON";; esac
[ "$B_STATE" = 23503 ] && [ "$B_CON" = subscriber_events_subscriber_id_fkey ] || fail "(b) expected 23503 subscriber_events_subscriber_id_fkey, got $B_STATE $B_CON"
[ "$C_STATE" = 23503 ] && [ "$C_CON" = subscribers_source_short_fkey ] || fail "(c) expected 23503 subscribers_source_short_fkey, got $C_STATE $C_CON"
[ "$D_STATE" = 23503 ] && [ "$D_CON" = qr_scans_short_fkey ] || fail "(d) expected 23503 qr_scans_short_fkey, got $D_STATE $D_CON"
echo "✅ GREEN — all three ledger premises reproduce (a=$A_CON b=$B_CON c=$C_CON); CORRECTION: (d) qr_scans_short_fkey is a fourth block on code erasure the ledger does not list"
