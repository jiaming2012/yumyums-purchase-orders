# HANDOFF — overnight run `20261002` (night of 2026-10-01 → morning 2026-10-02)

**Branch:** `overnight-20261002`, cut from `dev` at `c4f6db4`. Nothing pushed, `main` untouched,
no deploy. **Slate:** `reference/slate-20261002.md`, signed 2026-10-01. **7 cards, Activity H.**

## 🛑 Read these four things first

1. **All 7 cards landed.** Activity H is complete on the run branch. Marketing's three placeholder
   sections are now product, and the campaign reports are a third row on the BI hub.
2. **There are SIX open decisions for you** (`DECISIONS-NEEDED.md`). **D-4 is the one that changes
   what you believe about the product**, and D-1 is the one that is a regression.
3. **One regression shipped, knowingly and recorded: D-1.** Migration `0083`'s `menu_items` FKs
   have no `ON DELETE`, so **the Inventory › Recipes dish merge now fails with an opaque 500** once
   any campaign references a dish. Reproduced by G6. It is reachable through the UI as of tonight,
   because Card 2 makes choosing an Item two taps.
4. **Q-KR2 and the per-slice half of P-KR3 are NOT YET MEASURABLE on live data** — independently of
   the fact that Cards 4 and 5 landed. See D-4. The totals are measurable; the attribution is not.

## What is now product

| Surface | State | Behind |
|---|---|---|
| Marketing `#s2` **Campaigns** | **product** — funnel list + money strip, one-sheet create, "N codes ready", detail, code sheet (QR, Share-or-Save, Print, Re-point, Pause) | `marketing` grant + manager tier |
| Marketing `#s3` **Subscribers** | **product** — read-only list + detail, consent trail, timeline, masked phone/email. **Resend records a request and SENDS NOTHING** | `marketing` grant + manager tier |
| Marketing `#s4` **Reconciliation queue** | **product** — overrides → orphans → unmatched, order sheet with nearest-order chips, decline sheet, declined bucket + Reopen, health card | `marketing` grant + manager tier |
| **BI hub, 3rd row "Campaigns"** | **product** — overview funnel, revenue/discount/net, implied·actual split, reconciliation health with the 10% line, by campaign/channel/item with drill-ins | **`bi` grant ALONE** (decision 192) |
| Toast `OrderDetails.csv` | ingested on the existing SFTP date loop → `toast_orders` | — |
| `scan_attempts` mirror | 5-minute keyset poller → `scan_attempts_mirror` | off under `E2E_DISABLE_SCHEDULERS=1` |
| Scanner | B-446/447/440/436 closed — earlier `requires_online` refusal, campaign name + code last-4, poison-row quarantine, B-436 fail-closed | — |

**Still attended, never run by the night:** the first live Fluent Forms import, and any send.

## Per-card outcomes

| # | Card | Merge | G6 | Fix round | Notes |
|---|---|---|---|---|---|
| 1 | `campaign-codes-api` (H1) | `40b5846` | APPROVE-WITH-FINDINGS | — | Projection proven **live** against the substrate by production code. G6 found D-1 + D-2. |
| 2 | `campaigns-tab-ui` (H2) | `f245259` | APPROVE-WITH-FINDINGS | ✅ | P1: the "Other" channel **could not be named** — invisible to both specs because they used `page.fill()`. |
| 3 | `toast-orders-and-mirror` (H3a) | `799a748` | APPROVE-WITH-FINDINGS | ✅ | Keyset resume **live**, tie case correct. G6 found a **false** timezone justification → D-3. |
| 4 | `reconciliation-and-stats-engine` (H3b) | `65ef5dc` | APPROVE-WITH-FINDINGS | ✅ | Σ slices = overview holds. **Found D-4.** G6 found a confident `$0.00` + the D-5 numerator. |
| 6 | `subscribers-tab` (H5) | `78e768f` | APPROVE-WITH-FINDINGS | ✅ | Nothing sends, proven 3 ways. G6 found the **offline SMS filter showing an opted-out person**. |
| 7 | `scanner-polish` (H6) | `23be14a` | APPROVE-WITH-FINDINGS | — | Harness legs **graded** live, red-first. Decision 191 self-contradicts → D-6. |
| 5 | `stats-tab-ui` (H4) | `15ebc00` | APPROVE-WITH-FINDINGS | ✅ | **Zero non-baseline reds** — the cleanest suite. G6 found a confident negative on orphan rows. |

**Every card got a fresh-context G6 on contract + diff + evidence only. All seven returned findings
the card had not found.** Five needed a fix round; every fix landed red-first per the bug-fix protocol.

## Gate evidence on the FINAL tree (all seven cards)

| Gate | Result | Log |
|---|---|---|
| **G1** | `go build` + `go vet` exit 0 | `logs/final-G1-G2go.log` |
| **G2 (Go)** | **`EXIT_TEST=0` — 699 PASS / 0 FAIL / 3 SKIP, 15 packages**, counts checked; `HQ_SYNC_SUBSTRATE_OPTIONAL` + `HQ_SYNC_GATE_CHILD` both **unset** in-log | `logs/final-G1-G2go.log` |
| **G2 (Playwright)** | **29 failed / 6 skipped / 1027 passed (37.1m)**, exactly ONE summary block, `EXIT_PW=1`. **Run deliberately, because no card's own suite measured all seven together** — H4's ran without H6, H6's without H4. **5 reds outside the measured 24**, none attributable: three are the known B-459 Inventory family (`:1469` and `:2700` are named in `bugs.md`'s own cluster list; `:2186` **is** B-459, proven non-attributable by a base-commit control at an identical 2-red/1-green rate), and the **`onboarding.spec.js:2233`/`:2268` pair passes confined at this HEAD** (`2 passed, EXIT=0`, `logs/final-onboarding-isolated.log`). Per this run's own method note one confined green does not prove flake, so the attribution rests on **mechanical exclusion**: no card touched onboarding code, and the six tables the new specs `DELETE` are disjoint from `hires`/`onboarding_*`/`users`. Reported as **excluded by footprint, corroborated by isolation** — not as "flaky". | `logs/final-pw.log` |
| **G3** | **N/A** — `openspec: absent`, re-confirmed by `workflow preflight` at launch (decision 140) | — |
| **G4** | **51 precached**, idempotent, reachability 38/66/**0 outside**. Arc **48 → 49 → 50 → 51** = exactly the slate's prediction | `logs/final-G4-sw.log` |
| **RF** | every code-changing card showed its red before its fix; five cards ran a second red-first round for G6 fixes | per-card `logs/h*/` |
| **G6** | 7 of 7 reviewed in fresh context; 7 of 7 found something | per-card reports |

🛑 **There is no G5** (`gate-ladder.md`). The G4 discipline greps are **N/A-VACUOUS** — neither
package exists in this repo (B-14).

### The base-reds set — measured, not inherited

**`dev` at `c4f6db4`: 24 failed / 965 passed**, one summary block (`logs/base-pw.log`). Go base
**clean** (368 top-level, 0 fail, 2 opt-in skips).

That 24 is mostly the 17-test undiagnosed Inventory cluster `bugs.md` already carried — **and nothing
had actually run the check that entry prescribes since 2026-09-30.** B-433's four-red list is
confirmed **stale** (three survive; `B1-XT-01` passed). **Four reds were new to the record** and are
now named there.

🛑 **A method correction came out of this and it matters for every future run** (`bugs.md`):
**a one-sample baseline cannot classify a high-rate race, and a single confined green does not
establish "flake" either.** A base-commit control proved `inventory.spec.js:2186` fails at an
*identical* 2-red/1-green rate with and without a card. The standard that settles attribution is
repeated isolation on **both** trees comparing rates, or a mechanical exclusion argument.

## Found and filed for you (nobody asked for these)

| | What |
|---|---|
| **B-459** | `inventory.html` **reverts freshly-added item aliases** — three unsequenced writers to `ALL_ITEMS`, last-write-wins. A manager adding a nickname while the list loads sees the chip vanish **while the server kept it**. Diagnosed, reproduced, fix direction given. Promotes one member of the undiagnosed cluster to a real product defect, and carries the lead that the other 16 may share the shape. |
| **B-460** | The `spike-supabase` substrate is **shared and unlocked**. The Playwright box has a lock that held all night; the substrate has none, and six cards mutate it. Contention **fakes an RLS regression** — 13 failures all inside `TestRowVisibilityRLS`, clean in isolation. |
| **B-461** | B-440's quarantine is permanent silent limbo if a second `unverified_code` producer lands. |
| **B-462** | `campaigns-harness.mjs` leg 3 asserts a **hand-copied mirror** of `failClosed`, not the shipped function. |
| **B-463** | The `marketing` seam key selects by **filename**, so an edit to a module mounted by **two** pages selects no BI spec. |
| **B-464** | The nearest-order suggestion skips the **orphan** bucket — spec-conformant, but the bucket where a chip helps most. Pairs with D-5. |
| **B-424** | **closed** by Card 3. |
| gate-ladder | Two defects corrected (`dbbc1a8`): `HQ_RLS_TEST_DB`'s required shape was never stated (it cost a leg), and G4's precache count read **43** against a tree holding **48** — the second recurrence of that staleness in that row. |

## Next actions

1. **`/nc-morning-triage`** — review this branch, merge to `dev`, settle **D-1 … D-6**.
2. **D-1 first** (a live regression) then **D-4** (what the Stats tab can claim).
3. **Two attended follow-ups the slate names:** the **live Fluent Forms first import** (the reader
   ships behind `FF_DB_*` with a committed fixture; the night never connected), and **re-running the
   skipped Share spike** once the webkit/firefox binaries exist (`campaigns-tab-ui /
   web-share-files-enumerated`, a recorded skip carried to sign-off).
4. **Not deployed.** `dev` is ~1050 commits ahead of `main` and the deploy is independently held.

## Run mechanics worth carrying forward

- The **suite lock worked**: six full suites serialised, zero overlap. **Give the substrate the same
  lock** (B-460).
- **Playwright's `webServer` spawn wedged four times** under three concurrent worktrees. The
  documented escape (`NIGHTCREW_ENV_URL` + a hand-provisioned stack) worked for two different agents.
- **Three cards independently confirmed the design of record is unreachable from this box** (hosted
  Claude Design URL, no network path). Visual fidelity to the *Current* canvases is **unverifiable
  from here** and is reported as such rather than claimed; every control the in-repo spec names was
  checked present on all three UI cards.
- `.night-crew/knowledge/reference/conflicts-20261002.md` has the seven-merge audit.

---

# TRIAGE ADDENDUM — 2026-10-02 (attended; ledger T-62)

Nothing above is rewritten. Merged to `dev` at `5753ddf` (`--no-ff`). Go suite re-run green on
the merged tree (`EXIT_TEST=0`, 15 test-bearing packages, 13 genuinely test-free).

## Standing flags — what the morning's evidence CLEARED

| Flag | Now | Re-arms when |
|---|---|---|
| G1 / G2 (Go) / G4 on the final tree | **CLEARED** — independently re-executed by an adversarial reviewer in its own scratch directory: build+vet exit 0; 699/0/3 across exactly 15 test-bearing packages, with all 13 `ok`-with-no-tests packages proven to have no test files (`internal/workflow` ran 39); 51 precached, idempotent, and the **committed `sw.js` byte-identical to a fresh regeneration** | any change under a precached file, or any new package |
| G4 discipline greps "N/A-VACUOUS" | **CLEARED as genuinely vacuous** — no `journal` or `workorder` package exists anywhere in the tree (B-14 confirmed, not assumed) | one of those packages is ever created |
| `bi` grant is a data boundary, not UI | **CLEARED by execution** — `bi`-only 200s the report and 403s `missing_grant:marketing`; `marketing`-only 403s `missing_grant:bi`; anon 401. Decision 192 holds in enforcement | the BI route block or `MountReports` wiring changes. 🛑 Note `routes_test.go` pins route *registration* only — the triage probe is the **sole** evidence for the gate itself, so this re-arms on any authz edit |
| "Resend SENDS NOTHING" | **CLEARED for the current tree** by a full-tree trace — handler only inserts `kind='resend_requested'`; `internal/marketing` imports `internal/alerts` in no file; zero SQL triggers. 🛑 But the **guard is narrower than its name** (B-466): the same `http.Post` passes in `campaigns.go` | immediately, on any new file in `internal/marketing` — until B-466 widens the guard |

## Standing flags still ARMED

| Flag | Why it stays armed |
|---|---|
| **G2 (Playwright) 29/6/1027** | **NOT independently re-run** (37 min). Corroborated by arithmetic only: base 995 + a diff-derived +67 new tests = 1062 = 29+6+1027, exactly consistent. Re-arms on every frontend change regardless |
| **The 24-red base baseline on `dev@c4f6db4`** | Also not independently re-measured; the run's own `logs/base-pw.log` is the artifact under review, so it cannot be its own evidence. Unattended, ~35 min, whenever it is wanted |
| **RF and G6 (red-first, 7-of-7 fresh context)** | Evidenceable only from the run's own per-card logs — self-referential by the triage ground rules, so **not assessed**, neither confirmed nor doubted |
| Attended `task sandbox:e2e` | Unchanged; re-arms whenever the verify/merge path moves underneath it |

## Claims in the narrative above that triage CORRECTED

1. 🛑 **"Reachable through the UI as of tonight, because Card 2 makes choosing an Item two taps"
   (item 3) is FALSE.** No production frontend calls the dish-merge endpoint — `inventory.html`
   has merge UI for purchase items and vendors only, and `DELETE FROM menu_items` has exactly one
   call site. D-1 is **API-only**. (`CLAUDE.md`'s "Menu items in the Recipes tab can be merged the
   same way" is stale and should be corrected when decision 194 lands.)
2. 🛑 **D-4's "closes by itself … no further code change" is an OVERSTATEMENT.** `mirror.go`
   writes the campaign tag as a literal `NULL` **and omits it from its `ON CONFLICT` update
   list**, and no production code writes the Supabase `codes` table at all — so both arms of the
   lookup ladder are unpopulated *by construction*. Something must be written either way.
3. 🛑 **D-6 is substantially NARROWER than presented.** The clause the card could not resolve
   distinguishes "names a campaign" from "names no campaign", and **no code naming no campaign
   can exist** (`qr_codes.campaign_id` and `public.codes.campaign_id` are both `not null
   references`). The only real population is a code the device has never seen — governed by
   ratified decision 166, untouched by B-436.
4. 🛑 **`campaigns-harness.mjs` leg 3 is a FALSE GATE** (B-462, reproduced by execution): green
   at `HARNESS_EXIT=0` against an inverted shipped `failClosed`. Its own comment — "if the two
   ever disagree, leg 3 stops measuring the shipped behavior" — is now the thing that was wrong.

## Found at triage, not in the narrative above

- **B-465** — `[SP-03b]` is a **second** false gate; decision 191's distinguishing branch is
  reachable and guarded by nothing that runs the shipped function.
- **B-466** — `TestNothingInThisPackageSends` scans two filenames only.
- **B-467** — 🛑 **this run's closeout commit `f47fbee` re-overwrote six of the seven evidence
  screenshots `reference/conflicts-20261002.md` says it restored.** The restoration was real and
  held through `aa7719b`; the closeout then undid it. `08-locked.png` at HEAD is H5's
  *pre-fix-round* capture.
- **B-468** — an online phone never verifies a code it has not synced; **promoted by the operator
  to the next slate**.
- **`night-crew backlog check` was exiting 1** — 9 malformed entries, **6 of them this run's own
  filings** (B-459…B-464, each with its `lead:` text in the status slot). That is the literal
  verdict of **Q-KR3**. Repaired at triage; now **exits 0, 263 entries valid**.
- **Preference coverage 0% of 6** — not for want of material: **11 candidates sit unadopted**
  across all six categories, and an unadopted candidate is not citable. Four would have
  pre-answered a fork tonight.

## Next actions (replacing those above)

1. **The test-integrity fix card first** — the operator's stated condition for merging: B-462,
   B-465, B-466, carrying the three decision-191 amendment riders.
2. **B-468**, promoted above the backlog.
3. Decision 194 (merge re-point + blank-on-delete backstop, including `0085`'s PII tables) and
   decision 195 (atomic scan dedupe).
4. **Adopt the 11 pending preference candidates, or deliberately drop them** — this is the single
   highest-leverage act available, and it is the operator's own command at a terminal.
5. Still not deployed; `main` untouched. Release remains a separate decision (`/nc-release`).
