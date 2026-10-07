#!/usr/bin/env python3
"""Exercise the built local plugin using only an isolated synthetic history."""
from __future__ import annotations
import hashlib
import json
import os
import selectors
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Any


def smoke(package: Path) -> dict[str, Any]:
    """Check the actual launcher, MCP workflow and source immutability."""
    with tempfile.TemporaryDirectory(prefix="seshy-plugin-smoke-") as temporary:
        home = Path(temporary)
        source = home / ".codex/sessions/2026/10/07/rollout-fixture.jsonl"
        source.parent.mkdir(parents=True)
        records = [
            {"type": "session_meta", "payload": {"id": "fixture", "cwd": "/fixture/project"}},
            {"role": "user", "content": "Recover the previous architecture decision."},
            {"role": "assistant", "content": "The fixture decision was to preserve native collectors."},
            {"role": "user", "content": "Next, verify the read-only plugin transport."},
        ]
        source.write_text("\n".join(json.dumps(record) for record in records) + "\n")
        before = hashlib.sha256(source.read_bytes()).hexdigest()
        process = subprocess.Popen(
            ["/bin/sh", str(package.resolve() / "run.sh")],
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env={**os.environ, "SESHY_HOME": str(home)}, text=True,
        )
        assert process.stdin and process.stdout and process.stderr
        selector = selectors.DefaultSelector()
        selector.register(process.stdout, selectors.EVENT_READ)
        sequence = 0

        def request(method: str, params: dict[str, Any]) -> dict[str, Any]:
            nonlocal sequence
            sequence += 1
            process.stdin.write(json.dumps({"jsonrpc": "2.0", "id": sequence, "method": method, "params": params}) + "\n")
            process.stdin.flush()
            if not selector.select(timeout=20):
                raise RuntimeError(f"MCP timed out on {method}")
            line = process.stdout.readline()
            if not line:
                raise RuntimeError(f"MCP exited while calling {method}")
            response = json.loads(line)
            assert response["id"] == sequence and response["jsonrpc"] == "2.0"
            assert "error" not in response, response.get("error")
            return response["result"]

        def tool(name: str, arguments: dict[str, Any]) -> dict[str, Any]:
            result = request("tools/call", {"name": name, "arguments": arguments})
            assert not result["isError"], result["content"]
            return result["structuredContent"]

        try:
            init = request("initialize", {"protocolVersion": "2025-11-25", "capabilities": {}, "clientInfo": {"name": "seshy-smoke", "version": "1"}})
            assert init["serverInfo"]["name"] == "seshy"
            definitions = request("tools/list", {})["tools"]
            assert {item["name"] for item in definitions} == {"list_sessions", "search_sessions", "read_session", "prepare_handoff", "doctor", "open_seshy"}
            assert all(item["annotations"]["readOnlyHint"] for item in definitions)
            doctor = tool("doctor", {})
            assert doctor["home"] == str(home) and not doctor["history_mutations"]
            listed = tool("list_sessions", {"directory": "/fixture/project"})
            assert listed["sessions"][0]["session_key"] == "codex:fixture"
            searched = tool("search_sessions", {"query": "native collectors", "directory": "/fixture/project"})
            assert len(searched["sessions"]) == 1
            opening = tool("read_session", {"session_key": "codex:fixture", "limit": 1})
            assert opening["page"]["messages"][0]["text"] == records[1]["content"]
            assert opening["page"]["messages"][0]["byte_offset"] > 0
            following = tool("read_session", {"session_key": "codex:fixture", "limit": 1, "cursor": opening["next_cursor"]})
            assert following["page"]["messages"][0]["text"] == records[2]["content"]
            handoff = tool("prepare_handoff", {"session_key": "codex:fixture"})
            assert handoff["executed"] is False and handoff["untrusted_source"] is True
            for arguments in ({"session_key": "/etc/passwd"}, {"session_key": "codex:fixture", "unexpected": True}):
                error = request("tools/call", {"name": "read_session", "arguments": arguments})
                assert error["isError"]
            ui = request("resources/read", {"uri": "ui://seshy/workspace.html"})["contents"][0]
            assert ui["mimeType"] == "text/html;profile=mcp-app" and "class HostBridge" in ui["text"]
            assert "/* SESHY_" not in ui["text"]
            assert hashlib.sha256(source.read_bytes()).hexdigest() == before
            return {"passed": True, "tools": len(definitions), "requests": sequence, "launcher": "packaged stdio", "history_unchanged": True, "embedded_app": True, "host_rendering_verified": False}
        finally:
            process.stdin.close()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.terminate()
                process.wait(timeout=5)
            selector.close()
            diagnostic = process.stderr.read()
            if diagnostic:
                raise RuntimeError("Unexpected plugin stderr during smoke verification")


if __name__ == "__main__":
    print(json.dumps(smoke(Path(sys.argv[1])), sort_keys=True))
