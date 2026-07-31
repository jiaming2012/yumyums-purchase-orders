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

_(none yet — first merge appends here)_
