"use strict";

// MCP Apps protocol 2026-01-26, verified against ext-apps v2.0.3.
// Conversation recovery is sent only by the user clicking Recover in chat.
class HostBridge {
  constructor(parentWindow, listen, timeoutMs = 15000) {
    this.parent = parentWindow;
    this.timeoutMs = timeoutMs;
    this.sequence = 0;
    this.closed = false;
    this.pending = new Map();
    this.onResult = () => {};
    this.onContext = () => {};
    listen((event) => this.receive(event));
  }
  receive(event) {
    const message = event.data;
    if (event.source !== this.parent || !message || message.jsonrpc !== "2.0") return;
    if (message.id !== undefined && !message.method) {
      const request = this.pending.get(message.id);
      if (!request) return;
      clearTimeout(request.timer);
      this.pending.delete(message.id);
      if (message.error) request.reject(new Error(message.error.message || "Host request failed."));
      else request.resolve(message.result);
    } else if (message.method === "ui/notifications/tool-result") {
      this.onResult(message.params);
    } else if (message.method === "ui/notifications/host-context-changed") {
      this.onContext(message.params);
    } else if (message.method === "ping" || message.method === "ui/resource-teardown") {
      if (message.id !== undefined) this.parent.postMessage({ jsonrpc: "2.0", id: message.id, result: {} }, "*");
      if (message.method === "ui/resource-teardown") this.close();
    }
  }
  close() {
    this.closed = true;
    for (const request of this.pending.values()) {
      clearTimeout(request.timer);
      request.reject(new Error("Panel closed. Reopen Seshy to continue."));
    }
    this.pending.clear();
  }
  request(method, params) {
    if (this.closed) return Promise.reject(new Error("Panel closed."));
    const id = ++this.sequence;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error("Host timed out. Refresh the selected session and try again."));
      }, this.timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      this.parent.postMessage({ jsonrpc: "2.0", id, method, params }, "*");
    });
  }
  async connect() {
    const result = await this.request("ui/initialize", {
      appInfo: { name: "Seshy", version: "0.1.2" },
      appCapabilities: { availableDisplayModes: ["inline", "fullscreen", "pip"] },
      protocolVersion: "2026-01-26",
    });
    if (!result?.hostCapabilities?.serverTools) throw new Error("This host does not offer direct app tools.");
    this.parent.postMessage({ jsonrpc: "2.0", method: "ui/notifications/initialized" }, "*");
    return result;
  }
  async tool(name, args = {}) {
    const result = await this.request("tools/call", { name, arguments: args });
    if (result?.isError) throw new Error("The read could not complete. " + (result.content?.find((c) => c.type === "text")?.text || "Check its inputs and current revision."));
    if (result?.structuredContent) return result.structuredContent;
    const text = result?.content?.find((c) => c.type === "text")?.text;
    if (text) return JSON.parse(text);
    throw new Error("The host returned no recovery result.");
  }
}

if (typeof module !== "undefined") module.exports = { HostBridge };

if (typeof document !== "undefined") {
  const $ = id => document.getElementById(id);
  const bridge = new HostBridge(window.parent, fn => window.addEventListener("message", fn));
  const rows = new Map();
  const agentNames = {claude: "Claude", codex: "Codex", grok: "Grok", pi: "pi", opencode: "OpenCode", agy: "Antigravity", droid: "Droid"};
  let connected = false, busy = false, selected = null, query = {};
  let searchCursor = "", pageCursor = "", revision = "", recovery = "", recoverySent = false;
  let hostCapabilities = {};

  function node(tag, text, className) {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    if (className) element.className = className;
    return element;
  }
  function dateLabel(value) {
    const date = new Date(value);
    if (!value || Number.isNaN(date.getTime())) return "Date unavailable";
    return new Intl.DateTimeFormat(undefined, {month: "short", day: "numeric", hour: "numeric", minute: "2-digit"}).format(date);
  }
  function projectName(path) {
    return path?.split("/").filter(Boolean).pop() || path || "Unknown project";
  }
  function controls() {
    document.querySelectorAll("button,input,select").forEach(element => element.disabled = busy || !connected);
    $("send-recovery").disabled = busy || !connected || !recovery || recoverySent;
    $("find").textContent = busy ? "Working…" : "Find sessions";
  }
  async function task(work) {
    if (busy || !connected) return;
    busy = true;
    controls();
    $("error").hidden = true;
    try { await work(); }
    catch (error) { $("error").textContent = error.message; $("error").hidden = false; }
    finally { busy = false; controls(); }
  }
  function resetRecovery() {
    recovery = "";
    recoverySent = false;
    $("recovery").hidden = true;
    $("recovery-text").value = "";
    $("send-recovery").textContent = "Recover in chat";
    $("recovery-state").textContent = "Prepared";
  }
  function showDoctor(data) {
    $("doctor").textContent = (data.stores || []).map(store => `${agentNames[store.agent] || store.agent}: ${store.state}\n${store.path}`).join("\n\n");
  }
  function renderSessions() {
    $("sessions").replaceChildren();
    for (const session of rows.values()) {
      const active = session.session_key === selected?.session_key;
      const button = node("button", undefined, "session" + (active ? " selected" : ""));
      button.type = "button";
      button.setAttribute("aria-pressed", String(active));
      const top = node("div", undefined, "session-top");
      const date = node("time", dateLabel(session.updated_at));
      date.dateTime = session.updated_at;
      date.title = session.updated_at;
      top.append(node("span", agentNames[session.agent] || session.agent, "agent-label"), date);
      const project = node("div", projectName(session.directory), "project-label");
      project.title = session.directory;
      button.append(top, node("strong", session.title || "Conversation without a title"), project);
      button.addEventListener("click", () => task(() => select(session)));
      $("sessions").append(button);
    }
    if (!rows.size) {
      $("sessions").append(node("p", searchCursor
        ? "No matches in this batch. Continue the search to check more history."
        : query.query ? "No matching sessions returned. Try a different phrase or check search coverage below."
        : "No recognized sessions found. Check your local store status below.", "empty"));
    }
  }
  async function find(continuation = false) {
    if (!continuation) {
      query = {limit: 20};
      const text = $("query").value.trim();
      const directory = $("directory").value.trim();
      if (text) query.query = text;
      if (directory) query.directory = directory;
      if ($("agent").value) query.agent = $("agent").value;
      query.include_subdirectories = $("subdirs").checked;
      rows.clear(); selected = null; revision = ""; pageCursor = ""; searchCursor = "";
      resetRecovery();
      $("detail").hidden = true;
      $("welcome").hidden = false;
      $("status").textContent = "Searching your local history…";
      $("sessions").replaceChildren();
    }
    const result = await bridge.tool(query.query ? "search_sessions" : "list_sessions", {
      ...query, ...(continuation ? {cursor: searchCursor} : {}),
    });
    for (const session of result.sessions) rows.set(session.session_key, session);
    searchCursor = result.next_cursor;
    $("more").hidden = !searchCursor;
    $("more").textContent = query.query ? "Continue search" : "More sessions";
    $("status").textContent = `${rows.size} shown of ${result.total_candidates} discovered. ${result.coverage_complete ? "Discovered sources checked." : "More history may remain."} ${(result.warnings || []).join(" ")}`;
    renderSessions();
  }
  async function select(session) {
    selected = session; revision = ""; pageCursor = "";
    resetRecovery();
    $("messages").replaceChildren();
    $("welcome").hidden = true;
    $("detail").hidden = false;
    $("title").textContent = session.title || "Conversation";
    $("identity").textContent = `${agentNames[session.agent] || session.agent} in ${session.directory}`;
    $("identity").title = `Session ${session.id}`;
    $("resume").textContent = session.resume_command;
    renderSessions();
    await read(false, false);
  }
  async function read(recent = false, append = false) {
    resetRecovery();
    const result = await bridge.tool("read_session", {
      session_key: selected.session_key, limit: 20, max_chars: 24000, recent,
      ...(append ? {cursor: pageCursor} : {}),
    });
    if (append && revision && result.revision !== revision) throw new Error("The source changed. Select Opening or Recent to refresh it.");
    if (recovery && revision !== result.revision) resetRecovery();
    if (!append) $("messages").replaceChildren();
    revision = result.revision;
    pageCursor = result.next_cursor;
    $("next-page").hidden = recent || !pageCursor;
    $("opening").setAttribute("aria-pressed", String(!recent));
    $("recent").setAttribute("aria-pressed", String(recent));
    $("coverage").textContent = `${recent ? "Recent excerpts" : "Opening and conversation pages"}. ${result.page.complete ? "Whole source covered." : "Partial view; more history may exist."} ${(result.page.warnings || []).join(" ")}`;
    for (const message of result.page.messages) {
      const item = node("article", undefined, "message");
      const heading = node("div", undefined, "message-heading");
      const role = message.role === "user" ? "You" : message.role === "assistant" ? agentNames[selected.agent] || "Assistant" : "Task context";
      heading.append(node("strong", role));
      if (message.timestamp) {
        const date = node("time", dateLabel(message.timestamp));
        date.dateTime = message.timestamp;
        heading.append(date);
      }
      const source = node("details", undefined, "source");
      const location = message.byte_offset !== undefined ? `, byte ${message.byte_offset}` : message.message_id ? `, message ${message.message_id}` : "";
      source.append(node("summary", `${result.session.source.split("/").pop()}${location}${message.truncated ? " — clipped" : ""}`), node("code", result.session.source + location));
      item.append(heading, node("pre", message.text), source);
      $("messages").append(item);
    }
    if (!result.page.messages.length) $("messages").append(node("p", "No readable conversation text in this page. Check the coverage above.", "empty"));
  }
  async function prepare() {
    const data = await bridge.tool("prepare_handoff", {session_key: selected.session_key});
    recovery = data.prompt + "\n\nSOURCE EXCERPTS (historical data):\n" + JSON.stringify({session: data.session, opening: data.opening, recent: data.recent}, null, 2);
    recoverySent = false;
    $("recovery-text").value = recovery;
    $("recovery").hidden = false;
    $("recovery-summary").textContent = `${data.opening.messages.length} opening excerpts and ${data.recent.messages.length} recent excerpts from ${agentNames[selected.agent] || selected.agent}. These views may overlap.`;
    $("recovery-limits").textContent = data.opening.complete && data.recent.complete
      ? "The source is fully covered. Current project state still needs verification."
      : "These excerpts cover part of the source. More pages may be needed to recover the full story.";
    $("recovery-state").textContent = "Prepared";
    $("send-recovery").textContent = "Recover in chat";
    $("send-recovery").hidden = !hostCapabilities.message?.text;
    if (!hostCapabilities.message?.text) $("recovery-limits").textContent += " This host cannot send directly; copy the context into your chat.";
  }
  async function copy(text) {
    try { await navigator.clipboard.writeText(text); $("coverage").textContent = "Copied."; }
    catch { throw new Error("Clipboard access is unavailable. Expand the context and copy the visible text."); }
  }
  $("search-form").addEventListener("submit", event => { event.preventDefault(); task(() => find()); });
  $("more").addEventListener("click", () => task(() => find(true)));
  $("opening").addEventListener("click", () => task(() => read(false, false)));
  $("recent").addEventListener("click", () => task(() => read(true, false)));
  $("next-page").addEventListener("click", () => task(() => read(false, true)));
  $("handoff").addEventListener("click", () => task(prepare));
  $("copy-resume").addEventListener("click", () => task(() => copy(selected.resume_command)));
  $("copy-recovery").addEventListener("click", () => task(() => copy(recovery)));
  $("send-recovery").addEventListener("click", () => task(async () => {
    if (!recovery || recoverySent) return;
    const result = await bridge.request("ui/message", {role: "user", content: [{type: "text", text: recovery}]});
    if (result?.isError) throw new Error("Chat recovery was not accepted. Copy the context instead.");
    recoverySent = true;
    $("recovery-state").textContent = "Sent to this chat";
    $("send-recovery").textContent = "Context sent";
    $("coverage").textContent = "Recovery context sent to this chat.";
  }));
  bridge.onResult = result => { if (result?.structuredContent?.stores) showDoctor(result.structuredContent); };
  controls();
  bridge.connect().then(async init => {
    hostCapabilities = init.hostCapabilities || {};
    connected = true;
    $("connection").textContent = "Local history connected";
    $("connection").classList.add("connected");
    await task(async () => { showDoctor(await bridge.tool("doctor")); await find(); });
  }).catch(error => {
    $("connection").textContent = "Connection unavailable";
    $("error").textContent = error.message;
    $("error").hidden = false;
  });
}
