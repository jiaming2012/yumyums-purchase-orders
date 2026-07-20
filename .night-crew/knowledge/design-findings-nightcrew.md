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
- **Proposed fix (night-crew main):** add a guaranteed producer for the design-findings sweep at
  a cycle boundary. Candidates: a step in `/nc-okr-session` (cycle start reviews last cycle's
  findings) or a closeout step in `/nc-morning-triage` / a new cycle-complete skill. Make this
  standing file (`knowledge/design-findings-nightcrew.md`) the canonical target, scaffolded by
  `night-crew init` alongside the other knowledge files.

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
