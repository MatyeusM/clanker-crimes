//! Keyboard module. Maps raw key presses to Messages. Six files.
//!
//! Pipeline: key_event_mapper (orchestrator) -> numpad_mapper (physical
//! codes) -> logical_key_mapper (Key::Character/Named) -> text_mapper
//! (produced-text fallback). Plus a strategy trait and a factory, because
//! two files for a match statement is plenty but six is a career.
pub mod key_event_mapper;
pub mod logical_key_mapper;
pub mod mapper_factory;
pub mod mapper_trait;
pub mod numpad_mapper;
pub mod text_mapper;
