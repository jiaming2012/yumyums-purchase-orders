package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

// ── auth.RequirePermission — design §1.2/§1.3/§1.4 (Option (i)) ──────────────
//
// The gate this file proves:
//
//	pass  ⇔  superadmin  ∨  grant on the TAB slug  ∨  grant on the UMBRELLA app slug
//
// The umbrella disjunct is the operator's signature rider (design §8 amendment 1,
// verbatim: "App grant = All tabs granted. They should not be considered separate
// objects."). It REPLACES the §1.5 draft text that said a whole-app grant does not
// imply tab grants.
//
// The 403 envelope is required to be DISTINCT from the 401 envelope so the client
// can tell "log in again" from "you lack this grant" (§1.2 rule 3):
//
//	401 -> {"error":"unauthorized"}
//	403 -> {"error":"forbidden","missing_grant":"test-app-alpha"}

var permPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	// Computed BEFORE the fallback: the fallback is the *unset* case, and the
	// unset case still skips. See internal/testdb for the asymmetry.
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
		os.Exit(m.Run()) // DB_TEST_URL unset, no local DB — DB-backed tests skip
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		testdb.ExitIfRequested(requested, dbURL, "ping", err)
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	if err := db.SeedHQApps(ctx, pool); err != nil {
		pool.Close()
		panic("db.SeedHQApps failed: " + err.Error())
	}
	permPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func requireDB(t *testing.T) {
	t.Helper()
	if permPool == nil {
		t.Skip("no test database (set DB_TEST_URL)")
	}
}

// resetGrants clears app_permissions and the users this file creates, and
// (re)installs the fixture rows the gate tests run against.
//
// The fixtures are a synthetic umbrella app `test-app` with two tab rows
// `test-app-alpha` / `test-app-beta` under the `<app>-<tab>` convention. They
// used to be the real `inventory` + `inventory-trends` / `inventory-cost`
// rows; those tabs are retired since B-455 / WO-2b (0082 disables them, the
// seed no longer creates them), and the gate's semantics are not about any
// particular app, so the tests now own their rows.
func resetGrants(t *testing.T) {
	t.Helper()
	if _, err := permPool.Exec(t.Context(),
		`DELETE FROM app_permissions`); err != nil {
		t.Fatalf("clear app_permissions: %v", err)
	}
	if _, err := permPool.Exec(t.Context(),
		`INSERT INTO hq_apps (slug, name, icon) VALUES
		   ('test-app', 'Test App', '🧪'),
		   ('test-app-alpha', 'Test App · Alpha', '🧪'),
		   ('test-app-beta', 'Test App · Beta', '🧪')
		 ON CONFLICT (slug) DO UPDATE SET enabled = true`); err != nil {
		t.Fatalf("install fixture apps: %v", err)
	}
	if _, err := permPool.Exec(t.Context(),
		`DELETE FROM users WHERE email LIKE 'perm-test-%'`); err != nil {
		t.Fatalf("clear users: %v", err)
	}
}

// mkUser inserts a user with the given roles and returns an auth.User for it.
func mkUser(t *testing.T, name string, roles []string) *User {
	t.Helper()
	var id string
	err := permPool.QueryRow(t.Context(),
		`INSERT INTO users (email, roles, status, first_name, last_name)
		 VALUES ($1, $2, 'active', $3, 'Tester') RETURNING id::text`,
		"perm-test-"+name+"@yumyums.kitchen", roles, name,
	).Scan(&id)
	if err != nil {
		t.Fatalf("insert user %s: %v", name, err)
	}
	return &User{ID: id, Email: "perm-test-" + name + "@yumyums.kitchen", Roles: roles, Status: "active"}
}

// grantRole grants an app slug to a role.
func grantRole(t *testing.T, slug, role string) {
	t.Helper()
	_, err := permPool.Exec(t.Context(),
		`INSERT INTO app_permissions (app_id, role)
		 SELECT id, $2 FROM hq_apps WHERE slug = $1`, slug, role)
	if err != nil {
		t.Fatalf("grant %s to role %s: %v", slug, role, err)
	}
}

// grantUser grants an app slug to an individual user.
func grantUser(t *testing.T, slug, userID string) {
	t.Helper()
	_, err := permPool.Exec(t.Context(),
		`INSERT INTO app_permissions (app_id, user_id)
		 SELECT id, $2::uuid FROM hq_apps WHERE slug = $1`, slug, userID)
	if err != nil {
		t.Fatalf("grant %s to user %s: %v", slug, userID, err)
	}
}

// callGate runs RequirePermission(tabSlug, umbrellaSlug) with the given user in
// context and reports the status plus whether the wrapped handler ran.
func callGate(t *testing.T, user *User, tabSlug, umbrellaSlug string) (int, string, bool) {
	t.Helper()
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if user != nil {
		req = req.WithContext(context.WithValue(req.Context(), CtxKeyUser, user))
	}
	rec := httptest.NewRecorder()
	RequirePermission(permPool, tabSlug, umbrellaSlug)(next).ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String()), reached
}

// ── Seed state after B-455 / WO-2b ──────────────────────────────────────────
//
// Trends and Cost are the `bi` app; the per-tab rows that used to gate them are
// retired. The seed must register `bi` enabled and must never (re)create the
// two tab rows — 0082 disables any that exist, and a seed insert would leave a
// fresh pair enabled on a new database.

func TestSeedHQApps_BIRegistered_TabRowsRetired(t *testing.T) {
	requireDB(t)
	var n int
	if err := permPool.QueryRow(t.Context(),
		`SELECT count(*) FROM hq_apps WHERE slug = 'bi' AND enabled = true`).Scan(&n); err != nil {
		t.Fatalf("query hq_apps: %v", err)
	}
	if n != 1 {
		t.Errorf("hq_apps slug \"bi\": got %d enabled rows, want 1", n)
	}
	for _, slug := range []string{"inventory-trends", "inventory-cost"} {
		if err := permPool.QueryRow(t.Context(),
			`SELECT count(*) FROM hq_apps WHERE slug = $1 AND enabled = true`, slug,
		).Scan(&n); err != nil {
			t.Fatalf("query hq_apps: %v", err)
		}
		if n != 0 {
			t.Errorf("hq_apps slug %q: got %d enabled rows, want 0 (retired by 0082; the seed must not recreate it)", slug, n)
		}
	}
}

// ── Migration 0082 — the grant copy ─────────────────────────────────────────
//
// With legacy grants on the two tab rows (a role grant on one, a user grant on
// the other), running 0082 must leave matching grants on `bi`, disable the tab
// rows, admit the holders through RequirePermission("bi"), and refuse them
// through the retired tab gate — a disabled row grants nothing.
func TestMigration0082_CopiesTabGrantsOntoBI(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	t.Cleanup(func() {
		// Leave the schema where every other test expects it.
		if err := db.MigrateTo(permPool, 82); err != nil {
			t.Errorf("restore schema to 0082: %v", err)
		}
	})

	// Roll 0082 back and stage the pre-migration world.
	if err := db.MigrateTo(permPool, 81); err != nil {
		t.Fatalf("migrate down to 0081: %v", err)
	}
	if _, err := permPool.Exec(t.Context(),
		`INSERT INTO hq_apps (slug, name, icon) VALUES
		   ('inventory-trends', 'Inventory · Trends', '📈'),
		   ('inventory-cost', 'Inventory · Cost', '💵')
		 ON CONFLICT (slug) DO UPDATE SET enabled = true`); err != nil {
		t.Fatalf("stage legacy tab rows: %v", err)
	}
	mgr := mkUser(t, "m82", []string{"manager"})
	grantRole(t, "inventory-trends", "manager")
	usr := mkUser(t, "u82", []string{"team_member"})
	grantUser(t, "inventory-cost", usr.ID)

	if err := db.MigrateTo(permPool, 82); err != nil {
		t.Fatalf("migrate up to 0082: %v", err)
	}

	var n int
	if err := permPool.QueryRow(t.Context(), `
		SELECT count(*) FROM app_permissions p JOIN hq_apps a ON a.id = p.app_id
		WHERE a.slug = 'bi' AND p.role = 'manager'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("bi role grant for manager: count=%d err=%v, want 1", n, err)
	}
	if err := permPool.QueryRow(t.Context(), `
		SELECT count(*) FROM app_permissions p JOIN hq_apps a ON a.id = p.app_id
		WHERE a.slug = 'bi' AND p.user_id = $1::uuid`, usr.ID).Scan(&n); err != nil || n != 1 {
		t.Errorf("bi user grant: count=%d err=%v, want 1", n, err)
	}
	if err := permPool.QueryRow(t.Context(), `
		SELECT count(*) FROM hq_apps WHERE slug IN ('inventory-trends','inventory-cost') AND enabled = false`).Scan(&n); err != nil || n != 2 {
		t.Errorf("tab rows disabled: count=%d err=%v, want 2", n, err)
	}

	for _, tc := range []struct {
		who  *User
		name string
	}{{mgr, "manager (role grant copied)"}, {usr, "user (user grant copied)"}} {
		if code, _, reached := callGateNarrow(t, tc.who, "bi"); code != http.StatusOK || !reached {
			t.Errorf("%s through bi gate: status=%d reached=%v, want 200/true", tc.name, code, reached)
		}
	}
	if code, body, reached := callGateNarrow(t, mgr, "inventory-trends"); code != http.StatusForbidden || reached {
		t.Errorf("manager through retired tab gate: status=%d reached=%v body=%s, want 403/false", code, reached, body)
	}
}

// callGateNarrow runs RequirePermission(slug) with no umbrella.
func callGateNarrow(t *testing.T, user *User, slug string) (int, string, bool) {
	t.Helper()
	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if user != nil {
		req = req.WithContext(context.WithValue(req.Context(), CtxKeyUser, user))
	}
	rec := httptest.NewRecorder()
	RequirePermission(permPool, slug)(next).ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String()), reached
}

// ── The with/without-grant pair, per tab ────────────────────────────────────

func TestRequirePermission_WithoutGrant_403(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "nogrant", []string{"team_member"})

	for _, tc := range []struct{ tab, want string }{
		{"test-app-alpha", "test-app-alpha"},
		{"test-app-beta", "test-app-beta"},
	} {
		code, body, reached := callGate(t, u, tc.tab, "test-app")
		if code != http.StatusForbidden {
			t.Errorf("%s ungranted: status = %d, want 403", tc.tab, code)
		}
		if reached {
			t.Errorf("%s ungranted: wrapped handler RAN — the gate is not a gate", tc.tab)
		}
		var env map[string]string
		if err := json.Unmarshal([]byte(body), &env); err != nil {
			t.Fatalf("%s ungranted: body %q is not JSON: %v", tc.tab, body, err)
		}
		if env["error"] != "forbidden" {
			t.Errorf("%s ungranted: error = %q, want \"forbidden\" (must differ from the 401 envelope)", tc.tab, env["error"])
		}
		if env["missing_grant"] != tc.want {
			t.Errorf("%s ungranted: missing_grant = %q, want %q", tc.tab, env["missing_grant"], tc.want)
		}
	}
}

func TestRequirePermission_WithTabGrant_Passes(t *testing.T) {
	requireDB(t)
	resetGrants(t)

	// role grant
	u := mkUser(t, "rolegrant", []string{"manager"})
	grantRole(t, "test-app-alpha", "manager")
	if code, _, reached := callGate(t, u, "test-app-alpha", "test-app"); code != http.StatusOK || !reached {
		t.Errorf("role-granted trends: status=%d reached=%v, want 200/true", code, reached)
	}

	// individual grant
	u2 := mkUser(t, "usergrant", []string{"team_member"})
	grantUser(t, "test-app-beta", u2.ID)
	if code, _, reached := callGate(t, u2, "test-app-beta", "test-app"); code != http.StatusOK || !reached {
		t.Errorf("user-granted cost: status=%d reached=%v, want 200/true", code, reached)
	}
}

// The operator's umbrella rider: a whole-app `inventory` grant covers every tab.
func TestRequirePermission_UmbrellaAppGrant_Passes(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "umbrella", []string{"team_member"})
	grantUser(t, "test-app", u.ID)

	for _, tab := range []string{"test-app-alpha", "test-app-beta"} {
		code, body, reached := callGate(t, u, tab, "test-app")
		if code != http.StatusOK || !reached {
			t.Errorf("umbrella grant, %s: status=%d body=%s reached=%v, want 200/true "+
				"(design §8 amendment 1 — app grant = all tabs granted)", tab, code, body, reached)
		}
	}
}

// §1.6 — the mixed case: Trends granted, Cost not.
func TestRequirePermission_MixedGrant_TrendsOnly(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "mixed", []string{"team_member"})
	grantUser(t, "test-app-alpha", u.ID)

	if code, _, reached := callGate(t, u, "test-app-alpha", "test-app"); code != http.StatusOK || !reached {
		t.Errorf("mixed user, trends: status=%d reached=%v, want 200/true", code, reached)
	}
	code, body, reached := callGate(t, u, "test-app-beta", "test-app")
	if code != http.StatusForbidden || reached {
		t.Errorf("mixed user, cost: status=%d reached=%v, want 403/false", code, reached)
	}
	if !strings.Contains(body, `"missing_grant":"test-app-beta"`) {
		t.Errorf("mixed user, cost: body = %s, want missing_grant inventory-cost", body)
	}
}

// A grant on the OTHER tab must not leak, and neither must an unrelated app.
func TestRequirePermission_UnrelatedGrant_DoesNotLeak(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "unrelated", []string{"team_member"})
	grantUser(t, "purchasing", u.ID)
	grantRole(t, "operations", "team_member")

	for _, tab := range []string{"test-app-alpha", "test-app-beta"} {
		if code, _, reached := callGate(t, u, tab, "test-app"); code != http.StatusForbidden || reached {
			t.Errorf("unrelated grants, %s: status=%d reached=%v, want 403/false", tab, code, reached)
		}
	}
}

// §1.2 rule 4 / §4 flag 5 — superadmins implicitly hold every grant, mirroring
// queryAllApps. Otherwise a superadmin sees a tab whose endpoint 403s.
func TestRequirePermission_Superadmin_Passes(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "super", []string{"admin"})
	u.IsSuperadmin = true

	for _, tab := range []string{"test-app-alpha", "test-app-beta"} {
		if code, _, reached := callGate(t, u, tab, "test-app"); code != http.StatusOK || !reached {
			t.Errorf("superadmin, %s: status=%d reached=%v, want 200/true", tab, code, reached)
		}
	}
}

// Plain `admin` role is NOT an implicit grant — only superadmin is (§1.2 rule 4
// names superadmins, not admins). Admins get access the ordinary way: a role
// grant. This pins the rule so it cannot drift into "admin sees everything".
func TestRequirePermission_AdminRoleAlone_IsNotAGrant(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "adminrole", []string{"admin"})

	if code, _, reached := callGate(t, u, "test-app-alpha", "test-app"); code != http.StatusForbidden || reached {
		t.Errorf("admin role without grant: status=%d reached=%v, want 403/false", code, reached)
	}
	grantRole(t, "test-app", "admin")
	if code, _, reached := callGate(t, u, "test-app-alpha", "test-app"); code != http.StatusOK || !reached {
		t.Errorf("admin role WITH umbrella grant: status=%d reached=%v, want 200/true", code, reached)
	}
}

// Defence in depth: no user on the context (middleware misordered) must never
// fall through to the handler.
func TestRequirePermission_NoUser_401(t *testing.T) {
	requireDB(t)
	code, body, reached := callGate(t, nil, "test-app-alpha", "test-app")
	if code != http.StatusUnauthorized || reached {
		t.Errorf("no user: status=%d reached=%v, want 401/false", code, reached)
	}
	if !strings.Contains(body, `"unauthorized"`) {
		t.Errorf("no user: body = %s, want unauthorized envelope", body)
	}
}

// A disabled app row must not grant. Guards the reversal path in §1.4
// ("deleting the two rows cascades the grants") — disabling must gate too.
func TestRequirePermission_DisabledApp_DoesNotGrant(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "disabled", []string{"team_member"})
	grantUser(t, "test-app-alpha", u.ID)

	if _, err := permPool.Exec(t.Context(),
		`UPDATE hq_apps SET enabled = false WHERE slug = 'test-app-alpha'`); err != nil {
		t.Fatalf("disable app: %v", err)
	}
	t.Cleanup(func() {
		permPool.Exec(context.Background(),
			`UPDATE hq_apps SET enabled = true WHERE slug = 'test-app-alpha'`)
	})

	if code, _, reached := callGate(t, u, "test-app-alpha", "test-app"); code != http.StatusForbidden || reached {
		t.Errorf("disabled app: status=%d reached=%v, want 403/false", code, reached)
	}
}

// ── Fail-closed on a DB error (G6 observation 4) ────────────────────────────
//
// The property that makes this gate trustworthy is that it cannot fail OPEN. If
// the grant lookup errors — pool exhausted, table missing, query cancelled — the
// middleware must refuse, never wave the request through on the reasoning that
// it "couldn't tell". G6 verified this by hand; without a test it could regress
// to fail-open silently, which is the worst possible regression here because
// nothing observable changes until someone is already through the door.
//
// The error is forced with a CLOSED pool: every Query on it returns
// "closed pool", the same shape a real outage produces at this call site.
func TestRequirePermission_DBError_FailsClosed(t *testing.T) {
	requireDB(t)

	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	brokenPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("build pool: %v", err)
	}
	brokenPool.Close() // every subsequent query fails

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), CtxKeyUser,
		&User{ID: "00000000-0000-0000-0000-000000000001", Roles: []string{"team_member"}}))
	rec := httptest.NewRecorder()

	RequirePermission(brokenPool, "test-app-alpha", "test-app")(next).ServeHTTP(rec, req)

	if reached {
		t.Error("DB error FAILED OPEN — the wrapped handler ran without a grant check")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("DB error: status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("DB error: body = %s, want an error envelope", rec.Body.String())
	}
}

// A superadmin is resolved before the grant query, so an outage must not lock
// them out — the one case where skipping the lookup is correct.
func TestRequirePermission_DBError_SuperadminStillPasses(t *testing.T) {
	requireDB(t)

	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	brokenPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("build pool: %v", err)
	}
	brokenPool.Close()

	reached := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached = true })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), CtxKeyUser,
		&User{ID: "00000000-0000-0000-0000-000000000002", IsSuperadmin: true}))
	RequirePermission(brokenPool, "test-app-alpha", "test-app")(next).
		ServeHTTP(httptest.NewRecorder(), req)

	if !reached {
		t.Error("superadmin blocked by an unrelated DB outage")
	}
}
