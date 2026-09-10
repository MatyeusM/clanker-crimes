//! Produced-text fallback mapper. For platforms that report
//! `Key::Unidentified` but still fill in the produced text (e.g. text "5").
//! Last resort in the pipeline. Scrappy. Effective. Factory-built, like family.
use crate::messages::message::Message;
use crate::messages::message_factory::{binary_op_message, digit_message, dot_message};
use clanker_calculator::OpToken;

/// Maps produced key text to a message (single-char texts only).
pub fn map_text<S: AsRef<str>>(text: &Option<S>) -> Option<Message> {
  let t = text.as_ref()?;
  let t = t.as_ref();
  let mut chars = t.chars();
  if let (Some(c), None) = (chars.next(), chars.next()) {
    return match c {
      d @ '0'..='9' => Some(digit_message(d)),
      '.' | ',' => Some(dot_message()),
      '+' => Some(binary_op_message(OpToken::Add)),
      '-' => Some(binary_op_message(OpToken::Subtract)),
      '*' => Some(binary_op_message(OpToken::Multiply)),
      '/' => Some(binary_op_message(OpToken::Divide)),
      '(' => Some(binary_op_message(OpToken::OpenParen)),
      ')' => Some(binary_op_message(OpToken::CloseParen)),
      _ => None,
    };
  }
  None
}
