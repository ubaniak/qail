# ADR-0016: HTTP API layer over the actions package

**Status**: Accepted

## Context

qail had grown a clean seam stack — `cmd/` → `actions/` → domain packages
(`workspace`, `installer`, `git`, `scripts`, `tmux`) → `runner` → SQLite
`Store` — and a parallel non-interactive flag path through every Cobra
handler. The remaining blocker for a web UI was that the domain packages
wrote progress directly to `os.Stdout`, hardcoded `context.Background()`,
and one action (`OpenWorkspace`) inherited the parent process's stdio so
the editor could take over the terminal. None of those behaviours are
reachable from an HTTP handler — the server's stdout is not the user's
terminal, and a long-running clone needs to stream output to the browser
while honouring client disconnects.

A web UI is a value-add for users who don't live in the terminal, and
also unlocks remote-friendly workflows: a qail server on a dev box, the
UI in a browser tab on a laptop. The architecture audit (recorded in the
PR description) confirmed every action could be called from an HTTP
handler if four things were true: progress took an `io.Writer`, every
runner call took a `ctx`, concurrent reads-modify-writes serialised
correctly, and TTY-bound side effects (`OpenWorkspace`, `ExploreWorkspace`,
`scripts.Cd`) were split into pure command-string variants.

## Decision

Add `internal/api/` exposing the actions package over HTTP. Refactor the
underlying packages to satisfy the four preconditions:

1. **`io.Writer` for progress.** `workspace.Create`, `installer.Install`
   /`RunPostInstall`, `scripts.RunBashScript`, `workspace.Clean` accept
   an `io.Writer`. A nil writer falls through to `os.Stdout` so existing
   CLI callers don't change behaviour. The HTTP layer hands an SSE writer
   that emits one `data:` event per progress line.

2. **`context.Context` everywhere.** `git.Clone`, every `tmux.*` method,
   `scripts.RunBashScript`/`Open`, and the `OpenWorkspace` /
   `ExploreWorkspace` actions take a `ctx`. Cobra handlers pass
   `context.Background()`; HTTP handlers pass `r.Context()` so client
   disconnects propagate to the subprocess.

3. **Atomic read-modify-write.** `actions.readWrite` now holds a package-
   level `sync.Mutex` for the entire load-mutate-write cycle. SQLite's
   own write lock cannot prevent the lost-update race where two callers
   each `Read` a snapshot and then `Write` independently; the mutex
   does. Cost: all action mutations serialise across all callers in the
   same process. Acceptable for qail's workload (one user, occasional
   writes, never on a hot path).

4. **Split TTY actions from pure variants.** `OpenWorkspace` keeps its
   stdio-inheriting behaviour for CLI; a new `OpenWorkspaceCommand`
   returns the editor + path as data. `ExploreWorkspace` is paired with
   `ExploreWorkspacePath`. `scripts.Cd` becomes `CdCommand` returning the
   `cd <dir>` string; the cmd handler does the clipboard copy. HTTP
   handlers always call the pure variants and return JSON.

The HTTP layer:

- One `Server` struct holding only the `config.Store`. Constructed in
  `cmd/serve.go` and given to the API.
- Routes registered on `http.NewServeMux` using Go 1.22 `METHOD path`
  patterns. Sub-files split handlers by domain (config, repos,
  workspaces, scripts, mux).
- Long-running mutations (workspace add / edit / clone / on-disk create)
  emit Server-Sent Events: one event per progress line, plus a final
  `event: done` or `event: error`.
- All other endpoints are JSON request/response.
- No authentication. Default bind is `127.0.0.1:8765`. Documented in the
  serve command's long help.

Resulting layer cake:

```
HTTP client ─┐
             │
cmd/serve.go ┼─→ internal/api/  ─┐
             │                    ├─→ internal/actions/  ─→ internal/{workspace,installer,git,scripts,tmux,config}
cmd/{ws,...} ┘                    │
                                  └─ same actions called by Cobra
```

## Consequences

**Positive**

- Web UI can be built against a stable JSON+SSE surface without touching
  domain code.
- The same refactor that unblocks HTTP also makes the domain packages
  more pleasant for unit tests (writers are easier to assert on than
  captured stdout).
- `context` propagation gives every long-running op a real cancellation
  path, a latent CLI improvement (Ctrl-C now reaches the subprocess).

**Negative**

- Action signatures grew. Every caller of `AddWorkspace`,
  `EditWorkspace`, `CloneWorkspace`, `CreateWorkspaceOnDisk`,
  `OpenWorkspace`, `ExploreWorkspace`, `MuxWorkspace` had to be updated.
  One-time cost; net signature is more honest about what the action does.
- The package-level mutex in `actions/` serialises mutations process-
  wide. If qail ever grows long-running background jobs that need
  parallel mutation, this needs to become a per-store mutex on the Store
  interface. Documented in `actions/actions.go`.
- The HTTP server has no auth. Loopback-only default mitigates exposure
  but the user is responsible for not binding `0.0.0.0` without a proxy.

## Alternatives considered

- **Embed gRPC.** Web UIs would need grpc-web shim. JSON+SSE is enough
  for the workload; rejecting gRPC keeps the dependency surface flat.
- **Fork the actions package into HTTP-specific shapes.** Considered;
  rejected because the divergence would be small (mostly the
  command-string variants of session-side actions) and forking would
  create two action surfaces to keep in sync forever.
- **Use a websocket for everything.** SSE is one-way; that's all
  progress streaming needs. Bidirectional streams aren't justified yet.
