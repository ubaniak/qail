# ADR-0021: `qail tray` — Go systray icon with the UI in the browser

**Status**: Accepted (phase 1: runs alongside Electron; installers unchanged).

## Context

The desktop app (ADR-0017) is an Electron shell: about 600 lines of
`frontend/electron/` that spawn `qail desktop-backend`, bridge the React
UI's calls to it over stdio, and own a tray icon, a frameless popover
window, a folder dialog and the macOS CLI-install prompt. Installers carry
Chromium and Node for that, roughly 100 MB each, and electron-builder plus
Electron's security fuses are their own upkeep.

The Agent Console already runs inside the qail binary and opens in the
browser (ADR-0020). The UI talks to Go through one generic call,
`call(method, args)`, plus named progress events, so it doesn't depend on
Electron beyond that bridge.

## Decision

1. **`qail tray`** puts an icon in the system tray with
   [fyne.io/systray](https://github.com/fyne-io/systray) (the maintained
   fork of getlantern/systray): macOS via cgo (already required for
   SQLite), Windows natively, Linux via StatusNotifierItem over D-Bus (pure
   Go). Left-click opens the UI in the default browser; right-click shows
   Open qail, the Agent Console submenu (same items as Electron's) and
   Quit. The submenu is rebuilt on Refresh, after console actions, and
   every 30 s when workspaces changed.
2. **The same Bindings over HTTP** (`internal/app/http.go`):
   `POST /api/call` dispatches by method name through the same code as the
   stdio bridge, `GET /api/events` streams progress events (SSE), and
   `POST /api/pick-directory` shows a native folder dialog
   ([ncruces/zenity](https://github.com/ncruces/zenity)).
3. **The frontend is embedded** (`internal/ui`, `go:embed`). `make
   frontend` copies `frontend/dist` in; a plain `make build` still works
   and embeds a placeholder.
4. **One UI, two transports.** `frontend/src/desktop.ts` uses Electron's
   `window.qail` bridge when present and HTTP otherwise. The Electron-only
   close button is hidden in a browser tab; clipboard uses
   `navigator.clipboard` (127.0.0.1 is a secure context).
5. **Security: loopback, no token.** The server listens on 127.0.0.1 only
   (default port 4318, next to the console's 4317). It refuses non-loopback
   `Host` headers (DNS rebinding), cross-origin API calls (`Origin`,
   `Sec-Fetch-Site`), and API POSTs that aren't `application/json`, which
   web pages can't send cross-origin without a CORS preflight. A second
   `qail tray` finds the first via `/api/health` and just opens the UI.
6. **Windows**: when started from a shortcut rather than a terminal, `qail
   tray` frees the console window Windows gives it.

## Consequences

- Other programs running as the same user can call the API without a
  token. They could already run `qail` (or anything else) as that user;
  on a shared multi-user machine, another local account could reach the
  port too. A per-launch token in the tray's URL is the fix if that
  matters later.
- Lost against Electron: the Ctrl+Alt+Q hotkey and the popover that hides
  on blur. Each "Open qail" opens a new browser tab.
- Linux needs a tray host (KDE, XFCE, Cinnamon; GNOME with the
  AppIndicator extension); without one the UI is still served. The folder
  dialog needs `zenity` (or qarma/matedialog); the path can be typed.
- On Windows the console window may flash briefly at launch until the
  installer ships a GUI-subsystem build.

## Phase 2 (not done)

Replace electron-builder: a hand-built `.app` with `LSUIElement` and
`.dmg` (Intel and arm64), NSIS for Windows (keeping the PATH step, with a
GUI-subsystem tray exe next to the console `qail.exe`), and a deb (plus
optional AppImage). Port the macOS "Install Command Line Tool" prompt, do
a layout pass for full-width tabs, then delete `frontend/electron/`.

## Alternatives considered

- **getlantern/systray** (used by the systray-structure skeleton):
  unmaintained since 2022; the fyne fork has the same API, `ResetMenu`
  for dynamic menus, and D-Bus trays without GTK on Linux.
- **A token in the URL plus an HttpOnly cookie**: stronger against other
  local processes, but the user preferred loopback-only without auth.
- **Point the UI at the existing `internal/api` (`qail serve`)**: it
  covers a subset of Bindings with different shapes; reusing the Bindings
  dispatcher keeps one method set for both desktop transports.
