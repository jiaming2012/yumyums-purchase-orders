package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ═══════════════════════════════════════════════════════════════════════════
// Attack-variant suite — the JWT bridge, proved DISCRIMINATING
// ═══════════════════════════════════════════════════════════════════════════
//
// WHY THIS FILE EXISTS (card B `sync-jwt-bridge-endpoint`, run 2026-07-26)
//
// Modelled on tests/grant-enforcement-parity.spec.js. That file's structure is
// the ask, and its central lesson is the one this file is built around:
//
//     A 200 PROVES NOTHING.
//
// An endpoint that answers is not an endpoint that authorizes. Every scoped
// result below is therefore paired with two things, and is worthless without
// them:
//
//   1. A POSITIVE half — the authorized identity SEES the rows. Without it,
//      "the attacker saw nothing" is equally consistent with "nobody can see
//      anything", i.e. a policy that is merely broken.
//
//   2. 🛑 A `service_role` BYPASSRLS CONTROL — the same table, the same
//      endpoint, read with a god-token, returning ALL rows across both owners
//      and both apps. This is what rules out the boring explanation for every
//      empty result in this file: that the table was empty, or the URL wrong,
//      or PostgREST's schema cache stale. The rows were always there; RLS was
//      hiding them. Delete the control and the whole suite goes vacuous.
//
// ── RED-FIRST, which is this card's real gate and not a formality ──────────
//
// Every variant below was captured REFUSING before the policy that refuses it
// existed. The two SQL files are split precisely so that ordering is
// reproducible rather than asserted:
//
//	sql/hq-bridge-fixture.sql   tables, grants, projection, seed. NO RLS.
//	sql/hq-bridge-policies.sql  RLS enabled + the policies.
//
// Reproduce the red at any time:
//
//	SPIKE_SKIP_POLICIES=1 go test ./internal/sync/ -run TestJWTBridgeRLS -v
//
// The suite then applies only the fixture, actively tears RLS back down, and
// the RLS-gated variants fail — which is what they are supposed to do against
// a database with no policies in it.
//
// ── Which layer refuses what — stated, not blurred ────────────────────────
//
// Honesty about the red matters more than a uniform-looking table. The seven
// required variants do NOT all live at the same layer:
//
//	LAYER    variants                                     refused by
//	─────    ────────────────────────────────────────     ──────────────────
//	JWT      wrong signature · expired · invalid role     PostgREST's verifier
//	                                                       (pre-existing; these
//	                                                       are green in the red
//	                                                       state too, and this
//	                                                       comment is the record
//	                                                       of that, not a claim
//	                                                       that the card wrote
//	                                                       them)
//	GRANT    anon                                         table grants
//	RLS      missing sub · cross-owner · stale claim ·    the policies THIS
//	         replay after revocation · forged owner       card writes — the
//	         write · forged grant write · lockout ·       genuinely red set
//	         auth.uid() trap
//
// ── Stack precondition ────────────────────────────────────────────────────
//
// W1's self-hosted stack, LOCAL AND THROWAWAY:
//   docker compose -p spike-supabase -f docker-compose.supabase.yml up -d
// Never a hosted Supabase project. Never production. No real HQ data.
//
// 🛑 NO stack at all SKIPS; a RESOLVED-BUT-DEAD stack FAILS. That asymmetry
// lives in spikestack_gate_test.go and is not optional — the first version of
// this file carried HARD-CODED DEFAULT PORTS and skipped when they did not
// answer, and because docker-compose.supabase.yml publishes EPHEMERAL ports the
// defaults were wrong on essentially every run. This suite therefore reported
// `ok` while skipping in its entirety (finding F1, run overnight-20260801). The
// endpoints are now resolved with `docker compose port`; there is no port
// constant left here to go stale.

const (
	defaultSpikeSecret = "2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c"

	userAlice = "hq-user-alice"
	userBob   = "hq-user-bob"

	appOps = "operations"
	appInv = "inventory"
)

// Every seeded row id, so a refusal can be checked for LEAKAGE rather than
// merely for a status code. A variant that returns 200 with an empty body is
// refused; a variant that returns 200 carrying one of these ids is not.
var allSeededIDs = []string{
	"chk-alice-ops-1", "chk-alice-ops-2", "chk-bob-ops-1", "chk-alice-inv-1",
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

type spikeStack struct {
	rest   string
	secret string
	pool   *pgxpool.Pool
}

// sqlDir resolves .night-crew/qa/spike-supabase/sql/ from this package's
// location (backend/internal/sync), so the suite does not depend on cwd.
func sqlDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "..", ".night-crew", "qa", "spike-supabase", "sql")
}

// connectSpike resolves the stack and either SKIPS (no stack configured) or
// connects — and FAILS if a configured stack does not answer. See the gate in
// spikestack_gate_test.go. It then (re)applies the fixture, and applies the
// policies unless SPIKE_SKIP_POLICIES=1 — in which case it actively TEARS RLS
// BACK DOWN, so the red state is reproducible on a database where the policies
// have already been applied once.
func connectSpike(t *testing.T) *spikeStack {
	t.Helper()

	cfg, ok := resolveSpikeConfig(t)
	if !ok {
		// 🛑 Reached ONLY under HQ_SYNC_SUBSTRATE_OPTIONAL=1 since B-36.
		t.Skipf("%s=1 — skipping, and SKIPPED IS NOT PASSED: with this off there is no "+
			"JWT-bridge RLS evidence in the tree. Bring it up with: "+
			"docker compose -p %s -f docker-compose.supabase.yml up -d",
			spikeOptionalEnv, spikeComposeProject)
	}

	ctx := context.Background()
	pool := dialSpikeDB(t, ctx, cfg)
	t.Cleanup(pool.Close)
	requireSpikeREST(t, cfg)

	s := &spikeStack{rest: cfg.restURL, secret: cfg.secret, pool: pool}

	s.applySQL(t, filepath.Join(sqlDir(t), "hq-bridge-fixture.sql"))

	policies := filepath.Join(sqlDir(t), "hq-bridge-policies.sql")
	if os.Getenv("SPIKE_SKIP_POLICIES") == "1" {
		t.Logf("SPIKE_SKIP_POLICIES=1 — RED MODE: tearing RLS down and NOT applying %s", policies)
		s.tearDownRLS(t)
	} else if _, err := os.Stat(policies); err == nil {
		s.applySQL(t, policies)
	} else {
		t.Logf("RED MODE (implicit): %s does not exist yet", policies)
	}

	// PostgREST caches the schema; DDL above needs a reload before it is
	// visible. NOTIFY is cheaper than a restart, and the sleep is the settle.
	if _, err := s.pool.Exec(ctx, `notify pgrst, 'reload schema'`); err != nil {
		t.Fatalf("notify pgrst: %v", err)
	}
	time.Sleep(400 * time.Millisecond)
	return s
}

func (s *spikeStack) applySQL(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	// pgx v5's Exec uses the simple query protocol when there are no
	// arguments, which is what allows a multi-statement script here.
	if _, err := s.pool.Exec(context.Background(), string(b)); err != nil {
		t.Fatalf("apply %s: %v", filepath.Base(path), err)
	}
}

// tearDownRLS restores the RED state on a database the policies already ran
// against. Without this, "reproduce the red" would only work on a virgin stack.
func (s *spikeStack) tearDownRLS(t *testing.T) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		drop policy if exists hq_sync_checklists_select on public.hq_sync_checklists;
		drop policy if exists hq_sync_checklists_insert on public.hq_sync_checklists;
		drop policy if exists hq_sync_checklists_update on public.hq_sync_checklists;
		drop policy if exists hq_uid_trap_select        on public.hq_uid_trap;
		drop policy if exists hq_grant_projection_select on public.hq_grant_projection;
		alter table public.hq_sync_checklists  disable row level security;
		alter table public.hq_uid_trap         disable row level security;
		alter table public.hq_grant_projection disable row level security;
	`)
	if err != nil {
		t.Fatalf("tearDownRLS: %v", err)
	}
}

// ── Token construction ─────────────────────────────────────────────────────

// aliceToken is what the REAL bridge would mint for Alice: sub, authenticated,
// her live grants. Everything the attacker variants do is a mutation of this.
func (s *spikeStack) aliceToken(t *testing.T) string {
	t.Helper()
	return s.mint(t, Claims{
		Sub: userAlice, Role: SupabaseRole,
		Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
		Email: "alice@yumyums.kitchen", HQRoles: []string{"team_member"},
		HQGrants: []string{appOps},
	})
}

func (s *spikeStack) mint(t *testing.T, c Claims) string {
	t.Helper()
	tok, err := Sign(c, s.secret)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	return tok
}

// serviceRoleControl mints the BYPASSRLS god-token.
//
// 🛑 It goes through signHS256 directly, DELIBERATELY BYPASSING Sign(), because
// Sign refuses to emit `service_role` — see TestMint_NeverMintsServiceRole.
// That refusal is the product invariant; this is the one place in the codebase
// allowed around it, and only because a control that cannot see everything
// cannot prove that everything was there.
func (s *spikeStack) serviceRoleControl(t *testing.T) string {
	t.Helper()
	tok, err := signHS256(map[string]any{
		"sub":  "hq-sync-control",
		"role": ServiceRole,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Hour).Unix(),
	}, s.secret)
	if err != nil {
		t.Fatalf("service_role control mint: %v", err)
	}
	return tok
}

// ── PostgREST plumbing ─────────────────────────────────────────────────────

type restResp struct {
	status int
	body   string
}

func (r restResp) String() string {
	return fmt.Sprintf("HTTP %d %s", r.status, strings.TrimSpace(r.body))
}

// ids extracts the `id` field of every object in a PostgREST array response.
// A non-array (an error envelope) yields none, which is the correct reading:
// an error returned zero rows.
func (r restResp) ids() []string {
	var rows []map[string]any
	if err := json.Unmarshal([]byte(r.body), &rows); err != nil {
		return nil
	}
	out := []string{}
	for _, row := range rows {
		if id, ok := row["id"].(string); ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// preferRepresentation and preferMinimal are the two Prefer headers a PostgREST
// write can carry, and on the WRITE half of this package's suites the choice is
// not cosmetic — it decides WHICH POLICY ACTUALLY REFUSES THE ATTACK.
//
// 🛑 FINDING F1, run 20260802 G6 (A2's fix round). `do` used to set
// `return=representation` UNCONDITIONALLY, so every write in both suites went
// through PostgREST with RETURNING. Postgres then applies the SELECT policy's
// USING clause TO THE NEW ROW (CREATE POLICY, "Policies Applied by Command
// Type": for INSERT … RETURNING and for UPDATE, the SELECT/ALL USING expression
// covers the new row). 0003's SELECT predicates are identical to — or, on
// `submission_rejections`, BROADER than — 0004's WITH CHECK predicates, so
// THE READ POLICY WAS SILENTLY ENFORCING MOST OF THE WRITE HALF.
//
// Measured: un-nest the EXISTS in `hq_can_approve_field` (the exact mutation
// that function's own 🛑 comment warns about) and the suite stayed fully green
// at 52/52, while the same attack sent with `Prefer: return=minimal` returned
// HTTP 201 and landed the row. W8's rejection half was reading a 403 issued by
// `hq_can_see_field` — 0003's correctly-nested READ predicate — and reporting it
// as evidence about 0004.
//
// A push-only replication client sending `return=minimal` is entirely ordinary,
// so this is not a contrived probe: it is the shape `sync-hard-cutover` opens
// the door to. Every write REFUSAL in this package is therefore issued under
// BOTH headers — see rvPushRefused.
const (
	preferRepresentation = "return=representation"
	preferMinimal        = "return=minimal"
)

// do issues a request under `Prefer: return=representation` — the historical
// behaviour, kept as the default so every read-half variant is byte-unchanged.
// Write refusals must go through doPrefer/rvPushRefused instead.
func (s *spikeStack) do(t *testing.T, method, path, token, body string) restResp {
	t.Helper()
	return s.doPrefer(t, method, path, token, body, preferRepresentation)
}

// doPrefer is `do` with the Prefer header named at the call site. `prefer` is
// ignored when there is no body, because PostgREST only honours it on writes.
func (s *spikeStack) doPrefer(t *testing.T, method, path, token, body, prefer string) restResp {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, s.rest+path, rdr)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Prefer", prefer)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return restResp{status: resp.StatusCode, body: string(b)}
}

func (s *spikeStack) get(t *testing.T, path, token string) restResp {
	t.Helper()
	return s.do(t, "GET", path, token, "")
}

// ── Assertions ─────────────────────────────────────────────────────────────

// assertRefused is the shape every attack variant is judged by. It is
// deliberately NOT "status == 401": PostgREST refuses at three different layers
// with three different codes, and an RLS refusal is a 200 with an EMPTY array.
// What actually matters — the only thing that matters — is that NO SEEDED ROW
// CROSSED THE BOUNDARY.
func assertRefused(t *testing.T, variant string, r restResp) {
	t.Helper()
	leaked := []string{}
	for _, id := range r.ids() {
		for _, seeded := range allSeededIDs {
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
	t.Logf("REFUSED  %-46s %s", variant, r)
}

func assertRows(t *testing.T, label string, r restResp, want ...string) {
	t.Helper()
	if r.status != http.StatusOK {
		t.Errorf("%s: expected HTTP 200, got %s", label, r)
		return
	}
	got := r.ids()
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("%s: expected rows %v, got %v\n  response: %s", label, want, got, r)
		return
	}
	t.Logf("ADMITTED %-46s %v", label, got)
}

// ═══════════════════════════════════════════════════════════════════════════
// The suite
// ═══════════════════════════════════════════════════════════════════════════

func TestJWTBridgeRLS(t *testing.T) {
	s := connectSpike(t)

	const table = "/hq_sync_checklists?select=id,owner_id,app_slug&order=id"

	// ── CONTROL 0, run FIRST ─────────────────────────────────────────────
	// 🛑 Everything below is meaningless without this. `service_role` has
	// BYPASSRLS: handed that claim, PostgREST SET ROLEs into it and the same
	// table at the same URL returns every row across both owners and both
	// apps. That rules out the boring explanations for every empty result in
	// this file — empty table, wrong URL, stale schema cache.
	t.Run("CONTROL/service_role BYPASSRLS proves the rows are there", func(t *testing.T) {
		r := s.get(t, table, s.serviceRoleControl(t))
		assertRows(t, "service_role sees ALL rows", r, allSeededIDs...)
	})

	// ── POSITIVE baseline ────────────────────────────────────────────────
	// The authorized identity SEES rows. Without this half, every refusal
	// below is equally consistent with a policy that simply denies everyone.
	t.Run("POSITIVE/alice reads exactly her own rows in an app she holds", func(t *testing.T) {
		r := s.get(t, table, s.aliceToken(t))
		assertRows(t, "alice reads her operations rows", r,
			"chk-alice-ops-1", "chk-alice-ops-2")
	})

	// ── V1 · anon ────────────────────────────────────────────────────────
	t.Run("V1/anon (no token at all)", func(t *testing.T) {
		assertRefused(t, "V1 anon", s.get(t, table, ""))
	})

	// ── V2 · wrong signature ─────────────────────────────────────────────
	t.Run("V2/wrong signature", func(t *testing.T) {
		tok, err := Sign(Claims{
			Sub: userAlice, Role: SupabaseRole,
			Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
			HQGrants: []string{appOps},
		}, "definitely-not-the-stacks-secret")
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		assertRefused(t, "V2 wrong signature", s.get(t, table, tok))
	})

	// ── V3 · expired token ───────────────────────────────────────────────
	t.Run("V3/expired token", func(t *testing.T) {
		tok := s.mint(t, Claims{
			Sub: userAlice, Role: SupabaseRole,
			Iat:      time.Now().Add(-2 * time.Hour).Unix(),
			Exp:      time.Now().Add(-1 * time.Hour).Unix(),
			HQGrants: []string{appOps},
		})
		assertRefused(t, "V3 expired token", s.get(t, table, tok))
	})

	// ── V4 · tampered payload, original signature ────────────────────────
	// Distinct from V2: the attacker did not re-sign, they EDITED. This is
	// what an intercepted token invites, and the failure mode if a verifier
	// ever decoded claims before checking the MAC.
	t.Run("V4/tampered payload keeping the original signature", func(t *testing.T) {
		valid := s.aliceToken(t)
		parts := strings.Split(valid, ".")
		if len(parts) != 3 {
			t.Fatalf("expected 3 JWT segments, got %d", len(parts))
		}
		forged, err := json.Marshal(Claims{
			Sub: userBob, Role: SupabaseRole,
			Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
			HQGrants: []string{appOps, appInv},
		})
		if err != nil {
			t.Fatalf("marshal forged payload: %v", err)
		}
		tampered := parts[0] + "." + b64(forged) + "." + parts[2]
		assertRefused(t, "V4 tampered payload", s.get(t, table, tampered))
	})

	// ── V5 · missing sub ─────────────────────────────────────────────────
	// Correctly signed, correct role, NO identity. The policy's
	// `owner_id = claims ->> 'sub'` becomes `owner_id = NULL`, which is NULL,
	// which is not true — so it selects nothing. This is also the variant
	// that would EXPLODE under auth.uid() (::uuid on a null/non-uuid), which
	// is why V13 exists.
	t.Run("V5/missing sub claim", func(t *testing.T) {
		tok, err := signHS256(map[string]any{
			"role":      SupabaseRole,
			"iat":       time.Now().Unix(),
			"exp":       time.Now().Add(time.Hour).Unix(),
			"hq_grants": []string{appOps},
		}, s.secret)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		assertRefused(t, "V5 missing sub", s.get(t, table, tok))
	})

	// ── V6 · invalid role claim ──────────────────────────────────────────
	// `role` names the Postgres role PostgREST SET ROLEs into. A claim naming
	// a role that does not exist must fail, not fall back to something
	// permissive.
	t.Run("V6/invalid role claim (nonexistent postgres role)", func(t *testing.T) {
		tok, err := signHS256(map[string]any{
			"sub": userAlice, "role": "hq_superuser",
			"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		}, s.secret)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		assertRefused(t, "V6 invalid role claim", s.get(t, table, tok))
	})

	// ── V7 · cross-owner read ────────────────────────────────────────────
	// The identity axis. Alice explicitly asks for Bob's row by id.
	t.Run("V7/cross-owner read (alice asks for bob's row by id)", func(t *testing.T) {
		r := s.get(t, "/hq_sync_checklists?select=id,owner_id&id=eq.chk-bob-ops-1", s.aliceToken(t))
		assertRefused(t, "V7 cross-owner read", r)
	})

	// ── V8 · stale grant claim is not load-bearing ───────────────────────
	// 🛑 The entitlement axis, and the proof that the token's `hq_grants`
	// claim is ADVISORY. Alice OWNS chk-alice-inv-1, and here she carries a
	// token whose claim asserts the `inventory` grant. She still cannot read
	// it, because `hq_grant_projection` — the live table — says she does not
	// hold it. If the policy read the claim, this variant would pass.
	t.Run("V8/stale grant claim in the token opens nothing", func(t *testing.T) {
		tok := s.mint(t, Claims{
			Sub: userAlice, Role: SupabaseRole,
			Iat: time.Now().Unix(), Exp: time.Now().Add(time.Hour).Unix(),
			HQRoles:  []string{"team_member"},
			HQGrants: []string{appOps, appInv}, // ← the lie
		})
		r := s.get(t, "/hq_sync_checklists?select=id,owner_id,app_slug&app_slug=eq."+appInv, tok)
		assertRefused(t, "V8 stale grant claim", r)
	})

	// ── V9 · token replay after grant revocation ─────────────────────────
	// 🛑 The variant this whole design exists for. The SAME unexpired token,
	// which worked seconds ago, stops working the instant the grant is
	// revoked — with a service_role control taken WHILE it is refused, so
	// "the rows vanished" is excluded as the explanation.
	t.Run("V9/token replay after grant revocation", func(t *testing.T) {
		ctx := context.Background()
		tok := s.aliceToken(t)

		before := s.get(t, table, tok)
		assertRows(t, "V9 before revocation, the token works", before,
			"chk-alice-ops-1", "chk-alice-ops-2")

		if _, err := s.pool.Exec(ctx,
			`delete from public.hq_grant_projection where user_id=$1 and app_slug=$2`,
			userAlice, appOps); err != nil {
			t.Fatalf("revoke grant: %v", err)
		}
		t.Cleanup(func() {
			_, _ = s.pool.Exec(context.Background(),
				`insert into public.hq_grant_projection (user_id, app_slug) values ($1,$2)
				 on conflict do nothing`, userAlice, appOps)
		})

		assertRefused(t, "V9 replay after grant revocation", s.get(t, table, tok))

		// The control, taken while the replay is being refused.
		assertRows(t, "V9 control: rows still present during the refusal",
			s.get(t, table, s.serviceRoleControl(t)), allSeededIDs...)
	})

	// ── V10 · forged owner write ─────────────────────────────────────────
	// Reads and writes are separate policies; a WITH CHECK that is missing or
	// wrong is invisible to every read test in this file.
	t.Run("V10/forged owner write (alice inserts a row owned by bob)", func(t *testing.T) {
		body := `{"id":"chk-forged-owner","owner_id":"` + userBob + `","app_slug":"` + appOps + `","body":"forged"}`
		r := s.do(t, "POST", "/hq_sync_checklists", s.aliceToken(t), body)
		t.Cleanup(func() {
			_, _ = s.pool.Exec(context.Background(),
				`delete from public.hq_sync_checklists where id='chk-forged-owner'`)
		})
		if r.status >= 200 && r.status < 300 {
			t.Errorf("V10 forged owner write WAS NOT REFUSED: %s", r)
		} else {
			t.Logf("REFUSED  %-46s %s", "V10 forged owner write", r)
		}
		// The write must not have landed even if the status lied.
		var n int
		if err := s.pool.QueryRow(context.Background(),
			`select count(*) from public.hq_sync_checklists where id='chk-forged-owner'`).Scan(&n); err != nil {
			t.Fatalf("verify forged row: %v", err)
		}
		if n != 0 {
			t.Errorf("V10: the forged row LANDED in the table (%d row(s)) — the status code was not the truth", n)
		}
	})

	// ── V11 · forged grant write ─────────────────────────────────────────
	// Alice writing into an app she does not hold. Own identity, wrong
	// entitlement — the write-side mirror of V8.
	t.Run("V11/forged grant write (own identity, ungranted app)", func(t *testing.T) {
		body := `{"id":"chk-forged-grant","owner_id":"` + userAlice + `","app_slug":"` + appInv + `","body":"forged"}`
		r := s.do(t, "POST", "/hq_sync_checklists", s.aliceToken(t), body)
		t.Cleanup(func() {
			_, _ = s.pool.Exec(context.Background(),
				`delete from public.hq_sync_checklists where id='chk-forged-grant'`)
		})
		if r.status >= 200 && r.status < 300 {
			t.Errorf("V11 forged grant write WAS NOT REFUSED: %s", r)
		} else {
			t.Logf("REFUSED  %-46s %s", "V11 forged grant write", r)
		}
		var n int
		if err := s.pool.QueryRow(context.Background(),
			`select count(*) from public.hq_sync_checklists where id='chk-forged-grant'`).Scan(&n); err != nil {
			t.Fatalf("verify forged row: %v", err)
		}
		if n != 0 {
			t.Errorf("V11: the forged row LANDED in the table (%d row(s))", n)
		}
	})

	// ── V12 · deactivated-user lockout ───────────────────────────────────
	// HQ already deletes every session when a user is deactivated
	// (auth.DeleteAllSessionsByUserID). The bridge's equivalent is removing
	// every projection row: an unexpired token in the wild then reaches
	// nothing, in any app, immediately.
	t.Run("V12/deactivated user — every projection row removed", func(t *testing.T) {
		ctx := context.Background()
		tok := s.aliceToken(t)

		if _, err := s.pool.Exec(ctx,
			`delete from public.hq_grant_projection where user_id=$1`, userAlice); err != nil {
			t.Fatalf("lockout: %v", err)
		}
		t.Cleanup(func() {
			_, _ = s.pool.Exec(context.Background(),
				`insert into public.hq_grant_projection (user_id, app_slug) values ($1,$2)
				 on conflict do nothing`, userAlice, appOps)
		})

		assertRefused(t, "V12 deactivated user lockout", s.get(t, table, tok))
		assertRows(t, "V12 control: rows still present during the lockout",
			s.get(t, table, s.serviceRoleControl(t)), allSeededIDs...)
	})

	// ── V13 · the auth.uid() trap ────────────────────────────────────────
	// 🛑 Finding #1, made reproducible instead of merely believed.
	//
	// public.hq_uid_trap carries the same shape of rows but is governed by an
	// auth.uid() policy — the thing every copy-pasted hosted-Supabase policy
	// uses. Under this stack (no GoTrue, PGRST_DB_USE_LEGACY_GUCS=false),
	// auth.uid() reads the legacy SINGULAR GUC and casts to uuid. With the
	// exact same token that discriminates correctly on hq_sync_checklists, it
	// returns nothing — or raises. Either way it does NOT authorize, and it
	// fails in a way that looks like "the user has no data".
	t.Run("V13/auth.uid() policy fails on this stack (negative control)", func(t *testing.T) {
		tok := s.aliceToken(t)

		trap := s.get(t, "/hq_uid_trap?select=id,owner_id&order=id", tok)
		if trap.status == http.StatusOK && len(trap.ids()) > 0 {
			t.Errorf("V13: the auth.uid() policy ADMITTED %v — this stack was expected to make "+
				"auth.uid() unusable (legacy singular GUC + ::uuid cast). If this now passes, "+
				"the stack's PGRST_DB_USE_LEGACY_GUCS or the auth schema changed, and the "+
				"finding in the runbook needs re-verifying rather than trusting.", trap.ids())
		} else {
			t.Logf("auth.uid() policy yields NOTHING for a valid token: %s", trap)
		}

		// Side by side, same token, same instant: the plural-GUC policy works.
		assertRows(t, "V13 side-by-side: the plural-GUC policy DOES discriminate",
			s.get(t, table, tok), "chk-alice-ops-1", "chk-alice-ops-2")

		// And the control proves hq_uid_trap was not simply empty.
		ctrl := s.get(t, "/hq_uid_trap?select=id,owner_id&order=id", s.serviceRoleControl(t))
		if ctrl.status != http.StatusOK || len(ctrl.ids()) != 2 {
			t.Errorf("V13 control: service_role should see BOTH trap rows, got %s", ctrl)
		} else {
			t.Logf("V13 control: service_role sees %v — the trap table was NOT empty", ctrl.ids())
		}
	})

	// ── CONTROL 1, run LAST ──────────────────────────────────────────────
	// The suite mutates the projection table (V9, V12). Re-taking the control
	// at the end proves the cleanups restored the world and that no earlier
	// variant destroyed the evidence the later ones relied on.
	t.Run("CONTROL/service_role control re-taken after all variants", func(t *testing.T) {
		assertRows(t, "service_role still sees ALL rows", s.get(t, table, s.serviceRoleControl(t)), allSeededIDs...)
	})
}
