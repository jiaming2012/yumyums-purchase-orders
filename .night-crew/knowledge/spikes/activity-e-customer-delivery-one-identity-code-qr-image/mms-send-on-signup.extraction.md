# Extraction — mms-send-on-signup

Outcome: confirmed, no corrections

Approach used: a throwaway worktree of `dev` with a fresh Go database on `:5434`
(`TestMain` migrates), four `go test` legs for spike 01 — the egress guard as control,
the guard under an `http.Post` mutation inside `internal/marketing`, a throwaway
`internal/delivery` package (Sender interface + SignalWire over the Twilio-compatible
LaML Messages resource) tested against an `httptest.Server`, and the guard again with
marketing importing that package — and one `go test` leg for spike 02, a throwaway
`_test.go` beside `subscribers_test.go` importing the committed Fluent Forms fixture
through `ImportSubscribers` and probing the would-be STOP route through the mounted
mux. Tool-recorded runs: each spike exit 1 once (the sitting's own uncommitted edit
dirtied the main tree's `backend/`; every leg was green) then exit 0. No provider was
called. Candidate input for the card, not an adoption (NFR-6).

Confirmed: (a) `TestNothingInThisPackageSends` reds any outbound HTTP call in
`internal/marketing` by file name — a sender cannot live there; (b) a vendor-free
package `internal/delivery` whose `SignalWire.SendMMS` posts form-encoded
`From/To/Body/MediaUrl` with basic auth `ProjectID:APIToken` to
`/api/laml/2010-04-01/Accounts/{ProjectID}/Messages.json` is fully observable against
a local stub — sid on 201, an error carrying status and body on 401; (c) marketing
importing `internal/delivery` and holding a `delivery.Sender` passes the guard;
(d) on a fresh database the fixture yields 4 subscribers (1184/1180 merge on E.164),
3 with phones, 2 consenting; `+17735550117` has no SMS consent and is not opted out,
`+17735559930` consents; zero `code_sent` events; `POST /api/v1/marketing/sms/inbound`
is 404 — the consent refusal and the STOP door are both unbuilt.

Learned: (1) the denylist matches IMPORT PATHS by substring, so the sender's package
path must not contain the vendor name — `internal/delivery` (file `signalwire.go`) is
legal, `internal/delivery/signalwire` is not; (2) D-KR2's mocked transport is a
`delivery.Sender` injected through `Deps`, with the HTTP-level stub reserved for the
delivery package's own tests; (3) the fixture's phone normalization merges two rows,
so the compliance tests name subjects by phone, never by fixture count; (4) a spike's
final tree check reads all of `backend/` — commit the sitting's edits first.

Plan change: none to scope — the card ships `internal/delivery` (Sender + SignalWire),
the `Deps.Sender` seam, the consent refusal, the STOP webhook under `MountPublic`, and
the `code_sent` event as specified; its red-first recipes are spike 02's five baseline
lines and spike 01's leg (b).
