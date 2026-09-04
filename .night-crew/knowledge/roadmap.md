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
- **Status:** `DONE` · `DRAFTING` (overnight) · `PLANNED` (white) · `BLOCKED`.
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
  and Twilio to an app that is otherwise static-HTML + Go + self-hosted Postgres. That is a real
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
| **delivery** | Twilio integration (QR gen + MMS send on form submit), consent capture |
| **toast join** | SMTP ingest mailbox → CSV parser → staging table, reconciliation view + orphan-rate |
| **planning docs** | `.night-crew/knowledge/BACKLOG.md`, slate/closeout templates under `reference/` (Activity G) |

---

## Activity 0 — Resolve what the build rides on (unknowns + longest external leads)

> **Why first:** three build-blocking facts and two multi-week external processes must not sit on
> the critical path. Nothing expensive is built on an unanswered #6. **Trace:** Product objective.

- **`redemption-unknowns-spike`** · **PLANNED** · Attended / field observation. Answer handoff
  §14 #1 (business-date cutoff hour), #2 (order-number format — digit count, prefix, reset
  behavior), #3 (which Toast scheduled report carries order# + business date + discounts), and
  **#6 — the load-bearing one: do the truck's three devices genuinely lose connectivity
  independently, or are they always together on one hotspot?** Records each answer in the ledger.
  If #6 says "always together," the card flags that Activity B (offline-first replication) is
  likely over-built and proposes the lighter live-shared-server alternative before B is planned in
  detail. `n/a — no code change`. Footprint: planning docs (ledger) + Toast settings read-only.

- **`external-accounts-provision`** · **PLANNED** · Attended (operator holds accounts/billing).
  Stand up the two external dependencies the whole cycle needs: (a) a **Supabase project**
  (account, project, anon/service keys wired into HQ's existing secret pattern — dev/test project
  distinct from any prod project, per decision 155's spirit); (b) a **Twilio account + toll-free
  number** for delivery, and **start A2P/10DLC brand+campaign registration in parallel** (1–3 wk
  external lead — §11; toll-free covers the cycle, 10DLC covers later scale). STOP-handling and
  consent language drafted here so Activity E ships compliant (§11, R5). `n/a — no code change`.
  **Gates Activity A** (Supabase) and **Activity E** (Twilio). Footprint: external accounts +
  secret pattern.

## Activity A — The attribution spine (the Supabase arbiter)

> **Why here:** every client that reads or burns a code depends on the schema and the atomic
> `redeem()`. The single conditional `UPDATE` is the **only** thing enforcing single use (§6).
> **Trace:** Product + Engineering objectives.

- **`supabase-schema-and-rls`** · **DONE** (overnight-20260904, branch
  `wo-supabase-schema-and-rls` — migration + seed + verify harness landed in `supabase/`, all
  gates green fresh+warm against the local substrate; triaged 2026-09-04, merged to dev) · The `campaigns` / `codes` / `scan_attempts`
  schema (§4 — including the `unverified_code boolean` flag on `scan_attempts`, F2) with
  `token_hash` never storing the raw token, the `updated_at` index (the
  replication checkpoint key), RLS policies that let each device see only what it needs (§7.2),
  and the tables added to the `supabase_realtime` publication (§7.1 — the usual reason Realtime
  looks broken). Seeds `requires_online` per campaign (operator sets the threshold, #5). SQL kept
  in-repo under `supabase/`. done_when: schema applies clean against a fresh Supabase project and
  a row inserted on one client appears on a second subscriber. Footprint: supabase arbiter.

- **`redeem-rpc-race-proof`** · **DONE** (overnight-20260904, branch
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

- **`rxdb-pull-replica`** · **DRAFTING** (overnight-20260905, branch `wo-rxdb-pull-replica` —
  `marketing/sync/` pull modules with the keyset `{updated_at, id}` checkpoint closing GAP-1,
  vendor surface widened (`replicateRxCollection` + `Subject`), standalone substrate harness
  green, modules deliberately unwired pending Cards 5/6; awaiting morning triage) ·
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

- **`scan-attempts-push-conflict`** · **DRAFTING** (overnight-20260905, branch
  `wo-scan-attempts-push-conflict` — `marketing/sync/push-replication.js` device-owned push
  module: offline queue + `enqueueAttempt` dedupe + redeem-then-land handler with GAP-1's two
  belts (persisted burn outcome before landing; own-device `already_used` = accepted), loser's
  flip rendered from the codes-side pull replica, standalone substrate harness; module
  deliberately unwired pending Cards 5/6; awaiting morning triage) ·
  The device-owned, **push-only**
  `scan_attempts` collection (§4 — opposite replication direction, the key structural decision).
  The push handler batches pending attempts through `redeem()` and writes the outcome back onto
  the local row; the `conflictHandler` flips a losing device's UI from "redeemed ✓" to "already
  used at 6:42pm" (§6). done_when: a lost-race attempt renders "already used" with the winning
  time/device. Footprint: rxdb replica.

- **`clock-offset-on-sync`** · **PLANNED** · On every successful sync, store
  `serverNow − deviceNow` and apply that offset in the offline `expires_at` comparison (§5.1) —
  a tablet with a wrong date must not silently accept dead codes. done_when: with the device clock
  set 2 days fast, an expired code is still rejected offline. Footprint: rxdb replica.

## Activity C — The scanner screen (staff redemption at the window)

> **Why here:** this is the operator-facing action and the close bar's leg 1. It reads from B
> (offline) and burns through A (online, via D). **Trace:** Product objective. **Locks §14 #9
> (confirm-then-burn), #12 (offline_override holder), #13 (reachability signal).**
> **Opens with a spike (operator's call this round):** decide the client state-machine approach —
> **XState vs a hand-rolled parallel-region machine** — proving the §19 F1/F2/F3/F6 regions in
> HQ's vanilla-JS context before the scanner cards are built. Adopting XState is a new client
> dependency in a deliberately no-framework app; the spike settles it against the real screen.

- **`marketing-tile-and-page`** · **DRAFTING** (overnight-20260905, branch
  `wo-marketing-tile-and-page` — tile + `TILE_SLUGS` entry, `marketing.html` shell (Scan live,
  three labeled placeholders), `SeedHQApps` seeds `marketing` + the `marketing-offline-override`
  entitlement surface with first-registration-only grants, precache 31→32; awaiting morning
  triage) · Add the **Marketing** tile to `index.html`'s grid
  + `TILE_SLUGS`, create `marketing.html` (page shell with the four sub-sections: Scan / Campaigns
  / Subscribers / Redemption stats, §16), seed `('marketing','Marketing','📢')` in `SeedHQApps()`
  so the tile is permission-gated, and grant the `marketing` app to the relevant roles. Enforce
  the create/stats gates inside the handler, not just at the tab (§16 permissions table).
  Regenerate `sw.js` (new precached page — the precache-count invariant will move by 1
  deliberately) and commit it. done_when: a `team_member` sees Scan; a non-granted user sees no
  tile; `build-sw.js` exits 0. Footprint: scanner UI + redemption backend (seed) + `sw.js`.

- **`camera-scanner-decode`** · **PLANNED** · Camera via `getUserMedia`, decode with
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

- **`redemption-submit-flow`** · **PLANNED** · The heart of the window workflow (§13). Large
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

- **`gstate-arbitration-machine`** · **DRAFTING (overnight)** · `backend/internal/redemption` — the §18
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

- **`identity-code-and-qr`** · **PLANNED** · **One permanent identity code per customer** — the
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

- **`mms-send-on-signup`** · **PLANNED** · Form submit → generate the identity code in Supabase →
  **send the QR as an MMS image, not a link** (§11 — an image lives in the thread and opens with
  no signal; a link needs signal at the window). Explicit **consent capture at point of
  collection**, working **STOP handling**, sends over the registered number (§11 compliance,
  R5). done_when: the operator's test signup produces a scannable image in their Messages thread.
  Footprint: delivery. **External dependency: a live sending number (Activity 0).**

## Activity F — The join lands (SMTP ingest + reconciliation)

> **Why here:** needs codes actually redeemed to have something to reconcile; it is the close
> bar's leg 2. Everything is **T+1** by design (§13). **Trace:** Product + QA objectives.

- **`smtp-toast-ingest`** · **PLANNED** · A dedicated ingest mailbox receives the scheduled Toast
  report (#3); an inbox watcher extracts the CSV, normalizes, and loads a staging table. **Key on
  `(business_date, order_number)` and upsert — never blind-insert** (§13 idempotency; the same
  report *will* arrive twice). Dedicated mailbox, restricted access, no forwarding (§13 security).
  done_when: ingesting the same report twice leaves one row per order. Footprint: toast join.

- **`reconciliation-view`** · **PLANNED** · The three-bucket view built from day one (§13): `matched`
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

- **`backlog-machine-migration`** · **DONE** (overnight-20260904, branch
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

- **`team-records-from-hand-runs`** · **DONE** (overnight-20260904, branch
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
