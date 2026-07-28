# UI-SPEC — `sync-rxdb-conflict-notice`

**Status: DRAFT AWAITING OPERATOR SIGN-OFF. Nothing here is approved.**

CLAUDE.md gates UI code on phases introducing new components behind a committed mockup plus an
explicit human *"ok, build this"*. This file and `mockup.html` beside it are that artifact. The
sibling card `sync-rxdb-conflict-notice-ui` stays **ATTENDED-BLOCKED** until the operator answers.
A *no* is a successful outcome for the card that produced this — it is cheaper to redraw a mockup
than to redraw `workflows.html`.

- **Mockup:** [`mockup.html`](mockup.html) — open it in a browser; every state below is rendered.
- **Self-verification renders:** [`screenshots/`](screenshots/) — 480 px, light and dark, one pair
  per table row, produced by `screenshots/shoot.mjs`.

---

## The problem, in one paragraph

A crew member fills a checklist on a phone with no signal in the truck. A manager edits the same
submission from the office. The phone reconnects. **The crew member's work is dropped, and as
configured nothing tells them.** RxDB's default conflict handler is unconditional *master-wins* —
not the last-write-wins the explore session assumed — and it resolves silently: nothing thrown,
`error$` emits zero events, and from inside the app the offline edit simply never happened. For a
product whose stated core value is *accountability — who checked what*, that is a product-level
problem. This spec is the user-visible half of decision 50.

---

## What `conflict$` gives you — and what it does not

**Verified at W2**, not assumed. Sources:
`.night-crew/qa/spike-supabase/rxdb/proof-lww.js`,
`.night-crew/qa/spike-supabase/README.md` half 2 step 5,
`.night-crew/knowledge/designs/sync-rxdb-feasibility-spike.md` §"THE FINDING",
`.night-crew/runs/2026-07-25-autonomous/DECISIONS-NEEDED.md` FORK 3.

The observed run, verbatim, twice with identical results:

```
local  body after reconnect : REMOTE-EDIT (written first, T1)
remote body after reconnect : REMOTE-EDIT (written first, T1)
replication errors surfaced : 0 []
conflict handler invocations: 1
    newDocumentState.body  : LOCAL-EDIT (written second, T2)    <- the local (later) write
    realMasterState.body   : REMOTE-EDIT (written first, T1)    <- what the server actually held
    handler CHOSE          : REMOTE-EDIT (written first, T1)
```

### It DOES give you

| | |
|---|---|
| **One event per conflicting document** | `RxReplicationState.conflict$` (`rxdb/dist/esm/plugins/replication/index.js:44,51,287-289`). In the scenario above: `error$` **0** events, `conflict$` **1** event. |
| **The discarded local document, whole** | `e.input.newDocumentState` — every field as the crew member left it, **including the document id (the PK)**. This is the value the sheet recovers. |
| **The server document that won** | `e.input.realMasterState`, and `e.output` — the resolution the handler returned. Under the default handler these are the same thing. |
| **The state the push assumed** | `e.input.assumedMasterState` — what the client believed the server held when it pushed. |

### It does NOT give you — and each gap shapes a state below

| Gap | Consequence for this UI |
|---|---|
| **Which field clashed.** Whole documents only; no field name, no path. | The app must diff `newDocumentState` against `output` itself. A diff can come back with nothing a crew member would recognise → the **no discarded value available** row. |
| **Any replay.** `conflict$` is a plain RxJS `Subject` — not `ReplaySubject`, not `BehaviorSubject`. Subscribe where the replication is constructed or the event is gone. RxDB persists nothing about a resolved conflict. | **The app must write the discarded value to durable local storage the instant the event arrives**, or a reload loses the thing this screen exists to recover. Every state below assumes that record exists. It is also why **error** and **local conflict log unreadable** are distinct states: one is a failed write, the other is a lost record. |
| **Any event outside the leader tab.** `waitForLeadership` defaults to `true` in a browser (W3 confirmed election works in tabs, ~50 ms handover); only the leader replicates, so only its `conflict$` fires. | The banner must read from the shared local record, never from a live subscription. If nothing was subscribed at all, **the app never learns and no notice appears** — unfixable at the UI layer; stated in the mockup's limits panel so the sign-off is given with it in view. |
| **Any author or timestamp of its own.** No `who`, no `when`, no user-facing text, no severity. | "Dana M., 6:12 PM" is only as real as the replicated row. If the schema does not carry who-and-when, those lines degrade to "someone else". **That schema belongs to `sync-rxdb-schema-and-replication`; this spec states a requirement on it and does not decide it.** |
| **Per-collection scoping.** It fires per `replicateSupabase()` call, once per document. | One subscription per replication; the sheet is fed by their union. |

### One thing the operator should know is not free

Declaring `_modified` in the collection schema is a **semantics switch, not a formality** (W2 sharp
edge 11): it makes `addDocEqualityToQuery` include `_modified` in the compare-and-swap, so **any**
server-side touch becomes a conflict — including ones where no answer changed. That is a real
generator of the **no discarded value available** state. Whether to declare it is the parent card's
call; if it is declared, that row stops being rare.

---

## Component inventory

| Component | Where | Purpose |
|---|---|---|
| **Conflict banner** | top of My Checklists, `workflows.html` | The only thing that appears unprompted. Amber, exact count, one tap to the sheet. |
| **Overwritten-answers sheet** | full-height, over My Checklists | The whole feature. Reads the durable local conflict record. |
| **Conflict group** | one per document = one `conflict$` event | Checklist name, date, document id chip, count chip. Grouping is not cosmetic — it is the shape the event arrives in. |
| **Conflict field row** | one per differing field | *Yours* / *Now shows* value pair + actions. Six row renderings: default, in-flight, restored, failed, unrecoverable, removed-field. |
| **Restore mine** | primary action on a row | Writes the crew member's value again **now**, from the current master state. An ordinary local edit that pushes cleanly — it resurrects nothing and needs no new sync plumbing. |
| **Restore all N of mine** | foot of a group | Same, batched, for a group whose per-row buttons have collapsed. |
| **Copy value** | removed-field rows only | The only recovery available when there is nowhere to write the value back. |

---

## State Enumeration Table

Four base states plus four edge rows. The three edge rows the card names by hand are marked ★.
Each row maps 1:1 to a plate in `mockup.html` and to a screenshot pair in `screenshots/`.

| State | Trigger | Visual contract |
|-------|---------|-----------------|
| **empty** | Sheet opened from the sync menu; zero conflict records held locally. | No banner anywhere in the app. Sheet shows "Nothing was overwritten" plus one line saying what would put something here and how long records are kept. Copy must **not** read as a guarantee that nothing was ever lost — the app cannot know that (limits panel, item 1). |
| **loading** | Sheet opened; the local conflict log has not resolved within 500 ms (cold IndexedDB read). | Two skeleton group cards. No spinner. **No count in the header** — the header must not claim a number it does not have. Nothing renders at all under 500 ms. |
| **error** | **Restore mine** tapped and the write does not land: offline, or the server value moved again and the restore conflicted in turn. | The row **stays** and keeps showing both values. Red inline block names which of the two happened in the crew member's words ("Couldn't put 38 °F back — you're offline"), states that nothing was lost, and offers **Retry**. The banner count does **not** decrement. A failed restore must never look like a completed one. In-flight: both buttons disabled, label reads "Restoring…". |
| **success** | Replication reconnects; `conflict$` emits for a document; the diff names ≥1 template-backed field carrying a value in `input.newDocumentState`. | Amber banner above the checklist list with an **exact** count and a plain-language cause. Sheet lists the group; each row shows the pair labelled *Yours* (amber, the crew member's) / *Now shows* (with attribution when the row carries it). Every recoverable row offers **Restore mine** + **Keep theirs**, both ≥44 px. A restored row collapses to a green confirmation that **names the value that came back** and offers **Undo**. |
| ★ **edge: no discarded value available** | `conflict$` fired, but the diff between `input.newDocumentState` and `output` yields nothing showable — the only difference is bookkeeping (see `_modified` above), or the field is absent/null in the discarded document because the local write cleared it. | The row must **not** claim an answer was lost and must **not** render an empty slot where a value goes. Title reads "A change we couldn't identify"; the value slot reads *Not recoverable* in muted italic; one line explains in plain words. **No Restore button** — there is nothing to put back. Actions are **Open checklist** and **Dismiss**. The banner counts these on a separate, quieter line ("+ 1 change we couldn't identify") so the headline count stays literally true. |
| ★ **edge: several conflicts at once** | A long offline stretch ends; `conflict$` emits once per document, several times in a burst. | **One** banner, never one per conflict, carrying the total and the number of checklists. Sheet groups rows under their document. Past two rows in a group the per-row buttons collapse to values-only and a single **Restore all N of mine** sits at the foot of the group; a caption says the buttons come back on tap. Every group header carries a count chip so the size of the problem is legible without opening anything. |
| ★ **edge: conflict on a field since removed from the template** | The discarded document carries a value for a field id the current template no longer contains — the owner edited the template while the phone was offline. | **No invented label.** The raw field id renders in monospace and muted, so it is visibly not a question title. One line states the question was removed. The value is still shown — it is the thing being recovered. **Restore mine is absent** (nowhere to write it); the recovery is **Copy value**, which is a real recovery on a phone: it goes into a message to the manager in two taps. |
| **edge: local conflict log unreadable** | Sheet opens and the local store holding the conflict records cannot be read — iOS/Safari eviction under storage pressure, or private browsing. W3 named this the largest untested unknown for a phone-first PWA, so it gets a designed state rather than a stack trace. | Red-bordered card. **No fabricated count.** Copy says *the records* are gone and explicitly that the checklists themselves are fine — a crew member must not read a storage failure as "my work was deleted". Single **Try again** action. |

---

## The recovery path

**Restore mine** is the answer to "how do they get the value back", and it is deliberately boring.

Master-wins discarded the fork because `replicateSupabase`'s push issues a compare-and-swap
(`UPDATE … WHERE id = … AND <field> = <assumed master value>`) and the server value had already
moved, so zero rows matched. Nothing about that state is unrecoverable — it is simply *stale*.
Tapping Restore writes the crew member's value **again, now, from the current master state**. That
is an ordinary local edit. It matches the current CAS, it pushes cleanly, and it needs no new sync
plumbing, no handler special case, and no server endpoint. If the server moves again in between,
that write conflicts in turn and lands back in this same sheet — the loop is closed, not open.

Three degradations, in order of how much is left:

1. **Value present, field still on the template** → *Restore mine*, one tap. (success)
2. **Value present, field gone from the template** → nowhere to write it, so *Copy value* to the
   clipboard. (removed-field edge)
3. **No value to show** → nothing to recover; say so and offer *Open checklist* so the crew member
   can verify what it reads now. (no-discarded-value edge)

**The precondition, stated once and loudly:** because `conflict$` has no replay and RxDB persists
nothing about a resolved conflict, all three depend on the app writing the discarded value to
durable local storage **the instant the event arrives**. Without that, a reload destroys the value
before anyone can act on it. *Where* that record lives — a local-only RxDB collection, or something
simpler — is the UI card's implementation call; that it must exist is a contract of this design.

---

## `done_when:`

```yaml
done_when:
  # ── the artifact this card owes ────────────────────────────────────────────
  - "mockup.html exists at .planning/phases/sync-rxdb-conflict-notice/mockup.html — the exact path CLAUDE.md's sign-off gate reads; ls the path"
  - "Every State Enumeration row has a plate in mockup.html carrying its trigger and visual contract as a visible caption — count plates against table rows, 1:1"
  - "All three card-named edge rows are present and marked: no discarded value available, several conflicts at once, conflict on a field since removed from the template — grep the table for the three ★ rows"

  # ── the design contract, checkable against the renders ─────────────────────
  - "The recovery action is visible on every row that has a value to recover — screenshots/success-*.png and edge-removed-*.png both show a primary action; screenshots/edge-novalue-*.png shows no Restore button"
  - "A restored row names the value that came back, not just 'restored' — read success-light.png, assert the literal value string appears in the green confirmation"
  - "A failed restore keeps both values on screen and does not decrement the count — read error-light.png, assert Yours and Now shows are both still rendered above the red block"
  - "The removed-field row renders the raw field id in monospace and offers Copy value with no Restore — read edge-removed-light.png"
  - "The no-value row shows 'Not recoverable' and no Restore button — read edge-novalue-light.png"
  - "Several-at-once renders ONE banner carrying the total across all groups, not one banner per conflict — read edge-many-light.png, count banners"
  - "The loading state shows no count in the sheet header — read loading-light.png, assert the header is the bare title"
  - "The storage-error copy says the records are gone and the checklists are fine — read edge-storage-light.png"

  # ── house constraints ──────────────────────────────────────────────────────
  - "Renders at 480px with no horizontal overflow in either scheme — Playwright viewport 480, per-plate element screenshot, read all 18 PNGs"
  - "Both colour schemes render legibly from the shared CSS variable block — screenshots/*-light.png and *-dark.png pairs exist for all 9 plates and were read back"
  - "Touch targets on row actions are >=44px — .cf-btn and .cg-all both declare min-height:44px; grep the stylesheet"
  - "Zero production files changed — git diff --name-only d73580d..HEAD lists nothing outside .planning/phases/sync-rxdb-conflict-notice/ and .night-crew/"
  - "No version constant moved — version.go Backend 0.3.0 and package.json 1.2.1 both byte-identical to the branch point; git diff on both paths is empty"
```

---

## Explicitly NOT decided here

These belong to `sync-rxdb-schema-and-replication` and are named so the sign-off does not
accidentally cover them:

- **The `conflictHandler`'s actual merge rule.** This spec assumes a same-field clash falls back to
  master-wins, which is what the card says. A field-level three-way merge changes *how often* these
  states fire, not *what they look like*.
- **The replicated schema**, including whether rows carry who-and-when (which is what makes the
  attribution line real or fictional) and whether `_modified` is declared.
- **Where the durable conflict record lives** and how long it is kept. The mockup's empty state says
  30 days; that number is a placeholder for the operator to accept or change.
- **Any RLS predicate.** Row visibility is obligation 1 of the parent card.

## Open question for the operator

Beyond roughly ten conflict groups the sheet needs a cap or a date filter; it is not designed here.
Say so if you think a truck can realistically get there — one long dead-zone shift with an active
manager is the scenario to judge it against.
