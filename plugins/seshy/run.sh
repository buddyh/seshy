#!/bin/sh
set -eu
plugin_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
case "$(uname -s)" in Darwin) platform=darwin ;; Linux) platform=linux ;; *) echo "Seshy plugin supports macOS and Linux. Build the Go runtime for your host." >&2; exit 1 ;; esac
case "$(uname -m)" in arm64|aarch64) arch=arm64 ;; x86_64|amd64) arch=amd64 ;; *) echo "Unsupported Seshy CPU architecture." >&2; exit 1 ;; esac
runtime="$plugin_dir/bin/seshy-$platform-$arch"
if [ ! -x "$runtime" ]; then echo "Seshy runtime missing for $platform/$arch. Build the package on this host with tools/build-plugin.sh." >&2; exit 1; fi
exec "$runtime" mcp
