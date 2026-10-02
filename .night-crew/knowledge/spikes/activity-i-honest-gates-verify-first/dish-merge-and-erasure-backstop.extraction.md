# Extraction — dish-merge-and-erasure-backstop

Outcome: confirmed, with two corrections

Approach used: a spike-owned database on `:5434` migrated to goose 85 by booting
the built server once; the three deletes (and a fourth, found) each run in their
own transaction with SQLSTATE + constraint name captured; then the candidate
ALTERs applied and the deletes re-run. Two spikes, both exit 0 on the first run.
Candidate input for the card, not an adoption (NFR-6).

Confirmed: all three blocks reproduce by the exact statements the code runs
(`campaigns_admin_item_id_fkey`, `subscriber_events_subscriber_id_fkey`,
`subscribers_source_short_fkey`); `SET NULL` ×3 + `CASCADE` on the timeline makes
the merge, the subscriber erasure and an un-scanned code's deletion plain. The
migration 0086 draft, verbatim from the run (constraint names read from
`pg_constraint`):

    ALTER TABLE campaigns_admin   DROP CONSTRAINT campaigns_admin_item_id_fkey,         ADD CONSTRAINT campaigns_admin_item_id_fkey         FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
    ALTER TABLE qr_codes          DROP CONSTRAINT qr_codes_item_id_fkey,                ADD CONSTRAINT qr_codes_item_id_fkey                FOREIGN KEY (item_id)       REFERENCES menu_items(id)  ON DELETE SET NULL;
    ALTER TABLE subscribers       DROP CONSTRAINT subscribers_source_short_fkey,        ADD CONSTRAINT subscribers_source_short_fkey        FOREIGN KEY (source_short)  REFERENCES qr_codes(short) ON DELETE SET NULL;
    ALTER TABLE subscriber_events DROP CONSTRAINT subscriber_events_subscriber_id_fkey, ADD CONSTRAINT subscriber_events_subscriber_id_fkey FOREIGN KEY (subscriber_id) REFERENCES subscribers(id) ON DELETE CASCADE;

Learned: (1) `qr_scans.short → qr_codes(short)` is a fourth block — a scanned code
is undeletable under the four ALTERs; the sitting's call is that codes are
deactivated, never deleted, so the card does not cascade scan history. (2)
`qr_scans.subscriber_id` has no FK, so subscriber erasure leaves a dangling id; the
card adds `qr_scans.subscriber_id → subscribers(id) ON DELETE SET NULL` as a fifth
ALTER. (3) `users` at goose 85 is `first_name`/`last_name`/`roles TEXT[]`.

Plan change: the card's migration carries five ALTERs, not four (the fifth is the
`qr_scans.subscriber_id` FK), and its done_when gains the assertion that a
subscriber DELETE blanks their `qr_scans.subscriber_id`; a scanned code stays
undeletable by design and the card's test asserts THAT, not a cascade.
