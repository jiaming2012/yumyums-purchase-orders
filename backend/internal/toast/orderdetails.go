package toast

// orderdetails.go — OrderDetails.csv → toast_orders (card H3a, decision 188).
//
// WHY THIS FILE EXISTS: the reconciliation engine (H3b) joins a counter scan to
// the order it was supposed to discount. The order facts live in
// OrderDetails.csv, which sits in the SAME per-date export directory HQ has been
// syncing ItemSelectionDetails.csv out of since Phase 22 — confirmed by
// read-only listing on 2026-10-01 for 20260928/29/30, beside PaymentDetails.csv
// and ModifiersSelectionDetails.csv. Roadmap `smtp-toast-ingest` is superseded:
// a second ingest path for data we already pull would be two things to break.
//
// WHAT IT DELIBERATELY IS NOT: an aggregator. parseItemSelectionDetails sums
// rows per master_id and drops voided lines (D-06) because sales are a daily
// total. An order is a row; `voided` is a COLUMN here, not a filter, because a
// voided order a customer scanned against is exactly the thing reconciliation
// has to be able to say "that order was voided" about.

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrOrderMoneyFormat is returned for a money cell that is not real decimal
// money — blank, non-finite, exponent-notation or otherwise malformed. Named so
// a caller can tell "this report changed shape" from a transport or DB fault.
var ErrOrderMoneyFormat = errors.New("toast orders: money cell is not decimal money")

// moneyPattern is what a Toast money cell looks like once "$", thousands
// separators, whitespace and a single leading sign are stripped: plain decimal
// digits with the point in any position. Deliberately NO exponent, NO hex, NO
// "NaN"/"Inf" — all of which strconv.ParseFloat would otherwise accept.
var moneyPattern = regexp.MustCompile(`^(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)$`)

// maxOrderDollars bounds a single order so an arbitrarily long digit string
// cannot reach int() as +Inf.
const maxOrderDollars = 1e9

// OrderDetailsFilename is the per-date file this leg fetches. It sits beside
// ItemSelectionDetails.csv in /<ExportID>/<YYYYMMDD>/.
const OrderDetailsFilename = "OrderDetails.csv"

// orderTimeZone is the wall clock this parser reads Toast's `Opened` / `Closed`
// in. The report prints "09/28/26 11:42 AM" with NO offset, so something has to
// supply one, and it matters: `opened_at` is compared against a device's real
// `scanned_at` instant by H3b's ±30-minute suggestion, so an hour of drift here
// is a wrong suggestion there.
//
// 🛑 WHAT IS AND IS NOT ESTABLISHED HERE — read this before citing it.
//
//  1. The zone Toast actually writes `Opened` in is **UNCONFIRMED**. Spike 02
//     parsed the real sample NAIVE, with no zone at all, so it measured the
//     digits and not their offset. Nothing in this repo has established it.
//  2. **This constant disagrees with the repo's own app timezone.**
//     `users.DefaultTimezone` is `America/New_York` (ledger T-26 decision 83,
//     migration `0072_app_timezone_new_york.sql`, which moved the recipes drift
//     scheduler OFF Chicago on purpose). `purchasing/service.go`,
//     `recipes/scheduler.go`, `recipes/cost.go` and `inventory/handler.go` all
//     read that one constant. This file deliberately does not.
//  3. An earlier version of this comment claimed Chicago was "the same
//     America/Chicago the purchasing cutoff and the recipes drift check already
//     use". That was FALSE — those read New York — and the false claim is what
//     made the choice look settled when it is not. G6, run 20261002.
//
// TODO(h3a/F1): the zone is PARKED for an operator decision against a real
// export sample (routed to the decisions log by the run 20261002 orchestrator).
// Resolving it is a one-line change here plus a re-derivation of
// `business_date`; do NOT "tidy" this to users.DefaultTimezone without that
// decision, because a wrong zone silently moves orders between business days.
const orderTimeZone = "America/Chicago"

// orderTimeLayouts are the shapes seen in the real sample plus their 4-digit-year
// and with-seconds variants. Go's "1"/"2" match one OR two digits, so a
// zero-padded month and a bare one both parse against the same layout.
var orderTimeLayouts = []string{
	"1/2/06 3:04 PM",
	"1/2/2006 3:04 PM",
	"1/2/06 3:04:05 PM",
	"1/2/2006 3:04:05 PM",
	"1/2/06 15:04",
	"1/2/2006 15:04",
	"1/2/06 15:04:05",
	"1/2/2006 15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

// OrderRow is one parsed Toast order, in the shape toast_orders stores.
// Money is integer cents throughout — the report prints dollars, this is the
// only place the conversion happens.
type OrderRow struct {
	BusinessDate  time.Time // date only, in orderTimeZone; derived from Opened
	OrderNumber   string    // Toast "Order #" — TEXT, matched exactly (digits only, 1-4 long in the real sample)
	OrderID       string    // Toast "Order Id"
	OpenedAt      time.Time
	ClosedAt      *time.Time // nil when "Closed" is blank or the column is absent
	AmountCents   int
	DiscountCents int
	TotalCents    int
	Voided        bool
	OrderSource   *string // nil when "Order Source" is blank or the column is absent
}

// Required columns. A missing one is a FAIL, not a zero: Toast's schema is
// stable, so an absent column means the report changed shape and silently
// landing orders with amount_cents=0 would poison every money figure H3b
// computes off them. Same posture as parseItemSelectionDetails.
var orderRequiredColumns = []string{"Order #", "Order Id", "Opened", "Amount", "Discount Amount", "Total", "Voided"}

// Optional columns, absent-tolerant because the mirror column they feed is
// nullable in §4's DDL anyway:
//
//	"Closed"       — handoff §3 lists it on the export; spike 02 never read it,
//	                 so tonight it is parsed-if-present and NULL if not. That is
//	                 the stated gap, not a park (the PARK note is explicit: a
//	                 missing COLUMN falls back, only a missing SOURCE parks).
//	"Order Source" — spike 02 read it with .get(), i.e. already optional there.
var orderOptionalColumns = []string{"Closed", "Order Source"}

// parseOrderDetails reads an OrderDetails.csv stream into one OrderRow per
// DISTINCT (business_date, order_number). `dateDir` is the export directory's
// YYYYMMDD, used only for the disagreement warning below; pass "" when there
// isn't one.
//
// # It DEDUPLICATES, and says so (G6 F3, run 20261002)
//
// It used to return one row per CSV row, on the reasoning that
// `(business_date, order_number)` uniqueness is the DATABASE's statement
// (migration 0084's primary key) and that spike 02 found the duplicate set empty
// over 77 real orders. Both halves are still true, and the conclusion was still
// wrong: when two rows in ONE file share the key, the upsert collapses them
// last-write-wins and one order is simply GONE — while the caller was told
// "2 upserted", because the count was len(rows), i.e. rows PARSED. A silent loss
// reported as a success is the defect class this tree exists to retire.
//
// So: last-write-wins is kept (it is what the database does anyway), but the
// collision is WARNED, naming the key and both order ids, and the returned slice
// now holds exactly what will land — which is what makes the count honest.
//
// # And it warns when the business date disagrees with the export directory
//
// `internal/toast/ingest.go:60` hands parseItemSelectionDetails the EXPORT
// DIRECTORY date; this parser RE-DERIVES the business date from `Opened`. Those
// agree until Toast's business day has a late-night cutoff, at which point an
// order opened 00:30 sits in the previous directory but takes the new calendar
// date — and since Toast order numbers reset per business day, two days' order
// "#7" can then collide on the primary key above. The warning is free (`dateDir`
// is already in hand) and it is the only thing that would surface the
// disagreement before the collision does.
func parseOrderDetails(r io.Reader, dateDir string) ([]OrderRow, error) {
	loc, err := time.LoadLocation(orderTimeZone)
	if err != nil {
		// No tzdata: refuse rather than silently parsing in whatever the host's
		// clock is. A wrong opened_at is a wrong reconciliation suggestion.
		return nil, fmt.Errorf("load %s (no tzdata?): %w", orderTimeZone, err)
	}

	// Strip the UTF-8 BOM at the byte level — csv.Reader cannot tolerate one
	// glued to an opening quote. Same guard parseItemSelectionDetails carries.
	br := bufio.NewReader(r)
	if peek, _ := br.Peek(3); len(peek) >= 3 && peek[0] == 0xEF && peek[1] == 0xBB && peek[2] == 0xBF {
		_, _ = br.Discard(3)
	}

	rdr := csv.NewReader(br)
	// Toast's reports are not rectangular across every export vintage; the
	// per-column lookup below tolerates short rows, so let the reader through.
	rdr.FieldsPerRecord = -1

	headers, err := rdr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	colIdx := map[string]int{}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if i == 0 {
			h = strings.TrimPrefix(h, "\xef\xbb\xbf")
		}
		colIdx[h] = i
	}
	for _, col := range orderRequiredColumns {
		if _, ok := colIdx[col]; !ok {
			return nil, fmt.Errorf("OrderDetails.csv missing required column %q. Found: %v", col, headers)
		}
	}
	for _, col := range orderOptionalColumns {
		if _, ok := colIdx[col]; !ok {
			slog.Warn("toast orders: optional column absent; the matching field stays NULL",
				"column", col, "file", OrderDetailsFilename)
		}
	}

	var out []OrderRow
	// seen maps (business_date, order_number) -> index in out, for F3's
	// intra-file dedupe. warnedDates keeps F2's warning to one line per date.
	seen := map[string]int{}
	warnedDates := map[string]bool{}
	for {
		row, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		get := func(col string) string {
			idx, ok := colIdx[col]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}

		num := get("Order #")
		if num == "" {
			continue // a row with no order number cannot be keyed, matched or reconciled
		}
		if num == "Order #" {
			// Defensive, mirroring the parseItemSelectionDetails guard: a
			// concatenated archive leaves duplicate header rows mid-stream.
			continue
		}

		opened, err := parseOrderTime(get("Opened"), loc)
		if err != nil {
			return nil, fmt.Errorf("order %q: parse Opened %q: %w", num, get("Opened"), err)
		}
		var closed *time.Time
		if s := get("Closed"); s != "" {
			c, cErr := parseOrderTime(s, loc)
			if cErr != nil {
				return nil, fmt.Errorf("order %q: parse Closed %q: %w", num, s, cErr)
			}
			closed = &c
		}

		amount, err := parseCents(get("Amount"))
		if err != nil {
			return nil, fmt.Errorf("order %q: parse Amount %q: %w", num, get("Amount"), err)
		}
		discount, err := parseCents(get("Discount Amount"))
		if err != nil {
			return nil, fmt.Errorf("order %q: parse Discount Amount %q: %w", num, get("Discount Amount"), err)
		}
		total, err := parseCents(get("Total"))
		if err != nil {
			return nil, fmt.Errorf("order %q: parse Total %q: %w", num, get("Total"), err)
		}

		var source *string
		if s := get("Order Source"); s != "" {
			source = &s
		}

		// business_date is Opened's calendar date in the business's own
		// timezone. Toast has no business-date column on this report, and an
		// order opened at 23:50 belongs to the day it was opened — which is also
		// how the spike derived it, so the proven upsert key is the same key.
		y, m, d := opened.Date()
		businessDate := time.Date(y, m, d, 0, 0, 0, 0, loc)
		rec := OrderRow{
			BusinessDate:  businessDate,
			OrderNumber:   num,
			OrderID:       get("Order Id"),
			OpenedAt:      opened,
			ClosedAt:      closed,
			AmountCents:   amount,
			DiscountCents: discount,
			TotalCents:    total,
			Voided:        parseBoolish(get("Voided")),
			OrderSource:   source,
		}

		// F2 — the business date the parser derived vs the directory it came
		// from. Warned ONCE per distinct derived date, not once per row: a
		// 77-order file with a late-night cutoff would otherwise emit 77
		// identical lines and get tuned out.
		bd := businessDate.Format("2006-01-02")
		if dateDir != "" && strings.ReplaceAll(bd, "-", "") != dateDir && !warnedDates[bd] {
			warnedDates[bd] = true
			slog.Warn("toast orders: business_date disagrees with the export dir it came from "+
				"(Toast business-day cutoff? order numbers reset per business day, so this is how "+
				"two days' order numbers collide on the (business_date, order_number) primary key)",
				"business_date", bd, "export_dir", dateDir, "order_number", num,
				"opened_at", opened.Format(time.RFC3339), "zone", orderTimeZone)
		}

		// F3 — a duplicate key WITHIN one file. Last-write-wins, loudly.
		key := bd + "/" + num
		if prev, dup := seen[key]; dup {
			slog.Warn("toast orders: duplicate (business_date, order_number) WITHIN one file — "+
				"the primary key keeps the LAST row and the earlier order is lost",
				"business_date", bd, "order_number", num,
				"dropped_order_id", out[prev].OrderID, "kept_order_id", rec.OrderID,
				"dropped_total_cents", out[prev].TotalCents, "kept_total_cents", rec.TotalCents)
			out[prev] = rec
			continue
		}
		seen[key] = len(out)
		out = append(out, rec)
	}
	return out, nil
}

// parseOrderTime tries every known layout in the business timezone.
func parseOrderTime(s string, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}
	for _, layout := range orderTimeLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("no known layout matches (tried %d)", len(orderTimeLayouts))
}

// parseCents turns the report's dollar string into integer CENTS, and rejects
// anything that is not real decimal money.
//
// Accepted: an optional single leading "-" or a parenthesised negative, "$",
// thousands separators, surrounding whitespace, and a plain decimal with the
// point in any position ("0.00", ".50", "1,000,000", "$1,205.73").
//
// 🛑 REJECTED, each with ErrOrderMoneyFormat (G6 F5, run 20261002) — these were
// all silently accepted before, and every one of them lands a WRONG NUMBER
// rather than a visible failure:
//
//   - "" / "   " — a BLANK cell in a required money column. It used to return 0
//     with no error and no warn, while a missing COLUMN already failed loud.
//     This file's header is explicit that landing amount_cents=0 "would poison
//     every money figure H3b computes", and nothing downstream can tell a real
//     zero from an absent one. A blank now fails exactly as loudly as a missing
//     column. (If a real export turns out to print an empty "Discount Amount"
//     for undiscounted orders, the first live file FAILS LOUDLY and the fix is
//     one line here — which is the correct direction for toast-sync-fail-loud.)
//   - "NaN" / "Inf" / "Infinity" — ParseFloat accepts all of these, and
//     int(NaN*100+0.5) was -9223372036854775808, surfacing only later as an
//     opaque Postgres integer-range error with no mention of the cell.
//   - "1e3" / "1.5e2" — exponent notation silently became 100000 / 15000 cents.
//     Toast does not print money this way; accepting it means accepting a
//     typo'd or corrupted cell as a plausible figure.
//   - "--5", "1.2.3", "0x10", "$" — malformed, and "--5" used to come out as
//     499 cents through double negation and rounding.
//
// The rounding stays round-half-away-from-zero, which is what the spike's
// python round() produced over the real sample (Σ discount 341¢ / Σ amount
// 145,473¢), so real 2-dp money is unchanged: "$1,205.73" → 120573,
// "12.345" → 1235.
func parseCents(s string) (int, error) {
	raw := s
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%w: cell is blank (a required money column must carry a figure; "+
			"0 and absent are not the same number)", ErrOrderMoneyFormat)
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	// Exactly ONE leading sign, consumed here; anything else the pattern rejects.
	if strings.HasPrefix(s, "-") {
		neg = !neg
		s = strings.TrimSpace(strings.TrimPrefix(s, "-"))
	}
	if !moneyPattern.MatchString(s) {
		return 0, fmt.Errorf("%w: %q is not decimal money", ErrOrderMoneyFormat, raw)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q: %v", ErrOrderMoneyFormat, raw, err)
	}
	// The pattern allows any number of digits, so a 40-digit cell still reaches
	// ParseFloat as +Inf and int() of that is undefined. Bound it: a single Toast
	// order above a billion dollars is a corrupt cell, not a sale.
	if math.IsNaN(f) || math.IsInf(f, 0) || f > maxOrderDollars {
		return 0, fmt.Errorf("%w: %q is out of range for an order (max $%.0f)",
			ErrOrderMoneyFormat, raw, maxOrderDollars)
	}
	cents := int(f*100 + 0.5)
	if neg {
		cents = -cents
	}
	return cents, nil
}

// parseBoolish reads Toast's several spellings of a boolean cell.
func parseBoolish(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "t", "1", "yes", "y":
		return true
	}
	return false
}

// UpsertOrders writes rows into toast_orders, keyed on
// (business_date, order_number). The same daily report arrives on every tick of
// the 7-day re-pull window, so DO UPDATE is the whole point: a second arrival
// refreshes the money and the void flag and leaves exactly one row per order.
//
// ingested_at is refreshed on update so "when did HQ last see this order" is
// answerable; the key columns are never touched.
//
// The returned count is the sum of RowsAffected reported by Postgres, NOT
// len(rows) (G6 F3): the number in the log should be the database's statement
// about what landed, not Go's statement about what it tried.
//
// It runs in ONE transaction: a half-loaded report would let H3b compute money
// over a partial day and call it a total.
func UpsertOrders(ctx context.Context, pool *pgxpool.Pool, rows []OrderRow) (int, error) {
	if pool == nil {
		return 0, fmt.Errorf("toast orders: nil pool")
	}
	if len(rows) == 0 {
		return 0, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("toast orders: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch := &pgx.Batch{}
	for _, r := range rows {
		batch.Queue(`
			INSERT INTO toast_orders
			    (business_date, order_number, order_id, opened_at, closed_at,
			     amount_cents, discount_cents, total_cents, voided, order_source)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (business_date, order_number) DO UPDATE SET
			    order_id       = EXCLUDED.order_id,
			    opened_at      = EXCLUDED.opened_at,
			    closed_at      = EXCLUDED.closed_at,
			    amount_cents   = EXCLUDED.amount_cents,
			    discount_cents = EXCLUDED.discount_cents,
			    total_cents    = EXCLUDED.total_cents,
			    voided         = EXCLUDED.voided,
			    order_source   = EXCLUDED.order_source,
			    ingested_at    = now()`,
			r.BusinessDate, r.OrderNumber, r.OrderID, r.OpenedAt, r.ClosedAt,
			r.AmountCents, r.DiscountCents, r.TotalCents, r.Voided, r.OrderSource)
	}
	br := tx.SendBatch(ctx, batch)
	affected := 0
	for i := range rows {
		tag, err := br.Exec()
		if err != nil {
			_ = br.Close()
			return 0, fmt.Errorf("toast orders: upsert %s/%s: %w",
				rows[i].BusinessDate.Format("2006-01-02"), rows[i].OrderNumber, err)
		}
		affected += int(tag.RowsAffected())
	}
	if err := br.Close(); err != nil {
		return 0, fmt.Errorf("toast orders: batch close: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("toast orders: commit: %w", err)
	}
	return affected, nil
}

// IngestOrderDetails parses a stream and upserts it. The seam the SFTP leg and
// any future fixture loader share, so both paths land identical rows.
func IngestOrderDetails(ctx context.Context, pool *pgxpool.Pool, r io.Reader) (int, error) {
	return IngestOrderDetailsForDate(ctx, pool, r, "")
}

// IngestOrderDetailsForDate is the same seam with the EXPORT DIRECTORY date in
// hand, so the parsed business date can be checked against it.
func IngestOrderDetailsForDate(ctx context.Context, pool *pgxpool.Pool, r io.Reader, dateDir string) (int, error) {
	rows, err := parseOrderDetails(r, dateDir)
	if err != nil {
		return 0, err
	}
	return UpsertOrders(ctx, pool, rows)
}
