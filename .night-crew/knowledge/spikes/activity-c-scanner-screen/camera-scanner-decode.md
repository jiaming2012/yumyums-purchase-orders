# Spikes — camera-scanner-decode

Activity: Activity C — The scanner screen (staff redemption at the window)

> No `usm/roadmap.txt` on this target — hand-run convention (full preamble in
> `../activity-b-offline-first-replica/rxdb-pull-replica.md`). No substrate
> needed: the hash-scheme contract is asserted against the committed
> `supabase/seed.sql` literals, which ARE the schema's hash contract of record.

## The goal, and which legs need a spike

The card (roadmap Activity C): camera via `getUserMedia`, decode with
`html5-qrcode` (or `@zxing/browser`), hash the identity token on-device with
WebCrypto before any lookup (§12/§4), resolve replica-first then embedded-offer
(F5/F2/F3 handled at their own cards).

Falsifiable premises vs. environmental limits, stated:

- **Camera capture (`getUserMedia`) is NOT spiked** — it needs a physical
  camera and a per-device permission grant; headless this is an environmental
  refusal, not a premise. It is also proven-in-class browser platform API, and
  the card's own attended verification covers it. What IS falsifiable headless
  is everything downstream of the frame: decode → token extraction → hash →
  replica-key agreement.
- **Premise 1 — the candidate library decodes in HQ's no-build vanilla-JS
  context.** `html5-qrcode` (the design's first-named candidate) must load as
  a single committed script file in a plain page (no bundler) and decode a QR
  image in real Chromium. A library that needs a build step falsifies the
  choice for this app (convention: static, no build).
- **Premise 2 — hash-scheme agreement end-to-end.** The QR payload is the #10
  hybrid (URL wrapping the token). The on-device WebCrypto SHA-256 hex of the
  EXTRACTED token must equal the scheme the committed seed fixtures use
  (sha256 hex of the raw token — `supabase/seed.sql`). If the client's hash
  disagrees with the server's by even an encoding detail (case, encoding,
  wrapping-URL-vs-bare-token), every replica lookup silently misses — the
  worst failure class this screen has.

## Spike: decode-and-hash-chain

- proves: a QR PNG generated to the #10 hybrid payload shape
  (`https://hq.yumyums.kitchen/r/<token>` with token
  `card1-test-code-fixture-1` — the committed seed fixture's label), when
  loaded into real Chromium via a plain page that includes the vendored
  `html5-qrcode` single-file build, (a) decodes via the library's file-scan
  path to the exact payload; (b) the token extracted from the URL, digested
  in-page with `crypto.subtle.digest('SHA-256', …)` to lowercase hex, equals
  the committed seed literal
  `c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680` — i.e.
  the browser's hash IS a valid replica key against the built schema's
  contract; (c) cross-checked against Node's `createHash('sha256')` so a
  seed-literal typo cannot silently define the contract. Enumerated: decoded
  payload, extracted token, both hex digests, the vendored file's byte size
  (the no-build weight the card inherits).
- plan: spike-local `package.json`; `npm install qrcode html5-qrcode`
  (network → could-not-run if unreachable); Node generates the PNG with
  `qrcode`; copy `html5-qrcode.min.js` beside a minimal `decode.html`; a
  spike-local Playwright config (repo-root `@playwright/test` resolves by
  directory walk-up; NO repo webServer) drives Chromium: `setInputFiles` the
  PNG into the page's file input, run `Html5Qrcode.scanFile`, extract + digest
  in-page, assert against the literals passed in; node_modules git-ignored.
- script: .night-crew/spikes/activity-c-scanner-screen/camera-scanner-decode/01-decode-and-hash-chain.sh

## Verdict (run 2026-09-04, hand-run per the no-story-map convention)

- **decode-and-hash-chain: passed** — exit 0 (second execution; the first was
  RED on a script assertion defect, see Corrections — the page's own decode
  and hash were already correct on the first run's evidence). In real
  Chromium, from a plain page with the vendored single-file
  `html5-qrcode.min.js` (**375,364 bytes** — the no-build weight the card
  inherits): the generated QR decoded to
  `https://hq.yumyums.kitchen/r/card1-test-code-fixture-1`, the token
  extracted from the #10 URL wrapper, and the in-page WebCrypto SHA-256 hex
  equaled the committed seed literal
  `c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680` — with
  Node's `createHash` agreeing (the literal is not self-certifying).

**Conclusion:** the §12 chain — decode → extract → WebCrypto → replica key —
is proven in HQ's no-build context with `html5-qrcode` as ONE vendored classic
script. Build-facts: (1) the token extraction regex for the #10 hybrid is
`/\/r\/([^/?#]+)$/`; (2) camera capture (`getUserMedia`) remains the card's
attended-verification leg — environmental, not spikeable headless; (3) the
375 KB vendored file must be committed + precached (the sibling goal's spike
prices exactly that move).

## Corrections

- **Assertion-shape defect in the Playwright leg (fixed, re-run green):
  `waitForSelector` defaults to state `visible`, and the result div is an
  empty (hence hidden) element that only carries data attributes.** The first
  run timed out while its own error log showed `data-state="done"` and the
  correct hash already present — the page chain had succeeded; the wait
  condition was wrong. Fixed to `state: 'attached'`. Same class as Activity
  A's first-run `:t$`-vs-`:true$` grep (the assertion, not the premise).

## Review

- signed: operator, 2026-09-04 — covers 1 correction(s) (one-sitting batch
  across Activities B/C/D; "Sign off all three" on the phrase-checked batch
  question at the sitting's close).
