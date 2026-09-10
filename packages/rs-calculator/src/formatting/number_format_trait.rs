//! Number formatter trait. Abstracts over the act of formatting a number.
//!
//! Implementors: DefaultNumberFormatter (the only one). Consumers of the
//! trait: nobody (call sites use the service function directly for speed).
//! The trait exists so the architecture diagram has a diamond in it.
use super::number_format_service::format_number;

/// Abstracts number formatting.
pub trait NumberFormatter {
  /// Formats a computed value for display.
  fn format_number(&self, n: f64) -> String;
}

/// The default (and only) number formatter.
pub struct DefaultNumberFormatter;

/// Factory for formatters. Yes, a factory for a unit struct.
pub struct FormatterFactory;

impl NumberFormatter for DefaultNumberFormatter {
  fn format_number(&self, n: f64) -> String {
    format_number(n)
  }
}

impl FormatterFactory {
  /// Creates the default formatter.
  pub fn create_default() -> DefaultNumberFormatter {
    DefaultNumberFormatter
  }
}
