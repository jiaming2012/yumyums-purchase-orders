# Decisions needed — run `overnight-20260726`

Open items the run declined to decide. The run executes; it does not decide.

---

## Carried from card A's G6 review (not blocking — card A merged APPROVE-WITH-NOTES)

These are G6 findings the reviewer demonstrated but which fall **outside card A's signed scope**.
The run did not fix them, because widening a card's scope at 1am is how a signed slate stops
meaning anything. They are recorded here so the morning reader decides.

### D-1 — `workflows.html:2503` — an eighth status-reading site the slate did not name

**Severity: cosmetic.** The History view renders the raw status token:
`... + escapeHtml(s.status || '')`. After card A, a no-approval submission's history row displays
the literal lowercase string `completed` as user-facing copy. (Before card A it displayed
`pending`, which was worse — so this is an improvement, not a regression.)

No gate reads it and it does not misrepresent state. But the slate named **seven** call sites and
this is an eighth, and the card's own framing — *"teach the client the DB's vocabulary"* —
arguably covers it.

**The decision:** humanize it (a small status→label map) as a follow-up, or accept raw tokens in
the History view. Not decided here.

### D-2 — offline double-tap still writes two submission rows

**Severity: real, pre-existing, untouched by card A.** G6 demonstrated by code read:

`workflows.html:1656` mints a fresh `idempotency_key` per call; `:2778` handles `err.offline` by
returning to the list **without** pushing anything into `MY_SUBMISSIONS`. So: offline → submit →
"Queued for sync" → reopen the checklist (still editable, submit button live — *correctly* so,
since nothing was persisted) → submit again → a second UUID → `enqueueSubmission` writes a second
`submitQueue` entry → `drainQueue` POSTs both → **two rows.** `checklist_submissions` carries only
`idempotency_key UUID UNIQUE` (migration 0011), and there is no server-side duplicate guard —
`grep -rn duplicate_submission backend/` is empty; the only 409 is `template_archived`.

Card A's implementer defended leaving the key per-call, and **that defence is sound for the path it
covers** (a retry of the same enqueued payload — `sync.js:549` persists the key with the payload,
`drainQueue` replays it verbatim). G6 confirmed the reasoning and confirmed it **does not reach
this path**.

Fixing it means either a server-side guard (→ `backend/internal/workflow`, which is card A's PARK
trigger and reopens decision 49) or a client-side queued-submission marker. **Both are scope the
slate did not sign.**

**The decision:** backlog it, or slate it as its own card. Not decided here.

### D-3 — `tests/sync.spec.js:1584` stale comment

**Severity: doc rot.** The comment reads *"requires_approval false → submit yields `'submitted'`"*.
The server yields `'completed'`. It sits inside one of card A's two acceptance specs. One-line fix,
but it is a test file no card tonight owns.

**The decision:** fold into whichever card next touches `sync.spec.js`. Recorded so it is not lost.

---

## Parked cards

_(none yet)_
