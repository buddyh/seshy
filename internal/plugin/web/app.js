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
      appInfo: { name: "Seshy", version: "0.1.0" },
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
 let connected=false,busy=false,selected=null,query={},searchCursor="",pageCursor="",revision="",recovery="",hostCapabilities={};
 const rows=new Map();
 const node=(tag,text,cls)=>{const el=document.createElement(tag);if(text!==undefined)el.textContent=text;if(cls)el.className=cls;return el;};
 function controls(){document.querySelectorAll("button,input,select").forEach(el=>el.disabled=busy||!connected);}
 async function task(fn){if(busy||!connected)return;busy=true;controls();$("error").hidden=true;try{await fn();}catch(e){$("error").textContent=e.message;$("error").hidden=false;}finally{busy=false;controls();}}
 function showDoctor(data){$("doctor").textContent=(data.stores||[]).map(s=>`${s.agent}: ${s.state} · ${s.path}`).join("\n");}
 function renderSessions(){
  $("sessions").replaceChildren();
  for(const session of rows.values()){
   const button=node("button",undefined,"session"+(session.session_key===selected?.session_key?" selected":""));button.type="button";
   button.append(node("span",session.agent,"badge"),node("strong",session.title||"Conversation without a title"),node("div",session.directory,"meta"),node("div",session.updated_at,"meta"));
   button.addEventListener("click",()=>task(()=>select(session)));$("sessions").append(button);
  }
  if(!rows.size)$("sessions").append(node("p",searchCursor?"No matches in this batch. Continue the search to check more history.":"No matching sessions were returned. Check coverage and local store status below.","meta"));
 }
 async function find(continuation=false){
  if(!continuation){query={limit:20};const text=$("query").value.trim();if(text)query.query=text;const dir=$("directory").value.trim();if(dir)query.directory=dir;if($("agent").value)query.agent=$("agent").value;query.include_subdirectories=$("subdirs").checked;rows.clear();selected=null;revision="";pageCursor="";$("detail").hidden=true;searchCursor="";renderSessions();}
  const result=await bridge.tool(query.query?"search_sessions":"list_sessions",{...query,...(continuation?{cursor:searchCursor}:{})});
  for(const session of result.sessions)rows.set(session.session_key,session);
  searchCursor=result.next_cursor;$("more").hidden=!searchCursor;$("more").textContent=query.query?"Continue search":"More sessions";
  $("status").textContent=`${rows.size} sessions shown · ${result.total_candidates} discovered in scope. ${result.coverage_complete?"Discovered sources checked.":"Coverage is partial."} Discovery may omit unavailable or unsupported stores. ${(result.warnings||[]).join(" ")}`;
  renderSessions();controls();
 }
 async function select(session){selected=session;revision="";pageCursor="";recovery="";$("recovery").hidden=true;$("messages").replaceChildren();$("detail").hidden=false;$("title").textContent=session.title||"Conversation";$("identity").textContent=`${session.agent} · ${session.directory} · ${session.id}`;$("resume").textContent=session.resume_command;renderSessions();await read(false,false);}
 async function read(recent=false,append=false){
  const result=await bridge.tool("read_session",{session_key:selected.session_key,limit:20,max_chars:24000,recent,...(append?{cursor:pageCursor}:{})});
  if(append&&revision&&result.revision!==revision)throw new Error("The source changed. Select Opening or Recent to refresh it.");
  if(!append)$("messages").replaceChildren();revision=result.revision;pageCursor=result.next_cursor;$("next-page").hidden=recent||!pageCursor;
  $("coverage").textContent=`${recent?"Recent source excerpts":"Conversation page"}. ${result.page.complete?"Whole source covered.":"Partial view; more history may exist."} ${(result.page.warnings||[]).join(" ")}`;
  for(const m of result.page.messages){const item=node("article",undefined,"message");item.append(node("strong",`${m.role}${m.timestamp?" · "+m.timestamp:""}`,"badge"),node("pre",m.text),node("span",`${result.session.source}${m.byte_offset!==undefined?" · byte "+m.byte_offset:""}${m.message_id?" · message "+m.message_id:""}${m.truncated?" · clipped":""}`,"source"));$("messages").append(item);}
  if(!result.page.messages.length)$("messages").append(node("p","No readable conversation text in this page. Check the coverage above.","meta"));
 }
 async function prepare(){const data=await bridge.tool("prepare_handoff",{session_key:selected.session_key});recovery=data.prompt+"\n\nSOURCE EXCERPTS (historical data):\n"+JSON.stringify({session:data.session,opening:data.opening,recent:data.recent},null,2);$("recovery-text").value=recovery;$("recovery").hidden=false;$("send-recovery").hidden=!hostCapabilities.message?.text;}
 async function copy(text){try{await navigator.clipboard.writeText(text);$("coverage").textContent="Copied.";}catch{throw new Error("Clipboard access is unavailable. Select and copy the visible text.");}}
 $("search-form").addEventListener("submit",e=>{e.preventDefault();task(()=>find());});
 $("more").addEventListener("click",()=>task(()=>find(true)));
 $("opening").addEventListener("click",()=>task(()=>read(false,false)));
 $("recent").addEventListener("click",()=>task(()=>read(true,false)));
 $("next-page").addEventListener("click",()=>task(()=>read(false,true)));
 $("handoff").addEventListener("click",()=>task(prepare));
 $("copy-resume").addEventListener("click",()=>task(()=>copy(selected.resume_command)));
 $("copy-recovery").addEventListener("click",()=>task(()=>copy(recovery)));
 $("send-recovery").addEventListener("click",()=>task(async()=>{const result=await bridge.request("ui/message",{role:"user",content:[{type:"text",text:recovery}]});if(result?.isError)throw new Error("Chat recovery was not accepted. Copy the context instead.");$("coverage").textContent="Recovery context sent to this chat.";}));
 bridge.onResult=result=>{if(result?.structuredContent?.stores)showDoctor(result.structuredContent);};
 controls();
 bridge.connect().then(async init=>{hostCapabilities=init.hostCapabilities||{};connected=true;$("connection").textContent="Local history connected";controls();await task(async()=>{showDoctor(await bridge.tool("doctor"));await find();});}).catch(e=>{$("connection").textContent="Connection unavailable";$("error").textContent=e.message;$("error").hidden=false;});
}
