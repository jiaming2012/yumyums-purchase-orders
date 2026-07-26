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

### D-6 — `hq_grants` is NARROWER than what the user can actually reach (umbrella slugs)

**Raised by G6 against card B; the run corrected the CLAIM and did not change the behaviour.**

Card B's code comment said `GrantedSlugs` used *"the same predicate `RequirePermission` uses."*
G6 demonstrated that is imprecise. `auth.RequirePermission(pool, grantSlug, umbrellaSlugs...)`
matches `a.slug = ANY(candidate_set)`, and the umbrella position is **really in use** —
`main.go:628, 642, 652`. `GrantedSlugs` asks about one slug at a time with no umbrella context.

Concretely, for a user holding `inventory`:

| | result |
|---|---|
| token's `hq_grants` | `[inventory, operations]` |
| `RequirePermission("inventory-trends", "inventory")` | **true** |
| `RequirePermission("inventory-cost", "inventory")` | **true** |

So a client rendering its launcher naively from `hq_grants` would **hide two surfaces the user
can actually reach**.

**No security impact — it errs CLOSED.** The list is a subset of the reachable set, never a
superset, and it is advisory in the first place: the RLS gate is the live
`hq_grant_projection`, not this claim. The failure is a usability one.

🛑 **Two things the morning reader should know about how this was found and why it stayed open:**

1. **Card B's parity test structurally cannot catch it.** It compares per-*single*-slug
   (`a.slug = ANY(ARRAY[slug])`), which is precisely the umbrella-free case. Strengthening that
   test (see the F1 repair) does not close this — the two are independent defects, and a reader
   should not assume the now-non-vacuous parity test covers it.
2. **What the advisory list *should* contain is a design call, and the run did not make it.**
   The options are real and they differ: expand umbrellas at mint time (the claim then matches
   the mounted gates, but the token asserts more than a single `app_permissions` row does);
   ship the narrow list and require clients to expand (correct but pushes the umbrella table
   into every client); or emit both a narrow and an expanded field. Picking one is a product/UX
   decision about what a launcher should show, so it goes to the operator rather than getting
   invented at 11pm.

**Inherited by the client-layer card** (`sync-rxdb-schema-and-replication`, which now owns the
client-construction helper) — that is the card that will actually render a launcher from this
list and will hit the divergence first.

---

## Parked cards

_(none yet)_

---

## Card C — one UNATTRIBUTED red in the full-suite gate (NOT a park; card C landed GO)

**This is a record, not a fork.** Card C is not parked and needs no decision to
merge. It is filed here because the run's discipline says an unattributed red
belongs in the durable record rather than only in a card report.

**The red:** `tests/workflows.spec.js:2466` — *Loading states › unsubmit returns
checklist to editable draft [RUN-10]*. Failed **both** attempts in card C's full
549-test gate (`1 failed / 2 flaky / 6 skipped / 540 passed`, 47.0 m,
`TEST_PORT=8299`, `hq_test_c1`).

- attempt 1: `page.click('[data-action="unsubmit"]')` timed out — *"element was
  detached from the DOM, retrying"* (a re-render race).
- retry #1: after unsubmit, `[data-action="submit"]` never reappeared.

### 🛑 I REFUSE TO ATTRIBUTE IT, and here is the measurement rather than a guess

| Condition | Result |
|---|---|
| RUN-10 alone, `--repeat-each=3 --retries=0`, on card C's HEAD | **3 / 3 passed** |
| Full `tests/workflows.spec.js` (80 tests), `--retries=0`, on card C's HEAD | **80 / 80 passed**, RUN-10 green as test #57 |
| Full 549-test suite on card C's HEAD | **failed twice** |

So it is **not** deterministic, **not** within-file order sensitivity, and it did
**not** reproduce in 83 attempts outside the whole-suite condition.

**Why it is mechanically not card C's**, and this is provable rather than
asserted:

```
git diff --stat overnight-20260726..HEAD -- backend tests features lib \
  ':!.night-crew' workflows.html sync.js ptr.js index.html   # prints NOTHING
```

RUN-10 exercises `workflows.html` and the Go backend. **Every byte the browser
and the server see for this test is identical to `overnight-20260726`.** Card C's
entire diff is `vendor/**` (new, never imported by any HQ page), `build-sw.js`
(build-time only), `sw.js` (**never registered — `playwright.config.js:60` sets
`serviceWorkers: 'block'`**), and `.night-crew/**`.

**Why I still will not call it "pre-existing flake" as a fact.** Two correlations
exist and guessing either way is equally wrong:

1. **Card B ran its own full Playwright suite CONCURRENTLY** for the first half
   of card C's gate. Measured 1-min load: **35.48 at start, peaking above 61,
   settling to 8–17 once card B finished.** A green here bounds a *loaded*
   condition; so does this red. **Concurrency makes attribution harder, not
   easier**, and that is the honest statement.
2. **Card A's just-merged `workflow-submission-status-client-half` is topically
   adjacent** — RUN-10 is precisely an unsubmit/status re-render assertion, and
   card A's change is already in card C's base. **Flagged as a correlation. Not
   asserted as a cause.** Card C did not test at base and therefore cannot say.

**What would settle it:** run RUN-10 inside the full 549-test suite at
`overnight-20260726` (i.e. with card A merged, card C absent) on a quiet box. If
it reds there too, it belongs to card A or to the suite; if it stays green, the
question reopens. **That is one command and it was outside card C's remit.**

### The two flaky (passed on retry #1) — also not attributed

- `tests/sync.spec.js:836` — *sub-step checks on Device A appear checked on
  Device B [SYN-03]*
- `tests/sync.spec.js:1327` — *checkbox answer converges (live + catch-up)*

Both in `sync.spec.js`, the file carrying the **proven ~16–20 % flake** whose
exposure card A's merged seam fix deliberately raises. **Neither is the
specifically flagged `sync.spec.js:1198`** — that one passed. `purchasing.spec.js:1407`
(FR-13) also passed. Same refusal to attribute applies, same reason.

---

## Card C — G6 findings recorded for the operator and the next card (D-7 … D-11)

*Appended by card C after its G6 review (APPROVE-WITH-NOTES). **Record only — do
not fix here**; four separate repairs (F1/F2/F7/F8) were applied as code and are
not listed below. Numbering starts at **D-7** per the orchestrator: **D-4/D-5/D-6
are card B's** and land from card B's worktree, so they are not visible in this
file yet. **D-1…D-3 above are untouched.***

### D-7 — 🛑 Cross-tenant read through `api-cache`. Distinct from the `/api/` gate, and worse.

Workbox's `api-cache` is keyed **by URL only**. `Authorization` is not part of the
cache key and **no `Vary` is configured** in `build-sw.js`'s `runtimeCaching`
block. On a **shared truck phone**, a second crew member — or the same person
re-logging-in as a different user — issuing the same replication URL can be
served **the previous user's rows straight from cache**.

This is a **cross-tenant read**, not staleness. It shares a root cause with the
3c finding (the `/api/` `NetworkFirst` route) but is a **different and more
serious failure mode**: 3c loses data, this one *discloses* it, and RLS cannot
help because the request never reaches PostgREST.

**Hand to `sync-rxdb-schema-and-replication` explicitly.** Not mitigated by card
C; the "do not mount under `/api/`" gate happens to also close it, but only as a
side effect, and that is too thin a reason to leave it unstated.

### D-8 — The precache cost of the vendored bundle. **Operator's call.**

Card C's regenerated `sw.js` takes the PWA precache from **1452.1 KB / 22 files
to 1947.1 KB / 23 files — +34%**. The bundle is **25.4% of the entire precache**.

Every crew phone downloads **~495 KiB over LTE** on the next `task prod:deploy`,
for an asset **no page imports**. Card C's merge-intent said "precached but never
imported"; it never stated the cost. Stated now.

**Two options, and this is not card C's to choose:**
- **Keep it precached** — the offline path is then proven end-to-end (leg 3a
  demonstrates the bundle loading with the network cut), and adoption later costs
  nothing extra.
- **Exclude it until a page actually imports it** — drop the `globPatterns`
  entry, re-add it in `sync-rxdb-schema-and-replication`. Costs the truck nothing
  today, but the offline-availability proof then goes untested until adoption.

### D-9 — Two of the verdict's seven "may now assume" items are COUPLED, and the coupling was unflagged.

Verdict items **1** ("no `global.fetch` shim needed") and **7** ("do not mount
under `/api/`") both hold **only in a same-origin-fronted shape** — one origin
reverse-proxying PostgREST — which **HQ does not have today**. `browser/serve.mjs`
*invents* that reverse proxy for the harness; building the real equivalent is
**neither costed nor listed under "did NOT establish"** in card C's verdict.

**If `sync-rxdb-schema-and-replication` goes cross-origin** — the shape W1 and W2
actually used — **item 7 becomes moot** (a cross-origin URL never matches
`/\/api\//`) **and item 1 inverts: W2's `global.fetch` shim comes back.**

Card C has added the dependency to the verdict so item 1 cannot be read without
item 7. Recorded here because **the reverse proxy itself is unbuilt, uncosted
work** that the schema card must size.

### D-10 — Leg 4's millisecond figures are wall-clock, not monotonic.

`browser/specs/leg4-leader-election.spec.js` measures handover with `Date.now()`
deltas. **This box steps its clock** — G6's own run printed a **negative
`-1545 ms`** interval for the same measurement.

**The qualitative finding is unaffected and stands:** exactly one tab leads, the
follower does not replicate, and the survivor both wins the election and
**actually begins replicating**. Those are ordering facts, not timing facts.

**But the quoted numbers (47 ms / 65 ms / 87 ms) are wall-clock deltas on a
clock-stepping host and should be read as "sub-second", not as measurements.**
Anything that needs a real number should use `performance.now()`.

### D-11 — Card C's as-built design deviates from its own pre-written merge-intent.

The merge-intent (written **before** implementation, as required) promised that
`browser/` would carry **its own `package.json`, `node_modules/` and lockfile**,
with rxdb / supabase-js consumed from W2's `spike-supabase/rxdb/` lockfile.

**As built:** `browser/` has **zero dependencies and no lockfile** (it resolves
`@playwright/test` up the tree and `serve.mjs` is dependency-free), and a **new
`vendor/package.json` + `vendor/package-lock.json`** was created instead.

**The as-built design is better** — it is precisely what holds the root-package
line, since the bundle's sources must be pinned independently of both the root
project and W2's harness. **But the deviation was not disclosed at the time**,
and a merge-intent that silently diverges from what lands is worth less at the
next merge. Recorded so the next card's note is read with that in mind.

---

## Orchestrator finding (D-12) — surfaced at the card C merge, belongs to no card

### D-12 — 🛑 `build-sw.js` globs the WORKING TREE, so any untracked page silently enters the precache

**Found at merge time, not by a card.** After merging card C, the standing rule for generated files
says to confirm `sw.js` is consistent, so `node build-sw.js` was re-run. It produced **24 files /
2166.8 KB** against the committed **23 files / 1947.1 KB**.

The extra entry is **`backlog-round.html`** — an **untracked** file that has been sitting in the repo
root since before this run started (it appears in the run's very first `git status`). It is nobody's
card, and it is not new tonight.

**Why this matters more than a stray manifest line.** `build-sw.js` globs the filesystem, not the
git index. So:

1. Anyone with a work-in-progress page in the repo root who runs `task sw` — which `task test` and
   `task prod:deploy` **both run automatically as a dependency** — bakes that page into the precache
   manifest.
2. If that `sw.js` is committed and deployed, every phone's service worker tries to precache a URL
   that exists on no other machine.
3. **A Workbox precache entry that 404s fails the entire service-worker install**, not just that one
   asset. The failure mode is "the PWA stops updating," and its cause is invisible from the symptom.

**The regenerated output was discarded and the committed `sw.js` restored**, so nothing shipped. But
the trap is live and will fire again.

**The decision:** teach `build-sw.js` to glob the tracked set (or add an explicit allowlist /
`globIgnores` entry), or accept the foot-gun and document it loudly in CLAUDE.md next to the
existing "run `task sw` after changing HTML/JS" instruction. **Not decided here.**

*(Separately, and not a decision: `backlog-round.html` itself is untracked and unexplained. It is
left exactly as found — the run does not clean up files it did not create.)*
