#!/usr/bin/env bash
# Build a Debian .deb package for qail.
#
# Uses nfpm (https://github.com/goreleaser/nfpm) — single Go binary that
# produces deb/rpm/apk from a YAML descriptor. Avoids dpkg-deb directly
# so the same script works from non-Debian hosts (CI runners, macOS).
#
# Requires: nfpm on PATH. Install: `go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest`
#
# Output: build/installers/qail_<version>_amd64.deb

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v nfpm >/dev/null 2>&1; then
  echo "error: nfpm not installed. Run: go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest" >&2
  exit 1
fi

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)}"
# nfpm requires semver-ish versions; strip leading 'v' if present
VERSION="${VERSION#v}"

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

# Stage desktop entry + icon so the installed app appears in the
# application launcher (GNOME Activities, KDE menu, etc.).
STAGE="$(mktemp -d -t qail-deb)"
trap 'rm -rf "$STAGE"' EXIT

cat > "$STAGE/qail.desktop" <<EOF
[Desktop Entry]
Name=qail
GenericName=Workspace Manager
Comment=Multi-repo workspace manager
Exec=qail app
Icon=qail
Type=Application
Categories=Development;
Terminal=false
EOF

# Reuse the menubar icon as a desktop icon for now; a richer 256×256
# PNG would be ideal but the template icon works as a placeholder.
cp "$ROOT/internal/app/icon.png" "$STAGE/qail.png"

cat > "$STAGE/nfpm.yaml" <<EOF
name: qail
arch: amd64
platform: linux
version: ${VERSION}
section: utils
priority: optional
maintainer: qail <noreply@example.com>
description: |
  qail is a CLI + desktop workspace manager for multi-repo projects.
  Groups git repos into workspaces, clones them together, opens them
  in editors or tmux sessions.
vendor: qail
homepage: https://github.com/ubaniak/qail
license: MIT
contents:
  - src: ${BIN}
    dst: /usr/bin/qail
  - src: ${STAGE}/qail.desktop
    dst: /usr/share/applications/qail.desktop
  - src: ${STAGE}/qail.png
    dst: /usr/share/icons/hicolor/256x256/apps/qail.png
EOF

mkdir -p "$ROOT/build/installers"
OUT="$ROOT/build/installers/qail_${VERSION}_amd64.deb"
nfpm pkg --packager deb --config "$STAGE/nfpm.yaml" --target "$OUT"

echo
echo "✓ built $OUT"
ls -lh "$OUT"
