// Edge cases: all must produce a decent card or a kind message — never a
// stack trace. Covers zero sessions, rookie corpora, corrupt lines, missing
// model matches, and the OpenCode sqlite path (built in-memory, no binary
// fixture committed).
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import os from 'node:os';
import { mkdtempSync, mkdirSync, rmSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { collect } from '../src/pipeline.js';
import { parseSession } from '../src/parse.js';
import { buildSVG } from '../src/card.js';

const HOME = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'home');

test('zero sessions: empty report, no throw', async () => {
  const empty = mkdtempSync(path.join(os.tmpdir(), 'sw-empty-'));
  try {
    const r = await collect({ agent: 'all', home: empty });
    assert.equal(r.totals.sessions, 0);
    assert.equal(r.meta.rookie, false); // rookie needs at least one session
  } finally {
    rmSync(empty, { recursive: true, force: true });
  }
});

test('model filter with no matches: empty report, no throw', async () => {
  const r = await collect({ agent: 'all', home: HOME, model: 'no-such-model' });
  assert.equal(r.totals.sessions, 0);
});

test('rookie corpus (<10 sessions) still renders every theme', async () => {
  const r = await collect({ agent: 'all', home: HOME });
  assert.equal(r.meta.rookie, true);
  for (const theme of ['sunset', 'terminal', 'starfield', 'receipt', 'billboard', 'crt']) {
    const svg = buildSVG(r, { cut: 'classic', theme });
    assert.ok(svg.includes('ROOKIE CARD'), `${theme}: rookie tag missing`);
    assert.ok(svg.startsWith('<svg'), `${theme}: did not render`);
  }
});

test('corrupt jsonl lines are skipped and counted, parsing continues', async () => {
  const r = await collect({ agent: 'claude', home: HOME });
  assert.equal(r.meta.skippedLines, 1);
  assert.equal(r.totals.prompts, 2); // events after the corrupt line still landed
});

test('opencode sqlite sessions parse (built in a temp db)', async (t) => {
  let DatabaseSync;
  try {
    ({ DatabaseSync } = process.getBuiltinModule('node:sqlite'));
  } catch {
    t.skip('node:sqlite unavailable on this runtime');
    return;
  }
  const dir = mkdtempSync(path.join(os.tmpdir(), 'sw-oc-'));
  const dbPath = path.join(dir, 'opencode.db');
  const db = new DatabaseSync(dbPath);
  db.exec(`
    CREATE TABLE session (id TEXT PRIMARY KEY, time_updated INTEGER);
    CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT, time_created INTEGER, time_updated INTEGER, data TEXT);
    CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT, data TEXT);
  `);
  db.prepare('INSERT INTO session VALUES (?, ?)').run('ses_1', 1000);
  db.prepare('INSERT INTO message VALUES (?, ?, ?, ?, ?)').run(
    'msg_u1', 'ses_1', 1735689600000, 1735689600000,
    JSON.stringify({ role: 'user', time: { created: 1735689600000 }, path: { cwd: '/tmp/oc-proj' } }),
  );
  db.prepare('INSERT INTO part VALUES (?, ?, ?)').run('prt_1', 'msg_u1', JSON.stringify({ type: 'text', text: 'wait, fix this please' }));
  db.prepare('INSERT INTO message VALUES (?, ?, ?, ?, ?)').run(
    'msg_a1', 'ses_1', 1735689660000, 1735689660000,
    JSON.stringify({ role: 'assistant', time: { created: 1735689660000 }, modelID: 'google/gemini-3-pro', tokens: { input: 10, output: 20, cache: { read: 5 } } }),
  );
  db.prepare('INSERT INTO part VALUES (?, ?, ?)').run('prt_2', 'msg_a1', JSON.stringify({ type: 'text', text: 'Good catch. Let me fix it.' }));
  db.close();

  try {
    const events = [];
    const info = await parseSession({ agent: 'opencode', kind: 'sqlite', path: dbPath, sessionId: 'ses_1' }, (ev) => events.push(ev));
    assert.equal(info.events, 2);
    assert.equal(events[0].role, 'human');
    assert.equal(events[0].text, 'wait, fix this please');
    assert.equal(events[1].role, 'assistant');
    assert.equal(events[1].model, 'google/gemini-3-pro');
    assert.deepEqual(events[1].usage, { in: 10, out: 20, cacheRead: 5 });
    assert.equal(events[1].cwd, '/tmp/oc-proj');
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('gemini-only corpus renders a card (assistant stats simply absent)', async () => {
  const r = await collect({ agent: 'gemini', home: HOME });
  assert.equal(r.totals.sessions, 1);
  assert.equal(r.totals.replies, 0);
  const svg = buildSVG(r, { cut: 'classic', theme: 'sunset' });
  assert.ok(svg.startsWith('<svg'));
});
