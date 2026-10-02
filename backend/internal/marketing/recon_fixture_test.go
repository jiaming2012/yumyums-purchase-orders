package marketing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ── card H3b fixtures ──
//
// setupReconDB extends setupTestDB (card H1's, which truncates the 0083 tables)
// with the three tables migration 0084 adds. It is a SEPARATE helper rather
// than a widened setupTestDB so card H1's own tests keep the truncation set
// they were written against.
func setupReconDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := setupTestDB(t)
	if _, err := pool.Exec(t.Context(),
		`TRUNCATE reconciliation_decisions, scan_attempts_mirror, toast_orders RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("setupReconDB truncate: %v", err)
	}
	return pool
}

// subscribersTableExists reports whether card H5's migration 0085 has landed in
// the schema this test run migrated. The stats engine computes signups as 0
// with a stated basis when it has not (the card's own instruction: "compute 0
// and say so when it is absent, never fail"), so the fixture asserts the
// stronger signups figure only when the table is there.
func subscribersTableExists(t *testing.T, pool *pgxpool.Pool) bool {
	t.Helper()
	var exists bool
	if err := pool.QueryRow(t.Context(),
		`SELECT to_regclass('public.subscribers') IS NOT NULL`).Scan(&exists); err != nil {
		t.Fatalf("subscribersTableExists: %v", err)
	}
	return exists
}

// seedCampaignRow inserts a campaigns_admin row directly (no HTTP), so a stats
// fixture can pin face_value_cents and item_id without going through the create
// sheet's derivation.
func seedCampaignRow(t *testing.T, pool *pgxpool.Pool, userID, slug, name string, faceCents int, itemID *string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO campaigns_admin
		  (id, slug, name, offer_text, face_value_cents, requires_online, item_id,
		   landing, starts_at, ends_at, created_by)
		VALUES (gen_random_uuid(), $1, $2, $2 || ' offer', $3, false, $4,
		        'signup', now() - interval '2 days', now() + interval '30 days', $5)
		RETURNING id::text`, slug+"-"+randSuffix(t), name, faceCents, itemID, userID).Scan(&id)
	if err != nil {
		t.Fatalf("seedCampaignRow(%q): %v", slug, err)
	}
	return id
}

// seedCodeRow inserts a qr_codes row with an explicit short.
func seedCodeRow(t *testing.T, pool *pgxpool.Pool, userID, campaignID, short, channel string, itemID *string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO qr_codes (short, campaign_id, channel, item_id, created_by)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
		short, campaignID, channel, itemID, userID).Scan(&id)
	if err != nil {
		t.Fatalf("seedCodeRow(%q): %v", short, err)
	}
	return id
}

// seedScanRows inserts n qr_scans rows for a short.
func seedScanRows(t *testing.T, pool *pgxpool.Pool, short string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := pool.Exec(context.Background(),
			`INSERT INTO qr_scans (short, scanned_at, ip_hash) VALUES ($1, now() - interval '1 hour', $2)`,
			short, fmt.Sprintf("hash-%s-%d", short, i)); err != nil {
			t.Fatalf("seedScanRows(%q): %v", short, err)
		}
	}
}

// attemptFixture is one scan_attempts_mirror row a test wants to exist.
type attemptFixture struct {
	CodeID      *string // nil = no first-touch code (forces the 0084 CHECK's override arm)
	CampaignID  *string
	ScannedAt   time.Time
	OrderNumber *string
	Override    bool
	Status      string // "" => accepted
}

// seedAttempt inserts one mirrored attempt and returns its id.
func seedAttempt(t *testing.T, pool *pgxpool.Pool, f attemptFixture) string {
	t.Helper()
	status := f.Status
	if status == "" {
		status = "accepted"
	}
	scannedAt := f.ScannedAt
	if scannedAt.IsZero() {
		scannedAt = time.Now().Add(-2 * time.Hour)
	}
	// 0084's scan_attempts_mirror_names_a_code CHECK: an attempt with no
	// code_id must be an unverified offline override carrying a token hash.
	override := f.Override
	unverified := false
	var tokenHash *string
	if f.CodeID == nil {
		override, unverified = true, true
		h := "tok-" + randSuffix(t)
		tokenHash = &h
	}
	matchStatus := "orphan"
	if f.OrderNumber != nil {
		matchStatus = "unmatched"
	}
	var id string
	err := pool.QueryRow(context.Background(), `
		INSERT INTO scan_attempts_mirror
		  (id, code_id, campaign_id, device_id, scanned_at, status, offline_override,
		   unverified_code, token_hash, pos_order_number, pos_business_date, match_status)
		VALUES (gen_random_uuid(), $1, $2, 'tablet-1', $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text`,
		f.CodeID, f.CampaignID, scannedAt, status, override, unverified, tokenHash,
		f.OrderNumber, scannedAt.In(time.UTC).Format("2006-01-02"), matchStatus).Scan(&id)
	if err != nil {
		t.Fatalf("seedAttempt: %v", err)
	}
	return id
}

// seedToastOrder inserts one toast_orders row for the attempt's business date.
func seedToastOrder(t *testing.T, pool *pgxpool.Pool, businessDate time.Time, orderNumber string, amountCents, discountCents int, openedAt time.Time) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO toast_orders
		  (business_date, order_number, order_id, opened_at, closed_at,
		   amount_cents, discount_cents, total_cents, voided)
		VALUES ($1, $2, 'ord-' || $2, $3, $3 + interval '20 minutes', $4, $5, $4 - $5, false)`,
		businessDate.In(time.UTC).Format("2006-01-02"), orderNumber, openedAt,
		amountCents, discountCents); err != nil {
		t.Fatalf("seedToastOrder(%q): %v", orderNumber, err)
	}
}

// seedDecision appends a reconciliation_decisions row directly, for the states a
// test wants to start from rather than drive through HTTP.
func seedDecision(t *testing.T, pool *pgxpool.Pool, attemptID, decision string, reason, note, orderNumber *string, userID string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO reconciliation_decisions
		  (attempt_id, decision, order_number, reason, note, decided_by)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		attemptID, decision, orderNumber, reason, note, userID); err != nil {
		t.Fatalf("seedDecision(%s, %s): %v", attemptID, decision, err)
	}
}

func strptr(s string) *string { return &s }
