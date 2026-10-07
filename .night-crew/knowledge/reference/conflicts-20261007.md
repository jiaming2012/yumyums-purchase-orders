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
