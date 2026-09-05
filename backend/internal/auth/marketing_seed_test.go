package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/yumyums/hq/internal/db"
)

// ── Card marketing-tile-and-page (run 20260905) — the marketing permission ──
// seed and the offline_override ENTITLEMENT surface (fork #12, operator-
// resolved; design docs/qr-offline-redemption-handoff.md §16).
//
// RED-FIRST: written and run against the pre-change tree, where every test
// here reds because SeedHQApps registers neither `marketing` nor
// `marketing-offline-override`. Evidence in
// .night-crew/runs/2026-09-05-autonomous/card1-red.log.
//
// Three properties pinned:
//  1. the two rows are registered (the access editor and /me/apps render from
//     hq_apps — an unregistered surface is ungrantable through any UI);
//  2. the grant seed fires on FIRST REGISTRATION ONLY — SeedHQApps runs on
//     every server startup, and a restart must never resurrect a grant an
//     operator revoked;
//  3. the entitlement is NARROW at the gate: holding the `marketing` app
//     grant must NOT satisfy a check on `marketing-offline-override`. The
//     umbrella rider ("App grant = All tabs granted") is about TABS; the
//     override is an entitlement, and the enforcing card must mount it with
//     no umbrella slug.

// marketingRoleGrants returns the sorted role grants held on a slug.
func marketingRoleGrants(t *testing.T, slug string) []string {
	t.Helper()
	rows, err := permPool.Query(t.Context(), `
		SELECT p.role FROM app_permissions p
		JOIN hq_apps a ON a.id = p.app_id
		WHERE a.slug = $1 AND p.role IS NOT NULL`, slug)
	if err != nil {
		t.Fatalf("query role grants for %s: %v", slug, err)
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			t.Fatalf("scan role grant: %v", err)
		}
		roles = append(roles, r)
	}
	sort.Strings(roles)
	return roles
}

// requireMarketingRows makes the two rows present regardless of what an
// earlier test in this package did (SeedHQApps upserts; rows may have been
// deleted by the first-registration test's clean-slate step).
func requireMarketingRows(t *testing.T) {
	t.Helper()
	if err := db.SeedHQApps(t.Context(), permPool); err != nil {
		t.Fatalf("SeedHQApps: %v", err)
	}
	for _, slug := range []string{"marketing", "marketing-offline-override"} {
		var n int
		if err := permPool.QueryRow(t.Context(),
			`SELECT count(*) FROM hq_apps WHERE slug = $1 AND enabled = true`, slug,
		).Scan(&n); err != nil {
			t.Fatalf("query hq_apps: %v", err)
		}
		if n != 1 {
			t.Fatalf("hq_apps slug %q: got %d enabled rows, want 1 "+
				"(SeedHQApps must register the marketing surfaces — §16)", slug, n)
		}
	}
}

func TestSeedHQApps_RegistersMarketingSurfaces(t *testing.T) {
	requireDB(t)
	requireMarketingRows(t)
}

func TestSeedHQApps_MarketingGrants_SeedOnFirstRegistrationOnly(t *testing.T) {
	requireDB(t)
	ctx := t.Context()

	// Clean slate for the two rows — the FK cascade takes their grants with them.
	if _, err := permPool.Exec(ctx,
		`DELETE FROM hq_apps WHERE slug IN ('marketing','marketing-offline-override')`); err != nil {
		t.Fatalf("clean slate: %v", err)
	}

	// FIRST registration: rows AND grants appear.
	if err := db.SeedHQApps(ctx, permPool); err != nil {
		t.Fatalf("SeedHQApps (first registration): %v", err)
	}
	if got, want := marketingRoleGrants(t, "marketing"), []string{"admin", "manager", "team_member"}; !equalStrings(got, want) {
		t.Errorf("marketing role grants after first registration: got %v, want %v "+
			"(§16: scan/redeem min role team_member — the APP grant seeds to all three roles)", got, want)
	}
	if got, want := marketingRoleGrants(t, "marketing-offline-override"), []string{"admin"}; !equalStrings(got, want) {
		t.Errorf("offline_override role grants after first registration: got %v, want %v "+
			"(fork #12: seeded true for admin users ONLY — everyone else by explicit grant)", got, want)
	}

	// Operator revokes; the server restarts (SeedHQApps runs again). The grants
	// must STAY revoked — the seed is registration-gated, not an upsert.
	if _, err := permPool.Exec(ctx, `
		DELETE FROM app_permissions WHERE app_id IN
		  (SELECT id FROM hq_apps WHERE slug IN ('marketing','marketing-offline-override'))`); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if err := db.SeedHQApps(ctx, permPool); err != nil {
		t.Fatalf("SeedHQApps (restart after revocation): %v", err)
	}
	if got := marketingRoleGrants(t, "marketing"); len(got) != 0 {
		t.Errorf("marketing grants resurrected by a restart: got %v, want none "+
			"(the seed must fire on first registration only)", got)
	}
	if got := marketingRoleGrants(t, "marketing-offline-override"); len(got) != 0 {
		t.Errorf("offline_override grants resurrected by a restart: got %v, want none", got)
	}

	// Restore the seeded state for any later reader in this package.
	if _, err := permPool.Exec(ctx,
		`DELETE FROM hq_apps WHERE slug IN ('marketing','marketing-offline-override')`); err != nil {
		t.Fatalf("restore clean: %v", err)
	}
	if err := db.SeedHQApps(ctx, permPool); err != nil {
		t.Fatalf("SeedHQApps (restore): %v", err)
	}
}

// The gate-level narrowness proof: the marketing APP grant does not satisfy
// the entitlement, and the explicit entitlement grant does. Mounted the way
// the enforcing card must mount it — RequirePermission(pool,
// "marketing-offline-override") with NO umbrella slug.
func TestOfflineOverride_EntitlementNotImpliedByAppGrant(t *testing.T) {
	requireDB(t)
	resetGrants(t)
	requireMarketingRows(t)

	u := mkUser(t, "override-narrow", []string{"team_member"})
	grantRole(t, "marketing", "team_member")

	status, body, reached := callNarrowGate(t, u, "marketing-offline-override")
	if status != http.StatusForbidden || reached {
		t.Fatalf("app grant alone must NOT open the entitlement: got status=%d reached=%v, want 403/false", status, reached)
	}
	if !strings.Contains(body, `"missing_grant":"marketing-offline-override"`) {
		t.Errorf("403 envelope must name the narrow entitlement slug, got: %s", body)
	}

	grantUser(t, "marketing-offline-override", u.ID)
	status, _, reached = callNarrowGate(t, u, "marketing-offline-override")
	if status != http.StatusOK || !reached {
		t.Fatalf("explicit entitlement grant must open the gate: got status=%d reached=%v, want 200/true", status, reached)
	}
}

// callNarrowGate is callGate without an umbrella — the entitlement's mount shape.
func callNarrowGate(t *testing.T, user *User, slug string) (int, string, bool) {
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
	RequirePermission(permPool, slug)(next).ServeHTTP(rec, req)
	return rec.Code, strings.TrimSpace(rec.Body.String()), reached
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
