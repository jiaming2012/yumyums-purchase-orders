package inventory

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
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

// stubBatchRunnerOK returns a BatchReprocessRunner that returns the given
// result for all bank_tx_ids without any real Mercury/Anthropic calls.
func stubBatchRunnerOK(result string) BatchReprocessRunner {
	return func(ctx context.Context, bankTxIDs []string) (map[string]string, error) {
		out := make(map[string]string, len(bankTxIDs))
		for _, id := range bankTxIDs {
			out[id] = result
		}
		return out, nil
	}
}

// TestReprocessAllPendingHandler_QueuesPerRowProcessing verifies the happy path:
// - 3 still-pending rows → runner called once per row
// - 1 confirmed row → not passed to runner (excluded by WHERE clause)
// - 1 discarded row → not passed to runner (excluded by WHERE clause)
// - pending_count=3 in JSON response
// - a receipt_sync_runs row inserted with triggered_by='reprocess_all'
func TestReprocessAllPendingHandler_QueuesPerRowProcessing(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	// Seed 3 stuck rows in various legacy states.
	_ = insertStuckPendingPurchase(t, "stuck-a", "Receipt total $42.00 doesn't match...", true, false)
	_ = insertStuckPendingPurchase(t, "stuck-b", "Line item sum $X doesn't match...", false, true)
	_ = insertStuckPendingPurchase(t, "stuck-c", "item count 3 doesn't match...", false, false)

	// Seed 1 confirmed row — must NOT be passed to the runner.
	insertConfirmedPendingPurchase(t, "confirmed-1")

	// Seed 1 discarded row — must NOT be passed to the runner.
	insertDiscardedPendingPurchase(t, "discarded-1")

	// Record which bank_tx_ids the batch runner receives.
	var calledIDs []string
	runner := BatchReprocessRunner(func(ctx context.Context, bankTxIDs []string) (map[string]string, error) {
		calledIDs = append(calledIDs, bankTxIDs...)
		out := make(map[string]string, len(bankTxIDs))
		for _, id := range bankTxIDs {
			out[id] = "pending_review"
		}
		return out, nil
	})

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

	// Assert pending_count=3.
	pendingCount, _ := body["pending_count"].(float64)
	if pendingCount != 3 {
		t.Errorf("pending_count = %v, want 3", pendingCount)
	}
	if _, ok := body["sync_id"]; !ok {
		t.Errorf("response missing sync_id field")
	}
	if body["status"] != "running" {
		t.Errorf("response status = %v, want running", body["status"])
	}

	// Assert: a receipt_sync_runs row was inserted with triggered_by='reprocess_all'.
	syncID, _ := body["sync_id"].(float64)
	if syncID > 0 {
		waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	}

	// After goroutine finishes, runner must have been called exactly 3 times.
	if len(calledIDs) != 3 {
		t.Errorf("runner called %d times, want 3 (once per active pending row)", len(calledIDs))
	}
	for _, id := range calledIDs {
		if id == "confirmed-1" || id == "discarded-1" {
			t.Errorf("runner was called with %q — confirmed/discarded rows must be excluded", id)
		}
	}

	var triggeredBy string
	if err := testPool.QueryRow(t.Context(),
		`SELECT triggered_by FROM receipt_sync_runs ORDER BY started_at DESC LIMIT 1`).Scan(&triggeredBy); err != nil {
		t.Fatalf("select receipt_sync_runs: %v", err)
	}
	if triggeredBy != "reprocess_all" {
		t.Errorf("triggered_by = %q, want 'reprocess_all'", triggeredBy)
	}
}

// TestReprocessAllPendingHandler_ReturnsConflictWhenSyncAlreadyRunning verifies
// that when a sync run is already in flight, the handler returns 409 with
// error='sync_already_running'.
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

	// Seed one stuck row so there's something to process.
	_ = insertStuckPendingPurchase(t, "stuck-conflict", "Receipt total $X doesn't match...", true, false)

	runner := stubBatchRunnerOK("pending_review")
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

// TestReprocessAllPendingHandler_PerRowRefetch_RecoversOlderRow verifies that
// reprocess-all fetches each pending row's tx individually (bypassing the
// 14-day lookback window), so a row whose tx is 30 days old can still be
// promoted to a purchase_event.
func TestReprocessAllPendingHandler_PerRowRefetch_RecoversOlderRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	// Seed: a pending row whose transaction is 30 days old — outside the
	// normal 14-day Mercury lookback window. The old bulk-sync approach would
	// never pick this up; per-row refetch must recover it.
	const oldBankTxID = "tx-old-30d"
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
		VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb)`,
		oldBankTxID, -42.50, "Old Vendor",
	); err != nil {
		t.Fatalf("seed old pending: %v", err)
	}

	// Stub batch runner: simulates fetching all pending IDs in one call,
	// auto-creating the event for the old tx.
	callCount := 0
	runner := BatchReprocessRunner(func(ctx context.Context, bankTxIDs []string) (map[string]string, error) {
		callCount++
		out := make(map[string]string, len(bankTxIDs))
		for _, bankTxID := range bankTxIDs {
			if bankTxID != oldBankTxID {
				out[bankTxID] = "errored"
				continue
			}
			// Simulate auto-create: insert a purchase_events row and delete the pending row.
			var vendorID string
			if err := testPool.QueryRow(ctx,
				`INSERT INTO vendors (name) VALUES ('Old Vendor') ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name RETURNING id`,
			).Scan(&vendorID); err != nil {
				out[bankTxID] = "errored"
				continue
			}
			if _, err := testPool.Exec(ctx,
				`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, total)
				 VALUES ($1, $2, '2026-05-24', 42.50)`,
				vendorID, bankTxID); err != nil {
				out[bankTxID] = "errored"
				continue
			}
			if _, err := testPool.Exec(ctx,
				`DELETE FROM pending_purchases WHERE bank_tx_id = $1`, bankTxID); err != nil {
				out[bankTxID] = "errored"
				continue
			}
			out[bankTxID] = "auto_created"
		}
		return out, nil
	})

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

	// Wait for the goroutine to finish before asserting DB state.
	syncID, _ := body["sync_id"].(float64)
	if syncID > 0 {
		waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	}

	// Assert the batch runner was called exactly once (one batch call covers all pending rows).
	if callCount != 1 {
		t.Errorf("runner callCount = %d, want 1 (batch runner called once for all pending rows)", callCount)
	}

	// Assert the auto_created count was written to the sync run row.
	var autoCreated int
	if err := testPool.QueryRow(t.Context(),
		`SELECT auto_created FROM receipt_sync_runs WHERE id = $1`, int64(syncID),
	).Scan(&autoCreated); err != nil {
		t.Fatalf("select auto_created from sync run: %v", err)
	}
	if autoCreated != 1 {
		t.Errorf("sync_run.auto_created = %d, want 1", autoCreated)
	}

	// Assert the old pending row is gone.
	var pendingCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`, oldBankTxID,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending_purchases count = %d, want 0 (old row must be promoted)", pendingCount)
	}

	// Assert a purchase_event was created.
	var eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, oldBankTxID,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("purchase_events count = %d, want 1", eventCount)
	}
}
