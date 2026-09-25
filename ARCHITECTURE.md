# Clanker Crimes — Architecture

A static "spot the bad AI code" game. Each case hands the player a working,
AI-written portfolio project; the code is where the crimes hide. The player
browses the repo, submits a verdict (which crimes are present), and advances
blindly through cases. Wrong verdicts 404, correct ones resolve to the next case.

## Repo layout

```
src/
  pages/
    index.astro        landing page
    repo.astro         case viewer (route /repo?case=<slug>)
  components/          Header, Footer, Button, Welcome
  layouts/Layout.astro site shell
  styles/              global.css (tokens), fonts.css
  data.json            crime catalogue (the curriculum)
packages/
  <name>/              one playable project each ("crime scenes")
    mise.toml          pinned toolchain + setup/dev/build tasks
scripts/
  build.mjs            case-builder entry point (runs after `astro build`)
  build/*.mjs          pipeline modules (one concern each)
  english.txt          wordlist for random case slugs
.env.local.json        gitignored: [{ pkg, crimes }] — solutions per package
dist/case/<slug>/      generated cases (gitignored build output)
```

Route/data separation is load-bearing: the viewer lives at `/repo`, case data
at `/case/<slug>/`. Never add a page route at `/case` — the builder wipes and
rebuilds `dist/case/` from scratch, which would delete the route's own HTML
(this already bit us once with `case.astro`).

## Case-building pipeline (`pnpm build` = `astro build && node scripts/build.mjs`)

`scripts/build.mjs` orchestrates; modules in `scripts/build/`:

| module          | job                                                           |
| --------------- | ------------------------------------------------------------- |
| `config.mjs`    | load/validate `.env.local.json` + `src/data.json`, check refs |
| `slugs.mjs`     | random 4-word slugs from `english.txt`, `alpha` always first  |
| `files.mjs`     | package walk, skip rules, `ext:`-prefix applicability         |
| `highlight.mjs` | shiki (ayu-light/ayu-dark) → `light/`, `dark/`, `files.json`  |
| `archive.mjs`   | `zip` package (minus build junk) → `artifact.zip`             |
| `preview.mjs`   | vite-only single-file build → `preview.html`, tmp dir removed |
| `cases.mjs`     | `choices.json` + hashed solution file, chained case order     |

Per case in `dist/case/<slug>/`:

- `light/<path>.html`, `dark/<path>.html` — highlighted source, theme-matched
- `files.json` — `[{ path, size, binary? }]` manifest driving the file tree
  (binary font/media files are listed with `binary: true` and no HTML; the
  viewer shows a download hint for those instead of code)
- `artifact.zip` — full package (keeps lockfiles, drops `node_modules/`, `target/`, `dist/`, `.git/`)
- `preview.html` — vite projects only (see preview policy)
- `meta.json` — `{ hasPreview }`, so the viewer never probes for files
- `choices.json` — up to 24 crimes (`{ key, name, description }`), solutions shuffled in
- `<sha256(slug + bits)>.json` — `{ url: <next slug> | null }`

Rebuilds are randomized every compile: slug assignment, choice order, and
distractor picks all change. Difficulty varies per build by design.

## Runtime protocol (viewer ↔ generated data)

1. `repo.astro` reads `?case=<slug>` (legacy `?repo=` also accepted), fetches
   `files.json` + `choices.json` + `meta.json` in parallel.
2. File tree + shiki HTML render client-side. Code theme follows the site theme
   (`localStorage`, falls back to `prefers-color-scheme`); light/dark artifacts
   are pre-rendered, no client highlighting.
3. Verdict pane (sticky bottom, collapsed by default): checkboxes in `choices.json`
   order → bit string → `crypto.subtle.digest('SHA-256', slug + bits)` →
   fetch `<hash>.json`. A hit opens the triumph overlay; intermediate cases
   advance when the player clicks **Next case**, while the final case reveals
   the victory panel. A miss shows an error. No hand-rolled crypto on either
   side (build uses `node:crypto`, frontend uses Web Crypto — same digest).
4. Victory panel: share/copy-link actions appear after the final triumph step.

The hash is obscurity, not security: everything needed to cheat ships in `dist/`
by design. It just makes casual cheating harder.

## Crime catalogue (`src/data.json`)

Entries: `{ key, name, description }`. Keys are optionally language-scoped with
an `ext:` prefix — `ts:any-spam`, `rs:unwrap-spam`. Unprefixed crimes apply to
every package; prefixed ones only enter the distractor pool for packages
containing a matching extension (`LANG_ALIASES` in `files.mjs`; unknown prefixes
match literally, so `py:` works with no build changes). Solutions from
`.env.local.json` are always included regardless of prefix. Keys must match
`/^([A-Za-z0-9-]+:)?[A-Za-z0-9-]+$/`, no duplicates. Validation accepts `label`
as an alias of `name`, and `.env.local.json` accepts the legacy `{ pkg: [...] }`
object form.

## Preview policy

- Vite SPA → copy to `tmp-<pkg>/`, inject `vite-plugin-singlefile` (resolved
  from the repo root, so no registry install for the plugin), `pnpm install` +
  `pnpm build`, move `dist/index.html` → `preview.html`, delete tmp. The
  original package is never touched. Build failures warn and skip (no preview).
- Everything else (Rust, and all future server-side projects like Python +
  htmx) → no preview, ever. Static hosting can't run a backend, and there is
  no static-export trick worth the complexity. Code browser + zip is the game.

## Secrets policy

`.env.local.json` is gitignored and, in CI, generated from a GitHub secret.
The builder's default output mentions only original package names and counts —
never slugs, hashes, or file contents. `node scripts/build.mjs --debug` prints
the secrets for local debugging. Keep it that way: no new logging of generated
names.

## Package conventions (`packages/<name>/`)

- Self-contained and buildable on any OS via **mise**: each package pins its
  toolchain and exposes `setup`/`dev`/`build` tasks in its own `mise.toml`,
  so the local testing chain is just `mise run <task>`.
- Portfolio-realistic, crime-bearing: standard projects (calculator, todo list,
  …) with genuine AI-style smells baked in, matching entries in `data.json`.
- Solutions for a package live in `.env.local.json`, never in the package itself.

## Roadmap

Target: ≥7 projects before publish. Planned languages: C, C++, Rust, Java, Go,
Python, JS/TS + frameworks — standard portfolio projects each.

## Known limitations / future work

- Post-verdict explanations don't exist yet: crimes have no `explanation` or
  file pointers, so the teachable moment after submit is wasted. Highest-value
  addition to `data.json`.
- No difficulty curve: single pool, random distractors. Fine for now, revisit
  past ~5 packages.
- `scripts/english.txt` slug words are filtered to `[a-z]{4,8}`; longer words
  produced unreadable slugs.
