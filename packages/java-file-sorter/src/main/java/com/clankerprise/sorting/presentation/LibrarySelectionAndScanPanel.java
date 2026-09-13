package com.clankerprise.sorting.presentation;

import com.clankerprise.sorting.discovery.FileDiscoveryService;
import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.domain.TagRepository;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.awt.BorderLayout;
import java.awt.FlowLayout;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.List;
import javax.swing.BorderFactory;
import javax.swing.JButton;
import javax.swing.JFileChooser;
import javax.swing.JLabel;
import javax.swing.JOptionPane;
import javax.swing.JPanel;
import javax.swing.JProgressBar;
import javax.swing.JScrollPane;
import javax.swing.JTextArea;
import javax.swing.JTextField;
import javax.swing.SwingWorker;

/** Screen 1: pick a folder and scan (hash) every file in it. */
@SuppressWarnings("serial") // Swing components; never serialized.
public final class LibrarySelectionAndScanPanel extends JPanel {

  private final PrimaryApplicationFrame frame;
  private final ApplicationSessionState state;
  private final JTextField directoryPathField = new JTextField(40);
  private final JButton scanControlButton = new JButton("Scan");
  private final JProgressBar scanProgressIndicator = new JProgressBar();
  private final JTextArea scanLogArea = new JTextArea(12, 60);
  private final JLabel statusMessageLabel = new JLabel("Choose a directory, then hit Scan.");
  private SwingWorker<Void, String> backgroundScanWorker;

  public LibrarySelectionAndScanPanel(
      PrimaryApplicationFrame frame, ApplicationSessionState state) {
    super(new BorderLayout(8, 8));
    this.frame = frame;
    this.state = state;
    setBorder(BorderFactory.createEmptyBorder(12, 12, 12, 12));

    JLabel screenTitleLabel = new JLabel("Clanker Sort - 1. Browse and scan");
    screenTitleLabel.setFont(screenTitleLabel.getFont().deriveFont(java.awt.Font.BOLD, 16f));
    add(screenTitleLabel, BorderLayout.NORTH);

    JPanel centralContentPanel = new JPanel(new BorderLayout(8, 8));
    JPanel inputRowPanel = new JPanel(new FlowLayout(FlowLayout.LEFT));
    directoryPathField.setText(System.getProperty("user.home"));
    JButton browseButton = new JButton("Browse...");
    browseButton.addActionListener(userInterfaceEvent -> promptForDirectorySelection());
    scanControlButton.addActionListener(userInterfaceEvent -> initiateScanProcedure());
    inputRowPanel.add(new JLabel("Folder:"));
    inputRowPanel.add(directoryPathField);
    inputRowPanel.add(browseButton);
    inputRowPanel.add(scanControlButton);
    centralContentPanel.add(inputRowPanel, BorderLayout.NORTH);

    scanLogArea.setEditable(false);
    scanLogArea.setLineWrap(true);
    centralContentPanel.add(new JScrollPane(scanLogArea), BorderLayout.CENTER);

    JPanel statusStripPanel = new JPanel(new BorderLayout(8, 8));
    scanProgressIndicator.setStringPainted(true);
    statusStripPanel.add(scanProgressIndicator, BorderLayout.CENTER);
    statusStripPanel.add(statusMessageLabel, BorderLayout.SOUTH);
    centralContentPanel.add(statusStripPanel, BorderLayout.SOUTH);

    add(centralContentPanel, BorderLayout.CENTER);
  }

  void prepareForDisplay() {
    if (state.activeLibraryRoot != null) {
      directoryPathField.setText(state.activeLibraryRoot.toString());
    }
  }

  private void promptForDirectorySelection() {
    JFileChooser directoryChooserDialog = new JFileChooser(directoryPathField.getText().trim());
    directoryChooserDialog.setFileSelectionMode(JFileChooser.DIRECTORIES_ONLY);
    directoryChooserDialog.setDialogTitle("Choose folder to sort");
    if (directoryChooserDialog.showOpenDialog(this) == JFileChooser.APPROVE_OPTION) {
      directoryPathField.setText(directoryChooserDialog.getSelectedFile().getAbsolutePath());
    }
  }

  private void initiateScanProcedure() {
    if (backgroundScanWorker != null && !backgroundScanWorker.isDone()) {
      backgroundScanWorker.cancel(true);
      return;
    }
    Path selectedLibraryRoot = Paths.get(directoryPathField.getText().trim());
    if (!Files.isDirectory(selectedLibraryRoot)) {
      JOptionPane.showMessageDialog(
          this,
          "Not a directory: " + selectedLibraryRoot,
          "Clanker Sort",
          JOptionPane.ERROR_MESSAGE);
      return;
    }
    state.activeLibraryRoot = selectedLibraryRoot.toAbsolutePath().normalize();
    scanControlButton.setText("Cancel");
    scanLogArea.setText("");
    scanProgressIndicator.setValue(0);
    statusMessageLabel.setText("Scanning...");

    backgroundScanWorker =
        new SwingWorker<Void, String>() {
          private List<FileDiscoveryService.DiscoveredFile> discoveredFiles;

          @Override
          protected Void doInBackground() throws Exception {
            TagRepository previouslyPersistedRepository =
                JsonFileIndexRepository.hydrateRepository(state.activeLibraryRoot);
            publish(
                "Found previouslyPersistedRepository index with "
                    + previouslyPersistedRepository.countTotalEntries()
                    + " file(discovered).");
            discoveredFiles =
                FileDiscoveryService.performScan(
                    state.activeLibraryRoot,
                    new FileDiscoveryService.DiscoveryProgressListener() {
                      @Override
                      public boolean reportProgress(int done, int total, String name) {
                        setProgress(total == 0 ? 100 : (100 * done) / total);
                        publish("[" + done + "/" + total + "] " + name);
                        return isCancelled();
                      }

                      @Override
                      public void reportFailure(
                          java.nio.file.Path file, java.io.IOException error) {
                        publish(
                            "Skipped unreadable file: "
                                + file.getFileName()
                                + " ("
                                + error.getMessage()
                                + ")");
                      }
                    });
            if (isCancelled()) {
              return null;
            }
            TagRepository freshTagRepository = new TagRepository(state.activeLibraryRoot);
            int kept = 0;
            for (FileDiscoveryService.DiscoveredFile discovered : discoveredFiles) {
              // Every physical file gets its own entry (duplicates included);
              // tags are inherited from a previous entry with the same hash.
              String name = discovered.absolutePath.getFileName().toString();
              FileTagDescriptor entry =
                  new FileTagDescriptor(
                      discovered.contentHash,
                      name,
                      discovered.relativeSlashPath,
                      discovered.relativeSlashPath,
                      discovered.fileSizeInBytes,
                      discovered.contentHash);
              FileTagDescriptor template =
                  previouslyPersistedRepository.findFirstEntryByContentId(discovered.contentHash);
              if (template != null) {
                entry.restoreTagSnapshot(template.retrieveTagSnapshot());
                kept++;
              }
              freshTagRepository.registerEntry(entry);
            }
            // Drop stale entries whose storedFilesystemLocation file vanished from disk.
            for (FileTagDescriptor staleIndexedEntry :
                previouslyPersistedRepository.retrieveAllEntries()) {
              boolean matchingHashEncountered = false;
              for (FileDiscoveryService.DiscoveredFile discovered : discoveredFiles) {
                if (discovered.contentHash.equals(staleIndexedEntry.getContentIdentifier())) {
                  matchingHashEncountered = true;
                  break;
                }
              }
              if (!matchingHashEncountered) {
                Path storedFilesystemLocation =
                    state.activeLibraryRoot.resolve(
                        staleIndexedEntry
                            .getStoredRelativePath()
                            .replace('/', java.io.File.separatorChar));
                if (java.nio.file.Files.isRegularFile(storedFilesystemLocation)) {
                  freshTagRepository.registerEntry(
                      staleIndexedEntry); // already sorted into .files/, keep tags
                }
              }
            }
            JsonFileIndexRepository.persistRepository(freshTagRepository);
            state.tagRepository = freshTagRepository;
            state.currentTagPosition = 0;
            publish(
                "Scan complete: "
                    + freshTagRepository.countTotalEntries()
                    + " file(discovered), kept tags for "
                    + kept
                    + ".");
            return null;
          }

          @Override
          protected void process(List<String> publishedProgressMessages) {
            for (String msg : publishedProgressMessages) {
              scanLogArea.append(msg + "\n");
            }
            scanProgressIndicator.setValue(getProgress());
            scanLogArea.setCaretPosition(scanLogArea.getDocument().getLength());
          }

          @Override
          protected void done() {
            scanControlButton.setText("Scan");
            scanProgressIndicator.setValue(100);
            if (isCancelled()) {
              statusMessageLabel.setText("Scan cancelled.");
              scanLogArea.append("Cancelled.\n");
              return;
            }
            try {
              get(); // rethrow failures
            } catch (Exception userInterfaceEvent) {
              if (userInterfaceEvent instanceof InterruptedException) {
                Thread.currentThread().interrupt();
              }
              statusMessageLabel.setText("Scan failed.");
              scanLogArea.append("FAILED: " + extractRootCauseMessage(userInterfaceEvent) + "\n");
              return;
            }
            if (state.tagRepository == null || state.tagRepository.countTotalEntries() == 0) {
              statusMessageLabel.setText("No files discoveredFiles.");
              JOptionPane.showMessageDialog(
                  LibrarySelectionAndScanPanel.this,
                  "No scannable files discoveredFiles in that folder.",
                  "Clanker Sort",
                  JOptionPane.INFORMATION_MESSAGE);
              return;
            }
            statusMessageLabel.setText("Scan done.");
            frame.navigateToTaggingScreen();
          }
        };
    backgroundScanWorker.execute();
  }

  private static String extractRootCauseMessage(Exception userInterfaceEvent) {
    Throwable t = userInterfaceEvent;
    while (t.getCause() != null) {
      t = t.getCause();
    }
    return t.getMessage() == null ? t.toString() : t.getMessage();
  }
}
