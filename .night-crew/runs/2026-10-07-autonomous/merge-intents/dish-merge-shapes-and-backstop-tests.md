# Merge intent — dish-merge-shapes-and-backstop-tests (K2, run 20261007)

Card: the dish-merge endpoint (`POST /api/v1/inventory/recipes/merge`, API-only — no screen
calls it) answers honestly for every wrong input, its lock on the surviving dish is pinned by a
test that races a delete, and the deploy step that blanks orphaned campaign / code references
(migration 0086) is pinned by a test that deletes a dish and reads the blanks back. BACKLOG
B-484 (the lock is pinned by no test), B-485 (a merge with a missing source answers 200, a
malformed id answers 500), B-487 (the blank-on-delete backstop is asserted at schema level only).

## Shared files I touch (outside my own area)

- `backend/go.mod` — one line: `github.com/google/uuid v1.6.0` becomes a DIRECT requirement.
  The card's spec names `uuid.Parse`; the module was already in the build graph at this version
  (`go list -m` → v1.6.0, both hashes already in `go.sum`), so no new code enters the image and
  `go.sum` does not change. Outside the card's listed footprint; not a park (mechanism, decided
  under delegation/P-1 — agents decide implementation details).
- `backend/internal/marketing/erasure_test.go` — ONE new test appended,
  `TestMigration0086DishDeleteBlanksCampaignAndCode`. The four existing erasure tests are not
  edited.
- `.night-crew/knowledge/roadmap.md` — one word on this card's Activity K line,
  `PLANNED` → `LANDED`. Append-only otherwise.

My own area: `backend/internal/recipes/repository.go`, `handler.go`, `repository_test.go`.

## What must survive any merge

- In `repository.go`: the four sentinels `ErrMergeTargetNotFound`, `ErrMergeSourceNotFound`,
  `ErrMergeIntoSelf`, `ErrBadID`; both ids parsed with `uuid.Parse` BEFORE the transaction; the
  statement ORDER inside the transaction — the target's `FOR SHARE` read first, then the
  source's `FOR UPDATE` read, then the three re-points, then the delete.
- In `handler.go`: the four `errors.Is` arms (404 `target_not_found`, 404 `source_not_found`,
  400 `bad_id`, 400 `cannot_merge_into_self`). No `strings.Contains` on an error message.
- The three new recipes tests (`TestMergeMenuItem_MissingSourceIs404`,
  `TestMergeMenuItem_BadIDIs400`, `TestMergeMenuItem_LockHoldsAgainstConcurrentTargetDelete`)
  and the new marketing test.
- `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes`,
  `TestMergeMenuItem_MissingTargetIsRefused` and the four erasure tests exactly as they are.

## What is safe to drop

- Nothing in the code change. Under
  `.night-crew/runs/2026-10-07-autonomous/logs/dish-merge-shapes-and-backstop-tests/` the logs
  are evidence, not behaviour — a merge may keep either side's copy.

## Stubs and fixtures, per done-when clause

| Clause | Rides a stub or fixture? |
|---|---|
| Two shape tests RED on the pre-change tree, green after | No stub. Real handler, real repository, real Postgres (:5434); rows seeded by plain SQL fixtures. |
| Lock test RED with ` FOR SHARE` removed, green on the shipped code | No stub. Three real connections. The merge is held in flight by a real row lock a helper connection takes on the SOURCE dish (sequencing only — the delete under test and the lock under test are the real ones). |
| Backstop test green on the shipped 0086, RED with `SET NULL` → `NO ACTION` | No stub. The package's `TestMain` runs the real migrations; the test issues the real `DELETE`. Fixtures: one user, one campaign, one code, one dish. |
| Existing merge tests + four erasure tests untouched and green | No stub. |
| Full Go `-p 1` counts | No stub. |
| `inventory|recipes` Playwright seam confined | No stub added by this card. |
| Full Playwright suite, 10×/5× measurement leg, `sw.js` 51, G6 | The orchestrator's — not run by this card. |

No stub of the behaviour under test. No migration. No `night-crew.toml` key. `sw.js` /
`version.json` are not in the diff.

## Notes for the merge

- Edited outside the card's listed footprint: `backend/go.mod` only (above). `handler_test.go`
  is in the footprint and was NOT edited — the new handler-level tests sit beside
  `TestMergeMenuItem_MissingTargetIsRefused` in `repository_test.go`, the shape the card said to
  extend.
- Red legs that needed the tree changed (`FOR SHARE` removed; 0086's two `SET NULL` → `NO
  ACTION`) were run by editing this worktree's own copy, running, and restoring; each log ends
  with the exact diff that was applied. Neither mutation is in the diff. The pre-change red for
  the two shape tests used a throwaway two-line test file to declare the not-yet-existing
  sentinel names so the red is a behaviour failure, not a build failure (stated in that log).
- Behaviour seen while pinning the lock, NOT changed by this card: once the merge has
  committed, the queued `DELETE` of the surviving dish succeeds and takes the re-pointed recipe
  with it (`recipes.menu_item_id` cascades). The lock guarantees ordering — never a merge into a
  vanished dish — not that the survivor is undeletable afterwards. The test accepts either a
  23503 refusal or that outcome, as the card specifies.
- Confined seam (`logs/dish-merge-shapes-and-backstop-tests/confined-seam.log`): 252 passed /
  15 failed / 1 skipped. All 15 are `tests/inventory.spec.js` screen tests and every one is
  also red in the earlier card's seam on this same base tree
  (`logs/inventory-setup-races/confined-seam.log`; list in `confined-seam.reds.txt`). I did not
  re-run them on a clean base myself. `tests/recipes.spec.js` — the only spec that calls the
  merge endpoint — is fully green.
- Full Go suite (`full-go-suite.log`, under `/tmp/hq-full-suite.lock`): exit 0, 15 packages ok,
  0 failed; 460 top-level tests passed (725 counting subtests), 0 failed, 3 skipped
  (`TestProjectionConfiguredUpsertsToSubstrate`, `TestProxyLive_RealtimeUpgrade`,
  `TestProxyLive_RESTRequest` — not this card's).
