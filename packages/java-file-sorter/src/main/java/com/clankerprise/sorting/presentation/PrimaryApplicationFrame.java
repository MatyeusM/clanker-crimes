package com.clankerprise.sorting.presentation;

import java.awt.CardLayout;
import javax.swing.JFrame;
import javax.swing.JPanel;

/** ClankerSortApplication window; swaps between the browse / tag / sort screens. */
@SuppressWarnings("serial") // Swing components; never serialized.
public final class PrimaryApplicationFrame extends JFrame {

  public static final String BROWSE_SCREEN = "browse";
  public static final String TAGGING_SCREEN = "tag";
  public static final String SORTING_SCREEN = "sort";

  private final CardLayout cardLayoutController = new CardLayout();
  private final JPanel rootContainerPanel = new JPanel(cardLayoutController);
  private final ApplicationSessionState sessionState = new ApplicationSessionState();
  private final LibrarySelectionAndScanPanel librarySelectionPanel;
  private final InteractiveTaggingWorkflowPanel taggingWorkflowPanel;
  private final CategorySortingControlPanel sortingControlPanel;

  public PrimaryApplicationFrame() {
    super("Clanker Sort");
    setDefaultCloseOperation(JFrame.EXIT_ON_CLOSE);
    setSize(860, 620);
    setLocationRelativeTo(null);

    librarySelectionPanel = new LibrarySelectionAndScanPanel(this, sessionState);
    taggingWorkflowPanel = new InteractiveTaggingWorkflowPanel(this, sessionState);
    sortingControlPanel = new CategorySortingControlPanel(this, sessionState);

    rootContainerPanel.add(librarySelectionPanel, BROWSE_SCREEN);
    rootContainerPanel.add(taggingWorkflowPanel, TAGGING_SCREEN);
    rootContainerPanel.add(sortingControlPanel, SORTING_SCREEN);
    setContentPane(rootContainerPanel);
    navigateToBrowseScreen();
  }

  public void navigateToBrowseScreen() {
    librarySelectionPanel.prepareForDisplay();
    cardLayoutController.show(rootContainerPanel, BROWSE_SCREEN);
  }

  public void navigateToTaggingScreen() {
    taggingWorkflowPanel.prepareForDisplay();
    cardLayoutController.show(rootContainerPanel, TAGGING_SCREEN);
  }

  public void navigateToSortingScreen() {
    sortingControlPanel.prepareForDisplay();
    cardLayoutController.show(rootContainerPanel, SORTING_SCREEN);
  }
}
