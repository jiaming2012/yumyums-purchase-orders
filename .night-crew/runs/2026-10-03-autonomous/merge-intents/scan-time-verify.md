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

* ~~`[SV-03b]`~~ — **NOT safe to drop (corrected in the fix round).** It is the timeout
  budget's ONLY guard: with the race and the abort removed from `createServerLookup`, every
  other `[SV-*]` test stays green and only `[SV-03b]` reds
  (`fixround-red-sv03b-mutation-no-timeout.log`: `1 failed`, `8 passed`). `[SV-03]` kills the
  link, which fails fast on its own and never exercises the timeout. Must survive any merge.
* The `mockSyncTransports` `token_hash` filter, if a later card replaces the helper.
* `sw.js` from this branch — regenerate at the merged HEAD.

## Red-first

Logs under `.night-crew/runs/2026-10-03-autonomous/logs/scan-time-verify/`. The stack was
hand-provisioned on `:8212` (`hq_test_e2e_i2_20261003` on `:5434`) with `NIGHTCREW_ENV_URL`.

| Leg | Tree | Log | Exit line |
|---|---|---|---|
| RED, all six | PRE-change (`ff61ec1` = `73521b1` + this file; `marketing/` diff vs `73521b1` empty) | `red-pw-sv-prechange.log` | `6 failed` · `EXIT=1` |
| GREEN, all six | POST-change working tree (committed unchanged as `0d45d50` + `49b434b`) | `green-pw-sv-postchange.log` | `6 passed` · `EXIT=0` |
| GREEN, whole `tests/marketing.spec.js` | same | `green-pw-marketing-spec-postchange.log` | `60 passed` · `EXIT=0` |
| MUTATION RED, `[SV-04]` | POST-change with `online &&` deleted from `scanner.js` step 3 | `red-pw-sv04-mutation-online-guard.log` | `1 failed` · `EXIT=1` |
| fixture capture | PRE-change | `fixture-capture-prechange.log` | `EXIT=0`, sha256 `bfb6bb7c…2040c`, 972 bytes |

Where each test reds on the pre-change tree, and why that is the feature's absence:

* `[SV-01]` — the resolver answers `{kind:'unknownCode'}` where the test requires the
  server-sourced `offerReady` object (first assertion after the scan).
* `[SV-01b]` — the lookup counter reads 0: no request was sent.
* `[SV-02]` — `{kind:'unknownCode'}` where the test requires the server-sourced "already used".
* `[SV-03]` — the result has no `verified: false` (the phone never tried, so it cannot say it
  failed).
* `[SV-03b]` — "the lookup was sent": 0 requests held by the hanging route.
* `[SV-04]` — 🛑 the OFFLINE half passes on the pre-change tree, as it must (zero calls, exact
  object, markup equal to the capture — that is what "byte-identical to today" means); the test
  reds at its ONLINE control, "the same phone asks the server exactly once" — 0. So that
  `[SV-04]` is not green merely because nothing ever calls the server, the mutation leg deletes
  the `online &&` guard on the post-change tree: the offline scan then sends the lookup, gets
  the would-be live row, and the test reds at `data-mstate` (`offerReady`, expected
  `unknownCode`).

## What `failClosed` / `policyFor` see for a `source: 'server'` offer

The server row is shaped by the SAME `offerShape` the codes-replica rows go through:
`{code_id: row.id, campaign_id: row.campaign_id || null, expires_at}`. `submit-flow.js`
`onResult` handles it in its existing `offerReady` arm, untouched:
`policyFor(CAMPAIGN_POLICY, o.campaign_id || null, result.offers)` and
`policyUnresolvedFor(o.campaign_id || null)`, `stash.code_id = o.code_id`.

* **The row carries `campaign_id`** — the lookup's select names it, `[SV-01]` asserts the exact
  resolver object, and `public.codes.campaign_id` is what the spike read back populated. This
  path does not go through RxDB at all, so Card 1's "the phone has no validator wrapped" fact
  does not bite here: nothing is inserted into a collection.
* **Campaign replicated** (the `[SV-01]` case, LOW campaign): `policyFor` → the replica's
  `requires_online`; the online submit posts the hash and redeems (`[SV-01]` drives it to
  `redeemed` and asserts the posted body).
* **Campaign NOT yet replicated** (plausible for exactly this card's customer — a campaign
  minted since the last sync): the policy source answers unresolved for a KNOWN id, so
  `requiresOnline` is `true` and `policy_unresolved` is `true` — the B-432 fail-closed arm, by
  the unchanged predicates. Online, that changes nothing (the submit goes to the server). If
  the link drops before submit, the crew see the existing `requires-online-unresolved` refusal
  and no override. Read from the code; **not driven by a test in this card.**
* **A server row with a null `campaign_id`** (not expected — stated for completeness):
  `namesNoCampaign` → `policyFor` false → the decision-166 override stays available offline,
  with `unverified_code: false` because the machine kind is `offerReady`.

No predicate body, no submit-time behaviour and no machine pair needed changing to make the
server-sourced offer submit. No PARK condition was hit.

## Engineer-level calls made here (the slate left them to the night)

* **Timeout:** `PROBE_TIMEOUT_MS` (3500 ms), imported — one number for "how long before this
  link is dead".
* **Copy:** "Couldn't check the server just now — treat this code as unverified until the
  submit goes through." (one extra `.result-note`, `#scan-server-unchecked`, under today's
  unchanged sentence); server-redeemed: "The server shows this code already redeemed — don't
  apply the discount."
* **Result shape:** `verified: false` is ADDED only on the could-not-ask path; every other
  result keeps the exact keys it had. The box gets `data-verified="false"` in that case only.
* **Redeemed server row → kind `spentLocally` + `source: 'server'`** (the existing "Already
  used" card, which has no submit slot), and `scan-page.js` closes the machine session via the
  existing `onScanAgain` hook instead of sending `RESOLVED` — otherwise the machine's F3-online
  arm would open a submit session for a code the server just refused, and the next customer's
  scan would hit "finish the current customer first".
* **Expired server row → kind `expiredLocally` + `source: 'server'`** (the slate names three
  outcomes; an unredeemed-but-expired row is a fourth the spike showed the device can read).
  Existing card, existing machine pair. Not covered by a named test.
* **Could-not-ask with an embedded descriptor** falls back to today's `embeddedOffer` (not
  `unknownCode`) with `verified: false` and the same extra line — "today's behaviour" for that
  payload is the embedded card.
* **No coordinates → no lookup.** An unprovisioned phone that is online resolves as before,
  without the "couldn't check" line. It is true that the server was not checked; the phone has
  no door to check through, and today's sentence already says "can't verify … on this device".
* **The door on the e2e stack does not serve this read** (no substrate; the real
  `/api/v1/sync/token` is 503 there) — so the row is stubbed, as the slate anticipated. No
  route was built.

## Gates (tree `ebcb384` — every code, test and `sw.js` commit; later commits are docs/logs only)

| Gate | Log | Result |
|---|---|---|
| G1 | `g1.log` | `EXIT_BUILD=0`, `EXIT_VET=0`; `backend/` diff vs `73521b1` empty |
| G2-Go (`-p 1`, under the lock) | `g2-go.log` | `EXIT_TEST=0` — 15 packages `ok`, 447 top-level tests: 444 pass / 0 fail / 3 skip (same counts as tonight's base). `TestRowVisibilityRLS` PASS |
| G2-Playwright FULL (under the lock, hand-provisioned `:8212`, `--retries=0`) | `g2-pw-full.log`, `g2-pw-full.reds.txt` | `EXIT=1` — 27 failed / 7 skipped / 1035 passed (34.6m), 1069 tests. All 60 `tests/marketing.spec.js` tests green, the six `[SV-*]` included |
| G4 | `g4-sw.log` | 51 precached (unchanged); regenerating at the committed HEAD reproduces the committed file (`EXIT_GITDIFF_SW=0`); only `marketing/scanner.js` and `marketing/scan-page.js` changed revision |

Reds vs tonight's base (`logs/base-reds.txt`, 25): 24 of the 27 are in the base set. Base red
but green here: `inventory.spec.js:1469`. The 3 outside the base — `inventory.spec.js:2931`,
`onboarding.spec.js:2233` (both also seen by Card 1's run on this base) and
`recipes.spec.js:216` (new to tonight's lists) — were each run 3× in isolation on BOTH trees
(`iso-extra-reds.log`; the pre-change leg swaps in `marketing/` + `sw.js` from `73521b1`):

| Test | POST-change red | PRE-change red |
|---|---|---|
| `onboarding.spec.js:2233` | 3 / 3 | 3 / 3 |
| `inventory.spec.js:2931` | 1 / 3 | 1 / 3 |
| `recipes.spec.js:216` | 0 / 3 | 1 / 3 |

None of the three loads a file this card changed, and each behaves the same on both trees —
not attributed to this card. (The isolation ran against the database the full suite had just
used, so it measures "same on both trees", not "green on a clean database".)

The suite rewrote tracked PNGs under `.night-crew/runs/2026-10-02-autonomous/`; they were
restored with `git checkout` and none is committed.

## Fix round (G6 APPROVE-WITH-FINDINGS on `424b2e5`; no commit rebased or amended)

Logs: `logs/scan-time-verify/fixround-*.log`. Gates this round are CONFINED by the
orchestrator's instruction (whole `tests/marketing.spec.js`, then `sw.js`); the full Go and
Playwright suites were NOT re-run on this tree.

| Finding | Change | RED | GREEN |
|---|---|---|---|
| **P2-1** "only when in neither replica" had no test | `[SV-05]`: online phone, fixture 1 held live and fixture 4 held redeemed-locally → 0 requests on the lookup path, `deferToServer` unchanged, no checking card ever rendered (a `MutationObserver` records every `data-kind`); then a never-seen code → exactly 1 (control). Test only — the guard already held. | under the reviewer's mutation (an extra `serverLookup` ahead of step 1 when online): `fixround-red-sv05-mutation-lookup-when-held.log` — `1 failed`, `EXIT=1` | `fixround-green-postfix.log` — `3 passed`, `EXIT=0` (also green on `424b2e5`, as expected: `fixround-red-prefix.log`) |
| **P2-2** result area blank during the lookup | `resolve()` takes an optional `onServerLookup(token_hash)` observer, called only when the step-3 lookup is about to be issued. `scan-page.js` renders `{kind: 'checkingServer'}` — "Checking with the server…" — in the existing `#scan-result` (`#scan-server-checking`; no slot, no control), replaced by the outcome. A render-only pseudo-kind: it never reaches `submitFlow.onResult`, so no machine state, event or pair. `[SV-06]` holds the lookup route, asserts the text during the wait and its absence after. | on `424b2e5`: `fixround-red-prefix.log` — `[SV-06]` ✘ (`2 failed`, `1 passed`, `EXIT=1`) | `fixround-green-postfix.log` |
| **P3** unusable first element | `createServerLookup` rejects unless the first element is an object with a non-empty string `id` → `unknownCode` + `verified: false` + the note. `[SV-07]` drives `[{}]`, `[null]`, `["str"]`, `[{"id":""}]`. | on `424b2e5`: `fixround-red-prefix.log` — `[SV-07]` ✘ (`[{}]` rendered an Expired card) | `fixround-green-postfix.log` |
| **P3** `[SV-03b]` "safe to drop" | Struck above; mutation evidence `fixround-red-sv03b-mutation-no-timeout.log`. | — | — |

Confined gate: `fixround-green-marketing-spec.log` — `63 passed (1.2m)`, `EXIT=0`, unmutated.
`[SV-04]` green: the offline card still equals `tests/fixtures/sv04-offline-unknown-code.html`
byte-for-byte (fixture untouched; the checking state is not rendered offline). `sw.js`:
`fixround-g4-sw.log` — 51 precached, regeneration at the committed HEAD reproduces the file.

Added to "what must survive any merge": the `onServerLookup` call sits INSIDE the
`online && serverLookup` branch of step 3 — moved above it, an offline or held-code scan would
flash the checking card (`[SV-05]` and `[SV-04]` red).

Left for the morning notes, by instruction: a direct retry on the could-not-check card; the
`scanText` duplicate-lookup hook.
