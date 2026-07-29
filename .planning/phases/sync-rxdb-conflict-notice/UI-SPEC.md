# UI-SPEC — `sync-rxdb-conflict-notice`

**Status: REVISION 2 — DRAFT AWAITING OPERATOR SIGN-OFF. Nothing here is approved.**

CLAUDE.md gates UI code on phases introducing new components behind a committed mockup plus an
explicit human *"ok, build this"*. This file and `mockup.html` beside it are that artifact. The
sibling card `sync-rxdb-conflict-notice-ui` stays **ATTENDED-BLOCKED** until the operator answers.
A *no* is a successful outcome for the card that produced this — it is cheaper to redraw a mockup
than to redraw `workflows.html`.

**Revision history, because the sign-off state is not a simple yes or no:**

| Rev | Card | What changed |
|---|---|---|
| r1 | `sync-rxdb-conflict-notice-mockup` (run 2026-07-29) | First draft. 11 plates, 20 `done_when:` rows. Signed *"Ok, build this"* at 18:12 — **ledger decision 80**. |
| r2 | `sync-rxdb-conflict-notice-mockup-amendments` (run 2026-07-29-2) | **Decision 80 superseded in part at morning triage 2026-07-28 — ledger T-26 decision 82.** Amendments **A-1** and **A-2** below are drawn. 16 plates. The card returned to ATTENDED-BLOCKED and **this revision does not discharge it** — it produces the artifact the operator signs. |

- **Mockup:** [`mockup.html`](mockup.html) — open it in a browser; every state below is rendered.
- **Self-verification renders:** [`screenshots/`](screenshots/) — 480 px, light and dark, one pair
  per plate (16 pairs, 32 PNGs), produced by `screenshots/shoot.mjs`. That script also
  **measures** the five contracts that cannot be judged by eye — horizontal overflow, touch-target
  size, **that every banner carries both A-1 figures**, **that no banner line truncates at 480 px**
  (the card's PARK trigger) and **that every Restore control names what it replaces** — and exits
  non-zero if any fails, so those `done_when:` rows are checked rather than asserted. The three new
  measurements were **run red against the un-amended r1 mockup first** (5 banners carrying one
  figure, 7 Restore controls silent about the loss) and each was **mutation-tested** to prove it can
  fail rather than passing vacuously.

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
| **Any author or timestamp of its own.** No `who`, no `when`, no user-facing text, no severity. | "Dana M., 6:12 PM" is only as real as the replicated row. **r1 said those lines "degrade to someone else"; A-2 removes that escape** — the batch confirm exists to say *exactly* what is being overridden, so who-and-when is a **required output** of `sync-rxdb-schema-and-replication`, not an option it may decline. If it is declined, the confirm plate cannot be built as drawn. **That schema still belongs to the parent card; this spec states the requirement and does not decide it.** |
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
| **Restore all N of mine** | foot of a group | Same, batched, for a group whose per-row buttons have collapsed. **Styled primary** — on a collapsed group it is the only action on the card. **A-2:** it names what it replaces, carries the same attribution + timestamp the rows do, acts only on rows **still to review** (counting rule 7), and **opens a confirm rather than writing through**. |
| **Batch override confirm** | over the sheet, on `Restore all N of mine` | A-2's protection. Names the loss in its title, **lists the N server values about to be overwritten** with author and time, and offers Cancel at equal weight. Not used for the single-row restore — Undo is the net there. |
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

6. **The banner carries a second figure: rows still to review.** (Added by A-1.) A row is
   **reviewed** once it has been **restored, kept, or dismissed**. An **untouched** row is still to
   review. **A failed restore is still to review** — the crew member has unfinished business there.
   An **in-flight** restore has not landed, so it is too. **Unidentifiable changes are in neither
   this figure nor the headline** — they stay in the separately-counted `+N` (rule 4), because
   folding them in here would re-mix exactly what rule 4 keeps apart. Wording by case: nothing
   handled → *"N still to review"*; some handled → *"N still to review · M handled"*; all handled
   → *"All N reviewed"*.
7. **`Restore all N of mine` acts only on rows still to review.** (Added by A-2.) Its N is rule 6's
   figure for that group, **not** the chip base. A batch tap must never re-write a row the crew
   member deliberately kept — that would silently reverse their own decision, which is the opposite
   of what an override that "states what it destroys" is for.
8. **Collapse hides the Restore/Keep pair; it does not hide a row's outcome strip or its Undo.**
   (Added by A-2.) Collapsing exists to cut thumb-scrolling and an outcome strip is one line that
   *reduces* what a row costs. More to the point: after `Restore all N`, all N rows are green on a
   collapsed sheet, and if collapse ate Undo a batched mis-tap would be irreversible.

Consequence, stated plainly so the operator can reject it if they disagree: **the headline is not a
to-do list, and it is no longer alone.** Rule 1 reports how many answers were overwritten in the
window; rule 6 reports how many are still unhandled. Decrementing rule 1 on Restore/Keep — the
alternative — makes the sheet a queue that a mis-tap empties, and it destroys Undo, because a row
that has been removed cannot be undone. **A-1's whole content is that those were two questions, not
one**, and r1 answered both with a single number.

> ## 🛑 AMENDMENT A-1 — REQUIRED BEFORE `sync-rxdb-conflict-notice-ui` IS SLATED
>
> **Filed at morning triage 2026-07-28 (ledger T-26, decision 80). Operator-directed.**
> The rule above is **accepted in substance and rejected in presentation.** Rows still never
> leave the sheet except on Dismiss or retention expiry — that part stands, and Undo depends
> on it. What changes is what the banner prints.
>
> **The defect this closes.** The operator walked the success plate and asked the question the
> design could not answer: *"when she finishes the second, why does it still say three?"* Under
> rule 3 the answer is "because three answers were overwritten and that stays true" — literally
> correct, and wrong on a phone. A number in a coloured banner at the top of the screen reads as
> a **badge**, and badges count outstanding work. A crew member who handles two of three and still
> sees **3** concludes the restores did not take. The past-tense wording ("*were* overwritten")
> does not survive a two-second glance in daylight.
>
> The sheet already shows progress — a restored row turns green and carries Undo. **Only the
> banner is frozen.** So the reason to keep rows (preserving Undo) was never connected to the
> number the banner prints; treating those as one decision was the design's mistake, not the
> operator's.
>
> **What the amendment requires.**
>
> 1. The banner MUST show **both** figures: what happened in the window, and how many rows are
>    still unhandled — e.g. *"3 answers were overwritten · 1 still to review"*, or the headline
>    holding at 3 with a quieter handled/total beneath it. Exact wording is the UI card's to
>    choose; carrying both numbers is not optional.
> 2. Rule 3 is **unchanged for the sheet**: a restored, kept, failed or in-flight row is still
>    drawn and still counted in the "what happened" figure. Counts drop out of that figure only
>    on Dismiss or expiry.
> 3. **"Still to review" needs a definition, and it is not the same as "not green".** A row is
>    reviewed once it has been restored, kept, or dismissed. An **untouched** row counts as still
>    to review. A **failed restore counts as still to review**, not as handled — the crew member
>    has unfinished business there, and the error plate must agree with this.
> 4. **The two banner lines must coexist at 480px.** The second line is already spoken for by
>    `+ N change(s) we couldn't identify` (rule 4). A plate MUST exist showing the unidentifiable
>    case *and* a partially-handled sheet together, or the layout is unproven for the one
>    combination most likely to occur on a bad night.
> 5. Every affected plate MUST be re-shot and read back per the self-verification ritual, and
>    the State Enumeration Table's counting column updated to match.
>
> **Status of the rest of the sign-off.** The mockup is approved **in direction** — the recovery
> path, the degradation ladder, the honesty of the no-value and removed-field rows, and the limits
> panel all stand as drawn. Two decisions are **deliberately deferred until the revised plates
> exist**, because both are easier to judge against a banner that is no longer misleading:
>
> - whether a removed-field row counts in the chip base or moves to `+N` (the "recoverable"
>   ambiguity — today the app counts its two no-Restore cases differently from each other);
> - the retention window, still a **30-day placeholder**.
>
> `sync-rxdb-conflict-notice-ui` therefore **stays ATTENDED-BLOCKED**. A mockup existing is not a
> sign-off, and neither is a partial one.

> ## 🛑 AMENDMENT A-2 — THE OVERRIDE MUST STATE WHAT IT DESTROYS
>
> **Filed at morning triage 2026-07-28 (ledger T-26, decision 81). Operator-directed.**
>
> **The operator's test, in their words:** *"my question is to make sure that it is clear to the
> user on the mobile device that was offline exactly what they were overriding."*
>
> What already passes that test, and must not regress: **both values are always on screen above
> the action that overwrites one of them** — including in the collapsed several-at-once view, where
> the collapse hides *buttons*, never *values*. `Restore all 3 of mine` sits directly beneath the
> three `YOURS` / `NOW SHOWS` pairs it will replace. That is the right instinct and it stays.
>
> Three gaps close:
>
> 1. **The action must name what it replaces, not only what it restores.** "Restore mine" and
>    "Restore all 3 of mine" describe the gain and are silent on the loss. The layout implies the
>    destruction; the words never state it. This matters most on the batch button, where **one tap
>    overwrites three of someone else's values**.
> 2. **The batch override MUST confirm before writing.** A single-row restore may write straight
>    through — both values are right there and Undo covers a mis-tap. `Restore all N of mine` may
>    not: the confirm must show the **N server values about to be overwritten**, so the crew member
>    sees what disappears before it does.
> 3. **The collapsed view must carry the same attribution the expanded view does.** Today the
>    single-conflict plate shows *"Dana M., 6:12 PM"* while the collapsed batch plate shows only
>    *"Dana M."* — no timestamp. **The riskiest action currently carries the least information.**
>    Restore parity.
>
> **Not in scope for A-2, and deliberately so:** adding a confirm to the single-row restore. Undo is
> the safety net there, and a confirmation on every tap is friction on a phone in a hurry.
>
> ### A-2's dependency on the parent card (decision 82)
>
> The `· Dana M., 6:12 PM` attribution is drawn on every `NOW SHOWS` row, but `conflict$` carries
> **no author and no timestamp of its own** — those exist only if the replicated row carries them,
> which is a schema decision owned by `sync-rxdb-schema-and-replication`. The mockup previously
> treated this as *conditional*, degrading to "someone else".
>
> **That conditional is now a hard requirement.** *"You are overwriting someone"* is not adequate on
> a food-safety record; a crew member deciding whether to replace a walk-in cooler reading needs to
> know **whose** reading and **when**. Who-and-when is therefore a **REQUIRED output of
> `sync-rxdb-schema-and-replication`**, not an option it may decline, and this UI card's spec
> depends on it.

---

## State Enumeration Table

Four base states plus **eight** edge rows — six from r1, two added by r2 (marked ✚). The three edge
rows the card names by hand are marked ★. Each row maps 1:1 to a plate in `mockup.html` and to a
screenshot pair in `screenshots/`.

**Four plates in `mockup.html` are deliberately NOT states** and are not in this table: the
`limits` panel (the design's own boundary, carried from r1) and the three r2 plates that exist to
put an **open operator decision** on screen — `openq-count-a`, `openq-count-b`, `openq-retention`.
12 state rows + 4 non-state plates = **16 plates, 32 PNGs**.

| State | Trigger | Visual contract |
|-------|---------|-----------------|
| **empty** | Sheet opened from the sync menu; zero conflict records held locally. | No banner anywhere in the app. Sheet says what would put something here and how long records are kept. Because a non-leader tab, a replication with nothing subscribed to `conflict$`, and an evicted local store **all produce this identical screen**, the copy must be **scoped to the record, not phrased as a guarantee** — headline scoped to the retention window ("Nothing recorded in the last 30 days"), body scoped to what the app caught ("This is what HQ caught and kept — if it wasn't running when a change came in, that change won't be listed"). A flat "Nothing was overwritten" is **banned**: it is a claim the app is not in a position to make, on the screen it shows most often. |
| **loading** | Sheet opened; the local conflict log has not resolved within 500 ms (cold IndexedDB read). | Two skeleton group cards. No spinner. **No count in the header** — the header must not claim a number it does not have. Nothing renders at all under 500 ms. |
| **error** | **Restore mine** tapped and the write does not land: offline, or the server value moved again and the restore conflicted in turn. | The row **stays** and keeps showing both values. Red inline block names which of the two happened in the crew member's words ("Couldn't put 38 °F back — you're offline"), says **where the discarded value still is** — *on this list*, not in the checklist, which still reads the server's value — and offers **Retry**. It must **not** promise an automatic retry: nothing in this design commits to one, and the only retry drawn is the crew member's. **The count does not decrement**, and the plate must make that checkable rather than assert it: the banner is rendered, and **banner headline = group chip = rows drawn**, with a failed row and an in-flight row among them. **A-1: the second figure reads `2 still to review` — both rows, because a failed restore and an in-flight restore are each unfinished business, not handled.** A plate on which the second figure had dropped to 1 or 0 would falsify A-1 rule 3, so this row is where that definition is checkable. A failed restore must never look like a completed one. In-flight: both buttons disabled, label reads "Restoring…". |
| **success** | Replication reconnects; `conflict$` emits for a document; the diff names ≥1 template-backed field carrying a value in `input.newDocumentState`. | Amber banner above the checklist list carrying **both** A-1 figures — headline `2 answers were overwritten`, second line `1 still to review · 1 handled` — plus a plain-language cause. Sheet lists the group; each row shows the pair labelled *Yours* (amber, the crew member's) / *Now shows* (with attribution when the row carries it). Every recoverable row offers **Restore mine** + **Keep theirs**, and **each names the value it replaces on a second label line** (A-2: the button used to describe only the gain). A restored row collapses to a green confirmation that **names the value that came back** and offers **Undo** — a bordered ≥44 px control, never an inline text link, because it is the only escape from a mis-tapped Restore. |
| **edge: row already handled** | **Keep theirs** tapped on a row, or **Undo** tapped on a row that had been restored — the two outcomes one tap off the primary path. | Neither outcome removes the row and **neither changes the "what happened" figure** (counting rule 3): banner headline = chip = rows drawn, unchanged. **The A-1 second figure IS what moves** — the kept row is reviewed, the undone row is back to untouched, so `1 still to review · 1 handled`. One plate, two numbers, only one of them moving: that is the amendment in a single render. A kept-theirs row collapses to a **muted, non-alarming** confirmation naming **the value that is now standing** ("Kept theirs — the checklist reads 41 °F") and keeps an **Undo**. An undone row returns to the full two-value, two-button row with one muted line saying what happened, so the tap is not silent. **Dismiss** is deliberately not drawn: it removes record, row and count together and yields either a smaller sheet or the **empty** state — there is no third rendering. |
| ★ **edge: no discarded value available** | `conflict$` fired, but the diff between `input.newDocumentState` and `output` yields nothing showable — the only difference is bookkeeping (see `_modified` above), or the field is absent/null in the discarded document because the local write cleared it. | The row must **not** claim an answer was lost and must **not** render an empty slot where a value goes. Title reads "A change we couldn't identify"; the value slot reads *Not recoverable* in muted italic; one line explains in plain words. **No Restore button** — there is nothing to put back. Actions are **Open checklist** and **Dismiss** — two, not one. The contract is **per row**: a plate may (and the mockup's does) carry an ordinary recoverable row alongside, which *does* have Restore. Counting per rules 4 and 6: the banner headline carries the recoverable answers (`1 answer was overwritten`), the A-1 line carries `1 still to review`, a third quieter line carries these (`+ 1 change we couldn't identify`), and the group chip appends `+N` (`1 answer +1`) so chip base + `+N` = rows drawn. **The unidentifiable row is in neither of the first two figures.** |
| ★ **edge: several conflicts at once** | A long offline stretch ends; `conflict$` emits once per document, several times in a burst. | **One** banner, never one per conflict, carrying the total and the number of checklists. Sheet groups rows under their document. Every group header carries a count chip so the size of the problem is legible without opening anything. **Collapse rule:** per-row buttons collapse to values-only, with a single **Restore all N of mine** at the foot of each group, when the sheet holds **more than one group** *or* any one group holds **more than two rows** — and when it applies it applies **to the whole sheet**, so a 2-row group beside a 3-row one does not render in a different style (a mixed sheet reads as a bug, not as a rule). A caption says the buttons come back on tap. **Restore all N of mine is styled primary**, not secondary: on a collapsed group it is the only action on the card and must not be its least prominent element. **A-2 adds two contracts to this row.** (a) **Attribution parity:** every `NOW SHOWS` row in the collapsed view carries **name AND time** — r1 showed a bare `Dana M.`, so the riskiest action on the sheet carried the least information. (b) The batch control names **what it replaces** (`replaces 3 of Dana M.'s answers · 6:12–6:14 PM`) and states that it **asks first**; it opens the confirm below rather than writing through. |
| **edge: long value / long question text** | The discarded answer is a **free-text checklist note** — the field type most likely to survive a conflict and the only one with no length bound — or it contains an unbroken token with no spaces to wrap at (a pasted reference, an id, a URL), under a question title long enough to wrap. | The value **wraps inside the card** and is **never truncated**: it is the thing being recovered, so an ellipsis would hide the payload the crew member came for. The 88 px label column keeps its width; attribution may drop to its own line. **The page does not scroll sideways at 480 px** — `document.scrollWidth === document.clientWidth`, measured in the browser by `shoot.mjs`. Note for the implementer: the fixed label column inside a `display:flex` row makes this a real failure mode, not a hypothetical — a flex item's default `min-width:auto` will not shrink below its longest unbreakable word, so `min-width:0` on the row **and** the value plus `overflow-wrap:anywhere` on the value are all three required. |
| ★ **edge: conflict on a field since removed from the template** | The discarded document carries a value for a field id the current template no longer contains — the owner edited the template while the phone was offline. | **No invented label.** The raw field id renders in monospace and muted, so it is visibly not a question title. One line states the question was removed. The value is still shown — it is the thing being recovered. **Restore mine is absent** (nowhere to write it); the recovery is **Copy value**, which is a real recovery on a phone: it goes into a message to the manager in two taps. |
| **edge: local conflict log unreadable** | Sheet opens and the local store holding the conflict records cannot be read — iOS/Safari eviction under storage pressure, or private browsing. W3 named this the largest untested unknown for a phone-first PWA, so it gets a designed state rather than a stack trace. | Red-bordered card. **No fabricated count.** Single **Try again** action, offered without being oversold. The copy must carry **both halves, bad one first**: (a) that if Try again does not bring the list back the record is **permanently gone** and the overwritten answers **cannot be put back** — this is the one screen in the set where something really is unrecoverable, and an evicted IndexedDB is not recovered by tapping a button; (b) that the **checklists themselves are unaffected** and on the server. Half (a) is set at full text contrast, not muted. A crew member must not read a storage failure as "my work was deleted" — and must not read it as "nothing was lost" either. |
| ✚ **edge: partly handled AND unidentifiable, together** | A bad night: a burst of conflicts on one checklist, some rows already dealt with, some the diff could not name. **A-1 requires a plate for exactly this combination** — it is where the banner has the most to say and the least room to say it, and it is the combination most likely on the night the feature matters. | **All three counting lines coexist at 480 px with no truncation and no ellipsis**, plus the plain-language cause line — four lines, which is the worst case, drawn rather than the easiest case: `4 answers were overwritten` / `2 still to review · 2 handled` / `+ 2 changes we couldn't identify` / cause. Chip `4 +2` = **6** rows drawn. Headline 4 = the four answer rows. Still-to-review 2 = the two untouched rows; the restored and kept rows are handled, are **not** in that figure, and **are still drawn and still in the headline**. The two unidentifiable rows are in **neither** of the first two figures. Collapse applies (one group, >2 rows), and the plate proves counting rule 8: **handled rows keep their outcome strip and their Undo under collapse**. The batch control reads `Restore all 2 of mine`, not `all 4` — counting rule 7. Nothing may be `nowrap` or ellipsised: a line that did not fit must **wrap and grow the banner**, and `shoot.mjs` measures every line's `scrollWidth` in both schemes. |
| ✚ **edge: batch override confirm** | `Restore all N of mine` tapped. **The write does not go through on that tap** (A-2). | A confirm card that **names the loss in its title** — `Replace 3 of Dana M.'s answers?`, not "Restore mine?" — and **lists the N server values about to be overwritten**, one per row, each **struck through in the destructive colour** with **who saved it and when**, and each with the crew member's own value on the line beneath, so both are on screen at the moment of decision. A lead paragraph states that their values are what the checklist reads **right now**. A footer states what remains reversible (`each row keeps an Undo — but that is three taps to reverse one`) without overselling it. **Cancel is an equal-weight 44 px control**, not a text link, and sits before the destructive one. **The three values are listed, never summarised as "3 answers"** — a number is the thing a crew member can agree to without reading. **Explicitly out of scope:** a confirm on the single-row restore. Undo is the safety net there and a confirm on every tap is friction on a phone in a hurry. |

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

## How A-1 and A-2 are drawn — requirement by requirement

Each numbered requirement in the two amendment blocks above, and the plate that answers it. **The
sign-off is not discharged by this table** — it is what the operator checks the plates against.

| Req | Where it is drawn | How it can be checked from a screenshot |
|---|---|---|
| **A-1.1** banner shows both figures | Every banner-bearing plate: `success`, `a1-banner`, `outcomes`, `error`, `edge-novalue`, `edge-many`, `openq-count-a`, `openq-count-b` | Two lines under the icon, e.g. `2 answers were overwritten` + `1 still to review · 1 handled`. Machine-checked over all 8 banners by `shoot.mjs` measurement 3. |
| **A-1.2** rule 3 unchanged for the sheet | `outcomes`, `error`, `a1-banner` | The restored / kept / failed / in-flight rows are all **still drawn** and the headline still counts them. `outcomes`: headline 2 = chip 2 = 2 rows across a kept row and an undone row. |
| **A-1.3** still-to-review definition; **a failed restore counts** | `error` (definition's hard case), `outcomes` (kept vs undone), `a1-banner` (restored + kept vs untouched) | `error` reads `2 still to review` over one failed row and one in-flight row. If a plate showed a failed restore as handled, this row fails. |
| **A-1.4** two lines coexist with `+ N change(s) we couldn't identify` at 480 px | `a1-banner` (drawn at **four** lines — the worst case, one more than required), `edge-novalue` (three lines) | Read the PNGs; and `shoot.mjs` measurement 4 asserts `scrollWidth ≤ clientWidth` on **every** banner line in both schemes, exiting non-zero otherwise. |
| **A-1.5** affected plates re-shot; table's counting column updated | all 32 PNGs regenerated; State Enumeration Table rows for success / error / outcomes / novalue / many amended | `git diff` on `screenshots/` and on this file's table. |
| **A-2.1** the action names what it replaces | every Restore control on every plate, plus both batch controls | `Restore mine` carries a second label line `replaces 41 °F`; `Keep theirs` carries `41 °F stays`. Machine-checked by `shoot.mjs` measurement 5 over all 10 Restore controls. |
| **A-2.2** batch confirms before writing, showing the N server values | `a2-confirm` | Three named rows, each with the server value struck through in red, its author and its timestamp, and the crew member's value beneath. `Cancel` + `Replace 3 answers`. |
| **A-2.3** collapsed view carries the same attribution | `edge-many` | All five `NOW SHOWS` rows read `· Dana M., H:MM PM`. r1 read `· Dana M.` with no time. |
| **A-2** out of scope | — | No confirm was added to the single-row restore, deliberately. |

### The two decisions this revision deliberately does NOT settle

Both were reopened at morning triage and are the operator's. **Each is drawn so it is decidable from
the plates**, which is stronger than being listed in prose.

- **(i) Does a removed-field row count in the chip base, or move to `+N`?** **Both readings are
  drawn over identical data** — `openq-count-a` (Reading A: it is an answer; chip `2 answers`, no
  `+N`) and `openq-count-b` (Reading B: it is a `+N`; chip `1 answer +1`). Each plate's caption
  states its own consequence, and neither is recommended: Reading A makes the headline count
  something that cannot be put back, so a `Restore all 1 of mine` would sit under a `2 answers`
  chip; Reading B forces the second banner line to stop saying *"we couldn't identify"* — a removed
  field is identified perfectly well — and become a mixed bucket reading *"can't be put back"*. The
  `edge-removed` plate keeps r1's counting (Reading A) **and says so in its caption**, so it is a
  familiar baseline rather than a silent choice.
- **(ii) The retention window.** Rendered as the token `⟨30⟩`, in a dashed placeholder box, **never
  as prose** — on the `empty` plate and on `openq-retention`, which draws the one screen that prints
  it at two candidate values (`⟨30⟩`, `⟨7⟩`) to show the copy is indifferent to the number. Both are
  captioned as placeholders and neither is a recommendation. Implementations read it from one named
  constant (ledger T-27, decision 89).

---

## `done_when:`

```yaml
done_when:
  # ── the artifact this card owes ────────────────────────────────────────────
  - "1. mockup.html exists at .planning/phases/sync-rxdb-conflict-notice/mockup.html — the exact path CLAUDE.md's sign-off gate reads; ls the path"
  - "2. Every State Enumeration row has a plate in mockup.html carrying its trigger and visual contract as a visible caption — count plates against table rows, 1:1 (12 state rows + the limits panel + 3 open-decision plates = 16 plates, 32 PNGs). A row with no plate, or a plate with no row and no place in the 4-plate non-state list, fails this"
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
  - "12. The counting rule is stated in this file AND obeyed by every plate — read all 16 light PNGs; on each plate that draws a group, chip base + any '+N' == field rows drawn in that group; on each plate that draws a banner, the headline == the answer rows drawn, the still-to-review line == the untouched/failed/in-flight rows, and the '+N' line (when present) == the unidentifiable rows. REWRITTEN at r2: the old wording called the unidentifiable count 'the second line', which A-1 made false — the second line is now the still-to-review figure and the unidentifiable count moved to a third. A criterion that names the wrong line cannot catch a plate that gets it wrong. Any plate that disagrees fails this row"
  - "13. Every plate caption agrees with what that plate draws — read each PNG's caption against its own render; specifically edge-novalue names BOTH Open checklist and Dismiss, and edge-many's collapse rule accounts for its own 2-row collapsed group"
  - "14. The two outcomes one tap off the primary path are drawn, not just described — read outcomes-light.png: a kept-theirs row naming the standing value with an Undo, and an undone row back to two values + two buttons, with the count unchanged across both"
  - "15. Long content does not break the layout — read edge-longvalue-light.png: a multi-line free-text note and an unbroken token both wrap inside the card with no truncation and no ellipsis. Machine-checked: shoot.mjs asserts document.scrollWidth === document.clientWidth at viewport 480 in both schemes and exits non-zero otherwise"

  # ── house constraints ──────────────────────────────────────────────────────
  - "16. Renders at 480px with no horizontal overflow in either scheme — measured by shoot.mjs (scrollWidth vs clientWidth, both schemes), not judged by eye; the script exits non-zero on failure"
  - "17. Both colour schemes render legibly from the shared CSS variable block — screenshots/*-light.png and *-dark.png pairs exist for all 16 plates (32 PNGs) and were read back. A glyph that renders as a tofu box in the headless font stack fails this row: it did, on the first r2 render, for the U+1F6D1 marker used on the three open-decision captions"
  - "18. EVERY interactive element in the design is a >=44px touch target in both dimensions — not just the two classes already known to pass. shoot.mjs measures getBoundingClientRect over '.cf-btn, .cg-all, .cf-done-undo, .sc-close, .cn-banner-go, .sc-err button, .sc-empty button, .cfm-go, .cfm-cancel' in both schemes, prints the count measured (58 at r2, up from 38) and every element under 44px, and exits non-zero if the list is non-empty. Undo, Done and the banner's Review are inside that selector. Grepping the stylesheet does NOT satisfy this row — it was what let three sub-44px controls through"
  - "19. Zero production files changed — git diff --name-only 9bd9a72..HEAD lists nothing outside .planning/phases/sync-rxdb-conflict-notice/ and .night-crew/runs/2026-07-29-2-autonomous/ and .night-crew/knowledge/roadmap.md"
  - "20. No version constant moved — version.go Backend 0.3.0 and package.json 1.2.1 both byte-identical to the branch point; git diff on both paths is empty"

  # ── r2: amendment A-1 (ledger T-26 decision 82) ────────────────────────────
  - "21. EVERY banner in the file carries BOTH figures — the what-happened headline AND a still-to-review figure. Machine-checked: shoot.mjs measurement 3 walks every .cn-banner in both schemes, requires a .cn-banner-hd AND a .cn-banner-open whose text matches /(\\d+ still to review|All \\d+ reviewed)/, prints the count of banners carrying only one, and exits non-zero if any does. HOW IT FAILS: it did, red, on the un-amended r1 mockup — 5 banners, 5 carrying one figure. Mutation-tested green-side by deleting every .cn-banner-open at runtime: 8 banners flagged."
  - "22. The still-to-review figure is ARITHMETICALLY RIGHT on every banner plate, not merely present — read the 8 light PNGs and for each compute: still-to-review == rows drawn that are untouched, failed or in-flight; handled == rows that are green-restored or muted-kept; still-to-review + handled == the headline; unidentifiable rows are in NEITHER. Any plate where the printed figure disagrees with the rows drawn beneath it fails. HOW IT FAILS: a-1-banner prints 2/2 over 6 rows — miscount either side of the split and the numbers stop adding to 4."
  - "23. A FAILED restore counts as still to review, not as handled — read error-light.png: the second banner line reads 2 still to review over exactly two rows, one of which shows the red Couldn't-put-back block and one of which is mid-flight. HOW IT FAILS: if that line read 1 or 0, or if the failed row were rendered as an outcome strip, A-1 rule 3's definition would be contradicted by its own plate."
  - "24. A-1's PARK trigger is measured, not judged — NO banner line truncates or ellipsises at a 480px viewport in EITHER scheme. shoot.mjs measurement 4 checks scrollWidth <= clientWidth + 1 and text-overflow != ellipsis on every .cn-banner-hd/-open/-unid/-sub, prints the count measured (24) and every offender, and exits non-zero. HOW IT FAILS: mutation-tested by injecting white-space:nowrap;overflow:hidden;text-overflow:ellipsis on those selectors — 24 lines flagged. If it could not fail, the PARK trigger would be unenforceable."
  - "25. The worst-case banner is DRAWN, not described — a1-banner-light.png and a1-banner-dark.png each show FOUR banner lines together: the headline, '2 still to review · 2 handled', '+ 2 changes we couldn't identify', and the plain-language cause. Read both PNGs; assert all four are fully legible and none is clipped. HOW IT FAILS: A-1.4 asks only that two lines coexist with the +N line; a plate drawn at the minimum would pass A-1.4 and fail this row, which is deliberate — the combination most likely on a bad night is the one that must be proved."

  # ── r2: amendment A-2 (ledger T-26 decision 82) ────────────────────────────
  - "26. EVERY control whose label begins with 'Restore' also names what it replaces. Machine-checked: shoot.mjs measurement 5 over .cf-btn, .cg-all and .cfm-go in both schemes, printing the count of Restore controls and every one silent about the loss, exiting non-zero if the list is non-empty. HOW IT FAILS: red on r1 — 7 of 7 silent. Mutation-tested by removing the .cf-btn-s/.cg-all-s sub-labels at runtime: 10 controls flagged."
  - "27. The batch override CONFIRMS before writing and the confirm shows the N server values — read a2-confirm-light.png and assert ALL of: the title names the loss and counts it ('Replace 3 of Dana M.'s answers?'); exactly three server values are listed by question name; each is struck through AND carries both an author and a clock time; each has the crew member's own value on the line beneath; a Cancel control is present at equal weight. HOW IT FAILS: a confirm that said '3 answers will be replaced' without listing them would satisfy 'confirms before writing' and fail this row — which is the point, since a number is what a crew member can agree to without reading."
  - "28. The collapsed view carries the SAME attribution the expanded view does — read edge-many-light.png and assert every one of the five 'Now shows' rows carries a name AND a clock time. HOW IT FAILS: r1's render is the counter-example — all five read '· Dana M.' with no time. One bare name on any row fails this."
  - "29. The batch control's N is the still-to-review count, not the chip base — read a1-banner-light.png: chip reads '4 answers +2' while the batch control reads 'Restore all 2 of mine'. HOW IT FAILS: if it read 'all 4', one tap would re-write the row the crew member deliberately kept, silently reversing their own decision; a plate showing 'all 4' under a partially-handled group fails."

  # ── r2: the two decisions that must stay OPEN ──────────────────────────────
  - "30. Open decision (i) is drawn BOTH ways over IDENTICAL data and neither is chosen — read openq-count-a-light.png and openq-count-b-light.png: same two rows, same values, same group; chip '2 answers' vs '1 answer +1'; banner headline 2 vs 1; Reading B carries a third banner line whose wording differs from Reading A's absence of one. Each caption must contain the words NOT SETTLED and state its own consequence. HOW IT FAILS: drawing only one reading, drawing them over different data (which would make them incomparable), or any caption recommending one."
  - "31. No plate silently settles open decision (i) — grep mockup.html for every group chip and read edge-removed-light.png: the one plate that counts a removed-field row outside the openq pair MUST say in its own caption that its chip is Reading A and that the decision is not settled. HOW IT FAILS: r1's edge-removed plate is the counter-example — it printed '1 answer' with no note, which is a choice made by omission."
  - "32. The retention number is a VISIBLE placeholder wherever it renders, never prose — grep mockup.html: every occurrence of the retention figure sits inside a <span class=\"ph\"> token; assert zero bare occurrences of the retention figure inside `.sc-empty` — the app's own copy — where r1 printed it as plain prose. The `.ph-key` annotation beneath the sheet is EXEMPT and is expected to name the number in words: its entire job is to say the number is not decided, and a criterion that banned it would ban the fix. Read empty-light.png and openq-retention-light.png: the number renders in a dashed box as ⟨30⟩ and a placeholder key states it was accepted at the 18:12 sign-off and REOPENED at triage. HOW IT FAILS: r1's empty plate is the counter-example — 'Nothing recorded in the last 30 days' as plain prose, indistinguishable from a decision."
  - "33. openq-retention shows the SAME screen at two candidate values with the body copy unchanged, and captions BOTH as placeholders — read openq-retention-light.png: ⟨30⟩ and ⟨7⟩, identical body text, plus a key stating ⟨7⟩ is not a counter-proposal. HOW IT FAILS: presenting the second value as a recommendation, or changing the copy between them, would turn an illustration of indifference into an argument for one."

  # ── r2: the card's own boundary ────────────────────────────────────────────
  - "34. The sign-off is NOT discharged and the block is NOT lifted — grep this file for the status line and roadmap.md for the sibling card: sync-rxdb-conflict-notice-ui must still read ATTENDED-BLOCKED, and this file must still say nothing is approved. HOW IT FAILS: flipping either would be this card grading its own work; the correct outcome of this card is a blocked sibling and a revised artifact."
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
- **Where the durable conflict record lives.** Answered in part after r1: ledger T-27 decision 89 makes it a **personal, per-device undo in a local-only RxDB collection** — no server table, no endpoint, no replication of the record itself.
- **How long it is kept.** 30 days was accepted at the 18:12 sign-off and **reopened at morning triage 2026-07-28**. r2 renders it as the placeholder token `⟨30⟩` on the `empty` plate and draws `openq-retention` to make the choice decidable. **This card does not recommend a value.**
- **Whether a removed-field row counts in the chip base or moves to `+N`** — open decision (i), reopened at the same triage. r2 draws **both** readings over identical data (`openq-count-a`, `openq-count-b`) and picks neither.
- **Any RLS predicate.** Row visibility is obligation 1 of the parent card.

## Open question for the operator

Beyond roughly ten conflict groups the sheet needs a cap or a date filter; it is not designed here.
Say so if you think a truck can realistically get there — one long dead-zone shift with an active
manager is the scenario to judge it against.
