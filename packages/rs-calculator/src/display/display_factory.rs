//! Display factory. Builds the readout container from calculator state.
//! Delegates string-building to the render service and prettiness to the
//! style provider (well, to iced's rounded_box via the style provider's
//! padding constant — the layering is load-bearing).
use super::display_style_provider::display_padding;
use crate::messages::message::Message;
use crate::state::calculator::Calculator;
use crate::state::calculator_render_service::render_display;
use iced::{
  Element, Length,
  widget::{container, text},
};

/// Builds the calculator display widget.
pub fn create_display(app: &Calculator) -> Element<'_, Message> {
  container(text(render_display(app)).size(super::display_style_provider::display_font_size()))
    .width(Length::Fill)
    .padding(display_padding())
    .style(container::rounded_box)
    .into()
}
