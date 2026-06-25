package inventory

import (
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ReprocessAllPendingHandler resets every still-pending purchase row to the
// upgrade-eligible state (items=[], parse_error=NULL, reason='Receipt could
// not be parsed automatically') and triggers an immediate sync goroutine
// reusing the existing receipt_sync_runs single-flight machinery. This is
// the admin escape hatch for when backend parser/validator logic changes
// and legacy pending rows would otherwise stay stuck.
//
// Response (200 OK):
//
//	{ "reset_count": N, "sync_id": M, "started_at": "...", "status": "running" }
//
// Returns 409 if a sync is already running (the existing single-running index
// fires a unique-violation on INSERT).
func ReprocessAllPendingHandler(pool *pgxpool.Pool, runner IngestRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: bulk-update all still-pending rows to the upgrade-eligible
		// state so the next ingest cycle picks them up via the normal parseFailedRetry gate.
		tag, err := pool.Exec(r.Context(),
			`UPDATE pending_purchases
			   SET items = '[]'::jsonb,
			       parse_error = NULL,
			       reason = 'Receipt could not be parsed automatically'
			 WHERE confirmed_at IS NULL
			   AND discarded_at IS NULL`)
		if err != nil {
			log.Printf("ReprocessAllPending update: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		resetCount := tag.RowsAffected()

		// Step 2: kick off the sync goroutine reusing the existing single-flight
		// pattern from SyncReceiptsHandler. If a sync is already running the
		// partial unique index fires a 23505 unique violation → 409 conflict.
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
					"error":       "sync_already_running",
					"reset_count": resetCount,
				})
				return
			}
			log.Printf("ReprocessAllPending insert sync run: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Detach from the request context — the goroutine outlives the HTTP request.
		go runSyncGoroutine(pool, runner, id)

		writeJSON(w, http.StatusOK, map[string]any{
			"reset_count": resetCount,
			"sync_id":     id,
			"started_at":  startedAt,
			"status":      "running",
		})
	}
}
