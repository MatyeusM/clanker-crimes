//! Operator glyph service. Maps tokens to the pretty unicode shown on screen.
//!
//! The display string is never re-parsed, so there is no mismatch between
//! these glyphs (×, ÷, −) and what actually gets evaluated.
use clanker_calculator::OpToken;

/// The glyph shown on the display for an operator token.
pub fn op_glyph(op: OpToken) -> &'static str {
  match op {
    OpToken::Add => "+",
    OpToken::Subtract => "−",
    OpToken::Multiply => "×",
    OpToken::Divide => "÷",
    OpToken::Power => "^",
    OpToken::Root => "ʸ√",
    OpToken::OpenParen => "(",
    OpToken::CloseParen => ")",
    // These are never pushed into `expression` by this UI — they're
    // handled as immediate UnaryFn presses instead — but the match
    // must stay exhaustive.
    OpToken::Sqrt | OpToken::Negate | OpToken::Percent => "",
  }
}
