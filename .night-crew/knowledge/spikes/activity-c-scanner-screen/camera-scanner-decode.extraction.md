# Extraction — camera-scanner-decode

Outcome: confirmed

Approach used: `html5-qrcode` vendored as ONE single-file classic `<script>`
(**375,364 bytes** — the SortableJS pattern, no bundler), decoding via the
library's file-scan path; token extracted from the #10 hybrid URL wrapper with
`/\/r\/([^/?#]+)$/`; on-device `crypto.subtle.digest('SHA-256', …)` to
lowercase hex as the replica key. A candidate the card's design.md adopts or
not (NFR-6).

Confirmed: the full §12 chain in HQ's no-build context, in real Chromium — the
generated QR decoded to `https://hq.yumyums.kitchen/r/card1-test-code-fixture-1`,
the extracted token's in-page WebCrypto hash equaled the committed seed
literal `c5a1641409efd198e5a55417f209eda33500fd199f1fa7fa0d8a2567ee1f9680`,
and Node's `createHash` agreed (the seed literal is not self-certifying). The
browser's hash IS a valid replica key against the built schema's contract —
the silent-miss failure class (client/server hash-scheme disagreement) is
closed for this scheme.

Learned: (nothing new — the one correction was an assertion-shape defect in
the spike's own Playwright wait, not a premise.)

Plan change: none — approach offered as a candidate only. Two obligations
restated for the card: camera capture (`getUserMedia`) remains the card's
attended-verification leg (environmental, not spikeable headless), and if
adopted the 375 KB vendored file is committed + precached (the sibling goal's
spike priced exactly that precache move).
