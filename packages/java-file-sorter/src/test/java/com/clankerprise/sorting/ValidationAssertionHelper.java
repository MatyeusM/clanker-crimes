package com.clankerprise.sorting;

/** Tiny assertion helper so tests need no JUnit dependency. */
final class ValidationAssertionHelper {

  private ValidationAssertionHelper() {}

  static void assertEqual(Object expected, Object actual, String what) {
    if (expected == null ? actual != null : !expected.equals(actual)) {
      throw new AssertionError(what + ": expected <" + expected + "> but was <" + actual + ">");
    }
  }

  static void assertTrue(boolean condition, String what) {
    if (!condition) {
      throw new AssertionError(what + ": expected true");
    }
  }
}
