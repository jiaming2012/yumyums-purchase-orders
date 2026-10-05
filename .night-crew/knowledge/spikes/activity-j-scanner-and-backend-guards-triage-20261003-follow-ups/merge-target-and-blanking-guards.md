# Spikes — merge-target-and-blanking-guards

Activity: Activity J — Scanner and backend guards (triage 20261003 follow-ups)

> Tool-run (`night-crew spikes run`). Spike 1 runs on `:5434` in a spike-owned database
> `hq_test_spike_j2_20261005` migrated by the repo's own goose runner (the built server, run once
> to "database migrations applied successfully"); dropped at exit. Spike 2 mutates migration 0086
> in a THROWAWAY git worktree off `dev` (`../hq-worktrees/spike-j2-20261005`) and runs the Go
> tests against `hq_test_spike_j2_go` on `:5434`. Never `:5433`.

## The goal, and which legs need a spike

The card (J2 — B-479, B-480; adversarial review of run 20261003, ledger T-64): `recipes.MergeMenuItem`
never checks the target dish exists, so a merge into a bogus id with an unattached source deletes
the source dish and its `daily_menu_sales` rows and answers 200; and migration 0086's blanking of
dangling `qr_scans.subscriber_id` values is load-bearing at deploy (`ADD CONSTRAINT` fails
without it) yet no test pins it.

Two premises a script can settle now: (1) both defects reproduce by execution on today's schema —
the merge's verbatim statement sequence deletes an unattached dish into a missing target, and the
same sequence is refused `23503` once a campaign names the dish (the asymmetry B-479 reports); and
`ADD CONSTRAINT qr_scans_subscriber_id_fkey` fails `23503` on a table holding one dangling id
while the UPDATE makes it pass; (2) the test gap is real — with the UPDATE removed from 0086, the
erasure tests and every migration round-trip in `internal/marketing` still pass on a fresh database.

## Spike: missing-target-deletes-dish-and-blank-step-is-load-bearing

- proves: (a) `MergeMenuItem`'s four statements (verbatim) with target = a uuid naming no dish and
  source = an unattached dish with one `daily_menu_sales` row run to OK and leave the dish and its
  sales row gone; (b) the same sequence for a dish a campaign names fails `23503
  campaigns_admin_item_id_fkey` (the 500 the review saw); (c) with the FK dropped and one
  `qr_scans` row whose `subscriber_id` names nobody, `ADD CONSTRAINT qr_scans_subscriber_id_fkey
  … ON DELETE SET NULL` fails `23503`, and after 0086's `UPDATE … SET subscriber_id = NULL` the
  same `ADD CONSTRAINT` succeeds with the row still present and blanked.
- plan: create + migrate the spike DB, seed a user / dishes / a campaign / a code / a sales row,
  run the three legs through `try_sql` (SQLSTATE + constraint name captured), print the counts.
- script: .night-crew/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/merge-target-and-blanking-guards/01-missing-target-deletes-dish-and-blank-step-is-load-bearing.sh

### Runs
- 2026-10-04T22:56:41Z · exit 0 · passed
- 2026-10-04T22:58:27Z · exit 0 · passed

## Spike: blanking-step-removed-leaves-tests-green

- proves: in a throwaway worktree of `dev`, with the three-line `UPDATE qr_scans … SET
  subscriber_id = NULL` removed from `0086_merge_repoint_and_erasure_backstop.sql`, the four
  erasure tests and the three migration round-trip tests in `internal/marketing` all PASS on a
  fresh database (control on the unmutated worktree: PASS too) — no test pins the step.
- plan: worktree, control run against a fresh `hq_test_spike_j2_go`, mutate with a 0/3 numstat
  check, drop + recreate the database so the mutated 0086 is what migrates, re-run the same
  tests, restore, remove the worktree.
- script: .night-crew/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/merge-target-and-blanking-guards/02-blanking-step-removed-leaves-tests-green.sh

### Runs
- 2026-10-04T22:56:51Z · exit 1 · failed
- 2026-10-04T22:59:09Z · exit 0 · passed

## Verdict (tool-run 2026-10-04)

- **missing-target-deletes-dish-and-blank-step-is-load-bearing: passed** — exit 0 twice (~20 s):
  (a) the merge sequence into a uuid naming no dish ran to OK; dish A rows 1 → 0, its
  `daily_menu_sales` rows 1 → 0 (B-479 reproduced — a 200 that deletes); (b) the same sequence
  for the dish a campaign names → `23503 campaigns_admin_item_id_fkey`, dish C survives (the
  review's 500 — the asymmetry); (c1) with the FK dropped and one dangling `subscriber_id`,
  `ADD CONSTRAINT` → `23503 qr_scans_subscriber_id_fkey`; (c2) after 0086's UPDATE the same
  `ADD CONSTRAINT` → OK, the scan row survives blanked (rows=1, null=1). B-480's step is
  load-bearing at deploy.
- **blanking-step-removed-leaves-tests-green: passed** on its second run (exit 0, 45 s) — control
  6 pass / 0 fail / 0 skip on a fresh `hq_test_spike_j2_go`; with the three-line UPDATE removed
  (numstat 0/3, zero `SET subscriber_id = NULL` left in the Up) and a FRESH database so the
  mutated 0086 is what migrates: **6 pass / 0 fail / 0 skip** — no test pins the step. The first
  run's `exit 1 · failed` line above is a script bug, not a finding: `grep -c` exits 1 on a zero
  count and under `pipefail` that aborted the script after the mutate leg, before the mutated run;
  fixed with `|| true`, re-run clean.

## Corrections

- none agent-reached — both premises held as the ledger states them. Precision carried into the
  card: the dangling row for the migration test must be seeded at version 85 (`db.MigrateTo(pool,
  85)`), because at 86+ the constraint the test is about refuses the seed.

## Comebacks

- Script hygiene (spike-side, not a product gap): `grep -c` + `pipefail` on a zero count. Recorded
  in the script's comment; the pattern to copy is `$(grep -c … || true)`.

## Review

- signed: operator, 2026-10-04 — covers 0 correction(s) (reviewed at the slate sitting of 2026-10-04, §4 batch sign-off; no corrections to review; the
  script-side comeback stated).
