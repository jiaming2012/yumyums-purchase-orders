# Merge intent — A4 `gate-ladder-completeness`

Run `20260806` · branch `card/a4-gate-ladder` · based at `d1bc051` (tip of
`overnight-20260806` at worktree creation).

**Documentation / audit WO. No code change anywhere in the tree.**

## Red-first

**n/a — no code change. Re-verified after the fix round.**

`git diff --name-only d1bc051..HEAD` returns **three paths, all `.md`**:
`.night-crew/knowledge/reference/gate-ladder.md`, `.night-crew/knowledge/BACKLOG.md`, and
this merge intent. Zero non-markdown files, so the `n/a` still holds on the post-fix diff
and is not a stale claim carried from the first pass.

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
  4. **Fix round** — every defect the G6 reviewer found was in **(3)**, the undeclared
     third deliverable. Five corrections, one commit each: **F-1** the gate-child section
     rewritten (it disarmed one half, not both; A1's token closed the silent door; and its
     own repro pattern had excluded the falsifying test), **F-2** the retired manual
     fallback now cites `9b63958`, **F-3** the grep failure quoted tool-agnostically,
     **F-4** the B-14 provenance table (four triages carried it, three wrote the marker),
     **F-5** an in-file open-question marker on the G3 contradiction. The **B-14 and B-26
     halves were reviewed accurate** and are untouched by the fix round.

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
5. **The `❓ OPEN QUESTION` block on G3** (added by the fix round, F-5). The completeness
   sentence next to it ratifies one of two conflicting G3 definitions — `N/A, openspec
   absent` (decision 140) vs `red-first re-verified by G6` (decision 101's recovered
   contract). Red-first is graded this run (Q-KR3). **Dropping the marker while keeping the
   sentence is worse than dropping both**, because it leaves the contradiction settled in
   the wrong direction and invisible. Routed to `DECISIONS-NEEDED.md`; operator call.

## What is safe to drop

- Wording, ordering and emphasis anywhere in `gate-ladder.md`. Nothing in this card's
  diff is load-bearing for any *command*; the file is read by humans and by slate
  authors, and no tooling parses it.
- The parenthetical cross-references (ledger decision numbers, B-numbers, run ids). They
  are provenance, useful but reconstructible.

## Shared files touched outside the footprint

**Nothing here.** Both files this card writes are named in its slate footprint.

## ✅ Both cross-card dependencies are SATISFIED — A1 and A2 have merged

This card was written against branches that had not yet landed, and declared two
merge-order caveats. **Both are now discharged. The record of them is kept; the caveats
themselves are withdrawn.**

**1. B-22's closure is card A2's fact, written by this card.** The slate gives **A4** the
`BACKLOG.md` status lines, so card **A2** (`gate-harness-check-b-per-package`) deliberately
did not touch `BACKLOG.md` when it landed its per-package Check B tonight. B-22 therefore
still read open, and this card closes it on A2's behalf, attributing the fix to A2.

> *Withdrawn caveat:* "if A2's branch does not merge, B-22's closure line here is false and
> must be reverted." **A2 merged as `b75ac53`** (`merge(a2): gate-harness-check-b-per-package
> — Q-KR2 closed, B-22 closed`). The closure line is now true of the tree.

**2. The G2 (Go) row describes tests that were on card A1's branch.**
`TestRowVisibilitySubtestCount_Structural` / `_Executed` and the constant
`wantRowVisibilitySubtests = 59` were on `card/a1-rls-count-assert`, not on
`overnight-20260806`, when this card was written. The row was written to describe the
post-merge world.

> *Withdrawn caveat:* "merge A1 before A4, or the ladder cites assertions the tree does not
> yet carry; if A1 does not merge at all, the row must revert to the eyeball-the-`-v`-log
> instruction." **A1 merged as `9b63958`.** The assertions are in the tree, the row cites
> that commit, and the retirement of the manual fallback is recorded as deliberate (F-2).

**A1's fix round also hardened the mechanism this card documents** — `HQ_SYNC_GATE_CHILD`
is now a parent-minted token rather than a `== "1"` flag, so an externally-set value
`t.Fatalf`s instead of silently skipping. The ladder's requirement to state it unset is
kept and re-justified rather than dropped (F-1): it now buys a leg *not discovering this by
a red at 3am*, which is a smaller but real claim than the one the section first made.

## Gates

- **G1** — **run, green.** From `backend/` (the module root, per the ladder's own 🛑):
  `go build ./...` **EXIT=0**, `go vet ./...` **EXIT=0**, both logs captured whole and
  both empty. Sanity only — this card changes no Go. Nothing piped through `tail` (B-93);
  the exit codes come from the commands.
- **G2 (Go / Playwright)** — **not run.** No code change, and the slate says sanity run
  only; the suite mutex is contended by two other legs.
- **G3** — **N/A** on the definition this repo's ladder table currently carries:
  `openspec: absent` (ledger §T-34 decision 140, `ledger.md:2697`). No scaffolding created.
  🛑 **Reported against a contested definition** — decision 101's recovered contract
  (`ledger.md:1932`) defines G3 as *"red-first re-verified by G6"* instead, under which
  this card's `## Red-first` section above **is** the G3 evidence and "N/A" is wrong. Both
  readings are satisfied by this card either way (no code ⇒ nothing to write red first, and
  no OpenSpec scaffolding), so nothing here turns on the ruling. Flagged because the
  ambiguity is live for cards that *do* change code. See F-5.
- **G4** — **not run, and correctly so.** Markdown under `.night-crew/` is **not a
  precached asset** — `sw.js`'s manifest names shipped frontend files only. Running
  `build-sw.js` here would regenerate an unchanged file and prove nothing.
