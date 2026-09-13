package com.clankerprise.sorting.infrastructure;

import java.awt.Desktop;
import java.io.IOException;
import java.nio.file.Path;
import java.util.Locale;

/** Opens a file with the platform default application. */
public final class ExternalApplicationLauncher {

  private ExternalApplicationLauncher() {}

  public static void launchWithDefaultApplication(Path file) throws IOException {
    if (Desktop.isDesktopSupported()) {
      Desktop desktop = Desktop.getDesktop();
      if (desktop.isSupported(Desktop.Action.OPEN)) {
        desktop.open(file.toFile());
        return;
      }
    }
    String os = System.getProperty("os.name", "").toLowerCase(Locale.ROOT);
    ProcessBuilder pb;
    if (os.contains("mac")) {
      pb = new ProcessBuilder("open", file.toString());
    } else if (os.contains("win")) {
      pb = new ProcessBuilder("rundll32", "url.dll,FileProtocolHandler", file.toString());
    } else {
      pb = new ProcessBuilder("xdg-open", file.toString());
    }
    pb.start();
  }
}
