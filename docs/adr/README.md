# Architecture Decision Records

Decisions that shape `qail`'s architecture. Numbered, append-only.

| # | Title | Status |
|---|-------|--------|
| [0001](0001-sqlite-persistence.md) | SQLite + GORM for persistence | Accepted |
| [0002](0002-store-interface.md) | `config.Store` interface with SQLite and Memory adapters | Accepted |
| [0003](0003-runner-seam.md) | `runner` package as the subprocess seam | Accepted |
| [0004](0004-actions-package.md) | `actions` package owns atomic command-level operations | Accepted |
| [0005](0005-qailhome-paths.md) | `qailhome` is the single source of truth for `~/.qail/` layout | Accepted |
| [0006](0006-tmux-plan-pure.md) | tmux layout is a pure `Plan` function | Accepted |
| [0007](0007-workspace-injection-seams.md) | `Workspace` is wired with consumer-defined `Installer` and `FS` interfaces | Accepted |
| [0008](0008-forms-return-values.md) | Forms return values, never mutate caller state | Accepted |
| [0009](0009-actions-own-workspace-lifecycle.md) | Actions own the entire command lifecycle; `HandleConfig` retired | Accepted |
| [0010](0010-no-clip-package.md) | No `internal/clip` package; clipboard writes inline at call sites | Accepted |
| [0011](0011-tmux-attach-pure.md) | `Tmux.AttachCommand` returns a string; clipboard writes at the caller | Accepted |
| [0012](0012-qailhome-purpose-specific-paths.md) | `qailhome` exposes only purpose-specific paths | Accepted |

## Format

Each ADR: Status, Context, Decision, Consequences, Alternatives considered. Keep short. Reference commits when the decision landed in code.

When a future decision overturns one of these, add a new ADR (don't edit history); mark the old one *Superseded by ADR-NNNN*.
