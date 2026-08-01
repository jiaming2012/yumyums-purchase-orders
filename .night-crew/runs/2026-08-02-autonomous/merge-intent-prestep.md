# Merge intent — Pre-step `prestep-kr-artifacts`

Branch: `card/prestep-kr-artifacts` (cut from `overnight-20260802`)
Written BEFORE the work, as this leg's first commit. Required per DESIGN §15ad.65.

## This leg in one line

**Not a card. Zero product code.** Three key results (P-KR2, D-KR3, E-KR1) are unmet only because
artifacts were never updated to match decisions already made. The whole footprint is documentation
under `.night-crew/`. No `.html`, no `.js`, no `.go`, no `tests/`, no `sw.js`, no version constant.

## Shared files touched

- **`.night-crew/knowledge/BACKLOG.md`** — **ONE entry only.** The bullet beginning
  **`- **Cross-user checklist hydration divergence (approved-vs-rejected ghost state)**`**
  (`:591` at the branch point). Its trailing disposition `· new` is replaced with the ruling
  transcribed from ledger **T-24 decision 67**, and the sentence *"Needs a product ruling first…"*
  inside the entry is superseded in place. **Every other line of this file is untouched.** Other
  cards tonight will append discoveries and flip their own entries in this file — those are
  additive and elsewhere; resolve by **union**, and if any conflict lands on the hydration bullet,
  **mine is the disposition side** (the ledger transcription), theirs is anything else.
- **`.night-crew/knowledge/roadmap.md`** — **ONE table row only:** the
  *"Cross-user checklist hydration divergence…"* row of the `## Backlog dispositions this round`
  table (`:1770` at the branch point), which still reads *"left `new` — needs a product ruling…
  not resolved this round"* six days after the ruling landed. Nothing else in this file is mine.
  **Every card tonight flips its own card's status in this file, all far above `:1700`.** My edit
  is in the historical dispositions table at the file's tail — it should not collide with any card
  status flip at all. If it does, the collision is spurious: **keep both sides.**
- **`.night-crew/knowledge/reference/card-actuals.md`** — **append-only.** One new
  `## Cycle median` section at the end of the file computing D-KR3. Existing run tables are read,
  not edited. If a card tonight appends its own run row, resolve by **union** — append order does
  not matter.
- **`.night-crew/knowledge/designs/`** — **two NEW files**, one per superseded fetch-storm item.
  Net-new paths, unique to this leg. No conflict surface.
- **`.night-crew/runs/2026-08-02-autonomous/merge-intent-prestep.md`** — this note. New file,
  unique to this leg. No conflict surface.

**Files NOT touched — assert this against any merge that shows otherwise:**

- `workflows.html`, `sync.js`, `sync-rxdb/*`, `index.html`, `ptr.js`, any root HTML/JS — **NOT
  touched.** Zero product code. A diff attributed to this leg touching any of them is wrong; drop it.
- `sw.js` / `version.json` / `package.json` / `backend/internal/version/version.go` — **NOT
  touched, not regenerated.** No shipped file moved, so neither semver constant moves and
  `node build-sw.js` has nothing to do.
- `backend/`, `tests/`, `Taskfile.yml`, `docker-compose*.yml`, `.claude/`, `openspec/` — **NOT
  touched.** Preflight verdict for this repo is **`openspec: absent`**; no `openspec/` scaffolding
  is created, and none must be.
- **`.night-crew/knowledge/ledger.md`** — **NOT touched.** T-24 decision 67 is the record this leg
  transcribes *from*. A leg editing the ledger would be writing its own source.
- **`.night-crew/knowledge/okrs.md`** — **NOT touched.** This leg makes the KRs readable; it does
  not grade them. Grading is the close's.
- `.night-crew/runs/2026-08-02-autonomous/{HANDOFF.md,DECISIONS-NEEDED.md,timings.log}` — **NOT
  touched.** The orchestrator owns those.

## What must survive any merge

1. **The BACKLOG hydration entry must NOT read `· new` after this merges.** That string is the
   literal measure of P-KR2 (*"the entry's disposition text at the time `sync-hard-cutover`'s WO is
   dispatched"*). A merge that restores `· new` reddens the KR on text alone — which is the exact
   failure this pre-step exists to prevent.
2. **The transcribed ruling's three requirements, individually.** They are three, not one, and they
   are separable: (a) a rejected submission does NOT resurrect as current state — archived, visible
   as history; (b) a fresh 0/2 MUST accept clicks — the silent no-op is a bug, not intended
   behavior; (c) the convergence matrix still needs its missing asymmetric approved-for-A /
   rejected-for-B cell seeded. Dropping (b) or (c) would leave the entry claiming the item is
   settled while half the required work is unnamed.
3. **The `T-24 decision 67` citation itself.** The disposition is a transcription, not a judgment.
   Without the citation a future reader cannot tell which it was.
4. **The roadmap `:1770` row must not say "not resolved this round" after this merges.** Same
   reason as (1): the roadmap and the backlog are read by different people and a stale roadmap
   re-opens a settled question.
5. **The median's inputs, not just its number.** The `card-actuals.md` section names every card and
   duration it used and the basis it chose. A merge that keeps the headline figure and drops the
   input table turns an auditable number into an asserted one — which is what D-KR3 was unmet for.
6. **Both fetch-storm notes, as two files.** E-KR1's text is *"exactly 1 reviewed
   architectural-argument note per item."* Merging them into one file fails the KR on its own terms.

## What is safe to drop

- Wording, ordering, and formatting anywhere in this leg's output.
- The prose framing of the median section — the table of inputs and the stated basis are not in
  this bullet (see item 5).
- Anything in this note itself.

## Not done, deliberately

- **No product code, of any kind**, and therefore **no `task sw`, no `task test`, no deploy.**
- **No new card authored, and no scope widened.** Standing rule tonight: anything discovered routes
  to the NEXT milestone unless it reddens a named KR on the current OKR page. Discoveries are filed
  to `BACKLOG.md` with a handle and named in the report with a destination.
- **No ruling invented.** P-KR2's text is transcribed from `ledger.md` T-24 decision 67 and nothing
  else. If the ledger and the slate's summary of it had disagreed, the ledger would have won.
- **G1 / G2 / G4 are NOT run and are NOT claimed.** This leg touches no code; build, vet and suite
  gates have nothing to gate. Verification is `git diff` scope plus source-traceability of every
  claim written.

## Red-first

**Does not apply, and this is not a shrug.** There is no behaviour here to fail first: the leg's
entire output is transcription (P-KR2), arithmetic over an existing ledger (D-KR3), and argument
(E-KR1). E-KR1's own KR text is what makes the notes admissible *in place of* a test — no test is
constructible because `sync.js` is deleted by card S1 on Night B.

## Late additions

_(appended only if the work forces a file outside the list above; per B-11 the WHOLE note is
re-read at close-out and contradicted lines are **struck**, not merely appended to)_
