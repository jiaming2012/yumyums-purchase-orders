# Extraction — merge-target-and-blanking-guards

Outcome: confirmed, no corrections

Approach used: a spike-owned database on `:5434` migrated by booting the built
server once, then `MergeMenuItem`'s four statements run verbatim in a DO block
with SQLSTATE + constraint captured (spike 01, three legs); and a throwaway
worktree of `dev` with the three-line blanking `UPDATE` removed from migration
0086, running the six marketing-package tests that touch 0086's tables against a
fresh Go database before and after the mutation (spike 02). Tool-recorded runs:
spike 01 exit 0 twice; spike 02 exit 1 once on a script bug (`grep -c` under
`pipefail`, fixed with `|| true`) then exit 0. Candidate input for the card, not
an adoption (NFR-6).

Confirmed: (a) B-479 — merging an unattached dish into a uuid that names no dish
runs to OK and leaves the dish AND its `daily_menu_sales` row gone (seeded 1 → 0);
(b) the same sequence for a dish a campaign names fails `23503
campaigns_admin_item_id_fkey` and the dish survives — the asymmetry that made the
review see a 500 on one shape and a 200 on the other; (c) B-480 — with
`qr_scans_subscriber_id_fkey` dropped and one `qr_scans.subscriber_id` naming
nobody, `ADD CONSTRAINT … ON DELETE SET NULL` fails `23503 qr_scans_subscriber_id_fkey`,
and after 0086's `UPDATE … SET subscriber_id = NULL` the same `ADD CONSTRAINT`
succeeds with the row present and blanked (rows=1, null=1); (d) the test gap —
with the UPDATE removed, `TestSubscriberDeleteCascadesTimelineAndBlanksScans`,
`TestCodeDeleteBlanksFirstTouch`, `TestScannedCodeDeleteIsRefused`,
`TestMigration0086ErasureBackstopDownAndUpRoundTrip`,
`TestMigration0083DownAndUpRoundTrip` and
`TestMigration0085SubscribersDownAndUpRoundTrip` all PASS on a fresh database
(6 / 0 / 0, control 6 / 0 / 0).

Learned: (1) the missing-target guard must run BEFORE the three re-point
statements, inside the same transaction, or an attached source still reaches
the FK refusal path with a different error than an unattached one — one
`SELECT … FOR SHARE` on the target makes both shapes answer 404 the same way;
(2) a migration test for the blanking step has to seed the dangling row at
version 85 (the FK does not exist there) and migrate up — seeding at 86+ is
refused by the constraint the test is about; the `db.MigrateTo(pool, 85)` idiom
`TestMigration0086ErasureBackstopDownAndUpRoundTrip` already uses is the recipe;
(3) script hygiene: a `grep -c` that may count zero needs `|| true` under
`pipefail` — the first tool-recorded run of spike 02 died on exactly that, with
the mutation applied and the mutated leg never run.

Plan change: none to scope — the card ships the `FOR SHARE` guard + 404 arm and
the 85 → up migration test as specified; its red-first recipes are these two
spikes' legs (a) and (d).
