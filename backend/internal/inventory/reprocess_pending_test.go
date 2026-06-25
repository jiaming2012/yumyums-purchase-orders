package inventory

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/receipt"
)

// resetPendingPurchases truncates pending_purchases to ensure clean state per test.
func resetPendingPurchases(t *testing.T) {
	t.Helper()
	if _, err := testPool.Exec(t.Context(),
		`TRUNCATE pending_purchases RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate pending_purchases: %v", err)
	}
}

// insertStuckPendingPurchase inserts an unconfirmed/undiscarded pending row
// that simulates a "stuck" state from legacy parser output: items may be
// populated, parse_error may be set, and reason won't match the upgrade gate.
// Returns the inserted id.
func insertStuckPendingPurchase(t *testing.T, bankTxID, reason string, withItems bool, withParseError bool) string {
	t.Helper()
	itemsJSON := "'[]'::jsonb"
	if withItems {
		itemsJSON = `'[{"name":"x","quantity":1,"price":10.00,"is_case":false}]'::jsonb`
	}
	var id string
	var parseErr *string
	if withParseError {
		msg := "some parse error"
		parseErr = &msg
	}
	q := `INSERT INTO pending_purchases
	        (bank_tx_id, bank_total, vendor, items, reason, parse_error, created_at)
	      VALUES ($1, 0, 'TestVendor', ` + itemsJSON + `, $2, $3, now())
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID, reason, parseErr).Scan(&id); err != nil {
		t.Fatalf("insertStuckPendingPurchase %s: %v", bankTxID, err)
	}
	return id
}

// insertConfirmedPendingPurchase inserts a row with confirmed_at set.
func insertConfirmedPendingPurchase(t *testing.T, bankTxID string) string {
	t.Helper()
	var id string
	q := `INSERT INTO pending_purchases
	        (bank_tx_id, bank_total, vendor, items, reason, created_at, confirmed_at)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, 'already_confirmed', now(), now())
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID).Scan(&id); err != nil {
		t.Fatalf("insertConfirmedPendingPurchase %s: %v", bankTxID, err)
	}
	return id
}

// insertDiscardedPendingPurchase inserts a row with discarded_at set.
func insertDiscardedPendingPurchase(t *testing.T, bankTxID string) string {
	t.Helper()
	var id string
	q := `INSERT INTO pending_purchases
	        (bank_tx_id, bank_total, vendor, items, reason, created_at, discarded_at)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, 'already_discarded', now(), now())
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID).Scan(&id); err != nil {
		t.Fatalf("insertDiscardedPendingPurchase %s: %v", bankTxID, err)
	}
	return id
}

// pendingRow holds the columns we inspect after the reprocess call.
type pendingRow struct {
	Items      string
	ParseError *string
	Reason     *string
}

// fetchPendingRow reads items, parse_error, reason for the given id.
func fetchPendingRow(t *testing.T, id string) pendingRow {
	t.Helper()
	var row pendingRow
	err := testPool.QueryRow(t.Context(),
		`SELECT items::text, parse_error, reason FROM pending_purchases WHERE id=$1::uuid`, id).
		Scan(&row.Items, &row.ParseError, &row.Reason)
	if err != nil {
		t.Fatalf("fetchPendingRow %s: %v", id, err)
	}
	return row
}

// TestReprocessAllPendingHandler_ResetsAndQueues verifies the happy path:
// - 3 stuck rows → all reset to upgrade-eligible state
// - 1 confirmed row → untouched
// - 1 discarded row → untouched
// - reset_count=3 in JSON response
// - a receipt_sync_runs row inserted with triggered_by='reprocess_all'
func TestReprocessAllPendingHandler_ResetsAndQueues(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	// Seed 3 stuck rows in various legacy states.
	// Row A: items populated, reason is old mismatch message
	idA := insertStuckPendingPurchase(t, "stuck-a", "Receipt total $42.00 doesn't match...", true, false)
	// Row B: parse_error set, items empty
	idB := insertStuckPendingPurchase(t, "stuck-b", "Line item sum $X doesn't match...", false, true)
	// Row C: items empty, reason is another old mismatch
	idC := insertStuckPendingPurchase(t, "stuck-c", "item count 3 doesn't match...", false, false)

	// Seed 1 confirmed row — must NOT be affected.
	idConfirmed := insertConfirmedPendingPurchase(t, "confirmed-1")

	// Seed 1 discarded row — must NOT be affected.
	idDiscarded := insertDiscardedPendingPurchase(t, "discarded-1")

	// Stub runner: no actual Mercury/Anthropic calls.
	runner := stubRunnerOK(receipt.IngestResult{Processed: 0})

	handler := ReprocessAllPendingHandler(testPool, runner)

	req := httptest.NewRequest("POST", "/inventory/purchases/reprocess-all", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 200 {
		t.Fatalf("POST reprocess-all: status=%d body=%s want 200", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Assert reset_count=3.
	resetCount, _ := body["reset_count"].(float64)
	if resetCount != 3 {
		t.Errorf("reset_count = %v, want 3", resetCount)
	}
	if _, ok := body["sync_id"]; !ok {
		t.Errorf("response missing sync_id field")
	}
	if body["status"] != "running" {
		t.Errorf("response status = %v, want running", body["status"])
	}

	// Assert: the 3 stuck rows were reset to upgrade-eligible state.
	const wantItems = "[]"
	const wantReason = "Receipt could not be parsed automatically"
	for _, id := range []string{idA, idB, idC} {
		row := fetchPendingRow(t, id)
		if row.Items != wantItems {
			t.Errorf("row %s items = %q, want %q", id, row.Items, wantItems)
		}
		if row.ParseError != nil {
			t.Errorf("row %s parse_error = %q, want NULL", id, *row.ParseError)
		}
		if row.Reason == nil || *row.Reason != wantReason {
			reason := "<nil>"
			if row.Reason != nil {
				reason = *row.Reason
			}
			t.Errorf("row %s reason = %q, want %q", id, reason, wantReason)
		}
	}

	// Assert: confirmed row is unchanged.
	rowConf := fetchPendingRow(t, idConfirmed)
	if rowConf.Reason == nil || *rowConf.Reason != "already_confirmed" {
		t.Errorf("confirmed row reason = %v, want 'already_confirmed' (must be unchanged)", rowConf.Reason)
	}

	// Assert: discarded row is unchanged.
	rowDisc := fetchPendingRow(t, idDiscarded)
	if rowDisc.Reason == nil || *rowDisc.Reason != "already_discarded" {
		t.Errorf("discarded row reason = %v, want 'already_discarded' (must be unchanged)", rowDisc.Reason)
	}

	// Assert: a receipt_sync_runs row was inserted with triggered_by='reprocess_all'.
	var triggeredBy string
	err := testPool.QueryRow(t.Context(),
		`SELECT triggered_by FROM receipt_sync_runs ORDER BY started_at DESC LIMIT 1`).Scan(&triggeredBy)
	if err != nil {
		t.Fatalf("select receipt_sync_runs: %v", err)
	}
	if triggeredBy != "reprocess_all" {
		t.Errorf("triggered_by = %q, want 'reprocess_all'", triggeredBy)
	}

	// Wait for the goroutine to finish so it doesn't race with the next test's TRUNCATE.
	syncID, _ := body["sync_id"].(float64)
	if syncID > 0 {
		waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	}
}

// TestReprocessAllPendingHandler_ReturnsConflictWhenSyncAlreadyRunning verifies
// that when a sync run is already in flight, the handler returns 409 with
// error='sync_already_running'. The UPDATE is allowed to have already run
// (consistent with the non-transactional semantics: reset first, then insert).
func TestReprocessAllPendingHandler_ReturnsConflictWhenSyncAlreadyRunning(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	// Pre-insert a running sync row to trigger the conflict.
	_, err := testPool.Exec(t.Context(),
		`INSERT INTO receipt_sync_runs (status, triggered_by) VALUES ('running', 'manual')`)
	if err != nil {
		t.Fatalf("pre-insert running sync row: %v", err)
	}

	// Seed one stuck row so we can verify it was (or wasn't) touched.
	_ = insertStuckPendingPurchase(t, "stuck-conflict", "Receipt total $X doesn't match...", true, false)

	runner := stubRunnerOK(receipt.IngestResult{})
	handler := ReprocessAllPendingHandler(testPool, runner)

	req := httptest.NewRequest("POST", "/inventory/purchases/reprocess-all", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != 409 {
		t.Fatalf("POST reprocess-all: status=%d body=%s want 409", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "sync_already_running" {
		t.Errorf("error = %v, want 'sync_already_running'", body["error"])
	}
}
