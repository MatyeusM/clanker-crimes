package com.clankerprise.sorting;

import com.clankerprise.sorting.presentation.typography.EmbeddedTypographyService;
import java.awt.Font;
import java.io.InputStream;

/** Verifies the bundled Lato TTFs load (headless-safe, no UI shown). */
final class EmbeddedTypographyServiceTest {

  static void run() throws Exception {
    try (InputStream typefaceResourceStream =
        EmbeddedTypographyServiceTest.class.getResourceAsStream("/typography/Lato-Regular.ttf")) {
      ValidationAssertionHelper.assertTrue(
          typefaceResourceStream != null, "Lato-Regular.ttf on classpath");
      Font loadedTypeface = Font.createFont(Font.TRUETYPE_FONT, typefaceResourceStream);
      ValidationAssertionHelper.assertEqual(
          "Lato", loadedTypeface.getFamily(), "font resolvedFontFamily");
    }
    ValidationAssertionHelper.assertEqual(
        "Lato",
        EmbeddedTypographyService.initializeTypography(),
        "EmbeddedTypographyService.apply resolvedFontFamily");
    ValidationAssertionHelper.assertEqual(
        "Lato",
        EmbeddedTypographyService.initializeTypography(),
        "EmbeddedTypographyService.apply idempotent");

    // Recursive styling of a component tree (headless-safe, lightweight only).
    javax.swing.JPanel validationContainer = new javax.swing.JPanel(new java.awt.BorderLayout());
    javax.swing.JButton sampleButton = new javax.swing.JButton("Scan");
    sampleButton.setFont(sampleButton.getFont().deriveFont(java.awt.Font.BOLD));
    javax.swing.JLabel sampleLabel = new javax.swing.JLabel("hello");
    validationContainer.add(sampleButton, java.awt.BorderLayout.NORTH);
    validationContainer.add(sampleLabel, java.awt.BorderLayout.SOUTH);
    int originalPointSize = sampleButton.getFont().getSize();
    EmbeddedTypographyService.applyToComponentHierarchy(validationContainer);
    ValidationAssertionHelper.assertEqual(
        "Lato", sampleButton.getFont().getFamily(), "sampleButton resolvedFontFamily");
    ValidationAssertionHelper.assertEqual(
        "Lato", sampleLabel.getFont().getFamily(), "sampleLabel resolvedFontFamily");
    ValidationAssertionHelper.assertEqual(
        java.awt.Font.BOLD, sampleButton.getFont().getStyle(), "sampleButton style kept");
    ValidationAssertionHelper.assertEqual(
        originalPointSize, sampleButton.getFont().getSize(), "sampleButton size kept");
    EmbeddedTypographyService.applyToComponentHierarchy(
        validationContainer); // idempotent, no-op second time
    ValidationAssertionHelper.assertEqual(
        "Lato", sampleButton.getFont().getFamily(), "sampleButton resolvedFontFamily stable");
  }
}
