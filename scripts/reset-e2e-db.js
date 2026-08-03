'use strict';
// ─────────────────────────────────────────────────────────────────────────────
// reset-e2e-db.js — the ONE place the Playwright e2e database is reset, and the
// ONE place its Postgres coordinates are computed.
//
// WHY THIS FILE EXISTS (BACKLOG B-76)
// ----------------------------------
// Until this landed, the only `DROP DATABASE IF EXISTS hq_test_e2e` in the repo
// lived inside `task test` in the root Taskfile. `night-crew.toml` runs
// `npx playwright test` DIRECTLY for both `[e2e] suite` and `[e2e] subset`, and
// `playwright.config.js` had no reset of any kind — so NO night-crew gate leg,
// full or subset, had ever reset the database. Every suite figure this milestone
// has quoted was taken against a database that accumulated rows across runs.
// The dangerous half is not the false red (an onboarding test that fails only on
// a dirty DB); it is the false GREEN — a suite that passes because a previous
// run left a row behind. Measured on 2026-08-04 before the fix, `hq_test_e2e`
// held 890 sessions / 190 checklist_templates / 121 checklist_submissions with
// no run in flight.
//
// WHERE IT IS CALLED FROM, AND WHY NOT `globalSetup`
// --------------------------------------------------
// It is called from `playwright.config.js`'s `webServer.command`, as the first
// link of the shell chain that starts the Go server. That placement is
// deliberate and it is NOT a `globalSetup`:
//
//   * A `globalSetup` runs AFTER the web server. Read
//     `node_modules/playwright/lib/runner/tasks.js:100-110` — `createGlobalSetupTasks`
//     appends `createPluginSetupTasks(config)` (which is where the `webServer`
//     plugin starts the server) BEFORE `config.globalSetups`. A globalSetup that
//     dropped the database would drop it out from under a server that had
//     already connected, migrated and seeded it. The suite would then run against
//     an empty schema and fail in a way that looks like a product bug.
//   * Top-level code in `playwright.config.js` is no good either: the config file
//     is re-required in EVERY worker process (verified — the worker load carries
//     `TEST_WORKER_INDEX=0`), and it is also loaded by `npx bddgen` and by
//     `npx playwright test --list`. A reset there would fire mid-run.
//   * `webServer.command` runs exactly once, in the parent process, strictly
//     before any test and strictly before the server exists. It cannot be skipped
//     by a CLI argument, so it fires on the SUBSET path (`npx playwright test
//     "<regex>"`) just as it does on the full suite — which is the whole point of
//     B-76, since the subset path is the one night-crew uses most.
//   * When `NIGHTCREW_ENV_URL` is set, `webServer` is `undefined` and this file
//     never runs — correct, because in that mode the provisioned ephemeral stack
//     owns its own database and localhost Postgres is not it.
//
// THE GUARD
// ---------
// `assertTestDatabaseName` refuses any database whose name is not recognisably a
// test database, and it refuses LOUDLY. B-16 and B-35 are both the same lesson
// from the other direction: a harness that quietly destroys the wrong database,
// or quietly destroys nothing, is worse than one that stops and says so. A
// default that eats a dev database is unacceptable; an exception that names the
// database and the pattern it failed is not.
//
// Run it standalone with:  node scripts/reset-e2e-db.js
// ─────────────────────────────────────────────────────────────────────────────

const { execFileSync } = require('child_process');

// `hq_test`, or `hq_test_<segment>[_<segment>…]` with segments of [a-z0-9].
// Matches the names this repo actually uses — hq_test_e2e (Playwright),
// hq_test_ui (`task test:ui`), hq_test_go (the Go suite) — and refuses `hq`,
// `hq_dev`, `postgres`, `hq_testing`, and anything else. Widening this pattern
// is a deliberate act; do not widen it to make a one-off invocation work.
const TEST_DB_NAME_PATTERN = /^hq_test(?:_[a-z0-9]+)*$/;

// Coordinate defaults are duplicated NOWHERE ELSE — playwright.config.js reads
// them from here. DB_PORT 5433 is yumyums-dev-pg, the container that actually
// serves HQ; host :5432 is bound by infra-postgres-1 (slack-trading), which has
// no `yumyums` role.
function resolveE2eDb(env) {
  const e = env || process.env;
  const host = e.DB_HOST || 'localhost';
  const port = e.DB_PORT || '5433';
  const user = e.DB_USER || 'yumyums';
  const pass = e.DB_PASS || 'yumyums';
  const name = e.TEST_DB_NAME || 'hq_test_e2e';
  return {
    host,
    port,
    user,
    pass,
    name,
    adminUrl: `postgres://${user}:${pass}@${host}:${port}/postgres?sslmode=disable`,
    // testUrl carries `TimeZone`, which pgx understands and libpq does NOT
    // (`psql: error: invalid URI query parameter: "TimeZone"`). psqlUrl is the
    // same database without it, for anything shelling out to psql.
    testUrl: `postgres://${user}:${pass}@${host}:${port}/${name}?sslmode=disable&TimeZone=America/New_York`,
    psqlUrl: `postgres://${user}:${pass}@${host}:${port}/${name}?sslmode=disable`,
  };
}

function assertTestDatabaseName(name) {
  if (!TEST_DB_NAME_PATTERN.test(name)) {
    throw new Error(
      `refusing to reset database ${JSON.stringify(name)}: it does not look like a test database.\n` +
        `  TEST_DB_NAME must match ${TEST_DB_NAME_PATTERN} (e.g. hq_test_e2e, hq_test_ui, hq_test_go).\n` +
        `  This guard exists so a mistyped or inherited TEST_DB_NAME cannot DROP a dev or production\n` +
        `  database. If you genuinely need a new test database name, widen the pattern in\n` +
        `  scripts/reset-e2e-db.js deliberately — do not work around it with an env var.`
    );
  }
  return name;
}

// DROP + CREATE, not TRUNCATE. The server re-runs goose migrations on startup,
// so a bare CREATE is enough — and a drop also clears anything a previous run
// created OUTSIDE the migration set, which a truncate list would miss. This must
// complete before the Go server starts, which is why it is the first link of
// webServer.command rather than a globalSetup.
function resetE2eDatabase(env) {
  const db = resolveE2eDb(env);
  assertTestDatabaseName(db.name);
  execFileSync(
    'psql',
    [
      db.adminUrl,
      '-v',
      'ON_ERROR_STOP=1',
      '-q',
      '-c',
      `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${db.name}' AND pid<>pg_backend_pid();`,
      '-c',
      `DROP DATABASE IF EXISTS ${db.name};`,
      '-c',
      `CREATE DATABASE ${db.name};`,
    ],
    { stdio: ['ignore', 'ignore', 'inherit'] }
  );
  return db;
}

module.exports = { TEST_DB_NAME_PATTERN, resolveE2eDb, assertTestDatabaseName, resetE2eDatabase };

if (require.main === module) {
  try {
    const db = resetE2eDatabase();
    // Printed on every run so a gate log says which database it reset. A silent
    // reset is indistinguishable from no reset at all, which is B-76's mechanism.
    //
    // 🛑 console.ERROR, not console.log, and that is B-81 (do not "tidy" it back).
    // This runs as the first link of `webServer.command`, and Playwright's
    // webServer plugin pipes the child's stdout ONLY when `webServer.stdout` is
    // `'pipe'` — it defaults to `'ignore'`
    // (node_modules/playwright/lib/plugins/webServerPlugin.js:126;
    // node_modules/playwright/types/test.d.ts:10285-10289 — "Default to
    // 'ignore'"). stderr IS piped by default (same file, :10281-10283), which is
    // why the Go server's slog lines reach a gate log and this banner did not.
    // On a successful DROP psql emits no NOTICE either, so with the banner
    // swallowed a gate log carried ZERO evidence the reset had run — precisely
    // the "silent reset" this line exists to rule out.
    console.error(`── reset ${db.name} on ${db.host}:${db.port} ──`);
  } catch (err) {
    console.error(`\n[reset-e2e-db] ${err.message}\n`);
    process.exit(1);
  }
}
