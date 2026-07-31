// ─── sync.js — Shared Reactive Sync Module ───────────────────────────────────
// Loaded via <script src="sync.js"> before page-specific scripts.
// Exposes all sync primitives on window.* (no ES modules — same pattern as ptr.js).
// Plan 01 of Phase 10.2: extract from workflows.html without modifying it.

// Safe UUID v4 generator — crypto.randomUUID() requires HTTPS (secure context).
// On HTTP origins (e.g. Tailscale dev), falls back to Math.random-based UUID.
function generateUUID() {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) return crypto.randomUUID();
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) { var r = Math.random() * 16 | 0; return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16); });
}
window.generateUUID = generateUUID;

var _fallbackDeviceId = generateUUID();

// ─── API Wrapper ─────────────────────────────────────────────────────────────

async function api(method, path, body) {
  const opts = { method, credentials: 'include', headers: {} };
  if (body !== null && body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch('/api/v1/workflow/' + path, opts);
  if (res.status === 401) { location.href = 'login.html'; throw new Error('unauthorized'); }
  if (res.status === 409) {
    const winner = await res.json().catch(() => ({}));
    return { _conflict: true, winner };
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'unknown' }));
    err.status = res.status;
    throw err;
  }
  if (res.status === 204) return null;
  return res.json();
}

window.api = api;

// ─── IndexedDB Offline Queue ──────────────────────────────────────────────────

const HQ_DB = 'hq_offline_v1';
let _dbPromise = null;

function getDB() {
  if (!_dbPromise) _dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(HQ_DB, 2);
    req.onupgradeneeded = e => {
      const db = e.target.result;
      if (!db.objectStoreNames.contains('submitQueue')) {
        db.createObjectStore('submitQueue', { keyPath: 'id' });
      }
      if (!db.objectStoreNames.contains('syncMeta')) {
        db.createObjectStore('syncMeta', { keyPath: 'id' });
      }
    };
    req.onsuccess = e => resolve(e.target.result);
    req.onerror = e => reject(e.target.error);
  });
  return _dbPromise;
}

function idbGetAll(db, store) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(store, 'readonly');
    const req = tx.objectStore(store).getAll();
    req.onsuccess = () => resolve(req.result);
    req.onerror = e => reject(e.target.error);
  });
}

function idbGet(db, store, key) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(store, 'readonly');
    const req = tx.objectStore(store).get(key);
    req.onsuccess = () => resolve(req.result || null);
    req.onerror = () => reject(req.error);
  });
}

function idbPut(db, store, item) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(store, 'readwrite');
    tx.objectStore(store).put(item);
    tx.oncomplete = resolve;
    tx.onerror = e => reject(e.target.error);
  });
}

function idbDelete(db, store, key) {
  return new Promise((resolve, reject) => {
    const tx = db.transaction(store, 'readwrite');
    tx.objectStore(store).delete(key);
    tx.oncomplete = resolve;
    tx.onerror = e => reject(e.target.error);
  });
}

window.getDB = getDB;
window.idbGetAll = idbGetAll;
window.idbGet = idbGet;
window.idbPut = idbPut;
window.idbDelete = idbDelete;

// ─── LamportClock ─────────────────────────────────────────────────────────────

class LamportClock {
  constructor(db) { this._db = db; this._ts = 0; this._deviceId = null; }

  static async init(db) {
    const clock = new LamportClock(db);
    // device_id is per-tab (not shared via IndexedDB) so two tabs on the same
    // origin don't suppress each other's ops as self-echoes.
    clock._deviceId = generateUUID();
    const meta = await idbGet(db, 'syncMeta', 'clock');
    if (meta) {
      clock._ts = meta.lamport_ts;
    }
    await idbPut(db, 'syncMeta', { id: 'clock', lamport_ts: clock._ts, device_id: clock._deviceId });
    return clock;
  }

  async tick() {
    this._ts += 1;
    await idbPut(this._db, 'syncMeta', { id: 'clock', lamport_ts: this._ts, device_id: this._deviceId });
    return this._ts;
  }

  async receive(remoteTs) {
    this._ts = Math.max(this._ts, remoteTs) + 1;
    await idbPut(this._db, 'syncMeta', { id: 'clock', lamport_ts: this._ts, device_id: this._deviceId });
  }

  get ts() { return this._ts; }
  get deviceId() { return this._deviceId; }
}

window.LamportClock = LamportClock;
window.LAMPORT_CLOCK = null;

// ─── Store ────────────────────────────────────────────────────────────────────
// Reactive collection store. Pages register collections and subscribe to changes.
// Per D-05: pure store, no computed/derived values. Derivatives are plain functions.
// Per D-06: API surface — register, get, set, setAll, delete, on, off, batch.
// Per D-07: batch groups mutations and fires subscribers once after fn completes.

class Store {
  constructor() {
    this._collections = {};
    this._subscribers = {};
    this._batching = false;
    this._dirty = new Set();
  }

  register(defs) {
    for (const [key, initial] of Object.entries(defs)) {
      this._collections[key] = initial;
      this._subscribers[key] = [];
    }
  }

  get(key, subKey) {
    const col = this._collections[key];
    if (subKey !== undefined) {
      if (col && typeof col === 'object' && !Array.isArray(col)) return col[subKey];
      return undefined;
    }
    return col;
  }

  set(key, subKey, value) {
    if (!this._collections.hasOwnProperty(key)) return;
    const col = this._collections[key];
    if (typeof col === 'object' && !Array.isArray(col)) {
      col[subKey] = value;
    }
    this._notify(key);
  }

  setAll(key, value) {
    if (!this._collections.hasOwnProperty(key)) return;
    this._collections[key] = value;
    this._notify(key);
  }

  delete(key, subKey) {
    if (!this._collections.hasOwnProperty(key)) return;
    const col = this._collections[key];
    if (typeof col === 'object' && !Array.isArray(col)) {
      delete col[subKey];
    }
    this._notify(key);
  }

  on(key, callback) {
    if (!this._subscribers[key]) this._subscribers[key] = [];
    this._subscribers[key].push(callback);
    return () => this.off(key, callback);
  }

  off(key, callback) {
    const subs = this._subscribers[key];
    if (!subs) return;
    const idx = subs.indexOf(callback);
    if (idx !== -1) subs.splice(idx, 1);
  }

  batch(fn) {
    this._batching = true;
    this._dirty.clear();
    try { fn(); } finally {
      this._batching = false;
      for (const key of this._dirty) {
        this._fire(key);
      }
      this._dirty.clear();
    }
  }

  _notify(key) {
    if (this._batching) { this._dirty.add(key); return; }
    this._fire(key);
  }

  _fire(key) {
    const subs = this._subscribers[key];
    if (!subs) return;
    const value = this._collections[key];
    for (const cb of subs.slice()) { // slice() for safe iteration if cb removes itself
      try { cb(value, key); } catch(e) { console.error('Store subscriber error:', e); }
    }
  }
}

window.Store = Store;
window.store = new Store();

// Store API usage (called by page scripts after sync.js loads):
//   store.register({ fieldResponses: {}, myChecklists: [] })
//   store.set('fieldResponses', fieldId, value)
//   store.setAll('myChecklists', items)
//   store.get('fieldResponses')
//   store.get('fieldResponses', fieldId)
//   store.delete('fieldResponses', fieldId)
//   store.on('fieldResponses', callback)
//   store.off('fieldResponses', callback)
//   store.batch(() => { store.setAll(...); store.setAll(...); })

// ─── WebSocket Sync Client ────────────────────────────────────────────────────

const WS_MAX_BACKOFF = 30000;
const WS_MAX_ATTEMPTS = 15;
let _ws = null, _wsBackoff = 500, _wsAttempts = 0;

function wsConnect() {
  if (_wsAttempts >= WS_MAX_ATTEMPTS) return;
  const proto = location.protocol === 'https:' ? 'wss://' : 'ws://';
  _ws = new WebSocket(proto + location.host + '/ws');

  _ws.onopen = async () => {
    _wsBackoff = 500;
    _wsAttempts = 0;
    // Per D-15: drain offline queue BEFORE catch-up to avoid stale state
    try {
      await drainQueue();
    } catch(e) { console.error('drain error on ws open:', e); }
    wsCatchUp();
  };

  _ws.onmessage = (evt) => {
    try {
      const op = JSON.parse(evt.data);
      if (LAMPORT_CLOCK) LAMPORT_CLOCK.receive(op.lamport_ts);
      applyOp(op);
    } catch(e) { console.error('ws msg parse error:', e); }
  };

  _ws.onclose = () => {
    const jitter = Math.random() * 1000;
    const delay = Math.min(_wsBackoff + jitter, WS_MAX_BACKOFF);
    _wsBackoff = Math.min(_wsBackoff * 2, WS_MAX_BACKOFF);
    _wsAttempts++;
    setTimeout(wsConnect, delay);
  };
}

document.addEventListener('visibilitychange', async () => {
  if (document.visibilityState === 'visible') {
    if (!_ws || _ws.readyState === WebSocket.CLOSED || _ws.readyState === WebSocket.CLOSING) {
      _wsAttempts = 0;
      wsConnect(); // onopen will drain then catch up
    } else if (_ws.readyState === WebSocket.OPEN) {
      // Already connected — drain then catch up
      try {
        await drainQueue();
      } catch(e) { console.error('drain error on visibility:', e); }
      wsCatchUp();
    }
  }
});

async function wsCatchUp() {
  if (!LAMPORT_CLOCK) return;
  try {
    const ts = LAMPORT_CLOCK.ts;
    const ops = await api('GET', 'ops/since?lamport_ts=' + ts, null);
    if (!Array.isArray(ops)) return;
    ops.sort((a, b) => a.lamport_ts - b.lamport_ts);
    for (const op of ops) {
      await LAMPORT_CLOCK.receive(op.lamport_ts);
      applyOp(op, true); // silent: bulk historical replay, not live teammate edits
    }
  } catch(e) { console.error('catch-up error:', e); }
}

window.wsConnect = wsConnect;

// ─── Sync UX Functions ────────────────────────────────────────────────────────

function flashField(fieldId) {
  const row = document.querySelector('[data-field-id="' + fieldId + '"]');
  if (!row) return;
  row.classList.remove('sync-flash');
  void row.offsetWidth; // force reflow to restart animation
  row.classList.add('sync-flash');
  setTimeout(() => row.classList.remove('sync-flash'), 2000);
}

let _toastQueue = {}, _toastTimer = null;

function enqueueSyncToast(userId, userName, fieldIds) {
  const key = userId || '_conflict';
  if (!_toastQueue[key]) _toastQueue[key] = { name: userName, count: 0 };
  _toastQueue[key].count += fieldIds.length;
  clearTimeout(_toastTimer);
  _toastTimer = setTimeout(flushSyncToast, 500);
}

function flushSyncToast() {
  const entries = Object.values(_toastQueue);
  _toastQueue = {};
  if (!entries.length) return;
  // Only show sync toast when inside a checklist detail view
  if (typeof fillState !== 'undefined' && !fillState.activeTemplate) return;
  const msg = entries.map(e => e.count + ' field' + (e.count > 1 ? 's' : '') + ' updated by ' + e.name).join(', ');
  showSyncToast(msg);
}

function showSyncToast(msg) {
  let el = document.getElementById('sync-toast');
  if (!el) {
    el = document.createElement('div');
    el.id = 'sync-toast';
    el.className = 'sync-toast';
    document.body.appendChild(el);
  }
  el.textContent = msg;
  el.classList.add('show');
  setTimeout(() => el.classList.remove('show'), 3000);
}

window.flashField = flashField;
window.enqueueSyncToast = enqueueSyncToast;
window.showSyncToast = showSyncToast;

// ─── renderFieldResponse ──────────────────────────────────────────────────────
// References fillState and renderRunnerField which are globals in workflows.html.
// Stays here so applyOp can call it from sync.js.

function renderFieldResponse(fieldId) {
  const el = document.querySelector('[data-field-id="' + fieldId + '"]');
  if (!el) return; // field not visible in current view
  // Find the field definition from the current checklist's sections
  let fld = null;
  if (typeof fillState !== 'undefined' && fillState.activeTemplate && fillState.activeTemplate.sections) {
    for (const sec of fillState.activeTemplate.sections) {
      fld = (sec.fields || []).find(f => f.id === fieldId);
      if (fld) break;
    }
  }
  if (!fld) return;
  // Re-render the field row using the existing renderRunnerField function
  const tmp = document.createElement('div');
  tmp.innerHTML = (typeof renderRunnerField === 'function') ? renderRunnerField(fld) : '';
  const newEl = tmp.firstElementChild;
  if (newEl) el.replaceWith(newEl);
}

window.renderFieldResponse = renderFieldResponse;

// ─── applyOp ─────────────────────────────────────────────────────────────────
// Applies an incoming op from the server. References globals from workflows.html
// (FIELD_RESPONSES, DRAFT_RESPONSES, FAIL_NOTES, fillState, etc.).
// In Plan 02 these become store references. Self-echo via device_id (Plan 03).

// `silent` suppresses the "updated by" toast + field flash. Catch-up replays the
// entire historical op backlog on (re)connect; those are not live teammate edits
// (and after a reload the device_id is regenerated, so the user's own past ops no
// longer self-echo-suppress). Applying them silently avoids a toast flood.
function applyOp(op, silent) {
  // Skip self-originated ops (already applied optimistically)
  if (typeof LAMPORT_CLOCK !== 'undefined' && LAMPORT_CLOCK && op.device_id === LAMPORT_CLOCK.deviceId) return;

  if (op.op_type === 'SET_FIELD') {
    const { field_id, value, user_name } = op.payload;
    const displayName = user_name || 'Someone';
    if (value === null || value === undefined) {
      // Uncheck — remove from state via store
      store.delete('fieldResponses', field_id);
      var drafts = store.get('draftResponses');
      if (Array.isArray(drafts)) {
        const draftIdx = drafts.findIndex(d => d.field_id === field_id);
        if (draftIdx !== -1) { drafts.splice(draftIdx, 1); store._notify('draftResponses'); }
      }
    } else {
      const entry = { answeredBy: displayName, answeredAt: new Date(op.server_ts) };
      if (typeof value === 'object' && value !== null && value.value !== undefined) {
        entry.value = value.value;
        if (value.sub_steps) entry.sub_steps = value.sub_steps;
      } else {
        entry.value = value;
      }
      store.set('fieldResponses', field_id, entry);
      var drafts2 = store.get('draftResponses');
      if (Array.isArray(drafts2)) {
        const existing = drafts2.find(d => d.field_id === field_id);
        if (existing) { existing.value = value; existing.answered_at = op.server_ts; }
        else drafts2.push({ field_id, value, answered_at: op.server_ts });
        store._notify('draftResponses');
      }
    }
    renderFieldResponse(field_id);
    if (!silent) {
      flashField(field_id);
      enqueueSyncToast(op.user_id, displayName, [field_id]);
    }
    // Update progress: runner bar if inside checklist, list page if on checklist list
    if (typeof fillState !== 'undefined' && fillState.activeTemplate) {
      if (typeof updateProgress === 'function') updateProgress();
    } else if (typeof renderMyChecklists === 'function') { renderMyChecklists(); }
  } else if (op.op_type === 'SUBMIT_CHECKLIST') {
    // Gate the re-fetch exactly like the APPROVE_ITEM / SAVE_TEMPLATE branches
    // below: reconcile when a runner is open (flip the open checklist live to
    // its submitted/readonly shape) or for a genuinely LIVE op (converge the
    // list's progress + Pending Approval badge). SKIP a silent catch-up replay
    // with no runner open.
    //
    // Ungated, this was a fetch storm (T-18, root-caused 2026-07-21): a fresh
    // context starts at Lamport 0, so wsCatchUp replays the ENTIRE historical
    // ops journal, and every replayed SUBMIT fired a full myChecklists
    // re-fetch. Beyond the wasted requests, a stale snapshot landing mid-fill
    // clobbers an optimistic checkbox. The page-load's own loadMyChecklists has
    // already reconciled the list, so the replayed fetches add nothing.
    if (typeof loadMyChecklists === 'function' &&
        ((typeof fillState !== 'undefined' && fillState.activeTemplate) || !silent)) {
      loadMyChecklists(); // re-fetches data + re-renders runner if open
    }
  } else if (op.op_type === 'APPROVE_ITEM' || op.op_type === 'REJECT_ITEM') {
    // The approvers' queue always refreshes.
    if (typeof loadPendingApprovals === 'function') loadPendingApprovals();
    // The submitter's / observer's OWN checklist view must ALSO reconcile. An
    // approve/reject changes the submission's status, and that status drives
    // three things the old (approvals-only) refresh left stale on every
    // non-approver device until a hard reload (operator-found 2026-07-18):
    //   • the ⚠ Rejected correction banner (hydrateFieldState only builds
    //     REJECTION_FLAGS from a submission whose status is 'rejected'),
    //   • edit-vs-readonly mode (renderRunner derives fillState.readonly from
    //     the submission status), and
    //   • the My-Checklists list progress count (getProgress counts the frozen
    //     submission snapshot while status is pending/submitted/approved, so it
    //     never moves off the pre-rejection number until the status refreshes).
    // loadMyChecklists re-fetches MY_SUBMISSIONS, re-hydrates field/rejection
    // state, re-renders the list, AND re-renders an open runner in place — so
    // both symptoms converge live. Gate it like the SAVE_TEMPLATE branch below:
    // reconcile when a runner is open (flip the open checklist live) or for a
    // live op (converge the list); skip a silent catch-up replay with no runner
    // open, since the page-load's own loadMyChecklists already reconciled it and
    // replaying the backlog per-op would be a needless fetch storm.
    if (typeof loadMyChecklists === 'function' &&
        ((typeof fillState !== 'undefined' && fillState.activeTemplate) || !silent)) {
      loadMyChecklists();
    }
  } else if (op.op_type === 'SAVE_TEMPLATE' || op.op_type === 'ARCHIVE_TEMPLATE') {
    // The Builder template list always refreshes.
    if (typeof loadTemplates === 'function') loadTemplates();
    // If an unsubmitted checklist is open in the runner, re-fetch + re-render it
    // to the template's new shape (FR-4, INV-3). Bulk catch-up replay is silent
    // (INV-6, the 42eeb39 no-toast rule); a genuinely live teammate edit may
    // surface a toast.
    if (typeof fillState !== 'undefined' && fillState.activeTemplate) {
      rerenderOpenChecklistAfterSave(op, silent);
    } else if (!silent && typeof loadMyChecklists === 'function') {
      // No runner open, but the observer may be sitting on the My Checklists list:
      // a LIVE edit that adds/cuts a field changes the list-row progress
      // DENOMINATOR, so re-fetch today's list to converge it (mirrors the
      // SET_FIELD branch's no-runner refresh — SET_FIELD only needs a local
      // re-render because it already mutated state, whereas a structural edit
      // must re-fetch the template's new shape). Gated to LIVE ops (!silent): a
      // catch-up/reconnect replay must NOT fire a fetch per SAVE_TEMPLATE op —
      // the page-load's own loadMyChecklists already reconciled the list, so
      // replaying the backlog with per-op fetches would be a needless fetch storm
      // that blocks the main thread.
      loadMyChecklists();
    }
  }
}

// rerenderOpenChecklistAfterSave handles a SAVE_TEMPLATE/ARCHIVE_TEMPLATE op that
// lands while an unsubmitted checklist is open in the runner. It re-fetches
// myChecklists and re-renders the open runner to the template's new shape.
// Because hydrateFieldState keys by the stable checklist_fields.id, every
// surviving field keeps its rendered answer across the re-render — all 7
// persisted types + sub-steps + the photo-URL value (same code path as a reload);
// a cut field's answer is dropped and a new field renders empty.
//
// If the edit dropped today from the schedule (C5) or archived the template, the
// checklist leaves today's myChecklists list: the open runner is then removed
// live (INV-6 warned-live-removal — the admin was warned in the Builder before
// proceeding) and the device drops back to the checklist list.
async function rerenderOpenChecklistAfterSave(op, silent) {
  if (typeof fillState === 'undefined' || !fillState.activeTemplate) return;
  const openId = fillState.activeTemplate.id;
  // loadMyChecklists re-fetches today's list and, since a runner is open,
  // re-renders it in place with surviving answers hydrated.
  if (typeof loadMyChecklists === 'function') await loadMyChecklists();
  const stillToday = Array.isArray(typeof MY_CHECKLISTS !== 'undefined' ? MY_CHECKLISTS : null)
    && MY_CHECKLISTS.some(function(t) { return t.id === openId; });
  if (!stillToday) {
    // Schedule dropped today / template archived → remove the open checklist live.
    fillState.view = 'list';
    fillState.activeTemplate = null;
    fillState.readonly = false;
    if (typeof location !== 'undefined' && location.hash) location.hash = '';
    if (typeof renderFillOut === 'function') renderFillOut();
    if (!silent) showSyncToast('This checklist was removed');
    return;
  }
  // Surviving live edit: the runner already re-rendered inside loadMyChecklists.
  if (!silent && op.op_type === 'SAVE_TEMPLATE') showSyncToast('Checklist updated');
}

window.rerenderOpenChecklistAfterSave = rerenderOpenChecklistAfterSave;

window.applyOp = applyOp;

// ─── Offline Queue ────────────────────────────────────────────────────────────

// APP_TIMEZONE — the app's ONE timezone. The operator ruled it is New York
// (ledger T-26 decision 83); the backend states the same thing as
// users.DefaultTimezone (backend/internal/users/db.go).
//
// 🛑 If this ever needs to change, it changes HERE and in users.DefaultTimezone
// together. A second zone anywhere in the app is the bug card A1 removed.
const APP_TIMEZONE = 'America/New_York';
window.APP_TIMEZONE = APP_TIMEZONE;

// appDateString — the calendar date (YYYY-MM-DD) of an instant IN THE APP
// TIMEZONE. Pass nothing for "right now"; pass a Date or an ISO string to ask
// which app-day a timestamp fell on.
//
// 🛑 NOT `toISOString().slice(0, 10)`. That is UTC, and in EDT the app's "today"
// rolled over at 20:00 New York — MID-DINNER-SERVICE. A crew member pressing
// Submit at 19:58 and again at 20:02 was, to the app, two different days: the
// second press could not reuse the first's queued idempotency_key, so the drain
// wrote TWO submission rows for one operational evening. Card A1.
//
// formatToParts (not en-CA) so the output cannot depend on locale separators.
//
// Falls back to the old UTC slice if Intl or the zone is unavailable — degrades
// to the previous behaviour rather than throwing inside a submit path. That is
// still the right trade (a submit that works on the wrong day beats a submit
// that throws), but the fallback is NOT silent: it restores the exact UTC
// boundary this card removed, so it warns. Realistically it fires only on a
// small-ICU runtime or an embedded WebView shipped without full tzdata.
function appDateString(when) {
  const d = when === undefined || when === null ? new Date() : new Date(when);
  if (isNaN(d.getTime())) return '';
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: APP_TIMEZONE, year: 'numeric', month: '2-digit', day: '2-digit'
    }).formatToParts(d);
    const get = t => (parts.find(p => p.type === t) || {}).value;
    const y = get('year'), m = get('month'), day = get('day');
    if (y && m && day) return y + '-' + m + '-' + day;
  } catch (e) { /* fall through to the warned UTC fallback */ }
  if (!appDateString._warned) {
    appDateString._warned = true; // once per page — this sits in a submit path
    console.warn(
      '[appDateString] falling back to the UTC date: this runtime cannot resolve ' +
      APP_TIMEZONE + ' via Intl.DateTimeFormat. The app day will roll over at ' +
      '20:00 New York instead of midnight, which can split one dinner service ' +
      'across two days. See card A1 / ledger T-26 decision 83.');
  }
  return d.toISOString().slice(0, 10);
}
window.appDateString = appDateString;

// currentSubmitPeriod — the period a queued submission belongs to.
//
// The app's period is the calendar DAY and always has been: myChecklists is
// fetched per day-of-week, and workflows.html decides "already submitted today"
// by comparing today's date against submitted_at (three places: the list row,
// the runner, and the post-submit refresh). This is that same expression, named
// once, so the queue and the list cannot drift apart on what "today" means.
//
// The DAY is the app-timezone day, not the UTC day — see appDateString.
//
// It is stamped onto every queue entry because an entry may only lend its
// idempotency_key to a submit in the SAME period (workflows.html
// `findQueuedSubmission`). Without it, a persistently-failing server let
// Monday's queued key be adopted by Thursday's submit — upserting Thursday's
// answers onto Monday's submission row and collapsing every day in between.
// Ledger T-25 decision 71.
function currentSubmitPeriod() {
  return appDateString();
}
window.currentSubmitPeriod = currentSubmitPeriod;

async function enqueueSubmission(payload) {
  const db = await getDB();
  await idbPut(db, 'submitQueue', {
    ...payload,
    queuedAt: new Date().toISOString(),
    period: currentSubmitPeriod()
  });
  renderSyncBanner();
}

let _draining = false;

async function drainQueue() {
  if (_draining || !navigator.onLine) return;
  _draining = true;
  try {
    const db = await getDB();
    // Replay in the order the presses actually happened.
    //
    // idbGetAll returns entries in the store's key order, and the key is `id` — a
    // random UUID, so the order is effectively arbitrary. That was harmless while
    // one template could only ever have one queued entry. It stopped being
    // harmless when submitChecklistToAPI began reusing the queued idempotency_key
    // (workflows.html): two presses of one checklist now produce TWO entries that
    // upsert onto the SAME submission row, so whichever replays last wins any
    // field they both set. Sorting by queuedAt makes the later press win, which is
    // what the crew member saw last.
    //
    // Entries queued before this sort existed have no queuedAt; they sort first,
    // which is the conservative choice — an older entry should not beat a newer one.
    const entries = (await idbGetAll(db, 'submitQueue'))
      .slice()
      .sort((a, b) => String(a && a.queuedAt || '').localeCompare(String(b && b.queuedAt || '')));
    for (const entry of entries) {
      try {
        await api('POST', 'submitChecklist', entry);
        await idbDelete(db, 'submitQueue', entry.id);
        renderSyncBanner();
      } catch (err) {
        // DEAD BRANCH, kept for safety: the string `duplicate_submission` appears
        // nowhere in backend/. A repeat of an already-accepted idempotency_key
        // returns 201 with the same submission id and is evicted by the success
        // path above, not here (measured at G6 review, 2026-07-27).
        if (err && err.error === 'duplicate_submission') {
          await idbDelete(db, 'submitQueue', entry.id);
          renderSyncBanner();
        } else if (err && err.status === 409) {
          showConflictError(entry);
          await idbDelete(db, 'submitQueue', entry.id);
          renderSyncBanner();
        } else {
          break;
        }
      }
    }
  } finally {
    _draining = false;
    renderSyncBanner();
  }
}

window.addEventListener('online', drainQueue);

function showConflictError(entry) {
  const container = document.getElementById('checklist-list');
  if (!container) return;
  const card = document.createElement('div');
  card.className = 'inline-error';
  card.style.cssText = 'margin:8px 0;padding:12px 16px;background:var(--card);border:1px solid #c0392b;border-radius:8px;flex-direction:column;align-items:flex-start';
  card.innerHTML = 'This checklist was archived while you were offline. Your responses have not been submitted. <button class="retry-btn" style="margin-top:8px" aria-label="Dismiss">Dismiss</button>';
  card.querySelector('.retry-btn').onclick = () => card.remove();
  container.prepend(card);
}

// renderSyncBanner paints the offline queue: a banner counting queued
// submissions, and a per-row badge on each checklist that has one.
//
// 🛑 VOCABULARY — this badge and workflows.html's per-field chip are two
// different states and must never read the same (ledger T-25 decision 71,
// item 4; both read "Pending sync" between 2026-07-28 and this card, and both
// are reachable on the My Checklists screen at once):
//
//   "Queued"   — a WHOLE submitted checklist is sitting in submitQueue waiting
//                to be sent. Scope: a checklist. `.sync-badge`, this file.
//   "Unsaved"  — ONE field answer has not reached the server.
//                Scope: a field. `.unsaved-mark`, workflows.html.
//
// If either string changes, change the other so they still cannot collide.
async function renderSyncBanner() {
  try {
    const db = await getDB();
    const entries = await idbGetAll(db, 'submitQueue');
    const banner = document.getElementById('sync-banner');
    if (!banner) return;
    if (entries.length === 0) {
      banner.style.display = 'none';
      document.querySelectorAll('.sync-badge').forEach(b => b.remove());
      return;
    }
    banner.style.display = 'block';
    banner.textContent = entries.length + ' submission' + (entries.length > 1 ? 's' : '') + ' queued to send';
    const queuedIds = new Set(entries.map(e => e.template_id));
    document.querySelectorAll('[data-template-id]').forEach(row => {
      const existing = row.querySelector('.sync-badge');
      if (queuedIds.has(row.dataset.templateId)) {
        if (!existing) {
          const badge = document.createElement('span');
          badge.className = 'sync-badge';
          badge.textContent = 'Queued';
          row.appendChild(badge);
        }
      } else if (existing) {
        existing.remove();
      }
    });
  } catch (e) {
    // IndexedDB not available — ignore
  }
}

window.enqueueSubmission = enqueueSubmission;
window.drainQueue = drainQueue;
window.renderSyncBanner = renderSyncBanner;

// ─── Save Status Utilities ────────────────────────────────────────────────────
// updateSaveStatus and SAVE_DEBOUNCE are used by debouncedSaveField in workflows.html.

const SAVE_DEBOUNCE = {};
let _pendingSaves = 0;
let _syncedTimer = null;

function updateSaveStatus(delta) {
  _pendingSaves = Math.max(0, _pendingSaves + delta);
  const el = document.getElementById('save-status');
  if (!el) return;
  clearTimeout(_syncedTimer);
  if (_pendingSaves > 0) {
    el.textContent = 'Saving\u2026';
    el.className = 'pending';
    el.style.display = 'block';
    el.style.opacity = '1';
  } else {
    el.textContent = 'Synced \u2713';
    el.className = 'synced';
    el.style.display = 'block';
    el.style.opacity = '1';
    _syncedTimer = setTimeout(function() {
      el.style.opacity = '0';
      setTimeout(function() { el.style.display = 'none'; }, 300);
    }, 3000);
  }
}

window.SAVE_DEBOUNCE = SAVE_DEBOUNCE;
window.updateSaveStatus = updateSaveStatus;

// ─── submitOp — Single Write Channel (D-08) ──────────────────────────────────
// Sends ops through POST /ops with real device_id + lamport_ts (Plan 03 D-09).
// Self-echo suppression via op.device_id === LAMPORT_CLOCK.deviceId replaces
// the old _recentSaves timing hack.

async function submitOp(opType, entityId, entityType, payload) {
  var ts = 0;
  var deviceId = _fallbackDeviceId;
  if (LAMPORT_CLOCK) {
    try { ts = await LAMPORT_CLOCK.tick(); } catch(e) { console.warn('LamportClock tick failed:', e); }
    deviceId = LAMPORT_CLOCK.deviceId || deviceId;
  }
  const result = await api('POST', 'ops', {
    op_type: opType,
    entity_id: entityId,
    entity_type: entityType,
    payload: payload,
    lamport_ts: ts,
    device_id: deviceId,
  });
  return result;
}

window.submitOp = submitOp;
