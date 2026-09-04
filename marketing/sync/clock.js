// marketing/sync/clock.js — the sync clock: tamper-safe offline expiry (card
// clock-offset-on-sync, run 20260905; design docs/qr-offline-redemption-handoff.md
// §5.1; spike record .night-crew/knowledge/spikes/activity-b-offline-first-replica/
// clock-offset-on-sync.md).
//
// The problem (§5.1): an offline expiry check trusts the tablet's clock, and a
// tablet whose clock reads EARLIER than the server silently accepts dead codes
// (a rolled-back clock resurrects them); a clock reading LATER falsely rejects
// live ones. The spike proved both the hole and the fix on real wire data.
//
// The fix, exactly as spiked (196 ms skew recovery under a 2-day skew):
//   * serverNow source: the HTTP `Date` header on the PostgREST pull response
//     the sync ALREADY makes — no new endpoint, whole-second resolution,
//     plenty for expiry windows measured in hours.
//   * sign convention (binding, from the extraction record):
//         offset = serverNow − deviceNow
//   * every offline expires_at comparison runs as
//         deviceNow + offset  vs  expires_at          (isExpired below)
//   * update cadence: EVERY successful pull captures; latest capture wins.
//     Pure arithmetic — no smoothing, no grace windows, no thresholds.
//
// DEPENDENCY-INJECTED like its siblings — this module imports nothing. The
// device clock and the persistence both arrive as parameters, so the SAME file
// runs in the browser and in the Node gate harness, and the harness can inject
// a skewed deviceNow to prove the arithmetic.
//
// Wiring (Cards 5/6): create ONE clock per device, pass it as `clock` into
// startCodesReplica/startOffersReplica (window bounds then follow it and every
// pull calibrates it), store the `persist`-ed state beside the checkpoint in
// the page's storage, and hand it back as `initialState` on boot so a device
// reloaded OFFLINE keeps its calibration. THE offline expiry question — "is
// this code expired, right now, radio off?" — is clock.isExpired(expires_at).

/**
 * @param {object} [p]
 * @param {function(): number} [p.deviceNow]  the device clock, ms epoch.
 *   Injectable for tests; the browser passes nothing (Date.now).
 * @param {object|null} [p.initialState]  a previously persisted state()
 *   value — the reload path. Ignored unless it carries a finite offset_ms.
 * @param {function(object): void} [p.persist]  called with a state copy on
 *   every successful capture — the caller stores it beside the checkpoint.
 * @returns the device's sync clock:
 *   now()                   server-estimated now: deviceNow() + offset
 *   isExpired(expiresAt)    THE offline expiry check (§5.1) — true when
 *                           deviceNow + offset >= Date.parse(expiresAt).
 *                           Fail-closed: an unreadable expires_at is EXPIRED
 *                           (a code whose expiry cannot be read must not be
 *                           accepted offline).
 *   captureFromResponse(r)  read the Date header off a successful pull
 *                           response; returns the new offset_ms, or null when
 *                           no capture happened (header absent/unparseable —
 *                           prior offset retained; observable via .captures,
 *                           never a throw: a good pull still delivers rows).
 *   state()                 the persistable {offset_ms, captured_at_device,
 *                           captured_at_server} snapshot, or null before any
 *                           capture when no initialState was given.
 *   offsetMs                current offset (0 until first capture/state).
 *   captures                captures made by THIS instance (initialState does
 *                           not count — a rebooted clock starts at 0).
 */
export function createSyncClock({ deviceNow = Date.now, initialState = null, persist } = {}) {
  let state =
    initialState && Number.isFinite(initialState.offset_ms)
      ? {
          offset_ms: initialState.offset_ms,
          captured_at_device: initialState.captured_at_device ?? null,
          captured_at_server: initialState.captured_at_server ?? null,
        }
      : null;
  let captures = 0;

  const now = () => deviceNow() + (state ? state.offset_ms : 0);

  return {
    now,
    get offsetMs() {
      return state ? state.offset_ms : 0;
    },
    get captures() {
      return captures;
    },
    state() {
      return state ? { ...state } : null;
    },
    captureFromResponse(response) {
      const header =
        response && response.headers && typeof response.headers.get === 'function'
          ? response.headers.get('date')
          : null;
      if (!header) return null;
      const serverNow = Date.parse(header);
      if (Number.isNaN(serverNow)) return null;
      const atDevice = deviceNow();
      state = {
        offset_ms: serverNow - atDevice, // the binding sign convention
        captured_at_device: atDevice,
        captured_at_server: serverNow,
      };
      captures += 1;
      if (persist) persist({ ...state });
      return state.offset_ms;
    },
    isExpired(expiresAt, at = now()) {
      const t = typeof expiresAt === 'number' ? expiresAt : Date.parse(expiresAt);
      if (Number.isNaN(t)) return true; // fail-closed (§5.1's spirit)
      return at >= t;
    },
  };
}
