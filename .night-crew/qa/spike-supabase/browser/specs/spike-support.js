// spike-support.js — what all five legs need in common.
//
// ⚠ LOCAL SPIKE ONLY. The JWT secret below is the throwaway one committed in
//   docker-compose.supabase.yml on purpose. See ../../README.md.

const { execFileSync } = require('node:child_process');
const path = require('node:path');

const BROWSER_DIR = path.resolve(__dirname, '..');
const SPIKE_DIR = path.resolve(BROWSER_DIR, '..');            // .night-crew/qa/spike-supabase
const REPO_ROOT = path.resolve(SPIKE_DIR, '..', '..', '..');

// Same throwaway value as docker-compose.supabase.yml's JWT_SECRET.
const JWT_SECRET =
    '2508c659af3c4316b0a163a00725d33a9bc4eae75aa35ac9be6a007cacb8251c';

const TABLE = 'spike_notes';

function composePort(service, containerPort) {
    const out = execFileSync(
        'docker',
        ['compose', '-p', 'spike-supabase', '-f', 'docker-compose.supabase.yml',
            'port', service, String(containerPort)],
        { cwd: REPO_ROOT, encoding: 'utf8' }
    ).trim();
    return Number(out.split(':').pop());
}

const REST_PORT = Number(process.env.SPIKE_REST_PORT || composePort('rest', 3000));
const REALTIME_PORT = Number(process.env.SPIKE_REALTIME_PORT || composePort('realtime', 4000));

/**
 * Mint with W1's Go program, on purpose — not a JS re-implementation. The whole
 * premise of the migration is that HQ's GO BACKEND is the token authority, and
 * a spike that signs its own tokens in JS proves nothing about that.
 *
 * `ttl` is a Go time.Duration string: '1h', '20s', '-5m'.
 */
function mintToken(sub, { ttl = '1h', expired = false } = {}) {
    const args = ['run', './mintjwt', '-secret', JWT_SECRET, '-sub', sub, '-ttl', ttl];
    if (expired) args.push('-expired');
    return execFileSync('go', args, {
        cwd: SPIKE_DIR,
        encoding: 'utf8',
        env: { ...process.env, PATH: `/usr/local/go/bin:${process.env.PATH}` }
    }).trim();
}

/**
 * Read rows back over an INDEPENDENT request — straight at PostgREST's own
 * host port, bypassing the harness's proxy, the page, the service worker and
 * RxDB's own view of the world. W2's proofs did this for the same reason: a
 * replication client's opinion that it is "in sync" is not evidence.
 */
async function rowsById(token, ids) {
    const list = ids.map(i => `"${i}"`).join(',');
    const res = await fetch(
        `http://127.0.0.1:${REST_PORT}/${TABLE}?id=in.(${list})&select=*`,
        { headers: { Authorization: `Bearer ${token}` } }
    );
    return { status: res.status, rows: await res.json() };
}

/** A per-run id prefix, so no run can ever collide with a previous run's rows. */
const RUN = process.env.SPIKE_RUN_ID || `c${Date.now()}`;

module.exports = {
    JWT_SECRET, TABLE, REST_PORT, REALTIME_PORT, RUN,
    SPIKE_DIR, REPO_ROOT, mintToken, rowsById
};
