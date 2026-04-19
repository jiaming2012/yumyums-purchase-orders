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
	ID             string          `json:"id"`
	SubmissionID   *string         `json:"submission_id,omitempty"`
	FieldID        string          `json:"field_id"`
	Value          json.RawMessage `json:"value"`
	AnsweredBy     string          `json:"answered_by"`
	AnsweredByName string          `json:"answered_by_name,omitempty"`
	AnsweredAt     time.Time       `json:"answered_at"`
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

// JSONB is a json.RawMessage that can also be unmarshaled from YAML.
// YAML parses {"key": "val"} as a map, so we re-marshal it to JSON bytes.
type JSONB json.RawMessage

func (j *JSONB) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	if v == nil {
		*j = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	*j = b
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return []byte(j), nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = JSONB(data)
	return nil
}

type TemplateInput struct {
	Name             string            `json:"name" yaml:"name"`
	RequiresApproval bool              `json:"requires_approval" yaml:"requires_approval"`
	Sections         []SectionInput    `json:"sections" yaml:"sections"`
	Schedules        []ScheduleInput   `json:"schedules,omitempty" yaml:"schedules"`
	Assignments      []AssignmentInput `json:"assignments,omitempty" yaml:"assignments"`
}

type SectionInput struct {
	Title     string       `json:"title" yaml:"title"`
	Order     int          `json:"order" yaml:"order"`
	Condition JSONB        `json:"condition,omitempty" yaml:"condition"`
	Fields    []FieldInput `json:"fields" yaml:"fields"`
}

type FieldInput struct {
	Type        string       `json:"type" yaml:"type"`
	Label       string       `json:"label" yaml:"label"`
	Required    bool         `json:"required" yaml:"required"`
	Order       int          `json:"order" yaml:"order"`
	Config      JSONB        `json:"config,omitempty" yaml:"config"`
	FailTrigger JSONB        `json:"fail_trigger,omitempty" yaml:"fail_trigger"`
	Condition   JSONB        `json:"condition,omitempty" yaml:"condition"`
	SubSteps    []FieldInput `json:"sub_steps,omitempty" yaml:"sub_steps"`
}

type ScheduleInput struct {
	ActiveDays []int `json:"active_days" yaml:"active_days"`
}

type AssignmentInput struct {
	AssigneeType   string `json:"assignee_type" yaml:"assignee_type"`
	AssigneeID     string `json:"assignee_id" yaml:"assignee_id"`
	AssignmentRole string `json:"assignment_role" yaml:"assignment_role"`
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
