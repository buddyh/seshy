// Card appearances beyond the default sunset, all in the seshy synthwave
// family: deep-space purples, radial nebula glows, JetBrains Mono, rainbow
// wordmark gradient, terminal chrome, scanlines, vignette.
// Each theme: (report, opts) -> full SVG string, 1080x1350 logical (renders
// at 1600x2000). Pure template functions over the canonical report.
import { pickHeadline, pickTiles, signature, CUT_TAG, prettyModel } from './copy.js';

const C = {
  bg0: '#0d0221', bg1: '#14062b', bg2: '#1a0533', ink: '#F5EEFF', lav: '#B8A6D9',
  fog: '#6C5C8A', dim: '#8E7DB0', pink: '#FF6AC1', magenta: '#F92AAD',
  purple: '#B026FF', cyan: '#2DE2E6', aqua: '#05D9E8', yellow: '#FFD319',
  orange: '#FF8E42', surface: '#241B2F', edge: '#3A1C63',
};

export const W = 1080;
export const H = 1350;
const PAD = 72;

const MONO = 'JetBrains Mono, Menlo, monospace';
const DISPLAY = 'Futura, Avenir Next Condensed, Impact, sans-serif';
const LABEL = 'Avenir Next Condensed, Futura, Helvetica Neue, sans-serif';

export const esc = (s) => String(s).replace(/[<>&'"]/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', "'": '&apos;', '"': '&quot;' }[c]));

// fitMono shrinks a mono line's font-size so it fits maxPx (0.62em/glyph),
// capped at max — long punchlines scale down instead of colliding or clipping.
export const fitMono = (text, max, maxPx) => Math.min(max, Math.floor(maxPx / (String(text).length * 0.62)));

// Deterministic PRNG so every render is pixel-identical.
function rng(seed) {
  let t = seed >>> 0;
  return () => {
    t += 0x6d2b79f5;
    let r = Math.imul(t ^ (t >>> 15), 1 | t);
    r ^= r + Math.imul(r ^ (r >>> 7), 61 | r);
    return ((r ^ (r >>> 14)) >>> 0) / 4294967296;
  };
}

function stars(n, seed, yMax = H, minOp = 0.15, maxOp = 0.85) {
  const r = rng(seed);
  let out = '';
  for (let i = 0; i < n; i++) {
    const x = (r() * W).toFixed(1);
    const y = (r() * yMax).toFixed(1);
    const sz = (0.8 + r() * 1.8).toFixed(1);
    const o = (minOp + r() * (maxOp - minOp)).toFixed(2);
    out += `<circle cx="${x}" cy="${y}" r="${sz}" fill="#fdf6ff" opacity="${o}"/>`;
  }
  return out;
}

// The rainbow wordmark gradient from the seshy launch shots.
function rainbowDef(id = 'rainbow') {
  return `<linearGradient id="${id}" x1="0" y1="0" x2="1" y2="0">
    <stop offset="0" stop-color="${C.yellow}"/><stop offset="0.16" stop-color="${C.orange}"/>
    <stop offset="0.32" stop-color="${C.pink}"/><stop offset="0.5" stop-color="${C.magenta}"/>
    <stop offset="0.68" stop-color="${C.purple}"/><stop offset="0.82" stop-color="#7a3cff"/>
    <stop offset="1" stop-color="${C.cyan}"/>
  </linearGradient>`;
}

function glowFilter(id = 'glow', dev = 7) {
  return `<filter id="${id}" x="-40%" y="-40%" width="180%" height="180%">
    <feGaussianBlur stdDeviation="${dev}" result="b"/>
    <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
  </filter>`;
}

// Nebula washes: purple top-left, cyan top-right, magenta bottom.
function nebula(op = 1) {
  return `
    <ellipse cx="${W * 0.18}" cy="0" rx="${W * 0.9}" ry="${H * 0.35}" fill="url(#nebP)" opacity="${op}"/>
    <ellipse cx="${W * 0.85}" cy="${H * 0.08}" rx="${W * 0.8}" ry="${H * 0.3}" fill="url(#nebC)" opacity="${op}"/>
    <ellipse cx="${W * 0.5}" cy="${H * 1.05}" rx="${W * 1.1}" ry="${H * 0.4}" fill="url(#nebM)" opacity="${op}"/>`;
}

function nebulaDefs() {
  return `
    <radialGradient id="nebP" cx="0.5" cy="0.5" r="0.5"><stop offset="0" stop-color="${C.purple}" stop-opacity="0.26"/><stop offset="1" stop-color="${C.purple}" stop-opacity="0"/></radialGradient>
    <radialGradient id="nebC" cx="0.5" cy="0.5" r="0.5"><stop offset="0" stop-color="${C.cyan}" stop-opacity="0.18"/><stop offset="1" stop-color="${C.cyan}" stop-opacity="0"/></radialGradient>
    <radialGradient id="nebM" cx="0.5" cy="0.5" r="0.5"><stop offset="0" stop-color="${C.magenta}" stop-opacity="0.22"/><stop offset="1" stop-color="${C.magenta}" stop-opacity="0"/></radialGradient>`;
}

function skyDef() {
  return `<linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
    <stop offset="0" stop-color="${C.bg1}"/><stop offset="0.55" stop-color="${C.bg0}"/><stop offset="1" stop-color="#050309"/>
  </linearGradient>`;
}

function vignette(op = 0.55) {
  return `<radialGradient id="vig" cx="0.5" cy="0.46" r="0.75">
      <stop offset="0.54" stop-color="#050010" stop-opacity="0"/><stop offset="1" stop-color="#050010" stop-opacity="${op}"/>
    </radialGradient>`;
}

function scanlines(op = 0.05) {
  let out = `<g opacity="${op}">`;
  for (let y = 0; y < H; y += 4) out += `<rect x="0" y="${y}" width="${W}" height="1" fill="#ffffff"/>`;
  return out + '</g>';
}

// The one footer standard: @handle (if provided) · npx seshy-wrapped ·
// made with seshy. Rendered exactly once per card.
export function footer(handle) {
  return `
    <line x1="${PAD}" y1="${H - 96}" x2="${W - PAD}" y2="${H - 96}" stroke="${C.edge}" stroke-width="1.5"/>
    ${handle ? `<text x="${PAD}" y="${H - 58}" font-family="${MONO}" font-size="28" font-weight="700" fill="${C.ink}">${esc(handle)}</text>` : ''}
    <text x="${PAD}" y="${H - 34}" font-family="${MONO}" font-size="20" font-weight="700" fill="${C.lav}">npx seshy-wrapped</text>
    <text x="${W - PAD}" y="${H - 34}" text-anchor="end" font-family="${LABEL}" font-size="21" fill="${C.fog}">made with seshy</text>`;
}

export function cardMeta(report, opts) {
  const cut = opts.cut || 'classic';
  const agentLabel = report.meta.model
    ? prettyModel(report.deep?.topModel?.id || report.meta.model)
    : { claude: 'Claude Code', codex: 'Codex', gemini: 'Gemini', opencode: 'OpenCode', all: 'AI Coding', pi: 'pi', droid: 'Droid' }[report.meta.agent] || report.meta.agent;
  const year = opts.year || (report.span.lastTs ? new Date(report.span.lastTs).getFullYear() : 2026);
  const tag = report.meta.rookie ? 'ROOKIE CARD' : CUT_TAG[cut] || '';
  const kicker = `${agentLabel.toUpperCase()} · ${year}${tag ? ` · ${tag}` : ''}`;
  return {
    cut,
    kicker,
    head: pickHeadline(report, cut),
    tiles: pickTiles(report, cut),
    sig: signature(report, cut),
    grade: report.grade,
    handle: opts.handle ? String(opts.handle).replace(/^@?/, '@') : '',
  };
}

const svgOpen = `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">`;

// Grade badge with the "why" line under it.
function gradeBadge(cx, cy, grade, ringRef = 'url(#rainbow)') {
  return `
    <circle cx="${cx}" cy="${cy}" r="62" fill="#160c26" stroke="${ringRef}" stroke-width="4" filter="url(#glow)"/>
    <text x="${cx}" y="${cy + 22}" text-anchor="middle" font-family="${DISPLAY}" font-size="66" font-weight="700" fill="${C.ink}">${esc(grade.letter)}</text>
    <text x="${cx}" y="${cy + 94}" text-anchor="middle" font-family="${LABEL}" font-size="19" fill="${C.fog}" letter-spacing="2">DELEGATION GRADE</text>
    <text x="${cx}" y="${cy + 120}" text-anchor="middle" font-family="${LABEL}" font-size="18" fill="${C.dim}" font-style="italic">${esc(grade.why)}</text>`;
}

// ============ STARFIELD — deep space, rainbow hero, glass tiles ============
export function starfield(report, opts) {
  const m = cardMeta(report, opts);
  const tileW = (W - PAD * 2 - 28 * 2) / 3;
  const tileH = 150;
  const gridTop = 742;
  const row2 = gridTop + tileH + 26;
  const sigY = row2 + tileH + 66;

  let tiles = '';
  m.tiles.forEach((t, i) => {
    const x = PAD + (i % 3) * (tileW + 28);
    const y = gridTop + Math.floor(i / 3) * (tileH + 26);
    tiles += `
      <rect x="${x}" y="${y}" width="${tileW}" height="${tileH}" rx="18" fill="#160a28" fill-opacity="0.72" stroke="${t.accent}" stroke-opacity="0.5" stroke-width="1.5"/>
      <circle cx="${x + 26}" cy="${y + 30}" r="4" fill="${t.accent}" filter="url(#glow)"/>
      <text x="${x + 24}" y="${y + 86}" font-family="${MONO}" font-size="52" font-weight="700" fill="${t.accent}" filter="url(#glow)">${esc(t.value)}</text>
      <text x="${x + 25}" y="${y + tileH - 24}" font-family="${LABEL}" font-size="22" fill="${C.lav}">${esc(t.label)}</text>`;
  });

  return `${svgOpen}
  <defs>${skyDef()}${nebulaDefs()}${rainbowDef()}${glowFilter('glow', 8)}${vignette(0.6)}</defs>
  <rect width="${W}" height="${H}" fill="url(#sky)"/>
  ${nebula()}
  ${stars(120, 77)}
  <text x="${PAD}" y="122" font-family="${MONO}" font-size="30" font-weight="700" fill="${C.yellow}" letter-spacing="7" filter="url(#glow)">${esc(m.kicker)}</text>
  <text x="${PAD}" y="198" font-family="${DISPLAY}" font-size="74" font-weight="700" fill="url(#rainbow)" filter="url(#glow)">SESHY WRAPPED</text>
  <text x="${PAD}" y="244" font-family="${MONO}" font-size="21" fill="${C.lav}">${esc(m.head.sub)}</text>

  <text x="${W / 2}" y="520" text-anchor="middle" font-family="${DISPLAY}" font-size="190" font-weight="700" fill="url(#rainbow)" filter="url(#glow)">${esc(m.head.value)}</text>
  <text x="${W / 2}" y="580" text-anchor="middle" font-family="${MONO}" font-size="26" fill="${C.ink}" letter-spacing="5">${esc(m.head.label.toUpperCase())}</text>
  <line x1="${W / 2 - 220}" y1="626" x2="${W / 2 + 220}" y2="626" stroke="url(#rainbow)" stroke-width="2" filter="url(#glow)"/>

  ${tiles}
  ${gradeBadge(W - PAD - 60, sigY + 18, m.grade)}
  <text x="${PAD}" y="${sigY}" font-family="${DISPLAY}" font-size="${m.sig.line1.length > 42 ? 26 : 33}" font-weight="700" fill="${C.yellow}">${esc(m.sig.line1)}</text>
  <text x="${PAD}" y="${sigY + 42}" font-family="${LABEL}" font-size="26" fill="${C.lav}">${esc(m.sig.line2)}</text>
  ${footer(m.handle)}
  <rect width="${W}" height="${H}" fill="url(#vig)"/>
</svg>`;
}

// ============ TERMINAL — the whole card is a floating seshy window ============
export function terminal(report, opts) {
  const m = cardMeta(report, opts);
  const cardX = 64;
  const cardW = W - 128;
  const cardY = 268;
  const cardH = 880;
  const bx = cardX + 44;

  let rows = '';
  m.tiles.forEach((t, i) => {
    const y = cardY + 372 + i * 62;
    rows += `
      <text x="${bx}" y="${y}" font-family="${MONO}" font-size="25" fill="${C.lav}">${esc(t.label)}</text>
      <line x1="${bx + t.label.length * 15 + 18}" y1="${y - 7}" x2="${cardX + cardW - 44 - String(t.value).length * 17 - 20}" y2="${y - 7}" stroke="${C.edge}" stroke-width="1.5" stroke-dasharray="1 6"/>
      <text x="${cardX + cardW - 44}" y="${y}" text-anchor="end" font-family="${MONO}" font-size="28" font-weight="700" fill="${t.accent}" filter="url(#glow)">${esc(t.value)}</text>`;
  });

  let floor = '';
  for (let i = 1; i <= 10; i++) {
    const t = i / 10;
    const y = 900 + (H - 900) * t * t;
    floor += `<line x1="0" y1="${y.toFixed(1)}" x2="${W}" y2="${y.toFixed(1)}" stroke="${C.purple}" stroke-opacity="${(0.3 - 0.2 * t).toFixed(2)}" stroke-width="1.2"/>`;
  }
  for (let i = -7; i <= 7; i++) {
    floor += `<line x1="${W / 2}" y1="900" x2="${W / 2 + i * 170}" y2="${H}" stroke="${C.purple}" stroke-opacity="0.16" stroke-width="1.2"/>`;
  }

  const cmd = `--agent ${report.meta.agent}${report.meta.model ? ` --model ${report.meta.model}` : ''}`;

  return `${svgOpen}
  <defs>${skyDef()}${nebulaDefs()}${rainbowDef()}${glowFilter('glow', 6)}${vignette(0.5)}
    <linearGradient id="win" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#160a28"/><stop offset="1" stop-color="#0d0221"/></linearGradient>
  </defs>
  <rect width="${W}" height="${H}" fill="url(#sky)"/>
  ${nebula(0.9)}
  ${stars(70, 5, 900)}
  ${floor}

  <text x="${W / 2}" y="112" text-anchor="middle" font-family="${MONO}" font-size="30" font-weight="700" fill="${C.yellow}" letter-spacing="7" filter="url(#glow)">${esc(m.kicker)}</text>
  <text x="${W / 2}" y="186" text-anchor="middle" font-family="${DISPLAY}" font-size="70" font-weight="700" fill="url(#rainbow)" filter="url(#glow)">SESHY WRAPPED</text>
  <text x="${W / 2}" y="230" text-anchor="middle" font-family="${MONO}" font-size="20" fill="${C.lav}">${esc(m.head.sub)}</text>

  <rect x="${cardX - 3}" y="${cardY - 3}" width="${cardW + 6}" height="${cardH + 6}" rx="17" fill="${C.purple}" opacity="0.4" filter="url(#glow)"/>
  <rect x="${cardX}" y="${cardY}" width="${cardW}" height="${cardH}" rx="14" fill="url(#win)" stroke="${C.purple}" stroke-opacity="0.55" stroke-width="1.5"/>
  <rect x="${cardX}" y="${cardY}" width="${cardW}" height="46" rx="14" fill="#140a20" fill-opacity="0.95"/>
  <rect x="${cardX}" y="${cardY + 32}" width="${cardW}" height="14" fill="#140a20" fill-opacity="0.95"/>
  <line x1="${cardX}" y1="${cardY + 46}" x2="${cardX + cardW}" y2="${cardY + 46}" stroke="${C.purple}" stroke-opacity="0.35" stroke-width="1"/>
  <circle cx="${cardX + 26}" cy="${cardY + 23}" r="7" fill="#ff5f56"/>
  <circle cx="${cardX + 50}" cy="${cardY + 23}" r="7" fill="#ffbd2e"/>
  <circle cx="${cardX + 74}" cy="${cardY + 23}" r="7" fill="#27c93f"/>
  <text x="${cardX + 100}" y="${cardY + 30}" font-family="${MONO}" font-size="17" fill="#9A89BE">seshy wrapped — your life in the terminal</text>

  <text x="${bx}" y="${cardY + 104}" font-family="${MONO}" font-size="23"><tspan fill="${C.pink}">❯</tspan> <tspan fill="${C.cyan}">npx seshy-wrapped</tspan> <tspan fill="${C.lav}">${esc(cmd)}</tspan></text>

  <text x="${bx}" y="${cardY + 236}" font-family="${MONO}" font-size="110" font-weight="700" fill="${C.ink}" filter="url(#glow)">${esc(m.head.value)}</text>
  <text x="${bx + 4}" y="${cardY + 284}" font-family="${MONO}" font-size="24" fill="${C.pink}" letter-spacing="3">${esc(m.head.label.toUpperCase())}</text>

  <line x1="${bx}" y1="${cardY + 322}" x2="${cardX + cardW - 44}" y2="${cardY + 322}" stroke="${C.edge}" stroke-width="1.5"/>
  ${rows}
  <line x1="${bx}" y1="${cardY + 742}" x2="${cardX + cardW - 44}" y2="${cardY + 742}" stroke="${C.edge}" stroke-width="1.5"/>

  <text x="${bx}" y="${cardY + 796}" font-family="${MONO}" font-size="27" font-weight="700" fill="${C.yellow}">${esc(m.sig.line1)}</text>
  <text x="${bx}" y="${cardY + 832}" font-family="${MONO}" font-size="21" fill="${C.lav}">${esc(m.sig.line2)}</text>
  <text x="${cardX + cardW - 44}" y="${cardY + 810}" text-anchor="end" font-family="${MONO}" font-size="24" fill="${C.dim}">grade <tspan font-size="46" font-weight="700" fill="url(#rainbow)">${esc(m.grade.letter)}</tspan></text>
  <text x="${cardX + cardW - 44}" y="${cardY + 840}" text-anchor="end" font-family="${MONO}" font-size="17" fill="${C.fog}" font-style="italic">${esc(m.grade.why)}</text>

  ${footer(m.handle)}
  <rect width="${W}" height="${H}" fill="url(#vig)"/>
</svg>`;
}

// ============ CRT — phosphor screen, chromatic aberration, REC overlay ============
export function crt(report, opts) {
  const m = cardMeta(report, opts);
  const tileW = (W - PAD * 2 - 24 * 2) / 3;
  const tileH = 146;
  const gridTop = 748;
  const row2 = gridTop + tileH + 24;
  const sigY = row2 + tileH + 64;

  const aber = (x, y, text, size, family = DISPLAY, anchor = 'start', weight = 700) => `
    <text x="${x - 3.5}" y="${y}" text-anchor="${anchor}" font-family="${family}" font-size="${size}" font-weight="${weight}" fill="${C.cyan}" opacity="0.8">${esc(text)}</text>
    <text x="${x + 3.5}" y="${y}" text-anchor="${anchor}" font-family="${family}" font-size="${size}" font-weight="${weight}" fill="${C.magenta}" opacity="0.8">${esc(text)}</text>
    <text x="${x}" y="${y}" text-anchor="${anchor}" font-family="${family}" font-size="${size}" font-weight="${weight}" fill="${C.ink}">${esc(text)}</text>`;

  let tiles = '';
  m.tiles.forEach((t, i) => {
    const x = PAD + (i % 3) * (tileW + 24);
    const y = gridTop + Math.floor(i / 3) * (tileH + 24);
    tiles += `
      <path d="M ${x} ${y + 14} v -14 h 14 M ${x + tileW} ${y + tileH - 14} v 14 h -14" stroke="${t.accent}" stroke-width="2.5" fill="none" opacity="0.9"/>
      <rect x="${x}" y="${y}" width="${tileW}" height="${tileH}" fill="#0f0620" fill-opacity="0.6" stroke="${t.accent}" stroke-opacity="0.25" stroke-width="1"/>
      <text x="${x + 22}" y="${y + 76}" font-family="${MONO}" font-size="50" font-weight="700" fill="${t.accent}" filter="url(#glow)">${esc(t.value)}</text>
      <text x="${x + 23}" y="${y + tileH - 22}" font-family="${MONO}" font-size="15" fill="${C.lav}">${esc(t.label.length > 30 ? t.label.slice(0, 29) + '…' : t.label)}</text>`;
  });

  return `${svgOpen}
  <defs>
    <linearGradient id="crtbg" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#10062a"/><stop offset="0.5" stop-color="#0a0220"/><stop offset="1" stop-color="#04010d"/>
    </linearGradient>
    ${glowFilter('glow', 6)}
    <radialGradient id="crtvig" cx="0.5" cy="0.5" r="0.72">
      <stop offset="0.5" stop-color="#000" stop-opacity="0"/><stop offset="0.92" stop-color="#000" stop-opacity="0.5"/><stop offset="1" stop-color="#000" stop-opacity="0.85"/>
    </radialGradient>
    <radialGradient id="crtsheen" cx="0.5" cy="0.32" r="0.7">
      <stop offset="0" stop-color="#8be9fd" stop-opacity="0.07"/><stop offset="0.6" stop-color="#8be9fd" stop-opacity="0"/>
    </radialGradient>
  </defs>
  <rect width="${W}" height="${H}" fill="#000"/>
  <rect x="10" y="10" width="${W - 20}" height="${H - 20}" rx="46" fill="url(#crtbg)"/>
  <rect x="10" y="10" width="${W - 20}" height="${H - 20}" rx="46" fill="url(#crtsheen)"/>

  <text x="${PAD}" y="124" font-family="${MONO}" font-size="28" font-weight="700" fill="${C.cyan}" letter-spacing="7" filter="url(#glow)">${esc(m.kicker)}</text>
  ${aber(PAD, 202, 'SESHY WRAPPED', 74)}
  <text x="${PAD}" y="246" font-family="${MONO}" font-size="20" fill="${C.lav}">${esc(m.head.sub)}</text>

  <circle cx="${W - PAD - 116}" cy="112" r="9" fill="#ff3b30" filter="url(#glow)"/>
  <text x="${W - PAD - 96}" y="121" font-family="${MONO}" font-size="24" font-weight="700" fill="${C.ink}">REC</text>

  ${aber(W / 2, 512, m.head.value, 176, MONO, 'middle')}
  <text x="${W / 2}" y="568" text-anchor="middle" font-family="${MONO}" font-size="24" fill="${C.yellow}" letter-spacing="6" filter="url(#glow)">${esc(m.head.label.toUpperCase())}</text>
  <text x="${W / 2}" y="640" text-anchor="middle" font-family="${MONO}" font-size="19" fill="${C.fog}">PLAY ▶ TRACKING OK</text>

  ${tiles}

  ${aber(W - PAD - 96, sigY + 8, m.grade.letter, 92, MONO, 'middle')}
  <text x="${W - PAD}" y="${sigY + 78}" text-anchor="end" font-family="${MONO}" font-size="${fitMono('DELEGATION GRADE · ' + m.grade.why, 19, 560)}" fill="${C.fog}">DELEGATION GRADE · ${esc(m.grade.why.toUpperCase())}</text>

  <text x="${PAD}" y="${sigY}" font-family="${MONO}" font-size="${fitMono(m.sig.line1, 28, W - PAD * 2 - 180)}" font-weight="700" fill="${C.yellow}">${esc(m.sig.line1)}</text>
  <text x="${PAD}" y="${sigY + 40}" font-family="${MONO}" font-size="${fitMono(m.sig.line2, 20, W - PAD * 2 - 180)}" fill="${C.lav}">${esc(m.sig.line2)}</text>

  ${footer(m.handle)}
  ${scanlines(0.06)}
  <rect x="10" y="10" width="${W - 20}" height="${H - 20}" rx="46" fill="url(#crtvig)"/>
</svg>`;
}

// ============ RECEIPT — thermal paper on neon dark ============
// The footer standard lives INSIDE the paper here; no outer strip.
export function receipt(report, opts) {
  const m = cardMeta(report, opts);
  const px = 200;
  const pw = W - 400;
  const py = 96;
  const ph = H - 160;
  const cx = W / 2;
  const ink = '#241035';
  const faint = '#6b5585';

  const zig = (y, dir) => {
    let d = `M ${px} ${y}`;
    for (let x = px; x < px + pw; x += 28) d += ` L ${x + 14} ${y + dir * 10} L ${x + 28} ${y}`;
    return d;
  };
  const dash = (y) => `<line x1="${px + 40}" y1="${y}" x2="${px + pw - 40}" y2="${y}" stroke="${faint}" stroke-width="2" stroke-dasharray="8 7"/>`;

  let rows = '';
  m.tiles.forEach((t, i) => {
    const y = py + 452 + i * 62;
    // Ellipsize at the cap instead of hard-chopping — a sliced label reads
    // as a typo on a card people screenshot.
    const label = t.label.toUpperCase();
    const clipped = label.length > 34 ? label.slice(0, 33).trimEnd() + '…' : label;
    rows += `
      <text x="${px + 44}" y="${y}" font-family="${MONO}" font-size="24" fill="${ink}">${esc(clipped)}</text>
      <text x="${px + pw - 44}" y="${y}" text-anchor="end" font-family="${MONO}" font-size="26" font-weight="700" fill="${ink}">${esc(t.value)}</text>`;
  });

  const r = rng(9);
  let bars = '';
  let bxp = px + 110;
  while (bxp < px + pw - 110) {
    const bw = 2 + Math.floor(r() * 4) * 2;
    bars += `<rect x="${bxp}" y="${py + ph - 150}" width="${bw}" height="64" fill="${ink}"/>`;
    bxp += bw + 3 + Math.floor(r() * 5);
  }

  const dateStr = report.span.lastTs ? new Date(report.span.lastTs).toISOString().slice(0, 10) : '';

  return `${svgOpen}
  <defs>${skyDef()}${nebulaDefs()}${glowFilter('glow', 10)}
    <linearGradient id="paper" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#FBF6EC"/><stop offset="1" stop-color="#EFE6DA"/>
    </linearGradient>
  </defs>
  <rect width="${W}" height="${H}" fill="url(#sky)"/>
  ${nebula()}
  ${stars(80, 33)}

  <rect x="${px - 6}" y="${py - 4}" width="${pw + 12}" height="${ph + 8}" fill="${C.magenta}" opacity="0.45" filter="url(#glow)"/>
  <rect x="${px - 6}" y="${py - 4}" width="${pw + 12}" height="${ph + 8}" fill="${C.cyan}" opacity="0.25" filter="url(#glow)"/>
  <path d="${zig(py, -1)} L ${px + pw} ${py + ph} L ${px} ${py + ph} Z" fill="url(#paper)"/>
  <path d="${zig(py + ph, 1)} L ${px + pw} ${py} L ${px} ${py} Z" fill="url(#paper)"/>

  <text x="${cx}" y="${py + 86}" text-anchor="middle" font-family="${MONO}" font-size="44" font-weight="700" fill="${ink}" letter-spacing="4">SESHY WRAPPED</text>
  <text x="${cx}" y="${py + 124}" text-anchor="middle" font-family="${MONO}" font-size="20" fill="${faint}" letter-spacing="3">* SESHY GENERAL COMPUTING *</text>
  <text x="${cx}" y="${py + 156}" text-anchor="middle" font-family="${MONO}" font-size="19" fill="${faint}">${esc(m.kicker)}${dateStr ? `  ·  ${esc(dateStr)}` : ''}</text>
  ${dash(py + 186)}

  <text x="${cx}" y="${py + 310}" text-anchor="middle" font-family="${MONO}" font-size="120" font-weight="700" fill="${ink}">${esc(m.head.value)}</text>
  <text x="${cx}" y="${py + 356}" text-anchor="middle" font-family="${MONO}" font-size="22" fill="${ink}" letter-spacing="3">${esc(m.head.label.toUpperCase())}</text>
  <text x="${cx}" y="${py + 390}" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${faint}">${esc(m.head.sub)}</text>
  ${dash(py + 416)}

  ${rows}
  ${dash(py + 452 + 6 * 62 - 20)}

  <text x="${px + 44}" y="${py + 452 + 6 * 62 + 34}" font-family="${MONO}" font-size="27" font-weight="700" fill="${ink}">TOTAL · DELEGATION GRADE</text>
  <text x="${px + pw - 44}" y="${py + 452 + 6 * 62 + 36}" text-anchor="end" font-family="${MONO}" font-size="42" font-weight="700" fill="${ink}">${esc(m.grade.letter)}</text>
  <text x="${px + 44}" y="${py + 452 + 6 * 62 + 74}" font-family="${MONO}" font-size="19" fill="${faint}">${esc(m.grade.why.toUpperCase())} · CHANGE DUE: SLEEP</text>
  ${dash(py + 452 + 6 * 62 + 100)}

  <text x="${cx}" y="${py + ph - 174}" text-anchor="middle" font-family="${MONO}" font-size="${fitMono(m.sig.line1, 21, pw - 96)}" font-weight="700" fill="${ink}">${esc(m.sig.line1.toUpperCase())}</text>
  ${bars}
  <text x="${cx}" y="${py + ph - 58}" text-anchor="middle" font-family="${MONO}" font-size="20" fill="${ink}" letter-spacing="2">npx seshy-wrapped</text>
  <text x="${cx}" y="${py + ph - 28}" text-anchor="middle" font-family="${MONO}" font-size="18" fill="${faint}">${m.handle ? esc(m.handle) + ' · ' : ''}made with seshy · no refunds</text>
</svg>`;
}

// ============ BILLBOARD — roadside neon billboard over the grid ============
export function billboard(report, opts) {
  const m = cardMeta(report, opts);
  const hy = 470;

  const ridge = (seed, baseY, amp, fill, op) => {
    const r = rng(seed);
    let d = `M 0 ${baseY}`;
    for (let x = 0; x <= W; x += 90) d += ` L ${x + 45} ${baseY - amp * (0.3 + r())} L ${x + 90} ${baseY}`;
    return `<path d="${d} L ${W} ${hy} L 0 ${hy} Z" fill="${fill}" opacity="${op}"/>`;
  };

  let floor = '';
  for (let i = 1; i <= 14; i++) {
    const t = i / 14;
    const y = hy + (H - hy) * t * t;
    floor += `<line x1="0" y1="${y.toFixed(1)}" x2="${W}" y2="${y.toFixed(1)}" stroke="${C.purple}" stroke-opacity="${(0.45 - 0.3 * t).toFixed(2)}" stroke-width="1.5"/>`;
  }
  for (let i = -9; i <= 9; i++) {
    floor += `<line x1="${W / 2}" y1="${hy}" x2="${W / 2 + i * 150}" y2="${H}" stroke="${C.purple}" stroke-opacity="0.25" stroke-width="1.5"/>`;
  }

  const bbX = 92;
  const bbW = W - 184;
  const bbY = 336;
  const bbH = 700;
  const tileW = (bbW - 88 - 24 * 2) / 3;
  const tileH = 128;
  const tTop = bbY + 300;

  let tiles = '';
  m.tiles.forEach((t, i) => {
    const x = bbX + 44 + (i % 3) * (tileW + 24);
    const y = tTop + Math.floor(i / 3) * (tileH + 22);
    tiles += `
      <rect x="${x}" y="${y}" width="${tileW}" height="${tileH}" rx="12" fill="#12071f" stroke="${t.accent}" stroke-opacity="0.5" stroke-width="1.5"/>
      <text x="${x + 20}" y="${y + 62}" font-family="${DISPLAY}" font-size="46" font-weight="700" fill="${t.accent}" filter="url(#glow)">${esc(t.value)}</text>
      <text x="${x + 21}" y="${y + tileH - 20}" font-family="${LABEL}" font-size="19" fill="${C.lav}">${esc(t.label)}</text>`;
  });

  return `${svgOpen}
  <defs>${skyDef()}${rainbowDef()}${glowFilter('glow', 7)}${vignette(0.5)}
    <linearGradient id="sun3" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="${C.yellow}"/><stop offset="0.5" stop-color="${C.orange}"/><stop offset="1" stop-color="${C.magenta}"/>
    </linearGradient>
    <linearGradient id="panel" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#1c0d33"/><stop offset="1" stop-color="#0d0221"/></linearGradient>
    <radialGradient id="halo3" cx="0.5" cy="0.5" r="0.5"><stop offset="0" stop-color="${C.orange}" stop-opacity="0.5"/><stop offset="1" stop-color="${C.orange}" stop-opacity="0"/></radialGradient>
  </defs>
  <rect width="${W}" height="${H}" fill="url(#sky)"/>
  ${stars(70, 55, hy - 60)}
  <circle cx="880" cy="${hy - 70}" r="200" fill="url(#halo3)"/>
  <circle cx="880" cy="${hy - 70}" r="98" fill="url(#sun3)"/>
  ${ridge(3, hy, 130, '#170833', 1)}
  ${ridge(8, hy, 80, '#0e0424', 1)}
  <line x1="0" y1="${hy}" x2="${W}" y2="${hy}" stroke="${C.cyan}" stroke-width="2.5" stroke-opacity="0.85" filter="url(#glow)"/>
  ${floor}

  <rect x="${W / 2 - 130}" y="${bbY + bbH - 8}" width="18" height="${H - bbY - bbH - 120}" fill="#0a0416" stroke="${C.edge}" stroke-width="1"/>
  <rect x="${W / 2 + 112}" y="${bbY + bbH - 8}" width="18" height="${H - bbY - bbH - 120}" fill="#0a0416" stroke="${C.edge}" stroke-width="1"/>
  <rect x="${bbX - 4}" y="${bbY - 4}" width="${bbW + 8}" height="${bbH + 8}" rx="22" fill="${C.magenta}" opacity="0.5" filter="url(#glow)"/>
  <rect x="${bbX}" y="${bbY}" width="${bbW}" height="${bbH}" rx="18" fill="url(#panel)" stroke="url(#rainbow)" stroke-width="3"/>

  <text x="${W / 2}" y="108" text-anchor="middle" font-family="${MONO}" font-size="30" font-weight="700" fill="${C.yellow}" letter-spacing="8" filter="url(#glow)">${esc(m.kicker)}</text>
  <text x="${W / 2}" y="194" text-anchor="middle" font-family="${DISPLAY}" font-size="78" font-weight="700" fill="url(#rainbow)" filter="url(#glow)">SESHY WRAPPED</text>
  <text x="${W / 2}" y="240" text-anchor="middle" font-family="${MONO}" font-size="21" fill="${C.lav}">${esc(m.head.sub)}</text>

  <text x="${W / 2}" y="${bbY + 178}" text-anchor="middle" font-family="${DISPLAY}" font-size="150" font-weight="700" fill="${C.ink}" filter="url(#glow)">${esc(m.head.value)}</text>
  <text x="${W / 2}" y="${bbY + 232}" text-anchor="middle" font-family="${MONO}" font-size="24" fill="${C.pink}" letter-spacing="5">${esc(m.head.label.toUpperCase())}</text>

  ${tiles}

  <text x="${bbX + 44}" y="${bbY + bbH - 56}" font-family="${DISPLAY}" font-size="30" font-weight="700" fill="${C.yellow}">${esc(m.sig.line1)}</text>
  <text x="${bbX + 44}" y="${bbY + bbH - 20}" font-family="${LABEL}" font-size="23" fill="${C.lav}">${esc(m.sig.line2)}</text>
  <circle cx="${bbX + bbW - 92}" cy="${bbY + bbH - 52}" r="48" fill="#160c26" stroke="url(#rainbow)" stroke-width="3.5" filter="url(#glow)"/>
  <text x="${bbX + bbW - 92}" y="${bbY + bbH - 35}" text-anchor="middle" font-family="${DISPLAY}" font-size="50" font-weight="700" fill="${C.ink}">${esc(m.grade.letter)}</text>

  ${footer(m.handle)}
  <rect width="${W}" height="${H}" fill="url(#vig)"/>
</svg>`;
}

export const THEMES = { terminal, starfield, receipt, billboard, crt };
