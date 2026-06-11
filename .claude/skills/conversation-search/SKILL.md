---
name: conversation-search
description: Search conversation history across ALL local AI coding-agent sessions — Claude Code, Codex, Grok, pi, OpenCode, Antigravity, and Droid — powered by the seshy CLI. Every result includes the native command to resume that session. Use when the user asks to find previous conversations, search chat history, look up what was discussed before, or find a past session about a specific topic. Triggers on "/conversation-search", "/search", "search conversations", "find where we discussed", "what session talked about", "search my history for".
---

# Conversation Search

Search the contents of past sessions across **all seven coding agents** — Claude Code,
Codex, Grok, pi, OpenCode, Antigravity (`agy`), and Droid — using the **seshy** CLI as the
backbone. seshy scans every agent's session store concurrently (compiled Go) and returns
matches with a cleaned excerpt **and the native command to resume each session**.

This skill is for searching session *contents*. To simply enumerate or resume sessions
("what sessions do I have for this repo", "resume my last one"), use the **seshy** skill.

## Invocation

Run seshy's content search, then present the results:

```bash
seshy search "<pattern>" -o json -n 30
```

Summarize the matches newest-first — agent, repo, the snippet, and the **resume command**
for each hit — so the user can jump straight back into any session.

## IMPORTANT: search literally, reason semantically

`seshy search` matches **literally** — substring, or regex with `--regex`. It does NOT do
semantic/embedding search. A natural-language phrase like
`seshy search "where we built the session cli"` will match **nothing**, because that exact
string appears in no transcript. The intelligence is YOUR job, not the tool's.

So when the user asks in natural language ("find where we discussed the photo album thing"),
do this — don't fire one literal phrase and report "nothing found":

1. **Decompose** the request into concrete terms that would actually appear in the
   transcript — tool names, file names, error strings, repo names, distinctive nouns. Drop
   filler words. ("photo album thing" → `album`, `make-album`, `event-album`, `photos`.)
2. **Run multiple passes** — one per candidate term (use `-i` for case-insensitive), plus a
   `--regex` alternation when useful: `seshy search -i --regex "seshy|recall|antigravity"`.
3. **Synthesize** across the hits — dedupe by session `id`, rank by relevance + recency, and
   present the best sessions, not a raw dump of one query.
4. If the first terms miss, **try synonyms / adjacent terms** before concluding nothing
   exists. Zero hits on one literal phrase ≠ "we never discussed it."

This is what makes the search feel smart: literal engine, semantic operator on top.

## Map the user's request to seshy flags

| User wants | Flag |
|---|---|
| case-insensitive | `-i` |
| regex pattern | `--regex` |
| one agent only | `--agent claude` (or `codex`, `grok`, `pi`, `opencode`, `agy`, `droid`) |
| scope to a repo/dir | pass a path: `seshy search "<pattern>" /path/to/repo` |
| more / fewer results | `-n <N>` (default 25) |
| machine-readable | `-o json` (default when piped) |

For **"last N days"** or **"project named X"** filters seshy doesn't take directly, request
`-o json` and filter results by each match's `mtime` / `dir` before presenting.

## Examples

```bash
seshy search "react ink" -o json              # everywhere, every agent
seshy search "xterm" --agent codex -o json    # Codex only
seshy search -i "rate limiter" -o json          # case-insensitive
seshy search "TUI" ~/repos/myapp -o json      # scoped to one repo
```

## Output

Each JSON match includes: `agent`, `id`, `path` (session file), `dir` (repo), `mtime`,
`resume` (the native resume command, e.g. `claude --resume <id>`), and `snippet` (cleaned
excerpt around the match). Present newest-first and always surface the resume command so the
user can continue any session.

## Related seshy commands

- `seshy sessions -o json` — the full uncapped index of every session (path + resume) for
  broader tooling.
- `seshy all` — interactive picker of the most recent sessions across every repo.
- `seshy last` — resume the most recent session in the current repo.

See the **seshy** skill for session discovery and resume.

## Requires seshy

Install with `go install github.com/buddyh/seshy@latest` (or
`brew install buddyh/tap/seshy`). seshy reads every agent's store read-only and never
modifies history.
