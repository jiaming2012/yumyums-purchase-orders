package workflow

import (
	"context"
	"testing"
)

// ── created_by is nullable; the template reads must tolerate it ─────────────
//
// checklist_templates.created_by is a plain nullable FK (migrations/0006 —
// `UUID REFERENCES users(id)`, no NOT NULL), so a row with no creator is a
// legal state the schema admits. All three template reads scanned it into a
// bare string, which pgx refuses for NULL — so ONE such row errored the whole
// query: GET /api/v1/workflow/templates 500'd and the Builder tab died with
// "Couldn't load templates" (observed live 2026-08-25, dev HQ). myChecklists
// carries the identical scan, so the same row assigned to anyone would take
// down the crew's My Checklists list too — these tests pin all three sites.
//
// The write paths always stamp a creator, so the row is seeded here the way
// the defect actually arrived: straight SQL.

func insertNullCreatorTemplate(t *testing.T, name string) string {
	t.Helper()
	var id string
	if err := testPool.QueryRow(t.Context(),
		`INSERT INTO checklist_templates (name, requires_approval, created_by)
		 VALUES ($1, false, NULL)
		 RETURNING id::text`, name).Scan(&id); err != nil {
		t.Fatalf("insert NULL-creator template: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			`DELETE FROM checklist_templates WHERE id = $1`, id)
	})
	return id
}

func TestListTemplatesToleratesNullCreatedBy(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	id := insertNullCreatorTemplate(t, "Null Creator (list)")

	templates, err := listTemplates(t.Context(), testPool)
	if err != nil {
		t.Fatalf("listTemplates with a NULL created_by row present: %v", err)
	}
	for _, tmpl := range templates {
		if tmpl.ID == id {
			if tmpl.CreatedBy != "" {
				t.Errorf("CreatedBy = %q; want empty string for NULL", tmpl.CreatedBy)
			}
			return
		}
	}
	t.Errorf("template %s missing from listTemplates — the NULL row must be returned, not dropped", id)
}

func TestGetTemplateByIDToleratesNullCreatedBy(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	id := insertNullCreatorTemplate(t, "Null Creator (get)")

	tmpl, err := getTemplateByID(t.Context(), testPool, id)
	if err != nil {
		t.Fatalf("getTemplateByID(NULL created_by): %v", err)
	}
	if tmpl == nil {
		t.Fatalf("getTemplateByID returned nil for an existing template")
	}
	if tmpl.CreatedBy != "" {
		t.Errorf("CreatedBy = %q; want empty string for NULL", tmpl.CreatedBy)
	}
}

func TestMyChecklistsToleratesNullCreatedBy(t *testing.T) {
	if testPool == nil {
		t.Skip("DB unreachable — set DB_TEST_URL")
	}
	ctx := t.Context()
	userID := ensureUser(t, "null-creator@yumyums.kitchen")
	id := insertNullCreatorTemplate(t, "Null Creator (mychecklists)")

	if _, err := testPool.Exec(ctx,
		`INSERT INTO checklist_schedules (template_id, active_days)
		 VALUES ($1, ARRAY[0,1,2,3,4,5,6])`, id); err != nil {
		t.Fatalf("insert schedule: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
		 VALUES ($1, 'role', 'team_member', 'assignee')`, id); err != nil {
		t.Fatalf("insert assignment: %v", err)
	}

	dow := 2
	templates, _, err := myChecklists(ctx, testPool, userID, &dow)
	if err != nil {
		t.Fatalf("myChecklists with a NULL created_by template assigned: %v", err)
	}
	for _, tmpl := range templates {
		if tmpl.ID == id {
			return
		}
	}
	t.Errorf("template %s missing from myChecklists", id)
}
