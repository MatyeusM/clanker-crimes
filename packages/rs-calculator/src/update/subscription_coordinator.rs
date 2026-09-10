//! Subscription coordinator. Subscribes to raw keyboard events and wraps
//! them as Message::KeyEvent via the message factory. One line of logic.
//! One file. You do the math.
use crate::messages::message::Message;
use crate::messages::message_factory::key_event_message;
use crate::state::calculator::Calculator;
use iced::{Subscription, keyboard};

/// Listens for keyboard input.
pub fn subscription(_app: &Calculator) -> Subscription<Message> {
  keyboard::listen().map(key_event_message)
}
