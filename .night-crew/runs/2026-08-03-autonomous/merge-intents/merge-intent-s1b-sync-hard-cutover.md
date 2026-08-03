# merge-intent — S1b `sync-hard-cutover`

Run `overnight-20260803`, Track A, second and last card. Branch
`card/s1b-sync-hard-cutover`, cut off `overnight-20260803` **after S1a merged** (B-50).

Written BEFORE implementing (§15ad.65). **Updated in place once, after the card PARKED** —
the "shared files touched" table below is the *as-parked* truth, not the as-planned intent.
See `park-s1b-sync-hard-cutover.md` for why.

---

## 🛑 THE CARD PARKED. NO PRODUCTION CODE, SQL, POLICY, MIGRATION OR TEST WAS WRITTEN.

`git diff overnight-20260803..card/s1b-sync-hard-cutover -- . ':!.night-crew'` is **empty**.
Every file this note lists is a night-crew knowledge or run artifact. In particular:

- `workflows.html` — **byte-unchanged**
- `sync.js` — **still present, byte-unchanged** (not deleted)
- `sync-rxdb/bootstrap.js`, `sync-rxdb/client.js` — **byte-unchanged**
- `backend/internal/sync/**` — **byte-unchanged** (nothing deleted; `jwtbridge*`, `proxy*`,
  `rowvisibility_rls_test.go`, `access_test.go`, `spikestack_gate_test.go` all intact)
- `backend/internal/workflow/**`, `backend/cmd/server/main.go` — **byte-unchanged**
  (`/saveResponse` still mounted at `main.go:558`)
- `build-sw.js`, `sw.js`, `package.json`, `version.go`, `tests/**` — **byte-unchanged**
- `HQ_SYNC_REST_URL` — **not set anywhere.** The standing interlock (decision 119) is
  unchanged and still in force. Nothing was deployed.

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `.night-crew/runs/2026-08-03-autonomous/merge-intents/` | This note. |
| `.night-crew/runs/2026-08-03-autonomous/park-s1b-sync-hard-cutover.md` | The park write-up: the trigger, the measurements, and the three options for the successor card. |
| `.night-crew/runs/2026-08-03-autonomous/DECISIONS-NEEDED.md` | **Created by this card.** Two operator forks: the cutover data plane (F-1) and B-61's list narrowing (F-2). |
| `.night-crew/knowledge/BACKLOG.md` | Discoveries filed under the standing scope freeze, numbers **B-65..B-70** (allocated to this card up front), each with a destination. |
| `.night-crew/knowledge/roadmap.md` | The card's status flip to PARKED, in the same change set, with the trigger named. |

**NOT touched, and on this card that is the whole point:** every path in the card's
footprint. A park that edited half its footprint would leave a tree nobody can reason
about at triage.

---

## What must survive any merge

1. **The park verdict itself.** The card did not fail to finish; it hit a recorded PARK
   trigger — *"PARK if retiring `/saveResponse` turns out to reopen ledger decision 49"* —
   and decision 49 is reopened by measurement, not by opinion. A successor that treats
   this as "S1b ran out of time" will walk into the same wall.
2. **The finding that `backend/internal/sync` is load-bearing for the WORKFLOW package,
   not only for the op-log.** `backend/cmd/server/main.go:47-95`'s `workflowOpRouter`
   routes `SET_FIELD` → `workflow.SaveResponseFunc` and `SUBMIT_CHECKLIST` →
   `ValidateFailNotesFunc` + `ValidateResubmitPhotoFunc` + `SubmitChecklistFunc`.
   Deleting `ops.go`/`handler.go` deletes the transport for all of that validation.
   **This is a second footprint correction, of the same kind and severity as the one the
   slate already carries for `jwtbridge`/`proxy`.**
3. **The finding that `autoSaveField` does not exist.** The card, the roadmap bullet,
   `bootstrap.js:9` and `conflict-notice-ui.js:23` all name it as the live write path.
   It is defined nowhere in the tree, and `workflows.html:2219` calls it — a live
   `ReferenceError` on the fail-note-photo path (filed **B-65**). The real path is
   `debouncedSaveField` → `submitOp('SET_FIELD')` → `POST /ops`.
4. **B-65..B-70 with their destinations.** Six findings, none of them fixed here.
5. **The E-KR2 evidence measurement: 59 subtests, `HQ_SYNC_SUBSTRATE_OPTIONAL` unset.**
   Re-measured on this branch; the count moved 54 → 59 with S1a, as expected.

---

## What is safe to drop

- The **prose volume** of the park note. It is long because it is carrying measurements a
  successor would otherwise re-derive; compress freely, but keep every `file:line`.
- The exact **wording** of the BACKLOG entries B-65..B-70, provided each keeps its
  destination and its measured evidence.
- **B-65..B-70's numbers** if another leg of this run filed the same finding first — keep
  one copy, keep the lower number, keep the destination.
- The DECISIONS-NEEDED **framing** of F-1 as three options. If triage prefers a different
  cut of the same question, the options are not load-bearing — the question is.

---

## Nothing here (stated explicitly, not omitted)

- **Production code:** nothing here.
- **Migrations:** nothing here. No `0005_*.sql`; `sync-schema/sql/` is byte-unchanged.
- **Schema / collections:** nothing here. `sync-schema/collections.js` byte-unchanged.
- **Write path:** nothing here. Both `/saveResponse` and `/ops` are exactly as S1a left them.
- **Deploy / env:** nothing here. `HQ_SYNC_REST_URL` unset; no compose, Taskfile or `.env`
  touched; **nothing was deployed.**
- **Deletions:** nothing here. No file was deleted anywhere in the tree.
- **`sw.js` / precache:** nothing here. The count is untouched at 31 and **B-54's pin was
  NOT written** — the pin's rider is "write it and re-base it in the same commit if S1b
  moves the number", and S1b moved nothing. Re-destined, see B-54's disposition in the
  park note.
- **Conflicts expected with other legs of this run:** nothing known. This branch touches
  only `.night-crew/` artifacts, and the only overlap is `BACKLOG.md` / `roadmap.md`,
  where the edits are append-and-flip.
