package sync

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
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

const (
	spikeComposeProject = "spike-supabase"

	spikeDBService   = "db"
	spikeDBPort      = "5432"
	spikeRESTService = "rest"
	spikeRESTPort    = "3000"

	spikeDBURLEnv   = "SPIKE_DB_URL"
	spikeRESTURLEnv = "SPIKE_REST_URL"

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

	if cfg.dbURL == "" || cfg.restURL == "" {
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
		return spikeConfig{}, false
	}

	cfg.origin = strings.Join(origins, " + ")
	return cfg, true
}

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
