# Extraction — scanner-polish

Outcome: confirmed

Approach used: for B-440, drive the real `makePushHandler` with fake
collections and a fetch that answers 400 for the poison row shape; for
B-447, a device-JWT `select=id,name,requires_online,updated_at` against the
local PostgREST. Two spikes, both exit 0 on the first run. Candidate input
for the card, not an adoption (NFR-6).

Confirmed: B-440 reproduces exactly as filed on the shipped handler — divert
on `unverified_code` alone → constraint 400 → the handler throws → the row
stays `pending` (retry-forever head-of-line poison) — so the fix is the
predicate (`unverified_code && offline_override`), not the constraint. B-447
is a pull-selection + offer-card change, not a schema or RLS card: the
device role may read `campaigns.name` today.

Learned: (nothing new)

Plan change: none — the card closes B-440, B-446, B-447 and B-436
(fail-closed, decision 191) as authored.
