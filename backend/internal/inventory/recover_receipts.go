package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/receipt"
)

// RecoverRunner re-fetches dead-URL receipts from Mercury and re-uploads them
// to the current bucket. Production callers pass receipt.RecoverDeadReceiptURLs
// wrapped in a closure; tests inject a stub.
type RecoverRunner func(ctx context.Context, dead receipt.DeadRows) (receipt.RecoverResult, error)

// RecoverReceiptsHandler drives the B-172 recovery of receipt URLs stranded on
// a dead storage host (canceled DO Spaces account, expiring Mercury-fallback
// URLs). Finder semantics live in receipt.FindDeadReceiptRows: any stored URL
// off the current {endpoint}/{bucket}/ prefix counts as dead.
//
// Request body (optional; empty body = real run with no cap):
//
//	{ "dry_run": true|false, "limit": 0 }
//
// Dry-run responds synchronously with the finder inventory and touches
// nothing. A real run claims the receipt_sync_runs single-flight slot (409
// sync_already_running if another sync holds it) and recovers in a detached
// goroutine. Terminal counts land in receipt_sync_runs reusing the existing
// columns: processed = tx ids examined, auto_created = recovered,
// pending_review = missing at Mercury, cached = failed — visible via
// GET /inventory/sync-receipts/status.
func RecoverReceiptsHandler(pool *pgxpool.Pool, storagePrefix string, runner RecoverRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if storagePrefix == "" {
			writeError(w, http.StatusServiceUnavailable, "storage_unconfigured")
			return
		}

		var req struct {
			DryRun bool `json:"dry_run"`
			Limit  int  `json:"limit"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}

		dead, err := receipt.FindDeadReceiptRows(r.Context(), pool, storagePrefix, req.Limit)
		if err != nil {
			slog.Info(fmt.Sprintf("RecoverReceipts finder: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if req.DryRun {
			writeJSON(w, http.StatusOK, map[string]any{
				"dry_run":                true,
				"purchase_events_rows":   dead.EventRows,
				"pending_purchases_rows": dead.PendingRows,
				"distinct_tx_ids":        len(dead.TxIDs),
				"since":                  dead.Since.Format("2006-01-02"),
				"tx_ids":                 dead.TxIDs,
			})
			return
		}

		var id int64
		var startedAt time.Time
		err = pool.QueryRow(r.Context(),
			`INSERT INTO receipt_sync_runs (status, triggered_by)
			 VALUES ('running', 'recover_receipts')
			 RETURNING id, started_at`,
		).Scan(&id, &startedAt)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error":           "sync_already_running",
					"distinct_tx_ids": len(dead.TxIDs),
				})
				return
			}
			slog.Info(fmt.Sprintf("RecoverReceipts insert sync run: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		go runRecoverGoroutine(pool, runner, id, dead)

		writeJSON(w, http.StatusOK, map[string]any{
			"sync_id":         id,
			"started_at":      startedAt,
			"status":          "running",
			"distinct_tx_ids": len(dead.TxIDs),
		})
	}
}

// runRecoverGoroutine calls the runner, logs the full per-tx detail, and
// writes the terminal tally to receipt_sync_runs (column reuse documented on
// RecoverReceiptsHandler). recover() guarantees no orphan running rows.
func runRecoverGoroutine(pool *pgxpool.Pool, runner RecoverRunner, id int64, dead receipt.DeadRows) {
	ctx := context.Background()
	defer func() {
		if rec := recover(); rec != nil {
			msg := "panic in recover-receipts goroutine"
			_, _ = pool.Exec(ctx,
				`UPDATE receipt_sync_runs
				 SET status='failed', finished_at=now(), error=$1
				 WHERE id=$2`, msg, id)
			slog.Info(fmt.Sprintf("RecoverReceipts goroutine panic for run %d: %v", id, rec))
		}
	}()

	result, runErr := runner(ctx, dead)
	if runErr != nil {
		slog.Info(fmt.Sprintf("RecoverReceipts: runner error for run %d: %v", id, runErr))
		_, _ = pool.Exec(ctx,
			`UPDATE receipt_sync_runs
			 SET status='failed', finished_at=now(), error=$1
			 WHERE id=$2`, runErr.Error(), id)
		return
	}

	slog.Info(fmt.Sprintf("RecoverReceipts: run %d done — examined=%d recovered=%d missing_at_mercury=%v failed=%v",
		id, result.Examined, result.Recovered, result.MissingTxIDs, result.FailedTxIDs))

	_, updErr := pool.Exec(ctx,
		`UPDATE receipt_sync_runs
		 SET status='done', finished_at=now(),
		     processed=$1, auto_created=$2, pending_review=$3, cached=$4
		 WHERE id=$5`,
		result.Examined, result.Recovered, result.MissingAtMercury, result.Failed, id)
	if updErr != nil {
		slog.Info(fmt.Sprintf("RecoverReceipts done-update for run %d: %v", id, updErr))
	}
}
