package inventory

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/receipt"
)

// BatchReprocessRunner is a function that reprocesses all still-pending rows
// using their stored receipt URLs (from DO Spaces) without any Mercury API
// calls. Returns a map of bank_tx_id -> status where status is one of
// "auto_created", "pending_review", "no_attachments", or "errored".
// Production callers pass receipt.BatchReprocessFromSpaces wrapped in a
// closure; tests inject a stub.
type BatchReprocessRunner func(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error)

// ReprocessAllPendingHandler reprocesses every still-pending purchase row by
// downloading their stored receipt attachments directly from Spaces and running
// the full parse/validate/persist pipeline. No Mercury API calls are made.
//
// Approach:
//  1. Single-flight via receipt_sync_runs (same as SyncReceiptsHandler).
//  2. SELECT full row data (bank_tx_id, bank_total, vendor, event_date, URLs)
//     FROM pending_purchases WHERE still-active AND receipt_url IS NOT NULL.
//     COALESCE(receipt_urls, jsonb_build_array(receipt_url)) unifies legacy
//     single-URL rows and new multi-URL rows.
//  3. Filter rows with no URLs (can't reprocess from storage).
//  4. Call the BatchReprocessRunner with the row structs.
//  5. Aggregate the result map into counts and write the terminal tally to
//     receipt_sync_runs.
//
// Response (200 OK):
//
//	{ "pending_count": N, "sync_id": M, "started_at": "...", "status": "running" }
//
// Returns 409 if a sync is already running (the existing single-running index
// fires a unique-violation on INSERT).
func ReprocessAllPendingHandler(pool *pgxpool.Pool, runner BatchReprocessRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: collect all still-pending rows that have at least one receipt URL.
		// COALESCE unifies legacy single-URL rows (receipt_url column) and new
		// multi-URL rows (receipt_urls JSONB column). Rows where both are NULL are
		// filtered out by the WHERE clause — they have no stored attachment to
		// reprocess from.
		dbRows, err := pool.Query(r.Context(),
			`SELECT bank_tx_id,
			        bank_total,
			        vendor,
			        COALESCE(event_date::text, ''),
			        COALESCE(receipt_urls, jsonb_build_array(receipt_url))::text
			   FROM pending_purchases
			  WHERE confirmed_at IS NULL
			    AND discarded_at IS NULL
			    AND receipt_url IS NOT NULL`)
		if err != nil {
			slog.Info(fmt.Sprintf("ReprocessAllPending query pending: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		var pendingRows []receipt.PendingRowForReprocess
		for dbRows.Next() {
			var (
				bankTxID    string
				bankTotal   float64
				vendor      string
				eventDate   string
				urlsJSON    string
			)
			if scanErr := dbRows.Scan(&bankTxID, &bankTotal, &vendor, &eventDate, &urlsJSON); scanErr != nil {
				dbRows.Close()
				slog.Info(fmt.Sprintf("ReprocessAllPending scan: %v", scanErr))
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			var urls []string
			if jsonErr := json.Unmarshal([]byte(urlsJSON), &urls); jsonErr != nil {
				slog.Info(fmt.Sprintf("ReprocessAllPending parse receipt_urls for tx %s: %v (skipping)", bankTxID, jsonErr))
				continue
			}
			// Filter rows where the URL list came out empty after COALESCE.
			if len(urls) == 0 {
				continue
			}
			pendingRows = append(pendingRows, receipt.PendingRowForReprocess{
				BankTxID:    bankTxID,
				BankTotal:   bankTotal,
				Vendor:      vendor,
				EventDate:   eventDate,
				ReceiptURLs: urls,
			})
		}
		dbRows.Close()
		if err := dbRows.Err(); err != nil {
			slog.Info(fmt.Sprintf("ReprocessAllPending rows error: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		pendingCount := len(pendingRows)

		// Step 2: claim a single-flight slot via receipt_sync_runs. The partial
		// unique index on status='running' fires a 23505 unique violation if another
		// sync is already in progress → 409.
		var id int64
		var startedAt time.Time
		err = pool.QueryRow(r.Context(),
			`INSERT INTO receipt_sync_runs (status, triggered_by)
			 VALUES ('running', 'reprocess_all')
			 RETURNING id, started_at`,
		).Scan(&id, &startedAt)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error":         "sync_already_running",
					"pending_count": pendingCount,
				})
				return
			}
			slog.Info(fmt.Sprintf("ReprocessAllPending insert sync run: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Step 3: spawn the batch reprocess goroutine (detached from the request context).
		go runReprocessGoroutine(pool, runner, id, pendingRows)

		writeJSON(w, http.StatusOK, map[string]any{
			"pending_count": pendingCount,
			"sync_id":       id,
			"started_at":    startedAt,
			"status":        "running",
		})
	}
}

// runReprocessGoroutine calls the batch runner with all pending rows, aggregates
// the result map into counts, and writes the final tally to receipt_sync_runs.
// recover() guarantees no orphan running rows.
func runReprocessGoroutine(pool *pgxpool.Pool, runner BatchReprocessRunner, id int64, rows []receipt.PendingRowForReprocess) {
	ctx := context.Background()
	defer func() {
		if rec := recover(); rec != nil {
			msg := "panic in reprocess goroutine"
			_, _ = pool.Exec(ctx,
				`UPDATE receipt_sync_runs
				 SET status='failed', finished_at=now(), error=$1
				 WHERE id=$2`, msg, id)
			slog.Info(fmt.Sprintf("ReprocessAllPending goroutine panic for run %d: %v", id, rec))
		}
	}()

	results, runErr := runner(ctx, rows)
	if runErr != nil {
		slog.Info(fmt.Sprintf("ReprocessAllPending: batch runner error for run %d: %v", id, runErr))
		_, _ = pool.Exec(ctx,
			`UPDATE receipt_sync_runs
			 SET status='failed', finished_at=now(), error=$1
			 WHERE id=$2`, runErr.Error(), id)
		return
	}

	var autoCreated, pendingReview, noAttachments, errored int
	for _, result := range results {
		switch result {
		case "auto_created":
			autoCreated++
		case "pending_review":
			pendingReview++
		case "no_attachments":
			noAttachments++
		case "errored":
			errored++
		}
	}

	slog.Info(fmt.Sprintf("ReprocessAllPending: run %d done — auto_created=%d pending_review=%d no_attachments=%d errored=%d",
		id, autoCreated, pendingReview, noAttachments, errored))

	_, updErr := pool.Exec(ctx,
		`UPDATE receipt_sync_runs
		 SET status='done', finished_at=now(),
		     processed=$1, auto_created=$2, pending_review=$3, cached=$4
		 WHERE id=$5`,
		len(rows), autoCreated, pendingReview, errored, id)
	if updErr != nil {
		slog.Info(fmt.Sprintf("ReprocessAllPending done-update for run %d: %v", id, updErr))
	}
}
