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
//	403 -> {"error":"forbidden","missing_grant":"inventory-trends"}

var permPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		os.Exit(m.Run()) // no DB — DB-backed tests skip
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
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

// resetGrants clears app_permissions and the users this file creates.
func resetGrants(t *testing.T) {
	t.Helper()
	if _, err := permPool.Exec(t.Context(),
		`DELETE FROM app_permissions`); err != nil {
		t.Fatalf("clear app_permissions: %v", err)
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

// ── The two seed rows Option (i) depends on ─────────────────────────────────

func TestSeedHQApps_RegistersPerTabSlugs(t *testing.T) {
	requireDB(t)
	for _, slug := range []string{"inventory-trends", "inventory-cost"} {
		var n int
		if err := permPool.QueryRow(t.Context(),
			`SELECT count(*) FROM hq_apps WHERE slug = $1 AND enabled = true`, slug,
		).Scan(&n); err != nil {
			t.Fatalf("query hq_apps: %v", err)
		}
		if n != 1 {
			t.Errorf("hq_apps slug %q: got %d enabled rows, want 1 "+
				"(SeedHQApps must register the per-tab slugs — design §1.4)", slug, n)
		}
	}
}

// ── The with/without-grant pair, per tab ────────────────────────────────────

func TestRequirePermission_WithoutGrant_403(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "nogrant", []string{"team_member"})

	for _, tc := range []struct{ tab, want string }{
		{"inventory-trends", "inventory-trends"},
		{"inventory-cost", "inventory-cost"},
	} {
		code, body, reached := callGate(t, u, tc.tab, "inventory")
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
	grantRole(t, "inventory-trends", "manager")
	if code, _, reached := callGate(t, u, "inventory-trends", "inventory"); code != http.StatusOK || !reached {
		t.Errorf("role-granted trends: status=%d reached=%v, want 200/true", code, reached)
	}

	// individual grant
	u2 := mkUser(t, "usergrant", []string{"team_member"})
	grantUser(t, "inventory-cost", u2.ID)
	if code, _, reached := callGate(t, u2, "inventory-cost", "inventory"); code != http.StatusOK || !reached {
		t.Errorf("user-granted cost: status=%d reached=%v, want 200/true", code, reached)
	}
}

// The operator's umbrella rider: a whole-app `inventory` grant covers every tab.
func TestRequirePermission_UmbrellaAppGrant_Passes(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	u := mkUser(t, "umbrella", []string{"team_member"})
	grantUser(t, "inventory", u.ID)

	for _, tab := range []string{"inventory-trends", "inventory-cost"} {
		code, body, reached := callGate(t, u, tab, "inventory")
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
	grantUser(t, "inventory-trends", u.ID)

	if code, _, reached := callGate(t, u, "inventory-trends", "inventory"); code != http.StatusOK || !reached {
		t.Errorf("mixed user, trends: status=%d reached=%v, want 200/true", code, reached)
	}
	code, body, reached := callGate(t, u, "inventory-cost", "inventory")
	if code != http.StatusForbidden || reached {
		t.Errorf("mixed user, cost: status=%d reached=%v, want 403/false", code, reached)
	}
	if !strings.Contains(body, `"missing_grant":"inventory-cost"`) {
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

	for _, tab := range []string{"inventory-trends", "inventory-cost"} {
		if code, _, reached := callGate(t, u, tab, "inventory"); code != http.StatusForbidden || reached {
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

	for _, tab := range []string{"inventory-trends", "inventory-cost"} {
		if code, _, reached := callGate(t, u, tab, "inventory"); code != http.StatusOK || !reached {
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

	if code, _, reached := callGate(t, u, "inventory-trends", "inventory"); code != http.StatusForbidden || reached {
		t.Errorf("admin role without grant: status=%d reached=%v, want 403/false", code, reached)
	}
	grantRole(t, "inventory", "admin")
	if code, _, reached := callGate(t, u, "inventory-trends", "inventory"); code != http.StatusOK || !reached {
		t.Errorf("admin role WITH umbrella grant: status=%d reached=%v, want 200/true", code, reached)
	}
}

// Defence in depth: no user on the context (middleware misordered) must never
// fall through to the handler.
func TestRequirePermission_NoUser_401(t *testing.T) {
	requireDB(t)
	code, body, reached := callGate(t, nil, "inventory-trends", "inventory")
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
	grantUser(t, "inventory-trends", u.ID)

	if _, err := permPool.Exec(t.Context(),
		`UPDATE hq_apps SET enabled = false WHERE slug = 'inventory-trends'`); err != nil {
		t.Fatalf("disable app: %v", err)
	}
	t.Cleanup(func() {
		permPool.Exec(context.Background(),
			`UPDATE hq_apps SET enabled = true WHERE slug = 'inventory-trends'`)
	})

	if code, _, reached := callGate(t, u, "inventory-trends", "inventory"); code != http.StatusForbidden || reached {
		t.Errorf("disabled app: status=%d reached=%v, want 403/false", code, reached)
	}
}
