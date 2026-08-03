# DECISIONS-NEEDED — run `overnight-20260803`

> ### ✅ RESOLVED 2026-08-02 — recorded as `ledger.md` §T-32, decisions 126–128
>
> All three forks were resolved at attended morning triage. This file is kept as the
> **analysis record** — the evidence and option framing below are what the decisions were
> taken against, and are not superseded by them.
>
> - **F-1 → read on sync, write on REST** (option iii). RxDB serves reads; HQ's REST path
>   keeps owning writes. **P-KR3's parallel-run prohibition is WAIVED by the operator for
>   this shape** — a build WO still may not propose it. Ledger decision **126**. Captured as
>   pending preference candidate `architecture/C-3`.
> - **F-2 → nothing narrows.** Both list tabs stay on REST; a crew member keeps seeing a
>   colleague's completed checklist. Resolves as a consequence of F-1, not independently.
>   **B-61 closes.** Ledger decision **127**.
> - **F-3 → one combined notice**, amending ledger decision 106. 🛑 **Nothing has been
>   sent.** The B3/B4/B6–B10 fix-forward corrections come first, and the operator reads the
>   corrected draft before delivery. Ledger decision **128**. Captured as pending preference
>   candidate `process/C-1`.
>
> Still explicitly undecided (F-3 deliberately did not settle these): whether any past
> `ready:false` run needs reconciling, and `menu_item_name` vs `name` — only the
> counterparty can answer the latter.

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

---

## F-3 — Two notices are now owed to the same counterparty. Do they go together or separately?

*(From card **P6 `period-summary-contract-notice`**, which **MERGED**. This is not a blocker —
the notice is drafted and **UNSENT**, which is the card working as designed. It is the operator
act the card was scoped to stop short of.)*

**As the owner, I want the sales-processor maintainer to hear about our contract errors in a
way that reads as one honest correction rather than a drip of separate apologies, so that they
trust the numbers we send them afterwards.**

### What changed since this was last decided

**Ledger decision 106 already ruled on sequencing:** two notices, sent separately, the June
drift first and alone. That ruling stands unless you change it, and the draft should not be
read as re-opening it cold.

But P6's audit surfaced information that did not exist when 106 was taken:

1. **The problem is much bigger than the one row 106 was about.** The audit covered every `:NN`
   row of both contract documents — **111 rows, 45 wrong**. And the sharper half: only a
   minority *drifted*. **22 of the menu-cogs rows were never true at all**, authored 2026-06-04
   at 23:50 from a phase plan, thirteen hours after the handler they describe landed at 10:18.
   A notice scoped to "one expression changed in June" would understate this by an order of
   magnitude.
2. **Card A1's own notice carries an error the audit found** (`:31`/A10 — it attributes a
   timezone claim to `/menu-cogs`, which contains no `AT TIME ZONE` at all). Sending A1's notice
   alone would propagate a fresh error while apologising for old ones.
3. **A1's notice appears never to have been drafted.** The P6 draft asserts it was "drafted
   2026-08-01"; no such draft exists anywhere in the repo, and decision 106 records it as *owed*,
   not written. **The draft's own claim here is wrong and needs correcting before it goes
   anywhere** — flagged as B3 in the triage checklist.

### The shapes

- **Hold to decision 106** — send the June drift notice alone, first. Cheapest, already decided,
  but it now describes a small fraction of what we know is wrong, and a second larger notice
  lands days later.
- **Amend 106 and send one combined notice** — the drafted P6 notice already covers both
  documents and all 111 rows. One conversation, one apology, complete. Costs: it is a bigger
  and more alarming message, and A1's notice must be folded in rather than sent.
- **Send P6's now, A1's never** — if A1's notice was never drafted and its content is a strict
  subset of P6's audit, there may be nothing left for it to say. Verify that before choosing it.

🛑 **Two things stay yours regardless, and the card deliberately did not decide either:** whether
any past `ready:false` run needs **reconciling** (the draft says "we are not proposing to restate
any past figures — if you would rather we did, say so"), and the **`menu_item_name` vs `name`**
question, where renaming HQ's key *fixes* a client built from the doc and *breaks* one built from
the wire. HQ cannot see which exists; only the counterparty can answer it.
