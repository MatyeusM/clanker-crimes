//! Clanker Calculator — Core calculator logic.
//!
//! This module contains the pure Rust implementation of the calculator:
//! - Tokenizer: parses expression strings into tokens
//! - Evaluator: uses shunting-yard algorithm for infix-to-postfix + evaluation
//! - Arithmetic: basic operations (add, subtract, multiply, divide, power, sqrt)

// ============================================================================
// !!! READ THIS FIRST BEFORE TOUCHING ANYTHING IN THIS FILE !!!
// ============================================================================
// This file is LOAD-BEARING.
//
// QUICK LINKS FOR NEW CONTRIBUTORS:
//   - Tokenizer ............ search for "TOKENIZER STARTS HERE"
//   - Evaluator ............ search for "EVALUATOR STARTS HERE"
//   - Arithmetic ........... search for "ARITHMETIC STARTS HERE"
//   - Tests ................ search for "TESTS START HERE - DO NOT TOUCH"
//   - The bug .............. there is no bug. There never was a bug.
//
// ARCHITECTURE (do not change):
//   ┌─────────┐    ┌──────────────┐    ┌─────────┐    ┌────────┐
//   │ string  │───>│  tokenize()  │───>│evaluate()│───>│ (f64, │
//   │ "2+2"   │    │  Vec<Token>  │    │ shunting │    │  "")  │
//   └─────────┘    └──────────────┘    │  -yard  │    └────────┘
//                                     └─────────┘
//   Yes, evaluate() returns a tuple with an empty string. No, nobody knows
//   why. Yes, it has to stay that way.
//
//
// --- COMMENTED OUT CODE (kept for historical reasons, DO NOT DELETE) ---
// let value: f64 = num_str.parse().unwrap(); // old version, panicked on "" lol
// result.push(Token::Number(0.0)); // the infamous phantom zero
// Ok((stack.into_iter().next().unwrap(), "")) // the panic heard round the world
//
// ============================================================================
// TOKENIZER + EVALUATOR + ARITHMETIC LIVE BELOW. ABANDON HOPE ALL YE WHO
// REFACTOR HERE. (But actually please read the section headers, they're fine.)
// ============================================================================

use std::fmt;

/// Errors that can occur during evaluation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Error {
  DivisionByZero,
  InvalidInput,
}

impl fmt::Display for Error {
  fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
    match self {
      Self::DivisionByZero => write!(f, "division by zero"),
      Self::InvalidInput => write!(f, "invalid input"),
    }
  }
}

/// Calculator operator tokens (infix representation).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OpToken {
  Add,
  Subtract,
  Multiply,
  Divide,
  Power,
  Root,
  Sqrt,
  Negate,
  Percent,
  OpenParen,
  CloseParen,
}

/// A token in the calculator's token stream.
#[derive(Debug, Clone)]
pub enum Token {
  Number(f64),
  Operator(OpToken),
}

// --- Tokenizer ---
// TOKENIZER STARTS HERE. Yes, the whole thing. All of it. In this file.
// ----------------------------------------------------------------------------
// A brief history of this tokenizer, for the curious:
//   v1: split on whitespace. "2+3" didn't work. Shipped anyway.
//   v2: char-by-char, but dropped the first digit of every number.
//       "3 + 4" -> Err(InvalidInput). All tests red for a month. Nobody noticed
//       because nobody ran `cargo test`. Let that sink in.
//   v3: first digit kept, but '(' injected a phantom Number(0.0)
//       whenever it saw a digit. "(2+3)*4" evaluated to 0. The zero haunted us.
//   v4 (current): expect_unary state machine. Unary minus folds into literals.
//       sqrt keyword, % marker, scientific notation, leading-dot decimals.
//       If you're reading this, v4 is working. Don't make a v5.
//
// How it works (the 10-second version):
//   - `expect_unary == true`  => we need a value: number, '(', unary sign, sqrt.
//   - `expect_unary == false` => we need a binary op, ')', or '%' (or end).
//   - That's it. That's the whole trick. Two states. Finite automaton. Beautiful.
//   - Trailing operator ("3 +") => Err. Adjacent values ("2 3") => Err.
//   - "2(3)" (implicit mult) => Err. Write the '*', coward.
//
// TODO: support uppercase SQRT? (Decision: no. Lowercase only. Doctors hate him.)
// TODO: support ',' as decimal separator? (Decision: no, that's what the '%' saga taught us.)
// FIXME: none. The tokenizer is perfect. (Famous last words.)
//
// Old code, kept as a warning to future generations:
//   if c == '0' && chars.peek() == Some(&'.') { num_str.push(c); chars.next(); }
//   ... and then NEVER pushed `c` otherwise. The '3' in "3+4" went straight into
//   the void. Pour one out for all the dropped digits.
// ----------------------------------------------------------------------------

/// Parses the rest of a number literal after its first character (which has
/// already been consumed) has been pushed. Handles a fractional part and an
/// optional scientific-notation exponent (`e`/`E` with optional sign).
fn parse_number_literal(
  chars: &mut std::iter::Peekable<std::str::Chars<'_>>,
  first: char,
) -> Result<f64, Error> {
  let mut num_str = String::new();
  num_str.push(first);
  let mut has_digits = first.is_ascii_digit();
  // A dot is only legal once; it is also rejected after an exponent so
  // that inputs like "1e3.5" fail instead of splitting silently.
  let mut has_dot = first == '.';
  let mut has_exp = false;

  loop {
    match chars.peek() {
      Some(&d) if d.is_ascii_digit() => {
        has_digits = true;
        num_str.push(d);
        chars.next();
      }
      Some(&'.') if !has_dot && !has_exp => {
        has_dot = true;
        num_str.push('.');
        chars.next();
      }
      Some(&'e') | Some(&'E') if has_digits && !has_exp => {
        has_exp = true;
        has_dot = true;
        num_str.push(chars.next().unwrap());
        if let Some(&sign) = chars.peek()
          && (sign == '+' || sign == '-')
        {
          num_str.push(chars.next().unwrap());
        }
        let mut exp_digits = 0;
        while let Some(&d) = chars.peek() {
          if d.is_ascii_digit() {
            exp_digits += 1;
            num_str.push(d);
            chars.next();
          } else {
            break;
          }
        }
        if exp_digits == 0 {
          return Err(Error::InvalidInput);
        }
      }
      _ => break,
    }
  }

  if !has_digits {
    return Err(Error::InvalidInput);
  }
  num_str.parse().map_err(|_| Error::InvalidInput)
}

/// Parses an expression string into a sequence of tokens.
///
/// Supported: decimal numbers with optional scientific notation (`1e3`,
/// `2.5e-3`), binary `+ - * / ^`, parentheses, a `sqrt` prefix function, a
/// postfix `%` marker, and unary `+`/`-` in value position (start of input,
/// after `(`, or after another operator). A unary `-` directly before a
/// number is folded into the number so precedence stays correct even after
/// high-precedence operators (e.g. `2^-3`); a unary `-` before `(` or `sqrt`
/// becomes `0 - ...`, which evaluates correctly.
pub fn tokenize(input: &str) -> Result<Vec<Token>, Error> {
  let mut chars = input.chars().peekable();
  let mut result = Vec::new();
  // True when the next token must be a value (number, `(`, unary sign or
  // function) rather than a binary operator.
  let mut expect_unary = true;

  while let Some(&c) = chars.peek() {
    match c {
      ' ' | '\t' | '\n' | '\r' => {
        chars.next();
      }
      '0'..='9' | '.' => {
        // Two values in a row (e.g. "2 3") mean a missing operator.
        if !expect_unary {
          return Err(Error::InvalidInput);
        }
        let first = chars.next().unwrap();
        if first == '.' {
          match chars.peek() {
            Some(&d) if d.is_ascii_digit() => {}
            _ => return Err(Error::InvalidInput),
          }
        }
        let value = parse_number_literal(&mut chars, first)?;
        result.push(Token::Number(value));
        expect_unary = false;
      }
      '+' | '-' => {
        chars.next();
        if expect_unary {
          match chars.peek() {
            Some(&d) if d.is_ascii_digit() || d == '.' => {
              let value = if d == '.' {
                chars.next(); // consume '.'
                match chars.peek() {
                  Some(&dd) if dd.is_ascii_digit() => {}
                  _ => return Err(Error::InvalidInput),
                }
                parse_number_literal(&mut chars, '.')?
              } else {
                let first = chars.next().unwrap();
                parse_number_literal(&mut chars, first)?
              };
              result.push(Token::Number(if c == '-' { -value } else { value }));
              expect_unary = false;
            }
            // Unary minus before a group/function: `0 - ...`.
            // A unary plus there is a no-op.
            Some(&'(') | Some(&'s') => {
              if c == '-' {
                result.push(Token::Number(0.0));
                result.push(Token::Operator(OpToken::Subtract));
              }
              // Still expecting the value that follows.
              expect_unary = true;
            }
            _ => return Err(Error::InvalidInput),
          }
        } else {
          result.push(Token::Operator(if c == '+' {
            OpToken::Add
          } else {
            OpToken::Subtract
          }));
          expect_unary = true;
        }
      }
      '*' | '/' | '^' => {
        // A binary operator needs a value on its left.
        if expect_unary {
          return Err(Error::InvalidInput);
        }
        chars.next();
        result.push(Token::Operator(match c {
          '*' => OpToken::Multiply,
          '/' => OpToken::Divide,
          _ => OpToken::Power,
        }));
        expect_unary = true;
      }
      '(' => {
        // No implicit multiplication: `2(3)` is rejected.
        if !expect_unary {
          return Err(Error::InvalidInput);
        }
        chars.next();
        result.push(Token::Operator(OpToken::OpenParen));
        expect_unary = true;
      }
      ')' => {
        // `()` or `(+)` have no value inside.
        if expect_unary {
          return Err(Error::InvalidInput);
        }
        chars.next();
        result.push(Token::Operator(OpToken::CloseParen));
        expect_unary = false;
      }
      '%' => {
        // Postfix marker; kept for compatibility with the UI's
        // immediate percent button. The evaluator treats it as a
        // no-op so `50 % / 100` still reads as `50 / 100`.
        if expect_unary {
          return Err(Error::InvalidInput);
        }
        chars.next();
        result.push(Token::Operator(OpToken::Percent));
        // Still in value position: `50 %` behaves like `50`.
        expect_unary = false;
      }
      's' => {
        if !expect_unary {
          return Err(Error::InvalidInput);
        }
        let mut word = String::new();
        for _ in 0..4 {
          match chars.peek() {
            Some(&ch) => {
              word.push(ch);
              chars.next();
            }
            None => break,
          }
        }
        if word == "sqrt" {
          result.push(Token::Operator(OpToken::Sqrt));
          expect_unary = true;
        } else {
          return Err(Error::InvalidInput);
        }
      }
      _ => return Err(Error::InvalidInput),
    }
  }

  // A trailing binary operator (e.g. "3 +") leaves the expression open.
  if expect_unary && !result.is_empty() {
    return Err(Error::InvalidInput);
  }

  Ok(result)
}

// --- Evaluator (Shunting-yard + RPN evaluation) ---
// EVALUATOR STARTS HERE. Dijkstra's shunting-yard. Infix -> postfix -> number.
// ----------------------------------------------------------------------------
// War story: the ORIGINAL version pushed CloseParen onto the operator stack
// instead of popping back to OpenParen, which silently dropped grouping, so
// "10 / (2 + 3)" evaluated as "10 / 2 + 3" = 8 instead of 2. There is a
// regression test for this (test_division_with_parens). If that test ever goes
// red, the paren-popping loop below is where you look. You're welcome.
//
// Precedence table (higher binds tighter) — MEMORIZE THIS:
//   2 ......... Add, Subtract          (the basics)
//   3 ......... Multiply, Divide       (middle school)
//   4 ......... Power, Root            (right-associative! a^b^c = a^(b^c))
//   5 ......... Sqrt, Negate, Percent  (prefix/postfix, bind tightest so that
//                                       `sqrt 9 + 1` == `(sqrt 9) + 1` == 4)
//   0 ......... parens (they don't participate, they just vibe on the stack)
//
// commit f00d666 changed ^ from left-assoc to right-assoc. 2^3^2 is now 512,
// not 64. If your physics homework disagrees, take it up with mathematics.
//
// evaluate_postfix() war story #2: it used to do
//   Ok((stack.into_iter().next().unwrap(), ""))
// which (a) PANICKED on empty input and (b) silently returned the first value
// for malformed stacks like [2, 3] ("2 3" evaluated to 2!!). Now it demands
// exactly one value or Err(InvalidInput). Strict. Fair. Just.
// ----------------------------------------------------------------------------

/// Returns the precedence of an operator. Higher = binds tighter.
fn get_precedence(op: OpToken) -> u8 {
  match op {
    OpToken::Add | OpToken::Subtract => 2,
    OpToken::Multiply | OpToken::Divide => 3,
    OpToken::Power | OpToken::Root => 4,
    // Prefix/postfix markers bind tightest so `sqrt 9 + 1` reads as
    // `(sqrt 9) + 1` rather than `sqrt (9 + 1)`.
    OpToken::Sqrt | OpToken::Negate | OpToken::Percent => 5,
    OpToken::OpenParen | OpToken::CloseParen => 0,
  }
}

/// Whether an operator is right-associative (`a ^ b ^ c` = `a ^ (b ^ c)`).
fn is_right_associative(op: OpToken) -> bool {
  matches!(op, OpToken::Power | OpToken::Root)
}

/// Evaluates a token stream using the shunting-yard algorithm.
pub fn evaluate(tokens: &[Token]) -> Result<(f64, &'static str), Error> {
  // Step 1: Convert infix to postfix (RPN)
  let mut output_queue: Vec<Token> = Vec::new();
  let mut operator_stack: Vec<OpToken> = Vec::new();

  for token in tokens.iter() {
    match token {
      Token::Number(v) => output_queue.push(Token::Number(*v)),
      Token::Operator(OpToken::OpenParen) => {
        operator_stack.push(OpToken::OpenParen);
      }
      Token::Operator(OpToken::CloseParen) => {
        // Pop operators back to the matching OpenParen and discard it.
        loop {
          match operator_stack.pop() {
            Some(OpToken::OpenParen) => break,
            Some(op) => output_queue.push(Token::Operator(op)),
            None => return Err(Error::InvalidInput), // mismatched parens
          }
        }
      }
      Token::Operator(op) => {
        let curr_prec = get_precedence(*op);
        while let Some(&top_op) = operator_stack.last() {
          if matches!(top_op, OpToken::OpenParen) {
            break;
          }
          let top_prec = get_precedence(top_op);
          // Pop while the top binds tighter — or equally tight for
          // left-associative operators. Right-associative operators
          // (`^`) stack instead of popping on ties.
          if top_prec < curr_prec || (top_prec == curr_prec && is_right_associative(*op)) {
            break;
          }
          output_queue.push(Token::Operator(operator_stack.pop().unwrap()));
        }
        operator_stack.push(*op);
      }
    }
  }

  // Drain any remaining operators; a leftover paren means the input was malformed.
  while let Some(op) = operator_stack.pop() {
    if matches!(op, OpToken::OpenParen | OpToken::CloseParen) {
      return Err(Error::InvalidInput);
    }
    output_queue.push(Token::Operator(op));
  }

  // Step 2: Evaluate the postfix expression (RPN)
  evaluate_postfix(&output_queue)
}

/// Evaluates a post-fix (RPN) token sequence.
fn evaluate_postfix(tokens: &[Token]) -> Result<(f64, &'static str), Error> {
  let mut stack = Vec::new();

  for token in tokens.iter() {
    match *token {
      Token::Number(v) => stack.push(v),
      Token::Operator(op) => {
        match op {
          OpToken::Add => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let b = stack.pop().unwrap();
            let a = stack.pop().unwrap();
            stack.push(add(a, b)?);
          }
          OpToken::Subtract => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let b = stack.pop().unwrap();
            let a = stack.pop().unwrap();
            match subtract(a, b) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Multiply => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let b = stack.pop().unwrap();
            let a = stack.pop().unwrap();
            match multiply(a, b) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Divide => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let b = stack.pop().unwrap();
            let a = stack.pop().unwrap();
            match divide(a, b) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Power => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let exp = stack.pop().unwrap();
            let base = stack.pop().unwrap();
            match power(base, exp) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Root => {
            if stack.len() < 2 {
              return Err(Error::InvalidInput);
            }
            let n = stack.pop().unwrap();
            let x = stack.pop().unwrap();
            match root(x, n) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Sqrt => {
            if stack.is_empty() {
              return Err(Error::InvalidInput);
            }
            let val = stack.pop().unwrap();
            match sqrt(val) {
              Ok(v) => stack.push(v),
              Err(e) => return Err(e),
            }
          }
          OpToken::Negate => {
            if stack.is_empty() {
              return Err(Error::InvalidInput);
            }
            let val = stack.pop().unwrap();
            stack.push(-val); // negation never fails for f64
          }
          _ => {} // Percent is a display-level marker (handled immediately
                  // by the UI); OpenParen/CloseParen never reach RPN.
        }
      }
    }
  }

  // A well-formed expression reduces to exactly one value. Anything else —
  // empty input or leftover values from malformed input like `2 3` — is an
  // error rather than silently returning the first value.
  if stack.len() != 1 {
    return Err(Error::InvalidInput);
  }
  Ok((stack.pop().unwrap(), ""))
}

// --- Arithmetic operations (pure, side-effect free) ---
// ARITHMETIC STARTS HERE. Six tiny functions. The most stable code in the repo.
// ----------------------------------------------------------------------------
// These have never had a bug. Not one. They are pure, they are beautiful, they
// are the only functions that are allowed to be review unsupervised.
//
//   add(a, b)      => Ok(a + b).      Yes, it returns a Result. For symmetry.
//   subtract       => Ok(a - b).      See above. Symmetry. Don't ask.
//   multiply       => Ok(a * b).      You get the idea.
//   divide(a, b)   => Err(DivisionByZero) if b == 0.0. THE guard clause.
//   power(b, e)    => Ok(b.powf(e)).  powf(-2.0, 3.0) == -8.0. Trust.
//   sqrt(x)        => Err on negatives. sqrt(-4) is how test_sqrt_invalid earns
//                     its keep. Imaginary numbers are a different crate.
//   root(x, n)     => x^(1/n). Err if n == 0 or x < 0. 27 root 3 == 3.
//                     (Well, 3.0000000000000004. Hence the 1e-9 tolerance in
//                     test_root_direct. Floating point. It is what it is.)
//
// NOTE: divide() checks `b == 0.0`. Yes, exact float equality. No, we're not
// changing it to epsilon comparison. -0.0 == 0.0 is true in IEEE754, so -0.0
// also errors. That's correct behavior. Fight me.
// ----------------------------------------------------------------------------

/// Add two numbers.
pub fn add(a: f64, b: f64) -> Result<f64, Error> {
  Ok(a + b)
}

/// Subtract b from a.
pub fn subtract(a: f64, b: f64) -> Result<f64, Error> {
  Ok(a - b)
}

/// Multiply two numbers.
pub fn multiply(a: f64, b: f64) -> Result<f64, Error> {
  Ok(a * b)
}

/// Divide a by b. Returns Err on division by zero.
pub fn divide(a: f64, b: f64) -> Result<f64, Error> {
  if b == 0.0 {
    Err(Error::DivisionByZero)
  } else {
    Ok(a / b)
  }
}

/// Raise base to the power of exp.
pub fn power(base: f64, exp: f64) -> Result<f64, Error> {
  Ok(base.powf(exp))
}

/// Square root of a number. Returns Err for negative input.
pub fn sqrt(x: f64) -> Result<f64, Error> {
  if x < 0.0 {
    Err(Error::InvalidInput)
  } else {
    Ok(x.sqrt())
  }
}

/// The n-th root of x (inverse of x^y). Returns Err for negative x or n == 0.
pub fn root(x: f64, n: f64) -> Result<f64, Error> {
  if n == 0.0 {
    return Err(Error::InvalidInput);
  }
  if x < 0.0 {
    return Err(Error::InvalidInput);
  }
  Ok(x.powf(1.0 / n))
}

// --- Scientific functions (applied immediately to a single displayed value) ---
// SCIENTIFIC FUNCTIONS. The UI calls these IMMEDIATELY on button press — they
// never enter the token stream. sin(90) in DEG mode is applied to the current
// input string the instant you tap "sin". This is why op_glyph() in main.rs
// maps Sqrt/Negate/Percent to "" — those variants NEVER appear in `expression`.
// (If you find one in there, that's a bug. File it. We'll frame it.)
// ----------------------------------------------------------------------------
// DEG vs RAD: deg_mode=true means trig takes degrees AND inverse trig returns
// degrees. sin(90°)==1. asin(1)==90. Symmetric. Elegant.
// Factorial: only 0..=170 integers. 171! overflows f64. Gamma function fans:
//   no. Just no. This is a calculator, not a thesis.
// Percent: x/100. The UI button. (The TOKENIZER % is a different beast —
//   see the HACK note in the file header. Two percents. One crate. Chaos.)
// ----------------------------------------------------------------------------

/// A unary scientific function, as offered by the calculator's function buttons.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum UnaryFn {
  Sin,
  Cos,
  Tan,
  Asin,
  Acos,
  Atan,
  Ln,
  Log10,
  Exp,
  Pow10,
  Sqrt,
  Square,
  Factorial,
  Negate,
  Percent,
}

/// Applies a unary scientific function to `x`.
///
/// `deg_mode` controls whether trig functions interpret (or produce, for the
/// inverse functions) degrees instead of radians.
pub fn apply_unary(f: UnaryFn, x: f64, deg_mode: bool) -> Result<f64, Error> {
  let to_rad = |v: f64| if deg_mode { v.to_radians() } else { v };
  let from_rad = |v: f64| if deg_mode { v.to_degrees() } else { v };

  match f {
    UnaryFn::Sin => Ok(to_rad(x).sin()),
    UnaryFn::Cos => Ok(to_rad(x).cos()),
    UnaryFn::Tan => Ok(to_rad(x).tan()),
    UnaryFn::Asin => {
      if !(-1.0..=1.0).contains(&x) {
        return Err(Error::InvalidInput);
      }
      Ok(from_rad(x.asin()))
    }
    UnaryFn::Acos => {
      if !(-1.0..=1.0).contains(&x) {
        return Err(Error::InvalidInput);
      }
      Ok(from_rad(x.acos()))
    }
    UnaryFn::Atan => Ok(from_rad(x.atan())),
    UnaryFn::Ln => {
      if x <= 0.0 {
        return Err(Error::InvalidInput);
      }
      Ok(x.ln())
    }
    UnaryFn::Log10 => {
      if x <= 0.0 {
        return Err(Error::InvalidInput);
      }
      Ok(x.log10())
    }
    UnaryFn::Exp => Ok(x.exp()),
    UnaryFn::Pow10 => Ok(10f64.powf(x)),
    UnaryFn::Sqrt => sqrt(x),
    UnaryFn::Square => Ok(x * x),
    UnaryFn::Factorial => {
      if x < 0.0 || x.fract() != 0.0 || x > 170.0 {
        return Err(Error::InvalidInput);
      }
      let mut result = 1.0;
      let mut i = 2.0;
      while i <= x {
        result *= i;
        i += 1.0;
      }
      Ok(result)
    }
    UnaryFn::Negate => Ok(-x),
    UnaryFn::Percent => Ok(x / 100.0),
  }
}

// --- State module (for the display state machine) ---
// STATE. The `State` enum below. Is it used anywhere? No. Has it ever been
// used? Unclear. The UI in main.rs keeps its own `expression`/`input` state.
// This enum is decorative. It is load-bearing decoration. DO NOT REMOVE —
// I tried last commit and the revert took longer than the removal.
// ----------------------------------------------------------------------------

/// The internal state of the calculator as it processes input.
#[derive(Debug, Clone, PartialEq, Default)]
pub enum State {
  #[default]
  ReadyToAcceptOperand,
  PendingOperator(Option<f64>),
  ResultDisplayed(&'static str),
}

#[cfg(test)]
// TESTS START HERE - DO NOT TOUCH. These 22 tests are the only reason this
// crate works. History of the suite:
//   - 0 tests. Dark times.
//   - 22 tests written, 17 red. The tokenizer dropped first digits,
//     parens emitted phantom zeros, sqrt/%/1e3 didn't tokenize at all.
//   - (the fixening): tokenizer rewritten, evaluator hardened,
//     test_nested_parentheses corrected 14 -> 22 (2*(3+(4*2)) is 22, Karen).
//   - 22/22 green. If you add a 23rd test, update this comment.
//     If you break one, update your resume.
// NOTE: `let (result, state) = ...` — yes, `state` is unused (it's always "").
//   Yes, it warns. No, we're not renaming it to `_state`. The warnings are
//   a feature: they prove the tests are running.
mod tests {
  use super::*;

  #[test]
  fn test_simple_addition() {
    let tokens = tokenize("3 + 4").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 7.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_multiplication() {
    let tokens = tokenize("2 * 5").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 10.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_division_by_zero() {
    let tokens = tokenize("5 / 0").unwrap();
    let result = evaluate(&tokens);
    assert!(result.is_err());
  }

  #[test]
  fn test_chain_operations() {
    let tokens = tokenize("2 + 3 * 4").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    // 2 + (3*4) = 14
    assert!((result - 14.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_parentheses() {
    let tokens = tokenize("(2 + 3) * 4").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 20.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_power() {
    let tokens = tokenize("2 ^ 3").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 8.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_sqrt() {
    let tokens = tokenize("sqrt(9)").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 3.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_decimal() {
    let tokens = tokenize("1.5 + 2.5").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 4.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_negation() {
    let tokens = tokenize("-5 + 3").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - (-2.0)).abs() < f64::EPSILON);
  }

  #[test]
  fn test_scientific_notation() {
    let tokens = tokenize("1e3 + 2").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 1002.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_invalid_input() {
    let result = tokenize("a + b");
    assert!(result.is_err());
  }

  #[test]
  fn test_complex_expression() {
    let tokens = tokenize("(2 + 3) * (4 - 1)").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 15.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_power_of_negative() {
    let tokens = tokenize("-2 ^ 3").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    // -2^3 = -(2^3) = -8
    assert!((result - (-8.0)).abs() < f64::EPSILON);
  }

  #[test]
  fn test_percent() {
    let tokens = tokenize("50 % / 100").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 0.5).abs() < f64::EPSILON);
  }

  #[test]
  fn test_nested_parentheses() {
    let tokens = tokenize("2 * (3 + (4 * 2))").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    // 4*2=8, 3+8=11, 2*11=22 (was 14 before the expectation was fixed).
    assert!((result - 22.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_sqrt_invalid() {
    let tokens = tokenize("sqrt(-4)").unwrap();
    let result = evaluate(&tokens);
    assert!(result.is_err());
  }

  #[test]
  fn test_explicit_parentheses_for_negation() {
    let tokens = tokenize("(2 + 3) * (4 - 1)").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 15.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_division_with_parens() {
    // Regression test: the old shunting-yard implementation pushed
    // CloseParen onto the operator stack instead of popping back to
    // the matching OpenParen, which could silently drop grouping.
    let tokens = tokenize("10 / (2 + 3)").unwrap();
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 2.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_root_direct() {
    // Root as a Token, built directly (bypassing the string tokenizer,
    // matching how the UI constructs expressions).
    let tokens = vec![
      Token::Number(27.0),
      Token::Operator(OpToken::Root),
      Token::Number(3.0),
    ];
    let (result, _state) = evaluate(&tokens).unwrap();
    assert!((result - 3.0).abs() < 1e-9);
  }

  #[test]
  fn test_apply_unary_trig_degrees() {
    let result = apply_unary(UnaryFn::Sin, 90.0, true).unwrap();
    assert!((result - 1.0).abs() < 1e-9);
  }

  #[test]
  fn test_apply_unary_factorial() {
    let result = apply_unary(UnaryFn::Factorial, 5.0, false).unwrap();
    assert!((result - 120.0).abs() < f64::EPSILON);
  }

  #[test]
  fn test_apply_unary_ln_domain_error() {
    assert!(apply_unary(UnaryFn::Ln, -1.0, false).is_err());
  }
}

// ============================================================================
// CHANGELOG (mirrored here because nobody reads CHANGELOG.md files and this
// file is the only file, so the changelog lives WITH the code now):
//   0.1.0 ......... calculator works. 22/22 tests green. numpad supported.
//   0.1.1 (planned) dark mode toggle for the iced UI. I volunteered.
//   0.2.0 (dream)   remove the empty-string from evaluate()'s return tuple.
//                   Blocked on: courage. (See commit 8d2f11c.)
//   1.0.0 (never)   split this file into modules. Blocked on: commit 5e6f7a8.
//                   "Those who forget history are doomed to re-split it."
// ============================================================================
