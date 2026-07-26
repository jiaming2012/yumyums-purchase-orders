package sync

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// ═══════════════════════════════════════════════════════════════════════════
// The HQ → Supabase JWT bridge (card `sync-jwt-bridge-endpoint`, backend half)
// ═══════════════════════════════════════════════════════════════════════════
//
// HQ already has authentication (the `hq_session` cookie → `sessions` table →
// `auth.User`) and already has authorization (`app_permissions` ⋈ `hq_apps`,
// read by auth.RequirePermission). Supabase's substrate — PostgREST and
// Realtime — wants an HS256 JWT. This file is the ONLY thing between them.
//
// It is deliberately a *mapping*, not a model. Every claim below is a
// projection of a value HQ already stores. No new grant or permission concept
// is introduced here; introducing one is this card's park trigger, because it
// would be a product question.
//
// ── Why stdlib only ────────────────────────────────────────────────────────
// The spike (`.night-crew/qa/spike-supabase/mintjwt/main.go`) proved that both
// PostgREST v12.2.12 and Realtime v2.34.47 accept a token built from
// crypto/hmac + crypto/sha256 + encoding/base64 + encoding/json and nothing
// else. An HS256 JWT is a base64url-joined header, payload and HMAC-SHA256 —
// that is the entire specification we need. `backend/go.mod` is a HARD
// constraint for the whole night-crew cycle; reaching for github.com/golang-jwt
// to "do this properly" would break it in exchange for nothing.
//
// ── Why NOT auth.uid() ─────────────────────────────────────────────────────
// 🛑 Read this before writing any RLS policy against these tokens.
//
// This stack runs no GoTrue, so the `auth` schema ships only three functions
// (`email`, `role`, `uid`) and `auth.uid()` is:
//
//	select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid;
//
// That is the LEGACY SINGULAR GUC, which PostgREST populates only when
// PGRST_DB_USE_LEGACY_GUCS=true. This stack sets it "false", so auth.uid()
// returns NULL and the policy silently matches nothing — `HTTP 200 []`, which
// reads as "this user has no data" rather than as a broken policy.
//
// (auth.uid() also casts to `uuid`, so a non-UUID `sub` would RAISE. On this
// stack that raise is unreachable — nullif() is handed NULL by the GUC problem
// above and `NULL::uuid` is legal, so the cast never sees a bad string. It
// becomes live only if someone flips PGRST_DB_USE_LEGACY_GUCS to true, at
// which point the symptom changes from a silent empty result to a loud 500.
// Worth knowing; not what happens today.)
//
// Every copy-pasted policy from Supabase's hosted docs uses auth.uid() or
// auth.jwt() and will fail non-obviously here.
//
// Policies MUST read the PLURAL GUC directly:
//
//	current_setting('request.jwt.claims', true)::json ->> 'sub'
//
// See `.night-crew/qa/spike-supabase/sql/hq-bridge-policies.sql`, which does,
// and its `auth.uid()` negative control, which demonstrates the failure.
//
// ── Why hq_grants in the token is ADVISORY, not authoritative ──────────────
// 🛑 The sharpest edge in the whole bridge.
//
// A JWT's claims are frozen at mint time. A token minted at 09:00 still asserts
// the grants the user held at 09:00, even after an admin revokes them at 09:05.
// If RLS trusted the claim, revocation would not take effect until the token
// expired — a replay window measured in the token's TTL.
//
// So the claim is NOT the gate. The gate is `public.hq_grant_projection`, a
// live projection of HQ's `app_permissions` ⋈ `hq_apps` that lives in the sync
// database and that the policies join against on every row. Revoke the grant,
// delete the projection row, and the SAME unexpired token immediately stops
// seeing the rows. `hq_grants` is carried purely so the client can render its
// UI without a second round trip, and the suite has a variant
// (`stale grant claim is not load-bearing`) that proves the claim alone opens
// nothing.
//
// ── service_role ───────────────────────────────────────────────────────────
// 🛑 `service_role` has BYPASSRLS. It is a god-token. Sign refuses it — and
// refuses every other role too, because the guard is an ALLOWLIST on
// SupabaseRole rather than a denylist naming the roles we happened to think of
// (TestMint_RoleGuardIsAnAllowlist covers postgres, supabase_admin,
// authenticator and anon). `service_role` appears in this package exactly once
// more, in the test suite, as the control that proves the table was not empty.

// SupabaseRole is the Postgres role PostgREST SET ROLEs into for a request,
// read from the `role` claim. Every human HQ user maps to exactly one:
// `authenticated`. HQ's own role tiers (team_member / manager / admin /
// superadmin) are a DIFFERENT axis and travel in `hq_roles`; collapsing them
// onto Postgres roles would require inventing Postgres roles that do not exist
// in the stack, which is a permission model, not a mapping.
const SupabaseRole = "authenticated"

// ServiceRole is named here only so tests can refer to the BYPASSRLS role by
// name. Sign's guard is an ALLOWLIST on SupabaseRole and does not consult this
// constant — a denylist would be complete only by accident. See Sign.
const ServiceRole = "service_role"

// DefaultTokenTTL is deliberately short. The projection table makes grant
// revocation immediate, but identity revocation (a deleted session, a
// deactivated user) is only bounded by expiry, so expiry has to be small.
const DefaultTokenTTL = 15 * time.Minute

// ErrSecretNotConfigured is returned when HQ_SYNC_JWT_SECRET is unset. The
// endpoint turns it into 503, mirroring auth.ServiceTokenMiddleware: a
// misconfigured deploy fails closed, it does not silently mint unsigned junk.
var ErrSecretNotConfigured = errors.New("sync jwt bridge: signing secret not configured")

// ErrServiceRoleRefused is returned for ANY `role` claim other than
// SupabaseRole — not only `service_role`. The name is kept because
// `service_role` is the motivating case (it holds BYPASSRLS, so one leaked
// token defeats every policy at once), but the guard is an allowlist: see
// Sign. `postgres`, `supabase_admin`, `authenticator` and `anon` are refused
// by the same clause, and would be even if this stack's role graph changed
// underneath us.
var ErrServiceRoleRefused = errors.New("sync jwt bridge: refusing to mint a token for any role other than " + SupabaseRole)

// Claims is the exact claim set the bridge emits. Every field is annotated with
// the HQ column it projects, because "which HQ thing is this?" is the only
// question a reader of an RLS policy actually has.
type Claims struct {
	// ── Standard JWT / PostgREST claims ──────────────────────────────────
	Sub  string `json:"sub"`  // users.id — the HQ user id, verbatim
	Role string `json:"role"` // always SupabaseRole; PostgREST SET ROLEs into it
	Exp  int64  `json:"exp"`  // enforced by PostgREST (401 PGRST301 "JWT expired")
	Iat  int64  `json:"iat"`

	// ── HQ projections. Namespaced hq_* so they can never collide with a
	//    claim PostgREST, Realtime or a future GoTrue assigns meaning to. ──

	// Email projects users.email. Convenience for logging and support; no
	// policy in this card reads it.
	Email string `json:"email,omitempty"`

	// HQRoles projects users.roles — team_member / manager / admin, or the
	// synthetic ["superadmin"] LookupSession substitutes for a config
	// superadmin. Roles and grants are separate axes in HQ (see the AXIS test
	// in tests/grant-enforcement-parity.spec.js); they stay separate here.
	HQRoles []string `json:"hq_roles"`

	// HQGrants projects the app slugs this user holds a DIRECT grant on at
	// mint time — auth.RequirePermission's predicate evaluated once per
	// enabled app.
	//
	// 🛑 NOT "the same answer RequirePermission gives." An earlier version of
	// this comment claimed that and it was wrong. RequirePermission is
	// RequirePermission(pool, grantSlug, umbrellaSlugs...) and matches
	// `a.slug = ANY(candidate_set)` — the umbrella position is REAL and in use
	// (main.go:628, 642, 652). A user holding `inventory` gets
	// hq_grants = [inventory, ...] here, while
	// RequirePermission("inventory-trends", "inventory") returns TRUE. So this
	// list is NARROWER than what the user can actually reach: a launcher
	// rendered naively from it would hide `inventory-trends` and
	// `inventory-cost`, two surfaces the user does reach.
	//
	// It errs CLOSED, so there is no security consequence — but a client must
	// expand umbrellas itself rather than treat this as the reachable set.
	// What the advisory list SHOULD contain for umbrella slugs is a design
	// call the run did not make; it is recorded as D-6 in
	// .night-crew/runs/2026-07-26-autonomous/DECISIONS-NEEDED.md.
	//
	// 🛑 ADVISORY ONLY. Not the authorization gate. See the file header.
	HQGrants []string `json:"hq_grants"`

	// HQSid projects sessions.token_hash — the session this token was minted
	// from. Carried so a future card can bound identity revocation without
	// inventing anything; no policy in this card reads it.
	HQSid string `json:"hq_sid,omitempty"`
}

// b64 is base64url WITHOUT padding, which is what JWS compact serialization
// requires. StdEncoding here produces a token every verifier rejects.
func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// signHS256 builds a complete HS256 JWT from any claims value. This is the
// whole of the "JWT library" this backend needs.
func signHS256(claims any, secret string) (string, error) {
	if secret == "" {
		return "", ErrSecretNotConfigured
	}
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signing := b64(header) + "." + b64(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signing))
	return signing + "." + b64(mac.Sum(nil)), nil
}

// Sign mints a token from an explicit Claims value. Exported for the attack
// suite, which needs to construct deliberately malformed claim sets that
// MintForUser would never produce.
func Sign(c Claims, secret string) (string, error) {
	// 🛑 ALLOWLIST, not a denylist. This was `c.Role == ServiceRole` — a
	// denylist naming the one role we were worried about. G6 minted tokens
	// out-of-band for `postgres`, `supabase_admin`, `authenticator` and `anon`
	// and showed the denylist was adequate only ACCIDENTALLY: it holds today
	// because this stack's `authenticator` has no membership in those roles,
	// not because the guard covers them. That is a property of the stack's
	// role graph, which no test here owns and any operator can change.
	//
	// `role` is the claim PostgREST SET ROLEs into. There is exactly one value
	// this bridge is ever allowed to emit, so say that instead of enumerating
	// the values it is not.
	if c.Role != SupabaseRole {
		return "", ErrServiceRoleRefused
	}
	return signHS256(c, secret)
}

// GrantedSlugs returns every enabled app slug this user holds a DIRECT grant
// on, using auth.RequirePermission's own predicate:
//
//	superadmin ∨ a role grant ∨ an individual user grant
//
// Superadmins get every enabled slug, mirroring RequirePermission's superadmin
// bypass and me.queryAllApps — a superadmin who saw fewer slugs here than in
// the launcher would be a new, quieter permission model.
//
// 🛑 It does NOT expand umbrellas, and the result is therefore a SUBSET of
// what RequirePermission would admit. RequirePermission takes
// (grantSlug, umbrellaSlugs...) and a caller holding the umbrella passes the
// narrow gate; this function is asked about one slug at a time with no
// umbrella context, so `inventory` here does not imply `inventory-trends`
// even though the mounted gate says it does. Errs closed. See the HQGrants
// field comment and D-6.
//
// This is a read of existing tables. It adds no concept.
func GrantedSlugs(ctx context.Context, pool *pgxpool.Pool, user *auth.User) ([]string, error) {
	if user == nil {
		return nil, errors.New("sync jwt bridge: nil user")
	}
	var (
		rows pgx.Rows
		err  error
	)
	if user.IsSuperadmin {
		rows, err = pool.Query(ctx,
			`SELECT slug FROM hq_apps WHERE enabled = true ORDER BY slug`)
	} else {
		rows, err = pool.Query(ctx, `
			SELECT DISTINCT a.slug
			FROM app_permissions p
			JOIN hq_apps a ON a.id = p.app_id
			WHERE a.enabled = true
			  AND (p.role = ANY($1) OR p.user_id = $2)
			ORDER BY a.slug`, user.Roles, user.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("sync jwt bridge: grant lookup: %w", err)
	}
	defer rows.Close()

	slugs := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("sync jwt bridge: grant scan: %w", err)
		}
		slugs = append(slugs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("sync jwt bridge: grant rows: %w", err)
	}
	return slugs, nil
}

// MintForUser builds and signs the bridge token for an already-authenticated
// HQ user. `sid` is the caller's sessions.token_hash (auth.HashToken of the
// cookie), or "" when unavailable.
//
// It fails closed on every axis: no secret → error; nil user → error; a grant
// lookup that errors → error, never a token with an empty grant list, because
// an empty list would look exactly like "this user legitimately holds nothing."
func MintForUser(ctx context.Context, pool *pgxpool.Pool, user *auth.User, sid, secret string, ttl time.Duration) (string, Claims, error) {
	var zero Claims
	if secret == "" {
		return "", zero, ErrSecretNotConfigured
	}
	if user == nil {
		return "", zero, errors.New("sync jwt bridge: nil user")
	}
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}

	slugs, err := GrantedSlugs(ctx, pool, user)
	if err != nil {
		return "", zero, err
	}

	roles := user.Roles
	if roles == nil {
		roles = []string{}
	}

	now := time.Now()
	c := Claims{
		Sub:      user.ID,
		Role:     SupabaseRole, // never anything else — see ErrServiceRoleRefused
		Exp:      now.Add(ttl).Unix(),
		Iat:      now.Unix(),
		Email:    user.Email,
		HQRoles:  roles,
		HQGrants: slugs,
		HQSid:    sid,
	}
	tok, err := Sign(c, secret)
	if err != nil {
		return "", zero, err
	}
	return tok, c, nil
}
