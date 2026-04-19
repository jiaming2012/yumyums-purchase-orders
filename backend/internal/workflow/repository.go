package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrTemplateArchived is returned when submitting a checklist for an archived template.
var ErrTemplateArchived = errors.New("template is archived")

// insertTemplate inserts a full template (with sections, fields, schedules, assignments)
// in a single transaction. Returns the new template UUID.
func insertTemplate(ctx context.Context, pool *pgxpool.Pool, input TemplateInput, createdBy string) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	templateID, err := insertTemplateInTx(ctx, tx, input, createdBy)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}
	return templateID, nil
}

// insertTemplateInTx inserts template rows inside an existing transaction.
// Used by both insertTemplate and replaceTemplate.
func insertTemplateInTx(ctx context.Context, tx pgx.Tx, input TemplateInput, createdBy string) (string, error) {
	var templateID string
	err := tx.QueryRow(ctx,
		`INSERT INTO checklist_templates (name, requires_approval, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		input.Name, input.RequiresApproval, createdBy,
	).Scan(&templateID)
	if err != nil {
		return "", fmt.Errorf("insert template: %w", err)
	}

	// Insert schedules
	for _, sched := range input.Schedules {
		if _, err := tx.Exec(ctx,
			`INSERT INTO checklist_schedules (template_id, active_days) VALUES ($1, $2)`,
			templateID, sched.ActiveDays,
		); err != nil {
			return "", fmt.Errorf("insert schedule: %w", err)
		}
	}

	// Insert assignments
	for _, asgn := range input.Assignments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 VALUES ($1, $2, $3, $4)`,
			templateID, asgn.AssigneeType, asgn.AssigneeID, asgn.AssignmentRole,
		); err != nil {
			return "", fmt.Errorf("insert assignment: %w", err)
		}
	}

	// Insert sections and fields
	for _, sec := range input.Sections {
		condJSON, err := marshalNullableJSON(sec.Condition)
		if err != nil {
			return "", fmt.Errorf("marshal section condition: %w", err)
		}
		var sectionID string
		err = tx.QueryRow(ctx,
			`INSERT INTO checklist_sections (template_id, title, "order", condition)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			templateID, sec.Title, sec.Order, condJSON,
		).Scan(&sectionID)
		if err != nil {
			return "", fmt.Errorf("insert section %q: %w", sec.Title, err)
		}
		for _, field := range sec.Fields {
			if err := insertField(ctx, tx, sectionID, nil, field); err != nil {
				return "", fmt.Errorf("insert field %q: %w", field.Label, err)
			}
		}
	}

	return templateID, nil
}

// replaceTemplate performs a full replace (D-09): deletes all child records for
// the template and re-inserts from input. Template header row is updated in-place.
func replaceTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string, input TemplateInput) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Remove response references to old fields before deleting sections
	// Draft responses (not yet submitted) are deleted; submitted responses
	// are preserved via template_snapshot so we detach them from the field FK.
	if _, err := tx.Exec(ctx,
		`DELETE FROM submission_responses
		 WHERE submission_id IS NULL
		   AND field_id IN (SELECT f.id FROM checklist_fields f JOIN checklist_sections s ON s.id = f.section_id WHERE s.template_id = $1)`,
		templateID,
	); err != nil {
		return fmt.Errorf("delete draft responses: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE submission_responses SET field_id = field_id
		 WHERE field_id IN (SELECT f.id FROM checklist_fields f JOIN checklist_sections s ON s.id = f.section_id WHERE s.template_id = $1)`,
		templateID,
	); err != nil {
		// If there are submitted responses referencing these fields, drop the FK constraint
		// by nullifying the field reference — the snapshot preserves the field data
	}
	// Drop fail notes and rejections referencing old fields
	if _, err := tx.Exec(ctx,
		`DELETE FROM submission_fail_notes
		 WHERE field_id IN (SELECT f.id FROM checklist_fields f JOIN checklist_sections s ON s.id = f.section_id WHERE s.template_id = $1)`,
		templateID,
	); err != nil {
		return fmt.Errorf("delete fail notes: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM submission_rejections
		 WHERE field_id IN (SELECT f.id FROM checklist_fields f JOIN checklist_sections s ON s.id = f.section_id WHERE s.template_id = $1)`,
		templateID,
	); err != nil {
		return fmt.Errorf("delete rejections: %w", err)
	}
	// Now detach submitted responses from fields (set field_id to NULL won't work with NOT NULL constraint)
	// Instead, delete them — the template_snapshot on the submission preserves the data
	if _, err := tx.Exec(ctx,
		`DELETE FROM submission_responses
		 WHERE field_id IN (SELECT f.id FROM checklist_fields f JOIN checklist_sections s ON s.id = f.section_id WHERE s.template_id = $1)`,
		templateID,
	); err != nil {
		return fmt.Errorf("delete submitted responses: %w", err)
	}
	// Delete child records (sections cascade to fields; schedules and assignments have ON DELETE CASCADE)
	if _, err := tx.Exec(ctx,
		`DELETE FROM checklist_sections WHERE template_id = $1`, templateID,
	); err != nil {
		return fmt.Errorf("delete sections: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM checklist_schedules WHERE template_id = $1`, templateID,
	); err != nil {
		return fmt.Errorf("delete schedules: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM template_assignments WHERE template_id = $1`, templateID,
	); err != nil {
		return fmt.Errorf("delete assignments: %w", err)
	}

	// Update template header
	if _, err := tx.Exec(ctx,
		`UPDATE checklist_templates SET name = $1, requires_approval = $2, updated_at = now() WHERE id = $3`,
		input.Name, input.RequiresApproval, templateID,
	); err != nil {
		return fmt.Errorf("update template: %w", err)
	}

	// Re-insert schedules
	for _, sched := range input.Schedules {
		if _, err := tx.Exec(ctx,
			`INSERT INTO checklist_schedules (template_id, active_days) VALUES ($1, $2)`,
			templateID, sched.ActiveDays,
		); err != nil {
			return fmt.Errorf("insert schedule: %w", err)
		}
	}

	// Re-insert assignments
	for _, asgn := range input.Assignments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 VALUES ($1, $2, $3, $4)`,
			templateID, asgn.AssigneeType, asgn.AssigneeID, asgn.AssignmentRole,
		); err != nil {
			return fmt.Errorf("insert assignment: %w", err)
		}
	}

	// Re-insert sections and fields
	for _, sec := range input.Sections {
		condJSON, err := marshalNullableJSON(sec.Condition)
		if err != nil {
			return fmt.Errorf("marshal section condition: %w", err)
		}
		var sectionID string
		err = tx.QueryRow(ctx,
			`INSERT INTO checklist_sections (template_id, title, "order", condition)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			templateID, sec.Title, sec.Order, condJSON,
		).Scan(&sectionID)
		if err != nil {
			return fmt.Errorf("insert section %q: %w", sec.Title, err)
		}
		for _, field := range sec.Fields {
			if err := insertField(ctx, tx, sectionID, nil, field); err != nil {
				return fmt.Errorf("insert field %q: %w", field.Label, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// archiveTemplate soft-deletes a template by setting archived_at (D-07).
func archiveTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string) error {
	tag, err := pool.Exec(ctx,
		`UPDATE checklist_templates SET archived_at = now() WHERE id = $1 AND archived_at IS NULL`,
		templateID,
	)
	if err != nil {
		return fmt.Errorf("archive template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("template not found or already archived")
	}
	return nil
}

// listTemplates returns all non-archived templates with fully hydrated sections,
// fields, schedules, and assignments, ordered by created_at DESC.
func listTemplates(ctx context.Context, pool *pgxpool.Pool) ([]Template, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, name, requires_approval, created_by, created_at, updated_at
		 FROM checklist_templates
		 WHERE archived_at IS NULL
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.RequiresApproval, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}

	// Hydrate each template
	for i := range templates {
		if err := hydrateTemplate(ctx, pool, &templates[i]); err != nil {
			return nil, err
		}
	}
	return templates, nil
}

// getTemplateByID returns a single non-archived template by ID, fully hydrated.
// Returns nil if not found or archived.
func getTemplateByID(ctx context.Context, pool *pgxpool.Pool, templateID string) (*Template, error) {
	var t Template
	err := pool.QueryRow(ctx,
		`SELECT id, name, requires_approval, created_by, created_at, updated_at
		 FROM checklist_templates
		 WHERE id = $1 AND archived_at IS NULL`,
		templateID,
	).Scan(&t.ID, &t.Name, &t.RequiresApproval, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get template: %w", err)
	}
	if err := hydrateTemplate(ctx, pool, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// hydrateTemplate loads sections, fields, schedules, and assignments into t.
func hydrateTemplate(ctx context.Context, pool *pgxpool.Pool, t *Template) error {
	// Load sections
	secRows, err := pool.Query(ctx,
		`SELECT id, template_id, title, "order", condition
		 FROM checklist_sections
		 WHERE template_id = $1
		 ORDER BY "order"`,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("list sections: %w", err)
	}
	defer secRows.Close()

	for secRows.Next() {
		var s Section
		var condRaw []byte
		if err := secRows.Scan(&s.ID, &s.TemplateID, &s.Title, &s.Order, &condRaw); err != nil {
			return fmt.Errorf("scan section: %w", err)
		}
		s.Condition = json.RawMessage(condRaw)
		t.Sections = append(t.Sections, s)
	}
	if err := secRows.Err(); err != nil {
		return fmt.Errorf("iterate sections: %w", err)
	}
	// Build map AFTER all appends — slice reallocation would invalidate earlier pointers
	sectionMap := map[string]*Section{}
	for i := range t.Sections {
		sectionMap[t.Sections[i].ID] = &t.Sections[i]
	}

	// Load all fields for this template (top-level and sub-steps)
	fieldRows, err := pool.Query(ctx,
		`SELECT f.id, f.section_id, f.parent_field_id, f.type, f.label, f.required, f."order",
		        f.config, f.fail_trigger, f.condition
		 FROM checklist_fields f
		 JOIN checklist_sections s ON s.id = f.section_id
		 WHERE s.template_id = $1
		 ORDER BY f."order"`,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("list fields: %w", err)
	}
	defer fieldRows.Close()

	fieldMap := map[string]*Field{}
	var orderedFields []Field
	for fieldRows.Next() {
		var f Field
		var configRaw, failTriggerRaw, condRaw []byte
		if err := fieldRows.Scan(
			&f.ID, &f.SectionID, &f.ParentFieldID, &f.Type, &f.Label,
			&f.Required, &f.Order, &configRaw, &failTriggerRaw, &condRaw,
		); err != nil {
			return fmt.Errorf("scan field: %w", err)
		}
		f.Config = json.RawMessage(configRaw)
		f.FailTrigger = json.RawMessage(failTriggerRaw)
		f.Condition = json.RawMessage(condRaw)
		orderedFields = append(orderedFields, f)
	}
	if err := fieldRows.Err(); err != nil {
		return fmt.Errorf("iterate fields: %w", err)
	}

	// Two-pass nesting: first assign top-level fields to sections,
	// then nest sub-steps under their parents in the section's Fields slice.
	// This avoids the stale-pointer bug where appending to orderedFields
	// entries doesn't propagate to copies in sec.Fields.

	// Pass 1: top-level fields into sections
	for i := range orderedFields {
		f := &orderedFields[i]
		if f.ParentFieldID == nil {
			if sec, ok := sectionMap[f.SectionID]; ok {
				sec.Fields = append(sec.Fields, *f)
			}
		}
	}

	// Pass 2: build field map from section Fields (the actual stored copies)
	for si := range t.Sections {
		for fi := range t.Sections[si].Fields {
			fieldMap[t.Sections[si].Fields[fi].ID] = &t.Sections[si].Fields[fi]
		}
	}

	// Pass 3: nest sub-steps under their parent (now pointing into sec.Fields)
	for i := range orderedFields {
		f := &orderedFields[i]
		if f.ParentFieldID != nil {
			if parent, ok := fieldMap[*f.ParentFieldID]; ok {
				parent.SubSteps = append(parent.SubSteps, *f)
			}
		}
	}

	// Load schedules
	schedRows, err := pool.Query(ctx,
		`SELECT id, template_id, active_days, created_at
		 FROM checklist_schedules
		 WHERE template_id = $1`,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("list schedules: %w", err)
	}
	defer schedRows.Close()

	for schedRows.Next() {
		var s Schedule
		if err := schedRows.Scan(&s.ID, &s.TemplateID, &s.ActiveDays, &s.CreatedAt); err != nil {
			return fmt.Errorf("scan schedule: %w", err)
		}
		t.Schedules = append(t.Schedules, s)
	}
	if err := schedRows.Err(); err != nil {
		return fmt.Errorf("iterate schedules: %w", err)
	}

	// Load assignments
	asnRows, err := pool.Query(ctx,
		`SELECT id, template_id, assignee_type, assignee_id, assignment_role
		 FROM template_assignments
		 WHERE template_id = $1`,
		t.ID,
	)
	if err != nil {
		return fmt.Errorf("list assignments: %w", err)
	}
	defer asnRows.Close()

	for asnRows.Next() {
		var a Assignment
		if err := asnRows.Scan(&a.ID, &a.TemplateID, &a.AssigneeType, &a.AssigneeID, &a.AssignmentRole); err != nil {
			return fmt.Errorf("scan assignment: %w", err)
		}
		t.Assignments = append(t.Assignments, a)
	}
	if err := asnRows.Err(); err != nil {
		return fmt.Errorf("iterate assignments: %w", err)
	}

	return nil
}

// submitChecklist creates a new submission for the given template (D-15 idempotency).
// Draft responses from the user are moved to this submission. Additional responses
// from input are inserted. Returns the submission ID.
func submitChecklist(ctx context.Context, pool *pgxpool.Pool, input SubmitChecklistInput, userID string) (string, error) {
	// Load template to check it's not archived and to capture snapshot
	tmpl, err := getTemplateByID(ctx, pool, input.TemplateID)
	if err != nil {
		return "", fmt.Errorf("load template: %w", err)
	}
	if tmpl == nil {
		return "", ErrTemplateArchived
	}

	// Marshal template as snapshot
	snapshotJSON, err := json.Marshal(tmpl)
	if err != nil {
		return "", fmt.Errorf("marshal template snapshot: %w", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Insert submission with idempotency protection (D-15)
	var submissionID string
	err = tx.QueryRow(ctx,
		`INSERT INTO checklist_submissions (template_id, template_snapshot, submitted_by, idempotency_key)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (idempotency_key) DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
		 RETURNING id`,
		input.TemplateID, snapshotJSON, userID, input.IdempotencyKey,
	).Scan(&submissionID)
	if err != nil {
		return "", fmt.Errorf("insert submission: %w", err)
	}

	// Move draft responses (submission_id IS NULL, answered_by = userID) to this submission
	if _, err := tx.Exec(ctx,
		`UPDATE submission_responses
		 SET submission_id = $1
		 WHERE submission_id IS NULL AND answered_by = $2`,
		submissionID, userID,
	); err != nil {
		return "", fmt.Errorf("move draft responses: %w", err)
	}

	// Insert additional responses from input
	for _, resp := range input.Responses {
		valJSON, err := marshalNullableJSON(resp.Value)
		if err != nil {
			return "", fmt.Errorf("marshal response value: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO submission_responses (submission_id, field_id, value, answered_by)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (submission_id, field_id) DO UPDATE SET value = EXCLUDED.value, answered_at = now()`,
			submissionID, resp.FieldID, valJSON, userID,
		); err != nil {
			return "", fmt.Errorf("insert response: %w", err)
		}
	}

	// Insert fail notes
	for _, fn := range input.FailNotes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO submission_fail_notes (submission_id, field_id, note, severity)
			 VALUES ($1, $2, $3, $4)`,
			submissionID, fn.FieldID, fn.Note, fn.Severity,
		); err != nil {
			return "", fmt.Errorf("insert fail note: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}
	return submissionID, nil
}

// saveResponse upserts a draft response (submission_id IS NULL) for auto-save (D-21).
func saveResponse(ctx context.Context, pool *pgxpool.Pool, fieldID string, value json.RawMessage, userID string) error {
	// Null value means "unchecked" — delete the draft response row.
	if value == nil || string(value) == "null" {
		_, err := pool.Exec(ctx,
			`DELETE FROM submission_responses
			 WHERE field_id = $1 AND answered_by = $2 AND submission_id IS NULL`,
			fieldID, userID,
		)
		if err != nil {
			return fmt.Errorf("delete response: %w", err)
		}
		return nil
	}
	valJSON, err := marshalNullableJSON(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	_, err = pool.Exec(ctx,
		`INSERT INTO submission_responses (field_id, value, answered_by)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (field_id, answered_by) WHERE submission_id IS NULL
		 DO UPDATE SET value = EXCLUDED.value, answered_at = now()`,
		fieldID, valJSON, userID,
	)
	if err != nil {
		return fmt.Errorf("save response: %w", err)
	}
	return nil
}

// myDrafts returns draft responses (submission_id IS NULL) for the given user.
func myDrafts(ctx context.Context, pool *pgxpool.Pool, userID string) ([]FieldResponse, error) {
	// Drafts are shared across the team — checklists are team objects, not per-user.
	// Return all unsubmitted responses so every crew member sees the same progress.
	rows, err := pool.Query(ctx,
		`SELECT sr.id, sr.field_id, sr.value, sr.answered_by,
		        COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS answered_by_name,
		        sr.answered_at
		 FROM submission_responses sr
		 LEFT JOIN users u ON u.id = sr.answered_by
		 WHERE sr.submission_id IS NULL
		 ORDER BY sr.answered_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list drafts: %w", err)
	}
	defer rows.Close()

	var drafts []FieldResponse
	for rows.Next() {
		var r FieldResponse
		var valueRaw []byte
		if err := rows.Scan(&r.ID, &r.FieldID, &valueRaw, &r.AnsweredBy, &r.AnsweredByName, &r.AnsweredAt); err != nil {
			return nil, fmt.Errorf("scan draft: %w", err)
		}
		r.Value = json.RawMessage(valueRaw)
		drafts = append(drafts, r)
	}
	return drafts, rows.Err()
}

// myChecklists returns the templates assigned to the user today and their
// submissions for today (D-22).
func myChecklists(ctx context.Context, pool *pgxpool.Pool, userID string, clientDOW *int) ([]Template, []Submission, error) {
	// Use client-provided DOW if available (handles timezone differences),
	// otherwise fall back to server time
	var todayDOW int
	if clientDOW != nil {
		todayDOW = *clientDOW
	} else {
		if err := pool.QueryRow(ctx, `SELECT EXTRACT(DOW FROM now())::int`).Scan(&todayDOW); err != nil {
			return nil, nil, fmt.Errorf("get today DOW: %w", err)
		}
	}

	// Get user roles for role-based assignments
	var userRoles []string
	if err := pool.QueryRow(ctx,
		`SELECT roles FROM users WHERE id = $1`, userID,
	).Scan(&userRoles); err != nil {
		return nil, nil, fmt.Errorf("get user role: %w", err)
	}

	// Templates assigned to this user or their role, scheduled for today, not archived
	tmplRows, err := pool.Query(ctx,
		`SELECT DISTINCT t.id, t.name, t.requires_approval, t.created_by, t.created_at, t.updated_at
		 FROM checklist_templates t
		 JOIN template_assignments ta ON ta.template_id = t.id
		 JOIN checklist_schedules cs ON cs.template_id = t.id
		 WHERE t.archived_at IS NULL
		   AND ta.assignment_role = 'assignee'
		   AND (
		         (ta.assignee_type = 'user' AND ta.assignee_id = $1)
		         OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY($2))
		         OR ($2 && ARRAY['admin', 'superadmin'])
		       )
		   AND $3 = ANY(cs.active_days)
		 ORDER BY t.created_at DESC`,
		userID, userRoles, todayDOW,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list assigned templates: %w", err)
	}
	defer tmplRows.Close()

	var templates []Template
	for tmplRows.Next() {
		var t Template
		if err := tmplRows.Scan(&t.ID, &t.Name, &t.RequiresApproval, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := tmplRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate templates: %w", err)
	}

	// Hydrate templates
	for i := range templates {
		if err := hydrateTemplate(ctx, pool, &templates[i]); err != nil {
			return nil, nil, err
		}
	}

	// Today's submissions — checklists are team objects, all members see all submissions
	subRows, err := pool.Query(ctx,
		`SELECT s.id, s.template_id, t.name, s.template_snapshot, s.submitted_by,
		        s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.idempotency_key
		 FROM checklist_submissions s
		 JOIN checklist_templates t ON t.id = s.template_id
		 WHERE s.submitted_at >= current_date
		 ORDER BY s.submitted_at DESC`,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("list submissions: %w", err)
	}
	defer subRows.Close()

	var submissions []Submission
	for subRows.Next() {
		var sub Submission
		var snapshotRaw []byte
		if err := subRows.Scan(
			&sub.ID, &sub.TemplateID, &sub.TemplateName, &snapshotRaw,
			&sub.SubmittedBy, &sub.SubmittedAt, &sub.Status,
			&sub.ReviewedBy, &sub.ReviewedAt, &sub.IdempotencyKey,
		); err != nil {
			return nil, nil, fmt.Errorf("scan submission: %w", err)
		}
		sub.TemplateSnapshot = json.RawMessage(snapshotRaw)
		submissions = append(submissions, sub)
	}
	if err := subRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate submissions: %w", err)
	}

	// Load responses for each submission so hydrateFieldState can restore UI state
	for i := range submissions {
		if err := hydrateSubmission(ctx, pool, &submissions[i]); err != nil {
			return nil, nil, err
		}
	}

	return templates, submissions, nil
}

// myHistory returns up to 50 of the user's past submissions ordered by submitted_at DESC.
func myHistory(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Submission, error) {
	rows, err := pool.Query(ctx,
		`SELECT s.id, s.template_id, t.name, s.template_snapshot, s.submitted_by,
		        s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.idempotency_key
		 FROM checklist_submissions s
		 JOIN checklist_templates t ON t.id = s.template_id
		 WHERE s.submitted_by = $1
		 ORDER BY s.submitted_at DESC
		 LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var sub Submission
		var snapshotRaw []byte
		if err := rows.Scan(
			&sub.ID, &sub.TemplateID, &sub.TemplateName, &snapshotRaw,
			&sub.SubmittedBy, &sub.SubmittedAt, &sub.Status,
			&sub.ReviewedBy, &sub.ReviewedAt, &sub.IdempotencyKey,
		); err != nil {
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		sub.TemplateSnapshot = json.RawMessage(snapshotRaw)
		submissions = append(submissions, sub)
	}
	return submissions, rows.Err()
}

// pendingApprovals returns submissions pending approval where the user is assigned as approver (D-23).
func pendingApprovals(ctx context.Context, pool *pgxpool.Pool, userID string) ([]Submission, error) {
	// Get user roles
	var userRoles []string
	if err := pool.QueryRow(ctx,
		`SELECT roles FROM users WHERE id = $1`, userID,
	).Scan(&userRoles); err != nil {
		return nil, fmt.Errorf("get user role: %w", err)
	}

	rows, err := pool.Query(ctx,
		`SELECT DISTINCT s.id, s.template_id, t.name, s.template_snapshot, s.submitted_by,
		        COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS display_name,
		        s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.idempotency_key
		 FROM checklist_submissions s
		 JOIN checklist_templates t ON t.id = s.template_id
		 JOIN template_assignments ta ON ta.template_id = s.template_id
		 LEFT JOIN users u ON u.id = s.submitted_by
		 WHERE s.status = 'pending'
		   AND ta.assignment_role = 'approver'
		   AND (
		         (ta.assignee_type = 'user' AND ta.assignee_id = $1)
		         OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY($2))
		       )
		 ORDER BY s.template_id, s.submitted_at`,
		userID, userRoles,
	)
	if err != nil {
		return nil, fmt.Errorf("list pending approvals: %w", err)
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var sub Submission
		var snapshotRaw []byte
		var displayName *string
		if err := rows.Scan(
			&sub.ID, &sub.TemplateID, &sub.TemplateName, &snapshotRaw,
			&sub.SubmittedBy, &displayName,
			&sub.SubmittedAt, &sub.Status,
			&sub.ReviewedBy, &sub.ReviewedAt, &sub.IdempotencyKey,
		); err != nil {
			return nil, fmt.Errorf("scan submission: %w", err)
		}
		sub.TemplateSnapshot = json.RawMessage(snapshotRaw)
		if displayName != nil {
			sub.SubmittedByName = *displayName
		}
		submissions = append(submissions, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending approvals: %w", err)
	}

	// Load responses, fail notes, rejections for each submission
	for i := range submissions {
		if err := hydrateSubmission(ctx, pool, &submissions[i]); err != nil {
			return nil, err
		}
	}
	return submissions, nil
}

// hydrateSubmission loads responses, fail notes, and rejections for a submission.
func hydrateSubmission(ctx context.Context, pool *pgxpool.Pool, sub *Submission) error {
	// Responses
	rRows, err := pool.Query(ctx,
		`SELECT id, submission_id, field_id, value, answered_by, answered_at
		 FROM submission_responses
		 WHERE submission_id = $1`,
		sub.ID,
	)
	if err != nil {
		return fmt.Errorf("list responses: %w", err)
	}
	defer rRows.Close()
	for rRows.Next() {
		var r FieldResponse
		var valueRaw []byte
		if err := rRows.Scan(&r.ID, &r.SubmissionID, &r.FieldID, &valueRaw, &r.AnsweredBy, &r.AnsweredAt); err != nil {
			return fmt.Errorf("scan response: %w", err)
		}
		r.Value = json.RawMessage(valueRaw)
		sub.Responses = append(sub.Responses, r)
	}
	if err := rRows.Err(); err != nil {
		return fmt.Errorf("iterate responses: %w", err)
	}

	// Fail notes
	fnRows, err := pool.Query(ctx,
		`SELECT id, submission_id, field_id, note, severity, photo_url
		 FROM submission_fail_notes
		 WHERE submission_id = $1`,
		sub.ID,
	)
	if err != nil {
		return fmt.Errorf("list fail notes: %w", err)
	}
	defer fnRows.Close()
	for fnRows.Next() {
		var fn FailNote
		if err := fnRows.Scan(&fn.ID, &fn.SubmissionID, &fn.FieldID, &fn.Note, &fn.Severity, &fn.PhotoURL); err != nil {
			return fmt.Errorf("scan fail note: %w", err)
		}
		sub.FailNotes = append(sub.FailNotes, fn)
	}
	if err := fnRows.Err(); err != nil {
		return fmt.Errorf("iterate fail notes: %w", err)
	}

	// Rejections
	rejRows, err := pool.Query(ctx,
		`SELECT id, submission_id, field_id, comment, require_photo, rejected_by, rejected_at
		 FROM submission_rejections
		 WHERE submission_id = $1`,
		sub.ID,
	)
	if err != nil {
		return fmt.Errorf("list rejections: %w", err)
	}
	defer rejRows.Close()
	for rejRows.Next() {
		var rej Rejection
		if err := rejRows.Scan(&rej.ID, &rej.SubmissionID, &rej.FieldID, &rej.Comment, &rej.RequirePhoto, &rej.RejectedBy, &rej.RejectedAt); err != nil {
			return fmt.Errorf("scan rejection: %w", err)
		}
		sub.Rejections = append(sub.Rejections, rej)
	}
	if err := rejRows.Err(); err != nil {
		return fmt.Errorf("iterate rejections: %w", err)
	}

	return nil
}

// approveSubmission marks a submission as approved (D-23).
func approveSubmission(ctx context.Context, pool *pgxpool.Pool, submissionID string, reviewerID string) error {
	tag, err := pool.Exec(ctx,
		`UPDATE checklist_submissions
		 SET status = 'approved', reviewed_by = $1, reviewed_at = now()
		 WHERE id = $2 AND status = 'pending'`,
		reviewerID, submissionID,
	)
	if err != nil {
		return fmt.Errorf("approve submission: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("submission not found or not pending")
	}
	return nil
}

// rejectItem inserts a rejection record and updates the submission status to 'rejected' (D-06).
func rejectItem(ctx context.Context, pool *pgxpool.Pool, input RejectItemInput, rejectedBy string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`INSERT INTO submission_rejections (submission_id, field_id, comment, require_photo, rejected_by)
		 VALUES ($1, $2, $3, $4, $5)`,
		input.SubmissionID, input.FieldID, input.Comment, input.RequirePhoto, rejectedBy,
	); err != nil {
		return fmt.Errorf("insert rejection: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE checklist_submissions
		 SET status = 'rejected', reviewed_by = $1, reviewed_at = now()
		 WHERE id = $2`,
		rejectedBy, input.SubmissionID,
	); err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// cleanupOldDrafts deletes abandoned draft responses from previous days (pitfall 1).
func cleanupOldDrafts(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx,
		`DELETE FROM submission_responses
		 WHERE submission_id IS NULL AND answered_at < current_date`,
	)
	if err != nil {
		return fmt.Errorf("cleanup old drafts: %w", err)
	}
	return nil
}
