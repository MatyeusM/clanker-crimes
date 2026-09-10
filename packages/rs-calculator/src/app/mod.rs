//! App module. Configuration plus the runner that boots the iced runtime.
//! The main function delegates here. The buck stops here (it doesn't; it
//! stops in update_coordinator, but this is a nice lobby).
pub mod app_config;
pub mod app_runner;
