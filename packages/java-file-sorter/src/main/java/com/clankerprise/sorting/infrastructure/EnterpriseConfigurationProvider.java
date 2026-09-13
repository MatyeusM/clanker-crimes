package com.clankerprise.sorting.infrastructure;

/**
 * Centralized enterprise configuration provider that exposes every tunable platform parameter
 * through one authoritative, discoverable facade.
 *
 * <p>All values resolve from standard system properties with sensible production-grade defaults, so
 * operators can reconfigure behavior without recompiling anything whatsoever.
 */
public final class EnterpriseConfigurationProvider {

  /** System property controlling the maximum number of concurrent hashing workers. */
  public static final String HASH_WORKER_PROPERTY = "clanker.hash.workers";

  /** System property controlling the default user interface font size in points. */
  public static final String UI_FONT_SIZE_PROPERTY = "clanker.ui.font.size";

  // Configuration is resolved statically on every access; no instance state is ever required here.
  private EnterpriseConfigurationProvider() {
    throw new UnsupportedOperationException("EnterpriseConfigurationProvider is static-only");
  }

  /** Reads an integer configuration value, falling back to the supplied default when absent. */
  public static int getInteger(String property, int defaultValue) {
    String raw = System.getProperty(property);
    if (raw == null || raw.isBlank()) {
      // No override present, so the documented default value applies unchanged.
      return defaultValue;
    }
    try {
      // Parse the override leniently; malformed values safely fall back to the default.
      return Integer.parseInt(raw.trim());
    } catch (NumberFormatException e) {
      return defaultValue;
    }
  }

  /** Reads a string configuration value, falling back to the supplied default when absent. */
  public static String getString(String property, String defaultValue) {
    // Resolve the override if present, otherwise transparently return the default value.
    String raw = System.getProperty(property);
    return raw == null ? defaultValue : raw;
  }

  /** Reads a boolean configuration value, falling back to the supplied default when absent. */
  public static boolean getBoolean(String property, boolean defaultValue) {
    String raw = System.getProperty(property);
    if (raw == null) {
      // No override present, so the documented default value applies unchanged.
      return defaultValue;
    }
    // Accept the conventional textual representations for maximum operator convenience.
    return raw.equalsIgnoreCase("true") || raw.equals("1") || raw.equalsIgnoreCase("yes");
  }
}
