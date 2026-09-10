# Clanker Calculator

A scientific desktop calculator written in Rust, built with the
[iced](https://iced.rs/) GUI toolkit. It supports chained arithmetic,
parentheses, scientific functions, and full keyboard plus numpad input.

## Features

- Basic arithmetic (`+`, `−`, `×`, `÷`), powers (`xʸ`), n-th roots (`ʸ√x`)
- Parenthesized expressions with correct operator precedence
- Scientific functions: `sin`/`cos`/`tan` (and inverses), `ln`/`log`,
  `eˣ`/`10ˣ`, square root, square, factorial, percent, negation
- Degree/radian modes and an INV toggle for inverse functions
- Constants: π and e
- Full keyboard support, including the numpad (digits, operators, decimal
  separator, Enter for `=`, Backspace)
- Expression engine: tokenizer plus shunting-yard evaluator with 22 unit tests

## Prerequisites

This project uses [mise](https://mise.jdx.dev/) to pin the Rust toolchain
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
cd rs-calculator

# Install the pinned toolchain from mise.toml
mise install

# Run the calculator in development mode
mise run dev

# Build an optimized release binary (see [tasks.build] in mise.toml)
mise run build
```

See `mise.toml` for the complete toolchain configuration and available development tasks.

## Testing

```sh
cargo test
```

This runs the all tests covering the tokenizer, the shunting-yard
evaluator, arithmetic edge cases (division by zero, precedence,
parentheses), and the scientific functions.

## Formatting

Code formatting is configured through `rustfmt.toml` with a 2-space indentation style.

```sh
# Check formatting
cargo fmt --all -- --check

# Apply formatting
cargo fmt --all
```

`cargo clippy --all-targets` is also kept warning-free.

## Project layout

```text
src/
├── main.rs        # Entry point; wires the application together
├── lib.rs         # Core engine: tokenizer, evaluator, arithmetic, tests
├── app/           # Runtime configuration and startup
├── state/         # Calculator state, construction, display rendering
├── messages/      # UI messages, constructors, and helpers
├── operations/    # One handler per calculator action
├── formatting/    # Number formatting and operator glyphs
├── keyboard/      # Keyboard/numpad mapping pipeline
├── update/        # State updates and event subscriptions
├── buttons/       # Button widgets and styles
├── display/       # Result display widget
├── ui/            # Layout constants, button-grid rows, view assembly
├── window/        # Window title and settings
└── core/          # Core engine
```

## License

MIT — do whatever you want with it, just don't blame me for your math homework.
