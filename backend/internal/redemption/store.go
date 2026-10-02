package redemption

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RaceLostReconciled is the F4 domain event (§19.3 "Domain facts"): a synced
// offline_override attempt was arbitrated and the burn returned already_used —
// an after-the-fact double-redemption the Shift Manager follows up on. The
// counter never slowed for it; the customer is long gone (F4).
type RaceLostReconciled struct {
	TokenHash      string    // which code (the §4 hash — never a raw token)
	DeviceID       string    // which device lost the race
	Staff          string    // who submitted/synced the override (HQ session user)
	OrderNumber    string    // Toast order # if captured (§13)
	ScannedAt      time.Time // when the code was accepted at the counter
	Value          float64   // offer face value ($) as displayed at accept time
	ValueKnown     bool      // false → value unknown (e.g. F2 unverified code)
	UnverifiedCode bool      // F2: the override was on a code the replica couldn't verify
}

// PGRaceLostStore persists RaceLostReconciled into race_lost_notifications
// (HQ Postgres, migration 0077) — the Shift-Manager read-model entry the
// upcoming reconciliation-view card reads. Home decision + rationale in this
// card's merge-intent.
type PGRaceLostStore struct {
	Pool *pgxpool.Pool
}

// Emit writes the read-model row and logs the domain event. The write is
// synchronous by design: the arbitration response reports
// race_lost_reconciled only after the entry exists.
//
// B-424: the INSERT is idempotent on (code_token_hash, device_id, scanned_at) —
// WHICH code, WHICH device lost the race, WHEN the code was accepted at the
// counter. A reconciliation can be REPLAYED (the same synced offline_override
// attempt re-arbitrated, by a retry or by a second manager syncing the same
// device), and before this clause every replay produced another Shift-Manager
// ping for one loss. `staff` is deliberately outside the key: the same loss
// re-synced by someone else is the same loss.
//
// 🛑 The clause and migration 0084's unique index
// `race_lost_notifications_dedupe_uq` are ONE change in two files. Without the
// index Postgres has no inference target and this statement does not run at
// all; without the clause the index turns a replay into a 23505 at the
// arbitration response. Never land one alone.
//
// DO NOTHING and not DO UPDATE: the row is the notification the manager has
// already seen. Rewriting it on a replay would move it in a newest-first list
// for no new fact.
func (s PGRaceLostStore) Emit(ctx context.Context, ev RaceLostReconciled) error {
	var value any
	if ev.ValueKnown {
		value = ev.Value
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO race_lost_notifications
		    (code_token_hash, device_id, staff, order_number, scanned_at, value, unverified_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (code_token_hash, device_id, scanned_at) DO NOTHING`,
		ev.TokenHash, ev.DeviceID, ev.Staff, nullIfEmpty(ev.OrderNumber), ev.ScannedAt, value, ev.UnverifiedCode,
	)
	if err != nil {
		return fmt.Errorf("insert race_lost_notifications: %w", err)
	}
	slog.Warn("RaceLostReconciled",
		"code_token_hash", ev.TokenHash,
		"device_id", ev.DeviceID,
		"staff", ev.Staff,
		"order_number", ev.OrderNumber,
		"scanned_at", ev.ScannedAt,
		"value_known", ev.ValueKnown,
		"value", ev.Value,
		"unverified_code", ev.UnverifiedCode,
	)
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
