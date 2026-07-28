# Merge intent — Card A `precache-manifest-from-head`

Branch: `card/a-precache-manifest-from-head` (cut from `overnight-20260729` @ `ae37835`)
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

`build-sw.js` globs the git **index** (`git ls-files`), so a *staged-but-uncommitted* file enters
the precache manifest; move it to the **commit** (`git ls-tree -r --name-only HEAD`), because
`task prod:deploy` ships the **committed** `sw.js` without regenerating it, and one precached URL
that 404s fails the **entire** service-worker install for every returning client. Amends decision
58's literal text ("the tracked set (`git ls-files`)") in service of its intent — ledger T-25
decision 67.

## Shared files touched

- `build-sw.js` — **the card's core.** One function, `trackedFiles()` (`:25-28`), moves from
  `git ls-files -z` to `git ls-tree -r --name-only -z HEAD`, plus the surrounding comment block
  (`:11-24`) which currently *asserts* the ls-files equivalence this card refutes.
  **`pwa-cache-and-build-hygiene` (landed 2026-07-27, decisions 58 + 59) owns this file** — it is
  the card that ADDED `trackedFiles()`, `trackedOnlyTransform()`, the `GENERATED_BUT_SHIPPED`
  allowlist, and the `globPatterns` comment about the vendored bundle. This card changes exactly
  one execFileSync argv and the prose explaining it; **everything else that card wrote must
  survive**, most especially `GENERATED_BUT_SHIPPED` and the absence of `vendor/**` from
  `globPatterns`. Nothing else on tonight's slate names `build-sw.js` in its footprint.
- `tests/sw-manifest.spec.js` — **the card's core.** Test 1 (`:25-39`) uses the *same*
  `git ls-files` and will keep agreeing with the bug unless it moves to `ls-tree` too; plus one
  new red-first test that stages a file and asserts it is absent from a freshly built manifest.
  Existing tests 2 and 3 (vendored bundle; untracked repo-root file) are **not** re-aimed and must
  survive verbatim. Conflicts should be resolved by **keeping both sides' tests**.
- `backend/internal/version/version.go` — **shared file, every card touches it.** This card edits
  the `Frontend` constant ONLY; `Backend` is not touched. A merge that has to pick must take the
  **higher** frontend semver and then re-mirror it into `package.json` — the two must never
  diverge (CLAUDE.md). No other line in this file belongs to this card.
- `package.json` — **shared file, every card touches it.** `"version"` only, mirroring the
  `Frontend` constant above. **No devDependency, no script, no lockfile edit.**
  `package-lock.json` is NOT touched.
- `sw.js` / `version.json` — regenerated mechanically by `node build-sw.js`. Pure build output.
  See "safe to drop". `version.json` is git-ignored and is not committed; `sw.js` is.
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` status flip, in the same
  change set as the work. Single-card edit at ~line 362. Every card tonight edits its own card in
  this file; conflicts are per-card and both sides should be kept.
- `.night-crew/runs/2026-07-29-autonomous/merge-intent-a-precache-manifest-from-head.md` — this
  note. New file, unique to this card. No conflict surface.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean)_

**Nothing here.** Closed out after the gates: the footprint held exactly as declared above —
`build-sw.js`, `tests/sw-manifest.spec.js`, `backend/internal/version/version.go`, `package.json`,
the regenerated `sw.js`, the roadmap flip, and this note. No file outside the list was edited, and
nothing above is contradicted by what was implemented, so no line is struck.

Two things the note did not anticipate and which a merge should know:

- **`node_modules/` was installed in this worktree via `npm ci`** (it is git-ignored and is NOT
  part of any commit). `package-lock.json` was not modified — `npm ci` installs from the lock, it
  does not write it. Constraint 1 holds.
- **`build-sw.js:8`'s comment** ("Everything else must be tracked") was corrected to "committed in
  HEAD" in the final commit rather than the fix commit. Verified inert: re-running
  `node build-sw.js` after the edit produced a byte-identical `sw.js`. Comment-only, safe to drop
  on a conflict.

## What must survive any merge

1. **The precache manifest is built from the COMMIT, not the index.** The mechanism is
   `git ls-tree -r --name-only -z HEAD` inside `trackedFiles()`; what must survive is the
   *property* — **a staged-but-uncommitted file can never reach the manifest** — not the exact
   spelling. `git ls-files` reads the index and is precisely what fails here. Anything that
   restores an index-reading globber restores the bug.
   The `-z` and the `-r` are both load-bearing: `-r` recurses into trees (without it you get top-
   level directory entries, not `icons/icon-96x96.png`), and `-z` gives NUL-separated **unquoted**
   paths — plain `ls-tree --name-only` C-quotes any path with a space or non-ASCII byte, which
   would silently drop a legitimately tracked asset from the allow-set and therefore from the
   precache.
2. **`GENERATED_BUT_SHIPPED` stays, and `version.json` stays on it.** This is the card's loudest
   explicit "do not drop". `version.json` is git-ignored — it is in **neither** the index nor the
   commit — and `backend/Dockerfile:33-44` regenerates it inside the image from the authoritative
   `Frontend` constant **precisely because `sw.js` precaches it**. Drop the allowlist and every
   deploy ships an `sw.js` whose manifest omits `version.json`; keep the allowlist but drop
   `version.json` from it and the entry 404s and bricks the install. Same trap either way.
3. **The dropped-entry `console.warn` stays.** Loud, not silent (decision 58). An asset someone
   MEANT to ship must surface as "git add and COMMIT it", not as a dead service worker two deploys
   later.
4. **`globPatterns` is not touched.** The vendored RxDB bundle stays OUT (decision 59, −495 KiB);
   `sync-rxdb-schema-and-replication` re-adds it on adoption. The `runtimeCaching` block is not
   touched either — cache-key design is deferred by decision 57.
5. **No manual service-worker cache-version bump.** Workbox content-hashes every precache entry;
   CLAUDE.md forbids hand-bumped cache keys and none is introduced.
6. **The new red-first test.** Stage a file → rebuild → assert absent from the manifest. It is the
   only thing that stops the globber sliding back to the index. Test 1 moving to `ls-tree` is part
   of the same guarantee: on `ls-files` it agrees with the bug.
7. **`Frontend` in `version.go` and `"version"` in `package.json` are equal.** Whatever a merge
   picks, it must pick the same string in both files.

## What is safe to drop

- **`sw.js` and `version.json`** — regenerated by `task sw` / `node build-sw.js`. Take either side
  of a conflict and re-run `task sw`; the content hashes are derived, not authored.
- **Comment wording** throughout `build-sw.js` and `tests/sw-manifest.spec.js`, including the
  rewritten `trackedFiles()` block comment. The behaviour matters; the prose does not. (The one
  sentence that must NOT survive verbatim is the current claim that filtering the manifest is
  "equivalent to globbing `git ls-files`" — it is the assertion this card refutes.)
- **Test names and the probe filenames** (`zz-sw-manifest-*.html`). Any unique non-colliding name
  is fine.
- **The roadmap card's prose.** The **status flip** matters; the wording does not.
- **Anything in this note itself.**

## Not done, deliberately

- **No `Dockerfile` change.** The card's PARK trigger is exactly here: if `ls-tree` excluded
  something the image genuinely ships beyond the allowlist, that is a Dockerfile question, not a
  globber question, and it parks rather than being improvised. (Pre-check on the clean worktree:
  `git ls-files` and `git ls-tree -r HEAD` are byte-identical, and every `globPatterns` match is
  tracked in HEAD except the allowlisted `version.json`.)
- **No new allowlist entries.** `GENERATED_BUT_SHIPPED` keeps exactly one member.
- **No change to `Taskfile.yml`'s `prod:deploy`.** Making the box run `task sw` before the docker
  build is a *different* fix to the same failure class and is not this card's scope; this card
  makes the committed `sw.js` correct at authoring time instead.
- **No `openspec/` directory or OpenSpec mechanics.** `night-crew workflow preflight` reports
  openspec ABSENT for this repo. Universal per-change discipline only (red-first, atomic commits,
  `Night-Crew-Card:` trailer, roadmap flip).
- **`tests/sync.spec.js:1584`'s stale comment (B-06) is NOT folded in.** It belongs to a different
  card that is not on tonight's slate.
- **`backend/` is not opened** beyond the one-line `Frontend` constant. This card compiles no new
  Go.

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — `package.json` is touched on the `"version"`
   line ONLY, as the mandated mirror of the `Frontend` constant. **`package-lock.json` is
   UNTOUCHED.** No devDependency is added, no version moved, no script edited. This is the
   Playwright environment shared by every card and every worktree tonight.
2. **`backend/go.mod`** — **UNTOUCHED.** No dependency added, removed, or version-changed. This
   card is a build script + tests + a semver constant; it compiles no new Go.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** This card runs Playwright against its own
   `TEST_PORT=8201` / `TEST_DB_NAME=hq_test_a`; no compose service is added, renamed, or re-ported.
4. **Root `Taskfile.yml`** — **UNTOUCHED.** `task sw` and `task test:go` are invoked as they
   already exist; no task is added, no var default changed. The Playwright suite is run by
   invoking `npx bddgen` and then `npx playwright test` directly with explicit env, because
   `task test` omits the `bdd:gen` dependency and would silently run 19 of 20 spec files (B-09).
