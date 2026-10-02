package marketing

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// ── G6 fix round, card h3b (run 20261002) ──

// F1 — `GET /campaigns` must be able to say "this campaign's money could not be
// attributed" instead of a confident $0.00.
//
// G6's live shape: a campaign with scans and ONE MATCHED $30.00 order whose
// scan_attempts_mirror.campaign_id is NULL (which is every row on live data —
// see D-4). The money exists; it sits in the invisible `unattributed` row of the
// by-campaign slice. On the campaigns list the campaign reported
// funnel.redeemed 0 / revenue 0 / discount 0 / discount_unknown_rows 0 — the one
// honesty signal this card built did not fire on the endpoint where a manager
// actually reads a campaign's money, and card H2's money strip is ALREADY
// SHIPPED against that block.
func TestCampaignListDisclosesUnattributedMoney(t *testing.T) {
	pool := setupReconDB(t)
	manager := seedUser(t, pool, "manager")
	mux := mountedMuxWithReports(testDeps(pool))
	campaign := seedCampaignRow(t, pool, manager, "unattr", "Unattributed Money", 500, nil)
	_ = seedCodeRow(t, pool, manager, campaign, "UUUUU2", "truck_sign", nil)
	seedScanRows(t, pool, "UUUUU2", 6)

	// The live shape: a code_id that names no qr_codes row, campaign_id NULL,
	// and a MATCHED $30.00 Toast order.
	var raw string
	if err := pool.QueryRow(t.Context(), `SELECT gen_random_uuid()::text`).Scan(&raw); err != nil {
		t.Fatalf("gen uuid: %v", err)
	}
	scannedAt := time.Now().Add(-3 * time.Hour)
	_ = seedAttempt(t, pool, attemptFixture{
		RawCodeID: &raw, CampaignID: nil, ScannedAt: scannedAt, OrderNumber: strptr("3001"),
	})
	seedToastOrder(t, pool, scannedAt, "3001", 3000, 400, scannedAt)

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/campaigns?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /campaigns = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var out listCampaignsResponse
	decode(t, rec, &out)
	if len(out.Campaigns) != 1 {
		t.Fatalf("got %d campaigns, want 1", len(out.Campaigns))
	}
	m := out.Campaigns[0].Money

	// The campaign's OWN figures are genuinely zero — nothing was attributed to
	// it. That part was never wrong.
	if m.RevenueCents != 0 || out.Campaigns[0].Funnel.Redeemed != 0 {
		t.Errorf("campaign's own revenue/redeemed = %d/%d, want 0/0",
			m.RevenueCents, out.Campaigns[0].Funnel.Redeemed)
	}
	// 🛑 But the money EXISTS and the block must say so.
	if m.UnattributedRedeemed == nil {
		t.Fatalf("money.unattributed_redeemed is null on the campaigns list; it is a PERIOD fact and must be stated\nbody: %s", rec.Body.String())
	}
	if *m.UnattributedRedeemed != 1 {
		t.Errorf("money.unattributed_redeemed = %d, want 1", *m.UnattributedRedeemed)
	}
	if m.UnattributedRevenueCents == nil || *m.UnattributedRevenueCents != 3000 {
		t.Errorf("money.unattributed_revenue_cents = %v, want 3000 — a manager reading $0.00 must be able to see the $30.00 that exists", m.UnattributedRevenueCents)
	}

	// The detail route carries the same disclosure.
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/campaigns/"+campaign+"?period=30d", nil)
	var detail campaignDTO
	decode(t, rec, &detail)
	if detail.Money.UnattributedRedeemed == nil || *detail.Money.UnattributedRedeemed != 1 {
		t.Errorf("GET /campaigns/{id} unattributed_redeemed = %v, want 1", detail.Money.UnattributedRedeemed)
	}

	// Every key still present, and the two new ones among them.
	var envelope struct {
		Campaigns []struct {
			Money map[string]json.RawMessage `json:"money"`
		} `json:"campaigns"`
	}
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil)
	decode(t, rec, &envelope)
	for _, key := range []string{
		"revenue_cents", "discount_cents", "discount_basis", "net_cents", "per_dollar",
		"avg_order_cents_with", "avg_order_cents_without",
		"discount_implied_cents", "discount_actual_cents", "discount_unknown_rows",
		"unattributed_redeemed", "unattributed_revenue_cents",
	} {
		if _, ok := envelope.Campaigns[0].Money[key]; !ok {
			t.Errorf("GET /campaigns money is missing key %q", key)
		}
	}
}

// A period with NOTHING unattributed reports 0, not null: that zero is a real
// fact ("every redemption found its campaign"), and it is what lets a UI render
// a plain $0.00 with confidence.
func TestCampaignListReportsZeroUnattributedAsAFact(t *testing.T) {
	pool := setupReconDB(t)
	manager := seedUser(t, pool, "manager")
	mux := mountedMuxWithReports(testDeps(pool))
	campaign := seedCampaignRow(t, pool, manager, "clean", "Clean Period", 500, nil)
	code := seedCodeRow(t, pool, manager, campaign, "KKKKK2", "flyer", nil)
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign})

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/campaigns?period=30d", nil)
	var out listCampaignsResponse
	decode(t, rec, &out)
	m := out.Campaigns[0].Money
	if m.UnattributedRedeemed == nil || *m.UnattributedRedeemed != 0 {
		t.Errorf("unattributed_redeemed = %v on a fully-attributed period, want 0 (a stated fact, not a null)", m.UnattributedRedeemed)
	}
	if m.UnattributedRevenueCents == nil || *m.UnattributedRevenueCents != 0 {
		t.Errorf("unattributed_revenue_cents = %v, want 0", m.UnattributedRevenueCents)
	}
}

// A slice row has NO opinion on a period fact, so it carries null — the
// established meaning of null in this block. The `unattributed` row of the
// by-campaign slice is where a slice states the figure.
func TestSliceRowsCarryNullForThePeriodScopedKeys(t *testing.T) {
	pool := setupReconDB(t)
	f := seedSlicesFixture(t, pool)
	mux := mountedMuxWithReports(testDeps(pool))

	rec := do(t, mux, userCtx(f.manager, "manager"), http.MethodGet,
		"/api/v1/marketing/stats/by?dim=campaign&period=30d", nil)
	var envelope struct {
		Rows   []map[string]json.RawMessage `json:"rows"`
		Totals map[string]json.RawMessage   `json:"totals"`
	}
	decode(t, rec, &envelope)
	for i, row := range envelope.Rows {
		for _, key := range []string{"unattributed_redeemed", "unattributed_revenue_cents", "avg_order_cents_without"} {
			raw, ok := row[key]
			if !ok {
				t.Errorf("rows[%d] is missing key %q", i, key)
				continue
			}
			if string(raw) != "null" {
				t.Errorf("rows[%d].%s = %s, want null (a slice row has no opinion on a period fact)", i, key, raw)
			}
		}
	}
	if string(envelope.Totals["unattributed_redeemed"]) != "null" {
		t.Errorf("totals.unattributed_redeemed = %s, want null", envelope.Totals["unattributed_redeemed"])
	}
}

// F2 — the orphan rate must DISCLOSE which numerator produced it.
//
// 🛑 THE RULE IS NOT CHANGED HERE. `night-crew decisions log` returned
// `verdict: park` at top severity and the question is D-5 in
// DECISIONS-NEEDED.md. This test pins the numerator AS SHIPPED so that whichever
// way the operator rules, the change is a deliberate edit to an asserted value
// rather than a silent drift — and asserts the wire now names the definition.
//
// The fixture is G6's five-row shape, which this card's own spike fixture never
// modelled because it has no UNMATCHED row:
//
//	a1 matched                      → not in the numerator
//	a2 unmatched (order typed, no Toast match)  → IN, under the shipped rule
//	a3 orphan (no order number)     → IN under either reading
//	a4 declined duplicate_scan      → excluded by decision 190
//	a5 declined customer_left       → IN under either reading
//
// Shipped rule (every accepted row with no matched order, minus duplicate_scan
// declines): 3/5 = 60%. The narrower literal reading of "orphans ÷ accepted"
// (the orphan BUCKET only, plus declines): 2/5 = 40%. Against a 10% threshold.
func TestOrphanRateDisclosesItsNumerator(t *testing.T) {
	pool := setupReconDB(t)
	manager := seedUser(t, pool, "manager")
	mux := mountedMuxWithReports(testDeps(pool))
	campaign := seedCampaignRow(t, pool, manager, "orph", "Orphan Rate", 300, nil)
	code := seedCodeRow(t, pool, manager, campaign, "NNNNN2", "truck_sign", nil)
	scannedAt := time.Now().Add(-4 * time.Hour)

	// a1 — matched
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt, OrderNumber: strptr("201")})
	seedToastOrder(t, pool, scannedAt, "201", 1500, 300, scannedAt)
	// a2 — UNMATCHED: an order number was typed and Toast has no such order
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt, OrderNumber: strptr("999")})
	// a3 — orphan
	_ = seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt})
	// a4 — declined duplicate_scan (excluded)
	a4 := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt})
	seedDecision(t, pool, a4, "declined", strptr("duplicate_scan"), nil, nil, manager)
	// a5 — declined customer_left (counted)
	a5 := seedAttempt(t, pool, attemptFixture{CodeID: &code, CampaignID: &campaign, ScannedAt: scannedAt})
	seedDecision(t, pool, a5, "declined", strptr("customer_left"), nil, nil, manager)

	rec := do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/stats/overview?period=30d", nil)
	var ov StatsOverviewResponse
	decode(t, rec, &ov)
	r := ov.Reconciliation

	if r.OrphanRate == nil {
		t.Fatalf("orphan_rate is null with 5 accepted attempts\nbody: %s", rec.Body.String())
	}
	// PINNED, not endorsed: the shipped numerator includes the unmatched row.
	if got := *r.OrphanRate; got != 0.6 {
		t.Errorf("orphan_rate = %v, want 0.6 — the SHIPPED numerator (3 of 5) counts the unmatched row; the narrower literal reading would give 0.4. D-5 is parked; if the operator rules for the narrow reading, change this value deliberately.", got)
	}
	if r.OrphanNumerator != 3 || r.OrphanDenominator != 5 {
		t.Errorf("orphan_numerator/denominator = %d/%d, want 3/5 — the figure must carry its own arithmetic",
			r.OrphanNumerator, r.OrphanDenominator)
	}
	if r.OrphanRateBasis != OrphanBasisUnmatchedAndOrphans {
		t.Errorf("orphan_rate_basis = %q, want %q — the wire must NAME the definition in force (D-5)",
			r.OrphanRateBasis, OrphanBasisUnmatchedAndOrphans)
	}
	// The queue proves the decomposition the basis name claims.
	if r.Matched != 1 || r.Declined != 2 {
		t.Errorf("matched/declined = %d/%d, want 1/2", r.Matched, r.Declined)
	}
	if ov.NeedsLook.Unmatched != 1 || ov.NeedsLook.Orphans != 1 {
		t.Errorf("needs_look unmatched/orphans = %d/%d, want 1/1 — the unmatched row is the one D-5 is about",
			ov.NeedsLook.Unmatched, ov.NeedsLook.Orphans)
	}
	// Even with no attempts the basis is stated, so a consumer never has to
	// guess which definition a null rate would have used.
	if _, err := pool.Exec(t.Context(), `TRUNCATE reconciliation_decisions, scan_attempts_mirror CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	rec = do(t, mux, userCtx(manager, "manager"), http.MethodGet,
		"/api/v1/marketing/stats/overview?period=30d", nil)
	decode(t, rec, &ov)
	if ov.Reconciliation.OrphanRate != nil {
		t.Errorf("orphan_rate = %v with no attempts, want null", *ov.Reconciliation.OrphanRate)
	}
	if ov.Reconciliation.OrphanRateBasis != OrphanBasisUnmatchedAndOrphans {
		t.Errorf("orphan_rate_basis = %q on an empty period, want it stated anyway", ov.Reconciliation.OrphanRateBasis)
	}
}
