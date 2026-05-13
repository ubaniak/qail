#!/usr/bin/env bash
# Build a Windows installer (.exe) for qail using Wails' built-in NSIS
# template.
#
# Wails ships an NSIS script template; the `-nsis` flag wraps it around
# the compiled .exe so the user gets a familiar "Welcome → Install →
# Finish" wizard that installs to %LOCALAPPDATA%\Programs\qail and adds
# a Start Menu shortcut.
#
# Run this script ON Windows (recommended) or from a Linux/macOS host
# with mingw-w64 + makensis installed for cross-compilation. The Wails
# docs cover the cross setup; CI uses windows-latest GitHub runners
# which already have everything.
#
# Output: build/bin/qail-amd64-installer.exe

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v wails >/dev/null 2>&1; then
  echo "error: wails CLI not on PATH" >&2
  exit 1
fi

# WebView2 mode embed-bootstrapper is preferred — Windows 10 1803+ ships
# WebView2 by default but older systems need it; embed-bootstrapper has
# the installer pull it down at first run if missing.
WEBVIEW2_MODE="${WEBVIEW2_MODE:-embed}"

echo "==> wails build -nsis -platform windows/amd64"
wails build \
  -platform windows/amd64 \
  -nsis \
  -webview2 "$WEBVIEW2_MODE" \
  -tags production \
  -skipbindings

OUT="$ROOT/build/bin/qail-amd64-installer.exe"
if [[ ! -f "$OUT" ]]; then
  echo "error: expected $OUT not produced" >&2
  echo "wails build output:" >&2
  ls -la "$ROOT/build/bin/" >&2 || true
  exit 1
fi

mkdir -p "$ROOT/build/installers"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
FINAL="$ROOT/build/installers/qail-${VERSION}-windows-amd64-installer.exe"
cp "$OUT" "$FINAL"

echo
echo "✓ built $FINAL"
ls -lh "$FINAL"
