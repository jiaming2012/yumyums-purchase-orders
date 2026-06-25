package inventory

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PerRowReprocessRunner is a function that fetches a single Mercury transaction
// by bank_tx_id and runs the full ingest pipeline for it. It returns a result
// string ("auto_created" | "pending_review" | "cached" | "errored") and any
// non-fatal error. Production callers pass receipt.ProcessSingleTxByID wrapped
// in a closure; tests inject a stub.
type PerRowReprocessRunner func(ctx context.Context, bankTxID string) (string, error)

// ReprocessAllPendingHandler reprocesses every still-pending purchase row by
// fetching each row's Mercury transaction individually (bypassing the list
// endpoint's 14-day lookback limit) and running the full ingest pipeline.
//
// Per-row approach (fixes "lookback amnesia"):
//  1. Single-flight via receipt_sync_runs (same as SyncReceiptsHandler).
//  2. SELECT bank_tx_id FROM pending_purchases WHERE still-active.
//  3. For each bank_tx_id call the PerRowReprocessRunner which:
//     a. Hits Mercury GET /transaction/{id} for the latest state.
//     b. Runs the full classify → parse → validate → persist pipeline.
//  4. Aggregate counts and write the terminal tally to receipt_sync_runs.
//
// Response (200 OK):
//
//	{ "pending_count": N, "sync_id": M, "started_at": "...", "status": "running",
//	  "auto_created": A, "pending_review": P, "errored": E }
//
// Returns 409 if a sync is already running (the existing single-running index
// fires a unique-violation on INSERT).
func ReprocessAllPendingHandler(pool *pgxpool.Pool, runner PerRowReprocessRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Step 1: collect all still-pending bank_tx_ids.
		rows, err := pool.Query(r.Context(),
			`SELECT bank_tx_id FROM pending_purchases
			  WHERE confirmed_at IS NULL
			    AND discarded_at IS NULL`)
		if err != nil {
			log.Printf("ReprocessAllPending query pending: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		var bankTxIDs []string
		for rows.Next() {
			var id string
			if scanErr := rows.Scan(&id); scanErr != nil {
				rows.Close()
				log.Printf("ReprocessAllPending scan: %v", scanErr)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			bankTxIDs = append(bankTxIDs, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			log.Printf("ReprocessAllPending rows error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		pendingCount := len(bankTxIDs)

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
			log.Printf("ReprocessAllPending insert sync run: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Step 3: spawn the per-row reprocess goroutine (detached from the request context).
		go runReprocessGoroutine(pool, runner, id, bankTxIDs)

		writeJSON(w, http.StatusOK, map[string]any{
			"pending_count": pendingCount,
			"sync_id":       id,
			"started_at":    startedAt,
			"status":        "running",
		})
	}
}

// runReprocessGoroutine processes each bank_tx_id via the runner and writes the
// final tally to receipt_sync_runs. recover() guarantees no orphan running rows.
func runReprocessGoroutine(pool *pgxpool.Pool, runner PerRowReprocessRunner, id int64, bankTxIDs []string) {
	ctx := context.Background()
	defer func() {
		if rec := recover(); rec != nil {
			msg := "panic in reprocess goroutine"
			_, _ = pool.Exec(ctx,
				`UPDATE receipt_sync_runs
				 SET status='failed', finished_at=now(), error=$1
				 WHERE id=$2`, msg, id)
			log.Printf("ReprocessAllPending goroutine panic for run %d: %v", id, rec)
		}
	}()

	var autoCreated, pendingReview, errored int
	for _, bankTxID := range bankTxIDs {
		result, runErr := runner(ctx, bankTxID)
		if runErr != nil {
			log.Printf("ReprocessAllPending: runner error for tx %s: %v", bankTxID, runErr)
		}
		switch result {
		case "auto_created":
			autoCreated++
		case "pending_review":
			pendingReview++
		case "errored":
			errored++
		}
	}

	log.Printf("ReprocessAllPending: run %d done — auto_created=%d pending_review=%d errored=%d",
		id, autoCreated, pendingReview, errored)

	_, updErr := pool.Exec(ctx,
		`UPDATE receipt_sync_runs
		 SET status='done', finished_at=now(),
		     processed=$1, auto_created=$2, pending_review=$3, cached=$4
		 WHERE id=$5`,
		len(bankTxIDs), autoCreated, pendingReview, errored, id)
	if updErr != nil {
		log.Printf("ReprocessAllPending done-update for run %d: %v", id, updErr)
	}
}
