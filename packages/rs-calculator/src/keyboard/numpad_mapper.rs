//! Numpad mapper. Handles physical numpad key codes (`Code::Numpad*`).
//!
//! Layout-independent, works with NumLock on or off. Messages are built via
//! the message factory so construction stays centralized. The reason numpad
//! input works at all is this file. Show some respect.
use crate::messages::message::Message;
use crate::messages::message_factory::{
  backspace_message, binary_op_message, digit_message, dot_message, equals_message,
};
use clanker_calculator::OpToken;
use iced::keyboard::key::Code;

/// Maps a physical key code to a message if it is a numpad key.
pub fn map_numpad_code(code: &Code) -> Option<Message> {
  match code {
    Code::Numpad0 => Some(digit_message('0')),
    Code::Numpad1 => Some(digit_message('1')),
    Code::Numpad2 => Some(digit_message('2')),
    Code::Numpad3 => Some(digit_message('3')),
    Code::Numpad4 => Some(digit_message('4')),
    Code::Numpad5 => Some(digit_message('5')),
    Code::Numpad6 => Some(digit_message('6')),
    Code::Numpad7 => Some(digit_message('7')),
    Code::Numpad8 => Some(digit_message('8')),
    Code::Numpad9 => Some(digit_message('9')),
    // Decimal separator (`,` on some locales) maps to Dot.
    Code::NumpadDecimal | Code::NumpadComma => Some(dot_message()),
    Code::NumpadAdd => Some(binary_op_message(OpToken::Add)),
    Code::NumpadSubtract => Some(binary_op_message(OpToken::Subtract)),
    Code::NumpadMultiply | Code::NumpadStar => Some(binary_op_message(OpToken::Multiply)),
    Code::NumpadDivide => Some(binary_op_message(OpToken::Divide)),
    Code::NumpadEnter => Some(equals_message()),
    Code::NumpadBackspace => Some(backspace_message()),
    Code::NumpadParenLeft => Some(binary_op_message(OpToken::OpenParen)),
    Code::NumpadParenRight => Some(binary_op_message(OpToken::CloseParen)),
    // NumpadEqual ("=") is deliberately NOT bound to Equals,
    // matching the main-keyboard "=" behavior.
    _ => None,
  }
}
