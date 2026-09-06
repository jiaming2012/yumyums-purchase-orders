> **RESOLVED 2026-09-05 — recorded as `.night-crew/knowledge/ledger.md` §T-55 (decisions
> 173–177).** No fork was open, so nothing here needed a ruling; the two awareness items
> below were both acted on at triage. B-26's dangling pointer is carried back to the clone
> (unfixed hq-side, since the template that emits it is clone-side); `gate-ladder.md`'s two
> stale lines were **fixed in the file** rather than carried — decision 176. Kept as the
> analysis record.

# Decisions needed — run `20260906-2`

## Nothing parked.

The night's one card landed. **No card hit a park condition**, and none of the three narrow
PARK notes the slate made binding was approached:

- **A genuinely new terminal status** beyond the §9/§19 attempt taxonomy — **not needed.**
  The discriminator rides as a boolean column on `accepted`; the arbiter's distinct-status
  read is `accepted` alone, and the harness reds if any status outside
  `accepted|rejected|pending` appears.
- **Any weakening of the `requires_online = true` refusal** — **none.** The change is
  strictly a tightening: known codes with an unresolved campaign now refuse where they
  previously offered the override.
- **A new preference category or `night-crew.toml` key** — **none created.**

**No gray area was routed through `night-crew decisions log`**, so there is nothing here
awaiting ratification and no delegated record coming back at triage.
`night-crew decisions audit --run 20260906-2` reports *"No gray areas routed through the
resolver yet."*

The engineering calls the card made — the UI copy branch, the discriminator name, rider
(a)'s coupling, the schema-migration strategy — were **decided and stated**, not escalated.
They are role-level calls, which is what the slate said they were: *"the fail-closed
predicate mechanics, latch design, schema-migration strategy, discriminator field name, and
the build-fact-6 UI copy call are the night's."* They are recorded in HANDOFF.md
§"Decisions the card made" for review, not for a ruling.

---

## Not a fork, but the operator's by nature — carried, not asked

**Close-bar leg 3 is now attestable.** It was 🛑 BLOCKED on B-432; B-432 is closed as of
`afc9e97`. **The attestation itself is the operator's act** (decision 161's class) and no
overnight can perform it. This is a next action in HANDOFF.md, not a question — nothing is
being asked here, it is simply the one thing tonight unblocked that only you can finish.

---

## Raised at launch, for triage's awareness — neither blocked the night

1. **B-26's dangling pointer regressed in the slate template.** The launch prompt inherits
   G1–G6 from `reference/overnight-run-plan-20260707.md`, **a file that has never existed
   in this repo**. Decision 138 (triage 2026-08-03) already ruled that HQ slates cite
   `.night-crew/knowledge/reference/gate-ladder.md` instead. Resolved to that file at
   launch; nothing was blocked. **The fix is clone-side** — the template that emits the
   dead path — same as B-14's remedy.
2. **`gate-ladder.md` is stale in two places.** Precache count reads **31** (actual **43**),
   and the environment section still names Postgres **`:5433` / `yumyums:yumyums`** — the
   **production** cluster. Both were overridden deliberately at dispatch and every leg ran
   on `:5434` / `hqtest` per decision 155. **A future run that trusts that line points a
   suite at production.** Worth fixing in the file itself, hq-side, at triage.
