package marketing

import (
	"context"
	"testing"

	"github.com/yumyums/hq/internal/db"
)

// 0083's Down is claimed, so it is proven: migrate down to 82, assert all
// three tables are gone, migrate back up to 83, assert they are back. A Down
// nobody has ever run is a Down that does not work.
//
// Runs LAST in the package (zz_ prefix) and leaves the schema at 83, so no
// other test in this package — or, under -p 1, in a later package — sees a
// half-migrated database.
func TestMigration0083DownAndUpRoundTrip(t *testing.T) {
	pool := setupTestDB(t)
	ctx := context.Background()
	tables := []string{"campaigns_admin", "qr_codes", "qr_scans"}

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

	for _, tb := range tables {
		if !exists(tb) {
			t.Fatalf("precondition: %s is missing before the down leg", tb)
		}
	}

	if err := db.MigrateTo(pool, 82); err != nil {
		t.Fatalf("migrate down to 82: %v", err)
	}
	for _, tb := range tables {
		if exists(tb) {
			t.Errorf("%s survived 0083's Down", tb)
		}
	}

	// Back up to the LATEST, not to 83: later cards add migrations above this
	// one (H3a's 0084 is the first), and leaving the schema pinned at 83 would
	// hand the next package under -p 1 a database missing their tables. The
	// promise this test makes is "no other test sees a half-migrated database",
	// and db.Migrate is what keeps it true as the stack grows. (card H3a)
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate back up to latest: %v", err)
	}
	for _, tb := range tables {
		if !exists(tb) {
			t.Errorf("%s did not come back on re-apply", tb)
		}
	}
	t.Log("0083 Down/Up round-trip clean; schema left fully migrated")
}
