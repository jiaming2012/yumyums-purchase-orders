package inventory

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sort"
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
	return insertStuckPendingPurchaseWithURL(t, bankTxID, reason, withItems, withParseError, "")
}

// insertStuckPendingPurchaseWithURL is like insertStuckPendingPurchase but
// also sets receipt_url so the row is eligible for Spaces-based reprocess.
func insertStuckPendingPurchaseWithURL(t *testing.T, bankTxID, reason string, withItems bool, withParseError bool, receiptURL string) string {
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
	var receiptURLVal *string
	if receiptURL != "" {
		receiptURLVal = &receiptURL
	}
	q := `INSERT INTO pending_purchases
	        (bank_tx_id, bank_total, vendor, items, reason, parse_error, receipt_url, created_at)
	      VALUES ($1, 0, 'TestVendor', ` + itemsJSON + `, $2, $3, $4, now())
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID, reason, parseErr, receiptURLVal).Scan(&id); err != nil {
		t.Fatalf("insertStuckPendingPurchaseWithURL %s: %v", bankTxID, err)
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
// result for all rows without any real Spaces/Anthropic calls.
func stubBatchRunnerOK(result string) BatchReprocessRunner {
	return func(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error) {
		out := make(map[string]string, len(rows))
		for _, row := range rows {
			out[row.BankTxID] = result
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

	// Seed 3 stuck rows with receipt URLs so they pass the Spaces-reprocess filter.
	_ = insertStuckPendingPurchaseWithURL(t, "stuck-a", "Receipt total $42.00 doesn't match...", true, false, "https://spaces.example.com/r/stuck-a.jpg")
	_ = insertStuckPendingPurchaseWithURL(t, "stuck-b", "Line item sum $X doesn't match...", false, true, "https://spaces.example.com/r/stuck-b.jpg")
	_ = insertStuckPendingPurchaseWithURL(t, "stuck-c", "item count 3 doesn't match...", false, false, "https://spaces.example.com/r/stuck-c.jpg")

	// Seed 1 confirmed row — must NOT be passed to the runner.
	insertConfirmedPendingPurchase(t, "confirmed-1")

	// Seed 1 discarded row — must NOT be passed to the runner.
	insertDiscardedPendingPurchase(t, "discarded-1")

	// Record which bank_tx_ids the batch runner receives (via rows).
	var calledIDs []string
	runner := BatchReprocessRunner(func(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error) {
		for _, row := range rows {
			calledIDs = append(calledIDs, row.BankTxID)
		}
		out := make(map[string]string, len(rows))
		for _, row := range rows {
			out[row.BankTxID] = "pending_review"
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

// TestReprocessAllPendingHandler_SelectsAndRoutesRows verifies the new
// Spaces-based reprocess handler:
//   - Row with a single legacy receipt_url → wrapped to 1-element list and passed to runner
//   - Row with multi-URL receipt_urls JSONB → passed with all URLs to runner
//   - Row with no receipt_url (can't reprocess from storage) → filtered out before runner
//
// The runner stub records the rows it receives so we can assert on URL shape.
func TestReprocessAllPendingHandler_SelectsAndRoutesRows(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetPendingPurchases(t)
	resetSyncRuns(t)

	// Row 1: legacy single-URL (receipt_url set, receipt_urls NULL).
	const legacyTxID = "tx-legacy-url"
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url)
		VALUES ($1, -10.00, 'LegacyVendor', '[]'::jsonb, 'parse_failed', $2)`,
		legacyTxID, "https://spaces.example.com/receipts/legacy.jpg",
	); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// Row 2: multi-URL row (receipt_urls JSONB set, receipt_url also set to first URL).
	const multiTxID = "tx-multi-url"
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url, receipt_urls)
		VALUES ($1, -20.00, 'MultiVendor', '[]'::jsonb, 'parse_failed',
		        'https://spaces.example.com/receipts/multi-0.jpg',
		        '["https://spaces.example.com/receipts/multi-0.jpg","https://spaces.example.com/receipts/multi-1.jpg"]'::jsonb)`,
		multiTxID,
	); err != nil {
		t.Fatalf("seed multi-url row: %v", err)
	}

	// Row 3: no receipt_url — must be filtered out.
	const noURLTxID = "tx-no-url"
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason)
		VALUES ($1, -5.00, 'NoURLVendor', '[]'::jsonb, 'no_attachment_on_bank_tx')`,
		noURLTxID,
	); err != nil {
		t.Fatalf("seed no-url row: %v", err)
	}

	// Stub runner: record received rows for inspection.
	var receivedRows []receipt.PendingRowForReprocess
	runner := BatchReprocessRunner(func(ctx context.Context, rows []receipt.PendingRowForReprocess) (map[string]string, error) {
		receivedRows = append(receivedRows, rows...)
		out := make(map[string]string, len(rows))
		for _, row := range rows {
			out[row.BankTxID] = "pending_review"
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

	// pending_count reflects only rows with URLs (2, not 3).
	pendingCount, _ := body["pending_count"].(float64)
	if pendingCount != 2 {
		t.Errorf("pending_count = %v, want 2 (no-URL row must be excluded)", pendingCount)
	}

	syncID, _ := body["sync_id"].(float64)
	if syncID > 0 {
		waitForTerminalStatus(t, int64(syncID), 2*time.Second)
	}

	// Runner must have received exactly 2 rows (no-URL row filtered out).
	if len(receivedRows) != 2 {
		t.Fatalf("runner received %d rows, want 2", len(receivedRows))
	}

	// Assert no-URL row was NOT passed to runner.
	for _, row := range receivedRows {
		if row.BankTxID == noURLTxID {
			t.Errorf("runner received no-URL row %q — must be filtered before runner call", noURLTxID)
		}
	}

	// Build a map for easy assertion.
	rowMap := make(map[string]receipt.PendingRowForReprocess, len(receivedRows))
	for _, row := range receivedRows {
		rowMap[row.BankTxID] = row
	}

	// Legacy row: should arrive with a 1-element URL list.
	if legacyRow, ok := rowMap[legacyTxID]; !ok {
		t.Errorf("legacy row %q not found in runner input", legacyTxID)
	} else {
		if len(legacyRow.ReceiptURLs) != 1 {
			t.Errorf("legacy row ReceiptURLs len = %d, want 1", len(legacyRow.ReceiptURLs))
		} else if legacyRow.ReceiptURLs[0] != "https://spaces.example.com/receipts/legacy.jpg" {
			t.Errorf("legacy row ReceiptURLs[0] = %q, want legacy URL", legacyRow.ReceiptURLs[0])
		}
	}

	// Multi-URL row: should arrive with both URLs.
	if multiRow, ok := rowMap[multiTxID]; !ok {
		t.Errorf("multi-url row %q not found in runner input", multiTxID)
	} else {
		if len(multiRow.ReceiptURLs) != 2 {
			t.Errorf("multi-url row ReceiptURLs len = %d, want 2", len(multiRow.ReceiptURLs))
		} else {
			got := make([]string, len(multiRow.ReceiptURLs))
			copy(got, multiRow.ReceiptURLs)
			sort.Strings(got)
			want := []string{
				"https://spaces.example.com/receipts/multi-0.jpg",
				"https://spaces.example.com/receipts/multi-1.jpg",
			}
			for i, w := range want {
				if got[i] != w {
					t.Errorf("multi-url ReceiptURLs[%d] = %q, want %q", i, got[i], w)
				}
			}
		}
	}
}
