# Merge intent — Card H5 · `subscribers-tab`

Run `20261002` · branch `card/h5-subscribers-tab` off `overnight-20261002` at `94ad3b3`
Track E (full-stack). Base already carries Card H1's merge (`40b5846`), so
`qr_codes(short)` exists and this card's `0085` FK can reference it.

## Shared files touched

| File | Why |
|---|---|
| `backend/internal/marketing/routes.go` | **APPEND-ONLY.** One labelled block at the end of `Mount`, registering five `/subscribers*` routes. Nothing above it is edited — not `Deps`, not `MountReports`, not `MountPublic`, not `requireManager`. Cards 3 and 4 append their own blocks after mine; a conflict here is two additions and both sides survive. |
| `backend/go.mod` / `backend/go.sum` | `+ github.com/go-sql-driver/mysql v1.9.3` and its one indirect, `filippo.io/edwards25519 v1.2.0`. The Fluent Forms reader speaks to a WordPress MySQL. **Card 3 is editing these files concurrently — union merge, keep BOTH requires.** |
| `marketing.html` | **Section `#s3` ONLY**, plus one `<script type="module" src="marketing/subscribers.js">` line beside the two existing module entry points. See "What must survive" below — this is the field the merge turns on tonight. |
| `night-crew.toml` | **A ROLL-CALL COMMENT ONLY.** No new key, no new token (both would PARK). The `marketing` prefix key already covers `marketing.html` + `marketing/`, and the `marketing` TOKEN already selects any spec whose filename contains it, so `tests/marketing-subscribers.spec.js` and `tests/states-marketing-subscribers.spec.js` join the seam automatically. The comment records that they did. |
| `sw.js` | Regenerated after the commit (reads git HEAD, B-13). Precache **48 → 49**, the one new module `marketing/subscribers.js`. No `build-sw.js` edit: the existing `marketing/*.js` glob already matches it. |
| `.night-crew/runs/2026-10-02-autonomous/merge-intents/h5-subscribers-tab.md` | This note. |
| `.night-crew/runs/2026-10-02-autonomous/logs/h5/*` | This card's gate logs. Nothing else under the run directory is touched — `timings.log` and `DECISIONS-NEEDED.md` are the orchestrator's. |

**Nothing else.** No `backend/cmd/server/main.go` (H1's three call sites are
untouched — this card's routes go into `Mount`, which is already mounted), no
`backend/internal/db/db.go` (no new grant), no `build-sw.js`, no other HTML page,
no existing spec file, no `marketing/sync/*`, no `docs/`.

## What must survive any merge

1. **🛑 THIS CARD OWNS `marketing.html` SECTION `#s3` AND ONLY `#s3`.**
   Cards 2 and 5 own `#s2` (Campaigns) and `#s4` (Stats queue) of the same file
   tonight and **must not lose theirs**. Verified mechanically before the final
   commit: `git diff -U0 marketing.html` is three hunks — two inside the `#s3`
   block and one two-line insertion at the module-script list. The shared
   `<head>` style block, the `.tabs` bar, `show()`, the auth probe, the SW
   registration and sections `#s1`/`#s2`/`#s4` are **byte-identical** to
   `94ad3b3`.
   The section's CSS lives in a `<style>` element **inside `#s3`** on purpose —
   a style element applies document-wide wherever it sits, and putting it there
   keeps this card's whole `marketing.html` diff inside its own section instead
   of in the head, where all three cards would collide.
2. **The append-only block in `routes.go`.** Five routes, inside `Mount`:
   `GET /subscribers`, `GET /subscribers/{id}`, `POST /subscribers/{id}/resend`,
   `POST /subscribers/import/toast-guests`, `POST /subscribers/import/web-form`.
   Package-level identifiers carry this card's `Subs…`-or-domain prefix or are
   unmistakably subscriber-named (`ListSubscribersHandler`,
   `GetSubscriberHandler`, `ResendQRHandler`, `ImportToastGuestsHandler`,
   `ImportWebFormHandler`, `ImportSubscribers`, `SubsFixtureEnv`,
   `ConsentSMS/EmailOnly/Pending/Stop`, `FilterAll/SMS/EmailOnly/OptedOut`), so
   no two cards declare one name.
3. **Migration `0085_subscribers.sql`.** §4's H5 DDL verbatim. It **FKs
   `qr_codes(short)` from Card 1's `0083`**, so **0085 must sort after 0083**.
   `0084` is Card 3's. If a renumber is ever needed, 0085 may move UP but never
   below 0083.
4. **`backend/internal/marketing/sources/`** — a new sub-package, four source
   files plus `testdata/`. It imports nothing from `internal/marketing`, so the
   dependency runs one way and no card's edit to the parent package can break it.
5. **The committed fixture** `backend/internal/marketing/sources/testdata/fluentforms_submissions.json`
   and its `README.md`. This is what every automated gate reads instead of the
   live website database. Dropping it strands five tests.
6. **The two prohibition guards** — `TestNothingInThisPackageSends` (parses the
   import graph of `internal/marketing` + `…/sources` for senders, and the
   subscriber files for outbound HTTP) and `TestWebFormImportRefusesWhenUnconfigured`.
   They are the mechanical form of this card's two absolute prohibitions.
7. **The masking contract** — `subscriberRowDTO` has no `phone_e164` and no
   `email` field, and `TestNoResponseCarriesAFullPhoneOrEmail` plus `[SB-01]`
   assert the absence on the wire, on the cell and on the whole page HTML.
   Adding either field back is a privacy regression, not a convenience.

## What is safe to drop

- **The `source=` `<select>` and the `q=` suffix search** are this card's
  reading of §5's query parameters; if a later card restyles the control row,
  the later card's chrome wins as long as `filter` / `source` / `q` still reach
  the endpoint.
- **`TestE164NormalizeAndMaskHoldOnTheSpikesSet`** is a re-assertion in Go of
  spike 02's shell script. If it ever conflicts with a refactor of
  `sources.NormalizeE164`, the refactor's version of the test wins — the
  build-fact is the spike's, not this file's.
- **The `offers_now[]` query** (every campaign live right now) and the
  `identity_code` derivation (read off the timeline, because Activity E has not
  built the code yet) are both placeholders by construction; Activity E replaces
  them outright.
- **`importResultDTO`'s field set.** Nothing in §5 specifies it; it is a count
  summary for the attended import and a later card may reshape it.
- **The `subscribers_joined_at_idx` / `subscribers_source_short_idx` /
  `subscriber_events_subscriber_idx` indexes** are not in §4 — they are read-path
  additions. Dropping one costs latency, not correctness.

## Red-first

Observed on the **pre-change tree** by moving this card's implementation out of
the worktree (migration `0085` absent, `subscribers.go` absent,
`sources/*.go` absent, `routes.go` reverted to H1's HEAD, `go.mod`/`go.sum`
reverted) while leaving the tests in place.

### Go leg — `logs/h5/rf-go-red.log`, `EXIT=1`

```
github.com/yumyums/hq/internal/marketing/sources: no non-test Go files in
  …/backend/internal/marketing/sources
FAIL	github.com/yumyums/hq/internal/marketing [build failed]
internal/marketing/sources/sources_test.go:24:13: undefined: NewFluentFormsFromJSON
internal/marketing/sources/sources_test.go:28:20: undefined: SourceWebForm
internal/marketing/sources/sources_test.go:38:19: undefined: Candidate
internal/marketing/sources/sources_test.go:85:15: undefined: NewFluentForms
FAIL	github.com/yumyums/hq/internal/marketing/sources [build failed]
FAIL
EXIT=1
```

Both named `done_when` Go tests — `TestFluentFormsImportIsIdempotent` and
`TestSubscriberSourceShortSetsCampaignAttribution` — were in the `-run` filter
and could not compile, let alone run. Greenfield red: the behaviour does not
exist.

### Playwright leg — `logs/h5/rf-pw-red.log`, `EXIT=1` · **6 failed, 0 passed**

```
✘ [SB-01] list masks phone to last 4 …          Error: import/toast-guests → 404 page not found
✘ [SB-02] STOP renders an opted-out row …       Error: import/toast-guests → 404 page not found
✘ [SB-03] the sheet shows identity-code status… Error: import/toast-guests → 404 page not found
✘ [SB-04] Resend records an event and sends …   Error: import/toast-guests → 404 page not found
✘ a team_member sees the Locked state …         expect(received).toBe(expected)   // 404 ≠ 403
✘ the empty state names what is missing …       TimeoutError: page.waitForFunction: Timeout 15000ms exceeded
EXIT=1
```

`marketing.html` `#s3` was still the "Soon" placeholder, so `#subs-root` never
appeared (the `waitForFunction` red), and every `/subscribers*` route 404'd.

## Which State-Enumeration rows ride a FIXTURE and which hit the REAL endpoint

`tests/states-marketing-subscribers.spec.js`. **Three fixture rows, six real.**

| Row | Trigger | Fixture or real |
|---|---|---|
| loading | `page.route` holds `GET /subscribers` 1.5 s, then the §5 zero shape | **FIXTURE** |
| empty | `page.route` fulfils the §5 zero shape `{total:0,…,rows:[]}` | **FIXTURE** |
| error | `page.route` fulfils `500 {"error":"boom"}`, then succeeds on Retry | **FIXTURE** |
| success | REAL `POST /subscribers/import/toast-guests` → REAL `GET /subscribers` | real |
| long content | REAL import of a 70-char display name + a long email | real |
| detail sheet | REAL `GET /subscribers/{id}` | real |
| empty-filtered | REAL read with a `q` nothing matches | real |
| locked (403) | REAL `team_member` → REAL `403 {"error":"managers_only"}` | real |
| offline | `context.setOffline(true)` after a REAL load — **no response fixture at all**: `navigator.onLine` is genuinely false and `fetch` genuinely fails | real condition |

The three fixtures are fixtures because the condition cannot be produced against
a live server inside a shared E2E database: a deliberately slow server, a
deliberate 500, and a *globally* empty mailing list in a database the sibling
spec has already seeded. `tests/marketing-subscribers.spec.js` — where
`[SB-01]`–`[SB-04]` live — contains **no `page.route` at all**; every one of
those four hits the real endpoints.

## Decisions this card made that the slate left to it

The slate's PARK note: *"The mask format, the filter set and the timeline
rendering are the night's."*

- **Mask format.** Phone: `•••• 4821` — four bullets, one space, the last four
  digits (what spike 02 printed; never implies a digit count the number does not
  have). Email: `d•••@example.com` — first character of the local part, three
  bullets, the domain **verbatim**, because "is this the gmail one or the work
  one?" is the question the cell is asked and a domain is not a contact handle
  on its own; a one-character local part masks to `•••@domain`. Both computed
  **server-side**; `null` on the wire when there is nothing to mask, which the
  UI renders as "No phone".
- **Filter set.** `all` / `sms` (`sms_consent AND NOT opted out` — who an SMS
  blast may reach) / `email_only` (`email_consent AND NOT sms_consent AND NOT
  opted out` — the people an SMS campaign silently misses) / `opted_out`
  (`opted_out_at IS NOT NULL` — the suppression list, visible precisely because
  it must never be messaged). `opted_out_at` **wins** over both flags when
  deriving §5's `consent` value. Unrecognised `filter` falls back to `all` on a
  read rather than 400ing.
- **Timeline rendering.** One line per `subscriber_events` row, newest first,
  plain-English label + date (`signed_up` → "Signed up", `resend_requested` →
  "Resend requested", …). An **unknown kind renders its raw value** rather than
  vanishing, so a future Activity-E event is visible the day it starts being
  written.
- **`identity_code` status** (not named in the PARK note, so decided here):
  Activity E owns the identity code and no table holds one, so the status is
  **derived from the timeline** — `code_sent` present → `sent`, else
  `not_sent`, with the latest `resend_requested` carried beside it as
  `requested_at`. The sheet therefore says "Resend requested · <date>" without
  ever implying a send.
- **`offers_now[]`**: every `campaigns_admin` row with `status='live'` and now
  between `starts_at`/`ends_at`. Nothing in §4 scopes an offer to a person, so
  the sheet answers "what can I honor if they walk in".
- **`visits`**: `count(subscriber_events WHERE kind IN ('scanned','redeemed'))`.
  Honestly 0 tonight — `redeemed` needs Card 3's `scan_attempts_mirror`.
- **An unknown `source_short`** keeps the person and drops the attribution
  (`source_short` lands NULL) instead of failing the import on the FK. A printed
  code that no longer exists is the customer's reality, not an error.
- **`POST /subscribers/import/web-form` exists as a route** even though §5 does
  not list one, because "the live import is ATTENDED" needs something for the
  operator to call. It is manager-only and answers `503 ff_not_configured`
  whenever `FF_DB_*` are unset.

## Nothing was parked

The two prohibitions were honoured rather than parked: **nothing sends**
(`/resend` is 202 + an event + no outbound anything) and **no live Fluent Forms
connection was made** (the committed fixture is what the tests read). No
compliance question was decided, no new `night-crew.toml` key was added, no app
grant slug was created, nothing was written to a hosted Supabase project.
