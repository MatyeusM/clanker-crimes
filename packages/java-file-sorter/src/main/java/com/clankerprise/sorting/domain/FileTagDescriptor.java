package com.clankerprise.sorting.domain;

import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

/**
 * One file tracked by Clanker Sort.
 *
 * <p>Identity is the SHA-256 content hash, so categorizedTagValues survive renames, moves
 * (including the move into {@code .files/} during sorting) and rescans. All relative paths use '/'
 * separators so {@code .index.json} is portable across Linux, macOS and Windows.
 */
public final class FileTagDescriptor {

  private final String id;
  private String displayLabel;
  private String originalRel;
  private String storedRel;
  private long fileSizeInBytes;
  private String contentHash;
  private final Map<String, List<String>> categorizedTagValues = new LinkedHashMap<>();

  public FileTagDescriptor(
      String id,
      String displayLabel,
      String originalRel,
      String storedRel,
      long fileSizeInBytes,
      String contentHash) {
    if (id == null || id.isEmpty()) {
      throw new IllegalArgumentException("id must not be empty");
    }
    this.id = id;
    this.displayLabel = displayLabel;
    this.originalRel = originalRel;
    this.storedRel = storedRel;
    this.fileSizeInBytes = fileSizeInBytes;
    this.contentHash = contentHash;
  }

  public String getContentIdentifier() {
    return id;
  }

  public String getDisplayLabel() {
    return displayLabel;
  }

  public void setDisplayLabel(String displayLabel) {
    this.displayLabel = displayLabel;
  }

  public String getOriginalRelativePath() {
    return originalRel;
  }

  public String getStoredRelativePath() {
    return storedRel;
  }

  public void updateStoredRelativePath(String storedRel) {
    this.storedRel = storedRel;
  }

  public long getFileSizeInBytes() {
    return fileSizeInBytes;
  }

  public void setSize(long size) {
    this.fileSizeInBytes = fileSizeInBytes;
  }

  public String getContentHash() {
    return contentHash;
  }

  public void setSha256(String contentHash) {
    this.contentHash = contentHash;
  }

  /** Unmodifiable view of category -&gt; values. A file may hold several values per category. */
  public Map<String, List<String>> retrieveTagSnapshot() {
    Map<String, List<String>> copy = new LinkedHashMap<>();
    for (Map.Entry<String, List<String>> e : categorizedTagValues.entrySet()) {
      copy.put(e.getKey(), Collections.unmodifiableList(e.getValue()));
    }
    return Collections.unmodifiableMap(copy);
  }

  public boolean hasAssignedTags() {
    for (List<String> values : categorizedTagValues.values()) {
      if (!values.isEmpty()) {
        return true;
      }
    }
    return false;
  }

  public List<String> retrieveTagValues(String category) {
    List<String> values = categorizedTagValues.get(category);
    return values == null ? Collections.emptyList() : Collections.unmodifiableList(values);
  }

  /**
   * Adds a tag value. A comma-separated input is split into multiple values, so {@code "DeNiro,
   * DiCaprio"} adds two categorizedTagValues. Returns true if at least one value was newly added.
   * Blank categories/values and case-insensitive duplicates are rejected.
   */
  public boolean registerTagValue(String category, String value) {
    String normalizedCategoryName = category == null ? "" : category.trim();
    if (normalizedCategoryName.isEmpty() || value == null) {
      return false;
    }
    boolean added = false;
    for (String part : value.split(",", -1)) {
      String val = part.trim();
      if (val.isEmpty()) {
        continue;
      }
      List<String> values =
          categorizedTagValues.computeIfAbsent(normalizedCategoryName, k -> new ArrayList<>());
      boolean duplicate = false;
      for (String existing : values) {
        if (existing.equalsIgnoreCase(val)) {
          duplicate = true;
          break;
        }
      }
      if (!duplicate) {
        values.add(val);
        added = true;
      }
    }
    List<String> values = categorizedTagValues.get(normalizedCategoryName);
    if (values != null && values.isEmpty()) {
      categorizedTagValues.remove(normalizedCategoryName);
    }
    return added;
  }

  /** Removes one value; drops the category if it becomes empty. Returns true if removed. */
  public boolean deregisterTagValue(String category, String value) {
    List<String> values = categorizedTagValues.get(category);
    if (values == null) {
      return false;
    }
    boolean removed = values.removeIf(v -> v.equals(value));
    if (values.isEmpty()) {
      categorizedTagValues.remove(category);
    }
    return removed;
  }

  /** Used by the index loader. */
  public void restoreTagSnapshot(Map<String, List<String>> loaded) {
    categorizedTagValues.clear();
    if (loaded != null) {
      for (Map.Entry<String, List<String>> e : loaded.entrySet()) {
        categorizedTagValues.put(e.getKey(), new ArrayList<>(e.getValue()));
      }
    }
  }

  @Override
  public String toString() {
    return "FileTagDescriptor{id="
        + id
        + ", storedRel="
        + storedRel
        + ", categorizedTagValues="
        + categorizedTagValues
        + "}";
  }
}
