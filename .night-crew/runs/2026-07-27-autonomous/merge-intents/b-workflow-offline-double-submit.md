# Merge intent — Card B `workflow-offline-double-submit`

Branch: `card/b-workflow-offline-double-submit` (cut from `overnight-20260727` @ `2a78d89`,
i.e. a tree that already contains Card A). Written BEFORE implementation, as the card's first
commit. Required per DESIGN §15ad.65.

## Card in one line

Offline submit → reopen → submit again writes **two** `checklist_submissions` rows, because
`submitChecklistToAPI` mints a fresh `idempotency_key` on every call and the (correct) `err.offline`
handler leaves the checklist editable. Fix **client-side**: before minting, look up whether this
template already has a submission sitting in the IndexedDB `submitQueue`, and if so reuse its
`idempotency_key` (and its queue key) rather than minting a new one. Ledger T-23 decision 60.

## Shared files touched

- **`workflows.html` — the card.** One function: `submitChecklistToAPI` (`:1633-1670` on the
  pre-fix tree; the `idempotency_key: generateUUID()` line the roadmap card cites as `:1656` is at
  exactly `:1656` on this tree). **I own only the payload-construction head of that one function.**
  Specifically: the `submitPayload` object literal and a small `await` immediately above it that
  reads the offline queue. I do **not** touch the `responses` map above it, the `try`/`catch`
  around `api('POST','submitChecklist')` below it, or anything else in the file.
  **`sync-rxdb-schema-and-replication` (Card C) also owns `workflows.html`** and is cut from a tree
  that already contains this change, so the overlap should be free. If it does collide: the
  resolution is **keep this card's `submitPayload` head** — the reuse lookup and the two
  `queued ? … : generateUUID()` ternaries — and take C's side for everything else in the file.
  The two changes are disjoint in intent; there is no version of this where one supersedes
  the other.
- **`workflows.html:2781` (`err.offline` branch) — READ, NOT EDITED.** Named here because the
  roadmap card cites it (as `:2778`; it has drifted three lines). Its behaviour — return to the
  list, do **not** push into `MY_SUBMISSIONS`, leaving the checklist editable — is **deliberate and
  correct** (decision 60 says so explicitly). A merge that "fixes" it by pushing an optimistic
  submission is a regression, not a repair.
- **`sync.js` — READ, NOT EDITED.** It owns the queue (`enqueueSubmission`, `drainQueue`, the
  `submitQueue` object store keyed on `id`) and already exports `window.getDB` / `window.idbGetAll`
  (`sync.js:100-104`). The fix consumes those existing exports; no new export, no change to the
  store schema, no change to the drain loop. This was checked before writing, precisely so the
  card could hold its stated `workflows.html` footprint.
- **`tests/workflows.spec.js`** — new red-first tests appended to the existing
  `test.describe('Offline sync')` block (which already contains `[GATE-07 LST-15]` and
  `[GATE-08]`). Conflicts here should be resolved by **keeping both sides' tests**.
- **`.night-crew/knowledge/roadmap.md`** — the `workflow-offline-double-submit` card status flip,
  in the same change set as the work. Single-card edit, ~line 270, matching how the
  `pwa-cache-and-build-hygiene` card above it was flipped earlier tonight.
- **`.night-crew/runs/2026-07-27-autonomous/merge-intents/b-workflow-offline-double-submit.md`** —
  this note. New file, unique to this card. No conflict surface.
- **`.night-crew/runs/2026-07-27-autonomous/timings.log`** — append-only per-leg timing lines
  prefixed `B `. Conflicts are append-order only; keep both sides.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it
stays clean)_

**One, disclosed rather than left to diverge silently** (ledger T-23 decision 65).

- **`sw.js` — regenerated and COMMITTED.** The pre-implementation text above guessed it would not
  be, on the reasoning that the fix changes no precached asset's *URL*. That was wrong: Workbox's
  precache manifest carries a content **revision hash** per entry, so editing `workflows.html` at
  all changes its hash. `node build-sw.js` (gate G1) produced a **one-line** diff — the
  `workflows.html` revision hash — and CLAUDE.md requires `task sw` after any HTML/JS change, so it
  is committed with the fix. Precache totals are unchanged from Card A's: **22 files / 1457.7 KB**.
  Pure build output; take either side of a conflict and re-run `task sw`. `version.json` is
  git-ignored and did not appear.

Otherwise the footprint held exactly as planned: `workflows.html`, `tests/workflows.spec.js`, the
roadmap flip, this note, and the timings log. **`sync.js` was NOT edited** — the check that
`window.getDB` / `window.idbGetAll` are already exported (`sync.js:100-104`) is what let the card
stay inside `workflows.html`, and it held.

## What must survive any merge

1. **A re-submit for a template that already has a queued submission reuses that submission's
   `idempotency_key`.** This is the whole card. The server's guard is
   `ON CONFLICT (idempotency_key) DO UPDATE … RETURNING id`
   (`backend/internal/workflow/repository.go:722-726`) — an upsert that returns the *existing* row.
   It is already correct and already sufficient; it is simply never reached today because the
   client hands it a new UUID every time. **The fix is to stop minting, not to add a guard.**
2. **The lookup reads IndexedDB, not an in-memory variable.** A module-level
   `LAST_IDEMPOTENCY_KEY[tplId]` would look equivalent and would be wrong: the offline queue is
   durable across reloads and sessions, an in-memory map is not, and "reload the PWA, submit again"
   is a completely ordinary way for a crew member to produce the second submit. The durable store
   is the only thing that makes the reuse survive the reload.
3. **The queue key (`payload.id`) is reused too, not just `idempotency_key`.** `submitQueue` is
   keyed on `id` (`sync.js:52`), so reusing it makes `enqueueSubmission`'s `idbPut` **replace** the
   queued entry with the newer responses instead of appending a second one. Reusing only the
   idempotency key would still end at one DB row (the second POST would 409/`duplicate_submission`
   and `drainQueue` would evict it, `sync.js:571-574`) but would leave the user staring at
   "2 submissions pending sync" for one checklist, and would submit the *older* response set first.
   Both halves are load-bearing for the user-visible behaviour.
4. **The lookup must not be able to break a submit.** If IndexedDB is unavailable or throws, the
   code falls back to minting a fresh key — i.e. it degrades to today's behaviour rather than
   failing the submission. Losing this makes an offline-hostile environment worse than before.
5. **The new tests.** They go red on the pre-fix tree; that RED is captured in its own commit with
   zero production lines changed.

## What is safe to drop

- Comment wording, including the block comment explaining why the lookup reads IDB. The behaviour
  matters; the prose does not.
- Test names and the exact helper-function name for the lookup.
- The `timings.log` lines. They are a record, not a behaviour.
- The roadmap card's prose. The **status flip** matters; the wording does not.
- `sw.js` / `version.json` if they appear at all — regenerate with `task sw`; the content hashes
  are derived, not authored.
- Anything in this note itself.

## Not done, deliberately

- **🛑 No server-side duplicate guard. `backend/internal/workflow` is not opened.** That reopens
  ledger decision 49 and trips this card's park trigger for no added benefit (decision 60, verbatim).
  The existing `ON CONFLICT (idempotency_key)` upsert is left exactly as it is.
- **No change to the `err.offline` branch** (`workflows.html:2781`). Leaving the checklist editable
  after an offline submit is correct, and not pushing into `MY_SUBMISSIONS` is correct. Neither is
  the defect. The defect is only the fresh key.
- **No change to `sync.js`** — not the store schema, not `enqueueSubmission`, not `drainQueue`, not
  the `duplicate_submission` / 409 eviction arms. The card consumes the queue; it does not
  restructure it.
- **No de-duplication of the queue at drain time.** A drain-side "collapse entries by
  `template_id`" would paper over the mint rather than fix it, and would silently discard a
  submission in any future case where two queue entries for one template are legitimate.
- **No `openspec/` directory.** This repo has none, `night-crew workflow preflight` reports ABSENT,
  and B-105 (which per-change discipline hq adopts) is an open operator question. It is not
  answered here by importing another repo's convention. This note lives at the run-local
  `merge-intents/` path the slate specifies.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **UNTOUCHED**, both files, individually
   confirmed. No devDep added, no version moved, no script edited. This card adds no dependency of
   any kind; the fix is ~10 lines of vanilla JS against APIs `sync.js` already exports.
2. **`backend/go.mod`** — **UNTOUCHED**. No Go dependency added, removed, or version-changed. This
   card compiles no new Go; `go build` / `go vet` / `go test` are run as gates only.
3. **`backend/internal/workflow`** — **UNTOUCHED**. This is the card's park trigger, not merely a
   constraint. Files under it were *read* (`repository.go`, `model.go`) to confirm the server's
   upsert already returns the existing row; not one byte is written.
4. **`docker-compose.nc.yml`** — **UNTOUCHED**. This card runs Playwright against its own
   `TEST_PORT=8215` / `TEST_DB_NAME=hq_test_b27`; no compose service is added, renamed, or
   re-ported. The `spike-supabase` stack is left running for Card C.
5. **Root `Taskfile.yml`** — **UNTOUCHED**. No task added, no var default changed. The suite is run
   by invoking `npx playwright test` directly with `CI=1` and explicit env, and the test database
   is dropped/recreated the way the `test:` task does rather than by editing it.
