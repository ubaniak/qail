// Agent Console — local web UI for Claude Code agents across qail workspaces and repos.
// Runs on 127.0.0.1 only. Uses the Claude Agent SDK, so agents behave exactly as in Claude Code:
// same .claude/agents, skills, CLAUDE.md, settings and hooks of each project.
import express from "express";
import matter from "gray-matter";
import { query } from "@anthropic-ai/claude-agent-sdk";
import fs from "node:fs/promises";
import { existsSync } from "node:fs";
import path from "node:path";
import os from "node:os";
import crypto from "node:crypto";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const PORT = Number(process.env.PORT || 4317);
const HOST = "127.0.0.1";
const DATA_DIR = process.env.AGENT_CONSOLE_DATA || path.join(__dirname, "data");
const STORE = path.join(DATA_DIR, "store.json");
const HOME_CLAUDE = path.join(os.homedir(), ".claude");
const MAIN_AGENT = "main";

// ---------- store (threads only; workspaces/repos come from qail) ----------
let store = { threads: {} };
async function loadStore() {
  await fs.mkdir(DATA_DIR, { recursive: true });
  if (existsSync(STORE)) store = { threads: {}, ...JSON.parse(await fs.readFile(STORE, "utf8")) };
  delete store.projects;
}
let saving = Promise.resolve();
function saveStore() {
  saving = saving.then(() => fs.writeFile(STORE, JSON.stringify(store, null, 2)));
  return saving;
}
const WS_SCOPE = "_"; // repo slot meaning "the whole workspace"
const scopeId = (ws, repo) => `${ws}/${repo}`;
const threadKey = (scope, agent) => `${scope}::${agent}`;

// ---------- qail ----------
const QAIL_URL = (process.env.QAIL_URL || "http://127.0.0.1:8765").replace(/\/$/, "");
let qailProc = null;
let qailStatus = { ok: false, url: QAIL_URL, spawned: false, error: null };

function findQailBin() {
  if (process.env.QAIL_BIN) return process.env.QAIL_BIN;
  const home = process.env.QAIL_HOME || path.join(os.homedir(), ".qail");
  const candidates = [path.join(home, "bin", "qail"), ...(process.env.PATH || "").split(path.delimiter).map((d) => path.join(d, "qail"))];
  return candidates.find((c) => existsSync(c)) || null;
}

async function qail(p, opts = {}) {
  const r = await fetch(QAIL_URL + p, { ...opts, headers: { "content-type": "application/json", ...(opts.headers || {}) } });
  if (!r.ok) throw new Error(`qail ${p}: ${r.status} ${(await r.text()).trim()}`);
  return opts.raw ? r : r.json();
}

async function ensureQail() {
  try { await qail("/api/health"); qailStatus = { ...qailStatus, ok: true, error: null }; return true; } catch {}
  if (!qailProc && process.env.QAIL_AUTOSTART !== "0") {
    const bin = findQailBin();
    if (bin) {
      qailProc = spawn(bin, ["serve", "--addr", new URL(QAIL_URL).host], { stdio: "ignore" });
      qailProc.on("exit", () => { qailProc = null; });
      qailStatus.spawned = true;
      for (let i = 0; i < 20; i++) {
        await new Promise((r) => setTimeout(r, 250));
        try { await qail("/api/health"); qailStatus = { ...qailStatus, ok: true, error: null }; return true; } catch {}
      }
    }
  }
  qailStatus = { ...qailStatus, ok: false, error: findQailBin() ? `qail serve not reachable at ${QAIL_URL}` : "qail binary not found (set QAIL_BIN or add ~/.qail/bin to PATH)" };
  return false;
}
for (const sig of ["SIGINT", "SIGTERM", "exit"]) process.on(sig, () => { qailProc?.kill(); if (sig !== "exit") process.exit(0); });

let qailCache = { at: 0, data: null };
async function qailModel(force = false) {
  if (!force && qailCache.data && Date.now() - qailCache.at < 3000) return qailCache.data;
  if (!(await ensureQail())) throw Object.assign(new Error(qailStatus.error), { status: 503 });
  const [cfg, repos] = await Promise.all([qail("/api/config"), qail("/api/repos")]);
  // Layout per qail CONTEXT.md: <Root>/<workspace>/<repo>
  const workspaces = Object.entries(cfg.workspaces || {}).map(([name, w]) => {
    const wsPath = path.join(cfg.root, name);
    return {
      name, path: wsPath, exists: existsSync(wsPath), lastUsed: w.lastUsed, editor: w.editor || cfg.defaultEditor || "",
      postInstall: w.postInstall || [],
      repos: (w.repos || []).map((r) => {
        const rp = path.join(wsPath, r);
        return { name: r, url: repos[r]?.url || "", path: rp, cloned: existsSync(rp),
          hasClaude: existsSync(path.join(rp, ".claude")), hasClaudeMd: existsSync(path.join(rp, "CLAUDE.md")) || existsSync(path.join(rp, "AGENTS.md")) };
      }),
    };
  }).sort((a, b) => String(b.lastUsed).localeCompare(String(a.lastUsed)));
  qailCache = { at: Date.now(), data: { root: cfg.root, workspaces, repos } };
  return qailCache.data;
}

async function resolveScope(ws, repo) {
  const model = await qailModel();
  const w = model.workspaces.find((x) => x.name === ws);
  if (!w) throw Object.assign(new Error(`no qail workspace "${ws}"`), { status: 404 });
  if (repo === WS_SCOPE) return { id: scopeId(ws, repo), ws: w, repo: null, path: w.path, exists: w.exists, label: ws };
  const r = w.repos.find((x) => x.name === repo);
  if (!r) throw Object.assign(new Error(`repo "${repo}" is not in workspace "${ws}"`), { status: 404 });
  return { id: scopeId(ws, repo), ws: w, repo: r, path: r.path, exists: r.cloned, label: `${ws} / ${repo}` };
}

// ---------- discovery ----------
const splitList = (v) =>
  Array.isArray(v) ? v.map(String) : typeof v === "string" ? v.split(",").map((s) => s.trim()).filter(Boolean) : [];

async function readMdDir(dir, pattern) {
  // pattern: "agents" => dir/*.md ; "skills" => dir/*/SKILL.md
  const out = [];
  if (!existsSync(dir)) return out;
  for (const entry of await fs.readdir(dir, { withFileTypes: true })) {
    let file;
    if (pattern === "agents" && entry.isFile() && entry.name.endsWith(".md")) file = path.join(dir, entry.name);
    if (pattern === "skills" && entry.isDirectory()) file = path.join(dir, entry.name, "SKILL.md");
    if (!file || !existsSync(file)) continue;
    try {
      const { data, content } = matter(await fs.readFile(file, "utf8"));
      out.push({ data, content, file, slug: pattern === "skills" ? entry.name : entry.name.replace(/\.md$/, "") });
    } catch (e) {
      out.push({ data: {}, content: "", file, slug: entry.name, error: String(e.message || e) });
    }
  }
  return out;
}

async function discoverSkills(projectPath) {
  const byName = new Map();
  for (const [scope, dir] of [["user", path.join(HOME_CLAUDE, "skills")], ["project", path.join(projectPath, ".claude", "skills")]]) {
    for (const s of await readMdDir(dir, "skills")) {
      const name = s.data.name || s.slug;
      byName.set(name, { name, description: s.data.description || "", scope, file: s.file, error: s.error });
    }
  }
  return [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
}

async function discoverAgents(projectPath, skills) {
  const byName = new Map();
  for (const [scope, dir] of [["user", path.join(HOME_CLAUDE, "agents")], ["project", path.join(projectPath, ".claude", "agents")]]) {
    for (const a of await readMdDir(dir, "agents")) {
      const name = a.data.name || a.slug;
      const tools = splitList(a.data.tools);
      const preloaded = splitList(a.data.skills);
      const canInvokeSkills = tools.length === 0 || tools.includes("Skill");
      byName.set(name, {
        name,
        scope,
        description: a.data.description || "",
        model: a.data.model || "inherit",
        tools: tools.length ? tools : ["(all tools)"],
        preloadedSkills: preloaded,
        missingSkills: preloaded.filter((p) => !skills.some((s) => s.name === p)),
        canInvokeSkills,
        discoverableSkills: canInvokeSkills ? skills.map((s) => s.name).filter((n) => !preloaded.includes(n)) : [],
        prompt: a.content.trim(),
        file: a.file,
        error: a.error,
      });
    }
  }
  const agents = [...byName.values()].sort((a, b) => a.name.localeCompare(b.name));
  agents.unshift({
    name: MAIN_AGENT,
    scope: "session",
    description: "Default Claude Code session in this scope. Can run /ship and dispatch every agent.",
    model: "default",
    tools: ["(all tools)"],
    preloadedSkills: [],
    missingSkills: [],
    canInvokeSkills: true,
    discoverableSkills: skills.map((s) => s.name),
    prompt: "",
  });
  return agents;
}

async function listRuns(projectPath) {
  const dir = path.join(projectPath, ".agent", "runs");
  if (!existsSync(dir)) return [];
  const runs = [];
  for (const e of await fs.readdir(dir, { withFileTypes: true })) {
    if (!e.isDirectory()) continue;
    const runDir = path.join(dir, e.name);
    let state = null;
    try { state = JSON.parse(await fs.readFile(path.join(runDir, "state.json"), "utf8")); } catch {}
    const files = (await fs.readdir(runDir)).filter((f) => !f.startsWith("."));
    const stat = await fs.stat(runDir);
    runs.push({ id: e.name, state, files, updated: state?.updated_at || stat.mtime.toISOString() });
  }
  return runs.sort((a, b) => String(b.updated).localeCompare(String(a.updated)));
}

// ---------- live chat sessions ----------
const live = new Map(); // threadKey -> { abort }
const permissionWaiters = new Map(); // permId -> { resolve, input, key }

function ensureThread(scope, agent) {
  const key = threadKey(scope, agent);
  store.threads[key] ??= { sessionId: null, messages: [], skillsUsed: [], updated: null };
  return store.threads[key];
}

// ---------- app ----------
const app = express();
app.use(express.json({ limit: "2mb" }));
app.use(express.static(path.join(__dirname, "public")));

// Reject cross-origin requests (another site in your browser can't drive local agents).
app.use("/api", (req, res, next) => {
  const origin = req.headers.origin;
  if (origin && !/^http:\/\/(127\.0\.0\.1|localhost):\d+$/.test(origin)) return res.status(403).json({ error: "cross-origin blocked" });
  next();
});

const wrap = (fn) => (req, res) => fn(req, res).catch((e) => res.status(e.status || 500).json({ error: String(e.message || e) }));

app.get("/api/qail/status", wrap(async (_req, res) => {
  await ensureQail();
  res.json(qailStatus);
}));

app.get("/api/workspaces", wrap(async (req, res) => {
  const model = await qailModel(req.query.fresh === "1");
  const busyScopes = new Set([...live.keys()].map((k) => k.split("::")[0]));
  res.json({
    root: model.root,
    qail: qailStatus,
    workspaces: model.workspaces.map((w) => ({
      ...w,
      busy: busyScopes.has(scopeId(w.name, WS_SCOPE)),
      repos: w.repos.map((r) => ({ ...r, busy: busyScopes.has(scopeId(w.name, r.name)) })),
    })),
  });
}));

// Proxy qail's "create on disk" (clones missing repos + runs post-install), streaming its SSE through.
app.post("/api/workspaces/:ws/create", wrap(async (req, res) => {
  await resolveScope(req.params.ws, WS_SCOPE);
  const upstream = await qail(`/api/workspaces/${encodeURIComponent(req.params.ws)}/create`, { method: "POST", raw: true });
  res.writeHead(200, { "Content-Type": "text/event-stream", "Cache-Control": "no-cache" });
  for await (const chunk of upstream.body) res.write(chunk);
  qailCache.at = 0;
  res.end();
}));

app.get("/api/scope/:ws/:repo/overview", wrap(async (req, res) => {
  const scope = await resolveScope(req.params.ws, req.params.repo);
  if (!scope.exists) return res.json({ scope: { ...scope, ws: undefined }, workspace: scope.ws, agents: [], skills: [], runs: [], hasClaudeMd: [], missing: true });
  const skills = await discoverSkills(scope.path);
  const agents = await discoverAgents(scope.path, skills);
  for (const a of agents) {
    const t = store.threads[threadKey(scope.id, a.name)];
    a.thread = t ? { count: t.messages.length, skillsUsed: t.skillsUsed, updated: t.updated } : null;
    if (t?.runtimeSkills?.length && a.canInvokeSkills) a.discoverableSkills = t.runtimeSkills.filter((n) => !a.preloadedSkills.includes(n));
    a.busy = live.has(threadKey(scope.id, a.name));
  }
  for (const s of skills) s.preloadedBy = agents.filter((a) => a.preloadedSkills.includes(s.name)).map((a) => a.name);
  const hasClaudeMd = ["CLAUDE.md", "AGENTS.md"].filter((f) => existsSync(path.join(scope.path, f)));
  res.json({ scope: { id: scope.id, label: scope.label, path: scope.path, repo: scope.repo }, workspace: scope.ws, agents, skills, runs: await listRuns(scope.path), hasClaudeMd });
}));

app.get("/api/scope/:ws/:repo/runs/:run/:file", wrap(async (req, res) => {
  const scope = await resolveScope(req.params.ws, req.params.repo);
  const base = path.join(scope.path, ".agent", "runs");
  const f = path.resolve(base, req.params.run, req.params.file);
  if (!f.startsWith(base + path.sep)) return res.status(400).end();
  try { res.type("text/plain").send(await fs.readFile(f, "utf8")); } catch { res.status(404).end(); }
}));

app.get("/api/scope/:ws/:repo/threads/:agent", wrap(async (req, res) => {
  const t = store.threads[threadKey(scopeId(req.params.ws, req.params.repo), req.params.agent)];
  res.json(t || { sessionId: null, messages: [], skillsUsed: [] });
}));

app.delete("/api/scope/:ws/:repo/threads/:agent", wrap(async (req, res) => {
  delete store.threads[threadKey(scopeId(req.params.ws, req.params.repo), req.params.agent)];
  await saveStore();
  res.json({ ok: true });
}));

app.post("/api/permissions/:permId", (req, res) => {
  const w = permissionWaiters.get(req.params.permId);
  if (!w) return res.status(404).json({ error: "expired" });
  permissionWaiters.delete(req.params.permId);
  w.resolve(req.body?.allow
    ? { behavior: "allow", updatedInput: w.input }
    : { behavior: "deny", message: req.body?.message || "Denied by user in Agent Console" });
  res.json({ ok: true });
});

app.post("/api/scope/:ws/:repo/threads/:agent/stop", (req, res) => {
  live.get(threadKey(scopeId(req.params.ws, req.params.repo), req.params.agent))?.abort.abort();
  res.json({ ok: true });
});

app.post("/api/scope/:ws/:repo/threads/:agent/chat", async (req, res) => {
  let scope;
  try { scope = await resolveScope(req.params.ws, req.params.repo); } catch (e) { return res.status(e.status || 500).json({ error: e.message }); }
  if (!scope.exists) return res.status(409).json({ error: "Not cloned yet — create the workspace on disk first." });
  const agent = req.params.agent;
  const { message, mode = "default" } = req.body || {};
  if (!message) return res.status(400).json({ error: "message required" });
  const key = threadKey(scope.id, agent);
  if (live.has(key)) return res.status(409).json({ error: "agent is busy in this scope" });

  const thread = ensureThread(scope.id, agent);
  thread.messages.push({ role: "user", text: message, ts: new Date().toISOString() });

  res.writeHead(200, { "Content-Type": "text/event-stream", "Cache-Control": "no-cache", Connection: "keep-alive" });
  const send = (type, data) => res.write(`data: ${JSON.stringify({ type, ...data })}\n\n`);

  const abort = new AbortController();
  live.set(key, { abort });
  res.on("close", () => { if (live.has(key)) abort.abort(); });

  const reply = { role: "assistant", text: "", tools: [], skills: [], ts: null, cost: null };
  const READ_ONLY = new Set(["Read", "Grep", "Glob", "LS", "TodoWrite", "Skill", "WebSearch", "WebFetch", "NotebookRead"]);

  // Workspace scope: tell the agent how the qail workspace is laid out.
  const wsContext = scope.repo ? "" :
    `\n\nYou are working at the root of the qail workspace "${scope.ws.name}" (${scope.ws.path}). ` +
    `It contains these repos as subdirectories: ${scope.ws.repos.map((r) => `${r.name}${r.cloned ? "" : " (not cloned)"}`).join(", ")}. ` +
    `Each repo has its own CLAUDE.md, .claude/skills and conventions — read the ones for the repo you touch before changing it.`;

  const options = {
    cwd: scope.path,
    settingSources: ["user", "project", "local"],
    skills: "all",
    permissionMode: ["default", "acceptEdits", "plan"].includes(mode) ? mode : "default",
    includePartialMessages: true,
    abortController: abort,
    systemPrompt: { type: "preset", preset: "claude_code", ...(wsContext ? { append: wsContext.trim() } : {}) },
    canUseTool: (toolName, input, { signal }) => {
      if (READ_ONLY.has(toolName)) return Promise.resolve({ behavior: "allow", updatedInput: input });
      const permId = crypto.randomUUID();
      send("permission", { permId, tool: toolName, input });
      return new Promise((resolve) => {
        permissionWaiters.set(permId, { resolve, input, key });
        const timeout = setTimeout(() => {
          if (permissionWaiters.delete(permId)) resolve({ behavior: "deny", message: "No response in 10 minutes" });
        }, 10 * 60 * 1000);
        signal?.addEventListener("abort", () => {
          clearTimeout(timeout);
          if (permissionWaiters.delete(permId)) resolve({ behavior: "deny", message: "Aborted", interrupt: true });
        });
      });
    },
  };
  if (agent !== MAIN_AGENT) options.agent = agent;
  if (thread.sessionId) options.resume = thread.sessionId;

  try {
    for await (const msg of query({ prompt: message, options })) {
      if (msg.type === "system" && msg.subtype === "init") {
        thread.sessionId = msg.session_id;
        thread.runtimeSkills = msg.skills || [];
        send("init", { model: msg.model, skills: msg.skills, tools: msg.tools, permissionMode: msg.permissionMode, sessionId: msg.session_id });
      } else if (msg.type === "stream_event") {
        const ev = msg.event;
        if (ev?.type === "content_block_delta" && ev.delta?.type === "text_delta" && !msg.parent_tool_use_id) send("delta", { text: ev.delta.text });
      } else if (msg.type === "assistant") {
        for (const block of msg.message?.content || []) {
          if (block.type === "text" && !msg.parent_tool_use_id) {
            reply.text += (reply.text ? "\n\n" : "") + block.text;
            send("text_done", {});
          } else if (block.type === "tool_use") {
            const tool = { id: block.id, name: block.name, input: block.input, sub: !!msg.parent_tool_use_id };
            reply.tools.push(tool);
            send("tool", tool);
            if (block.name === "Skill") {
              const skill = block.input?.skill || block.input?.command || block.input?.name;
              if (skill) {
                if (!reply.skills.includes(skill)) reply.skills.push(skill);
                if (!thread.skillsUsed.includes(skill)) thread.skillsUsed.push(skill);
                send("skill", { skill });
              }
            }
          }
        }
      } else if (msg.type === "user") {
        for (const block of Array.isArray(msg.message?.content) ? msg.message.content : []) {
          if (block.type === "tool_result") {
            const content = Array.isArray(block.content) ? block.content.map((c) => c.text || "").join("\n") : String(block.content ?? "");
            send("tool_result", { id: block.tool_use_id, error: !!block.is_error, preview: content.slice(0, 1500) });
          }
        }
      } else if (msg.type === "result") {
        reply.cost = msg.total_cost_usd ?? null;
        send("result", { subtype: msg.subtype, cost: msg.total_cost_usd, turns: msg.num_turns, durationMs: msg.duration_ms, isError: msg.is_error });
      }
    }
  } catch (e) {
    const aborted = abort.signal.aborted;
    send("error", { message: aborted ? "Stopped." : String(e?.message || e) });
    if (!reply.text) reply.text = aborted ? "_Stopped._" : `**Error:** ${e?.message || e}`;
  } finally {
    for (const [id, w] of permissionWaiters) if (w.key === key) { permissionWaiters.delete(id); w.resolve({ behavior: "deny", message: "Session ended" }); }
    live.delete(key);
    reply.ts = new Date().toISOString();
    reply.tools = reply.tools.map(({ id, name, input, sub }) => ({ id, name, sub, input: JSON.stringify(input).slice(0, 400) }));
    thread.messages.push(reply);
    thread.updated = reply.ts;
    await saveStore();
    send("done", {});
    res.end();
  }
});

await loadStore();
ensureQail().then((ok) => console.log(ok ? `qail → ${QAIL_URL}${qailStatus.spawned ? " (started by console)" : ""}` : `qail: ${qailStatus.error}`));
app.listen(PORT, HOST, () => console.log(`Agent Console → http://${HOST}:${PORT}`));
