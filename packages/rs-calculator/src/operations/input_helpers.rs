//! Input helpers. Shared micro-operations used by several operation files:
//! reading the current value, committing operands, clearing/setting errors.
//! The most imported file in the crate. The social butterfly of modules.
use crate::state::calculator::Calculator;
use clanker_calculator::Token;

/// The value currently being typed (or 0 if nothing has been typed yet).
pub fn current_value(calc: &Calculator) -> f64 {
  if calc.input.is_empty() {
    0.0
  } else {
    calc.input.parse().unwrap_or(0.0)
  }
}

/// Clears any displayed error.
pub fn clear_error(calc: &mut Calculator) {
  calc.error = None;
}

/// Shows an error and clears the current input.
pub fn set_error(calc: &mut Calculator, msg: &str) {
  calc.error = Some(msg.to_string());
  calc.input.clear();
}

/// Commits whatever is currently typed as a Number token. If nothing
/// was typed (e.g. pressing "+" right after "=", or as the very first
/// button), commits an implicit 0 so the expression stays well-formed.
pub fn commit_operand(calc: &mut Calculator) {
  if !calc.input.is_empty() {
    let v: f64 = calc.input.parse().unwrap_or(0.0);
    calc.expression.push(Token::Number(v));
    calc.input.clear();
  } else if calc.expression.is_empty()
    || matches!(calc.expression.last(), Some(Token::Operator(op)) if !matches!(op, clanker_calculator::OpToken::CloseParen))
  {
    calc.expression.push(Token::Number(0.0));
  }
}
