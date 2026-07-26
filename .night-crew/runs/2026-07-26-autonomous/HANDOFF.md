# HANDOFF — run `overnight-20260726`

**Branch:** `overnight-20260726` (cut from `dev` @ `4bcd63d`) · **Slate:**
`.night-crew/knowledge/reference/slate-20260726.md` (batch sign-off 2026-07-25) ·
**Dispatch:** Wave-0-then-concurrent, operator-chosen.

**All three slated cards landed and merged. Nothing parked.**

You are the morning reader. The short version: the night did what it was signed to do, and the
things worth your attention are **five decisions** (D-5, D-7, D-8, D-12, and the RUN-10 verdict) and
**three operator-owned preconditions** that still block the rest of the milestone.

---

## Per-card outcomes

| Card | Verdict | G6 | Merge |
|---|---|---|---|
| **A** `workflow-submission-status-client-half` | **LANDED** | APPROVE-WITH-NOTES | `3cbc650` |
| **C** `sync-rxdb-browser-delivery-spike` | **LANDED — verdict GO** | APPROVE-WITH-NOTES | `a848189` |
| **B** `sync-jwt-bridge-endpoint` (backend half) | **LANDED** | APPROVE-WITH-NOTES | `566130a` |

Merge order was **C before B**, per the slate — C's `build-sw.js` change is the one with reach.

### Card A — Wave 0, dispatched alone

**All four Wave-0 acceptance items green.** `tests/repro-cut-task.spec.js:153` and
`tests/sync.spec.js:1581` — the two specs `dev` was knowingly RED on — are green. The new
"no-approval submitted state renders" test (RUN-09c) is green. The `night-crew.toml` seam expansion
landed on both named keys.

Gate: **164 passed / 1 skipped / 0 failed / 0 flaky of 165** on the post-fix 4-spec subset.

The red-first test `0b53d46` was cherry-picked byte-identical, not rewritten. G6 verified the
cherry-pick's byte-identity, verified the red was captured on a tree with **zero production lines
changed**, and — the check that actually matters — **independently demonstrated RUN-09c failing on
the pre-fix tree**, which is what proves the new test is not vacuous.

Two things the implementer did beyond the slate's literal text, both correct and both verified by
G6 against the Go source: it also fixed the optimistic pair's `'pending_approval'` → `'pending'`
half (likewise never persisted), and it declined to change `idempotency_key`, with reasoning G6
confirmed.

### Card C — the verdict is **GO**, and it is earned

Five legs, real Chromium, against W1's live stack: delivery, Dexie across reload, service-worker
interaction, multi-tab leader election, token expiry across an offline period.

G6 did not take the GO on faith. It **cold-rebuilt the bundle byte-identical** in its own scratch
directory, confirmed **`getRxStorageMemory` is not present in the bundle at all** (so the Dexie leg
could not have passed on memory storage even in principle), confirmed the rxdb.info phone-home
accusation at source, and ran the harness **twice itself at 11/11**.

Gate: **540 passed / 1 failed / 2 flaky / 6 skipped of 549**, 47.0 m, under heavy load with card B's
own suite concurrent for the first half. The single failure is RUN-10 — see below.

**Three findings that outrun the card and bind downstream work:**
1. 🛑 **Do not mount Supabase under `/api/`.** Offline, HQ's service worker answered a replication
   pull with **HTTP 200, `application/json`, a well-formed 26-row array** — stale, missing an
   out-of-band insert, indistinguishable from a fresh pull. `NetworkFirst` falls back to cache
   *before* `handlerDidError`. The same query outside `/api/` failed honestly with a `TypeError`.
   **Silent data loss wearing a success response.**
2. **RxDB 17.4.0's dev-mode plugin phones home** — a hidden 1×1 `rxdb.info` iframe whose own
   `!isLocalHost()` guard is commented out in the shipped build. Stripped from the bundle.
3. **PostgREST tolerates ~30 s of `exp` skew** — which nearly produced a false green on leg 5.

### Card B — backend half only, and the roadmap says so in those words

A **stdlib-only** HS256 mint (`crypto/hmac`) bridging HQ's existing session and grant data into
Supabase claims. No GoTrue, no Supabase Auth, no Kong, **no module dependency**.

**Red-first was this card's real gate and it is genuine.** Every attack variant was captured
refusing *before* the policy that refuses it existed — verified at `de00401`, where **forged rows
actually landed at HTTP 201**. 16/16 variants green after, with a `service_role` BYPASSRLS control
ruling out "the table was empty."

**The design call that matters:** token grants are **advisory, not the gate**. Claims freeze at
mint, so a claim-trusting policy would leave a replay window as long as the TTL. RLS joins a live
projection instead — proved by replaying **identical token bytes** across a grant insert and
delete. G6 confirmed this holds in the SQL rather than only in the prose: `hq_grants` appears in the
policy files **only in comments**.

G6 also minted privileged-role tokens **out-of-band with `openssl`** to probe for escalation the
card's own code could not test.

---

## Post-G6 repairs applied before merge (not deferred)

Both B and C were sent back for focused repairs on findings inside their own deliverables. Neither
was scope expansion; both are the G6 gate doing its job.

- **Card B, G6 finding F1 — the anti-drift guard could not detect drift.** The grant-mapping parity
  test was passing on **eleven agreements of absence** against an empty `app_permissions` table.
  G6's falsification: stub `GrantedSlugs` to return empty and the test still passes. Repaired to
  seed **both** grant disjuncts (role and individual — they resolve through different `OR` branches)
  and assert a positive floor. **Proven by stubbing `GrantedSlugs` and watching it fail.** Role
  guard also changed from a one-string denylist to an allowlist.
- **Card C, G6 finding F1 — the headline finding was not asserted.** The 3c trap test passed
  identically under the benign 503 and the dangerous silent-stale 200; only a `console.log` carried
  the discovery. Since 3c is now a hard architectural gate on two downstream cards, a Workbox
  upgrade could have dissolved the evidence while the test stayed green. Repaired to assert
  `silentlyStale === true` and the 200 specifically — and **mutation-tested**: forcing
  `silentlyStale = false` turns 3c red.
- **Card C, G6 finding F2 — it reproduced the rider's own defect.** Half 3 claimed *"every output
  block below is a verbatim capture… nothing in half 3 is annotated or abridged"* while two blocks
  below it were explicitly marked as trimmed. Card C corrected W1's over-strong integrity claim and
  then wrote its own, one generation later, in the same document. Narrowed.

---

## 🛑 The one red, and what was actually measured about it

`tests/workflows.spec.js` › **RUN-10** (*unsubmit returns checklist to editable draft*) went red on
**both** full-suite legs — card B's and card C's. Both cards **refused to attribute it**, which was
correct. A refusal is not an answer, so the orchestrator spent the measurement.

Full detail in **`run10-attribution.md`**. Summary of what is *known*:

| Condition | Tree | Result |
|---|---|---|
| Card A's own 4-spec gate | post-A | RUN-10 **green** |
| RUN-10 alone ×3 | post-A | **green 3/3** |
| Whole `workflows.spec.js` (80 tests) | post-A | **green 80/80** |
| RUN-10 alone ×6 **at load 44.33** | post-A | **green 6/6** |
| FULL suite (load →61) | post-A | **RED**, both attempts |
| FULL suite (load →57.6) | post-A | **RED** |

**89 isolated attempts clean, including at load higher than either leg that failed.** So it is not
the test, and not load alone — it lives in the whole-suite condition.

**PAIRED FULL-SUITE MEASUREMENT — the result** (reading rules were written down *before* it ran, so
the conclusion could not be fitted to the data):

| Leg | Result | RUN-10 |
|---|---|---|
| **POST-A** (card A present) | 544 passed / **0 failed** / 6 skipped, 32.2 m | ✅ **GREEN** |
| **PRE-A** (card A absent) | 539 passed / **2 failed** / 6 skipped, 32.4 m | ✅ **GREEN** |

**The control fired exactly as predicted.** Pre-A's only two failures are
`repro-cut-task.spec.js:153` and `sync.spec.js:1581` — **precisely the two specs card A was written
to repair, and nothing else.** That validates the setup: the pre-A tree really is pre-card-A and the
harness really does discriminate.

**Verdict: green on both → by the pre-committed rule, a BOUND, not an exoneration and not a fix.**

What it *does* establish: **card A did not deterministically break RUN-10.** This is the first
full-suite run where RUN-10 passed on a post-A tree (previously red twice) — and non-determinism on
identical code rules out a regression, which would be deterministic.

What it does **not** establish, said plainly: it does **not** clear card A (green-on-both is equally
consistent with a contention-sensitive interaction this load never reached — both suites finished in
**32 m** vs card C's **47 m**, and the failing legs peaked at load **57.6** and **61+**), and it does
**not** prove a pre-existing flake — **nobody has yet reproduced RUN-10 red on a pre-A tree in any
condition.**

**The cheapest way to close this** is the one observation nobody has made: run the full suite on
**pre-A under deliberate heavy load (1-min ≥ 55)**. A red there closes it immediately.

One methodological caveat, stated rather than buried: cards C and B were merged into the same
working tree the post-A leg runs from. Verified that the measurement's test surface
(`workflows.html`, `tests/workflows.spec.js`, `playwright.config.js`) is **unchanged** since the leg
launched, and the running server binary was compiled before the merges. The only change is an
11-line comment-only addition to a spec Playwright had already loaded.

**Also not attributed** (same discipline): `workflows.spec.js:2355` (LC-02), red on card B's leg
only; and two flaky-on-retry in `sync.spec.js` (`:836` SYN-03, `:1327`).

---

## Standing flags — every one addressed explicitly

| Flag | Status |
|---|---|
| **`dev` RED on two E2E specs** | ✅ **DISARMED.** Card A landed; both green, and both re-verified independently by G6 at `--retries=0`. |
| **Attended two-device convergence check** (carried since 2026-07-22) | 🔴 **RE-ARMED, and it is yours.** Card A changed the live submit/render path. This check cannot be done unattended. **You owe it.** |
| **`tests/sync.spec.js:1198`** (proven ~16–20 % flake) | ✅ **Did not red at any point, on any leg, all night** — card A's 3 legs, card B's full suite, card C's full suite. Zero retries consumed. **Still armed**; card A's seam fix deliberately raises its exposure, so tonight is one clean sample, not a fix. |
| **`tests/purchasing.spec.js:1407`** (FR-13) | ✅ **Passed** on both full-suite legs. Card A correctly reported it **not exercised** rather than claiming a green it had not measured. |
| **W1's runbook integrity claim** | ✅ **DISARMED** — card C's rider **narrowed the claim** (repair was not honestly possible: `spike_notes` accumulates by design, so a re-capture cannot be byte-exact and would recreate the defect one generation later). All six blocks inventoried; the one plain error corrected outright; G6 spot-verified four independently. The closed 20260725 `timings.log` was deliberately left as written. |

---

## Decisions waiting for you — `DECISIONS-NEEDED.md`, D-1 … D-12

**Nothing is parked.** These are records and questions, not blockers on the merge.

**The three I would read first:**

- 🛑 **D-7 — cross-tenant read through `api-cache`.** Workbox's API cache is keyed **by URL only**;
  `Authorization` is not in the cache key and no `Vary` is set. On a **shared truck phone**, a second
  crew member — or a re-login as a different user — hitting the same replication URL can be served
  **the previous user's rows**. This *discloses* data where the `/api/` finding merely loses it, and
  **RLS cannot help, because the request never reaches PostgREST.**
  **Read D-7 together with D-9:** the `/api/` gate closes D-7 only as a side effect. If the schema
  card goes cross-origin and drops that gate as moot, **D-7 does not become moot with it.**
- 🛑 **D-5 — `owner_id = sub` cannot express HQ's real ownership.** A submission belongs to a
  submitter *and* an approver, so under the shipped predicate **an approver sees nothing.** Card B
  declined to invent a permission model — that is its park trigger working as designed. The token
  already carries `hq_roles`, so the fix is expressible without minting anything new; only the
  decision is missing.
- **D-12 — `build-sw.js` globs the working tree, not the git index.** Found at merge time. Any
  untracked page in the repo root silently enters the precache manifest — and `task sw` runs
  automatically as a dependency of **both** `task test` and `task prod:deploy`. A Workbox precache
  entry that 404s **fails the entire service-worker install**, so the symptom is "the PWA stops
  updating" with an invisible cause. Nothing shipped; the regenerated `sw.js` was discarded.

**The rest:** D-1 (History view renders a raw status token), D-2 (offline double-tap still writes
two submission rows — real, pre-existing, outside card A's signed scope), D-3 (stale comment),
D-4 (who writes `hq_grant_projection`), D-6 (umbrella-slug divergence, errs closed), D-8 (**the
vendored bundle adds +34 % to the PWA precache — ~495 KiB per phone over LTE for an asset no page
imports; keep or exclude is your call**), D-9, D-10 (leg-4 ms figures are wall-clock, not
monotonic), D-11.

---

## What this run did NOT close

**The milestone is three cards short, and two of those three are blocked on you:**

1. **`sync-rxdb-schema-and-replication`** — **now unblocked**, card C returned GO. It must be planned
   against C's verdict, including the `/api/` gate (D-7/D-9) and the reverse proxy that is **unbuilt
   and uncosted**.
2. **`sync-rxdb-conflict-notice-ui`** — needs a **mockup sign-off from you**. CLAUDE.md requires a
   committed mockup and an explicit *"ok, build this"* before UI code on phases introducing new
   components. An unattended run cannot obtain that.
3. **`sync-hard-cutover`** — needs both of the above **and** a `/nc-pm-session` disposition on the
   BACKLOG item *"Cross-user checklist hydration divergence"* (`BACKLOG.md:591`, still plain `new`).
   Product KR2 forbids dispatching it until then.

---

## Conflict log

`.night-crew/knowledge/reference/conflicts-20260726.md` — **all three merges logged, clean or not.**

Every merge conflicted in exactly one place: `timings.log`, the append-only ledger. Every one was
resolved by **union, no side dropped**. **No code file conflicted at any merge** — cards C and B
share literally zero code files, which is the property that made concurrent dispatch safe, and it
held. `sw.js` never needed hand-resolution.

Decision numbering survived two merges without a renumber because card C deliberately started at
**D-7**, leaving room for card B's D-4/D-6 which had not yet landed.

---

## Four HARD constraints — held by all three cards, verified on the final merged tree

`backend/go.mod` · root `package.json` **and** `package-lock.json` · `docker-compose.nc.yml` ·
root `Taskfile.yml` → **empty diff vs `dev`.**

Two decisions made that possible while two cards ran concurrently: card C's **`npx esbuild@0.28.1`**
instead of a devDependency, and card B's **stdlib-only** mint instead of a JWT library. Both G6
reviewers verified these by hash at the true merge-base rather than taking the claim.

`vendor/node_modules` (8,919 files / 67 MB) is present on disk and **correctly not committed** —
`git ls-tree -r HEAD -- vendor/` returns exactly 5 files.
