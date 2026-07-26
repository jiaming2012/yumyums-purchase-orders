# Decisions needed — run `overnight-20260726`

Open items the run declined to decide. The run executes; it does not decide.

---

## Carried from card A's G6 review (not blocking — card A merged APPROVE-WITH-NOTES)

These are G6 findings the reviewer demonstrated but which fall **outside card A's signed scope**.
The run did not fix them, because widening a card's scope at 1am is how a signed slate stops
meaning anything. They are recorded here so the morning reader decides.

### D-1 — `workflows.html:2503` — an eighth status-reading site the slate did not name

**Severity: cosmetic.** The History view renders the raw status token:
`... + escapeHtml(s.status || '')`. After card A, a no-approval submission's history row displays
the literal lowercase string `completed` as user-facing copy. (Before card A it displayed
`pending`, which was worse — so this is an improvement, not a regression.)

No gate reads it and it does not misrepresent state. But the slate named **seven** call sites and
this is an eighth, and the card's own framing — *"teach the client the DB's vocabulary"* —
arguably covers it.

**The decision:** humanize it (a small status→label map) as a follow-up, or accept raw tokens in
the History view. Not decided here.

### D-2 — offline double-tap still writes two submission rows

**Severity: real, pre-existing, untouched by card A.** G6 demonstrated by code read:

`workflows.html:1656` mints a fresh `idempotency_key` per call; `:2778` handles `err.offline` by
returning to the list **without** pushing anything into `MY_SUBMISSIONS`. So: offline → submit →
"Queued for sync" → reopen the checklist (still editable, submit button live — *correctly* so,
since nothing was persisted) → submit again → a second UUID → `enqueueSubmission` writes a second
`submitQueue` entry → `drainQueue` POSTs both → **two rows.** `checklist_submissions` carries only
`idempotency_key UUID UNIQUE` (migration 0011), and there is no server-side duplicate guard —
`grep -rn duplicate_submission backend/` is empty; the only 409 is `template_archived`.

Card A's implementer defended leaving the key per-call, and **that defence is sound for the path it
covers** (a retry of the same enqueued payload — `sync.js:549` persists the key with the payload,
`drainQueue` replays it verbatim). G6 confirmed the reasoning and confirmed it **does not reach
this path**.

Fixing it means either a server-side guard (→ `backend/internal/workflow`, which is card A's PARK
trigger and reopens decision 49) or a client-side queued-submission marker. **Both are scope the
slate did not sign.**

**The decision:** backlog it, or slate it as its own card. Not decided here.

### D-3 — `tests/sync.spec.js:1584` stale comment

**Severity: doc rot.** The comment reads *"requires_approval false → submit yields `'submitted'`"*.
The server yields `'completed'`. It sits inside one of card A's two acceptance specs. One-line fix,
but it is a test file no card tonight owns.

**The decision:** fold into whichever card next touches `sync.spec.js`. Recorded so it is not lost.

---

## Card B — `sync-jwt-bridge-endpoint` (NOT BLOCKING — the card LANDED, 16/16 variants green)

**The card's PARK trigger did NOT fire.** Bridging HQ's grants into claims turned out to be a pure
mapping of existing data — `users.id` → `sub`, `users.roles` → `hq_roles`,
`app_permissions ⋈ hq_apps` → `hq_grants`, `sessions.token_hash` → `hq_sid` — with `role` a
constant `authenticated`. **No new grant or permission concept was invented.** An anti-drift test
(`TestGrantedSlugs_MatchesRequirePermission`) asserts per-slug that the token's grant list agrees
with the `EXISTS` predicate `auth.RequirePermission` already enforces, so the bridge cannot quietly
become a second permission model.

The two items below are downstream product forks the card **surfaced and deliberately did not
answer**. Neither blocks anything tonight; both are inherited by `sync-hard-cutover`, and D-5 in
particular should not be discovered by that card at 3am.

### D-4 — who WRITES `public.hq_grant_projection`? (mechanism, low stakes)

The card's central design call: the token's `hq_grants` claim is **advisory, not the gate.** Claims
freeze at mint, so a policy that trusted the claim would leave a revocation replay window as long
as the TTL — on the one admin action performed precisely because it should take effect *now*. RLS
therefore joins `public.hq_grant_projection`, a live (user_id, app_slug) projection of
`app_permissions ⋈ hq_apps`, on every row. Variants V8/V9/V12 prove revocation is immediate.

What the card did **not** decide is how that table stays current in a real deployment: push on
grant change, a periodic reconcile, or `postgres_fdw` straight at HQ's tables. All three are
ordinary plumbing and the card would have picked one, but the choice is only meaningful once the
sync database's relationship to HQ's own database is settled — which is the cutover card's subject,
not this one's. **Recorded as an explicit open contract rather than a silent assumption.**

### D-5 — 🛑 `owner_id = sub` cannot express HQ's real row ownership (PRODUCT)

**This one is a product question and it is load-bearing for the cutover.**

The policies ship a single-subject predicate: a row belongs to exactly one `owner_id`, compared
against the token's `sub`. That is the simplest predicate that can be observed both admitting and
refusing, which is what an attack-variant suite needs — and W1 flagged the same limitation as an
open question rather than a design.

But HQ rows are frequently **not** single-owner. A checklist submission belongs to a submitter
**and** to an approver; the Approvals tab exists precisely because a second person must see and act
on someone else's submission. Under the policy as written, an approver would see nothing.

Extending it means answering *who may see whose submissions* — which is a permission question, not
a mechanism one, and answering it is exactly what this card's park trigger forbids. So it is
recorded here instead. **The cutover card must not invent an answer either.**

Note this is a limitation of the fixture's *predicate*, not of the bridge: the token already
carries `hq_roles` alongside `hq_grants`, so a role-aware or assignment-aware policy is expressible
without minting anything new. What is missing is the decision about what it should say.

---

## Parked cards

_(none yet)_
