// The stat engine: accumulate honest, funny, brag-worthy metrics across a
// whole corpus of normalized session events. Fully deterministic: pure regex
// counting and arithmetic — no model calls, no wall-clock reads. Every stat's
// exact definition is documented in STATS.md; keep the two in sync.

// Count non-overlapping matches of a global regex in a string.
function count(re, s) {
  let n = 0;
  re.lastIndex = 0;
  while (re.exec(s) !== null) {
    n++;
    if (!re.global) break;
  }
  return n;
}

const words = (s) => (s.trim() ? s.trim().split(/\s+/).length : 0);
const pct = (num, den) => (den ? Math.round((num / den) * 1000) / 10 : 0);
const round1 = (x) => Math.round(x * 10) / 10;

// Estimate active wall-clock time in one session from its event timestamps.
// Sums gaps between consecutive events, capping each gap at capMin minutes so
// an overnight break counts as one short gap, not eight idle hours.
export function sessionActiveMs(events, capMin = 5) {
  const stamps = events.map((e) => e.ts).filter(Boolean).sort((a, b) => a - b);
  if (stamps.length < 2) return 0;
  const cap = capMin * 60000;
  let ms = 0;
  for (let i = 1; i < stamps.length; i++) ms += Math.min(stamps[i] - stamps[i - 1], cap);
  return ms;
}

// --- Phrase catalog. Each entry mines one role's text. See STATS.md. -------
// Assistant capitulations & tics
const RE_ABSOLUTELY_RIGHT = /\b(?:you(?:'|’)?re|you are)\s+absolutely\s+right\b/gi;
const RE_YOURE_RIGHT = /\b(?:you(?:'|’)?re|you are)\s+(?:absolutely\s+|totally\s+|completely\s+|so\s+|100%\s+)?right\b/gi;
const RE_GOOD_CATCH = /\bgood\s+catch\b/gi;
const RE_AGENT_SORRY = /\b(?:i\s+apologi[sz]e|my\s+apologies|i(?:'|’)?m\s+sorry|i\s+am\s+sorry|apologies\b|sorry\s+(?:about|for|,))/gi;
const RE_YOU_RIGHT_I_WRONG = /\bi\s+(?:was\s+wrong|made\s+a\s+mistake|got\s+that\s+wrong|missed)\b/gi;
const RE_PERFECT = /\bperfect[.!]/gi;
const RE_LETME = /\blet\s+me\b/gi;
const RE_I_SEE_ISSUE = /\b(?:i\s+(?:can\s+)?see\s+the\s+(?:issue|problem|bug)|found\s+the\s+(?:issue|problem|bug)|the\s+(?:issue|problem)\s+is)\b/gi;
const RE_SHOULD_WORK = /\bshould\s+(?:now\s+)?work\b|\bshould\s+be\s+(?:fixed|working|good\s+to\s+go)\b/gi;

// Human side
const RE_FBOMB = /f+u+c+k(?:ing|ed|er|s|in['’]?)?\b|\bwtf\b|\bfml\b|\bstfu\b/gi;
const RE_PROFANITY = /\b(?:sh[i1]t+|damn+|crap|hell|arse|ass(?:hole|hat)?|bullsh[i1]t|goddamn|bloody|piss(?:ed)?|bastard|b[i1]tch)\b/gi;
const RE_USER_SORRY = /\b(?:sorry|my\s+bad|apologi[sz]e|my\s+apologies|oops|whoops)\b/gi;
const RE_PLEASE = /\bplease\b|\bpls\b|\bplz\b/gi;
const RE_THANKS = /\b(?:thanks?|thank\s+you|thx|ty|appreciate\s+(?:it|you))\b/gi;
const RE_YOU_WRONG = /\b(?:that(?:'|’)?s\s+(?:wrong|not\s+right|incorrect)|you(?:'|’)?re\s+wrong|that\s+didn(?:'|’)?t\s+work|that\s+doesn(?:'|’)?t\s+work|still\s+(?:broken|not\s+working|failing|wrong)|nope\b|that\s+broke|you\s+broke)/gi;
const RE_ENTHUSIASM = /\b(?:ship\s+it|let(?:'|’)?s\s+go|lfg|hell\s+yeah|nice+\b|beautiful\b|perfect\b|amazing\b|love\s+it|gorgeous)\b/gi;
const RE_ONE_MORE = /\b(?:one\s+more\s+(?:thing|time)|just\s+one\s+more|actually\s+can\s+you|wait\b|hold\s+on|nvm|never\s?mind)\b/gi;
const RE_WAIT = /\bwait\b/gi;
const RE_ACTUALLY = /\bactually\b/gi;
const RE_JUST = /\bjust\b/gi;
const RE_WHY = /\bwhy\b/gi;
// "what??" — must follow a word character, else JS's `??` operator in pasted
// code counts as confusion (it is, but not yours).
const RE_DOUBLE_Q = /[a-z0-9)\]]\?{2,}/gi;
// Caps-lock shouting: a curated list of words that only mean one thing in
// all-caps. Generic [A-Z]{4,} counts pasted SQL and env vars, not rage.
const RE_CAPS = /\b(?:STOP|WAIT|WHY|DON'?T|PLEASE|HELP|WRONG|BROKEN|NEVER|SERIOUSLY|OMG|FFS|WTF|NO{2,}|ARGH+|FIX)\b/g;
const RE_UNDO = /\b(?:undo|revert|roll\s?back|put\s+it\s+back|go\s+back\s+to)\b/gi;
const RE_GREENLIGHT = /\b(?:do\s+it|go\s+ahead|proceed|continue|go\s+for\s+it|sounds\s+good|lgtm|approved?|yes\s+please)\b/gi;
const RE_CONTINUE_ONLY = /^(?:continue|go|k|ok|okay|yes|y|proceed|keep going|go ahead|do it)[.!]?$/i;

export function newAccumulator() {
  return {
    sessions: 0,
    sessionsWithHuman: 0,
    projects: new Set(),
    humanMsgs: 0,
    assistantMsgs: 0,
    interrupts: 0,
    userWords: 0,
    assistantWords: 0,
    firstTs: 0,
    lastTs: 0,
    byHour: new Array(24).fill(0),
    byDay: new Map(),
    byProject: new Map(),
    nightOwl: 0,
    absolutelyRight: 0,
    youreRight: 0,
    goodCatch: 0,
    agentSorry: 0,
    agentAdmitWrong: 0,
    perfect: 0,
    letMe: 0,
    iSeeIssue: 0,
    shouldWork: 0,
    fbombs: 0,
    profanity: 0,
    userSorry: 0,
    please: 0,
    thanks: 0,
    youWrong: 0,
    enthusiasm: 0,
    oneMore: 0,
    wait: 0,
    actually: 0,
    just: 0,
    why: 0,
    doubleQ: 0,
    capsRage: 0,
    undo: 0,
    greenlights: 0,
    shortOrders: 0,
    continueOnly: 0,
    byAgent: new Map(),
    byModel: new Map(),
    commands: new Map(),
    skills: new Map(),
    toolNames: new Map(),
    editsByFile: new Map(),
    subagents: 0,
    planMode: 0,
    webCalls: 0,
    reads: 0,
    editCalls: 0,
    toolResults: 0,
    toolErrors: 0,
    scary: 0,
    gitCommits: 0,
    weekendPrompts: 0,
    thinkingWords: 0,
    tokensIn: 0,
    tokensOut: 0,
    cacheRead: 0,
    toolCalls: 0,
    bashCmds: 0,
    linesWritten: 0,
    filesTouched: new Set(),
    skippedLines: 0,
    longestSession: { msgs: 0, cwd: '' },
  };
}

// feedSession runs one session's events into the accumulator. Sessions with
// no genuinely typed human prompt (pure automation) are skipped so every stat
// reflects a real conversation the user had.
export function feedSession(acc, events) {
  const hasHuman = events.some((e) => e.role === 'human' && e.kind === 'prompt');
  if (!hasHuman) return 0;
  acc.sessions++;
  let humanHere = 0;
  let cwd = '';
  const bump = (m, k, n = 1) => m.set(k, (m.get(k) || 0) + n);
  for (const ev of events) {
    if (ev.cwd) cwd = ev.cwd;
    if (ev.ts) {
      if (!acc.firstTs || ev.ts < acc.firstTs) acc.firstTs = ev.ts;
      if (ev.ts > acc.lastTs) acc.lastTs = ev.ts;
    }
    if (ev.role === 'system') {
      if (ev.kind === 'command' && ev.command) bump(acc.commands, ev.command);
      if (ev.kind === 'tool_result' && ev.results) {
        acc.toolResults += ev.results.total;
        acc.toolErrors += ev.results.errors;
      }
      continue;
    }
    if (ev.role === 'human') {
      if (ev.kind === 'interrupt') {
        acc.interrupts++;
        continue;
      }
      humanHere++;
      acc.humanMsgs++;
      acc.userWords += words(ev.text);
      const t = ev.text;
      acc.fbombs += count(RE_FBOMB, t);
      acc.profanity += count(RE_PROFANITY, t);
      acc.userSorry += count(RE_USER_SORRY, t);
      acc.please += count(RE_PLEASE, t);
      acc.thanks += count(RE_THANKS, t);
      acc.youWrong += count(RE_YOU_WRONG, t);
      acc.enthusiasm += count(RE_ENTHUSIASM, t);
      acc.oneMore += count(RE_ONE_MORE, t);
      acc.wait += count(RE_WAIT, t);
      acc.actually += count(RE_ACTUALLY, t);
      acc.just += count(RE_JUST, t);
      acc.why += count(RE_WHY, t);
      acc.doubleQ += count(RE_DOUBLE_Q, t);
      acc.capsRage += count(RE_CAPS, t);
      acc.undo += count(RE_UNDO, t);
      acc.greenlights += count(RE_GREENLIGHT, t);
      if (words(t) <= 3) acc.shortOrders++;
      if (RE_CONTINUE_ONLY.test(t.trim())) acc.continueOnly++;
      if (ev.agent) bump(acc.byAgent, ev.agent);
      if (ev.ts) {
        const d = new Date(ev.ts);
        const h = d.getHours();
        acc.byHour[h]++;
        if (h < 5) acc.nightOwl++;
        const dow = d.getDay();
        if (dow === 0 || dow === 6) acc.weekendPrompts++;
        const day = d.toISOString().slice(0, 10);
        bump(acc.byDay, day);
      }
    } else if (ev.role === 'assistant') {
      if (ev.usage) {
        acc.tokensIn += ev.usage.in;
        acc.tokensOut += ev.usage.out;
        acc.cacheRead += ev.usage.cacheRead;
      }
      if (ev.tools) {
        acc.toolCalls += ev.tools.calls;
        acc.bashCmds += ev.tools.bash;
        acc.linesWritten += ev.tools.lines;
        acc.scary += ev.tools.scary || 0;
        acc.gitCommits += ev.tools.commits || 0;
        for (const f of ev.tools.files) {
          acc.filesTouched.add(f);
          bump(acc.editsByFile, f);
        }
        for (const s of ev.tools.skills || []) bump(acc.skills, s);
        for (const n of ev.tools.names || []) {
          bump(acc.toolNames, n);
          if (n === 'Task' || n === 'Agent') acc.subagents++;
          else if (n === 'EnterPlanMode' || n === 'ExitPlanMode' || n === 'exit_plan_mode') acc.planMode++;
          else if (n === 'WebSearch' || n === 'WebFetch') acc.webCalls++;
          else if (n === 'Read') acc.reads++;
          else if (n === 'Edit' || n === 'Write' || n === 'MultiEdit' || n === 'NotebookEdit') acc.editCalls++;
        }
      }
      if (ev.thinking) acc.thinkingWords += words(ev.thinking);
      if (!ev.text) continue; // tool-only / usage-only event, no prose
      if (ev.model) bump(acc.byModel, ev.model);
      acc.assistantMsgs++;
      acc.assistantWords += words(ev.text);
      const t = ev.text;
      acc.absolutelyRight += count(RE_ABSOLUTELY_RIGHT, t);
      acc.youreRight += count(RE_YOURE_RIGHT, t);
      acc.goodCatch += count(RE_GOOD_CATCH, t);
      acc.agentSorry += count(RE_AGENT_SORRY, t);
      acc.agentAdmitWrong += count(RE_YOU_RIGHT_I_WRONG, t);
      acc.perfect += count(RE_PERFECT, t);
      acc.letMe += count(RE_LETME, t);
      acc.iSeeIssue += count(RE_I_SEE_ISSUE, t);
      acc.shouldWork += count(RE_SHOULD_WORK, t);
    }
  }
  if (cwd) {
    acc.projects.add(cwd);
    if (humanHere) acc.byProject.set(cwd, (acc.byProject.get(cwd) || 0) + humanHere);
  }
  if (humanHere) acc.sessionsWithHuman++;
  // Deterministic tiebreak: strictly-greater keeps the earliest-fed session.
  if (humanHere > acc.longestSession.msgs) acc.longestSession = { msgs: humanHere, cwd };
  return humanHere;
}

// Longest run of consecutive days in a 'YYYY-MM-DD' -> count map.
function longestStreak(byDay) {
  const days = [...byDay.keys()].sort();
  let best = 0;
  let cur = 0;
  let prev = 0;
  for (const d of days) {
    const t = Date.parse(d);
    cur = t - prev === 86400000 ? cur + 1 : 1;
    prev = t;
    if (cur > best) best = cur;
  }
  return best;
}

// Sort map entries by count desc, then key asc — a total, stable order.
const sortEntries = (m) => [...m.entries()].sort((a, b) => b[1] - a[1] || (a[0] < b[0] ? -1 : 1));
const topEntry = (m) => sortEntries(m)[0] || ['', 0];

// finalize turns the accumulator into the canonical report object. Everything
// any card renders comes from here — nothing recounts at render time.
export function finalize(acc, meta = {}) {
  const [topProjectPath, topProjectMsgs] = topEntry(acc.byProject);
  const [busiestDay, busiestDayMsgs] = topEntry(acc.byDay);
  const peakHour = acc.byHour.indexOf(Math.max(...acc.byHour));
  const days = acc.firstTs && acc.lastTs ? Math.max(1, Math.round((acc.lastTs - acc.firstTs) / 86400000)) : 0;
  const [topModel, topModelN] = topEntry(acc.byModel);
  const [topCommand, topCommandN] = topEntry(acc.commands);
  const [topSkill, topSkillN] = topEntry(acc.skills);
  const [topTool, topToolN] = topEntry(acc.toolNames);
  const [problemFile, problemFileN] = topEntry(acc.editsByFile);
  const commandTotal = [...acc.commands.values()].reduce((a, b) => a + b, 0);

  const perHarness = harnessSummary(meta.harness || {});
  const activeMs = perHarness.totals.activeMs || 0;
  const time = {
    activeMs,
    activeHours: round1(activeMs / 3600000),
    activeDays: acc.byDay.size,
    avgSessionMin: acc.sessionsWithHuman ? round1(activeMs / acc.sessionsWithHuman / 60000) : 0,
  };

  const report = {
    meta: {
      tool: 'seshy-wrapped',
      statsVersion: 1,
      agent: meta.agent || 'all',
      agents: meta.agents || [],
      model: meta.model || '',
      window: meta.window || 'all',
      filesScanned: meta.filesScanned || 0,
      skippedLines: acc.skippedLines,
      rookie: acc.sessionsWithHuman > 0 && acc.sessionsWithHuman < 10,
    },
    span: { firstTs: acc.firstTs, lastTs: acc.lastTs, days },
    harness: perHarness.byAgent,
    automation: perHarness.totals,
    time,
    totals: {
      sessions: acc.sessionsWithHuman,
      projects: acc.projects.size,
      prompts: acc.humanMsgs,
      replies: acc.assistantMsgs,
      userWords: acc.userWords,
      assistantWords: acc.assistantWords,
      interrupts: acc.interrupts,
    },
    tics: {
      absolutelyRight: acc.absolutelyRight,
      youreRight: acc.youreRight,
      goodCatch: acc.goodCatch,
      agentSorry: acc.agentSorry,
      agentAdmitWrong: acc.agentAdmitWrong,
      perfect: acc.perfect,
      letMe: acc.letMe,
      iSeeIssue: acc.iSeeIssue,
      shouldWork: acc.shouldWork,
    },
    machine: {
      tokensIn: acc.tokensIn,
      tokensOut: acc.tokensOut,
      tokensTotal: acc.tokensIn + acc.tokensOut,
      cacheRead: acc.cacheRead,
      toolCalls: acc.toolCalls,
      bashCmds: acc.bashCmds,
      linesWritten: acc.linesWritten,
      filesTouched: acc.filesTouched.size,
    },
    deep: {
      harness: Object.fromEntries(sortEntries(acc.byAgent)),
      topModel: { id: topModel, replies: topModelN },
      modelsTried: acc.byModel.size,
      models: Object.fromEntries(sortEntries(acc.byModel)),
      topCommand: { name: topCommand, count: topCommandN },
      commandsRun: commandTotal,
      commands: Object.fromEntries(sortEntries(acc.commands).slice(0, 15)),
      goals: (acc.commands.get('/goal') || 0) + (acc.commands.get('/loop') || 0),
      clears: acc.commands.get('/clear') || 0,
      topSkill: { name: topSkill, count: topSkillN },
      skillsInvoked: [...acc.skills.values()].reduce((a, b) => a + b, 0),
      distinctSkills: acc.skills.size,
      subagents: acc.subagents,
      planMode: acc.planMode,
      topTool: { name: topTool, count: topToolN },
      reads: acc.reads,
      editCalls: acc.editCalls,
      readWriteRatio: acc.editCalls ? round1(acc.reads / acc.editCalls) : 0,
      webCalls: acc.webCalls,
      toolResults: acc.toolResults,
      toolErrors: acc.toolErrors,
      errorRate: pct(acc.toolErrors, acc.toolResults),
      scary: acc.scary,
      gitCommits: acc.gitCommits,
      problemFile: { path: problemFile, edits: problemFileN },
      longestStreak: longestStreak(acc.byDay),
      weekendShare: pct(acc.weekendPrompts, acc.humanMsgs),
      continueOnly: acc.continueOnly,
      thinkingWords: acc.thinkingWords,
      thinkingShare: pct(acc.thinkingWords, acc.thinkingWords + acc.assistantWords),
    },
    alt: {
      wait: acc.wait,
      actually: acc.actually,
      just: acc.just,
      why: acc.why,
      doubleQ: acc.doubleQ,
      capsRage: acc.capsRage,
      undo: acc.undo,
      greenlights: acc.greenlights,
      shortOrders: acc.shortOrders,
    },
    you: {
      fbombs: acc.fbombs,
      profanity: acc.profanity,
      sorry: acc.userSorry,
      please: acc.please,
      thanks: acc.thanks,
      youWrong: acc.youWrong,
      enthusiasm: acc.enthusiasm,
      oneMore: acc.oneMore,
      nightOwl: acc.nightOwl,
    },
    awards: {
      longestSession: acc.longestSession,
      topProject: { path: topProjectPath, msgs: topProjectMsgs },
      busiestDay: { day: busiestDay, msgs: busiestDayMsgs },
      peakHour,
      byHour: acc.byHour,
    },
  };
  report.grade = computeGrade(acc, report);
  return report;
}

// Per-harness breakdown fed in by collect(): interactive vs headless session
// counts, interactive prompt volume, active time, and each agent's own range.
function harnessSummary(raw) {
  const byAgent = {};
  const totals = { sessions: 0, interactive: 0, headless: 0, prompts: 0, activeMs: 0, firstTs: 0, lastTs: 0 };
  for (const [agent, h] of Object.entries(raw).sort()) {
    const days = h.firstTs && h.lastTs ? Math.max(1, Math.round((h.lastTs - h.firstTs) / 86400000)) : 0;
    byAgent[agent] = {
      sessions: h.sessions,
      interactive: h.interactive,
      headless: h.headless,
      prompts: h.prompts,
      activeMs: h.activeMs || 0,
      activeHours: round1((h.activeMs || 0) / 3600000),
      firstTs: h.firstTs,
      lastTs: h.lastTs,
      days,
      headlessShare: pct(h.headless, h.sessions),
    };
    totals.sessions += h.sessions;
    totals.interactive += h.interactive;
    totals.headless += h.headless;
    totals.prompts += h.prompts;
    totals.activeMs += h.activeMs || 0;
    if (h.firstTs) totals.firstTs = totals.firstTs ? Math.min(totals.firstTs, h.firstTs) : h.firstTs;
    if (h.lastTs) totals.lastTs = Math.max(totals.lastTs, h.lastTs);
  }
  totals.headlessShare = pct(totals.headless, totals.sessions);
  totals.activeHours = round1(totals.activeMs / 3600000);
  const sorted = Object.fromEntries(Object.entries(byAgent).sort((a, b) => b[1].sessions - a[1].sessions || (a[0] < b[0] ? -1 : 1)));
  return { byAgent: sorted, totals };
}

// The delegation grade. Deterministic and window-fair: volume is measured as
// session DENSITY (sessions per active day), not absolute count, so a
// one-week or one-model card is not structurally capped. Rubric in STATS.md.
export function computeGrade(acc, report) {
  const prompts = acc.humanMsgs || 1;
  const avgLen = acc.userWords / prompts;
  const interruptRate = acc.interrupts / prompts;
  const politeness = (acc.please + acc.thanks) / prompts;
  const wrongRate = acc.youWrong / prompts;
  const density = acc.sessionsWithHuman / Math.max(1, acc.byDay.size); // sessions per active day
  const autonomy = (report.automation.headlessShare || 0) / 100; // share of sessions that ran headless

  const parts = {
    clarity: Math.min(18, round1(avgLen * 0.9)), // long prompts = clear briefs
    cadence: Math.min(12, round1(density * 4)), // daily-driver density
    leverage: Math.min(12, round1(autonomy * 24)), // automation share
    manners: Math.min(8, round1(politeness * 30)),
    temper: -Math.min(25, round1(interruptRate * 120)), // esc-esc rage
    friction: -Math.min(15, round1(wrongRate * 60)), // fighting the model
  };
  let score = 50;
  for (const v of Object.values(parts)) score += v;
  score = Math.max(0, Math.min(100, Math.round(score)));

  const letter =
    score >= 93 ? 'A' : score >= 87 ? 'A-' : score >= 83 ? 'B+' : score >= 78 ? 'B' :
    score >= 72 ? 'B-' : score >= 67 ? 'C+' : score >= 60 ? 'C' : score >= 52 ? 'C-' :
    score >= 45 ? 'D' : 'F';

  // One-line "why": strongest bonus + strongest penalty, fixed phrasing.
  const bonusPhrase = { clarity: 'clear briefs', cadence: 'daily driver', leverage: 'long leashes', manners: 'good manners' };
  const penaltyPhrase = { temper: 'short temper', friction: 'fights the model' };
  const bonuses = ['clarity', 'cadence', 'leverage', 'manners'].sort((a, b) => parts[b] - parts[a] || (a < b ? -1 : 1));
  const penalties = ['temper', 'friction'].sort((a, b) => parts[a] - parts[b] || (a < b ? -1 : 1));
  const worst = parts[penalties[0]] < -2 ? penaltyPhrase[penalties[0]] : 'no notes';
  const why = `${bonusPhrase[bonuses[0]]}, ${worst}`;

  return { score, letter, why, parts };
}
