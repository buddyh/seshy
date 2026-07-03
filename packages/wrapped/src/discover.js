// Discover local AI-coding session sources across agents.
// Shares seshy's on-disk layout knowledge. Each entry is one logical session:
//   { agent, path, mtime, kind: 'jsonl' | 'json-array' | 'sqlite', sessionId? }
import os from 'node:os';
import path from 'node:path';
import { readdirSync, statSync, existsSync } from 'node:fs';

const HOME = os.homedir();

// Recursively collect files matching a predicate, swallowing unreadable dirs.
function walk(root, match, out = []) {
  let entries;
  try {
    entries = readdirSync(root, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const e of entries) {
    const p = path.join(root, e.name);
    if (e.isDirectory()) walk(p, match, out);
    else if (match(e.name, p)) out.push(p);
  }
  return out;
}

const file = (agent, kind) => (p, extra = {}) => {
  try {
    return { agent, kind, path: p, mtime: statSync(p).mtimeMs, ...extra };
  } catch {
    return null;
  }
};

const SOURCES = {
  claude: (home) => {
    const mk = file('claude', 'jsonl');
    return walk(path.join(home, '.claude', 'projects'), (n) => n.endsWith('.jsonl') && !n.startsWith('agent-')).map((p) => mk(p));
  },
  codex: (home) => {
    const mk = file('codex', 'jsonl');
    return walk(path.join(home, '.codex', 'sessions'), (n) => n.startsWith('rollout-') && n.endsWith('.jsonl')).map((p) => mk(p));
  },
  // Gemini CLI keeps one logs.json per project hash: a JSON array of typed
  // user messages. Assistant text is not recorded there, so gemini feeds the
  // human-side stats only.
  gemini: (home) => {
    const mk = file('gemini', 'json-array');
    return walk(path.join(home, '.gemini', 'tmp'), (n) => n === 'logs.json').map((p) => mk(p));
  },
  // OpenCode keeps everything in one SQLite database. Each session row is one
  // logical "session file"; parse() reads its messages + text parts.
  opencode: (home) => {
    const db = path.join(home, '.local', 'share', 'opencode', 'opencode.db');
    if (!existsSync(db)) return [];
    let rows;
    try {
      rows = opencodeSessions(db);
    } catch {
      return []; // node:sqlite unavailable or db unreadable — degrade quietly
    }
    const mk = file('opencode', 'sqlite');
    return rows.map((r) => mk(db, { sessionId: r.id, sessionMtime: r.time_updated || 0 }));
  },
  pi: (home) => {
    const mk = file('pi', 'jsonl');
    return walk(path.join(home, '.pi', 'agent', 'sessions'), (n) => n.endsWith('.jsonl')).map((p) => mk(p));
  },
  droid: (home) => {
    const mk = file('droid', 'jsonl');
    return walk(path.join(home, '.factory', 'sessions'), (n) => n.endsWith('.jsonl')).map((p) => mk(p));
  },
};

export function opencodeSessions(dbPath) {
  // Lazy so machines without node:sqlite (or without OpenCode) never pay for it.
  const { DatabaseSync } = requireSqlite();
  const db = new DatabaseSync(dbPath, { readOnly: true });
  try {
    return db.prepare('SELECT id, time_updated FROM session ORDER BY id').all();
  } finally {
    db.close();
  }
}

let sqliteMod = null;
export function requireSqlite() {
  if (!sqliteMod) {
    // node:sqlite ships with Node >= 22.5; throws on older runtimes.
    sqliteMod = process.getBuiltinModule ? process.getBuiltinModule('node:sqlite') : null;
    if (!sqliteMod) throw new Error('node:sqlite unavailable');
  }
  return sqliteMod;
}

export const KNOWN_AGENTS = Object.keys(SOURCES);

// discover(agent, home) -> session entries. agent 'all' unions every source.
// home is injectable so tests can point at synthetic fixture trees.
export function discover(agent = 'all', home = HOME) {
  const wanted = agent === 'all' ? KNOWN_AGENTS : [agent];
  const out = [];
  for (const a of wanted) {
    const src = SOURCES[a];
    if (!src) continue;
    for (const f of src(home)) if (f) out.push(f);
  }
  return out;
}
