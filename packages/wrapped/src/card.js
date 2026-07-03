// Render the canonical report into a card PNG. Default theme: sunset — the
// outrun grid-and-sun look ported from seshy's palette. Other themes live in
// themes.js. All rendering is SVG -> resvg; no browser, no model calls.
import { Resvg } from '@resvg/resvg-js';
import { THEMES, W, H, esc, footer, cardMeta } from './themes.js';

const C = {
  bg0: '#0a0713', bg1: '#1a0f2e', ink: '#F5EEFF', lav: '#B8A6D9', fog: '#6C5C8A',
  pink: '#FF6AC1', magenta: '#F92AAD', purple: '#B026FF', violet: '#9D4EDD',
  cyan: '#2DE2E6', aqua: '#05D9E8', yellow: '#FFD319', orange: '#FF8E42',
  surface: '#241B2F', edge: '#3A1C63',
};

const PAD = 72;
const DISPLAY = 'Futura, Avenir Next Condensed, Impact, sans-serif';
const LABEL = 'Avenir Next Condensed, Futura, Helvetica Neue, sans-serif';
const MONO = 'JetBrains Mono, Menlo, monospace';

export const THEME_NAMES = ['sunset', ...Object.keys(THEMES)];

function defs() {
  return `
  <defs>
    <linearGradient id="sky" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="${C.bg1}"/>
      <stop offset="0.55" stop-color="${C.bg0}"/>
      <stop offset="1" stop-color="#050309"/>
    </linearGradient>
    <linearGradient id="sun" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="${C.yellow}"/>
      <stop offset="0.45" stop-color="${C.orange}"/>
      <stop offset="1" stop-color="${C.magenta}"/>
    </linearGradient>
    <linearGradient id="neon" x1="0" y1="0" x2="1" y2="0">
      <stop offset="0" stop-color="${C.magenta}"/>
      <stop offset="0.5" stop-color="${C.pink}"/>
      <stop offset="1" stop-color="${C.cyan}"/>
    </linearGradient>
    <linearGradient id="tile" x1="0" y1="0" x2="0" y2="1">
      <stop offset="0" stop-color="#241B2F" stop-opacity="0.92"/>
      <stop offset="1" stop-color="#160c26" stop-opacity="0.92"/>
    </linearGradient>
    <radialGradient id="sunGlow" cx="0.5" cy="0.5" r="0.5">
      <stop offset="0" stop-color="${C.orange}" stop-opacity="0.55"/>
      <stop offset="1" stop-color="${C.orange}" stop-opacity="0"/>
    </radialGradient>
    <radialGradient id="plate" cx="0.5" cy="0.5" r="0.5">
      <stop offset="0" stop-color="${C.bg0}" stop-opacity="0.9"/>
      <stop offset="0.68" stop-color="${C.bg0}" stop-opacity="0.6"/>
      <stop offset="1" stop-color="${C.bg0}" stop-opacity="0"/>
    </radialGradient>
    <filter id="glow" x="-40%" y="-40%" width="180%" height="180%">
      <feGaussianBlur stdDeviation="7" result="b"/>
      <feMerge><feMergeNode in="b"/><feMergeNode in="SourceGraphic"/></feMerge>
    </filter>
  </defs>`;
}

// The outrun sun: gradient disc, sliced by widening dark bands toward the base.
function sun(cx, cy, r) {
  let bands = '';
  for (let i = 0; i < 7; i++) {
    const y = cy - r * 0.15 + i * (r * 0.16);
    const h = 4 + i * 2.4;
    bands += `<rect x="${cx - r}" y="${y}" width="${r * 2}" height="${h}" fill="${C.bg0}"/>`;
  }
  return `
    <circle cx="${cx}" cy="${cy}" r="${r * 1.7}" fill="url(#sunGlow)"/>
    <clipPath id="sunclip"><circle cx="${cx}" cy="${cy}" r="${r}"/></clipPath>
    <g clip-path="url(#sunclip)">
      <circle cx="${cx}" cy="${cy}" r="${r}" fill="url(#sun)"/>
      ${bands}
    </g>`;
}

// Perspective floor grid receding to a horizon at hy.
function grid(hy) {
  const vp = W / 2;
  let lines = '';
  for (let i = 1; i <= 16; i++) {
    const t = i / 16;
    const y = hy + (H - hy) * (t * t);
    const o = 0.5 - 0.32 * t;
    lines += `<line x1="0" y1="${y.toFixed(1)}" x2="${W}" y2="${y.toFixed(1)}" stroke="${C.purple}" stroke-opacity="${o.toFixed(2)}" stroke-width="1.5"/>`;
  }
  for (let i = -9; i <= 9; i++) {
    const xBottom = vp + i * 150;
    lines += `<line x1="${vp}" y1="${hy}" x2="${xBottom}" y2="${H}" stroke="${C.purple}" stroke-opacity="0.28" stroke-width="1.5"/>`;
  }
  return `<g>${lines}
    <line x1="0" y1="${hy}" x2="${W}" y2="${hy}" stroke="${C.cyan}" stroke-width="2.5" stroke-opacity="0.85" filter="url(#glow)"/>
  </g>`;
}

function tile(x, y, w, h, value, label, accent) {
  return `
  <g>
    <rect x="${x}" y="${y}" width="${w}" height="${h}" rx="20" fill="url(#tile)" stroke="${accent}" stroke-opacity="0.55" stroke-width="1.5"/>
    <rect x="${x}" y="${y}" width="6" height="${h}" rx="3" fill="${accent}"/>
    <text x="${x + 26}" y="${y + 74}" font-family="${DISPLAY}" font-size="60" font-weight="700" fill="${accent}" filter="url(#glow)">${esc(value)}</text>
    <text x="${x + 27}" y="${y + h - 26}" font-family="${LABEL}" font-size="23" fill="${C.lav}">${esc(label)}</text>
  </g>`;
}

function sunsetTheme(report, opts) {
  const m = cardMeta(report, opts);
  const hy = 360;
  const heroY = 636;
  const gridTop = 772;
  const tileW = (W - PAD * 2 - 28 * 2) / 3;
  const tileH = 150;
  const row2 = gridTop + tileH + 26;
  const sigY = row2 + tileH + 62;
  const badgeCy = row2 + tileH + 12 + 60;

  let tileSVG = '';
  m.tiles.slice(0, 6).forEach((t, i) => {
    const col = i % 3;
    const row = Math.floor(i / 3);
    const x = PAD + col * (tileW + 28);
    const y = gridTop + row * (tileH + 26);
    tileSVG += tile(x, y, tileW, tileH, t.value, t.label, t.accent);
  });

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${W}" height="${H}" viewBox="0 0 ${W} ${H}">
    ${defs()}
    <rect width="${W}" height="${H}" fill="url(#sky)"/>
    ${sun(W / 2, 262, 138)}
    ${grid(hy)}

    <text x="${PAD}" y="118" font-family="${LABEL}" font-size="26" fill="${C.cyan}" letter-spacing="8">${esc(m.kicker)}</text>
    <text x="${PAD}" y="196" font-family="${DISPLAY}" font-size="88" font-weight="700" fill="url(#neon)" filter="url(#glow)">SESHY WRAPPED</text>
    <text x="${PAD}" y="240" font-family="${LABEL}" font-size="24" fill="${C.lav}">${esc(m.head.sub)}</text>

    <ellipse cx="${W / 2}" cy="${heroY - 26}" rx="360" ry="132" fill="url(#plate)"/>
    <text x="${W / 2}" y="${heroY}" text-anchor="middle" font-family="${DISPLAY}" font-size="150" font-weight="700" fill="${C.ink}" filter="url(#glow)">${esc(m.head.value)}</text>
    <text x="${W / 2}" y="${heroY + 52}" text-anchor="middle" font-family="${LABEL}" font-size="28" fill="${C.pink}" letter-spacing="4">${esc(m.head.label.toUpperCase())}</text>

    ${tileSVG}

    <circle cx="${W - PAD - 60}" cy="${badgeCy}" r="66" fill="#160c26" stroke="url(#neon)" stroke-width="5" filter="url(#glow)"/>
    <text x="${W - PAD - 60}" y="${badgeCy + 22}" text-anchor="middle" font-family="${DISPLAY}" font-size="72" font-weight="700" fill="${C.ink}">${esc(m.grade.letter)}</text>
    <text x="${W - PAD - 60}" y="${badgeCy + 100}" text-anchor="middle" font-family="${LABEL}" font-size="21" fill="${C.fog}" letter-spacing="2">DELEGATION GRADE</text>
    <text x="${W - PAD - 60}" y="${badgeCy + 126}" text-anchor="middle" font-family="${LABEL}" font-size="18" fill="${C.fog}" font-style="italic">${esc(m.grade.why)}</text>

    <text x="${PAD}" y="${sigY}" font-family="${DISPLAY}" font-size="33" font-weight="700" fill="${C.yellow}">${esc(m.sig.line1)}</text>
    <text x="${PAD}" y="${sigY + 40}" font-family="${LABEL}" font-size="26" fill="${C.lav}">${esc(m.sig.line2)}</text>

    ${footer(m.handle)}
  </svg>`;
}

// buildSVG(report, opts) — opts: { theme, cut, handle, year }
export function buildSVG(report, opts = {}) {
  const theme = opts.theme && opts.theme !== 'sunset' ? THEMES[opts.theme] : null;
  return theme ? theme(report, opts) : sunsetTheme(report, opts);
}

// 1600x2000 output (4:5, X-optimized) — 1600 is ~1.5x the 1080 logical canvas,
// crisp on retina without multi-MB files.
export function renderPNG(svg) {
  const r = new Resvg(svg, {
    background: C.bg0,
    fitTo: { mode: 'width', value: 1600 },
    font: { loadSystemFonts: true, defaultFontFamily: 'Menlo' },
  });
  return r.render().asPng();
}
