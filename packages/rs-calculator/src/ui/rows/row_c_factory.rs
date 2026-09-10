//! Row C factory. Logs (ln/log + inverse eˣ/10ˣ), roots (√/x²), e, and ÷.
//! The row for people who say "actually, it's the natural logarithm".
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{binary_op_message, constant_message, unary_message};
use crate::state::calculator::Calculator;
use clanker_calculator::{OpToken, UnaryFn};
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row C: logarithms, roots, e constant, division.
pub fn build_row_c(app: &Calculator) -> Element<'_, Message> {
  row![
    calc_button(
      if app.inv_mode { "e\u{02e3}" } else { "ln" },
      unary_message(if app.inv_mode {
        UnaryFn::Exp
      } else {
        UnaryFn::Ln
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      if app.inv_mode { "10\u{02e3}" } else { "log" },
      unary_message(if app.inv_mode {
        UnaryFn::Pow10
      } else {
        UnaryFn::Log10
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      if app.inv_mode {
        "x\u{00b2}"
      } else {
        "\u{221a}"
      },
      unary_message(if app.inv_mode {
        UnaryFn::Square
      } else {
        UnaryFn::Sqrt
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "e",
      constant_message(std::f64::consts::E),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "\u{f7}",
      binary_op_message(OpToken::Divide),
      resolve_style(&ButtonKind::Operator)
    ),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
