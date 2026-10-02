package marketing

// subscribers_test.go — card H5 (`subscribers-tab`), run 20261002.
//
// RED-FIRST (greenfield): every test here was written and RUN against the
// pre-change tree — migration 0085 absent, subscribers.go absent, the five
// routes unregistered — where the package does not compile and, once it does,
// the tables do not exist. Evidence:
// .night-crew/runs/2026-10-02-autonomous/logs/h5/rf-go-red.log and the
// ## Red-first section of merge-intents/h5-subscribers-tab.md.
//
// The two done_when rows are TestFluentFormsImportIsIdempotent and
// TestSubscriberSourceShortSetsCampaignAttribution. The rest guard the card's
// two absolute prohibitions and the privacy contract.

import (
	"context"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/marketing/sources"
)

// ffFixturePath is the ONE Fluent Forms input any automated gate here reads.
// 🛑 No test in this tree connects to the live website database.
const ffFixturePath = "sources/testdata/fluentforms_submissions.json"

// setupSubsTestDB is this card's own reset, deliberately NOT an edit to Card
// H1's setupTestDB: 0085's two tables sit downstream of qr_codes, so adding
// them to a shared helper would be a conflict for no gain.
func setupSubsTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and the local fallback is unreachable)")
	}
	if _, err := testPool.Exec(t.Context(),
		`TRUNCATE subscriber_events, subscribers, qr_scans, qr_codes, campaigns_admin RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("setupSubsTestDB truncate: %v", err)
	}
	return testPool
}

func ffImporter(t *testing.T) sources.Importer {
	t.Helper()
	raw, err := os.ReadFile(ffFixturePath)
	if err != nil {
		t.Fatalf("read committed fixture %s: %v", ffFixturePath, err)
	}
	imp, err := sources.NewFluentFormsFromJSON(raw)
	if err != nil {
		t.Fatalf("NewFluentFormsFromJSON: %v", err)
	}
	return imp
}

func countRows(t *testing.T, pool *pgxpool.Pool, q string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatalf("count (%s): %v", q, err)
	}
	return n
}

// ═══════════════════════════════════════════════════════════════════════════
// done_when row 1
// ═══════════════════════════════════════════════════════════════════════════

// TestFluentFormsImportIsIdempotent runs the SAME committed fixture twice and
// asserts the second run creates nothing and appends no second `signed_up`.
//
// It also pins the fixture's two designed collisions: row 1180 is row 1184's
// human in a different spelling ("17735554821" vs "(773) 555-4821"), so the
// E.164 dedupe must collapse them into ONE subscriber on the FIRST pass too —
// five fixture rows, four subscribers.
func TestFluentFormsImportIsIdempotent(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()

	first, err := ImportSubscribers(ctx, pool, ffImporter(t))
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if first.Read != 5 {
		t.Fatalf("first.Read = %d, want 5 (the committed fixture's row count)", first.Read)
	}
	// 1181 has a 7-digit phone AND an email, so it still lands; 1180 merges
	// onto 1184 by phone. 5 read → 4 created, 1 updated.
	if first.Created != 4 || first.Updated != 1 {
		t.Fatalf("first import = %d created / %d updated, want 4 / 1 (1180 dedupes onto 1184 by E.164)", first.Created, first.Updated)
	}
	subs := countRows(t, pool, `SELECT count(*) FROM subscribers`)
	events := countRows(t, pool, `SELECT count(*) FROM subscriber_events WHERE kind='signed_up'`)
	if subs != 4 {
		t.Fatalf("after one import: %d subscribers, want 4", subs)
	}
	if events != 4 {
		t.Fatalf("after one import: %d signed_up events, want 4 (one per person, not one per row)", events)
	}

	second, err := ImportSubscribers(ctx, pool, ffImporter(t))
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if second.Created != 0 {
		t.Fatalf("second import created %d rows, want 0 — the (source, external_ref) upsert is not idempotent", second.Created)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM subscribers`); n != subs {
		t.Fatalf("after two imports: %d subscribers, want %d", n, subs)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM subscriber_events WHERE kind='signed_up'`); n != events {
		t.Fatalf("after two imports: %d signed_up events, want %d — a re-run must not log itself onto the timeline", n, events)
	}

	// The field map is the goal ledger's binding correction, so assert it on
	// the row the fixture built for it: names.first_name → display_name,
	// input_text → E.164 phone, checkbox[] → the two consent flags.
	var name, phone, evidence string
	var sms, emailOK bool
	if err := pool.QueryRow(ctx, `
		SELECT display_name, phone_e164, consent_evidence, sms_consent, email_consent
		  FROM subscribers WHERE source='web_form' AND external_ref='1184'`).
		Scan(&name, &phone, &evidence, &sms, &emailOK); err != nil {
		t.Fatalf("read the 1184 row: %v", err)
	}
	if name != "Dana" {
		t.Errorf("display_name = %q, want %q (names.first_name)", name, "Dana")
	}
	if phone != "+17735554821" {
		t.Errorf("phone_e164 = %q, want %q (input_text, E.164-normalized)", phone, "+17735554821")
	}
	if !sms || !emailOK {
		t.Errorf("consent = sms:%v email:%v, want both true (checkbox ['Phone','Email'])", sms, emailOK)
	}
	if !strings.Contains(evidence, "Phone") || !strings.Contains(evidence, "Email") {
		t.Errorf("consent_evidence = %q, want it to name the ticked boxes", evidence)
	}

	// Row 1183 ticked Email only → the list's consent state must be
	// "email_only", which is what the designed chip filters on.
	var sms2, email2 bool
	if err := pool.QueryRow(ctx,
		`SELECT sms_consent, email_consent FROM subscribers WHERE external_ref='1183'`).Scan(&sms2, &email2); err != nil {
		t.Fatalf("read the 1183 row: %v", err)
	}
	if sms2 || !email2 {
		t.Errorf("1183 consent = sms:%v email:%v, want false/true", sms2, email2)
	}
	if got := consentState(sms2, email2, nil); got != ConsentEmailOnly {
		t.Errorf("consentState(false,true,nil) = %q, want %q", got, ConsentEmailOnly)
	}

	// Row 1181's 7-digit phone normalizes to nothing → phone_e164 NULL, and
	// the masked cell is null rather than a row of bullets.
	var p *string
	if err := pool.QueryRow(ctx,
		`SELECT phone_e164 FROM subscribers WHERE external_ref='1181'`).Scan(&p); err != nil {
		t.Fatalf("read the 1181 row: %v", err)
	}
	if p != nil {
		t.Errorf("1181 phone_e164 = %q, want NULL (555-4821 is not a 10-digit number)", *p)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// done_when row 2
// ═══════════════════════════════════════════════════════════════════════════

// TestSubscriberSourceShortSetsCampaignAttribution asserts the first-touch
// join: a subscriber whose `source_short` resolves carries that code's
// CAMPAIGN NAME on the §5 list row, and the earliest unclaimed qr_scans row for
// that short is bound to them.
//
// The fixture's row 1182 is the one carrying `source: "K7MNPQ"` — the field
// spike 01 found ABSENT on the newest live submission, which is exactly why the
// column is nullable and why this test seeds the code it points at.
func TestSubscriberSourceShortSetsCampaignAttribution(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	// A campaign with a code whose short is the fixture's.
	created := createCampaign(t, mux, mgr, map[string]any{
		"name": "Wing Wednesday", "offer_text": "$2 off any 6pc wings",
		"face_value_cents": 200, "runs_days": 14,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	if _, err := pool.Exec(ctx, `UPDATE qr_codes SET short='K7MNPQ' WHERE id = $1`, created.Codes[0].ID); err != nil {
		t.Fatalf("re-point the minted code's short: %v", err)
	}
	// Two scans of that short, the EARLIER one unclaimed: first-touch must
	// claim the earlier, not the later.
	early := time.Date(2026, 9, 26, 18, 0, 0, 0, time.UTC)
	late := time.Date(2026, 9, 26, 19, 30, 0, 0, time.UTC)
	for _, at := range []time.Time{late, early} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO qr_scans (short, scanned_at, ua_family) VALUES ('K7MNPQ', $1, 'ios')`, at); err != nil {
			t.Fatalf("seed qr_scans: %v", err)
		}
	}

	res, err := ImportSubscribers(ctx, pool, ffImporter(t))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if res.ScansBound != 1 {
		t.Fatalf("ScansBound = %d, want 1 (first-touch claims exactly one scan)", res.ScansBound)
	}

	var short, subID string
	if err := pool.QueryRow(ctx,
		`SELECT source_short, id::text FROM subscribers WHERE external_ref='1182'`).Scan(&short, &subID); err != nil {
		t.Fatalf("read the 1182 row: %v", err)
	}
	if short != "K7MNPQ" {
		t.Fatalf("source_short = %q, want K7MNPQ", short)
	}
	var boundAt time.Time
	if err := pool.QueryRow(ctx,
		`SELECT scanned_at FROM qr_scans WHERE subscriber_id = $1`, subID).Scan(&boundAt); err != nil {
		t.Fatalf("read the bound scan: %v", err)
	}
	if !boundAt.UTC().Equal(early) {
		t.Errorf("bound scan at %s, want the EARLIEST (%s) — decision 189 is first-touch", boundAt.UTC(), early)
	}

	// The §5 list row must carry the campaign name, which is the whole point
	// of the join.
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /subscribers = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var list subscriberListResponse
	decode(t, rec, &list)
	var found *subscriberRowDTO
	for i := range list.Rows {
		if list.Rows[i].ID == subID {
			found = &list.Rows[i]
		}
	}
	if found == nil {
		t.Fatalf("the attributed subscriber is missing from the list (%d rows)", len(list.Rows))
	}
	if found.CampaignName == nil || *found.CampaignName != "Wing Wednesday" {
		t.Errorf("campaign_name = %v, want %q", found.CampaignName, "Wing Wednesday")
	}
	if found.SourceShort == nil || *found.SourceShort != "K7MNPQ" {
		t.Errorf("source_short = %v, want K7MNPQ", found.SourceShort)
	}

	// A subscriber whose short does NOT resolve keeps the person and drops the
	// attribution — a printed code that no longer exists is the customer's
	// reality, not an import failure.
	if _, err := pool.Exec(ctx, `TRUNCATE subscriber_events, subscribers CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	stale, err := sources.NewQRSignup("ZZZZZZ", "Stale", "773-555-7777", "", true, false, time.Now().UTC(), "stale-1")
	if err != nil {
		t.Fatalf("NewQRSignup: %v", err)
	}
	r2, err := ImportSubscribers(ctx, pool, stale)
	if err != nil {
		t.Fatalf("stale-short import: %v", err)
	}
	if r2.Created != 1 {
		t.Fatalf("stale-short import created %d, want 1 — an unknown short must not cost the person", r2.Created)
	}
	var ss *string
	if err := pool.QueryRow(ctx, `SELECT source_short FROM subscribers WHERE external_ref='stale-1'`).Scan(&ss); err != nil {
		t.Fatalf("read the stale row: %v", err)
	}
	if ss != nil {
		t.Errorf("source_short = %q, want NULL for a short with no qr_codes row", *ss)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PROHIBITION 1 — nothing sends
// ═══════════════════════════════════════════════════════════════════════════

// senderImports is the denylist. A path matching any of these in
// internal/marketing or internal/marketing/sources means something in the
// subscriber surface acquired the ability to send, which is Activity E's and is
// a PARK on this card.
var senderImports = []string{
	"net/smtp", "twilio", "sendgrid", "mailgun", "postmark", "resend",
	"service/sns", "service/ses", "mailersend", "plivo", "messagebird",
}

// TestNothingInThisPackageSends parses the import graph of internal/marketing
// and internal/marketing/sources and asserts no sender is reachable from
// either, then asserts the subscriber FILES contain no outbound HTTP call at
// all.
//
// The second half is scoped to the subscriber files on purpose:
// projection.go legitimately speaks HTTP to Supabase (decision 187), so a
// package-wide ban would be false. What must hold is that NOTHING on the
// subscriber path — the handlers or any adapter — reaches the network except
// the attended, env-gated MySQL read.
func TestNothingInThisPackageSends(t *testing.T) {
	fset := token.NewFileSet()
	for _, dir := range []string{".", "sources"} {
		pkgs, err := parser.ParseDir(fset, dir, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", dir, err)
		}
		for _, pkg := range pkgs {
			for name, f := range pkg.Files {
				for _, imp := range f.Imports {
					path := strings.Trim(imp.Path.Value, `"`)
					for _, bad := range senderImports {
						if strings.Contains(strings.ToLower(path), bad) {
							t.Errorf("%s imports %q — SENDING is Activity E's and is a PARK on card H5", name, path)
						}
					}
				}
			}
		}
	}

	// No outbound HTTP on the subscriber path.
	outbound := regexp.MustCompile(`http\.(Post|Get|Head|PostForm|NewRequest)\b|http\.Client\b|\bDefaultClient\b`)
	files := []string{"subscribers.go"}
	entries, err := os.ReadDir("sources")
	if err != nil {
		t.Fatalf("read sources dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			files = append(files, filepath.Join("sources", e.Name()))
		}
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if m := outbound.FindString(string(src)); m != "" {
			t.Errorf("%s contains an outbound HTTP call (%q) — the subscriber path sends nothing and fetches nothing", f, m)
		}
	}
}

// TestResendRecords202AndAppendsNoCodeSent is the behavioral half of the same
// promise: 202, exactly one new `resend_requested`, and ZERO `code_sent` —
// because `code_sent` is the send's event and the Stats funnel counts it.
func TestResendRecords202AndAppendsNoCodeSent(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	if _, err := ImportSubscribers(ctx, pool, ffImporter(t)); err != nil {
		t.Fatalf("import: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM subscribers WHERE external_ref='1184'`).Scan(&id); err != nil {
		t.Fatalf("pick a subscriber: %v", err)
	}

	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPost, "/api/v1/marketing/subscribers/"+id+"/resend", nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("POST resend = %d, want 202\nbody: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	decode(t, rec, &body)
	if body["sent"] != false {
		t.Errorf("resend body sent = %v, want false — this route sends nothing", body["sent"])
	}
	if n := countRows(t, pool, `SELECT count(*) FROM subscriber_events WHERE subscriber_id=$1 AND kind='resend_requested'`, id); n != 1 {
		t.Errorf("resend_requested events = %d, want 1", n)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM subscriber_events WHERE kind='code_sent'`); n != 0 {
		t.Fatalf("code_sent events = %d, want 0 — resend must NOT write the send's event", n)
	}

	// The detail sheet reports the request without claiming a send.
	rec = do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET detail = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var detail subscriberDetailDTO
	decode(t, rec, &detail)
	if detail.IdentityCode.Status != "not_sent" {
		t.Errorf("identity_code.status = %q, want not_sent — a request is not a send", detail.IdentityCode.Status)
	}
	if detail.IdentityCode.RequestedAt == nil {
		t.Error("identity_code.requested_at is nil, want the resend request's timestamp")
	}
	var kinds []string
	for _, e := range detail.Events {
		kinds = append(kinds, e.Kind)
	}
	if !containsString(kinds, "resend_requested") || !containsString(kinds, "signed_up") {
		t.Errorf("timeline kinds = %v, want signed_up + resend_requested", kinds)
	}

	// A resend for an unknown id is a 404, not a silently-recorded event.
	rec = do(t, mux, userCtx(mgr, "manager"), http.MethodPost,
		"/api/v1/marketing/subscribers/00000000-0000-0000-0000-000000000000/resend", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("resend unknown id = %d, want 404", rec.Code)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PROHIBITION 2 — the live Fluent Forms database is never touched by the night
// ═══════════════════════════════════════════════════════════════════════════

// TestWebFormImportRefusesWhenUnconfigured asserts the attended-only gate: with
// FF_DB_* unset the route answers 503 ff_not_configured and opens NO
// connection. This is the state of every gate, every worktree and prod.
func TestWebFormImportRefusesWhenUnconfigured(t *testing.T) {
	pool := setupSubsTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	for _, k := range []string{sources.EnvFFHost, sources.EnvFFUser, sources.EnvFFPassword, sources.EnvFFName, sources.EnvFFPrefix, SubsFixtureEnv} {
		if v := os.Getenv(k); v != "" {
			t.Fatalf("%s is set (%q) — the night must never run with live Fluent Forms credentials", k, v)
		}
	}
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPost, "/api/v1/marketing/subscribers/import/web-form", nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("POST import/web-form = %d, want 503\nbody: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ff_not_configured") {
		t.Errorf("body = %s, want error ff_not_configured", rec.Body.String())
	}
	if _, err := sources.NewFluentForms(); err == nil {
		t.Error("NewFluentForms succeeded with no credentials — it must refuse")
	}
}

// TestWebFormImportReadsTheCommittedFixture drives the REAL HTTP route with
// HQ_FF_FIXTURE_PATH pointed at the committed fixture, so the attended import's
// own code path is exercised end to end with no network at all.
func TestWebFormImportReadsTheCommittedFixture(t *testing.T) {
	pool := setupSubsTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	abs, err := filepath.Abs(ffFixturePath)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	t.Setenv(SubsFixtureEnv, abs)
	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodPost, "/api/v1/marketing/subscribers/import/web-form", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST import/web-form = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var res importResultDTO
	decode(t, rec, &res)
	if res.Source != sources.SourceWebForm || res.Created != 4 {
		t.Fatalf("import result = %+v, want source web_form / 4 created", res)
	}
	// An import SUMMARY is a response body and the masking contract covers it.
	assertNoContactValuesInBody(t, rec.Body.Bytes())
}

// ═══════════════════════════════════════════════════════════════════════════
// the privacy contract — masking is server-side
// ═══════════════════════════════════════════════════════════════════════════

// fullPhonePattern matches any spelling of the fixture's numbers that would
// mean a full phone had leaked: ten consecutive digits, or the E.164 form.
var fullPhonePattern = regexp.MustCompile(`\+1\d{10}|\b\d{10}\b|\b\d{3}[-.\s]\d{3}[-.\s]\d{4}\b`)

func assertNoContactValuesInBody(t *testing.T, body []byte) {
	t.Helper()
	s := string(body)
	if m := fullPhonePattern.FindString(s); m != "" {
		t.Errorf("a response body carries a full phone number (%q) — masking is server-side", m)
	}
	for _, leak := range []string{"dana.r@gmail.com", "marcus@yahoo.com", "priya.k@outlook.com", "sam@proton.me"} {
		if strings.Contains(s, leak) {
			t.Errorf("a response body carries the full email %q — masking is server-side", leak)
		}
	}
	for _, key := range []string{`"phone_e164"`, `"email":`} {
		if strings.Contains(s, key) {
			t.Errorf("a response body carries the key %s — the wire shape is phone_last4 + email_masked only", key)
		}
	}
}

// TestNoResponseCarriesAFullPhoneOrEmail is the gate on §5's masking promise,
// asserted on the RENDERED JSON of every subscriber read.
func TestNoResponseCarriesAFullPhoneOrEmail(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	if _, err := ImportSubscribers(ctx, pool, ffImporter(t)); err != nil {
		t.Fatalf("import: %v", err)
	}

	rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /subscribers = %d: %s", rec.Code, rec.Body.String())
	}
	assertNoContactValuesInBody(t, rec.Body.Bytes())
	var list subscriberListResponse
	decode(t, rec, &list)
	if list.Total != 4 {
		t.Fatalf("total = %d, want 4", list.Total)
	}
	var masked int
	for _, r := range list.Rows {
		if r.PhoneLast4 != nil {
			if len(*r.PhoneLast4) != 4 {
				t.Errorf("phone_last4 = %q, want exactly four digits", *r.PhoneLast4)
			}
			masked++
		}
		if r.EmailMasked != nil && !strings.Contains(*r.EmailMasked, "•") {
			t.Errorf("email_masked = %q, want it masked", *r.EmailMasked)
		}
		rec2 := do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers/"+r.ID, nil)
		if rec2.Code != http.StatusOK {
			t.Fatalf("GET /subscribers/%s = %d: %s", r.ID, rec2.Code, rec2.Body.String())
		}
		assertNoContactValuesInBody(t, rec2.Body.Bytes())
	}
	if masked == 0 {
		t.Error("no row carried a phone_last4 — the fixture should produce three")
	}
}

// TestE164NormalizeAndMaskHoldOnTheSpikesSet re-asserts spike 02's enumerated
// set in Go, so the build-fact the card rests on is pinned by the suite and not
// only by a shell script that ran once.
func TestE164NormalizeAndMaskHoldOnTheSpikesSet(t *testing.T) {
	cases := map[string]string{
		"(773) 555-4821":  "+17735554821",
		"773-555-4821":    "+17735554821",
		"7735554821":      "+17735554821",
		"+1 773 555 4821": "+17735554821",
		"17735554821":     "+17735554821",
		"555-4821":        "",
		"":                "",
	}
	seen := map[string]bool{}
	for raw, want := range cases {
		got := sources.NormalizeE164(raw)
		if got != want {
			t.Errorf("NormalizeE164(%q) = %q, want %q", raw, got, want)
		}
		if got != "" {
			seen[got] = true
		}
	}
	if len(seen) != 1 {
		t.Errorf("the five spellings produced %d distinct E.164 values, want 1", len(seen))
	}
	if got := sources.MaskPhone("+17735554821"); got != "•••• 4821" {
		t.Errorf("MaskPhone = %q, want %q", got, "•••• 4821")
	}
	if got := sources.MaskPhone(""); got != "" {
		t.Errorf("MaskPhone(\"\") = %q, want empty", got)
	}
	for in, want := range map[string]string{
		"dana.r@gmail.com": "d•••@gmail.com",
		"a@b.com":          "•••@b.com",
		"":                 "",
		"nonsense":         "•••",
	} {
		if got := sources.MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// the Locked state, the filter set, and the CSV adapter's refusal
// ═══════════════════════════════════════════════════════════════════════════

// TestTeamMemberGets403ManagersOnlyOnSubscribers covers the whole surface —
// both reads, the write and both imports. The designed Locked state covers the
// TAB, not just its buttons.
func TestTeamMemberGets403ManagersOnlyOnSubscribers(t *testing.T) {
	pool := setupSubsTestDB(t)
	tm := seedUser(t, pool, "team_member")
	mux := mountedMux(testDeps(pool))
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/marketing/subscribers"},
		{http.MethodGet, "/api/v1/marketing/subscribers/00000000-0000-0000-0000-000000000000"},
		{http.MethodPost, "/api/v1/marketing/subscribers/00000000-0000-0000-0000-000000000000/resend"},
		{http.MethodPost, "/api/v1/marketing/subscribers/import/toast-guests"},
		{http.MethodPost, "/api/v1/marketing/subscribers/import/web-form"},
	} {
		rec := do(t, mux, userCtx(tm, "team_member"), c.method, c.path, nil)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s %s = %d, want 403", c.method, c.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"managers_only"`) {
			t.Errorf("%s %s body = %s, want managers_only", c.method, c.path, rec.Body.String())
		}
	}
}

// TestToastGuestCSVImportsAndFiltersByConsent drives the real upload route and
// then the four filter chips, pinning the filter semantics this card decided.
func TestToastGuestCSVImportsAndFiltersByConsent(t *testing.T) {
	pool := setupSubsTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))

	guestCSV := "Guest Id,Name,Phone Number,Email,SMS Opt In,Email Opt In,Opted Out,Created Date\n" +
		"g-1,Rosa,(773) 555-1000,rosa@example.com,Yes,Yes,No,2026-09-20 10:00:00\n" +
		"g-2,Theo,773.555.1001,theo@example.com,No,Yes,No,2026-09-21 10:00:00\n" +
		"g-3,Nia,7735551002,nia@example.com,Yes,No,Yes,2026-09-22 10:00:00\n" +
		"g-4,Cal,,cal@example.com,No,No,No,2026-09-23 10:00:00\n"
	req := do(t, mux, userCtx(mgr, "manager"), http.MethodPost,
		"/api/v1/marketing/subscribers/import/toast-guests", nil)
	_ = req // the no-body case first: an empty upload is a 400, not an import of zero.
	if req.Code != http.StatusBadRequest {
		t.Errorf("empty upload = %d, want 400", req.Code)
	}

	rec := postCSV(t, mux, userCtx(mgr, "manager"), "/api/v1/marketing/subscribers/import/toast-guests", guestCSV)
	if rec.Code != http.StatusOK {
		t.Fatalf("CSV import = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
	}
	var res importResultDTO
	decode(t, rec, &res)
	if res.Created != 4 {
		t.Fatalf("created = %d, want 4\nbody: %s", res.Created, rec.Body.String())
	}

	want := map[string]int{FilterAll: 4, FilterSMS: 1, FilterEmailOnly: 1, FilterOptedOut: 1}
	for filter, n := range want {
		rec := do(t, mux, userCtx(mgr, "manager"), http.MethodGet,
			"/api/v1/marketing/subscribers?filter="+filter, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("filter=%s = %d: %s", filter, rec.Code, rec.Body.String())
		}
		var list subscriberListResponse
		decode(t, rec, &list)
		if len(list.Rows) != n {
			t.Errorf("filter=%s returned %d rows, want %d", filter, len(list.Rows), n)
		}
		if filter == FilterOptedOut && len(list.Rows) == 1 && list.Rows[0].Consent != ConsentStop {
			t.Errorf("opted-out row consent = %q, want %q", list.Rows[0].Consent, ConsentStop)
		}
	}

	// q= searches the masked cell's last four, which is what a manager reads
	// off the screen and types.
	rec = do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers?q=1001", nil)
	var list subscriberListResponse
	decode(t, rec, &list)
	if len(list.Rows) != 1 || list.Rows[0].PhoneLast4 == nil || *list.Rows[0].PhoneLast4 != "1001" {
		t.Errorf("q=1001 returned %d rows (%+v), want the one ending 1001", len(list.Rows), list.Rows)
	}
	// source= filters on §4's enum.
	rec = do(t, mux, userCtx(mgr, "manager"), http.MethodGet, "/api/v1/marketing/subscribers?source=web_form", nil)
	decode(t, rec, &list)
	if len(list.Rows) != 0 {
		t.Errorf("source=web_form returned %d rows, want 0 — everything here came from Toast", len(list.Rows))
	}

	// A wrong file fails at the header rather than importing zero and
	// reporting success.
	rec = postCSV(t, mux, userCtx(mgr, "manager"), "/api/v1/marketing/subscribers/import/toast-guests",
		"Item,Qty,Net Price\nWings,2,12.00\n")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "no_guest_columns") {
		t.Errorf("wrong-file upload = %d %s, want 400 no_guest_columns", rec.Code, rec.Body.String())
	}
}

// postCSV sends a text/csv body. The shared `do` helper JSON-encodes its
// payload, which is the wrong content type for an upload route.
func postCSV(t *testing.T, mux *chi.Mux, ctx context.Context, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)).WithContext(ctx)
	req.Header.Set("Content-Type", "text/csv")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// ═══════════════════════════════════════════════════════════════════════════
// FIX ROUND (G6 findings F2, F4, and the E.164 low) — run 20261002
// ═══════════════════════════════════════════════════════════════════════════

// TestToastGuestIDAliasDoesNotMergeTwoPeople is G6 finding F2, and it is a
// CONSENT bug, not a data-hygiene one.
//
// `id` is a generic column name. A Toast export whose `id` is a stable guest
// handle and a LATER export whose `id` is a plain row number both land on
// external_ref "1" — so findSubscriber matches, mergeSubscriber runs, and
// `sms_consent = sms_consent OR true` lands Bob's opt-in on ALICE, who never
// consented. Exactly the two-export scenario G6 named.
func TestToastGuestIDAliasDoesNotMergeTwoPeople(t *testing.T) {
	pool := setupSubsTestDB(t)
	mgr := seedUser(t, pool, "manager")
	mux := mountedMux(testDeps(pool))
	ctx := userCtx(mgr, "manager")

	exportA := "id,name,phone,sms opt in\n1,Alice,(773) 570-0001,No\n"
	exportB := "id,name,phone,sms opt in\n1,Bob,(773) 570-0002,Yes\n"

	if rec := postCSV(t, mux, ctx, "/api/v1/marketing/subscribers/import/toast-guests", exportA); rec.Code != http.StatusOK {
		t.Fatalf("export A = %d: %s", rec.Code, rec.Body.String())
	}
	if rec := postCSV(t, mux, ctx, "/api/v1/marketing/subscribers/import/toast-guests", exportB); rec.Code != http.StatusOK {
		t.Fatalf("export B = %d: %s", rec.Code, rec.Body.String())
	}

	if n := countRows(t, pool, `SELECT count(*) FROM subscribers`); n != 2 {
		t.Fatalf("%d subscribers after two exports, want 2 — Alice and Bob are different people", n)
	}
	var aliceSMS bool
	var aliceName string
	if err := pool.QueryRow(context.Background(),
		`SELECT display_name, sms_consent FROM subscribers WHERE phone_e164 = '+17735700001'`).
		Scan(&aliceName, &aliceSMS); err != nil {
		t.Fatalf("read Alice: %v", err)
	}
	if aliceName != "Alice" {
		t.Errorf("display_name = %q, want Alice — Bob's row overwrote her", aliceName)
	}
	if aliceSMS {
		t.Error("🛑 Alice has sms_consent=true. She answered No. A generic `id` column merged Bob's opt-in onto her.")
	}
}

// TestReImportConvergesCorrectedFieldsAndCountsHonestly is G6 finding F4.
//
// Two halves, both of which were false before the fix:
//
//  1. CONVERGENCE — a display_name or email corrected upstream must land.
//     mergeSubscriber was all COALESCE(existing, new), so "Dana" never became
//     "Dana Reyes".
//  2. AN HONEST COUNT — re-importing an UNCHANGED file must not report
//     `updated: N`. It claimed an update that did not happen.
func TestReImportConvergesCorrectedFieldsAndCountsHonestly(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()

	const one = `[{"id":9001,"form_id":2,"created_at":"2026-09-20 10:00:00",
	  "response":"{\"names\":{\"first_name\":\"Dana\"},\"email\":\"dana@example.com\",\"input_text\":\"(773) 571-0001\",\"checkbox\":[\"Email\"]}"}]`
	// Same submission id, a corrected NAME and EMAIL, and an ADDED consent.
	const corrected = `[{"id":9001,"form_id":2,"created_at":"2026-09-20 10:00:00",
	  "response":"{\"names\":{\"first_name\":\"Dana Reyes\"},\"email\":\"dana.reyes@example.com\",\"input_text\":\"(773) 571-0001\",\"checkbox\":[\"Email\",\"Phone\"]}"}]`

	imp := func(raw string) importResultDTO {
		t.Helper()
		ff, err := sources.NewFluentFormsFromJSON([]byte(raw))
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		res, err := ImportSubscribers(ctx, pool, ff)
		if err != nil {
			t.Fatalf("import: %v", err)
		}
		return res
	}

	if r := imp(one); r.Created != 1 {
		t.Fatalf("first import created %d, want 1", r.Created)
	}

	// (2) An UNCHANGED re-import must report nothing updated.
	r := imp(one)
	if r.Created != 0 {
		t.Fatalf("unchanged re-import created %d, want 0", r.Created)
	}
	if r.Updated != 0 {
		t.Errorf("🛑 unchanged re-import reports updated=%d, want 0 — the count claims an update that did not happen", r.Updated)
	}
	if r.Unchanged != 1 {
		t.Errorf("unchanged re-import reports unchanged=%d, want 1", r.Unchanged)
	}

	// (1) A corrected name and email must CONVERGE, and that one IS an update.
	r = imp(corrected)
	if r.Updated != 1 {
		t.Errorf("corrected re-import reports updated=%d, want 1", r.Updated)
	}
	var name, email string
	var sms, emailOK bool
	if err := pool.QueryRow(ctx,
		`SELECT display_name, email, sms_consent, email_consent FROM subscribers WHERE external_ref='9001'`).
		Scan(&name, &email, &sms, &emailOK); err != nil {
		t.Fatalf("read the row: %v", err)
	}
	if name != "Dana Reyes" {
		t.Errorf("🛑 display_name = %q, want %q — an upstream correction was silently discarded", name, "Dana Reyes")
	}
	if email != "dana.reyes@example.com" {
		t.Errorf("🛑 email = %q, want the corrected address", email)
	}
	// Consent stays MONOTONIC — the added Phone box lands, and nothing that was
	// true becomes false. That asymmetry with display_name is deliberate and is
	// documented on mergeSubscriber.
	if !sms || !emailOK {
		t.Errorf("consent = sms:%v email:%v, want both true (the corrected submission ticked both)", sms, emailOK)
	}
	if n := countRows(t, pool, `SELECT count(*) FROM subscribers`); n != 1 {
		t.Fatalf("%d subscribers after three imports of one submission, want 1", n)
	}
}

// TestNormalizeE164RejectsImpossibleAreaCodes is the low finding G6 flagged:
// a mistyped "1 773 555 482" is ten digits, so the old heuristic accepted it as
// +11773555482 and rendered a CONFIDENTLY WRONG last4 of "5482".
//
// No NANP area code begins with 0 or 1, so such a number is not merely unknown
// — it cannot exist. Rejecting it turns a wrong value into no value, which the
// UI already renders honestly as "No phone".
func TestNormalizeE164RejectsImpossibleAreaCodes(t *testing.T) {
	for _, raw := range []string{"1 773 555 482", "1773555482", "0773555482", "(177) 355-5482"} {
		if got := sources.NormalizeE164(raw); got != "" {
			t.Errorf("NormalizeE164(%q) = %q, want \"\" — no NANP area code starts with 0 or 1", raw, got)
		}
	}
	// The spike's set is unaffected: 773 is a real area code.
	if got := sources.NormalizeE164("17735554821"); got != "+17735554821" {
		t.Errorf("NormalizeE164 broke the 11-digit country-code case: %q", got)
	}
}
