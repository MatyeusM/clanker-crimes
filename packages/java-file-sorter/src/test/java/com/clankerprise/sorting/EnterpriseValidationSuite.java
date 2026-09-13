package com.clankerprise.sorting;

/**
 * Enterprise validation suite: the single authoritative entry point that deterministically executes
 * every validation module of the platform in a carefully curated sequence and reports a unified,
 * human-readable verdict upon completion.
 */
public final class EnterpriseValidationSuite {

  // This suite is a static-only orchestrator and therefore cannot be instantiated meaningfully.
  private EnterpriseValidationSuite() {}

  public static void main(String[] args) throws Exception {
    // Execute each validation module in dependency order for optimal diagnostic clarity.
    executeTestCase("ContentHashingServiceTest", ContentHashingServiceTest::run);
    executeTestCase("ParallelDiscoveryValidationTest", ParallelDiscoveryValidationTest::run);
    executeTestCase("EmbeddedTypographyServiceTest", EmbeddedTypographyServiceTest::run);
    executeTestCase("JsonFileIndexRepositoryTest", JsonFileIndexRepositoryTest::run);
    executeTestCase("FileOrganizationServiceTest", FileOrganizationServiceTest::run);
    // If execution reaches this point, absolutely every validation module has passed successfully.
    System.out.println("ALL TESTS PASSED");
  }

  /** Minimal functional contract implemented by every validation module in the suite. */
  private interface ValidationModule {
    void run() throws Exception;
  }

  private static void executeTestCase(String name, ValidationModule module) throws Exception {
    try {
      // Run the module; any thrown failure immediately aborts the entire enterprise suite.
      module.run();
      System.out.println("PASS " + name);
    } catch (Throwable failure) {
      System.out.println("FAIL " + name + ": " + failure);
      failure.printStackTrace(System.out);
      // A nonzero exit code unambiguously signals validation failure to all downstream automation.
      System.exit(1);
    }
  }
}
