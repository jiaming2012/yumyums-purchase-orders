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

const recoverTestPrefix = "https://s3.us-test-000.backblazeb2.com/hq-test/"

const recoverDeadURL = "https://nyc3.digitaloceanspaces.com/hq.yumyums/receipts/tx/0.jpg"

func resetRecoverFixtures(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(t.Context(),
		`TRUNCATE pending_purchases, purchase_line_items, purchase_events, vendors,
		          receipt_sync_runs RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate recover fixtures: %v", err)
	}
}

func countSyncRuns(t *testing.T) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM receipt_sync_runs`).Scan(&n); err != nil {
		t.Fatalf("count receipt_sync_runs: %v", err)
	}
	return n
}

func TestRecoverReceiptsHandler_DryRun_WritesNoRunRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetRecoverFixtures(t)
	insertStuckPendingPurchaseWithURL(t, "tx-dry-1", "needs_review", false, false, recoverDeadURL)

	runner := func(ctx context.Context, dead receipt.DeadRows) (receipt.RecoverResult, error) {
		t.Error("runner must not be called on dry-run")
		return receipt.RecoverResult{}, nil
	}
	h := RecoverReceiptsHandler(testPool, recoverTestPrefix, runner)

	req := httptest.NewRequest("POST", "/purchases/recover-receipts", strings.NewReader(`{"dry_run":true}`))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var resp struct {
		DryRun        bool     `json:"dry_run"`
		PendingRows   int      `json:"pending_purchases_rows"`
		EventRows     int      `json:"purchase_events_rows"`
		DistinctTxIDs int      `json:"distinct_tx_ids"`
		Since         string   `json:"since"`
		TxIDs         []string `json:"tx_ids"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if !resp.DryRun || resp.PendingRows != 1 || resp.EventRows != 0 || resp.DistinctTxIDs != 1 {
		t.Errorf("response = %+v, want dry_run with 1 pending row / 1 tx id", resp)
	}
	if len(resp.TxIDs) != 1 || resp.TxIDs[0] != "tx-dry-1" {
		t.Errorf("tx_ids = %v, want [tx-dry-1]", resp.TxIDs)
	}
	if resp.Since == "" {
		t.Errorf("since missing from dry-run response")
	}
	if n := countSyncRuns(t); n != 0 {
		t.Errorf("receipt_sync_runs rows = %d, want 0 after dry-run", n)
	}
}

func TestRecoverReceiptsHandler_RealRun_SingleFlightAndTerminalCounts(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetRecoverFixtures(t)
	insertStuckPendingPurchaseWithURL(t, "tx-real-1", "needs_review", false, false, recoverDeadURL)

	release := make(chan struct{})
	started := make(chan struct{})
	runner := func(ctx context.Context, dead receipt.DeadRows) (receipt.RecoverResult, error) {
		close(started)
		<-release
		return receipt.RecoverResult{
			Examined: 3, Recovered: 1, MissingAtMercury: 1, Failed: 1,
			MissingTxIDs: []string{"tx-m"}, FailedTxIDs: []string{"tx-f"},
		}, nil
	}
	h := RecoverReceiptsHandler(testPool, recoverTestPrefix, runner)

	// First call claims the single-flight slot.
	w1 := httptest.NewRecorder()
	h(w1, httptest.NewRequest("POST", "/purchases/recover-receipts", strings.NewReader(`{}`)))
	if w1.Code != 200 {
		t.Fatalf("first call status = %d, body %s", w1.Code, w1.Body.String())
	}
	var first struct {
		SyncID        int64  `json:"sync_id"`
		Status        string `json:"status"`
		DistinctTxIDs int    `json:"distinct_tx_ids"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &first); err != nil {
		t.Fatalf("parse first response: %v", err)
	}
	if first.Status != "running" || first.DistinctTxIDs != 1 {
		t.Errorf("first response = %+v, want running with 1 tx id", first)
	}

	<-started

	// Second call while running → 409.
	w2 := httptest.NewRecorder()
	h(w2, httptest.NewRequest("POST", "/purchases/recover-receipts", strings.NewReader(`{}`)))
	if w2.Code != 409 {
		t.Errorf("concurrent call status = %d, want 409 (body %s)", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "sync_already_running") {
		t.Errorf("concurrent call body = %s, want sync_already_running", w2.Body.String())
	}

	close(release)

	// Poll for the terminal row and assert the documented column mapping:
	// processed=examined, auto_created=recovered, pending_review=missing, cached=failed.
	deadline := time.Now().Add(5 * time.Second)
	for {
		var status string
		var processed, autoCreated, pendingReview, cached int
		err := testPool.QueryRow(context.Background(),
			`SELECT status, processed, auto_created, pending_review, cached
			   FROM receipt_sync_runs WHERE id = $1`, first.SyncID,
		).Scan(&status, &processed, &autoCreated, &pendingReview, &cached)
		if err != nil {
			t.Fatalf("read sync run: %v", err)
		}
		if status == "done" {
			if processed != 3 || autoCreated != 1 || pendingReview != 1 || cached != 1 {
				t.Errorf("terminal counts = (%d,%d,%d,%d), want (3,1,1,1)", processed, autoCreated, pendingReview, cached)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("sync run %d never reached done (status %s)", first.SyncID, status)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestRecoverReceiptsHandler_StorageUnconfigured(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	h := RecoverReceiptsHandler(testPool, "", func(ctx context.Context, dead receipt.DeadRows) (receipt.RecoverResult, error) {
		t.Error("runner must not be called when storage is unconfigured")
		return receipt.RecoverResult{}, nil
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("POST", "/purchases/recover-receipts", strings.NewReader(`{"dry_run":true}`)))
	if w.Code != 503 {
		t.Errorf("status = %d, want 503 (body %s)", w.Code, w.Body.String())
	}
}

func TestRecoverReceiptsHandler_InvalidJSON(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	h := RecoverReceiptsHandler(testPool, recoverTestPrefix, func(ctx context.Context, dead receipt.DeadRows) (receipt.RecoverResult, error) {
		t.Error("runner must not be called on invalid JSON")
		return receipt.RecoverResult{}, nil
	})
	w := httptest.NewRecorder()
	h(w, httptest.NewRequest("POST", "/purchases/recover-receipts", strings.NewReader(`{not-json`)))
	if w.Code != 400 {
		t.Errorf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
}
