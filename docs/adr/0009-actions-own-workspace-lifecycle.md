# ADR-0009: Actions own the entire command lifecycle; `HandleConfig` retired

**Status**: Accepted (extends ADR-0004)

## Context

ADR-0004 introduced `internal/actions` as the home for atomic command-level operations and migrated the `repo`/`config`/`settings` cmds. The `workspace`, `scripts`, `tmux`, and `init` cmds remained on the older `HandleConfig` callback pattern: `HandleConfig(func(cfg *Config) error)` read a snapshot, handed the handler a mutable pointer, wrote back on success.

Workspace handlers were the worst offenders:

- `tmuxWsCmd` performed an explicit `config.WriteToFile(*cfg)` mid-handler to persist `LastUsed` before launching tmux. Two writes per invocation; the second `WriteToFile` from `HandleConfig` was a no-op.
- `createWsCmd`, `addWsCmd`, `cloneWsCmd`, `editWsCmd` each rebuilt a `Workspace` struct inline by reading `cfg.Root`, `cfg.Repos`, `cfg.PostInstallScripts.Repo`, `cfg.PostInstallScripts.Workspace` directly.
- `removeWsCmd` relied on `forms.RemoveWorkspace` mutating `cfg.Workspaces` via pointer (see ADR-0008).
- The "look up a workspace by name and bump LastUsed" pattern was hand-rolled in three places (`open`, `cd`, `mux`).

## Decision

Every cmd handler now routes through `internal/actions`. The deprecated `HandleConfig`, `config.WithConfig`, `config.ReadFromFile`, and `config.WriteToFile` shims are removed.

New workspace actions:

- `AddWorkspace(s, name, packages)` — disk create + register
- `EditWorkspace(s, name, packages)` — rebuild + update registry
- `CloneWorkspace(s, dstName, packages)` — alias of AddWorkspace with explicit dst
- `CreateWorkspaceOnDisk(s, name)` — re-materialise registered workspace without touching registry
- `RemoveWorkspace(s, name)` — strip from registry and post-install attachments
- `TouchWorkspace(s, name) (WorkspaceProfile, error)` — atomic LastUsed bump; returns the previous profile so the caller can do its side effect
- `ListWorkspaces(s)` — read-only, mirrors `ListRepos`
- `SetWorkspacePostInstall(s, name, scripts)` — manage workspace-scoped post-install
- `ReadWorkspaceContext(s) WorkspaceContext` — read-only struct for cmds that need Root/Editor/Workspaces without mutation

Cmd handlers shrink to: parse flags → forms prompt → call action(s) → format output. The previous pattern of "read mutable Config, mutate fields, write back" is gone from `cmd/`.

## Consequences

- Atomic operations are guaranteed at the action layer. The `mux` double-write is gone — `TouchWorkspace` does one read-write cycle.
- `MemoryStore` makes every workspace operation unit-testable (see `internal/actions/workspace_test.go`).
- The `Config` struct is no longer the de facto handler API — handlers see only what `WorkspaceContext`, `ListWorkspaces`, etc. expose.
- `cmd/config.go` lost the `HandleConfig` helper; all four files in `cmd/` use the same pattern.

## Alternatives considered

- *Keep `HandleConfig` for the simple read-only cases*: rejected — having two patterns is the leak. New `ReadWorkspaceContext`/`ListWorkspaces` actions cover the read-only cases just as concisely.
- *Move `forms.*` calls inside actions*: rejected — actions stay UI-agnostic, so the same actions back the cmd handlers and any future REST/TUI front-end.
- *Expose `*config.Config` from `ReadWorkspaceContext`*: rejected — that would re-introduce the wide schema-as-API leak. The struct intentionally exposes only Root/Editor/Workspaces.
