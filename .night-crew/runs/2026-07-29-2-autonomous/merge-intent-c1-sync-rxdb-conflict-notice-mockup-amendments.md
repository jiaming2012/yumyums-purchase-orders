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
  regenerated and extended.** `shoot.mjs` already measures horizontal overflow and every interactive
  element's touch-target box, exiting non-zero on failure. This card adds measurements for the
  contracts A-1 and A-2 introduce that also cannot be judged by eye — that **every** banner carries
  **both** figures, that **no banner line is truncated at 480px** (A-1's PARK trigger, so it must be
  measured and not asserted), and ~~that the batch override names what it replaces~~ **STRUCK at
  close-out — the check landed BROADER than the guess: it covers every control whose label begins
  with "Restore" (10 of them), not just the two batch buttons. Narrowing it back to `.cg-all` would
  silently un-check the eight single-row restores.** **A merge that keeps the PNGs and reverts
  `shoot.mjs` silently un-checks those rows.**
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
  so **neither** constant moves. `Backend` stays `0.3.0`, `Frontend` stays `1.2.1`.
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
exactly as declared: no file outside the list above was edited, and no file was added to it.** Three
commits, each carrying the `Night-Crew-Card:` trailer as one adjacent paragraph, each verified after
the fact with `git log -1 --format=%B | git interpret-trailers --parse`. `git diff --name-only
9bd9a72..HEAD` lists 38 paths, all under
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
and no version constant moved (`Backend` 0.3.0, `Frontend` 1.2.1; `git diff` on those paths is
empty). Playwright was read from the main clone's existing install; the temporary `node_modules`
symlink used to resolve it is gitignored and was removed after the final render.

**The card's own boundary held.** `sync-rxdb-conflict-notice-ui` is still **ATTENDED-BLOCKED** and
`UI-SPEC.md` still says nothing is approved. No sign-off was claimed, no block lifted, and neither
open decision was settled. **The restricted-input verifier gate is still owed before any SUMMARY.md
is written** — this card wrote none, and the orchestrator holds that gate.
