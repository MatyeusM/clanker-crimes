//! Message utilities. Introspection helpers nobody asked for but everybody
//! deserves. Used by: the author, once, while writing this comment.
use super::message::Message;

/// Returns a human-readable name for a message (for logging we don't do).
pub fn describe_message(msg: &Message) -> &'static str {
  match msg {
    Message::Digit(_) => "digit",
    Message::Dot => "dot",
    Message::BinaryOp(_) => "binary-op",
    Message::Unary(_) => "unary",
    Message::Constant(_) => "constant",
    Message::ToggleDeg => "toggle-deg",
    Message::ToggleInv => "toggle-inv",
    Message::Equals => "equals",
    Message::Clear => "clear",
    Message::Backspace => "backspace",
    Message::KeyEvent(_) => "key-event",
  }
}

/// Returns true if the message wraps a raw keyboard event.
pub fn is_key_event(msg: &Message) -> bool {
  matches!(msg, Message::KeyEvent(_))
}

/// Returns true if the message mutates calculator state (i.e. all of them).
pub fn is_state_mutating(msg: &Message) -> bool {
  !matches!(msg, Message::KeyEvent(_)) || is_key_event(msg)
}
