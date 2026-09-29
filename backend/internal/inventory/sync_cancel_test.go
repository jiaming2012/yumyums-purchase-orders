package inventory

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// runningRunID clears any leftover runs and inserts a fresh running row,
// standing in for a sync started by any of the three paths. The clear is not
// optional: receipt_sync_runs is not in resetFixtures' truncate list, and the
// single-running partial unique index means one stray row from an earlier test
// makes the next one see a sync that is not its own.
func runningRunID(t *testing.T, triggeredBy string) int64 {
	t.Helper()
	clearSyncRuns(t)
	var id int64
	if err := testPool.QueryRow(context.Background(),
		`INSERT INTO receipt_sync_runs (status, triggered_by) VALUES ('running', $1) RETURNING id`,
		triggeredBy,
	).Scan(&id); err != nil {
		t.Fatalf("insert running run: %v", err)
	}
	return id
}

// clearSyncRuns empties receipt_sync_runs so a test starts from no history.
func clearSyncRuns(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `DELETE FROM receipt_sync_runs`); err != nil {
		t.Fatalf("clear receipt_sync_runs: %v", err)
	}
}

func TestCancelSync_MarksCancelledAndFreesSingleFlight(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	id := runningRunID(t, "reprocess_all")

	rec := httptest.NewRecorder()
	CancelSyncHandler(testPool).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/cancel", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["cancelled"] != true {
		t.Errorf("cancelled = %v, want true", body["cancelled"])
	}

	var status string
	var finished sql.NullTime
	if err := testPool.QueryRow(context.Background(),
		`SELECT status, finished_at FROM receipt_sync_runs WHERE id=$1`, id,
	).Scan(&status, &finished); err != nil {
		t.Fatalf("select: %v", err)
	}
	// 'cancelled', not 'failed': the operator chose this.
	if status != "cancelled" {
		t.Errorf("status = %q, want %q", status, "cancelled")
	}
	if !finished.Valid {
		t.Error("finished_at is NULL; a terminal row must carry one")
	}

	// The single-running partial unique index keys off status='running', so a
	// cancelled row must leave the slot open for the next sync.
	if _, err := testPool.Exec(context.Background(),
		`INSERT INTO receipt_sync_runs (status, triggered_by) VALUES ('running','manual')`); err != nil {
		t.Errorf("a new sync could not start after a cancel: %v", err)
	}
	// That proof row is a running sync as far as any later test is concerned.
	clearSyncRuns(t)
}

func TestCancelSync_NoRunInFlightIsNotAnError(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	clearSyncRuns(t)

	rec := httptest.NewRecorder()
	CancelSyncHandler(testPool).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/cancel", nil))
	// The run most likely finished between the button rendering and the tap.
	// That is not the operator's mistake and must not read as a failure.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["cancelled"] != false {
		t.Errorf("cancelled = %v, want false", body["cancelled"])
	}
}

// TestCancelSync_GoroutineCannotOverwriteTheCancelledRow is the race that
// would otherwise show the operator "failed: context canceled" for something
// they chose: the goroutine notices its context died a moment AFTER the
// handler moved the row, and writes its own terminal status over it.
func TestCancelSync_GoroutineCannotOverwriteTheCancelledRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	id := runningRunID(t, "manual")

	rec := httptest.NewRecorder()
	CancelSyncHandler(testPool).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/cancel", nil))

	// The goroutine's late terminal write, guarded by status='running'.
	msg := "context canceled"
	finishSyncRun(context.Background(), testPool, id, "failed", &msg, nil)

	var status string
	var errCol sql.NullString
	if err := testPool.QueryRow(context.Background(),
		`SELECT status, error FROM receipt_sync_runs WHERE id=$1`, id,
	).Scan(&status, &errCol); err != nil {
		t.Fatalf("select: %v", err)
	}
	if status != "cancelled" {
		t.Errorf("status = %q, want it to stay %q", status, "cancelled")
	}
	if errCol.Valid && errCol.String != "" {
		t.Errorf("error = %q; a cancelled run must not show an error", errCol.String)
	}
}

// TestRegisterSyncCancel_ReleaseRemovesTheEntry — a CancelFunc left in the map
// after its run ended would cancel an unrelated later run that reused the id.
func TestRegisterSyncCancel_ReleaseRemovesTheEntry(t *testing.T) {
	ctx, release := registerSyncCancel(4242)
	if !cancelSyncRun(4242) {
		t.Fatal("registered run was not reachable by cancelSyncRun")
	}
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context was not cancelled")
	}
	release()
	if cancelSyncRun(4242) {
		t.Error("CancelFunc still registered after release")
	}
}
