package ui

import "fmt"

// errorMsg represents an error that occurred during operation.
type errorMsg struct {
	message string
	err     error
}

// ErrorMsg is the exported type for error messages.
type ErrorMsg = errorMsg

// Error implements the error interface for errorMsg.
func (e errorMsg) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

// Unwrap returns the underlying error.
func (e errorMsg) Unwrap() error {
	return e.err
}

// NewErrorMsg creates a new error message.
func NewErrorMsg(message string, err error) errorMsg {
	return errorMsg{
		message: message,
		err:     err,
	}
}

// NavigationMsg requests navigation to a specific page.
type NavigationMsg struct {
	pageID string
}

// PageID returns the target page ID.
func (n NavigationMsg) PageID() string {
	return n.pageID
}

// NewNavigationMsg creates a new navigation message.
func NewNavigationMsg(pageID string) NavigationMsg {
	return NavigationMsg{pageID: pageID}
}

// loadingMsg indicates loading state change.
type loadingMsg struct {
	isLoading bool
}

// IsLoading returns the loading state.
func (l loadingMsg) IsLoading() bool {
	return l.isLoading
}

// NewLoadingMsg creates a new loading message.
func NewLoadingMsg(isLoading bool) loadingMsg {
	return loadingMsg{isLoading: isLoading}
}

// contentLoadedMsg contains loaded content or error.
type contentLoadedMsg struct {
	content string
	err     error
}

// ContentLoadedMsg is the exported type for content loaded messages.
type ContentLoadedMsg = contentLoadedMsg

// Content returns the loaded content.
func (c contentLoadedMsg) Content() string {
	return c.content
}

// Err returns any error that occurred.
func (c contentLoadedMsg) Err() error {
	return c.err
}

// NewContentLoadedMsg creates a new content loaded message.
func NewContentLoadedMsg(content string, err error) contentLoadedMsg {
	return contentLoadedMsg{
		content: content,
		err:     err,
	}
}

// refreshMsg requests a data refresh.
type refreshMsg struct{}

// NewRefreshMsg creates a new refresh message.
func NewRefreshMsg() refreshMsg {
	return refreshMsg{}
}

// itemSelectedMsg indicates an item was selected.
type itemSelectedMsg struct {
	index int
	id    string
}

// Index returns the selected item index.
func (i itemSelectedMsg) Index() int {
	return i.index
}

// ID returns the selected item ID.
func (i itemSelectedMsg) ID() string {
	return i.id
}

// NewItemSelectedMsg creates a new item selected message.
func NewItemSelectedMsg(index int, id string) itemSelectedMsg {
	return itemSelectedMsg{
		index: index,
		id:    id,
	}
}

// syncStatusMsg indicates sync status change.
type syncStatusMsg struct {
	status string
}

// Status returns the sync status.
func (s syncStatusMsg) Status() string {
	return s.status
}

// NewSyncStatusMsg creates a new sync status message.
func NewSyncStatusMsg(status string) syncStatusMsg {
	return syncStatusMsg{status: status}
}

// viewModeChangedMsg indicates view mode change.
type viewModeChangedMsg struct {
	mode ViewMode
}

// Mode returns the new view mode.
func (v viewModeChangedMsg) Mode() ViewMode {
	return v.mode
}

// NewViewModeChangedMsg creates a new view mode changed message.
func NewViewModeChangedMsg(mode ViewMode) viewModeChangedMsg {
	return viewModeChangedMsg{mode: mode}
}
