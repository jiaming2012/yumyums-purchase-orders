// marketing/campaigns.js — the Campaigns section of marketing.html (#s2).
// Card `h2-campaigns-tab-ui`, run 20261002, roadmap H2.
// Design of record: *Current Campaigns 1–8*; contract: the handoff
// docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md §5, served by
// backend/internal/marketing (card H1).
//
// ── What this module owns ────────────────────────────────────────────────────
// #s2 and nothing else. It does not touch the shared head, the tab bar, the
// inline show(), #s1 (Scan), #s3 (Subscribers, card H5) or #s4 (the
// reconciliation queue, card H4). Every selector it reads or writes is under
// #mc-root, and every class it styles is prefixed `mc-`.
//
// ── Conventions this follows (CLAUDE.md) ────────────────────────────────────
//   * STATE-FIRST RENDERING. Mutate S, call render(). No handler reaches into
//     the DOM to patch a node; the DOM is a projection of S, always.
//   * ONE delegated click listener and ONE delegated input listener on
//     #mc-root, routed by `data-action`. No inline onclick on anything
//     dynamic.
//   * The frontend speaks the backend's field names (UI-R4): every key read
//     below is a json tag on a struct in backend/internal/marketing/types.go.
//     `money.per_dollar` and `money.avg_order_cents_*` are deliberately NULL
//     rather than absent in that contract, so null is rendered as "—" and
//     never coerced to 0 (UI-R3).
//
// ── The money block is rendered, not computed ───────────────────────────────
// H1 ships `money` as the stated ZERO SHAPE ({revenue_cents:0,
// discount_cents:0, discount_basis:"implied", net_cents:0, per_dollar:null,
// avg_order_cents_with:null, avg_order_cents_without:null}); card H3b lands the
// arithmetic. This module renders whatever shape arrives, with every key
// present, so the strip does not change when the numbers become real.
//
// ── Share, and why the fallback is not optional ─────────────────────────────
// `navigator.share({files})` is iOS 15+; headless desktop Chromium exposes
// NEITHER `navigator.share` NOR `navigator.canShare` (spike
// `web-share-files-enumerated`, chromium row, 2026-10-01; webkit unmeasured on
// that machine — GAP-H2-1). So Share is feature-detected AT SHEET-OPEN TIME
// with a real File (canShare's contract is per-payload, not per-API), and when
// the probe says no the sheet renders Save PNG + Copy link as the primary
// actions instead. Both branches ship; neither is a guess.
(function () {
  'use strict';

  var API = '/api/v1/marketing';
  // The item catalog read is the inventory app's (GET
  // /api/v1/inventory/menu-items, toast.ListMenuItemsHandler). A manager with
  // the `marketing` grant but not `inventory` is refused there — see
  // loadItems(): the Item field stays visible and says so rather than
  // vanishing.
  var ITEMS_URL = '/api/v1/inventory/menu-items?since=2000-01-01';
  var CACHE_PREFIX = 'hq.mc.list.v1:';
  // Offer text longer than this truncates in the list; tapping shows it whole
  // (the State Enumeration "long content" row).
  var OFFER_CLAMP = 40;

  // Channel enum == migration 0083's CHECK == `channels` in
  // backend/internal/marketing/codes.go. Order is the order the chips render.
  var CHANNELS = [
    ['truck_sign', 'Truck sign'],
    ['flyer', 'Flyer'],
    ['table_tent', 'Table tent'],
    ['menu_board', 'Menu board'],
    ['instagram', 'Instagram'],
    ['google_ads', 'Google Ads'],
    ['receipt', 'Receipt'],
    ['sms', 'SMS'],
    ['other', 'Other']
  ];
  var LANDINGS = [
    ['signup', 'Signup form'],
    ['menu', 'Menu'],
    ['offer', 'Offer page'],
    ['directions', 'Directions']
  ];

  // ── state ────────────────────────────────────────────────────────────────
  var S = {
    view: 'list',        // list | create | ready | detail | code
    status: 'idle',      // idle | loading | ok | empty | error | locked | offline
    period: '30d',
    campaigns: [],
    syncedAt: null,      // ISO stamp when the rows came out of the cache
    error: '',
    expanded: {},        // campaign id -> offer text shown in full
    draft: null,
    items: [],
    itemsState: 'idle',  // idle | loading | ok | locked
    saving: false,
    formError: '',
    created: null,       // {campaign, codes, warnings} — the "N codes ready" screen
    detail: null,        // one campaignDTO
    detailError: '',
    code: null,          // one codeDTO
    canShareFiles: false,
    repoint: false,
    note: ''             // transient line on the code sheet
  };

  function newDraft() {
    return {
      name: '', offer: '', value: '', days: '14',
      item: '', landing: 'signup', channels: [], otherLabel: ''
    };
  }

  // ── small helpers ────────────────────────────────────────────────────────
  function esc(s) {
    return String(s == null ? '' : s)
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
  }
  // money() renders integer cents. A null/undefined amount is "—", never
  // "$0.00": the contract distinguishes "zero" from "no opinion" (UI-R3).
  function money(cents) {
    if (cents == null) return '—';
    var neg = cents < 0;
    var v = Math.abs(cents) / 100;
    return (neg ? '-$' : '$') + v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  }
  function perDollar(v) {
    return v == null ? '—' : '$' + Number(v).toFixed(2);
  }
  function channelLabel(code) {
    if (code.channel === 'other' && code.channel_label) return code.channel_label;
    for (var i = 0; i < CHANNELS.length; i++) if (CHANNELS[i][0] === code.channel) return CHANNELS[i][1];
    return code.channel || 'Channel';
  }
  function landingLabel(v) {
    for (var i = 0; i < LANDINGS.length; i++) if (LANDINGS[i][0] === v) return LANDINGS[i][1];
    return v || '—';
  }
  function day(iso) {
    if (!iso) return '—';
    var d = new Date(iso);
    if (isNaN(d)) return '—';
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  }
  function clock(iso) {
    if (!iso) return 'never';
    var d = new Date(iso);
    if (isNaN(d)) return 'never';
    var today = new Date();
    var t = d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
    if (d.toDateString() === today.toDateString()) return t;
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }) + ' ' + t;
  }
  function $(id) { return document.getElementById(id); }

  // ── cache (the offline row's "last synced" list) ─────────────────────────
  //
  // Per-period, because the periods are different answers to different
  // questions and a 7d list standing in for a 90d one would be a quiet lie.
  function writeCache(period, rows) {
    try {
      localStorage.setItem(CACHE_PREFIX + period,
        JSON.stringify({ at: new Date().toISOString(), campaigns: rows }));
    } catch (e) { /* private mode / quota — the cache is a convenience */ }
  }
  function readCache(period) {
    try {
      var raw = localStorage.getItem(CACHE_PREFIX + period);
      if (!raw) return null;
      var v = JSON.parse(raw);
      return v && Array.isArray(v.campaigns) ? v : null;
    } catch (e) { return null; }
  }

  // ── transport ────────────────────────────────────────────────────────────
  //
  // A refused request carries its status and the server's error slug; a request
  // the network never completed carries NO status, and that is how offline is
  // told apart from a 500 (the two State Enumeration rows look nothing alike).
  function req(path, opts) {
    var o = opts || {};
    return fetch(API + path, {
      method: o.method || 'GET',
      headers: o.body ? { 'Content-Type': 'application/json' } : undefined,
      body: o.body ? JSON.stringify(o.body) : undefined
    }).then(function (res) {
      return res.text().then(function (text) {
        var body = null;
        if (text) { try { body = JSON.parse(text); } catch (e) { body = null; } }
        if (!res.ok) {
          var err = new Error((body && body.error) || ('http_' + res.status));
          err.status = res.status;
          err.body = body;
          throw err;
        }
        return body;
      });
    });
  }
  function isNetworkDown(err) { return !err || !err.status; }

  // ── loads ────────────────────────────────────────────────────────────────
  function loadList() {
    S.status = 'loading';
    S.error = '';
    render();
    return req('/campaigns?period=' + encodeURIComponent(S.period)).then(function (body) {
      S.campaigns = (body && body.campaigns) || [];
      S.syncedAt = null;
      S.status = S.campaigns.length ? 'ok' : 'empty';
      writeCache(S.period, S.campaigns);
      render();
    }).catch(function (err) {
      if (err.status === 403 || err.status === 401) {
        S.status = 'locked';
        S.campaigns = [];
        render();
        return;
      }
      if (isNetworkDown(err)) {
        var cached = readCache(S.period);
        S.campaigns = cached ? cached.campaigns : [];
        S.syncedAt = cached ? cached.at : null;
        S.status = 'offline';
        render();
        return;
      }
      S.status = 'error';
      S.error = err.message || 'load_failed';
      console.error('campaigns: list failed', err.message);
      render();
    });
  }

  // loadItems is best-effort by design. The Item field is part of the create
  // sheet in the design of record, and the catalog read lives behind the
  // `inventory` grant — so a marketing-only manager gets the field with an
  // honest "needs Inventory access" line instead of a silently missing control
  // (UI-R3: a blank slot is a defect) or an invented endpoint.
  function loadItems() {
    if (S.itemsState !== 'idle') return;
    S.itemsState = 'loading';
    fetch(ITEMS_URL).then(function (res) {
      if (!res.ok) throw new Error('http_' + res.status);
      return res.json();
    }).then(function (rows) {
      S.items = Array.isArray(rows) ? rows : [];
      S.itemsState = 'ok';
      render();
    }).catch(function () {
      S.items = [];
      S.itemsState = 'locked';
      render();
    });
  }

  function loadDetail(id) {
    S.view = 'detail';
    S.detail = null;
    S.detailError = '';
    render();
    return req('/campaigns/' + encodeURIComponent(id) + '?period=' + encodeURIComponent(S.period))
      .then(function (c) { S.detail = c; render(); })
      .catch(function (err) {
        S.detailError = isNetworkDown(err) ? 'offline' : (err.message || 'load_failed');
        render();
      });
  }

  // ── Web Share probe ──────────────────────────────────────────────────────
  //
  // Probed with a REAL File at sheet-open time: canShare's contract is about
  // the payload, not the API's existence, and a browser can expose share()
  // while refusing files.
  function detectShareFiles() {
    try {
      if (typeof navigator.share !== 'function') return false;
      if (typeof navigator.canShare !== 'function') return false;
      var probe = new File([new Uint8Array([137, 80, 78, 71])], 'probe.png', { type: 'image/png' });
      return navigator.canShare({ files: [probe] }) === true;
    } catch (e) { return false; }
  }

  function pngFile(code, size) {
    return fetch(code.png_url + '?size=' + (size || 1024)).then(function (res) {
      if (!res.ok) throw new Error('png_' + res.status);
      return res.blob();
    }).then(function (blob) {
      return new File([blob], 'yumyums-' + code.short + '.png', { type: 'image/png' });
    });
  }

  // ── render ───────────────────────────────────────────────────────────────
  var VIEWS = ['list', 'create', 'ready', 'detail', 'code'];

  function render() {
    var root = $('mc-root');
    if (!root) return;
    root.dataset.view = S.view;
    root.dataset.status = S.status;

    VIEWS.forEach(function (v) {
      var el = $('mc-view-' + v);
      if (el) el.hidden = v !== S.view;
    });

    renderBanner();
    if (S.view === 'list') renderList();
    else if (S.view === 'create') renderCreate();
    else if (S.view === 'ready') renderReady();
    else if (S.view === 'detail') renderDetail();
    else if (S.view === 'code') renderCode();
  }

  // The banner is the one error/offline surface, and it is always retryable
  // (UI-R6). It is suppressed on the Locked row: a permission refusal is not a
  // failure to retry, and offering Retry there would promise a second answer.
  function renderBanner() {
    var b = $('mc-banner');
    if (!b) return;
    if (S.view !== 'list' || (S.status !== 'error' && S.status !== 'offline')) {
      b.hidden = true;
      b.innerHTML = '';
      return;
    }
    b.hidden = false;
    if (S.status === 'offline') {
      b.className = 'mc-banner mc-banner-warn';
      b.innerHTML =
        '<div class="mc-banner-t">Offline &mdash; showing the last synced list</div>' +
        '<div class="mc-banner-s">Last synced ' + esc(clock(S.syncedAt)) +
        (S.campaigns.length ? '' : ' &middot; nothing synced on this phone yet') + '</div>' +
        '<button class="mc-btn mc-btn-quiet" data-action="reload">Retry</button>';
    } else {
      b.className = 'mc-banner mc-banner-bad';
      b.innerHTML =
        '<div class="mc-banner-t">Couldn&rsquo;t load campaigns</div>' +
        '<div class="mc-banner-s">' + esc(S.error) + '</div>' +
        '<button class="mc-btn mc-btn-quiet" data-action="reload">Retry</button>';
    }
  }

  function renderList() {
    var host = $('mc-view-list');
    if (!host) return;

    if (S.status === 'locked') {
      host.innerHTML =
        '<div class="card" id="mc-locked">' +
        '<div class="hd"><h1>Managers only</h1>' +
        '<div class="sub">Campaigns and codes are manager-level</div></div>' +
        '<div class="bd">Creating offers, minting QR codes and reading campaign money are ' +
        'manager-tier actions. Ask an admin to make you a manager if you need them.</div>' +
        '</div>';
      return;
    }

    var offline = S.status === 'offline';
    var html =
      '<div class="mc-chrome">' +
      '<div class="mc-seg" id="mc-period" role="tablist" aria-label="Period">' +
      ['7d', '30d', '90d', 'all'].map(function (p) {
        return '<button role="tab" aria-selected="' + (S.period === p) + '"' +
          ' class="' + (S.period === p ? 'on' : '') + '"' +
          ' data-action="period" data-period="' + p + '">' +
          (p === 'all' ? 'All' : p) + '</button>';
      }).join('') +
      '</div>' +
      '<button class="mc-btn mc-btn-quiet mc-icon" id="mc-refresh" data-action="reload" ' +
      'aria-label="Refresh">&#8635;</button>' +
      '</div>' +
      '<button class="mc-btn mc-btn-go" id="mc-new" data-action="new"' +
      (offline ? ' disabled title="A campaign cannot be created offline"' : '') +
      '>&#43; New campaign</button>';

    if (S.status === 'loading') {
      html += '<div id="mc-list" class="mc-cards" aria-busy="true">' +
        '<div class="mc-skel"></div><div class="mc-skel"></div><div class="mc-skel"></div></div>';
      host.innerHTML = html;
      return;
    }

    // "No campaigns yet" is a FACT the server stated, so it is rendered only
    // when the server stated it. On error — and on offline with nothing cached
    // — we do not know whether there are campaigns, and the banner above
    // already says what happened; printing "No campaigns yet" there would be
    // the app asserting something it has no basis for. (Caught by reading the
    // first error.png back: the red banner and "No campaigns yet" sat on screen
    // together, which reads as "the load failed AND you have none".)
    if (S.status === 'empty') {
      html += '<div class="card mc-empty" id="mc-empty"><div class="bd">' +
        '<div class="mc-empty-h">No campaigns yet</div>' +
        '<div>A campaign is an offer plus the places it will be seen. Saving one mints ' +
        'a QR code per channel, and the codes are what make scans, signups and ' +
        'redemptions countable.</div>' +
        '</div></div>';
      host.innerHTML = html;
      return;
    }

    // error / offline-with-no-cache fall through here with nothing to show.
    // The banner carries the whole message; #mc-list is not rendered at all so
    // there is no empty container pretending to be a list.
    if (S.campaigns.length) {
      html += '<div id="mc-list" class="mc-cards">' +
        S.campaigns.map(campaignCard).join('') + '</div>';
    }
    host.innerHTML = html;
  }

  function statusPill(c) {
    var cls = { live: 'ok', paused: 'warn', ended: 'mut', scheduled: 'info' }[c.status] || 'mut';
    return '<span class="mc-pill mc-pill-' + cls + '">' + esc(c.status || 'unknown') + '</span>';
  }

  function campaignCard(c) {
    var full = c.offer_text || '';
    var long = full.length > OFFER_CLAMP;
    var open = !!S.expanded[c.id];
    var shown = long && !open ? full.slice(0, OFFER_CLAMP).replace(/\s+$/, '') + '…' : full;
    var codes = c.codes || [];
    var m = c.money || {};
    var f = c.funnel || {};

    return '<div class="card mc-card" data-id="' + esc(c.id) + '">' +
      '<div class="mc-top" data-action="open-detail" data-id="' + esc(c.id) + '" role="button" tabindex="0">' +
      '<div class="mc-name">' + esc(c.name) + '</div>' +
      '<div class="mc-pills">' + statusPill(c) +
      (c.projected_at == null
        ? '<span class="mc-pill mc-pill-np">Not on tablets yet</span>' : '') +
      '</div>' +
      '</div>' +
      '<div class="mc-offer' + (long ? ' mc-offer-tap' : '') + '"' +
      (long ? ' data-action="offer" data-id="' + esc(c.id) + '" role="button" tabindex="0"' +
        ' aria-expanded="' + open + '"' : '') +
      '>' + esc(shown) +
      (long ? '<span class="mc-more">' + (open ? 'less' : 'more') + '</span>' : '') +
      '</div>' +
      '<div class="mc-meta">' + esc(money(c.face_value_cents)) + ' off' +
      (c.item && c.item.name ? ' &middot; ' + esc(c.item.name) : ' &middot; any item') +
      ' &middot; ends ' + esc(day(c.ends_at)) +
      ' &middot; ' + codes.length + (codes.length === 1 ? ' code' : ' codes') +
      '</div>' +

      '<div class="mc-funnel">' +
      leg('scans', 'Scans', f.scans) +
      '<span class="mc-arrow" aria-hidden="true">&rsaquo;</span>' +
      leg('signups', 'Signups', f.signups) +
      '<span class="mc-arrow" aria-hidden="true">&rsaquo;</span>' +
      leg('redeemed', 'Redeemed', f.redeemed) +
      '</div>' +

      '<div class="mc-money">' +
      cell('revenue', 'Revenue', money(m.revenue_cents)) +
      cell('discount', 'Discount', money(m.discount_cents)) +
      cell('net', 'Net', money(m.net_cents)) +
      '<span class="mc-per">Per $1 ' + esc(perDollar(m.per_dollar)) + '</span>' +
      '</div>' +
      (m.discount_basis
        ? '<div class="mc-basis">discount ' + esc(m.discount_basis) + '</div>' : '') +
      '</div>';
  }
  function leg(key, label, n) {
    return '<span class="mc-leg" data-leg="' + key + '">' +
      '<span class="mc-n">' + (n == null ? '—' : esc(n)) + '</span>' +
      '<span class="mc-l">' + label + '</span></span>';
  }
  function cell(key, label, v) {
    return '<span class="mc-cell" data-m="' + key + '">' +
      '<span class="mc-l">' + label + '</span>' +
      '<span class="mc-v">' + esc(v) + '</span></span>';
  }

  // ── create: ONE sheet (Current Campaigns 2) ──────────────────────────────
  function renderCreate() {
    var host = $('mc-view-create');
    if (!host) return;
    var d = S.draft || (S.draft = newDraft());
    var n = d.channels.length;

    var itemField;
    if (S.itemsState === 'locked') {
      itemField = '<select id="mc-f-item" disabled><option value="">Any item</option></select>' +
        '<div class="mc-hint">Picking a dish needs Inventory access &mdash; this offer will ' +
        'count against any item.</div>';
    } else if (S.itemsState === 'loading' || S.itemsState === 'idle') {
      itemField = '<select id="mc-f-item" disabled><option value="">Loading dishes…</option></select>';
    } else {
      itemField = '<select id="mc-f-item" data-field="item">' +
        '<option value=""' + (d.item ? '' : ' selected') + '>Any item</option>' +
        S.items.map(function (it) {
          return '<option value="' + esc(it.id) + '"' + (d.item === it.id ? ' selected' : '') + '>' +
            esc(it.name) + '</option>';
        }).join('') + '</select>' +
        (S.items.length ? '' : '<div class="mc-hint">No Toast dishes on file yet.</div>');
    }

    host.innerHTML =
      '<button class="back mc-back" data-action="to-list">Campaigns</button>' +
      '<div class="card"><div class="hd"><h1>New campaign</h1>' +
      '<div class="sub">One sheet. Saving mints one QR code per channel.</div></div>' +
      '<div class="bd mc-form">' +

      field('Name', '<input id="mc-f-name" data-field="name" type="text" ' +
        'placeholder="Wing Wednesday" value="' + esc(d.name) + '">') +
      field('Offer', '<input id="mc-f-offer" data-field="offer" type="text" ' +
        'placeholder="$2 off any 6pc wings" value="' + esc(d.offer) + '">') +

      '<div class="mc-two">' +
      field('Value', '<div class="mc-money-in"><span>$</span>' +
        '<input id="mc-f-value" data-field="value" type="text" inputmode="decimal" ' +
        'placeholder="2.00" value="' + esc(d.value) + '"></div>' +
        '<div class="mc-hint">What the offer is worth. The discount math needs it; ' +
        'nothing is guessed from the offer text.</div>') +
      field('Runs for', '<div class="mc-money-in">' +
        '<input id="mc-f-days" data-field="days" type="text" inputmode="numeric" ' +
        'value="' + esc(d.days) + '"><span>days</span></div>') +
      '</div>' +

      field('Item', itemField) +
      field('Code opens', '<select id="mc-f-landing" data-field="landing">' +
        LANDINGS.map(function (l) {
          return '<option value="' + l[0] + '"' + (d.landing === l[0] ? ' selected' : '') + '>' +
            l[1] + '</option>';
        }).join('') + '</select>') +

      '<div class="mc-field"><label>Channels</label>' +
      '<div id="mc-chips" class="mc-chips">' +
      CHANNELS.map(function (c) {
        var on = d.channels.indexOf(c[0]) !== -1;
        return '<button class="mc-chip' + (on ? ' on' : '') + '" data-action="chip" ' +
          'data-channel="' + c[0] + '" aria-pressed="' + on + '">' + c[1] + '</button>';
      }).join('') +
      '</div>' +
      (d.channels.indexOf('other') !== -1
        ? '<input id="mc-f-other" data-field="otherLabel" type="text" ' +
        'placeholder="Name the other channel" value="' + esc(d.otherLabel) + '">' : '') +
      '<div class="mc-hint">One code per channel, so &ldquo;best channel&rdquo; is ' +
      'countable instead of guessed.</div>' +
      '</div>' +

      '<div class="mc-field"><label>Each code will open</label>' +
      '<div id="mc-preview" class="mc-preview">' +
      (n
        ? d.channels.map(function (ch) {
          var lbl = ch === 'other' ? (d.otherLabel || 'Other') : channelLabel({ channel: ch });
          return '<div class="mc-preview-row"><span class="mc-preview-ch">' + esc(lbl) + '</span>' +
            '<code>hq.yumyums.kitchen/q/&middot;&middot;&middot;&middot;&middot;&middot;</code></div>';
        }).join('')
        : '<div class="mc-preview-row mut">Pick a channel to see the link each code carries.</div>') +
      '</div>' +
      '<div class="mc-hint">The short code is drawn when you save, and the printed code ' +
      'can be re-pointed later without a reprint.</div>' +
      '</div>' +

      (S.formError ? '<div class="mc-err" id="mc-form-err">' + esc(S.formError) + '</div>' : '') +

      '<button class="mc-btn mc-btn-go" id="mc-submit" data-action="create"' +
      (S.saving || !n ? ' disabled' : '') + '>' +
      (S.saving ? 'Creating…' : 'Create campaign + ' + n + (n === 1 ? ' code' : ' codes')) +
      '</button>' +
      '</div></div>';
  }
  function field(label, inner) {
    return '<div class="mc-field"><label>' + label + '</label>' + inner + '</div>';
  }

  // ── "N codes ready" (Current Campaigns 3) ────────────────────────────────
  function renderReady() {
    var host = $('mc-view-ready');
    if (!host) return;
    var made = S.created;
    if (!made) { host.innerHTML = ''; return; }
    var codes = made.codes || (made.campaign && made.campaign.codes) || [];
    var notProjected = (made.warnings || []).indexOf('not_projected') !== -1 ||
      made.campaign.projected_at == null;

    host.innerHTML =
      '<div class="card mc-ready">' +
      '<div class="bd">' +
      '<div class="mc-ready-tick" aria-hidden="true">&#10003;</div>' +
      '<div class="mc-ready-h" id="mc-ready-head">' + codes.length +
      (codes.length === 1 ? ' code ready' : ' codes ready') + '</div>' +
      '<div class="mc-ready-s">' + esc(made.campaign.name) + ' &middot; ' +
      esc(made.campaign.offer_text) + '</div>' +
      (notProjected
        ? '<div class="mc-pill mc-pill-np mc-pill-block">Not on tablets yet &mdash; ' +
        'the offer saved here, but the tablets have not picked it up</div>' : '') +
      '<div id="mc-ready-list" class="mc-ready-list">' +
      codes.map(function (c) {
        return '<div class="mc-ready-row" data-action="open-code" data-code="' + esc(c.id) +
          '" role="button" tabindex="0">' +
          '<span class="mc-ready-ch">' + esc(channelLabel(c)) + '</span>' +
          '<span class="mc-short">' + esc(c.short) + '</span>' +
          '<span class="mc-go" aria-hidden="true">&rsaquo;</span>' +
          '</div>';
      }).join('') +
      '</div>' +
      '<div class="mc-hint">Open a code to share, save or print it.</div>' +
      '<button class="mc-btn mc-btn-go" data-action="open-detail" ' +
      'data-id="' + esc(made.campaign.id) + '">Open the campaign</button>' +
      '<button class="mc-btn mc-btn-quiet" data-action="to-list">Back to campaigns</button>' +
      '</div></div>';
  }

  // ── detail: the Money card + the code rows (Current Campaigns 4) ─────────
  function renderDetail() {
    var host = $('mc-view-detail');
    if (!host) return;
    var back = '<button class="back mc-back" data-action="to-list">Campaigns</button>';

    if (S.detailError) {
      host.innerHTML = back + '<div class="mc-banner mc-banner-bad">' +
        '<div class="mc-banner-t">Couldn&rsquo;t load this campaign</div>' +
        '<div class="mc-banner-s">' + esc(S.detailError) + '</div>' +
        '<button class="mc-btn mc-btn-quiet" data-action="reload-detail">Retry</button></div>';
      return;
    }
    if (!S.detail) {
      host.innerHTML = back + '<div class="mc-skel"></div><div class="mc-skel"></div>';
      return;
    }

    var c = S.detail;
    var m = c.money || {};
    var codes = c.codes || [];
    var f = c.funnel || {};

    host.innerHTML = back +
      '<div class="card"><div class="hd">' +
      '<h1 id="mc-detail-name">' + esc(c.name) + '</h1>' +
      '<div class="sub">' + esc(c.offer_text) + '</div>' +
      '<div class="mc-pills">' + statusPill(c) +
      (c.projected_at == null
        ? '<span class="mc-pill mc-pill-np">Not on tablets yet</span>' : '') +
      (c.requires_online ? '<span class="mc-pill mc-pill-info">Online only</span>' : '') +
      '</div></div>' +
      '<div class="bd">' +
      '<div class="mc-meta">' + esc(money(c.face_value_cents)) + ' off' +
      (c.item && c.item.name ? ' &middot; ' + esc(c.item.name) : ' &middot; any item') +
      ' &middot; ' + esc(day(c.starts_at)) + ' &ndash; ' + esc(day(c.ends_at)) +
      ' &middot; opens ' + esc(landingLabel(c.landing)) + '</div>' +
      '<div class="mc-funnel">' +
      leg('scans', 'Scans', f.scans) +
      '<span class="mc-arrow" aria-hidden="true">&rsaquo;</span>' +
      leg('signups', 'Signups', f.signups) +
      '<span class="mc-arrow" aria-hidden="true">&rsaquo;</span>' +
      leg('redeemed', 'Redeemed', f.redeemed) +
      '</div>' +
      '</div></div>' +

      '<div class="card" id="mc-money"><div class="hd"><h1>Money</h1>' +
      '<div class="sub">discount ' + esc(m.discount_basis || '—') + '</div></div>' +
      '<div class="bd mc-money-card">' +
      row('Revenue', money(m.revenue_cents)) +
      row('Discount given', money(m.discount_cents)) +
      row('Net', money(m.net_cents), 'strong') +
      row('Back per $1 off', perDollar(m.per_dollar)) +
      row('Avg order with the offer', money(m.avg_order_cents_with)) +
      row('Avg order without', money(m.avg_order_cents_without)) +
      '</div></div>' +

      '<div class="card"><div class="hd"><h1>Codes</h1>' +
      '<div class="sub">' + codes.length + (codes.length === 1 ? ' code' : ' codes') +
      ' &middot; tap one to share, print or re-point it</div></div>' +
      '<div id="mc-codes" class="mc-codes">' +
      (codes.length ? codes.map(function (cd) {
        return '<div class="mc-code-row" data-action="open-code" data-code="' + esc(cd.id) +
          '" role="button" tabindex="0">' +
          '<span class="mc-code-ch">' + esc(channelLabel(cd)) +
          (cd.placement ? '<span class="mc-code-pl">' + esc(cd.placement) + '</span>' : '') +
          '</span>' +
          '<span class="mc-short">' + esc(cd.short) + '</span>' +
          '<span class="mc-code-sc">' + esc(cd.scans == null ? '—' : cd.scans) + ' scans</span>' +
          (cd.active ? '' : '<span class="mc-pill mc-pill-warn">paused</span>') +
          '<span class="mc-go" aria-hidden="true">&rsaquo;</span>' +
          '</div>';
      }).join('')
        : '<div class="bd">This campaign has no codes, so nothing can scan it.</div>') +
      '</div></div>' +

      '<button class="mc-btn mc-btn-quiet" id="mc-pause-campaign" data-action="toggle-campaign">' +
      (c.status === 'paused' ? 'Resume campaign' : 'Pause campaign') + '</button>';
  }
  function row(label, v, cls) {
    return '<div class="mc-row' + (cls ? ' mc-' + cls : '') + '">' +
      '<span class="mc-l">' + label + '</span>' +
      '<span class="mc-v">' + esc(v) + '</span></div>';
  }

  // ── the code sheet (Current Campaigns 5–8) ───────────────────────────────
  function renderCode() {
    var host = $('mc-view-code');
    if (!host) return;
    var cd = S.code;
    if (!cd) { host.innerHTML = ''; return; }

    // The share affordance is the probe's answer, not a guess. When the probe
    // says no there is no Share button at all — a button that cannot work is
    // worse than its absence, and Save PNG + Copy link do the same job.
    var share = S.canShareFiles
      ? '<button class="mc-btn mc-btn-go" id="mc-share" data-action="share">&#9650;&nbsp; Share code</button>'
      : '';

    host.innerHTML =
      '<button class="back mc-back" data-action="to-detail">' +
      esc(S.detail ? S.detail.name : 'Campaign') + '</button>' +
      '<div class="card"><div class="hd">' +
      '<h1>' + esc(channelLabel(cd)) + '</h1>' +
      '<div class="sub">' + esc(cd.short) + ' &middot; ' +
      esc(cd.scans == null ? '—' : cd.scans) + ' scans &middot; opens ' +
      esc(landingLabel(cd.landing || (S.detail && S.detail.landing))) + '</div>' +
      (cd.active ? '' : '<div class="mc-pills"><span class="mc-pill mc-pill-warn">paused</span></div>') +
      '</div>' +
      '<div class="bd mc-code-sheet">' +
      '<img id="mc-qr" class="mc-qr" alt="QR code ' + esc(cd.short) + '" ' +
      'src="' + esc(cd.png_url) + '?size=512">' +
      '<div id="mc-payload" class="mc-payload">' + esc(cd.payload_url) + '</div>' +
      (S.note ? '<div class="mc-note" id="mc-note">' + esc(S.note) + '</div>' : '') +
      share +
      '<div class="mc-actions">' +
      '<button class="mc-btn ' + (share ? 'mc-btn-quiet' : 'mc-btn-go') + '" id="mc-save" ' +
      'data-action="save-png">&#8681;&nbsp; Save PNG</button>' +
      '<button class="mc-btn mc-btn-quiet" id="mc-copy" data-action="copy-link">&#128279;&nbsp; Copy link</button>' +
      '<button class="mc-btn mc-btn-quiet" id="mc-print" data-action="print">&#128424;&nbsp; Print</button>' +
      '</div>' +
      (S.canShareFiles ? '' :
        '<div class="mc-hint">This browser cannot share a file. Save the PNG or copy ' +
        'the link instead.</div>') +
      '</div></div>' +

      '<div class="card"><div class="hd"><h1>After it is printed</h1>' +
      '<div class="sub">A printed code never has to be reprinted</div></div>' +
      '<div class="bd">' +
      '<button class="mc-btn mc-btn-quiet" id="mc-repoint" data-action="repoint">' +
      (S.repoint ? 'Cancel re-point' : 'Re-point this code…') + '</button>' +
      (S.repoint
        ? '<div id="mc-repoint-form" class="mc-form">' +
        field('Opens', '<select id="mc-rp-landing" data-field="rpLanding">' +
          LANDINGS.map(function (l) {
            var cur = cd.landing || (S.detail && S.detail.landing) || 'signup';
            return '<option value="' + l[0] + '"' + (cur === l[0] ? ' selected' : '') + '>' +
              l[1] + '</option>';
          }).join('') + '</select>') +
        field('Placement', '<input id="mc-rp-placement" data-field="rpPlacement" type="text" ' +
          'placeholder="Counter, left window…" value="' + esc(cd.placement || '') + '">') +
        '<button class="mc-btn mc-btn-go" id="mc-rp-save" data-action="repoint-save">' +
        'Save &mdash; no reprint needed</button>' +
        '</div>'
        : '') +
      '<button class="mc-btn mc-btn-quiet" id="mc-pause-code" data-action="toggle-code">' +
      (cd.active ? 'Pause this code' : 'Un-pause this code') + '</button>' +
      '<div class="mc-hint">A paused code still scans, but the landing says the offer ' +
      'has ended.</div>' +
      '</div></div>';
  }

  function note(msg) { S.note = msg; render(); }

  // ── actions ──────────────────────────────────────────────────────────────
  var ACTIONS = {
    reload: function () { loadList(); },
    'reload-detail': function () { if (S.detail || S.lastDetailId) loadDetail(S.lastDetailId); },
    period: function (el) {
      var p = el.dataset.period;
      if (!p || p === S.period) return;
      S.period = p;
      loadList();
    },
    offer: function (el) {
      var id = el.dataset.id;
      S.expanded[id] = !S.expanded[id];
      render();
    },
    new: function () {
      S.draft = newDraft();
      S.formError = '';
      S.view = 'create';
      render();
      loadItems();
    },
    'to-list': function () {
      S.view = 'list';
      S.note = '';
      render();
    },
    'to-detail': function () {
      S.view = 'detail';
      S.note = '';
      S.repoint = false;
      render();
    },
    chip: function (el) {
      var ch = el.dataset.channel;
      var i = S.draft.channels.indexOf(ch);
      if (i === -1) S.draft.channels.push(ch); else S.draft.channels.splice(i, 1);
      S.formError = '';
      render();
    },
    create: function () { submitCreate(); },
    'open-detail': function (el) {
      var id = el.dataset.id;
      if (!id) return;
      S.lastDetailId = id;
      loadDetail(id);
    },
    'open-code': function (el) { openCode(el.dataset.code); },
    share: function () { doShare(); },
    'save-png': function () { doSavePNG(); },
    'copy-link': function () { doCopyLink(); },
    print: function () { doPrint(); },
    repoint: function () { S.repoint = !S.repoint; S.note = ''; render(); },
    'repoint-save': function () { doRepoint(); },
    'toggle-code': function () { doToggleCode(); },
    'toggle-campaign': function () { doToggleCampaign(); }
  };

  function submitCreate() {
    var d = S.draft;
    if (S.saving) return;
    if (!d.name.trim()) { S.formError = 'Give the campaign a name.'; return render(); }
    if (!d.offer.trim()) { S.formError = 'Write the offer the customer will read.'; return render(); }
    var cents = Math.round(parseFloat(d.value) * 100);
    if (!isFinite(cents) || cents < 0) {
      S.formError = 'Value is what the offer is worth — enter it, even for a free side.';
      return render();
    }
    var days = parseInt(d.days, 10);
    if (!isFinite(days) || days < 1) { S.formError = 'How many days should it run?'; return render(); }
    if (!d.channels.length) { S.formError = 'Pick at least one channel — a campaign with no code can never be scanned.'; return render(); }
    if (d.channels.indexOf('other') !== -1 && !d.otherLabel.trim()) {
      S.formError = 'Name the other channel.';
      return render();
    }

    var body = {
      name: d.name.trim(),
      offer_text: d.offer.trim(),
      face_value_cents: cents,
      runs_days: days,
      landing: d.landing,
      channels: d.channels.map(function (ch) {
        return ch === 'other'
          ? { channel: 'other', channel_label: d.otherLabel.trim() }
          : { channel: ch };
      })
    };
    if (d.item) body.item_id = d.item;

    S.saving = true;
    S.formError = '';
    render();
    req('/campaigns', { method: 'POST', body: body }).then(function (out) {
      S.saving = false;
      S.created = out;
      S.view = 'ready';
      // The list behind the sheet is now stale.
      S.campaigns = [];
      S.status = 'idle';
      render();
      loadList().then(function () { if (S.view === 'ready') render(); });
    }).catch(function (err) {
      S.saving = false;
      S.formError = isNetworkDown(err)
        ? 'No connection — the campaign was not created. Try again when you have bars.'
        : createErrorText(err);
      render();
    });
  }

  // The server's 400 slugs, in crew language. An unmapped slug is shown
  // verbatim rather than swallowed (UI-R6: loud, not silent).
  function createErrorText(err) {
    var map = {
      name_required: 'Give the campaign a name.',
      offer_text_required: 'Write the offer the customer will read.',
      face_value_cents_required: 'Value is required — the discount math needs it.',
      runs_days_required: 'How many days should it run?',
      channels_required: 'Pick at least one channel.',
      bad_landing: 'That landing is not one this app knows.',
      bad_channel: 'That channel is not one this app knows.',
      channel_label_required: 'Name the other channel.',
      bad_item_id: 'That dish is no longer in the menu.',
      managers_only: 'Creating campaigns is manager-only.'
    };
    return map[err.message] || ('Could not create the campaign (' + err.message + ').');
  }

  function openCode(codeId) {
    var pool = (S.detail && S.detail.codes) ||
      (S.created && (S.created.codes || S.created.campaign.codes)) || [];
    var cd = null;
    for (var i = 0; i < pool.length; i++) if (pool[i].id === codeId) cd = pool[i];
    if (!cd) return;
    S.code = cd;
    S.note = '';
    S.repoint = false;
    S.canShareFiles = detectShareFiles();
    S.view = 'code';
    render();
  }

  function doShare() {
    var cd = S.code;
    if (!cd) return;
    note('Preparing the code…');
    pngFile(cd, 1024).then(function (file) {
      return navigator.share({
        files: [file],
        title: 'QR code ' + cd.short,
        text: cd.payload_url
      });
    }).then(function () {
      note('Shared.');
    }).catch(function (err) {
      // A user cancelling the OS sheet rejects with AbortError — not a failure.
      if (err && err.name === 'AbortError') { note(''); return; }
      note('Could not share — save the PNG or copy the link instead.');
    });
  }

  function doSavePNG() {
    var cd = S.code;
    if (!cd) return;
    note('Saving…');
    pngFile(cd, 1024).then(function (file) {
      var url = URL.createObjectURL(file);
      var a = document.createElement('a');
      a.href = url;
      a.download = file.name;
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(function () { URL.revokeObjectURL(url); }, 10000);
      note('Saved ' + file.name);
    }).catch(function () {
      note('Could not build the PNG — check the connection and try again.');
    });
  }

  function doCopyLink() {
    var cd = S.code;
    if (!cd) return;
    var text = cd.payload_url;
    var done = function () { note('Link copied.'); };
    var fallback = function () {
      try {
        var ta = document.createElement('textarea');
        ta.value = text;
        ta.setAttribute('readonly', '');
        ta.style.position = 'fixed';
        ta.style.left = '-9999px';
        document.body.appendChild(ta);
        ta.select();
        var ok = document.execCommand('copy');
        ta.remove();
        if (ok) return done();
      } catch (e) { /* fall through */ }
      note('Could not copy — the link is above, long-press it.');
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(done).catch(fallback);
    } else {
      fallback();
    }
  }

  function doPrint() {
    // Deferred out of the handler so the note paints before the browser's
    // modal print dialog takes the main thread.
    note('Opening the print dialog…');
    setTimeout(function () { try { window.print(); } catch (e) { /* no printer, no crash */ } }, 0);
  }

  function doRepoint() {
    var cd = S.code;
    if (!cd) return;
    var landingEl = $('mc-rp-landing');
    var placementEl = $('mc-rp-placement');
    var body = {};
    if (landingEl) body.landing = landingEl.value;
    if (placementEl) body.placement = placementEl.value;
    note('Re-pointing…');
    req('/codes/' + encodeURIComponent(cd.id), { method: 'PATCH', body: body })
      .then(function (out) {
        replaceCode(out);
        S.repoint = false;
        note('Re-pointed. The printed code keeps working.');
      }).catch(function (err) {
        note(isNetworkDown(err)
          ? 'No connection — the code was not re-pointed.'
          : 'Could not re-point (' + err.message + ').');
      });
  }

  function doToggleCode() {
    var cd = S.code;
    if (!cd) return;
    var next = !cd.active;
    note(next ? 'Un-pausing…' : 'Pausing…');
    req('/codes/' + encodeURIComponent(cd.id), { method: 'PATCH', body: { active: next } })
      .then(function (out) {
        replaceCode(out);
        note(out.active ? 'This code is live again.' : 'This code is paused.');
      }).catch(function (err) {
        note(isNetworkDown(err)
          ? 'No connection — nothing changed.'
          : 'Could not change the code (' + err.message + ').');
      });
  }

  // replaceCode keeps every projection of a code in step (UI-R7): the sheet,
  // the detail list it was opened from, and the "N codes ready" screen.
  function replaceCode(out) {
    S.code = out;
    [S.detail && S.detail.codes,
    S.created && S.created.codes,
    S.created && S.created.campaign && S.created.campaign.codes].forEach(function (arr) {
      if (!arr) return;
      for (var i = 0; i < arr.length; i++) if (arr[i].id === out.id) arr[i] = out;
    });
    render();
  }

  function doToggleCampaign() {
    var c = S.detail;
    if (!c) return;
    var next = c.status === 'paused' ? 'live' : 'paused';
    req('/campaigns/' + encodeURIComponent(c.id), { method: 'PATCH', body: { status: next } })
      .then(function (out) {
        S.detail = out;
        // The list row carries the same status pill — refresh it too (UI-R7).
        for (var i = 0; i < S.campaigns.length; i++) {
          if (S.campaigns[i].id === out.id) S.campaigns[i].status = out.status;
        }
        render();
      }).catch(function (err) {
        S.detailError = isNetworkDown(err) ? 'offline' : (err.message || 'patch_failed');
        render();
      });
  }

  // ── delegation: ONE click, ONE input, on #mc-root ────────────────────────
  function wire(root) {
    root.addEventListener('click', function (e) {
      var el = e.target.closest('[data-action]');
      if (!el || !root.contains(el)) return;
      if (el.disabled) return;
      var fn = ACTIONS[el.dataset.action];
      if (!fn) return;
      e.preventDefault();
      fn(el, e);
    });
    // Keyboard parity for the rows that are role="button" divs.
    root.addEventListener('keydown', function (e) {
      if (e.key !== 'Enter' && e.key !== ' ') return;
      var el = e.target.closest('[data-action][role="button"]');
      if (!el || !root.contains(el)) return;
      var fn = ACTIONS[el.dataset.action];
      if (!fn) return;
      e.preventDefault();
      fn(el, e);
    });
    root.addEventListener('input', function (e) {
      var el = e.target.closest('[data-field]');
      if (!el || !root.contains(el)) return;
      var f = el.dataset.field;
      // The re-point fields are read at save time, not mirrored into S — they
      // belong to one in-flight PATCH, not to page state.
      if (f === 'rpLanding' || f === 'rpPlacement') return;
      if (!S.draft) S.draft = newDraft();
      S.draft[f] = el.value;
      if (f === 'landing' || f === 'item' || f === 'otherLabel') render();
      else syncSubmitLabel();
    });
  }

  // syncSubmitLabel is the ONE place this module writes the DOM outside
  // render(). Re-rendering the create sheet on every keystroke would move the
  // caret out from under the thumb, which is a worse bug than this exception.
  // Nothing it writes is state the next render() would not reproduce.
  function syncSubmitLabel() {
    var b = $('mc-submit');
    if (!b) return;
    var n = S.draft.channels.length;
    b.disabled = S.saving || !n;
  }

  // ── activation ───────────────────────────────────────────────────────────
  //
  // marketing.html's inline show() is the shared tab switcher and this card
  // does not own it, so activation is OBSERVED rather than hooked: #s2 going
  // visible is the signal. That also covers any future caller of show(2).
  var booted = false;
  function activate() {
    if (!booted) {
      booted = true;
      var root = $('mc-root');
      if (!root) return;
      wire(root);
      render();
      loadList();
      return;
    }
    // Coming back to the tab re-reads the list, so a campaign created on
    // another device (or the Scan tab's own redemptions) is not stale here.
    if (S.view === 'list' && S.status !== 'loading') loadList();
  }

  function start() {
    var s2 = $('s2');
    if (!s2) return;
    var visible = function () { return s2.style.display !== 'none' && s2.offsetParent !== null; };
    new MutationObserver(function () { if (visible()) activate(); })
      .observe(s2, { attributes: true, attributeFilter: ['style', 'hidden', 'class'] });
    // Deep link: #tab=2 opens Campaigns. show() is the page's, called not
    // changed.
    if (/(^|[#&])tab=2(&|$)/.test(location.hash) && typeof window.show === 'function') {
      window.show(2);
    }
    if (visible()) activate();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', start);
  } else {
    start();
  }
})();
