package marketing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// landingGet drives the PUBLIC route with an explicit UA and RemoteAddr. It
// does NOT carry an auth user on the context — the whole point of GET /q/{short}
// is that a customer with a phone camera reaches it with no session at all.
func landingGet(mux http.Handler, method, short, ua, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/q/"+short, nil)
	req.Header.Set("User-Agent", ua)
	req.RemoteAddr = remoteAddr
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const iphoneUA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1"

// ─────────────────────────────────────────────────────────────────────────────
// done_when #2 — TestLandingLogsScanAndRedirectsWithUTM
//
// §5 public row: 302 to the landing with
// utm_source=qr&utm_medium=<channel>&utm_campaign=<slug>&utm_content=<item slug>&q=<short>,
// and a qr_scans row logged — under the 10-minute (short, ip_hash) dedupe, with
// HEAD and known link-preview UAs not logged.
// ─────────────────────────────────────────────────────────────────────────────
func TestLandingLogsScanAndRedirectsWithUTM(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	itemID := seedMenuItem(t, pool, "Six Piece Wings")
	mux := mountedMux(testDeps(pool))

	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Wing Wednesday", "offer_text": "$2 off any 6pc wings",
		"face_value_cents": 200, "runs_days": 14, "item_id": itemID,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	short := made.Codes[0].Short
	slug := made.Campaign.Slug

	rec := landingGet(mux, http.MethodGet, short, iphoneUA, "203.0.113.7:51515")
	if rec.Code != http.StatusFound {
		t.Fatalf("GET /q/%s = %d, want 302\nbody: %s", short, rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("302 with no Location header")
	}
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("Location %q is not a URL: %v", loc, err)
	}
	q := u.Query()
	for k, want := range map[string]string{
		"utm_source":   "qr",
		"utm_medium":   "flyer",
		"utm_campaign": slug,
		"utm_content":  "six-piece-wings",
		"q":            short,
	} {
		if got := q.Get(k); got != want {
			t.Errorf("Location %s = %q, want %q (full: %s)", k, got, want, loc)
		}
	}
	if u.Scheme != "https" {
		t.Errorf("Location scheme = %q, want https (%s)", u.Scheme, loc)
	}

	// Exactly one qr_scans row, carrying the coarse UA family and an ip_hash.
	if n := countScans(t, pool, short); n != 1 {
		t.Fatalf("qr_scans rows = %d after one scan, want 1", n)
	}
	var uaFamily, ipHash *string
	if err := pool.QueryRow(context.Background(),
		`SELECT ua_family, ip_hash FROM qr_scans WHERE short = $1`, short).Scan(&uaFamily, &ipHash); err != nil {
		t.Fatalf("select scan row: %v", err)
	}
	if uaFamily == nil || *uaFamily != "ios" {
		t.Errorf("ua_family = %v, want \"ios\"", uaFamily)
	}
	if ipHash == nil || len(*ipHash) != 64 {
		t.Errorf("ip_hash = %v, want a 64-char sha256 hex digest", ipHash)
	}

	// 10-minute (short, ip_hash) dedupe: the same phone re-scanning the same
	// sign does not inflate the funnel, but still gets forwarded.
	again := landingGet(mux, http.MethodGet, short, iphoneUA, "203.0.113.7:51616")
	if again.Code != http.StatusFound {
		t.Errorf("second scan = %d, want 302 (dedupe must not break the redirect)", again.Code)
	}
	if n := countScans(t, pool, short); n != 1 {
		t.Errorf("qr_scans rows = %d after a repeat scan inside the window, want 1", n)
	}

	// A different IP is a different person.
	other := landingGet(mux, http.MethodGet, short, iphoneUA, "198.51.100.9:4242")
	if other.Code != http.StatusFound {
		t.Errorf("other-IP scan = %d, want 302", other.Code)
	}
	if n := countScans(t, pool, short); n != 2 {
		t.Errorf("qr_scans rows = %d after a different IP, want 2", n)
	}

	// HEAD is not a scan (§5), but still answers the redirect.
	head := landingGet(mux, http.MethodHead, short, iphoneUA, "192.0.2.50:1111")
	if head.Code != http.StatusFound {
		t.Errorf("HEAD /q = %d, want 302", head.Code)
	}
	if n := countScans(t, pool, short); n != 2 {
		t.Errorf("qr_scans rows = %d after HEAD, want 2 (HEAD is not logged)", n)
	}

	// Known link-preview UAs are not scans either — one message pasted into a
	// group chat must not read as a customer.
	for _, bot := range []string{
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
		"Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
		"WhatsApp/2.23.20.0 A",
		"Twitterbot/1.0",
		"Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)",
	} {
		r := landingGet(mux, http.MethodGet, short, bot, "192.0.2.77:2222")
		if r.Code != http.StatusFound {
			t.Errorf("preview UA %q = %d, want 302", bot, r.Code)
		}
		if n := countScans(t, pool, short); n != 2 {
			t.Errorf("qr_scans rows = %d after preview UA %q, want 2 (previews are not logged)", n, bot)
		}
	}

	// ua_family is coarse: android and desktop are the other two live families.
	androidShort := made.Codes[0].Short
	_ = androidShort
	if r := landingGet(mux, http.MethodGet, short,
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120 Mobile Safari/537.36",
		"203.0.113.200:3333"); r.Code != http.StatusFound {
		t.Errorf("android scan = %d, want 302", r.Code)
	}
	var androidRows int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_scans WHERE short = $1 AND ua_family = 'android'`, short).Scan(&androidRows); err != nil {
		t.Fatalf("count android rows: %v", err)
	}
	if androidRows != 1 {
		t.Errorf("android-family rows = %d, want 1", androidRows)
	}
}

// A code with no item (the "Any item" campaign) omits utm_content rather than
// forwarding an empty one.
func TestLandingWithoutItemOmitsUTMContent(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Any Item Offer", "offer_text": "$2 off anything",
		"face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "receipt"}},
	})
	rec := landingGet(mux, http.MethodGet, made.Codes[0].Short, iphoneUA, "203.0.113.11:5555")
	if rec.Code != http.StatusFound {
		t.Fatalf("GET /q = %d, want 302", rec.Code)
	}
	u, _ := url.Parse(rec.Header().Get("Location"))
	if _, present := u.Query()["utm_content"]; present {
		t.Errorf("utm_content present with no item on the code: %s", rec.Header().Get("Location"))
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// done_when #3 — TestLandingInactiveCodeRendersEndedPage
//
// §5 public row: inactive or ended → 200 static "This offer has ended" page;
// unknown short → 404. A dead sign must never 302 a customer into a signup form
// for an offer that is gone.
// ─────────────────────────────────────────────────────────────────────────────
func TestLandingInactiveCodeRendersEndedPage(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "Ending Soon", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "truck_sign"}, {"channel": "flyer"}},
	})
	paused, live := made.Codes[0], made.Codes[1]

	// (a) the CODE is deactivated — the campaign is still live.
	if _, err := pool.Exec(context.Background(),
		`UPDATE qr_codes SET active = false WHERE id = $1`, paused.ID); err != nil {
		t.Fatalf("deactivate code: %v", err)
	}
	rec := landingGet(mux, http.MethodGet, paused.Short, iphoneUA, "203.0.113.21:6666")
	if rec.Code != http.StatusOK {
		t.Fatalf("inactive code = %d, want 200 (the ended page, not a redirect)\nbody: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("ended page Content-Type = %q, want text/html…", ct)
	}
	if !strings.Contains(rec.Body.String(), "This offer has ended") {
		t.Errorf("ended page body does not say \"This offer has ended\":\n%s", rec.Body.String())
	}
	if rec.Header().Get("Location") != "" {
		t.Error("ended page carried a Location header")
	}
	// The scan is still recorded — a dead sign that is still being scanned is
	// exactly what the manager needs to see.
	if n := countScans(t, pool, paused.Short); n != 1 {
		t.Errorf("qr_scans rows for the inactive code = %d, want 1", n)
	}

	// (b) the CAMPAIGN is ended — its still-active code shows the same page.
	if _, err := pool.Exec(context.Background(),
		`UPDATE campaigns_admin SET status = 'ended' WHERE id = $1`, made.Campaign.ID); err != nil {
		t.Fatalf("end campaign: %v", err)
	}
	rec = landingGet(mux, http.MethodGet, live.Short, iphoneUA, "203.0.113.22:7777")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "This offer has ended") {
		t.Fatalf("ended campaign = %d / %q, want 200 + the ended page", rec.Code, rec.Body.String())
	}

	// (c) the campaign's end date has passed, status untouched.
	if _, err := pool.Exec(context.Background(),
		`UPDATE campaigns_admin SET status = 'live', ends_at = now() - interval '1 day' WHERE id = $1`,
		made.Campaign.ID); err != nil {
		t.Fatalf("expire campaign: %v", err)
	}
	rec = landingGet(mux, http.MethodGet, live.Short, iphoneUA, "203.0.113.23:8888")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "This offer has ended") {
		t.Fatalf("expired campaign = %d, want 200 + the ended page\nbody: %s", rec.Code, rec.Body.String())
	}

	// (d) an unknown short is a 404, not an ended page — a typo is not an offer.
	rec = landingGet(mux, http.MethodGet, "ZZZZZZ", iphoneUA, "203.0.113.24:9999")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown short = %d, want 404\nbody: %s", rec.Code, rec.Body.String())
	}
	// …and it logs nothing: qr_scans.short is an FK to qr_codes.short, so an
	// unknown code has nowhere to be logged by construction.
	var strayRows int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_scans WHERE short = 'ZZZZZZ'`).Scan(&strayRows); err != nil {
		t.Fatalf("count stray rows: %v", err)
	}
	if strayRows != 0 {
		t.Errorf("unknown short logged %d qr_scans rows, want 0", strayRows)
	}
}
