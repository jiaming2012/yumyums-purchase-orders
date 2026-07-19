package inventory

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/receipt"
)

// TestSyncReceiptsStatus_ReturnsLookbackDays asserts that GET /sync-receipts/status
// returns the injected lookback_days as an integer in the JSON body. This is a
// display-only field (not a DB column) — the frontend uses it to compute the
// sync window start date for the "Synced from {Mon DD}" chip copy (260701-a23).
func TestSyncReceiptsStatus_ReturnsLookbackDays(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetSyncRuns(t)

	// Seed one 'done' row directly — we're testing the status handler, not the
	// full sync goroutine flow.
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO receipt_sync_runs
		 (status, triggered_by, started_at, finished_at,
		  processed, auto_created, pending_review, cached)
		 VALUES ('done', 'manual', now(), now(), 5, 1, 1, 0)`); err != nil {
		t.Fatalf("insert seed row: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/sync-receipts/status", nil)
	SyncReceiptsStatusHandler(testPool, 14)(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status handler: code=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v (raw=%s)", err, rec.Body.String())
	}
	// JSON ints decode into map[string]any as float64.
	if got := body["lookback_days"]; got != float64(14) {
		t.Errorf("lookback_days = %v (%T), want 14 (float64)", got, got)
	}
	// Regression safety: existing fields still present with expected values.
	if got := body["triggered_by"]; got != "manual" {
		t.Errorf("triggered_by = %v, want manual", got)
	}
	if got := body["processed"]; got != float64(5) {
		t.Errorf("processed = %v, want 5", got)
	}
	if got := body["status"]; got != "done" {
		t.Errorf("status = %v, want done", got)
	}
}

// resetSyncRuns truncates the receipt_sync_runs table so each test starts
// from id=1 with no rows.
func resetSyncRuns(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(t.Context(),
		`TRUNCATE receipt_sync_runs RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate receipt_sync_runs: %v", err)
	}
}

// stubRunnerOK returns the configured IngestResult.
func stubRunnerOK(res receipt.IngestResult) IngestRunner {
	return func(ctx context.Context) (receipt.IngestResult, error) {
		return res, nil
	}
}

// stubRunnerSlow waits on a channel before returning; lets the test hold the
// sync in 'running' status long enough to assert single-flight.
func stubRunnerSlow(release <-chan struct{}, res receipt.IngestResult) IngestRunner {
	return func(ctx context.Context) (receipt.IngestResult, error) {
		<-release
		return res, nil
	}
}

// stubRunnerPanic panics with the given value.
func stubRunnerPanic(msg string) IngestRunner {
	return func(ctx context.Context) (receipt.IngestResult, error) {
		panic(msg)
	}
}

// waitForStatus polls the receipt_sync_runs row by id until status != 'running'
// or the deadline expires. Returns the row's status + error column for assertion.
func waitForTerminalStatus(t *testing.T, id int64, deadline time.Duration) (string, *string) {
	t.Helper()
	end := time.Now().Add(deadline)
	for time.Now().Before(end) {
		var status string
		var errCol *string
		err := testPool.QueryRow(t.Context(),
			`SELECT status, error FROM receipt_sync_runs WHERE id=$1`, id).Scan(&status, &errCol)
		if err == nil && status != "running" {
			return status, errCol
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("waitForTerminalStatus: row %d still running after %s", id, deadline)
	return "", nil
}

// TestSyncReceipts_SingleFlight_Returns409 asserts the partial unique index on
// status='running' causes a concurrent POST to return 409 sync_already_running.
// After the first run is marked done, a third POST succeeds.
func TestSyncReceipts_SingleFlight_Returns409(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetSyncRuns(t)

	release := make(chan struct{})
	defer close(release) // ensure goroutine exits if test fails
	runner := stubRunnerSlow(release, receipt.IngestResult{})

	handler := SyncReceiptsHandler(testPool, runner)

	// First POST → 200, status running.
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/sync-receipts", nil)
	handler(rec1, req1)
	if rec1.Code != 200 {
		t.Fatalf("first POST: status=%d body=%s want 200", rec1.Code, rec1.Body.String())
	}
	var body1 map[string]any
	if err := json.Unmarshal(rec1.Body.Bytes(), &body1); err != nil {
		t.Fatalf("first POST decode: %v", err)
	}
	if body1["status"] != "running" {
		t.Errorf("first POST status = %v, want running", body1["status"])
	}

	// Second POST while the goroutine is still parked on the channel → 409.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/sync-receipts", nil)
	handler(rec2, req2)
	if rec2.Code != 409 {
		t.Fatalf("second POST: status=%d body=%s want 409", rec2.Code, rec2.Body.String())
	}
	if !strings.Contains(rec2.Body.String(), "sync_already_running") {
		t.Errorf("second POST body = %s, want sync_already_running", rec2.Body.String())
	}

	// Release the goroutine so it can finish; wait for status=done.
	release <- struct{}{}
	// Re-read id from body1.
	idF, _ := body1["id"].(float64)
	status, _ := waitForTerminalStatus(t, int64(idF), 2*time.Second)
	if status != "done" {
		t.Fatalf("waited for done, got %s", status)
	}

	// Third POST after row is no longer running → 200.
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/sync-receipts", nil)
	handler(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("third POST: status=%d body=%s want 200", rec3.Code, rec3.Body.String())
	}
}

// TestSyncReceipts_Goroutine_UpdatesRowToDone asserts the goroutine writes
// the IngestResult counts back to the row when the runner returns successfully.
func TestSyncReceipts_Goroutine_UpdatesRowToDone(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetSyncRuns(t)

	want := receipt.IngestResult{Processed: 7, AutoCreated: 2, PendingReview: 1, Cached: 4}
	runner := stubRunnerOK(want)
	handler := SyncReceiptsHandler(testPool, runner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sync-receipts", nil)
	handler(rec, req)
	if rec.Code != 200 {
		t.Fatalf("POST: status=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	idF, _ := body["id"].(float64)
	id := int64(idF)

	status, _ := waitForTerminalStatus(t, id, 2*time.Second)
	if status != "done" {
		t.Fatalf("status = %s, want done", status)
	}

	var processed, autoCreated, pendingReview, cached int
	var finishedAt *time.Time
	if err := testPool.QueryRow(t.Context(),
		`SELECT processed, auto_created, pending_review, cached, finished_at
		 FROM receipt_sync_runs WHERE id=$1`, id).
		Scan(&processed, &autoCreated, &pendingReview, &cached, &finishedAt); err != nil {
		t.Fatalf("select: %v", err)
	}
	if processed != want.Processed || autoCreated != want.AutoCreated ||
		pendingReview != want.PendingReview || cached != want.Cached {
		t.Errorf("counts = {processed:%d auto_created:%d pending_review:%d cached:%d}, want %+v",
			processed, autoCreated, pendingReview, cached, want)
	}
	if finishedAt == nil {
		t.Errorf("finished_at is NULL, want non-NULL")
	}
}

// TestSyncReceipts_Goroutine_RecoversFromPanic asserts the defer recover() in
// runSyncGoroutine writes status='failed' with error='panic: …' so no orphan
// running row blocks future syncs.
func TestSyncReceipts_Goroutine_RecoversFromPanic(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetSyncRuns(t)

	runner := stubRunnerPanic("boom")
	handler := SyncReceiptsHandler(testPool, runner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/sync-receipts", nil)
	handler(rec, req)
	if rec.Code != 200 {
		t.Fatalf("POST: status=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	idF, _ := body["id"].(float64)
	id := int64(idF)

	status, errCol := waitForTerminalStatus(t, id, 2*time.Second)
	if status != "failed" {
		t.Fatalf("status = %s, want failed", status)
	}
	if errCol == nil || !strings.Contains(*errCol, "panic: boom") {
		t.Errorf("error column = %v, want contains 'panic: boom'", errCol)
	}

	var finishedAt *time.Time
	if err := testPool.QueryRow(t.Context(),
		`SELECT finished_at FROM receipt_sync_runs WHERE id=$1`, id).Scan(&finishedAt); err != nil {
		t.Fatalf("select finished_at: %v", err)
	}
	if finishedAt == nil {
		t.Errorf("finished_at is NULL after panic recovery, want non-NULL")
	}

	// Critical: no orphan running row left behind — a subsequent POST should succeed.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/sync-receipts", nil)
	SyncReceiptsHandler(testPool, stubRunnerOK(receipt.IngestResult{}))(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("post-panic POST: status=%d body=%s want 200 (proves no orphan row)",
			rec2.Code, rec2.Body.String())
	}
}
