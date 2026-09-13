package com.clankerprise.sorting;

import com.clankerprise.sorting.presentation.PrimaryApplicationFrame;
import com.clankerprise.sorting.presentation.typography.EmbeddedTypographyService;
import java.awt.Color;
import java.awt.Font;
import java.awt.Graphics2D;
import java.awt.GraphicsEnvironment;
import java.awt.image.BufferedImage;
import java.security.MessageDigest;
import javax.swing.SwingUtilities;
import javax.swing.UIManager;

/** Entry point. Native look-and-feel, bundled Lato UI font, AA text on Linux. */
public final class ClankerSortApplication {

  private ClankerSortApplication() {}

  public static void main(String[] args) {
    boolean diagnosticsModeRequested = false;
    for (String commandLineArgument : args) {
      if ("--diagnose-fonts".equals(commandLineArgument)) {
        diagnosticsModeRequested = true;
      }
    }
    // Must precede any AWT init to take effect. Respects explicit overrides.
    setSystemPropertyIfAbsent("awt.useSystemAAFontSettings", "on");
    setSystemPropertyIfAbsent("swing.aatext", "true");
    if (diagnosticsModeRequested) {
      runHeadlessFontDiagnostics();
      return;
    }
    if (GraphicsEnvironment.isHeadless()) {
      System.err.println(
          "Clanker Sort needs a display. Run on a machine with a GUI, or use "
              + "the test task (mise run test) for headless verification.");
      System.exit(1);
    }
    // Register bundled fonts before the look-and-feel reads theme fonts.
    EmbeddedTypographyService.registerEmbeddedTypefaces();
    try {
      UIManager.setLookAndFeel(UIManager.getSystemLookAndFeelClassName());
    } catch (Exception caughtInitializationError) {
      System.err.println(
          "Could not set native look-and-feel, using default: "
              + caughtInitializationError.getMessage());
    }
    String resolvedTypographyFamilyName = EmbeddedTypographyService.initializeTypography();
    if (resolvedTypographyFamilyName == null) {
      System.err.println("Bundled font not found, using system fonts.");
    } else {
      System.out.println("Clanker Sort: UI font = " + resolvedTypographyFamilyName);
      EmbeddedTypographyService.installAutomaticWindowStylingHook();
    }
    SwingUtilities.invokeLater(
        () -> {
          PrimaryApplicationFrame primaryApplicationFrame = new PrimaryApplicationFrame();
          EmbeddedTypographyService.applyToComponentHierarchy(primaryApplicationFrame);
          primaryApplicationFrame.setVisible(true);
        });
  }

  private static void setSystemPropertyIfAbsent(String key, String value) {
    if (System.getProperty(key) == null) {
      System.setProperty(key, value);
    }
  }

  /**
   * Headless font diagnostics: proves whether Lato loads AND rasterizes (renders text to an
   * off-screen image and hashes the rasterPixelData), without opening any window. Run via {@code
   * mise run diagnose-fonts}.
   */
  private static void runHeadlessFontDiagnostics() {
    System.out.println(
        "java="
            + System.getProperty("java.version")
            + " vendor="
            + System.getProperty("java.vendor"));
    System.out.println(
        "os=" + System.getProperty("os.name") + " headless=" + GraphicsEnvironment.isHeadless());
    System.out.println(
        "awt.useSystemAAFontSettings=" + System.getProperty("awt.useSystemAAFontSettings"));
    System.out.println("swing.aatext=" + System.getProperty("swing.aatext"));
    System.out.println(
        "PRE-apply Label.font="
            + javax.swing.UIManager.getFont("Label.font")
            + " Button.font="
            + javax.swing.UIManager.getFont("Button.font"));
    String registeredTypefaceFamily = EmbeddedTypographyService.initializeTypography();
    System.out.println("registered family=" + registeredTypefaceFamily);
    boolean familyListingConfirmed = false;
    for (String name :
        GraphicsEnvironment.getLocalGraphicsEnvironment().getAvailableFontFamilyNames()) {
      if (name.equalsIgnoreCase("Lato")) {
        familyListingConfirmed = true;
        break;
      }
    }
    System.out.println("Lato in available families=" + familyListingConfirmed);
    Font typefaceProbeFont = new Font("Lato", Font.PLAIN, 12);
    System.out.println(
        "new Font(Lato): family="
            + typefaceProbeFont.getFamily()
            + " fontName="
            + typefaceProbeFont.getFontName());
    javax.swing.JLabel diagnosticSampleLabel = new javax.swing.JLabel("Ag");
    System.out.println("Label.font default: style=" + javax.swing.UIManager.getFont("Label.font"));
    System.out.println(
        "JLabel before style: family="
            + diagnosticSampleLabel.getFont().getFamily()
            + " fontName="
            + diagnosticSampleLabel.getFont().getFontName()
            + " style="
            + diagnosticSampleLabel.getFont().getStyle());
    EmbeddedTypographyService.applyToComponentHierarchy(diagnosticSampleLabel);
    System.out.println(
        "styled JLabel: family="
            + diagnosticSampleLabel.getFont().getFamily()
            + " fontName="
            + diagnosticSampleLabel.getFont().getFontName());
    System.out.println(
        "rasterPixelData(Lato)="
            + computeRenderedTextFingerprint("Lato")
            + " rasterPixelData(DejaVu Sans)="
            + computeRenderedTextFingerprint("DejaVu Sans"));
    System.out.println("diagnosticsModeRequested done");
  }

  private static String computeRenderedTextFingerprint(String familyName) {
    try {
      BufferedImage renderingSurface = new BufferedImage(320, 80, BufferedImage.TYPE_INT_ARGB);
      Graphics2D twoDimensionalGraphics = renderingSurface.createGraphics();
      twoDimensionalGraphics.setColor(Color.WHITE);
      twoDimensionalGraphics.fillRect(0, 0, 320, 80);
      twoDimensionalGraphics.setColor(Color.BLACK);
      twoDimensionalGraphics.setFont(new Font(familyName, Font.PLAIN, 48));
      twoDimensionalGraphics.drawString("Agfi 0123", 10, 58);
      twoDimensionalGraphics.dispose();
      MessageDigest digestComputer = MessageDigest.getInstance("SHA-256");
      int[] rasterPixelData = renderingSurface.getRGB(0, 0, 320, 80, null, 0, 320);
      java.nio.ByteBuffer pixelByteBuffer =
          java.nio.ByteBuffer.allocate(rasterPixelData.length * 4);
      pixelByteBuffer.asIntBuffer().put(rasterPixelData);
      byte[] digest = digestComputer.digest(pixelByteBuffer.array());
      return java.util.HexFormat.of().formatHex(digest).substring(0, 16);
    } catch (Exception caughtInitializationError) {
      return "ERROR:" + caughtInitializationError.getMessage();
    }
  }
}
