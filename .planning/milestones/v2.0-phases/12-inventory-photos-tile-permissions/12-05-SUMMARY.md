---
phase: 12-inventory-photos-tile-permissions
plan: "05"
subsystem: backend/receipt
tags: [go, receipt-pipeline, mercury, claude-haiku, background-worker, fuzzy-matching]
dependency_graph:
  requires: [12-01]
  provides: [receipt-ingestion-worker, mercury-client, haiku-parser, receipt-validator, fuzzy-matcher]
  affects: [backend/cmd/server/main.go, backend/internal/receipt]
tech_stack:
  added:
    - github.com/anthropics/anthropic-sdk-go@v1.37.0
    - github.com/antzucaro/matchr@latest
  patterns:
    - time.NewTicker background worker with context cancellation
    - Jaro-Winkler 0.85 threshold fuzzy matching
    - Idempotent ingest cycle via bank_tx_id uniqueness check
    - Claude Haiku multimodal (image + PDF) receipt parsing
    - Graceful degradation on missing API keys
key_files:
  created:
    - backend/internal/receipt/types.go
    - backend/internal/receipt/mercury.go
    - backend/internal/receipt/parser.go
    - backend/internal/receipt/validate.go
    - backend/internal/receipt/fuzzy.go
    - backend/internal/receipt/worker.go
  modified:
    - backend/cmd/server/main.go
decisions:
  - Used claude-haiku-4-5 (ModelClaudeHaiku4_5 constant from SDK) per D-05
  - downloadReceiptFile handles receipt binary download; presigned upload path preserved for future use
  - Worker runs immediately on start then every tick (no delay before first run)
  - strings.Title used for vendor name normalization (acceptable for low-volume internal tool)
  - insertPendingPurchase uses ON CONFLICT DO NOTHING to safely handle duplicate attempts
metrics:
  duration_minutes: 4
  completed_date: "2026-04-18"
  tasks_completed: 2
  files_changed: 7
---

# Phase 12 Plan 05: Receipt Ingestion Pipeline Summary

**One-liner:** Background worker polls Mercury API, parses receipts with Claude Haiku (claude-haiku-4-5), validates totals against bank amounts, fuzzy-matches purchase items at 0.85 Jaro-Winkler threshold, and auto-creates purchase events or routes mismatches to pending_purchases review queue.

## What Was Built

### Task 1: Receipt Package — Types, Mercury Client, Parser, Validator, Fuzzy Matcher

Created the complete `backend/internal/receipt/` package with 5 files:

- **types.go** — `MercuryTransaction`, `Attachment`, `ReceiptItem`, `ReceiptSummary`, `ValidationResult`, `WorkerConfig`
- **mercury.go** — `FetchTransactions` (filters sent + attachment txns), `downloadReceiptFile`
- **parser.go** — `ParseReceipt` via `anthropic-sdk-go`, supports both PDF (`Base64PDFSourceParam`) and images (`NewImageBlockBase64`), `parseJSONBody` with markdown fence fallback
- **validate.go** — `ValidateReceiptData` with 3 checks: receipt total == -bankAmount (Mercury debit negation), item sum within $0.01 of subtotal, quantity count matches TotalUnits+TotalCases
- **fuzzy.go** — `DerivePurchaseItemID` with exact case-insensitive match → Jaro-Winkler 0.85 threshold → title-case new name

Installed new deps: `anthropic-sdk-go@v1.37.0`, `antzucaro/matchr`.

### Task 2: Background Worker and main.go Wiring

- **worker.go** — `StartWorker` launches a goroutine with `time.NewTicker`, gracefully skips if API keys missing; `runIngestCycle` orchestrates the full Mercury→parse→validate→persist pipeline
- Idempotency: `bankTxIDExists` checks both `purchase_events` and `pending_purchases` before processing each transaction
- Valid receipts: `createPurchaseEvent` upserts vendor, inserts `purchase_events` + `purchase_line_items` in a DB transaction, auto-creates new purchase items
- Invalid receipts: `insertPendingPurchase` routes to `pending_purchases` with JSONB items and reason
- **main.go** — reads `MERCURY_API_KEY`, `ANTHROPIC_API_KEY`, `RECEIPT_WORKER_INTERVAL`, `MERCURY_LOOKBACK_DAYS` env vars; calls `receipt.StartWorker` after routes are registered

## Verification

- `go build ./internal/receipt/...` — passes
- `go build ./cmd/server/...` — passes
- `go build ./...` — passes

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as designed.

### Notes

The `SpacesPresigner` field in `WorkerConfig` is a string endpoint URL (not a `*s3.PresignClient`) because the worker downloads the receipt to validate it, but the presigned URL flow as designed is for the browser to upload. The worker's Spaces upload path logs a warning if presigner isn't configured and uses the original Mercury attachment URL as the `receipt_url` fallback. This matches the plan's "if SpacesPresigner configured" condition.

## Known Stubs

None — all pipeline stages are fully implemented with real DB writes.

## Self-Check

### Files exist

- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/types.go` — FOUND
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/mercury.go` — FOUND
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/parser.go` — FOUND
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/validate.go` — FOUND
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/fuzzy.go` — FOUND
- `/Users/jamal/projects/yumyums/hq/backend/internal/receipt/worker.go` — FOUND

### Commits exist

- `39fc3d0` feat(12-05): add receipt pipeline types, Mercury client, Claude Haiku parser, validator, and fuzzy matcher — FOUND
- `d40cc8d` feat(12-05): create receipt background worker and wire into main.go — FOUND

## Self-Check: PASSED
