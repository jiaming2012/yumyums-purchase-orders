# Spikes — toast-orders-and-reconciliation

Activity: Activity H — Campaign admin, subscribers, stats (the designed tabs)

> Hand-run convention (see `campaign-codes-api.md` header). Spike 1 dials the
> LIVE Toast AWS Transfer SFTP endpoint READ-ONLY (two `ReadDir`s) with the
> key the Toast worker already uses from `backend/.env`; nothing is
> downloaded, nothing written, the key path is never printed. Spike 2 uses a
> throwaway database on the `:5434` test cluster (created and dropped).
> Spike 3 uses the LOCAL spike-supabase substrate (reconcile mode).

## The goal, and which legs need a spike

The card (H3): `toast_orders` from `OrderDetails.csv` on the existing SFTP
sync, a `scan_attempts` mirror pulled server-side, the reconciliation engine
(matched / unmatched / orphan, decline-with-note, reopen), per-slice money,
B-424's dedupe. Three premises carry the card: decision 188 (the file is on
the export at all), the upsert key (`(business_date, order_number)` is
unique in real data and makes a double arrival idempotent), and the mirror's
read path (only the service role can read attempts; keyset order accepted).

## Spike: orderdetails-on-sftp

- proves: the per-date export directory HQ already syncs
  (`/<EXPORT_ID>/<YYYYMMDD>/`) contains `OrderDetails.csv` for every recent
  date listed — the file set per date dir is enumerated, not sampled (B-216).
  Falsified ⇒ decision 188 falls, H3 lands on a fixture loader and
  `smtp-toast-ingest` returns to PLANNED.
- plan: dial exactly as `backend/internal/toast` does (`x/crypto/ssh`
  public-key auth, nil host-key callback, `pkg/sftp`) from a throwaway Go
  module; `ReadDir` the root, take the three newest `YYYYMMDD` dirs, `ReadDir`
  each, assert the file.
- script: .night-crew/spikes/activity-h-designed-tabs/toast-orders-and-reconciliation/01-orderdetails-on-sftp.sh

## Spike: orderdetails-upsert-idempotent

- proves: on the real sample (`~/projects/yumyums/OrderDetails.csv`, 77
  orders, one business date) `(business_date, order_number)` has no
  duplicates (the duplicate set is enumerated), the money columns parse to
  integer cents, and loading the same file twice with
  `ON CONFLICT … DO UPDATE` leaves exactly one row per order.
- plan: python parses + emits upserts; `docker exec yumyums-test-pg psql` into
  a throwaway `spike_h3_<stamp>` db; load twice; compare count to distinct keys.
- script: .night-crew/spikes/activity-h-designed-tabs/toast-orders-and-reconciliation/02-orderdetails-upsert-idempotent.sh

## Spike: scan-attempts-service-read

- proves: a service-role JWT can GET `scan_attempts` with
  `order=scanned_at.asc,id.asc` (the mirror's keyset) → 200 + array, AND a
  device JWT still cannot (401/403) — the push-only asymmetry Activity A
  proved is intact, so the server is the only reader.
- plan: mint both tokens with the committed throwaway secret; two curls.
- script: .night-crew/spikes/activity-h-designed-tabs/toast-orders-and-reconciliation/03-scan-attempts-service-read.sh

## Verdict (hand-run 2026-10-01)

- **orderdetails-on-sftp: could-not-run (run 1, run 2) → passed (run 3)** —
  run 1: `TOAST_SFTP_KEY_PATH` is a path relative to `backend/` (the server's
  working dir) and the script resolved it against the repo root → key "not
  found". Run 2 (path fixed): the OpenSSH `sftp` CLI handshake was closed by
  AWS Transfer after key exchange (`Connection closed`) — the CLI is not the
  production dial path. Run 3 (production libraries): exit 0. **Date dirs
  20260928, 20260929, 20260930 each hold** `AccountingReport.xls
  AllItemsReport.csv ItemSelectionDetails.csv MenuExportV2_….json
  MenuExport_….json ModifiersSelectionDetails.csv OrderDetails.csv
  PaymentDetails.csv TimeEntries.csv`. Decision 188 holds; bonus finding for
  the card: `PaymentDetails.csv` and `ModifiersSelectionDetails.csv` are on
  the same export (tips / modifiers, if the by-item "also bought" card ever
  wants them).
- **orderdetails-upsert-idempotent: passed** — exit 0, first run. 77 rows,
  77 distinct keys, duplicate set `[]`; order-number shapes seen (set):
  1-, 2- and 4-digit (`9`, `99`, `9999`) — so handoff #2's "format" answer is
  *digits only, variable length, store as text*; double load → 77 rows;
  Σ discount 341¢ / Σ amount 145,473¢ parsed clean.
- **scan-attempts-service-read: passed** — exit 0, first run. service_role
  200 `[]` (no attempts seeded — the shape, not the content, is the premise);
  device 403.

## Corrections

- **the SFTP spike dials with the production libraries, not the OpenSSH CLI**
  — the agents replaced the `sftp -b` listing with a throwaway Go program
  using `golang.org/x/crypto/ssh` + `github.com/pkg/sftp` exactly as
  `backend/internal/toast/sftp.go` does (nil host-key callback included),
  after the CLI's handshake was closed post-KEX by AWS Transfer. Premise
  unchanged; the probe was wrong, not the claim. Also: the key path is
  resolved against `backend/` first, as the server does. Queued for the
  operator's batch review.

## Comebacks

- none recorded — no gap; the one finding (order-number format) answers handoff #2 rather than opening a gap

## Review

- signed: operator, 2026-10-01 — covers 1 correction(s) (batch sitting, presented as user stories at the operator's request; "Sign all three")
