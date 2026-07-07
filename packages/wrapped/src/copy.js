// Turn the canonical report into card copy: hero headline, tile grid, and the
// punchline. This is where the wit lives. Everything here is a pure function
// of the report — deterministic by construction.

const C = {
  pink: '#FF6AC1', cyan: '#2DE2E6', aqua: '#05D9E8', yellow: '#FFD319',
  orange: '#FF8E42', violet: '#B026FF', magenta: '#F92AAD', lav: '#B8A6D9',
};

export const fmt = (n) => {
  if (n >= 1_000_000_000) return (n / 1_000_000_000).toFixed(n >= 10_000_000_000 ? 0 : 1).replace(/\.0$/, '') + 'B';
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(n >= 10_000_000 ? 0 : 1).replace(/\.0$/, '') + 'M';
  if (n >= 10_000) return (n / 1000).toFixed(n >= 100_000 ? 0 : 1).replace(/\.0$/, '') + 'K';
  return n.toLocaleString('en-US');
};

const MONTHS = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
const monYr = (ts) => (ts ? `${MONTHS[new Date(ts).getMonth()]} ${new Date(ts).getFullYear()}` : '');
const base = (p) => (p ? p.split('/').filter(Boolean).pop() : 'a repo');

export const CUTS = ['classic', 'machine', 'chaos', 'agent'];

// Header kicker tag per cut, appended after the year.
export const CUT_TAG = { classic: '', machine: 'MACHINE CUT', chaos: 'CHAOS CUT', agent: 'AGENT CUT' };

// 'claude-fable-5' -> 'Fable 5', 'claude-opus-4-8' -> 'Opus 4.8',
// 'gpt-5.4' -> 'GPT-5.4'. Used for the card kicker on --model runs.
export function prettyModel(id) {
  if (!id) return '';
  if (/^gpt/i.test(id)) return id.toUpperCase();
  const parts = id.replace(/^claude-/, '').replace(/-\d{8}$/, '').split('-');
  const out = [];
  for (const p of parts) {
    if (/^\d+$/.test(p) && out.length && /\d$/.test(out[out.length - 1])) out[out.length - 1] += '.' + p;
    else out.push(/^\d/.test(p) ? p : p[0].toUpperCase() + p.slice(1));
  }
  return out.join(' ');
}

export function pickHeadline(report, cut = 'classic') {
  const { totals, span, machine, alt, tics, meta } = report;
  // Day-precise range so a tight window reads "Jul 1 – Jul 7, 2026" instead
  // of a vague "Jul 2026"; a single day collapses to "Jul 1, 2026".
  const monDay = (ts) => `${MONTHS[new Date(ts).getMonth()]} ${new Date(ts).getDate()}`;
  const range = !span.firstTs ? ''
    : (() => {
        const y = new Date(span.lastTs).getFullYear();
        const a = monDay(span.firstTs), b = monDay(span.lastTs);
        return a === b ? `${a}, ${y}` : `${a} – ${b}, ${y}`;
      })();
  const sub = [`${fmt(totals.sessions)} sessions`, `${fmt(totals.projects)} projects`, range]
    .filter(Boolean)
    .join('  ·  ');
  if (meta.rookie) return { value: fmt(totals.prompts), label: 'prompts fired · rookie card', sub };
  if (cut === 'machine') return { value: fmt(machine.linesWritten), label: 'lines of code it wrote for you', sub };
  if (cut === 'chaos') return { value: fmt(alt.actually), label: 'times you said "actually"', sub };
  if (cut === 'agent') return { value: fmt(tics.letMe), label: 'times it said "let me…"', sub };
  return { value: fmt(totals.prompts), label: 'prompts fired', sub };
}

// Ordered candidate tiles per cut; the first six that clear their threshold
// render. Each entry carries its raw number for the gate. Exported so
// tooling (stat pickers, audits) can enumerate the full candidate pool.
export function tilePool(report, cut) {
  const { totals, tics, you, awards, machine, alt, automation, time, deep } = report;
  if (cut === 'machine') {
    const ratio = totals.userWords ? totals.assistantWords / totals.userWords : 0;
    return [
      { raw: machine.tokensTotal, value: fmt(machine.tokensTotal), label: 'tokens burned', accent: C.pink, keep: true },
      { raw: machine.toolCalls, value: fmt(machine.toolCalls), label: 'tool calls it made', accent: C.cyan, keep: true },
      { raw: machine.bashCmds, value: fmt(machine.bashCmds), label: 'shell commands it ran', accent: C.orange, min: 10 },
      { raw: machine.filesTouched, value: fmt(machine.filesTouched), label: 'files it edited', accent: C.yellow, min: 5 },
      { raw: ratio, value: ratio >= 1 ? `${Math.round(ratio)}x` : '—', label: 'words back per word', accent: C.violet, min: 2 },
      { raw: time.activeHours, value: `${Math.round(time.activeHours)}h`, label: 'agent hours on the clock', accent: C.aqua, min: 2 },
      { raw: machine.cacheRead, value: fmt(machine.cacheRead), label: 'cached tokens re-read', accent: C.magenta, min: 1000 },
      { raw: automation.headless, value: fmt(automation.headless), label: 'sessions run headless for you', accent: C.cyan, min: 5 },
      { raw: totals.userWords, value: fmt(totals.userWords), label: 'words you typed', accent: C.yellow, min: 200 },
    ];
  }
  if (cut === 'chaos') {
    return [
      { raw: alt.wait, value: fmt(alt.wait), label: '"wait" — the keyboard brake pedal', accent: C.cyan, keep: true },
      { raw: alt.doubleQ, value: fmt(alt.doubleQ), label: 'double question marks ("??")', accent: C.pink, min: 3 },
      { raw: alt.capsRage, value: fmt(alt.capsRage), label: 'caps-lock outbursts', accent: C.orange, min: 3 },
      { raw: alt.undo, value: fmt(alt.undo), label: 'undos & reverts demanded', accent: C.yellow, min: 3 },
      { raw: alt.shortOrders, value: fmt(alt.shortOrders), label: 'prompts of 3 words or fewer', accent: C.violet, min: 5 },
      { raw: alt.why, value: fmt(alt.why), label: 'times you asked "why"', accent: C.aqua, min: 5 },
      { raw: alt.greenlights, value: fmt(alt.greenlights), label: 'green lights ("do it")', accent: C.magenta, min: 5 },
      { raw: alt.just, value: fmt(alt.just), label: '"just"s (it is never just)', accent: C.pink, min: 10 },
      { raw: you.oneMore, value: fmt(you.oneMore), label: '"one more thing"s', accent: C.cyan, min: 3 },
    ];
  }
  if (cut === 'agent') {
    return [
      { raw: tics.perfect, value: fmt(tics.perfect), label: '"Perfect!"s', accent: C.cyan, keep: true },
      { raw: tics.shouldWork, value: fmt(tics.shouldWork), label: '"should work now" promises', accent: C.yellow, keep: true },
      { raw: tics.iSeeIssue, value: fmt(tics.iSeeIssue), label: '"I see the issue"s', accent: C.pink, min: 3 },
      { raw: tics.absolutelyRight, value: fmt(tics.absolutelyRight), label: '"you’re absolutely right"s', accent: C.orange, min: 3 },
      { raw: tics.goodCatch, value: fmt(tics.goodCatch), label: '"good catch"es', accent: C.violet, min: 3 },
      { raw: Math.min(tics.agentSorry, you.sorry), value: `${fmt(tics.agentSorry)} : ${fmt(you.sorry)}`, label: 'apologies: got vs gave', accent: C.aqua, min: 2 },
      { raw: tics.agentSorry, value: fmt(tics.agentSorry), label: 'apologies issued', accent: C.aqua, min: 2 },
      { raw: tics.agentAdmitWrong, value: fmt(tics.agentAdmitWrong), label: '"I was wrong" confessions', accent: C.magenta, min: 2 },
      { raw: tics.youreRight, value: fmt(tics.youreRight), label: 'times it caved: "you’re right"', accent: C.yellow, min: 3 },
    ];
  }
  // The default card leads with leverage (projects, lines, commits, cache,
  // ratio) and closes with the funny scorelines. F-bombs may show as a zero,
  // but only when there's enough volume behind it for the zero to be a flex.
  const ratio = totals.userWords ? totals.assistantWords / totals.userWords : 0;
  return [
    { raw: totals.projects, value: fmt(totals.projects), label: 'projects touched', accent: C.cyan, keep: true },
    { raw: machine.linesWritten, value: fmt(machine.linesWritten), label: 'lines of code it wrote', accent: C.pink, min: 100 },
    { raw: deep.gitCommits, value: fmt(deep.gitCommits), label: 'git commits landed', accent: C.yellow, min: 5 },
    { raw: machine.cacheRead, value: fmt(machine.cacheRead), label: 'cached tokens re-read', accent: C.magenta, min: 1000 },
    { raw: ratio, value: ratio >= 1 ? `${Math.round(ratio)}x` : '—', label: 'words back per word', accent: C.violet, min: 2 },
    { raw: you.nightOwl, value: fmt(you.nightOwl), label: 'prompts after midnight', accent: C.aqua, min: 5 },
    { raw: tics.agentSorry, value: `${fmt(tics.agentSorry)} : ${fmt(you.sorry)}`, label: 'apologies: got vs gave', accent: C.orange, min: 2 },
    { raw: you.fbombs, value: fmt(you.fbombs), label: 'F-bombs you dropped', accent: C.orange, keep: you.fbombs > 0 || totals.prompts >= 200 },
    { raw: totals.sessions, value: fmt(totals.sessions), label: 'coding sessions', accent: C.cyan, keep: true },
    { raw: totals.assistantWords, value: fmt(totals.assistantWords), label: 'words written for you', accent: C.pink, min: 200 },
    { raw: tics.youreRight, value: fmt(tics.youreRight), label: 'times it caved: "you’re right"', accent: C.yellow, min: 3 },
    { raw: awards.longestSession.msgs, value: fmt(awards.longestSession.msgs), label: 'prompts in one marathon session', accent: C.magenta, min: 40 },
    { raw: totals.interrupts, value: fmt(totals.interrupts), label: 'rage-quit interruptions', accent: C.pink, min: 5 },
    { raw: you.youWrong, value: fmt(you.youWrong), label: 'times you told it it was wrong', accent: C.orange, min: 3 },
    { raw: you.please + you.thanks, value: fmt(you.please + you.thanks), label: 'pleases & thank-yous', accent: C.cyan, min: 3 },
    { raw: tics.goodCatch, value: fmt(tics.goodCatch), label: '"good catch"es', accent: C.yellow, min: 3 },
    { raw: you.enthusiasm, value: fmt(you.enthusiasm), label: 'hype moments ("ship it!")', accent: C.violet, min: 4 },
    { raw: totals.userWords, value: fmt(totals.userWords), label: 'words you typed', accent: C.aqua, min: 200 },
  ];
}

export const TILE_COUNT = 8;

export function pickTiles(report, cut = 'classic') {
  const pool = tilePool(report, cut);
  const chosen = [];
  for (const t of pool) {
    if (chosen.length >= TILE_COUNT) break;
    if (t.keep || t.raw >= (t.min || 0)) chosen.push(t);
  }
  for (const t of pool) {
    if (chosen.length >= TILE_COUNT) break;
    if (!chosen.includes(t)) chosen.push(t);
  }
  return chosen.slice(0, TILE_COUNT);
}

// --- The punchline picker. ------------------------------------------------
// Rule: every punchline is a stat + a twist, never a stat alone. Candidates
// are ranked by twist class — PAIRED (two stats that talk to each other)
// beats STREAK (time-shaped) beats RAW (one big number) — then by a
// deterministic salience score, then by fixed catalog order. Same report,
// same punchline, every time.
const PAIRED = 3;
const STREAK = 2;
const RAW = 1;

export function punchlineCatalog(report) {
  const { totals, tics, you, awards, machine, alt, deep, meta, time } = report;
  const p = base(awards.longestSession.cwd);
  return [
    // PAIRED — two numbers in tension.
    {
      cls: PAIRED, when: tics.shouldWork >= 10 && you.youWrong >= 5, score: tics.shouldWork + you.youWrong,
      line1: `"That should work now" — ${fmt(tics.shouldWork)} times.`,
      line2: `${fmt(you.youWrong)} times it did not.`,
    },
    {
      cls: PAIRED, when: you.fbombs >= 10, score: you.fbombs,
      line1: `You dropped ${fmt(you.fbombs)} F-bombs at your agent.`,
      line2: 'It never dropped one back. The machine wins on composure.',
    },
    {
      cls: PAIRED, when: tics.agentSorry >= 5 && you.sorry >= 1, score: tics.agentSorry + you.sorry,
      line1: `Apologies: ${fmt(tics.agentSorry)} extracted, ${fmt(you.sorry)} given.`,
      line2: 'The exchange rate is brutal and you set it.',
    },
    {
      cls: PAIRED, when: machine.linesWritten >= 1000 && totals.prompts >= 50, score: machine.linesWritten / 100,
      line1: `${fmt(machine.linesWritten)} lines of code, from ${fmt(totals.prompts)} prompts.`,
      line2: 'History’s best words-to-software exchange rate.',
    },
    {
      cls: PAIRED, when: deep.reads >= 100 && deep.editCalls > deep.reads, score: deep.editCalls,
      line1: `It edited more than it read: ${fmt(deep.editCalls)} edits, ${fmt(deep.reads)} reads.`,
      line2: 'Confidence is a tool call.',
    },
    // STREAK — time-shaped stats.
    {
      cls: STREAK, when: deep.longestStreak >= 10, score: deep.longestStreak,
      line1: `${fmt(deep.longestStreak)} days in a row with an agent.`,
      line2: 'Streaks are for duolingo. And, apparently, this.',
    },
    {
      cls: STREAK, when: you.nightOwl >= 40, score: you.nightOwl,
      line1: `${fmt(you.nightOwl)} prompts sent after midnight.`,
      line2: 'The neon grid never sleeps. Apparently neither do you.',
    },
    {
      cls: STREAK, when: awards.longestSession.msgs >= 120, score: awards.longestSession.msgs,
      line1: `One session. ${fmt(awards.longestSession.msgs)} prompts.`,
      line2: `${p} nearly broke you both — and you shipped it.`,
    },
    {
      cls: STREAK, when: time.activeHours >= 100, score: time.activeHours,
      line1: `${Math.round(time.activeHours)} agent hours on the clock.`,
      line2: 'You were only in the room for the interesting parts.',
    },
    // RAW — one big number with a written twist.
    {
      cls: RAW, when: alt.actually >= 100, score: alt.actually,
      line1: `You said "actually" ${fmt(alt.actually)} times.`,
      line2: 'Every single one changed the plan mid-flight.',
    },
    {
      cls: RAW, when: tics.youreRight >= 25, score: tics.youreRight,
      line1: `You bent it to your will ${fmt(tics.youreRight)} times.`,
      line2: "That's how often it folded and admitted you were right.",
    },
    {
      cls: RAW, when: totals.interrupts >= 40, score: totals.interrupts,
      line1: `${fmt(totals.interrupts)} rage-quit interruptions.`,
      line2: 'Esc, esc, esc. Patience is for people without agents.',
    },
    {
      cls: RAW, when: alt.undo >= 10, score: alt.undo,
      line1: `${fmt(alt.undo)} undos and reverts demanded.`,
      line2: 'Ctrl-Z is a lifestyle, not a shortcut.',
    },
    {
      cls: RAW, when: deep.subagents >= 20, score: deep.subagents,
      line1: `${fmt(deep.subagents)} subagents spawned on your behalf.`,
      line2: 'Synthetic headcount. Zero standups.',
    },
    // Rookie + fallback — always available.
    {
      cls: RAW, when: meta.rookie, score: 1e9, // rookies always get this one
      line1: `${fmt(totals.sessions)} sessions in. A rookie card.`,
      line2: 'Everyone’s F-bomb counter starts at zero.',
    },
    {
      cls: RAW, when: true, score: 0,
      line1: `${fmt(totals.prompts)} prompts across ${fmt(totals.projects)} projects.`,
      line2: 'You out-shipped most humans who own a keyboard this year.',
    },
  ];
}

// Cut-specific catalogs get first pick; the shared catalog backs them up.
function cutCatalog(report, cut) {
  const { tics, you, machine, alt, totals, deep } = report;
  if (cut === 'machine') {
    return [
      {
        cls: PAIRED, when: machine.linesWritten >= 1000 && totals.prompts >= 50, score: 1e6,
        line1: `${fmt(machine.linesWritten)} lines of code, from ${fmt(totals.prompts)} prompts.`,
        line2: 'History’s best words-to-software exchange rate.',
      },
      {
        cls: PAIRED, when: machine.tokensTotal >= 1_000_000 && deep.gitCommits >= 10, score: 1e5,
        line1: `${fmt(machine.tokensTotal)} tokens burned, ${fmt(deep.gitCommits)} commits landed.`,
        line2: 'Somewhere a GPU is asking for a raise.',
      },
      {
        cls: RAW, when: machine.bashCmds >= 100, score: machine.bashCmds,
        line1: `${fmt(machine.bashCmds)} shell commands executed on your behalf.`,
        line2: 'You have not touched a terminal in months. Allegedly.',
      },
    ];
  }
  if (cut === 'chaos') {
    return [
      {
        cls: PAIRED, when: alt.actually >= 100 && alt.wait >= 50, score: 1e6,
        line1: `"actually" × ${fmt(alt.actually)}. "wait" × ${fmt(alt.wait)}.`,
        line2: 'The two-word steering wheel of modern software.',
      },
      {
        cls: RAW, when: alt.shortOrders >= 50, score: alt.shortOrders,
        line1: `${fmt(alt.shortOrders)} prompts of three words or fewer.`,
        line2: 'A person of few words with infinite leverage.',
      },
    ];
  }
  if (cut === 'agent') {
    return [
      {
        cls: PAIRED, when: tics.shouldWork >= 10 && you.youWrong >= 5, score: 1e6,
        line1: `"That should work now" — ${fmt(tics.shouldWork)} times.`,
        line2: `${fmt(you.youWrong)} times it did not.`,
      },
      {
        cls: RAW, when: tics.perfect >= 20, score: tics.perfect,
        line1: `It said "Perfect!" ${fmt(tics.perfect)} times.`,
        line2: 'Perfection has never been this negotiable.',
      },
      {
        cls: RAW, when: tics.iSeeIssue >= 10, score: tics.iSeeIssue,
        line1: `"I see the issue" — ${fmt(tics.iSeeIssue)} declarations.`,
        line2: 'Seeing it and fixing it are different sports.',
      },
    ];
  }
  return [];
}

// signature(report, cut) -> { line1, line2 }. Deterministic: filter by
// eligibility, rank by (class, score, catalog order), take the top.
export function signature(report, cut = 'classic') {
  const candidates = [...cutCatalog(report, cut), ...punchlineCatalog(report)];
  let best = null;
  let bestKey = null;
  candidates.forEach((c, i) => {
    if (!c.when) return;
    const key = [c.cls, c.score, -i];
    if (!best || key[0] > bestKey[0] || (key[0] === bestKey[0] && (key[1] > bestKey[1] || (key[1] === bestKey[1] && key[2] > bestKey[2])))) {
      best = c;
      bestKey = key;
    }
  });
  return { line1: best.line1, line2: best.line2 };
}
