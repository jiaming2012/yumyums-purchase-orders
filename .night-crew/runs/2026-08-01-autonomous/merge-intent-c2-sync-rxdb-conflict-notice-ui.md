# Merge intent — Card C2 `sync-rxdb-conflict-notice-ui`

Run `overnight-20260801`, branch `card/c2-sync-rxdb-conflict-notice-ui`, cut from
`overnight-20260801` at `1eaa5bf` (i.e. **after** C1 merged, so the conflict
handler this card renders against is already in the tree).

Written BEFORE implementing, per the run's mandatory mechanics. If the build
contradicts a line below, that line is struck through **in place** — not merely
appended to (B-11).

---

## Shared files touched (files that exist on the base branch)

Every file here is outside my own new modules, with one line of why.

- **`workflows.html`** — **the card's whole point.** Three disjoint regions:
  (a) five new CSS custom properties (`--ok-bg/--ok-tx/--bad-bg/--bad-tx/--skel`)
  appended to the existing `:root` + dark blocks, because the mockup's palette
  needs them and workflows.html declares only nine; (b) one new stylesheet block
  for the conflict banner + sheet, appended at the end of the existing
  `<style>`; (c) one host `<div id="conflict-notice">` at the top of `#s1` and
  one `<script type="module">` that mounts it. **A1 and C1 have both already
  landed here.** My regions are disjoint from A1's (timezone comparisons around
  `isCurrentPeriodEntry` and the "already submitted today" checks) and from C1's
  (the single `<script type="module" src="sync-rxdb/bootstrap.js">` tag, which I
  do not move). **On conflict: take both sides.**
  🛑 **No write path is swapped.** `autoSaveField` → `POST /saveResponse` →
  `DRAFT_RESPONSES` → `hydrateFieldState` is byte-untouched; that is
  `sync-hard-cutover`.
- **`sync-rxdb/conflict-handler.js`** (C1's file, one night old) — **G6
  correction 1 only.** `describeConflict` re-ran `resolveConflict` with its own
  `opts` rather than the handler's configured `mergeOpts` (`:411`, call at
  `:414`), so a caller who customised `reservedFields`/`provenanceFields` at
  `createHQConflictHandler` got a `conflict$` clash list that disagreed with what
  the handler actually did. The handler now **exposes** its `mergeOpts` and the
  subscription threads them through. Nothing about decision 50's rule moves.
- **`sync-rxdb/client.js`** (C1's file) — the `conflict$` subscription in
  `startHQReplication` now passes the handler's own opts to `describeConflict`,
  and gains an optional recorder hook so a conflict can be written to the durable
  local record the instant it arrives (the mockup's stated precondition).
- **`sync-schema/collections.js`** (B1's file) — the `conflict_records` local
  collection gains the properties the sheet cannot be drawn without: the group
  key and its display fields, the **current server value** (every plate draws a
  `Now shows` line and the r2 schema carries no such field), the frozen
  `field_label` A-3 requires, the removed-field flag, and the row's status. **The
  three declarations B1's header says other cards must not re-litigate are
  untouched**: `_modified`/`_deleted` stay undeclared, who-and-when stays, the
  collection stays local with no `table`. Every pre-existing property and the
  whole `required` list are left exactly as B1 wrote them, so
  `tests/sync-schema.spec.js` keeps its meaning. **The literal `30` is still
  written exactly once in that file** — B1's retention test tokenises every
  numeric literal in it and I add none.
- **`.planning/phases/sync-rxdb-conflict-notice/mockup.html`** — the **A-3 plate
  redraws** (decision 95): `edge-removed`, `openq-count-a`, `openq-count-b`. The
  raw field id in muted monospace is replaced by the question's frozen label,
  struck through and read-only. This is a **deviation from the signed plates**
  and is noted here and in SUMMARY.md per CLAUDE.md's mockup rule.
- **`.planning/phases/sync-rxdb-conflict-notice/screenshots/*.png` + `shoot.mjs`**
  — the redrawn plates are re-shot, and `shoot.mjs`'s measurement-6 classifier
  learns the new struck-label class so a removed-field row is still classified as
  a removed-field row. Its seven population floors are **not** lowered.
- **`.planning/phases/sync-rxdb-conflict-notice/UI-SPEC.md`** — the status line
  ("DRAFT AWAITING OPERATOR SIGN-OFF. Nothing here is approved.") is now false:
  decision 98 signed revision 2. Corrected in place; nothing else in that file's
  normative content is edited.
- **`build-sw.js` / `sw.js` / `version.json`** — **no source edit to
  `build-sw.js`.** Its existing `'sync-rxdb/*.js'` glob already covers my two new
  modules, so `sw.js` is regenerated only. **NEVER merge the generated artifact**
  — take either side and immediately re-run `node build-sw.js` **after
  committing**, because the manifest transform reads git HEAD, not the working
  tree. Precache goes **27 → 29 files** (my two new `sync-rxdb/` modules).
- **`backend/internal/version/version.go` + `package.json`** — `Frontend` bumped
  (new shipped frontend assets). `Backend` untouched. **Resolve per-constant, not
  per-file** — precedent `79fa7cd`.
- **`.night-crew/knowledge/roadmap.md`** — status flip of my own bullet only.

## What must survive any merge

1. **The A-1 banner carries BOTH figures.** What was overwritten in the retention
   window *and* how many rows are still to review. A merge that resurrects the
   struck single-count rule reinstates the defect A-1 was filed against.
2. **Rows leave the sheet only on Dismiss or expiry** ((b) STANDS). Restore and
   Keep collapse to a confirmation that keeps an **Undo**; nothing is removed,
   because a removed row cannot be undone.
3. **A-2's confirm.** `Restore all N` never writes through: the confirm names the
   loss in its title, lists every server value struck through with who saved it
   and when, and its primary button reads **Replace**.
4. **A-3.** A removed question keeps its label, struck through and read-only —
   with the raw field id as the fallback *only* when the snapshot carries no
   label. **A malformed `template_snapshot` (B1's recorded-not-fixed item R-C)
   must not throw and must not render nothing** — it falls back to the raw id.
5. **One named constant for retention** (decision 96) — imported, never restated.
6. **The 10-group cap with an "and N more" line** (decision 97) — rows below the
   line are not dropped and the banner still reports the true total.
7. **`describeConflict` and the handler must agree.** The opts threading fix and
   the test that customises one field.
8. **`HQ_SYNC_REST_URL` is set nowhere.** Nothing deploys tonight.

## What is safe to drop

- `sw.js` and `version.json` — generated. Regenerate, never merge.
- The screenshot PNGs under `.planning/phases/sync-rxdb-conflict-notice/screenshots/`
  — regenerate with `node .planning/phases/sync-rxdb-conflict-notice/screenshots/shoot.mjs`
  and with `npx playwright test tests/states-sync-rxdb-conflict-notice.spec.js`.
- Prose in my own new modules' header comments.

## Files I deliberately do NOT touch

- **`sync.js`**, `backend/**` (any Go source), `index.html`, `inventory.html`,
  `purchasing.html`, `users.html`, `onboarding.html`, `login.html` — untouched.
  Go gates are run as evidence I did not break them, not because I edited them.
- **`playwright.config.js`** — in particular `serviceWorkers: 'block'` stays.
- **`tests/sync.spec.js`, `tests/inventory.spec.js`** — B-27 and B-30 are known
  reds and are not mine.
- Anything under `.night-crew/knowledge/` except my own roadmap bullet.

## Empty fields, stated explicitly

- **New npm dependency:** nothing here.
- **New Go package / migration / SQL:** nothing here.
- **Deploy / infra / env var change:** nothing here.

---

## Amendments made during the build (B-11 — struck in place, not appended)

*(none yet)*
