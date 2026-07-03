// Regenerate the golden stats snapshot after a DELIBERATE stat change:
//   TZ=UTC node test/update-golden.js
// Then re-verify the hand-counted assertions in golden.test.js still hold,
// and update STATS.md if a definition moved.
import path from 'node:path';
import { writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { collect, stableStringify } from '../src/pipeline.js';

const HERE = path.dirname(fileURLToPath(import.meta.url));
const report = await collect({ agent: 'all', home: path.join(HERE, 'fixtures', 'home') });
writeFileSync(path.join(HERE, 'fixtures', 'golden-stats.json'), stableStringify(report));
console.log('golden-stats.json regenerated');
