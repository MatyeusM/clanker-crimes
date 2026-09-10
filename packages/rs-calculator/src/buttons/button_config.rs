//! Button configuration. Dimensions as a struct, because bare constants are
//! for the layout_constants module (which also exists). Redundancy is just
//! backup. Backup is good.
use crate::ui::layout_constants::{BTN_H, BTN_W};

/// Dimensions for calculator grid buttons.
pub struct ButtonDims {
  /// Button width in pixels.
  pub width: f32,
  /// Button height in pixels.
  pub height: f32,
}

impl ButtonDims {
  /// The standard grid button size.
  pub fn default_dims() -> Self {
    Self {
      width: BTN_W,
      height: BTN_H,
    }
  }

  /// The double-width size used by the "=" button (two buttons + one gap).
  pub fn wide_dims() -> Self {
    Self {
      width: crate::ui::layout_constants::wide_button_width(),
      height: BTN_H,
    }
  }
}
