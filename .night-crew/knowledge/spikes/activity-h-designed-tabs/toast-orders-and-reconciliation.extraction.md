# Extraction — toast-orders-and-reconciliation

Outcome: learned

Approach used: read `OrderDetails.csv` off the Toast SFTP export directory HQ
already syncs per business date, dialled with the production libraries
(`x/crypto/ssh` public-key + `pkg/sftp`, nil host-key callback — exactly
`backend/internal/toast/sftp.go`); `toast_orders` keyed `(business_date,
order_number)` with `ON CONFLICT … DO UPDATE`; the `scan_attempts` mirror
pulled server-side with a service-role JWT in `(scanned_at, id)` keyset
order. Three spikes, all exit 0 (one after two could-not-run attempts, see
Learned). Candidate design input, not an adoption (NFR-6).

Confirmed: decision 188 — `OrderDetails.csv` is present on every recent date
dir listed (20260928, 20260929, 20260930), enumerated not sampled; the
upsert key is unique in real data (77 orders, 0 duplicate keys) and a double
load leaves one row per order; money columns parse to integer cents; the
service role reads `scan_attempts` while a device JWT gets 403 (the
push-only RLS asymmetry from Activity A is intact).

Learned: (a) the OpenSSH `sftp` CLI is NOT a valid probe of this endpoint —
AWS Transfer closes its handshake after key exchange — so any future check
of the export must dial the way the worker does; (b) `TOAST_SFTP_KEY_PATH`
is relative to `backend/`, the server's working directory; (c) Toast
`Order #` in the real sample is digits only and 1–4 characters long
(handoff #2 answered: store as text, match exactly); (d) the same export
carries `PaymentDetails.csv`, `ModifiersSelectionDetails.csv`,
`TimeEntries.csv`, `AllItemsReport.csv` and the menu JSON exports — tips,
modifiers and labour are one `ReadDir` away if a later card wants them.

Plan change: the card's "first act: list the remote date dir" leg is already
done — drop the fixture-loader fallback and the `smtp-toast-ingest` revival
clause; the card ingests `OrderDetails.csv` directly. Record handoff #2's
answer in the ledger at triage. Roadmap and handoff updated 2026-10-01.
