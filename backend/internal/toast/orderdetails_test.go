package toast

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/db"
)

// stringReader keeps the inline-CSV tests readable.
func stringReader(s string) io.Reader { return strings.NewReader(s) }

// fixturePath is the committed OrderDetails.csv fixture. It carries the column
// SET and the value FORMATS the real export was proven to use on 2026-10-01
// (spike 02 over 77 real orders): "Order #" digits only at 1, 2 and 4 digits,
// money as plain decimals AND as "$1,205.73", "Opened"/"Closed" as
// "MM/DD/YY h:MM AM", a voided order present as a ROW rather than filtered, one
// order with a BLANK "Closed" and a blank "Order Source".
//
// The real sample itself is not in this repo (it lived at
// ~/projects/yumyums/OrderDetails.csv on the spike box and is not committed),
// so this fixture is what the test suite can hold. The two things it cannot
// prove are named in the card's merge-intent.
func fixturePath(t *testing.T) string {
	t.Helper()
	p := filepath.Join("testdata", "OrderDetails.csv")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture missing: %v", err)
	}
	return p
}

func openFixture(t *testing.T) *os.File {
	t.Helper()
	f, err := os.Open(fixturePath(t))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// TestOrderDetailsUpsertIsIdempotent is the card's first done_when row: THE SAME
// FILE TWICE LEAVES ONE ROW PER ORDER. This is not a hypothetical — the Toast
// worker re-pulls a 7-day window on every 12-hour tick, so every order in the
// export arrives at least fourteen times.
//
// It asserts three things, because "one row" alone would pass against an
// INSERT ... DO NOTHING that never refreshes a corrected report:
//  1. the row count after two loads equals the distinct (business_date,
//     order_number) count of the file;
//  2. the money survives the second load unchanged (no double-add);
//  3. ingested_at MOVED on the second load — the row was re-seen, not skipped.
func TestOrderDetailsUpsertIsIdempotent(t *testing.T) {
	pool := setupOrdersDB(t)
	ctx := context.Background()

	first, err := IngestOrderDetails(ctx, pool, openFixture(t))
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	if first != 4 {
		t.Fatalf("first ingest upserted %d rows, want 4 (the fixture's order count)", first)
	}

	var countFirst int64
	var ingestedFirst time.Time
	if err := pool.QueryRow(ctx,
		`SELECT count(*), max(ingested_at) FROM toast_orders`).Scan(&countFirst, &ingestedFirst); err != nil {
		t.Fatalf("count after first load: %v", err)
	}
	if countFirst != 4 {
		t.Fatalf("after one load: %d rows, want 4", countFirst)
	}

	var amountFirst, discountFirst int64
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(sum(amount_cents),0), coalesce(sum(discount_cents),0) FROM toast_orders`).
		Scan(&amountFirst, &discountFirst); err != nil {
		t.Fatalf("sums after first load: %v", err)
	}

	// Second arrival of the SAME report.
	second, err := IngestOrderDetails(ctx, pool, openFixture(t))
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if second != 4 {
		t.Fatalf("second ingest upserted %d rows, want 4", second)
	}

	var rows, distinct int64
	if err := pool.QueryRow(ctx,
		`SELECT count(*), count(DISTINCT (business_date, order_number)) FROM toast_orders`).
		Scan(&rows, &distinct); err != nil {
		t.Fatalf("count after second load: %v", err)
	}
	if rows != distinct {
		t.Fatalf("after the same file twice: %d rows vs %d distinct keys — the upsert is NOT idempotent", rows, distinct)
	}
	if rows != 4 {
		t.Fatalf("after the same file twice: %d rows, want 4 (one per order)", rows)
	}

	var amountSecond, discountSecond int64
	if err := pool.QueryRow(ctx,
		`SELECT coalesce(sum(amount_cents),0), coalesce(sum(discount_cents),0) FROM toast_orders`).
		Scan(&amountSecond, &discountSecond); err != nil {
		t.Fatalf("sums after second load: %v", err)
	}
	if amountSecond != amountFirst || discountSecond != discountFirst {
		t.Fatalf("money changed on the second load: amount %d -> %d, discount %d -> %d",
			amountFirst, amountSecond, discountFirst, discountSecond)
	}

	// A DO NOTHING would also leave one row per order, and would silently never
	// pick up a CORRECTED report. ingested_at advancing is what tells the two
	// apart: every row was re-seen and refreshed, not skipped.
	var staleRows int64
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM toast_orders WHERE ingested_at <= $1`, ingestedFirst).Scan(&staleRows); err != nil {
		t.Fatalf("ingested_at check: %v", err)
	}
	if staleRows != 0 {
		t.Fatalf("%d of 4 rows kept their first ingested_at — the second arrival was SKIPPED, not upserted; "+
			"a corrected report would never land", staleRows)
	}

	t.Logf("same file twice: rows=%d distinct_keys=%d amount_cents=%d discount_cents=%d",
		rows, distinct, amountSecond, discountSecond)
}

// TestParseOrderDetailsCoercions pins the parse decisions this card made: the
// money coercions, the order-number shapes, the voided COLUMN (not a filter),
// and the nullable Closed / Order Source.
func TestParseOrderDetailsCoercions(t *testing.T) {
	rows, err := parseOrderDetails(openFixture(t), "")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(rows) != 4 {
		t.Fatalf("parsed %d rows, want 4", len(rows))
	}

	byNum := map[string]OrderRow{}
	for _, r := range rows {
		byNum[r.OrderNumber] = r
	}
	for _, want := range []string{"9", "99", "9999", "100"} {
		if _, ok := byNum[want]; !ok {
			t.Errorf("order number %q missing — digits-only variable-length text is the key shape", want)
		}
	}

	// "$1,205.73" -> 120573 cents; "$1.41" -> 141 cents.
	if got := byNum["9999"].AmountCents; got != 120573 {
		t.Errorf(`"$1,205.73" parsed to %d cents, want 120573`, got)
	}
	if got := byNum["9999"].DiscountCents; got != 141 {
		t.Errorf(`"$1.41" parsed to %d cents, want 141`, got)
	}
	if got := byNum["9"].DiscountCents; got != 0 {
		t.Errorf(`"0.00" discount parsed to %d cents, want 0`, got)
	}
	if got := byNum["99"].AmountCents; got != 2400 {
		t.Errorf(`"24.00" parsed to %d cents, want 2400`, got)
	}

	// A VOIDED order is a row with voided=true, not an absence. Reconciliation
	// has to be able to say "that order was voided" about a scan.
	if !byNum["100"].Voided {
		t.Errorf("order 100 should be voided=true")
	}
	if byNum["9"].Voided {
		t.Errorf("order 9 should be voided=false")
	}

	// Blank Closed / Order Source stay NULL rather than becoming a zero time or
	// an empty string.
	if byNum["9999"].ClosedAt != nil {
		t.Errorf("blank Closed should be nil, got %v", byNum["9999"].ClosedAt)
	}
	if byNum["9"].ClosedAt == nil {
		t.Errorf("populated Closed should parse")
	}
	if byNum["100"].OrderSource != nil {
		t.Errorf("blank Order Source should be nil, got %q", *byNum["100"].OrderSource)
	}
	if byNum["99"].OrderSource == nil || *byNum["99"].OrderSource != "Online" {
		t.Errorf("Order Source should be %q", "Online")
	}

	// business_date is Opened's calendar date in the business timezone — the
	// same derivation the spike's proven upsert key used.
	for _, r := range rows {
		if got := r.BusinessDate.Format("2006-01-02"); got != "2026-09-28" {
			t.Errorf("order %s: business_date %s, want 2026-09-28", r.OrderNumber, got)
		}
		if got := r.OpenedAt.Location().String(); got != orderTimeZone {
			t.Errorf("order %s: opened_at location %s, want %s", r.OrderNumber, got, orderTimeZone)
		}
	}
}

// TestParseOrderDetailsRequiresItsColumns proves the FAIL-LOUD posture: a report
// that lost a money column must not land orders with amount_cents=0, because
// every money figure H3b computes would then be silently wrong. The optional
// columns are the explicit exception.
func TestParseOrderDetailsRequiresItsColumns(t *testing.T) {
	missing := "Location,Order Id,Order #,Opened,Amount,Total,Voided\n" +
		"Yum Yums,ord-1,7,09/28/26 11:42 AM,12.50,13.53,false\n"
	if _, err := parseOrderDetails(stringReader(missing), ""); err == nil {
		t.Fatalf("a report with no \"Discount Amount\" column must FAIL, not parse to zeros")
	}

	// Closed and Order Source absent entirely: parses, both fields NULL.
	optional := "Order Id,Order #,Opened,Discount Amount,Amount,Total,Voided\n" +
		"ord-1,7,09/28/26 11:42 AM,0.00,12.50,13.53,false\n"
	rows, err := parseOrderDetails(stringReader(optional), "")
	if err != nil {
		t.Fatalf("absent OPTIONAL columns must parse: %v", err)
	}
	if len(rows) != 1 || rows[0].ClosedAt != nil || rows[0].OrderSource != nil {
		t.Fatalf("absent optional columns should leave NULLs, got %+v", rows)
	}
}

// TestMigration0084DownAndUpRoundTrip — 0084's Down is claimed, so it is proven.
// A Down nobody has run is a Down that does not work. Runs with a zz-free name
// but is ordered last within this file's set by being the only migration test;
// it leaves the schema fully UP, so no later package under -p 1 sees a
// half-migrated database.
func TestMigration0084DownAndUpRoundTrip(t *testing.T) {
	pool := setupOrdersDB(t)
	ctx := context.Background()
	tables := []string{"toast_orders", "scan_attempts_mirror", "reconciliation_decisions"}

	exists := func(name string) bool {
		var ok bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			                WHERE table_schema = current_schema() AND table_name = $1)`,
			name).Scan(&ok); err != nil {
			t.Fatalf("exists(%s): %v", name, err)
		}
		return ok
	}
	indexExists := func(name string) bool {
		var ok bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM pg_indexes
			                WHERE schemaname = current_schema() AND indexname = $1)`,
			name).Scan(&ok); err != nil {
			t.Fatalf("indexExists(%s): %v", name, err)
		}
		return ok
	}

	for _, tb := range tables {
		if !exists(tb) {
			t.Fatalf("precondition: %s is missing before the down leg", tb)
		}
	}
	if !indexExists("race_lost_notifications_dedupe_uq") {
		t.Fatalf("precondition: B-424's unique index is missing before the down leg")
	}

	if err := db.MigrateTo(pool, 83); err != nil {
		t.Fatalf("migrate down to 83: %v", err)
	}
	for _, tb := range tables {
		if exists(tb) {
			t.Errorf("%s survived 0084's Down", tb)
		}
	}
	if indexExists("race_lost_notifications_dedupe_uq") {
		t.Errorf("B-424's unique index survived 0084's Down")
	}

	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate back up: %v", err)
	}
	for _, tb := range tables {
		if !exists(tb) {
			t.Errorf("%s did not come back on re-apply", tb)
		}
	}
	if !indexExists("race_lost_notifications_dedupe_uq") {
		t.Errorf("B-424's unique index did not come back on re-apply")
	}
	t.Log("0084 Down/Up round-trip clean; schema left fully up")
}
