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
// POST /workflow/ops with op_type APPROVE_ITEM. (F5's G6 record additionally
// states the forged approval broadcast over the sync hub as legitimate; that
// part is quoted from the record, not re-verified here.)
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
// Source of truth is the `switch req.OpType` inside workflowOpRouter, not the
// constant block — a constant with no `case` falls through to `default:`
// (400 unknown_op_type) and opens no door.
//
// WHAT THIS PARSER ACTUALLY GUARANTEES (stated precisely, because an
// overstated loudness claim is the same failure class as a stale exception —
// a lie about the system that reassures instead of warning):
//
//   Handled correctly, each verified by mutation:
//     • trailing comments after the colon — `case opsync.OpX: // TODO`
//     • an inline statement after the colon — `case opsync.OpX: return nil, nil`
//     • grouped labels — `case opsync.OpA, opsync.OpB:`
//     • labels spanning multiple lines
//     • `case`/`switch` keywords appearing inside comments or string literals
//       (both are stripped before scanning)
//   Loud failure, never silent:
//     • router function or its switch not found
//     • a case label that is not `opsync.OpSomething`
//     • a case naming a constant with no string value
//     • fewer than MIN_ROUTED_OPS ops derived (the floor below)
//
//   RESIDUAL BRITTLENESS — named, not papered over:
//     • This is a SOURCE parse of one switch statement. If dispatch ever moves
//       to a map, a table, a second router, or a helper called from `default:`,
//       those ops would be invisible here and this guard would go quietly
//       green. The floor catches narrowing, not relocation.
//     • A PERMISSIVE `default:`. The parse asserts `default:` exists; it cannot
//       tell whether it refuses. G6 changed it to `return nil, nil` — every
//       unknown op type then accepted and broadcast via InsertOpAndNotify —
//       and the equality test still went green, because the derived and covered
//       sets were unchanged. This one is no longer merely assumed: the
//       DEFAULT-REFUSES test below posts a junk op_type at runtime and requires
//       a 4xx. Source parsing could not have caught it; the HTTP leg can.
//   Anyone changing HOW the router dispatches must revisit this function — a
//   comment cannot enforce that, so it is called out in DECISIONS-NEEDED §1
//   as well.

// The router handles six ops today. The floor exists so that a future parser
// regression which silently NARROWS the derived set trips regardless of cause,
// rather than shrinking toward a vacuous pass. Raise it when ops are added;
// never lower it to make a red go away.
const MIN_ROUTED_OPS = 6;

// stripGoCommentsAndStrings blanks comments and string/rune literals so that a
// `case` inside either cannot be mistaken for a real branch, and so brace
// matching cannot be thrown off by a brace inside a literal. Offsets are not
// preserved; only structure is.
function stripGoCommentsAndStrings(src) {
  let out = '';
  let i = 0;
  while (i < src.length) {
    const two = src.slice(i, i + 2);
    if (two === '//') {
      const nl = src.indexOf('\n', i);
      if (nl < 0) { out += '\n'; break; }
      out += '\n';
      i = nl + 1;
      continue;
    }
    if (two === '/*') {
      const end = src.indexOf('*/', i + 2);
      if (end < 0) { out += ' '; break; }
      // Preserve newlines so line-oriented error messages stay roughly honest.
      out += src.slice(i, end + 2).replace(/[^\n]/g, ' ');
      i = end + 2;
      continue;
    }
    const c = src[i];
    if (c === '"' || c === '`' || c === "'") {
      i++;
      while (i < src.length) {
        if (c !== '`' && src[i] === '\\') { i += 2; continue; }
        if (src[i] === c) { i++; break; }
        if (src[i] === '\n' && c === '`') { out += '\n'; }
        i++;
      }
      out += '_';
      continue;
    }
    out += c;
    i++;
  }
  return out;
}

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

  const clean = stripGoCommentsAndStrings(mainGo);

  const start = clean.indexOf('func workflowOpRouter(');
  if (start < 0) {
    throw new Error(`ops-authz-coverage: could not find func workflowOpRouter in ${MAIN_GO}. ` +
      'If the router was renamed or moved, update this parser — the coverage guard ' +
      'is worthless the moment it stops finding the router.');
  }

  // Scope to the op-type switch and brace-match its body, so a `case` belonging
  // to some unrelated switch added to this function later cannot leak in, and
  // so the scan cannot run off the end of the function.
  const swRel = clean.slice(start).search(/switch\s+req\.OpType\s*\{/);
  if (swRel < 0) {
    throw new Error('ops-authz-coverage: found workflowOpRouter but not its `switch req.OpType` — ' +
      'dispatch appears to have changed shape. Re-derive the op set before trusting this guard.');
  }
  const open = clean.indexOf('{', start + swRel);
  let depth = 0;
  let close = -1;
  for (let i = open; i < clean.length; i++) {
    if (clean[i] === '{') depth++;
    else if (clean[i] === '}') {
      depth--;
      if (depth === 0) { close = i; break; }
    }
  }
  if (close < 0) {
    throw new Error('ops-authz-coverage: could not brace-match the body of `switch req.OpType`.');
  }
  const body = clean.slice(open + 1, close);

  // Match `case` at a statement boundary through to the colon that terminates
  // the label — NOT to end-of-line. Anchoring to `$` made a trailing comment or
  // an inline statement invisible rather than an error: the op landed in
  // neither the routed nor the covered set, both diffs came back empty, and the
  // suite went green with a live uncovered door. Only `default:` may be skipped.
  const found = [];
  let sawDefault = false;
  for (const m of body.matchAll(/(?:^|[\n;{])\s*(case\s+([\s\S]*?)|default\s*)\s*:/g)) {
    if (m[2] === undefined) { sawDefault = true; continue; }
    for (const raw of m[2].split(',')) {
      const label = raw.trim();
      if (label === '') continue;
      const cm = /^opsync\.(Op[A-Za-z0-9_]*)$/.exec(label);
      if (!cm) {
        // A nested switch/select inside a case body is the likeliest cause: its
        // labels sit within the outer switch's braces and land here looking like
        // malformed op labels. Say so, rather than making the reader work it out.
        const nested = /\b(switch|select)\b/.test(body)
          ? ' NOTE: the router body contains a nested `switch`/`select`; its case labels are ' +
            'scanned too. If that is what this is, scope the scan to the outer switch\'s own ' +
            'cases before extending the label grammar.'
          : '';
        throw new Error(`ops-authz-coverage: unparsable case label "${label}" in workflowOpRouter. ` +
          'Every branch must be checkable — extend the parser rather than skipping it.' + nested);
      }
      const value = constMap[cm[1]];
      if (!value) {
        throw new Error(`ops-authz-coverage: case opsync.${cm[1]} has no string constant in ${OPS_GO}.`);
      }
      found.push(value);
    }
  }

  if (!sawDefault) {
    throw new Error('ops-authz-coverage: no `default:` branch found in `switch req.OpType`. ' +
      'The guard assumes unhandled op types are rejected there; if that changed, an ' +
      'unlisted op may no longer be a closed door.');
  }
  if (found.length < MIN_ROUTED_OPS) {
    throw new Error(`ops-authz-coverage: derived only ${found.length} op type(s) from the router switch, ` +
      `below the floor of ${MIN_ROUTED_OPS}. Either ops were genuinely removed (lower the floor ` +
      'deliberately, in the same commit) or this parser has silently narrowed. A shrinking derived ' +
      'set is exactly the vacuous pass this file exists to prevent.');
  }
  if (new Set(found).size !== found.length) {
    throw new Error(`ops-authz-coverage: the router switch lists a duplicate op type: ${found.join(', ')}.`);
  }
  return found.sort();
}

// ─── Helpers (duplicated per-file, matching this suite's convention) ─────────

// ── Card G1 baseline ─────────────────────────────────────────────────────────
// /workflow/* now sits behind the `operations` app grant
// (tests/grant-enforcement-parity.spec.js). The buildFixture team_member must
// remain "app-granted but unprivileged": this file's invariant is /ops ↔ REST
// PARITY for an authenticated caller who holds app access but no approver
// assignment or admin role. Without the grant every probe would collapse into
// the app gate's uniform 403 and the parity claims would go vacuous. Grant the
// app to the standard roles once up front, preserving user_grants.
test.beforeAll(async ({ browser }) => {
  const baseURL = process.env.NIGHTCREW_ENV_URL || 'http://localhost:' + (process.env.TEST_PORT || '8199');
  const page = await browser.newPage();
  await page.goto(baseURL + '/login.html');
  await page.fill('input[type="email"]', ADMIN_EMAIL);
  await page.fill('input[type="password"]', ADMIN_PASSWORD);
  await page.click('button.btn');
  await page.waitForURL(url => !url.pathname.includes('login'));
  await page.evaluate(async (slug) => {
    const perms = await (await fetch('/api/v1/apps/permissions')).json();
    const app = (perms || []).find(a => a.slug === slug) || {};
    const roles = [...new Set([...(app.role_grants || []), 'admin', 'manager', 'team_member'])];
    await fetch('/api/v1/apps/' + slug + '/permissions', {
      method: 'PUT', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role_grants: roles, user_grants: (app.user_grants || []).map(String) }),
    });
  }, 'operations');
  await page.close();
});

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

// NOTE on PARITY_OPEN, for whoever reads this next: both of these ops succeed
// against templates the caller is NOT assigned to. The REST twins behave
// identically, so the parity invariant this file guards genuinely holds and the
// ✅ is correct — neither op carries an attribution field, so the row is always
// written as the caller and there is no escalation here. But whether unassigned
// crew SHOULD be able to submit against an arbitrary template is a scope
// question nobody has asked. Filed as a backlog note in DECISIONS-NEEDED §1;
// it is explicitly NOT what these two tests assert.
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
  // OPEN DECISION: DECISIONS-NEEDED §1-B of run 2026-07-20c — "whether template
  // mutation needs any gate at all" is a PRODUCT question and has NOT been
  // answered. This entry is the living form of that question, not a waiver.
  //
  // UNGATED ≠ UNVALIDATED. Since fix/ops-savetemplate-approver-20260721, the
  // /ops SAVE_TEMPLATE path DOES enforce the requires_approval → hasApprover
  // rule its REST twins enforce (400 requires_approver), because the check now
  // lives inside insertTemplate/updateTemplate rather than in the handlers. That
  // is VALIDATION and changes nothing here: the probe below posts a VALID
  // template (minimalTemplate sets requires_approval:false), so a <400 from
  // /ops still means exactly what this entry claims — no privilege is required.
  // If you make this probe's payload approverless-with-approval, it will start
  // returning 400 and you will have made this exception unreadable: a 400 would
  // then be indistinguishable from the 403 the entry is watching for. The
  // validation contract has its own file, tests/ops-save-template-validation.spec.js,
  // whose NOT-A-GATE test asserts from the other side that this door is still open.
  SAVE_TEMPLATE: {
    decision: 'run 2026-07-20c DECISIONS-NEEDED §1-B — is template mutation meant to be crew-writable?',
    rest: (p, fx) => rest(p, 'POST', '/api/v1/workflow/createTemplate', minimalTemplate(`Cov Forge ${Date.now()}`)),
    ops: (p, fx) => op(p, 'SAVE_TEMPLATE', 'template', '', minimalTemplate(`Cov Forge Ops ${Date.now()}`)),
  },
  // ARCHIVE_TEMPLATE soft-deletes a template (archiveTemplate). Its REST twin
  // DELETE /workflow/archiveTemplate/{id} is admin-only (isAdmin, D-11). The
  // /ops branch carries no check, so any authenticated crew member can archive
  // any template by id — including one they cannot see.
  //
  // OPEN DECISION: same fork, DECISIONS-NEEDED §1-B.
  ARCHIVE_TEMPLATE: {
    decision: 'run 2026-07-20c DECISIONS-NEEDED §1-B — is template mutation meant to be crew-writable?',
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
  //
  // CAVEAT, deliberate: the submission sweep approves EVERY pending submission
  // in the shared database, not only the ones this file created — the same
  // shortcut `cleanupPendingApprovals` takes in workflows.spec.js, because the
  // server does not scope pendingApprovals by creator. That is harmless while
  // the suite runs serially (workers:1, the configured default) since no other
  // spec's submissions are in flight concurrently. It becomes real cross-file
  // interference under PW_WORKERS>1. If this suite is ever parallelised, give
  // each worker its own stack (TEST_DB_NAME + TEST_PORT, as playwright.config.js
  // documents) or narrow this sweep to submissions on this file's templates.
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

  // ── The closed-door assumption, enforced instead of assumed ──────────────
  //
  // Everything above rests on unlisted op types being REFUSED at `default:`.
  // The source parse can only see that a `default:` branch exists — G6 changed
  // its body to `return nil, nil`, making every unknown op type accepted and
  // broadcast, and COVERAGE-EQ stayed green because the derived and covered
  // sets were untouched. That is a silent-green this file must not have.
  test('DEFAULT-REFUSES: an unlisted op type is refused, not silently accepted', async ({ page }) => {
    await login(page);

    const status = await op(page, 'G6_NOT_A_REAL_OP_TYPE', 'submission', '', { probe: true });

    expect(status >= 400,
      `an unrecognised op_type returned ${status}. The router's default: branch must REFUSE ` +
      'unknown ops — every "this op is not in the router, therefore it is a closed door" ' +
      'conclusion in this file depends on it. A permissive default: makes the whole coverage ' +
      'set meaningless while every other test here stays green.'
    ).toBe(true);
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
