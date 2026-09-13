# Clanker Chat

A realtime chat web app written in Python, built with
[Starlette](https://www.starlette.io/) and [htmx](https://htmx.org/).
It supports channels, nicknames, @mentions, file attachments, and
channel moderation with simple `!commands`.

## Features

- Channels that are created on demand just by joining a `#name`
- Nicknames with uniqueness handling (`bob`, `bob_1`, `bob_2`, …)
- @mentions with highlighting and unread badges per channel
- `#channel` links inside messages for quick navigation
- Live polling for new messages plus a sidebar with presence roster
- Full 24h history on demand (page loads the last 10 messages instantly)
- File attachments with per-user and server-wide quotas
- Moderation commands: `!help`, `!nick`, `!auth`, `!owner`,
  `!setpassword`, `!kick`
- Locked (password-protected) channels with lock icons and owner badges
- SQLite storage with automatic 24h pruning of messages and files

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin the Python and
Node toolchains (see `mise.toml`).

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
cd py-htmx-chat

# Install the pinned toolchains from mise.toml
mise install

# Run the chat app in development mode
mise run dev

# Build an optimized release package (see [tasks.build] in mise.toml)
mise run build
```

See `mise.toml` for the complete toolchain configuration and available development tasks.

## Testing

```sh
uv run pytest tests -q
```

This runs all 13 tests covering mentions, message rendering, attachment
quotas, the full join/post/poll flow, owners and commands, lock gates and
kicks, on-demand history, legacy schema migration, XSS-safe nicknames,
nickname uniqueness, session handling, and file pruning.

## Formatting

Code formatting is configured through `ruff` with the default style.

```sh
# Check formatting
uv run ruff format --check src tests

# Apply formatting
uv run ruff format src tests
```

`uv run ruff check src tests` is also kept warning-free.

## Project layout

```text
src/
├── clanker_chat/
│   ├── main.py              # Entry point; wires the application together
│   ├── routes.py            # HTTP routes: landing, chat, polling, uploads
│   ├── commands.py          # One handler per !command (sender-only replies)
│   ├── config.py            # Runtime configuration and startup
│   ├── chat/
│   │   ├── format.py        # Linkify #channels / @mentions, nickname rules
│   │   └── render.py        # HTML fragments and channel pills
│   ├── cache/
│   │   └── store.py         # Last-10 messages, presence, mentions, unlocks
│   ├── attachments/
│   │   └── store.py         # Disk store, quotas, prune
│   ├── db/
│   │   ├── database.py      # SQLite access layer
│   │   ├── moderation.py    # Members, owners, passwords, legacy migration
│   │   └── schema.sql       # Database schema
│   ├── templates/           # Landing, chat, and locked pages
│   └── icons.py             # Generated icon paths (from @mdi/js)
├── static/                  # CSS (ANSI palette), JS (polling, completion)
└── tests/
    └── test_chat.py         # Pytest suite (ruff-clean, like everything else)
```

## License

MIT — do whatever you want with it, just don't blame me for your chat history.
