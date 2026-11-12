#!/usr/bin/env python3
"""Validate portable plugin files and archive only the self-contained package."""
from __future__ import annotations
import json
import os
import re
import sys
import zipfile
from pathlib import Path


def require(condition: bool, message: str) -> None:
    """Reject invalid packages even when Python runs with optimization enabled."""
    if not condition:
        raise ValueError(message)


def package(output: Path) -> Path:
    """Validate contained resources and write a ZIP beside the plugin directory."""
    root = output.resolve() / "seshy"
    require(not root.is_symlink(), "Plugin directory cannot be a symlink")
    entries = sorted(root.rglob("*"))
    require(not any(p.is_symlink() for p in entries), "Symlinks cannot be packaged")
    files = [p for p in entries if p.is_file()]
    required = {"plugin.json", "mcp.json", "run.sh", "README.md", "LICENSE",
                "assets/icon.svg", "skills/recover-session/SKILL.md"}
    runtimes = {f"bin/seshy-{platform}-{arch}" for platform in ("darwin", "linux")
                for arch in ("arm64", "amd64")}
    names = {p.relative_to(root).as_posix() for p in files}
    require(required <= names, f"Missing package files: {sorted(required - names)}")
    require(names <= required | runtimes, f"Unexpected package files: {sorted(names - required - runtimes)}")
    require(bool(names & runtimes), "Build the runtime before packaging")
    for name in {"run.sh"} | (names & runtimes):
        require(os.access(root / name, os.X_OK), f"Package entrypoint is not executable: {name}")
    manifest = json.loads((root / "plugin.json").read_text())
    require(manifest["name"] == root.name, "Manifest name must match directory")
    require(isinstance(manifest.get("version"), str) and
            re.fullmatch(r"\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?", manifest["version"]) is not None,
            "Manifest version must be a portable semantic version")
    interface = manifest["extensions"]["com.openai"]["interface"]
    require(len(interface["shortDescription"]) <= 30, "Subtitle exceeds 30 characters")
    mcp = json.loads((root / "mcp.json").read_text())
    server = mcp["mcpServers"]["seshy"]
    require(server["command"] == "/bin/sh" and server["args"] == ["${PLUGIN_ROOT}/run.sh"], "Unexpected MCP entrypoint")
    for field in ("logo", "composerIcon"):
        asset = (root / interface[field]).resolve()
        require(asset.is_relative_to(root) and asset.is_file(), f"Missing contained {field}")
    skill = (root / "skills/recover-session/SKILL.md").read_text()
    require(skill.startswith("---\nname: recover-session\ndescription:"), "Skill frontmatter missing")
    archive = output.resolve() / f"seshy-{manifest['version']}.zip"
    with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as target:
        for file in files:
            target.write(file, file.relative_to(output.resolve()).as_posix())
    print(json.dumps({"package": str(root), "archive": str(archive), "files": len(files), "bytes": archive.stat().st_size}))
    return archive


if __name__ == "__main__":
    package(Path(sys.argv[1]))
