// Golden-fixture test: every number here was counted BY HAND against the
// synthetic fixtures in test/fixtures/home. If any assertion drifts, a stat
// definition changed — update STATS.md and the golden file deliberately, or
// fix the regression. Timezone is pinned to UTC by the npm test script.
import test from 'node:test';
import assert from 'node:assert/strict';
import path from 'node:path';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { collect, stableStringify } from '../src/pipeline.js';

const HOME = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'home');
const GOLDEN = path.join(path.dirname(fileURLToPath(import.meta.url)), 'fixtures', 'golden-stats.json');

test('hand-verified counts across claude + codex + gemini fixtures', async () => {
  const r = await collect({ agent: 'all', home: HOME });

  // Corpus shape
  assert.equal(r.totals.sessions, 3);
  assert.equal(r.totals.projects, 2); // gemini logs carry no cwd
  assert.equal(r.totals.prompts, 5);
  assert.equal(r.totals.interrupts, 1);
  assert.equal(r.meta.rookie, true);
  assert.equal(r.meta.skippedLines, 1); // the deliberately corrupt claude line

  // Human phrases
  assert.equal(r.you.fbombs, 1); // "fuck"
  assert.equal(r.you.please, 1);
  assert.equal(r.you.thanks, 1); // gemini "thanks!"
  assert.equal(r.you.youWrong, 1); // "still broken"
  assert.equal(r.alt.actually, 1);
  assert.equal(r.alt.wait, 1); // codex "wait, why??"
  assert.equal(r.alt.why, 2); // codex + gemini
  assert.equal(r.alt.doubleQ, 1); // "why??"
  assert.equal(r.alt.just, 1);
  assert.equal(r.alt.greenlights, 2); // "continue" + "do it"
  assert.equal(r.alt.shortOrders, 2); // "continue" + "thanks!"
  assert.equal(r.deep.continueOnly, 1);
  assert.equal(r.totals.userWords, 10 + 1 + 5 + 4 + 1); // fix-prompt, continue, codex, gemini x2

  // Assistant tics
  assert.equal(r.tics.absolutelyRight, 1);
  assert.equal(r.tics.youreRight, 1);
  assert.equal(r.tics.letMe, 1);
  assert.equal(r.tics.perfect, 1);
  assert.equal(r.tics.shouldWork, 1);
  assert.equal(r.tics.goodCatch, 1);
  assert.equal(r.tics.agentSorry, 1);
  assert.equal(r.totals.replies, 3); // two claude text replies + one codex

  // Machine metrics
  assert.equal(r.machine.tokensIn, 100 + 10 + 5 + 5 + 200);
  assert.equal(r.machine.tokensOut, 50 + 5 + 5 + 5 + 100);
  assert.equal(r.machine.cacheRead, 1000 + 50);
  assert.equal(r.machine.toolCalls, 3); // Edit + Bash + codex shell
  assert.equal(r.machine.bashCmds, 2);
  assert.equal(r.machine.linesWritten, 3); // Edit new_string: 3 lines
  assert.equal(r.machine.filesTouched, 1);
  assert.equal(r.deep.gitCommits, 2);
  assert.equal(r.deep.scary, 1); // rm -rf
  assert.equal(r.deep.editCalls, 1);
  assert.equal(r.deep.reads, 0);
  assert.equal(r.deep.toolResults, 2);
  assert.equal(r.deep.toolErrors, 1);
  assert.equal(r.deep.errorRate, 50);

  // Commands, models, streaks
  assert.deepEqual(r.deep.commands, { '/goal': 1, '/help': 1 });
  assert.equal(r.deep.goals, 1);
  assert.deepEqual(r.deep.models, { 'claude-fable-5': 2, 'gpt-5.4': 1 });
  assert.equal(r.deep.topModel.id, 'claude-fable-5');
  assert.equal(r.deep.longestStreak, 3); // Jan 5, 6, 7
  assert.equal(r.you.nightOwl, 0); // 10:00, 23:30, 09:00 UTC — none before 05:00
  assert.equal(r.automation.headless, 0);
});

test('model filter scopes to turns answered by that model', async () => {
  const r = await collect({ agent: 'all', home: HOME, model: 'fable' });
  assert.equal(r.totals.sessions, 1);
  assert.equal(r.totals.prompts, 2);
  assert.equal(r.you.fbombs, 1);
  assert.equal(r.tics.goodCatch, 0); // codex session excluded
  assert.deepEqual(Object.keys(r.deep.models), ['claude-fable-5']);
});

test('report matches the committed golden stats.json byte-for-byte', async () => {
  const r = await collect({ agent: 'all', home: HOME });
  const expected = readFileSync(GOLDEN, 'utf8');
  assert.equal(stableStringify(r), expected);
});
