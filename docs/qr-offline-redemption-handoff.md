# Yumyums — Offline QR Redemption System

**Handoff document**
Status: design agreed, not yet built
Last updated: 2026-09-03

---

## 1. Why this exists

The origin is a marketing attribution gap, not a scanning problem.

The Google Ads account currently records ~496 local actions across six goals — 319 "other engagements," 81 menu views, 75 direction requests, 19 website visits, 2 orders, 0 calls. None of it ties to revenue. A direction request is the best available proxy for someone walking up to the truck, and there is currently no way to confirm whether any of those 75 people actually appeared.

A scannable code that a customer presents at the window, redeemed against a Toast order, closes that loop. Redemption is the only event in the funnel that is simultaneously (a) attributable to a campaign and (b) tied to real money.

Secondary goals: a durable customer identity across visits, and a measurable repeat rate.

---

## 2. Constraints that shape everything

- **The truck operates on an LTE hotspot.** Connectivity is unreliable at the exact moment of service. This is the central design constraint.
- **POS is Toast**, not Square. Toast has no self-serve transactional messaging API.
- **Three devices** at the window, standing near each other, sharing one flaky connection.
- **Single-use codes carry real money.** Convergence is not correctness.

---

## 3. Stack decision

**RxDB (local-first) replicating against Supabase (Postgres + Realtime).**

Rationale for the hybrid rather than either alone:

| | Alone | Problem |
|---|---|---|
| RxDB only | Local writes, offline-durable | No cross-device propagation without building replication; LWW cannot enforce single-use |
| Supabase only | Realtime propagation, atomic Postgres arbitration | Server-first — a write fails outright when the hotspot drops |
| **Both** | Local-first writes + Realtime pull stream + Postgres as arbiter | More moving parts; the tradeoff we accept |

### RxDB propagation facts worth knowing

- A local write propagates **within that device only**. RxDB uses BroadcastChannel with leader election, so multiple tabs on the same origin see each other's writes immediately. Separate physical devices see nothing without replication.
- `.$` subscriptions update instantly within the instance. That is local reactivity, not sync.
- Cross-device options: `replicateRxCollection` with push/pull handlers (chosen), or the WebRTC plugin for peer-to-peer on the truck's local network (still needs a signaling server).

---

## 4. Data model

Two collections with **opposite replication directions**. This is the key structural decision — mutating a single `codes` collection locally means fighting your own replication over who owns `redeemed_by`.

- `codes` — server-owned, **pull only**
- `scan_attempts` — device-owned, **push only**

```sql
create table campaigns (
  id              uuid primary key,
  name            text not null,
  face_value      numeric not null,
  requires_online boolean not null default false,  -- see §8
  updated_at      timestamptz not null default now()
);

create table codes (
  id            uuid primary key,
  token_hash    text not null unique,   -- never store the raw token
  campaign_id   uuid not null references campaigns(id),
  expires_at    timestamptz not null,
  redeemed_at   timestamptz,
  redeemed_by   text,
  updated_at    timestamptz not null default now(),
  _deleted      boolean not null default false
);
create index on codes (updated_at);

create table scan_attempts (
  id                uuid primary key,        -- generated on device
  code_id           uuid not null,
  device_id         text not null,
  scanned_at        timestamptz not null,
  status            text not null default 'pending',  -- pending | accepted | rejected
  reason            text,                             -- already_used | expired | not_found
  offline_override  boolean not null default false,   -- submitted offline via permissioned override (§13)
  override_by       text,                             -- who authorized the offline override
  unverified_code   boolean not null default false,   -- override on a code not in the replica (§19 F2)

  -- manual join key to Toast (see §13)
  pos_order_number  text,
  pos_business_date date not null,
  redeemed_value    numeric,
  match_status      text not null default 'unmatched' -- unmatched | matched | orphan
);
create index on scan_attempts (pos_business_date, pos_order_number);
```

**`token_hash`, not the raw token.** Three tablets hold a full replica of the code table. Hashing the scanned value with WebCrypto on-device means a dumped replica does not yield a list of live redeemable codes.

**`updated_at` index matters more than it looks** — it is the replication checkpoint key and gets hit on every pull tick.

---

## 5. The three offline requirements

### 5.1 Detect expiry with no network

Falls out of the model for free. `expires_at` is already on the replicated row, so the check is a local comparison — no lookup, no round trip.

**Caveat: device clock.** An offline expiry check trusts the tablet's clock, and a tablet with a wrong date silently accepts dead codes. On every successful sync, store `serverNow - deviceNow` and apply that offset in the comparison. Cheap, and it catches both drift and casual tampering.

### 5.2 Cache redemption state locally

A successful scan writes an attempt row with `status: accepted`. The authoritative `redeemed_at` comes back down on the next `codes` pull. Both persist in RxStorage, so a device that goes offline an hour later still knows the code is spent.

### 5.3 Propagate state to the other clients

Same mechanism, no extra work. Device A redeems → Postgres sets `redeemed_at` → `updated_at` bumps → B and C pick it up on their next pull tick or Realtime nudge. Bound the replica with a pull filter (`expires_at > now() - interval '2 days'`) so the local set stays small — a few hundred rows for a truck.

---

## 6. Server arbitration

A single conditional `UPDATE` is atomic in Postgres. This is a real mutex, not eventual convergence, and it is the **only** thing actually enforcing single use.

```sql
create function redeem(p_code uuid, p_device text)
returns table (ok boolean, reason text) language plpgsql as $$
begin
  update codes set redeemed_by = p_device, redeemed_at = now()
   where id = p_code and redeemed_by is null and expires_at > now();
  if found then
    return query select true, null::text;
  else
    return query select false,
      (select case when redeemed_by is not null then 'already_used'
                   else 'expired' end
         from codes where id = p_code);
  end if;
end $$;
```

Client-side equivalent if not using the RPC:

```js
const { data } = await supabase
  .from('codes')
  .update({ redeemed_by: deviceId, redeemed_at: new Date().toISOString() })
  .eq('id', codeId)
  .is('redeemed_by', null)
  .select();

if (!data?.length) { /* already claimed */ }
```

Empty array = someone got there first. Prefer the RPC — one round trip, and it returns a usable reason.

The push handler batches pending attempts through `redeem()` and writes the outcome back onto the local attempt row. RxDB's `conflictHandler` decides what the losing device shows — flip the UI from "redeemed ✓" to "already used at 6:42pm."

---

## 7. Supabase Realtime — three things that will bite

1. **Publication toggle.** Tables must be added to the `supabase_realtime` publication before anything is emitted. This is the usual reason people think Realtime is broken.
2. **RLS is enforced per subscriber.** Policies must let each device see the rows it needs.
3. **No delivery guarantee, no replay on reconnect.** A tablet that briefly loses signal returns with a stale view and no idea what it missed. **Refetch affected rows on every `SUBSCRIBED` event**, not just on mount.

On Postgres Changes vs Broadcast: Supabase now recommends Broadcast for most use cases, since Postgres Changes needs less setup but scales worse — the concern kicks in around 3,000 concurrent subscribers. With three tablets, Postgres Changes is fine and simpler. Ignore the recommendation here.

---

## 8. The unfixable case: offline double-redemption

None of the above prevents a genuine offline double-redeem. If two tablets are both offline and the same code is scanned at each, both read `redeemed_at IS NULL` locally, both accept, and the server rejects one **after the fact** — by which point two orders went out the window.

This is not a bug to engineer out. It is the price of offline acceptance.

**Handle it with policy, per campaign, via the `requires_online` flag:**

| Campaign value | Setting | Behavior offline |
|---|---|---|
| Low (e.g. $2 wing discount) | `requires_online = false` | Submit blocked by default; a **permissioned** user can override behind a confirmation (§13) |
| High (e.g. $40 catering credit) | `requires_online = true` | **No override** — "can't verify, try again in a moment" |

The scanner branches on this flag. Do not make it a global setting. Submit is
online-gated by default — §13 has the full scan → order-# → submit → offline-override
workflow.

Also worth noting: on an LTE hotspot, two devices scanning within the same second will not see each other's write in time regardless of how good replication is. Replication makes the *other* screens catch up afterward; it does not prevent the race.

---

## 9. Audit trail

`scan_attempts` is append-only and is the attribution artifact. Every attempt — accepted, rejected, expired — is a data point. It is the closest thing available to tying a Google Ads direction request to a real person at the window.

Flag offline overrides (`offline_override = true`, §13) and reconcile those first —
they are the only accepted attempts that can still turn into a real double-redeem.

Keep it. Do not prune it with the codes table.

---

## 10. Code issuance: one identity code, not per-offer codes

**Decision: issue one permanent customer code. The QR encodes *who they are*, not *what they get*.** Entitlements live server-side.

Reasoning: the moment a customer holds three codes, every delivery mechanism degrades — they scroll to the wrong text, they present an expired one, the line slows. Wallet passes make that less bad; they do not fix it. Starbucks, Chipotle, and Panera all work this way.

Consequences:
- One SMS ever. One wallet pass ever. Nothing expires, nothing to resend.
- Staff scans; the app *shows* the customer's available offers. Staff apply the appropriate one in Toast by hand — there is **no auto-apply** (no Toast terminal integration, §13) and which offer fits depends on what the customer actually ordered (wings vs. burger). See fork F5, §19.
- `entitlements` replicates down to tablets exactly like `codes` does, keyed on customer hash — offline lookup still local.
- Durable customer identity across visits, which is the actual prerequisite for measuring repeat rate.

Keep the on-device row minimal: hashed customer id and entitlement list. No names, no phone numbers sitting on three tablets in a truck.

---

## 11. Delivery to the customer

### Near-term: SMS as MMS image

Send the QR **as an image, not a link.** An MMS image lives in the thread and opens with no network. A link needs signal at the exact moment the customer is standing at the window — the one thing that can't be counted on.

### Provider: Twilio, not Toast

Toast has no self-serve transactional messaging API; its SMS is bundled inside Marketing Essentials (commonly cited around $185/month, quote-based, all figures from competitor blogs — verify with the account rep). Twilio is the only option that supports "form submit → generate code in Supabase → send that image to that person."

**Costs:**
- MMS outbound $0.022 + carrier fees (~$0.0035 AT&T, ~$0.01 T-Mobile registered) ≈ **$0.03 per delivery**
- Local number $1.15/month
- A2P 10DLC campaign $1.50–10/month; brand registration $4.50 one-time; vetting $15 one-time
- ~500 codes/month ≈ **$20–26/month all-in**

**Start 10DLC registration immediately** — approval takes 1–3 weeks and no US consumer sends are possible during that window. Alternative: a toll-free number at $2.15/month skips 10DLC entirely and uses toll-free verification, often faster, and fine for one-way sends.

### Compliance (non-optional)

Collecting phone numbers on a form and texting people puts this under **TCPA and carrier A2P 10DLC rules**. Required: explicit consent language at the point of collection, working STOP handling, and a registered sending campaign. Statutory damages run per message.

### Later: wallet passes

| | Effort | Requirements |
|---|---|---|
| Google Wallet | ~1 day | Cloud service account, issuer ID, signed JWT → "Save to Google Wallet" link |
| Apple Wallet | ~2–3 days | $99/yr developer account, Pass Type ID certificate, signed `.pkpass` bundle |

Both support pushing updates to a live pass, so a redeemed code can visibly flip to "used." Apple's location triggers surface the pass on the lock screen near Forestville — genuinely useful for a truck. Passkit / Passcreator do both for a monthly fee if building is not worth it.

**Sequence:** identity code → MMS image → Google Wallet when volume justifies → Apple Wallet last, if ever.

---

## 12. Scanning — phone camera inside the HQ app

**The scanner is a screen in the HQ app that uses the device's own camera through the browser.** Staff open HQ → Marketing → Scan on their own phone; no Toast app, no dedicated hardware.

- Camera via `getUserMedia`; decode with a JS QR library (`html5-qrcode` or `@zxing/browser`).
- Requires **HTTPS** and a one-time camera-permission grant per device.
- The scanned value is hashed on-device (WebCrypto) before lookup (§4) — a dumped replica never yields live codes.
- This scanner screen is also where the **RxDB replica lives** — it *is* the offline-first client described in §3–§8. The HQ app already uses RxDB (for workflows), so this reuses existing infrastructure rather than bolting on a new sync stack.

**Later, optional:** a **Bluetooth 2D scanner** (Socket Mobile 2D — 1D models will not read QR) paired to the phone, only if the service window needs a faster / more durable flow than the camera.

See §16 for how this screen fits into the HQ app (tab, routing, permissions).

---

## 13. Toast integration — manual order-number capture

**No Toast API. The join is manual at the window and deferred to the warehouse.**

When the employee taps **Redeemed**, the scanner prompts for the Toast order number. That number is written to the local `scan_attempts` row and pushed with everything else. Toast order data arrives separately at HQ via emailed reports (SMTP), and the two are joined downstream on `(pos_business_date, pos_order_number)`.

### Why this is the better call

- **It works offline.** An API call at redemption would fail exactly when the hotspot is down — the moment the whole system is designed around. A typed string does not. This approach strengthens the offline design rather than fighting it.
- **No API access to procure.** Removes the Standard-vs-custom tier decision, the production-only sandbox problem, and an unpriced line item.
- **Ships faster.** No integration work blocking the attribution loop.

### The join key is not just the order number

Toast check/order numbers are typically sequential **per business date** and reset. The order number alone is not unique. The join key must be `(business_date, order_number)`, and for a future second truck, location as well.

**Business date is a trap.** Toast's business date rolls at a configured cutoff — commonly 4am, not midnight. A scan at 12:30am belongs to the *previous* business date. The scanner must compute `pos_business_date` using the same cutoff rule Toast uses, not `new Date().toDateString()`. Confirm the configured cutoff in Toast settings and hard-code it as a constant with a comment.

### Data quality is now the primary risk

Manual entry means typos, transpositions, and skipped fields. Mitigations, in priority order:

1. **Validate format on entry.** Confirm the shape of Toast order numbers for this restaurant (digit count, any prefix) and reject anything that doesn't match before allowing the redemption to complete.
2. **Make the field required**, or require an explicit "skip" with a reason code. A silently blank order number is an orphaned redemption you can never recover.
3. **Always capture `scanned_at` and `device_id`.** When a number is wrong, the timestamp is what lets you fuzzy-match against the Toast report — a redemption at 6:42pm almost certainly corresponds to a check opened within a few minutes of it.
4. **Show the discount amount prominently** on the redemption screen. See the double-entry problem below.

### The double-entry problem

The employee now has to do **two** things: apply the discount in Toast *and* type the order number into the scanner. That is where compliance will break — under a rush, the second step gets skipped.

Design around it: make order-number entry the action that *completes* the redemption, so there is no path to "redeemed" without it. If the employee can mark something redeemed and walk away, they will.

### Redemption submit workflow (online-gated, with a permissioned offline override)

The submit step is where single-use is actually enforced, so it is gated on
connectivity by default:

1. **Scan — works offline.** The employee scans the QR and the screen shows the
   customer's entitlement — *what the discount is* — read from the local replica
   (§4). No network needed to see it.
2. **Enter the Toast order number — works offline.** Required; it is the action
   that *completes* the redemption (above).
3. **Submit — requires the server.** Submit calls the `redeem()` RPC (§6), which
   atomically confirms the code is not already used and burns it. This is the only
   path that guarantees no double-redeem. If the RPC returns `already_used`, show
   that instead of accepting.
4. **Offline → submit is blocked.** When the device can't reach the server, the
   submit button renders an **"Offline — can't verify"** state and will not submit.
5. **Permissioned override.** A user with the `offline_override` permission may
   force-submit while offline. Doing so opens a confirmation screen:
   > *"Are you sure? Confirming this coupon while your device is offline risks a
   > double-redemption that can't be undone."*
   On confirm, it writes a local `scan_attempts` row (device-owned, push-only, §4)
   marked `offline_override = true`. The server arbitrates it on the next sync, and
   the `conflictHandler` (§6) flips the UI to "already used" if it lost the race.

**This layers on `requires_online` (§8) — it doesn't replace it.** The per-campaign
flag decides whether the override is even offered:

| Campaign `requires_online` | Offline submit |
|---|---|
| `false` (low value) | Permissioned override allowed, behind the confirmation above |
| `true` (high value) | **No override** — button stays "can't verify, try again"; even a manager can't force it |

**Two things this depends on:**

- **"Offline" must mean "can't reach the server," not `navigator.onLine`.** On an LTE
  hotspot the browser reports *online* while requests hang — so the flag would lie at
  exactly the moment that matters. Drive the button state from a real signal: a recent
  successful sync/heartbeat or a short-timeout reachability probe against Supabase.
- **The override is the §8 optimistic-accept path, now made explicit.** It doesn't add
  new risk — it takes the double-redeem risk that was silent and puts it behind a
  permission and a confirmation, and flags it in the audit trail for priority
  reconciliation (§9).

### SMTP ingestion pipeline

To be built at HQ:

1. **Scheduled Toast report** emailed to a dedicated ingest address. Needs an order/check-level export containing at minimum: order number, business date, total, discounts applied, timestamp.
2. **Parser** — inbox watcher → extract CSV attachment → normalize → load to a staging table.
3. **Idempotency** — the same report will be emailed twice at some point. Key on (business_date, order_number) and upsert; never blind-insert.
4. **Latency is T+1.** Scheduled reports arrive next day, so attribution is never same-day. Fine for campaign analysis, useless for real-time dashboards. Set expectations accordingly.
5. **Security** — emailed CSVs contain order and payment data. Use a dedicated mailbox, not a shared one; restrict access; do not forward.

### Reconciliation

Build a reconciliation view from day one. Three buckets:

| Bucket | Meaning | Action |
|---|---|---|
| `matched` | Scan joined to a Toast order | Attribution complete |
| `unmatched` | Scan has an order number with no Toast match | Typo — fuzzy-match on timestamp |
| `orphan` | Scan accepted with no order number captured | Lost attribution; count these as a staff-compliance metric |

The orphan rate is the health metric for this whole approach. If it climbs above roughly 10%, the window workflow needs fixing, not the code.

### What is given up versus an API

- No real-time attribution — everything is T+1.
- No automatic discount application; the employee still applies it in Toast by hand.
- No validation at scan time that the order number actually exists.

All three are acceptable for the current goal, which is measuring whether ad spend produces revenue — not real-time ops.

---

## 14. Open decisions

| # | Decision | Blocked on |
|---|---|---|
| 1 | Toast business-date cutoff hour (needed to compute `pos_business_date` correctly) | Check Toast settings |
| 2 | Toast order-number format — digit count, prefix, reset behavior | Look at a day of real checks |
| 3 | Which Toast scheduled report contains order number + business date + discounts | Toast reporting menu |
| 4 | Toll-free vs 10DLC local number | Whether the 1–3 week 10DLC wait is acceptable |
| 5 | `requires_online` threshold — at what face value does a campaign require network? | Business call |
| 6 | Whether the truck's three devices genuinely go offline independently, or are always together on one hotspot | Field observation — if always together, the architecture may be over-built |
| 7 | Wallet pass timing | Volume |
| 8 | Welcome offer definition (free side / % off / drink) + per-code expiry window | Business call |
| 9 | Confirm-then-burn vs burn-on-scan at the counter | UX call (recommend confirm-then-burn — avoids accidental redemptions) |
| 10 | QR encodes the raw identity token vs a URL wrapping it | Build call (recommend URL so a customer's own phone can open it too) |
| 11 | Campaign admin (create campaigns, generate codes, stats): HQ Go/Postgres endpoints vs Supabase directly | Architecture call — the redemption arbiter is Supabase either way (§6) |
| 12 | Who holds the `offline_override` permission (managers only? owner only?) | Business call |
| 13 | Reachability signal for the "offline" button state — heartbeat interval / probe timeout | Build call (§13) |

---

## 15. Explicitly out of scope

- Toast Loyalty as a replacement (phone-number check-in covers part of this but has no wallet support, no data portability, and is bundled at a much higher price point)
- Preventing offline double-redemption entirely (see §8 — accepted risk, managed by policy)
- Multi-location / second truck replication topology
- **Automatic offer selection / "apply best offer"** — the app can't apply discounts (no Toast terminal integration, §13) and the right offer depends on what the customer ordered. The app *displays* entitlements; staff choose and apply in Toast. (Fork F5, §19.)

---

## 16. HQ app integration (merged from DESIGN.md)

The scanner and campaign admin live in the **Yumyums HQ** app
(`~/projects/yumyums/hq` — vanilla HTML/JS front end, file-based pages; already uses
RxDB for workflows, which §12 reuses). File/handler details below reflect a codebase
map dated 2026-09-03 — re-verify at build time.

### Placement
A new top-level **Marketing** tile → `marketing.html`, with sub-sections:
- **Scan to Redeem** — the prominent staff action (phone camera; §12).
- **Campaigns** — create offers, generate codes (manager only).
- **Subscribers** — the mailing list, read-only (sources below).
- **Redemption stats** — counts + orphan rate (§13 reconciliation).

### Wiring (files to touch)
- `index.html` — add the tile to the grid + register it in the `TILE_SLUGS` map.
- `marketing.html` — new page: camera scanner (`html5-qrcode`) + the RxDB replica client (§3–§8).
- `SeedHQApps()` in `backend/internal/db/db.go` — seed `('marketing','Marketing','📢')` so the tile is permission-gated.
- Grant the `marketing` app permission to the relevant staff / manager roles.

### Permissions
Reuses HQ's `hq_apps` + `app_permissions` (roles `admin` / `manager` / `team_member`)
and the existing `auth.RequirePermission` gate:

| Action | Min role |
|--------|----------|
| Scan / redeem | `team_member` |
| View campaigns & stats | `manager` |
| Create campaigns / generate codes | `manager` |
| Override & submit while offline | `manager` (dedicated `offline_override` grant) |

Middleware grants tab access; enforce the create/stats gates inside the handler.

### Scanner result states (UX)
On decode → attempt redeem → show a large result card, then auto-reset for the next customer:
- ✅ **Redeemed** — green; offer + entitlement; then require the Toast order # to *complete* the redemption (§13 "double-entry problem" — no path to redeemed without it).
- ⚠️ **Already used** — amber; when + which device (from the `conflictHandler`, §6).
- ❌ **Invalid / Expired** — red; reason (`not_found` / `expired`, §5/§8).
- Branch on `requires_online` (§8): high-value campaigns show "can't verify — try again" when offline instead of accepting optimistically.

### Subscriber sources
Subscribers enter from the **Fluent Forms web signup** (WordPress —
`website/scripts/count_customers.py` already reads that DB and reports total / email /
SMS opt-ins by ad source), **SMS keyword opt-in**, and **Toast imports**. The one
identity code per customer (§10) is keyed to the customer regardless of source.

---

## 17. Build order

1. Schema + RLS + `supabase_realtime` publication
2. `redeem()` RPC and test the race directly with two concurrent clients
3. RxDB collections + pull handler with `updated_at` checkpoint
4. Push handler + `conflictHandler`
5. Clock-offset capture on sync
6. Scanner UI with `requires_online` branching **and required order-number entry**
7. Twilio 10DLC registration **(start this in parallel with step 1 — it is the longest lead time)**
8. QR generation + MMS send on form submit
9. SMTP ingest: dedicated mailbox → CSV parser → staging table (idempotent upsert)
10. Reconciliation view (matched / unmatched / orphan) + orphan-rate metric
11. HQ app integration — Marketing tile + `marketing.html`, app-permission seed, and the camera scanner screen that hosts the RxDB client (see §16)

---

## 18. Implementation appendix — gstate arbitration machine

The server-side redemption arbitration (the online submit path in §13, and the
reconciliation of synced offline overrides) is modeled as a statechart using
**gstate** ([github.com/floodfx/gstate](https://github.com/floodfx/gstate) — a Go
statechart library, XState-inspired). The machine **orchestrates**; the atomic
`redeem()` in §6 is what actually enforces single use.

### States

```
validating ─[token present]→ burning ─(Invoke: redeem() §6)→ route_outcome ─┬[redeemed]→     redeemed
     └─[no token]────────────────────────────────────────────────→ failed  ├[already_used]→ already_used
burning ─(Invoke error)→ failed ─(After 2s backoff)→ burning               ├[expired]→      expired
                                                                            └[unknown/empty]→ failed
```

### Machine (Go)

```go
package redemption

import (
	"context"
	"time"
	"github.com/floodfx/gstate"
)

type State string
type Event string

const (
	Validating   State = "validating"
	Burning      State = "burning"
	RouteOutcome State = "route_outcome"
	Redeemed     State = "redeemed"      // terminal ✓
	AlreadyUsed  State = "already_used"  // terminal
	Expired      State = "expired"       // terminal
	Failed       State = "failed"        // transient error / unknown result
)

// Attempt is the machine data; must implement Cloner[Attempt] for snapshots.
type Attempt struct {
	TokenHash, OrderNumber, DeviceID, AuthorizedBy string
	OfflineOverride bool
	ScannedAt       time.Time
	Outcome         string // redeemed | already_used | expired  (set by the burn)
	Err             string
}

func (a Attempt) Clone() Attempt { return a }

// Redeemer wraps the ATOMIC arbiter — the §6 redeem() RPC / conditional UPDATE.
type Redeemer interface {
	Redeem(ctx context.Context, tokenHash, deviceID, authorizedBy string) (status string, err error)
}

func Machine(db Redeemer) *gstate.Machine[State, Event, Attempt] {
	return gstate.New[State, Event, Attempt]("redemption").
		Initial(Validating).
		State(Validating, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Always().Guard(func(a Attempt) bool { return a.TokenHash != "" }).GoTo(Burning)
			s.Always().GoTo(Failed)
		}).
		State(Burning, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			// gstate spawns a goroutine and AUTO-CANCELS this ctx on state exit —
			// important on the flaky LTE link (§2). db.Redeem must honor ctx.
			s.Invoke(func(ctx context.Context, a Attempt, mutate func(func(Attempt) Attempt)) error {
				status, err := db.Redeem(ctx, a.TokenHash, a.DeviceID, a.AuthorizedBy)
				if err != nil {
					mutate(func(a Attempt) Attempt { a.Err = err.Error(); return a })
					return err // → onError: Failed
				}
				mutate(func(a Attempt) Attempt { a.Outcome = status; return a })
				return nil // → onSuccess: RouteOutcome
			}, RouteOutcome, Failed)
		}).
		State(RouteOutcome, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			// Eventless fan-out on the burn result (ordered; last is the fallback).
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "redeemed" }).GoTo(Redeemed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "already_used" }).GoTo(AlreadyUsed)
			s.Always().Guard(func(a Attempt) bool { return a.Outcome == "expired" }).GoTo(Expired)
			s.Always().GoTo(Failed) // unknown/empty → error, NOT a silent "expired"
		}).
		State(Redeemed, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Entry(func(a Attempt) Attempt { /* audit accepted (§9) */ return a })
		}).
		State(AlreadyUsed, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.Entry(func(a Attempt) Attempt { /* if OfflineOverride: flag lost race (§8/§9) */ return a })
		}).
		State(Expired, func(s *gstate.StateBuilder[State, Event, Attempt]) {}).
		State(Failed, func(s *gstate.StateBuilder[State, Event, Attempt]) {
			s.After(2 * time.Second).GoTo(Burning) // bounded retry for transient errors
		}).
		Build()
}
```

Driving one redemption: `Validating → Burning` fire immediately (`Always`); the burn
runs async in `Invoke`; for a request/response caller, await the terminal state via an
observer that signals a channel.

### Edge cases learned (must-not-forget)

1. **The machine is NOT the single-use arbiter — the DB is.** `redeemed` is a UI/state
   outcome; the atomic `UPDATE … WHERE redeemed_at IS NULL RETURNING` (§6) is the only
   thing preventing a double-redeem. A check-then-act in the machine (a guard on a prior
   `SELECT`) reintroduces a TOCTOU race: two concurrent scans both read NULL, both reach
   `redeemed`, two discounts go out. Keep check-and-write in one SQL statement; the
   machine only reacts to its 1-row / 0-row verdict.
2. **Online path is safe; offline is not.** The atomic UPDATE only protects when the
   device can reach the row. Two offline devices can both accept the same code; the DB
   arbitrates only on sync, after the fact (§8). Hence the permissioned + warned offline
   override (§13), and why `AlreadyUsed` entry flags a lost offline override for priority
   reconciliation (§9).
3. **Unknown burn result → `failed`, not `expired`.** The routing fallback must be an
   explicit error branch, or a malformed/empty status silently masquerades as an expiry.
4. **Cancellation is the library's job here.** gstate cancels the `Invoke` ctx on state
   exit; `db.Redeem` must honor `ctx` so a hung call on a dropped hotspot doesn't wedge.

### Library choice (context)

gstate vs [qmuntal/stateless](https://github.com/qmuntal/stateless) is close. **gstate**
is primary here because it fits this domain's async needs — `Invoke` auto-cancel,
`After` retry/backoff, in-machine typed data, `Snapshot`/`Hydrate` resume, and generics
type-safety. **stateless** is a fine alternative — more mature, and its external-storage
model persists each attempt's state as a DB column (a clean per-row fit), at the cost of
you owning the async (its `OnEntry` runs the query synchronously and you `Fire` the
outcome trigger). Either way, the DB remains the arbiter.

---

## 19. Addendum — statechart model, redemption stories & acceptance criteria

Output of a domain-driven state brainstorm. Models both halves: the **client (XState)**
scanner and the **server (gstate, §18)** arbitration. Structural calls: the client is
**parallel/orthogonal regions**; the server is per-aggregate lifecycles.

### 19.1 Client (XState) — parallel regions

**Region A — Connectivity / sync** *(cross-cutting; the scan flow guards on it, F1)*

| State | Meaning |
|---|---|
| `online` | Server actually reachable (real probe, not `navigator.onLine`) |
| `probing` | Checking reachability |
| `offline` | Unreachable |
| `syncing` | Pushing queued attempts / pulling code updates |
| `stale` | Online but replica not refetched after reconnect (§7 `SUBSCRIBED`) |

*Events:* `CONN_UP`, `CONN_DOWN`, `PROBE_TIMEOUT`, `SYNC_START`, `SYNC_DONE`, `SYNC_ERR`, `RESUBSCRIBED`

**Region B — Scan / redemption flow**

| State | Meaning |
|---|---|
| `idle` | Ready to scan |
| `scanning` | Camera decoding |
| `resolving` | Looking up token hash in the local replica |
| ↳ `offerReady` | Entitlement found → show discount |
| ↳ `unknownCode` | Token not in the bounded replica (§5.3) → can't verify offline (F2) |
| ↳ `spentLocally` | Replica shows redeemed → advisory reject (F3) |
| ↳ `expiredLocally` | Past `expires_at` (clock-offset adjusted, §5.1) |
| `enteringOrder` | Typing the Toast order # (`orderInvalid` on bad format, §13) |
| `readyToSubmit` | Order # valid + entitlement good |
| `submitting` | Online — invoke redeem RPC/server |
| `offlineGate` | Offline — branch on rights → `blockedOffline` / `overrideConfirm` / `overridePending` |
| `redeemed` / `alreadyUsed` / `expired` / `notFound` / `error` | Terminal result cards |

*Events:* `SCAN`, `QR_DECODED`, `DECODE_FAIL`, `RESOLVED(kind)`, `ORDER_INPUT`, `ORDER_OK`, `ORDER_BAD`, `SUBMIT`, `OVERRIDE_REQUEST`, `OVERRIDE_CONFIRM`, `OVERRIDE_CANCEL`, `SRV_REDEEMED`, `SRV_ALREADY_USED`, `SRV_EXPIRED`, `SRV_NOT_FOUND`, `SRV_ERROR`, `NEXT_CUSTOMER`

**Region C — Camera / permission:** `cameraOff → permissionPrompt → (permissionDenied | cameraLive) → cameraError`

*(Region D — Session/roles: source of the `canOverride`/role context, provided by the HQ shell.)*

### 19.2 Server (gstate) — per-aggregate lifecycles

- **Redemption Attempt arbitration** (§18): `validating → burning → route_outcome → {redeemed | already_used | expired | failed}`. Synced offline overrides enter seeded `offline_override=true`; an `already_used` terminal → flag lost race (F4).
- **Reward Code (Entitlement):** `created → sent → active → {redeemed | expired | void}` (+ `deliveryFailed` branch).
- **Campaign:** `draft → scheduled → active → ended → archived`.
- **Issuance / Delivery (Twilio, v2):** `pending → generatingCode → sending → {sent | failed → retrying}`.

### 19.3 Event taxonomy

| Category | Origin | Examples |
|---|---|---|
| **Actor (staff)** | Human at counter | `SCAN`, `ORDER_INPUT`, `SUBMIT`, `OVERRIDE_CONFIRM`, `NEXT_CUSTOMER` |
| **System / device** | Client env | `CONN_UP/DOWN`, `PROBE_TIMEOUT`, `CAMERA_*`, `SYNC_*`, timers (`After`) |
| **Boundary (client↔server)** | Over the wire | req `SubmitRedemption` → resp `SRV_REDEEMED / ALREADY_USED / EXPIRED / NOT_FOUND / ERROR`; Twilio delivery webhook |
| **Domain facts** | Server-emitted (audit/BI, §9) | `CodeIssued`, `CodeRedeemed`, `RedemptionRejected{reason}`, `OfflineOverrideAccepted`, `RaceLostReconciled`, `CampaignEnded` |

### 19.4 Redemption stories & acceptance criteria (decided)

Roles: **Counter Staff**, **Shift Manager** (holds override), **Owner**, **Customer**.

#### F1 — Connectivity awareness · DECIDED: orthogonal region
> As **Counter Staff**, I want the app to always know whether the server is truly reachable — independent of what I'm scanning — so it tells me up front whether a scan will be verified or need an override.

- The Connectivity region is always in exactly one of {`online`,`probing`,`offline`,`syncing`,`stale`} and runs **in parallel** with the scan flow.
- **Given** I'm mid-scan, **when** connectivity changes, **then** the Connectivity region transitions **without resetting** my scan progress.
- **Given** connectivity is `offline`/`stale`, **when** I press Submit, **then** the flow routes to `offlineGate` instead of calling the server.
- Connectivity comes from a real reachability probe/heartbeat, **not** `navigator.onLine`; a "connected-but-hanging" hotspot resolves to `offline`/`probing` within the probe timeout (§14 #13).

#### F2 — Unverifiable code offline · DECIDED: permissioned override allowed
> As **Counter Staff**, when I scan a legit-looking code my device can't find while offline, I want a clear, safe path so I neither wrongly reject a real customer nor give away an unverifiable discount.

- **Given** offline and a scanned token hash **not** in the local replica, **then** resolution enters `unknownCode`.
- **Given** `unknownCode` and I **lack** override permission, **then** submit is blocked: "Can't verify — connect to redeem."
- **Given** `unknownCode` and I **hold** override permission, **then** I may override; because the code is unknown we cannot read its offer, value, or `requiresOnline` flag, so the confirmation must state that **neither the offer nor prior use can be verified.**
- **When** I confirm, **then** write a `scan_attempts` row with `offline_override=true` **and** `unverified_code=true`, queued for sync arbitration.
- **When** sync arbitrates and the code is `not_found` / `already_used` / `expired`, **then** reconcile as a lost/invalid override and notify the Shift Manager (F4).

#### F3 — Stale local "already used" · DECIDED: reject offline; online, server wins
> As a **Customer**, if my code is still valid I don't want to be turned away over a stale device copy; as **Counter Staff**, I want obviously-spent codes rejected instantly.

- **Given** the local replica marks the code redeemed **and** I'm offline, **then** reject immediately as `spentLocally` ("already used").
- **Given** the same **and** I'm online, **then** do **not** reject on the local flag alone — submit; the atomic server check decides.
- **Given** the online submit returns redeemable, **then** proceed (the local copy was stale).
- **Given** the online submit returns `already_used`, **then** show "already used" (with when/who if available). **Server is authoritative.**

#### F4 — After-the-fact double-redemption · DECIDED: server domain event → manager notification
> As a **Shift Manager**, I want to be alerted *after* the fact when an offline override turns out to have been a double-redemption, so I can follow up — without the counter ever slowing for it.

- The client scan flow has **no state** for this outcome (the customer has left by reconciliation).
- **Given** an `offline_override` attempt is arbitrated **and** the burn returns `already_used`, **then** the server emits a `RaceLostReconciled` domain event.
- **Given** `RaceLostReconciled`, **then** a Shift Manager notification / read-model entry is created (code, device, staff, time, value) for follow-up.
- **Then** the `scan_attempts` row's status reflects the loss for override/orphan reporting (§9/§13).

#### F5 — Multiple offers on one code · CUT
Not building auto-apply/offer-selection. The app can't apply a discount (no Toast terminal integration, §13), and which offer fits depends on what the customer actually ordered (wings vs. burger). The app **displays** the customer's entitlements; staff choose and apply in Toast. (See §10, §15.)

#### F6 — Accidental re-scan · DECIDED: dedupe within session
> As **Counter Staff**, if I scan the same customer's code twice on one order, I want the app to recognize it's the same redemption and not double-apply or error, so a fumble doesn't cost a discount or confuse the total.

- **Given** an active session for code X (any of `resolving`…`submitting`…result), **when** X is scanned again, **then** it's a **no-op** with a gentle "already scanning this code" (no new attempt).
- **Given** X just reached a terminal result this session, **when** re-scanned, **then** re-show the existing result instead of starting a new redemption.
- **Given** a **different** code Y is scanned mid-session on X, **then** prompt to finish/clear the current customer first (prevents cross-customer mixups). *(Small sub-decision — default proposed.)*
