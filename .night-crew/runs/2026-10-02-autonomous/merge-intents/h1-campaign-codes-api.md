# Merge intent — Card H1 · `campaign-codes-api`

Run `20261002` · branch `card/h1-campaign-codes-api` off `overnight-20261002` at `c4f6db4`
Wave 0 — this card creates `backend/internal/marketing/` and must merge before H2/H3a/H4/H5/H6.

## Shared files touched

| File | Why |
|---|---|
| `backend/cmd/server/main.go` | THREE call sites (see below). Undeclared seam in `night-crew.toml` → this card owes the FULL Playwright suite. |
| `backend/go.mod` / `backend/go.sum` | `+ github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e` (the one new dependency; the encoder spike 01 proved). |
| `backend/internal/db/migrations/0083_campaigns_admin.sql` | New file, next free number. If another card claims 0083 first, this one renumbers — nothing else references the filename. |
| `.night-crew/knowledge/roadmap.md` | H1's card line flips PLANNED → LANDED. Every card edits this file; expect a conflict and take both sides. |
| `.night-crew/runs/2026-10-02-autonomous/merge-intents/h1-campaign-codes-api.md` | This note. |

**Nothing else.** No frontend file, no `sw.js`, no `night-crew.toml` key, no `backend/internal/db/db.go`
(the `marketing` app slug is already seeded, and 0083 adds no grant).

### The three `main.go` call sites

The brief said "TWO calls". It is three, because the public landing cannot live inside either
gated block. Named here so the orchestrator can see all of them at merge:

1. `marketing.Mount(r, mktDeps)` — inside the existing `r.Route("/api/v1/marketing", …)` block
   that already carries `auth.RequirePermission(pool, "marketing")` and `redemption.SubmitHandler`.
   **This is the seam Cards 3/4/6 append into** — they add handlers to `routes.go` inside the
   package, not to `main.go`.
2. `marketing.MountReports(r)` — a NEW `r.Group` at the `/api/v1` level carrying
   `auth.RequirePermission(pool, "bi")`, sited immediately after the `r.Route("/inventory", …)`
   block that holds the existing `bi` groups (`/trends`, `/cost`). It is a **no-op today**
   (`TestMountReportsIsANoOpSeamToday` asserts it registers zero routes). It is NOT inside the
   `/inventory` Route block, because routes registered there would be prefixed `/api/v1/inventory/*`
   and decision 192's contract is `/api/v1/bi/campaigns/{overview,by}` — the `bi` GRANT is what
   the slate meant by "the BI route block", and that is what this group carries. H3b fills the
   seam from inside the package.
3. `marketing.MountPublic(r, mktDeps)` — on the ROOT router, before `r.Handle("/*", staticHandler(...))`.
   `GET`/`HEAD /q/{short}`, outside `/api/v1`, outside `auth.Middleware`, outside every
   `RequirePermission`. A customer with a phone camera has no HQ session.

## What must survive any merge

- **`backend/internal/marketing/` package shape.** `Deps` (Pool / Projection / QRBaseURL /
  LandingBaseURL), `Mount(r chi.Router, d Deps)`, `MountReports(r chi.Router)`,
  `MountPublic(r chi.Router, d Deps)`, `NewDeps(pool)`. `routes.go` is the one route table;
  later cards add rows to it, not to `main.go`.
- **All three `main.go` call sites** and the `mktDeps := marketing.NewDeps(pool)` line above them.
- **Migration 0083** — §4's H1 DDL verbatim (`campaigns_admin`, `qr_codes`, `qr_scans`), Down
  included. `qr_codes.short` is the key H5's `0085_subscribers.sql` references
  (`subscribers.source_short → qr_codes(short)`), so 0083 must sort before 0085.
- **`go.mod`/`go.sum` go-qrcode entry.**
- **The `managers_only` envelope** — `403 {"error":"managers_only"}` from every gated handler.
  H2's Locked state row asserts exactly this string.

## What is safe to drop

- The `TestPayloadStaysAtQRVersion3` / `TestUAFamily` / `TestIsLinkPreview` / `TestSlugify`
  hermetic units (hygiene around the night's own decisions, not done_when rows) — if one of
  them conflicts with a later card's refactor, the later card's version wins.
- `TestMountRegistersTheGatedRouteTable`'s expected-route list: Cards 3/4/6 ADD rows to `Mount`,
  and the assertion is "contains", not "equals" — but if a later card renames a route, update
  the list rather than preserving this card's spelling.
- The `money` zero block (below) is H3b's to replace outright.

## Deliberately NOT in this card (shipped as a stated zero shape)

- **`money` is H3b's.** `GET /campaigns` and `GET /campaigns/{id}` ship the block as the literal
  zero shape the slate names: `{revenue_cents:0, discount_cents:0, discount_basis:"implied",
  net_cents:0, per_dollar:null}` (plus `avg_order_cents_with/without: null` on the detail route).
  The arithmetic needs `toast_orders` + `scan_attempts_mirror`, which are migration 0084's
  (H3a). H2 renders whatever shape is there, zero shape included.
- **`funnel.signups` and `funnel.redeemed` are 0 for the same reason** — signups need
  `subscribers` (0085, H5) and redeemed needs `scan_attempts_mirror` (0084, H3a).
  `funnel.scans` IS real and comes from `qr_scans`.

## Decisions this card made that the slate left to it

- **Short-code alphabet / generator:** `23456789ABCDEFGHJKLMNPQRSTUVWXYZ` (decision 189's 32
  chars), 6 chars, `crypto/rand` with rejection sampling so the draw is uniform (`256 % 32 == 0`,
  so a plain modulo is already uniform here, but the rejection loop is kept so a future alphabet
  change cannot silently bias it). Collisions retry up to 8 times inside the same transaction
  against the `qr_codes_short_key` unique violation.
- **Scan dedupe is enforced at WRITE time, not read time.** The landing handler skips the INSERT
  when a row for the same `(short, ip_hash)` exists within 10 minutes, so §5's "scans =
  `qr_scans` rows after 10-minute `(short, ip_hash)` dedupe" is a plain `count(*)` on read. One
  definition, in one place — a second read-side rule is how two slices stop agreeing.
- **`ip_hash` = `sha256(ip + "|" + YYYY-MM-DD(UTC) + "|" + salt)`**, salt from
  `HQ_QR_IP_SALT` or the default literal `hq-qr`. The DDL comment scopes this hash to dedupe
  only, so the salt is a rotation knob, not a secret.
- **Link-preview UA list** (not logged, still redirected): `facebookexternalhit`, `facebot`,
  `twitterbot`, `slackbot`, `slack-imgproxy`, `discordbot`, `whatsapp`, `telegrambot`,
  `linkedinbot`, `pinterest`, `skypeuripreview`, `redditbot`, `applebot`, `googlebot`,
  `bingbot`, `yandexbot`, `embedly`, `quora link preview`, `vkshare`, `w3c_validator`,
  `developers.google.com/+/web/snippet`. Matched case-insensitively as substrings.
- **PNG size ladder:** `{256, 512, 1024, 2048}`, default `1024`. Off-ladder sizes are a
  `400 {"error":"bad_size","allowed":[…]}` rather than a silent snap — a print job that asked
  for 900px and got 1024 is a surprise at the printer.
- **`requires_online` threshold source.** `marketing_settings` is a **Supabase** table (RLS on,
  no policies, no client grants — `supabase/migrations/20260904000100`); HQ Postgres has no copy
  and this card does not add one. So the handler reads
  `requires_online_threshold_cents` over PostgREST as service_role when the projection is
  configured, with a 3s budget, and falls back to `DefaultRequiresOnlineThresholdCents = 2000`
  (the substrate's own seeded default, `$20.00`) when it is unset or unreachable — logged at WARN,
  never silent. Creating a campaign is never blocked by an unreachable substrate.
- **Landing base URL.** `HQ_MARKETING_LANDING_BASE_URL`, default `https://yumyums.kitchen`;
  `landing` ∈ `signup|menu|offer|directions` maps to `/signup|/menu|/offer|/directions`.
  The TARGET is the operator's stated default (the website signup form with UTM + `q=`) — only
  the host is configurable, which is mechanism, not the parked fork.
- **The projection writes exactly four columns** — `id, name, face_value, requires_online` —
  because that is the column list Supabase `public.campaigns` actually has (plus its own
  `updated_at` default). Decision 187's prose says "`expires_at`-equivalent"; there is no such
  column on `campaigns` (expiry lives on `codes`), and spike 02's proven upsert used these four.
  `face_value` is numeric DOLLARS upstream, so cents are divided by 100 on the way out.
- **PATCH re-projects.** `name` is one of the four projected columns, so a renamed campaign
  re-upserts and re-stamps `projected_at` (same fail-loud rule: a failed re-projection leaves
  the previous `projected_at` alone and returns `warnings:["not_projected"]`).
- **The manager tier gates reads too,** not only writes. The designed Locked state covers the
  whole Campaigns tab, so every handler `Mount` registers — `GET` included — answers a
  `team_member` with `403 managers_only`.

## Red-first

The red is **structural**, exactly as the slate predicted: the five done_when tests reference a
package that has no non-test source on the base tree. Captured BEFORE any implementation file
existed, at
`.night-crew/runs/2026-10-02-autonomous/logs/h1/RF-red-first.log`
(commit `<tests commit>`; the log is committed in the same change set).

Observed, from `backend/` with `PATH=/usr/local/go/bin:$PATH` and
`DB_TEST_URL=postgres://hqtest:hqtest@localhost:5434/hq_test_go_h1?sslmode=disable`:

```
$ go test -count=1 -run 'TestCreateCampaignMintsOneCodePerChannel|TestLandingLogsScanAndRedirectsWithUTM|TestLandingInactiveCodeRendersEndedPage|TestProjectionUnconfiguredLeavesProjectedAtNull|TestTeamMemberGets403ManagersOnly' ./internal/marketing/
# github.com/yumyums/hq/internal/marketing [github.com/yumyums/hq/internal/marketing.test]
internal/marketing/helpers_test.go:112:35: undefined: Deps
internal/marketing/helpers_test.go:123:19: undefined: Deps
internal/marketing/helpers_test.go:175:88: undefined: createCampaignResponse
internal/marketing/projection_test.go:189:41: undefined: ProjectionConfig
internal/marketing/campaigns_test.go:155:63: undefined: DefaultRequiresOnlineThresholdCents
internal/marketing/campaigns_test.go:247:10: undefined: listCampaignsResponse
internal/marketing/campaigns_test.go:290:10: undefined: codeDTO
internal/marketing/campaigns_test.go:317:10: undefined: campaignDTO
internal/marketing/campaigns_test.go:317:10: too many errors
FAIL	github.com/yumyums/hq/internal/marketing [build failed]
TEST_EXIT=1
```

```
$ go vet ./internal/marketing/
vet: internal/marketing/helpers_test.go:112:35: undefined: Deps
VET_EXIT=1
```

Note the honest asymmetry in that log: **`go build ./...` exits 0** on the red tree, because
`_test.go` files are not part of the non-test build and the package therefore has no buildable
source at all. The red is at `go vet` and `go test`, and that is where it is claimed. Naming
`go build` as the red would have been wrong.

Green-after is recorded in `logs/h1/G2-go.log` (counts, not `ok`).
