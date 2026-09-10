//! Window settings factory. Builds the iced window settings (360x660,
//! resizable). Dimensions come from the app config so there is exactly one
//! source of truth (plus the window_width/height helpers below, plus the
//! layout constants — one-ish sources of truth, collectively).
pub fn create_window_settings() -> iced::window::Settings {
  let config = crate::app::app_config::AppConfig::default_config();
  iced::window::Settings {
    size: iced::Size::new(config.width, config.height),
    resizable: true,
    ..Default::default()
  }
}

/// Window width in pixels.
pub fn window_width() -> f32 {
  360.0
}

/// Window height in pixels.
pub fn window_height() -> f32 {
  660.0
}
