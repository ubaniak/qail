# ADR-0005: `qailhome` is the single source of truth for `~/.qail/` layout

**Status**: Accepted (commit `d8ec057`)

## Context

Multiple modules independently computed paths under `~/.qail/`: `config` for the SQLite DB, `scripts` for the scripts directory, `convert` for the legacy JSON. Each re-implemented the `$QAIL_HOME` env override and `os.UserHomeDir()` fallback, plus the `MkdirAll` discipline. Adding a new well-known path (logs, backups, cache) would have required touching every consumer.

## Decision

`internal/qailhome` owns the layout. `Home` struct exposes path accessors (`DBPath`, `ScriptsDir`, …). `Default()` resolves once via `sync.Once`: reads `$QAIL_HOME`, falls back to `~/.qail/`, creates required directories. `New(root)` is for tests — no env reads, no mkdirs.

Consumers (`config.defaultStore`, `scripts.Default`) call `qailhome.Default()` and read accessors. They do not compute paths themselves.

## Consequences

- One place to add a new well-known path.
- One place to change the `$QAIL_HOME` resolution rules.
- Lazy idempotent init — order of `Default()` calls across packages does not matter.
- Tests that need a clean root use `qailhome.New(t.TempDir())` and inject downstream.

## Alternatives considered

- *Pass paths via flags / env vars at every call site*: rejected — bloats interfaces with paths that never vary at runtime.
- *Initialize from `main.go` and inject everywhere*: rejected — adds a wiring chore for what is logically a constant. Lazy-once gives the same guarantees.
