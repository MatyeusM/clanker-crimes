package com.clankerprise.sorting;

import com.clankerprise.sorting.discovery.FileDiscoveryService;
import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.domain.TagRepository;
import com.clankerprise.sorting.organization.FileOrganizationService;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/** FileOrganizationService tests: views/links, re-sort cleanup, and duplicate elimination. */
final class FileOrganizationServiceTest {

  static void run() throws Exception {
    runSortViews();
    runDedup();
  }

  /** End-to-end: scan -> tag -> sort -> re-sort, incl. viewLinkPath content checks. */
  static void runSortViews() throws Exception {
    Path testLibraryRoot = Files.createTempDirectory("clanker-sort");
    try {
      Files.write(
          testLibraryRoot.resolve("GoodFellas.mkv"), "goodfellas".getBytes(StandardCharsets.UTF_8));
      Files.createDirectories(testLibraryRoot.resolve("sub"));
      Files.write(
          testLibraryRoot.resolve("sub").resolve("Casino.mkv"),
          "casino".getBytes(StandardCharsets.UTF_8));
      Files.write(testLibraryRoot.resolve("Notes.txt"), "notes".getBytes(StandardCharsets.UTF_8));
      Files.createDirectories(testLibraryRoot.resolve("emptydir"));
      Files.createDirectories(testLibraryRoot.resolve("nested/deep"));
      Files.createDirectories(testLibraryRoot.resolve(".hidden-empty"));

      List<FileDiscoveryService.DiscoveredFile> discoveredFiles =
          FileDiscoveryService.performScan(testLibraryRoot, null);
      ValidationAssertionHelper.assertEqual(3, discoveredFiles.size(), "scanned count");
      // Parallel hashing must keep a deterministic, sortedReferenceOrder order.
      List<String> relativePathInventory = new ArrayList<>();
      for (FileDiscoveryService.DiscoveredFile discovered : discoveredFiles) {
        relativePathInventory.add(discovered.relativeSlashPath);
      }
      List<String> sortedReferenceOrder = new ArrayList<>(relativePathInventory);
      java.util.Collections.sort(sortedReferenceOrder);
      ValidationAssertionHelper.assertEqual(
          sortedReferenceOrder, relativePathInventory, "scan order deterministic");
      ValidationAssertionHelper.assertTrue(
          FileDiscoveryService.recommendedWorkerCount() >= 1, "hash threads");

      TagRepository tagRepository = new TagRepository(testLibraryRoot);
      for (FileDiscoveryService.DiscoveredFile discovered : discoveredFiles) {
        String name = discovered.absolutePath.getFileName().toString();
        tagRepository.registerEntry(
            new FileTagDescriptor(
                discovered.contentHash,
                name,
                discovered.relativeSlashPath,
                discovered.relativeSlashPath,
                discovered.fileSizeInBytes,
                discovered.contentHash));
      }
      for (FileTagDescriptor fileEntry : tagRepository.retrieveAllEntries()) {
        if (fileEntry.getDisplayLabel().equals("GoodFellas.mkv")) {
          fileEntry.registerTagValue("Actor", "Robert De Niro");
          fileEntry.registerTagValue("Genre", "Crime");
        } else if (fileEntry.getDisplayLabel().equals("Casino.mkv")) {
          fileEntry.registerTagValue("Actor", "Robert De Niro");
          fileEntry.registerTagValue("Actor", "Joe Pesci");
          fileEntry.registerTagValue("Genre", "Crime/Drama:cut"); // needs sanitizing in views
        }
      }
      JsonFileIndexRepository.persistRepository(tagRepository);

      List<String> operationLog = new ArrayList<>();
      FileOrganizationService.OrganizationOutcome organizationOutcome =
          FileOrganizationService.executeOrganization(tagRepository, "Actor", operationLog::add);
      ValidationAssertionHelper.assertEqual(3, organizationOutcome.relocatedCount, "moved");
      ValidationAssertionHelper.assertEqual(0, organizationOutcome.missingCount, "missing");
      ValidationAssertionHelper.assertTrue(
          organizationOutcome.symbolicLinkCount
                  + organizationOutcome.hardLinkCount
                  + organizationOutcome.copyCount
              == 4,
          "viewLinkPath count");

      // Originals live in .files/ now.
      ValidationAssertionHelper.assertTrue(
          Files.isRegularFile(testLibraryRoot.resolve(".files/GoodFellas.mkv")),
          ".files GoodFellas");
      ValidationAssertionHelper.assertTrue(
          Files.isRegularFile(testLibraryRoot.resolve(".files/Casino.mkv")), ".files Casino");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("sub/Casino.mkv")), "original gone");

      // Everything except .files/ and .indexFileLocation.json is wiped after the move.
      ValidationAssertionHelper.assertTrue(
          organizationOutcome.wipedEntryCount >= 5, "stale entries wiped");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("sub")), "emptied sub removed");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("emptydir")), "empty currentDirectory removed");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("nested")), "nested empties removed");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve(".hidden-empty")),
          "hidden currentDirectory removed");
      ValidationAssertionHelper.assertTrue(
          Files.isRegularFile(testLibraryRoot.resolve(".index.json")),
          "indexFileLocation survives the wipe");

      // Views resolve to identical bytes, whatever viewLinkPath provisionedLinkKind was used.
      assertContent(testLibraryRoot.resolve("Robert De Niro/GoodFellas.mkv"), "goodfellas");
      assertContent(testLibraryRoot.resolve("Robert De Niro/Casino.mkv"), "casino");
      assertContent(testLibraryRoot.resolve("Joe Pesci/Casino.mkv"), "casino");
      assertContent(testLibraryRoot.resolve("Unsorted/Notes.txt"), "notes");

      // A previous view replaced by a symlink must be unlinked, not followed,
      // while a foreign symlink outside the views must survive the re-sort.
      boolean symlinksOk = false;
      try {
        deleteRecursively(testLibraryRoot.resolve("Joe Pesci"));
        Files.createSymbolicLink(
            testLibraryRoot.resolve("Joe Pesci"), testLibraryRoot.resolve("Robert De Niro"));
        Files.createSymbolicLink(
            testLibraryRoot.resolve("user-viewLinkPath"), testLibraryRoot.resolve(".files"));
        symlinksOk = true;
      } catch (IOException | UnsupportedOperationException | SecurityException e) {
        // No symlink privileges (e.g. default Windows): restore a plain currentDirectory.
        Files.createDirectories(testLibraryRoot.resolve("Joe Pesci"));
      }

      // Re-sort by Genre: old Actor views must be cleaned, new ones created.
      FileOrganizationService.OrganizationOutcome genreSortOutcome =
          FileOrganizationService.executeOrganization(tagRepository, "Genre", operationLog::add);
      ValidationAssertionHelper.assertEqual(
          3,
          genreSortOutcome.viewDirectoryCount,
          "genre view dirs"); // Crime, Crime_Drama_cut, Unsorted
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("Robert De Niro")), "old view removed");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("Joe Pesci")), "old view removed");
      assertContent(testLibraryRoot.resolve("Crime/GoodFellas.mkv"), "goodfellas");
      // "Crime/Drama:cut" contains illegal chars -> sanitized to "Crime_Drama_cut".
      assertContent(testLibraryRoot.resolve("Crime_Drama_cut/Casino.mkv"), "casino");
      assertContent(testLibraryRoot.resolve("Unsorted/Notes.txt"), "notes");
      if (symlinksOk) {
        ValidationAssertionHelper.assertTrue(
            !Files.exists(testLibraryRoot.resolve("user-viewLinkPath")), "stale symlink wiped");
      }

      // Stale content inside old views is wiped too: only .files/ and .indexFileLocation.json
      // survive.
      Files.write(
          testLibraryRoot.resolve("Crime/DO_NOT_DELETE.txt"),
          "mine".getBytes(StandardCharsets.UTF_8));
      FileOrganizationService.executeOrganization(tagRepository, "Actor", operationLog::add);
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("Crime/DO_NOT_DELETE.txt")),
          "foreign candidateFilesystemEntry wiped");
      ValidationAssertionHelper.assertTrue(
          Files.isRegularFile(testLibraryRoot.resolve(".files/GoodFellas.mkv")),
          ".files groupEntry kept");
      ValidationAssertionHelper.assertTrue(
          Files.isRegularFile(testLibraryRoot.resolve(".index.json")),
          "indexFileLocation survives the re-sort");

      // Index survived with tags + last sort state.
      TagRepository reloaded = JsonFileIndexRepository.hydrateRepository(testLibraryRoot);
      ValidationAssertionHelper.assertEqual(3, reloaded.countTotalEntries(), "reloaded count");
      ValidationAssertionHelper.assertEqual(
          "Actor", reloaded.retrieveLastSortCategory(), "reloaded sort selectedSortCategory");
      ValidationAssertionHelper.assertTrue(
          reloaded.countTaggedEntries() == 2, "reloaded tagged count");
    } finally {
      deleteRecursively(testLibraryRoot);
    }
  }

  /** Byte-identical files are stored once; redundant originals are removed. */
  static void runDedup() throws Exception {
    Path testLibraryRoot = Files.createTempDirectory("clanker-dedup");
    try {
      Files.write(testLibraryRoot.resolve("Casino.mkv"), "casino".getBytes(StandardCharsets.UTF_8));
      Files.write(
          testLibraryRoot.resolve("Casino-backup.mkv"), "casino".getBytes(StandardCharsets.UTF_8));

      List<FileDiscoveryService.DiscoveredFile> discoveredFiles =
          FileDiscoveryService.performScan(testLibraryRoot, null);
      ValidationAssertionHelper.assertEqual(2, discoveredFiles.size(), "scanned count");
      ValidationAssertionHelper.assertEqual(
          discoveredFiles.get(0).contentHash, discoveredFiles.get(1).contentHash, "same hash");

      TagRepository tagRepository = new TagRepository(testLibraryRoot);
      for (FileDiscoveryService.DiscoveredFile discovered : discoveredFiles) {
        String name = discovered.absolutePath.getFileName().toString();
        tagRepository.registerEntry(
            new FileTagDescriptor(
                discovered.contentHash,
                name,
                discovered.relativeSlashPath,
                discovered.relativeSlashPath,
                discovered.fileSizeInBytes,
                discovered.contentHash));
      }
      for (FileTagDescriptor fileEntry : tagRepository.retrieveAllEntries()) {
        fileEntry.registerTagValue("Actor", "Robert De Niro");
      }

      List<String> operationLog = new ArrayList<>();
      FileOrganizationService.OrganizationOutcome organizationOutcome =
          FileOrganizationService.executeOrganization(tagRepository, "Actor", operationLog::add);
      ValidationAssertionHelper.assertEqual(1, organizationOutcome.relocatedCount, "moved");
      ValidationAssertionHelper.assertEqual(
          1, organizationOutcome.duplicateCount, "duplicates removed");
      ValidationAssertionHelper.assertEqual(
          0, organizationOutcome.wipedEntryCount, "nothing stale to wipe");
      ValidationAssertionHelper.assertTrue(
          !Files.exists(testLibraryRoot.resolve("Casino-backup.mkv")), "dup original gone");
      long storedFileCount;
      try (var directoryListingStream = Files.list(testLibraryRoot.resolve(".files"))) {
        storedFileCount = directoryListingStream.filter(Files::isRegularFile).count();
      }
      ValidationAssertionHelper.assertEqual(1L, storedFileCount, ".files holds one copy");
      // Both entries share the stored candidateFilesystemEntry and both views resolve.
      ValidationAssertionHelper.assertEqual(
          tagRepository.retrieveAllEntries().get(0).getStoredRelativePath(),
          tagRepository.retrieveAllEntries().get(1).getStoredRelativePath(),
          "shared storedRel");
      assertContent(testLibraryRoot.resolve("Robert De Niro/Casino.mkv"), "casino");
      assertContent(testLibraryRoot.resolve("Robert De Niro/Casino-backup.mkv"), "casino");
    } finally {
      deleteRecursively(testLibraryRoot);
    }
  }

  private static void assertContent(Path viewLinkPath, String expected) throws Exception {
    ValidationAssertionHelper.assertTrue(
        Files.exists(viewLinkPath), "view groupEntry exists: " + viewLinkPath);
    ValidationAssertionHelper.assertEqual(
        expected,
        Files.readString(viewLinkPath, StandardCharsets.UTF_8),
        "content via " + viewLinkPath);
  }

  private static void deleteRecursively(Path p) throws Exception {
    if (Files.isDirectory(p) && !Files.isSymbolicLink(p)) {
      try (var ds = Files.newDirectoryStream(p)) {
        for (Path child : ds) {
          deleteRecursively(child);
        }
      }
    }
    Files.deleteIfExists(p);
  }
}
