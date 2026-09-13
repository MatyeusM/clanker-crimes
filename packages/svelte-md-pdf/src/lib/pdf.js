import { writable } from 'svelte/store';

// Inline-level parsing: markdown-it inline children -> flat style runs.
// Run shape: { text, bold, italic, code, href }

/**
 * Turns a markdown-it inline token into flat style runs.
 * Each run keeps its own bold/italic/code/link flags so the renderer
 * can switch jsPDF fonts mid-line.
 *
 * @param {object} inlineToken markdown-it inline token (needs .children)
 * @returns {Array<{text: string, bold: boolean, italic: boolean, code: boolean, href: string|null}>}
 */
export function parseRuns(inlineToken) {
  const runs = [];
  const children = inlineToken?.children ?? [];
  let bold = 0;
  let italic = 0;
  let href = null;

  const push = (text, code = false) => {
    if (!text) return;
    runs.push({ text, bold: bold > 0, italic: italic > 0, code, href });
  };

  for (const tok of children) {
    switch (tok.type) {
      case 'text':
        push(tok.content);
        break;
      case 'softbreak':
      case 'hardbreak':
        push(' ');
        break;
      case 'code_inline':
        push(tok.content, true);
        break;
      case 'strong_open':
        bold += 1;
        break;
      case 'strong_close':
        bold = Math.max(0, bold - 1);
        break;
      case 'em_open':
        italic += 1;
        break;
      case 'em_close':
        italic = Math.max(0, italic - 1);
        break;
      case 'link_open':
        href = tok.attrGet ? tok.attrGet('href') : attrHref(tok);
        break;
      case 'link_close':
        href = null;
        break;
      case 'image': {
        // Images are out of scope: fall back to alt text.
        const alt = (tok.children ?? []).map((c) => c.content ?? '').join('');
        push(alt || 'image');
        break;
      }
      default:
        break;
    }
  }
  return mergeRuns(runs);
}

export function plainText(runs) {
  return (runs ?? []).map((r) => r.text).join('');
}

function attrHref(tok) {
  const found = (tok.attrs ?? []).find((a) => a[0] === 'href');
  return found ? found[1] : null;
}

/**
 * Merges neighbouring runs that share the same style flags.
 * Keeps the segment count down so PDFs with lots of bold text
 * don't end up with thousands of tiny text ops.
 *
 * TODO: check if this actually matters, numbers looked fine either way
 *
 * @param {Array} items the runs to merge (mutates the last one, careful)
 * @returns {Array} merged runs, empty-text runs dropped
 */
function mergeRuns(runs) {
  const out = [];
  for (const r of runs) {
    const prev = out[out.length - 1];
    if (
      prev &&
      prev.bold === r.bold &&
      prev.italic === r.italic &&
      prev.code === r.code &&
      prev.href === r.href
    ) {
      prev.text += r.text;
    } else {
      out.push({ ...r });
    }
  }
  return out.filter((r) => r.text.length > 0);
}
// Block-level parsing: markdown-it token stream -> flat document model.
// Block shapes:
//   { type: 'heading', level: 1|2|3, runs }
//   { type: 'paragraph', runs }
//   { type: 'code', content, lang }
//   { type: 'list', ordered, items: runs[][] }

// markdown-it loads on first parse, not with the initial bundle.
let _md = null;
async function getMd() {
  if (!_md) {
    const { default: MarkdownIt } = await import('markdown-it');
    _md = new MarkdownIt({ html: false, linkify: true, breaks: false });
  }
  return _md;
}

/**
 * Parses Markdown source into a flat list of typed blocks.
 * Supported: headings (h1-h3, deeper levels are clamped), paragraphs,
 * fenced/indented code, single-level bullet/ordered lists. Everything
 * else (tables, html, hrs...) is silently skipped to stay in budget.
 *
 * @param {string} src raw markdown
 * @returns {Array<object>} blocks, see block shapes in the README (TODO: move them here)
 */
export async function parseMarkdown(src) {
  const md = await getMd();
  const tokens = md.parse(src ?? '', {});
  const blocks = [];
  let i = 0;

  while (i < tokens.length) {
    const tok = tokens[i];

    if (tok.type === 'heading_open') {
      const level = Math.min(Math.max(parseInt(tok.tag.slice(1), 10) || 1, 1), 3);
      const inline = tokens[i + 1];
      const runs = inline?.type === 'inline' ? parseRuns(inline) : [];
      if (runs.length) blocks.push({ type: 'heading', level, runs });
      i += 3;
    } else if (tok.type === 'paragraph_open') {
      const inline = tokens[i + 1];
      const runs = inline?.type === 'inline' ? parseRuns(inline) : [];
      if (runs.length) blocks.push({ type: 'paragraph', runs });
      i += 3;
    } else if (tok.type === 'fence' || tok.type === 'code_block') {
      blocks.push({
        type: 'code',
        content: (tok.content ?? '').replace(/\n$/, ''),
        lang: (tok.info ?? '').trim().split(/\s+/)[0] ?? '',
      });
      i += 1;
    } else if (tok.type === 'bullet_list_open' || tok.type === 'ordered_list_open') {
      const ordered = tok.type === 'ordered_list_open';
      const { items, next } = readList(tokens, i);
      if (items.length) blocks.push({ type: 'list', ordered, items });
      i = next;
    } else {
      // hr, html_block, tables (no plugin enabled), etc. are out of scope.
      i += 1;
    }
  }
  return blocks;
}

function readList(tokens, start) {
  const open = tokens[start];
  const closeType =
    open.type === 'ordered_list_open' ? 'ordered_list_close' : 'bullet_list_close';
  const items = [];
  let current = null;
  let i = start + 1;

  while (i < tokens.length) {
    const t = tokens[i];
    if (t.type === 'list_item_open') {
      current = [];
    } else if (t.type === 'list_item_close') {
      if (current) items.push(flattenItem(current));
      current = null;
    } else if (t.type === 'inline' && current) {
      current.push(parseRuns(t));
    } else if (
      (t.type === 'bullet_list_open' || t.type === 'ordered_list_open') &&
      current
    ) {
      // Nested lists are out of scope: skip the whole nested list.
      i = skipBalanced(tokens, i);
      continue;
    } else if (t.type === closeType) {
      return { items, next: i + 1 };
    }
    i += 1;
  }
  return { items, next: i };
}

function flattenItem(paragraphs) {
  const out = [];
  for (const runs of paragraphs) {
    if (out.length) out.push({ text: ' ', bold: false, italic: false, code: false, href: null });
    out.push(...runs);
  }
  return out;
}

function skipBalanced(tokens, start) {
  const open = tokens[start].type;
  const close =
    open === 'bullet_list_open'
      ? 'bullet_list_close'
      : open === 'ordered_list_open'
        ? 'ordered_list_close'
        : null;
  let depth = 0;
  let i = start;
  while (i < tokens.length) {
    if (tokens[i].type === open) depth += 1;
    if (tokens[i].type === close) {
      depth -= 1;
      if (depth === 0) return i + 1;
    }
    i += 1;
  }
  return i;
}
// Text measurement + word wrapping on top of a jsPDF instance.
// A wrapped line is an array of segments: { text, run, width }.

export function styleOf(run) {
  if (run.bold && run.italic) return 'bolditalic';
  if (run.bold) return 'bold';
  if (run.italic) return 'italic';
  return 'normal';
}

export function applyRunFont(doc, run, family, size) {
  doc.setFont(run.code ? 'courier' : family, styleOf(run));
  doc.setFontSize(run.code ? size.code : size.body);
}

export function lineHeight(ptSize, factor = 1.35) {
  return ptSize * 0.3528 * factor;
}

// Greedy word-wrap across mixed-style runs. Whitespace is collapsed,
// like HTML. Returns an array of lines (each a segment array).
/**
 * Greedy word-wrap across mixed-style runs. Whitespace is collapsed,
 * like HTML. Measures with the live jsPDF instance so widths are exact
 * for whatever font is currently set.
 *
 * @param {object} doc jsPDF instance
 * @param {Array} runs style runs from parseRuns
 * @param {number} maxWidth column width in mm
 * @param {{family: string, body: number, code: number}} fonts sizes
 * @returns {Array<Array<{text: string, run: object, width: number}>>} lines
 */
export function wrapRuns(doc, runs, maxWidth, fonts) {
  const { family, body, code } = fonts;
  const lines = [];
  let line = [];
  let width = 0;

  const commit = () => {
    if (line.length) lines.push(line);
    line = [];
    width = 0;
  };

  for (const run of runs) {
    applyRunFont(doc, run, family, { body, code });
    const words = run.text.split(/\s+/).filter(Boolean);
    for (const word of words) {
      const w = doc.getTextWidth(word);
      if (w > maxWidth) {
        // Single word wider than the column: hard-split it.
        commit();
        const chunks = splitWord(doc, word, maxWidth);
        for (let c = 0; c < chunks.length; c++) {
          if (c > 0) commit();
          line.push({ text: chunks[c], run, width: doc.getTextWidth(chunks[c]) });
          width += line[line.length - 1].width;
        }
        commit();
        continue;
      }
      const gap = line.length ? doc.getTextWidth(' ') : 0;
      if (width + gap + w > maxWidth && line.length) commit();
      if (line.length) {
        const sw = doc.getTextWidth(' ');
        line.push({ text: ' ', run, width: sw });
        width += sw;
      }
      line.push({ text: word, run, width: w });
      width += w;
    }
  }
  commit();
  return lines.length ? lines : [[]];
}

// Char-level wrap for code blocks (whitespace preserved per source line).
/**
 * Char-level wrap for code blocks. Whitespace is preserved per source line,
 * unlike wrapRuns.
 *
 * @param {string} doc the code text (one string, may contain newlines)
 * @param {string} content same thing, split into lines first
 * @param {number} maxWidth available width in mm
 * @param {number} size font size in pt
 */
export function wrapCodeText(doc, content, maxWidth, size) {
  doc.setFont('courier', 'normal');
  doc.setFontSize(size);
  const out = [];
  for (const raw of (content || '').split('\n')) {
    if (!raw) {
      out.push('');
      continue;
    }
    let cur = '';
    for (const ch of raw) {
      if (doc.getTextWidth(cur + ch) > maxWidth && cur) {
        out.push(cur);
        cur = ch;
      } else {
        cur += ch;
      }
    }
    out.push(cur);
  }
  return out.length ? out : [''];
}

function splitWord(doc, word, maxWidth) {
  const chunks = [];
  let cur = '';
  for (const ch of word) {
    if (cur && doc.getTextWidth(cur + ch) > maxWidth) {
      chunks.push(cur);
      cur = ch;
    } else {
      cur += ch;
    }
  }
  if (cur) chunks.push(cur);
  return chunks.length ? chunks : [word];
}
// Block renderers: document-model blocks -> jsPDF drawing commands.
// Pagination lives here per block; the driver (compile.js) owns the
// outline tree and the heading widow/orphan lookahead.
// ctx: { x, maxWidth, topMargin, pageBottom, state: { y },
//        fonts: { heading, body } (jsPDF family names),
//        ensureSpace(h), newPage(), onHeading(title, level) }

const BODY = { family: 'helvetica', size: 11 };
const CODE_SIZE = 9.5;
const INK = [26, 26, 26];
const LINK = [26, 70, 192];
const MUTED = [120, 120, 120];
const CODE_BG = [245, 245, 245];

/** Heading sizes in pt + spacing in mm. h1 gets a rule. */
const HEADINGS = {
  1: { size: 20, before: 8, after: 5, rule: true },
  2: { size: 16, before: 6, after: 4, rule: false },
  3: { size: 13, before: 5, after: 3, rule: false },
};

function atTop(ctx) {
  return ctx.state.y <= ctx.topMargin + 1e-6;
}

// --- headings -----------------------------------------------------------

export function headingKeepHeight(doc, block, next, maxWidth) {
  const cfg = HEADINGS[block.level];
  const lh = lineHeight(cfg.size, 1.25);
  const lines = wrapRuns(doc, block.runs, maxWidth, fontsFor(cfg.size));
  let keep = lines.length * lh + cfg.after + (cfg.rule ? 3 : 0);
  keep += firstLineOf(doc, next, maxWidth);
  return keep;
}

function firstLineOf(doc, block, maxWidth) {
  if (!block) return 0;
  if (block.type === 'heading') return lineHeight(HEADINGS[block.level].size, 1.25);
  if (block.type === 'code') return lineHeight(CODE_SIZE, 1.3);
  return lineHeight(BODY.size);
}

/**
 * Draws a heading and registers the outline/bookmark entry.
 * h1 gets a rule underneath because it looked bare without one.
 *
 * @param {object} doc
 * @param {object} block heading block ({level, runs})
 * @param {object} ctx layout context, state.y is advanced past the heading
 */
export function renderHeading(doc, block, ctx) {
  const cfg = HEADINGS[block.level];
  const lh = lineHeight(cfg.size, 1.25);
  const lines = wrapRuns(doc, block.runs, ctx.maxWidth, fontsFor(cfg.size, ctx.fonts.heading));

  if (!atTop(ctx)) ctx.state.y += cfg.before;
  ctx.ensureSpace(lines.length * lh + cfg.after + (cfg.rule ? 3 : 0));

  for (const line of lines) {
    ctx.ensureSpace(lh);
    drawRichLine(doc, line, ctx.x, ctx.state.y + lh * 0.8, cfg.size, 0, ctx.fonts.heading);
    ctx.state.y += lh;
  }
  if (cfg.rule) {
    doc.setDrawColor(...MUTED);
    doc.setLineWidth(0.3);
    doc.line(ctx.x, ctx.state.y + 1, ctx.x + ctx.maxWidth, ctx.state.y + 1);
    ctx.state.y += 3;
  }
  ctx.state.y += cfg.after;
  ctx.onHeading(plainText(block.runs).trim() || 'Untitled', block.level);
}

// --- paragraphs ---------------------------------------------------------

export function renderParagraph(doc, block, ctx) {
  const lh = lineHeight(BODY.size);
  const lines = wrapRuns(doc, block.runs, ctx.maxWidth, fontsFor(BODY.size, ctx.fonts.body));
  if (!lines.length) return;

  if (!atTop(ctx)) ctx.state.y += 2;
  let i = 0;
  while (i < lines.length) {
    const left = lines.length - i;
    if (i === 0 && left > 2 && fitsLines(ctx, lh) < 2) ctx.newPage();
    if (left === 2 && ctx.state.y + 2 * lh > ctx.pageBottom && ctx.state.y + lh <= ctx.pageBottom) {
      ctx.newPage(); // keep the final two lines together (widow)
    }
    ctx.ensureSpace(lh);
    // Block text: justify every line but the last, which stays ragged.
    const fill = i === lines.length - 1 ? 0 : ctx.maxWidth;
    drawRichLine(doc, lines[i], ctx.x, ctx.state.y + lh * 0.8, BODY.size, fill, ctx.fonts.body);
    ctx.state.y += lh;
    i += 1;
  }
  ctx.state.y += 2;
}

// --- code blocks ---------------------------------------------------------

/**
 * Draws a code block with a grey background. The background is painted
 * per line (plus top/bottom padding rows) so it survives page breaks.
 *
 * @param {object} doc
 * @param {{content: string}} block
 * @param {object} ctx
 */
export function renderCodeBlock(doc, block, ctx) {
  const padX = 3;
  const padY = 2;
  const lh = lineHeight(CODE_SIZE, 1.3);
  const lines = wrapCodeText(doc, block.content, ctx.maxWidth - padX * 2, CODE_SIZE);

  ctx.state.y += 1;
  if (lines.length > 2 && fitsLines(ctx, lh) < 2) ctx.newPage();
  paintCodeLine(doc, ctx, '', padX, padY, lh, true);
  for (const text of lines) {
    ctx.ensureSpace(lh);
    paintCodeLine(doc, ctx, text, padX, padY, lh, false);
  }
  paintCodeLine(doc, ctx, '', padX, padY, lh, true);
  ctx.state.y += 3;
}

function paintCodeLine(doc, ctx, text, padX, padY, lh, spacer) {
  const h = spacer ? padY : lh;
  ctx.ensureSpace(h);
  doc.setFillColor(...CODE_BG);
  doc.rect(ctx.x, ctx.state.y, ctx.maxWidth, h, 'F');
  if (!spacer) {
    doc.setFont('courier', 'normal');
    doc.setFontSize(CODE_SIZE);
    doc.setTextColor(...INK);
    doc.text(text || ' ', ctx.x + padX, ctx.state.y + lh * 0.78);
  }
  ctx.state.y += h;
}

// --- lists ---------------------------------------------------------------

export function renderList(doc, block, ctx) {
  const lh = lineHeight(BODY.size);
  const indent = 8;
  ctx.state.y += 1;
  block.items.forEach((item, idx) => {
    const marker = block.ordered ? `${idx + 1}.` : '•';
    const lines = wrapRuns(doc, item, ctx.maxWidth - indent, fontsFor(BODY.size, ctx.fonts.body));
    if (!lines.length) return;
    // Keep short items together with their bullet.
    if (lines.length <= 3) ctx.ensureSpace(lines.length * lh + 1.5);
    lines.forEach((line, li) => {
      ctx.ensureSpace(lh);
      const baseline = ctx.state.y + lh * 0.8;
      if (li === 0) {
        doc.setFont(ctx.fonts.body, block.ordered ? 'bold' : 'normal');
        doc.setFontSize(BODY.size);
        doc.setTextColor(...INK);
        doc.text(marker, ctx.x, baseline);
      }
      drawRichLine(doc, line, ctx.x + indent, baseline, BODY.size, 0, ctx.fonts.body);
      ctx.state.y += lh;
    });
    ctx.state.y += 1.5;
  });
  ctx.state.y += 2;
}

// --- rich line drawing -----------------------------------------------------

/**
 * Draws one wrapped line, switching fonts/colors per segment. Link
 * segments are drawn blue with an underline via textWithLink.
 * Pass targetWidth to justify the line (extra space goes into the gaps).
 *
 * @param {object} doc jsPDF instance
 * @param {Array} segments from wrapRuns
 * @param {number} x left edge in mm
 * @param {number} baselineY baseline in mm (jsPDF text y is the baseline)
 * @param {number} bodySize body font size in pt
 * @param {number} [targetWidth=0] justify to this width, 0 = ragged right
 * @param {string} [family] jsPDF font family for non-code runs
 */
function drawRichLine(doc, segments, x, baselineY, bodySize, targetWidth = 0, family = BODY.family) {
  let cursor = x;
  // Justification: spread leftover space across the word gaps. Short lines
  // (< 70% full) stay left-aligned to avoid rivers of whitespace.
  let extraPerGap = 0;
  if (targetWidth > 0) {
    const used = segments.reduce((sum, seg) => sum + seg.width, 0);
    const gaps = segments.filter((seg) => seg.text === ' ').length;
    if (gaps > 0 && targetWidth > used && used / targetWidth >= 0.7) {
      extraPerGap = (targetWidth - used) / gaps;
    }
  }
  for (const seg of segments) {
    if (!seg.text) continue;
    applyRunFont(doc, seg.run, family, { body: bodySize, code: CODE_SIZE });
    if (seg.run.href) {
      doc.setTextColor(...LINK);
      doc.textWithLink(seg.text, cursor, baselineY, { url: seg.run.href });
      doc.setDrawColor(...LINK);
      doc.setLineWidth(0.2);
      doc.line(cursor, baselineY + 0.9, cursor + seg.width, baselineY + 0.9);
    } else {
      doc.setTextColor(...INK);
      doc.text(seg.text, cursor, baselineY);
    }
    cursor += seg.width + (seg.text === ' ' ? extraPerGap : 0);
  }
}

function fontsFor(body, family = BODY.family) {
  return { family, body, code: Math.min(body - 1, CODE_SIZE) };
}

function fitsLines(ctx, lh) {
  return Math.floor((ctx.pageBottom - ctx.state.y) / lh);
}
// Driver: document model -> paginated jsPDF with a nested outline tree.
// Layout rules:
// - h1 chapters always start on a fresh page.
// - Headings never strand alone at the bottom of a page (see headingKeepHeight).
// - Optional page numbers, centered in the bottom margin.

/**
 * Compiles the document model into a paginated PDF with a nested
 * outline tree (h1 top-level, h2/h3 under the nearest preceding parent).
 * h1 chapters always start on a fresh page. Optionally stamps page
 * numbers centered in the bottom margin.
 *
 * @param {Array<object>} blocks from parseMarkdown
 * @param {object} settings headingFont, bodyFont, paper, margins, pageNumbers
 * @returns {Promise<{doc: object, pageCount: number}>}
 */
export async function compileToPdf(blocks, settings) {
  const { jsPDF } = await import('jspdf');
  const paper = PAPER_SIZES[settings.paper] ?? PAPER_SIZES.a4;
  const m = settings.margins;
  const doc = new jsPDF({
    unit: 'mm',
    format: [paper.width, paper.height],
    orientation: paper.width > paper.height ? 'landscape' : 'portrait',
  });
  doc.setProperties({ title: 'Markdown Export', creator: 'svelte-md-pdf' });

  const fontNames = await ensureDocFonts(doc, settings.headingFont, settings.bodyFont);

  const pageWidth = doc.internal.pageSize.getWidth();
  const pageHeight = doc.internal.pageSize.getHeight();
  const maxWidth = pageWidth - m.left - m.right;
  const pageBottom = pageHeight - m.bottom;
  const state = { y: m.top };

  let lastH1 = null;
  let lastH2 = null;

  const ctx = {
    x: m.left,
    maxWidth,
    topMargin: m.top,
    pageBottom,
    state,
    fonts: {
      heading: fontNames[settings.headingFont],
      body: fontNames[settings.bodyFont],
    },
  };

  function newPage() {
    doc.addPage();
    state.y = m.top;
  }

/**
 * Makes room for h mm. Adds a page if the block would overflow.
 * @param {number} h height in mm
 */
  function ensureSpace(h) {
    if (state.y + h <= pageBottom + 1e-6) return false;
    newPage();
    return true;
  }

  function onHeading(title, level) {
    const pageNumber = doc.getNumberOfPages();
    const parent = level === 1 ? null : level === 2 ? lastH1 : (lastH2 ?? lastH1);
    let node = null;
    try {
      node = doc.outline.add(parent ?? null, title || 'Untitled', { pageNumber });
    } catch {
      node = doc.outline.add(null, title || 'Untitled', { pageNumber });
    }
    if (level === 1) {
      lastH1 = node;
      lastH2 = null;
    } else if (level === 2) {
      lastH2 = node;
    }
  }

  Object.assign(ctx, { ensureSpace, newPage, onHeading });

  if (!blocks.length) {
    doc.setFont(ctx.fonts.body, 'italic');
    doc.setFontSize(11);
    doc.setTextColor(120);
    doc.text('Empty document — nothing to compile.', m.left, state.y);
  }

  blocks.forEach((block, i) => {
    const next = blocks[i + 1] ?? null;
    if (block.type === 'heading' && block.level === 1) {
      if (state.y > m.top + 1e-6) newPage();
    }
    if (block.type === 'heading') {
      ensureSpace(headingKeepHeight(doc, block, next, maxWidth));
      renderHeading(doc, block, ctx);
    } else if (block.type === 'paragraph') {
      renderParagraph(doc, block, ctx);
    } else if (block.type === 'code') {
      renderCodeBlock(doc, block, ctx);
    } else if (block.type === 'list') {
      renderList(doc, block, ctx);
    }
  });

  if (settings.pageNumbers) {
    const total = doc.getNumberOfPages();
    doc.setFont(ctx.fonts.body, 'normal');
    doc.setFontSize(9);
    doc.setTextColor(110);
    for (let p = 1; p <= total; p++) {
      doc.setPage(p);
      doc.text(String(p), pageWidth / 2, pageHeight - m.bottom / 2, { align: 'center' });
    }
  }

  return { doc, pageCount: doc.getNumberOfPages() };
}
// Font registry + jsPDF loader.
// Custom families live in public/fonts and are fetched on demand at compile
// time, so they never bloat the JS bundle. Built-ins need no loading.
// Note: the VFS cache is per-document (WeakMap), since jsPDF instances
// each own their virtual file system.

const base = import.meta.env?.BASE_URL || '/';

const ttf = (family, name) => `${base}fonts/${family}/${name}`;

export const FONTS = [
  {
    id: 'pt-serif',
    label: 'PT Serif',
    files: {
      normal: ttf('pt-serif', 'PTSerif-Regular.ttf'),
      italic: ttf('pt-serif', 'PTSerif-Italic.ttf'),
      bold: ttf('pt-serif', 'PTSerif-Bold.ttf'),
      bolditalic: ttf('pt-serif', 'PTSerif-BoldItalic.ttf'),
    },
  },
  {
    id: 'pt-sans',
    label: 'PT Sans',
    files: {
      normal: ttf('pt-sans', 'PTSans-Regular.ttf'),
      italic: ttf('pt-sans', 'PTSans-Italic.ttf'),
      bold: ttf('pt-sans', 'PTSans-Bold.ttf'),
      bolditalic: ttf('pt-sans', 'PTSans-BoldItalic.ttf'),
    },
  },
  {
    id: 'roboto',
    label: 'Roboto',
    files: {
      normal: ttf('roboto', 'Roboto-Regular.ttf'),
      italic: ttf('roboto', 'Roboto-Italic.ttf'),
      bold: ttf('roboto', 'Roboto-Bold.ttf'),
      bolditalic: ttf('roboto', 'Roboto-BoldItalic.ttf'),
    },
  },
  {
    id: 'cormorant',
    label: 'Cormorant Garamond',
    files: {
      normal: ttf('cormorant-garamond', 'CormorantGaramond-Regular.ttf'),
      italic: ttf('cormorant-garamond', 'CormorantGaramond-Italic.ttf'),
      bold: ttf('cormorant-garamond', 'CormorantGaramond-Bold.ttf'),
      bolditalic: ttf('cormorant-garamond', 'CormorantGaramond-BoldItalic.ttf'),
    },
  },
  {
    id: 'pt-mono',
    label: 'PT Mono',
    // Ships only Regular; every style maps to it.
    files: {
      normal: ttf('pt-mono', 'PTMono-Regular.ttf'),
      italic: ttf('pt-mono', 'PTMono-Regular.ttf'),
      bold: ttf('pt-mono', 'PTMono-Regular.ttf'),
      bolditalic: ttf('pt-mono', 'PTMono-Regular.ttf'),
    },
  },
  { id: 'helvetica', label: 'Helvetica (built-in)', builtin: 'helvetica' },
  { id: 'times', label: 'Times (built-in)', builtin: 'times' },
  { id: 'courier', label: 'Courier (built-in)', builtin: 'courier' },
];

const loadedByDoc = new WeakMap();

// Module-level fetch cache so idle prefetching and compile-time loading
// share the same downloads.
const bufferCache = new Map();

function fetchBuffer(url) {
  let pending = bufferCache.get(url);
  if (!pending) {
    pending = fetch(url).then((res) => {
      if (!res.ok) throw new Error(`Could not load font ${url}`);
      return res.arrayBuffer();
    });
    bufferCache.set(url, pending);
  }
  return pending;
}

// Download (but don't register) the given families. Safe to call repeatedly;
// failures reject and are simply retried at compile time.
export function prefetchFonts(...ids) {
  const urls = new Set();
  for (const id of new Set(ids)) {
    const entry = fontById(id);
    if (!entry.builtin) {
      for (const url of Object.values(entry.files)) urls.add(url);
    }
  }
  return Promise.all([...urls].map((url) => fetchBuffer(url))).then(() => {});
}

export function fontById(id) {
  return FONTS.find((f) => f.id === id) ?? FONTS[0];
}

// Registers the given families on the doc. Returns a map of
// font id -> jsPDF family name to pass to setFont.
/**
 * Registers the given font families on the doc (fetch + addFileToVFS +
 * addFont per style). Built-ins skip loading. Results are cached per doc.
 *
 * @param {object} doc
 * @param {...string} ids font ids from FONTS
 * @returns {Promise<object>} id -> jsPDF family name for setFont
 */
export async function ensureDocFonts(doc, ...ids) {
  let loaded = loadedByDoc.get(doc);
  if (!loaded) {
    loaded = new Set();
    loadedByDoc.set(doc, loaded);
  }
  const names = {};
  for (const id of new Set(ids)) {
    const entry = fontById(id);
    if (entry.builtin) {
      names[id] = entry.builtin;
      continue;
    }
    if (!loaded.has(id)) {
      for (const [style, url] of Object.entries(entry.files)) {
        const vfsName = `${id}-${style}.ttf`;
        doc.addFileToVFS(vfsName, arrayBufferToBase64(await fetchBuffer(url)));
        doc.addFont(vfsName, id, style);
      }
      loaded.add(id);
    }
    names[id] = id;
  }
  return names;
}

function arrayBufferToBase64(buf) {
  const bytes = new Uint8Array(buf);
  let out = '';
  const CHUNK = 0x8000;
  for (let i = 0; i < bytes.length; i += CHUNK) {
    out += String.fromCharCode.apply(null, bytes.subarray(i, i + CHUNK));
  }
  return btoa(out);
}
// Document settings: paper, margins (geometry-style, in mm), page numbers, fonts.
// Applied at Compile time only — editing settings never recompiles by itself.

export const PAPER_SIZES = {
  a4: { label: 'A4', width: 210, height: 297 },
  letter: { label: 'US Letter', width: 215.9, height: 279.4 },
  a5: { label: 'A5', width: 148, height: 210 },
  legal: { label: 'US Legal', width: 215.9, height: 355.6 },
  a3: { label: 'A3', width: 297, height: 420 },
};

export const DEFAULT_SETTINGS = {
  headingFont: 'pt-sans',
  bodyFont: 'pt-serif',
  paper: 'a4',
  pageNumbers: true,
  margins: { top: 25, bottom: 25, left: 25, right: 25 },
};

export const settings = writable(structuredClone(DEFAULT_SETTINGS));

export function resetSettings() {
  settings.set(structuredClone(DEFAULT_SETTINGS));
}

/**
 * @param {string} id paper id, e.g. 'a4'
 */
export function paperLabel(id) {
  return PAPER_SIZES[id]?.label ?? PAPER_SIZES.a4.label;
}
// The compiled PDF blob. Set once per explicit Compile action so the
// preview pane can react to a new compile independently of the editor.
export const pdfBlob = writable(null);

// { state: 'idle' | 'compiling' | 'done' | 'error', message, pageCount }
export const compileStatus = writable({
  state: 'idle',
  message: 'Not compiled yet.',
  pageCount: 0,
});
/** Demo document shown on first visit. Covers every supported feature. */
export const SAMPLE_MARKDOWN = `# User Guide

Welcome to the **Markdown to PDF** demo. This *sample document* shows every
supported feature, including \`inline code\` and a [link](https://example.com).

## Getting Started

Paste Markdown on the left, press **Compile**, and read the PDF on the right.

### Headings

Headings become PDF bookmarks. An \`h1\` is a top-level entry, \`h2\` and
\`h3\` nest under the nearest preceding parent.

## Lists

Things to try:

- Compile this document
- Open the bookmarks panel in a PDF reader
- Download the result and share it

Steps in order:

1. Edit the Markdown
2. Press Compile
3. Preview and download

## Code

\`\`\`js
function greet(name) {
  return 'Hello, ' + name + '!';
}
\`\`\`

That's all — happy typesetting!
`;
