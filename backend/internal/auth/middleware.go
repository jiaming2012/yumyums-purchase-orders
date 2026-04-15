package auth

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/config"
)

type contextKey string

const CtxKeyUser contextKey = "user"

// UserFromContext extracts the authenticated user from request context.
// Returns nil if no user is set (should not happen behind middleware).
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(CtxKeyUser).(*User)
	return u
}

// Middleware validates the hq_session cookie, looks up the session in DB,
// and attaches the User to request context. Returns 401 if invalid.
func Middleware(pool *pgxpool.Pool, superadmins map[string]config.SuperadminEntry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("hq_session")
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			tokenHash := HashToken(cookie.Value)
			user, err := LookupSession(r.Context(), pool, tokenHash, superadmins)
			if err != nil || user == nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), CtxKeyUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
