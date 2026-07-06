# MARKETING.md — the July 7 playbook

Execution checklist. Everything below is copy-paste ready; fill `{...}` slots
with real numbers on the day.

## The moment

Fable 5 returned July 1; its included-in-subscription window closes **July 7**,
after which it moves to usage credits. Dev X will spend July 7 writing "end of
Fable week" posts from memory. We give that ritual an artifact: one command,
a card with *their* numbers on it, posted in under 90 seconds. The sharer gets
the status; we're the watermark riding along.

## Pre-flight (do these first, in order)

- [ ] `npm login` and **publish seshy-wrapped** (name reservation is the whole
      ballgame — a squatter kills the launch). Consider reserving `seshy` too.
- [ ] CI green on `feat/wrapped`: 14 tests + double-run byte diff.
- [ ] Dogfood on real data: `npx seshy-wrapped --model fable` and sanity-read
      every number on the card against `stats.json`.
- [ ] Raise `cleanupPeriodDays` NOW so the July 7 render still has the full
      Fable window (Claude deletes session logs after 30 days by default).
      One-liner: `seshy retention protect` — and the caveat doubles as content:
      the retention feature is the companion post ("your wrapped needs a year").

## Timeline

**July 4–5** — finish, dogfood on real data, record the demo video:
`packages/wrapped/demo/demo.sh` renders from synthetic fixtures (safe to
screen-record). 30 seconds: terminal scan → summary box → card reveal.

**July 6, ~8pm ET** — soft tease. Screenshot of the TERMINAL OUTPUT only (the
ANSI summary box), no card. Post:

> tomorrow you find out what your agent really thinks of you

**July 7, 9–10am ET** — hero post. Your real `FABLE 5 · MACHINE CUT` card.
One image, one stat, one command. Nothing else:

Then immediately:
- **Reply 1** (self-reply): the RECEIPT theme of the same data. "itemized, in
  case anyone's expensing their Fable week"
- **Reply 2** (self-reply): repo link + "works for Claude Code, Codex, Gemini,
  OpenCode — 100% local, your sessions never leave your machine. `npx seshy-wrapped`"

**July 7, ~7pm ET** — quote-tweet your own hero post with the CHAOS CUT:

> the honest version: {actually} "actually"s, {wait} "wait"s, {undo} demanded
> reverts. one week.

**July 8–10** — repost the best user cards (quote-tweet with one specific
observation each, never just "nice"). Ship ONE absurd follow-up stat as a new
card type based on what people ask for in replies (candidates already in
stats.json: scary commands, "continue"-only prompts, subagents spawned).

## Hero post — 3 variants (pick one, morning of)

**A. Deadpan stat-first** *(default pick)*

> fable 5 wrote {loc} lines of code for me this week.
> i said "you're right" to it 0 times. it said it to me {youreRight} times.
>
> npx seshy-wrapped --model fable

**B. Self-deprecating**

> got my fable week report card. delegation grade: {letter}.
> "{why}" is a real line my own tool printed about me.
>
> npx seshy-wrapped --model fable

**C. Leverage-flex**

> one week of fable 5: {sessions} sessions, {hours}h of agent time,
> {loc} lines of code, {commits} commits. i typed {userWords} words total.
>
> npx seshy-wrapped --model fable

Rules: numbers in the text must match the attached card exactly (screenshot
auditors are the distribution). Lowercase. No hashtags. No link in the hero
post — command only; the repo link goes in reply 2.

## Reply-guy templates (for other people's "Fable week" posts)

Add value first, card second, never link-drop. Rotate these:

1. > same — my week was {loc} lines and {fbombs} F-bombs. if you want the
   > actual numbers off your logs: npx seshy-wrapped --model fable
2. > you can pull your real stats for this — it reads the local session logs.
   > mine said {youreRight} "you're right"s in 7 days. npx seshy-wrapped
3. > the stat nobody posts: mine "should work now"-ed me {shouldWork} times
   > this week. {youWrong} times it did not. npx seshy-wrapped --model fable
4. > if you're doing a farewell post anyway, one command gives you the
   > receipt for it (literally, there's a receipt theme). npx seshy-wrapped
5. > yours will be funnier than mine — {n} sessions after midnight is a cry
   > for help. npx seshy-wrapped --model fable

## Distribution targets (July 7, engage don't spam)

Reply with your card + one specific observation to whoever posts Fable-week
retrospectives among: **@swyx, @simonw, @levelsio, @theo, @mattpocockuk,
@GergelyOrosz, @rauchg, @steipete, @transitive_bs (Travis Fischer), @jaredpalmer,
@karpathy (if he posts about Fable at all), @thorstenball, @mitchellh,
@kentcdodds, @wesbos**. Priority: whoever is ALREADY posting "Fable week"
content that morning — search `"fable" week`, `"fable 5" july 7`, `fable
credits` and sort by latest. Skip anyone who hasn't posted about it; cold
mentions read as spam.

**The reply-chain mechanic:** end the hero post's first reply with "post
yours 👇" — asking for replies (not reposts) is what the algorithm rewards,
and every user card in the thread is a testimonial with your watermark.

## Hacker News (July 8, not launch day — let X seed it)

Show HN title:

> Show HN: Seshy Wrapped – local, deterministic stats cards from your coding-agent logs

First comment (yours, immediately):

> Reads the JSONL/SQLite session logs Claude Code, Codex, Gemini CLI and
> OpenCode already keep on disk. Zero network calls, no telemetry, no
> account — the pipeline is one pure pass and CI diffs two runs byte-for-byte
> so the numbers are reproducible. Every stat's exact regex is documented in
> STATS.md because I assume you'll audit them (please do). The funny stats
> ("times it said 'you're absolutely right'") are the hook but the parser
> core is shared with seshy, a session search/resume tool.

HN's love language: local-only, deterministic, auditable, no telemetry. Lead
with that, let the humor be discovered.

## r/ClaudeAI (July 7 evening)

Title: `I counted every time Claude told me I was "absolutely right" (and 30 other stats) — local one-liner, no API key`
Body: your chaos-cut card + the command + STATS.md link + the honest caveat
about cleanupPeriodDays (that caveat alone is post-worthy there).

## Metrics + kill criteria

**Success by July 9:**
- Hero post ≥ 100k impressions or ≥ 500 likes
- npm downloads day-1 ≥ 300 (only honest proxy for runs; we have no telemetry)
- ≥ 25 user-posted cards findable via search ("seshy-wrapped" / "SESHY WRAPPED")
- ≥ 100 GitHub stars on seshy

**If it lands:** July 9 ship the most-requested stat as a new cut; pin a
"best cards" QT thread; start the `seshy wrapped --period month` recurring
ritual (posted monthly, becomes a franchise).

**If it doesn't (< 20k impressions, < 50 downloads):** don't push the same
post again. The asset still exists — fold it into seshy's README as the
visual hook, re-cut the video for the next model-retirement moment (they now
happen quarterly), and A/B the receipt theme as the hero image instead of
machine cut. One relaunch max.

## Risks

- **Someone audits a count and disputes it.** Mitigation: STATS.md defines
  every pattern; determinism suite proves reproducibility. Respond with the
  definition and a "PRs welcome on the regex" — never defensiveness. If they
  found a real bug: fix within hours, thank them publicly (auditors become
  advocates).
- **The Fable window gets extended.** Post still works — reframe to "week
  one of Fable" and keep the `--model fable` hook; the card says FABLE 5
  either way.
- **npm name squatting.** Reserve `seshy-wrapped` before this file is even
  merged. It's the first pre-flight checkbox for a reason.
- **A competitor Wrapped (ccwrapped, agentwrapped, ai-wrapped) rides the
  moment.** Ours is the only one with model-filtering (`--model fable`), the
  receipt theme, and documented deterministic counts. If they post first,
  reply-guy them with our card — the thread works in our favor.
- **This file is on a public branch.** Accepted tradeoff per launch owner;
  strategy leaking early costs less than the file being lost. Strip from the
  PR before merge if that changes.
