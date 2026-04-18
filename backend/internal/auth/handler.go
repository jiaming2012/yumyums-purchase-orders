package auth

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/config"
)

// LoginHandler handles POST /api/v1/auth/login.
// On valid credentials, creates a session and sets an httpOnly cookie.
// Returns 401 with {"error":"invalid_credentials"} on bad credentials.
func LoginHandler(pool *pgxpool.Pool, superadmins map[string]config.SuperadminEntry, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"validation_error"}`))
			return
		}

		user, err := AuthenticateUser(r.Context(), pool, body.Email, body.Password, superadmins)
		if err != nil {
			log.Printf("authenticate user error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal_error"}`))
			return
		}
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid_credentials"}`))
			return
		}

		rawToken, err := CreateSession(r.Context(), pool, user.ID)
		if err != nil {
			log.Printf("create session error: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal_error"}`))
			return
		}

		// Set httpOnly cookie — token never exposed in response body
		sameSite := http.SameSiteStrictMode
		if !secureCookie {
			sameSite = http.SameSiteLaxMode
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "hq_session",
			Value:    rawToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   secureCookie,
			SameSite: sameSite,
			// No MaxAge or Expires — indefinite session per D-03
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]any{
				"id":           user.ID,
				"email":        user.Email,
				"display_name": user.DisplayName,
				"roles":        user.Roles,
			},
		})
	}
}

// LogoutHandler handles POST /api/v1/auth/logout.
// Deletes the session and clears the cookie. Idempotent — returns 204 if no cookie.
func LogoutHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("hq_session")
		if err != nil {
			// No cookie — already logged out
			w.WriteHeader(http.StatusNoContent)
			return
		}

		tokenHash := HashToken(cookie.Value)
		if err := DeleteSessionByHash(r.Context(), pool, tokenHash); err != nil {
			log.Printf("delete session error: %v", err)
		}

		// Clear the cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "hq_session",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusNoContent)
	}
}
