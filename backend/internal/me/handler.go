package me

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
)

// MeHandler handles GET /api/v1/me.
// Returns the authenticated user's profile from request context.
func MeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":            user.ID,
			"email":         user.Email,
			"display_name":  user.DisplayName,
			"roles":         user.Roles,
			"status":        user.Status,
			"is_superadmin": user.IsSuperadmin,
		})
	}
}

// appRow holds a minimal app record for the /me/apps response.
type appRow struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

// MeAppsHandler handles GET /api/v1/me/apps.
// Superadmins see all enabled apps. Other users see apps they have access to
// via role-based or individual grants.
func MeAppsHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := auth.UserFromContext(r.Context())
		if user == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		var rows []appRow
		var err error

		if user.IsSuperadmin {
			rows, err = queryAllApps(r, pool)
		} else {
			rows, err = queryUserApps(r, pool, user)
		}

		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal_error"}`))
			return
		}

		if rows == nil {
			rows = []appRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rows)
	}
}

func queryAllApps(r *http.Request, pool *pgxpool.Pool) ([]appRow, error) {
	pgRows, err := pool.Query(r.Context(), `
		SELECT slug, name, icon FROM hq_apps
		WHERE enabled = true
		ORDER BY slug
	`)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var apps []appRow
	for pgRows.Next() {
		var a appRow
		if err := pgRows.Scan(&a.Slug, &a.Name, &a.Icon); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, pgRows.Err()
}

func queryUserApps(r *http.Request, pool *pgxpool.Pool, user *auth.User) ([]appRow, error) {
	pgRows, err := pool.Query(r.Context(), `
		SELECT DISTINCT a.slug, a.name, a.icon
		FROM hq_apps a
		LEFT JOIN app_permissions p ON p.app_id = a.id
		WHERE a.enabled = true
		  AND (p.role = ANY($1) OR p.user_id = $2)
		ORDER BY a.slug
	`, user.Roles, user.ID)
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	var apps []appRow
	for pgRows.Next() {
		var a appRow
		if err := pgRows.Scan(&a.Slug, &a.Name, &a.Icon); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, pgRows.Err()
}
