#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
# Windows 10+ 64-bit GUI build. Run this on Windows, or cross-compile from Linux.
# The executable written here is the build you run; a copy made elsewhere is not a substitute.
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-H windowsgui -s -w" -o dist/MPTube.exe ./cmd/youtubemp3
ls -lh dist/MPTube.exe
