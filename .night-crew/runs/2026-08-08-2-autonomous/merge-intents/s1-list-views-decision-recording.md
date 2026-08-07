# Merge intent — S1 · `list-views-decision-recording`

Run `20260808-2`, Activity 4 (reshaped by T-43 from `activate-list-views-or-state-they-stay-rest`).
Branch `card/s1-list-views-decision-recording`, based on the run branch with C1+C2+C3 merged
(`09aaa0e`).

Written and committed **before** implementation, per §15ad.65.

## Scope

Record what ledger T-43 ruled, in the code that lies about it today — a **docs/banner recording
card**, not a build card. It does not implement anything C3 has not already implemented; it states,
in the comments a future reader actually meets, what C2/C3 shipped and what the operator ruled:

(a) **Approvals tab stays on re-fetch** (T-43a, partial resolution of B-43) — record it.
(b) 🛑 **My Checklists read path is OPEN** — T-43b, the operator's own deferral. This card states
    it as open and predicts no outcome. If recording (a)/(c) turned out to require deciding (b),
    this card parks; it does not.
(c) **Concurrent multi-checklist fill is a recorded product requirement** (T-43c) — C3 already
    implemented it (`HQFillSync`, `FILL_SYNC_SCOPES` as a map). This card's job is the **banner
    text**, not the lifecycle: restate the standing cancel rule as *"cancel before re-scoping THE
    SAME shape"* (B-63's corrected wording) now that the lifecycle it describes exists, and fix
    B-64's stale scope banner in `sync-rxdb/bootstrap.js`.

## Where the stale banner actually is (found by content, not line number)

The slate cited `sync-rxdb/bootstrap.js:80-86`; C2 added ~250 lines above it and that range is now
`readIdentityToken()`. Read start-to-finish, the file already carries **two** pieces of correct,
current commentary that predate this card and must not be disturbed:

- Lines ~228–262: the **SHAPE block** (T-43c, "crew members work MULTIPLE CHECKLISTS
  CONCURRENTLY") — this is C3's, already correct, already cites the B-63/B-64 wording. Not touched.
- Lines ~51–57: C2's own note, *"THIS FILE'S REMAINING STALE BANNER IS B-64's, NOT THIS CARD'S"* —
  a pointer at the thing this card fixes. Updated in place to say it is now fixed (a pointer left
  saying "still open" after the fix would itself be a new staleness of the same shape B-64 named).

**The actual stale banner** is the comment on the `startReplication` property inside the `HQSync`
object literal, currently at **lines 420–427**: it names only the FILL shape
(`{userId, checklistId, templateId, fieldIds}`), says nothing about the LIST shape, says nothing
about what's REST vs RxDB-served today, says nothing about T-43a/b, and ends with the pre-B-63
full-stop wording *"CANCEL the previous states before starting a re-scoped replication"* — which
under T-43(c) is actively wrong (it would read as "opening a second checklist must cancel the
first"). This is B-64 exactly as filed: a reader trusting this comment draws the conclusion the
operator's rulings overturned.

## Shared files touched — one line why each

| File | Why |
|---|---|
| `sync-rxdb/bootstrap.js` | **Owned.** Rewrite the stale `startReplication` banner (lines ~420-427) to name both shapes, state what's live/REST/open today, and restate the cancel rule. Update the now-stale "not this card's" pointer at ~51-57. |
| `sync-rxdb/client.js` | **Touched, narrowly.** Two banners carry the pre-B-63 cancel wording: the `startHQReplication` docblock (~1232-1236, the one C3's merge-intent names explicitly as mine to restate) and the older REPLICATION SCOPE design-block tail (~446-453, same rule, predates list scope). Both restated; the FILL shape line C3 flagged as must-survive-verbatim (~1222-1227) is untouched. |
| `.night-crew/knowledge/BACKLOG.md` | Close B-63 (jointly with C3's SCOPE-05/FILL tests) and B-64 (banner fixed), per the file's own closure convention (`🟢 **CLOSED** — run, card, branch, commit`). |
| `.night-crew/knowledge/roadmap.md` | Flip the Activity 4 `list-views-decision-recording` card entry from PLANNED to DONE, same change set (universal mechanics). |

Nothing else. No `workflows.html` — S1's footprint does not own it and C3's merge-intent already
flags a conflict there as a mistake, not a merge. No backend Go file. No new doc file — `docs/`
has no existing home for this recording (checked: `docs/codebase/`, `docs/contracts/`,
`.night-crew/knowledge/designs/offline-ownership.md` — none mention the cancel rule or B-63/B-64),
so per the card's own instruction the docs trail is the commit message + roadmap entry, not a new
file.

## What must survive any merge (this card's own contract, going forward)

1. **The SHAPE block in `bootstrap.js` (~228-262)** — C3's, unmodified by this card.
2. **The F-1 eviction comments and F-2 sentence on `scopeKey()`** — C3's, unmodified.
3. **`client.js`'s FILL shape line** `{userId, checklistId, templateId, fieldIds}` in the
   `startHQReplication` docblock — C3's, must survive verbatim; this card edits only the
   surrounding CANCEL sentence, not that line.
4. **The corrected wording, going forward: "cancel before re-scoping THE SAME shape."** The
   pre-B-63 wording ("cancel before re-scoping", full stop) must not come back anywhere in either
   file — it contradicts T-43(c).
5. **T-43(b) stated as OPEN, with no predicted outcome**, in every place this card touches.

## What is safe to drop

- Exact prose wording of the rewritten banners — the facts they must carry (both shapes; REST vs
  RxDB-served; T-43a/b/c) are what's load-bearing, not the sentences themselves.
- The updated "not this card's" pointer at bootstrap.js ~51-57 — purely narrative, no code depends
  on it.

## Red-first

**N/A — non-code deliverable, as cards C1/A4 did on run 20260806.** This card changes only
comments (`sync-rxdb/bootstrap.js`, `sync-rxdb/client.js`), a backlog entry's status text, and a
roadmap card's status. No function body, no test assertion, no runtime behavior changes anywhere
in the diff. There is nothing to show red before green: a comment carries no executable claim a
test could fail against. G2 (Go) and G2 (Playwright) are still run in full as gates — proving the
comment-only diff caused no regression — but that is the gate ladder's job, not RF's; RF exists to
prove a *defect* was captured before it was fixed, and this card fixes no defect in running code
(B-63/B-64 are documentation defects, and C3's own tests already cover the *behavior* B-63
worried about — the two-concurrent-fill regression in `tests/sync-rxdb-client.spec.js`'s
`[SCOPE-05]` block).

## Parks

None at the time of writing. The PARK trigger (recording cannot be written without deciding My
Checklists) has not fired: T-43(a) and T-43(c) are both fully recordable without touching (b), and
(b) is stated as open in every banner this card writes, per the slate's own instruction.
