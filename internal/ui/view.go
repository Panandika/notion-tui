package ui

import (
	"fmt"
)

// RenderInitializing returns the loading screen text.
func RenderInitializing() string {
	return "Initializing...\n"
}

// RenderError formats error messages for display.
// Returns empty string if err is nil, otherwise returns "Error: {err}" format.
func RenderError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("Error: %v\n", err)
}

// RenderEmptyPageList returns a message when no pages are found.
func RenderEmptyPageList() string {
	return "No pages found in database.\n(r: refresh, q: quit)\n"
}

// RenderPageList renders the simple page list view with cursor selection.
// Displays list of pages with current cursor position and navigation help.
func RenderPageList(pages []string, cursor int) string {
	s := "Notion Pages\n\n"
	for i, page := range pages {
		cursorChar := " "
		if i == cursor {
			cursorChar = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursorChar, page)
	}
	s += fmt.Sprintf("\n(%d/%d)\n", cursor+1, len(pages))
	s += "(↑/↓ or k/j: navigate, r: refresh, q: quit)\n"
	return s
}

// RenderViewState is a helper that returns the complete view for the current model state.
// Handles the logic of when to show initializing, error, empty, or page list views.
func RenderViewState(ready bool, err error, pages []string, cursor int) string {
	if !ready {
		return RenderInitializing()
	}

	if err != nil {
		return RenderError(err)
	}

	if len(pages) == 0 {
		return RenderEmptyPageList()
	}

	return RenderPageList(pages, cursor)
}
