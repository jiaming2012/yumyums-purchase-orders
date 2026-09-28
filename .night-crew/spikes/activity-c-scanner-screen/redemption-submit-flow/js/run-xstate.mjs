import { runConformance } from './conformance.mjs';
import { createXstate } from './machine-xstate.mjs';

const ok = runConformance('xstate-v5', (input, effects) => createXstate(input, effects));
process.exit(ok ? 0 : 1);
