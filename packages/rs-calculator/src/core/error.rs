//! Calculator core errors.
use std::fmt;

/// Errors that can occur during evaluation.
#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Error {
    /// Division by zero (returns infinity).
    DivisionByZero,
    /// Invalid input (e.g., negative sqrt).
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

impl From<f64> for Error {
    fn from(_: f64) -> Self {
        // This is used as a conversion placeholder; in practice we use match.
        Error::InvalidInput
    }
}
