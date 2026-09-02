package onboarding

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	// Computed BEFORE the fallback: the fallback is the *unset* case, and the
	// unset case still skips. See internal/testdb for the asymmetry.
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://yumyums:yumyums@localhost:5432/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		testdb.ExitIfRequested(requested, dbURL, "ping", err)
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

// obEnsureUser inserts (or reuses) a user with the given roles and returns its
// id. users is not truncated between tests, so ON CONFLICT keeps it idempotent.
func obEnsureUser(t *testing.T, email string, roles []string) string {
	t.Helper()
	var id string
	err := testPool.QueryRow(t.Context(),
		`INSERT INTO users (email, first_name, last_name, roles, status)
		 VALUES ($1, 'Test', 'User', $2::text[], 'active')
		 ON CONFLICT (email) DO UPDATE SET roles = EXCLUDED.roles
		 RETURNING id::text`,
		email, roles).Scan(&id)
	if err != nil {
		t.Fatalf("insert user %s: %v", email, err)
	}
	return id
}

// obUserContext returns a request context carrying an authenticated user, the
// way auth middleware would.
func obUserContext(id, name string, roles ...string) context.Context {
	return context.WithValue(context.Background(), auth.CtxKeyUser,
		&auth.User{ID: id, DisplayName: name, Roles: roles})
}

// seedVideoPart creates template → section → video_series item → one part with
// the given URL, returning the part id. Cleaned up via template cascade.
func seedVideoPart(t *testing.T, tplName, partTitle, partURL string) (partID string) {
	t.Helper()
	ctx := t.Context()
	var tplID, secID, itemID string
	if err := testPool.QueryRow(ctx,
		`INSERT INTO ob_templates (name) VALUES ($1) RETURNING id::text`, tplName).Scan(&tplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), `DELETE FROM ob_templates WHERE id = $1`, tplID)
	})
	if err := testPool.QueryRow(ctx,
		`INSERT INTO ob_sections (template_id, title) VALUES ($1, 'Sec') RETURNING id::text`, tplID).Scan(&secID); err != nil {
		t.Fatalf("seed section: %v", err)
	}
	if err := testPool.QueryRow(ctx,
		`INSERT INTO ob_items (section_id, type, label) VALUES ($1, 'video_series', 'Grill Operation') RETURNING id::text`, secID).Scan(&itemID); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	if err := testPool.QueryRow(ctx,
		`INSERT INTO ob_video_parts (item_id, title, url) VALUES ($1, $2, $3) RETURNING id::text`, itemID, partTitle, partURL).Scan(&partID); err != nil {
		t.Fatalf("seed part: %v", err)
	}
	return partID
}
