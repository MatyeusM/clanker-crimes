# Clanker Sort

A cross-platform desktop app written in Java (Swing) that sorts a messy
folder of files by tags you assign yourself. Point it at a folder, walk
through each file adding `category = value` tags, then sort: originals move
into a hidden `.files/` content store, byte-identical duplicates are folded
away automatically, and one folder per tag value is created with links back
to the real files.

Example: tag movies with an `Actor` category, sort by `Actor`, and you get
`Robert De Niro/GoodFellas.mkv` linking to `.files/GoodFellas.mkv`.

## Features

- Three-screen workflow: browse & scan, one-at-a-time tagging, sort by
  category — files without a value land in `Unsorted/`
- Content hashing (SHA-256, multithreaded) with automatic duplicate
  elimination: identical files are stored once, redundant originals removed
- Persistent `.index.json` tag database: rescans and restarts remember
  everything, matched by hash, so renames and moves don't lose tags
- Comma-separated input splits into multiple values; legacy comma values
  migrate on load
- Destructive-but-honest sorting: after the move, only `.files/` and
  `.index.json` survive — old views, stale links, and leftover folders go
- Link strategy with graceful degradation: symlink, then hard link, then
  copy (whatever the platform allows)
- Ships its own typeface (Lato, OFL) for a unified look on Linux, macOS,
  and Windows — no system fonts required
- Small by design: ~3000 lines of Java across focused files, zero
  third-party dependencies

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin the Temurin JDK
(see `mise.toml`). Nothing else to install — no Maven, no Gradle.

**macOS / Linux:**

```sh
curl https://mise.run | sh
```

**Windows:**

```powershell
winget install jdx.mise
```

The JDK (Temurin 26, targeting Java 25 bytecode) installs itself on first
use. You need a display for the GUI; the test suite runs headless.

## Getting started

```sh
# Install the pinned tools from mise.toml
mise install

# Build and launch the GUI
mise run run

# Or run the packaged artifacts
mise run run-jar       # runnable jar
mise run run-native    # platform-native app image via jpackage
```

See `mise.toml` for the complete tool configuration and available tasks.

## Tagging

Pick a folder and hit **Scan**, then walk through the files. Each file gets
any number of `category = value` tags:

```text
Actor = Robert De Niro, Joe Pesci
Genre = Crime
```

Typing `De Niro, Pesci` in one value field adds two tags. **Open with
default app** lets you inspect each file before tagging. Every edit is
saved to `.index.json` immediately, so you can quit anytime and resume
later. Once everything is tagged, pick a category and hit **Sort Now**.

## CLI

```text
--diagnose-fonts  headless check that the bundled font loads and rasterizes
```

```sh
mise run diagnose-fonts
```

## Verifying

```sh
mise run test   # dependency-free suite: hashing, concurrency, fonts, index, sorting
mise run lint   # fail unless every file is google-java-format clean
```

`javac -Xlint:all` is kept at zero warnings, and the formatter enforces
2-space indent, no tabs.

## Project layout

```text
src/main/java/com/clankerprise/sorting/
├── ClankerSortApplication.java            # Entry point; LAF, fonts, screen wiring
├── domain/                                # File descriptors, tag repository, tag values
├── discovery/                             # Parallel recursive scan + SHA-256 hashing
├── organization/                          # Move-to-store, dedup, wipe, view generation
├── persistence/                           # Dependency-free JSON index codec
├── infrastructure/                        # Hashing factory, link provisioning, opener
├── presentation/                          # Browse/scan, tagging, and sort screens
│   └── typography/                        # Embedded Lato font loading
└── src/main/resources/typography/         # Lato TTFs + OFL license text
src/test/                                   # Five validation modules, no test framework
```

## License

MIT — do whatever you want with it, just don't blame me for your folder.
