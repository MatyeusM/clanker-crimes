//! Standard button factory. Builds one fixed-size grid button so every row
//! lines up. Label, message, style in; Element out. The workhorse.
use super::button_config::ButtonDims;
use crate::messages::message::Message;
use crate::ui::layout_constants::button_font_size;
use iced::{
  Element, Length, Theme,
  widget::{button, text},
};

/// A single grid button with a fixed size, so every row lines up.
pub fn calc_button(
  label: &'static str,
  msg: Message,
  style: fn(&Theme, button::Status) -> button::Style,
) -> Element<'static, Message> {
  let dims = ButtonDims::default_dims();
  button(text(label).size(button_font_size()))
    .width(Length::Fixed(dims.width))
    .height(Length::Fixed(dims.height))
    .style(style)
    .on_press(msg)
    .into()
}
