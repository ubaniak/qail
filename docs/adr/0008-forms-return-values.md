# ADR-0008: Forms return values, never mutate caller state

**Status**: Accepted

## Context

Several forms in `internal/forms` previously took `*config.X` parameters and mutated them as a side effect. The most blatant example was `RemoveWorkspace(*config.Workspace) error` which called `delete(*ws, name)` once the user confirmed. The form layer owned the domain decision (delete this entry) and smuggled it into the data model via pointer mutation.

Consequences:

1. Tests for the workspace removal flow couldn't separate UI from domain logic.
2. The form's contract was implicit: "after this call, your map may be smaller." Easy to miss when reading the call site.
3. Form reuse across alternate front-ends (REST, scripted) was blocked — every consumer would inherit the mutation.

## Decision

Every form in `internal/forms` follows one shape: it returns the user's input plus an error. Forms never accept mutable pointers and never modify caller state.

Concretely:

- `forms.SelectWorkspaceToRemove(ws config.Workspace) (name string, confirmed bool, err error)` replaces the old `RemoveWorkspace(*config.Workspace) error`.
- `forms.SelectReposToRemove`, already in this shape, is the reference.
- `forms.Confirm(prompt) (bool, error)` is the one-line version of the same pattern.

The caller (cmd handler or `actions.*`) applies the mutation against the `Store`. Forms do not import `config.Store` and do not import action helpers.

## Consequences

- Form return signatures grow (3-tuples are common) but the contract is explicit at the call site.
- Domain rules (e.g. cascade-delete a workspace from `PostInstallScripts`) live in actions where they belong; forms can't accidentally implement the wrong cascade.
- Forms remain testable as pure UI prompts (manually, since `huh` doesn't yet have a unit-test mode), and the domain rules become unit-testable via `MemoryStore`.

## Alternatives considered

- *Forms own a `config.Store` and write directly*: rejected — couples UI to persistence, blocks reuse, makes form unit tests require a fake Store.
- *Return a callback the caller invokes*: rejected — same readability hit as the pointer mutation, plus a closure layer.
- *Keep the pointer pattern for "convenience"*: rejected — convenience that breaks the deletion test isn't convenience.
