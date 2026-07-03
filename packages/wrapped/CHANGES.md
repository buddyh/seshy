# Changes vs the agent-wrapped prototype

Everything that changed on the way from the private `agent-wrapped` prototype
to `seshy-wrapped`, and why.

## Branding

- Command renamed `npx agent-wrapped` → `npx seshy-wrapped`; every footer,
  README, and terminal string updated. One funnel. (Also: npm rejects
  `agent-wrapped` as a punctuation-typosquat of the existing `agentwrapped`
  package, so the old name was unpublishable anyway.)
- Card title now reads **SESHY WRAPPED**.
- Footer standard on every theme: `@handle (if provided) · npx seshy-wrapped ·
  made with seshy`, rendered exactly once. The receipt theme previously drew a
  second footer strip outside the paper — removed.

## Determinism (the big one)

- **Single stats pass**: `collect()` produces one canonical report; every cut
  and theme renders from it. The prototype re-scanned the logs per invocation,
  so cards rendered minutes apart disagreed as new sessions landed — including
  the session doing the rendering.
- Canonical `stats.json` written with recursively sorted keys and no
  wall-clock fields; CI runs the pipeline twice and diffs bytes.
- Sessions are fed to the accumulator in sorted-path order (not filesystem or
  completion order), pinning the one order-sensitive tiebreak (longest
  session's project).
- All ratios go through fixed-decimal rounding helpers; map orderings sort by
  count desc then key asc — a total order.
- Golden fixtures with hand-verified counts gate every stat definition.

## Cuts & themes

- "variants" renamed **cuts** (`--cut`), matching the card kicker language
  (MACHINE CUT, CHAOS CUT, AGENT CUT).
- Theme `outrun` renamed **sunset** and stays the default.
- Themes kept: sunset, terminal, starfield, receipt, billboard, crt.
- Themes dropped: **oceanline** (the horizon sun fought the hero number for
  attention in every draft) and **wordmark** (closest to a plain poster,
  weakest share-bait of the eight).
- Punchline picker rebuilt: candidates are classed PAIRED (two stats in
  tension) > STREAK (time-shaped) > RAW (one number + written twist), ranked
  deterministically. Every punchline is stat + twist; a stat alone never
  renders.
- New rookie card: corpora with fewer than 10 sessions get a ROOKIE CARD
  kicker tag and their own punchline instead of an error or an embarrassing
  empty card.

## Stat fixes

- "hours in the driver's seat" renamed **agent hours on the clock** — it was
  agent wall-clock, not human attention. The interactive-vs-automated split
  carries the leverage story honestly.
- Grade rubric v2: absolute session volume replaced with **density**
  (sessions per active day) so short windows aren't structurally capped at
  B−; added an automation-leverage component; every grade now renders a
  one-line "why" (e.g. "long leashes, short temper"). Documented in STATS.md.
- Post-line copy no longer misattributes "you're right" counts.

## New capability

- **Gemini CLI** and **OpenCode** sources added (user-side stats for Gemini,
  full stats for OpenCode via `node:sqlite`, degrading quietly when absent).
- `--model` filter with per-turn attribution; the card retitles itself
  (`FABLE 5 · 2026`).
- `--period week|month|year|all`, `--since YYYY-MM-DD`, `--all-cuts`,
  `--json`, `--out <dir>` per the launch spec.
- Corrupt log lines are skipped, counted, and disclosed on the terminal
  summary instead of silently absorbed.
- Output PNGs are 1600×2000 (4:5) — X-optimized, retina-crisp.
- `seshy wrapped` Go subcommand delegates to `npx seshy-wrapped` so both
  invocations are the same product.
