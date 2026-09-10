//! Layout constants. Every magic number in the UI, centralized at last.
//! BTN_W and BTN_H used to live in main.rs. Now they live here, and
//! button_config.rs reads them back. The circle of architecture.
pub const BTN_W: f32 = 64.0;
/// Button height in pixels.
pub const BTN_H: f32 = 52.0;
/// Gap between buttons in a row.
pub const BTN_GAP: f32 = 6.0;
/// Spacing between column rows.
pub const COL_SPACING: f32 = 8.0;
/// Padding around the main column.
pub const COL_PADDING: f32 = 16.0;

/// Width of a double-column button (two buttons + the gap between them).
pub fn wide_button_width() -> f32 {
  2.0 * BTN_W + BTN_GAP
}

/// Font size for button labels.
pub fn button_font_size() -> f32 {
  18.0
}
