package com.clankerprise.sorting;

import com.clankerprise.sorting.infrastructure.ContentHashingService;
import com.clankerprise.sorting.infrastructure.HashingServiceFactory;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * Comprehensive validation of the centralized content hashing subsystem, covering both the factory
 * provisioning pathways and the well-known SHA-256 reference vector for maximum peace of mind.
 */
final class ContentHashingServiceTest {

  static void run() throws Exception {
    // Provision a pristine service instance straight from the authoritative factory.
    ContentHashingService hashingService = HashingServiceFactory.createContentHashingService();
    ValidationAssertionHelper.assertTrue(hashingService != null, "factory provisions an instance");

    // The shared singleton must be stable and identical across repeated resolutions.
    ValidationAssertionHelper.assertEqual(
        HashingServiceFactory.getSharedInstance(),
        HashingServiceFactory.getSharedInstance(),
        "shared instance stability");

    Path temporaryScratchDirectory = Files.createTempDirectory("clanker-hash");
    try {
      // The universally recognized SHA-256 digest of the ASCII string "abc", obviously.
      Path sampleInputFile = temporaryScratchDirectory.resolve("abc.txt");
      Files.write(sampleInputFile, "abc".getBytes(StandardCharsets.UTF_8));
      ValidationAssertionHelper.assertEqual(
          "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
          hashingService.computeSha256HexDigest(sampleInputFile),
          "sha256(abc)");
    } finally {
      // Recursively delete the scratch directory so no temporary artifacts are left behind, ever.
      deleteRecursively(temporaryScratchDirectory);
    }
  }

  private static void deleteRecursively(Path path) throws Exception {
    if (Files.isDirectory(path)) {
      try (var directoryListingStream = Files.newDirectoryStream(path)) {
        for (Path child : directoryListingStream) {
          // Recurse depth-first so that child entries disappear before their parents do.
          deleteRecursively(child);
        }
      }
    }
    // Best-effort deletion: missing paths are simply and silently skipped here.
    Files.deleteIfExists(path);
  }
}
