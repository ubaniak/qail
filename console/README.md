# Agent Console

Local web UI (bundled with qail — see ../docs/adr/0018-agent-console-sidecar.md) for your Claude Code agents, organised by **qail** workspaces and repos.

- **qail workspaces** – the sidebar is your qail workspace → repo tree (`<Root>/<workspace>/<repo>`), read live from `qail serve`. No separate project list to maintain.
- **Two scopes** – pick a *workspace* to run agents at the workspace root across all its repos (the agent is told the repo layout), or a *repo* to run inside it with that repo's own CLAUDE.md, skills and hooks.
- **Clone missing repos** – runs qail's workspace create (clone + post-install scripts) and streams its output.
- **Agents** – every agent in `<scope>/.claude/agents` and `~/.claude/agents`, with its model, tools, preloaded skills, and the skills it has actually invoked in that project. `main` is the plain Claude Code session (use it to run `/ship`).
- **Skills** – every skill in `<scope>/.claude/skills` and `~/.claude/skills`, and which agents preload it.
- **Chat** – talk to any agent, per project. Runs through the Claude Agent SDK with `agent: <name>`, so it uses the same prompt, tools, model, skills, CLAUDE.md, settings and hooks as Claude Code. Threads persist and resume per workspace/repo.
- **Permissions** – anything beyond read-only tools shows an Allow / Deny card in the chat. Modes: *Ask before changes*, *Auto-accept edits*, *Plan only*.
- **/ship runs** – browse `.agent/runs/*` (state, plan, context, verify and review reports).

## Run

```bash
npm install
npm start            # http://127.0.0.1:4317
```

Auth: uses your existing Claude Code login, or set `ANTHROPIC_API_KEY`.
qail: the console connects to `qail serve` at `QAIL_URL` (default `http://127.0.0.1:8765`). If it isn't running, the console starts it using `QAIL_BIN`, `~/.qail/bin/qail`, or `qail` on PATH, and stops it on exit. Set `QAIL_AUTOSTART=0` to disable.

Env: `PORT` (default 4317), `AGENT_CONSOLE_DATA` (default `./data`), `QAIL_URL`, `QAIL_BIN`, `QAIL_AUTOSTART`.

## Notes
- Binds to 127.0.0.1 only and rejects cross-origin API calls. There is no login, so don't expose it on a network.
- Project hooks (e.g. the /ship guard hook) still apply to every chat.
- Chat threads live in `data/store.json`; workspaces and repos are always qail's.
