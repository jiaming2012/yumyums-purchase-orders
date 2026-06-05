package inventory

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
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

// callHandler invokes PeriodSummaryHandler directly (not through chi).
func callHandler(t *testing.T, from, to string) (int, PeriodSummary) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/?from="+from+"&to="+to, nil)
	rec := httptest.NewRecorder()
	PeriodSummaryHandler(testPool).ServeHTTP(rec, req)
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
func insertEventAndLine(t *testing.T, vendorID, eventDate string, tax, total, price float64, qty int, purchaseItemID string) (string, string) {
	t.Helper()
	var eventID string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total)
		 VALUES ($1, $2, $3::date, $4, $5)
		 RETURNING id::text`,
		vendorID, "tx-"+eventDate+"-"+strconv.Itoa(int(price*10000)), eventDate, tax, total).Scan(&eventID)
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

// insertPendingPurchaseWithReason inserts an unconfirmed/undiscarded
// pending_purchases row with the given reason sentinel. Used by the
// completeness gate test to assert no_attachment_on_bank_tx rows block ready.
func insertPendingPurchaseWithReason(t *testing.T, createdAt, reason string) string {
	t.Helper()
	var id string
	q := `INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, reason, created_at)
	      VALUES ($1, 0, 'TestVendor', '[]'::jsonb, $2, $3::timestamptz)
	      RETURNING id::text`
	err := testPool.QueryRow(t.Context(), q,
		"pp-tx-noatt-"+createdAt, reason, createdAt).Scan(&id)
	if err != nil {
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
		resetFixtures(t)
		vendorID := insertVendor(t, "Acme")
		piID := insertPurchaseItem(t, "Salmon")
		insertEventAndLine(t, vendorID, "2026-05-26", 2.50, 25.00, 4.5, 5, piID)

		// No-attachment pending row created on 2026-05-28 (in range).
		ppID := insertPendingPurchaseWithReason(t,
			"2026-05-28 10:00:00-05:00", "no_attachment_on_bank_tx")

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
}
