# Decisions needed — run `overnight-20260804`

> **RESOLVED 2026-08-03 — recorded as ledger §T-34, decisions 135–141.**
>
> This file had **no open fork and no parked card**, and that emptiness was verified rather than
> taken on trust: `night-crew decisions ratify --run 20260804` reported nothing awaiting
> ratification, `preferences ratchet` nothing survived, and `decisions audit` no gray areas routed —
> all three consistent with the *Closed during the run* section below. Steps 3b and 3c of morning
> triage were genuine no-ops.
>
> What triage did decide, from the *For triage* section: **B-89 and F2 ride the next night**
> (decision 137); **B-26** gets a repo-local ladder at
> `reference/gate-ladder.md` (decision 138); **B-80 and the launch-prompt
> isolation gap** are answered as one decision in that same file, with `TEST_DB_NAME` added
> (decision 139); **B-105** is answered — HQ keeps branch-and-commit, no OpenSpec (decision 140).
> The operator's rider — *"agents decide implementation details"* — is recorded as a standing rule,
> which is why 138–141 were taken at role level rather than escalated.
>
> Triage also found what this run did not: a correction-photo coverage gap that a literal B-65
> mutation survived, **fixed attended** (decision 136, `[FLD-16C]`, commit `70ea466`), and five new
> backlog entries **B-131**–**B-135**. Kept as the analysis record.

Open forks and parked cards from this run. Morning triage resolves these with the operator.

**Entering the run: no open forks.** F-1/F-2/F-3 were resolved at triage 2026-08-02
(ledger §T-32, decisions 126–128); tonight's own calls were recorded in advance at ledger §T-33
(decisions 132–134).

---

## Open

**🛑 NOTHING IS PARKED, AND NO FORK IS OPEN.**

This section is deliberately empty, and the emptiness is a *result* — not a section that never got
written. All four slated cards (A1, A2, A4, and the A6 stretch) landed and merged. No card hit a
PARK trigger, no merge conflict required an operator judgment, and no question arose that only the
operator could settle.

The four PARK triggers the slate armed were all approached and none fired:

| Trigger | Card | What happened |
|---|---|---|
| A new `night-crew.toml` key or schema change | A1 | Never needed — `night-crew.toml` was **not touched at all**. The existing `subset` line resets as it stands, because the fix went into `playwright.config.js`'s `webServer.command`. |
| A new field type or a change to the persistence contract | A2 | Never needed — `CLAUDE.md`'s rule survives intact; only the false function name, transport and call shape were wrong. G6 verified the rule was not weakened. |
| Deciding a product fork rather than describing it | A4 | Avoided explicitly — the note states in as many words that adopting §8 is a roadmap decision, not a card's. G6 checked this specifically. |
| Feeding the badge from `/api/v1/health` | A6 | Never needed — the precached `version.json` was readable, and the implementation has **no** API fallback. |

**Decisions the run made for itself** (role-level, decided and stated rather than escalated, per the
standing rule that PM/PjM/Engineer-level calls get decided here):

1. **Reconstructed HQ's G1–G6 ladder from `slate-20260803.md`** when `slate-20260804.md`'s inherited
   pointer (`reference/overnight-run-plan-20260707.md`) proved to be a file that has never existed in
   this repo. Mechanism, not a fork. **B-26 recurring** — see HANDOFF §"Two process defects".
2. **Issued a unique `TEST_DB_NAME` to every leg** once A1 landed and made every Playwright
   invocation destructive to its target database. Tonight's launch prompt carries no isolation
   stanza at all, so this came from the orchestrator ad hoc. **Filed B-80**; the remedy is a
   launch-prompt template change, which *is* a triage decision.
3. **Fixed B-92 rather than filing it.** A6 introduced a regression — the new spec red on any clean
   checkout — and leaving it would have handed every future card leg a spurious red, partially
   undoing what A1 landed the same night. Fixing a defect a card introduces is in-scope, not creep.
4. **Filed B-87 with G6's corrected consequence, not the implementer's.** The implementer claimed a
   confined gate could silently run the *wrong* specs; G6 refuted it at source (the CLI filters are
   OR'd, so it can only over-run). The wrong version is struck in place rather than quietly rewritten.

None of these needs ratification. They are mechanism calls on the citation floor, not delegated
policy decisions — see CLAUDE.md §"Slate workflow" step 5 on the difference.

---

## Closed during the run

No question reached `night-crew decisions log` — none arose that was a gray area rather than a
mechanism call. Recorded explicitly so that "nothing logged" is not read as "the logging never ran."

---

## For triage — not blocking, but worth a decision

These are backlog items, not forks. They are listed here only because each is a *choice* rather than
a queued task, and the HANDOFF's next-actions section points at them.

- **B-89** is a live bug in shipped code (`cachedGrantSlugs()` returns `[]` unconditionally). Decide
  whether it rides the next night or is fixed attended.
- **B-80 and the launch-prompt isolation gap are one decision**, not two.
- **B-26** — decide where HQ's gate ladder lives, so the next slate stops inheriting from a
  nonexistent file.
- **B-105** remains open and untouched: which per-change discipline this repo adopts. No card
  answered it, deliberately — it is the operator's.
