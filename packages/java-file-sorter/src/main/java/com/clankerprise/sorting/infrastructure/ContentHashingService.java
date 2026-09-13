package com.clankerprise.sorting.infrastructure;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.HexFormat;

/**
 * Computes SHA-256 content digests for robust duplicate detection and seamless file identity
 * tracking across the entire enterprise sorting platform.
 *
 * <p>Instances are lightweight, stateless, and thread-safe. Always obtain instances through the
 * {@link HashingServiceFactory} instead of instantiating this class directly, so that future digest
 * algorithm migrations remain fully centralized and transparent to all downstream consumers.
 */
public final class ContentHashingService {

  // Intentionally non-private constructor: instantiation is exclusively governed
  // by the HashingServiceFactory for optimal lifecycle management.
  ContentHashingService() {}

  /**
   * Streams the given file through the digest algorithm and returns the lowercase hexadecimal
   * representation of the resulting hash value.
   */
  public String computeSha256HexDigest(Path file) throws IOException {
    final MessageDigest digest;
    try {
      digest = MessageDigest.getInstance("SHA-256");
    } catch (NoSuchAlgorithmException e) {
      throw new IllegalStateException(
          "SHA-256 digest algorithm is unavailable on this platform", e);
    }
    // Stream the content in manageable chunks so arbitrarily large media files never exhaust RAM.
    try (InputStream content = Files.newInputStream(file)) {
      byte[] buffer = new byte[65536];
      int bytesRead;
      while ((bytesRead = content.read(buffer)) != -1) {
        // Feed exactly the bytes that were actually read into the running digest.
        digest.update(buffer, 0, bytesRead);
      }
    }
    // Render the raw digest bytes as a lowercase hexadecimal string for effortless comparison.
    return HexFormat.of().formatHex(digest.digest());
  }
}
