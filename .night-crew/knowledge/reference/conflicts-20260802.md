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

_(appended as merges happen)_
