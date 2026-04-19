package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/photos"
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
		if err := runIngestCycle(ctx, cfg); err != nil {
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
				if err := runIngestCycle(ctx, cfg); err != nil {
					log.Printf("receipt worker: ingest cycle error: %v", err)
				}
			}
		}
	}()
}

// runIngestCycle executes one full Mercury → parse → validate → persist cycle.
func runIngestCycle(ctx context.Context, cfg WorkerConfig) error {
	lookback := cfg.LookbackDays
	if lookback <= 0 {
		lookback = 14
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -lookback)

	txns, err := FetchTransactions(ctx, cfg.MercuryAPIKey, startDate, endDate)
	if err != nil {
		return fmt.Errorf("runIngestCycle: FetchTransactions: %w", err)
	}

	if len(txns) == 0 {
		log.Println("receipt worker: no transactions with attachments found")
		return nil
	}

	var autoCreated, pendingReview int

	for _, tx := range txns {
		if len(tx.Attachments) == 0 {
			continue
		}

		// Idempotency: skip if already in purchase_events or pending_purchases
		already, err := bankTxIDExists(ctx, cfg.Pool, tx.ID)
		if err != nil {
			log.Printf("receipt worker: check existing tx %s: %v", tx.ID, err)
			continue
		}
		if already {
			continue
		}

		// Choose best attachment — prefer PDF for known multi-attachment vendors
		attachment := pickAttachment(tx)
		if attachment == nil {
			log.Printf("receipt worker: transaction %s has attachments but none selected — skipping", tx.ID)
			continue
		}

		// Download receipt file
		fileBytes, contentType, err := downloadReceiptFile(ctx, attachment.URL)
		if err != nil {
			log.Printf("receipt worker: download attachment for tx %s: %v", tx.ID, err)
			continue
		}

		// Optionally upload original to DO Spaces
		receiptURL := attachment.URL
		if cfg.SpacesPresigner != "" && cfg.SpacesBucket != "" {
			ext := strings.ToLower(filepath.Ext(attachment.FileName))
			if ext == "" {
				ext = ".jpg"
			}
			key := fmt.Sprintf("receipts/%s/original%s", tx.ID, ext)
			uploaded, uploadErr := photos.GeneratePresignedPutURL(ctx, nil, cfg.SpacesBucket, key, contentType, 15*time.Minute)
			if uploadErr != nil {
				log.Printf("receipt worker: presign for tx %s: %v (continuing)", tx.ID, uploadErr)
			} else {
				receiptURL = photos.PublicURL(cfg.SpacesPresigner, cfg.SpacesBucket, key)
				_ = uploaded // presigned PUT URL not used here; worker uploads directly
			}
		}

		// Parse with Claude Haiku
		items, summary, err := ParseReceipt(ctx, cfg.AnthropicAPIKey, fileBytes, contentType)
		if err != nil {
			log.Printf("receipt worker: ParseReceipt for tx %s: %v — routing to review queue", tx.ID, err)
			if routeErr := insertPendingPurchase(ctx, cfg.Pool, tx, items, summary, receiptURL, err.Error()); routeErr != nil {
				log.Printf("receipt worker: insertPendingPurchase for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		// Validate
		result := ValidateReceiptData(items, summary, tx.Amount)
		if !result.Valid {
			log.Printf("receipt worker: transaction %s routed to review queue: %s", tx.ID, result.Reason)
			if routeErr := insertPendingPurchase(ctx, cfg.Pool, tx, items, summary, receiptURL, result.Reason); routeErr != nil {
				log.Printf("receipt worker: insertPendingPurchase for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		// Auto-create purchase event
		if err := createPurchaseEvent(ctx, cfg.Pool, tx, items, summary, receiptURL); err != nil {
			log.Printf("receipt worker: createPurchaseEvent for tx %s: %v — routing to review queue", tx.ID, err)
			if routeErr := insertPendingPurchase(ctx, cfg.Pool, tx, items, summary, receiptURL, err.Error()); routeErr != nil {
				log.Printf("receipt worker: insertPendingPurchase for tx %s: %v", tx.ID, routeErr)
			}
			pendingReview++
			continue
		}

		autoCreated++
	}

	log.Printf("receipt worker: processed %d transactions, %d auto-created, %d pending review",
		len(txns), autoCreated, pendingReview)
	return nil
}

// bankTxIDExists returns true if the bank_tx_id already exists in either
// purchase_events or pending_purchases (idempotency guard).
func bankTxIDExists(ctx context.Context, pool *pgxpool.Pool, bankTxID string) (bool, error) {
	var n int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM (
			SELECT 1 FROM purchase_events WHERE bank_tx_id = $1
			UNION ALL
			SELECT 1 FROM pending_purchases WHERE bank_tx_id = $1
		) sub`,
		bankTxID,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("bankTxIDExists: %w", err)
	}
	return n > 0, nil
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
func createPurchaseEvent(ctx context.Context, pool *pgxpool.Pool, tx MercuryTransaction, items []ReceiptItem, summary ReceiptSummary, receiptURL string) error {
	dbTx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("createPurchaseEvent: begin: %w", err)
	}
	defer dbTx.Rollback(ctx) //nolint:errcheck

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

	// Insert purchase_event
	var eventID string
	err = dbTx.QueryRow(ctx,
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, receipt_url)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		vendorID, tx.ID, eventDate, summary.Tax, summary.Total, nullableString(receiptURL),
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
	itemsJSON, err := json.Marshal(items)
	if err != nil {
		itemsJSON = []byte("[]")
	}

	eventDate := parseEventDate(tx.CreatedAt)

	_, err = pool.Exec(ctx,
		`INSERT INTO pending_purchases
		 (bank_tx_id, bank_total, vendor, event_date, tax, total, items, reason, receipt_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT DO NOTHING`,
		tx.ID,
		tx.Amount,
		summary.Vendor,
		nullableString(eventDate),
		nullableFloat64(summary.Tax),
		nullableFloat64(summary.Total),
		itemsJSON,
		nullableString(reason),
		nullableString(receiptURL),
	)
	if err != nil {
		return fmt.Errorf("insertPendingPurchase: %w", err)
	}
	return nil
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
