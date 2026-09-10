//! Wide button factory. Builds the double-width "=" button. It could have
//! been a boolean parameter on calc_button. It is not. It is a file.
use super::button_config::ButtonDims;
use crate::messages::message::Message;
use crate::ui::layout_constants::button_font_size;
use iced::{
  Element, Length, Theme,
  widget::{button, text},
};

/// A grid button that spans two columns' worth of width (used for "=").
pub fn calc_button_wide(
  label: &'static str,
  msg: Message,
  style: fn(&Theme, button::Status) -> button::Style,
) -> Element<'static, Message> {
  let dims = ButtonDims::wide_dims();
  button(text(label).size(button_font_size()))
    .width(Length::Fixed(dims.width))
    .height(Length::Fixed(dims.height))
    .style(style)
    .on_press(msg)
    .into()
}
