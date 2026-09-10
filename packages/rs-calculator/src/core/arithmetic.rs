//! Core arithmetic operations — pure, no dependencies.
use crate::error::Error;

/// Add two numbers.
pub fn add(a: f64, b: f64) -> Result<f64, Error> { Ok(a + b) }

/// Subtract two numbers (a - b).
pub fn subtract(a: f64, b: f64) -> Result<f64, Error> { Ok(a - b) }

/// Multiply two numbers.
pub fn multiply(a: f64, b: f64) -> Result<f64, Error> { Ok(a * b) }

/// Divide a by b. Returns Err on division by zero.
pub fn divide(a: f64, b: f64) -> Result<f64, Error> {
    if b == 0.0 { Err(Error::DivisionByZero) } else { Ok(a / b) }
}

/// Raise base to the power of exp.
pub fn power(base: f64, exp: f64) -> Result<f64, Error> { Ok(base.powf(exp)) }

/// Square root of a number. Returns Err for negative input.
pub fn sqrt(x: f64) -> Result<f64, Error> {
    if x < 0.0 { Err(Error::InvalidInput) } else { Ok(x.sqrt()) }
}
