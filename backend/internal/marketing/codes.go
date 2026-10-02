package marketing

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// shortCodeAlphabet is decision 189's 32 characters. 0, 1, I and O are absent
// deliberately: a human reads these off a printed sign and types them into a
// phone, and those four are the pairs that get misread.
//
// 32 divides 256 exactly, so a byte taken modulo 32 is already uniform. The
// rejection loop in newShortCode is kept anyway so that shortening this
// alphabet later cannot silently bias the draw.
const shortCodeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// shortCodeLen is fixed at 6 by migration 0083's CHECK and by the version-3 QR
// ceiling: a 7th character pushes the 35-char payload to QR version 4 (33
// modules), which stops scanning from a truck sign (spike 01).
const shortCodeLen = 6

// maxShortAttempts bounds the collision retry. At 32^6 ≈ 1.07e9 keys a
// collision is already improbable; 8 attempts is there for the concurrent-mint
// race, not for exhaustion.
const maxShortAttempts = 8

// channels is migration 0083's CHECK, mirrored in Go so a bad channel is a 400
// with a useful message instead of a 500 from a constraint violation.
var channels = []string{
	"truck_sign", "flyer", "table_tent", "menu_board",
	"instagram", "google_ads", "receipt", "sms", "other",
}

// landings is the campaigns_admin.landing / qr_codes.landing CHECK.
var landings = []string{"signup", "menu", "offer", "directions"}

// newShortCode draws shortCodeLen characters uniformly from shortCodeAlphabet.
func newShortCode() (string, error) {
	const n = len(shortCodeAlphabet)
	// The rejection boundary. With n=32 it lands exactly on 256, so no byte is
	// ever rejected — the loop below is dead code TODAY and kept alive so that
	// shrinking the alphabet later cannot silently bias the draw.
	limit := 256 - (256 % n)
	out := make([]byte, 0, shortCodeLen)
	buf := make([]byte, shortCodeLen)
	for len(out) < shortCodeLen {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("short code entropy: %w", err)
		}
		for _, b := range buf {
			if int(b) >= limit {
				continue // reject, keep the draw uniform
			}
			out = append(out, shortCodeAlphabet[int(b)%n])
			if len(out) == shortCodeLen {
				break
			}
		}
	}
	return string(out), nil
}

// slugify renders a name as the URL-safe slug campaigns_admin.slug holds and
// utm_campaign / utm_content carry. ASCII letters and digits survive, runs of
// everything else collapse to a single hyphen, and the ends are trimmed.
//
// Non-ASCII is DROPPED rather than transliterated: "Café" becomes "caf". That
// is deliberate — a slug is a key in a URL, and a half-correct transliteration
// table is worse than a short slug, because uniqueness is handled separately
// by uniqueSlug.
func slugify(s string) string {
	var b strings.Builder
	lastHyphen := true // suppresses a leading hyphen
	for _, r := range strings.ToLower(s) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			lastHyphen = false
		default:
			if !lastHyphen {
				b.WriteByte('-')
				lastHyphen = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// uniqueSlug finds a free slug for base, appending -2, -3, … on collision.
// campaigns_admin.slug is UNIQUE, so two "Wing Wednesday" campaigns must not
// race into the same key.
func uniqueSlug(ctx context.Context, q queryRower, base string) (string, error) {
	if base == "" {
		base = "campaign"
	}
	for i := 1; i <= 50; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		var taken bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM campaigns_admin WHERE slug = $1)`, candidate).Scan(&taken); err != nil {
			return "", fmt.Errorf("slug probe: %w", err)
		}
		if !taken {
			return candidate, nil
		}
	}
	// 50 campaigns with the same name is not a shape to paper over with a uuid.
	return "", fmt.Errorf("no free slug for %q after 50 attempts", base)
}

// queryRower is the one method uniqueSlug needs, so it works against both a
// pool and a transaction.
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (SQLSTATE 23505), optionally on a named constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

// channelInput is one entry of POST /campaigns' `channels` array and the whole
// body of POST /campaigns/{id}/codes.
type channelInput struct {
	Channel      string  `json:"channel"`
	ChannelLabel *string `json:"channel_label"`
	Placement    *string `json:"placement"`
	Variant      *string `json:"variant"`
	Landing      *string `json:"landing"`
}

// validate returns "" when the input is usable, else the error slug to 400 with.
func (c channelInput) validate() string {
	if !containsString(channels, c.Channel) {
		return "bad_channel"
	}
	// §4: channel_label is "required when channel='other'". An unlabelled
	// 'other' is unreportable — it is the row that makes "best channel"
	// uncomputable, which is the whole point of the enum (decision 189).
	if c.Channel == "other" && (c.ChannelLabel == nil || strings.TrimSpace(*c.ChannelLabel) == "") {
		return "channel_label_required"
	}
	if c.Landing != nil && !containsString(landings, *c.Landing) {
		return "bad_landing"
	}
	return ""
}

func containsString(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

// mintCode inserts one qr_codes row, retrying the short on collision.
//
// Each attempt runs in its own SAVEPOINT (pgx models a nested Begin as one), so
// a unique violation does not poison the OUTER transaction that is minting the
// campaign and its other codes. Without that, one collision would abort the
// whole create.
func mintCode(ctx context.Context, tx pgx.Tx, campaignID, createdBy string, in channelInput, itemID *string) (codeDTO, error) {
	for attempt := 0; attempt < maxShortAttempts; attempt++ {
		short, err := newShortCode()
		if err != nil {
			return codeDTO{}, err
		}
		sp, err := tx.Begin(ctx)
		if err != nil {
			return codeDTO{}, fmt.Errorf("savepoint: %w", err)
		}
		var out codeDTO
		err = sp.QueryRow(ctx, `
			INSERT INTO qr_codes
			  (short, campaign_id, channel, channel_label, item_id, placement, variant, landing, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id::text, short, campaign_id::text, channel, channel_label,
			          item_id::text, placement, variant, landing, active, v`,
			short, campaignID, in.Channel, in.ChannelLabel, itemID,
			in.Placement, in.Variant, in.Landing, createdBy,
		).Scan(&out.ID, &out.Short, &out.CampaignID, &out.Channel, &out.ChannelLabel,
			&out.ItemID, &out.Placement, &out.Variant, &out.Landing, &out.Active, &out.V)
		if err != nil {
			_ = sp.Rollback(ctx)
			if isUniqueViolation(err, "qr_codes_short_key") {
				slog.Warn("marketing: short code collision, retrying", "short", short, "attempt", attempt+1)
				continue
			}
			return codeDTO{}, fmt.Errorf("insert qr_code: %w", err)
		}
		if err := sp.Commit(ctx); err != nil {
			return codeDTO{}, fmt.Errorf("release savepoint: %w", err)
		}
		return out, nil
	}
	return codeDTO{}, fmt.Errorf("could not mint a unique short code in %d attempts", maxShortAttempts)
}

// decorate fills the derived URL fields. Kept separate from the INSERT so the
// list, detail and create routes all render a code the same way.
func decorate(d Deps, c *codeDTO) {
	c.PayloadURL = d.QRBaseURL + "/q/" + c.Short
	c.PNGURL = "/api/v1/marketing/codes/" + c.ID + ".png"
}

// ── POST /campaigns/{id}/codes ──

// CreateCodeHandler adds one channel to an existing campaign — the designed
// "add a channel" affordance. The new code inherits the campaign's item unless
// the body names one.
func CreateCodeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := requireManager(w, r)
		if user == nil {
			return
		}
		campaignID := chi.URLParam(r, "id")
		var in channelInput
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_payload")
			return
		}
		if bad := in.validate(); bad != "" {
			writeError(w, http.StatusBadRequest, bad)
			return
		}

		ctx := r.Context()
		var itemID *string
		err := d.Pool.QueryRow(ctx,
			`SELECT item_id::text FROM campaigns_admin WHERE id = $1`, campaignID).Scan(&itemID)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "campaign_not_found")
			return
		}
		if err != nil {
			if isBadUUID(err) {
				writeError(w, http.StatusNotFound, "campaign_not_found")
				return
			}
			slog.Error("marketing: load campaign for code mint", "error", err, "campaign_id", campaignID)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			slog.Error("marketing: begin tx", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		code, err := mintCode(ctx, tx, campaignID, user.ID, in, itemID)
		if err != nil {
			slog.Error("marketing: mint code", "error", err, "campaign_id", campaignID)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			slog.Error("marketing: commit code mint", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		decorate(d, &code)
		writeJSON(w, http.StatusCreated, code)
	}
}

// ── PATCH /codes/{id} ──

// PatchCodeHandler re-points a code WITHOUT reprinting it (§5 row 6). `short`
// is absent from the body by construction: a code that is already on a sign
// must keep its key, and that is the whole reason the payload is a short URL
// resolved server-side (decision 189).
func PatchCodeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		id := chi.URLParam(r, "id")
		var in struct {
			Active    *bool   `json:"active"`
			Landing   *string `json:"landing"`
			Placement *string `json:"placement"`
			Variant   *string `json:"variant"`
			ItemID    *string `json:"item_id"`
		}
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_payload")
			return
		}
		if in.Landing != nil && !containsString(landings, *in.Landing) {
			writeError(w, http.StatusBadRequest, "bad_landing")
			return
		}

		var out codeDTO
		err := d.Pool.QueryRow(r.Context(), `
			UPDATE qr_codes SET
			  active    = COALESCE($2, active),
			  landing   = COALESCE($3, landing),
			  placement = COALESCE($4, placement),
			  variant   = COALESCE($5, variant),
			  item_id   = COALESCE($6::uuid, item_id)
			WHERE id = $1
			RETURNING id::text, short, campaign_id::text, channel, channel_label,
			          item_id::text, placement, variant, landing, active, v`,
			id, in.Active, in.Landing, in.Placement, in.Variant, in.ItemID,
		).Scan(&out.ID, &out.Short, &out.CampaignID, &out.Channel, &out.ChannelLabel,
			&out.ItemID, &out.Placement, &out.Variant, &out.Landing, &out.Active, &out.V)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "code_not_found")
			return
		}
		if err != nil {
			slog.Error("marketing: patch code", "error", err, "code_id", id)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		decorate(d, &out)
		writeJSON(w, http.StatusOK, out)
	}
}
