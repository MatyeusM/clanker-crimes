//! Equals operation. Handles Message::Equals. Commits the operand,
//! evaluates the whole expression, shows the result or "Error", then clears
//! the expression. The climax of every calculation.
use super::input_helpers::{commit_operand, set_error};
use crate::formatting::number_format_trait::{FormatterFactory, NumberFormatter};
use crate::state::calculator::Calculator;
use clanker_calculator::evaluate;

/// Evaluates the current expression and displays the result.
pub fn handle_equals(calc: &mut Calculator) {
  commit_operand(calc);
  if calc.expression.is_empty() {
    return;
  }
  let formatter = FormatterFactory::create_default();
  match evaluate(&calc.expression) {
    Ok((result, _)) => {
      calc.input = formatter.format_number(result);
      calc.error = None;
    }
    Err(_) => set_error(calc, "Error"),
  }
  calc.expression.clear();
}
