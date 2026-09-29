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
)

// redownloadHelper invokes RedownloadPendingHandler with the row id in the
// URL path (chi.URLParam), mirroring reprocessOneHelper.
func redownloadHelper(t *testing.T, id string, runner RedownloadRunner) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Post("/api/v1/inventory/purchases/pending/{id}/redownload",
		RedownloadPendingHandler(testPool, runner))
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/inventory/purchases/pending/"+id+"/redownload", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// recordingRedownloader captures what the handler asks Mercury for.
type recordingRedownloader struct {
	mu     sync.Mutex
	calls  int
	txIDs  []string
	sinces []time.Time
	status string
	err    error
}

func (r *recordingRedownloader) run(ctx context.Context, bankTxID string, since time.Time) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	r.txIDs = append(r.txIDs, bankTxID)
	r.sinces = append(r.sinces, since)
	return r.status, r.err
}

func (r *recordingRedownloader) snapshot() (int, []string, []time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls, append([]string(nil), r.txIDs...), append([]time.Time(nil), r.sinces...)
}

func insertPendingWithDate(t *testing.T, bankTxID, eventDate, reason string, receiptURL *string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url, event_date, created_at)
		VALUES ($1, -18.39, 'Restaurant Depot', '[]'::jsonb, $2, $3, $4::date, now())
		RETURNING id::text`, bankTxID, reason, receiptURL, eventDate).Scan(&id); err != nil {
		t.Fatalf("insertPendingWithDate %s: %v", bankTxID, err)
	}
	return id
}

// TestRedownload_AsksMercuryForThatChargeAndReparses is the reason the
// endpoint exists: the operator replaced the receipt on Mercury (the wrong
// file, or a purchase now paired with its refund) and the row in HQ still
// carries the old download. Only Mercury can supply the new files; a
// storage re-read (reprocess) would re-parse the stale ones.
func TestRedownload_AsksMercuryForThatChargeAndReparses(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	url := "https://spaces.example.com/r/old-wrong-receipt.jpg"
	id := insertPendingWithDate(t, "rd-target", "2026-09-11", "Receipt derived total does not match", &url)
	_ = insertPendingWithDate(t, "rd-sibling", "2026-09-12", "Receipt could not be parsed automatically", &url)

	rr := &recordingRedownloader{status: "pending_review"}
	rec := redownloadHelper(t, id, rr.run)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" || body["id"] != id {
		t.Errorf("body = %v, want status running for id %s", body, id)
	}
	syncID, _ := body["sync_id"].(float64)
	if syncID <= 0 {
		t.Fatalf("sync_id = %v, want a positive run id", body["sync_id"])
	}

	status, errCol := waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	if status != "done" || errCol != nil {
		t.Errorf("run finished status=%q error=%v, want done/nil", status, errCol)
	}

	calls, txIDs, sinces := rr.snapshot()
	if calls != 1 || len(txIDs) != 1 || txIDs[0] != "rd-target" {
		t.Fatalf("runner calls=%d txIDs=%v, want exactly one call for rd-target", calls, txIDs)
	}
	// The Mercury window must open before the charge's own date, or the
	// lookup cannot find it; a month is generous for a posted-vs-cleared gap.
	eventDate := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	if !sinces[0].Before(eventDate.AddDate(0, 0, -7)) {
		t.Errorf("since = %s, want well before the 2026-09-11 event date", sinces[0].Format("2006-01-02"))
	}

	var triggeredBy string
	var processed, pendingReview int
	if err := testPool.QueryRow(t.Context(),
		`SELECT triggered_by, processed, pending_review FROM receipt_sync_runs WHERE id=$1`, int64(syncID)).
		Scan(&triggeredBy, &processed, &pendingReview); err != nil {
		t.Fatalf("select receipt_sync_runs: %v", err)
	}
	if triggeredBy != "redownload" {
		t.Errorf("triggered_by = %q, want redownload", triggeredBy)
	}
	if processed != 1 || pendingReview != 1 {
		t.Errorf("processed=%d pending_review=%d, want 1/1", processed, pendingReview)
	}
}

// TestRedownload_MissingReceiptRowIsOffered — a "Missing Receipt" row is the
// case this exists for: the operator has since attached a file on Mercury.
func TestRedownload_MissingReceiptRowIsOffered(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	id := insertPendingWithDate(t, "rd-nourl", "2026-09-12", "no_attachment_on_bank_tx", nil)
	rr := &recordingRedownloader{status: "auto_created"}
	rec := redownloadHelper(t, id, rr.run)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s want 200 — a row with no stored receipt is exactly what re-download is for", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	syncID, _ := body["sync_id"].(float64)
	status, _ := waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	if status != "done" {
		t.Errorf("status = %q, want done", status)
	}
	var autoCreated int
	_ = testPool.QueryRow(t.Context(), `SELECT auto_created FROM receipt_sync_runs WHERE id=$1`, int64(syncID)).Scan(&autoCreated)
	if autoCreated != 1 {
		t.Errorf("auto_created = %d, want 1", autoCreated)
	}
}

// TestRedownload_NothingAtMercuryFailsLoudly — if Mercury does not return the
// charge, or returns it with no attachment, the run must end FAILED with a
// plain-words reason, not "done" with nothing changed.
func TestRedownload_NothingAtMercuryFailsLoudly(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	for _, tc := range []struct {
		status string
		want   string
	}{
		{"missing_at_mercury", "Mercury did not return this charge"},
		{"no_attachments", "Mercury has no receipt attached to this charge"},
	} {
		resetPendingPurchases(t)
		resetSyncRuns(t)
		id := insertPendingWithDate(t, "rd-"+tc.status, "2026-09-12", "Receipt could not be parsed automatically", nil)
		rr := &recordingRedownloader{status: tc.status}
		rec := redownloadHelper(t, id, rr.run)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d body=%s want 200", tc.status, rec.Code, rec.Body.String())
		}
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		syncID, _ := body["sync_id"].(float64)
		status, errCol := waitForTerminalStatus(t, int64(syncID), 2*time.Second)
		if status != "failed" || errCol == nil {
			t.Errorf("%s: run status=%q error=%v, want failed with a reason", tc.status, status, errCol)
			continue
		}
		if *errCol != tc.want {
			t.Errorf("%s: error = %q, want %q", tc.status, *errCol, tc.want)
		}
	}
}

// TestRedownload_RefusalsMatchTheSiblingEndpoints — 404 / 422 / 409, and no
// run row is claimed on a refusal.
func TestRedownload_RefusalsMatchTheSiblingEndpoints(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	rr := &recordingRedownloader{status: "pending_review"}

	rec := redownloadHelper(t, "00000000-0000-0000-0000-000000000000", rr.run)
	if rec.Code != http.StatusNotFound {
		t.Errorf("unknown: status=%d want 404", rec.Code)
	}
	closed := insertConfirmedPendingPurchase(t, "rd-confirmed")
	rec = redownloadHelper(t, closed, rr.run)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("confirmed: status=%d want 422", rec.Code)
	}
	var n int
	_ = testPool.QueryRow(t.Context(), `SELECT count(*) FROM receipt_sync_runs`).Scan(&n)
	if n != 0 {
		t.Errorf("receipt_sync_runs has %d rows after refusals, want 0", n)
	}

	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO receipt_sync_runs (status, triggered_by) VALUES ('running', 'manual')`); err != nil {
		t.Fatalf("pre-insert running sync row: %v", err)
	}
	open := insertPendingWithDate(t, "rd-open", "2026-09-12", "Receipt could not be parsed automatically", nil)
	rec = redownloadHelper(t, open, rr.run)
	if rec.Code != http.StatusConflict {
		t.Errorf("while running: status=%d body=%s want 409", rec.Code, rec.Body.String())
	}
	if calls, _, _ := rr.snapshot(); calls != 0 {
		t.Errorf("runner called %d times across refusals, want 0", calls)
	}
}
