// Package testdb holds the one asymmetric gate the Go test suite uses when it
// cannot reach its database.
//
// # The asymmetry, and why both halves matter
//
//	DB_TEST_URL UNSET            -> SKIP.  A contributor without a Postgres must
//	                                       still be able to run the tests that
//	                                       need none. This is correct and stays.
//	DB_TEST_URL SET, unreachable -> FAIL.  Setting it is a statement of intent:
//	                                       "run the integration tests." If the
//	                                       database it names is not there, that
//	                                       intent was not met, and saying `ok`
//	                                       is a lie.
//
// The bug this package closes was the SYMMETRY, not the skip. Before it, both
// arms skipped. `pgxpool.New` is lazy, so a database that had been DROPped
// surfaced only at Ping — as a skip — and `go test` without -v prints a bare
// `ok <pkg> 1.2s` for a package whose every test skipped. On overnight-20260729
// a reviewer ran `DROP DATABASE hq_test_go_c` while the implementer was still
// using it; the implementer's suite kept reporting success and caught it only
// by wondering why the green had arrived so fast. Destroying the test
// environment was indistinguishable from passing it. (BACKLOG B-16; ledger
// T-27 decision 90.)
//
// This is the same gate, in the same repo, as the one internal/sync built for
// HQ_SYNC_SPIKE_LIVE — see sync/proxy_live_test.go's requireSpikeService:
// "flag set + port dead => FAIL, with the reason written into the failure
// message." That was the pattern to copy, and this package is the copy. There
// is deliberately only one.
//
// It is imported exclusively from _test.go files, so it is linked into no
// shipped binary.
package testdb

import (
	"fmt"
	"os"
)

// EnvVar is the environment variable the Taskfile sets and every DB-backed test
// in this repo reads.
const EnvVar = "DB_TEST_URL"

// Reason renders the single failure message used at every conversion site, so
// the eight of them cannot drift into eight different explanations.
//
// stage is "connect" or "ping" and is load-bearing, not decoration: pgxpool.New
// is lazy, so a Postgres that is DOWN and a Postgres that is UP but missing the
// named database both reach the caller as a Ping error, while a malformed DSN
// fails at connect. Distinguishing them is the difference between "start the
// database" and "someone dropped my database out from under me" — which is the
// incident that produced this gate.
func Reason(dsn, stage string, cause error) string {
	return fmt.Sprintf(
		"%s=%q was set, so a DATABASE-BACKED run was intended, but the test database "+
			"is not reachable (%s failed): %v\n\n"+
			"This is a FAILURE and not a skip on purpose. A skip here prints nothing "+
			"without -v, so an intended integration run silently degrades to whatever "+
			"hermetic coverage the package happens to have and still reports `ok` — "+
			"which is how a DROPped database once read as a passing suite. Start the "+
			"test Postgres and create the database (`task backend:db-test`), or UNSET "+
			"%s to deliberately run only the tests that need no database.",
		EnvVar, dsn, stage, cause, EnvVar)
}

// ExitIfRequested is the TestMain-shaped half of the gate. TestMain has no
// *testing.T, so there is nothing to call t.Fatalf on: it prints the reason to
// stderr and exits the test binary with status 1.
//
// requested is whether DB_TEST_URL was actually set — NOT whether a DSN was
// resolved. Every TestMain in this repo falls back to a hard-coded local DSN
// when the variable is empty, and that fallback is the *unset* case: it must
// still skip. Pass os.Getenv(testdb.EnvVar) != "", computed before the
// fallback is applied.
//
// When requested is false this returns normally and the caller proceeds to its
// existing skip path.
//
// It exits rather than panics so the reason is legible: a panic would bury it
// under a goroutine dump.
func ExitIfRequested(requested bool, dsn, stage string, cause error) {
	if !requested {
		return
	}
	fmt.Fprintln(os.Stderr, "FAIL\t"+Reason(dsn, stage, cause))
	os.Exit(1)
}
