package com.clankerprise.sorting.presentation;

import com.clankerprise.sorting.domain.FileTagDescriptor;
import com.clankerprise.sorting.infrastructure.ExternalApplicationLauncher;
import com.clankerprise.sorting.persistence.JsonFileIndexRepository;
import java.awt.BorderLayout;
import java.awt.FlowLayout;
import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;
import javax.swing.BorderFactory;
import javax.swing.DefaultListModel;
import javax.swing.JButton;
import javax.swing.JComboBox;
import javax.swing.JLabel;
import javax.swing.JList;
import javax.swing.JOptionPane;
import javax.swing.JPanel;
import javax.swing.JScrollPane;
import javax.swing.JTextField;
import javax.swing.ListSelectionModel;

/** Screen 2: walk through files one at a time and attach tags. */
@SuppressWarnings("serial") // Swing components; never serialized.
public final class InteractiveTaggingWorkflowPanel extends JPanel {

  private final PrimaryApplicationFrame frame;
  private final ApplicationSessionState state;

  private final JLabel headerLabel = new JLabel();
  private final JLabel detailsLabel = new JLabel();
  private final JLabel progressCounterLabel = new JLabel();
  private final DefaultListModel<String> tagListContentsModel = new DefaultListModel<>();
  private final JList<String> tagSelectionList = new JList<>(tagListContentsModel);
  private final JComboBox<String> categorySelectionComboBox = new JComboBox<>();
  private final JTextField valueEntryField = new JTextField(20);
  private final JButton previousFileButton = new JButton("< Prev");
  private final JButton nextFileButton = new JButton("Next >");
  private final JButton proceedToSortButton = new JButton("Sort Now");

  public InteractiveTaggingWorkflowPanel(
      PrimaryApplicationFrame frame, ApplicationSessionState state) {
    super(new BorderLayout(8, 8));
    this.frame = frame;
    this.state = state;
    setBorder(BorderFactory.createEmptyBorder(12, 12, 12, 12));

    JLabel screenTitleLabel = new JLabel("Clanker Sort - 2. Tag your files");
    screenTitleLabel.setFont(screenTitleLabel.getFont().deriveFont(java.awt.Font.BOLD, 16f));
    add(screenTitleLabel, BorderLayout.NORTH);

    JPanel center = new JPanel(new BorderLayout(8, 8));
    JPanel currentFileInformationPanel = new JPanel(new BorderLayout());
    headerLabel.setFont(headerLabel.getFont().deriveFont(java.awt.Font.BOLD, 14f));
    currentFileInformationPanel.add(headerLabel, BorderLayout.NORTH);
    currentFileInformationPanel.add(detailsLabel, BorderLayout.CENTER);
    currentFileInformationPanel.add(progressCounterLabel, BorderLayout.SOUTH);
    center.add(currentFileInformationPanel, BorderLayout.NORTH);

    tagSelectionList.setSelectionMode(ListSelectionModel.SINGLE_SELECTION);
    tagSelectionList.setVisibleRowCount(6);
    JPanel tagDisplaySectionPanel = new JPanel(new BorderLayout(8, 8));
    tagDisplaySectionPanel.setBorder(
        BorderFactory.createTitledBorder("Tags (category = value1, value2, ...)"));
    tagDisplaySectionPanel.add(new JScrollPane(tagSelectionList), BorderLayout.CENTER);
    center.add(tagDisplaySectionPanel, BorderLayout.CENTER);

    JPanel tagEditorRowPanel = new JPanel(new FlowLayout(FlowLayout.LEFT));
    categorySelectionComboBox.setEditable(true);
    valueEntryField.setColumns(20);
    JButton commitTagButton = new JButton("Add tag");
    commitTagButton.addActionListener(userInterfaceEvent -> commitTagFromInput());
    valueEntryField.addActionListener(userInterfaceEvent -> commitTagFromInput());
    JButton removeTagButton = new JButton("Remove currentlySelectedRowContent");
    removeTagButton.addActionListener(userInterfaceEvent -> removeCurrentlySelectedTag());
    tagEditorRowPanel.add(new JLabel("Category:"));
    tagEditorRowPanel.add(categorySelectionComboBox);
    tagEditorRowPanel.add(new JLabel("Value:"));
    tagEditorRowPanel.add(valueEntryField);
    tagEditorRowPanel.add(commitTagButton);
    tagEditorRowPanel.add(removeTagButton);
    center.add(tagEditorRowPanel, BorderLayout.SOUTH);
    add(center, BorderLayout.CENTER);

    JPanel navigationButtonStrip = new JPanel(new FlowLayout(FlowLayout.CENTER, 10, 5));
    JButton openInExternalApplicationButton = new JButton("Open with default app");
    openInExternalApplicationButton.addActionListener(
        userInterfaceEvent -> openCurrentlyDisplayedFile());
    previousFileButton.addActionListener(userInterfaceEvent -> navigateByOffset(-1));
    nextFileButton.addActionListener(userInterfaceEvent -> navigateByOffset(1));
    JButton returnToUntaggedButton = new JButton("Back to first untagged");
    returnToUntaggedButton.addActionListener(userInterfaceEvent -> jumpToFirstUntaggedFile());
    proceedToSortButton.addActionListener(userInterfaceEvent -> frame.navigateToSortingScreen());
    JButton switchFolderButton = new JButton("Choose different folder");
    switchFolderButton.addActionListener(userInterfaceEvent -> frame.navigateToBrowseScreen());
    navigationButtonStrip.add(previousFileButton);
    navigationButtonStrip.add(openInExternalApplicationButton);
    navigationButtonStrip.add(nextFileButton);
    navigationButtonStrip.add(returnToUntaggedButton);
    navigationButtonStrip.add(proceedToSortButton);
    navigationButtonStrip.add(switchFolderButton);
    add(navigationButtonStrip, BorderLayout.SOUTH);
  }

  void prepareForDisplay() {
    refreshDisplayedContent();
  }

  private List<FileTagDescriptor> retrieveFileEntries() {
    return state.tagRepository == null ? List.of() : state.tagRepository.retrieveAllEntries();
  }

  private FileTagDescriptor displayedEntry() {
    List<FileTagDescriptor> completeEntryList = retrieveFileEntries();
    if (completeEntryList.isEmpty()) {
      return null;
    }
    state.currentTagPosition =
        Math.max(0, Math.min(state.currentTagPosition, completeEntryList.size() - 1));
    return completeEntryList.get(state.currentTagPosition);
  }

  private void refreshDisplayedContent() {
    List<FileTagDescriptor> completeEntryList = retrieveFileEntries();
    FileTagDescriptor currentlyDisplayedEntry = displayedEntry();
    if (currentlyDisplayedEntry == null) {
      headerLabel.setText("No files. Go back and scan a folder first.");
      detailsLabel.setText("");
      progressCounterLabel.setText("");
      tagListContentsModel.clear();
      updateNavigationAvailability(false);
      return;
    }
    updateNavigationAvailability(true);
    headerLabel.setText(
        "File "
            + (state.currentTagPosition + 1)
            + " of "
            + completeEntryList.size()
            + ": "
            + currentlyDisplayedEntry.getDisplayLabel());
    Path absoluteFilesystemPath =
        state.activeLibraryRoot.resolve(
            currentlyDisplayedEntry.getStoredRelativePath().replace('/', File.separatorChar));
    String contentDigestPreview = currentlyDisplayedEntry.getContentHash();
    detailsLabel.setText(
        "<html>Location: "
            + escape(absoluteFilesystemPath.toString())
            + "<br>Size: "
            + currentlyDisplayedEntry.getFileSizeInBytes()
            + " bytes &nbsp; SHA-256: "
            + (contentDigestPreview.length() > 16
                ? contentDigestPreview.substring(0, 16) + "..."
                : contentDigestPreview)
            + "</html>");
    int taggedEntryCount = state.tagRepository.countTaggedEntries();
    progressCounterLabel.setText(
        "Tagged "
            + taggedEntryCount
            + " of "
            + completeEntryList.size()
            + (taggedEntryCount == completeEntryList.size()
                ? " - completeEntryList done, you can hit Sort Now."
                : "."));

    tagListContentsModel.clear();
    for (Map.Entry<String, List<String>> userInterfaceEvent :
        currentlyDisplayedEntry.retrieveTagSnapshot().entrySet()) {
      tagListContentsModel.addElement(
          userInterfaceEvent.getKey() + " = " + String.join(", ", userInterfaceEvent.getValue()));
    }

    Object currentlySelectedRowContent = categorySelectionComboBox.getSelectedItem();
    categorySelectionComboBox.removeAllItems();
    for (String categoryName : state.tagRepository.collectAllCategories()) {
      categorySelectionComboBox.addItem(categoryName);
    }
    if (currentlySelectedRowContent != null) {
      categorySelectionComboBox.setSelectedItem(currentlySelectedRowContent.toString());
    }
    previousFileButton.setEnabled(state.currentTagPosition > 0);
    nextFileButton.setEnabled(state.currentTagPosition < completeEntryList.size() - 1);
  }

  private void updateNavigationAvailability(boolean enabled) {
    previousFileButton.setEnabled(enabled);
    nextFileButton.setEnabled(enabled);
    proceedToSortButton.setEnabled(enabled);
  }

  private void navigateByOffset(int delta) {
    // Move the current position forward (or backward) by exactly the requested offset amount.
    state.currentTagPosition += delta;
    refreshDisplayedContent();
  }

  private void jumpToFirstUntaggedFile() {
    List<FileTagDescriptor> completeEntryList = retrieveFileEntries();
    for (int i = 0; i < completeEntryList.size(); i++) {
      if (!completeEntryList.get(i).hasAssignedTags()) {
        state.currentTagPosition = i;
        refreshDisplayedContent();
        return;
      }
    }
    JOptionPane.showMessageDialog(
        this, "Every file has at least one tag.", "Clanker Sort", JOptionPane.INFORMATION_MESSAGE);
  }

  private void commitTagFromInput() {
    FileTagDescriptor currentlyDisplayedEntry = displayedEntry();
    if (currentlyDisplayedEntry == null) {
      return;
    }
    Object rawCategorySelection = categorySelectionComboBox.getSelectedItem();
    String category = rawCategorySelection == null ? "" : rawCategorySelection.toString().trim();
    String value = valueEntryField.getText().trim();
    if (category.isEmpty() || value.isEmpty()) {
      JOptionPane.showMessageDialog(
          this, "Enter both a category and a value.", "Clanker Sort", JOptionPane.WARNING_MESSAGE);
      return;
    }
    if (!currentlyDisplayedEntry.registerTagValue(category, value)) {
      JOptionPane.showMessageDialog(
          this, "Empty or duplicate tag.", "Clanker Sort", JOptionPane.WARNING_MESSAGE);
      return;
    }
    valueEntryField.setText("");
    persistChangesAndRefresh();
  }

  private void removeCurrentlySelectedTag() {
    FileTagDescriptor currentlyDisplayedEntry = displayedEntry();
    int selectedRowIndex = tagSelectionList.getSelectedIndex();
    if (currentlyDisplayedEntry == null || selectedRowIndex < 0) {
      return;
    }
    // Each row looks like "Category = v1, v2, ...".
    String currentlySelectedRowContent = tagListContentsModel.get(selectedRowIndex);
    int separatorPosition = currentlySelectedRowContent.indexOf(" = ");
    if (separatorPosition <= 0) {
      return;
    }
    String categoryName = currentlySelectedRowContent.substring(0, separatorPosition);
    String remainingValueText = currentlySelectedRowContent.substring(separatorPosition + 3);
    String[] selectableValueOptions = remainingValueText.split(",\\s*");
    if (selectableValueOptions.length == 1) {
      currentlyDisplayedEntry.deregisterTagValue(categoryName, selectableValueOptions[0]);
      persistChangesAndRefresh();
      return;
    }
    String userSelectedValue =
        (String)
            JOptionPane.showInputDialog(
                this,
                "Remove which value from '" + categoryName + "'?",
                "Clanker Sort",
                JOptionPane.QUESTION_MESSAGE,
                null,
                selectableValueOptions,
                selectableValueOptions[0]);
    if (userSelectedValue != null) {
      currentlyDisplayedEntry.deregisterTagValue(categoryName, userSelectedValue);
      persistChangesAndRefresh();
    }
  }

  private void persistChangesAndRefresh() {
    try {
      JsonFileIndexRepository.persistRepository(state.tagRepository);
    } catch (IOException userInterfaceEvent) {
      JOptionPane.showMessageDialog(
          this,
          "Could not save .index.json: " + userInterfaceEvent.getMessage(),
          "Clanker Sort",
          JOptionPane.ERROR_MESSAGE);
    }
    refreshDisplayedContent();
  }

  private void openCurrentlyDisplayedFile() {
    FileTagDescriptor currentlyDisplayedEntry = displayedEntry();
    if (currentlyDisplayedEntry == null) {
      return;
    }
    Path absoluteFilesystemPath =
        state.activeLibraryRoot.resolve(
            currentlyDisplayedEntry.getStoredRelativePath().replace('/', File.separatorChar));
    if (!Files.isRegularFile(absoluteFilesystemPath)) {
      JOptionPane.showMessageDialog(
          this,
          "File no longer exists:\n" + absoluteFilesystemPath,
          "Clanker Sort",
          JOptionPane.WARNING_MESSAGE);
      return;
    }
    new Thread(
            () -> {
              try {
                ExternalApplicationLauncher.launchWithDefaultApplication(absoluteFilesystemPath);
              } catch (IOException userInterfaceEvent) {
                javax.swing.SwingUtilities.invokeLater(
                    () ->
                        JOptionPane.showMessageDialog(
                            this,
                            "Could not open file: " + userInterfaceEvent.getMessage(),
                            "Clanker Sort",
                            JOptionPane.ERROR_MESSAGE));
              }
            },
            "opener")
        .start();
  }

  private static String escape(String s) {
    return s.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;");
  }
}
