# Spikes — scanner-polish

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> Hand-run convention (see `campaign-codes-api.md` header). Spike 2 runs
> against the LOCAL spike-supabase substrate (reconcile mode).

## The goal, and which legs need a spike

The card (H6): four scanner findings closed in one serial tail — B-446 (refuse
at scan time), B-447 (campaign name + code last-4 on the offer card), B-440
(divert predicate must agree with the constraint), B-436 (fail closed with no
policy source; decision 191). Two of them rest on premises a script can settle
now: that B-440 reproduces exactly as filed on the shipped handler (so the
fix is the predicate, not the constraint), and that B-447 is a pull-selection
change rather than a schema/RLS change.

## Spike: divert-predicate-diverts-on-unverified-alone

- proves: the SHIPPED `makePushHandler` diverts to the unverified landing on
  `doc.unverified_code` alone; with `offline_override:false` the (tightened)
  constraint answers 400, the handler throws, and the row stays `pending` —
  the head-of-line poison B-440 describes. This is the red baseline the card
  flips by tightening the predicate to `unverified_code && offline_override`.
- plan: import the real module with fake collections and a fake fetch that
  answers 400 for the poison shape; assert `land-unverified` was attempted,
  the handler threw, and the doc is still pending.
- script: .night-crew/spikes/activity-h-designed-tabs/scanner-polish/01-divert-predicate-diverts-on-unverified-alone.sh

## Spike: campaign-name-selectable-by-device

- proves: the device (authenticated) role may `select=id,name,requires_online,updated_at`
  on `campaigns` — the Activity A grant is table-wide, not column-scoped — so
  B-447 is a change to the replica's pull selection + the offer card, not a
  schema card.
- plan: mint a device JWT, GET campaigns with the widened select against the
  local PostgREST, assert 200 and `name` non-null on every row.
- script: .night-crew/spikes/activity-h-designed-tabs/scanner-polish/02-campaign-name-selectable-by-device.sh

## Verdict (hand-run 2026-10-01)

- **divert-predicate-diverts-on-unverified-alone: passed** — exit 0, first
  run: request log `land-unverified`; handler threw
  `unverified scan_attempts insert answered HTTP 400`; doc left `pending`.
  B-440 reproduces as filed.
- **campaign-name-selectable-by-device: passed** — exit 0, first run: HTTP
  200, 2 fixture rows, both with `name`.

## Corrections

- none — no agent-reached corrections

## Comebacks

- none recorded — no new gap; B-440 and B-447 are already filed and are what the card closes

## Review

- signed: operator, 2026-10-01 — covers 0 correction(s) (no agent-reached corrections; reviewed in the same batch sitting)
