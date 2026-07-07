// seshy-wrapped CLI. The terminal output is a share surface too — people
// screenshot terminals — so the summary block is part of the product.
import { writeFileSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import os from 'node:os';
import { execFileSync } from 'node:child_process';
import { createInterface } from 'node:readline';
import { collect, stableStringify } from './pipeline.js';
import { KNOWN_AGENTS } from './discover.js';
import { CUTS, fmt, pickHeadline, signature, prettyModel } from './copy.js';
import { yearbook } from './yearbook.js';

// ANSI helpers — degrade to plain text when not a TTY.
const tty = process.stdout.isTTY;
const c = (code, s) => (tty ? `\x1b[${code}m${s}\x1b[0m` : s);
const bold = (s) => c('1', s);
const dim = (s) => c('2', s);
const pink = (s) => c('38;5;212', s);
const cyan = (s) => c('38;5;51', s);
const yellow = (s) => c('38;5;220', s);
const orange = (s) => c('38;5;214', s);
const violet = (s) => c('38;5;135', s);
const green = (s) => c('38;5;84', s);

const SCAN_LINES = [
  'counting your sins…',
  'tallying F-bombs…',
  'measuring your patience…',
  'auditing the apologies…',
  'weighing the tokens…',
  'reading the room (all of them)…',
];

function parseArgs(argv) {
  const o = {
    agent: 'all', model: '', since: 0, until: 0, window: 'all',
    cut: 'classic', theme: 'sunset', allCuts: false, handle: '',
    json: false, out: '', open: false, help: false, now: 0,
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    const next = () => argv[++i];
    if (a === '--agent' || a === '-a') o.agent = next();
    else if (a === '--model' || a === '-m') o.model = next();
    else if (a === '--since') { o.window = next(); o.since = Date.parse(o.window) || 0; }
    else if (a === '--period' || a === '-p') o.window = next();
    else if (a === '--cut' || a === '-c') o.cut = next();
    else if (a === '--theme' || a === '-t') o.theme = next();
    else if (a === '--all-cuts') o.allCuts = true;
    else if (a === '--handle' || a === '-h') o.handle = next();
    else if (a === '--json') o.json = true;
    else if (a === '--out' || a === '-o') o.out = next();
    else if (a === '--open') o.open = true;
    else if (a === '--now') o.now = Date.parse(next()) || 0; // tests: pin the clock
    else if (a === '--help') o.help = true;
    else o.unknown ||= a; // typo'd flags must not silently render a full card
  }
  return o;
}

const HELP = `
${bold('seshy wrapped')} — your AI-coding life, on a card

Usage:
  npx seshy-wrapped [options]

Run bare in a terminal and it asks what to wrap: fable week (Jul 1–7,
Fable 5 only — the default), the last 30 days, or all time. Any scoping
flag below answers the question in advance.

Options:
  -a, --agent <name>    ${KNOWN_AGENTS.join(' | ')} | all   (default: all)
  -m, --model <substr>  only work answered by matching models — "fable",
                        "opus-4-8", "gpt-5.4"; the card retitles itself
      --since <date>    only sessions after YYYY-MM-DD
  -p, --period <win>    week | month | year | all            (default: all)
  -c, --cut <cut>       ${CUTS.join(' | ')}   (default: classic)
  -t, --theme <look>    sunset | terminal | starfield | receipt | billboard | crt
      --all-cuts        render every cut in the chosen theme
  -h, --handle <@you>   handle printed on the card (default: none)
      --json            emit stats.json to stdout only, no images
  -o, --out <dir>       output directory                     (default: ./seshy-wrapped)
      --open            open the card when done (macOS)
      --help            this help

100% local — your sessions never leave your machine.
`;

const PERIODS = { week: 7, month: 30, year: 365 };

// The moment: Fable 5's included-in-subscription window reopened July 1 2026.
const FABLE_SINCE = '2026-07-01';

// applyScope resolves the interactive scope choice onto the options object.
// Exported for tests; choice "1" (or Enter) = fable week, "2" = last 30
// days, "3" = all time.
export function applyScope(o, choice, now = Date.now()) {
  const c = String(choice).trim() || '1';
  if (c === '2') o.since = now - 30 * 86400000;
  else if (c !== '3') {
    o.model = 'fable';
    o.since = Date.parse(FABLE_SINCE);
  }
  return o;
}

// Bare interactive runs get one question instead of a flag. Any scoping
// flag (--model/--since/--period), --json, or a non-TTY stream skips it.
async function askScope(o, log) {
  log(`  what are we wrapping?\n\n`);
  log(`    1) ${bold('fable week')}     ${dim('Jul 1–7 · Fable 5 only   ← the moment')}\n`);
  log(`    2) ${bold('last 30 days')}   ${dim('every agent, every model')}\n`);
  log(`    3) ${bold('all time')}       ${dim('the whole archive')}\n\n`);
  const rl = createInterface({ input: process.stdin, output: process.stderr });
  const answer = await new Promise((res) => rl.question(`  pick ${dim('[1]')}: `, res));
  rl.close();
  log('\n');
  return applyScope(o, answer);
}

export async function main() {
  const o = parseArgs(process.argv.slice(2));
  if (o.help) {
    process.stdout.write(HELP + '\n');
    return;
  }
  const err = validate(o);
  if (err) {
    process.stderr.write(`\n  ${err}\n\n`);
    process.exit(1);
  }
  if (PERIODS[o.window]) {
    const now = o.now || Date.now();
    o.since = now - PERIODS[o.window] * 86400000;
  }

  const log = (m) => process.stderr.write(m);
  if (!o.json) {
    log('\n  ' + green('100% local') + dim(' — your sessions never leave your machine.') + '\n\n');
  }

  // One question on bare interactive runs; flags and pipes skip it.
  const unscoped = !o.model && !o.since && o.window === 'all';
  if (unscoped && !o.json && process.stdin.isTTY && process.stderr.isTTY) {
    await askScope(o, log);
  }

  let scanIdx = 0;
  let lastPct = -1;
  const report = await collect({
    agent: o.agent,
    since: o.since,
    model: o.model,
    window: o.window,
    // Demo/test hook: point discovery at a synthetic home tree.
    home: process.env.SESHY_WRAPPED_HOME || undefined,
    onProgress: (done, total) => {
      if (!process.stderr.isTTY || o.json) return;
      const pctN = Math.floor((done / total) * 100);
      if (pctN !== lastPct) {
        lastPct = pctN;
        const phrase = SCAN_LINES[Math.floor(pctN / (100 / SCAN_LINES.length)) % SCAN_LINES.length];
        log(`\r  reading ${fmt(total)} sessions… ${pctN}%  ${dim(phrase)}          `);
      }
    },
  });
  if (!o.json) log('\r' + ' '.repeat(100) + '\r');

  // Edge: nothing found at all.
  if (report.totals.sessions === 0) {
    if (o.model) {
      log(`\n  No sessions matched --model "${o.model}"${o.agent !== 'all' ? ` under --agent ${o.agent}` : ''}.\n` +
          `  Models are recorded per-reply, so try ${cyan('--agent all')} — or check the spelling.\n` +
          `  (If it's Fable you're after: the logs only know it as claude-fable-5.)\n\n`);
      return;
    }
    log(`\n  No coding-agent sessions found on this machine.\n` +
        `  seshy-wrapped reads local logs from: ${KNOWN_AGENTS.join(', ')}.\n` +
        `  Run your favorite agent for a while, then come back. We'll be here.\n\n`);
    return;
  }

  if (o.json) {
    process.stdout.write(stableStringify(report));
    return;
  }

  // Terminal summary — screenshot-worthy, printed BEFORE the images.
  printSummary(report, o);

  // Render.
  const outDir = path.resolve((o.out || './seshy-wrapped').replace(/^~/, os.homedir()));
  mkdirSync(outDir, { recursive: true });
  const statsPath = path.join(outDir, 'stats.json');
  writeFileSync(statsPath, stableStringify(report));

  const { buildSVG, renderPNG } = await import('./card.js');
  const cuts = o.allCuts ? CUTS : [o.cut];
  const written = [];
  for (const cut of cuts) {
    const svg = buildSVG(report, { cut, theme: o.theme, handle: o.handle });
    const name = ['wrapped', cut, o.theme === 'sunset' ? '' : o.theme].filter(Boolean).join('-') + '.png';
    const p = path.join(outDir, name);
    writeFileSync(p, renderPNG(svg));
    written.push(p);
  }

  log('\n');
  for (const p of written) log(`  ${green('→')} ${p}\n`);
  log(`  ${dim('→')} ${dim(statsPath)}\n`);

  // Ready-to-paste post line.
  const post = postLine(report);
  log('\n  ' + dim('ready to paste:') + '\n');
  log(`  ${yellow(post)}\n  ${cyan('npx seshy-wrapped')}\n\n`);
  // The cross-promo outro: the retention fact is a genuine PSA (this card is
  // built from logs Claude Code deletes after 30 days) and the pitch rides it.
  log(`  ${dim('heads up: Claude Code deletes session logs after 30 days — this card is built from the survivors.')}\n`);
  log(`  ${dim('keep a year, resume any session: ')}${bold('seshy')}${dim(' · ')}${cyan('seshy retention protect')}${dim(' · ')}${cyan('seshy.dev')}\n\n`);

  // Offer to open on macOS.
  if (o.open) {
    openCard(written[0]);
  } else if (process.platform === 'darwin' && process.stdin.isTTY && process.stderr.isTTY) {
    const rl = createInterface({ input: process.stdin, output: process.stderr });
    const answer = await new Promise((res) => rl.question(`  open the card? ${dim('(y/N)')} `, res));
    rl.close();
    if (/^y/i.test(answer.trim())) openCard(written[0]);
    log('\n');
  }
}

function validate(o) {
  if (o.unknown) return `Unknown option "${o.unknown}". Run with --help to see every flag.`;
  if (!['all', ...KNOWN_AGENTS].includes(o.agent)) return `Unknown agent "${o.agent}". Pick one of: ${KNOWN_AGENTS.join(', ')}, all.`;
  if (!CUTS.includes(o.cut)) return `Unknown cut "${o.cut}". Pick one of: ${CUTS.join(', ')}.`;
  const themes = ['sunset', 'terminal', 'starfield', 'receipt', 'billboard', 'crt'];
  if (!themes.includes(o.theme)) return `Unknown theme "${o.theme}". Pick one of: ${themes.join(', ')}.`;
  if (!['all', 'week', 'month', 'year'].includes(o.window) && !o.since) return `Could not parse --since/--period "${o.window}". Use YYYY-MM-DD or week|month|year|all.`;
  return '';
}

function openCard(p) {
  try {
    execFileSync('open', [p], { stdio: 'ignore' });
  } catch {
    /* not fatal */
  }
}

// The screenshotable block. Box-drawing + a neon-ish gradient title line.
function printSummary(report, o) {
  const log = (m) => process.stderr.write(m);
  const head = pickHeadline(report, o.cut);
  const sig = signature(report, o.cut);
  const t = report.totals;
  const label = report.meta.model ? prettyModel(report.deep.topModel.id || report.meta.model) : null;

  const title = `SESHY WRAPPED${label ? ` · ${label.toUpperCase()}` : ''}${report.meta.rookie ? ' · ROOKIE CARD' : ''}`;
  const rows = [
    [head.value, head.label],
    [fmt(t.sessions), `sessions across ${fmt(t.projects)} projects`],
    [`${Math.round(report.time.activeHours)}h`, 'agent hours on the clock'],
    // Headless runs sit outside the interactive session count above, so the
    // line names its own universe — screenshot auditors will do the math.
    [fmt(report.automation.headless), `headless runs alongside (${report.automation.headlessShare}% of all activity)`],
    [fmt(report.tics.youreRight), `"you're right"s pried out of it`],
    [fmt(report.you.fbombs), 'F-bombs dropped'],
    [fmt(report.you.nightOwl), 'prompts after midnight'],
    [report.grade.letter, `delegation grade — ${report.grade.why}`],
  ];

  const wV = Math.max(...rows.map((r) => r[0].length));
  const inner = 64;
  log(`  ${pink('╭' + '─'.repeat(inner) + '╮')}\n`);
  log(`  ${pink('│')}  ${bold(cyan(title))}${' '.repeat(Math.max(1, inner - 2 - title.length))}${pink('│')}\n`);
  log(`  ${pink('│')}  ${dim(head.sub)}${' '.repeat(Math.max(1, inner - 2 - head.sub.length))}${pink('│')}\n`);
  log(`  ${pink('├' + '─'.repeat(inner) + '┤')}\n`);
  for (const [v, l] of rows) {
    const vp = v.padStart(wV);
    const line = `${vp}  ${l}`;
    const colored = `${yellow(vp)}  ${l}`;
    log(`  ${pink('│')}  ${colored}${' '.repeat(Math.max(1, inner - 2 - line.length))}${pink('│')}\n`);
  }
  log(`  ${pink('├' + '─'.repeat(inner) + '┤')}\n`);
  const yb = yearbook(report);
  for (const [text, paint] of [
    [`YEARBOOK · ${yb.title}`, (s) => bold(cyan(s))],
    [yb.receipt, dim],
    [sig.line1, orange],
    [sig.line2, dim],
  ]) {
    const s = text.slice(0, inner - 4);
    log(`  ${pink('│')}  ${paint(s)}${' '.repeat(Math.max(1, inner - 2 - s.length))}${pink('│')}\n`);
  }
  if (report.meta.skippedLines) {
    const note = `${fmt(report.meta.skippedLines)} corrupt log lines skipped`;
    log(`  ${pink('│')}  ${dim(note)}${' '.repeat(Math.max(1, inner - 2 - note.length))}${pink('│')}\n`);
  }
  log(`  ${pink('╰' + '─'.repeat(inner) + '╯')}\n`);
}

// One suggested post line: strongest stat + the command. Nothing else.
function postLine(report) {
  const { machine, you, totals, tics } = report;
  const label = report.meta.model ? prettyModel(report.deep.topModel.id || report.meta.model) : 'my agents';
  if (machine.linesWritten >= 1000 && tics.youreRight >= 5) return `${label} wrote ${fmt(machine.linesWritten)} lines of code for me and said "you're right" ${fmt(tics.youreRight)} times along the way.`;
  if (machine.linesWritten >= 1000) return `${label} wrote ${fmt(machine.linesWritten)} lines of code for me this ${report.meta.window === 'all' ? 'run' : report.meta.window}.`;
  if (you.fbombs >= 10) return `${fmt(totals.prompts)} prompts, ${fmt(you.fbombs)} F-bombs. ${label} never swore back.`;
  return `${fmt(totals.prompts)} prompts across ${fmt(totals.projects)} projects with ${label}.`;
}
