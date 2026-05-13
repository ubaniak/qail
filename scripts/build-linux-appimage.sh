#!/usr/bin/env bash
# Build a portable AppImage for qail.
#
# AppImage = single-file Linux binary that runs on any distro without
# install. We use appimagetool from the AppImage project, which expects
# an AppDir laid out like a Linux install root.
#
# Requires: appimagetool on PATH. Get it via:
#   curl -L -o /usr/local/bin/appimagetool \
#     https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage
#   chmod +x /usr/local/bin/appimagetool
#
# Output: build/installers/qail-<version>-x86_64.AppImage

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v appimagetool >/dev/null 2>&1; then
  echo "error: appimagetool not on PATH. See script header for install." >&2
  exit 1
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"

echo "==> wails build -platform linux/amd64"
wails build \
  -platform linux/amd64 \
  -tags production \
  -skipbindings

BIN="$ROOT/build/bin/qail"
if [[ ! -f "$BIN" ]]; then
  echo "error: $BIN not produced" >&2
  exit 1
fi

APPDIR="$(mktemp -d -t qail-appimage)"
trap 'rm -rf "$APPDIR"' EXIT

mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" \
         "$APPDIR/usr/share/icons/hicolor/256x256/apps"

cp "$BIN" "$APPDIR/usr/bin/qail"
cp "$ROOT/internal/app/icon.png" "$APPDIR/qail.png"
cp "$ROOT/internal/app/icon.png" "$APPDIR/usr/share/icons/hicolor/256x256/apps/qail.png"

cat > "$APPDIR/qail.desktop" <<EOF
[Desktop Entry]
Name=qail
GenericName=Workspace Manager
Exec=qail app
Icon=qail
Type=Application
Categories=Development;
Terminal=false
EOF
cp "$APPDIR/qail.desktop" "$APPDIR/usr/share/applications/"

# AppRun is the AppImage entrypoint shim. It just execs the real binary
# with the user's args; no env setup needed for a static Go binary, but
# Wails apps need WebKitGTK at runtime — that's the user's distro's job.
cat > "$APPDIR/AppRun" <<'EOF'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
exec "$HERE/usr/bin/qail" "$@"
EOF
chmod +x "$APPDIR/AppRun"

mkdir -p "$ROOT/build/installers"
OUT="$ROOT/build/installers/qail-${VERSION}-x86_64.AppImage"
appimagetool "$APPDIR" "$OUT"

echo
echo "✓ built $OUT"
ls -lh "$OUT"
