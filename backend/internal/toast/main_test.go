package toast

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
	"github.com/yumyums/hq/internal/testdb"
)

// testPool is the migrated pool the DB-backed tests in this package use. Nil
// when no database is reachable, in which case those tests skip and the
// hermetic ones (parser, config, syncstatus, failloud) still run.
var testPool *pgxpool.Pool

// TestMain is this package's first — until card H3a, internal/toast had no
// DB-backed test at all. It is internal/marketing/helpers_test.go's TestMain
// verbatim in structure, including internal/testdb's asymmetric gate:
//
//	DB_TEST_URL UNSET            -> skip (a contributor with no Postgres still
//	                                runs the parser tests)
//	DB_TEST_URL SET, unreachable -> FAIL (setting it is a statement of intent;
//	                                saying `ok` would be a lie — B-16/decision 90)
//
// db.Migrate is called HERE rather than relying on another package's TestMain
// having gone first: migration 0084 is this card's, and under -p 1 a package
// whose schema depends on another package's ordering is a package that reds
// when the ordering changes.
func TestMain(m *testing.M) {
	dbURL := os.Getenv(testdb.EnvVar)
	requested := dbURL != ""
	if dbURL == "" {
		dbURL = "postgres://hqtest:hqtest@localhost:5434/hq_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testdb.ExitIfRequested(requested, dbURL, "connect", err)
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

// setupOrdersDB hands back the migrated pool with toast_orders emptied.
// reconciliation_decisions and scan_attempts_mirror are left alone — they are
// other tests' fixtures and nothing here touches them.
func setupOrdersDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and the local fallback is unreachable)")
	}
	if _, err := testPool.Exec(t.Context(), `TRUNCATE toast_orders`); err != nil {
		t.Fatalf("truncate toast_orders: %v", err)
	}
	return testPool
}
