# OKRs

Milestone: Close the loop — a code issued to a customer becomes a money-tied, campaign-attributable redemption at the truck window, and it works when the LTE hotspot is down.

Grading modes: declared

<!-- Authored 2026-09-03, attended `/nc-roadmap-round`, in the same sitting as
`.night-crew/knowledge/roadmap.md` (DESIGN §15j.42), decomposing the design of record
`docs/qr-offline-redemption-handoff.md`. Previous cycle's page archived at
`reference/okrs-2026-09-03-prod-current-and-honest.md`; its close at ledger T-50 (12 MET · 0
PARTIAL · 2 NOT MET). No `/nc-retro` preceded this round; recorded as an absence.

Authoring rules carried from the last three closes:
  1. Each objective has at least one KR that measures its own claim directly — where an
     objective says a thing works, a KR asserts the thing was RUN and seen.
  2. Every KR names its `(measured by: …)` artifact in the parenthesized form the validator
     parses (prose `Measured by:` silently fails — probed at the 2026-08-05 round).
  3. Every KR declares its grading mode at authoring — derived · attested · disclosed-deferred.
  4. A KR must not be failable by desirable behaviour.
  5. Denominators are named by rule, not literal count, where the population can grow.

Honesty note on modes — this cycle is greenfield. Most of what these KRs measure does not
exist yet (the Supabase project, `marketing.html`, `backend/internal/redemption`, the RxDB
client, the reconciliation view). Where the *measuring surface itself* is a not-yet-built file,
the honest mode is **disclosed-deferred** — declared here, graduating to a real grade once the
card that builds the surface lands. The operator-observed close-bar legs and the
connectivity-visibility KR (P-KR4) are **attested** (evidence + judgment). The two carried
QA-infra KRs (Q-KR3, Q-KR4) and the delivery compliance-mechanism KR (D-KR2) are **derived** — a test or command's exit code IS the verdict —
and their measuring surfaces (`BACKLOG.md`, the scorecard, `tests/`) exist today. -->

## Product

### Objective: The marketing-attribution loop is closed — a code becomes a money-tied, campaign-attributable redemption, and it works offline at the window.

- **P-KR1 — the operator personally observes all 3 close-bar legs** — a real end-to-end redemption that burns and then shows "already used" on the second scan; the redemption landing `matched` in reconciliation; and offline-safe-by-policy behaving in both branches — recorded as exactly 1 dated ledger line naming what was seen (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 The whole point of the cycle; no card status, KR grade or closeout substitutes, and the milestone may not close without this line.
- **P-KR2 — a scan resolves the customer's offers at the window in both connectivity states**: **online**, a scan shows the *complete, current* list ("N offers available" — the app **displays** them; staff apply the right one in Toast, no auto-apply, F5) resolved server-side; **offline**, a synced customer's **full** list resolves from the **replicated non-expired offers** (R3 — bounded by `expires_at > now()`, ~2 MB at truck scale), falling back to the **offer embedded in the QR** (D-KR3) for a customer not yet in the replica (a just-signed-up walk-up while the truck is offline) — observed at least 1 time per path (graded: disclosed-deferred · measured by: `marketing.html`) 🛑 The offers replica is bounded by *active* offers, not lifetime customers, so it does not grow with the customer base; the embedded offer is the not-yet-synced fallback.
- **P-KR3 — the orphan rate is visible and the loop is joinable**: the reconciliation view renders matched / unmatched / orphan with a computed orphan-rate figure, and the close-bar test redemption appears in the `matched` bucket (graded: disclosed-deferred · measured by: `marketing.html`) 🛑 The orphan rate is the health metric for the whole approach (§13); a view that can't compute it grades this NOT MET.
- **P-KR4 — the scanner always shows the crew whether they can submit, and updates live when it changes**: driven by the real reachability signal (§13, not `navigator.onLine`), the screen shows a clear **offline** state when the server can't be reached, and when reachability returns after an offline period the screen **transitions on its own** to a submit-enabled state — observed by the operator toggling reachability both ways (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 The crew must never be guessing: a screen still showing "offline" after reconnect, or "submit" while unreachable, grades this NOT MET. Complements Q-KR1 — P-KR4 is the *visibility and live transition*, Q-KR1 is the *policy enforcement*.

## Delivery

### Objective: A code reaches a real customer as an image, lawfully, over a registered number — delivery is real and compliant, not a mock.

- **D-KR1 — the operator's test signup produces a scannable QR image in their own Messages thread**: a form submit generates the identity code and sends it as an **MMS image, not a link** (§11), received and scanned successfully at least 1 time, recorded as a dated ledger line naming the sending number and its registration status — toll-free verified or 10DLC (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 The "was it RUN and seen" KR for delivery — a mock send grades this NOT MET.
- **D-KR2 — the consent + STOP compliance mechanism is proven end-to-end, independent of a registered production number**: an e2e test asserts (a) the signup path records explicit consent and refuses to issue or send a code without it, and (b) an inbound "STOP" marks the subscriber opted-out so the next send is blocked — run against Twilio test credentials or a mocked send transport, green with **no live consumer send required** (graded: derived · measured by: `tests/`) 🛑 STOP handling and consent are our code and non-optional (§11); a production number's A2P registration status is a separate operational fact recorded on D-KR1's send line, and this KR must be greenable whether or not that registration has cleared.
- **D-KR3 — one permanent per-customer identity code; entitlements are server-side of record, with the issued offer embedded in the QR for offline viewing**: every code issued this cycle is one durable identity code (denominator: every code generated this cycle; 0 per-offer identity codes). Scanning it **online** resolves the customer's *complete, current* entitlement list server-side (§10); the QR payload **also** carries the offer that was current at issue as a self-describing descriptor, so a scan can be *read* offline even before that customer's entitlements have replicated to the tablet — the server list stays source of truth, the embedded copy is the offline-display snapshot and the **not-yet-synced fallback** (the replicated non-expired offers per P-KR2 are the primary offline source) (graded: disclosed-deferred · measured by: `supabase/`) 🛑 The embedded offer is display-only; redemption still burns through `redeem()` (§6), so embedding an offer never authorizes a redemption by itself. Graduates once the code-generation path and schema exist.

## Engineering

### Objective: Single-use is enforced by the atomic arbiter and survives the offline race — the machine orchestrates, the database decides.

- **E-KR1 — the two-concurrent-client race proves single use**: two clients fire at one code, **exactly one** gets `ok=true` across at least 20 runs with 0 double-wins, red-first against a naive check-then-update that lets both win (graded: disclosed-deferred · measured by: `supabase/`) 🛑 The core correctness claim (§6/§18 #1); the atomic `UPDATE … WHERE redeemed_by IS NULL` is the only thing enforcing single use, and this KR runs the proof.
- **E-KR2 — an unknown or empty burn result routes to `failed`, never a silent `expired`**: pinned by 1 test that reds on a fallthrough-to-`expired` and greens on the explicit error branch (§18 #3) (graded: disclosed-deferred · measured by: `backend/internal/redemption`) — graduates once the gstate machine package exists.
- **E-KR3 — a lost offline override emits a domain event and notifies a manager**: a two-attempt reconciliation where a synced `offline_override` burns to `already_used` emits a `RaceLostReconciled` event and creates a Shift-Manager notification / read-model entry for the loser (§8/§9, F4), pinned by at least 1 test (graded: disclosed-deferred · measured by: `backend/internal/redemption`).
- **E-KR4 — the offline expiry check is clock-tamper-safe**: with the device clock set at least 2 days fast, an expired code is still rejected offline via the stored `serverNow − deviceNow` offset (§5.1), pinned by at least 1 test (graded: disclosed-deferred · measured by: `marketing.html`) — the offset lives in the scanner's replica client; graduates with it.

## QA

### Objective: The loop can't lie about what it did — offline risk is auditable and policy-gated, and the planning surface the rounds read is machine-true.

- **Q-KR1 — the offline-safety branches behave by policy, observed by the operator**: under a deliberately killed server-reachability probe, a `requires_online = true` campaign refuses submit ("can't verify") and a `requires_online = false` campaign offers the permissioned override behind the §13 confirmation — both branches seen and recorded (close-bar leg 3) (graded: attested · measured by: `.night-crew/knowledge/ledger.md`) 🛑 Failable only by a high-value code accepting offline, or an override with no confirmation — never by desirable behaviour.
- **Q-KR2 — every accepted offline override is auditable and reconciled first**: a test override lands in the append-only `scan_attempts` trail flagged `offline_override = true` (and `unverified_code = true` when the code wasn't in the replica, F2) and surfaces at the top of the reconciliation view for priority handling (§9) (graded: disclosed-deferred · measured by: `marketing.html`) — graduates once the scan trail and reconciliation view exist.
- **Q-KR3 — `night-crew backlog check` exits 0 on the migrated BACKLOG.md** and `night-crew backlog list` emits every entry the document holds (list count == document entry count), with a content-preservation diff recorded in the card's merge-intent (graded: derived · measured by: `.night-crew/knowledge/BACKLOG.md`) 🛑 Carried from last cycle's Q-KR2 (NOT MET — promoted, never slated); the exit code and count equality ARE the verdict, run at close, no judgment involved.
- **Q-KR4 — the scorecard sees the four roles**: at close, `night-crew scorecard` renders at least 1 record-backed row per rostered team from this cycle's hand-run nights, OR the mechanical blocker is recorded with its clone-side handle and the target-side half is done (graded: derived · measured by: `.night-crew/knowledge/scorecard`) 🛑 Carried from last cycle's Q-KR3 (NOT MET — promoted, never slated); the scorecard's own output is the verdict, and the OR-arm exists because the CLI may own half the fix.
