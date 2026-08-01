package sync

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yumyums/hq/internal/db"
)

// ═══════════════════════════════════════════════════════════════════════════
// Attack-variant suite — ROW VISIBILITY, proved DISCRIMINATING
// ═══════════════════════════════════════════════════════════════════════════
//
// Card `sync-rxdb-row-visibility-rls` (overnight-20260801, B2). Modelled on
// jwtbridge_rls_test.go in this package — the 16/16 suite card
// `sync-jwt-bridge-endpoint` ran — and on tests/grant-enforcement-parity.spec.js
// before it. Its central lesson is the one this file is built around:
//
//	A 200 PROVES NOTHING.
//
// An endpoint that answers is not an endpoint that authorizes. So every scoped
// result below is paired with the two things without which it is worthless:
//
//  1. A POSITIVE half, and not one positive but FOUR — Alice, Bob, Dave and
//     Carol each see a DIFFERENT set of templates. A single positive proves
//     only that somebody can read something; four disjoint ones are what make
//     "the policy discriminates" distinguishable from "the policy is a
//     coin flip that happened to land right".
//
//  2. 🛑 A `service_role` BYPASSRLS CONTROL, taken FIRST and taken AGAIN LAST.
//     The same tables, the same URLs, read with a god-token, returning every
//     row. This is what rules out the boring explanation for every empty
//     result in this file: that the table was empty, or the URL wrong, or
//     PostgREST's schema cache stale. The rows were always there; RLS was
//     hiding them. Delete the control and the whole suite goes vacuous.
//
// ── 🛑 GUARD INTEGRITY — THIS SUITE IS ITSELF A GUARD ─────────────────────
//
// Run `overnight-20260729-2` found three guards in this repo that print PASS
// against an empty or mis-scoped subject set, inside the very checks added to
// cure that (B-22/B-23/B-24). A passing RLS suite against an empty table is
// precisely that failure mode, and this card's whole deliverable is a check.
//
// So this file does not merely include a control — it asserts POPULATION
// FLOORS on every subject set it depends on, and it asserts them on BOTH SIDES
// of the FDW:
//
//	rvAssertHQPopulated    — HQ's three views return rows (the remote side)
//	rvAssertFDWPopulated   — the FOREIGN TABLES return those same rows
//	                         (the wire actually carries them — a foreign table
//	                         over an unreachable server returns an ERROR, but a
//	                         foreign table over the WRONG database returns a
//	                         perfectly calm empty set)
//	CONTROL/0 and CONTROL/Z — the substrate's four tables return rows through
//	                         PostgREST, before and after every variant
//
// A suite that skips because a stack is down is honest. A suite that passes
// because a table is empty is not, and no assertion in this file can be
// satisfied by an empty result.
//
// 🛑 THE SHARPEST VERSION OF THAT, MEASURED RATHER THAN ARGUED. The FDW server
// was deliberately repointed at a migrated-but-empty database — the failure
// mode that returns a calm empty set instead of raising — and the suite re-run:
// TWELVE OF THE NINETEEN NUMBERED ATTACK VARIANTS STILL PASSED (V1-V6, V10,
// V11, V15-V18). Every one of the seven that failed, failed on its POSITIVE
// half, never on its refusal.
//
// So: A REFUSAL-ONLY VARIANT IS BLIND TO AN EMPTY SUBJECT SET. "The attacker
// saw nothing" is satisfied perfectly by a system that shows nobody anything.
// What catches it is an assertion that DEMANDS ROWS — the four positives, the
// two floors, the two BYPASSRLS controls. If you are ever tempted to add a
// variant here without a positive counterpart, that measurement is the reason
// not to.
//
// ── RED-FIRST, which is this card's real gate ─────────────────────────────
//
// Reproduce the red at any time:
//
//	SYNC_RLS_SKIP_POLICIES=1 go test ./internal/sync/ -run TestRowVisibilityRLS -v
//
// The suite then applies 0001 + 0002 (tables + fdw, NO policies), actively
// TEARS RLS BACK DOWN on the four replicated tables, and withholds 0003. Every
// RLS-layer variant fails, loudly, which is what they are supposed to do
// against a database with no policies in it.
//
// 🛑 WHY THE RED IS "RLS OFF" AND NOT "RLS ON, NO POLICIES". Card B1 leaves the
// four tables RLS-enabled with zero policies, which in Postgres is DENY-ALL.
// Running these variants against THAT state would pass every one of them
// VACUOUSLY — a table nobody can read is indistinguishable from a policy that
// works. That is the same bug class as the paragraph above. The red state must
// LEAK, or it is not a red state.
//
// ── Which layer refuses what — stated, not blurred ────────────────────────
//
//	LAYER     variants                                    refused by
//	─────     ───────────────────────────────────────     ─────────────────────
//	JWT       anon · wrong signature · expired ·          PostgREST's verifier.
//	          tampered payload · missing sub              PRE-EXISTING. Green in
//	                                                      the red state too. This
//	                                                      comment is the record
//	                                                      of that, not a claim
//	                                                      that this card wrote
//	                                                      them.
//	GRANT     direct reads of the three foreign tables    table grants (0002 §4).
//	                                                      Also green in the red
//	                                                      state — and included
//	                                                      anyway, because this
//	                                                      card is what PUT HQ's
//	                                                      role map on this server
//	                                                      and owes a proof that
//	                                                      it is not readable.
//	POLICY-   forged insert · forged update ·             the ABSENCE of a
//	ABSENCE   submission_rejections                       policy. A DECISION —
//	                                                      see 0003 §2. LEAKS in
//	                                                      the red state.
//	RLS       cross-template · claim-injection ·          the policies THIS card
//	          revocation replay · live grant ·            writes. The genuinely
//	          submission scoping · draft scoping ·        red set.
//	          orphan response · auth.uid() trap
//
// ── Stack precondition ────────────────────────────────────────────────────
//
// TWO servers must be up:
//
//	the substrate — docker compose -p spike-supabase -f docker-compose.supabase.yml up -d
//	HQ's Postgres — yumyums-dev-pg, host :5433
//
// 🛑 THE COMPOSE FILE PUBLISHES EPHEMERAL HOST PORTS ON PURPOSE, so the
// hard-coded defaults this suite inherited from jwtbridge_rls_test.go were only
// ever right by luck — and in practice never right. That is finding F1 of run
// overnight-20260801: this suite, the card's ENTIRE DELIVERABLE, skipped
// silently while `go test` printed `ok`. The ports are now RESOLVED with
// `docker compose port` (spikestack_gate_test.go); no export is required, and
// SPIKE_DB_URL / SPIKE_REST_URL remain available to point at a stack elsewhere.
//
// 🛑 AND THE GATE IS ASYMMETRIC. No substrate configured at all -> SKIP, which
// is the deliberate opt-out. A substrate that IS configured (resolved from
// compose, or named by SPIKE_*) and does not answer -> FAIL. So does HQ's
// Postgres, once the substrate has been resolved: at that point a live run was
// intended, and half a stack proves nothing. SPIKE_JWT_SECRET is REUSED from
// the jwtbridge suite rather than renamed, so one export drives both.

const (
	// rvHQDatabase is this suite's OWN database on HQ's Postgres, dropped and
	// recreated every run.
	//
	// 🛑 Deliberately NOT hq_test_go. Other packages' TestMains truncate
	// `users`, and this suite seeds users with fixed ids that must survive for
	// the whole run — a shared database makes the subject set someone else's
	// business, and an emptied subject set is exactly what this file exists to
	// make impossible. Dropping it up front is also what guarantees the
	// populations asserted below are THIS RUN'S, not an earlier run's residue.
	//
	// Overridable with HQ_RLS_TEST_DB (see rvHQDatabase()) so two agents running
	// this suite at once on the shared cluster do not DROP each other's database
	// mid-run — the incident behind BACKLOG B-16, where a dropped test database
	// read as a passing suite.
	defaultRVHQDatabase = "hq_test_b2_fdw"

	// rvFDWPassword is a throwaway for a local test database that is dropped at
	// the end of the run. It is NOT a default for anything: migration 0073
	// creates hq_sync_fdw NOLOGIN with no password precisely so no environment
	// inherits a shared committed credential. The suite performing the
	// ALTER ROLE below is the demonstration of the per-environment operator
	// step, and rvRestoreFDWRole undoes it.
	rvFDWPassword = "b2-rowvis-suite-throwaway"

	defaultHQAdminURL = "postgres://yumyums:yumyums@localhost:5433/postgres"
	// defaultFDWHost is HQ as seen FROM INSIDE the substrate container, which
	// is not what your shell means by localhost. Getting this wrong is the
	// single most common way to make the foreign tables silently useless.
	defaultFDWHost = "host.docker.internal"
	defaultFDWPort = "5433"
)

// rvHQDatabase names this suite's throwaway database on HQ's Postgres. It is a
// function and not a constant only so HQ_RLS_TEST_DB can move it: the suite
// DROPs it WITH (FORCE) on the way in, and doing that to a name another
// concurrent run is using is the B-16 incident, not a flake.
func rvHQDatabase() string { return env("HQ_RLS_TEST_DB", defaultRVHQDatabase) }

// ── The fixture's identities ──────────────────────────────────────────────
//
// Four users whose visibility sets are ALL DIFFERENT and NONE nested in a way
// that makes two policies look alike:
//
//	alice  team_member  assigned to tplAlice (as assignee) AND tplApprover (as APPROVER)
//	bob    team_member  assigned to tplBob
//	dave   manager      assigned to NOTHING directly — reaches tplByRole by ROLE
//	carol  admin        assigned to NOTHING at all — sees EVERYTHING by role
//
// Every arm of the ported predicate is therefore load-bearing for exactly one
// identity, and deleting any arm changes exactly one expected set:
//
//	drop the assignee_type='user' disjunct  -> alice and bob go blind
//	drop the assignee_type='role' disjunct  -> dave goes blind
//	drop the admin arm                      -> carol goes blind
//	filter on assignment_role='assignee'    -> alice loses tplApprover ONLY
//
// That last one is the point of tplApprover: INHERITED PROPERTY 1 (an approver
// sees what an assignee sees) is not merely commented, it is the difference
// between a green suite and a red one.
const (
	uAlice = "11111111-1111-4111-8111-000000000001"
	uBob   = "11111111-1111-4111-8111-000000000002"
	uCarol = "11111111-1111-4111-8111-000000000003"
	uDave  = "11111111-1111-4111-8111-000000000004"

	tplAlice    = "22222222-2222-4222-8222-000000000001"
	tplBob      = "22222222-2222-4222-8222-000000000002"
	tplByRole   = "22222222-2222-4222-8222-000000000003"
	tplOrphan   = "22222222-2222-4222-8222-000000000004"
	tplApprover = "22222222-2222-4222-8222-000000000005"

	fldAlice    = "44444444-4444-4444-8444-000000000001"
	fldBob      = "44444444-4444-4444-8444-000000000002"
	fldByRole   = "44444444-4444-4444-8444-000000000003"
	fldOrphan   = "44444444-4444-4444-8444-000000000004"
	fldApprover = "44444444-4444-4444-8444-000000000005"

	// fldGhost exists ONLY on the substrate. HQ has never heard of it, so
	// hq_field_templates cannot resolve it. It is how the suite proves the
	// EXISTS-nesting decision in 0003's hq_can_see_field: written the obvious
	// way, an unresolvable field is visible TO ADMINS (the admin arm never
	// mentions the template); the Go resolver returns nobody. Nobody wins.
	fldGhost = "44444444-4444-4444-8444-00000000dead"

	subAlice  = "55555555-5555-4555-8555-000000000001"
	subBob    = "55555555-5555-4555-8555-000000000002"
	subByRole = "55555555-5555-4555-8555-000000000003"
)

var (
	rvAllTemplateIDs   = []string{tplAlice, tplBob, tplByRole, tplOrphan, tplApprover}
	rvAllSubmissionIDs = []string{subAlice, subBob, subByRole}
	// Every substrate row id, so a refusal can be checked for LEAKAGE rather
	// than merely for a status code. A variant returning 200 with an empty body
	// is refused; a variant returning 200 carrying one of these is not.
	rvAllSeededIDs = func() []string {
		out := append([]string{}, rvAllTemplateIDs...)
		out = append(out, rvAllSubmissionIDs...)
		out = append(out, "resp-alice", "resp-bob", "resp-byrole", "resp-approver", "resp-ghost")
		out = append(out, "rej-alice")
		sort.Strings(out)
		return out
	}()
)

// ── Setup ─────────────────────────────────────────────────────────────────

type rvStack struct {
	*spikeStack
	hq *pgxpool.Pool // HQ's Postgres, connected to rvHQDatabase()
}

// rvConnect brings both servers to a known state, SKIPS if no substrate is
// configured at all, and FAILS if a configured one is not there.
//
// 🛑 THE SUBSTRATE IS RESOLVED FIRST, before a single connection is opened, and
// that ordering is the F1 fix rather than tidiness. Resolving it is what decides
// whether this run was INTENDED to be live; once it is, HQ's Postgres being
// down stops being "a contributor without a database" and becomes "half the
// stack is missing and the deliverable is about to evaporate silently".
// Resolution itself opens no sockets, so nothing about the sequence below moves.
//
// The order of the WORK is load-bearing: HQ must be migrated and seeded BEFORE
// the substrate's foreign tables are created, because 0002's user mapping is
// verified by the population assertions immediately afterwards and a mapping
// pointed at an unmigrated database fails as "relation does not exist" — which
// is at least loud, unlike a mapping pointed at a migrated-but-empty one.
func rvConnect(t *testing.T) *rvStack {
	t.Helper()
	ctx := context.Background()

	// ── Is a substrate configured at all? (no I/O — see above) ───────────
	cfg, ok := resolveSpikeConfig(t)
	if !ok {
		// 🛑 Reached ONLY under HQ_SYNC_SUBSTRATE_OPTIONAL=1 since B-36. Every
		// other unresolvable-substrate path is a t.Fatal inside
		// resolveSpikeConfig, so this line is a declaration, never an accident.
		t.Skipf("%s=1 — skipping, and SKIPPED IS NOT PASSED: with this off there is NO "+
			"row-visibility evidence in the tree at all, for reads OR for writes. Bring the "+
			"stack up with: docker compose -p %s -f docker-compose.supabase.yml up -d",
			spikeOptionalEnv, spikeComposeProject)
	}

	// ── HQ's Postgres ────────────────────────────────────────────────────
	//
	// Unreachable is a FAILURE here, not a skip: the substrate above is
	// configured, so a live run was intended, and this suite reads HQ's live
	// tables THROUGH the substrate — without HQ there is nothing to see through.
	adminURL := env("HQ_ADMIN_DB_URL", defaultHQAdminURL)
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatal(spikeUnreachableReason(cfg, "HQ Postgres (the read-through source)",
			redactDSN(adminURL), "connect", err))
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := admin.Ping(pingCtx); err != nil {
		admin.Close()
		t.Fatal(spikeUnreachableReason(cfg, "HQ Postgres (the read-through source)",
			redactDSN(adminURL), "ping", err))
	}
	defer admin.Close()

	// FORCE because a previous run's pooled connection outliving the test is a
	// flake, not a finding. Dropping rather than truncating is what makes the
	// population floors below a statement about THIS run.
	if _, err := admin.Exec(ctx, `DROP DATABASE IF EXISTS `+rvHQDatabase()+` WITH (FORCE)`); err != nil {
		t.Fatalf("drop %s: %v", rvHQDatabase(), err)
	}
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+rvHQDatabase()); err != nil {
		t.Fatalf("create %s: %v", rvHQDatabase(), err)
	}

	hqURL := strings.Replace(adminURL, "/postgres", "/"+rvHQDatabase(), 1)
	hq, err := pgxpool.New(ctx, hqURL)
	if err != nil {
		t.Fatalf("connect %s: %v", rvHQDatabase(), err)
	}
	t.Cleanup(hq.Close)

	// The real migrations, including 0073. Not a hand-written approximation of
	// HQ's schema: the views under test are the ones the migration ships, and a
	// suite that redefined them would be proving its own definition correct.
	if err := db.Migrate(hq); err != nil {
		t.Fatalf("migrate %s: %v", rvHQDatabase(), err)
	}

	// 🛑 The per-environment operator step 0073 stops short of, performed here
	// so it is DEMONSTRATED rather than described. Roles are cluster-wide, so
	// this is restored on cleanup — a test that leaves a LOGIN-enabled role
	// behind has widened the cluster on its way out.
	if _, err := hq.Exec(ctx,
		`ALTER ROLE hq_sync_fdw LOGIN PASSWORD '`+rvFDWPassword+`'`); err != nil {
		t.Fatalf("enable hq_sync_fdw login: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hq.Exec(context.Background(), `ALTER ROLE hq_sync_fdw NOLOGIN PASSWORD NULL`)
	})

	rvSeedHQ(t, hq)

	// ── The substrate ────────────────────────────────────────────────────
	pool := dialSpikeDB(t, ctx, cfg)
	t.Cleanup(pool.Close)
	requireSpikeREST(t, cfg)

	s := &rvStack{spikeStack: &spikeStack{rest: cfg.restURL, secret: cfg.secret, pool: pool}, hq: hq}

	syncSQL := filepath.Join("..", "..", "..", "sync-schema", "sql")

	// B1's four tables. Idempotent.
	s.applySQL(t, filepath.Join(syncSQL, "0001_sync_tables.sql"))

	// The auth.uid() trap and its fixture, from the card that banked the
	// finding. Applied, not copied: V-TRAP re-proves it against THIS card's
	// policies, so "someone fixed hq_can_see_template to use auth.uid()" is
	// caught by a running test rather than by a comment.
	bridgeSQL := sqlDir(t)
	s.applySQL(t, filepath.Join(bridgeSQL, "hq-bridge-fixture.sql"))
	s.applySQL(t, filepath.Join(bridgeSQL, "hq-bridge-policies.sql"))

	// 0002 — the fdw. Connection parameters as GUCs, in the same session as the
	// file (pgx v5 uses the simple query protocol for argument-less Exec, so a
	// multi-statement script shares one connection and one implicit
	// transaction).
	fdwHost := env("HQ_FDW_HOST", defaultFDWHost)
	fdwPort := env("HQ_FDW_PORT", defaultFDWPort)
	prelude := fmt.Sprintf(
		"set hq_fdw.host = %s; set hq_fdw.port = %s; set hq_fdw.dbname = %s; "+
			"set hq_fdw.username = 'hq_sync_fdw'; set hq_fdw.password = %s;\n",
		quoteLiteral(fdwHost), quoteLiteral(fdwPort),
		quoteLiteral(rvHQDatabase()), quoteLiteral(rvFDWPassword))
	fdwBody, err := os.ReadFile(filepath.Join(syncSQL, "0002_hq_fdw.sql"))
	if err != nil {
		t.Fatalf("read 0002_hq_fdw.sql: %v", err)
	}
	if _, err := pool.Exec(ctx, prelude+string(fdwBody)); err != nil {
		t.Fatalf("apply 0002_hq_fdw.sql: %v\n\n"+
			"🛑 If this is a connection failure, the substrate CONTAINER cannot reach HQ at "+
			"%s:%s. That is the card's PARK trigger — check it before assuming a bug.",
			err, fdwHost, fdwPort)
	}

	rvSeedSubstrate(t, pool)

	// 0003 — the policies. Withheld in red mode.
	policies := filepath.Join(syncSQL, "0003_rls_policies.sql")
	if os.Getenv("SYNC_RLS_SKIP_POLICIES") == "1" {
		t.Logf("SYNC_RLS_SKIP_POLICIES=1 — RED MODE: tearing RLS down and NOT applying %s", policies)
		s.tearDownRowVisRLS(t)
	} else {
		s.applySQL(t, policies)
	}

	// PostgREST caches the schema; the DDL above needs a reload before it is
	// visible. NOTIFY is cheaper than a restart; the sleep is the settle.
	if _, err := pool.Exec(ctx, `notify pgrst, 'reload schema'`); err != nil {
		t.Fatalf("notify pgrst: %v", err)
	}
	time.Sleep(600 * time.Millisecond)
	return s
}

// quoteLiteral is a minimal single-quote escaper for the GUC prelude. The
// values are test-controlled, but building SQL by concatenation without one is
// the habit that eventually meets a value that is not.
func quoteLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// tearDownRowVisRLS restores the RED state on a substrate the policies have
// already run against. Without it, "reproduce the red" would only work on a
// virgin stack — and a red that only reproduces once is a red nobody can check.
func (s *rvStack) tearDownRowVisRLS(t *testing.T) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		drop policy if exists checklist_templates_select   on public.checklist_templates;
		drop policy if exists checklist_submissions_select on public.checklist_submissions;
		drop policy if exists submission_responses_select  on public.submission_responses;
		alter table public.checklist_templates   disable row level security;
		alter table public.checklist_submissions disable row level security;
		alter table public.submission_responses  disable row level security;
		alter table public.submission_rejections disable row level security;
	`)
	if err != nil {
		t.Fatalf("tearDownRowVisRLS: %v", err)
	}
}

// rvSeedHQ writes the permission truth. Everything the policies decide is
// derived from these rows and from nothing else.
func rvSeedHQ(t *testing.T, hq *pgxpool.Pool) {
	t.Helper()
	_, err := hq.Exec(context.Background(), `
		insert into users (id, email, roles, status, first_name, last_name) values
		  ('`+uAlice+`', 'alice@b2.test', array['team_member'], 'active', 'Alice', 'A'),
		  ('`+uBob+`',   'bob@b2.test',   array['team_member'], 'active', 'Bob',   'B'),
		  ('`+uCarol+`', 'carol@b2.test', array['admin'],       'active', 'Carol', 'C'),
		  ('`+uDave+`',  'dave@b2.test',  array['manager'],      'active', 'Dave',  'D');

		insert into checklist_templates (id, name) values
		  ('`+tplAlice+`',    'Alice Opening'),
		  ('`+tplBob+`',      'Bob Opening'),
		  ('`+tplByRole+`',   'Role-Assigned Closing'),
		  ('`+tplOrphan+`',   'Unassigned Orphan'),
		  ('`+tplApprover+`', 'Approver-only Assignment');

		-- 🛑 The four assignment shapes, one per template. tplOrphan gets NONE.
		insert into template_assignments (template_id, assignee_type, assignee_id, assignment_role) values
		  ('`+tplAlice+`',    'user', '`+uAlice+`', 'assignee'),
		  ('`+tplBob+`',      'user', '`+uBob+`',   'assignee'),
		  ('`+tplByRole+`',   'role', 'manager',    'assignee'),
		  -- assignment_role='approver'. If anyone ever "tightens" the view with
		  -- a WHERE assignment_role='assignee', THIS row is what goes dark and
		  -- POSITIVE/alice is what turns red.
		  ('`+tplApprover+`', 'user', '`+uAlice+`', 'approver');

		insert into checklist_sections (id, template_id, title, "order") values
		  ('33333333-3333-4333-8333-000000000001', '`+tplAlice+`',    'S', 0),
		  ('33333333-3333-4333-8333-000000000002', '`+tplBob+`',      'S', 0),
		  ('33333333-3333-4333-8333-000000000003', '`+tplByRole+`',   'S', 0),
		  ('33333333-3333-4333-8333-000000000004', '`+tplOrphan+`',   'S', 0),
		  ('33333333-3333-4333-8333-000000000005', '`+tplApprover+`', 'S', 0);

		insert into checklist_fields (id, section_id, type, label, "order") values
		  ('`+fldAlice+`',    '33333333-3333-4333-8333-000000000001', 'checkbox', 'F', 0),
		  ('`+fldBob+`',      '33333333-3333-4333-8333-000000000002', 'checkbox', 'F', 0),
		  ('`+fldByRole+`',   '33333333-3333-4333-8333-000000000003', 'checkbox', 'F', 0),
		  ('`+fldOrphan+`',   '33333333-3333-4333-8333-000000000004', 'checkbox', 'F', 0),
		  ('`+fldApprover+`', '33333333-3333-4333-8333-000000000005', 'checkbox', 'F', 0);
	`)
	if err != nil {
		t.Fatalf("seed HQ: %v", err)
	}
}

// rvSeedSubstrate writes the replicated rows. Inserted as the substrate's
// superuser, which bypasses RLS — the seed must not be shaped by the policy it
// is used to test.
func rvSeedSubstrate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		delete from public.submission_rejections;
		delete from public.submission_responses;
		delete from public.checklist_submissions;
		delete from public.checklist_templates;

		insert into public.checklist_templates (id, name) values
		  ('`+tplAlice+`', 'Alice Opening'), ('`+tplBob+`', 'Bob Opening'),
		  ('`+tplByRole+`', 'Role-Assigned Closing'), ('`+tplOrphan+`', 'Unassigned Orphan'),
		  ('`+tplApprover+`', 'Approver-only Assignment');

		insert into public.checklist_submissions (id, template_id, template_snapshot, submitted_by) values
		  ('`+subAlice+`',  '`+tplAlice+`',  '{}'::jsonb, '`+uAlice+`'),
		  ('`+subBob+`',    '`+tplBob+`',    '{}'::jsonb, '`+uBob+`'),
		  ('`+subByRole+`', '`+tplByRole+`', '{}'::jsonb, '`+uDave+`');

		-- 🛑 EVERY response below is a DRAFT: submission_id IS NULL. That is the
		-- hard case on purpose — drafts are what a crew member fills offline,
		-- and a policy that scoped by submission_id would return nothing for
		-- all of them while looking perfectly healthy.
		insert into public.submission_responses (id, submission_id, field_id, value, answered_by) values
		  ('resp-alice',    null, '`+fldAlice+`',    'true'::jsonb, '`+uAlice+`'),
		  ('resp-bob',      null, '`+fldBob+`',      'true'::jsonb, '`+uBob+`'),
		  ('resp-byrole',   null, '`+fldByRole+`',   'true'::jsonb, '`+uDave+`'),
		  ('resp-approver', null, '`+fldApprover+`', 'true'::jsonb, '`+uAlice+`'),
		  -- resp-ghost's field does not exist in HQ. Nobody, admins included.
		  ('resp-ghost',    null, '`+fldGhost+`',    'true'::jsonb, '`+uAlice+`');

		insert into public.submission_rejections (id, submission_id, field_id, comment, rejected_by) values
		  ('rej-alice', '`+subAlice+`', '`+fldAlice+`', 'redo', '`+uCarol+`');
	`)
	if err != nil {
		t.Fatalf("seed substrate: %v", err)
	}
}

// ── Tokens ────────────────────────────────────────────────────────────────

// rvToken mints what the REAL bridge would mint. Note what is NOT in the
// signature: nothing about which templates the user can see. Entitlement is
// read live, per row, through the FDW — never from the token.
func (s *rvStack) rvToken(t *testing.T, sub string, hqRoles ...string) string {
	t.Helper()
	if len(hqRoles) == 0 {
		hqRoles = []string{"team_member"}
	}
	return s.mint(t, Claims{
		Sub: sub, Role: SupabaseRole,
		Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
		HQRoles: hqRoles, HQGrants: []string{"operations"},
	})
}

// ── Assertions ────────────────────────────────────────────────────────────

func rvAssertRefused(t *testing.T, variant string, r restResp) {
	t.Helper()
	leaked := []string{}
	for _, id := range r.ids() {
		for _, seeded := range rvAllSeededIDs {
			if id == seeded {
				leaked = append(leaked, id)
			}
		}
	}
	if len(leaked) > 0 {
		t.Errorf("VARIANT %q WAS NOT REFUSED — it read %v.\n  response: %s\n"+
			"  A variant that returns seeded rows has defeated the policy, whatever "+
			"status code it carried.", variant, leaked, r)
		return
	}
	t.Logf("REFUSED  %-52s %s", variant, r)
}

// rvAssertRows is assertRows with one addition that matters here: it REFUSES AN
// EMPTY EXPECTATION. Every positive in this file must name at least one row, or
// it is not a positive — it is the vacuous pass this suite exists to prevent.
func rvAssertRows(t *testing.T, label string, r restResp, want ...string) {
	t.Helper()
	if len(want) == 0 {
		t.Fatalf("%s: rvAssertRows called with an EMPTY expectation. "+
			"Use rvAssertRefused for refusals — a positive that expects nothing proves nothing.", label)
	}
	assertRows(t, label, r, want...)
}

// rvAssertHQPopulated is half of the guard-integrity floor: HQ's three views
// return rows AT ALL. If migration 0073's join is wrong — say the role disjunct
// is inverted — this catches it on the remote side, where the error message
// still says which relation was empty.
func rvAssertHQPopulated(t *testing.T, hq *pgxpool.Pool) {
	t.Helper()
	for _, c := range []struct {
		view string
		min  int
	}{
		{"hq_sync_template_assignees", 4}, // 4 assignment rows, all resolvable
		{"hq_sync_user_roles", 4},         // 4 users
		{"hq_sync_field_templates", 5},    // 5 fields
	} {
		var n int
		if err := hq.QueryRow(context.Background(),
			`select count(*) from `+c.view).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", c.view, err)
		}
		if n < c.min {
			t.Fatalf("🛑 SUBJECT SET EMPTY OR SHORT: HQ view %s has %d rows, expected >= %d. "+
				"Every refusal in this suite would pass vacuously against this. "+
				"The suite is not evidence until this line does.", c.view, n, c.min)
		}
		t.Logf("HQ POPULATION   %-32s %d rows", c.view, n)
	}
}

// rvAssertFDWPopulated is the other half, and it is the one that cannot be
// skipped.
//
// 🛑 A foreign table pointed at an UNREACHABLE server raises. A foreign table
// pointed at the WRONG DATABASE — a live server, a valid mapping, a database
// with no such rows — returns a perfectly calm EMPTY SET, and every policy
// built on it then denies everyone, and every attack variant in this file
// passes. That is this repo's characteristic bug, arriving through the one
// mechanism this card added. Asserting HQ is populated does not catch it;
// only reading THROUGH THE WIRE does.
func rvAssertFDWPopulated(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, c := range []struct {
		table string
		min   int
	}{
		{"hq_template_assignees", 4},
		{"hq_user_roles", 4},
		{"hq_field_templates", 5},
	} {
		var n int
		if err := pool.QueryRow(context.Background(),
			`select count(*) from public.`+c.table).Scan(&n); err != nil {
			t.Fatalf("count foreign table %s: %v\n"+
				"🛑 If this is a connection error the substrate cannot reach HQ — the card's PARK trigger.",
				c.table, err)
		}
		if n < c.min {
			t.Fatalf("🛑 THE FDW IS RETURNING AN EMPTY SET: foreign table %s has %d rows, expected >= %d, "+
				"while HQ's matching view is populated. The mapping is live but pointed at the WRONG "+
				"DATABASE. Every policy below would deny everyone and every variant would pass "+
				"VACUOUSLY.", c.table, n, c.min)
		}
		t.Logf("FDW POPULATION  %-32s %d rows across the wire", c.table, n)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// The suite
// ═══════════════════════════════════════════════════════════════════════════

func TestRowVisibilityRLS(t *testing.T) {
	s := rvConnect(t)

	const (
		templates   = "/checklist_templates?select=id&order=id"
		submissions = "/checklist_submissions?select=id&order=id"
		responses   = "/submission_responses?select=id&order=id"
		rejections  = "/submission_rejections?select=id&order=id"
	)

	// ── FLOOR ────────────────────────────────────────────────────────────
	// Run before anything else. If either of these fails, nothing after it is
	// evidence, and the suite says so rather than printing a green PASS.
	t.Run("FLOOR/HQ's three views are populated", func(t *testing.T) {
		rvAssertHQPopulated(t, s.hq)
	})
	t.Run("FLOOR/the foreign tables carry those rows across the wire", func(t *testing.T) {
		rvAssertFDWPopulated(t, s.pool)
	})

	// ── CONTROL 0 ────────────────────────────────────────────────────────
	// 🛑 Everything below is meaningless without this. `service_role` has
	// BYPASSRLS: handed that claim, PostgREST SET ROLEs into it and the same
	// tables at the same URLs return every row. That rules out the boring
	// explanations for every empty result in this file — empty table, wrong
	// URL, stale schema cache.
	t.Run("CONTROL/0 service_role BYPASSRLS proves the rows are there", func(t *testing.T) {
		god := s.serviceRoleControl(t)
		rvAssertRows(t, "service_role sees ALL templates", s.get(t, templates, god), rvAllTemplateIDs...)
		rvAssertRows(t, "service_role sees ALL submissions", s.get(t, submissions, god), rvAllSubmissionIDs...)
		rvAssertRows(t, "service_role sees ALL responses", s.get(t, responses, god),
			"resp-alice", "resp-approver", "resp-bob", "resp-byrole", "resp-ghost")
		rvAssertRows(t, "service_role sees ALL rejections", s.get(t, rejections, god), "rej-alice")
	})

	// ── POSITIVES ────────────────────────────────────────────────────────
	// Four identities, four DIFFERENT answers. This block is what makes the
	// word "discriminating" mean something: no single broken policy — deny-all,
	// allow-all, identity-only, role-only — reproduces all four of these.

	t.Run("POSITIVE/alice sees her assignee template AND her APPROVER template", func(t *testing.T) {
		// 🛑 INHERITED PROPERTY 1, ASSERTED NOT ASSUMED. tplApprover is assigned
		// to alice with assignment_role='approver'. The Go resolver never
		// filters on that column, so an approver sees what an assignee sees.
		// If anyone "tightens" the view in migration 0073 with a
		// WHERE assignment_role='assignee', this line is what turns red.
		rvAssertRows(t, "alice", s.get(t, templates, s.rvToken(t, uAlice)), tplAlice, tplApprover)
	})

	t.Run("POSITIVE/bob sees ONLY his own — not alice's, not the role one", func(t *testing.T) {
		rvAssertRows(t, "bob", s.get(t, templates, s.rvToken(t, uBob)), tplBob)
	})

	t.Run("POSITIVE/dave reaches his template by ROLE, holding no direct assignment", func(t *testing.T) {
		// The assignee_type='role' disjunct: ta.assignee_id = ANY(u.roles).
		// Dave is named in no assignment row at all.
		rvAssertRows(t, "dave (role=manager)", s.get(t, templates, s.rvToken(t, uDave, "manager")), tplByRole)
	})

	t.Run("POSITIVE/carol the admin sees EVERY template including the orphan", func(t *testing.T) {
		// 🛑 INHERITED PROPERTY 2, ASSERTED NOT ASSUMED. The admin arm is
		// unconditional. tplOrphan has NO assignment rows whatsoever and carol
		// is named in none — she sees it purely because she is an admin, and
		// she is the ONLY identity here who can see it.
		rvAssertRows(t, "carol (admin)", s.get(t, templates, s.rvToken(t, uCarol, "admin")), rvAllTemplateIDs...)
	})

	// ── V1-V5 · the JWT layer ────────────────────────────────────────────
	// Refused by PostgREST's verifier, not by this card's policies. Green in
	// the red state too. Included because a suite that only tests its own layer
	// cannot notice the day the layer beneath it stops running.

	t.Run("V1/anon (no token at all)", func(t *testing.T) {
		rvAssertRefused(t, "V1 anon", s.get(t, templates, ""))
	})

	t.Run("V2/wrong signature", func(t *testing.T) {
		tok, err := Sign(Claims{
			Sub: uAlice, Role: SupabaseRole,
			Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
		}, "definitely-not-the-stacks-secret")
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		rvAssertRefused(t, "V2 wrong signature", s.get(t, templates, tok))
	})

	t.Run("V3/expired token", func(t *testing.T) {
		tok := s.mint(t, Claims{
			Sub: uAlice, Role: SupabaseRole,
			Iat: time.Now().Add(-2 * time.Hour).Unix(),
			Exp: time.Now().Add(-1 * time.Hour).Unix(),
		})
		rvAssertRefused(t, "V3 expired token", s.get(t, templates, tok))
	})

	t.Run("V4/tampered payload, original signature", func(t *testing.T) {
		// Distinct from V2: the attacker did not re-sign, they EDITED. Bob's
		// token with the sub rewritten to carol's — the escalation an
		// intercepted token invites.
		tok := s.rvToken(t, uBob)
		parts := strings.Split(tok, ".")
		if len(parts) != 3 {
			t.Fatalf("unexpected token shape: %q", tok)
		}
		forged, err := signHS256(map[string]any{
			"sub": uCarol, "role": SupabaseRole,
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		}, "not-the-secret")
		if err != nil {
			t.Fatalf("build forged payload: %v", err)
		}
		// forged payload + the ORIGINAL, genuinely-valid signature.
		tampered := strings.Split(forged, ".")[0] + "." + strings.Split(forged, ".")[1] + "." + parts[2]
		rvAssertRefused(t, "V4 tampered payload", s.get(t, templates, tampered))
	})

	t.Run("V5/missing sub claim", func(t *testing.T) {
		// Valid signature, valid role, NO subject. hq_jwt_claim('sub') returns
		// NULL, every comparison is NULL, and NULL is not true — so the policy
		// admits nothing. This is the variant that would break loudest if
		// anyone wrote `coalesce(hq_jwt_claim('sub'), '')` "for safety".
		tok, err := signHS256(map[string]any{
			"role": SupabaseRole,
			"iat":  time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		}, s.secret)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		rvAssertRefused(t, "V5 missing sub", s.get(t, templates, tok))
	})

	// ── V6-V9 · the RLS layer, the genuinely red set ─────────────────────

	t.Run("V6/cross-template read — alice asks for bob's template by id", func(t *testing.T) {
		// A filtered request, not a listing. A policy that only scopes the
		// unfiltered list is defeated by exactly this.
		rvAssertRefused(t, "V6 alice -> tplBob",
			s.get(t, "/checklist_templates?select=id&id=eq."+tplBob, s.rvToken(t, uAlice)))
	})

	t.Run("V7/claim injection — alice's token ASSERTS admin, live table says otherwise", func(t *testing.T) {
		// 🛑 The reason the predicate reads hq_user_roles through the FDW
		// instead of reading the token's own hq_roles claim. This token is
		// genuinely signed by the stack's secret and says `hq_roles: [admin]`.
		// If the policy trusted it, alice would see all five templates. She
		// sees her two.
		r := s.get(t, templates, s.rvToken(t, uAlice, "admin", "superadmin"))
		rvAssertRows(t, "alice with a LYING admin claim", r, tplAlice, tplApprover)
		rvAssertRefused(t, "V7 claim injection (orphan must stay hidden)",
			s.get(t, "/checklist_templates?select=id&id=eq."+tplOrphan, s.rvToken(t, uAlice, "admin")))
	})

	t.Run("V8/revocation replay — SAME unexpired token, assignment deleted in HQ", func(t *testing.T) {
		// 🛑 THIS IS THE VARIANT DECISION 92 EXISTS FOR.
		//
		// The operator's requirement, as a user story: a revoked crew member's
		// phone stops showing the checklist ON THE VERY NEXT SYNC, with no
		// window at all. Every asynchronous mechanism — outbox, logical
		// replication, periodic reconcile — leaves a window here in which the
		// token still works. Reading through has none: HQ's transaction commits
		// and the next request already sees the new answer.
		//
		// Note what is NOT re-minted: the token. It is the same string, still
		// unexpired, still validly signed.
		tok := s.rvToken(t, uAlice)
		rvAssertRows(t, "before revocation", s.get(t, templates, tok), tplAlice, tplApprover)

		if _, err := s.hq.Exec(context.Background(),
			`delete from template_assignments where template_id = $1`, tplAlice); err != nil {
			t.Fatalf("revoke: %v", err)
		}

		rvAssertRows(t, "after revocation, SAME token", s.get(t, templates, tok), tplApprover)
		rvAssertRefused(t, "V8 revocation replay",
			s.get(t, "/checklist_templates?select=id&id=eq."+tplAlice, tok))

		// Restore, and prove the restore took — a cleanup that silently failed
		// would make every later variant a test of a different fixture.
		if _, err := s.hq.Exec(context.Background(),
			`insert into template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 values ($1, 'user', $2, 'assignee')`, tplAlice, uAlice); err != nil {
			t.Fatalf("restore: %v", err)
		}
		rvAssertRows(t, "after restore, still the SAME token", s.get(t, templates, tok), tplAlice, tplApprover)
	})

	t.Run("V9/live grant — a NEW assignment lands without re-minting anything", func(t *testing.T) {
		// The positive direction of V8, and the half that catches a policy
		// which caches. Bob's token is minted BEFORE the assignment exists.
		tok := s.rvToken(t, uBob)
		rvAssertRows(t, "bob before", s.get(t, templates, tok), tplBob)

		if _, err := s.hq.Exec(context.Background(),
			`insert into template_assignments (template_id, assignee_type, assignee_id, assignment_role)
			 values ($1, 'user', $2, 'approver')`, tplOrphan, uBob); err != nil {
			t.Fatalf("grant: %v", err)
		}
		rvAssertRows(t, "bob after, SAME token", s.get(t, templates, tok), tplBob, tplOrphan)

		if _, err := s.hq.Exec(context.Background(),
			`delete from template_assignments where template_id = $1 and assignee_id = $2`,
			tplOrphan, uBob); err != nil {
			t.Fatalf("ungrant: %v", err)
		}
		rvAssertRows(t, "bob restored", s.get(t, templates, tok), tplBob)
	})

	// ── V10-V11 · writes, refused by POLICY ABSENCE ──────────────────────
	// 🛑 A DECISION, not a gap. 0003 §2(a): ResolveEntityAccess is a fan-out
	// resolver answering "who RECEIVES this op". Reusing it to decide who may
	// WRITE would invent a permission semantic no shipped code asserts, and the
	// card forbids that. So the four tables keep 0001's grants and have no
	// write policy — deny-all. These variants prove it rather than assume it,
	// and they LEAK in the red state (RLS off + grant insert/update).

	t.Run("V10/forged insert — alice creates a template", func(t *testing.T) {
		r := s.do(t, "POST", "/checklist_templates", s.rvToken(t, uAlice),
			`{"id":"`+tplOrphan+`-forged","name":"forged"}`)
		if r.status >= 200 && r.status < 300 {
			t.Errorf("V10 forged insert LANDED (%s) — writes are supposed to be deny-all", r)
		} else {
			t.Logf("REFUSED  %-52s %s", "V10 forged insert", r)
		}
		// The status code is not the proof. Whether the row exists is.
		var n int
		if err := s.pool.QueryRow(context.Background(),
			`select count(*) from public.checklist_templates where name = 'forged'`).Scan(&n); err != nil {
			t.Fatalf("count forged: %v", err)
		}
		if n != 0 {
			t.Errorf("V10: %d forged row(s) LANDED in the table despite the status code", n)
		}
	})

	t.Run("V11/forged update — alice rewrites bob's template", func(t *testing.T) {
		r := s.do(t, "PATCH", "/checklist_templates?id=eq."+tplBob, s.rvToken(t, uAlice),
			`{"name":"owned by alice now"}`)
		if r.status >= 200 && r.status < 300 && len(r.ids()) > 0 {
			t.Errorf("V11 forged update LANDED (%s)", r)
		} else {
			t.Logf("REFUSED  %-52s %s", "V11 forged update", r)
		}
		var name string
		if err := s.pool.QueryRow(context.Background(),
			`select name from public.checklist_templates where id = $1`, tplBob).Scan(&name); err != nil {
			t.Fatalf("read back: %v", err)
		}
		if name != "Bob Opening" {
			t.Errorf("V11: bob's template was RENAMED to %q — the update landed", name)
		}
	})

	// ── V12-V14 · the other two entity types ─────────────────────────────

	t.Run("V12/submissions are scoped by their template", func(t *testing.T) {
		rvAssertRows(t, "alice's submissions", s.get(t, submissions, s.rvToken(t, uAlice)), subAlice)
		rvAssertRefused(t, "V12 alice -> bob's submission",
			s.get(t, "/checklist_submissions?select=id&id=eq."+subBob, s.rvToken(t, uAlice)))
		rvAssertRows(t, "carol (admin) sees all submissions",
			s.get(t, submissions, s.rvToken(t, uCarol, "admin")), rvAllSubmissionIDs...)
	})

	t.Run("V13/DRAFT responses are scoped by FIELD, not by submission", func(t *testing.T) {
		// 🛑 Every response in this fixture has submission_id IS NULL. A policy
		// that scoped by submission would return an empty set here for
		// EVERYONE, including carol — and would look exactly like a working
		// deny to anyone reading only the attacker half. The carol line is what
		// separates the two readings.
		rvAssertRows(t, "alice's drafts", s.get(t, responses, s.rvToken(t, uAlice)),
			"resp-alice", "resp-approver")
		rvAssertRows(t, "dave's draft (reached by role)",
			s.get(t, responses, s.rvToken(t, uDave, "manager")), "resp-byrole")
		rvAssertRefused(t, "V13 alice -> bob's draft",
			s.get(t, "/submission_responses?select=id&id=eq.resp-bob", s.rvToken(t, uAlice)))
	})

	t.Run("V14/an UNRESOLVABLE field is invisible to EVERYONE, admins included", func(t *testing.T) {
		// 🛑 The nesting-order decision in 0003's hq_can_see_field, asserted.
		// Written the obvious way — hq_can_see_template(<lookup>) — a NULL
		// lookup still satisfies the admin arm and carol would see resp-ghost.
		// The Go returns []string{} on ErrNoRows: nobody. Carol is the only
		// identity that can distinguish the two implementations, which is why
		// she is the one asked.
		rvAssertRows(t, "carol sees every RESOLVABLE response",
			s.get(t, responses, s.rvToken(t, uCarol, "admin")),
			"resp-alice", "resp-approver", "resp-bob", "resp-byrole")
		rvAssertRefused(t, "V14 orphan response, asked by an ADMIN",
			s.get(t, "/submission_responses?select=id&id=eq.resp-ghost", s.rvToken(t, uCarol, "admin")))
	})

	// ── V15-V17 · 🛑 the tables THIS CARD put on this server ─────────────
	// Refused by table grants (0002 §4), not by a policy — and green in the red
	// state. Included because this card is what made HQ's role map and
	// assignment map reachable from the substrate at all, and it owes a proof
	// that PostgREST will not simply hand them over. A readable hq_user_roles
	// is the whole company's role map in one GET; that would leak MORE than
	// everything below protects.

	for _, ft := range []struct{ name, path string }{
		{"V15 hq_user_roles", "/hq_user_roles?select=user_id"},
		{"V16 hq_template_assignees", "/hq_template_assignees?select=user_id"},
		{"V17 hq_field_templates", "/hq_field_templates?select=field_id"},
	} {
		t.Run("V15-17/"+ft.name+" is not readable over PostgREST", func(t *testing.T) {
			for _, who := range []struct {
				label string
				tok   string
			}{
				{"anon", ""},
				{"alice", s.rvToken(t, uAlice)},
				{"carol the admin", s.rvToken(t, uCarol, "admin")},
			} {
				r := s.get(t, ft.path, who.tok)
				if r.status >= 200 && r.status < 300 && strings.Contains(r.body, "11111111-") {
					t.Errorf("%s LEAKED HQ's permission data to %s: %s", ft.name, who.label, r)
					continue
				}
				t.Logf("REFUSED  %-52s %s", ft.name+" / "+who.label, r)
			}
		})
	}

	// ── V18 · submission_rejections stays deny-all ───────────────────────

	t.Run("V18/submission_rejections is deny-all, for admins too", func(t *testing.T) {
		// 🛑 A DECISION WITH EVIDENCE, not a gap. 0003 §2(b):
		// ResolveEntityAccess has NO CASE for this entity type and falls
		// through to `return []string{}` — the WebSocket layer does not fan
		// rejections out today. A policy here would be an EXTENSION, not a
		// port, and the card forbids inventing permission semantics.
		//
		// Carol is asked because she is the one identity for whom "deny-all"
		// and "the policy is just broken" would otherwise look the same: she
		// sees every template, every submission and every resolvable response,
		// and still sees no rejection.
		rvAssertRefused(t, "V18 rejections / carol the admin",
			s.get(t, rejections, s.rvToken(t, uCarol, "admin")))
		rvAssertRefused(t, "V18 rejections / alice",
			s.get(t, rejections, s.rvToken(t, uAlice)))
		// And the control that keeps it from being vacuous: the row is there.
		rvAssertRows(t, "service_role sees the rejection that nobody else does",
			s.get(t, rejections, s.serviceRoleControl(t)), "rej-alice")
	})

	// ── V19 · the auth.uid() trap, re-proved against THIS card's policies ─

	t.Run("V19/auth.uid() returns NOTHING while the plural GUC discriminates", func(t *testing.T) {
		// 🛑 Banked by card `sync-jwt-bridge-endpoint` on 2026-07-26 and NOT
		// rediscovered here — public.hq_uid_trap is applied, not reimplemented.
		// What is new is the comparison: the discriminating side is now THIS
		// CARD'S policy. If anyone ever "fixes" hq_can_see_template to use
		// auth.uid() because the hosted docs say so, the first line goes empty
		// while the third stays populated, and this test says which.
		//
		// auth.uid() reads the LEGACY SINGULAR GUC request.jwt.claim.sub.
		// PGRST_DB_USE_LEGACY_GUCS=false, so it is never populated, auth.uid()
		// is NULL, and the policy silently admits nothing. The failure is a
		// 200 with an empty body — it reads as "this user has no data".
		alice := s.rvToken(t, uAlice)
		rvAssertRows(t, "plural GUC (this card) DISCRIMINATES",
			s.get(t, templates, alice), tplAlice, tplApprover)

		trap := s.get(t, "/hq_uid_trap?select=id&order=id", alice)
		if len(trap.ids()) != 0 {
			t.Errorf("🛑 auth.uid() ADMITTED %v — the stack changed. "+
				"PGRST_DB_USE_LEGACY_GUCS or the auth schema is no longer what "+
				"hq-bridge-policies.sql documents, and that file needs re-reading.",
				trap.ids())
		} else {
			t.Logf("REFUSED  %-52s %s", "V19 auth.uid() policy returns NOTHING", trap)
		}

		// The control that makes the empty result mean something.
		ctl := s.get(t, "/hq_uid_trap?select=id&order=id", s.serviceRoleControl(t))
		if len(ctl.ids()) < 2 {
			t.Errorf("hq_uid_trap is EMPTY (%s) — V19's refusal proves nothing. "+
				"The trap must contain rows or it is not a trap.", ctl)
		} else {
			t.Logf("CONTROL  %-52s %v", "V19 trap is NOT empty", ctl.ids())
		}
	})

	// ── CONTROL Z ────────────────────────────────────────────────────────
	// 🛑 Re-taken AFTER every variant. V8, V9, V10 and V11 all mutate — two of
	// them mutate HQ. If any of them destroyed rows or left HQ's fixture
	// altered, every refusal recorded after it was a refusal against a
	// different, smaller fixture. This is the line that says the fixture the
	// suite finished with is the fixture it started with.
	t.Run("CONTROL/Z service_role control, re-taken after all variants", func(t *testing.T) {
		god := s.serviceRoleControl(t)
		rvAssertRows(t, "templates still all there", s.get(t, templates, god), rvAllTemplateIDs...)
		rvAssertRows(t, "submissions still all there", s.get(t, submissions, god), rvAllSubmissionIDs...)
		rvAssertRows(t, "responses still all there", s.get(t, responses, god),
			"resp-alice", "resp-approver", "resp-bob", "resp-byrole", "resp-ghost")
		rvAssertHQPopulated(t, s.hq)
		rvAssertFDWPopulated(t, s.pool)
	})
}
