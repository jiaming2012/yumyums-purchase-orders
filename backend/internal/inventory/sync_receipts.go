package inventory

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/receipt"
)

// syncRunRow mirrors a row in receipt_sync_runs for JSON marshaling.
//
// LookbackDays (260701-a23) is NOT stored in the DB — it's injected from
// receiptCfg at handler construction so the frontend can compute the
// "Synced from {Mon DD}" headline (started_at − lookback_days).
type syncRunRow struct {
	ID            int64      `json:"id"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	Status        string     `json:"status"`
	Processed     int        `json:"processed"`
	AutoCreated   int        `json:"auto_created"`
	PendingReview int        `json:"pending_review"`
	Cached        int        `json:"cached"`
	Error         *string    `json:"error"`
	TriggeredBy   string     `json:"triggered_by"`
	LookbackDays  int        `json:"lookback_days"`
}

// IngestRunner is the function the sync handler calls to actually run the
// receipt ingest cycle. In production this is a closure around
// receipt.RunIngestCycle + the server's receiptCfg. Tests inject a stub so
// they don't depend on Mercury or Anthropic.
type IngestRunner func(ctx context.Context) (receipt.IngestResult, error)

// SyncReceiptsHandler returns a handler that triggers an on-demand Mercury
// receipt ingest cycle.
//
//   - First POST → INSERT receipt_sync_runs (status='running') → spawn goroutine
//     → respond 200 {id,status:"running",started_at}.
//   - Concurrent POST while a run is in flight → 23505 unique violation on the
//     partial unique index → 409 {"error":"sync_already_running"}.
//
// The goroutine survives the request lifetime by detaching to context.Background;
// it writes the terminal status (done/failed) back to the row regardless of how
// it exits (success, error return, or panic — see runSyncGoroutine).
func SyncReceiptsHandler(pool *pgxpool.Pool, runner IngestRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var id int64
		var startedAt time.Time
		err := pool.QueryRow(r.Context(),
			`INSERT INTO receipt_sync_runs (status, triggered_by)
			 VALUES ('running', 'manual')
			 RETURNING id, started_at`,
		).Scan(&id, &startedAt)
		if err != nil {
			if isUniqueViolation(err) {
				writeError(w, http.StatusConflict, "sync_already_running")
				return
			}
			slog.Error("SyncReceipts insert failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Detach the request context — the goroutine outlives the HTTP request.
		go runSyncGoroutine(pool, runner, id)

		writeJSON(w, http.StatusOK, map[string]any{
			"id":         id,
			"status":     "running",
			"started_at": startedAt,
		})
	}
}

// runSyncGoroutine runs the ingest runner and writes the terminal status to the
// receipt_sync_runs row identified by id. defer recover() guarantees no
// orphan running rows are left behind on panic.
func runSyncGoroutine(pool *pgxpool.Pool, runner IngestRunner, id int64) {
	ctx := context.Background()
	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprintf("panic: %v", rec)
			_, _ = pool.Exec(ctx,
				`UPDATE receipt_sync_runs
				 SET status='failed', finished_at=now(), error=$1
				 WHERE id=$2`, msg, id)
			slog.Error("SyncReceipts goroutine panic", "run_id", id, "panic", rec)
		}
	}()

	result, err := runner(ctx)
	if err != nil {
		_, updErr := pool.Exec(ctx,
			`UPDATE receipt_sync_runs
			 SET status='failed', finished_at=now(), error=$1
			 WHERE id=$2`, err.Error(), id)
		if updErr != nil {
			slog.Error("SyncReceipts failed-update", "run_id", id, "error", updErr)
		}
		return
	}
	_, updErr := pool.Exec(ctx,
		`UPDATE receipt_sync_runs
		 SET status='done', finished_at=now(),
		     processed=$1, auto_created=$2, pending_review=$3, cached=$4
		 WHERE id=$5`,
		result.Processed, result.AutoCreated, result.PendingReview, result.Cached, id)
	if updErr != nil {
		slog.Error("SyncReceipts done-update failed", "run_id", id, "error", updErr)
	}
}

// SyncReceiptsStatusHandler returns the latest receipt_sync_runs row as JSON,
// or `null` (with 200) when the table is empty. The frontend polls this on a
// 3s timer while the latest row is status='running'.
//
// lookbackDays is injected into the row's LookbackDays field (not a DB
// column — see syncRunRow). The frontend uses it to compute the sync-window
// start date for the "Synced from {Mon DD}" chip copy (260701-a23).
func SyncReceiptsStatusHandler(pool *pgxpool.Pool, lookbackDays int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var row syncRunRow
		err := pool.QueryRow(r.Context(),
			`SELECT id, started_at, finished_at, status, processed,
			        auto_created, pending_review, cached, error, triggered_by
			 FROM receipt_sync_runs
			 ORDER BY started_at DESC
			 LIMIT 1`,
		).Scan(&row.ID, &row.StartedAt, &row.FinishedAt, &row.Status,
			&row.Processed, &row.AutoCreated, &row.PendingReview, &row.Cached,
			&row.Error, &row.TriggeredBy)
		if errors.Is(err, pgx.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("null"))
			return
		}
		if err != nil {
			slog.Error("SyncReceiptsStatus query failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		row.LookbackDays = lookbackDays
		writeJSON(w, http.StatusOK, row)
	}
}

// isUniqueViolation reports whether err is a Postgres unique constraint
// violation (SQLSTATE 23505). Used to detect the partial-unique-index
// collision that signals "another sync run is already running".
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return true
	}
	return false
}
