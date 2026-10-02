package marketing

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── card H3b · the stats engine ──

// mountedMuxWithReports mounts BOTH halves of decision 192 on one router:
// /api/v1/marketing/* (marketing grant + manager tier, Mount's table) and
// /api/v1/bi/campaigns/* (bi grant, NO manager tier, the MountReports seam).
// The permission middleware itself is main.go's; what these tests exercise is
// the IN-HANDLER tier, which is the half that differs between the two.
func mountedMuxWithReports(d Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/marketing", func(r chi.Router) { Mount(r, d) })
	r.Route("/api/v1", func(r chi.Router) { MountReportsDeps(r, d) })
	MountPublic(r, d)
	return r
}

// slicesFixture is spike 01 `slices-reconcile`'s fixture, seeded into Postgres.
//
// 🛑 The arithmetic is NOT re-derived here. The expected totals below are the
// spike's own run-2 output (scans=12 signups=5 redeemed=7 revenue=4350
// discount=5450 net=-1100), which the operator signed on 2026-10-01 after the
// per-row discount correction. If this test and the spike disagree, the engine
// is wrong — not the numbers.
type slicesFixture struct {
	manager                    string
	welcome, wingwed, catering string
	wings6, tray               string
	// expected totals (spike run 2)
	wantScans, wantSignups, wantRedeemed int
	wantRevenue, wantDiscount, wantNet   int
	wantImplied, wantActual              int
	subscribersPresent                   bool
}

func seedSlicesFixture(t *testing.T, pool *pgxpool.Pool) slicesFixture {
	t.Helper()
	f := slicesFixture{
		wantScans: 12, wantSignups: 5, wantRedeemed: 7,
		wantRevenue: 4350, wantDiscount: 5450, wantNet: -1100,
		wantImplied: 5500, wantActual: 1050,
	}
	f.manager = seedUser(t, pool, "manager")
	f.wings6 = seedMenuItem(t, pool, "6pc Wings "+randSuffix(t))
	f.tray = seedMenuItem(t, pool, "Catering Tray "+randSuffix(t))

	f.welcome = seedCampaignRow(t, pool, f.manager, "welcome", "Welcome", 300, nil)
	f.wingwed = seedCampaignRow(t, pool, f.manager, "wingwed", "Wing Wednesday", 200, &f.wings6)
	f.catering = seedCampaignRow(t, pool, f.manager, "catering", "Catering", 4000, &f.tray)

	// codes: short ← the spike's A1/A2/B1/B2/C1, spelled in the 0083 alphabet.
	a1 := seedCodeRow(t, pool, f.manager, f.welcome, "AAAAA2", "truck_sign", nil)
	a2 := seedCodeRow(t, pool, f.manager, f.welcome, "AAAAA3", "flyer", nil)
	b1 := seedCodeRow(t, pool, f.manager, f.wingwed, "BBBBB2", "truck_sign", &f.wings6)
	_ = seedCodeRow(t, pool, f.manager, f.wingwed, "BBBBB3", "instagram", &f.wings6)
	c1 := seedCodeRow(t, pool, f.manager, f.catering, "CCCCC2", "google_ads", &f.tray)

	// scans: A1×3 A2×2 B1×4 B2×1 C1×2 = 12
	seedScanRows(t, pool, "AAAAA2", 3)
	seedScanRows(t, pool, "AAAAA3", 2)
	seedScanRows(t, pool, "BBBBB2", 4)
	seedScanRows(t, pool, "BBBBB3", 1)
	seedScanRows(t, pool, "CCCCC2", 2)

	// signups: first-touch source_short A1,A2,B1,B1,C1 plus one with none = 5
	// resolvable. Only seedable once card H5's migration 0085 has landed.
	f.subscribersPresent = subscribersTableExists(t, pool)
	if f.subscribersPresent {
		for _, short := range []any{"AAAAA2", "AAAAA3", "BBBBB2", "BBBBB2", "CCCCC2", nil} {
			if _, err := pool.Exec(t.Context(), `
				INSERT INTO subscribers (source, source_short, joined_at, external_ref)
				VALUES ('qr', $1, now() - interval '3 days', $2)`,
				short, randSuffix(t)); err != nil {
				t.Fatalf("seed subscriber: %v", err)
			}
		}
	} else {
		t.Logf("subscribers table ABSENT (card H5 / migration 0085 has not landed in this tree): " +
			"signups is asserted as 0 with basis=unavailable, per the card's own rule")
		f.wantSignups = 0
	}

	scannedAt := time.Now().Add(-6 * time.Hour)
	// r1 welcome / A1  — matched 1450 / 300
	r1 := seedAttempt(t, pool, attemptFixture{CodeID: &a1, CampaignID: &f.welcome, ScannedAt: scannedAt, OrderNumber: strptr("101")})
	seedToastOrder(t, pool, scannedAt, "101", 1450, 300, scannedAt)
	// r2 welcome / A2  — matched 900 / 250 (actual ≠ implied)
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &a2, CampaignID: &f.welcome, ScannedAt: scannedAt, OrderNumber: strptr("102")})
	seedToastOrder(t, pool, scannedAt, "102", 900, 250, scannedAt)
	// r3 wingwed / B1  — matched 1200 / 200
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &b1, CampaignID: &f.wingwed, ScannedAt: scannedAt, OrderNumber: strptr("103")})
	seedToastOrder(t, pool, scannedAt, "103", 1200, 200, scannedAt)
	// r4 wingwed / B1  — no order, declined customer_left → counts in the orphan rate
	r4 := seedAttempt(t, pool, attemptFixture{CodeID: &b1, CampaignID: &f.wingwed, ScannedAt: scannedAt})
	seedDecision(t, pool, r4, "declined", strptr("customer_left"), nil, nil, f.manager)
	// r5 catering / C1 — no order, no decision → open orphan
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &c1, CampaignID: &f.catering, ScannedAt: scannedAt})
	// r6 wingwed / B1  — declined duplicate_scan → EXCLUDED from the orphan rate
	r6 := seedAttempt(t, pool, attemptFixture{CodeID: &b1, CampaignID: &f.wingwed, ScannedAt: scannedAt})
	seedDecision(t, pool, r6, "declined", strptr("duplicate_scan"), nil, nil, f.manager)
	// r7 welcome / NO first-touch code — matched 800 / 300; channel+item slices
	// bucket it as "direct", the campaign slice as welcome.
	_ = seedAttempt(t, pool, attemptFixture{CodeID: nil, CampaignID: &f.welcome, ScannedAt: scannedAt, OrderNumber: strptr("107")})
	seedToastOrder(t, pool, scannedAt, "107", 800, 300, scannedAt)
	_ = r1
	return f
}

// 🛑 THE CARD'S KEYSTONE. Σ over the three slices == the overview for every
// metric, under decision 190's PER-ROW discount rule.
func TestSlicesReconcileToOverview(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))
	ctx := userCtx(f.manager, "manager")

	rec := do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /stats/overview = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var ov StatsOverviewResponse
	decode(t, rec, &ov)

	t.Logf("overview: scans=%d signups=%d redeemed=%d revenue=%d discount=%d implied=%d actual=%d net=%d basis=%s",
		ov.Funnel.Scans, ov.Funnel.Signups, ov.Funnel.Redeemed,
		ov.Money.RevenueCents, ov.Money.DiscountCents, ov.Money.DiscountImpliedCents,
		ov.Money.DiscountActualCents, ov.Money.NetCents, ov.Money.DiscountBasis)

	for _, c := range []struct {
		name string
		got  int
		want int
	}{
		{"scans", ov.Funnel.Scans, f.wantScans},
		{"signups", ov.Funnel.Signups, f.wantSignups},
		{"redeemed", ov.Funnel.Redeemed, f.wantRedeemed},
		{"revenue_cents", ov.Money.RevenueCents, f.wantRevenue},
		{"discount_cents", ov.Money.DiscountCents, f.wantDiscount},
		{"discount_implied_cents", ov.Money.DiscountImpliedCents, f.wantImplied},
		{"discount_actual_cents", ov.Money.DiscountActualCents, f.wantActual},
		{"net_cents", ov.Money.NetCents, f.wantNet},
	} {
		if c.got != c.want {
			t.Errorf("overview.%s = %d, want %d (spike 01 run 2)", c.name, c.got, c.want)
		}
	}
	// 4 of 7 rows matched → the label is "mixed"; it is a LABEL, never a choice
	// of arithmetic (decision 190 as refined).
	// Every accepted row in the spike's fixture resolves a campaign, so no row's
	// face value is unknown. A non-zero count here would mean the per-row
	// discount rule silently contributed 0 for a row it could not price.
	if ov.Money.DiscountUnknownRows != 0 {
		t.Errorf("overview.discount_unknown_rows = %d, want 0 (every fixture row resolves a campaign)", ov.Money.DiscountUnknownRows)
	}
	if ov.Money.DiscountBasis != "mixed" {
		t.Errorf("overview.discount_basis = %q, want mixed (4 of 7 accepted rows matched)", ov.Money.DiscountBasis)
	}

	for _, dim := range []string{"campaign", "channel", "item"} {
		rec := do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/by?dim="+dim+"&period=30d", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /stats/by?dim=%s = %d, want 200\nbody: %s", dim, rec.Code, rec.Body.String())
		}
		var by StatsByResponse
		decode(t, rec, &by)
		if len(by.Rows) == 0 {
			t.Fatalf("dim=%s returned no rows\nbody: %s", dim, rec.Body.String())
		}
		var scans, signups, redeemed, revenue, discount, implied, actual, net, unknown int
		for _, row := range by.Rows {
			unknown += row.DiscountUnknownRows
			scans += row.Scans
			signups += row.Signups
			redeemed += row.Redeemed
			revenue += row.RevenueCents
			discount += row.DiscountCents
			implied += row.DiscountImpliedCents
			actual += row.DiscountActualCents
			net += row.NetCents
		}
		t.Logf("Σ %s: scans=%d signups=%d redeemed=%d revenue=%d discount=%d implied=%d actual=%d net=%d (rows=%d)",
			dim, scans, signups, redeemed, revenue, discount, implied, actual, net, len(by.Rows))
		for _, c := range []struct {
			name      string
			got, want int
		}{
			{"scans", scans, ov.Funnel.Scans},
			{"signups", signups, ov.Funnel.Signups},
			{"redeemed", redeemed, ov.Funnel.Redeemed},
			{"revenue_cents", revenue, ov.Money.RevenueCents},
			{"discount_cents", discount, ov.Money.DiscountCents},
			{"discount_implied_cents", implied, ov.Money.DiscountImpliedCents},
			{"discount_actual_cents", actual, ov.Money.DiscountActualCents},
			{"discount_unknown_rows", unknown, ov.Money.DiscountUnknownRows},
			{"net_cents", net, ov.Money.NetCents},
		} {
			if c.got != c.want {
				t.Errorf("Σ(dim=%s).%s = %d, overview = %d — the slices must reconcile", dim, c.name, c.got, c.want)
			}
		}
		// `totals` must be the same arithmetic, not a second one.
		if by.Totals.RevenueCents != ov.Money.RevenueCents || by.Totals.DiscountCents != ov.Money.DiscountCents ||
			by.Totals.Redeemed != ov.Funnel.Redeemed || by.Totals.Scans != ov.Funnel.Scans ||
			by.Totals.Signups != ov.Funnel.Signups || by.Totals.NetCents != ov.Money.NetCents {
			t.Errorf("dim=%s totals disagree with the overview: %+v", dim, by.Totals)
		}
	}
}

// decision 190: declines count in the orphan rate EXCEPT reason duplicate_scan.
// The spike's fixture gives 2 of 7 = 0.2857.
func TestOrphanRateCountsDeclinesExceptDuplicateScan(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))
	ctx := userCtx(f.manager, "manager")

	rec := do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	var ov StatsOverviewResponse
	decode(t, rec, &ov)

	if ov.Reconciliation.OrphanRate == nil {
		t.Fatalf("orphan_rate is null with 7 accepted attempts\nbody: %s", rec.Body.String())
	}
	// 3 accepted rows have no matched Toast order (r4 declined customer_left,
	// r5 open, r6 declined duplicate_scan); duplicate_scan is excluded → 2/7.
	if got, want := *ov.Reconciliation.OrphanRate, 0.2857; got != want {
		t.Errorf("orphan_rate = %v, want %v (2 of 7: r4 declined customer_left + r5 open; r6 duplicate_scan excluded)", got, want)
	}
	if ov.Reconciliation.Threshold != 0.10 {
		t.Errorf("threshold = %v, want 0.10", ov.Reconciliation.Threshold)
	}
	if ov.Reconciliation.Matched != 4 {
		t.Errorf("reconciliation.matched = %d, want 4", ov.Reconciliation.Matched)
	}
	if ov.Reconciliation.Declined != 2 {
		t.Errorf("reconciliation.declined = %d, want 2 (customer_left + duplicate_scan)", ov.Reconciliation.Declined)
	}

	// Flipping the counted decline to duplicate_scan drops the rate to 1/7.
	if _, err := pool.Exec(t.Context(),
		`UPDATE reconciliation_decisions SET reason = 'duplicate_scan' WHERE reason = 'customer_left'`); err != nil {
		t.Fatalf("flip reason: %v", err)
	}
	rec = do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	decode(t, rec, &ov)
	if got, want := *ov.Reconciliation.OrphanRate, 0.1429; got != want {
		t.Errorf("after flipping customer_left → duplicate_scan, orphan_rate = %v, want %v (1 of 7)", got, want)
	}

	// An empty period has NO opinion on the rate, and says so with null rather
	// than a confident 0%.
	if _, err := pool.Exec(t.Context(), `TRUNCATE reconciliation_decisions, scan_attempts_mirror CASCADE`); err != nil {
		t.Fatalf("truncate attempts: %v", err)
	}
	rec = do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	decode(t, rec, &ov)
	if ov.Reconciliation.OrphanRate != nil {
		t.Errorf("orphan_rate = %v with zero accepted attempts, want null", *ov.Reconciliation.OrphanRate)
	}
}

// Decision 192's double registration: ONE pair of handlers, two mount points,
// BYTE-IDENTICAL bodies — and the manager tier on /marketing/stats/* only.
func TestStatsReportsAreRegisteredTwiceWithByteIdenticalBodies(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))
	manager := userCtx(f.manager, "manager")

	for _, pair := range []struct{ marketing, bi string }{
		{"/api/v1/marketing/stats/overview?period=30d", "/api/v1/bi/campaigns/overview?period=30d"},
		{"/api/v1/marketing/stats/by?dim=campaign&period=30d", "/api/v1/bi/campaigns/by?dim=campaign&period=30d"},
		{"/api/v1/marketing/stats/by?dim=channel&period=30d", "/api/v1/bi/campaigns/by?dim=channel&period=30d"},
		{"/api/v1/marketing/stats/by?dim=item&period=30d", "/api/v1/bi/campaigns/by?dim=item&period=30d"},
	} {
		a := do(t, mux, manager, http.MethodGet, pair.marketing, nil)
		b := do(t, mux, manager, http.MethodGet, pair.bi, nil)
		if a.Code != http.StatusOK || b.Code != http.StatusOK {
			t.Fatalf("%s = %d / %s = %d, want 200/200\nmarketing: %s\nbi: %s",
				pair.marketing, a.Code, pair.bi, b.Code, a.Body.String(), b.Body.String())
		}
		if a.Body.String() != b.Body.String() {
			t.Errorf("bodies differ (decision 192 says byte-identical)\n%s:\n%s\n%s:\n%s",
				pair.marketing, a.Body.String(), pair.bi, b.Body.String())
		}
	}

	// A team_member is refused on the MARKETING pair (manager tier) and served
	// on the BI pair (the `bi` grant alone — decision 192, the consequence the
	// operator accepted).
	member := seedUser(t, pool, "team_member")
	memberCtx := userCtx(member, "team_member")
	for _, target := range []string{
		"/api/v1/marketing/stats/overview?period=30d",
		"/api/v1/marketing/stats/by?dim=campaign&period=30d",
	} {
		rec := do(t, mux, memberCtx, http.MethodGet, target, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("GET %s as team_member = %d, want 403 managers_only", target, rec.Code)
		}
	}
	for _, target := range []string{
		"/api/v1/bi/campaigns/overview?period=30d",
		"/api/v1/bi/campaigns/by?dim=campaign&period=30d",
	} {
		rec := do(t, mux, memberCtx, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s as team_member = %d, want 200 (bi grant, NO manager tier)\nbody: %s",
				target, rec.Code, rec.Body.String())
		}
	}
}

// The money block's key set is Card H1's contract: EVERY key present, `null`
// meaning "no opinion" and never a confident 0. Making the arithmetic real must
// not drop a key or turn a null into a 0.
func TestMoneyBlockKeepsEveryKeyAndNullsMeanNoOpinion(t *testing.T) {
	pool := setupReconDB(t)
	manager := seedUser(t, pool, "manager")
	mux := mountedMuxWithReports(testDeps(pool))
	_ = seedCampaignRow(t, pool, manager, "empty", "Empty", 500, nil)

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("overview = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Money map[string]json.RawMessage `json:"money"`
	}
	decode(t, rec, &envelope)
	for _, key := range []string{
		"revenue_cents", "discount_cents", "discount_basis", "net_cents", "per_dollar",
		"avg_order_cents_with", "avg_order_cents_without",
		"discount_implied_cents", "discount_actual_cents", "discount_unknown_rows",
	} {
		if _, ok := envelope.Money[key]; !ok {
			t.Errorf("money is missing key %q — a UI cannot tell \"no opinion\" from \"no key\"\nbody: %s", key, rec.Body.String())
		}
	}
	// zero discount → per_dollar is null, never 0.00, never Inf, never NaN.
	if got := string(envelope.Money["per_dollar"]); got != "null" {
		t.Errorf("per_dollar = %s with zero discount, want null", got)
	}
	if got := string(envelope.Money["avg_order_cents_with"]); got != "null" {
		t.Errorf("avg_order_cents_with = %s with no matched orders, want null", got)
	}
	if got := string(envelope.Money["discount_basis"]); got != `"implied"` {
		t.Errorf("discount_basis = %s on an empty period, want \"implied\"", got)
	}

	// GET /campaigns' money block is REAL now (it was H1's zero shape) and
	// carries the same key set.
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil)
	var list struct {
		Campaigns []struct {
			Money map[string]json.RawMessage `json:"money"`
		} `json:"campaigns"`
	}
	decode(t, rec, &list)
	if len(list.Campaigns) != 1 {
		t.Fatalf("list returned %d campaigns, want 1\nbody: %s", len(list.Campaigns), rec.Body.String())
	}
	for _, key := range []string{
		"revenue_cents", "discount_cents", "discount_basis", "net_cents", "per_dollar",
		"avg_order_cents_with", "avg_order_cents_without",
		"discount_implied_cents", "discount_actual_cents", "discount_unknown_rows",
	} {
		if _, ok := list.Campaigns[0].Money[key]; !ok {
			t.Errorf("GET /campaigns money is missing key %q\nbody: %s", key, rec.Body.String())
		}
	}
}

// GET /campaigns' funnel and money are the real arithmetic now. Card H2's UI
// renders from them.
func TestCampaignListCarriesRealMoneyPerCampaign(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))

	rec := do(t, mux, userCtx(f.manager, "manager"), http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /campaigns = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var out listCampaignsResponse
	decode(t, rec, &out)

	// welcome: r1 (1450/300) + r2 (900/250) + r7 (800/300) → revenue 3150, discount 850
	want := map[string]struct{ revenue, discount, redeemed int }{
		"Welcome":        {3150, 850, 3},
		"Wing Wednesday": {1200, 600, 3},
		"Catering":       {0, 4000, 1},
	}
	seen := 0
	for _, c := range out.Campaigns {
		w, ok := want[c.Name]
		if !ok {
			continue
		}
		seen++
		if c.Money.RevenueCents != w.revenue {
			t.Errorf("%s revenue_cents = %d, want %d", c.Name, c.Money.RevenueCents, w.revenue)
		}
		if c.Money.DiscountCents != w.discount {
			t.Errorf("%s discount_cents = %d, want %d", c.Name, c.Money.DiscountCents, w.discount)
		}
		if c.Funnel.Redeemed != w.redeemed {
			t.Errorf("%s funnel.redeemed = %d, want %d", c.Name, c.Funnel.Redeemed, w.redeemed)
		}
		if c.Money.NetCents != w.revenue-w.discount {
			t.Errorf("%s net_cents = %d, want %d", c.Name, c.Money.NetCents, w.revenue-w.discount)
		}
	}
	if seen != 3 {
		t.Errorf("found %d of the 3 fixture campaigns in the list", seen)
	}
}

// §5's drill-ins: dim=channel scoped to one campaign, dim=code scoped to one item.
func TestStatsByDrillInsScopeToTheirParent(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))
	ctx := userCtx(f.manager, "manager")

	rec := do(t, mux, ctx, http.MethodGet,
		"/api/v1/marketing/stats/by?dim=channel&campaign_id="+f.wingwed+"&period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("drill-in dim=channel&campaign_id = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var by StatsByResponse
	decode(t, rec, &by)
	// wingwed holds codes B1 (truck_sign) and B2 (instagram) only.
	if by.Totals.Redeemed != 3 || by.Totals.RevenueCents != 1200 {
		t.Errorf("wingwed channel drill-in totals = redeemed %d revenue %d, want 3 / 1200\nbody: %s",
			by.Totals.Redeemed, by.Totals.RevenueCents, rec.Body.String())
	}
	for _, row := range by.Rows {
		if row.Key != "truck_sign" && row.Key != "instagram" {
			t.Errorf("wingwed channel drill-in leaked row %q", row.Key)
		}
	}

	rec = do(t, mux, ctx, http.MethodGet,
		"/api/v1/marketing/stats/by?dim=code&item_id="+f.tray+"&period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("drill-in dim=code&item_id = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	decode(t, rec, &by)
	if len(by.Rows) != 1 || by.Rows[0].Key != "CCCCC2" {
		t.Errorf("tray code drill-in rows = %+v, want exactly the CCCCC2 code", by.Rows)
	}

	// an unknown dim is a client bug on a read; it gets a 400 rather than a
	// silently-wrong slice.
	rec = do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/by?dim=nonsense", nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("dim=nonsense = %d, want 400", rec.Code)
	}
}

// 🛑 THE LIVE ATTRIBUTION CASE, pinned. scan_attempts_mirror.code_id is the
// SUPABASE public.codes id and HQ has no copy of that table, so on real data
// today a non-nil code_id resolves to NO qr_codes row (and campaign_id arrives
// NULL). This test forces exactly that and asserts the engine degrades honestly:
//
//   - the channel AND item dimensions both answer `direct` — not `any`, which
//     would be an assertion about the campaign the data does not support;
//   - the campaign dimension answers `unattributed`;
//   - the row is counted in `discount_unknown_rows`, so its 0 face value cannot
//     read as "this redemption was free";
//   - Σ over every slice still equals the overview.
func TestUnresolvableCodeIDBucketsAsDirectAndCountsAsUnknown(t *testing.T) {
	pool := setupReconDB(t)
	manager := seedUser(t, pool, "manager")
	mux := mountedMuxWithReports(testDeps(pool))
	ctx := userCtx(manager, "manager")

	// A code_id that is a perfectly good uuid and names nothing in qr_codes —
	// which is what the mirror writes today.
	var raw string
	if err := pool.QueryRow(t.Context(), `SELECT gen_random_uuid()::text`).Scan(&raw); err != nil {
		t.Fatalf("gen uuid: %v", err)
	}
	_ = seedAttempt(t, pool, attemptFixture{RawCodeID: &raw})

	rec := do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/overview?period=30d", nil)
	var ov StatsOverviewResponse
	decode(t, rec, &ov)
	if ov.Funnel.Redeemed != 1 {
		t.Fatalf("redeemed = %d, want 1\nbody: %s", ov.Funnel.Redeemed, rec.Body.String())
	}
	if ov.Money.DiscountUnknownRows != 1 {
		t.Errorf("discount_unknown_rows = %d, want 1 — an unpriceable row must not read as a free one",
			ov.Money.DiscountUnknownRows)
	}
	if ov.Money.DiscountCents != 0 || ov.Money.DiscountImpliedCents != 0 {
		t.Errorf("discount = %d implied = %d, want 0/0 (no campaign resolved, so no face value exists)",
			ov.Money.DiscountCents, ov.Money.DiscountImpliedCents)
	}

	want := map[string]string{"campaign": StatsUnattributedKey, "channel": StatsDirectKey, "item": StatsDirectKey}
	for dim, key := range want {
		rec := do(t, mux, ctx, http.MethodGet, "/api/v1/marketing/stats/by?dim="+dim+"&period=30d", nil)
		var by StatsByResponse
		decode(t, rec, &by)
		if len(by.Rows) != 1 {
			t.Fatalf("dim=%s rows = %d, want 1\nbody: %s", dim, len(by.Rows), rec.Body.String())
		}
		if by.Rows[0].Key != key {
			t.Errorf("dim=%s row key = %q, want %q", dim, by.Rows[0].Key, key)
		}
		if by.Rows[0].Redeemed != ov.Funnel.Redeemed || by.Rows[0].DiscountUnknownRows != 1 {
			t.Errorf("dim=%s row does not reconcile: %+v", dim, by.Rows[0])
		}
	}
}

// signups is 0 with a STATED basis when card H5's `subscribers` is absent —
// never a failure, never a silent omission.
func TestSignupsStatesItsBasisWhenSubscribersIsAbsent(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))

	rec := do(t, mux, userCtx(f.manager, "manager"), http.MethodGet,
		"/api/v1/marketing/stats/overview?period=30d", nil)
	var ov StatsOverviewResponse
	decode(t, rec, &ov)

	if f.subscribersPresent {
		if ov.SignupsBasis != "subscribers" {
			t.Errorf("signups_basis = %q with the table present, want subscribers", ov.SignupsBasis)
		}
		if ov.Funnel.Signups != 5 {
			t.Errorf("signups = %d with the table present, want 5", ov.Funnel.Signups)
		}
	} else {
		if ov.SignupsBasis != "unavailable" {
			t.Errorf("signups_basis = %q with the table absent, want unavailable", ov.SignupsBasis)
		}
		if ov.Funnel.Signups != 0 {
			t.Errorf("signups = %d with the table absent, want 0", ov.Funnel.Signups)
		}
	}
	if ov.CodesSentBasis == "" {
		t.Errorf("codes_sent_basis is empty; codes sent must state where it came from")
	}
}
