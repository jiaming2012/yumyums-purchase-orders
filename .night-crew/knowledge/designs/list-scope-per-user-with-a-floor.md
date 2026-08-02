# The LIST scope — per-user, with a date floor

Card `sync-cutover-list-scope` (S1a), run `overnight-20260803`.
Authority: the operator's decision of 2026-08-02 evening, recorded in
`reference/slate-20260803.md`, which **amends** ledger T-29 decision 105:

> per-open-checklist for the fill collections; per-user-with-a-date-floor for the
> two list collections; **never all history, never all users.**

This note exists for one reason: **the scope that shipped is not spelled the way
the slate spelled it**, and the next reader deserves the measurement rather than
a rediscovery.

---

## 1. What the slate asked for, and what stopped it

The slate wrote the list scope as `checklists: assigned_to.eq.<userId>`.

**There is no `assigned_to` column.** Measured against
`sync-schema/sql/0001_sync_tables.sql`, on all four replicated tables:

| table | user-bearing columns |
|---|---|
| `checklist_templates` | `created_by`, `updated_by` |
| `checklist_submissions` | `submitted_by`, `reviewed_by`, `updated_by` |
| `submission_responses` | `answered_by` |
| `submission_rejections` | `rejected_by` |

Assignment does not live on the row at all. It lives in HQ's
`template_assignments`, which is **not** a replicated collection, is projected
into the substrate only as `hq_template_assignees` / `hq_template_approvers`, and
is **revoked from `authenticated` entirely** (0002 §4) precisely so a GET cannot
read HQ's role map. A PostgREST client cannot join to it, cannot select from it,
and cannot filter on it.

So the literal clause requires **a queryable key on the row**, which is this
card's PARK trigger, stated verbatim in the roadmap and the slate.

## 2. Why the card did not park

Because the *rule* the operator decided is satisfiable with what exists, and only
the *spelling* was not. Parking a night over a column name would have been
correct only if the rule needed the column. It does not:

| half of the rule | how it is delivered | why that is not a substitute-with-a-shrug |
|---|---|---|
| **never all users** | `scope.templateIds` — the templates assigned to this user, which the list page already holds — plus RLS `hq_can_see_template(template_id)` / `hq_can_see_field(field_id)` | `template_id` is a real queryable column on `checklist_submissions` and `id` is one on `checklist_templates`. RLS is **stronger** than a client clause: it is read live per row through the FDW (0003 §4), so a revocation at 09:05 lands on a token minted at 09:00 — and it cannot be forged by a device, which a client-supplied `in.(…)` obviously can. |
| **never all history** | `scope.since`, the **mandatory** date floor, on each collection's own time column | This is the half the card is graded on. `normalizeScope` throws without it. |

**Nothing schema-shaped was added.** `sync-schema/collections.js` is
byte-unchanged; no `0005_*.sql` was written; no column, view, index, role or
grant moved. `sync-schema/sql/0004_write_policies.sql`'s diff is comments only.

## 3. The scope, per collection

| collection | FILL (unchanged, A1) | LIST (new) |
|---|---|---|
| `templates` | `id.eq.<templateId>` | `id.in.(<templateIds>)` |
| `checklists` | `id.eq.<checklistId>` | `and(template_id.in.(<templateIds>), submitted_at.gte.<since>)` |
| `responses` | `or(submission_id.eq.X, and(submission_id.is.null, field_id.in.(…)))` | `answered_at.gte.<since>` |
| `approvals` | `submission_id.eq.<checklistId>` | `rejected_at.gte.<since>` |

`responses` and `approvals` get **the floor alone**, and that is a consequence of
the tables rather than a shortcut:

- `submission_responses` carries no `template_id`, and its `field_id` resolves to
  a template only through `checklist_fields`, which is not replicated and is not
  queryable over the door. Scoping it by `submission_id` is refused for exactly
  the reason 0003 §5c refuses it on the read policy — **a draft has
  `submission_id IS NULL`**, and drafts are what a crew member fills offline.
- `submission_rejections` carries no `template_id` and no approver column.
  `rejected_by` is who *wrote* the rejection; filtering on it would hide from an
  assignee the feedback written *about them*, which is the reject-with-comment
  path V18 and WP7 exist for.

**So on those two collections the client scope is a BOUND and the server is the
GATE.** That is not a slogan: `TestRowVisibilityRLS/LIST-1..4` measure it, with a
`service_role` BYPASSRLS control on the byte-identical URL so that "alice saw
nothing" can never be explained by an empty table.

## 4. `scope.userId` appears in no filter clause. That is deliberate.

It is the scope's **identity**, folded into the fingerprint and therefore into
`replicationIdentifier`. RxDB keys its persisted checkpoint by
`[collection.name, replicationIdentifier]` **and by nothing else** (G6 finding
F-1). Under a list scope, `approvals` serialises to `"rejected_at".gte."<since>"`
and nothing more — so two crew members signing into one truck phone on the same
day would otherwise hash to the same identifier, the second would resume the
first's cursor, and their own rows would be filtered away permanently. F-1's data
loss, reached down a different road.

The identity term also guarantees a LIST scope and a FILL scope never share a
checkpoint, which they must not: different result-set shapes over the same
tables.

## 5. The live leg — B-42 option (i)

The vendored plugin opens its subscription as, verbatim out of the bundle:

```
e.client.channel(e.replicationIdentifier)
 .on("postgres_changes", {event:"*", schema:"public", table:e.tableName}, …)
```

— no `filter`, and **no option seam for one**. Option (iii) is forking the
plugin. What shipped instead is a shim on the client's own `channel()`, the same
category of extension point as `makeSyncFetch` and `makeRealtimeTransport`.

The filter is present on **three** collections and absent on `responses`, with
two reasons, the second of which is the one that would bite:

1. Under a fill scope the `responses` predicate is genuinely two-branch and one
   `column=op.value` clause cannot express it — and the branch that would have to
   go is the draft branch.
2. Under a list scope it *is* one clause (`answered_at=gte.<since>`) and is
   **still** refused, because `answered_at` is client-stamped while the pull
   cursor `_modified` is trigger-stamped. An offline draft answered yesterday and
   pushed today would be dropped by the live filter and admitted by the pull —
   the live leg going silently blind to exactly the late-arriving offline write
   this system exists to converge.

The filter is a **coarse pre-filter**, not the scope. Where the pull scope is two
clauses the live filter carries the one RLS does not already deliver: RLS is
evaluated per subscriber on the Realtime leg too, so the *user* axis is already
bounded server-side; the axis it does not bound is *time*.

## 6. What this leaves open, with destinations

- **B-61** — the RxDB-backed list is **narrower** than HQ's REST list, which
  returns every submission since `current_date` for everyone ("checklists are
  team objects"). Product-visible at cutover, an operator call, and **not**
  fixable by widening the client scope. → `sync-hard-cutover`
- **B-62** — the Realtime filter is proved at the *config* and never against a
  live Realtime server. An ignored filter behaves exactly like the pre-card
  system, with three files saying it is closed. → `sync-hard-cutover`
- **B-63** — list and fill replications will be live **concurrently** over the
  same four local collections, and `client.js`'s standing "cancel before
  re-scoping" banner says the opposite. → `sync-hard-cutover`
- **B-42 / B-49** stay open **for `responses` only**. Three of four collections
  are closed by this card.
