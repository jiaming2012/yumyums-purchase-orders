package marketing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// scanDedupeWindow is §5's "10-minute (short, ip_hash) dedupe". It is enforced
// at WRITE time — see logScan — so every read of qr_scans is a plain count and
// the three slices cannot disagree about what a scan is.
const scanDedupeWindow = 10 * time.Minute

// endedPage is the §5 "This offer has ended" response: a 200, not a redirect
// and not a 404. A dead sign that is still being scanned is real information —
// the customer gets a plain answer and the manager gets the scan in the funnel.
//
// Self-contained and tiny on purpose: it is served to a stranger's phone on a
// truck-side data connection, with no HQ session, so it loads no CSS, no JS and
// no font.
const endedPage = `<!doctype html>
<html lang="en"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>This offer has ended</title>
<style>
:root{color-scheme:light dark}
body{margin:0;min-height:100vh;display:grid;place-items:center;
  font:16px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;
  background:#f5f5f3;color:#1a1a1a;padding:24px}
@media (prefers-color-scheme:dark){body{background:#17171a;color:#f0f0f0}}
.c{max-width:22rem;text-align:center}
h1{font-size:1.4rem;margin:0 0 .5rem}
p{margin:0;opacity:.75}
.e{font-size:2.5rem;margin-bottom:.75rem}
</style></head>
<body><div class="c">
<div class="e">🚚</div>
<h1>This offer has ended</h1>
<p>Thanks for scanning. Come find the truck for what&rsquo;s on today.</p>
</div></body></html>
`

// linkPreviewUAs are the user agents that fetch a URL to render a preview card
// rather than because a human is looking at it. They are forwarded like anyone
// else but NOT logged (§5): one offer link pasted into a group chat would
// otherwise read as a crowd of customers and inflate every funnel it touches.
//
// Matched case-insensitively as substrings. This list is the night's call; it
// is deliberately generous, because a missed bot inflates the numbers the
// orphan rate is judged against while a mis-flagged human loses one scan.
var linkPreviewUAs = []string{
	"facebookexternalhit", "facebot",
	"twitterbot",
	"slackbot", "slack-imgproxy",
	"discordbot",
	"whatsapp",
	"telegrambot",
	"linkedinbot",
	"pinterest",
	"skypeuripreview",
	"redditbot",
	"applebot",
	"googlebot",
	"bingbot",
	"yandexbot",
	"embedly",
	"quora link preview",
	"vkshare",
	"w3c_validator",
	"developers.google.com/+/web/snippet",
	"bitlybot",
	"nuzzel",
	"outbrain",
	"flipboard",
	"tumblr",
	"viber",
	"line/",
}

// automationTokens are the generic self-identifying-automation substrings. They
// are NOT in linkPreviewUAs, and the split is deliberate.
//
// A named preview fetcher is something we KNOW is not a customer, so it is not
// logged at all. An unlisted agent that merely calls itself a bot is a
// SUSPICION: §5 defines scans as qr_scans rows after the dedupe and says
// nothing about excluding bots, so dropping these silently would be this card
// inventing a metric. Instead they are logged with ua_family='bot' — which is
// also the only thing that gives §4's own comment ("coarse: ios / android /
// desktop / bot") a reachable value, and leaves H3b/H4 free to decide whether
// a slice wants to exclude them.
var automationTokens = []string{
	"bot", "crawler", "spider", "scrapy", "curl/", "wget/",
	"python-requests", "go-http-client", "okhttp", "headlesschrome",
	"httpclient", "libwww", "lighthouse", "pingdom", "uptimerobot",
}

func looksAutomated(ua string) bool {
	l := strings.ToLower(ua)
	for _, needle := range automationTokens {
		if strings.Contains(l, needle) {
			return true
		}
	}
	return false
}

// isLinkPreview reports whether ua is one of the NAMED preview fetchers above —
// the set that is forwarded but never logged.
func isLinkPreview(ua string) bool {
	l := strings.ToLower(ua)
	for _, needle := range linkPreviewUAs {
		if strings.Contains(l, needle) {
			return true
		}
	}
	return false
}

// uaFamily reduces a user agent to the coarse family qr_scans.ua_family holds:
// ios / android / desktop / bot (§4's own comment). Coarse is the point — this
// is a channel-quality signal, not device analytics, and anything finer is
// personal data this table has no business keeping.
//
// An empty UA reads as desktop: a stripped UA is far more often a privacy
// browser than a bot, and the bot list above already catches self-identifying
// crawlers.
func uaFamily(ua string) string {
	l := strings.ToLower(ua)
	if isLinkPreview(ua) || looksAutomated(ua) {
		return "bot"
	}
	switch {
	case strings.Contains(l, "iphone"), strings.Contains(l, "ipad"), strings.Contains(l, "ipod"):
		return "ios"
	case strings.Contains(l, "android"):
		return "android"
	default:
		return "desktop"
	}
}

// hashIP renders the dedupe key for an IP.
//
// sha256(ip + "|" + YYYY-MM-DD + "|" + salt), per §4's "sha256(ip + daily
// salt); for 10-minute dedupe only". The date rotates the hash daily so the
// table cannot be used to follow one phone across weeks, and the salt
// (HQ_QR_IP_SALT) is a rotation knob rather than a secret — the hash is scoped
// to dedupe by the schema's own comment, not relied on for anonymity.
func hashIP(remoteAddr, salt string) string {
	ip := remoteAddr
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		ip = host
	}
	if ip == "" {
		return ""
	}
	day := time.Now().UTC().Format("2006-01-02")
	sum := sha256.Sum256([]byte(ip + "|" + day + "|" + salt))
	return hex.EncodeToString(sum[:])
}

// clientIP prefers the proxy headers Cloudflare Tunnel sets, because in prod
// RemoteAddr is the tunnel, not the customer — one shared RemoteAddr would
// dedupe every customer in the city down to a single scan.
//
// 🛑 THESE HEADERS ARE CLIENT-SUPPLIED AND FORGEABLE, and that is accepted here
// rather than overlooked. This route is public and unauthenticated, so anyone
// can vary CF-Connecting-IP and defeat the dedupe to inflate a scan count. The
// value feeds ONE thing — a 10-minute dedupe on a funnel number — and no
// authorization, no money and no row visibility. It is not a trust boundary and
// must never become one: if a later card wants to key anything that MATTERS off
// the caller's address, it needs a trusted-proxy allowlist first, not this
// function.
func clientIP(r *http.Request) string {
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		// Left-most entry is the original client.
		if first, _, found := strings.Cut(v, ","); found {
			return strings.TrimSpace(first)
		}
		return strings.TrimSpace(v)
	}
	return r.RemoteAddr
}

// codeTarget is everything the landing needs, in one query.
type codeTarget struct {
	Short        string
	Channel      string
	Active       bool
	CodeLanding  *string
	CampaignSlug string
	CampLanding  string
	Status       string
	EndsAt       time.Time
	ItemName     *string
}

// live reports whether this code should forward a customer to the offer.
func (t codeTarget) live(now time.Time) bool {
	if !t.Active {
		return false
	}
	if t.Status == "paused" || t.Status == "ended" {
		return false
	}
	return t.EndsAt.After(now)
}

// LandingHandler is §5's public row: GET /q/{short}.
//
// It is mounted by MountPublic on the ROOT router, OUTSIDE /api/v1 and outside
// every auth gate — a customer pointing a phone camera at a truck sign has no
// HQ session and must not be asked for one.
//
// Three outcomes:
//
//	live code    -> 302 to the landing with the UTM mirror + q=<short>
//	dead code    -> 200 "This offer has ended" (logged: a scanned dead sign is a signal)
//	unknown code -> 404 (a typo is not an offer, and qr_scans.short is an FK, so
//	                     there is nowhere to log it even if we wanted to)
func LandingHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		short := chi.URLParam(r, "short")
		ctx := r.Context()

		var t codeTarget
		err := d.Pool.QueryRow(ctx, `
			SELECT k.short, k.channel, k.active, k.landing,
			       c.slug, c.landing, c.status, c.ends_at,
			       COALESCE(m.name, cm.name)
			FROM qr_codes k
			JOIN campaigns_admin c ON c.id = k.campaign_id
			LEFT JOIN menu_items m  ON m.id = k.item_id
			LEFT JOIN menu_items cm ON cm.id = c.item_id
			WHERE k.short = $1`, short).
			Scan(&t.Short, &t.Channel, &t.Active, &t.CodeLanding,
				&t.CampaignSlug, &t.CampLanding, &t.Status, &t.EndsAt, &t.ItemName)
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			slog.Error("marketing: resolve short code", "error", err, "short", short)
			// A read failure must not look like "no such offer": a customer
			// standing at the truck gets the ended page, not a 404 that says
			// the sign is fake.
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
			return
		}

		// Log BEFORE answering, and never let a logging failure cost the
		// customer their redirect.
		//
		// HEAD is not a scan (§5): it is a monitor or an unfurler, and it still
		// gets the same answer a GET would.
		if r.Method != http.MethodHead && !isLinkPreview(r.UserAgent()) {
			logScan(r, d, t)
		}

		if !t.live(time.Now()) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			// A dead code is dead now; a live one may be dead in a minute.
			// Neither is worth a cache that outlives a re-point.
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusOK)
			if r.Method != http.MethodHead {
				_, _ = w.Write([]byte(endedPage))
			}
			return
		}

		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, d.landingURL(t), http.StatusFound)
	}
}

// logScan inserts the qr_scans row, honouring the 10-minute
// (short, ip_hash) dedupe. Failures are logged and swallowed: a funnel row is
// worth less than a customer reaching the offer.
func logScan(r *http.Request, d Deps, t codeTarget) {
	ctx := r.Context()
	ipHash := hashIP(clientIP(r), d.IPSalt)
	family := uaFamily(r.UserAgent())
	var referrer *string
	if ref := r.Referer(); ref != "" {
		referrer = &ref
	}

	if err := insertScan(ctx, d.Pool, t.Short, family, referrer, nullIfEmpty(ipHash)); err != nil {
		slog.Error("marketing: log qr scan", "error", err, "short", t.Short)
	}
}

// scanExecer is the one method insertScan needs. The pool, a single pooled
// connection and a transaction all satisfy it, which is what lets the
// concurrency test run THIS statement on twelve connections it opened itself.
type scanExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// insertScan writes one qr_scans row under the 10-minute (short, ip_hash)
// dedupe. It is the only statement in the tree that inserts a scan.
func insertScan(ctx context.Context, db scanExecer, short, family string, referrer, ipHash *string) error {
	// The dedupe lives in the WHERE NOT EXISTS, so it is one statement and one
	// definition. A NULL ip_hash never dedupes — we cannot tell two anonymous
	// scans apart, and over-counting is the honest failure here.
	_, err := db.Exec(ctx, `
		INSERT INTO qr_scans (short, ua_family, referrer, ip_hash)
		SELECT $1, $2, $3, $4
		WHERE $4::text IS NULL OR NOT EXISTS (
		  SELECT 1 FROM qr_scans s
		  WHERE s.short = $1 AND s.ip_hash = $4
		    AND s.scanned_at > now() - $5::interval
		)`,
		short, family, referrer, ipHash, scanDedupeWindow.String())
	return err
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// landingURL builds the forward target with decision 189's UTM mirror.
//
// The target is the operator's stated default — the website signup form,
// carrying UTM plus q=<short> so the signup can be attributed first-touch. The
// HOST is configurable (HQ_MARKETING_LANDING_BASE_URL); the target is not, and
// a different target is an operator fork this card deliberately did not take.
func (d Deps) landingURL(t codeTarget) string {
	landing := t.CampLanding
	if t.CodeLanding != nil && *t.CodeLanding != "" {
		landing = *t.CodeLanding // the code's own landing overrides the campaign's
	}
	path, ok := map[string]string{
		"signup":     "/signup",
		"menu":       "/menu",
		"offer":      "/offer",
		"directions": "/directions",
	}[landing]
	if !ok {
		path = "/signup"
	}

	q := url.Values{}
	q.Set("utm_source", "qr")
	q.Set("utm_medium", t.Channel)
	q.Set("utm_campaign", t.CampaignSlug)
	// utm_content is the ITEM slug, and it is OMITTED rather than sent empty
	// when the campaign is an "Any item" one — an empty utm_content would show
	// up in analytics as a real, nameless item.
	if t.ItemName != nil {
		if s := slugify(*t.ItemName); s != "" {
			q.Set("utm_content", s)
		}
	}
	q.Set("q", t.Short)
	return d.LandingBaseURL + path + "?" + q.Encode()
}
