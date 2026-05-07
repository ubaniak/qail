# ADR-0003: `runner` package as the subprocess seam

**Status**: Accepted

## Context

`git`, `tmux`, and `scripts` all shell out to external binaries. Without a shared seam, each module either calls `os/exec` directly (untestable) or invents its own wrapper. Tests would require either real subprocess execution or per-module mocking.

## Decision

`internal/runner` defines `Command` (Name, Args, Dir, Env, optional std streams) and `Result` (Stdout, Stderr, ExitCode). The `OS` adapter executes via `os/exec`; the `Recorder` adapter captures invocations for tests.

Modules that run subprocesses (`git`, `tmux`, `scripts`, `installer`) accept a `Runner` at construction. Default constructors (`Default()`) wire `runner.NewOS()`; tests inject `runner.Recorder`.

## Consequences

- All subprocess-using modules share one mocking strategy. New tests follow a copy-paste pattern.
- Pure-logic extractions become natural: `tmux.Plan` (see ADR-0006) returns `[]runner.Command` and stays pure; `Launch` executes them via `Runner`.
- `Runner` is now the canonical pattern for any future "shell out to X" need. New modules that touch subprocesses must accept a `Runner`.

## Alternatives considered

- *Per-module command interfaces (e.g. `GitClient`, `TmuxClient`)*: kept *in addition to* `Runner` — `installer` uses `GitClient`/`ScriptsClient` to compose at the right level of abstraction. `Runner` is the lower seam; module-shaped interfaces sit on top.
- *Mock `os/exec` directly*: rejected — leaks stdlib types into every test.
