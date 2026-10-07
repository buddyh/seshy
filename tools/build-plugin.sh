#!/bin/sh
# Build a self-contained plugin with installed Go dependencies; never installs.
set -eu
repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output_dir=${1:-"$repo_dir/dist"}
go_bin=${SESHY_GO:-go}
platform=${GOOS:-$("$go_bin" env GOOS)}
arch=${GOARCH:-$("$go_bin" env GOARCH)}
case "$platform/$arch" in darwin/arm64|darwin/amd64|linux/arm64|linux/amd64) ;; *) echo "Supported package platforms: macOS/Linux, ARM64/AMD64." >&2; exit 1 ;; esac
mkdir -p "$output_dir/seshy/bin" "$output_dir/seshy/skills" "$output_dir/seshy/assets"
output_dir=$(CDPATH= cd -- "$output_dir" && pwd)
mkdir -p "$output_dir/.agents/plugins"
cp "$repo_dir/plugins/marketplace.json" "$output_dir/.agents/plugins/marketplace.json"
cp "$repo_dir/plugins/seshy/plugin.json" "$repo_dir/plugins/seshy/mcp.json" "$repo_dir/plugins/seshy/run.sh" "$repo_dir/plugins/seshy/README.md" "$repo_dir/LICENSE" "$output_dir/seshy/"
cp -R "$repo_dir/plugins/seshy/skills/." "$output_dir/seshy/skills/"
cp -R "$repo_dir/plugins/seshy/assets/." "$output_dir/seshy/assets/"
cd "$repo_dir"
GOPROXY=off CGO_ENABLED=0 GOOS="$platform" GOARCH="$arch" "$go_bin" build -trimpath -ldflags='-s -w' -o "$output_dir/seshy/bin/seshy-$platform-$arch" .
chmod 755 "$output_dir/seshy/run.sh" "$output_dir/seshy/bin/seshy-$platform-$arch"
python3 "$repo_dir/tools/package-plugin.py" "$output_dir"
