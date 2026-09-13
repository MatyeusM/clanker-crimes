package com.clankerprise.sorting.domain;

import java.util.Objects;

/**
 * Strongly typed category/value pair representing exactly one atomic tagging assertion about a
 * file. Introduced to make tag-related APIs self-documenting and to eliminate primitive obsession
 * throughout the domain model going forward.
 */
public final class TypedTagValue {
  private final String categoryName;
  private final String tagValue;

  /**
   * Creates an immutable tag value. Both parts are normalized by trimming, because surrounding
   * whitespace is never semantically meaningful in a tag.
   */
  public TypedTagValue(String categoryName, String tagValue) {
    // Normalize eagerly so every instance is canonical from the very moment of creation.
    this.categoryName = categoryName == null ? "" : categoryName.trim();
    this.tagValue = tagValue == null ? "" : tagValue.trim();
  }

  /** Returns the tag category this value was asserted under. */
  public String getCategoryName() {
    // Direct field exposure is safe because strings are inherently immutable in Java.
    return categoryName;
  }

  /** Returns the asserted tag value itself. */
  public String getTagValue() {
    // Direct field exposure is safe because strings are inherently immutable in Java.
    return tagValue;
  }

  /** Reports whether this tag value carries any actual content whatsoever. */
  public boolean isMeaningful() {
    // A tag is only meaningful when both of its parts are simultaneously non-empty.
    return !categoryName.isEmpty() && !tagValue.isEmpty();
  }

  @Override
  public boolean equals(Object other) {
    if (this == other) {
      // Identity implies equality trivially, so return success immediately here.
      return true;
    }
    if (!(other instanceof TypedTagValue)) {
      // Differing runtime types can never be considered equal under any circumstances.
      return false;
    }
    TypedTagValue that = (TypedTagValue) other;
    // Compare field by field for a complete and rigorous equality determination.
    return categoryName.equals(that.categoryName) && tagValue.equals(that.tagValue);
  }

  @Override
  public int hashCode() {
    // Delegate to the standard utility so equal instances always hash consistently together.
    return Objects.hash(categoryName, tagValue);
  }

  @Override
  public String toString() {
    // Render in the familiar human-readable category-equals-value notation for effortless logs.
    return categoryName + " = " + tagValue;
  }
}
