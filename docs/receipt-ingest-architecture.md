# Receipt ingest architecture

**Last updated:** 2026-06-25
**Status:** Active. All work in commits `3f0240b..97c9a4d` on `dev` (unpushed).

## Problem statement

The receipt ingest pipeline pulls card transactions from Mercury, downloads attached receipts, and uses Claude to parse them into structured line items + summaries that auto-create `purchase_events` (no human review needed) or fall back to a `pending_purchases` queue for operator verification.

Three pain points surfaced during the 2026-06-25 work session:

1. **Multi-attachment cases** (a purchase receipt + a refund credit memo on the same bank transaction) were unsupported — only the first attachment ever reached Claude, validation failed, the row was stuck.
2. **What's stored vs. what Claude can reliably produce** drifted: validation required Claude to maintain self-consistency between `summary.total` and the sum of line items, which Claude can't do reliably on multi-image inputs.
3. **Stuck pending rows** from older code paths couldn't be re-processed without manual `Retry parse` per row OR a full server restart with carefully-staged DB resets.

## Data flow

```
Mercury API → FetchTransactions (14-day lookback, paginated)
            ↓
            for each tx with attachments:
              → Download all attachments to memory
              → Upload each to object storage — Backblaze B2 (receipts/{tx_id}/{idx}{ext})
              → Build []FileBlob from raw bytes (one entry per attachment)
              → ParseReceipt (Claude Haiku, 4096 maxTokens)
                  ↳ on JSON-unmarshal failure: ParseReceiptWithSonnet (fallback)
              → ValidateReceiptData
                  ↳ derive total from items + tax, compare to bank
                  ↳ on Receipt-derived-total mismatch: ParseReceiptWithFeedback
                     (Sonnet with reconciliation prompt that embeds bank amount)
              → attemptScore: keep BEST attempt across the retry loop (max 2)
              → Sanity gate: items non-empty AND items_sum > $0.50
              → enrichItemsWithMatches (token overlap → AI fallback for misses)
              → if validate passed AND sanity passed:
                    createPurchaseEvent (vendor upsert + event + line items;
                      auto-creates new purchase_items as needed)
                else:
                    routePending (insert/update pending_purchases with
                      full retry trace persisted in parse_error)

Admin reprocess (new, 2026-06-25):
  POST /api/v1/inventory/purchases/reprocess-all
    → SELECT pending rows + their stored receipt URLs
    → For each row: synthesize a fake MercuryTransaction from stored data
    → Run through processSingleTx(reprocess=true) — same pipeline as above
    → Mercury is NOT contacted; attachments come from object storage (Backblaze B2)

Admin URL recovery (B-172, 2026-08-24):
  POST /api/v1/inventory/purchases/recover-receipts   {"dry_run":bool, "limit":int}
    → Find purchase_events + undiscarded pending_purchases rows whose
      receipt_url / receipt_urls are OFF the current {STORAGE_ENDPOINT}/{bucket}/
      prefix (dead DO Spaces URLs, expiring Mercury-fallback URLs, any past host)
    → dry_run: report counts + tx ids, touch nothing
    → real run: re-fetch attachments from Mercury by bank_tx_id (windowed by the
      rows' earliest event_date), re-upload to receipts/{tx_id}/{i}{ext},
      rewrite both tables' URL columns; per-tx atomic, re-run-safe
    → single-flight via receipt_sync_runs (409); terminal counts reuse its
      columns: processed=examined, auto_created=recovered,
      pending_review=missing-at-Mercury, cached=failed
```

## Component responsibilities

### Parser (`backend/internal/receipt/parser.go`)

- Single entry point: `ParseReceipt(blobs []FileBlob)` — accepts 1..N file blobs and sends them all to Claude in one `messages.New` call.
- Haiku at 4096 maxTokens (bumped from 2048; 40-item receipts truncated).
- Sonnet at 4096 as fallback for JSON-decode failures.
- `ParseReceiptWithFeedback` is a reconciliation pass with the validate failure message embedded in the prompt — used by the worker's retry loop only on Check-1 (derived-total) mismatches.
- `StripJSONFence` extracted as a shared helper used by both `parseJSONBody` (receipt parsing) and `MatchItemsWithAI` (catalog matching).
- Prompt is highly prescriptive — explicit unit-price vs extended-total examples for weight-priced items, multi-image netting instructions, "no markdown code fences" reminder.

### Validator (`backend/internal/receipt/validate.go`)

Single check only:

```
derivedTotal = sum(item.price × item.quantity for item in items) + summary.tax
|derivedTotal - (-bankAmount)| ≤ 0.01  →  valid
```

History: previously also enforced `summary.total == -bankAmount` (Check 1, old) and `sum(items) == summary.total - tax` (Check 2) and `sum(quantities) == total_units + total_cases` (Check 3). All three replaced by the derived-total check. Reasoning: Claude can self-consistently produce items, but maintaining a separate `summary.total` field that exactly matches items is unreliable on multi-image inputs. Trust items; derive total in Go.

### Worker (`backend/internal/receipt/worker.go`)

- `runIngestCycle` is the top-level entry. Iterates transactions returned by `FetchTransactions`.
- `processSingleTx(tx, reprocess bool)` is the per-tx core; both `runIngestCycle` and the reprocess flow call it.
- Retry loop:
  - Max 2 attempts.
  - Attempt 1: `parseReceipt` (Haiku, Sonnet on JSON-error fallback).
  - Attempt 2: `parseReceiptWithFeedback` (Sonnet with reconciliation prompt).
  - Each attempt scored by closeness of derived total to bank amount (lower = better; empty items penalized hard so non-empty always wins).
  - Best attempt persisted, not last. Prevents the failure mode where attempt 2 regresses by emptying the items array entirely.
- Sanity gate after the loop: skip auto-create if items are empty or items_sum is trivial.
- Catalog enrichment runs after the gate: token overlap (threshold 0.5) → AI fallback for misses → populated `purchase_item_id` per item so the FE pre-fills the dropdowns.
- Dup-key cleanup: when `createPurchaseEvent` hits a `bank_tx_id` unique-constraint violation (the event already exists from a prior sync), the pending row is deleted and the function returns success rather than routing the duplicate attempt to review.

### Reprocess (`backend/internal/inventory/reprocess_pending.go`)

New endpoint `POST /api/v1/inventory/purchases/reprocess-all`:

- Selects all still-pending rows that have a stored attachment URL (legacy `receipt_url` OR new `receipt_urls`).
- For each row, builds a `PendingRowForReprocess` struct from stored fields (`bank_tx_id`, `bank_total`, `vendor`, `event_date`, URL list).
- `receipt.BatchReprocessFromSpaces` synthesizes a fake `MercuryTransaction` from that struct and calls `processSingleTx(reprocess=true)`.
- Mercury is **not contacted at all**. Attachments come from object storage (Backblaze B2).
- Single-flight against the `receipt_sync_runs` table — concurrent reprocess returns 409 conflict.
- Why Spaces, not Mercury: Mercury's individual-tx endpoint (`/api/v1/transactions/{id}`) doesn't exist; the list endpoint's `offset` pagination doesn't behave the way an offset+limit API normally does (paginated past 50000 offset without ever returning a short page). The Spaces approach sidesteps that uncertainty and is cheaper anyway.

### AI item matcher (`backend/internal/receipt/ai_matcher.go`)

- Called after token-overlap matching fails for some items.
- Batches all unmatched names + the full catalog into one Haiku call.
- Filters Claude's response to high-confidence matches only.
- Degrades gracefully on JSON parse failure or natural-language responses (logs and returns `(nil, nil)`) — never propagates an error that would abort the pipeline.
- Skip-on-empty: zero unmatched names → zero API calls.

### Frontend (`inventory.html`)

- Multi-receipt carousel in the "View Original Receipt" overlay (prev/next + "1 / N" counter + open-in-new-tab href tracks current URL).
- iOS PDF readability: `color-scheme: light` on the iframe (CSS + inline style) so iOS Safari/PWA's native PDF viewer renders in light mode regardless of system dark-mode preference.
- Item dropdown pre-fill: each line item in the JSONB carries `purchase_item_id` when matched; the FE renders matched items with the catalog name pre-selected, unmatched with the "Tap to select item…" placeholder.
- Admin "Reprocess all pending" button wires to the new endpoint. Confirmation dialog gates the destructive action.

## Key decisions and rationale

| Decision | Why | Trade-off |
|---|---|---|
| Multi-image: send all attachments in one Claude call | Claude needs to see purchase + refund together to net them | Single point of failure if Claude misreads either image |
| Validator derives total from items, ignores `summary.total` | Claude can produce correct items but not maintain a consistent separate total on multi-image; making consistency the parser's job thrashes on retry | If items are wrong AND Claude's wrong total agrees with the wrong items, we don't catch it |
| Tolerance = $0.01 on derived total vs bank | User explicitly chose strict | Real receipts have penny rounding; some legit cases route to review unnecessarily |
| Retry loop max 2 + best-attempt tracking | Attempt 2 can regress (Claude empties items array under feedback pressure) | One bad attempt can taint the score if the "better" attempt is still bad in different ways |
| Token overlap matching @ 0.5 | Receipt SKUs ("BF CHUCK TENDERS RW") and catalog names ("Chicken Tenders") share key words but not full strings | False positives — "BEEF CHUCK" might match "Chicken" (both meat words). Operator catches in review |
| AI matcher as token-matcher fallback (not primary) | Token matching is cheap and deterministic; AI is expensive and non-deterministic | Two layers to debug |
| Sanity gate (items > 0, items_sum > $0.50, match_rate ≥ 30%) | Catches the "Claude regressed to empty" and "Claude hallucinated names" failure modes | Match-rate gate can over-fire on legitimately low-overlap receipts; tunable |
| Reprocess pulls from Spaces, not Mercury | Mercury's per-tx endpoint doesn't exist; list-endpoint pagination is broken or different from what we assumed; attachments may have expired anyway; Spaces is always available | Rows whose original attachment was never uploaded to Spaces (`no_attachment_on_bank_tx` cases) can't be reprocessed |
| Mercury pagination uses raw page size, not filtered | Per-page filtering (`status="sent"` + supported kinds) shrinks each page below `limit`, which would short-circuit pagination immediately | Adds one extra count to track in the page-fetch function |
| Dup-key violation → delete pending and return success | The event already exists; the residual pending is by definition stale | None — atomic via the existing pgx transaction |
| Logs tee'd to `/tmp/hq-server.log` from `task dev:*` | Operator/Claude can read worker output without scrolling a terminal | `tee` truncates on each restart — older logs are lost |

## Storage schema

### `pending_purchases` (needs-review queue)

```
id                UUID PK
bank_tx_id        TEXT (Mercury transaction ID)
bank_total        NUMERIC (signed; negative for debit)
vendor            TEXT (Claude-parsed; falls back to Mercury bankDescription)
event_date        DATE (Claude-parsed; falls back to Mercury createdAt)
tax               NUMERIC (Claude-reported, signed)
total             NUMERIC (Claude-reported; derived total overwrites on confirm)
total_units       NUMERIC
total_cases       NUMERIC
items             JSONB (array of {name, quantity, price, is_case, purchase_item_id?})
reason            TEXT (validation failure message OR "Receipt could not be parsed automatically")
parse_error       TEXT (full retry trace: "attempt 1: ...; attempt 2: ...")
receipt_url       TEXT (legacy single URL — still populated as receipt_urls[0])
receipt_urls      JSONB (array of all attachment URLs, added in migration 0070)
mercury_category  TEXT
created_at        TIMESTAMPTZ
confirmed_at      TIMESTAMPTZ (NULL until operator confirms)
discarded_at      TIMESTAMPTZ (NULL until operator discards)
```

Upgrade gate (`classifyExistingTx` in worker.go) matches rows where:
- `reason = 'no_attachment_on_bank_tx'` AND Mercury now has attachments → upgrade
- `reason = 'Receipt could not be parsed automatically'` AND `parse_error IS NULL` AND `items = []` AND tx has attachments → upgrade

Confirmed/discarded rows are immutable from the worker's perspective.

### `purchase_events` (auto-created)

Same shape minus the pending-only fields. `bank_tx_id` is UNIQUE — this is the duplicate-key constraint the worker handles via `createPurchaseEvent`'s 23505 path.

### `receipt_sync_runs` (ingest tracking)

```
id              BIGSERIAL PK
started_at      TIMESTAMPTZ
finished_at     TIMESTAMPTZ
status          TEXT CHECK (running|done|failed)
processed       INT
auto_created    INT
pending_review  INT
cached          INT
error           TEXT
triggered_by    TEXT ('manual' | 'reprocess_all' | future values)
```

Unique index `receipt_sync_runs_single_running` on `(1) WHERE status = 'running'` enforces single-flight.

## Known issues and future work

1. **Mercury pagination scheme is uncertain.** `FetchTransactions` paginates with `offset`+`limit` and tests pass, but live data over a 1-year window paginated past offset=50000 without ever returning a short page. The 14-day ingest tick fits in one page so the issue doesn't bite there. If you ever need wide-window Mercury queries (analytics? historical backfill?), investigate Mercury's actual pagination semantics — it may be cursor-based, account-scoped, or differently bounded.

2. **Catalog match false positives.** Token overlap at 0.5 will map "BEEF CHUCK" to "Chicken Tenders" when "Chicken" and "Beef" don't overlap but the second word does. This is fine in the current model (operator verifies in review), but a stricter threshold + smarter tokenizer (e.g. word-stem matching, embeddings) would reduce noise.

3. **Match-rate sanity gate may be over-aggressive.** A 7-item tx with 2 catalog matches gets blocked at the 30% gate even when validation passed. Lower the threshold (10%? 15%?) or skip the gate entirely when validation passed cleanly.

4. **Older pending rows with NULL `receipt_url`.** ~7 rows can never be reprocessed because their attachment was never uploaded to Spaces. Operator must hand-edit. A real "no attachment" case has no fix; an upload-failed case could in principle be re-attempted if Mercury still has the attachment, but per known issue #1 we can't fetch individual txs reliably.

5. **SW caching causes stale FE state.** When the backend updates a row, the FE's cached row data doesn't refresh without a hard refresh. Documented in the hq `feedback_sw_cache_bump` memory. Real fix would be one of: SSE push, polling on tab focus, or unregistering the SW on backend-version mismatch.

6. **Background worker + on-demand sync can race.** Both invoke `runIngestCycle` directly; the single-flight is only on `receipt_sync_runs` (which the background worker doesn't write). Observed exactly once today — caused two duplicate-key violations that the new 23505 handler swallowed cleanly. Worth tightening eventually (e.g. an in-process mutex or extending single-flight to the background tick).

7. **Receipt-sync triggers haven't been audited end-to-end.** Adding a new `triggered_by` value (like `reprocess_all`) requires checking the `receipt_sync_runs.triggered_by` column's CHECK constraint (currently it has none — plain `text NOT NULL DEFAULT 'manual'`). If a constraint gets added later, this is a footgun.

## Commit timeline (rough order)

```
3f0240b fix(receipt): parse all attachments in a single Claude prompt
c5db8b1 fix(inventory): add color-scheme:light to PDF iframe so iOS doesn't dim
6874243 feat(db): migration 0070 — receipt_urls JSONB column
9eac675 feat(receipt): upload all attachments to Spaces and persist receipt_urls array
975ef46 feat(inventory): multi-receipt carousel in View Original Receipt overlay
0609afb test(receipt,inventory): assert receipt_urls array on multi-attachment paths
ca74697 chore(taskfile): add migrate tasks mirroring dev/lan/tailscale matrix
0a94830 chore(taskfile): tee dev server output to /tmp/hq-server.log
c43ff15 feat(receipt): goal-driven retry loop with feedback prompt + Playwright tests
fb310ec feat(receipt): tighten unit-price prompt and broaden retry to line-item-sum
01888f0 fix(retry-parse): clear items+reason+parse_error in one branch
911fe43 feat(receipt): broaden retry to all validation failures + total_units/cases
5e56eb3 feat(receipt): unit-price prompt + broaden retry guard to Line item sum mismatches
d54d82a feat(receipt): derive total from items in validate.go
b2588f8 feat(receipt): drop Check 3, track best attempt across retries, sanity gate
b10da0a feat(receipt): fuzzy-match items to catalog before persisting to pending rows
7bc39df feat(receipt): token-overlap matcher + Claude Haiku AI fallback for catalog lookup
91a6127 feat(inventory): admin Reprocess all pending receipts endpoint + button
9e12960 feat(receipt): per-row Mercury refetch + AI matcher hardening + dup-key cleanup
422cab9 fix(receipt): use Mercury list-with-date-range for reprocess (no per-tx endpoint)
fb8af8d feat(mercury): paginate FetchTransactions so wide reprocess windows work
653d063 fix(mercury): paginate on raw page size, not filtered, so filtering doesn't short-circuit
97c9a4d refactor(receipt): reprocess from Spaces storage instead of Mercury (no per-tx fetch API exists)
```

The arc of the session, in shorthand: fix the parser to handle multi-image → fix the storage to hold multi-URL → fix the FE to render multi-URL → fix the validator's reliance on Claude's self-consistency → fix the prompt's vagueness about unit prices → fix the retry guard's narrow scope → fix the retry-loop's regression-on-second-attempt → fix the post-parse catalog match → fix the FE pre-fill from catalog → fix the admin reprocess endpoint → fix the reprocess's reliance on a Mercury endpoint that doesn't exist → land on Spaces-based reprocess as the durable answer.
