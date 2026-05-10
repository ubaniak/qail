# ADR-0015: Forms split selection from confirmation

**Status**: Accepted (extends ADR-0008)

## Context

ADR-0008 settled the "forms return values, never mutate caller state" rule. The first wave produced three combined helpers that returned a `(selection, confirmed, error)` tuple:

- `forms.SelectWorkspaceToRemove(ws) (string, bool, error)`
- `forms.SelectReposToRemove(repos) ([]string, bool, error)`
- `forms.RemoveTmuxSession(sessions) (string, bool, error)`

Each glued together a `huh.Select`/`huh.MultiSelect` with a `huh.Confirm` group inside one `huh.Form`. The bool return forced every caller to interpret "did the user confirm?" identically. Three problems:

1. **Duplicated UX policy** — each combined helper had its own confirm prompt copy ("This will remove the selected workspace. Are you sure?"). Drift was inevitable.
2. **Untestable composition** — selection and confirmation can't be exercised independently because they're a single `huh.Form` invocation.
3. **No reuse** — selectors couldn't be used in non-destructive flows (e.g. "pick a workspace to inspect") without dragging the confirm prompt along.

A separate cmd-layer helper, `confirmOrSkip(prompt, ttyReason)`, already centralised the `--yes` + `requireTTY` + `forms.Confirm` chain for non-interactive removes. The combined forms duplicated part of that policy from inside the TUI.

## Decision

Forms expose pure selectors. Confirmation is a separate call.

```go
// forms/workspace.go
func SelectWorkspaceName(ws config.Workspace) (string, error)

// forms/repo.go
func SelectRepos(repos map[string]string) ([]string, error)

// forms/tmux.go
func SelectTmuxSession(sessions []string) (string, error)
```

Each runs a single-group `huh.Form` and returns the chosen name(s). No bool, no embedded confirm.

Cmd handlers chain explicitly:

```go
name, err := forms.SelectWorkspaceName(ctx.Workspaces)
// ... handle err
confirmed, err := forms.Confirm(fmt.Sprintf("Remove workspace %q?", name))
// ... handle err
if !confirmed { return }
actions.RemoveWorkspace(s, name)
```

Removed:

- `forms.SelectWorkspaceToRemove`, `forms.SelectReposToRemove`, `forms.RemoveTmuxSession`.
- `forms.CleanWorkspace` — was a `huh.Confirm` with a hardcoded title; replaced by `confirmOrSkip` at the cmd layer.

## Consequences

- One confirm prompt source (`forms.Confirm`); one TTY/--yes policy (`cmd.confirmOrSkip`). UX wording changes happen in one place.
- Selectors are reusable in any flow (current: remove paths; future: inspect, duplicate, archive).
- Each piece of the flow has a single-responsibility test surface. `huh` still lacks a unit-test mode but the seams are correct: when `huh` adds testability the selectors and the confirm are independently exercised.
- `forms/workspace.go` shrinks 47 → 21 lines for the remove helpers; `forms/repo.go` and `forms/tmux.go` similarly trim.

## Alternatives considered

- *Keep the combined helpers as "convenience" wrappers around the new selectors*: rejected — same readability hit as the original, plus a maintenance tax. ADR-0008's discipline (explicit at the call site) applies here too.
- *Move confirmation into the action layer*: rejected — actions stay UI-agnostic per ADR-0009. Confirm is a UX concern that lives with the cmd handler.
- *Hardcode the confirm copy inside each selector via a `confirmTitle` parameter*: rejected — that's the combined helper with extra steps, and it still glues the two `huh` groups together.
