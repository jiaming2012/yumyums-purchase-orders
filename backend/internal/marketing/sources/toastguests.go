package sources

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// ErrNoGuestColumns is the refusal when the uploaded CSV's header carries
// neither a phone nor an email column. A wrong file (an ItemSelectionDetails
// export, say) must fail loud at the header rather than import zero rows and
// report success.
var ErrNoGuestColumns = errors.New("toast guest CSV: header has no phone or email column")

// ToastGuests maps a Toast guest-book CSV upload onto candidates.
//
// # Why the header is matched loosely
//
// Toast's guest export is a report, and report headers move: "Phone", "Phone
// Number", "Guest Phone" have all been seen in the wild and HQ does not control
// the generator. So the reader matches on a NORMALIZED header (lowercased,
// non-alphanumerics stripped) against a small alias list, and refuses loudly
// when it finds no contact column at all. A column it does not recognize is
// ignored, never guessed at.
//
// # Consent
//
// Toast's export carries the marketing opt-in flags the guest actually gave at
// the POS. Those map onto §4's OWN columns — `sms_consent`, `email_consent`,
// `opted_out_at` — and nothing else: this card invents no consent semantics and
// adds no column (what counts as consent, and how STOP is represented beyond
// §4, is the operator's compliance call and is PARKED). A file with no consent
// column imports every row with both flags FALSE, i.e. `consent:"pending"`,
// which is the honest reading of "we have their number and no permission".
type ToastGuests struct {
	rows []Candidate
}

// Source implements Importer.
func (t *ToastGuests) Source() string { return SourceToastImport }

// Read implements Importer.
func (t *ToastGuests) Read() ([]Candidate, error) { return t.rows, nil }

// Header aliases, normalized. Order inside a group does not matter; the first
// column whose normalized header is in the group wins.
var (
	hdrName    = []string{"name", "guestname", "firstname", "fullname", "displayname"}
	hdrPhone   = []string{"phone", "phonenumber", "guestphone", "mobile", "mobilephone"}
	hdrEmail   = []string{"email", "emailaddress", "guestemail"}
	// 🛑 NO BARE "id". `id` is a generic column name: one export's `id` is a
	// stable guest handle, a later export's is a plain ROW NUMBER. Both land on
	// external_ref "1", the upsert matches (toast_import,'1'), and the merge
	// ORs a stranger's opt-in onto whoever imported first — a CONSENT bug, not a
	// data-hygiene one (G6 finding F2; TestToastGuestIDAliasDoesNotMergeTwoPeople
	// reproduces it as Bob's "Yes" landing on Alice's "No"). Only names that
	// unambiguously denote a GUEST identifier are accepted; an export without
	// one simply has no external_ref and dedupes on the phone instead, which is
	// the key spike 02 proved closed.
	hdrGuestID = []string{"guestid", "guestguid", "customerid", "customerguid", "guestidentifier"}
	hdrSMS     = []string{"smsconsent", "smsoptin", "textoptin", "marketingsmsoptin"}
	hdrEmailOK = []string{"emailconsent", "emailoptin", "marketingoptin", "marketingemailoptin"}
	hdrOptOut  = []string{"optedout", "optout", "unsubscribed", "donotcontact"}
	hdrJoined  = []string{"createddate", "created", "firstvisit", "joined", "signupdate"}
)

// NewToastGuests parses an uploaded CSV. maxRows caps the import so an upload
// cannot be used to exhaust the box.
func NewToastGuests(r io.Reader, maxRows int) (*ToastGuests, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	header, err := cr.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("toast guest CSV: file is empty")
	}
	if err != nil {
		return nil, fmt.Errorf("toast guest CSV header: %w", err)
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[normalizeHeader(h)] = i
	}
	col := func(aliases []string) int {
		for _, a := range aliases {
			if i, ok := idx[a]; ok {
				return i
			}
		}
		return -1
	}
	cName, cPhone, cEmail := col(hdrName), col(hdrPhone), col(hdrEmail)
	cID, cSMS, cEmailOK := col(hdrGuestID), col(hdrSMS), col(hdrEmailOK)
	cOptOut, cJoined := col(hdrOptOut), col(hdrJoined)
	if cPhone < 0 && cEmail < 0 {
		return nil, fmt.Errorf("%w (saw %v)", ErrNoGuestColumns, header)
	}

	out := []Candidate{}
	for n := 0; ; n++ {
		rec, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("toast guest CSV row %d: %w", n+2, err)
		}
		if n >= maxRows {
			return nil, fmt.Errorf("toast guest CSV: more than %d rows", maxRows)
		}
		get := func(i int) string {
			if i < 0 || i >= len(rec) {
				return ""
			}
			return strings.TrimSpace(rec[i])
		}
		phone := NormalizeE164(get(cPhone))
		email := get(cEmail)
		if phone == "" && email == "" {
			// A blank trailing line, or a guest row with no way to reach them.
			continue
		}
		joined := parseCreatedAt(get(cJoined))
		c := Candidate{
			DisplayName:  get(cName),
			PhoneE164:    phone,
			Email:        email,
			Source:       SourceToastImport,
			SMSConsent:   truthy(get(cSMS)),
			EmailConsent: truthy(get(cEmailOK)),
			OptedOut:     truthy(get(cOptOut)),
			JoinedAt:     joined,
			ExternalRef:  get(cID),
		}
		c.ConsentEvidence = ConsentEvidenceLine("toast guest import", time.Now())
		out = append(out, c)
	}
	return &ToastGuests{rows: out}, nil
}

func normalizeHeader(h string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(h) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// truthy reads the spellings a report generator uses for a boolean. Anything
// unrecognized is FALSE — the safe direction for a consent flag.
func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "y", "yes", "true", "t", "opted in", "opted_in", "optin":
		return true
	}
	return false
}
