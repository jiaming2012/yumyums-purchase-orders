# Merge intent — card `scan-time-verify` (Card 2, roadmap I2, run `20261003`)

Branch `card/i2-scan-time-verify`, cut from Card 1's **unmerged** branch tip `73521b1`
(`card/i1-test-integrity-fix`). Track A, second. BACKLOG **B-468** (an online phone never
verifies a code it has not synced); ledger T-62 decision 200 (one query on the connection the
device already has, no new backend surface, today's behaviour on timeout).

Contract: `.night-crew/knowledge/reference/slate-20261003.md` §"Card 2". Goal ledger:
`.night-crew/knowledge/spikes/activity-i-honest-gates-then-verify-before-the-till-triage-20261002-follow-ups/scan-time-verify.md`
(1/1 passed; build-facts binding).

**What a person sees differ.** An ONLINE phone that scans a code it has never synced now asks
the server about it before anything else is drawn: the real offer card appears (or "Already
used"), and only if the server cannot be reached inside 3.5 s does the phone show today's
"Code not recognized" card with one extra line saying it couldn't check the server. An OFFLINE
phone does exactly what it did yesterday and makes no network call at scan.

🛑 **Merge rule (slate, run 20261002's lesson):** this branch sits on an unmerged base. Never
squash it; 3-way merge and assert the artefacts afterwards. No Card 1 commit is modified,
reverted or amended here.

---

## Shared files touched, and why each

| File | In footprint? | Why |
|---|---|---|
| `marketing/scanner.js` | yes (solo owner) | `createScanResolver` gains the optional `serverLookup` dep; `resolve()` step 3 calls it only when `online` and the token is in neither replica, before the embedded-offer fallback. New pure, dependency-injected `createServerLookup({restUrl, bearer, fetchImpl, timeoutMs})` (the GET, the abort, the timeout race). |
| `marketing/scan-page.js` | yes (solo owner) | Wires the lookup from the provisioned coordinates inside `startSync` (same `restUrl`, same `bearer`, same `fetchImpl` the pull handlers get; timeout = `PROBE_TIMEOUT_MS`, imported from `submit-support.js`). Renders the "couldn't check the server" line, the server-sourced "Already used" copy, and `data-verified`. Closes the machine session for a server-redeemed code through the EXISTING `onScanAgain` registration hook (the declared `resolving` + `NEXT_CUSTOMER` pair — no new pair). |
| `marketing/submit-flow.js` | yes (copy) — **NOT TOUCHED** | The slate allowed copy here; the fallback line renders in the result card (`scan-page.js`) instead, so this card's diff against Card 1's file is empty. No predicate body, no stash field, no mapping changed. |
| `tests/marketing.spec.js` | yes | New `describe` appended: `[SV-01]`–`[SV-04]` plus `[SV-03b]` (hung link). One edit to an existing helper: `mockSyncTransports` now honours a `token_hash=eq.` filter on `GET /codes` the way PostgREST does (before this card nothing sent that query; without the edit the mock would answer every lookup with the whole pull batch). Card 1's `[TI-01]`/`[TI-02]` untouched — union at merge. |
| `tests/fixtures/sv04-offline-unknown-code.html` (**new**) | **no — new fixture beside the spec** | The `#scan-result` markup an offline never-seen code renders, captured from the PRE-change tree (`73521b1`). `[SV-04]` compares against it byte-for-byte; that comparison is what "byte-identical to today" means. Not precached (`tests/` never is). |
| `sw.js` | yes | Regenerated after the precached edits are committed. `scanner.js` + `scan-page.js` change revision; no file added or removed — **count stays 51**. The orchestrator regenerates at the merged HEAD; never hand-merge. |
| `.night-crew/knowledge/BACKLOG.md`, `roadmap.md` | yes (bookkeeping) | B-468 → `landed → scan-time-verify`; roadmap I2 `PLANNED` → `LANDED`. |
| `.night-crew/runs/2026-10-03-autonomous/logs/scan-time-verify/` | evidence | Red-first, mutation and gate logs. |

`night-crew.toml`: **not touched.** `backend/`: **no file touched** (no scope drift to state).
`marketing/submit-machine.js`: **not touched** — 460 declared pairs before and after.

## Per done_when clause — what rides a stub, what does not

Every test provisions through the SHIPPED path. The Playwright stack has no sync substrate
(the webServer carries no `HQ_SYNC_JWT_SECRET`, so the real `POST /api/v1/sync/token` answers
503 and the real door has nothing behind it), so the existing `mockSyncTransports` helper
serves the mint envelope and the three pull replicas at the network layer — the same
arrangement Card `sync-coordinates-provisioning` landed with. **That is the stack, not the
feature under test**; the resolver, the lookup, the fetch, the abort, the render and the
submit machine all run for real in every clause.

| Clause | The server's answer to the lookup | Stub? |
|---|---|---|
| `[SV-01]` live row | `page.route` on the door's path (`/sync/rest/codes?token_hash=eq.…`) fulfils one PostgREST-shaped row | **STUB — the one the slate permits** (the row). The request itself is asserted: exact query string and the bearer header. |
| `[SV-02]` redeemed row | same route, a row with `redeemed_at` / `redeemed_by` | **STUB — the permitted one** (the row). |
| `[SV-03]` server killed | every `/sync/rest/**` request is `route.abort('connectionrefused')` — the browser's `fetch` rejects with a network error | **UN-STUBBED.** No handler returns a status or a body; nothing answers. The test also asserts the lookup request was actually issued and failed (`requestfailed`), and the wall time. |
| `[SV-03b]` hung link (extra) | the route never answers until the test ends | **UN-STUBBED** — a hang, not a response. Proves the `AbortController` + timeout: the scan resolves at ~3.5 s, not never. |
| `[SV-04]` offline, zero calls | a counting route on the door's lookup path that would serve a LIVE row if it were ever reached, plus a `page.on('request')` log of every request in the scan window | **UN-STUBBED count.** The count is 0 because nothing was sent, not because something was intercepted and discarded; the would-be row makes a wrongly-sent call visible as `offerReady`. The same test then puts the phone online and requires the counter to read 1 — the control that proves the counter can see a call (and the reason the test is RED on the pre-change tree). |

## What must survive any merge

1. 🛑 **The `online &&` guard on the lookup in `scanner.js` `resolve()` step 3.** Dropping it
   makes an offline phone issue a request at scan — a change to what an offline phone does
   (decisions 166 / 199), and `[SV-04]` reds.
2. **The lookup runs only when the token is in NEITHER replica** (after steps 1 and 2 return
   nothing). A code the device holds keeps today's replica verdicts, F3's `deferToServer`
   included.
3. **Server row beats the embedded descriptor** (step 3 order: server, then embedded, then
   unknown) — an unauthenticated descriptor never outranks the server.
4. **The timeout is the probe's (`PROBE_TIMEOUT_MS`) and it is a race, not only an abort** —
   a `fetchImpl` that ignores `signal`, or a body that never finishes, still rejects on time.
5. **A device with no provisioned coordinates has no `serverLookup` at all** (the resolver is
   rebuilt with the dep inside `startSync`). Unprovisioned-online therefore stays today's
   `unknownCode` with no `verified` key — which is every existing online test in the suite.
6. Card 1's exports and `MARKETING_REPLICA_SCHEMA` v1 — untouched here; keep Card 1's side.

## What is safe to drop

* `[SV-03b]` (the hung-link extra) if it ever proves slow on the box — `[SV-03]` is the
  done_when clause. Dropping it loses the only test of the abort path.
* The `mockSyncTransports` `token_hash` filter, if a later card replaces the helper.
* `sw.js` from this branch — regenerate at the merged HEAD.

## Red-first

Filled in below as the evidence lands (logs under
`.night-crew/runs/2026-10-03-autonomous/logs/scan-time-verify/`).
