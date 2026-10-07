# Merge intent — scanner-refusal-seam-and-pick-feedback (K4, run 20261007)

Card, as narrowed by operator decision 214 (the photo-pick feedback line is withdrawn because
"Scan from photo" is being removed, decision 213): the refusal a phone shows when it cannot read
its campaign policy is proven through the page, not only through the predicate (BACKLOG B-482 —
the throwing-policy refusal had no page-level gate). The B-483 half (a line of feedback after a refused photo pick) is **NOT built**.

## Shared files I touch (outside my own area)

- `marketing/scan-page.js` — the card's subject. Four places only: the file header (documents the
  test-only boot-time override), the new module-level function `adoptPolicyOverride` just above
  `boot()` (builds the shape `submit-flow.js` and this file consume from a spec's override, every
  function bound to it), the one statement in `boot()` that creates the campaign policy source
  (it calls `adoptPolicyOverride` when an override is present), and the
  `campaignPolicy.attach(campaignsRep)` call in `startSync`, which now runs only when the source
  has an `attach` (the real one always does; a test's override need not).
  `onFilePicked` and the J1 guard (`decodeBusy`) are not edited.
- `tests/marketing.spec.js` — ONE new test, `[SV-12]`, appended as its own `test.describe` at the
  end of the file. No existing test or helper is edited.
- `.night-crew/knowledge/roadmap.md` — one word on this card's Activity K line,
  `PLANNED` → `LANDED`, and one appended addendum bullet under the same card saying only the
  B-482 half landed and the B-483 half was withdrawn and not built (ledger decision 214). The
  signed card text above the addendum is unchanged.
- `.night-crew/knowledge/ledger.md` — append-only: the review rounds' deferred-finding entries for
  this card, headed `## LDG 20261007/scanner-refusal-seam-and-pick-feedback/…` (the first two are
  the "test-only policy override has no environment guard" findings, `a1/c1/adversarial/1` and
  `a1/c1/edge-case/1`). No existing entry is edited.

## What must survive any merge

- In `marketing/scan-page.js` `boot()`: the read of `window.__MARKETING_POLICY_SOURCE__` BEFORE
  `createCampaignPolicySource(cols.campaigns)` — used only when it is an object with a
  `policyFor` function — and the header paragraph saying it is a TEST-ONLY boot-time override set
  by `page.addInitScript`, never by product code. A merge that drops the read turns `[SV-12]` red
  (that is its red-first leg: the override is not read, the offer renders).
- In `marketing/scan-page.js`: the `adoptPolicyOverride` function, together with the read above.
  `boot()` calls it, so a merge that keeps the read and drops the function leaves `boot()` calling
  an undefined name — the page shows "Scanner failed to start" under any override and `[SV-12]`
  goes red for that reason, not for a rendered offer.
- In `marketing/scan-page.js` `startSync`: the `typeof campaignPolicy.attach === 'function'` guard
  on the `attach(campaignsRep)` call.
- In `.night-crew/knowledge/roadmap.md`: the card's `LANDED` word AND its addendum bullet — without
  the addendum the roadmap says the withdrawn photo-pick feedback line shipped.
- In `.night-crew/knowledge/ledger.md`: every
  `## LDG 20261007/scanner-refusal-seam-and-pick-feedback/…` entry. They are the only durable
  record of the deferred hardening findings (the reports they cite are run-local and gitignored).
- The `[SV-12]` describe block in `tests/marketing.spec.js`.

## What is safe to drop

- Nothing in the code change. Under
  `.night-crew/runs/2026-10-07-autonomous/logs/scanner-refusal-seam-and-pick-feedback/` the raw
  Playwright logs are evidence, not behaviour — a merge may keep either side's copy.

## Stubs and fixtures, per done-when clause

| Clause | Rides a stub or fixture? |
|---|---|
| `[SV-12]` RED on the pre-change tree, then green | The sync door's `page.route` mock (the scanner card's permitted stub — the file's existing `mockSyncTransports` via `openProvisionedScanner`), and a locally seeded low-value code + campaign row (fixture). The throwing policy SOURCE is the test's input, installed through the override this card adds — it is the condition under test, not a stub of the behaviour: the refusal itself is produced by the shipped `submit-flow.js` `policyFor` catch arm, the shipped machine and the shipped render. Offline is `context.setOffline(true)` — real. |
| `[PS-01]`, `[PS-02]`, `[SV-01]`–`[SV-11]`, Scanner polish, Camera scanner describes untouched and green — whole `tests/marketing.spec.js` `--retries=0` | Whatever those tests already ride (their own headers state it); this card changes none of them. |
| the withdrawn feedback line's element id and state field appear nowhere in the diff | No stub — a grep of the diff (this file deliberately does not spell either identifier, so the grep stays empty). |
| no change under `marketing/submit-*.js` | No stub — a grep of the diff. |
| `sw.js` 51, G1, G6, the full Playwright suite, the 10×/5× measurement leg | The orchestrator's — not run by this card. |

No stub of the behaviour under test. No migration. No `night-crew.toml` key. No Go change. No new
machine state, event or (state, event) pair. `sw.js` / `version.json` are not in the diff
(generated from git HEAD by the orchestrator).

## Notes for the merge

- Nothing outside the card's footprint is edited (`marketing/scan-page.js`,
  `tests/marketing.spec.js`, the roadmap line and its addendum, the appended ledger entries, this
  run directory).
- `marketing/submit-flow.js` / `marketing/submit-machine.js` are untouched; a diff there would be
  scope drift.

## Red-first and confined gates — what was run, what was observed

All Playwright legs: `TEST_PORT=8284 TEST_DB_NAME=hq_test_e2e_k4_20261007 DB_HOST=localhost
DB_PORT=5434 DB_USER=hqtest DB_PASS=hqtest` (the test-only cluster; the database is dropped and
recreated by the webServer command on every leg). Logs, with exact commands and exit lines, in
`.night-crew/runs/2026-10-07-autonomous/logs/scanner-refusal-seam-and-pick-feedback/`.

| Leg | Command | Observed |
|---|---|---|
| `[SV-12]` RED, pre-change tree (`marketing/` identical to HEAD; only the new test present) | `npx playwright test tests/marketing.spec.js -g "\[SV-12\]" --retries=0` | `1 failed`, exit 1 — the scan reached the offer state, no refusal rendered, and the page showed the order-number field and a "Submit redemption" button (`red-SV-12.log`). The spike's leg b, through the new test. |
| `[SV-12]` GREEN, with the override read | same command | `1 passed`, exit 0 (`green-SV-12.log`) |
| The whole of `tests/marketing.spec.js`, `--retries=0` | `npx playwright test tests/marketing.spec.js --retries=0` | `71 passed (3.8m)`, exit 0 (`# exit=0`), none failed / flaky / skipped — re-run 2026-10-07T08:44Z on the final tree, after the last edit to `marketing/scan-page.js` (the `adoptPolicyOverride` rewrite, 04:38 local); it replaces the earlier 04:33 log, which was a green for the tree before that edit (`marketing-spec-whole.log`) |
| Diff greps | `git diff` for the withdrawn line's id / field; `git diff --stat -- 'marketing/submit-*.js'`; `onFilePicked` in the `scan-page.js` diff | all three empty |

`[SV-12]` drives its scans through `window.MarketingScan.scanText` — the programmatic entry the
`[SV-*]` specs already use. It does not touch the "Scan from photo" control.

NOT run by this card (owed by the orchestrator on the merged tree, under the lock): the full
Playwright suite, the 10×/5× measurement leg, the `sw.js` regeneration and its 51 count, G1, G6.
No Go code changed, so no Go suite was run.
