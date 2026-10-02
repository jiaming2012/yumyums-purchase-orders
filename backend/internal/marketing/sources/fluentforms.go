package sources

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	// The MySQL driver is registered for its side effect only. The website's
	// Fluent Forms tables live in a WordPress MySQL on Hostinger; HQ's own
	// store is and stays Postgres.
	_ "github.com/go-sql-driver/mysql"
)

// Env vars the reader is gated behind. ALL FIVE must be present or
// NewFluentForms refuses with ErrFluentFormsNotConfigured — which is what every
// automated gate hits, because the night never sets them.
const (
	EnvFFHost     = "FF_DB_HOST"
	EnvFFUser     = "FF_DB_USER"
	EnvFFPassword = "FF_DB_PASSWORD"
	EnvFFName     = "FF_DB_NAME"
	EnvFFPrefix   = "FF_DB_PREFIX"
)

// ErrFluentFormsNotConfigured is the refusal when the five env vars are not all
// set. The handler turns it into 503 {"error":"ff_not_configured"} — fail loud,
// never a silent "imported 0".
var ErrFluentFormsNotConfigured = errors.New("fluent forms reader not configured (FF_DB_HOST/USER/PASSWORD/NAME/PREFIX)")

// ffSubmission is one row of `<prefix>fluentform_submissions`: the id, the
// creation timestamp, and the response JSON blob the form stored.
//
// It is exported-by-shape through the fixture file, not through Go: tests read
// testdata/fluentforms_submissions.json, which is the real column set the
// 2026-10-01 spike saw.
type ffSubmission struct {
	ID        int64  `json:"id"`
	FormID    int64  `json:"form_id"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
}

// ffResponse is §8's REAL key set, corrected by spike 01 run 1.
//
// 🛑 It is NOT the website's own count_customers.py key set. That script's
// defaults (FF_PHONE_FIELD=phone, FF_SOURCE_FIELD=source) do not match the live
// form, which is why its SMS-opt-in and ad-source breakdowns count nothing
// today — GAP-H5-1, recorded against the WEBSITE repo and deliberately NOT
// fixed here. Building this adapter from that script's assumptions is the
// specific mistake the spike was run to prevent.
type ffResponse struct {
	Names struct {
		FirstName string `json:"first_name"`
	} `json:"names"`
	Email string `json:"email"`
	// InputText is the PHONE NUMBER. The live form's phone field is literally
	// named `input_text`.
	InputText string `json:"input_text"`
	// Checkbox is the consent channel array, e.g. ["Phone","Email"]. Membership
	// of "Phone" is SMS consent; membership of "Email" is email consent.
	Checkbox []string `json:"checkbox"`
	// Source is the hidden first-touch field. ABSENT on the live submission the
	// spike inspected, hence a plain string that stays "" rather than a
	// required field that would red the whole import.
	Source string `json:"source"`
}

// The two consent channel labels, exactly as the live form spells them.
const (
	consentLabelPhone = "Phone"
	consentLabelEmail = "Email"
)

// FluentForms reads the website's signup submissions. The `rows` func is the
// seam: the live implementation queries MySQL, and the fixture implementation
// reads the committed JSON — so every mapping assertion runs with no database
// of any kind.
type FluentForms struct {
	rows func() ([]ffSubmission, error)
}

// Source implements Importer.
func (f *FluentForms) Source() string { return SourceWebForm }

// NewFluentForms builds the LIVE reader from the environment.
//
// 🛑 Calling this with the five env vars set OPENS A CONNECTION TO THE LIVE
// WEBSITE DATABASE. The night never sets them, so the night never connects:
// every automated path gets ErrFluentFormsNotConfigured, and the first live
// import is an attended operator act (testdata/README.md).
func NewFluentForms() (*FluentForms, error) {
	cfg := map[string]string{}
	for _, k := range []string{EnvFFHost, EnvFFUser, EnvFFPassword, EnvFFName, EnvFFPrefix} {
		v := strings.TrimSpace(os.Getenv(k))
		if v == "" {
			return nil, fmt.Errorf("%w: %s is empty", ErrFluentFormsNotConfigured, k)
		}
		cfg[k] = v
	}
	host := cfg[EnvFFHost]
	if !strings.Contains(host, ":") {
		host += ":3306"
	}
	// parseTime=false on purpose: created_at comes back as a string and
	// parseCreatedAt owns the one timestamp format, shared with the fixture
	// path. Two parsers is two behaviors.
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&timeout=10s&readTimeout=30s",
		cfg[EnvFFUser], cfg[EnvFFPassword], host, cfg[EnvFFName])
	table := cfg[EnvFFPrefix] + "fluentform_submissions"
	return &FluentForms{rows: func() ([]ffSubmission, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, fmt.Errorf("fluent forms open: %w", err)
		}
		defer db.Close()
		db.SetMaxOpenConns(1)
		// The table name is interpolated because a prefix cannot be a bind
		// parameter. It comes from the operator's own env var, and the
		// allowlist below is what keeps that from being an injection seam.
		if !safeIdentifier(table) {
			return nil, fmt.Errorf("fluent forms: refusing table name %q (FF_DB_PREFIX must be [A-Za-z0-9_])", table)
		}
		q := "SELECT id, form_id, created_at, response FROM " + table + " ORDER BY id DESC LIMIT 5000"
		rs, err := db.Query(q)
		if err != nil {
			return nil, fmt.Errorf("fluent forms query: %w", err)
		}
		defer rs.Close()
		var out []ffSubmission
		for rs.Next() {
			var s ffSubmission
			if err := rs.Scan(&s.ID, &s.FormID, &s.CreatedAt, &s.Response); err != nil {
				return nil, fmt.Errorf("fluent forms scan: %w", err)
			}
			out = append(out, s)
		}
		return out, rs.Err()
	}}, nil
}

// NewFluentFormsFromJSON builds the reader over a JSON array of submissions —
// the committed fixture's shape. This is the constructor the tests use, and it
// touches no network.
func NewFluentFormsFromJSON(raw []byte) (*FluentForms, error) {
	var subs []ffSubmission
	if err := json.Unmarshal(raw, &subs); err != nil {
		return nil, fmt.Errorf("fluent forms fixture: %w", err)
	}
	return &FluentForms{rows: func() ([]ffSubmission, error) { return subs, nil }}, nil
}

func safeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
		default:
			return false
		}
	}
	return true
}

// Read maps every submission onto a Candidate.
//
// A submission whose response JSON will not parse is SKIPPED, not fatal: one
// malformed row from a form plugin update must not cost the operator the other
// 1,183 signups. A submission with neither a usable phone nor an email is also
// skipped — there is nothing to reach that person by and nothing to dedupe on.
func (f *FluentForms) Read() ([]Candidate, error) {
	subs, err := f.rows()
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(subs))
	for _, s := range subs {
		var resp ffResponse
		if err := json.Unmarshal([]byte(s.Response), &resp); err != nil {
			continue
		}
		phone := NormalizeE164(resp.InputText)
		email := strings.TrimSpace(resp.Email)
		if phone == "" && email == "" {
			continue
		}
		joined := parseCreatedAt(s.CreatedAt)
		c := Candidate{
			DisplayName:  strings.TrimSpace(resp.Names.FirstName),
			PhoneE164:    phone,
			Email:        email,
			Source:       SourceWebForm,
			SourceShort:  strings.ToUpper(strings.TrimSpace(resp.Source)),
			SMSConsent:   containsFold(resp.Checkbox, consentLabelPhone),
			EmailConsent: containsFold(resp.Checkbox, consentLabelEmail),
			JoinedAt:     joined,
			ExternalRef:  strconv.FormatInt(s.ID, 10),
		}
		c.ConsentEvidence = ConsentEvidenceLine(
			fmt.Sprintf("form checkbox %s", renderChecked(resp.Checkbox)), joined)
		out = append(out, c)
	}
	return out, nil
}

// renderChecked is what the consent trail shows: which boxes were ticked, in
// the form's own words. "none" when the submitter ticked nothing — which is a
// real and important state, not a blank.
func renderChecked(boxes []string) string {
	clean := make([]string, 0, len(boxes))
	for _, b := range boxes {
		if b = strings.TrimSpace(b); b != "" {
			clean = append(clean, b)
		}
	}
	if len(clean) == 0 {
		return "none"
	}
	return "[" + strings.Join(clean, ",") + "]"
}

func containsFold(hay []string, needle string) bool {
	for _, h := range hay {
		if strings.EqualFold(strings.TrimSpace(h), needle) {
			return true
		}
	}
	return false
}

// ffTimeLayouts is MySQL's DATETIME spelling first, then the two shapes a
// driver with parseTime or a JSON fixture can hand over.
var ffTimeLayouts = []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05"}

// parseCreatedAt falls back to now() rather than failing the row: a signup with
// an unreadable timestamp is still a signup, and `joined_at` is NOT NULL.
func parseCreatedAt(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	for _, l := range ffTimeLayouts {
		if t, err := time.Parse(l, raw); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}
