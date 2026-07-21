package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/auth"
	opsync "github.com/yumyums/hq/internal/sync"
)

// ErrTemplateArchived is returned when submitting a checklist for an archived template.
var ErrTemplateArchived = errors.New("template is archived")

// ErrUnknownField is returned when a response write names a field that does not
// exist in the current template (FR-3, INV-4). The save-response path surfaces
// it as HTTP 422 {"error":"unknown_field"} so the runner rolls the optimistic
// checkmark back instead of writing under a dead field id. It is an app-level
// existence check, NOT a restored FK — submitted responses reference
// template_snapshot ids by design, and an FK would break them.
var ErrUnknownField = errors.New("unknown_field")

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
// Used by insertTemplate.
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
			if _, err := insertField(ctx, tx, sectionID, nil, field); err != nil {
				return "", fmt.Errorf("insert field %q: %w", field.Label, err)
			}
		}
	}

	return templateID, nil
}

// updateTemplate diff-upserts a template's content against the field IDs the
// Builder already sends in toApiTemplate (FR-2, INV-2). Fields named with an
// existing id are UPDATED in place — keeping one permanent checklist_fields.id
// for life — genuinely new fields are INSERTED, and fields absent from the input
// are DELETED. Section rows are reused positionally so surviving fields keep a
// valid home. Conditions are remapped for NEW fields only; kept fields keep
// their id, so their draft responses, fail notes, and conditions never churn.
//
// This replaces the old delete-and-reinsert path that minted a fresh id for
// every field on each edit — the Friday P0 root cause. With stable identity a
// multi-device write always lands on the same real field, and field-id churn
// becomes structurally impossible.
func updateTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string, input TemplateInput) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := updateTemplateInTx(ctx, tx, templateID, input); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// updateTemplateAndEmit performs the template write and emits its SAVE_TEMPLATE
// op inside ONE transaction (FR-5, INV-1). The REST Builder-save path
// (UpdateTemplateHandler) uses this so the op that tells other devices to
// re-fetch + re-render is durably queued ATOMICALLY with the write — replacing
// the fire-and-forget opsync.EmitOp goroutine that could leave an accepted write
// with no queued op. The /ops OpRouter path keeps calling the bare
// updateTemplate: its op is already recorded by the sync layer, so emitting here
// too would double-queue.
func updateTemplateAndEmit(ctx context.Context, pool *pgxpool.Pool, templateID string, input TemplateInput, userID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if err := updateTemplateInTx(ctx, tx, templateID, input); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"template_id": templateID})
	if err != nil {
		return fmt.Errorf("marshal save_template op payload: %w", err)
	}
	if _, err := opsync.EmitOpTx(ctx, tx, opsync.OpInput{
		DeviceID:   "server",
		UserID:     userID,
		EntityID:   templateID,
		EntityType: "template",
		OpType:     opsync.OpSaveTemplate,
		Payload:    json.RawMessage(payload),
	}); err != nil {
		return fmt.Errorf("emit save_template op: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// updateTemplateInTx is the transaction-scoped core of the template write. Both
// updateTemplate (bare, used by the /ops OpRouter) and updateTemplateAndEmit
// (REST Builder save, with a transactional op) drive it.
func updateTemplateInTx(ctx context.Context, tx pgx.Tx, templateID string, input TemplateInput) error {
	// Update template header in place.
	if _, err := tx.Exec(ctx,
		`UPDATE checklist_templates SET name = $1, requires_approval = $2, updated_at = now() WHERE id = $3`,
		input.Name, input.RequiresApproval, templateID,
	); err != nil {
		return fmt.Errorf("update template: %w", err)
	}

	// Schedules and assignments carry no stable identity and are referenced by
	// nothing — delete and re-insert wholesale.
	if _, err := tx.Exec(ctx, `DELETE FROM checklist_schedules WHERE template_id = $1`, templateID); err != nil {
		return fmt.Errorf("delete schedules: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM template_assignments WHERE template_id = $1`, templateID); err != nil {
		return fmt.Errorf("delete assignments: %w", err)
	}
	for _, sched := range input.Schedules {
		if _, err := tx.Exec(ctx,
			`INSERT INTO checklist_schedules (template_id, active_days) VALUES ($1, $2)`,
			templateID, sched.ActiveDays,
		); err != nil {
			return fmt.Errorf("insert schedule: %w", err)
		}
	}
	for _, asgn := range input.Assignments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 VALUES ($1, $2, $3, $4)`,
			templateID, asgn.AssigneeType, asgn.AssigneeID, asgn.AssignmentRole,
		); err != nil {
			return fmt.Errorf("insert assignment: %w", err)
		}
	}

	// Existing section ids (ordered, for positional reuse) and the set of field
	// ids currently in the template (top-level + sub-steps).
	existingSectionIDs, err := loadSectionIDs(ctx, tx, templateID)
	if err != nil {
		return err
	}
	existingFieldIDs, err := loadFieldIDs(ctx, tx, templateID)
	if err != nil {
		return err
	}

	keptFields := map[string]bool{}     // existing ids named in the input → UPDATE
	remap := map[string]string{}        // new field client-id → generated id (conditions)
	reusedSections := map[string]bool{} // section ids reused this pass

	for i, sec := range input.Sections {
		condJSON, err := marshalNullableJSON(sec.Condition)
		if err != nil {
			return fmt.Errorf("marshal section condition: %w", err)
		}
		var sectionID string
		if i < len(existingSectionIDs) {
			sectionID = existingSectionIDs[i]
			if _, err := tx.Exec(ctx,
				`UPDATE checklist_sections SET title = $1, "order" = $2, condition = $3 WHERE id = $4`,
				sec.Title, sec.Order, condJSON, sectionID,
			); err != nil {
				return fmt.Errorf("update section %q: %w", sec.Title, err)
			}
		} else {
			if err := tx.QueryRow(ctx,
				`INSERT INTO checklist_sections (template_id, title, "order", condition)
				 VALUES ($1, $2, $3, $4)
				 RETURNING id`,
				templateID, sec.Title, sec.Order, condJSON,
			).Scan(&sectionID); err != nil {
				return fmt.Errorf("insert section %q: %w", sec.Title, err)
			}
		}
		reusedSections[sectionID] = true

		for _, field := range sec.Fields {
			if err := upsertField(ctx, tx, sectionID, nil, field, existingFieldIDs, keptFields, remap); err != nil {
				return err
			}
		}
	}

	// Delete removed fields (existing − kept). Sub-steps first because
	// checklist_fields.parent_field_id has no ON DELETE CASCADE.
	var removed []string
	for id := range existingFieldIDs {
		if !keptFields[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		// Discard crew's UNSUBMITTED answers on the cut fields (INV-6: the Builder
		// warned the admin, who confirmed — the loss is a warned operator action).
		// The field_id FK was dropped (0051/0053/0054), so without this the draft
		// rows would orphan and keep counting against the checklist. Only drafts
		// (submission_id IS NULL) are removed — submitted responses are historical
		// record and reference template-snapshot ids, so they are untouched.
		if _, err := tx.Exec(ctx,
			`DELETE FROM submission_responses WHERE field_id = ANY($1) AND submission_id IS NULL`, removed,
		); err != nil {
			return fmt.Errorf("discard removed-field drafts: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM checklist_fields WHERE id = ANY($1) AND parent_field_id IS NOT NULL`, removed,
		); err != nil {
			return fmt.Errorf("delete removed sub-steps: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM checklist_fields WHERE id = ANY($1)`, removed,
		); err != nil {
			return fmt.Errorf("delete removed fields: %w", err)
		}
	}

	// Delete sections no longer used. Kept fields were re-pointed above, so any
	// rows still in these sections are removed fields (already deleted); the
	// ON DELETE CASCADE on section_id is a safety net only.
	var unusedSections []string
	for _, id := range existingSectionIDs {
		if !reusedSections[id] {
			unusedSections = append(unusedSections, id)
		}
	}
	if len(unusedSections) > 0 {
		if _, err := tx.Exec(ctx,
			`DELETE FROM checklist_sections WHERE id = ANY($1)`, unusedSections,
		); err != nil {
			return fmt.Errorf("delete unused sections: %w", err)
		}
	}

	// Remap condition field_id references for NEW fields only. Kept fields keep
	// their id, so any condition referencing them is already correct.
	for oldID, newID := range remap {
		if _, err := tx.Exec(ctx,
			`UPDATE checklist_fields SET condition = jsonb_set(condition, '{field_id}', to_jsonb($1::text))
			 WHERE section_id IN (SELECT id FROM checklist_sections WHERE template_id = $2)
			   AND condition->>'field_id' = $3`,
			newID, templateID, oldID,
		); err != nil {
			return fmt.Errorf("remap condition field_id %s→%s: %w", oldID, newID, err)
		}
	}

	return nil
}

// loadSectionIDs returns the template's section ids ordered by "order".
func loadSectionIDs(ctx context.Context, tx pgx.Tx, templateID string) ([]string, error) {
	rows, err := tx.Query(ctx,
		`SELECT id FROM checklist_sections WHERE template_id = $1 ORDER BY "order"`, templateID)
	if err != nil {
		return nil, fmt.Errorf("load section ids: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan section id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// loadFieldIDs returns the set of field ids (top-level + sub-steps) in the template.
func loadFieldIDs(ctx context.Context, tx pgx.Tx, templateID string) (map[string]bool, error) {
	rows, err := tx.Query(ctx,
		`SELECT f.id FROM checklist_fields f
		 JOIN checklist_sections s ON s.id = f.section_id
		 WHERE s.template_id = $1`, templateID)
	if err != nil {
		return nil, fmt.Errorf("load field ids: %w", err)
	}
	defer rows.Close()
	ids := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan field id: %w", err)
		}
		ids[id] = true
	}
	return ids, rows.Err()
}

// upsertField UPDATEs a field in place when the Builder names it with an id that
// already exists (preserving its permanent checklist_fields.id and re-pointing
// its section/parent), or INSERTs it as a new field. Recurses into sub-steps.
// A new field carrying a client-provided id is recorded in remap so conditions
// referencing that placeholder id can be rewritten to the generated id.
func upsertField(ctx context.Context, tx pgx.Tx, sectionID string, parentFieldID *string, field FieldInput, existing, kept map[string]bool, remap map[string]string) error {
	if field.ID != "" && existing[field.ID] {
		configJSON, err := marshalNullableJSON(field.Config)
		if err != nil {
			return fmt.Errorf("marshal config: %w", err)
		}
		failTriggerJSON, err := marshalNullableJSON(field.FailTrigger)
		if err != nil {
			return fmt.Errorf("marshal fail_trigger: %w", err)
		}
		conditionJSON, err := marshalNullableJSON(field.Condition)
		if err != nil {
			return fmt.Errorf("marshal condition: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE checklist_fields
			 SET section_id = $1, parent_field_id = $2, type = $3, label = $4,
			     required = $5, "order" = $6, config = $7, fail_trigger = $8, condition = $9
			 WHERE id = $10`,
			sectionID, parentFieldID, field.Type, field.Label,
			field.Required, field.Order, configJSON, failTriggerJSON, conditionJSON, field.ID,
		); err != nil {
			return fmt.Errorf("update field %q: %w", field.Label, err)
		}
		kept[field.ID] = true
		parentID := field.ID
		for _, sub := range field.SubSteps {
			if err := upsertField(ctx, tx, sectionID, &parentID, sub, existing, kept, remap); err != nil {
				return err
			}
		}
		return nil
	}

	// Genuinely new field — insert with a fresh id (recurses into sub-steps).
	newID, err := insertField(ctx, tx, sectionID, parentFieldID, field)
	if err != nil {
		return fmt.Errorf("insert field %q: %w", field.Label, err)
	}
	if field.ID != "" {
		remap[field.ID] = newID
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
	// App-level existence check (FR-3, INV-4): a write naming a field absent from
	// the current template is rejected loudly with ErrUnknownField (→ 422). The
	// field_id FK was dropped (migrations 0051/0053/0054) so a dead-id write would
	// otherwise be accepted silently — the churn-driven Friday P0 symptom.
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM checklist_fields WHERE id = $1)`, fieldID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check field exists: %w", err)
	}
	if !exists {
		return ErrUnknownField
	}

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
		        COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS submitted_by_name,
		        s.submitted_at, s.status, s.reviewed_by, s.reviewed_at, s.idempotency_key
		 FROM checklist_submissions s
		 JOIN checklist_templates t ON t.id = s.template_id
		 LEFT JOIN users u ON u.id = s.submitted_by
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
		var displayName *string
		if err := subRows.Scan(
			&sub.ID, &sub.TemplateID, &sub.TemplateName, &snapshotRaw,
			&sub.SubmittedBy, &displayName,
			&sub.SubmittedAt, &sub.Status,
			&sub.ReviewedBy, &sub.ReviewedAt, &sub.IdempotencyKey,
		); err != nil {
			return nil, nil, fmt.Errorf("scan submission: %w", err)
		}
		if displayName != nil {
			sub.SubmittedByName = *displayName
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
		   AND t.archived_at IS NULL
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

// unsubmitChecklist deletes a submission and moves its responses back to drafts.
// Only the submitter can unsubmit, and only if not yet approved.
func unsubmitChecklist(ctx context.Context, pool *pgxpool.Pool, submissionID, userID string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Verify the submission exists, belongs to this user, and is not approved
	var status, submittedBy string
	err = tx.QueryRow(ctx,
		`SELECT status, submitted_by FROM checklist_submissions WHERE id = $1`,
		submissionID,
	).Scan(&status, &submittedBy)
	if err != nil {
		return fmt.Errorf("get submission: %w", err)
	}
	if submittedBy != userID {
		return fmt.Errorf("not the submitter")
	}
	if status == "approved" {
		return fmt.Errorf("cannot unsubmit approved checklist")
	}

	// Move submitted responses back to drafts (detach from submission)
	if _, err := tx.Exec(ctx,
		`UPDATE submission_responses SET submission_id = NULL WHERE submission_id = $1`,
		submissionID,
	); err != nil {
		return fmt.Errorf("detach responses: %w", err)
	}

	// Move fail notes back to drafts
	if _, err := tx.Exec(ctx,
		`UPDATE submission_fail_notes SET submission_id = NULL WHERE submission_id = $1`,
		submissionID,
	); err != nil {
		return fmt.Errorf("detach fail notes: %w", err)
	}

	// Delete rejections for this submission
	if _, err := tx.Exec(ctx,
		`DELETE FROM submission_rejections WHERE submission_id = $1`,
		submissionID,
	); err != nil {
		return fmt.Errorf("delete rejections: %w", err)
	}

	// Delete the submission record
	if _, err := tx.Exec(ctx,
		`DELETE FROM checklist_submissions WHERE id = $1`,
		submissionID,
	); err != nil {
		return fmt.Errorf("delete submission: %w", err)
	}

	return tx.Commit(ctx)
}

// countDraftHolders returns the number of DISTINCT crew members who have an
// unsubmitted draft response (submission_id IS NULL) dated today on any of the
// given field IDs. It powers the Builder's INV-6 discard warning: before a save
// that cuts those fields (or, when passed the whole template's field set, drops
// today from the schedule), the admin is told how many crew would lose answers.
// "Today" mirrors cleanupOldDrafts (answered_at >= current_date) — drafts are
// day-scoped. Read-only; returns 0 for an empty field set without a query.
func countDraftHolders(ctx context.Context, pool *pgxpool.Pool, fieldIDs []string) (int, error) {
	if len(fieldIDs) == 0 {
		return 0, nil
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(DISTINCT answered_by)
		 FROM submission_responses
		 WHERE submission_id IS NULL
		   AND answered_at >= current_date
		   AND field_id = ANY($1::uuid[])`,
		fieldIDs,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("count draft holders: %w", err)
	}
	return n, nil
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

// canReviewSubmission answers whether user may approve or reject submissionID
// (design `prove-surface-gating-and-endpoints.md` §8 amendment 4 — the B5
// fold-in). The role rule was fixed at slate time and is not a judgment call
// here:
//
//	allowed ⇔ approver assignment on the submission's template
//	          ∨ admin role
//	          ∨ superadmin
//
// "Approver assignment" means exactly what PendingApprovalsHandler already
// means by it (see pendingApprovals above): a template_assignments row with
// assignment_role = 'approver' matching by user id or by role. Reusing that
// definition is the point — the set of submissions a user can act on must equal
// the set their Approvals tab shows them, or the UI lies in one direction or
// the other.
//
// Two non-obvious exclusions, both pinned by tests:
//   - an 'assignee' assignment is NOT an approver assignment, so a crew member
//     assigned to FILL a checklist cannot approve their own submission;
//   - an approver assignment on a DIFFERENT template does not carry over — the
//     check resolves the submission's own template rather than asking "is this
//     user an approver anywhere".
//
// A missing submission returns false, not an error: an unknown id is not
// something the caller is entitled to act on either way.
func canReviewSubmission(ctx context.Context, pool *pgxpool.Pool, user *auth.User, submissionID string) (bool, error) {
	if user == nil {
		return false, nil
	}
	if user.IsSuperadmin {
		return true, nil
	}
	for _, r := range user.Roles {
		if r == "admin" {
			return true, nil
		}
	}

	var ok bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM checklist_submissions s
			JOIN template_assignments ta ON ta.template_id = s.template_id
			WHERE s.id = $1
			  AND ta.assignment_role = 'approver'
			  AND (
			        (ta.assignee_type = 'user' AND ta.assignee_id = $2)
			     OR (ta.assignee_type = 'role' AND ta.assignee_id = ANY($3))
			  )
		)`, submissionID, user.ID, user.Roles).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("check approver assignment: %w", err)
	}
	return ok, nil
}
