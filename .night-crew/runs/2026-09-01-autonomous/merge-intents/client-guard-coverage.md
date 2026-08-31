# Merge intent — `client-guard-coverage`

Run `20260901`, Track B (first in track). Closes **B-149**, **B-10**; carries the **B-154 rider**.

## Shared files touched

- **`night-crew.toml`** — the **B-154 rider**. Adds one `[e2e.seams]` row mapping the sync-rxdb
  client paths (`sync-rxdb/`) to the sync spec subset. This is the ONLY shared/config file
  touched. Until this row exists every sync-touching card de-confines to the full ~22 min suite;
  with it, later sync cards (`cdc-single-fire`, `app-slug-association`, `sync-doc-honesty`,
  `sync-dev-one-command`, …) confine their gate leg to the sync specs. The value is authored
  deliberately against the sync footprint — the spec name-stems that exercise the sync client.
- **`tests/sync-rxdb-client.spec.js`** — B-149 guard test added to the existing BROWSER describe
  block (`workflows.html actually imports and constructs the client`), reusing that block's
  `seedGrantEnvelope` / `login` helpers.
- **`tests/index.spec.js`** — B-10 guard test added to the api-cache-hygiene section, reusing its
  `login` helper.

No production code (`.js`/`.html`/`.go`) is changed — this is a coverage-only card. No precached
asset changes, so G4 is N/A-by-footprint.

## What must survive any merge

1. **The new `[e2e.seams]` row for `sync-rxdb/` paths in `night-crew.toml`.** LATER SYNC CARDS
   DEPEND ON IT to confine their gate legs — dropping it silently re-inflates every sync card's
   gate to the full suite and mis-prices the night.
2. **The B-149 guard test** (`tests/sync-rxdb-client.spec.js`) — a mismatched-uid envelope must
   resolve to nothing-cached (`window.HQSync.surfaces === []`). It reds when the
   `env.uid !== deviceId` clause is removed from `bootstrap.js`'s `cachedGrantSlugs()`.
3. **The B-10 guard test** (`tests/index.spec.js`) — logout's `await purgeDeviceIdentity()` must
   be honored: the redirect must not fire until the (deliberately slowed) cache-clear resolves.
   It reds when the `await` is dropped.

## What is safe to drop

Nothing.
