// serve.mjs — the ONE origin the browser spike runs against. Zero dependencies.
//
// ⚠ LOCAL SPIKE ONLY. See ../README.md for the banner. Never run this against
//   anything real; it proxies unauthenticated to a throwaway PostgREST.
//
// WHY A CUSTOM SERVER AND NOT `python3 -m http.server`
// ---------------------------------------------------
// The service-worker leg is the sharpest edge in this card and it only bites
// when the replication endpoint is SAME-ORIGIN with the page. HQ's Workbox
// `sw.js` registers exactly one runtime route — `NetworkFirst` on /\/api\//
// with a `handlerDidError` that answers `{"error":"offline"}` with HTTP 503.
// A cross-origin PostgREST on 127.0.0.1:46233 never matches that pattern, so a
// naive cross-origin harness would "pass" the service-worker leg by never
// touching the trap at all. That is precisely the kind of false green this card
// exists to avoid.
//
// So this server publishes the SAME PostgREST under TWO same-origin paths:
//
//   /rest/v1/*            -> PostgREST   (does NOT match /\/api\//)
//   /api/v1/rest/v1/*     -> PostgREST   (DOES match /\/api\//  <- the trap)
//
// and supabase-js picks which one it uses purely from the base URL it is given,
// because it derives `<url>/rest/v1` internally:
//
//   createClient('http://127.0.0.1:PORT')          -> /rest/v1/...
//   createClient('http://127.0.0.1:PORT/api/v1')   -> /api/v1/rest/v1/...
//
// No fetch shim is needed on the REST side at all — which is itself a finding,
// since W2's Node harness needed one (spike-env.js §3). Fronting Supabase on
// HQ's own origin removes the Kong-shaped problem and creates the Workbox one.
//
// Everything else is served statically from the REPO ROOT, so the page loads
// the real `/sw.js` (scope `/`) and the real `/vendor/rxdb.bundle.js`.

import http from 'node:http';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { execFileSync } from 'node:child_process';

const HERE = path.dirname(fileURLToPath(import.meta.url));
// browser/ -> spike-supabase/ -> qa/ -> .night-crew/ -> repo root
const REPO_ROOT = path.resolve(HERE, '..', '..', '..', '..');

const PORT = Number(process.env.SPIKE_HTTP_PORT || 8497);

// Host ports are Docker-assigned and change on every `up` — never hardcode.
function composePort(service, containerPort) {
    const out = execFileSync(
        'docker',
        ['compose', '-p', 'spike-supabase', '-f', 'docker-compose.supabase.yml',
            'port', service, String(containerPort)],
        { cwd: REPO_ROOT, encoding: 'utf8' }
    ).trim();
    const port = out.split(':').pop();
    if (!port) throw new Error(`could not resolve host port for ${service}: ${out}`);
    return Number(port);
}

const REST_PORT = Number(process.env.SPIKE_REST_PORT || composePort('rest', 3000));
const REALTIME_PORT = Number(process.env.SPIKE_REALTIME_PORT || composePort('realtime', 4000));
const REST_ORIGIN = `http://127.0.0.1:${REST_PORT}`;

const MIME = {
    '.html': 'text/html; charset=utf-8',
    '.js': 'text/javascript; charset=utf-8',
    '.mjs': 'text/javascript; charset=utf-8',
    '.json': 'application/json; charset=utf-8',
    '.css': 'text/css; charset=utf-8',
    '.png': 'image/png',
    '.ico': 'image/x-icon',
    '.map': 'application/json; charset=utf-8'
};

// Counters the specs read back, so a claim like "the service worker never saw
// the WebSocket handshake" is backed by a number from the server side rather
// than by an assertion about the spec.
const hits = { rest: 0, apiRest: 0, ping: 0, static: 0, ws: 0 };

function send(res, status, body, headers = {}) {
    res.writeHead(status, { 'Cache-Control': 'no-store', ...headers });
    res.end(body);
}

async function proxyToPostgrest(req, res, rest) {
    const target = REST_ORIGIN + '/' + rest;
    const headers = {};
    for (const [k, v] of Object.entries(req.headers)) {
        // Host must not be forwarded; hop-by-hop headers must not either.
        if (['host', 'connection', 'content-length'].includes(k)) continue;
        headers[k] = v;
    }
    let body;
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        const chunks = [];
        for await (const c of req) chunks.push(c);
        body = Buffer.concat(chunks);
    }
    try {
        const upstream = await fetch(target, { method: req.method, headers, body });
        const buf = Buffer.from(await upstream.arrayBuffer());
        const out = {};
        upstream.headers.forEach((v, k) => {
            if (['content-encoding', 'transfer-encoding', 'connection'].includes(k)) return;
            out[k] = v;
        });
        res.writeHead(upstream.status, out);
        res.end(buf);
    } catch (err) {
        send(res, 502, JSON.stringify({ proxy_error: String(err) }),
            { 'Content-Type': 'application/json' });
    }
}

const server = http.createServer(async (req, res) => {
    const url = new URL(req.url, `http://127.0.0.1:${PORT}`);
    const p = url.pathname;

    // CORS is not needed (everything is same-origin) but PostgREST wants the
    // preflight answered if a spec ever goes cross-origin deliberately.
    if (req.method === 'OPTIONS') {
        return send(res, 204, '', {
            'Access-Control-Allow-Origin': '*',
            'Access-Control-Allow-Headers': '*',
            'Access-Control-Allow-Methods': 'GET,POST,PATCH,PUT,DELETE,OPTIONS'
        });
    }

    // ---- the trap route: same-origin AND matching HQ's /\/api\// pattern ----
    let m = p.match(/^\/api\/v1\/rest\/v1\/(.*)$/);
    if (m) {
        hits.apiRest++;
        return proxyToPostgrest(req, res, m[1] + url.search);
    }

    // ---- the safe route: same-origin, NOT matching /\/api\// ----------------
    m = p.match(/^\/rest\/v1\/(.*)$/);
    if (m) {
        hits.rest++;
        return proxyToPostgrest(req, res, m[1] + url.search);
    }

    // ---- a plain HQ-shaped API endpoint, for the offline-fallback probe -----
    if (p === '/api/v1/spike-ping') {
        hits.ping++;
        return send(res, 200,
            JSON.stringify({ ok: true, from: 'spike serve.mjs', hits: hits.ping }),
            { 'Content-Type': 'application/json' });
    }

    // ---- the harness's own view of what it served --------------------------
    if (p === '/__spike/hits') {
        return send(res, 200, JSON.stringify(hits), { 'Content-Type': 'application/json' });
    }
    if (p === '/__spike/config') {
        return send(res, 200, JSON.stringify({
            restPort: REST_PORT, realtimePort: REALTIME_PORT, httpPort: PORT
        }), { 'Content-Type': 'application/json' });
    }

    // ---- static, from the repo root ----------------------------------------
    let rel = decodeURIComponent(p);
    if (rel === '/') rel = '/index.html';
    const file = path.join(REPO_ROOT, rel);
    if (!file.startsWith(REPO_ROOT)) return send(res, 403, 'no');
    fs.readFile(file, (err, buf) => {
        if (err) return send(res, 404, 'not found');
        hits.static++;
        send(res, 200, buf, {
            'Content-Type': MIME[path.extname(file)] || 'application/octet-stream',
            // Service-Worker-Allowed is NOT set: /sw.js sits at the root so its
            // default scope is already '/'. Setting it would hide a real
            // deployment constraint behind a harness convenience.
            ...(path.basename(file) === 'sw.js' ? { 'Service-Worker-Allowed': '/' } : {})
        });
    });
});

// ---------------------------------------------------------------------------
// A REAL WebSocket endpoint, mounted UNDER /api/ on purpose.
//
// Scope item 3 asks about the service worker's interaction with "a long-lived
// Realtime WebSocket". Dialing self-hosted Realtime from the browser drags in
// W1's tenant-vhost sharp edge and would answer a question about REALTIME when
// the question is about the SERVICE WORKER. So this server completes a real
// RFC 6455 handshake itself, at `/api/v1/spike-ws` — a path that MATCHES HQ's
// `/\/api\//` runtime route exactly. If the SW were going to touch a WebSocket
// handshake, this is where it would.
//
// ~12 lines and no dependency: the handshake is a SHA-1 of the client key plus
// the RFC's magic GUID, base64'd, returned with a 101.
// ---------------------------------------------------------------------------
const WS_GUID = '258EAFA5-E914-47DA-95CA-C5AB0DC85B11';

server.on('upgrade', async (req, socket) => {
    hits.ws++;
    const { createHash } = await import('node:crypto');
    const key = req.headers['sec-websocket-key'];
    if (!key || !/^\/api\/v1\/spike-ws/.test(req.url || '')) return socket.destroy();
    const accept = createHash('sha1').update(key + WS_GUID).digest('base64');
    socket.write(
        'HTTP/1.1 101 Switching Protocols\r\n' +
        'Upgrade: websocket\r\n' +
        'Connection: Upgrade\r\n' +
        `Sec-WebSocket-Accept: ${accept}\r\n\r\n`
    );
    // Left open deliberately — "long-lived" is the property under test. No
    // frames are sent; the client only needs onopen to fire.
    socket.on('error', () => { /* client went away */ });
});

server.listen(PORT, '127.0.0.1', () => {
    console.log(`spike serve.mjs  http://127.0.0.1:${PORT}  root=${REPO_ROOT}`);
    console.log(`  /rest/v1/*        -> ${REST_ORIGIN}   (NOT matched by HQ sw.js)`);
    console.log(`  /api/v1/rest/v1/* -> ${REST_ORIGIN}   (MATCHED by HQ sw.js /\\/api\\//)`);
    console.log(`  realtime host port ${REALTIME_PORT}`);
});
