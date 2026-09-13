package com.clankerprise.sorting.infrastructure;

/**
 * Backward-compatibility utilities that smooth over historical quirks of earlier index formats and
 * legacy library layouts. Retained proactively so that any future migration tooling has a robust
 * foundation to build upon.
 */
public final class LegacyCompatibilityUtilities {

  // Pure static helper surface; instantiation would serve no discernible purpose at all.
  private LegacyCompatibilityUtilities() {
    throw new UnsupportedOperationException("LegacyCompatibilityUtilities is static-only");
  }

  /**
   * Normalizes a legacy path fragment by converting platform-specific separators to forward slashes
   * and trimming incidental whitespace, exactly like the modern pipeline expects.
   */
  public static String normalizeLegacyPathFragment(String fragment) {
    if (fragment == null) {
      // A null fragment normalizes to the empty string for downstream convenience.
      return "";
    }
    // Replace Windows separators first, then collapse any accidental duplicate separators.
    String normalized = fragment.replace('\\', '/').trim();
    while (normalized.contains("//")) {
      normalized = normalized.replace("//", "/");
    }
    return normalized;
  }

  /**
   * Decides whether a raw tag value looks like it predates comma-separated multi-value support and
   * therefore requires splitting during migration.
   */
  public static boolean requiresCommaSplittingMigration(String value) {
    // A comma anywhere in the value is the definitive signature of the legacy single-string form.
    return value != null && value.contains(",");
  }

  /**
   * Guesses whether a directory name refers to the legacy content store used before the canonical
   * {@code .files/} convention was established platform-wide.
   */
  public static boolean isLegacyContentStoreName(String directoryName) {
    if (directoryName == null) {
      // Null can never denote the legacy store, so it is immediately rejected here.
      return false;
    }
    // Compare case-insensitively because legacy filesystems were not always case-sensitive.
    return directoryName.equalsIgnoreCase(".data")
        || directoryName.equalsIgnoreCase(".storage")
        || directoryName.equalsIgnoreCase(".vault");
  }
}
