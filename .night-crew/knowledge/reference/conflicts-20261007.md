# Conflict log — run `20261007`

One entry per merge onto `overnight-20261007`, clean or conflicted (§15ad.66).

## Launch record (2026-10-07 ~01:00–01:20 America/New_York)

- Branch `overnight-20261007` cut from `dev@c549c51`. No unmerged `overnight-*` branch.
- Launch checks: binary preflight satisfied (v3.6.1+140, built from night-crew `8c42997`);
  `next check /nc-run` allowed; `run-evidence check --run 20261007` → `no-run-evidence`;
  workflow preflight `openspec: absent`; `routing check` agrees; spike gate — four goals slatable.
- **Stranded work named by the checks, not re-asked — each already carries the operator's ruling:**
  `card/a3-rls-fixture-own` and `card/s2-demo-sync-target` (leave in place, ruled 2026-09-05,
  B-442, re-confirmed at triage 2026-10-05 and 2026-10-06); the detached worktree
  `hq-worktrees/verify-e1-20261006` at `fe45768` (last night's parked `identity-code-and-qr`
  build — triage 20261006 flag: armed, do not delete). Nothing was excluded with `--expect`.
  Untouched tonight.
- Precache count re-counted on the merged `dev`: **51**.
- `env-up.sh` reconcile: `EXIT=0`, GREEN. It stalled ~10 min first: `docker info` hangs in a
  Docker Desktop CLI plugin (`docker-ai` → `powershell.exe`). I ended that helper process (a child
  of my own call) and the script continued. Sessions are told not to call `docker info`.
- **Own scheduler lane again** (Temporal namespace `night-crew-hq`): the night-crew clone signed
  its own slate `20261007` today and would share the run number on the shared lane (B-494). Both
  lanes had no pollers at launch; a rehearsal worker of the night-crew clone (pid 8962) is on the
  rehearsal lane and is not ours. hq's worker: pid 22298, built from night-crew `8c42997`.
- One `night-crew run --run-id 20261007 --max-concurrent 1 --deadline 8h`, four `--wo` in slate
  order, submitted 01:18.
- **Card 4 lifted NARROWED** per operator decision 214 (triage receipt 20261006): only the
  throwing-policy refusal gate (B-482); the photo-pick feedback half (B-483, `[PS-03]`) is not
  built. Cards 1–3 lifted verbatim.
- **Card 3 Context carries triage's B-493 note**: the second states spec
  (`states-marketing-subscribers.spec.js`, `h5/states`) gets the same change.
- Loop gates are compile + vet (fenced); every card's Gates text is copied into its Context (B-492).

## Merges

### Merge 1 — Card 1 `inventory-setup-races` → `overnight-20261007` at `b9d7b62` (03:5x)

- **Source:** loop commit `3fb748a` (`night-crew/20261007/inventory-setup-races`), `--no-ff`. **Clean — no conflict.**
- **Loop record:** built 01:20–02:16; the loop's own review ran two rounds (16 findings, then 7),
  patched twice, approved "review cycle exhausted — last patch re-gated, not re-reviewed".
- **G6 (separate adversarial reviewer, own worktree `hq-worktrees/g6-k1-20261007`): PASS.**
  `[IS-01]`–`[IS-04]` 4 passed; `[IS-01]` at 1500 ms passed; parent's `inventory.html` → 4 failed;
  each single mutation reddened its test (sequence guard → `[IS-02]`; add-bar rebuild → `[IS-01]`;
  alert → `[IS-03]`; both halves of the nickname fix undone → `[IS-04]`). No stub beyond timing.
  The un-reviewed patch was probed by running: photo → nickname → Save kept the new photo; a star
  tapped during an add was kept. Whole `tests/inventory.spec.js`, no retries: parent 20 failed /
  199 passed, change 15 failed / 208 passed; **red on change and green on parent: none**.
- **G6 findings, none blocking (for triage):** `[IS-01]` does not pin "selected group kept when
  groups land" (mutation stayed green); the in-place photo-error swap is not pinned alone (only
  slower without it); behaviour wider than the card's four items — every field of an open item
  editor now survives a re-render, a `#newItem=` prefill is applied once per Setup open.
- **After merge:** `sw.js` regenerated at the merged HEAD, committed `d5946cf`, count 51, a second
  regeneration left the tree clean. `backlog check` exit 0 (292 entries); B-459, B-478 → `landed`.
- **Not a card edit I expected:** the loop appended its deferred review findings to
  `.night-crew/knowledge/ledger.md` as `## LDG …` entries (70 lines). Kept as merged; flagged for triage.
- **Owed on this tree:** Card 1's full browser suite and the 10×/10×/5× measurement leg (running next).

### Merge 2 — Card 2 `dish-merge-shapes-and-backstop-tests` → `overnight-20261007` at `05c9ccd` (04:1x)

- **Source:** loop commit `bc705a5`, `--no-ff`. **Two conflicts, both in documents, both resolved by intent:**
  - `.night-crew/knowledge/roadmap.md` — Card 1's line: my side carried the merge SHA I added
    after Merge 1, the card's side the bare `LANDED`. Kept mine (the SHA is the later fact).
  - `.night-crew/knowledge/ledger.md` — the loop's appended `## LDG …` entries for Card 2 against
    my side's end of file. Kept both (pure append; Card 1's entries were already present).
  - **Why it conflicted at all:** the loop lands each card on its own run branch as a squash commit
    (`d9a2234`), so Card 2's history does not contain Card 1's commit `3fb748a` that I merged — the
    same text arrived twice by two routes. No code file conflicted. Cards 3 and 4 will be brought
    over as their own commit's change only (cherry-pick), to avoid replaying this.
  - Card 2's merge-intent note read; it names no shared code file with Card 1.
- **Loop record:** built 02:51–03:17; two review rounds (12 findings, then 4), patched twice,
  approved "cycle exhausted — last patch not re-reviewed".
- **G6 (separate reviewer, `hq-worktrees/g6-k2-20261007`): PASS.** recipes 65/0/0, marketing 73/0/1.
  Mutations: ` FOR SHARE` removed → lock test FAIL 3 of 3; 0086 `NO ACTION` on a rebuilt database
  → backstop test FAIL (23503); source check removed → 200 `rows_re_pointed:0`, test FAIL; uuid
  parse removed → `BadIDIs400` FAIL; each handler arm's status swapped → its test FAIL. Lock test
  unmutated 10 of 10. Full Go on the card: exit 0, 461 pass / 0 fail / 3 skip, 15 packages (base 456).
  No migration; `go.mod` gains `github.com/google/uuid v1.6.0` as a new direct requirement,
  `go mod tidy` leaves it unchanged.
- **G6 findings, none blocking (for triage):** dish ids without hyphens or in braces, which the
  database used to accept, now answer 400 `bad_id` (deliberate, pinned); the lock test's
  foreign-key branch never runs on this schema; opposite-direction simultaneous merges can
  deadlock and answer 500 (read, not run). **Loop-deferred, major:** merging two dishes that share
  an ingredient answers 500 (existing; `recipes` unique key) — in `ledger.md` as an `LDG` entry.
- **After merge (G1+G2 re-run on the merged tree):** `go build ./...` and `go vet ./...` exit 0;
  `sw.js` regeneration left the tree clean (no precached file touched), count 51; `backlog check`
  exit 0; B-484, B-485, B-487 → `landed`.

### Merge 3 — Card 3 `states-screenshots-out-of-tree` → `overnight-20261007` at `e5563ab` (04:4x)

- **Source:** loop commit `45f42a4`, brought over as its own change only (cherry-pick, for the
  reason recorded under Merge 2). **Clean — no conflict** (`roadmap.md` auto-merged). The three
  code files are byte-identical to the loop commit's.
- **Loop record:** built 03:32–03:52; two review rounds, patched twice, approved "cycle exhausted".
- **G6 (separate reviewer, `hq-worktrees/g6-k3-20261007`): PASS.** Stats spec alone 17 passed,
  subscribers spec alone 10 passed, `git status --porcelain` empty and no diff under
  `runs/2026-10-02-autonomous/logs/` after each; marketing seam 131 passed, tree still clean.
  Mutation: defaults pointed back at `h4/states` / `h5/states` → `[SS-01]`, `[SS-02]` FAIL and 18
  committed PNGs modified (then restored in the reviewer's worktree). `STATES_SHOT_DIR` honoured.
  No PNG in the commit; `night-crew.toml` is 8 added comment lines; row counts unchanged (16, 9).
- **Covers both directories** (B-486 `h4`, B-493 `h5`) — the triage launch note.
- **G6 / loop notes, none blocking (for triage):** with `STATES_SHOT_DIR` set, all three marketing
  states specs write into one flat folder; the guards check only the default directory. The loop's
  review also recorded that 10 of the 36 committed H4/H5 PNGs were already re-captured by run
  20261002's closeout commit (B-467's subject) — in `ledger.md` as an `LDG` entry.
- **After merge:** `sw.js` regeneration left the tree clean, count 51; `backlog check` exit 0;
  B-486 → `landed`. **The B-486/B-493 checkout rule is moot from this merge on.** Legs before it:
  the base leg and Card 1's full suite (both restored by hand). Legs after it: the final suite.

### Merge 4 — Card 4 `scanner-refusal-seam-and-pick-feedback` → `overnight-20261007` at `16090f7` (05:1x)

- **Source:** loop commit `cbf5598`, brought over as its own change only (cherry-pick).
  **Clean — no conflict** (`roadmap.md` auto-merged). Code files byte-identical to the loop commit's.
- **Built NARROWED** (operator decision 214): the policy-source override and `[SV-12]` only. No
  `#scan-note`, no `SCAN_STATE.note`, no `[PS-03]`, `onFilePicked` untouched — confirmed by G6's grep.
- **Loop record:** built 04:22–04:34; two review rounds, patched twice, approved "cycle exhausted".
- **G6 (separate reviewer, `hq-worktrees/g6-k4-20261007`): PASS.** `[SV-12]` 9 clean runs, 9 green
  (one further run was invalidated by the reviewer's own overlapping mutation and discarded).
  Mutations: parent's `scan-page.js` → FAIL (the offer rendered, the installed source never asked);
  the installed source made non-throwing → FAIL; the catch at `submit-flow.js:103` made permissive
  → FAIL. Whole `tests/marketing.spec.js`, no retries: parent 70 passed, change 71 passed.
  `marketing/submit-*.js` diff empty. The test drives `MarketingScan.scanText`, not the photo control.
- **G6 findings, none blocking (for triage):**
  - **The override is not limited to test environments** (low–medium): any same-origin script that
    sets `window.__MARKETING_POLICY_SOURCE__` before boot can turn refusals into offers for that
    page load. The card's signed spec asked for exactly this read; an existing post-boot setter
    already allows the same. Cheap closes: gate the read on a localhost origin or a test-server flag.
  - `[SV-12]` also routes `/api/v1/marketing/redeem` as a tripwire (never answers) — outside the
    slate's named stub list, low. Its closing "belt" drives the machine directly, not the screen.
- **After merge:** `sw.js` regenerated at the merged HEAD, committed `827ee37`, count 51, second
  regeneration clean. `backlog check` exit 0; B-482 → `landed` (B-483 was already dropped at triage).
- **Roadmap text:** the K4 line still describes the withdrawn photo-pick half; left as authored
  (append-only), the narrowing is stated here and in the backlog line.
- **Owed on this tree:** the FINAL full browser suite and the full Go suite (queued behind Card 1's
  measurement leg, which is still running on Card 1's merged tree).
