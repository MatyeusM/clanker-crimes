//! Constant operation. Handles Message::Constant. Replaces the input with
//! π or e, formatted. The simplest file. It knows what it is.
use super::input_helpers::clear_error;
use crate::formatting::number_format_trait::{FormatterFactory, NumberFormatter};
use crate::state::calculator::Calculator;

/// Pushes a constant (π, e) as the current input.
pub fn handle_constant(calc: &mut Calculator, value: f64) {
  clear_error(calc);
  let formatter = FormatterFactory::create_default();
  calc.input = formatter.format_number(value);
}
