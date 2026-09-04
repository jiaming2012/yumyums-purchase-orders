import { runConformance } from './conformance.mjs';
import { createXstateOverlay } from './machine-xstate-overlay.mjs';

const ok = runConformance('xstate-overlay', (input, effects) => createXstateOverlay(input, effects));
process.exit(ok ? 0 : 1);
