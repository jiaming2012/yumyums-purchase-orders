package marketing

// stats.go — card H3b, run 20261002. The numbers a manager reads.
//
// # The one rule this file exists to keep
//
// Handoff §5's metric-definitions paragraph is BINDING, and it is binding for
// one reason: Σ(by campaign) == Σ(by channel) == Σ(by item) == overview, for
// scans, signups, redeemed, revenue and discount. A Stats tab that shows three
// totals which disagree is worse than no Stats tab. TestSlicesReconcileToOverview
// is that invariant, seeded from spike 01 `slices-reconcile`'s own fixture.
//
// The way it is kept here is structural, not arithmetical: ONE loader
// (reconLoadAttempts, shared with the queue), ONE accumulator (statsAgg), and
// every slice a PARTITION of the same two input slices — accepted attempts, and
// qr_codes with their scan/signup counts. Each attempt lands in exactly one row
// of each dimension; each code lands in exactly one row of each dimension.
// Nothing is computed twice, so nothing can drift. The spike's run-1 red is the
// cautionary tale: choosing the discount basis per GROUP made a fully-matched
// slice disagree with a partly-matched overview by exactly one row's
// actual−implied gap.
//
// # Decision 190 as refined (operator-signed 2026-10-01): discount is PER ROW
//
// For each accepted attempt: the matched Toast order's ACTUAL discount where
// there is one, the campaign's FACE VALUE where there is not. Summed row by row,
// on every slice and the overview alike. `discount_basis` is a LABEL
// (actual / implied / mixed), NEVER a choice of arithmetic. The implied total is
// carried beside it (`discount_implied_cents`) so a row can render
// "implied −$X · actual −$Y" where they differ.
//
// # 🛑 All money is integer cents. There is no float64 in any money path.
//
// `per_dollar` is the only ratio on the wire and it is computed by INTEGER
// rounding to hundredths before the single float64 division, so a malformed
// input cannot become a confident $0.00 (bugs.md: fmtMoney's `Number(n)||0`
// rendered $0.00 for a malformed price and, because `NaN > 0.01` is false, the
// mismatch check then said "Amounts match. Ready to confirm." on a receipt that
// did not balance). It is `null` when discount is 0 — never 0.00, never Inf,
// never NaN. `orphan_rate` is null when there are no accepted attempts: a rate
// with no denominator is not 0%.
//
// # 🛑 "Unknown" must never render as a confident zero
//
// Three places where a number is genuinely not computable, each stated on the
// wire rather than silently zeroed:
//
//	signups      — needs card H5's `subscribers` (migration 0085). Absent →
//	               literal 0 WITH `signups_basis:"unavailable"`, so a consumer
//	               can tell it from a real zero. Present → the real count with
//	               `signups_basis:"subscribers"`. Never an error (the card's own
//	               instruction), never an omitted key.
//	codes_sent   — same table family (`subscriber_events`), same treatment via
//	               `codes_sent_basis`.
//	discount     — an accepted attempt whose CAMPAIGN cannot be resolved has no
//	               face value, so its implied price is unknown. It contributes
//	               nothing to `discount_implied_cents` and is COUNTED in
//	               `discount_unknown_rows`. That count is why the slices can
//	               still reconcile (the row is in exactly one group of each
//	               dimension, contributing 0) without the 0 being mistaken for
//	               "this redemption was free".
//
// `revenue_cents`, `discount_cents` and `net_cents` stay non-pointer ints on
// purpose: after this card they are always COMPUTED, and a period with no
// matched orders really is $0.00 of revenue. The honest-zero question for those
// three is answered by `discount_unknown_rows` + `discount_basis`, not by a
// pointer.
//
// # Attribution (decision 189, first-touch)
//
// §5: "by-channel and by-item attribute each redemption to the subscriber's
// first-touch code". HQ has no attempt→subscriber link today, so the attempt's
// OWN code is its first-touch code — which is the card's stated fallback ("else
// the scanned code's own channel/item"). An attempt with no resolvable code
// lands in the `direct` row of the channel and item dimensions, exactly as the
// spike's r7 does. An attempt with no resolvable CAMPAIGN lands in
// `unattributed` in the campaign dimension; see the card's report for why that
// is the live state today.

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Fallback group keys. They are STABLE WIRE VALUES — card H4/H5's UI renders
// labels off them.
const (
	// StatsDirectKey is a redemption with no resolvable QR code: it cannot be
	// attributed to a channel or an item.
	StatsDirectKey = "direct"
	// StatsAnyItemKey is §5's "Any item" row — campaigns and codes with
	// item_id NULL.
	StatsAnyItemKey = "any"
	// StatsUnattributedKey is a redemption with no resolvable CAMPAIGN.
	StatsUnattributedKey = "unattributed"
)

// Basis labels for the two metrics that depend on a table this card does not own.
const (
	StatsBasisSubscribers      = "subscribers"
	StatsBasisSubscriberEvents = "subscriber_events"
	StatsBasisUnavailable      = "unavailable"
)

// StatsDims are the slice dimensions §5 names, plus the two drill-in forms.
var StatsDims = []string{"campaign", "channel", "item", "code"}

// ── wire shapes (card H4 `stats-tab-ui` is built against these json tags) ──

// StatsFunnel is §5's overview funnel.
type StatsFunnel struct {
	Scans     int `json:"scans"`
	Signups   int `json:"signups"`
	CodesSent int `json:"codes_sent"`
	Redeemed  int `json:"redeemed"`
}

// StatsReconciliation is the health card.
//
// The three counts deliberately do NOT partition the accepted attempts:
// `matched` means "joined to a Toast order" (the same definition revenue uses),
// while `open` means "still needs a human". A matched offline override is both,
// because it is both. OrphanRate is null when there is no denominator.
type StatsReconciliation struct {
	Matched    int      `json:"matched"`
	Open       int      `json:"open"`
	Declined   int      `json:"declined"`
	OrphanRate *float64 `json:"orphan_rate"`
	Threshold  float64  `json:"threshold"`
}

// StatsNeedsLook is the "needs a look" banner's three bucket counts.
type StatsNeedsLook struct {
	Overrides int `json:"overrides"`
	Orphans   int `json:"orphans"`
	Unmatched int `json:"unmatched"`
}

// StatsOverviewResponse is §5's `GET /stats/overview` — and, byte for byte,
// `GET /api/v1/bi/campaigns/overview` (decision 192).
type StatsOverviewResponse struct {
	Period         string              `json:"period"`
	Funnel         StatsFunnel         `json:"funnel"`
	Money          moneyDTO            `json:"money"`
	Reconciliation StatsReconciliation `json:"reconciliation"`
	NeedsLook      StatsNeedsLook      `json:"needs_look"`
	// SignupsBasis / CodesSentBasis say WHERE the figure came from, so a 0 is
	// never ambiguous. See this file's header.
	SignupsBasis   string `json:"signups_basis"`
	CodesSentBasis string `json:"codes_sent_basis"`
}

// StatsRow is one slice row. The money fields are FLATTENED onto the row (§5:
// "rows with scans,signups,redeemed,revenue_cents,discount_cents,
// discount_basis,net_cents,per_dollar"), while the overview nests them under
// `money` — that asymmetry is §5's, kept verbatim so UI code written against
// the contract works.
type StatsRow struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Scans    int    `json:"scans"`
	Signups  int    `json:"signups"`
	Redeemed int    `json:"redeemed"`
	moneyDTO
}

// StatsByResponse is §5's `GET /stats/by` — and `GET /api/v1/bi/campaigns/by`.
type StatsByResponse struct {
	Dim          string     `json:"dim"`
	Period       string     `json:"period"`
	Rows         []StatsRow `json:"rows"`
	Totals       StatsRow   `json:"totals"`
	SignupsBasis string     `json:"signups_basis"`
}

// ── the accumulator ──

// statsAgg is the ONE place the arithmetic lives. Every row, every total and the
// overview are this struct; that is what makes them reconcile.
type statsAgg struct {
	scans, signups, redeemed int
	revenue                  int
	discount                 int // the PER-ROW total (decision 190)
	implied                  int // Σ face value over every accepted row that has one
	actual                   int // Σ actual discount over matched rows
	unknownRows              int // accepted rows with no resolvable face value
	matchedRows              int
	matchedAmountSum         int
}

func (g *statsAgg) addAttempt(a reconAttempt) {
	g.redeemed++
	if a.matched() {
		g.matchedRows++
		g.revenue += *a.OrderAmountCents
		g.matchedAmountSum += *a.OrderAmountCents
		g.discount += derefInt(a.OrderDiscountCents)
		g.actual += derefInt(a.OrderDiscountCents)
	} else if a.FaceValueCents != nil {
		g.discount += *a.FaceValueCents
	}
	if a.FaceValueCents != nil {
		g.implied += *a.FaceValueCents
	} else {
		g.unknownRows++
	}
}

func (g *statsAgg) addCode(c statsCode) {
	g.scans += c.Scans
	g.signups += c.Signups
}

// money renders the accumulator as the wire block. avgWithout is the
// period-level "orders with no offer attached" baseline, which is only
// meaningful at period scope — slice rows pass nil, and nil means "no opinion",
// never 0.
func (g statsAgg) money(avgWithout *int) moneyDTO {
	m := moneyDTO{
		RevenueCents:         g.revenue,
		DiscountCents:        g.discount,
		DiscountBasis:        statsDiscountBasis(g.redeemed, g.matchedRows),
		NetCents:             g.revenue - g.discount,
		PerDollar:            statsPerDollar(g.revenue, g.discount),
		DiscountImpliedCents: g.implied,
		DiscountActualCents:  g.actual,
		DiscountUnknownRows:  g.unknownRows,
		AvgOrderCentsWithout: avgWithout,
	}
	if g.matchedRows > 0 {
		// Integer division, integer cents. A truncated average cent is the
		// honest answer; a float64 here would be the first crack in the rule.
		v := g.matchedAmountSum / g.matchedRows
		m.AvgOrderCentsWith = &v
	}
	return m
}

func (g statsAgg) row(key, label string) StatsRow {
	return StatsRow{
		Key: key, Label: label,
		Scans: g.scans, Signups: g.signups, Redeemed: g.redeemed,
		moneyDTO: g.money(nil),
	}
}

// statsDiscountBasis is decision 190's LABEL. It never changes the arithmetic.
// An empty set is "implied", which is also card H1's zero shape — a period with
// no matched orders at all has only implied discount to report.
func statsDiscountBasis(rows, matched int) string {
	switch {
	case rows == 0 || matched == 0:
		return "implied"
	case matched == rows:
		return "actual"
	default:
		return "mixed"
	}
}

// statsPerDollar is revenue ÷ discount to 2dp, or nil when discount is 0.
//
// The rounding is INTEGER (half-up on hundredths) and the single float64
// division happens only to put a 2dp value on the wire. No money value is ever
// a float.
func statsPerDollar(revenueCents, discountCents int) *float64 {
	if discountCents == 0 {
		return nil
	}
	hundredths := (revenueCents*100 + discountCents/2) / discountCents
	v := float64(hundredths) / 100
	return &v
}

// statsRate is a 4dp rate by the same integer rounding, or nil with no
// denominator. 4dp because the UI renders a percentage with one decimal and a
// 10% threshold marker; rounding at the edge of what is displayed is what keeps
// two renderings of one number from disagreeing.
func statsRate(numerator, denominator int) *float64 {
	if denominator == 0 {
		return nil
	}
	tenThousandths := (numerator*10000 + denominator/2) / denominator
	v := float64(tenThousandths) / 10000
	return &v
}

// ── input loading ──

// statsCode is one qr_codes row with its period scan and signup counts, and the
// campaign / channel / item it attributes to.
type statsCode struct {
	Short        string
	CampaignID   string
	CampaignName string
	Channel      string
	ChannelLabel *string
	ItemID       *string
	ItemName     *string
	Scans        int
	Signups      int
}

// statsData is everything the engine needs, read once per request.
type statsData struct {
	attempts       []reconAttempt
	orders         []reconOrder
	codes          []statsCode
	codesSent      int
	signupsBasis   string
	codesSentBasis string
}

func statsLoad(ctx context.Context, pool *pgxpool.Pool, since *time.Time) (statsData, error) {
	var out statsData
	var err error
	if out.attempts, err = reconLoadAttempts(ctx, pool, since); err != nil {
		return out, err
	}
	if out.orders, err = reconLoadOrders(ctx, pool, since); err != nil {
		return out, err
	}

	// Card H5's tables may not exist yet. `to_regclass` is UNQUALIFIED so it
	// resolves through the connection's search_path — this tree runs the same
	// migrations under `public` in test and `production` in prod.
	var hasSubscribers, hasEvents bool
	if err := pool.QueryRow(ctx,
		`SELECT to_regclass('subscribers') IS NOT NULL,
		        to_regclass('subscriber_events') IS NOT NULL`).
		Scan(&hasSubscribers, &hasEvents); err != nil {
		return out, err
	}
	out.signupsBasis = StatsBasisUnavailable
	out.codesSentBasis = StatsBasisUnavailable
	if hasSubscribers {
		out.signupsBasis = StatsBasisSubscribers
	}
	if hasEvents {
		out.codesSentBasis = StatsBasisSubscriberEvents
	}

	rows, err := pool.Query(ctx, `
		SELECT k.short, cam.id::text, cam.name, k.channel, k.channel_label,
		       COALESCE(k.item_id, cam.item_id)::text, mi.name,
		       (SELECT count(*) FROM qr_scans s
		         WHERE s.short = k.short
		           AND ($1::timestamptz IS NULL OR s.scanned_at >= $1))::int
		FROM qr_codes k
		JOIN campaigns_admin cam ON cam.id = k.campaign_id
		LEFT JOIN menu_items mi ON mi.id = COALESCE(k.item_id, cam.item_id)
		ORDER BY k.created_at, k.short`, since)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	out.codes = []statsCode{}
	for rows.Next() {
		var c statsCode
		if err := rows.Scan(&c.Short, &c.CampaignID, &c.CampaignName, &c.Channel,
			&c.ChannelLabel, &c.ItemID, &c.ItemName, &c.Scans); err != nil {
			return out, err
		}
		out.codes = append(out.codes, c)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}

	if hasSubscribers {
		// §5: signups = subscribers whose source_short RESOLVES, by joined_at.
		// A subscriber with no first-touch code is a real subscriber but not an
		// attributable signup, which is exactly what makes Σ over the slices
		// equal the overview.
		sRows, err := pool.Query(ctx, `
			SELECT source_short, count(*)::int
			FROM subscribers
			WHERE source_short IS NOT NULL
			  AND ($1::timestamptz IS NULL OR joined_at >= $1)
			GROUP BY source_short`, since)
		if err != nil {
			return out, err
		}
		defer sRows.Close()
		byShort := map[string]int{}
		for sRows.Next() {
			var short string
			var n int
			if err := sRows.Scan(&short, &n); err != nil {
				return out, err
			}
			byShort[short] = n
		}
		if err := sRows.Err(); err != nil {
			return out, err
		}
		for i := range out.codes {
			out.codes[i].Signups = byShort[out.codes[i].Short]
		}
	}

	if hasEvents {
		if err := pool.QueryRow(ctx, `
			SELECT count(*)::int FROM subscriber_events
			WHERE kind = 'code_sent'
			  AND ($1::timestamptz IS NULL OR at >= $1)`, since).Scan(&out.codesSent); err != nil {
			return out, err
		}
	}
	return out, nil
}

// statsAvgOrderWithout is the design's "average check WITHOUT the offer": the
// mean amount over Toast orders in the period that no accepted attempt matched.
// nil when there are none — a baseline with no orders behind it is no opinion,
// not $0.00.
func statsAvgOrderWithout(attempts []reconAttempt, orders []reconOrder) *int {
	type key struct {
		date  string
		order string
	}
	matched := map[key]bool{}
	for _, a := range attempts {
		if a.matched() && a.OrderNumber != nil {
			matched[key{a.BusinessDate.Format("2006-01-02"), *a.OrderNumber}] = true
		}
	}
	sum, n := 0, 0
	for _, o := range orders {
		if matched[key{o.BusinessDate.Format("2006-01-02"), o.OrderNumber}] {
			continue
		}
		sum += o.AmountCents
		n++
	}
	if n == 0 {
		return nil
	}
	v := sum / n
	return &v
}

// ── grouping ──

// statsAttemptKey is the (key, label) an attempt contributes to in `dim`.
func statsAttemptKey(a reconAttempt, dim string) (string, string) {
	switch dim {
	case "campaign":
		if a.CampaignID != nil {
			return *a.CampaignID, derefOr(a.CampaignName, *a.CampaignID)
		}
		return StatsUnattributedKey, "Unattributed"
	case "channel":
		if a.CodeID == nil || a.Channel == nil {
			return StatsDirectKey, "Direct"
		}
		return *a.Channel, statsChannelLabel(*a.Channel, a.ChannelLabel)
	case "item":
		if a.CodeID == nil {
			return StatsDirectKey, "Direct"
		}
		if a.ItemID != nil {
			return *a.ItemID, derefOr(a.ItemName, *a.ItemID)
		}
		return StatsAnyItemKey, "Any item"
	case "code":
		if a.Short == nil {
			return StatsDirectKey, "Direct"
		}
		return *a.Short, *a.Short
	}
	return "", ""
}

// statsCodeKey is the (key, label) a code's scans and signups contribute to.
// It MUST agree with statsAttemptKey for every dimension, or a slice stops
// reconciling.
func statsCodeKey(c statsCode, dim string) (string, string) {
	switch dim {
	case "campaign":
		return c.CampaignID, c.CampaignName
	case "channel":
		return c.Channel, statsChannelLabel(c.Channel, c.ChannelLabel)
	case "item":
		if c.ItemID != nil {
			return *c.ItemID, derefOr(c.ItemName, *c.ItemID)
		}
		return StatsAnyItemKey, "Any item"
	case "code":
		return c.Short, c.Short
	}
	return "", ""
}

// statsChannelLabel prefers the operator's own label for channel='other'.
// Two codes on one channel with different labels share a row; the first label
// read wins, which is why channel_label is only ever set for 'other'.
func statsChannelLabel(channel string, label *string) string {
	if label != nil && *label != "" {
		return *label
	}
	return channel
}

// statsBuild partitions the inputs into rows for `dim`, plus the totals.
func statsBuild(dim string, data statsData, filter statsFilter) ([]StatsRow, statsAgg) {
	groups := map[string]*statsAgg{}
	labels := map[string]string{}
	order := []string{}
	touch := func(key, label string) *statsAgg {
		g, ok := groups[key]
		if !ok {
			g = &statsAgg{}
			groups[key] = g
			labels[key] = label
			order = append(order, key)
		}
		return g
	}
	var totals statsAgg

	for _, a := range data.attempts {
		if !filter.keepAttempt(a) {
			continue
		}
		key, label := statsAttemptKey(a, dim)
		touch(key, label).addAttempt(a)
		totals.addAttempt(a)
	}
	for _, c := range data.codes {
		if !filter.keepCode(c) {
			continue
		}
		key, label := statsCodeKey(c, dim)
		touch(key, label).addCode(c)
		totals.addCode(c)
	}

	rows := make([]StatsRow, 0, len(order))
	for _, key := range order {
		rows = append(rows, groups[key].row(key, labels[key]))
	}
	// Deterministic, and the design's order: the slices that cost the most
	// first. Key breaks every tie so two requests never disagree.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Redeemed != b.Redeemed {
			return a.Redeemed > b.Redeemed
		}
		if a.RevenueCents != b.RevenueCents {
			return a.RevenueCents > b.RevenueCents
		}
		if a.Scans != b.Scans {
			return a.Scans > b.Scans
		}
		return a.Key < b.Key
	})
	return rows, totals
}

// statsFilter is §5's two drill-in scopes.
type statsFilter struct {
	CampaignID string
	ItemID     string
}

func (f statsFilter) keepAttempt(a reconAttempt) bool {
	if f.CampaignID != "" && (a.CampaignID == nil || *a.CampaignID != f.CampaignID) {
		return false
	}
	if f.ItemID != "" && (a.ItemID == nil || *a.ItemID != f.ItemID) {
		return false
	}
	return true
}

func (f statsFilter) keepCode(c statsCode) bool {
	if f.CampaignID != "" && c.CampaignID != f.CampaignID {
		return false
	}
	if f.ItemID != "" && (c.ItemID == nil || *c.ItemID != f.ItemID) {
		return false
	}
	return true
}

// ── GET /stats/overview (and /api/v1/bi/campaigns/overview) ──

// statsOverview is the pure computation, so both mount points answer from one
// implementation and cannot drift.
func statsOverview(period string, data statsData) StatsOverviewResponse {
	var all statsAgg
	for _, a := range data.attempts {
		all.addAttempt(a)
	}
	for _, c := range data.codes {
		all.addCode(c)
	}

	var needs StatsNeedsLook
	matched, declined, orphanNumerator := 0, 0, 0
	for _, a := range data.attempts {
		switch a.bucket() {
		case "override":
			needs.Overrides++
		case "orphan":
			needs.Orphans++
		case "unmatched":
			needs.Unmatched++
		case "declined":
			declined++
		}
		if a.matched() {
			matched++
		}
		if a.countsInOrphanRate() {
			orphanNumerator++
		}
	}

	return StatsOverviewResponse{
		Period: statsPeriodLabel(period),
		Funnel: StatsFunnel{
			Scans:     all.scans,
			Signups:   all.signups,
			CodesSent: data.codesSent,
			Redeemed:  all.redeemed,
		},
		Money: all.money(statsAvgOrderWithout(data.attempts, data.orders)),
		Reconciliation: StatsReconciliation{
			Matched:    matched,
			Open:       needs.Overrides + needs.Orphans + needs.Unmatched,
			Declined:   declined,
			OrphanRate: statsRate(orphanNumerator, len(data.attempts)),
			Threshold:  ReconOrphanThreshold,
		},
		NeedsLook:      needs,
		SignupsBasis:   data.signupsBasis,
		CodesSentBasis: data.codesSentBasis,
	}
}

// statsPeriodLabel echoes the period back exactly as periodStart interpreted it,
// so a UI never renders a window the server did not use.
func statsPeriodLabel(period string) string {
	switch period {
	case "7d", "90d", "all":
		return period
	default:
		return "30d"
	}
}

// StatsOverviewHandler is §5's `GET /stats/overview`.
//
// 🛑 managerTier is the ONE difference between decision 192's two mount points:
// true for /api/v1/marketing/stats/* (marketing grant + manager tier), FALSE for
// /api/v1/bi/campaigns/* (`bi` grant alone — anyone holding BI sees campaign
// money, the consequence the operator accepted). The BODY is byte-identical
// because it comes from the same statsOverview call.
func StatsOverviewHandler(d Deps, managerTier bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if managerTier && requireManager(w, r) == nil {
			return
		}
		if d.Pool == nil {
			// Only reachable if the MountReports seam ran before NewDeps. Loud,
			// never a silent empty report.
			slog.Error("marketing: stats overview requested with no pool; MountReports ran before NewDeps")
			writeError(w, http.StatusServiceUnavailable, "reports_unavailable")
			return
		}
		period := r.URL.Query().Get("period")
		data, err := statsLoad(r.Context(), d.Pool, periodStart(period))
		if err != nil {
			slog.Error("marketing: stats overview", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		writeJSON(w, http.StatusOK, statsOverview(period, data))
	}
}

// ── GET /stats/by (and /api/v1/bi/campaigns/by) ──

// StatsByHandler is §5's `GET /stats/by?dim=…`, including the two drill-ins
// (`dim=channel&campaign_id=`, `dim=code&item_id=`).
//
// An unknown `dim` is a 400, not a fallback. periodStart deliberately tolerates
// a bad period because the UI's period control is a fixed segmented control and
// returning data beats a 400 on a read — but a bad `dim` has no sensible
// default, and silently serving the campaign slice when the caller asked for
// items is a wrong number, not a degraded one.
func StatsByHandler(d Deps, managerTier bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if managerTier && requireManager(w, r) == nil {
			return
		}
		if d.Pool == nil {
			slog.Error("marketing: stats by requested with no pool; MountReports ran before NewDeps")
			writeError(w, http.StatusServiceUnavailable, "reports_unavailable")
			return
		}
		q := r.URL.Query()
		dim := q.Get("dim")
		if dim == "" {
			dim = "campaign"
		}
		if !containsString(StatsDims, dim) {
			writeErrorWith(w, http.StatusBadRequest, "bad_dim", map[string]any{"dims": StatsDims})
			return
		}
		period := q.Get("period")
		data, err := statsLoad(r.Context(), d.Pool, periodStart(period))
		if err != nil {
			slog.Error("marketing: stats by", "error", err, "dim", dim)
			writeError(w, http.StatusInternalServerError, "internal_error")
			return
		}
		filter := statsFilter{CampaignID: q.Get("campaign_id"), ItemID: q.Get("item_id")}
		rows, totals := statsBuild(dim, data, filter)
		writeJSON(w, http.StatusOK, StatsByResponse{
			Dim:          dim,
			Period:       statsPeriodLabel(period),
			Rows:         rows,
			Totals:       totals.row("totals", "Total"),
			SignupsBasis: data.signupsBasis,
		})
	}
}

// ── the campaign list's money (card H1 shipped the zero shape) ──

// statsCampaignMoney is what makes `GET /campaigns`' money block REAL. It
// returns, per campaign id, the funnel's signups/redeemed and the money block —
// the SAME arithmetic the campaign slice uses, so a manager reading the list and
// the Stats tab reads one number twice, not two numbers.
//
// avg_order_cents_without is the PERIOD baseline (orders with no offer
// attached), identical for every campaign because that is what it means: the
// average check when the offer was not used.
func statsCampaignMoney(ctx context.Context, pool *pgxpool.Pool, since *time.Time) (map[string]StatsRow, *int, error) {
	data, err := statsLoad(ctx, pool, since)
	if err != nil {
		return nil, nil, err
	}
	rows, _ := statsBuild("campaign", data, statsFilter{})
	without := statsAvgOrderWithout(data.attempts, data.orders)
	out := make(map[string]StatsRow, len(rows))
	for _, row := range rows {
		row.AvgOrderCentsWithout = without
		out[row.Key] = row
	}
	// `without` is returned alongside so a campaign with NO redemptions gets the
	// same baseline as one with some. It is a period fact, not a campaign fact —
	// handing one campaign the number and another a null would read as "we have
	// no baseline for this campaign", which is not what is true.
	return out, without, nil
}

// statsApplyCampaignMoney fills one campaign's funnel and money from the map,
// leaving card H1's own `funnel.scans` alone.
//
// A campaign the map does not know (no codes, no redemptions) gets the computed
// EMPTY money — zeroMoney()'s shape with the new keys — which is a true zero,
// not an unknown: it really has no redemptions.
func statsApplyCampaignMoney(c *campaignDTO, byCampaign map[string]StatsRow, without *int) {
	row, ok := byCampaign[c.ID]
	if !ok {
		var empty statsAgg
		m := empty.money(without)
		c.Money = m
		return
	}
	c.Funnel.Signups = row.Signups
	c.Funnel.Redeemed = row.Redeemed
	c.Money = row.moneyDTO
}
