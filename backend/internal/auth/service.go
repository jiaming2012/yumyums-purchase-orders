package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// User represents an authenticated user (returned from session lookup or login)
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	IsSuperadmin bool   `json:"is_superadmin,omitempty"`
}

// GenerateToken creates a cryptographically secure random token.
// Returns the raw token (for cookie) and its SHA-256 hash (for storage).
func GenerateToken() (raw string, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	raw = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return raw, hash, nil
}

// HashToken computes the SHA-256 hash of a raw token string.
// Used by middleware to look up sessions without storing the raw token.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// VerifyPassword checks a bcrypt hash against a plaintext password.
func VerifyPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CreateSession generates a new session token, stores its hash in the DB,
// and returns the raw token to be set in the session cookie.
func CreateSession(ctx context.Context, pool *pgxpool.Pool, userID string) (rawToken string, err error) {
	raw, hash, err := GenerateToken()
	if err != nil {
		return "", err
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id) VALUES ($1, $2)`,
		hash, userID,
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return raw, nil
}

// DeleteSessionByHash deletes a session by its token hash.
func DeleteSessionByHash(ctx context.Context, pool *pgxpool.Pool, tokenHash string) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM sessions WHERE token_hash = $1`,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// LookupSession finds the user associated with a session token hash.
// Returns nil, nil if the session does not exist or has expired.
func LookupSession(ctx context.Context, pool *pgxpool.Pool, tokenHash string, superadmins map[string]config.SuperadminEntry) (*User, error) {
	row := pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.display_name, u.role, u.status
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.token_hash = $1
		  AND (s.expires_at IS NULL OR s.expires_at > now())
	`, tokenHash)

	var u User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.Status)
	if err != nil {
		// pgx returns pgx.ErrNoRows when no row found
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup session: %w", err)
	}

	if _, ok := superadmins[u.Email]; ok {
		u.IsSuperadmin = true
		u.Role = "superadmin"
	}
	return &u, nil
}

// AuthenticateUser verifies email + password credentials and returns the user if valid.
// Returns nil, nil if credentials are invalid (no such user, no password set, wrong password).
func AuthenticateUser(ctx context.Context, pool *pgxpool.Pool, email, password string, superadmins map[string]config.SuperadminEntry) (*User, error) {
	row := pool.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, role, status
		FROM users
		WHERE email = $1
	`, email)

	var (
		u            User
		passwordHash *string
	)
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &passwordHash, &u.Role, &u.Status)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("authenticate user: %w", err)
	}

	// Invited but not yet accepted (no password set)
	if passwordHash == nil {
		return nil, nil
	}

	if err := VerifyPassword(*passwordHash, password); err != nil {
		return nil, nil
	}

	if _, ok := superadmins[u.Email]; ok {
		u.IsSuperadmin = true
		u.Role = "superadmin"
	}
	return &u, nil
}

// UpsertSuperadmins ensures each superadmin from config exists in the users table.
// If the user already exists, only the display_name is updated. This allows
// superadmins to set their password via the invite flow.
func UpsertSuperadmins(ctx context.Context, pool *pgxpool.Pool, superadmins map[string]config.SuperadminEntry) error {
	for email, entry := range superadmins {
		_, err := pool.Exec(ctx, `
			INSERT INTO users (email, display_name, role, status)
			VALUES ($1, $2, 'admin', 'invited')
			ON CONFLICT (email) DO UPDATE SET display_name = EXCLUDED.display_name
		`, email, entry.DisplayName)
		if err != nil {
			return fmt.Errorf("upsert superadmin %s: %w", email, err)
		}

		// If dev_password is set, hash and store it so login works without running seed
		if entry.DevPassword != "" {
			hash, err := HashPassword(entry.DevPassword)
			if err != nil {
				return fmt.Errorf("hash dev_password for %s: %w", email, err)
			}
			_, err = pool.Exec(ctx, `
				UPDATE users SET password_hash = $1, status = 'active'
				WHERE email = $2 AND (password_hash IS NULL OR password_hash = '')
			`, hash, email)
			if err != nil {
				return fmt.Errorf("set dev_password for %s: %w", email, err)
			}
		}

		log.Printf("Upserted superadmin: %s", email)
	}
	return nil
}
