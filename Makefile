# CLI build — produces bin/qail. The same binary is the desktop app's
# backend, so this stays fast and never needs the frontend.
EXE := $(if $(filter Windows_NT,$(OS)),.exe,)

build:
	go build -o bin/qail$(EXE) .

# Frontend build — emits frontend/dist/, which Electron loads.
frontend:
	cd frontend && npm install --silent && npm run build

# Desktop app, unpacked — builds the Go backend (bin/qail) and the
# frontend, then launches Electron against them. Electron spawns
# `bin/qail desktop-backend` and talks to it over stdin/stdout.
app: build frontend
	cd frontend && npm run electron

# Dev loop — Vite dev server with HMR inside Electron. Rebuild the Go
# side with `make build` and relaunch to pick up backend changes.
app-dev: build
	cd frontend && npm install --silent && npm run electron:dev

# Run every test. node_modules ships a Go reference impl for `flatted`
# that we don't care about; explicit package list skips it.
test:
	go test ./cmd/... ./internal/... .

# Installers — electron-builder packages the frontend, bin/qail (as
# resources/bin/qail) and the Agent Console bundle (as resources/console,
# see console-bundle) into build/installers/. The Go binary links SQLite
# via cgo, so build mac and linux installers on their target OS; Windows
# can also be cross-built (see build-windows).
installer-mac: build frontend
	$(MAKE) console-bundle CONSOLE_PLATFORM=darwin
	cd frontend && npx electron-builder --mac

# Apple silicon dmg from either Mac. Overwrites bin/qail with an arm64
# build; run `make build` afterwards to get a host binary back.
installer-mac-arm64: frontend
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o bin/qail .
	$(MAKE) console-bundle CONSOLE_PLATFORM=darwin CONSOLE_ARCH=arm64
	cd frontend && npx electron-builder --mac --arm64

# The Windows installer needs bin/qail.exe. On Windows that's plain
# `make build`; anywhere else it cross-compiles, and cgo then needs a
# MinGW C compiler (macOS: brew install mingw-w64).
build-windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(if $(filter Windows_NT,$(OS)),,CC=x86_64-w64-mingw32-gcc) go build -o bin/qail.exe .

installer-windows: build-windows frontend
	$(MAKE) console-bundle CONSOLE_PLATFORM=win32 CONSOLE_ARCH=x64
	cd frontend && npx electron-builder --win --x64

installer-linux: build frontend
	$(MAKE) console-bundle CONSOLE_PLATFORM=linux
	cd frontend && npx electron-builder --linux

# Convenience: native installer for the current host OS.
installer: build frontend console-bundle
	cd frontend && npx electron-builder

.PHONY: build build-windows frontend app app-dev test console-install console-dev console-bundle installer installer-mac installer-mac-arm64 installer-windows installer-linux

# --- Agent Console (ADR-0018, ADR-0019) ---------------------------------------
# Stages the console for one installer target into build/console/<os>-<arch>:
# sources, production node_modules with the Claude Agent SDK's native CLI
# for that target, and a pinned Node runtime. Needs node + npm on the build
# host; defaults to the host platform/arch, and cross-targets work anywhere.
CONSOLE_PLATFORM ?=
CONSOLE_ARCH ?=
console-bundle:
	node scripts/bundle-console.mjs $(if $(CONSOLE_PLATFORM),--platform $(CONSOLE_PLATFORM)) $(if $(CONSOLE_ARCH),--arch $(CONSOLE_ARCH))

# Source install for development without the app: the Node sidecar goes
# into ~/.qail/console (or $QAIL_HOME/console) and runs on the system Node.
# The installers don't need this. Chat threads in data/ survive reinstalls.
QAIL_HOME ?= $(HOME)/.qail
CONSOLE_DIR ?= $(QAIL_HOME)/console

console-install:
	mkdir -p $(CONSOLE_DIR)
	rsync -a --delete --exclude node_modules --exclude data console/ $(CONSOLE_DIR)/
	cd $(CONSOLE_DIR) && npm ci --omit=dev
	@echo "Agent Console installed in $(CONSOLE_DIR) — run 'qail console' or use the tray menu."

console-dev:
	cd console && npm install && QAIL_AUTOSTART=0 node server.mjs
