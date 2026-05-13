# CLI build — produces bin/qail. Works without the frontend dist (the
# `qail app` subcommand refuses to launch without it), so this stays fast.
build:
	go build -o bin/qail .

# Frontend build — emits frontend/dist/, the embed target of assets_app.go.
frontend:
	cd frontend && npm install --silent && npm run build

# Regenerate wails3 typed bindings under frontend/bindings/. Run after
# editing internal/app/bindings.go method signatures.
bindings:
	wails3 generate bindings -d frontend/bindings

# Desktop app build — wails v3 doesn't ship a single all-in-one
# `wails build` like v2 did; we assemble the bundle by hand:
#   1. vite build  → frontend/dist
#   2. go build -tags production → build/bin/qail (embeds dist + icon)
#   3. cp into a .app bundle with build/darwin/Info.plist
#   4. ad-hoc codesign so the OS will run it without quarantine fuss
# Output: build/bin/qail.app
app: frontend
	mkdir -p build/bin/qail.app/Contents/MacOS build/bin/qail.app/Contents/Resources
	CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=10.15 \
		go build -tags production -trimpath -buildvcs=false \
			-ldflags="-w -s" \
			-o build/bin/qail.app/Contents/MacOS/qail .
	cp build/darwin/Info.plist build/bin/qail.app/Contents/Info.plist
	@if [ -f build/darwin/iconfile.icns ]; then \
		cp build/darwin/iconfile.icns build/bin/qail.app/Contents/Resources/iconfile.icns; \
	else \
		echo "(no build/darwin/iconfile.icns — bundle will show a generic icon)"; \
	fi
	codesign --force --deep --sign - build/bin/qail.app

# Dev loop — wails3 dev with hot reload. Falls back to running the prod
# binary directly if no wails3 config is wired up yet.
app-dev:
	cd frontend && npm install --silent
	wails3 dev || (echo "wails3 dev failed; falling back to make app && open"; $(MAKE) app && open build/bin/qail.app)

# Run every test. node_modules ships a Go reference impl for `flatted`
# that we don't care about; explicit package list skips it.
test:
	go test ./cmd/... ./internal/... .

# Installers — each targets one OS and writes to build/installers/.
# CI workflow .github/workflows/release.yml invokes the same scripts on
# native runners so cross-compile pain is avoided.
installer-mac:
	./scripts/build-mac-dmg.sh

installer-windows:
	./scripts/build-windows-installer.sh

installer-linux-deb:
	./scripts/build-linux-deb.sh

installer-linux-appimage:
	./scripts/build-linux-appimage.sh

# Convenience: native installer for the current host OS.
installer:
ifeq ($(shell uname),Darwin)
	$(MAKE) installer-mac
else ifeq ($(shell uname),Linux)
	$(MAKE) installer-linux-deb
	$(MAKE) installer-linux-appimage
else
	$(MAKE) installer-windows
endif

.PHONY: build frontend bindings app app-dev test installer installer-mac installer-windows installer-linux-deb installer-linux-appimage
