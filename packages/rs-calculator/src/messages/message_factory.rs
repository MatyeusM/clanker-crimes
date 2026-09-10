//! Message factory. Constructs Messages so nobody has to write the enum
//! variants directly. Ten tiny constructors. Enterprise-grade indirection.
//!
//! Used by: almost nothing (keyboard mappers construct variants inline for
//! speed, UI rows use variants inline for clarity). This factory exists for
//! architectural completeness. It is complete.
use super::message::Message;
use clanker_calculator::{OpToken, UnaryFn};

/// Creates a digit message for the given character.
pub fn digit_message(d: char) -> Message {
  Message::Digit(d)
}

/// Creates a dot (decimal separator) message.
pub fn dot_message() -> Message {
  Message::Dot
}

/// Creates a binary-operator message.
pub fn binary_op_message(op: OpToken) -> Message {
  Message::BinaryOp(op)
}

/// Creates a unary-function message.
pub fn unary_message(f: UnaryFn) -> Message {
  Message::Unary(f)
}

/// Creates a constant message for the given value.
pub fn constant_message(value: f64) -> Message {
  Message::Constant(value)
}

/// Creates a degree-mode toggle message.
pub fn toggle_deg_message() -> Message {
  Message::ToggleDeg
}

/// Creates an inverse-mode toggle message.
pub fn toggle_inv_message() -> Message {
  Message::ToggleInv
}

/// Creates an equals message.
pub fn equals_message() -> Message {
  Message::Equals
}

/// Creates a clear message.
pub fn clear_message() -> Message {
  Message::Clear
}

/// Creates a backspace message.
pub fn backspace_message() -> Message {
  Message::Backspace
}

/// Wraps a raw keyboard event into a message.
pub fn key_event_message(event: iced::keyboard::Event) -> Message {
  Message::KeyEvent(event)
}
