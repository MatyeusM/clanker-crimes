package com.clankerprise.sorting.infrastructure;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * Creates file links for the sort views.
 *
 * <p>Strategy (cross-platform): prefer a relative symbolic link (cheap, obvious, works on
 * Linux/macOS out of the box), fall back to a hard link (works on Windows without special
 * privileges since source and view live on the same volume), and finally to a plain copy. The
 * caller is told which kind was used so the UI can report it.
 */
public final class FileLinkProvisioningService {

  public enum ProvisionedLinkKind {
    SYMBOLIC_LINK,
    HARD_LINK,
    PLAIN_COPY
  }

  private FileLinkProvisioningService() {}

  public static ProvisionedLinkKind provisionLink(Path link, Path target) throws IOException {
    IOException symlinkFailure = null;
    try {
      Path relative = link.getParent().relativize(target);
      Files.createSymbolicLink(link, relative);
      return ProvisionedLinkKind.SYMBOLIC_LINK;
    } catch (IOException | UnsupportedOperationException | SecurityException e) {
      symlinkFailure =
          e instanceof IOException
              ? (IOException) e
              : new IOException("symlink not supported: " + e.getMessage(), e);
    }
    try {
      Files.createLink(link, target);
      return ProvisionedLinkKind.HARD_LINK;
    } catch (IOException | UnsupportedOperationException | SecurityException e) {
      // Last resort: copy the bytes. Original stays safe in .files/.
      try {
        Files.copy(target, link);
        return ProvisionedLinkKind.PLAIN_COPY;
      } catch (IOException copyFailure) {
        copyFailure.addSuppressed(symlinkFailure);
        copyFailure.addSuppressed(
            e instanceof IOException
                ? (IOException) e
                : new IOException("hardlink not supported: " + e.getMessage(), e));
        throw copyFailure;
      }
    }
  }
}
