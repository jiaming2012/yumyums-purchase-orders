package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultTimezone is the IANA timezone applied when a user has no explicit timezone set.
const DefaultTimezone = "America/New_York"

// displayNameExpr computes display_name from first_name/last_name/nickname.
// Matches the same expression used in auth/service.go.
const displayNameExpr = `COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS display_name`

// ErrTokenInvalid is returned when an invite token is invalid, expired, or already used.
var ErrTokenInvalid = errors.New("token_invalid")

// UserRow is the full user record returned from DB queries.
type UserRow struct {
	ID                   string     `json:"id"`
	Email                string     `json:"email"`
	FirstName            string     `json:"first_name"`
	LastName             string     `json:"last_name"`
	Nickname             *string    `json:"nickname,omitempty"`
	DisplayName          string     `json:"display_name"`
	Roles                []string   `json:"roles"`
	Status               string     `json:"status"`
	NotificationChannels []string   `json:"notification_channels"`
	Timezone             string     `json:"timezone"`
	EmployeeNumber       *int       `json:"employee_number,omitempty"`
	EmployeeType         *string    `json:"employee_type,omitempty"`
	StartingSalary       *float64   `json:"starting_salary,omitempty"`
	ToastPosNumber       *string    `json:"toast_pos_number,omitempty"`
	CashAppID            *string    `json:"cash_app_id,omitempty"`
	PhoneNumber          *string    `json:"phone_number,omitempty"`
	InvitedAt            time.Time  `json:"invited_at"`
	AcceptedAt           *time.Time `json:"accepted_at,omitempty"`
}

// NotificationPreferenceContact holds the minimal fields needed for alert delivery.
type NotificationPreferenceContact struct {
	UserID               string
	Email                string
	DisplayName          string
	NotificationChannels []string
	Timezone             string
}

// CreateUserInput holds fields required to create an invited user.
type CreateUserInput struct {
	FirstName      string
	LastName       string
	Email          string
	Roles          []string
	EmployeeType   *string
	StartingSalary *float64
}

// UpdateUserInput holds optional fields for partial user updates.
type UpdateUserInput struct {
	FirstName        *string
	LastName         *string
	Nickname         *string
	Roles            *[]string
	NotificationPref *[]string // array of "zoho_cliq" and/or "email"
	Timezone         *string   // IANA timezone name, e.g. "America/New_York"
}

// AppPermission represents a single app's permission state.
type AppPermission struct {
	Slug       string   `json:"slug"`
	Name       string   `json:"name"`
	Icon       string   `json:"icon"`
	RoleGrants []string `json:"role_grants"`
	UserGrants []string `json:"user_grants"`
}

// SetPermInput holds the desired role and user grants for an app.
type SetPermInput struct {
	RoleGrants []string `json:"role_grants"`
	UserGrants []string `json:"user_grants"`
}

// ListUsers returns all users ordered alphabetically by display name.
func ListUsers(ctx context.Context, pool *pgxpool.Pool) ([]UserRow, error) {
	query := fmt.Sprintf(`
		SELECT u.id, u.email, u.first_name, u.last_name, u.nickname,
		       %s, u.roles, u.status, u.notification_channel, u.timezone,
		       u.employee_number, u.employee_type, u.starting_salary,
		       u.toast_pos_number, u.cash_app_id, u.phone_number,
		       u.invited_at, u.accepted_at
		FROM users u
		ORDER BY display_name ASC
	`, displayNameExpr)

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []UserRow
	for rows.Next() {
		var u UserRow
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Nickname,
			&u.DisplayName, &u.Roles, &u.Status, &u.NotificationChannels, &u.Timezone,
			&u.EmployeeNumber, &u.EmployeeType, &u.StartingSalary,
			&u.ToastPosNumber, &u.CashAppID, &u.PhoneNumber,
			&u.InvitedAt, &u.AcceptedAt,
		); err != nil {
			return nil, fmt.Errorf("list users scan: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users rows: %w", err)
	}
	return users, nil
}

// GetUser returns a single user by ID.
func GetUser(ctx context.Context, pool *pgxpool.Pool, userID string) (*UserRow, error) {
	query := fmt.Sprintf(`
		SELECT u.id, u.email, u.first_name, u.last_name, u.nickname,
		       %s, u.roles, u.status, u.notification_channel, u.timezone,
		       u.employee_number, u.employee_type, u.starting_salary,
		       u.toast_pos_number, u.cash_app_id, u.phone_number,
		       u.invited_at, u.accepted_at
		FROM users u
		WHERE u.id = $1
	`, displayNameExpr)

	var u UserRow
	err := pool.QueryRow(ctx, query, userID).Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Nickname,
		&u.DisplayName, &u.Roles, &u.Status, &u.NotificationChannels, &u.Timezone,
		&u.EmployeeNumber, &u.EmployeeType, &u.StartingSalary,
		&u.ToastPosNumber, &u.CashAppID, &u.PhoneNumber,
		&u.InvitedAt, &u.AcceptedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// GetNotificationPreference returns a user's notification_channels as a []string.
func GetNotificationPreference(ctx context.Context, pool *pgxpool.Pool, userID string) ([]string, error) {
	var channels []string
	err := pool.QueryRow(ctx, `SELECT notification_channel FROM users WHERE id = $1`, userID).Scan(&channels)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get notification preference: %w", err)
	}
	return channels, nil
}

// UpdateNotificationPreference sets notification_channel (TEXT[]) for a user.
// Valid values are 'zoho_cliq' and 'email'. At least one channel is required.
func UpdateNotificationPreference(ctx context.Context, pool *pgxpool.Pool, userID string, channels []string) error {
	if len(channels) == 0 {
		return fmt.Errorf("at least one notification channel is required")
	}
	for _, ch := range channels {
		if ch != "zoho_cliq" && ch != "email" {
			return fmt.Errorf("invalid notification_channel %q: must be 'zoho_cliq' or 'email'", ch)
		}
	}
	tag, err := pool.Exec(ctx, `UPDATE users SET notification_channel = $2 WHERE id = $1`, userID, channels)
	if err != nil {
		return fmt.Errorf("update notification preference: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}
	return nil
}

// GetUsersForAlerts returns contact info for all users who should receive a given type of alert.
// For cutoff reminders (D-09): users with 'order' app permission.
// For shopping completion (D-12): admin users only (callers add the shopper separately).
func GetUsersForAlerts(ctx context.Context, pool *pgxpool.Pool, alertType string) ([]NotificationPreferenceContact, error) {
	var query string
	switch alertType {
	case "cutoff_reminder":
		// Users with 'order' permission: role grants to 'crew', 'manager', 'admin' + individual user grants
		query = `
			SELECT DISTINCT u.id, u.email,
			       COALESCE(NULLIF(u.nickname,''), u.first_name || ' ' || LEFT(u.last_name,1) || '.') AS display_name,
			       u.notification_channel, u.timezone
			FROM users u
			WHERE u.status = 'active'
			  AND (
			    'crew' = ANY(u.roles) OR 'manager' = ANY(u.roles) OR 'admin' = ANY(u.roles)
			    OR EXISTS (
			      SELECT 1 FROM app_permissions ap
			      JOIN hq_apps a ON a.id = ap.app_id
			      WHERE a.slug = 'purchasing' AND ap.user_id = u.id
			    )
			  )
			ORDER BY display_name
		`
	default: // shopping_completion and others — admins only
		query = `
			SELECT u.id, u.email,
			       COALESCE(NULLIF(u.nickname,''), u.first_name || ' ' || LEFT(u.last_name,1) || '.') AS display_name,
			       u.notification_channel, u.timezone
			FROM users u
			WHERE u.status = 'active'
			  AND ('admin' = ANY(u.roles))
			ORDER BY display_name
		`
	}

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetUsersForAlerts: %w", err)
	}
	defer rows.Close()

	var contacts []NotificationPreferenceContact
	for rows.Next() {
		var c NotificationPreferenceContact
		if err := rows.Scan(&c.UserID, &c.Email, &c.DisplayName, &c.NotificationChannels, &c.Timezone); err != nil {
			return nil, fmt.Errorf("GetUsersForAlerts scan: %w", err)
		}
		contacts = append(contacts, c)
	}
	return contacts, rows.Err()
}

// CreateInvitedUser inserts a new user with status='invited' and returns the new user ID.
// Returns an error if first_name or last_name are empty.
func CreateInvitedUser(ctx context.Context, pool *pgxpool.Pool, input CreateUserInput) (string, error) {
	if input.FirstName == "" {
		return "", fmt.Errorf("first_name is required")
	}
	if input.LastName == "" {
		return "", fmt.Errorf("last_name is required")
	}

	var id string
	err := pool.QueryRow(ctx, `
		INSERT INTO users (email, first_name, last_name, roles, status, employee_type, starting_salary)
		VALUES ($1, $2, $3, $4, 'invited', $5, $6)
		RETURNING id
	`, input.Email, input.FirstName, input.LastName, input.Roles, input.EmployeeType, input.StartingSalary).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create invited user: %w", err)
	}
	return id, nil
}

// UpdateUser applies partial updates to a user's fields.
func UpdateUser(ctx context.Context, pool *pgxpool.Pool, userID string, input UpdateUserInput) error {
	// Build SET clause dynamically based on non-nil fields
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if input.FirstName != nil {
		setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIdx))
		args = append(args, *input.FirstName)
		argIdx++
	}
	if input.LastName != nil {
		setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIdx))
		args = append(args, *input.LastName)
		argIdx++
	}
	if input.Nickname != nil {
		setClauses = append(setClauses, fmt.Sprintf("nickname = $%d", argIdx))
		args = append(args, *input.Nickname)
		argIdx++
	}
	if input.Roles != nil {
		setClauses = append(setClauses, fmt.Sprintf("roles = $%d", argIdx))
		args = append(args, *input.Roles)
		argIdx++
	}
	if input.NotificationPref != nil {
		channels := *input.NotificationPref
		if len(channels) == 0 {
			return fmt.Errorf("invalid notification_channel: at least one channel required")
		}
		for _, ch := range channels {
			if ch != "zoho_cliq" && ch != "email" {
				return fmt.Errorf("invalid notification_channel %q: must be 'zoho_cliq' or 'email'", ch)
			}
		}
		setClauses = append(setClauses, fmt.Sprintf("notification_channel = $%d", argIdx))
		args = append(args, channels)
		argIdx++
	}
	if input.Timezone != nil {
		tz := *input.Timezone
		if _, err := time.LoadLocation(tz); err != nil {
			return fmt.Errorf("invalid timezone %q: must be a valid IANA timezone name", tz)
		}
		setClauses = append(setClauses, fmt.Sprintf("timezone = $%d", argIdx))
		args = append(args, tz)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil // nothing to update
	}

	args = append(args, userID)
	query := "UPDATE users SET "
	for i, clause := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += clause
	}
	query += fmt.Sprintf(" WHERE id = $%d", argIdx)

	_, err := pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// CheckNicknameCollision checks if nickname collides with another user's nickname or derived display_name.
// Returns the display_name of the colliding user, or empty string if no collision.
func CheckNicknameCollision(ctx context.Context, pool *pgxpool.Pool, nickname string, excludeUserID string) (string, error) {
	query := `
		SELECT COALESCE(NULLIF(u.nickname,''), u.first_name || ' ' || LEFT(u.last_name,1) || '.') AS display_name
		FROM users u
		WHERE u.id != $2
		  AND (LOWER(u.nickname) = LOWER($1) OR LOWER(COALESCE(NULLIF(u.nickname,''), u.first_name || ' ' || LEFT(u.last_name,1) || '.')) = LOWER($1))
		LIMIT 1
	`

	var collidingDisplayName string
	err := pool.QueryRow(ctx, query, nickname, excludeUserID).Scan(&collidingDisplayName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("check nickname collision: %w", err)
	}
	return collidingDisplayName, nil
}

// InsertInviteToken inserts a new invite token into the invite_tokens table.
func InsertInviteToken(ctx context.Context, pool *pgxpool.Pool, userID, tokenHash string, expiresDays int) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO invite_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, now() + $3 * interval '1 day')
	`, userID, tokenHash, expiresDays)
	if err != nil {
		return fmt.Errorf("insert invite token: %w", err)
	}
	return nil
}

// ClaimInviteToken atomically marks a token as used and returns the associated user_id.
// Returns ErrTokenInvalid if the token is not found, already used, or expired.
func ClaimInviteToken(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (string, error) {
	var userID string
	err := pool.QueryRow(ctx, `
		UPDATE invite_tokens
		SET used_at = now()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > now()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrTokenInvalid
		}
		return "", fmt.Errorf("claim invite token: %w", err)
	}
	return userID, nil
}

// ActivateUser sets a user's status to 'active' and stores their password hash.
// If nickname is non-empty, it is also set.
// ActivateUserInput holds the fields set during invite acceptance.
type ActivateUserInput struct {
	PasswordHash   string
	Nickname       string
	ToastPosNumber *string
	CashAppID      *string
	PhoneNumber    *string
}

func ActivateUser(ctx context.Context, pool *pgxpool.Pool, userID string, input ActivateUserInput) error {
	_, err := pool.Exec(ctx, `
		UPDATE users
		SET status = 'active',
		    password_hash = $2,
		    accepted_at = now(),
		    nickname = CASE WHEN $3 = '' THEN nickname ELSE $3 END,
		    toast_pos_number = COALESCE($4, toast_pos_number),
		    cash_app_id = COALESCE($5, cash_app_id),
		    phone_number = COALESCE($6, phone_number)
		WHERE id = $1
	`, userID, input.PasswordHash, input.Nickname, input.ToastPosNumber, input.CashAppID, input.PhoneNumber)
	if err != nil {
		return fmt.Errorf("activate user: %w", err)
	}
	return nil
}

// DeleteUser removes a user and all their associated sessions (cascaded by FK).
func DeleteUser(ctx context.Context, pool *pgxpool.Pool, userID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// GetAppPermissions returns all apps with their current role and user grants.
func GetAppPermissions(ctx context.Context, pool *pgxpool.Pool) ([]AppPermission, error) {
	rows, err := pool.Query(ctx, `
		SELECT a.slug, a.name, a.icon,
		       COALESCE(array_agg(ap.role ORDER BY ap.role) FILTER (WHERE ap.role IS NOT NULL), '{}') AS role_grants,
		       COALESCE(array_agg(ap.user_id::text ORDER BY ap.user_id::text) FILTER (WHERE ap.user_id IS NOT NULL), '{}') AS user_grants
		FROM hq_apps a
		LEFT JOIN app_permissions ap ON ap.app_id = a.id
		WHERE a.enabled = true
		GROUP BY a.slug, a.name, a.icon
		ORDER BY a.name
	`)
	if err != nil {
		return nil, fmt.Errorf("get app permissions: %w", err)
	}
	defer rows.Close()

	var perms []AppPermission
	for rows.Next() {
		var p AppPermission
		if err := rows.Scan(&p.Slug, &p.Name, &p.Icon, &p.RoleGrants, &p.UserGrants); err != nil {
			return nil, fmt.Errorf("get app permissions scan: %w", err)
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get app permissions rows: %w", err)
	}
	return perms, nil
}

// SetAppPermissions replaces all permissions for a given app slug.
// Uses a transaction to DELETE existing + INSERT new grants atomically.
func SetAppPermissions(ctx context.Context, pool *pgxpool.Pool, slug string, input SetPermInput) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("set app permissions begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Get app_id for the slug
	var appID string
	if err := tx.QueryRow(ctx, `SELECT id FROM hq_apps WHERE slug = $1 AND enabled = true`, slug).Scan(&appID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("app not found: %s", slug)
		}
		return fmt.Errorf("set app permissions lookup app: %w", err)
	}

	// Delete all existing permissions for this app
	if _, err := tx.Exec(ctx, `DELETE FROM app_permissions WHERE app_id = $1`, appID); err != nil {
		return fmt.Errorf("set app permissions delete: %w", err)
	}

	// Insert role grants
	for _, role := range input.RoleGrants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_permissions (app_id, role) VALUES ($1, $2)
		`, appID, role); err != nil {
			return fmt.Errorf("set app permissions insert role %s: %w", role, err)
		}
	}

	// Insert user grants
	for _, userID := range input.UserGrants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO app_permissions (app_id, user_id) VALUES ($1, $2)
		`, appID, userID); err != nil {
			return fmt.Errorf("set app permissions insert user %s: %w", userID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("set app permissions commit: %w", err)
	}
	return nil
}

// GetInviteInfo returns the first_name of the user associated with a valid invite token.
// Returns ErrTokenInvalid if token is expired or already used.
func GetInviteInfo(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (string, string, error) {
	var firstName, status string
	err := pool.QueryRow(ctx, `
		SELECT u.first_name, u.status
		FROM invite_tokens it
		JOIN users u ON it.user_id = u.id
		WHERE it.token_hash = $1
		  AND it.used_at IS NULL
		  AND it.expires_at > now()
	`, tokenHash).Scan(&firstName, &status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrTokenInvalid
		}
		return "", "", fmt.Errorf("get invite info: %w", err)
	}
	return firstName, status, nil
}
