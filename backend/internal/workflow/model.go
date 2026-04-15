package workflow

import (
	"encoding/json"
	"time"
)

type Template struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	RequiresApproval bool            `json:"requires_approval"`
	CreatedBy        string          `json:"created_by"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	ArchivedAt       *time.Time      `json:"archived_at,omitempty"`
	Sections         []Section       `json:"sections,omitempty"`
	Schedules        []Schedule      `json:"schedules,omitempty"`
	Assignments      []Assignment    `json:"assignments,omitempty"`
}

type Schedule struct {
	ID         string    `json:"id"`
	TemplateID string    `json:"template_id"`
	ActiveDays []int     `json:"active_days"`
	CreatedAt  time.Time `json:"created_at"`
}

type Assignment struct {
	ID             string `json:"id"`
	TemplateID     string `json:"template_id"`
	AssigneeType   string `json:"assignee_type"`
	AssigneeID     string `json:"assignee_id"`
	AssignmentRole string `json:"assignment_role"`
}

type Section struct {
	ID         string          `json:"id"`
	TemplateID string          `json:"template_id"`
	Title      string          `json:"title"`
	Order      int             `json:"order"`
	Condition  json.RawMessage `json:"condition,omitempty"`
	Fields     []Field         `json:"fields,omitempty"`
}

type Field struct {
	ID            string          `json:"id"`
	SectionID     string          `json:"section_id"`
	ParentFieldID *string         `json:"parent_field_id,omitempty"`
	Type          string          `json:"type"`
	Label         string          `json:"label"`
	Required      bool            `json:"required"`
	Order         int             `json:"order"`
	Config        json.RawMessage `json:"config,omitempty"`
	FailTrigger   json.RawMessage `json:"fail_trigger,omitempty"`
	Condition     json.RawMessage `json:"condition,omitempty"`
	SubSteps      []Field         `json:"sub_steps,omitempty"`
}

type Submission struct {
	ID               string          `json:"id"`
	TemplateID       string          `json:"template_id"`
	TemplateName     string          `json:"template_name,omitempty"`
	TemplateSnapshot json.RawMessage `json:"template_snapshot"`
	SubmittedBy      string          `json:"submitted_by"`
	SubmittedByName  string          `json:"submitted_by_name,omitempty"`
	SubmittedAt      time.Time       `json:"submitted_at"`
	Status           string          `json:"status"`
	ReviewedBy       *string         `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time      `json:"reviewed_at,omitempty"`
	IdempotencyKey   *string         `json:"idempotency_key,omitempty"`
	Responses        []FieldResponse `json:"responses,omitempty"`
	FailNotes        []FailNote      `json:"fail_notes,omitempty"`
	Rejections       []Rejection     `json:"rejections,omitempty"`
}

type FieldResponse struct {
	ID           string          `json:"id"`
	SubmissionID *string         `json:"submission_id,omitempty"`
	FieldID      string          `json:"field_id"`
	Value        json.RawMessage `json:"value"`
	AnsweredBy   string          `json:"answered_by"`
	AnsweredAt   time.Time       `json:"answered_at"`
}

type FailNote struct {
	ID           string  `json:"id"`
	SubmissionID string  `json:"submission_id"`
	FieldID      string  `json:"field_id"`
	Note         string  `json:"note"`
	Severity     *string `json:"severity,omitempty"`
	PhotoURL     *string `json:"photo_url,omitempty"`
}

type Rejection struct {
	ID           string    `json:"id"`
	SubmissionID string    `json:"submission_id"`
	FieldID      string    `json:"field_id"`
	Comment      string    `json:"comment"`
	RequirePhoto bool      `json:"require_photo"`
	RejectedBy   string    `json:"rejected_by"`
	RejectedAt   time.Time `json:"rejected_at"`
}

// Input types for API requests

type TemplateInput struct {
	Name             string            `json:"name"`
	RequiresApproval bool              `json:"requires_approval"`
	Sections         []SectionInput    `json:"sections"`
	Schedules        []ScheduleInput   `json:"schedules,omitempty"`
	Assignments      []AssignmentInput `json:"assignments,omitempty"`
}

type SectionInput struct {
	Title     string          `json:"title"`
	Order     int             `json:"order"`
	Condition json.RawMessage `json:"condition,omitempty"`
	Fields    []FieldInput    `json:"fields"`
}

type FieldInput struct {
	Type        string          `json:"type"`
	Label       string          `json:"label"`
	Required    bool            `json:"required"`
	Order       int             `json:"order"`
	Config      json.RawMessage `json:"config,omitempty"`
	FailTrigger json.RawMessage `json:"fail_trigger,omitempty"`
	Condition   json.RawMessage `json:"condition,omitempty"`
	SubSteps    []FieldInput    `json:"sub_steps,omitempty"`
}

type ScheduleInput struct {
	ActiveDays []int `json:"active_days"`
}

type AssignmentInput struct {
	AssigneeType   string `json:"assignee_type"`
	AssigneeID     string `json:"assignee_id"`
	AssignmentRole string `json:"assignment_role"`
}

type SaveResponseInput struct {
	FieldID string          `json:"field_id"`
	Value   json.RawMessage `json:"value"`
}

type SubmitChecklistInput struct {
	TemplateID     string              `json:"template_id"`
	IdempotencyKey string              `json:"idempotency_key"`
	Responses      []SaveResponseInput `json:"responses,omitempty"`
	FailNotes      []FailNoteInput     `json:"fail_notes,omitempty"`
}

type FailNoteInput struct {
	FieldID  string  `json:"field_id"`
	Note     string  `json:"note"`
	Severity *string `json:"severity,omitempty"`
}

type RejectItemInput struct {
	SubmissionID string `json:"submission_id"`
	FieldID      string `json:"field_id"`
	Comment      string `json:"comment"`
	RequirePhoto bool   `json:"require_photo"`
}
