package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// RenderInitializing returns the loading screen text.
func RenderInitializing() string {
	return "Initializing...\n"
}

// helpSection is one titled group of key bindings on the help screen.
type helpSection struct {
	title string
	keys  [][2]string // key, description
}

// helpSections lists every key the app responds to. Keep it in sync with
// the key handling in model.go and the pages.
var helpSections = []helpSection{
	{"Everywhere", [][2]string{
		{"?", "show this help"},
		{"ctrl+p", "command palette"},
		{"ctrl+b", "show or hide the sidebar"},
		{"tab", "switch focus between sidebar and page"},
		{"esc", "go back"},
		{"q / ctrl+c", "quit"},
	}},
	{"Sidebar", [][2]string{
		{"↑/k  ↓/j", "move"},
		{"←/h  →/l", "collapse / expand"},
		{"enter", "open page or database; expand a parent"},
	}},
	{"Page", [][2]string{
		{"↑/k  ↓/j", "scroll"},
		{"e", "edit the page in $VISUAL / $EDITOR"},
		{"p", "change properties (status, select, date, checkbox)"},
		{"t", "check or uncheck to-dos"},
		{"o", "follow a link: child page, database, mention, web"},
		{"[  ]", "back / forward through pages read"},
		{"/  n  N", "find in page, next, previous"},
		{"ctrl+n", "new page inside this page"},
		{"D", "archive the page (asks first)"},
		{"r", "reload from Notion"},
	}},
	{"Database rows", [][2]string{
		{"enter", "open the row"},
		{"n", "new row"},
		{"s", "change sort order"},
		{"/", "filter rows by title"},
		{"r", "reload"},
	}},
	{"Search", [][2]string{
		{"enter", "search, or open the selected result"},
		{"↓ / ↑", "move between the search box and results"},
		{"ctrl+t", "switch between workspace and database search"},
	}},
	{"Command palette", [][2]string{
		{"↑/k  ↓/j", "move"},
		{"/", "filter commands by name"},
		{"enter", "run: search, switch database, refresh, new page, export, help"},
	}},
}

// RenderHelp renders the full-screen key reference, centered in the terminal.
func RenderHelp(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(darkTheme.Primary)
	sectionStyle := lipgloss.NewStyle().Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(darkTheme.Primary).Width(14)
	mutedStyle := lipgloss.NewStyle().Foreground(darkTheme.Muted)

	lines := []string{titleStyle.Render("Keyboard shortcuts"), ""}
	for _, section := range helpSections {
		lines = append(lines, sectionStyle.Render(section.title))
		for _, binding := range section.keys {
			lines = append(lines, "  "+keyStyle.Render(binding[0])+binding[1])
		}
		lines = append(lines, "")
	}
	lines = append(lines, mutedStyle.Render("Press any key to close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(darkTheme.Secondary).
		Padding(1, 3).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	if width <= 0 || height <= 0 {
		return box
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
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
