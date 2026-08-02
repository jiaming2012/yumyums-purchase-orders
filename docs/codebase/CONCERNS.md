# Codebase Concerns

**Analysis Date:** 2026-04-12

## Tech Debt

**Entire frontend is static mockups with no real backend:**
- Issue: All data is hardcoded JavaScript arrays. `users.html` has `const USERS=[...]`, `purchasing.html` has `const CATS=[...]`. No API calls exist anywhere. Authentication in `login.html` is a stub that always shows the error state.
- Files: `users.html` (lines 129–159), `purchasing.html` (line 80), `login.html` (lines 52–61)
- Impact: Nothing persists. Every page load resets all state. The app cannot be used by real users until a backend is built.
- Fix approach: The backend design is fully documented in `docs/user-management-api.md`. Replace hardcoded arrays with `fetch('/api/v1/...')` calls as described in the PWA Integration Plan section of that doc.

**Login page is non-functional — always fails:**
- Issue: `login.html` `signIn()` function always calls `err.classList.add('show')` regardless of input. There is no success path. A comment says "In production this will POST to `/api/v1/auth/login`" but the code is not wired.
- Files: `login.html` (lines 52–61)
- Impact: Users cannot log in. The login page is UI-only.
- Fix approach: Implement `POST /api/v1/auth/login`, store the returned Bearer token in `localStorage` as `hq_token`, and redirect to `index.html` on success.

**No authentication protection on any page:**
- Issue: `index.html`, `purchasing.html`, `users.html` are all directly accessible with no token check. `login.html` exists but is not enforced anywhere.
- Files: All HTML files
- Impact: Any visitor with the URL can access all app pages and see the (currently mock) data.
- Fix approach: Add a token guard at the top of each protected page's `<script>` block: check `localStorage.getItem('hq_token')`, redirect to `login.html` if absent. Once the backend exists, validate the token via `GET /api/v1/me`.

**Service worker serves stale content with no validation:**
- Issue: `sw.js` uses a cache-first strategy for all requests, falling back to network only on cache miss. There is no stale-while-revalidate or cache invalidation for API responses. Cache is only busted by bumping the `CACHE` constant manually (`yumyums-v5` currently).
- Files: `sw.js` (lines 18–22)
- Impact: When new HTML is deployed, existing installed PWA users continue seeing the old version until they manually refresh or the service worker updates. Network-fetched API responses (once backend exists) could also be cached unexpectedly.
- Fix approach: Add a network-first or stale-while-revalidate strategy for API requests. For HTML assets, the current cache-first with version bump is acceptable but requires discipline.

**Duplicated CSS across every page:**
- Issue: The full CSS variable block (`:root`, dark mode `@media`, `*` reset, `body`, `.app`, `.back`, `.tabs`, `.card`, `.hd`, `.row`, `.pill`) is copy-pasted verbatim into `index.html`, `purchasing.html`, `users.html`, and `login.html`.
- Files: All four HTML files (each `<style>` block)
- Impact: Any design change requires editing four files. Styles are already diverging slightly (e.g., `login.html` has a slightly different `.btn` vs `.btn-primary` naming).
- Fix approach: Extract shared styles to a `styles.css` file and `<link>` it from all pages.

**Pull-to-refresh indicator uses `background:inherit` which is unreliable:**
- Issue: `ptr.js` sets `indicator.style.cssText` including `background:inherit`. `inherit` on a `position:fixed` element inherits from the `<body>`, not the element's visual position.
- Files: `ptr.js` (line 8)
- Impact: The PTR indicator may appear transparent or show the wrong background color depending on page scroll state and browser rendering.
- Fix approach: Use an explicit background color, e.g., `background:var(--bg)` or pass the current page's background as a parameter.

**`icon-192.png` and `icon-512.png` referenced but deleted:**
- Issue: `index.html` (line 11), `login.html` (line 11), `users.html` (line 11), `purchasing.html` (line 11) all reference `href="icons/icon-192.png"` via `<link rel="apple-touch-icon">`. These files are deleted (git status shows `D icons/icon-192.png`, `D icons/icon-512.png`). The replacement files use `icon-192x192.png` naming.
- Files: All HTML files (`<link rel="apple-touch-icon" href="icons/icon-192.png">`)
- Impact: Apple home screen icon fails to load on iOS when adding to home screen. Broken icon reference.
- Fix approach: Update all four HTML files to use `href="icons/icon-192x192.png"`.

**`purchasing.html` date is hardcoded:**
- Issue: "Week of Apr 13" and "Cutoff Sun 6pm" are hardcoded strings in the HTML markup.
- Files: `purchasing.html` (lines 55, 61)
- Impact: The app will show stale dates perpetually once the week changes.
- Fix approach: When wiring to a backend, derive these from the API response. As a short-term fix, compute the current week's ordering cutoff dynamically in JavaScript.

## Known Bugs

**Login always shows error state:**
- Symptoms: Submitting valid credentials on `login.html` shows the "Invalid credentials" error message instead of proceeding.
- Files: `login.html` (line 60)
- Trigger: Any form submission.
- Workaround: None; the page is a static mockup.

**Tab switching in `users.html` calls `clearEdit()` when switching away from Tab 2:**
- Symptoms: If a user clicks a person to edit (switches to Tab 2), then taps Tab 1 to check something, and returns to Tab 2, the form is reset — all edits lost.
- Files: `users.html` (lines 286–292, specifically `if(n!==2)clearEdit()`)
- Trigger: Switching to Tab 1 or Tab 3 while editing a user.
- Workaround: Complete all edits before switching tabs.

**`purchasing.html` stepper state is lost on tab switch:**
- Symptoms: Quantity values set in Tab 1 (Form) are lost if the user switches to Tab 2 (Locked) or Tab 3 (PO) and returns.
- Files: `purchasing.html` (line 83, `show()` function hides/shows divs but DOM state for `<span>` counters persists; however, navigating away and back via browser resets the page entirely)
- Trigger: Navigation away from the page (not tab switching within the page, which preserves DOM).
- Workaround: Don't navigate away mid-order.

## Security Considerations

**Bearer token stored in `localStorage`:**
- Risk: `docs/user-management-api.md` (line 401) specifies storing the session token as `localStorage.getItem('hq_token')`. `localStorage` is accessible to any JavaScript on the page, making it vulnerable to XSS.
- Files: `docs/user-management-api.md` (lines 397–403), all future API integration code
- Current mitigation: No backend exists yet, so no real tokens are stored.
- Recommendations: Consider `HttpOnly` cookies instead of `localStorage`. If `localStorage` is kept, ensure a strong Content Security Policy header and sanitize all dynamic HTML generation.

**Unsafe `innerHTML` assignment with user-derived data:**
- Risk: `users.html` builds HTML via template literals containing data from the `USERS` array, e.g., `d.innerHTML = \`...\${u.name}\${u.email}...\``. When this is replaced with real API data, any unescaped characters in names or emails could result in XSS.
- Files: `users.html` (lines 182, 204, 209, 214)
- Current mitigation: Data is currently hardcoded, so no real injection risk today.
- Recommendations: Sanitize all user-provided strings before inserting into `innerHTML`, or use `textContent`/`createElement` for dynamic content.

**No CSRF protection designed into API spec:**
- Risk: `docs/user-management-api.md` does not mention CSRF tokens or SameSite cookie policy. If Bearer token auth is used with `localStorage`, CSRF is less of a concern, but the spec should be explicit.
- Files: `docs/user-management-api.md`
- Current mitigation: Backend does not exist yet.
- Recommendations: Add explicit CSRF guidance to the API spec. If using cookies, require `SameSite=Strict`.

**`alert()` and `confirm()` used for destructive operations:**
- Risk: `users.html` uses native `confirm()` for delete confirmation and `alert()` for success feedback. These are browser-native dialogs that bypass the app's visual design and behave differently across environments.
- Files: `users.html` (lines 269, 275, 279)
- Current mitigation: Mockup only; no real deletes happen.
- Recommendations: Replace with inline confirmation UI before wiring to real API.

## Performance Bottlenecks

**Service worker ASSETS list is not exhaustive:**
- Problem: `sw.js` caches only 7 assets. If additional pages or a stylesheet are added without updating the list, they won't be available offline.
- Files: `sw.js` (line 2)
- Cause: Manual maintenance of the assets array.
- Improvement path: When moving to a build system, generate the SW cache manifest automatically (e.g., via Workbox).

**`buildAccess()` in `users.html` rebuilds the entire Access tab DOM on every toggle/grant change:**
- Problem: `addGrant()` and `removeGrant()` both call `buildAccess()` which tears down and recreates all six app permission cards.
- Files: `users.html` (lines 218, 222–228, 230–233)
- Cause: Stateless re-render approach with no targeted DOM updates.
- Improvement path: Update only the affected card's chips and picker on mutation, rather than rebuilding all six.

## Fragile Areas

**Service worker cache version requires manual bumping:**
- Files: `sw.js` (line 1, `const CACHE = 'yumyums-v5'`)
- Why fragile: If a developer deploys new HTML without bumping the cache version, all installed PWA users continue seeing the old version silently. The version has already been bumped 5 times in rapid succession (v1→v5 across two planning phases).
- Safe modification: Always bump the version string when modifying any cached asset. Consider a build step that injects a hash.
- Test coverage: None.

**`apple-touch-icon` path inconsistency across pages:**
- Files: All HTML files
- Why fragile: `index.html` references `icons/icon-192.png` (deleted file). Other pages may reference the new `icon-192x192.png` naming. One wrong path silently fails on iOS.
- Safe modification: Standardize on `icons/icon-192x192.png` across all pages.
- Test coverage: None.

**`ptr.js` is included on `login.html`:**
- Files: `login.html` (line 65)
- Why fragile: The login page is a centered card — pull-to-refresh on a non-scrollable page triggers a reload immediately on accidental downward drag. This could interrupt a user mid-credential-entry.
- Safe modification: Remove `ptr.js` from `login.html` or add a guard in `ptr.js` checking for a page-level opt-in attribute.
- Test coverage: None.

## Missing Critical Features

**No backend exists:**
- Problem: The entire app is a static mockup. `docs/user-management-api.md` specifies a complete Go + Postgres backend but it has not been built.
- Blocks: Real user accounts, real authentication, real purchasing data, any multi-user functionality.

**No route protection / session management:**
- Problem: No page checks for an authenticated session. `login.html` cannot successfully authenticate anyone.
- Blocks: Security, multi-user use.

**`index.html` tile grid is hardcoded, not driven by `GET /api/v1/me/apps`:**
- Problem: The plan in `docs/user-management-api.md` (lines 394–402) specifies that the tile grid should be dynamically filtered by the user's app permissions. Currently all tiles are hardcoded HTML.
- Blocks: Access control — any logged-in user would see all tiles regardless of role permissions.

**No invite acceptance page:**
- Problem: `docs/user-management-api.md` describes a `POST /api/v1/auth/accept-invite` flow that lands on `login.html` "in accept-invite mode", but `login.html` has no `?token=` query param handling or password-set form.
- Blocks: New user onboarding via invite email.

## Test Coverage Gaps

**No tests of any kind:**
- What's not tested: All JavaScript logic in all four HTML files — tab switching, form state management, stepper arithmetic, permission toggle state, role filtering.
- Files: `index.html`, `purchasing.html`, `users.html`, `login.html`, `ptr.js`, `sw.js`
- Risk: Regressions in UI behavior are invisible until manually discovered.
- Priority: Medium — app is pre-backend, but once API integration begins, untested state management logic will be a source of subtle bugs.

**No service worker tests:**
- What's not tested: Cache install behavior, offline fallback, cache version rotation logic.
- Files: `sw.js`
- Risk: A bad cache version bump could silently serve stale pages to all installed PWA users with no indication of failure.
- Priority: Medium.

---

*Concerns audit: 2026-04-12*
