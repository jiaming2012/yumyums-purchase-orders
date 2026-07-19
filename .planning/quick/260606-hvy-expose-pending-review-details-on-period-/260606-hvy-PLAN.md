---
phase: 260606-hvy
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/inventory/types.go
  - backend/internal/inventory/handler.go
  - backend/internal/inventory/period_summary_test.go
autonomous: true
requirements:
  - 260606-hvy
must_haves:
  truths:
    - "GET /api/v1/inventory/period-summary response includes completeness.pending_review_details as a non-nil array"
    - "pending_review_details[i].id == pending_review_ids[i] for every i (same length, same order)"
    - "Each detail row carries bank_tx_id, vendor, event_date (YYYY-MM-DD), bank_total, and optional reason"
    - "When event_date IS NULL the detail's event_date falls back to (created_at AT TIME ZONE 'America/Chicago')::date"
    - "When vendor IS NULL the detail's vendor serialises as \"\" (not null)"
    - "When reason IS NULL the detail's reason is omitted from JSON via omitempty"
    - "Empty period returns pending_review_details: [] (never null) in JSON"
    - "pending_review_ids remains unchanged and same-ordered (no breaking change to existing consumers)"
  artifacts:
    - path: "backend/internal/inventory/types.go"
      provides: "PendingReviewDetail struct + PendingReviewDetails field on CompletenessBlock"
      contains: "type PendingReviewDetail struct"
    - path: "backend/internal/inventory/handler.go"
      provides: "Extended pending-IDs query + pendingDetails wiring into response"
      contains: "PendingReviewDetails:"
    - path: "backend/internal/inventory/period_summary_test.go"
      provides: "Shape, parity, fallback, null-handling, and empty-period test cases"
      contains: "pending_review_details"
  key_links:
    - from: "backend/internal/inventory/handler.go (PeriodSummaryHandler)"
      to: "backend/internal/inventory/types.go (CompletenessBlock.PendingReviewDetails)"
      via: "response struct literal"
      pattern: "PendingReviewDetails: pendingDetails"
    - from: "backend/internal/inventory/handler.go (pending query SELECT)"
      to: "pending_purchases columns (id, bank_tx_id, vendor, event_date, created_at, bank_total, reason)"
      via: "pool.Query + rows.Scan"
      pattern: "SELECT id::text, bank_tx_id, COALESCE\\(vendor"
---

<objective>
Add a parallel `pending_review_details` array to the `completeness` block on `GET /api/v1/inventory/period-summary` so service-token callers (sales-processor) can render richer pending-review context (vendor, event_date, bank_total, reason) without a second round-trip to the cookie-gated `/purchases/pending` endpoint.

Purely additive — no breaking changes, no schema migration, no feature flag, no version bump. `pending_review_ids` stays unchanged and same-ordered. Field order in the new array MUST match the existing IDs array (same SELECT, same WHERE/ORDER BY, same scan loop).

Purpose: unblock sales-processor's payroll failure messaging — today it only sees an opaque list of UUIDs.
Output: extended Go struct, expanded SQL query + scan, and a tightened test suite that pins the parity + fallback invariants.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md

<interfaces>
<!-- Current CompletenessBlock (backend/internal/inventory/types.go lines 190-194) -->
```go
type CompletenessBlock struct {
    Ready               bool     `json:"ready"`
    PendingReviewIDs    []string `json:"pending_review_ids"`
    UnlinkedLineItemIDs []string `json:"unlinked_line_item_ids"`
}
```

<!-- Current pending-IDs query (backend/internal/inventory/handler.go ~line 1204)
     PeriodSummaryHandler step 3 — pulls IDs only; we extend to also pull
     bank_tx_id, vendor, event_date (Chicago-cast fallback), bank_total, reason. -->
```go
pendingIDs := []string{}
rows, err := pool.Query(r.Context(), `
    SELECT id::text
    FROM pending_purchases
    WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date) BETWEEN $1 AND $2
      AND confirmed_at IS NULL
      AND discarded_at IS NULL
    ORDER BY COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date), created_at`, fromStr, toStr)
```

<!-- Current response wiring (backend/internal/inventory/handler.go ~line 1311) -->
```go
resp := PeriodSummary{
    From:               fromStr,
    ...
    Completeness: CompletenessBlock{
        Ready:               len(pendingIDs) == 0 && len(unlinkedIDs) == 0,
        PendingReviewIDs:    pendingIDs,
        UnlinkedLineItemIDs: unlinkedIDs,
    },
}
```

<!-- Existing test helpers (backend/internal/inventory/period_summary_test.go) -->
- `resetFixtures(t)` — TRUNCATE between subtests
- `callHandler(t, from, to)` → returns (statusCode, PeriodSummary) with default ["COGS"] allowlist
- `insertPendingPurchase(t, createdAt, confirmed, discarded)` (line ~175)
- `insertPendingPurchaseWithBankTotal(t, bankTxID, bankTotal, createdAt)` (line ~224)
- `insertPendingPurchaseWithEventDate(t, bankTxID, eventDate, createdAt, reason)` (line ~241) — empty string = NULL for eventDate/reason
- `TestMain` skips tests if DB_TEST_URL unreachable — use the same `if testPool == nil { t.Skip(...) }` guard
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Extend CompletenessBlock + add PendingReviewDetail struct in types.go</name>
  <files>backend/internal/inventory/types.go</files>
  <action>
Edit `backend/internal/inventory/types.go`. Modify the `CompletenessBlock` struct (currently lines 190-194) and append a new `PendingReviewDetail` struct immediately after it. The final shape MUST be exactly:

```go
type CompletenessBlock struct {
    Ready                bool                  `json:"ready"`
    PendingReviewIDs     []string              `json:"pending_review_ids"`
    PendingReviewDetails []PendingReviewDetail `json:"pending_review_details"`
    UnlinkedLineItemIDs  []string              `json:"unlinked_line_item_ids"`
}

// PendingReviewDetail is one row of operator-facing context per pending
// review. Exposed on /period-summary so service-token callers
// (sales-processor) can render a meaningful failure message without a
// second round trip to the cookie-auth-only /purchases/pending
// endpoint.
type PendingReviewDetail struct {
    ID        string  `json:"id"`
    BankTxID  string  `json:"bank_tx_id"`
    Vendor    string  `json:"vendor"`     // "" when receipt parser couldn't extract one
    EventDate string  `json:"event_date"` // YYYY-MM-DD; falls back to created_at::date
    BankTotal float64 `json:"bank_total"`
    Reason    *string `json:"reason,omitempty"`
}
```

Field ordering, JSON tag names, types, and comments MUST match exactly — sales-processor will consume these tag names verbatim. Keep the existing CompletenessBlock doc comment (above line 190) intact. Do NOT add any other fields, helpers, or methods.

Compile-only validation in this task — handler.go still references the old struct literal (no `PendingReviewDetails:` field yet), which is fine because Go zero-values the new slice field when omitted from the literal. The build MUST stay green at the end of this task.
  </action>
  <verify>
    <automated>cd backend && go build ./...</automated>
  </verify>
  <done>
`types.go` declares the new `PendingReviewDetails []PendingReviewDetail` field in the documented position and a new exported `PendingReviewDetail` struct with all six fields and exact JSON tags. `go build ./...` succeeds from the `backend/` directory.
  </done>
</task>

<task type="auto">
  <name>Task 2: Extend pending-purchases query + wire pendingDetails into response in handler.go</name>
  <files>backend/internal/inventory/handler.go</files>
  <action>
Edit `PeriodSummaryHandler` in `backend/internal/inventory/handler.go` — the pending-IDs section starting around line 1204 (step 3 comment "Pending review IDs — receipts whose business date falls in the period…").

Two changes inside the same function:

(a) Replace the existing `SELECT id::text` query and its scan loop. The new query MUST be:

```sql
SELECT id::text,
       bank_tx_id,
       COALESCE(vendor, '') AS vendor,
       COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)::text AS event_date,
       bank_total,
       reason
FROM pending_purchases
WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date) BETWEEN $1 AND $2
  AND confirmed_at IS NULL
  AND discarded_at IS NULL
ORDER BY COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date), created_at
```

The WHERE and ORDER BY clauses are IDENTICAL to the current query — this guarantees row order matches what callers see today in `pending_review_ids`.

Initialize BOTH slices to non-nil empty before the query so JSON serialization yields `[]` not `null`:

```go
pendingIDs := []string{}
pendingDetails := []PendingReviewDetail{}
```

Scan each row into a local `PendingReviewDetail` (declared inside the loop), then append `d.ID` to `pendingIDs` and `d` to `pendingDetails` in the same iteration. This guarantees the parity invariant by construction — `pendingIDs[i] == pendingDetails[i].ID` for every i. `reason` is nullable in the DB; scan it into `*string` (matching the struct field) so SQL NULL → Go nil → JSON omitted via `omitempty`.

Keep error handling and logging IDENTICAL to the existing pattern: `log.Printf("PeriodSummary pending scan: %v", err)` on scan error, `log.Printf("PeriodSummary pending query: %v", err)` on query error, `log.Printf("PeriodSummary pending rows.Err: %v", err)` on iteration error. Each returns `writeError(w, http.StatusInternalServerError, "internal_error")`. Do NOT add new log strings or rename existing ones — sales-processor's log-tailing alerts grep on these.

(b) In the `resp := PeriodSummary{...}` literal (around line 1311), add `PendingReviewDetails: pendingDetails,` between `PendingReviewIDs:` and `UnlinkedLineItemIDs:` to match the struct field ordering from Task 1. Final shape:

```go
Completeness: CompletenessBlock{
    Ready:                len(pendingIDs) == 0 && len(unlinkedIDs) == 0,
    PendingReviewIDs:     pendingIDs,
    PendingReviewDetails: pendingDetails,
    UnlinkedLineItemIDs:  unlinkedIDs,
},
```

Do NOT change the `Ready` calculation — it MUST remain `len(pendingIDs) == 0 && len(unlinkedIDs) == 0` (the details array is informational only; readiness is still gated on the same two ID lists).
  </action>
  <verify>
    <automated>cd backend && go build ./... && go vet ./internal/inventory/...</automated>
  </verify>
  <done>
Handler returns a non-nil `pending_review_details` array on every successful response. Each element carries id/bank_tx_id/vendor/event_date/bank_total/reason. NULL vendor → `""`. NULL event_date → Chicago-cast `created_at`. NULL reason → omitted via omitempty. Order is identical to `pending_review_ids`. `go build` and `go vet` both clean.
  </done>
</task>

<task type="auto" tdd="true">
  <name>Task 3: Add pending_review_details test cases to period_summary_test.go</name>
  <files>backend/internal/inventory/period_summary_test.go</files>
  <behavior>
    - Shape: response body has `pending_review_details` as a non-nil array (never null)
    - Parity-with-IDs: `len(details) == len(ids)` and `details[i].ID == ids[i]` for every i
    - Field population: detail row reflects inserted vendor / event_date / bank_total / reason
    - Null event_date fallback: a row with event_date=NULL + created_at='2026-05-29 22:02:00+00' produces `EventDate == "2026-05-29"` (America/Chicago calendar date)
    - Null vendor: a row with vendor=NULL produces `Vendor == ""` (empty string, not omitted, not null)
    - Null reason: a row with reason=NULL produces `Reason == nil` (pointer nil → JSON omitted via omitempty)
    - Empty period: when no pending rows exist in range, the raw response JSON contains `"pending_review_details":[]` (NOT `:null`)
  </behavior>
  <action>
Append new subtests to the existing `TestPeriodSummary` function in `backend/internal/inventory/period_summary_test.go`. Reuse the established patterns from the file:
- Guard with `if testPool == nil { t.Skip("DB_TEST_URL not reachable; skipping integration test") }` (already at top of TestPeriodSummary, no need to re-add inside subtests)
- Call `resetFixtures(t)` at the top of every subtest
- Use the `const from = "2026-05-25"` / `const to = "2026-05-31"` window already in scope, OR declare local bounds when a subtest needs a different range
- Use `callHandler(t, from, to)` for the typed-struct path
- For the empty-period raw-JSON assertion, call `PeriodSummaryHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)` directly so you can inspect `rec.Body.String()` and substring-match `"pending_review_details":[]`

Add these subtests under the existing `t.Run` blocks. Suggested names:

1. `t.Run("pending_review_details parity with pending_review_ids", ...)` — insert 2-3 pending rows with distinct event_dates in range using `insertPendingPurchaseWithEventDate`, call handler, assert `len(got.Completeness.PendingReviewDetails) == len(got.Completeness.PendingReviewIDs)` and loop asserting `details[i].ID == ids[i]`.

2. `t.Run("pending_review_details populates vendor/event_date/bank_total/reason", ...)` — insert one row directly via raw SQL (since the existing helpers don't accept all four fields at once):
   ```go
   var id string
   q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at)
         VALUES ('mx-100', -87.50, 'Restaurant Depot', '[]'::jsonb, '2026-05-28'::date, 'tax_mismatch', '2026-05-28 12:00:00-05:00'::timestamptz)
         RETURNING id::text`
   testPool.QueryRow(t.Context(), q).Scan(&id)
   ```
   Call handler, assert `details[0].ID == id`, `BankTxID == "mx-100"`, `Vendor == "Restaurant Depot"`, `EventDate == "2026-05-28"`, `BankTotal == -87.50`, `Reason != nil && *Reason == "tax_mismatch"`.

3. `t.Run("pending_review_details event_date falls back to Chicago cast of created_at", ...)` — insert a row with `event_date IS NULL` and `created_at='2026-05-29 22:02:00+00'::timestamptz` via `insertPendingPurchaseWithEventDate(t, "mx-200", "", "2026-05-29 22:02:00+00", "")`. The America/Chicago cast of 2026-05-29 22:02 UTC is 2026-05-29 17:02 Chicago → calendar date `2026-05-29`. Assert `details[0].EventDate == "2026-05-29"`.

4. `t.Run("pending_review_details vendor=NULL serializes as empty string", ...)` — insert directly via raw SQL with `vendor=NULL`:
   ```go
   q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, created_at)
         VALUES ('mx-300', -10.00, NULL, '[]'::jsonb, '2026-05-27'::date, '2026-05-27 12:00:00-05:00'::timestamptz)
         RETURNING id::text`
   ```
   Call handler, assert `details[0].Vendor == ""` (the `COALESCE(vendor, '')` in the SQL ensures this).

5. `t.Run("pending_review_details reason=NULL omitted from JSON", ...)` — use `insertPendingPurchaseWithEventDate(t, "mx-400", "2026-05-27", "2026-05-27 12:00:00-05:00", "")` (empty reason → NULL). Call handler via the raw-recorder path (not `callHandler`) so you can inspect `rec.Body.String()`. Assert (a) the typed `details[0].Reason == nil` AND (b) the raw JSON body does NOT contain `"reason":` for that row (since omitempty drops nil pointers).

6. `t.Run("pending_review_details serializes as [] when empty period", ...)` — no inserts (just `resetFixtures(t)`). Build a raw httptest request, call `PeriodSummaryHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)`, assert `rec.Code == 200`, and substring-assert `strings.Contains(rec.Body.String(), "\"pending_review_details\":[]")`. If `strings` isn't imported, add it to the imports.

Use the existing import block patterns. If `strings` isn't already imported (it isn't — only `bytes`, `context`, `encoding/json`, `net/http`, `net/http/httptest`, `os`, `strconv`, `testing` plus the project packages are), add `"strings"` in the std-lib block.

Do NOT modify existing subtests or helpers. Only append new subtests.
  </action>
  <verify>
    <automated>cd backend && go test ./internal/inventory/... -run TestPeriodSummary -count=1</automated>
  </verify>
  <done>
All six new subtests pass under `go test ./internal/inventory/... -run TestPeriodSummary -count=1`. Existing subtests still pass (no regressions). When `DB_TEST_URL` is unreachable, the subtests cleanly skip via the existing `testPool == nil` guard.
  </done>
</task>

</tasks>

<verification>
End-to-end checks after all three tasks:

1. **Build clean:** `cd backend && go build ./...` exits 0.
2. **Vet clean:** `cd backend && go vet ./internal/inventory/...` exits 0.
3. **Full inventory tests:** `cd backend && go test ./internal/inventory/... -count=1` passes (no existing test breaks).
4. **Backward compat:** existing assertions on `pending_review_ids` still pass — same query, same ORDER BY, no semantic shift.
5. **Manual JSON spot-check (optional):** hit `/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31` against a dev DB with at least one pending row; confirm `pending_review_details` array length matches `pending_review_ids` and field values render correctly.
</verification>

<success_criteria>
- `CompletenessBlock` has a `PendingReviewDetails []PendingReviewDetail` field with exact JSON tag `pending_review_details`
- New `PendingReviewDetail` struct has all six fields with the exact JSON tags specified
- Handler's pending query returns id + bank_tx_id + vendor (COALESCE'd) + event_date (COALESCE'd via Chicago cast) + bank_total + reason
- `pending_review_ids` and `pending_review_details` are guaranteed same-length and same-order by construction (single SELECT, single scan loop)
- Empty period → `"pending_review_details":[]` (never null) in JSON output
- `Ready` calculation unchanged (still `len(pendingIDs) == 0 && len(unlinkedIDs) == 0`)
- All six new test cases pass; no existing test regresses
- No schema migration, no version bump, no feature flag, no breaking change to `pending_review_ids`
</success_criteria>

<output>
After completion, create `.planning/quick/260606-hvy-expose-pending-review-details-on-period-/260606-hvy-SUMMARY.md`.
</output>
