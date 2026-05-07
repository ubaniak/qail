# ADR-0010: No `internal/clip` package; clipboard writes inline at call sites

**Status**: Accepted

## Context

`internal/clip` exposed two functions — `Cd(path)` and `Cmd(cmd)` — each ~3 lines: format a string, print "copied X to clipboard," call `clipboard.WriteAll`. Three callers: `workspace.Cd`, `scripts.Cd`, `tmux.Attach`.

Apply the deletion test: if `clip` were removed, would complexity vanish or reappear at the callers? It vanishes. Each caller writes ~3 lines of `fmt.Printf` + `clipboard.WriteAll`. There is no policy, no testability seam, no abstraction the callers benefit from. The module is a pass-through.

## Decision

Delete `internal/clip`. Inline the three-line "format → print → copy" snippet at each caller (`workspace.Cd`, `scripts.Cd`, the cmd handler that previously called `tmux.Attach`).

Tmux's `Attach` method is gone too: it now exposes `AttachCommand(sessionName) string` (no side effects) and the cmd handler decides whether to copy the result to the clipboard. See ADR-0011.

## Consequences

- One fewer module in `internal/`. The module map matches the architecture: every remaining package is doing something the deletion test rewards.
- Three call sites duplicate the "copied to clipboard" message format. If that message ever needs to change consistently, a new helper can be reintroduced — but only when there's a real policy to centralise, not just identical strings.
- The `github.com/atotto/clipboard` import lives at the call sites. That's the right place: clipboard writes are a UI side effect, and the call sites are where UI policy already lives.

## Alternatives considered

- *Keep `clip` and add a single `clip.WriteWithMessage(msg, cmd)` policy function*: rejected — the policy is "always print this exact message and copy this exact string," which is identical to inlining. A wrapper that adds nothing but a function call is the textbook shallow module.
- *Move `clip` into `internal/color` or another UI utility module*: rejected — same problem, just relocated.
- *Replace clipboard writes with a `Clipboard` interface for testability*: rejected — clipboard writes are a leaf side effect at the very edge of the system. There is nothing to compose past them, and tests don't need to assert on clipboard content.
