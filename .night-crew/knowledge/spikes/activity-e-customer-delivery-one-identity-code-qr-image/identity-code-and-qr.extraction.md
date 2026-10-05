# Extraction — identity-code-and-qr

Outcome: confirmed with one agent-reached correction (the encoding), signed

Approach used: a throwaway worktree of `dev` with a throwaway `go-qrcode` generator
built inside the backend module, writing seven server-generated hybrid QR PNGs into the
worktree's `tests/fixtures/`, and a parametrized Playwright spec run through the SHIPPED
`marketing.html` on a spike-owned e2e stack (`:5434`, port 8331) — offline with nothing
seeded per variant, then online with the sync door mocked and the server lookup
answering a live row; plus five `curl` legs against the LOCAL spike-supabase substrate
as `service_role` and as an `authenticated` device. Tool-recorded runs: spike 01 exit 1
(finding: Medium/234 chars = version 11 is undecodable on the 320 px file surface),
exit 1 (spec bug: a corrupted decode chosen as "decodable"), exit 0; spike 02 exit 0
three times. Candidate input for the card, not an adoption (NFR-6).

Confirmed: (a) the shipped reader decodes a server-made hybrid QR and renders
`embeddedOffer` with the descriptor's label and `data-token-hash` = SHA-256(token) when
the code is at version ≤ 10 (57 modules) — the card's encoding (16-char token, the same
four descriptor keys, date-only `expires_at`, Low correction: 214 chars, version 9, 53
modules) decodes offline and resolves `offerReady` from the server online; (b) version
11 (Medium, 234 chars) does NOT decode on the 320 px `#scan-file-surface` at any pixel
size, and a 1024 px copy decoded to a CORRUPTED payload (`invalidPayload`); (c) a
`codes` row upserted over PostgREST as `service_role` with `Prefer:
resolution=merge-duplicates` is visible to an `authenticated` device through the
tablet's own offers-pull filter under `codes_select_device`; re-upsert is idempotent;
delete reconciles.

Learned: (1) the file-scan path's bound is module density on a 320 px surface, not
image size — the camera path has its own optics, so the card encodes with headroom
(version 9) AND widens the surface to 640 px; (2) the decode boundary is not monotonic
(version 8 failed, 9 and 10 passed in the same run), so no encoding near the edge is
trusted; (3) Low correction is the right level for a phone-screen image and buys one
version; (4) a 16-character token (80 bits, hashed on-device) is enough identity and
saves seven characters; (5) the projection seam for identity codes already exists —
`projection.go`'s allowlisted PostgREST client — so the card adds `ProjectIdentityCode`
beside `ProjectCampaign`, no new egress; (6) the `codes.expires_at` NOT NULL means an
identity code's entitlement row carries the campaign's `ends_at`.

Plan change: the card's encoding is fixed to the measured one above (not Medium, not the
original token length); the card gains a one-line reader widening (`#scan-file-surface`
640 px, `sw.js` regenerated, count 51); the descriptor contract and `parseEmbeddedOffer`
stay as shipped. Red-first recipes: the spike's variant spec (re-homed as `[IC-*]`) with
`full-M-512` as the negative control, and spike 02's five legs as the projection test's
shape against a mocked PostgREST.
