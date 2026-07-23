# Session handoff — 20260720c triage follow-up + run-id conformance (2026-07-23)

**For:** the next agent picking up this thread. Read cold; everything you need is here or
cited by path/commit. **Written 2026-07-23.**

**One-line status:** hq triage of `overnight-20260720c` is done and pushed. The run-id
conformance change is committed but **NOT pushed** (one hq loose end). The cross-repo
"numeric matcher → night-crew main → converge this machine" task is **now unblocked** and
waiting on an **operator-only** release step — an agent literally cannot complete it.

> ⚠️ **RE-VERIFY BEFORE ACTING — the night-crew tree is live.** At least **2 other Claude
> sessions** were cwd'd in `/home/jcole/projects/night-crew` while this was written, and its
> state changed twice in the last hour (a whole change, `add-attended-release-runner`, was
> spec'd → implemented → archived during the session). Treat every night-crew fact below as a
> snapshot to re-check, not a given. This is cross-contamination surface #4/#6 from the
> 2026-07-21 audit, live. **Do not stash, remove, or overwrite anything in the night-crew tree
> you did not create** — that clobbers another session's work.

---

## 1. hq — DONE and pushed

Branch `dev`, pushed to `origin/dev` through `771a0da`.

| Commit | What |
|---|---|
| `c2cfc13` | merge of `overnight-20260720c` (F1 trends endpoint, F3 trends tab, F5 gating) |
| `22d85c4` | Go/E2E DB separation (`hq_test_go` / `hq_test_e2e`), proven concurrent-safe |
| `771a0da` | triage record — ledger **T-20** (decisions 34–41), roadmap flips, backlog, HANDOFF flags |

Full triage detail is in this run's `HANDOFF.md` (the `> ✅ TRIAGED 2026-07-21` block) and
ledger `.night-crew/knowledge/ledger.md` §T-20. Gates on the merged tree: `go build`/`vet`
clean, 8 Go pkgs / 0 FAIL, `task test` **528 passed / 6 skipped / 0 failed / 0 flaky**.

## 2. hq — THE ONE LOOSE END: an unpushed commit

```
04704b6 docs(nc): adopt fleet-standard numeric run-id suffix; retire the cycle-letter rule
```

This is committed on local `dev` but **NOT on `origin/dev`** (working tree is clean). It
retires hq's cycle-letter naming and adopts the fleet-standard numeric suffix
(`overnight-YYYYMMDD`, then `-2`/`-3`, `sort -V`) in `.night-crew/knowledge/preferences.md`.

**Action:** push it — `git -C /home/jcole/projects/hq push origin dev`. It's a docs-only
commit; safe. (Left unpushed only because the session moved on to the release investigation.)

**Deliberately NOT done, and correct:** no run-id artifacts were renamed. `overnight-20260720c`
(the sole letter artifact) is baked into 5 pushed commits + a merged branch; renaming recreates
split-identity corruption for zero benefit. It stays an opaque legacy identifier. The rationale
+ the letter→number mapping (documented, not applied) are in the preferences.md "Legacy
artifacts" section. Do not rename it.

## 3. The cross-repo task — numeric matcher onto main + converge this machine

**Why:** the tooling that runs hq's rituals (`~/.claude/skills/nc-*`, symlinked into
`/home/jcole/projects/night-crew-main`) matches run branches with `^overnight-[0-9]+$`, which
skips any `-N` suffix. The numeric matcher `^overnight-[0-9]{8}(-[0-9]+)?$` lives on night-crew
**dev only**; `main` (what runtime machines run) still has the old one. So hq is conformant
going forward, but deployed tooling won't *read* a numeric hq run until the matcher reaches
main. **Benefit is purely forward-looking** — every existing hq branch except `20260720c` is
already visible under both matchers; the matcher upgrade only matters the next time hq runs 2+
times in one day. **There is no urgency.**

**Blocker status: CLEARED as of 2026-07-23** (re-verify). When the operator ran `/nc-release`
it stopped on two gates — a dirty tree (`?? scripts/nc-release-run.sh`) and an unarchived change
(`add-attended-release-runner`, 0/16). Both were the *same* in-flight work, and it has since
**landed and archived** (`60a238c chore(spec): archive add-attended-release-runner`). night-crew
`dev` is now `60a238c`, tree clean, no open changes. `main` still `b9bf632`; **no release has
ever been cut** (v0.1.0 would be the first).

### The steps (verify each before running; commands + directories)

```
# 0. RE-VERIFY readiness (read-only) — from the night-crew dev clone
cd /home/jcole/projects/night-crew
git status --porcelain          # must be clean
ls openspec/changes/ | grep -v archive   # must show no open changes
task nc:release:status          # confirms: not yet promoted, dev ahead of main

# 1. CUT THE FIRST RELEASE — OPERATOR ONLY, in the operator's own terminal
#    add-attended-release-runner replaced the copy-paste block with:
task nc:release:run             # TTY-guarded: verifies tree, shows plan, y/N, tags,
                                # promotion-merges into pinned main, pushes main + tag
#    ⚠️ AN AGENT CANNOT DO THIS. By design the runner reads /dev/tty and refuses without a
#    controlling terminal — an agent shell (including one running /nc-release) has none.
#    The next agent's job here is to confirm readiness and PROMPT THE OPERATOR, not to run it.

# 2. CONVERGE THIS MACHINE — after main moves
cd /home/jcole/projects/night-crew      # (or night-crew-main; it targets the pinned source either way)
task nc:update                  # ff's night-crew-main to origin/main + reinstalls the skills

# 3. VERIFY — from hq
cd /home/jcole/projects/hq
git branch --format='%(refname:short)' | grep -E '^overnight-[0-9]{8}(-[0-9]+)?$' | sort -V
# then re-run /nc-status. NOTE overnight-20260720c stays invisible (letter != numeric) — correct.
```

**A release ships everything, not just the matcher:** step 1 is the *first-ever* night-crew
release and promotes all of `dev` (~200+ commits) to main. It's a milestone-level attended
decision, not a quick fix. night-crew is at a milestone boundary (74 roadmap cards green), so it
may be the right moment — but that's the operator's call, and there's no urgency forcing it.

## 4. Standing flags carried from T-20 (still open)

- **Attended two-device convergence check — ARMED, NOT run** (since the 07-22 `sync.js` change).
- **Prod deploy — NOT done.** hq frontend semver untouched (1.0.3); bump belongs to `/save-project`.
- **Cross-contamination surface #4 OPEN** — dev, prod, test share one Postgres cluster under one
  role/password, separated only by `search_path`. Sharpest isolation item; in hq BACKLOG as HIGH.
- **`stash@{0}`** holds unattributed WIP in a slot shared across worktrees.

## 5. Open decisions / offers not yet taken

- **Push `04704b6`?** (§2) — recommend yes.
- **File a `night-crew update`-from-target-repo backlog item?** The operator asked why tooling
  converge (`task nc:update`) can't run from a target repo like hq. Verified: it can't today
  (hq's Taskfile includes only `backend`; the `night-crew` binary has no update verb; the
  skill-distribution spec defines update as a clone-side `task nc:update`). The operator's
  instinct is right that it *should* be target-repo-invocable — every other operator act is —
  and the fix is a `night-crew update` binary verb (ff-only consume side, so it doesn't cross the
  "no unattended publish" boundary). Offered to file in night-crew BACKLOG (like B-100); not yet
  actioned. Independent of the release; won't collide.
- **B-100** (slate template dispatches OpenSpec mechanics into repos without `openspec/`) was
  filed in night-crew `9649090`, and it IS on `origin/dev`.

## 6. Hard cautions for the next agent

1. **You cannot cut the release.** `task nc:release:run` fails closed without a TTY, by design.
   Prompt the operator; do not attempt it or look for a `--yes`/override — there isn't one.
2. **Do not touch night-crew tree state you didn't create.** ≥2 live sessions share that
   checkout; it changed twice this hour. Re-verify, coordinate, never stash/clobber.
3. **Proportionality:** the whole matcher-to-main effort buys a forward-looking convenience
   only. Do not force the first-ever release or override another session's work to get it.

---

> ✅ **ACTIONED 2026-07-23 (same day, follow-up session):**
> §2 `04704b6` pushed to hq `origin/dev`. §3 step 0 readiness re-verified live: night-crew
> tree clean at `60a238c`, no open changes, `task nc:release:status` confirms no release ever
> cut (dev fully unreleased, main `b9bf632`). §5 the `night-crew update`-from-target-repo item
> is filed as **B-103** (night-crew `7ba1d16`, on `origin/dev`). Remaining: §3 step 1
> (`task nc:release:run`) is operator-TTY-only and still waiting on the operator; steps 2–3
> (converge + verify) blocked behind it. §4 standing flags unchanged.
