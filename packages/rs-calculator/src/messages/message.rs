//! The Message enum. Messages the view can send in response to user input.
//!
//! Unchanged since main.rs days, just relocated. Everything matches on this.
use clanker_calculator::{OpToken, UnaryFn};
use iced::keyboard;

/// Messages the view can send in response to user input.
#[derive(Debug, Clone)]
pub enum Message {
  Digit(char),
  Dot,
  BinaryOp(OpToken),
  Unary(UnaryFn),
  Constant(f64),
  ToggleDeg,
  ToggleInv,
  Equals,
  Clear,
  Backspace,
  KeyEvent(keyboard::Event),
}
