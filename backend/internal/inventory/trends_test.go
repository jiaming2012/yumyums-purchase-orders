package inventory

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ── GET /api/v1/inventory/trends — AC-1/AC-6, design §2.2 AS AMENDED ─────────
//
// The parked card's fixture was rigged on every axis at once: it asserted the
// reconciliation identity against a hand-computed constant, so the constant
// moved with the bug. This fixture does the opposite — it calls
// PeriodSummaryHandler itself, on the window Trends reports, and compares the
// two responses. Nothing here is allowed to know the "right" total in advance.
//
// G6's five breakers are all present (see seedTrendsFixture):
//   B1 unitemized delivery fee   — event E_FEE
//   B2 non-COGS-category event   — event E_SOFTWARE ('Software')
//   B3 NULL-category event       — event E_NULLCAT
//   B4 eligible pending row      — pending PP_ELIGIBLE
//   B5 linked-but-groupless item — item "Sriracha" → D2 "Ungrouped"

// callTrends invokes TrendsHandler directly with the given allowlist.
func callTrends(t *testing.T, allowlist []string) (int, TrendsResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	TrendsHandler(testPool, allowlist).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		return rec.Code, TrendsResponse{}
	}
	var out TrendsResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode trends: %v (body=%s)", err, rec.Body.String())
	}
	return rec.Code, out
}

// insertGroup inserts an item_group and returns its UUID.
func insertGroup(t *testing.T, name string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO item_groups (name) VALUES ($1) RETURNING id::text`, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert item_group: %v", err)
	}
	return id
}

// insertPurchaseItemInGroup inserts a purchase_item bound to a group.
func insertPurchaseItemInGroup(t *testing.T, description, groupID string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_items (description, group_id) VALUES ($1, $2::uuid) RETURNING id::text`,
		description, groupID).Scan(&id)
	if err != nil {
		t.Fatalf("insert purchase_item in group: %v", err)
	}
	return id
}

// insertEvent inserts a bare purchase_event (no line items) and returns its ID.
// Empty category is stored as SQL NULL.
func insertEvent(t *testing.T, vendorID, bankTxID, eventDate string, tax, total float64, category string) string {
	t.Helper()
	var categoryArg interface{}
	if category != "" {
		categoryArg = category
	}
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO purchase_events (vendor_id, bank_tx_id, event_date, tax, total, mercury_category)
		 VALUES ($1, $2, $3::date, $4, $5, $6) RETURNING id::text`,
		vendorID, bankTxID, eventDate, tax, total, categoryArg).Scan(&id)
	if err != nil {
		t.Fatalf("insert purchase_event %s: %v", bankTxID, err)
	}
	return id
}

// insertLine attaches a line item to an event. Empty purchaseItemID → NULL
// (the unlinked case).
func insertLine(t *testing.T, eventID, purchaseItemID, description string, qty int, price float64) {
	t.Helper()
	var itemArg interface{}
	if purchaseItemID != "" {
		itemArg = purchaseItemID
	}
	_, err := testPool.Exec(t.Context(),
		`INSERT INTO purchase_line_items (purchase_event_id, purchase_item_id, description, quantity, price, is_case)
		 VALUES ($1, $2, $3, $4, $5, false)`,
		eventID, itemArg, description, qty, price)
	if err != nil {
		t.Fatalf("insert purchase_line_item %s: %v", description, err)
	}
}

// insertEligiblePending inserts a pending_purchases row matching the
// period-summary eligible population exactly (handler.go:1345-1351):
// confirmed_at IS NULL · discarded_at IS NULL · mercury_category = ANY($3) ·
// reason != 'no_attachment_on_bank_tx'.
func insertEligiblePending(t *testing.T, bankTxID, eventDate string, bankTotal float64, category, reason string) {
	t.Helper()
	var categoryArg interface{}
	if category != "" {
		categoryArg = category
	}
	_, err := testPool.Exec(t.Context(),
		`INSERT INTO pending_purchases (bank_tx_id, bank_total, vendor, items, event_date, reason, mercury_category)
		 VALUES ($1, $2, 'PendingVendor', '[]'::jsonb, $3::date, $4, $5)`,
		bankTxID, bankTotal, eventDate, reason, categoryArg)
	if err != nil {
		t.Fatalf("insert pending_purchase %s: %v", bankTxID, err)
	}
}

// weekOf returns the YYYY-MM-DD of win.From shifted forward by n weeks plus
// dayOffset days — used to place fixture rows in known ISO week buckets.
//
// The final week of the window is PARTIAL (it ends on `to` = today, which may
// be any weekday), so a naive offset can land past `to` and silently drop rows
// from the fixture. Dates are clamped to win.To. Negative n is deliberately
// NOT clamped — that is the out-of-window seed.
func weekOf(t *testing.T, win TrendsWindow, n, dayOffset int) string {
	t.Helper()
	d, err := time.Parse("2006-01-02", win.From)
	if err != nil {
		t.Fatalf("parse from %q: %v", win.From, err)
	}
	out := d.AddDate(0, 0, 7*n+dayOffset).Format("2006-01-02")
	if n >= 0 && out > win.To {
		return win.To
	}
	return out
}

// trendsFixture records what the fixture created, so assertions can name
// groups without re-deriving totals.
type trendsFixture struct {
	proteinsID string
	produceID  string
}

// seedTrendsFixture builds an honest 12-week fixture containing all five G6
// breakers plus two real groups spanning ≥8 distinct weeks. It deliberately
// does NOT return any expected total — the identity is asserted against
// period-summary, never against a constant authored here.
func seedTrendsFixture(t *testing.T, win TrendsWindow) trendsFixture {
	t.Helper()
	vendor := insertVendor(t, "Restaurant Depot")

	proteins := insertGroup(t, "Proteins")
	produce := insertGroup(t, "Produce")

	salmon := insertPurchaseItemInGroup(t, "Salmon Fillet", proteins)
	chicken := insertPurchaseItemInGroup(t, "Chicken Thigh", proteins)
	kale := insertPurchaseItemInGroup(t, "Kale", produce)
	// B5 — linked-but-groupless: a real purchase_item with group_id IS NULL.
	// Must land in the D2 "Ungrouped" pseudo-group, NOT in `unlinked`.
	sriracha := insertPurchaseItem(t, "Sriracha")

	// Ordinary confirmed COGS spend across 8 distinct week buckets.
	type seed struct {
		week  int
		item  string
		qty   int
		price float64
	}
	seeds := []seed{
		{0, salmon, 2, 41.25},
		{0, kale, 3, 6.10},
		{1, chicken, 4, 12.99},
		{3, salmon, 1, 39.95},
		{3, sriracha, 2, 7.45}, // B5 → Ungrouped
		{5, kale, 5, 5.85},
		{6, chicken, 2, 13.40},
		{8, salmon, 3, 42.00},
		{9, kale, 1, 6.75},
		{11, chicken, 6, 11.85},
		{11, sriracha, 1, 8.20}, // B5 again, different week
	}
	for i, s := range seeds {
		date := weekOf(t, win, s.week, 2) // Wednesday of that week
		ev := insertEvent(t, vendor, "tx-cogs-"+date+"-"+itoa(i), date, 0, float64(s.qty)*s.price, "COGS")
		insertLine(t, ev, s.item, "line", s.qty, s.price)
	}

	// Unlinked spend (purchase_item_id IS NULL) — excluded from group buckets,
	// reported per-week in `unlinked`. Two distinct weeks.
	for i, wk := range []int{2, 7} {
		date := weekOf(t, win, wk, 3)
		ev := insertEvent(t, vendor, "tx-unlinked-"+date+"-"+itoa(i), date, 0, 31.55, "COGS")
		insertLine(t, ev, "", "Mystery Case", 1, 31.55)
	}

	// B1 — THE MINIMAL BREAKER, and the normal case: a receipt with an
	// unitemized delivery fee. total(120.00) - tax(0) exceeds the itemized
	// lines (2 × 52.50 = 105.00) by a 15.00 remainder. Amendment 3: that
	// remainder is surfaced, NOT smeared across the lines by proration, and
	// is NOT an addend to the identity.
	feeDate := weekOf(t, win, 4, 1)
	feeEv := insertEvent(t, vendor, "tx-delivery-fee", feeDate, 0, 120.00, "COGS")
	insertLine(t, feeEv, salmon, "Salmon Fillet", 2, 52.50)

	// A second B1 with tax present, so the remainder formula
	// (total - tax) - Σlines is exercised with a non-zero tax term.
	feeDate2 := weekOf(t, win, 10, 1)
	feeEv2 := insertEvent(t, vendor, "tx-delivery-fee-2", feeDate2, 4.40, 78.40, "COGS")
	insertLine(t, feeEv2, kale, "Kale", 4, 15.00)

	// B2 — non-COGS category. Amendment 1: excluded from Trends entirely
	// (this is the +500.00 over-report G6 measured).
	swDate := weekOf(t, win, 6, 1)
	swEv := insertEvent(t, vendor, "tx-software", swDate, 0, 500.00, "Software")
	insertLine(t, swEv, salmon, "Adobe CC", 1, 500.00)

	// B3 — NULL mercury_category. `= ANY($3)` does not match SQL NULL, so this
	// is excluded from both Trends and period-summary. Intended (Amendment 1).
	nullDate := weekOf(t, win, 7, 1)
	nullEv := insertEvent(t, vendor, "tx-nullcat", nullDate, 0, 250.00, "")
	insertLine(t, nullEv, chicken, "Uncategorized Charge", 1, 250.00)

	// B4 — eligible pending row (Amendment 2): unreviewed, so it has no linked
	// line items and cannot be bucketed; it is a completeness figure AND an
	// addend to the identity, because period-summary counts it.
	insertEligiblePending(t, "pp-eligible", weekOf(t, win, 9, 1), 240.00, "COGS", "parse_failed")
	// Ineligible pending rows — each must be excluded by exactly one clause.
	insertEligiblePending(t, "pp-blocking", weekOf(t, win, 9, 2), 90.00, "COGS", "no_attachment_on_bank_tx")
	insertEligiblePending(t, "pp-noncogs", weekOf(t, win, 9, 3), 70.00, "Software", "parse_failed")
	insertEligiblePending(t, "pp-nullcat", weekOf(t, win, 9, 4), 60.00, "", "parse_failed")

	// Out-of-window spend — one week BEFORE `from`. Must not appear anywhere.
	outDate := weekOf(t, win, -1, 2)
	outEv := insertEvent(t, vendor, "tx-out-of-window", outDate, 0, 999.99, "COGS")
	insertLine(t, outEv, salmon, "Ancient Salmon", 1, 999.99)

	return trendsFixture{proteinsID: proteins, produceID: produce}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// cents rounds a float to whole cents so two independently-summed money paths
// compare without float64 representation noise.
func cents(f float64) int64 { return int64(math.Round(f * 100)) }

func TestTrends(t *testing.T) {
	if testPool == nil {
		t.Skip("no DB_TEST_URL / Postgres unreachable")
	}
	resetFixtures(t)

	// Ask the endpoint for its window FIRST, then seed into it. The fixture
	// never chooses the window; the handler does.
	_, probe := callTrends(t, []string{"COGS"})
	win := probe.Window

	// The window contract itself (Assumption A1).
	expected := trendsWindow(time.Now())
	if win.From != expected.From || win.To != expected.To || win.Weeks != TrendsWeeks {
		t.Fatalf("window: got %+v, want %+v", win, expected)
	}

	fx := seedTrendsFixture(t, win)

	code, resp := callTrends(t, []string{"COGS"})
	if code != http.StatusOK {
		t.Fatalf("trends status: got %d, want 200", code)
	}

	// ── THE IDENTITY ────────────────────────────────────────────────────────
	// Σcells + Σunlinked + pending_total == period_summary.cogs_excl_tax
	//
	// The right-hand side is obtained by CALLING period-summary on the exact
	// window Trends just reported. No constant appears on either side.
	psCode, ps := callHandler(t, resp.Window.From, resp.Window.To)
	if psCode != http.StatusOK {
		t.Fatalf("period-summary status: got %d, want 200", psCode)
	}

	var cellSum, unlinkedSum float64
	for _, c := range resp.Cells {
		cellSum += c.Spend
	}
	for _, u := range resp.Unlinked {
		unlinkedSum += u.Spend
	}
	lhs := cellSum + unlinkedSum + resp.Completeness.PendingTotal

	if cents(lhs) != cents(ps.COGSExclTax) {
		t.Errorf("RECONCILIATION IDENTITY FAILED on window %s..%s:\n"+
			"  Σcells            = %.2f\n"+
			"  Σunlinked         = %.2f\n"+
			"  pending_total     = %.2f\n"+
			"  ------------------------------\n"+
			"  LHS               = %.2f\n"+
			"  period_summary.cogs_excl_tax (RHS) = %.2f\n"+
			"  delta             = %.2f",
			resp.Window.From, resp.Window.To,
			cellSum, unlinkedSum, resp.Completeness.PendingTotal,
			lhs, ps.COGSExclTax, lhs-ps.COGSExclTax)
	}

	// The endpoint must publish its own left-hand side, and it must agree.
	if cents(resp.Completeness.ReconcilesToCogsExclTax) != cents(lhs) {
		t.Errorf("reconciles_to_cogs_excl_tax: got %.2f, want %.2f (Σcells+Σunlinked+pending)",
			resp.Completeness.ReconcilesToCogsExclTax, lhs)
	}

	// ── Amendment 2 — pending is a completeness figure, never a cell ────────
	// Exactly one of the four seeded pending rows is eligible, at 240.00.
	if resp.Completeness.PendingCount != 1 {
		t.Errorf("pending_count: got %d, want 1 (only pp-eligible qualifies)", resp.Completeness.PendingCount)
	}
	if cents(resp.Completeness.PendingTotal) != cents(240.00) {
		t.Errorf("pending_total: got %.2f, want 240.00", resp.Completeness.PendingTotal)
	}
	// And it must agree with period-summary's own eligible population.
	psLines := ps.COGSExclTax - resp.Completeness.PendingTotal
	if cents(psLines) != cents(cellSum+unlinkedSum) {
		t.Errorf("pending population disagrees with period-summary: "+
			"cogs_excl_tax(%.2f) - pending(%.2f) = %.2f, but Σcells+Σunlinked = %.2f",
			ps.COGSExclTax, resp.Completeness.PendingTotal, psLines, cellSum+unlinkedSum)
	}
	for _, g := range resp.Groups {
		if g.Name == "Unreviewed" || g.ID == "pending" {
			t.Errorf("Amendment 2 violated: pending surfaced as a pseudo-group %+v", g)
		}
	}

	// ── Amendment 3 — no proration; unitemized remainder is surfaced ────────
	// B1: (120.00 - 0) - 105.00 = 15.00 ; B1b: (78.40 - 4.40) - 60.00 = 14.00
	if cents(resp.Completeness.UnitemizedRemainder) != cents(29.00) {
		t.Errorf("unitemized_remainder: got %.2f, want 29.00 (15.00 + 14.00)",
			resp.Completeness.UnitemizedRemainder)
	}
	// The B1 receipt's salmon line must be at FACE VALUE (105.00), not
	// inflated by a 120/120 proration factor. Find its week's Proteins cell.
	feeWeek := weekOf(t, win, 4, 0)
	var feeCell float64
	for _, c := range resp.Cells {
		if c.WeekStart == feeWeek && c.GroupID == fx.proteinsID {
			feeCell = c.Spend
		}
	}
	if cents(feeCell) != cents(105.00) {
		t.Errorf("B1 unitemized-fee week: Proteins cell = %.2f, want 105.00 face value "+
			"(proration would inflate it toward 120.00)", feeCell)
	}

	// ── Amendment 1 — COGS allowlist filter ─────────────────────────────────
	// B2 (Software, 500.00) and B3 (NULL category, 250.00) must be absent.
	swWeek := weekOf(t, win, 6, 0)
	nullWeek := weekOf(t, win, 7, 0)
	for _, c := range resp.Cells {
		if c.WeekStart == swWeek && cents(c.Spend) >= cents(500.00) {
			t.Errorf("Amendment 1 violated: non-COGS 500.00 leaked into cell %+v", c)
		}
		if c.WeekStart == nullWeek && cents(c.Spend) >= cents(250.00) {
			t.Errorf("Amendment 1 violated: NULL-category 250.00 leaked into cell %+v", c)
		}
	}

	// ── D2 — linked-but-groupless items get an Ungrouped bucket ─────────────
	var haveUngrouped bool
	for _, g := range resp.Groups {
		if g.ID == UngroupedGroupID {
			haveUngrouped = true
			if g.Name != UngroupedGroupName {
				t.Errorf("Ungrouped pseudo-group name: got %q, want %q", g.Name, UngroupedGroupName)
			}
		}
	}
	if !haveUngrouped {
		t.Errorf("D2 violated: no %q pseudo-group in groups %+v", UngroupedGroupID, resp.Groups)
	}
	var ungroupedSum float64
	for _, c := range resp.Cells {
		if c.GroupID == UngroupedGroupID {
			ungroupedSum += c.Spend
		}
	}
	// Sriracha: 2×7.45 + 1×8.20 = 23.10
	if cents(ungroupedSum) != cents(23.10) {
		t.Errorf("Ungrouped spend: got %.2f, want 23.10 (linked-but-groupless Sriracha)", ungroupedSum)
	}
	// D2 explicitly: groupless money must NOT be folded into `unlinked`.
	// Unlinked is purchase_item_id IS NULL only: 2 × 31.55 = 63.10
	if cents(resp.UnlinkedTotal) != cents(63.10) {
		t.Errorf("unlinked_total: got %.2f, want 63.10 (purchase_item_id IS NULL only — "+
			"groupless items belong in Ungrouped)", resp.UnlinkedTotal)
	}
	if cents(unlinkedSum) != cents(resp.UnlinkedTotal) {
		t.Errorf("Σunlinked(%.2f) != unlinked_total(%.2f)", unlinkedSum, resp.UnlinkedTotal)
	}
	if len(resp.Unlinked) != 2 {
		t.Errorf("unlinked weeks: got %d, want 2", len(resp.Unlinked))
	}

	// ── Real groups are present and named ───────────────────────────────────
	seen := map[string]string{}
	for _, g := range resp.Groups {
		seen[g.ID] = g.Name
	}
	if seen[fx.proteinsID] != "Proteins" {
		t.Errorf("Proteins group missing/misnamed: %+v", resp.Groups)
	}
	if seen[fx.produceID] != "Produce" {
		t.Errorf("Produce group missing/misnamed: %+v", resp.Groups)
	}

	// ── Windowing + sparseness ──────────────────────────────────────────────
	weeks := map[string]bool{}
	for _, c := range resp.Cells {
		if c.WeekStart < win.From || c.WeekStart > win.To {
			t.Errorf("cell outside window: %+v (window %s..%s)", c, win.From, win.To)
		}
		if cents(c.Spend) == 0 {
			t.Errorf("cells must be sparse — zero cell emitted: %+v", c)
		}
		if cents(c.Spend) == cents(999.99) {
			t.Errorf("out-of-window event leaked into cells: %+v", c)
		}
		weeks[c.WeekStart] = true
	}
	if len(weeks) < 8 {
		t.Errorf("fixture must span ≥8 week buckets, cells cover %d", len(weeks))
	}

	// ── AC-1 regression: period-summary contract unchanged ──────────────────
	// The Trends work must not perturb the service-token contract. Assert the
	// existing endpoint still answers and its own internal invariant holds.
	// The only tax in the fixture is B1b's 4.40, so cogs_incl_tax must exceed
	// cogs_excl_tax by exactly that — and Trends must not have moved it.
	if cents(ps.COGSInclTax-ps.COGSExclTax) != cents(4.40) {
		t.Errorf("period-summary regression: cogs_incl_tax(%.2f) - cogs_excl_tax(%.2f) = %.2f, want 4.40",
			ps.COGSInclTax, ps.COGSExclTax, ps.COGSInclTax-ps.COGSExclTax)
	}
}

// TestTrendsAllowlistIsNotHardcoded proves Trends reads the SAME allowlist
// slice it is constructed with (Amendment 1's "do not re-derive or hardcode"),
// by widening it and watching the previously-excluded Software event appear.
func TestTrendsAllowlistIsNotHardcoded(t *testing.T) {
	if testPool == nil {
		t.Skip("no DB_TEST_URL / Postgres unreachable")
	}
	resetFixtures(t)

	_, probe := callTrends(t, []string{"COGS"})
	win := probe.Window
	seedTrendsFixture(t, win)

	_, narrow := callTrends(t, []string{"COGS"})
	_, wide := callTrends(t, []string{"COGS", "Software"})

	var narrowSum, wideSum float64
	for _, c := range narrow.Cells {
		narrowSum += c.Spend
	}
	for _, c := range wide.Cells {
		wideSum += c.Spend
	}
	if cents(wideSum-narrowSum) != cents(500.00) {
		t.Errorf("widening the allowlist to include Software should add exactly 500.00: "+
			"narrow=%.2f wide=%.2f delta=%.2f", narrowSum, wideSum, wideSum-narrowSum)
	}

	// And the identity must still hold on the widened allowlist — against
	// period-summary called with the SAME widened allowlist.
	_, ps := callHandlerWithAllowlist(t, wide.Window.From, wide.Window.To, []string{"COGS", "Software"})
	var wideUnlinked float64
	for _, u := range wide.Unlinked {
		wideUnlinked += u.Spend
	}
	lhs := wideSum + wideUnlinked + wide.Completeness.PendingTotal
	if cents(lhs) != cents(ps.COGSExclTax) {
		t.Errorf("identity failed on widened allowlist: LHS=%.2f RHS=%.2f delta=%.2f",
			lhs, ps.COGSExclTax, lhs-ps.COGSExclTax)
	}
}
