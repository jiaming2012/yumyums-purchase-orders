package sync

// ═══════════════════════════════════════════════════════════════════════════
// Fixture-ownership guard — card `gate-rls-fixture-ownership` (A3 re-gate)
// ═══════════════════════════════════════════════════════════════════════════
//
// B-141: HQ_RLS_TEST_DB used to flow unguarded into `DROP DATABASE … WITH
// (FORCE)`. On 2026-08-06 a G6 probe set it to `yumyums` and the suite
// accepted it — against the then-production admin URL — and destroyed the
// production database (ledger decision 155). The A3 card that first fixed
// this guarded the value with a four-item blocklist; the review's finding was
// that a blocklist enumerates what must not be destroyed and can never be
// complete, while a prefix enumerates what may be — only the latter is a
// boundary. This file is the attended re-gate's evidence for the prefix form.
//
// B-142(a): the same card's own test blind-dropped a database it did not
// create (EXIT=0). The property re-gated here is REFUSE, DON'T DROP: the
// suite may only ever drop a database this run itself created.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestRVFixtureGuard_RefusalFiresBeforeSubstrate re-runs TestRowVisibilityRLS
// in a child process with the exact configuration of the 2026-08-06 incident
// probe — HQ_RLS_TEST_DB=yumyums — plus a substrate marked optional and an
// admin URL where nothing listens. The fixed suite must FAIL the child fast
// with the prefix-boundary refusal, having opened no socket. The unfixed
// suite skips at the substrate gate (or worse, proceeds to connect): the name
// was never examined, which is precisely B-141.
func TestRVFixtureGuard_RefusalFiresBeforeSubstrate(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run", "^TestRowVisibilityRLS$", "-test.v")
	cmd.Env = append(os.Environ(),
		"HQ_RLS_TEST_DB=yumyums", // the name that destroyed production
		"HQ_SYNC_SUBSTRATE_OPTIONAL=1",
		// Port 9 (discard) — nothing listens. If the child tries to connect
		// anywhere before refusing, that is itself the defect.
		"HQ_ADMIN_DB_URL=postgres://hqtest:hqtest@127.0.0.1:9/postgres",
	)
	out, err := cmd.CombinedOutput()
	s := string(out)

	if err == nil {
		t.Fatalf("child accepted HQ_RLS_TEST_DB=yumyums (exit 0) — the incident probe "+
			"passes again. Output tail:\n%s", tail(s, 800))
	}
	if !strings.Contains(s, "hq_rls_") {
		t.Fatalf("child failed, but not with the prefix-boundary refusal — the name was "+
			"not what was examined. Output tail:\n%s", tail(s, 800))
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

// TestRVFixtureDBNameCheck_Boundary pins the prefix boundary from both sides.
// The refused list includes every name the 2026-08-06 blocklist carried, the
// name that destroyed production, the OLD shared default (B-35's constant is
// now outside the boundary), and injection-shaped strings.
func TestRVFixtureDBNameCheck_Boundary(t *testing.T) {
	refused := []string{
		"yumyums", // accepted by the blocklist; dropped production
		"postgres",
		"hq_test_go",
		"hq_test_e2e",
		"hq_test_b2_fdw", // the old shared default — B-35
		"hq_rls_UPPER",
		"hq_rls_x; DROP DATABASE hq_test_go",
		`hq_rls_x"`,
		"hq_rls_",
		"hq_rlsx",
		"",
	}
	for _, name := range refused {
		if err := rvFixtureDBNameCheck(name); err == nil {
			t.Errorf("boundary accepted %q — it must refuse everything outside ^hq_rls_[a-z0-9_]+$", name)
		}
	}
	accepted := []string{"hq_rls_b2_fdw_p123", "hq_rls_g6a3_0806", "hq_rls_claimprobe"}
	for _, name := range accepted {
		if err := rvFixtureDBNameCheck(name); err != nil {
			t.Errorf("boundary refused %q: %v — it is inside the prefix", name, err)
		}
	}
}

// TestRVFixtureDefault_PerProcessAndInsideBoundary pins the B-35 fix: with no
// override the name is derived per-process (two concurrent legs cannot
// collide) and sits inside the same boundary an override must.
func TestRVFixtureDefault_PerProcessAndInsideBoundary(t *testing.T) {
	t.Setenv("HQ_RLS_TEST_DB", "")
	name := rvHQDatabase()
	if err := rvFixtureDBNameCheck(name); err != nil {
		t.Fatalf("the derived default %q fails its own boundary: %v", name, err)
	}
	if !strings.Contains(name, fmt.Sprintf("_p%d", os.Getpid())) {
		t.Fatalf("derived default %q is not per-process — a shared constant is B-35", name)
	}
}

// TestRVClaimFixtureDatabase_RefusesExisting proves B-142(a) against the real
// test cluster: an existing database is refused, never dropped; a free name
// is created. Runs against :5434 (`task test:db:up`); skips loudly if that
// cluster is down.
func TestRVClaimFixtureDatabase_RefusesExisting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	adminURL := env("HQ_ADMIN_DB_URL", defaultHQAdminURL)
	admin, err := pgxpool.New(ctx, adminURL)
	if err == nil {
		err = admin.Ping(ctx)
	}
	if err != nil {
		t.Skipf("test cluster unreachable (%s) — bring it up with `task test:db:up`: %v",
			redactDSN(adminURL), err)
	}
	defer admin.Close()

	// Out-of-boundary names are refused before the cluster is even asked.
	if err := rvClaimFixtureDatabase(ctx, admin, "hq_test_go"); err == nil {
		t.Fatal("claim accepted hq_test_go — the claim must re-check the B-141 boundary")
	}

	name := fmt.Sprintf("hq_rls_claimprobe_p%d", os.Getpid())
	sane := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+sane); err != nil {
		t.Fatalf("pre-create %s: %v", name, err)
	}
	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dcancel()
		if _, err := admin.Exec(dctx, `DROP DATABASE IF EXISTS `+sane+` WITH (FORCE)`); err != nil {
			t.Errorf("probe cleanup: drop %s: %v", name, err)
		}
	}()

	// Held by "another leg" (this test) → refused, and still standing after.
	if err := rvClaimFixtureDatabase(ctx, admin, name); err == nil {
		t.Fatalf("claim of the existing %s succeeded — that is the blind drop (B-142a)", name)
	}
	var exists bool
	if err := admin.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, name).Scan(&exists); err != nil {
		t.Fatalf("post-refusal existence check: %v", err)
	}
	if !exists {
		t.Fatalf("%s is GONE after a refused claim — refusal must not destroy", name)
	}

	// Freed → the claim creates it.
	if _, err := admin.Exec(ctx, `DROP DATABASE `+sane+` WITH (FORCE)`); err != nil {
		t.Fatalf("free %s: %v", name, err)
	}
	if err := rvClaimFixtureDatabase(ctx, admin, name); err != nil {
		t.Fatalf("claim of the freed %s failed: %v", name, err)
	}
}
