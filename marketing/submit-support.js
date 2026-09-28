// marketing/submit-support.js — pure/injectable helpers for the redemption
// submit flow (card redemption-submit-flow, run 20260905; design
// docs/qr-offline-redemption-handoff.md §13, §14 #13). Imports nothing —
// every dependency (fetch, storage, timers) arrives as a parameter, the
// marketing-family convention.

// ── §13 order-number validation ─────────────────────────────────────────────
//
// 🛑 NAMED PLACEHOLDER (build call recorded in the merge intent): the real
// Toast order-number format for this restaurant is Activity 0's deliverable
// (open decision #2 — digit count, any prefix). Until it lands, this constant
// IS the validator: digits only, 1–6 of them, trimmed. The constant name is
// the grep target for the card that replaces it.
export const TOAST_ORDER_NUMBER_PLACEHOLDER_PATTERN = /^\d{1,6}$/;

/** Validate a raw order-number entry. Returns {ok, value} — value trimmed. */
export function validateOrderNumber(raw) {
  const value = typeof raw === 'string' ? raw.trim() : '';
  return { ok: TOAST_ORDER_NUMBER_PLACEHOLDER_PATTERN.test(value), value };
}

// ── §13 business date ───────────────────────────────────────────────────────
//
// "Business date is a trap": Toast's business date rolls at a configured
// cutoff — commonly 4am, not midnight. A 12:30am scan belongs to the PREVIOUS
// business date. Constants, not guesses at call sites:
//   * CUTOFF_HOUR = 4 — the common Toast default. 🛑 Confirm the configured
//     cutoff in Toast settings (§13, open decision #1) and correct here.
//   * TIMEZONE — the truck's POS timezone (the repo's standing Chicago
//     convention, e.g. the weekly drift check).
export const TOAST_BUSINESS_DATE_CUTOFF_HOUR = 4;
export const TOAST_BUSINESS_DATE_TIMEZONE = 'America/Chicago';

/**
 * The §13 business date for a scan at `nowMs` — which MUST be the sync
 * clock's epoch (clock.now(), §5.1 offset-adjusted), never Date.now().
 * `new Date(ms)` below is an Intl formatting CONTAINER for that epoch, not a
 * time source — the card's "never new Date()" rule is about where the number
 * comes from.
 */
export function toastBusinessDate(nowMs, {
  cutoffHour = TOAST_BUSINESS_DATE_CUTOFF_HOUR,
  timeZone = TOAST_BUSINESS_DATE_TIMEZONE,
} = {}) {
  const shifted = new Date(nowMs - cutoffHour * 3600 * 1000);
  // en-CA formats as YYYY-MM-DD — the join-key shape (§13).
  return new Intl.DateTimeFormat('en-CA', {
    timeZone, year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(shifted);
}

// ── #13 reachability probe ──────────────────────────────────────────────────
//
// "Offline" must mean "can't reach the server the submit needs", never
// navigator.onLine (which lies on a hanging LTE hotspot). Timings (build
// call, recorded in the merge intent):
//   * PROBE_TIMEOUT_MS 3500 — a connected-but-hanging link resolves to
//     offline within the timeout; long enough for a slow-but-alive
//     Cloudflare-Tunnel hop, short enough that staff see the flip before
//     they re-tap.
//   * PROBE_INTERVAL_MS 10000 — bounds the stale-indicator window at the
//     counter to one customer interaction; ~6 req/min/device against
//     /api/v1/health is noise.
export const PROBE_TIMEOUT_MS = 3500;
export const PROBE_INTERVAL_MS = 10000;
export const SUBMIT_TIMEOUT_MS = 12000; // outlasts the server's ~10s arbitration budget so a 504 arrives as itself

/**
 * The probe: GET `url` with a hard abort at `timeoutMs`; `onResult(ok)` after
 * EVERY completed probe. Target is HQ's /api/v1/health — Card 7 moved the
 * online submit to HQ, so HQ reachability IS the signal that decides whether
 * submit can succeed (recorded deviation from the §13 "probe Supabase"
 * sketch; the substrate's own liveness rides the future realtime-channel
 * wiring — see reportChannelStatus in submit-flow.js).
 *
 * probeNow() is the test seam and the boot call: it resolves AFTER onResult
 * ran, so callers never wait on the production cadence.
 */
export function createReachabilityProbe({
  fetchImpl,
  url = '/api/v1/health',
  intervalMs = PROBE_INTERVAL_MS,
  timeoutMs = PROBE_TIMEOUT_MS,
  onResult,
}) {
  let timer = null;

  async function probeNow() {
    let ok = false;
    try {
      const ctl = typeof AbortController === 'function' ? new AbortController() : null;
      const kill = ctl ? setTimeout(() => ctl.abort(), timeoutMs) : null;
      try {
        const res = await fetchImpl(url, { method: 'GET', cache: 'no-store', signal: ctl ? ctl.signal : undefined });
        ok = !!(res && res.ok);
      } finally {
        if (kill) clearTimeout(kill);
      }
    } catch (e) {
      ok = false; // network error, abort, hang — all read as unreachable
    }
    try { onResult(ok); } catch (e) { /* the probe never dies on a consumer error */ }
    return ok;
  }

  return {
    probeNow,
    start() {
      if (timer) return;
      timer = setInterval(probeNow, intervalMs);
    },
    stop() {
      if (timer) { clearInterval(timer); timer = null; }
    },
  };
}

// ── device identity ─────────────────────────────────────────────────────────

export const DEVICE_ID_KEY = 'hq_marketing_device_v1';

/** Stable per-device id for scan_attempts/redeem calls (§13 #3). */
export function getDeviceId(storage) {
  try {
    let id = storage.getItem(DEVICE_ID_KEY);
    if (!id) {
      id = (typeof crypto !== 'undefined' && crypto.randomUUID) ? crypto.randomUUID() : `dev-${Date.now()}-${Math.floor(Math.random() * 1e9)}`;
      storage.setItem(DEVICE_ID_KEY, id);
    }
    return id;
  } catch (e) {
    // Storage blocked: a session-scoped id still attributes the scan.
    return `dev-session-${Date.now()}`;
  }
}
