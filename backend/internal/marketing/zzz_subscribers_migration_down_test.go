package marketing

// zzz_subscribers_migration_down_test.go — 0085's Down, proven. A Down nobody
// has ever run is a Down that does not work.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 WHY THIS FILE EXISTS SEPARATELY, AND WHY IT RESTORES TO HEAD
//
// Card H1's zz_migration_down_test.go ends its leg with db.MigrateTo(pool, 83)
// — a HARD-CODED version. The Go suite shares ONE database and runs with -p 1,
// so a package that leaves the schema pinned at 83 leaves every LATER package
// looking at a database with 0084's and 0085's tables dropped. Card H3a hit
// exactly that when 0084 landed and changed that call to db.Migrate(pool).
//
// This card does NOT touch that file: the fix is Card H3a's and the
// orchestrator takes its side at merge. This file adds only its own leg, and
// it is named `zzz_` so it sorts AFTER `zz_` and therefore runs after it —
// which means it also REPAIRS whatever version the previous leg left behind,
// because its first and last acts are both db.Migrate(pool).
//
// Two rules this file keeps, and the reason for each:
//
//   1. IT RESTORES WITH db.Migrate(pool), NEVER WITH A LITERAL. A literal is
//      correct only until the next card adds a migration, and then it silently
//      drops that card's tables out from under every later package.
//   2. IT DERIVES the version under test from the migration FILENAME rather
//      than hard-coding 85. This card's merge-intent says 0085 may have to
//      renumber upward if another card claims the number; a derived version
//      renumbers with it, a literal reds for a reason that has nothing to do
//      with the behaviour under test.
// ═══════════════════════════════════════════════════════════════════════════

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/yumyums/hq/internal/db"
)

// migrationsDir is this package's relative path to the goose migrations.
const migrationsDir = "../db/migrations"

var migrationNumRe = regexp.MustCompile(`^(\d+)_`)

// subscribersMigrationVersion finds THIS CARD's migration by its name suffix
// and returns its goose version. It fails loudly rather than guessing: a
// missing migration must red here, not silently skip the coverage.
func subscribersMigrationVersion(t *testing.T) int64 {
	t.Helper()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read %s: %v", migrationsDir, err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, "_subscribers.sql") {
			continue
		}
		m := migrationNumRe.FindStringSubmatch(name)
		if m == nil {
			t.Fatalf("migration %q does not start with a version number", name)
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Fatalf("parse version from %q: %v", name, err)
		}
		return v
	}
	t.Fatalf("no *_subscribers.sql migration found in %s", migrationsDir)
	return 0
}

// TestMigration0085SubscribersDownAndUpRoundTrip migrates down past this
// card's migration, asserts both of its tables are gone, then migrates back up
// to HEAD and asserts they return.
func TestMigration0085SubscribersDownAndUpRoundTrip(t *testing.T) {
	// 🛑 NOT setupSubsTestDB. That helper TRUNCATEs this card's two tables, and
	// Card H1's zz_ leg — which runs immediately before this one — leaves the
	// schema at 83, where those tables DO NOT EXIST. Truncating first fails
	// with 42P01 before a single assertion runs. Observed exactly that:
	//
	//   zzz_subscribers_migration_down_test.go:81: setupSubsTestDB truncate:
	//   ERROR: relation "subscriber_events" does not exist (SQLSTATE 42P01)
	//
	// So this test migrates to HEAD FIRST and truncates nothing: it reads
	// information_schema and never touches a row.
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and the local fallback is unreachable)")
	}
	pool := testPool
	ctx := context.Background()
	tables := []string{"subscribers", "subscriber_events"}

	exists := func(name string) bool {
		var ok bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			                WHERE table_schema = current_schema() AND table_name = $1)`,
			name).Scan(&ok); err != nil {
			t.Fatalf("exists(%s): %v", name, err)
		}
		return ok
	}

	// Repair first. Whatever the previous leg in this package left the schema
	// at, this test starts from HEAD — so it never reds for someone else's
	// hard-coded version, and never asserts against a half-migrated database.
	// This line is the one that actually undoes zz_'s MigrateTo(pool, 83).
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate up to HEAD before the down leg: %v", err)
	}
	for _, tb := range tables {
		if !exists(tb) {
			t.Fatalf("precondition: %s is missing before the down leg", tb)
		}
	}

	version := subscribersMigrationVersion(t)
	if err := db.MigrateTo(pool, version-1); err != nil {
		t.Fatalf("migrate down to %d: %v", version-1, err)
	}
	for _, tb := range tables {
		if exists(tb) {
			t.Errorf("%s survived migration %d's Down", tb, version)
		}
	}

	// 🛑 BACK UP TO HEAD, NOT TO A NUMBER. This is the line that keeps every
	// later package under -p 1 looking at a complete schema.
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate back up to HEAD: %v", err)
	}
	for _, tb := range tables {
		if !exists(tb) {
			t.Errorf("%s did not come back on re-apply", tb)
		}
	}

	// The FK onto Card H1's qr_codes(short) is the join 0085 exists to make,
	// so prove it survived the round trip rather than only that the table did.
	var fkCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.table_constraints tc
		  JOIN information_schema.constraint_column_usage ccu
		    ON ccu.constraint_name = tc.constraint_name
		 WHERE tc.table_name = 'subscribers'
		   AND tc.constraint_type = 'FOREIGN KEY'
		   AND ccu.table_name = 'qr_codes'`).Scan(&fkCount); err != nil {
		t.Fatalf("read the source_short FK: %v", err)
	}
	if fkCount == 0 {
		t.Error("subscribers.source_short lost its FK onto qr_codes(short) across the round trip")
	}
	t.Logf("migration %d Down/Up round-trip clean; schema restored to HEAD (not to a literal)", version)
}
