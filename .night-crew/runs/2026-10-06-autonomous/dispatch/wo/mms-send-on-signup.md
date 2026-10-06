---
complexity: new-mechanism
---
# WO: mms-send-on-signup — the code reaches the customer as an MMS image, lawfully, through a stub tonight

## Intent

When a signup lands with SMS consent, the identity code goes out as an MMS image over
the registered number; without consent nothing is issued or sent and the sheet says so; an
inbound STOP opts the person out and blocks the next send. Every gate proves this with a
recording stub — no message leaves this box tonight.

## Spec

Package `backend/internal/delivery` (path carries NO vendor name): `type Sender
interface { SendMMS(ctx, to, mediaURL, body string) (sid string, err error) }`; `SignalWire{BaseURL,
ProjectID, APIToken, From, HTTP}` posting form-encoded `From/To/Body/MediaUrl` with basic auth to
`{BaseURL}/api/laml/2010-04-01/Accounts/{ProjectID}/Messages.json`, non-2xx → error carrying
status+body, reply `sid` required (the spike's `signalwire.go` is the draft); `Stub{Pool}` that
INSERTs into `delivery_log (id, subscriber_id, to_e164, media_url, body, sid, provider, sent_at)`
(migration `0089_delivery_log`, Down drops) and returns a `stub-` sid. `FromEnv()`: all four of
`HQ_SW_SPACE_URL`, `HQ_SW_PROJECT_ID`, `HQ_SW_API_TOKEN`, `HQ_SW_FROM` set → `SignalWire`;
otherwise `Stub`, logged once at boot. `Deps.Sender delivery.Sender` wired in `main.go` and in
`helpers_test.go`'s `testDeps` (always the stub). `internal/marketing` imports `internal/delivery`
ONLY for the interface (spike-proven legal). In `ImportSubscribers`, after Card 1's mint for a
NEW subscriber: `sms_consent && phone_e164 != ""` → `Sender.SendMMS(phone, media_url, "Your
Yumyums code — show this at the window")` → on success append `code_sent` with `ref =
{sid, provider, media_id}`; on error append nothing and carry `warnings:["send_failed"]`;
without consent or without a phone, Card 1 still mints (identity is who they are — one code
ever) and ONLY the send is withheld: the `signed_up` event's `ref` gains
`{"code_withheld":"no_consent"}` (or `"no_phone"`). `POST /subscribers/{id}/resend`: with consent and a minted
media → a real `SendMMS` of the SAME media, `code_sent` appended, 202 `{sent:true, sid}`; without
consent → 202 `{sent:false, reason:"no_consent"}` and `resend_requested` as today. `POST
/api/v1/marketing/sms/inbound` under `MountPublic`: form-encoded (`From`, `Body`, `MessageSid`);
when `HQ_SW_SIGNING_KEY` is set, validate `X-SignalWire-Signature` (HMAC-SHA1 over URL + sorted
params, the Twilio-compatible scheme; 403 on mismatch); when unset AND the sender is the real
`SignalWire` → 503 `inbound_unsigned`; stub mode accepts unsigned. `Body` trimmed, upper-cased ∈
{STOP, STOPALL, UNSUBSCRIBE, CANCEL, END, QUIT} → `opted_out_at = now()`, `sms_consent = false`,
`opted_out` event `ref {via:"sms", sid}`; unknown `From` → 204 with nothing written; other
bodies → 204. A send to a subscriber with `opted_out_at` set is refused before the sender is
called. `marketing/subscribers.js`: the sheet's identity-code line reads **Sent · <date>** /
**Not sent — no SMS consent** / **Opted out · <date>**; the Resend button's answer shows
`reason` when `sent:false`. `sw.js` regenerated (51).

## Gates

Red-first, shown against the spike's baseline (spike 02: `+17735550117` no consent,
`+17735559930` consenting, 0 `code_sent`, `/sms/inbound` 404): Go
`TestNewConsentingSubscriberIsMintedAndSent` (stub called once with the media URL; `code_sent`
= 1; detail `identity_code.status = sent`), `TestNoConsentIsNeverSent` (stub never called; 0
`code_sent`; `signed_up.ref.code_withheld = no_consent`), `TestInboundStopOptsOutAndBlocksNextSend`
(STOP → `opted_out_at` set; a resend answers `sent:false`; stub call count unchanged),
`TestInboundRejectsBadSignature` (signing key set, wrong signature → 403, nothing written),
`TestInboundUnknownNumberWritesNothing`, `TestResendSendsSameMediaOnce`,
`TestMigration0089DeliveryLogDownAndUpRoundTrip`, `TestNothingInThisPackageSends` STILL GREEN
(the sender is outside; **no allowlist change**), and `internal/delivery`'s own
`TestSignalWireSendIsMockableAtHTTP` + `TestSignalWireRefusalIsAnError` (the spike's tests
re-homed). Playwright `tests/marketing-delivery.spec.js`: `[MS-01]` import the fixture through the
Subscribers tab → `+17735559930`'s sheet shows **Sent**; `[MS-02]` `+17735550117`'s sheet shows
**Not sent — no SMS consent** and Resend answers "no consent"; `[MS-03]` after an unsigned stub
STOP posted by the test, the sheet shows **Opted out** and Resend is refused. **D-KR2 is measured
by `tests/`** — these three are the KR's gate. G1; full Go `-p 1` counts; `sw.js` 51; the **full
Playwright suite on the complete tree = the final suite** under the lock. G6 greps the diff for
`signalwire.com` outside `internal/delivery` and runs `[MS-02]` itself.

The loop itself executes only the fenced lines below (compile and vet, at baseline, at VERIFY and at the run-branch tip after merge). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's G6 review, and by the orchestrator's full suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```

## Context

the Stats funnel counts `code_sent` — append it only when a sender returned a sid.
`ResendQRHandler` and `TestResendRecords202AndAppendsNoCodeSent` are the shapes to extend (the
latter's assertion flips to "no `code_sent` WITHOUT consent"). The spike's `delivery/` pair and
`zz_spike_e2_test.go` under `.night-crew/spikes/activity-e-…/mms-send-on-signup/` are the drafts.
The fixture's five rows land as FOUR subscribers (1184/1180 merge) — name subjects by phone. Box
rules and the B-486 checkout rule apply.

DONE-WHEN (the slate's Gates section, verbatim — these are the clauses you must prove and report on; references to G6 and the full Playwright suite are the orchestrator's legs): Red-first, shown against the spike's baseline (spike 02: `+17735550117` no consent,
`+17735559930` consenting, 0 `code_sent`, `/sms/inbound` 404): Go
`TestNewConsentingSubscriberIsMintedAndSent` (stub called once with the media URL; `code_sent`
= 1; detail `identity_code.status = sent`), `TestNoConsentIsNeverSent` (stub never called; 0
`code_sent`; `signed_up.ref.code_withheld = no_consent`), `TestInboundStopOptsOutAndBlocksNextSend`
(STOP → `opted_out_at` set; a resend answers `sent:false`; stub call count unchanged),
`TestInboundRejectsBadSignature` (signing key set, wrong signature → 403, nothing written),
`TestInboundUnknownNumberWritesNothing`, `TestResendSendsSameMediaOnce`,
`TestMigration0089DeliveryLogDownAndUpRoundTrip`, `TestNothingInThisPackageSends` STILL GREEN
(the sender is outside; **no allowlist change**), and `internal/delivery`'s own
`TestSignalWireSendIsMockableAtHTTP` + `TestSignalWireRefusalIsAnError` (the spike's tests
re-homed). Playwright `tests/marketing-delivery.spec.js`: `[MS-01]` import the fixture through the
Subscribers tab → `+17735559930`'s sheet shows **Sent**; `[MS-02]` `+17735550117`'s sheet shows
**Not sent — no SMS consent** and Resend answers "no consent"; `[MS-03]` after an unsigned stub
STOP posted by the test, the sheet shows **Opted out** and Resend is refused. **D-KR2 is measured
by `tests/`** — these three are the KR's gate. G1; full Go `-p 1` counts; `sw.js` 51; the **full
Playwright suite on the complete tree = the final suite** under the lock. G6 greps the diff for
`signalwire.com` outside `internal/delivery` and runs `[MS-02]` itself.

Slate lead (mechanism of record pointers): Roadmap Activity E card E2 (the "Slated 2026-10-05" paragraph is the mechanism of record);
handoff §11 (SignalWire, decision 208; compliance R5); D-KR1 (attended), D-KR2 (tonight's);
goal ledger `spikes/activity-e-…/mms-send-on-signup.md` (**binding build-facts** — the egress
guard refuses a sender in marketing; the LaML request shape proven against a stub; the fixture's
refusal and send subjects; the 1184/1180 merge) and its extraction record (no corrections).

PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): a new `subscriber_events.kind` (the CHECK constraint is
the lifecycle); any copy that promises a send the stub did not make; a product rule for
re-consent after STOP (the card blocks forever; re-opt-in is a later card); a `night-crew.toml`
key; any real provider call. The env var names, the stub's sid prefix, the signature scheme's
exact canonicalisation and the test names are the night's.

RUN RULES — run 20261006, repo yumyums/hq (binding; from the signed launch prompt `.night-crew/knowledge/reference/launch-20261006.md`, which you may read in full):

Authority and limits. Batch sign-off was given by the operator on 2026-10-05; do not pause for per-change sign-off. Never push, never tag, never deploy, never touch `main`. You never choose a product fork — what to build, what a milestone means. If the card cannot be built as specified without one, STOP and report it as a park with the question written out; do not improvise. Editing a file outside the card's footprint is NOT a park and not a breach — say so in the merge-intent note.

How your work is recorded (this night rides night-crew's run loop). You are working in a loop-owned git worktree on a loop-owned branch. Do NOT run `git commit`, do not create or switch branches, do not stash: leave every change in the working tree and the loop commits your whole diff as one commit when you report through the result contract. Consequently the commit-level mechanics (atomic commits, the `Night-Crew-Run: 20261006` trailer) are applied by the orchestrator at its merge, not by you. A park is reported through the result contract as `halt`, with the operator question written out in the summary.

Merge-intent note (REQUIRED, the FIRST file you write, before implementing): write `.night-crew/runs/2026-10-06-autonomous/merge-intents/mms-send-on-signup.md` — the shared files you will touch (each file outside your own packages, one line of why), what must survive any merge, what is safe to drop. Empty fields say "nothing here" explicitly. It also states, per done-when clause, which rows ride a stub or fixture — tonight the SENDER STUB IS THE POINT (D-KR2): say so per clause.

Per-change mechanics (this repo has NO OpenSpec — create no `openspec/` scaffolding). Red-first evidence: show each named test RED on the pre-change tree (the slate names the spike recipe to use as the red) and then GREEN; save both outputs, with the exact commands and exit lines, under `.night-crew/runs/2026-10-06-autonomous/logs/mms-send-on-signup/` (they ride your diff). Flip this card's Activity E line in `.night-crew/knowledge/roadmap.md` from `PLANNED` to `LANDED` (the orchestrator adds the merge SHA); otherwise roadmap.md is append-only. Migration numbers are fixed (Card 1 = `0088`, Card 2 = `0089`); renumbering is scope drift. No `night-crew.toml` key or token (that is a park) — only its roll-call comment count moves (7 → 8 → 9 as each new spec joins the `marketing` seam by filename). Write nothing under `.night-crew/qa/` (the loop refuses the commit).

Service worker. `sw.js` is generated from git HEAD, so it CANNOT be regenerated correctly from an uncommitted tree: do NOT hand-edit it and do NOT include `sw.js` or `version.json` in your diff — if a test target rewrote them, run `git checkout -- sw.js version.json` before you report. The orchestrator regenerates `sw.js` at the merged HEAD and checks the precache count is still 51. You add or remove no precached file.

Suites. You run the confined gates: your new spec(s) plus the `marketing` seam subset in Playwright, the Go packages you touched, and ONE full Go suite (`go test ./... -p 1 -count=1`, minutes) under the lock. You do NOT run the full Playwright suite (55 min) — the orchestrator runs it on the merged tree under the lock; say plainly in your report that it is owed.

Box rules. `export PATH="/usr/local/go/bin:$PATH"` before every Go or Playwright leg (without it Playwright's webServer dies `go: not found` / exit 127, which is not a test failure). 🛑 Tests run on Postgres port 5434 (`yumyums-test-pg`, role `hqtest`, `task test:db:up`, coordinates via `task test:targets`) — NEVER 5433, which is the dev AND production cluster; no suite, probe or psql may point at 5433. Use your own databases: `TEST_PORT` of your own and `TEST_DB_NAME=hq_test_e2e_e2_20261006` for Playwright; Go database `hq_test_go_e2` on :5434 for `DB_TEST_URL`. `go test ./... -p 1` — `-p 1` is load-bearing; `DB_TEST_URL` MUST be set or the Go suite exits 0 while skipping every DB test: report test counts, not `ok`. A Go test database must be migrated before DB tests run (boot the built server once against it). Only ONE full suite runs on the box at a time: every full Playwright suite AND every full Go suite runs under `flock /tmp/hq-full-suite.lock`; a `TestRowVisibilityRLS` red is re-run alone before it is reported, both exit lines stated. 🛑 B-477: in a fresh worktree make THREE symlinks before the first Playwright leg — `node_modules`, `marketing/sync/harness/node_modules`, `.night-crew/qa/spike-supabase/rxdb/node_modules` — each pointing at the main checkout's (`/home/jcole/projects/hq/...`); then warm the backend build (`cd backend && go build -o /dev/null ./cmd/server/`) BEFORE Playwright's 60 s webServer window. If `webServer` wedges, hand-provision the stack and point `NIGHTCREW_ENV_URL` at it — first resort, not last. 🛑 B-486: after EVERY Playwright leg and before any commit run `git checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/` — a states spec rewrites committed PNGs there; never commit them. Never use `git add -A` / `git add .`; stage files by name. The local `spike-supabase` substrate is up and is read-only to you: never `--fresh`, never tear it down.

🛑 No provider is ever called. The `HQ_SW_*` environment variables are unset on this box and in every gate; the sender is the recording stub. Any real send, or any code path reaching `*.signalwire.com` outside `backend/internal/delivery`, is a stop. No new egress allowlist entry anywhere.

The server lookup in `[IC-02]` is served by `page.route` on the door's path (the e2e stack has no substrate) — the one permitted stub.

Report honestly: for each Gates clause say what you RAN and what you OBSERVED (command, exit line, counts), which clauses ride a stub or fixture, and anything you did not run. A clause you could not prove is reported as unproven, never as passed.

## Result

(reserved for the loop — sessions report through the result contract)
