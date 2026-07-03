// Read one session source and emit normalized events. Deterministic: pure
// text extraction, no model calls, corrupt lines skipped and counted.
//
// Event: { role: 'human'|'assistant'|'system', text, ts (ms|0), cwd, kind,
//          usage?: { in, out, cacheRead },              // API tokens
//          tools?: { calls, bash, lines, files[], names[], skills[], scary, commits },
//          results?: { total, errors },                 // tool results (system)
//          command?: '/name',                           // slash command (system)
//          model?: '' }
// kind: 'prompt' | 'reply' | 'interrupt' | 'command' | 'tool_result'.
import { createReadStream, readFileSync } from 'node:fs';
import { createInterface } from 'node:readline';
import { requireSqlite } from './discover.js';

// Text that is machine-injected into a "user" turn, not something the human
// typed: environment blocks, command wrappers, tool results, system reminders.
const NOISE_PREFIX = /^\s*(<environment_context|<user_instructions|<system-reminder|<command-name|<command-message|<command-args|<local-command|<bash-|Caveat:|<user-prompt-submit|<post-tool)/i;

function isInterrupt(t) {
  return /\[Request interrupted by user/i.test(t) || /\[Request cancelled/i.test(t);
}

// Flatten a Claude-style content value (string | array of blocks).
function claudeText(content) {
  if (typeof content === 'string') return { text: content, thinking: '' };
  if (!Array.isArray(content)) return { text: '', thinking: '' };
  let text = '';
  let thinking = '';
  for (const b of content) {
    if (!b || typeof b !== 'object') continue;
    if (b.type === 'text' && typeof b.text === 'string') text += b.text + '\n';
    else if (b.type === 'thinking' && typeof b.thinking === 'string') thinking += b.thinking + '\n';
  }
  return { text: text.trim(), thinking: thinking.trim() };
}

const RE_SCARY = /\brm\s+-[a-z]*rf?\b|\bsudo\b|\bgit\s+push\s+(?:-f\b|--force)|\bdrop\s+table\b/i;
const RE_GIT_COMMIT = /\bgit\s+commit\b/;

// Mine tool_use blocks for machine metrics.
function toolMetrics(content) {
  if (!Array.isArray(content)) return null;
  let calls = 0;
  let bash = 0;
  let lines = 0;
  let scary = 0;
  let commits = 0;
  const files = [];
  const names = [];
  const skills = [];
  for (const b of content) {
    if (!b || b.type !== 'tool_use') continue;
    calls++;
    const name = b.name || '';
    const inp = b.input && typeof b.input === 'object' ? b.input : {};
    names.push(name);
    if (name === 'Skill' && typeof inp.skill === 'string') skills.push(inp.skill);
    if (name === 'Bash') {
      bash++;
      if (typeof inp.command === 'string') {
        if (RE_SCARY.test(inp.command)) scary++;
        if (RE_GIT_COMMIT.test(inp.command)) commits++;
      }
    }
    let added = '';
    if (name === 'Write' && typeof inp.content === 'string') added = inp.content;
    else if (name === 'Edit' && typeof inp.new_string === 'string') added = inp.new_string;
    else if (name === 'MultiEdit' && Array.isArray(inp.edits)) added = inp.edits.map((e) => (e && typeof e.new_string === 'string' ? e.new_string : '')).join('\n');
    else if (name === 'NotebookEdit' && typeof inp.new_source === 'string') added = inp.new_source;
    if (added) {
      lines += added.split('\n').length;
      const f = inp.file_path || inp.notebook_path;
      if (f) files.push(f);
    }
  }
  return calls ? { calls, bash, lines, files, names, skills, scary, commits } : null;
}

// Count tool_result blocks in a user-turn content array (and how many errored).
function toolResults(content) {
  if (!Array.isArray(content)) return null;
  let total = 0;
  let errors = 0;
  for (const b of content) {
    if (!b || b.type !== 'tool_result') continue;
    total++;
    if (b.is_error) errors++;
  }
  return total ? { total, errors } : null;
}

function codexText(content) {
  if (typeof content === 'string') return content;
  if (!Array.isArray(content)) return '';
  let out = '';
  for (const b of content) {
    if (!b || typeof b !== 'object') continue;
    if ((b.type === 'input_text' || b.type === 'output_text' || b.type === 'text') && typeof b.text === 'string') {
      out += b.text + '\n';
    }
  }
  return out.trim();
}

function ts(v) {
  if (!v) return 0;
  if (typeof v === 'number') return v;
  const n = Date.parse(v);
  return Number.isNaN(n) ? 0 : n;
}

// Per-agent line handlers for JSONL sources. Each returns an event or null.
const HANDLERS = {
  claude(rec, ctx) {
    const t = rec.type;
    // How the session was launched: 'cli'/'claude-desktop' = interactive,
    // 'sdk-cli' = headless (claude -p / Agent SDK / dispatched automation).
    if (rec.entrypoint) ctx.entrypoint = rec.entrypoint;
    if (t === 'user') {
      const msg = rec.message;
      if (!msg) return null;
      const { text } = claudeText(msg.content);
      if (text && isInterrupt(text)) return { role: 'human', text, ts: ts(rec.timestamp), cwd: rec.cwd || ctx.cwd, kind: 'interrupt' };
      // Tool results ride in as user turns with array content.
      if (typeof msg.content !== 'string') {
        const res = toolResults(msg.content);
        return res ? { role: 'system', kind: 'tool_result', results: res, ts: ts(rec.timestamp), cwd: rec.cwd || ctx.cwd, text: '' } : null;
      }
      if (!text) return null;
      // Slash commands arrive as machine-wrapped user turns; mine the name.
      const cmd = /<command-name>\s*(\/[\w:-]+)/.exec(text);
      if (cmd) return { role: 'system', kind: 'command', command: cmd[1], ts: ts(rec.timestamp), cwd: rec.cwd || ctx.cwd, text: '' };
      if (NOISE_PREFIX.test(text)) return null;
      const typed = !rec.promptSource || rec.promptSource === 'typed';
      const human = !rec.origin || rec.origin.kind === 'human';
      if (!typed || !human) return null;
      return { role: 'human', text, ts: ts(rec.timestamp), cwd: rec.cwd || ctx.cwd, kind: 'prompt' };
    }
    if (t === 'assistant') {
      const msg = rec.message;
      if (!msg) return null;
      const { text, thinking } = claudeText(msg.content);
      const tools = toolMetrics(msg.content);
      // One API message streams as several JSONL lines that repeat the same
      // usage object — dedupe on message id so tokens count once.
      let usage = null;
      const u = msg.usage;
      if (u && msg.id && msg.id !== ctx.lastUsageId) {
        ctx.lastUsageId = msg.id;
        usage = { in: u.input_tokens || 0, out: u.output_tokens || 0, cacheRead: u.cache_read_input_tokens || 0 };
      }
      if (!text && !thinking && !tools && !usage) return null;
      return { role: 'assistant', text, thinking, ts: ts(rec.timestamp), cwd: rec.cwd || ctx.cwd, kind: 'reply', tools, usage, model: msg.model || '' };
    }
    return null;
  },

  codex(rec, ctx) {
    if (rec.type === 'session_meta') {
      const p = rec.payload || {};
      if (p.cwd) ctx.cwd = p.cwd;
      if (p.timestamp && !ctx.firstTs) ctx.firstTs = ts(p.timestamp);
      // source: 'cli'/'vscode' = interactive, 'exec' = headless (codex exec),
      // an object = a spawned subagent (also headless).
      if (p.source !== undefined) ctx.source = p.source;
      return null;
    }
    if (rec.type === 'turn_context' && rec.payload) {
      if (rec.payload.cwd) ctx.cwd = rec.payload.cwd;
      if (rec.payload.model) ctx.model = rec.payload.model;
      return null;
    }
    if (rec.type === 'event_msg' && rec.payload?.type === 'token_count') {
      const u = rec.payload.info?.last_token_usage;
      if (!u) return null;
      return { role: 'assistant', text: '', thinking: '', ts: ts(rec.timestamp), cwd: ctx.cwd, kind: 'reply', usage: { in: u.input_tokens || 0, out: u.output_tokens || 0, cacheRead: u.cached_input_tokens || 0 } };
    }
    if (rec.type !== 'response_item') return null;
    const p = rec.payload || {};
    if (p.type === 'function_call') {
      const bash = /shell|exec|bash/i.test(p.name || '') ? 1 : 0;
      let scary = 0;
      let commits = 0;
      if (bash && typeof p.arguments === 'string') {
        if (RE_SCARY.test(p.arguments)) scary = 1;
        if (RE_GIT_COMMIT.test(p.arguments)) commits = 1;
      }
      return { role: 'assistant', text: '', thinking: '', ts: ts(rec.timestamp), cwd: ctx.cwd, kind: 'reply', model: ctx.model || '', tools: { calls: 1, bash, lines: 0, files: [], names: [p.name || 'tool'], skills: [], scary, commits } };
    }
    if (p.type !== 'message') return null;
    const text = codexText(p.content);
    if (!text) return null;
    const at = ts(rec.timestamp);
    if (p.role === 'user') {
      if (isInterrupt(text)) return { role: 'human', text, ts: at, cwd: ctx.cwd, kind: 'interrupt' };
      if (NOISE_PREFIX.test(text)) return null;
      return { role: 'human', text, ts: at, cwd: ctx.cwd, kind: 'prompt' };
    }
    if (p.role === 'assistant') {
      return { role: 'assistant', text, thinking: '', ts: at, cwd: ctx.cwd, kind: 'reply', model: ctx.model || '' };
    }
    return null;
  },

  // pi / droid share a loose { message: { role, content } } or { role, content } shape.
  generic(rec, ctx) {
    const msg = rec.message && typeof rec.message === 'object' ? rec.message : rec;
    const role = msg.role;
    if (rec.cwd && !ctx.cwd) ctx.cwd = rec.cwd;
    if (role !== 'user' && role !== 'assistant') return null;
    const { text } = claudeText(msg.content);
    if (!text) return null;
    if (role === 'user') {
      if (isInterrupt(text)) return { role: 'human', text, ts: ts(rec.timestamp), cwd: ctx.cwd, kind: 'interrupt' };
      if (typeof msg.content !== 'string' || NOISE_PREFIX.test(text)) return null;
      return { role: 'human', text, ts: ts(rec.timestamp), cwd: ctx.cwd, kind: 'prompt' };
    }
    return { role: 'assistant', text, thinking: '', ts: ts(rec.timestamp), cwd: ctx.cwd, kind: 'reply' };
  },
};

// Fast pre-filter so we skip JSON.parse on lines with no message payload.
const RELEVANT = {
  claude: (l) => l.includes('"type":"user"') || l.includes('"type":"assistant"'),
  codex: (l) => l.includes('"message"') || l.includes('session_meta') || l.includes('turn_context') || l.includes('function_call') || l.includes('token_count'),
};

async function parseJsonl(file, onEvent) {
  const handler = HANDLERS[file.agent] || HANDLERS.generic;
  const relevant = RELEVANT[file.agent] || (() => true);
  const ctx = { cwd: '', firstTs: 0 };
  const rl = createInterface({ input: createReadStream(file.path, { encoding: 'utf8' }), crlfDelay: Infinity });
  let events = 0;
  let skipped = 0;
  try {
    for await (const line of rl) {
      if (!line || !relevant(line)) continue;
      let rec;
      try {
        rec = JSON.parse(line);
      } catch {
        skipped++; // truncated / corrupt line — count it, keep going
        continue;
      }
      const ev = handler(rec, ctx);
      if (ev) {
        onEvent(ev, ctx);
        events++;
      }
    }
  } catch {
    skipped++; // unreadable/partial file — take what we got
  }
  return { events, skipped, cwd: ctx.cwd, headless: isHeadless(ctx), startTs: ctx.firstTs, endTs: 0 };
}

// Gemini CLI: logs.json is one JSON array of typed user prompts. Assistant
// text is not recorded, so gemini contributes human-side stats only.
function parseGemini(file, onEvent) {
  let arr;
  try {
    arr = JSON.parse(readFileSync(file.path, 'utf8'));
  } catch {
    return { events: 0, skipped: 1, cwd: '', headless: false, startTs: 0, endTs: 0 };
  }
  if (!Array.isArray(arr)) return { events: 0, skipped: 1, cwd: '', headless: false, startTs: 0, endTs: 0 };
  let events = 0;
  for (const rec of arr) {
    if (!rec || rec.type !== 'user' || typeof rec.message !== 'string' || !rec.message) continue;
    const text = rec.message;
    if (text.startsWith('/')) {
      onEvent({ role: 'system', kind: 'command', command: text.split(/\s/)[0], ts: ts(rec.timestamp), cwd: '', text: '' });
    } else {
      onEvent({ role: 'human', text, ts: ts(rec.timestamp), cwd: '', kind: 'prompt' });
    }
    events++;
  }
  return { events, skipped: 0, cwd: '', headless: false, startTs: 0, endTs: 0 };
}

// OpenCode: one SQLite db, one logical session per discover() entry. Message
// role/model/tokens live in message.data JSON; prose lives in part rows.
function parseOpencode(file, onEvent) {
  let events = 0;
  let skipped = 0;
  let cwd = '';
  try {
    const { DatabaseSync } = requireSqlite();
    const db = new DatabaseSync(file.path, { readOnly: true });
    try {
      const msgs = db
        .prepare('SELECT id, time_created, data FROM message WHERE session_id = ? ORDER BY time_created, id')
        .all(file.sessionId);
      const partStmt = db.prepare("SELECT data FROM part WHERE message_id = ? ORDER BY id");
      for (const m of msgs) {
        let d;
        try {
          d = JSON.parse(m.data);
        } catch {
          skipped++;
          continue;
        }
        if (d.path?.cwd) cwd = d.path.cwd;
        let text = '';
        for (const p of partStmt.all(m.id)) {
          try {
            const pd = JSON.parse(p.data);
            if (pd.type === 'text' && typeof pd.text === 'string') text += pd.text + '\n';
          } catch {
            skipped++;
          }
        }
        text = text.trim();
        const at = d.time?.created || m.time_created || 0;
        const model = d.modelID || d.model?.modelID || '';
        if (d.role === 'user' && text) {
          onEvent({ role: 'human', text, ts: at, cwd, kind: 'prompt' });
          events++;
        } else if (d.role === 'assistant') {
          const tok = d.tokens || {};
          const usage = tok.input || tok.output ? { in: tok.input || 0, out: tok.output || 0, cacheRead: tok.cache?.read || 0 } : null;
          if (text || usage) {
            onEvent({ role: 'assistant', text, thinking: '', ts: at, cwd, kind: 'reply', model, usage });
            events++;
          }
        }
      }
    } finally {
      db.close();
    }
  } catch {
    skipped++;
  }
  return { events, skipped, cwd, headless: false, startTs: 0, endTs: 0 };
}

function isHeadless(ctx) {
  if (ctx.entrypoint) return ctx.entrypoint !== 'cli' && ctx.entrypoint !== 'claude-desktop';
  if (ctx.source !== undefined) return typeof ctx.source === 'object' || ctx.source === 'exec';
  return false;
}

// parseSession(file, onEvent) -> { events, skipped, cwd, headless, startTs }
export async function parseSession(file, onEvent) {
  if (file.kind === 'json-array') return parseGemini(file, onEvent);
  if (file.kind === 'sqlite') return parseOpencode(file, onEvent);
  return parseJsonl(file, onEvent);
}
