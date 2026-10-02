// marketing/stats.js — ONE module, TWO mounts. Card H4 `stats-tab-ui`,
// run 20261002. Roadmap H4 as rewritten for DECISION 192; spec
// docs/handoffs/HANDOFF-marketing-campaigns-subscribers-stats.md §2/§3 row 192
// /§5/§6 row H4; design "Current Stats 1–5" (the report) and "Current Stats
// 6–8" (the queue).
//
// ── why one file on two pages ───────────────────────────────────────────────
// Decision 192 split the old Stats tab in half: the REPORTS went to the BI hub
// (bi.html #s3, the `bi` grant, no manager tier) and the reconciliation QUEUE
// stayed on Marketing (marketing.html #s4, manager-only). Both halves render
// the SAME money block and the SAME orphan-rate arithmetic from the SAME wire
// shapes, so they share one module and mount by which root element is present:
//
//     #bi-campaigns-root  -> the report   (bi.html)
//     #ms-stats-root      -> the queue    (marketing.html)
//
// Neither mount touches the other page's DOM, and the file is precached by
// build-sw.js's existing 'marketing/*.js' glob — no new glob, no Dockerfile
// change (the decision-59 pairing is already satisfied by `COPY marketing/*.js`).
//
// ── how it is built ─────────────────────────────────────────────────────────
// State-first (CLAUDE.md): every interaction mutates STATE and calls paint();
// nothing reads an answer back out of the DOM. ONE delegated click listener and
// ONE delegated input listener per container — no inline onclick anywhere,
// including on the sheets, which are rendered INSIDE the container precisely so
// the one delegation covers them.
//
// UI-R3: every key below is a Go struct json tag read VERBATIM from
// backend/internal/marketing/{stats,reconciliation,types}.go.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 FOUR WIRE FACTS THIS FILE IS BUILT AROUND
// (logs/h3b/wire-shapes-for-card-h4.log, written for this card by card H3b)
// ═══════════════════════════════════════════════════════════════════════════
// 1. §5's ASYMMETRY, kept verbatim: /stats/overview NESTS money under `money`;
//    /stats/by ROWS FLATTEN the same keys onto the row. `totals` is row-shaped,
//    so moneyOf() reads either and ONE renderer does both.
// 2. /reconciliation/queue ships BOTH `queue` (the three buckets concatenated in
//    ladder order, each row tagged `bucket`) and `overrides`/`orphans`/
//    `unmatched`. We RENDER FROM `queue` and take section counts from the arrays.
// 3. Figures that must be rendered HONESTLY:
//      discount_implied_cents / discount_actual_cents -> the
//        "implied −$X · actual −$Y" line, shown only where they differ;
//      discount_unknown_rows non-zero -> say that those rows had no campaign to
//        price them, so the implied comparison is a FLOOR;
//      signups_basis / codes_sent_basis "unavailable" -> an EM DASH, never 0;
//      orphan_rate is NULLABLE — a rate with no denominator is not 0%;
//      suggestion.basis is `window` or `business_date`, with gap_seconds.
// 4. unattributed_redeemed / unattributed_revenue_cents are NON-NULL on
//    /stats/overview and the campaigns routes (0 there is a stated fact) and
//    NULL on /stats/by rows and totals. NEVER summed across rows.
//
// ═══════════════════════════════════════════════════════════════════════════
// 🛑 THREE OPEN, PARKED DECISIONS CONSTRAIN WHAT THIS FILE MAY CLAIM ON SCREEN.
//    It fixes none of them; it renders each one's uncertainty.
// ═══════════════════════════════════════════════════════════════════════════
// D-4 — campaign attribution cannot resolve on live data. Card 3's mirror
//   leaves campaign_id NULL and HQ holds no copy of Supabase public.codes, so
//   TODAY every accepted attempt is unattributable: the by-campaign slice is
//   per-campaign rows with scans>0, redeemed=0, revenue=0 PLUS one
//   `unattributed` row carrying all the money, and channel/item/code collapse
//   to `direct`. So: the `unattributed` row is rendered as a REAL, LABELLED row
//   and is NEVER filtered out — filtering it is what makes Σ rows ≠ overview,
//   and Σ = overview is the engine's keystone. The overview states the same
//   fact at period scope beside the money, from unattributed_redeemed /
//   unattributed_revenue_cents, so a per-campaign $0.00 is never presented as
//   "this campaign earned nothing" when the money is sitting in that row.
// D-5 — what the orphan rate COUNTS is unsettled (the shipped numerator
//   includes `unmatched` as well as orphans: 3/5 = 60% where the literal spec
//   reading gives 2/5 = 40%), and it is graded against a 10% line. So: the
//   BASIS and the numerator/denominator the SERVER sends are rendered beside
//   the rate, and this file does NO orphan arithmetic of its own — it never
//   divides, it reads orphan_rate / orphan_numerator / orphan_denominator.
//   Changing what it counts is a PARK.
// D-3 — opened_at's timezone is unconfirmed, so suggestion.basis may legitimately
//   come back `business_date` with a gap of an hour or more instead of `window`.
//   So: the basis and the gap are rendered ON the chip, and a business_date
//   suggestion is labelled advisory rather than presented as a confident match.

// ── shared formatting ──────────────────────────────────────────────────────

const esc = s => String(s == null ? '' : s).replace(/[&<>"']/g, c =>
  ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

const EM_DASH = '—';
const MINUS = '−';   // U+2212, not a hyphen: this is arithmetic.

function dollars(cents) {
  const v = Math.abs(Number(cents || 0)) / 100;
  return '$' + v.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}
// A signed money figure: negative reads −$X.XX.
function money(cents) {
  const n = Number(cents || 0);
  return (n < 0 ? MINUS : '') + dollars(n);
}
// A deduction is always shown as one, even though the wire carries it positive.
function deduct(cents) {
  return MINUS + dollars(cents);
}
function perDollar(v) {
  return v === null || v === undefined ? EM_DASH : Number(v).toFixed(2);
}
// A rate the SERVER computed. Null is not 0% — it is "no denominator".
function ratePct(v) {
  return v === null || v === undefined ? EM_DASH : (Number(v) * 100).toFixed(1) + '%';
}
function plural(n, one, many) {
  return n === 1 ? one : many;
}
// "4m" / "8h 26m" / "45s" — the gap on a suggestion chip.
function gapText(seconds) {
  const s = Math.abs(Number(seconds || 0));
  if (s < 60) return s + 's';
  const m = Math.round(s / 60);
  if (m < 60) return m + 'm';
  const h = Math.floor(m / 60);
  const rest = m % 60;
  return rest ? h + 'h ' + rest + 'm' : h + 'h';
}
function timeOf(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  return d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' });
}
function dateOf(iso) {
  const d = new Date(iso);
  if (isNaN(d)) return '';
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

// The period selector. `all` is deliberately offered: an orphan rate over a
// 7-day window on a food truck is a handful of rows.
const PERIODS = [
  { key: '7d', label: '7 days', long: 'Last 7 days' },
  { key: '30d', label: '30 days', long: 'Last 30 days' },
  { key: '90d', label: '90 days', long: 'Last 90 days' },
  { key: 'all', label: 'All time', long: 'All time' },
];
const PERIOD_LONG = Object.fromEntries(PERIODS.map(p => [p.key, p.long]));

// §5's slice dimensions, in the design's order.
const DIMS = [
  { key: 'campaign', label: 'Campaign' },
  { key: 'channel', label: 'Channel' },
  { key: 'item', label: 'Item' },
  { key: 'code', label: 'Code' },
];

// moneyDTO, wherever it sits: NESTED under `money` on an overview, FLATTENED
// onto a /stats/by row or `totals`. Wire fact 1, in one function.
function moneyOf(obj) {
  if (!obj) return null;
  return obj.money ? obj.money : obj;
}

// api is the one fetch. It distinguishes the three refusals this card renders
// differently — offline, a grant/tier 403, and everything else — because
// "failures are loud + retryable" (UI-R7) needs to say WHICH failure.
async function api(path, opts) {
  let res;
  try {
    res = await fetch(path, opts);
  } catch (e) {
    // A thrown fetch is the network, not the server. Distinguishable so the
    // card can say "offline" rather than blame the backend.
    const err = new Error('offline');
    err.offline = true;
    throw err;
  }
  if (res.status === 401) {
    window.location.href = '/login.html';
    throw new Error('unauthorized');
  }
  let body = null;
  try { body = await res.json(); } catch (e) { body = null; }
  if (!res.ok) {
    const err = new Error((body && body.error) || 'api_error');
    err.status = res.status;
    err.body = body || {};
    throw err;
  }
  return body;
}
// A 403 is a GRANT or a TIER refusal, and both render as the designed Locked
// state rather than as an error. `forbidden` carries missing_grant (the app
// grant middleware); `managers_only` is the handler's own tier (handoff §16).
const isLocked = e => e && e.status === 403;

// ═══════════════════════════════════════════════════════════════════════════
// MOUNT A · the BI Campaigns report (bi.html #s3)
// ═══════════════════════════════════════════════════════════════════════════
//
// The `bi` grant alone opens this — decision 192, NO manager tier. Trends and
// Food cost are not touched by anything in here: this mount reads and writes
// ONLY inside #bi-campaigns-root, and tells the hub its status through one
// CustomEvent rather than reaching into the hub's DOM.

const BI_API = '/api/v1/bi/campaigns';

const BI = {
  status: 'loading',          // loading | ready | empty | error | locked
  period: '30d',
  overview: null,
  dim: 'campaign',
  // The slice opens on MONEY: the report above it is a money report, and
  // "what did this cost and what came back" is the question the slice answers.
  // The Funnel side is one tap away.
  mode: 'money',              // money | funnel
  by: null,
  byStatus: 'loading',        // loading | ready | error
  byError: null,
  // A drill-in: {dim, key, label} — the slice we came FROM, so the crumb can
  // say what is being filtered and the way back is one tap.
  drill: null,
  error: null,
  offline: false,
  lockedReason: null,
};

let biRoot = null;
let biSeq = 0;

function biEmit() {
  // The hub row reads this, and only this. Keeping it an event means bi.html's
  // hub code and this module share no variable and no DOM.
  // Three shapes, and the hub maps each to one of its three row states:
  // 'loading' keeps the skeleton (emitting a reading of 0 here would paint
  // "no work waiting" before anything had been read), 'error' is the row's
  // "Status unavailable", and the object is a reading.
  const detail = BI.status === 'loading'
    ? 'loading'
    : (BI.status === 'error' || BI.status === 'locked')
      ? 'error'
      : {
        needs: BI.overview ? biNeedsCount(BI.overview) : 0,
        periodLabel: PERIOD_LONG[BI.period] || PERIOD_LONG['30d'],
      };
  document.dispatchEvent(new CustomEvent('bi-campaigns-status', { detail }));
}
function biNeedsCount(ov) {
  const n = (ov && ov.needs_look) || {};
  return (n.overrides || 0) + (n.orphans || 0) + (n.unmatched || 0);
}
// An "empty period" is a period in which NOTHING happened — no scan, no
// redemption, no reconciliation history. A period with scans but no redemptions
// is NOT empty; it has a real story (nobody redeemed) and gets the full report.
function biIsEmpty(ov) {
  if (!ov) return false;
  const f = ov.funnel || {};
  const r = ov.reconciliation || {};
  return !(f.scans || 0) && !(f.redeemed || 0)
    && !(r.matched || 0) && !(r.open || 0) && !(r.declined || 0);
}

async function biLoad() {
  const seq = ++biSeq;
  BI.status = 'loading'; BI.byStatus = 'loading'; BI.error = null; BI.offline = false;
  biPaint();
  biEmit();
  try {
    const ov = await api(`${BI_API}/overview?period=${encodeURIComponent(BI.period)}`);
    if (seq !== biSeq) return;
    BI.overview = ov;
    BI.status = biIsEmpty(ov) ? 'empty' : 'ready';
  } catch (e) {
    if (seq !== biSeq) return;
    BI.overview = null;
    if (isLocked(e)) {
      BI.status = 'locked';
      BI.lockedReason = e.message;
    } else {
      BI.status = 'error';
      BI.offline = !!e.offline;
      BI.error = e.message;
    }
    biPaint();
    biEmit();
    return;
  }
  biPaint();
  biEmit();
  if (BI.status === 'ready') await biLoadBy();
}

async function biLoadBy() {
  const seq = biSeq;
  BI.byStatus = 'loading'; BI.byError = null;
  biPaint();
  const q = [`dim=${encodeURIComponent(BI.dim)}`, `period=${encodeURIComponent(BI.period)}`];
  if (BI.drill && BI.drill.dim === 'campaign') q.push(`campaign_id=${encodeURIComponent(BI.drill.key)}`);
  if (BI.drill && BI.drill.dim === 'item') q.push(`item_id=${encodeURIComponent(BI.drill.key)}`);
  try {
    const by = await api(`${BI_API}/by?${q.join('&')}`);
    if (seq !== biSeq) return;
    BI.by = by;
    BI.byStatus = 'ready';
  } catch (e) {
    if (seq !== biSeq) return;
    BI.by = null;
    BI.byStatus = 'error';
    BI.byError = e.offline ? 'offline' : e.message;
  }
  biPaint();
}

// ── BI rendering ───────────────────────────────────────────────────────────

function biPeriodChips() {
  return `<div class="bic-periods" role="group" aria-label="Period">` +
    PERIODS.map(p => `<button type="button" class="bic-period${p.key === BI.period ? ' on' : ''}" data-period="${p.key}">${esc(p.label)}</button>`).join('') +
    `</div>`;
}

function biFunnel(ov) {
  const f = ov.funnel || {};
  // An "unavailable" basis is an EM DASH, never 0 (wire fact 3). The reason is
  // stated under the row so the dash is not a mystery.
  const unavailable = [];
  const step = (k, label, value, basis) => {
    let v;
    if (basis === 'unavailable') { v = EM_DASH; unavailable.push(label); }
    else v = String(value || 0);
    return `<div class="bic-step" data-k="${k}"><span class="bic-step-v">${v}</span><span class="bic-step-l">${esc(label)}</span></div>`;
  };
  const steps = step('scans', 'Scans', f.scans, null)
    + step('signups', 'Signups', f.signups, ov.signups_basis)
    + step('codes_sent', 'Codes sent', f.codes_sent, ov.codes_sent_basis)
    + step('redeemed', 'Redeemed', f.redeemed, null);
  const note = unavailable.length
    ? `<div class="bic-basis-note">${esc(unavailable.join(' and '))} ${plural(unavailable.length, 'is', 'are')} not being recorded yet, so ${plural(unavailable.length, 'it is', 'they are')} shown as ${EM_DASH} rather than 0.</div>`
    : '';
  return `<div class="card"><div class="hd"><h1>Funnel</h1><div class="sub">${esc(PERIOD_LONG[BI.period] || '')}</div></div>
    <div id="bic-funnel" class="bic-funnel">${steps}</div>${note}</div>`;
}

function biMoney(ov) {
  const m = moneyOf(ov) || {};
  const row = (k, label, value, cls) =>
    `<div class="bic-money-row${cls ? ' ' + cls : ''}" data-k="${k}"><span class="bic-money-l">${esc(label)}</span><span class="bic-money-v">${value}</span></div>`;

  let out = `<div class="card" id="bic-money"><div class="hd"><h1>Money</h1><div class="sub">Revenue on matched orders, less what the offers gave away</div></div><div class="bic-money-body">`;
  out += row('revenue', 'Revenue', money(m.revenue_cents));
  out += row('discount', 'Discount given', deduct(m.discount_cents));
  out += row('net', 'Net', money(m.net_cents), 'bic-net');

  // decision 190's decomposition — shown ONLY where the two differ, which is
  // the whole information content of the line.
  if (Number(m.discount_implied_cents || 0) !== Number(m.discount_actual_cents || 0)) {
    out += `<div class="bic-discount-split">Discount: implied ${deduct(m.discount_implied_cents)} · actual ${deduct(m.discount_actual_cents)}
      <span class="bic-split-why">The offers were worth the first figure; the till took off the second.</span></div>`;
  }
  // discount_unknown_rows: those rows had NO campaign to price them, so the
  // IMPLIED comparison is a floor. Stated, never a silent zero.
  const unk = Number(m.discount_unknown_rows || 0);
  if (unk > 0) {
    out += `<div class="bic-unknown-note">${unk} ${plural(unk, 'redemption', 'redemptions')} had no campaign to price ${plural(unk, 'it', 'them')}, so the implied figure above is a floor, not a total.</div>`;
  }
  // D-4 at PERIOD scope. unattributed_redeemed is non-null here (wire fact 4),
  // so a 0 is a stated fact and is said as one rather than hidden.
  const ur = m.unattributed_redeemed;
  if (ur !== null && ur !== undefined) {
    out += Number(ur) > 0
      ? `<div class="bic-unattributed">${ur} ${plural(Number(ur), 'redemption', 'redemptions')} (${dollars(m.unattributed_revenue_cents)}) could not be attributed to any campaign. That money is real and is in the totals above — it sits in the <b>Unattributed</b> row of the Campaign slice, not in any campaign.</div>`
      : `<div class="bic-unattributed bic-ok">Every redemption in this period found its campaign.</div>`;
  }
  const pd = m.per_dollar;
  out += `<div class="bic-money-foot">Per $1 given away: <b>${perDollar(pd)}</b>`;
  if (m.avg_order_cents_with !== null && m.avg_order_cents_with !== undefined) {
    out += ` · Average check with the offer: <b>${dollars(m.avg_order_cents_with)}</b>`;
  }
  if (m.avg_order_cents_without !== null && m.avg_order_cents_without !== undefined) {
    out += ` · without: <b>${dollars(m.avg_order_cents_without)}</b>`;
  }
  out += `</div></div></div>`;
  return out;
}

// The reconciliation health card, and the 10% line. The rate, the numerator and
// the denominator and the BASIS all come from the server (D-5 is parked); this
// function divides nothing.
function biHealth(ov) {
  const r = (ov && ov.reconciliation) || {};
  const rate = r.orphan_rate;
  const threshold = r.threshold === null || r.threshold === undefined ? 0.1 : r.threshold;
  const over = rate !== null && rate !== undefined && Number(rate) > Number(threshold);
  const cell = (k, label, v) => `<div class="bic-h" data-k="${k}"><b>${v}</b><span>${esc(label)}</span></div>`;
  const basis = `<div class="bic-orphan-basis">${r.orphan_numerator} of ${r.orphan_denominator} redemptions, counted as <code>${esc(r.orphan_rate_basis || 'unspecified')}</code></div>`;
  return `<div class="card${over ? ' over' : ''}" id="bic-health">
    <div class="hd"><h1>Reconciliation</h1><div class="sub">Whether the till and the scanner agree</div></div>
    <div class="bic-hs">${cell('matched', 'Matched', r.matched || 0)}${cell('open', 'Open', r.open || 0)}${cell('declined', 'Declined', r.declined || 0)}</div>
    <div class="bic-orphan-wrap">
      <div class="bic-orphan-line"><span class="bic-orphan-rate">${ratePct(rate)}</span>
        <span class="bic-orphan-target">unexplained · target under ${Math.round(Number(threshold) * 100)}%</span></div>
      ${basis}
      ${rate === null || rate === undefined
      ? `<div class="bic-orphan-note">No redemptions to rate yet — a rate with no denominator is not 0%.</div>`
      : (over ? `<div class="bic-orphan-note bic-bad">Over the ${Math.round(Number(threshold) * 100)}% line. The open redemptions are worked in Marketing › Redemption stats.</div>`
        : `<div class="bic-orphan-note bic-ok">Under the ${Math.round(Number(threshold) * 100)}% line.</div>`)}
    </div></div>`;
}

function biSliceLinks() {
  return `<div class="bic-slices" id="bic-slices" role="group" aria-label="Slice by">` +
    DIMS.map(d => `<button type="button" class="bic-slice${d.key === BI.dim ? ' on' : ''}" data-dim="${d.key}">${esc(d.label)}</button>`).join('') +
    `</div>`;
}

// One row renderer for both modes and for the totals line, because `totals` is
// row-shaped (wire fact 1) and a second renderer is how two readings of one
// number start to disagree.
function biRowCells(r) {
  const m = moneyOf(r) || {};
  if (BI.mode === 'money') {
    return `<span class="bic-c" data-k="revenue">${money(m.revenue_cents)}</span>`
      + `<span class="bic-c" data-k="discount">${deduct(m.discount_cents)}</span>`
      + `<span class="bic-c" data-k="per_dollar">${perDollar(m.per_dollar)}</span>`;
  }
  return `<span class="bic-c" data-k="scans">${r.scans || 0}</span>`
    + `<span class="bic-c" data-k="signups">${BI.by && BI.by.signups_basis === 'unavailable' ? EM_DASH : (r.signups || 0)}</span>`
    + `<span class="bic-c" data-k="redeemed">${r.redeemed || 0}</span>`;
}
function biHeadCells() {
  return BI.mode === 'money'
    ? `<span class="bic-c" data-k="revenue">Revenue</span><span class="bic-c" data-k="discount">Discount</span><span class="bic-c" data-k="per_dollar">Per $1</span>`
    : `<span class="bic-c" data-k="scans">Scans</span><span class="bic-c" data-k="signups">Signups</span><span class="bic-c" data-k="redeemed">Redeemed</span>`;
}

function biSlice() {
  const dimLabel = (DIMS.find(d => d.key === BI.dim) || {}).label || BI.dim;
  let head = `<div class="hd"><h1>By ${esc(dimLabel.toLowerCase())}</h1><div class="sub">Every redemption in the period lands in exactly one row, so these add up to the figures above</div></div>`;
  let body = '';
  if (BI.byStatus === 'loading') {
    body = `<div class="bic-skel-rows">${'<div class="bic-skel"></div>'.repeat(4)}</div>`;
  } else if (BI.byStatus === 'error') {
    body = `<div class="bic-error bic-error-inline">${BI.byError === 'offline'
      ? `You’re offline, so this slice could not be read.`
      : `Couldn’t load this slice.`}<button type="button" class="bic-retry" data-action="retry-by">Retry</button></div>`;
  } else {
    const rows = (BI.by && BI.by.rows) || [];
    if (!rows.length) {
      body = `<div class="bic-none">Nothing in this slice for the period.</div>`;
    } else {
      body = `<div class="bic-head"><span class="bic-row-label">${esc(dimLabel)}</span><span class="bic-cells">${biHeadCells()}</span></div>`
        + rows.map(r => {
          // D-4: the unattributed / direct rows are REAL, LABELLED rows and are
          // never filtered out. They are marked so a manager can see at a glance
          // that the money in them belongs to no campaign — not hidden, which
          // is what would stop the slice summing to the overview.
          const special = r.key === 'unattributed' || r.key === 'direct' || r.key === 'any';
          const why = r.key === 'unattributed' ? 'No campaign could be resolved for these redemptions'
            : r.key === 'direct' ? 'No QR code could be resolved, so the channel and item are unanswerable'
              : r.key === 'any' ? 'The campaign or code names no single menu item' : '';
          return `<button type="button" class="bic-row${special ? ' bic-row-special' : ''}" data-key="${esc(r.key)}" data-label="${esc(r.label)}">
            <span class="bic-row-label">${esc(r.label)}${why ? `<span class="bic-row-note">${esc(why)}</span>` : ''}</span><span class="bic-cells">${biRowCells(r)}</span></button>`;
        }).join('')
        + (BI.by && BI.by.totals
          ? `<div class="bic-totals"><span class="bic-row-label">${esc(BI.by.totals.label || 'Total')}</span><span class="bic-cells">${biRowCells(BI.by.totals)}</span></div>`
          : '');
    }
  }
  const toggle = `<div class="bic-toggles" role="group" aria-label="Show">`
    + `<button type="button" class="bic-toggle${BI.mode === 'funnel' ? ' on' : ''}" data-mode="funnel">Funnel</button>`
    + `<button type="button" class="bic-toggle${BI.mode === 'money' ? ' on' : ''}" data-mode="money">Money</button></div>`;
  const drill = BI.drill
    ? `<div class="bic-drill">Filtered to <b>${esc(BI.drill.label)}</b><button type="button" class="bic-drill-clear" data-action="clear-drill">Clear</button></div>`
    : '';
  return `<div class="card" id="bic-slice" data-dim="${esc(BI.dim)}" data-mode="${esc(BI.mode)}">${head}${toggle}${drill}${body}</div>`;
}

function biPaint() {
  if (!biRoot) return;
  biRoot.dataset.state = BI.status;

  if (BI.status === 'locked') {
    biRoot.innerHTML = `<div class="card bic-locked"><div class="hd"><h1>Campaigns</h1></div>
      <div class="bd">${BI.lockedReason === 'managers_only'
        ? 'Managers only. Ask an admin if you need to read the campaign reports.'
        : 'BI access required. Ask an admin for the BI grant to read the campaign reports.'}</div></div>`;
    return;
  }
  if (BI.status === 'loading') {
    // No figure is on screen before one is known: skeletons carry no digits and
    // no currency, so there is nothing here to misread as data.
    biRoot.innerHTML = `${biPeriodChips()}
      <div class="card"><div class="hd"><h1>Campaigns</h1><div class="sub">Loading…</div></div>
      <div class="bic-skel-rows">${'<div class="bic-skel"></div>'.repeat(4)}</div></div>
      <div class="card"><div class="bic-skel-rows">${'<div class="bic-skel"></div>'.repeat(3)}</div></div>`;
    return;
  }
  if (BI.status === 'error') {
    // An error state says ONE thing: it failed, and here is how to try again.
    // It does NOT also claim "no data", and it carries no zeros.
    biRoot.innerHTML = `${biPeriodChips()}
      <div class="card bic-error"><div class="hd"><h1>Campaigns</h1></div>
      <div class="bd">${BI.offline
        ? 'You’re offline, so the campaign reports could not be read. They need the server.'
        : 'Couldn’t load the campaign reports.'}</div>
      <div class="bic-error-foot"><button type="button" class="bic-retry" data-action="retry">Retry</button></div></div>`;
    return;
  }
  if (BI.status === 'empty') {
    // Deliberately NO money card and NO health card: three confident zeros over
    // a period in which nothing happened is a worse answer than saying so.
    biRoot.innerHTML = `${biPeriodChips()}
      <div class="card bic-empty"><div class="hd"><h1>Campaigns</h1><div class="sub">${esc(PERIOD_LONG[BI.period] || '')}</div></div>
      <div class="bd"><b>No redemptions yet</b><br>Nothing was scanned or redeemed in this period, so there is no money to report. Try a longer period, or check back after service.</div></div>`;
    return;
  }
  const ov = BI.overview;
  biRoot.innerHTML = biPeriodChips() + biFunnel(ov) + biMoney(ov) + biHealth(ov) + biSliceLinks() + biSlice();
}

function biMount(root) {
  biRoot = root;
  // ONE delegated click listener for the whole report.
  root.addEventListener('click', e => {
    const period = e.target.closest('.bic-period');
    if (period) {
      if (period.dataset.period === BI.period) return;
      BI.period = period.dataset.period;
      BI.drill = null;
      biLoad();
      return;
    }
    const act = e.target.closest('[data-action]');
    if (act) {
      const a = act.dataset.action;
      if (a === 'retry') { biLoad(); return; }
      if (a === 'retry-by') { biLoadBy(); return; }
      if (a === 'clear-drill') { BI.drill = null; biLoadBy(); return; }
    }
    const slice = e.target.closest('.bic-slice');
    if (slice) {
      if (slice.dataset.dim === BI.dim && !BI.drill) return;
      BI.dim = slice.dataset.dim;
      BI.drill = null;
      biLoadBy();
      return;
    }
    const toggle = e.target.closest('.bic-toggle');
    if (toggle) {
      if (toggle.dataset.mode === BI.mode) return;
      BI.mode = toggle.dataset.mode;
      biPaint();
      return;
    }
    const row = e.target.closest('.bic-row');
    if (row) {
      // Drill in: a campaign row shows that campaign's items; an item row shows
      // the campaigns that sold it. The `unattributed` / `direct` rows name no
      // filterable entity, so they report why instead of drilling nowhere.
      if (row.dataset.key === 'unattributed' || row.dataset.key === 'direct' || row.dataset.key === 'any') return;
      if (BI.dim === 'campaign') {
        BI.drill = { dim: 'campaign', key: row.dataset.key, label: row.dataset.label };
        BI.dim = 'item';
      } else if (BI.dim === 'item') {
        BI.drill = { dim: 'item', key: row.dataset.key, label: row.dataset.label };
        BI.dim = 'campaign';
      } else {
        return;
      }
      biLoadBy();
    }
  });
  biLoad();
}

// bi.html's hub calls this when its Campaigns row is opened after a failure.
function biEnsure() {
  if (BI.status === 'error') biLoad();
}

// ═══════════════════════════════════════════════════════════════════════════
// MOUNT B · the reconciliation queue (marketing.html #s4)
// ═══════════════════════════════════════════════════════════════════════════
//
// MANAGER-ONLY, and it stays in Marketing: decision 192 — it is WRITES, not a
// report. The reports moved to BI; this is the worklist, and its health card is
// what P-KR3 and Q-KR2 read.

const MS_API = '/api/v1/marketing';

// §5's decline reasons, in the order a manager meets them. 🛑 THE ENUM IS A
// LIFECYCLE ROW (migration 0084's CHECK + reconDeclineReasons in
// internal/marketing/reconciliation.go). A SEVENTH REASON IS A PARK, not an
// edit here: adding one to this array without the migration and the Go slice
// produces a 400 bad_reason at the till.
const REASONS = [
  { key: 'no_such_order', label: 'No such order' },
  { key: 'customer_left', label: 'Customer left' },
  { key: 'comped', label: 'Comped' },
  { key: 'duplicate_scan', label: 'Duplicate scan' },
  { key: 'no_discount_applied', label: 'No discount applied' },
  { key: 'other', label: 'Other' },
];
const REASON_LABEL = Object.fromEntries(REASONS.map(r => [r.key, r.label]));

// 🛑 THE BUCKETS ARE A LIFECYCLE ROW TOO (reconAttempt.bucket()). A NEW BUCKET
// IS A PARK. The order here is the ladder card H3b ships in `queue`:
// overrides → orphans → unmatched, then the declined bucket below them.
const BUCKETS = [
  {
    key: 'override', title: 'Offline overrides',
    sub: 'A code was accepted without being verified. Confirm it was real.',
  },
  {
    key: 'orphan', title: 'No order number',
    sub: 'A redemption with no till order attached. Find the order, or say why you can’t.',
  },
  {
    key: 'unmatched', title: 'Order not in Toast',
    sub: 'An order number was typed in, but Toast has no such order on that date.',
  },
];

const MS = {
  status: 'loading',       // loading | ready | empty | error | locked
  period: '30d',
  overview: null,
  queue: null,
  declined: [],
  error: null,
  offline: false,
  lockedReason: null,
  // the open sheet, if any
  sheet: null,            // null | 'order' | 'decline'
  sheetId: null,
  order: '',
  reason: null,
  note: '',
  busy: false,
  sheetErr: null,
};

let msRoot = null;
let msSeq = 0;

function msOpenCount() {
  const q = MS.queue;
  if (!q) return 0;
  // Section counts come from the ARRAYS; the rows come from `queue` (wire fact 2).
  return (q.overrides || []).length + (q.orphans || []).length + (q.unmatched || []).length;
}
function msRowsIn(bucket) {
  const q = MS.queue;
  if (!q) return [];
  return (q.queue || []).filter(r => r.bucket === bucket);
}
function msRowById(id) {
  const q = MS.queue;
  const inQueue = q ? (q.queue || []).find(r => r.id === id) : null;
  return inQueue || MS.declined.find(r => r.id === id) || null;
}
function msIsEmpty() {
  const r = (MS.overview && MS.overview.reconciliation) || {};
  return msOpenCount() === 0 && MS.declined.length === 0
    && !(r.matched || 0) && !(r.declined || 0);
}

async function msLoad() {
  const seq = ++msSeq;
  MS.status = 'loading'; MS.error = null; MS.offline = false;
  msPaint();
  const p = encodeURIComponent(MS.period);
  try {
    const [ov, queue, declined] = await Promise.all([
      api(`${MS_API}/stats/overview?period=${p}`),
      api(`${MS_API}/reconciliation/queue?period=${p}`),
      api(`${MS_API}/reconciliation/declined?period=${p}`),
    ]);
    if (seq !== msSeq) return;
    MS.overview = ov;
    MS.queue = queue;
    MS.declined = (declined && declined.declined) || [];
    MS.status = msIsEmpty() ? 'empty' : 'ready';
  } catch (e) {
    if (seq !== msSeq) return;
    MS.overview = null; MS.queue = null; MS.declined = [];
    if (isLocked(e)) {
      MS.status = 'locked';
      MS.lockedReason = e.message;
    } else {
      MS.status = 'error';
      MS.offline = !!e.offline;
      MS.error = e.message;
    }
  }
  msPaint();
}

// Every write goes through here: POST, then re-read. The re-read is what makes
// the bucket a row actually landed in the server's answer and not this file's
// guess — ReconDecisionHandler re-derives the bucket for exactly that reason.
async function msDecide(id, kind, body) {
  MS.busy = true; MS.sheetErr = null;
  msPaint();
  try {
    await api(`${MS_API}/reconciliation/${encodeURIComponent(id)}/${kind}`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {}),
    });
  } catch (e) {
    MS.busy = false;
    // A refusal is REPORTED, never swallowed: 409 order_not_found names the
    // order number Toast does not have, on the sheet the manager typed it into.
    if (e.message === 'order_not_found') {
      const b = e.body || {};
      MS.sheetErr = `Toast has no order ${b.order_number || body.order_number} on ${b.business_date || 'that date'}. Check the number on the ticket.`;
    } else if (e.message === 'note_required') {
      MS.sheetErr = 'A note is required when the reason is Other.';
    } else if (e.message === 'reason_required') {
      MS.sheetErr = 'Choose a reason first.';
    } else if (e.offline) {
      MS.sheetErr = 'You’re offline. This needs the server — try again when you have signal.';
    } else if (isLocked(e)) {
      MS.status = 'locked'; MS.lockedReason = e.message; MS.sheet = null;
    } else {
      MS.sheetErr = 'That didn’t save. Try again.';
    }
    msPaint();
    return false;
  }
  MS.busy = false;
  MS.sheet = null; MS.sheetId = null; MS.order = ''; MS.reason = null; MS.note = '';
  await msLoad();
  return true;
}

// ── MS rendering ───────────────────────────────────────────────────────────

function msHealth() {
  const r = (MS.overview && MS.overview.reconciliation) || {};
  const threshold = r.threshold === null || r.threshold === undefined ? 0.1 : r.threshold;
  const rate = r.orphan_rate;
  const over = rate !== null && rate !== undefined && Number(rate) > Number(threshold);
  const cell = (k, label, v) => `<div class="msq-h" data-k="${k}"><b>${v}</b><span>${esc(label)}</span></div>`;
  return `<div class="card${over ? ' over' : ''}" id="msq-health">
    <div class="msq-hs">${cell('matched', 'Matched', r.matched || 0)}${cell('open', 'Open', r.open || 0)}${cell('declined', 'Declined', r.declined || 0)}</div>
    <div class="msq-orphan-wrap">
      <span class="msq-orphan">${ratePct(rate)}</span>
      <span class="msq-orphan-target">unexplained · target under ${Math.round(Number(threshold) * 100)}%</span>
      <div class="msq-orphan-basis">${r.orphan_numerator} of ${r.orphan_denominator} redemptions, counted as <code>${esc(r.orphan_rate_basis || 'unspecified')}</code></div>
    </div></div>`;
}

function msProvenance(r) {
  const bits = [];
  bits.push(timeOf(r.scanned_at) + ' · ' + dateOf(r.scanned_at));
  if (r.device_id) bits.push(esc(r.device_id));
  if (r.code_short) bits.push(esc(r.code_short));
  if (r.order_number) bits.push('order #' + esc(r.order_number));
  return bits.join(' · ');
}

function msRow(r) {
  const flags = [];
  if (r.offline_override) flags.push('<span class="msq-flag">offline override</span>');
  if (r.unverified_code) flags.push('<span class="msq-flag">code not verified</span>');
  if (r.policy_unresolved) flags.push('<span class="msq-flag">policy unresolved</span>');
  if (r.order && r.order.voided) flags.push('<span class="msq-flag msq-flag-bad">order voided</span>');
  const name = r.campaign_name || 'Unknown campaign';
  const value = r.face_value_cents !== null && r.face_value_cents !== undefined
    ? dollars(r.face_value_cents) + ' offer' : 'offer value unknown';
  const fixLabel = r.bucket === 'override' ? 'Verify' : 'Add order number…';
  // "Can't match…" is on EVERY row, override included: a manager must always be
  // able to say "this one cannot be reconciled" and record why.
  const actions = `<div class="msq-actions">
      <button type="button" class="msq-fix" data-action="fix" data-id="${esc(r.id)}">${fixLabel}</button>
      ${r.bucket === 'override' ? `<button type="button" class="msq-reject" data-action="reject" data-id="${esc(r.id)}">Reject</button>` : ''}
      <button type="button" class="msq-decline" data-action="decline" data-id="${esc(r.id)}">Can’t match…</button>
    </div>`;
  return `<div class="msq-row" data-id="${esc(r.id)}" data-bucket="${esc(r.bucket)}">
    <div class="msq-row-main">
      <div class="msq-name">${esc(name)}</div>
      <div class="msq-meta">${value}${r.item_name ? ' · ' + esc(r.item_name) : ''}</div>
      <div class="msq-prov">${msProvenance(r)}</div>
      ${flags.length ? `<div class="msq-flags">${flags.join('')}</div>` : ''}
    </div>${actions}</div>`;
}

function msDeclinedRow(r) {
  return `<div class="msq-row" data-id="${esc(r.id)}" data-bucket="declined">
    <div class="msq-row-main">
      <div class="msq-name">${esc(r.campaign_name || 'Unknown campaign')}</div>
      <div class="msq-decl-reason">${esc(REASON_LABEL[r.reason] || r.reason || 'Declined')}</div>
      ${r.note ? `<div class="msq-decl-note">${esc(r.note)}</div>` : ''}
      <div class="msq-prov">Declined by ${esc(r.decided_by || 'someone')} · ${dateOf(r.decided_at)} ${timeOf(r.decided_at)}</div>
    </div>
    <div class="msq-actions"><button type="button" class="msq-reopen" data-action="reopen" data-id="${esc(r.id)}">Reopen</button></div></div>`;
}

function msSection(b) {
  const rows = msRowsIn(b.key);
  const count = ((MS.queue && MS.queue[b.key === 'override' ? 'overrides' : b.key === 'orphan' ? 'orphans' : 'unmatched']) || []).length;
  if (!count) return '';
  return `<div class="card msq-section" data-bucket="${b.key}">
    <div class="hd msq-section-h"><h1>${esc(b.title)} · ${count}</h1><div class="sub">${esc(b.sub)}</div></div>
    ${rows.map(msRow).join('')}</div>`;
}

// ── the two sheets ─────────────────────────────────────────────────────────
// Both are real modals rendered INSIDE #ms-stats-root (so the one delegated
// listener covers them), with a labelled 44px way out and bottom padding that
// clears the home indicator (UI-R1 / UI-R2).

function msOrderSheet() {
  const r = msRowById(MS.sheetId);
  if (!r) return '';
  const s = r.suggestion;
  let chips = '';
  if (s) {
    // D-3: the BASIS and the GAP are on the chip. `window` is the card's rule
    // (within ±30 min of the scan); `business_date` is the advisory fallback —
    // the nearest order on the same business date, OUTSIDE that window — and is
    // labelled as such so a far-away order is never presented as a match.
    const advisory = s.basis !== 'window';
    chips = `<div class="msq-sugs">
      <div class="msq-sug-head">Nearest till order</div>
      <button type="button" class="msq-sug${advisory ? ' msq-sug-advisory' : ''}" data-action="pick" data-order="${esc(s.order_number)}">
        <b>#${esc(s.order_number)}</b>
        <span>${dollars(s.amount_cents)}${s.discount_cents ? ' · ' + deduct(s.discount_cents) + ' off' : ''}</span>
        <span class="msq-sug-basis">${gapText(s.gap_seconds)} ${advisory ? 'apart · same business date' : 'apart · within 30 min'}</span>
      </button>
      ${advisory ? `<div class="msq-sug-why">Outside the 30-minute window, so this is the nearest order on the same business date rather than a likely match. Check the ticket before you accept it.</div>` : ''}
    </div>`;
  } else {
    chips = `<div class="msq-sug-why">No till order is near this scan. Type the number off the ticket.</div>`;
  }
  return `<div class="msq-sheet" id="msq-sheet-order" role="dialog" aria-modal="true" aria-label="Add an order number">
    <div class="msq-sheet-card">
      <button type="button" class="msq-close" id="msq-order-close" data-action="close">Close</button>
      <div class="msq-sheet-title">Add an order number</div>
      <div class="msq-sheet-sub">${esc(r.campaign_name || 'Unknown campaign')} · ${msProvenance(r)}</div>
      ${chips}
      <label class="msq-field"><span>Order number</span>
        <input id="msq-order-input" inputmode="numeric" autocomplete="off" value="${esc(MS.order)}"></label>
      ${MS.sheetErr ? `<div class="msq-err">${esc(MS.sheetErr)}</div>` : ''}
      <button type="button" class="msq-primary" id="msq-order-save" data-action="save-order"${MS.order.trim() && !MS.busy ? '' : ' disabled'}>${MS.busy ? 'Saving…' : 'Match this order'}</button>
    </div></div>`;
}

function msDeclineSheet() {
  const r = msRowById(MS.sheetId);
  if (!r) return '';
  const needsNote = MS.reason === 'other';
  const ok = !!MS.reason && (!needsNote || MS.note.trim().length > 0) && !MS.busy;
  return `<div class="msq-sheet" id="msq-sheet-decline" role="dialog" aria-modal="true" aria-label="Can’t match this redemption">
    <div class="msq-sheet-card">
      <button type="button" class="msq-close" id="msq-decline-close" data-action="close">Close</button>
      <div class="msq-sheet-title">Can’t match this one</div>
      <div class="msq-sheet-sub">${esc(r.campaign_name || 'Unknown campaign')} · ${msProvenance(r)}</div>
      <div class="msq-decline-what">Declining records your call and takes this redemption out of the matched money. It deletes nothing: the scan stays on the record, the reason stays with it, and you can reopen it later.</div>
      <div class="msq-reasons" role="group" aria-label="Reason">
        ${REASONS.map(x => `<button type="button" class="msq-reason${MS.reason === x.key ? ' on' : ''}" data-action="reason" data-reason="${x.key}">${esc(x.label)}</button>`).join('')}
      </div>
      <label class="msq-field"><span>Note${needsNote ? ' (required for Other)' : ' (optional)'}</span>
        <textarea id="msq-note" rows="2">${esc(MS.note)}</textarea></label>
      ${!MS.reason ? `<div class="msq-hint">Choose a reason to continue.</div>` : ''}
      ${needsNote && !MS.note.trim() ? `<div class="msq-hint">Other needs a note saying what happened.</div>` : ''}
      ${MS.sheetErr ? `<div class="msq-err">${esc(MS.sheetErr)}</div>` : ''}
      <button type="button" class="msq-primary msq-danger" id="msq-decline-save" data-action="save-decline"${ok ? '' : ' disabled'}>${MS.busy ? 'Saving…' : 'Decline this redemption'}</button>
    </div></div>`;
}

function msPaint() {
  if (!msRoot) return;
  msRoot.dataset.state = MS.status;
  const title = `<div class="hd"><h1>Redemption stats</h1><div class="sub">Reconciliation queue · ${esc(PERIOD_LONG[MS.period] || '')}</div></div>`;

  if (MS.status === 'locked') {
    // The designed Locked state covers the WHOLE section, not just its buttons,
    // and it still NAMES itself: blank render = defect. No control is offered to
    // someone who may not act.
    msRoot.innerHTML = `<div class="card ms-locked">${title}<div class="bd">${MS.lockedReason === 'managers_only'
      ? 'Managers only. The reconciliation queue is a manager’s call on real money, so it is not open to the whole crew. Ask a manager or an admin.'
      : 'Marketing access required. Ask an admin for the Marketing grant if you need to work the reconciliation queue.'}</div></div>`;
    return;
  }
  if (MS.status === 'loading') {
    msRoot.innerHTML = `<div class="card">${title}<div class="msq-skel-rows">${'<div class="msq-skel"></div>'.repeat(3)}</div></div>`;
    return;
  }
  if (MS.status === 'error') {
    msRoot.innerHTML = `<div class="card msq-error">${title}<div class="bd">${MS.offline
      ? 'You’re offline, so the queue could not be read. It needs the server — nothing here can be worked until you have signal.'
      : 'Couldn’t load the reconciliation queue.'}</div>
      <div class="msq-error-foot"><button type="button" class="msq-retry" data-action="retry">Retry</button></div></div>`;
    return;
  }
  if (MS.status === 'empty') {
    msRoot.innerHTML = `<div class="card msq-empty">${title}
      <div class="bd"><b>No redemptions yet</b><br>Nothing has been scanned and redeemed in this period, so there is nothing to reconcile. Rows land here on their own once the crew starts scanning.</div></div>`;
    return;
  }

  const open = msOpenCount();
  // The section NAMES ITSELF in every state, this one included (blank render =
  // defect, and a crew member who lands here from the tab bar needs to know
  // which screen they are on before they read a count).
  const head = `<div class="card" id="msq-head-card"><div class="hd">
    <div class="msq-kicker">Redemption stats · ${esc(PERIOD_LONG[MS.period] || '')}</div>
    <h1 id="msq-head">${open} ${plural(open, 'redemption needs', 'redemptions need')} a look</h1>
    <div class="sub">Worked top to bottom: unverified overrides first, then redemptions with no order, then order numbers Toast doesn’t have.</div></div></div>`;
  const declined = MS.declined.length
    ? `<div class="card msq-section" data-bucket="declined">
        <div class="hd msq-section-h"><h1>Declined · ${MS.declined.length}</h1><div class="sub">Calls already made. Reopen one if it was wrong.</div></div>
        ${MS.declined.map(msDeclinedRow).join('')}</div>`
    : '';
  const nothingOpen = !open
    ? `<div class="card"><div class="bd">Nothing open. Every redemption in this period is either matched or declined.</div></div>`
    : '';
  const sheet = MS.sheet === 'order' ? msOrderSheet() : MS.sheet === 'decline' ? msDeclineSheet() : '';
  msRoot.innerHTML = head + msHealth() + BUCKETS.map(msSection).join('') + nothingOpen + declined + sheet;
}

// Targeted patches, so typing in a sheet never re-creates the field under the
// caret. State is still the single source of truth — these only re-read it.
function msSyncOrderSave() {
  const btn = msRoot && msRoot.querySelector('#msq-order-save');
  if (btn) btn.disabled = !(MS.order.trim() && !MS.busy);
}
function msSyncDeclineSave() {
  const btn = msRoot && msRoot.querySelector('#msq-decline-save');
  if (!btn) return;
  const needsNote = MS.reason === 'other';
  btn.disabled = !(MS.reason && (!needsNote || MS.note.trim().length > 0) && !MS.busy);
}

function msMount(root) {
  msRoot = root;
  // ONE delegated click listener.
  root.addEventListener('click', e => {
    const act = e.target.closest('[data-action]');
    if (!act) return;
    const a = act.dataset.action;
    const id = act.dataset.id;
    if (a === 'retry') { msLoad(); return; }
    if (a === 'close') {
      MS.sheet = null; MS.sheetId = null; MS.order = ''; MS.reason = null; MS.note = ''; MS.sheetErr = null;
      msPaint(); return;
    }
    if (a === 'fix') {
      const row = msRowById(id);
      if (!row) return;
      if (row.bucket === 'override') { msDecide(id, 'verify', {}); return; }
      MS.sheet = 'order'; MS.sheetId = id; MS.sheetErr = null;
      // Pre-fill from the server's own suggestion when it is the card's rule
      // (`window`). An advisory `business_date` suggestion is NOT pre-filled:
      // it is offered as a chip the manager must choose, because D-3 means it
      // may be an hour away for no good reason.
      MS.order = row.suggestion && row.suggestion.basis === 'window' ? String(row.suggestion.order_number) : '';
      msPaint(); return;
    }
    if (a === 'reject') { msDecide(id, 'reject', {}); return; }
    if (a === 'decline') {
      MS.sheet = 'decline'; MS.sheetId = id; MS.reason = null; MS.note = ''; MS.sheetErr = null;
      msPaint(); return;
    }
    if (a === 'reopen') { msDecide(id, 'reopen', {}); return; }
    if (a === 'pick') { MS.order = String(act.dataset.order); MS.sheetErr = null; msPaint(); return; }
    if (a === 'reason') { MS.reason = act.dataset.reason; MS.sheetErr = null; msPaint(); return; }
    if (a === 'save-order') {
      if (!MS.order.trim() || MS.busy) return;
      msDecide(MS.sheetId, 'match', { order_number: MS.order.trim() });
      return;
    }
    if (a === 'save-decline') {
      const needsNote = MS.reason === 'other';
      if (!MS.reason || (needsNote && !MS.note.trim()) || MS.busy) return;
      msDecide(MS.sheetId, 'decline', {
        reason: MS.reason,
        note: MS.note.trim() ? MS.note.trim() : null,
      });
      return;
    }
  });
  // ONE delegated input listener.
  root.addEventListener('input', e => {
    if (e.target.id === 'msq-order-input') { MS.order = e.target.value; MS.sheetErr = null; msSyncOrderSave(); return; }
    if (e.target.id === 'msq-note') { MS.note = e.target.value; MS.sheetErr = null; msSyncDeclineSave(); return; }
  });
  msLoad();
}

// ═══════════════════════════════════════════════════════════════════════════
// boot — mount whichever root this page has, and neither one's DOM otherwise
// ═══════════════════════════════════════════════════════════════════════════

function boot() {
  const bi = document.getElementById('bi-campaigns-root');
  if (bi) {
    window.BICampaigns = { ensure: biEnsure, reload: biLoad };
    biMount(bi);
  }
  const ms = document.getElementById('ms-stats-root');
  if (ms) {
    window.MarketingStats = { reload: msLoad };
    msMount(ms);
  }
}
if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', boot);
else boot();
