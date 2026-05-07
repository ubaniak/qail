# ADR-0006: tmux layout is a pure `Plan` function

**Status**: Accepted (commit `1ce6b2e`)

## Context

`tmux.Launch` previously combined three concerns: read the workspace directory to find subfolders, decide the pane/window layout, and execute the resulting `tmux` commands via subprocess. Layout decisions were untestable in isolation — verifying "split-window vs. new-window for the Nth subfolder" meant running real `tmux`.

## Decision

Layout policy is a pure function:

```go
func Plan(sessionName, root string, subfolders []string) []runner.Command
```

No I/O, no globals, no `Runner`. Returns the exact sequence of `tmux` commands to execute. `Tmux.Launch` reads the directory, calls `Plan`, then runs each `runner.Command` via the injected `Runner`.

## Consequences

- Layout tests are pure: feed in subfolders, assert command shape. See `internal/tmux/layout_test.go`.
- Changing layout policy (e.g. tabs vs. panes, ordering) touches only `Plan`.
- Future "dry-run" mode is trivial: call `Plan`, print the commands, skip execution.
- Same pattern is the goal for any module where policy and execution can separate. Treat it as the reference shape for "pure plan + side-effecting executor."

## Alternatives considered

- *Keep layout inline in `Launch`, mock `Runner` for tests*: rejected — possible, but couples policy tests to execution mechanics. Pure function is sharper.
- *Configurable layouts via a strategy interface*: deferred — one layout exists. Add the interface when a second layout appears (one adapter is a hypothetical seam).
