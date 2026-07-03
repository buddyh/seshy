// The single stats pass: session sources in -> one canonical stats object out.
// Every card, theme, and cut renders from this object (or its stats.json
// serialization). Nothing downstream recounts anything.
import { discover } from './discover.js';
import { parseSession } from './parse.js';
import { newAccumulator, feedSession, finalize, sessionActiveMs } from './stats.js';

// Run tasks with a bounded concurrency pool.
async function pool(items, limit, worker, onTick) {
  let i = 0;
  let done = 0;
  const runners = new Array(Math.min(limit, items.length)).fill(0).map(async () => {
    while (i < items.length) {
      const idx = i++;
      await worker(items[idx], idx);
      done++;
      if (onTick && done % 25 === 0) onTick(done, items.length);
    }
  });
  await Promise.all(runners);
}

// Attribute every event to the model that answered its turn (a human prompt
// inherits the model of the assistant reply that follows it), then keep only
// events matching the filter. Sessions can mix models via /model switches.
function filterToModel(events, modelRe) {
  let carry = '';
  for (let i = events.length - 1; i >= 0; i--) {
    const ev = events[i];
    if (ev.role === 'assistant' && ev.model) carry = ev.model;
    ev._model = ev.model || carry;
  }
  let fwd = '';
  for (const ev of events) {
    if (ev.role === 'assistant' && ev.model) fwd = ev.model;
    else if (!ev._model) ev._model = fwd;
  }
  return events.filter((e) => modelRe.test(e._model || ''));
}

// collect(opts) -> canonical report object.
// opts: { agent, since (ms|0), until (ms|0), model, window, concurrency, onProgress }
export async function collect({ agent = 'all', since = 0, until = 0, model = '', window = 'all', concurrency = 16, onProgress } = {}) {
  let files = discover(agent);
  if (since) files = files.filter((f) => (f.sessionMtime || f.mtime) >= since);
  // Deterministic processing order regardless of filesystem enumeration:
  // sort by path, then per-session id for sqlite-backed sources.
  files.sort((a, b) => (a.path === b.path ? String(a.sessionId || '') < String(b.sessionId || '') ? -1 : 1 : a.path < b.path ? -1 : 1));
  const modelRe = model ? new RegExp(model.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'i') : null;

  const acc = newAccumulator();
  const harness = {};
  const bump = (a) =>
    (harness[a] ||= { sessions: 0, interactive: 0, headless: 0, prompts: 0, activeMs: 0, firstTs: 0, lastTs: 0 });

  // Parsing runs in parallel; feeding is serialized per-session by the pool
  // worker, and the accumulator is order-insensitive for every stat except
  // longestSession's cwd tiebreak — which the sorted file order makes stable.
  const fed = [];
  await pool(
    files,
    concurrency,
    async (file, idx) => {
      let events = [];
      const info = await parseSession(file, (ev) => events.push(ev));
      for (const ev of events) {
        if (!ev.cwd && info.cwd) ev.cwd = info.cwd;
        ev.agent = file.agent;
      }
      if (until) events = events.filter((e) => !e.ts || e.ts <= until);
      if (since) events = events.filter((e) => !e.ts || e.ts >= since);
      if (modelRe) {
        events = filterToModel(events, modelRe);
        if (!events.length) return; // session never touched this model
      }
      fed[idx] = { file, events, info };
    },
    onProgress,
  );

  // Feed strictly in sorted-file order so any order-sensitive derivation
  // (longest-session tiebreaks) is identical run to run.
  for (const item of fed) {
    if (!item) continue;
    const { file, events, info } = item;
    acc.skippedLines += info.skipped || 0;
    const h = bump(file.agent);
    h.sessions++;
    if (info.headless) h.headless++;
    else h.interactive++;
    const stamps = events.map((e) => e.ts).filter(Boolean);
    if (stamps.length) {
      const first = Math.min(...stamps);
      const last = Math.max(...stamps);
      h.firstTs = h.firstTs ? Math.min(h.firstTs, first) : first;
      h.lastTs = Math.max(h.lastTs, last);
    }
    // Only interactive sessions feed the content stats — the card is about
    // conversations you actually had, kept fair across every harness.
    if (!info.headless && events.length) {
      h.prompts += feedSession(acc, events);
      h.activeMs += sessionActiveMs(events);
    }
  }

  const agents = agent === 'all' ? [...new Set(files.map((f) => f.agent))].sort() : [agent];
  return finalize(acc, { agent, agents, filesScanned: files.length, model, window, harness });
}

// Canonical serialization: recursively sorted keys, LF newline, no wall-clock
// fields — the same data always produces byte-identical stats.json.
export function stableStringify(value) {
  return JSON.stringify(sortKeys(value), null, 2) + '\n';
}

function sortKeys(v) {
  if (Array.isArray(v)) return v.map(sortKeys);
  if (v && typeof v === 'object') {
    const out = {};
    for (const k of Object.keys(v).sort()) out[k] = sortKeys(v[k]);
    return out;
  }
  return v;
}
