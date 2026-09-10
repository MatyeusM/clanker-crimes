//! Degree-toggle operation. Handles Message::ToggleDeg. Flips a bool.
//! An entire file for `calc.deg_mode = !calc.deg_mode`. Architecture.
use crate::state::calculator::Calculator;

/// Toggles degree/radian mode.
pub fn handle_toggle_deg(calc: &mut Calculator) {
  calc.deg_mode = !calc.deg_mode;
}
