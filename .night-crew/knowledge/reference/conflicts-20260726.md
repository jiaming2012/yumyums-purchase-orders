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

### 1. Card A → `overnight-20260726` — **CONFLICTED (1 file, non-code)**

- **Merge commit:** `3cbc650` · card branch `card/a-workflow-submission-status-client-half`
- **Cards involved:** card A only. Wave 0 ran alone, so the other side of this conflict was the
  **orchestrator itself**, not another card.
- **Files and hunks:** `.night-crew/runs/2026-07-26-autonomous/timings.log`, one hunk, lines 10–20.
  Both sides appended to the same append-only region: HEAD carried the orchestrator's impl/G6/land
  stamps, the card branch carried its three leg measurements (red-first, verify, gate).
  **No code file conflicted.** `sw.js` merged clean and needed no regeneration.
- **Intents read:** `merge-intents/a-workflow-submission-status-client-half.md` (card side). The
  orchestrator side has no merge-intent document by construction — its writes are the run's own
  bookkeeping, and the slate's surface table already rules `timings.log` **append-only, union on
  conflict, never drop a side**.
- **Resolution taken:** **union, chronologically ordered.** Neither side dropped. Both sets of
  stamps are present in the merged file.
- **One thing NOT laundered.** G6 finding **F1** (demonstrated): the card's gate line carries epoch
  `1785029700` (01:35:00Z) while the commit that introduced it, `c70581c`, has committer time
  `1785027271` (00:54:31Z) — the stamp postdates its own commit by 2429 s, so it is not a
  measurement. It was **kept in place and annotated**, not silently rewritten: the ledger is
  Delivery KR3's forecasting input, and quietly correcting a bad stamp is how a ledger stops being
  evidence. The `wall=732s` figure IS a real measurement and is retained; its derived minutes were
  also wrong (732 s = 12.2 m, not 13.0 m) and are corrected inline. **Delivery KR3 should read the
  wall figure for this leg, not the epoch.**
- **Gate result after resolution:** the conflict was confined to a documentation file, so a full
  suite re-run would have cost ~13 min for zero information. Verified instead by construction:
  `git diff card/a-... HEAD -- workflows.html tests/workflows.spec.js night-crew.toml sw.js backend/`
  → **empty**, proving the merge altered no code. Card A's own gate stands as the evidence:
  **164 passed / 1 skipped / 0 failed / 0 flaky of 165** on the post-fix 4-spec seam subset, plus
  G6's independent re-run of all four acceptance tests green at `--retries=0` in its own stack.
- **HARD constraints re-verified post-merge:** `git diff --stat dev HEAD --` on `backend/go.mod`,
  `package.json`, `package-lock.json`, `docker-compose.nc.yml`, `Taskfile.yml` → **empty.**

