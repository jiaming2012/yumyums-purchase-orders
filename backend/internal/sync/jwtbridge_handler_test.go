package sync

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yumyums/hq/internal/auth"
)

// withUser builds a request carrying an authenticated user on its context, the
// way auth.Middleware would.
func withUser(u *auth.User) *http.Request {
	r := httptest.NewRequest("POST", "/api/v1/sync/token", nil)
	if u == nil {
		return r
	}
	return r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUser, u))
}

// TestTokenHandler_FailsClosedWithoutSecret — an unset HQ_SYNC_JWT_SECRET is a
// misconfigured deploy, and it must 503 rather than mint something no verifier
// will accept. Mirrors auth.ServiceTokenMiddleware's 503 precedent.
func TestTokenHandler_FailsClosedWithoutSecret(t *testing.T) {
	t.Setenv(SyncJWTSecretEnv, "")
	w := httptest.NewRecorder()
	TokenHandler(nil)(w, withUser(&auth.User{ID: "u-1", Roles: []string{"team_member"}}))

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sync_bridge_not_configured") {
		t.Errorf("body = %s, want the sync_bridge_not_configured envelope", w.Body.String())
	}
}

// TestTokenHandler_RejectsAnonymous — defence in depth. Only reachable if the
// route were ever mounted outside the cookie group; it must never fall through
// to a mint.
func TestTokenHandler_RejectsAnonymous(t *testing.T) {
	t.Setenv(SyncJWTSecretEnv, "unit-test-secret")
	w := httptest.NewRecorder()
	TokenHandler(nil)(w, withUser(nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", w.Code, w.Body.String())
	}
}

// TestTokenHandler_MintsForTheCallerAndNobodyElse is the impersonation guard.
// The handler takes NO user-id parameter, so the only thing this can assert is
// the invariant that makes that safe: whatever the request body or query says,
// the `sub` claim is the CONTEXT user's id.
func TestTokenHandler_MintsForTheCallerAndNobodyElse(t *testing.T) {
	pool := hqTestPool(t) // skips when HQ's test DB is unavailable
	t.Setenv(SyncJWTSecretEnv, "unit-test-secret")

	caller := &auth.User{
		ID: "00000000-0000-0000-0000-000000000000", Email: "caller@yumyums.kitchen",
		Roles: []string{"team_member"},
	}

	// A request that tries every obvious impersonation channel at once.
	r := httptest.NewRequest("POST",
		"/api/v1/sync/token?user_id=victim&sub=victim",
		strings.NewReader(`{"user_id":"victim","sub":"victim","role":"service_role"}`))
	r.Header.Set("Content-Type", "application/json")
	r = r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUser, caller))

	w := httptest.NewRecorder()
	TokenHandler(pool)(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store — a bearer credential must not be "+
			"cacheable by a proxy or replayable from a browser cache", cc)
	}

	var got TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Sub != caller.ID {
		t.Errorf("sub = %q, want the CONTEXT user %q — the endpoint minted for a subject "+
			"supplied by the request, which is an impersonation primitive", got.Sub, caller.ID)
	}
	if got.Role != SupabaseRole {
		t.Errorf("role = %q, want %q — never anything else, and never service_role",
			got.Role, SupabaseRole)
	}

	// Decode the token itself; the envelope could agree while the claims do not.
	parts := strings.Split(got.Token, ".")
	if len(parts) != 3 {
		t.Fatalf("token is not a 3-segment JWT: %q", got.Token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("unmarshal claims: %v", err)
	}
	if claims["sub"] != caller.ID {
		t.Errorf("token sub claim = %v, want %q", claims["sub"], caller.ID)
	}
	if claims["role"] != SupabaseRole {
		t.Errorf("token role claim = %v, want %q (NEVER service_role)", claims["role"], SupabaseRole)
	}

	// Expiry must be short. Grant revocation is immediate via the projection,
	// but identity revocation (deleted session, deactivated user) is bounded
	// only by exp — so exp has to be small.
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatalf("exp claim missing or not numeric: %v", claims["exp"])
	}
	if lifetime := time.Until(time.Unix(int64(exp), 0)); lifetime > time.Hour {
		t.Errorf("token lifetime = %v, which is too long. Identity revocation is bounded "+
			"only by expiry; a long-lived bridge token is a long replay window.", lifetime)
	}
}
