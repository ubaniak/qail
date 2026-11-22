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

# Installers — electron-builder packages the frontend and bin/qail (as
# resources/bin/qail, which also serves the Agent Console) into
# build/installers/. The Go binary links SQLite
# via cgo, so build mac and linux installers on their target OS; Windows
# can also be cross-built (see build-windows).
installer-mac: build frontend
	cd frontend && npx electron-builder --mac

# Apple silicon dmg from either Mac. Overwrites bin/qail with an arm64
# build; run `make build` afterwards to get a host binary back.
installer-mac-arm64: frontend
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o bin/qail .
	cd frontend && npx electron-builder --mac --arm64

# The Windows installer needs bin/qail.exe. On Windows that's plain
# `make build`; anywhere else it cross-compiles, and cgo then needs a
# MinGW C compiler (macOS: brew install mingw-w64).
build-windows:
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 $(if $(filter Windows_NT,$(OS)),,CC=x86_64-w64-mingw32-gcc) go build -o bin/qail.exe .

installer-windows: build-windows frontend
	cd frontend && npx electron-builder --win --x64

installer-linux: build frontend
	cd frontend && npx electron-builder --linux

# Convenience: native installer for the current host OS.
installer: build frontend
	cd frontend && npx electron-builder

.PHONY: build build-windows frontend app app-dev test installer installer-mac installer-mac-arm64 installer-windows installer-linux
