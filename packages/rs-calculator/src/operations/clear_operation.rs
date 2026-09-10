//! Clear operation. Handles Message::Clear. Empties expression, input, and
//! error. The big red AC button. The fresh start. Tabula rasa.
use crate::state::calculator::Calculator;

/// Clears the entire calculator state.
pub fn handle_clear(calc: &mut Calculator) {
  calc.expression.clear();
  calc.input.clear();
  calc.error = None;
}
