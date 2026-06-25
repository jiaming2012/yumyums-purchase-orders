package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	fetchTransactions          = FetchTransactions
	parseReceipt               = ParseReceipt
	parseReceiptWithSonnet     = ParseReceiptWithSonnet
	parseReceiptWithFeedback   = ParseReceiptWithFeedback
	downloadReceiptFileFn      = downloadReceiptFile
)

// StartWorker launches a background goroutine that polls Mercury for new
// transactions on the configured interval. If either API key is missing the
// worker logs a warning and returns immediately (graceful skip).
func StartWorker(ctx context.Context, cfg WorkerConfig) {
	if cfg.MercuryAPIKey == "" || cfg.AnthropicAPIKey == "" {
		log.Println("WARNING: receipt worker: skipping — missing API keys (MERCURY_API_KEY or ANTHROPIC_API_KEY not set)")
		return
	}

	interval := cfg.Interval
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	log.Printf("receipt worker: starting (interval=%s, lookback=%dd)", interval, cfg.LookbackDays)

	go func() {
		// Run immediately on start, then on each tick
		if _, err := runIngestCycle(ctx, cfg); err != nil {
			log.Printf("receipt worker: ingest cycle error: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("receipt worker: shutting down")
				return
			case <-ticker.C:
				if _, err := runIngestCycle(ctx, cfg); err != nil {
					log.Printf("receipt worker: ingest cycle error: %v", err)
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
		log.Println("receipt worker: no supported transactions found")
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
				log.Printf("receipt worker: refresh mercury_category for tx %s: %v (continuing)", tx.ID, refreshErr)
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
				log.Printf("receipt worker: refresh pending_purchases.mercury_category for tx %s: %v (continuing)", tx.ID, refreshErr)
			}
		}

		// Backfill pending_purchases.vendor from Mercury's bankDescription
		// for rows that landed before the BankDescription-fallback shipped
		// (260606-hew). Runs alongside mercury_category refresh — before
		// the `already` short-circuit — so cached rows within the lookback
		// window auto-backfill on the next poll. No separate one-shot
		// migration.
		if backfillErr := backfillPendingVendor(ctx, cfg.Pool, tx); backfillErr != nil {
			log.Printf("receipt worker: backfill vendor for tx %s: %v (continuing)", tx.ID, backfillErr)
		}

		// Idempotency: 3-way classify so we can detect the "stale no-attachment
		// pending row whose Mercury tx now has a receipt" upgrade case.
		// Phase 260607-co0: replaces the 2-way bankTxIDExists short-circuit
		// that previously skipped these rows forever.
		// Phase 260607-fxl: classifyExistingTx now also returns hasParseError
		// + hasItems so the upgrade gate can retry pre-260607-e1c parse-failed
		// rows once (parse_error NULL → never tried Sonnet) without clobbering
		// user-edited rows (items non-empty).
		kind, existingReason, hasParseError, hasItems, err := classifyExistingTx(ctx, cfg.Pool, tx.ID)
		if err != nil {
			log.Printf("receipt worker: classifyExistingTx tx %s: %v", tx.ID, err)
			continue
		}
		isUpgrade := false
		switch kind {
		case "event":
			// Already a purchase_event (or a confirmed pending) — done.
			skippedCached++
			continue
		case "pending":
			// Two upgrade cases:
			//  (a) Existing: no-attachment row whose Mercury tx now has a
			//      receipt — promote it through the normal parse path.
			//  (b) Phase 260607-fxl: pre-e1c parse-failed row that never
			//      got a Sonnet attempt. Gate by parse_error IS NULL (NOT
			//      both-models-failed already) AND items empty (operator
			//      hasn't started editing). Requires attachments to retry
			//      against.
			noAttachmentUpgrade := existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0
			parseFailedRetry := existingReason == "Receipt could not be parsed automatically" &&
				!hasParseError && !hasItems && len(tx.Attachments) > 0
			if noAttachmentUpgrade || parseFailedRetry {
				isUpgrade = true
				// Fall through to download/parse path below.
			} else {
				skippedCached++
				continue
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
					log.Printf("receipt worker: insertPendingPurchase (no-attachment) for tx %s: %v", tx.ID, routeErr)
				}
				pendingReview++
			}
			continue
		}

		// Download every attachment and collect FileBlobs so all receipts for
		// this transaction are sent to Claude in a single multi-image prompt.
		// This handles the purchase + refund case: both files are seen together,
		// and Claude returns a single combined summary whose Total is the net.
		var blobs []FileBlob
		for _, att := range tx.Attachments {
			fb, ct, dlErr := downloadReceiptFileFn(ctx, att.URL)
			if dlErr != nil {
				log.Printf("receipt worker: download attachment %s for tx %s: %v (skipping attachment)", att.URL, tx.ID, dlErr)
				continue
			}
			blobs = append(blobs, FileBlob{Bytes: fb, ContentType: ct})
		}
		if len(blobs) == 0 {
			log.Printf("receipt worker: all attachments failed to download for tx %s — skipping", tx.ID)
			continue
		}

		// Upload all attachments to DO Spaces in order. Each gets a
		// per-index key receipts/{tx.ID}/{i}{ext} so they can coexist.
		// receiptURLs collects the final public (or fallback Mercury) URL
		// for each slot. receiptURL (singular) is set to receiptURLs[0]
		// for backward compat with the existing singular-column INSERT calls.
		receiptURLs := make([]string, 0, len(blobs))
		for i, blob := range blobs {
			att := tx.Attachments[i]
			slotURL := att.URL // fallback: original Mercury URL
			if cfg.SpacesPresigner != nil && cfg.SpacesBucket != "" {
				ext := strings.ToLower(filepath.Ext(att.FileName))
				if ext == "" {
					ext = ".jpg"
				}
				key := fmt.Sprintf("receipts/%s/%d%s", tx.ID, i, ext)
				presignedURL, uploadErr := photos.GeneratePresignedPutURL(ctx, cfg.SpacesPresigner, cfg.SpacesBucket, key, blob.ContentType, 15*time.Minute)
				if uploadErr != nil {
					log.Printf("receipt worker: presign slot %d for tx %s: %v (falling back to Mercury URL)", i, tx.ID, uploadErr)
				} else {
					putReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(blob.Bytes))
					if reqErr != nil {
						log.Printf("receipt worker: create PUT request slot %d for tx %s: %v (falling back to Mercury URL)", i, tx.ID, reqErr)
					} else {
						putReq.Header.Set("Content-Type", blob.ContentType)
						putReq.Header.Set("x-amz-acl", "public-read")
						putReq.ContentLength = int64(len(blob.Bytes))
						putResp, putErr := (&http.Client{Timeout: 60 * time.Second}).Do(putReq)
						if putErr != nil {
							log.Printf("receipt worker: upload slot %d to Spaces for tx %s: %v (falling back to Mercury URL)", i, tx.ID, putErr)
						} else {
							putResp.Body.Close()
							if putResp.StatusCode >= 200 && putResp.StatusCode < 300 {
								slotURL = photos.PublicURL(cfg.SpacesEndpoint, cfg.SpacesBucket, key)
							} else {
								log.Printf("receipt worker: Spaces PUT slot %d for tx %s returned %d (falling back to Mercury URL)", i, tx.ID, putResp.StatusCode)
							}
						}
					}
				}
			}
			receiptURLs = append(receiptURLs, slotURL)
		}
		receiptURL := ""
		if len(receiptURLs) > 0 {
			receiptURL = receiptURLs[0]
		}

		// Parse with Claude Haiku (via the parseReceipt seam). On Haiku failure,
		// retry once with Sonnet (parseReceiptWithSonnet seam). If Sonnet ALSO
		// fails, route to pending review with the concatenated parse_error so
		// the owner can see WHY parsing failed on the FE pending card.
		// Phase 260607-e1c.
		items, summary, err := parseReceipt(ctx, cfg.AnthropicAPIKey, blobs)
		if err != nil {
			haikuErr := err
			log.Printf("receipt worker: Haiku failed for tx %s, retrying with Sonnet: %v", tx.ID, haikuErr)
			items, summary, err = parseReceiptWithSonnet(ctx, cfg.AnthropicAPIKey, blobs)
			if err != nil {
				combined := fmt.Sprintf("haiku: %v; sonnet: %v", haikuErr, err)
				log.Printf("receipt worker: Sonnet also failed for tx %s: %v — routing to review queue", tx.ID, err)
				if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, "Receipt could not be parsed automatically", combined, isUpgrade); routeErr != nil {
					log.Printf("receipt worker: routePending (parse-fail) for tx %s: %v", tx.ID, routeErr)
				}
				pendingReview++
				continue
			}
			// Sonnet succeeded — fall through to ValidateReceiptData with
			// Sonnet's output. items/summary/err are now populated by Sonnet.
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
			log.Printf("receipt worker: tx %s attempt %d/%d valid=%v score=%.2f reason=%s",
				tx.ID, attempt, maxParseAttempts, validate.Valid, score, validate.Reason)
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
				log.Printf("receipt worker: tx %s feedback retry failed: %v — using prior attempt", tx.ID, feedbackErr)
				retryTrace = append(retryTrace, fmt.Sprintf("feedback retry errored: %v", feedbackErr))
				break
			}
			items, summary = newItems, newSummary
		}

		// Use the best attempt (lowest score) for all downstream decisions.
		items, summary, lastValidate := bestItems, bestSummary, bestValidate

		// Sanity gate: never auto-create on empty items or trivially small item
		// sums. Catches Claude's "regressed to empty" failure mode where validate
		// passes vacuously (0 items, 0 sum, 0 tax matches a 0 bank amount).
		if lastValidate.Valid {
			itemsSum := 0.0
			for _, item := range items {
				itemsSum += item.Price * item.Quantity
			}
			if len(items) == 0 || math.Abs(itemsSum) < 0.50 {
				log.Printf("receipt worker: tx %s sanity gate FAILED (items=%d itemsSum=%.2f) — routing to review",
					tx.ID, len(items), itemsSum)
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
			log.Printf("receipt worker: tx %s routed to review queue: %s", tx.ID, lastValidate.Reason)
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, lastValidate.Reason, parseErrForRow, isUpgrade); routeErr != nil {
				log.Printf("receipt worker: routePending (validate-fail) for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
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
		if err := createPurchaseEvent(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, isUpgrade); err != nil {
			log.Printf("receipt worker: createPurchaseEvent for tx %s: %v — routing to review queue", tx.ID, err)
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, receiptURLs, "Receipt could not be saved automatically", "", isUpgrade); routeErr != nil {
				log.Printf("receipt worker: routePending (save-fail) for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		autoCreated++
	}

	log.Printf("receipt worker: processed %d transactions, %d auto-created, %d pending review, %d already cached",
		len(txns), autoCreated, pendingReview, skippedCached)
	return IngestResult{
		Processed:     len(txns),
		AutoCreated:   autoCreated,
		PendingReview: pendingReview,
		Cached:        skippedCached,
	}, nil
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

// loadPurchaseItemsMap returns a map of description -> id for all purchase_items.
func loadPurchaseItemsMap(ctx context.Context, pool *pgxpool.Pool) (map[string]string, error) {
	rows, err := pool.Query(ctx, `SELECT id, description FROM purchase_items`)
	if err != nil {
		return nil, fmt.Errorf("loadPurchaseItemsMap: %w", err)
	}
	defer rows.Close()

	m := make(map[string]string)
	for rows.Next() {
		var id, desc string
		if err := rows.Scan(&id, &desc); err != nil {
			return nil, fmt.Errorf("loadPurchaseItemsMap scan: %w", err)
		}
		m[desc] = id
	}
	return m, rows.Err()
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
