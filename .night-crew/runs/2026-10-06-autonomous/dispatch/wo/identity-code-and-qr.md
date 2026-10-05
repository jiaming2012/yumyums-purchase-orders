---
complexity: new-mechanism
---
# WO: identity-code-and-qr — one permanent identity code per customer, as a QR the shipped scanner reads offline and online

## Intent

A customer who signs up gets ONE permanent identity code. Its QR is a URL wrapping
the identity token plus the offer current at issue, so the truck's scanner shows that offer
offline before the customer has replicated, and the customer's full server-side entitlement list
online. Nothing is sent by this card; it mints, renders and projects.

## Spec

Migration `0088_identity_codes`: `subscribers.identity_token_hash text unique null`,
`subscribers.identity_minted_at timestamptz null`, table `identity_media (id uuid pk default
gen_random_uuid(), subscriber_id uuid not null references subscribers(id) on delete cascade, png
bytea not null, created_at timestamptz not null default now())`, Down drops them.
`MintIdentityCode(ctx, tx, subscriberID)` in `backend/internal/marketing/identity.go`: idempotent
(a subscriber with a hash returns it); token = 16 chars from `crypto/rand` over the base32
alphabet `23456789abcdefghjkmnpqrstuvwxyz` (lower-case, no vowels-as-digits ambiguity);
`token_hash = hex(sha256(token))`; descriptor = the subscriber's first-touch campaign via
`source_short → qr_codes.campaign_id → campaigns_admin` as `{"label": offer_text, "campaign_id":
id, "expires_at": ends_at as YYYY-MM-DD, "face_value": face_value_cents/100}`; with no
first-touch code the payload is `<QRBaseURL>/r/<token>` with NO fragment (identity-only); the PNG
is `qrcode.New(payload, qrcode.Low).PNG(512)` stored in `identity_media`; the raw token is never
written anywhere else. `GET /m/{id}.png` under `MountPublic` (beside `/q/{short}`): `image/png`,
`Cache-Control: private, max-age=3600`, 404 on unknown/bad id, HEAD supported.
`ProjectIdentityCode(ctx, cfg, row)` in `projection.go`: POST `[{id, token_hash, campaign_id,
expires_at}]` to `/codes` with `Prefer: resolution=merge-duplicates` as service_role —
`id` = a uuid derived deterministically from the subscriber id (uuid v5 over the subscriber id,
so re-projection is an upsert), `expires_at` = the campaign's `ends_at`; identity-only codes are
NOT projected (no campaign → no entitlement row). Unconfigured or failing projection → the mint
still commits and the import result carries `warnings:["not_projected"]` (decision 187's shape).
`ImportSubscribers` mints for every NEW subscriber in the same request after its transaction
commits. `GET /subscribers/{id}` adds `identity_code.status` ∈ `not_minted | minted | sent`
(`sent` only when a `code_sent` event exists — Card 2's) and `identity_code.media_url`.
`marketing.html`: `#scan-file-surface` 320 → 640 px (one line); `sw.js` regenerated (51).

## Gates

Red-first, shown: `[IC-01]` in `tests/marketing-identity.spec.js` — import the fixture
with `HQ_FF_FIXTURE_PATH`, fetch `/m/{media_id}.png` for the first-touch subscriber (seed a
`qr_codes` row and set `source_short` in the fixture path or via SQL), scan it through the
SHIPPED page OFFLINE with nothing seeded → `data-kind=embeddedOffer`, label = the campaign's
`offer_text`, `data-token-hash` = sha256 of the token the server minted (the test reads the token
ONLY through the PNG it decodes — the spike's spec is the recipe; red on the pre-change tree
because `/m/` is 404); `[IC-02]` the same PNG ONLINE with the lookup route answering the projected
row → `offerReady`, source `server`; `[IC-03]` NEGATIVE control: a Medium/version-11 PNG of the
same payload (generated in-test with the committed `qr-embedded.png`'s payload length) renders no
offer — `decodeError` or `invalidPayload`, never `embeddedOffer`/`offerReady` with a wrong hash.
Go: `TestMintIdentityCodeIsIdempotent`, `TestMintIdentityCodeNoFirstTouchIsIdentityOnly`
(payload has no `#o=`; nothing projected), `TestProjectIdentityCodeUpserts` (httptest PostgREST:
path `/codes`, `Prefer` header, body fields, 201 → nil; 500 → error, mint still committed),
`TestMigration0088IdentityCodesDownAndUpRoundTrip`, `TestNothingInThisPackageSends` GREEN with
NO allowlist change. G1 build+vet; full Go `-p 1` counts; `sw.js` idempotent, count 51; the
**full Playwright suite** on the merged tree under the lock (precache moved). G6 re-runs `[IC-01]`
and `[IC-03]` itself and reads the migration's Down.

The loop itself executes only the fenced lines below (compile and vet, at baseline, at VERIFY and at the run-branch tip after merge). Every test clause above is proven by the session's recorded red/green evidence, re-run by the orchestrator's G6 review, and by the orchestrator's full suites on the merged tree under the lock.

```
go build -C backend ./...
go vet -C backend ./...
```

## Context

the shipped reader's bound (320 px surface, ~57 modules, non-monotonic) is why the
encoding is Low/16-char/date-only and why the surface widens — do not "upgrade" to Medium. The
spike's `qrgen.go` and `spike-e1.spec.js` under
`.night-crew/spikes/activity-e-…/identity-code-and-qr/` are the working drafts. `CodePNGHandler`
(`qrpng.go`) is the PNG-handler template; `ProjectCampaign` the projection template;
`TestMigration0086ErasureBackstopDownAndUpRoundTrip` the migration-test template. Box rules and
the B-486 checkout rule from the slate's precondition flags apply.

DONE-WHEN (the slate's Gates section, verbatim — these are the clauses you must prove and report on; references to G6 and the full Playwright suite are the orchestrator's legs): Red-first, shown: `[IC-01]` in `tests/marketing-identity.spec.js` — import the fixture
with `HQ_FF_FIXTURE_PATH`, fetch `/m/{media_id}.png` for the first-touch subscriber (seed a
`qr_codes` row and set `source_short` in the fixture path or via SQL), scan it through the
SHIPPED page OFFLINE with nothing seeded → `data-kind=embeddedOffer`, label = the campaign's
`offer_text`, `data-token-hash` = sha256 of the token the server minted (the test reads the token
ONLY through the PNG it decodes — the spike's spec is the recipe; red on the pre-change tree
because `/m/` is 404); `[IC-02]` the same PNG ONLINE with the lookup route answering the projected
row → `offerReady`, source `server`; `[IC-03]` NEGATIVE control: a Medium/version-11 PNG of the
same payload (generated in-test with the committed `qr-embedded.png`'s payload length) renders no
offer — `decodeError` or `invalidPayload`, never `embeddedOffer`/`offerReady` with a wrong hash.
Go: `TestMintIdentityCodeIsIdempotent`, `TestMintIdentityCodeNoFirstTouchIsIdentityOnly`
(payload has no `#o=`; nothing projected), `TestProjectIdentityCodeUpserts` (httptest PostgREST:
path `/codes`, `Prefer` header, body fields, 201 → nil; 500 → error, mint still committed),
`TestMigration0088IdentityCodesDownAndUpRoundTrip`, `TestNothingInThisPackageSends` GREEN with
NO allowlist change. G1 build+vet; full Go `-p 1` counts; `sw.js` idempotent, count 51; the
**full Playwright suite** on the merged tree under the lock (precache moved). G6 re-runs `[IC-01]`
and `[IC-03]` itself and reads the migration's Down.

Slate lead (mechanism of record pointers): Roadmap Activity E card E1 (the "Slated 2026-10-05" paragraph on the card is the mechanism of
record); handoff §10 and #10 (operator-resolved hybrid payload); D-KR3; goal ledger
`spikes/activity-e-customer-delivery-one-identity-code-qr-image/identity-code-and-qr.md` (**binding
build-facts** — the variant table; spike 02's five PostgREST legs) and its extraction record (one
signed correction: the encoding).

PARK note (narrow, operator-only — if you meet one of these, STOP and report it as a park; never decide it): a rule for which campaign a signup with NO first-touch
code is entitled to (the card ships identity-only; guessing a campaign is a product call); an
objection to HQ holding the scannable PNG at rest (the card holds it in Postgres, unguessable id,
cascade-deleted with the subscriber — the erasure path); a new `scan_attempts` status or
submit-machine pair; a `night-crew.toml` key or token. The token alphabet, the uuid-v5 namespace,
the test names and the 640 px figure are the night's.

RUN RULES — run 20261006, repo yumyums/hq (binding; from the signed launch prompt `.night-crew/knowledge/reference/launch-20261006.md`, which you may read in full):

Authority and limits. Batch sign-off was given by the operator on 2026-10-05; do not pause for per-change sign-off. Never push, never tag, never deploy, never touch `main`. You never choose a product fork — what to build, what a milestone means. If the card cannot be built as specified without one, STOP and report it as a park with the question written out; do not improvise. Editing a file outside the card's footprint is NOT a park and not a breach — say so in the merge-intent note.

How your work is recorded (this night rides night-crew's run loop). You are working in a loop-owned git worktree on a loop-owned branch. Do NOT run `git commit`, do not create or switch branches, do not stash: leave every change in the working tree and the loop commits your whole diff as one commit when you report through the result contract. Consequently the commit-level mechanics (atomic commits, the `Night-Crew-Run: 20261006` trailer) are applied by the orchestrator at its merge, not by you. A park is reported through the result contract as `halt`, with the operator question written out in the summary.

Merge-intent note (REQUIRED, the FIRST file you write, before implementing): write `.night-crew/runs/2026-10-06-autonomous/merge-intents/identity-code-and-qr.md` — the shared files you will touch (each file outside your own packages, one line of why), what must survive any merge, what is safe to drop. Empty fields say "nothing here" explicitly. It also states, per done-when clause, which rows ride a stub or fixture — tonight the SENDER STUB IS THE POINT (D-KR2): say so per clause.

Per-change mechanics (this repo has NO OpenSpec — create no `openspec/` scaffolding). Red-first evidence: show each named test RED on the pre-change tree (the slate names the spike recipe to use as the red) and then GREEN; save both outputs, with the exact commands and exit lines, under `.night-crew/runs/2026-10-06-autonomous/logs/identity-code-and-qr/` (they ride your diff). Flip this card's Activity E line in `.night-crew/knowledge/roadmap.md` from `PLANNED` to `LANDED` (the orchestrator adds the merge SHA); otherwise roadmap.md is append-only. Migration numbers are fixed (Card 1 = `0088`, Card 2 = `0089`); renumbering is scope drift. No `night-crew.toml` key or token (that is a park) — only its roll-call comment count moves (7 → 8 → 9 as each new spec joins the `marketing` seam by filename). Write nothing under `.night-crew/qa/` (the loop refuses the commit).

Service worker. `sw.js` is generated from git HEAD, so it CANNOT be regenerated correctly from an uncommitted tree: do NOT hand-edit it and do NOT include `sw.js` or `version.json` in your diff — if a test target rewrote them, run `git checkout -- sw.js version.json` before you report. The orchestrator regenerates `sw.js` at the merged HEAD and checks the precache count is still 51. You add or remove no precached file.

Suites. You run the confined gates: your new spec(s) plus the `marketing` seam subset in Playwright, the Go packages you touched, and ONE full Go suite (`go test ./... -p 1 -count=1`, minutes) under the lock. You do NOT run the full Playwright suite (55 min) — the orchestrator runs it on the merged tree under the lock; say plainly in your report that it is owed.

Box rules. `export PATH="/usr/local/go/bin:$PATH"` before every Go or Playwright leg (without it Playwright's webServer dies `go: not found` / exit 127, which is not a test failure). 🛑 Tests run on Postgres port 5434 (`yumyums-test-pg`, role `hqtest`, `task test:db:up`, coordinates via `task test:targets`) — NEVER 5433, which is the dev AND production cluster; no suite, probe or psql may point at 5433. Use your own databases: `TEST_PORT` of your own and `TEST_DB_NAME=hq_test_e2e_e1_20261006` for Playwright; Go database `hq_test_go_e1` on :5434 for `DB_TEST_URL`. `go test ./... -p 1` — `-p 1` is load-bearing; `DB_TEST_URL` MUST be set or the Go suite exits 0 while skipping every DB test: report test counts, not `ok`. A Go test database must be migrated before DB tests run (boot the built server once against it). Only ONE full suite runs on the box at a time: every full Playwright suite AND every full Go suite runs under `flock /tmp/hq-full-suite.lock`; a `TestRowVisibilityRLS` red is re-run alone before it is reported, both exit lines stated. 🛑 B-477: in a fresh worktree make THREE symlinks before the first Playwright leg — `node_modules`, `marketing/sync/harness/node_modules`, `.night-crew/qa/spike-supabase/rxdb/node_modules` — each pointing at the main checkout's (`/home/jcole/projects/hq/...`); then warm the backend build (`cd backend && go build -o /dev/null ./cmd/server/`) BEFORE Playwright's 60 s webServer window. If `webServer` wedges, hand-provision the stack and point `NIGHTCREW_ENV_URL` at it — first resort, not last. 🛑 B-486: after EVERY Playwright leg and before any commit run `git checkout -- .night-crew/runs/2026-10-02-autonomous/logs/h4/states/` — a states spec rewrites committed PNGs there; never commit them. Never use `git add -A` / `git add .`; stage files by name. The local `spike-supabase` substrate is up and is read-only to you: never `--fresh`, never tear it down.

🛑 No provider is ever called. The `HQ_SW_*` environment variables are unset on this box and in every gate; the sender is the recording stub. Any real send, or any code path reaching `*.signalwire.com` outside `backend/internal/delivery`, is a stop. No new egress allowlist entry anywhere.

The server lookup in `[IC-02]` is served by `page.route` on the door's path (the e2e stack has no substrate) — the one permitted stub.

Report honestly: for each Gates clause say what you RAN and what you OBSERVED (command, exit line, counts), which clauses ride a stub or fixture, and anything you did not run. A clause you could not prove is reported as unproven, never as passed.

## Result

(reserved for the loop — sessions report through the result contract)
