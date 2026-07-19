---
id: 260606-9y0
slug: tracked-bank-tx-ids-on-period-summary
date: 2026-06-06
status: verified
commit: f730485
---

# Verification

## Goal recap

Expose a `tracked_bank_tx_ids` array on `GET /period-summary` so
sales-processor can detect Mercury transactions the receipt worker
hasn't ingested yet.

## Goal-backward checks

### 1. Response shape matches spec

`PeriodSummary` struct in `backend/internal/inventory/types.go:163`
contains:

```go
TrackedBankTxIDs []string `json:"tracked_bank_tx_ids"`
```

…between `ByVendor` and `Completeness`, with a doc comment explaining
the cross-state semantics and pointing at the consumer doc.

### 2. Field is populated and non-nil

`backend/internal/inventory/handler.go` step 3b declares
`trackedTxIDs := []string{}` (not `var trackedTxIDs []string`) so an
empty period marshals to `[]` not `null`. The response struct literal
wires `TrackedBankTxIDs: trackedTxIDs`.

Test `tracked_bank_tx_ids: empty period renders [] not null` confirms
JSON contains `"tracked_bank_tx_ids":[]`.

### 3. All states are included, with dedupe and ordering

Test `tracked_bank_tx_ids: all states present, deduped, sorted`
constructs:

- **tx-A**: `purchase_events` only (confirmed, no pending)
- **tx-B**: `pending_purchases` only, `confirmed_at`/`discarded_at` NULL
- **tx-C**: `pending_purchases` only, `discarded_at` set
- **tx-D**: BOTH `pending_purchases` (with `confirmed_at` set) AND a
  matching `purchase_events` row — the confirm-path case
- **tx-Z-out-of-period**: `pending_purchases` with both dates outside
  the window — must NOT appear

Asserts `TrackedBankTxIDs == ["tx-A","tx-B","tx-C","tx-D"]` exactly.
This proves:
- discarded rows are listed (they are "touched");
- the confirm-path tx D is deduped to one entry by `UNION`;
- out-of-period rows are excluded;
- ordering is ASC lexicographic by bank_tx_id.

### 4. Period membership for pending half uses the same COALESCE as the gate

Test `tracked_bank_tx_ids: period boundary uses COALESCE(event_date,
created_at::Chicago)` inserts four pending rows covering the
(event_date present/null) × (created_at in/out) quadrant and asserts
only the rows whose effective date lands in the window appear:

| bank_tx_id | event_date | created_at | expected |
|---|---|---|---|
| tx-in-by-event-date  | 2026-05-29 (in)  | 2026-06-02 (out) | present |
| tx-out-by-event-date | 2026-05-20 (out) | 2026-05-27 (in)  | absent  |
| tx-in-by-created-at  | NULL             | 2026-05-27 (in)  | present |
| tx-out-by-created-at | NULL             | 2026-06-05 (out) | absent  |

This mirrors the existing pending-review-gate axis tests added in
260606-0gh, confirming the UNION's pending half agrees with the gate
on what "belongs to the period."

### 5. Build + test results

```
cd backend && go build ./...      # clean
cd backend && go vet ./internal/inventory/...  # clean
DB_TEST_URL=... go test ./internal/inventory/ -run TestPeriodSummary -v -count=1
```

22 subtests run. 21 PASS, including all three new ones:

- `tracked_bank_tx_ids:_empty_period_renders_[]_not_null` — PASS (0.47s)
- `tracked_bank_tx_ids:_all_states_present,_deduped,_sorted` — PASS (0.75s)
- `tracked_bank_tx_ids:_period_boundary_uses_COALESCE(...)` — PASS (0.46s)

The one failure (`end-to-end_empty-items_confirm_increments_cogs`) is a
pre-existing remote-DB schema drift (`users.display_name` column
missing) reproducible on plain `dev` via `git stash` — not introduced
by this phase.

## Sign-off

- [x] Field is additive — no breaking shape change for existing
  consumers (sales-processor decodes silently until it reads the field).
- [x] No schema migration required — both `bank_tx_id` columns already
  exist and are populated by the receipt worker.
- [x] No feature flag required — sales-processor consumer treats the
  field as optional during rollout.
- [x] Spec parity: doc comment, SQL shape (UNION + COALESCE), scan
  pattern, response wiring, and test coverage all match
  `cogs-hq-handoff` companion doc.
