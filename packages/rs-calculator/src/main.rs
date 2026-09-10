//! Clanker Calculator — Main entry point using iced v0.14.
//!
//! State is built directly as a `Vec<Token>` from button presses and
//! evaluated with `clanker_calculator::evaluate`. The display string is
//! rendered separately, purely for showing the user what's been entered —
//! it is never re-parsed, so there is no mismatch between the pretty
//! unicode operator glyphs (×, ÷) shown on screen and what actually gets
//! evaluated.
//!
//! ARCHITECTURE:
//!   main.rs ........... this file. Declares modules, runs the app runner.
//!   app/ .............. config + runner (2 files to start the program).
//!   state/ ............ Calculator struct, factory, render service.
//!   messages/ ......... Message enum, factory, utils.
//!   operations/ ....... one file per button action (11 files, RIP impl block).
//!   formatting/ ....... number + glyph services, each with a trait.
//!   keyboard/ ......... 3-stage mapping pipeline + strategy trait + factory.
//!   update/ ........... boot/update/subscription coordinators.
//!   buttons/ .......... normal factory, wide factory, config, style provider.
//!   display/ .......... display factory + style provider.
//!   ui/ ............... layout constants, 7 row factories, column + view
//!                       assemblers.
//!   window/ ........... title provider + window settings factory.
//!   core/ ............. Core engine.
//!   lib.rs ............ Additional core logic.

mod app;
mod buttons;
mod display;
mod formatting;
mod keyboard;
mod messages;
mod operations;
mod state;
mod ui;
mod update;
mod window;

// --- Entry point ---

fn main() -> iced::Result {
  app::app_runner::run_app()
}
