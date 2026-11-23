const $ = (s) => document.querySelector(s);
const esc = (s) => String(s ?? "").replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
const api = async (url, opts = {}) => {
  const r = await fetch(url, { headers: { "Content-Type": "application/json" }, ...opts, body: opts.body ? JSON.stringify(opts.body) : undefined });
  if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText);
  return r.json();
};

const MODEL_COLOR = { opus: "var(--opus)", sonnet: "var(--sonnet)", haiku: "var(--haiku)" };
const modelColor = (m = "") => Object.entries(MODEL_COLOR).find(([k]) => m.includes(k))?.[1] || "var(--accent)";
const initials = (n) => n === "main" ? "◆" : n.split(/[-_\s]/).map((w) => w[0]).join("").slice(0, 2).toUpperCase();
const ago = (iso) => {
  if (!iso) return "";
  const s = (Date.now() - new Date(iso)) / 1000;
  if (s < 60) return "just now"; if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`; return `${Math.floor(s / 86400)}d ago`;
};

// --- tiny markdown renderer (escape first, then format) ---
function md(src = "") {
  const blocks = [];
  let s = esc(src).replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) => { blocks.push(`<pre><code>${code}</code></pre>`); return `\u0000${blocks.length - 1}\u0000`; });
  const inline = (t) => t
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/(^|[^*])\*([^*\n]+)\*/g, "$1<em>$2</em>")
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g, '<a href="$2" target="_blank" rel="noopener">$1</a>');
  const out = [];
  const lines = s.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const l = lines[i];
    if (/^\u0000\d+\u0000$/.test(l.trim())) { out.push(blocks[+l.trim().slice(1, -1)]); continue; }
    const h = l.match(/^(#{1,3})\s+(.*)/);
    if (h) { out.push(`<h${h[1].length}>${inline(h[2])}</h${h[1].length}>`); continue; }
    if (/^\s*\|.*\|\s*$/.test(l) && /^\s*\|[\s:|-]+\|\s*$/.test(lines[i + 1] || "")) {
      const row = (r, tag) => "<tr>" + r.trim().slice(1, -1).split("|").map((c) => `<${tag}>${inline(c.trim())}</${tag}>`).join("") + "</tr>";
      let html = "<table>" + row(l, "th"); i += 2;
      while (i < lines.length && /^\s*\|.*\|\s*$/.test(lines[i])) html += row(lines[i++], "td");
      i--; out.push(html + "</table>"); continue;
    }
    if (/^\s*([-*]|\d+\.)\s+/.test(l)) {
      const ordered = /^\s*\d+\./.test(l); const items = [];
      while (i < lines.length && /^\s*([-*]|\d+\.)\s+/.test(lines[i])) items.push(`<li>${inline(lines[i++].replace(/^\s*([-*]|\d+\.)\s+/, ""))}</li>`);
      i--; out.push(`<${ordered ? "ol" : "ul"}>${items.join("")}</${ordered ? "ol" : "ul"}>`); continue;
    }
    if (l.trim()) out.push(`<p>${inline(l)}</p>`);
  }
  return out.join("");
}

// --- state ---
const state = { model: null, ws: null, repo: null, overview: null, agent: null, streaming: false, open: new Set() };
const enc = encodeURIComponent;
const base = () => `/api/scope/${enc(state.ws)}/${enc(state.repo || "_")}`;
const curWs = () => state.model?.workspaces.find((w) => w.name === state.ws);

async function loadWorkspaces(fresh) {
  try {
    state.model = await api(`/api/workspaces${fresh ? "?fresh=1" : ""}`);
  } catch (e) {
    state.model = null;
    $("#qailStatus").className = "qail-status bad";
    $("#qailStatus").innerHTML = `<span class="dot"></span><span>${esc(e.message)}</span>`;
    $("#wsTree").innerHTML = "";
    $("#emptyTitle").textContent = "Can't read qail's workspaces";
    $("#emptyText").innerHTML = esc(e.message);
    return;
  }
  const c = state.model.claude || {};
  $("#qailStatus").className = `qail-status ${c.ok ? "ok" : "bad"}`;
  $("#qailStatus").innerHTML = c.ok
    ? `<span class="dot"></span><span>root <code>${esc(state.model.root)}</code><br/>claude <code>${esc(c.path)}</code></span>`
    : `<span class="dot"></span><span>${esc(c.error || "claude CLI not found")}<br/>Agents need <a href="https://docs.anthropic.com/en/docs/claude-code" target="_blank" rel="noopener">Claude Code</a> installed.</span>`;
  if (!state.ws && state.model.workspaces[0]) state.open.add(state.model.workspaces[0].name);
  renderTree();
  if (!state.model.workspaces.length) { $("#emptyTitle").textContent = "No qail workspaces yet"; $("#emptyText").innerHTML = "Create one with <code>qail workspace add</code>."; }
}

function renderTree() {
  $("#wsTree").innerHTML = state.model.workspaces.map((w) => `
    <div class="ws-node ${state.open.has(w.name) ? "open" : ""}" data-ws="${esc(w.name)}">
      <button class="ws-btn ${state.ws === w.name && !state.repo ? "active" : ""}" data-ws="${esc(w.name)}">
        <span class="caret">▶</span><span>${esc(w.name)}</span>${w.busy ? '<span class="dotsm busy"></span>' : ""}
        <span class="ws-meta">${w.repos.length} repo${w.repos.length === 1 ? "" : "s"}</span></button>
      <div class="repos">${w.repos.map((r) => `
        <button class="repo-btn ${state.ws === w.name && state.repo === r.name ? "active" : ""}" data-ws="${esc(w.name)}" data-repo="${esc(r.name)}" title="${esc(r.path)}">
          <span class="dotsm ${r.busy ? "busy" : !r.cloned ? "uncloned" : r.hasClaude ? "claude" : ""}"></span>${esc(r.name)}</button>`).join("")}
      </div>
    </div>`).join("");
  document.querySelectorAll(".ws-btn").forEach((b) => (b.onclick = (e) => {
    const ws = b.dataset.ws;
    if (e.target.classList.contains("caret")) { state.open.has(ws) ? state.open.delete(ws) : state.open.add(ws); return renderTree(); }
    state.open.add(ws); selectScope(ws, null);
  }));
  document.querySelectorAll(".repo-btn").forEach((b) => (b.onclick = () => selectScope(b.dataset.ws, b.dataset.repo)));
}

async function selectScope(ws, repo) {
  state.ws = ws; state.repo = repo; closeChat(); renderTree();
  history.replaceState(null, "", `#/ws/${enc(ws)}${repo ? `/${enc(repo)}` : ""}`);
  $("#empty").classList.add("hidden"); $("#projectView").classList.remove("hidden");
  $("#reposTabBtn").classList.toggle("hidden", !!repo);
  showTab(repo ? "agents" : "repos");
  await refresh();
}

function showTab(name) {
  document.querySelectorAll(".tab").forEach((x) => x.classList.toggle("active", x.dataset.tab === name));
  ["repos", "agents", "skills", "runs"].forEach((n) => $(`#tab-${n}`).classList.toggle("hidden", n !== name));
}

async function refresh() {
  if (!state.ws) return;
  await loadWorkspaces();
  if (!state.model) return;
  const o = (state.overview = await api(`${base()}/overview`));
  const w = curWs();
  $("#crumbs").innerHTML = state.repo ? `<a id="crumbWs">${esc(w.name)}</a><span>/</span><span>repo</span>` : `<span>qail workspace</span>`;
  $("#crumbWs") && ($("#crumbWs").onclick = () => selectScope(w.name, null));
  $("#projName").textContent = state.repo || w.name;
  const repo = state.repo && w.repos.find((r) => r.name === state.repo);
  $("#projPath").textContent = o.scope?.path || (repo ? repo.path : w.path);
  const missingRepos = w.repos.filter((r) => !r.cloned);
  $("#createBtn").classList.toggle("hidden", !missingRepos.length);
  $("#missing").classList.toggle("hidden", !o.missing);
  if (o.missing) $("#missing").innerHTML = `Not on disk yet. <b>Clone missing repos</b> runs <code>qail</code>'s workspace create (clone + post-install scripts).`;
  const b = $("#claudeMdBadge");
  b.textContent = o.hasClaudeMd.length ? `${o.hasClaudeMd.join(" + ")} found` : "no CLAUDE.md";
  b.className = "pill " + (o.hasClaudeMd.length ? "good" : "warn");
  const realAgents = o.agents.filter((a) => a.name !== "main");
  const preloadedCount = new Set(realAgents.flatMap((a) => a.preloadedSkills)).size;
  $("#stats").innerHTML = (state.repo ? [
    [realAgents.length, "agents"], [o.skills.length, "skills available"], [preloadedCount, "skills preloaded by agents"],
    [o.runs.filter((r) => r.state?.status === "running").length + " / " + o.runs.length, "/ship runs active / total"],
  ] : [
    [w.repos.length, "repos"], [`${w.repos.filter((r) => r.cloned).length} / ${w.repos.length}`, "cloned"],
    [w.repos.filter((r) => r.hasClaude).length, "repos with .claude/"], [realAgents.length, "workspace-level agents"],
  ]).map(([n, l]) => `<div class="stat"><div class="n">${n}</div><div class="l">${l}</div></div>`).join("");
  renderRepos(); renderAgents(); renderSkills(); renderRuns();
}

function renderRepos() {
  const w = curWs(); if (!w || state.repo) return;
  $("#repoGrid").innerHTML = w.repos.map((r) => `
    <div class="repo-card" data-repo="${esc(r.name)}">
      <div style="display:flex;gap:8px;align-items:center"><span class="dotsm ${r.busy ? "busy" : !r.cloned ? "uncloned" : r.hasClaude ? "claude" : ""}"></span><span class="rn">${esc(r.name)}</span></div>
      <div class="ru">${esc(r.url || "no remote registered")}</div>
      <div class="chips">
        <span class="chip ${r.cloned ? "" : "warnchip"}">${r.cloned ? "cloned" : "not cloned"}</span>
        ${r.hasClaude ? '<span class="chip skill">.claude/</span>' : ""}
        ${r.hasClaudeMd ? '<span class="chip">CLAUDE.md</span>' : ""}
      </div>
    </div>`).join("") || '<div class="muted">This workspace has no repos.</div>';
  document.querySelectorAll(".repo-card").forEach((c) => (c.onclick = () => selectScope(state.ws, c.dataset.repo)));
}

function renderAgents() {
  const { agents } = state.overview;
  $("#agentGrid").innerHTML = agents.map((a) => {
    const used = a.thread?.skillsUsed || [];
    const disc = a.discoverableSkills.filter((s) => !used.includes(s));
    return `
    <div class="card ${state.agent === a.name ? "selected" : ""}" data-agent="${esc(a.name)}" style="--m:${modelColor(a.model)}">
      <div class="card-head">
        <div class="avatar">${esc(initials(a.name))}</div>
        <div><div class="card-title">${esc(a.name)}</div>
          <div class="badges"><span class="badge model">${esc(a.model)}</span><span class="badge">${esc(a.scope)}</span></div></div>
        ${a.busy ? '<div class="busy-dot" title="working"></div>' : ""}
      </div>
      <div class="card-desc">${esc(a.description)}</div>
      <div><div class="sect-label">Skills</div><div class="chips">
        ${a.preloadedSkills.map((s) => `<span class="chip skill ${a.missingSkills.includes(s) ? "missing" : ""}" title="${a.missingSkills.includes(s) ? "not found in this project" : "preloaded"}">${esc(s)}</span>`).join("")}
        ${used.filter((s) => !a.preloadedSkills.includes(s)).map((s) => `<span class="chip skill used" title="invoked in this project">${esc(s)}</span>`).join("")}
        ${a.canInvokeSkills ? (disc.length ? `<span class="chip more">+${disc.length} discoverable</span>` : "") : '<span class="chip more">no Skill tool</span>'}
        ${!a.preloadedSkills.length && !used.length && !a.canInvokeSkills ? "" : ""}
      </div></div>
      <div><div class="sect-label">Tools</div><div class="chips">${a.tools.map((t) => `<span class="chip">${esc(t)}</span>`).join("")}</div></div>
      <div class="card-foot"><span>${a.thread ? `${a.thread.count} messages · ${ago(a.thread.updated)}` : "No conversation yet"}</span><span class="talk">Talk →</span></div>
    </div>`;
  }).join("");
  document.querySelectorAll(".card").forEach((c) => (c.onclick = () => openChat(c.dataset.agent)));
}

function renderSkills() {
  const { skills } = state.overview;
  $("#skillList").innerHTML = skills.length ? skills.map((s) => `
    <div class="skill-card">
      <div class="skill-head"><span class="skill-name">${esc(s.name)}</span><span class="badge">${esc(s.scope)}</span></div>
      <div class="skill-desc">${esc(s.description) || '<span class="muted">No description — agents can\'t match it to tasks.</span>'}</div>
      <div class="sect-label">Preloaded by</div>
      <div class="chips">${s.preloadedBy.length ? s.preloadedBy.map((a) => `<span class="chip">${esc(a)}</span>`).join("") : '<span class="chip more">on demand only</span>'}</div>
    </div>`).join("") : '<div class="muted">No skills in this project or ~/.claude/skills.</div>';
}

function renderRuns() {
  const { runs } = state.overview;
  $("#runList").innerHTML = runs.length ? runs.map((r) => `
    <button class="run" data-run="${esc(r.id)}">
      <div class="run-id">${esc(r.id)}</div>
      <div class="run-meta"><span class="status ${esc(r.state?.status || "")}">${esc(r.state?.status || "unknown")}</span>
        <span>${esc(r.state?.phase || "")}${r.state?.iteration ? ` · iter ${r.state.iteration}` : ""}</span><span>${ago(r.updated)}</span></div>
    </button>`).join("") : '<div class="muted">No runs yet. Ask the <b>main</b> agent to <code>/ship</code> something here.</div>';
  document.querySelectorAll(".run").forEach((b) => (b.onclick = () => openRun(b.dataset.run)));
}

async function openRun(id) {
  document.querySelectorAll(".run").forEach((b) => b.classList.toggle("active", b.dataset.run === id));
  const run = state.overview.runs.find((r) => r.id === id);
  const order = ["summary.md", "plan.md", "context.md", "task.md"];
  const files = [...run.files].sort((a, b) => (order.indexOf(a) + 1 || 99) - (order.indexOf(b) + 1 || 99) || a.localeCompare(b));
  $("#runViewer").innerHTML = `<div class="file-tabs">${files.map((f) => `<button class="file-tab" data-f="${esc(f)}">${esc(f)}</button>`).join("")}</div><div class="file-body body" id="fileBody"></div>`;
  const show = async (f) => {
    document.querySelectorAll(".file-tab").forEach((b) => b.classList.toggle("active", b.dataset.f === f));
    const txt = await (await fetch(`${base()}/runs/${enc(id)}/${enc(f)}`)).text();
    $("#fileBody").innerHTML = f.endsWith(".json") ? `<pre>${esc(txt)}</pre>` : md(txt);
  };
  document.querySelectorAll(".file-tab").forEach((b) => (b.onclick = () => show(b.dataset.f)));
  if (files[0]) show(files[0]);
}

// --- chat ---
function agentInfo() { return state.overview.agents.find((a) => a.name === state.agent); }

async function openChat(name) {
  state.agent = name; renderAgents();
  const a = agentInfo();
  $("#chat").classList.remove("hidden");
  $("#chat").style.setProperty("--m", modelColor(a.model));
  $("#chatAvatar").textContent = initials(a.name);
  $("#chatName").textContent = a.name;
  $("#chatMeta").textContent = `${a.model} · ${state.repo ? `${state.ws} / ${state.repo}` : `${state.ws} (whole workspace)`}`;
  const thread = await api(`${base()}/threads/${enc(name)}`);
  renderChatSkills(thread.skillsUsed);
  const box = $("#messages"); box.innerHTML = "";
  if (!thread.messages.length) box.innerHTML = `<div class="empty-chat">Start a conversation with <b>${esc(a.name)}</b>.<br/>It runs in <code>${esc(state.overview.scope.path)}</code>${state.repo ? " with this repo's CLAUDE.md, skills and hooks" : " across every repo in the workspace"}.</div>`;
  for (const m of thread.messages) appendMessage(m);
  box.scrollTop = box.scrollHeight;
  $("#input").focus();
}

function renderChatSkills(used = []) {
  const a = agentInfo();
  const live = [...new Set([...(used || [])])];
  $("#chatSkills").innerHTML = `
    <div><div class="sect-label">Preloaded</div><div class="chips">${a.preloadedSkills.length ? a.preloadedSkills.map((s) => `<span class="chip skill">${esc(s)}</span>`).join("") : '<span class="chip more">none</span>'}</div></div>
    <div><div class="sect-label">Used in this project</div><div class="chips" id="usedChips">${live.length ? live.map((s) => `<span class="chip skill used">${esc(s)}</span>`).join("") : `<span class="chip more">${a.canInvokeSkills ? "none yet — picked up on demand" : "agent has no Skill tool"}</span>`}</div></div>`;
}

function closeChat() { state.agent = null; $("#chat").classList.add("hidden"); if (state.overview) renderAgents(); }

function appendMessage(m) {
  const box = $("#messages");
  box.querySelector(".empty-chat")?.remove();
  const el = document.createElement("div");
  el.className = `msg ${m.role}`;
  if (m.role === "user") el.innerHTML = `${esc(m.text)}<div class="ts">${ago(m.ts)}</div>`;
  else {
    const tools = (m.tools || []).map((t) => eventHtml(t.name, t.input, t.sub)).join("");
    el.innerHTML = `${tools ? `<div class="events">${tools}</div>` : ""}<div class="body">${md(m.text) || '<span class="muted">(no text)</span>'}</div>
      <div class="ts">${ago(m.ts)}${m.cost != null ? ` · $${m.cost.toFixed(4)}` : ""}</div>`;
  }
  box.appendChild(el);
  return el;
}

function eventHtml(name, input, sub, id) {
  const isSkill = name === "Skill";
  let label = "";
  try { const o = typeof input === "string" ? JSON.parse(input) : input; label = o?.skill || o?.file_path || o?.pattern || o?.command || o?.subagent_type || o?.description || ""; } catch { label = ""; }
  const pretty = typeof input === "string" ? input : JSON.stringify(input, null, 2);
  return `<details class="event ${isSkill ? "skill" : ""} ${sub ? "sub" : ""}" ${id ? `data-id="${id}"` : ""}>
    <summary>${id ? '<span class="spinner"></span>' : ""}<span class="tname">${isSkill ? "⚡ skill" : esc(name)}</span><span>${esc(String(label).slice(0, 80))}</span>${sub ? '<span class="badge">subagent</span>' : ""}</summary>
    <pre>${esc(pretty)}</pre></details>`;
}

async function send(text) {
  if (state.streaming || !text.trim()) return;
  const agent = state.agent, scopeBase = base();
  appendMessage({ role: "user", text, ts: new Date().toISOString() });
  const box = $("#messages");
  const el = document.createElement("div");
  el.className = "msg assistant";
  el.innerHTML = `<div class="events"></div><div class="body hidden"></div><div class="typing"><span></span><span></span><span></span></div>`;
  box.appendChild(el); box.scrollTop = box.scrollHeight;
  const events = el.querySelector(".events"), body = el.querySelector(".body"), typing = el.querySelector(".typing");
  let text_ = "", committed = "";
  setStreaming(true);

  const scroll = () => { if (box.scrollHeight - box.scrollTop - box.clientHeight < 200) box.scrollTop = box.scrollHeight; };
  const handle = (ev) => {
    switch (ev.type) {
      case "delta": typing.classList.add("hidden"); body.classList.remove("hidden"); text_ += ev.text; body.innerHTML = md(committed + text_); break;
      case "text_done": committed += text_ + "\n\n"; text_ = ""; break;
      case "tool": events.insertAdjacentHTML("beforeend", eventHtml(ev.name, ev.input, ev.sub, ev.id)); break;
      case "tool_result": {
        const d = events.querySelector(`[data-id="${CSS.escape(ev.id)}"]`);
        if (d) { d.querySelector(".spinner")?.replaceWith(Object.assign(document.createElement("span"), { className: ev.error ? "err" : "ok", textContent: ev.error ? "✕" : "✓" }));
          d.querySelector("pre").textContent += "\n\n— result —\n" + ev.preview; }
        break;
      }
      case "skill": {
        const u = $("#usedChips");
        if (u && ![...u.children].some((c) => c.textContent === ev.skill)) { u.querySelector(".more")?.remove(); u.insertAdjacentHTML("beforeend", `<span class="chip skill used">${esc(ev.skill)}</span>`); }
        break;
      }
      case "permission": {
        const p = document.createElement("div"); p.className = "perm";
        p.innerHTML = `<div class="perm-title">${esc(ev.tool)} wants to run</div><pre>${esc(ev.input?.command || ev.input?.file_path || JSON.stringify(ev.input, null, 2))}</pre>
          <div class="row"><button class="allow">Allow</button><button class="deny">Deny</button></div>`;
        const decide = async (allow) => { p.querySelector(".row").innerHTML = `<span class="muted">${allow ? "Allowed" : "Denied"}</span>`; await api(`/api/permissions/${ev.permId}`, { method: "POST", body: { allow } }); };
        p.querySelector(".allow").onclick = () => decide(true); p.querySelector(".deny").onclick = () => decide(false);
        events.appendChild(p); break;
      }
      case "result": el.insertAdjacentHTML("beforeend", `<div class="result-line">${ev.turns ?? "?"} turns · ${((ev.durationMs || 0) / 1000).toFixed(1)}s${ev.cost != null ? ` · $${ev.cost.toFixed(4)}` : ""}</div>`); break;
      case "error": body.classList.remove("hidden"); body.innerHTML = md(committed + text_) + `<p style="color:var(--bad)">${esc(ev.message)}</p>`; break;
    }
    scroll();
  };

  try {
    const r = await fetch(`${scopeBase}/threads/${enc(agent)}/chat`, {
      method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ message: text, mode: $("#mode").value }),
    });
    if (!r.ok) throw new Error((await r.json().catch(() => ({}))).error || r.statusText);
    const reader = r.body.getReader(); const dec = new TextDecoder(); let buf = "";
    for (;;) {
      const { value, done } = await reader.read(); if (done) break;
      buf += dec.decode(value, { stream: true });
      let i; while ((i = buf.indexOf("\n\n")) >= 0) { const chunk = buf.slice(0, i); buf = buf.slice(i + 2); if (chunk.startsWith("data: ")) handle(JSON.parse(chunk.slice(6))); }
    }
  } catch (e) { handle({ type: "error", message: e.message }); }
  typing.remove(); setStreaming(false);
  if (!body.innerHTML) { body.classList.remove("hidden"); body.innerHTML = '<span class="muted">(no text)</span>'; }
  refresh();
}

function setStreaming(on) {
  state.streaming = on;
  $("#sendBtn").disabled = on; $("#stopBtn").classList.toggle("hidden", !on);
}

// --- wiring ---
$("#composer").onsubmit = (e) => { e.preventDefault(); const t = $("#input").value; $("#input").value = ""; send(t); };
$("#input").onkeydown = (e) => { if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); $("#composer").requestSubmit(); } };
$("#stopBtn").onclick = () => api(`${base()}/threads/${enc(state.agent)}/stop`, { method: "POST" });
$("#closeChat").onclick = closeChat;
$("#newThreadBtn").onclick = async () => { if (state.streaming) return; await api(`${base()}/threads/${enc(state.agent)}`, { method: "DELETE" }); openChat(state.agent); refresh(); };
$("#refreshBtn").onclick = async () => { await loadWorkspaces(true); refresh(); };
$("#createBtn").onclick = async () => {
  const btn = $("#createBtn"); btn.disabled = true; btn.textContent = "Cloning…";
  $("#missing").classList.remove("hidden"); $("#missing").innerHTML = `Running qail workspace create…<div class="log" id="createLog"></div>`;
  const r = await fetch(`/api/workspaces/${enc(state.ws)}/create`, { method: "POST" });
  const reader = r.body.getReader(), dec = new TextDecoder();
  for (;;) { const { value, done } = await reader.read(); if (done) break;
    $("#createLog").textContent += dec.decode(value, { stream: true }).replace(/^(data|event): ?/gm, "").replace(/\n\n/g, "\n");
    $("#createLog").scrollTop = 1e9; }
  btn.disabled = false; btn.textContent = "Clone missing repos";
  await loadWorkspaces(true); refresh();
};
document.querySelectorAll(".tab").forEach((t) => (t.onclick = () => showTab(t.dataset.tab)));

// Deep links from the qail tray / `qail console <ws> [repo]`: #/ws/<workspace>[/<repo>]
function scopeFromHash() {
  const m = location.hash.match(/^#\/ws\/([^/]+)(?:\/([^/]+))?/);
  return m ? { ws: decodeURIComponent(m[1]), repo: m[2] ? decodeURIComponent(m[2]) : null } : null;
}
async function openFromHash() {
  const t = scopeFromHash();
  if (!t || !state.model) return;
  const w = state.model.workspaces.find((x) => x.name === t.ws);
  if (!w || (t.repo && !w.repos.some((r) => r.name === t.repo))) return;
  if (t.ws === state.ws && t.repo === state.repo) return;
  state.open.add(t.ws); await selectScope(t.ws, t.repo);
}
window.addEventListener("hashchange", openFromHash);
loadWorkspaces().then(openFromHash);
setInterval(() => { if (!state.streaming) (state.ws ? refresh() : loadWorkspaces()); }, 15000);
