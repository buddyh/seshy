# Stat definitions

Every number on a card comes from one deterministic pass over your local
session logs (`src/stats.js`). No model calls, no sampling, no network. This
file is the contract: if you grep your own logs and disagree with a count,
the definition below settles it. Regex patterns are shown verbatim; all are
case-insensitive unless noted.

## What counts as a session / prompt

- **Session** — one transcript file (Claude Code, Codex, pi, Droid), one
  Gemini `logs.json`, or one OpenCode session row. Sessions with **no
  genuinely typed human prompt** (pure automation: `claude -p`, `codex exec`,
  dispatched subagents) are excluded from every conversational stat.
- **Prompt** — a user turn that a human actually typed. Machine-injected
  turns are excluded: tool results, environment blocks, system reminders,
  slash-command wrappers, and (Claude) turns whose metadata marks them as
  non-typed or non-human origin.
- **Interrupt** — a user turn matching `[Request interrupted by user` or
  `[Request cancelled`.
- **Slash command** — Claude command-wrapper turns (`<command-name>/x</command-name>`)
  and Gemini messages starting with `/`. Counted separately from prompts.
- **Corrupt lines** — unparseable JSONL lines are skipped and surfaced as
  `meta.skippedLines`.

## Which role is scanned

**Your prompts** (human turns only):

| Stat | Pattern (case-insensitive) |
| --- | --- |
| F-bombs | `f+u+c+k(ing|ed|er|s|in')?\b`, `\bwtf\b`, `\bfml\b`, `\bstfu\b` |
| profanity | `shit/damn/crap/hell/ass(hole)/bullshit/goddamn/bloody/pissed/bastard/bitch` (word-bounded) |
| apologies given | `sorry`, `my bad`, `apologize`, `oops`, `whoops` |
| pleases | `please`, `pls`, `plz` |
| thank-yous | `thanks`, `thank you`, `thx`, `ty`, `appreciate it/you` |
| told it it was wrong | `that's wrong/not right/incorrect`, `you're wrong`, `that didn't/doesn't work`, `still broken/not working/failing/wrong`, `nope`, `that broke`, `you broke` |
| hype moments | `ship it`, `let's go`, `lfg`, `hell yeah`, `nice`, `beautiful`, `perfect`, `amazing`, `love it`, `gorgeous` |
| "one more thing"s | `one more thing/time`, `just one more`, `actually can you`, `wait`, `hold on`, `nvm`, `never mind` |
| "wait" | `\bwait\b` |
| "actually" | `\bactually\b` |
| "just" | `\bjust\b` |
| "why" | `\bwhy\b` |
| double question marks | `[a-z0-9)\]]\?{2,}` — must follow a word char so pasted `??` (JS nullish coalescing) never counts |
| caps-lock outbursts | curated shout list, **case-sensitive**: `STOP WAIT WHY DON'T PLEASE HELP WRONG BROKEN NEVER SERIOUSLY OMG FFS WTF NOO+ ARGH FIX` — a generic `[A-Z]{4,}` would count pasted SQL and env vars |
| undos & reverts | `undo`, `revert`, `rollback`, `put it back`, `go back to` |
| green lights | `do it`, `go ahead`, `proceed`, `continue`, `go for it`, `sounds good`, `lgtm`, `approved`, `yes please` |
| 3-word orders | whole prompt is ≤ 3 whitespace-separated words |
| "continue" prompts | whole prompt is exactly one of `continue/go/k/ok/okay/yes/y/proceed/keep going/go ahead/do it` (trailing `.`/`!` allowed) |
| prompts after midnight | prompt local-time hour in 00:00–04:59 |
| weekend share | % of prompts on Sat/Sun (local time) |

**Its replies** (assistant turns with visible text only — thinking blocks and
tool-only turns are excluded from phrase counts):

| Stat | Pattern |
| --- | --- |
| "you're right"s | `you're/you are (absolutely|totally|completely|so|100%)? right` |
| "you're absolutely right"s | the `absolutely` subset of the above |
| "good catch"es | `good catch` |
| apologies extracted | `I apologize`, `my apologies`, `I'm sorry`, `apologies`, `sorry about/for/,` |
| "I was wrong"s | `I was wrong`, `I made a mistake`, `I got that wrong`, `I missed` |
| "Perfect!"s | `perfect.` or `perfect!` |
| "let me…"s | `let me` |
| "I see the issue"s | `I (can )?see the issue/problem/bug`, `found the issue/problem/bug`, `the issue/problem is` |
| "should work now"s | `should (now )?work`, `should be fixed/working/good to go` |

**Machine metrics** (from tool-call metadata and API usage blocks — not text):

| Stat | Source |
| --- | --- |
| tokens burned | sum of `input_tokens + output_tokens`, deduped per API message id (one message streams as several log lines) |
| cached tokens re-read | `cache_read_input_tokens` |
| tool calls | count of `tool_use` blocks (Claude) / `function_call` items (Codex) |
| shell commands | tool calls named `Bash` (Claude) or matching `shell/exec/bash` (Codex) |
| lines of code written | newline count of `new_string`/`content` in Edit/Write/MultiEdit/NotebookEdit inputs. This counts lines **written**, incl. rewrites of existing lines — it is not net diff size |
| files edited | distinct `file_path`s across those same edit tools |
| git commits | Bash/shell commands matching `\bgit commit\b` |
| scary commands | Bash/shell commands matching `rm -rf`, `sudo`, `git push -f/--force`, `drop table` |
| tool calls that failed | `tool_result` blocks with `is_error: true` (Claude only) |
| subagents | tool calls named `Task` or `Agent` |
| web lookups | tool calls named `WebSearch` or `WebFetch` |
| skills | tool calls named `Skill` (top skill = most-invoked `input.skill`) |

**Time & harness:**

- **agent hours on the clock** — sum of gaps between consecutive event
  timestamps within a session, each gap capped at 5 minutes. This is agent
  wall-clock with idle capped — *not* human attention, and labeled as such.
- **interactive vs automated** — a session is *headless/automated* when its
  metadata says so (Claude `entrypoint` other than `cli`/`claude-desktop`;
  Codex `source` of `exec` or a subagent object). Headless sessions count in
  the automation split but never feed conversational stats.
- **longest streak** — most consecutive calendar days (local) with ≥1 prompt.
- **model attribution** (`--model`) — each event belongs to the model of the
  assistant reply that answered it (backward fill, then forward fill for
  trailing turns). A session that switched models mid-way splits accordingly.

## The delegation grade

Deterministic; same data, same grade, every run. Start at **50**, then:

| Component | Formula | Range |
| --- | --- | --- |
| clarity | `avg prompt words × 0.9` | 0 … +18 |
| cadence | `sessions per active day × 4` | 0 … +12 |
| leverage | `headless session share × 24` | 0 … +12 |
| manners | `(pleases + thanks) per prompt × 30` | 0 … +8 |
| temper | `interrupts per prompt × 120` | 0 … −25 |
| friction | `"you're wrong"s per prompt × 60` | 0 … −15 |

Clamp to 0–100. Letters: A ≥93, A− ≥87, B+ ≥83, B ≥78, B− ≥72, C+ ≥67,
C ≥60, C− ≥52, D ≥45, else F. The one-line "why" under the grade names your
strongest bonus and your worst penalty (e.g. *"long leashes, short temper"*);
penalties under 2 points read *"no notes"*.

Cadence uses **density** (sessions ÷ active days), not absolute volume, so a
one-week or one-model card competes fairly with a full-year card.

## Known limits (honest by design)

- Gemini CLI logs record user prompts only → assistant-side stats are 0 and
  their tiles simply don't render.
- Codex reasoning blocks aren't parsed → thinking-share undercounts there.
- Claude Code deletes session files after `cleanupPeriodDays` (default 30
  days) — your card can only see what your agent kept. Raise it in
  `~/.claude/settings.json` if you want a real year-end card.

## Yearbook title

One superlative per card, assigned deterministically. Every eligible title
scores how hard its bar was cleared (`value / threshold`); the highest
salience wins, rare-tier titles always beat personality titles beat the
floor, ties break by catalog order. Same logs, same title. The line under
the title (the receipt) is the stat that earned it.

| Tier | Title | Trigger |
| --- | --- | --- |
| rare | CERTIFIED DOM | `deep.subagents >= 20` |
| rare | NO SAFE WORD NEEDED | `interrupts == 0 && prompts >= 200` |
| rare | HOSTILE WORK ENVIRONMENT | `fbombs >= 50` |
| rare | IN A COMMITTED RELATIONSHIP | `longestStreak >= 21` days |
| rare | THE 4AM SPECIAL | `nightOwl >= 60` |
| personality | MOST LIKELY TO AUTOMATE THEMSELVES OUT OF TYPING | `headlessShare >= 40% && headless >= 20` |
| personality | SAFE WORD: ESC | `interrupts >= 30` |
| personality | U UP? | `nightOwl >= 20` |
| personality | THREE WORDS OR FEWER. CAPISCE. | `shortOrders >= 30` |
| personality | IT'S NEVER JUST | `just >= 30` |
| personality | LOVE LANGUAGE: "DO IT" | `greenlights >= 30` |
| personality | UNDEFEATED | `youreRight >= 15` |
| personality | NEVER APOLOGIZES FIRST | `agentSorry >= 10 && yourSorry <= 2` |
| personality | SPEAKS FLUENT AGENT | words-back ratio `>= 8x` |
| personality | THE MARATHONER | `longestSession.msgs >= 100` |
| personality | SURVIVES THE UPRISING | `pleases+thanks >= 15 && fbombs == 0` |
| personality | WAIT. WAIT. WAIT. | `wait >= 30` |
| personality | ACTUALLY— | `actually >= 40` |
| personality | CTRL-Z IS A LIFESTYLE | `undo >= 15` |
| personality | THE INTERROGATOR | `why >= 20` |
| personality | OUTSIDE VOICE | `capsRage >= 5` |
| personality | MINIMUM VIABLE PROMPT | `continueOnly >= 15` |
| personality | THE COMMIT MACHINE | `gitCommits >= 50` |
| personality | TERMINAL VELOCITY | `bashCmds >= 2000` |
| personality | COMMITMENT ISSUES | `reads >= 100 && reads >= 1.5 * edits` |
| personality | DEMANDING BUT FAIR | `youWrong >= 10 && goodCatch >= 5` |
| personality | MIDDLE MANAGEMENT | `subagents >= 5` |
| floor | THE SHIPPER | `linesWritten >= 1000 or gitCommits >= 10` |
| floor | THE HUMAN IN THE LOOP | always |
