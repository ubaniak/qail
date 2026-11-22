# ADR-0018: Agent Console as a supervised Node sidecar

**Status**: Superseded by ADR-0020 (the console now runs inside the qail binary). Its scope model, UI and tray menu still stand.

## Context

We want a browser UI for Claude Code agents organised the way qail already
organises code: pick a **workspace** to run agents at `<Root>/<workspace>`
across every package, or a **repo** to run them inside
`<Root>/<workspace>/<repo>` with that repo's own `CLAUDE.md`, `.claude/agents`,
skills and hooks. It should open from the menubar app ("Agent Console"
submenu) and from the CLI.

The agent runtime is the Claude Agent SDK. It ships for TypeScript and
Python only — there is no Go SDK — and it is what gives the UI exact parity
with Claude Code (named agents, preloaded skills, settings, hooks, session
resume, permission callbacks).

## Decision

1. **The console is a Node sidecar** living in `console/` (`server.mjs` +
   static `public/`), installed to `qailhome.ConsoleDir()`
   (`~/.qail/console`) by `make console-install`.
2. **qail stays the source of truth.** The console reads workspaces, repos
   and root from the existing HTTP API (ADR-0016) — never from SQLite — and
   only writes through it (`POST /api/workspaces/{name}/create` for
   "Clone missing repos"). It stores nothing but its own chat threads.
3. **`internal/console` supervises it**: locate node (PATH, Homebrew, Volta,
   nvm — the menubar app doesn't inherit the shell's PATH), start
   `node server.mjs` with `PORT`/`QAIL_URL`/`QAIL_AUTOSTART`, wait for
   health, stop on quit, open the browser through the `runner` seam
   (ADR-0003). A console already answering on the port is reused, not
   duplicated.
4. **Menubar app**: the Electron app's Go backend (`qail desktop-backend`,
   ADR-0017) serves `internal/api` in-process on `127.0.0.1:8765`
   (reusing an existing `qail serve` if one owns the port) and owns the
   Manager; it stops the sidecar when Electron closes its stdin. The
   Electron tray adds an *Agent Console* submenu, built from the
   `ConsoleStatus` binding and driving `OpenConsole` / `StopConsole`: Open, one submenu per workspace (most recently
   used first) with *Whole workspace* + each repo, Refresh, Stop. Items
   deep-link with `#/ws/<workspace>[/<repo>]`.
5. **CLI**: `qail console [workspace] [repo]` — same Manager, with
   `QAIL_AUTOSTART=1` so the console can start `qail serve` itself.

## Consequences

- Node ≥ 22 becomes a runtime dependency **for the console only**; CLI,
  server and app work without it (the submenu shows "Not installed").
- Two loopback ports: 8765 (qail API) and 4317 (console). Both bind
  127.0.0.1 and have no auth; the console rejects cross-origin API calls.
- Installers don't bundle the console yet; `make console-install` is the
  install path until we decide whether to ship node_modules in the DMG.
- The menu lists workspaces at build time; it is rebuilt after every open,
  on *Refresh workspaces*, and on stop.

## Alternatives considered

- **Rewrite the chat backend in Go over `claude -p --output-format
  stream-json`.** No Node dependency, but re-implements session resume,
  permission prompts and skill/agent loading that the SDK already provides,
  and tracks a CLI wire format that isn't a stable API.
- **Put the console inside the desktop window.** The 480×640 menubar panel is
  the wrong shape for multi-pane agent chat, and the browser gives tabs per
  workspace for free.
- **Console reads `qail.db` directly.** Duplicates the Store and bypasses
  the actions mutex; rejected for the same reasons as ADR-0016.
