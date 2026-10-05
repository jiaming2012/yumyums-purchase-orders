# Spikes — dish-merge-shapes-and-backstop-tests

Activity: Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> Tool-run (`night-crew spikes run`). One Go spike in a THROWAWAY worktree off `dev`
> (`../hq-worktrees/spike-k2-20261007`) against a fresh spike-owned database
> `hq_test_spike_k2_go` on `:5434` (the recipes package's `TestMain` migrates it). Never `:5433`.

## The goal, and which legs need a spike

The card (K2 — B-484, B-485, B-487; adversarial review of run `20261005`, triage T-66): a dish
merge with a missing SOURCE answers 200, a non-uuid target 500, the sentinel is matched by
substring; the `FOR SHARE` lock on the target is pinned by no test; migration 0086's `ON DELETE
SET NULL` on `campaigns_admin.item_id` / `qr_codes.item_id` is asserted at schema level only.
Three premises a script can settle now: (1) the two wrong shapes reproduce by execution through
the handler; (2) with ` FOR SHARE` removed from the target read in a worktree copy,
`TestMergeMenuItem_MissingTargetIsRefused` still PASSES — the lock is unpinned; (3) deleting a
dish that a campaign and a code name leaves both rows present with `item_id IS NULL` — the
backstop works and the card's test has a known green.

## Spike: wrong-shapes-reproduce-lock-is-unpinned-backstop-blanks

- proves: (a) B-485 — `POST /inventory/recipes/merge` with a source uuid naming no dish and a
  real target answers 200 `{"rows_re_pointed":0}`; with a target of `not-a-uuid` answers 500;
  (b) B-484 — with ` FOR SHARE` deleted from `repository.go`'s target read (worktree only),
  `TestMergeMenuItem_MissingTargetIsRefused` passes (3 subtests) on a fresh database — nothing
  pins the lock; (c) B-487 — a user, a campaign naming dish X and a code naming dish X; `DELETE
  FROM menu_items WHERE id = X` succeeds; both rows survive with `item_id IS NULL`.
- plan: worktree, fresh Go database, a throwaway `_test.go` beside `repository_test.go` for
  (a) and (c) using the package's own helpers, one `sed` mutation + `go test -run` for (b),
  restore, worktree clean at exit.
- script: .night-crew/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/dish-merge-shapes-and-backstop-tests/01-wrong-shapes-reproduce-lock-is-unpinned-backstop-blanks.sh

### Runs

- 2026-10-05T15:07:30Z · exit 1 · failed
- 2026-10-05T15:08:29Z · exit 143 · failed
- 2026-10-05T15:09:09Z · exit 1 · failed
- 2026-10-05T15:09:38Z · exit 2 · failed
- 2026-10-05T15:10:23Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **wrong-shapes-reproduce-lock-is-unpinned-backstop-blanks: passed** on its fifth run (exit 0,
  ~20 s): (a) `POST /inventory/recipes/merge` with a source uuid naming no dish → **200
  `{"rows_re_pointed":0}`**; target `not-a-uuid` → **500 `internal_error`** (`22P02` in the server
  log) — B-485's two wrong shapes, by execution; (b) control `TestMergeMenuItem_MissingTargetIsRefused`
  PASS (3 subtests); with the target read's ` FOR SHARE` removed (numstat 1/1) the same test still
  **PASS** — B-484: nothing pins the lock; (c) a campaign and a code naming dish X, `DELETE FROM
  menu_items WHERE id = X` → `campaign rows=1 item_id=<nil> | code rows=1 item_id=<nil>` — the
  0086 backstop works by behaviour, so the card's test has a known green. **The four failed lines
  above are script-side, named:** run 1 — the recipes package's `TestMain` does not migrate a
  fresh database (`drift_check_results` missing; the marketing package's does), so the spike now
  boots the built server once to migrate (the J2 recipe); run 2 — `exit 143`: `wait` on the
  TERMed server returned 143 under `set -e` (swallowed now); run 3 — `campaigns_admin.id` has no
  default (seed fixed); run 4 — ` FOR SHARE` also appears in a comment, so the mutation touched
  two lines (narrowed to the SQL line). None is a finding about the product.

## Corrections

- none agent-reached — all three premises held. Precisions carried into the card: the missing-
  source check must run AFTER the target's `FOR SHARE` inside the same transaction (the spike's
  statement order); the non-uuid guard is a parse before the query (both ids); the backstop test
  seeds `campaigns_admin.id` explicitly.

## Comebacks

- Spike-side only: `internal/recipes` tests need a migrated database (boot the server once);
  a `wait` on a TERMed child under `set -e` needs `|| true`; a `sed` mutation must target the SQL
  line, not a phrase that recurs in comments. All recorded in `_golib.sh` and the script.

## Review

- signed: operator, 2026-10-05 — covers 0 correction(s) (reviewed at the slate sitting of
  2026-10-05, slate-20261007 §4 batch sign-off; no corrections to review).
