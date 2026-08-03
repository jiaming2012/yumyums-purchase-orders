# Merge intent — A4 `offline-ownership-design-note`

- **Run:** `overnight-20260804`
- **Branch:** `card/a4-offline-ownership-design-note`
- **Base commit:** `2041477` — every diff claim in this note is measured against
  `2041477`, never against run-branch HEAD (T-32 decision 130a).
- **Closes:** reworded **E-KR3** (`okr-completion-plan-20260804.md` §4).
- **Specified content:** `okr-completion-plan-20260804.md` §3 A4, lines 143–261.

Written BEFORE the deliverable (§15ad.65). Updated in place only for facts that changed.

---

## Red-first

n/a — no code change

*(Q-KR2 as reworded in `okr-completion-plan-20260804.md` §4, ledger T-33 decision 132. The
deliverable is a design note; the section is written explicitly so that "absent" and "not
applicable" are distinguishable at triage. No synthetic red was invented to fill it.)*

---

## Shared files touched — each line says why

| File | Why this card touches it |
|---|---|
| `.night-crew/knowledge/designs/offline-ownership.md` | **New. The card's own deliverable.** The single written answer to "when a phone is offline, who owns each piece of what is on it." Nothing else in the repo carries this. |
| `.night-crew/knowledge/roadmap.md` | The card's status flip, required in the same change set. |
| `.night-crew/runs/2026-08-04-autonomous/merge-intents/` | This note. |

**Outside the slate's stated footprint:** nothing here. The slate's footprint is
`designs/offline-ownership.md` + `roadmap.md`; the merge-intent note is universal mechanics.

**Explicitly NOT touched — this is a docs-only card.** No `backend/**`, no `*.html`, no `*.js`,
no `sw.js`, no `build-sw.js`, no `package.json`, no `backend/internal/version/version.go`, no
`night-crew.toml`, no `tests/**`. `git diff 2041477 --stat` names only files under
`.night-crew/`. **Re-verified at landing** rather than asserted — the stat output is quoted in
the card report.

🛑 **`sw.js` is byte-identical to base.** `node build-sw.js` was run twice as the G4 sanity leg
and produced no diff either time; neither file this card touches is precached, so the precache
count holding at **31** is the expected result and a move in it would have been a finding.

---

## What must survive any merge

1. **The count.** The note publishes **8 classes across 6 named stores**. E-KR3's own
   parenthetical names two (*"static assets → Workbox, checklist data → RxDB"*); the count *is*
   the finding. A merge that collapses rows back toward two destroys the deliverable.
2. **Both splits.**
   - `hq_offline_v1` is **two** classes — `submitQueue` (#6) and `syncMeta` (#7) — because they
     have **different fates**: retire the op-log and `syncMeta` is dead weight while
     `submitQueue` survives untouched. A row reading "`hq_offline_v1` → `sync.js`" conceals that.
   - `api-cache` is **two** classes — #2 (non-replicated apps) and #3 (checklist responses) —
     split by whether RxDB replicates the underlying rows. #2 is uncontested forever; #3 is the
     only class that can ever become dual-owned.
3. **Rules 1–4 stated as rules**, not as observations and not as prose. Rule 2 in particular is a
   **prohibition** a future implementer must hit: nothing may read from RxDB on a code path that
   can execute offline.
4. 🛑 **The class with no owner, in those words.** REST writes land in HQ's Postgres, RxDB push
   writes land in the substrate, **nothing reconciles them**. Suppressing this to make the table
   look clean is the laundering case the reworded E-KR3 exists to forbid.
5. 🛑 **Rule 1's two prohibitions.** The note must not claim RxDB "handles offline" generally and
   must not describe Workbox as transitional. Both are permanent, and the mechanism is at source:
   Workbox precaches `vendor/rxdb.bundle.js`, so RxDB cannot bootstrap itself.
6. **The target state cites §8 by name**, not "a future card". §8 is decision 126 option (i),
   which triage assessed as *"the honest end state but a milestone rather than a card."* The note
   **describes** that destination; it does not adopt it. That distinction must survive.
7. **The stated deviations.** Six places where measurement contradicted or refined
   `okr-completion-plan-20260804.md` §3 A4's table are recorded in the note's own Deviations
   section, each with the file:line that settles it. A merge that quietly drops them leaves the
   next reader unable to tell a verified row from a patched one.

---

## What is safe to drop

- Any wording, ordering or table formatting in `designs/offline-ownership.md`, provided items
  1–7 above survive in substance.
- The roadmap bullet's phrasing (not its existence).
- This note itself, after triage.

---

## What this card deliberately did NOT do

- **It did not decide the two-store architecture.** §8 is described and cited as the destination;
  adopting it is milestone-sized (decision 126 option (i)) and is an operator/roadmap call. The
  PARK trigger for "deciding a product fork" did not fire because the card describes rather than
  decides.
- **It did not run the full Playwright suite or the Go suite.** The diff is documentation under
  `.night-crew/`; neither suite can observe it. Gates run: **G4 only** (`node build-sw.js` twice,
  31 both times, clean tree both times, frontend 1.4.0 == `package.json` == `version.go`
  `Frontend`).
- **It did not create OpenSpec scaffolding.** Workflow preflight: `openspec: absent`. Which
  per-change discipline this repo adopts is operator question **B-105**, not this card's.
