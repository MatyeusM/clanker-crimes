//! Key event orchestrator. Runs the three-stage mapping pipeline:
//! physical numpad codes -> logical key -> produced text. First hit wins.
//!
//! `iced::keyboard::listen()` reports every raw keyboard event; this maps a
//! key press into a calculator Message, or None to ignore it.
use crate::messages::message::Message;
use iced::keyboard::{self, key::Physical};

/// Maps a key press (key + physical code + text) to a calculator message.
pub fn key_event_to_message<S: AsRef<str>>(
  key: &keyboard::Key,
  physical_key: &Physical,
  text: &Option<S>,
) -> Option<Message> {
  // 1. Physical numpad keys — layout-independent, works with NumLock on or off.
  if let Physical::Code(code) = physical_key
    && let Some(action) = super::numpad_mapper::map_numpad_code(code)
  {
    return Some(action);
  }

  // 2. Logical key (existing main-keyboard behavior).
  if let Some(action) = super::logical_key_mapper::key_to_message(key) {
    return Some(action);
  }

  // 3. Produced text fallback (e.g. `Key::Unidentified` with text "5").
  super::text_mapper::map_text(text)
}
