#!/usr/bin/env bash
# Build seshy and install it onto PATH at ~/scripts/seshy.
#
# On Apple Silicon, copying *over* an existing signed binary invalidates its
# code-signature mapping and the kernel SIGKILLs it ("zsh: killed"). So we write
# to a temp path, atomically rename it into place (fresh inode), then ad-hoc
# re-sign. SESHY_INSTALL overrides the destination.
set -euo pipefail
cd "$(dirname "$0")/.."

dest="${SESHY_INSTALL:-$HOME/scripts/seshy}"
tmp="$(mktemp "${dest}.XXXXXX")"

go build -o "$tmp" .
chmod 755 "$tmp"
codesign --force --sign - "$tmp" 2>/dev/null || true
mv -f "$tmp" "$dest"          # atomic: new inode, no mapping mismatch

echo "installed $("$dest" --version) -> $dest"
