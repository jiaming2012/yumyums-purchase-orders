package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// QRSignup is the third adapter: a signup that arrived through a campaign QR
// landing, where the only first-touch signal is the `q=<short>` the landing
// forwarded onto the website URL (spike 01's correction — the live form carries
// no `source` field, so for a web signup `q=` is the ONLY attribution there is).
//
// It is an Importer like the other two, but its input is one signup rather than
// a batch, because that is how it arrives: one person, one landing, one short.
type QRSignup struct {
	rows []Candidate
}

// Source implements Importer. Note the enum value is `qr`, NOT `web_form`: a
// signup that came through a campaign code is attributable, and keeping it as
// its own source is what makes the list's `source=qr` filter mean anything.
func (q *QRSignup) Source() string { return SourceQR }

// Read implements Importer.
func (q *QRSignup) Read() ([]Candidate, error) { return q.rows, nil }

// NewQRSignup builds the one-row importer. `short` is required — without it
// this is not a QR signup and belongs to another adapter.
func NewQRSignup(short, displayName, rawPhone, email string, sms, emailOK bool, at time.Time, externalRef string) (*QRSignup, error) {
	if short == "" {
		return nil, fmt.Errorf("qr signup: short is required")
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	c := Candidate{
		DisplayName:     displayName,
		PhoneE164:       NormalizeE164(rawPhone),
		Email:           email,
		Source:          SourceQR,
		SourceShort:     short,
		SMSConsent:      sms,
		EmailConsent:    emailOK,
		ConsentEvidence: ConsentEvidenceLine("qr landing signup q="+short, at),
		JoinedAt:        at,
		ExternalRef:     externalRef,
	}
	if c.PhoneE164 == "" && c.Email == "" {
		return nil, fmt.Errorf("qr signup: needs a phone or an email")
	}
	return &QRSignup{rows: []Candidate{c}}, nil
}

// BackfillScanSubscriber points the FIRST-TOUCH qr_scans row at a subscriber.
//
// Decision 189 is first-touch: a subscriber's campaign is the first code they
// scanned. So the back-fill claims the EARLIEST unclaimed scan of that short at
// or before the signup — not the latest, and not every scan of the short, which
// would attribute strangers' scans to whoever signed up next.
//
// Returns the number of qr_scans rows claimed (0 or 1). A signup with no
// matching scan is normal: the landing dedupes scans on (short, ip_hash) for
// ten minutes and a customer can type the URL.
func BackfillScanSubscriber(ctx context.Context, tx pgx.Tx, short, subscriberID string, joinedAt time.Time) (int, error) {
	tag, err := tx.Exec(ctx, `
		UPDATE qr_scans SET subscriber_id = $2
		WHERE id = (
			SELECT id FROM qr_scans
			WHERE short = $1 AND subscriber_id IS NULL AND scanned_at <= $3
			ORDER BY scanned_at ASC, id ASC
			LIMIT 1
		)`, short, subscriberID, joinedAt)
	if err != nil {
		return 0, fmt.Errorf("backfill qr_scans.subscriber_id: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
