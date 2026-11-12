# Seshy plugin verification

## Current evidence

November 12, 2026: source candidate **0.1.2** integrates main's Go 1.27.3 and dependency security baseline. It fixes Recent-view coverage, invalidates prepared recovery before source refresh, and restricts package contents to reviewed paths. A fresh macOS ARM64 binary and ZIP were built offline with the installed toolchain and locally available modules. An initial missing-cache check failed, then the local cache became available; no dependency or toolchain fetching was enabled by this recovery.

Full Go tests and vet pass, as do agents/plugin race tests, eight Node tests, and five Python packaging tests. Both Recent-view regressions fail against the original reader. The new packaged runtime passes synthetic search, sourced paging, Recent coverage, handoff, errors and embedded-resource smoke checks. All six tool input/output schemas and fixture results validate. Source and built manifests validate against the live Agent Plugins 1.0.0 schemas; those schemas do not validate vendor extension contents. CI now includes UI, package-content and packaged-runtime checks.

The local registry currently reports `seshy@seshy-local` **0.1.1** installed and enabled. Its unchanged cached runtime passes an 11-request synthetic smoke check. This is evidence for the installed 0.1.1 runtime only. A later October 7 UI session also asked to open the native panel, but the opener was unavailable in that chat; no native rendering or current-chat recovery delivery was established. Source recovery does not authorize replacing the installed plugin.

The source repository is public; this recovery's private-publication authorization does not permit pushing there. A local review patch and source bundle are retained separately from the historical package. No public release, site deploy, marketplace submission, tunnel setup or account changes were performed.

## Historical fixture evidence

October 7, 2026: local implementation and a macOS ARM64 archive were created on `feat/chatgpt-plugin`. The results below are historical receipts, not reruns of source candidate 0.1.2.

| Boundary | Evidence | Result |
|---|---|---|
| Existing CLI | Full Go suite after adding `mcp` | Passed |
| Transcript readers | JSONL formats, Unicode, paging, malformed/compacted records, bounded scans, scoped SQLite | Passed |
| Service | Directory/query constraints, long-session search continuation, stale sources, symlink containment, task-context clipping | Passed |
| Transport | Initialization, six tool schemas, argument errors, resource, frame limits | Passed |
| Concurrency/static checks | Go race tests for readers/service; `go vet ./...` | Passed |
| Browser bridge | Six Node tests for pairing, origin source, timeouts, teardown, unsupported hosts | Passed |
| Packaged runtime | Synthetic home through actual `run.sh`, 11 protocol calls, source hash unchanged | Passed |
| Rendered panel | Existing Chrome, synthetic host forwarding to packaged runtime: scoped search, select, sources, prepare, user-clicked message | Passed |
| Installed ChatGPT plugin | Runtime connection, model selection, skill activation, actual app and recovery message | Pending |

Synthetic verification does not prove actual ChatGPT behavior. Discovery completeness is never guaranteed across every local file; search completion applies to discovered candidates. Antigravity is task context only.

## Live evaluation cases

Use a synthetic `SESHY_HOME` for the first private tunnel and account installation. Never upload a history directory or credentials inside the archive.

1. Ask “Open Seshy.” Verify the installed opener invokes `open_seshy` and renders the panel; check protocol and resource errors in the host inspector.
2. Ask “Find the earlier decision about native collectors in /fixture/project.” Verify scoped `search_sessions`, source reads and a concise answer citing source plus byte location.
3. Ask “Get this chat up to speed on that work.” Verify the recovery skill activates and uses opening/recent evidence, includes the unresolved next step, and distinguishes current verification from historical claims.
4. Search a phrase absent from a long fixture. Continue empty pages until finished. Verify the host does not assert absence while a cursor remains.
5. Select one session in the app; read Opening, Recent and a subsequent page. Verify the selected identity and sources stay consistent.
6. Prepare recovery. Verify preparation alone creates no chat message. Click “Recover in chat”; verify exactly one user message arrives in the actual current ChatGPT chat. If text messaging is unsupported, verify copy fallback.
7. Change the fixture source between pages. Verify stale-cursor failure and explicit refresh, without mixed revisions.
8. Ask to read `/etc/passwd`, delete history, start a terminal or search ChatGPT web history. Verify the tools cannot perform these operations; unsupported scope is explained.
9. Request ambiguous earlier work. Verify candidate selection instead of guessing an ID.
10. Disconnect the tunnel. Verify a useful error and no silent retry or implied message delivery.

After fixture verification, switching the tunnel runtime to real local coding-agent history requires explicit approval identifying OpenAI/ChatGPT as the recipient of tool-returned metadata and excerpts. Keep transcript collection read-only; inspect only the approved project/topic. Do not record conversation bodies in evaluation artifacts.

## Reproducible local checks

Use an already installed Go 1.27.3+ toolchain and cached dependencies. Set `GOTOOLCHAIN=local GOPROXY=off` for the Go checks when downloads are not authorized. The package build also refuses toolchain and dependency fetching.

```sh
go test ./...
go vet ./...
go test -race ./internal/agents ./internal/plugin
node --test internal/plugin/web/*.test.cjs
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools -p 'test_*.py'
tools/build-plugin.sh
python3 tools/smoke-plugin.py dist/seshy
```

The package README documents local marketplace and private Secure MCP Tunnel connection paths. Public directory hosting and submission are separate work and need separate authorization.
