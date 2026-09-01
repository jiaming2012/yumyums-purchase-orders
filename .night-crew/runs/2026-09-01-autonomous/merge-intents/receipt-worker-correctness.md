# Merge intent — `receipt-worker-correctness` (Card 1, Track A)

Closes **B-28** and **B-175**, two measured defects in `backend/internal/receipt/worker.go`.

## Shared files touched

- `.night-crew/knowledge/roadmap.md` — flip this card's status `PLANNED → DONE` in the
  final commit (required roadmap/status flip). One line, this card's bullet only.

Everything else is confined to my footprint:
- `backend/internal/receipt/worker.go` — both fixes (parseEventDate zone+observability; blob/attachment pairing).
- `backend/internal/receipt/worker_test.go` — red-first regression tests for both defects.

No other shared file (no `sw.js`, no HTML/JS, no migrations, no `night-crew.toml`,
no contracts, no `version.go`/`package.json`) is touched — **nothing here** beyond the
roadmap line above.

## What must survive any merge

- **B-28 (parseEventDate):** the fallback for an unparseable `CreatedAt` must format the
  date in `users.DefaultTimezone` (the app's single-source-of-truth zone every other
  timezone-sensitive site uses — `recipes/cost.go`, `purchasing/service.go`,
  `inventory/trends.go`, etc.), NOT `time.Now()` in server-local/UTC. AND the unparseable
  path must be **observable** — a WARN log naming the tx-less raw value so a wrong COGS
  period is distinguishable from a correct one. A future card that unifies the zone
  (card A1's successor) must not silently revert this to `time.Now()`.
- **B-175 (blob/attachment misindex):** the live ingest download→upload loop in
  `runIngestCycle` must pair each downloaded blob with ITS OWN attachment via a single
  struct, so a skipped (failed) download does not shift every later blob onto the wrong
  attachment's `FileName`/`URL`. Do not reintroduce parallel `blobs []FileBlob` +
  `tx.Attachments[i]` indexing. `recoverOneTx` (all-or-nothing) already avoids this; the
  live path now does too, keeping its skip-and-continue semantics.
- The two red-first regression tests in `worker_test.go` guard both — must survive.

## What is safe to drop

- Exact WARN log wording / message phrasing in the B-28 fallback — any log at WARN level
  that carries the raw unparseable value is acceptable; a merge may reword it.
- The chosen struct name for the blob+attachment pairing (`blobWithAttachment` or similar)
  — cosmetic; any equivalent single-struct pairing satisfies the invariant.
