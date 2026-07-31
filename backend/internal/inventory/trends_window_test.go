package inventory

import (
	"testing"
	"time"

	"github.com/yumyums/hq/internal/users"
)

// ── trendsWindow evaluates its 12-week window in the APP timezone ────────────
//
// RED BEFORE THE FIX. trendsWindow used to read the weekday and format the
// dates in whatever location the caller's time.Time carried — and its one
// caller passes a bare time.Now(), which in the production container is UTC.
// So the Trends tab's 12-week COGS window was on server-local while the
// Recipes/Cost window (recipes.costWindow) and the period-summary date cast
// (pendingPeriodDateExpr) were on America/New_York.
//
// Two 12-week COGS windows on two different zones is precisely the
// "two boundaries disagreeing" state card A1 exists to end, so this is a
// defect in its own right and not a tidy-up.
//
// EXPECTED FAILURE before the fix, on the UTC-Monday case:
//
//	from = 2026-05-11, to = 2026-07-27   (UTC: it is Monday)
//
// want, in New York, where the same instant is Sunday evening:
//
//	from = 2026-05-04, to = 2026-07-26
//
// A full week off, plus a "today" that names tomorrow.

func TestTrendsWindow_IsEvaluatedInAppTimezone(t *testing.T) {
	ny, err := time.LoadLocation(users.DefaultTimezone)
	if err != nil {
		t.Fatalf("LoadLocation(%s): %v", users.DefaultTimezone, err)
	}

	cases := []struct {
		name string
		// now is deliberately constructed in a NON-app location, the way the
		// production container hands time.Now() to trendsWindow.
		now      time.Time
		wantFrom string
		wantTo   string
	}{
		{
			// 01:30 UTC Monday 2026-07-27 == 21:30 New York Sunday 2026-07-26.
			// The zones disagree about the weekday AND the calendar date, so
			// this case cannot pass by coincidence in either zone.
			name:     "UTC has rolled into Monday, New York is still Sunday evening",
			now:      time.Date(2026, 7, 27, 1, 30, 0, 0, time.UTC),
			wantFrom: "2026-05-04",
			wantTo:   "2026-07-26",
		},
		{
			// 02:15 UTC Wednesday 2026-06-10 == 22:15 New York Tuesday
			// 2026-06-09. Same ISO week in both zones, so `from` agrees — but
			// `to` ("today") still differs by a day. This case proves the fix
			// is not only about the week boundary.
			name:     "mid-week, UTC is a day ahead of New York",
			now:      time.Date(2026, 6, 10, 2, 15, 0, 0, time.UTC),
			wantFrom: "2026-03-23",
			wantTo:   "2026-06-09",
		},
		{
			// A time-of-day where the two zones agree, to prove the fix does
			// not simply shift everything by a day unconditionally.
			name:     "midday, both zones agree",
			now:      time.Date(2026, 6, 10, 16, 0, 0, 0, time.UTC),
			wantFrom: "2026-03-23",
			wantTo:   "2026-06-10",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := trendsWindow(tc.now)
			if got.From != tc.wantFrom || got.To != tc.wantTo {
				t.Errorf("trendsWindow(%s) = {from:%s to:%s}, want {from:%s to:%s}\n"+
					"  the same instant in %s is %s",
					tc.now.Format(time.RFC3339), got.From, got.To, tc.wantFrom, tc.wantTo,
					users.DefaultTimezone, tc.now.In(ny).Format(time.RFC3339))
			}
			if got.Weeks != TrendsWeeks {
				t.Errorf("weeks = %d, want %d", got.Weeks, TrendsWeeks)
			}
		})
	}
}

// TestTrendsWindow_AgreesWithTheAppTimezoneCalendarDate states the contract
// independently of trendsWindow's own arithmetic: whatever else it does, `to`
// is TODAY in the app timezone. If someone later "fixes" trendsWindow by
// hardcoding a zone that is not users.DefaultTimezone, this fails.
func TestTrendsWindow_ToIsTodayInTheAppTimezone(t *testing.T) {
	ny, err := time.LoadLocation(users.DefaultTimezone)
	if err != nil {
		t.Fatalf("LoadLocation(%s): %v", users.DefaultTimezone, err)
	}

	// 03:00 UTC is 23:00 the PREVIOUS day in New York year-round (EST -5,
	// EDT -4), so this instant always straddles the date line between the two.
	now := time.Date(2026, 9, 3, 3, 0, 0, 0, time.UTC)

	got := trendsWindow(now)
	want := now.In(ny).Format("2006-01-02")
	if got.To != want {
		t.Errorf("trendsWindow(%s).To = %s, want %s (today in %s)",
			now.Format(time.RFC3339), got.To, want, users.DefaultTimezone)
	}
	if got.To == now.UTC().Format("2006-01-02") {
		t.Errorf("trendsWindow(%s).To = %s, which is the UTC date — the window "+
			"is still being evaluated in server-local time",
			now.Format(time.RFC3339), got.To)
	}
}
