//! Dot operation. Handles Message::Dot. Ensures "0." prefix and a single
//! dot max. The gatekeeper of decimal points.
use super::input_helpers::clear_error;
use crate::state::calculator::Calculator;

/// Pushes a decimal point onto the current input (at most one).
pub fn handle_dot(calc: &mut Calculator) {
  clear_error(calc);
  if calc.input.is_empty() {
    calc.input.push('0');
  }
  if !calc.input.contains('.') {
    calc.input.push('.');
  }
}
