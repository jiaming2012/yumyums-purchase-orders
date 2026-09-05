// marketing/submit-flow.js — the redemption submit flow at the window (card
// redemption-submit-flow, run 20260905; design
// docs/qr-offline-redemption-handoff.md §8/§13/§16, F1/F2/F3/F6, P-KR4;
// docs/ui-design-rules.md UI-R1/2/3/6).
//
// Mounts into Card 5's #scan-submit-slot inside #scan-result and extends the
// SAME Scan section through the two landed contracts: window.MarketingScan
// (serialized enqueue, setOnlineProbe, clock, collections, scanText) and the
// setSubmitFlow registration surface scan-page.js exposes (scan gate, result
// mapping, render hook, action routing through the ONE delegated listener).
//
// The brain is marketing/submit-machine.js — the strict overlay-region XState
// machine (mode 'model' here: an undeclared pair raises the visible,
// retryable unexpectedEvent card, never a dead actor). This file is the
// machine's hands and eyes: it maps resolver results to RESOLVED events,
// renders each machine state into the slot, drives the #13 reachability
// probe, posts the online submit to Card 7's endpoint, and enqueues the §13
// offline override through the serialized wrapper.

import { createSubmitMachine } from './submit-machine.js';
import { createCampaignPolicySource } from './sync/replicas.js';
import { createTokenHasher, extractToken } from './scanner.js';
import {
  validateOrderNumber, toastBusinessDate, createReachabilityProbe, getDeviceId,
  SUBMIT_TIMEOUT_MS,
} from './submit-support.js';

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
const $ = (id) => document.getElementById(id);

const ENTITLEMENT_KEY = 'hq_marketing_entitlement_v1';

async function boot() {
  const MS = await window.MarketingScan.ready;
  const X = window.XState;
  if (!X) {
    // Loud, not blank (UI-R3/R6): the vendored engine failed to load.
    const status = $('scan-status');
    if (status) status.textContent = 'Submit flow failed to start — xstate did not load. Reload to retry.';
    throw new Error('window.XState missing (lib/xstate.umd.min.js not loaded)');
  }

  const DEVICE_ID = getDeviceId(localStorage);

  // ── entitlement (#12): per-user grant `marketing-offline-override`, any
  // role, read from /me/apps. Cached so an offline RELOAD keeps the grant —
  // the override exists FOR offline. No cache + no server -> false
  // (fail-closed). Client-side gating IS the enforcement (Card 7 G6: the
  // server trusts the flag).
  let canOverride = false;
  let ME = null;
  try {
    const res = await fetch('/api/v1/me/apps');
    if (res.ok) {
      const apps = await res.json();
      canOverride = Array.isArray(apps) && apps.some((a) => a && a.slug === 'marketing-offline-override');
      try { localStorage.setItem(ENTITLEMENT_KEY, JSON.stringify({ canOverride })); } catch (e) { /* storage blocked */ }
    } else {
      throw new Error(`me/apps ${res.status}`);
    }
  } catch (e) {
    try {
      const cached = JSON.parse(localStorage.getItem(ENTITLEMENT_KEY));
      canOverride = !!(cached && cached.canOverride);
    } catch (e2) { canOverride = false; }
  }
  try {
    const res = await fetch('/api/v1/me');
    if (res.ok) ME = await res.json();
  } catch (e) { /* offline reload — override_by stays null */ }

  // ── session stashes (page state; the machine owns the flow state) ─────────
  let SUBMIT_CTX = null;        // {token_hash, code_id, value, displayKind, winner?}
  let ORDER_STATE = { raw: '', ok: false, value: '' };
  let WINNER = null;            // {at, by} for the alreadyUsed card
  let SUBMIT_ERROR = null;      // named reason for the error card
  let OVERRIDE_WRITE_ERROR = null;

  function clearSession() {
    SUBMIT_CTX = null;
    ORDER_STATE = { raw: '', ok: false, value: '' };
    WINNER = null;
    SUBMIT_ERROR = null;
    OVERRIDE_WRITE_ERROR = null;
  }

  // ── the machine (production 'model' build — loud, logged, retryable) ──────
  const machine = createSubmitMachine(
    X,
    { canOverride },
    { push: (e) => queueMicrotask(() => handleEffect(e)) },
    {
      mode: 'model',
      onTrip: (info) => {
        // "logged": log.js ships console.error to /api/v1/logs.
        console.error('[marketing submit] undeclared (state,event) pair', JSON.stringify(info));
      },
    },
  );

  // ── campaign policy (§8): the DEFAULT source is the campaigns replica
  // (card requires-online-replication — the refusal arms on real data). The
  // policy source mirrors the local campaigns collection into a sync-readable
  // lookup; setCampaignPolicy stays the injection seam (tests / a later card
  // may override). Unknown campaign -> null -> false: the ratified
  // unknown→false default (decision 166) survives for GENUINELY unknown
  // codes — this card removes "unknown" for replicated campaigns, it does not
  // change what unknown means (unknown -> true would silently delete F2's
  // DECIDED affordance for every code).
  let CAMPAIGN_POLICY = null;
  try {
    if (MS.collections && MS.collections.campaigns) {
      CAMPAIGN_POLICY = createCampaignPolicySource(MS.collections.campaigns).policyFor;
    }
  } catch (e) {
    // A stale-cached scan-page without the campaigns collection degrades to
    // the honest unknown→false default — loudly, never a bricked scanner.
    console.error('[marketing submit] campaign policy source failed to start', e);
    CAMPAIGN_POLICY = null;
  }
  function policyFor(campaignId, offers) {
    if (!CAMPAIGN_POLICY) return false;
    try {
      const p = CAMPAIGN_POLICY(campaignId, offers);
      return !!(p && p.requiresOnline);
    } catch (e) { return false; }
  }

  // The PAGE's memoized hasher (exposed by scan-page.js): one digest per
  // distinct token page-wide, and a same-code re-scan the gate suppresses
  // still registers as a cache HIT — Card 5's hash-caching guarantee stays
  // observable at hasherStats(). Fallback only if an older scan-page is
  // cached without the export.
  const hashToken = typeof MS.hashToken === 'function'
    ? MS.hashToken
    : createTokenHasher({ subtle: crypto.subtle });

  // ── #13 reachability ──────────────────────────────────────────────────────
  function afterReachable() {
    // Refetch what can be refetched, then declare the replica as fresh as
    // this device can make it. With no sync configured (tonight), resync() is
    // a no-op and RESUBSCRIBED is the honest reading; when provisioning
    // lands, RESUBSCRIBED moves to the SUBSCRIBED/initial-replication
    // callback (reportChannelStatus below is the wiring point).
    try { MS.resync(); } catch (e) { /* unconfigured */ }
    machine.send('RESUBSCRIBED'); // P-KR4: the gate auto-resumes on this
  }
  function handleProbe(ok) {
    const conn = machine.conn();
    if (ok) {
      if (conn === 'offline') { machine.send('CONN_UP'); afterReachable(); }
      else if (conn === 'stale') afterReachable();
      // online: steady state — nothing to send
    } else {
      machine.send('PROBE_TIMEOUT'); // while offline: a DECLARED ignore, not a brick
    }
  }
  const probe = createReachabilityProbe({ fetchImpl: (...a) => fetch(...a), onResult: handleProbe });

  // The future realtime-channel wiring point (the slate's onStatus call): a
  // dead channel reads as unreachable; SUBSCRIBED while stale IS the refetch
  // signal.
  function reportChannelStatus(status) {
    if (status === 'SUBSCRIBED') {
      if (machine.conn() === 'stale') machine.send('RESUBSCRIBED');
    } else if (status === 'CHANNEL_ERROR' || status === 'TIMED_OUT' || status === 'CLOSED') {
      machine.send('PROBE_TIMEOUT');
    }
  }

  // Card 5's resolver asks THIS question at scan time (F3's online branch).
  MS.setOnlineProbe(() => machine.conn() === 'online');

  // ── effects ───────────────────────────────────────────────────────────────
  function handleEffect(e) {
    if (!e || !e.type) return;
    if (e.type === 'submitToServer') doOnlineSubmit();
    else if (e.type === 'writeAttempt') doOverrideWrite(e);
  }

  async function lookupWinner(tokenHash) {
    try {
      const docs = await MS.collections.codes.find({ selector: { token_hash: tokenHash } }).exec();
      const redeemed = docs.filter((d) => d.redeemed_at)
        .sort((a, b) => Date.parse(b.redeemed_at) - Date.parse(a.redeemed_at));
      if (redeemed.length) return { at: redeemed[0].redeemed_at, by: redeemed[0].redeemed_by || null };
    } catch (err) { /* replica unreadable — fall through */ }
    return (SUBMIT_CTX && SUBMIT_CTX.winner) || null;
  }

  async function doOnlineSubmit() {
    if (!SUBMIT_CTX) { SUBMIT_ERROR = 'no session'; machine.send('SRV_ERROR'); return; }
    const body = {
      token_hash: SUBMIT_CTX.token_hash,
      device_id: DEVICE_ID,
      order_number: ORDER_STATE.value || null,
      offline_override: false,
      unverified_code: false,
      // offset-true stamp: the sync clock's epoch (never Date.now() as source)
      scanned_at: new Date(MS.clock.now()).toISOString(),
    };
    if (SUBMIT_CTX.value !== null && Number.isFinite(SUBMIT_CTX.value)) body.value = SUBMIT_CTX.value;

    let evt = 'SRV_ERROR';
    try {
      const ctl = typeof AbortController === 'function' ? new AbortController() : null;
      const kill = ctl ? setTimeout(() => ctl.abort(), SUBMIT_TIMEOUT_MS) : null;
      let res;
      try {
        res = await fetch('/api/v1/marketing/redeem', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
          signal: ctl ? ctl.signal : undefined,
        });
      } finally { if (kill) clearTimeout(kill); }
      if (res.status === 200) {
        const out = await res.json();
        evt = {
          redeemed: 'SRV_REDEEMED', already_used: 'SRV_ALREADY_USED',
          expired: 'SRV_EXPIRED', not_found: 'SRV_NOT_FOUND',
        }[out.result] || 'SRV_ERROR';
        if (evt === 'SRV_ERROR') SUBMIT_ERROR = out.error || 'server reported an error';
      } else {
        const out = await res.json().catch(() => ({}));
        SUBMIT_ERROR = out.error ? `${out.error} (HTTP ${res.status})` : `HTTP ${res.status}`;
      }
    } catch (err) {
      SUBMIT_ERROR = (err && err.name === 'AbortError')
        ? 'timed out — the server did not answer'
        : 'network failure — the request never completed';
    }
    if (evt === 'SRV_ALREADY_USED') WINNER = await lookupWinner(SUBMIT_CTX.token_hash);
    machine.send(evt);
  }

  async function doOverrideWrite(e) {
    if (!SUBMIT_CTX) return;
    try {
      // THE serialized enqueue (Card 5's wrapper) — never raw enqueueAttempt.
      await MS.enqueue({
        // engineering call (merge intent): an unknown code's local code_id IS
        // its token_hash — no code row exists to name.
        code_id: SUBMIT_CTX.code_id || SUBMIT_CTX.token_hash,
        device_id: DEVICE_ID,
        offline_override: true,
        override_by: ME ? ((ME.email || (ME.id != null ? String(ME.id) : '')) || null) : null,
        unverified_code: !!e.unverified_code, // F2: true for unknown codes
        pos_order_number: ORDER_STATE.value || null,
        // §13: the Toast-cutoff business date off the SYNC CLOCK's epoch.
        pos_business_date: toastBusinessDate(MS.clock.now()),
        redeemed_value: (SUBMIT_CTX.value !== null && Number.isFinite(SUBMIT_CTX.value)) ? SUBMIT_CTX.value : null,
      }, { now: MS.clock.now });
      OVERRIDE_WRITE_ERROR = null;
    } catch (err) {
      OVERRIDE_WRITE_ERROR = String((err && err.message) || err);
      console.error('[marketing submit] offline-override enqueue failed', err);
    }
    renderAll();
  }

  // ── scan gate + result mapping (F6 session semantics — this card owns them) ──
  async function gate(payload) {
    const token = extractToken(payload);
    const inSession = machine.ctx().sessionCode !== null;
    if (!token) return !inSession; // garbage mid-session never clobbers the customer's card
    const hash = await hashToken(token);
    if (machine.ctx().sc === 'idle') machine.send('SCAN');
    machine.send('QR_DECODED', { code: hash });
    // Proceed only when the machine accepted a FRESH session for this code;
    // same-code re-scans (no-op/re-show), the finish-first prompt, and the
    // mid-submit/mid-confirmation protections all suppress re-resolution.
    return machine.ctx().sc === 'resolving' && machine.ctx().sessionCode === hash;
  }

  async function onResult(result) {
    if (!result || !result.token_hash) return;          // non-codes never reach the machine
    if (machine.ctx().sc !== 'resolving') return;       // the session moved on
    const stash = {
      token_hash: result.token_hash, code_id: null, value: null,
      displayKind: result.kind, winner: null,
    };
    let kind = null;
    let requiresOnline = false;
    switch (result.kind) {
      case 'offerReady': {
        const o = result.offers[0];
        stash.code_id = o.code_id;
        requiresOnline = policyFor(o.campaign_id || null, result.offers);
        kind = 'offerReady';
        break;
      }
      case 'deferToServer':
      case 'spentLocally': {
        stash.winner = { at: result.redeemed_at, by: result.redeemed_by || null };
        try {
          const docs = await MS.collections.codes.find({ selector: { token_hash: result.token_hash } }).exec();
          if (docs.length) {
            stash.code_id = docs[0].id;
            requiresOnline = policyFor(docs[0].campaign_id || null, []);
          }
        } catch (e) { /* replica unreadable — token_hash fallback below */ }
        // ONE authority: the machine's own conn guard reproduces F3 (online →
        // submit path, offline → spentLocally reject).
        kind = 'spentLocally';
        break;
      }
      case 'embeddedOffer':
        // Recorded mapping: for POLICY this is an unknown code (prior use
        // unverifiable; the offer is unauthenticated display data) — the
        // display keeps Card 5's embedded card.
        stash.value = result.offer && Number.isFinite(result.offer.face_value) ? result.offer.face_value : null;
        kind = 'unknownCode';
        break;
      case 'unknownCode':
        kind = 'unknownCode';
        break;
      case 'expiredLocally':
        kind = 'expiredLocally';
        break;
      default:
        return; // unmapped kind: the page mapping is the validity boundary
    }
    SUBMIT_CTX = stash;
    ORDER_STATE = { raw: '', ok: false, value: '' };
    WINNER = null; SUBMIT_ERROR = null; OVERRIDE_WRITE_ERROR = null;
    machine.send('RESOLVED', { kind, requiresOnline });
  }

  // ── rendering ─────────────────────────────────────────────────────────────
  const CONN_COPY = {
    online: 'Online — codes verify at submit',
    offline: "Offline — can't verify codes",
    stale: 'Reconnecting…',
  };

  function ensureConnIndicator() {
    const host = $('scanner-host');
    if (!host) return null;
    let el = $('scan-conn');
    if (!el) {
      el = document.createElement('div');
      el.id = 'scan-conn';
      host.insertBefore(el, host.firstChild);
    }
    return el;
  }

  function renderConn() {
    const el = ensureConnIndicator();
    if (!el) return;
    const conn = machine.conn();
    el.setAttribute('data-conn', conn);
    el.textContent = CONN_COPY[conn] || conn;
  }

  const ORDER_STATES = new Set(['offerReady', 'readyToSubmit', 'unknownCode', 'blockedOffline', 'overrideConfirm']);

  function orderFieldHtml() {
    const showErr = ORDER_STATE.raw.trim() !== '' && !ORDER_STATE.ok;
    return `<div class="ms-field">
      <label for="ms-order">Toast order # <span class="ms-req">required to complete</span></label>
      <input id="ms-order" inputmode="numeric" autocomplete="off" placeholder="e.g. 4321" value="${esc(ORDER_STATE.raw)}">
      <div id="ms-order-err" class="ms-err"${showErr ? '' : ' hidden'}>Order numbers are 1–6 digits — check the Toast check number.</div>
    </div>`;
  }

  function gateBranch() {
    if (machine.flags().overrideAvailable) return 'override';
    if (machine.ctx().requiresOnline) return 'requires-online';
    return 'no-permission';
  }

  function flowHtml(sc) {
    const next = '<button class="ms-btn ms-btn-quiet" data-action="ms-next">Next customer</button>';
    switch (sc) {
      case 'offerReady':
      case 'readyToSubmit': {
        const armed = sc === 'readyToSubmit' && ORDER_STATE.ok;
        return `${orderFieldHtml()}
          <button class="ms-btn ms-btn-go" data-action="ms-submit"${armed ? '' : ' disabled'}>Submit redemption</button>
          <div class="ms-note">Apply the discount in Toast, then enter the order # — that completes the redemption.</div>`;
      }
      case 'unknownCode':
        return `${orderFieldHtml()}
          <button class="ms-btn ms-btn-go" data-action="ms-submit"${ORDER_STATE.ok ? '' : ' disabled'}>Submit — server will verify</button>
          <div class="ms-note">This code isn&#39;t on this device — the server has the final say at submit.</div>`;
      case 'blockedOffline': {
        const branch = gateBranch();
        const body = branch === 'requires-online'
          ? `<div class="ms-gate-head">Can&#39;t verify — try again in a moment.</div>
             <div class="ms-note">High-value offer: online verification is <b>required</b>. There is no offline override for this campaign (§8) — not even for a manager.</div>`
          : branch === 'override'
            ? `<div class="ms-gate-head">Offline — can&#39;t verify this code right now.</div>
               <div class="ms-note">You hold the offline-override permission. Forcing it risks a double-redemption and is flagged for review.</div>
               <button class="ms-btn ms-btn-warn" data-action="ms-override">Force submit (offline)</button>`
            : `<div class="ms-gate-head">Can&#39;t verify — connect to redeem.</div>
               <div class="ms-note">Submit needs the server. Ask a manager if this can&#39;t wait.</div>`;
        return `${orderFieldHtml()}
          <div id="ms-gate" data-branch="${branch}">${body}
          <div class="ms-note ms-auto">Submit re-enables itself the moment the connection returns.</div></div>`;
      }
      case 'overrideConfirm': {
        const unverified = machine.flags().unverifiedWarning
          ? '<div class="ms-warn-line"><b>Neither the offer nor prior use can be verified</b> for this code on this device.</div>'
          : '';
        return `${orderFieldHtml()}
          <div id="ms-confirm">
            <div class="ms-gate-head">Are you sure?</div>
            ${unverified}
            <div class="ms-note">Confirming this coupon while your device is offline risks a <b>double-redemption that can&#39;t be undone</b>. The attempt is recorded and reconciled first (§9).</div>
            <button class="ms-btn ms-btn-warn" data-action="ms-confirm-override"${ORDER_STATE.ok ? '' : ' disabled'}>Yes — force submit</button>
            <button class="ms-btn ms-btn-quiet" data-action="ms-cancel-override">Cancel</button>
          </div>`;
      }
      case 'overridePending':
        return `<div class="ms-card ms-card-warn">
          <div class="ms-head">Queued — will verify on reconnect</div>
          <div class="ms-note">Order #${esc(ORDER_STATE.value || '—')} recorded with the override flag. The server arbitrates it on the next sync${OVERRIDE_WRITE_ERROR ? ` — <b>local write failed: ${esc(OVERRIDE_WRITE_ERROR)}</b>` : ''}.</div>
          ${next}</div>`;
      case 'submitting':
        return `${orderFieldHtml()}<button class="ms-btn ms-btn-go" disabled>Submitting…</button>`;
      case 'redeemed':
        return `<div class="ms-card ms-card-ok">
          <div class="ms-head">Redeemed ✓</div>
          <div class="ms-note">Order #${esc(ORDER_STATE.value || '—')} recorded. Hand over the goods.</div>
          ${next}</div>`;
      case 'alreadyUsed': {
        const w = WINNER || (SUBMIT_CTX && SUBMIT_CTX.winner) || null;
        const detail = w
          ? `at ${esc(fmtWhen(w.at))}${w.by ? ` by ${esc(w.by)}` : ''}`
          : 'by another device (details arrive with the next sync)';
        return `<div class="ms-card ms-card-bad">
          <div class="ms-head">Already used</div>
          <div class="ms-note">The server refused this code — redeemed ${detail}.</div>
          ${next}</div>`;
      }
      case 'expired':
        return `<div class="ms-card ms-card-bad"><div class="ms-head">Expired</div>
          <div class="ms-note">The server refused this code: past its expiry.</div>${next}</div>`;
      case 'notFound':
        return `<div class="ms-card ms-card-bad"><div class="ms-head">Not found</div>
          <div class="ms-note">The server doesn&#39;t know this code. Check it&#39;s a Yumyums QR.</div>${next}</div>`;
      case 'error':
        return `<div class="ms-card ms-card-bad" id="ms-error">
          <div class="ms-head">Submit failed</div>
          <div class="ms-note">${esc(SUBMIT_ERROR || 'unknown failure')}. Nothing was redeemed.</div>
          <button class="ms-btn ms-btn-go" data-action="ms-retry">Retry submit</button>
          ${next}</div>`;
      case 'unexpectedEvent': {
        const info = machine.ctx().unexpectedInfo || {};
        return `<div class="ms-card ms-card-bad" id="ms-unexpected">
          <div class="ms-head">Unexpected scanner event</div>
          <div class="ms-note">The screen hit an event it never expected (<code>${esc(info.event || '?')}</code> while <code>${esc(info.scan || '?')}</code>). It was logged. Retry to continue where you were.</div>
          <button class="ms-btn ms-btn-go" data-action="ms-retry">Retry</button>
          ${next}</div>`;
      }
      case 'spentLocally':
        return `<div class="ms-note">Connection dropped — this device&#39;s copy shows the code already used, so submit is closed.</div>${next}`;
      default:
        return '';
    }
  }

  function renderSlot() {
    const sc = machine.ctx().sc;
    let slot = $('scan-submit-slot');
    if (!slot) {
      if (sc !== 'unexpectedEvent') return;
      // The error card must be visible even when the result card carries no
      // slot (loud beats tidy — UI-R3/R6).
      const host = $('scanner-host');
      if (!host) return;
      slot = document.createElement('div');
      slot.id = 'scan-submit-slot';
      host.appendChild(slot);
    }
    // Preserve the order entry across repaints (state-first rendering would
    // otherwise eat the caret mid-typing).
    const prev = $('ms-order');
    const hadFocus = prev && document.activeElement === prev;
    const caret = hadFocus ? prev.selectionStart : null;
    slot.innerHTML = `<div id="ms-flow" data-mstate="${esc(sc)}">${flowHtml(sc)}</div>`;
    if (hadFocus) {
      const input = $('ms-order');
      if (input) {
        input.focus();
        if (caret !== null) { try { input.setSelectionRange(caret, caret); } catch (e) { /* non-text state */ } }
      }
    }
  }

  function renderPrompt() {
    const open = machine.scan() === 'promptFinishCurrent';
    let el = $('scan-prompt');
    if (!open) { if (el) el.remove(); return; }
    if (!el) {
      el = document.createElement('div');
      el.id = 'scan-prompt';
      const host = $('scanner-host');
      if (!host) return;
      host.appendChild(el);
    }
    el.innerHTML = `<div class="ms-sheet">
      <div class="ms-head">Finish the current customer first</div>
      <div class="ms-note">A different code was scanned mid-redemption. Wrap up this customer, then scan the next one.</div>
      <button class="ms-btn ms-btn-go" data-action="ms-dismiss">Back to current customer</button>
      <button class="ms-btn ms-btn-quiet" data-action="ms-prompt-next">Start next customer</button>
    </div>`;
  }

  function renderAll() {
    renderConn();
    renderSlot();
    renderPrompt();
  }

  // Light-touch update on keystrokes (no repaint — the input keeps focus).
  function updateOrderUi() {
    const err = $('ms-order-err');
    if (err) err.hidden = !(ORDER_STATE.raw.trim() !== '' && !ORDER_STATE.ok);
    const sc = machine.ctx().sc;
    const submit = document.querySelector('[data-action="ms-submit"]');
    if (submit) {
      const armed = ORDER_STATE.ok && (sc === 'readyToSubmit' || sc === 'unknownCode');
      if (armed) submit.removeAttribute('disabled');
      else submit.setAttribute('disabled', '');
    }
    const confirm = document.querySelector('[data-action="ms-confirm-override"]');
    if (confirm) {
      if (ORDER_STATE.ok) confirm.removeAttribute('disabled');
      else confirm.setAttribute('disabled', '');
    }
  }

  // ── machine → UI ──────────────────────────────────────────────────────────
  let lastSc = machine.ctx().sc;
  machine.onChange(() => {
    const sc = machine.ctx().sc;
    if (sc === 'idle' && lastSc !== 'idle') clearSession();
    lastSc = sc;
    renderAll();
  });

  // ── registration: ONE surface on Card 5's delegated listeners ─────────────
  const dispatchOrValidate = (eventName) => {
    const v = validateOrderNumber(ORDER_STATE.raw);
    if (!v.ok) {
      ORDER_STATE.ok = false;
      updateOrderUi();
      const err = $('ms-order-err');
      if (err) err.hidden = false; // §13: the order number completes the redemption
      return;
    }
    ORDER_STATE.ok = true;
    ORDER_STATE.value = v.value;
    machine.send(eventName);
  };

  MS.setSubmitFlow({
    gate,
    onResult,
    onRender: renderAll,
    onScanAgain: () => {
      if (machine.ctx().sc !== 'idle') machine.send('NEXT_CUSTOMER');
    },
    onInput: (e) => {
      if (!e.target || e.target.id !== 'ms-order') return;
      ORDER_STATE.raw = e.target.value;
      const v = validateOrderNumber(ORDER_STATE.raw);
      ORDER_STATE.ok = v.ok;
      ORDER_STATE.value = v.ok ? v.value : '';
      if (v.ok && machine.ctx().sc === 'offerReady') machine.send('ORDER_OK');
      else updateOrderUi();
    },
    onChange: () => { /* no change-shaped controls yet */ },
    actions: {
      'ms-submit': () => dispatchOrValidate('SUBMIT'),
      'ms-override': () => machine.send('OVERRIDE_REQUEST'),
      'ms-confirm-override': () => dispatchOrValidate('OVERRIDE_CONFIRM'),
      'ms-cancel-override': () => machine.send('OVERRIDE_CANCEL'),
      'ms-retry': () => machine.send('RETRY'),
      'ms-dismiss': () => machine.send('DISMISS'),
      'ms-next': () => { machine.send('NEXT_CUSTOMER'); MS.scanAgain(); },
      'ms-prompt-next': () => { machine.send('NEXT_CUSTOMER'); MS.scanAgain(); },
    },
  });

  // ── boot the reachability signal: never claim liveness before a probe ─────
  machine.send('PROBE_TIMEOUT');   // honest start: unverified reads offline
  renderAll();
  await probe.probeNow();          // first verdict before booted flips
  probe.start();

  return {
    machine,
    probeNow: () => probe.probeNow(),
    stopProbe: () => probe.stop(),
    setCampaignPolicy: (fn) => { CAMPAIGN_POLICY = typeof fn === 'function' ? fn : null; },
    // Debug/test read of the ACTIVE policy (replica-fed default, or whatever
    // setCampaignPolicy injected): campaignId → {requiresOnline} | null.
    campaignPolicyFor: (campaignId) => {
      if (!CAMPAIGN_POLICY) return null;
      try { return CAMPAIGN_POLICY(campaignId) || null; } catch (e) { return null; }
    },
    reportChannelStatus,
    deviceId: DEVICE_ID,
    canOverride,
  };
}

const ready = boot().then((api) => {
  Object.assign(window.MarketingSubmit, api, { booted: true });
  return window.MarketingSubmit;
}).catch((e) => {
  const status = document.getElementById('scan-status');
  if (status) status.textContent = 'Submit flow failed to start — reload to retry. (' + (e && e.message ? e.message : e) + ')';
  throw e;
});

window.MarketingSubmit = { booted: false, ready };
