/**
 * Parallel file discovery subsystem. Recursively walks library folders and computes content hashes
 * on a managed worker pool, delivering deterministic, stably ordered results to every downstream
 * consumer regardless of thread scheduling vagaries.
 */
package com.clankerprise.sorting.discovery;
