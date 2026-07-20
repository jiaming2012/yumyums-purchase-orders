package sync

import (
	"context"
	"os"
	"reflect"
	"sort"
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

// seedTemplate creates a bare template with one section + one checkbox field and
// NO assignments. Returns (templateID, fieldID) so tests can add assignment rows
// individually (unlike seedTemplateWithField, which bakes in one 'assignee' row).
func seedTemplate(t *testing.T, pool *pgxpool.Pool) (string, string) {
	t.Helper()
	ctx := context.Background()
	var tmplID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_templates (name) VALUES ('Cartesian ' || gen_random_uuid()::text) RETURNING id::text`,
	).Scan(&tmplID); err != nil {
		t.Fatalf("seedTemplate template: %v", err)
	}
	var secID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_sections (template_id, title, "order") VALUES ($1, 'S', 0) RETURNING id::text`,
		tmplID,
	).Scan(&secID); err != nil {
		t.Fatalf("seedTemplate section: %v", err)
	}
	var fieldID string
	if err := pool.QueryRow(ctx,
		`INSERT INTO checklist_fields (section_id, type, label, "order") VALUES ($1, 'checkbox', 'Do C', 0) RETURNING id::text`,
		secID,
	).Scan(&fieldID); err != nil {
		t.Fatalf("seedTemplate field: %v", err)
	}
	return tmplID, fieldID
}

// addAssignment inserts one template_assignments row.
func addAssignment(t *testing.T, pool *pgxpool.Pool, tmplID, assigneeType, assigneeID, assignmentRole string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(),
		`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
		 VALUES ($1, $2, $3, $4)`,
		tmplID, assigneeType, assigneeID, assignmentRole,
	); err != nil {
		t.Fatalf("addAssignment(%s→%s/%s as %s): %v", tmplID, assigneeType, assigneeID, assignmentRole, err)
	}
}

// seedSubmission inserts a checklist_submissions row for the template.
func seedSubmission(t *testing.T, pool *pgxpool.Pool, tmplID, submittedBy string) string {
	t.Helper()
	var subID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO checklist_submissions (template_id, template_snapshot, submitted_by)
		 VALUES ($1, '{}'::jsonb, $2) RETURNING id::text`,
		tmplID, submittedBy,
	).Scan(&subID); err != nil {
		t.Fatalf("seedSubmission: %v", err)
	}
	return subID
}

// assertSameSet fails unless got (order-insensitive) equals want exactly.
func assertSameSet(t *testing.T, got, want []string) {
	t.Helper()
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	sort.Strings(g)
	sort.Strings(w)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("recipient set mismatch:\n got:  %v\n want: %v", g, w)
	}
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

// TestResolveEntityAccess_RoleAssignmentCartesian asserts the resolver's contract
// across the full {role}×{assignment} cartesian:
//
//	recipient set = admins/superadmins ∪ assignees (direct + role-matched)
//
// The op AUTHOR is deliberately NOT part of the resolver's contract — the listener
// unions the author in afterwards (listener.go:63-72), so a non-admin, non-assignee
// author must be ABSENT here (see the ×none rows).
//
// Role axis (from users_roles_check, migration 0023_multi_role.sql):
// admin | manager | team_member. 'superadmin' is NOT representable in the DB —
// config-derived superadmins are upserted with role 'admin' — so the resolver's
// `roles && ARRAY['admin','superadmin']` union is exercised through its 'admin'
// arm; a literal-superadmin row is N/A (constraint violation).
//
// Assignment axis (from the resolver's query, ops.go): directly assigned
// (assignee_type='user'), role-assigned (assignee_type='role' matching the
// user's roles array), and not assigned.
//
// Each combo runs against a freshly-truncated DB (admins receive EVERY
// template's ops, so cross-combo seeding would pollute exact-set assertions).
// Every combo's world also seeds:
//   - anchor:   a directly-assigned team_member  → always IN (assignee arm)
//   - adminCtl: an unassigned admin              → always IN (admins-union arm, the ESC-1 class)
//   - outsider: an unassigned non-admin whose role never matches the combo's
//     role assignment                            → always OUT (negative control)
func TestResolveEntityAccess_RoleAssignmentCartesian(t *testing.T) {
	type combo struct {
		name          string
		subjectRoles  []string
		assignment    string // "direct" | "role" | "none"
		roleAssignee  string // assignee_id when assignment == "role"
		outsiderRoles []string
		expectSubject bool
	}
	combos := []combo{
		{"admin×direct", []string{"admin"}, "direct", "", []string{"manager"}, true},
		{"admin×role-assigned", []string{"admin"}, "role", "admin", []string{"manager"}, true},
		// admin×none is the ESC-1 class: unassigned admin still receives the op.
		{"admin×not-assigned", []string{"admin"}, "none", "", []string{"manager"}, true},
		{"manager×direct", []string{"manager"}, "direct", "", []string{"team_member"}, true},
		// outsider must be team_member here or the 'manager' role assignment would catch it.
		{"manager×role-assigned", []string{"manager"}, "role", "manager", []string{"team_member"}, true},
		{"manager×not-assigned", []string{"manager"}, "none", "", []string{"team_member"}, false},
		{"team_member×direct", []string{"team_member"}, "direct", "", []string{"manager"}, true},
		{"team_member×role-assigned", []string{"team_member"}, "role", "team_member", []string{"manager"}, true},
		// team_member×none doubles as the author-exclusion proof: if this subject
		// authored the op, the RESOLVER still excludes them — the listener adds the
		// author (listener.go:63-72).
		{"team_member×not-assigned", []string{"team_member"}, "none", "", []string{"manager"}, false},
		// Multi-role membership: ANY(u.roles) must match a role assignment against
		// any held role, not just the first.
		{"manager+team_member×role-assigned(team_member)", []string{"manager", "team_member"}, "role", "team_member", []string{"manager"}, true},
	}

	for _, c := range combos {
		t.Run(c.name, func(t *testing.T) {
			pool := setupAccessTestDB(t)
			ctx := context.Background()

			subject := seedUser(t, pool, "subject@cartesian.test", c.subjectRoles)
			anchor := seedUser(t, pool, "anchor@cartesian.test", []string{"team_member"})
			adminCtl := seedUser(t, pool, "adminctl@cartesian.test", []string{"admin"})
			outsider := seedUser(t, pool, "outsider@cartesian.test", c.outsiderRoles)

			tmplID, fieldID := seedTemplate(t, pool)
			addAssignment(t, pool, tmplID, "user", anchor, "assignee")
			switch c.assignment {
			case "direct":
				addAssignment(t, pool, tmplID, "user", subject, "assignee")
			case "role":
				addAssignment(t, pool, tmplID, "role", c.roleAssignee, "assignee")
			case "none":
				// template stays assigned only to the anchor
			}

			got, err := ResolveEntityAccess(ctx, pool, fieldID, "field_response")
			if err != nil {
				t.Fatalf("ResolveEntityAccess: %v", err)
			}

			want := []string{anchor, adminCtl}
			if c.expectSubject {
				want = append(want, subject)
			}
			assertSameSet(t, got, want)
			if contains(got, outsider) {
				t.Errorf("outsider %s (roles %v, unassigned, non-admin) must NOT be a recipient, got %v", outsider, c.outsiderRoles, got)
			}
			if !c.expectSubject && contains(got, subject) {
				t.Errorf("subject %s (roles %v, not assigned, non-admin) must NOT be a recipient — the listener, not the resolver, adds the author (listener.go:63-72)", subject, c.subjectRoles)
			}
		})
	}
}

// TestResolveEntityAccess_SubmissionEntity covers the entity_type='submission'
// branch (submission → template_id → shared recipient query). The role×assignment
// cartesian is NOT re-run per entity type: all three branches converge on the same
// recipient query (ops.go), differing only in template resolution — one
// representative fan-out per branch suffices.
func TestResolveEntityAccess_SubmissionEntity(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	tm := seedUser(t, pool, "tm@sub.test", []string{"team_member"})
	admin := seedUser(t, pool, "admin@sub.test", []string{"admin"})
	outsider := seedUser(t, pool, "outsider@sub.test", []string{"manager"})

	tmplID, _ := seedTemplate(t, pool)
	addAssignment(t, pool, tmplID, "user", tm, "assignee")
	subID := seedSubmission(t, pool, tmplID, tm)

	got, err := ResolveEntityAccess(ctx, pool, subID, "submission")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	assertSameSet(t, got, []string{tm, admin})
	if contains(got, outsider) {
		t.Errorf("unassigned manager %s must NOT receive submission ops, got %v", outsider, got)
	}
}

// TestResolveEntityAccess_TemplateEntity covers the entity_type='template'
// branch (entity id IS the template id — no lookup hop).
func TestResolveEntityAccess_TemplateEntity(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	mgr := seedUser(t, pool, "mgr@tmpl.test", []string{"manager"})
	admin := seedUser(t, pool, "admin@tmpl.test", []string{"admin"})
	outsider := seedUser(t, pool, "outsider@tmpl.test", []string{"team_member"})

	tmplID, _ := seedTemplate(t, pool)
	addAssignment(t, pool, tmplID, "role", "manager", "assignee")

	got, err := ResolveEntityAccess(ctx, pool, tmplID, "template")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	assertSameSet(t, got, []string{mgr, admin})
	if contains(got, outsider) {
		t.Errorf("unassigned team_member %s must NOT receive template ops, got %v", outsider, got)
	}
}

// TestResolveEntityAccess_UnknownEntityType asserts the default branch: an
// unrecognized entity type resolves to the empty set (listener then falls back
// to author-only) with no error.
func TestResolveEntityAccess_UnknownEntityType(t *testing.T) {
	pool := setupAccessTestDB(t)
	seedUser(t, pool, "admin@unknown.test", []string{"admin"})

	got, err := ResolveEntityAccess(context.Background(), pool, "00000000-0000-0000-0000-000000000000", "bogus_entity")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("unknown entity type must resolve to empty set (even with admins present), got %v", got)
	}
}

// TestResolveEntityAccess_MissingEntityRows asserts the ErrNoRows paths: a
// field_response or submission id with no backing row resolves to the empty set,
// not an error.
func TestResolveEntityAccess_MissingEntityRows(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()
	seedUser(t, pool, "admin@missing.test", []string{"admin"})

	const ghost = "00000000-0000-0000-0000-000000000001"
	for _, entityType := range []string{"field_response", "submission"} {
		got, err := ResolveEntityAccess(ctx, pool, ghost, entityType)
		if err != nil {
			t.Fatalf("ResolveEntityAccess(%s, missing row): %v", entityType, err)
		}
		if len(got) != 0 {
			t.Errorf("missing %s row must resolve to empty set, got %v", entityType, got)
		}
	}
}

// TestResolveEntityAccess_ApproverIncluded_CurrentBehavior pins the CONTRACT:
// the recipient query (ops.go) matches template_assignments rows WITHOUT
// filtering assignment_role, so an 'approver'-linked user receives live ops
// exactly like an 'assignee'.
//
// This was previously a reviewer-NOTE flagging approver inclusion as an open
// contract question ("the stated contract says assignees"). That question is
// settled: signed decision B4 — "everyone with entity access sees live ops."
// Approver fan-out is therefore INTENDED behavior, not merely observed
// behavior, and this test is a contract pin. Narrowing the fan-out to
// assignee-only would be a deliberate contract change requiring a new
// decision, not a bug fix.
func TestResolveEntityAccess_ApproverIncluded_CurrentBehavior(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	approver := seedUser(t, pool, "approver@appr.test", []string{"manager"})
	tm := seedUser(t, pool, "tm@appr.test", []string{"team_member"})

	tmplID, fieldID := seedTemplate(t, pool)
	addAssignment(t, pool, tmplID, "user", tm, "assignee")
	addAssignment(t, pool, tmplID, "user", approver, "approver")

	got, err := ResolveEntityAccess(ctx, pool, fieldID, "field_response")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	assertSameSet(t, got, []string{tm, approver})
}

// TestResolveEntityAccess_NoDuplicates asserts a user matching BOTH arms of the
// contract (an admin who is also directly assigned) appears exactly once.
func TestResolveEntityAccess_NoDuplicates(t *testing.T) {
	pool := setupAccessTestDB(t)
	ctx := context.Background()

	admin := seedUser(t, pool, "admin@dedup.test", []string{"admin"})
	tmplID, fieldID := seedTemplate(t, pool)
	addAssignment(t, pool, tmplID, "user", admin, "assignee")
	// A second matching assignment row too — still one recipient entry.
	addAssignment(t, pool, tmplID, "role", "admin", "assignee")

	got, err := ResolveEntityAccess(ctx, pool, fieldID, "field_response")
	if err != nil {
		t.Fatalf("ResolveEntityAccess: %v", err)
	}
	n := 0
	for _, uid := range got {
		if uid == admin {
			n++
		}
	}
	if n != 1 {
		t.Errorf("admin+assignee must appear exactly once, appeared %d times in %v", n, got)
	}
}
