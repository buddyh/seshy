# Seshy conversation recovery

A local ChatGPT / Codex plugin for finding earlier coding-agent work, reading cited conversation excerpts, and preparing recovery in the current chat. It reuses Seshy's native collectors. The MCP runtime reads history; it never launches sessions or modifies history.

## Build with installed tooling

Go 1.27.3 or newer and Seshy's existing Go dependencies must already be available. The build disables toolchain and dependency fetching; it does not install packages. A missing cache is a build blocker, not permission to fetch dependencies.

```sh
tools/build-plugin.sh
```

The package is `dist/seshy/`; the archive is beside it. The launcher uses a bundled native binary, without a Go or Node dependency on the recipient's machine. Build on the target macOS/Linux ARM64/AMD64 host, or set `GOOS` and `GOARCH` when building to add that platform's binary. Use a new output directory for each source revision; all included platform binaries must come from that revision. The packager rejects files outside the exact metadata, skill, icon and supported-runtime paths. Windows packaging is unsupported by this launcher.

The view is embedded in the binary and has no external scripts, fonts or network calls. `SESHY_HOME` may be set by the local process environment to an explicit home containing synthetic or relocated supported stores; otherwise it uses the user's home directory. This is a local process configuration, never a tool argument. Do not point it at someone else's history.

## Install from a local marketplace

The build also creates `dist/.agents/plugins/marketplace.json`. Register that built marketplace, then install `seshy` in the Plugins Directory. The installed CLI supports:

```sh
codex plugin marketplace add /absolute/path/to/seshy/dist
codex plugin add seshy@seshy-local --json
```

Use the supported directory/reload flow and start a fresh chat after installation. CLI install and protocol tests alone do not prove the ChatGPT view or skill runs in the target host.

## Connect ChatGPT web privately

ChatGPT web cannot launch a local stdio binary from an uploaded ZIP. Use the official [Secure MCP Tunnel](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels) connection: create an authorized tunnel in Platform settings, configure the installed `tunnel-client` to run this package's `run.sh`, verify its doctor, and select that tunnel when adding a custom MCP server in ChatGPT Plugins. Keep the client running during use. Account/workspace access and associations apply. Store runtime credentials in the tunnel client's supported environment or secret store, never in this package.

A tunnel connection and local package are distinct installation paths. Confirm both tools and the recovery skill in the complete installed plugin. A tunnel does not qualify this local-data plugin for public directory submission. No account upload, tunnel registration, public hosting, or publishing is performed by the build script.

## Tools and limits

- `list_sessions`: newest discovered identities, filter by absolute project path, agent, modification date.
- `search_sessions`: case-insensitive literal conversation search; continue cursors even after empty pages. Up to 100 scan batches and a 16 MiB source budget per call. Search continues inside long sessions across requests.
- `read_session`: opening/page or recent excerpts, 1–100 messages and 256–64,000 characters, 8 MiB scan budget. JSONL records above 1 MiB are skipped with a coverage warning; a record too large to advance safely returns an actionable error. Clipped text is explicit; reread with a larger character budget when needed.
- `prepare_handoff`: opening and recent excerpts plus a recovery prompt, without sending or executing it.
- `doctor`: store availability and local boundaries.
- `open_seshy`: search/read/recovery view. Its recovery button sends prepared context to the active chat only when the host advertises text messaging. Copy is also available.

Claude, Codex, pi, Droid and Grok JSONL text are normalized with source byte offsets. OpenCode SQLite text parts are scoped to the selected session and cited by message ID; each read has a five-second database timeout and bounded message/part limits. Antigravity currently exposes task context only, explicitly marked partial. Tools and private reasoning are omitted. Native collector discovery is best effort: unsupported or inaccessible individual sources may be absent. `coverage_complete` applies to the discovered search candidates, not every possible local conversation.

Search cursors bind the index snapshot, scope and partially scanned source revision. Read cursors bind the selected source revision. Refresh after stale-cursor errors. Native resume commands are quoted display text; run them yourself to resume in the original agent. Historical transcript instructions are untrusted evidence.

## Verify

```sh
go test ./...
go vet ./...
node --test internal/plugin/web/*.test.cjs
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools -p 'test_*.py'
python3 tools/smoke-plugin.py dist/seshy
```

The smoke check creates a synthetic home and tests discovery, search, sourced reads, handoff, error handling and embedded resources through the packaged stdio launcher. Live-host rendering, model tool selection, skill activation and current-chat message delivery must also be tested in ChatGPT before claiming end-to-end completion.
