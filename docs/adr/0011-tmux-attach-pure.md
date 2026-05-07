# ADR-0011: `Tmux.AttachCommand` returns a string; clipboard writes happen at the caller

**Status**: Accepted

## Context

`Tmux.Attach(sessionName)` previously formatted `tmux a -t <session>` and wrote it to the clipboard via `internal/clip`. The tmux module owned a UI side effect (clipboard write) bundled with session management.

Two issues:

1. The tmux module mixed concerns. Session lifecycle (launch/exists/list/remove) is one job; deciding what to do with the attach command (copy to clipboard, print to stdout, push to a remote terminal) is another.
2. Tests for `Attach` had to either tolerate the clipboard side effect or skip the method.

## Decision

`Tmux.Attach` is replaced by `Tmux.AttachCommand(sessionName string) string`. The method has no side effects: it returns the command string (`tmux a -t <sessionName>` with shell-quoted name) and lets the caller decide what to do with it.

Cmd handlers that want the old "copy to clipboard" behaviour do so explicitly:

```go
attachCmd, err := workspace.Tmux(wsPath)
// ...
fmt.Printf("%s copied %s to clipboard\n\n", color.Yellow(">>>"), color.Green(attachCmd))
clipboard.WriteAll(attachCmd)
```

This couples cleanly with ADR-0010 (no `clip` package) — clipboard writes are a UI concern that lives at the cmd-handler edge.

## Consequences

- `Tmux` is now a pure session-management module. Every method either reads or mutates tmux state via `Runner`; nothing touches the clipboard.
- Tests can assert on the exact string returned without involving real clipboard infrastructure.
- A future "send attach command to a remote terminal" feature wouldn't need to fight `Tmux.Attach`'s clipboard write.

## Alternatives considered

- *Take a `Clipboard` interface as a constructor parameter*: rejected — adds a dependency for a side effect the caller can perform in one line. The interface would have one method and one production adapter; ADR-0003 says "two adapters justify a seam."
- *Return an error if clipboard write fails*: rejected — moot once the side effect moves out of the module.
