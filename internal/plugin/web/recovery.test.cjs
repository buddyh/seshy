const {test} = require('node:test');
const assert = require('node:assert/strict');
const {readFileSync} = require('node:fs');
const {join} = require('node:path');
const {runInNewContext} = require('node:vm');

function panel() {
  const elements = new Map();
  function element() {
    return {
      hidden: false, value: '', children: [], handlers: {}, classList: {add() {}},
      append(...children) { this.children.push(...children); },
      replaceChildren(...children) { this.children = children; },
      setAttribute() {},
      addEventListener(name, handler) { this.handlers[name] = handler; },
    };
  }
  const html = readFileSync(join(__dirname, 'index.html'), 'utf8');
  for (const match of html.matchAll(/id="([^"]+)"/g)) elements.set(match[1], element());
  const selected = {session_key: 'codex:fixture', agent: 'codex', id: 'fixture',
    directory: '/fixture/project', source: '/fixture/history.jsonl', resume_command: "'codex' 'resume' 'fixture'"};
  const page = {messages: [{role: 'user', text: 'Synthetic goal', byte_offset: 0}], complete: true};
  const sent = [];
  let listener, readError = false;
  const parent = {
    postMessage(request) {
      sent.push(request);
      if (request.id === undefined) return;
      let result = {};
      if (request.method === 'ui/initialize') result = {hostCapabilities: {serverTools: {}, message: {text: {}}}};
      if (request.method === 'tools/call') {
        const name = request.params.name;
        const data = name === 'doctor' ? {stores: []}
          : name === 'list_sessions' ? {sessions: [selected], next_cursor: '', total_candidates: 1, coverage_complete: true}
          : name === 'prepare_handoff' ? {session: selected, opening: page, recent: page, prompt: 'Recover synthetic goal'}
          : {session: selected, page, revision: 'fixture-revision', next_cursor: ''};
        result = name === 'read_session' && readError
          ? {isError: true, content: [{type: 'text', text: 'Source changed; refresh this session'}]}
          : {structuredContent: data};
      }
      queueMicrotask(() => listener({source: parent, data: {jsonrpc: '2.0', id: request.id, result}}));
    },
  };
  runInNewContext(readFileSync(join(__dirname, 'app.js'), 'utf8'), {
    window: {parent, addEventListener(name, handler) { listener = handler; }},
    document: {getElementById: id => elements.get(id), createElement: element,
      querySelectorAll: () => [...elements.values()]}, setTimeout, clearTimeout,
  });
  return {elements, sent, failReads() { readError = true; }};
}

async function settle() {
  await new Promise(resolve => setImmediate(resolve));
}

test('preparation stays local and explicit recovery sends only once', async () => {
  const h = panel();
  await settle();
  h.elements.get('sessions').children[0].handlers.click();
  await settle();
  h.elements.get('handoff').handlers.click();
  await settle();
  assert.equal(h.elements.get('recovery').hidden, false);
  assert.equal(h.sent.filter(request => request.method === 'ui/message').length, 0);
  h.elements.get('send-recovery').handlers.click();
  await settle();
  h.elements.get('send-recovery').handlers.click();
  await settle();
  assert.equal(h.sent.filter(request => request.method === 'ui/message').length, 1);
  assert.equal(h.elements.get('send-recovery').disabled, true);
});

test('a failed source refresh invalidates previously prepared recovery', async () => {
  const h = panel();
  await settle();
  h.elements.get('sessions').children[0].handlers.click();
  await settle();
  h.elements.get('handoff').handlers.click();
  await settle();
  h.failReads();
  h.elements.get('opening').handlers.click();
  await settle();
  assert.match(h.elements.get('error').textContent, /Source changed/);
  assert.equal(h.elements.get('recovery').hidden, true);
  assert.equal(h.elements.get('recovery-text').value, '');
  assert.equal(h.elements.get('send-recovery').disabled, true);
  h.elements.get('send-recovery').handlers.click();
  await settle();
  assert.equal(h.sent.filter(request => request.method === 'ui/message').length, 0);
});
