import { runConformance } from './conformance.mjs';
import { createHandrolled } from './machine-handrolled.mjs';

const ok = runConformance('hand-rolled', (input, effects) => createHandrolled(input, effects));
process.exit(ok ? 0 : 1);
