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
