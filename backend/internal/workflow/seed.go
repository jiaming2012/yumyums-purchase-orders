package workflow

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/yaml.v3"
)

type templatesConfig struct {
	Templates []TemplateInput `yaml:"templates"`
}

// LoadTemplateConfig parses the templates.yaml file and returns a slice of
// TemplateInput ready for seeding. Follows the LoadSuperadmins pattern.
func LoadTemplateConfig(path string) ([]TemplateInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read templates config: %w", err)
	}
	var cfg templatesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse templates config: %w", err)
	}
	return cfg.Templates, nil
}

// SeedTemplates inserts templates from the provided list into the database.
// It is idempotent — templates with the same name are skipped.
// createdBy is the user ID to set as the template creator (typically a superadmin).
func SeedTemplates(ctx context.Context, pool *pgxpool.Pool, templates []TemplateInput, createdBy string) error {
	for _, tmpl := range templates {
		if err := seedTemplate(ctx, pool, tmpl, createdBy); err != nil {
			return fmt.Errorf("seed template %q: %w", tmpl.Name, err)
		}
	}
	return nil
}

func seedTemplate(ctx context.Context, pool *pgxpool.Pool, tmpl TemplateInput, createdBy string) error {
	// Check if template with this name already exists (idempotent on name)
	var existing int
	err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM checklist_templates WHERE name = $1 AND archived_at IS NULL`,
		tmpl.Name,
	).Scan(&existing)
	if err != nil {
		return fmt.Errorf("check existing template: %w", err)
	}
	if existing > 0 {
		log.Printf("Template %q already exists, skipping", tmpl.Name)
		return nil
	}

	// Insert template + all child records in a single transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var templateID string
	err = tx.QueryRow(ctx,
		`INSERT INTO checklist_templates (name, requires_approval, created_by)
		 VALUES ($1, $2, $3)
		 RETURNING id`,
		tmpl.Name, tmpl.RequiresApproval, createdBy,
	).Scan(&templateID)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}

	// Insert schedules
	for _, sched := range tmpl.Schedules {
		_, err = tx.Exec(ctx,
			`INSERT INTO checklist_schedules (template_id, active_days) VALUES ($1, $2)`,
			templateID, sched.ActiveDays,
		)
		if err != nil {
			return fmt.Errorf("insert schedule: %w", err)
		}
	}

	// Insert assignments
	for _, asgn := range tmpl.Assignments {
		_, err = tx.Exec(ctx,
			`INSERT INTO template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 VALUES ($1, $2, $3, $4)`,
			templateID, asgn.AssigneeType, asgn.AssigneeID, asgn.AssignmentRole,
		)
		if err != nil {
			return fmt.Errorf("insert assignment: %w", err)
		}
	}

	// Insert sections and their fields
	for _, sec := range tmpl.Sections {
		var sectionID string
		condJSON, err := marshalNullableJSON(sec.Condition)
		if err != nil {
			return fmt.Errorf("marshal section condition: %w", err)
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO checklist_sections (template_id, title, "order", condition)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			templateID, sec.Title, sec.Order, condJSON,
		).Scan(&sectionID)
		if err != nil {
			return fmt.Errorf("insert section %q: %w", sec.Title, err)
		}

		// Insert top-level fields
		for _, field := range sec.Fields {
			if _, err := insertField(ctx, tx, sectionID, nil, field); err != nil {
				return fmt.Errorf("insert field %q: %w", field.Label, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	log.Printf("Seeded template %q (id=%s)", tmpl.Name, templateID)
	return nil
}

// insertField inserts a single field and recursively inserts its sub-steps.
func insertField(ctx context.Context, tx pgx.Tx, sectionID string, parentFieldID *string, field FieldInput) (string, error) {
	configJSON, err := marshalNullableJSON(field.Config)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}
	failTriggerJSON, err := marshalNullableJSON(field.FailTrigger)
	if err != nil {
		return "", fmt.Errorf("marshal fail_trigger: %w", err)
	}
	conditionJSON, err := marshalNullableJSON(field.Condition)
	if err != nil {
		return "", fmt.Errorf("marshal condition: %w", err)
	}

	var fieldID string
	err = tx.QueryRow(ctx,
		`INSERT INTO checklist_fields
		   (section_id, parent_field_id, type, label, required, "order", config, fail_trigger, condition)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		sectionID, parentFieldID, field.Type, field.Label,
		field.Required, field.Order, configJSON, failTriggerJSON, conditionJSON,
	).Scan(&fieldID)
	if err != nil {
		return "", fmt.Errorf("insert field row: %w", err)
	}

	// Insert sub-steps with this field as parent
	for _, sub := range field.SubSteps {
		if _, err := insertField(ctx, tx, sectionID, &fieldID, sub); err != nil {
			return "", fmt.Errorf("insert sub-step %q: %w", sub.Label, err)
		}
	}
	return fieldID, nil
}

// marshalNullableJSON returns nil if the input is empty/null, otherwise
// returns the raw bytes for use as a JSONB parameter.
func marshalNullableJSON(raw []byte) (interface{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	return raw, nil
}
