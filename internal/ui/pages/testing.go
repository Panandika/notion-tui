package pages

import tea "github.com/charmbracelet/bubbletea"

// NewLoadedMsgForTest builds the message a DetailPage receives when its
// content has loaded.
//
// It exists for tests in other packages (the root model's), which cannot
// construct the unexported message and must not call the real API to get a
// page into the loaded state. Not for use outside tests.
func NewLoadedMsgForTest(pageID, markdown string) tea.Msg {
	return pageLoadedMsg{pageID: pageID, markdown: markdown}
}

// NewRowsLoadedMsgForTest builds the message a ListPage receives when its
// rows have loaded. Not for use outside tests, for the same reason as
// NewLoadedMsgForTest.
//
// The message answers the list's latest load, as a real result would;
// results of earlier loads are dropped by the page.
func NewRowsLoadedMsgForTest(list *ListPage, rows ...Page) tea.Msg {
	return pagesLoadedMsg{epoch: list.epoch, pages: rows}
}
