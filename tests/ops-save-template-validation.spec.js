// ═══════════════════════════════════════════════════════════════════════════
// /ops SAVE_TEMPLATE — requires_approval must name an approver
// ═══════════════════════════════════════════════════════════════════════════
//
// THE DEFECT
//
// Both REST twins refuse a template that requires approval but names no
// approver: POST /workflow/createTemplate (handler.go, CreateTemplateHandler)
// and PUT /workflow/updateTemplate/{id} (UpdateTemplateHandler) each call
// hasApprover(input.Assignments) and answer 400 {"error":"requires_approver"}.
//
// POST /workflow/ops with op_type SAVE_TEMPLATE reaches the SAME two writes
// through workflowOpRouter → workflow.CreateTemplateFunc / UpdateTemplateFunc,
// and carried NO such check. An authenticated caller with no privilege at all
// could therefore author a template with requires_approval=true and zero
// approver assignments. Submissions against it are accepted (201) and then sit
// `pending` forever: pendingApprovals matches reviewers via
// template_assignments WHERE assignment_role='approver', so a template with
// none matches nobody. The submission is in no queue and has no in-app route to
// resolution — the approval ledger is silently corrupted.
//
// WHAT THIS FILE ASSERTS — AND WHAT IT DELIBERATELY DOES NOT
//
// It asserts a VALIDATION contract: 400, at both doors, for an approverless
// approval template, with the mutation provably absent.
//
// It does NOT assert an AUTHORIZATION contract. Whether an unprivileged crew
// member should be able to mutate templates over /ops at all is an OPEN
// PRODUCT question (run 2026-07-20c DECISIONS-NEEDED §1-B), recorded as a live
// exception in tests/ops-authz-coverage.spec.js. The "still accepted" tests
// below pin that distinction down: the same unprivileged caller posting a VALID
// template over /ops must still be accepted. If this file ever starts seeing
// 403 there, the validation fix has overshot into the authz question and both
// this file and the exception list in ops-authz-coverage.spec.js are wrong.

const { test, expect } = require('@playwright/test');

const ADMIN_EMAIL = 'jamal@yumyums.kitchen';
const ADMIN_PASSWORD = 'test123';

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

// op() posts through the /ops side door and returns BOTH status and body — the
// body carries the error code, and "400 requires_approver" vs "400 something
// else" is the difference between the gate firing and a coincidence.
async function op(page, opType, entityType, entityId, payload) {
  return page.evaluate(async ([t, et, eid, p]) => {
    const r = await fetch('/api/v1/workflow/ops', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        op_type: t,
        device_id: 'ra-' + Date.now() + '-' + Math.random().toString(36).slice(2),
        entity_id: eid,
        entity_type: et,
        lamport_ts: Date.now(),
        payload: p,
      }),
    });
    let body = null;
    try { body = await r.json(); } catch (_) { /* empty body is fine */ }
    return { status: r.status, body };
  }, [opType, entityType, entityId, payload]);
}

async function restStatus(page, method, url, body) {
  return page.evaluate(async ([m, u, b]) => {
    const opts = { method: m, headers: { 'Content-Type': 'application/json' } };
    if (b) opts.body = JSON.stringify(b);
    const r = await fetch(u, opts);
    let body_ = null;
    try { body_ = await r.json(); } catch (_) { /* empty body is fine */ }
    return { status: r.status, body: body_ };
  }, [method, url, body]);
}

const ASSIGNEE = { assignee_type: 'role', assignee_id: 'team_member', assignment_role: 'assignee' };
const APPROVER = { assignee_type: 'role', assignee_id: 'admin', assignment_role: 'approver' };

// tpl builds a template payload. assignments defaults to assignee-only — the
// approverless shape at the heart of the defect.
function tpl(name, requiresApproval, assignments) {
  return {
    name,
    requires_approval: requiresApproval,
    sections: [{
      title: 'S1', order: 0, condition: null,
      fields: [{ type: 'checkbox', label: 'A', required: false, order: 0, config: null, fail_trigger: null, condition: null }],
    }],
    schedules: [{ active_days: [0, 1, 2, 3, 4, 5, 6] }],
    assignments: assignments || [ASSIGNEE],
  };
}

// Provision a fresh team_member holding no approver assignment anywhere — the
// unprivileged caller. Returns their credentials, logged in.
async function asUnprivilegedCrew(page, tag) {
  await login(page);
  const email = `ra-${tag}-${Date.now()}@yumyums.kitchen`;
  const invite = await page.evaluate(async (em) => {
    const res = await fetch('/api/v1/users/invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ first_name: 'Ra', last_name: 'Crew', email: em, roles: ['team_member'] }),
    });
    return res.json();
  }, email);
  await page.evaluate(async (t) => {
    await fetch('/api/v1/auth/accept-invite', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token: t, password: 'test456' }),
    });
  }, invite.invite_path.split('token=')[1]);
  await login(page, email, 'test456');
  return { email, password: 'test456' };
}

test.describe('/ops SAVE_TEMPLATE requires_approver validation', () => {

  // Shared server + shared DB: forged artefacts must not leak into other specs.
  test.afterEach(async ({ page }) => {
    try {
      await login(page);
      const templates = await apiCall(page, 'GET', 'templates');
      for (const t of (templates || [])) {
        if (/^RA(-| )/.test(t.name || '')) {
          await apiCall(page, 'DELETE', 'archiveTemplate/' + t.id);
        }
      }
    } catch (_) { /* best-effort; never mask a real failure */ }
  });

  test('CREATE: an approverless approval template is refused at /ops AND not written', async ({ page }) => {
    await asUnprivilegedCrew(page, 'create');

    const name = `RA Forge Ops ${Date.now()}`;
    const res = await op(page, 'SAVE_TEMPLATE', 'template', '', tpl(name, true, [ASSIGNEE]));

    expect(res.status,
      'the /ops SAVE_TEMPLATE branch accepted a template with requires_approval=true and no ' +
      'approver assignment — the exact payload its REST twin refuses with 400 requires_approver. ' +
      'Submissions against such a template land in NOBODY\'s pendingApprovals queue.'
    ).toBe(400);
    expect(res.body && res.body.error,
      'refused, but not for the documented reason — the error code must match the REST twin\'s.'
    ).toBe('requires_approver');

    // The status is the claim; the absent row is the proof.
    await login(page);
    const templates = await apiCall(page, 'GET', 'templates');
    expect((templates || []).some(t => t.name === name),
      `refused with ${res.status} but the template "${name}" was written anyway.`).toBe(false);
  });

  test('CREATE: the REST twin refuses the identical payload (parity, in the same run)', async ({ page }) => {
    await login(page);
    const name = `RA Forge Rest ${Date.now()}`;
    const res = await restStatus(page, 'POST', '/api/v1/workflow/createTemplate', tpl(name, true, [ASSIGNEE]));
    expect(res.status, 'the REST gate this fix mirrors has itself changed').toBe(400);
    expect(res.body && res.body.error).toBe('requires_approver');
  });

  test('UPDATE: stripping the approver over /ops is refused AND the template is untouched', async ({ page }) => {
    await login(page);
    const name = `RA Good ${Date.now()}`;
    const created = await apiCall(page, 'POST', 'createTemplate', tpl(name, true, [ASSIGNEE, APPROVER]));
    expect(created.id, 'seed template must be created').toBeTruthy();

    await asUnprivilegedCrew(page, 'update');

    const rewritten = tpl(`RA Rewritten ${Date.now()}`, true, [ASSIGNEE]);
    rewritten.id = created.id;
    const res = await op(page, 'SAVE_TEMPLATE', 'template', created.id, rewritten);

    expect(res.status,
      'the /ops SAVE_TEMPLATE update branch accepted a rewrite that removes the last approver ' +
      'while keeping requires_approval=true. updateTemplate deletes assignments before ' +
      're-inserting them, so this ORPHANS every pending submission on an existing template.'
    ).toBe(400);
    expect(res.body && res.body.error).toBe('requires_approver');

    await login(page);
    const templates = await apiCall(page, 'GET', 'templates');
    const after = (templates || []).find(t => t.id === created.id);
    expect(after, 'the seed template disappeared').toBeTruthy();
    expect(after.name, `refused with ${res.status} but the template header was rewritten anyway`).toBe(name);
    expect((after.assignments || []).filter(a => a.assignment_role === 'approver').length,
      `refused with ${res.status} but the approver assignment was destroyed anyway`).toBe(1);
  });

  // ── ANTI-OVERSHOOT ────────────────────────────────────────────────────────
  // The fix adds a 400, not a 403. The authz question stays open.
  test('NOT-A-GATE: the same unprivileged caller\'s VALID /ops SAVE_TEMPLATE is still accepted', async ({ page }) => {
    await asUnprivilegedCrew(page, 'valid');

    const noApproval = await op(page, 'SAVE_TEMPLATE', 'template', '', tpl(`RA Valid NoApproval ${Date.now()}`, false, [ASSIGNEE]));
    expect(noApproval.status,
      'an unprivileged /ops SAVE_TEMPLATE with requires_approval=false now returns ' +
      `${noApproval.status}. If that is 403, the requires_approver fix has overshot into the ` +
      'OPEN authz question (DECISIONS-NEEDED §1-B) and has silently answered a product ' +
      'decision nobody made — and the SAVE_TEMPLATE exception in ops-authz-coverage.spec.js ' +
      'is now a lie.').toBeLessThan(400);

    const withApprover = await op(page, 'SAVE_TEMPLATE', 'template', '', tpl(`RA Valid WithApprover ${Date.now()}`, true, [ASSIGNEE, APPROVER]));
    expect(withApprover.status,
      'requires_approval=true WITH an approver must still be accepted — the gate must fire on ' +
      'the missing approver, not on requires_approval itself.').toBeLessThan(400);
  });
});
