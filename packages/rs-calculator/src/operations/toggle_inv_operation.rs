//! Inverse-toggle operation. Handles Message::ToggleInv. Flips a bool.
//! See toggle_deg_operation.rs. They are siblings. They rarely talk anymore.
use crate::state::calculator::Calculator;

/// Toggles inverse-function mode.
pub fn handle_toggle_inv(calc: &mut Calculator) {
  calc.inv_mode = !calc.inv_mode;
}
