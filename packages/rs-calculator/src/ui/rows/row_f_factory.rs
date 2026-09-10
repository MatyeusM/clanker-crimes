//! Row F factory. 1-2-3, percent, and +. The people's row.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{binary_op_message, digit_message, unary_message};
use crate::state::calculator::Calculator;
use clanker_calculator::{OpToken, UnaryFn};
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row F: 1-2-3, percent, addition.
pub fn build_row_f(app: &Calculator) -> Element<'_, Message> {
  let _ = app;
  row![
    calc_button("1", digit_message('1'), resolve_style(&ButtonKind::Digit)),
    calc_button("2", digit_message('2'), resolve_style(&ButtonKind::Digit)),
    calc_button("3", digit_message('3'), resolve_style(&ButtonKind::Digit)),
    calc_button(
      "%",
      unary_message(UnaryFn::Percent),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "+",
      binary_op_message(OpToken::Add),
      resolve_style(&ButtonKind::Operator)
    ),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
