package marketing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// maxBodyBytes caps a request body. The largest legitimate create is a name,
// an offer line and nine channels.
const maxBodyBytes = 1 << 16

// decodeBody reads a JSON body, tolerating an empty one (every PATCH route here
// treats "no fields" as a no-op rather than an error).
func decodeBody(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(body))) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}

// isBadUUID reports whether err is Postgres' invalid_text_representation — what
// comes back when a path parameter that should be a uuid is not one. A typo in
// a URL is a 404, not a 500.
func isBadUUID(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "22P02"
}

// periodStart turns §5's period parameter into a lower bound for the funnel.
// Unknown values fall back to 30d rather than erroring: the designed list has a
// fixed segmented control, so an unexpected value is a client bug, not an
// operator one, and returning data beats returning a 400 on a read.
func periodStart(period string) *time.Time {
	now := time.Now()
	var d time.Duration
	switch period {
	case "7d":
		d = 7 * 24 * time.Hour
	case "90d":
		d = 90 * 24 * time.Hour
	case "all":
		return nil
	default: // "", "30d", anything else
		d = 30 * 24 * time.Hour
	}
	t := now.Add(-d)
	return &t
}

// ── POST /campaigns ──

type createCampaignRequest struct {
	Name           string         `json:"name"`
	OfferText      string         `json:"offer_text"`
	FaceValueCents *int           `json:"face_value_cents"`
	RunsDays       *int           `json:"runs_days"`
	ItemID         *string        `json:"item_id"`
	Landing        *string        `json:"landing"`
	Channels       []channelInput `json:"channels"`
}

// validate returns "" when usable, else the 400 error slug.
func (c createCampaignRequest) validate() string {
	if strings.TrimSpace(c.Name) == "" {
		return "name_required"
	}
	if strings.TrimSpace(c.OfferText) == "" {
		return "offer_text_required"
	}
	// face_value_cents is REQUIRED and explicit, never inferred from offer
	// text: §8 — "the create sheet makes Value explicit precisely because the
	// discount math needs it". A "free side" offer still has a face value.
	if c.FaceValueCents == nil || *c.FaceValueCents < 0 {
		return "face_value_cents_required"
	}
	if c.RunsDays == nil || *c.RunsDays < 1 {
		return "runs_days_required"
	}
	if len(c.Channels) == 0 {
		// A campaign with no channel mints no code and can never be scanned.
		return "channels_required"
	}
	if c.Landing != nil && !containsString(landings, *c.Landing) {
		return "bad_landing"
	}
	for _, ch := range c.Channels {
		if bad := ch.validate(); bad != "" {
			return bad
		}
	}
	return ""
}

// CreateCampaignHandler is §5 row 2: one sheet in, one campaign and ONE CODE
// PER CHANNEL out, all in a single transaction, then the Supabase projection
// AFTER the commit.
//
// The ordering is the point. Minting codes inside the campaign's transaction
// means a manager never sees a campaign with some of its codes; projecting
// after the commit means a dead substrate never costs them the campaign.
func CreateCampaignHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := requireManager(w, r)
		if user == nil {
			return
		}
		var in createCampaignRequest
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_payload")
			return
		}
		if bad := in.validate(); bad != "" {
			writeError(w, http.StatusBadRequest, bad)
			return
		}
		ctx := r.Context()

		// requires_online is DERIVED here and never read off the request body
		// (#5). An unreachable substrate yields the default threshold with a
		// WARN — see thresholdCents.
		threshold := thresholdCents(ctx, d.Projection)
		requiresOnline := *in.FaceValueCents >= threshold

		landing := "signup"
		if in.Landing != nil {
			landing = *in.Landing
		}

		tx, err := d.Pool.Begin(ctx)
		if err != nil {
			slog.Error("marketing: begin create tx", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		slug, err := uniqueSlug(ctx, tx, slugify(in.Name))
		if err != nil {
			slog.Error("marketing: slug", "error", err, "name", in.Name)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// id is supplied explicitly, not defaulted: it IS the Supabase
		// projection key (campaigns_admin.id == campaigns.id), so the value
		// that lands locally is the value projected.
		var camp campaignDTO
		var campItemID *string
		err = tx.QueryRow(ctx, `
			INSERT INTO campaigns_admin
			  (id, slug, name, offer_text, face_value_cents, requires_online,
			   item_id, landing, starts_at, ends_at, created_by)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, now(),
			        now() + ($8 || ' days')::interval, $9)
			RETURNING id::text, slug, name, offer_text, face_value_cents,
			          requires_online, item_id::text, landing, status,
			          starts_at, ends_at, projected_at`,
			slug, strings.TrimSpace(in.Name), strings.TrimSpace(in.OfferText),
			*in.FaceValueCents, requiresOnline, in.ItemID, landing,
			fmt.Sprintf("%d", *in.RunsDays), user.ID,
		).Scan(&camp.ID, &camp.Slug, &camp.Name, &camp.OfferText, &camp.FaceValueCents,
			&camp.RequiresOnline, &campItemID, &camp.Landing, &camp.Status,
			&camp.StartsAt, &camp.EndsAt, &camp.ProjectedAt)
		if err != nil {
			if isBadUUID(err) || isFKViolation(err, "campaigns_admin_item_id_fkey") {
				writeError(w, http.StatusBadRequest, "bad_item_id")
				return
			}
			slog.Error("marketing: insert campaign", "error", err, "slug", slug)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// One code per channel, in this same transaction.
		codes := make([]codeDTO, 0, len(in.Channels))
		for _, ch := range in.Channels {
			// §4: qr_codes.item_id "defaults to the campaign's item". §5's
			// create body carries no per-channel item, and "more than one item
			// per code" is an operator fork the slate deliberately parked — so
			// every minted code inherits the campaign's single item.
			code, err := mintCode(ctx, tx, camp.ID, user.ID, ch, campItemID)
			if err != nil {
				slog.Error("marketing: mint code during create", "error", err, "channel", ch.Channel)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			decorate(d, &code)
			codes = append(codes, code)
		}

		if err := tx.Commit(ctx); err != nil {
			slog.Error("marketing: commit create", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		// ── AFTER COMMIT: the Supabase projection (decision 187) ──
		var warnings []string
		if at, err := projectAndStamp(ctx, d, camp); err != nil {
			// Fail LOUD, never silent: the campaign is saved, the tablets do
			// not have it yet, and the response says so.
			slog.Error("marketing: campaign projection failed; projected_at left NULL",
				"error", err, "campaign_id", camp.ID)
			warnings = append(warnings, WarningNotProjected)
		} else {
			camp.ProjectedAt = at
		}

		camp.Item = loadItemRef(ctx, d, camp.ID)
		camp.Funnel = funnelDTO{}
		camp.Money = zeroMoney()
		camp.Codes = codes
		writeJSON(w, http.StatusCreated, createCampaignResponse{
			Campaign: camp, Codes: codes, Warnings: warnings,
		})
	}
}

func isFKViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}

// projectAndStamp projects the four tablet columns and, on success, stamps
// projected_at. On any failure it returns the error and touches nothing, so
// projected_at keeps whatever it had (NULL on a create, the previous timestamp
// on a PATCH).
func projectAndStamp(ctx context.Context, d Deps, camp campaignDTO) (*time.Time, error) {
	if err := ProjectCampaign(ctx, d.Projection, ProjectionRow{
		ID:             camp.ID,
		Name:           camp.Name,
		FaceValueCents: camp.FaceValueCents,
		RequiresOnline: camp.RequiresOnline,
	}); err != nil {
		return nil, err
	}
	var at time.Time
	if err := d.Pool.QueryRow(ctx,
		`UPDATE campaigns_admin SET projected_at = now(), updated_at = now()
		 WHERE id = $1 RETURNING projected_at`, camp.ID).Scan(&at); err != nil {
		// The projection landed but the stamp did not. Report it: a NULL
		// projected_at with a projected row is the one state that would read as
		// "not on tablets" while the tablets have it.
		return nil, fmt.Errorf("stamp projected_at: %w", err)
	}
	return &at, nil
}

// loadItemRef fills the item's NAME once the id is known. Kept out of the main
// INSERT because menu_items is a join and the create path should not hold the
// transaction open for it.
func loadItemRef(ctx context.Context, d Deps, campaignID string) *itemRef {
	var ref itemRef
	err := d.Pool.QueryRow(ctx, `
		SELECT m.id::text, m.name
		FROM campaigns_admin c JOIN menu_items m ON m.id = c.item_id
		WHERE c.id = $1`, campaignID).Scan(&ref.ID, &ref.Name)
	if err != nil {
		return nil // no item, or a read that is not worth failing a 201 over
	}
	return &ref
}

// ── GET /campaigns ──

// ListCampaignsHandler is §5 row 1. The funnel's `scans` is real (from
// qr_scans, already deduped at write time by the landing handler); `signups`
// and `redeemed` and the whole money block are the zero shapes H5/H3a/H3b fill
// — see the card's merge-intent.
func ListCampaignsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		ctx := r.Context()
		since := periodStart(r.URL.Query().Get("period"))

		rows, err := d.Pool.Query(ctx, `
			SELECT c.id::text, c.slug, c.name, c.offer_text, c.face_value_cents,
			       c.requires_online, c.landing, c.status, c.starts_at, c.ends_at,
			       c.projected_at, m.id::text, m.name
			FROM campaigns_admin c
			LEFT JOIN menu_items m ON m.id = c.item_id
			ORDER BY c.created_at DESC`)
		if err != nil {
			slog.Error("marketing: list campaigns", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		defer rows.Close()

		out := listCampaignsResponse{Campaigns: []campaignDTO{}}
		for rows.Next() {
			var c campaignDTO
			var itemID, itemName *string
			if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.OfferText, &c.FaceValueCents,
				&c.RequiresOnline, &c.Landing, &c.Status, &c.StartsAt, &c.EndsAt,
				&c.ProjectedAt, &itemID, &itemName); err != nil {
				slog.Error("marketing: scan campaign", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			if itemID != nil {
				c.Item = &itemRef{ID: *itemID, Name: derefOr(itemName, "")}
			}
			c.Money = zeroMoney()
			out.Campaigns = append(out.Campaigns, c)
		}
		if err := rows.Err(); err != nil {
			slog.Error("marketing: iterate campaigns", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		for i := range out.Campaigns {
			codes, scans, err := loadCodes(ctx, d, out.Campaigns[i].ID, since)
			if err != nil {
				slog.Error("marketing: load codes", "error", err, "campaign_id", out.Campaigns[i].ID)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			out.Campaigns[i].Codes = codes
			out.Campaigns[i].Funnel = funnelDTO{Scans: scans}
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// loadCodes returns a campaign's codes with their per-code scan counts in the
// period, plus the campaign's total.
//
// The 10-minute (short, ip_hash) dedupe §5 defines is enforced ONCE, at WRITE
// time in the landing handler — so this is a plain count. One definition in one
// place: a second, read-side rule is exactly how two slices stop agreeing.
func loadCodes(ctx context.Context, d Deps, campaignID string, since *time.Time) ([]codeDTO, int, error) {
	rows, err := d.Pool.Query(ctx, `
		SELECT k.id::text, k.short, k.campaign_id::text, k.channel, k.channel_label,
		       k.item_id::text, k.placement, k.variant, k.landing, k.active, k.v,
		       (SELECT count(*) FROM qr_scans s
		         WHERE s.short = k.short
		           AND ($2::timestamptz IS NULL OR s.scanned_at >= $2))::int AS scans
		FROM qr_codes k
		WHERE k.campaign_id = $1
		ORDER BY k.created_at`, campaignID, since)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	codes := []codeDTO{}
	total := 0
	for rows.Next() {
		var c codeDTO
		if err := rows.Scan(&c.ID, &c.Short, &c.CampaignID, &c.Channel, &c.ChannelLabel,
			&c.ItemID, &c.Placement, &c.Variant, &c.Landing, &c.Active, &c.V, &c.Scans); err != nil {
			return nil, 0, err
		}
		decorate(d, &c)
		total += c.Scans
		codes = append(codes, c)
	}
	return codes, total, rows.Err()
}

// ── GET /campaigns/{id} ──

// GetCampaignHandler is §5 row 3: campaign + codes + money. The money block
// carries the two avg_order fields the design's Money card shows; both are null
// until H3b lands the arithmetic.
func GetCampaignHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		ctx := r.Context()
		id := chi.URLParam(r, "id")
		since := periodStart(r.URL.Query().Get("period"))

		var c campaignDTO
		var itemID, itemName *string
		err := d.Pool.QueryRow(ctx, `
			SELECT c.id::text, c.slug, c.name, c.offer_text, c.face_value_cents,
			       c.requires_online, c.landing, c.status, c.starts_at, c.ends_at,
			       c.projected_at, m.id::text, m.name
			FROM campaigns_admin c
			LEFT JOIN menu_items m ON m.id = c.item_id
			WHERE c.id = $1`, id).
			Scan(&c.ID, &c.Slug, &c.Name, &c.OfferText, &c.FaceValueCents,
				&c.RequiresOnline, &c.Landing, &c.Status, &c.StartsAt, &c.EndsAt,
				&c.ProjectedAt, &itemID, &itemName)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "campaign_not_found")
			return
		}
		if err != nil {
			slog.Error("marketing: get campaign", "error", err, "campaign_id", id)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if itemID != nil {
			c.Item = &itemRef{ID: *itemID, Name: derefOr(itemName, "")}
		}
		codes, scans, err := loadCodes(ctx, d, c.ID, since)
		if err != nil {
			slog.Error("marketing: load codes", "error", err, "campaign_id", c.ID)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		c.Codes = codes
		c.Funnel = funnelDTO{Scans: scans}
		c.Money = zeroMoney()
		writeJSON(w, http.StatusOK, c)
	}
}

// ── PATCH /campaigns/{id} ──

// PatchCampaignHandler is §5 row 4. `status` drives the lifecycle the designed
// list filters on, and `name` is one of the FOUR projected columns — so a
// rename re-projects, under the same fail-loud rule as a create.
func PatchCampaignHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		ctx := r.Context()
		id := chi.URLParam(r, "id")
		var in struct {
			Status    *string    `json:"status"`
			Name      *string    `json:"name"`
			OfferText *string    `json:"offer_text"`
			EndsAt    *time.Time `json:"ends_at"`
		}
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_payload")
			return
		}
		// §5 names three reachable statuses. 'scheduled' is set by the create
		// path's own logic, not by a manager editing a live campaign, so it is
		// deliberately not accepted here.
		if in.Status != nil && !containsString([]string{"live", "paused", "ended"}, *in.Status) {
			writeError(w, http.StatusBadRequest, "bad_status")
			return
		}
		if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
			writeError(w, http.StatusBadRequest, "name_required")
			return
		}

		var c campaignDTO
		var itemID, itemName *string
		err := d.Pool.QueryRow(ctx, `
			WITH upd AS (
			  UPDATE campaigns_admin SET
			    status     = COALESCE($2, status),
			    name       = COALESCE($3, name),
			    offer_text = COALESCE($4, offer_text),
			    ends_at    = COALESCE($5, ends_at),
			    updated_at = now()
			  WHERE id = $1
			  RETURNING *
			)
			SELECT u.id::text, u.slug, u.name, u.offer_text, u.face_value_cents,
			       u.requires_online, u.landing, u.status, u.starts_at, u.ends_at,
			       u.projected_at, m.id::text, m.name
			FROM upd u LEFT JOIN menu_items m ON m.id = u.item_id`,
			id, in.Status, in.Name, in.OfferText, in.EndsAt).
			Scan(&c.ID, &c.Slug, &c.Name, &c.OfferText, &c.FaceValueCents,
				&c.RequiresOnline, &c.Landing, &c.Status, &c.StartsAt, &c.EndsAt,
				&c.ProjectedAt, &itemID, &itemName)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "campaign_not_found")
			return
		}
		if err != nil {
			slog.Error("marketing: patch campaign", "error", err, "campaign_id", id)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		if itemID != nil {
			c.Item = &itemRef{ID: *itemID, Name: derefOr(itemName, "")}
		}

		// A rename changes a projected column, so re-project. Same fail-loud
		// rule: on failure projected_at keeps its previous value and the
		// response carries the warning.
		var warnings []string
		if in.Name != nil {
			if at, err := projectAndStamp(ctx, d, c); err != nil {
				slog.Error("marketing: re-projection after rename failed", "error", err, "campaign_id", c.ID)
				warnings = append(warnings, WarningNotProjected)
			} else {
				c.ProjectedAt = at
			}
		}

		codes, scans, err := loadCodes(ctx, d, c.ID, periodStart(""))
		if err != nil {
			slog.Error("marketing: load codes", "error", err, "campaign_id", c.ID)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		c.Codes = codes
		c.Funnel = funnelDTO{Scans: scans}
		c.Money = zeroMoney()

		if len(warnings) > 0 {
			// The campaign shape plus the warning, so the UI can show the
			// "not on tablets yet" pill on a rename too.
			writeJSON(w, http.StatusOK, struct {
				campaignDTO
				Warnings []string `json:"warnings"`
			}{c, warnings})
			return
		}
		writeJSON(w, http.StatusOK, c)
	}
}

func derefOr(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}
