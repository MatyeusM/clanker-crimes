# Clanker TODO

A fast local-first todo list built with [Preact](https://preactjs.com/),
[Vite](https://vitejs.dev/), and TypeScript. Add tasks, set priorities,
filter and search, and track your progress — everything is validated with
[zod](https://zod.dev/) and persisted to `localStorage`, so your list
survives a refresh.

## Features

- Add tasks with a title, optional notes, and a low/medium/high priority
- Mark tasks done, edit them inline, or delete them
- Filter by all / active / completed, with live counts
- Search across titles and notes
- Progress bar showing done vs. total
- Light / dark mode switcher (system preference by default, remembered
  in `localStorage`)
- 30 seeded demo tasks on first run; resetting the demo restores the
  seeds without wiping tasks you added yourself
- Scrollable list (7 visible rows, then it scrolls) with a custom scrollbar
- Small motion details: hover lift on the add button, pop on filter tabs,
  slide-in for tasks (respects `prefers-reduced-motion`)
- Unique ids via [nanoid](https://github.com/ai/nanoid), icons via
  [lucide](https://lucide.dev/)

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
cd ts-preact-todo

# Install the pinned toolchain from mise.toml
mise install

# Install dependencies
mise run setup

# Run the app in development mode
mise run dev

# Build an optimized production bundle (see [tasks.build] in mise.toml)
mise run build
```

Or, with pnpm directly:

```sh
pnpm install
pnpm dev
pnpm build
pnpm preview
```

See `mise.toml` for the complete toolchain configuration and available
development tasks.

## Typechecking and linting

```sh
# Typecheck (dist/ is excluded in tsconfig.json)
pnpm exec tsc --noEmit

# Lint
pnpm lint

# Check formatting
pnpm fmt:check

# Apply formatting
pnpm fmt
```

`oxlint` (correctness) and `oxfmt` are kept clean.

## Project layout

```text
src/
├── index.tsx               # Entry point; router wiring and #app render
├── style.css               # Global theme, variables, header styles
├── lib/
│   ├── todos.ts            # Zod schemas, localStorage helpers, factories
│   ├── Seed.ts             # 30 seeded demo tasks
│   └── theme.ts            # Light/dark theme hook and helpers
├── components/
│   ├── Header.tsx          # Brand, nav, theme switcher
│   ├── TodoComposer.tsx    # New-task form with validation
│   └── TodoItem.tsx        # Task row with toggle, edit, delete
├── pages/
│   ├── Home/
│   │   ├── index.tsx       # Task list page: filters, search, progress
│   │   └── style.css       # Page styles (BEM class names)
│   └── _404.tsx            # Not-found page
index.html                  # Shell, fonts, theme init script
mise.toml                   # Pinned Node/pnpm toolchain and tasks
oxfmt.config.ts             # Formatter config
oxlint.config.ts            # Linter config
```

## License

MIT — do whatever you want with it, just don't blame me for your
unfinished todos.
