#!/usr/bin/env bash
# Build seshy, install it onto PATH, run the VHS showcase tape, and archive the
# screenshots into a timestamped folder so each redesign pass is reviewable.
#
#   tools/shoot.sh [label]
#
# PNGs land in tools/shots/ (latest) AND tools/shots/passes/<timestamp>[__label]/.
set -euo pipefail
cd "$(dirname "$0")/.."

label="${1:-}"
stamp="$(date +%Y-%m-%d_%H%M%S)"
dir="tools/shots/passes/${stamp}${label:+__$label}"

go build -o seshy .
tools/install.sh        # keep the on-PATH `seshy` in sync with this build
vhs tools/showcase.tape

mkdir -p "$dir"
cp tools/shots/*.png "$dir"/ 2>/dev/null || true
cp tools/shots/walkthrough.gif "$dir"/ 2>/dev/null || true

echo "archived $(ls "$dir"/*.png 2>/dev/null | wc -l | tr -d ' ') shots -> $dir"
