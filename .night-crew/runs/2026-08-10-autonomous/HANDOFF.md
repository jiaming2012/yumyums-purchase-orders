# HANDOFF — run `20260810`

**Slate:** `reference/slate-20260810.md` (signed 2026-08-09, operator "sign off"). 2 cards.
**Run branch:** `overnight-20260810` (off `dev` @ `03c3e06`). **NOT merged to `dev`** — that is
morning triage's act.
**Executed:** attended-evening 2026-08-09, closeout written **2026-08-09 11:52 AST**. No `--night`
cut line; both cards fit comfortably.
**Dispatch:** serial (forced — Card 2 depends on Card 1), subagent-per-card in worktrees; control
loop = orchestrator; a separate fresh-context G6 subagent per card. Temporal was down the whole run
(this repo uses Claude-Code subagent dispatch, NOT the Temporal queue — so no poller could intercept
work; "nothing left polling" holds by construction: a down server has zero pollers).

## Outcome: BOTH cards landed. 🟢🟢

`sync-live-in-dev` (Activity 5, the milestone close-bar work) is delivered: the RxDB sync capability
now **runs persistently in the operator's dev environment and is usable in the app**, with the same
round trip proven **red-first and automated** (Card 2). Per the slate this night does **not** close
the milestone and cannot by design — the close bar is the operator personally opening `workflows.html`
in `dev:tailscale`, seeing a field sync in the app, and recording the ledger line
(`dev-complete-attestation`, attended-by-design). After tonight the milestone is left **one card
short** of close: `dev-complete-attestation` (the operator's own act), plus the separately-tracked A3
attended re-gate (`gate-rls-fixture-ownership`, decision 155 — Activity-1 gate hygiene, not the sync
close bar).

## Per-card outcomes

| # | Card | Branch | Merge | Verdict | G6 | Notes |
|---|---|---|---|---|---|---|
| 1 | `sync-live-in-dev-substrate` (Activity 5, legs 1+2 + FDW persistence) | `wo-sync-live-in-dev-substrate` | `bd03059` (clean) | **DEV-COMPLETE** — door 503→200 (vars unset→set); relay carries a `/saveResponse` write into the substrate in ~267ms; both done_when GREEN | **PASS-WITH-ISSUES** → must-fix applied (`167bc7e`), no fix round needed | Persistent `sync:dev:*` task family, 4× `HQ_SYNC_*` in dev targets **only** (proxy.go:78 guard honored), FDW→HQ pointing SQL, relay-as-service bring-up + `sync-dev-proof.sh`. G6 reproduced both done_when items. |
| 2 | `sync-live-in-dev-app-proof` (Activity 5, leg 3) | `wo-sync-live-in-dev-app-proof` | `489145e` (clean) | **DEV-COMPLETE** — real `workflows.html` (no stub) surfaces a `/saveResponse` field via RxDB in the app (`#sync-one-row → served`, ~454–508ms); SAME spec fails relay-down | **PASS** (no fix round) | Standalone red-first harness `sync-app-proof.sh` (form (a), gated on its own exit — B-345-aligned). `night-crew.toml` = comment-only footprint note (no key/token → no park). **No `workflows.html` edit needed** (spike prediction held). G6 independently re-ran the asymmetry. |

## Gate evidence — on the final merged tree (`overnight-20260810` @ `489145e`)

- **G1:** `go build ./...` exit 0, `go vet ./...` exit 0 (0 warnings), from `backend/` — verified by
  execution on the merged tree.
- **G2(Go):** `go test -p 1 -count=1 ./...` on `:5434`/`hq_test_go` → exit 0, **9 packages `ok`, 0
  FAIL**; DB-coupled tests genuinely ran (`internal/workflow` 1.18s, `internal/sync` 22.7s);
  `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` attested **UNSET**. (G6 verified the `-v`
  counts on Card 1's branch: workflow=35, sync=59 subtests.)
- **G2(Playwright):** **N/A-by-footprint** — no `tests/*.spec.js` changed; no `[e2e.seams]` key
  matches any changed path. Card 2's app-proof is a **standalone** harness (outside `tests/`) whose
  own exit code IS its verdict; `tests/repo-hygiene.spec.js` count unchanged at 11.
- **G3:** **N/A** — `openspec: absent` (universal per-change mechanics only; no OpenSpec/GSD
  scaffolding created).
- **G4:** **N/A-by-footprint** — neither card changed any HTML/JS/`sw.js`/`sync-rxdb`/`manifest.json`;
  committed `sw.js` untouched, precache count **31**, tree clean.
- **G4 discipline greps:** **N/A-VACUOUS** — neither `internal/journal` nor `internal/workorder`
  exists in this repo (B-14). (Not `clean`, not `PASS`.)
- **RF:** Card 1 — door 503(unset)→200(set) captured red-first; Card 2 — relay-up pass / relay-down
  fail in one invocation, gated on the script exit (B-163). Both independently re-reproduced by G6.
- **G6:** Card 1 PASS-WITH-ISSUES (must-fix applied), Card 2 PASS. Both reviewers re-ran the
  deliverable and could not break it. Full verdicts: `c1-g6-review.md`, `c2-g6-review.md`.

## Conflict log — `reference/conflicts-20260810.md`

**2 merges, both CLEAN, both logged.** Merge 1 (Card 1): first card, zero divergence. Merge 2 (Card
2): branched off the post-Card-1 tree, so its `Taskfile.yml` (`sync:app-proof`) and `roadmap.md`
(leg-3 flip) sit on disjoint stanzas from Card 1's — no collision. No conflict resolution required.

## Environment left clean

Substrate up (reconciled, never destroyed — `spike:down` never run); FDW `hq_pg` restored to
`:5434/hq_test_b2_fdw`; zero `spikec-`/`appproof-` rows left in the substrate; no scratch-HQ
containers leaked. `hq_test_go`/`hq_test_e2e` were (re)created on the `:5434` test cluster where
absent — a test-cluster fixture, not a repo change. **No `:5433` command ran the entire night** (B-164).

## Next actions (morning triage)

1. Review this run branch on its merits; merge `overnight-20260810` → `dev` (triage's act).
2. No forks to resolve — `DECISIONS-NEEDED.md` carries **no parks**, only advisory items.
3. **The milestone is now one attended act from close:** `dev-complete-attestation` — the operator
   opens `workflows.html` in `dev:tailscale` against the persistent substrate (`task sync:dev:up`),
   sees a field sync in the app, records the ledger line. Card 2 proves the same round trip red-first
   and automated, so the capability it attests is LIVE. (Plus the separately-tracked A3 re-gate.)
4. Consider the backlog candidate in `DECISIONS-NEEDED.md` (bare `npx playwright test` in the spike
   scripts — a foreign-PATH-binary vulnerability that bit Card 2's setup twice; Card 2's own harness
   is already hardened against it).
