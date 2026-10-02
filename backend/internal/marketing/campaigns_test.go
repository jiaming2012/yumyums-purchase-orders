package marketing

import (
	"context"
	"net/http"
	"regexp"
	"testing"
)

// shortRe mirrors the CHECK constraint migration 0083 puts on qr_codes.short:
// exactly 6 characters from the Crockford-style alphabet decision 189 fixes.
// The alphabet is repeated here deliberately — a test that imports the constant
// it is checking proves nothing about the constant.
var shortRe = regexp.MustCompile(`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$`)

// ─────────────────────────────────────────────────────────────────────────────
// done_when #1 — TestCreateCampaignMintsOneCodePerChannel
//
// POST /campaigns mints exactly one qr_codes row per requested channel, in the
// same transaction as the campaign, each with a unique 6-char short and the
// campaign's item inherited. Spec §5 row 2; slate Card 1 scope.
// ─────────────────────────────────────────────────────────────────────────────
func TestCreateCampaignMintsOneCodePerChannel(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	itemID := seedMenuItem(t, pool, "Six Piece Wings")
	mux := mountedMux(testDeps(pool))

	out := createCampaign(t, mux, mgr, map[string]any{
		"name":             "Wing Wednesday",
		"offer_text":       "$2 off any 6pc wings",
		"face_value_cents": 200,
		"runs_days":        14,
		"item_id":          itemID,
		"channels": []map[string]any{
			{"channel": "truck_sign", "placement": "Driver side"},
			{"channel": "flyer"},
			{"channel": "instagram"},
			{"channel": "other", "channel_label": "Farmers market table"},
		},
	})

	if got, want := len(out.Codes), 4; got != want {
		t.Fatalf("minted %d codes, want %d (one per channel)", got, want)
	}

	// One row per channel, no duplicates, no extras.
	seenChannel := map[string]int{}
	seenShort := map[string]bool{}
	for _, c := range out.Codes {
		seenChannel[c.Channel]++
		if !shortRe.MatchString(c.Short) {
			t.Errorf("code %s: short %q does not match the decision-189 alphabet", c.ID, c.Short)
		}
		if seenShort[c.Short] {
			t.Errorf("short %q minted twice in one campaign", c.Short)
		}
		seenShort[c.Short] = true
		if c.CampaignID != out.Campaign.ID {
			t.Errorf("code %s: campaign_id %q, want %q", c.ID, c.CampaignID, out.Campaign.ID)
		}
		if !c.Active {
			t.Errorf("code %s: minted inactive", c.ID)
		}
	}
	for _, ch := range []string{"truck_sign", "flyer", "instagram", "other"} {
		if seenChannel[ch] != 1 {
			t.Errorf("channel %q minted %d times, want exactly 1", ch, seenChannel[ch])
		}
	}

	// The transaction actually committed both halves: the rows are in the DB,
	// not merely in the response body.
	var dbCodes int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_codes WHERE campaign_id = $1`, out.Campaign.ID).Scan(&dbCodes); err != nil {
		t.Fatalf("count qr_codes: %v", err)
	}
	if dbCodes != 4 {
		t.Errorf("qr_codes rows for campaign = %d, want 4", dbCodes)
	}

	// item_id defaults to the campaign's item (§4: "defaults to the campaign's item").
	var codesWithItem int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_codes WHERE campaign_id = $1 AND item_id = $2`,
		out.Campaign.ID, itemID).Scan(&codesWithItem); err != nil {
		t.Fatalf("count qr_codes with item: %v", err)
	}
	if codesWithItem != 4 {
		t.Errorf("%d of 4 codes inherited the campaign item, want 4", codesWithItem)
	}

	// channel_label survives for channel='other' (the only channel that requires it).
	var label *string
	if err := pool.QueryRow(context.Background(),
		`SELECT channel_label FROM qr_codes WHERE campaign_id = $1 AND channel = 'other'`,
		out.Campaign.ID).Scan(&label); err != nil {
		t.Fatalf("select channel_label: %v", err)
	}
	if label == nil || *label != "Farmers market table" {
		t.Errorf("channel_label = %v, want %q", label, "Farmers market table")
	}

	if out.Campaign.Slug == "" {
		t.Error("campaign slug is empty — the UTM mirror and payload preview need it")
	}
}

// Channel 'other' without a label is the one shape §4 marks required.
func TestCreateCampaignRejectsOtherWithoutLabel(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPost, "/api/v1/marketing/campaigns",
		map[string]any{
			"name": "Unlabelled", "offer_text": "x", "face_value_cents": 100, "runs_days": 7,
			"channels": []map[string]any{{"channel": "other"}},
		})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST with unlabelled 'other' = %d, want 400\nbody: %s", rec.Code, rec.Body.String())
	}
}

// Two campaigns with the same name get distinct slugs — the slug is unique in
// the schema and is what UTM carries.
func TestCreateCampaignSlugsAreUnique(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	body := map[string]any{
		"name": "Wing Wednesday", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	}
	a := createCampaign(t, mux, mgr, body)
	b := createCampaign(t, mux, mgr, body)
	if a.Campaign.Slug == b.Campaign.Slug {
		t.Fatalf("both campaigns got slug %q — campaigns_admin.slug is UNIQUE", a.Campaign.Slug)
	}
}

// requires_online is DERIVED, never posted: face value at or above the
// marketing_settings threshold (#5, default $20.00) flips it on.
func TestRequiresOnlineDerivedFromThreshold(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	cheap := createCampaign(t, mux, mgr, map[string]any{
		"name": "Two Dollar Off", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	if cheap.Campaign.RequiresOnline {
		t.Errorf("face_value_cents=200 (< %d) set requires_online", DefaultRequiresOnlineThresholdCents)
	}

	rich := createCampaign(t, mux, mgr, map[string]any{
		"name": "Forty Dollar Off", "offer_text": "$40 off", "face_value_cents": 4000, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	if !rich.Campaign.RequiresOnline {
		t.Errorf("face_value_cents=4000 (>= %d) left requires_online false", DefaultRequiresOnlineThresholdCents)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// done_when #5 — TestTeamMemberGets403ManagersOnly
//
// §16 / handoff §5: the marketing grant opens the surface; the manager tier is
// enforced INSIDE the handler, and a team_member gets exactly
// 403 {"error":"managers_only"} — the envelope the UI renders as Locked.
// ─────────────────────────────────────────────────────────────────────────────
func TestTeamMemberGets403ManagersOnly(t *testing.T) {
	pool := setupTestDB(t)
	crew := seedUser(t, pool, "team_member")
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	// A real campaign to aim the per-id routes at.
	live := createCampaign(t, mux, mgr, map[string]any{
		"name": "Locked Probe", "offer_text": "$1 off", "face_value_cents": 100, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})

	cases := []struct {
		method, target string
		body           any
	}{
		{http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil},
		{http.MethodPost, "/api/v1/marketing/campaigns", map[string]any{
			"name": "Nope", "offer_text": "x", "face_value_cents": 100, "runs_days": 7,
			"channels": []map[string]any{{"channel": "flyer"}}}},
		{http.MethodGet, "/api/v1/marketing/campaigns/" + live.Campaign.ID, nil},
		{http.MethodPatch, "/api/v1/marketing/campaigns/" + live.Campaign.ID, map[string]any{"status": "paused"}},
		{http.MethodPost, "/api/v1/marketing/campaigns/" + live.Campaign.ID + "/codes", map[string]any{"channel": "sms"}},
		{http.MethodPatch, "/api/v1/marketing/codes/" + live.Codes[0].ID, map[string]any{"active": false}},
		{http.MethodGet, "/api/v1/marketing/codes/" + live.Codes[0].ID + ".png", nil},
	}

	for _, c := range cases {
		rec := do(t, mux, userCtx(crew, "team_member"), c.method, c.target, c.body)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s as team_member = %d, want 403\nbody: %s",
				c.method, c.target, rec.Code, rec.Body.String())
			continue
		}
		var env struct {
			Error string `json:"error"`
		}
		decode(t, rec, &env)
		if env.Error != "managers_only" {
			t.Errorf("%s %s as team_member error = %q, want %q", c.method, c.target, env.Error, "managers_only")
		}
	}

	// And the same routes answer a manager.
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /campaigns as manager = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
}

// GET /campaigns carries the funnel from qr_scans and the H3b zero-shape money
// block (this card ships the shape, not the arithmetic — see merge-intent).
func TestListCampaignsFunnelAndZeroMoney(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Funnel Probe", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}, {"channel": "receipt"}},
	})
	// Two scans on the first code, one on the second.
	short0, short1 := made.Codes[0].Short, made.Codes[1].Short
	for _, s := range []string{short0, short0, short1} {
		if _, err := pool.Exec(context.Background(),
			`INSERT INTO qr_scans (short, ip_hash) VALUES ($1, $2)`, s, randSuffix(t)); err != nil {
			t.Fatalf("seed scan: %v", err)
		}
	}

	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/campaigns?period=30d", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /campaigns = %d\nbody: %s", rec.Code, rec.Body.String())
	}
	var out listCampaignsResponse
	decode(t, rec, &out)
	if len(out.Campaigns) != 1 {
		t.Fatalf("got %d campaigns, want 1", len(out.Campaigns))
	}
	c := out.Campaigns[0]
	if c.Funnel.Scans != 3 {
		t.Errorf("funnel.scans = %d, want 3", c.Funnel.Scans)
	}
	if c.Money.DiscountBasis != "implied" || c.Money.RevenueCents != 0 ||
		c.Money.DiscountCents != 0 || c.Money.NetCents != 0 || c.Money.PerDollar != nil {
		t.Errorf("money = %+v, want the H3b zero shape {0,0,\"implied\",0,null}", c.Money)
	}
	if len(c.Codes) != 2 {
		t.Fatalf("got %d codes on the campaign row, want 2", len(c.Codes))
	}
	byShort := map[string]int{}
	for _, cc := range c.Codes {
		byShort[cc.Short] = cc.Scans
	}
	if byShort[short0] != 2 || byShort[short1] != 1 {
		t.Errorf("per-code scans = %v, want {%s:2, %s:1}", byShort, short0, short1)
	}
}

// PATCH /codes/{id} re-points a printed code without reprinting it: the short
// is immutable, everything else is editable.
func TestPatchCodeRepointsWithoutChangingShort(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Repoint Probe", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "truck_sign"}},
	})
	code := made.Codes[0]

	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPatch,
		"/api/v1/marketing/codes/"+code.ID,
		map[string]any{"active": false, "landing": "menu", "placement": "Passenger side"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /codes/{id} = %d\nbody: %s", rec.Code, rec.Body.String())
	}
	var out codeDTO
	decode(t, rec, &out)
	if out.Short != code.Short {
		t.Errorf("short changed from %q to %q — a printed code must survive a re-point", code.Short, out.Short)
	}
	if out.Active {
		t.Error("active stayed true after PATCH {active:false}")
	}
	if out.Landing == nil || *out.Landing != "menu" {
		t.Errorf("landing = %v, want \"menu\"", out.Landing)
	}
}

// PATCH /campaigns/{id} drives the lifecycle the designed list filters on.
func TestPatchCampaignStatus(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Status Probe", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPatch,
		"/api/v1/marketing/campaigns/"+made.Campaign.ID, map[string]any{"status": "paused"})
	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH /campaigns/{id} = %d\nbody: %s", rec.Code, rec.Body.String())
	}
	var out campaignDTO
	decode(t, rec, &out)
	if out.Status != "paused" {
		t.Errorf("status = %q, want \"paused\"", out.Status)
	}

	bad := do(t, mux, userCtx(mgr, "manager"), http.MethodPatch,
		"/api/v1/marketing/campaigns/"+made.Campaign.ID, map[string]any{"status": "archived"})
	if bad.Code != http.StatusBadRequest {
		t.Errorf("PATCH status=archived = %d, want 400", bad.Code)
	}
}

// POST /campaigns/{id}/codes adds a channel after the fact (the designed
// "add a channel" affordance on the detail sheet).
func TestAddCodeToExistingCampaign(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Add Code Probe", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPost,
		"/api/v1/marketing/campaigns/"+made.Campaign.ID+"/codes",
		map[string]any{"channel": "table_tent", "placement": "Table 4"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /campaigns/{id}/codes = %d\nbody: %s", rec.Code, rec.Body.String())
	}
	var out codeDTO
	decode(t, rec, &out)
	if out.Channel != "table_tent" || !shortRe.MatchString(out.Short) {
		t.Errorf("new code = %+v, want channel table_tent and a valid short", out)
	}
	if out.Short == made.Codes[0].Short {
		t.Error("new code reused the existing short")
	}
}
