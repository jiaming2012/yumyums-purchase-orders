package inventory

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Cancellation for in-flight sync runs.
//
// Every sync path (manual, deep, reprocess-all) detaches its goroutine to
// context.Background() so it outlives the HTTP request that started it. That
// is what lets the UI survive a reload mid-run — and it is also why nothing
// could stop a run once started. A run is a receipt download plus a model call
// per row, so a queue of a dozen takes minutes; the operator watching
// "Syncing… 3m 10s" needs a way out.
//
// syncCancels holds one CancelFunc per running row id. It is deliberately
// process-local: it does NOT survive a restart, and it cannot reach a run
// started by another instance. Both are correct here — HQ runs as a single
// container, and a restart kills the goroutine anyway (the row is then
// reconciled by reapRunawaySyncRuns below, not left running forever).
var syncCancels = struct {
	sync.Mutex
	m map[int64]context.CancelFunc
}{m: make(map[int64]context.CancelFunc)}

// registerSyncCancel returns a context the run must use, plus a release func
// the caller MUST defer. Release removes the entry whether the run finished,
// failed, panicked or was cancelled — a stale CancelFunc keyed by a reused id
// would cancel an unrelated later run.
func registerSyncCancel(id int64) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	syncCancels.Lock()
	syncCancels.m[id] = cancel
	syncCancels.Unlock()
	return ctx, func() {
		syncCancels.Lock()
		delete(syncCancels.m, id)
		syncCancels.Unlock()
		cancel() // release context resources; a no-op if already cancelled
	}
}

// cancelSyncRun signals the goroutine for id. Reports whether a live
// CancelFunc was found — a row can be 'running' in the database with no
// goroutine behind it (the process restarted mid-run), and the caller marks
// the row terminal either way so the single-flight slot is released.
func cancelSyncRun(id int64) bool {
	syncCancels.Lock()
	cancel, ok := syncCancels.m[id]
	syncCancels.Unlock()
	if ok {
		cancel()
	}
	return ok
}

// CancelSyncHandler marks the in-flight run cancelled and signals its
// goroutine. POST /api/v1/inventory/sync-receipts/cancel.
//
// The UPDATE is the source of truth and is applied FIRST, guarded by
// status='running' so two racing cancels produce one transition. The
// goroutine's own terminal write is then suppressed by that same guard (see
// finishSyncRun) — whichever lands first wins, and the loser is a no-op rather
// than an overwrite. Counts are left as-is: receipts already re-parsed keep
// their results, so zeroing them would misreport what the run did.
func CancelSyncHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var id int64
		err := pool.QueryRow(r.Context(),
			`UPDATE receipt_sync_runs
			    SET status='cancelled', finished_at=now()
			  WHERE status='running'
			RETURNING id`,
		).Scan(&id)
		if err != nil {
			// No running row: nothing to cancel. Not an error the operator
			// caused — the run most likely finished between render and tap.
			writeJSON(w, http.StatusOK, map[string]any{"cancelled": false, "reason": "no run in flight"})
			return
		}
		live := cancelSyncRun(id)
		slog.Info("sync run cancelled", "run_id", id, "goroutine_signalled", live)
		writeJSON(w, http.StatusOK, map[string]any{"cancelled": true, "id": id, "goroutine_signalled": live})
	}
}

// finishSyncRun writes a terminal status, but ONLY over a row still marked
// running. Every sync goroutine ends through here so a cancel that already
// moved the row cannot be overwritten by the goroutine noticing a moment later
// and reporting 'failed: context canceled' — which would show the operator an
// error for something they chose.
func finishSyncRun(ctx context.Context, pool *pgxpool.Pool, id int64, status string, errMsg *string, counts *syncCounts) {
	q := `UPDATE receipt_sync_runs
	         SET status=$1, finished_at=now(), error=$2`
	args := []any{status, errMsg}
	if counts != nil {
		q += `, processed=$4, auto_created=$5, pending_review=$6, cached=$7`
	}
	q += ` WHERE id=$3 AND status='running'`
	args = append(args, id)
	if counts != nil {
		args = append(args, counts.Processed, counts.AutoCreated, counts.PendingReview, counts.Cached)
	}
	if _, err := pool.Exec(ctx, q, args...); err != nil {
		slog.Error("finishSyncRun update failed", "run_id", id, "status", status, "error", err)
	}
}

// syncCounts is the shape every run path reports; kept separate from
// receipt.IngestResult so reprocess (which has its own tallies) can use it too.
type syncCounts struct {
	Processed     int
	AutoCreated   int
	PendingReview int
	Cached        int
}
