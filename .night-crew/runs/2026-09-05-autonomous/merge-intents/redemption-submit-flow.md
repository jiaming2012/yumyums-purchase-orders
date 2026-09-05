# Merge intent — Card 6 · `redemption-submit-flow` (run 20260905)

Branch: `wo-redemption-submit-flow` (cut from the run base @ ea19130 — carries all six
landed cards: Card 1's shell + `marketing`/`marketing-offline-override` seed, Card 7's
POST /api/v1/marketing/redeem, Cards 2/3/4's marketing/sync pull + push + clock, Card 5's
scanner + `window.MarketingScan` + `#scan-submit-slot`). Card authority: slate-20260905
Card 6 (the signed entry, verbatim in the launch prompt). Footprint: scanner UI
(`marketing.html` + `marketing/`) + `sw.js` move — plus the mechanically-forced
companions stated below. Spike authority:
`.night-crew/knowledge/spikes/activity-c-scanner-screen/redemption-submit-flow.md`
(+ `.extraction.md`) — 18-sequence conformance suite, lockstep fuzz, the overlay-region
XState variant, Addendum 2 strictness pricing. Engine fork **operator-resolved at
slate sign-off: XState, overlay-region shape, no-silent-no-ops strictness** (roadmap
"Modeling approach" paragraph — overriding the extraction's hand-rolled recommendation).
Design authority: docs/qr-offline-redemption-handoff.md §8, §13, §16, §19.1/.3/.4
(F1/F2/F3/F6), P-KR4; docs/ui-design-rules.md UI-R1/2/3/6.

## Shared files touched

- `lib/xstate.umd.min.js` (NEW, vendored) — XState **v5.32.6** UMD single-file classic
  script (the SortableJS/html5-qrcode pattern, spike-proven in real Chromium).
  🛑 Recovered like Card 5's lib: the slate says "the vendored file sits in the
  committed spike dir", but the spike's own `.gitignore` lists `web/xstate.js` and
  `js/node_modules/` — the file was never committed anywhere. This card copies the main
  checkout's spike-working-tree artifact (both copies there are byte-identical) and
  commits it — the first committed copy in any tree. **Recorded identity:
  47,268 bytes (matches the slate's expected size exactly), sha256
  `e7f04e1f780f5f67b0d4286b85e6d01e16473ab42b5a466cd56282556838fa28`.**
  Placed in `lib/` (single-file classic-script precedent; `vendor/` stays pinned to
  exactly `['vendor/rxdb.bundle.js']` by tests/sw-manifest.spec.js). UMD global:
  `window.XState`; the same file answers Node `require()` (CommonJS branch of the UMD
  wrapper) — so the conformance/fuzz gates run against the EXACT shipped artifact.
- `marketing/submit-machine.js` (NEW) — the strict overlay-region XState machine
  (§19.1 client statechart; base shape = the spike's proven
  `machine-xstate-overlay.mjs`, behavior-pinned by the 18-sequence suite + fuzz).
  Dependency-injected like the marketing family: XState primitives
  (`{createMachine, createActor, assign, raise}`), input, effects sink and mode all
  arrive as parameters — the same file runs in the browser (window.XState) and under
  the Node gates (require of the vendored UMD). Strictness per the operator's verbatim
  call: every (state, event) pair over the 20-event alphabet is a DECLARED decision —
  real transitions or per-state enumerated `ignores:` lists compiled to explicit no-op
  transitions; anything else falls to a per-region `'*'` wildcard that **throws in
  test/dev mode (`mode:'throw'`)** and in the **production build (`mode:'model'`,
  the page default) raises the modeled, visible, retryable `unexpectedEvent` scan
  state** (loud + logged via onTrip → console.error → log.js beacon; RETRY returns to
  the interrupted state, NEXT_CUSTOMER resets — never a dead actor at the window).
  Exports `createSubmitMachine` + `EVENT_ALPHABET` + a `declaredPairs()` accounting.
- `marketing/submit-support.js` (NEW) — pure/injectable helpers: the named placeholder
  order-number validator (constant below), the Toast business-date computation off the
  sync clock's epoch (constants below), the #13 reachability probe factory
  (timings below), and the stable device-id (localStorage `hq_marketing_device_v1`,
  crypto.randomUUID on first boot).
- `marketing/submit-flow.js` (NEW) — the browser wiring: mounts the order-number +
  submit flow into Card 5's `#scan-submit-slot`, renders the persistent `#scan-conn`
  online/offline indicator and the F6 finish-current-customer overlay, owns the
  resolver-result → machine event mapping, the online POST to Card 7's endpoint, the
  offline-override enqueue through `window.MarketingScan.enqueue` (the serialized
  wrapper — never raw enqueueAttempt), and exposes `window.MarketingSubmit` (machine
  adapter, `probeNow`, `setCampaignPolicy`, `reportChannelStatus` — the test seam and
  the future realtime-channel wiring point).
- `marketing/scan-page.js` (EDIT — Card 5's file) — **what of Card 5 must survive:**
  everything its intent froze (the DOM contract, `window.MarketingScan`'s surface,
  the scanner.js exports, the ONE-delegated-listener rule, the default
  `() => false` online probe). This card adds ONE additive registration surface:
  `setSubmitFlow(handlers)` (+ `scanAgain` exposed on the api), consulted at five
  existing choke points — a scan gate before resolution (F6 session semantics), an
  onResult/onRender pair after it, an onScanAgain session-reset hook, and an action
  route inside the SAME delegated click/change listeners (plus ONE new delegated
  `input` listener on the host, the workflows.html click+input convention). Every
  hook is fail-open (a submit-flow error degrades to Card 5's scanner, loudly
  console.error'd) — Card 5's card must keep working if this card's JS breaks.
- `marketing/sync/push-replication.js` (EDIT — Card 3's module, ADDITIVE, the Card 4
  precedent) — `enqueueAttempt` gains one optional field: `pos_business_date`
  (default: the existing `scanned_at.slice(0,10)` UTC-date behavior, byte-identical
  for every existing caller — Card 3's harness stays green unchanged). Why: Card 3
  recorded "Toast business-date semantics belong to §13's card" — this IS the §13
  card, and the slate binds business-date to the Toast-cutoff constant, never
  new Date(). No other function in the file changes.
- `marketing.html` (EDIT — Card 1's shell, Card 5's scanner section) — adds
  `<script src="lib/xstate.umd.min.js" defer>` before the module scripts and
  `<script type="module" src="marketing/submit-flow.js">` after scan-page.js, plus
  inline CSS for the indicator / submit slot / prompt overlay. Four-tab shell, auth
  probe, `#scanner-host` contract untouched.
- `tests/machine/` (NEW) — the card's own gate, repo-owned so the spike's throwaway
  scripts are not load-bearing: `conformance.mjs` (the 18 sequences, transcribed from
  §19.4 — machine-agnostic, byte-equivalent behavior to the spike's),
  `reference-machine.mjs` (the spike's hand-rolled machine, copied in as the fuzz's
  behavioral reference — proven observably equivalent to the overlay shape across
  40k walks in the spike), `run-conformance.mjs` (wires vendored UMD + production
  machine, throw mode), `strictness.mjs` (undeclared-pair throws in test build;
  modeled `unexpectedEvent` in prod build; the three reachable-benign declared
  ignores incl. the PROBE_TIMEOUT-while-offline brick), `lockstep-fuzz.mjs`
  (reference vs production, **with the per-step liveness assertion — a dead actor
  fails the walk instantly, pass-by-death impossible**; 20,000 walks × 20 steps ×
  2 seeds).
- `tests/marketing.spec.js` (EDIT — the slate-named home) — a new
  `describe('Redemption submit flow …')` appended; Card 1's and Card 5's tests
  untouched.
- `build-sw.js` (EDIT) — ONE glob added: `'lib/xstate.umd.min.js'` (the narrow
  single-file form, Card 5's precedent — never `lib/**`). The three new
  `marketing/*.js` modules ride the EXISTING `'marketing/*.js'` glob.
- `backend/Dockerfile` — **zero edits needed, verified:** `COPY lib ./lib` +
  `cp -r ../lib` already stage all of lib/, and `COPY marketing/*.js ./marketing/`
  already stages the new modules (decision-59 pairing intact by construction).
- `sw.js` (REGENERATED after the asset commits — build-sw.js reads git HEAD) —
  **precache count moves 39 → 43; all four deliberate additions enumerated:**
  `lib/xstate.umd.min.js`, `marketing/submit-machine.js`,
  `marketing/submit-support.js`, `marketing/submit-flow.js`. Regenerated in THIS
  worktree's lockfile-true `npm ci` env (Card 2's toolchain finding).
- `CLAUDE.md` (EDIT) — the precache-count line, 39 → 43, same change set as sw.js.
- `night-crew.toml` (EDIT) — the `marketing` seam, decision recorded below.
- `.night-crew/knowledge/roadmap.md` — one-line flip `redemption-submit-flow`
  PLANNED → DRAFTING (overnight-20260905), the run's convention. No other line moves.
- This file (amended as evidence lands); `card6-*.log` — committed whole gate logs.
- `index.html`, `backend/**/*.go`, `supabase/`, `sync-rxdb/`, `vendor/`,
  `tests/grant-enforcement-parity.spec.js`, other `tests/*.spec.js` — **nothing
  here.** No backend change: the endpoint exists (Card 7), and this card mounts no
  new route, so the `marketing-offline-override` NA_WITH_REASON parity entry stays
  truthful (no RequirePermission mount exists for it; the entitlement gate is
  client-side by design — fork #12 + Card 7's G6 note both anticipate exactly this).

## What must survive any merge

1. **The vendored `lib/xstate.umd.min.js` at its recorded identity** (47,268 B,
   sha256 above) + the `'lib/xstate.umd.min.js'` glob + the four-entry precache
   addition — dropping any one bricks the SW install for returning clients
   (decision 59 / B-37).
2. **The strict machine's contract:** the 18-sequence conformance suite and the
   lockstep fuzz (with per-step liveness) in `tests/machine/` are the behavior pin —
   the machine may be refactored only against them. Mode split: `'throw'` in every
   Node gate; `'model'` (the `unexpectedEvent` state) is the page default.
3. **The §8 refusal:** `requires_online=true` → no override, ever, for anyone — the
   machine guard (`overrideAvailable = canOverride && !requiresOnline`), the UI
   branch, and the REDEMPTION-SUBMIT-FLOW-3 Playwright test that pins it.
4. **Confirm-then-burn (§13):** no dispatch path to SUBMIT or OVERRIDE_CONFIRM
   without a validator-passing order number — the UI gate + the Playwright test
   asserting zero POSTs without it.
5. **The F2 write shape:** an unknown-code offline override enqueues
   `offline_override=true` AND `unverified_code=true` through the SERIALIZED
   `window.MarketingScan.enqueue`.
6. **Card 5's frozen surfaces** (listed in its intent) — this card consumes, never
   re-shapes, them; `setSubmitFlow` is additive and fail-open.
7. **`enqueueAttempt`'s additive-optional `pos_business_date`** — existing callers
   byte-identical (Card 3's harness green unchanged).
8. The roadmap flip; the `card6-*.log` evidence; this intent.

## What is safe to drop

- Probe/timeout tuning constants (values below) — any values keep the machine
  correct; tests drive `probeNow()` and never wait on the interval.
- The `reportChannelStatus` future-wiring surface if a later realtime card replaces
  it wholesale.
- Nothing else in this branch is scratch.

## Red-first

Greenfield (the family pattern — git history is the chronology). The test files land
FIRST (tests/machine/* + the new Playwright describe), are RUN against the pre-change
tree (no `lib/xstate.umd.min.js`, no `marketing/submit-*.js`, no hooks), and every
leg reds in the behavior-absent shape; commands + exit codes captured to
`card6-red.log` and committed BEFORE any production file. Card 5's process lesson is
binding: the working tree is NOT touched while a webServer-based Playwright leg runs.

- Node leg: `node tests/machine/run-conformance.mjs` (and strictness.mjs, fuzz) —
  red because the production machine module and the vendored lib do not exist
  (import failure named in-log, EXIT≠0).
- Playwright leg: the new describe only (`-g 'Redemption submit flow'`) — red
  because no submit UI mounts, no indicator renders, no machine exists.

Evidence: `card6-red.log` (committed with the tests), greens in `card6-conformance.log` /
`card6-fuzz.log` / the full-suite `card6-e2e.log`.

## The card's recorded build calls

1. **#13 probe/heartbeat timings** (`marketing/submit-support.js`):
   - `PROBE_TIMEOUT_MS = 3500` — the §13 requirement is that a connected-but-hanging
     LTE link resolves to offline "within the probe timeout"; 3.5 s is long enough to
     survive a slow-but-alive tunnel hop (prod sits behind Cloudflare Tunnel to a
     home box; healthy p95 is well under 1 s) and short enough that the window staff
     see the indicator flip before they would naturally re-tap.
   - `PROBE_INTERVAL_MS = 10_000` — a 10 s cadence bounds the stale-indicator window
     at the counter to one customer interaction, at ~6 requests/min/device against
     the lightest endpoint HQ has; with ≤5 crew devices that is noise.
   - `SUBMIT_TIMEOUT_MS = 12_000` — the submit POST's own abort guard (server-side
     arbitration budget is 504-guarded around 10 s; the client outlasts it slightly
     so a server 504 arrives as itself, not as a client abort), then `SRV_ERROR` —
     loud + retryable (UI-R6).
   - **Probe target: `GET /api/v1/health` on HQ, not Supabase** — recorded deviation
     from the §13 sketch, with cause: Card 7 moved the online submit to HQ
     (`POST /api/v1/marketing/redeem`), so HQ reachability IS the signal that decides
     whether submit can succeed; a Supabase probe could green-light a submit whose
     actual target is down. The substrate's liveness rides the future realtime
     channel wiring (`reportChannelStatus`: SUBSCRIBED → RESUBSCRIBED,
     CHANNEL_ERROR/TIMED_OUT → PROBE_TIMEOUT — the slate's onStatus wiring point,
     exposed and documented; no channel exists in the browser wiring tonight).
   - **RESUBSCRIBED synthesis without configured sync** (recorded): on a probe
     success from `offline` the flow sends `CONN_UP`, calls
     `MarketingScan.resync()` (no-op until provisioning lands), then sends
     `RESUBSCRIBED` — with no replica configured there is nothing to refetch, so
     "as fresh as this device can be" is the honest reading and P-KR4's auto-resume
     stays live. When provisioning lands, RESUBSCRIBED moves to the
     SUBSCRIBED/initial-replication callback (comment at the site).
2. **The named placeholder order-number validator:**
   `TOAST_ORDER_NUMBER_PLACEHOLDER_PATTERN = /^\d{1,6}$/` in
   `marketing/submit-support.js` — digits only, 1–6, trimmed. PLACEHOLDER until
   Activity 0 delivers the confirmed Toast format (#2); the constant name is the
   grep target for that card. Companion §13 constants:
   `TOAST_BUSINESS_DATE_CUTOFF_HOUR = 4` (the common Toast default — confirm in
   Toast settings, #1) and `TOAST_BUSINESS_DATE_TIMEZONE = 'America/Chicago'`
   (the truck's POS timezone, the repo's standing Chicago convention).
   `toastBusinessDate(clock.now())` shifts the SYNC CLOCK's epoch back by the
   cutoff and formats in the POS timezone — `new Date(ms)` appears only as an
   Intl formatting container for the clock's epoch, never as a time source.
3. **night-crew.toml `marketing` seam — ADDED** (the slate homes this call here).
   Keys `marketing.html`, `marketing/`, `tests/marketing.spec.js` → token
   `["marketing"]`, which as a Playwright path regex selects exactly
   `tests/marketing.spec.js` (verified at landing: no other spec filename contains
   the token; the repo-hygiene roll-call guard covers the Operations tokens only and
   is untouched). Honest confinement: a future PURE marketing-JS/html card runs its
   own spec file; any card that moves the precache touches `sw.js`/`build-sw.js`
   (undeclared) and still de-confines to the full suite — exactly the B-37 posture.
   `marketing/sync/` is INSIDE the seam key on purpose: the only page that imports
   it is marketing.html's module graph, and its substrate gates are standalone
   harnesses graded on their own exit codes (B-345), not Playwright.
4. **Vendored xstate identity:** 47,268 bytes, sha256
   `e7f04e1f780f5f67b0d4286b85e6d01e16473ab42b5a466cd56282556838fa28` (recorded
   above with the recovery story).
5. **Machine-shape calls** (internals delegated to this card by the slate):
   - Base shape: the spike's overlay-region variant — connectivity ∥ overlay ∥ scan;
     the finish-first prompt is the overlay region, so scan progress NEVER moves
     under it and DISMISS is the whole go-back mechanism (F1's principle).
   - Declared ignores are per-state enumerated `ignores:` arrays compiled to
     explicit no-op transitions; the compiler rejects an event listed as both
     transition and ignore. Guard-fail and prompt-gated fall-throughs are declared
     no-op tails (`{}`) — pair-level strictness; payload-level validity is the
     page mapping's job (an unmapped resolver kind never reaches the machine).
   - The three Addendum-2 reachable-benign pairs are DECLARED ignores, named in
     tests: PROBE_TIMEOUT while offline (the brick), RESUBSCRIBED while online,
     late RESOLVED after NEXT_CUSTOMER.
   - `unexpectedEvent` (model mode) captures `{event, scan, conn, overlay}` +
     the interrupted state; RETRY returns there (generated guarded branches, the
     spike's 13-branch precedent), NEXT_CUSTOMER resets; the overlay closes over
     it; connectivity keeps running.
   - **No ORDER_BAD event**: ORDER_OK means "a validator-passing order number was
     captured"; an edit back to invalid disables dispatch UI-side and SUBMIT/
     OVERRIDE_CONFIRM re-validate at dispatch — the machine alphabet stays exactly
     the spike's 20 events, so the fuzz runs against the unmodified proven
     reference.
6. **Resolver-kind → machine-kind mapping** (page wiring, recorded):
   `offerReady`→`offerReady`; `deferToServer`→`spentLocally` (the machine's own
   conn-guard reproduces F3-online, single authority); `embeddedOffer`→
   `unknownCode` for POLICY (prior use unverifiable; display keeps Card 5's
   embedded card; an override on it honestly writes `unverified_code=true`);
   `expiredLocally`→`expiredLocally`; `spentLocally`→`spentLocally`;
   `invalidPayload`/`decodeError` never reach the machine (no token → no session).
7. **`code_id` for an unknown-code attempt = the token_hash** (64 hex < the
   column's 100): no code row exists to name; the local queue needs a dedupe key
   and the arbitration card owns server-side resolution of unverified attempts.
8. **Entitlement (`canOverride`)**: read from `/api/v1/me/apps`
   (slug `marketing-offline-override` — #12, any role, per-user grant), cached in
   localStorage so an offline RELOAD keeps the grant (the override exists FOR
   offline); cache refreshed on every successful read; no cache + no server →
   false (fail-closed). Client-side gating IS the enforcement (Card 7 G6: the
   server trusts the flag).
9. **`requires_online` is not client-readable tonight — flagged for the operator.**
   The flag lives on `campaigns` server-side and no replica carries it (Card 2's
   schema: codes/offers rows only). The machine + UI + tests implement the §8
   refusal fully via an injectable `setCampaignPolicy(fn)` (the Playwright test
   injects `requiresOnline:true` and proves the no-override branch); the DEFAULT
   policy answers unknown → `requiresOnline:false`, because unknown→true would
   silently delete F2's DECIDED override affordance for every code on the truck.
   Nothing existing is weakened (the branch is BUILT tonight, and it fires wherever
   the flag is known); arming it with real data needs a small future card
   (replicate the flag or embed it in the pull). Morning-list item, not a PARK:
   no policy fork is decided here beyond the honest default.
10. **Already-used display data** (⚠️ card): when the online submit answers
    `already_used`, when + which device render from the LOCAL codes replica
    (`redeemed_at`/`redeemed_by` — Card 3's flip source; devices cannot read
    scan_attempts), with an honest "another device" fallback when the replica
    has not caught up.
11. **Order-number draft persistence — decided NO** (mechanic 6): the draft lives
    in page state for the session only. A window interaction is seconds long and
    single-customer; persisting a typed order number across a reload risks
    attaching a STALE number to the wrong customer — the §13 data-quality risk —
    for no workflow gain. The machine's F6/P-KR4 session persistence needs are
    in-memory by design. Explicitly NOT wired into debouncedSaveField (different
    subsystem, per the card).

## Gate evidence (run in this worktree; full logs committed beside this file)

- **RF** — `card6-red.log` (committed `46701b2`, BEFORE any production file):
  the three node legs EXIT=1 (production machine absent, named in-log); the
  new Playwright describe **10/10 failed, EXIT=1**, every failure the same
  behavior-absent shape (`window.MarketingSubmit` never appears — 30s
  waitForFunction timeout). Greens: the same legs below. Disclosed in-log:
  npm's "Exit handler never called" bug hit the worktree `npm ci` twice
  (reify complete, bin links missing; `npm rebuild` restored them).
- **THE CARD'S OWN GATE** —
  - `card6-conformance.log` (HEAD 5f1a8ec): `run-conformance.mjs` — **ALL 18
    SEQUENCES HELD** on the production machine in THROW mode, **0 tripwire
    hits**, **460 declared pairs across 23 states × the 20-event alphabet**
    (460 = 23×20: full coverage, nothing implicit); `strictness.mjs` — **9/9
    held** (undeclared pair throws in the test build + kills the actor;
    modeled `unexpectedEvent` in prod — alive, named, RETRY returns to the
    interrupted state, NEXT resets, the prompt closes over it; the three
    reachable-benign pairs declared by name). Both **EXIT=0**.
  - `card6-fuzz.log`: lockstep vs the spike's proven hand-rolled reference —
    **40,000 walks × 20 steps, seeds 20260904+20260905, 25-entry alphabet,
    per-step liveness ARMED: deaths=0 (tripwire hits 0), divergences=0,
    EXIT=0.** Pass-by-death impossible; zero behavior drift in the fuzzed
    space.
- **G2 (Playwright, FULL suite ×2 — page-touching card → full fallback,
  `TEST_PORT=3106 TEST_DB_NAME=hq_test_e2e_c6 --retries=0`, one summary block
  each):**
  - **Run 1** (`card6-e2e.log`, HEAD 07e9835): **856 passed / 5 failed / 6
    skipped (19.3m), EXIT=1.** Verdicts: `marketing.spec.js:521` hash-caching
    — **INTRODUCED by this card** (the gate's private hasher starved the
    page's cache of its hit) → fixed in `f226834` (gate shares the page's
    memoized hasher, sw regen `3318ee5`); `workflows DBL-05` + `GATE-02` —
    slate-armed baseline; `sync SYNC-RF-01` + `workflows DBL-04` — not
    slate-named, both **green in isolation on the same tree** (targeted
    re-runs, 1 passed each) → load-shaped flakes in exactly the specs the
    seam notes call load-sensitive.
  - **Run 2** (`card6-e2e-2.log`, HEAD 3318ee5): **859 passed / 2 failed / 6
    skipped (18.5m), EXIT=1.** `workflows DBL-05` — the armed baseline red;
    `marketing.spec.js:448` F3-offline — **the second and last
    diff-attributable find**: Card 5's test assumed the `() => false` default
    probe, which THIS card's boot replaces (a race: green in run 1, red in
    run 2) → test setup fixed in `9c6f04e` (forces offline through the real
    probe; assertions unchanged). Whole-file proof after both fixes:
    `npx playwright test tests/marketing.spec.js --retries=0` → **30 passed
    (23.4s)** — Card 1 + Card 5 + Card 6 describes all green together.
  - Net: **zero unexplained reds**; the two attributable finds are exactly
    the two seams this card touches (the shared hasher, the probe handoff),
    each fixed with the existing test as the regression guard.
- **G4** — `card6-g4.log` (final HEAD): `node build-sw.js` twice — both
  **EXIT=0, 43 files precached (2788.5 KB)**, reachability 30 files parsed /
  46 refs resolved / 0 outside, tree clean after pass 2 (committed sw.js
  reproduced byte-identical; only uncommitted gate logs in status). Version
  parity **1.6.2 ≡ 1.6.2 ≡ 1.6.2** (version.go Frontend / package.json /
  version.json) — no bumps tonight.
- **G1 / G2 (Go)** — n/a: no `.go` file touched (verified by the branch
  diff); the endpoint is Card 7's landed surface, consumed as-is.
