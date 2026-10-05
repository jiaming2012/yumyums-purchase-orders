package marketing

// erasure_test.go — card I3 (`dish-merge-and-erasure-backstop`, decision 194).
//
// Migration 0086 makes three deletes plain and leaves a fourth refused ON
// PURPOSE. Every test here runs the DELETE itself against the migrated schema
// — no handler, no stub — because the contract is the database's:
//
//   - DELETE FROM subscribers  → the timeline goes with them (CASCADE, the one
//     cascade in the migration) and every qr_scans row that named them is
//     blanked, not removed (SET NULL);
//   - DELETE FROM qr_codes for an UN-scanned code → the subscriber who
//     first-touched it survives with source_short NULL;
//   - 🛑 DELETE FROM qr_codes for a SCANNED code → still 23503 on
//     qr_scans_short_fkey. Codes are deactivated, never deleted; scan history
//     is attribution evidence. That refusal is the design, and a test that
//     starts passing a delete here means someone added a cascade.

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yumyums/hq/internal/db"
)

// erasureFixture is one campaign with one code, inserted by plain SQL so these
// tests depend on the schema and nothing else.
type erasureFixture struct {
	campaignID string
	short      string
}

func seedErasureCampaignAndCode(t *testing.T, pool *pgxpool.Pool, short string) erasureFixture {
	t.Helper()
	ctx := context.Background()
	uid := seedUser(t, pool, "manager")
	var f erasureFixture
	f.short = short
	if err := pool.QueryRow(ctx, `
		INSERT INTO campaigns_admin (id, slug, name, offer_text, face_value_cents, requires_online, ends_at, created_by)
		VALUES (gen_random_uuid(), $1, 'Erasure Test', '$1 off', 100, false, now() + interval '7 days', $2)
		RETURNING id::text`, "erasure-"+randSuffix(t), uid).Scan(&f.campaignID); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO qr_codes (short, campaign_id, channel, created_by)
		VALUES ($1, $2, 'flyer', $3)`, short, f.campaignID, uid); err != nil {
		t.Fatalf("seed code %s: %v", short, err)
	}
	return f
}

func seedErasureSubscriber(t *testing.T, pool *pgxpool.Pool, sourceShort *string) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `
		INSERT INTO subscribers (display_name, source, source_short, joined_at)
		VALUES ('Erasure Subject', 'qr', $1, now()) RETURNING id::text`, sourceShort).Scan(&id); err != nil {
		t.Fatalf("seed subscriber: %v", err)
	}
	return id
}

func countWhere(t *testing.T, pool *pgxpool.Pool, table, where string, args ...any) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM `+table+` WHERE `+where, args...).Scan(&n); err != nil {
		t.Fatalf("count %s where %s: %v", table, where, err)
	}
	return n
}

// One subscriber, one timeline event, one scan naming them. A single DELETE
// erases the subscriber: no event remains, and the scan survives with its
// subscriber_id blanked. Before 0086 the DELETE is refused with 23503
// (subscriber_events_subscriber_id_fkey), and qr_scans.subscriber_id had no FK
// at all, so even a forced delete left the id dangling.
func TestSubscriberDeleteCascadesTimelineAndBlanksScans(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	f := seedErasureCampaignAndCode(t, pool, "ERASE2")
	subID := seedErasureSubscriber(t, pool, nil)

	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriber_events (subscriber_id, kind) VALUES ($1, 'signed_up')`, subID); err != nil {
		t.Fatalf("seed event: %v", err)
	}
	var scanID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO qr_scans (short, subscriber_id) VALUES ($1, $2) RETURNING id`,
		f.short, subID).Scan(&scanID); err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	ct, err := pool.Exec(ctx, `DELETE FROM subscribers WHERE id = $1`, subID)
	if err != nil {
		t.Fatalf("DELETE FROM subscribers: %v", err)
	}
	if ct.RowsAffected() != 1 {
		t.Fatalf("DELETE FROM subscribers affected %d rows, want 1", ct.RowsAffected())
	}

	if n := countWhere(t, pool, "subscriber_events", "subscriber_id = $1", subID); n != 0 {
		t.Errorf("%d subscriber_events rows remain for the erased subscriber, want 0", n)
	}
	if n := countWhere(t, pool, "subscriber_events", "true"); n != 0 {
		t.Errorf("%d subscriber_events rows remain in total, want 0", n)
	}

	// The scan is attribution evidence and STAYS; only the pointer is blanked.
	var scanSub *string
	if err := pool.QueryRow(ctx,
		`SELECT subscriber_id::text FROM qr_scans WHERE id = $1`, scanID).Scan(&scanSub); err != nil {
		t.Fatalf("the scan row is gone — erasure must blank it, not delete it: %v", err)
	}
	if scanSub != nil {
		t.Errorf("qr_scans.subscriber_id = %q after the subscriber was deleted, want NULL", *scanSub)
	}

	// Erasure removes the subscriber's own rows and nothing else.
	if n := countWhere(t, pool, "qr_codes", "short = $1", f.short); n != 1 {
		t.Errorf("the code did not survive the subscriber's erasure (count %d)", n)
	}
	if n := countWhere(t, pool, "campaigns_admin", "id = $1", f.campaignID); n != 1 {
		t.Errorf("the campaign did not survive the subscriber's erasure (count %d)", n)
	}
}

// An UN-scanned code a subscriber first-touched can be deleted; the subscriber
// stays and loses only the attribution link. Before 0086: 23503
// (subscribers_source_short_fkey).
func TestCodeDeleteBlanksFirstTouch(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	f := seedErasureCampaignAndCode(t, pool, "ERASE3")
	subID := seedErasureSubscriber(t, pool, &f.short)

	ct, err := pool.Exec(ctx, `DELETE FROM qr_codes WHERE short = $1`, f.short)
	if err != nil {
		t.Fatalf("DELETE FROM qr_codes: %v", err)
	}
	if ct.RowsAffected() != 1 {
		t.Fatalf("DELETE FROM qr_codes affected %d rows, want 1", ct.RowsAffected())
	}

	var sourceShort *string
	if err := pool.QueryRow(ctx,
		`SELECT source_short FROM subscribers WHERE id = $1`, subID).Scan(&sourceShort); err != nil {
		t.Fatalf("the subscriber is gone — deleting a code must never delete a person: %v", err)
	}
	if sourceShort != nil {
		t.Errorf("subscribers.source_short = %q after its code was deleted, want NULL", *sourceShort)
	}
	if n := countWhere(t, pool, "campaigns_admin", "id = $1", f.campaignID); n != 1 {
		t.Errorf("the campaign did not survive its code's deletion (count %d)", n)
	}
}

// 🛑 A code that has been scanned is NOT deletable, before or after 0086. This
// test is green on both trees on purpose: it pins the refusal so that a later
// "fix" which cascades qr_scans.short reds here.
func TestScannedCodeDeleteIsRefused(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	f := seedErasureCampaignAndCode(t, pool, "ERASE4")
	if _, err := pool.Exec(ctx, `INSERT INTO qr_scans (short) VALUES ($1)`, f.short); err != nil {
		t.Fatalf("seed scan: %v", err)
	}

	_, err := pool.Exec(ctx, `DELETE FROM qr_codes WHERE short = $1`, f.short)
	if err == nil {
		t.Fatal("DELETE FROM qr_codes succeeded for a SCANNED code — scan history must block it (codes are deactivated, never deleted)")
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		t.Fatalf("want a Postgres error, got %T: %v", err, err)
	}
	if pgErr.Code != "23503" || pgErr.ConstraintName != "qr_scans_short_fkey" {
		t.Fatalf("refusal = %s %s, want 23503 qr_scans_short_fkey", pgErr.Code, pgErr.ConstraintName)
	}
	if n := countWhere(t, pool, "qr_codes", "short = $1", f.short); n != 1 {
		t.Errorf("the scanned code is gone (count %d)", n)
	}
	if n := countScans(t, pool, f.short); n != 1 {
		t.Errorf("scan history count = %d, want 1", n)
	}
}

// erasureMigrationSuffix names THIS card's migration; the version is derived
// from the filename (the zzz_ file's rule 2), never hard-coded.
const erasureMigrationSuffix = "_merge_repoint_and_erasure_backstop.sql"

func erasureMigrationVersion(t *testing.T) int64 {
	t.Helper()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read %s: %v", migrationsDir, err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), erasureMigrationSuffix) {
			continue
		}
		m := migrationNumRe.FindStringSubmatch(e.Name())
		if m == nil {
			t.Fatalf("migration %q does not start with a version number", e.Name())
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			t.Fatalf("parse version from %q: %v", e.Name(), err)
		}
		return v
	}
	t.Fatalf("no *%s migration found in %s", erasureMigrationSuffix, migrationsDir)
	return 0
}

// fkDeleteActions reads pg_constraint for every FK this card cares about and
// returns constraint name → confdeltype ('a' no action, 'n' set null,
// 'c' cascade). A constraint that does not exist is absent from the map.
func fkDeleteActions(t *testing.T, pool *pgxpool.Pool) map[string]string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT c.conname, c.confdeltype::text
		  FROM pg_constraint c
		  JOIN pg_namespace n ON n.oid = c.connamespace
		 WHERE c.contype = 'f' AND n.nspname = current_schema()
		   AND c.conname = ANY($1)`,
		[]string{
			"campaigns_admin_item_id_fkey", "qr_codes_item_id_fkey",
			"subscribers_source_short_fkey", "subscriber_events_subscriber_id_fkey",
			"qr_scans_subscriber_id_fkey", "qr_scans_short_fkey",
		})
	if err != nil {
		t.Fatalf("read pg_constraint: %v", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var name, action string
		if err := rows.Scan(&name, &action); err != nil {
			t.Fatalf("scan pg_constraint: %v", err)
		}
		out[name] = action
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pg_constraint: %v", err)
	}
	return out
}

func assertFKActions(t *testing.T, leg string, got, want map[string]string) {
	t.Helper()
	for name, w := range want {
		g, ok := got[name]
		switch {
		case w == "" && ok:
			t.Errorf("%s: %s exists (confdeltype %q), want it absent", leg, name, g)
		case w != "" && !ok:
			t.Errorf("%s: %s is missing, want confdeltype %q", leg, name, w)
		case w != "" && g != w:
			t.Errorf("%s: %s confdeltype = %q, want %q", leg, name, g, w)
		}
	}
}

// TestMigration0086ErasureBackstopDownAndUpRoundTrip proves 0086's Down: the
// four re-declared FKs go back to NO ACTION, the fifth (which 0083 never
// declared) is dropped, and re-applying restores n/n/n/c + n. In every leg
// qr_scans_short_fkey is NO ACTION — the migration never touches it.
//
// Same rules as the zzz_ leg: start from HEAD, restore with db.Migrate (never
// a literal), read the catalog and touch no row.
func TestMigration0086ErasureBackstopDownAndUpRoundTrip(t *testing.T) {
	if testPool == nil {
		t.Skip("no test database (DB_TEST_URL unset and the local fallback is unreachable)")
	}
	pool := testPool
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate up to HEAD before the down leg: %v", err)
	}
	// 🛑 Whatever happens below, the next test (and the next package under
	// -p 1) gets a fully migrated schema.
	t.Cleanup(func() {
		if err := db.Migrate(pool); err != nil {
			t.Errorf("restore to HEAD: %v", err)
		}
	})

	up := map[string]string{
		"campaigns_admin_item_id_fkey":         "n",
		"qr_codes_item_id_fkey":                "n",
		"subscribers_source_short_fkey":        "n",
		"subscriber_events_subscriber_id_fkey": "c",
		"qr_scans_subscriber_id_fkey":          "n",
		"qr_scans_short_fkey":                  "a",
	}
	down := map[string]string{
		"campaigns_admin_item_id_fkey":         "a",
		"qr_codes_item_id_fkey":                "a",
		"subscribers_source_short_fkey":        "a",
		"subscriber_events_subscriber_id_fkey": "a",
		"qr_scans_subscriber_id_fkey":          "", // absent: 0083 never declared it
		"qr_scans_short_fkey":                  "a",
	}

	assertFKActions(t, "at HEAD", fkDeleteActions(t, pool), up)

	version := erasureMigrationVersion(t)
	if err := db.MigrateTo(pool, version-1); err != nil {
		t.Fatalf("migrate down to %d: %v", version-1, err)
	}
	assertFKActions(t, "after Down", fkDeleteActions(t, pool), down)

	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate back up to HEAD: %v", err)
	}
	assertFKActions(t, "after re-apply", fkDeleteActions(t, pool), up)
	t.Logf("migration %d Down/Up round-trip clean; schema restored to HEAD (not to a literal)", version)
}

// TestMigration0086BlanksDanglingScanReferences pins the one statement in 0086
// that touches rows (card J2, BACKLOG B-480). `qr_scans.subscriber_id` had no
// FK from 0083 to 0085, so a database can hold a scan whose subscriber_id
// names nobody. 0086 blanks such ids before it adds the FK; without that
// UPDATE the ADD CONSTRAINT is refused with 23503 and the whole migration —
// and with it the deploy — fails. Until this test, removing the UPDATE left
// every erasure test and every round-trip green, because none of them holds a
// dangling id at the moment 0086 runs.
//
// The dangling row can only be seeded BELOW 0086 (at 86+ the constraint this
// test is about refuses the seed), so: migrate down to the version before
// 0086, seed one dangling scan and one control scan naming a real subscriber,
// migrate up, and read both back.
//
// Same rules as the round-trip above: the version comes from the filename, the
// restore is db.Migrate (never a literal). The cleanup empties the seeded
// tables BEFORE it restores, so even a 0086 that cannot swallow the dangling
// row leaves the next test a fully migrated schema — one red, not a cascade.
func TestMigration0086BlanksDanglingScanReferences(t *testing.T) {
	pool := setupSubsTestDB(t)
	ctx := context.Background()
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate up to HEAD before the down leg: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(),
			`TRUNCATE subscriber_events, subscribers, qr_scans, qr_codes, campaigns_admin RESTART IDENTITY CASCADE`); err != nil {
			t.Errorf("empty the seeded tables before the restore: %v", err)
		}
		if err := db.Migrate(pool); err != nil {
			t.Errorf("restore to HEAD: %v", err)
		}
	})

	version := erasureMigrationVersion(t)
	if err := db.MigrateTo(pool, version-1); err != nil {
		t.Fatalf("migrate down to %d: %v", version-1, err)
	}
	if action, ok := fkDeleteActions(t, pool)["qr_scans_subscriber_id_fkey"]; ok {
		t.Fatalf("at %d qr_scans_subscriber_id_fkey exists (confdeltype %q) — the dangling row cannot be seeded", version-1, action)
	}

	f := seedErasureCampaignAndCode(t, pool, "DANGL2")
	realSub := seedErasureSubscriber(t, pool, nil)
	const nobody = "00000000-0000-4000-8000-000000000480" // names no subscriber
	if n := countWhere(t, pool, "subscribers", "id = $1", nobody); n != 0 {
		t.Fatalf("the 'nobody' id names %d subscribers, want 0", n)
	}
	var danglingScan, controlScan int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO qr_scans (short, subscriber_id) VALUES ($1, $2) RETURNING id`,
		f.short, nobody).Scan(&danglingScan); err != nil {
		t.Fatalf("seed the dangling scan at %d: %v", version-1, err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO qr_scans (short, subscriber_id) VALUES ($1, $2) RETURNING id`,
		f.short, realSub).Scan(&controlScan); err != nil {
		t.Fatalf("seed the control scan at %d: %v", version-1, err)
	}

	// The deploy. With the blanking UPDATE gone this is where it stops:
	// 23503 on ADD CONSTRAINT qr_scans_subscriber_id_fkey.
	if err := db.Migrate(pool); err != nil {
		t.Fatalf("migrate %d -> HEAD over one dangling qr_scans.subscriber_id: %v", version-1, err)
	}

	// Blanked, not deleted: the scan is attribution evidence.
	var got *string
	if err := pool.QueryRow(ctx,
		`SELECT subscriber_id::text FROM qr_scans WHERE id = $1`, danglingScan).Scan(&got); err != nil {
		t.Fatalf("the dangling scan row is gone — the migration must blank it, not delete it: %v", err)
	}
	if got != nil {
		t.Errorf("dangling scan: subscriber_id = %q after the migration, want NULL", *got)
	}
	// The control keeps its subscriber: the UPDATE blanks only ids naming nobody.
	if err := pool.QueryRow(ctx,
		`SELECT subscriber_id::text FROM qr_scans WHERE id = $1`, controlScan).Scan(&got); err != nil {
		t.Fatalf("the control scan row is gone: %v", err)
	}
	if got == nil || *got != realSub {
		t.Errorf("control scan: subscriber_id = %v after the migration, want %s", got, realSub)
	}
	if n := countScans(t, pool, f.short); n != 2 {
		t.Errorf("scan rows for the code = %d after the migration, want 2", n)
	}
	if n := countWhere(t, pool, "subscribers", "id = $1", realSub); n != 1 {
		t.Errorf("the real subscriber did not survive the migration (count %d)", n)
	}
	assertFKActions(t, "after the migration", fkDeleteActions(t, pool),
		map[string]string{"qr_scans_subscriber_id_fkey": "n", "qr_scans_short_fkey": "a"})
}
