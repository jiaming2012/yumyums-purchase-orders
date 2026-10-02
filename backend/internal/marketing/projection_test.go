package marketing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// done_when #4 — TestProjectionUnconfiguredLeavesProjectedAtNull
//
// Decision 187's fail-loud half, verbatim: "with HQ_SYNC_REST_URL unset the
// campaign saves with projected_at NULL and the UI shows a 'not on tablets yet'
// pill — never a silent success."
//
// Two things must both hold, and the second is the one that matters: the save
// SUCCEEDS (a campaign a manager typed is not lost because a substrate is
// down), and the response says so out loud via warnings:["not_projected"].
// ─────────────────────────────────────────────────────────────────────────────
func TestProjectionUnconfiguredLeavesProjectedAtNull(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")

	// Belt and braces: assert the config this test depends on really is the
	// unconfigured one, rather than trusting the ambient environment.
	d := testDeps(pool)
	if d.Projection.Configured() {
		t.Fatalf("testDeps handed a CONFIGURED projection (%q) — this test must exercise the unconfigured branch",
			d.Projection.RESTURL)
	}
	mux := mountedMux(d)

	out := createCampaign(t, mux, mgr, map[string]any{
		"name": "Not On Tablets Yet", "offer_text": "$2 off",
		"face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})

	if out.Campaign.ProjectedAt != nil {
		t.Errorf("projected_at = %v with the projection unconfigured, want null", *out.Campaign.ProjectedAt)
	}
	if len(out.Warnings) != 1 || out.Warnings[0] != WarningNotProjected {
		t.Errorf("warnings = %v, want exactly [%q] — a silent success is the failure mode decision 187 names",
			out.Warnings, WarningNotProjected)
	}

	// And the column itself is NULL on disk, not merely omitted from the body.
	var projectedAt *time.Time
	if err := pool.QueryRow(context.Background(),
		`SELECT projected_at FROM campaigns_admin WHERE id = $1`, out.Campaign.ID).Scan(&projectedAt); err != nil {
		t.Fatalf("select projected_at: %v", err)
	}
	if projectedAt != nil {
		t.Errorf("campaigns_admin.projected_at = %v on disk, want NULL", *projectedAt)
	}

	// The campaign and its code still exist — the save was not rolled back.
	var codes int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_codes WHERE campaign_id = $1`, out.Campaign.ID).Scan(&codes); err != nil {
		t.Fatalf("count codes: %v", err)
	}
	if codes != 1 {
		t.Errorf("qr_codes rows = %d, want 1 — an unreachable substrate must not lose the campaign", codes)
	}
}

// A projection that is configured but REFUSES takes the same fail-loud path as
// one that is unset: the campaign saves, projected_at stays NULL, the warning
// fires.
//
// The stand-in is an httptest server answering 503 rather than an unreachable
// port: on this box a connection to a dead local port HANGS rather than being
// refused, and the handler's own 15s transport backstop then made this one test
// 18s of the package's 20s. A 503 is also the stricter assertion — it proves a
// non-2xx response is treated as a failure, not just a dropped connection.
func TestProjectionFailureLeavesProjectedAtNull(t *testing.T) {
	pool := setupTestDB(t)
	mgr := seedUser(t, pool, "manager")

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"message":"substrate down"}`))
	}))
	defer srv.Close()

	d := testDeps(pool)
	d.Projection = ProjectionConfig{RESTURL: srv.URL, ServiceKey: "not-a-key"}
	if !d.Projection.Configured() {
		t.Fatal("a REST url plus a key must read as configured")
	}
	mux := mountedMux(d)

	out := createCampaign(t, mux, mgr, map[string]any{
		"name": "Refusing Substrate", "offer_text": "$2 off",
		"face_value_cents": 200, "runs_days": 7,
		"channels": []map[string]any{{"channel": "flyer"}},
	})
	if out.Campaign.ProjectedAt != nil {
		t.Errorf("projected_at = %v against a refusing substrate, want null", *out.Campaign.ProjectedAt)
	}
	if len(out.Warnings) != 1 || out.Warnings[0] != WarningNotProjected {
		t.Errorf("warnings = %v, want [%q]", out.Warnings, WarningNotProjected)
	}
	// Two calls landed: the marketing_settings threshold read and the upsert.
	// Both refused, and neither cost the manager their campaign.
	if hits < 2 {
		t.Errorf("the substrate saw %d requests, want >= 2 (the threshold read and the upsert)", hits)
	}
	// The threshold read refusing means the DEFAULT threshold was used, and at
	// 200 cents that still means requires_online is false — stated here so a
	// future threshold change cannot make this assertion silently vacuous.
	if out.Campaign.RequiresOnline {
		t.Errorf("requires_online = true with face_value 200 and the default threshold %d",
			DefaultRequiresOnlineThresholdCents)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// The CONFIGURED path, against the real local substrate.
//
// This is the leg the card calls out by name: "the projection's CONFIGURED path
// proven against the local substrate (spike 02's upsert, now from Go)". It is
// the Go port of
// .night-crew/spikes/activity-h-designed-tabs/campaign-codes-api/02-projection-upsert-over-postgrest.sh
// and it uses the SAME asymmetry internal/sync's HQ_SYNC_SPIKE_LIVE gate does:
//
//	MKT_PROJECTION_LIVE unset        -> SKIP (no substrate on a contributor's box)
//	MKT_PROJECTION_LIVE=1, no config -> FAIL (the intent was stated and not met)
//	MKT_PROJECTION_LIVE=1, configured-> run it for real
//
// 🛑 The test NEVER writes the value the shipped code is supposed to write. It
// calls ProjectCampaign — production code — and then reads the row back over
// its own independent PostgREST GET. A stubbed green here would be exactly the
// thing the run's reviewer is briefed to hunt.
//
// 🛑 LOCAL substrate only: the Taskfile wrapper resolves the port from the
// `spike-supabase` compose project. Never a hosted project, never :5433.
// ─────────────────────────────────────────────────────────────────────────────
func TestProjectionConfiguredUpsertsToSubstrate(t *testing.T) {
	if os.Getenv("MKT_PROJECTION_LIVE") != "1" {
		t.Skip("MKT_PROJECTION_LIVE != 1 — the local spike-supabase substrate leg is opt-in")
	}
	cfg := LoadProjectionConfig()
	if !cfg.Configured() {
		t.Fatalf("MKT_PROJECTION_LIVE=1 was set, so a LIVE projection run was intended, but %s and/or %s is empty. "+
			"This is a FAILURE and not a skip on purpose: a skip here would let the one leg that proves the "+
			"configured path degrade into a stubbed green.", RESTURLEnv, ServiceKeyEnv)
	}
	if strings.Contains(cfg.RESTURL, "5433") {
		t.Fatalf("refusing to run: %s points at :5433, which is the dev AND PRODUCTION cluster (decision 155)", RESTURLEnv)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// A campaign id this test owns outright, so nothing the substrate already
	// holds is touched. Cleaned up at the end (reconcile, never destroy).
	id := fmt.Sprintf("a1000000-0000-4000-8000-%012d", time.Now().UnixNano()%1e12)
	row := ProjectionRow{
		ID:             id,
		Name:           "H1 Go projection " + id[len(id)-6:],
		FaceValueCents: 4000,
		RequiresOnline: true,
	}
	t.Cleanup(func() { deleteProjectedRow(t, cfg, id) })

	// ── PRODUCTION CODE DOES THE WRITE ──
	if err := ProjectCampaign(ctx, cfg, row); err != nil {
		t.Fatalf("ProjectCampaign against the local substrate: %v", err)
	}

	// ── THE TEST DOES ITS OWN INDEPENDENT READ-BACK ──
	got := readProjectedRow(t, cfg, id)
	if got.Name != row.Name {
		t.Errorf("projected name = %q, want %q", got.Name, row.Name)
	}
	if got.FaceValue != 40.00 {
		t.Errorf("projected face_value = %v, want 40 (4000 cents as the numeric dollars the replica carries)", got.FaceValue)
	}
	if !got.RequiresOnline {
		t.Error("projected requires_online = false, want true")
	}
	t.Logf("LIVE projection OK: %s -> campaigns{name=%q, face_value=%v, requires_online=%v} via %s",
		id, got.Name, got.FaceValue, got.RequiresOnline, cfg.RESTURL)

	// Upsert, not insert: a second projection of the same id updates in place
	// (Prefer: resolution=merge-duplicates — the leg the UPDATE path rests on).
	row.Name = row.Name + " v2"
	row.RequiresOnline = false
	if err := ProjectCampaign(ctx, cfg, row); err != nil {
		t.Fatalf("second ProjectCampaign (the upsert leg): %v", err)
	}
	got = readProjectedRow(t, cfg, id)
	if got.Name != row.Name || got.RequiresOnline {
		t.Errorf("after re-projection: name=%q requires_online=%v, want %q / false", got.Name, got.RequiresOnline, row.Name)
	}
}

type projectedRow struct {
	Name           string  `json:"name"`
	FaceValue      float64 `json:"face_value"`
	RequiresOnline bool    `json:"requires_online"`
}

// readProjectedRow is the test's OWN PostgREST read — deliberately not routed
// through any helper the shipped projection uses.
func readProjectedRow(t *testing.T, cfg ProjectionConfig, id string) projectedRow {
	t.Helper()
	url := strings.TrimRight(cfg.RESTURL, "/") +
		"/campaigns?id=eq." + id + "&select=name,face_value,requires_online"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build read-back request: %v", err)
	}
	req.Header.Set("apikey", cfg.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("read-back GET: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read-back GET status %d: %s", resp.StatusCode, body)
	}
	var rows []projectedRow
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatalf("read-back decode %s: %v", body, err)
	}
	if len(rows) != 1 {
		t.Fatalf("read-back returned %d rows for id %s, want 1 (body: %s)", len(rows), id, body)
	}
	return rows[0]
}

// deleteProjectedRow removes only the row this test created.
func deleteProjectedRow(t *testing.T, cfg ProjectionConfig, id string) {
	t.Helper()
	url := strings.TrimRight(cfg.RESTURL, "/") + "/campaigns?id=eq." + id
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Logf("cleanup: build delete: %v", err)
		return
	}
	req.Header.Set("apikey", cfg.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceKey)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Logf("cleanup: delete %s: %v", id, err)
		return
	}
	resp.Body.Close()
	t.Logf("cleanup: deleted projected row %s (status %d)", id, resp.StatusCode)
}
