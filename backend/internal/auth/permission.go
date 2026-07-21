package auth

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RequirePermission is the per-surface authorization gate (design
// `prove-surface-gating-and-endpoints.md` §1.3 station 1, Option (i) §1.4).
//
// Mount it INSIDE the cookie group, i.e. after Middleware, so a user is already
// on the context. It answers exactly one question:
//
//	may this user reach the surface identified by grantSlug?
//
// and passes iff:
//
//	superadmin
//	  ∨ a grant on grantSlug          (the narrow, per-tab grant)
//	  ∨ a grant on any umbrellaSlug   (the whole-app grant)
//
// The umbrella disjunct is the operator's signed rider (§8 amendment 1,
// verbatim: "App grant = All tabs granted. They should not be considered
// separate objects."). It REPLACES the §1.5 draft reading in which a whole-app
// grant did not imply its tab grants — a user granted `inventory` reaches every
// gated Inventory tab without a second toggle.
//
// Superadmins bypass the grant check deliberately, mirroring me.queryAllApps
// (§1.2 rule 4, §4 flag 5): /me/apps hands superadmins every enabled app, so a
// superadmin would otherwise see a tab whose endpoint 403s.
//
// The denial envelope is deliberately DISTINCT from Middleware's 401
// (§1.2 rule 3) so the client can tell "log in again" from "you lack this
// grant" — the former is recoverable by the user, the latter only by an admin:
//
//	401 {"error":"unauthorized"}
//	403 {"error":"forbidden","missing_grant":"inventory-trends"}
//
// missing_grant always names the NARROW slug, never the umbrella: it is the
// grant an admin would go issue, and naming the umbrella would advise a wider
// grant than the surface needs.
func RequirePermission(pool *pgxpool.Pool, grantSlug string, umbrellaSlugs ...string) func(http.Handler) http.Handler {
	// The candidate set is fixed at mount time, not per request.
	slugs := append([]string{grantSlug}, umbrellaSlugs...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := UserFromContext(r.Context())
			if user == nil {
				// Defence in depth: only reachable if this middleware is mounted
				// outside the cookie group. Never fall through to the handler.
				writeJSONStatus(w, http.StatusUnauthorized, `{"error":"unauthorized"}`)
				return
			}

			if user.IsSuperadmin {
				next.ServeHTTP(w, r)
				return
			}

			var ok bool
			err := pool.QueryRow(r.Context(), `
				SELECT EXISTS (
					SELECT 1
					FROM app_permissions p
					JOIN hq_apps a ON a.id = p.app_id
					WHERE a.slug = ANY($1)
					  AND a.enabled = true
					  AND (p.role = ANY($2) OR p.user_id = $3)
				)`, slugs, user.Roles, user.ID).Scan(&ok)
			if err != nil {
				slog.Error("RequirePermission grant lookup failed",
					"error", err, "grant", grantSlug, "user_id", user.ID)
				// Fail CLOSED. A database hiccup must not open a gated surface.
				writeJSONStatus(w, http.StatusInternalServerError, `{"error":"internal_error"}`)
				return
			}

			if !ok {
				writeJSONStatus(w, http.StatusForbidden,
					`{"error":"forbidden","missing_grant":`+jsonString(grantSlug)+`}`)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeJSONStatus(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(body))
}

// jsonString quotes a slug for embedding in the hand-built envelope above.
// Slugs are developer-authored constants, but this keeps the envelope
// well-formed if one ever grows a quote.
func jsonString(s string) string {
	out := make([]byte, 0, len(s)+2)
	out = append(out, '"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\\':
			out = append(out, '\\', c)
		default:
			out = append(out, c)
		}
	}
	return string(append(out, '"'))
}
