//! State module — owns the calculator's memory.
//!
//! History: extracted from main.rs on slop day. Previously an `impl` block,
//! now a beautiful constellation of free functions across many files.
//! See also: operations/, which does what the impl block used to do.
pub mod calculator;
pub mod calculator_factory;
pub mod calculator_render_service;
