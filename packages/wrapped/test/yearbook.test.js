// Yearbook titles: deterministic salience ladder — rare tier beats
// personality beats floor; within a tier the hardest-cleared bar wins.
import test from 'node:test';
import assert from 'node:assert/strict';
import { yearbook } from '../src/yearbook.js';

// Minimal report with everything zeroed; override per case.
function mkReport(over = {}) {
  const base = {
    totals: { prompts: 0, projects: 1, sessions: 1, interrupts: 0, userWords: 0, assistantWords: 0 },
    tics: { youreRight: 0, agentSorry: 0, goodCatch: 0 },
    you: { fbombs: 0, nightOwl: 0, sorry: 0, please: 0, thanks: 0, youWrong: 0 },
    alt: { wait: 0, actually: 0, just: 0, shortOrders: 0, greenlights: 0, undo: 0, why: 0, capsRage: 0 },
    deep: { subagents: 0, longestStreak: 0, gitCommits: 0, reads: 0, editCalls: 0, continueOnly: 0 },
    machine: { bashCmds: 0, linesWritten: 0 },
    automation: { headless: 0, headlessShare: 0 },
    awards: { longestSession: { msgs: 0 } },
    time: { activeHours: 0 },
  };
  for (const [k, v] of Object.entries(over)) base[k] = { ...base[k], ...v };
  return base;
}

test('everyone gets a title: empty report lands on the floor', () => {
  const y = yearbook(mkReport());
  assert.equal(y.title, 'THE HUMAN IN THE LOOP');
  assert.match(y.receipt, /prompts across/);
});

test('rare tier beats a higher-salience personality title', () => {
  // greenlights cleared 5x its bar, but subagents (rare) only 1.25x — rare wins.
  const y = yearbook(mkReport({
    deep: { subagents: 25 },
    alt: { greenlights: 150 },
  }));
  assert.equal(y.title, 'CERTIFIED DOM');
  assert.match(y.receipt, /25 subagents/);
});

test('within a tier the hardest-cleared bar wins', () => {
  // automation 65/40 = 1.63 vs greenlights 42/30 = 1.4 -> automation.
  const y = yearbook(mkReport({
    automation: { headless: 60, headlessShare: 65.2 },
    alt: { greenlights: 42 },
  }));
  assert.equal(y.title, 'MOST LIKELY TO AUTOMATE THEMSELVES OUT OF TYPING');
  assert.match(y.receipt, /65.2% of everything/);
});

test('zero-valued titles require the proving denominator', () => {
  // 0 interrupts only counts with >= 200 prompts behind it.
  const few = yearbook(mkReport({ totals: { prompts: 50, interrupts: 0 } }));
  assert.notEqual(few.title, 'NO SAFE WORD NEEDED');
  const many = yearbook(mkReport({ totals: { prompts: 300, interrupts: 0 } }));
  assert.equal(many.title, 'NO SAFE WORD NEEDED');
  assert.match(many.receipt, /0 interruptions in 300 prompts/);
});

test('subagent ladder: middle management below 20, dom at 20+', () => {
  assert.equal(yearbook(mkReport({ deep: { subagents: 9 } })).title, 'MIDDLE MANAGEMENT');
  assert.equal(yearbook(mkReport({ deep: { subagents: 20 } })).title, 'CERTIFIED DOM');
});

test('deterministic: same report, same title, every time', () => {
  const r = mkReport({ alt: { wait: 45, undo: 20 }, you: { nightOwl: 25 } });
  const a = yearbook(r);
  for (let i = 0; i < 5; i++) assert.deepEqual(yearbook(mkReport({ alt: { wait: 45, undo: 20 }, you: { nightOwl: 25 } })), a);
});
