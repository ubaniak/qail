# ADR-0017: Electron desktop app over a stdio RPC backend

**Status**: Accepted

## Context

The desktop app was a Wails v3 (alpha) menubar app: `qail app` embedded
`frontend/dist` into the Go binary, bound `internal/app.Bindings` to the
webview through generated JS bindings, and drew the tray and the global
hotkey (a cgo Carbon shim on macOS) from Go. Wails v3 is still alpha,
its bindings need regenerating on every signature change, and the
Linux/Windows stories required separate toolchains (WebKitGTK, NSIS via
`wails build`). We want the frontend to be an Electron app instead.

## Decision

- **Electron owns the UI shell.** `frontend/electron/main.cjs` draws the
  tray icon, the frameless 480×640 window anchored under it (hide on
  close and on blur), the `Ctrl+Alt+Q` global shortcut, the clipboard
  and the directory picker. The React UI is unchanged apart from the
  four Wails runtime calls, which now go through `src/desktop.ts`.
- **The Go binary is the backend, spoken to over stdio.** Electron spawns
  `qail desktop-backend` (hidden command) and exchanges newline-delimited
  JSON on its stdin/stdout: `{id, method, args}` requests, `{id, result |
  error}` responses, and `{event, data}` progress events. `internal/app`
  dispatches requests to `Bindings` methods by name via reflection, so
  the `Bindings` method set is the API, as it was under Wails.
- **The renderer never touches Node or the backend directly.** A preload
  script exposes `window.qail` (`call`, `onEvent`, `copyText`,
  `pickDirectory`, `hideWindow`); the main process validates the sender
  and forwards to the backend.
- **Packaging is electron-builder.** `bin/qail` ships as
  `resources/bin/qail`; dmg / NSIS / deb / AppImage targets replace the
  hand-rolled scripts in `scripts/`.

## Consequences

**Positive**

- No Wails, no generated bindings, no cgo hotkey shim; `go.mod` drops the
  whole Wails dependency tree.
- No listening socket: nothing else on the machine (or a web page
  hitting localhost) can reach the desktop backend, so no auth token is
  needed.
- The packaged app's backend is the full `qail` CLI binary.

**Negative**

- Electron bundles Chromium: the installed app is ~100 MB larger than
  the Wails build.
- `desktop-backend` reserves stdout for the protocol. It reassigns
  `os.Stdout` to stderr so stray prints can't corrupt frames, but a
  subprocess that writes to fd 1 it inherited some other way would.
- Reflection dispatch means a renamed Go method only fails at call time;
  `frontend/src/api/bindings.ts` must be kept in sync by hand.

## Alternatives considered

- **Talk to `qail serve` (ADR-0016) over HTTP.** The REST surface lacks
  roughly half of what the desktop UI calls (orphans, launch editor/AI,
  script read/write/run), and a loopback port needs a token to keep
  browsers and other local users out. Extending it is still the right
  path for a browser UI; the desktop app doesn't need it.
- **Spawn the backend per call.** Simple, but loses progress streaming
  for in-flight clones and pays process start-up on every list refresh.
