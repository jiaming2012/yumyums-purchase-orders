package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
)

// Test seams. Production callers MUST go through these package-level vars so
// tests can inject fakes without changing exported function signatures. See
// worker_test.go — each test saves the original, swaps in a stub, and restores
// in a t.Cleanup callback.
//
// Phase 260607-co0: parseReceipt + fetchTransactions + downloadReceiptFileFn
// are introduced so the upgrade-path tests (no_attachment → re-sync with
// receipt attached) can drive runIngestCycle end-to-end without hitting
// Mercury / Anthropic / the receipt CDN.
var (
	fetchTransactions        = FetchTransactions
	parseReceipt             = ParseReceipt
	parseReceiptWithSonnet   = ParseReceiptWithSonnet
	parseReceiptWithFeedback = ParseReceiptWithFeedback
	downloadReceiptFileFn    = downloadReceiptFile
	uploadReceiptSlotFn      = uploadReceiptSlot
)

// PendingRowForReprocess is the minimal data needed from a pending_purchases
// row to drive a reprocess: bank tx ID for upsert keying, bank amount for
// validation, vendor for fallback display, event_date for the synthesized
// MercuryTransaction.CreatedAt, and the receipt URL list to download.
type PendingRowForReprocess struct {
	BankTxID    string
	BankTotal   float64  // negative for debit, matches Mercury convention
	Vendor      string
	EventDate   string   // YYYY-MM-DD
	ReceiptURLs []string // may be one URL (legacy) or many (multi-attachment)
}

// ReprocessFromSpaces synthesizes a MercuryTransaction from the pending row's
// stored data, downloads each receipt URL from the Spaces bucket (or wherever
// the URL points), and runs the standard parse/validate/persist pipeline. No
// Mercury API calls are made. Returns one of "auto_created", "pending_review",
// "errored", "no_attachments".
func ReprocessFromSpaces(ctx context.Context, cfg WorkerConfig, row PendingRowForReprocess) (string, error) {
	if len(row.ReceiptURLs) == 0 {
		slog.Info(fmt.Sprintf("receipt worker: tx %s reprocess skipped — no receipt URLs stored", row.BankTxID))
		return "no_attachments", nil
	}

	// Synthesize a MercuryTransaction from row data.
	syntheticTx := MercuryTransaction{
		ID:              row.BankTxID,
		Amount:          row.BankTotal,
		BankDescription: row.Vendor,
		Status:          mercuryStatusSent,    // implied: already in pending_purchases
		Kind:            mercuryKindDebitCard, // best-effort; not used by parse logic
		CreatedAt:       row.EventDate,
		Attachments:     make([]Attachment, len(row.ReceiptURLs)),
	}
	for i, url := range row.ReceiptURLs {
		syntheticTx.Attachments[i] = Attachment{
			URL:      url,
			FileName: fmt.Sprintf("attachment_%d", i),
		}
	}

	return processSingleTx(ctx, cfg, syntheticTx, true)
}

// BatchReprocessFromSpaces is the entry point the reprocess handler calls.
// It runs ReprocessFromSpaces for each row and aggregates the results into a
// map of bank_tx_id -> status. Errors per row are logged and counted as
// "errored" without aborting the batch.
func BatchReprocessFromSpaces(ctx context.Context, cfg WorkerConfig, rows []PendingRowForReprocess) (map[string]string, error) {
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		status, err := ReprocessFromSpaces(ctx, cfg, row)
		if err != nil {
			slog.Info(fmt.Sprintf("receipt worker: tx %s reprocess error: %v", row.BankTxID, err))
			out[row.BankTxID] = "errored"
			continue
		}
		out[row.BankTxID] = status
	}
	return out, nil
}

// StartWorker launches a background goroutine that polls Mercury for new
// transactions on the configured interval. If either API key is missing the
// worker logs a warning and returns immediately (graceful skip).
func StartWorker(ctx context.Context, cfg WorkerConfig) {
	if cfg.MercuryAPIKey == "" || cfg.AnthropicAPIKey == "" {
		slog.Warn("receipt worker: skipping — missing API keys (MERCURY_API_KEY or ANTHROPIC_API_KEY not set)")
		return
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	slog.Info("receipt worker: starting", "interval", interval, "lookback_days", cfg.LookbackDays)

	go func() {
		// Run immediately on start, then on each tick
		if _, err := runIngestCycle(ctx, cfg); err != nil {
			slog.Error("receipt worker: ingest cycle error", "error", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("receipt worker: shutting down")
				return
			case <-ticker.C:
				if _, err := runIngestCycle(ctx, cfg); err != nil {
					slog.Error("receipt worker: ingest cycle error", "error", err)
				}
			}
		}
	}()
}

// IngestResult captures the counts produced by one ingest cycle.
// Returned by runIngestCycle so callers (the on-demand sync handler) can
// persist these counts to receipt_sync_runs alongside the terminal status.
type IngestResult struct {
	Processed     int
	AutoCreated   int
	PendingReview int
	Cached        int
}

// RunIngestCycle runs one Mercury ingest cycle and returns the result counts.
// Used by the on-demand sync endpoint; the background worker calls
// runIngestCycle directly.
func RunIngestCycle(ctx context.Context, cfg WorkerConfig) (IngestResult, error) {
	return runIngestCycle(ctx, cfg)
}

// runIngestCycle executes one full Mercury → parse → validate → persist cycle.
func runIngestCycle(ctx context.Context, cfg WorkerConfig) (IngestResult, error) {
	lookback := cfg.LookbackDays
	if lookback <= 0 {
		lookback = 14
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -lookback)

	txns, err := fetchTransactions(ctx, cfg.MercuryAPIKey, startDate, endDate)
	if err != nil {
		return IngestResult{}, fmt.Errorf("runIngestCycle: FetchTransactions: %w", err)
	}

	if len(txns) == 0 {
		slog.Info("receipt worker: no supported transactions found")
		return IngestResult{}, nil
	}

	var autoCreated, pendingReview, skippedCached int

	for _, tx := range txns {
		// Refresh mercury_category on existing events so values set by the
		// sales-processor classify pipeline (async, weekly or nightly) propagate
		// into HQ without a separate scheduler. Idempotent via IS DISTINCT FROM.
		// Runs for cached AND new transactions (before the `already` short-circuit)
		// so a previously-ingested row can pick up a late classify pass.
		if tx.CategoryData != nil {
			_, refreshErr := cfg.Pool.Exec(ctx,
				`UPDATE purchase_events
				 SET mercury_category = $1
				 WHERE bank_tx_id = $2
				   AND (mercury_category IS DISTINCT FROM $1)`,
				tx.CategoryData.Name, tx.ID)
			if refreshErr != nil {
				slog.Warn("receipt worker: refresh mercury_category failed", "tx_id", tx.ID, "error", refreshErr)
			}
		}

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
				slog.Warn("receipt worker: refresh pending_purchases.mercury_category failed", "tx_id", tx.ID, "error", refreshErr)
			}
		}

		// Backfill pending_purchases.vendor from Mercury's bankDescription
		// for rows that landed before the BankDescription-fallback shipped
		// (260606-hew). Runs alongside mercury_category refresh — before
		// the `already` short-circuit — so cached rows within the lookback
		// window auto-backfill on the next poll. No separate one-shot
		// migration.
		if backfillErr := backfillPendingVendor(ctx, cfg.Pool, tx); backfillErr != nil {
			slog.Warn("receipt worker: backfill vendor failed", "tx_id", tx.ID, "error", backfillErr)
		}

		result, err := processSingleTx(ctx, cfg, tx, false)
		if err != nil {
			slog.Error("receipt worker: processSingleTx failed", "tx_id", tx.ID, "error", err)
			continue
		}
		switch result {
		case "auto_created":
			autoCreated++
		case "pending_review":
			pendingReview++
		case "cached":
			skippedCached++
		}
	}

	slog.Info("receipt worker: cycle complete",
		"processed", len(txns), "auto_created", autoCreated, "pending_review", pendingReview, "cached", skippedCached)
	return IngestResult{
		Processed:     len(txns),
		AutoCreated:   autoCreated,
		PendingReview: pendingReview,
		Cached:        skippedCached,
	}, nil
}

// processSingleTx runs the full ingest pipeline for one Mercury transaction:
// classify → download → upload → parse → validate → persist.
//
// Returns a result enum:
//
//	"auto_created"  – the tx was ingested and a purchase_event row was created
//	"pending_review" – the tx was routed to pending_purchases for human review
//	"cached"        – the tx was already ingested (skipped idempotently)
//	"errored"       – a non-fatal error occurred; caller should log
//
// reprocess=true modifies classify behaviour for the per-row reprocess path:
//   - kind="event": the purchase_event already exists; the pending row is a
//     residual artifact. Delete it and return "cached".
//   - kind="pending": always treat as upgrade-eligible regardless of reason,
//     parse_error, or items (the caller has already confirmed we want a fresh
//     attempt, and the dup-key handler in createPurchaseEvent will clean up any
//     residual pending row if the event INSERT fails with a duplicate key).
func processSingleTx(ctx context.Context, cfg WorkerConfig, tx MercuryTransaction, reprocess bool) (string, error) {
	kind, existingReason, hasParseError, hasItems, err := classifyExistingTx(ctx, cfg.Pool, tx.ID)
	if err != nil {
		slog.Info(fmt.Sprintf("receipt worker: classifyExistingTx tx %s: %v", tx.ID, err))
		return "errored", err
	}

	isUpgrade := false
	switch kind {
	case "event":
		if reprocess {
			// purchase_event already exists — pending row is residual. Clean it up.
			slog.Info(fmt.Sprintf("receipt worker: tx %s already auto-created (reprocess found existing event) — clearing residual pending row", tx.ID))
			if _, delErr := cfg.Pool.Exec(ctx,
				`DELETE FROM pending_purchases WHERE bank_tx_id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
				tx.ID,
			); delErr != nil {
				slog.Info(fmt.Sprintf("receipt worker: delete residual pending for tx %s: %v (continuing)", tx.ID, delErr))
			}
		}
		return "cached", nil
	case "pending":
		if reprocess {
			// Reprocess forces the upgrade path regardless of the existing
			// pending row state — we want a fresh attempt.
			isUpgrade = true
		} else {
			// Normal worker path: only upgrade on the two known upgrade cases.
			noAttachmentUpgrade := existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0
			parseFailedRetry := existingReason == "Receipt could not be parsed automatically" &&
				!hasParseError && !hasItems && len(tx.Attachments) > 0
			if noAttachmentUpgrade || parseFailedRetry {
				isUpgrade = true
			} else {
				return "cached", nil
			}
		}
	case "none":
		// New transaction — fall through to ingest.
	}

	// No-attachment branch: surface every unreceipted card swipe in
	// pending_purchases so the completeness gate can block payroll on
	// unresolved card spend. ON CONFLICT DO NOTHING on bank_tx_id keeps
	// re-polls idempotent.
	//
	// isUpgrade is impossible here (we only set it when len(Attachments)>0),
	// but the !isUpgrade guard is kept defensively so a future refactor
	// can't accidentally re-INSERT over an existing row via this branch.
	if len(tx.Attachments) == 0 {
		if !isUpgrade {
			if routeErr := insertPendingPurchase(
				ctx, cfg.Pool, tx,
				nil,              // items unknown without receipt
				ReceiptSummary{}, // summary unknown without receipt
				"",               // no receiptURL
				nil,              // no receiptURLs
				"no_attachment_on_bank_tx",
				"", // parseError: no parse attempted on no-attachment branch
			); routeErr != nil {
				slog.Info(fmt.Sprintf("receipt worker: insertPendingPurchase (no-attachment) for tx %s: %v", tx.ID, routeErr))
			}
			return "pending_review", nil
		}
		return "cached", nil
	}

	// Download every attachment and collect FileBlobs so all receipts for
	// this transaction are sent to Claude in a single multi-image prompt.
	// This handles the purchase + refund case: both files are seen together,
	// and Claude returns a single combined summary whose Total is the net.
	var blobs []FileBlob
	for _, att := range tx.Attachments {
		fb, ct, dlErr := downloadReceiptFileFn(ctx, att.URL)
		if dlErr != nil {
			slog.Info(fmt.Sprintf("receipt worker: download attachment %s for tx %s: %v (skipping attachment)", att.URL, tx.ID, dlErr))
			continue
		}
		blobs = append(blobs, FileBlob{Bytes: fb, ContentType: ct})
	}
	if len(blobs) == 0 {
		slog.Info(fmt.Sprintf("receipt worker: all attachments failed to download for tx %s — skipping", tx.ID))
		return "errored", nil
	}

	// Upload all attachments to object storage in order. Each gets a
	// per-index key receipts/{tx.ID}/{i}{ext} so they can coexist.
	// receiptURLs collects the final public (or fallback Mercury) URL
	// for each slot. receiptURL (singular) is set to receiptURLs[0]
	// for backward compat with the existing singular-column INSERT calls.
	receiptURLs := make([]string, 0, len(blobs))
	for i, blob := range blobs {
		att := tx.Attachments[i]
		slotURL := att.URL // fallback: original Mercury URL
		if cfg.SpacesPresigner != nil && cfg.SpacesBucket != "" {
			publicURL, upErr := uploadReceiptSlotFn(ctx, cfg, tx.ID, i, blob, att.FileName)
			if upErr != nil {
				slog.Info(fmt.Sprintf("receipt worker: upload slot %d for tx %s: %v (falling back to Mercury URL)", i, tx.ID, upErr))
			} else {
				slotURL = publicURL
			}
		}
		receiptURLs = append(receiptURLs, slotURL)
	}
	receiptURL := ""
	if len(receiptURLs) > 0 {
		receiptURL = receiptURLs[0]
	}

	// Parse with Claude Sonnet (via the parseReceipt seam). On transient
	// failure, retry once (parseReceiptWithSonnet seam, also Sonnet). If the
	// retry ALSO fails, route to pending review with the concatenated
	// parse_error so the owner can see WHY parsing failed on the FE pending
	// card. Phase 260607-e1c (originally Haiku→Sonnet; now Sonnet→Sonnet
	// after Haiku was retired as primary for producing silent errors).
	items, summary, parseErr := parseReceipt(ctx, cfg.AnthropicAPIKey, blobs)
	if parseErr != nil {
		primaryErr := parseErr
		slog.Info(fmt.Sprintf("receipt worker: Sonnet failed for tx %s, retrying: %v", tx.ID, primaryErr))
		items, summary, parseErr = parseReceiptWithSonnet(ctx, cfg.AnthropicAPIKey, blobs)
		if parseErr != nil {
			combined := fmt.Sprintf("sonnet: %v; sonnet-retry: %v", primaryErr, parseErr)
			slog.Info(fmt.Sprintf("receipt worker: Sonnet retry also failed for tx %s: %v — routing to review queue", tx.ID, parseErr))
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, "Receipt could not be parsed automatically", combined, isUpgrade); routeErr != nil {
				slog.Info(fmt.Sprintf("receipt worker: routePending (parse-fail) for tx %s: %v", tx.ID, routeErr))
			}
			return "pending_review", nil
		}
		// Retry succeeded — fall through to ValidateReceiptData with
		// the retry's output. items/summary are now populated by the retry.
	}

	// Goal-driven retry loop with best-attempt tracking.
	//
	// Goal: derivedTotal (sum(price*qty) + tax) == -tx.Amount (Check 1 in
	// validate.go). On failure, retry once with ParseReceiptWithFeedback so
	// Claude sees the bank's ground truth and can find a missed refund / credit
	// memo / etc. MAX_ATTEMPTS=2 caps cost.
	//
	// Best-attempt tracking: instead of always persisting the LAST attempt, we
	// keep the attempt whose score is lowest (closest derivedTotal to -bankAmount,
	// with a hard penalty for empty items). This prevents a feedback regression
	// (Claude returning empty items or a worse parse) from overwriting a better
	// attempt 1 result.
	const maxParseAttempts = 2
	var bestItems []ReceiptItem
	var bestSummary ReceiptSummary
	var bestValidate ValidationResult
	bestScore := math.MaxFloat64
	var retryTrace []string

	for attempt := 1; attempt <= maxParseAttempts; attempt++ {
		validate := ValidateReceiptData(items, summary, tx.Amount)
		score := attemptScore(items, summary, tx.Amount)
		slog.Info(fmt.Sprintf("receipt worker: tx %s attempt %d/%d valid=%v score=%.2f reason=%s",
			tx.ID, attempt, maxParseAttempts, validate.Valid, score, validate.Reason))
		retryTrace = append(retryTrace, fmt.Sprintf("attempt %d: score=%.2f total=%.2f reason=%s",
			attempt, score, summary.Total, validate.Reason))

		if score < bestScore {
			bestItems = items
			bestSummary = summary
			bestValidate = validate
			bestScore = score
		}

		if validate.Valid {
			break
		}
		if attempt == maxParseAttempts {
			break
		}
		// Retry on ANY validation failure. The feedback prompt in
		// ParseReceiptWithFeedback dispatches per-check guidance based on
		// the full validate.Reason string, so Claude gets actionable hints
		// regardless of which check failed.
		newItems, newSummary, feedbackErr := parseReceiptWithFeedback(
			ctx, cfg.AnthropicAPIKey, blobs, summary.Total, tx.Amount, validate.Reason)
		if feedbackErr != nil {
			slog.Info(fmt.Sprintf("receipt worker: tx %s feedback retry failed: %v — using prior attempt", tx.ID, feedbackErr))
			retryTrace = append(retryTrace, fmt.Sprintf("feedback retry errored: %v", feedbackErr))
			break
		}
		items, summary = newItems, newSummary
	}

	// Use the best attempt (lowest score) for all downstream decisions.
	items, summary, lastValidate := bestItems, bestSummary, bestValidate

	// Pre-fill enrichment: fuzzy-match each item name against the existing
	// purchase_items catalog and set PurchaseItemID on matched items BEFORE
	// deciding the route (routePending vs createPurchaseEvent). This lets the
	// FE pre-fill the dropdowns on pending-review cards without any FE change.
	// Enrichment errors are non-fatal — log and proceed with unmatched items.
	matchedCount, catalogSize, enrichErr := enrichItemsWithMatches(ctx, cfg.Pool, items, cfg.AnthropicAPIKey)
	if enrichErr != nil {
		slog.Info(fmt.Sprintf("receipt worker: tx %s enrichItemsWithMatches: %v (continuing without match enrichment)", tx.ID, enrichErr))
	} else {
		slog.Info(fmt.Sprintf("receipt worker: tx %s matched %d/%d items to catalog", tx.ID, matchedCount, len(items)))
	}

	// Low-match-rate sanity gate: when > 5 items but < 30% match the catalog
	// and the catalog itself is healthy (≥ 20 entries), Claude likely returned
	// plausible-sounding but hallucinated names. Force routePending so a human
	// reviews before the row auto-creates. Skip the gate for small catalogs
	// (defensive: production catalog has 106 entries) and for small receipts
	// (< 5 items are statistically insignificant).
	if enrichErr == nil && catalogSize >= 20 && len(items) > 5 &&
		matchedCount*100 < len(items)*30 {
		slog.Info(fmt.Sprintf("receipt worker: tx %s low catalog match rate (%d/%d items matched) — forcing pending review",
			tx.ID, matchedCount, len(items)))
		lastValidate.Valid = false
		lastValidate.Reason = fmt.Sprintf("Low catalog match rate (matched %d/%d items) — verify line items",
			matchedCount, len(items))
	}

	// Sanity gate: never auto-create on empty items or trivially small item
	// sums. Catches Claude's "regressed to empty" failure mode where validate
	// passes vacuously (0 items, 0 sum, 0 tax matches a 0 bank amount).
	if lastValidate.Valid {
		itemsSum := 0.0
		for _, item := range items {
			itemsSum += item.Price * item.Quantity
		}
		if len(items) == 0 || math.Abs(itemsSum) < 0.50 {
			slog.Info(fmt.Sprintf("receipt worker: tx %s sanity gate FAILED (items=%d itemsSum=%.2f) — routing to review",
				tx.ID, len(items), itemsSum))
			lastValidate.Valid = false
			if lastValidate.Reason == "" {
				lastValidate.Reason = fmt.Sprintf("Sanity gate: items=%d itemsSum=%.2f", len(items), itemsSum)
			}
		}
	}

	if !lastValidate.Valid {
		parseErrForRow := ""
		if len(retryTrace) > 0 {
			parseErrForRow = strings.Join(retryTrace, "; ")
		}
		slog.Info(fmt.Sprintf("receipt worker: tx %s routed to review queue: %s", tx.ID, lastValidate.Reason))
		if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, lastValidate.Reason, parseErrForRow, isUpgrade); routeErr != nil {
			slog.Info(fmt.Sprintf("receipt worker: routePending (validate-fail) for tx %s: %v", tx.ID, routeErr))
		}
		return "pending_review", nil
	}

	// Overwrite summary.Total with the derived value so the persisted
	// purchase_events.total reflects the bank-matched amount, not whatever
	// Claude reported. Claude's summary.Total may be purchase-only on a
	// purchase+refund receipt (multi-image inconsistency); the derived value
	// (itemsSum + tax) is what validate.go used to pass Check 1 and is
	// guaranteed to match the bank amount within $0.01.
	derivedItemsSum := 0.0
	for _, item := range items {
		derivedItemsSum += item.Price * item.Quantity
	}
	summary.Total = derivedItemsSum + summary.Tax

	// Auto-create purchase event. isUpgrade=true causes the helper to
	// DELETE the stale pending row inside the same DB transaction as
	// the event INSERT — atomic upgrade with no window for a concurrent
	// re-sync to see both rows.
	//
	// nil return from createPurchaseEvent also covers the duplicate-key
	// dead-letter path: if the INSERT fails with a unique violation on
	// purchase_events_bank_tx_id_key, createPurchaseEvent deletes the
	// residual pending row and returns nil so we count it as auto_created
	// (the event exists).
	if err := createPurchaseEvent(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, isUpgrade); err != nil {
		slog.Info(fmt.Sprintf("receipt worker: createPurchaseEvent for tx %s: %v — routing to review queue", tx.ID, err))
		if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, "Receipt could not be saved automatically", "", isUpgrade); routeErr != nil {
			slog.Info(fmt.Sprintf("receipt worker: routePending (save-fail) for tx %s: %v", tx.ID, routeErr))
		}
		return "pending_review", nil
	}

	return "auto_created", nil
}

// attemptScore returns a quality score for a parse attempt. LOWER is better.
// Primary: distance of derivedTotal from -bankAmount (in dollars).
// Tiebreaker: empty items penalize hard (so non-empty always beats empty).
func attemptScore(items []ReceiptItem, summary ReceiptSummary, bankAmount float64) float64 {
	itemsSum := 0.0
	for _, item := range items {
		itemsSum += item.Price * item.Quantity
	}
	derived := itemsSum + summary.Tax
	dist := math.Abs(derived - (-bankAmount))
	if len(items) == 0 {
		dist += 10000 // hard penalty so empty NEVER beats non-empty
	}
	return dist
}

// routePending dispatches the parse-fail / validate-fail / save-fail branches
// of runIngestCycle to either INSERT a new pending_purchases row (cold path)
// or UPDATE the existing one in place (upgrade path, isUpgrade=true). Keeps
// the call sites in runIngestCycle one-liners.
//
// parseError carries the concatenated Haiku+Sonnet error string from the
// (haiku→sonnet) double-fail path; all other call sites pass "" (column
// stays NULL).
func routePending(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, receiptURLs []string, reason string, parseError string, isUpgrade bool) error {
	if isUpgrade {
		return updatePendingPurchase(ctx, pool, tx, items, summary, receiptURL, receiptURLs, reason, parseError)
	}
	return insertPendingPurchase(ctx, pool, tx, items, summary, receiptURL, receiptURLs, reason, parseError)
}

// classifyExistingTx reports what HQ already has for a given Mercury bank_tx_id.
// Replaces the old bankTxIDExists 2-way guard so the worker can distinguish a
// "real" cached row (skip) from a stale no_attachment_on_bank_tx pending row
// whose Mercury tx now has receipt attachments (upgrade — reprocess).
//
// Return contract:
//   - kind "none"    → not seen yet; runIngestCycle should ingest normally.
//   - kind "event"   → already in purchase_events OR a confirmed pending row.
//                      Idempotency win: a confirmed pending row represents a
//                      real, locked purchase, so it behaves like an event.
//                      reason is "" for this kind.
//   - kind "pending" → still-open pending_purchases row
//                      (confirmed_at IS NULL AND discarded_at IS NULL).
//                      reason carries pending_purchases.reason so the caller
//                      can decide if this is the upgrade-eligible
//                      "no_attachment_on_bank_tx" case.
//
// hasParseError / hasItems are ONLY meaningful for kind="pending":
//   - hasParseError = true when pending_purchases.parse_error IS NOT NULL.
//                     Phase 260607-fxl uses this to gate parse-failed retries
//                     so we never re-call the parser on a row where BOTH Haiku
//                     and Sonnet already failed (parse_error populated).
//   - hasItems      = true when pending_purchases.items is a non-empty JSONB
//                     array. Phase 260607-fxl uses this so a user-edited row
//                     (operator added line items already) is never clobbered
//                     by a worker re-parse.
//
// For kind="event" and kind="none", hasParseError and hasItems are always
// false — callers should not branch on them in those cases.
//
// Discarded pending rows return kind="none" — the user explicitly threw the
// row away, so the worker is free to re-process the same bank_tx_id.
func classifyExistingTx(ctx context.Context, pool *pgxpool.Pool, bankTxID string) (kind, reason string, hasParseError, hasItems bool, err error) {
	err = pool.QueryRow(ctx, `
		SELECT 'event' AS kind, '' AS reason, false AS has_parse_error, false AS has_items
		  FROM purchase_events WHERE bank_tx_id = $1
		UNION ALL
		SELECT 'event' AS kind, COALESCE(reason,'') AS reason, false, false
		  FROM pending_purchases
		 WHERE bank_tx_id = $1 AND confirmed_at IS NOT NULL
		UNION ALL
		SELECT 'pending' AS kind, COALESCE(reason,'') AS reason,
		       (parse_error IS NOT NULL),
		       (jsonb_typeof(items) = 'array' AND jsonb_array_length(items) > 0)
		  FROM pending_purchases
		 WHERE bank_tx_id = $1
		   AND confirmed_at IS NULL
		   AND discarded_at IS NULL
		LIMIT 1`, bankTxID).Scan(&kind, &reason, &hasParseError, &hasItems)
	if errors.Is(err, pgx.ErrNoRows) {
		return "none", "", false, false, nil
	}
	if err != nil {
		return "", "", false, false, fmt.Errorf("classifyExistingTx: %w", err)
	}
	return kind, reason, hasParseError, hasItems, nil
}

// createPurchaseEvent inserts a new purchase_event and its line items within
// a DB transaction, auto-creating the vendor and any new purchase items.
//
// isUpgrade=true (Phase 260607-co0): the caller has classified an existing
// pending_purchases row with reason='no_attachment_on_bank_tx' as upgrade-
// eligible (the Mercury tx now has a receipt attachment). In that case the
// helper runs `DELETE FROM pending_purchases WHERE bank_tx_id=$1` as the FIRST
// statement inside the same dbTx as the event INSERT, so the swap is atomic:
// either both the DELETE and the INSERT commit, or neither does.
func createPurchaseEvent(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, receiptURLs []string, isUpgrade bool) error {
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("createPurchaseEvent: begin: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

	// Upgrade path: delete the stale no-attachment pending row in the same
	// transaction so we never publish a state where both rows exist
	// simultaneously. A concurrent re-sync hitting between classify and
	// commit will either see the pending row (and re-process — at worst
	// triggers a duplicate event INSERT that fails at commit, the loser
	// retries next tick) or see the new event row (and short-circuit).
	if isUpgrade {
		if _, err := dbTx.Exec(ctx,
			`DELETE FROM pending_purchases WHERE bank_tx_id = $1`,
			tx.ID,
		); err != nil {
			return fmt.Errorf("createPurchaseEvent: delete upgrade pending: %w", err)
		}
	}

	// Upsert vendor
	var vendorID string
	err = dbTx.QueryRow(ctx,
		`INSERT INTO vendors (name) VALUES ($1)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`,
		strings.Title(strings.TrimSpace(summary.Vendor)), //nolint:staticcheck
	).Scan(&vendorID)
	if err != nil {
		return fmt.Errorf("createPurchaseEvent: upsert vendor: %w", err)
	}

	// Parse event date from Mercury CreatedAt
	eventDate := parseEventDate(tx.CreatedAt)

	// Derive mercury_category (nil-safe — NULL when Mercury hasn't classified yet)
	var mercuryCategory string
	if tx.CategoryData != nil {
		mercuryCategory = tx.CategoryData.Name
	}

	// Insert purchase_event
	var eventID string
	err = dbTx.QueryRow(ctx,
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, receipt_url, receipt_urls, mercury_category)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id`,
		vendorID, tx.ID, eventDate, summary.Tax, summary.Total, nullableString(receiptURL), receiptURLsJSON(receiptURLs), nullableString(mercuryCategory),
	).Scan(&eventID)
	if err != nil {
		// Duplicate bank_tx_id: a purchase_event for this tx was already created
		// (e.g. by an earlier sync today). The pending row is a residual artifact.
		// After a constraint violation, the pgx transaction is in an aborted state;
		// no further commands can run on it. Roll it back explicitly, then clean up
		// the residual pending row in a fresh connection from the pool.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "bank_tx_id") {
			slog.Info(fmt.Sprintf("receipt worker: tx %s already auto-created (duplicate bank_tx_id) — clearing residual pending row", tx.ID))
			// Rollback the aborted transaction explicitly (defer above would also do
			// it, but being explicit avoids a log-confusing error from the defer).
			_ = dbTx.Rollback(ctx)
			// Delete the residual pending row via the pool (outside the aborted tx).
			if _, delErr := pool.Exec(ctx,
				`DELETE FROM pending_purchases WHERE bank_tx_id = $1 AND confirmed_at IS NULL AND discarded_at IS NULL`,
				tx.ID,
			); delErr != nil {
				return fmt.Errorf("createPurchaseEvent: clear residual pending for duplicate tx %s: %w", tx.ID, delErr)
			}
			return nil
		}
		return fmt.Errorf("createPurchaseEvent: insert purchase_event: %w", err)
	}

	// Load existing purchase items for fuzzy matching
	existingItems, err := loadPurchaseItemsMap(ctx, pool)
	if err != nil {
		return fmt.Errorf("createPurchaseEvent: load purchase items: %w", err)
	}

	// Insert line items
	for _, item := range items {
		itemID, itemName, isNew := DerivePurchaseItemID(item.Name, existingItems)

		// Never auto-create a catalog item from an empty name — description
		// is UNIQUE, so a single '' row would absorb every future unnamed
		// line. Validation (Check 0 in validate.go) routes unnamed receipts
		// to review before reaching here; this guard covers any other caller.
		// nullableStringPtr maps the empty itemID to NULL, leaving the line
		// unlinked instead.
		if isNew && strings.TrimSpace(itemName) == "" {
			isNew = false
		}

		if isNew {
			// Auto-create the purchase item
			err = dbTx.QueryRow(ctx,
				`INSERT INTO purchase_items (description)
				 VALUES ($1)
				 ON CONFLICT (description) DO UPDATE SET description = EXCLUDED.description
				 RETURNING id`,
				itemName,
			).Scan(&itemID)
			if err != nil {
				return fmt.Errorf("createPurchaseEvent: upsert purchase_item %q: %w", itemName, err)
			}
			existingItems[itemName] = itemID
		}

		// ReceiptItem.Quantity is float64 (tolerates LLM-returned decimals like 40.0)
		// but purchase_line_items.quantity is INTEGER — round at the DB-write boundary.
		_, err = dbTx.Exec(ctx,
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			eventID, nullableStringPtr(&itemID), itemName, int(math.Round(item.Quantity)), item.Price, item.IsCase,
		)
		if err != nil {
			return fmt.Errorf("createPurchaseEvent: insert line_item %q: %w", item.Name, err)
		}
	}

	if err := dbTx.Commit(ctx); err != nil {
		return fmt.Errorf("createPurchaseEvent: commit: %w", err)
	}
	return nil
}

// insertPendingPurchase inserts a failed-validation transaction into the
// pending_purchases review queue.
//
// parseError, when non-empty, is stored on pending_purchases.parse_error so
// the FE can render the actual Anthropic/parse error string on the pending
// card. Empty string stays NULL (column was added in migration 0069).
func insertPendingPurchase(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, receiptURLs []string, reason string, parseError string) error {
	// items==nil guard: json.Marshal(nil) returns []byte("null"), which the FE
	// can crash on with .length / .map. A nil slice and an empty
	// []ReceiptItem{} slice are distinguishable via items == nil; treat nil
	// (the no_attachment branch) as the empty JSON array.
	var itemsJSON []byte
	if items == nil {
		itemsJSON = []byte("[]")
	} else {
		var err error
		itemsJSON, err = json.Marshal(items)
		if err != nil {
			itemsJSON = []byte("[]")
		}
	}

	eventDate := parseEventDate(tx.CreatedAt)

	// Vendor fallback: Mercury attaches a bankDescription to every card
	// transaction (e.g. "RESTAURANT DEPOT 0123 CHICAGO IL"). The
	// no_attachment_on_bank_tx branch passes an empty ReceiptSummary so
	// summary.Vendor is "" — without this fallback every unreceipted
	// pending renders as "Unknown Vendor" in the Purchases tab. When
	// Claude successfully parses a receipt, summary.Vendor wins (the
	// curated name beats the raw bank string).
	vendor := summary.Vendor
	if vendor == "" {
		vendor = tx.BankDescription
	}

	// Derive mercury_category (nil-safe — NULL when Mercury hasn't classified yet).
	// Mirrors createPurchaseEvent. /period-summary uses this to gate whether
	// a pending row blocks payroll (COGS-category + no_attachment = blocks)
	// or rolls into COGS at bank_total (COGS-category + parse-failed receipt).
	var mercuryCategory string
	if tx.CategoryData != nil {
		mercuryCategory = tx.CategoryData.Name
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO pending_purchases
		 (bank_tx_id, bank_total, vendor, event_date, tax, total, items, reason, receipt_url, receipt_urls, mercury_category, parse_error)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 ON CONFLICT (bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL DO NOTHING`,
		tx.ID,
		tx.Amount,
		vendor,
		nullableString(eventDate),
		nullableFloat64(summary.Tax),
		nullableFloat64(summary.Total),
		itemsJSON,
		nullableString(reason),
		nullableString(receiptURL),
		receiptURLsJSON(receiptURLs),
		nullableString(mercuryCategory),
		nullableString(parseError),
	)
	if err != nil {
		return fmt.Errorf("insertPendingPurchase: %w", err)
	}
	return nil
}

// updatePendingPurchase updates an existing pending_purchases row IN PLACE,
// keyed by bank_tx_id. Mirrors insertPendingPurchase field-for-field but
// issues UPDATE instead of INSERT — used by the upgrade path
// (re-syncing a no_attachment_on_bank_tx row whose Mercury tx now has a
// receipt) when the parse / validate still fails so the row stays pending
// for human review. Preserves the row's UUID PK so any FE references to the
// pending row remain stable.
//
// rowcount==0 is NOT an error: between classifyExistingTx and this UPDATE
// the row may have been confirmed or discarded by a concurrent user action.
// The next worker poll will re-classify cleanly.
func updatePendingPurchase(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, receiptURLs []string, reason string, parseError string) error {
	// items==nil guard — mirrors insertPendingPurchase. See note there.
	var itemsJSON []byte
	if items == nil {
		itemsJSON = []byte("[]")
	} else {
		var err error
		itemsJSON, err = json.Marshal(items)
		if err != nil {
			itemsJSON = []byte("[]")
		}
	}

	eventDate := parseEventDate(tx.CreatedAt)

	// Vendor fallback — same rule as insertPendingPurchase: curated summary
	// wins; raw bankDescription as fallback.
	vendor := summary.Vendor
	if vendor == "" {
		vendor = tx.BankDescription
	}

	var mercuryCategory string
	if tx.CategoryData != nil {
		mercuryCategory = tx.CategoryData.Name
	}

	_, err := pool.Exec(ctx,
		`UPDATE pending_purchases
		    SET bank_total       = $2,
		        vendor           = $3,
		        event_date       = $4,
		        tax              = $5,
		        total            = $6,
		        items            = $7,
		        reason           = $8,
		        receipt_url      = $9,
		        receipt_urls     = $10,
		        mercury_category = $11,
		        parse_error      = $12
		  WHERE bank_tx_id = $1`,
		tx.ID,
		tx.Amount,
		vendor,
		nullableString(eventDate),
		nullableFloat64(summary.Tax),
		nullableFloat64(summary.Total),
		itemsJSON,
		nullableString(reason),
		nullableString(receiptURL),
		receiptURLsJSON(receiptURLs),
		nullableString(mercuryCategory),
		nullableString(parseError),
	)
	if err != nil {
		return fmt.Errorf("updatePendingPurchase: %w", err)
	}
	return nil
}

// backfillPendingVendor sets pending_purchases.vendor to Mercury's
// bankDescription for the given tx when the row exists with a missing
// vendor. The IS NULL OR = '' guard means a receipt-parsed pending whose
// vendor Claude already set is never overwritten. Idempotent on re-poll.
// Empty bankDescription is a no-op.
func backfillPendingVendor(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction) error {
	if tx.BankDescription == "" {
		return nil
	}
	_, err := pool.Exec(ctx,
		`UPDATE pending_purchases
		 SET vendor = $1
		 WHERE bank_tx_id = $2
		   AND (vendor IS NULL OR vendor = '')`,
		tx.BankDescription, tx.ID)
	return err
}

// loadPurchaseItemsMap returns a map of name -> id for all purchase_items,
// including learned aliases (item_aliases) as additional keys. Descriptions
// take precedence: an alias that case-insensitively collides with any
// description is skipped, so an alias can never shadow a catalog name.
// Alias keys make previously human-linked receipt text an exact match in
// DerivePurchaseItemID, ahead of the fuzzy/AI stages.
func loadPurchaseItemsMap(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT id, description FROM purchase_items`)
	if err != nil {
		return nil, fmt.Errorf("loadPurchaseItemsMap: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	seen := make(map[string]bool)
	for rows.Next() {
		var id, desc string
		if err := rows.Scan(&id, &desc); err != nil {
			return nil, fmt.Errorf("loadPurchaseItemsMap scan: %w", err)
		}
		m[desc] = id
		seen[strings.ToLower(desc)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	aliasRows, err := pool.Query(ctx, `SELECT purchase_item_id, alias FROM item_aliases`)
	if err != nil {
		return nil, fmt.Errorf("loadPurchaseItemsMap aliases: %w", err)
	}
	defer aliasRows.Close()
	for aliasRows.Next() {
		var id, alias string
		if err := aliasRows.Scan(&id, &alias); err != nil {
			return nil, fmt.Errorf("loadPurchaseItemsMap alias scan: %w", err)
		}
		if seen[strings.ToLower(alias)] {
			continue
		}
		m[alias] = id
		seen[strings.ToLower(alias)] = true
	}
	return m, aliasRows.Err()
}

// enrichItemsWithMatches populates each item's PurchaseItemID using a
// two-stage pipeline:
//
//  1. Stage 1 (fast, deterministic): exact case-insensitive match + Jaro-Winkler
//     (DerivePurchaseItemID, threshold 0.85), then token-overlap match at 0.7
//     threshold (matchByTokens). Bridges the gap between catalog names like
//     "Lemonade Mix" and SKU-style receipt names like "4C LEMONADE 35QT".
//
//  2. Stage 2 (AI fallback): items still unmatched after Stage 1 are sent to
//     Claude Haiku in a single batch call. Only "high" confidence AI matches
//     are accepted; lower confidence is left unmatched for the human to resolve
//     in the FE. Errors are non-fatal — we log and proceed.
//
// Returns the number of items matched and the catalog size so the caller can
// apply the low-match-rate sanity gate.
func enrichItemsWithMatches(ctx context.Context, pool *pgxpool.Pool, items []ReceiptItem, anthropicAPIKey string) (matched int, catalogSize int, err error) {
	existingMap, err := loadPurchaseItemsMap(ctx, pool)
	if err != nil {
		return 0, 0, fmt.Errorf("enrichItemsWithMatches: %w", err)
	}
	catalogSize = len(existingMap)

	var unmatchedNames []string
	var unmatchedIdx []int

	// Pass 1: exact/JW match (DerivePurchaseItemID) then token-overlap match.
	for i := range items {
		if items[i].PurchaseItemID != nil {
			// Already matched upstream (e.g. a prior pass) — count and skip.
			matched++
			continue
		}

		// Step 1a: exact case-insensitive + Jaro-Winkler 0.85 (existing logic).
		if id, _, isNew := DerivePurchaseItemID(items[i].Name, existingMap); !isNew {
			idCopy := id
			items[i].PurchaseItemID = &idCopy
			matched++
			continue
		}

		// Step 1b: token-overlap at 0.7 threshold.
		if id, _, _ := matchByTokens(items[i].Name, existingMap, tokenMatchThreshold); id != "" {
			idCopy := id
			items[i].PurchaseItemID = &idCopy
			matched++
			continue
		}

		// Neither stage matched — defer to AI.
		unmatchedNames = append(unmatchedNames, items[i].Name)
		unmatchedIdx = append(unmatchedIdx, i)
	}

	// Pass 2: AI fallback for remaining unmatched items.
	if len(unmatchedNames) > 0 && anthropicAPIKey != "" {
		aiMatches, aiErr := matchItemsWithAI(ctx, anthropicAPIKey, unmatchedNames, existingMap)
		if aiErr != nil {
			slog.Info(fmt.Sprintf("receipt worker: AI item matching failed: %v (continuing with partial matches)", aiErr))
		} else {
			for k, idx := range unmatchedIdx {
				rawName := unmatchedNames[k]
				if id, ok := aiMatches[rawName]; ok {
					idCopy := id
					items[idx].PurchaseItemID = &idCopy
					matched++
				}
			}
		}
	}

	return matched, catalogSize, nil
}

// parseEventDate extracts a YYYY-MM-DD date string from a Mercury CreatedAt
// value, which is typically an ISO 8601 timestamp.
func parseEventDate(createdAt string) string {
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, createdAt); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return time.Now().Format("2006-01-02")
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableStringPtr(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func nullableFloat64(f float64) interface{} {
	if f == 0 {
		return nil
	}
	return f
}

// receiptURLsJSON marshals a URL slice to JSON bytes for JSONB storage.
// Returns nil when the slice is empty so the column stays NULL rather than
// storing an empty array. pgx accepts []byte for JSONB parameters directly.
// uploadReceiptSlot uploads one receipt attachment blob to object storage at
// the per-index key receipts/{txID}/{i}{ext} and returns the permanent public
// URL. The extension comes from the attachment's original file name (default
// .jpg). No x-amz-acl header is sent: the presigned URL doesn't sign one and
// B2 rejects per-object ACLs — the bucket itself is public (B-172).
// Callers decide the failure policy: the ingest worker falls back to the
// expiring Mercury URL, the recovery path counts the tx as failed.
func uploadReceiptSlot(ctx context.Context, cfg WorkerConfig, txID string, i int, blob FileBlob, fileName string) (string, error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		ext = ".jpg"
	}
	key := fmt.Sprintf("receipts/%s/%d%s", txID, i, ext)
	presignedURL, err := photos.GeneratePresignedPutURL(ctx, cfg.SpacesPresigner, cfg.SpacesBucket, key, blob.ContentType, 15*time.Minute)
	if err != nil {
		return "", fmt.Errorf("presign slot %d: %w", i, err)
	}
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(blob.Bytes))
	if err != nil {
		return "", fmt.Errorf("create PUT request slot %d: %w", i, err)
	}
	putReq.Header.Set("Content-Type", blob.ContentType)
	putReq.ContentLength = int64(len(blob.Bytes))
	putResp, err := (&http.Client{Timeout: 60 * time.Second}).Do(putReq)
	if err != nil {
		return "", fmt.Errorf("PUT slot %d: %w", i, err)
	}
	putResp.Body.Close()
	if putResp.StatusCode < 200 || putResp.StatusCode >= 300 {
		return "", fmt.Errorf("PUT slot %d returned status %d", i, putResp.StatusCode)
	}
	return photos.PublicURL(cfg.SpacesEndpoint, cfg.SpacesBucket, key), nil
}

func receiptURLsJSON(urls []string) interface{} {
	if len(urls) == 0 {
		return nil
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return nil
	}
	return b
}
