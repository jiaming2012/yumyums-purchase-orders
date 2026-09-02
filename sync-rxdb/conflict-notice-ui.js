// sync-rxdb/conflict-notice-ui.js — the RENDER half of the overwritten-answers
// UI: the banner, the sheet, and A-2's batch-override confirm.
//
// Card `sync-rxdb-conflict-notice-ui` (overnight-20260801, C2).
// Built against `.planning/phases/sync-rxdb-conflict-notice/mockup.html`
// **revision 2**, signed by the operator at morning triage 2026-07-29 (ledger
// T-28 decision 98), plus the three build obligations that came out of that
// walk: A-1 (both banner figures), A-2 (the override states what it destroys),
// A-3 (a removed question keeps its label, struck through and read-only).
//
// ===========================================================================
// STATE-FIRST, ONE LISTENER, NO INLINE HANDLERS — workflows.html's convention.
// ===========================================================================
// Mutate the store -> rebuild the model -> re-render from the model. Every
// control routes through ONE delegated click listener on the mount root via
// `data-action`, exactly as `workflows.html` does. Nothing here reads the DOM
// for state.
//
// ===========================================================================
// 🛑 WHAT THIS DOES NOT DO, AND WHY THAT IS NOT AN OMISSION.
// ===========================================================================
// `sync-hard-cutover` owns the write path. This card does NOT swap
// `debouncedSaveField` -> `submitOp('SET_FIELD')` -> POST /ops ->
// DRAFT_RESPONSES -> hydrateFieldState, and replication is not started
// (`HQ_SYNC_REST_URL` is unset everywhere, so the /sync door answers 503 by
// design). The open precondition is the CUTOVER, not row-visibility RLS: that
// card (`sync-rxdb-row-visibility-rls`) MERGED 2026-08-01, but no page starts
// replication yet, so setting the var today would start a replication nothing
// reads. `sync-hard-cutover` switches the producer on.
//
// (This comment named `autoSaveField` -> POST /saveResponse until B-65 — a
// function defined nowhere, and an endpoint no frontend code posts to.
// Corrected by card A2, run 20260804.)
//
// The consequence, stated plainly rather than discovered later: **in today's
// tree no conflict record is ever written, so this UI is mounted and dormant in
// production.** It renders from the durable local record; the record is written
// when `conflict$` fires; `conflict$` fires when replication runs. Every state
// below is therefore forced from a seeded store in
// `tests/states-sync-rxdb-conflict-notice.spec.js`, which is the honest way to
// verify a screen whose producer is not switched on yet.
//
// `applyRestore` is injected for the same reason. Today `workflows.html` hands
// it the LIVE persistence path (`POST /api/v1/workflow/saveResponse`), because
// that is what actually writes an answer in this tree — "Restore mine writes the
// crew member's value again, now, from the current master state" is an ordinary
// edit, and the ordinary edit today is that endpoint. The cutover card replaces
// the injected function with the RxDB local write and changes nothing else here.

import {
  buildSheetModel,
  bannerModel,
  RETENTION_DAYS,
  MAX_GROUPS,
  KIND_UNIDENTIFIED,
  KIND_REMOVED,
} from './conflict-notice.js';

// The sheet does not render at all under this; a flash of skeleton on a warm
// IndexedDB read is worse than nothing. UI-SPEC's `loading` row.
export const LOADING_AFTER_MS = 500;

function esc(s) {
  return String(s === null || s === undefined ? '' : s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// ---------------------------------------------------------------------------
// The banner. A-1: BOTH figures, always.
// ---------------------------------------------------------------------------

export function renderBanner(model) {
  const b = bannerModel(model);
  if (!b) return '';
  const open = typeof b.open === 'string'
    ? `<span class="cn-banner-open">${esc(b.open)}</span>`
    : `<span class="cn-banner-open">${esc(b.open.lead)} <span class="hnd">${esc(b.open.hand)}</span></span>`;
  return `<div class="cn-banner" data-testid="conflict-banner">
    <span class="cn-banner-ico">⚠️</span>
    <span class="cn-banner-body">
      <span class="cn-banner-hd">${esc(b.headline)}</span>
      ${open}
      ${b.unid ? `<span class="cn-banner-unid">${esc(b.unid)}</span>` : ''}
      <span class="cn-banner-sub">${esc(b.cause)}</span>
    </span>
    <span class="cn-banner-go" role="button" tabindex="0" data-action="cn-open">Review</span>
  </div>`;
}

// ---------------------------------------------------------------------------
// One field row. Eight renderings (UI-SPEC "Conflict field row").
// ---------------------------------------------------------------------------

function rowTitle(row) {
  if (row.kind === KIND_UNIDENTIFIED) {
    return `<div class="cf-q">${esc(row.label)}</div>`;
  }
  if (row.kind === KIND_REMOVED) {
    // 🛑 AMENDMENT A-3 (decision 95). The removed question keeps its LABEL,
    // struck through and read-only — the operator's words were "show the
    // deleted question crossed out and read only so that the user isnt
    // confused". It is in the SAME TYPE as any other question title, not
    // monospace and not a raw id: `template_snapshot` is json.Marshal of the
    // whole template and `Field.Label` is on it, so the discarded document
    // carries its own frozen label for a field the live template has dropped.
    //
    // The raw field id survives ONLY as the fallback A-3 names, for a snapshot
    // that genuinely carries no label for that id — which, because nothing
    // validates `template_snapshot` (B1's R-C), includes every malformed one.
    // Rendered exactly as r2 drew it.
    return row.labelSource === 'snapshot'
      ? `<div class="cf-q cf-q-struck" data-label-source="snapshot">${esc(row.label)}</div>`
      : `<div class="cf-q-gone" data-label-source="fallback">${esc(row.label)}</div>`;
  }
  return `<div class="cf-q">${esc(row.label)}</div>`;
}

function valuePair(row) {
  if (row.kind === KIND_UNIDENTIFIED) {
    return `<div class="cf-vals">
      <div class="cf-v none"><span class="cf-v-lab">Yours</span><span class="cf-v-val">Not recoverable</span></div>
    </div>`;
  }
  const theirs = row.kind === KIND_REMOVED
    ? ''
    : `<div class="cf-v"><span class="cf-v-lab">Now shows</span><span class="cf-v-val">${esc(row.theirs)}</span><span class="cf-v-who">· ${esc(row.who)}</span></div>`;
  return `<div class="cf-vals">
    <div class="cf-v mine"><span class="cf-v-lab">Yours</span><span class="cf-v-val">${esc(row.yours)}</span></div>
    ${theirs}
  </div>`;
}

/**
 * 🛑 G6 FINDING F-5. `label` used to be interpolated RAW, because exactly one
 * call site needed to prepend a spinner. Every call site passes a code literal,
 * so it was a latent hazard rather than a defect — but it is the only unescaped
 * hole in a module whose whole job is rendering values a crew member typed, and
 * a future label built from `row.*` would be a real one. The spinner is a flag
 * now; the label is escaped like everything else.
 */
function btn(action, id, label, sub, cls, disabled, spinner) {
  const dis = disabled ? ' aria-disabled="true"' : '';
  const act = disabled ? '' : ` data-action="${action}" data-id="${esc(id)}"`;
  const spin = spinner ? '<span class="spin"></span>' : '';
  return `<span class="cf-btn ${cls}" role="button" tabindex="0"${act}${dis}>
    <span>${spin}${esc(label)}</span>${sub ? `<span class="cf-btn-s">${esc(sub)}</span>` : ''}
  </span>`;
}

function rowActions(row, collapsed) {
  // Counting rule 8. Collapse hides the Restore/Keep PAIR and nothing else —
  // not an outcome strip, not an Undo, and not the actions of a row that has no
  // Restore/Keep pair to hide. A collapsed sheet that dropped the second kind
  // would leave the row with no exit at all, on the sheet with the most rows to
  // get through.
  if (row.kind === KIND_UNIDENTIFIED) {
    return `<div class="cf-acts">
      ${btn('cn-open-checklist', row.id, 'Open checklist', '', 'cf-btn-sec cf-btn-wide')}
      ${btn('cn-dismiss', row.id, 'Dismiss', '', 'cf-btn-sec cf-btn-wide')}
    </div>`;
  }
  if (row.kind === KIND_REMOVED) {
    return `<div class="cf-acts">
      ${btn('cn-copy', row.id, 'Copy value', '', 'cf-btn-pri cf-btn-wide')}
      ${btn('cn-dismiss', row.id, 'Dismiss', '', 'cf-btn-sec cf-btn-wide')}
    </div>`;
  }
  if (collapsed && !row.expanded) return '';

  // "replaces —" is not a sentence. When the server value is empty the loss is
  // still real and must still be named, so it is named in words — the mockup's
  // own `edge-longvalue` wording.
  const EMPTY = '—';
  const replaces = row.theirs === EMPTY ? 'replaces an empty answer' : `replaces ${row.theirs}`;
  const replacing = row.theirs === EMPTY ? 'replacing an empty answer' : `replacing ${row.theirs}`;
  const stays = row.theirs === EMPTY ? 'stays empty' : `${row.theirs} stays`;

  if (row.status === 'restoring') {
    return `<div class="cf-acts">
      ${btn('cn-restore', row.id, 'Restoring…', replacing, 'cf-btn-pri', true, true)}
      ${btn('cn-keep', row.id, 'Keep theirs', stays, 'cf-btn-sec', true)}
    </div>`;
  }
  // A-2.1 — the action names what it REPLACES, not only what it restores. That
  // holds for Retry too: it is the same destructive write, on the one plate
  // where the crew member has already failed once.
  const primary = row.status === 'failed' ? 'Retry' : 'Restore mine';
  return `<div class="cf-acts">
    ${btn('cn-restore', row.id, primary, replaces, 'cf-btn-pri')}
    ${btn('cn-keep', row.id, 'Keep theirs', stays, 'cf-btn-sec')}
  </div>`;
}

function undoBtn(id) {
  return `<span class="cf-done-undo" role="button" tabindex="0" data-action="cn-undo" data-id="${esc(id)}">Undo</span>`;
}

export function renderRow(row, collapsed) {
  const cls = 'cf';
  // `.unrec` is the mockup's class for a row with no Restore/Keep pair. Both
  // kinds that carry it are the ones counting rule 8's third clause protects.
  const unrec = row.kind === KIND_UNIDENTIFIED || row.kind === KIND_REMOVED ? ' unrec' : '';

  if (row.status === 'restored') {
    // Rows never leave on Restore — (b) STANDS. The row collapses to a green
    // confirmation that NAMES THE VALUE THAT CAME BACK and keeps an Undo, which
    // is the only escape from a mis-tap and is why handled rows are kept.
    return `<div class="${cls}${unrec}" data-row="${esc(row.id)}" data-state="restored">
      ${rowTitle(row)}
      <div class="cf-done">
        <span>✓ Restored — <b>${esc(row.yours)}</b> is back, replacing ${esc(row.whoName || 'their')}${row.whoName ? "'s" : ''} ${esc(row.theirs)}</span>
        ${undoBtn(row.id)}
      </div>
    </div>`;
  }
  if (row.status === 'kept') {
    return `<div class="${cls}${unrec}" data-row="${esc(row.id)}" data-state="kept">
      ${rowTitle(row)}
      <div class="cf-kept">
        <span>Kept theirs — the checklist reads <b>${esc(row.theirs)}</b>, ${esc(row.who)}</span>
        ${undoBtn(row.id)}
      </div>
    </div>`;
  }

  const why = row.kind === KIND_UNIDENTIFIED
    ? `<div class="cf-why">Something on this checklist changed on the server, but nothing you'd typed came back with it. Open the checklist to check it reads the way you left it.</div>`
    : row.kind === KIND_REMOVED
      ? `<div class="cf-why">This question was removed from the checklist while you were offline, so your answer can't go back on it. Copy it if you still need it.</div>`
      : '';

  // A failed restore keeps BOTH values on screen, says which of the two things
  // happened in the crew member's words, says WHERE the discarded value still
  // is (on this list, not in the checklist), and offers Retry. It promises no
  // automatic retry — nothing in this design commits to one.
  const err = row.status === 'failed'
    ? `<div class="cf-err">
        <b>Couldn't put ${esc(row.yours)} back — ${row.failure === 'conflict' ? 'someone changed it again' : "you're offline"}</b>
        Your ${esc(row.yours)} is still saved <em>on this list</em> — the checklist itself still reads ${esc(row.theirs)}.
        ${row.failure === 'conflict' ? 'Tap Retry to put yours back over the new value.' : 'Tap Retry when you have signal.'}
      </div>`
    : '';

  const undone = row.undone && row.status === 'open'
    ? `<div class="cf-undone">Undone — ${esc(row.whoName || 'their')}${row.whoName ? "'s" : ''} ${esc(row.theirs)} is showing again.</div>`
    : '';

  return `<div class="${cls}${unrec}" data-row="${esc(row.id)}" data-state="${esc(row.status)}"${collapsed ? ' data-action="cn-expand" data-id="' + esc(row.id) + '"' : ''}>
    ${rowTitle(row)}
    ${valuePair(row)}
    ${why}
    ${err}
    ${undone}
    ${rowActions(row, collapsed)}
  </div>`;
}

// ---------------------------------------------------------------------------
// A group = one document = one conflict$ event.
// ---------------------------------------------------------------------------

export function renderGroup(g, collapsed) {
  const chip = g.chipPlus > 0
    ? `${g.chipBase} answer${g.chipBase === 1 ? '' : 's'} +${g.chipPlus}`
    : `${g.chipBase} answer${g.chipBase === 1 ? '' : 's'}`;

  // A-2: the batch names what it REPLACES, carries the same attribution the
  // rows do, acts only on the rows still to review (counting rule 7), and OPENS
  // A CONFIRM rather than writing through.
  const batch = collapsed && g.batchCount > 0
    ? `<span class="cg-all" role="button" tabindex="0" data-action="cn-batch" data-id="${esc(g.docId)}">
        <span>Restore all ${g.batchCount} of mine</span>
        <span class="cg-all-s">the ${g.batchCount} still to review · replaces ${g.batchCount} of ${esc(g.batchWho)}'s answer${g.batchCount === 1 ? '' : 's'}${g.batchRange ? ` · ${esc(g.batchRange)}` : ''} · asks first</span>
      </span>`
    : '';

  return `<div class="cg" data-group="${esc(g.docId)}">
    <div class="cg-hd">
      <span><span class="cg-name">${esc(g.name)}</span><span class="cg-meta">${esc(g.day)} · <span class="cg-doc">${esc(g.docChip)}</span></span></span>
      <span class="cg-count">${esc(chip)}</span>
    </div>
    ${g.rows.map((r) => renderRow(r, collapsed)).join('')}
    ${batch}
  </div>`;
}

// ---------------------------------------------------------------------------
// The sheet.
// ---------------------------------------------------------------------------

const SHEET_HEAD = `<div class="sc-hd"><span class="sc-title">Overwritten answers</span><span class="sc-close" role="button" tabindex="0" data-action="cn-close">Done</span></div>`;

export function renderSheet(state) {
  // "Nothing renders at all under 500 ms." A flash of skeleton on a warm
  // IndexedDB read costs more than an empty frame.
  if (state.phase === 'booting') return '';

  if (state.phase === 'loading') {
    // No count in the header — it must not claim a number it does not have.
    return `<div class="sc-sheet" data-testid="conflict-sheet" data-phase="loading">${SHEET_HEAD}
      <div class="cn-sk"><div class="cn-sk-l" style="width:58%"></div><div class="cn-sk-l" style="width:34%;height:9px"></div><div class="cn-sk-l" style="width:76%;margin-top:14px"></div><div class="cn-sk-l" style="width:66%"></div></div>
      <div class="cn-sk"><div class="cn-sk-l" style="width:47%"></div><div class="cn-sk-l" style="width:30%;height:9px"></div><div class="cn-sk-l" style="width:72%;margin-top:14px"></div></div>
    </div>`;
  }

  if (state.phase === 'unreadable') {
    // Both halves, BAD ONE FIRST, and half (a) at full text contrast. This is
    // the one screen in the set where something really is unrecoverable — an
    // evicted IndexedDB is not recovered by tapping a button — and it is the one
    // that must say so. A crew member must not read a storage failure as "my
    // work was deleted", and must not read it as "nothing was lost" either.
    return `<div class="sc-sheet" data-testid="conflict-sheet" data-phase="unreadable">${SHEET_HEAD}
      <div class="sc-err">
        <h3>Couldn't open the list</h3>
        <p>Your phone cleared this list to free up space. <b>If Try again doesn't bring it back, it's gone — we can't tell you what was overwritten, and any answers on it can't be put back.</b></p>
        <p>Your checklists are not affected. They're on the server and they're fine — this is only the record of what got overwritten.</p>
        <button type="button" data-action="cn-reload">Try again</button>
      </div>
    </div>`;
  }

  const model = state.model;
  if (!model || model.status === 'empty') {
    // Scoped to the RECORD, never phrased as a guarantee: a non-leader tab, a
    // replication with nothing subscribed to conflict$, and an evicted local
    // store all produce this identical screen, so a flat "Nothing was
    // overwritten" would be a claim the app is not in a position to make — on
    // the screen it shows most often.
    //
    // Decision 96: the retention figure comes from the ONE named constant. No
    // surface restates the literal, including this one.
    const days = (model && model.retentionDays) || RETENTION_DAYS;
    return `<div class="sc-sheet" data-testid="conflict-sheet" data-phase="empty">${SHEET_HEAD}
      <div class="sc-empty">
        <h3>Nothing recorded in the last <span data-testid="retention-days">${esc(days)}</span> days</h3>
        <p>Answers land here when someone edits the same question while your phone is offline and the server's version wins. This is what HQ caught and kept — if it wasn't running when a change came in, that change won't be listed.</p>
      </div>
    </div>`;
  }

  const collapsed = model.collapsed;
  const groups = model.groups.map((g) => renderGroup(g, collapsed)).join('');

  // Decision 97 — the cap. Rows below the line are NOT dropped: they stay in
  // the store, they stay in the banner's totals, and this line says how many.
  const more = model.hiddenGroups > 0
    ? `<p class="cn-more" data-testid="cap-line">and ${model.hiddenGroups} more checklist${model.hiddenGroups === 1 ? '' : 's'} · ${model.hiddenAnswers} more row${model.hiddenAnswers === 1 ? '' : 's'} — the count above is the true total</p>`
    : '';

  const caption = collapsed
    ? `<p class="cn-more">Individual Restore / Keep buttons are collapsed. Tap a row to expand it. <b>Rows you have already handled keep their Undo.</b> <b>A row with nothing to restore keeps its Open checklist and Dismiss</b> — collapse hides the Restore/Keep pair, and those rows have no such pair to hide.</p>`
    : '';

  return `<div class="sc-sheet" data-testid="conflict-sheet" data-phase="ready" data-collapsed="${collapsed}">${SHEET_HEAD}
    ${groups}
    ${more}
    ${caption}
  </div>`;
}

// ---------------------------------------------------------------------------
// A-2 — the batch override confirm. The write does NOT go through on the tap.
// ---------------------------------------------------------------------------

export function renderConfirm(group) {
  const targets = group.rows.filter((r) => r.isBatchTarget);
  const n = targets.length;
  const who = group.batchWho || 'someone else';
  return `<div class="cfm-scrim" data-testid="conflict-confirm">
    <div class="cfm">
      <div class="cfm-hd">Replace ${n} of ${esc(who)}'s answer${n === 1 ? '' : 's'}?</div>
      <p class="cfm-lead">This puts your ${n} answer${n === 1 ? '' : 's'} back on <b>${esc(group.name)}</b> and overwrites what ${esc(who)} saved. <b>Their values are what the checklist reads right now</b> — ${n === 1 ? 'this one' : `these ${n}`} will not be what anyone sees after you tap Replace.</p>
      <div class="cfm-list">
        ${targets.map((r) => `<div class="cfm-row">
          <div class="cfm-q">${esc(r.label)}</div>
          <div class="cf-v gone"><span class="cf-v-lab">Replaces</span><span class="cf-v-val">${esc(r.theirs)}</span><span class="cf-v-who">· ${esc(r.who)}</span></div>
          <div class="cf-v mine"><span class="cf-v-lab">With yours</span><span class="cf-v-val">${esc(r.yours)}</span></div>
        </div>`).join('')}
      </div>
      <p class="cfm-foot">Each row keeps an <b>Undo</b> afterwards — but that is ${n} tap${n === 1 ? '' : 's'} to reverse one. Nothing else on the checklist changes.</p>
      <div class="cfm-acts">
        <span class="cfm-cancel" role="button" tabindex="0" data-action="cn-cancel">Cancel</span>
        <span class="cfm-go" role="button" tabindex="0" data-action="cn-commit" data-id="${esc(group.docId)}">Replace ${n} answer${n === 1 ? '' : 's'}<span class="cfm-go-s">replaces ${n} of ${esc(who)}'s</span></span>
      </div>
    </div>
  </div>`;
}

// ---------------------------------------------------------------------------
// Mount. One delegated listener, state-first rendering.
// ---------------------------------------------------------------------------

/**
 * @param {object} o
 * @param {Element} o.bannerRoot  where the banner lives (top of My Checklists)
 * @param {Element} o.sheetRoot   full-height host for the sheet + confirm
 * @param {object}  o.store       the durable conflict record store
 * @param {function} [o.applyRestore] `(row) => Promise` — writes the crew
 *        member's value again, now. Injected: see the header.
 * @param {function} [o.openChecklist] `(row) => void`
 * @param {function} [o.copyValue] `(text) => Promise`
 * @param {object}  [o.modelOpts] passed to buildSheetModel (now, snapshots, …)
 */
export function mountConflictNotice(o) {
  const state = {
    phase: 'loading',
    open: false,
    confirmFor: null,
    records: [],
    expanded: new Set(),
    model: null,
  };

  const rebuild = () => {
    state.model = buildSheetModel(state.records, o.modelOpts || {});
    for (const g of state.model.groups) {
      for (const r of g.rows) r.expanded = state.expanded.has(r.id);
    }
  };

  const paint = () => {
    o.bannerRoot.innerHTML = state.phase === 'ready' ? renderBanner(state.model) : '';
    if (!state.open) {
      o.sheetRoot.innerHTML = '';
      o.sheetRoot.style.display = 'none';
      return;
    }
    o.sheetRoot.style.display = '';
    let html = renderSheet(state);
    if (state.confirmFor && state.model) {
      const g = state.model.groups.find((x) => x.docId === state.confirmFor);
      if (g) html += renderConfirm(g);
    }
    o.sheetRoot.innerHTML = html;
  };

  const load = async () => {
    const slow = setTimeout(() => {
      if (state.phase === 'booting') {
        state.phase = 'loading';
        paint();
      }
    }, LOADING_AFTER_MS);
    state.phase = 'booting';
    try {
      state.records = await o.store.all();
      state.phase = 'ready';
    } catch (err) {
      // The store is unreadable — a DESIGNED state (iOS eviction, private
      // browsing), not a stack trace, and W3 named it the largest untested
      // unknown for a phone-first PWA.
      state.phase = 'unreadable';
    }
    clearTimeout(slow);
    rebuild();
    paint();
  };

  const patch = async (id, fields) => {
    await o.store.patch(id, fields);
    state.records = await o.store.all();
    rebuild();
    paint();
  };

  const restore = async (id) => {
    const row = findRow(state.model, id);
    if (!row) return;
    await patch(id, { status: 'restoring', failure: null });
    try {
      if (o.applyRestore) await o.applyRestore(row);
      await patch(id, { status: 'restored', undone: false, failure: null });
    } catch (err) {
      // A failed restore is UNFINISHED BUSINESS, not a completed one: it stays
      // still-to-review (counting rule 6) and it must never look handled.
      await patch(id, {
        status: 'failed',
        failure: err && err.name === 'ConflictError' ? 'conflict' : 'offline',
      });
    }
  };

  const onClick = async (ev) => {
    const el = ev.target.closest('[data-action]');
    if (!el) return;
    const action = el.getAttribute('data-action');
    const id = el.getAttribute('data-id');
    if (el.getAttribute('aria-disabled') === 'true') return;
    ev.preventDefault();

    if (action === 'cn-open') { state.open = true; paint(); return; }
    if (action === 'cn-close') { state.open = false; state.confirmFor = null; paint(); return; }
    if (action === 'cn-reload') { await load(); return; }
    if (action === 'cn-expand') { state.expanded.add(id); rebuild(); paint(); return; }
    if (action === 'cn-restore') { await restore(id); return; }
    if (action === 'cn-keep') { await patch(id, { status: 'kept', undone: false }); return; }
    // Undo returns the row to untouched with a muted line saying what happened,
    // so the tap is not silent. It is the ONLY escape from a mis-tap and is why
    // handled rows stay on the sheet: a removed row cannot be undone.
    if (action === 'cn-undo') { await patch(id, { status: 'open', undone: true }); return; }
    if (action === 'cn-dismiss') {
      // The only way a record ever LEAVES the sheet, other than expiry.
      await o.store.remove(id);
      state.records = await o.store.all();
      rebuild(); paint();
      return;
    }
    if (action === 'cn-copy') {
      const row = findRow(state.model, id);
      if (row && o.copyValue) await o.copyValue(String(row.yours));
      return;
    }
    if (action === 'cn-open-checklist') {
      const row = findRow(state.model, id);
      if (row && o.openChecklist) o.openChecklist(row);
      return;
    }
    if (action === 'cn-batch') { state.confirmFor = id; paint(); return; }
    if (action === 'cn-cancel') { state.confirmFor = null; paint(); return; }
    if (action === 'cn-commit') {
      const g = state.model.groups.find((x) => x.docId === id);
      state.confirmFor = null;
      paint();
      if (!g) return;
      for (const rowId of g.batchIds) await restore(rowId);
      return;
    }
  };

  o.bannerRoot.addEventListener('click', onClick);
  o.sheetRoot.addEventListener('click', onClick);

  const unsub = o.store.subscribe
    ? o.store.subscribe((rows) => { state.records = rows; rebuild(); paint(); })
    : null;

  load();

  return {
    state,
    reload: load,
    open() { state.open = true; paint(); },
    close() { state.open = false; state.confirmFor = null; paint(); },
    destroy() {
      if (unsub) unsub();
      o.bannerRoot.removeEventListener('click', onClick);
      o.sheetRoot.removeEventListener('click', onClick);
    },
  };
}

function findRow(model, id) {
  if (!model) return null;
  for (const g of model.groups) {
    for (const r of g.rows) if (r.id === id) return r;
  }
  return null;
}

export { RETENTION_DAYS, MAX_GROUPS };
