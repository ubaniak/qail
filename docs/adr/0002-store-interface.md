# ADR-0002: `config.Store` interface with SQLite and Memory adapters

**Status**: Accepted (commit `617b339`)

## Context

Persistence was previously called via package-level functions (`config.ReadFromFile`, `config.WriteToFile`). These talked directly to the JSON file (later, the SQLite file) and were untestable without touching disk. Tests for any module that touched config had to mock the filesystem or use a temp directory.

## Decision

Introduce a `Store` interface in `internal/config/store.go`:

```go
type Store interface {
    Read() (*Config, error)
    Write(cfg *Config) error
}
```

Two adapters:

- `SQLiteStore` (production): GORM-backed, opens `~/.qail/qail.db`.
- `MemoryStore` (tests): in-memory, supports cloning so tests can snapshot starting state.

Two adapters justify the seam (one adapter would be a hypothetical seam, two is a real one).

The package-level `ReadFromFile` / `WriteToFile` / `WithConfig` functions remain as deprecated shims that delegate to a lazily-initialized default `SQLiteStore`. New code should accept a `Store` parameter.

## Consequences

- `actions` package can be unit-tested without disk via `MemoryStore`.
- Future stores (e.g. an HTTP-backed store, a read-only export) plug in here without touching callers.
- Two write paths exist temporarily (`Store.Write` + the `WriteToFile` shim). The shim is a known migration target; remove once all handlers move to `actions`.

## Alternatives considered

- *Keep package-level functions, mock the file path*: rejected — leaks implementation, can't substitute persistence strategy.
- *Repository-per-entity interfaces (`RepoStore`, `WorkspaceStore`)*: rejected for now — the snapshot model means a single read/write covers all entities. Revisit if granular writes become necessary.
