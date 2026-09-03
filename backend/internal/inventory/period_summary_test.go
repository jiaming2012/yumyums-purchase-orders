package inventory

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	// Computed BEFORE the fallback: the fallback is the *unset* case, and the
	// unset case still skips. See internal/testdb for the asymmetry.
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
		// DB_TEST_URL unset and the local fallback is not there — leave
		// testPool nil so TestPeriodSummary skips.
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		testdb.ExitIfRequested(requested, dbURL, "ping", err)
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
	// Phase 260606-jvs: default mercury_category='COGS' + reason='no_attachment_on_bank_tx'
	// so the row is a "blocking" pending under the narrowed gate (the prior call sites
	// of this helper all asserted the row blocks ready). Tests that need a non-blocking
	// row use insertPendingPurchaseFull instead.
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, created_at, confirmed_at, discarded_at, mercury_category, reason)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, $2::timestamptz, ` + confirmedSQL + `, ` + discardedSQL + `, 'COGS', 'no_attachment_on_bank_tx')
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
	// Phase 260607-fxl: column "display_name" and column "role" were removed
	// by migrations 0017_users_naming.sql (display_name → first_name+last_name)
	// and the 11-onboarding role-array refactor (role → roles[]). The helper
	// was already broken on HEAD prior to this fix; updating to the current
	// schema is a Rule 3 unblock so the confirm tests can resolve confirmed_by.
	//
	// resetFixtures does NOT truncate users, so use ON CONFLICT (email) to
	// stay idempotent across repeated test runs against the same DB.
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ($1, $2, $3, ARRAY['admin']::text[], 'active')
		 ON CONFLICT (email) DO UPDATE SET status = EXCLUDED.status
		 RETURNING id::text`,
		email, "Test", "User").Scan(&id)
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
// createdAt is a timestamptz string. reason is a string sentinel or "" for the
// default blocking reason ("no_attachment_on_bank_tx").
// Returns the inserted id::text.
//
// Phase 260606-jvs: defaults mercury_category='COGS' so the row passes the
// narrowed pending-gate allowlist filter. Defaults reason to
// 'no_attachment_on_bank_tx' (the only blocking reason) when caller passes "".
// Tests that need a non-blocking row use insertPendingPurchaseFull instead.
func insertPendingPurchaseWithEventDate(t *testing.T, bankTxID, eventDate, createdAt, reason string) string {
	t.Helper()
	var ed *string
	if eventDate != "" {
		ed = &eventDate
	}
	if reason == "" {
		reason = "no_attachment_on_bank_tx"
	}
	var id string
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at, mercury_category)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, $2::date, $3, $4::timestamptz, 'COGS')
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q, bankTxID, ed, reason, createdAt).Scan(&id); err != nil {
		t.Fatalf("insert pending_purchase: %v", err)
	}
	return id
}

// insertPendingPurchaseFull inserts an unconfirmed/undiscarded pending row with
// full control over the four fields the narrowed /period-summary contract cares
// about: reason, mercuryCategory, vendor, bankTotal.
//
// Pass mercuryCategory == "" for SQL NULL (uncategorised — excluded by the
// allowlist filter). Pass eventDate == "" for SQL NULL (the app-timezone cast
// of created_at becomes the period-filter input — see pendingPeriodDateExpr). Pass reason == "" for SQL NULL (treated
// as non-blocking by the narrowed gate since reason != 'no_attachment_on_bank_tx').
func insertPendingPurchaseFull(t *testing.T, bankTxID, eventDate, createdAt, reason, mercuryCategory, vendor string, bankTotal float64) string {
	t.Helper()
	var (
		ed *string
		rs *string
		mc *string
	)
	if eventDate != "" {
		ed = &eventDate
	}
	if reason != "" {
		rs = &reason
	}
	if mercuryCategory != "" {
		mc = &mercuryCategory
	}
	var id string
	q := `INSERT INTO pending_purchases
	        (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at, mercury_category)
	      VALUES ($1, $2, $3, '[]'::jsonb, $4::date, $5, $6::timestamptz, $7)
	      RETURNING id::text`
	if err := testPool.QueryRow(t.Context(), q,
		bankTxID, bankTotal, vendor, ed, rs, createdAt, mc).Scan(&id); err != nil {
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

		// Pending purchase created on 2026-05-27, comfortably mid-day so it is in
		// range in ANY North American zone. The -05:00 offset in the literal is
		// just how the fixture was written; nothing here turns on it.
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
		// Phase 260607-fxl: the empty-items confirm branch now requires
		// reason='no_attachment_on_bank_tx' on the pending row. The original
		// insertPendingPurchaseWithBankTotal helper writes reason=NULL by
		// default; inject the allowlist value here so this test continues to
		// exercise the 260605-pk1 no-attachment confirm flow.
		if _, err := testPool.Exec(t.Context(),
			`UPDATE pending_purchases SET reason = 'no_attachment_on_bank_tx' WHERE id = $1`,
			ppID); err != nil {
			t.Fatalf("set reason: %v", err)
		}

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
	// filter must use COALESCE(event_date, created_at::app-tz::date) so a
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
		// NULL event_date → COALESCE falls back to the app-timezone cast of created_at.
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
		// NULL event_date → COALESCE falls back to the app-timezone cast of created_at.
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

	t.Run("tracked_bank_tx_ids: period boundary uses COALESCE(event_date, created_at::app-tz)", func(t *testing.T) {
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

	// pending_review_details: parallel array to pending_review_ids exposed for
	// service-token callers (sales-processor) so they can render rich pending
	// review context without a second round trip to the cookie-gated
	// /purchases/pending endpoint. Field order MUST match the IDs array
	// (same SELECT, same WHERE/ORDER BY, same scan loop in handler.go).

	t.Run("pending_review_details parity with pending_review_ids", func(t *testing.T) {
		resetFixtures(t)
		// Three pending rows in range, distinct event_dates so order is stable.
		_ = insertPendingPurchaseWithEventDate(t,
			"mx-parity-1", "2026-05-26", "2026-05-26 12:00:00-05:00", "")
		_ = insertPendingPurchaseWithEventDate(t,
			"mx-parity-2", "2026-05-28", "2026-05-28 12:00:00-05:00", "")
		_ = insertPendingPurchaseWithEventDate(t,
			"mx-parity-3", "2026-05-30", "2026-05-30 12:00:00-05:00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		ids := got.Completeness.PendingReviewIDs
		details := got.Completeness.PendingReviewDetails
		if details == nil {
			t.Fatalf("PendingReviewDetails is nil; want non-nil slice")
		}
		if len(details) != len(ids) {
			t.Fatalf("len(details) = %d, len(ids) = %d; want equal (details=%v ids=%v)",
				len(details), len(ids), details, ids)
		}
		if len(ids) != 3 {
			t.Fatalf("len(ids) = %d, want 3", len(ids))
		}
		for i := range ids {
			if details[i].ID != ids[i] {
				t.Errorf("details[%d].ID = %q, ids[%d] = %q; want equal",
					i, details[i].ID, i, ids[i])
			}
		}
	})

	t.Run("pending_review_details populates vendor/event_date/bank_total/reason", func(t *testing.T) {
		// Phase 260606-jvs: only rows with reason='no_attachment_on_bank_tx' AND
		// mercury_category in allowlist surface in pending_review_details now.
		// Previously asserted with reason='tax_mismatch'; updated to the only
		// blocking reason. The serialisation behaviour the test pins (Reason
		// pointer non-nil, value = reason text) is unchanged in shape.
		resetFixtures(t)
		var id string
		q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, created_at, mercury_category)
		      VALUES ('mx-100', -87.50, 'Restaurant Depot', '[]'::jsonb, '2026-05-28'::date, 'no_attachment_on_bank_tx', '2026-05-28 12:00:00-05:00'::timestamptz, 'COGS')
		      RETURNING id::text`
		if err := testPool.QueryRow(t.Context(), q).Scan(&id); err != nil {
			t.Fatalf("insert pending: %v", err)
		}

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		details := got.Completeness.PendingReviewDetails
		if len(details) != 1 {
			t.Fatalf("len(details) = %d, want 1 (full=%v)", len(details), details)
		}
		d := details[0]
		if d.ID != id {
			t.Errorf("details[0].ID = %q, want %q", d.ID, id)
		}
		if d.BankTxID != "mx-100" {
			t.Errorf("details[0].BankTxID = %q, want %q", d.BankTxID, "mx-100")
		}
		if d.Vendor != "Restaurant Depot" {
			t.Errorf("details[0].Vendor = %q, want %q", d.Vendor, "Restaurant Depot")
		}
		if d.EventDate != "2026-05-28" {
			t.Errorf("details[0].EventDate = %q, want %q", d.EventDate, "2026-05-28")
		}
		if d.BankTotal != -87.50 {
			t.Errorf("details[0].BankTotal = %v, want %v", d.BankTotal, -87.50)
		}
		if d.Reason == nil {
			t.Errorf("details[0].Reason = nil, want non-nil pointer to %q", "no_attachment_on_bank_tx")
		} else if *d.Reason != "no_attachment_on_bank_tx" {
			t.Errorf("*details[0].Reason = %q, want %q", *d.Reason, "no_attachment_on_bank_tx")
		}
	})

	t.Run("pending_review_details event_date falls back to the app-tz cast of created_at", func(t *testing.T) {
		resetFixtures(t)
		// NULL event_date, created_at 2026-05-29 22:02:00 UTC.
		// New York = UTC-4 (EDT in late May) → local 2026-05-29 18:02 → date 2026-05-29.
		//
		// 🛑 This instant resolves to the SAME date in Chicago (17:02) and New
		// York, so this case deliberately does NOT distinguish the two zones —
		// it pins the COALESCE fallback, not the zone. The zone is pinned by
		// TestPeriodSummary_PendingPeriodBoundaryIsTheAppTimezone at the end of
		// this file, which places rows in the one hour where they disagree.
		_ = insertPendingPurchaseWithEventDate(t,
			"mx-200", "", "2026-05-29 22:02:00+00", "")

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		details := got.Completeness.PendingReviewDetails
		if len(details) != 1 {
			t.Fatalf("len(details) = %d, want 1 (full=%v)", len(details), details)
		}
		if details[0].EventDate != "2026-05-29" {
			t.Errorf("details[0].EventDate = %q, want %q (app-timezone cast of created_at)",
				details[0].EventDate, "2026-05-29")
		}
	})

	t.Run("pending_review_details vendor='' serializes as empty string", func(t *testing.T) {
		// Note: pending_purchases.vendor is NOT NULL at the schema level
		// (tightened by quick task 260606-hew which always populates vendor
		// from Mercury BankDescription). The SQL still uses COALESCE(vendor,'')
		// defensively; this test pins the closest observable behaviour —
		// an empty-string vendor surfaces as "" in the API response.
		// Phase 260606-jvs: added mercury_category='COGS' + reason='no_attachment_on_bank_tx'
		// so the row stays a blocking pending under the narrowed gate.
		resetFixtures(t)
		var id string
		q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, created_at, reason, mercury_category)
		      VALUES ('mx-300', -10.00, '', '[]'::jsonb, '2026-05-27'::date, '2026-05-27 12:00:00-05:00'::timestamptz, 'no_attachment_on_bank_tx', 'COGS')
		      RETURNING id::text`
		if err := testPool.QueryRow(t.Context(), q).Scan(&id); err != nil {
			t.Fatalf("insert pending: %v", err)
		}

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		details := got.Completeness.PendingReviewDetails
		if len(details) != 1 {
			t.Fatalf("len(details) = %d, want 1 (full=%v)", len(details), details)
		}
		if details[0].Vendor != "" {
			t.Errorf("details[0].Vendor = %q, want empty string", details[0].Vendor)
		}
	})

	// Phase 260606-jvs: the prior `pending_review_details reason=NULL omitted from JSON`
	// test was removed — under the narrowed gate, only rows with
	// reason='no_attachment_on_bank_tx' surface in pending_review_details, so a
	// NULL-reason row can never appear there. The omitempty serialisation
	// behaviour on `PendingReviewDetail.Reason` remains a structural property
	// of the struct (json:"reason,omitempty") — not exercised here because the
	// surface is unreachable post-narrowing.

	t.Run("pending_review_details serializes as [] when empty period", func(t *testing.T) {
		resetFixtures(t)
		req := httptest.NewRequest(http.MethodGet, "/?from="+from+"&to="+to, nil)
		rec := httptest.NewRecorder()
		PeriodSummaryHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"pending_review_details":[]`) {
			t.Errorf("JSON missing `\"pending_review_details\":[]` (want non-null empty array).\nbody=%s",
				rec.Body.String())
		}
	})

	// Phase 260606-jvs: 2×2 truth table on (mercury_category in allowlist) ×
	// (reason == 'no_attachment_on_bank_tx') for the narrowed completeness
	// gate, plus the rolled-into-COGS + by_vendor merge semantics for the
	// non-blocking food-category branch. See 260606-jvs-HANDOFF.md §4.
	//
	// Allowlist sentinel is the default ["COGS"] used by callHandler — keeps
	// these tests aligned with the existing convention in this file.

	t.Run("case_a_food_no_attachment_blocks", func(t *testing.T) {
		// Case A: food category + reason='no_attachment_on_bank_tx' → ONLY blocker.
		// Blocks ready, surfaces in pending_review_ids, does NOT roll into COGS
		// (blocking rows stay out of the aggregate per the data-model invariant).
		resetFixtures(t)
		ppID := insertPendingPurchaseFull(t,
			"jvs-case-a", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"no_attachment_on_bank_tx", "COGS", "Restaurant Depot", -50.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if got.Completeness.Ready {
			t.Errorf("Ready = true, want false (case A must block)")
		}
		if len(got.Completeness.PendingReviewIDs) != 1 || got.Completeness.PendingReviewIDs[0] != ppID {
			t.Errorf("PendingReviewIDs = %v, want [%s]", got.Completeness.PendingReviewIDs, ppID)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0 (blocking row stays out of COGS)", got.COGSExclTax)
		}
	})

	t.Run("case_b_food_parse_failed_rolls_into_cogs", func(t *testing.T) {
		// Case B: food category + parse-failed reason → non-blocking, rolled
		// into COGS at ABS(bank_total).
		resetFixtures(t)
		ppID := insertPendingPurchaseFull(t,
			"jvs-case-b", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"Receipt could not be parsed automatically", "COGS", "Save A Lot", -19.28)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (case B must NOT block). pending=%v unlinked=%v",
				got.Completeness.PendingReviewIDs, got.Completeness.UnlinkedLineItemIDs)
		}
		for _, id := range got.Completeness.PendingReviewIDs {
			if id == ppID {
				t.Errorf("PendingReviewIDs contains %s, want it excluded (non-blocking)", ppID)
			}
		}
		if got.COGSExclTax != 19.28 {
			t.Errorf("COGSExclTax = %v, want 19.28 (ABS(bank_total))", got.COGSExclTax)
		}
		// Tax assumption: pending bank_total flows into both excl + incl.
		if got.COGSInclTax != 19.28 {
			t.Errorf("COGSInclTax = %v, want 19.28 (pending bank_total flows into both)", got.COGSInclTax)
		}
		if got.PurchaseEventCount != 1 {
			t.Errorf("PurchaseEventCount = %d, want 1 (eligible pending counts)", got.PurchaseEventCount)
		}
	})

	t.Run("case_c_non_food_no_attachment_does_not_block_or_roll", func(t *testing.T) {
		// Case C: non-food category + no_attachment → non-blocking, NOT in COGS.
		// Amazon refund scenario from the morning failure.
		resetFixtures(t)
		_ = insertPendingPurchaseFull(t,
			"jvs-case-c", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"no_attachment_on_bank_tx", "Software, SaaS & Subscriptions", "Amazon", -14.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (case C is non-blocking). pending=%v",
				got.Completeness.PendingReviewIDs)
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0 (non-food never rolls into COGS)", got.COGSExclTax)
		}
	})

	t.Run("case_d_non_food_parse_failed_does_not_block_or_roll", func(t *testing.T) {
		// Case D: non-food category + parse-failed reason → non-blocking, NOT in COGS.
		resetFixtures(t)
		_ = insertPendingPurchaseFull(t,
			"jvs-case-d", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"Receipt could not be parsed automatically", "Software, SaaS & Subscriptions", "DropboxPro", -9.99)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (case D is non-blocking)")
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0", got.COGSExclTax)
		}
	})

	t.Run("null_mercury_category_with_no_attachment_does_not_block", func(t *testing.T) {
		// NULL mercury_category + no_attachment_on_bank_tx → NOT blocking, NOT in COGS.
		// Postgres `column = ANY($1)` returns NULL (not true) when column IS NULL,
		// which is consistent with the principle that uncategorised txns are
		// operator-triage chores, not data blockers.
		resetFixtures(t)
		_ = insertPendingPurchaseFull(t,
			"jvs-null-cat", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"no_attachment_on_bank_tx", "" /* NULL */, "Unknown Vendor", -42.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (NULL category must not block)")
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0", got.COGSExclTax)
		}
	})

	t.Run("date_filter_still_applies_to_blocking_row_out_of_period", func(t *testing.T) {
		// Case-A blocking row whose period anchor (event_date or fallback
		// created_at::app-tz) is outside the window must NOT block, must NOT
		// surface in pending_review_ids, and must NOT roll into COGS.
		resetFixtures(t)
		_ = insertPendingPurchaseFull(t,
			"jvs-case-a-oop", "2026-04-15", "2026-04-15 10:00:00-05:00",
			"no_attachment_on_bank_tx", "COGS", "Restaurant Depot", -50.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if !got.Completeness.Ready {
			t.Errorf("Ready = false, want true (out-of-period must not block)")
		}
		if len(got.Completeness.PendingReviewIDs) != 0 {
			t.Errorf("PendingReviewIDs = %v, want []", got.Completeness.PendingReviewIDs)
		}
		if got.COGSExclTax != 0 {
			t.Errorf("COGSExclTax = %v, want 0", got.COGSExclTax)
		}
	})

	t.Run("pending_review_details_parity_under_narrowed_gate", func(t *testing.T) {
		// Single case-A row → pending_review_ids has 1 entry, pending_review_details
		// has 1 entry, IDs match index-wise.
		resetFixtures(t)
		ppID := insertPendingPurchaseFull(t,
			"jvs-parity", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"no_attachment_on_bank_tx", "COGS", "Restaurant Depot", -50.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		ids := got.Completeness.PendingReviewIDs
		details := got.Completeness.PendingReviewDetails
		if len(ids) != 1 || ids[0] != ppID {
			t.Fatalf("PendingReviewIDs = %v, want [%s]", ids, ppID)
		}
		if len(details) != 1 || details[0].ID != ppID {
			t.Fatalf("PendingReviewDetails IDs = %+v, want one row with id=%s", details, ppID)
		}
		if details[0].BankTxID != "jvs-parity" {
			t.Errorf("details[0].BankTxID = %q, want %q", details[0].BankTxID, "jvs-parity")
		}
	})

	t.Run("case_b_by_vendor_match_merges_into_vendor_row", func(t *testing.T) {
		// One confirmed RD event ($100) + one case-B Save A Lot pending row ($19).
		// cogs_excl_tax == 119; by_vendor has BOTH an RD row at $100 and a
		// Save A Lot row at $19, each with a real (non-empty) vendor_id.
		resetFixtures(t)
		depotID := insertVendor(t, "Restaurant Depot")
		salID := insertVendor(t, "Save A Lot")
		piID := insertPurchaseItem(t, "Salmon")
		// Confirmed RD event: 1 * $100 = $100.
		insertEventAndLineWithCategory(t,
			depotID, "2026-05-26", 0, 100.00, 100.00, 1, piID, "COGS")
		// Case-B Save A Lot pending: ABS(-19) = $19.
		_ = insertPendingPurchaseFull(t,
			"jvs-byv-match", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"Receipt could not be parsed automatically", "COGS", "Save A Lot", -19.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if diff := got.COGSExclTax - 119.00; diff > 0.01 || diff < -0.01 {
			t.Errorf("COGSExclTax = %v, want 119.00", got.COGSExclTax)
		}
		// Find each vendor row.
		var foundDepot, foundSAL bool
		for _, row := range got.ByVendor {
			switch row.VendorName {
			case "Restaurant Depot":
				foundDepot = true
				if row.VendorID != depotID {
					t.Errorf("RD VendorID = %q, want %q (real vendor id)", row.VendorID, depotID)
				}
				if diff := row.TotalExclTax - 100.00; diff > 0.01 || diff < -0.01 {
					t.Errorf("RD TotalExclTax = %v, want 100.00", row.TotalExclTax)
				}
			case "Save A Lot":
				foundSAL = true
				if row.VendorID != salID {
					t.Errorf("SAL VendorID = %q, want %q (real vendor id)", row.VendorID, salID)
				}
				if diff := row.TotalExclTax - 19.00; diff > 0.01 || diff < -0.01 {
					t.Errorf("SAL TotalExclTax = %v, want 19.00", row.TotalExclTax)
				}
			}
		}
		if !foundDepot {
			t.Errorf("by_vendor missing Restaurant Depot row; got=%+v", got.ByVendor)
		}
		if !foundSAL {
			t.Errorf("by_vendor missing Save A Lot row; got=%+v", got.ByVendor)
		}
	})

	t.Run("case_b_by_vendor_unmatched_renders_with_empty_vendor_id", func(t *testing.T) {
		// Case-B pending with vendor text that has no vendors.name match →
		// surfaces in by_vendor with vendor_id == "" and vendor_name == original text.
		resetFixtures(t)
		_ = insertPendingPurchaseFull(t,
			"jvs-byv-unmatched", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"Receipt could not be parsed automatically", "COGS",
			"Brand New Vendor not in vendors table", -27.50)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		var found bool
		for _, row := range got.ByVendor {
			if row.VendorName == "Brand New Vendor not in vendors table" {
				found = true
				if row.VendorID != "" {
					t.Errorf("unmatched VendorID = %q, want empty string", row.VendorID)
				}
				if diff := row.TotalExclTax - 27.50; diff > 0.01 || diff < -0.01 {
					t.Errorf("unmatched TotalExclTax = %v, want 27.50", row.TotalExclTax)
				}
			}
		}
		if !found {
			t.Errorf("by_vendor missing unmatched row; got=%+v", got.ByVendor)
		}
	})

	t.Run("case_b_by_vendor_vendor_name_fuzz_joins_case_and_trim_insensitive", func(t *testing.T) {
		// Pre-insert vendors.name='Save A Lot'. Case-B pending with vendor=
		// 'save a lot ' (lowercase + trailing space) must join via LOWER(TRIM())
		// and merge into the existing Save A Lot row — no duplicate row with
		// vendor_id == "".
		resetFixtures(t)
		salID := insertVendor(t, "Save A Lot")
		_ = insertPendingPurchaseFull(t,
			"jvs-byv-fuzz", "2026-05-27", "2026-05-27 10:00:00-05:00",
			"Receipt could not be parsed automatically", "COGS", "save a lot ", -19.00)

		code, got := callHandler(t, from, to)
		if code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		// Exactly one row, real vendor_id, name canonicalised to "Save A Lot".
		if len(got.ByVendor) != 1 {
			t.Fatalf("len(ByVendor) = %d, want 1 (no duplicate unmatched row); got=%+v",
				len(got.ByVendor), got.ByVendor)
		}
		row := got.ByVendor[0]
		if row.VendorID != salID {
			t.Errorf("VendorID = %q, want %q (real vendor id, not empty)", row.VendorID, salID)
		}
		if row.VendorName != "Save A Lot" {
			t.Errorf("VendorName = %q, want %q (canonicalised)", row.VendorName, "Save A Lot")
		}
		if diff := row.TotalExclTax - 19.00; diff > 0.01 || diff < -0.01 {
			t.Errorf("TotalExclTax = %v, want 19.00", row.TotalExclTax)
		}
	})
}

// ─── Phase 260607-fxl: ConfirmPendingPurchaseHandler gates ──────────────────

// confirmPendingHelper performs the boilerplate for invoking
// ConfirmPendingPurchaseHandler with a seeded user in context. Returns the
// raw httptest.ResponseRecorder so the caller can assert on status + body.
func confirmPendingHelper(t *testing.T, userID string, body ConfirmPendingInput) *httptest.ResponseRecorder {
	t.Helper()
	buf, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal confirm body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/inventory/purchases/confirm", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), auth.CtxKeyUser, &auth.User{
		ID:    userID,
		Email: "confirm-test@yumyums.test",
	})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	ConfirmPendingPurchaseHandler(testPool).ServeHTTP(rec, req)
	return rec
}

// TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached asserts the new
// FIX B gate: a pending row whose reason is the parse-failure sentinel (NOT
// the no-attachment sentinel) cannot be confirmed with empty line_items.
// The receipt IS attached — the operator must itemize or discard.
func TestConfirmPending_RejectsEmptyItemsWhenReceiptAttached(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	userID := insertTestUser(t, "confirm-reject-empty@yumyums.test")

	// Seed a parse-failed pending row directly (no helper covers this reason).
	var ppID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, created_at)
		 VALUES ($1, $2, $3, '[]'::jsonb, $4, $5::timestamptz)
		 RETURNING id::text`,
		"fxl-reject-empty-tx", -50.00, "TestVendor",
		"Receipt could not be parsed automatically",
		"2026-05-27 10:00:00-05:00",
	).Scan(&ppID); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	rec := confirmPendingHelper(t, userID, ConfirmPendingInput{
		ID:         ppID,
		VendorName: "TestVendor",
		EventDate:  "2026-05-27",
		LineItems:  nil, // empty — should be rejected
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "empty_items_not_allowed" {
		t.Errorf("error = %q, want %q", body["error"], "empty_items_not_allowed")
	}
}

// TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment proves the 260605-pk1
// no-attachment confirm flow still returns 200 after the new gate. This is
// the regression guard for that pre-existing flow.
func TestConfirmPending_AcceptsEmptyItemsWhenNoAttachment(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	insertNoItemizedReceiptSeed(t)
	userID := insertTestUser(t, "confirm-accept-empty@yumyums.test")
	ppID := insertPendingPurchaseWithBankTotal(t,
		"fxl-accept-empty-tx", -75.00, "2026-05-27 10:00:00-05:00")
	// Set the allowlist reason so the new gate permits the empty-items branch.
	if _, err := testPool.Exec(t.Context(),
		`UPDATE pending_purchases SET reason = 'no_attachment_on_bank_tx' WHERE id = $1`,
		ppID); err != nil {
		t.Fatalf("set reason: %v", err)
	}

	rec := confirmPendingHelper(t, userID, ConfirmPendingInput{
		ID:         ppID,
		VendorName: "TestVendor",
		EventDate:  "2026-05-27",
		LineItems:  nil,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	// Sanity: purchase_events row was created.
	var eventCount int
	if err := testPool.QueryRow(t.Context(),
		`SELECT COUNT(*) FROM purchase_events WHERE bank_tx_id='fxl-accept-empty-tx'`,
	).Scan(&eventCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventCount != 1 {
		t.Errorf("event rows = %d, want 1", eventCount)
	}
}

// TestConfirmPending_RejectsTotalMismatchWith422 asserts the upgrade of the
// old text-400 total-mismatch rejection to a structured 422 envelope with
// numeric line_total + bank_total so the FE can render them directly.
func TestConfirmPending_RejectsTotalMismatchWith422(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)
	// Seed the placeholder catalog row so the line_items[].purchase_item_id
	// references something real even though the handler short-circuits
	// before insertion. resetFixtures TRUNCATEd it.
	seedID := insertNoItemizedReceiptSeed(t)
	userID := insertTestUser(t, "confirm-mismatch@yumyums.test")

	// Seed a parse-failed pending row with bank_total=-50.00.
	var ppID string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, created_at)
		 VALUES ($1, $2, $3, '[]'::jsonb, $4, $5::timestamptz)
		 RETURNING id::text`,
		"fxl-mismatch-tx", -50.00, "TestVendor",
		"Receipt could not be parsed automatically",
		"2026-05-27 10:00:00-05:00",
	).Scan(&ppID); err != nil {
		t.Fatalf("seed pending: %v", err)
	}

	rec := confirmPendingHelper(t, userID, ConfirmPendingInput{
		ID:         ppID,
		VendorName: "TestVendor",
		EventDate:  "2026-05-27",
		// 1 x $42.00 = $42.00 ≠ $50.00 abs bank total → mismatch.
		LineItems: []CreateLineItemInput{
			{PurchaseItemID: &seedID, Description: "x", Quantity: 1, Price: 42.00, IsCase: false},
		},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "total_mismatch" {
		t.Errorf("error = %v, want %q", body["error"], "total_mismatch")
	}
	gotLineTotal, ok := body["line_total"].(float64)
	if !ok {
		t.Errorf("line_total type = %T, want float64", body["line_total"])
	}
	if gotLineTotal != 42.00 {
		t.Errorf("line_total = %v, want 42.00", gotLineTotal)
	}
	gotBankTotal, ok := body["bank_total"].(float64)
	if !ok {
		t.Errorf("bank_total type = %T, want float64", body["bank_total"])
	}
	if gotBankTotal != 50.00 {
		t.Errorf("bank_total = %v, want 50.00", gotBankTotal)
	}
}

// ─── Phase 260607-koi: RetryParsePendingPurchaseHandler ─────────────────────
//
// The retry-parse endpoint nulls the parse_error column on a stuck pending row
// so the next worker sync cycle (260607-fxl parseFailedRetry branch) will
// re-attempt parsing through Sonnet. The 4 tests below pin the 4 dispositions:
//   - 200 + parse_error nulled when the row is pending and has a parse_error
//   - 404 pending_purchase_not_found when the id does not exist
//   - 422 row_not_pending when the row is confirmed or discarded
//   - 422 nothing_to_retry when parse_error is already NULL

// retryParseHelper performs the boilerplate for invoking
// RetryParsePendingPurchaseHandler with the row id encoded into the URL path
// (the route uses chi.URLParam, NOT a JSON body). Wires the chi router so
// chi.URLParam("id") resolves correctly inside the handler.
func retryParseHelper(t *testing.T, id string) *httptest.ResponseRecorder {
	t.Helper()
	router := chi.NewRouter()
	router.Post("/api/v1/inventory/purchases/pending/{id}/retry-parse",
		RetryParsePendingPurchaseHandler(testPool))
	req := httptest.NewRequest(http.MethodPost,
		"/api/v1/inventory/purchases/pending/"+id+"/retry-parse", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// TestRetryParse_ClearsParseError verifies the happy path: a pending row with a
// parse_error set is updated so parse_error becomes NULL and the row is
// returned in the 200 response body.
func TestRetryParse_ClearsParseError(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	ppID := insertPendingPurchaseWithEventDate(t,
		"koi-clears-tx", "2026-05-27", "2026-05-27 10:00:00-05:00",
		"Receipt could not be parsed automatically")
	if _, err := testPool.Exec(t.Context(),
		`UPDATE pending_purchases SET parse_error = $1 WHERE id = $2`,
		"haiku: boom; sonnet: boom", ppID); err != nil {
		t.Fatalf("set parse_error: %v", err)
	}

	rec := retryParseHelper(t, ppID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	// Response body is the full PendingPurchase row with parse_error nil/empty.
	var body PendingPurchase
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.ID != ppID {
		t.Errorf("response id = %q, want %q", body.ID, ppID)
	}
	if body.ParseError != nil && *body.ParseError != "" {
		t.Errorf("response parse_error = %q, want nil or empty", *body.ParseError)
	}

	// DB sanity: column is now NULL.
	var parseErrAfter sql.NullString
	if err := testPool.QueryRow(t.Context(),
		`SELECT parse_error FROM pending_purchases WHERE id = $1`, ppID,
	).Scan(&parseErrAfter); err != nil {
		t.Fatalf("select parse_error: %v", err)
	}
	if parseErrAfter.Valid {
		t.Errorf("parse_error column = %q, want NULL", parseErrAfter.String)
	}
}

// TestRetryParse_404OnUnknownId verifies that a valid-uuid-but-unknown id
// returns 404 with the existing pending_purchase_not_found envelope (matches
// the UpdatePendingItemsHandler / ConfirmPendingPurchaseHandler convention).
func TestRetryParse_404OnUnknownId(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	// Valid UUID format, but no pending row inserted — Postgres lookup returns
	// ErrNoRows which maps to 404.
	unknownID := "00000000-0000-0000-0000-0000000000aa"
	rec := retryParseHelper(t, unknownID)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "pending_purchase_not_found" {
		t.Errorf("error = %q, want %q", body["error"], "pending_purchase_not_found")
	}
}

// TestRetryParse_422OnConfirmedRow verifies that a confirmed (or discarded)
// row returns the 422 row_not_pending envelope — the operator cannot re-arm a
// row that already finished its lifecycle.
func TestRetryParse_422OnConfirmedRow(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	ppID := insertPendingPurchaseWithEventDate(t,
		"koi-confirmed-tx", "2026-05-27", "2026-05-27 10:00:00-05:00",
		"Receipt could not be parsed automatically")
	if _, err := testPool.Exec(t.Context(),
		`UPDATE pending_purchases SET confirmed_at = NOW(), parse_error = $1 WHERE id = $2`,
		"haiku: boom; sonnet: boom", ppID); err != nil {
		t.Fatalf("set confirmed_at + parse_error: %v", err)
	}

	rec := retryParseHelper(t, ppID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "row_not_pending" {
		t.Errorf("error = %q, want %q", body["error"], "row_not_pending")
	}
	if body["reason"] == "" {
		t.Errorf("reason is empty; want explanatory text")
	}
}

// TestRetryParse_422WhenNothingToRetry verifies the no-op guard: a pending row
// whose parse_error is already NULL returns the 422 nothing_to_retry envelope
// rather than uselessly executing the UPDATE.
func TestRetryParse_422WhenNothingToRetry(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	// insertPendingPurchaseWithEventDate writes parse_error=NULL by default.
	ppID := insertPendingPurchaseWithEventDate(t,
		"koi-noop-tx", "2026-05-27", "2026-05-27 10:00:00-05:00",
		"Receipt could not be parsed automatically")

	rec := retryParseHelper(t, ppID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "nothing_to_retry" {
		t.Errorf("error = %q, want %q", body["error"], "nothing_to_retry")
	}
	if body["reason"] == "" {
		t.Errorf("reason is empty; want explanatory text")
	}
}

// TestRetryParse_ItemsMismatch_Accepted verifies the 260607-s6r broadening:
// a pending row with parse_error=NULL but items populated whose line_total
// doesn't match bank_total is accepted, items are cleared to '[]'::jsonb,
// and reason is reset to the parse-failed sentinel so the worker's
// parseFailedRetry gate matches on next sync.
func TestRetryParse_ItemsMismatch_Accepted(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	// Use insertPendingPurchaseFull so we can control bank_total. Pass
	// reason="no_attachment_on_bank_tx" (any non-NULL non-parse-failed
	// reason works) and bank_total=50.00, then override items to a $100
	// single line so line_total != bank_total triggers the s6r branch.
	ppID := insertPendingPurchaseFull(t,
		"s6r-mismatch-tx",           // bank_tx_id
		"2026-05-27",                // event_date
		"2026-05-27 10:00:00-05:00", // created_at
		"no_attachment_on_bank_tx",  // reason
		"COGS",                      // mercury_category
		"TestVendor",                // vendor
		50.0,                        // bank_total
	)
	if _, err := testPool.Exec(t.Context(),
		`UPDATE pending_purchases
		    SET items = '[{"name":"x","quantity":1,"price":100,"is_case":false}]'::jsonb
		  WHERE id = $1`, ppID); err != nil {
		t.Fatalf("set items: %v", err)
	}

	rec := retryParseHelper(t, ppID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	// DB sanity: items emptied, reason updated, parse_error still NULL,
	// row still pending (confirmed_at unchanged).
	var (
		itemsAfter     []byte
		reasonAfter    sql.NullString
		parseErrAfter  sql.NullString
		confirmedAfter sql.NullTime
	)
	if err := testPool.QueryRow(t.Context(),
		`SELECT items, reason, parse_error, confirmed_at
		   FROM pending_purchases WHERE id = $1`, ppID,
	).Scan(&itemsAfter, &reasonAfter, &parseErrAfter, &confirmedAfter); err != nil {
		t.Fatalf("select after: %v", err)
	}
	if string(itemsAfter) != "[]" {
		t.Errorf("items after = %q, want %q", string(itemsAfter), "[]")
	}
	if !reasonAfter.Valid || reasonAfter.String != "Receipt could not be parsed automatically" {
		t.Errorf("reason after = %v, want %q", reasonAfter, "Receipt could not be parsed automatically")
	}
	if parseErrAfter.Valid {
		t.Errorf("parse_error after = %q, want NULL", parseErrAfter.String)
	}
	if confirmedAfter.Valid {
		t.Errorf("confirmed_at after = %v, want NULL", confirmedAfter.Time)
	}
}

// TestRetryParse_ItemsMatchTotals_StillRejected verifies that the 260607-s6r
// broadening does NOT scope-creep into "any populated items row": when items
// are populated AND totals match within 0.01, the row is healthy and
// retry-parse must still return 422 nothing_to_retry (matches the existing
// koi guard).
func TestRetryParse_ItemsMatchTotals_StillRejected(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	ppID := insertPendingPurchaseFull(t,
		"s6r-match-tx",
		"2026-05-27",
		"2026-05-27 10:00:00-05:00",
		"no_attachment_on_bank_tx",
		"COGS",
		"TestVendor",
		100.0, // bank_total matches the items line_total below
	)
	if _, err := testPool.Exec(t.Context(),
		`UPDATE pending_purchases
		    SET items = '[{"name":"x","quantity":1,"price":100,"is_case":false}]'::jsonb
		  WHERE id = $1`, ppID); err != nil {
		t.Fatalf("set items: %v", err)
	}

	rec := retryParseHelper(t, ppID)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 (body=%s)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "nothing_to_retry" {
		t.Errorf("error = %q, want %q", body["error"], "nothing_to_retry")
	}
}

// TestListPendingPurchases_ReceiptURLsField asserts that ListPendingPurchasesHandler
// includes the receipt_urls array in its response when the column is populated,
// and omits it (rather than returning null) when the column is NULL. This covers
// the backward-compat path for rows written before migration 0070.
func TestListPendingPurchases_ReceiptURLsField(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}
	resetFixtures(t)

	// Row A: receipt_urls populated with two URLs.
	var idA string
	if err := testPool.QueryRow(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, receipt_url, receipt_urls)
		VALUES ($1, $2, $3, '[]'::jsonb, $4, $5::jsonb)
		RETURNING id::text`,
		"pp-ru-a", 100.0, "VendorA",
		"http://spaces/0.jpg",
		`["http://spaces/0.jpg","http://spaces/1.pdf"]`,
	).Scan(&idA); err != nil {
		t.Fatalf("insert row A: %v", err)
	}

	// Row B: receipt_urls NULL — backward-compat legacy row.
	var idB string
	if err := testPool.QueryRow(t.Context(), `
		INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, receipt_url)
		VALUES ($1, $2, $3, '[]'::jsonb, $4)
		RETURNING id::text`,
		"pp-ru-b", 50.0, "VendorB", "http://mercury/r.jpg",
	).Scan(&idB); err != nil {
		t.Fatalf("insert row B: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ListPendingPurchasesHandler(testPool, []string{"COGS"}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}

	var pending []PendingPurchase
	if err := json.NewDecoder(rec.Body).Decode(&pending); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Find rows by ID.
	byID := make(map[string]PendingPurchase)
	for _, p := range pending {
		byID[p.ID] = p
	}

	rowA, ok := byID[idA]
	if !ok {
		t.Fatalf("row A (%s) not found in response", idA)
	}
	if len(rowA.ReceiptURLs) != 2 {
		t.Errorf("row A ReceiptURLs length = %d, want 2; got %v", len(rowA.ReceiptURLs), rowA.ReceiptURLs)
	} else {
		if rowA.ReceiptURLs[0] != "http://spaces/0.jpg" {
			t.Errorf("ReceiptURLs[0] = %q, want %q", rowA.ReceiptURLs[0], "http://spaces/0.jpg")
		}
		if rowA.ReceiptURLs[1] != "http://spaces/1.pdf" {
			t.Errorf("ReceiptURLs[1] = %q, want %q", rowA.ReceiptURLs[1], "http://spaces/1.pdf")
		}
	}

	rowB, ok := byID[idB]
	if !ok {
		t.Fatalf("row B (%s) not found in response", idB)
	}
	// Legacy row with receipt_urls=NULL must not return the field at all.
	if len(rowB.ReceiptURLs) != 0 {
		t.Errorf("row B ReceiptURLs = %v, want nil/empty (legacy row must not expose receipt_urls)", rowB.ReceiptURLs)
	}
}

// ── The money path's boundary test ───────────────────────────────────────────
//
// RED BEFORE pendingPeriodDateExpr MOVED TO NEW YORK.
//
// pendingPeriodDateExpr is this card's headline site: the COGS window AND the
// completeness gate that feeds sales-processor's weekly payroll. Until this
// test it was covered by no red at all. The nearest existing case,
// "pending_review_details event_date falls back to ... cast of created_at",
// uses 2026-05-29 22:02:00+00 — which is 17:02 in Chicago and 18:02 in New
// York, i.e. the SAME calendar date in both. It cannot tell the two zones
// apart, so it could not have caught a wrong zone here.
//
// This one can. Both rows are placed in the one-hour gap where Chicago and New
// York disagree about the calendar date, and they are placed at OPPOSITE edges
// of the period so they move in OPPOSITE directions:
//
//	instant 2026-06-01T04:30Z  NY 2026-06-01 00:30 (OUT)  CHI 2026-05-31 23:30 (IN)
//	instant 2026-05-25T04:30Z  NY 2026-05-25 00:30 (IN)   CHI 2026-05-24 23:30 (OUT)
//
// A single-zone answer therefore cannot satisfy both assertions. Under the old
// Chicago cast the expected set was exactly inverted, which is what "the
// boundary moved across a repo line" means in rows.
//
// Both rows carry NULL event_date deliberately — an extracted event_date wins
// the COALESCE, so a row that has one is not exposed to the zone at all. This
// is the entire blast radius of the change, stated as a test.
func TestPeriodSummary_PendingPeriodBoundaryIsTheAppTimezone(t *testing.T) {
	if testPool == nil {
		t.Skip("DB_TEST_URL not reachable; skipping integration test")
	}

	const from = "2026-05-25"
	const to = "2026-05-31"

	resetFixtures(t)

	// Just past the END of the period in New York, still inside it in Chicago.
	// MUST NOT appear.
	_ = insertPendingPurchaseWithEventDate(t,
		"tx-after-period-ends-in-ny", "", "2026-06-01 04:30:00+00", "")

	// Just past the START of the period in New York, still before it in
	// Chicago. MUST appear.
	_ = insertPendingPurchaseWithEventDate(t,
		"tx-after-period-starts-in-ny", "", "2026-05-25 04:30:00+00", "")

	// A control in the uncontested middle of the period, to prove the window
	// itself works and the two edge rows are not both being dropped by some
	// unrelated filter.
	_ = insertPendingPurchaseWithEventDate(t,
		"tx-mid-period", "", "2026-05-28 16:00:00+00", "")

	code, got := callHandler(t, from, to)
	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200", code)
	}

	want := []string{"tx-after-period-starts-in-ny", "tx-mid-period"}
	if len(got.TrackedBankTxIDs) != len(want) {
		t.Fatalf("TrackedBankTxIDs = %v, want %v\n"+
			"  if this reads [tx-after-period-ends-in-ny tx-mid-period], the pending "+
			"period cast is still on America/Chicago", got.TrackedBankTxIDs, want)
	}
	for i, id := range want {
		if got.TrackedBankTxIDs[i] != id {
			t.Errorf("TrackedBankTxIDs[%d] = %q, want %q (full=%v)",
				i, got.TrackedBankTxIDs[i], id, got.TrackedBankTxIDs)
		}
	}

	// The completeness gate is the half that reaches payroll. Assert it
	// separately rather than trusting that it shares the tracked-ids filter —
	// "they share a const" is the claim under test, not a premise.
	if len(got.Completeness.PendingReviewIDs) != len(want) {
		t.Errorf("len(PendingReviewIDs) = %d, want %d — the completeness gate and "+
			"the tracked-ids UNION disagree about which day a receipt belongs to",
			len(got.Completeness.PendingReviewIDs), len(want))
	}
	if got.Completeness.Ready {
		t.Errorf("Completeness.Ready = true, want false (there are blocking pendings)")
	}
}
