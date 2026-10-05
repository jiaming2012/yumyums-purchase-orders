# Extraction — dish-merge-shapes-and-backstop-tests

Outcome: confirmed, no corrections

Approach used: a throwaway worktree of `dev` with a fresh Go database on `:5434`
migrated by booting the built server once (the recipes package's `TestMain` does not
migrate), a throwaway `_test.go` beside `repository_test.go` driving the merge handler
through `mountRouter`/`doJSON` and deleting a dish a campaign and a code name, then a
one-line `sed` mutation removing ` FOR SHARE` from the target read and re-running the
existing missing-target test. Tool-recorded runs: four exit-non-zero lines, all script
bugs named in the ledger, then exit 0. Candidate input for the card, not an adoption.

Confirmed: (a) B-485 — a merge whose source names no dish answers 200
`{"rows_re_pointed":0}` and a non-uuid target answers 500 (`22P02`); (b) B-484 — with
` FOR SHARE` removed, `TestMergeMenuItem_MissingTargetIsRefused` still passes its three
subtests on a fresh database — the lock is unpinned; (c) B-487 — deleting a dish that a
campaign and a code reference leaves both rows present with `item_id` NULL; the
backstop works by behaviour.

Learned: (1) the recipes package needs a migrated database before any test runs — the
card's new tests inherit that; (2) the lock test must open a SECOND connection and race
a delete against the merge, because a single-connection test cannot observe `FOR
SHARE`; (3) `campaigns_admin.id` carries no default, so seeds name it; (4) the
missing-source check belongs after the target read inside the same transaction.

Plan change: none to scope — the card ships the 404/400 shapes, `errors.Is` on both
sentinels, the two-connection lock test and the behavioural backstop test; red-first
recipes are this spike's legs (a) and (b), and a `NO ACTION` mutation of 0086 for (c).
