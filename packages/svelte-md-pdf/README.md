# Markdown → PDF

A fully client-side single-page app that compiles Markdown into a properly
typeset, paginated PDF with a navigable outline (bookmarks) — LaTeX-style,
no backend, no round-trip. Built with [Svelte](https://svelte.dev/),
[Vite](https://vite.dev/), [markdown-it](https://github.com/markdown-it/markdown-it),
[jsPDF](https://github.com/parallax/jsPDF), and [pdf.js](https://mozilla.github.io/pdf.js/).
Paste or drop in a `.md` file, press **Compile**, preview, download.

## Features

- Markdown input pane: type, paste, open a file, or drag-and-drop a `.md`
  anywhere onto the editor
- Explicit **Compile** action (no recompile-on-keystroke) that parses via
  markdown-it into a typed document model, then lays out a paginated PDF
- Headings (h1–h3) become nested PDF outline/bookmark entries; every h1
  chapter starts on a fresh page
- Justified block text with widow/orphan avoidance (headings never strand
  at a page bottom)
- Bold/italic, inline code, fenced code blocks, single-level ordered and
  unordered lists, links rendered as underlined clickable text
- Settings modal: heading + body fonts independently, paper size (A4, US
  Letter, A5, US Legal, A3), geometry-style margins in mm, optional page
  numbers centered in the bottom margin — all applied on the next Compile
- Five embedded font families (PT Serif, PT Sans, Roboto, Cormorant
  Garamond, PT Mono, all OFL-licensed) fetched on demand and prefetched
  while the browser is idle, plus the jsPDF built-ins
- In-app pdf.js preview: prev/next navigation, Ctrl/Cmd + scroll to zoom,
  pages scroll naturally instead of being squeezed to fit
- One-click PDF download of the compiled blob
- Small initial bundle: markdown-it, jsPDF, and pdf.js each load on demand
  as separate chunks (~69 KB entry)

Out of scope by design: tables, images, nested lists, syntax highlighting,
custom themes.

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin Node and pnpm
(see `mise.toml`).

**macOS / Linux:**

```sh
curl https://mise.run | sh
```

**Windows:**

```powershell
winget install jdx.mise
```

## Getting started

```sh
# Clone the repository, then enter it
cd svelte-md-pdf

# Install the pinned toolchain from mise.toml
mise install

# Install dependencies
pnpm install

# Run the app in development mode
mise run dev

# Build an optimized production bundle (see [tasks.build] in mise.toml)
mise run build
```

Or, with pnpm directly (tasks run `pnpm exec vite` under the hood):

```sh
pnpm install
pnpm dev
pnpm build
pnpm preview
```

See `mise.toml` for the complete toolchain configuration and available
development tasks.

## Scripts and bundles

```sh
pnpm dev      # Vite dev server with HMR
pnpm build    # Production build to dist/
pnpm preview  # Serve the production build locally
```

The production bundle is code-split by route of use, not just by vendor:
`index` (app shell), `markdown-it` (loaded on first Compile), `jspdf`
(loaded on first Compile), `pdf` (pdf.js, loaded on first preview),
plus the pdf.js worker and the `public/fonts` TTFs, which are fetched
at compile time and never bundled.

## Project layout

```text
src/
├── main.js                 # Entry point; mounts App to #app
├── app.css                 # Global styles
├── App.svelte              # Header, Compile/Download actions, pane wiring,
│                           # idle font prefetching, dynamic heavy imports
│   └── lib/
│       ├── pdf.js            # Everything: parsing, layout, jsPDF driver,
│       │                     # fonts, settings, stores, sample doc (~900 LOC,
│       │                     # documented in places)
│       ├── Editor.svelte   # Markdown input pane with file open + drag-drop
│       ├── Preview.svelte  # pdf.js canvas preview, page nav, wheel zoom
│       └── Settings.svelte # Settings modal dialog (some CSS lives in app.css)
public/
└── fonts/                  # OFL-licensed TTFs (Regular/Italic/Bold/BoldItalic)
index.html                   # Shell
mise.toml                    # Pinned Node/pnpm toolchain and tasks
```

## How it works

Markdown is parsed with markdown-it (never hand-rolled) into a flat list
of typed blocks. The compiler walks that model with a tracked Y position,
calling `doc.addPage()` on overflow, and registers each heading with
`doc.outline.add()` under its nearest preceding parent — the same shape
LaTeX derives from `\chapter`/`\section`. The resulting blob feeds both
the pdf.js preview and the download button.

## License

MIT — do whatever you want with it, just don't blame me for your
unfinished documents.
