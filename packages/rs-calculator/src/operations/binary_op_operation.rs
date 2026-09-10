//! Binary-op operation. Handles Message::BinaryOp. Commits the pending
//! operand, then pushes the operator. OpenParen gets special treatment: it
//! commits a typed number but never invents a synthetic 0 just to open a
//! group (see the commit_operand docs for the phantom-zero saga).
use super::input_helpers::commit_operand;
use crate::state::calculator::Calculator;
use clanker_calculator::{OpToken, Token};

/// Pushes a binary operator (or paren) onto the expression.
pub fn handle_binary_op(calc: &mut Calculator, op: OpToken) {
  calc.error = None;
  if matches!(op, OpToken::OpenParen) {
    // Commit a number the user already typed (if any), but don't
    // insert a synthetic 0 just to open a group.
    if !calc.input.is_empty() {
      let v: f64 = calc.input.parse().unwrap_or(0.0);
      calc.expression.push(Token::Number(v));
      calc.input.clear();
    }
    calc.expression.push(Token::Operator(op));
    return;
  }
  commit_operand(calc);
  calc.expression.push(Token::Operator(op));
}
