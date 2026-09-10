//! Column assembler. Provides the spacing, padding, and alignment policy
//! for the main column. The view assembler does the actual assembling; this
//! file provides the numbers it assembles with. Separation of concerns.
use crate::ui::layout_constants::{COL_PADDING, COL_SPACING};
use iced::Alignment;

/// Spacing between the display and rows.
pub fn column_spacing() -> f32 {
  COL_SPACING
}

/// Padding around the main column.
pub fn column_padding() -> f32 {
  COL_PADDING
}

/// Horizontal alignment of the main column.
pub fn column_alignment() -> Alignment {
  Alignment::Center
}
