//! Unary-op operation. Handles Message::Unary. Applies the scientific
//! function to the current value immediately (never enters the token stream).
use super::input_helpers::{clear_error, current_value, set_error};
use crate::formatting::number_format_trait::{FormatterFactory, NumberFormatter};
use crate::state::calculator::Calculator;
use clanker_calculator::{UnaryFn, apply_unary};

/// Applies a unary scientific function to the current input.
pub fn handle_unary(calc: &mut Calculator, f: UnaryFn) {
  clear_error(calc);
  let x = current_value(calc);
  // Format through the shared formatter so display rules live in one place
  // (plus its trait, plus its factory — the whole formatting family).
  let formatter = FormatterFactory::create_default();
  match apply_unary(f, x, calc.deg_mode) {
    Ok(result) => calc.input = formatter.format_number(result),
    Err(_) => set_error(calc, "Error"),
  }
}
