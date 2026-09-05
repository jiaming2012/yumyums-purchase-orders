// marketing/scan-page.js — browser wiring for the Scan section of
// marketing.html (card camera-scanner-decode, run 20260905; design §12/§16,
// F2/F3/F5/F6, D-KR3). This is the module entry point the page loads; Card 6's
// submit flow extends the SAME section through the two contracts this file
// exposes: the DOM inside #scanner-host (notably #scan-submit-slot) and
// window.MarketingScan (notably setOnlineProbe + the serialized enqueue).
//
// State-first rendering (repo convention): mutate SCAN_STATE → render() → the
// DOM updates from state. ONE delegated click listener + ONE change listener
// on #scanner-host, routed via data-action / target id.

import {
  createRxDatabase, getRxStorageDexie, replicateRxCollection, Subject,
} from '../vendor/rxdb.bundle.js';
import {
  marketingCollectionSpec, startCodesReplica, startOffersReplica, resolveOffers,
} from './sync/replicas.js';
import { scanAttemptsCollectionSpec, enqueueAttempt } from './sync/push-replication.js';
import { createSyncClock } from './sync/clock.js';
import {
  createTokenHasher, createScanResolver, makeSerializedEnqueue,
} from './scanner.js';

const CLOCK_KEY = 'hq_marketing_clock_v1';
const SYNC_KEY = 'hq_marketing_sync_v1'; // {restUrl, bearer} — provisioning lands later; absent tonight

const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
}[c]));
const fmtWhen = (iso) => {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return String(iso);
  return new Date(t).toLocaleString(undefined, {
    month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit',
  });
};
const readJson = (key) => {
  try { return JSON.parse(localStorage.getItem(key)); } catch (e) { return null; }
};

// ── page state ──────────────────────────────────────────────────────────────
const SCAN_STATE = {
  result: null,      // last resolver result (rendered into #scan-result)
  camError: null,    // loud, retryable camera failure (UI-R6)
  cameraOn: false,
  synced: false,     // true once a configured replica finishes its initial pull
};

// Card 6 (#13) replaces this with the real reachability probe. Tonight the
// default NEVER claims liveness we cannot see (landed-card rule): offline is
// the honest default, and F3's online branch is reachable via setOnlineProbe.
let onlineProbe = () => false;
const isOnline = () => { try { return !!onlineProbe(); } catch (e) { return false; } };

// Card 6 (redemption-submit-flow): the ONE registration surface the submit
// flow plugs into — {gate, onResult, onRender, onScanAgain, onInput, onChange,
// actions} — consulted at the existing choke points below. Every consult is
// FAIL-OPEN on purpose: if the submit flow's JS breaks, Card 5's scanner keeps
// resolving and rendering (loudly console.error'd), it just loses the submit
// affordance — never a bricked Scan section.
let submitFlow = null;
const sfCall = (name, ...args) => {
  if (!submitFlow || typeof submitFlow[name] !== 'function') return undefined;
  try { return submitFlow[name](...args); } catch (e) {
    console.error(`[marketing scan] submit-flow ${name} failed`, e);
    return undefined;
  }
};

// ── render ──────────────────────────────────────────────────────────────────
const $ = (id) => document.getElementById(id);

function statusLine() {
  if (SCAN_STATE.synced) return 'Replica synced — offers verified against the last pull.';
  return 'Local verification only — not yet synced this session.';
}

// Which kinds carry Card 6's mount slot: everything except the hard offline
// rejects and non-codes. unknownCode keeps it — F2's permissioned override is
// a SUBMIT-time affordance (Card 6), so the slot must exist there.
const SLOT_KINDS = new Set(['offerReady', 'embeddedOffer', 'unknownCode', 'deferToServer']);

function resultCard(r) {
  const slot = SLOT_KINDS.has(r.kind) ? '<div id="scan-submit-slot"></div>' : '';
  const again = '<button class="scan-again" data-action="scan-again">Scan next</button>';
  switch (r.kind) {
    case 'offerReady': {
      const rows = r.offers.map((o) => `
        <div class="offer-row" data-code-id="${esc(o.code_id)}">
          <div class="offer-main">Offer</div>
          <div class="offer-sub">Expires ${esc(fmtWhen(o.expires_at))}${o.campaign_id ? ` &middot; campaign ${esc(String(o.campaign_id).slice(0, 8))}` : ''}</div>
        </div>`).join('');
      return `<div class="rc rc-ok">
        <div class="rc-head">${r.offers.length} offer${r.offers.length === 1 ? '' : 's'} available</div>
        <div id="scan-offer-list">${rows}</div>
        <div class="result-note">Apply the matching offer in Toast by hand &mdash; the app never auto-applies.</div>
        ${slot}${again}</div>`;
    }
    case 'embeddedOffer': {
      const exp = r.offer.expires_at
        ? `<div class="offer-sub${r.expired ? ' struck' : ''}">Expires ${esc(fmtWhen(r.offer.expires_at))}${r.expired ? ' &middot; EXPIRED' : ''}</div>` : '';
      return `<div class="rc rc-warn">
        <div class="rc-head">${esc(r.offer.label)} <span class="badge-inline">Unverified</span></div>
        ${exp}
        <div class="result-note">Read from the code itself &mdash; not yet verified with the server. Redemption is still checked at submit.</div>
        ${slot}${again}</div>`;
    }
    case 'unknownCode':
      return `<div class="rc rc-warn">
        <div class="rc-head">Code not recognized</div>
        <div class="result-note">Can&#39;t verify this code on this device &mdash; it may not have synced yet. Connect to verify at submit.</div>
        ${slot}${again}</div>`;
    case 'spentLocally':
      return `<div class="rc rc-bad">
        <div class="rc-head">Already used</div>
        <div class="offer-sub">at ${esc(fmtWhen(r.redeemed_at))}${r.redeemed_by ? ` by ${esc(r.redeemed_by)}` : ''}</div>
        <div class="result-note">This device&#39;s copy shows the code redeemed, and you&#39;re offline.</div>
        ${again}</div>`;
    case 'deferToServer':
      return `<div class="rc rc-warn">
        <div class="rc-head">Shows as used on this device</div>
        <div class="offer-sub">at ${esc(fmtWhen(r.redeemed_at))}${r.redeemed_by ? ` by ${esc(r.redeemed_by)}` : ''}</div>
        <div class="result-note">You&#39;re online &mdash; the local copy may be stale, so the server has the final say at submit.</div>
        ${slot}${again}</div>`;
    case 'expiredLocally':
      return `<div class="rc rc-bad">
        <div class="rc-head">Expired</div>
        <div class="offer-sub">Expired ${esc(fmtWhen(r.expires_at))}</div>
        ${again}</div>`;
    case 'invalidPayload':
      return `<div class="rc rc-bad">
        <div class="rc-head">Not a Yumyums code</div>
        <div class="result-note">That QR doesn&#39;t look like a customer code.</div>
        ${again}</div>`;
    case 'decodeError':
    default:
      return `<div class="rc rc-bad">
        <div class="rc-head">No QR code found</div>
        <div class="result-note">Try again with the code centered, flat and well-lit.</div>
        ${again}</div>`;
  }
}

function render() {
  $('scan-status').textContent = statusLine();

  const err = $('scan-cam-error');
  if (SCAN_STATE.camError) {
    err.textContent = SCAN_STATE.camError;
    err.hidden = false;
  } else {
    err.hidden = true;
  }
  $('scan-camera-wrap').classList.toggle('live', SCAN_STATE.cameraOn);

  const box = $('scan-result');
  const r = SCAN_STATE.result;
  if (!r) {
    box.hidden = true;
    box.removeAttribute('data-kind');
    box.removeAttribute('data-token-hash');
    box.removeAttribute('data-source');
    box.innerHTML = '';
    sfCall('onRender'); // Card 6: the cleared view repaints too
    return;
  }
  box.hidden = false;
  box.setAttribute('data-kind', r.kind);
  if (r.token_hash) box.setAttribute('data-token-hash', r.token_hash);
  else box.removeAttribute('data-token-hash');
  if (r.source) box.setAttribute('data-source', r.source);
  else box.removeAttribute('data-source');
  box.innerHTML = resultCard(r);
  // Card 6 repaint: the submit flow repaints its slot content after EVERY
  // card render (this function rebuilds #scan-result's DOM, the
  // #scan-submit-slot mount point included).
  sfCall('onRender');
}

// ── boot ────────────────────────────────────────────────────────────────────
async function boot() {
  const clock = createSyncClock({
    initialState: readJson(CLOCK_KEY),
    persist: (s) => { try { localStorage.setItem(CLOCK_KEY, JSON.stringify(s)); } catch (e) { /* storage full/blocked — offset still live in-memory */ } },
  });

  const db = await createRxDatabase({ name: 'hqmarketing', storage: getRxStorageDexie() });
  const cols = await db.addCollections({
    ...marketingCollectionSpec(),
    ...scanAttemptsCollectionSpec(),
  });

  const hashToken = createTokenHasher({ subtle: crypto.subtle });
  const resolver = createScanResolver({
    codesCollection: cols.codes,
    offersCollection: cols.offers,
    resolveOffers,
    clock,
    hashToken,
  });
  const enqueue = makeSerializedEnqueue(enqueueAttempt, cols.scan_attempts);

  // ── replica wiring. The MECHANISM is fully threaded (clock included — §5.1);
  // coordinates arrive via provisioning (localStorage SYNC_KEY) or an explicit
  // startSync() call. Realtime resubscribe/liveness is Card 6's machine (#13)
  // — resync() is the manual nudge until then.
  let syncHandles = null;
  async function startSync(cfg) {
    const { restUrl, bearer } = cfg || {};
    if (!restUrl || !bearer || syncHandles) return syncHandles;
    const deps = (collection, replicationIdentifier) => ({
      replicateRxCollection,
      collection,
      restUrl,
      bearer,
      fetchImpl: (...a) => fetch(...a),
      stream$: new Subject(),
      clock,
      replicationIdentifier,
    });
    syncHandles = {
      codes: startCodesReplica(deps(cols.codes, 'marketing-codes-pull')),
      offers: startOffersReplica(deps(cols.offers, 'marketing-offers-pull')),
    };
    Promise.all([
      syncHandles.codes.awaitInitialReplication(),
      syncHandles.offers.awaitInitialReplication(),
    ]).then(() => { SCAN_STATE.synced = true; render(); }).catch(() => { /* stays honest: not synced */ });
    return syncHandles;
  }
  function resync() {
    if (!syncHandles) return;
    syncHandles.codes.reSync();
    syncHandles.offers.reSync();
  }

  // ── scanning ──
  let cameraQr = null;   // html5-qrcode camera instance (#scan-camera-view)
  let fileQr = null;     // separate instance for the file-scan path
  let cameraPaused = false;
  let decodeBusy = false;

  async function doScan(payload) {
    // Card 6's scan gate (F6 session semantics): same-code re-scans re-show,
    // a different code mid-session prompts finish-current, mid-submit scans
    // are ignored — all decided by the machine BEFORE resolution runs. A gate
    // error fails open (Card 5's scanner must survive a Card 6 defect).
    if (submitFlow) {
      let proceed = true;
      try { proceed = await submitFlow.gate(payload); } catch (e) {
        console.error('[marketing scan] submit-flow gate failed', e);
        proceed = true;
      }
      if (proceed === false) { render(); return null; }
    }
    const result = await resolver.resolve(payload, { online: isOnline() });
    SCAN_STATE.result = result;
    render();
    if (submitFlow) {
      try { await submitFlow.onResult(result); } catch (e) {
        console.error('[marketing scan] submit-flow onResult failed', e);
      }
    }
    return result;
  }

  async function onCameraDecode(text) {
    if (decodeBusy) return;
    decodeBusy = true;
    try {
      if (cameraQr && SCAN_STATE.cameraOn && !cameraPaused) {
        try { cameraQr.pause(true); cameraPaused = true; } catch (e) { /* already stopped */ }
      }
      await doScan(text);
    } finally {
      decodeBusy = false;
    }
  }

  async function startCamera() {
    SCAN_STATE.camError = null;
    render();
    try {
      if (cameraPaused && cameraQr) {
        cameraQr.resume();
        cameraPaused = false;
        SCAN_STATE.cameraOn = true;
        render();
        return;
      }
      cameraQr = cameraQr || new Html5Qrcode('scan-camera-view');
      await cameraQr.start(
        { facingMode: 'environment' },
        { fps: 10, qrbox: { width: 230, height: 230 } },
        (text) => { onCameraDecode(text); },
        () => { /* per-frame decode misses are normal */ },
      );
      SCAN_STATE.cameraOn = true;
      render();
    } catch (e) {
      // UI-R6: loud, named, retryable (the button stays; tapping it retries).
      SCAN_STATE.cameraOn = false;
      SCAN_STATE.camError = 'Camera unavailable — ' + (e && e.message ? e.message : 'permission denied or no camera found') + '. Fix camera access and tap Start camera to retry, or scan from a photo.';
      render();
    }
  }

  async function onFilePicked(input) {
    const file = input.files && input.files[0];
    if (!file) return;
    try {
      fileQr = fileQr || new Html5Qrcode('scan-file-surface');
      const text = await fileQr.scanFile(file, false);
      await doScan(text);
    } catch (e) {
      SCAN_STATE.result = { kind: 'decodeError' };
      render();
    } finally {
      input.value = ''; // same photo can be re-scanned
    }
  }

  function scanAgain() {
    // Card 6: clearing the view also closes the machine session (F6 — a
    // stale session must not prompt on the next customer's code).
    sfCall('onScanAgain');
    SCAN_STATE.result = null;
    render();
    if (cameraPaused && cameraQr) {
      try { cameraQr.resume(); cameraPaused = false; } catch (e) { /* camera gone — button still there */ }
    }
  }

  // ── event delegation: ONE click + ONE change + ONE input listener on the
  // host (workflows.html convention). Card 6 routes its data-actions through
  // the SAME click listener via its registered actions map — no new listeners.
  const host = $('scanner-host');
  host.addEventListener('click', (e) => {
    const el = e.target.closest('[data-action]');
    if (!el) return;
    const action = el.getAttribute('data-action');
    if (action === 'start-camera') startCamera();
    else if (action === 'scan-again') scanAgain();
    else if (submitFlow && submitFlow.actions && typeof submitFlow.actions[action] === 'function') {
      try { submitFlow.actions[action](el, e); } catch (err) {
        console.error(`[marketing scan] submit-flow action ${action} failed`, err);
      }
    }
  });
  host.addEventListener('change', (e) => {
    if (e.target && e.target.id === 'scan-file') onFilePicked(e.target);
    else sfCall('onChange', e);
  });
  host.addEventListener('input', (e) => { sfCall('onInput', e); });

  render();

  // Provisioned coordinates (absent tonight — Card 6 / later provisioning).
  const sc = readJson(SYNC_KEY);
  if (sc) startSync(sc).catch(() => {});

  return {
    db,
    collections: cols,
    clock,
    resolver,
    scanText: doScan,
    hasherStats: () => hashToken.stats(),
    enqueue,
    setOnlineProbe: (fn) => { onlineProbe = typeof fn === 'function' ? fn : (() => false); },
    startSync,
    resync,
    scanAgain,
    // Card 6's registration point (see the submitFlow block above).
    setSubmitFlow: (handlers) => { submitFlow = handlers && typeof handlers === 'object' ? handlers : null; },
  };
}

const ready = boot().then((api) => {
  Object.assign(window.MarketingScan, api, { booted: true });
  return window.MarketingScan;
}).catch((e) => {
  // Loud, not blank (UI-R3/R6): the Scan section names its failure.
  const status = document.getElementById('scan-status');
  if (status) status.textContent = 'Scanner failed to start — reload to retry. (' + (e && e.message ? e.message : e) + ')';
  throw e;
});

window.MarketingScan = { booted: false, ready };
