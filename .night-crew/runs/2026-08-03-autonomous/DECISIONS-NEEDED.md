# DECISIONS-NEEDED — run `overnight-20260803`

Two forks, both from card **S1b `sync-hard-cutover`**, which **PARKED**. Both are
operator-level: one is an architecture fork the roadmap's own decision record already
ruled on once (and got wrong on a premise), the other is a product rule.

Full evidence: `park-s1b-sync-hard-cutover.md`.

---

## F-1 — The cutover has no data plane. Where does a field answer live?

**As the owner, I want a crew member's checklist answers to be in the database my reports
and payroll read, so that a shift's work is not silently lost between the phone and the
books.**

### The situation, in one paragraph

The RxDB sync layer replicates against a **second Postgres** — the self-hosted Supabase
substrate — not HQ's. The bridge between them (`sync-schema/sql/0002_hq_fdw.sql`) is
one-directional and carries **permissions**, not data: the substrate reads four HQ views so
its row-level security can decide who may see what. **Nothing carries a checklist answer back
from the substrate into HQ's Postgres** — no worker, no reverse bridge, no backfill.

So "make RxDB the single write path" today would move answers into a database that
`/myChecklists`, `/myDrafts`, **`/submit`**, approvals and every report do not read. The
runner would look correct — answers save, survive reload, reappear on reopen — and Submit
would produce an empty checklist. **The failure is silent.**

This also reopens **ledger decision 49**, whose deciding argument was *"Activity 1 ends in
`sync-hard-cutover`, where RxDB replicates rows straight from Postgres and there is no API
boundary left to translate at."* That premise is false as built, which is the card's own
recorded PARK trigger.

### The three shapes

| | What it means | Cost / consequence |
|---|---|---|
| **(i) Substrate becomes the truth source** | HQ's Go read paths repoint at the substrate for the four replicated tables. | Largest. The only option that actually delivers "one write path". Makes HQ and the substrate mutually dependent — `0002`'s header already accepts that HQ sits on the network path of every row check; this adds the converse. |
| **(ii) Build a substrate→HQ propagation path** | Logical replication, a reconcile worker, or a writable reverse bridge. | Re-opens decision 92's territory. The same-transaction version is *already proven impossible* (`max_prepared_transactions` is 0 at both ends), so any version is eventually-consistent and needs its own conflict rule — a new design, not a card. |
| **(iii) Narrow the cutover** | RxDB owns **reads** (both list views + the fill view); HQ's existing REST path keeps owning **writes**. | 🛑 **This is a parallel run**, which P-KR3 forbids build WOs from proposing. It is listed because it is genuinely available **to you**, not because a card may choose it. Cheapest by a wide margin, and it still buys the two-device live list the widening was bought for. |

**Engineering's read, offered not to pre-empt the call:** (iii) delivers most of the user-visible
value at a fraction of the risk, and its cost is admitting that "hard swap" was scoped before
the two-server topology was settled. (i) is the honest end state but is a milestone, not a
card. (ii) is the one to be most careful with — it looks incremental and is not.

---

## F-2 — At cutover, does a crew member still see their colleague's completed checklist?

**As a crew member splitting a shift, I want to see the opening checklist my colleague
already completed, so that I don't redo it or assume it was skipped.**

This is **B-61**, filed by S1a and addressed to S1b. S1b parked, so **nothing has narrowed
yet** — this is a decision to take *before* the successor card renders a list from RxDB, not
a regression to fix after.

**The facts.** HQ's REST list (`backend/internal/workflow/repository.go`, `myChecklists`)
returns every submission since `current_date` **for everyone**, with no per-user predicate,
under a comment stating the rule as a product decision: *"Today's submissions — checklists
are team objects, all members see all submissions."* The sync substrate cannot reproduce
that: `checklist_submissions_select` is `hq_can_see_template(template_id)`, so a crew member
replicating the list sees submissions on **their own assigned templates only**.

**Two things this is not.** It is not a security finding — the substrate is strictly
*tighter*. And it is not fixable by widening the client scope; the refusal is the server's.

**The options** (S1a's framing, unchanged):

- **(i) Accept the narrowing** and say so in the release note. Cheapest. Arguably the more
  defensible rule — "all members see all submissions" was never an authorization decision
  anyone took deliberately.
- **(ii) Keep the two list tabs on REST**, RxDB-back only the fill view. Note this is the
  same shape as F-1 option (iii); if you pick that there, this resolves with it.
- **(iii) A fifth write/read policy row** — needs **decision 111** re-opened. Not a card's to
  invent.

🛑 **Do not resolve this by widening `scope.templateIds`.** That is the one move that looks
like a fix and changes nothing.
