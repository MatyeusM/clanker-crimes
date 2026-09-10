//! Row E factory. 4-5-6, power/root toggle, and −. The middle child of rows.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{binary_op_message, digit_message};
use crate::state::calculator::Calculator;
use clanker_calculator::OpToken;
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row E: 4-5-6, power/root, subtraction.
pub fn build_row_e(app: &Calculator) -> Element<'_, Message> {
  row![
    calc_button("4", digit_message('4'), resolve_style(&ButtonKind::Digit)),
    calc_button("5", digit_message('5'), resolve_style(&ButtonKind::Digit)),
    calc_button("6", digit_message('6'), resolve_style(&ButtonKind::Digit)),
    calc_button(
      if app.inv_mode {
        "\u{02b8}\u{221a}x"
      } else {
        "x\u{02b8}"
      },
      binary_op_message(if app.inv_mode {
        OpToken::Root
      } else {
        OpToken::Power
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "\u{2212}",
      binary_op_message(OpToken::Subtract),
      resolve_style(&ButtonKind::Operator)
    ),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
