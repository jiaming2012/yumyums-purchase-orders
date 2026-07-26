# Conflict log — run `overnight-20260726`

Per §15ad.66: **every merge to the run branch gets an entry here, clean or conflicted.**
Clean merges get a one-line entry, so an empty log can never read as "no conflicts" when
what it actually means is "the logging never ran."

Each entry records: the cards involved, files and hunks, the `merge-intent.md` documents
read, the resolution taken, and the gate result after it.

**Standing rule for this repo:** `sw.js` is a GENERATED file. Never hand-resolve a conflict
in it — take either side, re-run `node build-sw.js` (or `task sw`), commit the regenerated
output.

---

## Merges

_(appended as they happen)_
