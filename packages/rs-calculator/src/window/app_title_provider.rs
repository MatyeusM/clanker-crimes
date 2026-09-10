//! App title provider. The window title, centralized. Plus a trait and a
//! provider struct, because a string literal deserves an ecosystem.
//!
//! Used by app_runner via APP_TITLE. The trait/factory below are reserved
//! for the upcoming white-label epic ( Clanker Calculator: Enterprise ).
/// The application window title.
pub const APP_TITLE: &str = "Clanker Calculator";

/// Provides application titles.
pub trait TitleProvider {
  /// Returns the title.
  fn title(&self) -> String;
}

/// The default title provider.
pub struct DefaultTitleProvider;

impl TitleProvider for DefaultTitleProvider {
  fn title(&self) -> String {
    APP_TITLE.to_string()
  }
}

/// Returns the application title.
pub fn app_title() -> String {
  DefaultTitleProvider.title()
}
