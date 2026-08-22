# Night-crew design findings — pending graduation to night-crew main

> **Purpose.** A standing, cross-run sink for findings that are **night-crew framework
> changes, not HQ-app changes** — surfaced while dogfooding night-crew on HQ. Distinct from
> `BACKLOG.md` (HQ-app items). Mirrors the per-run `runs/<date>/design-findings.md` precedent
> (`runs/2026-07-09-attended/design-findings.md`), but durable across runs so findings aren't
> orphaned.
>
> **The loop this file feeds (operator intent, 2026-07-14):** entries here → a post-run/
> post-cycle **batch against the night-crew main repo** → the fix lands in main → the **tool-pin
> advance** (roadmap.md "Tool pin" standing rule) propagates it to HQ and every other project
> pulling from that pin. Today the sweep is un-triggered (no guaranteed producer); the desired
> end state is a feedback step — ideally at cycle/roadmap completion or in `/nc-morning-triage`
> — that graduates these to main and advances the pin.

## Open findings

### NF-1 · `/nc-slate-plan` budget sizing is systematically over-conservative (under-packs the night)
- **Symptom (persistent on HQ):** slates finish far under budget. `overnight-20260714` used
  **~2h26m of a 6h budget (~41%)** and barely dented the token/quota budget.
  `reference/card-actuals.md` documents the estimate bias directly: cards ran **6–10× under**
  first-of-kind estimates (first slate), then **2–4× under** (second slate; e.g. `ops-nfr3`
  23m actual vs 45–75m estimate).
- **Root cause (in night-crew main, the `/nc-slate-plan` skill — §2b cut-line logic):** the
  cut line is drawn against the estimate's **high end**, with a full ~30 min closeout reserve
  and conservative stretch-gating, so the night is under-packed even as the actuals ledger
  shows the estimates themselves are already high. The skill treats the budget as a cautious
  ceiling rather than a floor to fill.
- **Operator directive (2026-07-14):** treat the stated night budget as a **LOWER limit**, not
  a conservative upper bound. Size slates ambitiously; lean **concurrent** dispatch when
  footprints are disjoint (multiplies throughput); set generous stretch so the run never idles
  with budget in hand. The quality bar (G1–G6, red-first) is the guardrail — throughput should
  push against it. Prefer "attempt more, exit clean-early" over sandbagging.
- **Proposed fix (night-crew main):** re-anchor §2b estimation on the *recorded actuals'
  central tendency* (not the first-of-kind high end); make the cut line pack to the budget with
  a small fixed margin; and, since actuals now show wide headroom, make **concurrent** the
  default recommendation whenever §5's disjoint-footprint criteria hold. Calibrate against
  `card-actuals.md` size-class ranges.
- **Scope note:** the fix belongs in night-crew **main**, not HQ — HQ is only where the symptom
  is persistent. Until the pin advances, `/nc-slate-plan` on HQ compensates by sizing
  ambitiously per the operator directive above.

### NF-2 · The design-findings feedback loop has no guaranteed producer / no cycle-completion trigger
- **Symptom:** `design-findings.md` — the channel for framework findings → night-crew main —
  has fired **exactly once** (2026-07-09). Nothing *triggers* a findings sweep; a finding is
  captured only if someone remembers to write it. This is the same **NO-ORPHAN-INPUTS**
  violation that the first design-findings doc (Finding 1) was itself written to fix.
- **Operator intent (2026-07-14):** there should be a feedback step — "maybe after the roadmap
  is done" (cycle/roadmap completion) or folded into `/nc-morning-triage` — that (a) sweeps
  persistent framework findings into a durable sink (this file), (b) batches them against
  night-crew main, and (c) advances the tool pin so all consuming projects inherit the fixes.
- **Root cause — analyzed 2026-08-01. A two-factor law that predicts every working and
  non-working step in the system: a ritual step fires IF AND ONLY IF (a) a verb exists for it
  AND (b) a skill invokes that verb.** Neither alone suffices.

  | Step | Verb? | Invoked? | Actual |
  |---|---|---|---|
  | backlog capture | ✓ | ✓ **11 skills** | 38 findings, healthy |
  | OKR grade | ✓ | ✓ dev-close, milestone-close | graded every cycle |
  | preference *propose* | ✓ | ✓ triage §4, grill-back §4 | 3 captured |
  | **`decisions log`** | ✓ | ✗ **no skill invokes it** | **0 routed, ever** (B-03) |
  | **preferences *adopt*** | ✗ **no verb** | — | **0 of 3 adopted** (NF-5) |
  | **design-findings sweep** | ✗ | ✗ **0 skills reference the file** | **0 of 6 graduated, 18 days** |

  The two failure modes need different fixes: `decisions log` is a **one-line skill change**
  (the verb already exists, nothing calls it); the other two need code.
- **Why this file specifically — THREE deficiencies, not one:** (1) **no CLI verb** — nothing in
  `night-crew`'s surface mentions findings or sweeps; (2) **no skill reads or writes it** — 0 of
  the 11 `nc-*` skills reference it, against **11** for `BACKLOG`; (3) **not scaffolded by
  `init`** — the scaffold creates `okrs.md`, `bugs.md`, `rulebook.md`, `roadmap.md`, `ledger.md`,
  `backlog.md`, `CONTEXT.md`, `DESIGN.md` and the five preference files, and this file is not in
  the list. On a fresh target repo **it does not exist at all**; hq's copy survives only because a
  human made it and humans keep opening it by path. This is DESIGN §15l's **"lost feature"**
  failure mode — the one the inbox CLI feeder exists to prevent — recurring here with no feeder.
- **🛑 The structural reason, and it is the operator's own bug class one level up.** Night-crew's
  rituals are **prose**, and a prose step degrades **silently**: an agent that skips *"you should
  also sweep findings"* emits output indistinguishable from one that did it. There is no red.
  That is exactly hq's characteristic defect — B-09, B-14, B-16, B-22, B-23, B-24, B-36, B-37 are
  all *a check whose subject set can go empty reports green*. **A ritual step with no verb and no
  check is a guard whose subject set is ALWAYS empty; it always passes, so nobody ever sees it
  fail, so nobody ever fixes it.** hq's characteristic bug is night-crew's characteristic bug.
- **Proposed fix (night-crew main):** add a guaranteed producer for the design-findings sweep at
  a cycle boundary. Candidates: a step in `/nc-okr-session` (cycle start reviews last cycle's
  findings) or a closeout step in `/nc-morning-triage` / a new cycle-complete skill. Make this
  standing file (`knowledge/design-findings-nightcrew.md`) the canonical target, scaffolded by
  `night-crew init` alongside the other knowledge files.
- **🛑 Meta-fix, without which the above rots identically: the new producer must itself be
  CHECKABLE.** Add `night-crew findings check`, exiting non-zero when an NF entry has sat N days
  with no disposition, so a ritual **cannot** report clean while findings rot. A producer added as
  prose is just more prose — and in six weeks the file carries NF-8, explaining why NF-7 never
  fired.

### NF-3 · `/nc-morning-triage` skill references CLI features not deployed to night-crew `main` (skill/tool skew)
- **Symptom (hit at triage 2026-07-20):** the skill's capture-on-answer step (§3.4) instructs
  running `night-crew preferences propose --repo <target-repo>`, and §4.7 instructs
  `night-crew decisions audit`. Neither subcommand exists in the installed CLI — both live only
  on night-crew `dev` (57 commits ahead of `main` at the time), while hq's tooling tracks `main`
  (`nc:update tracks main`, commit `65ddf9a`). The triage had to skip both steps.
- **Operator rule (2026-07-20, verbatim intent):** "Only should consider whats been deployed to
  main." Target-repo rituals must only depend on main-deployed tool features; a skill step naming
  an undeployed subcommand is skipped-with-reason, never satisfied by rebuilding the CLI from dev
  mid-ritual (this happened briefly at this triage and was reverted the same session).
- **ROOT CAUSE (corrected same day, attended follow-up):** the skill did not "get ahead of
  main" — **the `nc:update` pin is defeated by the clone's checkout.** `nc:update` = fetch +
  `git checkout main` + ff-only merge + `install.sh`, and `install.sh` symlinks every
  `~/.claude/skills/nc-*` **into the clone's working tree** ("so `git pull` updates them in
  place"). The pin therefore holds only while the clone stays checked out on `main`. The WSL
  clone is checked out on `dev` (56+ ahead, dirty) for night-crew's own development — so every
  nc-* skill has silently served **dev** text since that checkout, while the installed binary
  remained main-vintage. Verified: `git show main:.claude/skills/nc-morning-triage/SKILL.md`
  has **0** mentions of `preferences propose`/`decisions audit`; the working tree (what the
  symlink serves) has 2. Binary-vs-skill skew was the *symptom*; working-tree-tracking
  symlinks are the defect.
- **Fix candidates (night-crew main):** (a) `install.sh` symlinks into a **main-pinned
  worktree** (`git worktree add ../night-crew-main main`) instead of the development checkout —
  the pin then survives any branch switching in the dev clone; (b) or `nc:install` COPIES the
  skills at install time so they change only when `nc:update` runs (loses the pull-updates-
  in-place property, gains checkout independence); (c) or keep the rule "the installed clone
  never checks out anything but main" and do night-crew development in a separate clone/
  worktree; (d) independently: promote preferences/decisions dev → main, and add a capability
  probe to skill steps naming subcommands ("not deployed — skipped" report line).
- **Cost of the gap:** two operator-answered riders (umbrella grants; everyone-sees-live-ops)
  could not enter the preference pending queue and live only in the target repo's ledger/design —
  re-offer them if/when the machinery ships.

### NF-4 · Run-mechanics defects surfaced by overnight-20260721 (brief-template candidates)
- **Never-background rule (cost ~25–30m):** A1's implementer twice suspended itself by
  backgrounding long suite runs and had to be resumed by the orchestrator. The run fixed it
  forward in every later brief ("never background; foreground legs ≤10m, detach+`tail --pid` for
  longer") — graduate that sentence into the standing brief template so no future run re-learns it.
- **Compose pg publishes no host port:** target-repo ephemeral pg16 legs needed a scratchpad
  compose override adding `ports:` — 5 of 6 agents used the same workaround independently. Add a
  commented-out ports stanza or a documented override file to the sandbox/brief so it's one copy
  step, not six rediscoveries.
- **`task test` surfaces Playwright failure as go-task exit 201, not 1** (INFO): gate logic should
  grep the Playwright summary line, not name the exit code.

### NF-5 · The preference loop is broken at every stage — answers are captured, never adopted, and the unattended run has no channel to them anyway
- **Symptom (operator-stated, 2026-08-01):** *"in theory… define a milestone, grill any gray areas,
  and let the run run unattended until the milestone is done. in practice, i am constantly giving
  inputs even after the milestones have been defined."* Measured in hq: **113 numbered decisions
  across 21 triage sections**, **~5 forks per run** over the last 9 runs (3·7·1·4·7·9·7·4·3), across
  20 runs in 23 days — plus ~9 attended ritual gates per cycle.
- **The loop as designed** (`/nc-morning-triage` §4 capture-on-answer, §7 preference coverage) is
  exactly the right shape, and its own text states the intent: gray area → routed through the
  decisions resolver → answered at triage → offered back as a candidate preference → adopted →
  `decisions audit` measures coverage → named gaps become the next evening's offer-back, *"each …
  a question the operator could answer once and never be asked again."*
- **Root cause — every stage is unwired, measured against hq:**

  | Stage | Designed | Actual |
  |---|---|---|
  | 1 · gray areas → resolver | every fork | **0 ever.** `decisions audit` → *"No gray areas routed through the resolver yet"*; runs park straight into `DECISIONS-NEEDED.md` |
  | 2 · offer back as candidate | per answered fork | **3 candidates, total** |
  | 3 · operator adopts | renumber `C-n` → `P-n` | **0 of 3 adopted** |
  | 4 · coverage measured | `decisions audit` | no denominator — *"decorative every cycle"* (B-03's own words) |
  | 5 · gaps → next offer-back | shortlist | never runs |

  (a) **No producer for stage 1.** Already filed as **B-03** (hq BACKLOG, *morning triage
  2026-07-25*), whose lead is the fix: *"wire the park path to route through it so the coverage
  number has a denominator."* Seven days later B-03 is itself still `· new` — the bug describing why
  the operator keeps being asked things is unrouted by the same routing failure, one level up.
  (b) **No adoption verb.** `preferences list|validate|propose|pending` — there is no
  `promote`/`adopt`. Adoption means hand-editing markdown: renumber `C-n` to a free `P-n` and move it
  above a section divider. **DESIGN §15l's consent boundary is correct and should not change** — the
  defect is that expressing consent costs a manual file edit, so it never happens.
  (c) **The unattended run has no channel.** `nc-slate-plan`, `nc-run` and `nc-pm-session` cite
  preferences **0 times**; only `nc-pm-grill-back` and `nc-morning-triage` read them, and **both are
  attended**. `slate-20260801-2.md` and `launch-20260801-2.md` cite them 0 times. So even a fully
  adopted preference never reaches the one actor that is supposed to stop asking.
  (d) **The only end-to-end-wired category is `constitutional`, whose semantic is "always
  escalate."** 6 adopted entries, thresholds mostly `always`. The single part of the preference
  system that works start-to-finish is the part that *generates* operator interrupts; the parts that
  would prevent them (`architecture`, `design`, `ux`) parse to **0 entries** despite holding content.
- **Concrete harm, same night:** `architecture/C-2` was captured 2026-07-31 (hq commit `43492c5`) —
  *"Any client-side fetch or replication over a collection that can grow without bound is batched and
  scoped — never pulled whole. Scope it to what the current view actually needs (for workflows, the
  open checklist)."* That is card **A1 `sync-replication-scope-per-checklist`** stated as policy — its
  acceptance test is *"a device does NOT hold rows for a checklist it never opened."* The operator
  stated the general rule the evening before; it sits `pending`; the slate that plans the card
  implementing it cites preferences zero times. An A1 implementer hitting an edge C-2 already answers
  parks and asks.
- **Lost-feature irony (DESIGN §15l(b)):** the inbox CLI feeder exists specifically because *"a bare
  drop-zone folder tends to become a 'lost feature'"* — the operator's own diagnosis. The preferences
  **Pending** section is now exactly that: a drop-zone with a feeder in (`preferences propose`) and
  none out.
- **Proposed fix (night-crew main), in leverage order:**
  1. **`night-crew preferences adopt <category>/C-n`** — one command, operator-run, does the renumber
     and the move. Preserves §15l consent; removes the friction that is eating it.
  2. **Wire the park path through the decisions resolver** (B-03's lead) so stage 1 has a producer
     and coverage has a denominator.
  3. **Give the run a preference channel** — `/nc-slate-plan` reads adopted preferences and cites the
     relevant ones per card; `/nc-run` pastes them into every implementer and G6 brief, the same way
     it pastes the verification ladder.
  4. **Add a delegation semantic to balance `constitutional`.** Today a preference can only say
     *always escalate*. Add the inverse — *decide under this policy and file for ratification* — so an
     answered gray area converts future instances from a **blocking park** into an **asynchronous
     ratification** at triage. This is the change that actually reduces back-and-forth; 1–3 only make
     the existing loop functional.
  5. **Grill for POLICY, not only for instances** (`/nc-pm-grill-back` §4). Capture-on-answer today
     offers back *the answer to this gray area*; nothing ever asks whether it should govern the class.
     That is why the captured candidates are uneven — `architecture/C-1` (timezone) generalises,
     `ux/C-1` (struck-through label) is instance-shaped. **Add one follow-up on the resolved door:**
     *"Is this a standing rule for this class, or a one-off here?"* A yes yields an
     operator-worded **policy**, which satisfies §15l and is the only kind of capture that
     reduces future grilling. Instances can never be pre-grilled; policies can — and the gray
     areas that actually park runs (T-29 dec. 105, T-30 dec. 111 and 113) were all discovered at
     **build/G6 time**, after the PRD grill had already closed.
  6. **Raise the unattended session's autonomy ceiling — three coupled changes, none sufficient
     alone.** Fix 4 supplies the data model; without these the run still cannot act on it.
     (a) **Bound the park rule.** Replace the per-card *"PARK if any operator-only question
     surfaces"* with *"PARK only if the question is not covered by an adopted delegation policy;
     otherwise decide under it, record the decision and its governing `P-n`, and continue."*
     Slate-template + launch-prompt change.
     (b) **Retire "the run never decides, it executes"** (`launch-20260801-2.md`) as an
     unconditional. It is correct for *product forks* and wrong for everything a policy already
     covers; state it as the former.
     (c) **Ratify at triage.** Every decision the run took under delegation is reviewed next
     morning — ratified or reversed. This is what makes (a) safe: nothing becomes silently
     permanent, and the operator's input moves from **synchronous blocking** to **asynchronous
     review**.
     **Ratchet:** a delegated judgment that survives a cycle unreversed is evidence it should be
     offered as a policy — which is the missing feedback edge that would let autonomy *compound*
     instead of resetting each cycle, as the delegated door makes it do today.
- **Prior art the operator has already written, in the wrong store:** two delegation policies exist
  today in Claude Code memory — *"ask about product forks and intent; decide mechanism yourself and
  prove it by execution"* and *"in night-crew attended skills, decide PM/PjM/Engineer-level calls
  yourself and state them; don't ask the operator to make or bless them."* They govern the
  interactive assistant and **have no expressible form in night-crew**, so no run inherits them.
  That they were written at all is the requirement statement for fixes 4–6.
- **Scope note:** all four belong in night-crew **main**, not hq — hq is only where the symptom is
  measurable. Until the pin advances, the compensating move is to adopt the 3 pending candidates by
  hand and paste them into slates manually.

### NF-6 · The knowledge store has no scope axis — it cannot flow across repos, cannot segment operators, and the NF-5 ratchet has nowhere to land
- **Requirement (operator, 2026-08-01), three constraints at once:** knowledge must **flow across
  multiple target repos**; operators must be **segmented** (one operator's leanings must not silently
  govern another's sessions); **except** where two operators share a repo, where a common policy is
  *desirable* — *"the policy decisions might create consistency, which is positive."*
- **Current state — greenfield, verified:** no scope concept, no multi-operator concept, no global
  store (`~/.night-crew` does not exist), and the entry grammar (`Preference` / `Why (operator)` /
  `Weight` / `Evidence` / `Recorded`, plus `Threshold` on constitutional) carries **no operator and no
  scope field**. The store is repo-local and single-operator **by assumption**, never by decision.
- **Root cause:** the store conflates two different kinds of thing under one axis. `process/P-1..P-4`
  (clean DB, separate schemas) are *how this operator works* — portable to any repo, private to them.
  `architecture/C-1` (America/New_York) is a fact about *this business* — must bind **any** operator
  on hq, and is meaningless in another project. `architecture/C-2` (batch unbounded fetches) is an
  *engineering conviction* — portable across repos and reasonably binding on co-operators. All three
  live in one flat, repo-local store. **Scope is orthogonal to category and the axis does not exist.**
- **Proposed model — three scopes:**

  | Scope | Lives | Travels | Binds |
  |---|---|---|---|
  | `operator` | `~/.night-crew/operators/<id>/preferences/` | every repo that operator touches | only that operator's sessions |
  | `project` | `<repo>/.night-crew/knowledge/preferences/` (git-tracked, today's location) | with the repo | every operator on that repo |
  | `org` | a shared repo pinned from `night-crew.toml` | across repos **and** operators | everyone |

  Identity: derive `<id>` from git `user.email` — already present, stable, no new config.
  `org` is likely **YAGNI** until a second repo pair genuinely shares policy; ship `operator` +
  `project` first and leave the resolution order able to accept a third tier.
- **Merge semantics — three rules, chosen so that ADDING A SCOPE CAN ONLY EVER MAKE THE SYSTEM MORE
  CAUTIOUS, NEVER LESS.** This is the load-bearing safety property; naive merging of preferences is
  how a shared store silently widens someone's autonomy.
  1. **Product facts** (`architecture`, `design`, `ux`) → **most specific wins**: `project` overrides
     `operator` overrides `org`. The gitconfig/editorconfig cascade. The repo is the shared artifact;
     personal taste cannot override what the product *is*.
  2. **Escalation** (today's `constitutional`) → **UNION.** Any applicable rule that says *ask* wins;
     the most cautious applicable policy binds. An escalation can never be lost by adding a scope.
  3. **Delegation** (NF-5 fix 4's inverse semantic) → **INTERSECTION.** The run may act autonomously
     only where **every** applicable scope permits it. Autonomy is granted by unanimity only.
  4. **Process** → `operator`-scoped by default; `project` may **add** discipline, never remove it.
- **`constitutional` must split before it is shared.** It currently mixes personal risk tolerance
  (*"spend above 10% of the run budget"*) with hard invariants (*"pushes and promotions"*). Under
  multi-operator the latter must **not** be operator-waivable. Split into **`safety`** (project/org
  scope, non-waivable, union) and **`escalation`** (operator scope, personal). Without this split,
  segmenting operators silently makes "never push to main" a per-person opinion.
- **Attribution is a hard requirement, not polish.** Add `Operator:` and `Scope:` to the grammar, and
  make `preferences list` print provenance per entry. A merged store without provenance is unreadable
  at the only moment it matters — when a run stopped and nobody can tell whose rule stopped it.
- **The NF-5 ratchet under multi-operator:**
  - A delegated judgment that survives a cycle unreversed ratchets **into the scope where it was
    ratified**, and only into a scope the ratifying operator owns.
  - Ratifying a **process** judgment → that operator's `operator` scope: binds only them, travels with
    them to every repo.
  - Ratifying a **product** judgment → the `project` scope: binds everyone on the repo. **This is
    precisely the desirable consistency the operator named** — one operator's earned autonomy becomes
    the team's, and because `project` scope is git-tracked, the advance arrives as a reviewable commit
    rather than a silent change.
  - **Asymmetric authority, safety-biased: reversal is unilateral, advancement is not.** Any operator
    may reverse a `project`-scoped delegation; a reversal drops it back to *delegated, not policy*.
    Advancement requires the ratifying operator. The ratchet therefore converges under disagreement
    instead of racing.
- **Scope is chosen at capture, by extending NF-5 fix 5's single question.** The policy question and
  the scope question are the same question: *"Standing rule — for this repo, or for all your work?"*
  One prompt, asked once, on the resolved door. `project` answers stay in the repo; `operator` answers
  are written to the home store and are thereby portable — which is the whole cross-repo flow, with no
  sync protocol, no copying, and no drift.
- **Open questions, flagged rather than assumed:** (a) whether `org` earns its existence before a
  second sharing repo exists; (b) how a co-operator's reversal is surfaced to the ratifying operator
  (git history is necessary, probably not sufficient); (c) whether `process` really defaults to
  `operator` — a repo-imposed test-isolation rule may deserve `project` primacy.
- **Scope note:** night-crew **main**. Depends on NF-5 fixes 1 and 4 landing first — a scope axis over
  a store nothing adopts from and nothing reads is inert in three places instead of two.

### NF-7 · "Done"/"deploy" verbs are environment-relative, but the framework has no environments contract — "dev complete" silently conflated *deployed* with *usable*
- **Surfaced:** 2026-08-09 (operator), working the `sync-live-in-dev` leg-3 spike. Operator's
  framing: *"I specified a milestone goal as dev complete, but that is infrastructure-dependent —
  for some projects a dev environment is a local script, for others it is cloud-deployed. Deploying
  and development should be grilled so that, per project, we know exactly what deploy-to-dev means —
  and the same for QA, prod, etc."*
- **Symptom (two payments already made on HQ this cycle):**
  1. **Mid-milestone redefinition.** The "dev complete" close bar had to be *corrected in the
     roadmap itself* (decision 161, morning triage 2026-08-08): from "`task demo:sync` passes"
     to "the sync capability **runs in the operator's persistent dev environment and is usable in
     the app**." The demo cleared the letter of the bar while the capability ran in *no persistent
     environment* — `HQ_SYNC_*` set nowhere, the substrate/relay alive only inside a throwaway
     stack. The bar was ambiguous because "the dev environment" was never pinned.
  2. **A topology fact discovered by driving, not by design.** The leg-3 spike (GREEN, run
     `f20260808232119`) established that "usable in the app" for HQ's dev environment requires the
     substrate's FDW server to point at a *persistent* dev HQ — a reachability/topology fact that
     only surfaced because the spike drove the real integration. An environments contract would
     have *forced* that question up front.
- **Root cause (the missing noun):** night-crew made **`done_when`** first-class — every criterion
  names an observable and the check that proves it. But every observable is evaluated *somewhere*,
  and that "somewhere" is **ambient**. `dev complete`, `ship`, `deploy to prod` are all
  **`when`-clauses with an unstated `where`** — and the `where` is exactly the infrastructure-
  dependent part that varies per project (static PWA → a local `task` against a shared box;
  cloud service → a deployed slot; a library → *no* dev environment at all, "done" = a consumer
  exercises the API). Same words, structurally different meanings, rediscovered per milestone the
  hard way.
- **The distinction that is the whole finding — deploy ≠ usable.** Code can be *deployed* to an
  environment without the *capability* being *usable* there; that gap is precisely where decision
  161 lived. A contract that records only "how to deploy" reproduces the bug. It must separately
  record **(a) the deploy mechanism** (how code gets there) and **(b) the usable-here definition**
  (the falsifiable observable that the capability actually functions there) — and, first-class,
  **(c) persistence** (persistent vs ephemeral), because conflating a throwaway stand-in with the
  operator's persistent environment *is* the decision-161 bug.
- **Proposed artifact — a per-project Environments contract** (durable, e.g.
  `reference/environments.md`), one row per *named place you can be "done" in*, each pinning what
  "deploy" and "complete" are silently relative to:
  1. **Name & purpose** — dev / qa / staging / prod / … (and the deliberately-absent ones recorded
     as considered-nothings, so a future milestone can't assume a QA that doesn't exist).
  2. **Persistence** — persistent | ephemeral. A persistent-env `usable-here` clause may **not** be
     satisfied by an ephemeral stand-in (the decision-161 rule, generalized).
  3. **Topology & substrate** — where it runs; DB/queues/external deps; the concrete hosts/ports.
  4. **Deploy mechanism** — the *exact* command/flow to get code there, and what triggers it.
  5. **Usable-here definition** — the falsifiable observable that the capability functions in this
     environment. *This is the clause a close bar cites.* Distinct from #4.
  6. **Verification** — who/what checks #5 and with what evidence (operator-manual, a script, a
     health endpoint). An operator-manual attestation (like `dev-complete-attestation`) *is* this
     field for that environment.
  7. **Data posture & blast radius** — throwaway / shared-dev / real-prod, and what a mistake here
     can destroy (the `:5433` prod-destruction lesson made structural).
  8. **Reachability** — who/what must reach this environment and by what path (the FDW-pointing
     fact is exactly this — the place `usable-here` quietly fails).
- **Proposed mechanism — an environments grill,** authored in a **dedicated attended pass** (like
  the OKR session is dedicated), **revised on infra change** (a project constant, not per-milestone),
  and **read by every roadmap round** (like OKRs are read). The grill is the question set above,
  asked once per environment; the deploy/topology answers are Eng/infra facts, the usable-here/done
  answers are operator/PM judgments, so the pass is genuinely cross-cutting.
- **How it threads into the ceremonies you already have:**
  - **`done_when` / close-bar authoring cites `⟨env⟩.usable-here` + `⟨env⟩.verification`** instead
    of re-inventing them each milestone. "Dev complete" becomes "meets `dev.usable-here`, verified
    per `dev.verification`" — a fixed, reviewed definition, not a phrase re-litigated at triage.
  - **The spike gate generalizes Spike A.** Spike A already proved "a clean machine reaches the
    substrate unattended." Generalized: each *targeted* environment earns a **"reach `usable-here`
    from nothing"** spike before a milestone leans on it — same B-345 economics, applied to the
    *where* instead of the *what*.
  - **The milestone close reads the contract** so "dev complete / ship / prod" grade against a
    named clause, closing the decision-161 class (a close bar that can be cleared while the
    capability runs nowhere).
- **Enforcement is capability, not detection (added 2026-08-09, operator).** Field #7 (data
  posture & blast radius) must be **enforced**, not merely recorded: scope credentials so the
  agent's environment *cannot reach* a higher-blast-radius environment at all — prod credentials
  hidden from the agent so a test or probe cannot run against prod *regardless of whether it
  carries a destructive statement*. Operator, verbatim: *"the right system is one in which the
  production credentials are hidden from claude so that tests cannot be ran against prod at all
  (regardless if it contains a delete or not)."* A coordinate guard — e.g. the `:5433` port
  refusal in `sync-dev-up.sh` — is a **denylist**, and fails the moment the same cluster is
  reachable by an unlisted coordinate; proven 2026-08-09 by driving the guard, which matches the
  port *string* `5433`, so an aliased/forwarded port sails past. The contract therefore owes an
  **enforcement field** — *what credential does an agent operating here hold, and what is that
  credential scoped to* — not just a prose blast-radius description. **HQ prerequisite (blocks
  this today):** dev and prod are ONE cluster / role / password (BACKLOG T-20 HIGH — shared-cluster
  + `dotenv` 21-live-creds), so there is no separate prod credential to hide yet; the coordinate
  guard is a stopgap for that missing boundary, not the fix. Genuine prod/dev separation (a
  separate cluster, or roles with `REVOKE`) is the precondition for capability-based enforcement.
- **Open questions, flagged rather than assumed:** (a) **granularity** — is one environment a
  single `(topology, data-posture, deploy)` tuple, so "dev-with-fixtures" and "dev-with-shared-data"
  are *two* rows, or one row with variants? (leaning: two rows when the data posture changes the
  `usable-here` check); (b) **ownership split** — does the dedicated pass sit with the OKR/roadmap
  round, or is it its own `/nc-*` step, given it mixes Eng-facts and PM-judgments? (c) whether the
  "reach `usable-here` from nothing" spike should be *mandatory* for every targeted environment or
  only for ones whose deploy mechanism is novel/unproven (leaning: mandatory the first time an
  environment is targeted, then it's proven mechanics like Spike A now is).
- **Scope note:** night-crew **main** — HQ is only where the symptom is persistent (twice this
  cycle). Graduates via the NF-2 loop (design-findings → main → tool-pin advance). *Not* applied to
  HQ unilaterally; when it graduates, HQ's own `reference/environments.md` (dev = `task backend:dev`
  against the Windows-box Postgres over Tailscale, persistent; prod = `task prod:deploy` to Docker
  on the box behind Cloudflare Tunnel; QA/staging = deliberately none) is the first grill's output
  and the natural proof-of-concept.
