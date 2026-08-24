# Planning Templates

Copyable blocks for the Definition of Done conventions (see CLAUDE.md → Conventions → Definition of Done).
Three blocks: `done_when:` for PLAN.md, the State Enumeration Table for UI-SPEC.md, and the decision-row format for STATE.md.

## Block A — `done_when:` (PLAN.md, paired with `must_haves:`)

Every criterion names the observable behavior AND the check that proves it. No subjective
language — "looks good," "feels right," "polished," "clean," and "nice" are banned.

```yaml
done_when:
  # Each row: "<observable behavior> — <how to check it>"
  - "GET /api/v1/foo returns 503 when token unset — curl + assert status"
  - "Empty state renders 'No X yet' when DB returns [] — load page with empty fixture, screenshot"
  - "Sliders snap to 5% increments — Playwright drag + assert step, screenshot"
  - "Touch targets ≥44px on .pill and .btn — Playwright getBoundingClientRect assertion"
  - "Dark mode contrast passes on warning pill — Playwright with prefers-color-scheme: dark, screenshot"
```

UI-SPEC.md files also carry a `done_when:` block for criteria that are visual rather than
functional (spacing, dark mode, truncation behavior).

## Block B — State Enumeration Table (UI-SPEC.md, new section after Component Inventory)

The four base states are mandatory. **At least 2 phase-specific edge rows are required** —
the table is incomplete without them. The rows below are a template, not a constant:
replace triggers and contracts with the phase's actual behavior.

| State | Trigger | Visual contract |
|-------|---------|-----------------|
| empty | no data returned | icon + "Add your first X" CTA |
| loading | fetch in flight | skeleton rows; no spinner under 500ms |
| error | fetch failed | red banner + "Couldn't load" + retry button |
| success | data present | list as designed in Component Inventory |
| edge: long content | name > 40 chars | truncate with ellipsis, full text on tap |
| edge: offline | `navigator.onLine === false` | yellow banner "Working offline" |
| edge: conflict | 409 from API | inline error with conflict-specific copy |

Each row maps 1:1 to a screenshot in `tests/states-<phase>.spec.js` during self-verification.

## Block C — STATE.md decision-row format (new entries only)

```
- [Phase NN] [LOCKED|PROBATIONARY|FLUID] [YYYY-MM-DD]: <decision text>
```

Status legend:

- **LOCKED** — frozen architecture; changing requires a new ADR-style row that supersedes this one.
- **PROBATIONARY** — recently changed; watch for regression. Auto-promotes to LOCKED after 2 phases pass without re-litigation.
- **FLUID** — current best guess; cheap to revise.

Existing untagged rows are treated as implicitly LOCKED unless contradicted. Do not bulk
retro-tag old rows — the convention applies from the next entry forward.
