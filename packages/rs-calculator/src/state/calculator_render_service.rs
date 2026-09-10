//! Calculator render service. Turns state into the display string.
//!
//! Formerly `Calculator::render()`. Now a free function in its own file, as
//! nature intended. The display string is never re-parsed, so there is no
//! mismatch between the pretty unicode glyphs (×, ÷) and what gets evaluated.
use super::calculator::Calculator;
use crate::formatting::glyph_trait::{GlyphProvider, GlyphProviderFactory};
use crate::formatting::number_format_service::format_number;
use clanker_calculator::Token;

/// What to show on the display.
pub fn render_display(calc: &Calculator) -> String {
  if let Some(err) = &calc.error {
    return err.clone();
  }
  // Glyphs resolve through the provider so display rules stay centralized
  // (in the service, behind the trait, behind the factory — you know how).
  let glyphs = GlyphProviderFactory::create_default();
  let mut parts: Vec<String> = Vec::new();
  for tok in &calc.expression {
    match tok {
      Token::Number(n) => parts.push(format_number(*n)),
      Token::Operator(op) => parts.push(glyphs.glyph(*op).to_string()),
    }
  }
  if !calc.input.is_empty() {
    parts.push(calc.input.clone());
  } else if calc.expression.is_empty() {
    parts.push("0".to_string());
  }
  parts.join(" ")
}
