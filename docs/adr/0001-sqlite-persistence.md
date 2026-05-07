# ADR-0001: SQLite + GORM for persistence

**Status**: Accepted (2026-04 ish — see commit `617b339` for the Store split that completed this)

## Context

Original `qail` persisted state as a single `config.json` under `~/.qail/`. As workspaces, repos, and per-repo / per-workspace post-install script attachments grew, the JSON model became awkward: every write rewrote the whole file, schema changes needed ad-hoc migration code, and querying (e.g. "which workspaces reference this repo") required loading and walking the entire file.

## Decision

Persist state in SQLite at `~/.qail/qail.db` via GORM. Schema covers settings, repos, workspaces, workspace_repos (workspace ↔ package join), and post_install_scripts (script attachments). Migration from JSON is handled by `internal/config/convert.go` (`ConvertJSON`); the legacy JSON file remains readable for backup/restore.

The full `Config` snapshot is still treated atomically: `Store.Read` reconstructs the in-memory `Config`; `Store.Write` replaces all rows in a single transaction (delete-all + re-insert).

## Consequences

- Schema migrations now go through GORM `AutoMigrate` rather than ad-hoc JSON shape checks.
- Writes are still snapshot-style (full replace), so concurrent writers are not safe — same constraint as the JSON era.
- The relational shape lets future queries (cascade-delete a repo from all workspaces, find dangling references) live in SQL or in adapter logic, not in handler code.
- `qail.db` is the user's primary state file; back it up the same way `config.json` was backed up. `BackUp` / `Restore` in `convert.go` handle this.

## Alternatives considered

- *Stay on JSON*: rejected — schema growth was already painful.
- *BoltDB / Badger*: rejected — adds a non-standard file format, no SQL ergonomics.
- *Per-record writes (incremental)*: deferred — snapshot writes match the existing mental model and are atomic by virtue of the transaction. Revisit if write contention or DB size becomes a concern.
