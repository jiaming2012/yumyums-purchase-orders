# Merge intent — Card C1 `sync-rxdb-conflict-notice-mockup-amendments`

Branch: `card/c1-sync-rxdb-conflict-notice-mockup-amendments` (cut from `overnight-20260729-2` @ `9bd9a72`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

**Amend an existing, committed artifact — do not start over.** The 2026-07-29 run produced
`.planning/phases/sync-rxdb-conflict-notice/{mockup.html,UI-SPEC.md,screenshots/}` (11 plates, 22
PNGs, 20 `done_when:` rows passing). Morning triage 2026-07-28 (ledger T-26 **decision 82**)
superseded that sign-off **in part** and filed two amendments — **A-1** (the banner must carry what
happened AND how many rows are still unhandled) and **A-2** (the override must state what it
destroys, confirm before writing, and carry attribution parity). Both are already written into
`UI-SPEC.md` as the contract. This card draws them. It **does NOT discharge the sign-off**:
`sync-rxdb-conflict-notice-ui` stays **ATTENDED-BLOCKED** when this card is done, which is the
correct outcome, not a failure. **Zero production code.**

## Shared files touched

- `.planning/phases/sync-rxdb-conflict-notice/mockup.html` — **existing file, amended.** Every
  banner-bearing plate gains A-1's second figure; every restore control gains A-2's "what it
  replaces"; the collapsed batch view gains the timestamp it was missing. New plates are appended
  for the cases the amendments require to be *proved* rather than described. No other card on
  tonight's slate names `.planning/phases/`. No conflict surface.
- `.planning/phases/sync-rxdb-conflict-notice/UI-SPEC.md` — **existing file, amended.** The State
  Enumeration Table is extended for the new plates, the counting rule gains A-1's still-to-review
  definition, and the `done_when:` block gains rows for each new plate. The two 🛑 AMENDMENT blocks
  (A-1, A-2) are the *contract this card is graded against* and are **kept verbatim** — a merge that
  drops them drops the reason the plates changed. No conflict surface.
- `.planning/phases/sync-rxdb-conflict-notice/screenshots/*.png` + `shoot.mjs` — **existing,
  regenerated and extended.** ~~`shoot.mjs` already measures horizontal overflow and every interactive
  element's touch-target box, exiting non-zero on failure.~~ **STRUCK AT THE HARDENING ROUND — the
  inherited touch-target check measured every interactive element that EXISTS, which is not the same
  claim. It had no population floor, so DELETING controls passed green: removing all four
  `.cf-done-undo` Undo controls printed `58 measured, 0 under 44px -> PASS` and exited 0. It is now
  floored at `EXPECTED_TAP_TARGETS` (62), as is the banner-line check at 24.** This card adds
  measurements for the
  contracts A-1 and A-2 introduce that also cannot be judged by eye — that **every** banner carries
  **both** figures, that **no banner line is truncated at 480px** (A-1's PARK trigger, so it must be
  measured and not asserted), and ~~that the batch override names what it replaces~~ ~~**STRUCK at
  close-out — the check landed BROADER than the guess: it covers every control whose label begins
  with "Restore" (10 of them), not just the two batch buttons. Narrowing it back to `.cg-all` would
  silently un-check the eight single-row restores.**~~ **STRUCK AGAIN AT THE REPAIR ROUND — "every
  control whose label begins with Restore" was the wrong population, not a broader one. It excluded
  `Retry` and `Restoring…` on `plate-error` (the same destructive write, on the plate where the crew
  member has already failed once) and it EVAPORATED under a rename while printing `0 Restore
  controls, 0 silent -> PASS`. The check is now scoped by what a control DOES — everything in
  `.cf-btn, .cg-all, .cfm-go` that is not one of five non-destructive labels — with a floor on the
  population so a rename or a deletion reds. 13 controls, not 10.** **A merge that keeps the PNGs
  and reverts `shoot.mjs` silently un-checks those rows.**
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip (~`:1056`), plus a
  short annotation on the sibling `sync-rxdb-conflict-notice-ui` (~`:1083`) recording that the
  **revised** plates now exist. **That annotation is NOT a status change** — the parent stays
  ATTENDED-BLOCKED; only the operator can move it. Every card tonight edits its own region of this
  file; conflicts are per-card and **both sides should be kept**.
- `.night-crew/runs/2026-07-29-2-autonomous/merge-intent-c1-sync-rxdb-conflict-notice-mockup-amendments.md`
  — this note. New file (and, on this branch, a new run directory). Unique to this card. No conflict
  surface. This is the **only** file this card touches outside `.planning/`, and it is expected: the
  full-suite clause is triggered by *production* code outside `.planning/`, of which there is none.

**Files NOT touched — assert this against any merge that shows otherwise:**

- `workflows.html`, `sync.js`, `ptr.js`, `index.html`, any other root HTML/JS — **NOT touched.**
  This card is the mockup, not the UI. A diff attributed to this card touching any of them is wrong;
  drop it.
- `sw.js` / `version.json` — **NOT touched, not regenerated.** No shipped HTML/JS file changed, so
  `node build-sw.js` has nothing to do, and `.planning/**` is in `build-sw.js`'s `globIgnores`
  (`build-sw.js:115`) so a tracked `.html` under `.planning/` cannot enter the precache manifest.
- `package.json` / `package-lock.json` — **NOT touched.** No dependency, no script, no version bump.
  Playwright is read from the main clone's already-installed `node_modules` through a **gitignored**
  symlink, removed after the render.
- `backend/internal/version/version.go` — **NOT touched.** No shipped file changed on either side,
  so **neither** constant moves. `Backend` stays `0.3.0`, ~~`Frontend` stays `1.2.1`~~ **`Frontend`
  stays `1.2.2` — STRUCK at the repair round: `1.2.1` was never what the tree held. The claim that
  matters (nothing moved) is true and verified by an empty `git diff` on both paths; the literal was
  wrong, and the same wrong literal had been copied into `done_when:` row 20.**
- `backend/` generally, `tests/`, `Taskfile.yml`, `docker-compose*.yml`, `.claude/` — **NOT touched.**
- **`.night-crew/knowledge/ledger.md`** — **NOT touched.** Decisions 80/82/91 are the record this
  card executes against; an implementer editing the ledger would be writing its own grade.
- `.night-crew/runs/2026-07-29-2-autonomous/timings.log`, `HANDOFF.md`, `DECISIONS-NEEDED.md` —
  **NOT touched.** The orchestrator owns those (established on the 2026-07-29 run: every line in
  `timings.log` is an orchestrator-written dispatch/done pair).

## What must survive any merge

1. **The two 🛑 AMENDMENT blocks in `UI-SPEC.md`, verbatim.** They are operator-directed and they
   are the contract. The plates are the *answer*; the blocks are the *question*. A merge that keeps
   the redrawn plates and drops the blocks leaves a future reader unable to tell whether the design
   was amended or drifted.
2. **A-1 unbundled, not re-bundled.** Two facts that the original design fused must stay apart:
   *rows never leave the sheet except on Dismiss or expiry* (so **Undo survives** — this is
   unchanged and must not be "simplified" into a decrementing queue), and *the banner prints two
   figures*. A merge that restores the single-figure banner reinstates the exact defect decision 82
   was filed against.
3. **A failed restore counts as still-to-review.** Not "handled", not "not green". It is the one
   place the still-to-review definition can be got wrong silently, and the error plate is drawn so
   the arithmetic can be checked from a screenshot.
4. **A-2's confirm step, showing the N server values about to be overwritten.** The batch override
   writes N of someone else's answers on one tap. The confirm is the whole amendment; keeping the
   reworded button and dropping the confirm keeps the label and drops the protection.
5. **Attribution + timestamp parity in the collapsed view.** The collapsed batch view is the
   *riskiest* action and carried the *least* information. A merge that drops the timestamps
   reinstates that inversion.
6. **Both readings of open decision (i), drawn side by side, with neither selected.** The card
   requires the choice to be visibly decidable. A merge that keeps one plate and drops the other
   silently settles an operator decision.
7. **The retention number as a visible placeholder.** Decision 80 accepted 30 days; triage reopened
   it. A merge that renders a bare "30 days" turns a reopened question back into a settled fact.
8. **`shoot.mjs`'s non-zero exit.** Overflow, touch targets, banner-line truncation and the
   two-figure banner are all checks that a screenshot alone does not settle.
9. **Zero production code**, and **`sync-rxdb-conflict-notice-ui` still ATTENDED-BLOCKED.**

## What is safe to drop

- **The visual design itself** — colours, exact copy, whether still-to-review sits on its own line
  or after a `·`. UI-SPEC A-1 explicitly leaves the wording to the UI card; what is not optional is
  that both numbers are carried. A *no* from the operator remains a successful outcome.
- **The screenshots.** Evidence, not artifact; regenerable from `mockup.html` at any time.
  (`shoot.mjs` itself is **not** in this bullet — see item 8 above.)
- **The roadmap card's prose.** The status flip matters; the wording does not.
- **Anything in this note itself.**

## Not done, deliberately

- **No production code, of any kind.** No `conflict$` subscription, no `conflictHandler`, no
  `workflows.html` change, no RxDB client.
- **No sign-off is claimed, and no block is lifted.** This card produces the revised artifact; the
  human *"ok, build this"* is the operator's and has not happened.
- **Neither open decision is settled.** (i) removed-field row in the chip base vs `+N` — **both**
  readings drawn, neither chosen. (ii) retention window — drawn as a placeholder token, not a value.
  An implementer choosing either would be making an operator's call.
- **No confirm added to the single-row restore.** A-2 rules that out explicitly: Undo is the safety
  net there and a confirm on every tap is friction on a phone in a hurry.
- **No `openspec/` scaffolding.** Preflight verdict for this repo is **ABSENT**. Universal
  per-change discipline only: atomic commits, the `Night-Crew-Card:` trailer as **one adjacent
  paragraph** (B-21, ledger decision 86 — verified after each commit with
  `git log -1 --format=%B | git interpret-trailers --parse`), and the roadmap flip in the same
  change set.
- **No full Playwright suite.** Zero production code, so the SW and the suite do not move — the card
  states this explicitly. If a file outside `.planning/` other than this note is touched, the suite
  is owed and this section must be struck.
- **No `tests/states-*.spec.js`.** That spec is the *UI* card's self-verification against shipped
  code; `tests/` is outside this card's footprint. `shoot.mjs` is the render+measure harness here.

## Red-first

**Applies, and is not omitted with a shrug this time.** The 2026-07-29 note recorded "red-first DOES
NOT APPLY — an HTML mockup has no behaviour". That was true when the only checks were eyeball
contracts; it is **no longer true**, because `shoot.mjs` now holds machine checks. The three new
measurements (every banner carries two figures; no banner line truncates at 480px; the batch
override names what it replaces) are run against the **un-amended** mockup first and must FAIL, and
that failing output is captured in the commit that adds them — before a single plate is redrawn.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **UNTOUCHED.** No dependency, no script, no
   version bump. Playwright is read from the main clone's existing install via a gitignored symlink;
   nothing is installed into this worktree. This is the environment shared with concurrent cards.
2. **`backend/go.mod`** — **UNTOUCHED.** No Go file is read or written by this card. `go` is not
   invoked.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** No container is brought up, torn down, or dialled.
   The mockup renders from `file://`.
4. **Root `Taskfile.yml`** — **UNTOUCHED.** No task added, no var default changed. `task sw`,
   `task test` and `task prod:deploy` are all deliberately NOT run: nothing shipped changed.

### Late additions

_(appended only if implementation forces a file outside the list above; per B-11 the WHOLE note is
re-read at close-out and contradicted lines are **struck**, not merely appended to)_

**Closed out.** The note was **re-read in full**, not appended to blindly. **The footprint held
exactly as declared: no file outside the list above was edited, and no file was added to it.**
~~Three commits~~ **— four at this point, and more after the repair round below; see the branch
log** — each carrying the `Night-Crew-Card:` trailer as one adjacent paragraph, each verified after
the fact with `git log -1 --format=%B | git interpret-trailers --parse`. ~~`git diff --name-only
9bd9a72..HEAD` lists 38 paths~~ **— STRUCK at the repair round: at this close-out (`ab34f41`) the
count was **37**, not 38 (32 PNGs + `mockup.html` + `UI-SPEC.md` + `shoot.mjs` + `roadmap.md` + this
note). At the repair round's HEAD it is **35**: `loading-light.png` and `loading-dark.png`
re-rendered back to bytes identical to the branch point, because the reflow that shifted every plate
by one device pixel happened to shift that one back into its pre-r2 alignment. Both numbers verified
with `git diff --name-only 9bd9a72..<rev> | wc -l`. The footprint claim is unchanged and still
holds; only the arithmetic was wrong** — all under
`.planning/phases/sync-rxdb-conflict-notice/`, plus `.night-crew/knowledge/roadmap.md` and this
note. No production file, no version constant, no `sw.js`.

**One line is struck above**, in the shared-files list, and because reality was *broader* than the
guess, not different in kind: the A-2 machine check covers every "Restore" control, not just the
batch button.

**Five things a merge should know that the note did not anticipate:**

1. **`.planning/` is in `.gitignore`, so the phase directory needs `git add -f`** — including the
   ten new PNGs. This is the existing precedent, not a workaround; every tracked file under
   `.planning/` got there the same way. A merge that "helpfully" drops these as ignored artifacts
   destroys the card's deliverable. (The r1 note recorded this; this note should have carried it
   forward and did not.)
2. **Two design rules were forced out into the open by the amendments and must survive** — they are
   additions to the collapse and batch behaviour, not restatements. **(a) `Restore all N of mine`
   acts only on rows still to review**, never the chip base: a batch tap must not silently reverse a
   deliberate *Keep theirs*. **(b) Collapse hides the Restore/Keep pair but never a row's outcome
   strip or its Undo**: after a batch restore all N rows are green on a collapsed sheet, so a
   collapse that ate Undo would make a *batched* mis-tap irreversible. Both are stated in
   `UI-SPEC.md` §"The counting rule" as rules 7 and 8 and drawn on the `a1-banner` plate. Dropping
   either re-opens a hole A-2 exists to close.
3. **The `edge-removed` plate now discloses that its chip is Reading A of open decision (i).** It
   was silently embodying one side of a question this card is required to leave open — a choice made
   by omission. The caption is load-bearing, not commentary; a merge that drops it re-settles the
   decision.
4. **Red-first was strengthened into mutation testing.** The note promised the three new checks
   would be run red first, and they were (5 banners carrying one figure, 7 Restore controls silent,
   exit 1). What it did not anticipate is that a check passing red is not proof it can fail —
   measurement 4 (banner truncation) passed on r1 because there was nothing yet to truncate. All
   three were therefore mutation-tested against the finished mockup (inject `nowrap`+ellipsis → 24
   lines flagged; delete every `.cn-banner-open` → 8 banners flagged; strip every sub-label → 10
   controls flagged). The mutation script was **not** committed — it is a throwaway probe, and its
   results are recorded in the r2 commit message and in `UI-SPEC.md`'s `HOW IT FAILS` clauses.
5. **Two defects were found by reading the renders back, which is the ritual working as intended
   and is worth recording as such.** The open-decision captions did not say "NOT SETTLED" inside the
   plate itself (the framing block sits between plates and so is absent from every screenshot), and
   the U+1F6D1 marker rendered as a **tofu box** in the headless font stack. Neither is visible from
   the source; both were repaired and re-shot. `done_when:` row 17 now names the tofu case
   explicitly so the next revision cannot re-introduce it silently.

**Nothing from this round contradicts the four HARD-constraints attestation** — `package.json`,
`package-lock.json`, `backend/go.mod`, `docker-compose.nc.yml` and `Taskfile.yml` remain untouched,
and no version constant moved (`Backend` 0.3.0, ~~`Frontend` 1.2.1~~ **`Frontend` 1.2.2 — struck
above**; `git diff` on those paths is
empty). Playwright was read from the main clone's existing install; the temporary `node_modules`
symlink used to resolve it is gitignored and was removed after the final render.

**The card's own boundary held.** `sync-rxdb-conflict-notice-ui` is still **ATTENDED-BLOCKED** and
`UI-SPEC.md` still says nothing is approved. No sign-off was claimed, no block lifted, and neither
open decision was settled. **The restricted-input verifier gate is still owed before any SUMMARY.md
is written** — this card wrote none, and the orchestrator holds that gate.

---

## Repair round — close-out (same run, after two independent gates)

**The whole note was re-read again under B-11, not appended to. Four lines above are struck** — the
A-2 check's scope (twice now), `Frontend 1.2.1` in two places, and the "38 paths" count. Each strike
names what replaced it and why. **The footprint did NOT widen: the same six declared paths.**
`git diff --name-only 9bd9a72..HEAD` lists **35** files at the repair round's HEAD (37 at the
previous close-out — `loading-*.png` re-rendered identical to the branch point; see the strike
above). No production file, no version constant, no `sw.js`, nothing new added to the list.

**The plates survived; the criteria did not.** Both gates independently confirmed the red commit is
honest, the three claimed mutations reproduce, all four trailers parse, all 32 PNGs re-render, and
nothing truncates, clips, overlaps or tofus in either scheme. No PARK was owed and none is taken.
What they found was that ~~four checks~~ **— STRUCK at the hardening round: SIX. A third gate found
two more of exactly the same kind (m2 tap targets, m4 banner lines), which this round's two gates did
not; see the hardening round below** — checks could not fail in the way they claimed to, plus one design hole
and two fixture defects. All are repaired **and each repair was falsified by re-running the
mutation**, because a repair to a falsifiability defect that is not itself falsified is not a repair.

1. **`done_when:` 32 was scoped to the file the fix was made in.** `mockup.html` got the `⟨30⟩`
   placeholder; `UI-SPEC.md`'s State Enumeration `empty` row went on printing *"Nothing recorded in
   the last 30 days"* as settled prose, **byte-identical to the base**, while row 32 named that exact
   string as its own counter-example. Table row fixed; row 32 now greps this spec too.
2. **`done_when:` 26 selected by LABEL, not behaviour.** `/^Restore/` matched neither `Retry` nor
   `Restoring…` on `plate-error`, and the whole guard evaporated under a rename while reporting
   success. Re-scoped by what the control does, with a population floor. **13 destructive controls,
   not 10.**
3. **`done_when:` 21 could not detect a DELETED banner.** Count now pinned at 8.
4. **`done_when:` 22 checked presence, not arithmetic.** New measurement 6 reconciles headline /
   still-to-review / handled / `+N` / chip base against the rows actually drawn, per plate, in both
   schemes — **without settling open decision (i)**: a removed-field row may sit in either bucket and
   the plate must balance under exactly one reading, which the script derives and names.
5. **`done_when:` 20 carried a false literal** (`1.2.1`; the tree reads `1.2.2`). Corrected here and
   in the spec.
6. **A design hole: collapse removed the only exit from an unidentifiable row.** Counting rule 8
   protected the outcome strip and Undo under collapse and was silent about the row that has
   neither, so `a1-banner` drew two *"A change we couldn't identify"* rows with **no actions at
   all** — and `Dismiss` is the only way such a row ever leaves the sheet. **Resolved in favour of
   keeping the actions**, which is rule 8's own argument rather than a new one: collapse hides the
   Restore/Keep pair, and those rows have no such pair to hide. Rule 8 extended, plate redrawn,
   `done_when:` 35 added, measurement 7 added. **This is a design change to an artifact the operator
   has not signed — it is drawn, not decided, and it is listed here so the operator can reject it
   with the rest of revision 2.**
7. **Fixture drift repaired.** *Sanitizer concentration* on `Opening — Truck A / sub_9f31c4` read
   `6:12 PM` on `outcomes` and `6:13 PM` on `a2-confirm` and `edge-many` — one document, one
   question, one author, two times, on the surface whose whole thesis is that the numbers agree.
   `6:13 PM` is correct; `outcomes` was the outlier.
8. **One reported nit did NOT reproduce.** A stray `U+FE0F` was said to have been left behind when
   the tofu'd `U+1F6D1` was replaced. **There is none.** Every variation selector in every file this
   card touches is attached to a valid emoji base (`U+26A0 FE0F` on the eight banner icons,
   `U+1F58A FE0F` on one roadmap heading); the three open-decision captions carry a bare `U+25B2`
   with no selector. Checked codepoint-by-codepoint across all five files. Recorded rather than
   silently dropped.

**What a merge must additionally know after this round:**

- **`shoot.mjs` now holds SEVEN measurements, and four of them are the repair** — **and after the
  hardening round SIX of the seven pin their population; only m1 (page overflow) is not a population
  walk and has nothing to pin.** Reverting it to the
  r2 version does not just lose coverage — it restores checks that pass on a mutated file. The file
  carries a header block naming the generic failure mode (*a check scoped to the place a fix was
  made is the same escape as a criterion scoped to the members that already pass*); keep it.
- **Counting rule 8 has a third clause.** A row with no Restore/Keep pair keeps its actions under
  collapse. Dropping it re-opens a hole with no exit.
- **`roadmap.md`'s pre-existing r1 sign-off block is now struck where it contradicted itself.** It
  stated in the imperative that the card *"is no longer ATTENDED-BLOCKED and may enter a slate"* and
  that the **pre-amendment** counting rule was *"settled"* and must be implemented *"as drawn"* —
  nine lines below the 🛑 block saying the opposite. Four claims are struck with reasons; **(b)
  handled rows stay on the sheet** is explicitly marked as SURVIVING, because A-1 depends on it.
  This completes a supersession the card had already started; it does not widen the footprint.
- **Nothing was lifted, discharged or settled.** `sync-rxdb-conflict-notice-ui` is still
  **ATTENDED-BLOCKED**, `UI-SPEC.md` still says nothing is approved, and both open decisions are
  still open and still drawn as decidable.

---

## Hardening round — close-out (same run, after the restricted-input verifier gate PASSED)

**The whole note was re-read a third time under B-11, not appended to. Three more lines above are
struck or qualified in place** — the inherited touch-target claim in the shared-files list, the
*"four checks could not fail"* count in the repair round's close-out, and the *"SEVEN measurements"*
bullet. Each names what replaced it.

**The gate passed, and this round is not a re-audit of it.** A restricted-input verifier confirmed
all `done_when:` rows hold, `shoot.mjs` green at raw exit 0, no truncation or tofu at 480 px in
either scheme, both open decisions open, the block on. Four defects were fixed and nothing else was
touched.

1. **The repair round pinned three populations and left two unpinned — the same defect, one round
   later.** `m3`, `m5` and `m7` got `EXPECTED_BANNERS` / `EXPECTED_DESTRUCTIVE*` / `EXPECTED_UNREC_ROWS`;
   `m2` (tap targets) and `m4` (banner lines) did not, and both are population walks. **Deleting every
   `<span class="cf-done-undo">Undo</span>` — the only escape from a mis-tapped Restore, and the
   control `done_when:` 18 exists for — turned the whole suite green:** `58 measured, 0 under 44px ->
   PASS` in both schemes, `self-verification PASS`, raw exit 0. Row 18's own text legitimised `58`,
   so the printed number did not indict it either. Deleting a `.cn-banner-sub` cause line did the same
   to `m4` (24 -> 23, PASS). **Both are now floored (`EXPECTED_TAP_TARGETS` 62, `EXPECTED_BANNER_LINES`
   24) and both floors were falsified by re-running the mutation** — 58/62 and 23/24, red, exit 1.
   Rows 18 and 24 were rewritten to name the **pinned** count rather than bless the observed one.
2. **`done_when:` 18 and 24 were themselves part of the escape.** Row 18 recited *"prints the count
   measured (62 after the repair round, 58 at r2, up from 38)"* — a criterion that recites the count
   it happened to observe cannot indict the count that shrank, and `58` was literally listed in it as
   an acceptable value. Row 24 read the same way about `24`. **No new criteria were added; the two
   existing rows were corrected.**
3. **Decision citations were internally inconsistent, on the page the operator reads at sign-off.**
   `UI-SPEC.md` cited A-1 as *decision 80* and A-2 as *decision 81*, while its own revision table, the
   `done_when:` section headers and `roadmap.md` all attribute **both** amendments to **decision 82**.
   Corrected to 82 in both amendment blocks, each now stating explicitly that **decision 80 is the
   18:12 r1 sign-off, superseded in part**. No decision was re-decided; a citation was.
4. **`roadmap.md` said `UI-SPEC.md` carries 34 `done_when:` rows. It carries 35** — the line was
   written before the repair round added row 35. Corrected; verified by count, not by memory.

**Footprint did NOT widen, and the file count did not move.** `git diff --name-only 9bd9a72..HEAD`
still lists **35** files: the same six declared paths, three of them touched this round
(`shoot.mjs`, `UI-SPEC.md`, `roadmap.md`) and this note. **All 32 PNGs re-rendered byte-identical** —
no plate content, copy, fixture or rule changed, so nothing shifted a pixel; `git status` showed no
PNG. A-1's four-line banner was re-read back from `a1-banner-light.png` and still holds
(headline / `2 still to review · 2 handled` / `+ 2 changes we couldn't identify` / the cause line,
none clipped). No production file, no version constant, no `sw.js`.

**The card's boundary held again.** Nothing was lifted, discharged or settled:
`sync-rxdb-conflict-notice-ui` is still **ATTENDED-BLOCKED**, `UI-SPEC.md` still says nothing is
approved, and both open decisions — (i) the removed-field row's bucket and (ii) the retention window —
are still open and still drawn as decidable.
