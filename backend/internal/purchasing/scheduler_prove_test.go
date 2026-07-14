package purchasing

import (
	"testing"
	"time"

	"github.com/yumyums/hq/internal/auth"
)

// ─────────────────────────────────────────────────────────────────────────────
// Night-crew prove-sweep — purchasing-prove-state-auth-scheduler
//
// This file adds NEW unit tests (in-footprint: adding *_test.go is allowed).
// It does NOT modify any production .go file.
//
// PARK RECORD — FR-19 / FR-20 / FR-21 / FR-22 cron DECISION logic is NOT
// unit-testable against the current code without a production change:
//
//   scheduler.go:80,192,266 and repurchase.go:151 each read the wall clock
//   INLINE via `now := time.Now().In(loc)`. The four run*Check functions take
//   only (ctx, *pgxpool.Pool) — there is no injectable clock, no `now`
//   parameter, and the has-cutoff-passed / reminder-window / low-stock-week /
//   reset-passed decision arithmetic is embedded inside those DB-coupled
//   functions. To assert "at cutoff+1 the auto-lock fires" (AC-7) a test would
//   have to freeze/inject time — which requires editing production code
//   (add a clock param or a package-level nowFn seam). Per the card + runbook
//   PARK rule ("time isn't injectable, hardcoded time.Now() → PARK; do NOT
//   modify production code to make it testable"), the cron decision logic is
//   PARKED. Graduate to a fix WO: add a `now time.Time` (or `nowFn func()
//   time.Time`) seam to the four run*Check funcs, then a Go unit test can seed
//   config + a past cutoff and assert the transition without waiting 15m.
//
// What IS unit-testable WITHOUT touching production is proven below:
//   - parseCutoffTime: the shared HH:MM(:SS) parse the auto-lock (FR-19),
//     reminder (FR-20) and repurchase-reset (FR-22) crons all depend on.
//   - isAdmin: the pure admin-gate predicate every NFR-2 admin handler calls.
//   - the cutoff-candidate weekday/"has-passed" arithmetic, re-derived here
//     as a pure function mirroring scheduler.go:196-204 and fed a FROZEN now,
//     so the auto-lock DECISION (given now) is asserted deterministically.
//     (This proves the decision rule; wiring it to the live clock+DB is the
//     parked fix seam.)
// ─────────────────────────────────────────────────────────────────────────────

// TestParseCutoffTime_Prove asserts the shared cutoff-time parser the crons rely
// on. FR-19/FR-20/FR-22 all call parseCutoffTime before computing the cutoff
// occurrence; a wrong parse silently disables every cron. (scheduler.go:362-375)
func TestParseCutoffTime_Prove(t *testing.T) {
	cases := []struct {
		in           string
		wantH, wantM int
		wantErr      bool
	}{
		{"14:30", 14, 30, false},    // plain HH:MM
		{"06:00", 6, 0, false},      // leading-zero hour
		{"14:30:00", 14, 30, false}, // Postgres TIME cast HH:MM:SS — seconds ignored
		{"09:05", 9, 5, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"nope", 0, 0, true}, // no colon → error
		{"7", 0, 0, true},    // single field → error
	}
	for _, c := range cases {
		h, m, err := parseCutoffTime(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseCutoffTime(%q): expected error, got h=%d m=%d nil err", c.in, h, m)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCutoffTime(%q): unexpected error %v", c.in, err)
			continue
		}
		if h != c.wantH || m != c.wantM {
			t.Errorf("parseCutoffTime(%q) = (%d,%d), want (%d,%d)", c.in, h, m, c.wantH, c.wantM)
		}
	}
}

// TestIsAdmin_Prove asserts the pure admin-gate predicate. NFR-2's whole
// authorization tier (cutoff PUT, simulate-cutoff, lock, unlock, approve, all
// repurchase-reset handlers) short-circuits on !isAdmin → 403. This proves the
// predicate itself; the E2E test (tests/purchasing.spec.js) proves the wiring.
// (service.go:42-56)
func TestIsAdmin_Prove(t *testing.T) {
	cases := []struct {
		name string
		user *auth.User
		want bool
	}{
		{"nil user is not admin", nil, false},
		{"superadmin is admin", &auth.User{IsSuperadmin: true}, true},
		{"admin role is admin", &auth.User{Roles: []string{"admin"}}, true},
		{"admin among many roles is admin", &auth.User{Roles: []string{"manager", "admin", "cashier"}}, true},
		{"team_member is NOT admin", &auth.User{Roles: []string{"team_member"}}, false},
		{"manager is NOT admin", &auth.User{Roles: []string{"manager"}}, false},
		{"no roles is NOT admin", &auth.User{Roles: nil}, false},
		{"empty roles slice is NOT admin", &auth.User{Roles: []string{}}, false},
	}
	for _, c := range cases {
		if got := isAdmin(c.user); got != c.want {
			t.Errorf("%s: isAdmin() = %v, want %v", c.name, got, c.want)
		}
	}
}

// cutoffHasPassed mirrors the auto-lock DECISION arithmetic in
// scheduler.go:196-204 as a PURE function of an injected `now` (so it is
// testable without the wall clock or a DB). runCutoffCheck computes the
// most-recent day_of_week+HH:MM occurrence and auto-locks iff now is strictly
// after it. This helper re-derives that rule; the test below feeds a frozen now
// and asserts the boundary (AC-7: "cutoff time in the past → fires").
//
// NOTE: this proves the decision RULE in isolation. It does NOT prove the live
// cron wires this rule to time.Now()+the DB — that wiring is the PARKED fix
// (the production func has no clock seam). Kept explicit so the report's
// GREEN/PARK split is honest.
func cutoffHasPassed(now time.Time, dayOfWeek, hour, minute int) bool {
	daysBack := int(now.Weekday()) - dayOfWeek
	if daysBack < 0 {
		daysBack += 7
	}
	cutoffCandidate := time.Date(now.Year(), now.Month(), now.Day()-daysBack, hour, minute, 0, 0, now.Location())
	return now.After(cutoffCandidate)
}

// TestCutoffDecisionRule_Prove asserts the auto-lock decision rule (FR-19) at
// its boundary with a FROZEN clock. This is the deterministic decision proof
// the live cron can't give (no injectable clock — see PARK record above).
func TestCutoffDecisionRule_Prove(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("LoadLocation: %v", err)
	}
	// Cutoff configured for Wednesday (time.Weekday Wednesday = 3) at 17:00.
	const cutoffDOW, cutoffH, cutoffM = 3, 17, 0

	// Wednesday 2026-01-07 is a Wednesday (verify to keep the fixture honest).
	wed := time.Date(2026, 1, 7, 0, 0, 0, 0, loc)
	if wed.Weekday() != time.Wednesday {
		t.Fatalf("fixture date is not Wednesday: %s", wed.Weekday())
	}

	// One second BEFORE cutoff on cutoff day → has NOT passed → must NOT fire.
	before := time.Date(2026, 1, 7, 16, 59, 59, 0, loc)
	if cutoffHasPassed(before, cutoffDOW, cutoffH, cutoffM) {
		t.Errorf("cutoff should NOT have passed at %s (1s before 17:00 Wed)", before)
	}

	// One second AFTER cutoff → has passed → auto-lock DECISION fires (AC-7).
	after := time.Date(2026, 1, 7, 17, 0, 1, 0, loc)
	if !cutoffHasPassed(after, cutoffDOW, cutoffH, cutoffM) {
		t.Errorf("cutoff SHOULD have passed at %s (1s after 17:00 Wed)", after)
	}

	// A later day in the same week (Friday) → most-recent Wed cutoff is in the
	// past → still fires.
	fri := time.Date(2026, 1, 9, 9, 0, 0, 0, loc)
	if fri.Weekday() != time.Friday {
		t.Fatalf("fixture date is not Friday: %s", fri.Weekday())
	}
	if !cutoffHasPassed(fri, cutoffDOW, cutoffH, cutoffM) {
		t.Errorf("cutoff SHOULD have passed on Friday (Wed 17:00 already elapsed)")
	}

	// Earlier in the same week (Monday, before this week's Wed cutoff) → the
	// most-recent occurrence is LAST Wednesday, already elapsed → fires. This
	// documents the current rule (runCutoffCheck computes the most-recent past
	// occurrence, not strictly this-week's), so the assertion matches code.
	mon := time.Date(2026, 1, 5, 12, 0, 0, 0, loc)
	if mon.Weekday() != time.Monday {
		t.Fatalf("fixture date is not Monday: %s", mon.Weekday())
	}
	if !cutoffHasPassed(mon, cutoffDOW, cutoffH, cutoffM) {
		t.Errorf("per current rule, Monday's most-recent Wed cutoff is in the past → should fire")
	}
}
