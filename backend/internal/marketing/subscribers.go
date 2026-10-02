package marketing

// subscribers.go — card H5 (`subscribers-tab`). The mailing list: §5's
// GET /subscribers, GET /subscribers/{id}, POST /subscribers/{id}/resend, and
// the two manager-only import entry points the three adapters in
// internal/marketing/sources sit behind.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 TWO PROHIBITIONS, BOTH LOAD-BEARING
//
//  1. NOTHING HERE SENDS. POST /subscribers/{id}/resend returns 202, appends a
//     `resend_requested` event, and sends NOTHING — no SMS, no email, no
//     Resend API call, not behind a flag. Sending is Activity E's. The event
//     it appends is deliberately `resend_requested` and never `code_sent`:
//     `code_sent` is the SEND's event and is what the Stats funnel counts, so
//     writing it here would both lie to the operator and inflate a KR.
//     TestNothingInThisPackageSends parses this package's import graph to keep
//     the promise mechanical rather than editorial.
//
//  2. MASKING IS SERVER-SIDE. `phone_e164` and `email` never appear in a
//     response body — not in the list, not in the detail sheet, not in an
//     import summary. The wire carries `phone_last4` and `email_masked` only.
//     TestNoResponseCarriesAFullPhoneOrEmail asserts the absence on the
//     rendered JSON, and tests/marketing-subscribers.spec.js [SB-01] asserts
//     it again on what the BROWSER actually received.
// ═══════════════════════════════════════════════════════════════════════════

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yumyums/hq/internal/marketing/sources"
)

// maxCSVBytes caps a guest-CSV upload. A Toast guest book for a one-truck
// business is kilobytes; 4 MB is four orders of magnitude of headroom and still
// refuses a mis-drag of a video.
const maxCSVBytes = 4 << 20

// maxCSVRows caps the row count independently of the byte count.
const maxCSVRows = 50000

// SubsFixtureEnv points the web-form importer at a committed JSON fixture
// instead of the live MySQL. It exists so the ATTENDED first import can be
// rehearsed against the fixture on the operator's own box without touching the
// website database, and so the E2E gate can drive the real HTTP route with no
// network at all.
//
// 🛑 It is never set in production and never set by the night's gates except
// where a test sets it explicitly for one request.
const SubsFixtureEnv = "HQ_FF_FIXTURE_PATH"

// ── wire shapes (§5, field for field; UI reads these json tags verbatim) ──

// Consent states §5 enumerates. "stop" is an OPTED-OUT row, derived from §4's
// own `opted_out_at` column and nothing else.
const (
	ConsentSMS       = "sms"
	ConsentEmailOnly = "email_only"
	ConsentPending   = "pending"
	ConsentStop      = "stop"
)

// subscriberRowDTO is one row of §5's `rows:[…]`.
//
// 🛑 There is no `phone_e164` and no `email` field, on purpose. Adding one is a
// privacy regression, not a convenience.
type subscriberRowDTO struct {
	ID          string    `json:"id"`
	DisplayName *string   `json:"display_name"`
	PhoneLast4  *string   `json:"phone_last4"`
	EmailMasked *string   `json:"email_masked"`
	Source      string    `json:"source"`
	SourceShort *string   `json:"source_short"`
	CampaignName *string  `json:"campaign_name"`
	JoinedAt    time.Time `json:"joined_at"`
	Visits      int       `json:"visits"`
	Consent     string    `json:"consent"`
}

// subscriberListResponse is §5's `{total, sms_opt_in, joined_this_week, rows}`.
// The three counts are over the WHOLE list, not the filtered page: they are the
// header's standing reading of the mailing list, and a count that moved when
// the operator tapped a filter chip would be a different number every tap.
type subscriberListResponse struct {
	Total          int                `json:"total"`
	SMSOptIn       int                `json:"sms_opt_in"`
	JoinedThisWeek int                `json:"joined_this_week"`
	Rows           []subscriberRowDTO `json:"rows"`
}

// eventDTO is one timeline entry.
type eventDTO struct {
	ID   int64          `json:"id"`
	Kind string         `json:"kind"`
	At   time.Time      `json:"at"`
	Ref  map[string]any `json:"ref"`
}

// offerDTO is one row of the detail sheet's `offers_now[]`: a campaign live
// right now that this subscriber could walk in and redeem.
type offerDTO struct {
	CampaignID     string    `json:"campaign_id"`
	Name           string    `json:"name"`
	OfferText      string    `json:"offer_text"`
	FaceValueCents int       `json:"face_value_cents"`
	EndsAt         time.Time `json:"ends_at"`
}

// identityCodeDTO is the detail sheet's "identity-code status".
//
// Activity E owns the identity code itself (`identity-code-and-qr`), so there
// is no code to show yet and no table holding one. The honest status is
// therefore derived from the TIMELINE: a `code_sent` event means a code went
// out and when; no such event means it has not. `status` is "sent" or
// "not_sent", and `requested_at` carries the last `resend_requested` so the
// sheet can say "Resend requested · 2026-10-01" without implying a send
// happened.
type identityCodeDTO struct {
	Status      string     `json:"status"`
	SentAt      *time.Time `json:"sent_at"`
	RequestedAt *time.Time `json:"requested_at"`
}

// consentTrailDTO is the audit half of the sheet: the §4 columns, spelled out.
type consentTrailDTO struct {
	SMSConsent   bool       `json:"sms_consent"`
	EmailConsent bool       `json:"email_consent"`
	Evidence     *string    `json:"consent_evidence"`
	OptedOutAt   *time.Time `json:"opted_out_at"`
	State        string     `json:"state"`
}

// subscriberDetailDTO is §5's `detail + events[] + offers_now[]`.
type subscriberDetailDTO struct {
	subscriberRowDTO
	IdentityCode identityCodeDTO  `json:"identity_code"`
	Consent      consentTrailDTO  `json:"consent_trail"`
	Events       []eventDTO       `json:"events"`
	OffersNow    []offerDTO       `json:"offers_now"`
}

// importResultDTO is what an import route answers with. It carries COUNTS and
// no contact values — an import summary is a response body like any other and
// the masking contract applies to it too.
type importResultDTO struct {
	Source     string `json:"source"`
	Read       int    `json:"read"`
	Created    int    `json:"created"`
	Updated    int    `json:"updated"`
	Skipped    int    `json:"skipped"`
	ScansBound int    `json:"scans_bound"`
}

// ── GET /subscribers ──

// The §5 filter set. `all` is the default for anything unrecognized: the
// designed chip row is fixed, so an unexpected value is a client bug and a read
// that returns the list beats a read that returns a 400.
//
// Semantics (this card's call — the slate: "the filter set … is the night's"):
//
//	all        every subscriber
//	sms        sms_consent AND NOT opted out — who an SMS blast may reach
//	email_only email_consent AND NOT sms_consent AND NOT opted out — reachable,
//	           but not by text; the chip exists so a manager can see the people
//	           an SMS campaign will silently miss
//	opted_out  opted_out_at IS NOT NULL — the suppression list, which must be
//	           visible precisely because it must never be messaged
const (
	FilterAll       = "all"
	FilterSMS       = "sms"
	FilterEmailOnly = "email_only"
	FilterOptedOut  = "opted_out"
)

// ListSubscribersHandler is §5's GET /subscribers?q=&filter=&source=.
func ListSubscribersHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		filter := r.URL.Query().Get("filter")
		source := r.URL.Query().Get("source")

		where := []string{"TRUE"}
		args := []any{}
		// Placeholders are numbered explicitly rather than rewritten into a
		// template. A $N-substituting helper reads tidier and is wrong: once the
		// first clause has shifted $1 to $2 the clause contains two "$2"s and the
		// next substitution picks whichever comes first in the string.
		ph := func(v any) string {
			args = append(args, v)
			return fmt.Sprintf("$%d", len(args))
		}
		switch filter {
		case FilterSMS:
			where = append(where, "(s.sms_consent AND s.opted_out_at IS NULL)")
		case FilterEmailOnly:
			where = append(where, "(s.email_consent AND NOT s.sms_consent AND s.opted_out_at IS NULL)")
		case FilterOptedOut:
			where = append(where, "(s.opted_out_at IS NOT NULL)")
		}
		if containsString(subscriberSources, source) {
			where = append(where, "s.source = "+ph(source))
		}
		if q != "" {
			// q matches the display name, the email, the campaign name, the
			// short, or the phone BY SUFFIX. The suffix match is what makes the
			// masked cell searchable: a manager reads "•••• 4821" off the screen
			// and types 4821. The full number is never rendered, but it is still
			// the server's to search.
			like := ph("%" + q + "%")
			clause := "(s.display_name ILIKE " + like +
				" OR s.email ILIKE " + like +
				" OR c.name ILIKE " + like +
				" OR s.source_short ILIKE " + like
			if digits := onlyDigits(q); digits != "" {
				clause += " OR s.phone_e164 LIKE " + ph("%"+digits)
			}
			where = append(where, clause+")")
		}

		sql := `
			SELECT s.id::text, s.display_name, s.phone_e164, s.email, s.source,
			       s.source_short, c.name, s.joined_at, s.sms_consent,
			       s.email_consent, s.opted_out_at,
			       (SELECT count(*) FROM subscriber_events e
			          WHERE e.subscriber_id = s.id AND e.kind IN ('scanned','redeemed'))
			  FROM subscribers s
			  LEFT JOIN qr_codes k ON k.short = s.source_short
			  LEFT JOIN campaigns_admin c ON c.id = k.campaign_id
			 WHERE ` + strings.Join(where, " AND ") + `
			 ORDER BY s.joined_at DESC, s.id
			 LIMIT 500`
		rows, err := d.Pool.Query(r.Context(), sql, args...)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "list_failed")
			return
		}
		defer rows.Close()
		out := subscriberListResponse{Rows: []subscriberRowDTO{}}
		for rows.Next() {
			var (
				id, source                         string
				name, phone, email, short, camName *string
				joined                             time.Time
				sms, emailOK                       bool
				optedOut                           *time.Time
				visits                             int
			)
			if err := rows.Scan(&id, &name, &phone, &email, &source, &short, &camName,
				&joined, &sms, &emailOK, &optedOut, &visits); err != nil {
				writeError(w, http.StatusInternalServerError, "list_failed")
				return
			}
			out.Rows = append(out.Rows, subscriberRowDTO{
				ID: id, DisplayName: name,
				// 🛑 maskPtr is where the privacy contract is kept. `phone` and
				// `email` are local variables that END HERE.
				PhoneLast4:  maskPtr(sources.Last4(deref(phone))),
				EmailMasked: maskPtr(sources.MaskEmail(deref(email))),
				Source:      source, SourceShort: short, CampaignName: camName,
				JoinedAt: joined, Visits: visits,
				Consent: consentState(sms, emailOK, optedOut),
			})
		}
		if rows.Err() != nil {
			writeError(w, http.StatusInternalServerError, "list_failed")
			return
		}
		if err := d.Pool.QueryRow(r.Context(), `
			SELECT count(*),
			       count(*) FILTER (WHERE sms_consent AND opted_out_at IS NULL),
			       count(*) FILTER (WHERE joined_at >= now() - interval '7 days')
			  FROM subscribers`).Scan(&out.Total, &out.SMSOptIn, &out.JoinedThisWeek); err != nil {
			writeError(w, http.StatusInternalServerError, "list_failed")
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ── GET /subscribers/{id} ──

// GetSubscriberHandler is §5's detail route: the masked identity, the
// identity-code status, the consent trail, the timeline and the live offers.
func GetSubscriberHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		id := chi.URLParam(r, "id")
		var (
			name, phone, email, short, camName, evidence *string
			source                                       string
			joined                                       time.Time
			sms, emailOK                                 bool
			optedOut                                     *time.Time
			visits                                       int
		)
		err := d.Pool.QueryRow(r.Context(), `
			SELECT s.display_name, s.phone_e164, s.email, s.source, s.source_short,
			       c.name, s.consent_evidence, s.joined_at, s.sms_consent,
			       s.email_consent, s.opted_out_at,
			       (SELECT count(*) FROM subscriber_events e
			          WHERE e.subscriber_id = s.id AND e.kind IN ('scanned','redeemed'))
			  FROM subscribers s
			  LEFT JOIN qr_codes k ON k.short = s.source_short
			  LEFT JOIN campaigns_admin c ON c.id = k.campaign_id
			 WHERE s.id = $1`, id).
			Scan(&name, &phone, &email, &source, &short, &camName, &evidence,
				&joined, &sms, &emailOK, &optedOut, &visits)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "detail_failed")
			return
		}

		events, ic, err := subscriberTimeline(r.Context(), d, id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "detail_failed")
			return
		}
		offers, err := offersNow(r.Context(), d)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "detail_failed")
			return
		}
		writeJSON(w, http.StatusOK, subscriberDetailDTO{
			subscriberRowDTO: subscriberRowDTO{
				ID: id, DisplayName: name,
				PhoneLast4:  maskPtr(sources.Last4(deref(phone))),
				EmailMasked: maskPtr(sources.MaskEmail(deref(email))),
				Source:      source, SourceShort: short, CampaignName: camName,
				JoinedAt: joined, Visits: visits,
				Consent: consentState(sms, emailOK, optedOut),
			},
			IdentityCode: ic,
			Consent: consentTrailDTO{
				SMSConsent: sms, EmailConsent: emailOK, Evidence: evidence,
				OptedOutAt: optedOut, State: consentState(sms, emailOK, optedOut),
			},
			Events:    events,
			OffersNow: offers,
		})
	}
}

// subscriberTimeline reads the events newest-first and derives the
// identity-code status from them in one pass.
func subscriberTimeline(ctx context.Context, d Deps, id string) ([]eventDTO, identityCodeDTO, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id, kind, at, ref FROM subscriber_events
		 WHERE subscriber_id = $1 ORDER BY at DESC, id DESC LIMIT 200`, id)
	if err != nil {
		return nil, identityCodeDTO{}, err
	}
	defer rows.Close()
	out := []eventDTO{}
	ic := identityCodeDTO{Status: "not_sent"}
	for rows.Next() {
		var e eventDTO
		if err := rows.Scan(&e.ID, &e.Kind, &e.At, &e.Ref); err != nil {
			return nil, identityCodeDTO{}, err
		}
		// Rows arrive newest-first, so the FIRST of each kind is the latest.
		switch e.Kind {
		case "code_sent":
			if ic.SentAt == nil {
				at := e.At
				ic.SentAt = &at
				ic.Status = "sent"
			}
		case "resend_requested":
			if ic.RequestedAt == nil {
				at := e.At
				ic.RequestedAt = &at
			}
		}
		out = append(out, e)
	}
	return out, ic, rows.Err()
}

// offersNow is every campaign a walk-in could redeem right now. It is not
// per-subscriber because nothing in §4 scopes an offer to a person — the sheet
// answers "what can I honor if they show up", which is a property of the
// campaign set and of the clock.
func offersNow(ctx context.Context, d Deps) ([]offerDTO, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT id::text, name, offer_text, face_value_cents, ends_at
		  FROM campaigns_admin
		 WHERE status = 'live' AND starts_at <= now() AND ends_at >= now()
		 ORDER BY ends_at ASC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []offerDTO{}
	for rows.Next() {
		var o offerDTO
		if err := rows.Scan(&o.CampaignID, &o.Name, &o.OfferText, &o.FaceValueCents, &o.EndsAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ── POST /subscribers/{id}/resend ──

// ResendQRHandler is §5's one write: 202, an appended `resend_requested`
// event, and NO SEND.
//
// 🛑 Read the whole function. There is no branch, no flag and no env var that
// makes it send anything; the only side effect is the INSERT below. The event
// kind is `resend_requested` and NOT `code_sent` precisely so the Stats funnel
// (which counts `code_sent`) cannot be inflated by a request nobody fulfilled.
// Activity E will append `code_sent` when a message actually leaves.
func ResendQRHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := requireManager(w, r)
		if user == nil {
			return
		}
		id := chi.URLParam(r, "id")
		var exists bool
		err := d.Pool.QueryRow(r.Context(),
			`SELECT true FROM subscribers WHERE id = $1`, id).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "resend_failed")
			return
		}
		if _, err := d.Pool.Exec(r.Context(), `
			INSERT INTO subscriber_events (subscriber_id, kind, ref)
			VALUES ($1, 'resend_requested', jsonb_build_object('requested_by', $2::text, 'sent', false))`,
			id, user.ID); err != nil {
			writeError(w, http.StatusInternalServerError, "resend_failed")
			return
		}
		// 202 Accepted, and the body says so in words: the request is
		// recorded, the send is Activity E's. A 200 here would read as "done".
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "recorded",
			"sent":   false,
			"note":   "resend_requested recorded; sending is not implemented yet",
		})
	}
}

// ── the import entry points ──

// ImportToastGuestsHandler takes a Toast guest CSV on the request body
// (text/csv or multipart `file`) and lands it through the one upsert.
func ImportToastGuestsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		body, err := csvBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_upload")
			return
		}
		imp, err := sources.NewToastGuests(strings.NewReader(body), maxCSVRows)
		if err != nil {
			if errors.Is(err, sources.ErrNoGuestColumns) {
				writeError(w, http.StatusBadRequest, "no_guest_columns")
				return
			}
			writeError(w, http.StatusBadRequest, "bad_csv")
			return
		}
		runImport(w, r, d, imp)
	}
}

// ImportWebFormHandler runs the Fluent Forms reader.
//
// 🛑 ATTENDED ONLY. With FF_DB_* unset — which is their state in every gate,
// every worktree and prod — this answers 503 {"error":"ff_not_configured"} and
// opens no connection. The first live import is an operator act, by design
// (see internal/marketing/sources/testdata/README.md).
func ImportWebFormHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		var imp *sources.FluentForms
		var err error
		if p := strings.TrimSpace(os.Getenv(SubsFixtureEnv)); p != "" {
			var raw []byte
			if raw, err = os.ReadFile(p); err == nil {
				imp, err = sources.NewFluentFormsFromJSON(raw)
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "ff_fixture_unreadable")
				return
			}
		} else {
			imp, err = sources.NewFluentForms()
			if errors.Is(err, sources.ErrFluentFormsNotConfigured) {
				writeError(w, http.StatusServiceUnavailable, "ff_not_configured")
				return
			}
			if err != nil {
				writeError(w, http.StatusInternalServerError, "ff_open_failed")
				return
			}
		}
		runImport(w, r, d, imp)
	}
}

func runImport(w http.ResponseWriter, r *http.Request, d Deps, imp sources.Importer) {
	res, err := ImportSubscribers(r.Context(), d.Pool, imp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "import_failed")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// csvBody reads either a raw body or a multipart `file` part.
func csvBody(r *http.Request) (string, error) {
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxCSVBytes); err != nil {
			return "", err
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			return "", err
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, maxCSVBytes))
		return string(b), err
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, maxCSVBytes))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(string(b)) == "" {
		return "", fmt.Errorf("empty body")
	}
	return string(b), nil
}

// ── the ONE upsert every adapter lands through ──

// ImportSubscribers reads an Importer and lands its candidates.
//
// # Idempotency, in the order it is attempted
//
//  1. `(source, external_ref)` — §4's unique key, and the only handle a source
//     with stable ids needs. Re-running the same import updates in place.
//  2. `phone_e164` — §4's other unique, and the key spike 02 proved closed.
//     This is what collapses "17735554821" onto the row "(773) 555-4821"
//     already created, across sources: a Toast guest who later fills in the web
//     form is ONE person, not two rows the operator has to reconcile by eye.
//  3. otherwise INSERT.
//
// An UPDATE is deliberately a MERGE and never a replace: it fills NULLs, ORs
// the consent flags up, and sets `source_short` only when it is currently NULL
// — first-touch (decision 189) means the FIRST code wins, so a later import
// carrying a different short must not overwrite it.
//
// `signed_up` is appended on INSERT only. An idempotent re-run that appended an
// event every time would make the timeline a log of imports instead of a
// history of the person.
func ImportSubscribers(ctx context.Context, pool interface {
	Begin(context.Context) (pgx.Tx, error)
}, imp sources.Importer) (importResultDTO, error) {
	cands, err := imp.Read()
	if err != nil {
		return importResultDTO{}, err
	}
	res := importResultDTO{Source: imp.Source(), Read: len(cands)}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	for _, c := range cands {
		if c.PhoneE164 == "" && c.Email == "" {
			res.Skipped++
			continue
		}
		// An unknown short would violate the FK onto qr_codes(short) and cost
		// the whole import. A printed code that no longer exists is the
		// customer's reality, not an error: keep the person, drop the
		// attribution, and say so in the evidence line.
		short := c.SourceShort
		if short != "" {
			var ok bool
			if err := tx.QueryRow(ctx, `SELECT true FROM qr_codes WHERE short = $1`, short).Scan(&ok); err != nil {
				if !errors.Is(err, pgx.ErrNoRows) {
					return res, err
				}
				short = ""
			}
		}

		id, found, err := findSubscriber(ctx, tx, c)
		if err != nil {
			return res, err
		}
		if found {
			if err := mergeSubscriber(ctx, tx, id, c, short); err != nil {
				return res, err
			}
			res.Updated++
		} else {
			id, err = insertSubscriber(ctx, tx, c, short)
			if err != nil {
				return res, err
			}
			res.Created++
		}
		if short != "" {
			n, err := sources.BackfillScanSubscriber(ctx, tx, short, id, c.JoinedAt)
			if err != nil {
				return res, err
			}
			res.ScansBound += n
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	return res, nil
}

func findSubscriber(ctx context.Context, tx pgx.Tx, c sources.Candidate) (string, bool, error) {
	if c.ExternalRef != "" {
		var id string
		err := tx.QueryRow(ctx,
			`SELECT id::text FROM subscribers WHERE source = $1 AND external_ref = $2`,
			c.Source, c.ExternalRef).Scan(&id)
		if err == nil {
			return id, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, err
		}
	}
	if c.PhoneE164 != "" {
		var id string
		err := tx.QueryRow(ctx,
			`SELECT id::text FROM subscribers WHERE phone_e164 = $1`, c.PhoneE164).Scan(&id)
		if err == nil {
			return id, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return "", false, err
		}
	}
	return "", false, nil
}

func insertSubscriber(ctx context.Context, tx pgx.Tx, c sources.Candidate, short string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		INSERT INTO subscribers (display_name, phone_e164, email, source, source_short,
		                         sms_consent, email_consent, consent_evidence,
		                         opted_out_at, joined_at, external_ref)
		VALUES (NULLIF($1,''), NULLIF($2,''), NULLIF($3,''), $4, NULLIF($5,''),
		        $6, $7, NULLIF($8,''),
		        CASE WHEN $9 THEN now() ELSE NULL END, $10, NULLIF($11,''))
		RETURNING id::text`,
		c.DisplayName, c.PhoneE164, c.Email, c.Source, short,
		c.SMSConsent, c.EmailConsent, c.ConsentEvidence, c.OptedOut,
		c.JoinedAt, c.ExternalRef).Scan(&id)
	if err != nil {
		// A concurrent import, or a phone that findSubscriber could not see
		// because the candidate carried only an email on the first pass. Fall
		// back to the merge rather than failing the whole file.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && c.PhoneE164 != "" {
			if e2 := tx.QueryRow(ctx,
				`SELECT id::text FROM subscribers WHERE phone_e164 = $1`, c.PhoneE164).Scan(&id); e2 == nil {
				return id, mergeSubscriber(ctx, tx, id, c, short)
			}
		}
		return "", fmt.Errorf("insert subscriber: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO subscriber_events (subscriber_id, kind, at, ref)
		VALUES ($1, 'signed_up', $2, jsonb_build_object('source', $3::text))`,
		id, c.JoinedAt, c.Source); err != nil {
		return "", fmt.Errorf("insert signed_up: %w", err)
	}
	if c.OptedOut {
		if _, err := tx.Exec(ctx, `
			INSERT INTO subscriber_events (subscriber_id, kind, ref)
			VALUES ($1, 'opted_out', jsonb_build_object('source', $2::text))`,
			id, c.Source); err != nil {
			return "", fmt.Errorf("insert opted_out: %w", err)
		}
	}
	return id, nil
}

// mergeSubscriber fills gaps and never clobbers. See ImportSubscribers' doc for
// why each column behaves the way it does.
func mergeSubscriber(ctx context.Context, tx pgx.Tx, id string, c sources.Candidate, short string) error {
	_, err := tx.Exec(ctx, `
		UPDATE subscribers SET
		  display_name     = COALESCE(display_name, NULLIF($2,'')),
		  phone_e164       = COALESCE(phone_e164,   NULLIF($3,'')),
		  email            = COALESCE(email,        NULLIF($4,'')),
		  source_short     = COALESCE(source_short, NULLIF($5,'')),
		  sms_consent      = sms_consent   OR $6,
		  email_consent    = email_consent OR $7,
		  -- EARLIEST evidence wins, like joined_at below: the trail records the
		  -- consent originally given, and successive actions live in
		  -- subscriber_events where they belong. A later import must not
		  -- overwrite the line that documents the first grant.
		  consent_evidence = COALESCE(consent_evidence, NULLIF($8,'')),
		  opted_out_at     = CASE WHEN $9 THEN COALESCE(opted_out_at, now()) ELSE opted_out_at END,
		  joined_at        = LEAST(joined_at, $10),
		  external_ref     = COALESCE(external_ref, NULLIF($11,''))
		WHERE id = $1`,
		id, c.DisplayName, c.PhoneE164, c.Email, short,
		c.SMSConsent, c.EmailConsent, c.ConsentEvidence, c.OptedOut,
		c.JoinedAt, c.ExternalRef)
	if err != nil {
		return fmt.Errorf("merge subscriber: %w", err)
	}
	return nil
}

// ── small helpers ──

// subscriberSources is §4's enum, used to validate the `source=` filter.
var subscriberSources = []string{
	sources.SourceWebForm, sources.SourceToastImport,
	sources.SourceSMSKeyword, sources.SourceQR,
}

// consentState derives §5's four-value consent field from the §4 columns.
// opted_out_at WINS over both flags: a person who said STOP is "stop" even if
// a stale checkbox is still true, and that precedence is the whole point of
// surfacing the state at all.
func consentState(sms, email bool, optedOut *time.Time) string {
	switch {
	case optedOut != nil:
		return ConsentStop
	case sms:
		return ConsentSMS
	case email:
		return ConsentEmailOnly
	default:
		return ConsentPending
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// maskPtr turns "" into a JSON null, so the UI can tell "no phone on file"
// from "a phone whose mask is empty" — there is no second case, and null is
// the honest wire value for the first.
func maskPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
