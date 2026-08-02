# HANDOFF — run `overnight-20260803`

**Night B of the two-night milestone close.** Branch `overnight-20260803`, cut off `dev` at
`60b9edb`. Dispatch **SERIAL**, as the operator chose at sign-off.

> ### 🛑 Read this first
>
> - **The milestone does NOT close on cards.** `sync-hard-cutover` — the last white card of the
>   cycle — **PARKED**, and it parked on a finding that says it is *not buildable as specified*.
>   There is no third white card behind it. See **F-1**.
> - **The cutover has no data plane.** RxDB replicates to a **different Postgres** than the one
>   `/submit`, `/myChecklists` and every report read, and nothing carries rows back. Making RxDB
>   the single write path today would have made checklist answers **silently vanish at Submit**.
>   The card's own `done_when:` test would have passed while that happened.
> - **D-KR2 still needs an attended deploy plus 2/2 tab screenshots on a RETURNING client** — that
>   was true before tonight and is true after. Nothing tonight touched it either way.
> - **P6's outbound notice is DRAFTED and UNSENT.** Nothing was sent to sales-processor. Two
>   notices are now owed to them; see **F-3**.
> - 🛑 **Do NOT trust `run-evidence check` on this repo at the next launch.** It reports
>   `no-run-evidence` for `20260803` even though the closeout record and conflict log are both
>   written and committed — it looks for them in a root `reference/` that hq does not have. **This
>   night ran.** Filed **B-77**, destination the night-crew clone. If `/nc-run` is invoked before
>   that lands, it will claim this slate never executed and may offer to re-run it.

---

## Per-card outcomes

| Card | Verdict | G6 | End-to-end | What landed |
|---|---|---|---|---|
| **S1a** `sync-cutover-list-scope` | ✅ **MERGED** | **PASS**, first submission | ~90m | The operator's list-scope widening, with a mandatory date floor. B-42 option (i) and B-58 folded in. |
| **S1b** `sync-hard-cutover` | 🅿️ **PARKED** | — (no code to review) | ~20m | Nothing but evidence. `git diff` outside `.night-crew/` is **empty**. |
| **P6** `period-summary-contract-notice` | ✅ **MERGED** | **FAIL → FAIL → fixed** (2 rounds, 2 fix rounds) | ~116m | 111 contract rows audited, 45 wrong. An UNSENT notice drafted. **No code changed.** |

**2 of 2 committed cards resolved; 1 merged, 1 parked with evidence. The stretch card also merged.**

---

## S1a `sync-cutover-list-scope` — MERGED

Implements the fork the operator resolved on 2026-08-02: **lists stay live, the scope is widened.**
`normalizeScope`/`scopeFilterFor` now accept a **list scope** beside the existing fill scope,
bounded per-user and by a **mandatory date floor**.

**One deviation from the slate's literal text, decided at engineer level and stated.** The slate
specified `checklists: assigned_to.eq.<userId>`. **That column does not exist** — not on that table
nor on any of the four replicated ones (assignment lives in `template_assignments`, which is not
replicated and is unreachable from a PostgREST client). The *rule* was satisfiable and only the
spelling was not, so the card delivered the same bound via `scope.templateIds` **plus RLS**
(`hq_can_see_template` / `hq_can_see_field`, `security definer`, resolved live per row). **No column,
table, view, index, role or grant was added**; no `0005_*.sql` exists. The PARK trigger for "needs a
schema change" therefore did not fire, correctly.

**Why the per-user bound is real and not client-side theatre:** the client's `templateIds` is a
*bound*, the RLS predicates are the *gate*. Subtest `LIST-2` measures it — user A hands the server
user B's `template_id=in.(…)` string verbatim and gets **nothing** back, with a `service_role`
BYPASSRLS control on a **byte-identical URL** proving the table was not merely empty.

**G6 passed it on first submission — and it was a hard pass, not an unexamined one.** The reviewer
re-ran the B-58 SQL mutation independently (W17 the only red, and red for the *right reason* — the
attack row actually moved), drove the Realtime filter shim against the **real** `@supabase/supabase-js`
client rather than the test double, and ran **seven feature-removal mutations** to confirm the new
tests actually discriminate. All seven reddened. Every mutation was reverted; the tree was clean.

**Riders filed by S1a, all addressed to S1b — and S1b parked, so all four are still open:** B-61
(the RxDB list is *narrower* than the REST list it replaces — an operator product call), B-62 (the
Realtime filter is proven at the config, never against a live Realtime server), B-63 (list + fill
replications will be live concurrently over the same four local collections), B-64 (`bootstrap.js`'s
scope banner went stale the moment S1a landed).

---

## S1b `sync-hard-cutover` — PARKED. This is the night's headline.

**Trigger:** the card's own recorded one — *"PARK if retiring `/saveResponse` turns out to reopen
ledger decision 49 — that is a recorded fork, not a judgement call."*

It reopens it, and worse: **decision 49's decisive argument is a claim about the cutover, and the
claim is false as built.** Decision 49 reasoned that

> *"Activity 1 ends in `sync-hard-cutover`, where RxDB replicates rows **straight from Postgres**
> and **there is no API boundary left to translate at**."*

### What is actually true — verified independently by the orchestrator, not taken on the card's report

- `sync-schema/sql/0001_sync_tables.sql`'s own header: **"⚠ THIS FILE DOES NOT RUN AGAINST HQ's
  POSTGRES."** The four replicated tables live on the **self-hosted Supabase substrate** — a second,
  different database.
- `sync-schema/sql/0002_hq_fdw.sql` creates foreign tables **on the substrate, pointing at `hq_pg`**.
  The bridge runs **HQ → substrate**, is **read-only**, and carries **permissions, not data** —
  exactly four tables (`hq_template_assignees`, `hq_user_roles`, `hq_field_templates`,
  `hq_template_approvers`).
- HQ's submit path reads `submission_responses` from **HQ's own** Postgres
  (`backend/internal/workflow/repository.go:734, 846, 1100`).
- **Nothing carries a checklist row from the substrate back into HQ.** No worker, no reverse bridge,
  no backfill — a repo-wide search returns nothing.

### The failure this park prevented

Unmounting `/saveResponse` does not *move* the write. It **detaches answers from submission**. A
crew member fills a checklist; every answer persists (IndexedDB + substrate); they press Submit; and
they submit an **empty checklist**, because `SUBMIT_CHECKLIST` builds from HQ's `submission_responses`.

🛑 **The card's own `done_when:` test would have passed while that happened** — it asserts a value
survives back-to-list and reopen, which it would, from local IndexedDB. This is the same wall
decision 92 already ruled out and card B2 already parked on (`max_prepared_transactions = 0` at both
ends).

**20 minutes to establish that, against a 4h30m–7h estimate.** The slate's instruction — *"prefer a
clean early exit over starting a card you cannot finish cleanly"* and *"a park with evidence is a
better night than a merge with a caveat"* — paid for itself roughly fifteen-fold.

### Two further findings that would have bitten regardless

- **🛑 `autoSaveField` does not exist — and it is called in production code.** Verified by the
  orchestrator: **zero definitions** repo-wide, yet `workflows.html:2219` calls it, which is a live
  `ReferenceError` on the fail-note-with-photo path. The card text, the roadmap bullet,
  `bootstrap.js:9` and `conflict-notice-ui.js:23` all name it as the live write path. **It is not the
  write path and never was** — the real one is `debouncedSaveField` → `submitOp('SET_FIELD')` →
  `POST /ops`. Filed **B-65**.
- **A second footprint correction, same severity as the one the slate already carries.**
  `main.go:47-95`'s `workflowOpRouter` routes `SUBMIT_CHECKLIST` → `ValidateFailNotesFunc` →
  `ValidateResubmitPhotoFunc` → `SubmitChecklistFunc`. **`ops.go`/`handler.go` are the transport for
  all workflow validation, not just the Lamport layer** — deleting them deletes the submit
  validators, which have no RLS counterpart. Filed **B-68**.

---

## P6 `period-summary-contract-notice` — MERGED, after two G6 rounds and two fix rounds

**111 rows audited across both sales-processor contract documents. 45 wrong.** The sharper half of
that: only a minority *drifted*. **22 of the menu-cogs rows were never true** — authored
2026-06-04 at 23:50 from a phase plan, **thirteen hours after** the handler they describe landed at
10:18 the same day. Nobody diffed them.

**No code changed.** `backend/` is byte-identical to the base. The defect is documentary, and
correcting code to match prose would have altered behaviour under an external consumer.

### 🛑 The review history is the lesson, and it should shape how this card class is sized

Each pass found errors **in the previous pass's own corrections**:

1. **G6 round 1 → FAIL.** Three regions of the documents had **never been audited at all** — the
   audit table simply jumped over them. One was the State A example still publishing the exact wrong
   keys (`name`, `menu`, `menu_subgroup: null`) that the same document calls *"the most directly
   harmful item on the page."*
2. **Fix round 1 → expanded the audit 76 → 111 rows** and found genuinely new defects in the newly
   covered regions. But it also **introduced a new factual error** in the very row the card exists to
   fix, misstating where non-blocking pending money goes.
3. **G6 round 2 → FAIL.** Found a **fourth** instance of that same forgot-the-pending-summand defect,
   at `:27`/`:102`, claiming *"the COGS aggregate has no timezone dependency at all."*
4. **Fix round 2 (surgical, scoped to two findings) → FIXED.**

**Verified by the orchestrator at source before acting on it:** the `pending` CTE
(`handler.go:1366-1375`) *is* period-filtered by `pendingPeriodDateExpr` (`handler.go:43` =
`COALESCE(event_date, (created_at AT TIME ZONE 'America/New_York')::date)`). So `cogs_excl_tax`,
`cogs_incl_tax`, `purchase_event_count` and `by_vendor` **all** carry the zone dependency the
document denied. Note the likely origin: **the code's own comment at `handler.go:1318-1319` says
"plain DATE comparison for COGS" and describes only the confirmed half.** That comment is still
there and will re-seed the error the next time someone documents from it.

### One orchestrator intervention worth naming

Fix round 2 corrected `"fourteen months"` → `"eight weeks"` repo-wide (the contracts are stamped
**Authored 2026-06-02/06-04**; the audit is 2026-08-03; ledger decision 106 says *"eight weeks"*).
That sweep caught **`reference/slate-20260803.md:331` — a signed artifact. I reverted that one line.**
The slate records what the operator signed, and "fourteen months" is what was believed at planning
time; the audit is precisely what *discovered* otherwise. Backdating a correction into a signed plan
of record would erase the evidence that the discovery happened. The corrected figure stands
everywhere it is a live claim.

The figure mattered: it appeared **twice inside the draft notice body**, once as a section heading,
in a message whose entire argument is *"we are being exact with you now"* — and the counterparty
could have falsified it from the "Authored 2026-06-02" line at the top of the document they hold.

---

## 🛑 Final gate — taken by the orchestrator on the FINAL MERGED TREE, not inherited from card reports

| Gate | Result |
|---|---|
| **G1** | `go build ./...` **0** · `go vet ./...` **0** (from `backend/`, the module root) |
| **G2 (Go)** | `go test -p 1 -count=1 ./...` → **exit 0**, all **9** packages `ok`. **Counts checked, not just `ok`**: `internal/workflow` **35 tests, 0 skips** (this is the package CLAUDE.md warns can print `ok` on zero tests — it did not). `TestRowVisibilityRLS` **59 subtests executed** with `HQ_SYNC_SUBSTRATE_OPTIONAL` **unset** — S1a's +5 over the inherited 54, confirmed on the merged tree. |
| **G2 (Playwright)** | See the block below. Full suite, run **alone on the box**. |
| **G3** | **N/A** — preflight verdict `openspec: absent`, re-confirmed at launch. No scaffolding created. |
| **G4** | `node build-sw.js` **after** the merge commits (B-37) → **31 files precached**, reachability 18 parsed / 30 resolved / 0 outside. Second run left the tree **clean** ⇒ idempotent. Version parity three-way: `version.go Frontend` ≡ `package.json` ≡ `version.json` = **1.4.0**. Backend **0.3.0**. |

### G2 (Playwright) — full suite on the merged tree, run alone on the box

```
1 failed
6 skipped
778 passed (24.2m)
```

**Exactly ONE summary block** (verified: one `N passed (` line). 785 tests. Run with `--retries=0`,
no Go or RLS suite running alongside it — the contention that produced 20260802's discarded
51.7m/7-failure gate was not recreated. 24.2m matches that run's *quiet-box* figure (24.5m), which
is the corroboration that the box really was quiet.

#### 🛑 The one failure is NOT an armed red, and I did not treat it as one

```
tests/onboarding.spec.js:689:3 › Manager tab › sign-off form requires readiness rating (notes optional)
Test timeout of 30000ms exceeded — waiting for locator('[data-action="view-training"][data-template-id="…"]')
```

That title appears on **no** baseline list. It is not B-27, not either `[LST-17]`, not `[A1-TZ-02]`,
not `[LC-02]`, and not the B-45 convergence test. So it was investigated rather than absorbed.

**What was measured, in order:**

| Check | Result |
|---|---|
| Did tonight's run touch onboarding at all? | **No.** `git diff --name-only 60b9edb HEAD` matches nothing under `onboarding` — not `onboarding.html`, not `backend/internal/onboarding`, not `tests/onboarding.spec.js`. The run's 17 changed files are sync-scope, contracts, docs and `sw.js`. |
| Re-run in isolation, merged tree, **shared** `hq_test_e2e` | **FAILED** — so not the B-45 load-sensitivity class |
| Same test on the **base commit `60b9edb`** (pre-run `dev`), **shared** DB | **FAILED** — so not introduced by this run |
| Base commit, **fresh** DB `base_fresh_e2e` | ✅ **PASSED** (2.4m) |
| **Merged tree**, **fresh** DB `merged_fresh_e2e` | ✅ **PASSED** (2.4m) |

**Conclusion: accumulated state in the shared `hq_test_e2e` database, not a code defect and not
tonight's.** The behaviour is *symmetric* across the base and merged trees — dirty DB fails on both,
fresh DB passes on both — which is what rules out this run as the cause. The mechanism was
demonstrated, not asserted.

🛑 **This is deliberately not laundered into "flaky, ignore it."** It is a **gate-integrity finding**
and it is filed as **B-76**: the full-suite gate runs against a database that carries state between
runs, so a red can be an artifact of a previous run — **and so can a green.** That second half is the
uncomfortable one, and it applies to every full-suite figure in this milestone, including the 778
passes above.

**Verdict on the gate: green except the armed reds, plus one demonstrated shared-database artifact.**
Not "green."

### Baseline honesty

Judged against **"green except the armed reds"**, never against green. **B-27 and the three other
armed reds remain ARMED** — per decision 100 / T-31 decision 120, passing retires nothing, and no
card tonight claimed otherwise.

### An orchestrator error, disclosed

The final Go gate was **first run with `postgres:postgres` credentials** when this box uses
**`yumyums:yumyums`**. It failed **loud** — the harness correctly refused to skip the DB-coupled
tests and said so — which is the fail-loud design working exactly as intended. Cost: one ~15-minute
run. Recorded because a gate that failed for an environment reason and a gate that failed for a code
reason must never be reported the same way. Separately, an earlier `go build ./...` was run from the
repo root (where `./...` matches no module) **with the error masked by a pipe into `tail`**, printing
a false green; caught and re-run correctly.

---

## Conflict log

`.night-crew/knowledge/reference/conflicts-20260803.md` — **3 merges, 3 clean, 0 conflicted hunks.**

The honest read is in the log itself: serial dispatch plus a footprint split that gave each card sole
ownership of its files produced this, and **merge 2 was clean because S1b parked**, not because two
overlapping diffs were reconciled well. A night where S1b implemented its full footprint would have
been the real test of that log, and it did not happen.

---

## What is NOT done — every item with a destination (B-38)

| Item | Destination |
|---|---|
| **`sync-hard-cutover` itself** | **Blocked on F-1.** Cannot be re-slated until the data-plane question is answered — it is not a bigger card, it is a card with a missing premise. |
| **B-61** RxDB list narrower than the REST list | Operator call, **F-2**; then the successor cutover card |
| **B-62** Realtime filter unproven against a live server | Successor cutover card (needs `HQ_SYNC_REST_URL` set, which F-1 gates) |
| **B-63** list + fill replications concurrent over the same collections | Successor cutover card |
| **B-64** `bootstrap.js` scope banner stale | Successor cutover card (deliberately not fixed — its own lead says fix it *with* B-63's decision) |
| **B-65** `autoSaveField` is a phantom, and is **called** | **Next milestone — but see the note below; this one is live in production code** |
| **B-66** push plugin's 23505 recovery re-fetches by PK; draft uniqueness is `(field_id, answered_by)` | Successor cutover card |
| **B-67** `sync-schema/sql/*` applied by nothing outside the Go test harness | Successor cutover card |
| **B-68** `ops/handler.go` carries **all** workflow validation | Successor cutover card |
| **B-69** `sync.js` also holds `api()` (12 call sites), the submit queue and `APP_TIMEZONE` | Successor cutover card |
| **B-70** a raw NUL byte in `sync-rxdb/client.js` puts every `grep` into binary mode | **Next milestone — see the note below; this one corrupts evidence** |
| **B-71** contract tests decoding into the handler's own struct cannot detect doc-vs-code drift | Next milestone (golden raw-JSON key tests) — **this is P6's root cause** |
| **B-72** `/menu-cogs` omits menu items that sold but have no recipe | Next milestone, gated on operator |
| **B-73** cross-endpoint reconciliation invariant broke 2026-06-06, still published | Next milestone — **genuine product fork** |
| **B-74** `HQ_COGS_CATEGORY_ALLOWLIST` silently restates every historical period | Next milestone |
| **B-75** menu-cogs assumption A6 publishes the wrong rounding *locus* (SQL, not Go-decode) | Next milestone, with B-71 |
| **B-76** the e2e gate shares one accumulating database across runs — **a green can be an artifact too** | **Next milestone (test isolation / gate integrity), as one sitting with B-50 and B-35** |
| **B-20** Queued badge on Builder rows | **Survives** — `sync.js:671` untouched because S1b parked. Stays destined to P3 `sync-banner-builder-tab-scope`, which never needed the cutover |
| **P6 fix-forward checklist** (B3, B4, B6–B10 from G6 round 2) | **Attended triage, ~10 min** — listed below |
| **D-KR2** prod parity + 2/2 tab screenshots on a returning client | **Attended deploy** — unchanged by tonight |
| **Attended `task sandbox:e2e`** | Still not satisfied. S1b would have re-armed it; it parked, so the status is unchanged |

### 🛑 Two backlog items that are more urgent than "next milestone" implies

- **B-65 — `autoSaveField` is called at `workflows.html:2219` and does not exist.** That is a live
  `ReferenceError` on the fail-note-with-photo path, in shipped code, today. It is filed to the next
  milestone under the scope freeze because it reddens no named KR — **but it is a user-facing defect,
  not a documentation problem**, and it deserves a decision at triage rather than a queue position.
- **B-70 — a raw NUL byte in `sync-rxdb/client.js` makes plain `grep` return nothing and exit 0.**
  Several `done_when:` rows this milestone are of the form *"grep returns nothing"*. **That grep is
  currently unreliable in the passing direction** — it cannot distinguish "clean" from "binary file,
  suppressed." No evidence in tonight's run depended on it (S1a's row 5 used `git diff` and sha256;
  S1b's greps targeted Go and HTML files), but it is a live hazard to gate integrity.

### P6 fix-forward checklist for attended triage

Non-blocking, none of them a wrong code-level claim, all in already-merged documentation:

- **B3** — the notice asserts A1's timezone notice was *"drafted 2026-08-01"*; **no such draft exists**
  in the repo, and decision 106 records it as *owed*. Correct to "owed and undrafted."
- **B4** — the notice re-opens the sequencing question that **decision 106 already decided**, without
  citing it. Frame as an amendment request against 106, not a cold question. *(This is **F-3**.)*
- **B6** — `:255`–`:270` says "nine json tags across three types"; it is **eight**.
- **B7** — `:298` says "the four slice fields carry no `omitempty`"; there are **five**.
- **B8** — the `by_vendor` field row lacks the allowlist/`reason` qualifier the sibling rows got.
- **B9** — "four of the nine per-item JSON field names have never matched" over-rolls-up: it is one
  wrong name, one phantom field, one wrong nullability, one undocumented field.
- **B10** — the merge-intent lists discoveries as B-71..B-74; **B-75** was filed after and is missing.

---

## Next actions, in order

1. **Answer F-1 — the data-plane fork.** Everything in the sync programme is behind it. This is not
   a card-sizing question; the cutover as specified has a missing premise, and no amount of budget
   fixes that.
2. **Answer F-2** (does a crew member still see a colleague's completed checklist after cutover?) —
   it changes what the successor card is allowed to be.
3. **Answer F-3** (one combined notice, or hold to decision 106's two?) — then send, or don't. **P6's
   notice is drafted and UNSENT; nothing has been delivered.** Do the ~10-minute B3/B4 correction
   *before* it goes anywhere near the counterparty.
4. **Decide B-65 and B-70 out of band** — a live `ReferenceError` and a gate-integrity hazard are
   both filed to "next milestone" under the scope freeze, which is correct by the rule and
   uncomfortable on the merits.
5. **Milestone close: on cards, it does not close.** `sync-hard-cutover` is white and blocked.
   **D-KR2 needs the attended deploy and 2/2 tab screenshots on a RETURNING client** regardless —
   that was the gap before tonight and it is the gap now.
