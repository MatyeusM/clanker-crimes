//! Backspace operation. Handles Message::Backspace. Pops one char. The ⌫
//! button and the Backspace key (and NumpadBackspace, for the numpad elite).
use super::input_helpers::clear_error;
use crate::state::calculator::Calculator;

/// Deletes the last input character.
pub fn handle_backspace(calc: &mut Calculator) {
  clear_error(calc);
  calc.input.pop();
}
