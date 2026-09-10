//! App configuration. Title plus window dimensions as a struct, for the day
//! the window size comes from a config file, an env var, or the blockchain.
use crate::window::app_title_provider::app_title;
use crate::window::window_settings_factory::{window_height, window_width};

/// Application-level configuration.
pub struct AppConfig {
  /// Window title.
  pub title: String,
  /// Window width in pixels.
  pub width: f32,
  /// Window height in pixels.
  pub height: f32,
}

impl AppConfig {
  /// The default configuration (matches the original main.rs literals).
  pub fn default_config() -> Self {
    Self {
      // Title flows through the provider so white-labeling only ever
      // touches one file (plus this one, plus the const, plus the trait).
      title: app_title(),
      width: window_width(),
      height: window_height(),
    }
  }
}
