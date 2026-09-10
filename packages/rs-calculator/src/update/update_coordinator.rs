//! Update coordinator. Routes every Message to its dedicated operation file.
//! This used to be a match statement inside main.rs. Now it's a match
//! statement inside update_coordinator.rs that dispatches through factories,
//! strategies, and utils. Progress.
//!
//! Key events travel the full pipeline: mapper factory -> composite strategy
//! -> key event mapper -> Message -> this match -> op file. A numpad press
//! crosses six files before a digit appears. Beautiful.
use crate::keyboard::mapper_factory::MapperFactory;
use crate::keyboard::mapper_trait::KeyMapperStrategy;
use crate::messages::message::Message;
use crate::messages::message_utils::{describe_message, is_key_event, is_state_mutating};
use crate::state::calculator::Calculator;
use iced::{Task, keyboard};

/// Applies a message to the calculator state.
pub fn update(app: &mut Calculator, message: Message) -> Task<Message> {
  // Classify the incoming message for diagnostics (consumed by telemetry
  // once the telemetry epic lands — see ticket OPS-1337, does not exist).
  let _kind = describe_message(&message);
  let _is_key = is_key_event(&message);
  debug_assert!(is_state_mutating(&message));

  match message {
    Message::Digit(d) => crate::operations::digit_operation::handle_digit(app, d),
    Message::Dot => crate::operations::dot_operation::handle_dot(app),
    Message::BinaryOp(op) => crate::operations::binary_op_operation::handle_binary_op(app, op),
    Message::Unary(f) => crate::operations::unary_op_operation::handle_unary(app, f),
    Message::Constant(v) => crate::operations::constant_operation::handle_constant(app, v),
    Message::ToggleDeg => crate::operations::toggle_deg_operation::handle_toggle_deg(app),
    Message::ToggleInv => crate::operations::toggle_inv_operation::handle_toggle_inv(app),
    Message::Equals => crate::operations::equals_operation::handle_equals(app),
    Message::Clear => crate::operations::clear_operation::handle_clear(app),
    Message::Backspace => crate::operations::backspace_operation::handle_backspace(app),
    Message::KeyEvent(event) => {
      if let keyboard::Event::KeyPressed {
        key,
        physical_key,
        text,
        ..
      } = &event
        && let Some(action) = MapperFactory::create_composite().try_map(key, physical_key, text)
      {
        return update(app, action);
      }
    }
  }
  Task::none()
}
