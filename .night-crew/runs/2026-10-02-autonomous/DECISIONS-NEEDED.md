# Decisions needed — run `20261002`

Two open operator forks, both raised by Card 1's G6 adversarial review, both routed through
`night-crew decisions log --run 20261002` and both returned **`verdict: park`** — *escalated,
top severity, voted top by all three roles; "top-severity questions are the operator's — no
number of citations clears them."* Recorded in
`.night-crew/knowledge/decisions/20261002.jsonl`.

Neither is in Card 1's narrow PARK note. Both are nonetheless the operator's by the ladder's own
verdict, so the run did **not** choose. Card 1 **merged anyway** (`40b5846`) — see the bottom of
this file for why that was the orchestrator's call and not a silent override.

---

## D-1 · Migration 0083's `menu_items` foreign keys block the Recipes dish merge

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h1-campaign-codes-api` · **Reproduced:** yes, by G6, against both FK shapes

`0083_campaigns_admin.sql:40` and `:57` declare `campaigns_admin.item_id` and `qr_codes.item_id`
as `references menu_items(id)` with **no `ON DELETE` clause**. Both pre-existing `menu_items`
FKs (migrations 0061, 0062) are `ON DELETE CASCADE`.

**The failure, reachable today via the API:** a manager `POST`s a campaign with
`item_id=<wings>`; someone then opens Inventory › Recipes › By dish and merges "Wings" into
"6pc Wings"; `recipes/repository.go:162`'s `DELETE FROM menu_items` raises
`23503 campaigns_admin_item_id_fkey`; the merge returns an **opaque 500** and stays broken until
the campaign is deleted. G6 reproduced the exact UPDATE/DELETE sequence.

🛑 **This becomes reachable through the UI the moment Card 2 (`campaigns-tab-ui`) lands**, which
makes choosing an `item_id` a routine operator action.

**Why it is yours and not ours:** the `0083` DDL is **verbatim** from the signed handoff §4, so
every available fix deviates from the signed spec somewhere. §4 omitted the clause rather than
choosing none — it never considered the merge path.

| Option | What you would see |
|---|---|
| **`ON DELETE SET NULL` on both FKs** *(run's recommendation)* | The dish merge succeeds. The campaign and its QR codes survive with a null item; `utm_content` degrades to empty on an already-printed code. Adds a clause §4 omitted. |
| `ON DELETE CASCADE`, matching 0061/0062 | The merge succeeds, but **silently deletes the campaign and its `qr_codes` rows** — a dish rename destroys scan history. |
| Re-point `item_id` inside `MergeMenuItem` | `0083` stays byte-verbatim to the spec, but the fix lives in the `recipes` package and every future table referencing `menu_items` must remember to join the merge path. |

**The run's recommendation:** `ON DELETE SET NULL`. A campaign must not be destroyed by a dish
rename (rules out CASCADE) and a dish merge must not be blocked by a campaign (rules out the
status quo). Losing one UTM parameter is the cheapest honest degradation.

---

## D-2 · The 10-minute scan dedupe is not atomic, and `scans` is P-KR3's denominator

**Status:** OPEN · **Severity:** MEDIUM (ladder: top, parked) · **Raised by:** G6 on card
`h1-campaign-codes-api` · **Measured:** yes, 5 runs

`landing.go:306-312` dedupes with `INSERT … SELECT … WHERE NOT EXISTS` — a read-then-write under
READ COMMITTED. G6 drove **12 concurrent `GET /q/{short}`** with one identical
`CF-Connecting-IP`: **8 `qr_scans` rows in 4 of 5 runs** (1 row only on the cold-pool first run).

Card 1 enforces dedupe at **write** time only — a deliberate, stated choice so the §5 metric is a
plain `count(*)` with one definition in one place. The consequence is that **no read-side
correction is available** to Cards 4 and 5, which consume `scans`. `scans` is the **denominator of
the orphan rate P-KR3 is graded on**, so the KR inflates by an unknown factor under concurrent
double-taps.

| Option | What you would see |
|---|---|
| **Tumbling 10-minute bucket column + unique index on `(short, ip_hash, bucket)` + `ON CONFLICT DO NOTHING`** *(run's recommendation)* | The count is correct under concurrency. Adds one column and one index that §4 does not enumerate. |
| Leave as measured | `scans` over-counts; P-KR3 is graded on an inflated denominator; later cards inherit a number they cannot correct. |
| Advisory lock around the existing read-then-write | No schema change, correctness restored, at the cost of a serialization point on an unauthenticated public route. |

**Note on grading:** if you take option 2, P-KR3's orphan rate for this cycle should be read as an
**upper bound**, not a measurement.

---

## Not decisions — recorded so triage does not mistake them for forks

- **Four LOW/INFO G6 findings on Card 1**, all engineer-level and none parked: the public landing
  `503`s and logs at ERROR on a malformed `short` (`landing.go:256` — unauthenticated, unthrottled
  log flood; the admin routes have the analogous `isBadUUID` guard, the landing has none);
  `PATCH /codes/{id}` accepts an undocumented `item_id` and answers `404 code_not_found` for a code
  that exists (`codes.go:325`), and its `COALESCE` means `landing`/`placement`/`variant` can never
  be cleared back to NULL; one code is minted per channel **entry** rather than per distinct
  channel (`campaigns.go:191`); and `live()` ignores `starts_at`/`status='scheduled'`
  (`landing.go:212`), unreachable today but live the moment Card 2 adds scheduling. These are
  follow-ups, not questions.
- **Card 1's three non-baseline Playwright reds are flake, not a fork.** Re-run green confined at
  the card HEAD twice, independently (orchestrator + G6); one baseline red went green in the same
  run, so the set flakes both ways under load. No causal path from `0083` or the `main.go` mounts
  to the Inventory/Recipes frontend surfaces was found by either reviewer.
- **The `dev` browser baseline is 24 reds, not the 4 of B-433.** Measured tonight on a detached
  `dev` worktree at `c4f6db4` (24 failed / 965 passed). Mostly the 17-test undiagnosed cluster
  `bugs.md` already carried; **four reds were new to the record** and are now named there
  (`09c6586`). Not tonight's, and no card owns that surface.
- **Two gate-ladder defects were corrected mid-run** (`dbbc1a8`), not parked: the per-leg isolation
  stanza never stated that `HQ_RLS_TEST_DB` must match `^hq_rls_[a-z0-9_]+$`, which cost the
  base-reds leg a discarded run whose three `internal/sync` reds read exactly like a substrate
  defect; and G4's precache count read **43** against a tree holding **48** — the second recurrence
  of that staleness in that row, which would have let a genuine drop to 43 read as the documented
  number.

---

## Why Card 1 merged with D-1 and D-2 open

Stated plainly so triage can disagree with it. The ladder parked **which fix to apply**; it did not
rule that the card's delivered scope was wrong. Card 1's five `done_when` tests, its DDL-vs-§4
equivalence, its Down round-trip and its live projection were each independently re-verified by G6
in a fresh context. Against that, parking Card 1 would have ended the night at **zero** merged
cards, because the remaining six structurally cannot start without `0083` and the `marketing`
package seam (the launch prompt's own instruction for that case is to close out early). Nothing
deploys tonight: `main` is untouched, the deploy is independently held, and no UI yet reaches the
`item_id` path that D-1 needs. Both findings carry a reproduction, three options and a
recommendation, so each is a short conversation at triage rather than an investigation.
