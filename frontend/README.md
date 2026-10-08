# qail desktop frontend

React + Vite UI, packaged as an Electron menubar app.

- `src/` — the React UI. `src/desktop.ts` is the only file that talks to
  Electron (via the `window.qail` bridge); `src/api/bindings.ts` types the
  Go methods it can call.
- `electron/main.cjs` — main process: tray, window, global shortcut, and
  IPC handlers.
- `electron/backend.cjs` — spawns `qail desktop-backend` and speaks its
  newline-delimited JSON protocol (see `internal/app/rpc.go`).
- `electron/preload.cjs` — exposes `window.qail` to the renderer.

From the repo root:

```sh
make app        # build bin/qail + dist/, launch Electron
make app-dev    # Vite HMR inside Electron
make installer  # electron-builder → build/installers/
```

`QAIL_BIN` overrides which `qail` binary Electron spawns.
