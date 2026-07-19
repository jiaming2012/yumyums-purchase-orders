package sync

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupAccessTestDB connects to the test DB and truncates the tables this file
// seeds so each test starts clean. Mirrors the connect-or-skip idiom used across
// the backend test suite (a missing/unreachable DB skips rather than fails).
// Required env: DB_TEST_URL (set by the Taskfile) or TEST_DATABASE_URL.
func setupAccessTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DB_TEST_URL")
	if dbURL == "" {
		dbURL = os.Getenv("TEST_DATABASE_URL")
	}
	if dbURL == "" {
		t.Skip("DB_TEST_URL / TEST_DATABASE_URL not set — skipping integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("DB_TEST_URL not reachable (connect failed): %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("DB_TEST_URL not reachable (ping failed): %v", err)
	}
	_, err = pool.Exec(ctx,
		`TRUNCATE ops, submission_responses, template_assignments,
		          checklist_fields, checklist_sections, checklist_templates, users
		 RESTART IDENTITY CASCADE`)
	if err != nil {
		pool.Close()
		t.Fatalf("setupAccessTestDB truncate: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, email string, roles []string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO users (email, roles) VALUES ($1, $2) RETURNING id::text`,
		email, roles,
	).Scan(&id)
	if err != nil {
		t.Fatalf("seedUser(%q): %v", email, err)
	}
	return id
}

// seedTemplateWithField creates a template assigned to (assigneeType, assigneeID)
// as 'assignee', with one section holding one checkbox field. Returns the field id.
func seedTemplateWithField(t *testing.T, pool *pgxpool.Pool, assigneeType, assigneeID string) string {
	t.Helper()
	ctx := context.Background()
	var tmplID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_templates (name) VALUES ('Access Test') RETURNING id::text`,
	).Scan(&tmplID); err != nil {
		t.Fatalf("seed template: %v", err)
	}
	var secID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_sections (template_id, title, "order") VALUES ($1, 'S', 0) RETURNING id::text`,
		tmplID,
	).Scan(&secID); err != nil {
		t.Fatalf("seed section: %v", err)
	}
	var fieldID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_fields (section_id, type, label, "order") VALUES ($1, 'checkbox', 'Do C', 0) RETURNING id::text`,
		secID,
	).Scan(&fieldID); err != nil {
		t.Fatalf("seed field: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
		 VALUES ($1, $2, $3, 'assignee')`,
		tmplID, assigneeType, assigneeID,
	); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	return fieldID
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

// TestResolveEntityAccess_AdminReceivesLiveOps reproduces the operator-found live-sync
// bug: a field_response op on a checklist assigned to `team_member` must fan out to
// admins/superadmins too (they can VIEW every checklist — myChecklists grants
// `roles && {admin,superadmin}` view-all — so live sync must MIRROR that access, or an
// admin editing a checklist they aren't assigned to never sees it on their other device).
// RED before the fix (admin absent from the recipient set); GREEN after.
func TestResolveEntityAccess_AdminReceivesLiveOps(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	adminID := seedUser(t, pool, "admin@access.test", []string{"admin"})
	tmID := seedUser(t, pool, "tm@access.test", []string{"team_member"})
	fieldID := seedTemplateWithField(t, pool, "role", "team_member")

	userIDs, err := ResolveEntityAccess(ctx, pool, fieldID, "field_response")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}

	// The assignee (team_member) must always be a recipient (baseline, already true).
	if !contains(userIDs, tmID) {
		t.Errorf("expected team_member assignee %s in recipients, got %v", tmID, userIDs)
	}
	// The bug: an admin (who can view every checklist) must also receive the live op,
	// so an admin editing a non-assigned checklist converges on their other devices.
	// (Config-derived superadmins carry DB role 'admin' — users_roles_check permits only
	// admin|manager|team_member — so this case also covers the operator's superadmin login.)
	if !contains(userIDs, adminID) {
		t.Errorf("admin %s must receive live ops for any checklist (mirrors myChecklists view-all), got %v", adminID, userIDs)
	}
}

// TestResolveEntityAccess_AuthorFallbackWhenNoAssignees keeps the existing guarantee:
// with no assignees resolvable, the resolver returns empty and the listener falls back
// to the op author (this test asserts the empty-set precondition the fallback relies on).
func TestResolveEntityAccess_EmptyWhenNoMatchingUsers(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()
	// Template assigned to a role no seeded user holds → no assignees, no admins.
	fieldID := seedTemplateWithField(t, pool, "role", "manager")
	userIDs, err := ResolveEntityAccess(ctx, pool, fieldID, "field_response")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	if len(userIDs) != 0 {
		t.Errorf("expected empty recipient set (listener adds author), got %v", userIDs)
	}
}
