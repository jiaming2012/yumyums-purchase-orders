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

---

### 2. Card C → `overnight-20260726` — **CONFLICTED (1 file, non-code)**

- **Merge commit:** `a848189` · card branch `card/c-sync-rxdb-browser-delivery-spike`
- **Cards involved:** card C and the orchestrator. Card C merges **before** card B, per the slate —
  C's `build-sw.js` change is the one with reach.
- **Files and hunks:** `.night-crew/runs/2026-07-26-autonomous/timings.log`, one hunk. HEAD carried
  the orchestrator's per-card stamps; the card branch carried card C's own leg ledger (bundle
  generation, the five legs, the exp-leeway probe, the 47-minute gate, its RUN-10 attribution legs,
  and the post-G6 repair round).
  **`DECISIONS-NEEDED.md` auto-merged clean** — card C appended D-7…D-11 below D-1…D-3 and
  deliberately started at D-7, leaving room for card B's D-4/D-5/D-6 which had not yet landed. That
  foresight is why the numbering survived two merges without a renumber.
  **No code file conflicted.**
- **Intents read:** `merge-intents/c-sync-rxdb-browser-delivery-spike.md`. Note that card C repaired
  this document's own verification command during its post-G6 round (G6 finding F7): as originally
  written the `git diff --stat` pathspec `'*.html'` matched at any depth and reported
  `browser/spike.html`, which would have shown an orchestrator an apparent production-HTML violation
  at exactly this step. The corrected command (`:(top,glob)*.html`) prints nothing, and it was run.
- **Resolution taken:** **union**, both sides retained, neither dropped.
- **Gate result after resolution:** conflict confined to a documentation file. Verified structurally
  instead of by re-running a 47-minute suite:
  - `git ls-tree -r HEAD -- vendor/` → **exactly 5 files**. `vendor/node_modules` (8,919 files /
    67 MB) is present on disk and **correctly NOT committed**.
  - HARD constraints vs `dev` → **empty**.
  - Card C's own gate stands: **540 passed / 1 failed / 2 flaky / 6 skipped of 549**, plus G6's two
    independent 11/11 harness runs and its byte-identical cold rebuild of the bundle.

#### 🛑 A generated-file finding surfaced AT this merge, and it is not card C's

Per the standing rule, `sw.js` is generated and is never hand-resolved. It did not conflict, but the
merge is the right moment to confirm the generated output is consistent — so `node build-sw.js` was
re-run. It produced **24 files / 2166.8 KB** against the committed **23 files / 1947.1 KB**.

The extra entry is **`backlog-round.html` — an UNTRACKED file sitting in the repo root since before
this run began** (it is in the run's very first `git status`). `build-sw.js` globs the working tree,
not the index, so **any untracked HTML in the repo root silently enters the precache manifest.**

**The regenerated output was DISCARDED and the committed `sw.js` restored.** Committing it would
have shipped a precache entry for a file that exists on no other machine — and a Workbox precache
entry that 404s fails the whole service-worker install, not just that asset.

This is filed as **D-12** (orchestrator finding). It is a live foot-gun for anyone who runs
`task sw` with an untracked page present, which is the normal state of a work-in-progress tree.

---

### 3. Card B → `overnight-20260726` — **CONFLICTED (1 file, non-code)**

- **Merge commit:** `566130a` · card branch `card/b-sync-jwt-bridge-endpoint`
- **Cards involved:** card B and the orchestrator. Merged **after** card C, per the slate.
- **Files and hunks:** `.night-crew/runs/2026-07-26-autonomous/timings.log`, one hunk — the same
  append-only collision, orchestrator stamps vs card B's leg ledger.
  **`DECISIONS-NEEDED.md` and `roadmap.md` both auto-merged clean.** The decision numbering came
  through intact end to end: **D-1…D-3** (card A's G6), **D-4…D-6** (card B), **D-7…D-11** (card C),
  verified by reading the merged file rather than assuming.
  **No code file conflicted** — cards C and B share literally zero code files, which is exactly the
  property that made concurrent dispatch safe tonight, and it held.
- **Intents read:** `merge-intents/b-sync-jwt-bridge-endpoint.md`, including its disclosed late
  addition of two files outside the declared footprint (an 11-line comment-only change to
  `tests/grant-enforcement-parity.spec.js`, and the new captures directory).
- **Resolution taken:** **union**, both sides retained, neither dropped.
- **Gate result after resolution:** card B's gates stand — `go test ./... -count=1 -p 1` all packages
  pass, the 16-variant attack suite green with the 9-variant red reproducible, and the full
  Playwright suite at 541/2/6. G6 independently reproduced both the red and the green.
- **HARD constraints re-verified on the final merged tree:** `backend/go.mod`, `package.json`,
  `package-lock.json`, `docker-compose.nc.yml`, `Taskfile.yml` → **empty diff vs `dev`.** All three
  cards held the line; card C's `npx esbuild@0.28.1` choice and card B's stdlib-only mint are the
  two decisions that made that possible while running concurrently.

