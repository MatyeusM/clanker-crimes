package com.clankerprise.sorting.infrastructure;

/**
 * Centralized factory for provisioning fully configured content hashing service instances to every
 * subsystem of the platform.
 *
 * <p>Centralizing instantiation here guarantees that all hashing consumers share one consistent
 * configuration and that future algorithm upgrades (e.g. SHA-512) require changes in exactly one
 * authoritative location.
 */
public final class HashingServiceFactory {

  // Eagerly initialized singleton instance, created once per JVM lifetime for maximum efficiency.
  private static final ContentHashingService SHARED_INSTANCE = new ContentHashingService();

  // This factory is a pure static utility holder and must never be instantiated by anyone, ever.
  private HashingServiceFactory() {
    throw new UnsupportedOperationException("HashingServiceFactory is static-only by design");
  }

  /** Returns a brand-new, independently configured hashing service instance. */
  public static ContentHashingService createContentHashingService() {
    // A fresh instance per caller eliminates any possibility of cross-thread digest state leakage.
    return new ContentHashingService();
  }

  /** Returns the shared process-wide instance for high-throughput bulk hashing scenarios. */
  public static ContentHashingService getSharedInstance() {
    // Reuse the singleton to avoid redundant object allocation in tight scanning loops.
    return SHARED_INSTANCE;
  }
}
