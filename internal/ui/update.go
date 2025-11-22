package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/ui/pages"
)

// HandleGlobalKeys processes global keyboard shortcuts that work across all pages.
// Returns (handled bool, cmd tea.Cmd) where handled indicates if the key was processed.
// Global keys include quit (Ctrl+C, q) and help (?).
func HandleGlobalKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return true, tea.Quit
	case "?":
		// TODO: Show help overlay
		return true, nil
	default:
		return false, nil
	}
}

// HandleNavigationKeys processes navigation keys (up, down, k, j).
// Returns updated cursor position and a boolean indicating if key was handled.
// Does not move cursor beyond bounds (0 to len(pages)-1).
func HandleNavigationKeys(msg tea.KeyMsg, cursor int, pageCount int) (int, bool) {
	newCursor := cursor
	handled := false

	switch msg.String() {
	case "up", "k":
		if cursor > 0 {
			newCursor = cursor - 1
		}
		handled = true
	case "down", "j":
		if cursor < pageCount-1 {
			newCursor = cursor + 1
		}
		handled = true
	}

	return newCursor, handled
}

// HandleWindowResize updates model dimensions and marks the model as ready.
// Must be called to enable UI rendering after terminal size is known.
func HandleWindowResize(m *AppModel, width, height int) {
	m.width = width
	m.height = height
	m.ready = true
}

// HandlePagesLoaded updates the model with newly loaded pages.
// Adjusts cursor position if it exceeds new page list length.
// Note: This function is deprecated and kept for compatibility.
// Use pages.ListPage directly instead.
func HandlePagesLoaded(m *AppModel, pagesList []pages.Page, err error) {
	m.pageList = pagesList
	m.err = err
}

// ExtractPageTitle extracts the title from a Notion page.
// Tries Title property first, then Name, then falls back to page ID.
func ExtractPageTitle(page notionapi.Page) string {
	// Try to find a Title property
	for key, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			if len(titleProp.Title) > 0 {
				return titleProp.Title[0].PlainText
			}
		}
		// Some databases use "Name" instead of "Title"
		if key == "Name" || key == "Title" {
			if titleProp, ok := prop.(*notionapi.RichTextProperty); ok {
				if len(titleProp.RichText) > 0 {
					return titleProp.RichText[0].PlainText
				}
			}
		}
	}

	// Fallback to page ID
	return page.ID.String()
}
