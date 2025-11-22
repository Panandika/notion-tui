package ui

// Navigator manages page navigation and history for the application.
// It maintains a stack of visited pages and provides methods to navigate
// forward and backward through the history.
type Navigator struct {
	currentPage PageID
	history     []PageID
	maxHistory  int
}

// NewNavigatorInput contains parameters for creating a new Navigator.
type NewNavigatorInput struct {
	InitialPage PageID
	MaxHistory  int // Default: 50
}

const (
	// PageList represents the list view page.
	PageList PageID = "list"
	// PageDetail represents the detail/content view page.
	PageDetail PageID = "detail"
	// PageEdit represents the edit mode page.
	PageEdit PageID = "edit"
)

const (
	// DefaultMaxHistory is the default maximum history size.
	DefaultMaxHistory = 50
)

// NewNavigator creates a new Navigator instance with the specified configuration.
// If MaxHistory is not set or is less than 1, it defaults to DefaultMaxHistory.
func NewNavigator(input NewNavigatorInput) Navigator {
	maxHistory := input.MaxHistory
	if maxHistory < 1 {
		maxHistory = DefaultMaxHistory
	}

	return Navigator{
		currentPage: input.InitialPage,
		history:     make([]PageID, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

// NavigateTo navigates to a new page, pushing the current page to history.
// If the history exceeds maxHistory, the oldest entry is removed.
func (n *Navigator) NavigateTo(pageID PageID) {
	// Push current page to history before navigating
	if !n.currentPage.IsEmpty() {
		n.history = append(n.history, n.currentPage)

		// Enforce history size limit
		if len(n.history) > n.maxHistory {
			// Remove oldest entry (shift left)
			n.history = n.history[1:]
		}
	}

	n.currentPage = pageID
}

// Back navigates to the previous page in history.
// Returns the previous page and true if successful, or empty PageID and false if history is empty.
func (n *Navigator) Back() (PageID, bool) {
	if len(n.history) == 0 {
		return PageID(""), false
	}

	// Pop from history
	previousPage := n.history[len(n.history)-1]
	n.history = n.history[:len(n.history)-1]

	// Set as current page
	n.currentPage = previousPage

	return previousPage, true
}

// CanGoBack returns true if there is history to navigate back to.
func (n *Navigator) CanGoBack() bool {
	return len(n.history) > 0
}

// CurrentPage returns the currently active page.
func (n *Navigator) CurrentPage() PageID {
	return n.currentPage
}

// History returns a copy of the navigation history.
// The returned slice is ordered from oldest to newest.
func (n *Navigator) History() []PageID {
	// Return a copy to prevent external modification
	historyCopy := make([]PageID, len(n.history))
	copy(historyCopy, n.history)
	return historyCopy
}

// ClearHistory removes all history entries but keeps the current page.
func (n *Navigator) ClearHistory() {
	n.history = make([]PageID, 0, n.maxHistory)
}

// Reset resets the navigator to a new page and clears all history.
func (n *Navigator) Reset(pageID PageID) {
	n.currentPage = pageID
	n.history = make([]PageID, 0, n.maxHistory)
}
