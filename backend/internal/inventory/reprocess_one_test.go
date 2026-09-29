package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/yumyums/hq/internal/receipt"
)

// reprocessOneHelper invokes ReprocessOnePendingHandler with the row id in the
// URL path (chi.URLParam, not a JSON body), mirroring retryParseHelper.
func reprocessOneHelper(t *testing.T, id string, runner BatchReprocessRunner) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Post("/api/v1/inventory/purchases/pending/{id}/reprocess",
		ReprocessOnePendingHandler(testPool, runner))
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/inventory/purchases/pending/"+id+"/reprocess", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// recordingRunner captures every row the handler hands to the runner, across
// goroutines, so the test can assert exactly which receipt was re-read.
type recordingRunner struct {
	mu    sync.Mutex
	rows  []receipt.PendingRowForReprocess
	calls int
}

func (r *recordingRunner) run(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.rows = append(r.rows, rows...)
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.BankTxID] = "pending_review"
	}
	return out, nil
}

func (r *recordingRunner) snapshot() (int, []receipt.PendingRowForReprocess) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, append([]receipt.PendingRowForReprocess(nil), r.rows...)
}

// TestReprocessOne_ReReadsExactlyThatReceipt is the reason the endpoint
// exists: the per-card "Retry parse" used to re-arm a flag and start a full
// Mercury sync, which cannot reach a charge older than the 14-day lookback —
// so the row the operator tapped was never revisited and the only fix was the
// admin-only "Retry Parse (All Receipts)" sweep. This runs the storage
// re-read for ONE row, and leaves every sibling alone.
func TestReprocessOne_ReReadsExactlyThatReceipt(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	target := insertStuckPendingPurchaseWithURL(t, "one-target", "Receipt derived total $47.58 does not match", true, true, "https://spaces.example.com/r/target.jpg")
	_ = insertStuckPendingPurchaseWithURL(t, "one-sibling", "Receipt could not be parsed automatically", false, true, "https://spaces.example.com/r/sibling.jpg")

	rr := &recordingRunner{}
	rec := reprocessOneHelper(t, target, rr.run)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" {
		t.Errorf("status = %v, want running", body["status"])
	}
	if body["id"] != target {
		t.Errorf("id = %v, want %s (the row that was tapped)", body["id"], target)
	}
	syncID, _ := body["sync_id"].(float64)
	if syncID <= 0 {
		t.Fatalf("sync_id = %v, want a positive run id", body["sync_id"])
	}
	if _, ok := body["started_at"]; !ok {
		t.Errorf("response missing started_at")
	}

	status, errCol := waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	if status != "done" || errCol != nil {
		t.Errorf("run finished status=%q error=%v, want done/nil", status, errCol)
	}

	calls, rows := rr.snapshot()
	if calls != 1 {
		t.Fatalf("runner called %d times, want 1", calls)
	}
	if len(rows) != 1 || rows[0].BankTxID != "one-target" {
		t.Fatalf("runner rows = %+v, want exactly [one-target]", rows)
	}
	if len(rows[0].ReceiptURLs) != 1 || rows[0].ReceiptURLs[0] != "https://spaces.example.com/r/target.jpg" {
		t.Errorf("ReceiptURLs = %v, want the stored legacy receipt_url wrapped as a one-element list", rows[0].ReceiptURLs)
	}

	var triggeredBy string
	var processed int
	if err := testPool.QueryRow(t.Context(),
		`SELECT triggered_by, processed FROM receipt_sync_runs WHERE id=$1`, int64(syncID)).Scan(&triggeredBy, &processed); err != nil {
		t.Fatalf("select receipt_sync_runs: %v", err)
	}
	if triggeredBy != "reprocess_one" {
		t.Errorf("triggered_by = %q, want reprocess_one", triggeredBy)
	}
	if processed != 1 {
		t.Errorf("processed = %d, want 1", processed)
	}
}

// TestReprocessOne_MultiURLRowPassesEveryAttachment: a multi-attachment row
// stores receipt_urls; all of them go to the runner, same as the sweep.
func TestReprocessOne_MultiURLRowPassesEveryAttachment(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	var id string
	if err := testPool.QueryRow(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url, receipt_urls, created_at)
		VALUES ('one-multi', -12.5, 'TestVendor', '[]'::jsonb, 'Receipt could not be parsed automatically',
		        'https://spaces.example.com/r/m-0.jpg',
		        '["https://spaces.example.com/r/m-0.jpg","https://spaces.example.com/r/m-1.jpg"]'::jsonb, now())
		RETURNING id::text`).Scan(&id); err != nil {
		t.Fatalf("insert multi-url row: %v", err)
	}

	rr := &recordingRunner{}
	rec := reprocessOneHelper(t, id, rr.run)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	syncID, _ := body["sync_id"].(float64)
	waitForTerminalStatus(t, int64(syncID), 2*time.Second)

	_, rows := rr.snapshot()
	if len(rows) != 1 || len(rows[0].ReceiptURLs) != 2 {
		t.Fatalf("rows = %+v, want one row carrying both attachment URLs", rows)
	}
	if rows[0].BankTotal != -12.5 || rows[0].Vendor != "TestVendor" {
		t.Errorf("row = %+v, want bank_total -12.5 / vendor TestVendor carried through", rows[0])
	}
}

// TestReprocessOne_UnknownRowIs404 — an id nothing matches.
func TestReprocessOne_UnknownRowIs404(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	rr := &recordingRunner{}
	rec := reprocessOneHelper(t, "00000000-0000-0000-0000-000000000000", rr.run)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s want 404", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["error"] != "pending_purchase_not_found" {
		t.Errorf("error = %v, want pending_purchase_not_found", body["error"])
	}
	if calls, _ := rr.snapshot(); calls != 0 {
		t.Errorf("runner called %d times for an unknown row, want 0", calls)
	}
	var n int
	_ = testPool.QueryRow(t.Context(), `SELECT count(*) FROM receipt_sync_runs`).Scan(&n)
	if n != 0 {
		t.Errorf("receipt_sync_runs has %d rows, want 0 — a refused request must not claim the sync slot", n)
	}
}

// TestReprocessOne_ClosedRowIs422 — confirmed or discarded rows are not
// re-read; the operator already decided what they are.
func TestReprocessOne_ClosedRowIs422(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	for _, tc := range []struct {
		name string
		id   string
	}{
		{"confirmed", insertConfirmedPendingPurchase(t, "one-confirmed")},
		{"discarded", insertDiscardedPendingPurchase(t, "one-discarded")},
	} {
		rr := &recordingRunner{}
		rec := reprocessOneHelper(t, tc.id, rr.run)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status=%d body=%s want 422", tc.name, rec.Code, rec.Body.String())
			continue
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if body["error"] != "row_not_pending" {
			t.Errorf("%s: error = %v, want row_not_pending", tc.name, body["error"])
		}
		if calls, _ := rr.snapshot(); calls != 0 {
			t.Errorf("%s: runner called %d times, want 0", tc.name, calls)
		}
	}
}

// TestReprocessOne_NoStoredReceiptIs422 — with nothing in storage there is
// nothing to re-read. The card falls back to the Mercury path for these; the
// handler must say so rather than claim the sync slot and do nothing.
func TestReprocessOne_NoStoredReceiptIs422(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	id := insertStuckPendingPurchase(t, "one-no-url", "no_attachment_on_bank_tx", false, false)

	rr := &recordingRunner{}
	rec := reprocessOneHelper(t, id, rr.run)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s want 422", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["error"] != "no_stored_receipt" {
		t.Errorf("error = %v, want no_stored_receipt", body["error"])
	}
	if calls, _ := rr.snapshot(); calls != 0 {
		t.Errorf("runner called %d times, want 0", calls)
	}
	var n int
	_ = testPool.QueryRow(t.Context(), `SELECT count(*) FROM receipt_sync_runs`).Scan(&n)
	if n != 0 {
		t.Errorf("receipt_sync_runs has %d rows, want 0", n)
	}
}

// TestReprocessOne_ConflictsWithRunningSync — same single-flight row as every
// other run kind, so the Stop-sync control and the status poll keep working.
func TestReprocessOne_ConflictsWithRunningSync(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO receipt_sync_runs (status, triggered_by) VALUES ('running', 'manual')`); err != nil {
		t.Fatalf("pre-insert running sync row: %v", err)
	}
	id := insertStuckPendingPurchaseWithURL(t, "one-conflict", "Receipt could not be parsed automatically", false, true, "https://spaces.example.com/r/c.jpg")

	rr := &recordingRunner{}
	rec := reprocessOneHelper(t, id, rr.run)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s want 409", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["error"] != "sync_already_running" {
		t.Errorf("error = %v, want sync_already_running", body["error"])
	}
	if calls, _ := rr.snapshot(); calls != 0 {
		t.Errorf("runner called %d times while a sync was running, want 0", calls)
	}
}
