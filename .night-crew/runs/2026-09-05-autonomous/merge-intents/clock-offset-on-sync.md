# Merge intent — Card 4 · `clock-offset-on-sync` (run 20260905)

Branch: `wo-clock-offset-on-sync` (cut from the Track B base @ 37aa1a6 — includes Card 1's
marketing shell + seed, Card 7's arbitration machine, Card 2's marketing/sync pull replicas +
widened vendor bundle, and Card 3's scan_attempts push module). Card authority: slate-20260905
Card 4. Footprint: rxdb replica (`marketing/sync/` — extended, no page files, no sw.js, no
backend). Spike authority: `.night-crew/knowledge/spikes/activity-b-offline-first-replica/
clock-offset-on-sync.md` (Date-header source confirmed; 2-day skew recovered to 196 ms; the
`now()` RPC fallback not needed). Design authority: docs/qr-offline-redemption-handoff.md §5.1.

## Shared files touched

- `marketing/sync/clock.js` (NEW) — the sync clock: `createSyncClock()` capturing
  `offset = serverNow − deviceNow` from a pull response's `Date` header, holding the persistable
  state Cards 5/6 store beside the checkpoint, and exposing the offline expiry API (surface
  below). Dependency-injected like its siblings — zero imports; the same file runs under the
  Node harness and the browser bundle.
- `marketing/sync/pull-replication.js` (EDIT — Card 2's module) — **one function touched:
  `makePullHandler`** gains an optional `clock` parameter; after a 200 pull response the handler
  calls `clock.captureFromResponse(res)` before parsing the body ("on every successful pull",
  the slate's words). **What must survive:** everything Card 2's intent froze — the exported API
  names, the keyset checkpoint shape `{updated_at, id}`, the compound OR predicate, the
  URL-encoding build-fact, `wireRealtimeResync`'s §7.3 semantics. The `clock` parameter is
  additive and optional: a caller passing no clock (Card 2's harness, Card 3's harness) gets
  byte-identical behavior. No other function in the file changes.
- `marketing/sync/replicas.js` (EDIT — Card 2's module) — **one function touched:
  `startReplica`** (the private helper both `startCodesReplica` and `startOffersReplica`
  call) accepts `clock` in deps, threads it into `makePullHandler`, and defaults the
  window-bound `now` to `clock.now` when no explicit `now` is injected — the file's own
  comment already named `now` as "the injection point for the clock-offset card (§5.1)".
  **What must survive:** the exported API Card 2 froze (`startCodesReplica` /
  `startOffersReplica` / `resolveOffers` / `marketingCollectionSpec` / schemas / window
  constants). `resolveOffers`' signature is unchanged (`{now}` stays; Cards 5/6 pass
  `clock.now`). Explicit `now` still beats the clock (harness injection precedence).
- `marketing/sync/harness/clock-run.sh` + `marketing/sync/harness/clock-harness.mjs` (NEW) —
  this card's standalone gate (B-345 precedent), a SIBLING of Card 2's `run.sh`/`harness.mjs`
  and Card 3's `push-run.sh`/`push-harness.mjs`, deliberately not an edit to them: the landed
  gates stay byte-identical and re-runnable. Reuses the family scaffolding (lib.sh substrate
  discipline, JWT minting, the node_modules symlink, red/green mode contract) and drives
  Card 2's PRODUCTION replicas with the new clock injected.
- `marketing/sync/push-replication.js` — **zero edits.** Audited for expiry decisions per the
  slate ("Card 3's expiry decisions, if any, also adjust"): the push module makes NO local
  `expires_at` comparison anywhere — expiry verdicts on the push path come from the server-side
  `redeem()` (`expires_at > now()` in SQL, server clock). Its only clock uses are the
  `scanned_at` stamp and a local wait deadline, both already behind the existing `now`
  injection — Cards 5/6 may pass `now: clock.now` into `enqueueAttempt`'s opts for an
  offset-true `scanned_at` with zero edits here. Nothing to adjust.
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the `clock-offset-on-sync` card
  (PLANNED → DRAFTING (overnight-20260905)), the run's flip convention. No other line moves.
- `.night-crew/runs/2026-09-05-autonomous/merge-intents/clock-offset-on-sync.md` — this file
  (amended as evidence lands).
- `.night-crew/runs/2026-09-05-autonomous/card4-*.log` — committed whole gate logs, each ending
  in its `EXIT=` line.
- `marketing.html`, `index.html`, `sw.js`, `tests/*.spec.js`, `backend/`, `supabase/`,
  `night-crew.toml`, `package.json`, `vendor/` — **nothing here.** The clock module is UNWIRED
  tonight (Cards 5/6 wire and state their precache moves); no vendor regen (the module needs no
  new bundle export — it is plain arithmetic the pages import alongside the sync modules).

## The API Cards 5/6 call — "is this code expired, offline"

`createSyncClock({deviceNow?, initialState?, persist?})` from `marketing/sync/clock.js`
returns the device's one clock instance:

- **`clock.isExpired(expiresAt, at?)`** — THE offline expiry check. True when
  `deviceNow() + offset >= Date.parse(expiresAt)` (§5.1's comparison, exact). Every place the
  scanner asks "is this code dead?" calls this — never a raw `Date.now()` comparison.
- **`clock.now()`** — server-estimated now (`deviceNow() + offset`), for anything else
  time-shaped (window bounds follow it automatically once the clock is passed to
  `startCodesReplica`/`startOffersReplica`; `resolveOffers(col, hash, {now: clock.now})`).
- **`clock.captureFromResponse(response)`** — called by `makePullHandler` on every successful
  pull; Cards 5/6 never call it directly, they just pass `clock` into the replica deps.
- **`clock.state()`** — the persistable `{offset_ms, captured_at_device, captured_at_server}`;
  `persist(state)` fires on every capture — Cards 5/6 store it beside the checkpoint (their
  storage, injected) and hand it back as `initialState` on boot so a reloaded-offline device
  keeps its calibration.
- Sign convention (spike + extraction record, binding): **`offset = serverNow − deviceNow`**.
- Update cadence: **every successful pull response; latest capture wins.** Pure arithmetic —
  no smoothing, no grace window, no thresholds (anything of that shape is a policy call and
  was not made — none turned out to be needed).

## What must survive any merge

- `marketing/sync/clock.js` and its exported surface above — Cards 5/6 wire exactly this.
- The `clock` parameter threading in `makePullHandler` and `startReplica`, and its OPTIONAL
  nature — Cards 2/3's harnesses run clock-less and must stay green unchanged.
- Card 2's frozen surfaces (keyset checkpoint shape, predicate, exports) and Card 3's frozen
  push surface — this card touches neither's semantics.
- The roadmap flip; the `card4-*.log` evidence; this intent.

## What is safe to drop

- `marketing/sync/harness/node_modules` symlink (runtime-created, gitignored).
- Nothing else in this branch is scratch.

## Red-first

To be captured to `card4-red.log` BEFORE the production clock lands (git history is the
chronology, the family pattern). The probe is the naive comparison (`deviceNow < expires_at`,
no offset), inline in the harness, never the production code. **A direction reconciliation,
stated up front because the slate's wording and the arithmetic disagree on a label:**

- A dead code is naively ACCEPTED only when `deviceNow < expires_at < serverNow` — i.e. the
  device clock reads EARLIER than the server (set back / behind; the spike's proven
  "2-days-slow" leg, §5.1's dangerous direction: rolled-back clock resurrects dead codes).
- A clock 2 days AHEAD (a "fast" watch) cannot make the naive check accept a dead code; its
  naive failure mode is the mirror image — valid codes falsely REJECTED.
- The slate/E-KR4 phrase "device clock ≥2 days fast" is therefore read as "wrong by ≥2 days",
  and the harness exercises BOTH signs of skew: red shows the naive check accepting the dead
  code under the behind-skew AND falsely rejecting a valid code under the ahead-skew; green
  shows the offset-adjusted check correct under BOTH skews — the done_when's literal text
  (clock ≥2 days fast → expired code still rejected offline, valid code still resolves) is
  asserted directly, and the sign convention is pinned from both sides. This is an arithmetic
  reading, not a policy change — no decisions-log entry owed.

## Known residual (documented, no machinery added)

The FIRST pull of a fresh device runs before any capture exists, so its window bound derives
from the unadjusted device clock; a row that first window filters out, whose keyset position
the checkpoint then advances past, stays absent until its `updated_at` next bumps. Strictly
better than the status quo (without the clock, EVERY pull's window is wrong forever), and
self-healing on any row update. Mitigations (one-shot RESYNC after first capture, checkpoint
reset on large offset change) are Cards 5/6 wiring choices or policy calls — deliberately not
invented here.

## Gate evidence

(amended when gates run — logs land beside this file)
