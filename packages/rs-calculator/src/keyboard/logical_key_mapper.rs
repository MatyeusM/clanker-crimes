//! Logical key mapper. Handles the main keyboard via `Key::Character` and
//! a few `Named` keys. The OG mapping, now constructing through the message
//! factory like everything else. Note: literal "=" is deliberately NOT bound
//! to Equals — only Enter/Return triggers it, per ancient request.
use crate::messages::message::Message;
use crate::messages::message_factory::{
  backspace_message, binary_op_message, clear_message, digit_message, dot_message, equals_message,
};
use clanker_calculator::OpToken;
use iced::keyboard::{self, key::Named};

/// Maps a logical key to a calculator message, or None to ignore it.
pub fn key_to_message(key: &keyboard::Key) -> Option<Message> {
  match key.as_ref() {
    keyboard::Key::Character(c) => match c.chars().next()? {
      d @ '0'..='9' => Some(digit_message(d)),
      '.' => Some(dot_message()),
      '+' => Some(binary_op_message(OpToken::Add)),
      '-' => Some(binary_op_message(OpToken::Subtract)),
      '*' => Some(binary_op_message(OpToken::Multiply)),
      '/' => Some(binary_op_message(OpToken::Divide)),
      '(' => Some(binary_op_message(OpToken::OpenParen)),
      ')' => Some(binary_op_message(OpToken::CloseParen)),
      _ => None,
    },
    keyboard::Key::Named(Named::Enter) => Some(equals_message()),
    keyboard::Key::Named(Named::Escape) => Some(clear_message()),
    keyboard::Key::Named(Named::Backspace) => Some(backspace_message()),
    _ => None,
  }
}
