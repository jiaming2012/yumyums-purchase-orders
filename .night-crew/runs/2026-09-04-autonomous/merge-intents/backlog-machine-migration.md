# Merge intent — Card 3 · `backlog-machine-migration` (run 20260904)

Branch: `wo-backlog-machine-migration` (cut from `overnight-20260904` @ 4617b8a, after
Cards 1–2 merged). Card authority: slate-20260904 Card 3. Closes **B-02, B-168, B-12,
B-133** (Q-KR3's producer, carried two cycles). DOCS/PLANNING card — no app code, no
database, no Go/Playwright/build-sw legs. The gate IS the night-crew CLI verbs' exit
codes (`backlog check`, `backlog list`) plus the whole-document content-preservation
proof recorded below.

## Shared files touched

- `.night-crew/knowledge/BACKLOG.md` — the card's subject, **exclusive footprint** per
  the slate. Reshaped to canonical `- **B-NN · Title** — desc · origin · status ·
  lead: …` form, content-preserved (proof below). Iterated on a SCRATCH COPY via
  `night-crew backlog check --file <copy>` (spike-proven instrument); the real
  document's sha moves only at the landing commit.
- `.night-crew/knowledge/COMMANDS.md` — the triage §4.5 gate arming: a ritual step +
  check invocation added to the `/nc-morning-triage` entry (step 6), where a triage in
  this hand-run repo actually reads. ⚠️ The slate names this file **cross-track with
  Card 4** (different ritual moments — Card 3 arms the triage gate, Card 4 adds the
  closeout scorecard step). On conflict: keep BOTH additions; they touch different
  ritual moments and do not overlap semantically. Log any merge in the conflict log.
- `.night-crew/knowledge/roadmap.md` — one-line status flip of the
  `backlog-machine-migration` card (PLANNED → DRAFTING), matching Cards 1–2's flip
  convention. No other roadmap line moves.
- `.night-crew/runs/2026-09-04-autonomous/merge-intents/backlog-machine-migration.md`
  — this file (amended as evidence lands).
- `.night-crew/runs/2026-09-04-autonomous/card3-*.log` — committed whole logs, each
  ending in its `EXIT=` line: red baseline check + list, final green check, list-count
  comparison, preservation-proof run.
- `.night-crew/runs/2026-09-04-autonomous/card3-reshape.py` and
  `card3-preservation-proof.py` — the mechanical reshape tool (every per-entry edit
  explicit, reviewable) and the skeptic-rerunnable preservation proof (committed per
  the card: the G6 reviewer must be able to re-run the proof from this intent alone).
- `night-crew.toml` — **nothing here** (planning docs match no `[e2e.seams]` key; no
  spec subset is owed by this footprint).
- App code, `sw.js`, `package.json`, `backend/`, `tests/`, `supabase/` — **nothing
  here** (untouched).
- `.night-crew/knowledge/spikes/**`, `.night-crew/spikes/**` — **nothing here**
  (read-only inputs: the spike ledger + extraction are this card's build facts).

## What must survive any merge

- The reshaped `.night-crew/knowledge/BACKLOG.md` **whole** — the reshape is a single
  consistent renumbering (new handles B-350+ assigned in document order); merging half
  of it re-introduces the grammar red AND can split a handle assignment. If a conflict
  forces a choice, take this branch's BACKLOG.md wholesale and re-apply the other
  side's edit as a NEW canonical entry on top (then re-run
  `night-crew backlog check --repo .` — it must exit 0).
- The triage §4.5 gate step in `COMMANDS.md` (keep alongside Card 4's closeout step —
  both, not either).
- The roadmap status flip; the `card3-*.log` evidence files; the two committed card3
  scripts; this intent.

## What is safe to drop

- Nothing in this branch is scratch. The scratch copy of BACKLOG.md lives only in the
  session scratchpad and is not committed. Run-dir logs are append-only evidence — in
  a conflict keep both sides.

## Red-first

Docs-card RF form: the checker's own red IS the before-state, captured from the
worktree root BEFORE any reshape, document sha-pinned.

- **RED (check):** `night-crew backlog check --repo .` → **EXIT=1**, final line
  verbatim: `night-crew: backlog invalid: 297 issue(s) across 207 entries`
  (`card3-red-01-check-baseline.log`; 297 issue lines counted in the log).
  Document sha256 (pre-reshape):
  `342c96cdc429adf764edbba413dcba23e9c60118507b0a4ec44eaa61e31e5f0b` — byte-identical
  to the sha the 2026-09-03 spike pinned, so the spike's baseline carries unchanged.
- **RED (comparator left side):** `night-crew backlog list --repo .` → **EXIT=0**,
  **208** output lines = 207 entry lines + 1 view-time footer line ("priority derived
  at view time …"), of which **144** are handle-bearing `B-NN [status]` lines
  (`card3-red-02-list-baseline.log`). Baseline comparator reads 208 (list, incl.
  footer) vs 207 (the checker's own "across M entries" parse) — per the spike, the
  done_when anchors on the checker's parse, never a hand grep.
- **Issue census by defect class** (parsed from the red log, sums to 297):
  63 missing handle · 63 missing one-line description · 91 unrecognized status ·
  75 missing plain-language lead · 2 missing origin · 3 document-level issues
  (2 × "carries an origin but does not read as an entry", 1 × duplicate handle
  B-77 ×2).

## Content-preservation proof

**Method (Q-KR3's mechanical form, spike-proven shape scaled to the whole document):**
token-multiset **containment** over the WHOLE document. Tokenize both versions with
`re.findall(r"[A-Za-z0-9]+", text)` into `collections.Counter` multisets; the proof
holds iff `(before - after)` is EMPTY — every alphanumeric token occurrence present
before is present after (words may move or be added — handles, canonical status heads,
leads — never vanish). Runner: `.night-crew/runs/2026-09-04-autonomous/card3-preservation-proof.py`.

**Refs compared (proof-integrity rule 6):** BEFORE = `.night-crew/knowledge/BACKLOG.md`
at the red-baseline commit (the first commit of this branch, which precedes every
reshape commit — resolved by the proof script via
`git show <red-baseline-sha>:.night-crew/knowledge/BACKLOG.md`), NOT a scratch copy.
AFTER = the landed working-tree file at the landing commit. The exact SHAs are printed
inside `card3-green-04-preservation-proof.log`.

**Result (landed):**
- BEFORE = `426e51a8f163f26e42ec9f3480d21a46f5be083f:.night-crew/knowledge/BACKLOG.md`
  (the red-baseline commit; content sha256 `342c96cd…e31e5f0b`, the spike-pinned pre-doc):
  **58 307 token occurrences / 6 070 distinct**
- AFTER = the landed `.night-crew/knowledge/BACKLOG.md`
  (content sha256 `ede3a29ae8927e8c436d935db29a63130106a42d33291c0943196b86998b898e`):
  **59 661 token occurrences / 6 138 distinct**
- `before − after` (lost token occurrences): **0**
- Verdict: **GREEN — containment holds** (`card3-green-04-preservation-proof.log`,
  EXIT=0 inside the log; re-run:
  `python3 .night-crew/runs/2026-09-04-autonomous/card3-preservation-proof.py`).
  Growth (+1 354 occurrences) is the additive canonical material: 65 new handles,
  canonical status heads, and the written/folded leads.

**done_when state at landing:**
1. `night-crew backlog check --repo .` → **EXIT=0**, `backlog: valid — 209 entries`
   (`card3-green-01-check-final.log`).
2. `night-crew backlog list --repo .` → EXIT=0, **209 entry-shaped lines** + 1
   view-time footer == the checker's own 209 parse — **MATCH**
   (`card3-green-02-list-compare.log`). Baseline 207→209 explained in that log:
   +B-350 (the machine-invisible PRIORITY bullet), +B-387 (the machine-invisible
   Rejection-feedback bullet), +B-54 (nested-bold title fixed), −1 (B-77
   duplicate folded into the original B-77's lead, content moved not deleted).
3. Preservation proof green (above).

## Commit plan (stated per mechanics rule 2)

1. This merge-intent + the red baseline logs (FIRST commit, before any reshape).
2. The reshape itself as ONE landing commit (BACKLOG.md + card3-reshape.py +
   green/comparison/proof logs + this intent's Result filled) — one commit rather
   than per-class because the handle renumbering is a single document-order sequence;
   splitting it would leave intermediate commits with dangling handle gaps.
3. Triage §4.5 gate arming (COMMANDS.md) + roadmap flip.

All commits carry trailer `Night-Crew-Run: 20260904` and name
`backlog-machine-migration` in the body.
