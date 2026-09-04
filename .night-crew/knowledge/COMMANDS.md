# Night-crew commands — cheatsheet

The `/nc-*` slash commands and the order they run in. Also available as the global
`/nc-help` skill (invoke it in any repo). Type `/nc` in the prompt to autocomplete
all commands with their descriptions.

## The cycle

One planning funnel (attended, evening) feeding one execution loop (unattended
overnight → morning).

### PLAN — attended, evening

1. **`/nc-okr-session`** — Set the cycle's OKRs with the operator.
   → `.night-crew/knowledge/okrs.md`. Run **once** per planning cycle. Everything
   below traces back to a key result here. (Missing/stale OKRs? Start here.)

2. **`/nc-pm-session`** — Evening PM session: read OKRs + roadmap + ledger + the
   run's intake + Operator Brief, draft a PRD, and route each intake item through
   three doors (fold into the PRD / graduate to backlog / drop with a note).
   Needs `okrs.md` to exist first.

3. **`/nc-pm-grill-back`** — Grill the drafted PRD's gray areas — each driven to
   *resolved / delegated / queued*. Emits the sign-off Assumptions. Rides on
   `/nc-pm-session`.

4. **`/nc-slate-plan [activity] [--night 8h]`** — Turn ready roadmap cards into
   **tonight's signed slate**: WO-sized cards, module footprints, parallel tracks,
   per-card time estimates vs the night budget, batch sign-off, and a paste-ready
   launch prompt.

### EXECUTE — unattended overnight → morning

5. **(launch prompt)** — Paste the slate's launch prompt into a **fresh session**.
   Runs autonomously on branch `overnight-YYYYMMDD`. Never push, never touch `main`.

6. **`/nc-morning-triage`** — Next morning: review the run branch, merge to `dev`,
   resolve `DECISIONS-NEEDED` forks with the operator, record resolutions to
   DESIGN §15x, flip roadmap cards + HANDOFF flags. Then loop back to step 2 (or
   step 1 at a milestone boundary).

   **§4.5 backlog gate (ARMED as of run 20260904 — this step is mandatory, and it
   passes now).** After any triage edit to `.night-crew/knowledge/BACKLOG.md` —
   and before the triage merge commit — run:

   ```
   night-crew backlog check --repo .
   ```

   It must **exit 0**. The document was migrated to full canonical form by
   `backlog-machine-migration` (run 20260904: `backlog: valid — 209 entries`,
   content-preservation proven in that card's merge-intent), so a red here is a
   real grammar defect **introduced by the edit just made** — fix that entry to
   the canonical shape
   `- **B-NN · Title** — one-line description · origin · status · lead: plain line`
   (statuses: `new` · `promoted → <card>` · `landed → <artifact>` ·
   `done — <what shipped>` · `dropped — reason`; new handles above the current
   max, **never reused**). Never skip the check, never narrate around a red —
   a skipped ritual step reads exactly like a clean one (B-133). Ledger decision
   141's "the verb is advisory for this file" rider is **retired** by the
   migration: the gate is binding again. Record the check's exit line in the
   triage notes. Prefer `night-crew backlog add` for new entries — it emits the
   canonical shape and assigns the next handle itself.

## Flags & args

**Only `/nc-slate-plan` takes any.** The other four are invoked bare — they read
state from `.night-crew/knowledge/` and drive the operator interactively.

| Command | Args / flags |
|---------|--------------|
| `/nc-slate-plan` | `[<activity name(s)>]` — e.g. `"Gate and verify work"`; omit → next activity inferred from roadmap order |
| | `--night <duration>` — size slate to the night (e.g. `--night 8.5h`); draws the cut line, cards past it become budget-gated stretch, and lets an inferred slate spill into later activities |
| | `--hours <duration>` — alias for `--night` |
| | `--help`, `-h` — show the full usage block and stop |
| `/nc-okr-session` | none (bare). Shell helper: `night-crew okr validate --repo .` |
| `/nc-pm-session` | none (bare) |
| `/nc-pm-grill-back` | none (bare) |
| `/nc-morning-triage` | none (bare) |

## Quick answers

- **Starting a new cycle?** → `/nc-okr-session`
- **OKRs set, planning tonight?** → `/nc-pm-session` → `/nc-pm-grill-back` → `/nc-slate-plan`
- **Just want tonight's slate?** → `/nc-slate-plan --night 8h`
- **Run finished overnight?** → `/nc-morning-triage`
- **Forgot a command's params?** → `/nc-slate-plan --help` (only slate-plan has params)
