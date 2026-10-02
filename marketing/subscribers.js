// marketing/subscribers.js — the Subscribers tab (marketing.html #s3).
// Card H5 `subscribers-tab`, run 20261002. Design: "Current Subscribers 1–2".
// Wire shapes: HANDOFF §5 — the json tags are read VERBATIM (UI-R3).
//
// ── how this file is built ───────────────────────────────────────────────────
// State-first rendering (CLAUDE.md): every interaction mutates STATE and calls
// render(); nothing reads the answer back out of the DOM. ONE delegated click
// listener and ONE delegated input listener, both on #subs-root — no inline
// onclick on anything dynamic.
//
// ── what it never does ───────────────────────────────────────────────────────
// 🛑 It never renders a full phone number or a full email address, because it
// never RECEIVES one: §5's wire shape is `phone_last4` + `email_masked`, masked
// server-side in internal/marketing/subscribers.go. If you find yourself
// wanting the full value here, the answer is no — that is the privacy contract,
// and tests/marketing-subscribers.spec.js [SB-01] asserts the absence on the
// response body, on the rendered cell AND on the whole page HTML.
//
// 🛑 Resend QR is a RECORDED REQUEST, not a send. It POSTs /resend, which
// answers 202 and appends a `resend_requested` event. The note it renders says
// "requested" and must never say "sent" — sending is Activity E's.

const API = '/api/v1/marketing/subscribers';

// The §5 filter set. `label` is the chip's text; the designed row is fixed, so
// this array IS the chip row.
const FILTERS = [
  { key: 'all', label: 'All' },
  { key: 'sms', label: 'SMS' },
  { key: 'email_only', label: 'Email only' },
  { key: 'opted_out', label: 'Opted out' },
];

// §4's `source` enum, plus the "any" row. The labels are the operator's words
// for where a person came from, not the enum's.
const SOURCES = [
  { key: '', label: 'Any source' },
  { key: 'qr', label: 'Campaign QR' },
  { key: 'web_form', label: 'Web form' },
  { key: 'sms_keyword', label: 'SMS keyword' },
  { key: 'toast_import', label: 'Toast import' },
];
const SOURCE_LABEL = Object.fromEntries(SOURCES.map(s => [s.key, s.label]));

// The consent pill. `pill` is the class suffix; `text` is what it says.
const CONSENT = {
  sms: { text: 'SMS', cls: 'ok' },
  email_only: { text: 'Email', cls: 'info' },
  pending: { text: 'No consent', cls: 'mut' },
  stop: { text: 'STOP', cls: 'bad' },
};

// Timeline rendering (this card's call — the slate: "the timeline rendering
// [is] the night's"). One line per event, newest first, each a plain-English
// label and a date. The kinds are §4's enum; an unknown kind renders its raw
// value rather than vanishing, so a future Activity-E event is visible the day
// it starts being written instead of silently dropped.
const EVENT_LABEL = {
  signed_up: 'Signed up',
  code_sent: 'Code sent',
  scanned: 'Scanned a code',
  redeemed: 'Redeemed an offer',
  resend_requested: 'Resend requested',
  opted_out: 'Opted out',
};

const STATE = {
  status: 'loading',   // loading | ready | empty | error | locked
  q: '',
  filter: 'all',
  source: '',
  rows: [],
  counts: { total: 0, sms_opt_in: 0, joined_this_week: 0 },
  lastSynced: null,    // Date of the last SUCCESSFUL read — the offline row's reading
  offline: false,
  error: null,
  openId: null,
  detail: null,
  detailError: null,
  resendState: null,   // null | 'busy' | 'recorded' | 'failed'
};

let root = null;
let loaded = false;
let searchTimer = null;
let reqSeq = 0;

// ── rendering ───────────────────────────────────────────────────────────────

const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c =>
  ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

function fmtDate(iso) {
  if (!iso) return '';
  const d = new Date(iso);
  if (isNaN(d)) return '';
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
}
function fmtTime(d) {
  return d ? d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' }) : '';
}
function money(cents) {
  return '$' + (Number(cents || 0) / 100).toFixed(2).replace(/\.00$/, '');
}

function paint() {
  if (!root) return;
  root.dataset.state = STATE.status;
  root.dataset.filter = STATE.filter;
  root.dataset.source = STATE.source;
  if (STATE.offline) root.dataset.offline = '1'; else delete root.dataset.offline;

  // Locked short-circuits the whole tab, not just its writes: the designed
  // Locked state is the tab (handoff §16, same envelope H2 renders).
  root.innerHTML = STATE.status === 'locked' ? lockedHTML() : [
    headHTML(), bannerHTML(), controlsHTML(), listHTML(), sheetHTML(),
  ].join('');
}

function lockedHTML() {
  return `<div class="card subs-locked">
    <div class="hd"><h1>Subscribers</h1><div class="sub">Managers only</div></div>
    <div class="bd">The mailing list is manager-only. Ask an admin for the manager role
    if you need to see who is on the list and what they consented to.</div>
  </div>`;
}

function headHTML() {
  const c = STATE.counts;
  const n = v => STATE.status === 'loading' ? '<span class="subs-skel"></span>' : esc(v ?? 0);
  return `<div class="card">
    <div class="hd"><h1>Subscribers</h1>
      <div class="sub">Web signups, SMS opt-ins, Toast imports and campaign QR codes</div></div>
    <div class="subs-counts">
      <div class="subs-stat"><b id="subs-total">${n(c.total)}</b><span>On the list</span></div>
      <div class="subs-stat"><b id="subs-sms">${n(c.sms_opt_in)}</b><span>SMS opt-in</span></div>
      <div class="subs-stat"><b id="subs-week">${n(c.joined_this_week)}</b><span>This week</span></div>
    </div>
  </div>`;
}

function bannerHTML() {
  if (STATE.offline) {
    const seen = STATE.lastSynced ? `Last synced ${esc(fmtTime(STATE.lastSynced))}` : 'Never synced on this device';
    return `<div class="subs-banner subs-banner-warn" id="subs-banner">
      <b>Offline.</b> ${seen} &#183; Resend is disabled until you are back on.
      <button class="subs-retry" data-action="retry">Retry</button></div>`;
  }
  if (STATE.status === 'error') {
    return `<div class="subs-banner subs-banner-bad" id="subs-banner">
      <b>Could not load subscribers.</b> ${esc(STATE.error || '')}
      <button class="subs-retry" data-action="retry">Retry</button></div>`;
  }
  return '';
}

function controlsHTML() {
  const chips = FILTERS.map(f =>
    `<button class="subs-chip${STATE.filter === f.key ? ' on' : ''}" data-action="filter" data-filter="${f.key}">${esc(f.label)}</button>`).join('');
  const opts = SOURCES.map(s =>
    `<option value="${s.key}"${STATE.source === s.key ? ' selected' : ''}>${esc(s.label)}</option>`).join('');
  return `<div class="card subs-controls">
    <input id="subs-q" type="search" inputmode="search" autocomplete="off"
           placeholder="Name, email, campaign, or last 4 digits" value="${esc(STATE.q)}">
    <div class="subs-chips" id="subs-filters">${chips}</div>
    <select id="subs-source" aria-label="Source">${opts}</select>
  </div>`;
}

function listHTML() {
  if (STATE.status === 'loading') {
    return `<div class="card" id="subs-list">${
      '<div class="subs-row subs-row-skel"><span class="subs-skel"></span><span class="subs-skel subs-skel-sm"></span></div>'.repeat(4)
    }</div>`;
  }
  if (STATE.status === 'error' && !STATE.rows.length) return '';
  if (!STATE.rows.length) {
    // 🛑 A blank render is a defect (UI-R). Say which list is empty and why it
    // might be, and distinguish "nothing matched your search" from "nobody has
    // signed up" — they call for different next actions.
    const filtered = STATE.q || STATE.filter !== 'all' || STATE.source;
    return `<div class="card subs-empty" id="subs-empty">
      <div class="bd">${filtered
        ? '<b>No subscribers match.</b><br>Nothing on the list fits this search and these filters. Clear them to see everyone.'
        : '<b>No subscribers yet.</b><br>People land here when they fill in the website form, reply to an SMS keyword, scan a campaign QR code, or arrive in a Toast guest import.'}</div>
    </div>`;
  }
  return `<div class="card" id="subs-list">${STATE.rows.map(rowHTML).join('')}</div>`;
}

function rowHTML(r) {
  const consent = CONSENT[r.consent] || CONSENT.pending;
  const where = r.campaign_name
    ? `${esc(SOURCE_LABEL[r.source] || r.source)} &#183; ${esc(r.campaign_name)}`
    : esc(SOURCE_LABEL[r.source] || r.source);
  // phone_last4 / email_masked are the ONLY contact values that exist here.
  const phone = r.phone_last4 ? `•••• ${esc(r.phone_last4)}` : '';
  const contact = [
    phone ? `<span class="subs-phone">${phone}</span>` : '<span class="subs-phone subs-none">No phone</span>',
    r.email_masked ? `<span class="subs-email">${esc(r.email_masked)}</span>` : '',
  ].filter(Boolean).join('<span class="subs-dot">&#183;</span>');
  return `<button class="subs-row" data-action="open" data-id="${esc(r.id)}" data-consent="${esc(r.consent)}">
    <span class="subs-row-main">
      <span class="subs-name">${esc(r.display_name || 'Unnamed')}</span>
      <span class="subs-meta">${contact}</span>
      <span class="subs-meta subs-where">${where}</span>
    </span>
    <span class="subs-row-side">
      <span class="subs-consent subs-pill-${consent.cls}">${esc(consent.text)}</span>
      <span class="subs-joined">${esc(fmtDate(r.joined_at))}</span>
      ${r.visits ? `<span class="subs-visits">${esc(r.visits)} visit${r.visits === 1 ? '' : 's'}</span>` : ''}
    </span>
  </button>`;
}

function sheetHTML() {
  if (!STATE.openId) return '';
  const d = STATE.detail;
  // The cached LIST row is the floor. Offline, or while the detail read is in
  // flight, the sheet renders the identity it already has rather than a
  // skeleton with no name on it — and says plainly which parts it could not
  // load. A blank sheet would be a defect (UI-R3); so would a sheet that
  // silently omitted the history and let the operator think there was none.
  const row = STATE.rows.find(r => r.id === STATE.openId) || null;
  const who = d || row;
  if (!who) {
    return `<div class="subs-sheet" id="subs-sheet" data-id="${esc(STATE.openId)}">
      <div class="subs-sheet-card"><button class="subs-close" data-action="close">Close</button>
      <div class="subs-skel subs-skel-lg"></div><div class="subs-skel"></div></div></div>`;
  }

  const phone = who.phone_last4 ? `\u2022\u2022\u2022\u2022 ${esc(who.phone_last4)}` : 'No phone';
  const header = `<h2 class="subs-sheet-name">${esc(who.display_name || 'Unnamed')}</h2>
      <div class="subs-meta"><span class="subs-phone">${phone}</span>
        ${who.email_masked ? `<span class="subs-dot">&#183;</span><span class="subs-email">${esc(who.email_masked)}</span>` : ''}</div>
      <div class="subs-meta subs-where">${esc(SOURCE_LABEL[who.source] || who.source)}${
        who.campaign_name ? ' &#183; ' + esc(who.campaign_name) : ''}${
        who.source_short ? ' &#183; code ' + esc(who.source_short) : ''}</div>`;

  const resendDisabled = STATE.offline || !d || STATE.resendState === 'busy';
  const note = {
    busy: 'Recording the request\u2026',
    recorded: 'Resend requested. Nothing has been sent \u2014 sending is not switched on yet.',
    failed: 'Could not record the request. Try again.',
  }[STATE.resendState] || (STATE.offline
    ? 'Offline \u2014 a resend cannot be recorded until you are back on.'
    : (!d ? 'This record could not be loaded, so a resend cannot be recorded.' : ''));

  // Everything below the header needs the detail read. Without it, say so once
  // and offer the retry, rather than four empty blocks.
  if (!d) {
    const why = STATE.offline ? 'You are offline.' : esc(STATE.detailError || '');
    return `<div class="subs-sheet" id="subs-sheet" data-id="${esc(STATE.openId)}">
      <div class="subs-sheet-card">
        <button class="subs-close" data-action="close">Close</button>
        ${header}
        <div class="subs-block">
          <h3>Identity code</h3>
          <div id="subs-code-status">Identity-code status unavailable.</div>
          <button id="subs-resend" data-action="resend" disabled>Resend QR</button>
          <div id="subs-resend-note" class="subs-sub">${esc(note)}</div>
        </div>
        <div class="subs-banner subs-banner-warn">
          <b>Consent trail and history not loaded.</b> ${why}
          <button class="subs-retry" data-action="reopen">Retry</button></div>
      </div></div>`;
  }

  const ic = d.identity_code || {};
  const codeStatus = ic.status === 'sent'
    ? `Identity code sent &#183; ${esc(fmtDate(ic.sent_at))}`
    : 'Identity code not sent yet';
  const trail = d.consent_trail || {};
  const consentLines = [
    `<div>SMS: <b>${trail.sms_consent ? 'yes' : 'no'}</b></div>`,
    `<div>Email: <b>${trail.email_consent ? 'yes' : 'no'}</b></div>`,
    trail.consent_evidence ? `<div class="subs-evidence">Evidence: ${esc(trail.consent_evidence)}</div>` : '',
    trail.opted_out_at ? `<div class="subs-stop">Opted out ${esc(fmtDate(trail.opted_out_at))} &#8212; do not message.</div>` : '',
  ].filter(Boolean).join('');
  const events = (d.events || []).map(e =>
    `<div class="subs-event" data-kind="${esc(e.kind)}">
       <span class="subs-event-kind">${esc(EVENT_LABEL[e.kind] || e.kind)}</span>
       <span class="subs-event-at">${esc(fmtDate(e.at))}</span>
     </div>`).join('') ||
    '<div class="subs-event subs-none">Nothing has happened on this record yet.</div>';
  const offers = (d.offers_now || []).map(o =>
    `<div class="subs-offer"><b>${esc(o.name)}</b><span>${esc(o.offer_text)} &#183; ${esc(money(o.face_value_cents))}</span></div>`).join('') ||
    '<div class="subs-none">No campaign is live right now.</div>';

  return `<div class="subs-sheet" id="subs-sheet" data-id="${esc(d.id)}">
    <div class="subs-sheet-card">
      <button class="subs-close" data-action="close">Close</button>
      ${header}

      <div class="subs-block">
        <h3>Identity code</h3>
        <div id="subs-code-status">${codeStatus}</div>
        ${ic.requested_at ? `<div class="subs-sub">Resend requested ${esc(fmtDate(ic.requested_at))}</div>` : ''}
        <button id="subs-resend" data-action="resend"${resendDisabled ? ' disabled' : ''}>Resend QR</button>
        <div id="subs-resend-note" class="subs-sub">${esc(note)}</div>
      </div>

      <div class="subs-block">
        <h3>Consent trail</h3>
        <div id="subs-consent-trail">${consentLines}</div>
      </div>

      <div class="subs-block">
        <h3>Offers now</h3>
        <div id="subs-offers">${offers}</div>
      </div>

      <div class="subs-block">
        <h3>History</h3>
        <div id="subs-timeline">${events}</div>
      </div>
    </div></div>`;
}

// ── data ────────────────────────────────────────────────────────────────────

async function load() {
  const seq = ++reqSeq;
  if (!STATE.rows.length) STATE.status = 'loading';
  render();
  const qs = new URLSearchParams();
  if (STATE.q) qs.set('q', STATE.q);
  if (STATE.filter !== 'all') qs.set('filter', STATE.filter);
  if (STATE.source) qs.set('source', STATE.source);
  try {
    const res = await fetch(`${API}?${qs}`);
    if (seq !== reqSeq) return;           // a newer keystroke already won
    if (res.status === 403) { STATE.status = 'locked'; render(); return; }
    if (res.status === 401) { window.location.href = '/login.html'; return; }
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const body = await res.json();
    STATE.rows = body.rows || [];
    STATE.counts = { total: body.total, sms_opt_in: body.sms_opt_in, joined_this_week: body.joined_this_week };
    STATE.status = STATE.rows.length ? 'ready' : 'empty';
    STATE.offline = false;
    STATE.error = null;
    STATE.lastSynced = new Date();
  } catch (e) {
    if (seq !== reqSeq) return;
    // Offline keeps the last good list on screen and says how old it is; a
    // real failure while online is a loud, retryable banner (UI-R5).
    if (!navigator.onLine) {
      STATE.offline = true;
      if (!STATE.rows.length) STATE.status = 'empty';
    } else {
      STATE.status = 'error';
      STATE.error = String(e.message || e);
    }
  }
  render();
}

async function loadDetail(id) {
  STATE.openId = id;
  STATE.detail = null;
  STATE.detailError = null;
  STATE.resendState = null;
  render();
  try {
    const res = await fetch(`${API}/${encodeURIComponent(id)}`);
    if (res.status === 403) { STATE.status = 'locked'; render(); return; }
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    if (STATE.openId !== id) return;
    STATE.detail = await res.json();
  } catch (e) {
    if (STATE.openId !== id) return;
    STATE.detailError = navigator.onLine ? String(e.message || e) : 'You are offline.';
  }
  render();
}

// resend records a REQUEST. It sends nothing — see the file header.
async function resend() {
  const id = STATE.openId;
  if (!id || STATE.offline) return;
  STATE.resendState = 'busy';
  render();
  try {
    const res = await fetch(`${API}/${encodeURIComponent(id)}/resend`, { method: 'POST' });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    STATE.resendState = 'recorded';
    render();
    await loadDetailQuiet(id);
  } catch (e) {
    STATE.resendState = 'failed';
    render();
  }
}

// loadDetailQuiet refreshes the sheet without flashing it back to a skeleton —
// the resend note must stay readable while the timeline updates underneath it.
async function loadDetailQuiet(id) {
  try {
    const res = await fetch(`${API}/${encodeURIComponent(id)}`);
    if (!res.ok || STATE.openId !== id) return;
    STATE.detail = await res.json();
    render();
  } catch (e) { /* the note already says what happened */ }
}

// ── one click listener, one input listener ───────────────────────────────────

function onClick(ev) {
  const el = ev.target.closest('[data-action]');
  if (!el) return;
  switch (el.dataset.action) {
    case 'open':   loadDetail(el.dataset.id); break;
    case 'close':  STATE.openId = null; STATE.detail = null; STATE.resendState = null; render(); break;
    case 'reopen': loadDetail(STATE.openId); break;
    case 'filter': STATE.filter = el.dataset.filter; load(); break;
    case 'retry':  STATE.openId ? loadDetail(STATE.openId) : load(); break;
    case 'resend': resend(); break;
  }
}

function onInput(ev) {
  const t = ev.target;
  if (t.id === 'subs-q') {
    STATE.q = t.value.trim();
    clearTimeout(searchTimer);
    searchTimer = setTimeout(load, 250);
    return;
  }
  if (t.id === 'subs-source') { STATE.source = t.value; load(); }
}

// Re-rendering replaces the search input, so render() restores focus and the
// caret around paint(). Without it the box loses focus mid-word and the
// operator types the second half of their query into nothing.
function render() {
  const active = document.activeElement;
  const wasSearch = active && active.id === 'subs-q';
  const caret = wasSearch ? active.selectionStart : null;
  paint();
  if (!wasSearch) return;
  const next = document.getElementById('subs-q');
  if (!next) return;
  next.focus();
  try { next.setSelectionRange(caret, caret); } catch (e) { /* unsupported input type */ }
}

// ── boot ────────────────────────────────────────────────────────────────────

function boot() {
  const section = document.getElementById('s3');
  if (!section) return;
  root = document.getElementById('subs-root');
  if (!root) return;
  root.addEventListener('click', onClick);
  root.addEventListener('input', onInput);
  root.addEventListener('change', onInput);   // <select> fires change, not input, on iOS
  window.addEventListener('online', () => { STATE.offline = false; load(); });
  window.addEventListener('offline', () => { STATE.offline = true; render(); });

  // Load on FIRST REVEAL, not on page load: the tab is one of four and a
  // team_member who only ever scans should not pay for a 403 they never see.
  // A MutationObserver on the section's own style attribute means this file
  // never touches the shared show() — which is Cards 2 and 5's neighbour.
  const maybeLoad = () => {
    if (loaded || section.style.display === 'none') return;
    loaded = true;
    load();
  };
  new MutationObserver(maybeLoad).observe(section, { attributes: true, attributeFilter: ['style'] });
  maybeLoad();
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', boot);
else boot();
