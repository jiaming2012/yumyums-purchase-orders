# Backlog — advisory items that ride future cards

> Durable parking lot for triage-surfaced items that aren't roadmap cards yet but must
> survive run-to-run (HANDOFF is rewritten each run). Format: `title · description · origin ·
> status`. Promote to a roadmap card with `promoted → <card>`; drop with `dropped — reason`
> (struck through, kept as record).

- **Users stale-E2E repair** · `tests/users.spec.js` has two Access-tab tests navigating dead
  `#t3`/`#s3` DOM (removed in the 3-tab→2-tab refactor; Access now renders into `#s2`).
  Features work; tests can't run — marked UNPROVEN (stale-test), not BROKEN. Repoint
  `#t3`/`#s3` → `#t2`/`#s2`. Folds into the **Users Activity-4 prove-UNPROVEN WO** (low effort).
  · origin: triage 2026-07-10 (D-4) · new
