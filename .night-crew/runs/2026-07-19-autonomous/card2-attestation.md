## §2 — Attestation (Card 2: cycle-gate-attestation)

**All 4 audit areas verified.** Every citation below was opened at the line/commit/test before it was written. Two honest carries surface and are marked, not hidden: (a) the Engineering-KR **commit SHAs handed to this card are dangling pre-squash worktree objects, not branch-reachable, and the slate's SHA→claim mapping for "loud rejection" and "transactional emission" is scrambled** — the *work* all exists and is verified; the attestation cites the **actual landed squashes** and corrects the mapping; (b) the red-first pairs are **documented but not git-reconstructable** because the night-crew merge protocol squashes each card's worktree into one landing commit (the 07-16 §2 caveat, per T-14).

### Reachability note (load-bearing — read before the tables)

The five editprop SHAs the slate cites (`6a483d1`, `0d49f27`, `1c7c73c`, `72fffba`, `6c3aafb`) **resolve as git objects** (`git show` succeeds) but `git merge-base --is-ancestor 6a483d1 HEAD` → **NO**, and `git branch --contains` returns nothing: they are **dangling pre-squash worktree commits**. The branch-reachable **landed** commits are the squashes, all ancestors of `HEAD` under merge `22cb7dd` (`overnight-20260717 → dev`):

| Pre-squash (cited, dangling) | Landed squash (branch-reachable) | Card |
|---|---|---|
| `6a483d1` | **`86bd09c`** | `editprop-stable-field-identity` |
| `0d49f27` + `1c7c73c` | **`186e14c`** | `editprop-broadcast-rerender` |
| `72fffba` + `6c3aafb` | **`3e5b921`** | `editprop-convergence-matrix` |

Verified the squashes carry identical diffs to the worktree objects (e.g. `86bd09c` and `6a483d1` both add `ErrUnknownField`/422 in `handler.go`+`repository.go`; `186e14c` and `0d49f27` both add `EmitOpTx` in `sync/ops.go`). Attestation cites the **landed** SHA with the worktree SHA in parentheses.

### (1) Product KR1 — 100% of PRD requirements trace to a reproduced failure or a named invariant

**Verdict: PASS (12/12 requirements trace).** Audited the PRD trace table (`PRD-data-integrity.md:170-185`), all 12 rows:

| Rows | Trace anchor | Type | PRD lines |
|---|---|---|---|
| FR-2, FR-3, FR-4, FR-7 | **REPRO** (`tests/repro-cut-task.spec.js` — the reproduced P0) | reproduced failure | `:174-177,:180` |
| FR-1, FR-5, FR-6, FR-8, FR-9 | **INV-1 / INV-3 / INV-6** (named invariants, `:64-83`) | named invariant | `:173,:178-182` |
| NFR-1, NFR-2, NFR-3 | QA discipline · Brief hard constraint · operator-signed design gate | process anchor | `:183-185` |

Every functional requirement (FR-1…FR-9, 9/9) traces to REPRO or an INV-1…6 named invariant; the three NFRs trace to named process anchors. **0 untraced rows.** Honest nuance: the 3 NFR anchors are process constraints, not members of the INV-1…6 set — still named, still auditable, consistent with the PRD's own §Requirements preamble (`:62-63`).

### (2) Product KR2 / Delivery repro-pair — 1 ratified decision + ≥2 passing acceptance tests + 1 RED/1 GREEN in the WO record

**Verdict: PASS.**

**The edit semantic as exactly 1 sign-off-ratified decision (INV-3, frozen-at-submit):**

| Artifact | Citation | Content |
|---|---|---|
| PRD invariant | `PRD-data-integrity.md:70-73` | INV-3 — "operator-delegated, PM-chosen, sign-off-ratified 2026-07-16" |
| Ledger decision record | `ledger.md:329-353` (G-2) + commit `5e7c161` "evening PM session + grill-back — frozen-at-submit PRD signed (G-1/G-2)" | one decision: FROZEN-AT-SUBMIT chosen head-to-head over run-pinned versioning |
| OKR | `okrs.md:30` | KR-2 names it as "1 operator-delegated, sign-off-ratified decision" |

Exactly **1** decision, ratified at sign-off. ✓

**≥2 passing acceptance tests (opened `tests/repro-cut-task.spec.js`, 237 lines, in the current tree):**

| Test | file:line | Asserts (claimed semantic) |
|---|---|---|
| **Test A** — AC-6a "cut a task mid-run → surviving checkbox stays checked on the observing device (live + catch-up)" | `repro-cut-task.spec.js:87-148` | Device B has the checklist open; admin (A) cuts a field via `updateTemplate`; B **live** shows the cut field gone (count 0, `:133`) AND the surviving `Wipe counters` checkbox **still checked** (`:135`), and again after reload/catch-up (`:143-144`). Mid-run edit re-renders open devices with surviving answers intact. ✓ |
| **Test B** — AC-6b "a later edit (rename + add + cut) does not change the submitted record's rendered review" | `repro-cut-task.spec.js:152-236` | After submit freezes the record, a maximally disruptive edit (rename + add + cut) is applied to the **live** template; the submitted record's rendered review is **byte-identical** (`afterFp` `toEqual(before)`, `:228`) and the frozen server `template_snapshot` string is unchanged (`snapAfter === snapBefore`, `:235`). A submitted checklist is unaffected by later edits. ✓ |

Both tests exist and assert the exact claimed semantics.

**Exactly 1 RED + 1 GREEN in the WO record:** `421ceee` committed the repro spec **skip-guarded** (baseline RED preserved, message: "Skipped so task test stays green until the editprop build card un-skips it… and records the red→green pair in its WO record"). The 07-17 HANDOFF W-1/W-2/W-3 rows (`.night-crew/runs/2026-07-17-autonomous/HANDOFF.md:70-72`) record the pair: "AC-6a … is a real red→green" (`:51`), assertions un-skipped and extended to frozen-at-submit, **red on pristine (churn build), green on the fix**. **Documented 1-RED/1-GREEN; squash-caveated** (test+fix land in one squash commit, not git-bisectable).

### (3) Engineering KRs — each landed commit verified against its claim

**Verdict: behaviors PASS; provided citations corrected (2 of 4 SHA mappings were misattributed).**

| Eng KR | Slate's cited SHA | Correct landed commit | `git show --stat` verified content |
|---|---|---|---|
| **Stable identity** — `updateTemplate` diff-upsert, old delete-and-reinsert path deleted | `6a483d1` ✓ (worktree) | **`86bd09c`** (`6a483d1`) | `repository.go` +272: `updateTemplate` diff-upserts by Builder-sent IDs (kept UPDATE, new INSERT, removed DELETE); **old `replaceTemplate` delete-and-reinsert path deleted** (`func replaceTemplate` removed). `stable_identity_test.go` +265 asserts surviving-ID guarantee. **Correction:** the slate's phrase "replaceTemplate reinserts deleted" is inverted — the delete-and-reinsert `replaceTemplate` was the *bug* and is **deleted**; `updateTemplate` replaces it. |
| **Loud rejection** — distinct 422 envelope, 0 dead-id 200s | ~~`0d49f27`/`1c7c73c`~~ **WRONG** | **`86bd09c`** (`6a483d1`) | The 422 `{"error":"unknown_field"}` envelope + `var ErrUnknownField` + `writeError(w, http.StatusUnprocessableEntity, "unknown_field")` are in the **stable-identity** commit (`handler.go`/`repository.go` in `6a483d1`), **not** `0d49f27`/`1c7c73c`. `0d49f27` is broadcast+transactional; `1c7c73c` is the INV-6 discard warning. `stable_identity_test.go` covers the 422 contract at function + handler level. |
| **Transactional op emission** — op commits in the write's txn, 0 writes with unqueued op | ~~`72fffba`/`6c3aafb`~~ **WRONG** | **`186e14c`** (`0d49f27`) | `EmitOpTx(ctx, tx, op)` — records op (lamport bump + row INSERT + `pg_notify`) **inside the caller's transaction, no self-commit** — is in `sync/ops.go` +64 in `0d49f27`/`186e14c`, with `broadcast_emit_test.go` +118 (RED on the goroutine path). **Not** `72fffba`/`6c3aafb` (those are the matrix). |
| **Convergence matrix — 0 cells red** | (find artifact) | **`3e5b921`** (`72fffba`+`6c3aafb`) → de-flaked `6291ef2` | `tests/sync.spec.js` (1534 L, in tree): two "Convergence matrix (W-3)" describes (`:832`, `:1023`) = **~11 cells** — checkbox/yes-no/text/temperature/sub-step/fail-note text+severity/fail-note photo-URL (7 field-type cells) + submit/unsubmit transitions + list-progress + list-denominator-on-cut — plus the W-6/W-6b LWW-conflict cells (`:1211,:1414`). Zero-flake under `--retries=0` proven by the 07-18 hardening (`14a36e8`, merged `6291ef2`; ledger `:441-450`). |

**Net:** all four Engineering behaviors are landed and verified in the tree; **the "loud rejection" and "transactional emission" SHA citations the card was given are misattributed** and are corrected above. **Convergence-matrix "0 cells red" rests on the landed spec + the recorded 07-18 de-flake; the live re-run is Card 1's job.**

### (4) QA KR2 — 100% of this cycle's fix-WOs carry red-run evidence

**Verdict: PASS (documented), with the squash caveat carried.** Enumerated this cycle's fix-classified WOs from the 07-17 + 07-18 HANDOFFs; each carries a documented red-run in its WO record (`.night-crew/runs/2026-07-17-autonomous/HANDOFF.md:68-78`):

| Fix-WO | Landed commit | Red-run evidence (documented) | Git-reconstructable? |
|---|---|---|---|
| W-1 `editprop-stable-field-identity` | `86bd09c` | 422 `unknown_field` + cross-device identity red on churn build → green; G6 re-reproduced | No — squash |
| W-2 `editprop-broadcast-rerender` | `186e14c` | 5 sub-behaviors red→green (SAVE_TEMPLATE re-render, silent catch-up, transactional emission, INV-6) | No — squash |
| W-3 `editprop-convergence-matrix` | `3e5b921` | AC-6a bug-guard + AC-6b snapshot-lock red→green; unsubmit-broadcast gap fixed | No — squash |
| W-4 `engine-approval-feedback-loud` | `f50dd32` | 200 (false "Approved") → 500 `feedback_persist_failed`; G6-reproduced | No — squash |
| W-5 `ops-nfr3-resubmit-photo-gate` | `733fa16` | direct-API resubmit 201 bypass → 400 `resubmit_photo_required`; G6-reproduced | No — squash |
| W-6 `engine-conflict-refetch` | `fc0ed6b` | LWW loser `undefined` → `WINNER` via `applyOp`; deterministic 3/3, G6-reproduced | No — squash |
| (07-18) `editprop-convergence-cell-hardening` | `14a36e8` (merged `6291ef2`) | 4 conflict types red→green + no-retry de-flake streak; 2 fail-note types parked (D-1, footprint) | No — squash |

Excluded from the fix denominator (correctly, not fixes): **U-1** `users-s3-orphan-cleanup` (`a11a58f`, hygiene, zero behavior change) and **T-2** `vacuous-tests-18-to-0` (`3fd4d3f`, test-only conversion, retires waiver #2). **T-1** `carried-fix-wos-sweep` (`c5aede8`) carries a behavioral red→green (clock-seam) though it is seam/test-shaped.

**100% of fix-WOs (6/6 core + the 07-18 hardening) carry documented red-run evidence.**

**Honest caveat (carried, not hidden — per T-14):** the night-crew merge protocol squashes each app-fix card's worktree into a **single** landing commit, so the new spec test and the fix arrive together — git cannot show a standalone failing-before commit for any of them. Red-first ordering is **documented** (07-17 HANDOFF per-card table naming each observable break; commit messages assert "red-first"; independent fresh-G6 re-reproduction recorded at `:80-82`) but **not git-reconstructable** for any fix this cycle. Unlike the 07-16 cycle — which had exactly one git-verifiable RED→GREEN pair (Inventory NFR-1, `1a0265e` precedes `77957c1`) — **this cycle has zero standalone git-bisectable red→green pairs**; the repro baseline `421ceee` was committed *skip-guarded* (kept green on purpose), so even it is a documented, not executed-failing, baseline. QA-KR2 is attested **on the WO/ledger record**, corroborated by the independent per-card G6 re-reproduction.
