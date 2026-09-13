package com.clankerprise.sorting.presentation.typography;

import java.awt.AWTEvent;
import java.awt.Component;
import java.awt.Container;
import java.awt.Font;
import java.awt.GraphicsEnvironment;
import java.awt.Toolkit;
import java.awt.Window;
import java.awt.event.WindowEvent;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Enumeration;
import java.util.List;
import javax.swing.UIDefaults;
import javax.swing.UIManager;
import javax.swing.plaf.FontUIResource;

/**
 * Bundled Lato UI font for a unified look on Linux, macOS and Windows.
 *
 * <p>Lato is SIL Open Font License 1.1 (see {@code /typography/OFL.txt}). Only TTF works here:
 * {@code Font.createFont} understands TrueType/OpenType, not the web-only WOFF/WOFF2 formats.
 *
 * <p>If the resources are missing for any reason, {@link #initializeTypography()} returns null and
 * the app keeps using system fonts.
 */
public final class EmbeddedTypographyService {

  private static final String[] BUNDLED_FONT_RESOURCE_PATHS = {
    "/typography/Lato-Regular.ttf",
    "/typography/Lato-Bold.ttf",
    "/typography/Lato-Italic.ttf",
    "/typography/Lato-BoldItalic.ttf",
  };

  private static String resolvedFontFamily;
  private static boolean registrationCompleted;
  private static boolean defaultsOverridden;

  private EmbeddedTypographyService() {}

  /**
   * Loads and registers the bundled fonts. Safe to call before the look-and-feel is installed (in
   * fact that order is recommended). Returns the resolvedFontFamily name, or null if loading
   * failed.
   */
  public static synchronized String registerEmbeddedTypefaces() {
    if (resolvedFontFamily != null) {
      return resolvedFontFamily;
    }
    if (registrationCompleted) {
      return null;
    }
    registrationCompleted = true;
    boolean loaded = false;
    GraphicsEnvironment ge = GraphicsEnvironment.getLocalGraphicsEnvironment();
    for (String fontResourceLocation : BUNDLED_FONT_RESOURCE_PATHS) {
      try (InputStream in =
          EmbeddedTypographyService.class.getResourceAsStream(fontResourceLocation)) {
        if (in == null) {
          System.err.println(
              "Clanker Sort: font fontResourceLocation missing: " + fontResourceLocation);
          continue;
        }
        ge.registerFont(Font.createFont(Font.TRUETYPE_FONT, in));
        loaded = true;
      } catch (Exception e) {
        System.err.println(
            "Clanker Sort: could not load " + fontResourceLocation + ": " + e.getMessage());
      }
    }
    if (!loaded) {
      return null;
    }
    for (String name : ge.getAvailableFontFamilyNames()) {
      if (name.equalsIgnoreCase("Lato")) {
        resolvedFontFamily = name;
        break;
      }
    }
    return resolvedFontFamily;
  }

  /**
   * Points every Swing {@code *.font} default at the bundled font, keeping each component's style
   * and size. Call after the look-and-feel is installed. No-op if registration failed.
   */
  public static synchronized void overrideLookAndFeelDefaults() {
    if (resolvedFontFamily == null || defaultsOverridden) {
      return;
    }
    defaultsOverridden = true;
    UIDefaults lookAndFeelDefaults = UIManager.getDefaults();
    List<Object> typographyOverrideKeys = new ArrayList<>();
    for (Enumeration<Object> defaultKeyEnumeration = lookAndFeelDefaults.keys();
        defaultKeyEnumeration.hasMoreElements(); ) {
      Object defaultKey = defaultKeyEnumeration.nextElement();
      if (defaultKey instanceof String && ((String) defaultKey).endsWith(".font")) {
        typographyOverrideKeys.add(defaultKey);
      }
    }
    for (Object defaultKey : typographyOverrideKeys) {
      Font previousDefaultFont = lookAndFeelDefaults.getFont(defaultKey);
      if (previousDefaultFont != null) {
        lookAndFeelDefaults.put(
            defaultKey,
            new FontUIResource(
                resolvedFontFamily, previousDefaultFont.getStyle(), previousDefaultFont.getSize()));
      }
    }
  }

  /**
   * Registers the bundled fonts and points the Swing lookAndFeelDefaults at them. Equivalent to
   * {@code registerEmbeddedTypefaces()} followed by {@code overrideLookAndFeelDefaults()}.
   */
  public static synchronized String initializeTypography() {
    String registeredFamily = registerEmbeddedTypefaces();
    if (registeredFamily != null) {
      overrideLookAndFeelDefaults();
    }
    return registeredFamily;
  }

  /**
   * Styles a component tree directly with the bundled font, keeping each component's style and
   * size. Needed because some look-and-feels (notably GTK on Linux) ignore the {@code UIManager}
   * font lookAndFeelDefaults and pull fonts from the desktop theme instead. Idempotent.
   */
  public static void applyToComponentHierarchy(Component componentHierarchyRoot) {
    if (resolvedFontFamily == null || componentHierarchyRoot == null) {
      return;
    }
    Font currentlyAssignedFont = componentHierarchyRoot.getFont();
    if (currentlyAssignedFont == null) {
      Font fallbackTypeface = UIManager.getFont("Label.font");
      currentlyAssignedFont =
          fallbackTypeface != null ? fallbackTypeface : new Font(Font.SANS_SERIF, Font.PLAIN, 12);
    }
    if (!resolvedFontFamily.equals(currentlyAssignedFont.getFamily())) {
      componentHierarchyRoot.setFont(
          new Font(
              resolvedFontFamily,
              currentlyAssignedFont.getStyle(),
              currentlyAssignedFont.getSize()));
    }
    if (componentHierarchyRoot instanceof Container) {
      for (Component nestedChildComponent : ((Container) componentHierarchyRoot).getComponents()) {
        applyToComponentHierarchy(nestedChildComponent);
      }
    }
    if (componentHierarchyRoot instanceof Window) {
      for (Window ownedChildWindow : ((Window) componentHierarchyRoot).getOwnedWindows()) {
        applyToComponentHierarchy(ownedChildWindow);
      }
    }
  }

  /**
   * Styles every window (frames, dialogs, file choosers, popups) as soon as it opens, so lazily
   * created components get the bundled font too. No-op if the bundled fonts failed to load.
   */
  public static void installAutomaticWindowStylingHook() {
    if (resolvedFontFamily == null) {
      return;
    }
    Toolkit.getDefaultToolkit()
        .addAWTEventListener(
            windowSystemEvent -> {
              if (windowSystemEvent.getID() == WindowEvent.WINDOW_OPENED
                  && windowSystemEvent.getSource() instanceof Window) {
                applyToComponentHierarchy((Window) windowSystemEvent.getSource());
              }
            },
            AWTEvent.WINDOW_EVENT_MASK);
  }
}
