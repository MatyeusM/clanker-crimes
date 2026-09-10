//! Glyph provider trait. Abstracts glyph resolution behind an interface.
//!
//! See number_format_trait.rs for the bit about diamonds. Same energy.
use super::glyph_service::op_glyph;
use clanker_calculator::OpToken;

/// Provides display glyphs for operator tokens.
pub trait GlyphProvider {
  /// Returns the glyph for an operator token.
  fn glyph(&self, op: OpToken) -> &'static str;
}

/// The default glyph provider.
pub struct DefaultGlyphProvider;

/// Factory for glyph providers.
pub struct GlyphProviderFactory;

impl GlyphProvider for DefaultGlyphProvider {
  fn glyph(&self, op: OpToken) -> &'static str {
    op_glyph(op)
  }
}

impl GlyphProviderFactory {
  /// Creates the default glyph provider.
  pub fn create_default() -> DefaultGlyphProvider {
    DefaultGlyphProvider
  }
}
