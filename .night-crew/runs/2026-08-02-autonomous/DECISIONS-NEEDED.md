# DECISIONS NEEDED — run 20260802

Forks this run could not resolve. **The run never decides a PRODUCT fork; it executes.**
Batch sign-off (2026-08-01) covers the signed specs and nothing else.

---

## For ratification at triage (decided by the slate, implemented by the run)

### R1 — B-13: the Taskfile is right, the doc is wrong

The slate DECIDED this rather than parking it, under the delegation principle.
`precache-manifest-from-head` made `build-sw.js` read **git HEAD**, and the prod clone's HEAD after
`git reset --hard origin/main` **is** the shipped tree — so regenerating `sw.js` on the box is
redundant by construction, and `sw.js` committed with its source is the correct contract.
Card P1 fixed CLAUDE.md to match `Taskfile.yml:178-190`.

**Ratify or reverse at triage.**

---

## Open forks

_(none yet)_

---

## Parked cards

_(none yet)_
