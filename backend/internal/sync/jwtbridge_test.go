package sync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// Unit-level guards on the mint itself. These need no stack and no DB, except
// the two grant tests at the bottom, which use HQ's own test database via the
// connect-or-skip idiom.

// TestSign_IsAValidHS256JWT verifies the hand-rolled mint against the JWS
// compact serialization by hand — three base64url segments, no padding, and an
// HMAC-SHA256 over "header.payload". If this passes, no JWT library is needed;
// that is the entire argument for HARD constraint 1 (backend/go.mod untouched).
func TestSign_IsAValidHS256JWT(t *testing.T) {
	const secret = "unit-test-secret"
	c := Claims{
		Sub: "u-1", Role: SupabaseRole, Iat: 1700000000, Exp: 1700003600,
		Email: "u1@example.com", HQRoles: []string{"team_member"}, HQGrants: []string{"operations"},
	}
	tok, err := Sign(c, secret)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 segments, got %d (%q)", len(parts), tok)
	}
	if strings.ContainsAny(tok, "=+/") {
		t.Errorf("token contains padded/standard base64 characters; JWS requires base64url "+
			"WITHOUT padding — every verifier rejects the padded form: %q", tok)
	}

	var hdr map[string]string
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decode header: %v", err)
	}
	if err := json.Unmarshal(raw, &hdr); err != nil {
		t.Fatalf("unmarshal header: %v", err)
	}
	if hdr["alg"] != "HS256" || hdr["typ"] != "JWT" {
		t.Errorf("header = %v, want alg=HS256 typ=JWT", hdr)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if parts[2] != want {
		t.Errorf("signature mismatch:\n got %s\nwant %s", parts[2], want)
	}
}

// TestSign_ClaimsRoundTrip pins the wire names. RLS policies are written
// against these strings; renaming a JSON tag silently breaks every policy in
// sql/hq-bridge-policies.sql with no compile error anywhere.
func TestSign_ClaimsRoundTrip(t *testing.T) {
	tok, err := Sign(Claims{
		Sub: "u-1", Role: SupabaseRole, Iat: 1, Exp: 2,
		Email: "e@x", HQRoles: []string{"admin"}, HQGrants: []string{"inventory"}, HQSid: "sid-1",
	}, "s")
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.Split(tok, ".")[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	for _, key := range []string{"sub", "role", "iat", "exp", "email", "hq_roles", "hq_grants", "hq_sid"} {
		if _, ok := got[key]; !ok {
			t.Errorf("claim %q missing from the payload — RLS policies read these names by string; "+
				"renaming one breaks every policy with no compile error", key)
		}
	}
	if got["role"] != SupabaseRole {
		t.Errorf("role claim = %v, want %q", got["role"], SupabaseRole)
	}
}

// TestMint_NeverMintsServiceRole is the product invariant behind W1's warning:
// service_role is a BYPASSRLS god-token and must never be mintable by anything
// a client can reach. The one deliberate bypass is the test control in
// jwtbridge_rls_test.go, which goes around Sign() on purpose and says so.
func TestMint_NeverMintsServiceRole(t *testing.T) {
	_, err := Sign(Claims{Sub: "attacker", Role: ServiceRole, Exp: 1, Iat: 0}, "s")
	if !errors.Is(err, ErrServiceRoleRefused) {
		t.Fatalf("Sign(service_role) = %v, want ErrServiceRoleRefused. A BYPASSRLS token "+
			"reachable from a client-facing path defeats every policy at once.", err)
	}
}

// TestMint_RoleGuardIsAnAllowlist — the guard used to be a DENYLIST naming
// `service_role` alone. G6 minted tokens out-of-band for the roles below and
// showed the denylist was adequate only accidentally: it holds because this
// stack's `authenticator` happens to have no membership in them, which is a
// property of the stack's role graph rather than of this code. An allowlist
// does not depend on that, so it cannot be invalidated by an operator adding
// a GRANT somewhere this package never sees.
func TestMint_RoleGuardIsAnAllowlist(t *testing.T) {
	for _, role := range []string{
		ServiceRole, "postgres", "supabase_admin", "authenticator", "anon",
		"", "AUTHENTICATED", "authenticated ", "hq_superuser",
	} {
		_, err := Sign(Claims{Sub: "attacker", Role: role, Exp: 1, Iat: 0}, "s")
		if !errors.Is(err, ErrServiceRoleRefused) {
			t.Errorf("Sign(role=%q) = %v, want refusal. The guard is an allowlist: %q is the "+
				"only role this bridge may ever emit.", role, err, SupabaseRole)
		}
	}
	// And the one permitted value still works, or the allowlist has swallowed
	// the feature instead of guarding it.
	if _, err := Sign(Claims{Sub: "u", Role: SupabaseRole, Exp: 1, Iat: 0}, "s"); err != nil {
		t.Errorf("Sign(role=%q) = %v, want success", SupabaseRole, err)
	}
}

// TestMint_FailsClosedWithoutSecret mirrors auth.ServiceTokenMiddleware: an
// unset secret is a misconfigured deploy and must fail closed, never emit an
// unverifiable token.
func TestMint_FailsClosedWithoutSecret(t *testing.T) {
	if _, err := Sign(Claims{Sub: "u", Role: SupabaseRole}, ""); !errors.Is(err, ErrSecretNotConfigured) {
		t.Errorf("Sign with empty secret = %v, want ErrSecretNotConfigured", err)
	}
	if _, _, err := MintForUser(context.Background(), nil,
		&auth.User{ID: "u"}, "", "", time.Minute); !errors.Is(err, ErrSecretNotConfigured) {
		t.Errorf("MintForUser with empty secret = %v, want ErrSecretNotConfigured", err)
	}
}

// TestMintForUser_RejectsNilUser — no anonymous minting path, ever.
func TestMintForUser_RejectsNilUser(t *testing.T) {
	if _, _, err := MintForUser(context.Background(), nil, nil, "", "secret", time.Minute); err == nil {
		t.Error("MintForUser(nil user) succeeded; it must never mint for an unidentified caller")
	}
}

// ── Grant mapping against HQ's own database ────────────────────────────────

func hqTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("DB_TEST_URL / TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("DB_TEST_URL not reachable (connect failed): %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("DB_TEST_URL not reachable (ping failed): %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// TestGrantedSlugs_MatchesRequirePermission is the anti-drift guard on the
// grant mapping. GrantedSlugs must answer, for every enabled app at once,
// exactly what auth.RequirePermission answers for one named app — otherwise
// the token would carry an entitlement list that disagrees with the gate HQ's
// own endpoints enforce, which is a second permission model by accident.
//
// 🛑 THIS TEST SEEDS ITS OWN GRANTS, AND THAT IS NOT OPTIONAL.
//
// The first version of this test did not, and it was VACUOUS in the very run
// that gated the card. `app_permissions` was EMPTY in hq_test_b1, so all 11
// enabled slugs compared false-against-false and the test passed on eleven
// agreements of absence. G6 falsified it in one move: replacing GrantedSlugs's
// body with `return []string{}, nil` still passed. An anti-drift guard that
// cannot detect the drift it exists to catch is not a guard — it is a green
// light wired to nothing.
//
// Two things prevent a recurrence, and both are load-bearing:
//
//  1. The fixture below seeds a REAL role grant and a REAL individual grant,
//     so the comparison has true-values in it on any database, however fresh.
//  2. requirePositives asserts that a minimum number of slugs actually came
//     back TRUE. Agreement alone is not evidence; agreement that includes
//     observed positives is. With the stub, `operations` and `inventory` are
//     true from the predicate and absent from the token, so the per-slug loop
//     fails — the guard now bites.
//
// The two grants are deliberately of DIFFERENT KINDS because GrantedSlugs
// resolves them through different disjuncts of the same predicate
// (`p.role = ANY($1) OR p.user_id = $2`). Seeding only one kind would leave
// the other unproven.
func TestGrantedSlugs_MatchesRequirePermission(t *testing.T) {
	pool := hqTestPool(t)
	ctx := context.Background()

	var enabled []string
	rows, err := pool.Query(ctx, `SELECT slug FROM hq_apps WHERE enabled = true ORDER BY slug`)
	if err != nil {
		t.Skipf("hq_apps not present in this database: %v", err)
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		enabled = append(enabled, s)
	}
	rows.Close()
	if len(enabled) == 0 {
		t.Skip("no enabled hq_apps rows — nothing to compare")
	}

	// ── Fixture ──────────────────────────────────────────────────────────
	const (
		roleGrantSlug = "operations" // reached via the ROLE disjunct
		userGrantSlug = "inventory"  // reached via the INDIVIDUAL disjunct
	)
	for _, slug := range []string{roleGrantSlug, userGrantSlug} {
		if !slices.Contains(enabled, slug) {
			t.Skipf("enabled apps %v do not include %q — this fixture needs it to produce a "+
				"positive; update the fixture rather than letting the test go vacuous", enabled, slug)
		}
	}

	var userID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, roles, status)
		 VALUES ('grantparity-'||gen_random_uuid()||'@yumyums.test', ARRAY['team_member']::TEXT[], 'active')
		 RETURNING id::text`).Scan(&userID); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE removes the individual grant with the user.
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO app_permissions (app_id, role)
		SELECT id, 'team_member' FROM hq_apps WHERE slug = $1
		ON CONFLICT DO NOTHING`, roleGrantSlug); err != nil {
		t.Fatalf("seed role grant on %q: %v", roleGrantSlug, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			DELETE FROM app_permissions
			WHERE role = 'team_member'
			  AND app_id = (SELECT id FROM hq_apps WHERE slug = $1)`, roleGrantSlug)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO app_permissions (app_id, user_id)
		SELECT id, $2::uuid FROM hq_apps WHERE slug = $1
		ON CONFLICT DO NOTHING`, userGrantSlug, userID); err != nil {
		t.Fatalf("seed individual grant on %q: %v", userGrantSlug, err)
	}

	user := &auth.User{ID: userID, Roles: []string{"team_member"}}

	got, err := GrantedSlugs(ctx, pool, user)
	if err != nil {
		t.Fatalf("GrantedSlugs: %v", err)
	}

	// ── Per-slug agreement, plus a floor on observed positives ───────────
	// Evaluate RequirePermission's own EXISTS predicate verbatim and require
	// agreement. Any disagreement means the token disagrees with the gate, in
	// whichever direction.
	positives := []string{}
	for _, slug := range enabled {
		var ok bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM app_permissions p
				JOIN hq_apps a ON a.id = p.app_id
				WHERE a.slug = ANY($1)
				  AND a.enabled = true
				  AND (p.role = ANY($2) OR p.user_id = $3)
			)`, []string{slug}, user.Roles, user.ID).Scan(&ok)
		if err != nil {
			t.Fatalf("RequirePermission predicate for %q: %v", slug, err)
		}
		if ok {
			positives = append(positives, slug)
		}
		inToken := slices.Contains(got, slug)
		if ok != inToken {
			t.Errorf("slug %q: RequirePermission says reachable=%v but the token claim says %v. "+
				"The bridge must project the SAME grant answer HQ's own endpoints enforce; a "+
				"divergence here is a second permission model arriving by accident.", slug, ok, inToken)
		}
	}

	// 🛑 THE ANTI-VACUITY ASSERTION. Without it, "every slug agreed" is
	// satisfied by a database with no grants in it at all — which is exactly
	// how this test passed while proving nothing.
	if len(positives) < 2 {
		t.Fatalf("only %d slug(s) evaluated TRUE (%v). This test is VACUOUS below 2: "+
			"agreement between two functions that both say \"no\" to everything is not evidence. "+
			"The fixture seeds a role grant on %q and an individual grant on %q; if neither "+
			"produced a positive, the fixture is broken and must be repaired rather than the "+
			"floor lowered.", len(positives), positives, roleGrantSlug, userGrantSlug)
	}
	// And name them, so a positive arriving from some unrelated leftover row
	// cannot stand in for the two disjuncts this test exists to exercise.
	if !slices.Contains(got, roleGrantSlug) {
		t.Errorf("the ROLE-grant disjunct produced nothing: %q missing from %v", roleGrantSlug, got)
	}
	if !slices.Contains(got, userGrantSlug) {
		t.Errorf("the INDIVIDUAL-grant disjunct produced nothing: %q missing from %v", userGrantSlug, got)
	}
	t.Logf("non-vacuous: %d/%d slugs TRUE (%v)", len(positives), len(enabled), positives)
}

// TestGrantedSlugs_SuperadminGetsEveryEnabledApp mirrors RequirePermission's
// superadmin bypass and me.queryAllApps. A superadmin who saw fewer slugs here
// than in the launcher would be exactly that: a new, quieter permission model.
func TestGrantedSlugs_SuperadminGetsEveryEnabledApp(t *testing.T) {
	pool := hqTestPool(t)
	ctx := context.Background()

	var want []string
	rows, err := pool.Query(ctx, `SELECT slug FROM hq_apps WHERE enabled = true ORDER BY slug`)
	if err != nil {
		t.Skipf("hq_apps not present in this database: %v", err)
	}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan: %v", err)
		}
		want = append(want, s)
	}
	rows.Close()
	if len(want) == 0 {
		t.Skip("no enabled hq_apps rows")
	}

	got, err := GrantedSlugs(ctx, pool, &auth.User{
		ID: "00000000-0000-0000-0000-000000000000", IsSuperadmin: true, Roles: []string{"superadmin"},
	})
	if err != nil {
		t.Fatalf("GrantedSlugs: %v", err)
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("superadmin grants = %v, want every enabled app %v", got, want)
	}
}
