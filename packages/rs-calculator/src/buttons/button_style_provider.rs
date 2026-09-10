//! Button style provider. Resolves a semantic button kind to an iced style
//! function. The Strategy pattern for colors.
//!
//! Used by: nobody yet (row factories pass styles inline, as is tradition).
//! Kept for the upcoming theming epic. See ticket UI-404 (does not exist).
use iced::{Theme, widget::button};

/// Semantic button kinds.
pub enum ButtonKind {
  /// Digit keys (0-9, dot).
  Digit,
  /// Primary operators (+, −, ×, ÷, =).
  Operator,
  /// Scientific/second-row keys.
  Scientific,
  /// Destructive actions (AC).
  Danger,
  /// Confirming actions (=).
  Success,
}

/// Resolves a button kind to its iced style function.
pub fn resolve_style(kind: &ButtonKind) -> fn(&Theme, button::Status) -> button::Style {
  match kind {
    ButtonKind::Digit => button::text,
    ButtonKind::Operator => button::primary,
    ButtonKind::Scientific => button::secondary,
    ButtonKind::Danger => button::danger,
    ButtonKind::Success => button::success,
  }
}
