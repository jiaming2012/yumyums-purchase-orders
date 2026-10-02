package toast

// orderdetails_fixes_test.go — the G6 fix round on card H3a (run 20261002,
// APPROVE-WITH-FINDINGS). Three behavioural findings, each written RED first
// per CLAUDE.md's bug-fix protocol:
//
//	F2  the export-directory date and the Opened-derived business date can
//	    disagree, and nothing said so
//	F3  a duplicate key WITHIN one file silently loses an order, and the log
//	    over-reported rows parsed as rows landed
//	F5  parseCents accepted a blank required money cell (as 0) and the
//	    non-finite / exponent forms this file's own header forbids

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// captureLogs redirects the default slog logger into a buffer for the duration
// of one test. Returns a func that renders what was logged.
//
// The warnings these fixes add are the entire deliverable for F2 — a warn that
// is not asserted is a warn that silently stops firing — so the test has to read
// the log, not just the return values.
func captureLogs(t *testing.T) func() string {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf.String
}

// orderCSV builds a minimal OrderDetails.csv with the required column set.
func orderCSV(rows ...string) string {
	head := "Order Id,Order #,Opened,Closed,Discount Amount,Amount,Total,Voided,Order Source\n"
	return head + strings.Join(rows, "\n") + "\n"
}

// ── F2 ──────────────────────────────────────────────────────────────────────

// TestOrderDetailsWarnsWhenBusinessDateDiffersFromExportDir pins the warning
// that makes the H3a/Phase-22 disagreement visible.
//
// internal/toast/ingest.go:60 hands parseItemSelectionDetails the EXPORT
// DIRECTORY date (`d.Format("2006-01-02")`). orderdetails.go re-derives the
// business date from `Opened`. Those are the same number right up until Toast's
// business day has a late-night cutoff: an order opened 00:30 sits in the
// PREVIOUS directory but takes the NEW calendar date. And because Toast order
// numbers reset per business day, two days' order "#7" can then collide on
// toast_orders' (business_date, order_number) primary key — one of them is lost.
//
// `dateDir` is already in syncOrderDetails' hand. One warn line is the whole
// cost of making that visible instead of silent.
func TestOrderDetailsWarnsWhenBusinessDateDiffersFromExportDir(t *testing.T) {
	pool := setupOrdersDB(t)
	ctx := context.Background()
	logs := captureLogs(t)

	// The export directory says 20260928; the order was opened just after
	// midnight, so Opened's calendar date is the 29th.
	csv := orderCSV(`ord-1,7,09/29/26 12:30 AM,09/29/26 12:48 AM,0.00,12.50,13.53,false,In Store`)

	n, err := IngestOrderDetailsForDate(ctx, pool, strings.NewReader(csv), "20260928")
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if n != 1 {
		t.Fatalf("ingested %d rows, want 1", n)
	}

	out := logs()
	if !strings.Contains(out, "business_date") || !strings.Contains(out, "export dir") {
		t.Fatalf("no business-date/export-dir disagreement warning was logged.\nlogs:\n%s", out)
	}
	if !strings.Contains(out, "2026-09-29") || !strings.Contains(out, "20260928") {
		t.Fatalf("the warning does not name BOTH dates (want 2026-09-29 and 20260928).\nlogs:\n%s", out)
	}
	if !strings.Contains(out, "level=WARN") {
		t.Fatalf("the disagreement was logged below WARN — it has to be visible.\nlogs:\n%s", out)
	}
}

// TestOrderDetailsSilentWhenBusinessDateMatchesExportDir — the warning must not
// cry wolf on the normal case, or it will be tuned out before it ever matters.
func TestOrderDetailsSilentWhenBusinessDateMatchesExportDir(t *testing.T) {
	pool := setupOrdersDB(t)
	ctx := context.Background()
	logs := captureLogs(t)

	csv := orderCSV(`ord-1,7,09/28/26 11:42 AM,09/28/26 11:48 AM,0.00,12.50,13.53,false,In Store`)
	if _, err := IngestOrderDetailsForDate(ctx, pool, strings.NewReader(csv), "20260928"); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if out := logs(); strings.Contains(out, "export dir") {
		t.Fatalf("warned on a MATCHING date — the warning must stay quiet on the normal case.\nlogs:\n%s", out)
	}
}

// ── F3 ──────────────────────────────────────────────────────────────────────

// TestOrderDetailsIntraFileDuplicateIsWarnedAndCountedHonestly pins G6's probe:
// one file carrying two rows with the same (business_date, order_number).
//
// Before the fix IngestOrderDetails returned n=2 while the table held 1 row —
// last-write-wins on the primary key, with the LOSS reported as a success,
// because UpsertOrders returned len(rows) (rows PARSED) rather than rows LANDED.
//
// Spike 02 measured no duplicates over 77 real orders, so this is bounded today.
// F2's mechanism is exactly what creates it.
func TestOrderDetailsIntraFileDuplicateIsWarnedAndCountedHonestly(t *testing.T) {
	pool := setupOrdersDB(t)
	ctx := context.Background()
	logs := captureLogs(t)

	csv := orderCSV(
		`ord-A,7,09/28/26 11:42 AM,09/28/26 11:48 AM,0.00,12.50,13.53,false,In Store`,
		`ord-B,7,09/28/26 6:10 PM,09/28/26 6:30 PM,1.00,40.00,42.00,false,Online`,
	)

	n, err := IngestOrderDetailsForDate(ctx, pool, strings.NewReader(csv), "20260928")
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}

	var rows int64
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM toast_orders`).Scan(&rows); err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 1 {
		t.Fatalf("the primary key should collapse these to %d row, got %d", 1, rows)
	}
	if int64(n) != rows {
		t.Fatalf("the count is dishonest: reported %d rows upserted, the table holds %d. "+
			"The log must report rows LANDED, not rows parsed", n, rows)
	}

	out := logs()
	if !strings.Contains(out, "duplicate") {
		t.Fatalf("an intra-file duplicate key was not warned about — an order was silently lost.\nlogs:\n%s", out)
	}
	if !strings.Contains(out, "level=WARN") {
		t.Fatalf("the duplicate was logged below WARN.\nlogs:\n%s", out)
	}

	// Last-write-wins is the behaviour; what matters is that it is VISIBLE.
	// Assert which row survived so a future change to the rule is a test change.
	var orderID string
	if err := pool.QueryRow(ctx, `SELECT order_id FROM toast_orders`).Scan(&orderID); err != nil {
		t.Fatalf("read surviving row: %v", err)
	}
	if orderID != "ord-B" {
		t.Fatalf("surviving row is %q, want the LAST one in the file (ord-B)", orderID)
	}
}

// ── F5 ──────────────────────────────────────────────────────────────────────

// TestParseCentsRejectsBlankAndNonDecimal closes the two doors parseCents left
// open, both of which contradict this file's own header: a blank cell in a
// REQUIRED money column became 0 with no error and no warn (while a missing
// COLUMN already failed loud), and "NaN" / "Inf" / "1e3" were accepted —
// "NaN" landing as -9223372036854775808, which surfaced only later as an opaque
// database range error.
func TestParseCentsRejectsBlankAndNonDecimal(t *testing.T) {
	bad := []string{"", "   ", "NaN", "nan", "Inf", "-Inf", "+Inf", "Infinity", "1e3", "1E3", "1.5e2", "abc", "$", "--5", "1.2.3", "0x10"}
	for _, in := range bad {
		if got, err := parseCents(in); err == nil {
			t.Errorf("parseCents(%q) = %d with NO error; it must be rejected", in, got)
		} else if !errors.Is(err, ErrOrderMoneyFormat) {
			t.Errorf("parseCents(%q) errored with %v; want ErrOrderMoneyFormat", in, err)
		}
	}

	// Real 2-dp money was already correct and must stay correct.
	good := map[string]int{
		"0.00":      0,
		"12.50":     1250,
		"24.00":     2400,
		"$1,205.73": 120573,
		"$1.41":     141,
		"12.345":    1235, // round-half-away-from-zero, as the spike measured
		"-5":        -500,
		"(2.50)":    -250,
		"  7.05  ":  705,
		"1,000,000": 100000000,
		".50":       50,
		"$0.01":     1,
	}
	for in, want := range good {
		got, err := parseCents(in)
		if err != nil {
			t.Errorf("parseCents(%q) errored: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseCents(%q) = %d, want %d", in, got, want)
		}
	}
}

// TestParseOrderDetailsRejectsBlankRequiredMoneyCell — the file-level half of
// F5(a). A report that prints an empty Amount must fail as loudly as a report
// that dropped the Amount column, because landing amount_cents=0 poisons every
// money figure computed off it and nothing downstream can tell 0 from absent.
func TestParseOrderDetailsRejectsBlankRequiredMoneyCell(t *testing.T) {
	for _, blank := range []string{
		`ord-1,7,09/28/26 11:42 AM,09/28/26 11:48 AM,0.00,,13.53,false,In Store`,  // Amount blank
		`ord-1,7,09/28/26 11:42 AM,09/28/26 11:48 AM,,12.50,13.53,false,In Store`, // Discount Amount blank
		`ord-1,7,09/28/26 11:42 AM,09/28/26 11:48 AM,0.00,12.50,,false,In Store`,  // Total blank
	} {
		if _, err := parseOrderDetails(strings.NewReader(orderCSV(blank)), ""); err == nil {
			t.Errorf("a blank required money cell parsed without error: %s", blank)
		}
	}

	// A fully-populated row still parses.
	ok := `ord-1,7,09/28/26 11:42 AM,09/28/26 11:48 AM,0.00,12.50,13.53,false,In Store`
	if _, err := parseOrderDetails(strings.NewReader(orderCSV(ok)), ""); err != nil {
		t.Fatalf("a well-formed row must still parse: %v", err)
	}
}
