package purchasing

import (
	"testing"
	"time"

	"github.com/yumyums/hq/internal/users"
)

// ─────────────────────────────────────────────────────────────────────────────
// Card A1 · app-timezone-unify-new-york — RED-FIRST BOUNDARY PROOF
//
// The operator ruled the app's timezone is America/New_York (ledger T-26
// decision 83). users.DefaultTimezone already encodes that ruling. This test
// pins CurrentWeekStart — the Monday every purchasing week hangs off — to that
// same zone, using a FROZEN clock (weekStartNow) because the zone is otherwise
// unobservable: for ~23 hours of every day Chicago and New York agree on the
// calendar date, and the disagreement is exactly the bug.
//
// The decisive instant is 2026-07-27T00:30:00-04:00:
//
//	New York : Monday    2026-07-27 00:30 EDT  → week starts 2026-07-27
//	Chicago  : Sunday    2026-07-26 23:30 CDT  → week starts 2026-07-20
//
// A FULL WEEK apart, for one hour every Sunday night, on the boundary the
// purchase-order table is keyed by (purchase_orders.week_start is UNIQUE).
//
// EXPECTED: RED while CurrentWeekStart loads America/Chicago.
//           GREEN once it loads users.DefaultTimezone.
// ─────────────────────────────────────────────────────────────────────────────

// withFrozenClock runs fn with weekStartNow pinned to at, restoring it after.
func withFrozenClock(t *testing.T, at time.Time, fn func()) {
	t.Helper()
	prev := weekStartNow
	weekStartNow = func() time.Time { return at }
	defer func() { weekStartNow = prev }()
	fn()
}

func TestCurrentWeekStart_UsesAppTimezone(t *testing.T) {
	ny, err := time.LoadLocation(users.DefaultTimezone)
	if err != nil {
		t.Fatalf("LoadLocation(%s): %v", users.DefaultTimezone, err)
	}
	chi, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("LoadLocation(America/Chicago): %v", err)
	}

	// Fixture honesty: assert the pivot instant really is Monday in New York and
	// Sunday in Chicago, so a tzdata surprise fails loudly instead of silently
	// making the test tautological.
	pivot := time.Date(2026, 7, 27, 0, 30, 0, 0, ny)
	if got := pivot.In(ny).Weekday(); got != time.Monday {
		t.Fatalf("fixture: pivot is %s in New York, want Monday", got)
	}
	if got := pivot.In(chi).Weekday(); got != time.Sunday {
		t.Fatalf("fixture: pivot is %s in Chicago, want Sunday", got)
	}

	cases := []struct {
		name string
		now  time.Time
		want string
	}{
		{
			// THE BOUNDARY. New York has already rolled to the new purchasing
			// week; Chicago has not. Chicago answers 2026-07-20 here.
			name: "Monday 00:30 New York (= Sunday 23:30 Chicago)",
			now:  pivot,
			want: "2026-07-27",
		},
		{
			// One hour earlier: Sunday in BOTH zones. Both answer 2026-07-20,
			// so this case is a control — it must stay green either way and
			// proves the test is not simply asserting "New York" everywhere.
			name: "Sunday 23:30 New York (= Sunday 22:30 Chicago) — control",
			now:  time.Date(2026, 7, 26, 23, 30, 0, 0, ny),
			want: "2026-07-20",
		},
		{
			// Mid-week control, far from any boundary.
			name: "Thursday 12:00 New York — control",
			now:  time.Date(2026, 7, 30, 12, 0, 0, 0, ny),
			want: "2026-07-27",
		},
		{
			// Winter-side boundary (EST/CST, not EDT/CDT) so the proof is not
			// an artifact of a single DST offset.
			name: "Monday 00:30 New York in January (= Sunday 23:30 Chicago)",
			now:  time.Date(2026, 1, 5, 0, 30, 0, 0, ny),
			want: "2026-01-05",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			withFrozenClock(t, tc.now, func() { got = CurrentWeekStart() })
			if got != tc.want {
				t.Errorf("CurrentWeekStart() at %s = %q, want %q\n"+
					"  instant in New York: %s\n"+
					"  instant in Chicago : %s",
					tc.now.Format(time.RFC3339), got, tc.want,
					tc.now.In(ny).Format("Mon 2006-01-02 15:04 MST"),
					tc.now.In(chi).Format("Mon 2006-01-02 15:04 MST"))
			}
		})
	}
}
