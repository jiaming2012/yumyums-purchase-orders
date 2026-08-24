# Merge intent — A2 · `shipped-bug-sweep`

Run `20260807` · branch `card/a2-shipped-bug-sweep` · closes **B-89** and **B-132**.

---

## Scope

(a) **B-89** — `sync-rxdb/bootstrap.js`'s `cachedGrantSlugs()` `Array.isArray`-gates the raw
`localStorage['hq_apps']` parse. Production (`index.html:228-237`) has written the identity-stamped
envelope `{uid, apps}` since decision 112 (T-30) — never a bare array — so the gate rejects every
real client and `cachedGrantSlugs()` silently returns `[]` always. Fix: read the envelope the way
`index.html:234-236`'s `readCachedApps()` does, including the `uid` check against the identity
token (`hq-identity` cache / `/__hq_identity`, same store `index.html:150-179` reads/writes —
CacheStorage is origin-scoped, reachable from `workflows.html` too, not launcher-only).

(b) **B-132** — `workflows.html`'s `fireworks()` animate loop guards `p.life<=0` *before* decrementing
`p.life`, then draws `ctx.arc(..., p.size*p.life, ...)` *after* the decrement with no floor — a
particle whose life crosses zero mid-frame draws with a negative radius, `ctx.arc` throws, and the
uncaught exception in the `requestAnimationFrame` callback halts the loop before it ever reaches
`canvas.remove()`. The confetti canvas (fixed, full-viewport, `z-index:10000`) is orphaned on top
of the page indefinitely. Fix: clamp the radius (`Math.max(0, p.size*p.life)`), leaving the guard
order and every other particle-loop behavior unchanged.

---

## Shared files touched

Footprint is `sync-rxdb/bootstrap.js`, `tests/sync-rxdb-client.spec.js`, `workflows.html` — plus
the two the card names explicitly:

| File | Why |
|---|---|
| `.night-crew/knowledge/roadmap.md` | Card status flip for `shipped-bug-sweep`, per that file's existing convention. |
| `sw.js` (+ `version.json` if content-hash changes cascade) | `workflows.html` and `sync-rxdb/bootstrap.js` are both precached; regenerated via `node build-sw.js` AFTER the content commits, committed as its own commit. No version bump (that's `/save-project`'s job, not this card's). |
| `.night-crew/knowledge/BACKLOG.md` | B-89 and B-132 entries marked closed, citing the closing commit. |

Nothing else. No backend Go file, no other HTML tool page, no `night-crew.toml` entry, no
`docker-compose*.yml`, no `Taskfile.yml`.

**Declared gap, not a deviation:** `night-crew.toml`'s footprint table has no row for
`sync-rxdb/bootstrap.js` or `tests/sync-rxdb-client.spec.js` (only a stale comment references them
around line 66). This card does not add one — out of the declared footprint — but the full
Playwright suite is run as this card's own final gate regardless, so no coverage gap results from
it tonight. Left as-is for a maintenance card to consider.

---

## What must survive any merge

- **`cachedGrantSlugs()`'s async signature and its one call site's `await`.** The fix makes the
  function read `caches` (async CacheStorage), so it and `const built = createHQSyncClient(...)`
  both change from sync to async. Any other card touching `sync-rxdb/bootstrap.js`'s module-body
  construction block must keep the `await`.
- **The uid-mismatch behavior: silent empty, not a thrown error.** Mirrors `index.html:234-236`
  exactly (`env.uid!==deviceId` → treated as absent, no cached grants used) — a product decision
  already shipped in `index.html`, not one this card made. See PARK analysis below.
- **The `Math.max(0, p.size*p.life)` clamp in `fireworks()`'s `animate()`.** Any later rewrite of
  the particle loop must keep the radius non-negative or B-132 regresses.
- **The regression test in `tests/workflows.spec.js`** (`B-132 — fireworks confetti canvas` describe
  block) and the fixture rewrite in `tests/sync-rxdb-client.spec.js` at the former line 1385 (now
  planting the real `{uid, apps}` envelope + matching identity-cache token, not a bare array).

## What is safe to drop

- The prose/comments explaining *why* (informative, not load-bearing).
- The screenshot artifacts under `a2-logs/` — evidence only, regenerable by rerunning the tests.
- The `night-crew.toml` footprint-gap note above — observation, not a code change.

## Nothing here

- No backend Go change. No API contract change. No `docker-compose*.yml`, no `Taskfile.yml`.
- No version bump (`package.json` / `backend/internal/version/version.go` untouched, per the
  card's explicit instruction — that is `/save-project`'s job).
- `main`, `dev`, `card/a3-rls-fixture-own`, and the run worktree at
  `hq-worktrees/run-20260807` are untouched — this card never had cause to touch any of them.

---

## PARK condition — resolved, NOT parked

The card's PARK condition is tripped only if the `uid`-mismatch behavior for B-89 required a fresh
product judgment. It does not: `index.html:228-237`'s `readCachedApps(deviceId)` already ships the
answer — `env.uid!==deviceId` (or a bare array, or `!Array.isArray(env.apps)`) returns `null`,
meaning "treat as nothing cached," not a thrown/surfaced error. `checkAuth()` (`index.html:325-326`)
then simply skips painting cached tiles and falls through to the live `/api/v1/me/apps` fetch — the
crew sees no error, just a beat with no cached tiles, exactly as if the device had never cached
anything. `cachedGrantSlugs()` mirrors this: an invalid/mismatched envelope resolves to `[]`, its
own established "nothing cached" value (already returned today for `!raw` and for a parse
exception) — not a distinct error path. Cited: `index.html:224-237` (the comment block + function),
`index.html:262-336` (`checkAuth()`'s consumption of `readCachedApps`'s `null`).

---

## Red-first

**B-89.** Fixture rewritten first (`tests/sync-rxdb-client.spec.js`, `window.HQSync is
constructed, pinned and umbrella-expanded`) to plant the real `{uid, apps}` envelope +
matching `hq-identity` cache token instead of the old bare array. Run against
UNMODIFIED `sync-rxdb/bootstrap.js`:

```
npx playwright test tests/sync-rxdb-client.spec.js -g "window.HQSync is constructed, pinned and umbrella-expanded"
  ✘ ... (1.5s)
    Error: expect(received).toEqual(expected)
    - Expected: ["inventory","inventory-cost","inventory-trends","operations"]
    + Received: []
  1 failed
EXIT=1
```

Then `cachedGrantSlugs()` fixed to read the envelope + verify `uid`. Same test, same
command, post-fix:

```
  ✓ 1 [chromium] › ... window.HQSync is constructed, pinned and umbrella-expanded (1.0s)
  1 passed (7.6s)
EXIT=0
```

Full spec file re-run post-fix: **55/55 passed**. Logs:
`.night-crew/runs/2026-08-07-autonomous/a2-logs/b89-{red,green,full-spec-green}.log`.

**B-132.** New regression test added first (`tests/workflows.spec.js`, describe block
`B-132 — fireworks confetti canvas`) asserting zero `pageerror` events matching
`/radius|arc/` AND zero `<canvas>` elements in the DOM 3s after a completed checklist
submit at 393×852. Run against UNMODIFIED `workflows.html`:

```
npx playwright test tests/workflows.spec.js -g "completed submit does not throw ctx.arc negative radius"
[WebServer] client log ERROR: Uncaught IndexSizeError: Failed to execute 'arc' on
  'CanvasRenderingContext2D': The radius provided (-0.0101965) is negative.
  at http://localhost:4523/workflows.html?t=...:711
[B-132 RF] canvas elements still in DOM at 3s: 1
  ✘ 1 ... completed submit does not throw ctx.arc negative radius; overlay does not
    freeze on screen (5.0s)
    Error: ctx.arc threw a negative-radius error inside the fireworks() animate() loop
    - Expected: []
    + Received: ["IndexSizeError: ... radius provided (-0.0101965) is negative."]
  1 failed
EXIT=1
```

Then the radius clamped (`Math.max(0, p.size*p.life)`) in `workflows.html`'s
`fireworks()`. Same test, same command, post-fix:

```
  ✓ 1 [chromium] › ... completed submit does not throw ctx.arc negative radius; overlay
    does not freeze on screen (4.8s)
  1 passed (12.9s)
EXIT=0
```

Screenshot 3s after the completed submit, 393×852, captured BOTH pre-fix
(`b132-before-frozen-overlay-3s.png`) and post-fix (`b132-completed-submit-3s.png`) —
**visually identical**: the My Checklists list + "Submitted for approval" toast, no
confetti visible in either. This settles what the card asked to establish: the "frozen
overlay" is NOT a visible frozen burst of confetti (contrary to this bug's original
BACKLOG.md description, which speculated "bursts 2–5 are still bright"). The crash fires
on the frame where the FIRST burst's particles cross `life<=0` — at that point
`ctx.clearRect()` has already wiped the previous frame, and the throw happens on the very
first particle processed this frame (all of burst 1 shares one `life` trajectory, so they
cross zero simultaneously), before any particle — from burst 1 or any later burst — gets
redrawn. The canvas freezes fully transparent. The real, confirmed defect is a **leaked,
invisible `<canvas>` DOM node** (1 pre-fix / 0 post-fix, asserted directly via
`document.querySelectorAll('canvas').length`), not a visual glitch. Logs + screenshots:
`.night-crew/runs/2026-08-07-autonomous/a2-logs/b132-{red,green}.log`,
`.night-crew/runs/2026-08-07-autonomous/a2-logs/b132-{before-frozen-overlay,completed-submit}-3s.png`.
