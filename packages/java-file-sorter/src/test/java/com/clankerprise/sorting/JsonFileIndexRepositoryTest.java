package com.clankerprise.sorting;

import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.domain.TagRepository;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

/** Round-trip test for .index.json, including tricky strings. */
final class JsonFileIndexRepositoryTest {

  static void run() throws Exception {
    Path libraryRoot = Files.createTempDirectory("clanker-index");
    try {
      TagRepository sourceRepository = new TagRepository(libraryRoot);
      FileTagDescriptor fileEntry =
          new FileTagDescriptor(
              "id\"\\\n ä \uD83C\uDFB5",
              "Good\"Fellas\\: <cut>.mkv",
              "sub/dir/GoodFellas.mkv",
              ".files/GoodFellas.mkv",
              42L,
              "id\"\\\n ä \uD83C\uDFB5");
      ValidationAssertionHelper.assertTrue(
          fileEntry.registerTagValue("Actor", "Robert De \"Niro\""), "add tag 1");
      ValidationAssertionHelper.assertTrue(
          fileEntry.registerTagValue("Actor", "Joe Pesci"), "add tag 2");
      ValidationAssertionHelper.assertTrue(
          fileEntry.registerTagValue("Genre", "Crime\\Drama\nNoir"), "add tag 3");
      ValidationAssertionHelper.assertTrue(
          !fileEntry.registerTagValue("Actor", "robert de \"niro\""), "duplicate rejected");
      ValidationAssertionHelper.assertTrue(
          !fileEntry.registerTagValue("  ", "x"), "blank category rejected");
      sourceRepository.registerEntry(fileEntry);
      sourceRepository.recordLastSortCategory("Actor");
      sourceRepository.recordLastSortDirectories(List.of("Robert De Niro"));
      JsonFileIndexRepository.persistRepository(sourceRepository);

      TagRepository loaded = JsonFileIndexRepository.hydrateRepository(libraryRoot);
      ValidationAssertionHelper.assertEqual(1, loaded.countTotalEntries(), "file count");
      FileTagDescriptor reloadedFileEntry =
          loaded.findFirstEntryByContentId("id\"\\\n ä \uD83C\uDFB5");
      ValidationAssertionHelper.assertTrue(reloadedFileEntry != null, "lookup by id");
      ValidationAssertionHelper.assertEqual(
          "Good\"Fellas\\: <cut>.mkv", reloadedFileEntry.getDisplayLabel(), "displayName");
      ValidationAssertionHelper.assertEqual(
          ".files/GoodFellas.mkv", reloadedFileEntry.getStoredRelativePath(), "storedRel");
      ValidationAssertionHelper.assertEqual(42L, reloadedFileEntry.getFileSizeInBytes(), "size");
      ValidationAssertionHelper.assertEqual(
          List.of("Robert De \"Niro\"", "Joe Pesci"),
          reloadedFileEntry.retrieveTagValues("Actor"),
          "actor associatedValueList");
      ValidationAssertionHelper.assertEqual(
          List.of("Crime\\Drama\nNoir"),
          reloadedFileEntry.retrieveTagValues("Genre"),
          "genre value");
      ValidationAssertionHelper.assertEqual(
          "Actor", loaded.retrieveLastSortCategory(), "lastSortCategory");
      ValidationAssertionHelper.assertEqual(
          List.of("Robert De Niro"), loaded.retrieveLastSortDirectories(), "lastSortDirs");
      ValidationAssertionHelper.assertEqual(
          List.of("Actor", "Genre"), loaded.collectAllCategories(), "categories");

      // Loading from a folder without an index yields an missingIndexRepository sourceRepository.
      TagRepository missingIndexRepository =
          JsonFileIndexRepository.hydrateRepository(libraryRoot.resolve("nonexistent"));
      ValidationAssertionHelper.assertEqual(
          0, missingIndexRepository.countTotalEntries(), "missingIndexRepository sourceRepository");

      // Comma-separated input splits into multiple associatedValueList.
      FileTagDescriptor commaSeparatedInputEntry = new FileTagDescriptor("x", "x", "", "", 0L, "x");
      ValidationAssertionHelper.assertTrue(
          commaSeparatedInputEntry.registerTagValue("Actor", "Robert DeNiro, Leonardo DiCaprio ,"),
          "comma split added");
      ValidationAssertionHelper.assertEqual(
          List.of("Robert DeNiro", "Leonardo DiCaprio"),
          commaSeparatedInputEntry.retrieveTagValues("Actor"),
          "split associatedValueList");
      ValidationAssertionHelper.assertTrue(
          !commaSeparatedInputEntry.registerTagValue("Actor", "robert deniro"),
          "dup after split rejected");
      ValidationAssertionHelper.assertTrue(
          !commaSeparatedInputEntry.registerTagValue("Actor", " , , "), "blank rejected");
      ValidationAssertionHelper.assertTrue(
          commaSeparatedInputEntry.retrieveTagSnapshot().containsKey("Actor"), "category kept");

      // Legacy index files with comma-containing associatedValueList migrate on load.
      Path legacyIndexRoot = Files.createTempDirectory("clanker-migratedRepository");
      try {
        String legacyIndexDocument =
            "{\"version\": 1, \"lastSortCategory\": \"\", \"lastSortDirs\": [], \"files\": ["
                + "{\"id\": \"a\", \"displayName\": \"a\", \"originalRel\": \"a\","
                + " \"storedRel\": \"a\", \"size\": 1, \"sha256\": \"a\","
                + " \"tags\": {\"Actor\": [\"A, B\", \"C\", \"b \"]}}]}";
        Files.writeString(
            legacyIndexRoot.resolve(".index.json"),
            legacyIndexDocument,
            java.nio.charset.StandardCharsets.UTF_8);
        TagRepository migratedRepository =
            JsonFileIndexRepository.hydrateRepository(legacyIndexRoot);
        ValidationAssertionHelper.assertEqual(
            List.of("A", "B", "C"),
            migratedRepository.findFirstEntryByContentId("a").retrieveTagValues("Actor"),
            "migratedRepository comma associatedValueList split");
      } finally {
        deleteRecursively(legacyIndexRoot);
      }
    } finally {
      deleteRecursively(libraryRoot);
    }
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
