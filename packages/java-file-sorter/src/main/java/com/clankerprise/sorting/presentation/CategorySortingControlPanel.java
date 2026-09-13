package com.clankerprise.sorting.presentation;

import com.clankerprise.sorting.organization.FileOrganizationService;
import java.awt.BorderLayout;
import java.awt.FlowLayout;
import java.util.List;
import javax.swing.BorderFactory;
import javax.swing.JButton;
import javax.swing.JComboBox;
import javax.swing.JLabel;
import javax.swing.JOptionPane;
import javax.swing.JPanel;
import javax.swing.JScrollPane;
import javax.swing.JTextArea;
import javax.swing.SwingWorker;

/** Screen 3: pick a tag targetSortCategory and sort (move to .files/ + link views). */
@SuppressWarnings("serial") // Swing components; never serialized.
public final class CategorySortingControlPanel extends JPanel {

  private final PrimaryApplicationFrame frame;
  private final ApplicationSessionState state;
  private final JComboBox<String> sortCategorySelector = new JComboBox<>();
  private final JButton executeSortButton = new JButton("Sort Now");
  private final JTextArea sortProgressLog = new JTextArea(14, 60);
  private final JLabel explanatoryHintLabel = new JLabel();

  public CategorySortingControlPanel(PrimaryApplicationFrame frame, ApplicationSessionState state) {
    super(new BorderLayout(8, 8));
    this.frame = frame;
    this.state = state;
    setBorder(BorderFactory.createEmptyBorder(12, 12, 12, 12));

    JLabel screenTitleLabel = new JLabel("Clanker Sort - 3. Sort by tag");
    screenTitleLabel.setFont(screenTitleLabel.getFont().deriveFont(java.awt.Font.BOLD, 16f));
    add(screenTitleLabel, BorderLayout.NORTH);

    JPanel centralContentPanel = new JPanel(new BorderLayout(8, 8));
    JPanel categorySelectionRow = new JPanel(new FlowLayout(FlowLayout.LEFT));
    executeSortButton.addActionListener(userInterfaceEvent -> initiateSortProcedure());
    categorySelectionRow.add(new JLabel("Tag targetSortCategory:"));
    categorySelectionRow.add(sortCategorySelector);
    categorySelectionRow.add(executeSortButton);
    centralContentPanel.add(categorySelectionRow, BorderLayout.NORTH);
    centralContentPanel.add(explanatoryHintLabel, BorderLayout.CENTER);

    sortProgressLog.setEditable(false);
    sortProgressLog.setLineWrap(true);
    JPanel progressLogSection = new JPanel(new BorderLayout());
    progressLogSection.setBorder(BorderFactory.createTitledBorder("Progress"));
    progressLogSection.add(new JScrollPane(sortProgressLog), BorderLayout.CENTER);
    centralContentPanel.add(progressLogSection, BorderLayout.SOUTH);
    add(centralContentPanel, BorderLayout.CENTER);

    JPanel nav = new JPanel(new FlowLayout(FlowLayout.CENTER, 10, 5));
    JButton returnToTaggingButton = new JButton("Back to tagging");
    returnToTaggingButton.addActionListener(userInterfaceEvent -> frame.navigateToTaggingScreen());
    JButton chooseDifferentFolderButton = new JButton("Choose different folder");
    chooseDifferentFolderButton.addActionListener(
        userInterfaceEvent -> frame.navigateToBrowseScreen());
    nav.add(returnToTaggingButton);
    nav.add(chooseDifferentFolderButton);
    add(nav, BorderLayout.SOUTH);
  }

  void prepareForDisplay() {
    sortProgressLog.setText("");
    sortCategorySelector.removeAllItems();
    if (state.tagRepository != null) {
      for (String cat : state.tagRepository.collectAllCategories()) {
        sortCategorySelector.addItem(cat);
      }
      if (state.tagRepository.retrieveLastSortCategory() != null
          && !state.tagRepository.retrieveLastSortCategory().isEmpty()) {
        sortCategorySelector.setSelectedItem(state.tagRepository.retrieveLastSortCategory());
      }
      int tagged = state.tagRepository.countTaggedEntries();
      int total = state.tagRepository.countTotalEntries();
      explanatoryHintLabel.setText(
          "<html>Tagged "
              + tagged
              + " of "
              + total
              + " file(s). "
              + "Sorting moves originals into the hidden <b>.files/</b> folder and creates "
              + "one folder per tag value with links back to the real files. "
              + "Untagged files land in <b>Unsorted/</b>.<br><b>Warning:</b> everything outside "
              + "<b>.files/</b> and <b>.index.json</b> is deleted — scan first so every file is "
              + "indexed.</html>");
      executeSortButton.setEnabled(total > 0 && sortCategorySelector.getItemCount() > 0);
    }
  }

  private void initiateSortProcedure() {
    Object chosenCategorySelection = sortCategorySelector.getSelectedItem();
    if (chosenCategorySelection == null || state.tagRepository == null) {
      JOptionPane.showMessageDialog(
          this, "Tag at least one file first.", "Clanker Sort", JOptionPane.WARNING_MESSAGE);
      return;
    }
    String targetSortCategory = chosenCategorySelection.toString();
    executeSortButton.setEnabled(false);
    sortProgressLog.append(
        "Sorting '" + state.activeLibraryRoot + "' by '" + targetSortCategory + "'...\n");

    SwingWorker<FileOrganizationService.OrganizationOutcome, String> backgroundSortWorker =
        new SwingWorker<FileOrganizationService.OrganizationOutcome, String>() {
          @Override
          protected FileOrganizationService.OrganizationOutcome doInBackground() throws Exception {
            return FileOrganizationService.executeOrganization(
                state.tagRepository, targetSortCategory, this::publish);
          }

          @Override
          protected void process(List<String> publishedLogMessages) {
            for (String msg : publishedLogMessages) {
              sortProgressLog.append(msg + "\n");
            }
            sortProgressLog.setCaretPosition(sortProgressLog.getDocument().getLength());
          }

          @Override
          protected void done() {
            executeSortButton.setEnabled(true);
            try {
              FileOrganizationService.OrganizationOutcome organizationOutcome = get();
              sortProgressLog.append(
                  "Finished: "
                      + organizationOutcome.viewDirectoryCount
                      + " view folder(s), "
                      + organizationOutcome.relocatedCount
                      + " file(s) moved to .files/.\n");
              JOptionPane.showMessageDialog(
                  CategorySortingControlPanel.this,
                  "Sorted into " + organizationOutcome.viewDirectoryCount + " folder(s).",
                  "Clanker Sort",
                  JOptionPane.INFORMATION_MESSAGE);
            } catch (Exception userInterfaceEvent) {
              if (userInterfaceEvent instanceof InterruptedException) {
                Thread.currentThread().interrupt();
              }
              sortProgressLog.append(
                  "FAILED: " + extractRootCauseMessage(userInterfaceEvent) + "\n");
              JOptionPane.showMessageDialog(
                  CategorySortingControlPanel.this,
                  "Sorting failed: " + extractRootCauseMessage(userInterfaceEvent),
                  "Clanker Sort",
                  JOptionPane.ERROR_MESSAGE);
            }
          }
        };
    backgroundSortWorker.execute();
  }

  private static String extractRootCauseMessage(Exception userInterfaceEvent) {
    Throwable t = userInterfaceEvent;
    while (t.getCause() != null) {
      t = t.getCause();
    }
    return t.getMessage() == null ? t.toString() : t.getMessage();
  }
}
