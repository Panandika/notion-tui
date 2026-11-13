package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/Panandika/notion-tui/internal/keymap"
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
// the key handling in model.go and the pages. "{action}" stands for the key
// of an action the user can rebind (see the keymap package).
var helpSections = []helpSection{
	{"Everywhere", [][2]string{
		{"{help}", "show this help"},
		{"{palette}", "command palette"},
		{"{sidebar}", "show or hide the sidebar"},
		{"{focus}", "switch focus between sidebar and page"},
		{"esc", "go back"},
		{"{quit} / ctrl+c", "quit"},
	}},
	{"Sidebar", [][2]string{
		{"↑/k  ↓/j", "move"},
		{"←/h  →/l", "collapse / expand"},
		{"enter", "open page or database; expand a parent"},
		{"{tree_reload}", "reload the workspace"},
	}},
	{"Page", [][2]string{
		{"↑/k  ↓/j", "scroll"},
		{"{edit}", "edit the page in $VISUAL / $EDITOR"},
		{"{properties}", "change properties (status, select, date, checkbox)"},
		{"{todos}", "check or uncheck to-dos"},
		{"{links}", "follow a link: child page, database, mention, web"},
		{"{back}  {forward}", "back / forward through pages read"},
		{"{find}  {find_next}  {find_previous}", "find in page, next, previous"},
		{"{new_page}", "new page inside this page"},
		{"{archive}", "archive the page (asks first)"},
		{"{reload}", "reload from Notion"},
	}},
	{"Database rows", [][2]string{
		{"enter", "open the row"},
		{"{row_new}", "new row"},
		{"{row_sort}", "change sort order"},
		{"{row_filter}", "filter rows by title"},
		{"{row_reload}", "reload"},
		{"{row_more}", "load more rows"},
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

// RenderHelp renders the full-screen key reference, centered in the
// terminal, with the keys the user chose.
func RenderHelp(width, height int, keys *keymap.Map) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(darkTheme.Primary)
	sectionStyle := lipgloss.NewStyle().Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(darkTheme.Primary).Width(14)
	mutedStyle := lipgloss.NewStyle().Foreground(darkTheme.Muted)

	lines := []string{titleStyle.Render("Keyboard shortcuts"), ""}
	for _, section := range helpSections {
		lines = append(lines, sectionStyle.Render(section.title))
		for _, binding := range section.keys {
			lines = append(lines, "  "+keyStyle.Render(keys.Label(binding[0]))+" "+binding[1])
		}
		lines = append(lines, "")
	}
	lines = append(lines, mutedStyle.Render("Keys can be changed under \"keys:\" in the config file. Press any key to close"))

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
