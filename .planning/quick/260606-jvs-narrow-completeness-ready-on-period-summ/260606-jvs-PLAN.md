---
phase: 260606-jvs
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql
  - backend/internal/receipt/worker.go
  - backend/internal/inventory/handler.go
  - backend/internal/inventory/period_summary_test.go
autonomous: true
requirements:
  - 260606-jvs
must_haves:
  truths:
    - "GET /period-summary completeness.ready == false IFF (∃ pending row in period with mercury_category ∈ cogsAllowlist AND reason = 'no_attachment_on_bank_tx') OR (∃ unlinked line item)"
    - "pending_review_ids and pending_review_details on response include ONLY blocking pending rows (COGS-category + no attachment). Non-blocking pending stays out of these arrays."
    - "cogs_excl_tax = SUM(confirmed purchase_line_items in period, allowlist) + SUM(ABS(bank_total)) of non-blocking eligible pending (mercury_category ∈ allowlist AND reason != 'no_attachment_on_bank_tx' AND unconfirmed AND undiscarded AND in period)"
    - "by_vendor merges confirmed events with non-blocking eligible pending rows; pending vendor names join vendors.name via LOWER(TRIM()) case-insensitive match; unmatched names get their own row with vendor_id == \"\""
    - "Confirmed RD ($100) + case-B Save A Lot pending ($19) in same period → cogs_excl_tax == 119; by_vendor has RD row $100 + Save A Lot row $19, both with real vendor_id"
    - "pending_purchases.mercury_category is populated on insertPendingPurchase from tx.CategoryData.Name (nullable) and refreshed by the worker on every poll via UPDATE … IS DISTINCT FROM"
    - "NULL mercury_category fails = ANY($allowlist) — uncategorised pending rows neither block payroll nor roll into COGS"
    - "Out-of-period blocking pending row is excluded from pending_review_ids (existing date filter still applies)"
  artifacts:
    - path: "backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql"
      provides: "ALTER TABLE pending_purchases ADD COLUMN mercury_category TEXT (goose Up/Down with BEGIN/COMMIT, mirroring 0065)"
      contains: "ADD COLUMN mercury_category"
    - path: "backend/internal/receipt/worker.go"
      provides: "Refresh pass for pending_purchases.mercury_category in the per-tx loop, plus mercury_category written on insertPendingPurchase INSERT"
      contains: "UPDATE pending_purchases"
    - path: "backend/internal/inventory/handler.go"
      provides: "Three updated SQL blocks in PeriodSummaryHandler: cogs aggregate UNIONs eligible pending, by_vendor merges eligible pending, pending-IDs query narrowed to blocking rows only"
      contains: "reason != 'no_attachment_on_bank_tx'"
    - path: "backend/internal/inventory/period_summary_test.go"
      provides: "10 new test cases pinning the 2×2 (category × reason) truth table, NULL category, date filter, pending_review_details parity, by_vendor match, by_vendor unmatched, vendor name fuzz"
      contains: "no_attachment_on_bank_tx"
  key_links:
    - from: ".planning/quick/260606-jvs-narrow-completeness-ready-on-period-summ/260606-jvs-HANDOFF.md"
      to: "backend/internal/inventory/handler.go (PeriodSummaryHandler steps 1, 2, 3)"
      via: "three-query rewrite spec'd verbatim in HANDOFF.md §3"
      pattern: "WITH events|WITH confirmed|mercury_category = ANY"
    - from: "backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql"
      to: "backend/internal/db/migrations/0065_mercury_category_on_purchase_events.sql"
      via: "mirror format (goose Up/Down, BEGIN/COMMIT, COMMENT ON COLUMN)"
      pattern: "ADD COLUMN mercury_category TEXT"
    - from: "backend/internal/receipt/worker.go (per-tx loop)"
      to: "pending_purchases.mercury_category column"
      via: "UPDATE … IS DISTINCT FROM mirroring the purchase_events refresh at worker.go:85-95"
      pattern: "UPDATE pending_purchases\\s+SET mercury_category"
    - from: ".planning/quick/260606-hvy-expose-pending-review-details-on-period-/260606-hvy-PLAN.md"
      to: "current /period-summary response shape (pending_review_ids + pending_review_details)"
      via: "shipped immediately before this task — same scan loop is being narrowed, not restructured"
      pattern: "PendingReviewDetails"
---

<objective>
Narrow `/api/v1/inventory/period-summary` `completeness.ready` so it only fires `false` when a pending row would actually break the books — and roll non-blocking food-category pending into the COGS aggregate at their bank totals.

After this ships:
- **Blocking** (still in `pending_review_ids` / `pending_review_details`, still flips `ready=false`): pending row in period with `mercury_category ∈ HQ_COGS_CATEGORY_ALLOWLIST` AND `reason = 'no_attachment_on_bank_tx'` — i.e. a real food-category card swipe with no receipt photo on file.
- **Non-blocking, rolled into COGS**: pending row in period with `mercury_category ∈ allowlist` AND `reason != 'no_attachment_on_bank_tx'` (e.g. receipt attached but Claude couldn't itemise it). Adds `ABS(bank_total)` to `cogs_excl_tax`, `cogs_incl_tax`, and per-vendor totals.
- **Non-blocking, not in COGS**: everything else (non-food category, NULL category, etc.). Stays in `pending_purchases` for operator triage in the Inventory UI; invisible to the payroll endpoint.

Purpose: stop blocking weekly payroll on non-food card swipes (Amazon refunds) and on food pending whose receipt is attached but parse-failed (Save A Lot). The only legitimate blocker — food charge with no receipt photo (Restaurant Depot) — keeps blocking.

Sales-processor needs zero code changes; it already hard-fails on `ready=false` and renders `pending_review_details`. The endpoint's contract just becomes more honest about what "ready" means.

Output: one new migration, one worker patch (populate + refresh), three handler SQL rewrites (one narrow + two UNIONs), and 10 new test cases pinning the new truth table + COGS roll-up + vendor merge semantics.

See `260606-jvs-HANDOFF.md` for the verbatim SQL and full design rationale — this plan is execution-only.
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md
@.planning/quick/260606-jvs-narrow-completeness-ready-on-period-summ/260606-jvs-HANDOFF.md
@.planning/quick/260606-hvy-expose-pending-review-details-on-period-/260606-hvy-SUMMARY.md

<interfaces>
<!-- Goose migration format (mirror 0065 exactly — NOT the "+migrate Up" form from HANDOFF.md): -->
```sql
-- +goose Up
BEGIN;
ALTER TABLE … ;
COMMIT;

-- +goose Down
BEGIN;
ALTER TABLE … ;
COMMIT;
```

<!-- Current insertPendingPurchase INSERT (backend/internal/receipt/worker.go:363-377) — 9 columns -->
```go
_, err = pool.Exec(ctx,
    `INSERT INTO pending_purchases
     (bank_tx_id, bank_total, vendor, event_date, tax, total, items, reason, receipt_url)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
     ON CONFLICT DO NOTHING`,
    tx.ID, tx.Amount, vendor,
    nullableString(eventDate),
    nullableFloat64(summary.Tax), nullableFloat64(summary.Total),
    itemsJSON,
    nullableString(reason),
    nullableString(receiptURL),
)
```

<!-- Existing per-tx mercury_category refresh for purchase_events (backend/internal/receipt/worker.go:85-95) — the pattern to mirror for pending_purchases -->
```go
if tx.CategoryData != nil {
    _, refreshErr := cfg.Pool.Exec(ctx,
        `UPDATE purchase_events
         SET mercury_category = $1
         WHERE bank_tx_id = $2
           AND (mercury_category IS DISTINCT FROM $1)`,
        tx.CategoryData.Name, tx.ID)
    if refreshErr != nil {
        log.Printf("receipt worker: refresh mercury_category for tx %s: %v (continuing)", tx.ID, refreshErr)
    }
}
```

<!-- Helpers already in period_summary_test.go (use these; only add a new helper if a category param is genuinely required): -->
- `insertVendor(t, name string) string` — returns vendor UUID
- `insertEventAndLineWithCategory(t, vendorID, eventDate, tax, total, price, qty, purchaseItemID, category string)` — confirmed event with mercury_category set
- `insertPendingPurchaseWithEventDate(t, bankTxID, eventDate, createdAt, reason string) string` — pending row with reason but NO category. Either extend this signature OR add `insertPendingPurchaseWithCategory(t, bankTxID, eventDate, createdAt, reason, category, vendor string, bankTotal float64) string` — pick whichever is cleaner; existing callers must keep compiling.
- `callHandlerWithAllowlist(t, from, to string, allowlist []string) (int, PeriodSummary)` — invokes the handler with a specific cogsAllowlist (use `[]string{"Food, Beverage & Groceries"}` or whatever sentinel the existing tests use).

<!-- HANDLER.GO LINE RANGE DRIFT (verified on disk 2026-06-06):
     HANDOFF.md was written against a slightly earlier snapshot. Actual current ranges:
       - COGS aggregate query: handler.go:1118-1142    (matches HANDOFF.md)
       - by_vendor query:      handler.go:1144-1195    (matches HANDOFF.md)
       - pending-IDs query:    handler.go:1197-1238    (HANDOFF.md said 1201-1228 — drifted because 260606-hvy added the pending_review_details scan into the same loop. The query body itself starts at line 1206 with `SELECT id::text`; the surrounding loop ends at 1238.)
     The query bodies match HANDOFF.md byte-for-byte. Patch them in place; do NOT rewrite from scratch. -->
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add 0066 migration — mercury_category on pending_purchases</name>
  <files>backend/internal/db/migrations/0066_mercury_category_on_pending_purchases.sql</files>
  <action>
Create a new file mirroring `0065_mercury_category_on_purchase_events.sql`. Use **goose** format (`-- +goose Up` / `-- +goose Down`) with BEGIN/COMMIT — NOT the `+migrate Up` form shown in HANDOFF.md (that was the spec author's pseudocode; this codebase is on goose, see 0065).

Contents (exact):

```sql
-- +goose Up
BEGIN;

-- Mercury's categoryData.name at the time HQ ingested the bank transaction.
-- Nullable because: (a) existing rows pre-date this column, (b) future rows
-- where Mercury hasn't been categorized yet by the classify pipeline.
-- The receipt worker re-syncs this column on every poll within its 14-day
-- lookback window via UPDATE … IS DISTINCT FROM, so NULLs self-heal as
-- Mercury catches up. /period-summary uses this column to decide whether a
-- pending row blocks payroll: COGS-category + no_attachment_on_bank_tx
-- blocks; everything else either rolls into COGS (food + attached receipt
-- that parse-failed) or stays out of the payroll endpoint entirely
-- (non-food, NULL category). See 260606-jvs-HANDOFF.md.
ALTER TABLE pending_purchases
  ADD COLUMN mercury_category TEXT;

COMMIT;

-- +goose Down
BEGIN;

ALTER TABLE pending_purchases
  DROP COLUMN mercury_category;

COMMIT;
```

(The HANDOFF.md COMMENT ON COLUMN statement is folded into the SQL comment block above — goose-format migrations in this repo use comments not COMMENT ON COLUMN.)
  </action>
  <verify>
    <automated>cd backend &amp;&amp; task migrate-up &amp;&amp; psql "$DATABASE_URL" -c "\d pending_purchases" | grep mercury_category</automated>
  </verify>
  <done>Column `mercury_category text` exists on `pending_purchases`. `task migrate-down` cleanly removes it. Existing rows have NULL.</done>
</task>

<task type="auto">
  <name>Task 2: worker.go — populate mercury_category on insert + refresh on every poll</name>
  <files>backend/internal/receipt/worker.go</files>
  <action>
Two edits to `backend/internal/receipt/worker.go`:

**Edit A — refresh pass for pending_purchases (mirror the existing purchase_events refresh at lines 85-95).**

Locate the existing block:

```go
if tx.CategoryData != nil {
    _, refreshErr := cfg.Pool.Exec(ctx,
        `UPDATE purchase_events
         SET mercury_category = $1
         WHERE bank_tx_id = $2
           AND (mercury_category IS DISTINCT FROM $1)`,
        tx.CategoryData.Name, tx.ID)
    if refreshErr != nil {
        log.Printf("receipt worker: refresh mercury_category for tx %s: %v (continuing)", tx.ID, refreshErr)
    }
}
```

Immediately after its closing brace (before the existing `backfillPendingVendor` call around line 103), add an analogous block for `pending_purchases`:

```go
// Same refresh pattern for pending_purchases — catches the race where a
// pending row was created before Mercury's classify pipeline tagged a
// category. Idempotent via IS DISTINCT FROM. Runs for cached AND new
// transactions (before the `already` short-circuit) so the next worker
// poll backfills any row inside the 14-day lookback window.
if tx.CategoryData != nil {
    _, refreshErr := cfg.Pool.Exec(ctx,
        `UPDATE pending_purchases
         SET mercury_category = $1
         WHERE bank_tx_id = $2
           AND (mercury_category IS DISTINCT FROM $1)`,
        tx.CategoryData.Name, tx.ID)
    if refreshErr != nil {
        log.Printf("receipt worker: refresh pending_purchases.mercury_category for tx %s: %v (continuing)", tx.ID, refreshErr)
    }
}
```

**Edit B — extend `insertPendingPurchase` INSERT to write mercury_category.**

In `insertPendingPurchase` (around line 343-382), change the INSERT from 9 columns to 10. Derive the category nil-safely the same way `createPurchaseEvent` does (lines 282-286):

```go
// Derive mercury_category (nil-safe — NULL when Mercury hasn't classified yet).
var mercuryCategory string
if tx.CategoryData != nil {
    mercuryCategory = tx.CategoryData.Name
}

_, err = pool.Exec(ctx,
    `INSERT INTO pending_purchases
     (bank_tx_id, bank_total, vendor, event_date, tax, total, items, reason, receipt_url, mercury_category)
     VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
     ON CONFLICT DO NOTHING`,
    tx.ID,
    tx.Amount,
    vendor,
    nullableString(eventDate),
    nullableFloat64(summary.Tax),
    nullableFloat64(summary.Total),
    itemsJSON,
    nullableString(reason),
    nullableString(receiptURL),
    nullableString(mercuryCategory),
)
```

Use `nullableString(mercuryCategory)` (already in this file — same idiom as `nullableString(receiptURL)` on the same INSERT) so an empty Mercury category serialises as SQL NULL rather than empty string. This keeps the `= ANY($allowlist)` semantics consistent between insert-time and refresh-time.

Do NOT touch the vendor-fallback block (lines 358-361) or any other logic in this function.
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go build ./internal/receipt/... &amp;&amp; go vet ./internal/receipt/...</automated>
  </verify>
  <done>worker.go compiles. The new refresh block lives right after the existing purchase_events refresh, before backfillPendingVendor. insertPendingPurchase writes 10 columns with mercury_category derived nil-safely from tx.CategoryData.</done>
</task>

<task type="auto">
  <name>Task 3: handler.go — three SQL rewrites (narrow pending + UNION pending into COGS + UNION pending into by_vendor)</name>
  <files>backend/internal/inventory/handler.go</files>
  <action>
Three in-place SQL patches inside `PeriodSummaryHandler`. **The HANDOFF.md SQL bodies are byte-for-byte correct** — copy them verbatim; do NOT rewrite from scratch. The Go scaffolding (Scan, error handling, append loops) stays identical.

**3a. Tighten the pending-IDs query (handler.go:1197-1238, query body starting line 1206).**

The current SELECT already pulls 6 columns into `PendingReviewDetail` (id, bank_tx_id, vendor, event_date, bank_total, reason). Keep the column list and the scan loop EXACTLY as-is — only add two WHERE clauses and one query parameter:

```go
pendingIDs := []string{}
pendingDetails := []PendingReviewDetail{}
rows, err := pool.Query(r.Context(), `
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
      AND mercury_category = ANY($3)                    -- NEW: COGS-category only
      AND reason = 'no_attachment_on_bank_tx'           -- NEW: no receipt uploaded
    ORDER BY COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date), created_at`,
    fromStr, toStr, cogsAllowlist)
```

The `cogsAllowlist` variable is already in scope at this site (it's used by the COGS aggregate query at line 1137 and the by_vendor query at line 1175). Add it as the 3rd Query arg — Go binding for `$3`.

`Completeness.Ready` construction at the bottom (`len(pendingIDs) == 0 && len(unlinkedIDs) == 0`) is **unchanged** — the SQL narrowing is the entire behavioural change. `Ready` now sees fewer (the right) rows.

**3b. UNION eligible pending into the COGS aggregate (handler.go:1118-1142).**

Replace the entire `pool.QueryRow(...)` call body — `&cogsExcl, &cogsIncl, &eventCount` scan stays unchanged, only the SQL changes. Use the HANDOFF.md §3b SQL verbatim:

```go
err := pool.QueryRow(r.Context(), `
    WITH events AS (
        SELECT id, tax
        FROM purchase_events
        WHERE event_date BETWEEN $1 AND $2
          AND mercury_category = ANY($3)
    ),
    lines AS (
        SELECT ROUND(COALESCE(SUM(pli.quantity * pli.price), 0)::numeric, 2) AS total
        FROM purchase_line_items pli
        WHERE pli.purchase_event_id IN (SELECT id FROM events)
    ),
    pending AS (
        SELECT ROUND(COALESCE(SUM(ABS(bank_total)), 0)::numeric, 2) AS total,
               COUNT(*)                                              AS event_count
        FROM pending_purchases
        WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)
                BETWEEN $1 AND $2
          AND confirmed_at IS NULL
          AND discarded_at IS NULL
          AND mercury_category = ANY($3)
          AND reason != 'no_attachment_on_bank_tx'   -- skip the blocking ones
    ),
    event_tax AS (
        SELECT COALESCE(SUM(tax), 0)::numeric AS total FROM events
    )
    SELECT
        (SELECT total FROM lines) + (SELECT total FROM pending)                       AS cogs_excl_tax,
        (SELECT total FROM lines) + (SELECT total FROM pending)
            + (SELECT total FROM event_tax)                                           AS cogs_incl_tax,
        (SELECT COUNT(*) FROM events) + (SELECT event_count FROM pending)             AS event_count`,
    fromStr, toStr, cogsAllowlist).Scan(&cogsExcl, &cogsIncl, &eventCount)
```

Notes captured straight from HANDOFF.md §3b (already vetted by the spec author — do not second-guess):
- `ABS(bank_total)`: Mercury debits are negative; normalise to positive at the pending boundary.
- Pending rows contribute `bank_total` to both `cogs_excl_tax` and `cogs_incl_tax`; we accept ~5% tax inaccuracy per pending row vs. excluding it entirely. Operator confirming the receipt later replaces this with parsed `total + tax`.
- `reason != 'no_attachment_on_bank_tx'` keeps the blocking rows out of COGS (they'd be moot anyway since `Ready=false`, but keeps the data model honest).

**3c. UNION eligible pending into by_vendor (handler.go:1144-1195).**

Replace the SQL body of the `pool.Query(...)` call (`&v.VendorID, &v.VendorName, &v.TotalExclTax, &v.TotalInclTax, &v.TripCount` scan stays unchanged). Use HANDOFF.md §3c SQL verbatim — this is a 3-CTE union with an outer GROUP BY:

```go
rowsV, err := pool.Query(r.Context(), `
    WITH confirmed AS (
        SELECT
            v.id::text                                                                AS vendor_id,
            v.name                                                                    AS vendor_name,
            ROUND(COALESCE(SUM(pli.quantity * pli.price), 0)::numeric, 2)             AS total_excl_tax,
            ROUND(
              COALESCE(SUM(pli.quantity * pli.price), 0)::numeric
              + COALESCE(
                  (SELECT SUM(pe2.tax)
                     FROM purchase_events pe2
                    WHERE pe2.vendor_id = v.id
                      AND pe2.event_date BETWEEN $1 AND $2
                      AND pe2.mercury_category = ANY($3)),
                  0
                )::numeric,
              2
            )                                                                         AS total_incl_tax,
            COUNT(DISTINCT pe.id)                                                     AS trip_count
        FROM purchase_events pe
        JOIN vendors v                    ON v.id = pe.vendor_id
        LEFT JOIN purchase_line_items pli ON pli.purchase_event_id = pe.id
        WHERE pe.event_date BETWEEN $1 AND $2
          AND pe.mercury_category = ANY($3)
        GROUP BY v.id, v.name
    ),
    pending_eligible AS (
        SELECT id, bank_total, vendor
        FROM pending_purchases
        WHERE COALESCE(event_date, (created_at AT TIME ZONE 'America/Chicago')::date)
                BETWEEN $1 AND $2
          AND confirmed_at IS NULL
          AND discarded_at IS NULL
          AND mercury_category = ANY($3)
          AND reason != 'no_attachment_on_bank_tx'
    ),
    pending_matched AS (
        SELECT
            v.id::text                                            AS vendor_id,
            v.name                                                AS vendor_name,
            ROUND(SUM(ABS(pe.bank_total))::numeric, 2)            AS total_excl_tax,
            ROUND(SUM(ABS(pe.bank_total))::numeric, 2)            AS total_incl_tax,
            COUNT(*)                                              AS trip_count
        FROM pending_eligible pe
        JOIN vendors v ON LOWER(TRIM(v.name)) = LOWER(TRIM(pe.vendor))
        GROUP BY v.id, v.name
    ),
    pending_unmatched AS (
        SELECT
            ''::text                                                                  AS vendor_id,
            COALESCE(NULLIF(TRIM(pe.vendor), ''), '(unknown vendor)')                 AS vendor_name,
            ROUND(SUM(ABS(pe.bank_total))::numeric, 2)                                AS total_excl_tax,
            ROUND(SUM(ABS(pe.bank_total))::numeric, 2)                                AS total_incl_tax,
            COUNT(*)                                                                  AS trip_count
        FROM pending_eligible pe
        WHERE NOT EXISTS (
            SELECT 1 FROM vendors v
            WHERE LOWER(TRIM(v.name)) = LOWER(TRIM(pe.vendor))
        )
        GROUP BY pe.vendor
    )
    SELECT vendor_id, vendor_name,
           SUM(total_excl_tax)  AS total_excl_tax,
           SUM(total_incl_tax)  AS total_incl_tax,
           SUM(trip_count)::int AS trip_count
    FROM (
        SELECT vendor_id, vendor_name, total_excl_tax, total_incl_tax, trip_count FROM confirmed
        UNION ALL
        SELECT vendor_id, vendor_name, total_excl_tax, total_incl_tax, trip_count FROM pending_matched
        UNION ALL
        SELECT vendor_id, vendor_name, total_excl_tax, total_incl_tax, trip_count FROM pending_unmatched
    ) combined
    GROUP BY vendor_id, vendor_name
    ORDER BY total_excl_tax DESC, vendor_name ASC`, fromStr, toStr, cogsAllowlist)
```

The outer GROUP BY collapses confirmed + matched-pending into one row per real vendor. `vendor_id = ''` rows (unmatched pending) stay separate.

**Do not change:**
- The `Completeness.Ready = len(pendingIDs) == 0 && len(unlinkedIDs) == 0` construction
- The PendingReviewDetail scan loop
- The unlinked line items query (step 4 at lines 1286-1316)
- The tracked_bank_tx_ids query (step 3b at lines 1240-1284)
- Any handler other than PeriodSummaryHandler
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go build ./internal/inventory/... &amp;&amp; go vet ./internal/inventory/...</automated>
  </verify>
  <done>handler.go compiles. All three queries use $1=fromStr, $2=toStr, $3=cogsAllowlist. The pending query has the two new WHERE clauses. The cogs aggregate query has the new `pending` and `event_tax` CTEs. The by_vendor query is the 3-CTE form with outer GROUP BY. Completeness.Ready construction at the bottom is unchanged.</done>
</task>

<task type="auto">
  <name>Task 4: period_summary_test.go — add 10 cases covering the new truth table + COGS roll-up + vendor merge</name>
  <files>backend/internal/inventory/period_summary_test.go</files>
  <action>
Add a new sub-test block inside `TestPeriodSummary` (or a new top-level test if the file structure is cleaner that way — match existing idioms). Use the existing helper signatures listed in the `<interfaces>` block in `<context>` above; only add a new helper if a category param genuinely can't be threaded through `insertPendingPurchaseWithEventDate`.

If a new helper is needed, the cleanest shape is:

```go
// insertPendingPurchaseFull inserts an unconfirmed/undiscarded pending row
// with full control over reason, mercury_category, vendor, and bank_total —
// the four fields the narrowed /period-summary contract cares about.
// Pass mercuryCategory == "" for SQL NULL. Pass eventDate == "" for SQL NULL
// (created_at::Chicago::date will be the period filter input).
func insertPendingPurchaseFull(t *testing.T, bankTxID, eventDate, createdAt, reason, mercuryCategory, vendor string, bankTotal float64) string { ... }
```

Use `insertVendor(t, name)` for any case-B by_vendor match cases — the `vendors.name` row must exist before the pending row references it.

**The 10 cases (verbatim from HANDOFF.md §4 — assert both `pending_review_ids`/`pending_review_details` and `cogs_excl_tax`/`by_vendor` on every case unless noted):**

The cogsAllowlist sentinel used by tests should be a single category string the existing tests use — e.g. `[]string{"Food, Beverage & Groceries"}`. Use `callHandlerWithAllowlist` to inject it.

| # | Case | mercury_category | reason | Expected: blocks? | Expected: in COGS? |
|---|------|------------------|--------|-------------------|--------------------|
| 1 | **Case A** food + no attachment | "Food, Beverage & Groceries" | "no_attachment_on_bank_tx" | **YES — only blocker** | no |
| 2 | **Case B** food + parse-failed | "Food, Beverage & Groceries" | "Receipt could not be parsed automatically" | no | **YES — at bank_total** |
| 3 | **Case C** non-food + no attachment | "Software, SaaS & Subscriptions" (or any non-allowlist value) | "no_attachment_on_bank_tx" | no | no |
| 4 | **Case D** non-food + parse-failed | non-food | "Receipt could not be parsed automatically" | no | no |
| 5 | **NULL category** + no attachment | "" (NULL) | "no_attachment_on_bank_tx" | no — NULL fails `= ANY` | no |
| 6 | **Date filter still applies** — case A row *outside* the period | "Food, Beverage & Groceries" | "no_attachment_on_bank_tx" | NOT returned | NOT in COGS |
| 7 | **pending_review_details parity** — single case A row in period | (same as #1) | (same as #1) | `pending_review_ids` length 1; `pending_review_details` length 1; both arrays index-match the same id |
| 8 | **Case B by_vendor MATCH** — insert one confirmed RD event ($100) + one case-B Save A Lot pending row ($19) where vendor field matches existing `vendors.name` case-insensitively | (same as #2) | (same as #2) | `cogs_excl_tax == 119`; `by_vendor` contains BOTH an RD row at $100 and a Save A Lot row at $19, **each with a real (non-empty) vendor_id** |
| 9 | **Case B by_vendor UNMATCHED** — case-B pending with `vendor = 'Brand New Vendor not in vendors table'` | (case B) | (case B) | `by_vendor` has a row with `vendor_id == ""` and `vendor_name == "Brand New Vendor not in vendors table"` |
| 10 | **Vendor name fuzz** — pre-insert `vendors.name = 'Save A Lot'`. Case-B pending with `vendor = 'save a lot '` (lowercase, trailing space) | (case B) | (case B) | Joins via `LOWER(TRIM())`; pending merges into the existing Save A Lot row in `by_vendor` (no duplicate row with `vendor_id == ""`) |

Notes:
- All in-period cases use createdAt around `2026-05-27 10:00:00-05:00` and event_date `2026-05-27`; period filter `from=2026-05-25 to=2026-05-31` (matches existing pattern in this file).
- Case 6's out-of-period row uses createdAt `2026-04-15` (or any date strictly before the period).
- For case 8, insert RD via `insertEventAndLineWithCategory(...)` so the confirmed event is in the allowlist and contributes its $100. Then insert the case-B Save A Lot pending. Assert `cogs_excl_tax == 119` exactly (use `assert.InDelta` if the existing tests do, otherwise direct equality).
- Reset fixtures between sub-tests via `resetFixtures(t)` (existing helper, already used elsewhere in this file).
- Use sub-test names that map cleanly to the table: `t.Run("case_a_food_no_attachment_blocks", ...)`, `t.Run("case_b_food_parse_failed_rolls_into_cogs", ...)`, etc.

Do NOT modify existing tests other than these additions. The `pending_review_details` tests shipped in 260606-hvy stay green — they all use "Receipt could not be parsed automatically" or no-reason rows that today land in `pending_review_ids` and after this change land outside `pending_review_ids`. **Audit the existing tests for category-implicit assumptions and add `mercury_category = "Food, Beverage & Groceries"` to any existing pending insert that the test expects to surface in `pending_review_ids`.** This is the only collateral edit; flag any test you change in the SUMMARY.
  </action>
  <verify>
    <automated>cd backend &amp;&amp; go test ./internal/inventory/ -run TestPeriodSummary -count=1 -v</automated>
  </verify>
  <done>All 10 new sub-tests pass. All pre-existing `TestPeriodSummary` sub-tests still pass. `go test ./...` is green for the inventory package.</done>
</task>

</tasks>

<verification>
End-to-end sanity (executor runs after all 4 tasks land):

```bash
cd backend && task migrate-up && go test ./internal/inventory/ ./internal/receipt/ -count=1
```

Then the curl from HANDOFF.md §Verification against a dev DB with seeded RD/Save A Lot/Amazon pending rows in the period:

```bash
curl -s -H "Authorization: Bearer $HQ_INVENTORY_SERVICE_TOKEN" \
  "$HQ_BASE_URL/api/v1/inventory/period-summary?from=2026-05-25&to=2026-05-31" \
  | python3 -m json.tool | grep -E '(ready|pending_review_ids|pending_review_details|cogs_excl_tax|by_vendor)'
```

Expected: `ready: false`, `pending_review_ids` has ONLY the RD id, `cogs_excl_tax` includes Save A Lot's $19.28, `by_vendor` lists Save A Lot.
</verification>

<success_criteria>
- 0066 migration applies + rolls back cleanly. `pending_purchases.mercury_category` exists, nullable.
- Worker fills `mercury_category` on new pending inserts and backfills existing in-window rows on the next poll.
- `/period-summary`:
  - `completeness.ready=false` ONLY when ∃ blocking pending row (allowlist category + no_attachment_on_bank_tx) OR ∃ unlinked line item.
  - `pending_review_ids` + `pending_review_details` include ONLY blocking pending rows.
  - `cogs_excl_tax` = confirmed line-item sum + ABS(bank_total) of non-blocking eligible pending.
  - `by_vendor` merges confirmed events with non-blocking eligible pending; matched pending vendor names collapse into the existing vendor row; unmatched stay separate with `vendor_id=""`.
- All 10 new test cases pass. Pre-existing `TestPeriodSummary` sub-tests still pass.
- No changes to sales-processor. No changes to the response struct shape. No feature flag.
</success_criteria>

<output>
After completion, create `.planning/quick/260606-jvs-narrow-completeness-ready-on-period-summ/260606-jvs-SUMMARY.md`.
</output>
