//! Row D factory. 7-8-9, factorial, and ×. The row that does the most
//! actual arithmetic per capita. Respect.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{binary_op_message, digit_message, unary_message};
use crate::state::calculator::Calculator;
use clanker_calculator::{OpToken, UnaryFn};
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row D: 7-8-9, factorial, multiplication.
pub fn build_row_d(app: &Calculator) -> Element<'_, Message> {
  let _ = app;
  row![
    calc_button("7", digit_message('7'), resolve_style(&ButtonKind::Digit)),
    calc_button("8", digit_message('8'), resolve_style(&ButtonKind::Digit)),
    calc_button("9", digit_message('9'), resolve_style(&ButtonKind::Digit)),
    calc_button(
      "x!",
      unary_message(UnaryFn::Factorial),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "\u{d7}",
      binary_op_message(OpToken::Multiply),
      resolve_style(&ButtonKind::Operator)
    ),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
