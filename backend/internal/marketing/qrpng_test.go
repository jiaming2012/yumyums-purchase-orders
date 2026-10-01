package marketing

import (
	"bytes"
	"image/png"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

// ─────────────────────────────────────────────────────────────────────────────
// GET /codes/{id}.png — spec §5 row 7: image/png of the short URL, quiet zone 4
// modules, Cache-Control: private, max-age=3600.
//
// The BINDING build-fact from the goal ledger (spike 01, exit 0): the 35-char
// URL https://hq.yumyums.kitchen/q/<6 chars> encodes at QR version 3 (29 modules
// a side) at Medium, and version 3 is the CEILING — a longer host or a 7-char
// key tips to version 4 (33 modules) and stops scanning from a truck sign. This
// test pins that ceiling so a later change to the payload cannot quietly cross
// it. (The decode half of spike 01 belongs to the vendored browser scanner and
// is asserted there, not here.)
// ─────────────────────────────────────────────────────────────────────────────
func TestPayloadStaysAtQRVersion3(t *testing.T) {
	payload := DefaultQRBaseURL + "/q/7KQ2M3"
	if got, want := len(payload), 35; got != want {
		t.Errorf("payload %q is %d chars, want %d — the version-3 ceiling was measured at %d", payload, got, want, want)
	}
	q, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	if q.VersionNumber > 3 {
		t.Errorf("payload %q encodes at QR version %d; the signed build-fact is that version 3 (29 modules) "+
			"is the ceiling for a truck-sign scan. Do not grow the payload.", payload, q.VersionNumber)
	}
	if q.DisableBorder {
		t.Error("DisableBorder is set — §5 requires a 4-module quiet zone")
	}
}

func TestCodePNGHandler(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": "PNG Probe", "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "truck_sign"}},
	})
	code := made.Codes[0]

	// Default size, no query param.
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet,
		"/api/v1/marketing/codes/"+code.ID+".png", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /codes/{id}.png = %d\nbody: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "private, max-age=3600" {
		t.Errorf("Cache-Control = %q, want %q", cc, "private, max-age=3600")
	}
	img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("response body is not a decodable PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != DefaultPNGSize || b.Dy() != DefaultPNGSize {
		t.Errorf("default PNG is %dx%d, want %dx%d", b.Dx(), b.Dy(), DefaultPNGSize, DefaultPNGSize)
	}

	// Each size on the ladder is served at that size.
	for _, size := range PNGSizeLadder {
		r := do(t, mux, userCtx(mgr, "manager"), http.MethodGet,
			"/api/v1/marketing/codes/"+code.ID+".png?size="+strconv.Itoa(size), nil)
		if r.Code != http.StatusOK {
			t.Errorf("size=%d = %d, want 200", size, r.Code)
			continue
		}
		im, err := png.Decode(bytes.NewReader(r.Body.Bytes()))
		if err != nil {
			t.Errorf("size=%d: not a PNG: %v", size, err)
			continue
		}
		if b := im.Bounds(); b.Dx() != size {
			t.Errorf("size=%d produced %dpx", size, b.Dx())
		}
	}

	// Off-ladder sizes are refused loudly rather than silently snapped — a
	// print job that asked for 900px and got 1024 is a surprise at the printer.
	for _, bad := range []string{"900", "0", "-512", "huge", "99999"} {
		r := do(t, mux, userCtx(mgr, "manager"), http.MethodGet,
			"/api/v1/marketing/codes/"+code.ID+".png?size="+bad, nil)
		if r.Code != http.StatusBadRequest {
			t.Errorf("size=%s = %d, want 400", bad, r.Code)
		}
	}

	// Unknown code id → 404.
	r := do(t, mux, userCtx(mgr, "manager"), http.MethodGet,
		"/api/v1/marketing/codes/00000000-0000-4000-8000-000000000000.png", nil)
	if r.Code != http.StatusNotFound {
		t.Errorf("unknown code png = %d, want 404", r.Code)
	}
}

// ── shortCode / slugify units — no DB ──

func TestNewShortCodeShape(t *testing.T) {
	re := regexp.MustCompile(`^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$`)
	seen := map[string]bool{}
	for i := 0; i < 2000; i++ {
		s, err := newShortCode()
		if err != nil {
			t.Fatalf("newShortCode: %v", err)
		}
		if !re.MatchString(s) {
			t.Fatalf("newShortCode produced %q, which is outside the decision-189 alphabet", s)
		}
		seen[s] = true
	}
	// 32^6 ≈ 1.07e9: 2000 draws colliding more than a handful of times would
	// mean the generator is not drawing uniformly.
	if len(seen) < 1990 {
		t.Errorf("2000 draws produced only %d distinct codes — the generator is biased", len(seen))
	}
	// The alphabet excludes the four characters a human misreads off a sign.
	for _, bad := range []string{"0", "1", "I", "O"} {
		if strings.Contains(shortCodeAlphabet, bad) {
			t.Errorf("alphabet contains %q — decision 189 excludes 0/1/I/O", bad)
		}
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Wing Wednesday":       "wing-wednesday",
		"$2 off any 6pc wings": "2-off-any-6pc-wings",
		"  Trailing  spaces  ": "trailing-spaces",
		"Six Piece Wings":      "six-piece-wings",
		"Café   Special!!":     "caf-special",
		"---":                  "",
		"":                     "",
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUAFamily(t *testing.T) {
	cases := map[string]string{
		iphoneUA: "ios",
		"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15":                     "ios",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/120 Mobile":          "android",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120 Safari":   "desktop",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120 Safari/537.36":  "desktop",
		"facebookexternalhit/1.1":                                                                "bot",
		"Slackbot-LinkExpanding 1.0":                                                             "bot",
		"":                                                                                       "desktop",
	}
	for ua, want := range cases {
		if got := uaFamily(ua); got != want {
			t.Errorf("uaFamily(%q) = %q, want %q", ua, got, want)
		}
	}
}

func TestIsLinkPreview(t *testing.T) {
	for _, ua := range []string{
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
		"Facebot",
		"Twitterbot/1.0",
		"Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
		"Mozilla/5.0 (compatible; Discordbot/2.0; +https://discordapp.com)",
		"WhatsApp/2.23.20.0 A",
		"TelegramBot (like TwitterBot)",
		"LinkedInBot/1.0 (compatible; Mozilla/5.0; Jakarta Commons-HttpClient/3.1)",
		"SkypeUriPreview Preview/0.5",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"Applebot/0.1; +http://www.apple.com/go/applebot",
		"redditbot/1.0",
		"Pinterest/0.2 (+http://www.pinterest.com/bot.html)",
	} {
		if !isLinkPreview(ua) {
			t.Errorf("isLinkPreview(%q) = false, want true", ua)
		}
	}
	for _, ua := range []string{iphoneUA, "Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/120 Mobile"} {
		if isLinkPreview(ua) {
			t.Errorf("isLinkPreview(%q) = true — a real customer was discarded", ua)
		}
	}
}

// looksAutomated and isLinkPreview are deliberately DIFFERENT sets: a named
// preview fetcher is known not to be a customer and is never logged; a generic
// self-identifying bot is a suspicion, is logged, and lands as ua_family='bot'.
func TestLooksAutomatedIsWiderThanIsLinkPreview(t *testing.T) {
	// Named preview fetchers are a subset of "looks automated".
	for _, ua := range []string{"facebookexternalhit/1.1", "Slackbot-LinkExpanding 1.0", "Twitterbot/1.0"} {
		if !isLinkPreview(ua) {
			t.Errorf("isLinkPreview(%q) = false", ua)
		}
	}
	// …and these are automated but NOT named, so they get logged as 'bot'.
	for _, ua := range []string{
		"SomeUnlistedBot/3.1", "curl/8.4.0", "python-requests/2.31.0",
		"Go-http-client/2.0", "Scrapy/2.11", "Mozilla/5.0 HeadlessChrome/120",
	} {
		if isLinkPreview(ua) {
			t.Errorf("isLinkPreview(%q) = true — it is not a NAMED preview fetcher and must be logged", ua)
		}
		if !looksAutomated(ua) {
			t.Errorf("looksAutomated(%q) = false", ua)
		}
		if got := uaFamily(ua); got != "bot" {
			t.Errorf("uaFamily(%q) = %q, want \"bot\"", ua, got)
		}
	}
	// A real phone is neither.
	if looksAutomated(iphoneUA) || isLinkPreview(iphoneUA) {
		t.Error("an iPhone UA read as automation")
	}
}
