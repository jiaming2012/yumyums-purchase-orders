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
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderDetailsFilename is the per-date file this leg fetches. It sits beside
// ItemSelectionDetails.csv in /<ExportID>/<YYYYMMDD>/.
const OrderDetailsFilename = "OrderDetails.csv"

// orderTimeZone is the wall clock Toast's export is written in: the
// restaurant's own, which is the same America/Chicago the purchasing cutoff and
// the recipes drift check already use. The report prints "09/28/26 11:42 AM"
// with no offset, so SOMETHING has to supply one, and the only correct answer is
// the business's timezone — `opened_at` is compared against a device's real
// `scanned_at` instant by H3b's ±30-minute suggestion, so an hour of drift here
// is a wrong suggestion there.
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

// parseOrderDetails reads an OrderDetails.csv stream into one OrderRow per CSV
// row. It does not deduplicate: `(business_date, order_number)` uniqueness is
// the DATABASE's statement (migration 0084's primary key), and UpsertOrders is
// what makes a second arrival of the same report idempotent. Spike 02 enumerated
// the duplicate set over 77 real orders and found it empty.
func parseOrderDetails(r io.Reader) ([]OrderRow, error) {
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
		out = append(out, OrderRow{
			BusinessDate:  time.Date(y, m, d, 0, 0, 0, 0, loc),
			OrderNumber:   num,
			OrderID:       get("Order Id"),
			OpenedAt:      opened,
			ClosedAt:      closed,
			AmountCents:   amount,
			DiscountCents: discount,
			TotalCents:    total,
			Voided:        parseBoolish(get("Voided")),
			OrderSource:   source,
		})
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

// parseCents turns the report's dollar string into integer cents. Tolerates
// "$", thousands separators, parentheses-negatives and an empty cell (0, which
// is what an order with no discount prints).
//
// The rounding is math/round-half-away-from-zero via +/-0.5 truncation, which is
// what the spike's python round() produced over the real sample (Σ discount
// 341¢ / Σ amount 145,473¢).
func parseCents(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		neg = true
		s = strings.TrimSuffix(strings.TrimPrefix(s, "("), ")")
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "-") {
		neg = !neg
		s = strings.TrimPrefix(s, "-")
	}
	if s == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
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
	for i := range rows {
		if _, err := br.Exec(); err != nil {
			_ = br.Close()
			return 0, fmt.Errorf("toast orders: upsert %s/%s: %w",
				rows[i].BusinessDate.Format("2006-01-02"), rows[i].OrderNumber, err)
		}
	}
	if err := br.Close(); err != nil {
		return 0, fmt.Errorf("toast orders: batch close: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("toast orders: commit: %w", err)
	}
	return len(rows), nil
}

// IngestOrderDetails parses a stream and upserts it. The seam the SFTP leg and
// any future fixture loader share, so both paths land identical rows.
func IngestOrderDetails(ctx context.Context, pool *pgxpool.Pool, r io.Reader) (int, error) {
	rows, err := parseOrderDetails(r)
	if err != nil {
		return 0, err
	}
	return UpsertOrders(ctx, pool, rows)
}
