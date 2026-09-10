//! Operations module. Every former `Calculator` method is now a free function
//! in its own file. Twelve files. The `impl` block is gone. Long live the
//! functions. Coordinated by update/update_coordinator.rs.
pub mod backspace_operation;
pub mod binary_op_operation;
pub mod clear_operation;
pub mod constant_operation;
pub mod digit_operation;
pub mod dot_operation;
pub mod equals_operation;
pub mod input_helpers;
pub mod toggle_deg_operation;
pub mod toggle_inv_operation;
pub mod unary_op_operation;
