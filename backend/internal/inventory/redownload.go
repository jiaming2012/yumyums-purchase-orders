package inventory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RedownloadRunner re-fetches one bank transaction from Mercury and re-runs
// the receipt pipeline on its current attachments. Returns a status string —
// see receipt.RedownloadFromMercury for the vocabulary. Production callers
// pass that function wrapped in a closure; tests inject a stub.
type RedownloadRunner func(ctx context.Context, bankTxID string, since time.Time) (string, error)

// redownloadSinceMargin is how far before the charge's own date the Mercury
// lookup window opens. Mercury lists by its createdAt, which can precede the
// posted event_date; a month is generous and cheap (one page for one id).
const redownloadSinceMargin = 30 * 24 * time.Hour

// RedownloadPendingHandler is "Re-download from source" on a pending card:
// go back to Mercury for THIS charge's attachments as they are now, store
// them over the old download, and parse them — one run, under the same
// single-flight slot as every other kind.
//
// Endpoint:  POST /api/v1/inventory/purchases/pending/{id}/redownload
// Response:  200 { "id", "sync_id", "started_at", "status": "running" }
//
//	404 pending_purchase_not_found — id does not exist
//	422 row_not_pending            — already confirmed or discarded
//	409 sync_already_running       — the single-flight slot is taken
//
// It is offered on rows WITHOUT a stored receipt too ("Missing Receipt"):
// that is precisely the row the operator has since attached a file to.
// If Mercury does not return the charge, or returns it with nothing attached,
// the run ends FAILED with a plain-words reason — nothing changed, and the
// operator must not read "done" as "fixed".
func RedownloadPendingHandler(pool *pgxpool.Pool, runner RedownloadRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id_required")
			return
		}

		var (
			confirmedAt sql.NullTime
			discardedAt sql.NullTime
			bankTxID    string
			eventDate   sql.NullTime
		)
		err := pool.QueryRow(r.Context(),
			`SELECT confirmed_at, discarded_at, bank_tx_id, event_date
			   FROM pending_purchases WHERE id = $1`, id,
		).Scan(&confirmedAt, &discardedAt, &bankTxID, &eventDate)
		if err != nil {
			writeError(w, http.StatusNotFound, "pending_purchase_not_found")
			return
		}
		if confirmedAt.Valid || discardedAt.Valid {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error":  "row_not_pending",
				"reason": "row is already confirmed or discarded",
			})
			return
		}
		since := time.Now().Add(-redownloadSinceMargin)
		if eventDate.Valid {
			since = eventDate.Time.Add(-redownloadSinceMargin)
		}

		var runID int64
		var startedAt time.Time
		err = pool.QueryRow(r.Context(),
			`INSERT INTO receipt_sync_runs (status, triggered_by)
			 VALUES ('running', 'redownload')
			 RETURNING id, started_at`,
		).Scan(&runID, &startedAt)
		if err != nil {
			if isUniqueViolation(err) {
				writeJSON(w, http.StatusConflict, map[string]any{"error": "sync_already_running"})
				return
			}
			slog.Info(fmt.Sprintf("RedownloadPending insert sync run: %v", err))
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		slog.Info("RedownloadPending: run started", "id", id, "bank_tx_id", bankTxID, "run_id", runID, "since", since.Format("2006-01-02"))
		go runRedownloadGoroutine(pool, runner, runID, bankTxID, since)

		writeJSON(w, http.StatusOK, map[string]any{
			"id":         id,
			"sync_id":    runID,
			"started_at": startedAt,
			"status":     "running",
		})
	}
}

// runRedownloadGoroutine mirrors runReprocessGoroutine for a single
// Mercury-sourced row: cancellable, panic-safe, and it writes the terminal
// row with a context the cancel cannot reach. The two "nothing to do"
// outcomes are reported as failures with the words the chip should show.
func runRedownloadGoroutine(pool *pgxpool.Pool, runner RedownloadRunner, id int64, bankTxID string, since time.Time) {
	ctx, release := registerSyncCancel(id)
	defer release()
	wctx := context.Background()
	defer func() {
		if rec := recover(); rec != nil {
			msg := "panic in redownload goroutine"
			finishSyncRun(wctx, pool, id, "failed", &msg, nil)
			slog.Info(fmt.Sprintf("RedownloadPending goroutine panic for run %d: %v", id, rec))
		}
	}()

	status, runErr := runner(ctx, bankTxID, since)
	if runErr != nil {
		if errors.Is(runErr, context.Canceled) {
			slog.Info(fmt.Sprintf("RedownloadPending: run %d cancelled", id))
			return
		}
		slog.Info(fmt.Sprintf("RedownloadPending: run %d error: %v", id, runErr))
		msg := runErr.Error()
		finishSyncRun(wctx, pool, id, "failed", &msg, nil)
		return
	}

	counts := &syncCounts{Processed: 1}
	switch status {
	case "auto_created":
		counts.AutoCreated = 1
	case "pending_review":
		counts.PendingReview = 1
	case "cached":
		// The charge is already a confirmed purchase; the residual pending
		// row was cleared. Nothing to count, nothing failed.
	case "missing_at_mercury":
		msg := "Mercury did not return this charge"
		finishSyncRun(wctx, pool, id, "failed", &msg, nil)
		return
	case "no_attachments":
		msg := "Mercury has no receipt attached to this charge"
		finishSyncRun(wctx, pool, id, "failed", &msg, nil)
		return
	default: // "errored" without an error value — say so rather than "done"
		msg := "receipt could not be re-read from Mercury"
		finishSyncRun(wctx, pool, id, "failed", &msg, nil)
		return
	}
	slog.Info(fmt.Sprintf("RedownloadPending: run %d done — %s", id, status))
	finishSyncRun(wctx, pool, id, "done", nil, counts)
}
