package receipt

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

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
		"", // parseError empty: no-attachment branch never attempts parse
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
		"",
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
			"",
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
		"",
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
	if err := createPurchaseEvent(t.Context(), testPool, attachedTx, items, summary, "", false /* isUpgrade */); err != nil {
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

// ─── Phase 260607-co0: upgrade path for stale no_attachment_on_bank_tx rows ──
//
// Background: When a Mercury card swipe lands without a receipt attachment,
// the worker writes a pending_purchases row with reason='no_attachment_on_bank_tx'.
// Before this phase, the worker's 2-way bankTxIDExists guard treated that row
// as "already ingested" forever, so if the user later uploaded a receipt in
// Mercury, the worker still short-circuited and the line items stayed at
// $0.00. These tests cover the new 3-way classifyExistingTx + isUpgrade flow.

// installWorkerStubs swaps the package-level seams to controllable fakes for
// the duration of a single test. Each stub records whether it was invoked so
// tests can assert on "must not be called" paths (e.g. the cached short-circuit
// must NOT call parseReceipt).
type workerStubs struct {
	txns            []MercuryTransaction
	parseItems      []ReceiptItem
	parseSummary    ReceiptSummary
	parseErr        error
	parseCallCount  int
	sonnetItems     []ReceiptItem  // Phase 260607-e1c
	sonnetSummary   ReceiptSummary // Phase 260607-e1c
	sonnetErr       error          // Phase 260607-e1c
	sonnetCallCount int            // Phase 260607-e1c
	fetchCalled     bool
	dlCalled        bool
}

func installWorkerStubs(t *testing.T, s *workerStubs) {
	t.Helper()

	origFetch := fetchTransactions
	fetchTransactions = func(_ context.Context, _ string, _, _ time.Time) ([]MercuryTransaction, error) {
		s.fetchCalled = true
		return s.txns, nil
	}
	t.Cleanup(func() { fetchTransactions = origFetch })

	origParse := parseReceipt
	parseReceipt = func(_ context.Context, _ string, _ []byte, _ string) ([]ReceiptItem, ReceiptSummary, error) {
		s.parseCallCount++
		return s.parseItems, s.parseSummary, s.parseErr
	}
	t.Cleanup(func() { parseReceipt = origParse })

	// Phase 260607-e1c: Sonnet fallback seam — installed alongside parseReceipt
	// so each test can independently drive the (haiku ok / haiku fail+sonnet ok /
	// both fail) branches.
	origSonnet := parseReceiptWithSonnet
	parseReceiptWithSonnet = func(_ context.Context, _ string, _ []byte, _ string) ([]ReceiptItem, ReceiptSummary, error) {
		s.sonnetCallCount++
		return s.sonnetItems, s.sonnetSummary, s.sonnetErr
	}
	t.Cleanup(func() { parseReceiptWithSonnet = origSonnet })

	origDL := downloadReceiptFileFn
	downloadReceiptFileFn = func(_ context.Context, _ string) ([]byte, string, error) {
		s.dlCalled = true
		return []byte("FAKE-RECEIPT-BYTES"), "image/jpeg", nil
	}
	t.Cleanup(func() { downloadReceiptFileFn = origDL })
}

// TestRunIngestCycle_UpgradesPendingNoAttachmentRow exercises the happy upgrade
// path: a pending row with reason='no_attachment_on_bank_tx' exists, the
// Mercury tx now has an attachment, parse succeeds → the pending row is
// DELETED and a purchase_events row is INSERTED in one DB transaction.
func TestRunIngestCycle_UpgradesPendingNoAttachmentRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Seed an existing no-attachment pending row, simulating a prior worker
	// poll that saw the unreceipted card swipe. Mercury debits are NEGATIVE
	// values — bank_total mirrors what the no-attachment branch stored.
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
		VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb)`,
		"T-upgrade-ok", -42.50, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Mercury Amount is negative (debit). ValidateReceiptData requires
	// summary.Total == -bankAmount AND sum(item.Quantity) == TotalUnits+TotalCases.
	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-upgrade-ok",
			Amount:      -42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
		parseItems: []ReceiptItem{
			{Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
		},
		parseSummary: ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1, TotalCases: 0},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey:   "stub",
		AnthropicAPIKey: "stub",
		Pool:            testPool,
		LookbackDays:    14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.AutoCreated != 1 {
		t.Errorf("AutoCreated = %d, want 1", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}
	if result.Cached != 0 {
		t.Errorf("Cached = %d, want 0", result.Cached)
	}

	var pendingCount, eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id='T-upgrade-ok'`,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='T-upgrade-ok'`,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending rows = %d, want 0 (upgrade should DELETE the row)", pendingCount)
	}
	if eventCount != 1 {
		t.Errorf("event rows = %d, want 1 (upgrade should INSERT the event)", eventCount)
	}

	var lineCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_line_items pli
		   JOIN purchase_events pe ON pe.id = pli.purchase_event_id
		  WHERE pe.bank_tx_id = 'T-upgrade-ok' AND pli.description = 'Salmon'`,
	).Scan(&lineCount); err != nil {
		t.Fatalf("count line items: %v", err)
	}
	if lineCount != 1 {
		t.Errorf("line items = %d, want 1", lineCount)
	}
}

// TestRunIngestCycle_UpgradesPendingNoAttachmentRow_ParseFails exercises the
// upgrade path when the receipt parse fails: the pending row's UUID is
// PRESERVED via UPDATE (no ON CONFLICT skip, no orphan), the reason now
// reflects the parser failure, and counts roll into PendingReview.
func TestRunIngestCycle_UpgradesPendingNoAttachmentRow_ParseFails(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
		VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb)`,
		"T-upgrade-fail", 42.50, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Capture the seeded row's UUID PK so we can assert the UPDATE path
	// preserved it rather than DELETE+re-INSERT.
	var origID string
	if err := testPool.QueryRow(t.Context(),
		`SELECT id::text FROM pending_purchases WHERE bank_tx_id='T-upgrade-fail'`,
	).Scan(&origID); err != nil {
		t.Fatalf("read original id: %v", err)
	}

	// Phase 260607-e1c: Sonnet fallback now retries Haiku failures, so this
	// test must drive BOTH stubs to error to keep the path landing on the
	// parse-fail routePending branch (vs Sonnet's empty summary passing through
	// to ValidateReceiptData and producing a validate-fail reason).
	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-upgrade-fail",
			Amount:      42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
		parseErr:  errors.New("haiku timeout"),
		sonnetErr: errors.New("sonnet timeout"),
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey: "stub", AnthropicAPIKey: "stub",
		Pool: testPool, LookbackDays: 14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.PendingReview != 1 {
		t.Errorf("PendingReview = %d, want 1", result.PendingReview)
	}
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
	}
	if result.Cached != 0 {
		t.Errorf("Cached = %d, want 0", result.Cached)
	}

	var gotID, gotReason string
	if err := testPool.QueryRow(t.Context(),
		`SELECT id::text, COALESCE(reason,'') FROM pending_purchases WHERE bank_tx_id='T-upgrade-fail'`,
	).Scan(&gotID, &gotReason); err != nil {
		t.Fatalf("read updated row: %v", err)
	}
	if gotID != origID {
		t.Errorf("pending row id changed: was %q, now %q (upgrade should UPDATE, not DELETE+INSERT)", origID, gotID)
	}
	if gotReason != "Receipt could not be parsed automatically" {
		t.Errorf("reason = %q, want %q", gotReason, "Receipt could not be parsed automatically")
	}
}

// TestRunIngestCycle_SkipsRealCached asserts that pending rows with any
// reason other than 'no_attachment_on_bank_tx' (e.g. a prior parse failure
// the human still needs to review) still short-circuit as Cached, even when
// the tx has attachments. parseReceipt must NOT be called for cached rows.
//
// Phase 260607-fxl: parse_error must be populated on the seed so the new
// parseFailedRetry gate (parse_error IS NULL AND items empty) does NOT fire.
// Without parse_error set, this row would now retry through Sonnet — the
// "real cached" intent here is "BOTH models already failed, human-review
// territory" so parse_error is the correct signal.
func TestRunIngestCycle_SkipsRealCached(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
		VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb, 'haiku boom; sonnet boom')`,
		"T-real-cached", 19.99, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-real-cached",
			Amount:      19.99,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey: "stub", AnthropicAPIKey: "stub",
		Pool: testPool, LookbackDays: 14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.Cached != 1 {
		t.Errorf("Cached = %d, want 1", result.Cached)
	}
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}
	if stubs.parseCallCount != 0 {
		t.Errorf("parseReceipt called %d times, want 0 (cached path must short-circuit before parse)", stubs.parseCallCount)
	}

	var reason string
	if err := testPool.QueryRow(t.Context(),
		`SELECT COALESCE(reason,'') FROM pending_purchases WHERE bank_tx_id='T-real-cached'`,
	).Scan(&reason); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if reason != "Receipt could not be parsed automatically" {
		t.Errorf("reason = %q, want unchanged", reason)
	}
}

// ─── Phase 260607-fxl: parse-failed retry gate ──────────────────────────────

// TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull covers the
// happy path of the new parseFailedRetry branch. A pre-260607-e1c row that hit
// Haiku-fails-no-Sonnet (reason='Receipt could not be parsed automatically',
// parse_error=NULL, items='[]') gets retried — Haiku fails, Sonnet succeeds,
// the row is DELETEd and a purchase_events row is INSERTed atomically.
func TestRunIngestCycle_RetriesPriorHaikuFailureWhenParseErrorNull(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Negative amount = debit (food spend). Pre-e1c rows stored bank_total
	// the same way.
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
		VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb, NULL)`,
		"T-retry-ok", -42.50, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-retry-ok",
			Amount:      -42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
		parseErr: errors.New("haiku boom"),
		sonnetItems: []ReceiptItem{
			{Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
		},
		sonnetSummary: ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1, TotalCases: 0},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey:   "stub",
		AnthropicAPIKey: "stub",
		Pool:            testPool,
		LookbackDays:    14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.AutoCreated != 1 {
		t.Errorf("AutoCreated = %d, want 1", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}
	if result.Cached != 0 {
		t.Errorf("Cached = %d, want 0", result.Cached)
	}
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1 (Haiku should be tried once)", stubs.parseCallCount)
	}
	if stubs.sonnetCallCount != 1 {
		t.Errorf("sonnetCallCount = %d, want 1 (Sonnet fallback should fire after Haiku fails)", stubs.sonnetCallCount)
	}

	var pendingCount, eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id='T-retry-ok'`,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='T-retry-ok'`,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending rows = %d, want 0 (retry success should DELETE)", pendingCount)
	}
	if eventCount != 1 {
		t.Errorf("event rows = %d, want 1", eventCount)
	}
}

// TestRunIngestCycle_DoesNotRetryWhenParseErrorSet asserts the parse-failed
// retry gate's primary guard: once parse_error is populated (meaning BOTH
// Haiku and Sonnet already failed in a previous run), the row STAYS cached
// — no infinite parse loop. parseReceipt / parseReceiptWithSonnet must NOT
// be invoked, and the download seam must NOT be hit either.
func TestRunIngestCycle_DoesNotRetryWhenParseErrorSet(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
		VALUES ($1, $2, $3, 'Receipt could not be parsed automatically', '[]'::jsonb, 'haiku boom; sonnet boom')`,
		"T-no-retry-err", -42.50, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-no-retry-err",
			Amount:      -42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.Cached != 1 {
		t.Errorf("Cached = %d, want 1", result.Cached)
	}
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
	}
	if stubs.parseCallCount != 0 {
		t.Errorf("parseCallCount = %d, want 0 (cached row must not call Haiku)", stubs.parseCallCount)
	}
	if stubs.sonnetCallCount != 0 {
		t.Errorf("sonnetCallCount = %d, want 0 (cached row must not call Sonnet)", stubs.sonnetCallCount)
	}
	if stubs.dlCalled {
		t.Errorf("dlCalled = true, want false (cached row must not download)")
	}
}

// TestRunIngestCycle_DoesNotRetryParseFailedWithItems asserts the second
// guard: a parse-failed row with parse_error=NULL but items populated (user
// has started editing the row manually) is NOT retried — the operator's edits
// must not be clobbered by a worker re-parse.
func TestRunIngestCycle_DoesNotRetryParseFailedWithItems(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, parse_error)
		VALUES ($1, $2, $3, 'Receipt could not be parsed automatically',
		        '[{"name":"foo","quantity":1,"price":1.0,"is_case":false}]'::jsonb, NULL)`,
		"T-no-retry-items", -42.50, "STUB",
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	var origItems string
	if err := testPool.QueryRow(t.Context(),
		`SELECT items::text FROM pending_purchases WHERE bank_tx_id='T-no-retry-items'`,
	).Scan(&origItems); err != nil {
		t.Fatalf("read original items: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-no-retry-items",
			Amount:      -42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey: "stub", AnthropicAPIKey: "stub", Pool: testPool, LookbackDays: 14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.Cached != 1 {
		t.Errorf("Cached = %d, want 1", result.Cached)
	}
	if stubs.parseCallCount != 0 {
		t.Errorf("parseCallCount = %d, want 0 (user-edited row must not be reparsed)", stubs.parseCallCount)
	}

	var gotItems string
	if err := testPool.QueryRow(t.Context(),
		`SELECT items::text FROM pending_purchases WHERE bank_tx_id='T-no-retry-items'`,
	).Scan(&gotItems); err != nil {
		t.Fatalf("read items after run: %v", err)
	}
	if gotItems != origItems {
		t.Errorf("items changed: was %q, now %q (worker must not clobber user edits)", origItems, gotItems)
	}
}

// TestRunIngestCycle_SkipsExistingPurchaseEvent asserts that a tx already in
// purchase_events short-circuits as Cached regardless of whether it now has
// attachments. parseReceipt must NOT be called.
func TestRunIngestCycle_SkipsExistingPurchaseEvent(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Direct INSERT (not createPurchaseEvent) so the fixture doesn't couple
	// to the upgrade refactor we're testing.
	var vendorID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO vendors (name) VALUES ($1) RETURNING id`,
		"Existing Vendor",
	).Scan(&vendorID); err != nil {
		t.Fatalf("seed vendor: %v", err)
	}
	if _, err := testPool.Exec(t.Context(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, total)
		 VALUES ($1, $2, '2026-05-27', $3)`,
		vendorID, "T-existing-event", 50.00,
	); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-existing-event",
			Amount:      50.00,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey: "stub", AnthropicAPIKey: "stub",
		Pool: testPool, LookbackDays: 14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.Cached != 1 {
		t.Errorf("Cached = %d, want 1", result.Cached)
	}
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}
	if stubs.parseCallCount != 0 {
		t.Errorf("parseReceipt called %d times, want 0", stubs.parseCallCount)
	}

	var pendingCount, eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id='T-existing-event'`,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='T-existing-event'`,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending rows = %d, want 0 (cached event must not create pending)", pendingCount)
	}
	if eventCount != 1 {
		t.Errorf("event rows = %d, want 1 (no duplicate)", eventCount)
	}
}

// TestClassifyExistingTx is a pure unit test (no worker stubs) for the
// 3-way classifyExistingTx helper that replaces bankTxIDExists.
func TestClassifyExistingTx(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	t.Run("empty DB returns none", func(t *testing.T) {
		resetReceiptFixtures(t)
		kind, reason, _, _, err := classifyExistingTx(t.Context(), testPool, "no-such-tx")
		if err != nil {
			t.Fatalf("classify: %v", err)
		}
		if kind != "none" {
			t.Errorf("kind = %q, want %q", kind, "none")
		}
		if reason != "" {
			t.Errorf("reason = %q, want \"\"", reason)
		}
	})

	t.Run("pending no_attachment row → pending", func(t *testing.T) {
		resetReceiptFixtures(t)
		if _, err := testPool.Exec(t.Context(), `
			INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
			VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb)`,
			"T-cls-pending", 10.00, "STUB",
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		kind, reason, _, _, err := classifyExistingTx(t.Context(), testPool, "T-cls-pending")
		if err != nil {
			t.Fatalf("classify: %v", err)
		}
		if kind != "pending" {
			t.Errorf("kind = %q, want %q", kind, "pending")
		}
		if reason != "no_attachment_on_bank_tx" {
			t.Errorf("reason = %q, want %q", reason, "no_attachment_on_bank_tx")
		}
	})

	t.Run("discarded pending row → none (re-ingest allowed)", func(t *testing.T) {
		resetReceiptFixtures(t)
		if _, err := testPool.Exec(t.Context(), `
			INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, discarded_at)
			VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb, now())`,
			"T-cls-discarded", 10.00, "STUB",
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		kind, _, _, _, err := classifyExistingTx(t.Context(), testPool, "T-cls-discarded")
		if err != nil {
			t.Fatalf("classify: %v", err)
		}
		if kind != "none" {
			t.Errorf("kind = %q, want %q (discarded rows MUST NOT block re-ingest)", kind, "none")
		}
	})

	t.Run("confirmed pending row → event (idempotent like locked purchase)", func(t *testing.T) {
		resetReceiptFixtures(t)
		// confirmed_by references users(id); leave NULL — confirmed_at alone
		// is sufficient for the WHERE filter.
		if _, err := testPool.Exec(t.Context(), `
			INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items, confirmed_at)
			VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb, now())`,
			"T-cls-confirmed", 10.00, "STUB",
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		kind, _, _, _, err := classifyExistingTx(t.Context(), testPool, "T-cls-confirmed")
		if err != nil {
			t.Fatalf("classify: %v", err)
		}
		if kind != "event" {
			t.Errorf("kind = %q, want %q (confirmed pending behaves like event)", kind, "event")
		}
	})

	t.Run("purchase_events row → event", func(t *testing.T) {
		resetReceiptFixtures(t)
		var vendorID string
		if err := testPool.QueryRow(t.Context(),
			`INSERT INTO vendors (name) VALUES ('Cls Vendor') RETURNING id`,
		).Scan(&vendorID); err != nil {
			t.Fatalf("seed vendor: %v", err)
		}
		if _, err := testPool.Exec(t.Context(),
			`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, total)
			 VALUES ($1, $2, '2026-05-27', $3)`,
			vendorID, "T-cls-event", 25.00,
		); err != nil {
			t.Fatalf("seed event: %v", err)
		}
		kind, reason, _, _, err := classifyExistingTx(t.Context(), testPool, "T-cls-event")
		if err != nil {
			t.Fatalf("classify: %v", err)
		}
		if kind != "event" {
			t.Errorf("kind = %q, want %q", kind, "event")
		}
		if reason != "" {
			t.Errorf("reason = %q, want \"\"", reason)
		}
	})

	t.Run("legacy items='null' scalar does not error", func(t *testing.T) {
		// Regression: pre-260607-dg9 rows have items stored as the JSON literal
		// `null` (a scalar, not an array). jsonb_array_length(null::jsonb) raises
		// SQLSTATE 22023, so the items-empty check must guard via jsonb_typeof.
		resetReceiptFixtures(t)
		if _, err := testPool.Exec(t.Context(), `
			INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
			VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', 'null'::jsonb)`,
			"T-cls-legacy-null", 10.00, "STUB",
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		kind, reason, _, hasItems, err := classifyExistingTx(t.Context(), testPool, "T-cls-legacy-null")
		if err != nil {
			t.Fatalf("classify must not error on legacy items='null' rows: %v", err)
		}
		if kind != "pending" {
			t.Errorf("kind = %q, want %q", kind, "pending")
		}
		if reason != "no_attachment_on_bank_tx" {
			t.Errorf("reason = %q, want %q", reason, "no_attachment_on_bank_tx")
		}
		if hasItems {
			t.Errorf("hasItems = true, want false (JSON scalar null is not a non-empty array)")
		}
	})
}

// ─── Phase 260607-dg9: partial unique index + items-nil dedupe regression ────

// TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason proves that
// when a second insert with the SAME bank_tx_id but a DIFFERENT reason hits
// the partial unique index, ON CONFLICT DO NOTHING preserves the ORIGINAL
// reason (guards against accidentally switching to DO UPDATE).
func TestInsertPendingPurchase_DedupesOnReinsertWithDifferentReason(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:        "tx-dedupe-reason",
		Amount:    25.00,
		CreatedAt: "2026-05-27T10:00:00Z",
	}

	// First insert with reason A.
	if err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, ReceiptSummary{}, "",
		"no_attachment_on_bank_tx",
		"",
	); err != nil {
		t.Fatalf("first insertPendingPurchase: %v", err)
	}

	// Second insert with reason B — must be a no-op due to partial unique index.
	if err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, ReceiptSummary{}, "",
		"parse_failed",
		"",
	); err != nil {
		t.Fatalf("second insertPendingPurchase: %v", err)
	}

	var count int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`,
		"tx-dedupe-reason",
	).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("pending_purchases rows = %d, want 1", count)
	}

	var reason sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT reason FROM pending_purchases WHERE bank_tx_id = $1`,
		"tx-dedupe-reason",
	).Scan(&reason); err != nil {
		t.Fatalf("select reason: %v", err)
	}
	if !reason.Valid || reason.String != "no_attachment_on_bank_tx" {
		t.Errorf("reason = %+v, want %q (ON CONFLICT DO NOTHING must preserve original)", reason, "no_attachment_on_bank_tx")
	}
}

// TestMigration_DedupesExistingPendingDuplicates proves the dedupe CTE in
// migration 0068 removes duplicate active pending rows (keeping latest
// created_at) and that the partial unique index then blocks new duplicates.
// Because TestMain auto-applies all migrations before tests run, this test
// manually drops the partial unique index, seeds duplicates, re-runs the
// dedupe CTE, then re-creates the index — exercising the same SQL the
// migration runs in production on its first apply.
func TestMigration_DedupesExistingPendingDuplicates(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)
	ctx := t.Context()

	// 1. Drop the partial unique index so we can seed duplicates.
	if _, err := testPool.Exec(ctx,
		`DROP INDEX IF EXISTS pending_purchases_bank_tx_id_uniq`,
	); err != nil {
		t.Fatalf("drop index: %v", err)
	}

	// 2. Seed two active pending rows with the same bank_tx_id — older
	//    created_at first, then newer. Dedupe must keep the newer.
	if _, err := testPool.Exec(ctx,
		`INSERT INTO pending_purchases
		 (bank_tx_id, bank_total, vendor, reason, items, created_at)
		 VALUES
		 ($1, 10.00, 'VendorA', 'reason_older', '[]'::jsonb, '2026-05-25T10:00:00Z'),
		 ($1, 10.00, 'VendorB', 'reason_newer', '[]'::jsonb, '2026-05-27T10:00:00Z')`,
		"tx-mig-dupe",
	); err != nil {
		t.Fatalf("seed duplicates: %v", err)
	}

	// 3. Re-run the migration's dedupe CTE (must match migration 0068 exactly).
	if _, err := testPool.Exec(ctx,
		`WITH ranked AS (
		   SELECT id,
		          ROW_NUMBER() OVER (
		            PARTITION BY bank_tx_id
		            ORDER BY created_at DESC, id DESC
		          ) AS rn
		     FROM pending_purchases
		    WHERE confirmed_at IS NULL
		      AND discarded_at IS NULL
		 )
		 DELETE FROM pending_purchases
		  WHERE id IN (SELECT id FROM ranked WHERE rn > 1)`,
	); err != nil {
		t.Fatalf("dedupe CTE: %v", err)
	}

	// 4. Re-create the partial unique index — same DDL as migration 0068.
	if _, err := testPool.Exec(ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS pending_purchases_bank_tx_id_uniq
		   ON pending_purchases(bank_tx_id)
		   WHERE confirmed_at IS NULL AND discarded_at IS NULL`,
	); err != nil {
		t.Fatalf("create unique index: %v", err)
	}

	// 5. Assert exactly one row remains and it's the newer one.
	var count int
	var reason sql.NullString
	if err := testPool.QueryRow(ctx,
		`SELECT COUNT(*), MAX(reason) FROM pending_purchases WHERE bank_tx_id = $1`,
		"tx-mig-dupe",
	).Scan(&count, &reason); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows after dedupe = %d, want 1", count)
	}
	if !reason.Valid || reason.String != "reason_newer" {
		t.Errorf("kept row reason = %+v, want %q (dedupe must keep latest created_at)", reason, "reason_newer")
	}

	// 6. Assert the partial unique index now blocks a fresh duplicate insert.
	//    ON CONFLICT DO NOTHING (with the partial-index target) must
	//    silently absorb the duplicate. Use the same ON CONFLICT clause
	//    the worker uses to prove parity.
	cmdTag, err := testPool.Exec(ctx,
		`INSERT INTO pending_purchases
		 (bank_tx_id, bank_total, vendor, items)
		 VALUES ($1, 10.00, 'VendorC', '[]'::jsonb)
		 ON CONFLICT (bank_tx_id) WHERE confirmed_at IS NULL AND discarded_at IS NULL DO NOTHING`,
		"tx-mig-dupe",
	)
	if err != nil {
		t.Fatalf("conflict insert: %v", err)
	}
	if cmdTag.RowsAffected() != 0 {
		t.Errorf("rows affected on conflict insert = %d, want 0 (partial unique index must block)", cmdTag.RowsAffected())
	}

	// 7. And confirmed/discarded rows should NOT conflict — the partial
	//    predicate excludes them. Mark the existing row discarded, then a
	//    fresh active row for the same bank_tx_id should succeed.
	if _, err := testPool.Exec(ctx,
		`UPDATE pending_purchases SET discarded_at = now() WHERE bank_tx_id = $1`,
		"tx-mig-dupe",
	); err != nil {
		t.Fatalf("mark discarded: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`INSERT INTO pending_purchases
		 (bank_tx_id, bank_total, vendor, items)
		 VALUES ($1, 10.00, 'VendorD', '[]'::jsonb)`,
		"tx-mig-dupe",
	); err != nil {
		t.Fatalf("re-insert after discard should succeed (partial predicate excludes discarded): %v", err)
	}
}

// ─── Phase 260607-e1c: Sonnet fallback + persist parse_error ────────────────
//
// Cover the new (haiku→sonnet) retry seam. Three scenarios:
//   1. Haiku fails, Sonnet succeeds → auto-create proceeds normally, no pending row.
//   2. Haiku fails, Sonnet ALSO fails → pending row created with parse_error
//      containing BOTH "haiku" and "sonnet" substrings.
//   3. insertPendingPurchase called with parseError="" leaves the column NULL
//      (default-NULL contract).

// TestRunIngestCycle_FallsBackToSonnet exercises the happy fallback path:
// Haiku errs, Sonnet returns valid items+summary → ValidateReceiptData passes,
// createPurchaseEvent runs, AutoCreated=1, no pending row.
func TestRunIngestCycle_FallsBackToSonnet(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Mercury debits are NEGATIVE values; ValidateReceiptData requires
	// summary.Total == -bankAmount AND sum(item.Quantity) == TotalUnits+TotalCases.
	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-sonnet-ok",
			Amount:      -42.50,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
		}},
		parseErr: errors.New("haiku timeout"),
		// Sonnet returns valid output — fall through to validate+create.
		sonnetItems: []ReceiptItem{
			{Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
		},
		sonnetSummary: ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1, TotalCases: 0},
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey:   "stub",
		AnthropicAPIKey: "stub",
		Pool:            testPool,
		LookbackDays:    14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.AutoCreated != 1 {
		t.Errorf("AutoCreated = %d, want 1", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1 (Haiku must be tried first)", stubs.parseCallCount)
	}
	if stubs.sonnetCallCount != 1 {
		t.Errorf("sonnetCallCount = %d, want 1 (Sonnet must be tried after Haiku fail)", stubs.sonnetCallCount)
	}

	var pendingCount, eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id='T-sonnet-ok'`,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='T-sonnet-ok'`,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending rows = %d, want 0 (Sonnet success must NOT route to pending)", pendingCount)
	}
	if eventCount != 1 {
		t.Errorf("event rows = %d, want 1 (Sonnet success must auto-create event)", eventCount)
	}
}

// TestRunIngestCycle_BothModelsFail_StoresParseError covers the (haiku→sonnet)
// double-fail path: pending row created with parse_error containing both
// error strings concatenated as "haiku: <h>; sonnet: <s>".
func TestRunIngestCycle_BothModelsFail_StoresParseError(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:          "T-both-fail",
			Amount:      -391.96,
			CreatedAt:   "2026-05-27T10:00:00Z",
			Attachments: []Attachment{{URL: "http://fake/r.pdf", FileName: "r.pdf"}},
		}},
		parseErr:  errors.New("haiku boom"),
		sonnetErr: errors.New("sonnet boom"),
	}
	installWorkerStubs(t, stubs)

	result, err := runIngestCycle(t.Context(), WorkerConfig{
		MercuryAPIKey:   "stub",
		AnthropicAPIKey: "stub",
		Pool:            testPool,
		LookbackDays:    14,
	})
	if err != nil {
		t.Fatalf("runIngestCycle: %v", err)
	}

	if result.PendingReview != 1 {
		t.Errorf("PendingReview = %d, want 1", result.PendingReview)
	}
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
	}
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1", stubs.parseCallCount)
	}
	if stubs.sonnetCallCount != 1 {
		t.Errorf("sonnetCallCount = %d, want 1", stubs.sonnetCallCount)
	}

	var parseError sql.NullString
	var reason sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT parse_error, reason FROM pending_purchases WHERE bank_tx_id='T-both-fail'`,
	).Scan(&parseError, &reason); err != nil {
		t.Fatalf("select pending: %v", err)
	}
	if !parseError.Valid {
		t.Fatalf("parse_error = NULL, want non-null with haiku+sonnet errors")
	}
	pe := parseError.String
	for _, want := range []string{"haiku", "sonnet", "boom"} {
		if !strings.Contains(pe, want) {
			t.Errorf("parse_error %q missing substring %q", pe, want)
		}
	}
	if !reason.Valid || reason.String != "Receipt could not be parsed automatically" {
		t.Errorf("reason = %+v, want parse-fail sentinel", reason)
	}
}

// TestInsertPendingPurchase_ParseErrorNullByDefault asserts the (default,
// no-parse-attempt) write leaves parse_error column NULL. Mirrors the
// no-attachment branch's call shape with parseError="".
func TestInsertPendingPurchase_ParseErrorNullByDefault(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	tx := MercuryTransaction{
		ID:        "tx-default-null",
		Amount:    19.99,
		CreatedAt: "2026-05-27T10:00:00Z",
	}
	if err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, ReceiptSummary{}, "",
		"no_attachment_on_bank_tx",
		"", // parseError empty — must land as NULL on the column.
	); err != nil {
		t.Fatalf("insertPendingPurchase: %v", err)
	}

	var ns sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT parse_error FROM pending_purchases WHERE bank_tx_id = $1`,
		"tx-default-null",
	).Scan(&ns); err != nil {
		t.Fatalf("select: %v", err)
	}
	if ns.Valid {
		t.Errorf("parse_error = %q, want NULL (empty parseError must not be stored)", ns.String)
	}
}

// TestRunIngestCycle_ScenarioTable exercises the worker's parse → validate
// → persist branches end-to-end with stubbed Mercury + Anthropic seams.
//
// Each case drives a fresh single-transaction ingest cycle and asserts the
// resulting purchase_events / pending_purchases rows match the expected
// shape. Companion to the existing TestRunIngestCycle_* tests — those
// cover the upgrade/no-attachment edges; this covers the parse/validate
// happy/fallback/failure matrix.
//
// Phase 260607-k1n: motivated by the decimal-quantity Restaurant Depot
// bug. The both_fail_decimal_qty case locks in coverage of the exact error
// string the worker logged so a regression in error-propagation surfaces
// here.
func TestRunIngestCycle_ScenarioTable(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	// Mercury debits are NEGATIVE; ValidateReceiptData requires
	// summary.Total == -bankAmount AND sum(item.Quantity) == TotalUnits+TotalCases.
	// Use unique bank_tx_ids per case so cases are independent if t.Parallel()
	// is later enabled.
	validItems := []ReceiptItem{
		{Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
	}
	validSummary := ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50, TotalUnits: 1, TotalCases: 0}

	// The exact unmarshal error string from the production log
	// (262e21 / Restaurant Depot, 2026-06-07). Locks in error-text
	// preservation through the haiku+sonnet seam.
	decimalQtyErrText := "json: cannot unmarshal number 40.0 into Go struct field ReceiptItem.items.quantity of type int"

	cases := []struct {
		name string
		// stubs setup
		parseItems    []ReceiptItem
		parseSummary  ReceiptSummary
		parseErr      error
		sonnetItems   []ReceiptItem
		sonnetSummary ReceiptSummary
		sonnetErr     error
		// tx setup
		bankTxID string
		amount   float64 // Mercury debit (negative)
		// expectations
		wantAutoCreated   int
		wantPendingReview int
		wantPendingReason string   // substring match
		wantParseErrParts []string // substrings that MUST appear in pending.parse_error (empty list = column should be NULL)
		wantSonnetCalled  bool
	}{
		{
			name:            "happy_haiku_succeeds",
			parseItems:      validItems,
			parseSummary:    validSummary,
			bankTxID:        "T-scenario-haiku-ok",
			amount:          -42.50,
			wantAutoCreated: 1,
		},
		{
			name:             "haiku_fails_sonnet_recovers",
			parseErr:         fmt.Errorf("ParseReceipt: failed to parse JSON body: %s", decimalQtyErrText),
			sonnetItems:      validItems,
			sonnetSummary:    validSummary,
			bankTxID:         "T-scenario-sonnet-recovers",
			amount:           -42.50,
			wantAutoCreated:  1,
			wantSonnetCalled: true,
		},
		{
			name:              "both_fail_with_realistic_errors",
			parseErr:          fmt.Errorf("ParseReceipt: API call failed: 529 overloaded"),
			sonnetErr:         fmt.Errorf("ParseReceiptWithSonnet: failed to parse JSON body: invalid character 'x'"),
			bankTxID:          "T-scenario-both-fail",
			amount:            -42.50,
			wantPendingReview: 1,
			wantPendingReason: "Receipt could not be parsed automatically",
			wantParseErrParts: []string{"haiku:", "sonnet:", "529 overloaded", "invalid character"},
			wantSonnetCalled:  true,
		},
		{
			name:              "both_fail_decimal_qty",
			parseErr:          fmt.Errorf("ParseReceipt: failed to parse JSON body: parseJSONBody: %s", decimalQtyErrText),
			sonnetErr:         fmt.Errorf("ParseReceiptWithSonnet: failed to parse JSON body: parseJSONBody: %s", decimalQtyErrText),
			bankTxID:          "T-scenario-decimal-qty",
			amount:            -391.96,
			wantPendingReview: 1,
			wantPendingReason: "Receipt could not be parsed automatically",
			// Locks in that today's exact error substring is preserved end-to-end.
			wantParseErrParts: []string{"haiku:", "sonnet:", "40.0", "type int"},
			wantSonnetCalled:  true,
		},
		{
			name:              "total_mismatch",
			parseItems:        validItems,
			parseSummary:      ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 99.99, TotalUnits: 1, TotalCases: 0}, // doesn't match -bankAmount
			bankTxID:          "T-scenario-total-mismatch",
			amount:            -42.50,
			wantPendingReview: 1,
			wantPendingReason: "does not match", // matches Check 1 or Check 2 reason text
			wantParseErrParts: nil,              // parse_error should be NULL on validate-fail
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resetReceiptFixtures(t)

			stubs := &workerStubs{
				txns: []MercuryTransaction{{
					ID:          tc.bankTxID,
					Amount:      tc.amount,
					CreatedAt:   "2026-06-07T10:00:00Z",
					Attachments: []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}},
				}},
				parseItems:    tc.parseItems,
				parseSummary:  tc.parseSummary,
				parseErr:      tc.parseErr,
				sonnetItems:   tc.sonnetItems,
				sonnetSummary: tc.sonnetSummary,
				sonnetErr:     tc.sonnetErr,
			}
			installWorkerStubs(t, stubs)

			result, err := runIngestCycle(t.Context(), WorkerConfig{
				MercuryAPIKey:   "stub",
				AnthropicAPIKey: "stub",
				Pool:            testPool,
				LookbackDays:    14,
			})
			if err != nil {
				t.Fatalf("runIngestCycle: %v", err)
			}

			if result.AutoCreated != tc.wantAutoCreated {
				t.Errorf("AutoCreated = %d, want %d", result.AutoCreated, tc.wantAutoCreated)
			}
			if result.PendingReview != tc.wantPendingReview {
				t.Errorf("PendingReview = %d, want %d", result.PendingReview, tc.wantPendingReview)
			}
			if tc.wantSonnetCalled && stubs.sonnetCallCount == 0 {
				t.Errorf("expected Sonnet to be called, got %d calls", stubs.sonnetCallCount)
			}
			if !tc.wantSonnetCalled && stubs.sonnetCallCount > 0 {
				t.Errorf("expected Sonnet NOT to be called, got %d calls", stubs.sonnetCallCount)
			}

			if tc.wantPendingReview > 0 {
				var reason sql.NullString
				var parseError sql.NullString
				if err := testPool.QueryRow(t.Context(),
					`SELECT reason, parse_error FROM pending_purchases WHERE bank_tx_id=$1`,
					tc.bankTxID,
				).Scan(&reason, &parseError); err != nil {
					t.Fatalf("query pending row: %v", err)
				}
				if !strings.Contains(reason.String, tc.wantPendingReason) {
					t.Errorf("pending reason %q does not contain %q", reason.String, tc.wantPendingReason)
				}
				if len(tc.wantParseErrParts) == 0 {
					// validate-fail path: parse_error column should be NULL
					if parseError.Valid && parseError.String != "" {
						t.Errorf("expected parse_error NULL, got %q", parseError.String)
					}
				} else {
					if !parseError.Valid {
						t.Errorf("expected parse_error populated, got NULL")
					}
					for _, part := range tc.wantParseErrParts {
						if !strings.Contains(parseError.String, part) {
							t.Errorf("parse_error %q missing substring %q", parseError.String, part)
						}
					}
				}
			}

			if tc.wantAutoCreated > 0 {
				var n int
				if err := testPool.QueryRow(t.Context(),
					`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id=$1`,
					tc.bankTxID,
				).Scan(&n); err != nil {
					t.Fatalf("count purchase_events: %v", err)
				}
				if n != 1 {
					t.Errorf("purchase_events count = %d, want 1", n)
				}
			}
		})
	}
}
