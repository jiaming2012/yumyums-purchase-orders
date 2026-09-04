# Extraction — scan-attempts-push-conflict

Outcome: learned

Approach used: push-only replication with a custom handler needing NO server
read-back — for each locally-queued `pending` attempt: POST `/rpc/redeem` as
the device JWT → POST `/scan_attempts` with the resolved status/reason →
return `[]`, with already-resolved rows skipped (`status !== 'pending'`); the
losing device's display data ("already used at 6:42pm by …") comes from its
own codes-side PULL replica, never from reading `scan_attempts` back. A
candidate the card's design.md adopts or not (NFR-6).

Confirmed: the unproven round trip on the losing device, against the
write-only table the RLS design imposes. Two devices queued offline attempts
for the same code; concurrent push; exactly one `accepted`, the other's LOCAL
row flipped to `rejected / already_used` carrying the winning device + time;
handler invocations bounded (no write-back loop); a device-role SELECT on
`scan_attempts` still answered 403; server-side enumeration showed exactly 2
attempt rows and `codes.redeemed_by` matching the winner.

Learned: two facts the sketch did not carry. (1) `scan_attempts.id` is a
**uuid** — devices must generate `crypto.randomUUID()`, not app-prefixed
strings (the spike's first run drew PostgREST 400 on exactly this).
(2) **Redeem-then-land is not atomic client-side**: if landing the attempt row
fails after `redeem()` succeeded, RxDB's push retry re-runs the handler and
the re-burn answers `already_used` to the WINNING device itself — a transient
landing failure would mis-flip the winner's UI to "already used." The spike
hit this retry path on its red first run; the green run's resolved-row skip
covers the patched-row loop but not that partial-failure window.

Plan change: the card's push handler must treat `already_used` where
`codes.redeemed_by == own device` as accepted (or persist the burn outcome
locally before landing the attempt row) — anchored as GAP-1 in this goal's
ledger `## Comebacks`; the fixing card owes the one validation run that
exercises the land-fails-after-redeem window. Device attempt ids are
`crypto.randomUUID()` (§4's "generated on device" is literal).
