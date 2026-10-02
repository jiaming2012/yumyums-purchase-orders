package marketing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── the live-substrate gate ──────────────────────────────────────────────────
//
// MirrorLiveEnv arms internal/sync's asymmetric-gate pattern (see
// sync/proxy_live_test.go's requireSpikeService, and internal/testdb's header):
//
//	unset + substrate down -> SKIP, loudly, with the reason
//	set   + substrate down -> FAIL, with the reason
//
// Setting it is a statement of intent ("run the live leg"); a skip would then be
// a silent downgrade of exactly the assertion the card's done_when names. The
// substrate is auto-detected when the variable is unset, so a box that has it up
// runs live without being told to — which is how this ran on run 20261002.
const MirrorLiveEnv = "HQ_MARKETING_MIRROR_LIVE"

// substrateCompose resolves the local spike-supabase stack exactly as
// .night-crew/spikes/.../03-scan-attempts-service-read.sh does: the compose
// PROJECT, not hardcoded ports.
type substrate struct {
	RESTURL    string
	ServiceKey string
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// backend/internal/marketing -> repo root
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

// liveSubstrate discovers the local substrate, mints a service_role JWT with the
// committed throwaway secret, and proves the token works with one read. Returns
// nil when the stack is not up and MirrorLiveEnv is unset.
func liveSubstrate(t *testing.T) *substrate {
	t.Helper()
	armed := os.Getenv(MirrorLiveEnv) != ""
	give := func(format string, a ...any) *substrate {
		msg := fmt.Sprintf(format, a...)
		if armed {
			t.Fatalf("%s=%q was set, so a LIVE substrate run was intended, but %s. "+
				"This is a FAILURE and not a skip on purpose: a skip here silently drops the one "+
				"assertion this test exists for. Bring the stack up with "+
				".night-crew/qa/spike-supabase/env-up.sh (or task spike:up).",
				MirrorLiveEnv, os.Getenv(MirrorLiveEnv), msg)
		}
		t.Skipf("no local spike-supabase substrate (%s); set %s=1 to make this a failure instead", msg, MirrorLiveEnv)
		return nil
	}

	root := repoRoot(t)
	composeFile := filepath.Join(root, "docker-compose.supabase.yml")
	if _, err := os.Stat(composeFile); err != nil {
		return give("docker-compose.supabase.yml not found at %s", composeFile)
	}
	out, err := exec.Command("docker", "compose", "-p", "spike-supabase",
		"--project-directory", root, "-f", composeFile, "port", "rest", "3000").Output()
	if err != nil {
		return give("`docker compose -p spike-supabase port rest 3000` failed: %v", err)
	}
	hostPort := strings.TrimSpace(string(out))
	if hostPort == "" {
		return give("spike-supabase rest is not up (no published port)")
	}
	port := hostPort[strings.LastIndex(hostPort, ":")+1:]

	raw, err := os.ReadFile(composeFile)
	if err != nil {
		return give("read %s: %v", composeFile, err)
	}
	m := regexp.MustCompile(`JWT_SECRET: *([0-9a-f]{32,})`).FindSubmatch(raw)
	if m == nil {
		return give("no JWT_SECRET in %s", composeFile)
	}
	key := mintHS256(t, string(m[1]), map[string]any{
		"role": "service_role",
		"sub":  "hq-server",
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(10 * time.Minute).Unix(),
	})

	s := &substrate{RESTURL: "http://127.0.0.1:" + port, ServiceKey: key}
	// Prove the coordinates before the test leans on them: a 401 here is a bad
	// secret, not a bug in the mirror.
	code, _, err := s.do(t, t.Context(), http.MethodGet, "/scan_attempts?select=id&limit=1", nil)
	if err != nil {
		return give("probe read failed: %v", err)
	}
	if code != http.StatusOK {
		return give("service_role probe read answered HTTP %d (expected 200)", code)
	}
	return s
}

// mintHS256 is the same ~10 lines .night-crew/qa/spike-supabase/mintjwt is —
// stdlib only, no JWT library pulled into this module for a test.
func mintHS256(t *testing.T, secret string, claims map[string]any) string {
	t.Helper()
	b64 := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	signing := b64(header) + "." + b64(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	return signing + "." + b64(mac.Sum(nil))
}

// do carries an explicit context because the CLEANUP path cannot use
// t.Context(): that context is already cancelled by the time t.Cleanup runs, so
// a cleanup built on it silently fails and leaves this test's far-future rows on
// a SHARED substrate for the next card to trip over.
func (s *substrate) do(t *testing.T, ctx context.Context, method, path string, body any) (int, []byte, error) {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.RESTURL+path, rdr)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("apikey", s.ServiceKey)
	req.Header.Set("Authorization", "Bearer "+s.ServiceKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, out, nil
}

// seedUpstreamAttempt inserts one row into the substrate's OWN
// public.scan_attempts — the DEVICE's job, which is why the test does it. The
// shipped code under test does the READ and the HQ-side upsert; this test writes
// nothing the shipped code is supposed to write.
func (s *substrate) seedUpstreamAttempt(t *testing.T, row map[string]any) {
	t.Helper()
	code, body, err := s.do(t, t.Context(), http.MethodPost, "/scan_attempts", []map[string]any{row})
	if err != nil {
		t.Fatalf("seed upstream attempt: %v", err)
	}
	if code < 200 || code > 299 {
		t.Fatalf("seed upstream attempt: HTTP %d: %s", code, string(body))
	}
}

func (s *substrate) deleteUpstreamAttempts(t *testing.T, ids ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	code, body, err := s.do(t, ctx, http.MethodDelete, "/scan_attempts?id=in.("+strings.Join(ids, ",")+")", nil)
	if err != nil {
		t.Logf("cleanup: delete upstream attempts: %v", err)
		return
	}
	if code < 200 || code > 299 {
		t.Logf("cleanup: delete upstream attempts HTTP %d: %s", code, string(body))
	}
}

// setupMirrorDB empties the HQ-side mirror (and the decisions that reference it).
func setupMirrorDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and the local fallback is unreachable)")
	}
	if _, err := testPool.Exec(t.Context(),
		`TRUNCATE reconciliation_decisions, scan_attempts_mirror RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate mirror: %v", err)
	}
	return testPool
}

// TestScanAttemptsMirrorKeysetResumes is the card's second done_when row.
//
// 🛑 THIS RUNS AGAINST THE LIVE SUBSTRATE, NOT A RECORDED FIXTURE. The rows are
// seeded into the substrate's own public.scan_attempts over PostgREST as
// service_role (what a device does), and then the SHIPPED MirrorPollOnce does
// the PostgREST read, builds the keyset predicate itself, and performs the
// HQ-side upsert. Nothing is stubbed, and the test never writes a value the
// shipped code is supposed to write.
//
// What it proves, in one sequence:
//
//	poll 0  drains whatever the substrate already holds, so the assertions below
//	        are about THIS test's rows and not about another card's leftovers.
//	seed A  (scanned_at = T, id = 1111…)
//	poll 1  fetches exactly A; the cursor becomes (T, 1111…)
//	seed B  (scanned_at = T, id = 2222…)  — a TIE on scanned_at, greater id
//	seed C  (scanned_at = T+1m)
//	poll 2  fetches exactly TWO rows — B and C, not three:
//	          · A is NOT re-read            => the poll RESUMED at poll 1's last
//	                                           (scanned_at, id)
//	          · B IS read                   => the tie-break arm works; a
//	                                           scanned_at-only "since" filter
//	                                           would have lost B forever
//	        and A's mirrored_at is UNCHANGED, which is the independent witness
//	        that the resume happened in the QUERY rather than being hidden by the
//	        idempotent upsert.
func TestScanAttemptsMirrorKeysetResumes(t *testing.T) {
	pool := setupMirrorDB(t)
	sub := liveSubstrate(t)
	if sub == nil {
		return // liveSubstrate already skipped or failed
	}
	cfg := ProjectionConfig{RESTURL: sub.RESTURL, ServiceKey: sub.ServiceKey}
	ctx := context.Background()

	const (
		idA = "11111111-1111-4111-8111-111111111111"
		idB = "22222222-2222-4222-8222-222222222222"
		idC = "33333333-3333-4333-8333-333333333333"
	)
	t.Cleanup(func() { sub.deleteUpstreamAttempts(t, idA, idB, idC) })
	sub.deleteUpstreamAttempts(t, idA, idB, idC) // a previous aborted run's rows

	// Far-future timestamps so this test's rows sort strictly after every real
	// attempt on a shared substrate, whatever else is on it.
	const tie = "2099-01-01T12:00:00Z"
	const later = "2099-01-01T12:01:00Z"
	attempt := func(id, scannedAt, status, matchStatus string) map[string]any {
		return map[string]any{
			"id":                id,
			"code_id":           "44444444-4444-4444-8444-444444444444",
			"device_id":         "device-h3a",
			"scanned_at":        scannedAt,
			"status":            status,
			"reason":            nil,
			"offline_override":  false,
			"unverified_code":   false,
			"policy_unresolved": false,
			"pos_order_number":  "9999",
			"pos_business_date": "2099-01-01",
			"redeemed_value":    "2.00",
			"match_status":      matchStatus,
		}
	}

	// poll 0 — drain the substrate's existing content.
	drain, err := MirrorPollOnce(ctx, pool, cfg)
	if err != nil {
		t.Fatalf("drain poll: %v", err)
	}
	t.Logf("poll 0 (drain): fetched=%d pages=%d cursor=%v", drain.Fetched, drain.Pages, mirrorCursorLog(drain.To))

	// poll 1 — one new row.
	sub.seedUpstreamAttempt(t, attempt(idA, tie, "accepted", "unmatched"))
	p1, err := MirrorPollOnce(ctx, pool, cfg)
	if err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	if p1.Fetched != 1 || p1.Upserted != 1 {
		t.Fatalf("poll 1 fetched=%d upserted=%d, want 1/1", p1.Fetched, p1.Upserted)
	}
	if p1.To == nil || p1.To.ID != idA {
		t.Fatalf("poll 1 left cursor %v, want id %s", mirrorCursorLog(p1.To), idA)
	}
	var mirroredA time.Time
	if err := pool.QueryRow(ctx,
		`SELECT mirrored_at FROM scan_attempts_mirror WHERE id = $1`, idA).Scan(&mirroredA); err != nil {
		t.Fatalf("read A's mirrored_at: %v", err)
	}
	t.Logf("poll 1: fetched=%d cursor=%s", p1.Fetched, p1.To)

	// poll 2 — a TIE on scanned_at (greater id) plus a later row.
	sub.seedUpstreamAttempt(t, attempt(idB, tie, "rejected", "orphan"))
	sub.seedUpstreamAttempt(t, attempt(idC, later, "accepted", "matched"))

	p2, err := MirrorPollOnce(ctx, pool, cfg)
	if err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if p2.From == nil || p2.From.ID != idA {
		t.Fatalf("poll 2 STARTED at %v, want poll 1's last (scanned_at,id) with id %s", mirrorCursorLog(p2.From), idA)
	}
	if p2.Fetched != 2 {
		t.Fatalf("poll 2 fetched %d rows, want exactly 2 (B and C). "+
			"3 means it did NOT resume and re-read A; 1 means the (scanned_at,id) tie-break dropped B", p2.Fetched)
	}
	if p2.To == nil || p2.To.ID != idC {
		t.Fatalf("poll 2 left cursor %v, want id %s", mirrorCursorLog(p2.To), idC)
	}

	// A was not re-read: its mirrored_at never moved. The upsert would have
	// hidden a re-read from a row count; this is the independent witness.
	var mirroredAAfter time.Time
	if err := pool.QueryRow(ctx,
		`SELECT mirrored_at FROM scan_attempts_mirror WHERE id = $1`, idA).Scan(&mirroredAAfter); err != nil {
		t.Fatalf("re-read A's mirrored_at: %v", err)
	}
	if !mirroredAAfter.Equal(mirroredA) {
		t.Fatalf("A's mirrored_at moved (%s -> %s): poll 2 re-read the cursor row instead of resuming after it",
			mirroredA, mirroredAAfter)
	}

	// And the mirror carries the CONTENT, not just the keys — this is the F4
	// scan_attempts-status bullet: status, reason and match_status are now
	// visible inside HQ.
	var rows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM scan_attempts_mirror WHERE id = ANY($1)`,
		[]string{idA, idB, idC}).Scan(&rows); err != nil {
		t.Fatalf("count mirrored rows: %v", err)
	}
	if rows != 3 {
		t.Fatalf("mirror holds %d of this test's 3 attempts", rows)
	}
	var status, matchStatus, deviceID string
	var posOrder *string
	if err := pool.QueryRow(ctx,
		`SELECT status, match_status, device_id, pos_order_number FROM scan_attempts_mirror WHERE id = $1`, idB).
		Scan(&status, &matchStatus, &deviceID, &posOrder); err != nil {
		t.Fatalf("read B: %v", err)
	}
	if status != "rejected" || matchStatus != "orphan" || deviceID != "device-h3a" {
		t.Fatalf("B mirrored as status=%q match_status=%q device=%q, want rejected/orphan/device-h3a",
			status, matchStatus, deviceID)
	}
	if posOrder == nil || *posOrder != "9999" {
		t.Fatalf("B's pos_order_number did not mirror (the §13 Toast join key)")
	}

	t.Logf("LIVE substrate %s — poll1 fetched=1 cursor=%s; poll2 fetched=2 from=%s to=%s; A's mirrored_at unchanged",
		sub.RESTURL, p1.To, p2.From, p2.To)
}

// TestMirrorUnconfiguredIsIdleNotFatal pins the other half of the card's
// posture: no substrate coordinates means the poller never starts and says so.
// A box with no substrate is the common case, and a crash there would take the
// whole server down for a background copier.
func TestMirrorUnconfiguredIsIdleNotFatal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Must not panic, must not start, must not touch the pool.
	MirrorStart(ctx, nil, ProjectionConfig{})

	if _, err := MirrorPollOnce(ctx, nil, ProjectionConfig{}); err == nil {
		t.Fatalf("MirrorPollOnce with no coordinates should error, not silently report success")
	}
	if _, err := MirrorPollOnce(ctx, nil, ProjectionConfig{RESTURL: "http://x", ServiceKey: "k"}); err == nil {
		t.Fatalf("MirrorPollOnce with a nil pool should error")
	}
}
