```
███████╗███████╗███████╗██╗  ██╗██╗   ██╗
██╔════╝██╔════╝██╔════╝██║  ██║╚██╗ ██╔╝
███████╗█████╗  ███████╗███████║ ╚████╔╝
╚════██║██╔══╝  ╚════██║██╔══██║  ╚██╔╝
███████║███████╗███████║██║  ██║   ██║
╚══════╝╚══════╝╚══════╝╚═╝  ╚═╝   ╚═╝
```

# seshy

**Find and resume any past coding-agent session, in any directory.**

Every coding agent stores its history in its own place, in its own format. Yesterday's
session is in there somewhere — you just can't get back to it. Run `seshy` in a repo and
get a terminal UI of every past agent session that ran there, newest first, color-coded by
agent. Fuzzy-find it, press **Enter**, and you're dropped straight back into that
conversation via the agent's own native resume.

A single static Go binary. Zero dependencies. Works with no config — with a small optional
config when you want to hide headless runs.

- One fuzzy-searchable list per directory, newest first, color-coded by agent
- Resume the **real** conversation via each agent's own native command — never a summary
- Search the *contents* of every past session across every agent
- Browse the most recent sessions across **every repo** on your machine
- Scriptable: JSON out, with the exact resume command on every session
- Read-only — it never modifies your history

> Website: **[seshy.dev](https://seshy.dev)** · Repo: **github.com/buddyh/seshy**

## Supported agents

seshy reads sessions from seven coding agents and shows them in one unified list:

| Agent | Vendor | Resume command |
| --- | --- | --- |
| Claude Code | Anthropic | `claude --resume <id>` |
| Codex | OpenAI | `codex resume <id>` |
| Grok | xAI | `grok --resume <id>` |
| pi | coding agent | `pi --session <path>` |
| OpenCode | opencode | `opencode -s <id>` |
| Antigravity | Google | `agy --conversation=<id>` |
| Droid | Factory | `droid --resume <id>` |

Each agent's session store is read natively, concurrently, and read-only.

## Install

**Homebrew**

```sh
brew install buddyh/tap/seshy
```

**Go**

```sh
go install github.com/buddyh/seshy@latest
```

## Quick start

```sh
seshy            # interactive picker for the current directory
seshy last       # resume the most recent session here, no questions asked
seshy all        # picker of the most recent sessions across every repo
seshy search "force push"   # find sessions whose contents mention something
```

Run `seshy` in a repo to open the interactive picker:

| Key | Action |
| --- | --- |
| `↑` / `↓` | move |
| `/` or type | fuzzy-filter (by prompt, agent, or repo) |
| `p` | toggle preview pane (first prompt · last message · resume command) |
| `h` | hide/show headless & automated runs (`claude -p` / SDK, `codex exec`); persists to config |
| `↵` | resume the selected session in its native agent |
| `q` | quit |

## Commands

```sh
seshy [path]            # interactive picker for a directory (default: cwd)
seshy list [path]       # table for humans; JSON when piped
seshy summary [path]    # compact digest of a project's sessions (agent-friendly)
seshy last [path]       # resume the most recent session in the directory
seshy all               # most recent sessions across every repo (paged as you scroll)
seshy sessions          # every session across all agents/repos as JSON (an index for tooling)
seshy search <pattern>  # search session contents across agents (excerpt + resume per hit)
seshy config            # show settings (and where they live)
```

### `seshy all` — across every repo, paged

`seshy all` opens a picker of the most recent sessions across **every** repo on the
machine, newest first, showing the repo each session belongs to. It loads a page at a time
and pulls in more as you scroll — the header shows `showing N of M` — so it stays instant
even with thousands of sessions on disk. Repo names are fuzzy-filterable.

### `seshy search` — search the contents of past sessions

```sh
seshy search "react ink"                 # everywhere, every agent
seshy search "xterm" --agent codex       # one agent
seshy search -i "rate limiter"             # case-insensitive
seshy search --regex "seshy|recall"      # regex
seshy search "TUI" ~/repos/myapp         # scope to one repo
```

Search is **literal** (substring, or `--regex`) — not semantic. Each hit prints a cleaned
excerpt and the native command to resume that session. Add `--format json` (the default
when piped) for tooling.

### `seshy sessions` — the full index

`seshy sessions` prints every discovered session across all agents and directories as JSON
— path, agent, dir, mtime, id, and resume command — uncapped and **without** reading file
contents. It's a fast index for search skills and pipelines. Scope it with `--agent`.

## Config

seshy works with no config. When you want to hide non-interactive (headless) runs — `claude
-p` / Agent-SDK sessions and `codex exec` sessions — toggle them off:

```sh
seshy config                               # show current settings + file path
seshy config set hideClaudeHeadless true   # hide claude -p / Agent-SDK sessions
seshy config set hideCodexExec true        # hide `codex exec` sessions
```

Settings live in `~/.config/seshy/config.json` (or `$XDG_CONFIG_HOME/seshy/config.json`)
and apply to every listing — the picker, `list`, `all`, `sessions`, and `search`. Both
filters are **off by default**, so interactive sessions and headless runs all show until
you opt out.

| Key | Hides |
| --- | --- |
| `hideClaudeHeadless` | Claude sessions started with `claude -p` / the Agent SDK (`entrypoint: sdk-cli`) |
| `hideCodexExec` | Codex sessions started via `codex exec` (`source: exec`) |

## Flags

```
-C, --cwd <dir>       target directory (default: current dir)
    --all             include subdirectories
    --agent <name>    filter to one agent (claude|codex|grok|pi|opencode|agy|droid)
-n, --num <int>       max sessions per agent (default 10; 'all' defaults to 20)
-o, --format <fmt>    output: table | json | ndjson
```

## Scriptable

`seshy list`, `seshy sessions`, and `seshy search` print JSON when their output is piped,
and every session includes the exact native command to resume it:

```sh
seshy list ~/repos/myapp | jq -r '.sessions[] | "\(.agent)\t\(.resume)"'
# claude   claude --resume e95bcb24-5f2d-4a11-b604-cc8cef437c2c
# codex    codex resume 019e41f5-ece9-73f3-858e-56aa1901baab

# resume the latest Codex session in a repo from a script
cd "$(seshy list ~/repos/api --agent codex -o json | jq -r '.sessions[0].dir')" \
  && eval "$(seshy list ~/repos/api --agent codex -o json | jq -r '.sessions[0].resume')"
```

## How resume works

seshy never reformats or summarizes your history. When you pick a session it execs the
agent's own resume command (`claude --resume …`, `codex resume …`, etc.) in the right
directory — so you continue the **real** conversation with the agent's full native context,
not a lossy summary.

Everything runs locally against the session files the agents already write. seshy reads
them read-only; it never modifies your history.

## Claude Code skills

This repo ships two [Claude Code](https://claude.com/claude-code) skills under
[`.claude/skills/`](.claude/skills) so an agent can search and resume your history
conversationally:

- **`conversation-search`** — search the contents of past sessions across every agent and
  surface each hit with its resume command ("find where we discussed the album thing").
- **`seshy`** — operate the seshy CLI: list, summarize, find the last session, browse
  across repos, and configure headless filtering.

To use them globally, copy them into your skills directory:

```sh
cp -r .claude/skills/seshy .claude/skills/conversation-search ~/.claude/skills/
```

Or, when working inside a clone of this repo, Claude Code discovers them automatically.

## Build from source

```sh
git clone https://github.com/buddyh/seshy
cd seshy
go build -o seshy .
./seshy --help
```

Requires Go 1.25+. The codebase lives in `internal/` — `agents/` (per-agent collectors and
the resume model), `render/` (table/JSON/summary output), `tui/` (the Bubble Tea picker),
`store/` (shared read helpers), and `config/` (settings).

## License

MIT — see [LICENSE](LICENSE).
```
