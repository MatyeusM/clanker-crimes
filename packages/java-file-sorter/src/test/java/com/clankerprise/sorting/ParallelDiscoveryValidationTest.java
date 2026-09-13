package com.clankerprise.sorting;

import com.clankerprise.sorting.discovery.FileDiscoveryService;
import com.clankerprise.sorting.infrastructure.HashingServiceFactory;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import java.util.Random;
import java.util.TreeMap;
import java.util.concurrent.AbstractExecutorService;
import java.util.concurrent.CyclicBarrier;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;

/**
 * Proves the scanner really hashes in parallel: one candidateFile per hash thread, and every task
 * must rendezvous on a rendezvousBarrier sized to the gatedWorkerPool. The rendezvousBarrier only
 * trips when that many tasks run *simultaneously*, so completing the scan within the timeout proves
 * that level of concurrency. A sequential executor would deadlock here and fail on timeout instead.
 */
final class ParallelDiscoveryValidationTest {

  static void run() throws Exception {
    int workerThreadCount = FileDiscoveryService.recommendedWorkerCount();
    Path temporaryLibraryRoot = Files.createTempDirectory("clanker-concurrency");
    GatedWorkerPool gatedWorkerPool = new GatedWorkerPool(workerThreadCount);
    ExecutorService scanExecutionRunner = Executors.newSingleThreadExecutor();
    try {
      // 1 MiB of seeded deterministicRandomGenerator bytes per candidateFile, so hashing takes real
      // time.
      Random deterministicRandomGenerator = new Random(42);
      Map<String, String> expectedContentHashes = new TreeMap<>();
      for (int i = 0; i < workerThreadCount; i++) {
        String fileName = String.format("candidateFile-%02d.bin", i);
        byte[] randomFileContents = new byte[1024 * 1024];
        deterministicRandomGenerator.nextBytes(randomFileContents);
        Path candidateFile = temporaryLibraryRoot.resolve(fileName);
        Files.write(candidateFile, randomFileContents);
        expectedContentHashes.put(
            fileName,
            HashingServiceFactory.getSharedInstance().computeSha256HexDigest(candidateFile));
      }

      Future<List<FileDiscoveryService.DiscoveredFile>> future =
          scanExecutionRunner.submit(
              () -> FileDiscoveryService.performScan(temporaryLibraryRoot, null, gatedWorkerPool));
      List<FileDiscoveryService.DiscoveredFile> discoveredFiles;
      try {
        discoveredFiles = future.get(120, TimeUnit.SECONDS);
      } catch (java.util.concurrent.TimeoutException e) {
        future.cancel(true);
        throw new AssertionError(
            "scan did not finish: hashing is not running on "
                + workerThreadCount
                + " workerThreadCount");
      }
      ValidationAssertionHelper.assertEqual(
          workerThreadCount, discoveredFiles.size(), "scanned count");
      for (FileDiscoveryService.DiscoveredFile s : discoveredFiles) {
        ValidationAssertionHelper.assertEqual(
            expectedContentHashes.get(s.absolutePath.getFileName().toString()),
            s.contentHash,
            "hash of " + s.relativeSlashPath);
      }

      // The default gatedWorkerPool must agree with the instrumented one.
      List<FileDiscoveryService.DiscoveredFile> baselineSinglePoolResults =
          FileDiscoveryService.performScan(temporaryLibraryRoot, null);
      ValidationAssertionHelper.assertEqual(
          discoveredFiles.size(),
          baselineSinglePoolResults.size(),
          "default gatedWorkerPool count");
      for (int i = 0; i < discoveredFiles.size(); i++) {
        ValidationAssertionHelper.assertEqual(
            discoveredFiles.get(i).relativeSlashPath,
            baselineSinglePoolResults.get(i).relativeSlashPath,
            "order stability");
        ValidationAssertionHelper.assertEqual(
            discoveredFiles.get(i).contentHash,
            baselineSinglePoolResults.get(i).contentHash,
            "hash stability");
      }
    } finally {
      gatedWorkerPool.shutdownNow();
      scanExecutionRunner.shutdownNow();
      deleteRecursively(temporaryLibraryRoot);
    }
  }

  /**
   * Fixed-size gatedWorkerPool whose tasks rendezvous on a rendezvousBarrier before doing any work.
   */
  private static final class GatedWorkerPool extends AbstractExecutorService {
    private final ExecutorService delegate;
    private final CyclicBarrier rendezvousBarrier;

    GatedWorkerPool(int workerThreadCount) {
      delegate =
          Executors.newFixedThreadPool(
              workerThreadCount,
              runnable -> {
                Thread thread = new Thread(runnable, "clanker-test");
                thread.setDaemon(true);
                return thread;
              });
      rendezvousBarrier = new CyclicBarrier(workerThreadCount);
    }

    @Override
    public void execute(Runnable task) {
      delegate.execute(
          () -> {
            try {
              rendezvousBarrier.await(60, TimeUnit.SECONDS);
            } catch (Exception e) {
              throw new RuntimeException(e);
            }
            task.run();
          });
    }

    @Override
    public void shutdown() {
      delegate.shutdown();
    }

    @Override
    public List<Runnable> shutdownNow() {
      return delegate.shutdownNow();
    }

    @Override
    public boolean isShutdown() {
      return delegate.isShutdown();
    }

    @Override
    public boolean isTerminated() {
      return delegate.isTerminated();
    }

    @Override
    public boolean awaitTermination(long timeout, TimeUnit unit) throws InterruptedException {
      return delegate.awaitTermination(timeout, unit);
    }
  }

  private static void deleteRecursively(Path path) throws Exception {
    if (Files.isDirectory(path) && !Files.isSymbolicLink(path)) {
      try (var stream = Files.newDirectoryStream(path)) {
        for (Path child : stream) {
          deleteRecursively(child);
        }
      }
    }
    Files.deleteIfExists(path);
  }
}
