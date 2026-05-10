# ADR-0013: Installer owns spinner UI; `git` package stays pure

**Status**: Accepted (extends ADR-0003, ADR-0007)

## Context

`git.CloneWithProgress(repo, path, message)` wrapped `Clone` in a `forms.Spinner` closure. The `git` package therefore imported `internal/forms`, pulling a TUI dependency into a pure subprocess module. Direction of the import was wrong: domain (subprocess seam) reaching up into UI.

Symptoms:

1. The closure swallowed the clone error into a captured variable because `forms.Spinner` has no error channel — untestable shape.
2. `git` had a TUI dependency for one method, blocking reuse from non-interactive contexts (cron, future server-side flows).
3. The "show progress while cloning" decision was scattered: only one caller (`installer.Install`) wanted it, but every git consumer paid the import cost.

## Decision

`git.Git.Clone(repo, path) (string, error)` is the only entry point. No spinner, no UI. The `forms` import is removed from `internal/git`.

`internal/installer/spinner.go` defines a `spinnerGit` adapter that satisfies `installer.GitClient`:

```go
type spinnerGit struct{ inner *git.Git }

func (s spinnerGit) Clone(repo, path, message string) error {
    var err error
    forms.Spinner(func() { _, err = s.inner.Clone(repo, path) }, message)
    return err
}
```

`installer.Default()` wires `spinnerGit{inner: git.Default()}`. Tests inject a plain `fakeGit` and skip the spinner entirely.

The `installer.GitClient` interface keeps the `message` parameter so future adapters (silent, structured-log, web-progress) can take it or ignore it.

## Consequences

- `internal/git` now imports only `runner` and stdlib — domain is pure, follows the seam discipline of ADR-0003.
- The orchestration layer (`installer`) owns the "show progress" decision, where it always belonged. Future callers (`qail update`, `qail ws add-package`) inherit the same UX without each one reimplementing it.
- `git_test.go` is unchanged (already tested `Clone` directly). `installer_test.go` renames `fakeGit.CloneWithProgress` → `Clone` — no test logic changes.
- The `spinnerGit` adapter is internal to `installer`; nothing outside the package can construct one, so the seam stays narrow.

## Alternatives considered

- *Inline `forms.Spinner` inside `installer.Install`*: rejected — works, but `installer` then imports `forms` directly. The adapter pattern keeps `Installer` itself UI-free; only the `spinnerGit` file imports `forms`.
- *Return a progress channel from `git.Clone`*: rejected — overkill for a binary "spin while waiting" affordance, and `huh.Spinner` has no compatible API.
- *Keep `CloneWithProgress` as a deprecated alias*: rejected — no external callers; deprecation noise without benefit.
