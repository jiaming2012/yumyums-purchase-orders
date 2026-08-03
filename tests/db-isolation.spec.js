// ─────────────────────────────────────────────────────────────────────────────
// db-isolation.spec.js — the red-first test for BACKLOG B-76.
//
// It asserts ONE property: the database this suite is running against was reset
// before the run started, so nothing a PREVIOUS invocation left behind is
// visible to this one.
//
// It proves that across process boundaries, which is the only way this property
// can be proved. Each invocation checks for a marker table and then leaves one
// behind for the next invocation to trip over. So:
//
//     npx playwright test "db-isolation"     # 1st — passes, leaves the marker
//     npx playwright test "db-isolation"     # 2nd — FAILS unless the DB was reset
//
// That second invocation is the red. On the tree before `webServer.command`
// began with `node scripts/reset-e2e-db.js`, it failed with the marker present
// and a `previous_run_id` naming the run that left it. It is also, verbatim, the
// SUBSET path `night-crew.toml`'s `[e2e] subset` expands to — a positional path
// regex — which is the invocation B-76 says has never reset anything.
//
// It talks to Postgres directly rather than through the app, on purpose: the
// claim under test is about the database, not about any feature, and routing it
// through an authenticated API would make a harness assertion depend on the
// product it is supposed to be grading.
// ─────────────────────────────────────────────────────────────────────────────
const { test, expect } = require('@playwright/test');
const { execFileSync } = require('child_process');
const { resolveE2eDb } = require('../scripts/reset-e2e-db');

const db = resolveE2eDb();

function psql(sql) {
  return execFileSync('psql', [db.psqlUrl, '-v', 'ON_ERROR_STOP=1', '-tAc', sql], {
    encoding: 'utf8',
  }).trim();
}

// A run id that is stable within one process and different between processes, so
// the failure message can name WHICH earlier invocation left the residue.
const RUN_ID = `${process.pid}-${Date.now()}`;

test.describe('E2E database isolation (B-76)', () => {
  test('the e2e database carries nothing over from the previous invocation', async () => {
    // In NIGHTCREW_ENV_URL mode the suite runs against a provisioned ephemeral
    // stack whose database is NOT the localhost one these coordinates name.
    // Asserting against localhost there would grade the wrong database — a
    // worse outcome than not grading it.
    test.skip(
      !!process.env.NIGHTCREW_ENV_URL,
      'NIGHTCREW_ENV_URL is set: the provisioned stack owns its own database, not this one'
    );

    const leftovers = psql("SELECT to_regclass('public.e2e_isolation_marker')::text");

    let detail = '';
    if (leftovers) {
      detail = psql(
        'SELECT string_agg(run_id || \' @ \' || run_started_at::text, \', \') FROM e2e_isolation_marker'
      );
    }

    expect(
      leftovers,
      `${db.name} still holds the e2e_isolation_marker table left by an earlier invocation ` +
        `(${detail || 'no rows'}). The database was NOT reset before this run started, so every ` +
        `result in this suite — red OR green — may be an artifact of accumulated state (B-76). ` +
        `The reset lives in playwright.config.js's webServer.command; check that it is still there.`
    ).toBe('');

    // Leave the marker for the NEXT invocation. If the reset is working it will
    // be gone by then; if someone removes the reset, the very next run reds here.
    psql(
      'CREATE TABLE e2e_isolation_marker (' +
        'run_id text NOT NULL, run_started_at timestamptz NOT NULL DEFAULT now())'
    );
    psql(`INSERT INTO e2e_isolation_marker (run_id) VALUES ('${RUN_ID}')`);
  });
});
