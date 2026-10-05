# Extraction — photo-scan-guard-and-lookup-arms

Outcome: confirmed, no corrections

Approach used: a throwaway git worktree off `dev` with the repo's `node_modules`
symlinked in (root, harness, qa/rxdb — the B-477 rule), the backend build warmed
before Playwright's 60 s webServer window, and a spike-owned stack
(`TEST_DB_NAME=hq_test_spike_j1_20261005`, `TEST_PORT=8321`, `:5434`). Spike 01
copied a self-contained spec into the worktree's `tests/` and drove the SHIPPED
page with the sync door mocked at the network layer (the lookup HELD by
`page.route`, the photo picked with `page.setInputFiles('#scan-file', …)`); spike
02 applied a one-line mutation to `marketing/scanner.js` and ran the whole of
`tests/marketing.spec.js` `--retries=0`. Tool-recorded: both exit 0 on their
first run (3.1 m, 3.9 m). Candidate input for the card, not an adoption (NFR-6).

Confirmed: (1) B-475 — with never-seen code A's lookup held (`#scan-result` =
`checkingServer`), a photo of locally-held code B picked mid-wait raises the F6
prompt "Finish the current customer first"; when A is then released as a live
row the offer card renders (`offerReady`, `data-source=server`) with the submit
slot mounted EMPTY — `#ms-order` count 0, no submit control — and the F6 prompt
still open; the control with no photo renders `#ms-order` and no prompt.
(2) B-476 (arm 3) — with `createServerLookup`'s non-200 branch answering `null`
instead of rejecting, `tests/marketing.spec.js` is 63 passed / 0 failed; nothing
distinguishes "the server does not know it" from "could not ask".

Learned: (1) the stuck state is the submit machine left in `resolving` by a
second `doScan` the F6 gate refused mid-resolution — the camera path's
`decodeBusy` guard is exactly what the photo path lacks, and applying it means the
second scan never reaches the machine, so no new state, event or pair is needed;
(2) a correct `[PS-01]` asserts both halves — `#ms-order` present AND
`#scan-prompt` absent — because the spike shows the prompt outliving the
resolution; (3) the held-lookup + `setInputFiles` recipe is the red-first run and
the G6 drive; (4) a mutation run of the full marketing file costs ~4 m on this
box, so the card's four mutation reds are ~16 m of its implement leg, not noise.

Plan change: none to scope — the card ships the busy guard on `onFilePicked` and
the six specs (`[PS-01]`, `[PS-02]`, `[SV-08]`–`[SV-11]`) as specified; `[SV-11]`
(the throwing policy source) keeps its stated fallback — name `campaigns-run.sh`
as the only gate if the throw cannot be forced un-stubbed — rather than a
stubbed green.
