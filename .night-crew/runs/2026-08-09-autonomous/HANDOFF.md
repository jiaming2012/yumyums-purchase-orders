# HANDOFF — run `20260809`

**Slate:** `reference/slate-20260809.md` (signed 2026-08-08 evening, "Yes — 2 cards").
**Run branch:** `overnight-20260809` (off `dev` @ `bdf9f5a`). **NOT merged to `dev`** — that is morning triage's act.
**Executed:** attended-afternoon 2026-08-08, ~13:39 → ~14:31 AST (no `--night` cut line; ~52 min wall-clock).
**Dispatch:** serial, subagent-per-card in worktrees (control loop = orchestrator). Temporal was down the whole
run; this run uses Claude-Code subagent dispatch, NOT the Temporal queue — so no poller could intercept work,
and "nothing left polling" holds by construction (a down server has zero pollers).

## Outcome: BOTH cards landed. 🟢🟢

The milestone close-bar deliverable (`task demo:sync`) now EXISTS and runs GREEN. Per the slate, this night does
**not** close the milestone and cannot by design — the close bar is the operator personally running `task demo:sync`
and seeing it pass (`dev-complete-attestation`, attended-by-design). After tonight, the milestone is left **one card
short** of close: `dev-complete-attestation` (the operator's own act), plus the separately-tracked A3 attended
re-gate (`gate-rls-fixture-ownership`, decision 155 — Activity-1 gate hygiene, not the sync close bar).

## Per-card outcomes

| # | Card | Branch | Merge | Verdict | G6 | Notes |
|---|---|---|---|---|---|---|
| 1 | `demo-sync-target` (Activity 5, close-bar) | `card/c1-demo-sync-target` | `0fade6b` (clean) | **GREEN** — round-trip 117 ms via real `POST /saveResponse` → NOTIFY relay → PostgREST → RxDB read; tri-state exits 0/1/2 all captured distinct (red-first) | **PASS-with-findings** (no fix round) | Read surface = Node RxDB replication client, NOT the app UI → operator-awareness item (see DECISIONS-NEEDED). `demo-sync.sh` is a documented re-export of `spike-c-roundtrip.sh`. |
| 2 | `spike-exit-code-honesty` (B-163) | `card/c2-spike-exit-code-honesty` | `3e6cd5c` (clean) | **DONE** — 3 exit-code conflations fixed, all proven red-first 1→2 (no live stack) | **PASS** (no fix round) | Found **seven** unguarded `srcpsql` substitutions (card cited four) — all guarded. No over-correction: `die(RED)` still exits 1, green still 0. B-163 → RESOLVED. |

## Gate evidence — on the final merged tree (`overnight-20260809` @ `3e6cd5c`)

- **G1 / G2(Go):** **N/A-by-footprint** — zero `.go` files changed across the entire run (card 1 diff: demo-sync.sh,
  Taskfile, roadmap, merge-intent; card 2 diff: spike-e scripts, Taskfile, BACKLOG, merge-intent). Nothing to build/vet/test.
- **G2(Playwright):** **N/A-by-footprint** — no `[e2e.seams]` key in `night-crew.toml` matches any changed path;
  `workflows.html` was read/driven only (no source edit); the spike scripts and `spike:`/`demo:` targets are exercised
  by no spec. No seam fires. Each card's own run (demo tri-state / exit-code probes) IS its verdict.
- **G3:** **N/A** — `openspec: absent` (universal per-change mechanics only; no OpenSpec/GSD scaffolding created).
- **G4:** **precache 31**, `node build-sw.js` exit 0, import-reachability 0-outside — verified by execution on the
  merged tree (`3e6cd5c`). No served/precached asset changed by either card; `sw.js` byte-identical to committed
  (benign workbox-toolchain regen noise reverted, not shipped — the standing artifact, not B-37).
- **G4 discipline greps:** **N/A-VACUOUS** — neither `internal/journal` nor `internal/workorder` exists in this repo (B-14).
- **RF:** card 1 — tri-state 0/1/2 captured red-first; card 2 — three conflations each proven wrong-then-right (1→2).
  Both independently re-reproduced by G6.
- **G6:** card 1 PASS-with-findings, card 2 PASS. Both adversarial reviewers re-ran the deliverable and could not break it.

## Conflict log — `reference/conflicts-20260809.md`

**2 merges, both CLEAN, both logged.** Merge 1 (card 1): first card, zero divergence. Merge 2 (card 2): branched off
the post-card-1 tree, so its `Taskfile.yml` note sits on a disjoint stanza from card 1's `demo:sync` — no collision.
No conflict resolution was required on either.

## Parked

**Nothing parked.** No PARK trigger hit on either card. See DECISIONS-NEEDED.md for the one operator-**awareness**
item (the read-surface choice) — it is a "know before you attest" note, NOT a blocking fork.

## Next actions (for the morning reader — the operator)

1. **`/nc-morning-triage`** — review this run branch, merge `overnight-20260809` → `dev`. Both merges are clean; the
   conflict log (2/2 clean) and both G6 reports are committed under `.night-crew/runs/2026-08-09-autonomous/`.
2. **The milestone close is now ONE attended act away.** Once triaged to `dev`, run **`task demo:sync`** yourself in dev,
   see it pass, and record the outcome in `ledger.md` (`dev-complete-attestation`). That line closes the milestone —
   no card, grade or closeout substitutes for it.
   - 🛑 **Before you attest, read the read-surface note (DECISIONS-NEEDED item 1).** `task demo:sync` proves the
     data-plane round trip (real `/saveResponse` → substrate → RxDB replication) but reads through a **Node RxDB client,
     not the app UI**. It clears the close-bar *letter*; decide consciously whether your definition of "the sync
     capability running in my dev environment" is satisfied without an app-surface read, or whether you want a
     follow-up card to drive the real browser fill-view against the real substrate.
3. **Separately tracked (not the sync close bar):** `gate-rls-fixture-ownership` (A3) remains BLOCKED on an attended
   re-gate (decision 155). Untouched by this run.
4. **`task demo:sync:red` / `task spike:reconnect:red`** both exit **201** (go-task's code), not the script's — gate
   any red-first capture on the SCRIPT directly (both cards document this; it is B-163's lesson).
