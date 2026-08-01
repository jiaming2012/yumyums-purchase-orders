# Conflict log — run 20260802

Per §15ad.66: an entry for **every** merge to `overnight-20260802`, clean or conflicted, so an
empty log never reads as "no conflicts" when it means "the logging never ran."

Each entry records: the cards involved, the files and hunks, the merge-intent notes read, the
resolution taken, and the gate result after it.

Pre-resolved by the slate, not findings:
- `sw.js` is GENERATED — never merge the artifact; take either side, regenerate with `task sw`
  **after** the merge commit (B-37 — `build-sw.js` reads git HEAD, not the working tree), then
  re-run G4 (idempotence + version parity + **file count against expectation**, not just exit 0).
- `version.go` — resolve **per-constant**, not per-file (precedent `79fa7cd`, 07-29).
- `build-sw.js` is the one real collision tonight: **B1 merges before P1 is cut.** Resolve against
  both merge-intent notes, never against text.
- `workflows.html`: **P2 merges before P3 is cut.** Same rule.

---

## Entries

### Merge 1 — `card/prestep-kr-artifacts` → `overnight-20260802`

**Result: CLEAN — zero conflicts.** Logged because every merge gets an entry, so an empty log
cannot be mistaken for "no conflicts" when it means "the logging never ran."

- **Cards involved:** the pre-step leg only (not a card). Nothing else had landed.
- **Files:** 6 changed, 609 insertions / 5 deletions, **all under `.night-crew/`** —
  `BACKLOG.md`, `roadmap.md`, `reference/card-actuals.md`, two new `designs/` notes, and the
  leg's merge-intent. No product code, no `sw.js`, no version constant, no `openspec/`.
- **Merge-intent read:** `merge-intent-prestep.md`. It declared `BACKLOG.md` as "ONE entry only"
  and landed as four touches (the hydration disposition plus three appended handles B-39/B-40/B-41
  and two pointers into the struck fetch-storm entries). The leg **struck the line in place** in
  its own note with what replaced it — disclosed, not silently diverged. Accepted: the extra
  touches are appends at the file tail and the struck entries, none of them a line another card
  tonight owns.
- **Resolution:** none required.
- **Gate result after it:** G1/G2/G4 **not applicable and not claimed** — the leg touches no code,
  so `sw.js` is unaffected and no precache regeneration is owed. Orchestrator verification instead:
  diff scope confirmed docs-only; all four commits carry `Night-Crew-Card:` / `Night-Crew-Run:`
  (checked with `git interpret-trailers --parse`); P-KR2's transcription checked against
  `ledger.md:1337` verbatim and the measured disposition string confirmed no longer `· new`.

**Shared-file note for later merges tonight:** `BACKLOG.md` and `roadmap.md` are now touched.
P1/P2/P3 flip their own `BACKLOG.md` entries and A1/A2/B1 flip their own `roadmap.md` bullets —
different lines from this leg's, so these are expected to stay clean.

---

### Merge 2 — `card/a1-sync-replication-scope-per-checklist` → `overnight-20260802`

**Result: ONE CONFLICT, resolved. `BACKLOG.md`.** Merge commit `2dc4eef`.

- **Cards involved:** A1 vs the already-merged pre-step leg.
- **Files and hunks:** `BACKLOG.md`, one hunk at the file tail (lines 744–754 pre-resolution) —
  HEAD carried the pre-step's `B-39`/`B-40`/`B-41`, the incoming side carried A1's `B-42`/`B-43`.
  `roadmap.md` **auto-merged clean** (80 insertions / 19 deletions); verified afterwards that both
  legs' edits survived — the pre-step's T-24 ruling text at `:1831` and A1's own card bullet.
- **Intents read:** `merge-intent-prestep.md` (BACKLOG touches are appends at the tail plus the
  struck fetch-storm entries) and `merge-intent-a1-…md` (BACKLOG is "append-only, one entry",
  later two after the G6 fix round filed B-43).
- **Resolution:** **kept BOTH sides, in numeric order.** Append-vs-append with no semantic overlap:
  the two sides describe five different findings and the renumbering had already de-collided them.
  Resolved against intent — both notes declare their BACKLOG writes as appends, so neither side's
  content is a candidate for dropping.
  🛑 **Why the renumber existed at all:** three legs tonight — the pre-step, A1 and B1 — each
  independently claimed `B-39`, because each read the same base. The pre-step merged first and
  keeps `B-39`/`B-40`/`B-41`; A1 was renumbered to `B-42`/`B-43` during its G6 fix round.
  **`B-44`/`B-45` are reserved for B1; A2 was told to start at `B-46`.** This is the
  duplicate-backlog-number failure of 2026-08-01 recurring under concurrency, and it is the
  orchestrator's to resolve — a card finding its number taken stops and reports, it does not
  renumber itself.
- **Gate result after it**, run by the orchestrator on the merged tree, not inherited:
  - **G1** — `go build ./...` **exit 0**; `go vet ./...` **exit 0**.
  - **G4** — `node build-sw.js` **after** the merge commit (B-37 ordering): **29 files precached,
    2130.2 KB**, count matches expectation. `git status` clean afterwards, so the card's committed
    `sw.js` was already correct at the merge commit — idempotent. Frontend version 1.4.0, no bump.
  - **Targeted G2** — `tests/sync-rxdb-client.spec.js -g "SCOPE-0"`, isolated
    (`TEST_PORT=8320`, `hq_n802_m1_e2e`/`_go`, own `TEST_OUTPUT_DIR`): **15 passed, exactly one
    summary block.** All 9 `[SCOPE-01]` and 6 `[SCOPE-02]`, including the F-3/F-4/F-5 regressions
    and `[SCOPE-02]`'s anti-vacuity control *"the SAME scope still RESUMES — the fix did not just
    disable checkpoints"*.
  - 🛑 **First attempt at this gate DID NOT RUN** — launched without creating its database, so the
    Playwright `webServer` exited 1 while the harness still reported exit code 0. Recorded because
    a gate that failed to run is indistinguishable from a gate that passed unless it is named.
    Full-suite G2 is deferred to the final merged tree.

**G6:** APPROVE WITH FINDINGS — 9 findings, **2 BLOCKING**, both fixed and re-proved before merge.
F-1 the checkpoint outlived the scope (one meta-store key spanned all scopes ⇒ opening an older
checklist replicated zero rows *permanently*); F-2 the fixture could not discriminate per-checklist
from per-template scoping (two unbounded-pull mutations survived 6/6 green). The fix round applied
both mutations again and showed each now goes red.

**PARK condition did not fire:** `sync-schema/` is byte-unchanged. `HQ_SYNC_REST_URL` appears only
in prose and is set nowhere.

🛑 **Carried forward for A2, P1 and Night B's S1 — A1's merge-intent, item 6:** the
`replicationIdentifier` **must carry the scope**. A merge that restores plain `hq-sync-${table}`
re-introduces silent data loss **and will look like a simplification, because the comment that
shipped with the card originally argued for exactly that.** The corrected comment must survive with
the code.
