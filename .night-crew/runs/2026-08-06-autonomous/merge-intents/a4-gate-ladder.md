# Merge intent — A4 `gate-ladder-completeness`

Run `20260806` · branch `card/a4-gate-ladder` · based at `d1bc051` (tip of
`overnight-20260806` at worktree creation).

**Documentation / audit WO. No code change anywhere in the tree.**

## Red-first

**n/a — no code change.**

Documentation/audit WO: the deliverable is prose in
`.night-crew/knowledge/reference/gate-ladder.md` plus three status lines in
`.night-crew/knowledge/BACKLOG.md`. Nothing under `backend/`, `sync-rxdb/`, `tests/`, and
no shipped `*.html` / `sw.js` / `package.json` / `version.go`, so there is no Go or
Playwright spec that could have been written red first. The slate names A4 as *"the one
card tonight that records `n/a`."*

Scoped to **this card's own commits** (`git diff --name-only d1bc051..HEAD`), not
`dev...HEAD` — everything else visible from this branch arrives via the W0 and C1 merges
already in `overnight-20260806`'s ancestry.

## Files owned by this card

- `.night-crew/knowledge/reference/gate-ladder.md` — **sole owner tonight** (slate
  §A4 footprint). Three edits:
  1. **(a)** a new `### G5 — there is none` subsection under §"The gates", recording G5
     as retired-never-defined per ledger decision 101, so no future slate inherits the
     gap as an open question.
  2. **(b)** a new §"G4's morning-triage discipline greps are VACUOUS here (B-14)"
     recording — **not fixing** — the vacuous greps, and naming the night-crew-clone
     destination for the remedy.
  3. **F7** — the G2 (Go) row is rewritten so the 59-subtest count reads as *asserted by
     the suite*, not eyeballed from a `-v` log, and the evidence line now requires
     stating **`HQ_SYNC_GATE_CHILD` unset** alongside `HQ_SYNC_SUBSTRATE_OPTIONAL` unset.

- `.night-crew/knowledge/BACKLOG.md` — declared in the slate's A4 footprint. Status
  lines only; no entry's body, lead or provenance is rewritten.

## What MUST survive any merge

1. **The G2 (Go) row's `HQ_SYNC_GATE_CHILD` requirement.** One leaked env var disarms
   *both* A1's count assertion and B-36's exit-code pin while `internal/sync` still
   prints `ok`. That is B-36's own defect class recurring one layer up, and the ladder is
   the belt to A1's braces. If a merge has to choose, keep this line over any prose
   around it.
2. **The G5 subsection.** Its whole purpose is that the gap reads as *history* rather
   than a hole. Dropping it re-opens B-26's residual and the next slate inherits the same
   silent question.
3. **The B-14 section's clone-side destination.** The entry is worthless without the
   sentence naming where the remedy lives, because the one thing an hq run branch may
   never do is apply it here.
4. **B-22's closure line**, which belongs to **card A2**, not to this card — see below.

## What is safe to drop

- Wording, ordering and emphasis anywhere in `gate-ladder.md`. Nothing in this card's
  diff is load-bearing for any *command*; the file is read by humans and by slate
  authors, and no tooling parses it.
- The parenthetical cross-references (ledger decision numbers, B-numbers, run ids). They
  are provenance, useful but reconstructible.

## Shared files touched outside the footprint

**Nothing here.** Both files this card writes are named in its slate footprint.

## 🛑 Declared: B-22's closure is card A2's fact, written by this card

The slate gives **A4** the `BACKLOG.md` status lines, so card **A2**
(`gate-harness-check-b-per-package`) deliberately did not touch `BACKLOG.md` when it
landed its per-package Check B tonight. B-22 therefore still read open. This card closes
it on A2's behalf, attributing the fix to A2.

**Merge consequence:** if A2's branch does not merge, **B-22's closure line here is
false** and must be reverted. It is the one line in this card's diff whose truth depends
on another card's branch landing.

## 🛑 Declared: the G2 (Go) row now describes tests that live on card A1's branch

`TestRowVisibilitySubtestCount_Structural` / `_Executed` and the constant
`wantRowVisibilitySubtests = 59` are on `card/a1-rls-count-assert` (`2b3a50f`), **not** on
`overnight-20260806` at the time this card was written. The ladder row is written to
describe the post-merge world.

**Merge consequence:** merge **A1 before A4**, or the ladder cites assertions the tree
does not yet carry. If A1 does not merge at all, the G2 (Go) row's "asserted by the suite"
sentence must revert to the eyeball-the-`-v`-log instruction it replaced. The
`HQ_SYNC_GATE_CHILD` half survives either way — the variable exists on A1's branch, and a
ladder line requiring that its absence be *stated* costs nothing if it is never set.

## Gates

- **G1** — **run, green.** From `backend/` (the module root, per the ladder's own 🛑):
  `go build ./...` **EXIT=0**, `go vet ./...` **EXIT=0**, both logs captured whole and
  both empty. Sanity only — this card changes no Go. Nothing piped through `tail` (B-93);
  the exit codes come from the commands.
- **G2 (Go / Playwright)** — **not run.** No code change, and the slate says sanity run
  only; the suite mutex is contended by two other legs.
- **G3** — **N/A.** `openspec: absent` (ledger §T-34 decision 140). No scaffolding
  created.
- **G4** — **not run, and correctly so.** Markdown under `.night-crew/` is **not a
  precached asset** — `sw.js`'s manifest names shipped frontend files only. Running
  `build-sw.js` here would regenerate an unchanged file and prove nothing.
