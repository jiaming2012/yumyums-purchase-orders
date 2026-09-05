# Merge intent — card `refusal-holds-before-sync` (run 20260906-2, Activity B)

Branch `card/c1-refusal-holds-before-sync`, cut from `overnight-20260906-2` at `1f7d384`.
Closes **B-432** (the §8 refusal fails OPEN before the campaigns replica has delivered) and
disposes **B-434** (a)(b)(c). Goal ledger:
`.night-crew/knowledge/spikes/activity-b-offline-first-replica/refusal-holds-before-sync.md`
(three spikes green first-run; six build-facts binding).

---

## 🛑 What the refusal is, and in which window it holds

**Do not read this card as "the refusal is now unconditional."** That exact overclaim is how
B-432 hid for a whole run — run `20260906`'s merge intent wrote *"no override, **for anyone**"*
and the sentence was false during the campaigns-replica gap.

What lands here is a **predicate**, and it is worth stating as a predicate:

| the scanned code | the campaign policy | offline override | why |
|---|---|---|---|
| KNOWN (`campaign_id` non-null), campaign **in** the replica, `requires_online=true` | resolved, true | **refused** | §8, unchanged (run 20260906) |
| KNOWN, campaign **in** the replica, `requires_online=false` | resolved, false | offered | not over-refusing (spike 02 run 5) |
| **KNOWN, campaign NOT in the replica** — empty, still delivering, erroring, or a brand-new campaign whose codes arrived first | **unresolved** | **refused** | 🆕 **this card** — the B-432 window |
| genuinely-unknown code (`campaign_id` null — nothing to look up) | not asked | **offered**, with the F2 unverified warning | **decision 166, deliberately preserved** |

The last row is the one the refusal does **not** cover, on purpose. The fail-closed arm keys on
`campaignId != null`, so a genuinely-unknown code can never enter it — decision 166's ratified
F2 affordance survives *by construction*, not by care.

**One window is knowingly left open and is NOT closed here** (see "Carried", below): when the
policy source itself failed to construct (`CAMPAIGN_POLICY === null` — a stale-cached
`scan-page.js` with no `campaigns` collection), `submit-flow.js` still coerces to `false` and
a known code stays overridable. That path is unchanged from `20260906`, is measured by the
landed `campaigns-run.sh` leg 3 negative ("provably dead without a source"), and is now
recorded in the attempt row as `policy_unresolved = true`. Filed as **B-436**.

---

## Shared files touched (each outside this card's own packages, one line of why)

| File | Why |
|---|---|
| `marketing/sync/replicas.js` | `createCampaignPolicySource` becomes fail-closed for a KNOWN-but-unresolved campaign (build-fact 3) and gains the `error$` latch + readiness settle (build-fact 1/2). |
| `marketing/submit-flow.js` | The policy seam carries the unresolved bit to the render + the attempt row; `setCampaignPolicy` gains an optional second arg; new `blockedOffline` branch copy (build-fact 6). **Zero machine changes.** |
| `marketing/scan-page.js` | Owns the policy source now (it owns the collections *and* the replicas), so `error$` can be subscribed **before** `startCampaignsReplica`'s handle does anything (build-fact 1: `error$` does not replay). |
| `marketing/sync/push-replication.js` | `SCAN_ATTEMPTS_SCHEMA` v0→v1 + migration strategy, `enqueueAttempt` plumbing, ONE landing-body whitelist line (build-fact 5). |
| `marketing/sync/harness/campaigns-harness.mjs` | Its leg-3 readiness guard (`policyFor(...) !== null`) is vacuous under fail-closed — re-pointed at `size()`. Behavioural assertions untouched. |
| `vendor/rxdb.bundle.js` + `vendor/src/rxdb-hq-entry.mjs` | The schema version bump is inert without `rxdb/plugins/migration-schema`: rxdb 17.4.0's `createRxCollection` does `version!==0 && await migratePromise()`, and the un-plugged prototype **throws** — the Scan page would brick on every device. Regenerated at the SAME pins (rxdb 17.4.0 / supabase-js 2.109.0). |
| `supabase/migrations/20260906000300_*.sql` | NEW numbered file. `policy_unresolved` column + rider (a)'s constraint tightening. Never an edit to a landed migration. |
| `sw.js` | Regenerated after the change set (B-13/B-37) — `vendor/rxdb.bundle.js`'s content hash moved. |
| `tests/marketing.spec.js` | The red-first test + the discriminator + rider (c). |
| `.night-crew/knowledge/{BACKLOG.md,roadmap.md,spikes/…/refusal-holds-before-sync.md}` | B-434 disposition, card flip, GAP-1's `validated:` line. |

---

## What MUST survive any merge

1. **The fail-closed predicate keys on `campaignId != null`.** If a merge ever makes the
   fail-closed arm reachable for a null campaign id, decision 166 is silently deleted and every
   walk-up customer's code stops being overridable offline. The `F2:` test is the guard.
2. **`policyFor(null)` still returns `null`.** The card text binds this; `submit-flow`'s
   `null → false` coercion is what keeps the F2 affordance alive.
3. **`error$` is subscribed BEFORE the replica handle is used** (`scan-page.js`'s `startSync`
   subscribes synchronously on the line after `startCampaignsReplica` returns, with no `await`
   in between). Build-fact 1 measured **zero** replayed emissions to a late subscriber — move
   the subscribe behind an `await` and the latch silently never latches.
4. **Zero new (state,event) pairs in `submit-machine.js`.** Nothing in this diff touches the
   machine; Card 6's 460-pair strictness proof and `tests/machine/` are untouched. The fix
   lives entirely in the policy **answer** and in the render.
5. **The migration is in the tree BEFORE any client sends the field.** Spike 03 measured the
   pre-migration landing body drawing HTTP 400 `PGRST204` — the F-2 throw-retry head-of-line
   poison class. The commit order on this branch is migration → client, deliberately.
6. **`SCAN_ATTEMPTS_SCHEMA.version === 1` and the vendored `migration-schema` plugin ship
   together.** Either alone bricks the Scan page: schema v1 without the plugin throws at
   `addCollections`; the plugin without v1 is dead weight but harmless.
7. **GAP-1's two belts and the F-2 guard in `push-replication.js`** — untouched. The landing
   whitelist gains exactly one key on the `land-unverified` path.
8. **`scan_attempts_names_a_code`** still refuses a row that names neither a code nor a hash;
   rider (a) only *tightens* it (adds `offline_override`).

## What is safe to drop

- The new `data-branch="requires-online-unresolved"` **copy wording** — the branch VALUE is
  asserted by the e2e, the sentences are not. Reword freely; keep "can't verify" and "try
  again" (the roadmap's done_when quotes them) and keep it from claiming the campaign is
  known-high-value, which is the UI-R3/R6 defect this branch exists to fix.
- The discriminator **field name** `policy_unresolved` — the spike offered it as a candidate and
  the card allows a rename. If it moves, it moves in the migration, the schema, `enqueueAttempt`,
  the landing body and the e2e together.
- `campaignPolicyFor`'s debug surface, if a later card replaces the policy plumbing wholesale
  (it now has a test, so the test goes with it).
- `marketing/sync/harness/refusal-run.sh` + `refusal-harness.mjs` — the GAP-1 validation run.
  Keep the ledger's `validated:` line; the scripts are re-runnable evidence, not production.

## Nothing here

- **Go / `backend/`:** nothing here. No Go file is touched; G1/G2(Go) are skipped with that
  reason stated.
- **New terminal status:** nothing here. `status='accepted'` + flags, per §9/§19 — the PARK line
  was never approached.
- **New preference category or `night-crew.toml` key:** nothing here. The `"marketing"` seam
  already covers `marketing/`; `supabase/` is undeclared, which de-confines this card to the
  FULL Playwright suite (the B-37 safe direction, priced into the slate).
- **`openspec/`:** nothing here — `openspec: absent`, G3 N/A.

---

## Red-first

**Captured BEFORE any production code was mutated**, on `1f7d384` + a test-only addition.

- **Test:** `tests/marketing.spec.js` › *"B-432: a requires_online=true code is refused while its
  campaign is UNRESOLVED (branch-3 minus the campaigns: seed)"* — the shipped branch-3 e2e with
  **only** the `campaigns: [campaignHighRow()]` seed removed. The admin holds
  `marketing-offline-override`; the $40 `requires_online=true` code (…0005 → campaign …0002,
  committed seed literals) is in the codes + offers replicas; the reachability probe is killed.
- **Command:**
  `npx playwright test tests/marketing.spec.js -g "B-432" --retries=0`
- **Result:** `EXIT=1`, `1 failed`.
- **The observed defect, not a paraphrase:**

      Expected: "requires-online-unresolved"
      Received: "override"
      9 × locator resolved to <div id="ms-gate" data-branch="override">…</div>

  The gate offered **Force submit (offline)** on a must-verify-online $40 code. That is B-432,
  reproduced in the browser, on the shipped policy path.
- **Full log:** `.night-crew/runs/2026-09-06-2-autonomous/c1-red-branch3-nocampaign.log`
- **Green after:** same command, same filter, on the fixed tree —
  `.night-crew/runs/2026-09-06-2-autonomous/c1-green-branch3-nocampaign.log`.
- **Second red (the harness half, spike 02 row 3 re-proved against the SHIPPED policy source):**
  `marketing/sync/harness/refusal-run.sh red-preserved` re-runs the validation harness against
  the pre-change `null`-answering policy shape and must exit NON-ZERO; the green mode runs the
  shipped source. Logs under the same run directory.
