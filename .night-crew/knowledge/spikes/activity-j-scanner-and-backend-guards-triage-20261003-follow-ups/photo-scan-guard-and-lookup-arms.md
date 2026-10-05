# Spikes — photo-scan-guard-and-lookup-arms

Activity: Activity J — Scanner and backend guards (triage 20261003 follow-ups)

> Tool-run (`night-crew spikes run`). Both spikes run in a THROWAWAY git worktree off `dev`
> (`../hq-worktrees/spike-j1-20261005`) with the repo's `node_modules` symlinked in, against a
> spike-owned Playwright stack: `TEST_DB_NAME=hq_test_spike_j1_20261005`, `TEST_PORT=8321`, on
> `:5434` (`yumyums-test-pg`, role `hqtest`). The sync door is mocked at the network layer exactly
> as `tests/marketing.spec.js` mocks it; no substrate is touched. Never `:5433`.

## The goal, and which legs need a spike

The card (J1 — B-475, B-476; ledger T-64 decision 205): a photo picked while the scanner waits on
the server for code A leaves A's offer on screen with no submit control (`onFilePicked` in
`marketing/scan-page.js` has no busy guard; the camera path has `decodeBusy`), and the scan-time
check's four untested arms — held-and-locally-expired, an expired server row, a non-200 answer, a
throwing policy source — each leave every spec in `tests/marketing.spec.js` green when broken.

Two premises a script can settle now: (1) the stuck state reproduces by execution on today's
`dev` through the shipped page — the card's `[PS-01]` red-first baseline — and the camera-path
guard is the right shape (no second scan reaches the machine while one is resolving); (2) at least
one of the four B-476 arms is genuinely ungated: with the non-200 branch of `createServerLookup`
mutated to answer `null` ("the server does not know it") instead of rejecting ("could not ask"),
the whole marketing spec stays green — which is what makes B-476 a test-gap card and sizes it.

## Spike: photo-pick-during-held-lookup-leaves-no-submit

- proves: on a throwaway worktree of `dev`, with the lookup for never-seen code A held at the
  network layer, picking a photo of locally-held code B mid-wait raises the F6 "Finish the
  current customer first" prompt, and when A's lookup is then released as a live row A's offer
  card renders with NO `#ms-order` (the stuck state); the control — the same scan with no photo
  picked — renders A's offer WITH `#ms-order`.
- plan: `git worktree add --detach` off `dev`, symlink node_modules (root, harness, qa/rxdb),
  warm the backend build, copy the spike's spec into the worktree's `tests/`, run it with the
  spike-owned `TEST_DB_NAME` / `TEST_PORT`, read the JSON reporter per spec, remove the worktree
  and drop the database.
- script: .night-crew/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/photo-scan-guard-and-lookup-arms/01-photo-pick-during-held-lookup-leaves-no-submit.sh

### Runs
- 2026-10-04T22:56:45Z · exit 0 · passed

## Spike: non-200-lookup-answering-null-leaves-marketing-green

- proves: in the same kind of worktree, with `createServerLookup`'s `if (res.status !== 200)
  throw …` mutated to `return null`, the WHOLE of `tests/marketing.spec.js` (`--retries=0`) exits
  0 with 0 failed — the non-200 arm has no gate (one of B-476's four).
- plan: worktree as above; apply the mutation with a single-hunk check; run the full marketing
  spec with the JSON reporter; assert exit 0, failed == 0, passed ≥ 60; restore; remove.
- script: .night-crew/spikes/activity-j-scanner-and-backend-guards-triage-20261003-follow-ups/photo-scan-guard-and-lookup-arms/02-non-200-lookup-answering-null-leaves-marketing-green.sh

### Runs
- 2026-10-04T22:59:53Z · exit 0 · passed

## Verdict (tool-run 2026-10-04)

- **photo-pick-during-held-lookup-leaves-no-submit: passed** — exit 0 on the first run (3.1 m
  Playwright, both specs ok). Through the SHIPPED page (provisioning, resolver, lookup, submit
  machine real; the sync door mocked at the network layer): with never-seen code A's lookup held,
  `#scan-result` read `checkingServer`; a photo of locally-held code B (`tests/fixtures/qr-fixture-1.png`)
  picked mid-wait raised `#scan-prompt` "Finish the current customer first"; releasing A as a live
  row rendered `offerReady` / `data-source=server` and, 1.5 s later, the page read
  `{"msOrder":0,"slot":1,"prompt":true,"kind":"offerReady"}` — the submit slot mounted EMPTY, no
  order-number field, and the F6 prompt STILL open. The control (same scan, no photo) rendered
  `#ms-order` with no prompt. B-475 reproduced exactly as the review described it.
- **non-200-lookup-answering-null-leaves-marketing-green: passed** — exit 0 on the first run
  (3.9 m): `createServerLookup`'s non-200 branch mutated from `throw` to `return null` (numstat
  1/1), the WHOLE of `tests/marketing.spec.js` `--retries=0`: **63 passed / 0 failed / 0 skipped /
  0 flaky**, exit 0. The arm has no gate; B-476's test-gap premise holds for it.

## Corrections

- none agent-reached — both premises held as the ledger states them. Two precisions carried into
  the card: (1) `[PS-01]`'s green must assert BOTH `#ms-order` present AND `#scan-prompt` absent —
  the spike shows the prompt outliving A's resolution, so a fix that only mounts the field but
  leaves the prompt would still strand the crew behind a sheet; (2) the spec's photo-pick lands via
  `page.setInputFiles('#scan-file', …)` with the lookup held by `page.route` — the recipe for the
  card's red-first run and for G6's own drive.

## Comebacks

- none recorded. B-477 (the vendored-rxdb symlink a fresh worktree needs for `[TI-01]`) was
  handled by `_lib.sh`'s three symlinks, as the slate's box rule prescribes; not a spike gap.

## Review

- signed: operator, 2026-10-04 — covers 0 correction(s) (reviewed at the slate sitting of 2026-10-04, §4 batch sign-off; no corrections to review; the two
  precisions stated).
