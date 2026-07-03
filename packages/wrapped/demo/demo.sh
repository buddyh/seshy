#!/usr/bin/env bash
# 30-second demo for the launch video. Renders cards from the synthetic test
# fixtures so nothing personal appears on screen. Record the terminal while
# this runs, then cut to the PNGs it writes.
set -euo pipefail
cd "$(dirname "$0")/.."

OUT="${1:-./seshy-wrapped-demo}"
export SESHY_WRAPPED_HOME="$PWD/test/fixtures/home"

banner() {
  printf '\n\033[38;5;212m❯\033[0m \033[38;5;51m%s\033[0m\n\n' "$1"
  sleep 1
}

banner "npx seshy-wrapped"
TZ=UTC node bin/seshy-wrapped.js --out "$OUT" < /dev/null

banner "npx seshy-wrapped --cut machine --theme receipt"
TZ=UTC node bin/seshy-wrapped.js --cut machine --theme receipt --out "$OUT" < /dev/null

banner "npx seshy-wrapped --all-cuts --theme terminal"
TZ=UTC node bin/seshy-wrapped.js --all-cuts --theme terminal --out "$OUT" < /dev/null

printf '\n\033[38;5;84mdemo cards written to %s\033[0m\n' "$OUT"
