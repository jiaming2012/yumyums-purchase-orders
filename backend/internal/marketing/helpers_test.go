package marketing

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/testdb"
)

// setupTestDB connects to the test database named by DB_TEST_URL and truncates
// the three tables migration 0083 adds, so every test in this package starts
// from an empty campaign admin.
//
// It applies internal/testdb's asymmetric gate verbatim (the same shape
// internal/recipes uses): DB_TEST_URL UNSET skips — a contributor without a
// Postgres must still be able to run the hermetic tests in this package — and
// DB_TEST_URL SET but unreachable FAILS, because setting it is a statement of
// intent that a database-backed run was wanted. Both arms used to skip, which
// is how a DROPped database once read as `ok` (B-16, decision 90).
//
// It deliberately does NOT truncate `users` or `menu_items`: no other package's
// TestMain does, and campaigns_admin.created_by / item_id are FKs into them.
// Fixtures here create their own uniquely-named rows instead.
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv(testdb.EnvVar)
	if dbURL == "" {
		t.Skip("DB_TEST_URL not set — skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(testdb.Reason(dbURL, "connect", err))
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(testdb.Reason(dbURL, "ping", err))
	}
	if _, err := pool.Exec(ctx,
		`TRUNCATE qr_scans, qr_codes, campaigns_admin RESTART IDENTITY CASCADE`); err != nil {
		pool.Close()
		t.Fatalf("setupTestDB truncate: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

// randSuffix returns 8 random hex chars, used to keep fixture emails and
// menu_items.master_id unique across repeated runs without pulling in a uuid
// dependency this module does not otherwise have.
func randSuffix(t *testing.T) string {
	t.Helper()
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("randSuffix: %v", err)
	}
	return hex.EncodeToString(b)
}

// seedUser inserts a users row with the given roles and returns its id. The
// email carries a uuid so repeated runs never collide on users_email_key.
func seedUser(t *testing.T, pool *pgxpool.Pool, roles ...string) string {
	t.Helper()
	var id string
	email := fmt.Sprintf("mkt-%s@test.invalid", randSuffix(t))
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, display_name, roles, status)
		 VALUES ($1, 'Marketing Test', $2, 'active') RETURNING id::text`,
		email, roles).Scan(&id)
	if err != nil {
		t.Fatalf("seedUser(%v): %v", roles, err)
	}
	return id
}

// seedMenuItem inserts a menu_items row and returns its id.
func seedMenuItem(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO menu_items (master_id, name, menu, menu_group, last_seen)
		 VALUES ($1, $2, 'Main', 'Lunch', CURRENT_DATE) RETURNING id::text`,
		"mkt-"+randSuffix(t), name).Scan(&id)
	if err != nil {
		t.Fatalf("seedMenuItem(%q): %v", name, err)
	}
	return id
}

// userCtx returns a context carrying an authenticated user the way
// auth.Middleware would, so the in-handler manager tier (§16) has something to
// read without a second DB lookup (spike manager-tier-derivable-in-handler).
func userCtx(id string, roles ...string) context.Context {
	return context.WithValue(context.Background(), auth.CtxKeyUser,
		&auth.User{ID: id, DisplayName: "Marketing Test", Roles: roles})
}

// testDeps builds Deps with the projection DELIBERATELY unconfigured, which is
// the normal state of this box and of the test harness: every test that is not
// specifically about the projection must exercise the fail-loud branch.
func testDeps(pool *pgxpool.Pool) Deps {
	return Deps{
		Pool:           pool,
		QRBaseURL:      DefaultQRBaseURL,
		LandingBaseURL: DefaultLandingBaseURL,
		Projection:     ProjectionConfig{},
	}
}

// mountedMux returns a chi mux with the gated /marketing routes AND the public
// landing mounted, so tests drive the same route table main.go does.
func mountedMux(d Deps) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/marketing", func(r chi.Router) { Mount(r, d) })
	MountPublic(r, d)
	return r
}

// do sends a request (optional JSON body) carrying ctx and returns the recorder.
func do(t *testing.T, mux *chi.Mux, ctx context.Context, method, target string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var body *bytes.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(b)
	} else {
		body = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// decode unmarshals a recorder body into v, failing loudly with the raw body so
// a 500's error envelope is visible in the test output instead of a type error.
func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode body (status %d): %v\nbody: %s", rec.Code, err, rec.Body.String())
	}
}

// countScans returns the number of qr_scans rows for a short code.
func countScans(t *testing.T, pool *pgxpool.Pool, short string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM qr_scans WHERE short = $1`, short).Scan(&n); err != nil {
		t.Fatalf("countScans(%q): %v", short, err)
	}
	return n
}

// createCampaign POSTs a campaign as a manager and returns the decoded
// response, failing the test unless the status is 201.
func createCampaign(t *testing.T, mux *chi.Mux, userID string, payload map[string]any) createCampaignResponse {
	t.Helper()
	rec := do(t, mux, userCtx(userID, "manager"), http.MethodPost, "/api/v1/marketing/campaigns", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /campaigns = %d, want 201\nbody: %s", rec.Code, rec.Body.String())
	}
	var out createCampaignResponse
	decode(t, rec, &out)
	return out
}
