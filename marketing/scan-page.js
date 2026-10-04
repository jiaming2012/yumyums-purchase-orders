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
  addRxPlugin, RxDBMigrationSchemaPlugin,
} from '../vendor/rxdb.bundle.js';
import {
  marketingCollectionSpec, startCodesReplica, startOffersReplica,
  startCampaignsReplica, createCampaignPolicySource, resolveOffers,
} from './sync/replicas.js';
import {
  scanAttemptsCollectionSpec, enqueueAttempt, makePushHandler,
  startScanAttemptsReplica,
} from './sync/push-replication.js';
import { createSyncClock } from './sync/clock.js';
import {
  createTokenHasher, createScanResolver, makeSerializedEnqueue, createServerLookup,
} from './scanner.js';
// The scan-time lookup's budget IS the connectivity probe's (card
// scan-time-verify): one number says how long this app waits on a link before
// calling it dead, whichever question it was asking.
import { PROBE_TIMEOUT_MS } from './submit-support.js';
import { sha256Hex } from './sync/sha256.js';
import { reportBootFailure, SECURE_ADDRESS } from './boot-failure.js';

const CLOCK_KEY = 'hq_marketing_clock_v1';
// {restUrl, bearer, deviceId} — written ONLY by provisionSync() below (card
// sync-coordinates-provisioning: the tree's one writer). `bearer` is inert at
// the door — internal/sync/proxy.go substitutes a per-request mint for the
// SESSION user (spike fact 1) — it exists to satisfy startSync's truthiness
// check and keep the direct-substrate path open. `deviceId` is the mint
// envelope's `sub` and is the push replica's identity coordinate.
const SYNC_KEY = 'hq_marketing_sync_v1';

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

// B-447: the campaign-name lookup, set in boot() from the §8 policy source
// (which already keeps a live Map mirror of the local campaigns collection —
// no second query, and synchronous, which is what a render needs). Default is
// the honest one: before boot wires it, nothing on this device knows any
// campaign's name.
let campaignNameFor = () => null;

/**
 * B-447 — "The offer card names a campaign UUID, not the campaign or the
 * code." The crew member holding the phone was shown `campaign 02d234bc`
 * while the campaign's `name` sat in the same synced replica row, and nothing
 * named the code at all; verifying against a customer's claim meant comparing
 * a UUID prefix to nothing.
 *
 * Two labels, each degrading honestly (UI-R3 — a render that states a fact the
 * app does not have is a defect, and "Campaign undefined" is the classic
 * shape of it):
 *
 *   campaignLabel  the replica's `name`, or `Campaign <id prefix>` when the
 *                  row predates the widened pull / has not arrived. Never a
 *                  bare UUID, never the word undefined.
 *   codeLabel      `Code ····<last 4 of code_id>`. The code row's id is what
 *                  `scan_attempts.code_id` carries, so the four characters the
 *                  crew member reads out are the four a manager can match in
 *                  HQ. (The token hash was the alternative and is worse: it
 *                  is nowhere a human can see it.)
 */
function campaignLabel(campaignId) {
  if (!campaignId) return '';
  const name = campaignNameFor(campaignId);
  if (name) return esc(name);
  return `Campaign ${esc(String(campaignId).slice(0, 8))}`;
}
function codeLabel(codeId) {
  if (!codeId) return '';
  const s4 = String(codeId).slice(-4);
  return `Code \u00b7\u00b7\u00b7\u00b7${esc(s4)}`;
}

function statusLine() {
  if (SCAN_STATE.synced) return 'Replica synced — offers verified against the last pull.';
  return 'Local verification only — not yet synced this session.';
}

// Which kinds carry Card 6's mount slot: everything except the hard offline
// rejects and non-codes. unknownCode keeps it — F2's permissioned override is
// a SUBMIT-time affordance (Card 6), so the slot must exist there.
const SLOT_KINDS = new Set(['offerReady', 'embeddedOffer', 'unknownCode', 'deferToServer']);

// Card scan-time-verify (B-468): the one line added to today's fallback card
// when an ONLINE phone tried to ask the server about a never-synced code and
// could not (timeout / dead link / non-200). Empty for every other result —
// in particular for every OFFLINE result, whose markup is byte-for-byte what
// it was before the card ([SV-04] compares it against a pre-change capture).
function uncheckedNote(r) {
  if (r.verified !== false) return '';
  return `<div class="result-note" id="scan-server-unchecked">Couldn&#39;t check the server just now &mdash; treat this code as unverified until the submit goes through.</div>
        `;
}

function resultCard(r) {
  const slot = SLOT_KINDS.has(r.kind) ? '<div id="scan-submit-slot"></div>' : '';
  const again = '<button class="scan-again" data-action="scan-again">Scan next</button>';
  switch (r.kind) {
    case 'offerReady': {
      const rows = r.offers.map((o) => `
        <div class="offer-row" data-code-id="${esc(o.code_id)}">
          <div class="offer-main">Offer</div>
          <div class="offer-sub">Expires ${esc(fmtWhen(o.expires_at))}${o.campaign_id ? ` &middot; ${campaignLabel(o.campaign_id)}` : ''}${o.code_id ? ` &middot; ${codeLabel(o.code_id)}` : ''}</div>
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
        ${uncheckedNote(r)}${slot}${again}</div>`;
    }
    case 'unknownCode':
      return `<div class="rc rc-warn">
        <div class="rc-head">Code not recognized</div>
        <div class="result-note">Can&#39;t verify this code on this device &mdash; it may not have synced yet. Connect to verify at submit.</div>
        ${uncheckedNote(r)}${slot}${again}</div>`;
    case 'spentLocally':
      // Card scan-time-verify: the SERVER said so (an online phone asked about
      // a code it had never synced). Same head, same shape, no submit slot —
      // only the sentence about WHO says it is used differs.
      if (r.source === 'server') {
        return `<div class="rc rc-bad">
        <div class="rc-head">Already used</div>
        <div class="offer-sub">at ${esc(fmtWhen(r.redeemed_at))}${r.redeemed_by ? ` by ${esc(r.redeemed_by)}` : ''}</div>
        <div class="result-note">The server shows this code already redeemed &mdash; don&#39;t apply the discount.</div>
        ${again}</div>`;
      }
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

function startCameraButton() {
  return document.querySelector('#scanner-host [data-action="start-camera"]');
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
  // A running camera has no "start" affordance. Leaving the button up invited
  // a second tap, and html5-qrcode's start() refuses to re-enter a live scan
  // ("Cannot clear while scan is ongoing") — which the catch below then
  // misreported as "camera unavailable" over a visibly live preview (2026-10-01).
  const startBtn = startCameraButton();
  if (startBtn) startBtn.hidden = !!SCAN_STATE.cameraOn;

  const box = $('scan-result');
  const r = SCAN_STATE.result;
  if (!r) {
    box.hidden = true;
    box.removeAttribute('data-kind');
    box.removeAttribute('data-token-hash');
    box.removeAttribute('data-source');
    box.removeAttribute('data-verified');
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
  // Present ONLY on the could-not-ask-the-server fallback (scan-time-verify).
  if (r.verified === false) box.setAttribute('data-verified', 'false');
  else box.removeAttribute('data-verified');
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

  // 🛑 REQUIRED before addCollections (card refusal-holds-before-sync):
  // SCAN_ATTEMPTS_SCHEMA is version 1 now, and rxdb runs
  // `autoMigrate && version !== 0 && await migratePromise()` unconditionally.
  // Without this plugin that call hits a prototype stub that THROWS, the
  // collection creation rejects, and boot()'s catch below renders "Scanner
  // failed to start" on every device. The plugin is idempotent to register.
  addRxPlugin(RxDBMigrationSchemaPlugin);

  // hashFunction: RxDB's default is `crypto.subtle.digest`, which browsers
  // withhold on insecure origins (plain http to the dev box's LAN IP) — boot
  // died there with RxDB error UT8 before a single collection existed.
  // sha256Hex is WebCrypto when present and a byte-identical JS digest when
  // not, so the same IndexedDB hashes the same way on either origin.
  const db = await createRxDatabase({ name: 'hqmarketing', storage: getRxStorageDexie(), hashFunction: sha256Hex });
  const cols = await db.addCollections({
    ...marketingCollectionSpec(),
    ...scanAttemptsCollectionSpec(),
  });

  // The §8 policy source lives HERE, not in submit-flow.js, and the reason is
  // build-fact 1: `error$` does not replay to late subscribers, so the source
  // has to exist before the campaigns replica does and latch the first
  // emission itself. This file owns both the collections and the replicas —
  // it is the only place that ordering can be guaranteed. submit-flow.js
  // consumes it through MS.campaignPolicy.
  const campaignPolicy = createCampaignPolicySource(cols.campaigns);
  // B-447: the offer card's name lookup. Reads the SAME Map the §8 predicate
  // reads, so the label and the policy can never disagree about which
  // campaign row the device holds.
  campaignNameFor = (id) => {
    try { return campaignPolicy.nameFor(id); } catch (e) { return null; }
  };

  // No `subtle` handed in: the hasher takes WebCrypto when the origin has it
  // and the JS digest otherwise (same reason as hashFunction above).
  const hashToken = createTokenHasher();
  // Card scan-time-verify (B-468): the resolver is rebuilt WITH the
  // `serverLookup` dep the moment sync coordinates exist (startSync below),
  // and `resolver` is a stable facade over whichever one is current — the
  // object handed to window.MarketingScan never changes identity.
  //
  // 🛑 A device with NO coordinates has no lookup at all, on purpose: there
  // is no door to ask, so an unprovisioned phone — online or not — resolves
  // exactly as it did before the card (no request, no `verified` key).
  const resolverDeps = {
    codesCollection: cols.codes,
    offersCollection: cols.offers,
    resolveOffers,
    clock,
    hashToken,
  };
  let activeResolver = createScanResolver(resolverDeps);
  const resolver = { resolve: (payload, opts) => activeResolver.resolve(payload, opts) };
  const enqueue = makeSerializedEnqueue(enqueueAttempt, cols.scan_attempts);

  // ── replica wiring. The MECHANISM is fully threaded (clock included — §5.1);
  // coordinates arrive via provisioning (localStorage SYNC_KEY) or an explicit
  // startSync() call. Realtime resubscribe/liveness is Card 6's machine (#13)
  // — resync() is the manual nudge until then.
  let syncHandles = null;
  async function startSync(cfg) {
    const { restUrl, bearer, deviceId } = cfg || {};
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
    // Card requires-online-replication: the §8 policy flag rides its own
    // replica (no expiry bound — campaigns has no expires_at); the submit
    // flow's default policy source reads the local collection.
    //
    // 🛑 THE NEXT TWO STATEMENTS MUST STAY ADJACENT AND UNAWAITED (card
    // refusal-holds-before-sync, build-fact 1). `error$` emits on the FIRST
    // failed pull (spike 01 measured t+145ms) and does NOT replay — a
    // subscriber attached even one `await` later sees zero emissions and the
    // policy source can never report the replica as erroring. Start the
    // campaigns replica, attach the latch, THEN do everything else.
    // 🛑 THE IDENTIFIER IS BUMPED, DELIBERATELY (card scanner-polish, B-447).
    // The checkpoint lives under the replicationIdentifier, so a device that
    // already synced campaigns would never re-pull the rows it holds and
    // would show the `Campaign <prefix>` fallback until each campaign was
    // next touched server-side. A new identifier resets the checkpoint and
    // costs exactly one full pull of a table with a handful of rows.
    const campaignsRep = startCampaignsReplica(deps(cols.campaigns, 'marketing-campaigns-pull-v2'));
    campaignPolicy.attach(campaignsRep);

    // Card scan-time-verify (B-468 / decision 200): arm the scan-time lookup
    // from the SAME coordinates the pull replicas just got — same door
    // (`restUrl`), same bearer path, same fetch — with the connectivity
    // probe's timeout. One GET per never-synced code, online only (the guard
    // lives in scanner.js resolve() step 3); no new backend surface.
    activeResolver = createScanResolver({
      ...resolverDeps,
      serverLookup: createServerLookup({
        restUrl,
        bearer,
        fetchImpl: (...a) => fetch(...a),
        timeoutMs: PROBE_TIMEOUT_MS,
      }),
    });

    syncHandles = {
      codes: startCodesReplica(deps(cols.codes, 'marketing-codes-pull')),
      offers: startOffersReplica(deps(cols.offers, 'marketing-offers-pull')),
      campaigns: campaignsRep,
    };
    // Push wiring (card sync-coordinates-provisioning, spike fact 4's decided
    // scope): the queue's FIRST caller. 🛑 `deviceId` comes ONLY from the mint
    // envelope's `sub` — spike fact 3 measured the alternative both ways: any
    // other value draws the RLS with-check 403 AFTER /rpc/redeem has burned
    // the code (a stranded burn that can never record) plus the F-2
    // throw-retry head-of-line poison. With no deviceId in the coordinates
    // the push does NOT start — a pull-only page queues attempts locally
    // (today's shape) rather than poisoning them, and the next provisioned
    // load (which always writes deviceId) drains them.
    if (deviceId) {
      syncHandles.scanAttempts = startScanAttemptsReplica({
        replicateRxCollection,
        collection: cols.scan_attempts,
        pushHandler: makePushHandler({
          restUrl,
          bearer,
          deviceId,
          fetchImpl: (...a) => fetch(...a),
          attemptsCollection: cols.scan_attempts,
          codesCollection: cols.codes,
        }),
      });
    }
    Promise.all([
      syncHandles.codes.awaitInitialReplication(),
      syncHandles.offers.awaitInitialReplication(),
      syncHandles.campaigns.awaitInitialReplication(),
    ]).then(() => { SCAN_STATE.synced = true; render(); }).catch(() => { /* stays honest: not synced */ });
    return syncHandles;
  }
  function resync() {
    if (!syncHandles) return;
    syncHandles.codes.reSync();
    syncHandles.offers.reSync();
    syncHandles.campaigns.reSync();
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
      if (result.kind === 'spentLocally' && result.source === 'server') {
        // Card scan-time-verify: the SERVER has already said "used", so there
        // is nothing left for the submit machine to decide and no session to
        // hold open. Handing it RESOLVED would take the machine's F3-online
        // arm (spentLocally + online → offerReady, "the server decides at
        // submit") — a submit session for a code the server just refused, and
        // a "finish the current customer first" prompt on the next scan.
        // Close the session through the EXISTING hook instead: onScanAgain
        // sends NEXT_CUSTOMER, a pair `resolving` already declares. No new
        // state, event or pair; the "Already used" card stays on screen.
        sfCall('onScanAgain');
      } else {
        try { await submitFlow.onResult(result); } catch (e) {
          console.error('[marketing scan] submit-flow onResult failed', e);
        }
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

  function cameraLive() {
    if (!cameraQr || typeof cameraQr.getState !== 'function') return false;
    const st = cameraQr.getState();
    return st === Html5QrcodeScannerState.SCANNING || st === Html5QrcodeScannerState.PAUSED;
  }

  // html5-qrcode throws plain strings for its state errors (no .message), so
  // reading only e.message discarded the real reason behind a generic
  // "permission denied" guess.
  function cameraErrorText(e) {
    if (typeof e === 'string' && e.trim()) return e.trim().replace(/[.\s]+$/, '');
    if (e && e.message) return String(e.message).replace(/[.\s]+$/, '');
    return 'permission denied or no camera found';
  }

  async function startCamera() {
    // Already running (or paused mid-session)? Re-entering html5-qrcode's
    // start() throws a string and leaves the stream up; there is nothing to
    // start. Reconcile state and return — the button is hidden while live,
    // but the guard lives here, not in the button's visibility.
    if (cameraLive() && !cameraPaused) {
      SCAN_STATE.camError = null;
      SCAN_STATE.cameraOn = true;
      render();
      return;
    }
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
      // On a plain-http origin the camera API itself is withheld — no amount
      // of "fix camera access" helps. Name the way out instead.
      SCAN_STATE.camError = window.isSecureContext === false
        ? 'Camera unavailable — this page was opened over a plain http address, and phones only allow the camera on a secure one. Open ' + SECURE_ADDRESS + ' instead, or scan from a photo.'
        : 'Camera unavailable — ' + cameraErrorText(e) + '. Fix camera access and tap Start camera to retry, or scan from a photo.';
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

  // ── provisioning (card sync-coordinates-provisioning, build-fact 1) ──────
  // The tree's ONLY SYNC_KEY writer. On page init with a session:
  // POST /api/v1/sync/token → 200 → write {restUrl, bearer, deviceId} → arm.
  // On 401 (no session) / 503 (secret-less deploy) / network failure: write
  // NOTHING — the `if (sc)` guard below keeps today's no-sync behavior, the
  // B-436-adjacent degenerate case that deliberately stays. Re-visits
  // re-provision idempotently (startSync's syncHandles guard, measured).
  // The credential is the hq_session cookie riding fetch's same-origin
  // default; the bearer itself is inert at the door (spike fact 1), so there
  // is no client-side refresh machinery to run.
  async function provisionSync() {
    let res;
    try {
      res = await fetch('/api/v1/sync/token', { method: 'POST' });
    } catch (e) { return null; } // offline load — stored coordinates (if any) already armed below
    if (res.status !== 200) return null; // fail closed: write nothing
    let envelope;
    try { envelope = await res.json(); } catch (e) { return null; }
    if (!envelope || !envelope.token || !envelope.sub) return null;
    const cfg = {
      restUrl: '/sync/rest',          // same-origin door (decision 69)
      bearer: envelope.token,
      deviceId: String(envelope.sub), // the push identity — mint sub, nothing else
    };
    try { localStorage.setItem(SYNC_KEY, JSON.stringify(cfg)); } catch (e) { /* storage blocked — still arm in-memory */ }
    return startSync(cfg);
  }

  // Stored coordinates arm sync immediately — an OFFLINE reload of a
  // provisioned device keeps its replicas (the mint above fails without a
  // network and writes nothing; the shipped guard is exactly this line).
  const sc = readJson(SYNC_KEY);
  if (sc) startSync(sc).catch(() => {});
  // Then (re-)provision from the session — unawaited so boot() never blocks
  // on the mint; startSync's guard makes the second arm a no-op.
  provisionSync().catch(() => {});

  return {
    db,
    collections: cols,
    clock,
    resolver,
    scanText: doScan,
    hasherStats: () => hashToken.stats(),
    enqueue,
    // The §8 policy source (card refusal-holds-before-sync) — created at boot,
    // latched to the campaigns replica the moment one starts. submit-flow.js
    // reads it from here rather than constructing its own, so the error latch
    // is in place before the replica can emit.
    campaignPolicy,
    setOnlineProbe: (fn) => { onlineProbe = typeof fn === 'function' ? fn : (() => false); },
    startSync,
    resync,
    scanAgain,
    // Card 6's scan gate hashes the SAME way through the SAME memoized
    // instance — one digest per distinct token page-wide (hasherStats stays
    // the page's single truth about cache behavior).
    hashToken,
    // Card 6's registration point (see the submitFlow block above).
    setSubmitFlow: (handlers) => { submitFlow = handlers && typeof handlers === 'object' ? handlers : null; },
  };
}

const ready = boot().then((api) => {
  Object.assign(window.MarketingScan, api, { booted: true });
  return window.MarketingScan;
}).catch((e) => {
  // Loud, not blank (UI-R3/R6): the Scan section names its failure — in a
  // sentence, with the cause behind "Details" (boot-failure.js).
  reportBootFailure('scanner', e);
  throw e;
});

window.MarketingScan = { booted: false, ready };
