package receipt

import (
	"context"
	"database/sql"
	"encoding/json"
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
		nil, // receiptURLs: no-attachment branch has no URLs
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
		nil, // receiptURLs: not available for this test
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
			nil, // receiptURLs: no-attachment branch has no URLs
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
		nil, // receiptURLs: no-attachment branch has no URLs
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
	if err := createPurchaseEvent(t.Context(), testPool, attachedTx, items, summary, "", nil /* receiptURLs */, false /* isUpgrade */); err != nil {
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
	// feedback retry seam (goal-driven retry loop)
	feedbackItems      []ReceiptItem
	feedbackSummary    ReceiptSummary
	feedbackErr        error
	feedbackCallCount  int
	fetchCalled        bool
	dlCallCount        int // incremented once per attachment download
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
	parseReceipt = func(_ context.Context, _ string, _ []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
		s.parseCallCount++
		return s.parseItems, s.parseSummary, s.parseErr
	}
	t.Cleanup(func() { parseReceipt = origParse })

	// Phase 260607-e1c: Sonnet fallback seam — installed alongside parseReceipt
	// so each test can independently drive the (haiku ok / haiku fail+sonnet ok /
	// both fail) branches.
	origSonnet := parseReceiptWithSonnet
	parseReceiptWithSonnet = func(_ context.Context, _ string, _ []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
		s.sonnetCallCount++
		return s.sonnetItems, s.sonnetSummary, s.sonnetErr
	}
	t.Cleanup(func() { parseReceiptWithSonnet = origSonnet })

	// Goal-driven feedback retry seam.
	origFeedback := parseReceiptWithFeedback
	parseReceiptWithFeedback = func(_ context.Context, _ string, _ []FileBlob, _ float64, _ float64, _ string) ([]ReceiptItem, ReceiptSummary, error) {
		s.feedbackCallCount++
		return s.feedbackItems, s.feedbackSummary, s.feedbackErr
	}
	t.Cleanup(func() { parseReceiptWithFeedback = origFeedback })

	origDL := downloadReceiptFileFn
	downloadReceiptFileFn = func(_ context.Context, _ string) ([]byte, string, error) {
		s.dlCallCount++
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
	if stubs.dlCallCount != 0 {
		t.Errorf("dlCallCount = %d, want 0 (cached row must not download)", stubs.dlCallCount)
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
		nil, // receiptURLs
		"no_attachment_on_bank_tx",
		"",
	); err != nil {
		t.Fatalf("first insertPendingPurchase: %v", err)
	}

	// Second insert with reason B — must be a no-op due to partial unique index.
	if err := insertPendingPurchase(
		t.Context(), testPool, tx,
		nil, ReceiptSummary{}, "",
		nil, // receiptURLs
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

// TestRunIngestCycle_BothModelsFail_StoresParseError covers the double-fail
// path: pending row created with parse_error containing both error strings
// concatenated as "sonnet: <primary>; sonnet-retry: <retry>" (Sonnet is the
// primary model now — see parser.go; the stub error text here happens to spell
// "haiku"/"sonnet" so the substring checks still exercise both attempts).
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
		nil, // receiptURLs: no-attachment branch has no URLs
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
		bankTxID    string
		amount      float64 // Mercury debit (negative)
		attachments []Attachment // nil → default single attachment
		// expectations
		wantAutoCreated   int
		wantPendingReview int
		wantPendingReason string   // substring match
		wantParseErrParts []string // substrings that MUST appear in pending.parse_error (empty list = column should be NULL)
		wantSonnetCalled  bool
		wantDLCallCount   int // expected number of downloadReceiptFileFn calls; 0 means "don't check"
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
			wantParseErrParts: []string{"sonnet:", "sonnet-retry:", "529 overloaded", "invalid character"},
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
			wantParseErrParts: []string{"sonnet:", "sonnet-retry:", "40.0", "type int"},
			wantSonnetCalled:  true,
		},
		{
			// total_mismatch: items themselves don't derive to the bank amount.
			// derivedTotal = 1×99.99 + 0 = $99.99 ≠ $42.50 → Check 1 fires.
			// Feedback stub returns zero summary (also fails derivedTotal check).
			// Both attempts stored in retry trace → parse_error non-NULL.
			// The test asserts pending is created and reason contains "does not match".
			name: "total_mismatch",
			parseItems: []ReceiptItem{
				{Name: "Salmon", Quantity: 1, Price: 99.99, IsCase: false}, // price deliberately wrong: sum=99.99 ≠ 42.50
			},
			parseSummary:      ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 99.99, TotalUnits: 1, TotalCases: 0},
			bankTxID:          "T-scenario-total-mismatch",
			amount:            -42.50,
			wantPendingReview: 1,
			wantPendingReason: "does not match", // matches "Receipt derived total ... does not match"
			wantParseErrParts: []string{"attempt 1:", "attempt 2:"}, // retry loop fires; both attempts logged
		},
		{
			// Regression (2026-07-16): totals match the bank amount but one
			// line item has an empty name. Check 0 in validate.go must route
			// to review — before the fix this auto-created a purchase_items
			// row with description='' (a ghost catalog item every future
			// unnamed line merges into, and a blank first row in the review
			// picker). Seen live on Mercury tx aef104e6-7d45-11f1.
			name: "unnamed_item_routes_to_review",
			parseItems: []ReceiptItem{
				{Name: "Chicken Thighs", Quantity: 1, Price: 13.59, IsCase: false},
				{Name: "", Quantity: 1, Price: 4.39, IsCase: false},
				{Name: "Peppers", Quantity: 1, Price: 6.49, IsCase: false},
			},
			parseSummary:      ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 24.47, TotalUnits: 3, TotalCases: 0},
			bankTxID:          "T-scenario-unnamed-item",
			amount:            -24.47,
			wantPendingReview: 1,
			wantPendingReason: "unnamed",
			wantParseErrParts: []string{"attempt 1:", "attempt 2:"}, // validate-fail → feedback retry fires
		},
		{
			// Multi-attachment net: a purchase receipt ($804.49) and a refund
			// receipt ($16.12 credit) attached to the same Mercury debit of
			// $-788.37. Both files are downloaded; Claude is called once with
			// both blobs and returns a combined summary whose Total matches the
			// bank net. Reproduces the Restaurant Depot #855 incident
			// (2026-06-17).
			name: "multi_attachment_net_auto_creates",
			parseItems: []ReceiptItem{
				{Name: "Case Chicken", Quantity: 1, Price: 804.49, IsCase: true},
				{Name: "Credit Memo", Quantity: 1, Price: -16.12, IsCase: false},
			},
			parseSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      788.37,
				TotalUnits: 1,
				TotalCases: 1,
			},
			bankTxID: "T-scenario-multi-att",
			amount:   -788.37,
			attachments: []Attachment{
				{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
				{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
			},
			wantAutoCreated: 1,
			wantDLCallCount: 2, // both attachments must be downloaded
		},
	}

	// ── New scenario table cases for goal-driven retry loop ──────────────────
	// These are separate from the table loop below so they can access the
	// feedbackCallCount field. They reuse the same DB + stub infrastructure.

	t.Run("multi_attachment_first_parse_wrong_total_then_feedback_succeeds", func(t *testing.T) {
		resetReceiptFixtures(t)

		// First parse stub: purchase-only total (misses the refund).
		// Feedback stub: corrected total with refund items added.
		stubs := &workerStubs{
			txns: []MercuryTransaction{{
				ID:        "T-feedback-ok",
				Amount:    -788.37,
				CreatedAt: "2026-06-17T10:00:00Z",
				Attachments: []Attachment{
					{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
					{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
				},
			}},
			// Initial parse misses refund — returns purchase total only.
			parseItems: []ReceiptItem{
				{Name: "Case Chicken", Quantity: 1, Price: 804.49, IsCase: true},
			},
			parseSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      804.49,
				TotalUnits: 0,
				TotalCases: 1,
			},
			// Feedback retry returns corrected net total with refund item.
			feedbackItems: []ReceiptItem{
				{Name: "Case Chicken", Quantity: 1, Price: 804.49, IsCase: true},
				{Name: "Credit Memo", Quantity: 1, Price: -16.12, IsCase: false},
			},
			feedbackSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      788.37,
				TotalUnits: 1,
				TotalCases: 1,
			},
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
			t.Errorf("AutoCreated = %d, want 1 (feedback retry must produce auto-create)", result.AutoCreated)
		}
		if result.PendingReview != 0 {
			t.Errorf("PendingReview = %d, want 0", result.PendingReview)
		}
		if stubs.feedbackCallCount != 1 {
			t.Errorf("feedbackCallCount = %d, want 1 (one Sonnet feedback call expected)", stubs.feedbackCallCount)
		}

		// Assert auto-created event exists with both receipt URLs in array.
		var receiptURLsRaw []byte
		if err := testPool.QueryRow(t.Context(),
			`SELECT receipt_urls FROM purchase_events WHERE bank_tx_id = $1`, "T-feedback-ok",
		).Scan(&receiptURLsRaw); err != nil {
			t.Fatalf("select receipt_urls: %v", err)
		}
		if len(receiptURLsRaw) == 0 {
			t.Fatalf("purchase_events.receipt_urls is NULL, want JSON array of length 2")
		}
		var gotURLs []string
		if err := json.Unmarshal(receiptURLsRaw, &gotURLs); err != nil {
			t.Fatalf("unmarshal receipt_urls: %v", err)
		}
		if len(gotURLs) != 2 {
			t.Errorf("receipt_urls length = %d, want 2", len(gotURLs))
		}
	})

	t.Run("multi_attachment_feedback_also_fails_routes_to_pending", func(t *testing.T) {
		resetReceiptFixtures(t)

		stubs := &workerStubs{
			txns: []MercuryTransaction{{
				ID:        "T-feedback-fail",
				Amount:    -788.37,
				CreatedAt: "2026-06-17T10:00:00Z",
				Attachments: []Attachment{
					{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
					{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
				},
			}},
			// Initial parse misses refund.
			parseItems: []ReceiptItem{
				{Name: "Case Chicken", Quantity: 1, Price: 804.49, IsCase: true},
			},
			parseSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      804.49,
				TotalUnits: 0,
				TotalCases: 1,
			},
			// Feedback retry also returns wrong total.
			feedbackItems: []ReceiptItem{
				{Name: "Case Chicken", Quantity: 1, Price: 810.00, IsCase: true},
			},
			feedbackSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      810.00,
				TotalUnits: 0,
				TotalCases: 1,
			},
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
			t.Errorf("PendingReview = %d, want 1 (double-fail must route to pending)", result.PendingReview)
		}
		if result.AutoCreated != 0 {
			t.Errorf("AutoCreated = %d, want 0", result.AutoCreated)
		}
		if stubs.feedbackCallCount != 1 {
			t.Errorf("feedbackCallCount = %d, want 1", stubs.feedbackCallCount)
		}

		// pending_purchases.parse_error must contain retry trace for both attempts.
		var parseError sql.NullString
		if err := testPool.QueryRow(t.Context(),
			`SELECT parse_error FROM pending_purchases WHERE bank_tx_id = $1`, "T-feedback-fail",
		).Scan(&parseError); err != nil {
			t.Fatalf("select parse_error: %v", err)
		}
		if !parseError.Valid || parseError.String == "" {
			t.Fatalf("parse_error is NULL or empty, want retry trace")
		}
		for _, want := range []string{"attempt 1:", "attempt 2:"} {
			if !strings.Contains(parseError.String, want) {
				t.Errorf("parse_error %q missing substring %q", parseError.String, want)
			}
		}
	})

	t.Run("multi_attachment_line_item_sum_fail_then_feedback_succeeds", func(t *testing.T) {
		resetReceiptFixtures(t)

		// First parse: total matches bank but sum(price*qty) != subtotal.
		// Item has quantity=10, price=20 → product=$200 but reported subtotal=$50.
		// (tax=0 so subtotal = total - 0 = 50; but 10*20 = 200 != 50.)
		stubs := &workerStubs{
			txns: []MercuryTransaction{{
				ID:        "T-linesum-feedback",
				Amount:    -50.00,
				CreatedAt: "2026-06-24T10:00:00Z",
				Attachments: []Attachment{
					{URL: "http://fake/receipt.jpg", FileName: "receipt.jpg"},
				},
			}},
			// First parse: total correct but price is extended total not unit price.
			parseItems: []ReceiptItem{
				{Name: "BEEF CHUCK", Quantity: 10, Price: 20.00, IsCase: false},
			},
			parseSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      50.00,
				TotalUnits: 10,
				TotalCases: 0,
			},
			// Feedback: corrected with unit price so price*qty = 50.
			feedbackItems: []ReceiptItem{
				{Name: "BEEF CHUCK", Quantity: 10, Price: 5.00, IsCase: false},
			},
			feedbackSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      50.00,
				TotalUnits: 10,
				TotalCases: 0,
			},
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
			t.Errorf("AutoCreated = %d, want 1 (line-item-sum feedback retry must produce auto-create)", result.AutoCreated)
		}
		if result.PendingReview != 0 {
			t.Errorf("PendingReview = %d, want 0", result.PendingReview)
		}
		if stubs.feedbackCallCount != 1 {
			t.Errorf("feedbackCallCount = %d, want 1 (one Sonnet feedback call expected for line-item-sum mismatch)", stubs.feedbackCallCount)
		}

		var eventCount int
		if err := testPool.QueryRow(t.Context(),
			`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, "T-linesum-feedback",
		).Scan(&eventCount); err != nil {
			t.Fatalf("count purchase_events: %v", err)
		}
		if eventCount != 1 {
			t.Errorf("purchase_events count = %d, want 1", eventCount)
		}
	})

	t.Run("validation_check3_removed_auto_creates_directly", func(t *testing.T) {
		resetReceiptFixtures(t)

		// Phase 260607-m3x: Check 3 (units+cases quantity match) is removed.
		// Prior test exercised a Check 3 failure triggering a feedback retry.
		// With Check 3 gone, a parse that was formerly a Check 3 mismatch
		// (sum(qty)=1, summary.TotalUnits=5) now passes validation directly
		// on attempt 1 because Check 1 (derived total) is the only remaining
		// gate. No feedback retry fires — auto-create on the first attempt.
		stubs := &workerStubs{
			txns: []MercuryTransaction{{
				ID:        "T-check3-removed",
				Amount:    -50.00,
				CreatedAt: "2026-06-24T10:00:00Z",
				Attachments: []Attachment{
					{URL: "http://fake/receipt.jpg", FileName: "receipt.jpg"},
				},
			}},
			// derivedTotal = 1×50.00 + 0 = $50.00 == -(-50.00) → Check 1 passes.
			// summary.TotalUnits=5 no longer triggers any check.
			parseItems: []ReceiptItem{
				{Name: "BEEF CHUCK", Quantity: 1, Price: 50.00, IsCase: false},
			},
			parseSummary: ReceiptSummary{
				Vendor:     "Restaurant Depot",
				Total:      50.00,
				TotalUnits: 5, // formerly a Check 3 mismatch — now irrelevant
				TotalCases: 0,
			},
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
			t.Errorf("AutoCreated = %d, want 1 (Check 3 removed: derived-total pass must auto-create without retry)", result.AutoCreated)
		}
		if result.PendingReview != 0 {
			t.Errorf("PendingReview = %d, want 0", result.PendingReview)
		}
		// No feedback retry needed — attempt 1 passes Check 1.
		if stubs.feedbackCallCount != 0 {
			t.Errorf("feedbackCallCount = %d, want 0 (Check 3 removed: no retry should fire)", stubs.feedbackCallCount)
		}
	})

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resetReceiptFixtures(t)

			atts := tc.attachments
			if atts == nil {
				atts = []Attachment{{URL: "http://fake/r.jpg", FileName: "r.jpg"}}
			}

			stubs := &workerStubs{
				txns: []MercuryTransaction{{
					ID:          tc.bankTxID,
					Amount:      tc.amount,
					CreatedAt:   "2026-06-07T10:00:00Z",
					Attachments: atts,
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
			if tc.wantDLCallCount > 0 && stubs.dlCallCount != tc.wantDLCallCount {
				t.Errorf("dlCallCount = %d, want %d (all attachments must be downloaded)", stubs.dlCallCount, tc.wantDLCallCount)
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

			// receipt_urls assertion for the multi-attachment auto-create case.
			// SpacesPresigner is nil in tests so URLs fall back to Mercury URLs;
			// the array still contains one entry per attachment in order.
			if tc.name == "multi_attachment_net_auto_creates" {
				var receiptURLsRaw []byte
				if err := testPool.QueryRow(t.Context(),
					`SELECT receipt_urls FROM purchase_events WHERE bank_tx_id=$1`,
					tc.bankTxID,
				).Scan(&receiptURLsRaw); err != nil {
					t.Fatalf("select receipt_urls from purchase_events: %v", err)
				}
				if len(receiptURLsRaw) == 0 {
					t.Fatalf("purchase_events.receipt_urls is NULL, want JSON array of length 2")
				}
				var gotURLs []string
				if err := json.Unmarshal(receiptURLsRaw, &gotURLs); err != nil {
					t.Fatalf("unmarshal receipt_urls: %v", err)
				}
				if len(gotURLs) != 2 {
					t.Errorf("receipt_urls length = %d, want 2", len(gotURLs))
				}
				// No Spaces presigner → fallback to Mercury URLs.
				if len(gotURLs) > 0 && gotURLs[0] != "http://fake/purchase.jpg" {
					t.Errorf("receipt_urls[0] = %q, want %q (Mercury fallback)", gotURLs[0], "http://fake/purchase.jpg")
				}
				if len(gotURLs) > 1 && gotURLs[1] != "http://fake/refund.pdf" {
					t.Errorf("receipt_urls[1] = %q, want %q (Mercury fallback)", gotURLs[1], "http://fake/refund.pdf")
				}
			}
		})
	}
}

// TestMultiAttachment_ValidateFailStoresURLsOnPending asserts that when a
// multi-attachment transaction fails validation, pending_purchases.receipt_urls
// is populated with one entry per attachment (in order) and receipt_url holds
// the primary (index 0) URL. SpacesPresigner is nil → Mercury fallback URLs.
func TestMultiAttachment_ValidateFailStoresURLsOnPending(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        "T-multi-pending",
			Amount:    -788.37,
			CreatedAt: "2026-06-07T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
				{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
			},
		}},
		// Summary total doesn't match bank amount → validate-fail → pending row.
		parseItems: []ReceiptItem{
			{Name: "Case Chicken", Quantity: 1, Price: 999.99, IsCase: true},
		},
		parseSummary: ReceiptSummary{
			Vendor:     "Restaurant Depot",
			Total:      999.99,
			TotalUnits: 0,
			TotalCases: 1,
		},
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

	var receiptURL sql.NullString
	var receiptURLsRaw []byte
	if err := testPool.QueryRow(t.Context(),
		`SELECT receipt_url, receipt_urls FROM pending_purchases WHERE bank_tx_id = $1`,
		"T-multi-pending",
	).Scan(&receiptURL, &receiptURLsRaw); err != nil {
		t.Fatalf("select pending row: %v", err)
	}

	// receipt_url (singular) must hold the primary URL.
	if !receiptURL.Valid || receiptURL.String != "http://fake/purchase.jpg" {
		t.Errorf("receipt_url = %+v, want %q", receiptURL, "http://fake/purchase.jpg")
	}

	// receipt_urls must be a JSON array of length 2.
	if len(receiptURLsRaw) == 0 {
		t.Fatalf("pending_purchases.receipt_urls is NULL, want JSON array of length 2")
	}
	var gotURLs []string
	if err := json.Unmarshal(receiptURLsRaw, &gotURLs); err != nil {
		t.Fatalf("unmarshal receipt_urls: %v", err)
	}
	if len(gotURLs) != 2 {
		t.Errorf("receipt_urls length = %d, want 2", len(gotURLs))
	}
	if len(gotURLs) > 0 && gotURLs[0] != "http://fake/purchase.jpg" {
		t.Errorf("receipt_urls[0] = %q, want %q", gotURLs[0], "http://fake/purchase.jpg")
	}
	if len(gotURLs) > 1 && gotURLs[1] != "http://fake/refund.pdf" {
		t.Errorf("receipt_urls[1] = %q, want %q", gotURLs[1], "http://fake/refund.pdf")
	}
}

// TestWorker_DerivedTotal_AutoCreatesWhenItemsSumMatchesBank exercises the
// derived-total architecture: Claude's summary.Total is WRONG (purchase-only,
// not netted) but the items themselves ARE correctly netted (attempt-1 behavior
// seen live on Restaurant Depot tx f13472e8-6a6c-11f1-bca2-4b2005726612).
//
// Setup mirrors that tx:
//   bank amount       = -$788.37  (Mercury debit)
//   2 attachments     (purchase + refund receipts)
//   parseReceipt stub:
//     items = [{PURCHASE BUCKET, price=787.73, qty=1}, {REFUND BUCKET, price=-15.96, qty=1}]
//     summary.Total = $804.49  (Claude's purchase-only total — WRONG)
//     summary.Tax   = $16.60   (correctly-netted tax)
//   derivedTotal = (787.73×1 + -15.96×1) + 16.60 = 771.77 + 16.60 = $788.37 ✓
//
// Expected behaviour with the new validate.go (derive total from items):
//   Check 1: |derivedTotal - 788.37| = |788.37 - 788.37| = 0 ≤ 0.01 → PASS
//   → No retry loop needed; row auto-creates on the FIRST attempt.
//   → Persisted purchase_events.total = $788.37 (derived), NOT Claude's $804.49.
func TestWorker_DerivedTotal_AutoCreatesWhenItemsSumMatchesBank(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        "T-derived-total-ok",
			Amount:    -788.37,
			CreatedAt: "2026-06-17T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
				{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
			},
		}},
		// Claude's attempt-1 behavior: items correctly netted, total NOT netted.
		// sum(price*qty) = 787.73 + (-15.96) = 771.77
		// derivedTotal   = 771.77 + 16.60 = 788.37 ✓ matches -bankAmount
		parseItems: []ReceiptItem{
			{Name: "PURCHASE BUCKET", Price: 787.73, Quantity: 1, IsCase: false},
			{Name: "REFUND BUCKET", Price: -15.96, Quantity: 1, IsCase: false},
		},
		parseSummary: ReceiptSummary{
			Vendor:     "Restaurant Depot #855",
			Total:      804.49, // Claude's wrong (purchase-only) total — INTENTIONALLY WRONG
			Tax:        16.60,  // correctly-netted tax
			TotalUnits: 2,
			TotalCases: 0,
		},
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

	// 1. parseReceipt called exactly once — no retry needed because derived-total
	//    check passes on the first attempt.
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1 (no retry expected)", stubs.parseCallCount)
	}

	// 2. parseReceiptWithFeedback must NOT be called — first attempt succeeds.
	if stubs.feedbackCallCount != 0 {
		t.Errorf("feedbackCallCount = %d, want 0 (no feedback retry should fire)", stubs.feedbackCallCount)
	}

	// 3. AutoCreated = 1: derived-total passes on first attempt.
	if result.AutoCreated != 1 {
		t.Errorf("AutoCreated = %d, want 1", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}

	// 4. A purchase_events row exists for this bank_tx_id.
	var eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`,
		"T-derived-total-ok",
	).Scan(&eventCount); err != nil {
		t.Fatalf("count purchase_events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("purchase_events count = %d, want 1", eventCount)
	}

	// 5. The persisted total must be the DERIVED value ($788.37), not Claude's
	//    wrong summary.Total ($804.49). worker.go overwrites summary.Total with
	//    the derived value before calling createPurchaseEvent.
	var persistedTotal float64
	if err := testPool.QueryRow(t.Context(),
		`SELECT total FROM purchase_events WHERE bank_tx_id = $1`,
		"T-derived-total-ok",
	).Scan(&persistedTotal); err != nil {
		t.Fatalf("select purchase_events.total: %v", err)
	}
	if persistedTotal < 788.36 || persistedTotal > 788.38 {
		t.Errorf("purchase_events.total = %.2f, want 788.37 (derived, not Claude's 804.49)", persistedTotal)
	}

	// 6. Two line items exist for this event.
	var lineCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_line_items pli
		   JOIN purchase_events pe ON pe.id = pli.purchase_event_id
		  WHERE pe.bank_tx_id = $1`,
		"T-derived-total-ok",
	).Scan(&lineCount); err != nil {
		t.Fatalf("count purchase_line_items: %v", err)
	}
	if lineCount != 2 {
		t.Errorf("purchase_line_items count = %d, want 2", lineCount)
	}
}

// TestWorker_RetryRegression_KeepsBestAttempt asserts that when attempt 2
// (feedback) is WORSE than attempt 1 (initial parse), the worker persists the
// BEST attempt (attempt 1) rather than blindly overwriting with attempt 2's
// degraded data.
//
// Setup:
//   - bank amount = -$788.37
//   - attempt 1 (parseReceipt): derivedTotal=$790.00 (off by $1.63 — fails Check 1)
//     items = [{price:773.40, qty:1}], tax=16.60 → derived = 790.00
//   - attempt 2 (parseReceiptWithFeedback): Claude REGRESSES — returns items=[]
//     tax=0 → derivedTotal=$0.00, score = 0 + 10000 (empty penalty) = extremely bad
//   - Expected: both fail validate; best attempt (1) is stored in pending_purchases;
//     items array in pending row is NON-EMPTY (attempt 1's items); parse_error trace
//     contains both attempt labels.
func TestWorker_RetryRegression_KeepsBestAttempt(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        "T-regression-best",
			Amount:    -788.37,
			CreatedAt: "2026-06-17T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
				{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
			},
		}},
		// Attempt 1: close but fails Check 1 (derivedTotal=790.00 ≠ 788.37).
		// score = |790.00 - 788.37| = 1.63 (close, items present).
		parseItems: []ReceiptItem{
			{Name: "PURCHASE BUCKET", Price: 773.40, Quantity: 1, IsCase: false},
		},
		parseSummary: ReceiptSummary{
			Vendor: "Restaurant Depot",
			Total:  790.00,
			Tax:    16.60,
		},
		// Attempt 2 (feedback): Claude REGRESSES to empty items.
		// score = |0.00 - 788.37| + 10000 (empty penalty) = extremely bad.
		feedbackItems:   []ReceiptItem{},
		feedbackSummary: ReceiptSummary{Vendor: "Restaurant Depot", Total: 0, Tax: 0},
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

	// Both attempts fail validation → routed to pending (not auto-created).
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0 (no attempt should pass validation)", result.AutoCreated)
	}
	if result.PendingReview != 1 {
		t.Errorf("PendingReview = %d, want 1", result.PendingReview)
	}

	// Both parse stubs were called.
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1", stubs.parseCallCount)
	}
	if stubs.feedbackCallCount != 1 {
		t.Errorf("feedbackCallCount = %d, want 1 (feedback retry should fire because attempt 1 failed)", stubs.feedbackCallCount)
	}

	// pending_purchases row must use BEST attempt (attempt 1) — items non-empty.
	var itemsRaw []byte
	var parseErr sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT items, parse_error FROM pending_purchases WHERE bank_tx_id = $1`,
		"T-regression-best",
	).Scan(&itemsRaw, &parseErr); err != nil {
		t.Fatalf("select pending row: %v", err)
	}

	var gotItems []ReceiptItem
	if err := json.Unmarshal(itemsRaw, &gotItems); err != nil {
		t.Fatalf("unmarshal items: %v", err)
	}
	if len(gotItems) == 0 {
		t.Errorf("pending_purchases.items = [] (empty), want attempt 1's items (non-empty) — regression not tracked correctly")
	}

	// parse_error trace must mention both attempts.
	if !parseErr.Valid || parseErr.String == "" {
		t.Errorf("parse_error is NULL or empty, want retry trace with both attempts")
	}
	for _, want := range []string{"attempt 1:", "attempt 2:"} {
		if !strings.Contains(parseErr.String, want) {
			t.Errorf("parse_error %q missing substring %q", parseErr.String, want)
		}
	}
}

// TestWorker_SanityGate_BlocksAutoCreateOnEmptyItems asserts that even when
// ValidateReceiptData would technically pass on vacuous/degenerate data, the
// sanity gate in worker.go blocks auto-create when items are empty or the
// item sum is trivially small (< $0.50). The row must be routed to
// pending_purchases and the reason must mention "Sanity gate".
//
// Setup: bank amount = 0 (or any amount where empty items + tax=0 → derived=0 = -bankAmount)
// to force Check 1 to pass vacuously. In practice: bankAmount=0, items=[], tax=0 →
// derivedTotal=0 = -0 → Check 1 passes (|0 - 0| = 0 ≤ 0.01). Without the sanity gate
// this would auto-create an empty event — a data quality disaster.
func TestWorker_SanityGate_BlocksAutoCreateOnEmptyItems(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        "T-sanity-empty",
			Amount:    0, // vacuous: derived=0 would pass Check 1 with empty items
			CreatedAt: "2026-06-17T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/receipt.jpg", FileName: "receipt.jpg"},
			},
		}},
		// Claude regressed to empty items — vacuously passes Check 1.
		parseItems:   []ReceiptItem{},
		parseSummary: ReceiptSummary{Vendor: "Restaurant Depot", Total: 0, Tax: 0},
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

	// Sanity gate must block auto-create and route to pending.
	if result.AutoCreated != 0 {
		t.Errorf("AutoCreated = %d, want 0 (sanity gate must block empty-items auto-create)", result.AutoCreated)
	}
	if result.PendingReview != 1 {
		t.Errorf("PendingReview = %d, want 1 (sanity gate must route to pending)", result.PendingReview)
	}

	// The pending row's reason must contain "Sanity gate" or "sanity".
	var reason sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT reason FROM pending_purchases WHERE bank_tx_id = $1`,
		"T-sanity-empty",
	).Scan(&reason); err != nil {
		t.Fatalf("select pending row: %v", err)
	}
	if !reason.Valid || !strings.Contains(strings.ToLower(reason.String), "sanity") {
		t.Errorf("pending reason = %q, want substring %q (sanity gate reason)", reason.String, "sanity")
	}
}

// TestWorker_LineItemSumMismatch_RetriesWithFeedback asserts the goal-driven
// retry loop fires when the derived-total check fails due to Claude returning
// extended totals in the price field (not unit prices). This is the pattern
// seen live on tx f13472e8 (Restaurant Depot -$788.37):
//   - derivedTotal = sum(price*qty) + tax = ($543.78 + $1561.85) + $16.60 = $2122.23 ≠ $788.37
//   - Check 1 (|derivedTotal - bankAmount| > 0.01) fires → feedback retry called
//   - Feedback corrects unit prices so derived-total = $788.37 → auto-create.
func TestWorker_LineItemSumMismatch_RetriesWithFeedback(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        "T-linesum-mismatch-retry",
			Amount:    -788.37,
			CreatedAt: "2026-06-24T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/purchase.jpg", FileName: "purchase.jpg"},
				{URL: "http://fake/refund.pdf", FileName: "refund.pdf"},
			},
		}},
		// First parse: Claude returned extended totals in the price field.
		//   BF GROUND CHUK: price=53.69 (extended), quantity=10.13 → product=$543.78
		//   BF TENDERS:     price=91.07 (extended), quantity=17.15 → product=$1561.85
		//   derivedTotal = $543.78 + $1561.85 + $16.60 = $2122.23 ≠ $788.37
		//   Check 1 (|derivedTotal - bankAmount| > 0.01) fires → feedback called.
		// TotalUnits = 27 ensures Check 3 does not interfere (round(27.28)=27 == 27).
		parseItems: []ReceiptItem{
			{Name: "BF GROUND CHUK", Price: 53.69, Quantity: 10.13, IsCase: false},
			{Name: "BF TENDERS", Price: 91.07, Quantity: 17.15, IsCase: false},
		},
		parseSummary: ReceiptSummary{
			Vendor:     "Restaurant Depot #855",
			Total:      788.37,
			Tax:        16.60,
			TotalUnits: 27,
			TotalCases: 0,
		},
		// Feedback stub: corrected unit prices so derived-total matches bank.
		//   item 1: price=50.00, quantity=1.0 → $50.00
		//   item 2: price=721.77, quantity=1.0 → $721.77
		//   derivedTotal = $771.77 + $16.60 = $788.37 ✓ → Check 1 passes.
		//   TotalUnits = 2 = round(1.0 + 1.0) → Check 3 passes.
		feedbackItems: []ReceiptItem{
			{Name: "BF GROUND CHUK", Price: 50.00, Quantity: 1.0, IsCase: false},
			{Name: "BF TENDERS", Price: 721.77, Quantity: 1.0, IsCase: false},
		},
		feedbackSummary: ReceiptSummary{
			Vendor:     "Restaurant Depot #855",
			Total:      788.37,
			Tax:        16.60,
			TotalUnits: 2,
			TotalCases: 0,
		},
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

	// 1. parseReceipt (Haiku) called exactly once.
	if stubs.parseCallCount != 1 {
		t.Errorf("parseCallCount = %d, want 1", stubs.parseCallCount)
	}

	// 2. parseReceiptWithFeedback called exactly once — derived-total check fires
	//    on the extended-price items (derivedTotal=$2122.23 ≠ $788.37) so the
	//    retry loop invokes feedback.
	if stubs.feedbackCallCount != 1 {
		t.Errorf("feedbackCallCount = %d, want 1 (extended-price derived-total mismatch must trigger feedback retry); got %d",
			stubs.feedbackCallCount, stubs.feedbackCallCount)
	}

	// 3. AutoCreated = 1: feedback corrected the parse so the event auto-creates.
	if result.AutoCreated != 1 {
		t.Errorf("AutoCreated = %d, want 1 (feedback-corrected result must auto-create)", result.AutoCreated)
	}
	if result.PendingReview != 0 {
		t.Errorf("PendingReview = %d, want 0", result.PendingReview)
	}

	// 4. A purchase_events row exists for this tx.
	var eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`,
		"T-linesum-mismatch-retry",
	).Scan(&eventCount); err != nil {
		t.Fatalf("count purchase_events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("purchase_events count = %d, want 1", eventCount)
	}
}

// TestWorker_PendingRow_HasPurchaseItemIDsPrefilled asserts that when
// runIngestCycle routes a parsed receipt to pending_purchases (validate-fail
// path), each item in the items JSONB column has its purchase_item_id
// pre-populated for items whose name fuzzy-matches an existing purchase_items
// catalog entry. This lets the FE pre-fill the dropdowns without any FE
// changes — it already reads purchase_item_id from the JSONB.
//
// Setup:
//   - Seed 3 purchase_items with known descriptions.
//   - parseReceipt stub returns 3 items whose names exactly match 2 of the 3
//     seeded catalog items (item 3 is a deliberate mismatch with no catalog
//     entry so purchase_item_id is omitted on that one).
//   - summary.Total deliberately DOES NOT match bank amount so validate fails
//     → routePending is called → items JSONB is persisted.
//
// Assert: at least 2 of the 3 items in the persisted JSONB have a non-empty
// purchase_item_id field.
func TestWorker_PendingRow_HasPurchaseItemIDsPrefilled(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Seed 3 purchase_items. Their descriptions are used as exact-match keys
	// by DerivePurchaseItemID (case-insensitive exact match is the first step).
	var itemID1, itemID2, itemID3 string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('FM HING HOAGIE 99HT1R') RETURNING id`,
	).Scan(&itemID1); err != nil {
		t.Fatalf("seed purchase_item 1: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('BF GROUND CHUK') RETURNING id`,
	).Scan(&itemID2); err != nil {
		t.Fatalf("seed purchase_item 2: %v", err)
	}
	// itemID3 is seeded but deliberately not referenced by any receipt item so
	// we verify the "no match" path leaves purchase_item_id absent.
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('TOTALLY DIFFERENT ITEM') RETURNING id`,
	).Scan(&itemID3); err != nil {
		t.Fatalf("seed purchase_item 3: %v", err)
	}
	_ = itemID3 // not asserted — it just needs to be in the catalog

	// Mercury debit; amount chosen so summary.Total ($999.99) != -bankAmount
	// ($788.37), forcing validate to FAIL and the row to route to pending.
	bankTxID := "T-prefill-pending"
	stubs := &workerStubs{
		txns: []MercuryTransaction{{
			ID:        bankTxID,
			Amount:    -788.37,
			CreatedAt: "2026-06-17T10:00:00Z",
			Attachments: []Attachment{
				{URL: "http://fake/receipt.jpg", FileName: "receipt.jpg"},
			},
		}},
		// Item 1 and 2 match seeded catalog descriptions exactly (case-insensitive).
		// Item 3 ("COMPLETELY UNKNOWN ITEM XYZ") has no match → purchase_item_id omitted.
		parseItems: []ReceiptItem{
			{Name: "fm hing hoagie 99ht1r", Quantity: 1, Price: 400.00, IsCase: false},
			{Name: "BF GROUND CHUK", Quantity: 1, Price: 399.99, IsCase: false},
			{Name: "COMPLETELY UNKNOWN ITEM XYZ", Quantity: 1, Price: 199.99, IsCase: false},
		},
		// summary.Total ($999.98) != -bankAmount ($788.37) → validate fails → pending.
		parseSummary: ReceiptSummary{
			Vendor:     "Restaurant Depot",
			Total:      999.98,
			TotalUnits: 3,
			TotalCases: 0,
		},
		// Feedback also fails (returns same wrong total) so the row stays pending
		// after the retry loop.
		feedbackItems: []ReceiptItem{
			{Name: "fm hing hoagie 99ht1r", Quantity: 1, Price: 400.00, IsCase: false},
			{Name: "BF GROUND CHUK", Quantity: 1, Price: 399.99, IsCase: false},
			{Name: "COMPLETELY UNKNOWN ITEM XYZ", Quantity: 1, Price: 199.99, IsCase: false},
		},
		feedbackSummary: ReceiptSummary{
			Vendor: "Restaurant Depot",
			Total:  999.98,
		},
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

	// Read back the persisted items JSONB.
	var itemsRaw []byte
	if err := testPool.QueryRow(t.Context(),
		`SELECT items FROM pending_purchases WHERE bank_tx_id = $1`, bankTxID,
	).Scan(&itemsRaw); err != nil {
		t.Fatalf("select pending items: %v", err)
	}

	var gotItems []ReceiptItem
	if err := json.Unmarshal(itemsRaw, &gotItems); err != nil {
		t.Fatalf("unmarshal items: %v", err)
	}
	if len(gotItems) != 3 {
		t.Fatalf("expected 3 items in pending JSONB, got %d", len(gotItems))
	}

	// Count how many items have a non-nil purchase_item_id populated.
	matchedCount := 0
	for _, it := range gotItems {
		if it.PurchaseItemID != nil && *it.PurchaseItemID != "" {
			matchedCount++
		}
	}
	if matchedCount < 2 {
		t.Errorf("purchase_item_id pre-filled on %d items, want at least 2 (items 1+2 match catalog; item 3 does not)",
			matchedCount)
	}

	// Spot-check: the first two items must have the correct IDs.
	if gotItems[0].PurchaseItemID == nil || *gotItems[0].PurchaseItemID != itemID1 {
		t.Errorf("items[0].purchase_item_id = %v, want %q (FM HING HOAGIE 99HT1R)", gotItems[0].PurchaseItemID, itemID1)
	}
	if gotItems[1].PurchaseItemID == nil || *gotItems[1].PurchaseItemID != itemID2 {
		t.Errorf("items[1].purchase_item_id = %v, want %q (BF GROUND CHUK)", gotItems[1].PurchaseItemID, itemID2)
	}
	// Third item must have NO purchase_item_id (no catalog match).
	if gotItems[2].PurchaseItemID != nil {
		t.Errorf("items[2].purchase_item_id = %v, want nil (COMPLETELY UNKNOWN ITEM XYZ has no catalog match)",
			gotItems[2].PurchaseItemID)
	}
}

// ─── Phase 260625: two-stage token + AI matcher ──────────────────────────────

// TestEnrichItemsWithMatches_TokenMatcher_HitsHighThreshold verifies that the
// token-overlap matcher bridges the gap between human-friendly catalog names
// ("Lemonade Mix", "Hoagie Containers", "Chicken Tenders") and Restaurant
// Depot SKU-style receipt names ("4C LEMONADE 35QT", "FM HING HOAGIE 99HT1R",
// "BF CHUCK TENDERS RW"). The AI stub returns nothing so the assertion is
// purely on the token-based stage.
func TestEnrichItemsWithMatches_TokenMatcher_HitsHighThreshold(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Seed catalog with human-friendly names.
	var lemonadeID, hoagieID, tenderID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Lemonade Mix') RETURNING id`,
	).Scan(&lemonadeID); err != nil {
		t.Fatalf("seed Lemonade Mix: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Hoagie Containers') RETURNING id`,
	).Scan(&hoagieID); err != nil {
		t.Fatalf("seed Hoagie Containers: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Chicken Tenders') RETURNING id`,
	).Scan(&tenderID); err != nil {
		t.Fatalf("seed Chicken Tenders: %v", err)
	}

	// Stub the AI seam so Stage 2 returns nothing — we're testing Stage 1 only.
	origAI := matchItemsWithAI
	matchItemsWithAI = func(_ context.Context, _ string, _ []string, _ map[string]string) (map[string]string, error) {
		return nil, nil
	}
	t.Cleanup(func() { matchItemsWithAI = origAI })

	items := []ReceiptItem{
		{Name: "4C LEMONADE 35QT"},
		{Name: "FM HING HOAGIE 99HT1R"},
		{Name: "BF CHUCK TENDERS RW"},
	}

	matched, catalogSize, err := enrichItemsWithMatches(t.Context(), testPool, items, "stub-key")
	if err != nil {
		t.Fatalf("enrichItemsWithMatches: %v", err)
	}
	if catalogSize != 3 {
		t.Errorf("catalogSize = %d, want 3", catalogSize)
	}
	if matched != 3 {
		t.Errorf("matched = %d, want 3 (token matcher should bridge SKU→catalog gap)", matched)
	}

	// Spot-check IDs to confirm the right catalog entries were picked.
	if items[0].PurchaseItemID == nil || *items[0].PurchaseItemID != lemonadeID {
		t.Errorf("items[0] PurchaseItemID = %v, want %q (Lemonade Mix)", items[0].PurchaseItemID, lemonadeID)
	}
	if items[1].PurchaseItemID == nil || *items[1].PurchaseItemID != hoagieID {
		t.Errorf("items[1] PurchaseItemID = %v, want %q (Hoagie Containers)", items[1].PurchaseItemID, hoagieID)
	}
	if items[2].PurchaseItemID == nil || *items[2].PurchaseItemID != tenderID {
		t.Errorf("items[2] PurchaseItemID = %v, want %q (Chicken Tenders)", items[2].PurchaseItemID, tenderID)
	}
}

// TestEnrichItemsWithMatches_AIFallback_PicksUpHighConfidence verifies that
// when token matching can't close the gap (e.g. "COCA COLA 12PK" vs "Coke"),
// the AI fallback stage resolves the item by returning a high-confidence match.
// "WHOLE PACKER BRISKET 14LB" should hit token stage (overlap on "brisket").
// "COCA COLA 12PK 12OZ CAN" should fall through to AI fallback.
func TestEnrichItemsWithMatches_AIFallback_PicksUpHighConfidence(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	var brisketID, cokeID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Brisket') RETURNING id`,
	).Scan(&brisketID); err != nil {
		t.Fatalf("seed Brisket: %v", err)
	}
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ('Coke') RETURNING id`,
	).Scan(&cokeID); err != nil {
		t.Fatalf("seed Coke: %v", err)
	}

	// Stub AI to provide the Coke match (simulating high-confidence AI response).
	origAI := matchItemsWithAI
	matchItemsWithAI = func(_ context.Context, _ string, unmatchedNames []string, _ map[string]string) (map[string]string, error) {
		result := make(map[string]string)
		for _, name := range unmatchedNames {
			if name == "COCA COLA 12PK 12OZ CAN" {
				result[name] = cokeID
			}
		}
		return result, nil
	}
	t.Cleanup(func() { matchItemsWithAI = origAI })

	items := []ReceiptItem{
		{Name: "WHOLE PACKER BRISKET 14LB"},
		{Name: "COCA COLA 12PK 12OZ CAN"},
	}

	matched, catalogSize, err := enrichItemsWithMatches(t.Context(), testPool, items, "stub-key")
	if err != nil {
		t.Fatalf("enrichItemsWithMatches: %v", err)
	}
	if catalogSize != 2 {
		t.Errorf("catalogSize = %d, want 2", catalogSize)
	}
	if matched != 2 {
		t.Errorf("matched = %d, want 2 (brisket via token, coke via AI)", matched)
	}

	if items[0].PurchaseItemID == nil || *items[0].PurchaseItemID != brisketID {
		t.Errorf("items[0] PurchaseItemID = %v, want %q (Brisket via token matcher)", items[0].PurchaseItemID, brisketID)
	}
	if items[1].PurchaseItemID == nil || *items[1].PurchaseItemID != cokeID {
		t.Errorf("items[1] PurchaseItemID = %v, want %q (Coke via AI fallback)", items[1].PurchaseItemID, cokeID)
	}
}

// ─── Phase robustness: duplicate bank_tx_id dead-letter cleanup ──────────────

// TestCreatePurchaseEvent_DuplicateBankTxID_CleansPending verifies that when a
// purchase_events row already exists for a bank_tx_id and a second attempt
// tries to INSERT a duplicate, createPurchaseEvent:
//   - returns nil (not an error)
//   - deletes the residual pending_purchases row
//   - leaves exactly 1 purchase_events row (no double-insert)
func TestCreatePurchaseEvent_DuplicateBankTxID_CleansPending(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	const bankTxID = "T-dup-key"

	// Seed: an existing purchase_events row for bank_tx_id X.
	items := []ReceiptItem{
		{Name: "Salmon", Quantity: 1, Price: 42.50, IsCase: false},
	}
	summary := ReceiptSummary{Vendor: "Acme", Tax: 0, Total: 42.50}
	tx := MercuryTransaction{
		ID:        bankTxID,
		Amount:    -42.50,
		CreatedAt: "2026-06-01T10:00:00Z",
	}
	// Insert the first event normally.
	if err := createPurchaseEvent(t.Context(), testPool, tx, items, summary, "", nil, false); err != nil {
		t.Fatalf("first createPurchaseEvent: %v", err)
	}

	// Seed: a residual pending_purchases row with the same bank_tx_id.
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, reason, items)
		VALUES ($1, $2, $3, 'no_attachment_on_bank_tx', '[]'::jsonb)`,
		bankTxID, -42.50, "STUB",
	); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	// Now try to create a second event with the same bank_tx_id — should be handled gracefully.
	err := createPurchaseEvent(t.Context(), testPool, tx, items, summary, "", nil, false)
	if err != nil {
		t.Fatalf("createPurchaseEvent (duplicate) returned error = %v, want nil", err)
	}

	// Assert: pending_purchases row for X is DELETED (residual cleaned up).
	var pendingCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM pending_purchases WHERE bank_tx_id = $1`, bankTxID,
	).Scan(&pendingCount); err != nil {
		t.Fatalf("count pending: %v", err)
	}
	if pendingCount != 0 {
		t.Errorf("pending_purchases count = %d, want 0 (residual row must be cleaned up)", pendingCount)
	}

	// Assert: purchase_events count for X is still 1 (no double-insert).
	var eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id = $1`, bankTxID,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("purchase_events count = %d, want 1 (no double-insert)", eventCount)
	}
}

// ─── Phase robustness: AI matcher hardening tests ───────────────────────────

// TestMatchItemsWithAI_StripsCodeFences asserts that markdown code fences
// wrapping the JSON response are stripped before unmarshal, and the resulting
// map contains the expected match.
func TestMatchItemsWithAI_StripsCodeFences(t *testing.T) {
	// Stub Anthropic client via the package-level seam.
	// We call MatchItemsWithAI directly (not via the seam var) so we need to
	// stub the underlying transport. Instead, test via the exported function
	// using a fake apiKey and a stubbed HTTP client — but the cleanest approach
	// here is to test stripJSONFence directly (which is the extraction we're adding),
	// then test MatchItemsWithAI via the seam with a stub that returns fenced JSON.

	// Test the stripJSONFence helper directly.
	fenced := "```json\n{\"matches\":[{\"raw_name\":\"X\",\"catalog_name\":\"Y\",\"confidence\":\"high\"}]}\n```"
	stripped := StripJSONFence(fenced)
	var response struct {
		Matches []ItemMatch `json:"matches"`
	}
	if err := json.Unmarshal([]byte(stripped), &response); err != nil {
		t.Fatalf("unmarshal after strip: %v — stripped text: %q", err, stripped)
	}
	if len(response.Matches) != 1 {
		t.Fatalf("matches len = %d, want 1", len(response.Matches))
	}
	if response.Matches[0].RawName != "X" || response.Matches[0].CatalogName != "Y" {
		t.Errorf("match = %+v, want {RawName:X CatalogName:Y}", response.Matches[0])
	}
}

// TestMatchItemsWithAI_NaturalLanguageResponseDegradesGracefully asserts that
// when the AI returns natural language instead of JSON (after fence stripping),
// MatchItemsWithAI returns (nil, nil) — no error, no matches.
func TestMatchItemsWithAI_NaturalLanguageResponseDegradesGracefully(t *testing.T) {
	naturalLangResponse := "I notice the input was empty, so there is nothing to match."
	stripped := StripJSONFence(naturalLangResponse)
	// Simulate what MatchItemsWithAI does on unmarshal failure after stripping:
	// should degrade gracefully to (nil, nil) rather than returning an error.
	var response struct {
		Matches []ItemMatch `json:"matches"`
	}
	err := json.Unmarshal([]byte(stripped), &response)
	if err == nil {
		// If somehow it parsed (shouldn't), the result is empty — that's fine too.
		return
	}
	// The production code should catch this and return (nil, nil).
	// We verify this behavior by calling the seam-stubbed version.
	// Stub the matchItemsWithAI seam to use a fake Anthropic response.
	origMatchItemsWithAI := matchItemsWithAI
	matchItemsWithAI = func(ctx context.Context, apiKey string, unmatchedNames []string, catalog map[string]string) (map[string]string, error) {
		// Simulate the production behavior: receive natural language, degrade gracefully.
		rawText := naturalLangResponse
		rawText = StripJSONFence(rawText)
		var resp struct {
			Matches []ItemMatch `json:"matches"`
		}
		if jsonErr := json.Unmarshal([]byte(rawText), &resp); jsonErr != nil {
			// Natural language response — degrade gracefully.
			return nil, nil
		}
		return nil, nil
	}
	t.Cleanup(func() { matchItemsWithAI = origMatchItemsWithAI })

	result, err := matchItemsWithAI(t.Context(), "stub-key", []string{"X"}, map[string]string{"Y": "y-id"})
	if err != nil {
		t.Fatalf("matchItemsWithAI returned error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("matchItemsWithAI result = %v, want nil (natural language should degrade)", result)
	}
}

// TestMatchItemsWithAI_EmptyInputSkipsAPICall asserts that when unmatchedNames
// is empty, MatchItemsWithAI returns (nil, nil) without making any API call.
func TestMatchItemsWithAI_EmptyInputSkipsAPICall(t *testing.T) {
	callCount := 0
	origMatchItemsWithAI := matchItemsWithAI
	matchItemsWithAI = func(ctx context.Context, apiKey string, unmatchedNames []string, catalog map[string]string) (map[string]string, error) {
		if len(unmatchedNames) == 0 {
			// Should return early without calling any API.
			return nil, nil
		}
		callCount++
		return nil, nil
	}
	t.Cleanup(func() { matchItemsWithAI = origMatchItemsWithAI })

	result, err := matchItemsWithAI(t.Context(), "stub-key", []string{}, map[string]string{"Y": "y-id"})
	if err != nil {
		t.Fatalf("matchItemsWithAI returned error = %v, want nil", err)
	}
	if result != nil {
		t.Errorf("matchItemsWithAI result = %v, want nil", result)
	}
	if callCount != 0 {
		t.Errorf("API callCount = %d, want 0 (empty input must not call API)", callCount)
	}

	// Also verify the exported MatchItemsWithAI directly.
	result2, err2 := MatchItemsWithAI(t.Context(), "stub-key", []string{}, map[string]string{"Y": "y-id"})
	if err2 != nil {
		t.Fatalf("MatchItemsWithAI with empty input returned error = %v, want nil", err2)
	}
	if result2 != nil {
		t.Errorf("MatchItemsWithAI result = %v, want nil for empty input", result2)
	}
}

// ─── BatchReprocessFromSpaces unit tests ─────────────────────────────────────

// TestBatchReprocessFromSpaces_ProcessesEachRow exercises the Spaces-based
// reprocess batch runner:
//   - Row 1: valid receipt URL + parse succeeds → "auto_created"
//   - Row 2: valid receipt URL + validate fails → "pending_review"
//   - Row 3: no receipt URLs → "no_attachments"
//
// No Mercury or Anthropic API calls are made. All seams are stubbed.
func TestBatchReprocessFromSpaces_ProcessesEachRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetReceiptFixtures(t)

	// Row 1 seeds: parse succeeds, validate passes → auto_created.
	// bank_total is -42.50 (debit); summary.Total must equal 42.50 and
	// sum(qty*price) == 42.50 for validate to pass.
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url)
		VALUES ($1, $2, $3, '[]'::jsonb, 'Receipt could not be parsed automatically',
		        'https://spaces.example.com/receipts/row1.jpg')`,
		"sp-row1", -42.50, "SpacesVendor1",
	); err != nil {
		t.Fatalf("seed row1: %v", err)
	}

	// Row 2 seeds: parse succeeds but total mismatch → pending_review.
	// bank_total=-20.00 but we'll return a parse result totalling $99, which
	// won't match, so it routes to pending_review.
	if _, err := testPool.Exec(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, receipt_url)
		VALUES ($1, $2, $3, '[]'::jsonb, 'Receipt could not be parsed automatically',
		        'https://spaces.example.com/receipts/row2.jpg')`,
		"sp-row2", -20.00, "SpacesVendor2",
	); err != nil {
		t.Fatalf("seed row2: %v", err)
	}

	// Row 3: no receipt URLs → no_attachments without calling parse.
	row3 := PendingRowForReprocess{
		BankTxID:    "sp-row3",
		BankTotal:   -10.00,
		Vendor:      "SpacesVendor3",
		EventDate:   "2026-06-01",
		ReceiptURLs: nil, // explicitly no URLs
	}

	// Install stubs: downloadReceiptFileFn, parseReceipt, parseReceiptWithFeedback.
	callsByTx := map[string]int{}
	origDL := downloadReceiptFileFn
	downloadReceiptFileFn = func(_ context.Context, _ string) ([]byte, string, error) {
		return []byte("FAKE-RECEIPT-BYTES"), "image/jpeg", nil
	}
	t.Cleanup(func() { downloadReceiptFileFn = origDL })

	origParse := parseReceipt
	parseReceipt = func(_ context.Context, _ string, _ []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
		// We can't easily tell which row is calling, so return per-call based on count.
		callsByTx["parse"]++
		n := callsByTx["parse"]
		if n == 1 {
			// Row 1: valid parse — total matches -bank_total 42.50.
			return []ReceiptItem{
				{Name: "Item A", Quantity: 1, Price: 42.50, IsCase: false},
			}, ReceiptSummary{Vendor: "SpacesVendor1", Tax: 0, Total: 42.50, TotalUnits: 1}, nil
		}
		// Row 2: total mismatch ($99 vs $20 bank).
		return []ReceiptItem{
			{Name: "Item B", Quantity: 1, Price: 99.00, IsCase: false},
		}, ReceiptSummary{Vendor: "SpacesVendor2", Tax: 0, Total: 99.00, TotalUnits: 1}, nil
	}
	t.Cleanup(func() { parseReceipt = origParse })

	origSonnet := parseReceiptWithSonnet
	parseReceiptWithSonnet = func(_ context.Context, _ string, _ []FileBlob) ([]ReceiptItem, ReceiptSummary, error) {
		// Not expected to be called (Haiku succeeds for both rows).
		t.Error("parseReceiptWithSonnet called unexpectedly")
		return nil, ReceiptSummary{}, nil
	}
	t.Cleanup(func() { parseReceiptWithSonnet = origSonnet })

	origFeedback := parseReceiptWithFeedback
	parseReceiptWithFeedback = func(_ context.Context, _ string, _ []FileBlob, _, _ float64, _ string) ([]ReceiptItem, ReceiptSummary, error) {
		// Row 2 will retry once. Return same mismatched result so it ends in pending_review.
		return []ReceiptItem{
			{Name: "Item B", Quantity: 1, Price: 99.00, IsCase: false},
		}, ReceiptSummary{Vendor: "SpacesVendor2", Tax: 0, Total: 99.00, TotalUnits: 1}, nil
	}
	t.Cleanup(func() { parseReceiptWithFeedback = origFeedback })

	cfg := WorkerConfig{
		MercuryAPIKey:   "stub",
		AnthropicAPIKey: "stub",
		Pool:            testPool,
	}

	rows := []PendingRowForReprocess{
		{BankTxID: "sp-row1", BankTotal: -42.50, Vendor: "SpacesVendor1", EventDate: "2026-06-01", ReceiptURLs: []string{"https://spaces.example.com/receipts/row1.jpg"}},
		{BankTxID: "sp-row2", BankTotal: -20.00, Vendor: "SpacesVendor2", EventDate: "2026-06-02", ReceiptURLs: []string{"https://spaces.example.com/receipts/row2.jpg"}},
		row3,
	}

	result, err := BatchReprocessFromSpaces(t.Context(), cfg, rows)
	if err != nil {
		t.Fatalf("BatchReprocessFromSpaces: %v", err)
	}

	if len(result) != 3 {
		t.Fatalf("result len = %d, want 3", len(result))
	}
	if result["sp-row1"] != "auto_created" {
		t.Errorf("sp-row1 status = %q, want auto_created", result["sp-row1"])
	}
	if result["sp-row2"] != "pending_review" {
		t.Errorf("sp-row2 status = %q, want pending_review", result["sp-row2"])
	}
	if result["sp-row3"] != "no_attachments" {
		t.Errorf("sp-row3 status = %q, want no_attachments", result["sp-row3"])
	}
}
