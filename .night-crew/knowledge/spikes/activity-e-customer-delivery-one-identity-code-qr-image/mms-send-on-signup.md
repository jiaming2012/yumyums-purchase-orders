# Spikes — mms-send-on-signup

Activity: Activity E — Customer delivery (one identity code → QR → image)

> Tool-run (`night-crew spikes run`). Both spikes work in a THROWAWAY worktree off `dev`
> (`../hq-worktrees/spike-e2-20261006`) and run Go tests against a fresh spike-owned database
> `hq_test_spike_e2_go` on `:5434` (the marketing package's `TestMain` migrates it). Spike 1's
> provider call goes to an `httptest.Server` stub — **no spike talks to SignalWire**; live
> acceptance over a registered number is D-KR1's attended leg. Never `:5433`.

## The goal, and which legs need a spike

The card (E2 — roadmap Activity E, D-KR1 / D-KR2, handoff §11, decision 208): when a signup
lands (the Fluent Forms import, H5), mint the identity code, render its QR, and send it as an
MMS image through SignalWire over the registered number — refusing to issue or send without
explicit SMS consent, honouring an inbound STOP, and appending the `code_sent` event the Stats
funnel counts. Three premises a script can settle now: (1) the sender cannot live in
`internal/marketing` — `TestNothingInThisPackageSends` reds any outbound HTTP there and any
import path containing a vendor name, so the send is a NEW package (`internal/delivery`) behind
an interface the marketing package imports, and that import passes the guard; (2) the provider
call (SignalWire's Twilio-compatible LaML REST — form-encoded `From`/`To`/`Body`/`MediaUrl`
to `/api/laml/2010-04-01/Accounts/{ProjectID}/Messages.json` under basic auth) is fully
mockable at the HTTP layer with no credentials, success and failure both observable — the
transport D-KR2's e2e test runs against; (3) the consent and STOP gaps are real on today's tree
and the committed fixture carries a subject for each branch: a subscriber with a phone and NO
SMS consent (the refusal subject), one WITH (the send subject), zero `code_sent` events, and no
inbound-SMS door at all.

## Spike: sender-outside-marketing-and-the-provider-call-is-mockable

- proves: (a) control — `TestNothingInThisPackageSends` passes on the unmutated worktree;
  (b) mutation — a file in `internal/marketing` that takes `http.Post`'s address reds the
  guard (so a sender placed there can never ship); (c) a throwaway `internal/delivery` package
  with a `Sender` interface and a `SignalWire` implementation passes its own tests against an
  `httptest.Server` that asserts method, path, basic auth and the four form fields, answers
  `{"sid","status":"queued"}`, and on a 401 the call returns an error carrying the body;
  (d) a marketing file importing `internal/delivery` and holding a `delivery.Sender` passes
  the guard (the seam is legal; a package path containing `signalwire` would not be).
- plan: worktree, fresh Go database, four `go test` legs with pass/fail counts read from `-v`
  output, every mutation removed before the next leg, worktree clean at exit.
- script: .night-crew/spikes/activity-e-customer-delivery-one-identity-code-qr-image/mms-send-on-signup/01-sender-outside-marketing-and-the-provider-call-is-mockable.sh

### Runs

- 2026-10-05T13:12:08Z · exit 1 · failed
- 2026-10-05T13:13:17Z · exit 0 · passed

## Spike: consent-refusal-and-stop-have-no-door-today

- proves: on a fresh database with the committed Fluent Forms fixture imported through
  `ImportSubscribers`: (a) a subscriber exists with phone `+17735550117` and `sms_consent =
  false` (fixture 1183, consent `["Email"]`) — the refusal subject; (b) one exists with
  `+17735559930` and `sms_consent = true` (fixture 1182, `["Phone"]`) — the send subject;
  (c) `subscriber_events` holds zero `code_sent` rows; (d) `POST
  /api/v1/marketing/sms/inbound` as an unauthenticated carrier webhook answers 404 — no STOP
  door exists; (e) `opted_out_at` is NULL on the refusal subject. All five are the red-first
  baseline for D-KR2's two tests.
- plan: a throwaway `_test.go` beside `subscribers_test.go` using the package's own helpers
  (`setupTestDB`, `ffImporter`, `mountedMux`, `do`), one `go test -run` leg, counts read from
  `-v` output.
- script: .night-crew/spikes/activity-e-customer-delivery-one-identity-code-qr-image/mms-send-on-signup/02-consent-refusal-and-stop-have-no-door-today.sh

### Runs

- 2026-10-05T13:12:22Z · exit 1 · failed
- 2026-10-05T13:13:32Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **sender-outside-marketing-and-the-provider-call-is-mockable: passed** on its second run (exit
  0, 14 s): (a) control `TestNothingInThisPackageSends` PASS; (b) with `var spikeEgress =
  http.Post` in `internal/marketing`, the guard FAILS on exactly that file — "zz_spike_egress.go
  contains an outbound HTTP call ("http.Post") and is not in egressAllowlist"; (c) the throwaway
  `internal/delivery` package vets and its two tests PASS against the `httptest` stub — `POST
  /api/laml/2010-04-01/Accounts/proj-spike/Messages.json`, basic auth `proj-spike`, form
  `From/To/Body/MediaUrl` all as sent, `sid=SM-spike-e2-0001`; a 401 answers an error carrying
  `HTTP 401: {"code":20003,"message":"Authenticate"}`; (d) marketing importing
  `github.com/yumyums/hq/internal/delivery` and holding a `delivery.Sender` — the guard PASSES.
  **The first run's `exit 1 · failed` line is a sitting-side slip, not a finding:** every leg
  (a)–(d) was green and the script went red at its final `main_tree_untouched` check because the
  main tree's `backend/` was dirty with the sitting's own uncommitted SignalWire denylist edit
  (`subscribers_test.go`, committed as `7d252ce` moments later). Re-run on the committed tree:
  clean.
- **consent-refusal-and-stop-have-no-door-today: passed** on its second run (exit 0, 8 s) —
  `SPIKE-E2-2: subscribers=4 with_phone=3 sms_consenting=2 code_sent=0 inbound_status=404`;
  refusal subject `+17735550117` (fixture 1183, `["Email"]`) has `sms_consent=false` and
  `opted_out_at` NULL; send subject `+17735559930` (fixture 1182, `["Phone"]`) has
  `sms_consent=true`. Note the fixture's five rows land as FOUR subscribers: 1184 `(773)
  555-4821` and 1180 `17735554821` normalize to the same E.164 number and merge — a fact the
  card's e2e test must not trip over. The first run's failed line is the same main-tree-dirty
  slip as above; the test itself passed both times.

## Corrections

- none agent-reached — all three premises held. Precisions carried into the card: the sender
  package path must carry NO vendor name (the denylist matches import paths by substring —
  `internal/delivery/signalwire` would red marketing for importing it; `internal/delivery` with
  a `signalwire.go` file inside does not); the fixture merges 1184/1180 into one subscriber, so
  "the consenting subject" in tests is `+17735559930`, never a count of five.

## Comebacks

- Sitting hygiene: a spike's `main_tree_untouched` check reads the WHOLE main-tree `backend/`,
  so an uncommitted edit anywhere under it — even one unrelated to the spike — reds the run.
  Commit the sitting's own edits before running spikes. Recorded here so the two failed lines
  above are read as what they are.

## Review

- signed: operator, 2026-10-05 — covers 0 correction(s) (reviewed at the slate sitting of
  2026-10-05, §4 batch sign-off; no corrections to review; the sitting-side comeback stated).
