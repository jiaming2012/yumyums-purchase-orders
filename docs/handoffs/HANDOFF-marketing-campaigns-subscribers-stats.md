# Handoff — Marketing · Campaigns, Subscribers, Stats (build the designed tabs)

**Task:** turn the three placeholder tabs in `marketing.html` (Campaigns / Subscribers / Stats) into the designed product, end to end: schema, Go endpoints, UI, tests.
**Handle:** HQ backlog **B-458** · roadmap **Activity H** (six cards) · ledger **T-60, decisions 187–191**
**Design of record:** Claude Design project *Yumyums HQ Marketing* — https://claude.ai/design/p/a8ffc065-b005-4020-bc6a-f42dc7e8f0e3 . Read the three **Current** pages (`Current Campaigns`, `Current Subscribers`, `Current Stats`) and `00 Brief`. The non-Current canvases are exploration history and are **not** the spec.
**Raised:** 2026-10-01, attended design sitting with the operator (this is the operator's ask, decomposed).
**Size:** one overnight slate, five parallel tracks + one serial tail. Honest estimate 9–11 h of agent time; the budget is a floor, not a ceiling.
**Blocking:** nothing hard for the code. Three external facts are **assumed and verified by the cards**, with a stated fallback each (§8).

---

## 1. Why this exists

The attribution cycle ("Close the loop") built the arbiter (A), the replica (B), the scanner (C) and the server machine (D). What a manager can *do* with that is still three "Soon" cards. The operator's framing, verbatim: *"the main point of the QR code is to make tracking, advertising, and gaining insights as easy as possible, with as few clicks and steps as possible"*, carrying *"at least campaign_id, channel, and [item_id]"*.

Two KRs are waiting on this surface: **P-KR3** (the orphan rate is visible and the loop is joinable) and **Q-KR2** (every accepted offline override is auditable and reconciled first). Both are graded against `marketing.html`.

## 2. The product, in one paragraph each

- **Campaigns.** A manager creates a campaign on one sheet — name, offer text, value, run length, optional menu item, and the channels it will be seen on. Saving mints **one QR code per channel**. Each code opens big with Share as the primary action. The list shows every campaign's funnel (scans → signups → redeemed) and its money (revenue, discount given, net, dollars back per $1 off). See design: Current Campaigns 1–8.
- **Subscribers.** Read-only list of everyone on the mailing list, where they came from (web form, Toast import, SMS keyword, or a campaign QR), consent state, visits. A detail sheet shows the identity-code status, the consent trail and the visit history. One write, **Resend QR**, which this handoff ships as a stub that records the request (the send itself is Activity E's). See design: Current Subscribers 1–2.
- **Stats.** Zero-setup. Overview = period, "needs a look" banner, the funnel with revenue / discount / net, reconciliation health (matched / open / declined, orphan rate against the 10% line), then three slices: by campaign, by channel, by item, each with the same money columns. The reconciliation queue (overrides → orphans → unmatched) is where a human fixes or **declines with a reason and a note**; declined items can be reopened. See design: Current Stats 1–8.

## 3. Decisions made at this sitting (engineer-level, stated so the night does not stall)

| # | Decision | Why |
|---|---|---|
| **187** | **Campaign admin lives in HQ Go + HQ Postgres.** `campaigns_admin`, `qr_codes`, `qr_scans`, `subscribers`, `toast_orders`, `reconciliation_decisions`, `scan_attempts_mirror` are HQ tables (migrations 0083+). The Supabase `campaigns` row is a **projection** written by the same handler (upsert by id over PostgREST with the service key: `name`, `face_value`, `requires_online`, `expires_at`-equivalent) so tablets keep replicating what they need and nothing else. This closes handoff §14 **#11**'s campaign-admin half. | Tablets need four columns offline; managers need twenty online. Putting the twenty in Supabase would widen the replica and RLS surface for no offline benefit. One writer, one projection, fail-loud: with `HQ_SYNC_REST_URL` unset the campaign saves with `projected_at NULL` and the UI shows a "not on tablets yet" pill — never a silent success. |
| **188** | **Reconciliation joins Toast order data read off the existing SFTP export**, not an SMTP mailbox. The Toast daily export directory HQ already syncs (`/<ExportID>/<YYYYMMDD>/ItemSelectionDetails.csv`) is the same export that carries `OrderDetails.csv` (`Order #`, `Opened`, `Discount Amount`, `Amount`, `Total`, `Voided`, `Paid`, `Closed`). Card H3 adds that file to the sync and lands `toast_orders`. Roadmap `smtp-toast-ingest` is **superseded**, and reverts only if the export provably lacks the file (§8). | HQ already has the key, the worker, the Spaces cache and the date loop (Phase 22). An inbox watcher would be a second ingest path for data we already pull. |
| **189** | **Payload = a short URL, record server-side.** QR encodes `https://hq.yumyums.kitchen/q/<short>`; `short` is 6 chars from `23456789ABCDEFGHJKLMNPQRSTUVWXYZ`. The record holds `campaign_id, channel (enum), item_id, placement, variant, landing, active, v`. Attribution is **first-touch**: a subscriber's `source_short` is the first code they scanned; redemptions inherit channel and item from it. UTM mirror when forwarding to the website. | Dense QRs don't scan from a truck sign; printed codes must be re-pointable; the enum is what makes "best channel" computable. |
| **190** | **Discount is implied until it is actual.** Implied = face value × accepted redemptions. Actual = Σ `toast_orders.discount_amount` over matched orders; when both exist and differ, both render. Net = revenue − discount (actual if present). **Declined attempts count in the orphan rate** except reason `duplicate_scan`. | The operator asked for the implied discount beside every revenue figure; "explained, not excused" keeps the orphan rate honest. |
| **191** | **B-436 fails closed.** A device that cannot construct a campaign policy source gets **no offline override** for any code (uniform with the source's own predicate); `campaigns-harness.mjs` leg 3's negative assertion moves with it. | The refusal is now armed on real data; a device with no policy at all is the one case left where a $40 code could burn offline. Total loss of override on that device is the lesser harm, and it is visible in `policy_unresolved`. |

Operator forks deliberately **not** taken (they change what a customer sees, so they are the operator's — defaults chosen so the night is not blocked):

- **Landing page** of a campaign QR: defaults to the website signup form with UTM + `q=<short>`. (Alternative: the menu with a "get the offer" button.)
- **Items per code:** one (nullable). (Alternative: several.)
- **Resend QR:** shipped as a recorded request, not a send.
- **Net after food cost:** not in scope; BI's per-dish COGS makes it a one-card follow-up.

## 4. Data model (HQ Postgres, goose migrations `0083_…` onward; one migration per card, Down included)

```sql
-- H1 ─ campaigns_admin: the manager-facing record. id == Supabase campaigns.id (projection key).
create table campaigns_admin (
  id              uuid primary key,
  slug            text not null unique,              -- "wing-wednesday", for UTM + payload preview
  name            text not null,
  offer_text      text not null,                     -- "$2 off any 6pc wings"
  face_value_cents integer not null check (face_value_cents >= 0),
  requires_online boolean not null,                  -- derived: face_value_cents >= marketing_settings threshold (#5)
  item_id         uuid null references menu_items(id),
  landing         text not null default 'signup' check (landing in ('signup','menu','offer','directions')),
  starts_at       timestamptz not null default now(),
  ends_at         timestamptz not null,
  status          text not null default 'live' check (status in ('scheduled','live','paused','ended')),
  projected_at    timestamptz null,                  -- last successful Supabase projection; NULL = "not on tablets yet"
  created_by      uuid not null references users(id),
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
);

create table qr_codes (
  id          uuid primary key default gen_random_uuid(),
  short       text not null unique check (short ~ '^[23456789ABCDEFGHJKLMNPQRSTUVWXYZ]{6}$'),
  campaign_id uuid not null references campaigns_admin(id),
  channel     text not null check (channel in ('truck_sign','flyer','table_tent','menu_board','instagram','google_ads','receipt','sms','other')),
  channel_label text null,                           -- required when channel='other'
  item_id     uuid null references menu_items(id),   -- defaults to the campaign's item
  placement   text null,
  variant     text null,
  landing     text null,                             -- NULL = inherit campaign.landing
  active      boolean not null default true,
  v           smallint not null default 1,
  created_by  uuid not null references users(id),
  created_at  timestamptz not null default now()
);

create table qr_scans (
  id          bigserial primary key,
  short       text not null references qr_codes(short),
  scanned_at  timestamptz not null default now(),
  ua_family   text null,                             -- coarse: ios / android / desktop / bot
  referrer    text null,
  ip_hash     text null,                             -- sha256(ip + daily salt); for 10-minute dedupe only
  subscriber_id uuid null                            -- set when the visit converts (H5 joins by source_short)
);
create index on qr_scans (short, scanned_at);

-- H3 ─ Toast orders (from OrderDetails.csv on the existing SFTP export) + attempts mirror + decisions
create table toast_orders (
  business_date   date not null,
  order_number    text not null,                     -- Toast "Order #", stored as text (format is #2's open question)
  order_id        text not null,
  opened_at       timestamptz not null,
  closed_at       timestamptz null,
  amount_cents    integer not null,                  -- "Amount" (pre-discount)
  discount_cents  integer not null default 0,        -- "Discount Amount"
  total_cents     integer not null,
  voided          boolean not null default false,
  order_source    text null,
  ingested_at     timestamptz not null default now(),
  primary key (business_date, order_number)          -- upsert key: the same report WILL arrive twice (§13)
);

create table scan_attempts_mirror (                  -- pulled from Supabase scan_attempts by the H3 poller (service key, keyset on scanned_at,id)
  id uuid primary key, code_id uuid not null, campaign_id uuid null, device_id text not null,
  scanned_at timestamptz not null, status text not null, reason text null,
  offline_override boolean not null, override_by text null, unverified_code boolean not null,
  policy_unresolved boolean not null default false,
  pos_order_number text null, pos_business_date date not null, redeemed_value numeric null,
  match_status text not null,                        -- as stored upstream: unmatched | matched | orphan
  mirrored_at timestamptz not null default now()
);

create table reconciliation_decisions (
  id          bigserial primary key,
  attempt_id  uuid not null references scan_attempts_mirror(id),
  decision    text not null check (decision in ('matched','declined','reopened','verified','rejected')),
  order_number text null,                            -- for 'matched'
  reason      text null check (reason in ('no_such_order','customer_left','comped','duplicate_scan','no_discount_applied','other')),
  note        text null,                             -- required when reason='other'
  decided_by  uuid not null references users(id),
  decided_at  timestamptz not null default now()
);
create index on reconciliation_decisions (attempt_id, decided_at desc);

-- H5 ─ subscribers
create table subscribers (
  id            uuid primary key default gen_random_uuid(),
  display_name  text null,
  phone_e164    text null unique,
  email         text null,
  source        text not null check (source in ('web_form','toast_import','sms_keyword','qr')),
  source_short  text null references qr_codes(short),  -- first-touch campaign code, when known
  sms_consent   boolean not null default false,
  email_consent boolean not null default false,
  consent_evidence text null,                        -- "form checkbox 2026-09-28" / "YES reply 2026-09-28"
  opted_out_at  timestamptz null,
  joined_at     timestamptz not null,
  external_ref  text null,                           -- Fluent Forms submission id / Toast guest id (idempotency)
  unique (source, external_ref)
);
create table subscriber_events (                     -- the detail sheet's timeline
  id bigserial primary key, subscriber_id uuid not null references subscribers(id),
  kind text not null check (kind in ('signed_up','code_sent','scanned','redeemed','resend_requested','opted_out')),
  at timestamptz not null default now(), ref jsonb null
);
```

Supabase: **one** migration, `supabase/migrations/20261002000100_campaigns_name_in_replica.sql` — no schema change if `name` already exists (it does, §4); the change is the **pull selection** in `marketing/sync/replicas.js` adding `name` (B-447). The projection writes only the four tablet columns.

## 5. API contract (`/api/v1/marketing/*`, behind `RequirePermission(pool,"marketing")`; **manager tier enforced inside the handler** per handoff §16 — a `team_member` gets `403 {"error":"managers_only"}`, which the UI renders as the designed Locked state)

| Method · path | Body / query | Returns |
|---|---|---|
| `GET /campaigns?period=30d` | period ∈ 7d/30d/90d/all | `{campaigns:[{id,slug,name,offer_text,face_value_cents,item:{id,name}|null,status,ends_at,projected_at,funnel:{scans,signups,redeemed},money:{revenue_cents,discount_cents,discount_basis:"implied"|"actual",net_cents,per_dollar},codes:[{id,short,channel,channel_label,scans,active}]}]}` |
| `POST /campaigns` | `{name,offer_text,face_value_cents,runs_days,item_id?,landing?,channels:[{channel,channel_label?,placement?}]}` | `201 {campaign, codes:[…]}` — mints one `qr_codes` row per channel in the same transaction; projects to Supabase after commit; `projected_at` null + `warnings:["not_projected"]` when unconfigured or failed |
| `GET /campaigns/{id}` | | campaign + codes (each with `scans, signups`) + `money` incl. `avg_order_cents_with`, `avg_order_cents_without` |
| `PATCH /campaigns/{id}` | `{status?:"paused"|"ended"|"live", name?, offer_text?, ends_at?}` | campaign |
| `POST /campaigns/{id}/codes` | `{channel,channel_label?,placement?,variant?,landing?}` | `201 code` |
| `PATCH /codes/{id}` | `{active?, landing?, placement?, variant?}` | code (re-point without reprint) |
| `GET /codes/{id}.png?size=1024` | | `image/png` of the short URL, quiet zone 4 modules; `Cache-Control: private, max-age=3600` |
| `GET /subscribers?q=&filter=all|sms|email_only|opted_out&source=` | | `{total,sms_opt_in,joined_this_week,rows:[{id,display_name,phone_last4,email_masked,source,source_short,campaign_name,joined_at,visits,consent:"sms"|"email_only"|"pending"|"stop"}]}` |
| `GET /subscribers/{id}` | | detail + `events[]` + `offers_now[]` |
| `POST /subscribers/{id}/resend` | | `202` — appends `resend_requested`; **no send** (Activity E) |
| `GET /stats/overview?period=` | | `{funnel:{scans,signups,codes_sent,redeemed},money:{…},reconciliation:{matched,open,declined,orphan_rate,threshold:0.10},needs_look:{overrides,orphans,unmatched}}` |
| `GET /stats/by?dim=campaign|channel|item&period=` | | rows with `scans,signups,redeemed,revenue_cents,discount_cents,discount_basis,net_cents,per_dollar`, plus `totals` |
| `GET /stats/by?dim=channel&campaign_id=` / `dim=code&item_id=` | | the drill-ins the design shows under a tapped row |
| `GET /reconciliation/queue` | | `{overrides:[…],orphans:[…],unmatched:[{…,suggestion:{order_number,opened_at,amount_cents}|null}],matched_count,declined_count}` |
| `POST /reconciliation/{attempt_id}/match` | `{order_number}` | decision; `409 order_not_found` if not in `toast_orders` |
| `POST /reconciliation/{attempt_id}/decline` | `{reason, note?}` | `400 note_required` when `reason="other"` and note empty |
| `POST /reconciliation/{attempt_id}/reopen` | | decision |
| `POST /reconciliation/{attempt_id}/verify` · `/reject` | `{note?}` | for offline overrides |
| `GET /reconciliation/declined` | | the declined bucket with reason, note, who, when |
| **public** `GET /q/{short}` | | `302` to the landing URL with `utm_source=qr&utm_medium=<channel>&utm_campaign=<slug>&utm_content=<item slug>&q=<short>`; logs `qr_scans` (HEAD and known link-preview UAs are not logged); inactive/ended → `200` static "This offer has ended" page; unknown → `404` |

All money is **cents, integers**; the UI formats. `per_dollar` is `revenue_cents / discount_cents` rounded to 2dp, `null` when discount is 0.

**Metric definitions (so the three slices agree):** scans = `qr_scans` rows after 10-minute `(short, ip_hash)` dedupe; signups = `subscribers` whose `source_short` resolves, by `joined_at`; codes sent = `subscriber_events.kind='code_sent'`; redeemed = `scan_attempts_mirror.status='accepted'` by `scanned_at`; revenue = Σ `toast_orders.amount_cents` over matched attempts; by-channel and by-item attribute each redemption to the subscriber's first-touch code. The by-item "Any item" row is campaigns with `item_id NULL`.

## 6. Cards (roadmap Activity H) — footprints, red-first tests, seams

Every card: red named on the pre-change tree, then green; atomic commits with `Night-Crew-Run:` trailer; `sw.js` regenerated **and committed** in the same change set whenever a precached file changes (B-13); precache count stated (today **48**; H2/H4/H5 each add one module → **51** if all land). `export PATH="/usr/local/go/bin:$PATH"` before any Go/Playwright leg; tests on `:5434` only; `go test ./... -p 1` with `DB_TEST_URL` set and counts checked.

| Card | Track | Footprint | Red-first test(s) | Seam / gate |
|---|---|---|---|---|
| **H1 `campaign-codes-api`** | A (backend) | `backend/internal/marketing/` (new: `campaigns.go, codes.go, landing.go, projection.go, qrpng.go`), `backend/internal/db/migrations/0083_campaigns_admin.sql`, `backend/cmd/server/main.go` (mount `/marketing/*` + public `/q/{short}`), `go.mod` (+`github.com/skip2/go-qrcode`), `backend/internal/db/db.go` (nothing new to seed — `marketing` slug exists) | `TestCreateCampaignMintsOneCodePerChannel`, `TestLandingLogsScanAndRedirectsWithUTM`, `TestLandingInactiveCodeRendersEndedPage`, `TestProjectionUnconfiguredLeavesProjectedAtNull`, `TestTeamMemberGets403ManagersOnly` | `backend/cmd/server` is an **undeclared seam → full Playwright suite** (deliberate); Go suite required |
| **H2 `campaigns-tab-ui`** | B (UI; builds against §5's shapes with a fixture server until H1 merges, then switches) | `marketing.html` (s2), `marketing/campaigns.js` (new), `tests/marketing-campaigns.spec.js` (new), `tests/states-marketing-campaigns.spec.js` (new; the State Enumeration Table rows below), `sw.js`, `night-crew.toml` (+2 seam rows) | `[MC-01] create sheet mints N codes and lands on "N codes ready"`, `[MC-02] list renders funnel + money strip`, `[MC-03] code sheet Share calls navigator.share with the PNG file`, `[MC-04] team_member sees Locked state`, `[MC-05] offline shows last-synced list + disabled create` | `marketing` seam → `tests/marketing*.spec.js`; `sw.js` de-confines to full suite |
| **H3 `toast-orders-and-reconciliation`** | C (backend) — absorbs roadmap `reconciliation-view` + `smtp-toast-ingest` + **B-424** | `backend/internal/toast/` (+`orderdetails.go`: parse + upsert; sync fetches `OrderDetails.csv` beside ItemSelection), `backend/internal/marketing/reconciliation.go`, `…/mirror.go` (scan_attempts poller, 5 min, keyset), `backend/internal/db/migrations/0084_toast_orders_reconciliation.sql`, `backend/internal/redemption/store.go` (unique index for B-424 dedupe) | `TestOrderDetailsUpsertIsIdempotent` (same file twice → one row), `TestQueueOrdersOverridesThenOrphansThenUnmatched`, `TestDeclineOtherRequiresNote`, `TestOrphanRateCountsDeclinesExceptDuplicateScan`, `TestRaceLostNotificationDedupe` | Go suite; `backend/internal/toast` undeclared → full suite |
| **H4 `stats-tab-ui`** | D (UI; fixture-first like H2) | `marketing.html` (s4), `marketing/stats.js` (new), `tests/marketing-stats.spec.js`, `tests/states-marketing-stats.spec.js`, `sw.js`, `night-crew.toml` | `[MS-01] overview renders funnel + revenue/discount/net + orphan rate with the 10% marker`, `[MS-02] by-item rows carry discount and Per $1`, `[MS-03] decline sheet requires a reason and posts reason+note`, `[MS-04] declined bucket shows note and Reopen`, `[MS-05] empty period renders "No redemptions yet"` | `marketing` seam |
| **H5 `subscribers-tab`** | E (full-stack) | `backend/internal/marketing/subscribers.go` (+ `sources/fluentforms.go` behind `FF_DB_*` env, `sources/toastguests.go` CSV upload, `sources/qr.go` joining `source_short`), `backend/internal/db/migrations/0085_subscribers.sql`, `marketing.html` (s3), `marketing/subscribers.js`, `tests/marketing-subscribers.spec.js`, `tests/states-marketing-subscribers.spec.js`, `sw.js`, `night-crew.toml` | `TestFluentFormsImportIsIdempotent`, `TestSubscriberSourceShortSetsCampaignAttribution`, `[SB-01] list masks phone to last 4`, `[SB-02] STOP renders opted-out row`, `[SB-03] sheet shows identity-code status + timeline`, `[SB-04] Resend records an event and sends nothing` | `marketing` seam |
| **H6 `scanner-polish`** | F (serial tail — touches `marketing/sync/*`, which H3's mirror reads; land last) — **B-446, B-447, B-440, B-436** | `marketing/sync/replicas.js` (+`name` in the campaigns pull; **B-436 fail-closed**), `marketing/sync/push-replication.js:303` (divert predicate `unverified_code && offline_override`), `marketing/submit-flow.js` + `scan-page.js` (B-446 refusal at scan-resolve; B-447 campaign name + code last-4 on the offer card), `marketing/sync/harness/campaigns-harness.mjs` (leg-3 assertion flips with B-436), `tests/marketing.spec.js`, `sw.js` | `[SP-01] offline + requires_online renders the refusal before any tap`, `[SP-02] offer card shows campaign name and code ····4821`, `f2-run.sh` poison-row case, `[SP-03] no policy source → no override affordance` | `marketing` seam + harness exit codes |

**Parallelism:** A ∥ B ∥ C ∥ D ∥ E; F after C. B and D start on fixtures and must switch to the real endpoints before their final gate (the merge-intent states which).

**State Enumeration Table rows every UI card must screenshot** (`tests/states-marketing-*.spec.js`, read the PNGs back): empty · loading · error (API 500 → red banner + retry) · success · **locked (403 managers_only)** · **offline (last synced + disabled writes)** · **not projected** (campaign saved, `projected_at` null → pill) · **long content** (offer text > 40 chars truncates with full text on tap).

## 7. Backlog folded in

| Item | Where it lands |
|---|---|
| B-424 (race_lost dedupe + F4 status bullet unowned) | H3 — unique index + status bullet owned |
| B-440 (F-2 divert predicate ≠ constraint) | H6 |
| B-446 (refuse at scan time, not after Submit) | H6 |
| B-447 (offer card names a UUID) | H6 (+ `name` in the replica selection) |
| B-436 (policy source fails to construct → fails open) | H6, **fail-closed** (decision 191) |
| roadmap `reconciliation-view` (Activity F) | absorbed by H3 + H4 |
| roadmap `smtp-toast-ingest` (Activity F) | superseded by H3 (decision 188), kept as fallback |

## 8. What I could not fill in — assumed, verified by the card, with a fallback

- **`OrderDetails.csv` is on the SFTP export.** Toast's standard daily export includes it next to `ItemSelectionDetails.csv`; the repo's own `OrderDetails.csv` sample has the columns §4 relies on. **H3 lists the remote date directory first** (read-only) and records what it found in its merge-intent. If the file is absent, H3 lands `toast_orders` + the reconciliation engine against a **fixture loader** (`cmd/sync-toast --orders-csv <path>`) and `smtp-toast-ingest` returns to PLANNED.
- **Toast `Order #` format** (handoff #2) — stored as text, matched exactly, with the ±30-minute nearest-order suggestion as the safety net. No format assumption baked in.
- **Fluent Forms credentials** (`FF_DB_HOST/USER/PASSWORD/NAME/PREFIX`, the Hostinger MySQL behind `website/scripts/count_customers.py`). H5 ships the reader behind those env vars and a committed fixture of the submission JSON shape; the live import is an **attended** first run (operator holds the creds).
- **Supabase hosted project** — still Activity 0's. Every card runs against the local `spike-supabase` substrate in reconcile mode (`-p yumyums-test`, never `--fresh`, never `:5433`).
- **The welcome offer's face value** for "free side" style offers — the create sheet makes Value explicit precisely because the discount math needs it; nothing is inferred from offer text.
- **Share on iOS** — `navigator.share({files})` is supported on iOS 15+; H2 asserts the call and falls back to Save PNG when `canShare` is false.
