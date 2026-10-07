# Roadmap — "Close the loop" cycle (a scannable code becomes money-tied, campaign-attributable revenue — offline, at the window)

> **Cycle:** Close the loop — a code issued to a customer is redeemed at the truck window against
> a real Toast order and tied back to the campaign that produced it, and it works when the LTE
> hotspot is down. **Traces to:** `.night-crew/knowledge/okrs.md` (Product / Delivery /
> Engineering / QA, authored in the same sitting per DESIGN §15j.42) and the design of record
> `docs/qr-offline-redemption-handoff.md` — the **§19-addendum version**, synced this round from
> `~/projects/yumyums/marketing/qr-redemption/` (design agreed 2026-09-03, decomposed here; §19
> adds the decided scanner forks F1–F6 and the `unverified_code` column).
> **Produced:** 2026-09-03 attended `/nc-roadmap-round`, at the milestone boundary. Previous
> cycle ("Prod current and honest") archived at
> `reference/roadmap-2026-09-03-prod-current-and-honest.md` +
> `reference/okrs-2026-09-03-prod-current-and-honest.md`; its close at ledger T-50 (close line,
> marker `hq-20260903`, 12 MET · 0 PARTIAL · 2 NOT MET). No `/nc-retro` was run for that cycle;
> this round proceeds on the close record, the backlog, and the OKR grades, and records the
> absence.

## Current state (2026-10-05, milestone "Close the loop" — 37 cards, **27 green, 7 white**, 3 retired)

> **Same sitting, ledger T-71: the owner removed "Scan from photo" (decision 213) — the crew scans
> with the camera only.** That closes the run's parked question. The next Activity E slate is
> three cards: remove the photo control (B-496), then `identity-code-and-qr` with its gates
> restated, then `mms-send-on-signup`. Before it: read the owner's live-camera check (B-497).
>
> **Morning triage 2026-10-06 (ledger T-70): run `20261006` landed nothing — Activity E's two
> cards stay PLANNED.** The code card was built and parked on a measurement: the tablet's "Scan from
> photo" misses about one customer code in seven (B-488). The owner checks the truck's tablet
> before a fix is chosen (decision 210; kit at `runs/2026-10-06-autonomous/tablet-check/`). The
> milestone tally is unchanged, and the close bar now needs at least one more Activity E night.
> Slate `20261007` (Activity K) launches as written (decision 211).
>
> **Two slates signed at the evening sitting of 2026-10-05.** Slate `20261006` (ledger T-68):
> Activity E's two cards ride tonight — the LAST overnight the milestone's close bar needs. Slate
> `20261007` (ledger T-69): **Activity K — the review leftovers (B-482 … B-487) and the Inventory
> Setup races (B-459, B-478), four cards — launches after `20261006`'s morning triage merges**;
> the operator asked for the bug fixes as a second slate after the milestone night. Activity K is
> QA debt and crew-visible polish, not close-bar work: the milestone's bar is unaffected by it.
>
> **Slate `20261006` (ledger T-68): Activity E's two cards ride tonight — the LAST overnight the
> milestone's close bar needs.** The operator chose it over a
> leftovers night ("I want this last loop to clear the milestone"); both goals were spiked and
> closed inline (one signed correction: the QR encodes at Low with a 16-char token). The same
> sitting retired `redemption-unknowns-spike` (ABSORBED → Activity H — its questions are answered
> by the landed reconciliation; #6 is moot) and swapped the delivery provider to SignalWire
> (decision 208, T-67). After tonight the only white card is `external-accounts-provision`, and
> the rest is the operator's attended acts (D-KR1's live send; close-bar legs 1 and 2).
>
> **Activities I and J are complete and triaged.** Run `overnight-20261005` landed three of three
> cards (merge `57a2083`): Activity I's last, `atomic-scan-dedupe`, and Activity J's
> `photo-scan-guard-and-lookup-arms` + `merge-target-and-blanking-guards`. Triage is recorded at
> ledger **T-66** (decision 206 — the next launch drives night-crew's run loop; no night here has
> yet). The review's leftovers are B-482 … B-487; the Inventory Setup races B-459 + B-478 remain
> the obvious next small card.
>
> **Activity J was authored at the slate sitting of 2026-10-04** (two cards from triage T-64's
> findings — the scanner photo-pick stuck state B-475 with the scan-time check's untested arms
> B-476, and the merge-target / migration-blanking guards B-479 + B-480), slated with Activity I's
> remaining `atomic-scan-dedupe` as `reference/slate-20261005.md`.
>
> **Activity H is complete and triaged.** Run `overnight-20261002` landed seven of seven cards
> (merge `5753ddf`); its morning triage settled six forks and is recorded at ledger **T-62**
> (decisions 194–200). Campaign admin, subscribers and the designed stats tabs are in.
>
> **Activity I:** run `overnight-20261003` landed three of four cards (merge `cf0a482`, ledger
> **T-64**, decisions 203–205); `atomic-scan-dedupe` landed with run `20261005` above.
>
> **The remaining white cards are the cycle's two open fronts.** Activity 0
> (`redemption-unknowns-spike`, `external-accounts-provision`) still gates the spine, and
> Activity E (`identity-code-and-qr`, `mms-send-on-signup`) is customer delivery. Two cards are
> retired rather than white: `reconciliation-view` **ABSORBED** into Activity H (T-60) and one
> **SUPERSEDED**; neither counts toward the 31-card tally's green/white split.
>
> **The close bar is unmet and is not a card.** P-KR1 requires the operator to personally observe
> all three legs of a real redemption; no card status, KR grade or closeout substitutes for that
> dated ledger line, and the milestone may not close without it (`okrs.md`).

## Why this cycle exists

The origin is a **marketing-attribution gap, not a scanning problem** (handoff §1). Google Ads
records ~496 local actions across six goals — 75 direction requests the best proxy for someone
walking up to the truck — and **none of it ties to revenue.** A redemption is the only event in
the funnel that is simultaneously (a) attributable to a campaign and (b) tied to real money.
This cycle builds the thing that closes that loop, under the one constraint that shapes every
decision: **the truck runs on a flaky LTE hotspot, and the write that matters happens exactly
when connectivity is worst** (handoff §2).

Secondary outcomes the same build delivers: a durable customer identity across visits (one
identity code, §10) and a measurable repeat rate.

## The operator's acceptance criterion

> *As the owner spending on ads with no idea which spend produces revenue, I want a code a
> customer can present at the window — redeemed against a real Toast order and tied to the
> campaign that sent them — so that I can finally see whether ad spend turns into money, and it
> has to work when the hotspot is down.*

**The close bar, chosen at this round — three legs, all operator-verifiable:**

1. **A real redemption, end-to-end.** The operator issues one test identity code, receives it as
   an image on their phone, opens HQ → Marketing → Scan on a second phone, scans it, sees the
   entitlement, types a Toast order number, submits **online**, and watches it burn — then scans
   the *same* code again and sees **"already used."** One code, one redemption, single-use proven
   by observation, not by test alone.
2. **The join lands.** That redemption shows up **`matched`** in the reconciliation view after
   the Toast report ingests (T+1), joined on `(business_date, order_number)`, with the orphan
   rate visible. Attribution is closed, not asserted.
3. **Offline is safe by policy.** With the device's *server reachability* deliberately killed
   (not `navigator.onLine`), a `requires_online = true` campaign **refuses** submit
   ("can't verify — try again"), and a `requires_online = false` campaign **offers the
   permissioned override** behind the §13 confirmation, writing an audit-flagged
   `offline_override` attempt that the server reconciles on the next sync. Both branches observed.

   🛑 **BLOCKED as of morning triage 2026-09-05 (T-54, decision 170) — leg 3 is NOT attestable
   on the current build.** The refusal **fails open while the campaigns replica lags**: codes,
   offers and campaigns are three independent replicas started together, nothing gates scanning
   or submitting on initial replication, and an as-yet-unreplicated campaign coerces to
   "override allowed". Reproduced at triage with **zero production code mutated** — removing only
   the `campaigns:` test seed leaves a `requires_online = true` $40 code rendering
   `data-branch="override"`. So the leg's own promise ("refuses submit") holds only once the
   tablet has finished syncing, which is precisely the condition leg 3 exists to test. Filed
   **B-432**; unblocked by the card below.

   > ✅ **ATTESTED 2026-09-06 (ledger T-58) — leg 3 is CLOSED.** The operator personally ran
   > both branches on a real phone (attended remote sitting: Claude drove the dev stack, the
   > operator held the device), with server reachability genuinely killed at the process level
   > while wifi stayed up. Branch 1: the `requires_online=true` $40 code **refused** — "online
   > verification is required. There is no offline override for this campaign (§8) — not even
   > for a manager." Branch 2: the `requires_online=false` $10 code offered the **§13
   > confirmation** behind the operator's #12 entitlement ("Forcing it risks a double-redemption
   > and is flagged for review"), the forced attempt queued offline and pushed on reconnect, and
   > the landed row carried the full contract: `offline_override=t`,
   > `override_by=jamal@yumyums.kitchen`, `policy_unresolved=f`, `device_id` = the mint `sub`.
   > B-432's pre-sync window was also exercised for real: a fresh page load that had never
   > synced this session offered **no submit path at all** (fails closed). Findings filed
   > B-446–B-449.

🛑 **The milestone may not close until the operator has personally seen all three.** No KR grade,
card count or closeout substitutes — the standing "dev complete means the operator ran it" rule
(decision 161's class).

**De-risking note on leg 1's delivery:** MMS delivery depends on A2P registration, which has a
1–3 week external lead (§11). To keep the close bar off that critical path, Activity 0 registers
a **toll-free number** (skips 10DLC, faster verification, fine for one-way sends — §11, decision
#4) so leg 1's "receive as an image" is provable within the cycle even if a 10DLC campaign is
still pending.

## How this roadmap works

- **Activity-level cards**, WO-sized, each carrying a module footprint and a KR trace.
- **Status:** `DONE` / `LANDED` (green — the card's run merged to `dev`; `DONE` is the word the
  `20260904`–`20260906` runs used and `LANDED` the word every run from `20260906-2` on uses — one
  state, two spellings) · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED` ·
  `SUPERSEDED` / `ABSORBED → <card>` (retired — the work moved into a named card or is no longer
  needed; neither green nor white, and outside the Current state tally's green/white split).
- **Build order is load-bearing.** Activity 0 (unknowns + longest external leads) gates the
  spine; Activity A (the Supabase arbiter) gates every client that reads or burns a code;
  Activity B (the offline replica) gates the scanner's offline reads; the scanner (C) gates the
  server machine (D) and delivery-driven volume (E); the join (F) needs codes actually redeemed
  to have anything to reconcile. Activity G (carried QA debt) is **disjoint-footprint and
  overnight-parallel** with everything.
- **Red-first (gate RF)** per decision 153: every code card records the named test red on the
  pre-change tree, then green. Docs/registration/attended cards record `n/a — no code change`
  explicitly. **Greenfield note:** most cards here create new files, so "red on the pre-change
  tree" means the new test fails because the behavior does not yet exist — name it and show it.
- **New-stack reality (retro-in-advance).** This cycle adds Supabase (hosted Postgres + Realtime)
  and SignalWire (decision 208 — replaces Twilio, 2026-10-05) to an app that is otherwise
  static-HTML + Go + self-hosted Postgres. That is a real
  surface-area increase; it is the tradeoff the handoff §3 accepts. Cards that touch the new stack
  carry their external-dependency note explicitly so an overnight leg never discovers a missing
  account mid-run.
- 🛑 **Tests run on `:5434` (`yumyums-test-pg`), never `:5433`** — standing rule, decision 155.
  The Supabase work runs against a **separate Supabase project / local Supabase**, never the
  production project; the schema-and-race card states its target coordinates read-only before any
  write (the §"Prod safety" habit, applied to the new arbiter).

## Decisions resolved at this round (so the build isn't blocked on them)

Two calls were the round's to make; both are engineering-level and stated here rather than
carried. Both are **revisitable in Activity 0's spike** if field observation (#6) changes the
device topology.

- **R2 — where redemption is orchestrated.** The scanner's **online submit** posts to a new HQ
  Go endpoint that drives the §18 **gstate** machine, which `Invoke`s Supabase's atomic
  `redeem()` RPC. So HQ's Go backend is the *orchestration* layer (consistent with the rest of
  HQ, and where §18's machine lives), while Supabase's `redeem()` remains the **sole** single-use
  arbiter (§18 edge-case 1 — the machine never re-checks; it reacts to the 1-row/0-row verdict).
  The **offline** path is unchanged from §6: the RxDB push handler runs client-side and calls
  Supabase directly on reconnect, and its synced `offline_override` rows are reconciled
  server-side by the same machine. This is what makes Activity D load-bearing rather than
  decorative. (Neighbours handoff open-decision #11; #11's *campaign-admin* half stays open.)
- **R1 — the RxDB "reuse" is greenfield.** §12/§16 say the scanner "reuses existing RxDB
  infrastructure." The library may be vendored, but the sync-rxdb **cutover never happened in
  prod** (last cycle's Activity 4: "no page calls `startHQReplication`"), and Supabase
  replication is a **separate replication target** regardless. Activity B is therefore planned as
  **new** replication work, not a reuse — sized accordingly.
- **R3 — replicate all non-expired offers; it is not bloat (operator challenge, arithmetic).**
  Storing every customer's *active* offers on each tablet costs ~1 KB per doc all-in (RxDB /
  IndexedDB overhead included), so **1,000 customers × 2 non-expired offers ≈ 2,000 docs ≈ ~2 MB**
  — a fraction of a percent of a phone's storage budget, and on the order of the `codes` replica
  already carried (§5.3). The pull filter `expires_at > now()` bounds the set by *active* offers,
  **decoupled from lifetime customer count** (even 5,000 active offers ≈ ~5 MB). So the
  **replicated offers are the primary offline path** (full list for any synced customer) and the
  QR's embedded offer (D-KR3) is the **fallback** for the not-yet-synced case (a just-signed-up
  walk-up while the truck is offline). The earlier "bloat" framing is retracted.

## Addendum §19 — decided scanner forks (the build contract for Activities C, D, F)

The design of record carries a §19 addendum (a domain-driven statechart model) with six **decided**
redemption forks and a schema addition. These are the acceptance criteria Activities C, D and F
build to — cited so the cards trace to them:

- **F1 — connectivity is an orthogonal region.** The scan flow and a parallel connectivity region
  ({`online`, `probing`, `offline`, `syncing`, **`stale`**}) run concurrently; a connectivity change
  never resets scan progress; `stale` = online but replica not refetched after reconnect (§7).
  → P-KR4, `redemption-submit-flow`.
- **F2 — unverifiable code offline → permissioned override, flagged `unverified_code`.** A scanned
  token not in the local replica while offline is `unknownCode`; without override permission, submit
  is blocked ("connect to redeem"); with it, the confirmation must state that **neither the offer
  nor prior use can be verified**, and the attempt is written `offline_override=true` **and**
  `unverified_code=true`. Adds the `unverified_code boolean` column to `scan_attempts` (§4).
  → `supabase-schema-and-rls`, `redemption-submit-flow`, Q-KR2.
- **F3 — stale local "already used": reject offline; online, server wins.** Offline + local replica
  shows redeemed → reject immediately (`spentLocally`); online → do **not** reject on the local flag,
  submit and let the atomic server check decide. → `camera-scanner-decode`, `redemption-submit-flow`.
- **F4 — after-the-fact double-redeem → domain event + manager notification.** When a synced
  `offline_override` arbitrates to `already_used`, the server emits **`RaceLostReconciled`**; a
  Shift-Manager notification / read-model entry (code, device, staff, time, value) is created for
  follow-up — the counter never slows for it. → `gstate-arbitration-machine`, `reconciliation-view`,
  E-KR3.
- **F5 — auto-apply / "best offer" is CUT.** The app **displays** the customer's offers; staff pick
  and apply the right one in Toast by hand (no Toast terminal integration; the right offer depends on
  what was ordered). Explicitly out of scope (§15). → P-KR2, `camera-scanner-decode`.
- **F6 — accidental re-scan dedupes within session.** Re-scanning the in-session code is a no-op /
  re-shows its terminal result; a different code mid-session prompts to finish the current customer
  first. → `redemption-submit-flow`.

**Modeling approach (§19.1/19.2):** the client scanner is a parallel/orthogonal state machine —
**decided at slate-20260905 (operator, overriding the spike extraction's hand-rolled
recommendation): XState, overlay-region variant, with no-silent-no-ops strictness** — every
(state, event) pair is a declared decision; undeclared pairs throw in dev/test and raise a
modeled, visible `unexpectedEvent` error state in production. Rationale on the record: explicit
modeling "brings out design decisions … for raising edge cases." The server side is the
per-aggregate gstate machines
(§18): the redemption-attempt machine (Activity D), plus the Reward-Code, Campaign and
Issuance/Delivery lifecycles that map onto Activity E and the deferred campaign-admin (#11).

## Open decisions carried into the cycle (handoff §14 — each has a home)

These are not blockers to *starting*; each is pinned to the activity that must answer it. Several
are business calls the operator makes; the spike (Activity 0) gathers the Toast facts.

| # | Decision | Answered in |
|---|---|---|
| 1 | Toast business-date cutoff hour | Activity 0 spike (Toast settings) |
| 2 | Toast order-number format (digits/prefix/reset) | Activity 0 spike (a day of real checks) |
| 3 | Which Toast scheduled report carries order# + business date + discounts | Activity 0 spike (Toast reporting menu) |
| 4 | Toll-free vs 10DLC | **Resolved:** toll-free for the cycle (de-risk leg 1); 10DLC registered in parallel for later scale |
| 5 | `requires_online` face-value threshold | **Resolved (operator, slate-20260904 sitting): make the $ amount configurable** — a settings surface (`marketing_settings.requires_online_threshold_cents`, seeded default $20, changeable without a migration) rather than a hardcoded policy; campaign creation derives `requires_online` from face value vs the setting. The Activity A schema card carries it |
| 6 | Do the 3 devices genuinely go offline independently, or always together on one hotspot | **Activity 0 field observation — still owed.** Operator chose (slate-20260905 sitting) to slate Activity B ahead of it: the spike record prices both topologies (the pull mechanism is what a thin live cache would use too), so the worst case is bounded over-build, not rework. The observation can still re-scope B's *remaining* surface |
| 8 | Welcome-offer definition + per-code expiry window | Operator business call, at Activity A/E |
| 9 | Confirm-then-burn vs burn-on-scan | **Resolved (operator, slate-20260905 sitting): confirm-then-burn, locked** — the code burns only when staff submit with the Toast order number; a mis-scan costs nothing and every redemption carries the join key |
| 10 | QR payload shape | **Resolved (operator, this round):** URL-wrapping the identity token (→ full server-side entitlements when online) **plus an embedded offer descriptor** for offline viewing — hybrid; locked at Activity E |
| 11 | Campaign admin in HQ Go/Postgres vs Supabase directly | Activity A/D (arbiter is Supabase either way) |
| 12 | Who holds `offline_override` | **Resolved (operator, slate-20260905 sitting): a per-user ENTITLEMENT managed in the HQ Users app**, grantable to any role (admin, manager, or team member) — not derived from role. Engineering call, stated: seeded `true` for admins so the branch isn't dead on day one; everyone else by explicit grant/revoke in `users.html` |
| 13 | Reachability signal (heartbeat interval / probe timeout) | Build call at Activity C |

## Module footprints (independent → parallelizable)

| Footprint | Files |
|---|---|
| **supabase arbiter** | Supabase project: `campaigns`/`codes`/`scan_attempts` schema, RLS, `redeem()` RPC, `supabase_realtime` publication (SQL migrations kept in-repo under `supabase/`) |
| **rxdb replica** | new `marketing/` client JS (RxDB collections + pull/push handlers + clock-offset); vendored RxDB reuse per R1 |
| **scanner UI** | `marketing.html`, `index.html` (tile + `TILE_SLUGS`), camera decode + submit flow |
| **redemption backend** | `backend/internal/redemption/**` (gstate machine + HQ redeem endpoint), `backend/internal/db/db.go` (`SeedHQApps`) |
| **delivery** | SignalWire integration (QR gen + MMS send on form submit), consent capture |
| **toast join** | SMTP ingest mailbox → CSV parser → staging table, reconciliation view + orphan-rate |
| **planning docs** | `.night-crew/knowledge/BACKLOG.md`, slate/closeout templates under `reference/` (Activity G) |

---

## Activity 0 — Resolve what the build rides on (unknowns + longest external leads)

> **Why first:** three build-blocking facts and two multi-week external processes must not sit on
> the critical path. Nothing expensive is built on an unanswered #6. **Trace:** Product objective.

### Activity 0 — Resolve what the build rides on (unknowns + longest external leads)

- `redemption-unknowns-spike` · **ABSORBED → Activity H `toast-orders-and-reconciliation`** (retired at the slate sitting of 2026-10-05, ledger T-68 — #1–#3 are answered by the landed reconciliation that joins `OrderDetails.csv` on `(business_date, order_number)`; #6 is moot with Activities B–D built; was:) · Attended / field observation. Answer handoff
  §14 #1 (business-date cutoff hour), #2 (order-number format — digit count, prefix, reset
  behavior), #3 (which Toast scheduled report carries order# + business date + discounts), and
  **#6 — the load-bearing one: do the truck's three devices genuinely lose connectivity
  independently, or are they always together on one hotspot?** Records each answer in the ledger.
  If #6 says "always together," the card flags that Activity B (offline-first replication) is
  likely over-built and proposes the lighter live-shared-server alternative before B is planned in
  detail. `n/a — no code change`. Footprint: planning docs (ledger) + Toast settings read-only.

- `external-accounts-provision` · **PLANNED** · Attended (operator holds accounts/billing).
  Stand up the two external dependencies the whole cycle needs: (a) a **Supabase project**
  (account, project, anon/service keys wired into HQ's existing secret pattern — dev/test project
  distinct from any prod project, per decision 155's spirit); (b) a **SignalWire account +
  toll-free number** for delivery (decision 208, 2026-10-05 — SignalWire replaces Twilio; the
  carrier-side A2P/10DLC and toll-free verification steps are unchanged), and **start A2P/10DLC
  brand+campaign registration in parallel** (1–3 wk external lead — §11; toll-free covers the
  cycle, 10DLC covers later scale). STOP-handling and consent language drafted here so Activity E
  ships compliant (§11, R5). `n/a — no code change`. **Gates Activity A** (Supabase) and
  **Activity E** (SignalWire). Footprint: external accounts + secret pattern.

## Activity A — The attribution spine (the Supabase arbiter)

> **Why here:** every client that reads or burns a code depends on the schema and the atomic
> `redeem()`. The single conditional `UPDATE` is the **only** thing enforcing single use (§6).
> **Trace:** Product + Engineering objectives.

### Activity A — The attribution spine (the Supabase arbiter)

- `supabase-schema-and-rls` · **DONE** (overnight-20260904, branch
  `wo-supabase-schema-and-rls` — migration + seed + verify harness landed in `supabase/`, all
  gates green fresh+warm against the local substrate; triaged 2026-09-04, merged to dev) · The `campaigns` / `codes` / `scan_attempts`
  schema (§4 — including the `unverified_code boolean` flag on `scan_attempts`, F2) with
  `token_hash` never storing the raw token, the `updated_at` index (the
  replication checkpoint key), RLS policies that let each device see only what it needs (§7.2),
  and the tables added to the `supabase_realtime` publication (§7.1 — the usual reason Realtime
  looks broken). Seeds `requires_online` per campaign (operator sets the threshold, #5). SQL kept
  in-repo under `supabase/`. done_when: schema applies clean against a fresh Supabase project and
  a row inserted on one client appears on a second subscriber. Footprint: supabase arbiter.

- `redeem-rpc-race-proof` · **DONE** (overnight-20260904, branch
  `wo-redeem-rpc-race-proof` — atomic `redeem()` with the operator-signed v2 body landed as
  `supabase/migrations/20260904000200_redeem_rpc.sql` + repeatable race harness
  `supabase/verify/04-redeem-race.sh`; 20 rounds × 2 clients, 0 double-wins, red-first against the
  naive analog, `not_found` leg green, GAP-1 `validated:` recorded; triaged 2026-09-04, merged to dev) · The `redeem(p_code, p_device)` plpgsql function
  (§6) — conditional `UPDATE … WHERE redeemed_by IS NULL AND expires_at > now()` returning
  `(ok, reason)` — **and the race test that is the point of this card**: two concurrent clients
  fire at one code; **exactly one** gets `ok=true`, the other gets `already_used`. Red-first: the
  test reds against a naive check-then-update (both win) and greens against the atomic RPC.
  done_when: the race test passes 20× with 0 double-wins. Footprint: supabase arbiter.

## Activity B — Offline-first replica (RxDB ↔ Supabase)

> **Why here:** the scanner must know, with **no network**, which codes are already spent or
> expired (§5) and must show the scanned customer's offers offline. **Offline offer source
> (settled this round — R3 sizing):** replicate **all non-expired offers** to every tablet
> (bounded by `expires_at > now()`, ~2 MB at truck scale — cheap, and the §10 design), with the
> QR's **embedded offer** (D-KR3) as the fallback for a customer not yet in the replica. Planned as
> **new** replication work (R1). **Gated on Activity 0 #6** — if the tablets are never
> independently offline, this activity collapses to a thin live cache and its cards shrink.
> **Trace:** Engineering objective.

### Activity B — Offline-first replica (RxDB ↔ Supabase)

- `rxdb-pull-replica` · **DONE** (overnight-20260905, branch `wo-rxdb-pull-replica` —
  `marketing/sync/` pull modules with the keyset `{updated_at, id}` checkpoint closing GAP-1,
  vendor surface widened (`replicateRxCollection` + `Subject`), standalone substrate harness
  green, wired by Cards 5/6; triaged 2026-09-05, merged to dev) ·
  Two server-owned, **pull-only** replicas (§4) via
  `replicateRxCollection` with an `updated_at` checkpoint: (1) `codes` / redemption-state, filtered
  `expires_at > now() - interval '2 days'` (§5.3), so the scanner knows offline which codes are
  already spent or expired (§5); (2) **non-expired offers**, filtered `expires_at > now()` and
  keyed on customer hash (§10), so a synced customer's **full** offer list resolves offline. Both
  stay bounded by *active* rows, not lifetime customers — ~2 MB at truck scale (R3 sizing). The
  QR's embedded offer (D-KR3) is the **fallback** for a customer not yet in the offers replica.
  (Whether these are one table or two is Activity A's schema call.) Refetch affected rows on every
  `SUBSCRIBED` event, not just on mount (§7.3 — no replay on reconnect). done_when: a code redeemed
  on device A shows spent on device B after a pull tick; a synced customer's full offer list
  renders offline; and an un-synced customer falls back to the embedded offer. Footprint: rxdb
  replica.

- `scan-attempts-push-conflict` · **DONE** (overnight-20260905, branch
  `wo-scan-attempts-push-conflict` — `marketing/sync/push-replication.js` device-owned push
  module: offline queue + `enqueueAttempt` dedupe + redeem-then-land handler with GAP-1's two
  belts (persisted burn outcome before landing; own-device `already_used` = accepted), loser's
  flip rendered from the codes-side pull replica, standalone substrate harness; wired by
  Cards 5/6; triaged 2026-09-05, merged to dev — belt-2 cross-session gap filed B-423,
  harness HTTP-failure injection gap filed B-429) ·
  The device-owned, **push-only**
  `scan_attempts` collection (§4 — opposite replication direction, the key structural decision).
  The push handler batches pending attempts through `redeem()` and writes the outcome back onto
  the local row; the `conflictHandler` flips a losing device's UI from "redeemed ✓" to "already
  used at 6:42pm" (§6). done_when: a lost-race attempt renders "already used" with the winning
  time/device. Footprint: rxdb replica.

- `clock-offset-on-sync` · **DONE** (overnight-20260905, branch
  `wo-clock-offset-on-sync` — `marketing/sync/clock.js` sync clock capturing
  `offset = serverNow − deviceNow` from the pull response's `Date` header on every successful
  pull (spike-proven source), persisted-beside-the-checkpoint state via injected `persist`,
  `clock.isExpired()` as the offline expiry API, window bounds following `clock.now`; both skew
  signs exercised in the standalone substrate harness and re-proven by triage mutation probes;
  wired by Cards 5/6; triaged 2026-09-05, merged to dev) · On every successful sync, store
  `serverNow − deviceNow` and apply that offset in the offline `expires_at` comparison (§5.1) —
  a tablet with a wrong date must not silently accept dead codes. done_when: with the device clock
  set 2 days fast, an expired code is still rejected offline. Footprint: rxdb replica.

- `requires-online-replication` · **DONE** (overnight-20260906, branch
  `wo-requires-online-replication` — campaigns pull replica on the shipped pull mechanism
  (`buildPullUrl` expiry bound now optional, never removed; the codes-embed alternative CLOSED
  by the spike), replica-fed §8 policy (`createCampaignPolicySource` becomes the
  `setCampaignPolicy` default), migration 20260906000100 (supabase_realtime membership +
  touch trigger — a post-flip campaign downgrade re-delivers on the next RESYNC,
  harness-proven), F-2 guard BEFORE `redeem()` via the distinct landing path (migration
  20260906000200: `code_id` nullable + `token_hash` + check constraint; lands
  `accepted`+both flags — no new terminal status per the §9/§19 re-read), branch-3 e2e
  flipped seam-injected → real data, GAP-1 validation run green (f2-run.sh re-executes
  spike 03 against the shipped guard); **merged to `dev` at morning triage 2026-09-05
  (ledger T-54)** — gates independently re-executed, three mutation probes confirmed the
  policy source, the F-2 guard and the touch trigger each load-bearing; but the refusal was
  found to **fail open while the campaigns replica lags** (B-432), so this card does NOT
  unblock close-bar leg 3 on its own. Named at morning triage
  2026-09-05, ledger T-53 — the follow-up card D-1's ratification requires) · Replicate each campaign's
  `requires_online` flag to devices (a campaigns replica, or embed the flag in the codes pull)
  so the §8 refusal ARMS ON REAL DATA — today every campaign resolves policy-unknown and the
  shipped unknown→false default makes the refusal unreachable. Also owns **F-2** (G6-c6,
  latent cross-card): the F2 unknown-code write puts `code_id = token_hash` (64 hex) into a
  `uuid not null` column — give unverified attempts a distinct landing path or a
  skip-until-arbitration guard so Card 3's push handler can't 400/retry-poison the queue when
  provisioning arms sync. 🛑 **REQUIRED BEFORE any real campaign is provisioned**; close-bar
  leg 3 / Q-KR1 cannot be attested until this lands. done_when: a `requires_online=true`
  campaign's code, scanned offline, shows "can't verify — try again" with NO override even for
  an entitlement holder (the branch-3 e2e flips from seam-injected to real-data); an unknown-code
  override lands without poisoning the push queue. Footprint: rxdb replica + supabase arbiter.

- `refusal-holds-before-sync` · **LANDED** (run `20260906-2`, branch
  `card/c1-refusal-holds-before-sync`) · **UNBLOCKS close-bar leg 3 / Q-KR1** (authored at morning triage 2026-09-05, ledger T-54 decision 170 — the
  same "triage authors the unblocking card" precedent as decision 167). `requires-online-replication`
  armed the refusal on real data but left it **failing open during the window before the
  campaigns replica has delivered**: `createCampaignPolicySource` answers `null` for any
  campaign not yet in its Map, `submit-flow.js` coerces `null → false`, and the three replicas
  start together in `scan-page.js` with `SCAN_STATE.synced` display-only — nothing gates
  scanning or submitting on initial replication. So a known, replicated, entitlement-bearing
  high-value code whose campaign has not arrived is offline-overridable. 🛑 **This is NOT the
  ratified unknown→false default (decision 166)** — that covers genuinely-unknown *codes*; this
  is a known code with an unreplicated *campaign*, a case the ratification never considered.
  done_when: with codes + offers replicated and the campaigns replica empty or erroring, a
  `requires_online = true` code scanned offline shows "can't verify — try again" with NO
  override for an entitlement holder — proven by the shipped branch-3 e2e run with the
  `campaigns:` seed removed (the exact triage reproduction), red before, green after; and the
  campaigns-replica failure path is distinguishable from a genuinely-unknown campaign in the
  attempt record. Footprint: rxdb replica + submit-flow policy seam. Candidate shapes (the
  card decides): gate override availability on campaigns-replica readiness, or fail closed for
  codes whose campaign is unresolved, or carry the flag on the code row as the CLOSED
  codes-embed alternative did not.

  **Triage disposition 2026-09-05 (ledger T-55, decisions 173–174): merged, and the card is
  LANDED on its FIRST done_when clause only.** The chosen shape was fail-closed at the policy
  seam (`replicas.js:320-325`) — a KNOWN code whose campaign is absent from the Map refuses,
  readiness aside, which also closes the codes-arrive-first window a bare readiness latch would
  leave open. Clause 1 is **MET and independently re-verified by mutation**, not by reading the
  closeout: reverting only the fail-closed arm reds `tests/marketing.spec.js:877` and `:1043`
  with the reported signature, and the RED commit touched zero production files. 🛑 **Clause 2
  — "the campaigns-replica failure path is distinguishable from a genuinely-unknown campaign in
  the attempt record" — is ruled NOT LANDED.** `scan_attempts.policy_unresolved` shipped, but
  on the shipped tree it can only ever record `false`: `campaignPolicy.attach()` runs only
  inside `startSync`, gated on a localStorage key nothing in the tree writes, so the campaigns
  replica never attaches and every override files as "replica was healthy" on a device where no
  replica has ever run. The e2e asserting the discriminating value stubs both functions with
  literals. Carried as **B-438**; the clause re-opens when the policy source attaches outside
  `startSync`, which is sync provisioning's card. Also carried from this card's own G6:
  **B-439** (`lastError` latched, never re-cleared) and **B-440** (F-2 divert predicate
  disagrees with the constraint rider (a) tightened) — both reproduced by execution at triage;
  **B-441** (the schema-v1 Dexie migration path is executed by no test — UNVERIFIED, not
  passed).

  🛑 **close-bar leg 3 / Q-KR1: this card cleared ONE blocker, not the last one.** ⚠️
  **Corrected 2026-09-06, same attended sitting** — this line first read "UNBLOCKED but NOT
  attested", which overstated it. B-432 is genuinely closed, so the *policy* no longer fails
  open. But Q-KR1 requires a real device holding a `requires_online=true` code **and** its
  campaign, then the reachability probe killed — and the device cannot obtain either, because
  `startSync` is gated on `readJson(SYNC_KEY)` and **nothing in the tree writes
  `hq_marketing_sync_v1`** (`scan-page.js:392`, "Provisioned coordinates — absent tonight").
  With no sync the local collections are empty and every scan resolves `unknownCode`, so the
  high-value-refusal branch is unreachable on real data. The same fact is B-438's mechanism;
  it was filed at triage without being connected to leg 3's attestability. **Operator decided
  2026-09-06: land provisioning first rather than hand-seed the coordinates in devtools** — the
  attestation should be against the shipped path, not a device configured by hand. Discharged
  by `sync-coordinates-provisioning` below.
  **Landed as the second shape, fail-closed at the policy seam** — and NOT gated on replica
  readiness, because a readiness latch alone leaves the "new campaign whose codes arrive
  first" window open; the shipped predicate subsumes it. 🛑 **Read the refusal as a
  predicate, not as unconditional** (the overclaim that hid B-432 for a run): a KNOWN code
  (`campaign_id` non-null) whose campaign is unresolved is refused; a genuinely-unknown code
  keeps its override and its F2 unverified warning, by construction (decision 166). Both
  done_when halves proven: the branch-3 e2e minus its `campaigns:` seed red
  `data-branch="override"` (EXIT=1) → green (EXIT=0), and the attempt record now
  distinguishes a replica-failure override (`unverified_code=t, policy_unresolved=t`) from a
  genuinely-unknown-campaign one (`t, f`), both `status='accepted'` — no new terminal status.
  The owed GAP-1 validation run re-executed spike 02 against the SHIPPED policy source
  (`marketing/sync/harness/refusal-run.sh`, EXIT=0; red probe EXIT=1) and COVERED the
  codes-arrive-first sub-case for real. Riders B-434 (a)(b)(c) all disposed; residual B-436
  (the policy source failing to CONSTRUCT) filed, stated, not closed.

- `sync-coordinates-provisioning` · **LANDED** (run `20260907`, branch
  `card/sync-coordinates-provisioning`) — scan-page.js gained the tree's only `SYNC_KEY`
  writer: page init with a session mints `POST /api/v1/sync/token` and on 200 writes
  `{restUrl: '/sync/rest', bearer, deviceId: sub}` then calls the exported `startSync`;
  401/503/network failure writes nothing (the secret-less-deploy degradation stays, B-436
  untouched). `startSync` also starts `startScanAttemptsReplica` — its first caller — with
  `deviceId` from the mint `sub` only (spike fact 3's stranded-burn cliff), and the B-439
  latch now clears on the successful-pull edge (`onPullSuccess` on the pull-handler seam,
  the `clock.captures` witness) in BOTH recovery shapes while still latching on every error.
  All four done_when clauses green in `tests/marketing.spec.js` ("Sync provisioning"
  describe): clause 1 red-first structurally (SYNC_KEY never written pre-change), transports
  served/killed at the NETWORK layer (build-fact 6's decided call — decision-174-consistent;
  full-stack proof in `marketing/sync/harness/recovery-clear-run.sh` GREEN EXIT=0 + spikes
  01–03), no `setCampaignPolicy` anywhere in the proving tests. B-438 and B-439 discharged
  (BACKLOG dispositions in the same change set). · **GATES close-bar leg 3 / Q-KR1
  attestation** (authored at the attended sitting of 2026-09-06, operator decision "land
  provisioning first"; same "triage authors the card that discharges the finding it raised"
  precedent as decisions 167 and 170) · Wire the two coordinates that arm sync on a real
  device, so the replicas actually start. 🛑 **This is WIRING, not new mechanism** — both ends
  already exist and are landed: the bearer is minted by `POST /api/v1/sync/token`
  (`backend/cmd/server/main.go:633`, `internal/sync/jwtbridge_handler_test.go`), the REST
  surface is `internal/sync/proxy.go`, and `startSync` is already exported
  (`scan-page.js:410`). What is missing is only the caller: `SYNC_KEY`
  (`hq_marketing_sync_v1`) has exactly one occurrence in the tree — its own declaration at
  `scan-page.js:27` — so `readJson(SYNC_KEY)` is always null, `startSync` never runs,
  `campaignPolicy.attach()` never runs, and every local collection stays empty.
  **done_when:**
  (1) **The scanner holds real codes offline.** On a provisioned device with the network then
  cut, scanning a code that exists server-side resolves to that code rather than
  `unknownCode` — proven by an e2e that provisions through the shipped path (never by writing
  `SYNC_KEY` from the test), kills the probe, and asserts `#scan-result` `data-kind`.
  (2) **The campaigns replica genuinely attaches.** `campaignPolicy.attached()` is `true` and
  `size()` is non-zero after initial replication — asserted against the real source, with the
  seam NOT stubbed.
  (3) **B-438 is discharged: `policy_unresolved` records what its DDL comment says.** With
  the campaigns replica made to fail for real (not seam-injected), a genuinely-unknown-code
  override lands `policy_unresolved = true`; with it healthy, the same override lands `false`.
  🛑 **The proving test may not call `setCampaignPolicy`** — today's clause-2 test injects
  `() => true` where the shipped function returns `() => false`, which is why the clause
  passed while being unreachable (ledger T-55 decision 174). If the new test stubs the seam,
  the clause is not met.
  (4) **B-439 is discharged in the same pass** — after a pull failure and a full recovery,
  `unresolved()` returns to `false`; asserted by a test that errors, recovers, then reads it.
  **Footprint:** `marketing/scan-page.js` (the provisioning call site) + `marketing/sync/` +
  whatever mints/refreshes the bearer client-side; backend is expected untouched, and a diff
  that touches `backend/` should be treated as scope drift and stated.
  **Spike gate:** needs its own spike file under
  `.night-crew/knowledge/spikes/activity-b-offline-first-replica/` before it can be slated —
  `/nc-spike` authors it. Likely thin, since the mechanism is threaded and both endpoints are
  landed; the open questions are bearer lifetime/refresh and where provisioning is triggered
  from (page init vs login).

> **Why here:** this is the operator-facing action and the close bar's leg 1. It reads from B
> (offline) and burns through A (online, via D). **Trace:** Product objective. **Locks §14 #9
> (confirm-then-burn), #12 (offline_override holder), #13 (reachability signal).**
> **Opens with a spike (operator's call this round):** decide the client state-machine approach —
> **XState vs a hand-rolled parallel-region machine** — proving the §19 F1/F2/F3/F6 regions in
> HQ's vanilla-JS context before the scanner cards are built. Adopting XState is a new client
> dependency in a deliberately no-framework app; the spike settles it against the real screen.

- `marketing-tile-and-page` · **DONE** (overnight-20260905, branch
  `wo-marketing-tile-and-page` — tile + `TILE_SLUGS` entry, `marketing.html` shell (Scan live,
  three labeled placeholders), `SeedHQApps` seeds `marketing` + the `marketing-offline-override`
  entitlement surface with first-registration-only grants, precache 31→32; triaged 2026-09-05,
  merged to dev) · Add the **Marketing** tile to `index.html`'s grid
  + `TILE_SLUGS`, create `marketing.html` (page shell with the four sub-sections: Scan / Campaigns
  / Subscribers / Redemption stats, §16), seed `('marketing','Marketing','📢')` in `SeedHQApps()`
  so the tile is permission-gated, and grant the `marketing` app to the relevant roles. Enforce
  the create/stats gates inside the handler, not just at the tab (§16 permissions table).
  Regenerate `sw.js` (new precached page — the precache-count invariant will move by 1
  deliberately) and commit it. done_when: a `team_member` sees Scan; a non-granted user sees no
  tile; `build-sw.js` exits 0. Footprint: scanner UI + redemption backend (seed) + `sw.js`.

- `camera-scanner-decode` · **DONE** (overnight-20260905, branch
  `wo-camera-scanner-decode` — scanner screen wired into `#scanner-host`: vendored
  html5-qrcode (single classic script, `lib/`, registry-verified sha256 660b1243…8b1d8e),
  on-device WebCrypto hash, replica-first resolution with embedded-offer fallback + F3
  offline/online branches, precache 32→39; triaged 2026-09-05, merged to dev —
  **live-camera phone check still ATTENDED-OWED**) ·
  Camera via `getUserMedia`, decode with
  `html5-qrcode` (or `@zxing/browser`), **hash the identity token on-device with WebCrypto before
  any lookup** (§12/§4 — a dumped replica never yields live codes), then resolve and **display**
  the customer's offers — the scanner never auto-picks one (**F5** — staff apply the right offer in
  Toast by hand). Resolution order: the **local replica** first (full server-side list once
  synced), then the **offer embedded in the QR** (D-KR3) for a customer not yet replicated; a token
  in neither is `unknownCode` (**F2**, handled at submit). Stale redeemed-state per **F3**: offline
  → reject as `spentLocally`; online → do **not** reject on the local flag, let the server decide.
  Requires HTTPS + a one-time per-device camera grant. done_when: a printed test QR decodes and
  shows its offer offline — synced (replica) and un-synced (embedded); and a locally-redeemed code
  rejects offline but defers to the server online (F3). Footprint: scanner UI.

- `redemption-submit-flow` · **DONE** (overnight-20260905, branch
  `wo-redemption-submit-flow` — strict XState submit machine (vendored xstate 5.32.6,
  registry-verified sha256 e7f04e1f…38fa28), conformance 18/18 + strictness 460 declared
  pairs re-proven at triage; parked overnight on fork D-1, ratified at morning triage
  2026-09-05 (ledger T-53): unknown→false stands as shipped, follow-up card
  `requires-online-replication` REQUIRED before any real campaign; merged to dev) ·
  The heart of the window workflow (§13). Large
  result cards then auto-reset (§16): ✅ Redeemed (offer + entitlement) → **required Toast
  order-number entry that *completes* the redemption** (no path to "redeemed" without it — §13
  double-entry problem), with **format validation** (#2) and business-date computed from the
  **Toast cutoff constant** (#1), not `new Date()`; ⚠️ Already used (when + which device, from the
  `conflictHandler`); ❌ Invalid/Expired (reason). **Submit is online-gated** (§13): the button
  state is driven by a **real reachability signal** — a recent successful sync/heartbeat or a
  short-timeout probe against Supabase, **never `navigator.onLine`** (which lies on a hanging LTE
  link, #13). The screen carries a **persistent, visible online/offline indicator**, and when
  reachability returns after an offline period the submit control **transitions on its own** — no
  manual refresh (P-KR4). **Permissioned offline override** (§13): a user with `offline_override`
  may force-submit while offline behind the confirmation warning, writing an audit-flagged
  attempt — **but only when the campaign's `requires_online = false`**; a `true` campaign shows
  "can't verify — try again" with no override, even for a manager (§8). **Connectivity is a
  parallel region** (F1): a connectivity change never resets scan progress, and a `stale` state
  (online but not refetched after reconnect, §7) routes like offline. **Unknown code offline**
  (F2): a token not in the replica is `unknownCode` — submit blocked without override; with
  override, the confirmation states **neither the offer nor prior use can be verified** and the
  attempt is written `offline_override=true` **and** `unverified_code=true`. **Re-scan dedupe**
  (F6): re-scanning the in-session code is a no-op / re-shows its result; a different code
  mid-session prompts to finish the current customer first. done_when: the three offline branches
  (blocked / override-offered / high-value-refused) each render correctly under a killed
  reachability probe; the indicator flips + submit re-enables **live** when the probe recovers
  (P-KR4); an `unknownCode` offline override writes `unverified_code=true` (F2); and a same-code
  re-scan is a no-op (F6). Footprint: scanner UI.

## Activity D — The server arbitration machine (gstate)

> **Why here:** the online submit (C) and the reconciliation of synced offline overrides need an
> orchestrator; R2 puts it in HQ Go. **Trace:** Engineering objective. The DB stays the arbiter
> (§18 edge-case 1); the machine only reacts to its verdict.

### Activity D — The server arbitration machine (gstate)

- `gstate-arbitration-machine` · **DONE** (overnight-20260905, branch
  `wo-gstate-arbitration-machine` — gstate v0.3.1, go 1.26.2 toolchain bump,
  `POST /api/v1/marketing/redeem`, F4 read-model migration 0077; G6 FAIL→fix `98e189e`→PASS;
  triaged 2026-09-05, merged to dev) · `backend/internal/redemption` — the §18
  statechart (`validating → burning → route_outcome → {redeemed|already_used|expired|failed}`)
  wrapping the atomic `redeem()` via `Invoke` (ctx auto-cancel on state exit, so a hung call on a
  dropped hotspot doesn't wedge — §18 edge-case 4), plus the HQ endpoint the scanner's online
  submit posts to. Must-not-forget edge cases baked into tests: (1) unknown/empty burn result →
  `failed`, **never** a silent `expired` (§18 #3); (2) an `AlreadyUsed` terminal on a synced
  `offline_override` emits a **`RaceLostReconciled`** domain event → a Shift-Manager notification /
  read-model entry (code, device, staff, time, value) for follow-up (**F4**, §8/§9); (3) no
  check-then-act guard that reintroduces TOCTOU (§18 #1). Red-first each. done_when: a two-attempt
  reconciliation emits `RaceLostReconciled` and creates the manager notification (F4). Footprint:
  redemption backend.

## Activity E — Customer delivery (one identity code → QR → image)

> **Why here:** delivery is how codes reach real customers at volume; the loop is provable at
> close with a single test send, and scales once 10DLC clears. **Trace:** Product objective.
> **Compliance is non-optional (R5).** Needs Activity 0's number live. **Locks §14 #10 (URL-wrapped
> QR).**

> **Goal rung added at the slate sitting of 2026-10-05** (run `20261006`): one goal per card, 1:1
> with the spike ledgers under
> `.night-crew/knowledge/spikes/activity-e-customer-delivery-one-identity-code-qr-image/`. The
> signup door is the Fluent Forms import (H5), not an HQ-hosted form; the sending provider is
> SignalWire (decision 208).

### identity-code-and-qr

- `identity-code-and-qr` · **PLANNED** · **One permanent identity code per customer** — the
  QR is primarily *who they are*, not *what they get*; the customer's full, current entitlement
  list lives server-side and replicates down keyed on customer hash (§10). **Hybrid payload
  (operator call, this round):** the QR is a **URL wrapping the identity token** (so the
  customer's own phone can open it, and an *online* scan resolves the complete server-side list)
  **and also embeds the offer current at issue** as a self-describing descriptor, so a scan is
  *readable offline* even before that customer's entitlements have replicated to the tablet — the
  new-signup first-visit case. The embedded copy is a display snapshot; the server list stays
  source of truth, and the embedded offer **never authorizes a redemption by itself** — the burn
  still goes through `redeem()` (§6). **Trust note:** the embedded offer is unauthenticated
  display data, so a forged one could mislead staff offline; that risk is bounded by the §8 policy
  (high-value campaigns require online) and caught in reconciliation as `not_found`/orphan (§9).
  Keep the on-device row minimal — hashed customer id + entitlement list, no names/phone numbers
  on three tablets (§10). done_when: an **offline** scan of a freshly-issued code (customer not
  yet in the local replica) still shows its embedded offer, and an **online** scan of the same
  code shows the full server-side list. Footprint: delivery + supabase arbiter (entitlements).
  **Slated 2026-10-05 (run `20261006`) — spike-fixed mechanism** (goal ledger
  `spikes/activity-e-…/identity-code-and-qr.md`, one signed correction): the scanner half is
  SHIPPED (`parseEmbeddedOffer`, `embeddedOffer` / `offerReady`) and the card is server-side.
  Migration `0088`: `subscribers.identity_token_hash text unique null`,
  `subscribers.identity_minted_at`, and `identity_media (id uuid pk, subscriber_id fk, png bytea,
  created_at)`. Minting (`MintIdentityCode`, once per subscriber, idempotent on the hash): a
  **16-character** base32 token, hashed SHA-256 (the raw token is never stored as text — the
  PNG is the only copy, served publicly at `GET /m/{media_id}.png` for the carrier fetch); the
  hybrid payload `https://hq.yumyums.kitchen/r/<token>#o=<base64url JSON{label, campaign_id,
  expires_at (date-only), face_value}>` — the SAME keys the shipped reader parses — encoded with
  `go-qrcode` at **Low** correction (measured: version 9, 53 modules, decodes through the shipped
  reader; Medium/version 11 does NOT on the 320 px file surface). The embedded offer is the
  subscriber's first-touch campaign (`source_short` → `qr_codes.campaign_id`); with no first-touch
  code the QR is identity-only (no descriptor) — never a guessed campaign. The entitlement row is
  projected to the arbiter's `codes` table through `projection.go`'s existing PostgREST client
  (`ProjectIdentityCode` beside `ProjectCampaign`, merge-duplicates, `expires_at` = the campaign's
  `ends_at`; failure → `warnings:["not_projected"]`, never a rollback — decision 187's shape).
  **One reader line:** `#scan-file-surface` 320 → 640 px (`marketing.html`, `sw.js` regenerated,
  count stays 51) — headroom, not a contract change. done_when (red-first): `[IC-01]` the
  server-minted PNG for a fixture subscriber decodes OFFLINE through the shipped page to
  `embeddedOffer` with the first-touch campaign's label and the hash of the minted token;
  `[IC-02]` the same PNG ONLINE with the lookup answering the projected row → `offerReady` from
  `server`; `[IC-03]` a Medium/version-11 image is the NEGATIVE control (decodeError or no offer —
  never a wrong offer); `TestMintIdentityCodeIsIdempotent`, `TestMintIdentityCodeNoFirstTouchIsIdentityOnly`,
  `TestProjectIdentityCodeUpserts` (mocked PostgREST, asserting the body and `Prefer` header);
  `GET /subscribers/{id}` carries `identity_code.status = minted|sent` and `media_url`.
  Footprint: `backend/internal/marketing/identity.go` (new), `projection.go`, `subscribers.go`,
  `routes.go` (`/m/{id}.png` under `MountPublic`), migration `0088`, `marketing.html` (one
  line), `tests/marketing-identity.spec.js` (new, joins the `marketing` seam by name), `sw.js`.

### mms-send-on-signup

- `mms-send-on-signup` · **PLANNED** · Form submit → generate the identity code in Supabase →
  **send the QR as an MMS image, not a link** (§11 — an image lives in the thread and opens with
  no signal; a link needs signal at the window). Explicit **consent capture at point of
  collection**, working **STOP handling**, sends over the registered number (§11 compliance,
  R5). done_when: the operator's test signup produces a scannable image in their Messages thread.
  Footprint: delivery. **External dependency: a live sending number (Activity 0).**
  **Slated 2026-10-05 (run `20261006`) — spike-fixed mechanism** (goal ledger
  `spikes/activity-e-…/mms-send-on-signup.md`, no corrections; provider SignalWire, decision
  208): the signup door is the Fluent Forms import (H5), so "form submit" = a NEW subscriber
  landing through `ImportSubscribers`. New package **`internal/delivery`** (the path carries no
  vendor name — the egress guard matches import paths by substring): `Sender` interface
  (`SendMMS(ctx, to, mediaURL, body) (sid, error)`) + `SignalWire` over the Twilio-compatible LaML
  Messages resource (`POST {space}/api/laml/2010-04-01/Accounts/{project}/Messages.json`, basic
  auth, form `From/To/Body/MediaUrl`), configured from `HQ_SW_SPACE_URL`, `HQ_SW_PROJECT_ID`,
  `HQ_SW_API_TOKEN`, `HQ_SW_FROM`; unset → a **recording stub** (`delivery.Stub`, rows in
  `delivery_log`) so every gate and the e2e stack send nothing real — D-KR2's mocked transport.
  `internal/marketing` imports ONLY the interface through `Deps.Sender` (spike-proven legal).
  On a new subscriber with `sms_consent = true` and a phone: mint (E1) → `SendMMS(phone,
  media_url, "Your Yumyums code — show this at the window")` → append `code_sent` (ref: sid,
  provider) — the event the Stats funnel counts; `sms_consent = false` or no phone → E1 still
  mints (identity is who they are — one code ever) and **only the send is withheld**,
  `code_withheld: no_consent | no_phone` noted on the `signed_up` event's ref (no new `kind` —
  the CHECK constraint is the lifecycle; adding a kind is a park). `POST /subscribers/{id}/resend` becomes
  a real send of the SAME media (one code ever, §10) and still refuses without consent (202
  `sent:false, reason:"no_consent"`). **STOP:** `POST /api/v1/marketing/sms/inbound` under
  `MountPublic`, form-encoded as the carrier posts it, validated with the SignalWire signing key
  when `HQ_SW_SIGNING_KEY` is set (403 otherwise, 503 when unset in prod mode; the stub mode
  accepts unsigned), `Body` ∈ {STOP, STOPALL, UNSUBSCRIBE, CANCEL, END, QUIT} (case-insensitive)
  → `opted_out_at = now()`, `sms_consent = false`, `opted_out` event; the next send for that
  phone is refused. Consent evidence on the subscriber is untouched. done_when (red-first, the
  spike's five baseline lines): `TestNewConsentingSubscriberIsMintedAndSent` (stub sender called
  once, `code_sent` 1, `identity_code.status = sent`); `TestNoConsentIsNeverSent` (fixture
  `+17735550117`: sender never called, 0 `code_sent`, timeline ref says why);
  `TestInboundStopOptsOutAndBlocksNextSend`; `TestInboundRejectsBadSignature`;
  `TestResendSendsSameMediaOnce`; `TestNothingInThisPackageSends` STILL GREEN (the sender is
  outside); Playwright `[MS-01]`–`[MS-03]` in `tests/marketing-delivery.spec.js` (D-KR2 is
  measured by `tests/`): import the fixture through the UI → the subscriber sheet shows **Sent**
  for `+17735559930` and **Not sent — no SMS consent** for `+17735550117`, and after a stubbed
  STOP the sheet shows **Opted out** and Resend answers "no consent". `marketing/subscribers.js`
  copy only; `sw.js` regenerated, count 51. **Attended, never the night:** the live send
  (D-KR1), the SignalWire account, the number, the signing key.
  Footprint: `backend/internal/delivery/` (new), `backend/internal/marketing/subscribers.go`,
  `routes.go`, `helpers_test.go` (stub sender in `testDeps`), migration `0088` (shares E1's —
  `delivery_log`), `marketing/subscribers.js`, `tests/marketing-delivery.spec.js` (new), `sw.js`,
  `backend/cmd/server/main.go` (wire `Deps.Sender` from env).

## Activity F — The join lands (SMTP ingest + reconciliation)

> **Why here:** needs codes actually redeemed to have something to reconcile; it is the close
> bar's leg 2. Everything is **T+1** by design (§13). **Trace:** Product + QA objectives.

### Activity F — The join lands (SMTP ingest + reconciliation)

- `smtp-toast-ingest` · **SUPERSEDED** (T-60, decision 188 — Activity H `toast-orders-and-reconciliation` reads `OrderDetails.csv` off the Toast SFTP export HQ already syncs; this card returns to PLANNED only if H3 finds the export lacks the file) · A dedicated ingest mailbox receives the scheduled Toast
  report (#3); an inbox watcher extracts the CSV, normalizes, and loads a staging table. **Key on
  `(business_date, order_number)` and upsert — never blind-insert** (§13 idempotency; the same
  report *will* arrive twice). Dedicated mailbox, restricted access, no forwarding (§13 security).
  done_when: ingesting the same report twice leaves one row per order. Footprint: toast join.

- `reconciliation-view` · **ABSORBED → Activity H** (`toast-orders-and-reconciliation` + `stats-tab-ui`, T-60) · The three-bucket view built from day one (§13): `matched`
  (scan joined to a Toast order), `unmatched` (order number with no Toast match → fuzzy-match on
  `scanned_at` timestamp), `orphan` (accepted with no order number — lost attribution). The
  **orphan rate is the health metric** — surfaced in the Marketing → Redemption-stats section; if
  it climbs above ~10% the window workflow needs fixing, not the code (§13). Offline overrides —
  including `unverified_code` ones (F2) — are flagged and reconciled first, and a lost race surfaces
  the **F4** Shift-Manager notification (code / device / staff / time / value). done_when: the
  close-bar test redemption shows `matched`, the orphan rate renders, and a reconciled lost race
  produces the manager notification. Footprint: toast join + scanner UI (stats section).

## Activity G — Planning surface honest (carried QA debt)

> **Why here:** overnight-parallel, disjoint footprint. These two cards produced the **only two
> NOT-MET KRs** of last cycle (Q-KR2, Q-KR3); they were promoted at T-46 and again at T-48 but
> never slated, so they reddened the close through no fault of their own. Carried, not re-derived.
> **Trace:** QA objective.

### Activity G — Planning surface honest (carried QA debt)

- `backlog-machine-migration` · **DONE** (overnight-20260904, branch
  `wo-backlog-machine-migration` — all 297 issues retired: `backlog check --repo .` exit 0
  `valid — 209 entries`, list count == the checker's own parse, whole-document token-multiset
  containment proven 0-lost against the red-baseline commit, handles B-350..B-414 assigned,
  triage §4.5 gate armed in COMMANDS.md; triaged 2026-09-04, merged to dev) · (Carried from last cycle's Activity 5.) Closes
  **B-02**, **B-168**, **B-12**, **B-133**. Reshape the ~193 legacy-shape entries to the canonical
  `B-NN` form until `night-crew backlog check` exits 0, with **content preservation proven**
  (stripped-text diff: every entry body present before is present after; handles assigned above
  the current max — collisions have happened, B-39→B-44). Then **arm the triage §4.5 gate** so the
  document cannot drift back. done_when is mechanical: `check` exit 0, and `backlog list` count ==
  document entry count. Footprint: planning docs.

- `team-records-from-hand-runs` · **DONE** (overnight-20260904, branch
  `wo-team-records-from-hand-runs` — template landed at
  `.night-crew/knowledge/scorecard/TEMPLATE.md` (.md by design — inert to the union read,
  proven) + closeout ritual stanza armed under COMMANDS.md step 5; validation render green:
  transient fake-run-id record → all four roles record-backed, EXIT=0, record deleted, no
  real-run-id jsonl committed — tonight's `20260904.jsonl` is emitted by the run's closeout
  per the new stanza; triaged 2026-09-04, merged to dev) · (Carried from last cycle's Activity 5.) The
  scorecard sees no rostered role on this hand-run target — every close renders `—` for all four
  teams. Scope: emit the per-run scorecard files the CLI already reads, from this repo's hand-run
  slate/closeout ritual (template + ritual step). If that provably requires CLI changes, the card
  records the finding, files it clone-side, and closes with the target-side half done. Footprint:
  planning docs.


## Activity H — Campaign admin, subscribers, stats (the designed tabs)

> **Why here:** the arbiter (A), replica (B), scanner (C) and server machine (D) are landed; what a
> manager can *do* with them is still three "Soon" cards. This activity turns the operator's
> 2026-10-01 design sitting into product. **Design of record:** Claude Design project *Yumyums HQ
> Marketing* (the three **Current** pages) — https://claude.ai/design/p/a8ffc065-b005-4020-bc6a-f42dc7e8f0e3 .
> **Spec:** `docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md` (B-458; decisions
> 187–191 at T-60). **Trace:** Product objective — P-KR3 (orphan rate visible, loop joinable) and
> Q-KR2 (overrides auditable, reconciled first) are graded against `marketing.html` and have no
> surface until this lands. Absorbs Activity F's `reconciliation-view`, supersedes
> `smtp-toast-ingest` (decision 188), and folds in B-424, B-436, B-440, B-446, B-447.
> **Decision 192 (operator, slate sitting 2026-10-01):** the Stats REPORTS live on the BI hub behind
> the `bi` grant; the reconciliation QUEUE stays as Marketing's fourth tab. See H4 / H3b / H1.
> **Dispatch (as slated 2026-10-01, slate-20261002):** H1 alone first (Wave 0 — it creates the
> `backend/internal/marketing/` package, migration 0083 and the `qr_codes(short)` key that 0085
> references), then three concurrent tracks — B: H2 → H4 · C: H3a → H3b → H6 · E: H5 — with H4
> starting only once H3b has landed (it switches from fixtures to the real stats endpoints before
> its final gate). **H3 was split at the slate sitting** (split-before-slating rule): `toast-orders-and-mirror`
> (H3a — ingest + mirror + migration 0084 + B-424) and `reconciliation-and-stats-engine` (H3b — the
> engine and its endpoints). Both are children of the `toast-orders-and-reconciliation` goal ledger.

> **TRIAGED 2026-10-02 (ledger T-62) — Activity H is COMPLETE: 7 of 7 cards landed and merged
> to `dev` at `5753ddf`.** Every gate number independently reproduced by an adversarial reviewer
> (G2 `EXIT_TEST=0`, 699/0/3 across exactly 15 test-bearing packages; G4 51 precached with the
> committed `sw.js` byte-identical to a fresh regeneration). Six forks settled as decisions
> 194–200; ratified decision 191 amended.
>
> 🛑 **Two key results from this activity are NOT MEASURABLE this cycle, by decision rather than
> by defect: Q-KR2 and the per-slice half of P-KR3.** Campaign attribution cannot resolve on live
> data — the mirror writes the campaign tag as a literal `NULL` and omits it from its
> `ON CONFLICT` update list, so nothing in the tree can populate either arm of the lookup
> (decision 197). The totals, funnel, revenue, discount and orphan rate are all correct and
> measurable; only the per-campaign / per-channel / per-dish attribution is held.
>
> **Carried to the next roadmap round as a named architectural question:** *should Supabase be
> the single source of truth for the attribution spine?* Decision 187 split ownership (campaign
> admin in HQ, codes and scans in Supabase) and was written "revisable at the first morning
> triage that finds them wrong"; triage found the two-source split, not a missing column, to be
> what makes attribution unresolvable. Evidence attached in ledger T-62 decision 197.
>
> **Next-slate candidates out of this activity, in the operator's stated order:**
> 1. **The test-integrity fix card** — the operator's explicit condition for merging: B-462
>    (`campaigns-harness.mjs` leg 3 green against inverted production code), B-465 (`[SP-03b]`,
>    a second false gate, leaving decision 191's distinguishing branch guarded by nothing), B-466
>    (`TestNothingInThisPackageSends` scans two filenames, so egress elsewhere in the package
>    passes). Carries the three decision-191 amendment riders.
> 2. **B-468** — an online phone never verifies a code it has not synced, so the discount is
>    keyed into Toast before anything asks whether the code exists. Promoted by the operator
>    above the backlog (decision 200).
> 3. **Decision 194's two halves** — teach the dish merge about `campaigns_admin` / `qr_codes`,
>    and add the blank-on-delete backstop on those plus migration `0085`'s subscriber tables,
>    which is what makes a right-to-erasure request servable by a plain `DELETE`.
> 4. **Decision 195** — the atomic scan dedupe (tumbling bucket + unique index + do-nothing on
>    conflict), since the orphan-rate denominator is inflatable from an unauthenticated route.
>
> **Attended work no overnight can close:** the first live Fluent Forms import; the skipped Share
> spike (`campaigns-tab-ui / web-share-files-enumerated`, webkit/firefox binaries); and
> decision 196's check — read one real Toast export's `Opened` column against a known order time.

### Activity H — Campaign admin, subscribers, stats (the designed tabs)

- `campaign-codes-api` · **LANDED** (run `20261002`, branch `card/h1-campaign-codes-api`) · (H1, track A — backend) Campaign admin in HQ Go +
  Postgres (decision 187): migration `0083_campaigns_admin` (`campaigns_admin`, `qr_codes`,
  `qr_scans`), `POST /api/v1/marketing/campaigns` mints **one `qr_codes` row per channel** in
  one transaction, projects the four tablet columns to Supabase `campaigns` over PostgREST
  (service key; `projected_at NULL` + `warnings:["not_projected"]` when unconfigured — fail
  loud, never silent), `PATCH` campaign/code (re-point without reprint), `GET /codes/{id}.png`
  (`skip2/go-qrcode`), and the **public** `GET /q/{short}` landing that logs a scan and 302s
  with UTM (decision 189; inactive → "offer has ended" page). Manager tier enforced in the
  handler (§16): `team_member` → `403 managers_only`. done_when: `TestCreateCampaignMintsOneCodePerChannel`,
  `TestLandingLogsScanAndRedirectsWithUTM`, `TestLandingInactiveCodeRendersEndedPage`,
  `TestProjectionUnconfiguredLeavesProjectedAtNull`, `TestTeamMemberGets403ManagersOnly` red on
  the pre-change tree, green after; Go suite counts checked. Footprint: `backend/internal/marketing/`
  (new), migration 0083, `backend/cmd/server/main.go` (undeclared seam → full Playwright suite),
  `go.mod`. Also lands the no-op `marketing.MountReports(r)` seam inside the BI
  route block (`RequirePermission(pool,"bi")`) so H3b's report reads mount there without touching
  `main.go` again (decision 192).
  **Landed as slated**, with four things stated rather than assumed: (1) `main.go` has THREE call
  sites, not two — `Mount`, `MountReports` and `MountPublic`, because the public `/q/{short}`
  cannot live inside either gated block; (2) `MountReports` sits in a `bi` `r.Group` of its own,
  not inside the `/inventory` Route, whose prefix would have made the contract
  `/api/v1/inventory/bi/*`; (3) `marketing_settings` is a **Supabase** table with no HQ copy, so
  the #5 threshold is read over PostgREST when the projection is configured and falls back to the
  substrate's own seeded 2000 cents at WARN — a dead substrate never blocks a save; (4) the
  projection writes **four** columns (`id, name, face_value, requires_online`), which is what
  Supabase `public.campaigns` has — decision 187's "`expires_at`-equivalent" has no column to land
  in, expiry living on `codes`. `funnel.signups` / `funnel.redeemed` and the whole `money` block
  ship as the stated zero shapes H5 / H3a / H3b fill. The CONFIGURED projection path is proven
  against the local `spike-supabase` substrate from Go, not stubbed
  (`TestProjectionConfiguredUpsertsToSubstrate`).

- `campaigns-tab-ui` · **LANDED** · (H2, track B — UI) The Campaigns section of
  `marketing.html` per Current Campaigns 1–8: list as funnel cards with the money strip
  (revenue / discount / net / Per $1 pill), the one-sheet create (channels as chips, Value and
  Item visible, payload preview, "Create campaign + N codes"), the "N codes ready" screen,
  detail with the Money card and code rows, the code sheet (big QR, Share → `navigator.share`
  with the PNG file, Save PNG / Copy link / Print, Re-point, Pause), and the empty / locked /
  offline / not-projected states. Builds against §5's JSON shapes on a fixture server until H1
  merges, then switches (merge-intent states which). done_when: `[MC-01]`–`[MC-05]` red → green;
  `tests/states-marketing-campaigns.spec.js` screenshots every State Enumeration Table row and
  the PNGs are read back; `sw.js` regenerated + committed, precache count stated. Footprint:
  `marketing.html`, `marketing/campaigns.js` (new), `tests/marketing-campaigns.spec.js` (new),
  `tests/states-marketing-campaigns.spec.js` (new), `sw.js`, `night-crew.toml` (+seam rows).
  **Landed as slated**, built against H1's REAL endpoints rather than a fixture server (H1 had
  merged when this started), with four things stated rather than assumed: (1) the State
  Enumeration rows split **5 real / 3 fixture** — success, locked, offline, not-projected and
  long-content hit the real endpoints, while empty / loading / error ride `page.route` because a
  shared e2e database and a real server cannot produce "no campaigns", "slow" or "500" on demand;
  (2) `not projected` needed **no** fixture at all — `HQ_SYNC_REST_URL` is unset on the test
  stack, so the handler genuinely returns `projected_at: null` + `warnings:["not_projected"]`;
  (3) Share is feature-detected per PAYLOAD at sheet-open time with a real `image/png` File, and
  when the probe says no there is **no Share button** — Save PNG becomes primary — which is the
  branch `[MC-03b]` asserts un-stubbed beside `[MC-03a]`'s installed-API observation; (4) the
  create sheet's **Item** field degrades to "Any item · needs Inventory access" for a manager
  without the `inventory` grant, because the dish catalog read is the Inventory app's — noted as
  a deviation rather than solved by a backend route in another card's footprint. `night-crew.toml`
  took a **roll-call comment only** (no new key, no new token): the existing `marketing` token
  now selects 3 specs, not 1. Precache **48 → 49** (`marketing/campaigns.js`; no `globPatterns`
  change — `marketing/*.js` already matched it).

- `toast-orders-and-mirror` · **LANDED** (run `20261002`, branch `card/h3a-toast-orders-and-mirror`) · (H3a, track C — backend; the first half of the
  `toast-orders-and-reconciliation` goal, split at the slate sitting 2026-10-01) Supersedes
  `smtp-toast-ingest`, closes **B-424**. The Toast SFTP sync fetches `OrderDetails.csv` beside
  `ItemSelectionDetails.csv` and upserts `toast_orders` keyed `(business_date, order_number)` —
  the same report WILL arrive twice (§13); **the export was listed read-only at the spike
  (2026-10-01): `OrderDetails.csv` is present on every recent date dir** — no fallback needed;
  `Order #` is digits only, 1–4 long, stored as text. A 5-minute keyset poller mirrors Supabase
  `scan_attempts` into HQ (`scan_attempts_mirror`; service key; off under
  `E2E_DISABLE_SCHEDULERS=1`). Migration `0084_toast_orders_reconciliation` lands ALL THREE
  tables (`toast_orders`, `scan_attempts_mirror`, `reconciliation_decisions`) so H3b adds no
  migration. B-424: unique index on `race_lost_notifications (code_id, losing_device, scanned_at)`
  (store insert becomes `ON CONFLICT DO NOTHING`) and the F4 status bullet owned here.
  done_when: `TestOrderDetailsUpsertIsIdempotent`, `TestScanAttemptsMirrorKeysetResumes`,
  `TestRaceLostNotificationDedupe` red → green; Go suite counts checked. **Landed as planned, with
  four stated engineer-level calls** (full text in the card's merge-intent): `scan_attempts_mirror.code_id`
  ships NULLABLE and gains `token_hash`, because upstream dropped that NOT NULL for F2 and a
  verbatim §4 NOT NULL would have silently refused exactly the unverified-code attempts F4 cares
  about most; `campaign_id` mirrors as NULL (upstream has no such column and no FK to embed
  through); B-424's index is on 0077's real column names `(code_token_hash, device_id, scanned_at)`;
  and the keyset's one real gap — a late-arriving offline attempt whose `scanned_at` predates the
  cursor is never mirrored — is named in `mirror.go` with its fix rather than left to be
  rediscovered. `TestScanAttemptsMirrorKeysetResumes` ran against the **LIVE** `spike-supabase`
  substrate, not a fixture. Footprint:
  `backend/internal/toast/` (+`orderdetails.go`, `sync.go`), `backend/internal/marketing/mirror.go`,
  migration 0084, `backend/internal/redemption/store.go`, `backend/cmd/server/main.go` (poller
  start beside the Toast worker → undeclared seam → full Playwright suite).

- `reconciliation-and-stats-engine` · **LANDED** (run `20261002`, branch `card/h3b-reconciliation-and-stats-engine`, merge `65ef5dc`) · (H3b, track C, serial after H3a — backend;
  the second half of the `toast-orders-and-reconciliation` goal) Absorbs `reconciliation-view`.
  The engine buckets attempts matched / unmatched (±30 min nearest-order suggestion) / orphan,
  orders the queue overrides → orphans → unmatched, and records `reconciliation_decisions`
  (match / decline-with-reason+note / reopen / verify / reject). Money per campaign·channel·item
  with the **per-row discount rule** (decision 190 as refined: actual where matched, face value
  where not; `discount_basis` is a label) and `per_dollar`; orphan rate counts declines except
  `duplicate_scan`. Ships the §5 endpoints `GET /stats/overview`, `GET /stats/by`,
  `GET /reconciliation/queue`, `POST /reconciliation/{id}/match|decline|reopen|verify|reject`,
  `GET /reconciliation/declined`, and the `money` block on `GET /campaigns`. The report reads
  (`overview`, `by`) are registered twice: under `/marketing/stats/*` (marketing grant + manager
  tier) and under `/bi/campaigns/*` through H1's `MountReports` seam (`bi` grant) — decision 192.
  done_when:
  `TestQueueOrdersOverridesThenOrphansThenUnmatched`, `TestDeclineOtherRequiresNote`,
  `TestOrphanRateCountsDeclinesExceptDuplicateScan`, `TestSlicesReconcileToOverview` (the
  spike `stats-tab-ui/01-slices-reconcile` arithmetic, as a Go test) red → green. Footprint:
  `backend/internal/marketing/` (`reconciliation.go`, `stats.go`, `routes.go`), no migration.

- `stats-tab-ui` · **LANDED** (run `20261002`, branch `card/h4-stats-tab-ui`, merge `15ebc00`) · (H4, track B after H2, starts once H3b has landed — UI, two
  pages per **decision 192**) **Reports in BI:** a third `hub-row` "Campaigns" on `bi.html`
  (behind the `bi` grant, idiom of Trends / Food cost) rendering Current Stats 1–5 — overview
  (period, funnel with revenue / discount / net lines, "implied −$X · actual −$Y" where they
  differ, reconciliation health card with the 10% orphan line, slice links), by campaign / by
  channel / by item with the Funnel ⇄ Money toggle and drill-ins — from `GET /api/v1/bi/campaigns/*`.
  **Queue in Marketing:** the fourth tab of `marketing.html` becomes the reconciliation queue
  (Current Stats 6–8): "N redemptions need a look", overrides → orphans → unmatched with the fix
  **and** "Can't match…" on every row, the add-order-number sheet with nearest-order chips, the
  decline sheet (reason chips, note required for Other, plain statement of what declining does),
  the declined bucket with Reopen, and its own health card with the orphan rate (P-KR3 / Q-KR2
  read here). One module `marketing/stats.js` loaded by both pages. Fixture-first like H2 for the
  forced state rows only. done_when: `[MS-01]` BI overview renders funnel + revenue/discount/net +
  orphan rate with the 10% marker; `[MS-02]` by-item rows carry discount and Per $1; `[MS-03]`
  decline sheet requires a reason and posts reason+note; `[MS-04]` declined bucket shows note and
  Reopen; `[MS-05]` empty period renders "No redemptions yet" on BOTH pages; `[MS-06]` a user with
  `bi` but not `marketing` sees the BI row and gets the Locked state on the Marketing queue — red →
  green; `tests/states-bi-hub.spec.js` updated from 2 rows to 3; states spec screenshots read
  back. Footprint: `bi.html`, `marketing.html` (#s4), `marketing/stats.js` (new),
  `tests/marketing-stats.spec.js`, `tests/states-marketing-stats.spec.js`,
  `tests/states-bi-hub.spec.js`, `sw.js`, `night-crew.toml`.

- `subscribers-tab` · **LANDED** (run `20261002`, branch `card/h5-subscribers-tab`, merge `78e768f`) · (H5, track E — full-stack) Migration `0085_subscribers`
  (`subscribers`, `subscriber_events`), three source adapters behind one interface — Fluent
  Forms reader (`FF_DB_*` env; maps the live form's REAL keys per spike 2026-10-01 — `names.first_name`,
  `email`, `input_text` → phone, `checkbox[]` → consent, `source` nullable; fixture committed; live import is an
  **attended** first run), Toast guest CSV upload, QR signup join on `source_short` (first-touch
  attribution) — `GET /subscribers` (masked phone, consent state, visits), `GET /subscribers/{id}`
  (identity-code status, consent trail, timeline), `POST …/resend` recording a
  `resend_requested` event and **sending nothing** (Activity E owns the send). UI per Current
  Subscribers 1–2. done_when: `TestFluentFormsImportIsIdempotent`,
  `TestSubscriberSourceShortSetsCampaignAttribution`, `[SB-01]`–`[SB-04]` red → green. Footprint:
  `backend/internal/marketing/subscribers.go` + `sources/`, migration 0085, `marketing.html`,
  `marketing/subscribers.js` (new), `tests/marketing-subscribers.spec.js`,
  `tests/states-marketing-subscribers.spec.js`, `sw.js`, `night-crew.toml`.

- `scanner-polish` · **LANDED** (run `20261002`, branch `card/h6-scanner-polish`, merge `23be14a`) · (H6, serial after H3b — touches `marketing/sync/*`) Closes
  **B-446** (render the `requires_online` refusal at scan-resolve while offline, same copy,
  earlier; post-submit guard stays), **B-447** (add `name` to the campaigns pull selection;
  offer card shows the campaign name and the code's last four), **B-440** (divert predicate
  `unverified_code && offline_override`, poison-row case in `f2-run.sh`), **B-436 fail-closed**
  (decision 191: no policy source → no offline override; `campaigns-harness.mjs` leg 3's negative
  assertion flips). done_when: `[SP-01]`–`[SP-03]` + the harness legs red → green, exit codes
  graded. Footprint: `marketing/sync/replicas.js`, `marketing/sync/push-replication.js`,
  `marketing/submit-flow.js`, `marketing/scan-page.js`, `marketing/sync/harness/*`,
  `tests/marketing.spec.js`, `sw.js`.

## Activity I — Honest gates, then verify before the till (triage 20261002 follow-ups)

> **Why here:** Activity H landed seven cards and its morning triage (ledger **T-62**, decisions
> 194–200) found that two of the gates guarding the scanner's offline policy are false, that one
> guard behind a legal commitment ("nothing sends") is narrower than its name, that an online phone
> commits a discount before anything has verified the code, and two schema gaps the night parked.
> The operator ordered these at triage — the test-integrity fix "lands before any new product
> work", B-468 "promoted above the backlog" (decision 200) — and the slate sitting of 2026-10-02
> authored them as cards (the "triage authors the card that discharges the finding it raised"
> precedent, decisions 167 / 170 / 180). **Trace:** QA objective (Q-KR1's refusal is what the false
> gates claim to guard; P-KR3's orphan-rate denominator is what the dedupe protects) and the
> Product objective (B-468 is the window workflow). Four cards, two tracks: **A (client)** I1 → I2,
> **B (backend)** I3 → I4. Spike ledgers under
> `.night-crew/knowledge/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/`.

### test-integrity-fix

- `test-integrity-fix` · **LANDED** (run `20261003`, branch `card/i1-test-integrity-fix`, merge `8a7065d`) · (I1, track A — the operator's stated condition for
  merging run 20261002) Three gates pass against inverted or widened production code, and all
  three are fixed so they red on the mutation that fooled them. **B-462** — `campaigns-harness.mjs`
  leg 3 reimplements `failClosed` instead of importing `marketing/submit-flow.js`'s; it must
  import the shipped predicate (export it; the harness asserts the IMPORT, since its comment's
  "negative assertion moves with it" was proven never true). **B-465** — `[SP-03b]` passes
  against an inverted `failClosed` because `kind='unknownCode'` never consults the predicate; the
  branch it claims to guard exists only because `MARKETING_REPLICA_SCHEMA` omits `campaign_id`
  from `required`, a row both databases forbid (`not null references`). Add `campaign_id` to
  `required` **as schema version 1 with a migration strategy** (the three-part shape
  `replicas.js` documents for the campaigns v1 schema — a bare edit bricks a synced phone), and
  retire `[SP-03b]` in favour of a test that the invariant is a schema fact. **B-466** —
  `TestNothingInThisPackageSends` scans two filenames; make it walk every non-test `.go` file in
  `internal/marketing` and `sources/`, with an explicit allowlist naming the files permitted to
  speak HTTP and why (`projection.go` — decision 187's projection; `mirror.go` — the poller), so
  an `http.Post` added anywhere else reds it. Carries the three decision-191 amendment riders
  (T-62). 🛑 The refusal behaviour itself does not change: the B-432 fail-closed predicate, the
  `requires_online = true` refusal and decisions 166/199 are untouchable — this card changes what
  the TESTS measure, not what the phone does. done_when (each mutation-proven, red-first):
  with `failClosed` inverted, `campaigns-run.sh` exits 1 (today 0) and `[SP-03]` reds while no
  spec in the file stays green by never consulting the predicate; with an `http.Post` in
  `campaigns.go`, `TestNothingInThisPackageSends` reds (today green); a codes-replica store
  created at v0 reopens at v1 with its rows intact (the B-441 lead, executed). Footprint:
  `marketing/sync/harness/campaigns-harness.mjs`, `marketing/submit-flow.js` (exports only),
  `marketing/sync/replicas.js` (schema v1 + strategy), `tests/marketing.spec.js`,
  `backend/internal/marketing/subscribers_test.go`, `sw.js` (regenerated, count unchanged),
  BACKLOG dispositions B-462 / B-465 / B-466.

### scan-time-verify

- `scan-time-verify` · **LANDED** (run `20261003`, branch `card/i2-scan-time-verify`, merge `3986161`) · (I2, track A after I1 — **B-468**, decision 200)
  Today `marketing/scanner.js` `resolve()` consults three LOCAL sources only and contains no
  network call, so a code minted since the phone last synced resolves `unknownCode` even on a
  fully-online phone, the crew keys the discount into Toast, and only the submit asks the server.
  Behaviour after this card: **an online phone that has never seen a code asks the server about it
  at scan time, before any discount is applied** — the real offer renders (or "already used"), and
  only if the server cannot be reached inside the probe budget does the phone fall back to today's
  "this code isn't on this device" path, saying it could not check. **Offline behaviour is
  unchanged** (decisions 166 / 199 govern the never-seen code offline). Mechanism (decision 200's
  direction, spike-proven): one `GET /codes?token_hash=eq.<hash>&select=id,campaign_id,expires_at,redeemed_at,redeemed_by`
  through the sync door the device already uses (`restUrl` from the provisioned coordinates; the
  proxy mints the session's device JWT; RLS lets the device role read any code row — redeemed
  and expired included, spike 01), with a timeout no longer than the connectivity probe's, wired
  as an optional `serverLookup` dep on `createScanResolver` and called only when `online` and
  the token is in neither replica. A server row that is live → `offerReady` with
  `source: 'server'`; redeemed → the "already used" result (the existing F3 online handling
  decides at submit, as today); no row → `unknownCode`; timeout/error → `unknownCode` with
  `verified: false` so the copy says the server could not be checked. Preference candidate
  `process/C-4` ("put the check before the step that cannot be undone") is the operator's
  stated reason. done_when: `[SV-01]` online + code absent locally + server has it → the offer
  card renders from the server row before any submit control; `[SV-02]` online + server says
  redeemed → "already used" at scan, no discount prompt; `[SV-03]` server killed at the network
  layer → today's copy plus "couldn't check the server", inside the budget; `[SV-04]` offline →
  zero network calls (route-interception count 0) and behaviour byte-identical to today — red →
  green in `tests/marketing.spec.js`. Footprint: `marketing/scanner.js`, `marketing/scan-page.js`,
  `marketing/submit-flow.js` (copy only), `tests/marketing.spec.js`, `sw.js`; **no backend file**
  (a `backend/` diff is scope drift, stated). BACKLOG B-468 `promoted → scan-time-verify`.

### dish-merge-and-erasure-backstop

- `dish-merge-and-erasure-backstop` · **LANDED** (run `20261003`, branch `card/i3-dish-merge-and-erasure-backstop`, merge `ed4f568`) · (I3, track B — decision 194) Migration
  `0083`'s `campaigns_admin.item_id` and `qr_codes.item_id` reference `menu_items(id)` with no
  `ON DELETE`, so `recipes.MergeMenuItem` (re-point recipes, delete the source dish) fails
  `23503` once any campaign references the dish — API-only today, no frontend calls the merge.
  Behaviour after this card: **a dish merge re-points the campaigns and codes that named the
  source dish to the surviving one** (the house convention — "merge re-points all FKs, deletes
  source"), **and a dish, code or subscriber can be deleted without a 500**: the blank-on-delete
  backstop (`ON DELETE SET NULL` on `campaigns_admin.item_id`, `qr_codes.item_id`,
  `subscribers.source_short`) so a future table that forgets the merge path degrades to an empty
  label; and `ON DELETE CASCADE` on `subscriber_events.subscriber_id` (a timeline row cannot be
  blanked — engineer-level call, stated) so a right-to-erasure request is one `DELETE FROM
  subscribers`; plus the FK `0083` never declared, `qr_scans.subscriber_id → subscribers(id)
  ON DELETE SET NULL`, so that one DELETE leaves no dangling id behind (spike correction 2).
  **A code that has been scanned stays undeletable by design** — codes are deactivated, never
  deleted, and scan history is attribution evidence — so `qr_scans.short → qr_codes(short)`
  keeps its plain FK and the card asserts the refusal rather than cascading it (spike
  correction 1). Migration `0086` (Down included; five ALTERs, verbatim in the extraction
  record). `CLAUDE.md`'s stale "Menu items in the Recipes
  tab can be merged the same way" line corrected (triage T-62 finding 1). done_when:
  `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes` red (`23503`) → green;
  `TestSubscriberDeleteCascadesTimelineAndBlanksScans` and `TestCodeDeleteBlanksFirstTouch`
  red → green; `TestScannedCodeDeleteIsRefused` green (asserts `23503 qr_scans_short_fkey`);
  the migration round-trips Down in the package's `zz_migration_down_test` pattern; Go suite
  counts checked. Footprint: `backend/internal/db/migrations/0086_*.sql`,
  `backend/internal/recipes/repository.go` (+`repository_test.go`),
  `backend/internal/marketing/*_test.go` (new test file), `CLAUDE.md`.

### atomic-scan-dedupe

- `atomic-scan-dedupe` · **LANDED** (run `20261005`, branch `card/i4-atomic-scan-dedupe`, merge `f980e12`) · (I4, track B after I3 — decision 195; built and reviewed in run `20261003`, parked on its cleanup of past scan rows, settled at triage as decision 203 — ship as built) The public
  landing's 10-minute scan dedupe (`landing.go` `INSERT … WHERE NOT EXISTS`) is a read-then-write
  under READ COMMITTED: 12 concurrent hits from one IP left 3/7/8/8/8 rows where one was wanted,
  on an unauthenticated route, so the orphan rate's denominator (P-KR3) is inflatable from
  outside. Behaviour after this card: **double-tapping a QR link counts once; a return visit after
  ten minutes still counts; an anonymous scan with no IP hash still counts every time** (the
  stated honest over-count). Mechanism (decision 195, spike-proven shape): migration `0087` adds a
  10-minute tumbling `bucket` column (a STORED generated column —
  `date_bin('10 minutes', scanned_at, …)`, accepted by PostgreSQL 16, spike-proven) and a partial
  unique index on `(short, ip_hash, bucket) WHERE ip_hash IS NOT NULL`; the insert becomes
  `ON CONFLICT (short, ip_hash, bucket) WHERE ip_hash IS NOT NULL DO NOTHING`. Stated trade-off
  (decision 195's, spike correction 2): a tumbling bucket is not a sliding window — two taps that
  straddle a bucket edge count twice. done_when:
  `TestLandingDedupeIsAtomicUnderConcurrency` (12 goroutines on already-open connections
  released by one barrier — a spawn-staggered test passes against the broken code, spike
  correction 1 — one short + ip_hash → exactly 1 row, run 5×) red → green; `TestLandingAnonymousScansNeverDedupe` and `TestLandingCountsAgainAfterWindow`
  green; `TestLandingLogsScanAndRedirectsWithUTM` untouched and green; Go counts checked.
  Footprint: `backend/internal/db/migrations/0087_*.sql`, `backend/internal/marketing/landing.go`,
  `backend/internal/marketing/landing_test.go`.

## Activity J — Scanner and backend guards (triage 20261003 follow-ups)

> **Why here:** Activity I's morning triage (ledger **T-64**, decisions 203–205; receipt
> `reference/triage-20261003.md`) merged run `20261003` with one crew-visible defect the run
> itself introduced still on `dev` — a photo picked while the scanner is waiting on the server
> leaves the current customer's offer with no way to submit it (B-475) — and the operator took
> `dev` with it rather than hold back the verify-before-the-till behaviour (decision 205: "B-475 is
> the fix; it and B-476 want to ride the same card"). The same adversarial review found a dish
> merge that deletes the source dish when the target does not exist (B-479) and a deploy step in
> migration `0086` no test pins (B-480). Authored at the slate sitting of 2026-10-04 (the "triage
> authors the card that discharges the finding it raised" precedent, decisions 167 / 170 / 180 and
> Activity I itself), with the operator's scope choice recorded in `reference/slate-20261005.md`.
> **Trace:** Product objective (P-KR2 — the window workflow must not strand a customer mid-scan)
> and QA objective (the scan-time check's untested arms are gates that pass against broken
> code, the class Q-KR1's refusal depends on). Two cards, two tracks: **A (client)** J1;
> **B (backend)** J2. Spike ledgers under
> `.night-crew/knowledge/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/`.

### photo-scan-guard-and-lookup-arms

- `photo-scan-guard-and-lookup-arms` · **LANDED** (run `20261005`, branch `card/j1-photo-scan-guard-and-lookup-arms`, merge `b0bc48c`) · (J1, track A — **B-475**, **B-476**; decision
  205) Today `onFilePicked` in `marketing/scan-page.js` is not guarded the way the camera path is
  (`decodeBusy`): while the resolver waits up to 3.5 s on the server for code A, a photo of code B
  decodes, the submit machine's F6 gate refuses B ("Finish the current customer first"), and when
  the server then answers A as live the offer card renders with the machine still in `resolving` —
  no order-number field, no submit control; only dismiss-and-rescan recovers. No wrong discount
  results. Behaviour after this card: **a photo picked while the phone is still checking a code is
  not decoded as a second scan — the screen already says "Checking with the server…" and the first
  customer's offer arrives with its submit control, every time.** Mechanism (the backlog lead,
  spike-proven): `onFilePicked` takes the same busy flag the camera path uses; no new machine
  state, event or (state, event) pair (strictness: 460 declared pairs — adding one is a park).
  B-476 in the same card: four spec cases beside `[SV-05]`, each shown red under its mutation
  first — held locally **and** expired by the local clock → the replica's expired result, zero
  lookups; an expired server row → the expired card, never an offer (`clock.isExpired` on the
  server row); a non-200 answer → "couldn't check the server" (`unknownCode`, `verified:false`),
  never an offer; a throwing policy source → the fail-closed refusal (if that arm cannot be
  forced without stubbing the seam, the card says so and names `campaigns-run.sh` as its only
  gate — stated, never silently dropped). done_when: `[PS-01]` online, lookup held for code A,
  photo of held code B picked mid-wait, server answers A live → A's offer card renders **with**
  `#ms-order` and no `#scan-prompt` (red on the pre-change tree — the spike's spec is the red);
  `[PS-02]` the same photo picked with no scan in flight still decodes (the guard is a wait, not a
  disable); `[SV-08]`–`[SV-11]` the four arms, each red under its named mutation then green;
  `[SV-01]`–`[SV-07]` untouched and green. Footprint: `marketing/scan-page.js`,
  `marketing/scanner.js` (only if an arm needs a seam — expected untouched),
  `tests/marketing.spec.js`, `sw.js` (regenerated, count 51 stays). **No `backend/` file.**
  BACKLOG B-475 / B-476 `promoted → photo-scan-guard-and-lookup-arms`.

### merge-target-and-blanking-guards

- `merge-target-and-blanking-guards` · **LANDED** (run `20261005`, branch `card/j2-merge-target-and-blanking-guards`, merge `42fbd67`) · (J2, track B — **B-479**, **B-480**) Today
  `recipes.MergeMenuItem` re-points recipes, campaigns and codes to the target and deletes the
  source without ever checking the target exists, so `POST /api/v1/inventory/recipes/merge` with a
  bogus target id and an unattached source returns 200 `{"rows_re_pointed":0}` and the source dish
  **and its `daily_menu_sales` rows are gone** (cascade); with a campaign attached the same call
  500s and rolls back (the FK refuses). API-only — no screen calls the merge. And migration
  `0086`'s `UPDATE qr_scans … SET subscriber_id = NULL` for dangling ids is load-bearing at deploy
  (without it `ADD CONSTRAINT` fails on a database holding one such row) yet no test pins it:
  removing the statement leaves the four erasure tests and all three migration round-trips green.
  Behaviour after this card: **a dish merge aimed at a dish that does not exist is refused with
  404 and changes nothing; and the deploy step that blanks orphaned scan references has a test
  that fails if it is ever removed.** Mechanism: `MergeMenuItem` reads the target row `FOR SHARE`
  inside the existing transaction before any UPDATE and returns a typed not-found error the
  handler maps to `404 target_not_found` (the handler's existing `cannot_merge_into_self` → 400
  shape); a migration test that migrates to 85, inserts a `qr_scans` row with a `subscriber_id`
  naming nobody, migrates up, and asserts the row survives with the id blanked. done_when:
  `TestMergeMenuItem_MissingTargetIsRefused` (bogus target, unattached source with one
  `daily_menu_sales` row → error, dish and sales row survive; handler → 404) red → green;
  `TestMigration0086BlanksDanglingScanReferences` red (with the UPDATE removed from a worktree copy
  of 0086 → `23503`) → green on the shipped migration; `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes`
  and the four erasure tests untouched and green; Go counts checked (`-p 1`). No new migration.
  Footprint: `backend/internal/recipes/repository.go` + `repository_test.go`,
  `backend/internal/recipes/handler.go` (the 404 arm), `backend/internal/marketing/erasure_test.go`
  (or a sibling migration test file). No frontend file; no `sw.js` move. BACKLOG B-479 / B-480
  `promoted → merge-target-and-blanking-guards`.


## Activity K — Review leftovers and Inventory Setup races (triage 20261005 follow-ups)

> **Why here:** morning triage T-66 (receipt `reference/triage-20261005.md`, decision 207) filed
> the adversarial review's six unfixed findings on run `20261005` — B-482 … B-487 — and named the
> two diagnosed Inventory Setup races (B-459, B-478) as the obvious next small card. At the slate
> sitting of 2026-10-05 the operator chose Activity E for the milestone's last night and then asked
> for the bug fixes as **a second slate after it** (run `20261007`); this activity is that slate's
> content, authored the "triage authors the card that discharges the finding it raised" way
> (decisions 167 / 170 / 180; Activities I, J). **Trace:** QA objective (gates that pass against
> broken code; a states spec that dirties every gate run) and Product (P-KR2's window workflow —
> a crew member who picks a photo mid-check sees something happen; a manager who types in Setup
> keeps what they typed). Four cards, two tracks: **A (client)** K1 → K4; **B (backend + tests)**
> K2 → K3. Spike ledgers under
> `.night-crew/knowledge/spikes/activity-k-review-leftovers-and-inventory-setup-races-triage-20261005-follow-ups/`.

### inventory-setup-races

- `inventory-setup-races` · **LANDED** · (K1, track A — **B-459**, **B-478**) Today
  `ALL_ITEMS` in `inventory.html` has three unsequenced writers (`loadItems()`, the
  `DOMContentLoaded` preload, the alias handler's own refetch) and whichever response lands
  LAST wins, so a nickname added while the item list is still loading can vanish from the chips
  although the server kept it (B-459, the dangerous shape: the view lies and the manager re-adds
  an alias that exists); and opening Setup starts `loadItems()`, whose late re-render of the add
  bar empties a name the manager has already typed, after which the create click returns
  silently with no POST (B-478 — the mechanism behind `inventory.spec.js:2931` / `:2919`).
  Behaviour after this card: **what a manager typed or added in Setup stays on screen — a late
  response never overwrites a newer one, and an empty-name create says so instead of doing
  nothing.** **Spike-corrected scope (2026-10-05):** B-478 reproduces exactly as filed (a late
  groups response wipes the typed name; the create click sends no POST). **B-459's diagnosed
  mechanism does NOT reproduce** under the filed timing — two spike runs with the Setup tab's
  first `GET /items` held until after the alias add and then released gave opposite outcomes
  (run 2: view and server both held the nickname; run 3: NEITHER did — the add never reached the
  server), and in neither did the view disagree with the server. So the B-459 half of this card
  is **the add path's robustness** (why can a click on "add nickname" be dropped while the
  item list's first fetch is in flight? — the night diagnoses it, with the spike's harness as the
  reproduction, and fixes it) **plus request-sequencing hardening, proven by measurement**, not
  the filed overwrite fix; `inventory.spec.js:2186`'s red is retired only if the measurement
  says so. Mechanism: a monotonic request sequence on the items/groups
  fetches (each writer captures `seq` before its fetch and discards a response older than the
  latest applied — last REQUEST wins, never last response); the add bar is rendered once and only
  its group `<select>` options are refreshed when groups land, so `#new-item-name`'s value and the
  chosen group survive; the create click with an empty name shows the same loud alert the
  no-group path uses (UI-R: failures are loud). done_when (red-first, the spike's recipe):
  `[IS-01]` with `GET /inventory/groups` delayed 600 ms, a name typed right after opening Setup is
  still in `#new-item-name` when the response lands and the create click POSTs (RED today — the
  spike's leg 1); `[IS-02]` a stale `GET /items` released after a newer one leaves `ALL_ITEMS` at
  the newer snapshot (a unit-style assertion through `window` on the sequence, RED today because
  no sequence exists); `[IS-03]` empty-name create → the alert names the name field (RED today:
  silent); `[IS-04]` with the Setup tab's first `GET /items` held, a nickname added through the
  UI reaches the server (its POST is observed) and renders — RED today in 1 of 2 spike runs
  (nondeterministic; the night names the mechanism it finds); **measurement, not assertion:**
  `inventory.spec.js:2186` and `:2919/:2931` run 10× alone and 5× inside the `inventory|recipes`
  seam on the merged tree, tallies in HANDOFF; a red that persists is reported with its
  mechanism, never papered over. Footprint: `inventory.html`,
  `tests/inventory.spec.js`, `sw.js` (regenerated, count 51 stays), `bugs.md` (the three named
  reds retire by diagnosis — decision 100's rule). BACKLOG B-459 / B-478 `promoted → inventory-setup-races`.

### dish-merge-shapes-and-backstop-tests

- `dish-merge-shapes-and-backstop-tests` · **LANDED** · (K2, track B — **B-484**, **B-485**,
  **B-487**) Today a dish merge whose SOURCE names no dish answers `200 {"rows_re_pointed":0}`
  and a non-uuid target answers `500 internal_error` (`22P02`); the handler matches
  `ErrMergeTargetNotFound` by `strings.Contains` on the message; the `FOR SHARE` lock and the
  guard's in-transaction placement are pinned by no test (the missing-target test passes with the
  lock removed); and migration 0086's `ON DELETE SET NULL` on `campaigns_admin.item_id` /
  `qr_codes.item_id` is asserted at schema level only — nothing deletes a dish and reads the NULL
  back. API-only; no screen calls the merge. Behaviour after this card: **a merge with a missing
  source is 404, a malformed id is 400, both sentinels are matched with `errors.Is`, the lock is
  pinned by a test that races a delete against the merge, and the deploy backstop is pinned by a
  test that deletes a dish and reads both surviving rows.** Mechanism: `ErrMergeSourceNotFound`
  beside the target sentinel (checked after the target's `FOR SHARE`, inside the transaction);
  `isBadUUID`-style 400 before the query; the handler's two `strings.Contains` arms become
  `errors.Is`; `TestMergeMenuItem_LockHoldsAgainstConcurrentTargetDelete` opens a second
  connection that `DELETE`s the target while the merge holds `FOR SHARE` and asserts the merge
  commits with the target present OR the delete is refused — never a merge into a vanished
  target; `TestMigration0086DishDeleteBlanksCampaignAndCode` seeds a campaign and a code naming a
  dish, deletes the dish, asserts both rows survive with `item_id IS NULL`. done_when (red-first,
  the spike's shapes): missing-source 200 → 404 `source_not_found`; non-uuid 500 → 400 `bad_id`;
  the lock test RED with ` FOR SHARE` removed (the spike's mutation) → green; the backstop test
  green on the shipped migration and RED against a worktree copy of 0086 with `ON DELETE SET
  NULL` changed to `NO ACTION`; `TestRepository_MergeMenuItem_RePointsCampaignsAndCodes` and
  `TestMergeMenuItem_MissingTargetIsRefused` untouched and green. No migration. Footprint:
  `backend/internal/recipes/repository.go`, `handler.go`, `repository_test.go`, `handler_test.go`,
  `backend/internal/marketing/erasure_test.go` (or a sibling). BACKLOG B-484 / B-485 / B-487
  `promoted → dish-merge-shapes-and-backstop-tests`.

### states-screenshots-out-of-tree

- `states-screenshots-out-of-tree` · **PLANNED** · (K3, track B, after K2 — **B-486**) Today
  `tests/states-marketing-stats.spec.js` writes its PNGs into
  `.night-crew/runs/2026-10-02-autonomous/logs/h4/states/`, which is committed, so every gate leg
  that includes the `marketing` seam leaves the tree dirty (five modified files after one subset
  run) and every night since has needed a checkout rule to keep its commits clean. Behaviour
  after this card: **running the states spec never dirties the tree — screenshots go to the
  ignored `test-screenshots/` by default, and a run that wants durable evidence points
  `STATES_SHOT_DIR` at its own logs (the `.gitignore` convention already written for exactly
  this).** Mechanism: `SHOT_DIR = process.env.STATES_SHOT_DIR || path.join(__dirname, '..',
  'test-screenshots', 'marketing-stats')`; the committed H4 set stays where it is as the reviewed
  evidence (not moved, not rewritten); the spec's header comment says so. done_when (red-first):
  `[SS-01]` a Node check beside the spec (or a `[TI-*]` assertion) that `SHOT_DIR` resolves
  outside any git-tracked path — RED today, green after; running the spec in a fresh worktree
  leaves `git status --porcelain` empty (the spike's recipe); the H4 PNGs untouched (B-467).
  Footprint: `tests/states-marketing-stats.spec.js`, `night-crew.toml` roll-call comment only (no
  key, no token). BACKLOG B-486 `promoted → states-screenshots-out-of-tree`.

### scanner-refusal-seam-and-pick-feedback

- `scanner-refusal-seam-and-pick-feedback` · **PLANNED** · (K4, track A, after K1 — **B-482**,
  **B-483**) Today a photo picked while the scanner is checking a code is dropped by the J1 guard
  with no feedback of its own — the result area keeps saying "Checking with the server…" and a
  photo picked while an earlier photo decodes shows nothing at all (B-483); and the refusal
  screen a crew member sees when the campaign-policy source throws is gated by nothing at page
  level — `submit-flow.js` captures `policyFor` ONCE at boot, so no test can reach that arm
  without a seam, and `campaigns-run.sh` never passes a throwing source (B-482). Behaviour after
  this card: **a refused photo pick shows one line — "Finish checking this code first, then pick
  again" — and the fail-closed refusal under a throwing policy source is rendered through the
  page by a test, via a documented boot-time override reachable only from tests.** Mechanism:
  (B-483) `onFilePicked`'s refused branch sets a transient `#scan-note` line under the result
  (no new machine state, event or pair — strictness: 460 declared pairs; a new pair is a park),
  cleared on the next scan; (B-482) scan-page reads `window.__MARKETING_POLICY_SOURCE__` (set
  by `page.addInitScript` before boot, documented in the file header as test-only) in place of
  `createCampaignPolicySource` when present, so `[SV-11]`'s predicate proof gains a page-level
  twin. done_when (red-first, the spike's two recipes): `[PS-03]` photo picked mid-wait → the
  note renders with that copy and clears on the next scan (RED today: no note); `[SV-12]` with the
  override throwing, an offline scan of a held low-value code renders the fail-closed refusal copy
  (RED today: the override is not read, the offer renders); `[PS-01]`/`[PS-02]`/`[SV-01]`–`[SV-11]`
  untouched and green — the whole of `tests/marketing.spec.js` `--retries=0`. Footprint:
  `marketing/scan-page.js`, `tests/marketing.spec.js`, `sw.js` (regenerated, 51);
  `marketing/submit-flow.js` only if the override must be read there (expected untouched —
  stated). **No `backend/` file.** BACKLOG B-482 / B-483 `promoted → scanner-refusal-seam-and-pick-feedback`.


---

## Backlog dispositions this round

**Walked:** the QR offline-redemption **handoff** (`docs/qr-offline-redemption-handoff.md`) —
promoted whole, decomposed into Activities 0–F — and the **two carried QA KR producers** from
last cycle's Activity 5 (`backlog-machine-migration`, `team-records-from-hand-runs`) — re-promoted
into Activity G.

**Not walked (deliberate, said out loud):** the round was scoped by the operator to the handoff,
so the **46 CLI-visible `[new]` backlog items were not individually walked** this round. They stay
untouched at `new`. The genuinely roadmap-worthy carry-overs among them are already known — they
are the deliberately-parked families from T-46 (armed reds retired only by diagnosis per decision
100; gate-coordinate-safety guarded structurally by the `:5434` test cluster; measurement debts)
plus a small post-T-46 tail (`B-176` armed red; `B-349`'s remaining half tracked clone-side). The
still-`PLANNED` `media-recovery` card (B-173) is **not** superseded by this cycle and carries
forward as open. **A dedicated backlog-walk round is worth scheduling** once
`backlog-machine-migration` (Activity G) makes the legacy entries machine-visible — the exact gap
that card closes.

| Item | Disposition |
|---|---|
| `docs/qr-offline-redemption-handoff.md` (design of record) | **promoted** → Activities 0–F (whole handoff, operator's scope call) |
| `backlog-machine-migration` (B-02, B-168, B-12, B-133) | **promoted** → Activity G (carried; Q-KR2) |
| `team-records-from-hand-runs` | **promoted** → Activity G (carried; Q-KR3) |
| `media-recovery` (B-173) | **left open** — unrelated to this cycle; carries forward |
| 46 CLI-visible `[new]` items | **left `new`** — not walked this round (scoped to the handoff); machine-visible walk deferred to a backlog round after Activity G |

**Round notes recorded at this sitting** (ledger entry accompanies the sign-off commit): the two
engineering calls R1 (RxDB reuse is greenfield) and R2 (redemption orchestrated in HQ Go, Supabase
stays the sole arbiter) were decided here and are revisitable in Activity 0's spike if field
observation (#6) changes device topology. No `/nc-retro` preceded this round; recorded as an
absence, round proceeded on close record + backlog + OKR grades. This target has no `openspec/`, so
cards are plain-markdown items and no OpenSpec deferral markers apply.
