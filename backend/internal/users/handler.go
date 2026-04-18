package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// writeJSON writes a JSON response with the given HTTP status code.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data) //nolint:errcheck
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// isAdmin returns true if the user has admin or superadmin role.
func isAdmin(user *auth.User) bool {
	return user.Role == "admin" || user.IsSuperadmin
}

// ListUsersHandler handles GET /api/v1/users — admin only.
func ListUsersHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		users, err := ListUsers(r.Context(), pool)
		if err != nil {
			log.Printf("ListUsersHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if users == nil {
			users = []UserRow{}
		}
		writeJSON(w, http.StatusOK, users)
	}
}

// InviteHandler handles POST /api/v1/users/invite — admin only.
// Creates a user with status='invited' and returns an invite link with opaque token.
func InviteHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil || !isAdmin(user) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		var body struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
			Email     string `json:"email"`
			Role      string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}
		if strings.TrimSpace(body.FirstName) == "" || strings.TrimSpace(body.LastName) == "" || strings.TrimSpace(body.Email) == "" {
			writeError(w, http.StatusUnprocessableEntity, "validation_error")
			return
		}

		userID, err := CreateInvitedUser(r.Context(), pool, CreateUserInput{
			FirstName: body.FirstName,
			LastName:  body.LastName,
			Email:     body.Email,
			Role:      body.Role,
		})
		if err != nil {
			// Check for unique constraint violation on email
			if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
				writeError(w, http.StatusConflict, "email_already_exists")
				return
			}
			log.Printf("InviteHandler CreateInvitedUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		rawToken, hash, err := auth.GenerateToken()
		if err != nil {
			log.Printf("InviteHandler GenerateToken error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if err := InsertInviteToken(r.Context(), pool, userID, hash, 7); err != nil {
			log.Printf("InviteHandler InsertInviteToken error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		newUser, err := GetUser(r.Context(), pool, userID)
		if err != nil || newUser == nil {
			log.Printf("InviteHandler GetUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"user":        newUser,
			"invite_path": fmt.Sprintf("/login.html?token=%s", rawToken),
		})
	}
}

// UpdateUserHandler handles PATCH /api/v1/users/{id} — admin only.
// Supports partial updates of first_name, last_name, nickname, role.
// Returns 409 on nickname collision.
func UpdateUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		userID := chi.URLParam(r, "id")

		var body struct {
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
			Nickname  *string `json:"nickname"`
			Role      *string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}

		// Check nickname collision before updating
		if body.Nickname != nil && *body.Nickname != "" {
			colliding, err := CheckNicknameCollision(r.Context(), pool, *body.Nickname, userID)
			if err != nil {
				log.Printf("UpdateUserHandler CheckNicknameCollision error: %v", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			if colliding != "" {
				writeJSON(w, http.StatusConflict, map[string]string{
					"error":   "nickname_taken",
					"message": fmt.Sprintf("'%s' is already taken by %s.", *body.Nickname, colliding),
				})
				return
			}
		}

		if err := UpdateUser(r.Context(), pool, userID, UpdateUserInput{
			FirstName: body.FirstName,
			LastName:  body.LastName,
			Nickname:  body.Nickname,
			Role:      body.Role,
		}); err != nil {
			log.Printf("UpdateUserHandler UpdateUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		updated, err := GetUser(r.Context(), pool, userID)
		if err != nil || updated == nil {
			log.Printf("UpdateUserHandler GetUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, updated)
	}
}

// ResetPasswordHandler handles POST /api/v1/users/{id}/reset-password — admin only.
// Generates a new invite token for password reset and returns the reset path.
func ResetPasswordHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		userID := chi.URLParam(r, "id")

		rawToken, hash, err := auth.GenerateToken()
		if err != nil {
			log.Printf("ResetPasswordHandler GenerateToken error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if err := InsertInviteToken(r.Context(), pool, userID, hash, 7); err != nil {
			log.Printf("ResetPasswordHandler InsertInviteToken error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"reset_path": fmt.Sprintf("/login.html?token=%s", rawToken),
		})
	}
}

// RevokeHandler handles POST /api/v1/users/{id}/revoke — admin only.
// Deletes all sessions for the specified user. Returns 204.
func RevokeHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		userID := chi.URLParam(r, "id")

		if err := auth.DeleteAllSessionsByUserID(r.Context(), pool, userID); err != nil {
			log.Printf("RevokeHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// DeleteUserHandler handles DELETE /api/v1/users/{id} — admin only.
// Deletes the user (sessions cascade via FK). Returns 204.
func DeleteUserHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		userID := chi.URLParam(r, "id")

		if err := DeleteUser(r.Context(), pool, userID); err != nil {
			log.Printf("DeleteUserHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// InviteInfoHandler handles GET /api/v1/auth/invite-info?token= — unauthenticated.
// Returns the first_name for a valid invite token so the accept-invite page can personalize.
func InviteInfoHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawToken := r.URL.Query().Get("token")
		if rawToken == "" {
			writeError(w, http.StatusBadRequest, "token_required")
			return
		}

		tokenHash := auth.HashToken(rawToken)
		firstName, err := GetInviteInfo(r.Context(), pool, tokenHash)
		if err != nil {
			if errors.Is(err, ErrTokenInvalid) {
				writeError(w, http.StatusBadRequest, "token_expired")
				return
			}
			log.Printf("InviteInfoHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"first_name": firstName})
	}
}

// AcceptInviteHandler handles POST /api/v1/auth/accept-invite — unauthenticated.
// Atomically claims the invite token, activates the user, creates a session, and sets cookie.
func AcceptInviteHandler(pool *pgxpool.Pool, secureCookie bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token    string `json:"token"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Token == "" || body.Password == "" {
			writeError(w, http.StatusBadRequest, "validation_error")
			return
		}

		tokenHash := auth.HashToken(body.Token)

		// Atomically claim the token — returns ErrTokenInvalid if not valid
		userID, err := ClaimInviteToken(r.Context(), pool, tokenHash)
		if err != nil {
			if errors.Is(err, ErrTokenInvalid) {
				writeError(w, http.StatusBadRequest, "token_expired")
				return
			}
			log.Printf("AcceptInviteHandler ClaimInviteToken error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		passwordHash, err := auth.HashPassword(body.Password)
		if err != nil {
			log.Printf("AcceptInviteHandler HashPassword error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		if err := ActivateUser(r.Context(), pool, userID, passwordHash); err != nil {
			log.Printf("AcceptInviteHandler ActivateUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		rawToken, err := auth.CreateSession(r.Context(), pool, userID)
		if err != nil {
			log.Printf("AcceptInviteHandler CreateSession error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Set hq_session cookie — same pattern as LoginHandler
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
			// No MaxAge — indefinite session per D-03
		})

		user, err := GetUser(r.Context(), pool, userID)
		if err != nil || user == nil {
			log.Printf("AcceptInviteHandler GetUser error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"user": user})
	}
}

// GetAppPermissionsHandler handles GET /api/v1/apps/permissions — admin only.
func GetAppPermissionsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		perms, err := GetAppPermissions(r.Context(), pool)
		if err != nil {
			log.Printf("GetAppPermissionsHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if perms == nil {
			perms = []AppPermission{}
		}
		writeJSON(w, http.StatusOK, perms)
	}
}

// SetAppPermissionsHandler handles PUT /api/v1/apps/{slug}/permissions — admin only.
// Replaces the full permission set for an app (role grants + user grants).
func SetAppPermissionsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller := auth.UserFromContext(r.Context())
		if caller == nil || !isAdmin(caller) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		slug := chi.URLParam(r, "slug")

		var input SetPermInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json")
			return
		}

		if err := SetAppPermissions(r.Context(), pool, slug, input); err != nil {
			if strings.Contains(err.Error(), "app not found") {
				writeError(w, http.StatusNotFound, "not_found")
				return
			}
			log.Printf("SetAppPermissionsHandler error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// Return updated permissions for this app
		perms, err := GetAppPermissions(r.Context(), pool)
		if err != nil {
			log.Printf("SetAppPermissionsHandler GetAppPermissions error: %v", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		for _, p := range perms {
			if p.Slug == slug {
				writeJSON(w, http.StatusOK, p)
				return
			}
		}
		writeError(w, http.StatusNotFound, "not_found")
	}
}
