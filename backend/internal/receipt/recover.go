package receipt

// One-time-style recovery for receipt URLs stranded on a dead storage host
// (B-172: the DO Spaces account was canceled; rows also accumulated expiring
// Mercury-fallback URLs while the x-amz-acl bug broke uploads). Mercury still
// holds the original attachments, so recovery re-downloads them per bank
// transaction and re-uploads to the current bucket, rewriting the stored URLs.
//
// "Dead" is structural, not host-enumerated: any stored URL that does NOT
// start with the current public prefix ({STORAGE_ENDPOINT}/{STORAGE_BUCKET}/)
// is foreign — DO Spaces, Mercury CDN, or any past provider — and gets
// rewritten. Rewritten rows carry the current prefix and stop matching, which
// makes re-runs resume-safe no-ops.

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yumyums/hq/internal/photos"
)

// deadURLPredicate matches rows whose receipt_url or any receipt_urls element
// is non-empty and not on the current storage prefix ($N is the prefix param
// index). Kept as a single fragment so the finder SELECTs and the rewrite
// UPDATEs cannot drift apart.
func deadURLPredicate(prefixParam int) string {
	return fmt.Sprintf(`((receipt_url IS NOT NULL AND receipt_url <> '' AND NOT starts_with(receipt_url, $%d))
	    OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(receipt_urls) u(url)
	               WHERE u.url <> '' AND NOT starts_with(u.url, $%d)))`, prefixParam, prefixParam)
}

// StoragePublicPrefix returns the public-URL prefix objects on the configured
// bucket start with ("{endpoint}/{bucket}/"), or "" when storage is not
// configured.
func StoragePublicPrefix(cfg WorkerConfig) string {
	if cfg.SpacesEndpoint == "" || cfg.SpacesBucket == "" {
		return ""
	}
	return photos.PublicURL(cfg.SpacesEndpoint, cfg.SpacesBucket, "")
}

// DeadRows is the finder's inventory of rows needing URL recovery.
type DeadRows struct {
	TxIDs       []string  // distinct bank_tx_ids across both tables (capped by limit)
	EventRows   int       // matching purchase_events rows
	PendingRows int       // matching, undiscarded pending_purchases rows
	Since       time.Time // earliest affected event_date minus buffer — Mercury fetch window start
}

// FindDeadReceiptRows inventories purchase_events and pending_purchases rows
// whose stored receipt URLs are off-prefix (see deadURLPredicate). Discarded
// pending rows are skipped — nothing renders their receipts. limit > 0 caps
// the number of distinct tx ids returned (for a cautious first run); row
// counts always reflect the full match set.
func FindDeadReceiptRows(ctx context.Context, pool *pgxpool.Pool, storagePrefix string, limit int) (DeadRows, error) {
	if storagePrefix == "" {
		return DeadRows{}, fmt.Errorf("FindDeadReceiptRows: storage prefix is empty (storage unconfigured)")
	}

	var dead DeadRows
	seen := map[string]bool{}
	var minDate time.Time

	collect := func(query string, rowCount *int) error {
		rows, err := pool.Query(ctx, query, storagePrefix)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var txID string
			var eventDate *time.Time
			if err := rows.Scan(&txID, &eventDate); err != nil {
				return err
			}
			*rowCount++
			if eventDate != nil && (minDate.IsZero() || eventDate.Before(minDate)) {
				minDate = *eventDate
			}
			if txID != "" && !seen[txID] {
				seen[txID] = true
				if limit <= 0 || len(dead.TxIDs) < limit {
					dead.TxIDs = append(dead.TxIDs, txID)
				}
			}
		}
		return rows.Err()
	}

	if err := collect(
		`SELECT bank_tx_id, event_date FROM purchase_events WHERE `+deadURLPredicate(1)+` ORDER BY event_date`,
		&dead.EventRows,
	); err != nil {
		return DeadRows{}, fmt.Errorf("FindDeadReceiptRows: purchase_events: %w", err)
	}
	if err := collect(
		`SELECT bank_tx_id, event_date FROM pending_purchases WHERE discarded_at IS NULL AND `+deadURLPredicate(1)+` ORDER BY event_date NULLS LAST`,
		&dead.PendingRows,
	); err != nil {
		return DeadRows{}, fmt.Errorf("FindDeadReceiptRows: pending_purchases: %w", err)
	}

	// Mercury fetch window: earliest affected event date minus a buffer for
	// posting-date vs receipt-date skew. If no matched row carried a date,
	// fall back to a 2-year lookback rather than silently narrowing.
	if minDate.IsZero() {
		dead.Since = time.Now().AddDate(-2, 0, 0)
	} else {
		dead.Since = minDate.AddDate(0, 0, -30)
	}
	return dead, nil
}

// RecoverResult tallies one recovery run. MissingAtMercury counts tx ids
// Mercury no longer returns (or returns with zero attachments) — their rows
// are left untouched as evidence a receipt once existed.
type RecoverResult struct {
	Examined         int
	Recovered        int
	MissingAtMercury int
	Failed           int
	MissingTxIDs     []string
	FailedTxIDs      []string
}

// RecoverDeadReceiptURLs re-fetches each dead tx's attachments from Mercury,
// re-uploads them to the configured bucket under the standard
// receipts/{tx_id}/{i}{ext} keys, and rewrites receipt_url + receipt_urls on
// both tables. Per-tx atomic: any download or upload failure leaves that tx's
// rows untouched (counted Failed); same-key re-uploads are harmless
// overwrites, so re-running after a partial failure just resumes.
func RecoverDeadReceiptURLs(ctx context.Context, cfg WorkerConfig, dead DeadRows) (RecoverResult, error) {
	var res RecoverResult
	if cfg.SpacesPresigner == nil || cfg.SpacesBucket == "" || cfg.SpacesEndpoint == "" {
		return res, fmt.Errorf("RecoverDeadReceiptURLs: object storage not configured")
	}
	if cfg.MercuryAPIKey == "" {
		return res, fmt.Errorf("RecoverDeadReceiptURLs: MERCURY_API_KEY not set")
	}
	if len(dead.TxIDs) == 0 {
		return res, nil
	}
	prefix := StoragePublicPrefix(cfg)

	txMap, err := FetchTransactionsByIDs(ctx, cfg.MercuryAPIKey, dead.TxIDs, dead.Since, time.Now())
	if err != nil {
		return res, fmt.Errorf("RecoverDeadReceiptURLs: %w", err)
	}

	for _, txID := range dead.TxIDs {
		res.Examined++
		tx, ok := txMap[txID]
		if !ok || len(tx.Attachments) == 0 {
			res.MissingAtMercury++
			res.MissingTxIDs = append(res.MissingTxIDs, txID)
			continue
		}

		urls, txErr := recoverOneTx(ctx, cfg, tx)
		if txErr != nil {
			slog.Info(fmt.Sprintf("receipt recover: tx %s: %v (rows left untouched)", txID, txErr))
			res.Failed++
			res.FailedTxIDs = append(res.FailedTxIDs, txID)
			continue
		}

		if updErr := rewriteReceiptURLs(ctx, cfg.Pool, txID, urls, prefix); updErr != nil {
			slog.Info(fmt.Sprintf("receipt recover: rewrite URLs for tx %s: %v", txID, updErr))
			res.Failed++
			res.FailedTxIDs = append(res.FailedTxIDs, txID)
			continue
		}
		res.Recovered++
	}
	return res, nil
}

// recoverOneTx downloads every attachment for tx and uploads each to its slot
// key, returning the new public URLs in slot order. All-or-nothing: the first
// failure aborts so the caller never rewrites a row with a partial URL list
// (also avoids indexing attachments by a skip-compacted blob slice).
func recoverOneTx(ctx context.Context, cfg WorkerConfig, tx MercuryTransaction) ([]string, error) {
	urls := make([]string, 0, len(tx.Attachments))
	for i, att := range tx.Attachments {
		raw, contentType, err := downloadReceiptFileFn(ctx, att.URL)
		if err != nil {
			return nil, fmt.Errorf("download attachment %d: %w", i, err)
		}
		publicURL, err := uploadReceiptSlotFn(ctx, cfg, tx.ID, i, FileBlob{Bytes: raw, ContentType: contentType}, att.FileName)
		if err != nil {
			return nil, fmt.Errorf("upload slot %d: %w", i, err)
		}
		urls = append(urls, publicURL)
	}
	return urls, nil
}

// rewriteReceiptURLs points both tables' receipt columns at the recovered
// URLs. The dead-URL predicate is repeated in the WHERE clause so already-
// rewritten rows (and rows another run just fixed) are never touched twice.
func rewriteReceiptURLs(ctx context.Context, pool *pgxpool.Pool, txID string, urls []string, prefix string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no URLs to write")
	}
	if _, err := pool.Exec(ctx,
		`UPDATE purchase_events SET receipt_url = $2, receipt_urls = $3
		  WHERE bank_tx_id = $1 AND `+deadURLPredicate(4),
		txID, urls[0], receiptURLsJSON(urls), prefix,
	); err != nil {
		return fmt.Errorf("purchase_events: %w", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE pending_purchases SET receipt_url = $2, receipt_urls = $3
		  WHERE bank_tx_id = $1 AND discarded_at IS NULL AND `+deadURLPredicate(4),
		txID, urls[0], receiptURLsJSON(urls), prefix,
	); err != nil {
		return fmt.Errorf("pending_purchases: %w", err)
	}
	return nil
}
