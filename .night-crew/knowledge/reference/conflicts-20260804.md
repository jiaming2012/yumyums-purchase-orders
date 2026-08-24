# Conflict log — run `overnight-20260804`

Per §15ad.66: **every** merge gets an entry, clean or conflicted. A clean merge gets a one-line
entry, so an empty log never reads as "no conflicts" when it means "the logging never ran."

Each entry: the cards involved, files and hunks, the merge-intent notes read, the resolution taken,
and the gate result after it.

---

<!-- entries appended in merge order -->

## Merge 1 — A1 `e2e-gate-database-isolation` → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. First card of the run; nothing to collide with.

- **Cards involved:** A1 alone.
- **Files:** `playwright.config.js`, `Taskfile.yml`, `scripts/reset-e2e-db.js` (new),
  `tests/db-isolation.spec.js` (new), `BACKLOG.md`, `roadmap.md`, and the card's merge-intent note.
  7 files, 638+/40−.
- **Merge-intent read:** `merge-intents/merge-intent-a1-e2e-gate-database-isolation.md`. Its
  must-survive list carries 7 items; item 7 (the reset banner reaching stderr) was *promoted* from
  "safe to drop" during the fix round, because dropping it is exactly B-81.
- **Outside the stated footprint:** `scripts/reset-e2e-db.js` — a new module; the slate named only
  "a new spec under `tests/`" plus the three config files. Disclosed in the note. Legitimate.
- **`night-crew.toml` NOT touched** — a result, not an omission. The slate listed it expecting
  `[e2e] suite`/`subset` would need repointing; they didn't. The PARK trigger never fired.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G1 `go build ./...` rc=0 · `go vet ./...` rc=0 (from `backend/`).
  G4 `node build-sw.js` → **31 files precached** (unmoved), reachability 18 parsed / 30 resolved /
  0 outside, frontend 1.4.0; second run left the tree clean ⇒ idempotent. Full G2 is deferred to
  the closeout's final-tree gate, per the run's normal shape.
- **G6 verdict:** APPROVE-WITH-NOTES. Acceptance gate independently re-derived by execution.

---

## Merge 2 — A2 `workflows-autosavefield-phantom` → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. A2 branched off the run branch *after* A1 merged, so the
two cards' shared files (`BACKLOG.md`, `roadmap.md`) were already reconciled by construction — this
is the serial dispatch doing the job concurrency would have made harder.

- **Cards involved:** A2 alone. A1 was already in the base.
- **Files:** 15 files, 594+/46−. `workflows.html` (the fix), `tests/persistence.spec.js` (the
  red-first `[FLD-16B]`), `sw.js`, `CLAUDE.md`, `sync-rxdb/bootstrap.js`,
  `sync-rxdb/conflict-notice-ui.js`, `docs/data-flow-audit.md`, `README.md`,
  `.claude/skills/save-project/SKILL.md`, `prds/PRD-operations-hardening.md`,
  `designs/fetchstorm-replay-class-superseded.md`, `tests/sync-rxdb-client.spec.js`,
  `BACKLOG.md`, `roadmap.md`, and the card's merge-intent note.
- **Merge-intent read:** `merge-intents/merge-intent-a2-workflows-autosavefield-phantom.md`.
  Must-survive carries 4 items; item 4 records that B-65's "match a symbol not a substring" lead is
  **partially, not fully, discharged**, with the failure scenario stated.
- **Outside the stated footprint:** five files —`docs/data-flow-audit.md`, `README.md`,
  `.claude/skills/save-project/SKILL.md`, `PRD-operations-hardening.md`,
  `tests/sync-rxdb-client.spec.js`, plus `designs/fetchstorm-replay-class-superseded.md` from the
  fix round. All disclosed. The slate named four doc sites; **eight** were wrong, and the transport
  (`POST /saveResponse`) was a second phantom the slate did not know about. Correcting `CLAUDE.md`
  while leaving its downstream restatements would have made the correction unfollowable.
- **🛑 The PRD was appended to, not rewritten.** Both corrections sit in blockquotes *below* the
  signed requirement text, each stating the requirement itself is unchanged. Annotation is a card's
  to do; rewriting signed requirement text is not. G6 verified this specifically.
- **🛑 PARK trigger honoured, not driven through.** `CLAUDE.md`'s persistence rule survives intact —
  G6's separate verdict. Only the false function name, transport and call shape changed. The
  persisted-states list went 7 → 9 (*widening* the rule's coverage), and step 2 of the
  add-a-field-type procedure got **stricter**, not looser.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G4 `node build-sw.js` run **after** the merge commit (B-37) →
  **31 files precached** (unmoved), reachability 18 parsed / 30 resolved / 0 outside, frontend
  1.4.0; `sw.js` unchanged and tree clean on both runs ⇒ idempotent. The card's own G2 was the
  **full 787-test suite** — not by choice but because of B-87 — `1 failed / 6 skipped / 780 passed`,
  the one failure being armed red `yes/no answer converges (live + catch-up)` matched by full title,
  which **stays ARMED**. First figures this milestone measured against a **fresh** database.
- **G6 verdict:** APPROVE-WITH-NOTES, and it **refuted a claimed finding at source** — B-87's
  proposed consequence (a confined gate silently running the *wrong* specs) does not follow, because
  the CLI path filters are OR'd and can only widen. Filed with the corrected consequence, wrong
  version struck in place.

---

## Merge 3 — A4 `offline-ownership-design-note` → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. The card's merge-intent flagged a plausible textual
conflict in `roadmap.md` (its insertion point sits in the region A2 edited); **it did not
materialize** — serial dispatch again, A4 branched off a base that already contained A2.

- **Cards involved:** A4 alone. A1 and A2 were already in the base.
- **Files:** 4 files, 740+/0−. `designs/offline-ownership.md` (new, the deliverable), `roadmap.md`,
  `BACKLOG.md`, and the card's merge-intent note. **No code** — `git diff 2041477 --name-only`
  matches nothing outside `.night-crew/`, and `sw.js` is byte-identical to base
  (`sha256 f977f67d…`).
- **Nothing outside the stated footprint.**
- **🛑 The deliverable publishes NINE classes across six stores, not the plan's eight.** G6 found
  `LOCAL_COLLECTIONS.conflict_records` (`sync-schema/collections.js:277-284`) — `replicated: false`,
  no `table` key *by design*, 30-day local sweep, with a shipped write (`conflict-notice.js:788-796`)
  and a shipped read (`workflows.html:3588-3592`) — while the note's own Rule 4 footnote denied a
  ninth class existed. It is distinct by the note's own splitting logic: class #8 is a replication
  buffer with a server copy; #9 **never leaves the phone and has no server row anywhere.** Losing it
  loses the only record of a crew member's overwritten edit.
  **The omission was inherited** — the plan's §3 A4 table and its §8 mapping table *independently*
  both have 8 rows, and `grep -c conflict_records` over the whole plan returns **0**. The store count
  did not move; only the class count did.
- **The signed non-negotiable ("publish all 8 classes across 6 stores") is satisfied, not violated** —
  all 8 are published unchanged at the same 6 stores, and a ninth is added under the card's own
  governing rule: *where measurement contradicts the plan's table, the measurement wins and the
  deviation is stated.* The count being the finding is the note's whole point.
- **Rule 2's enforcement claim corrected.** The note had said `tests/sync-rxdb-client.spec.js:1468-1470`
  is "the rule's only enforcement." True but misleading: `workflows.html:3588-3592` already reads
  RxDB via **`window.HQSync.db`**, which that guard does not watch, on an eagerly-mounted page-load
  path that runs offline. Rule 2 is gated today by the database not existing. Filed as **B-88**.
- **Seven deviations from the planning-time table stated, not silently patched** — including **D5**,
  where a claim was attributed to ledger **T-21d** that T-21d does not make (T-21d ends `:1041`;
  the supporting text is **T-21e**). G6 independently confirmed D5's underlying claim is correct.
- **🛑 §8 described, not decided.** G6 checked this specifically: the note states in as many words
  that it does not adopt §8, and that adopting it is a roadmap decision. No product fork improvised.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G4 `node build-sw.js` → **31 files precached** (unmoved), frontend
  1.4.0, tree clean ⇒ idempotent. No Playwright or Go suite run, and that is stated rather than
  skipped silently: the diff is three markdown files under `.night-crew/` and neither suite can
  observe it.
- **G6 verdict:** APPROVE-WITH-NOTES. Filed **B-88..B-91** — including **B-89**, a live code bug
  found while verifying row 5 (`index.html:232-235` writes `hq_apps` as an envelope `{uid, apps}`;
  `sync-rxdb/bootstrap.js:62-71` `Array.isArray`-gates it, so `cachedGrantSlugs()` returns `[]`
  unconditionally, and the test at `sync-rxdb-client.spec.js:1385` plants the *array* shape so
  nothing catches it — a sibling of B-65 in kind).

---

## Merge 4 — A6 `app-version-badge` 🅢 → `overnight-20260804`

**CLEAN.** No conflicts, no hunks resolved. The night's stretch card, merged last.

- **Cards involved:** A6 alone. A1, A2 and A4 were already in the base.
- **Files:** 9 files, 732+/12−. `index.html` (the badge), `tests/version-badge.spec.js` (new, 5
  tests), `playwright.config.js` (the B-92 fix), `scripts/write-version-json.js` (new, extracted),
  `sw.js`, `BACKLOG.md`, `roadmap.md`, and the card's merge-intent note.
- **Merge-intent read:** `merge-intents/merge-intent-a6-app-version-badge.md`. Its must-survive list
  centres on the one thing that makes the card worth having: **the badge's value comes from the
  precached `version.json` and there is no fallback to `/api/v1/health`.**
- **Outside the stated footprint:** two — `BACKLOG.md` (B-92) and `scripts/write-version-json.js`,
  the latter added by the fix round. Both disclosed.
- **🛑 A6 touched `playwright.config.js`, which A1 owns.** This is the one place tonight where two
  cards' work met in the same file, and it merged clean because A1 was already in A6's base — the
  fix *composes* with A1's reset rather than replacing it. Verified at merge time and by the fix
  round: `webServer.command` now reads
  `node scripts/reset-e2e-db.js && node scripts/write-version-json.js && cd backend && …`, and both
  banners appear in order:
  `[WebServer] ── reset hq_test_e2e_a6fix on localhost:5433 ──` then
  `[WebServer] ── wrote version.json frontend=1.4.0 ──`.
  **A1's reset still runs first and is still unskippable.** A fix that silently disabled it would
  have re-opened B-76 while looking green; it did not.
- **`serviceWorkers: 'block'` (B-15) untouched** — held by every card this run, as by every card
  before it.
- **Resolution taken:** none required — `--no-ff` merge, ort strategy, no conflicts.
- **Gate result after the merge:** G4 `node build-sw.js` run **after** the merge commit (B-37) →
  **31 files precached** (2165.0 KB, unmoved — `version.json` was already in the set), reachability
  18 parsed / 30 resolved / 0 outside; tree clean on the second run ⇒ idempotent. Three-way version
  parity **1.4.0**; **no version bump**. The run's authoritative G1/G2/G4 figures are in HANDOFF.md,
  taken by the orchestrator on this final merged tree rather than inherited from card reports.
- **G6 verdict:** APPROVE-WITH-NOTES. It proved the anti-reroute guarantee **by execution** — writing
  two forbidden implementations (API-only, and `version.json`-with-fallback) and confirming the spec
  reds on both — and it found that the card was introducing a regression (**B-92**), which the fix
  round closed rather than filed.
