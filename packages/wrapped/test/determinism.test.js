// The launch-day quote-tweet test: two runs over the same data must produce
// byte-identical stats and byte-identical SVG markup. No exceptions.
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { collect, stableStringify } from '../src/pipeline.js';
import { buildSVG } from '../src/card.js';
import { CUTS, signature } from '../src/copy.js';

const HOME = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'home');
const THEMES = ['sunset', 'terminal', 'starfield', 'receipt', 'billboard', 'crt'];

test('double run produces byte-identical stats.json', async () => {
  const a = stableStringify(await collect({ agent: 'all', home: HOME }));
  const b = stableStringify(await collect({ agent: 'all', home: HOME }));
  assert.equal(a, b);
});

test('double run with model filter is byte-identical too', async () => {
  const a = stableStringify(await collect({ agent: 'all', home: HOME, model: 'fable' }));
  const b = stableStringify(await collect({ agent: 'all', home: HOME, model: 'fable' }));
  assert.equal(a, b);
});

test('every cut x theme renders identical SVG across runs', async () => {
  const report = await collect({ agent: 'all', home: HOME });
  for (const cut of CUTS) {
    for (const theme of THEMES) {
      const a = buildSVG(report, { cut, theme, handle: '@fixture' });
      const b = buildSVG(report, { cut, theme, handle: '@fixture' });
      assert.equal(a, b, `${theme}/${cut} SVG differs between renders`);
    }
  }
});

test('punchline picker is deterministic and always stat + twist', async () => {
  const report = await collect({ agent: 'all', home: HOME });
  for (const cut of CUTS) {
    const s1 = signature(report, cut);
    const s2 = signature(report, cut);
    assert.deepEqual(s1, s2);
    assert.ok(s1.line1.length > 0, `${cut}: line1 empty`);
    assert.ok(s1.line2.length > 0, `${cut}: punchline has no twist line`);
    assert.ok(/\d/.test(s1.line1), `${cut}: line1 carries no stat`);
  }
});
