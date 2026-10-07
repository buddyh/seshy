#!/usr/bin/env python3
"""Validate portable plugin files and archive only the self-contained package."""
from __future__ import annotations
import json
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
    manifest = json.loads((root / "plugin.json").read_text())
    require(manifest["name"] == root.name, "Manifest name must match directory")
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
    require(any((root / "bin").glob("seshy-*")), "Build the runtime before packaging")
    files = sorted(p for p in root.rglob("*") if p.is_file())
    require(not any(p.is_symlink() for p in root.rglob("*")), "Symlinks cannot be packaged")
    allowed = {"plugin.json", "mcp.json", "run.sh", "README.md", "LICENSE"}
    for file in files:
        relative = file.relative_to(root)
        require(relative.parts[0] in allowed | {"assets", "skills", "bin"}, f"Unexpected file: {relative}")
    archive = output.resolve() / f"seshy-{manifest['version']}.zip"
    with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED) as target:
        for file in files:
            target.write(file, file.relative_to(output.resolve()).as_posix())
    print(json.dumps({"package": str(root), "archive": str(archive), "files": len(files), "bytes": archive.stat().st_size}))
    return archive


if __name__ == "__main__":
    package(Path(sys.argv[1]))
