# qail

`qail` (pronounced *kyle*) is a workspace manager for multi-repo projects. CLI, HTTP API, and macOS/Windows/Linux desktop app all share the same engine.

Group git repos into a workspace, clone them together, open the whole set in your editor or a tmux session. Run post-install scripts per repo or per workspace.

## Table of Contents

- [Why?](#why)
- [Three ways to run qail](#three-ways-to-run-qail)
- [Build](#build)
  - [CLI](#cli)
  - [HTTP server](#http-server)
  - [Desktop app](#desktop-app)
  - [Installers](#installers)
- [Run](#run)
- [Quick start](#quick-start)
- [Example: microservice workspace](#example-microservice-workspace)
- [Commands](#commands)
- [HTTP API](#http-api)
- [Agent Console](#agent-console)
- [Data location](#data-location)
- [Architecture](#architecture)
- [Development](#development)

## Why?

When working on microservices or any feature touching multiple repos, you want them cloned side-by-side, opened together, and bootstrapped the same way each time. `qail` encapsulates that as a *workspace*.

## Three ways to run qail

| Frontend | Command | Use when |
|---|---|---|
| **CLI** | `qail <subcommand>` | Default; scripts, terminal-first workflows |
| **HTTP API** | `qail serve` | Drive qail from a web UI, scripts, or another machine |
| **Desktop app** | `make app`, or the installed qail app | Electron menubar window; shares state with CLI/server |

All three call into the same `internal/actions` package — changes made in one are immediately visible to the others (single SQLite DB at `~/.qail/qail.db`).

## Build

### Prerequisites

| Tool | Needed for |
|---|---|
| Go ≥ 1.24 | All builds |
| Node ≥ 22 + npm | Desktop app + installers (Vite + Electron) |
| C toolchain (Xcode CLI tools, gcc, or MinGW) | cgo SQLite driver |

### CLI

Fastest path — no frontend toolchain needed:

```sh
make build                  # → bin/qail
mkdir -p ~/.qail/bin
cp bin/qail ~/.qail/bin
```

Add to `.bashrc` / `.zshrc`:

```sh
export PATH="$HOME/.qail/bin:$PATH"
```

### HTTP server

The HTTP server is built into the same `qail` binary — no extra build step.

```sh
qail serve --addr 127.0.0.1:8765
```

See [HTTP API](#http-api) for endpoints.

### Desktop app

The desktop app is an Electron shell (`frontend/electron/`) around the React UI. It spawns the `qail` binary as `qail desktop-backend` and talks to it over stdin/stdout, so there's no port to open and the Go code is the same binary as the CLI.

```sh
make app                    # build bin/qail + frontend/dist, launch Electron
make app-dev                # Vite HMR inside Electron (run `make build` after Go changes)
```

The window lives under a tray icon: left-click toggles it, right-click opens the menu (on Linux the menu is the only entry point), and `Ctrl+Alt+Q` toggles it from anywhere. Set `QAIL_BIN` to run Electron against a different `qail` binary.

### Tray app (experimental)

`qail tray` is the Electron-free version of the desktop app: a tray icon (the menu bar on macOS) whose UI opens in your default browser, served by the `qail` binary itself on `127.0.0.1:4318`. Left-click opens the UI; right-click shows **Open qail**, the **Agent Console** submenu and **Quit**. Running `qail tray` again while one is running just opens the UI. See [ADR-0021](docs/adr/0021-tray-app-browser-ui.md).

```sh
make tray                   # build the frontend into bin/qail, run `qail tray --open`
qail tray --port 5000       # another port (loopback only)
```

The UI is embedded at build time, so run `make frontend` before `make build`; a binary built without it serves a page saying so. On Linux the tray needs a StatusNotifierItem host (KDE, XFCE, Cinnamon; GNOME needs the AppIndicator extension), and **Browse** for the workspace root needs `zenity` installed (you can always type the path). Without a tray host the UI is still served at the URL above.

### Installers

electron-builder packages the frontend plus `bin/qail` (shipped as `resources/bin/qail`). The Go binary links SQLite via cgo, so build the macOS and Linux installers on their target OS. The Windows installer can also be built on macOS or Linux: `make installer-windows` cross-compiles `bin/qail.exe`, which needs a MinGW C compiler (`brew install mingw-w64` on macOS).

```sh
make installer              # native installer(s) for the host OS
make installer-mac          # → .dmg
make installer-windows      # → NSIS .exe (x64)
make installer-linux        # → .deb + .AppImage
```

Output: `build/installers/`. Config lives under `"build"` in `frontend/package.json`.

On macOS the `.dmg` also gives you the CLI: on first launch from Applications the app offers to install the `qail` command, and the tray menu has **Install Command Line Tool…** for later. It links `/usr/local/bin/qail` to the binary inside the app (asking for an admin password if that folder isn't writable), so the CLI and the app always run the same version.

On Windows the installer adds the app's `resources\bin` folder to your user `PATH`, so `qail` works in any new terminal; uninstalling removes it again (`build/installer.nsh`).

## Run

| Action | Command |
|---|---|
| Initialise | `qail init` |
| CLI commands | `qail <subcommand>` (see [Commands](#commands)) |
| HTTP server (loopback) | `qail serve` |
| HTTP server (custom port) | `qail serve --addr 127.0.0.1:9000` |
| Desktop app | `make app` — or installed via DMG/AppImage/etc. |
| Desktop app (from source, see logs) | `make app` prints Electron and backend logs to the terminal |
| Sandbox install | `QAIL_HOME=/tmp/qail qail <cmd>` |

## Quick start

```sh
qail init                              # set workspace root dir
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
qail repo add                          # register a git repo
qail workspace add                     # create a workspace from registered repos
qail open                              # open workspace in editor
```

## Example: microservice workspace

Say you work on a payments product split across `payments-api`, `payments-worker`, `payments-web`. One command to clone all three and run `npm install` in the web repo.

```sh
# 1. one-time setup
qail init
qail config editor "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"

# 2. register each repo
qail r a   # → git@github.com:acme/payments-api.git
qail r a   # → git@github.com:acme/payments-worker.git
qail r a   # → git@github.com:acme/payments-web.git

# 3. create a post-install script for the web repo
qail scripts a    # name: web-install
qail scripts o    # edit body to: npm install

# 4. attach script to the web repo
qail repo p       # pick payments-web → web-install

# 5. create the workspace
qail workspace add
# name: payments; select all 3 repos

# 6. open it
qail open         # editor launches with all 3 repos
qail mux o        # or tmux session w/ a window per repo
```

Reopen later:

```sh
qail ws ls
qail open
qail ws cd                  # print path; pair with shell alias to cd into it
```

Tear down a stale clone, keep the definition:

```sh
qail ws clean
```

## Commands

Aliases: `workspace` → `ws`, `repo` → `r`, `add` → `a`, `list` → `ls`, `remove` → `rm`, `open` → `o`, `create` → `c`.

### Config

```sh
qail init                  # set workspace root (interactive)
qail config ls             # show current config
qail config editor <path>  # set editor binary
qail config root <path>    # set workspace root
```

### Repos

```sh
qail repo add              # register a git repo (alias: r a)
qail repo list
qail repo remove
qail repo p                # attach a post-install script
```

### Workspaces

```sh
qail workspace add
qail workspace list
qail workspace edit
qail workspace remove
qail workspace clone       # re-clone repos
qail workspace clean       # remove orphaned dirs
qail workspace explore     # browse workspace
qail workspace cd          # print path
qail open                  # open in editor
qail mux o                 # open in tmux
```

### Scripts

Bash scripts in `~/.qail/scripts/`, runnable as repo or workspace post-install hooks.

```sh
qail scripts add
qail scripts open
qail scripts list
qail scripts remove
```

Default template:

```sh
#!/bin/bash
# Add your custom logic here
ls -l
```

### Tmux

```sh
qail mux o                 # open workspace as tmux session
qail mux ls
qail mux rm
```

Requires `tmux` installed.

### Serve

```sh
qail serve                 # default 127.0.0.1:8765
qail serve --addr :9000    # custom bind
```

## HTTP API

`qail serve` exposes the same actions as JSON+SSE. Long-running ops (workspace create/edit/clone) stream progress as Server-Sent Events; everything else is plain JSON.

```
GET    /api/health
GET    /api/config
PUT    /api/config/{root,editor}        {value}

GET    /api/repos
POST   /api/repos                       {name, url}
DELETE /api/repos                       {names: []}
PUT    /api/repos/{name}/post-install   {scripts: []}

GET    /api/workspaces
POST   /api/workspaces                  {name, packages}        ← SSE
PUT    /api/workspaces/{name}           {packages}              ← SSE
DELETE /api/workspaces/{name}
POST   /api/workspaces/{name}/clone     {dst, packages}         ← SSE
POST   /api/workspaces/{name}/create                            ← SSE
PUT    /api/workspaces/{name}/post-install {scripts: []}
GET    /api/workspaces/{name}/path
GET    /api/workspaces/{name}/mux
GET    /api/workspaces/{name}/open-cmd

GET    /api/scripts
POST   /api/scripts                     {name}
DELETE /api/scripts/{name}
GET    /api/scripts/path

GET    /api/mux/sessions
DELETE /api/mux/sessions/{name}
```

See [docs/adr/0016-http-api-layer.md](docs/adr/0016-http-api-layer.md) for the design.

Example:

```sh
qail serve &
curl -s http://127.0.0.1:8765/api/workspaces | jq
curl -X POST -d '{"name":"svc","url":"git@host:foo.git"}' \
  -H 'content-type: application/json' \
  http://127.0.0.1:8765/api/repos
```

## Agent Console

A browser UI for Claude Code agents, organised by qail workspace and repo. Pick a workspace to run agents across all its repos, or a repo to run them inside it with that repo's own `CLAUDE.md`, agents, skills and hooks. See [ADR-0020](docs/adr/0020-agent-console-in-binary.md).

The console is built into the `qail` binary: no Node, nothing extra to install. Agents run on your installed [Claude Code](https://docs.anthropic.com/en/docs/claude-code) CLI (`claude`), so they use the same version, login, settings, agents and skills as in your terminal. Install Claude Code first; the console says so if it can't find it.

Open it:

| From | How |
|---|---|
| Menubar app | Tray menu → **Agent Console** → *Open*, or a workspace → *Whole workspace* / a repo |
| CLI | `qail console` · `qail console payments` · `qail console payments payments-api` |

The console runs on `127.0.0.1:4317`, served by the qail process that opened it (the menubar app, or `qail console` in a terminal). Auth is whatever your `claude` uses: your Claude Code login or `ANTHROPIC_API_KEY`.

| Env | Purpose |
|---|---|
| `QAIL_CLAUDE` | Claude Code CLI to use instead of the one found on PATH or in the usual install locations |

## Data location

Everything lives under `~/.qail/` (override with `QAIL_HOME`):

- `qail.db` — SQLite database (config, repos, workspaces)
- `scripts/` — user scripts
- `console/data/` — Agent Console chat threads
- `bin/` — the binary (if installed via the steps above)

## Architecture

```
              ┌─────────────────┐
   terminal ─►│ cmd/* Cobra CLI │─┐
              └─────────────────┘ │
                                  │
              ┌─────────────────┐ │
   browser ──►│ cmd/serve.go    │─┤
              │ internal/api/   │ │
              │ (HTTP + SSE)    │ │
              └─────────────────┘ ├─►  internal/actions/  ──► internal/{workspace,installer,git,scripts,tmux}
                                  │   (atomic ops, mutex)        │
              ┌─────────────────┐ │                              ▼
   menubar ──►│ cmd/desktop.go  │─┘    internal/config/Store ──► ~/.qail/qail.db
  (Electron,  │ internal/app/   │                                (SQLite)
   stdio RPC) │ (Bindings)      │
              └─────────────────┘
```

Three frontends, one engine. ADRs in [docs/adr/](docs/adr/) record the architecture decisions; [docs/adr/0016-http-api-layer.md](docs/adr/0016-http-api-layer.md) covers the HTTP layer refactor and [docs/adr/0017-electron-desktop-app.md](docs/adr/0017-electron-desktop-app.md) the move from Wails to Electron.

## Development

```sh
make test                  # all Go tests
go vet ./cmd/... ./internal/... .

# desktop app dev (hot reload)
make app-dev

# build a one-shot CLI binary
go build -o bin/qail .
```

Companion docs:

- [app-status.md](app-status.md) — desktop app build state + deferred tray work
- [frontend.md](frontend.md) — web/desktop UI design
- [docs/adr/](docs/adr/) — architecture decisions
