//! Expression parser and evaluator with proper operator precedence.
//! Uses the shunting-yard algorithm to convert infix expressions to RPN, then evaluates.

use crate::core::{error, state};

/// Operator type with precedence level encoded as a u8.
#[derive(Debug, Clone, Copy)]
struct OpType(u8);

impl OpType {
    fn new(prec: u8) -> Self { Self(prec) }
}

/// Parse an expression string into a token stream.
pub fn tokenize(input: &str) -> Result<Vec<state::Token>, error::Error> {
    let mut tokens = Vec::new();
    let chars: Vec<char> = input.chars().collect();
    let mut i = 0;

    while i < chars.len() {
        match chars[i] {
            '+' => {
                if is_unary_plus(&chars, i) {
                    // +5 -> Number(0) Add(Number(5)) handled by prepend_unary_ops
                } else {
                    tokens.push(state::Token::Operator(state::OpToken::Add));
                }
            }
            '-' => {
                if is_unary_minus(&chars, i) {
                    // Unary minus handled by prepend_unary_ops
                } else {
                    tokens.push(state::Token::Operator(state::OpToken::Subtract));
                }
            }
            '*' => tokens.push(state::Token::Operator(state::OpToken::Multiply)),
            '/' => tokens.push(state::Token::Operator(state::OpToken::Divide)),
            '√' if i + 1 < chars.len() && (chars[i + 1].is_ascii_digit() || is_unary_sqrt(&chars, i)) => {
                let num_start = find_number_start(&chars, i);
                let num_str: String = chars[num_start..find_number_end(&chars, num_start)].iter().collect();
                tokens.push(state::Token::Number(num_str.parse::<f64>().unwrap_or(0.0)));
            }
            'p' | 'P' if i + 1 < chars.len() && chars[i + 1] == '(' => {
                tokens.push(state::Token::Operator(state::OpToken::Power));
                i += 2; // skip "pow"
            }
            c if c.is_ascii_digit() || c == '.' => {
                let num_str: String = chars[i..find_number_end(&chars, i)].iter().collect();
                tokens.push(state::Token::Number(num_str.parse::<f64>().unwrap_or(0.0)));
            }
            _ if chars[i].is_whitespace() => {}
            c => return Err(error::Error::InvalidInput),
        }
        i += 1;
    }

    // Prepend unary operators (Number + operator pattern) for negative numbers, etc.
    prepend_unary_ops(&mut tokens);

    Ok(tokens)
}

fn is_unary_minus(chars: &[char], pos: usize) -> bool {
    if pos == 0 { return true; }
    let prev = chars[pos - 1];
    matches!(prev, '+' | '-' | '*' | '/' | '^' | '(' | ')')
}

fn is_unary_plus(chars: &[char], pos: usize) -> bool {
    if pos == 0 { return true; }
    let prev = chars[pos - 1];
    matches!(prev, '+' | '-' | '*' | '/' | '^' | '(' | ')')
}

fn is_unary_sqrt(chars: &[char], pos: usize) -> bool {
    if pos == 0 { return true; }
    let prev = chars[pos - 1];
    matches!(prev, '+' | '-' | '*' | '/' | '^' | '(' | ')')
}

fn find_number_start(chars: &[char], start: usize) -> usize {
    let mut i = start;
    if i < chars.len() && (chars[i] == 'p' || chars[i] == 'P') {
        while i + 1 < chars.len() && matches!(chars[i + 1], '(' | ')') { i += 2; }
        if i > 0 && matches!(chars[i - 1], '+' | '-') { i -= 1; }
    } else if i < chars.len() && (chars[i] == '-' || chars[i] == '+') {
        i += 1;
    }
    i
}

fn find_number_end(chars: &[char], start: usize) -> usize {
    let mut i = start;
    while i < chars.len() {
        match chars[i] {
            c if c.is_ascii_digit() || c == '.' => {}
            ')' | ',' => break,
            _ => return i,
        }
        i += 1;
    }
    i
}

fn prepend_unary_ops(tokens: &mut Vec<state::Token>) {
    let mut result = Vec::new();
    for (i, tok) in tokens.iter().enumerate() {
        match *tok {
            state::Token::Number(v) => {
                let needs_plus = i == 0;
                let needs_minus = !needs_plus && is_unary_minus_preceding(tokens, i);

                if needs_plus {
                    result.push(state::Token::Number(0.0));
                    result.push(state::Token::Operator(state::OpToken::Add));
                }
                if needs_minus {
                    result.push(state::Token::Number(0.0));
                    result.push(state::Token::Operator(state::OpToken::Subtract));
                }
                result.push(tok.clone());
            }
            state::Token::Operator(op) => match op {
                state::OpToken::Add | state::OpToken::Subtract
                | state::OpToken::Multiply | state::OpToken::Divide
                | state::OpToken::Power => {}
                state::OpToken::Negate => {}, // already a binary subtract
                state::OpToken::Sqrt => {
                    result.push(state::Token::Number(0.0));
                    result.push(state::Token::Operator(state::OpToken::Add));
                }
            },
        }
    }
    *tokens = result;
}

fn is_unary_minus_preceding(tokens: &[state::Token], pos: usize) -> bool {
    for i in (0..pos).rev() {
        match &tokens[i] {
            state::Token::Number(_) => return false,
            state::Token::Operator(op) => {
                matches!(op, state::OpToken::Add | state::OpToken::Subtract
                    | state::OpToken::Multiply | state::OpToken::Divide
                    | state::OpToken::Power | state::OpToken::Sqrt)
            }
        }
    }
    true
}

/// Evaluate an expression from a token stream using shunting-yard algorithm.
pub fn evaluate(tokens: &[state::Token]) -> Result<f64, error::Error> {
    if tokens.is_empty() {
        return Err(error::Error::InvalidInput);
    }

    let mut rpn = Vec::<&state::Token>::new();
    let mut op_stack: Vec<(u8, OpType)> = Vec::new(); // (precedence, type)

    for tok in tokens {
        match *tok {
            state::Token::Number(v) => {
                rpn.push(tok);
            }
            state::Token::Operator(op) => {
                let op_prec = get_precedence(op);

                // Pop operators with higher or equal precedence (except power is right-associative)
                while let Some(&(top_prec, top_type)) = op_stack.last() {
                    if matches!(top_type.0, OpType(2)) && op_prec == 4 {
                        // Power is right-associative: don't pop on equal precedence
                        break;
                    }
                    if top_prec >= op_prec {
                        let result = eval_op(&mut rpn, &op_stack[op_stack.len() - 1].1);
                        match result {
                            Ok(v) => rpn.push(&state::Token::Number(v)),
                            Err(e) => return Err(e),
                        }
                        op_stack.pop(); // remove the top operator
                    } else {
                        break;
                    }
                }

                op_stack.push((op_prec, OpType::new(op_prec)));
            }
        }
    }

    // Pop remaining operators and evaluate
    while let Some(&(prec, typ)) = op_stack.pop() {
        let result = eval_op(&mut rpn, &typ);
        if result.is_err() {
            // On error, push the operator back and continue
            op_stack.push((prec, typ));
        } else {
            rpn.push(&state::Token::Number(result.unwrap()));
        }
    }

    if let Some(TokenResult) = rpn.pop() {
        match TokenResult {
            state::Token::Number(v) => Ok(*v),
            _ => Err(error::Error::InvalidInput),
        }
    } else {
        Err(error::Error::InvalidInput)
    }
}

fn get_precedence(op: &state::OpToken) -> u8 {
    match op {
        state::OpToken::Add | state::OpToken::Subtract => 2,
        state::OpToken::Multiply | state::OpToken::Divide => 3,
        state::OpToken::Power => 4,
        _ => 0,
    }
}

fn eval_op(rpn: &mut Vec<&state::Token>, op_type: &OpType) -> Result<f64, error::Error> {
    match *op_type.0 {
        OpType(2) if matches!(op_type.1, state::OpToken::Add) => {} // Add
        OpType(2) if matches!(op_type.1, state::OpToken::Subtract) => {} // Subtract
        OpType(3) if matches!(op_type.1, state::OpToken::Multiply) => {} // Multiply
        OpType(3) if matches!(op_type.1, state::OpToken::Divide) => {}   // Divide  
        OpType(4) if matches!(op_type.1, state::OpToken::Power) => {}    // Power
        _ => return Err(error::Error::InvalidInput),
    }

    let arg2 = rpn.pop();
    let arg1 = rpn.pop();

    match (arg1, arg2) {
        (Some(state::Token::Number(a)), Some(state::Token::Number(b))) => {
            Ok(match op_type.0 {
                OpType(2) if matches!(op_type.1, state::OpToken::Add) => arithmetic::add(*a, *b)?,
                OpType(2) if matches!(op_type.1, state::OpToken::Subtract) => arithmetic::subtract(*a, *b)?,
                OpType(3) if matches!(op_type.1, state::OpToken::Multiply) => arithmetic::multiply(*a, *b)?,
                OpType(3) if matches!(op_type.1, state::OpToken::Divide) => arithmetic::divide(*a, *b)?,
                OpType(4) if matches!(op_type.1, state::OpToken::Power) => arithmetic::power(*a, *b)?,
                _ => return Err(error::Error::InvalidInput),
            })
        }
        (Some(state::Token::Number(a)), None) if matches!(op_type.0, OpType(2)) && matches!(op_type.1, state::OpToken::Subtract) => {
            // Unary minus: 0 - a
            Ok(arithmetic::subtract(0.0, *a)?)
        }
        _ => Err(error::Error::InvalidInput),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn tokenize_simple() {
        let tokens = tokenize("3").unwrap();
        assert_eq!(tokens.len(), 1);
        if let state::Token::Number(v) = &tokens[0] { 
            assert!(*v == 3.0); 
        } else { panic!("Expected Number token"); }
    }

    #[test]
    fn evaluate_addition() {
        let tokens = tokenize("3 + 4").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 7.0).abs() < f64::EPSILON);
    }

    #[test]
    fn evaluate_subtraction() {
        let tokens = tokenize("10 - 3").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 7.0).abs() < f64::EPSILON);
    }

    #[test]
    fn evaluate_multiplication_precedence() {
        let tokens = tokenize("2 + 3 * 4").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 14.0).abs() < f64::EPSILON); // 2 + (3*4) = 14, NOT (2+3)*4=20
    }

    #[test]
    fn evaluate_power_precedence() {
        let tokens = tokenize("2 ^ 3 + 1").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 9.0).abs() < f64::EPSILON); // (2^3) + 1 = 8+1=9, NOT 2^(3+1)=16
    }

    #[test]
    fn evaluate_nested_power() {
        let tokens = tokenize("2 ^ 3 ^ 2").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 512.0).abs() < f64::EPSILON); // Right associative: 2^(3^2) = 2^9 = 512
    }

    #[test]
    fn evaluate_negative_number() {
        let tokens = tokenize("-3 + 7").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 4.0).abs() < f64::EPSILON); // -3 + 7 = 4
    }

    #[test]
    fn evaluate_division_by_zero() {
        let tokens = tokenize("1 / 0").unwrap();
        assert_eq!(evaluate(&tokens), Err(error::Error::DivisionByZero));
    }

    #[test]
    fn evaluate_decimal() {
        let tokens = tokenize("3.14 + 2.86").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 6.0).abs() < f64::EPSILON);
    }

    #[test]
    fn evaluate_complex_parentheses() {
        let tokens = tokenize("(2 + 3) * (4 - 1)").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 15.0).abs() < f64::EPSILON); // 5 * 3 = 15
    }

    #[test]
    fn evaluate_chained_operations() {
        let tokens = tokenize("2 + 3 * 4 + 5").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 17.0).abs() < f64::EPSILON); // 2 + (3*4) + 5 = 2+12+5=17
    }

    #[test]
    fn evaluate_power_of_negative_base_integer_exp() {
        let tokens = tokenize("-2 ^ 3").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - (-8.0)).abs() < f64::EPSILON); // -8 (unary minus applies after power)
    }

    #[test]
    fn evaluate_complex_with_powers_and_mult() {
        let tokens = tokenize("2 ^ 3 * 4 + 1").unwrap();
        let result = evaluate(&tokens).unwrap();
        assert!((result - 33.0).abs() < f64::EPSILON); // (2^3)*4 + 1 = 8*4+1=33
    }
}
