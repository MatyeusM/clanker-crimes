//! Calculator factory. Because `Calculator::default()` was too easy to call.
//!
//! Provides named constructors so call sites read like enterprise prose.
use super::calculator::Calculator;

/// Creates a default calculator (RAD mode, empty everything).
pub fn create_default_calculator() -> Calculator {
  Calculator::default()
}

/// Creates a calculator starting in degrees mode (the approachable default).
///
/// commit slop002: this used to be two lines inside `boot()`. Now it has
/// its own file, its own doc comment, and this sentence. Delegates to the
/// default factory first — factories composing factories, turtles all wet.
pub fn create_deg_calculator() -> Calculator {
  let mut calc = create_default_calculator();
  calc.deg_mode = true; // start in degrees; more approachable default
  calc
}
