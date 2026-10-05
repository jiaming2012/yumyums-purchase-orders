# Spikes — identity-code-and-qr

Activity: Activity E — Customer delivery (one identity code → QR → image)

> Tool-run (`night-crew spikes run`). Spike 1 generates a hybrid QR PNG with the repo's own
> `go-qrcode` in a THROWAWAY worktree off `dev` (`../hq-worktrees/spike-e1-20261006`) and decodes
> it through the SHIPPED scanner page on a spike-owned e2e stack (`hq_test_spike_e1_20261006`,
> `TEST_PORT=8331`, `:5434`). Spike 2 writes ONE `public.codes` row to the LOCAL `spike-supabase`
> substrate as `service_role` and reads it back as an `authenticated` device, then deletes it.
> Never `:5433`; never a hosted project.

## The goal, and which legs need a spike

The card (E1 — roadmap Activity E, D-KR3, handoff §10 + #10): one permanent identity code per
customer, whose QR is a URL wrapping the identity token PLUS an embedded self-describing offer,
readable offline before the customer's entitlements have replicated, resolving the full
server-side list online. The scanner half is SHIPPED and gated (`tests/marketing.spec.js` —
`parseEmbeddedOffer`, the `embeddedOffer` and `offerReady` results); nothing today GENERATES a
payload or an identity-code row. Two premises a script can settle now: (1) a server-generated
hybrid QR — a ~190-character URL with a base64url descriptor in the fragment — encodes at a
phone-scannable version and the shipped reader decodes it from a 512 px PNG, offline to the
embedded offer and online to the server's list; (2) an identity-code row upserted into the
arbiter's `codes` table by the projection path HQ already has (`projection.go`, service_role
over PostgREST) is visible to a device through the tablet's own offers-pull filter under the
shipped RLS, and the upsert is idempotent on re-import.

## Spike: hybrid-qr-encodes-and-the-shipped-scanner-reads-it

- proves: (a) `go-qrcode` encodes `https://hq.yumyums.kitchen/r/<token>#o=<base64url JSON>`
  (label, campaign_id, expires_at, face_value) — version and module count recorded per
  variant; (b) through the SHIPPED page, OFFLINE with nothing seeded, a photo of a
  server-generated PNG decodes to the token's hash and, for the full JSON descriptor, renders
  `embeddedOffer` with the label; (c) ONLINE with the server lookup answering a live row for
  that hash, a decodable variant renders `offerReady` from source `server`.
  **Amended after the first run (exit 1, below):** the 234-char payload at Medium/512 is
  version 11 and the shipped reader answered `decodeError` — html5-qrcode draws a picked file
  onto the hidden 320 px `#scan-file-surface`, so module density is the bound, not pixel size.
  The second run measures SIX encodings in one stack boot (full descriptor at Medium 512 /
  Medium 1024 / Low 512 / Low 512 without quiet zone; a compact descriptor that keeps
  `campaign_id` at Medium / Low) and passes when at least one FULL-descriptor variant decodes
  offline to `embeddedOffer` and one variant resolves online — the variant table is the build
  fact the card encodes to.
- plan: worktree, warm build, `go run` a throwaway generator inside the worktree's backend
  module per variant (writing `tests/fixtures/spike-e1-<variant>.png` and a variants manifest),
  copy a parametrized Playwright file into the worktree's `tests/`, run it on the spike stack,
  read the per-variant lines and the online outcome from the log and JSON reporter.
- script: .night-crew/spikes/activity-e-customer-delivery-one-identity-code-qr-image/identity-code-and-qr/01-hybrid-qr-encodes-and-the-shipped-scanner-reads-it.sh

### Runs

- 2026-10-05T13:12:03Z · exit 1 · failed
- 2026-10-05T13:18:42Z · exit 1 · failed
- 2026-10-05T13:22:53Z · exit 0 · passed

## Spike: identity-code-row-projects-and-the-device-pull-sees-it

- proves: against the local substrate: (a) BEFORE, a device JWT's filtered pull for the spike
  hash returns `[]`; (b) a `service_role` POST to `/codes` with `Prefer:
  resolution=merge-duplicates` lands (201/200) a row {id, token_hash, campaign_id = the seeded
  TEST campaign, expires_at 2028}; (c) AFTER, the tablet's own offers-pull query
  (`expires_at=gt.<now>&order=updated_at.asc,id.asc&limit=…`) as `authenticated` contains the
  hash — `codes_select_device` lets the device see it; (d) a second identical upsert leaves
  exactly one row for the hash (idempotent re-mint); (e) the row is deleted as `service_role`
  and the device pull is `[]` again (reconcile, never leave a spike row behind).
- plan: REST port from `docker compose port`, JWT secret from the compose file, two tokens from
  `mintjwt` (service_role / authenticated), five `curl` legs with exit codes and bodies printed.
- script: .night-crew/spikes/activity-e-customer-delivery-one-identity-code-qr-image/identity-code-and-qr/02-identity-code-row-projects-and-the-device-pull-sees-it.sh

### Runs

- 2026-10-05T13:14:46Z · exit 0 · passed
- 2026-10-05T13:21:12Z · exit 0 · passed
- 2026-10-05T13:25:23Z · exit 0 · passed

## Verdict (tool-run 2026-10-05)

- **hybrid-qr-encodes-and-the-shipped-scanner-reads-it: passed** on its third run (exit 0,
  150 s). The two failed lines above are a finding and a spec bug, named: **run 1** — the
  234-char payload (full JSON descriptor, `go-qrcode` Medium, 512 px) is **version 11, 61
  modules**, and the shipped reader answered `decodeError` on it, because `html5-qrcode` draws a
  picked file onto the hidden **320 px** `#scan-file-surface`, so module density is the bound
  and pixel size is irrelevant; **run 2** measured six encodings (table below) and its online
  leg chose the first variant whose kind was not `decodeError` — which was the 1024 px image's
  `invalidPayload`, a CORRUPTED decode — a spec bug, fixed to require the token's hash. **Run 3**
  added the card's own encoding and passed. Variant table (offline, shipped reader, `dev@7d252ce`):

  | variant | payload | level | version | modules | result |
  |---|---|---|---|---|---|
  | **card-L-512** — 16-char token, same descriptor keys, `expires_at` date-only | 214 | L | **9** | **53** | **embeddedOffer, hash ok** |
  | full-M-512 — 23-char token, full JSON | 234 | M | 11 | 61 | decodeError |
  | full-M-1024 — same at 1024 px | 234 | M | 11 | 61 | **invalidPayload** (corrupted decode) |
  | full-L-512 | 234 | L | 10 | 57 | embeddedOffer, hash ok |
  | full-L-nb — no quiet zone in the PNG | 234 | L | 10 | 57 | embeddedOffer, hash ok |
  | compact-M-512 — short-key descriptor the reader cannot parse | 162 | M | 9 | 53 | unknownCode, hash ok (decoded) |
  | compact-L-512 | 162 | L | 8 | 49 | decodeError |

  Online: `card-L-512` → `offerReady`, source `server`, hash
  `204933dc…b10ece1` = SHA-256(`ic7k3m9q2x5p8w4z`). The reader's boundary on the 320 px surface
  sits at ~57 modules and is NOT monotonic (a version-8 variant failed while 9 and 10 passed),
  so the card encodes with headroom and widens the surface rather than trusting the margin.
- **identity-code-row-projects-and-the-device-pull-sees-it: passed** three times (exit 0, ~1 s):
  device pull `[]` before; service_role upsert `201` (row `e1000000-…-0001`, the TEST campaign,
  expires 2028); the tablet's offers-pull query as `authenticated` returned the hash among 5 live
  rows; re-upsert `200` with exactly 1 row for the hash; delete `204`, device sees `[]`.

## Corrections

- **One, agent-reached, carried into the card:** the premise "the hybrid payload encodes at
  Medium like campaign codes" is wrong for the shipped reader. The card's encoding is **Low
  correction, a 16-character identity token, the SAME descriptor keys `parseEmbeddedOffer`
  already reads (`label`, `campaign_id`, `expires_at`, `face_value`) with a date-only
  `expires_at`** — 214 chars, version 9, 53 modules — and the card ALSO widens
  `#scan-file-surface` from 320 to 640 px (a one-line `marketing.html` change, precached → `sw.js`
  regenerated, count stays 51) so a version ≤ 11 photo decodes with headroom. Low correction is
  acceptable because the image lives on a phone screen, never on a damaged sign. No descriptor
  contract change: the reader is untouched.
- Precision: the corrupted `invalidPayload` decode from a dense image is a reader behaviour the
  card's spec must assert does not occur for its own encoding (a decode that yields a wrong hash
  must never render an offer — it does not today; `invalidPayload` renders nothing).

## Comebacks

- Spike-side: the online leg's "first decodable variant" must be defined by hash match, never by
  `kind !== decodeError` — a corrupted decode is a non-decode. Fixed in run 3.
- Reader-side (not this card's, filed for the roadmap round): the file-scan surface's 320 px
  bound and the non-monotonic failure around version 8–11 are worth a `[CS-*]` density spec once
  identity codes are in the field.

## Review

- signed: operator, 2026-10-05 — covers 1 correction(s) (reviewed at the slate sitting of
  2026-10-05, §4 batch sign-off: the encoding correction and the 640 px surface widening were on
  screen with the variant table).
