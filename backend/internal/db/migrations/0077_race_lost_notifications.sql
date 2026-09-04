-- +goose Up
BEGIN;

-- ===========================================================================
-- 0077 — race_lost_notifications: the F4 Shift-Manager read-model.
-- Card `gstate-arbitration-machine` (Activity D, run 20260905).
-- ===========================================================================
--
-- F4 (docs/qr-offline-redemption-handoff.md §19.4): when a synced
-- offline_override attempt is arbitrated and the burn returns already_used,
-- the server emits a RaceLostReconciled domain event and a Shift-Manager
-- notification / read-model entry is created (code, device, staff, time,
-- value) for follow-up. The client scan flow has NO state for this outcome —
-- the customer left before reconciliation; this table is where the loss
-- becomes visible.
--
-- WHY HQ POSTGRES (not a supabase/ migration): the consumer is the Shift
-- Manager inside the HQ app (session + grant stack; the reconciliation-view
-- card reads from here), the emitter is HQ Go's arbitrator which holds this
-- pool, and a manager-only table on the device-facing Supabase substrate
-- would demand new RLS policy design for zero benefit. Full rationale in the
-- card's merge-intent.
--
-- Field set is exactly F4's "code, device, staff, time, value" plus the
-- order number (§13 attribution join key), the F2 unverified_code flag
-- (an unverifiable-code override that lost is the highest-priority follow-up)
-- and created_at. Deliberately NO acknowledgement/assignment workflow columns
-- — who else gets told and when is notification POLICY, out of this card's
-- scope (its PARK boundary).
CREATE TABLE race_lost_notifications (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code_token_hash  TEXT NOT NULL,             -- §4: the hash, never a raw token
  device_id        TEXT NOT NULL,             -- which device lost the race
  staff            TEXT NOT NULL,             -- HQ user (email) who submitted/synced the override
  order_number     TEXT,                      -- Toast order # if captured (§13)
  scanned_at       TIMESTAMPTZ NOT NULL,      -- when the code was accepted at the counter
  value            NUMERIC,                   -- offer face value at accept time; NULL = unknown (F2)
  unverified_code  BOOLEAN NOT NULL DEFAULT FALSE, -- F2: override on a code the replica couldn't verify
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The reconciliation view lists newest-first.
CREATE INDEX race_lost_notifications_created_at_idx
  ON race_lost_notifications (created_at DESC);

COMMENT ON TABLE race_lost_notifications IS
  'F4 RaceLostReconciled read-model (card gstate-arbitration-machine): one row per '
  'offline-override attempt that lost the double-redemption race at reconciliation. '
  'Written by backend/internal/redemption (PGRaceLostStore.Emit); read by the '
  'reconciliation view. Append-only from the app''s perspective.';

COMMIT;

-- +goose Down
BEGIN;

DROP TABLE IF EXISTS race_lost_notifications;

COMMIT;
