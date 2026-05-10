# ADR-0014: Actions own session-side commands (open / cd / explore / mux)

**Status**: Accepted (extends ADR-0009)

## Context

ADR-0009 migrated mutation-heavy workspace handlers (add/edit/clone/create/remove) into `internal/actions`. The session-side handlers — `open`, `cd`, `explore`, `mux` — were left straddling the line:

- `cmd/workspace.go` resolved a workspace name + path via `resolveWorkspaceName`, called `actions.TouchWorkspace`, then invoked `workspace.Open(editor, path)` / `workspace.Cd(path)` / `workspace.Explore(path)` / `workspace.Tmux(path)`.
- `workspace.Open`, `Cd`, `Explore` were free functions that called `defaultRunner()` internally — no Runner injection seam, untestable.
- `workspace.Cd` wrote to the clipboard inline, which contradicted ADR-0010 (clipboard writes belong at cmd).
- The "touch + side effect" lifecycle was hand-rolled in four places.

## Decision

Four new actions in `internal/actions/sessions.go`:

```go
func OpenWorkspace(s config.Store, r Runner, name string) error
func ExploreWorkspace(s config.Store, r Runner, name string) error
func CdWorkspace(s config.Store, name string) (path string, err error)
func MuxWorkspace(s config.Store, name string) (attachCmd string, err error)
```

Each owns the full lifecycle: read snapshot → validate registration → assert on-disk path exists → bump LastUsed → execute side effect (or return the value the cmd needs to act on).

`Runner` is a narrow consumer interface defined in `actions`, satisfied by `runner.OS` in production and `runner.Recorder` in tests — the same pattern as ADR-0007's `Installer` and `FS` seams.

Removed:

- `workspace.Open`, `workspace.Cd`, `workspace.Explore` (free functions).
- `workspace.defaultRunner` (no longer needed; Runner injection happens at the action seam).
- `cmd.resolveWorkspaceName`'s 4-tuple shape — replaced with a name-only resolver since disk validation moved into the action.

`workspace.Tmux` is kept (the `Workspace` package still owns the tmux-launch invariant) but `MuxWorkspace` wraps it with the touch + path-resolution lifecycle.

Cmd handler shape after migration:

```go
name, err := resolveWorkspaceName(s, firstArg(args))
// ... handle err
if err := actions.OpenWorkspace(s, runner.NewOS(), name); err != nil {
    log.Fatalln(err)
}
```

`cd` and `mux` cmd handlers receive a string from the action (the wsPath or attachCmd respectively) and write it to the clipboard inline, honoring ADR-0010 and ADR-0011.

## Consequences

- All four session handlers are unit-testable via `MemoryStore` + `runner.Recorder`. Today they have no tests; future tests can assert "Open invokes editor with path X and bumps LastUsed."
- The "touch + side effect" invariant is impossible to forget — it lives once, in the action.
- `internal/workspace` shrinks (-43 lines): no more free functions, no more `defaultRunner`, no `clipboard` import, no `log` import.
- Cmd handlers for open/cd/explore/mux drop from ~10 lines to ~6, and never mention `runner` directly except at the injection point (`runner.NewOS()`).

## Alternatives considered

- *Keep free functions in `workspace` and add Runner parameter*: rejected — moves injection burden onto every cmd handler instead of concentrating it in one action call. Doesn't fix the "touch + side effect" duplication.
- *Make `MuxWorkspace` re-implement tmux launch directly*: rejected — `workspace.Tmux` already encapsulates the "ensure session, return attach command" invariant. The action wraps it for the lifecycle concerns.
- *Return a `runner.Command` from the action and let cmd execute it*: rejected — pushes execution choice (stdio inheritance for editors, etc.) back into cmd handlers. The action is the right place to know the editor needs `os.Stdin` inherited.
