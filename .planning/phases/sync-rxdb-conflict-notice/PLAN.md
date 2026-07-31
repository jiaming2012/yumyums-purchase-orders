# PLAN — `sync-rxdb-conflict-notice-ui` (the BUILD)

Run `overnight-20260801`, card C2, branch `card/c2-sync-rxdb-conflict-notice-ui`.

**This file is the build's contract.** `UI-SPEC.md` beside it is the *mockup's* contract — its
`done_when:` block grades the plates, and it is discharged. This one grades the **implementation**,
and it is the document the verifier subagent reads. Per CLAUDE.md the verifier's inputs are ONLY:
this file's `done_when:` block, the State Enumeration Table below, the diff, and the screenshots.
Nothing from the build conversation and nothing from the implementer's reasoning.

**Scope.** The user-visible half of ledger T-22 **decision 50**: when a same-field clash falls back
to master-wins, `conflict$` must surface it to the crew member **with the discarded value
recoverable** — not silently dropped. Built against **revision 2** of the plates, signed at morning
triage 2026-07-29 (**decision 98**), plus the three obligations from that walk: **A-1** (the banner
carries both figures), **A-2** (the override states what it destroys), **A-3** (decision 95 — a
removed question keeps its label, struck through and read-only).

---

## 🛑 What is DORMANT, stated before anything else

The sheet renders from the durable local conflict record. The record is written when `conflict$`
fires. `conflict$` fires when replication runs. **Replication is not started in this tree** —
`HQ_SYNC_REST_URL` is unset in every environment and the `/sync` door answers 503 by design until
`sync-rxdb-row-visibility-rls` lands, and `sync-hard-cutover` owns the write-path swap.

So **in production today there are zero records, the banner never appears, and nothing about
`workflows.html`'s existing behaviour changes.** `autoSaveField` → `submitOp('SET_FIELD')` →
`DRAFT_RESPONSES` → `hydrateFieldState` is byte-untouched.

Every state below is therefore forced from a **seeded store** through the same seam the cutover card
will hand a real RxDB collection to (`window.HQConflictNotice.mount({store})`). That is the honest
way to verify a screen whose producer is not switched on yet, and it is said here so no reader
mistakes "the screenshots are seeded" for a gap that was hidden.

---

## State Enumeration Table

Four base states plus **ten** edge rows. Two of the ten are this card's own additions beyond the
mockup's twelve (marked ✚) and both exist because the *implementation* can fail in ways a static
plate cannot: a malformed `template_snapshot`, and more groups than the cap.

Each row names its trigger, its visual contract, and the **population floor** the spec asserts
before the shutter — because a screenshot spec that renders an empty sheet passes vacuously
(B-22/B-23/B-24), and seeding zero records into `success` produced exactly such a clean, passing PNG
before the floors were added.

Screenshots live in `test-results/states-sync-rxdb-conflict-notice/`. States with a banner have
**two frames**: `<name>-banner-*.png` (sheet closed — the banner as a crew member meets it, over the
real checklist list) and `<name>-*.png` (the sheet). The real sheet is `position:fixed` over My
Checklists, so unlike the mockup's plates one capture cannot show both.

| State | Trigger | Visual contract | Population floor |
|---|---|---|---|
| **empty** | Sheet opened; zero records in the window. | No banner anywhere. Headline scoped to the retention window and printing the number **from the one named constant** (decision 96). Body scoped to what HQ caught. A flat "Nothing was overwritten" is **banned** — a non-leader tab, an unsubscribed replication and an evicted store all produce this identical screen. | 0 banners, 0 groups, 0 rows, exactly 1 `.sc-empty` |
| **loading** | Sheet opened; the store has not resolved within 500 ms. | Two skeleton cards, no spinner, **no count in the header** — it must not claim a number it does not have. Nothing renders at all under 500 ms. | 0 banners, exactly 2 `.sk`, header contains no digit |
| **error** | Restore tapped and the write does not land — offline, or the server moved again. | The row **stays** and keeps both values. Red inline block names which of the two happened, says the discarded value is still **on this list** (not in the checklist), and offers **Retry**. It must **not** promise an automatic retry. **A-1: `2 still to review` over one failed and one in-flight row** — both are unfinished business. headline = chip = rows drawn. In-flight: both buttons disabled, label reads "Restoring…". | 1 banner, 1 group, exactly 2 rows |
| **success** | `conflict$` emitted; the diff named ≥1 template-backed field carrying a value. | Amber banner carrying **both** A-1 figures. Each row shows *Yours* / *Now shows* with attribution. Every recoverable row offers Restore + Keep and **each names the value it replaces**. A restored row collapses to a green confirmation that **names the value that came back** and keeps a bordered ≥44 px **Undo**. | 1 banner, ≥3 banner lines, 1 group, exactly 2 rows |
| **edge: row already handled** | Keep theirs tapped; Undo tapped on a restored row. | Neither removes the row and **neither moves the "what happened" figure**. The **second figure is what moves**. A kept row collapses to a muted confirmation naming the value now standing and keeps Undo; an undone row returns to two values + two buttons with one muted line, so the tap is not silent. | 1 banner, 1 group, exactly 2 rows |
| **edge: no discarded value available** | The diff yields nothing showable — bookkeeping only, or the local write cleared the field. | Must **not** claim an answer was lost and must **not** render an empty value slot. Title "A change we couldn't identify", value slot *Not recoverable*, one plain-words line. **No Restore.** Actions are **Open checklist and Dismiss — two, not one.** The contract is **per row**: an ordinary recoverable row sits above it and does have Restore. Counting: headline 1, still-to-review 1, third line `+1`, chip `1 answer +1`. | 1 banner, 1 group, exactly 2 rows, exactly 1 `.cf.unrec` with exactly 2 buttons |
| **edge: several conflicts at once** | A long offline stretch ends; `conflict$` emits in a burst. | **One** banner, never one per conflict. Groups under their document, each with a count chip. Collapse when the sheet holds >1 group **or** any group holds >2 rows, applied **sheet-wide**. Batch control **styled primary**. **A-2.3: every `Now shows` row carries name AND time**, collapsed included. The batch names what it replaces and says it **asks first**. | 1 banner, exactly 2 groups, exactly 5 rows, exactly 5 attribution strings, exactly 2 batch controls |
| **edge: long value / long question text** | A free-text note, or an unbroken token with no spaces to wrap at, under a wrapping question title. | The value **wraps inside the card** and is **never truncated** — it is the thing being recovered. The 88 px label column keeps its width. `document.scrollWidth === clientWidth` at 480 px. | 1 group, exactly 2 rows, 0 clipped/ellipsised values, label column exactly 88 px |
| **edge: field since removed from the template** | The discarded document carries a value for a field id the live template no longer has. | 🛑 **A-3.** The row renders the question's **own label from `template_snapshot`, struck through**, in the **same type as any question title** — not monospace, not a raw id. Visibly **read-only**: no Restore, and the strike is what says so. The value is still shown in full; **Copy value** is the recovery, with Dismiss. Counting follows **Reading A**: it is in the chip base. | 1 banner, 1 group, exactly 2 rows, exactly 1 `.cf-q-struck`, zero `.cf-q-gone` |
| ✚ **edge: MALFORMED `template_snapshot`** | The snapshot carries no label for the id — including because it is junk. **B1's recorded-not-fixed item R-C**: `template_snapshot` is `{type:'object'}` with no nested `properties` and nothing rejects a malformed value; A-3 makes its shape load-bearing, so this is a dependency, not a curiosity. | A-3's **fallback**: the raw field id in muted monospace, **exactly as revision 2 drew it** — visibly not a question title. Copy value and Dismiss unchanged. **No page error**, no blank row, no invented label. | 1 banner, 1 group, exactly 2 rows, exactly 1 `.cf-q-gone`, zero `.cf-q-struck`, **zero pageerrors** |
| **edge: local conflict log unreadable** | The store cannot be read — iOS/Safari eviction, or private browsing. | Red-bordered card, **no fabricated count**, single **Try again** offered without being oversold. Copy carries **both halves, bad one first**: (a) if Try again does not work the record is **permanently gone** and the answers **cannot be put back**; (b) the checklists themselves are fine. **Half (a) at full text contrast, not muted.** | 0 banners, 0 rows, exactly 1 `.sc-err`, exactly 2 paragraphs, strong colour ≠ muted colour |
| **edge: partly handled AND unidentifiable together** | A bad night: a burst on one checklist, some rows handled, some the diff could not name. **A-1 requires a plate for exactly this.** | **All four banner lines coexist at 480 px with no truncation and no ellipsis.** Chip `4 +2` = 6 rows. Headline 4; still-to-review 2 (the restored and kept rows are handled, still drawn, still in the headline); the two unidentifiable rows in **neither** of the first two figures. Collapse applies, and counting rule 8 holds in **both** halves: handled rows keep their outcome strip and Undo, and the unidentifiable rows keep Open checklist + Dismiss. Batch reads `Restore all 2 of mine`, not `all 4`. | 1 banner, **exactly 4** banner lines, 1 group, exactly 6 rows, exactly 2 Undos, exactly 2 `.cf.unrec` × 2 buttons |
| **edge: batch override confirm** | `Restore all N of mine` tapped. **The write does not go through on that tap.** | A confirm that **names the loss in its title** and **lists the N server values**, each **struck through in the destructive colour** with **who saved it and when**, and each with the crew member's own value beneath. Footer states what stays reversible without overselling it. **Cancel at equal weight, before the destructive control.** Primary reads **Replace**. The three values are **listed, never summarised**. | exactly 3 `.cfm-row`, each with exactly 1 struck value + 1 own value, 0 restored rows after the tap |
| ✚ **edge: more groups than the cap** | More than ten conflict groups in the window (decision 97). | Exactly **10 groups** drawn plus an **"and N more"** line. Rows below the line are **not dropped** and the **banner reports the TRUE total**. **No date filter** — decision 97 rejected one. | 1 banner, **exactly 10** groups, exactly 10 rows, cap line present, zero date inputs |

---

## `done_when:`

```yaml
done_when:
  # ── A-1: the banner carries BOTH figures (decision 82, overturning the struck rule)
  - "1. Every rendered banner carries a what-happened headline AND a still-to-review figure — read success-banner-light.png, error-banner-light.png, outcomes-banner-light.png, edge-novalue-banner-light.png, edge-many-banner-light.png, a1-banner-banner-light.png, edge-removed-banner-light.png, edge-cap-banner-light.png; each shows two lines, not one. HOW IT FAILS: a banner printing only the headline is the exact defect A-1 was filed against, and a run that implements the struck single-count rule 'as drawn' reinstates it."
  - "2. The what-happened figure is FROZEN under Restore and Keep, and the still-to-review figure MOVES — driven, not seeded: tests/states-sync-rxdb-conflict-notice.spec.js 'Restore -> green + Undo' taps Restore and asserts the headline reads '2 answers were overwritten' before and after, while the second line goes '2 still to review' -> '1 still to review · 1 handled' -> '2 still to review' on Undo. A static plate cannot show a transition; this drives one."
  - "3. A FAILED restore counts as still to review, not as handled — read error-banner-light.png: '2 still to review' over one failed row and one in-flight row. HOW IT FAILS: if it read 1 or 0, A-1 rule 3's definition would be contradicted by its own screen."
  - "4. The worst case is DRAWN, not described — read a1-banner-banner-light.png and a1-banner-banner-dark.png: FOUR lines together (headline, '2 still to review · 2 handled', '+ 2 changes we couldn't identify', cause), all fully legible, none clipped. Machine-checked in the same test: every banner line's scrollWidth <= clientWidth and no text-overflow:ellipsis, with the line population floored at 4."
  - "5. No banner line truncates or ellipsises at 480px in either scheme, and the page never scrolls sideways — assertNoBannerTruncation() measures it in the browser on the banner-bearing states and exits the test red otherwise. This is the card's PARK trigger, so it is measured rather than judged."

  # ── A-2: the override states what it destroys
  - "6. EVERY control that overwrites someone else's value names what it replaces — read success-light.png ('Restore mine / replaces 41 °F'), error-light.png ('Retry / replaces 41 °F' AND 'Restoring… / replacing No'), edge-many-light.png (both batch sub-labels carry 'replaces'). Population is scoped by what the control DOES: Retry is the same destructive write as Restore, on the one screen where the crew member has already failed once."
  - "7. The batch tap does NOT write — read edge-confirm-light.png, and the same test asserts zero rows are in the restored state while the confirm is open. HOW IT FAILS: a confirm that appeared AFTER the write, or a batch that wrote through, would still screenshot as a confirm."
  - "8. The confirm names the loss in its title and LISTS the N server values — read edge-confirm-light.png: title 'Replace 3 of Dana M.'s answers?'; exactly three rows named by question; each server value struck through in the destructive colour (computed textDecorationLine contains line-through, asserted) with an author AND a clock time; each with the crew member's own value beneath. HOW IT FAILS: '3 answers will be replaced' without listing them would satisfy 'confirms before writing' and fail this row — a number is what a crew member can agree to without reading."
  - "9. Cancel is an equal-weight >=44px control and sits BEFORE the destructive one, whose label reads Replace — asserted on the rendered order of .cfm-acts children and on the button text, and visible in edge-confirm-light.png."
  - "10. The batch acts ONLY on rows still to review — driven: 'the batch confirm COMMITS only on Replace' seeds three rows one of which is already Kept, asserts the control reads 'Restore all 2 of mine' under a '3 answers' chip, commits, and asserts the KEPT row is still kept. HOW IT FAILS: re-writing a row the crew member deliberately kept silently reverses their own decision."
  - "11. Attribution parity: every 'Now shows' row carries name AND time, collapsed included — read edge-many-light.png; the test asserts all five match /Dana M\\., \\d{1,2}:\\d{2} (AM|PM)/. HOW IT FAILS: r1's render is the counter-example — five bare 'Dana M.' with no time, so the riskiest action carried the least information."

  # ── A-3 (decision 95): a removed question keeps its label
  - "12. A removed question renders ITS OWN LABEL, struck through, in the same type as a question title — read edge-removed-light.png and edge-removed-dark.png: 'Prep sink temperature', struck. Machine-checked: computed textDecorationLine contains line-through AND computed fontFamily does NOT match /mono/, and zero .cf-q-gone elements are present. HOW IT FAILS: the raw field id in monospace is what revision 2 drew and what decision 95 overturned."
  - "13. The row is visibly READ-ONLY and Copy value is the recovery — read edge-removed-light.png: no Restore anywhere on the row, Copy value + Dismiss present, and the value (72 °F) shown in full because it is the thing being recovered."
  - "14. The raw field id survives ONLY as the fallback, drawn exactly as revision 2 drew it — read edge-removed-fallback-light.png: 'fld_prep_sink_temp' in muted monospace (computed fontFamily matches /mono/), zero .cf-q-struck elements on that row, Copy value still offered."
  - "15. A MALFORMED template_snapshot degrades to that fallback and DOES NOT THROW — the same test hands the renderer {sections:'this is not a template'}, listens on pageerror, and asserts zero errors. Backed by 18 malformed shapes plus a cycle plus a throwing property accessor in tests/sync-rxdb-conflict-notice.spec.js, each asserting fieldLabelFromSnapshot returns null rather than throwing, recursing forever, or rendering nothing. HOW IT FAILS: B1's item R-C means nothing rejects a malformed value, so 'the snapshot is well-formed' is an assumption with no enforcement behind it anywhere in the system."
  - "16. Counting follows READING A — read edge-removed-light.png: chip '2 answers' over two rows, no '+N', and the banner headline 2. The consequence decision 95 accepted is VISIBLE rather than hidden: a group whose only restorable row is one of two prints 'Restore all 1 of mine' under a '2 answers' chip, and the struck-through read-only row is what makes that legible."

  # ── (b) STANDS: rows leave only on Dismiss or expiry
  - "17. Restore and Keep never remove a row, and both keep an Undo — driven: after Restore the sheet still holds 2 rows; read outcomes-light.png for the kept row's muted confirmation naming the value now standing, with a bordered Undo, and the undone row back to two values + two buttons with one muted line."
  - "18. Dismiss is the ONLY way a row leaves — driven: 'Dismiss is the only way a row leaves the sheet' taps Dismiss on the unidentifiable row and asserts the row count drops 2 -> 1, the '+N' banner line disappears and the chip goes '1 answer +1' -> '1 answer'."
  - "19. Collapse hides the Restore/Keep pair and NOTHING else — read a1-banner-light.png: on a collapsed sheet the two handled rows keep their outcome strip and their Undo (2 Undos asserted), and the two unidentifiable rows keep Open checklist + Dismiss (2 buttons each, asserted). HOW IT FAILS: a collapse that ate those actions would strand a row whose only exit is Dismiss on the sheet with the most rows to get through; and after a batch restore, if collapse ate Undo, a batched mis-tap would be irreversible."

  # ── decisions 96 and 97
  - "20. The retention window is read from ONE named constant — read empty-light.png for the rendered number, and the test compares it against CONFLICT_RECORD_RETENTION_DAYS imported from sync-schema/collections.js rather than against a literal. sync-rxdb/conflict-notice.js imports and re-exports that constant; grep the repo's new files for a second occurrence of the figure and find none."
  - "21. The sheet caps at 10 groups, rows below the line are NOT dropped, and the banner reports the TRUE total — read edge-cap-light.png (10 groups + 'and 3 more checklists · 3 more rows — the count above is the true total') and edge-cap-banner-light.png ('13 answers were overwritten'). Zero date inputs are rendered: decision 97 rejected a date filter."

  # ── the empty / loading / storage-error contracts
  - "22. The empty state is scoped to the RECORD, never phrased as a guarantee — read empty-light.png; the test asserts the copy does NOT contain 'nothing was overwritten' and DOES contain 'This is what HQ caught and kept'. HOW IT FAILS: a non-leader tab, an unsubscribed replication and an evicted store all produce this identical screen, so a flat claim is one the app cannot make — on the screen it shows most often."
  - "23. Loading shows two skeletons and NO COUNT in the header — read loading-light.png; the test asserts the header text contains no digit and nothing renders under 500 ms."
  - "24. The storage-error copy carries BOTH halves, bad one first, and half (a) is not muted-only — read edge-storage-light.png; the test asserts paragraph 1 contains 'it's gone' and 'can't be put back', paragraph 2 contains 'checklists are not affected', and the computed colour of the strong run differs from the muted paragraph colour. Neither half alone passes."

  # ── house constraints
  - "25. Every interactive control is a >=44px target in both dimensions, AND the population is floored per state — assertTapTargets() measures getBoundingClientRect over the banner's Review, .cf-btn, .cg-all, .cf-done-undo, .sc-close, the storage button and both confirm buttons, asserts the count is at or above the state's floor FIRST, and lists every offender. Grepping the stylesheet does NOT satisfy this row: walking the controls that exist is what let a deletion pass green on the mockup's own hardening round."
  - "26. Both colour schemes render legibly and no glyph tofus — 46 PNGs exist under test-results/states-sync-rxdb-conflict-notice/, one pair per state plus one banner pair per banner-bearing state, and they were read back with a multimodal Read."
  - "27. Every state asserts a POPULATION FLOOR before the shutter — assertPopulation() takes an exact expected count per bucket and is called in all 14 state tests. HOW IT FAILS: this repo's characteristic bug is a check whose subject set can go empty, and a screenshot spec is exactly that shape — seeding zero records into `success` produced a clean, empty, entirely passable PNG before the floors existed."
  - "28. The live persistence path is BYTE-UNTOUCHED — git diff the autoSaveField / submitOp('SET_FIELD') / DRAFT_RESPONSES / hydrateFieldState region of workflows.html and find nothing. No write path is swapped; that is sync-hard-cutover."
  - "29. HQ_SYNC_REST_URL is set nowhere — grep the diff and the repo. Nothing deploys tonight, and replication is not started."

  # ── the G6 correction that landed on this card
  - "30. describeConflict and the handler it was configured beside AGREE — tests/sync-rxdb-conflict.spec.js 'describeConflict agrees with the handler it was configured beside (G6)', four tests, pinned with a CUSTOMISED field on both halves of the option surface (provenanceFields and reservedFields). RED FIRST: run against the unfixed tree with the fix stashed, 4/4 failed. HOW IT FAILS: a custom provenance field is suppressed as a clash by resolve() and, unthreaded, came back as a ROW ON THIS SHEET — a row for a value nothing lost, on the screen whose whole thesis is that its numbers are true."
```

---

## What this card did NOT decide

- **The write path.** `applyRestore` is injected. Today `workflows.html` hands it the live
  `submitOp('SET_FIELD')` path; `sync-hard-cutover` replaces it with the RxDB local write and
  changes nothing else in the renderer.
- **When replication starts, and `HQ_SYNC_REST_URL`.** Not this card's, not tonight's.
- **`_deleted`.** C1 recorded it as an open question inherited by `sync-hard-cutover`, and the
  severity is higher than C1's first draft claimed: HQ hard-deletes three of the four mirrored
  tables from live paths — `saveResponse` deletes a `submission_responses` row on a null value,
  which is **unchecking a checkbox**, the tool's highest-frequency write. It is unreachable today
  **only** because no page writes through RxDB. Nothing in this card touches it, and the withdrawn
  "HQ's domain doesn't delete" claim is not restated anywhere here.
- **B1's item R-C as a schema fix.** This card **handles** a malformed `template_snapshot`; it does
  not constrain it. Constraining the snapshot's shape in `sync-schema` would make that file a
  second, drifting definition of the builder's output, which is exactly the reason B1 gave for
  leaving it open. Handling it at the read site is the cheaper half and it is the half this UI needs.
