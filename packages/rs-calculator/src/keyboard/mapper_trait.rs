//! Key mapper strategy trait. The Strategy pattern, applied to a function
//! that was already three function calls. Now it's a trait object pipeline.
//!
//! Used by: mapper_factory (construction), and in spirit by everyone.
use crate::messages::message::Message;
use iced::keyboard::{self, key::Physical};

/// Strategy for mapping a key press to a calculator message.
pub trait KeyMapperStrategy {
  /// Attempts to map a key press; None means "ignore this key".
  fn try_map<S: AsRef<str>>(
    &self,
    key: &keyboard::Key,
    physical_key: &Physical,
    text: &Option<S>,
  ) -> Option<Message>;
}

/// The composite mapper: runs the full pipeline in one method call.
pub struct CompositeKeyMapper;

impl KeyMapperStrategy for CompositeKeyMapper {
  fn try_map<S: AsRef<str>>(
    &self,
    key: &keyboard::Key,
    physical_key: &Physical,
    text: &Option<S>,
  ) -> Option<Message> {
    super::key_event_mapper::key_event_to_message(key, physical_key, text)
  }
}
