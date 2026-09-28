// gen-qr.mjs — generate the spike's QR PNG in the #10 hybrid payload shape
// (URL wrapping the identity token) and print Node's own sha256 of the token
// (the cross-check leg — a seed-literal typo must not define the contract).
// argv: <token> <outPng>
import { createHash } from 'node:crypto';
import QRCode from 'qrcode';

const [token, outPng] = process.argv.slice(2);
if (!token || !outPng) { console.error('usage: gen-qr.mjs <token> <outPng>'); process.exit(2); }

const payload = `https://hq.yumyums.kitchen/r/${token}`;
await QRCode.toFile(outPng, payload, { width: 512, margin: 2 });
const nodeHash = createHash('sha256').update(token, 'utf8').digest('hex');
console.log(JSON.stringify({ payload, token, nodeHash }));
