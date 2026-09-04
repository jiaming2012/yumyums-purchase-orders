package redemption

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	// Computed BEFORE the fallback: the fallback is the *unset* case, and the
	// unset case still skips. See internal/testdb for the asymmetry.
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://hqtest:hqtest@localhost:5434/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
		// DB_TEST_URL unset and the local fallback is not there — leave
		// testPool nil so the DB-coupled tests skip.
		os.Exit(m.Run())
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		testdb.ExitIfRequested(requested, dbURL, "ping", err)
		os.Exit(m.Run())
	}
	if err := db.Migrate(pool); err != nil {
		pool.Close()
		panic("db.Migrate failed: " + err.Error())
	}
	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func resetF4(t *testing.T) {
	t.Helper()
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and local fallback unreachable)")
	}
	if _, err := testPool.Exec(t.Context(), `TRUNCATE race_lost_notifications`); err != nil {
		t.Fatalf("truncate race_lost_notifications: %v", err)
	}
}

// 🔴 RED-FIRST · F4 / E-KR3 — the card's done_when: a two-attempt
// reconciliation emits RaceLostReconciled and creates the manager
// notification. Attempt 1 (device A, online) wins the atomic burn; attempt 2
// (device B, a SYNCED offline_override) arbitrates to the already_used
// terminal — the server must emit the RaceLostReconciled domain event and
// persist the Shift-Manager read-model entry (code, device, staff, time,
// value). Red while the observer resolves the await but emits nothing.
func TestF4_RaceLostReconciledEmitsNotification(t *testing.T) {
	resetF4(t)

	stub := &atomicStubArbiter{}
	arb := testArbitrator(stub, PGRaceLostStore{Pool: testPool})
	ctx := context.Background()

	// Attempt 1 — device A, plain online submit. Wins.
	res1, err := arb.Arbitrate(ctx, Attempt{
		TokenHash:    "f4-code-hash",
		DeviceID:     "device-a",
		AuthorizedBy: "counter@yumyums.kitchen",
	})
	if err != nil {
		t.Fatalf("attempt 1: %v", err)
	}
	if res1.Result != ResultRedeemed {
		t.Fatalf("attempt 1 result=%s, want redeemed", res1.Result)
	}
	if res1.RaceLostReconciled {
		t.Fatal("attempt 1 (the winner) flagged race_lost_reconciled")
	}

	// Attempt 2 — device B's synced offline override. Loses the race.
	scannedAt := time.Date(2026, 9, 4, 19, 42, 0, 0, time.UTC)
	res2, err := arb.Arbitrate(ctx, Attempt{
		TokenHash:       "f4-code-hash",
		DeviceID:        "device-b",
		AuthorizedBy:    "manager@yumyums.kitchen",
		OrderNumber:     "4021",
		OfflineOverride: true,
		UnverifiedCode:  true,
		ScannedAt:       scannedAt,
		Value:           2.50,
		ValueKnown:      true,
	})
	if err != nil {
		t.Fatalf("attempt 2: %v", err)
	}
	if res2.Result != ResultAlreadyUsed {
		t.Fatalf("attempt 2 result=%s, want already_used", res2.Result)
	}
	if !res2.RaceLostReconciled {
		t.Fatal("F4: already_used on a synced offline_override did NOT flag race_lost_reconciled")
	}

	// The read-model entry: code, device, staff, time, value (F4).
	var (
		device, staff, orderNumber string
		gotScannedAt               time.Time
		value                      float64
		unverified                 bool
	)
	err = testPool.QueryRow(ctx, `
		SELECT device_id, staff, order_number, scanned_at, value, unverified_code
		  FROM race_lost_notifications
		 WHERE code_token_hash = $1`, "f4-code-hash").
		Scan(&device, &staff, &orderNumber, &gotScannedAt, &value, &unverified)
	if err != nil {
		t.Fatalf("F4: no race_lost_notifications entry was created: %v", err)
	}
	if device != "device-b" {
		t.Fatalf("entry device=%q, want device-b", device)
	}
	if staff != "manager@yumyums.kitchen" {
		t.Fatalf("entry staff=%q, want manager@yumyums.kitchen", staff)
	}
	if orderNumber != "4021" {
		t.Fatalf("entry order_number=%q, want 4021", orderNumber)
	}
	if !gotScannedAt.Equal(scannedAt) {
		t.Fatalf("entry scanned_at=%v, want %v", gotScannedAt, scannedAt)
	}
	if value != 2.50 {
		t.Fatalf("entry value=%v, want 2.50", value)
	}
	if !unverified {
		t.Fatal("entry unverified_code=false, want true (F2 flag must carry through)")
	}
}

// Negative control: an online already_used WITHOUT offline_override is the
// normal F3 "server wins" outcome — no domain event, no notification row.
func TestF4_OnlineAlreadyUsedWithoutOverrideDoesNotEmit(t *testing.T) {
	resetF4(t)

	stub := &atomicStubArbiter{}
	arb := testArbitrator(stub, PGRaceLostStore{Pool: testPool})
	ctx := context.Background()

	if _, err := arb.Arbitrate(ctx, Attempt{TokenHash: "f3-code", DeviceID: "device-a"}); err != nil {
		t.Fatalf("attempt 1: %v", err)
	}
	res2, err := arb.Arbitrate(ctx, Attempt{TokenHash: "f3-code", DeviceID: "device-b"})
	if err != nil {
		t.Fatalf("attempt 2: %v", err)
	}
	if res2.Result != ResultAlreadyUsed {
		t.Fatalf("attempt 2 result=%s, want already_used", res2.Result)
	}
	if res2.RaceLostReconciled {
		t.Fatal("plain online already_used flagged race_lost_reconciled")
	}
	var n int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM race_lost_notifications`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("%d notification rows for a non-override already_used, want 0", n)
	}
}
