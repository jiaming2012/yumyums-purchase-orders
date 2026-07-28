# UI-SPEC — `sync-rxdb-conflict-notice`

**Status: DRAFT AWAITING OPERATOR SIGN-OFF. Nothing here is approved.**

CLAUDE.md gates UI code on phases introducing new components behind a committed mockup plus an
explicit human *"ok, build this"*. This file and `mockup.html` beside it are that artifact. The
sibling card `sync-rxdb-conflict-notice-ui` stays **ATTENDED-BLOCKED** until the operator answers.
A *no* is a successful outcome for the card that produced this — it is cheaper to redraw a mockup
than to redraw `workflows.html`.

- **Mockup:** [`mockup.html`](mockup.html) — open it in a browser; every state below is rendered.
- **Self-verification renders:** [`screenshots/`](screenshots/) — 480 px, light and dark, one pair
  per table row, produced by `screenshots/shoot.mjs`. That script also **measures** the two
  contracts that cannot be judged by eye — horizontal overflow and touch-target size — and exits
  non-zero if either fails, so those `done_when:` rows are checked rather than asserted.

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
| **Conflict field row** | one per differing field | *Yours* / *Now shows* value pair + actions. Eight row renderings: default, in-flight, restored, **kept-theirs**, **undone**, failed, unrecoverable, removed-field. |
| **Restore mine** | primary action on a row | Writes the crew member's value again **now**, from the current master state. An ordinary local edit that pushes cleanly — it resurrects nothing and needs no new sync plumbing. |
| **Undo** | on a restored or kept-theirs row | The only escape from a mis-tap, used on a truck in daylight with wet hands. A bordered **≥44 px** control, never an inline text link. Its existence is why handled rows are kept rather than removed. |
| **Restore all N of mine** | foot of a group | Same, batched, for a group whose per-row buttons have collapsed. **Styled primary** — on a collapsed group it is the only action on the card. |
| **Copy value** | removed-field rows only | The only recovery available when there is nowhere to write the value back. |

---

## The counting rule

Stated once, here, because three surfaces show a number and they must agree. **Every plate in
`mockup.html` obeys this and the plates are drawn so it can be checked from a screenshot.**

1. **The banner headline** counts **recoverable answers** across the whole sheet.
2. **A group's count chip** counts the recoverable answers **in that group**, and equals the number
   of field rows drawn in that group.
3. **What has been done to a row never changes any count.** A restored row, a kept-theirs row, a
   failed restore and an in-flight restore are all still counted, because the record still exists
   and the crew member can still act on it (Undo, Retry). A count drops only when a record **leaves**
   the sheet — an explicit **Dismiss**, or ageing out of the retention window.
4. **Unidentifiable changes are counted separately**, never folded into the headline: a second
   banner line (`+ 1 change we couldn't identify`) and a `+N` appended to that group's chip
   (`1 answer +1`). The headline number therefore stays literally true as a number of *answers*.
   Chip base + chip `+N` = rows drawn.
5. **A repeat conflict on the same question does not add a row.** The durable conflict record is
   keyed by **document id + field id**, so a restore that conflicts in turn *replaces* the record it
   already has and updates the existing row in place. This is what closes the loop described under
   *The recovery path* below without the sheet growing a duplicate row each time, and it is why the
   count is stable under Retry.

Consequence, stated plainly so the operator can reject it if they disagree: **the banner is not a
to-do list.** It reports how many answers were overwritten in the window, not how many are still
unhandled. The alternative — decrementing on Restore/Keep — makes the sheet a queue that a mis-tap
empties, and it destroys Undo, because a row that has been removed cannot be undone.

---

## State Enumeration Table

Four base states plus six edge rows. The three edge rows the card names by hand are marked ★.
Each row maps 1:1 to a plate in `mockup.html` and to a screenshot pair in `screenshots/`.

| State | Trigger | Visual contract |
|-------|---------|-----------------|
| **empty** | Sheet opened from the sync menu; zero conflict records held locally. | No banner anywhere in the app. Sheet says what would put something here and how long records are kept. Because a non-leader tab, a replication with nothing subscribed to `conflict$`, and an evicted local store **all produce this identical screen**, the copy must be **scoped to the record, not phrased as a guarantee** — headline scoped to the retention window ("Nothing recorded in the last 30 days"), body scoped to what the app caught ("This is what HQ caught and kept — if it wasn't running when a change came in, that change won't be listed"). A flat "Nothing was overwritten" is **banned**: it is a claim the app is not in a position to make, on the screen it shows most often. |
| **loading** | Sheet opened; the local conflict log has not resolved within 500 ms (cold IndexedDB read). | Two skeleton group cards. No spinner. **No count in the header** — the header must not claim a number it does not have. Nothing renders at all under 500 ms. |
| **error** | **Restore mine** tapped and the write does not land: offline, or the server value moved again and the restore conflicted in turn. | The row **stays** and keeps showing both values. Red inline block names which of the two happened in the crew member's words ("Couldn't put 38 °F back — you're offline"), says **where the discarded value still is** — *on this list*, not in the checklist, which still reads the server's value — and offers **Retry**. It must **not** promise an automatic retry: nothing in this design commits to one, and the only retry drawn is the crew member's. **The count does not decrement**, and the plate must make that checkable rather than assert it: the banner is rendered, and **banner count = group chip = rows drawn**, with a failed row and an in-flight row among them. A failed restore must never look like a completed one. In-flight: both buttons disabled, label reads "Restoring…". |
| **success** | Replication reconnects; `conflict$` emits for a document; the diff names ≥1 template-backed field carrying a value in `input.newDocumentState`. | Amber banner above the checklist list with an **exact** count and a plain-language cause. Sheet lists the group; each row shows the pair labelled *Yours* (amber, the crew member's) / *Now shows* (with attribution when the row carries it). Every recoverable row offers **Restore mine** + **Keep theirs**. A restored row collapses to a green confirmation that **names the value that came back** and offers **Undo** — a bordered ≥44 px control, never an inline text link, because it is the only escape from a mis-tapped Restore. |
| **edge: row already handled** | **Keep theirs** tapped on a row, or **Undo** tapped on a row that had been restored — the two outcomes one tap off the primary path. | Neither outcome removes the row and **neither changes any count** (counting rule 3): banner = chip = rows drawn, unchanged. A kept-theirs row collapses to a **muted, non-alarming** confirmation naming **the value that is now standing** ("Kept theirs — the checklist reads 41 °F") and keeps an **Undo**. An undone row returns to the full two-value, two-button row with one muted line saying what happened, so the tap is not silent. **Dismiss** is deliberately not drawn: it removes record, row and count together and yields either a smaller sheet or the **empty** state — there is no third rendering. |
| ★ **edge: no discarded value available** | `conflict$` fired, but the diff between `input.newDocumentState` and `output` yields nothing showable — the only difference is bookkeeping (see `_modified` above), or the field is absent/null in the discarded document because the local write cleared it. | The row must **not** claim an answer was lost and must **not** render an empty slot where a value goes. Title reads "A change we couldn't identify"; the value slot reads *Not recoverable* in muted italic; one line explains in plain words. **No Restore button** — there is nothing to put back. Actions are **Open checklist** and **Dismiss** — two, not one. The contract is **per row**: a plate may (and the mockup's does) carry an ordinary recoverable row alongside, which *does* have Restore. Counting per rule 4: banner headline carries the recoverable answers, a quieter second banner line carries these ("+ 1 change we couldn't identify"), and the group chip appends `+N` (`1 answer +1`) so chip base + `+N` = rows drawn. |
| ★ **edge: several conflicts at once** | A long offline stretch ends; `conflict$` emits once per document, several times in a burst. | **One** banner, never one per conflict, carrying the total and the number of checklists. Sheet groups rows under their document. Every group header carries a count chip so the size of the problem is legible without opening anything. **Collapse rule:** per-row buttons collapse to values-only, with a single **Restore all N of mine** at the foot of each group, when the sheet holds **more than one group** *or* any one group holds **more than two rows** — and when it applies it applies **to the whole sheet**, so a 2-row group beside a 3-row one does not render in a different style (a mixed sheet reads as a bug, not as a rule). A caption says the buttons come back on tap. **Restore all N of mine is styled primary**, not secondary: on a collapsed group it is the only action on the card and must not be its least prominent element. |
| **edge: long value / long question text** | The discarded answer is a **free-text checklist note** — the field type most likely to survive a conflict and the only one with no length bound — or it contains an unbroken token with no spaces to wrap at (a pasted reference, an id, a URL), under a question title long enough to wrap. | The value **wraps inside the card** and is **never truncated**: it is the thing being recovered, so an ellipsis would hide the payload the crew member came for. The 88 px label column keeps its width; attribution may drop to its own line. **The page does not scroll sideways at 480 px** — `document.scrollWidth === document.clientWidth`, measured in the browser by `shoot.mjs`. Note for the implementer: the fixed label column inside a `display:flex` row makes this a real failure mode, not a hypothetical — a flex item's default `min-width:auto` will not shrink below its longest unbreakable word, so `min-width:0` on the row **and** the value plus `overflow-wrap:anywhere` on the value are all three required. |
| ★ **edge: conflict on a field since removed from the template** | The discarded document carries a value for a field id the current template no longer contains — the owner edited the template while the phone was offline. | **No invented label.** The raw field id renders in monospace and muted, so it is visibly not a question title. One line states the question was removed. The value is still shown — it is the thing being recovered. **Restore mine is absent** (nowhere to write it); the recovery is **Copy value**, which is a real recovery on a phone: it goes into a message to the manager in two taps. |
| **edge: local conflict log unreadable** | Sheet opens and the local store holding the conflict records cannot be read — iOS/Safari eviction under storage pressure, or private browsing. W3 named this the largest untested unknown for a phone-first PWA, so it gets a designed state rather than a stack trace. | Red-bordered card. **No fabricated count.** Single **Try again** action, offered without being oversold. The copy must carry **both halves, bad one first**: (a) that if Try again does not bring the list back the record is **permanently gone** and the overwritten answers **cannot be put back** — this is the one screen in the set where something really is unrecoverable, and an evicted IndexedDB is not recovered by tapping a button; (b) that the **checklists themselves are unaffected** and on the server. Half (a) is set at full text contrast, not muted. A crew member must not read a storage failure as "my work was deleted" — and must not read it as "nothing was lost" either. |

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
**And it lands on the same row, not a new one:** the durable conflict record is keyed by
**document id + field id**, so a repeat clash on the same question replaces the record already
held. Without that key the closed loop would grow a duplicate row per retry and the counting rule
above would not hold. This is a requirement on the UI card's storage choice, not a free property.

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
  - "1. mockup.html exists at .planning/phases/sync-rxdb-conflict-notice/mockup.html — the exact path CLAUDE.md's sign-off gate reads; ls the path"
  - "2. Every State Enumeration row has a plate in mockup.html carrying its trigger and visual contract as a visible caption — count plates against table rows, 1:1 (10 state rows + the limits panel = 11 plates)"
  - "3. All three card-named edge rows are present and marked: no discarded value available, several conflicts at once, conflict on a field since removed from the template — grep the table for the three ★ rows"

  # ── the design contract, checkable against the renders ─────────────────────
  - "4. The recovery action is visible on every ROW that has a value to recover, and absent from every row that has none — assert PER ROW, not per plate: in edge-novalue-light.png the .cf.unrec row has only Open checklist + Dismiss while the ordinary row above it does have Restore mine; success-light.png and edge-removed-light.png each show a primary recovery on their valued rows"
  - "5. A restored row names the value that came back, not just 'restored' — read success-light.png, assert the literal value string appears in the green confirmation"
  - "6. A failed restore keeps both values on screen AND is still counted — read error-light.png and assert all four: Yours and Now shows are both still rendered above the red block; a banner IS drawn; the banner headline, the group count chip and the number of field rows are the same number; the failed row is one of the rows counted. (The old wording asked for 'does not decrement', which a single static plate cannot show. Equality of the three numbers on a plate whose rows have already failed/started is what proves the rule.)"
  - "7. The removed-field row renders the raw field id in monospace and offers Copy value with no Restore — read edge-removed-light.png"
  - "8. The no-value row shows 'Not recoverable' and no Restore button — read edge-novalue-light.png"
  - "9. Several-at-once renders ONE banner carrying the total across all groups, not one banner per conflict — read edge-many-light.png, count banners"
  - "10. The loading state shows no count in the sheet header — read loading-light.png, assert the header is the bare title"
  - "11. The storage-error copy carries BOTH halves — read edge-storage-light.png and assert it states (a) that the record may be permanently gone and the overwritten answers cannot be put back, and (b) that the checklists themselves are fine. Neither half alone passes; half (a) must not be muted-only text"

  # ── rules the verifier found undefined or contradicted ─────────────────────
  - "12. The counting rule is stated in this file AND obeyed by every plate — read all 11 light PNGs; on each plate that draws a group, chip base + any '+N' == field rows drawn in that group; on each plate that draws a banner, headline == sum of recoverable rows and the second line == sum of unidentifiable rows. Any plate that disagrees fails this row"
  - "13. Every plate caption agrees with what that plate draws — read each PNG's caption against its own render; specifically edge-novalue names BOTH Open checklist and Dismiss, and edge-many's collapse rule accounts for its own 2-row collapsed group"
  - "14. The two outcomes one tap off the primary path are drawn, not just described — read outcomes-light.png: a kept-theirs row naming the standing value with an Undo, and an undone row back to two values + two buttons, with the count unchanged across both"
  - "15. Long content does not break the layout — read edge-longvalue-light.png: a multi-line free-text note and an unbroken token both wrap inside the card with no truncation and no ellipsis. Machine-checked: shoot.mjs asserts document.scrollWidth === document.clientWidth at viewport 480 in both schemes and exits non-zero otherwise"

  # ── house constraints ──────────────────────────────────────────────────────
  - "16. Renders at 480px with no horizontal overflow in either scheme — measured by shoot.mjs (scrollWidth vs clientWidth, both schemes), not judged by eye; the script exits non-zero on failure"
  - "17. Both colour schemes render legibly from the shared CSS variable block — screenshots/*-light.png and *-dark.png pairs exist for all 11 plates (22 PNGs) and were read back"
  - "18. EVERY interactive element in the design is a >=44px touch target in both dimensions — not just the two classes already known to pass. shoot.mjs measures getBoundingClientRect over '.cf-btn, .cg-all, .cf-done-undo, .sc-close, .cn-banner-go, .sc-err button, .sc-empty button' in both schemes, prints the count measured and every element under 44px, and exits non-zero if the list is non-empty. Undo, Done and the banner's Review are inside that selector. Grepping the stylesheet does NOT satisfy this row — it was what let three sub-44px controls through"
  - "19. Zero production files changed — git diff --name-only d73580d..HEAD lists nothing outside .planning/phases/sync-rxdb-conflict-notice/ and .night-crew/"
  - "20. No version constant moved — version.go Backend 0.3.0 and package.json 1.2.1 both byte-identical to the branch point; git diff on both paths is empty"
```

### Why two criteria were rewritten rather than re-run

Criteria **6** and **18** are the repair round's own findings about this block. Both were written so
they could not fail:

- **6** asked a *plate* to demonstrate a *transition* ("does not decrement"). A static render has no
  before-and-after, so the second half was unfalsifiable — and the plate as first drawn showed a
  "1 answer" chip above two rows, which is what a decrement looks like. It now asks for an equality
  between three numbers on one plate, which a single screenshot can settle.
- **18** named the two classes it already knew declared `min-height:44px` and checked them by
  grepping CSS. That excluded Undo (35×16), Done (37×16) and Review (53×15) — including the only
  escape from a mis-tapped Restore. It now enumerates every interactive selector and measures the
  rendered box, in the browser, in both schemes.

Criterion **4** was scoped to the wrong unit: it checked a *plate* for the absence of a button that
the plate deliberately contains on a different, recoverable row. The unit is the row.

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
