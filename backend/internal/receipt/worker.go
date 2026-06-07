package receipt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	fetchTransactions     = FetchTransactions
	parseReceipt          = ParseReceipt
	downloadReceiptFileFn = downloadReceiptFile
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
		kind, existingReason, err := classifyExistingTx(ctx, cfg.Pool, tx.ID)
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
			// Re-process if (and only if) the row was the no-attachment
			// sentinel and the Mercury tx now has at least one attachment.
			// Any other reason (parse failure, validate failure) stays in
			// the human-review queue.
			if existingReason == "no_attachment_on_bank_tx" && len(tx.Attachments) > 0 {
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
					"no_attachment_on_bank_tx",
				); routeErr != nil {
					log.Printf("receipt worker: insertPendingPurchase (no-attachment) for tx %s: %v", tx.ID, routeErr)
				}
				pendingReview++
			}
			continue
		}

		// Choose best attachment — prefer PDF for known multi-attachment vendors
		attachment := pickAttachment(tx)
		if attachment == nil {
			log.Printf("receipt worker: transaction %s has attachments but none selected — skipping", tx.ID)
			continue
		}

		// Download receipt file (via the downloadReceiptFileFn seam so tests
		// can inject fake bytes without an httptest.Server).
		fileBytes, contentType, err := downloadReceiptFileFn(ctx, attachment.URL)
		if err != nil {
			log.Printf("receipt worker: download attachment for tx %s: %v", tx.ID, err)
			continue
		}

		// Optionally upload original to DO Spaces
		receiptURL := attachment.URL
		if cfg.SpacesPresigner != nil && cfg.SpacesBucket != "" {
			ext := strings.ToLower(filepath.Ext(attachment.FileName))
			if ext == "" {
				ext = ".jpg"
			}
			key := fmt.Sprintf("receipts/%s/original%s", tx.ID, ext)
			presignedURL, uploadErr := photos.GeneratePresignedPutURL(ctx, cfg.SpacesPresigner, cfg.SpacesBucket, key, contentType, 15*time.Minute)
			if uploadErr != nil {
				log.Printf("receipt worker: presign for tx %s: %v (continuing)", tx.ID, uploadErr)
			} else {
				putReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, bytes.NewReader(fileBytes))
				if reqErr != nil {
					log.Printf("receipt worker: create PUT request for tx %s: %v (continuing)", tx.ID, reqErr)
				} else {
					putReq.Header.Set("Content-Type", contentType)
					putReq.Header.Set("x-amz-acl", "public-read")
					putReq.ContentLength = int64(len(fileBytes))
					putResp, putErr := (&http.Client{Timeout: 60 * time.Second}).Do(putReq)
					if putErr != nil {
						log.Printf("receipt worker: upload to Spaces for tx %s: %v (continuing)", tx.ID, putErr)
					} else {
						putResp.Body.Close()
						if putResp.StatusCode >= 200 && putResp.StatusCode < 300 {
							receiptURL = photos.PublicURL(cfg.SpacesEndpoint, cfg.SpacesBucket, key)
						} else {
							log.Printf("receipt worker: Spaces PUT for tx %s returned %d (continuing)", tx.ID, putResp.StatusCode)
						}
					}
				}
			}
		}

		// Parse with Claude Haiku (via the parseReceipt seam).
		items, summary, err := parseReceipt(ctx, cfg.AnthropicAPIKey, fileBytes, contentType)
		if err != nil {
			log.Printf("receipt worker: ParseReceipt for tx %s: %v — routing to review queue", tx.ID, err)
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, "Receipt could not be parsed automatically", isUpgrade); routeErr != nil {
				log.Printf("receipt worker: routePending (parse-fail) for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		// Validate
		result := ValidateReceiptData(items, summary, tx.Amount)
		if !result.Valid {
			log.Printf("receipt worker: transaction %s routed to review queue: %s", tx.ID, result.Reason)
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, result.Reason, isUpgrade); routeErr != nil {
				log.Printf("receipt worker: routePending (validate-fail) for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		// Auto-create purchase event. isUpgrade=true causes the helper to
		// DELETE the stale pending row inside the same DB transaction as
		// the event INSERT — atomic upgrade with no window for a concurrent
		// re-sync to see both rows.
		if err := createPurchaseEvent(ctx, cfg.Pool, tx, items, summary, receiptURL, isUpgrade); err != nil {
			log.Printf("receipt worker: createPurchaseEvent for tx %s: %v — routing to review queue", tx.ID, err)
			if routeErr := routePending(ctx, cfg.Pool, tx, items, summary, receiptURL, "Receipt could not be saved automatically", isUpgrade); routeErr != nil {
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

// routePending dispatches the parse-fail / validate-fail / save-fail branches
// of runIngestCycle to either INSERT a new pending_purchases row (cold path)
// or UPDATE the existing one in place (upgrade path, isUpgrade=true). Keeps
// the call sites in runIngestCycle one-liners.
func routePending(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, reason string, isUpgrade bool) error {
	if isUpgrade {
		return updatePendingPurchase(ctx, pool, tx, items, summary, receiptURL, reason)
	}
	return insertPendingPurchase(ctx, pool, tx, items, summary, receiptURL, reason)
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
// Discarded pending rows return kind="none" — the user explicitly threw the
// row away, so the worker is free to re-process the same bank_tx_id.
func classifyExistingTx(ctx context.Context, pool *pgxpool.Pool, bankTxID string) (kind, reason string, err error) {
	err = pool.QueryRow(ctx, `
		SELECT 'event' AS kind, '' AS reason
		  FROM purchase_events WHERE bank_tx_id = $1
		UNION ALL
		SELECT 'event' AS kind, COALESCE(reason,'') AS reason
		  FROM pending_purchases
		 WHERE bank_tx_id = $1 AND confirmed_at IS NOT NULL
		UNION ALL
		SELECT 'pending' AS kind, COALESCE(reason,'') AS reason
		  FROM pending_purchases
		 WHERE bank_tx_id = $1
		   AND confirmed_at IS NULL
		   AND discarded_at IS NULL
		LIMIT 1`, bankTxID).Scan(&kind, &reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return "none", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("classifyExistingTx: %w", err)
	}
	return kind, reason, nil
}

// pickAttachment selects the best receipt attachment from a transaction.
// For transactions with multiple attachments, prefers PDF files.
func pickAttachment(tx MercuryTransaction) *Attachment {
	if len(tx.Attachments) == 1 {
		return &tx.Attachments[0]
	}
	// Multiple attachments — prefer PDF
	for i, att := range tx.Attachments {
		if strings.ToLower(filepath.Ext(att.FileName)) == ".pdf" {
			return &tx.Attachments[i]
		}
	}
	// Fallback: first attachment
	return &tx.Attachments[0]
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
func createPurchaseEvent(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, isUpgrade bool) error {
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
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, receipt_url, mercury_category)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		vendorID, tx.ID, eventDate, summary.Tax, summary.Total, nullableString(receiptURL), nullableString(mercuryCategory),
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

		_, err = dbTx.Exec(ctx,
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			eventID, nullableStringPtr(&itemID), itemName, item.Quantity, item.Price, item.IsCase,
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
func insertPendingPurchase(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, reason string) error {
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
		 (bank_tx_id, bank_total, vendor, event_date, tax, total, items, reason, receipt_url, mercury_category)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		nullableString(mercuryCategory),
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
func updatePendingPurchase(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string, reason string) error {
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
		        mercury_category = $10
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
		nullableString(mercuryCategory),
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
