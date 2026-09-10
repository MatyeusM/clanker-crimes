//! Digit operation. Handles Message::Digit. Appends a digit, replacing a
//! lone "0". Clears errors first, because digits heal all wounds.
use super::input_helpers::clear_error;
use crate::state::calculator::Calculator;

/// Pushes a digit onto the current input.
pub fn handle_digit(calc: &mut Calculator, d: char) {
  clear_error(calc);
  if calc.input == "0" {
    calc.input.clear();
  }
  calc.input.push(d);
}
