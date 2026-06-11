---
name: seshy
description: Discover and resume past AI coding-agent sessions with the seshy CLI — across Claude Code, Codex, Grok, pi, OpenCode, Antigravity, and Droid. Use to list/summarize the sessions for a repo, resume the most recent one, browse the latest sessions across every repo, produce a machine-readable index of all sessions, or hide headless (claude -p / codex exec) runs. Triggers on "seshy", "what sessions do I have", "resume my last session", "recent sessions", "sessions in this repo", "resume the previous conversation", "pick up where I left off". For searching session CONTENTS (find where something was discussed), use the conversation-search skill instead.
---

# seshy

`seshy` is a single static Go binary that finds and resumes your past coding-agent sessions
for any directory, unifying **seven** agents — Claude Code, Codex, Grok, pi, OpenCode,
Antigravity (`agy`), and Droid. It reads each agent's session store natively, concurrently,
and **read-only**, and every result carries the agent's own native command to resume it.

Use this skill to **discover and resume** sessions. To search the *contents* of past
sessions ("find where we discussed X"), use the **conversation-search** skill.

## The two jobs

1. **Discovery / resume** (this skill): enumerate sessions for a repo or across the machine,
   resume the most recent, or produce an index for tooling.
2. **Content search** (conversation-search skill): `seshy search` over transcript text.

## Commands

```bash
seshy [path]            # interactive picker for a directory (default: cwd) — for humans
seshy list [path]       # sessions for a directory; table for humans, JSON when piped
seshy summary [path]    # compact one-line digest of a project's sessions
seshy last [path]       # resume the most recent session in the directory (execs the agent)
seshy all               # most recent sessions across EVERY repo (interactive, paged)
seshy sessions          # every session across all agents/repos as JSON (uncapped index)
seshy search <pattern>  # content search across agents (see conversation-search skill)
seshy config            # show settings; `seshy config set <key> <true|false>` to change
```

When running on the user's behalf in a non-interactive context, prefer the **JSON** forms
(`-o json`, the default when piped) and read the fields you need — do not try to drive the
interactive picker.

## Flags

| Want | Flag |
|---|---|
| target a directory | `-C <dir>` or pass a path argument |
| include subdirectories | `--all` |
| one agent only | `--agent claude` (or `codex`, `grok`, `pi`, `opencode`, `agy`, `droid`) |
| limit / widen results | `-n <N>` |
| machine-readable | `-o json` (default when piped) or `-o ndjson` |

## Common tasks → commands

- **"What sessions do I have for this repo?"**
  `seshy list -C <dir> -o json` — every agent, newest first. Or `seshy summary -C <dir>` for
  a one-line digest (counts by agent + most recent).
- **"Resume my last session here."**
  `seshy last -C <dir>` (optionally `--agent claude`). This **execs** the native resume
  command and replaces the process — only run it when the user actually wants to jump in.
  To just *show* the command instead, use `seshy list -C <dir> -o json` and read
  `.sessions[0].resume` / `.sessions[0].dir`.
- **"What have I been working on lately / across all my repos?"**
  `seshy all` (interactive) or, for tooling, `seshy sessions -o json` and rank by `mtime`.
- **"Give me an index of everything for a script."**
  `seshy sessions -o json` — uncapped, no content reads; fields below.

## JSON shape

`seshy list -o json` returns `{ directory, most_recent, sessions[], format_version }`.
Each session: `agent`, `id`, `path`, `dir`, `mtime` (RFC3339 UTC), `preview` (first user
message), optional `model` / `cost` / `msgs`, and `resume` (the exact native command, e.g.
`claude --resume <id>`, run from `dir`). `seshy sessions -o json` returns the same session
objects, uncapped, without previews. `seshy summary` (JSON) returns counts by agent and the
most-recent session.

## Resume model

Resuming **execs the agent's own command** (`claude --resume`, `codex resume`,
`grok --resume`, `pi --session`, `opencode -s`, `agy --conversation=`, `droid --resume`) in
the session's directory — the user continues the real conversation with full native context,
never a summary. seshy never modifies history.

To resume from a script without seshy taking over the process:

```bash
S=$(seshy list -C ~/repos/api --agent codex -o json)
cd "$(jq -r '.sessions[0].dir' <<<"$S")" && eval "$(jq -r '.sessions[0].resume' <<<"$S")"
```

## Hiding headless runs (config)

Some agents leave a lot of non-interactive sessions on disk. Hide them so listings show only
real interactive work:

```bash
seshy config                                # show current settings + file path
seshy config set hideClaudeHeadless true    # hide `claude -p` / Agent-SDK sessions
seshy config set hideCodexExec true         # hide `codex exec` sessions
```

Settings live in `~/.config/seshy/config.json` (XDG-aware) and apply to every listing
(picker, `list`, `all`, `sessions`, `search`). Both are off by default.

## Install

`go install github.com/buddyh/seshy@latest` or `brew install buddyh/tap/seshy`.
