package com.clankerprise.sorting.presentation;

import com.clankerprise.sorting.domain.TagRepository;
import java.nio.file.Path;

/**
 * Mutable session state shared by all presentation panels throughout the lifetime of one
 * application run. Acts as the single authoritative carrier for the currently selected library
 * location, its associated tag repository, and the operator's position within the tagging workflow.
 */
public final class ApplicationSessionState {
  // Root directory of the library currently loaded into the application, if any has been chosen.
  public Path activeLibraryRoot;
  // Tag repository belonging to the active library; populated incrementally by every scan.
  public TagRepository tagRepository;
  // Zero-based position of the file currently displayed in the interactive tagging workflow.
  public int currentTagPosition;
}
