# Conflict log — run `20260803`

Every merge onto `overnight-20260803` gets an entry, **clean or conflicted** (§15ad.66). A one-line
entry for a clean merge is the point: an empty log must never be readable as "no conflicts" when it
means "the logging never ran."

Dispatch was **SERIAL** (operator's choice at sign-off), so cards never developed concurrently and
the structural opportunity for a collision was small by construction — but that is a reason the log
is short, not a reason to skip it.

Entries are appended **after** each merge lands, never before.

---

## Merge 1 — S1a `sync-cutover-list-scope` → `overnight-20260803`

- **When:** 2026-08-02 ~11:32, after G6 PASS
- **Cards involved:** S1a only. Nothing else had been cut; `overnight-20260803` was still at the
  branch point (`60b9edb`).
- **Result:** 🟢 **CLEAN — no conflicts, no hunks resolved by hand.**
- **Files:** 9 changed, +1397/−45 — `sync-rxdb/client.js`, `sync-schema/sql/0004_write_policies.sql`
  (comments only), `backend/internal/sync/rowvisibility_rls_test.go`,
  `tests/sync-rxdb-client.spec.js`, `sw.js`, plus run/knowledge artifacts (`BACKLOG.md`,
  `roadmap.md`, the design note, the merge-intent).
- **Intents read:** `merge-intent-s1a-sync-cutover-list-scope.md` only — there was no second intent
  to weigh it against. Its "must survive" set (the list scope, the date-floor guard, the three-
  collection Realtime filter, W17) is intact in the merge commit; nothing was dropped.
- **Gate after the merge:** G1 `go build ./...` 0 · `go vet ./...` 0 (run from `backend/`, the module
  root — the first attempt was run from the repo root, where `./...` matches no module, and a pipe
  masked the error; re-run correctly). G4 `node build-sw.js` → **31 files precached**, tree clean on
  the rebuild ⇒ idempotent, version parity 1.4.0 three-way. Regenerated **after** the merge commit
  per B-37.

---

## Merge 2 — S1b `sync-hard-cutover` → `overnight-20260803` (**PARK artifacts only**)

- **When:** 2026-08-02 ~11:58
- **Cards involved:** S1b only. It was cut *after* S1a merged, so it developed against the merged
  scope model — B-50 makes that ordering mandatory, not stylistic.
- **Result:** 🟢 **CLEAN — no conflicts.** Nothing to resolve, because the card **parked before
  writing any production code**: `git diff overnight-20260803 card/s1b-sync-hard-cutover -- . ':!.night-crew'`
  is **empty**. What merged is the park write-up, `DECISIONS-NEEDED.md`, the merge-intent, the
  B-65..B-70 backlog entries and the roadmap flip.
- **Files:** 5 changed, +496/−2 — all under `.night-crew/`.
- **Intents read:** `merge-intent-s1b-sync-hard-cutover.md` against S1a's already-merged intent. The
  one real interaction is `sync-rxdb/client.js` — S1a **owns** it, S1b only **reads** it — and S1b's
  diff does not touch it, so the ownership held and there was nothing to resolve against intent.
- **Gate after the merge:** not re-run at this point. The merge introduced no code, so the tree's
  gate state is unchanged from merge 1; the final-gate section of `HANDOFF.md` carries the evidence
  taken on the fully merged tree.

---

## Merge 3 — P6 `period-summary-contract-notice` → `overnight-20260803`

- **When:** 2026-08-02 ~13:58, after two G6 rounds and two fix rounds
- **Cards involved:** P6 only. Its footprint (`docs/contracts/**`, the two contract documents) is
  disjoint from both S1 cards by construction — that disjointness is why it was the only
  Track-B-eligible card in the slate.
- **Result:** 🟢 **CLEAN — no conflicts, no hunks resolved by hand.**
- **Files:** 6 changed, +786/−94 — both contract documents, the new UNSENT notice, plus
  `BACKLOG.md`, `roadmap.md` and the merge-intent. `backend/` **byte-unchanged** (verified: `git diff
  overnight-20260803 --name-only -- backend/` returns 0 files).
- **Intents read:** `merge-intent-p6-period-summary-contract-notice.md` against both merged S1
  intents. No overlap in either direction — no shared file, no shared package.
- **One resolution taken by the orchestrator, and it is worth naming:** P6's second fix round
  corrected `"fourteen months"` → `"eight weeks"` repo-wide, which swept up
  `reference/slate-20260803.md:331` — a **signed** artifact. I reverted that single line. The slate
  records what the operator signed on 2026-08-02, and "fourteen months" is what was believed then;
  the audit is precisely what *discovered* the figure was wrong. Backdating a correction into a
  signed plan of record would destroy the evidence that the discovery happened. The corrected figure
  stands everywhere it is a live claim.
- **Gate after the merge:** see `HANDOFF.md`'s final-gate section — G1/G2/G4 taken on the fully
  merged tree by the orchestrator, not inherited from card reports.

---

## Summary

**3 merges, 3 clean, 0 conflicted hunks, 0 cards parked for an unresolvable intent collision.**

The honest read: **serial dispatch plus a footprint split that gave each card sole ownership of its
files is what produced this** — it is not evidence that the collision risk was low. The slate fanned
`sync-hard-cutover` into S1a + S1b specifically so the scope model was proven and merged before the
write path was swapped, and S1b then developed on top of S1a rather than beside it. That arrangement
leaves nothing to conflict over by construction.

It should also be said plainly that **merge 2 was clean because the card parked**, not because two
overlapping cards were reconciled well. A night where S1b had implemented its full footprint —
`workflows.html`, four deleted Go files, `build-sw.js`, `sw.js` — would have been the real test of
this log, and that test did not happen.
