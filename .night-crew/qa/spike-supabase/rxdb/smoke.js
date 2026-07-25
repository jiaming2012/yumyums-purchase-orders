// smoke.js — the cheapest possible check that the two shims in spike-env.js
// actually reach W1's stack, BEFORE any RxDB is involved.
//
// It catches the failure "my replication does not work" being really "my
// client never reached PostgREST / Realtime at all", which is the single most
// misleading way a sync spike wastes a night.
import { makeSupabaseClient, mintToken, TABLE, banner } from './spike-env.js';

banner();
const token = mintToken('user-alice');
console.log('token len', token.length, 'segments', token.split('.').length);
const sb = makeSupabaseClient(token);

// 1. REST reachable + RLS shape (alice must see her seed row, not bob's)
const { data, error } = await sb.from(TABLE).select('*');
console.log('REST error:', error);
console.log('REST rows visible to user-alice:', data);

// 2. Realtime reachable — did the channel actually SUBSCRIBE?
const status = await new Promise((resolve) => {
    const t = setTimeout(() => resolve('TIMEOUT'), 20000);
    sb.channel(`smoke-${Date.now()}`)
        .on('postgres_changes', { event: '*', schema: 'public', table: TABLE },
            (p) => console.log('realtime payload', p.eventType, p.new))
        .subscribe((s, err) => {
            console.log('realtime status:', s, err || '');
            if (s === 'SUBSCRIBED' || s === 'CHANNEL_ERROR' || s === 'TIMED_OUT') {
                clearTimeout(t);
                resolve(s);
            }
        });
});
console.log('FINAL realtime status:', status);
process.exit(0);
