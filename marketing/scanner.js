// marketing/scanner.js — pure scan logic for the Marketing scanner screen
// (card camera-scanner-decode, run 20260905; design
// docs/qr-offline-redemption-handoff.md §12/§4/§10, F2/F3/F5/F6, D-KR3; spike
// .night-crew/knowledge/spikes/activity-c-scanner-screen/camera-scanner-decode.md).
//
// DEPENDENCY-INJECTED like the marketing/sync/ family — this module imports
// nothing. WebCrypto, the RxDB collections, the clock (Card 4's createSyncClock
// instance) and resolveOffers (Card 2's) all arrive as parameters, so the same
// file can run under a Node harness and in the browser (marketing/scan-page.js
// is the browser wiring; Card 6's submit flow builds on the SAME surface).
//
// The §12 chain, exactly as spiked: decode (caller's job — html5-qrcode) →
// extract the token from the #10 hybrid URL wrapper → WebCrypto SHA-256 →
// lowercase hex. THE HASH IS THE REPLICA KEY: no lookup ever sees the raw
// token, and the raw token never lands in the DOM — a dumped replica or a
// screenshot never yields a live redeemable code (§4/§12).

// ── #10 payload readers ─────────────────────────────────────────────────────
//
// #10 (operator-resolved): the QR is a URL wrapping the identity token, PLUS a
// self-describing embedded-offer descriptor for offline viewing — "hybrid;
// locked at Activity E". The spike's proven extraction regex was
// /\/r\/([^/?#]+)$/ (end-anchored, bare-URL payloads). This reader keeps that
// behavior bit-for-bit on bare payloads and ADDS suffix tolerance (a ? or #
// after the token) so the descriptor-carrying form is readable too. That is a
// READER-side engineering call, not a payload-contract change: nothing tonight
// GENERATES QR payloads, and Activity E locks the final descriptor encoding —
// parseEmbeddedOffer below is the one function it replaces.
export const TOKEN_PATTERN = /\/r\/([^/?#]+)(?=[?#]|$)/;

export function extractToken(text) {
  if (typeof text !== 'string') return null;
  const m = text.match(TOKEN_PATTERN);
  return m ? m[1] : null;
}

function b64urlToUtf8(s) {
  const b64 = s.replace(/-/g, '+').replace(/_/g, '/')
    .padEnd(Math.ceil(s.length / 4) * 4, '=');
  if (typeof atob === 'function') {
    const bin = atob(b64);
    const bytes = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i += 1) bytes[i] = bin.charCodeAt(i);
    return new TextDecoder().decode(bytes);
  }
  // Node harness path (no atob on very old Nodes).
  return Buffer.from(b64, 'base64').toString('utf8');
}

/**
 * D-KR3 descriptor reader — CANDIDATE encoding, replaceable at Activity E:
 * `#o=<base64url(UTF-8 JSON)>` (fragment preferred: a customer opening the URL
 * on their own phone never sends the descriptor to any server; `?o=` is
 * accepted too). The JSON is self-describing; `label` is required — a
 * descriptor with nothing displayable is no descriptor (UI-R3: blank is never
 * a valid render). Returns null on ANY parse failure: the embedded offer is
 * display-only, unauthenticated data (roadmap Activity E trust note), so a
 * malformed one degrades to unknownCode rather than erroring the scan.
 */
export function parseEmbeddedOffer(text) {
  if (typeof text !== 'string') return null;
  const m = text.match(/[#?](?:[^#]*&)?o=([A-Za-z0-9_-]+)/);
  if (!m) return null;
  try {
    const obj = JSON.parse(b64urlToUtf8(m[1]));
    if (!obj || typeof obj !== 'object' || Array.isArray(obj)) return null;
    if (typeof obj.label !== 'string' || !obj.label.trim()) return null;
    return {
      label: obj.label,
      campaign_id: typeof obj.campaign_id === 'string' ? obj.campaign_id : null,
      expires_at: typeof obj.expires_at === 'string' ? obj.expires_at : null,
      face_value: Number.isFinite(obj.face_value) ? obj.face_value : null,
    };
  } catch (e) {
    return null;
  }
}

// ── On-device hashing (§12/§4) ──────────────────────────────────────────────

/**
 * Memoized WebCrypto SHA-256 → lowercase hex (the replica key; spike-proven
 * equal to the committed seed literals AND Node's createHash).
 *
 * Hash caching (this card's engineering call): the PROMISE is cached, not the
 * hex — two concurrent scans of one token share a single digest (misses counts
 * real digests, exactly once per distinct token). The cache is bounded
 * (maxEntries, default 512 — a full shift is a few hundred scans) and simply
 * cleared when full: correctness never depends on a hit, a miss just digests.
 *
 * @param {{subtle?: SubtleCrypto, maxEntries?: number}} [p]
 * @returns {function(string): Promise<string>} with .stats() → {hits, misses, size}
 */
export function createTokenHasher({ subtle, maxEntries = 512 } = {}) {
  const engine = subtle
    || (typeof crypto !== 'undefined' && crypto.subtle ? crypto.subtle : null);
  if (!engine) throw new Error('createTokenHasher: no SubtleCrypto available');
  const cache = new Map(); // token -> Promise<hex>
  let hits = 0;
  let misses = 0;
  const hashToken = (token) => {
    const cached = cache.get(token);
    if (cached) { hits += 1; return cached; }
    misses += 1;
    const p = engine.digest('SHA-256', new TextEncoder().encode(token)).then(
      (buf) => Array.from(new Uint8Array(buf))
        .map((b) => b.toString(16).padStart(2, '0')).join(''),
    );
    if (cache.size >= maxEntries) cache.clear();
    cache.set(token, p);
    // A failed digest must not poison the cache.
    p.catch(() => { if (cache.get(token) === p) cache.delete(token); });
    return p;
  };
  hashToken.stats = () => ({ hits, misses, size: cache.size });
  return hashToken;
}

// ── Resolution (the card's order + F3) ──────────────────────────────────────

const offerShape = (d) => ({
  code_id: d.id, campaign_id: d.campaign_id || null, expires_at: d.expires_at,
});
const byExpiry = (a, b) => Date.parse(a.expires_at) - Date.parse(b.expires_at);

/**
 * The scanner's resolution engine. Order (slate/roadmap, verbatim intent):
 * the LOCAL REPLICA first (the full server-side list once synced — P-KR2),
 * then the QR-EMBEDDED offer (D-KR3) for a customer in no replica (the
 * just-signed-up walk-up), else unknownCode (F2 — the override lives at
 * submit, Card 6). F3 rides on the codes replica's redeemed state: OFFLINE a
 * locally-redeemed code rejects as spentLocally; ONLINE the local flag never
 * rejects — the atomic server check decides at submit. Every expiry decision
 * is clock.isExpired (§5.1 — offset-adjusted, NEVER raw Date.now()).
 *
 * @param {object} deps
 * @param {object}  deps.codesCollection   Card 2's codes-replica RxCollection
 * @param {object}  deps.offersCollection  Card 2's offers-replica RxCollection
 * @param {function} deps.resolveOffers    Card 2's resolveOffers(col, hash, {now})
 * @param {object}  deps.clock             Card 4's createSyncClock instance
 * @param {function} deps.hashToken        createTokenHasher instance
 * @returns {{resolve: function(string, {online?: boolean}=): Promise<object>}}
 *   result kinds: invalidPayload | offerReady | spentLocally | deferToServer |
 *   expiredLocally | embeddedOffer | unknownCode — every result except
 *   invalidPayload carries token_hash (never the raw token).
 */
export function createScanResolver({
  codesCollection, offersCollection, resolveOffers, clock, hashToken,
}) {
  async function resolve(payload, { online = false } = {}) {
    const token = extractToken(payload);
    if (!token) return { kind: 'invalidPayload' };
    const token_hash = await hashToken(token);
    const embedded = parseEmbeddedOffer(payload);

    // 1 — replica first: the offers replica IS the synced customer's full
    // server-side list (P-KR2), redeemed/expired rows already excluded.
    let offers = await resolveOffers(offersCollection, token_hash, { now: clock.now });
    let codeDocs = null;
    if (!offers.length) {
      // The two replicas pull independently and their windows differ (§5.3
      // codes window ⊇ live offers window) — a live row the codes replica
      // holds is replica truth even before the offers replica has it.
      codeDocs = await codesCollection.find({ selector: { token_hash } }).exec();
      offers = codeDocs
        .filter((d) => !d.redeemed_at && !clock.isExpired(d.expires_at))
        .map(offerShape);
    } else {
      offers = offers.map((o) => ({ ...o }));
    }
    if (offers.length) {
      return { kind: 'offerReady', token_hash, offers: offers.sort(byExpiry), source: 'replica' };
    }

    // 2 — known customer, no live offer: F3 / local expiry verdicts.
    if (codeDocs === null) {
      codeDocs = await codesCollection.find({ selector: { token_hash } }).exec();
    }
    if (codeDocs.length) {
      const redeemed = codeDocs
        .filter((d) => d.redeemed_at)
        .sort((a, b) => Date.parse(b.redeemed_at) - Date.parse(a.redeemed_at));
      if (redeemed.length) {
        const r = redeemed[0];
        if (!online) {
          // F3 offline: obviously-spent codes reject instantly.
          return {
            kind: 'spentLocally', token_hash,
            redeemed_at: r.redeemed_at, redeemed_by: r.redeemed_by || null,
          };
        }
        // F3 online: do NOT reject on the local flag alone — the atomic
        // server check decides at submit (Card 6 mounts there).
        return {
          kind: 'deferToServer', token_hash,
          redeemed_at: r.redeemed_at, redeemed_by: r.redeemed_by || null,
          embedded,
        };
      }
      // Unredeemed but dead by the offset clock (§5.1).
      const latest = codeDocs
        .slice().sort((a, b) => Date.parse(b.expires_at) - Date.parse(a.expires_at))[0];
      return { kind: 'expiredLocally', token_hash, expires_at: latest.expires_at };
    }

    // 3 — in no replica: the not-yet-synced walk-up (D-KR3 fallback), else F2.
    if (embedded) {
      return {
        kind: 'embeddedOffer', token_hash, offer: embedded, source: 'embedded',
        expired: embedded.expires_at ? clock.isExpired(embedded.expires_at) : false,
      };
    }
    return { kind: 'unknownCode', token_hash };
  }

  return { resolve };
}

// ── Serialized enqueue (Card 6's mandatory entry point) ─────────────────────

/**
 * Card 3's enqueueAttempt dedupe is find-then-insert, NOT atomic (landed-card
 * G6 note): two concurrent same-code calls can BOTH miss the find and both
 * insert. This wrapper serializes enqueues PER code_id — at most one in-flight
 * enqueue per code on this device; later calls chain behind the earlier one
 * and therefore see its insert. Card 6's submit flow MUST enqueue through this
 * wrapper (window.MarketingScan.enqueue), never raw enqueueAttempt.
 *
 * A failed enqueue does not poison the chain (the tail swallows rejections;
 * the caller still receives its own rejection).
 */
export function makeSerializedEnqueue(enqueueAttempt, attemptsCollection) {
  const tails = new Map(); // code_id -> settled-safe promise tail
  return function enqueue(fields, opts) {
    const key = fields && fields.code_id;
    const prev = tails.get(key) || Promise.resolve();
    const run = prev.then(() => enqueueAttempt(attemptsCollection, fields, opts));
    const tail = run.catch(() => {});
    tails.set(key, tail);
    tail.then(() => { if (tails.get(key) === tail) tails.delete(key); });
    return run;
  };
}
