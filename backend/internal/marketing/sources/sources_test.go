package sources

// sources_test.go — hermetic. No Postgres, no MySQL, no network: every
// assertion here runs off the committed fixture and the enumerated phone set,
// which is the whole point of putting the adapters behind Read().

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestFluentFormsMapsTheRealKeys pins the goal ledger's binding correction.
// Spike 01 run 1 was RED because the premise (`names, email, phone, source` —
// the website script's defaults) was false. The real keys are `names` (an
// object), `email`, `input_text` (the phone) and `checkbox` (an array), with
// `source` often absent.
func TestFluentFormsMapsTheRealKeys(t *testing.T) {
	raw, err := os.ReadFile("testdata/fluentforms_submissions.json")
	if err != nil {
		t.Fatalf("read committed fixture: %v", err)
	}
	ff, err := NewFluentFormsFromJSON(raw)
	if err != nil {
		t.Fatalf("NewFluentFormsFromJSON: %v", err)
	}
	if ff.Source() != SourceWebForm {
		t.Fatalf("Source() = %q, want %q", ff.Source(), SourceWebForm)
	}
	cands, err := ff.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(cands) != 5 {
		t.Fatalf("read %d candidates, want 5", len(cands))
	}
	by := map[string]Candidate{}
	for _, c := range cands {
		by[c.ExternalRef] = c
	}

	c := by["1184"]
	if c.DisplayName != "Dana" {
		t.Errorf("1184 display_name = %q, want Dana (names.first_name)", c.DisplayName)
	}
	if c.PhoneE164 != "+17735554821" {
		t.Errorf("1184 phone = %q, want +17735554821 (input_text, normalized)", c.PhoneE164)
	}
	if !c.SMSConsent || !c.EmailConsent {
		t.Errorf("1184 consent = %v/%v, want true/true (checkbox ['Phone','Email'])", c.SMSConsent, c.EmailConsent)
	}
	if c.SourceShort != "" {
		t.Errorf("1184 source_short = %q, want empty — `source` is absent on this row", c.SourceShort)
	}
	if !strings.Contains(c.ConsentEvidence, "[Phone,Email]") {
		t.Errorf("1184 consent_evidence = %q, want it to name the ticked boxes", c.ConsentEvidence)
	}
	if got := c.JoinedAt.Format("2006-01-02"); got != "2026-09-28" {
		t.Errorf("1184 joined_at = %s, want 2026-09-28 (created_at)", got)
	}

	if c := by["1183"]; c.SMSConsent || !c.EmailConsent {
		t.Errorf("1183 consent = %v/%v, want false/true (checkbox ['Email'])", c.SMSConsent, c.EmailConsent)
	}
	if c := by["1182"]; c.SourceShort != "K7MNPQ" {
		t.Errorf("1182 source_short = %q, want K7MNPQ — `source` IS present on this row", c.SourceShort)
	}
	if c := by["1181"]; c.PhoneE164 != "" {
		t.Errorf("1181 phone = %q, want empty (555-4821 is 7 digits)", c.PhoneE164)
	} else if !strings.Contains(c.ConsentEvidence, "none") {
		t.Errorf("1181 consent_evidence = %q, want it to say the submitter ticked nothing", c.ConsentEvidence)
	}
	if c := by["1180"]; c.PhoneE164 != by["1184"].PhoneE164 {
		t.Errorf("1180 phone = %q, want it to normalize onto 1184's %q", c.PhoneE164, by["1184"].PhoneE164)
	}
}

// TestFluentFormsRefusesWithoutCredentials: the live reader is env-gated and
// the night never sets the env, so this is the branch every gate takes.
func TestFluentFormsRefusesWithoutCredentials(t *testing.T) {
	for _, k := range []string{EnvFFHost, EnvFFUser, EnvFFPassword, EnvFFName, EnvFFPrefix} {
		t.Setenv(k, "")
	}
	if _, err := NewFluentForms(); err == nil {
		t.Fatal("NewFluentForms succeeded with empty credentials — it must refuse")
	}
}

func TestToastGuestsHeaderAliasesAndRefusal(t *testing.T) {
	if _, err := NewToastGuests(strings.NewReader("Item,Qty\nWings,2\n"), 100); err == nil {
		t.Error("a CSV with no phone or email column must be refused")
	}
	tg, err := NewToastGuests(strings.NewReader(
		"Guest Phone,Guest Name,Marketing Opt In\n(773) 555-0199,Ada,Yes\n,,\n"), 100)
	if err != nil {
		t.Fatalf("NewToastGuests: %v", err)
	}
	cands, _ := tg.Read()
	if len(cands) != 1 {
		t.Fatalf("read %d candidates, want 1 (the blank line is skipped)", len(cands))
	}
	if cands[0].PhoneE164 != "+17735550199" || cands[0].DisplayName != "Ada" || !cands[0].EmailConsent {
		t.Errorf("mapped %+v, want phone +17735550199 / Ada / email consent", cands[0])
	}
	if tg.Source() != SourceToastImport {
		t.Errorf("Source() = %q, want %q", tg.Source(), SourceToastImport)
	}
}

func TestQRSignupRequiresAShortAndAWayToReachThem(t *testing.T) {
	if _, err := NewQRSignup("", "A", "7735550100", "", true, false, timeZero(), ""); err == nil {
		t.Error("a QR signup with no short must be refused")
	}
	if _, err := NewQRSignup("K7MNPQ", "A", "555-0100", "", true, false, timeZero(), ""); err == nil {
		t.Error("a QR signup with no usable phone and no email must be refused")
	}
	q, err := NewQRSignup("K7MNPQ", "A", "7735550100", "", true, false, timeZero(), "x")
	if err != nil {
		t.Fatalf("NewQRSignup: %v", err)
	}
	if q.Source() != SourceQR {
		t.Errorf("Source() = %q, want %q", q.Source(), SourceQR)
	}
	c, _ := q.Read()
	if !strings.Contains(c[0].ConsentEvidence, "q=K7MNPQ") {
		t.Errorf("consent_evidence = %q, want it to name the landing's q= param", c[0].ConsentEvidence)
	}
}

func timeZero() time.Time { return time.Time{} }
