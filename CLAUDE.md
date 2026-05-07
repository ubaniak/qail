# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`qail` (pronounced "kyle") is a CLI workspace manager for multi-repo/microservice projects. It groups git repos into workspaces, clones them together, and supports opening them in editors or tmux sessions. Runs post-install scripts per-repo or per-workspace.

## Build & Run

```sh
make build          # outputs bin/qail
go build -o bin/qail main.go   # same thing
```

No test suite exists. No linter configured.

## Architecture

Go CLI app using **Cobra** for commands and **GORM + SQLite** for persistence (`~/.qail/qail.db`). Recently migrated from JSON config (`config.json`) to SQLite; `internal/config/convert.go` handles the migration.

### Package Layout

- `cmd/` — Cobra command definitions. Each file maps to a top-level command (`repo`, `workspace`, `config`, `scripts`, `tmux`, `init`). Commands use `HandleConfig` or `config.WithConfig` to read/modify/write config within a callback.
- `internal/config/` — SQLite-backed config layer. `Config` struct is the central data model (root path, editor, workspaces, repos, post-install scripts). `ReadFromFile()`/`WriteToFile()` serialize entire config to/from DB. `WithConfig()` wraps read-modify-write in one call.
- `internal/forms/` — Interactive TUI forms using **charmbracelet/huh**. One file per domain (repo, workspace, scripts, tmux, config, init).
- `internal/workspace/` — Workspace operations: create (clone repos + run scripts), open in editor, cd, tmux, clean orphaned dirs.
- `internal/git/` — Git clone with progress spinner.
- `internal/tmux/` — Tmux session management (launch, attach, kill).
- `internal/scripts/` — Manages bash scripts stored in `~/.qail/scripts/`.
- `internal/clip/` — Clipboard helper for `cd` (copies path).
- `internal/color/` — Lipgloss color helpers.

### Data Flow Pattern

Commands follow: Cobra handler → `WithConfig(func(cfg *Config) error)` → read DB → modify `Config` struct → write DB. The entire config is replaced on each write (delete-all + re-insert in a transaction).

### Key Command Aliases

| Command | Alias |
|---------|-------|
| `workspace` | `ws` |
| `repo` | `r` |
| `add` | `a` |
| `list` | `ls` |
| `remove` | `rm` |
| `open` | `o` |
| `create` | `c` |

### User Data Location

All user data lives in `~/.qail/` — the SQLite DB (`qail.db`), scripts dir, and binary.
