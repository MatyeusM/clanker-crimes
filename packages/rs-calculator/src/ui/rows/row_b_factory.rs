//! Row B factory. Trig (sin/cos/tan + inverse variants), π, and backspace.
//! The row where degrees mode goes to prove itself.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{backspace_message, constant_message, unary_message};
use crate::state::calculator::Calculator;
use clanker_calculator::UnaryFn;
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row B: trig functions, π constant, backspace.
pub fn build_row_b(app: &Calculator) -> Element<'_, Message> {
  row![
    calc_button(
      if app.inv_mode {
        "sin\u{207b}\u{00b9}"
      } else {
        "sin"
      },
      unary_message(if app.inv_mode {
        UnaryFn::Asin
      } else {
        UnaryFn::Sin
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      if app.inv_mode {
        "cos\u{207b}\u{00b9}"
      } else {
        "cos"
      },
      unary_message(if app.inv_mode {
        UnaryFn::Acos
      } else {
        UnaryFn::Cos
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      if app.inv_mode {
        "tan\u{207b}\u{00b9}"
      } else {
        "tan"
      },
      unary_message(if app.inv_mode {
        UnaryFn::Atan
      } else {
        UnaryFn::Tan
      }),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "\u{3c0}",
      constant_message(std::f64::consts::PI),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      "\u{232b}",
      backspace_message(),
      resolve_style(&ButtonKind::Scientific)
    ),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
