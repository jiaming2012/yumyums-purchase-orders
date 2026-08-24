package sync

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// SyncJWTSecretEnv names the env var holding the HS256 signing secret. It must
// equal the stack's JWT_SECRET — the single shared value that is simultaneously
// PostgREST's PGRST_JWT_SECRET and Realtime's API_JWT_SECRET.
const SyncJWTSecretEnv = "HQ_SYNC_JWT_SECRET"

// TokenResponse is the endpoint's success envelope.
//
// `grants` is echoed so the client can render its launcher without a second
// round trip. 🛑 It is ADVISORY. It is not what authorizes anything — RLS reads
// public.hq_grant_projection live, per row. Do not let a client treat this list
// as permission to do something; treat it as a hint about what to show.
type TokenResponse struct {
	Token     string   `json:"token"`
	ExpiresAt int64    `json:"expires_at"`
	Sub       string   `json:"sub"`
	Role      string   `json:"role"`
	Grants    []string `json:"grants"`
}

// TokenHandler mints the Supabase-compatible bridge token for the CALLER'S OWN
// session. There is no user-id parameter anywhere in its signature, and that is
// deliberate: an endpoint that could mint for an arbitrary subject would be an
// impersonation primitive one missing authorization check away from a breach.
// Identity comes only from the context the cookie middleware attached.
//
// MOUNT IT INSIDE THE COOKIE GROUP (after auth.Middleware). It is deliberately
// NOT behind auth.RequirePermission: this endpoint is access-resolution
// plumbing, the same category as /api/v1/me and /api/v1/me/apps, which
// tests/grant-enforcement-parity.spec.js already records as outside every app
// gate. It must serve an ungranted user — so it can hand them a token whose
// projection lets them reach nothing. Gating it behind a grant would be
// circular, and picking WHICH grant gates the sync bridge would be inventing a
// permission concept, which is this card's park trigger.
//
// Failure modes, all fail-closed:
//
//	503 {"error":"sync_bridge_not_configured"}  HQ_SYNC_JWT_SECRET unset.
//	                                            Mirrors auth.ServiceTokenMiddleware:
//	                                            a misconfigured deploy must not
//	                                            silently mint unverifiable tokens.
//	401 {"error":"unauthorized"}                no user on context (defence in
//	                                            depth; only reachable if mounted
//	                                            outside the cookie group)
//	500 {"error":"internal_error"}              grant lookup failed. NEVER a
//	                                            token with an empty grant list —
//	                                            that would be indistinguishable
//	                                            from a user who legitimately
//	                                            holds nothing.
func TokenHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		secret := os.Getenv(SyncJWTSecretEnv)
		if secret == "" {
			writeJSON(w, http.StatusServiceUnavailable,
				map[string]string{"error": "sync_bridge_not_configured"})
			return
		}

		user := auth.UserFromContext(r.Context())
		if user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		// sid projects sessions.token_hash for the session this call rode in
		// on. Best-effort: absent cookie simply omits the claim rather than
		// failing the mint, since no policy in this card reads it.
		sid := ""
		if c, err := r.Cookie("hq_session"); err == nil {
			sid = auth.HashToken(c.Value)
		}

		tok, claims, err := MintForUser(r.Context(), pool, user, sid, secret, DefaultTokenTTL)
		if err != nil {
			// The secret is never logged or echoed, here or anywhere.
			slog.Error("sync jwt bridge mint failed", "error", err, "user_id", user.ID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}

		// A bridge token is a bearer credential with a short life. It must
		// never be stored by a proxy or replayed from a browser cache.
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, TokenResponse{
			Token:     tok,
			ExpiresAt: claims.Exp,
			Sub:       claims.Sub,
			Role:      claims.Role,
			Grants:    claims.HQGrants,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
