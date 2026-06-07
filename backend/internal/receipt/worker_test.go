package receipt

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		// Cannot reach DB — leave testPool nil so individual tests skip.
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		// Migration failure is a hard error — abort.
		panic("db.Migrate failed: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func resetReceiptFixtures(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(t.Context(),
		`TRUNCATE purchase_line_items, purchase_events, pending_purchases,
		          purchase_items, vendors RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults asserts the DB
// write produced by the no-attachment worker branch matches the spec: empty
// items array, NULL parsed total, reason sentinel set, bank_total carried
// through, event_date derived from CreatedAt.
func TestInsertPendingPurchase_NoAttachmentBranch_ShapeAndDefaults(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:              "tx-noatt-1",
		Amount:          42.50,
		BankDescription: "RESTAURANT DEPOT 0123 CHICAGO IL",
		CreatedAt:       "2026-05-27T10:00:00Z",
	}
	err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, ReceiptSummary{}, "",
		"no_attachment_on_bank_tx",
	)
	if err != nil {
		t.Fatalf("insertPendingPurchase: %v", err)
	}

	var (
		bankTxID  string
		bankTotal float64
		items     string
		total     sql.NullFloat64
		reason    sql.NullString
		vendor    sql.NullString
		eventDate sql.NullString
	)
	err = testPool.QueryRow(t.Context(),
		`SELECT bank_tx_id, bank_total, items::text, total, reason, vendor, event_date::text
		   FROM pending_purchases WHERE bank_tx_id = $1`, "tx-noatt-1",
	).Scan(&bankTxID, &bankTotal, &items, &total, &reason, &vendor, &eventDate)
	if err != nil {
		t.Fatalf("select pending_purchase: %v", err)
	}

	if bankTxID != "tx-noatt-1" {
		t.Errorf("bank_tx_id = %q, want %q", bankTxID, "tx-noatt-1")
	}
	if bankTotal != 42.50 {
		t.Errorf("bank_total = %v, want 42.50", bankTotal)
	}
	if items != "[]" {
		t.Errorf("items = %q, want %q", items, "[]")
	}
	if total.Valid {
		t.Errorf("total = %v, want NULL", total.Float64)
	}
	if !reason.Valid || reason.String != "no_attachment_on_bank_tx" {
		t.Errorf("reason = %+v, want valid %q", reason, "no_attachment_on_bank_tx")
	}
	// Phase 260606-hew: empty summary.Vendor falls back to tx.BankDescription
	// so unreceipted card swipes no longer render as "Unknown Vendor".
	if !vendor.Valid || vendor.String != "RESTAURANT DEPOT 0123 CHICAGO IL" {
		t.Errorf("vendor = %+v, want %q (BankDescription fallback)", vendor, "RESTAURANT DEPOT 0123 CHICAGO IL")
	}
	if !eventDate.Valid || eventDate.String != "2026-05-27" {
		t.Errorf("event_date = %+v, want 2026-05-27", eventDate)
	}
}

// TestInsertPendingPurchase_VendorFallback_PrefersSummary asserts that when
// the receipt parser populated summary.Vendor (curated name from the image),
// the fallback does NOT overwrite it with Mercury's raw bankDescription.
func TestInsertPendingPurchase_VendorFallback_PrefersSummary(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:              "tx-parsed-1",
		Amount:          25.00,
		BankDescription: "ACME FOOD CO 0123 CHICAGO IL",
		CreatedAt:       "2026-05-27T10:00:00Z",
	}
	summary := ReceiptSummary{Vendor: "Acme Foods", Tax: 0, Total: 25.00}
	if err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, summary, "",
		"some_reason",
	); err != nil {
		t.Fatalf("insertPendingPurchase: %v", err)
	}

	var vendor sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT vendor FROM pending_purchases WHERE bank_tx_id = $1`, "tx-parsed-1",
	).Scan(&vendor); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !vendor.Valid || vendor.String != "Acme Foods" {
		t.Errorf("vendor = %+v, want %q (curated name beats bank string)", vendor, "Acme Foods")
	}
}

// TestBackfillPendingVendor_SetsWhenEmpty asserts the re-poll backfill
// helper populates pending_purchases.vendor from tx.BankDescription for
// rows whose vendor is the empty string. Mirrors what the worker loop
// does on every poll for cached transactions within the lookback
// window. Production rows hit this case because the no-attachment
// branch used to write summary.Vendor == "" before this phase.
// (Schema has vendor NOT NULL — empty string, not NULL, is what
// pre-260606-hew rows actually contain.)
func TestBackfillPendingVendor_SetsWhenEmpty(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Simulate a pre-260606-hew row: vendor is '' because the no-att
	// branch handed in ReceiptSummary{} at the time.
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items)
		 VALUES ($1, $2, '', '[]'::jsonb)`,
		"tx-backfill-empty", -391.96,
	); err != nil {
		t.Fatalf("seed empty-vendor pending: %v", err)
	}

	tx := MercuryTransaction{ID: "tx-backfill-empty", BankDescription: "RESTAURANT DEPOT"}
	if err := backfillPendingVendor(t.Context(), testPool, tx); err != nil {
		t.Fatalf("backfillPendingVendor: %v", err)
	}

	var got sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT vendor FROM pending_purchases WHERE bank_tx_id = $1`, "tx-backfill-empty",
	).Scan(&got); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !got.Valid || got.String != "RESTAURANT DEPOT" {
		t.Errorf("vendor = %+v, want %q", got, "RESTAURANT DEPOT")
	}
}

// TestBackfillPendingVendor_DoesNotOverwriteExisting asserts the
// IS NULL OR = '' guard protects a receipt-parsed pending whose vendor
// Claude (or a human) already set.
func TestBackfillPendingVendor_DoesNotOverwriteExisting(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items)
		 VALUES ($1, $2, 'Acme Foods', '[]'::jsonb)`,
		"tx-curated", -50.00,
	); err != nil {
		t.Fatalf("seed curated-vendor pending: %v", err)
	}

	tx := MercuryTransaction{ID: "tx-curated", BankDescription: "ACME FOOD CO 0123 CHICAGO IL"}
	if err := backfillPendingVendor(t.Context(), testPool, tx); err != nil {
		t.Fatalf("backfillPendingVendor: %v", err)
	}

	var vendor sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT vendor FROM pending_purchases WHERE bank_tx_id = $1`, "tx-curated",
	).Scan(&vendor); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !vendor.Valid || vendor.String != "Acme Foods" {
		t.Errorf("vendor = %+v, want %q (curated name must NOT be overwritten)", vendor, "Acme Foods")
	}
}

// TestBackfillPendingVendor_EmptyBankDescriptionIsNoOp asserts the helper
// short-circuits when tx.BankDescription is empty — leaves the row's
// existing vendor untouched. (Schema has vendor NOT NULL, so we seed ''.)
func TestBackfillPendingVendor_EmptyBankDescriptionIsNoOp(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items)
		 VALUES ($1, $2, '', '[]'::jsonb)`,
		"tx-empty-bd", -10.00,
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	tx := MercuryTransaction{ID: "tx-empty-bd", BankDescription: ""}
	if err := backfillPendingVendor(t.Context(), testPool, tx); err != nil {
		t.Fatalf("backfillPendingVendor: %v", err)
	}

	var vendor sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT vendor FROM pending_purchases WHERE bank_tx_id = $1`, "tx-empty-bd",
	).Scan(&vendor); err != nil {
		t.Fatalf("select: %v", err)
	}
	if !vendor.Valid || vendor.String != "" {
		t.Errorf("vendor = %+v, want \"\" (empty BankDescription must not overwrite to anything else)", vendor)
	}
}

// TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun asserts the
// ON CONFLICT DO NOTHING guard on bank_tx_id keeps re-polls from duplicating
// the no-attachment row.
func TestInsertPendingPurchase_NoAttachmentBranch_IdempotentOnRerun(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:        "tx-noatt-rerun",
		Amount:    19.99,
		CreatedAt: "2026-05-27T10:00:00Z",
	}
	for i := 0; i < 2; i++ {
		err := insertPendingPurchase(
			t.Context(), testPool, tx,
			nil, ReceiptSummary{}, "",
			"no_attachment_on_bank_tx",
		)
		if err != nil {
			t.Fatalf("insertPendingPurchase call %d: %v", i+1, err)
		}
	}

	var count int
	err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`,
		"tx-noatt-rerun",
	).Scan(&count)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("pending_purchases rows = %d, want 1", count)
	}
}

// TestRunIngestCycle_NoTransactions_ReturnsZeroResult asserts the refactored
// runIngestCycle signature: when the Mercury fetch returns an error (the only
// reachable early-exit path without a live Mercury API key), the result must be
// the zero IngestResult and err must be non-nil. This exercises the
// `return IngestResult{}, fmt.Errorf("…")` path on FetchTransactions failure.
func TestRunIngestCycle_NoTransactions_ReturnsZeroResult(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Empty MercuryAPIKey triggers the Mercury API to reject the request,
	// causing FetchTransactions to return a non-nil error — the path we want
	// to verify returns the zero IngestResult.
	cfg := WorkerConfig{
		MercuryAPIKey: "",
		Pool:          testPool,
		LookbackDays:  14,
	}
	result, err := runIngestCycle(t.Context(), cfg)
	if err == nil {
		t.Fatalf("runIngestCycle: expected error from empty MercuryAPIKey, got nil")
	}
	if result != (IngestResult{}) {
		t.Errorf("runIngestCycle: result = %+v, want zero IngestResult on error", result)
	}
}

// TestInsertPendingPurchase_CoexistsWithAttachmentBranch asserts the
// no-attachment branch (pending_purchases insert) and the parsed-receipt
// branch (purchase_events insert via createPurchaseEvent) do not contaminate
// each other's tables for distinct bank_tx_ids.
func TestInsertPendingPurchase_CoexistsWithAttachmentBranch(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// No-attachment row.
	noattTx := MercuryTransaction{
		ID:        "tx-noatt-2",
		Amount:    15.00,
		CreatedAt: "2026-05-27T10:00:00Z",
	}
	if err := insertPendingPurchase(
		t.Context(), testPool, noattTx,
		nil, ReceiptSummary{}, "",
		"no_attachment_on_bank_tx",
	); err != nil {
		t.Fatalf("insertPendingPurchase: %v", err)
	}

	// Parsed-receipt row → purchase_event via createPurchaseEvent.
	attachedTx := MercuryTransaction{
		ID:        "tx-attached-1",
		Amount:    10.50,
		CreatedAt: "2026-05-28T10:00:00Z",
	}
	items := []ReceiptItem{
		{Name: "Salmon", Quantity: 1, Price: 10.00, IsCase: false},
	}
	summary := ReceiptSummary{Vendor: "Acme", Tax: 0.50, Total: 10.50}
	if err := createPurchaseEvent(t.Context(), testPool, attachedTx, items, summary, ""); err != nil {
		t.Fatalf("createPurchaseEvent: %v", err)
	}

	// Coexistence assertions.
	checks := []struct {
		name   string
		query  string
		txID   string
		expect int
	}{
		{"pending_purchases has noatt row", `SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`, "tx-noatt-2", 1},
		{"purchase_events has attached row", `SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, "tx-attached-1", 1},
		{"no pending row for attached tx", `SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`, "tx-attached-1", 0},
		{"no event row for noatt tx", `SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, "tx-noatt-2", 0},
	}
	for _, c := range checks {
		var got int
		if err := testPool.QueryRow(t.Context(), c.query, c.txID).Scan(&got); err != nil {
			t.Fatalf("%s: query: %v", c.name, err)
		}
		if got != c.expect {
			t.Errorf("%s: got %d, want %d", c.name, got, c.expect)
		}
	}
}
