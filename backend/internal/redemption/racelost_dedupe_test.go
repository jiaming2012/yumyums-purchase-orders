package redemption

import (
	"context"
	"testing"
	"time"
)

// TestRaceLostNotificationDedupe is card H3a's third done_when row and the fix
// for B-424: a REPLAYED reconciliation must leave ONE manager notification, not
// one per replay.
//
// The replay is real, not hypothetical. The same synced offline_override attempt
// reaches arbitration again whenever the arbitration response is retried, when a
// second device syncs the same attempt record, or when a manager re-runs the
// reconciliation over a window already processed — and before this card every one
// of those produced another Shift-Manager ping for a single lost race. The
// customer left once.
//
// The dedupe key is the F4 identity of the event: WHICH code (the §4 token hash,
// never a raw token), WHICH device lost the race, WHEN the code was accepted at
// the counter. `staff` is deliberately OUTSIDE the key — the same loss re-synced
// by a different manager is the same loss — and the test proves that by replaying
// with a different staff and a different order number.
func TestRaceLostNotificationDedupe(t *testing.T) {
	resetF4(t)
	ctx := context.Background()
	store := PGRaceLostStore{Pool: testPool}

	scannedAt := time.Date(2026, 10, 1, 18, 42, 7, 0, time.UTC)
	ev := RaceLostReconciled{
		TokenHash:      "sha256:b424-dedupe-fixture",
		DeviceID:       "device-b",
		Staff:          "manager-a@yumyums.com",
		OrderNumber:    "1042",
		ScannedAt:      scannedAt,
		Value:          4.00,
		ValueKnown:     true,
		UnverifiedCode: true,
	}

	if err := store.Emit(ctx, ev); err != nil {
		t.Fatalf("first emit: %v", err)
	}

	// Replay 1: byte-identical.
	if err := store.Emit(ctx, ev); err != nil {
		t.Fatalf("replayed emit must not ERROR (the unique index needs the "+
			"ON CONFLICT DO NOTHING clause to absorb it): %v", err)
	}

	// Replay 2: same loss, re-synced by someone else with a late-captured order
	// number. Still the same (code, device, accept time) — still one row.
	ev2 := ev
	ev2.Staff = "manager-b@yumyums.com"
	ev2.OrderNumber = "1042-A"
	if err := store.Emit(ctx, ev2); err != nil {
		t.Fatalf("third emit: %v", err)
	}

	var n int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM race_lost_notifications
		  WHERE code_token_hash = $1 AND device_id = $2 AND scanned_at = $3`,
		ev.TokenHash, ev.DeviceID, scannedAt).Scan(&n); err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if n != 1 {
		t.Fatalf("B-424: a replayed reconciliation created %d notifications, want 1", n)
	}

	// DO NOTHING, not DO UPDATE: the row is the notification the manager has
	// already seen, so the first one's facts are the ones that stand.
	var staff, orderNumber string
	if err := testPool.QueryRow(ctx,
		`SELECT staff, order_number FROM race_lost_notifications
		  WHERE code_token_hash = $1`, ev.TokenHash).Scan(&staff, &orderNumber); err != nil {
		t.Fatalf("read the surviving row: %v", err)
	}
	if staff != "manager-a@yumyums.com" || orderNumber != "1042" {
		t.Fatalf("the surviving row was rewritten by a replay (staff=%q order=%q); "+
			"DO NOTHING must leave the first notification alone", staff, orderNumber)
	}

	// A genuinely DIFFERENT loss is still a second notification — the dedupe must
	// not swallow a second device losing the same code, nor the same device
	// losing it at a different accept time.
	evOtherDevice := ev
	evOtherDevice.DeviceID = "device-c"
	if err := store.Emit(ctx, evOtherDevice); err != nil {
		t.Fatalf("emit for a different device: %v", err)
	}
	evOtherTime := ev
	evOtherTime.ScannedAt = scannedAt.Add(time.Minute)
	if err := store.Emit(ctx, evOtherTime); err != nil {
		t.Fatalf("emit for a different accept time: %v", err)
	}
	var total int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM race_lost_notifications`).Scan(&total); err != nil {
		t.Fatalf("count all: %v", err)
	}
	if total != 3 {
		t.Fatalf("after 1 real loss + 2 replays + 2 genuinely different losses: %d rows, want 3", total)
	}
}
