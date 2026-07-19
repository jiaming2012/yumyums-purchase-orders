---
name: "save-project"
description: "Snapshot the current session before deploy: bump semver versions for the changed side(s), commit with a structured message, and push. Pairs with `task prod:deploy` to keep dev and prod versions diffable via `/api/v1/health`."
argument-hint: "Optional: any additional notes or context about the current changes"
user-invocable: true
disable-model-invocation: false
---

## User Input

```text
$ARGUMENTS
```

Consider any user-provided notes before proceeding. They may clarify scope or flag known issues that should be woven into the commit message.

---

## Overview

This skill runs two sequential phases. Each phase **gates on explicit user confirmation** before the next begins. The user may stop after any phase.

| Phase | Role | Output |
|-------|------|--------|
| 1 | QA Assistant | New / updated Playwright test files (optional) |
| 2 | Git Commit   | Version bump + signed commit + push |

> Spec / plan updates belong to GSD — use `/gsd:quick`, `/gsd:plan-phase`, or `/gsd:execute-phase` for that. This skill assumes specs and plans already exist (or aren't needed for this change).

---

## Phase 1 — QA Assistant: Test Generation (optional)

**GATE**: Display the following and **wait for explicit confirmation**:

> Ready to review test coverage before bumping versions? (yes / skip)

If the user says "skip", jump to Phase 2.

### 1.1 Gather changes

```bash
git diff HEAD --stat
git status --short
```

Filter out `dist/`, `node_modules/`, `.planning/`, `playwright-report/`, `test-results/`.

### 1.2 Identify coverage gaps

For each functional change, decide whether existing tests cover it. Pay special attention to **the persistence rule** documented in `CLAUDE.md`:

> Every user-entered value must follow `autoSaveField` → `DRAFT_RESPONSES` → `hydrateFieldState`. Every new field type or data entry feature MUST have a back-and-reopen test in `tests/persistence.spec.js`.

If a new user-entered value type or field was added without a corresponding back-and-reopen test, **flag it as a blocking gap** and offer to write the test before Phase 2.

### 1.3 Write tests (if needed)

Place tests in the matching file by domain:
- `tests/workflows.spec.js` — workflow templates, checklists, approvals
- `tests/inventory.spec.js` — purchases, stock, recipes (cross-tab)
- `tests/recipes.spec.js` — recipe BOM / sliders specifically
- `tests/onboarding.spec.js` — invite, password set, role gates
- `tests/users.spec.js` — user mgmt, role pills, grants
- `tests/persistence.spec.js` — back-and-reopen for any new field type
- `tests/sync.spec.js` — cross-device WebSocket sync
- `tests/states-<phase>.spec.js` — State Enumeration Table coverage from a UI-SPEC.md

Conventions (existing yumyums style — do not import Vaada conventions like `seedAuth` or `getByRole`):
- CSS / `data-action` selectors are the existing norm — match the surrounding file's style
- Login helper is `await login(page)` (per `tests/persistence.spec.js`)
- API helper is `apiCall(page, method, path, body)` for `/api/v1/workflow/*`
- Unique test data via `Date.now()` suffixes — the test DB is not reset between tests

Run only the new test(s) to confirm they pass (not the full suite):

```bash
npx playwright test tests/<file>.spec.js -g "<test name>"
```

### 1.4 Report

List each test file touched with positive/negative counts and any `// TODO` items needing follow-up.

---

## Phase 2 — Git Commit + Version Bump

**GATE**: Display the following and **wait for explicit confirmation**:

> Phase 1 complete. Ready to bump versions and commit? (yes / skip)

If the user says "skip", end the skill here.

### 2.0 Bump semantic versions

Read the current values:
- `backend/internal/version/version.go` — `Backend` and `Frontend` constants (**authoritative**)
- `package.json` — `"version"` field (**must mirror** `Frontend` constant)

If `package.json.version` and `version.go` `Frontend` disagree, flag it and use the `version.go` value as the authority.

**Detect which side changed** by inspecting the diff:
- Any file under `backend/` (excluding `backend/Taskfile.yml`, `backend/.env`) → **backend changed**
- Any frontend file at repo root (`*.html`, `*.js` except `build-sw.js`, `manifest.json`, `icons/`, `lib/`) → **frontend changed**
- Both can be true — bump them independently

Spec/test/docs-only changes (only under `tests/`, `docs/`, `.planning/`, `*.md`) count as **patch** on whichever side owns the feature, or skip the bump entirely if purely meta.

Apply these rules **independently** for each side that changed:

| Level | Frontend triggers | Backend triggers |
|-------|-------------------|------------------|
| **major** | Removed route/page; auth contract change visible to user; breaking shape of persisted state | Removed endpoint; breaking change to an API contract; breaking DB migration (column drop, type narrowing) |
| **minor** | New tool page in the launcher grid; new tab in an existing tool; new user-visible feature | New handler / endpoint; new internal package surfaced via the API; new migration (additive) |
| **patch** | Bug fix; copy / style; test-only; refactor | Bug fix; test-only; refactor; dependency bump; non-functional migration (e.g. index) |

Default to **patch** when uncertain.

**Apply the bumps**:
- If frontend changed: update **both** the `Frontend` constant in `backend/internal/version/version.go` AND `"version"` in `package.json` to the same new value.
- If backend changed: update the `Backend` constant in `backend/internal/version/version.go`.

> ⚠️ `package.json` and the `Frontend` constant in `version.go` must always match. Never bump one without the other. Do not run `npm version` — always update both files in the same commit.

Display the decisions inline — no confirmation needed:

```
Backend:  0.1.0 → 0.1.1 (patch — retry-parse bug fix)
Frontend: 1.0.0 → 1.1.0 (minor — scheduling tile added to launcher)
```

> **Note**: The versions baked into the running backend binary reflect when it was last built and deployed. Bumping here and committing does not update production until `task prod:deploy` runs. This is intentional: `task version` will show local-source / dev / prod side-by-side after deploy, surfacing any drift.

### 2.1 Rebuild service worker (mandatory before any UI deploy)

```bash
task sw
```

This regenerates `sw.js` with content-hashed precache and writes `version.json` from `package.json`. Without this step, returning users see cached stale assets. See `feedback_sw_cache_bump.md` in memory for the rationale.

### 2.2 Draft commit message

Refresh state:

```bash
git diff HEAD --stat
git status --short
```

Draft a message in this format:

```
<type>(<scope>): <imperative summary — 72 chars max>

## What changed
- [path or component]: [one sentence on the change]
- [repeat for each meaningful change; omit node_modules/, .planning/, test-results/]

## Why
[1–3 sentences on the user problem solved or product goal advanced —
write from the product perspective, not the engineering perspective]

## Version bumps
- Backend:  0.1.0 → 0.1.1 (patch — <reason>)
- Frontend: 1.0.0 → 1.1.0 (minor — <reason>)
(omit either line if that side did not change)

## Test coverage
- tests/<file>.spec.js: new | updated — positive: N, negative: N
(omit section if no tests were touched)

## Changed files
backend/  [list]
root/     [list of frontend files]
tests/    [list]
```

Use these `<type>` values:
- `feat` — new user-visible capability (typically a minor bump)
- `update` — enhancement to an existing feature
- `fix` — bug or regression fix (typically a patch bump)
- `test` — test-only change
- `refactor` — internal restructure with no behaviour change
- `chore` — build / tooling / dependency change
- `docs` — README, CLAUDE.md, or other docs only

Use a concise `<scope>` that names the primary area changed (e.g., `inventory`, `recipes`, `auth`, `deploy`). For multi-area, use `multi`.

### 2.3 Display for approval — STOP and wait

Display the full drafted commit message and ask:

> **Does this commit message look good?**
> Reply **"yes"** to commit, or tell me what to change.

If the user requests changes, revise and re-display. **Do not commit without explicit "yes".**

### 2.4 Stage and commit

Once the user approves:

1. **Stage** the version files plus the affected source/tests:
   ```bash
   git add backend/internal/version/version.go package.json sw.js
   git add backend/ tests/ <changed frontend files>
   ```
   If other directories have changes (e.g. `.claude/`, `docs/`), list them and ask before staging.

2. **Commit** with the approved message using a heredoc to preserve formatting. End the message with:
   ```
   Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
   ```

3. **Verify**: run `git status` to confirm no unexpected staged/unstaged changes.

4. **Push** to the remote:
   ```bash
   git push
   ```
   Report the result.

### 2.5 Offer to deploy

Display:

```
✓ Committed and pushed.

To deploy to prod (Windows box via Tailscale → Docker rebuild + restart):
  task prod:deploy

After deploy, verify with:
  task version          # diff local / dev / prod
  task health:prod      # confirm new version is live
```

Do **not** run `task prod:deploy` automatically — production is real and the user owns that decision.

---

## Guiding Principles

**Phase 1 — test quality rules**
- Tests document behaviour, not implementation — they should still pass after a refactor
- A failing negative test should point directly to a regression, not a test setup issue
- The persistence rule is non-negotiable — every new user-entered value type needs a back-and-reopen test

**Phase 2 — commit quality rules**
- The "Why" section should be readable by someone who never opens the diff
- `package.json` `"version"` and the `Frontend` constant in `version.go` move together — always
- Do not stage build artifacts (`backend/server`, `version.json`, `backend/bin/`), credentials, or generated files unless explicitly asked
- One commit per `/save-project` invocation — do not split or amend unless the user asks
