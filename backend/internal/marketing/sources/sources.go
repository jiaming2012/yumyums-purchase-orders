// Package sources holds the three subscriber adapters behind ONE interface, and
// the two pure functions every one of them (and the read path) depends on:
// E.164 normalization and the masked render.
//
// # Why one interface
//
// A subscriber arrives from four places (§4's `source` enum) and the list must
// not care which. Each adapter's whole job is to turn its own wire shape into
// []Candidate; internal/marketing/subscribers.go owns the single upsert that
// lands them, so idempotency, phone dedupe and the `signed_up` event are
// written ONCE rather than three times with three sets of bugs.
//
// # Why E.164 is here and not in the handler
//
// Spike `e164-normalize-mask` (2026-10-01, exit 0 first run) proved the phone
// handling is a CLOSED, enumerated set: eight spellings of one number collapse
// to one E.164 value, short/empty/nil yield nothing, and the list's masked
// render is the last four of the normalized value. That makes the normalized
// value the dedupe key — so it has to be computed identically by every adapter,
// which means exactly one implementation.
//
// 🛑 NOTHING IN THIS PACKAGE SENDS. No SMS, no email, no outbound HTTP of any
// kind. Sending is Activity E's, and `marketing.TestNothingInThisPackageSends`
// parses the import graph of this package and of internal/marketing to keep it
// that way.
package sources

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// The §4 `source` enum values, so no adapter spells one wrong.
const (
	SourceWebForm     = "web_form"
	SourceToastImport = "toast_import"
	SourceSMSKeyword  = "sms_keyword"
	SourceQR          = "qr"
)

// Candidate is one subscriber as an adapter saw it: the §4 columns, nothing
// derived. The caller normalizes nothing and masks nothing — an adapter hands
// over a finished Candidate or it hands over an error.
type Candidate struct {
	// DisplayName is the human label. Empty means "unknown" and lands NULL.
	DisplayName string
	// PhoneE164 is ALREADY normalized (NormalizeE164 returned it) or empty.
	PhoneE164 string
	Email     string
	// Source is one of the four constants above.
	Source string
	// SourceShort is the first-touch campaign short, or empty. Often empty:
	// the live signup form carries no source field (spike 01's correction).
	SourceShort string
	SMSConsent  bool
	// EmailConsent is separate from SMSConsent because the live form's
	// `checkbox` array can carry either, both, or neither.
	EmailConsent bool
	// ConsentEvidence is the audit line the detail sheet's consent trail
	// renders — "form checkbox ['Phone','Email'] 2026-09-28".
	ConsentEvidence string
	// OptedOut, when true, lands `opted_out_at` and makes the row's consent
	// state "stop". It is only ever set from a §4 column an adapter actually
	// read; this card invents no new consent semantics (that is the operator's
	// compliance call, and is PARKED).
	OptedOut bool
	JoinedAt time.Time
	// ExternalRef is the adapter's own idempotency handle — a Fluent Forms
	// submission id or a Toast guest id. Empty means "this source has no
	// stable id", and the upsert falls back to the phone.
	ExternalRef string
}

// Importer is the one interface. Read() is deliberately the WHOLE contract:
// an adapter reads its own input and returns candidates. It does not touch
// Postgres, so an adapter can be exercised with no database at all — which is
// what the committed Fluent Forms fixture does.
type Importer interface {
	// Source returns the §4 enum value every candidate from this importer
	// carries, so the caller can scope its idempotency lookup.
	Source() string
	// Read returns the candidates this importer sees right now.
	Read() ([]Candidate, error)
}

// ── E.164 (spike 02, verbatim logic) ──

var nonDigit = regexp.MustCompile(`\D`)

// NormalizeE164 collapses the spellings the three sources emit into one +1
// E.164 value, and returns "" for anything that is not a 10-digit US number.
//
// The enumerated set spike 02 asserted, all → "+17735554821":
//
//	"(773) 555-4821"  "773-555-4821"  "7735554821"
//	"+1 773 555 4821" "17735554821"
//
// and all → "": "555-4821", "", a nil/absent field.
//
// It is deliberately NOT a general phone parser. The truck is in one country
// and a wrong guess at a country code is a message to a stranger.
func NormalizeE164(raw string) string {
	d := nonDigit.ReplaceAllString(raw, "")
	if len(d) == 11 && strings.HasPrefix(d, "1") {
		d = d[1:]
	}
	if len(d) != 10 {
		return ""
	}
	return "+1" + d
}

// Last4 is the only part of a phone number that may cross to the browser.
// Returns "" when there is no normalized value to take four digits from.
func Last4(e164 string) string {
	if len(e164) < 4 {
		return ""
	}
	return e164[len(e164)-4:]
}

// MaskPhone renders the list's phone cell: "•••• 4821". Empty in, empty out —
// the UI renders "No phone" for that, not a row of bullets with nothing after
// them.
//
// Format decision (this card's, per the slate's PARK note: "the mask format …
// is the night's"): four bullets, one space, the last four digits. It is what
// spike 02 printed, it is unambiguous at 393px, and it never implies a digit
// count the number does not have.
func MaskPhone(e164 string) string {
	l4 := Last4(e164)
	if l4 == "" {
		return ""
	}
	return "•••• " + l4
}

// MaskEmail renders the list's email cell: "j•••@gmail.com".
//
// Format decision (this card's): the first character of the local part, three
// bullets, then the domain VERBATIM. The domain is kept because "is this the
// gmail one or the work one?" is the question a manager actually asks of this
// cell, and a domain is not a contact handle on its own. A one-character local
// part masks to "•••@domain" rather than leaking the whole thing.
//
// Empty in, empty out. Anything with no "@" is treated as a local part with no
// domain and masks to "•••" — a malformed address never renders raw.
func MaskEmail(email string) string {
	e := strings.TrimSpace(email)
	if e == "" {
		return ""
	}
	at := strings.LastIndex(e, "@")
	if at <= 0 {
		return "•••"
	}
	local, domain := e[:at], e[at+1:]
	if domain == "" {
		return "•••"
	}
	if len(local) < 2 {
		return "•••@" + domain
	}
	return local[:1] + "•••@" + domain
}

// ConsentEvidenceLine builds the audit string §4's `consent_evidence` column
// holds, in one place so the three adapters write one format.
func ConsentEvidenceLine(what string, at time.Time) string {
	return fmt.Sprintf("%s %s", what, at.UTC().Format("2006-01-02"))
}
