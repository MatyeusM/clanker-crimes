package com.clankerprise.sorting.discovery;

import com.clankerprise.sorting.infrastructure.HashingServiceFactory;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.io.File;
import java.io.IOException;
import java.nio.file.FileVisitResult;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.SimpleFileVisitor;
import java.nio.file.attribute.BasicFileAttributes;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.EnumSet;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Recursively scans a library folder and hashes every candidateFile.
 *
 * <p>Hashing runs on a thread hashingWorkerPool (up to 8 threads), which matters for large media
 * files where SHA-256 dominates. Results keep a deterministic order (sorted by relative path), so
 * tagging order is stable across runs. Unreadable files are reported via {@link
 * DiscoveryProgressListener#onError} and skipped instead of aborting the whole scan.
 *
 * <p>Skipped: dot-files/dot-directories, the {@code .files/} content store, the {@code .index.json}
 * database, and symbolic links (so previously generated sort views are never double-counted). Links
 * are not followed.
 */
public final class FileDiscoveryService {

  /** One candidateFile found on disk, before merging with the tag db. */
  public static final class DiscoveredFile {
    public final Path absolutePath;
    public final String relativeSlashPath;
    public final long fileSizeInBytes;
    public final String contentHash;

    DiscoveredFile(
        Path absolutePath, String relativeSlashPath, long fileSizeInBytes, String contentHash) {
      this.absolutePath = absolutePath;
      this.relativeSlashPath = relativeSlashPath;
      this.fileSizeInBytes = fileSizeInBytes;
      this.contentHash = contentHash;
    }
  }

  public interface DiscoveryProgressListener {
    /** Called after each hashed candidateFile. Return true to cancellationRequested the scan. */
    boolean reportProgress(int completedFileCounter, int totalFileCount, String entryName);

    /** Called for files that cannot be read; the candidateFile is skipped. */
    default void reportFailure(Path candidateFile, IOException error) {}
  }

  private FileDiscoveryService() {}

  public static boolean isSkippableEntryName(String entryName) {
    return entryName.startsWith(".");
  }

  /** Hash threads used; also useful for tests. */
  public static int recommendedWorkerCount() {
    return Math.max(1, Math.min(Runtime.getRuntime().availableProcessors(), 8));
  }

  public static List<DiscoveredFile> performScan(
      Path libraryRoot, DiscoveryProgressListener listener) throws IOException {
    ExecutorService hashingWorkerPool =
        Executors.newFixedThreadPool(
            recommendedWorkerCount(),
            runnable -> {
              Thread thread = new Thread(runnable, "clanker-scan");
              thread.setDaemon(true);
              return thread;
            });
    try {
      return performScan(libraryRoot, listener, hashingWorkerPool);
    } finally {
      hashingWorkerPool.shutdownNow();
    }
  }

  /**
   * Same as {@link #scan(Path, DiscoveryProgressListener)} but hashes on the given executor, which
   * the caller owns (it is not shut down here). Exists so tests can inject an instrumented
   * hashingWorkerPool and prove hashing really runs in parallel.
   */
  public static List<DiscoveredFile> performScan(
      Path libraryRoot, DiscoveryProgressListener listener, ExecutorService hashingWorkerPool)
      throws IOException {
    final List<Path> candidateFileList = new ArrayList<>();
    Files.walkFileTree(
        libraryRoot,
        EnumSet.noneOf(java.nio.file.FileVisitOption.class),
        Integer.MAX_VALUE,
        new SimpleFileVisitor<Path>() {
          @Override
          public FileVisitResult preVisitDirectory(Path dir, BasicFileAttributes fileAttributes) {
            if (Files.isSymbolicLink(dir)) {
              return FileVisitResult.SKIP_SUBTREE;
            }
            if (!dir.equals(libraryRoot)) {
              String entryName = dir.getFileName() == null ? "" : dir.getFileName().toString();
              if (entryName.equals(JsonFileIndexRepository.CONTENT_STORE_DIRECTORY_NAME)
                  || isSkippableEntryName(entryName)) {
                return FileVisitResult.SKIP_SUBTREE;
              }
            }
            return FileVisitResult.CONTINUE;
          }

          @Override
          public FileVisitResult visitFile(Path candidateFile, BasicFileAttributes fileAttributes) {
            if (Files.isSymbolicLink(candidateFile) || !fileAttributes.isRegularFile()) {
              return FileVisitResult.CONTINUE;
            }
            String entryName = candidateFile.getFileName().toString();
            if (entryName.equals(JsonFileIndexRepository.INDEX_STORAGE_FILENAME)
                || isSkippableEntryName(entryName)) {
              return FileVisitResult.CONTINUE;
            }
            candidateFileList.add(candidateFile);
            return FileVisitResult.CONTINUE;
          }
        });
    candidateFileList.sort(Comparator.comparing(p -> libraryRoot.relativize(p).toString()));

    final int totalFileCount = candidateFileList.size();
    final DiscoveredFile[] indexedResultsByPosition = new DiscoveredFile[totalFileCount];
    final AtomicInteger completedFileCounter = new AtomicInteger();
    final AtomicBoolean cancellationRequested = new AtomicBoolean(false);
    final CountDownLatch completionLatch = new CountDownLatch(totalFileCount);

    // Submit one hashing task per candidate file; the pool intentionally runs them concurrently.
    for (int i = 0; i < totalFileCount; i++) {
      final int index = i;
      final Path candidateFile = candidateFileList.get(i);
      hashingWorkerPool.execute(
          () -> {
            try {
              if (cancellationRequested.get() || Thread.currentThread().isInterrupted()) {
                return;
              }
              long size = Files.size(candidateFile);
              // Delegate digest computation to the centrally provisioned hashing service.
              String contentDigestHex =
                  HashingServiceFactory.getSharedInstance().computeSha256HexDigest(candidateFile);
              String slashSeparatedRelativePath =
                  libraryRoot.relativize(candidateFile).toString().replace(File.separatorChar, '/');
              indexedResultsByPosition[index] =
                  new DiscoveredFile(
                      candidateFile, slashSeparatedRelativePath, size, contentDigestHex);
              // Atomically bump the shared completed-files counter and snapshot the new total.
              int completedCountSnapshot = completedFileCounter.incrementAndGet();
              if (listener != null && !cancellationRequested.get()) {
                try {
                  if (listener.reportProgress(
                      completedCountSnapshot,
                      totalFileCount,
                      candidateFile.getFileName().toString())) {
                    cancellationRequested.set(true);
                  }
                } catch (RuntimeException e) {
                  // Progress reporting must never kill the scan.
                  cancellationRequested.set(true);
                }
              }
            } catch (IOException e) {
              if (!cancellationRequested.get()
                  && !Thread.currentThread().isInterrupted()
                  && listener != null) {
                try {
                  listener.reportFailure(candidateFile, e);
                } catch (RuntimeException ignored) {
                  // Error reporting must never kill the scan either.
                }
              }
            } finally {
              completionLatch.countDown();
            }
          });
    }
    try {
      completionLatch.await();
    } catch (InterruptedException e) {
      cancellationRequested.set(true);
      Thread.currentThread().interrupt();
      throw new IOException("Scan cancelled", e);
    }

    List<DiscoveredFile> result = new ArrayList<>(totalFileCount);
    for (DiscoveredFile scanned : indexedResultsByPosition) {
      if (scanned != null) {
        result.add(scanned);
      }
    }
    return result;
  }
}
