# ADR-0020: Agent Console inside the qail binary, driving the installed claude CLI

**Status**: Accepted. Supersedes ADR-0018 (Node sidecar) and ADR-0019 (bundling Node and the SDK).

## Context

ADR-0018 ran the Agent Console as a Node/Express sidecar because the Claude
Agent SDK, which gives agents parity with Claude Code, only ships for
TypeScript and Python. ADR-0019 then bundled that sidecar, a pinned Node
and the SDK's native Claude Code binary into every installer, about 500 MB
unpacked, plus a second loopback API (`:8765`) so the sidecar could read
qail's workspaces.

The SDK turns out to be a thin client. It spawns the Claude Code CLI with
`--input-format stream-json --output-format stream-json` and, for
`canUseTool`, `--permission-prompt-tool stdio`. The CLI then asks for each
permission as a `control_request` (subtype `can_use_tool`) on stdout and
waits for a `control_response` on stdin. Session resume, named agents
(`--agent`), setting sources, skills and the `claude_code` system prompt
are all CLI flags. Go can speak that protocol directly, so the reason for
Node is gone.

## Decision

1. **The console runs in-process.** `internal/console` serves the UI
   (embedded from `internal/console/web` with `go:embed`) and its JSON/SSE
   API on `127.0.0.1:4317` from whichever qail process opens it: the
   desktop backend (tray menu) or `qail console`. If one already answers on
   the port, the other just opens the browser.
2. **It reads the config Store directly**, in the same process, instead of
   going through the HTTP API. "Clone missing repos" calls
   `actions.CreateWorkspaceOnDisk`. The desktop backend no longer serves
   `internal/api` on `:8765`; `qail serve` is unchanged.
3. **Chats drive the installed `claude` CLI** (`claude.go`): one process
   per turn with `--input-format/--output-format stream-json --verbose
   --include-partial-messages --permission-prompt-tool stdio
   --setting-sources user,project,local --allowedTools Skill`, plus
   `--permission-mode`, `--agent`, `--resume` and `--append-system-prompt`
   (workspace layout) as needed. Permission requests become the same
   browser cards as before; read-only tools are allowed without asking,
   unanswered cards are denied after 10 minutes, and Stop cancels the
   process (the session stays resumable).
4. **The CLI is found, not shipped**: `$QAIL_CLAUDE`, else PATH, else the
   usual install locations (`~/.local/bin`, `~/.claude/local`, Homebrew,
   npm global, Volta, Bun, nvm), since the menubar app doesn't inherit the
   shell's PATH. The user chose this over bundling the binary.
5. **Threads stay in `~/.qail/console/data/store.json`**, in the Node
   console's format, so existing conversations carry over and resume.
6. **The browser guard gets stricter**: cross-origin API calls are refused
   as before, and so is any Host header other than `127.0.0.1`/`localhost`
   (DNS rebinding).

## Consequences

- No Node at runtime or in the installers; `console/`,
  `scripts/bundle-console.mjs` and the `console-*` make targets are gone.
  Installers shrink by roughly 500 MB unpacked.
- The console needs Claude Code installed. Without it the console still
  opens and shows where it looked; the tray menu shows a hint.
- Agents use the same `claude` version, login, settings, agents and skills
  as the user's terminal. Updating Claude Code no longer needs a qail
  release, but a CLI change to the stream-json control protocol could
  break chats. The protocol is what the official SDKs depend on, and the
  tests pin our side of it with a fake CLI.
- One loopback port for the console instead of two.

## Alternatives considered

- **Keep the Node sidecar (ADR-0018/0019).** Works, but costs ~500 MB per
  installer, a second process and port, and a JS toolchain at build time.
- **Bundle the claude binary without Node.** Works with no Claude Code
  install, but keeps ~350 MB per installer and pins the agent's version to
  qail releases.
- **`--permission-prompt-tool` pointing at an MCP server qail hosts.**
  Also documented, but needs an MCP server and config per chat for what
  the stdio control protocol does in-band.
