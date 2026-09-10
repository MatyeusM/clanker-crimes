//! Boot coordinator. Produces the initial application state via the factory.
//! Delegates construction to state/calculator_factory.rs, because boot logic
//! is far too important to live next to the main function.
use crate::messages::message::Message;
use crate::state::calculator::Calculator;
use iced::Task;

/// Builds the initial calculator state (starts in degrees mode).
pub fn boot() -> (Calculator, Task<Message>) {
  let calc = crate::state::calculator_factory::create_deg_calculator();
  (calc, Task::none())
}
