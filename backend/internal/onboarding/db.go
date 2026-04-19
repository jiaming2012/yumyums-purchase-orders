package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Template represents an onboarding template with nested sections.
type Template struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Roles     []string  `json:"roles"`
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
	// Checked is populated when returning HireTraining — true if this item is checked by the hire.
	Checked bool `json:"checked"`
	// Viewed is populated for FAQ items — true if the hire has expanded/viewed this FAQ.
	Viewed bool `json:"viewed,omitempty"`
}

// VideoPart represents a single part of a video series item.
type VideoPart struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	SortOrder   int    `json:"sort_order"`
	// Checked is populated when returning HireTraining — true if this part is watched by the hire.
	Checked bool `json:"checked"`
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
	TemplateID        string   `json:"template_id"`
	TemplateName      string   `json:"template_name"`
	Roles             []string `json:"roles"`
	AssignedAt        string   `json:"assigned_at"`
	SectionsComplete  int      `json:"sections_complete"`
	SectionsTotal     int      `json:"sections_total"`
	ProgressPercent   int      `json:"progress_percent"`
}

// AssignedTemplateSummary is a minimal summary of a template assignment for the manager hire list.
type AssignedTemplateSummary struct {
	TemplateID   string `json:"template_id"`
	TemplateName string `json:"template_name"`
	ProgressPct  int    `json:"progress_pct"`
}

// HireOverview summarizes a hire's onboarding state for the manager view.
type HireOverview struct {
	HireID            string                    `json:"hire_id"`
	DisplayName       string                    `json:"display_name"`
	Email             string                    `json:"email"`
	TemplateCount     int                       `json:"template_count"`
	SectionsComplete  int                       `json:"sections_complete"`
	SectionsTotal     int                       `json:"sections_total"`
	ProgressPercent   int                       `json:"progress_percent"`
	AssignedTemplates []AssignedTemplateSummary `json:"assigned_templates"`
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
	Roles    []string             `json:"roles"`
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
		SELECT id, name, roles, created_at
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
		var createdAt time.Time
		if err := rows.Scan(&t.ID, &t.Name, &t.Roles, &createdAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		t.CreatedAt = createdAt.UTC().Format(time.RFC3339)
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
	var createdAt time.Time
	err := pool.QueryRow(ctx, `
		SELECT id, name, roles, created_at
		FROM ob_templates
		WHERE id = $1
	`, templateID).Scan(&t.ID, &t.Name, &t.Roles, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("query template: %w", err)
	}
	t.CreatedAt = createdAt.UTC().Format(time.RFC3339)

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
		var checkedAt time.Time
		if err := progressRows.Scan(&pe.ItemID, &pe.ProgressType, &checkedAt); err != nil {
			return nil, fmt.Errorf("scan progress: %w", err)
		}
		pe.CheckedAt = checkedAt.UTC().Format(time.RFC3339)
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
		var signedOffAt time.Time
		if err := signoffRows.Scan(&sectionID, &so.ManagerID, &so.Notes, &so.Rating, &signedOffAt); err != nil {
			return nil, fmt.Errorf("scan signoff: %w", err)
		}
		so.SignedOffAt = signedOffAt.UTC().Format(time.RFC3339)
		signoffMap[sectionID] = &so
	}
	if err := signoffRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signoffs: %w", err)
	}

	// Compute section states: locked/active/complete/signed_off
	// Also populate Checked on each item/video_part using progressMap.
	sectionProgresses := make([]SectionProgress, len(tmpl.Sections))
	for i, sec := range tmpl.Sections {
		// Populate Checked on items and video parts
		for j := range sec.Items {
			item := &sec.Items[j]
			if item.Type == "video_series" {
				for k := range item.VideoParts {
					item.VideoParts[k].Checked = progressMap[item.VideoParts[k].ID+":video_part"]
				}
			} else if item.Type == "faq" {
				item.Viewed = progressMap[item.ID+":faq"]
			} else {
				item.Checked = progressMap[item.ID+":item"]
			}
		}

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

// isSectionComplete returns true if all items in the section are done.
// For FAQ sections, all FAQ items must be viewed. For video_series, all parts must be watched.
func isSectionComplete(sec Section, progressMap map[string]bool) bool {
	for _, item := range sec.Items {
		if item.Type == "faq" {
			if !progressMap[item.ID+":faq"] {
				return false
			}
		} else if item.Type == "video_series" {
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
	return len(sec.Items) > 0
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

// GetMyTrainings returns training templates for a hire — both explicitly assigned
// and role-auto-assigned (template.roles overlaps user.roles).
func GetMyTrainings(ctx context.Context, pool *pgxpool.Pool, hireID string) ([]AssignedTemplate, error) {
	// Get user roles for auto-assignment matching
	var userRoles []string
	if err := pool.QueryRow(ctx, `SELECT roles FROM users WHERE id = $1`, hireID).Scan(&userRoles); err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT
			ot.id AS template_id,
			ot.name,
			ot.roles,
			COALESCE(ota.assigned_at, ot.created_at) AS assigned_at,
			COALESCE(sec_complete.cnt, 0) AS sections_complete,
			COALESCE(sec_total.cnt, 0)    AS sections_total
		FROM ob_templates ot
		LEFT JOIN ob_template_assignments ota ON ota.template_id = ot.id AND ota.hire_id = $1
		LEFT JOIN (
			-- Count sections where ALL checkable items have progress
			SELECT os.template_id, COUNT(*) AS cnt
			FROM ob_sections os
			WHERE NOT EXISTS (
				-- Find any checkable item in this section WITHOUT progress
				SELECT 1 FROM ob_items oi
				WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series')
				AND NOT EXISTS (
					SELECT 1 FROM ob_progress op
					WHERE op.item_id = oi.id AND op.hire_id = $1
				)
			)
			-- Only count sections that have at least one checkable item
			AND EXISTS (
				SELECT 1 FROM ob_items oi
				WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series')
			)
			GROUP BY os.template_id
		) sec_complete ON sec_complete.template_id = ot.id
		LEFT JOIN (
			-- Count total sections per template
			SELECT template_id, COUNT(*) AS cnt
			FROM ob_sections
			GROUP BY template_id
		) sec_total ON sec_total.template_id = ot.id
		WHERE ota.hire_id = $1
		   OR (ot.roles IS NOT NULL AND ot.roles && $2)
		ORDER BY assigned_at
	`, hireID, userRoles)
	if err != nil {
		return nil, fmt.Errorf("query my trainings: %w", err)
	}
	defer rows.Close()

	var result []AssignedTemplate
	for rows.Next() {
		var at AssignedTemplate
		var assignedAt time.Time
		if err := rows.Scan(
			&at.TemplateID, &at.TemplateName, &at.Roles, &assignedAt,
			&at.SectionsComplete, &at.SectionsTotal,
		); err != nil {
			return nil, fmt.Errorf("scan assigned template: %w", err)
		}
		at.AssignedAt = assignedAt.UTC().Format(time.RFC3339)
		if at.SectionsTotal > 0 {
			at.ProgressPercent = (at.SectionsComplete * 100) / at.SectionsTotal
		}
		result = append(result, at)
	}
	return result, rows.Err()
}

// GetManagerHires returns hires with assigned templates and aggregate progress.
// Includes both explicitly assigned and role-auto-assigned templates.
func GetManagerHires(ctx context.Context, pool *pgxpool.Pool) ([]HireOverview, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			u.id,
			COALESCE(NULLIF(u.nickname, ''), u.first_name || ' ' || LEFT(u.last_name, 1) || '.') AS display_name,
			u.email,
			COUNT(DISTINCT matched_tpl.id) AS template_count,
			COALESCE(SUM(sec_complete.cnt), 0) AS sections_complete,
			COALESCE(SUM(sec_total.cnt), 0)    AS sections_total
		FROM users u
		JOIN ob_templates matched_tpl ON (
			EXISTS (SELECT 1 FROM ob_template_assignments ota WHERE ota.hire_id = u.id AND ota.template_id = matched_tpl.id)
			OR (matched_tpl.roles IS NOT NULL AND matched_tpl.roles && u.roles)
		)
		LEFT JOIN (
			SELECT os.template_id, COUNT(*) AS cnt
			FROM ob_sections os
			GROUP BY os.template_id
		) sec_total ON sec_total.template_id = matched_tpl.id
		LEFT JOIN (
			SELECT os.template_id, u2.id AS hire_id, COUNT(*) AS cnt
			FROM ob_sections os
			CROSS JOIN users u2
			WHERE NOT EXISTS (
				SELECT 1 FROM ob_items oi
				WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series', 'faq')
				AND NOT EXISTS (
					SELECT 1 FROM ob_progress op
					WHERE op.item_id = oi.id AND op.hire_id = u2.id
				)
			)
			AND EXISTS (
				SELECT 1 FROM ob_items oi
				WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series', 'faq')
			)
			GROUP BY os.template_id, u2.id
		) sec_complete ON sec_complete.template_id = matched_tpl.id AND sec_complete.hire_id = u.id
		WHERE u.status = 'active'
		GROUP BY u.id, display_name, u.email
		HAVING COUNT(DISTINCT matched_tpl.id) > 0
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
			&ho.TemplateCount, &ho.SectionsComplete, &ho.SectionsTotal,
		); err != nil {
			return nil, fmt.Errorf("scan hire overview: %w", err)
		}
		if ho.SectionsTotal > 0 {
			ho.ProgressPercent = (ho.SectionsComplete * 100) / ho.SectionsTotal
		}
		ho.AssignedTemplates = []AssignedTemplateSummary{}
		result = append(result, ho)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hire overviews: %w", err)
	}

	// Fetch per-hire template details (explicit + role-auto-assigned)
	for i := range result {
		var userRoles []string
		_ = pool.QueryRow(ctx, `SELECT roles FROM users WHERE id = $1`, result[i].HireID).Scan(&userRoles)

		tmplRows, err := pool.Query(ctx, `
			SELECT
				ot.id,
				ot.name,
				COALESCE(sec_total.cnt, 0) AS sections_total,
				COALESCE(sec_complete.cnt, 0) AS sections_complete
			FROM ob_templates ot
			LEFT JOIN (
				SELECT template_id, COUNT(*) AS cnt FROM ob_sections GROUP BY template_id
			) sec_total ON sec_total.template_id = ot.id
			LEFT JOIN (
				SELECT os.template_id, COUNT(*) AS cnt
				FROM ob_sections os
				WHERE NOT EXISTS (
					SELECT 1 FROM ob_items oi
					WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series', 'faq')
					AND NOT EXISTS (
						SELECT 1 FROM ob_progress op WHERE op.item_id = oi.id AND op.hire_id = $1
					)
				)
				AND EXISTS (
					SELECT 1 FROM ob_items oi WHERE oi.section_id = os.id AND oi.type IN ('checkbox', 'video_series', 'faq')
				)
				GROUP BY os.template_id
			) sec_complete ON sec_complete.template_id = ot.id
			WHERE EXISTS (SELECT 1 FROM ob_template_assignments ota WHERE ota.hire_id = $1 AND ota.template_id = ot.id)
			   OR (ot.roles IS NOT NULL AND ot.roles && $2)
			ORDER BY ot.name
		`, result[i].HireID, userRoles)
		if err != nil {
			return nil, fmt.Errorf("query assigned templates for hire %s: %w", result[i].HireID, err)
		}
		for tmplRows.Next() {
			var ts AssignedTemplateSummary
			var secTotal, secComplete int
			if err := tmplRows.Scan(&ts.TemplateID, &ts.TemplateName, &secTotal, &secComplete); err != nil {
				tmplRows.Close()
				return nil, fmt.Errorf("scan assigned template summary: %w", err)
			}
			if secTotal > 0 {
				ts.ProgressPct = (secComplete * 100) / secTotal
			}
			result[i].AssignedTemplates = append(result[i].AssignedTemplates, ts)
		}
		tmplRows.Close()
		if err := tmplRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate assigned templates for hire %s: %w", result[i].HireID, err)
		}
	}

	return result, nil
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
		INSERT INTO ob_templates (name, roles)
		VALUES ($1, $2)
		RETURNING id
	`, input.Name, input.Roles).Scan(&templateID)
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
		UPDATE ob_templates SET name = $1, roles = $2, updated_at = NOW()
		WHERE id = $3
	`, input.Name, input.Roles, templateID)
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
