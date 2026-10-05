package marketing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
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

	// Named link-preview UAs are not scans either — one message pasted into a
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
	// An agent that merely calls itself a bot but is not a NAMED preview
	// fetcher is a suspicion, not a certainty: it IS logged, with
	// ua_family='bot', which is what gives §4's comment a reachable value.
	if r := landingGet(mux, http.MethodGet, short, "SomeUnlistedBot/3.1 (+http://example.invalid)",
		"203.0.113.199:4444"); r.Code != http.StatusFound {
		t.Errorf("unlisted automation UA = %d, want 302", r.Code)
	}
	var botRows int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_scans WHERE short = $1 AND ua_family = 'bot'`, short).Scan(&botRows); err != nil {
		t.Fatalf("count bot rows: %v", err)
	}
	if botRows != 1 {
		t.Errorf("ua_family='bot' rows = %d, want 1 — an unlisted bot is logged as a suspicion, not dropped", botRows)
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

// ═════════════════════════════════════════════════════════════════════════════
// Card I4 `atomic-scan-dedupe` (ledger T-62 decision 195, migration 0087).
//
// The landing's 10-minute dedupe used to be a read-then-write
// (INSERT … WHERE NOT EXISTS) under READ COMMITTED, so hits that arrived
// together all read "no row yet" and all inserted. It is now a tumbling
// 10-minute bucket + a partial unique index + ON CONFLICT DO NOTHING.
//
// Every test below runs the PRODUCTION statement (insertScan) against the real
// database. None of them goes through a mock.
// ═════════════════════════════════════════════════════════════════════════════

// seedLandingCode creates one live campaign with one code through the real
// create-campaign route and returns the code's short.
func seedLandingCode(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	made := createCampaign(t, mux, mgr, map[string]any{
		"name": name, "offer_text": "$2 off", "face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	return made.Codes[0].Short
}

// scanRower is satisfied by both *pgxpool.Pool and pgx.Tx.
type scanRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// countScansFor counts the rows for one (short, ip_hash), through the pool or
// through an open transaction.
func countScansFor(t *testing.T, q scanRower, short, ipHash string) int {
	t.Helper()
	var n int
	if err := q.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_scans WHERE short = $1 AND ip_hash = $2`, short, ipHash).Scan(&n); err != nil {
		t.Fatalf("count scans for (%s, %s): %v", short, ipHash, err)
	}
	return n
}

// TestLandingDedupeIsAtomicUnderConcurrency — twelve hits for one
// (short, ip_hash) that arrive TOGETHER leave exactly one row. Five rounds.
//
// 🛑 The shape of this test is the test. The race only shows when the twelve
// statements start together: the connections are opened (and warmed) FIRST and
// the goroutines are parked on one barrier, then released at once. A version
// that spawns a goroutine, lets it connect and run, then spawns the next one
// staggers the statements enough that the old read-then-write passes it (spike
// correction 1: process-spawned clients gave 1/1/2/1/1 against broken code;
// barrier-aligned clients gave 12/12/12/11/12).
func TestLandingDedupeIsAtomicUnderConcurrency(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	short := seedLandingCode(t, pool, "Dedupe Race")

	const clients, rounds = 12, 5

	// A pool of its own, sized so twelve connections can be held at once — the
	// package pool's ceiling depends on the box's CPU count.
	cfg := pool.Config().Copy()
	cfg.MaxConns = clients
	racePool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open race pool: %v", err)
	}
	defer racePool.Close()

	// Open all twelve connections BEFORE anything races, and warm each one by
	// running the statement once inside a transaction that is rolled back — so
	// the first round is not staggered by twelve statement-prepare round trips.
	conns := make([]*pgxpool.Conn, clients)
	for i := range conns {
		c, err := racePool.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire connection %d: %v", i, err)
		}
		defer c.Release()
		tx, err := c.Begin(ctx)
		if err != nil {
			t.Fatalf("warm connection %d: begin: %v", i, err)
		}
		warm := "warm-" + randSuffix(t)
		if err := insertScan(ctx, tx, short, "ios", nil, &warm); err != nil {
			t.Fatalf("warm connection %d: %v", i, err)
		}
		if err := tx.Rollback(ctx); err != nil {
			t.Fatalf("warm connection %d: rollback: %v", i, err)
		}
		conns[i] = c
	}
	if n := countScans(t, pool, short); n != 0 {
		t.Fatalf("precondition: %d qr_scans rows before the first round, want 0", n)
	}

	counts := make([]string, 0, rounds)
	for round := 0; round < rounds; round++ {
		// Never race across a bucket edge: twelve hits either side of one would
		// correctly count twice and red this test for the wrong reason.
		var untilEdge float64
		if err := pool.QueryRow(ctx, `
			SELECT extract(epoch FROM date_bin('10 minutes', now(), timestamptz '2000-01-01 00:00:00+00')
			                          + interval '10 minutes' - now())`).Scan(&untilEdge); err != nil {
			t.Fatalf("read time to the next bucket edge: %v", err)
		}
		if untilEdge < 3 {
			time.Sleep(time.Duration((untilEdge + 0.5) * float64(time.Second)))
		}

		ipHash := fmt.Sprintf("race-%d-%s", round, randSuffix(t))
		barrier := make(chan struct{})
		errs := make([]error, clients)
		var ready, done sync.WaitGroup
		for i := range conns {
			ready.Add(1)
			done.Add(1)
			go func(i int) {
				defer done.Done()
				ready.Done()
				<-barrier // every client parks here …
				errs[i] = insertScan(ctx, conns[i], short, "ios", nil, &ipHash)
			}(i)
		}
		ready.Wait()
		close(barrier) // … and all twelve are released by this one line.
		done.Wait()

		for i, err := range errs {
			if err != nil {
				t.Errorf("round %d client %d: insertScan: %v", round, i, err)
			}
		}
		n := countScansFor(t, pool, short, ipHash)
		counts = append(counts, strconv.Itoa(n))
		if n != 1 {
			t.Errorf("round %d: %d qr_scans rows after %d simultaneous hits from one (short, ip_hash), want exactly 1",
				round, n, clients)
		}
	}
	t.Logf("rows per round after %d simultaneous hits: %s", clients, strings.Join(counts, " / "))
}

// TestLandingAnonymousScansNeverDedupe — a scan with no ip_hash cannot be told
// apart from another, so both count. This is the stated honest over-count and
// the bucket must not change it (NULLs are outside the partial index).
func TestLandingAnonymousScansNeverDedupe(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	short := seedLandingCode(t, pool, "Anonymous Scans")

	for i := 0; i < 2; i++ {
		if err := insertScan(ctx, pool, short, "ios", nil, nil); err != nil {
			t.Fatalf("anonymous insertScan %d: %v", i, err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM qr_scans WHERE short = $1 AND ip_hash IS NULL`, short).Scan(&n); err != nil {
		t.Fatalf("count anonymous scans: %v", err)
	}
	if n != 2 {
		t.Errorf("anonymous (NULL ip_hash) scans = %d after two hits, want 2 — they never dedupe", n)
	}
}

// TestLandingCountsAgainAfterWindow — a scan in a DIFFERENT 10-minute bucket
// counts again; a scan in the SAME bucket does not.
//
// 🛑 This asserts TUMBLING-BUCKET semantics (decision 195), not a sliding
// window. Leg (b) is the accepted trade-off, written down as an assertion: two
// taps that straddle a bucket edge count twice even though they are far less
// than ten minutes apart. Under the old sliding rule that pair counted once.
//
// The earlier scan of each pair is a fixture row with an explicit scanned_at
// (the clock cannot be moved); the later one is the production statement. All
// of it runs in ONE transaction so now() is frozen and the bucket edge cannot
// move between two statements.
func TestLandingCountsAgainAfterWindow(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	short := seedLandingCode(t, pool, "Return Visit")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// (a) a return visit eleven minutes later counts.
	later := "return-" + randSuffix(t)
	if _, err := tx.Exec(ctx,
		`INSERT INTO qr_scans (short, ip_hash, scanned_at) VALUES ($1, $2, now() - interval '11 minutes')`,
		short, later); err != nil {
		t.Fatalf("seed the earlier scan (a): %v", err)
	}
	if err := insertScan(ctx, tx, short, "ios", nil, &later); err != nil {
		t.Fatalf("insertScan (a): %v", err)
	}
	if n := countScansFor(t, tx, short, later); n != 2 {
		t.Errorf("(a) rows = %d for a return visit eleven minutes later, want 2", n)
	}

	// (b) the earlier scan is the LAST instant of the previous bucket — under
	// ten minutes ago, but a different bucket, so the new scan counts.
	edge := "edge-" + randSuffix(t)
	if _, err := tx.Exec(ctx, `
		INSERT INTO qr_scans (short, ip_hash, scanned_at)
		VALUES ($1, $2, date_bin('10 minutes', now(), timestamptz '2000-01-01 00:00:00+00') - interval '1 microsecond')`,
		short, edge); err != nil {
		t.Fatalf("seed the earlier scan (b): %v", err)
	}
	if err := insertScan(ctx, tx, short, "ios", nil, &edge); err != nil {
		t.Fatalf("insertScan (b): %v", err)
	}
	if n := countScansFor(t, tx, short, edge); n != 2 {
		t.Errorf("(b) rows = %d for two scans either side of a bucket edge, want 2 (tumbling bucket, not a sliding window)", n)
	}

	// (c) and a repeat inside the SAME bucket still does not.
	if err := insertScan(ctx, tx, short, "ios", nil, &edge); err != nil {
		t.Fatalf("insertScan (c): %v", err)
	}
	if n := countScansFor(t, tx, short, edge); n != 2 {
		t.Errorf("(c) rows = %d after a repeat inside the same bucket, want still 2", n)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Migration 0087 — the Down round-trip and the pre-existing-duplicates leg.
// Same rules as the zzz_ leg: start from HEAD, restore with db.Migrate (never a
// literal), derive the version from the filename.
// ─────────────────────────────────────────────────────────────────────────────

const dedupeMigrationSuffix = "_qr_scans_dedupe_bucket.sql"

func dedupeMigrationVersion(t *testing.T) int64 {
	t.Helper()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read %s: %v", migrationsDir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), dedupeMigrationSuffix) {
			continue
		}
		m := migrationNumRe.FindStringSubmatch(e.Name())
		if m == nil {
			t.Fatalf("migration %q does not start with a version number", e.Name())
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Fatalf("parse version from %q: %v", e.Name(), err)
		}
		return v
	}
	t.Fatalf("no *%s migration found in %s", dedupeMigrationSuffix, migrationsDir)
	return 0
}

// dedupeShape reads the catalog: is qr_scans.bucket a generated column, and
// what is the definition of qr_scans_dedupe_idx ("" when absent).
func dedupeShape(t *testing.T, pool *pgxpool.Pool) (bucketGenerated bool, indexDef string) {
	t.Helper()
	ctx := context.Background()
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM information_schema.columns
		                WHERE table_schema = current_schema() AND table_name = 'qr_scans'
		                  AND column_name = 'bucket' AND is_generated = 'ALWAYS')`).Scan(&bucketGenerated); err != nil {
		t.Fatalf("read qr_scans.bucket: %v", err)
	}
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT indexdef FROM pg_indexes
		                  WHERE schemaname = current_schema() AND tablename = 'qr_scans'
		                    AND indexname = 'qr_scans_dedupe_idx'), '')`).Scan(&indexDef); err != nil {
		t.Fatalf("read qr_scans_dedupe_idx: %v", err)
	}
	return bucketGenerated, indexDef
}

func assertDedupeShapePresent(t *testing.T, leg string, pool *pgxpool.Pool) {
	t.Helper()
	gen, def := dedupeShape(t, pool)
	if !gen {
		t.Errorf("%s: qr_scans.bucket is not a generated column", leg)
	}
	for _, want := range []string{"UNIQUE", "(short, ip_hash, bucket)", "WHERE (ip_hash IS NOT NULL)"} {
		if !strings.Contains(def, want) {
			t.Errorf("%s: qr_scans_dedupe_idx = %q, want it to contain %q", leg, def, want)
		}
	}
}

// TestMigration0087DedupeBucketDownAndUpRoundTrip proves 0087's Down: the
// index and the column go, the table and its rows stay, and re-applying brings
// both back.
func TestMigration0087DedupeBucketDownAndUpRoundTrip(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate up to HEAD before the down leg: %v", err)
	}
	// 🛑 Whatever happens below, the next test (and the next package under
	// -p 1) gets a fully migrated schema.
	t.Cleanup(func() {
		if err := db.Migrate(pool); err != nil {
			t.Errorf("restore to HEAD: %v", err)
		}
	})

	short := seedLandingCode(t, pool, "Round Trip")
	ip := "roundtrip-" + randSuffix(t)
	if err := insertScan(ctx, pool, short, "ios", nil, &ip); err != nil {
		t.Fatalf("seed a scan at HEAD: %v", err)
	}
	assertDedupeShapePresent(t, "at HEAD", pool)

	version := dedupeMigrationVersion(t)
	if err := db.MigrateTo(pool, version-1); err != nil {
		t.Fatalf("migrate down to %d: %v", version-1, err)
	}
	if gen, def := dedupeShape(t, pool); gen || def != "" {
		t.Errorf("after Down: bucket generated = %v, index = %q — want both gone", gen, def)
	}
	if n := countScans(t, pool, short); n != 1 {
		t.Errorf("after Down: %d qr_scans rows, want the 1 seeded row untouched", n)
	}

	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate back up to HEAD: %v", err)
	}
	assertDedupeShapePresent(t, "after re-apply", pool)
	if n := countScans(t, pool, short); n != 1 {
		t.Errorf("after re-apply: %d qr_scans rows, want 1", n)
	}
	t.Logf("migration %d Down/Up round-trip clean; schema restored to HEAD (not to a literal)", version)
}

// TestMigration0087CollapsesPreExistingDuplicates — the shipped statement let
// racing hits through, so a real table can already hold several rows with one
// (short, ip_hash, bucket). A unique index cannot be built over those; without
// a clean-up step the migration fails and the server does not boot.
//
// The rule the Up applies, asserted row by row:
//
//   - one row per (short, ip_hash, bucket) survives with its ip_hash — a row a
//     signup already claimed (subscriber_id set) if there is one, else the
//     earliest;
//   - a surplus row with no subscriber_id is deleted (it is a row the 10-minute
//     rule always meant to refuse: two rows in one bucket are under ten minutes
//     apart);
//   - a surplus row that DOES carry a subscriber_id is never deleted — its
//     ip_hash is blanked, so it stays in the count as an anonymous scan;
//   - NULL-ip rows, other buckets and other codes are untouched.
func TestMigration0087CollapsesPreExistingDuplicates(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate up to HEAD before the down leg: %v", err)
	}
	t.Cleanup(func() {
		err := db.Migrate(pool)
		if err == nil {
			return
		}
		t.Errorf("restore to HEAD: %v", err)
		// The Up refused this test's own duplicate rows. Do not leave every
		// later test and package staring at a half-migrated database for it:
		// clear the rows this test seeded and go up again.
		if _, terr := pool.Exec(context.Background(), `TRUNCATE qr_scans`); terr != nil {
			t.Errorf("clear seeded scans after a failed Up: %v", terr)
		}
		if err := db.Migrate(pool); err != nil {
			t.Errorf("restore to HEAD after clearing the seeded scans: %v", err)
		}
	})

	seedErasureCampaignAndCode(t, pool, "DEDUPA")
	seedErasureCampaignAndCode(t, pool, "DEDUPB")
	sub1 := seedErasureSubscriber(t, pool, nil)
	sub2 := seedErasureSubscriber(t, pool, nil)
	sub3 := seedErasureSubscriber(t, pool, nil)

	version := dedupeMigrationVersion(t)
	if err := db.MigrateTo(pool, version-1); err != nil {
		t.Fatalf("migrate down to %d: %v", version-1, err)
	}

	// The table as the shipped statement could have left it. `tag` rides in
	// the otherwise-unused referrer column so each row can be named afterwards.
	type row struct {
		tag, short string
		ip, sub    *string
		at         string
	}
	s := func(v string) *string { return &v }
	seed := []row{
		// G1 — three unclaimed rows in the 12:00 bucket: the earliest survives.
		{"g1-first", "DEDUPA", s("h1"), nil, "2026-09-01 12:01:00+00"},
		{"g1-dup-a", "DEDUPA", s("h1"), nil, "2026-09-01 12:03:00+00"},
		{"g1-dup-b", "DEDUPA", s("h1"), nil, "2026-09-01 12:09:59+00"},
		// Same code + ip, the NEXT bucket: a different scan, untouched.
		{"g1-next-bucket", "DEDUPA", s("h1"), nil, "2026-09-01 12:10:00+00"},
		// G2 — an unclaimed row then a claimed one: the claimed row survives.
		{"g2-unclaimed", "DEDUPA", s("h2"), nil, "2026-09-01 12:02:00+00"},
		{"g2-claimed", "DEDUPA", s("h2"), &sub1, "2026-09-01 12:05:00+00"},
		// G3 — two claimed rows and an unclaimed one: the earlier claimed row
		// keeps its ip_hash, the later claimed row is kept but blanked, the
		// unclaimed one goes.
		{"g3-claimed-first", "DEDUPA", s("h3"), &sub2, "2026-09-01 12:01:00+00"},
		{"g3-claimed-second", "DEDUPA", s("h3"), &sub3, "2026-09-01 12:04:00+00"},
		{"g3-unclaimed", "DEDUPA", s("h3"), nil, "2026-09-01 12:06:00+00"},
		// Same ip and bucket on ANOTHER code: untouched.
		{"other-code", "DEDUPB", s("h1"), nil, "2026-09-01 12:01:00+00"},
		// Anonymous scans never dedupe: both untouched.
		{"anon-a", "DEDUPA", nil, nil, "2026-09-01 12:01:00+00"},
		{"anon-b", "DEDUPA", nil, nil, "2026-09-01 12:01:00+00"},
	}
	for _, r := range seed {
		if _, err := pool.Exec(ctx, `
			INSERT INTO qr_scans (short, ip_hash, subscriber_id, scanned_at, referrer)
			VALUES ($1, $2, $3, $4::timestamptz, $5)`, r.short, r.ip, r.sub, r.at, r.tag); err != nil {
			t.Fatalf("seed %s at version %d: %v", r.tag, version-1, err)
		}
	}

	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migration %d Up over a table that already holds duplicates: %v", version, err)
	}
	assertDedupeShapePresent(t, "after Up over duplicates", pool)

	// tag → ip_hash ("<null>" for NULL) for every surviving row.
	got := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT referrer, COALESCE(ip_hash, '<null>') FROM qr_scans`)
	if err != nil {
		t.Fatalf("read surviving rows: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tag, ip string
		if err := rows.Scan(&tag, &ip); err != nil {
			t.Fatalf("scan surviving row: %v", err)
		}
		got[tag] = ip
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate surviving rows: %v", err)
	}

	want := map[string]string{
		"g1-first":          "h1",
		"g1-next-bucket":    "h1",
		"g2-claimed":        "h2",
		"g3-claimed-first":  "h3",
		"g3-claimed-second": "<null>",
		"other-code":        "h1",
		"anon-a":            "<null>",
		"anon-b":            "<null>",
	}
	for tag, ip := range want {
		g, ok := got[tag]
		switch {
		case !ok:
			t.Errorf("row %s was deleted, want it kept with ip_hash %s", tag, ip)
		case g != ip:
			t.Errorf("row %s ip_hash = %s, want %s", tag, g, ip)
		}
	}
	for tag := range got {
		if _, ok := want[tag]; !ok {
			t.Errorf("row %s survived, want it deleted as a same-bucket duplicate", tag)
		}
	}

	// No subscriber link was lost: all three claims are still in the table.
	for name, sub := range map[string]string{"sub1": sub1, "sub2": sub2, "sub3": sub3} {
		if n := countWhere(t, pool, "qr_scans", "subscriber_id = $1", sub); n != 1 {
			t.Errorf("%s is named by %d qr_scans rows after the Up, want 1", name, n)
		}
	}
}
