package sync

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ═══════════════════════════════════════════════════════════════════════════
// THE SUBSTRATE GATE — where the spike stack is, and what a missing one means
// ═══════════════════════════════════════════════════════════════════════════
//
// WHY THIS FILE EXISTS (finding F1, run overnight-20260801)
//
// The two attack suites in this package — TestJWTBridgeRLS and
// TestRowVisibilityRLS — are the entire deliverable of two cards. Before this
// file they resolved the substrate from HARD-CODED DEFAULT PORTS and SKIPPED
// when nothing answered there. Measured:
//
//	$ env -u SPIKE_DB_URL -u SPIKE_REST_URL -u SPIKE_JWT_SECRET \
//	    DB_TEST_URL=<live> go test ./internal/sync/ -count=1
//	ok    github.com/yumyums/hq/internal/sync   13.012s      <- exit 0
//
//	...the same run with -v:
//	--- SKIP: TestJWTBridgeRLS (5.00s)
//	--- SKIP: TestRowVisibilityRLS (8.01s)
//
// 🛑 AND THE DEFAULTS WERE NOT MERELY UNLUCKY — THEY WERE STRUCTURALLY WRONG.
// docker-compose.supabase.yml publishes EPHEMERAL host ports on purpose
// (`- "5432"`, no host mapping), so the compose project gets a NEW port every
// time it is recreated. The committed defaults said 46011/46233; the stack was
// observed at 51737/51717 and at 51317/51715 on other runs. SILENT SKIP WAS
// THEREFORE THE DEFAULT OUTCOME ON EVERY MACHINE, not an edge case — and
// scripts/verify-test-harness.sh Check B covers DB_TEST_URL only, so nothing
// caught it. A suite whose whole job is to be a guard was itself "a check whose
// subject set can go empty" (B-22/B-23/B-24).
//
// Two things fix it, and both are needed:
//
//  1. RESOLVE, DON'T GUESS. The published ports come from `docker compose port`,
//     which is the same question the compose file's own banner tells a human to
//     ask. There is no port constant left in this package to go stale.
//
//  2. THE ASYMMETRIC GATE, which this repo already has TWICE and which is
//     copied here rather than reinvented — internal/testdb (DB_TEST_URL) and
//     proxy_live_test.go's requireSpikeService (HQ_SYNC_SPIKE_LIVE):
//
//     substrate NOT configured      -> SKIP. Nobody brought a stack up; a
//     (compose project not running,        contributor without one must still be
//     no SPIKE_* set)                      able to run this package's hermetic
//     tests. This arm is correct and stays.
//
//     substrate CONFIGURED but      -> FAIL. Loudly. Resolving a published port
//     unreachable                          (or being handed SPIKE_DB_URL) is a
//     statement that a stack exists. If it
//     does not answer, that intent was not
//     met, and `ok` is a lie.
//
// 🛑 A SKIP IS NOT A PASS. When these suites skip, there is NO row-visibility
// evidence in the tree at all. Say so rather than implying otherwise.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 B-36 — THE FIRST ARM ABOVE WAS STILL A HOLE, AND IT WAS THE WHOLE HOLE
// ═══════════════════════════════════════════════════════════════════════════
//
// Card `sync-rxdb-write-policies` (overnight-20260802, A2). Fixed here rather
// than in a night of its own because this card DOUBLES the attack suite living
// in this package, and writing a second security gate into a package whose own
// gate cannot prove it ran is building on a foundation that needs rip-out
// (ledger T-30, decision 111's "Also folded" paragraph).
//
// F1's fix made "configured but unreachable" fail. It left "could not tell
// whether one is configured" as a SKIP — and that branch is reached by ANY
// failure of the `docker compose … port` shell-out, for ANY reason:
//
//	docker not installed · docker daemon down · the user not in the docker
//	group · a `docker` on PATH that is a wrapper returning non-zero · the
//	compose file renamed · cwd resolution wrong · the 20s context expiring
//	on a loaded machine
//
// Every one of those is INDISTINGUISHABLE, at this call site, from "this
// contributor deliberately has no stack". So the deliberate opt-out and the
// broken toolchain took the SAME door, and the door was silent. MEASURED on
// 2026-08-02 against the tree as it stood, with a `docker` shim on PATH that
// exits 1 (the daemon-down case, verbatim):
//
//	$ go test ./internal/sync/ -count=1 -run 'TestRowVisibilityRLS|TestJWTBridgeRLS'
//	ok    github.com/yumyums/hq/internal/sync   0.014s      <- exit 0
//
//	...the same run with -v:
//	--- SKIP: TestJWTBridgeRLS (0.00s)
//	--- SKIP: TestRowVisibilityRLS (0.00s)
//
// 0.014 SECONDS. Two attack suites, forty-nine subtests between them, and the
// package's `ok` line is the same `ok` line it prints when they all pass.
//
// 🛑 THE FIX: THE OPT-OUT MUST BE TYPED, NOT INFERRED. `HQ_SYNC_SUBSTRATE_OPTIONAL=1`
// is now the ONLY door to a skip. Absent it, an unresolvable substrate is a
// FAILURE — the same asymmetry F1 applied one layer up, applied to the layer
// that decides whether there is a layer at all.
//
//	HQ_SYNC_SUBSTRATE_OPTIONAL=1, nothing resolves   -> SKIP   (deliberate)
//	nothing set, nothing resolves                    -> FAIL   (🛑 B-36)
//	SPIKE_* declared, half resolves                  -> FAIL   (F1, unchanged)
//	resolved                                         -> RUN
//
// A contributor who genuinely does not want the substrate types six characters
// once. A CI box whose docker broke at 3am gets a red build instead of a green
// one that proved nothing. Those are not the same event and they no longer
// produce the same output.
//
// The decision is `spikeResolution`, a pure function, so it can be ASSERTED
// rather than described — and `resolveSpikeConfig` switches on it directly, so
// the assertion is about the shipped decision and not a parallel copy of it.

const (
	spikeComposeProject = "spike-supabase"

	spikeDBService   = "db"
	spikeDBPort      = "5432"
	spikeRESTService = "rest"
	spikeRESTPort    = "3000"

	spikeDBURLEnv   = "SPIKE_DB_URL"
	spikeRESTURLEnv = "SPIKE_REST_URL"

	// spikeOptionalEnv is the ONE door to a skip (B-36). It is deliberately a
	// NEW variable rather than a reuse of SPIKE_DB_URL/SPIKE_REST_URL: those two
	// say WHERE the substrate is, and inferring "there isn't one" from "you did
	// not say where it is" is precisely the inference that made a broken docker
	// look like a contributor's choice. This one says only "I know there is no
	// substrate here and I accept that this package proves nothing today."
	spikeOptionalEnv = "HQ_SYNC_SUBSTRATE_OPTIONAL"

	// spikeDBSuperPassword is the throwaway POSTGRES_PASSWORD committed
	// literally in docker-compose.supabase.yml (see the banner in that file:
	// generated for the spike, public in git on purpose, safe only because
	// nothing real is behind it). Naming it here introduces no new secret.
	//
	// It is a PASSWORD, not a port: it is fixed by the compose file and does not
	// change when the project is recreated, which is exactly why it may stay a
	// constant while the ports may not.
	spikeDBSuperPassword = "d8d866978aef6ac99c610bcb75b72431"
)

// spikeConfig is a RESOLVED substrate — every field is known-good enough to
// hand to a connection attempt.
type spikeConfig struct {
	dbURL   string
	restURL string
	secret  string
	// origin records how the endpoints were found, so a failure message can say
	// whether the operator declared them or the compose project did.
	origin string
}

// repoRootFromPackage resolves the repository root from this package's
// directory (backend/internal/sync), so nothing here depends on cwd.
func repoRootFromPackage() string { return filepath.Join("..", "..", "..") }

// spikeComposePort asks docker what host port the compose project actually
// published for a service. This is the whole point of the file: the answer is
// different every time the project is recreated, so it must be ASKED, never
// remembered.
func spikeComposePort(service, containerPort string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "compose",
		"-p", spikeComposeProject,
		"-f", "docker-compose.supabase.yml",
		"port", service, containerPort)
	cmd.Dir = repoRootFromPackage()

	out, err := cmd.Output()
	if err != nil {
		detail := ""
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			detail = ": " + strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("docker compose -p %s port %s %s failed (%v)%s",
			spikeComposeProject, service, containerPort, err, detail)
	}

	// One line per published mapping; take the first non-empty one. An empty
	// answer with exit 0 is docker's way of saying the service is not running or
	// publishes nothing — which is "not configured", not "broken".
	addr := ""
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			addr = line
			break
		}
	}
	if addr == "" {
		return "", fmt.Errorf("compose project %q publishes no host port for %s/%s "+
			"(is it up?)", spikeComposeProject, service, containerPort)
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("unparseable published address %q for %s/%s: %w",
			addr, service, containerPort, err)
	}
	return port, nil
}

// resolveSpikeConfig answers the ONLY question the gate's first arm asks: is a
// substrate configured at all?
//
// ok == false means genuinely not configured — no SPIKE_* variable, and the
// compose project is not publishing ports. That is the deliberate opt-out and
// the callers skip on it.
//
// ok == true means the endpoints are known, which is a STATEMENT THAT A STACK
// EXISTS. From that point on, unreachable is a failure, never a skip.
//
// A PARTIAL declaration (one SPIKE_* set, the other unresolvable) is treated as
// intent-declared-but-unmet and fails here rather than skipping: someone typed
// a variable, so they meant to run.
func resolveSpikeConfig(t *testing.T) (spikeConfig, bool) {
	t.Helper()

	cfg := spikeConfig{
		dbURL:   os.Getenv(spikeDBURLEnv),
		restURL: os.Getenv(spikeRESTURLEnv),
		secret:  env("SPIKE_JWT_SECRET", defaultSpikeSecret),
	}
	declared := cfg.dbURL != "" || cfg.restURL != ""

	origins := []string{}
	if declared {
		origins = append(origins, "SPIKE_* env")
	}

	var unresolved []string
	if cfg.dbURL == "" {
		if port, err := spikeComposePort(spikeDBService, spikeDBPort); err != nil {
			unresolved = append(unresolved, err.Error())
		} else {
			cfg.dbURL = fmt.Sprintf("postgres://supabase_admin:%s@127.0.0.1:%s/postgres",
				spikeDBSuperPassword, port)
			origins = append(origins, "docker compose port "+spikeDBService)
		}
	}
	if cfg.restURL == "" {
		if port, err := spikeComposePort(spikeRESTService, spikeRESTPort); err != nil {
			unresolved = append(unresolved, err.Error())
		} else {
			cfg.restURL = "http://127.0.0.1:" + port
			origins = append(origins, "docker compose port "+spikeRESTService)
		}
	}

	resolved := cfg.dbURL != "" && cfg.restURL != ""

	switch spikeResolution(declared, resolved, spikeSubstrateOptional()) {
	case spikeGateRun:
		cfg.origin = strings.Join(origins, " + ")
		return cfg, true

	case spikeGateSkip:
		// The ONLY door. Reached only with HQ_SYNC_SUBSTRATE_OPTIONAL=1 set.
		return spikeConfig{}, false
	}

	// spikeGateFail. Two shapes, and the message must tell them apart or the
	// person reading it at 3am cannot act on it.
	if declared {
		t.Fatalf("a SPIKE_* endpoint was set, so a SUBSTRATE RUN WAS INTENDED, but the "+
			"rest of the stack could not be resolved:\n  %s\n\n"+
			"This is a FAILURE and not a skip on purpose — see the banner in "+
			"spikestack_gate_test.go. Set both %s and %s, or unset both and bring the "+
			"stack up with:\n"+
			"  docker compose -p %s -f docker-compose.supabase.yml up -d",
			strings.Join(unresolved, "\n  "),
			spikeDBURLEnv, spikeRESTURLEnv, spikeComposeProject)
	}

	// 🛑 B-36. This branch used to `return spikeConfig{}, false` and the package
	// printed `ok` in 0.014s having run no attack variant at all.
	t.Fatalf("🛑 THE SYNC SUBSTRATE COULD NOT BE RESOLVED, and no opt-out was declared:\n  %s\n\n"+
		"This is a FAILURE and NOT a skip (B-36). This package carries the row-visibility and "+
		"JWT-bridge attack suites — the ONLY evidence in this repository that the sync substrate's "+
		"RLS refuses anything. Skipping them silently is how they were found running on NO machine "+
		"while `go test` printed `ok`, and the failure above is reached by a broken docker just as "+
		"readily as by a deliberate choice. Those are not the same event.\n\n"+
		"Bring the stack up:\n"+
		"  docker compose -p %s -f docker-compose.supabase.yml up -d\n"+
		"or point at one elsewhere with %s and %s.\n\n"+
		"If you genuinely have no substrate and accept that THIS PACKAGE THEN PROVES NOTHING about "+
		"row visibility, say so explicitly:\n"+
		"  %s=1 go test ./internal/sync/\n"+
		"🛑 A run carrying that variable is NOT evidence for any security gate. Do not cite it as one.",
		strings.Join(unresolved, "\n  "),
		spikeComposeProject, spikeDBURLEnv, spikeRESTURLEnv, spikeOptionalEnv)
	return spikeConfig{}, false
}

// spikeSubstrateOptional reports the explicit opt-out. `== "1"` and not
// truthiness: HQ_SYNC_SPIKE_LIVE already taught this package that a permissive
// truthiness table on a gate variable is a foot-gun (TestSpikeLiveRequested), and
// a variable whose whole job is to be deliberate should not be satisfiable by
// `HQ_SYNC_SUBSTRATE_OPTIONAL=0` or by an accidental empty export.
func spikeSubstrateOptional() bool { return os.Getenv(spikeOptionalEnv) == "1" }

// spikeUnreachableReason renders the single failure message used at every
// conversion site, so they cannot drift into several different explanations.
// Deliberately shaped like testdb.Reason, which is the same gate for the same
// reason one layer down.
func spikeUnreachableReason(cfg spikeConfig, what, target, stage string, cause error) string {
	return fmt.Sprintf(
		"the sync substrate is CONFIGURED (%s -> %s), so a LIVE SUBSTRATE RUN was "+
			"intended, but its %s is not reachable at %s (%s failed): %v\n\n"+
			"This is a FAILURE and not a skip on purpose. A skip here prints nothing "+
			"without -v, so an intended run of the row-visibility / JWT-bridge attack "+
			"suites silently degrades to whatever hermetic coverage this package happens "+
			"to have and still reports `ok` — which is exactly how these suites were "+
			"found skipping on every machine (F1, run overnight-20260801). With them "+
			"skipped there is NO row-visibility evidence in the tree at all.\n\n"+
			"Bring the stack up:\n"+
			"  docker compose -p %s -f docker-compose.supabase.yml up -d\n"+
			"or unset %s and %s to deliberately run only the tests that need no substrate.",
		cfg.origin, target, what, target, stage, cause,
		spikeComposeProject, spikeDBURLEnv, spikeRESTURLEnv)
}

// dialSpikeDB is the second arm of the gate for Postgres. pgxpool.New is lazy,
// so a dead stack surfaces at Ping, not at New — the same trap internal/testdb
// documents.
func dialSpikeDB(t *testing.T, ctx context.Context, cfg spikeConfig) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(ctx, cfg.dbURL)
	if err != nil {
		t.Fatal(spikeUnreachableReason(cfg, "Postgres", redactDSN(cfg.dbURL), "connect", err))
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		t.Fatal(spikeUnreachableReason(cfg, "Postgres", redactDSN(cfg.dbURL), "ping", err))
	}
	return pool
}

// requireSpikeREST is the same arm for PostgREST. Every attack variant in both
// suites is an HTTP request, so a REST container that is down makes the suites
// fail row by row with a connection error and no explanation; one dial up front
// says what actually happened.
func requireSpikeREST(t *testing.T, cfg spikeConfig) {
	t.Helper()

	u, err := url.Parse(cfg.restURL)
	if err != nil {
		t.Fatalf("unparseable substrate REST URL %q (%s): %v", cfg.restURL, cfg.origin, err)
	}
	host := u.Host
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "80")
	}
	c, err := net.DialTimeout("tcp", host, 2*time.Second)
	if err != nil {
		t.Fatal(spikeUnreachableReason(cfg, "PostgREST", cfg.restURL, "dial", err))
	}
	_ = c.Close()
}

// redactDSN keeps a password out of a failure message that will be pasted into
// a run log. The spike password is a committed throwaway, but SPIKE_DB_URL can
// carry any DSN a human exports.
func redactDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil || u.User == nil {
		return dsn
	}
	if _, hasPw := u.User.Password(); hasPw {
		// ASCII, not an ellipsis: url.String() percent-encodes the replacement,
		// and "%E2%80%A6" in a failure message reads like part of the DSN.
		u.User = url.UserPassword(u.User.Username(), "REDACTED")
	}
	return u.String()
}

// ── The asymmetry, as a table ──────────────────────────────────────────────

type spikeGateVerdict string

const (
	spikeGateRun  spikeGateVerdict = "run"
	spikeGateSkip spikeGateVerdict = "skip"
	spikeGateFail spikeGateVerdict = "fail"
)

// spikeGate is the decision the two arms above implement, extracted so it can
// be asserted rather than described. Modelled on TestSpikeLiveRequested's pin of
// HQ_SYNC_SPIKE_LIVE's truthiness table: a two-line function guarding a
// foot-gun that has already fired.
func spikeGate(configured, reachable bool) spikeGateVerdict {
	switch {
	case !configured:
		return spikeGateSkip
	case !reachable:
		return spikeGateFail
	default:
		return spikeGateRun
	}
}

// spikeResolution is the decision resolveSpikeConfig makes BEFORE any socket is
// opened: given what the environment said and what the compose project answered,
// does this run go ahead, opt out, or fail?
//
// 🛑 Extracted as a pure function so it can be ASSERTED, and called by
// resolveSpikeConfig itself so the assertion is about the shipped decision
// rather than a description of it. spikeGate below is its sibling for the
// reachability half; keeping them separate keeps F1's pinned row untouched by
// B-36's change.
//
//	declared  resolved  optOut   verdict
//	────────  ────────  ──────   ───────────────────────────────────────────
//	 -         true      -        RUN    endpoints known; a live run is on
//	 true      false     -        FAIL   F1: intent declared, stack incomplete
//	 false     false     true     SKIP   🛑 THE ONLY SKIP DOOR
//	 false     false     false    FAIL   🛑 B-36: this used to be a SKIP
func spikeResolution(declared, resolved, optOut bool) spikeGateVerdict {
	switch {
	case resolved:
		return spikeGateRun
	case declared:
		return spikeGateFail
	case optOut:
		return spikeGateSkip
	default:
		return spikeGateFail
	}
}

// TestSpikeResolution_OptOutIsTheOnlySkipDoor is B-36's pin.
//
// 🛑 IT GUARDS ITSELF, because this repo has already shipped three guards that
// printed PASS against an empty or mis-scoped subject set (B-22/B-23/B-24) and
// this is a guard on a guard on a security suite. A table test that enumerated
// the wrong rows, or an implementation that returned `skip` for everything,
// would satisfy a naive row-by-row loop. So the test additionally asserts the
// SHAPE of the table it just walked: all eight combinations are covered, and
// EXACTLY ONE of them is a skip.
func TestSpikeResolution_OptOutIsTheOnlySkipDoor(t *testing.T) {
	type row struct{ declared, resolved, optOut bool }
	want := map[row]spikeGateVerdict{
		// resolved — the endpoints are known, so the run is on regardless of
		// what anyone declared or opted out of. Opting out does not turn a
		// working stack off; it only excuses a missing one.
		{false, true, false}: spikeGateRun,
		{false, true, true}:  spikeGateRun,
		{true, true, false}:  spikeGateRun,
		{true, true, true}:   spikeGateRun,

		// declared but unresolvable — F1's row, unchanged by this card. Someone
		// typed a SPIKE_* variable, so they meant to run.
		{true, false, false}: spikeGateFail,
		{true, false, true}:  spikeGateFail,

		// 🛑 THE TWO ROWS B-36 IS ABOUT. Same observable state — nothing
		// resolved, nothing declared — and they must now differ.
		{false, false, true}:  spikeGateSkip, // typed the opt-out: deliberate
		{false, false, false}: spikeGateFail, // 🛑 was SKIP; that was the bug
	}

	if len(want) != 8 {
		t.Fatalf("the table enumerates %d of 8 combinations — an unenumerated row is "+
			"an unguarded one", len(want))
	}

	skips := 0
	for r, expect := range want {
		got := spikeResolution(r.declared, r.resolved, r.optOut)
		if got != expect {
			t.Errorf("spikeResolution(declared=%v, resolved=%v, optOut=%v) = %q, want %q",
				r.declared, r.resolved, r.optOut, got, expect)
		}
		if got == spikeGateSkip {
			skips++
		}
	}

	// 🛑 The guard's own guard. An implementation that skipped on every row
	// would pass a loop that only compared against a table someone had edited to
	// match it. This line cannot be satisfied that way.
	if skips != 1 {
		t.Errorf("🛑 %d of 8 resolution outcomes are a SKIP, want exactly 1. "+
			"%s=1 is supposed to be the ONLY door out of running the attack suites; "+
			"any second door is B-36 reopened under a different name.", skips, spikeOptionalEnv)
	}
}

// TestSpikeSubstrateOptional_IsExplicit pins the truthiness table of the opt-out
// itself. A gate variable that answers to "0", "false" or "" is a gate that
// opens by accident — the same foot-gun HQ_SYNC_SPIKE_LIVE has its own pin for.
func TestSpikeSubstrateOptional_IsExplicit(t *testing.T) {
	for _, tc := range []struct {
		set  string
		want bool
	}{
		{"1", true},
		{"", false},
		{"0", false},
		{"true", false},  // 🛑 deliberately NOT accepted
		{"yes", false},   // 🛑 ditto
		{" 1", false},    // no trimming; a stray space is not a decision
		{"11", false},
	} {
		t.Setenv(spikeOptionalEnv, tc.set)
		if got := spikeSubstrateOptional(); got != tc.want {
			t.Errorf("%s=%q -> spikeSubstrateOptional() = %v, want %v",
				spikeOptionalEnv, tc.set, got, tc.want)
		}
	}
}

// TestSpikeGate_Asymmetry pins the one row that F1 was about: CONFIGURED plus
// UNREACHABLE must be `fail`. If a future edit makes that row `skip`, the two
// attack suites go back to reporting `ok` while proving nothing, and this test
// is what says so.
func TestSpikeGate_Asymmetry(t *testing.T) {
	for _, tc := range []struct {
		configured, reachable bool
		want                  spikeGateVerdict
	}{
		{false, false, spikeGateSkip}, // nobody asked for a stack
		{false, true, spikeGateSkip},  // ditto; reachability is moot
		{true, false, spikeGateFail},  // 🛑 THE ROW. Never a skip.
		{true, true, spikeGateRun},
	} {
		if got := spikeGate(tc.configured, tc.reachable); got != tc.want {
			t.Errorf("spikeGate(configured=%v, reachable=%v) = %q, want %q",
				tc.configured, tc.reachable, got, tc.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// THE COUNT — "59 subtests" ASSERTED RATHER THAN INFERRED (Q-KR1's residual)
// ═══════════════════════════════════════════════════════════════════════════
//
// Everything above answers "did the suite get to run?". NOTHING above, and
// nothing anywhere else in the tree before this block, answered "did all of it
// run?" — and those are different questions with the same green.
//
// The standing evidence rule (ledger T-29, decision 108, kept with amendments by
// decision 116) says gate evidence for this package must cite
// `-run TestRowVisibilityRLS -v` output showing **59** subtests ran. That number
// lived ONLY in prose: in the ladder, in slates, in run reports. `grep -n '59'`
// across this package's tests returned a JWT secret and a hex literal and
// nothing else. So the number was RE-COUNTED BY A HUMAN, off a log, once a
// night — which is exactly the shape of check this repo keeps finding has gone
// quietly empty (B-22 / B-23 / B-24, and B-36 one layer up).
//
// 🛑 THE FAILURE MODE THIS CLOSES IS NOT "THE SUITE BREAKS". A broken subtest
// reds the package and everyone sees it. The failure mode is a subtest that
// stops being REGISTERED — deleted in a refactor, commented out during a debug
// session and not restored, moved inside an `if` that is now always false, or
// dropped off the end of a `for … range` case list. All of those leave a green
// `--- PASS` wall and a package that says `ok`. The suite gets smaller and
// nothing says so. A security suite that can silently shrink is a security
// suite whose coverage claim expires without notice.
//
// Two independent counters assert it, deliberately not one:
//
//	STRUCTURAL   parses rowvisibility_rls_test.go and counts the t.Run
//	             registrations in the source. Needs NO substrate, so the count
//	             is still guarded on a machine that legitimately opted out —
//	             which is the machine most likely to lose a case unnoticed.
//
//	EXECUTION    re-runs `go test -run '^TestRowVisibilityRLS$' -v` as a
//	             subprocess and counts the depth-1 subtests the RUNTIME
//	             actually reported. This is the one Q-KR1 asked for: it is the
//	             same artifact the ladder cites, read by a machine instead of
//	             by a person at 3am.
//
// They can disagree, and their disagreeing is informative: source that
// registers 59 while the runtime reports 58 means a registration is behind a
// condition that was false. Neither counter alone catches that.

// wantRowVisibilitySubtests is the count the gate ladder cites.
//
// 🛑 IF YOU CHANGED TestRowVisibilityRLS AND LANDED HERE: adding or removing a
// subtest is allowed and expected — bump this constant IN THE SAME COMMIT and
// say so in your report. What is NOT allowed is deleting or relaxing the
// assertion to make the red go away: the whole point is that the suite cannot
// change size in silence. If you cannot reconcile the number, that is a finding
// for triage, not a waiver.
const wantRowVisibilitySubtests = 59

// rowVisibilitySourceFile is the file the structural counter reads. `go test`
// runs with the package directory as cwd, so this needs no path resolution.
const rowVisibilitySourceFile = "rowvisibility_rls_test.go"

// gateChildEnv marks a `go test` process this file spawned. Every subprocess
// test below refuses to run when it is set — without this guard, the child
// compiles the same package, runs these same tests, and forks again.
const gateChildEnv = "HQ_SYNC_GATE_CHILD"

// ── STRUCTURAL ─────────────────────────────────────────────────────────────

// rvTopLevelSubtestCount counts the top-level `t.Run(...)` registrations inside
// a node, expanding `for … range []T{…}` loops by the length of their case list.
//
// Two rules do all the work:
//
//   - DO NOT DESCEND INTO A FUNC LITERAL. A t.Run inside a closure is a NESTED
//     subtest (or a helper's), not a top-level registration of the suite. This
//     is what makes the count mean "how many cases does the suite have" rather
//     than "how many times does the token t.Run appear".
//
//   - A `for … range` over a COMPOSITE LITERAL multiplies. The V15/V16/V17/V20
//     block registers one t.Run per element of a four-element case list; losing
//     an element there is precisely the silent shrink this is here to catch, and
//     a counter that scored that block as 1 would not see it.
//
// A range over anything whose length is not visible in the source (a variable, a
// function call) is reported through `unknown` rather than guessed. Guessing
// would make the counter agree with a source it can no longer read.
func rvTopLevelSubtestCount(fset *token.FileSet, node ast.Node, mult int, unknown *[]string) int {
	count := 0
	ast.Inspect(node, func(n ast.Node) bool {
		if n == nil || n == node {
			return true
		}
		switch cur := n.(type) {
		case *ast.FuncLit:
			return false

		case *ast.RangeStmt:
			lit, ok := cur.X.(*ast.CompositeLit)
			if !ok || len(lit.Elts) == 0 {
				*unknown = append(*unknown, fset.Position(cur.Pos()).String())
				return false
			}
			count += rvTopLevelSubtestCount(fset, cur.Body, mult*len(lit.Elts), unknown)
			return false

		case *ast.CallExpr:
			if isTRunCall(cur) {
				count += mult
			}
			return true
		}
		return true
	})
	return count
}

// isTRunCall reports whether a call is literally `t.Run(...)`.
func isTRunCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Run" {
		return false
	}
	recv, ok := sel.X.(*ast.Ident)
	return ok && recv.Name == "t"
}

// TestRowVisibilitySubtestCount_Structural asserts the SOURCE still registers
// the number of cases the gate ladder cites — with no substrate, no docker and
// no network.
func TestRowVisibilitySubtestCount_Structural(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, rowVisibilitySourceFile, nil, 0)
	if err != nil {
		t.Fatalf("cannot parse %s — the structural half of the count assertion is "+
			"blind without it: %v", rowVisibilitySourceFile, err)
	}

	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Recv == nil && fn.Name.Name == "TestRowVisibilityRLS" {
			body = fn.Body
			break
		}
	}
	if body == nil {
		t.Fatalf("🛑 TestRowVisibilityRLS is not declared in %s. It is the row-visibility "+
			"attack suite and the ONLY read/write RLS evidence in this repository; if it moved, "+
			"move this assertion with it rather than deleting it.", rowVisibilitySourceFile)
	}

	var unknown []string
	got := rvTopLevelSubtestCount(fset, body, 1, &unknown)

	if len(unknown) > 0 {
		t.Fatalf("🛑 %d loop(s) in TestRowVisibilityRLS range over something whose length is "+
			"not visible in the source, so the case list can now shrink without this counter "+
			"noticing:\n  %s\nKeep the case list a composite literal, or teach "+
			"rvTopLevelSubtestCount to read the new shape.",
			len(unknown), strings.Join(unknown, "\n  "))
	}

	if got != wantRowVisibilitySubtests {
		t.Errorf("🛑 TestRowVisibilityRLS registers %d top-level subtests, want %d.\n\n"+
			"The gate ladder, the slates and every run report for this package cite %d as the "+
			"number of attack variants this suite runs — that number is a COVERAGE CLAIM about "+
			"the sync substrate's RLS, and it is now wrong.\n\n"+
			"If you deliberately added or removed a variant, bump wantRowVisibilitySubtests in "+
			"%s and say so in your card report. If you did NOT touch the case list, a "+
			"registration has gone missing — find it before this package is cited as evidence "+
			"again.",
			got, wantRowVisibilitySubtests, wantRowVisibilitySubtests, "spikestack_gate_test.go")
	}
}

// TestRVTopLevelSubtestCount_CountsWhatItClaims is the guard's own guard.
//
// 🛑 A counter that returned a constant, ignored `for … range` expansion, or
// descended into t.Run closures would still print PASS above as long as the two
// errors happened to cancel — and this repo has shipped three checks that passed
// against a subject set that was wrong or empty (B-22/B-23/B-24). So the walker
// is exercised against a synthetic function whose true answer is known by
// construction and whose shape contains every trap the real file contains.
func TestRVTopLevelSubtestCount_CountsWhatItClaims(t *testing.T) {
	const src = `package p

import "testing"

func TestFixture(t *testing.T) {
	t.Run("one", func(t *testing.T) {
		// 🛑 nested: must NOT be counted
		t.Run("nested-a", func(t *testing.T) {})
		for _, x := range []string{"p", "q", "r"} {
			t.Run("nested-"+x, func(t *testing.T) {})
		}
	})
	t.Run("two", func(t *testing.T) {})
	for _, c := range []struct{ n string }{{"a"}, {"b"}, {"c"}, {"d"}} {
		t.Run("looped/"+c.n, func(t *testing.T) {})
	}
	helper(t)
}

func helper(t *testing.T) { t.Run("in-a-helper", func(t *testing.T) {}) }
`
	// 2 direct + (1 loop × 4 elements) = 6. The three nested registrations and
	// the one in helper() are correctly invisible.
	const wantFixture = 6

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "fixture_test.go", src, 0)
	if err != nil {
		t.Fatalf("the fixture itself does not parse: %v", err)
	}
	var body *ast.BlockStmt
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "TestFixture" {
			body = fn.Body
		}
	}
	if body == nil {
		t.Fatal("fixture lost its TestFixture declaration")
	}

	var unknown []string
	if got := rvTopLevelSubtestCount(fset, body, 1, &unknown); got != wantFixture {
		t.Errorf("🛑 rvTopLevelSubtestCount = %d on a fixture whose answer is %d by "+
			"construction. The structural count above is therefore not measuring what it "+
			"says it measures.", got, wantFixture)
	}
	if len(unknown) != 0 {
		t.Errorf("fixture reported %d unreadable range(s), want 0: %v", len(unknown), unknown)
	}

	// And the counter must NOTICE an unreadable case list rather than scoring it 0.
	const opaque = `package p

import "testing"

func TestOpaque(t *testing.T) {
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {})
	}
}
`
	fset2 := token.NewFileSet()
	file2, err := parser.ParseFile(fset2, "opaque_test.go", opaque, 0)
	if err != nil {
		t.Fatalf("the opaque fixture does not parse: %v", err)
	}
	var body2 *ast.BlockStmt
	for _, decl := range file2.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "TestOpaque" {
			body2 = fn.Body
		}
	}
	var unknown2 []string
	rvTopLevelSubtestCount(fset2, body2, 1, &unknown2)
	if len(unknown2) != 1 {
		t.Errorf("🛑 a range over a non-literal was not reported as unreadable (%d reports, "+
			"want 1) — the structural counter would silently under-count a case list it "+
			"cannot see.", len(unknown2))
	}
}

// ── EXECUTION ──────────────────────────────────────────────────────────────

// rvDepthOneSubtest matches the result line `go test -v` prints for a DEPTH-1
// subtest, and only for a depth-1 subtest.
//
// 🛑 Depth cannot be read off the subtest NAME here. This suite's names contain
// slashes on purpose ("FLOOR/HQ's three views are populated", "V19/auth.uid()…"),
// so `TestRowVisibilityRLS/FLOOR/HQ's_three_views…` is ONE level deep despite
// carrying two separators. `go test`'s indentation is the only reliable signal:
// four spaces per level. PASS, FAIL and SKIP are all matched — counting only
// PASS would make a failing suite look like a shrinking one.
var rvDepthOneSubtest = regexp.MustCompile(`(?m)^ {4}--- (?:PASS|FAIL|SKIP): TestRowVisibilityRLS/`)

// goToolPath finds the `go` binary for a subprocess. PATH first, GOROOT as the
// fallback for a caller who invoked an absolute `go` without exporting it — a
// real shape in this repo, where the non-interactive shell does not carry Go.
func goToolPath() (string, error) {
	if p, err := exec.LookPath("go"); err == nil {
		return p, nil
	}
	p := filepath.Join(runtime.GOROOT(), "bin", "go")
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("no `go` on PATH and none at %s: %w", p, err)
	}
	return p, nil
}

// childEnv builds an environment for a nested `go test`, with `drop` removed and
// `set` applied. The child is always marked so it cannot recurse.
func childEnv(drop []string, set map[string]string) []string {
	skip := map[string]bool{gateChildEnv: true}
	for _, k := range drop {
		skip[k] = true
	}
	for k := range set {
		skip[k] = true
	}
	out := []string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 && skip[kv[:i]] {
			continue
		}
		out = append(out, kv)
	}
	out = append(out, gateChildEnv+"=1")
	for k, v := range set {
		out = append(out, k+"="+v)
	}
	return out
}

// runNestedRowVisibility runs `go test -run '^TestRowVisibilityRLS$' -v` in this
// package directory and returns its combined output and its REAL exit code.
func runNestedRowVisibility(t *testing.T, env []string) (string, int) {
	t.Helper()

	goBin, err := goToolPath()
	if err != nil {
		t.Fatalf("cannot locate the go tool to re-run TestRowVisibilityRLS: %v", err)
	}

	// -count=1 is load-bearing twice over: it defeats the test cache (a CACHED
	// exit 0 would make the asymmetry assertion below vacuously true) and it
	// forces the substrate gate to be re-evaluated against the env we just built.
	cmd := exec.Command(goBin, "test", "-count=1", "-timeout=5m",
		"-run", "^TestRowVisibilityRLS$", "-v", ".")
	cmd.Env = env
	out, err := cmd.CombinedOutput()

	code := 0
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			code = ee.ExitCode()
		} else {
			t.Fatalf("nested `go test` could not be run at all (this is a toolchain "+
				"problem, not a test result): %v\n%s", err, out)
		}
	}
	return string(out), code
}

// TestRowVisibilitySubtestCount_Executed is Q-KR1's residual, closed.
//
// The gate ladder's evidence line for this package is a `-run TestRowVisibilityRLS -v`
// run showing 59 subtests. This test IS that run, with the counting done by the
// suite instead of by a person reading a log — so "59 ran" is asserted, not
// inferred from a wall of green.
func TestRowVisibilitySubtestCount_Executed(t *testing.T) {
	if os.Getenv(gateChildEnv) == "1" {
		t.Skipf("%s=1 — this process IS the nested run; not forking again", gateChildEnv)
	}

	// Reuse the SHIPPED gate rather than a copy of it: no substrate and no
	// opt-out fails here exactly as it fails everywhere else in this package.
	cfg, ok := resolveSpikeConfig(t)
	if !ok {
		t.Skipf("%s=1 — the count cannot be asserted BY EXECUTION without a substrate. "+
			"TestRowVisibilitySubtestCount_Structural still holds the source-side count, but "+
			"a run carrying this variable is NOT evidence that %d attack variants executed.",
			spikeOptionalEnv, wantRowVisibilitySubtests)
	}

	// Hand the child the endpoints we already resolved: it skips a second pair of
	// `docker compose port` shell-outs and, more importantly, cannot end up
	// pointed at a different stack than the one this assertion is about.
	out, code := runNestedRowVisibility(t, childEnv(
		[]string{spikeOptionalEnv},
		map[string]string{spikeDBURLEnv: cfg.dbURL, spikeRESTURLEnv: cfg.restURL},
	))

	got := len(rvDepthOneSubtest.FindAllString(out, -1))

	if code != 0 {
		t.Fatalf("🛑 the nested TestRowVisibilityRLS run exited %d — the count below (%d) is "+
			"not evidence of anything until the suite is green again:\n%s", code, got, out)
	}

	if got != wantRowVisibilitySubtests {
		t.Errorf("🛑 TestRowVisibilityRLS reported %d depth-1 subtests AT RUNTIME, want %d.\n\n"+
			"This is the number the gate ladder cites as this package's evidence. A suite that "+
			"quietly loses cases keeps printing `--- PASS` and `ok`; this line is the only thing "+
			"that says the subject set changed size.\n\n"+
			"If the structural counter agrees with %d and this one does not, a registration is "+
			"behind a condition that is now false — that is a coverage hole, not a bookkeeping "+
			"error.\n\nfull output:\n%s", got, wantRowVisibilitySubtests, got, out)
	}
}

// ── THE EXIT-CODE ASYMMETRY, AS A TEST RATHER THAN AS A BANNER ─────────────

// TestSubstrateGate_ExitCodeAsymmetry pins B-36's actual observable.
//
// TestSpikeResolution_OptOutIsTheOnlySkipDoor pins the DECISION FUNCTION. That is
// necessary and it is not sufficient: what B-36 was about is the EXIT CODE `go
// test` hands the gate ladder, and between `spikeResolution` and that exit code
// sit resolveSpikeConfig's switch, the t.Fatalf, and rvConnect's skip. Any of
// those could be rewired — a `t.Skip` reintroduced on the fail branch, an
// `if err != nil { return cfg, false }` added for "robustness" — and every
// existing test in this file would still pass while the package went back to
// printing `ok` in 0.014s having proved nothing.
//
// So this asserts the property end to end, the way the bug was originally
// measured: a `docker` on PATH that exits 1 (the daemon-down case, verbatim from
// the banner above), and the two arms that must differ.
//
//	no opt-out declared, docker broken  -> NON-ZERO   🛑 the row B-36 fixed
//	HQ_SYNC_SUBSTRATE_OPTIONAL=1        -> ZERO       the deliberate opt-out
//
// It costs two nested `go test` invocations. Both fail fast at the gate — the
// substrate is unreachable by construction, so neither one touches Postgres,
// PostgREST or the network.
func TestSubstrateGate_ExitCodeAsymmetry(t *testing.T) {
	if os.Getenv(gateChildEnv) == "1" {
		t.Skipf("%s=1 — this process IS a nested run; not forking again", gateChildEnv)
	}
	if runtime.GOOS == "windows" {
		t.Skip("the docker shim below is a /bin/sh script")
	}

	// A `docker` that exits non-zero for every invocation. This is not a
	// simulation of the bug's cause — it IS one of the listed causes (docker
	// daemon down), reproduced exactly.
	shimDir := t.TempDir()
	shim := filepath.Join(shimDir, "docker")
	if err := os.WriteFile(shim, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("cannot write the docker shim: %v", err)
	}

	// Prepend the shim, keep the rest of PATH so `go` itself stays reachable.
	brokenPath := shimDir + string(os.PathListSeparator) + os.Getenv("PATH")

	// 🛑 SPIKE_DB_URL / SPIKE_REST_URL must be DROPPED, not overridden. Either one
	// set makes `declared` true, which routes to F1's fail branch instead of
	// B-36's — the same verdict for the wrong reason, and the test would then pass
	// against a tree with B-36 reverted.
	drop := []string{spikeDBURLEnv, spikeRESTURLEnv, spikeOptionalEnv}

	t.Run("no opt-out declared and the substrate cannot be resolved -> NON-ZERO", func(t *testing.T) {
		out, code := runNestedRowVisibility(t, childEnv(drop,
			map[string]string{"PATH": brokenPath}))

		if code == 0 {
			t.Errorf("🛑 B-36 IS BACK. `go test -run TestRowVisibilityRLS` exited 0 with a "+
				"broken docker and NO %s declared. That is the exact measurement in this file's "+
				"banner: the row-visibility attack suite did not run, and the package said `ok`. "+
				"Whatever change made this pass has re-opened the silent-skip door.\n\n%s",
				spikeOptionalEnv, out)
		}
		if !strings.Contains(out, "THE SYNC SUBSTRATE COULD NOT BE RESOLVED") {
			t.Errorf("the run failed (exit %d) but not with the substrate gate's message, so "+
				"this test is no longer measuring the gate:\n%s", code, out)
		}
	})

	t.Run("HQ_SYNC_SUBSTRATE_OPTIONAL=1 -> ZERO", func(t *testing.T) {
		out, code := runNestedRowVisibility(t, childEnv(drop,
			map[string]string{"PATH": brokenPath, spikeOptionalEnv: "1"}))

		if code != 0 {
			t.Errorf("🛑 the deliberate opt-out no longer opens the door: exit %d with %s=1. "+
				"A contributor with no substrate can no longer run this package's hermetic "+
				"tests, which is the arm the gate is supposed to KEEP.\n\n%s",
				code, spikeOptionalEnv, out)
		}
		if !strings.Contains(out, "--- SKIP: TestRowVisibilityRLS") {
			t.Errorf("exit was 0 but TestRowVisibilityRLS did not report a SKIP — a green that "+
				"is not the opt-out's green is a different bug:\n%s", out)
		}
	})
}
