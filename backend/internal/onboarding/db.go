package onboarding

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Template represents an onboarding template with nested sections.
type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      *string   `json:"role"`
	Sections  []Section `json:"sections"`
	CreatedAt string    `json:"created_at"`
}

// Section represents a section within an onboarding template.
type Section struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	SortOrder       int    `json:"sort_order"`
	RequiresSignOff bool   `json:"requires_sign_off"`
	IsFaq           bool   `json:"is_faq"`
	Items           []Item `json:"items"`
}

// Item represents a single item in an onboarding section (checkbox, video_series, or faq).
type Item struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	Label      string      `json:"label"`
	Answer     *string     `json:"answer,omitempty"`
	SortOrder  int         `json:"sort_order"`
	VideoParts []VideoPart `json:"video_parts,omitempty"`
}

// VideoPart represents a single part of a video series item.
type VideoPart struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	SortOrder   int    `json:"sort_order"`
}

// SectionState is the computed state for a section during hire training.
type SectionState string

const (
	SectionLocked    SectionState = "locked"
	SectionActive    SectionState = "active"
	SectionComplete  SectionState = "complete"
	SectionSignedOff SectionState = "signed_off"
)

// SectionProgress wraps Section with computed state for a specific hire.
type SectionProgress struct {
	Section
	State   SectionState `json:"state"`
	SignOff *SignOffInfo `json:"sign_off,omitempty"`
}

// SignOffInfo holds sign-off data for a section.
type SignOffInfo struct {
	ManagerID   string `json:"manager_id"`
	Notes       string `json:"notes"`
	Rating      string `json:"rating"`
	SignedOffAt string `json:"signed_off_at"`
}

// HireTraining is the full training detail for a hire+template pair.
type HireTraining struct {
	Template Template          `json:"template"`
	Sections []SectionProgress `json:"sections"`
	Progress []ProgressEntry   `json:"progress"`
}

// ProgressEntry records a single progress item (item or video_part checked).
type ProgressEntry struct {
	ItemID       string `json:"item_id"`
	ProgressType string `json:"progress_type"`
	CheckedAt    string `json:"checked_at"`
}

// AssignedTemplate is a template assignment with computed progress for a hire.
type AssignedTemplate struct {
	TemplateID      string  `json:"template_id"`
	TemplateName    string  `json:"template_name"`
	Role            *string `json:"role"`
	AssignedAt      string  `json:"assigned_at"`
	ItemsChecked    int     `json:"items_checked"`
	TotalItems      int     `json:"total_items"`
	ProgressPercent int     `json:"progress_percent"`
}

// HireOverview summarizes a hire's onboarding state for the manager view.
type HireOverview struct {
	HireID          string `json:"hire_id"`
	DisplayName     string `json:"display_name"`
	Email           string `json:"email"`
	TemplateCount   int    `json:"template_count"`
	ItemsChecked    int    `json:"items_checked"`
	TotalItems      int    `json:"total_items"`
	ProgressPercent int    `json:"progress_percent"`
}

// SignOffInput is the input for the SignOff DB function.
type SignOffInput struct {
	ManagerID string `json:"manager_id"`
	SectionID string `json:"section_id"`
	HireID    string `json:"hire_id"`
	Notes     string `json:"notes"`
	Rating    string `json:"rating"`
}

// CreateTemplateInput is the input for creating or updating an onboarding template.
type CreateTemplateInput struct {
	Name     string               `json:"name"`
	Role     *string              `json:"role"`
	Sections []CreateSectionInput `json:"sections"`
}

// CreateSectionInput is the input for a section within a template.
type CreateSectionInput struct {
	Title           string            `json:"title"`
	SortOrder       int               `json:"sort_order"`
	RequiresSignOff bool              `json:"requires_sign_off"`
	IsFaq           bool              `json:"is_faq"`
	Items           []CreateItemInput `json:"items"`
}

// CreateItemInput is the input for a single item in a section.
type CreateItemInput struct {
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Answer     *string                `json:"answer,omitempty"`
	SortOrder  int                    `json:"sort_order"`
	VideoParts []CreateVideoPartInput `json:"video_parts,omitempty"`
}

// CreateVideoPartInput is the input for a video part.
type CreateVideoPartInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	SortOrder   int    `json:"sort_order"`
}

// GetTemplates returns all onboarding templates with nested structure.
func GetTemplates(ctx context.Context, pool *pgxpool.Pool) ([]Template, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, name, role, created_at
		FROM ob_templates
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("query ob_templates: %w", err)
	}
	defer rows.Close()

	var templates []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Role, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}

	for i := range templates {
		sections, err := getSections(ctx, pool, templates[i].ID)
		if err != nil {
			return nil, err
		}
		templates[i].Sections = sections
	}

	return templates, nil
}

// GetTemplate returns a single template by ID with nested structure.
func GetTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string) (*Template, error) {
	var t Template
	err := pool.QueryRow(ctx, `
		SELECT id, name, role, created_at
		FROM ob_templates
		WHERE id = $1
	`, templateID).Scan(&t.ID, &t.Name, &t.Role, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("query template: %w", err)
	}

	sections, err := getSections(ctx, pool, t.ID)
	if err != nil {
		return nil, err
	}
	t.Sections = sections
	return &t, nil
}

// getSections fetches all sections for a template with their items.
func getSections(ctx context.Context, pool *pgxpool.Pool, templateID string) ([]Section, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, title, sort_order, requires_sign_off, is_faq
		FROM ob_sections
		WHERE template_id = $1
		ORDER BY sort_order
	`, templateID)
	if err != nil {
		return nil, fmt.Errorf("query sections: %w", err)
	}
	defer rows.Close()

	var sections []Section
	for rows.Next() {
		var s Section
		if err := rows.Scan(&s.ID, &s.Title, &s.SortOrder, &s.RequiresSignOff, &s.IsFaq); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}
		sections = append(sections, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sections: %w", err)
	}

	for i := range sections {
		items, err := getItems(ctx, pool, sections[i].ID)
		if err != nil {
			return nil, err
		}
		sections[i].Items = items
	}

	return sections, nil
}

// getItems fetches all items for a section, including video parts for video_series items.
func getItems(ctx context.Context, pool *pgxpool.Pool, sectionID string) ([]Item, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, type, label, answer, sort_order
		FROM ob_items
		WHERE section_id = $1
		ORDER BY sort_order
	`, sectionID)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Type, &it.Label, &it.Answer, &it.SortOrder); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}

	for i := range items {
		if items[i].Type == "video_series" {
			parts, err := getVideoParts(ctx, pool, items[i].ID)
			if err != nil {
				return nil, err
			}
			items[i].VideoParts = parts
		}
	}

	return items, nil
}

// getVideoParts fetches all video parts for a video_series item.
func getVideoParts(ctx context.Context, pool *pgxpool.Pool, itemID string) ([]VideoPart, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, title, description, url, sort_order
		FROM ob_video_parts
		WHERE item_id = $1
		ORDER BY sort_order
	`, itemID)
	if err != nil {
		return nil, fmt.Errorf("query video parts: %w", err)
	}
	defer rows.Close()

	var parts []VideoPart
	for rows.Next() {
		var vp VideoPart
		if err := rows.Scan(&vp.ID, &vp.Title, &vp.Description, &vp.URL, &vp.SortOrder); err != nil {
			return nil, fmt.Errorf("scan video part: %w", err)
		}
		parts = append(parts, vp)
	}
	return parts, rows.Err()
}

// GetHireTraining returns the template with progress and computed section states for a hire.
func GetHireTraining(ctx context.Context, pool *pgxpool.Pool, hireID, templateID string) (*HireTraining, error) {
	tmpl, err := GetTemplate(ctx, pool, templateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	// Fetch all progress for this hire+template
	progressRows, err := pool.Query(ctx, `
		SELECT op.item_id, op.progress_type, op.checked_at
		FROM ob_progress op
		JOIN ob_items oi ON oi.id = op.item_id
		JOIN ob_sections os ON os.id = oi.section_id
		WHERE op.hire_id = $1 AND os.template_id = $2
	`, hireID, templateID)
	if err != nil {
		return nil, fmt.Errorf("query progress: %w", err)
	}
	defer progressRows.Close()

	// progressMap: "item_id:progress_type" -> true
	progressMap := map[string]bool{}
	var progressEntries []ProgressEntry
	for progressRows.Next() {
		var pe ProgressEntry
		if err := progressRows.Scan(&pe.ItemID, &pe.ProgressType, &pe.CheckedAt); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}
		progressMap[pe.ItemID+":"+pe.ProgressType] = true
		progressEntries = append(progressEntries, pe)
	}
	if err := progressRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate progress: %w", err)
	}

	// Fetch all signoffs for this hire+template
	signoffRows, err := pool.Query(ctx, `
		SELECT os2.section_id, os2.manager_id, os2.notes, os2.rating, os2.signed_off_at
		FROM ob_signoffs os2
		JOIN ob_sections osec ON osec.id = os2.section_id
		WHERE os2.hire_id = $1 AND osec.template_id = $2
	`, hireID, templateID)
	if err != nil {
		return nil, fmt.Errorf("query signoffs: %w", err)
	}
	defer signoffRows.Close()

	// signoffMap: section_id -> SignOffInfo
	signoffMap := map[string]*SignOffInfo{}
	for signoffRows.Next() {
		var sectionID string
		var so SignOffInfo
		if err := signoffRows.Scan(&sectionID, &so.ManagerID, &so.Notes, &so.Rating, &so.SignedOffAt); err != nil {
			return nil, fmt.Errorf("scan signoff: %w", err)
		}
		signoffMap[sectionID] = &so
	}
	if err := signoffRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signoffs: %w", err)
	}

	// Compute section states: locked/active/complete/signed_off
	sectionProgresses := make([]SectionProgress, len(tmpl.Sections))
	for i, sec := range tmpl.Sections {
		sp := SectionProgress{Section: sec}

		if so, ok := signoffMap[sec.ID]; ok {
			sp.State = SectionSignedOff
			sp.SignOff = so
		} else if isSectionComplete(sec, progressMap) {
			sp.State = SectionComplete
		} else if canActivateSection(i, sectionProgresses) {
			sp.State = SectionActive
		} else {
			sp.State = SectionLocked
		}

		sectionProgresses[i] = sp
	}

	return &HireTraining{
		Template: *tmpl,
		Sections: sectionProgresses,
		Progress: progressEntries,
	}, nil
}

// isSectionComplete returns true if all non-faq items in the section are checked.
// For FAQ sections, always returns true. For video_series, all parts must be watched.
func isSectionComplete(sec Section, progressMap map[string]bool) bool {
	if sec.IsFaq {
		return true
	}
	for _, item := range sec.Items {
		if item.Type == "faq" {
			continue
		}
		if item.Type == "video_series" {
			for _, part := range item.VideoParts {
				if !progressMap[part.ID+":video_part"] {
					return false
				}
			}
		} else {
			// checkbox
			if !progressMap[item.ID+":item"] {
				return false
			}
		}
	}
	return true
}

// canActivateSection returns true if idx==0 or all preceding sections are complete/signed_off.
func canActivateSection(idx int, sections []SectionProgress) bool {
	if idx == 0 {
		return true
	}
	for i := 0; i < idx; i++ {
		if sections[i].State != SectionComplete && sections[i].State != SectionSignedOff {
			return false
		}
	}
	return true
}

// GetMyTrainings returns all template assignments for a hire with progress percentages.
func GetMyTrainings(ctx context.Context, pool *pgxpool.Pool, hireID string) ([]AssignedTemplate, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			ota.template_id,
			ot.name,
			ot.role,
			ota.assigned_at,
			COALESCE(checked.cnt, 0) AS items_checked,
			COALESCE(total.cnt, 0)   AS total_items
		FROM ob_template_assignments ota
		JOIN ob_templates ot ON ot.id = ota.template_id
		LEFT JOIN (
			SELECT os.template_id, COUNT(*) AS cnt
			FROM ob_progress op
			JOIN ob_items oi ON oi.id = op.item_id
			JOIN ob_sections os ON os.id = oi.section_id
			WHERE op.hire_id = $1
			GROUP BY os.template_id
		) checked ON checked.template_id = ota.template_id
		LEFT JOIN (
			SELECT os.template_id, SUM(
				CASE oi.type
					WHEN 'video_series' THEN (
						SELECT COUNT(*) FROM ob_video_parts vp WHERE vp.item_id = oi.id
					)
					WHEN 'checkbox' THEN 1
					ELSE 0
				END
			) AS cnt
			FROM ob_items oi
			JOIN ob_sections os ON os.id = oi.section_id
			WHERE oi.type IN ('checkbox', 'video_series')
			GROUP BY os.template_id
		) total ON total.template_id = ota.template_id
		WHERE ota.hire_id = $1
		ORDER BY ota.assigned_at
	`, hireID)
	if err != nil {
		return nil, fmt.Errorf("query my trainings: %w", err)
	}
	defer rows.Close()

	var result []AssignedTemplate
	for rows.Next() {
		var at AssignedTemplate
		if err := rows.Scan(
			&at.TemplateID, &at.TemplateName, &at.Role, &at.AssignedAt,
			&at.ItemsChecked, &at.TotalItems,
		); err != nil {
			return nil, fmt.Errorf("scan assigned template: %w", err)
		}
		if at.TotalItems > 0 {
			at.ProgressPercent = (at.ItemsChecked * 100) / at.TotalItems
		}
		result = append(result, at)
	}
	return result, rows.Err()
}

// GetManagerHires returns hires with assigned templates and aggregate progress.
func GetManagerHires(ctx context.Context, pool *pgxpool.Pool) ([]HireOverview, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS display_name,
			u.email,
			COUNT(DISTINCT ota.template_id) AS template_count,
			COALESCE(SUM(checked.cnt), 0)   AS items_checked,
			COALESCE(SUM(total.cnt), 0)      AS total_items
		FROM users u
		JOIN ob_template_assignments ota ON ota.hire_id = u.id
		LEFT JOIN (
			SELECT op.hire_id, os.template_id, COUNT(*) AS cnt
			FROM ob_progress op
			JOIN ob_items oi ON oi.id = op.item_id
			JOIN ob_sections os ON os.id = oi.section_id
			GROUP BY op.hire_id, os.template_id
		) checked ON checked.hire_id = u.id AND checked.template_id = ota.template_id
		LEFT JOIN (
			SELECT os.template_id, SUM(
				CASE oi.type
					WHEN 'video_series' THEN (
						SELECT COUNT(*) FROM ob_video_parts vp WHERE vp.item_id = oi.id
					)
					WHEN 'checkbox' THEN 1
					ELSE 0
				END
			) AS cnt
			FROM ob_items oi
			JOIN ob_sections os ON os.id = oi.section_id
			WHERE oi.type IN ('checkbox', 'video_series')
			GROUP BY os.template_id
		) total ON total.template_id = ota.template_id
		GROUP BY u.id, display_name, u.email
		ORDER BY display_name
	`)
	if err != nil {
		return nil, fmt.Errorf("query manager hires: %w", err)
	}
	defer rows.Close()

	var result []HireOverview
	for rows.Next() {
		var ho HireOverview
		if err := rows.Scan(
			&ho.HireID, &ho.DisplayName, &ho.Email,
			&ho.TemplateCount, &ho.ItemsChecked, &ho.TotalItems,
		); err != nil {
			return nil, fmt.Errorf("scan hire overview: %w", err)
		}
		if ho.TotalItems > 0 {
			ho.ProgressPercent = (ho.ItemsChecked * 100) / ho.TotalItems
		}
		result = append(result, ho)
	}
	return result, rows.Err()
}

// SaveProgress records or removes an onboarding progress entry for a hire.
func SaveProgress(ctx context.Context, pool *pgxpool.Pool, hireID, itemID, progressType string, checked bool) error {
	if checked {
		_, err := pool.Exec(ctx, `
			INSERT INTO ob_progress (hire_id, item_id, progress_type)
			VALUES ($1, $2, $3)
			ON CONFLICT (hire_id, item_id, progress_type) DO NOTHING
		`, hireID, itemID, progressType)
		return err
	}
	_, err := pool.Exec(ctx, `
		DELETE FROM ob_progress
		WHERE hire_id = $1 AND item_id = $2 AND progress_type = $3
	`, hireID, itemID, progressType)
	return err
}

// SignOff records a manager sign-off for a section (idempotent on conflict).
func SignOff(ctx context.Context, pool *pgxpool.Pool, input SignOffInput) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO ob_signoffs (manager_id, section_id, hire_id, notes, rating)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (section_id, hire_id) DO NOTHING
	`, input.ManagerID, input.SectionID, input.HireID, input.Notes, input.Rating)
	return err
}

// CreateTemplate inserts a new onboarding template with all sections, items, and video parts.
func CreateTemplate(ctx context.Context, pool *pgxpool.Pool, input CreateTemplateInput) (string, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var templateID string
	err = tx.QueryRow(ctx, `
		INSERT INTO ob_templates (name, role)
		VALUES ($1, $2)
		RETURNING id
	`, input.Name, input.Role).Scan(&templateID)
	if err != nil {
		return "", fmt.Errorf("insert template: %w", err)
	}

	if err := insertSectionsTx(ctx, tx, templateID, input.Sections); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit transaction: %w", err)
	}
	return templateID, nil
}

// UpdateTemplate replaces a template's sections, items, and video parts entirely (full replace).
func UpdateTemplate(ctx context.Context, pool *pgxpool.Pool, templateID string, input CreateTemplateInput) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `
		UPDATE ob_templates SET name = $1, role = $2, updated_at = NOW()
		WHERE id = $3
	`, input.Name, input.Role, templateID)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}

	// Full replace: delete sections (items and video parts cascade via FK)
	_, err = tx.Exec(ctx, `DELETE FROM ob_sections WHERE template_id = $1`, templateID)
	if err != nil {
		return fmt.Errorf("delete old sections: %w", err)
	}

	if err := insertSectionsTx(ctx, tx, templateID, input.Sections); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// insertSectionsTx inserts sections + items + video_parts within an existing transaction.
func insertSectionsTx(ctx context.Context, tx pgx.Tx, templateID string, sections []CreateSectionInput) error {
	for _, sec := range sections {
		var sectionID string
		err := tx.QueryRow(ctx, `
			INSERT INTO ob_sections (template_id, title, sort_order, requires_sign_off, is_faq)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, templateID, sec.Title, sec.SortOrder, sec.RequiresSignOff, sec.IsFaq).Scan(&sectionID)
		if err != nil {
			return fmt.Errorf("insert section %q: %w", sec.Title, err)
		}

		for _, item := range sec.Items {
			var itemID string
			err := tx.QueryRow(ctx, `
				INSERT INTO ob_items (section_id, type, label, answer, sort_order)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
			`, sectionID, item.Type, item.Label, item.Answer, item.SortOrder).Scan(&itemID)
			if err != nil {
				return fmt.Errorf("insert item %q: %w", item.Label, err)
			}

			for _, vp := range item.VideoParts {
				_, err := tx.Exec(ctx, `
					INSERT INTO ob_video_parts (item_id, title, description, url, sort_order)
					VALUES ($1, $2, $3, $4, $5)
				`, itemID, vp.Title, vp.Description, vp.URL, vp.SortOrder)
				if err != nil {
					return fmt.Errorf("insert video part %q: %w", vp.Title, err)
				}
			}
		}
	}
	return nil
}

// AssignTemplate assigns an onboarding template to a hire (idempotent).
func AssignTemplate(ctx context.Context, pool *pgxpool.Pool, hireID, templateID, assignedBy string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO ob_template_assignments (hire_id, template_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (hire_id, template_id) DO NOTHING
	`, hireID, templateID, assignedBy)
	return err
}

// UnassignTemplate removes a template assignment from a hire.
func UnassignTemplate(ctx context.Context, pool *pgxpool.Pool, hireID, templateID string) error {
	_, err := pool.Exec(ctx, `
		DELETE FROM ob_template_assignments
		WHERE hire_id = $1 AND template_id = $2
	`, hireID, templateID)
	return err
}
