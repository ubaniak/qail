# ADR-0007: `Workspace` is wired with consumer-defined `Installer` and `FS` interfaces

**Status**: Accepted

## Context

`Workspace.Create` previously called `installer.Default()` mid-method and used `os.Stat`/`os.Mkdir`/`os.RemoveAll` directly. Tests for workspace lifecycle either touched real disk or were absent. The same pattern applied to `Workspace.Remove`, `Workspace.RemoveRepo`, and the package-level `Clean`.

This contradicted the trajectory established by ADR-0003 (Runner seam) and ADR-0004 (actions): every other module that does subprocess or mutating work has been wired to inject its dependencies, but workspace was still constructing them inline.

## Decision

`Workspace` gains two injected dependencies:

- `Installer` — narrow consumer interface, defined in `workspace`:
  ```go
  type Installer interface {
      Install(spec installer.PackageSpec) error
      RunPostInstall(dir string, scripts []string) error
  }
  ```
- `FS` — narrow consumer interface, defined in `workspace`:
  ```go
  type FS interface {
      Stat(name string) (os.FileInfo, error)
      Mkdir(path string, perm os.FileMode) error
      RemoveAll(path string) error
      ReadDir(name string) ([]os.DirEntry, error)
  }
  ```

Two constructors:

- `New(root, name, packages, repos, inst, fs)` — explicit, used by tests and any future caller that wants to wire alternative adapters.
- `NewDefault(root, name, packages, repos)` — wires `installer.Default()` + `OSFS{}`. Cmd handlers and actions use this.

Both interfaces follow the Go convention of being defined by the consumer (workspace) so tests can fake them without depending on the production module.

The free function `Clean` keeps its signature for callers but delegates to `cleanWithDeps(fs, confirm, root, ws)` — the testable form. Production wires `OSFS{}` and `forms.Confirm`; tests inject fakes.

## Consequences

- `internal/workspace` gains its first unit tests: `TestCreateInvokesInstallerForEachPackage`, `TestCreateRollsBackWorkspaceDirOnFailure`, and clean variants. Disk and subprocess no longer required.
- Workspace lifecycle invariants (clone all packages → run scripts → roll back on failure) become testable in isolation.
- Cmd handlers stop calling `installer.Default()` indirectly through `Workspace.Create`; the construction is explicit at handler/action level.

## Alternatives considered

- *Keep calling `installer.Default()` inside `Create`*: rejected — blocks testing, contradicts the seam trajectory.
- *Use `os/exec` directly and let workspace tests mock via `Runner`*: rejected — workspace's I/O is filesystem, not subprocess. A separate seam keeps each module focused on the right abstraction level.
- *Define `Installer` and `FS` in their own packages*: rejected — Go's convention is consumer-defined interfaces. Nothing else in qail consumes these specific shapes.
