package marketing

// reconciliation.go — card H3b, run 20261002. The three buckets, the five
// decision kinds, the declined bucket.
//
// # What this file is for
//
// A tablet at the counter records an attempt (internal/marketing/mirror.go
// copies it into scan_attempts_mirror); Toast's daily export records the order
// it should have been attached to (internal/toast/orderdetails.go lands
// toast_orders). The join between them is NOT reliable — the staff member types
// the order number, or forgets to, or overrides offline for a code the device
// could not verify. This file is where a human closes that gap, and
// reconciliation_decisions is the append-only log of what they decided.
//
// # The ladder (handoff §5, card H3b: "overrides → orphans → unmatched")
//
// Every ACCEPTED attempt lands in exactly one bucket, in this order:
//
//	override  — offline_override and not yet verified/rejected. FIRST, because
//	            Q-KR2 grades on "every accepted offline override is auditable and
//	            reconciled first": an override spends money on the operator's
//	            word alone, so it outranks a bookkeeping gap.
//	orphan    — no order number at all. Nothing to join on; a human must say
//	            what happened.
//	unmatched — an order number that is not in toast_orders. Carries the nearest
//	            order within ±30 minutes of scanned_at as a suggestion, so the
//	            common case (a transposed digit) is one tap.
//	matched   — joined to a toast_orders row. Out of the queue; it is revenue.
//	declined  — a human said "explained, not excused". Out of the queue, into the
//	            declined bucket, reopenable.
//
// `declined` SHORT-CIRCUITS the ladder (a declined override is declined, not an
// override), which is why the switch below tests it first.
//
// # 🛑 Append-only, never UPDATE
//
// reconciliation_decisions is a log. "The current state of attempt X" is its
// LATEST row (DISTINCT ON … ORDER BY decided_at DESC, id DESC), never a mutated
// one. Reopen is a new row, not a delete. That is what makes the declined
// bucket able to say reason, note, who and when, and what lets an auditor see
// that a call was changed rather than only that it is now different.
//
// # 🛑 Timezone dependency, scoped and bounded (card H3a's G6, 2026-10-02)
//
// Card H3a parses the Toast export's `Opened` NAIVELY and stamps it with a
// hardcoded America/Chicago, while every other scheduled reader in this tree
// (purchasing, recipes, inventory) reads users.DefaultTimezone =
// America/New_York (ledger T-26 decision 83, migration 0072). Nothing in the
// repo establishes which wall clock Toast actually writes, so `opened_at`'s zone
// is UNCONFIRMED — possibly an hour off. That question is the operator's and is
// not resolved here. What matters is how far it reaches into this file:
//
//	MATCHED and the orphan rate — UNAFFECTED, and provably so. The matched
//	bucket joins on EQUALITY of scan_attempts_mirror.pos_business_date
//	(device-reported) and toast_orders.business_date, and business_date is
//	`opened.Date()` of the wall-clock string, i.e. zone-INDEPENDENT (H3a's G6
//	constructed the 00:30 and 23:50 cases). The orphan rate's numerator is
//	defined on matched(), so it does not move either. Revenue, discount, net and
//	every slice read matched(), so they do not move.
//
//	The ±30-MINUTE SUGGESTION — this is the only thing that moves. A one-hour
//	offset puts every real order outside the window and the suggestion goes
//	null. It CANNOT reclassify anything: bucket() reads the ORDER NUMBER, never
//	the suggestion, so an unmatched attempt with no suggestion still reads as
//	`unmatched`, never as `orphan`. Nothing silently reclassifies and no metric
//	changes; the human just loses a one-tap hint.
//
// reconNearestOrder therefore keeps the card's ±30-minute rule exactly as
// specified as its FIRST rung, and adds a clearly-labelled SECOND rung — the
// nearest order on the same (zone-independent) business_date — so a one-hour
// offset surfaces as `{"basis":"business_date","gap_seconds":3600}` instead of
// as silence. The suggestion is advisory by §8's own framing, so widening the
// ADVICE changes no rule decision 190 fixes.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ReconSuggestionWindow is the ±window around scanned_at inside which a Toast
// order is offered as the nearest-order suggestion for an unmatched attempt.
const ReconSuggestionWindow = 30 * time.Minute

// ReconOrphanThreshold is the line the designed health card draws (§5's
// `threshold:0.10`). It is a DISPLAY constant, not a gate: nothing refuses at
// 10%, the card turns red.
const ReconOrphanThreshold = 0.10

// reconDecisionKinds / reconDeclineReasons mirror migration 0084's CHECK
// constraints exactly. A sixth kind or a seventh reason is a PARK, not an edit.
var reconDecisionKinds = []string{"matched", "declined", "reopened", "verified", "rejected"}

var reconDeclineReasons = []string{
	"no_such_order", "customer_left", "comped", "duplicate_scan",
	"no_discount_applied", "other",
}

// ReconDuplicateScanReason is the one decline reason decision 190 excludes from
// the orphan rate: a double scan of one code is a counter mistake, not an
// unexplained redemption.
const ReconDuplicateScanReason = "duplicate_scan"

// ── the row model ──

// reconAttempt is one accepted mirrored attempt with everything BOTH the queue
// and the stats engine need, loaded once by reconLoadAttempts.
//
// One loader for both is the point: handoff §5's metric-definitions paragraph
// exists so the three slices and the overview agree, and the surest way to make
// them disagree is to compute "matched" twice.
type reconAttempt struct {
	ID               string
	ScannedAt        time.Time
	BusinessDate     time.Time
	DeviceID         string
	Override         bool
	OverrideBy       *string
	UnverifiedCode   bool
	PolicyUnresolved bool
	DeviceReason     *string
	// OrderNumber is the EFFECTIVE order number: the one a `matched` decision
	// supplied if there is one, else the device's pos_order_number.
	OrderNumber        *string
	RedeemedValueCents *int

	Decision       *string
	DecisionReason *string
	DecisionNote   *string
	DecidedBy      *string
	DecidedAt      *time.Time

	CodeID       *string
	Short        *string
	Channel      *string
	ChannelLabel *string
	ItemID       *string
	ItemName     *string
	CampaignID   *string
	CampaignName *string
	// FaceValueCents is the campaign's face value — the IMPLIED discount for a
	// row with no matched order (decision 190). nil means no campaign resolved,
	// which is the one case where a row cannot be priced at all; the stats
	// engine counts those in `discount_unknown_rows` rather than treating them
	// as free.
	FaceValueCents *int

	OrderAmountCents   *int
	OrderDiscountCents *int
	OrderOpenedAt      *time.Time
	OrderVoided        *bool
}

// matched reports whether this attempt is joined to a toast_orders row. It is
// the ONE definition of "matched" in this package: revenue, the per-row discount
// rule, the health card and the queue all read it.
//
// A VOIDED order still counts as matched. §5 defines revenue as "Σ
// toast_orders.amount_cents over matched attempts" with no voided exclusion, and
// inventing one here would be a change to the money rule (decision 190's PARK
// list). `voided` is surfaced on the queue row instead, so a human can decline
// it with a reason — which is the designed way that money leaves the total.
func (a reconAttempt) matched() bool { return a.OrderAmountCents != nil }

// latestDecision returns the latest decision kind, or "".
func (a reconAttempt) latestDecision() string {
	if a.Decision == nil {
		return ""
	}
	return *a.Decision
}

// bucket places the attempt on the ladder documented at the top of this file.
func (a reconAttempt) bucket() string {
	switch a.latestDecision() {
	case "declined":
		return "declined"
	case "verified", "rejected":
		// The override itself is resolved; the attempt rejoins the match ladder.
	default:
		if a.Override {
			return "override"
		}
	}
	if a.OrderNumber == nil || strings.TrimSpace(*a.OrderNumber) == "" {
		return "orphan"
	}
	if a.matched() {
		return "matched"
	}
	return "unmatched"
}

// countsInOrphanRate is decision 190's refinement: declines count EXCEPT
// duplicate_scan. A matched attempt never counts.
func (a reconAttempt) countsInOrphanRate() bool {
	if a.matched() {
		return false
	}
	if a.latestDecision() == "declined" && a.DecisionReason != nil &&
		*a.DecisionReason == ReconDuplicateScanReason {
		return false
	}
	return true
}

// ── loading ──

// reconAttemptsSQL is the one query both halves of this card read. `latest` is
// the append-only log collapsed to the current call per attempt.
const reconAttemptsSQL = `
WITH latest AS (
  SELECT DISTINCT ON (attempt_id)
         attempt_id, decision, order_number, reason, note, decided_by, decided_at
  FROM reconciliation_decisions
  ORDER BY attempt_id, decided_at DESC, id DESC
), eff AS (
  SELECT a.*, l.decision, l.reason AS d_reason, l.note AS d_note,
         l.decided_by, l.decided_at,
         CASE WHEN l.decision = 'matched'
              THEN COALESCE(l.order_number, a.pos_order_number)
              ELSE a.pos_order_number END AS eff_order_number
  FROM scan_attempts_mirror a
  LEFT JOIN latest l ON l.attempt_id = a.id
  WHERE a.status = 'accepted'
    AND ($1::timestamptz IS NULL OR a.scanned_at >= $1)
)
SELECT e.id::text, e.scanned_at, e.pos_business_date, e.device_id,
       e.offline_override, e.override_by, e.unverified_code, e.policy_unresolved,
       e.reason, e.eff_order_number,
       (e.redeemed_value * 100)::bigint,
       e.decision, e.d_reason, e.d_note,
       COALESCE(NULLIF(trim(u.first_name || ' ' || u.last_name), ''), u.email),
       e.decided_at,
       k.id::text, k.short, k.channel, k.channel_label,
       COALESCE(k.item_id, cam.item_id)::text, mi.name,
       cam.id::text, cam.name, cam.face_value_cents,
       o.amount_cents, o.discount_cents, o.opened_at, o.voided
FROM eff e
LEFT JOIN users u ON u.id = e.decided_by
LEFT JOIN qr_codes k ON k.id = e.code_id
LEFT JOIN campaigns_admin cam ON cam.id = COALESCE(e.campaign_id, k.campaign_id)
LEFT JOIN menu_items mi ON mi.id = COALESCE(k.item_id, cam.item_id)
LEFT JOIN toast_orders o
       ON o.business_date = e.pos_business_date
      AND o.order_number  = e.eff_order_number
ORDER BY e.scanned_at, e.id`

// reconLoadAttempts reads every accepted attempt in the period.
func reconLoadAttempts(ctx context.Context, pool *pgxpool.Pool, since *time.Time) ([]reconAttempt, error) {
	rows, err := pool.Query(ctx, reconAttemptsSQL, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []reconAttempt{}
	for rows.Next() {
		var a reconAttempt
		var redeemedCents *int64
		if err := rows.Scan(
			&a.ID, &a.ScannedAt, &a.BusinessDate, &a.DeviceID,
			&a.Override, &a.OverrideBy, &a.UnverifiedCode, &a.PolicyUnresolved,
			&a.DeviceReason, &a.OrderNumber, &redeemedCents,
			&a.Decision, &a.DecisionReason, &a.DecisionNote, &a.DecidedBy, &a.DecidedAt,
			&a.CodeID, &a.Short, &a.Channel, &a.ChannelLabel,
			&a.ItemID, &a.ItemName, &a.CampaignID, &a.CampaignName, &a.FaceValueCents,
			&a.OrderAmountCents, &a.OrderDiscountCents, &a.OrderOpenedAt, &a.OrderVoided,
		); err != nil {
			return nil, err
		}
		if redeemedCents != nil {
			// redeemed_value is numeric DOLLARS upstream (projection.go's own
			// note). The ×100 happened in SQL, in Postgres numeric — exact, and
			// no float64 ever touches a money path here.
			v := int(*redeemedCents)
			a.RedeemedValueCents = &v
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ── wire shapes (Card 5's UI is built against these json tags tonight) ──

// ReconOrderRef is a toast_orders row as the queue renders it — the matched
// order, or the ±30-minute suggestion.
type ReconOrderRef struct {
	OrderNumber   string    `json:"order_number"`
	OpenedAt      time.Time `json:"opened_at"`
	AmountCents   int       `json:"amount_cents"`
	DiscountCents int       `json:"discount_cents"`
	Voided        bool      `json:"voided"`
}

// ReconSuggestion is the nearest-order hint on an unmatched attempt. `basis`
// says which rung produced it and `gap_seconds` how far off it is, so a
// systematic offset (see the timezone note at the top of this file) is visible
// on the row rather than hidden behind a null.
//
//	window        — within ±ReconSuggestionWindow of scanned_at. The card's rule.
//	business_date — the nearest order on the attempt's own business date, OUTSIDE
//	                that window. Advisory only, and labelled so a UI can say so.
type ReconSuggestion struct {
	ReconOrderRef
	GapSeconds int    `json:"gap_seconds"`
	Basis      string `json:"basis"`
}

// ReconDecisionRef is the latest reconciliation_decisions row for an attempt.
type ReconDecisionRef struct {
	Decision  string    `json:"decision"`
	Reason    *string   `json:"reason"`
	Note      *string   `json:"note"`
	DecidedBy string    `json:"decided_by"`
	DecidedAt time.Time `json:"decided_at"`
}

// ReconQueueRow is one attempt awaiting a human.
type ReconQueueRow struct {
	Bucket             string            `json:"bucket"`
	ID                 string            `json:"id"`
	ScannedAt          time.Time         `json:"scanned_at"`
	BusinessDate       string            `json:"business_date"`
	DeviceID           string            `json:"device_id"`
	OrderNumber        *string           `json:"order_number"`
	OfflineOverride    bool              `json:"offline_override"`
	OverrideBy         *string           `json:"override_by"`
	UnverifiedCode     bool              `json:"unverified_code"`
	PolicyUnresolved   bool              `json:"policy_unresolved"`
	DeviceReason       *string           `json:"device_reason"`
	RedeemedValueCents *int              `json:"redeemed_value_cents"`
	CampaignID         *string           `json:"campaign_id"`
	CampaignName       *string           `json:"campaign_name"`
	FaceValueCents     *int              `json:"face_value_cents"`
	CodeShort          *string           `json:"code_short"`
	Channel            *string           `json:"channel"`
	ChannelLabel       *string           `json:"channel_label"`
	ItemName           *string           `json:"item_name"`
	Order              *ReconOrderRef    `json:"order"`
	Suggestion         *ReconSuggestion  `json:"suggestion"`
	Decision           *ReconDecisionRef `json:"decision"`
}

// ReconQueueResponse is §5's `GET /reconciliation/queue`.
//
// `queue` is the three buckets CONCATENATED in ladder order (overrides →
// orphans → unmatched) with a `bucket` tag on each row, because that is the one
// list the designed queue renders; `overrides`/`orphans`/`unmatched` are the
// same rows grouped, because that is what §5 specifies and what the section
// counts come from. They are built from ONE pass over ONE slice, so they cannot
// drift.
type ReconQueueResponse struct {
	Queue         []ReconQueueRow `json:"queue"`
	Overrides     []ReconQueueRow `json:"overrides"`
	Orphans       []ReconQueueRow `json:"orphans"`
	Unmatched     []ReconQueueRow `json:"unmatched"`
	MatchedCount  int             `json:"matched_count"`
	DeclinedCount int             `json:"declined_count"`
}

// ReconDeclinedRow is §5's declined bucket: the attempt plus reason, note, who
// and when, flat on the row.
type ReconDeclinedRow struct {
	ReconQueueRow
	Reason    *string   `json:"reason"`
	Note      *string   `json:"note"`
	DecidedBy string    `json:"decided_by"`
	DecidedAt time.Time `json:"decided_at"`
}

// ReconDeclinedResponse is `GET /reconciliation/declined`.
type ReconDeclinedResponse struct {
	Declined []ReconDeclinedRow `json:"declined"`
}

// ReconDecisionResponse is what every write returns: the decision that was
// logged plus the attempt's new bucket, so the UI can move the row without a
// second round trip.
type ReconDecisionResponse struct {
	AttemptID string           `json:"attempt_id"`
	Bucket    string           `json:"bucket"`
	Decision  ReconDecisionRef `json:"decision"`
}

// ── row rendering ──

func reconRow(a reconAttempt, bucket string, orders []reconOrder) ReconQueueRow {
	row := ReconQueueRow{
		Bucket:             bucket,
		ID:                 a.ID,
		ScannedAt:          a.ScannedAt,
		BusinessDate:       a.BusinessDate.Format("2006-01-02"),
		DeviceID:           a.DeviceID,
		OrderNumber:        a.OrderNumber,
		OfflineOverride:    a.Override,
		OverrideBy:         a.OverrideBy,
		UnverifiedCode:     a.UnverifiedCode,
		PolicyUnresolved:   a.PolicyUnresolved,
		DeviceReason:       a.DeviceReason,
		RedeemedValueCents: a.RedeemedValueCents,
		CampaignID:         a.CampaignID,
		CampaignName:       a.CampaignName,
		FaceValueCents:     a.FaceValueCents,
		CodeShort:          a.Short,
		Channel:            a.Channel,
		ChannelLabel:       a.ChannelLabel,
		ItemName:           a.ItemName,
	}
	if a.matched() {
		row.Order = &ReconOrderRef{
			OrderNumber:   derefOr(a.OrderNumber, ""),
			OpenedAt:      derefTime(a.OrderOpenedAt),
			AmountCents:   *a.OrderAmountCents,
			DiscountCents: derefInt(a.OrderDiscountCents),
			Voided:        a.OrderVoided != nil && *a.OrderVoided,
		}
	}
	if bucket == "unmatched" {
		row.Suggestion = reconNearestOrder(a, orders)
	}
	if a.Decision != nil {
		row.Decision = &ReconDecisionRef{
			Decision:  *a.Decision,
			Reason:    a.DecisionReason,
			Note:      a.DecisionNote,
			DecidedBy: derefOr(a.DecidedBy, ""),
			DecidedAt: derefTime(a.DecidedAt),
		}
	}
	return row
}

// reconOrder is a toast_orders row in the suggestion candidate set.
type reconOrder struct {
	BusinessDate time.Time
	OrderNumber  string
	OpenedAt     time.Time
	AmountCents  int
	Discount     int
	Voided       bool
}

// reconNearestOrder returns the nearest-order hint for an unmatched attempt, by
// the two-rung ladder documented on ReconSuggestion. Ties break on the lower
// order number so the suggestion is stable across requests.
func reconNearestOrder(a reconAttempt, orders []reconOrder) *ReconSuggestion {
	day := a.BusinessDate.Format("2006-01-02")
	pick := func(within time.Duration, sameDay bool) *ReconSuggestion {
		var best *reconOrder
		var bestGap time.Duration
		for i := range orders {
			o := &orders[i]
			if sameDay && o.BusinessDate.Format("2006-01-02") != day {
				continue
			}
			gap := o.OpenedAt.Sub(a.ScannedAt)
			if gap < 0 {
				gap = -gap
			}
			if within > 0 && gap > within {
				continue
			}
			if best == nil || gap < bestGap || (gap == bestGap && o.OrderNumber < best.OrderNumber) {
				best, bestGap = o, gap
			}
		}
		if best == nil {
			return nil
		}
		return &ReconSuggestion{
			ReconOrderRef: ReconOrderRef{
				OrderNumber:   best.OrderNumber,
				OpenedAt:      best.OpenedAt,
				AmountCents:   best.AmountCents,
				DiscountCents: best.Discount,
				Voided:        best.Voided,
			},
			GapSeconds: int(bestGap / time.Second),
		}
	}
	// Rung 1 — the card's rule, verbatim: within ±30 min of scanned_at, over
	// every order in the period (NOT constrained to the business date, so a
	// scan near midnight still sees the neighbouring day's orders).
	if s := pick(ReconSuggestionWindow, false); s != nil {
		s.Basis = "window"
		return s
	}
	// Rung 2 — advisory: the nearest order on the attempt's own business date,
	// which is the zone-independent key. This is what makes a systematic
	// opened_at offset visible instead of silent.
	if s := pick(0, true); s != nil {
		s.Basis = "business_date"
		return s
	}
	return nil
}

// reconLoadOrders reads the toast_orders rows in the period. The suggestion
// search and the "orders without the offer" baseline both read this slice; a
// food truck's daily order count makes one read cheaper than two indexed
// correlated subqueries, and it keeps the ±30-minute arithmetic in ONE place.
func reconLoadOrders(ctx context.Context, pool *pgxpool.Pool, since *time.Time) ([]reconOrder, error) {
	rows, err := pool.Query(ctx, `
		SELECT business_date, order_number, opened_at, amount_cents, discount_cents, voided
		FROM toast_orders
		WHERE ($1::timestamptz IS NULL OR business_date >= ($1::timestamptz)::date)
		ORDER BY business_date, order_number`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reconOrder{}
	for rows.Next() {
		var o reconOrder
		if err := rows.Scan(&o.BusinessDate, &o.OrderNumber, &o.OpenedAt,
			&o.AmountCents, &o.Discount, &o.Voided); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ── GET /reconciliation/queue ──

// ReconQueueHandler is §5's queue read. Manager-only: it is the write surface's
// worklist and stays in Marketing (decision 192).
func ReconQueueHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		ctx := r.Context()
		since := periodStart(r.URL.Query().Get("period"))
		attempts, err := reconLoadAttempts(ctx, d.Pool, since)
		if err != nil {
			slog.Error("marketing: load attempts for queue", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		orders, err := reconLoadOrders(ctx, d.Pool, since)
		if err != nil {
			slog.Error("marketing: load toast orders for queue", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		out := ReconQueueResponse{
			Queue:     []ReconQueueRow{},
			Overrides: []ReconQueueRow{},
			Orphans:   []ReconQueueRow{},
			Unmatched: []ReconQueueRow{},
		}
		for _, a := range attempts {
			switch b := a.bucket(); b {
			case "override":
				out.Overrides = append(out.Overrides, reconRow(a, b, orders))
			case "orphan":
				out.Orphans = append(out.Orphans, reconRow(a, b, orders))
			case "unmatched":
				out.Unmatched = append(out.Unmatched, reconRow(a, b, orders))
			case "matched":
				out.MatchedCount++
			case "declined":
				out.DeclinedCount++
			}
		}
		// Ladder order: overrides → orphans → unmatched (card H3b).
		out.Queue = append(out.Queue, out.Overrides...)
		out.Queue = append(out.Queue, out.Orphans...)
		out.Queue = append(out.Queue, out.Unmatched...)
		writeJSON(w, http.StatusOK, out)
	}
}

// ── GET /reconciliation/declined ──

// ReconDeclinedHandler is §5's declined bucket: reason, note, who, when.
func ReconDeclinedHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if requireManager(w, r) == nil {
			return
		}
		ctx := r.Context()
		since := periodStart(r.URL.Query().Get("period"))
		attempts, err := reconLoadAttempts(ctx, d.Pool, since)
		if err != nil {
			slog.Error("marketing: load attempts for declined bucket", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		out := ReconDeclinedResponse{Declined: []ReconDeclinedRow{}}
		for _, a := range attempts {
			if a.bucket() != "declined" {
				continue
			}
			out.Declined = append(out.Declined, ReconDeclinedRow{
				ReconQueueRow: reconRow(a, "declined", nil),
				Reason:        a.DecisionReason,
				Note:          a.DecisionNote,
				DecidedBy:     derefOr(a.DecidedBy, ""),
				DecidedAt:     derefTime(a.DecidedAt),
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ── the five decision writes ──

type reconDecisionRequest struct {
	OrderNumber *string `json:"order_number"`
	Reason      *string `json:"reason"`
	Note        *string `json:"note"`
}

// reconAttemptRef is the minimum an attempt write needs before it can log a
// decision: does the attempt exist, and on which business date.
type reconAttemptRef struct {
	ID           string
	BusinessDate time.Time
}

func reconLoadAttemptRef(ctx context.Context, pool *pgxpool.Pool, id string) (reconAttemptRef, error) {
	var ref reconAttemptRef
	err := pool.QueryRow(ctx,
		`SELECT id::text, pos_business_date FROM scan_attempts_mirror WHERE id = $1`, id).
		Scan(&ref.ID, &ref.BusinessDate)
	return ref, err
}

// ReconDecisionHandler is all five decision kinds behind one implementation —
// the differences are entirely in validation, and three near-identical handlers
// is how two of them drift.
//
// `kind` is one of reconDecisionKinds and is a COMPILE-SITE constant from
// routes.go, never request data.
func ReconDecisionHandler(d Deps, kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := requireManager(w, r)
		if user == nil {
			return
		}
		if !containsString(reconDecisionKinds, kind) {
			// Unreachable from routes.go; a loud 500 beats a silent bad write.
			slog.Error("marketing: ReconDecisionHandler mounted with an unknown kind", "kind", kind)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		ctx := r.Context()
		attemptID := chi.URLParam(r, "attempt_id")

		var in reconDecisionRequest
		if err := decodeBody(r, &in); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_payload")
			return
		}

		ref, err := reconLoadAttemptRef(ctx, d.Pool, attemptID)
		if errors.Is(err, pgx.ErrNoRows) || isBadUUID(err) {
			writeError(w, http.StatusNotFound, "attempt_not_found")
			return
		}
		if err != nil {
			slog.Error("marketing: load attempt for decision", "error", err, "attempt_id", attemptID)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}

		var orderNumber, reason, note *string
		note = trimmedOrNil(in.Note)

		switch kind {
		case "matched":
			orderNumber = trimmedOrNil(in.OrderNumber)
			if orderNumber == nil {
				writeError(w, http.StatusBadRequest, "order_number_required")
				return
			}
			// §5: `409 order_not_found` if it is not in toast_orders. The check
			// is on (business_date, order_number) because that pair is
			// toast_orders' primary key AND the join the matched bucket uses —
			// a 200 here that the queue would then read as still-unmatched is
			// exactly the lie this status code exists to prevent.
			var exists bool
			if err := d.Pool.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM toast_orders
				                 WHERE business_date = $1 AND order_number = $2)`,
				ref.BusinessDate, *orderNumber).Scan(&exists); err != nil {
				slog.Error("marketing: check toast order", "error", err)
				writeError(w, http.StatusInternalServerError, "internal_error")
				return
			}
			if !exists {
				writeErrorWith(w, http.StatusConflict, "order_not_found", map[string]any{
					"order_number":  *orderNumber,
					"business_date": ref.BusinessDate.Format("2006-01-02"),
				})
				return
			}
		case "declined":
			reason = trimmedOrNil(in.Reason)
			if reason == nil {
				writeError(w, http.StatusBadRequest, "reason_required")
				return
			}
			if !containsString(reconDeclineReasons, *reason) {
				writeErrorWith(w, http.StatusBadRequest, "bad_reason", map[string]any{
					"reasons": reconDeclineReasons,
				})
				return
			}
			// §5: `400 note_required` when reason="other" and the note is
			// empty. trimmedOrNil already collapsed a whitespace-only note to
			// nil — "   " is not an explanation.
			if *reason == "other" && note == nil {
				writeError(w, http.StatusBadRequest, "note_required")
				return
			}
		}

		var logged ReconDecisionRef
		err = d.Pool.QueryRow(ctx, `
			INSERT INTO reconciliation_decisions
			  (attempt_id, decision, order_number, reason, note, decided_by)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING decision, reason, note, decided_at`,
			ref.ID, kind, orderNumber, reason, note, user.ID).
			Scan(&logged.Decision, &logged.Reason, &logged.Note, &logged.DecidedAt)
		if err != nil {
			slog.Error("marketing: log reconciliation decision", "error", err,
				"attempt_id", attemptID, "kind", kind)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		logged.DecidedBy = user.DisplayName

		// Re-read the one attempt so the response reports the bucket the queue
		// will actually put it in — derived by the same bucket() every other
		// reader uses, never guessed from `kind`.
		bucket := ""
		if attempts, err := reconLoadAttempts(ctx, d.Pool, nil); err == nil {
			for _, a := range attempts {
				if a.ID == ref.ID {
					bucket = a.bucket()
				}
			}
		}
		writeJSON(w, http.StatusOK, ReconDecisionResponse{
			AttemptID: ref.ID, Bucket: bucket, Decision: logged,
		})
	}
}

// ── small helpers ──

func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

func derefInt(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
