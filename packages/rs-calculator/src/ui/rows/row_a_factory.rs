//! Row A factory. Mode toggles (DEG/RAD, INV), parens, and the big red AC.
//! The control row. The row that means business. Messages come from the
//! factory, styles from the provider — this file contains zero direct enum
//! constructions and is proud of it.
use crate::buttons::button_factory::calc_button;
use crate::buttons::button_style_provider::{ButtonKind, resolve_style};
use crate::messages::message_factory::{
  binary_op_message, clear_message, toggle_deg_message, toggle_inv_message,
};
use crate::state::calculator::Calculator;
use clanker_calculator::OpToken;
use iced::{Element, widget::row};

use crate::messages::message::Message;

/// Builds row A: mode toggles, parentheses, all-clear.
pub fn build_row_a(app: &Calculator) -> Element<'_, Message> {
  row![
    calc_button(
      if app.deg_mode { "DEG" } else { "RAD" },
      toggle_deg_message(),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      if app.inv_mode { "INV\u{2022}" } else { "INV" },
      toggle_inv_message(),
      if app.inv_mode {
        resolve_style(&ButtonKind::Operator)
      } else {
        resolve_style(&ButtonKind::Scientific)
      }
    ),
    calc_button(
      "(",
      binary_op_message(OpToken::OpenParen),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button(
      ")",
      binary_op_message(OpToken::CloseParen),
      resolve_style(&ButtonKind::Scientific)
    ),
    calc_button("AC", clear_message(), resolve_style(&ButtonKind::Danger)),
  ]
  .spacing(crate::ui::layout_constants::BTN_GAP)
  .into()
}
