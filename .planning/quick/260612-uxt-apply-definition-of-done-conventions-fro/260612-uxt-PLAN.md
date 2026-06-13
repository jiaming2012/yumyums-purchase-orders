---
phase: 260612-uxt
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - CLAUDE.md
  - .planning/PLANNING-TEMPLATES.md
  - .planning/STATE.md
  - APPLYdefinitionofdone.md
autonomous: true
requirements:
  - QUICK-DOD-CONVENTIONS

must_haves:
  truths:
    - "CLAUDE.md contains the Definition of Done section under Conventions"
    - ".planning/PLANNING-TEMPLATES.md exists with Block A, Block B, Block C labeled"
    - ".planning/STATE.md contains the LOCKED/PROBATIONARY/FLUID status legend block under Decisions"
    - "APPLYdefinitionofdone.md staging file is deleted from the repo root"
    - "All three grep verification commands pass with expected counts"
  artifacts:
    - path: "CLAUDE.md"
      provides: "Definition of Done convention block inserted after Bug fix protocol bullet"
      contains: "### Definition of Done"
    - path: ".planning/PLANNING-TEMPLATES.md"
      provides: "Copyable templates for done_when, State Enumeration Table, STATE.md decision rows"
      contains: "Block A"
    - path: ".planning/STATE.md"
      provides: "Status tags legend under Recent decisions affecting current work"
      contains: "Status tags (new entries only"
  key_links:
    - from: "CLAUDE.md ### Definition of Done"
      to: ".planning/PLANNING-TEMPLATES.md"
      via: "first line reference: 'Templates for all blocks below live in `.planning/PLANNING-TEMPLATES.md`'"
      pattern: "Templates for all blocks below live in"
    - from: ".planning/STATE.md status tags legend"
      to: ".planning/PLANNING-TEMPLATES.md Block C"
      via: "inline reference"
      pattern: "PLANNING-TEMPLATES.md.*Block C"
---

<objective>
Apply the Definition of Done conventions from the staging file `APPLYdefinitionofdone.md` into three target files via three idempotent insertions, then delete the staging file.

Purpose: Establish a project-wide Definition of Done convention (done_when blocks, State Enumeration Tables, self-verification ritual, mockup sign-off, verifier subagent gate) so future phases declare completion against observable behavior rather than subjective judgment.

Output:
- `CLAUDE.md` gains a `### Definition of Done` block under `## Conventions`
- `.planning/PLANNING-TEMPLATES.md` is created with Blocks A/B/C
- `.planning/STATE.md` gains a status tags legend under `### Decisions`
- `APPLYdefinitionofdone.md` is deleted
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
</execution_context>

<context>
# Source of exact text to insert (read this first; use exact text, do not paraphrase)
@/Users/jamal/projects/yumyums/hq/APPLYdefinitionofdone.md

# Target file 1 — anchor line is "- **Bug fix protocol (approval phase):**" (currently at line 129)
@/Users/jamal/projects/yumyums/hq/CLAUDE.md

# Target file 3 — anchor line is "Recent decisions affecting current work:" (currently at line 116, under `### Decisions`)
@/Users/jamal/projects/yumyums/hq/.planning/STATE.md

# Project context
@/Users/jamal/projects/yumyums/hq/.planning/PROJECT.md

<critical_notes>
- All three insertions use EXACT text from `APPLYdefinitionofdone.md`. Do not paraphrase, do not adjust wording, do not "improve" the text.
- `.planning/` is gitignored in this repo. The two `.planning/` files MUST be added with `git add -f` when committing. Do NOT commit the change as part of this plan unless the user explicitly asks — execute the file edits only.
- Line numbers in the description are point-in-time; locate anchors by their unique text, not by line number.
- The CLAUDE.md insertion goes BETWEEN the "Bug fix protocol" bullet and the `<!-- GSD:project-start source:PROJECT.md -->` HTML comment marker. There is a blank line between them in the current file — preserve a blank line on each side of the new block.
- The STATE.md insertion goes BETWEEN the `Recent decisions affecting current work:` line and the first existing `- [v2.0 roadmap]:` decision row. Preserve a blank line on each side of the inserted blockquote.
- Idempotency: before inserting, grep for a unique marker from the inserted block. If it already exists, skip that edit and report "already applied" rather than duplicating.
</critical_notes>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Apply the three insertions/creations</name>
  <files>
    CLAUDE.md,
    .planning/PLANNING-TEMPLATES.md,
    .planning/STATE.md
  </files>
  <action>
Read `APPLYdefinitionofdone.md` in full first so the exact text is in context. Then perform the three edits below. For each edit, first run an idempotency check (grep for a unique marker from the block to be inserted) — if the marker is already present, skip that edit and note "already applied".

**Edit 1 — CLAUDE.md insertion:**
- Idempotency check: `grep -n "### Definition of Done" CLAUDE.md` — if it returns a hit, skip.
- Locate the anchor line that begins exactly `- **Bug fix protocol (approval phase):**` in `CLAUDE.md`. Use the Edit tool. The `old_string` should be the entire "Bug fix protocol" bullet line followed by the blank line followed by the `<!-- GSD:project-start source:PROJECT.md -->` marker. The `new_string` should be the same content but with the Definition of Done block inserted between the bullet and the GSD marker. The inserted block is exactly the markdown fenced as `Change 1` in the source doc (starting with `### Definition of Done` and ending with the `- **Verifier subagent gate...` paragraph). Preserve one blank line above `### Definition of Done` and one blank line below the last bullet before the `<!-- GSD:project-start ... -->` marker.
- Use the EXACT text from the source doc between the ```markdown fences in "Change 1" — copy verbatim, no edits.

**Edit 2 — Create `.planning/PLANNING-TEMPLATES.md`:**
- Idempotency check: `test -f .planning/PLANNING-TEMPLATES.md` — if it exists with `Block A` already in it, skip.
- Use the Write tool to create `.planning/PLANNING-TEMPLATES.md` with the EXACT contents of "Change 2" in the source doc (the markdown inside the outer ```` ```markdown ```` fence, starting with `# Planning Templates` and ending with `... the convention applies from the next entry forward.`). Note that the source uses a four-backtick fence to wrap a block that itself contains three-backtick fences — when writing the file, include only the inner content, not the four-backtick wrapper.

**Edit 3 — STATE.md insertion:**
- Idempotency check: `grep -n "Status tags (new entries only" .planning/STATE.md` — if it returns a hit, skip.
- Locate the anchor line `Recent decisions affecting current work:` under the `### Decisions` heading in `.planning/STATE.md`. Use the Edit tool with `old_string` = the anchor line plus the blank line plus the first existing decision row (`- [v2.0 roadmap]: httpOnly, Secure, SameSite=Strict cookies...`). `new_string` should insert the blockquote from "Change 3" of the source doc between them. Preserve a blank line above and below the blockquote. Use the EXACT text from the source doc between the ```markdown fences in "Change 3".
  </action>
  <verify>
    <automated>cd /Users/jamal/projects/yumyums/hq &amp;&amp; A=$(grep -cE "done_when|State Enumeration|Verifier subagent|Mockup sign-off|Self-verification" CLAUDE.md) &amp;&amp; B=$(grep -cE "Block A|Block B|Block C" .planning/PLANNING-TEMPLATES.md) &amp;&amp; C=$(grep -c "LOCKED" .planning/STATE.md) &amp;&amp; echo "CLAUDE.md=$A (expect 6) PLANNING-TEMPLATES.md=$B (expect 3) STATE.md=$C (expect >=2)" &amp;&amp; test "$A" -ge 6 &amp;&amp; test "$B" -ge 3 &amp;&amp; test "$C" -ge 2</automated>
  </verify>
  <done>
- `grep -n "### Definition of Done" CLAUDE.md` returns exactly one hit
- `.planning/PLANNING-TEMPLATES.md` exists and contains `## Block A`, `## Block B`, `## Block C` headings
- `grep -n "Status tags (new entries only" .planning/STATE.md` returns exactly one hit
- All three verification grep counts from the source doc meet expectations (CLAUDE.md: 6, PLANNING-TEMPLATES.md: 3, STATE.md: >= 2)
- No surrounding content in CLAUDE.md or STATE.md was reformatted or lost (existing bullets immediately before/after the insertion are byte-identical to before)
  </done>
</task>

<task type="auto">
  <name>Task 2: Delete the staging file</name>
  <files>APPLYdefinitionofdone.md</files>
  <action>
Delete `/Users/jamal/projects/yumyums/hq/APPLYdefinitionofdone.md` only after Task 1's verification passes. Use `rm /Users/jamal/projects/yumyums/hq/APPLYdefinitionofdone.md`. Do not use `git rm` — the staging file may not be tracked.

If Task 1 reported "already applied" for any of the three edits (idempotent re-run), still delete the staging file at the end.
  </action>
  <verify>
    <automated>test ! -f /Users/jamal/projects/yumyums/hq/APPLYdefinitionofdone.md &amp;&amp; echo "staging file deleted"</automated>
  </verify>
  <done>
- `APPLYdefinitionofdone.md` no longer exists at the repo root
- `ls /Users/jamal/projects/yumyums/hq/APPLYdefinitionofdone.md` returns "No such file or directory"
  </done>
</task>

</tasks>

<verification>
End-to-end check that mirrors the source doc's "Verify after applying" block:

```bash
cd /Users/jamal/projects/yumyums/hq
grep -c "done_when\|State Enumeration\|Verifier subagent\|Mockup sign-off\|Self-verification" CLAUDE.md   # expect 6
grep -c "Block A\|Block B\|Block C" .planning/PLANNING-TEMPLATES.md                                       # expect 3
grep -c "LOCKED" .planning/STATE.md                                                                       # expect >=2
test ! -f APPLYdefinitionofdone.md && echo "staging file removed"
```

Additionally:
- The `<!-- GSD:project-start source:PROJECT.md -->` marker in CLAUDE.md is still present and on its own line after the inserted block.
- The first existing decision row in STATE.md (`- [v2.0 roadmap]: httpOnly, Secure...`) is still present, unchanged, after the inserted blockquote.
</verification>

<success_criteria>
- Three edits applied with exact text from the source doc, or skipped via idempotency check with explicit "already applied" report.
- All four verification commands above pass.
- No collateral edits to CLAUDE.md or STATE.md beyond the inserted blocks.
- `APPLYdefinitionofdone.md` deleted.
- No commit created by this plan unless the user explicitly requests one (`.planning/` is gitignored — committing requires `git add -f` and is out of scope for this apply step).
</success_criteria>

<output>
After completion, create `.planning/quick/260612-uxt-apply-definition-of-done-conventions-fro/260612-uxt-SUMMARY.md` with:
- Which of the three edits were applied vs skipped (idempotent)
- The four verification command results
- Confirmation the staging file was deleted
- Note for the user: `.planning/PLANNING-TEMPLATES.md` and `.planning/STATE.md` are gitignored — use `git add -f` if you want to commit them
</output>
