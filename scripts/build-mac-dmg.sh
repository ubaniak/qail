#!/usr/bin/env bash
# Build a macOS .dmg installer for qail.
#
# Flow:
#   1. Run `make app` to produce build/bin/qail.app via Wails.
#   2. Stage qail.app + an "Applications" symlink into a temp dir so the
#      DMG window shows the standard drag-to-Applications layout.
#   3. Run hdiutil to compress the staging dir into a UDZO .dmg.
#
# Output: build/installers/qail-<version>-<arch>.dmg
#
# Optional dependencies:
#   - create-dmg (brew install create-dmg) for a prettier window with
#     icon positioning + background image. Detected at runtime; falls
#     back to plain hdiutil if absent.
#
# Codesigning: not done here. For distribution, set DEVELOPER_ID to
# your "Developer ID Application" identity and the script will codesign
# qail.app before packaging. Notarization is a separate step (notarytool).

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
ARCH="${ARCH:-$(uname -m)}"
case "$ARCH" in
  x86_64) ARCH_LABEL="x64" ;;
  arm64)  ARCH_LABEL="arm64" ;;
  *)      ARCH_LABEL="$ARCH" ;;
esac

OUT_DIR="$ROOT/build/installers"
APP_PATH="$ROOT/build/bin/qail.app"
DMG_NAME="qail-${VERSION}-${ARCH_LABEL}.dmg"
DMG_PATH="$OUT_DIR/$DMG_NAME"

echo "==> building qail.app via wails (arch=$ARCH)"
make app

if [[ ! -d "$APP_PATH" ]]; then
  echo "error: $APP_PATH not produced by wails build" >&2
  exit 1
fi

if [[ -n "${DEVELOPER_ID:-}" ]]; then
  echo "==> codesigning with $DEVELOPER_ID"
  codesign --force --deep --options runtime \
    --sign "$DEVELOPER_ID" "$APP_PATH"
fi

mkdir -p "$OUT_DIR"
rm -f "$DMG_PATH"

STAGE="$(mktemp -d -t qail-dmg)"
trap 'rm -rf "$STAGE"' EXIT

echo "==> staging DMG contents in $STAGE"
cp -R "$APP_PATH" "$STAGE/"
ln -s /Applications "$STAGE/Applications"

if command -v create-dmg >/dev/null 2>&1; then
  echo "==> packaging with create-dmg"
  create-dmg \
    --volname "qail $VERSION" \
    --window-pos 200 120 \
    --window-size 600 380 \
    --icon-size 100 \
    --icon "qail.app" 150 190 \
    --hide-extension "qail.app" \
    --app-drop-link 450 190 \
    --no-internet-enable \
    "$DMG_PATH" \
    "$STAGE/"
else
  echo "==> create-dmg not installed; using hdiutil (plain layout)"
  hdiutil create \
    -volname "qail $VERSION" \
    -srcfolder "$STAGE" \
    -ov \
    -format UDZO \
    "$DMG_PATH"
fi

echo
echo "✓ built $DMG_PATH"
ls -lh "$DMG_PATH"
