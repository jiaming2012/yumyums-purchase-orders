# Merge intent — Card B `workflow-queue-period-and-failnote-upsert`

Branch: `card/b-workflow-queue-period-and-failnote-upsert` (cut from `overnight-20260729` @ `25fbc16`,
i.e. **after** Card A's Wave 0 merge — `Frontend` is already at `1.2.1` on this base).
Written BEFORE implementation, as the card's first commit. Required per DESIGN §15ad.65.

## Card in one line

Four consequences of the key-reuse decision (T-23 dec. 60 / T-25 dec. 71) land in one card because
they all need `backend/internal/workflow`: (1) `submission_fail_notes` has no unique constraint and a
bare INSERT, so one idempotency key POSTed twice writes the note **twice** — fix with a matching
`ON CONFLICT` + unique index; (2) `findQueuedSubmission` filters on `template_id` only, so a stale
queue entry can adopt a **later day's** submit — bound the lookup to the current period; (3) the
comment at `workflows.html:1781` claims "the server upserts only the fields present in each payload",
which (1) makes **true** rather than deleting; (4) `sync.js` and `workflows.html` render the same two
words, "Pending sync", for two different states on the same screen — name them apart.

## Shared files touched

- `backend/internal/db/migrations/0071_*.sql` — **new file, the card's core.** Additive: one unique
  index on `submission_fail_notes (submission_id, field_id)`. **The next free ordinal is 0071**
  (`0070_receipt_urls.sql` is the current head). If another card tonight also adds a migration, the
  merge must **renumber, not overwrite** — goose orders by filename and two `0071_`s is a silent
  data-shape divergence between environments. Nothing else on tonight's slate declares a migration
  in its footprint.
- `backend/internal/workflow/repository.go` — **the card's core.** The fail-note INSERT loop
  (`:759-767`, inside `submitChecklist`) gains `ON CONFLICT (submission_id, field_id) DO UPDATE`,
  mirroring the `submission_responses` insert directly above it (`:747-755`). **The responses insert,
  the `ON CONFLICT (idempotency_key) DO UPDATE ... RETURNING id` submission upsert (`:722-728`), and
  the `status` normalisation block (`:706-718`) all belong to OTHER landed cards and must survive
  verbatim.** Card B changes exactly one SQL string and its comment.
- `backend/internal/workflow/*_test.go` — one **new** test file for the red-first duplicate
  reproduction. It reuses this package's existing harness (`testPool`/`TestMain` in
  `stable_identity_test.go`, `ensureUser`, `newUUID` from `resubmit_photo_gate_test.go`) and adds
  **no** new harness. Conflicts here should be resolved by **keeping both sides' tests**.
- `workflows.html` — **the card's core, and the file most likely to conflict tonight.** Three
  regions: `findQueuedSubmission` (`:1719-1739`) gains a period bound; the long comment block in
  `submitChecklistToAPI` (`:1760-1786`) is where the durable falsehood lives and is corrected in
  place; and the `.pending-sync-mark` chip (CSS `:151`, logic `:329-342`, render `:2370`) is renamed
  for the vocabulary collision. **`workflow-offline-double-submit` (landed `bc8721e`) owns all three
  regions** — it wrote the key-reuse call, the comment, and the chip. Card B does not undo any of it:
  key reuse stays, the chip stays, only the *string and class* move and the *lookup narrows*.
- `sync.js` — **the card's core.** `enqueueSubmission` (`:549-556`) stamps a `period` alongside the
  existing `queuedAt`; `renderSyncBanner` (`:621-650`) is the other half of the vocabulary decision.
  **The `queuedAt` sort in `drainQueue` (`:576-580`) and the dead-branch `duplicate_submission`
  comment (`:585-589`) belong to `workflow-offline-double-submit` and must survive.**
- `tests/persistence.spec.js` — carries ~10 assertions on the literal string `Pending sync` and the
  `.pending-sync-mark` selector (`:498, 607, 616, 629-642, 692-737`). The vocabulary rename obliges
  a mechanical update of every one. **These are assertion updates, not behaviour changes** — the
  tests still assert exactly what they asserted before, under the new name. Plus the card's own
  back-and-reopen regression test (CLAUDE.md persistence rule).
- `tests/sync.spec.js` — read for the queue-entry shape; edited only if a queue assertion depends on
  the entry's field set. **`tests/sync.spec.js:1584`'s stale comment (B-06) is explicitly NOT folded
  in** — it belongs to `sync-rxdb-schema-and-replication`.
- `backend/internal/version/version.go` — **shared file, every card touches it.** This card touches
  `Backend` **and** `Frontend`. A merge that has to pick must take the **higher** semver on each
  constant independently and then re-mirror `Frontend` into `package.json`.
- `package.json` — **shared file, every card touches it.** `"version"` only, mirroring `Frontend`.
  **No devDependency, no script, no lockfile edit.** `package-lock.json` is NOT touched.
- `sw.js` — regenerated mechanically by `node build-sw.js` because `workflows.html` and `sync.js`
  move. Pure build output; see "safe to drop". (`version.json` is git-ignored and not committed.)
- `.night-crew/knowledge/roadmap.md` — this card's `PLANNED` → `DONE` flip at ~line 420, in the same
  change set as the work. Every card tonight edits its own card here; conflicts are per-card and
  both sides should be kept.
- `.night-crew/runs/2026-07-29-autonomous/merge-intent-b-workflow-queue-period-and-failnote-upsert.md`
  — this note. New file, unique to this card. No conflict surface.

### Late additions

_(appended only if implementation forces a file outside the list above; "nothing here" if it stays
clean. If a later repair contradicts a line ABOVE, that line gets struck here, not merely
supplemented.)_

Closed out after the gates. The whole note above was re-read; three lines are contradicted by what
was implemented and are **struck**, and one file outside the list was touched.

- **ONE file outside the list: `.night-crew/knowledge/designs/offline-save-honesty.md`** (one
  paragraph, `:147`). It instructs a FUTURE cutover card to delete "the `.pending-sync-mark` CSS" —
  a class this card renamed out of existence. Left alone it would send that card looking for a
  selector that is not there. Updated to `.unsaved-mark` with a note on why it moved. Documentation
  only; safe to drop on a conflict.
- ~~"`tests/sync.spec.js` — read for the queue-entry shape; edited only if a queue assertion depends
  on the entry's field set."~~ **STRUCK: `tests/sync.spec.js` was NOT edited.** It was read; no
  assertion enumerates a queue entry's fields, so adding `period` needed nothing there. The file is
  untouched by this card, including `:1584`'s stale comment (B-06), which stays for its own card.
- ~~"`workflows.html` … the `.pending-sync-mark` chip (CSS `:151`, logic `:329-342`, render
  `:2370`)"~~ **STRUCK as to the line numbers only** — the mechanism is right, the anchors moved
  with the edits. Final: CSS `:151`, logic `:344-357`, render `:2417`. Search by class name, not by
  line, when merging.
- ~~"`tests/persistence.spec.js` — carries ~10 assertions on the literal string `Pending sync`"~~
  **STRUCK as to the count**: 11 selector occurrences plus 1 `toHaveText`, 1 test title and 1
  comment — 14 sites, all mechanical. No assertion was weakened or removed.
- **The vocabulary was decided** (it was open when this note was written). `"Unsaved"` /
  `.unsaved-mark` for one unsent field answer; `"Queued"` / `.sync-badge` for a whole queued
  submission; banner `"N submissions queued to send"`. Guarded by `[VOC-01]`, which drives both
  states onto one screen and asserts the collided string is gone from the app.
- **Two files gained a 🛑 VOCABULARY comment block** (`workflows.html` above the unsaved-field
  state, `sync.js` above `renderSyncBanner`), each naming BOTH states and pointing at the other
  file. If a merge keeps only one side's block, the collision can be reintroduced from the
  unguarded side.
- **`node_modules/` was installed in this worktree via `npm ci`** (git-ignored, not in any commit).
  `package-lock.json` was not modified — `npm ci` installs from the lock, it does not write it.
  Constraint 1 holds.
- **`hq_test_e2e_b` must be DROPped and CREATEd before a full Playwright run.** Learned the hard
  way here: `task test` resets `hq_test_e2e` as a dedicated step because the suite shares one
  database across all 20 spec files, and running against an accumulated `hq_test_e2e_b` produced
  three order-sensitive failures in `inventory.spec.js` and `onboarding.spec.js` that vanished on a
  fresh DB. A card running with `TEST_DB_NAME=<own db>` does NOT inherit that reset — it has to do
  it itself. Nothing in the repo changed for this; it is a note for whoever runs the suite next.

## What must survive any merge

1. **`submission_fail_notes` cannot hold two rows for one `(submission_id, field_id)`.** The
   mechanism is a unique index plus a matching `ON CONFLICT ... DO UPDATE`; what must survive is the
   *property*. **Both halves are load-bearing** — the index without the `ON CONFLICT` turns a silent
   duplicate into a hard 500 on the second POST, which is strictly worse than the bug.
2. **`ON CONFLICT` updates `note` and `severity` and NOT `photo_url`.** `photo_url` is not in the
   INSERT column list, so `EXCLUDED.photo_url` is NULL; setting it would erase an attached photo on
   every re-POST. This is a deliberate omission, not an oversight.
3. **The migration is additive and creates no dedup rule.** The card's PARK trigger is exactly here:
   *which* duplicate to keep is a data decision, and the migration must not improvise one. It is
   safe to write only because the check below came back empty.
4. **A queue entry may only lend its `idempotency_key` to a submit in the SAME period.** The period
   is the calendar day, spelled the way the rest of `workflows.html` already spells it
   (`new Date().toISOString().slice(0, 10)`, as at `:2212`, `:2246`, `:2612`). An entry with no
   `period` (queued by pre-this-card code) is treated as **not** current — conservative in the
   direction that matters, since adopting a stale key is the defect.
5. **Aging out is NON-DESTRUCTIVE.** A stale entry is retired from key-reuse eligibility; it is
   **never deleted**. Offline, the queued payload is the only durable copy of what the crew member
   entered (the same reason `id` is deliberately not reused — `workflows.html:1775-1785`), and a
   submission queued Monday that drains on Friday is *correct*, not garbage. Any merge that turns
   age-out into a delete reintroduces silent data loss.
6. **Key reuse itself stays.** Decision 60 authorised it and `workflow-offline-double-submit` shipped
   it; this card narrows *when* it applies, it does not revoke it.
7. **The two "pending" states have two different names.** Whatever strings a merge picks, the
   field-level mark and the queued-submission badge must not read the same on one screen.
8. **`Frontend` in `version.go` and `"version"` in `package.json` are equal.**

## What is safe to drop

- **`sw.js`** — regenerated by `task sw` / `node build-sw.js`. Take either side and re-run.
- **The exact wording** of the new strings, the CSS class names, the migration filename's slug, and
  every comment. The behaviour matters; the prose does not. (The one sentence that must NOT survive
  in its *old, unqualified* form is `workflows.html`'s "the server upserts only the fields present in
  each payload" — this card is what makes it true, so the two must land together or not at all.)
- **Test names.**
- **The roadmap card's prose.** The **status flip** matters; the wording does not.
- **Anything in this note itself.**

## Not done, deliberately

- **No dedup SQL, no `DELETE`/`DISTINCT ON` in the migration.** See PARK trigger. The pre-check ran
  against the live Postgres on `localhost:5433` before a line of migration was written — both the
  `production` schema (prod's, per `docker-compose.prod.yml:41`) and `public` (dev's) hold
  **0 rows** in `submission_fail_notes` and therefore **0** `(submission_id, field_id)` duplicates.
  The trigger did **not** fire and no dedup rule was chosen.
- **No fix for orphaned fail notes.** `unsubmitChecklist` (`repository.go:1261-1266`) detaches fail
  notes to `submission_id = NULL` and nothing ever re-attaches them (unlike responses, which
  `submitChecklist` re-claims by `answered_by` — a column `submission_fail_notes` does not have).
  Those rows leak. It is a real defect, it is **not** this card, and the unique index does not touch
  it: Postgres treats NULLs as distinct, so orphans neither collide nor block the index.
- **No change to `drainQueue`'s ordering or eviction.** The `queuedAt` sort stays exactly as
  `workflow-offline-double-submit` left it.
- **No `openspec/` directory or OpenSpec mechanics.** `night-crew workflow preflight` reports
  openspec ABSENT for this repo. Universal per-change discipline only (red-first, atomic commits,
  `Night-Crew-Card:` trailer, roadmap flip).
- **No manual service-worker cache-version bump.** Workbox content-hashes every entry; CLAUDE.md
  forbids hand-bumped keys.
- **`tests/sync.spec.js:1584`'s stale comment (B-06) is NOT folded in.**

## Four-HARD-constraints attestation

1. **Root `package.json` AND `package-lock.json`** — `package.json` is touched on the `"version"`
   line ONLY, as the mandated mirror of the `Frontend` constant. **`package-lock.json` is
   UNTOUCHED.** (`npm ci` was run in this worktree to install the git-ignored `node_modules`; it
   installs *from* the lock and does not write it.)
2. **`backend/go.mod`** — **UNTOUCHED.** No dependency added, removed, or version-changed.
3. **`docker-compose.nc.yml`** — **UNTOUCHED.** This card runs Go against its own
   `hq_test_go_b` and Playwright against `TEST_PORT=8202` / `TEST_DB_NAME=hq_test_e2e_b`; no compose
   service is added, renamed, or re-ported.
4. **Root `Taskfile.yml`** — **UNTOUCHED.** The Playwright suite is run by invoking `npx bddgen` and
   then `npx playwright test` directly with explicit env, because `task test` omits the `bdd:gen`
   dependency and would silently run 19 of 20 spec files (B-09).
