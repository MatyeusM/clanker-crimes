//! The calculator's state. Yes, the whole struct. It gets its own file now.
//!
//! commit slop001: extracted from main.rs. Fields went pub(crate) so the other
//! 15 files in this architecture can reach in and touch them. Encapsulation
//! is a legacy concept. See operations/ for the former methods.
use clanker_calculator::Token;

/// The calculator's state.
///
/// `expression` holds committed number/operator tokens; `input` holds the
/// digits currently being typed for the *next* operand.
#[derive(Debug, Default)]
pub struct Calculator {
  pub(crate) expression: Vec<Token>,
  pub(crate) input: String,
  pub(crate) deg_mode: bool,
  pub(crate) inv_mode: bool,
  pub(crate) error: Option<String>,
}
