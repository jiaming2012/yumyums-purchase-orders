// ═══════════════════════════════════════════════════════════════════════════
// /ops ↔ REST authorization-parity coverage
// ═══════════════════════════════════════════════════════════════════════════
//
// WHY THIS FILE EXISTS
//
// POST /api/v1/workflow/ops dispatches client ops through `workflowOpRouter`
// (backend/cmd/server/main.go) to the same workflow mutations the REST routes
// call, from the SAME cookie-auth group. Two doors, one mutation, and — until
// 8c71022 — not the same lock: a `team_member` with zero approver assignments
// was refused at POST /workflow/approveSubmission and served 200 at
// POST /workflow/ops with op_type APPROVE_ITEM, and the forged approval
// broadcast over the sync hub as legitimate.
//
// That instance was fixed by moving the check into the mutation. This file
// stops the CLASS from recurring. The invariant it encodes is NOT "every
// mutating op requires authz" — some ops are deliberately open, and one pair is
// deliberately divergent pending an operator decision. The invariant is:
//
//     For every op type the router handles, the /ops door and the REST door
//     must treat an unprivileged authenticated caller IDENTICALLY — and where
//     they do not, that divergence must be written down here on purpose.
//
// ANTI-VACUITY (the whole point — see the F1 guard landed the same night)
//
// A hand-maintained list of today's ops passes green forever while a new op
// quietly opens a new door. So the op set is DERIVED from the router's own
// switch statement in main.go, and asserted EQUAL to the set covered here.
// Adding `case opsync.OpWhatever:` to the router without adding an entry below
// fails the build. Removing the derivation (or letting it silently return an
// empty set) also fails the build.
//
// `workflowOpRouter` lives in `package main`, which has no test files and
// cannot be imported from another Go package, so a Go unit test cannot walk the
// router directly. This file therefore derives the op set by parsing main.go
// and proves the authz behaviour over HTTP against a live stack — which is also
// the only way to satisfy the third property below, since "is this path
// ungated?" is a runtime question, not a source question.
//
// STALE-EXCEPTION DETECTION
//
// Exceptions assert that the /ops path IS still open. If someone later gates
// SAVE_TEMPLATE and forgets to delete its entry here, the exception goes RED
// rather than passing as fiction. The exception list cannot drift.

const { test, expect } = require('@playwright/test');
const fs = require('fs');
const path = require('path');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

const MAIN_GO = path.join(__dirname, '..', 'backend', 'cmd', 'server', 'main.go');
const OPS_GO = path.join(__dirname, '..', 'backend', 'internal', 'sync', 'ops.go');

// ─── Derivation: what op types does the router ACTUALLY handle? ──────────────
//
// Source of truth is the switch in workflowOpRouter, not the constant block —
// a constant with no `case` falls through to `default:` (400 unknown_op_type)
// and opens no door. Every failure mode of this parser throws loudly; it must
// never degrade into "found nothing, therefore nothing to check".

function routerOpTypes() {
  const mainGo = fs.readFileSync(MAIN_GO, 'utf8');
  const opsGo = fs.readFileSync(OPS_GO, 'utf8');

  const constMap = {};
  for (const m of opsGo.matchAll(/^\s*(Op[A-Za-z0-9_]*)\s*=\s*"([^"]+)"/gm)) {
    constMap[m[1]] = m[2];
  }
  if (Object.keys(constMap).length === 0) {
    throw new Error(`ops-authz-coverage: parsed ZERO op-type constants from ${OPS_GO}. ` +
      'The derivation is broken — fix the parser, do not delete this assertion.');
  }

  const start = mainGo.indexOf('func workflowOpRouter(');
  if (start < 0) {
    throw new Error(`ops-authz-coverage: could not find func workflowOpRouter in ${MAIN_GO}. ` +
      'If the router was renamed or moved, update this parser — the coverage guard ' +
      'is worthless the moment it stops finding the router.');
  }
  const after = mainGo.slice(start);
  const endRel = after.search(/\n\}\n/);
  if (endRel < 0) {
    throw new Error('ops-authz-coverage: could not find the end of workflowOpRouter.');
  }
  const body = after.slice(0, endRel);

  const found = [];
  for (const m of body.matchAll(/^\s*case\s+([^:\n]+):\s*$/gm)) {
    for (const raw of m[1].split(',')) {
      const label = raw.trim();
      const cm = /^opsync\.(Op[A-Za-z0-9_]*)$/.exec(label);
      if (!cm) {
        throw new Error(`ops-authz-coverage: unparsable case label "${label}" in workflowOpRouter. ` +
          'Every branch must be checkable — extend the parser rather than skipping it.');
      }
      const value = constMap[cm[1]];
      if (!value) {
        throw new Error(`ops-authz-coverage: case opsync.${cm[1]} has no string constant in ${OPS_GO}.`);
      }
      found.push(value);
    }
  }
  if (found.length === 0) {
    throw new Error('ops-authz-coverage: derived ZERO op types from the router switch. ' +
      'A green run on an empty derived set is exactly the vacuous pass this file exists to prevent.');
  }
  return found.sort();
}

// ─── Helpers (duplicated per-file, matching this suite's convention) ─────────

async function login(page, email, password) {
  await page.goto('/login.html');
  await page.fill('input[type="email"]', email || ADMIN_EMAIL);
  await page.fill('input[type="password"]', password || ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
}

async function apiCall(page, method, path_, body) {
  return page.evaluate(async ([m, p, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const res = await fetch('/api/v1/workflow/' + p, opts);
    if (res.status === 204) return null;
    return res.json();
  }, [method, path_, body]);
}

function uuid() {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0;
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16);
  });
}

// rest() issues a raw REST call from the logged-in browser context and returns
// the status only — the status IS the authz answer.
async function rest(page, method, url, body) {
  return page.evaluate(async ([m, u, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const r = await fetch(u, opts);
    return r.status;
  }, [method, url, body]);
}

// op() issues the same mutation through the /ops side door.
async function op(page, opType, entityType, entityId, payload) {
  return page.evaluate(async ([t, et, eid, p]) => {
    const r = await fetch('/api/v1/workflow/ops', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        op_type: t,
        device_id: 'coverage-' + Date.now() + '-' + Math.random().toString(36).slice(2),
        entity_id: eid,
        entity_type: et,
        lamport_ts: Date.now(),
        payload: p,
      }),
    });
    return r.status;
  }, [opType, entityType, entityId, payload]);
}

function minimalTemplate(name) {
  return {
    name,
    requires_approval: false,
    sections: [{
      title: 'S1', order: 0, condition: null,
      fields: [{ type: 'checkbox', label: 'A', required: false, order: 0, config: null, fail_trigger: null, condition: null }],
    }],
    schedules: [{ active_days: [0, 1, 2, 3, 4, 5, 6] }],
    assignments: [],
  };
}

// buildFixture provisions, as superadmin: a template requiring approval with an
// ADMIN-role approver, a pending submission on it, and two throwaway templates
// for the archive legs. Then it invites and logs in a fresh `team_member` who
// holds NO approver assignment anywhere — the unprivileged caller under test.
async function buildFixture(page, tag) {
  await login(page);

  const email = `ops-cov-${tag}-${Date.now()}@yumyums.kitchen`;
  const invite = await page.evaluate(async (em) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Cov', last_name: 'Stranger', email: em, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' }),
    });
  }, invite.invite_path.split('token=')[1]);

  await login(page);

  const tpl = await apiCall(page, 'POST', 'createTemplate', {
    name: `Ops Coverage ${tag} ${Date.now()}`,
    sections: [{
      title: 'S1', order: 0, condition: null,
      fields: [{ type: 'checkbox', label: 'Item A', required: false, order: 0, config: null, fail_trigger: null, condition: null }],
    }],
    schedules: [{ active_days: [0, 1, 2, 3, 4, 5, 6] }],
    assignments: [
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' },
      { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' },
    ],
    requires_approval: true,
  });
  const full = (await apiCall(page, 'GET', 'templates')).find(t => t.id === tpl.id);
  const fieldId = full.sections[0].fields[0].id;

  await apiCall(page, 'POST', 'submitChecklist', {
    template_id: tpl.id, idempotency_key: uuid(),
    responses: [{ field_id: fieldId, value: { value: true } }],
  });
  const pending = await apiCall(page, 'GET', 'pendingApprovals');
  const submissionId = (pending.find(s => s.template_id === tpl.id) || pending[0]).id;

  // Two disposable templates: one per archive door, so neither leg's result can
  // be explained by the other leg having already archived the row.
  const throwawayRest = await apiCall(page, 'POST', 'createTemplate', minimalTemplate(`Cov Throwaway REST ${tag} ${Date.now()}`));
  const throwawayOps = await apiCall(page, 'POST', 'createTemplate', minimalTemplate(`Cov Throwaway OPS ${tag} ${Date.now()}`));

  await login(page, email, 'test456');
  return { templateId: tpl.id, fieldId, submissionId, throwawayRest: throwawayRest.id, throwawayOps: throwawayOps.id, email };
}

// ─── THE COVERAGE DECLARATION ───────────────────────────────────────────────
//
// Every op type the router handles MUST appear in exactly one bucket. The
// equality test below enforces that mechanically.
//
//   PARITY_GATED  — both doors REFUSE the unprivileged caller (403).
//   PARITY_OPEN   — both doors ACCEPT any authenticated caller. These ops are
//                   self-scoped by construction (the mutation is keyed to the
//                   caller's own user id), so there is no privilege to escalate
//                   and no REST-side gate to be out of step with.
//   EXCEPTIONS    — KNOWN, DELIBERATE divergence: the REST twin refuses and the
//                   /ops path accepts. Each entry names the open decision.
//                   These are NOT approved; they are recorded so the guard is
//                   honest about them instead of being switched off.

const PARITY_GATED = {
  // approveSubmission — sets checklist_submissions.status='approved'.
  // Both doors gated by requireReviewAuthz INSIDE the mutation (8c71022).
  APPROVE_ITEM: {
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/approveSubmission', { submission_id: fx.submissionId }),
    ops: (p, fx) => op(p, 'APPROVE_ITEM', 'submission', fx.submissionId, { submission_id: fx.submissionId }),
  },
  // rejectItem — inserts submission_rejections + status='rejected'. Same gate.
  REJECT_ITEM: {
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/rejectItem', { submission_id: fx.submissionId, field_id: fx.fieldId, comment: 'front door' }),
    ops: (p, fx) => op(p, 'REJECT_ITEM', 'submission', fx.submissionId, { submission_id: fx.submissionId, field_id: fx.fieldId, comment: 'side door' }),
  },
};

const PARITY_OPEN = {
  // saveResponse — upserts a DRAFT response row keyed (field_id, answered_by).
  // The caller can only ever write their own draft; REST enforces authn only.
  SET_FIELD: {
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/saveResponse', { field_id: fx.fieldId, value: { value: true } }),
    ops: (p, fx) => op(p, 'SET_FIELD', 'field_response', fx.fieldId, { field_id: fx.fieldId, value: { value: true } }),
  },
  // submitChecklist — creates a submission attributed to the caller and sweeps
  // the caller's own drafts into it. REST enforces authn + the same two
  // validators the router calls (validateFailNotes, validateResubmitPhoto).
  SUBMIT_CHECKLIST: {
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/submitChecklist', { template_id: fx.templateId, idempotency_key: uuid(), responses: [] }),
    ops: (p, fx) => op(p, 'SUBMIT_CHECKLIST', 'submission', '', { template_id: fx.templateId, idempotency_key: uuid(), responses: [] }),
  },
};

const EXCEPTIONS = {
  // ── KNOWINGLY UNGATED ──────────────────────────────────────────────────────
  // SAVE_TEMPLATE creates/updates a checklist template (insertTemplate /
  // updateTemplate). Its REST twins — POST /workflow/createTemplate and
  // PUT /workflow/updateTemplate/{id} — are admin-only (isAdmin, D-11). The
  // /ops branch carries NO check, so any authenticated crew member can author
  // or rewrite a template through the side door.
  //
  // OPEN DECISION: DECISIONS-NEEDED §1(b) of run 2026-07-20c — "whether template
  // mutation needs any gate at all" is a PRODUCT question and has NOT been
  // answered. This entry is the living form of that question, not a waiver.
  SAVE_TEMPLATE: {
    decision: 'run 2026-07-20c DECISIONS-NEEDED §1(b) — is template mutation meant to be crew-writable?',
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/createTemplate', minimalTemplate(`Cov Forge ${Date.now()}`)),
    ops: (p, fx) => op(p, 'SAVE_TEMPLATE', 'template', '', minimalTemplate(`Cov Forge Ops ${Date.now()}`)),
  },
  // ARCHIVE_TEMPLATE soft-deletes a template (archiveTemplate). Its REST twin
  // DELETE /workflow/archiveTemplate/{id} is admin-only (isAdmin, D-11). The
  // /ops branch carries no check, so any authenticated crew member can archive
  // any template by id — including one they cannot see.
  //
  // OPEN DECISION: same fork, DECISIONS-NEEDED §1(b).
  ARCHIVE_TEMPLATE: {
    decision: 'run 2026-07-20c DECISIONS-NEEDED §1(b) — is template mutation meant to be crew-writable?',
    rest: (p, fx) => rest(p, 'DELETE', '/api/v1/workflow/archiveTemplate/' + fx.throwawayRest, null),
    ops: (p, fx) => op(p, 'ARCHIVE_TEMPLATE', 'template', fx.throwawayOps, { template_id: fx.throwawayOps }),
  },
};

// ═══════════════════════════════════════════════════════════════════════════

test.describe('/ops ↔ REST authz parity coverage', () => {

  // This suite shares ONE server and ONE database with every other spec file.
  // These tests deliberately leave forged artefacts behind (that IS the proof),
  // so clean up after each one — an un-approved submission or a live template
  // named by this file must not surface in another spec's listings.
  test.afterEach(async ({ page }) => {
    try {
      await login(page);
      const pending = await apiCall(page, 'GET', 'pendingApprovals');
      for (const s of (pending || [])) {
        await apiCall(page, 'POST', 'approveSubmission', { submission_id: s.id });
      }
      const templates = await apiCall(page, 'GET', 'templates');
      for (const t of (templates || [])) {
        if (/^(Ops Coverage|Cov )/.test(t.name || '')) {
          await apiCall(page, 'DELETE', 'archiveTemplate/' + t.id);
        }
      }
    } catch (_) { /* cleanup is best-effort; never mask a real failure */ }
  });

  // ── (b) ANTI-VACUITY ─────────────────────────────────────────────────────
  // This test needs no server. If it is ever the only thing standing between a
  // new op type and an ungated door, it must still work.
  test('COVERAGE-EQ: the covered op set equals the set workflowOpRouter handles', async () => {
    const routed = routerOpTypes();
    const covered = [
      ...Object.keys(PARITY_GATED),
      ...Object.keys(PARITY_OPEN),
      ...Object.keys(EXCEPTIONS),
    ].sort();

    // No op may be declared twice — a duplicate would make the counts line up
    // while leaving a real op uncovered.
    expect(new Set(covered).size, 'an op type is declared in more than one bucket').toBe(covered.length);

    const uncovered = routed.filter(o => !covered.includes(o));
    expect(uncovered,
      `workflowOpRouter handles op type(s) with NO authz-parity coverage: ${uncovered.join(', ')}. ` +
      'Add each to PARITY_GATED, PARITY_OPEN, or EXCEPTIONS in this file — a new op ' +
      'type is a new door, and this guard exists so a new door cannot ship unexamined.'
    ).toEqual([]);

    const stale = covered.filter(o => !routed.includes(o));
    expect(stale,
      `this file covers op type(s) the router no longer handles: ${stale.join(', ')}. ` +
      'Remove them — coverage of a dead branch is coverage of nothing.'
    ).toEqual([]);

    expect(covered).toEqual(routed);
  });

  // ── PARITY_GATED: both doors refuse ──────────────────────────────────────
  for (const [opType, spec] of Object.entries(PARITY_GATED)) {
    test(`GATED ${opType}: refused (403) at BOTH the REST door and /ops`, async ({ page }) => {
      const fx = await buildFixture(page, opType.toLowerCase().replace(/_/g, ''));

      const restStatus = await spec.rest(page, fx);
      expect(restStatus, `${opType}: the REST twin stopped refusing an unprivileged caller`).toBe(403);

      const opsStatus = await spec.ops(page, fx);
      expect(opsStatus,
        `${opType}: the /ops side door does NOT enforce the gate its REST twin enforces. ` +
        'This is the exact bypass class that let a zero-assignment team_member forge an approval.'
      ).toBe(403);

      // The status code is the claim; the absent mutation is the proof.
      await login(page);
      const stillPending = await apiCall(page, 'GET', 'pendingApprovals');
      expect((stillPending || []).some(s => s.id === fx.submissionId),
        `${opType}: refused with 403 but the submission was mutated anyway`).toBe(true);
    });
  }

  // ── PARITY_OPEN: both doors accept ───────────────────────────────────────
  for (const [opType, spec] of Object.entries(PARITY_OPEN)) {
    test(`OPEN ${opType}: accepted at BOTH the REST door and /ops`, async ({ page }) => {
      const fx = await buildFixture(page, opType.toLowerCase().replace(/_/g, ''));

      const restStatus = await spec.rest(page, fx);
      const opsStatus = await spec.ops(page, fx);

      // Parity, in the other direction: if one door were gated and the other not,
      // that is the same class of defect regardless of which side moved.
      expect(restStatus < 400,
        `${opType}: the REST twin now refuses (${restStatus}) an ordinary caller. If that gate ` +
        'is intended, the /ops branch must match it and this op must move to PARITY_GATED.').toBe(true);
      expect(opsStatus < 400,
        `${opType}: /ops now refuses (${opsStatus}) an ordinary caller while REST accepts (${restStatus}). ` +
        'The two doors have diverged.').toBe(true);
    });
  }

  // ── (c) EXCEPTIONS: documented divergence, asserted to STILL be divergent ─
  for (const [opType, spec] of Object.entries(EXCEPTIONS)) {
    test(`EXCEPTION ${opType}: still knowingly ungated on /ops (${EXCEPTIONS[opType].decision})`, async ({ page }) => {
      const fx = await buildFixture(page, opType.toLowerCase().replace(/_/g, ''));

      const restStatus = await spec.rest(page, fx);
      expect(restStatus,
        `${opType}: the REST twin is documented as admin-only but returned ${restStatus}. ` +
        'Either the REST gate was removed, or this exception entry describes a world that no longer exists.'
      ).toBe(403);

      const opsStatus = await spec.ops(page, fx);
      expect(opsStatus < 400,
        `${opType}: /ops returned ${opsStatus} — it now appears to be GATED. If that is intended, ` +
        `DELETE this exception entry and move ${opType} to PARITY_GATED. A stale exception is a lie ` +
        'about the system, and this assertion exists so the list cannot rot into one.'
      ).toBe(true);
    });
  }
});
