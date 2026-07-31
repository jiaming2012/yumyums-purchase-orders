# Conflict log — run 20260801

Per §15ad.66: an entry for **every** merge to `overnight-20260801`, clean or conflicted, so an
empty log never reads as "no conflicts" when it means "the logging never ran."

Each entry records: the cards involved, the files and hunks, the merge-intent notes read, the
resolution taken, and the gate result after it.

Pre-resolved by the slate, not findings:
- `sw.js` is GENERATED — never merge the artifact; take either side, regenerate with `task sw`,
  re-run G4 (idempotence + version parity).
- `version.go` — resolve **per-constant**, not per-file (precedent `79fa7cd`, 07-29).
- Migration numbers are ASSIGNED: A1 = `0072`, B2 = `0073`. A card finding its number taken
  **stops and reports**; it does not renumber.

---

## Entries

### Merge 1 — `card/b2-sync-rxdb-row-visibility-rls` → `overnight-20260801`

**Result: CLEAN — zero conflicts.** Logged because every merge gets an entry, so an empty log
cannot be mistaken for "no conflicts" when it means "the logging never ran."

- **Cards involved:** B2 only. No other card had landed at this point.
- **Files:** 12 changed, 3238 insertions / 32 deletions. New: migration `0073_sync_fdw_views.sql`,
  `sync-schema/sql/0002_hq_fdw.sql`, `0003_rls_policies.sql`,
  `backend/internal/sync/{rowvisibility_rls_test.go,spikestack_gate_test.go}`, two QA captures,
  the card's merge-intent and closeout notes. Modified: `proxy_live_test.go`,
  `jwtbridge_rls_test.go`, `.night-crew/knowledge/roadmap.md` (this card's own bullet only).
- **Intents read:** B2's merge-intent note plus its F2/F1/F4 addendum. It declared
  `backend/internal/sync` as its own, no `sw.js` / `version.go` / `workflows.html` /
  `tests/sync.spec.js` change, and `roadmap.md` as its only shared file (one bullet).
  Nothing on the run branch contradicted that.
- **Why no conflict was possible:** B2 had already fast-forwarded `33bd8c0..01107fc`; the run
  branch's only later commits (`2a9d052`, `469e775`) are `DECISIONS-NEEDED.md`, a file B2 never
  touched. The three pre-resolved collision surfaces the slate warned about — `sw.js`,
  `version.go`, migration numbering — were all untouched by this card. Migration `0073` was
  claimed as assigned; `0072` remains free for A1.
- **Resolution taken:** none required. `--no-ff` so the card's history stays identifiable.
- **Gate after merge:** see below.

**Carried forward from this card's review, not resolved by the merge:**
- Fork 1 in `DECISIONS-NEEDED.md` — the ~23 ms/row cost (G6 F3). Operator call at triage.
- `tests/purchasing.spec.js:1792` (FR-23 repurchase-reset) failed once during the fix pass with a
  640 ms backwards jump between two server-side `now()` reads, then **passed in isolation**.
  Newly-observed red, distinct from B-27, not retired by the non-reproduction.
- The merge-intent's "safe to drop … port defaults" bullet is now **false** — those constants were
  the F1 bug. Reintroducing a port constant under `backend/internal/sync/` reintroduces it.

---

### Merge 2 — `card/a1-app-timezone-unify-new-york` → `overnight-20260801`

**Result: CLEAN — zero conflicts.**

- **Cards involved:** A1 landing onto a branch already carrying B2.
- **Files:** 28 changed, 1579 insertions / 107 deletions. New: migration
  `0072_app_timezone_new_york.sql`, `trends_window_test.go`, `service_weekstart_test.go`, two
  merge-intent notes. Modified: `inventory.html`, `purchasing.html`, `workflows.html`, `sync.js`,
  `sw.js`, `tests/{inventory,sync,workflows}.spec.js`, the `recipes`/`purchasing`/`inventory`
  packages, both `*-SALES-PROCESSOR-CONTRACT.md` documents.
- **Intents read:** A1's merge-intent plus its fix-round §9/§9a addendum, against B2's. B2 had
  declared no `sw.js`, no `version.go`, no `workflows.html`, no `tests/sync.spec.js` change — so
  every surface A1 touches was untouched by the card already on the branch. Disjoint by
  construction, not by luck.
- **Migration numbering held as assigned:** `0072` (A1) landed beside `0073` (B2) with no
  collision. Verified before the merge: A1's branch carried no `0073`.
- **Resolution taken:** none required. `--no-ff`.

**Carried forward, not resolved by the merge:**
- Fork 2 in `DECISIONS-NEEDED.md` — the June 2026 completeness-gate drift. **Stays OPEN.**
- A1's own gate honesty: two full Playwright runs, **exit 1 on both**, a different single failure
  each time and neither reproducing. Run 2's was the armed red `[LST-17]`; run 1's was
  `[A1-TZ-02]`, which was **not** armed and is now filed as **B-30** with its mechanism recorded
  rather than left as folklore.

---

### Merge 3 — `card/c1-sync-rxdb-replication-and-conflict-handler` → `overnight-20260801`

**Result: CONFLICTED — three files, all three pre-resolved by the slate or pre-flagged by the
cards. No card was parked; intent was readable on both sides in every case.**

**3a · `.night-crew/knowledge/BACKLOG.md` — content conflict.**
Both A1 and C1 independently filed **B-28**, for entirely unrelated findings, each appending at
the same line. Neither could see the other — the signature of concurrent dispatch, not a defect
in either card. **Resolution: took BOTH.** A1 keeps `B-28` (receipt worker COGS period) along
with `B-29` (the undisclosed `:67` drift) and `B-30` (`[A1-TZ-02]`); C1's launcher-gap entry was
renumbered to **`B-31`**. Nothing was dropped, and the renumber is recorded here because the
entry's own text refers to the card that raised it.

**3b · `.night-crew/runs/2026-08-01-autonomous/timings.log` — add/add conflict.**
The orchestrator's per-card table versus C1's own card-level record. **Resolution: took BOTH** —
the table, then C1's line under a "Card-level record" heading. C1's line carries the corrected
red-first figure (33 tests at `ab53478`, not 35), which is itself a G6 finding and must survive.

**3c · `sw.js` — GENERATED ARTIFACT, never merged.**
Took C1's side into the merge commit, then regenerated on the merged HEAD.

> 🛑 **A trap worth recording, because it nearly shipped the exact outage C1's card guards
> against.** `build-sw.js` builds its manifest from **git HEAD**, not the working tree. Running it
> *during* the unresolved merge emitted **24 files / 1986.1 KB**, silently skipping
> `sync-rxdb/bootstrap.js` and dropping `vendor/rxdb.bundle.js` from the precache — i.e. exactly
> Obligation 5's failure mode, produced by the merge procedure rather than by any card. It printed
> `skipped (not in HEAD)` and exited 0. The regeneration **must follow the merge commit**. After
> committing, the same command emitted **27 files / 2027.3 KB** and is idempotent (tree clean on a
> second run). The size differs from C1's own 2020.2 KB because A1's frontend changes are now in
> the tree — that difference is the merge working, not drift.

**3d · `backend/internal/version/version.go` — expected to conflict, did NOT.**
The slate predicted a conflict on nearly every merge. Git auto-resolved it correctly because only
C1 moved a constant: result `Frontend = "1.3.0"`, `Backend = "0.3.0"`. That **is** the per-constant
outcome precedent `79fa7cd` prescribes, reached without hand resolution. Verified after the merge
rather than assumed. Parity confirmed: `version.go` ≡ `package.json` ≡ `version.json` at 1.3.0.

**Gate after the merges:** G4 re-run as required after a merge touching a precached file —
idempotent, tree clean, parity green. Full G1/G2 on the final tree recorded in HANDOFF.md.
