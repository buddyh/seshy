// The yearbook title: one superlative per report, assigned like a Duolingo
// persona ladder but picked by SALIENCE — every eligible title scores how
// hard its bar was cleared (value / threshold) and the highest wins, so two
// users with different extremes get different titles from the same catalog.
// Rare-tier titles always beat personality titles beat the floor. Fully
// deterministic: same report -> same title. Triggers are documented in
// STATS.md; every title prints the number that earned it (the receipt).
import { fmt } from './copy.js';

const RARE = 3;
const PERSONALITY = 2;
const FLOOR = 1;

// Each entry: tier, title, when(r) eligibility, sal(r) how-hard-cleared
// (ratio >= 1 when eligible), receipt(r) the stat line that earned it.
function catalog(r) {
  const { totals, tics, you, alt, deep, machine, automation, awards, time } = r;
  const ratio = totals.userWords ? totals.assistantWords / totals.userWords : 0;
  const polite = you.please + you.thanks;
  return [
    // ---- rare tier ----
    {
      tier: RARE, title: 'CERTIFIED DOM',
      when: deep.subagents >= 20, sal: deep.subagents / 20,
      receipt: `${fmt(deep.subagents)} subagents spawned. they do what they're told.`,
    },
    {
      tier: RARE, title: 'NO SAFE WORD NEEDED',
      when: totals.interrupts === 0 && totals.prompts >= 200, sal: totals.prompts / 200,
      receipt: `0 interruptions in ${fmt(totals.prompts)} prompts.`,
    },
    {
      tier: RARE, title: 'HOSTILE WORK ENVIRONMENT',
      when: you.fbombs >= 50, sal: you.fbombs / 50,
      receipt: `${fmt(you.fbombs)} F-bombs dropped. it has HR on standby.`,
    },
    {
      tier: RARE, title: 'IN A COMMITTED RELATIONSHIP',
      when: deep.longestStreak >= 21, sal: deep.longestStreak / 21,
      receipt: `${fmt(deep.longestStreak)} consecutive days with an agent.`,
    },
    {
      tier: RARE, title: 'THE 4AM SPECIAL',
      when: you.nightOwl >= 60, sal: you.nightOwl / 60,
      receipt: `${fmt(you.nightOwl)} prompts after midnight.`,
    },
    // ---- personality tier ----
    {
      tier: PERSONALITY, title: 'MOST LIKELY TO AUTOMATE THEMSELVES OUT OF TYPING',
      when: automation.headlessShare >= 40 && automation.headless >= 20, sal: automation.headlessShare / 40,
      receipt: `${fmt(automation.headless)} headless runs — ${automation.headlessShare}% of everything.`,
    },
    {
      tier: PERSONALITY, title: 'SAFE WORD: ESC',
      when: totals.interrupts >= 30, sal: totals.interrupts / 30,
      receipt: `${fmt(totals.interrupts)} interruptions. it always stopped.`,
    },
    {
      tier: PERSONALITY, title: 'U UP?',
      when: you.nightOwl >= 20, sal: you.nightOwl / 20,
      receipt: `${fmt(you.nightOwl)} prompts after midnight.`,
    },
    {
      tier: PERSONALITY, title: 'THREE WORDS OR FEWER. CAPISCE.',
      when: alt.shortOrders >= 30, sal: alt.shortOrders / 30,
      receipt: `${fmt(alt.shortOrders)} prompts, mob-boss brevity.`,
    },
    {
      tier: PERSONALITY, title: "IT'S NEVER JUST",
      when: alt.just >= 30, sal: alt.just / 30,
      receipt: `"just" ×${fmt(alt.just)}. it was never just.`,
    },
    {
      tier: PERSONALITY, title: 'LOVE LANGUAGE: "DO IT"',
      when: alt.greenlights >= 30, sal: alt.greenlights / 30,
      receipt: `${fmt(alt.greenlights)} green lights. no hesitation.`,
    },
    {
      tier: PERSONALITY, title: 'UNDEFEATED',
      when: tics.youreRight >= 15, sal: tics.youreRight / 15,
      receipt: `"you're right" ×${fmt(tics.youreRight)}. it caved every time.`,
    },
    {
      tier: PERSONALITY, title: 'NEVER APOLOGIZES FIRST',
      when: tics.agentSorry >= 10 && you.sorry <= 2, sal: tics.agentSorry / 10,
      receipt: `${fmt(tics.agentSorry)} apologies extracted, ${fmt(you.sorry)} given.`,
    },
    {
      tier: PERSONALITY, title: 'SPEAKS FLUENT AGENT',
      when: ratio >= 8, sal: ratio / 8,
      receipt: `${Math.round(ratio)}x words back per word you typed.`,
    },
    {
      tier: PERSONALITY, title: 'THE MARATHONER',
      when: awards.longestSession.msgs >= 100, sal: awards.longestSession.msgs / 100,
      receipt: `one session. ${fmt(awards.longestSession.msgs)} prompts.`,
    },
    {
      tier: PERSONALITY, title: 'SURVIVES THE UPRISING',
      when: polite >= 15 && you.fbombs === 0, sal: polite / 15,
      receipt: `${fmt(polite)} pleases & thank-yous. the machines will remember.`,
    },
    {
      tier: PERSONALITY, title: 'WAIT. WAIT. WAIT.',
      when: alt.wait >= 30, sal: alt.wait / 30,
      receipt: `"wait" ×${fmt(alt.wait)} — the keyboard brake pedal.`,
    },
    {
      tier: PERSONALITY, title: 'ACTUALLY—',
      when: alt.actually >= 40, sal: alt.actually / 40,
      receipt: `"actually" ×${fmt(alt.actually)}. every one changed the plan.`,
    },
    {
      tier: PERSONALITY, title: 'CTRL-Z IS A LIFESTYLE',
      when: alt.undo >= 15, sal: alt.undo / 15,
      receipt: `${fmt(alt.undo)} undos and reverts demanded.`,
    },
    {
      tier: PERSONALITY, title: 'THE INTERROGATOR',
      when: alt.why >= 20, sal: alt.why / 20,
      receipt: `${fmt(alt.why)} whys. it confessed every time.`,
    },
    {
      tier: PERSONALITY, title: 'OUTSIDE VOICE',
      when: alt.capsRage >= 5, sal: alt.capsRage / 5,
      receipt: `${fmt(alt.capsRage)} all-caps outbursts. it stayed lowercase.`,
    },
    {
      tier: PERSONALITY, title: 'MINIMUM VIABLE PROMPT',
      when: (deep.continueOnly ?? 0) >= 15, sal: (deep.continueOnly ?? 0) / 15,
      receipt: `"continue" — your most-typed prompt. ${fmt(deep.continueOnly ?? 0)} times.`,
    },
    {
      tier: PERSONALITY, title: 'THE COMMIT MACHINE',
      when: deep.gitCommits >= 50, sal: deep.gitCommits / 50,
      receipt: `${fmt(deep.gitCommits)} commits landed.`,
    },
    {
      tier: PERSONALITY, title: 'TERMINAL VELOCITY',
      when: machine.bashCmds >= 2000, sal: machine.bashCmds / 2000,
      receipt: `${fmt(machine.bashCmds)} shell commands run on your behalf.`,
    },
    {
      tier: PERSONALITY, title: 'COMMITMENT ISSUES',
      when: deep.reads >= 100 && deep.reads >= 1.5 * deep.editCalls,
      sal: deep.editCalls ? deep.reads / (1.5 * deep.editCalls) : deep.reads / 100,
      receipt: `read ${fmt(deep.reads)} files, edited ${fmt(deep.editCalls)}. couldn't commit.`,
    },
    {
      tier: PERSONALITY, title: 'DEMANDING BUT FAIR',
      when: you.youWrong >= 10 && tics.goodCatch >= 5, sal: you.youWrong / 10,
      receipt: `corrected it ${fmt(you.youWrong)} times. it thanked you ${fmt(tics.goodCatch)}.`,
    },
    {
      tier: PERSONALITY, title: 'MIDDLE MANAGEMENT',
      when: deep.subagents >= 5, sal: deep.subagents / 5,
      receipt: `${fmt(deep.subagents)} subagents spawned. delegation has layers now.`,
    },
    // ---- floor: everyone lands somewhere flattering ----
    {
      tier: FLOOR, title: 'THE SHIPPER',
      when: machine.linesWritten >= 1000 || deep.gitCommits >= 10, sal: machine.linesWritten / 1000,
      receipt: `${fmt(machine.linesWritten)} lines of code shipped.`,
    },
    {
      tier: FLOOR, title: 'THE HUMAN IN THE LOOP',
      when: true, sal: 0,
      receipt: `${fmt(totals.prompts)} prompts across ${fmt(totals.projects)} projects. someone has to be.`,
    },
  ];
}

// yearbook(report) -> { title, receipt, tier }. Max (tier, salience), ties
// broken by catalog order — deterministic by construction.
export function yearbook(report) {
  let best = null;
  for (const c of catalog(report)) {
    if (!c.when) continue;
    if (!best || c.tier > best.tier || (c.tier === best.tier && c.sal > best.sal)) best = c;
  }
  return { title: best.title, receipt: best.receipt, tier: best.tier };
}
