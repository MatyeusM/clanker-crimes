//! App runner. Wires boot/update/view/subscription/window into
//! `iced::application` and runs it. The entire program, orchestrated in one
//! function that mostly names other functions. The conductor doesn't play an
//! instrument, and yet.
use crate::app::app_config::AppConfig;
use crate::state::calculator::Calculator;

/// Runs the calculator application.
pub fn run_app() -> iced::Result {
  // Resolve configuration first so the window title and the runtime agree
  // on what this application is called. Single source of truth (the config,
  // which itself reads the title provider, which reads the const).
  let config = AppConfig::default_config();

  iced::application(
    crate::update::boot_coordinator::boot,
    crate::update::update_coordinator::update,
    crate::ui::view_assembler::view,
  )
  .title(move |_state: &Calculator| config.title.clone())
  .subscription(crate::update::subscription_coordinator::subscription)
  .window(crate::window::window_settings_factory::create_window_settings())
  .run()
}
