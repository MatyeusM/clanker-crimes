//! Row G factory. Negate (±), 0, dot, and the double-wide "=". The closer.
//! Uses the wide button factory — the only row with that privilege. Equals
//! gets the Success style, as is proper for a finale.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::buttons::wide_button_factory::calc_button_wide;
use crate::messages::message_factory::{digit_message, dot_message, equals_message, unary_message};
use crate::state::calculator::Calculator;
use clanker_calculator::UnaryFn;
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row G: negate, zero, dot, wide equals.
pub fn build_row_g(app: &Calculator) -> Element<'_, Message> {
  let _ = app;
  row![
    calc_button(
      "\u{b1}",
      unary_message(UnaryFn::Negate),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button("0", digit_message('0'), resolve_style(&ButtonKind::Digit)),
    calc_button(".", dot_message(), resolve_style(&ButtonKind::Digit)),
    calc_button_wide("=", equals_message(), resolve_style(&ButtonKind::Success)),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
