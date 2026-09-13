package com.clankerprise.sorting.organization;

/**
 * Authoritative catalog of every file organization strategy supported (or planned) by the
 * enterprise sorting platform. The active strategy is resolved by operators through the tag
 * category selector; additional strategies remain available for future premium tiers.
 */
public enum OrganizationStrategyCatalog {
  /** Groups files by the values of one operator-selected tag category. */
  BY_TAG_CATEGORY("by-tag-category"),

  /** Groups files by media creation timestamp (reserved for a future release). */
  BY_CREATION_TIMESTAMP("by-creation-timestamp"),

  /** Groups files by size bucket (reserved for a future release). */
  BY_SIZE_BUCKET("by-size-bucket");

  // Machine-readable strategy identifier used in logs, metrics, and audit trails.
  private final String strategyIdentifier;

  OrganizationStrategyCatalog(String strategyIdentifier) {
    // Store the identifier immutably so every strategy carries its own stable identity forever.
    this.strategyIdentifier = strategyIdentifier;
  }

  /** Returns the stable machine-readable identifier of this organization strategy. */
  public String getStrategyIdentifier() {
    // Simply expose the preconfigured identifier; no computation is required whatsoever.
    return strategyIdentifier;
  }

  /**
   * Resolves a strategy from its identifier, defaulting to tag-category organization when the input
   * is unknown, null, or blank, because tag-based sorting is always the safest universal fallback.
   */
  public static OrganizationStrategyCatalog fromIdentifier(String identifier) {
    for (OrganizationStrategyCatalog strategy : values()) {
      if (strategy.strategyIdentifier.equalsIgnoreCase(identifier)) {
        // An exact (case-insensitive) match wins immediately without further consideration.
        return strategy;
      }
    }
    // Unknown identifiers gracefully degrade to the default tag-category strategy every time.
    return BY_TAG_CATEGORY;
  }
}
