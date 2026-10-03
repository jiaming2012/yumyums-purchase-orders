# Spikes — dish-merge-and-erasure-backstop

Activity: Activity I — Honest gates, then verify before the till (triage 20261002 follow-ups)

> Hand-run convention (see README.md in this directory). Runs on `:5434` in a spike-owned
> database `hq_test_spike_i3_20261003` migrated by the repo's own goose runner; dropped at the
> end. Never `:5433`.

## The goal, and which legs need a spike

The card (I3 — decision 194, ledger T-62): migration `0083`'s `campaigns_admin.item_id` and
`qr_codes.item_id` reference `menu_items(id)` with no `ON DELETE`, so `recipes.MergeMenuItem`
(re-point recipes, then `DELETE FROM menu_items`) fails `23503` once any campaign references
the dish — API-only today, no frontend calls the merge. Decided: teach the merge to re-point
`campaigns_admin` and `qr_codes` to the surviving dish (house convention), AND add a
blank-on-delete backstop so a future table that forgets the merge path degrades to an empty
label rather than an opaque 500. The same shape goes on `0085`'s subscriber tables, which is
what makes a right-to-erasure request servable by a plain `DELETE` — today
`subscriber_events.subscriber_id` and `subscribers.source_short` block it both ways.
Behaviour: a dish rename never breaks a campaign, a campaign never blocks a dish rename, and
deleting a subscriber removes their timeline without manual table-by-table surgery.

Two premises a script can settle now: (1) all three blocks reproduce on a freshly migrated
database by the exact statements the code runs; (2) the candidate backstop shape (`SET NULL`
on the three nullable links, `CASCADE` on the subscriber timeline — an engineer-level call,
since a timeline row cannot be blanked) makes every one of those deletes succeed.

## Spike: merge-and-erasure-blocked-by-fks

- proves: on a migrated `0085` schema, (a) a campaign referencing dish A makes
  `DELETE FROM menu_items WHERE id = A` fail `23503`; (b) `DELETE FROM subscribers` for a
  subscriber with one event fails `23503`; (c) `DELETE FROM qr_codes` for a code a subscriber
  first-touched fails `23503`.
- plan: create + migrate the spike DB, seed the minimum rows (a user, two dishes, a campaign
  with its code, a subscriber with an event), run the three deletes, assert each SQLSTATE.
- script: .night-crew/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/dish-merge-and-erasure-backstop/01-merge-and-erasure-blocked-by-fks.sh

### Runs

- 2026-10-03T13:37:21Z · exit 0 · passed

## Spike: backstop-shape-makes-the-deletes-plain

- proves: after `ALTER TABLE` to `ON DELETE SET NULL` on `campaigns_admin.item_id`,
  `qr_codes.item_id` and `subscribers.source_short`, and `ON DELETE CASCADE` on
  `subscriber_events.subscriber_id`, the same three deletes succeed: the campaign and code
  survive with a null item, the subscriber's events are gone with the subscriber, and the
  code's deletion blanks the subscriber's first-touch link.
- plan: same DB, apply the candidate ALTERs (the card's migration `0086` draft), re-run.
- script: .night-crew/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/dish-merge-and-erasure-backstop/02-backstop-shape-makes-the-deletes-plain.sh

### Runs

- 2026-10-03T13:37:29Z · exit 0 · passed

## Verdict (hand-run 2026-10-02)

- **merge-and-erasure-blocked-by-fks: passed** — exit 0, first run, 7 s (scratch DB migrated
  to goose 85 by booting the built server once, dropped by trap): (a) the merge sequence
  verbatim from `repository.go:151-162` → `23503 campaigns_admin_item_id_fkey` (the campaign's
  RI trigger fires first; `qr_codes_item_id_fkey` would block next); (b) `DELETE FROM
  subscribers` → `23503 subscriber_events_subscriber_id_fkey`; (c) `DELETE FROM qr_codes` for a
  first-touched code → `23503 subscribers_source_short_fkey`; (d) beyond the ledger: a code
  with one `qr_scans` row and no subscriber → `23503 qr_scans_short_fkey`.
- **backstop-shape-makes-the-deletes-plain: passed** — exit 0, first run, 10 s: after the four
  ledger ALTERs (`confdeltype` n/n/n/c read back from `pg_constraint`) the merge succeeds with
  campaign and code surviving `item_id NULL`; the subscriber and its events are gone with one
  DELETE; a code delete blanks the re-seeded subscriber's `source_short`. A SCANNED code stays
  undeletable until a fifth ALTER (`qr_scans.short … ON DELETE CASCADE`), which the spike also
  ran: scans gone, subscriber blanked. Constraint names are Postgres' auto-generated ones
  (`<table>_<column>_fkey`); the exact statements are in the script's output and the
  extraction record.

## Corrections

- **Agent-reached correction 1 — a fourth block the ledger did not list.** `qr_scans.short →
  qr_codes(short)` (migration 0083) is `NOT NULL` with no `ON DELETE`, so the four candidate
  ALTERs leave any code that has ever been scanned undeletable. **Sitting's engineer-level call,
  stated for the operator's review:** codes are DEACTIVATED, never deleted — the H1 design
  already pauses a campaign and renders "this offer has ended" for an inactive code, and a scan
  history is attribution evidence — so the card keeps the ledger's four ALTERs and does NOT
  cascade `qr_scans`. Deleting a code remains a refusal, which is what the backstop is for.
- **Agent-reached correction 2 — subscriber erasure leaves a dangling id.** `qr_scans.subscriber_id`
  is a bare `uuid null` with no FK, so a subscriber DELETE never touches `qr_scans` and leaves
  the id behind. **Sitting's call:** the card adds the missing FK
  `qr_scans.subscriber_id → subscribers(id) ON DELETE SET NULL` as a fifth ALTER — the same
  blank-on-delete shape as the other three — so "one DELETE erases the subscriber" is true of
  every table that names them.
- Precision: the task's seed facts for `users` were stale (`display_name` → `first_name` /
  `last_name` in 0017; `role` → `roles TEXT[]` in 0023); the scripts seed the real shape.

## Comebacks

- none recorded — `recipes` and `daily_menu_sales` already cascade on `menu_items` and never
  block the merge; the H1 G6's reading "no `ON DELETE` on either FK" was exact.

## Review

- signed: operator, 2026-10-02 — covers 2 correction(s) (reviewed at the slate sitting of 2026-10-02, §4 batch sign-off, with the sitting's call for each in view)
