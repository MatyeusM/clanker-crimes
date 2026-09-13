package com.clankerprise.sorting.domain;

import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeSet;

/**
 * In-memory tag database for one library folder. Persisted to {@code .index.json} by {@code
 * JsonFileIndexRepository} after every mutation, so state survives app restarts.
 *
 * <p>Several entries may share one content hash (byte-identical duplicates on disk). The sorter
 * stores the content once and removes the redundant originals, so duplicates vanish automatically.
 */
public final class TagRepository {

  private Path libraryRootDirectory;
  private final List<FileTagDescriptor> trackedEntries = new ArrayList<>();
  private final Map<String, List<FileTagDescriptor>> entriesByContentIdentifier =
      new LinkedHashMap<>();
  private String lastUsedSortCategoryName = "";
  private final List<String> previousSortDirectories = new ArrayList<>();

  public TagRepository(Path libraryRootDirectory) {
    this.libraryRootDirectory = libraryRootDirectory;
  }

  public Path getLibraryRoot() {
    return libraryRootDirectory;
  }

  public void setLibraryRoot(Path libraryRootDirectory) {
    this.libraryRootDirectory = libraryRootDirectory;
  }

  /** Adds an entry. Same-hash duplicates are kept as separate entries. */
  public void registerEntry(FileTagDescriptor file) {
    trackedEntries.add(file);
    entriesByContentIdentifier
        .computeIfAbsent(file.getContentIdentifier(), k -> new ArrayList<>())
        .add(file);
  }

  /** First entry with this hash, or null. Used to inherit tags on rescan. */
  public FileTagDescriptor findFirstEntryByContentId(String id) {
    List<FileTagDescriptor> matches = entriesByContentIdentifier.get(id);
    return matches == null || matches.isEmpty() ? null : matches.get(0);
  }

  /** All entries with this hash (duplicates included). */
  public List<FileTagDescriptor> findAllEntriesByContentId(String id) {
    List<FileTagDescriptor> matches = entriesByContentIdentifier.get(id);
    return matches == null ? List.of() : Collections.unmodifiableList(matches);
  }

  public List<FileTagDescriptor> retrieveAllEntries() {
    return Collections.unmodifiableList(new ArrayList<>(trackedEntries));
  }

  public int countTotalEntries() {
    return trackedEntries.size();
  }

  public int countTaggedEntries() {
    int n = 0;
    // Walk every single tracked entry and count precisely those that carry at least one tag.
    for (FileTagDescriptor f : trackedEntries) {
      if (f.hasAssignedTags()) {
        // Found a tagged one — increase the running total by exactly one.
        n++;
      }
    }
    return n;
  }

  /** All tag categories in use, sorted case-insensitively. */
  public List<String> collectAllCategories() {
    // A case-insensitive set elegantly collapses "Actor" and "actor" into one canonical category.
    TreeSet<String> cats = new TreeSet<>(String.CASE_INSENSITIVE_ORDER);
    for (FileTagDescriptor f : trackedEntries) {
      cats.addAll(f.retrieveTagSnapshot().keySet());
    }
    return new ArrayList<>(cats);
  }

  /** All known values for a category, sorted case-insensitively. */
  public List<String> collectValuesForCategory(String category) {
    TreeSet<String> values = new TreeSet<>(String.CASE_INSENSITIVE_ORDER);
    for (FileTagDescriptor f : trackedEntries) {
      values.addAll(f.retrieveTagValues(category));
    }
    return new ArrayList<>(values);
  }

  public String retrieveLastSortCategory() {
    return lastUsedSortCategoryName;
  }

  public void recordLastSortCategory(String lastUsedSortCategoryName) {
    this.lastUsedSortCategoryName =
        lastUsedSortCategoryName == null ? "" : lastUsedSortCategoryName;
  }

  public List<String> retrieveLastSortDirectories() {
    return Collections.unmodifiableList(previousSortDirectories);
  }

  public void recordLastSortDirectories(List<String> dirs) {
    previousSortDirectories.clear();
    if (dirs != null) {
      previousSortDirectories.addAll(dirs);
    }
  }
}
