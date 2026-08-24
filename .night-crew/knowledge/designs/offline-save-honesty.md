# User stories — an answer that didn't save must never look saved

> **Origin:** operator bug report 2026-07-27, reproduced headlessly the same day as
> `tests/persistence.spec.js` `[OFF-01]` (red in 8s): with the internet off, checking two
> checkboxes turns both green and the list overview reads **2/2**; reopening the checklist
> shows both cleared.
> **Scope:** the *small* fix — stop the UI claiming a save that never happened. Making offline
> answers actually survive is a separate, larger card (§Out of scope).
> **Surface:** `workflows.html` only. No backend change, no `sync.js` change, no schema change.

## The mechanism being fixed

1. `submitOp` (`sync.js:695`) has no offline queue — IndexedDB `hq_offline_v1` holds one store,
   `submitQueue` (`sync.js:51`), and `drainQueue` (`sync.js:560`) drains only that. Field answers
   were never in it.
2. `debouncedSaveField`'s catch (`workflows.html:258-276`) paints a 4px `.save-error-dot` and
   returns. `draftResponses` is never written.
3. The green check lives only in the in-memory `fieldResponses` store (`workflows.html:328`) —
   which is also what **both** progress counters read, via `isFieldAnswered` (`:2471`) →
   `getProgress` (`:2483`) → the runner's "X of N" (`:2502`) and the list row's "a/b" (`:2134`).
4. `hydrateFieldState` (`:1470`) clears `FIELD_RESPONSES` on every open and rebuilds from
   `DRAFT_RESPONSES` + `MY_SUBMISSIONS`. Both empty → both boxes clear.

**Additional finding, 2026-07-27:** the error dot's retry handler reads `row.dataset.lastValue`
(`:285`), and `data-last-value` is **never written anywhere in the codebase**. Tapping the dot
has always been a no-op. There is currently no recovery path at all.

## Deviation from the literal ask, and why

The fix as first proposed was "drop the optimistic state and re-render, so the box visibly
reverts." That is right for a checkbox and **wrong for typed input**: `debouncedSaveField` is the
shared path for text, temperature and fail-note fields too, so a revert would erase text the crew
just typed on any transient failure — trading a false positive for real input destruction.

**Adopted rule instead:** *an answer that has not reached the server is never counted as complete,
and says so on the field.* The value stays on screen; the lie (progress + silent success) is what
gets removed. Same outcome for the reported checkbox case — 0/2, not 2/2 — without destroying
anyone's typing.

---

## US-1 — Progress never counts an answer the server doesn't have

**As** a crew member filling the opening checklist with no signal,
**I want** the checklist list to show the number of answers that are actually recorded,
**so that** I never walk away from a truck believing the checklist is done when nothing was saved.

`done_when:`
- With the network off, checking 2 of 2 checkboxes and tapping back shows **`0/2`** on the list
  row — check: `[OFF-02]` forces offline, checks both, asserts the row text.
- The runner's own counter reads **`0 of 2 items complete`** while those saves are failing —
  check: same test asserts `.progress-line` before the back-tap.
- A field that saves successfully still counts — check: `[FLD-02]` (existing) stays green.

## US-2 — An unsaved answer is visibly marked on the field

**As** a crew member,
**I want** each answer that failed to save to carry a legible mark, not a 4px dot,
**so that** I can see exactly which entries are at risk without decoding punctuation.

`done_when:`
- A field whose save failed renders a **"Not saved"** marker in the field row — check:
  `[OFF-03]` asserts the marker is visible, and reads the literal text, for both failed fields.
- A field that has never been answered carries no marker — check: `[OFF-03]` uses a 3-field
  template, answers 2, and asserts exactly 2 markers.
- The "answered by · time" attribution does not render for an unsaved field — check: `[OFF-03]`
  asserts the row has no `.fill-attribution`, since that line reads as a completed record.

## US-3 — I am told, in words, when a save fails and why

**As** a crew member,
**I want** a message that names the connection when the save failed because I have no signal,
**so that** I know to move the truck or wait rather than assume it went through.

`done_when:`
- An offline save failure surfaces the toast **"Save failed. Check your connection."** — check:
  `[OFF-04]` forces offline and asserts the toast text.
- An online failure surfaces **"Save failed. Please try again."** (existing vocabulary,
  `workflows.html:1257`) — check: `[OFF-04]` routes `POST /ops` to a 500 with the network up and
  asserts the other string.
- Two fields failing in the same burst coalesce into one message rather than flickering the
  banner per field — check: `[OFF-04]` counts `showToast` invocations over the burst (the `#toast`
  node is a singleton, so the DOM alone cannot distinguish one call from two).

## US-4 — The sync happens by itself

**As** a crew member who drives back into signal,
**I want** everything I entered to go up on its own,
**so that** I never have to remember which fields to re-tap.

`done_when:`
- When the connection returns, pending fields sync with **no further input** — the mark clears
  and the answer is on the server — check: `[OFF-05]` goes offline, answers, comes back online,
  and asserts the sync without touching the field again.
- A failure that happens *while online* (5xx) also retries by itself, since no `online` event
  will ever fire for it — check: `[OFF-06]` holds `POST /ops` at 500, then releases it, and
  asserts the field syncs on the retry tick.
- The dead `data-last-value` retry handler is removed rather than left in place — check: `grep`
  for `lastValue` in `workflows.html` returns nothing.

> **Not RxDB.** `workflows.html` loads `sync.js` only — the hand-rolled op-log + WebSocket +
> Lamport clock. The RxDB spikes are DONE (GO) and `vendor/rxdb.bundle.js` is vendored, but
> `sync-rxdb-schema-and-replication` and `sync-hard-cutover` are both PLANNED, so nothing in the
> running app replicates anything. The retry above is this file's own, and it is temporary — see
> the deletion note below.

## US-5 — A recovered save leaves no residue

**As** a crew member,
**I want** the warning state to disappear completely once the answer is recorded,
**so that** a stale "Not saved" mark never makes me redo work that is already saved.

`done_when:`
- On a subsequent successful save of that field, the marker is gone and the field counts toward
  progress — check: `[OFF-05]` asserts both after the retry.
- Reopening the checklist shows no marker for any field — check: `[OFF-05]` asserts zero markers
  after reopen (`hydrateFieldState` clears the unsaved set with the rest of the state).

---

## State table

| State | Trigger | Field shows | Runner counter | List row |
|---|---|---|---|---|
| Saved | POST /ops 200 | value, brief "Saved" | counts it | counts it |
| Saving | debounce in flight | value, "Saving" | counts it | counts it |
| **Unsaved — offline** | POST /ops fails, `navigator.onLine` false | value + **"Not saved"** | excludes it | excludes it |
| **Unsaved — server error** | POST /ops 5xx, online | value + **"Not saved"** | excludes it | excludes it |
| Field cut from template | 422 `unknown_field` | value dropped, field clear | excludes it | excludes it |
| LWW conflict | 409 | winning value rendered | counts winner | counts winner |
| Reopened | `hydrateFieldState` | server truth only | server truth | server truth |

## Deletion note for `sync-hard-cutover`

Everything this design adds is **scaffolding with an expiry date**. When RxDB + Supabase
replication lands, one local store replaces all three of today's mechanisms:

| Today | After cutover |
|---|---|
| `draftResponses` (in-memory, `hydrateFieldState` reads it) | RxDB/Dexie collection — durable, survives reload |
| `PENDING_SYNC` + `PENDING_VALUES` + the retry timer (this design) | RxDB replication's own queue and retry |
| `submitQueue` in IndexedDB (`sync.js:51`) + `drainQueue` | same replication queue |

So `draftResponses` is needed **today** — it is the only reason a pending answer survives
back-and-reopen — and should be **deleted, not ported**, by the cutover. The cutover card owns
removing `PENDING_SYNC`, `PENDING_VALUES`, `schedulePendingRetry`, the `online` listener and the
`.unsaved-mark` CSS along with it (that class was `.pending-sync-mark` when this design was
written; `workflow-queue-period-and-failnote-upsert` renamed it, and its chip now reads
"Unsaved", because sync.js's queued-submission badge read the same two words on the same
screen). The user-facing contract in US-1…US-5 is what must
survive the swap; the mechanism under it should not.

## Out of scope — stated so nobody reads this as shipped

**Answers entered offline do not survive reopening.** This fix makes the loss honest and visible;
it does not prevent it. Preventing it means extending the IndexedDB queue to `SET_FIELD` ops and
writing `draftResponses` optimistically (`hydrateFieldState` already reads drafts, so
reopen-survival falls out) — with Lamport-ts-at-queue-vs-drain and the LWW arm on replay as the
real work. Tracked as its own backlog item; `[OFF-01]` is parked, not deleted, as that card's
red-first spec. Activity 1's RxDB + Supabase cutover may subsume it — sequencing is a planning
call, not this fix's.
