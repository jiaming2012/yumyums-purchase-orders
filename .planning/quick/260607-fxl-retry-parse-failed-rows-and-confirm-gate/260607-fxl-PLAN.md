---
phase: quick-260607-fxl
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/receipt/worker.go
  - backend/internal/receipt/worker_test.go
  - backend/internal/inventory/handler.go
  - backend/internal/inventory/period_summary_test.go
  - inventory.html
  - sw.js
  - tests/inventory.spec.js
autonomous: true
requirements:
  - FIX-A-WORKER-RETRY
  - FIX-A-WORKER-TESTS
  - FIX-B-BACKEND-CONFIRM-GATE
  - FIX-B-FE-CONFIRM-DISABLED

must_haves:
  truths:
    - "Worker re-parses pending rows where Haiku previously failed once parse_error IS NULL AND items is empty AND attachments exist"
    - "Worker does NOT re-parse rows where parse_error is populated (BOTH models already failed)"
    - "Worker does NOT clobber rows where the user has added line items (items non-empty)"
    - "Backend rejects POST /confirm with empty line_items UNLESS pending row's reason='no_attachment_on_bank_tx', returning HTTP 422 envelope {error:'empty_items_not_allowed'}"
    - "Backend returns HTTP 422 with structured envelope {error:'total_mismatch',line_total,bank_total} on receipt-vs-bank total mismatch (was 400 text)"
    - "FE Confirm Receipt button is disabled when (items empty AND reason!='no_attachment_on_bank_tx') OR (items non-empty AND totals don't match)"
    - "Existing 260605-pk1 empty-items-no-attachment confirm flow still returns 200 (no regression on end-to-end empty-items confirm test)"
  artifacts:
    - path: "backend/internal/receipt/worker.go"
      provides: "Extended classifyExistingTx return shape (kind, reason, hasParseError, hasItems) + isUpgrade=true on parse-failed retry branch"
      contains: "hasParseError"
    - path: "backend/internal/receipt/worker_test.go"
      provides: "3 new tests: RetriesPriorHaikuFailureWhenParseErrorNull, DoesNotRetryWhenParseErrorSet, DoesNotRetryParseFailedWithItems"
      contains: "TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull"
    - path: "backend/internal/inventory/handler.go"
      provides: "ConfirmPendingPurchaseHandler fetches reason; gates empty-items branch behind reason check; total_mismatch upgraded to 422 envelope"
      contains: "empty_items_not_allowed"
    - path: "backend/internal/inventory/period_summary_test.go"
      provides: "3 new ConfirmPending tests added to existing file (it already hosts the end-to-end empty-items confirm test)"
      contains: "TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached"
    - path: "inventory.html"
      provides: "canConfirm computation + disabled attribute on Confirm Receipt button; CSS for .btn-primary:disabled; input-listener refresh of disabled state"
      contains: "canConfirm"
    - path: "tests/inventory.spec.js"
      provides: "3 new Playwright tests for the Confirm Receipt disabled state"
      contains: "Confirm Receipt button is disabled when totals do not match"
  key_links:
    - from: "worker.go runIngestCycle"
      to: "worker.go classifyExistingTx"
      via: "kind/reason/hasParseError/hasItems return values"
      pattern: "classifyExistingTx\\(ctx"
    - from: "handler.go ConfirmPendingPurchaseHandler"
      to: "pending_purchases.reason column"
      via: "extended SELECT at line ~671 to also fetch reason"
      pattern: "SELECT bank_tx_id, bank_total, .*reason FROM pending_purchases"
    - from: "inventory.html renderReviewForm (~line 649)"
      to: "REVIEW_FORM_STATE[id].line_items + p.reason"
      via: "canConfirm computation → disabled attribute on btn-primary"
      pattern: "canConfirm"
    - from: "inventory.html input listener (~line 951)"
      to: "Confirm Receipt button disabled state"
      via: "after captureReviewFormInputs, recompute canConfirm and set/remove disabled"
      pattern: "btn-primary.*disabled"
---

<objective>
Two bundled fixes for the pending-purchases pipeline that ship together because they share the same DB column (`pending_purchases.reason`) and the same review-form FE surface.

**FIX A — Worker retries Haiku-failed rows once:** Today `classifyExistingTx` only marks `isUpgrade=true` when `reason='no_attachment_on_bank_tx'`. Rows that hit `Receipt could not be parsed automatically` from a pre-260607-e1c poll (before Sonnet fallback shipped) are stuck — they have receipts but never got a Sonnet attempt. Extend the upgrade allowlist so they get retried once, gated by `parse_error IS NULL AND items is empty` to prevent infinite loops and to avoid clobbering user-edited rows.

**FIX B — Backend + FE block bad confirms:** Backend currently lets empty `line_items` through into the no-itemized-receipt placeholder branch regardless of why the row is pending. That's correct for `reason='no_attachment_on_bank_tx'` (260605-pk1 flow) but wrong for parse-failed rows where the receipt PDF IS attached — the operator should be forced to itemize or discard. Add a backend 422 gate. Also upgrade the existing total-mismatch text-400 to a structured 422 envelope. Mirror the rule on the FE with a `disabled` attribute on Confirm Receipt so the operator can see the gate before clicking.

Purpose: Get the stuck Restaurant Depot $391.96 row auto-resolved on next sync, and prevent a class of accidental wrong-COGS confirms.
Output: 4 atomic commits, no migrations, no new deps.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@./CLAUDE.md

<!-- Source files the executor will modify -->
@backend/internal/receipt/worker.go
@backend/internal/receipt/worker_test.go
@backend/internal/inventory/handler.go
@backend/internal/inventory/period_summary_test.go
@inventory.html
@tests/inventory.spec.js

<interfaces>
<!-- Extracted from worker.go — current shape of the seam being extended -->

From backend/internal/receipt/worker.go (current):
```go
// Current classifyExistingTx signature — we're extending the return shape.
func classifyExistingTx(ctx context.Context, pool *pgxpool.Pool, bankTxID string) (kind, reason string, err error)
// kind ∈ {"none", "event", "pending"}
// reason carries pending_purchases.reason for the "pending" kind, "" otherwise.

// Current upgrade gate at ~line 179:
// if existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0 {
//     isUpgrade = true
// }

// routePending dispatches UPDATE (upgrade) vs INSERT (cold path) — unchanged by this plan.
func routePending(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem,
    summary ReceiptSummary, receiptURL string, reason string, parseError string, isUpgrade bool) error
```

From backend/internal/receipt/worker_test.go (helpers we'll reuse, NOT redefine):
```go
type workerStubs struct {
    txns            []MercuryTransaction
    parseItems      []ReceiptItem
    parseSummary    ReceiptSummary
    parseErr        error
    parseCallCount  int
    sonnetItems     []ReceiptItem
    sonnetSummary   ReceiptSummary
    sonnetErr       error
    sonnetCallCount int
    fetchCalled     bool
    dlCalled        bool
}
func installWorkerStubs(t *testing.T, s *workerStubs)
func resetReceiptFixtures(t *testing.T)
```
`workerStubs.parseCallCount` and `sonnetCallCount` are the existing call counters T2's "must not be called" assertions use.

From backend/internal/inventory/handler.go (current SELECT to extend):
```go
// Line 671-674 in ConfirmPendingPurchaseHandler:
err = tx.QueryRow(r.Context(),
    `SELECT bank_tx_id, bank_total FROM pending_purchases
       WHERE id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
    input.ID,
).Scan(&bankTxID, &bankTotal)

// Line 30-32 — writeJSON and writeError helpers we'll use for the 422 envelopes:
func writeJSON(w http.ResponseWriter, status int, v any)
func writeError(w http.ResponseWriter, status int, msg string)  // shape: {"error": msg}
```

From backend/internal/inventory/period_summary_test.go (test seeders to reuse):
```go
// insertPendingPurchaseWithBankTotal inserts a no-attachment-style row
// (reason defaults to NULL because the column isn't passed; items='[]').
// Used by the end-to-end empty-items confirm test at line 521.
func insertPendingPurchaseWithBankTotal(t *testing.T, bankTxID string, bankTotal float64, createdAt string) string

// insertPendingPurchaseWithEventDate defaults reason to 'no_attachment_on_bank_tx'
// when caller passes "". Use this for the positive-case test.
func insertPendingPurchaseWithEventDate(t *testing.T, bankTxID, eventDate, createdAt, reason string) string

func insertTestUser(t *testing.T, email string) string  // returns user UUID for auth context
func insertNoItemizedReceiptSeed(t *testing.T)          // seeds the placeholder purchase_items row
func resetFixtures(t *testing.T)
```

For T3's NEW tests, the existing empty-items test at line 521 (`t.Run("end-to-end empty-items confirm increments cogs"...`) uses `insertPendingPurchaseWithBankTotal` which inserts WITHOUT a reason → the column ends up NULL in DB. The handler currently treats NULL reason the same as "any reason" for the empty-items branch. **After FIX B, NULL reason is NOT the allowlist value 'no_attachment_on_bank_tx' → the existing test would break.** The test must therefore be migrated to either (a) set reason='no_attachment_on_bank_tx' on the seeded row via a direct UPDATE, OR (b) switch the seeder call to `insertPendingPurchaseWithEventDate` which defaults reason='no_attachment_on_bank_tx'. Pick (a) — smallest diff, preserves the existing test's bank_total negative-debit semantics that `insertPendingPurchaseWithBankTotal` provides but `insertPendingPurchaseWithEventDate` does not.

From inventory.html (current Confirm Receipt button render at ~line 649):
```javascript
'<button class="btn-primary" data-action="confirm-receipt" data-id="'+p.id+'">Confirm Receipt</button>'
```

From inventory.html (current input listener at ~line 951-995) — it already recomputes `lineTotal`/`grandTotal`/`mismatch` and toggles `.correction-banner`/`.match-banner`. We extend it to also toggle the Confirm Receipt button's `disabled` attribute under the SAME canConfirm rule used at render time.

From tests/inventory.spec.js (existing page.route stub pattern at ~line 2912-2935):
```javascript
await page.route('**/api/v1/inventory/purchases/pending', async route => {
  if (route.request().method() !== 'GET') return route.continue();
  await route.fulfill({
    status: 200, contentType: 'application/json',
    body: JSON.stringify([{
      id: 'pe-1', bank_tx_id: 'tx-parse-err', bank_total: -391.96,
      vendor: 'RESTAURANT DEPOT', event_date: '2026-06-05',
      reason: 'Receipt could not be parsed automatically',
      parse_error: "...",
      items: [], created_at: new Date().toISOString(),
    }])
  });
});
```
Use this exact pattern for T4's tests — change `id`, `reason`, `items`, `bank_total` per scenario.
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1 (FIX A): Extend classifyExistingTx + upgrade allowlist in runIngestCycle</name>
  <files>backend/internal/receipt/worker.go</files>
  <action>
**Goal:** Add ONE new isUpgrade branch in runIngestCycle that retries pre-260607-e1c parse-failed rows, gated by parse_error IS NULL AND items empty.

**Step 1 — Extend `classifyExistingTx` (worker.go ~line 358):**
- Change signature to: `func classifyExistingTx(ctx context.Context, pool *pgxpool.Pool, bankTxID string) (kind, reason string, hasParseError, hasItems bool, err error)`
- `hasParseError` = true when `pending_purchases.parse_error IS NOT NULL` for the matched pending row.
- `hasItems` = true when `pending_purchases.items` is non-NULL AND not an empty JSONB array. Use `COALESCE(jsonb_array_length(items), 0) > 0` in the query.
- For `kind="event"` and `kind="none"` branches, return `false, false` for both new booleans (they're meaningless when there's no open pending row).
- Update the SELECT:
  ```sql
  SELECT 'event' AS kind, '' AS reason, false AS has_parse_error, false AS has_items
    FROM purchase_events WHERE bank_tx_id = $1
  UNION ALL
  SELECT 'event' AS kind, COALESCE(reason,''), false, false
    FROM pending_purchases
   WHERE bank_tx_id = $1 AND confirmed_at IS NOT NULL
  UNION ALL
  SELECT 'pending' AS kind, COALESCE(reason,''),
         (parse_error IS NOT NULL),
         (COALESCE(jsonb_array_length(items), 0) > 0)
    FROM pending_purchases
   WHERE bank_tx_id = $1
     AND confirmed_at IS NULL
     AND discarded_at IS NULL
  LIMIT 1
  ```
- `pgx.ErrNoRows` → return `"none", "", false, false, nil`.
- Update the doc-comment block above the function to describe the new return values and their meaning. Note that `hasParseError`/`hasItems` are only meaningful for `kind="pending"`.

**Step 2 — Update the single caller in runIngestCycle (~line 163):**
- Change call site:
  ```go
  kind, existingReason, hasParseError, hasItems, err := classifyExistingTx(ctx, cfg.Pool, tx.ID)
  ```
- Inside the `case "pending":` block (~line 174-185), extend the upgrade condition. Current code:
  ```go
  if existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0 {
      isUpgrade = true
  } else {
      skippedCached++
      continue
  }
  ```
  New code:
  ```go
  // Existing upgrade case: no-attachment row whose Mercury tx now has a receipt.
  noAttachmentUpgrade := existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0
  // NEW (260607-fxl): retry rows where Haiku failed before Sonnet fallback
  // existed. Gate by parse_error IS NULL (haven't retried yet — and not stuck
  // in a both-models-failed loop) AND items empty (user hasn't started
  // editing the row). Caller must have attachments to retry against.
  parseFailedRetry := existingReason == "Receipt could not be parsed automatically" &&
      !hasParseError && !hasItems && len(tx.Attachments) > 0
  if noAttachmentUpgrade || parseFailedRetry {
      isUpgrade = true
      // Fall through to download/parse path below.
  } else {
      skippedCached++
      continue
  }
  ```
- The fall-through path is unchanged: download → parse (haiku then sonnet) → validate → either `createPurchaseEvent(isUpgrade=true)` (which DELETE+INSERTs the pending row atomically) or `routePending(isUpgrade=true)` (which UPDATEs the existing row, populating parse_error so next sync skips). No new code in that section.

**Step 3 — Verify `routePending` already covers the parse-failed retry path:**
- For parse-failed retry where BOTH Haiku and Sonnet fail again, control flows through `routePending(ctx, ..., "Receipt could not be parsed automatically", combined, isUpgrade=true)` at ~line 277. With `isUpgrade=true`, `routePending` calls `updatePendingPurchase` which writes the new parse_error column. Next sync → `hasParseError=true` → `parseFailedRetry=false` → row skipped. This is correct as-is; no code change needed.

**Do NOT:**
- Add a manual "Retry Parse" FE button (out of scope).
- Backfill existing rows (out of scope).
- Add new migrations (column shipped in 0069).
- Touch the `case "event":` or `case "none":` paths.
- Use `gofmt -w` or `go fmt` aggressively — preserve existing formatting style.

**After editing:** Run `cd backend && go build ./...` to confirm signature change compiles across all callers (there's only one caller, in runIngestCycle, but the build will catch typos in `worker_test.go` too — note T2's tests come next so test file may not yet compile against the new signature; T1 verify step uses `go vet ./internal/receipt/...` instead).
  </action>
  <verify>
    <automated>cd backend && go vet ./internal/receipt/... 2>&amp;1 | grep -v "^# " | (! grep -E "error|wrong number|cannot use")</automated>
  </verify>
  <done>
- `classifyExistingTx` returns 5 values including `hasParseError bool` and `hasItems bool`.
- `runIngestCycle` `case "pending":` block has both `noAttachmentUpgrade` and `parseFailedRetry` conditions, OR'd into `isUpgrade`.
- `go vet ./internal/receipt/...` passes (existing tests may fail to compile against new signature — that's fixed in T2).
- No migration files added.
- Only one file modified: `backend/internal/receipt/worker.go`.
- Commit message: `feat(260607-fxl): worker retries parse-failed rows when parse_error null AND items empty`
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 2 (FIX A tests): 3 new ingest-cycle tests + fix any T1 signature breakage</name>
  <files>backend/internal/receipt/worker_test.go</files>
  <behavior>
- Test 1 (`TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull`): Seeded pending row with `reason='Receipt could not be parsed automatically'`, `parse_error=NULL`, `items='[]'::jsonb`. Mercury returns the same tx with an attachment. `parseErr` set (Haiku fails), `sonnetItems`/`sonnetSummary` populated to pass `ValidateReceiptData`. Expect: `result.AutoCreated == 1`, `result.PendingReview == 0`, `result.Cached == 0`. Pending row gone, `purchase_events` row exists for that `bank_tx_id`. `stubs.parseCallCount == 1`, `stubs.sonnetCallCount == 1`.
- Test 2 (`TestRunIngestCycle_DoesNotRetryWhenParseErrorSet`): Seeded pending row with `reason='Receipt could not be parsed automatically'`, `parse_error='haiku boom; sonnet boom'`, `items='[]'::jsonb`. Mercury returns the same tx with an attachment. Expect: `result.Cached == 1`, `result.AutoCreated == 0`, `result.PendingReview == 0`. `stubs.parseCallCount == 0` AND `stubs.sonnetCallCount == 0` (cached short-circuit must NOT call parser). `stubs.dlCalled == false` (no download either). Pending row untouched.
- Test 3 (`TestRunIngestCycle_DoesNotRetryParseFailedWithItems`): Seeded pending row with `reason='Receipt could not be parsed automatically'`, `parse_error=NULL`, `items='[{"name":"foo","quantity":1,"price":1.0,"is_case":false}]'::jsonb` (user added a line item already). Expect: `result.Cached == 1`. `parseCallCount == 0`. Pending row's `items` field unchanged (asserted by re-reading the row and comparing items::text).
  </behavior>
  <action>
**Step 1 — Fix existing tests broken by T1's signature change:**
- Search `worker_test.go` for any direct calls to `classifyExistingTx` (likely none — it's only called from runIngestCycle in prod code). If any exist, update to the 5-return-value form.
- The existing tests at lines 455, 547, 622, 681, 1046, 1118 call `runIngestCycle` not `classifyExistingTx` — they should compile unchanged. Run `cd backend && go test -count=1 ./internal/receipt/...` to confirm before adding new tests.

**Step 2 — Add the 3 new tests near the other `TestRunIngestCycle_*` tests** (after `TestRunIngestCycle_SkipsRealCached` at ~line 622 is a good neighborhood — they're all about the cached/upgrade discrimination path).

**Pattern to follow** (copy/adapt from `TestRunIngestCycle_UpgradesPendingNoAttachmentRow` at line 459):
```go
func TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull(t *testing.T) {
    if testPool == nil { t.Skip("DB_TEST_URL not reachable; skipping integration test") }
    resetReceiptFixtures(t)

    if _, err := testPool.Exec(t.Context(), `
        INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
        VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb, NULL)`,
        "T-retry-ok", -42.50, "STUB",
    ); err != nil { t.Fatalf("seed: %v", err) }

    stubs := &workerStubs{
        txns: []MercuryTransaction{{
            ID:          "T-retry-ok",
            Amount:      -42.50,
            CreatedAt:   "2026-05-27T10:00:00Z",
            Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
        }},
        parseErr: errors.New("haiku boom"),
        sonnetItems: []ReceiptItem{
            {Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
        },
        sonnetSummary: ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1, TotalCases: 0},
    }
    installWorkerStubs(t, stubs)

    result, err := runIngestCycle(t.Context(), WorkerConfig{
        MercuryAPIKey:   "stub",
        AnthropicAPIKey: "stub",
        Pool:            testPool,
        LookbackDays:    14,
    })
    if err != nil { t.Fatalf("runIngestCycle: %v", err) }

    if result.AutoCreated != 1 { t.Errorf("AutoCreated = %d, want 1", result.AutoCreated) }
    if result.Cached != 0 { t.Errorf("Cached = %d, want 0", result.Cached) }
    if stubs.parseCallCount != 1 { t.Errorf("parseCallCount = %d, want 1", stubs.parseCallCount) }
    if stubs.sonnetCallCount != 1 { t.Errorf("sonnetCallCount = %d, want 1", stubs.sonnetCallCount) }

    var pendingCount, eventCount int
    testPool.QueryRow(t.Context(),
        `SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id='T-retry-ok'`).Scan(&pendingCount)
    testPool.QueryRow(t.Context(),
        `SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='T-retry-ok'`).Scan(&eventCount)
    if pendingCount != 0 { t.Errorf("pending rows = %d, want 0 (retry success should DELETE)", pendingCount) }
    if eventCount != 1 { t.Errorf("event rows = %d, want 1", eventCount) }
}

func TestRunIngestCycle_DoesNotRetryWhenParseErrorSet(t *testing.T) {
    if testPool == nil { t.Skip("DB_TEST_URL not reachable; skipping integration test") }
    resetReceiptFixtures(t)

    if _, err := testPool.Exec(t.Context(), `
        INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
        VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb, 'haiku boom; sonnet boom')`,
        "T-no-retry-err", -42.50, "STUB",
    ); err != nil { t.Fatalf("seed: %v", err) }

    stubs := &workerStubs{
        txns: []MercuryTransaction{{
            ID:          "T-no-retry-err",
            Amount:      -42.50,
            CreatedAt:   "2026-05-27T10:00:00Z",
            Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
        }},
    }
    installWorkerStubs(t, stubs)

    result, err := runIngestCycle(t.Context(), WorkerConfig{
        MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14,
    })
    if err != nil { t.Fatalf("runIngestCycle: %v", err) }

    if result.Cached != 1 { t.Errorf("Cached = %d, want 1", result.Cached) }
    if result.AutoCreated != 0 { t.Errorf("AutoCreated = %d, want 0", result.AutoCreated) }
    if stubs.parseCallCount != 0 { t.Errorf("parseCallCount = %d, want 0 (cached row must not call Haiku)", stubs.parseCallCount) }
    if stubs.sonnetCallCount != 0 { t.Errorf("sonnetCallCount = %d, want 0 (cached row must not call Sonnet)", stubs.sonnetCallCount) }
    if stubs.dlCalled { t.Errorf("dlCalled = true, want false (cached row must not download)") }
}

func TestRunIngestCycle_DoesNotRetryParseFailedWithItems(t *testing.T) {
    if testPool == nil { t.Skip("DB_TEST_URL not reachable; skipping integration test") }
    resetReceiptFixtures(t)

    if _, err := testPool.Exec(t.Context(), `
        INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
        VALUES ($1, $2, $3, 'Receipt could not be parsed automatically',
                '[{"name":"foo","quantity":1,"price":1.0,"is_case":false}]'::jsonb, NULL)`,
        "T-no-retry-items", -42.50, "STUB",
    ); err != nil { t.Fatalf("seed: %v", err) }

    var origItems string
    testPool.QueryRow(t.Context(),
        `SELECT items::text FROM pending_purchases WHERE bank_tx_id='T-no-retry-items'`).Scan(&origItems)

    stubs := &workerStubs{
        txns: []MercuryTransaction{{
            ID:          "T-no-retry-items",
            Amount:      -42.50,
            CreatedAt:   "2026-05-27T10:00:00Z",
            Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
        }},
    }
    installWorkerStubs(t, stubs)

    result, err := runIngestCycle(t.Context(), WorkerConfig{
        MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14,
    })
    if err != nil { t.Fatalf("runIngestCycle: %v", err) }

    if result.Cached != 1 { t.Errorf("Cached = %d, want 1", result.Cached) }
    if stubs.parseCallCount != 0 { t.Errorf("parseCallCount = %d, want 0 (user-edited row must not be reparsed)", stubs.parseCallCount) }

    var gotItems string
    testPool.QueryRow(t.Context(),
        `SELECT items::text FROM pending_purchases WHERE bank_tx_id='T-no-retry-items'`).Scan(&gotItems)
    if gotItems != origItems {
        t.Errorf("items changed: was %q, now %q (worker must not clobber user edits)", origItems, gotItems)
    }
}
```

**Step 3 — Run the new tests:**
```bash
cd backend && go test -count=1 -run 'TestRunIngestCycle_RetriesPriorHaikuFailure|TestRunIngestCycle_DoesNotRetryWhenParseErrorSet|TestRunIngestCycle_DoesNotRetryParseFailedWithItems' ./internal/receipt/...
```

Then run the full receipt package suite to confirm no regressions:
```bash
cd backend && go test -count=1 ./internal/receipt/...
```

**Do NOT:**
- Add to or modify `workerStubs` struct (call counters and seam wiring already exist from 260607-co0/e1c).
- Define a new `resetReceiptFixtures` or test pool — reuse the package-level `testPool`.
- Add tests for the `case "event":` (already covered by `TestRunIngestCycle_SkipsExistingPurchaseEvent`) or no-attachment branch (already covered).
- Skip the "stubs must not be called" assertions — those are the regression guards for the parse_error/items gates.
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go test -count=1 ./internal/receipt/...</automated>
  </verify>
  <done>
- 3 new tests added to `backend/internal/receipt/worker_test.go`.
- Full receipt package test suite passes (`go test ./internal/receipt/...`).
- All 3 new tests assert on `parseCallCount` / `sonnetCallCount` to guard against the cached-but-still-called regression.
- Test 3 asserts items column is byte-for-byte unchanged after the worker pass.
- Commit message: `test(260607-fxl): worker retries parse-failed rows only when parse_error null AND items empty`
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 3 (FIX B backend): Gate empty-items confirm + 422 envelopes + tests</name>
  <files>backend/internal/inventory/handler.go, backend/internal/inventory/period_summary_test.go</files>
  <behavior>
**handler.go behavior changes:**
- ConfirmPendingPurchaseHandler fetches `reason` alongside `bank_tx_id`/`bank_total`.
- When `len(input.LineItems) == 0` AND `reason != "no_attachment_on_bank_tx"` → respond `422` with body `{"error":"empty_items_not_allowed","reason":"add at least one line item or set pending reason to no_attachment_on_bank_tx"}`. The "reason" KEY in the response envelope is the human hint string, NOT the DB column — naming is awkward but matches the task spec verbatim.
- When `len(input.LineItems) > 0` AND totals don't match (current logic at line 700-703) → respond `422` (was `400`) with body `{"error":"total_mismatch","line_total":X.XX,"bank_total":Y.YY}` (numeric values, not pre-formatted strings).
- All other branches (200/201 on success, 404 on not found, etc.) unchanged.

**period_summary_test.go tests:**
- `TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached`: seed pending row with `reason='Receipt could not be parsed automatically'`, POST confirm with empty `LineItems`. Expect 422, body's `error` field equals `"empty_items_not_allowed"`.
- `TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment`: seed pending row with `reason='no_attachment_on_bank_tx'` and `bank_total=-75.00`, POST confirm with empty `LineItems`. Expect 200, purchase_events row created. (This is the existing 260605-pk1 flow — proves no regression.)
- `TestConfirmPending_RejectsTotalMismatchWith422`: seed pending row with `reason='Receipt could not be parsed automatically'` and `bank_total=-50.00`. POST confirm with line_items summing to $42.00 (mismatch). Expect 422, body's `error == "total_mismatch"`, `line_total == 42.00`, `bank_total == 50.00`.

**Existing regression to preserve:**
- The existing test at period_summary_test.go:521 (`end-to-end empty-items confirm increments cogs`) MUST still pass after this change. Currently the seeded row uses `insertPendingPurchaseWithBankTotal` which inserts WITHOUT a reason → DB stores NULL. After FIX B, NULL reason fails the allowlist check → test breaks. Fix it by injecting `reason='no_attachment_on_bank_tx'` on the seeded row.
  </behavior>
  <action>
**Step 1 — Modify `ConfirmPendingPurchaseHandler` in `backend/internal/inventory/handler.go`:**

(a) Extend the SELECT at line ~671:
```go
var bankTxID string
var bankTotal float64
var pendingReason sql.NullString
err = tx.QueryRow(r.Context(),
    `SELECT bank_tx_id, bank_total, reason
       FROM pending_purchases
      WHERE id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
    input.ID,
).Scan(&bankTxID, &bankTotal, &pendingReason)
if err != nil {
    writeError(w, http.StatusNotFound, "pending_purchase_not_found")
    return
}
reasonStr := ""
if pendingReason.Valid { reasonStr = pendingReason.String }
```
Add `"database/sql"` to the import block if not already imported (it is — used elsewhere in the package; verify via grep first).

(b) Insert the empty-items gate BEFORE the `emptyResolution := len(input.LineItems) == 0` line at ~line 689:
```go
// 260607-fxl: empty items only allowed for explicit no-attachment rows.
// Parse-failed rows (where a receipt IS attached) must be itemized or
// discarded — the operator should not be able to write the placeholder
// $bank_total into cogs and bypass the line-item review.
if len(input.LineItems) == 0 && reasonStr != "no_attachment_on_bank_tx" {
    writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
        "error":  "empty_items_not_allowed",
        "reason": "add at least one line item or set pending reason to no_attachment_on_bank_tx",
    })
    return
}
```

(c) Replace the existing total-mismatch rejection at line ~700-703:
```go
// BEFORE:
if absBankTotal-lineTotal > 0.01 || lineTotal-absBankTotal > 0.01 {
    writeError(w, http.StatusBadRequest, fmt.Sprintf("total_mismatch: receipt total $%.2f does not match bank transaction $%.2f", lineTotal, absBankTotal))
    return
}
// AFTER:
if absBankTotal-lineTotal > 0.01 || lineTotal-absBankTotal > 0.01 {
    writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
        "error":      "total_mismatch",
        "line_total": math.Round(lineTotal*100) / 100,
        "bank_total": math.Round(absBankTotal*100) / 100,
    })
    return
}
```
Add `"math"` to the imports if not present. The Round-to-2dp avoids floating-point fuzz like `42.000000001` in the JSON.

**Step 2 — Migrate existing test at period_summary_test.go:521 to set reason:**
Currently seeded with `insertPendingPurchaseWithBankTotal(t, "e2e-empty-tx-1", -75.00, "2026-05-27 10:00:00-05:00")`. Right after that line, inject:
```go
if _, err := testPool.Exec(t.Context(),
    `UPDATE pending_purchases SET reason = 'no_attachment_on_bank_tx' WHERE id = $1`,
    ppID); err != nil {
    t.Fatalf("set reason: %v", err)
}
```
Do NOT modify `insertPendingPurchaseWithBankTotal` itself — other callers may rely on the NULL-reason default.

**Step 3 — Add 3 new tests near the existing confirm test in period_summary_test.go (e.g. after the `t.Run("end-to-end empty-items confirm increments cogs", ...)` block closes):**

Use the same `httptest.NewRequest` / `ConfirmPendingPurchaseHandler(testPool).ServeHTTP(rec, req)` pattern from the existing test at line 521. Wrap each in a `t.Run` subtest under a parent `TestConfirmPending_GatesEmptyItems` function so they share a fixture-reset prelude, OR add as top-level `Test*` functions — pick whichever yields a smaller diff. Each test needs:
- `resetFixtures(t)` to clear pending_purchases.
- `insertNoItemizedReceiptSeed(t)` (only needed for the positive-case AcceptsEmptyItemsWhenNoAttachment test, since it lands in the placeholder-line-item branch).
- `insertTestUser(t, "...")` + auth context wiring (copy from existing test at lines 546-550).
- The seed: insert pending_purchases with the exact reason for the scenario. For Test 1 and Test 3, use a direct `testPool.Exec` INSERT with `reason='Receipt could not be parsed automatically'` (no helper exists with that reason). For Test 2, call `insertPendingPurchaseWithBankTotal` then UPDATE reason to `'no_attachment_on_bank_tx'` (mirror the existing-test fix from Step 2).
- The POST body: `ConfirmPendingInput{ID: ppID, VendorName: "X", EventDate: "2026-05-27", LineItems: ...}`.
- Assert `rec.Code == 422` (Tests 1,3) or `rec.Code == 200` (Test 2).
- Parse `rec.Body` with `json.NewDecoder(...).Decode(&body)` into a `map[string]any` (since values are mixed types) for Test 3, or `map[string]string` for Test 1.
- Test 1 assertion: `body["error"] == "empty_items_not_allowed"`.
- Test 3 assertions: `body["error"] == "total_mismatch"`, `body["line_total"].(float64) == 42.00`, `body["bank_total"].(float64) == 50.00`. JSON numbers decode as `float64` into `any`.

For Test 3's line_items, use one line: `{PurchaseItemID: "<seed-or-fresh>", Description: "x", Quantity: 1, Price: 42.00, IsCase: false}`. You'll need a `purchase_items` row to satisfy the FK in `purchase_line_items` — but since the mismatch check rejects BEFORE the insert, no line_item INSERT actually runs, so the FK is never exercised. Use a `PurchaseItemID` value of `"00000000-0000-0000-0000-000000000001"` (the placeholder seed) and seed it via `insertNoItemizedReceiptSeed(t)` to be safe.

**Step 4 — Run the new tests + verify existing test still passes:**
```bash
cd backend && go test -count=1 -run 'TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached|TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment|TestConfirmPending_RejectsTotalMismatchWith422|TestPeriodSummary' ./internal/inventory/...
```

Then full inventory package suite:
```bash
cd backend && go test -count=1 ./internal/inventory/...
```

**Do NOT:**
- Change the response shape for the SUCCESS path of ConfirmPendingPurchaseHandler.
- Touch `ListPendingPurchasesHandler` or the pending list query (only confirm matters here).
- Add new helper functions to period_summary_test.go unless one of the 3 tests genuinely needs >1 use.
- Use `writeError` for the 422 responses — it produces `{"error":msg}` but Tests 1 and 3 need MORE fields. Use `writeJSON` directly with a map literal.
- Forget to migrate the existing line-521 test — leaving it broken is a hard regression.
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go test -count=1 ./internal/inventory/...</automated>
  </verify>
  <done>
- `handler.go` ConfirmPendingPurchaseHandler fetches `reason`; 422 envelope for `empty_items_not_allowed`; 422 envelope for `total_mismatch` with numeric line_total/bank_total.
- 3 new tests added to `period_summary_test.go`.
- Existing `end-to-end empty-items confirm increments cogs` test still passes (with the new `UPDATE pending_purchases SET reason = 'no_attachment_on_bank_tx'` injection).
- Full inventory package test suite passes.
- Commit message: `feat(260607-fxl): handler gates empty-items confirm + structured 422 envelopes for empty_items_not_allowed and total_mismatch`
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 4 (FIX B FE): canConfirm gate + disabled button + 3 Playwright tests + SW rebuild</name>
  <files>inventory.html, sw.js, tests/inventory.spec.js</files>
  <behavior>
- The Confirm Receipt button renders with the `disabled` HTML attribute when `canConfirm` is false at first paint.
- `canConfirm` = `(hasItems AND matches) OR (!hasItems AND isNoAttachment)`, where:
  - `hasItems` = `state.line_items.length > 0`
  - `isNoAttachment` = `p.reason === 'no_attachment_on_bank_tx'`
  - `matches` = `Math.abs((lineTotal + tax) - bankTotal) <= 0.01`
- After every input event in the review form, the disabled state is re-evaluated and toggled in-place — the existing input handler at ~line 951 already recomputes totals; we extend it to flip `disabled` on the Confirm Receipt button under the same canConfirm rule.
- CSS `.btn-primary:disabled` provides visible disabled styling (opacity 0.5, cursor not-allowed).
- The mismatch banner at ~line 624 is unchanged.

Playwright tests assert button.toBeDisabled() / toBeEnabled() across three scenarios.
  </behavior>
  <action>
**Step 1 — Update `renderReviewForm` (or equivalent function around line 636) in inventory.html:**

Right after the existing `mismatchBanner` computation (~line 624), add the canConfirm computation:
```javascript
var hasItems=state.line_items.length>0;
var isNoAttachment=p.reason==='no_attachment_on_bank_tx';
var matches=!mismatch;  // mismatch already computed above as Math.abs(grandTotal-bankTotal)>0.01
var canConfirm=(hasItems&&matches)||(!hasItems&&isNoAttachment);
```

Then at line 649, change the Confirm Receipt button render to inject the disabled attribute:
```javascript
'<button class="btn-primary" data-action="confirm-receipt" data-id="'+p.id+'"'+(canConfirm?'':' disabled')+'>Confirm Receipt</button>'+
```

**Step 2 — Update the input listener at ~line 951 in inventory.html:**

The existing handler already recomputes `mismatch` after each input. Extend it to also flip the Confirm Receipt button's disabled state. After the existing block that toggles `.correction-banner` / `.match-banner` (around line 999), add (still inside the `if(state){...}` block):
```javascript
var hasItemsNow=state.line_items.length>0;
var pendingReason=PENDING_PURCHASES.find(function(pp){return pp.id===pid;});
var isNoAttachmentNow=pendingReason&&pendingReason.reason==='no_attachment_on_bank_tx';
var canConfirmNow=(hasItemsNow&&!mismatch)||(!hasItemsNow&&isNoAttachmentNow);
var btn=form.querySelector('.btn-primary[data-action="confirm-receipt"]');
if(btn){
  if(canConfirmNow)btn.removeAttribute('disabled');
  else btn.setAttribute('disabled','');
}
```
Note: `PENDING_PURCHASES` is the canonical client list — that's where we look up `reason` since the form DOM doesn't carry it as a data attribute today. Confirm `PENDING_PURCHASES` is in scope at this point in the file via `grep -n "PENDING_PURCHASES" inventory.html`. (It's a module-level global used by `renderHistoryList` per line 712.)

ALSO: the existing click handler at line 942 (`else if(action==='confirm-receipt')`) currently fires unconditionally. Add an early-return guard so disabled clicks are a true no-op even if a stale handler fires:
```javascript
} else if(action==='confirm-receipt'){
  if(el.disabled)return;
  captureReviewFormInputs(id);
  confirmReceipt(id);
}
```

**Step 3 — Add CSS for `.btn-primary:disabled`:**

Search inventory.html for an existing `.btn-primary` rule. If a `:disabled` selector doesn't already exist for it, add ONE line near the existing `.btn-primary{...}` rule:
```css
.btn-primary:disabled{opacity:0.5;cursor:not-allowed}
```
If the existing CSS uses single-line minified style (matches CLAUDE.md convention), keep that style — append to the same `<style>` block, not a new one.

**Step 4 — Rebuild service worker:**
```bash
cd /Users/jamal/projects/yumyums/hq && node build-sw.js
```
This regenerates `sw.js` with content-hashed precaching for the updated `inventory.html`. The diff in `sw.js` should be limited to the precache manifest entry for inventory.html.

**Step 5 — Add 3 new Playwright tests at the bottom of `tests/inventory.spec.js`:**

Use the existing `page.route` stub pattern from line 2916. Add a new describe block near the end (after the `'PDF receipt iframe (260607-e1c)'` describe at line 2938):

```javascript
// ─── Phase 260607-fxl: Confirm Receipt disabled state ────────────────────────
test.describe('Confirm Receipt disabled state (260607-fxl)', () => {
  test.beforeEach(async ({ page }) => { await login(page); });

  test('Confirm Receipt button is disabled when totals do not match and items are non-empty', async ({ page }) => {
    await page.route('**/api/v1/inventory/purchases/pending', async route => {
      if (route.request().method() !== 'GET') return route.continue();
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify([{
          id: 'fxl-mismatch', bank_tx_id: 'tx-mismatch', bank_total: -50.00,
          vendor: 'TEST VENDOR', event_date: '2026-06-05',
          reason: 'Receipt could not be parsed automatically',
          items: [{name:'Item A', quantity:1, price:42.00, is_case:false, purchase_item_id:null}],
          created_at: new Date().toISOString(),
        }])
      });
    });
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    const card = page.locator('[data-action="review-pending"][data-id="fxl-mismatch"]');
    await expect(card).toBeVisible();
    await card.click();
    const btn = page.locator('.btn-primary[data-action="confirm-receipt"][data-id="fxl-mismatch"]');
    await expect(btn).toBeVisible();
    await expect(btn).toBeDisabled();
  });

  test('Confirm Receipt button is disabled when items are empty and pending reason is parse-failed', async ({ page }) => {
    await page.route('**/api/v1/inventory/purchases/pending', async route => {
      if (route.request().method() !== 'GET') return route.continue();
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify([{
          id: 'fxl-empty-parsefail', bank_tx_id: 'tx-empty-pf', bank_total: -50.00,
          vendor: 'TEST VENDOR', event_date: '2026-06-05',
          reason: 'Receipt could not be parsed automatically',
          items: [], created_at: new Date().toISOString(),
        }])
      });
    });
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    const card = page.locator('[data-action="review-pending"][data-id="fxl-empty-parsefail"]');
    await expect(card).toBeVisible();
    await card.click();
    const btn = page.locator('.btn-primary[data-action="confirm-receipt"][data-id="fxl-empty-parsefail"]');
    await expect(btn).toBeVisible();
    await expect(btn).toBeDisabled();
  });

  test('Confirm Receipt button is enabled when items match bank total', async ({ page }) => {
    await page.route('**/api/v1/inventory/purchases/pending', async route => {
      if (route.request().method() !== 'GET') return route.continue();
      await route.fulfill({
        status: 200, contentType: 'application/json',
        body: JSON.stringify([{
          id: 'fxl-match', bank_tx_id: 'tx-match', bank_total: -50.00,
          vendor: 'TEST VENDOR', event_date: '2026-06-05',
          reason: 'Receipt could not be parsed automatically',
          items: [{name:'Item A', quantity:1, price:50.00, is_case:false, purchase_item_id:'00000000-0000-0000-0000-000000000001'}],
          created_at: new Date().toISOString(),
        }])
      });
    });
    await page.goto('/inventory.html');
    await page.waitForLoadState('networkidle');
    const card = page.locator('[data-action="review-pending"][data-id="fxl-match"]');
    await expect(card).toBeVisible();
    await card.click();
    const btn = page.locator('.btn-primary[data-action="confirm-receipt"][data-id="fxl-match"]');
    await expect(btn).toBeVisible();
    await expect(btn).toBeEnabled();
  });
});
```

NOTE on the third test: tax is 0 by default in `REVIEW_FORM_STATE` initialization. `lineTotal = 1 * 50.00 = 50.00`; `grandTotal = 50.00 + 0 = 50.00`; `bankTotal = |-50.00| = 50.00`; `matches = true`. Confirm by reading inventory.html's review-form-state init code (search for `REVIEW_FORM_STATE[p.id]=` or similar) before writing the test — if tax defaults to something other than 0, the math won't pan out and the button will render disabled, breaking the positive test.

**Step 6 — Run the 3 new Playwright tests:**
```bash
cd /Users/jamal/projects/yumyums/hq && npx playwright test tests/inventory.spec.js -g "Confirm Receipt disabled state"
```

Per CLAUDE.md: "Only run the new test(s) during iteration, not the full suite." The full suite runs via `task test` if needed for a final check.

**Do NOT:**
- Add a manual "Retry Parse" button (out of scope).
- Touch the mismatch banner styling/content.
- Remove the existing alert/inline-error fallback in confirmReceipt() — disabled button is a defense-in-depth layer; the backend 422 is the real gate.
- Skip `node build-sw.js` — CLAUDE.md and the GSD memory both insist on bumping the SW before commit so the next browser reload picks up the new inventory.html.
- Add `disabled` styling that hides the button — keep it visible but unclickable (matches existing form-field disabled affordances).
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq &amp;&amp; npx playwright test tests/inventory.spec.js -g "Confirm Receipt disabled state"</automated>
  </verify>
  <done>
- `inventory.html` Confirm Receipt button renders with `disabled` when canConfirm is false.
- Input listener at ~line 951 toggles disabled in-place under the SAME canConfirm rule (no full re-render).
- Click handler at ~line 942 short-circuits when `el.disabled` is true.
- `.btn-primary:disabled` CSS adds the visible affordance.
- `sw.js` regenerated via `node build-sw.js` and committed alongside `inventory.html`.
- 3 new Playwright tests in `tests/inventory.spec.js` all pass.
- Commit message: `feat(260607-fxl): fe disables Confirm Receipt button when items empty (non-no-attachment reason) or totals mismatch`
  </done>
</task>

</tasks>

<verification>
Run these commands at the end of execution to confirm the full plan landed cleanly:

```bash
# Backend
cd /Users/jamal/projects/yumyums/hq/backend
go build ./...
go test -count=1 ./internal/receipt/...
go test -count=1 ./internal/inventory/...

# Frontend
cd /Users/jamal/projects/yumyums/hq
node build-sw.js
npx playwright test tests/inventory.spec.js -g "Confirm Receipt disabled state"
npx playwright test tests/inventory.spec.js -g "Receipt sync button|Parser error line|PDF receipt"  # smoke-test prior phases not regressed

# Git — 4 atomic commits should be present
git log --oneline -5 | grep "260607-fxl"
# Expected (newest first):
#   feat(260607-fxl): fe disables Confirm Receipt button ...
#   feat(260607-fxl): handler gates empty-items confirm ...
#   test(260607-fxl): worker retries parse-failed rows ...
#   feat(260607-fxl): worker retries parse-failed rows ...
```

Manual smoke verification (the user will do this — list for awareness, not for Claude to execute):
1. Click Sync Receipts → Restaurant Depot $391.96 row auto-resolves via Sonnet, OR stays pending with parse_error populated.
2. Pending row with attached receipt + empty items → Confirm button visibly disabled.
3. Pending row with attached receipt + 1 item priced wrong → Confirm button visibly disabled.
4. Pending row with attached receipt + 1 item priced right → Confirm button enabled, click confirms.
5. (Backend) POST /confirm with `line_items=[]` for `reason='Receipt could not be parsed automatically'` → 422 envelope.
6. (Backend) POST /confirm with `line_items=[]` for `reason='no_attachment_on_bank_tx'` → 200 (no regression).
7. (Backend) POST /confirm with mismatch line_items → 422 envelope with `line_total` + `bank_total`.
</verification>

<success_criteria>
- FIX A worker retries parse-failed rows when (parse_error IS NULL) AND (items empty) AND (attachments exist); skips when parse_error set or items non-empty.
- FIX A test suite: 3 new tests added, all pass; existing receipt suite still passes.
- FIX B backend rejects empty-items confirm with HTTP 422 + `empty_items_not_allowed` envelope unless reason is `no_attachment_on_bank_tx`; the existing 260605-pk1 no-attachment empty-items flow still returns 200.
- FIX B backend total-mismatch upgraded from text-400 to structured-422 `{error:'total_mismatch',line_total,bank_total}`.
- FIX B FE disables Confirm Receipt button under the same rule, with input-listener live updates and click-handler guard.
- FIX B FE: 3 new Playwright tests pass.
- All 4 commits land with `260607-fxl` prefix in the commit message.
- No new migrations, no new external deps.
- `sw.js` regenerated and committed in the FE commit.
</success_criteria>

<output>
After completion, create `.planning/quick/260607-fxl-retry-parse-failed-rows-and-confirm-gate/260607-fxl-SUMMARY.md` per the standard summary.md template.
</output>
