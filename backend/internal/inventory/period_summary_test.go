package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
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
		// Cannot reach DB — leave testPool nil so TestPeriodSummary skips.
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

func resetFixtures(t *testing.T) {
	t.Helper()
	_, err := testPool.Exec(t.Context(), `
		TRUNCATE purchase_line_items, purchase_events, pending_purchases,
		         purchase_items, item_groups, vendors
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// callHandler invokes PeriodSummaryHandler directly (not through chi)
// with the default ["COGS"] allowlist.
func callHandler(t *testing.T, from, to string) (int, PeriodSummary) {
	return callHandlerWithAllowlist(t, from, to, []string{"COGS"})
}

// callHandlerWithAllowlist is the underlying helper that lets a subtest pass
// a custom Mercury category allowlist (Phase 260605-v0n).
func callHandlerWithAllowlist(t *testing.T, from, to string, allowlist []string) (int, PeriodSummary) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?from="+from+"&to="+to, nil)
	rec := httptest.NewRecorder()
	PeriodSummaryHandler(testPool, allowlist).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return rec.Code, PeriodSummary{}
	}
	var out PeriodSummary
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}
	return rec.Code, out
}

// insertVendor inserts one vendor and returns its UUID.
func insertVendor(t *testing.T, name string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO vendors (name) VALUES ($1) RETURNING id::text`, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert vendor: %v", err)
	}
	return id
}

// insertPurchaseItem inserts one canonical purchase item and returns its UUID.
// item_groups is required by FK in some setups — keep it optional here by
// leaving group_id NULL.
func insertPurchaseItem(t *testing.T, description string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description) VALUES ($1) RETURNING id::text`,
		description).Scan(&id)
	if err != nil {
		t.Fatalf("insert purchase_item: %v", err)
	}
	return id
}

// insertEventAndLine inserts a purchase_event + a single purchase_line_item.
// If purchaseItemID is empty string, the line item's purchase_item_id is NULL
// (the unlinked case).
//
// Phase 260605-v0n: Defaults mercury_category='COGS' so events are included
// under the default allowlist. Use insertEventAndLineWithCategory for tests
// that need a non-COGS or NULL category.
func insertEventAndLine(t *testing.T, vendorID, eventDate string, tax, total, price float64, qty int, purchaseItemID string) (string, string) {
	t.Helper()
	return insertEventAndLineWithCategory(t, vendorID, eventDate, tax, total, price, qty, purchaseItemID, "COGS")
}

// insertEventAndLineWithCategory is the underlying helper. Empty `category`
// string is treated as NULL (uncategorized — excluded by default allowlist).
func insertEventAndLineWithCategory(t *testing.T, vendorID, eventDate string, tax, total, price float64, qty int, purchaseItemID, category string) (string, string) {
	t.Helper()
	var eventID string
	var categoryArg interface{}
	if category == "" {
		categoryArg = nil
	} else {
		categoryArg = category
	}
	bankTxID := "tx-" + eventDate + "-" + strconv.Itoa(int(price*10000))
	if category != "" && category != "COGS" {
		bankTxID += "-" + category
	}
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
		 VALUES ($1, $2, $3::date, $4, $5, $6)
		 RETURNING id::text`,
		vendorID, bankTxID, eventDate, tax, total, categoryArg).Scan(&eventID)
	if err != nil {
		t.Fatalf("insert purchase_event: %v", err)
	}
	var lineID string
	if purchaseItemID == "" {
		err = testPool.QueryRow(t.Context(),
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, NULL, $2, $3, $4, false)
			 RETURNING id::text`,
			eventID, "Test Item", qty, price).Scan(&lineID)
	} else {
		err = testPool.QueryRow(t.Context(),
			`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
			 VALUES ($1, $2, $3, $4, $5, false)
			 RETURNING id::text`,
			eventID, purchaseItemID, "Test Item", qty, price).Scan(&lineID)
	}
	if err != nil {
		t.Fatalf("insert purchase_line_item: %v", err)
	}
	return eventID, lineID
}

func insertPendingPurchase(t *testing.T, createdAt string, confirmed, discarded bool) string {
	t.Helper()
	var id string
	// confirmed/discarded controlled by setting confirmed_at/discarded_at to now() or NULL.
	confirmedSQL := "NULL"
	if confirmed {
		confirmedSQL = "now()"
	}
	discardedSQL := "NULL"
	if discarded {
		discardedSQL = "now()"
	}
	// items JSONB requires a value — use empty array.
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, created_at, confirmed_at, discarded_at)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, $2::timestamptz, ` + confirmedSQL + `, ` + discardedSQL + `)
	      RETURNING id::text`
	err := testPool.QueryRow(t.Context(), q, "pp-tx-"+createdAt+strconv.Itoa(int(boolByte(confirmed))+int(boolByte(discarded))*2), createdAt).Scan(&id)
	if err != nil {
		t.Fatalf("insert pending_purchase: %v", err)
	}
	return id
}

func boolByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// insertNoItemizedReceiptSeed re-inserts the seed purchase_items row that
// resetFixtures TRUNCATEs away. Idempotent. See migration
// 0064_no_itemized_receipt_seed.sql for the production source of this row.
func insertNoItemizedReceiptSeed(t *testing.T) string {
	t.Helper()
	const seedID = "00000000-0000-0000-0000-000000000001"
	_, err := testPool.Exec(t.Context(),
		`INSERT INTO purchase_items (id, description) VALUES ($1, '(no itemized receipt)') ON CONFLICT (description) DO NOTHING`,
		seedID)
	if err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	return seedID
}

// insertTestUser inserts a minimal active user row so confirmed_by FK on
// pending_purchases can resolve. Returns the user's UUID.
func insertTestUser(t *testing.T, email string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO users (email, display_name, role, status) VALUES ($1, $2, 'admin', 'active') RETURNING id::text`,
		email, "Test User").Scan(&id)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

// insertPendingPurchaseWithBankTotal inserts an unconfirmed/undiscarded
// pending_purchases row with a real bank_total (typically negative for a
// debit). Used by the end-to-end empty-items confirm test.
func insertPendingPurchaseWithBankTotal(t *testing.T, bankTxID string, bankTotal float64, createdAt string) string {
	t.Helper()
	var id string
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, created_at)
	      VALUES ($1, $2, 'TestVendor', '[]'::jsonb, $3::timestamptz)
	      RETURNING id::text`
	err := testPool.QueryRow(t.Context(), q, bankTxID, bankTotal, createdAt).Scan(&id)
	if err != nil {
		t.Fatalf("insert pending_purchase: %v", err)
	}
	return id
}

// insertPendingPurchaseWithEventDate inserts an unconfirmed/undiscarded
// pending_purchases row. eventDate is a DATE string ("YYYY-MM-DD") or "" for NULL.
// createdAt is a timestamptz string. reason is a string sentinel or "" for NULL.
// Returns the inserted id::text.
func insertPendingPurchaseWithEventDate(t *testing.T, bankTxID, eventDate, createdAt, reason string) string {
	t.Helper()
	var (
		ed *string
		rs *string
	)
	if eventDate != "" {
		ed = &eventDate
	}
	if reason != "" {
		rs = &reason
	}
	var id string
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, $2::date, $3, $4::timestamptz)
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID, ed, rs, createdAt).Scan(&id); err != nil {
		t.Fatalf("insert pending_purchase: %v", err)
	}
	return id
}

func TestPeriodSummary(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	const from = "2026-05-25"
	const to = "2026-05-31"

	t.Run("ready=true with linked items and no pending", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		// Two line items totaling 5*4.5 + 2*10.25 = 22.5 + 20.5 = 43.00.
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		insertEventAndLine(t, vendorID, "2026-05-28", 1.50, 22.00, 10.25, 2, piID)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (pending=%v unlinked=%v)",
				got.Completeness.PendingReviewIDs, got.Completeness.UnlinkedLineItemIDs)
		}
		if got.COGSExclTax != 43.00 {
			t.Errorf("COGSExclTax = %v, want 43.00", got.COGSExclTax)
		}
		// cogs_incl_tax = 43.00 + (2.50 + 1.50) = 47.00
		if got.COGSInclTax != 47.00 {
			t.Errorf("COGSInclTax = %v, want 47.00", got.COGSInclTax)
		}
		if got.PurchaseEventCount != 2 {
			t.Errorf("PurchaseEventCount = %d, want 2", got.PurchaseEventCount)
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=false when pending purchase in range is unconfirmed", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)

		// Pending purchase created on 2026-05-27 (in range, in America/Chicago).
		// Use timestamptz in Chicago to mirror the ingestion semantics.
		ppID := insertPendingPurchase(t, "2026-05-27 10:00:00-05:00", false, false)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (pending should block)")
		}
		if len(got.Completeness.PendingReviewIDs) != 1 || got.Completeness.PendingReviewIDs[0] != ppID {
			t.Errorf("PendingReviewIDs = %v, want [%s]", got.Completeness.PendingReviewIDs, ppID)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=false when a confirmed event has an unlinked line item", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		// One linked line + one event with an unlinked line.
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		_, unlinkedID := insertEventAndLine(t, vendorID, "2026-05-28", 1.00, 11.00, 5.0, 2, "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (unlinked line should block)")
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 1 || got.Completeness.UnlinkedLineItemIDs[0] != unlinkedID {
			t.Errorf("UnlinkedLineItemIDs = %v, want [%s]", got.Completeness.UnlinkedLineItemIDs, unlinkedID)
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
	})

	t.Run("discarded pending purchase does NOT block ready", func(t *testing.T) {
		// Roadmap constraint: "Discarded pending_purchases (discarded_at IS
		// NOT NULL) count as resolved." If this subtest fails, the SQL filter
		// on discarded_at IS NULL was dropped.
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		// A discarded pending purchase in range — confirms exclusion clause.
		_ = insertPendingPurchase(t, "2026-05-27 10:00:00-05:00", false, true)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (discarded should be excluded). pending=%v unlinked=%v",
				got.Completeness.PendingReviewIDs, got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=false when no_attachment_on_bank_tx pending row in range", func(t *testing.T) {
		// Phase 260605-pk1: HQ now surfaces every supported Mercury card swipe
		// (not just photographed ones) as a pending_purchases row with
		// reason='no_attachment_on_bank_tx'. The completeness gate's existing
		// filter (confirmed_at IS NULL AND discarded_at IS NULL) is
		// reason-agnostic, so these rows must block ready.
		// Phase 260606-0gh: event_date is now set explicitly in range (it is
		// load-bearing for the COALESCE filter — this confirms the new WHERE
		// picks it up even when created_at also happens to be in range).
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)

		// No-attachment pending row: event_date in range, created_at also in range.
		ppID := insertPendingPurchaseWithEventDate(t,
			"pp-tx-noatt-in-range", "2026-05-28", "2026-05-28 10:00:00-05:00", "no_attachment_on_bank_tx")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (no_attachment row should block)")
		}
		found := false
		for _, id := range got.Completeness.PendingReviewIDs {
			if id == ppID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PendingReviewIDs = %v, want to contain %s", got.Completeness.PendingReviewIDs, ppID)
		}
	})

	t.Run("bad date format returns 400", func(t *testing.T) {
		resetFixtures(t)
		code, _ := callHandler(t, "not-a-date", to)
		if code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", code)
		}
	})

	t.Run("from > to returns 400", func(t *testing.T) {
		resetFixtures(t)
		code, _ := callHandler(t, "2026-05-31", "2026-05-25")
		if code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", code)
		}
	})

	// Phase 260605-q7b: empty-items "confirm without receipt" path now
	// inserts a placeholder purchase_line_items row linked to the seed
	// purchase_items row from migration 0064. These three subtests cover:
	// (1) the placeholder dollars land in cogs_excl_tax,
	// (2) the placeholder does NOT trip unlinked_line_item_ids, and
	// (3) the end-to-end ConfirmPendingPurchaseHandler empty-items path
	//     produces the same result as inserting the rows by hand.

	t.Run("placeholder line item lands in cogs_excl_tax", func(t *testing.T) {
		resetFixtures(t)
		seedID := insertNoItemizedReceiptSeed(t)
		vendorID := insertVendor(t, "RestaurantDepot")
		// Insert one purchase_events row in window plus a single placeholder
		// purchase_line_items row mirroring an abs(bank_total) of $50.00.
		// Uses insertEventAndLine — it accepts the seed UUID like any other
		// purchase_item_id.
		insertEventAndLine(t, vendorID, "2026-05-26", 0, 50.00, 50.00, 1, seedID)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.COGSExclTax != 50.00 {
			t.Errorf("COGSExclTax = %v, want 50.00", got.COGSExclTax)
		}
		if got.PurchaseEventCount != 1 {
			t.Errorf("PurchaseEventCount = %d, want 1", got.PurchaseEventCount)
		}
	})

	t.Run("placeholder does NOT trip unlinked_line_item_ids", func(t *testing.T) {
		resetFixtures(t)
		seedID := insertNoItemizedReceiptSeed(t)
		vendorID := insertVendor(t, "RestaurantDepot")
		insertEventAndLine(t, vendorID, "2026-05-26", 0, 50.00, 50.00, 1, seedID)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want [] (placeholder must be linked to seed)",
				got.Completeness.UnlinkedLineItemIDs)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true. pending=%v unlinked=%v",
				got.Completeness.PendingReviewIDs, got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("end-to-end empty-items confirm increments cogs", func(t *testing.T) {
		resetFixtures(t)
		// Re-seed the placeholder catalog row (resetFixtures TRUNCATEs it),
		// then create a user (FK for pending_purchases.confirmed_by) and a
		// pending_purchases row with bank_total=-75.00 (negative = debit).
		insertNoItemizedReceiptSeed(t)
		userID := insertTestUser(t, "confirm-empty@yumyums.test")
		ppID := insertPendingPurchaseWithBankTotal(t,
			"e2e-empty-tx-1", -75.00, "2026-05-27 10:00:00-05:00")

		// Invoke ConfirmPendingPurchaseHandler via httptest with the seeded
		// user in context (mirrors what auth.Middleware does in prod).
		body := ConfirmPendingInput{
			ID:         ppID,
			VendorName: "RestaurantDepot",
			EventDate:  "2026-05-27",
			LineItems:  nil, // empty — triggers the empty-resolution branch
		}
		buf, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal confirm body: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost,
			"/api/v1/inventory/purchases/confirm", bytes.NewReader(buf))
		req.Header.Set("Content-Type", "application/json")
		ctx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{
			ID:    userID,
			Email: "confirm-empty@yumyums.test",
		})
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		ConfirmPendingPurchaseHandler(testPool).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("confirm status = %d, body=%s", rec.Code, rec.Body.String())
		}

		// Phase 260605-v0n: ConfirmPendingPurchaseHandler writes NULL for
		// mercury_category by design (worker re-sync fills it on next tick).
		// In production the 6h worker pass populates it; here we simulate
		// that pass with a direct UPDATE so the default ["COGS"] allowlist
		// includes the just-confirmed event.
		if _, err := testPool.Exec(t.Context(),
			`UPDATE purchase_events SET mercury_category = 'COGS' WHERE bank_tx_id = $1`,
			"e2e-empty-tx-1"); err != nil {
			t.Fatalf("simulate worker re-sync: %v", err)
		}

		// Now query period-summary — cogs_excl_tax should equal abs(-75.00).
		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("period-summary status = %d, want 200", code)
		}
		if got.COGSExclTax != 75.00 {
			t.Errorf("COGSExclTax = %v, want 75.00 (abs(bank_total))", got.COGSExclTax)
		}
		if got.PurchaseEventCount != 1 {
			t.Errorf("PurchaseEventCount = %d, want 1", got.PurchaseEventCount)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []",
				got.Completeness.UnlinkedLineItemIDs)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true. pending=%v unlinked=%v",
				got.Completeness.PendingReviewIDs, got.Completeness.UnlinkedLineItemIDs)
		}
	})

	// Phase 260605-u0i: per-vendor COGS breakdown on /period-summary.
	// The sales-processor weekly payroll PDF renders a per-vendor spend
	// table from `by_vendor`. These subtests pin shape, the sums-match
	// invariant, the order contract (spend desc, name asc tiebreaker),
	// the empty-period JSON shape ([] not null), and a regression guard
	// for purchase_events with zero line items.

	t.Run("by_vendor: shape + sums match + order", func(t *testing.T) {
		resetFixtures(t)
		// Two vendors with different spend; one tied pair on a third
		// vendor to exercise the name-ASC tiebreaker.
		depotID := insertVendor(t, "Restaurant Depot")
		salID := insertVendor(t, "Save-A-Lot")
		zetaID := insertVendor(t, "Zeta")
		acmeID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")

		// Restaurant Depot: 2 events, qty*price = 5*10 + 3*20 = 110; tax 5+5=10
		insertEventAndLine(t, depotID, "2026-05-26", 5.00, 60.00, 10.00, 5, piID)
		insertEventAndLine(t, depotID, "2026-05-27", 5.00, 65.00, 20.00, 3, piID)
		// Save-A-Lot: 1 event, 2*15 = 30; tax 2
		insertEventAndLine(t, salID, "2026-05-28", 2.00, 32.00, 15.00, 2, piID)
		// Tied pair (both $10.00 excl) — Acme must come before Zeta
		insertEventAndLine(t, zetaID, "2026-05-29", 0.50, 10.50, 10.00, 1, piID)
		insertEventAndLine(t, acmeID, "2026-05-30", 0.50, 10.50, 10.00, 1, piID)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}

		if got.ByVendor == nil {
			t.Fatalf("ByVendor is nil; want non-nil slice")
		}
		if len(got.ByVendor) != 4 {
			t.Fatalf("len(ByVendor) = %d, want 4 (got=%+v)", len(got.ByVendor), got.ByVendor)
		}

		// Order: Depot (110) > Save-A-Lot (30) > Acme (10, ASC) > Zeta (10)
		wantOrder := []string{"Restaurant Depot", "Save-A-Lot", "Acme", "Zeta"}
		for i, name := range wantOrder {
			if got.ByVendor[i].VendorName != name {
				t.Errorf("ByVendor[%d].VendorName = %q, want %q", i, got.ByVendor[i].VendorName, name)
			}
		}

		// Shape: every row has non-empty ID/name and TripCount >= 1
		for i, row := range got.ByVendor {
			if row.VendorID == "" {
				t.Errorf("row %d VendorID empty", i)
			}
			if row.VendorName == "" {
				t.Errorf("row %d VendorName empty", i)
			}
			if row.TripCount < 1 {
				t.Errorf("row %d TripCount = %d, want >=1", i, row.TripCount)
			}
		}

		// Sums invariant
		var sumExcl, sumIncl float64
		for _, row := range got.ByVendor {
			sumExcl += row.TotalExclTax
			sumIncl += row.TotalInclTax
		}
		if diff := sumExcl - got.COGSExclTax; diff > 0.01 || diff < -0.01 {
			t.Errorf("Σ TotalExclTax (%v) != COGSExclTax (%v)", sumExcl, got.COGSExclTax)
		}
		if diff := sumIncl - got.COGSInclTax; diff > 0.01 || diff < -0.01 {
			t.Errorf("Σ TotalInclTax (%v) != COGSInclTax (%v)", sumIncl, got.COGSInclTax)
		}

		// Trip count for Restaurant Depot must be 2
		if got.ByVendor[0].TripCount != 2 {
			t.Errorf("Depot TripCount = %d, want 2", got.ByVendor[0].TripCount)
		}
	})

	t.Run("by_vendor: empty period renders [] not null", func(t *testing.T) {
		resetFixtures(t)
		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if got.ByVendor == nil {
			t.Fatalf("ByVendor is nil; want empty slice")
		}
		if len(got.ByVendor) != 0 {
			t.Fatalf("len(ByVendor) = %d, want 0", len(got.ByVendor))
		}
		// JSON-level check: must render as [] not null
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !bytes.Contains(raw, []byte(`"by_vendor":[]`)) {
			t.Errorf("JSON missing `\"by_vendor\":[]` — got %s", string(raw))
		}
	})

	t.Run("by_vendor: zero-line-items event still appears (regression)", func(t *testing.T) {
		resetFixtures(t)
		// Direct INSERT — bypass insertEventAndLine (which always adds
		// a line) AND bypass the confirm handler (which would create
		// the placeholder line item from the undercount fix). This
		// guards against future code paths that create events without
		// line items.
		vendorID := insertVendor(t, "OrphanVendor")
		var eventID string
		err := testPool.QueryRow(t.Context(),
			`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
			 VALUES ($1, $2, '2026-05-28'::date, 0, 0, 'COGS')
			 RETURNING id::text`,
			vendorID, "tx-orphan-260528").Scan(&eventID)
		if err != nil {
			t.Fatalf("insert orphan event: %v", err)
		}

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if len(got.ByVendor) != 1 {
			t.Fatalf("len(ByVendor) = %d, want 1 (got=%+v)", len(got.ByVendor), got.ByVendor)
		}
		row := got.ByVendor[0]
		if row.VendorName != "OrphanVendor" {
			t.Errorf("VendorName = %q, want OrphanVendor", row.VendorName)
		}
		if row.TotalExclTax != 0 {
			t.Errorf("TotalExclTax = %v, want 0", row.TotalExclTax)
		}
		if row.TripCount != 1 {
			t.Errorf("TripCount = %d, want 1", row.TripCount)
		}
	})

	// Phase 260605-v0n: Mercury category allowlist filters the COGS aggregate.
	// CubeSmart storage rent (category "Rent & Utilities") stays in
	// purchase_events for bookkeeping but does NOT roll up into food-cost
	// numbers sent to sales-processor. NULL is also excluded (Postgres
	// ANY(NULL) returns NULL, not true). Custom allowlists let ops include
	// additional categories without a code change.

	t.Run("allowlist excludes non-COGS rows", func(t *testing.T) {
		resetFixtures(t)
		acmeID := insertVendor(t, "Acme")
		cubeID := insertVendor(t, "CubeSmart")
		piID := insertPurchaseItem(t, "Salmon")
		// COGS event ($30 line). Default helper sets mercury_category='COGS'.
		insertEventAndLine(t, acmeID, "2026-05-26", 0, 30.00, 30.00, 1, piID)
		// Non-COGS event ($999 line) — should be filtered out.
		insertEventAndLineWithCategory(t, cubeID, "2026-05-27", 0, 999.00, 999.00, 1, piID, "Rent & Utilities")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if got.COGSExclTax != 30.00 {
			t.Errorf("COGSExclTax = %v, want 30.00 (non-COGS event excluded)", got.COGSExclTax)
		}
		if got.PurchaseEventCount != 1 {
			t.Errorf("PurchaseEventCount = %d, want 1", got.PurchaseEventCount)
		}
		if len(got.ByVendor) != 1 {
			t.Fatalf("len(ByVendor) = %d, want 1 (CubeSmart should be excluded; got=%+v)",
				len(got.ByVendor), got.ByVendor)
		}
		if got.ByVendor[0].VendorName != "Acme" {
			t.Errorf("ByVendor[0].VendorName = %q, want Acme", got.ByVendor[0].VendorName)
		}
		// tax aggregate must also exclude the non-COGS event (tax was 0 on both
		// but the assertion is structural — by_vendor only has Acme so any tax
		// on CubeSmart could only leak via the correlated subquery on pe2).
		if got.ByVendor[0].TotalInclTax != 30.00 {
			t.Errorf("ByVendor[0].TotalInclTax = %v, want 30.00", got.ByVendor[0].TotalInclTax)
		}
	})

	t.Run("NULL mercury_category is excluded by default", func(t *testing.T) {
		resetFixtures(t)
		acmeID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		// Empty category string → NULL in DB. ANY(NULL) returns NULL → excluded.
		insertEventAndLineWithCategory(t, acmeID, "2026-05-26", 0, 50.00, 50.00, 1, piID, "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0 (NULL category excluded)", got.COGSExclTax)
		}
		if got.PurchaseEventCount != 0 {
			t.Errorf("PurchaseEventCount = %d, want 0", got.PurchaseEventCount)
		}
		if len(got.ByVendor) != 0 {
			t.Errorf("len(ByVendor) = %d, want 0 (NULL-category vendor must not appear)",
				len(got.ByVendor))
		}
	})

	t.Run("custom multi-element allowlist includes both", func(t *testing.T) {
		resetFixtures(t)
		acmeID := insertVendor(t, "Acme")
		otherID := insertVendor(t, "OtherVendor")
		piID := insertPurchaseItem(t, "Salmon")
		// COGS event ($30) — included by default + custom allowlist.
		insertEventAndLine(t, acmeID, "2026-05-26", 0, 30.00, 30.00, 1, piID)
		// "Other / Needs Review" event ($40) — excluded by default, included
		// when the custom allowlist names it.
		insertEventAndLineWithCategory(t, otherID, "2026-05-27", 0, 40.00, 40.00, 1, piID, "Other / Needs Review")

		code, got := callHandlerWithAllowlist(t, from, to, []string{"COGS", "Other / Needs Review"})
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if got.COGSExclTax != 70.00 {
			t.Errorf("COGSExclTax = %v, want 70.00 (both categories included)", got.COGSExclTax)
		}
		if got.PurchaseEventCount != 2 {
			t.Errorf("PurchaseEventCount = %d, want 2", got.PurchaseEventCount)
		}
		if len(got.ByVendor) != 2 {
			t.Errorf("len(ByVendor) = %d, want 2 (got=%+v)", len(got.ByVendor), got.ByVendor)
		}
	})

	// Phase 260606-0gh: event_date × created_at axis tests. The pending-review
	// filter must use COALESCE(event_date, created_at::Chicago::date) so a
	// late-discovered May receipt ingested in June is still caught in the May
	// period gate, and an old row ingested during the period but whose
	// event_date is outside the window is correctly excluded.

	t.Run("ready=false when pending row has event_date in range but created_at out of range", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		// One linked event so COGS is non-zero and ready could be true absent pending.
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		// event_date='2026-05-29' (in range), created_at='2026-06-02' (out of range).
		// COALESCE picks event_date → row is IN the period → should block ready.
		ppID := insertPendingPurchaseWithEventDate(t,
			"pp-evt-in-cr-out", "2026-05-29", "2026-06-02 10:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (event_date in range must block)")
		}
		found := false
		for _, id := range got.Completeness.PendingReviewIDs {
			if id == ppID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PendingReviewIDs = %v, want to contain %s", got.Completeness.PendingReviewIDs, ppID)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=true when pending row has event_date out of range but created_at in range", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		// event_date='2026-05-20' (before period), created_at='2026-05-27' (in range).
		// COALESCE picks event_date → row is OUTSIDE the period → must NOT block.
		_ = insertPendingPurchaseWithEventDate(t,
			"pp-evt-out-cr-in", "2026-05-20", "2026-05-27 10:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (event_date out of range must not block). pending=%v",
				got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=false when pending row has NULL event_date and created_at in range", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		// NULL event_date → COALESCE falls back to created_at::Chicago.
		// created_at='2026-05-27' (in range) → row is in the period → must block.
		ppID := insertPendingPurchaseWithEventDate(t,
			"pp-evt-null-cr-in", "", "2026-05-27 10:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (NULL event_date with created_at in range must block)")
		}
		found := false
		for _, id := range got.Completeness.PendingReviewIDs {
			if id == ppID {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PendingReviewIDs = %v, want to contain %s", got.Completeness.PendingReviewIDs, ppID)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	t.Run("ready=true when pending row has NULL event_date and created_at out of range", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)
		// NULL event_date → COALESCE falls back to created_at::Chicago.
		// created_at='2026-06-05' (after period) → row is outside the period → must NOT block.
		_ = insertPendingPurchaseWithEventDate(t,
			"pp-evt-null-cr-out", "", "2026-06-05 10:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (NULL event_date with created_at out of range must not block). pending=%v",
				got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.UnlinkedLineItemIDs) != 0 {
			t.Errorf("UnlinkedLineItemIDs = %v, want []", got.Completeness.UnlinkedLineItemIDs)
		}
	})

	// Phase 260606-9y0: tracked_bank_tx_ids surfaces every bank_tx_id HQ
	// has touched for the period across all states. Sales-processor will
	// diff this against Mercury's own transaction list to detect "Mercury
	// has it, HQ hasn't ingested it yet" gaps.

	t.Run("tracked_bank_tx_ids: empty period renders [] not null", func(t *testing.T) {
		resetFixtures(t)
		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		if got.TrackedBankTxIDs == nil {
			t.Fatalf("TrackedBankTxIDs is nil; want empty slice")
		}
		if len(got.TrackedBankTxIDs) != 0 {
			t.Fatalf("len(TrackedBankTxIDs) = %d, want 0", len(got.TrackedBankTxIDs))
		}
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !bytes.Contains(raw, []byte(`"tracked_bank_tx_ids":[]`)) {
			t.Errorf("JSON missing `\"tracked_bank_tx_ids\":[]` — got %s", string(raw))
		}
	})

	t.Run("tracked_bank_tx_ids: all states present, deduped, sorted", func(t *testing.T) {
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")

		// A: purchase_events row, confirmed (no pending_purchases counterpart).
		var eventA string
		err := testPool.QueryRow(t.Context(),
			`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
			 VALUES ($1, 'tx-A', '2026-05-26'::date, 0, 0, 'COGS')
			 RETURNING id::text`, vendorID).Scan(&eventA)
		if err != nil {
			t.Fatalf("insert event A: %v", err)
		}

		// B: pending_purchases, untouched (confirmed_at/discarded_at NULL).
		_ = insertPendingPurchaseWithEventDate(t,
			"tx-B", "2026-05-27", "2026-05-27 10:00:00-05:00", "")

		// C: pending_purchases with discarded_at set.
		_, err = testPool.Exec(t.Context(),
			`INSERT INTO pending_purchases
			   (bank_tx_id, bank_total, vendor, items, event_date, created_at, discarded_at)
			 VALUES ('tx-C', 0, 'TestVendor', '[]'::jsonb,
			         '2026-05-28'::date, '2026-05-28 10:00:00-05:00'::timestamptz, now())`)
		if err != nil {
			t.Fatalf("insert pending C: %v", err)
		}

		// D: pending_purchases WITH a matching purchase_events row (confirm
		// path). UNION must dedupe to a single entry.
		_, err = testPool.Exec(t.Context(),
			`INSERT INTO pending_purchases
			   (bank_tx_id, bank_total, vendor, items, event_date, created_at, confirmed_at)
			 VALUES ('tx-D', 0, 'TestVendor', '[]'::jsonb,
			         '2026-05-29'::date, '2026-05-29 10:00:00-05:00'::timestamptz, now())`)
		if err != nil {
			t.Fatalf("insert pending D: %v", err)
		}
		_, err = testPool.Exec(t.Context(),
			`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
			 VALUES ($1, 'tx-D', '2026-05-29'::date, 0, 0, 'COGS')`, vendorID)
		if err != nil {
			t.Fatalf("insert event D: %v", err)
		}

		// Out-of-period sentinel: must NOT appear.
		_, err = testPool.Exec(t.Context(),
			`INSERT INTO pending_purchases
			   (bank_tx_id, bank_total, vendor, items, event_date, created_at)
			 VALUES ('tx-Z-out-of-period', 0, 'TestVendor', '[]'::jsonb,
			         '2026-06-15'::date, '2026-06-15 10:00:00-05:00'::timestamptz)`)
		if err != nil {
			t.Fatalf("insert pending Z: %v", err)
		}

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		want := []string{"tx-A", "tx-B", "tx-C", "tx-D"}
		if len(got.TrackedBankTxIDs) != len(want) {
			t.Fatalf("TrackedBankTxIDs = %v, want %v", got.TrackedBankTxIDs, want)
		}
		for i, id := range want {
			if got.TrackedBankTxIDs[i] != id {
				t.Errorf("TrackedBankTxIDs[%d] = %q, want %q (full=%v)",
					i, got.TrackedBankTxIDs[i], id, got.TrackedBankTxIDs)
			}
		}
	})

	t.Run("tracked_bank_tx_ids: period boundary uses COALESCE(event_date, created_at::Chicago)", func(t *testing.T) {
		// Mirrors the existing pending-gate event_date × created_at axis
		// tests above — the UNION's pending half must agree with the
		// pending-review filter on which rows belong to the period.
		resetFixtures(t)

		// Row in: event_date in range, created_at out of range. COALESCE
		// picks event_date → must appear.
		_ = insertPendingPurchaseWithEventDate(t,
			"tx-in-by-event-date", "2026-05-29", "2026-06-02 10:00:00-05:00", "")

		// Row out: event_date out of range, created_at in range. COALESCE
		// picks event_date → must NOT appear.
		_ = insertPendingPurchaseWithEventDate(t,
			"tx-out-by-event-date", "2026-05-20", "2026-05-27 10:00:00-05:00", "")

		// Row in: NULL event_date, created_at in range. COALESCE falls
		// back to created_at → must appear.
		_ = insertPendingPurchaseWithEventDate(t,
			"tx-in-by-created-at", "", "2026-05-27 10:00:00-05:00", "")

		// Row out: NULL event_date, created_at out of range. COALESCE
		// falls back to created_at → must NOT appear.
		_ = insertPendingPurchaseWithEventDate(t,
			"tx-out-by-created-at", "", "2026-06-05 10:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d", code)
		}
		want := []string{"tx-in-by-created-at", "tx-in-by-event-date"}
		if len(got.TrackedBankTxIDs) != len(want) {
			t.Fatalf("TrackedBankTxIDs = %v, want %v", got.TrackedBankTxIDs, want)
		}
		for i, id := range want {
			if got.TrackedBankTxIDs[i] != id {
				t.Errorf("TrackedBankTxIDs[%d] = %q, want %q (full=%v)",
					i, got.TrackedBankTxIDs[i], id, got.TrackedBankTxIDs)
			}
		}
	})
}
