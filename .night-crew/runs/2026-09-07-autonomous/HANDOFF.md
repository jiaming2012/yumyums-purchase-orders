# HANDOFF — run 20260907 (night of 2026-09-06)

**Slate:** `slate-20260907.md`, signed 2026-09-06. 1 card, Activity B.
**Branch:** `overnight-20260907`, cut from `dev` @ `db4bcd6`. Never pushed; `main` untouched.
**Outcome: the card landed first-pass. Activity B is closed — every one of its six cards is
now LANDED — and close-bar leg 3 is genuinely attestable for the first time (decision 178's
blocker is gone). The attestation itself is the operator's attended act.**

## Per-card outcomes

| # | Card | Outcome | G6 | Merge | Branch |
|---|---|---|---|---|---|
| 1 | `sync-coordinates-provisioning` | DONE — provisioning writer on page init (mint → SYNC_KEY → startSync), push replica wired with `deviceId = sub`, B-439 latch clears on the HTTP-200 pull edge, B-438 discharged as a consequence. All four done_when clauses proven UNSTUBBED | **APPROVE, first pass** — all four named stub-seams attacked and confirmed by independent re-execution | `6f3ca30` (clean, zero conflicts) | `card/sync-coordinates-provisioning` (worktree preserved at `hq-worktrees/c1-sync-coordinates-provisioning`) |

## The four stub-seam questions (the reason this G6 was pointed, answered with evidence)

- **(a) Clause-1 e2e provisions through the SHIPPED path** — the test only ever *reads*
  `hq_marketing_sync_v1` (polls `getItem`; no `setItem`, no fixture seeds it); the shipped
  `provisionSync()` chain does the writing. G6 re-ran the 6-test set independently: EXIT=0.
- **(b) Clause-3 fails campaigns at the NETWORK layer** — `route.fulfill({status:503})` on the
  campaigns pull; `setCampaignPolicy` appears only in a pre-existing describe from the prior
  card. Healthy control asserts against the real attached source.
- **(c) Spike 04 re-run wrapper-free, BOTH shapes** — the harness imports the shipped
  `startCampaignsReplica`/`createCampaignPolicySource` and gates only `fetch`; sha256-pinned to
  the pre-fix/post-fix trees. G6 re-ran it: EXIT=0, with-docs AND recovery-EMPTY clear, and
  independently replayed the recovery-EMPTY keyset pull at live PostgREST → `[]` (zero rows,
  latch still cleared — the witness is the HTTP-200 edge, nothing docs-based).
- **(d) `deviceId` = mint `sub`, no device-local fallback** — envelope without `sub` → nothing
  written, push does not start; the only `randomUUID` is the attempt-row id. The e2e asserts
  the landed `device_id === sub` off the push transport.

## Gate evidence on the final tree

| Gate | Result |
|---|---|
| **G1** | `go build ./...` and `go vet ./...` from `backend/` EXIT=0. ⚠️ The implementer's committed logs (`c1-g1-build.log`, `c1-g1-vet.log`) are **0-byte** — an evidence hole; G6 closed it by re-running both (EXIT=0). Discipline note below |
| **G2 (Go)** | **NOT RUN — correctly**: `backend/` untouched (verified by diff-stat at implement, G6, and merge). `HQ_SYNC_SUBSTRATE_OPTIONAL` and `HQ_SYNC_GATE_CHILD` unset on every leg regardless |
| **G2 (Playwright)** | Full suite (de-confined by `sw.js`, priced in the slate): **866 passed / 4 failed / 6 skipped, 29.2m, EXIT=1, exactly one summary block** (`c1-g2-full.log`). Reds: SYNC-FC-01, SYNC-RF-01, SYNC-RF-02 = baseline; **B1-XT-01 PASSED (second consecutive run); FILL-04 red outside baseline** — see Findings. Marketing standalone: 39 passed EXIT=0 (`c1-g2-marketing.log`) |
| **G3** | N/A — `openspec: absent` (preflight re-run at launch, exit 0). No scaffolding created |
| **G4** | Re-run at **merged HEAD `6f3ca30`**: `task sw` EXIT=0, tree clean on regeneration (idempotent), **43 precached**, version parity `1.6.2` ≡ `1.6.2` ≡ `1.6.2` |
| **RF** | Satisfied — red commit `bf433e4` precedes fix `74f27f1`; Playwright red 6/6 failed on the pre-change tree (`c1-red-provisioning.log` — the poll on `SYNC_KEY` receives null, the slate's named structural red); harness red EXIT=1 stuck-`true` with captures advancing (`c1-red-recovery-clear.log`). G6 verified order and content |

**Post-merge on the final tree:** `tests/marketing.spec.js` — **39 passed, EXIT=0**, one
summary block (`c1-postmerge-marketing.log`).
**Spike-04 comeback:** wrapper-free re-run against the shipped clear GREEN EXIT=0 in both
recovery shapes (`c1-recovery-clear-green.log`); the *unmodified* spike script now exits 1 at
its own "measuring a ghost" check (`c1-spike04-rerun-postfix.log`) — the inverse measurement.
`validated:` line appended to the goal ledger Comebacks (`c6a34ed`).
**Build-fact-6 call (decided, stated):** network-layer route interception for the e2e
(mint + `/sync/rest/*` served/failed at the transport, decision-174-consistent; no seam
stubbed, no test writes SYNC_KEY); the full-stack proof lives in the harnesses
(spikes 01–03 + the new committed `marketing/sync/harness/recovery-clear-run.sh`).
**Conflict log:** `reference/conflicts-20260907.md` — one merge, clean, logged.
**Scorecard record:** `scorecard/20260907.jsonl` emitted at closeout; `night-crew scorecard
--repo .` **EXIT=0**, run 20260907 rendered, all four roles record-backed. Points follow the
20260906-2 single-risk-flagged-card convention (4) rather than the template's no-points
default (2), for trend comparability; stated rather than silent.
**Nothing left polling:** `night-crew workers check` re-run at 13:08 UTC — queues
`night-crew`, `night-crew-env` clear, no pollers. (Poller-TTL caveat: this is what the check
reported at that time.)
**G4 discipline greps:** **N/A-VACUOUS — neither package exists in this repo (B-14).** Not
"clean", not "PASS".
**Decisions resolver:** zero questions routed through `night-crew decisions log` — no
delegated decides, nothing awaiting ratification. The night's in-scope calls (B-439 clear
mechanism = `onSuccess` on the pull-handler seam; build-fact-6 = interception) were
engineering-level, decided and stated per the slate's explicit grant.

## Findings (for triage, none blocking)

1. **FILL-04 (`tests/sync-fill-view.spec.js:451`) red outside the four-red baseline** — one
   failure under full-suite load (poll for `window.HQSync.db` −1 for 60s); standalone the
   whole spec is 9/9 green on the same tree + coordinates (`c1-fill04-isolation.log`). No
   card seam reaches it (diff = `marketing/*`, `tests/marketing.spec.js`, `sw.js`). The exact
   B-437 shape from run 20260906-2. Candidate for a B-437-style flake filing — but per the
   standing rule, "rare, mechanism known" must not be laundered into "not flaky"; it is
   recorded as a red outside baseline, characterized, not excused.
2. **B1-XT-01 passed again** (second consecutive run) — B-433's four-red baseline is drifting
   from observation; triage may want to re-measure it on a clean detached base.
3. **0-byte G1 logs** — the implementer's G1 evidence was not captured (`echo EXIT=$?` missing
   from the redirect). G6 closed the hole by re-execution; future implementer prompts should
   require the EXIT marker *inside* the log file.
4. **G6 non-blocking observations**, kept for the record: `onSuccess` fires before
   `res.json()` in `pull-replication.js:153` — an HTTP-200 with a malformed body clears the
   latch for one cycle before that cycle's json failure re-latches; fail-safe direction
   survives. And the clause-1 e2e's substrate is interception, not the real door — exactly the
   build-fact-6 call the slate authorized, flagged so nobody later reads the e2e as real-door
   coverage.
5. **Isolation-name deviation, handled in-run:** the launch prompt's `TEST_DB_NAME=hq_c1_impl_0907`
   fails `scripts/reset-e2e-db.js`'s name guard (`/^hq_test(?:_[a-z0-9]+)*$/`); the implementer
   used `hq_test_c1impl0907` and did **not** widen the guard. Future launch prompts should mint
   guard-conforming names.

## Next actions

1. **`/nc-morning-triage`** — review this branch, merge `overnight-20260907` → `dev`. One
   card, clean merge, G6 first-pass approve; the conflict log has one entry.
2. **Attest close-bar leg 3** — the attended act this card existed to unblock. The refusal
   is landed and mutation-proven (T-55); tonight makes the replica path real: provisioning
   through the shipped door, real codes offline, `policy_unresolved` recording truthfully
   against a network-layer failure. Leg 3 is attestable **now**.
3. Triage the FILL-04 finding (B-437-style filing or fold into B-437) and consider
   re-measuring the B-433 baseline (B1-XT-01 green twice running).
4. Activity B is closed — the overnight queue is empty by design. The unlock for E/F is
   attended: Activity 0's facts + accounts, then spike sittings (slate §"Explicitly NOT").
5. Standing items unchanged: B-14's clone-side remedy; the ATTENDED live-camera check from
   run 20260905 card 5 stays ARMED; stranded branches per decision 175 (B-442) untouched
   tonight, as ruled.
