# Merge intent — Card D `sync-rxdb-conflict-notice-mockup`

Branch: `card/d-sync-rxdb-conflict-notice-mockup` (cut from `overnight-20260729` @ `d73580d`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

Draft the committed mockup the attended sign-off consumes, so `sync-rxdb-conflict-notice-ui` —
ATTENDED-BLOCKED with nothing for the operator to review — becomes unblockable. Show what a crew
member sees when a same-field clash falls back to **master-wins**, and **how they recover the
discarded value**, grounded in what `conflict$` actually emits (verified at W2, not imagined).
**Zero production code.**

## Shared files touched

- `.planning/phases/sync-rxdb-conflict-notice/mockup.html` — **new, the card's core.** New
  directory. No other card on tonight's slate names `.planning/phases/`. No conflict surface.
- `.planning/phases/sync-rxdb-conflict-notice/UI-SPEC.md` — **new, the card's core.** Holds the
  State Enumeration Table and the `done_when:` block CLAUDE.md's Definition of Done requires, plus
  the `conflict$` evidence the design rests on. No conflict surface.
- `.planning/phases/sync-rxdb-conflict-notice/screenshots/*.png` + `shoot.mjs` — **new, the card's
  core.** The self-verification renders (480px, light and dark, one pair per table row) and the
  script that produced them, committed so the render is reproducible. No conflict surface.
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip, in the same
  change set as the work, matching the convention Cards A and C used tonight (`:359`, incl. the
  `Original card text:` preservation block). Single-card edit at `~:666`. It also **annotates the
  sibling card `sync-rxdb-conflict-notice-ui` (`~:683`)** to record that the sign-off artifact now
  exists — that is the whole point of this card, and it is a two-line addition inside that card's
  own bullet, not a status change (that card stays ATTENDED-BLOCKED; only the operator can move
  it). Every card tonight edits its own region of this file; conflicts are per-card and **both
  sides should be kept**.
- `.night-crew/runs/2026-07-29-autonomous/merge-intent-d-sync-rxdb-conflict-notice-mockup.md` —
  this note. New file, unique to this card. No conflict surface.
- `.night-crew/runs/2026-07-29-autonomous/timings.log` — append-only card timing line, if the
  run's convention calls for one. Append-only; a merge should keep **both** sides' lines.

**Files NOT touched — assert this against any merge that shows otherwise:**

- `workflows.html`, `sync.js`, `ptr.js`, `index.html`, any other root HTML/JS — **NOT touched.**
  This card is the mockup, not the UI. A diff attributed to this card touching any of them is
  wrong; drop it.
- `sw.js` / `version.json` — **NOT touched, not regenerated.** No shipped HTML/JS file changed, so
  `node build-sw.js` has nothing to do. `.planning/` is not precached. If a merge shows an `sw.js`
  diff attributed to this card, take the other side.
- `package.json` / `package-lock.json` — **NOT touched.** No version bump (`Frontend` stays
  `1.2.1`), no dependency. Playwright is invoked from the main clone's already-installed
  `node_modules` read-only to render the mockup; nothing is installed into this worktree.
- `backend/internal/version/version.go` — **NOT touched.** No shipped file changed on either side,
  so **neither** constant moves. `Backend` stays `0.3.0`, `Frontend` stays `1.2.1`. A merge that
  shows this card bumping either constant is wrong.
- `backend/` generally, `tests/`, `Taskfile.yml`, `docker-compose*.yml`, `.claude/` — **NOT
  touched.**

## What must survive any merge

1. **`mockup.html` exists at `.planning/phases/sync-rxdb-conflict-notice/mockup.html`.** That exact
   path is what CLAUDE.md's sign-off gate reads (`.planning/.../<phase>/mockup.html`) and what the
   blocked card's unblock condition names. Moving or renaming it re-blocks the card.
2. **The State Enumeration Table, with all seven rows.** Four base (empty, loading, error, success)
   plus the three edge rows the slate names by hand: **no discarded value available**, **several
   conflicts at once**, **conflict on a field since removed from the template**. CLAUDE.md calls
   the table incomplete without ≥2 edge rows; this card owes three specific ones. Dropping an edge
   row silently converts a signed-off design into an unsigned one.
3. **The recovery path.** "How they recover the discarded value" is the card's point, not
   decoration. A crew member who loses an entered answer to master-wins must have a visible,
   concrete way to get it back. A merge that keeps the notice and drops the recovery affordance
   keeps the complaint and drops the remedy.
4. **The honest statement of what `conflict$` does NOT give.** The design is built on a verified
   signal with real limits (in-memory `Subject`, no replay, whole-document not per-field, fires
   only where something is subscribed). Those limits are why two of the edge rows exist. A merge
   that keeps the pretty states and drops the limits section leaves the next implementer free to
   assume an API that does not exist.
5. **Zero production code.** The card's own hard boundary.

## What is safe to drop

- **The visual design itself** — colours, copy, ordering, whether the recovery lives in a sheet or
  inline. That is exactly what the operator is being asked to say yes or no to, and a "no" is a
  successful outcome for this card. What must not be dropped is *that a recovery path is shown*.
- **The screenshots and `shoot.mjs`.** They are the self-verification evidence, not the artifact.
  Regenerable from `mockup.html` at any time.
- **The roadmap card's prose.** The status flip matters; the wording does not.
- **Anything in this note itself.**

## Not done, deliberately

- **No production code, of any kind.** No `conflict$` subscription, no `conflictHandler`, no
  `workflows.html` change, no RxDB client. The card is the artifact the gate reads.
- **Red-first DOES NOT APPLY and is not silently omitted.** There is no code and no test in this
  card — an HTML mockup has no behaviour to capture red. The substitute discipline is the
  self-verification ritual (render at 480px light+dark, read the PNGs back, compare row-by-row
  against the visual contract), which is what CLAUDE.md actually demands of a UI artifact in a
  headless environment.
- **No `openspec/` directory or OpenSpec mechanics.** `night-crew workflow preflight` reports
  openspec ABSENT for this repo. Universal per-change discipline only: atomic commits by logical
  unit, the `Night-Crew-Card:` trailer on every commit, the roadmap flip in the same change set.
- **No decision that belongs to the parent card.** The conflictHandler's actual merge rule, the
  replicated schema, `_modified`-in-schema, the collections and their RLS are
  `sync-rxdb-schema-and-replication`'s. Where the mockup needs one of them to be true it states the
  assumption explicitly rather than deciding it.
- **No sign-off is claimed.** This card produces the artifact; the human "ok, build this" is the
  operator's and has not happened. `sync-rxdb-conflict-notice-ui` stays ATTENDED-BLOCKED.
- **No `tests/states-sync-rxdb-conflict-notice.spec.js`.** That spec is the *UI* card's
  self-verification against shipped code; writing it here would put a Playwright spec in `tests/`
  asserting behaviour that does not exist, and `tests/` is outside this card's footprint.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — **UNTOUCHED.** No dependency, no script, no
   version bump. Playwright is read from the main clone's existing install; nothing is added to
   this worktree. This is the environment shared with the concurrently-running card.
2. **`backend/go.mod`** — **UNTOUCHED.** No Go file is read or written by this card.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** No container is brought up, torn down, or dialled.
   The mockup is a static file rendered from `file://`.
4. **Root `Taskfile.yml`** — **UNTOUCHED.** No task added, no var default changed. `task sw`,
   `task test` and `task prod:deploy` are all deliberately NOT run: nothing shipped changed.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean)_

**Nothing here yet** — written before implementation. This section is filled in at close-out, and
if a later change contradicts any line above, the note is re-read IN FULL and the contradicted
lines are struck rather than merely appended to (B-11).
