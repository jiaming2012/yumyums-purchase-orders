// ─── sync.js — Shared Reactive Sync Module ───────────────────────────────────
// Loaded via <script src="sync.js"> before page-specific scripts.
// Exposes all sync primitives on window.* (no ES modules — same pattern as ptr.js).
// Plan 01 of Phase 10.2: extract from workflows.html without modifying it.

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
    const meta = await idbGet(db, 'syncMeta', 'clock');
    if (meta) {
      clock._ts = meta.lamport_ts;
      clock._deviceId = meta.device_id;
    } else {
      clock._deviceId = crypto.randomUUID();
      await idbPut(db, 'syncMeta', { id: 'clock', lamport_ts: 0, device_id: clock._deviceId });
    }
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
      applyOp(op);
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
  if (!entries.length) return;
  const msg = entries.map(e => e.count + ' field' + (e.count > 1 ? 's' : '') + ' updated by ' + e.name).join(', ');
  showSyncToast(msg);
  _toastQueue = {};
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
// In Plan 02 these become store references. _recentSaves check removed in Plan 03.

function applyOp(op) {
  // Skip self-originated ops (already applied optimistically)
  if (typeof LAMPORT_CLOCK !== 'undefined' && LAMPORT_CLOCK && op.device_id === LAMPORT_CLOCK.deviceId) return;

  if (op.op_type === 'SET_FIELD') {
    const { field_id, value, user_name } = op.payload;
    // Skip echo of our own save (server emits ops with device_id="server", not our client ID)
    if (typeof _recentSaves !== 'undefined' && _recentSaves[field_id]) return;
    const displayName = user_name || 'Someone';
    if (value === null || value === undefined) {
      // Uncheck — remove from state
      if (typeof FIELD_RESPONSES !== 'undefined') delete FIELD_RESPONSES[field_id];
      if (typeof DRAFT_RESPONSES !== 'undefined') {
        const draftIdx = DRAFT_RESPONSES.findIndex(d => d.field_id === field_id);
        if (draftIdx !== -1) DRAFT_RESPONSES.splice(draftIdx, 1);
      }
    } else {
      const entry = { answeredBy: displayName, answeredAt: new Date(op.server_ts) };
      if (typeof value === 'object' && value !== null && value.value !== undefined) {
        entry.value = value.value;
        if (value.sub_steps) entry.sub_steps = value.sub_steps;
      } else {
        entry.value = value;
      }
      if (typeof FIELD_RESPONSES !== 'undefined') FIELD_RESPONSES[field_id] = entry;
      if (typeof DRAFT_RESPONSES !== 'undefined') {
        const existing = DRAFT_RESPONSES.find(d => d.field_id === field_id);
        if (existing) { existing.value = value; existing.answered_at = op.server_ts; }
        else DRAFT_RESPONSES.push({ field_id, value, answered_at: op.server_ts });
      }
    }
    renderFieldResponse(field_id);
    flashField(field_id);
    enqueueSyncToast(op.user_id, displayName, [field_id]);
    // Update progress: runner bar if inside checklist, list page if on checklist list
    if (typeof fillState !== 'undefined' && fillState.activeTemplate) {
      if (typeof updateProgress === 'function') updateProgress();
    } else if (typeof renderMyChecklists === 'function') { renderMyChecklists(); }
  } else if (op.op_type === 'SUBMIT_CHECKLIST') {
    if (typeof loadMyChecklists === 'function') loadMyChecklists(); // re-fetches data + re-renders runner if open
  } else if (op.op_type === 'APPROVE_ITEM' || op.op_type === 'REJECT_ITEM') {
    if (typeof loadPendingApprovals === 'function') loadPendingApprovals();
  } else if (op.op_type === 'SAVE_TEMPLATE' || op.op_type === 'ARCHIVE_TEMPLATE') {
    if (typeof loadTemplates === 'function') loadTemplates();
  }
}

window.applyOp = applyOp;

// ─── Offline Queue ────────────────────────────────────────────────────────────

async function enqueueSubmission(payload) {
  const db = await getDB();
  await idbPut(db, 'submitQueue', {
    ...payload,
    queuedAt: new Date().toISOString()
  });
  renderSyncBanner();
}

let _draining = false;

async function drainQueue() {
  if (_draining || !navigator.onLine) return;
  _draining = true;
  try {
    const db = await getDB();
    const entries = await idbGetAll(db, 'submitQueue');
    for (const entry of entries) {
      try {
        await api('POST', 'submitChecklist', entry);
        await idbDelete(db, 'submitQueue', entry.id);
        renderSyncBanner();
      } catch (err) {
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
    banner.textContent = entries.length + ' submission' + (entries.length > 1 ? 's' : '') + ' pending sync';
    const queuedIds = new Set(entries.map(e => e.template_id));
    document.querySelectorAll('[data-template-id]').forEach(row => {
      const existing = row.querySelector('.sync-badge');
      if (queuedIds.has(row.dataset.templateId)) {
        if (!existing) {
          const badge = document.createElement('span');
          badge.className = 'sync-badge';
          badge.textContent = 'Pending sync';
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

// ─── Save Debounce + autoSaveField ───────────────────────────────────────────
// Kept for backward compatibility — workflows.html still uses these directly.
// In Plan 02, autoSaveField is replaced by submitOp + per-field debounce wrapper.

const SAVE_DEBOUNCE = {};
const _recentSaves = {}; // { fieldId: timestamp } — fields saved locally within last 3s
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

function autoSaveField(fieldId, value) {
  if (SAVE_DEBOUNCE[fieldId]) { clearTimeout(SAVE_DEBOUNCE[fieldId]); updateSaveStatus(-1); }
  updateSaveStatus(1);
  SAVE_DEBOUNCE[fieldId] = setTimeout(async () => {
    SAVE_DEBOUNCE[fieldId] = null;
    const indicator = document.querySelector(`[data-field-id="${fieldId}"] .save-indicator`);
    try {
      // Bundle fail note with value if one exists
      var saveValue = value;
      var fn = (typeof FAIL_NOTES !== 'undefined') ? FAIL_NOTES[fieldId] : null;
      if (fn && (fn.note || fn.severity)) {
        saveValue = { _v: value, _fail_note: { note: fn.note, severity: fn.severity } };
      }
      const result = await api('POST', 'saveResponse', { field_id: fieldId, value: saveValue });
      // Handle 409 conflict — revert to winner value per D-14
      if (result && result._conflict) {
        const winner = result.winner;
        if (typeof FIELD_RESPONSES !== 'undefined') FIELD_RESPONSES[fieldId] = { value: winner.value };
        renderFieldResponse(fieldId);
        flashField(fieldId);
        enqueueSyncToast(null, 'Conflict resolved', [fieldId]);
        updateSaveStatus(-1);
        return;
      }
      // Update DRAFT_RESPONSES so reopening the checklist restores this value
      if (typeof DRAFT_RESPONSES !== 'undefined') {
        if (saveValue === null || saveValue === undefined) {
          // Uncheck — remove draft entry (server deleted the row)
          var idx = DRAFT_RESPONSES.findIndex(function(d) { return d.field_id === fieldId; });
          if (idx !== -1) DRAFT_RESPONSES.splice(idx, 1);
        } else {
          var existing = DRAFT_RESPONSES.find(function(d) { return d.field_id === fieldId; });
          if (existing) {
            existing.value = saveValue;
            existing.answered_at = new Date().toISOString();
          } else {
            DRAFT_RESPONSES.push({ field_id: fieldId, value: saveValue, answered_at: new Date().toISOString() });
          }
        }
      }
      if (indicator) { indicator.textContent = 'Saved'; indicator.className = 'save-indicator saved'; setTimeout(() => { indicator.textContent = ''; indicator.className = 'save-indicator'; }, 800); }
      // Mark as recently saved so incoming WS echo is suppressed
      _recentSaves[fieldId] = Date.now();
      setTimeout(() => { delete _recentSaves[fieldId]; }, 3000);
      if (typeof clearFieldError === 'function') clearFieldError(fieldId);
    } catch (e) {
      if (typeof showFieldError === 'function') showFieldError(fieldId);
    }
    updateSaveStatus(-1);
  }, 400);
}

window.SAVE_DEBOUNCE = SAVE_DEBOUNCE;
window.updateSaveStatus = updateSaveStatus;
window.autoSaveField = autoSaveField;

// ─── submitOp — Single Write Channel (D-08) ──────────────────────────────────
// The single client-side write function. Initially routes to existing endpoints.
// Plan 03 switches this to POST /ops with optimistic apply + rollback on failure.

async function submitOp(opType, entityId, entityType, payload) {
  // Route to existing endpoints based on opType
  switch (opType) {
    case 'SET_FIELD':
      return api('POST', 'saveResponse', { field_id: entityId, value: payload.value });
    case 'SUBMIT_CHECKLIST':
      return api('POST', 'submitChecklist', payload);
    case 'APPROVE_ITEM':
      return api('POST', 'approveSubmission', payload);
    case 'REJECT_ITEM':
      return api('POST', 'rejectItem', payload);
    case 'SAVE_TEMPLATE':
      if (payload.id) return api('PUT', 'updateTemplate/' + payload.id, payload);
      return api('POST', 'createTemplate', payload);
    case 'ARCHIVE_TEMPLATE':
      return api('DELETE', 'archiveTemplate/' + entityId);
    default:
      throw new Error('Unknown op type: ' + opType);
  }
}

window.submitOp = submitOp;
