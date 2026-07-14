# DECISIONS-NEEDED — overnight-20260715 (for the morning of 2026-07-15)

> **Run branch:** `overnight-20260715` (cut from `dev`; never pushed, `main` untouched).
> **Slate:** `.night-crew/knowledge/reference/slate-20260715.md` (batch-signed 2026-07-14).

## ⚑ No open operator forks block triage.

The prove-UNPROVEN sweep is a **test-only guaranteed deliverable** and it landed 16/16 clean.
Every item below is either (a) a flow PARKED for a reason the slate pre-authorized, or (b) a
future fix-WO whose *scheduling* is already delegated to the planning agents (ledger T-10 model).
**None requires an operator decision to proceed.** They are surfaced here for the record and for
the planners to schedule into a future slate.

---

## A. Flows PARKED during the sweep (proof needs plumbing/refactor beyond a test fixture)

Each was PARKED per an explicit slate PARK trigger — the worktree change (a `test.skip` with an
inline reason, or a "not authored" note) is committed on the run branch so the gap is visible in
the suite, not silently dropped.

| # | Flow | Why parked | Committed as |
|---|---|---|---|
| P-1 | **Onboarding FR-18** custom-thumbnail upload | Needs `/api/v1/photos/presign` + live DO-Spaces S3 PUT + file input; thumbnail_url DB round-trip is a different flow (FR-20). | `test.skip` w/ reason (onboarding.spec.js) |
| P-2 | **Inventory FR-27** item photo → JPEG convert/resize | `UploadHandler` returns 503 "photo storage not configured" when the S3 client is nil (ephemeral stack sets no `SPACES_*`). Convert/resize contract unprovable without live S3. | `test.skip` w/ reason (inventory.spec.js) |
| P-3 | **Operations NFR-2** photo presign→PUT→round-trip (PUT leg) | Same S3 plumbing. **Presign endpoint SHAPE proven GREEN** (auth-gated 503 degraded contract); only the PUT-to-S3 + public-URL round-trip is parked. | note in workflows.spec.js block |
| P-4 | **Operations NFR-5** offline sync queue/conflict/cleanup | Needs IndexedDB + service-worker harness; `serviceWorkers:'block'` in `playwright.config.js:29`. No test authored (honest — not forced into a false classification). | inline PARK note |
| P-5 | **Operations NFR-7** draft-persist-across-redirect (draft leg) | IndexedDB (`hq_offline_v1`) plumbing. **Redirect leg proven GREEN** (401→/login.html); only the local-draft-survives-redirect leg is parked. | note in workflows.spec.js block |
| P-6 | **Purchasing FR-19/20/21/22** cron DECISION logic (4 crons) | `runReminderCheck`/`runCutoffCheck`/`runLowStockCheck` (scheduler.go:54/167/247) + `runRepurchaseResetCheck` (repurchase.go:129) each read `now := time.Now().In(loc)` inline; the funcs take only `(ctx, *pool)` with no injectable clock. Unit-testing "at cutoff+1 auto-lock fires" requires a production seam — a PARK, not a test-only fix. Adjacent pure logic (parseCutoffTime, isAdmin, the cutoff-decision rule fed a frozen clock) proven GREEN in `scheduler_prove_test.go`. | `scheduler_prove_test.go` + PARK note |

## B. Future fix-WOs surfaced (schedule into a later slate — NOT done tonight)

These are the *fixes* the PARKs above imply. They are beyond the prove-then-fix "same-footprint
test-only" rule (each needs a production refactor or a new test harness), so per the slate they
were **not** graduated tonight. Hand to the planners (they own scheduling, per ledger T-10).

- **WO-cron-clock-seam** — add a `now time.Time` (or package `nowFn`) seam to the 4 `run*Check`
  functions in `internal/purchasing/scheduler.go` + `repurchase.go`, then add real cron-decision
  unit tests (seed config + past-cutoff → assert transition, no 15-minute wait). Unblocks P-6.
- **WO-photo-s3-harness** — a way to exercise presign→PUT→public-URL in E2E (a mock S3 / test
  DO-Spaces bucket). Unblocks P-1, P-2, P-3 (3 flows) in one harness.
- **WO-offline-indexeddb-harness** — a dedicated Playwright project with the service worker +
  IndexedDB enabled to test offline sync/queue/conflict/draft-persist. Unblocks P-4, P-5.

## C. Untestable (recorded, no WO — do not schedule)

- **Onboarding FR-28** seed idempotent re-seed — boot-time / in-process (`SeedOnboardingTemplates`);
  no E2E trigger, no in-test restart. Asserted **indirectly** via a presence anchor ("Kitchen Basics
  Training" exists after boot). No fix graduates from an untestable assertion (per slate).

## D. Graduated fix landed tonight (Eng KR-1 movement)

- **Inventory NFR-1 item-edit + confirm-vendor normalization** — the sweep's **one** graduatable
  RED. `UpdateItemHandler` wrote `input.Description` raw (no `normalizeItemName`, unlike
  `CreateItemHandler`); `ConfirmPendingPurchaseHandler` upserted the vendor name raw while
  line-items were normalized. Fixed in `internal/inventory/handler.go` (red→green: the E4-committed
  NFR-1 test flips RED→GREEN). See HANDOFF.md for the fix card's gate evidence.
  *(If this line still says "in progress" in your copy, check HANDOFF.md for the final verdict —
  it was the last card of the run.)*

---

**Bottom line for triage:** sign off the 16 prove cards + the 1 graduated fix (all G6-PASS,
red→green honest), flip the roadmap rows, and hand sections A/B to the planners as backlog. No
decision gates the merge to `dev`.
