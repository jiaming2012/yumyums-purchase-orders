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

---

## Card C — one UNATTRIBUTED red in the full-suite gate (NOT a park; card C landed GO)

**This is a record, not a fork.** Card C is not parked and needs no decision to
merge. It is filed here because the run's discipline says an unattributed red
belongs in the durable record rather than only in a card report.

**The red:** `tests/workflows.spec.js:2466` — *Loading states › unsubmit returns
checklist to editable draft [RUN-10]*. Failed **both** attempts in card C's full
549-test gate (`1 failed / 2 flaky / 6 skipped / 540 passed`, 47.0 m,
`TEST_PORT=8299`, `hq_test_c1`).

- attempt 1: `page.click('[data-action="unsubmit"]')` timed out — *"element was
  detached from the DOM, retrying"* (a re-render race).
- retry #1: after unsubmit, `[data-action="submit"]` never reappeared.

### 🛑 I REFUSE TO ATTRIBUTE IT, and here is the measurement rather than a guess

| Condition | Result |
|---|---|
| RUN-10 alone, `--repeat-each=3 --retries=0`, on card C's HEAD | **3 / 3 passed** |
| Full `tests/workflows.spec.js` (80 tests), `--retries=0`, on card C's HEAD | **80 / 80 passed**, RUN-10 green as test #57 |
| Full 549-test suite on card C's HEAD | **failed twice** |

So it is **not** deterministic, **not** within-file order sensitivity, and it did
**not** reproduce in 83 attempts outside the whole-suite condition.

**Why it is mechanically not card C's**, and this is provable rather than
asserted:

```
git diff --stat overnight-20260726..HEAD -- backend tests features lib \
  ':!.night-crew' workflows.html sync.js ptr.js index.html   # prints NOTHING
```

RUN-10 exercises `workflows.html` and the Go backend. **Every byte the browser
and the server see for this test is identical to `overnight-20260726`.** Card C's
entire diff is `vendor/**` (new, never imported by any HQ page), `build-sw.js`
(build-time only), `sw.js` (**never registered — `playwright.config.js:60` sets
`serviceWorkers: 'block'`**), and `.night-crew/**`.

**Why I still will not call it "pre-existing flake" as a fact.** Two correlations
exist and guessing either way is equally wrong:

1. **Card B ran its own full Playwright suite CONCURRENTLY** for the first half
   of card C's gate. Measured 1-min load: **35.48 at start, peaking above 61,
   settling to 8–17 once card B finished.** A green here bounds a *loaded*
   condition; so does this red. **Concurrency makes attribution harder, not
   easier**, and that is the honest statement.
2. **Card A's just-merged `workflow-submission-status-client-half` is topically
   adjacent** — RUN-10 is precisely an unsubmit/status re-render assertion, and
   card A's change is already in card C's base. **Flagged as a correlation. Not
   asserted as a cause.** Card C did not test at base and therefore cannot say.

**What would settle it:** run RUN-10 inside the full 549-test suite at
`overnight-20260726` (i.e. with card A merged, card C absent) on a quiet box. If
it reds there too, it belongs to card A or to the suite; if it stays green, the
question reopens. **That is one command and it was outside card C's remit.**

### The two flaky (passed on retry #1) — also not attributed

- `tests/sync.spec.js:836` — *sub-step checks on Device A appear checked on
  Device B [SYN-03]*
- `tests/sync.spec.js:1327` — *checkbox answer converges (live + catch-up)*

Both in `sync.spec.js`, the file carrying the **proven ~16–20 % flake** whose
exposure card A's merged seam fix deliberately raises. **Neither is the
specifically flagged `sync.spec.js:1198`** — that one passed. `purchasing.spec.js:1407`
(FR-13) also passed. Same refusal to attribute applies, same reason.
