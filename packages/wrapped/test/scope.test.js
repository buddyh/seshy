import test from 'node:test';
import assert from 'node:assert/strict';
import { applyScope } from '../src/cli.js';

const NOW = Date.parse('2026-07-07T12:00:00Z');

test('scope: Enter and "1" mean fable week', () => {
  for (const c of ['', '1', ' 1 ']) {
    const o = applyScope({ model: '', since: 0, window: 'all' }, c, NOW);
    assert.equal(o.model, 'fable');
    assert.equal(o.since, Date.parse('2026-07-01'));
  }
});

test('scope: "2" is the trailing 30 days, model unfiltered', () => {
  const o = applyScope({ model: '', since: 0, window: 'all' }, '2', NOW);
  assert.equal(o.model, '');
  assert.equal(o.since, NOW - 30 * 86400000);
});

test('scope: "3" leaves everything untouched', () => {
  const o = applyScope({ model: '', since: 0, window: 'all' }, '3', NOW);
  assert.equal(o.model, '');
  assert.equal(o.since, 0);
});
