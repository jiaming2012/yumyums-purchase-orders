# Decisions needed — `overnight-20260729-2`

> Run: `overnight-20260729-2` · Slate: `slate-20260729-2.md` (signed 2026-07-28, 5 cards)
> **Two cards parked: B2 `sync-rxdb-row-visibility-rls` and A1 `app-timezone-unify-new-york`.**
> Three landed and merged.
>
> Every item below is a fork the run **refused to decide**, not a blocker it hit. The batch
> sign-off covers the signed specs; it does not cover any of these.
>
> 🛑 **D-2 is the one to read first if you read only one.** It concerns payroll inputs and spans
> two repositories.

---

## D-1 · 🛑 The projection cannot be written where decision 61 says it must — and B1 is what settled that

**Status:** OPEN — architecture decision, and it blocks the milestone's security card.
**Found by:** Card B2 during orientation, before writing any SQL. **Park verified CORRECT** by an
independent G6 that re-executed every probe on both servers.

**What decision 61 requires.** The `template_assignments ⋈ users` projection is written
**push-on-grant-change, in the same transaction as the mutation** — because reconcile reintroduces
the exact replay window the projection exists to eliminate.

**Why it cannot be.** The projection and the mutation are in **two different Postgres servers**.
No transaction can contain both. This is not the obstacle the PARK trigger anticipated — the
trigger asks about *restructuring the mutation* — and it is strictly stronger: no restructuring of
the mutation changes it.

Measured on both servers, twice, by two different agents:

| Probe | HQ `5433` | Substrate `46011` |
|---|---|---|
| `template_assignments`, `users`, `app_permissions`, `checklist_fields`, `checklist_sections` | present | **absent** |
| supabase roles (`anon`, `authenticated`, `service_role`, …) | **0 of 5** | 5 of 5 |
| `max_prepared_transactions` | **`0`** | **`0`** |

The backend has exactly one pool on one DSN. `Sign()` is an **allowlist** that can emit only
`authenticated` — so HQ cannot mint `service_role`, and the only channel to the substrate is HTTP
to PostgREST, which is not a transaction participant. G6 searched for a construction that would
honour decision 61 and **found none**.

**The timing matters, and it is the real finding.** D-4 (07-26) said this choice *"is only
meaningful once the sync database's relationship to HQ's own database is settled."* Decision 61
answered anyway. **Card B1 settled that relationship tonight — in the direction that makes
decision 61's contract impossible.** Decision 61 is not wrong about what is *wanted*; it is
written against a topology that had not yet been chosen.

**The fork, ordered as the run would have you consider it:**

| | Option | Window | Real cost |
|---|---|---|---|
| **(a)** | `postgres_fdw` substrate → HQ | **zero** — the policy reads HQ's live tables, so there is no projection to write and "same transaction" is vacuous | **Reverses decision 61** — an operator/architect call. The extension is *not* an obstacle: proven installable at **both** ends by executing the C symbol, one line of DDL. Standing cost is HQ's Postgres on the network path of **every RLS row check**. |
| **(d)** | Defer to `sync-hard-cutover`, co-locate | zero, eventually | Makes decision 61 *come true* rather than amending it. Reorders the roadmap; B2 cannot land first. |
| **(b′)** | Native logical replication | bounded lag; **unbounded** if the subscription is down and the slot backs up | Best async mechanism — crash-safe, built in, **zero code in the mutation path**. Needs `wal_level=logical` on HQ (bump + restart). Replicates `users` wholesale, which is its own data-surface decision. |
| **(b)** | Transactional outbox | same as (b′) | Strictly more moving parts than (b′) for the same window. Only worth it for app-level control over what is projected. |
| **(c)** | 2PC | zero | `max_prepared_transactions = 0` at **both** ends; a stuck prepared transaction can wedge HQ. |
| **(e)** | Restructure the assignment write path | — | The separate card the trigger itself names. |

🛑 **The park's own warning, worth repeating:** (b′) being *nicer* than (b) makes the 3am reach for
an async option **easier, not safer**. Both leave a stale-permissive window. Improvising either
without reopening decision 61 is precisely what the park exists to prevent.

**The door is still shut.** B1's four tables remain deny-all (`policy_count = 0`),
`HQ_SYNC_REST_URL` is set nowhere, and B2's roadmap bullet was deliberately left at
`PLANNED — SLATE-READY` rather than flipped — a parked card claiming DONE would tell you the door
is closable.

**Banked so the resuming card does not redo it:** the full port of `ResolveEntityAccess` into
`hq_can_see_template()` is written out in the park note. G6 verified it is a character-for-character
faithful transposition with **both inherited properties preserved** — `assignment_role` referenced
nowhere (an approver still sees what an assignee sees), and the admin arm still a free-standing
unconditional disjunct. `ops.go` is byte-identical to base.

---

## D-2 · 🛑 HQ's timezone ruling collides with a published two-repo contract — and payroll is downstream

**Status:** OPEN — product decision spanning **two repositories**. **Card A1 is PARKED on it and
was NOT merged.**
**Found by:** A1's G6 adversarial review (verdict **REJECT**). The decisive evidence was
re-verified first-hand by the orchestrator before the park was taken.

**The collision.** Ledger decision 83 ruled the app's timezone is `America/New_York`. But HQ
**publishes** a contract to sales-processor that pins `America/Chicago`, and names it as a
coordinated cross-repo assumption:

| Where | What it says |
|---|---|
| `21-SALES-PROCESSOR-CONTRACT.md:27` | "Both are interpreted in `America/Chicago` (the food-truck operating timezone)." |
| `:67` | `completeness.pending_review_ids` is published as `(created_at AT TIME ZONE 'America/Chicago')::date BETWEEN from AND to` — **the exact expression A1 replaced** |
| `:319` | **"A5: `America/Chicago` is the correct operating timezone** … **If the food truck moves to a different TZ, both repos must update.**" |
| `999.2-SALES-PROCESSOR-CONTRACT.md:30` | Same pin, for the menu-COGS endpoint. |

**Decision 83 never addresses this.** It names sales-processor only as a downstream consumer, and
says nothing about the published contract or A5. So the ruling and the contract were both true
statements about different things, and A1 is where they met.

**Why this parked rather than landed.** The card's own PARK trigger names this case exactly —
*"an external contract … sales-processor expecting Chicago — that is a product question, not a
refactor. Park with the evidence."* A1 reported *"Nothing parked — no site turned out to be
deliberately Chicago"*, which is **false against the repo's own artifacts**. The run does not get
to decide a two-repo agreement at 3am.

**The concrete consequence, not the procedural one.** If sales-processor computes its Monday–Sunday
payroll window in Chicago while HQ evaluates the completeness gate in New York, **the two disagree
for one hour at each period edge**, on rows with no extracted `event_date`. A1 did not remove the
two-boundary bug on that path — **it moved the boundary across a repo line, where nothing in this
repository can detect the disagreement.**

**The fork:**

1. **Move both repos to New York together**, and update both contract documents plus A5. Honours
   decision 83; requires coordinated release with sales-processor.
2. **Keep the money paths on Chicago** as a published operating-timezone constant, and apply
   decision 83 only to user-facing/day-boundary surfaces. Splits the app's timezone deliberately
   instead of accidentally — which needs saying out loud, because "one timezone" was the point.
3. **Re-affirm decision 83 as written** and accept that it silently amends A5 — in which case
   sales-processor must be notified before the next payroll run, not after.

**A second, smaller product question rides along** (park note has detail): the Setup-tab Badge
Reset form at `inventory.html:2713` writes the **browser's** timezone into
`repurchase_reset_config.timezone` on every save, which would overwrite A1's migration. An existing
test — `tests/inventory.spec.js:2022`, *"badge reset saves with browser timezone, not hardcoded
value"* — asserts that behaviour and **passes**. Someone once chose "follow the device." No ledger
entry records it. **Should badge reset follow the operator's phone or the app zone?**

**Do not discard the branch.** `card/a1-app-timezone-unify-new-york` @ `8da3ded` is preserved with
its worktree. The review called it well-built work that landed a decision it lacked authority to
land: migration `0072` is clean on fix-forward (two config columns, no reported figure can move),
the red-first evidence reproduced independently, and it found a test that was *asserting the
defect*. **Whichever way you answer, most of that branch is reusable.** The park note lists three
further sites the resuming card must absorb — including `trends.go:89-98`, which would otherwise
leave two 12-week COGS windows on two different zones.

---

## D-3 · Does decision 61 govern this card by letter, or only by analogy?

**Status:** OPEN — small, but it changes what D-1 is even about.

The card text and decision 61 both name the **`app_permissions`** mutation. The projection B2
actually needs is fed by **`template_assignments`** (`repository.go:236` DELETE / `:249` re-insert).
The park note adopts the latter; that substitution was silent until G6 caught it and is now flagged.

It does **not** change D-1's conclusion — `app_permissions` is likewise on `5433` and absent from
`46011`, so the topology argument holds either way. But decision 61's literal text may have been
written about a different table than this card needs. **Applied by letter or by analogy is yours to
say.**

---

## D-4 · Card C1's revised plates are ready for your signature

**Status:** OPEN — **and this is the only thing that unblocks `sync-rxdb-conflict-notice-ui`.**

16 plates, 32 renders, 35 `done_when:` rows, all passing. Two independent gates cleared it; the
restricted-input verifier failed it twice on criteria that could not fail and passed it on the third.

**What you owe:** walk the plates, then give or refuse *"ok, build this"* on revision 2. The card
produced the artifact; **only your signature discharges the block.** It is still ATTENDED-BLOCKED
and that is the correct outcome.

**Two decisions inside the plates are deliberately LEFT OPEN and drawn so you can pick:**

1. **Does a removed-field row count in the chip base, or move to `+N`?** It has no Restore — its
   recovery is *Copy value* — yet it is counted as "1 answer". **Both readings are drawn over
   identical data** (`openq-count-a` / `openq-count-b`), neither recommended, each captioned with
   its own consequence. The arithmetic check accepts either, so nothing has quietly settled it.
2. **The retention window.** 30 days was accepted in decision 80 and reopened at triage. It renders
   only as the placeholder token `⟨30⟩` in a dashed box — never as prose — with `openq-retention`
   showing the same screen at `⟨30⟩` and `⟨7⟩` and body copy byte-identical between them, so you can
   see the screen is indifferent to the value. `⟨7⟩` is **not** a counter-proposal.

---

## D-5 · `tests/sync.spec.js:1198` has been an unfalsifiable instruction for five nights

**Status:** OPEN — process defect, cheap to fix, and it silently weakened five slates.

Line 1198 is `await p.waitForTimeout(400);` **inside a helper's loop body**. It names no test. The
test it used to name is now at **`:1372`**. Two cards hit this independently tonight.

It has been **known dead since 2026-07-24** — `runs/2026-07-24-autonomous/HANDOFF.md:102` says so
and `:164` files an action item to migrate it — and tonight's slate preconditions table still armed
it. So every card told to "expect `:1198`" for five nights was told to expect something that could
not be observed, and every report saying "it passed" was unfalsifiable.

`:446` **is** live and correct (`[LST-17]`). **Recommendation:** migrate the anchor to a title/grep
handle so it cannot rot again, and check whether other armed reds carry line anchors.

---

## D-6 · The file every slate inherits its gates from does not exist

**Status:** OPEN — documentation integrity.

Every slate since 07-15, and tonight's launch prompt, inherit standing rules and gates **G1–G6**
"unchanged from `reference/overnight-run-plan-20260707.md`". **That file is not in the repo.**

The contract was recoverable from practice and the run used it — G1 build+vet, G2 Go+Playwright,
G3 red-first re-verified by G6, G4 `sw.js` idempotence + version parity, G6 adversarial review.
**G5 has no definition anywhere in the tree** and is not practiced.

**Recommendation:** either write the file the prompts point at, or change the prompts to point at
what actually defines the gates — and decide whether G5 exists.

---

## D-7 · `inventory.spec.js:883` — a real red, nobody's card, and normally masked

**Status:** OPEN — backlog candidate, not a blocker.

`item modal pre-fills search with current line item text` fails: `Expected "Special Sauce",
Received "Test Item"`. **Proven pre-existing by reproduction**, not by argument: G6 ran the
preceding specs with B1's new spec file **entirely absent** and got the byte-identical failure, and
`inventory.spec.js` alone on a fresh database passes 150/150. It is cross-spec pollution from one of
`broadcast-rerender` / `grant-enforcement-parity` / `index`.

**Two things worth knowing.** The mechanism first proposed (a `.first()` collision over a shared
`eventDate`) is **wrong** — the pending list is `ORDER BY created_at DESC` with no re-sort, so
`.first()` is the newest row and `event_date` is not in the sort key. The likelier cause is
`seedPendingPurchase` swallowing a failed POST (`tests/inventory.spec.js:70`). Recorded so a wrong
mechanism does not become folklore.

And **`playwright.config.js` defaults to `retries: 1`**, so a normal `task test` may mask this on
retry — which is probably why the baseline reads green. Cards that run `--retries=0` see it.
