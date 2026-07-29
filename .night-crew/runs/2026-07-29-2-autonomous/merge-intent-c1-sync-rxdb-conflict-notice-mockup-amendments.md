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
  measured and not asserted), and that the batch override names what it replaces. **A merge that
  keeps the PNGs and reverts `shoot.mjs` silently un-checks those rows.**
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

**Nothing here yet** — written before implementation.
