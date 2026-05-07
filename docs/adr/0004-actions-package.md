# ADR-0004: `actions` package owns atomic command-level operations

**Status**: Accepted, migration in progress (commit `5808659`)

## Context

Cobra command handlers historically read the full `Config` via `HandleConfig(func(cfg *Config) error)`, mutated fields directly, and let the wrapper write back. This:

- exposed the entire `Config` struct to every handler (schema as the API),
- spread invariants across handlers (e.g. "removing a repo cascades to workspaces" was repeated, or worse, missed),
- made handlers integration-test-only — there was no module to unit-test that wrapped a single semantic operation.

## Decision

Introduce `internal/actions`. Each function is one atomic command-level operation against a `Store`:

```go
func AddRepo(s Store, name, url string) error
func RemoveRepos(s Store, names []string) error
func SetRepoPostInstall(s Store, repo, script string) error
func SetRoot(s Store, root string) error
func SetEditor(s Store, editor string) error
```

A small private helper `readWrite(s Store, fn func(*Config) error)` handles the read-modify-write boilerplate. Handlers shrink to: parse input → call action → format output.

Migrated so far: `repo`, `config`, `settings` commands. `workspace`, `scripts`, `tmux` commands still use `HandleConfig` and remain on the migration list (see candidate #1 in the architecture review).

## Consequences

- Invariants (cascade delete, validation) live in one place per operation.
- `MemoryStore` makes every action unit-testable; coverage lives in `internal/actions/*_test.go`.
- Two patterns coexist during migration. New command handlers must use `actions`; do not add new `HandleConfig` callers.

## Alternatives considered

- *Move logic into `Store` adapter methods (e.g. `SQLiteStore.AddRepo`)*: rejected — couples persistence to command semantics, would need to be re-implemented per adapter.
- *Keep `HandleConfig`, just add helpers*: rejected — the callback shape is the leak.
