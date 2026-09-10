//! Calculator state and expression evaluation.
use crate::arithmetic::{add, divide, multiply, power, sqrt};
use crate::error::Error;
use crate::state;
use iced::{Color, Font};

/// A token in the calculator's token stream.
pub enum Token {
    Number(f64),
    Operator(state::OpToken),
}

impl Token {
    /// Returns `true` if this is a number token.
    pub fn is_number(&self) -> bool { matches!(self, Self::Number(_)) }
}

/// The internal state machine of the calculator.
#[derive(Debug, Clone, PartialEq)]
pub enum State<'a> {
    /// Just started or result was just displayed; ready to accept a new operand.
    ReadyToAcceptOperand,
    /// Waiting for an operator after a number.
    PendingOperator(Option<f64>),
    /// A unary operation (sqrt) is pending, waiting for its operand.
    PendingUnaryOperation,
    /// Result has just been displayed; next input starts fresh.
    ResultDisplayed(&'a str),
}

impl<'a> State<'a> {
    pub fn new() -> Self {
        State::ReadyToAcceptOperand
    }

    pub fn result_displayed(text: &'a str) -> Self {
        State::ResultDisplayed(text)
    }
}

/// The core evaluation engine using a shunting-yard algorithm.
pub struct Evaluator;

impl<'a> Evaluator {
    /// Evaluates the given token stream and returns (result, next_state).
    pub fn evaluate(tokens: &[state::Token]) -> Result<(f64, State<'a>), Error> {
        let mut output_queue = Vec::new();      // post-fix tokens (numbers)
        let mut operator_stack: Vec<(u8, state::OpToken)> = vec![];

        for i in 0..tokens.len() {
            match &tokens[i] {
                state::Token::Number(v) => output_queue.push(*v),
                state::Token::Operator(op) => {
                    // Unary operators have higher precedence.
                    let unary = matches!(op, state::OpToken::Sqrt | state::OpToken::Negate);

                    // Pop higher/equal precedence operators from stack.
                    while let Some(&(prec, top_op)) = operator_stack.last() {
                        if !(unary && prec < get_precedence(op)) ||
                           !matches!(&top_op, state::OpToken::OpenParen) {
                            output_queue.push(operator_stack.pop().unwrap().1);
                        } else {
                            break;
                        }
                    }

                    operator_stack.push((get_precedence(op), *op));
                }
            }
        }

        // Drain remaining operators.
        while let Some(&(prec, op)) = operator_stack.last() {
            output_queue.push(operator_stack.pop().unwrap().1);
        }

        // Evaluate the post-fix expression.
        Self::evaluate_postfix(&output_queue)
    }

    fn evaluate_postfix(tokens: &[state::OpToken]) -> Result<(f64, State<'a>), Error> {
        let mut stack = Vec::new();

        for op in tokens.iter().rev() {
            match op {
                state::OpToken::Add => {
                    if stack.len() < 2 { return Err(Error::InvalidInput); }
                    let b = stack.pop().unwrap();
                    let a = stack.pop().unwrap();
                    stack.push(add(a, b)?);
                }
                state::OpToken::Subtract => {
                    if stack.len() < 2 { return Err(Error::InvalidInput); }
                    let b = stack.pop().unwrap();
                    let a = stack.pop().unwrap();
                    stack.push(subtract(a, b)?);
                }
                state::OpToken::Multiply => {
                    if stack.len() < 2 { return Err(Error::InvalidInput); }
                    let b = stack.pop().unwrap();
                    let a = stack.pop().unwrap();
                    stack.push(multiply(a, b)?);
                }
                state::OpToken::Divide => {
                    if stack.len() < 2 { return Err(Error::InvalidInput); }
                    let b = stack.pop().unwrap();
                    let a = stack.pop().unwrap();
                    stack.push(divide(a, b)?)?; // propagate error from divide
                }
                state::OpToken::Power => {
                    if stack.len() < 2 { return Err(Error::InvalidInput); }
                    let exp = stack.pop().unwrap();
                    let base = stack.pop().unwrap();
                    stack.push(power(base, exp)?)?;
                }
                state::OpToken::Sqrt => {
                    if stack.is_empty() { return Err(Error::InvalidInput); }
                    let val = stack.pop().unwrap();
                    stack.push(sqrt(val)?)?;
                }
                state::OpToken::Negate => {
                    if stack.is_empty() { return Err(Error::InvalidInput); }
                    let val = stack.pop().unwrap();
                    stack.push(-val)?;
                }
                _ => {} // no-op for paren tokens in postfix form
            }
        }

        Ok((stack.into_iter().next().unwrap(), State::ResultDisplayed("")))
    }

    /// Returns the text representation of the current state.
    pub fn display_text(&self) -> String {
        match self {
            State::ReadyToAcceptOperand => String::new(),
            State::PendingOperator(_) => String::from("(..."),
            State::PendingUnaryOperation => String::from("sqrt(...)"),
            State::ResultDisplayed(s) => s.to_string(),
        }
    }
}

/// Returns the precedence of an operator. Higher = binds tighter.
fn get_precedence(op: &state::OpToken) -> u8 {
    match op {
        state::OpToken::Add | state::OpToken::Subtract => 2,
        state::OpToken::Multiply | state::OpToken::Divide => 3,
        state::OpToken::Power => 4,
        _ => 0,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::tokenizer::tokenize;

    #[test]
    fn test_simple_addition() {
        let tokens = tokenize("3 + 4").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 7.0).abs() < f64::EPSILON);
        assert!(matches!(state, State::ResultDisplayed(_)));
    }

    #[test]
    fn test_multiplication() {
        let tokens = tokenize("2 * 5").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 10.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_division_by_zero() {
        let tokens = tokenize("5 / 0").unwrap();
        let result = Evaluator::evaluate(&tokens);
        assert!(result.is_err());
    }

    #[test]
    fn test_chain_operations() {
        let tokens = tokenize("2 + 3 * 4").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        // 2 + (3*4) = 14
        assert!((result - 14.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_parentheses() {
        let tokens = tokenize("(2 + 3) * 4").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 20.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_power() {
        let tokens = tokenize("2 ^ 3").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 8.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_sqrt() {
        let tokens = tokenize("sqrt(9)").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 3.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_decimal() {
        let tokens = tokenize("1.5 + 2.5").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 4.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_negation() {
        let tokens = tokenize("-5 + 3").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - (-2.0)).abs() < f64::EPSILON);
    }

    #[test]
    fn test_scientific_notation() {
        let tokens = tokenize("1e3 + 2").unwrap();
        let (result, state) = Evaluator::evaluate(&tokens).unwrap();
        assert!((result - 1002.0).abs() < f64::EPSILON);
    }

    #[test]
    fn test_invalid_input() {
        let result = tokenize("a + b");
        assert!(result.is_err());
    }
}
