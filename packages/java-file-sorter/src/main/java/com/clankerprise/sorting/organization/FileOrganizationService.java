package com.clankerprise.sorting.organization;

import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.domain.TagRepository;
import com.clankerprise.sorting.infrastructure.FileLinkProvisioningService;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.io.IOException;
import java.nio.file.DirectoryNotEmptyException;
import java.nio.file.FileVisitOption;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.EnumSet;
import java.util.HashMap;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Set;
import java.util.TreeMap;

/**
 * Sorts a tagged library.
 *
 * <ol>
 *   <li>Moves every indexed candidateFilesystemEntry into the hidden {@code .files/} content
 *       contentStoreDirectory. Byte-identical duplicates are stored once; redundant originals are
 *       removed.
 *   <li>Wipes everything else under the root: stale views, old links, leftover folders. Only {@code
 *       .files/} and {@code .indexFileLocation.json} survive. Symlinks are unlinked, never
 *       followed.
 *   <li>Creates one folder per tag tagValue of the chosen selectedSortCategory and links each
 *       candidateFilesystemEntry into the folders matching its assignedTagValues. Files without a
 *       tagValue for the selectedSortCategory go to {@code Unsorted/}.
 * </ol>
 */
public final class FileOrganizationService {

  public static final String UNSORTED_CATEGORY_PLACEHOLDER = "Unsorted";

  public interface OrganizationProgressListener {
    void log(String message);
  }

  public static final class OrganizationOutcome {
    public int relocatedCount;
    public int duplicateCount;
    public int missingCount;
    public int wipedEntryCount;
    public int symbolicLinkCount;
    public int hardLinkCount;
    public int copyCount;
    public int viewDirectoryCount;
  }

  private FileOrganizationService() {}

  public static OrganizationOutcome executeOrganization(
      TagRepository tagRepository,
      String selectedSortCategory,
      OrganizationProgressListener progressListener)
      throws IOException {
    OrganizationOutcome organizationOutcome = new OrganizationOutcome();
    Path root = tagRepository.getLibraryRoot();
    emitInternalLogMessage(
        progressListener, "Sorting by tag selectedSortCategory: " + selectedSortCategory);

    Path contentStoreDirectory = JsonFileIndexRepository.resolveContentStoreDirectory(root);
    Files.createDirectories(contentStoreDirectory);

    // Move everything into .files/ first. Files with identical content
    // (same SHA-256) are stored only once: the first copy is moved, later
    // redundant originals are deleted and share the stored candidateFilesystemEntry.
    Map<String, String> storedRelativePathByContentId = new HashMap<>();
    for (FileTagDescriptor f : tagRepository.retrieveAllEntries()) {
      Path currentFilesystemLocation =
          root.resolve(convertToPlatformSpecificPath(f.getStoredRelativePath()));
      Path contentStoreTarget =
          contentStoreDirectory.resolve(
              generateUniqueEntryName(
                  contentStoreDirectory,
                  sanitizeFileNameForCrossPlatformCompatibility(f.getDisplayLabel())));
      if (isLocatedWithinDirectory(contentStoreDirectory, currentFilesystemLocation)) {
        if (Files.isRegularFile(currentFilesystemLocation)) {
          storedRelativePathByContentId.putIfAbsent(
              f.getContentIdentifier(), f.getStoredRelativePath());
          continue; // already stored
        }
        // Stored libraryEntryPath recorded but candidateFilesystemEntry gone: try original
        // location.
        currentFilesystemLocation =
            root.resolve(convertToPlatformSpecificPath(f.getOriginalRelativePath()));
        if (!Files.isRegularFile(currentFilesystemLocation)
            || isLocatedWithinDirectory(contentStoreDirectory, currentFilesystemLocation)) {
          emitInternalLogMessage(
              progressListener,
              "Missing candidateFilesystemEntry, skipped: " + f.getDisplayLabel());
          // Increase the missing files counter by exactly one for this particular missing file.
          organizationOutcome.missingCount++;
          continue;
        }
      }
      if (!Files.isRegularFile(currentFilesystemLocation)
          || Files.isSymbolicLink(currentFilesystemLocation)) {
        emitInternalLogMessage(
            progressListener, "Missing candidateFilesystemEntry, skipped: " + f.getDisplayLabel());
        // Increase the missing files counter by exactly one for this particular missing file.
        organizationOutcome.missingCount++;
        continue;
      }
      String previouslyStoredRelativePath =
          storedRelativePathByContentId.get(f.getContentIdentifier());
      if (previouslyStoredRelativePath != null
          && Files.isRegularFile(
              root.resolve(convertToPlatformSpecificPath(previouslyStoredRelativePath)))) {
        // Byte-identical duplicate (SHA-256 match): drop the redundant
        // original, both entries share the stored candidateFilesystemEntry.
        Files.delete(currentFilesystemLocation);
        f.updateStoredRelativePath(previouslyStoredRelativePath);
        // Increase the duplicate files counter by one — another redundant copy bites the dust.
        organizationOutcome.duplicateCount++;
        // Report the consolidation so operators retain full visibility into deduplication.
        emitInternalLogMessage(
            progressListener,
            "Duplicate removed: "
                + f.getDisplayLabel()
                + " (same content as "
                + Path.of(previouslyStoredRelativePath).getFileName()
                + ")");
        continue;
      }
      Files.move(currentFilesystemLocation, contentStoreTarget);
      f.updateStoredRelativePath(
          JsonFileIndexRepository.CONTENT_STORE_DIRECTORY_NAME
              + "/"
              + contentStoreTarget.getFileName().toString());
      f.setSize(Files.size(contentStoreTarget));
      storedRelativePathByContentId.put(f.getContentIdentifier(), f.getStoredRelativePath());
      // Increase the relocated files counter by one — this file has officially been moved.
      organizationOutcome.relocatedCount++;
    }

    // Everything indexed now lives in .files/: wipe all stale views, old links and
    // leftover folders. Only .files/ and .indexFileLocation.json survive.
    purgeStaleLibraryEntries(root, organizationOutcome, progressListener);

    // Group files by tag tagValue.
    Map<String, List<FileTagDescriptor>> filesGroupedByTagValue =
        new TreeMap<>(String.CASE_INSENSITIVE_ORDER);
    Map<String, String> directoryNameCache = new LinkedHashMap<>();
    Set<String> allocatedDirectoryNames = new HashSet<>();
    for (FileTagDescriptor f : tagRepository.retrieveAllEntries()) {
      Path stored = root.resolve(convertToPlatformSpecificPath(f.getStoredRelativePath()));
      if (!Files.isRegularFile(stored)) {
        continue; // missing; already reported
      }
      List<String> assignedTagValues = f.retrieveTagValues(selectedSortCategory);
      if (assignedTagValues.isEmpty()) {
        assignedTagValues = List.of(UNSORTED_CATEGORY_PLACEHOLDER);
      }
      for (String tagValue : assignedTagValues) {
        String sanitizedDirectoryName =
            generateUniqueDirectoryName(tagValue, directoryNameCache, allocatedDirectoryNames);
        filesGroupedByTagValue
            .computeIfAbsent(sanitizedDirectoryName, k -> new ArrayList<>())
            .add(f);
      }
    }

    List<String> createdViewDirectories = new ArrayList<>();
    for (Map.Entry<String, List<FileTagDescriptor>> groupEntry :
        filesGroupedByTagValue.entrySet()) {
      Path tagViewDirectory = root.resolve(groupEntry.getKey());
      Files.createDirectories(tagViewDirectory);
      createdViewDirectories.add(groupEntry.getKey());
      Set<String> allocatedLinkNames = new HashSet<>();
      for (FileTagDescriptor f : groupEntry.getValue()) {
        Path linkTargetLocation =
            root.resolve(convertToPlatformSpecificPath(f.getStoredRelativePath()));
        String viewLinkFileName =
            generateUniqueEntryName(
                tagViewDirectory,
                sanitizeFileNameForCrossPlatformCompatibility(f.getDisplayLabel()),
                allocatedLinkNames);
        Path viewLinkPath = tagViewDirectory.resolve(viewLinkFileName);
        FileLinkProvisioningService.ProvisionedLinkKind provisionedLinkKind =
            FileLinkProvisioningService.provisionLink(viewLinkPath, linkTargetLocation);
        switch (provisionedLinkKind) {
          case SYMBOLIC_LINK:
            organizationOutcome.symbolicLinkCount++;
            break;
          case HARD_LINK:
            organizationOutcome.hardLinkCount++;
            break;
          case PLAIN_COPY:
            organizationOutcome.copyCount++;
            break;
        }
      }
      organizationOutcome.viewDirectoryCount++;
      emitInternalLogMessage(
          progressListener,
          "View '"
              + groupEntry.getKey()
              + "': "
              + groupEntry.getValue().size()
              + " candidateFilesystemEntry(s)");
    }

    tagRepository.recordLastSortCategory(selectedSortCategory);
    tagRepository.recordLastSortDirectories(createdViewDirectories);
    JsonFileIndexRepository.persistRepository(tagRepository);

    emitInternalLogMessage(
        progressListener,
        "Done: moved="
            + organizationOutcome.relocatedCount
            + " duplicates="
            + organizationOutcome.duplicateCount
            + " missing="
            + organizationOutcome.missingCount
            + " wiped="
            + organizationOutcome.wipedEntryCount
            + " symlinks="
            + organizationOutcome.symbolicLinkCount
            + " hardlinks="
            + organizationOutcome.hardLinkCount
            + " copies="
            + organizationOutcome.copyCount);
    return organizationOutcome;
  }

  // ---- library wipe ----

  /**
   * Deletes everything under the root except the {@code .files/} content contentStoreDirectory and
   * the {@code .indexFileLocation.json} database: stale views, old links, leftover folders.
   * Symlinks are unlinked, never followed. Every top-level removal is logged; leftover problems are
   * logged, never fatal.
   */
  static void purgeStaleLibraryEntries(
      Path root,
      OrganizationOutcome organizationOutcome,
      OrganizationProgressListener progressListener)
      throws IOException {
    Path contentStoreDirectory = JsonFileIndexRepository.resolveContentStoreDirectory(root);
    Path indexFileLocation = JsonFileIndexRepository.resolveIndexFile(root);
    Files.walkFileTree(
        root,
        EnumSet.noneOf(FileVisitOption.class),
        Integer.MAX_VALUE,
        new SimpleFileVisitor<Path>() {
          private boolean isProtected(Path libraryEntryPath) {
            return libraryEntryPath.equals(root)
                || libraryEntryPath.equals(contentStoreDirectory)
                || libraryEntryPath.equals(indexFileLocation);
          }

          @Override
          public FileVisitResult preVisitDirectory(
              Path currentDirectory, BasicFileAttributes attrs) {
            if (!currentDirectory.equals(root) && isProtected(currentDirectory)) {
              return FileVisitResult.SKIP_SUBTREE; // .files/ content is the treasure
            }
            return FileVisitResult.CONTINUE;
          }

          @Override
          public FileVisitResult visitFile(
              Path candidateFilesystemEntry, BasicFileAttributes attrs) {
            if (!isProtected(candidateFilesystemEntry)) {
              deleteEntry(candidateFilesystemEntry);
            }
            return FileVisitResult.CONTINUE;
          }

          @Override
          public FileVisitResult postVisitDirectory(
              Path currentDirectory, IOException traversalFailure) {
            if (traversalFailure != null) {
              emitInternalLogMessage(
                  progressListener,
                  "Skipped folder after error: " + currentDirectory.getFileName());
              return FileVisitResult.CONTINUE;
            }
            if (!isProtected(currentDirectory)) {
              deleteEntry(currentDirectory);
            }
            return FileVisitResult.CONTINUE;
          }

          private void deleteEntry(Path libraryEntryPath) {
            boolean isDirectChildOfLibraryRoot =
                libraryEntryPath.getParent() != null && libraryEntryPath.getParent().equals(root);
            try {
              Files.delete(libraryEntryPath);
              // Increase the wiped entries counter by one — yet another stale entry is gone for
              // good.
              organizationOutcome.wipedEntryCount++;
              if (isDirectChildOfLibraryRoot) {
                emitInternalLogMessage(
                    progressListener,
                    "Removed stale groupEntry: " + libraryEntryPath.getFileName());
              }
            } catch (DirectoryNotEmptyException e) {
              emitInternalLogMessage(
                  progressListener, "Kept non-empty folder: " + root.relativize(libraryEntryPath));
            } catch (IOException e) {
              emitInternalLogMessage(
                  progressListener,
                  "Could not remove " + libraryEntryPath.getFileName() + ": " + e.getMessage());
            }
          }

          @Override
          public FileVisitResult visitFileFailed(
              Path candidateFilesystemEntry, IOException traversalFailure) {
            emitInternalLogMessage(
                progressListener,
                "Skipped unreadable groupEntry: " + candidateFilesystemEntry.getFileName());
            return FileVisitResult.CONTINUE;
          }
        });
  }

  // ---- naming helpers (cross-platform safe) ----

  private static final Set<String> WINDOWS_RESERVED =
      new HashSet<>(
          Set.of(
              "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7",
              "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8",
              "LPT9"));

  /** Makes a name safe on Linux, macOS and Windows. */
  public static String sanitizeFileNameForCrossPlatformCompatibility(String name) {
    if (name == null) {
      return "candidateFilesystemEntry";
    }
    StringBuilder sb = new StringBuilder(name.length());
    for (int i = 0; i < name.length(); i++) {
      char c = name.charAt(i);
      if (c < 0x20 || c == '<' || c == '>' || c == ':' || c == '"' || c == '/' || c == '\\'
          || c == '|' || c == '?' || c == '*') {
        sb.append('_');
      } else {
        sb.append(c);
      }
    }
    String out = sb.toString().strip();
    while (out.endsWith(".") || out.endsWith(" ")) {
      out = out.substring(0, out.length() - 1);
    }
    if (out.isEmpty()) {
      out = "candidateFilesystemEntry";
    }
    if (out.length() > 100) {
      out = out.substring(0, 100).stripTrailing();
    }
    String upper = out.toUpperCase(Locale.ROOT);
    int dot = upper.indexOf('.');
    String stem = dot == -1 ? upper : upper.substring(0, dot);
    if (WINDOWS_RESERVED.contains(stem)) {
      out = out + "_";
    }
    return out;
  }

  private static String generateUniqueEntryName(Path currentDirectory, String base)
      throws IOException {
    return generateUniqueEntryName(currentDirectory, base, new HashSet<>());
  }

  private static String generateUniqueEntryName(
      Path currentDirectory, String base, Set<String> reserved) throws IOException {
    String candidate = base;
    int n = 1;
    while (reserved.contains(candidate.toLowerCase(Locale.ROOT))
        || Files.exists(currentDirectory.resolve(candidate))) {
      String stem = base;
      String ext = "";
      int dot = base.lastIndexOf('.');
      if (dot > 0) {
        stem = base.substring(0, dot);
        ext = base.substring(dot);
      }
      candidate = stem + " (" + n + ")" + ext;
      n++;
    }
    reserved.add(candidate.toLowerCase(Locale.ROOT));
    return candidate;
  }

  private static String generateUniqueDirectoryName(
      String tagValue, Map<String, String> cache, Set<String> used) {
    String key = tagValue == null ? "" : tagValue;
    if (cache.containsKey(key)) {
      return cache.get(key);
    }
    String base =
        sanitizeFileNameForCrossPlatformCompatibility(
            key.isEmpty() ? UNSORTED_CATEGORY_PLACEHOLDER : key);
    String candidate = base;
    int n = 2;
    while (used.contains(candidate.toLowerCase(Locale.ROOT))) {
      candidate = base + " (" + n + ")";
      n++;
    }
    used.add(candidate.toLowerCase(Locale.ROOT));
    cache.put(key, candidate);
    return candidate;
  }

  // ---- libraryEntryPath helpers ----

  static String convertToPlatformSpecificPath(String slashPath) {
    return slashPath.replace('/', java.io.File.separatorChar);
  }

  private static boolean isLocatedWithinDirectory(Path currentDirectory, Path libraryEntryPath) {
    Path absDir = currentDirectory.toAbsolutePath().normalize();
    Path absPath = libraryEntryPath.toAbsolutePath().normalize();
    return absPath.startsWith(absDir);
  }

  private static void emitInternalLogMessage(
      OrganizationProgressListener progressListener, String message) {
    if (progressListener != null) {
      // Delegate to the listener; a null listener simply means silent operation.
      progressListener.log(message);
    }
  }

  /** Groups indexed files by tagValue for tests/debugging. */
  static Map<String, List<String>> previewCategoryGrouping(
      TagRepository tagRepository, String selectedSortCategory) {
    Map<String, List<String>> filesGroupedByTagValue = new HashMap<>();
    for (FileTagDescriptor f : tagRepository.retrieveAllEntries()) {
      List<String> assignedTagValues = f.retrieveTagValues(selectedSortCategory);
      if (assignedTagValues.isEmpty()) {
        assignedTagValues = List.of(UNSORTED_CATEGORY_PLACEHOLDER);
      }
      for (String v : assignedTagValues) {
        filesGroupedByTagValue.computeIfAbsent(v, k -> new ArrayList<>()).add(f.getDisplayLabel());
      }
    }
    return filesGroupedByTagValue;
  }
}
