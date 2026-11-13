package pages

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// detailMode is what the detail page is showing in place of the page view.
type detailMode int

const (
	modeView       detailMode = iota // The page itself
	modeProperties                   // List of editable properties
	modeOptions                      // Choices for one status or select property
	modeDate                         // Text prompt for one date property
	modeTodos                        // List of the page's to-dos
	modeLinks                        // List of things to follow from the page
	modeSearch                       // Text prompt for searching the page
	modeConfirm                      // Yes/no question before archiving
)

const (
	propertiesHelpText = "↑/↓: move | enter: change | esc: done"
	optionsHelpText    = "↑/↓: move | enter: choose | esc: back"
	dateHelpText       = "type YYYY-MM-DD, empty to clear | enter: save | esc: back"
	todosHelpText      = "↑/↓: move | space: check / uncheck | esc: done"
	linksHelpText      = "↑/↓: move | enter: open | esc: cancel"
	searchHelpText     = "type text to find | enter: search | esc: cancel"
	confirmHelpText    = "y: archive | n / esc: cancel"

	// clearOption is the picker entry that unsets a select property.
	clearOption = "(none)"
)

// schemaLoadedMsg carries the database that defines a page's properties.
type schemaLoadedMsg struct {
	pageID   string
	database *notionapi.Database
	err      error
}

// propertySavedMsg is the result of changing a property. On success it
// carries the page as Notion now has it.
type propertySavedMsg struct {
	pageID string
	page   *notionapi.Page
	err    error
}

// todoSavedMsg is the result of checking or unchecking a to-do.
type todoSavedMsg struct {
	pageID    string
	markdown  string
	truncated bool
	err       error
}

// OpenLinkMsg asks the root model to open a Notion page or database that
// the current page links to.
type OpenLinkMsg struct {
	ID       string
	Database bool
}

// PageArchivedMsg is emitted after the page was moved to Notion's trash.
type PageArchivedMsg struct {
	ID string
}

// archivedMsg is the result of archiving the page.
type archivedMsg struct {
	pageID string
	err    error
}

// linkOpenedMsg is the result of handing a web link to the browser.
type linkOpenedMsg struct {
	pageID string
	url    string
	err    error
}

func (archivedMsg) detailResult()      {}
func (linkOpenedMsg) detailResult()    {}
func (schemaLoadedMsg) detailResult()  {}
func (propertySavedMsg) detailResult() {}
func (todoSavedMsg) detailResult()     {}

// todoItem is one to-do line of the page's Markdown.
type todoItem struct {
	line    int // Index of the line in the Markdown
	checked bool
	text    string
}

var (
	todoLinePattern = regexp.MustCompile(`^(\s*- \[)([ xX])(\] ?)(.*)$`)
	fenceOpen       = regexp.MustCompile("^\\s*(`{3,}|~{3,})(.*)$")
)

// parseTodos finds the to-do lines of a page, ignoring code blocks.
func parseTodos(markdown string) []todoItem {
	var todos []todoItem
	fence := "" // Marker that opened the current code block
	for index, line := range strings.Split(markdown, "\n") {
		if fence != "" {
			if closesCodeFence(line, fence) {
				fence = ""
			}
			continue
		}
		if m := fenceOpen.FindStringSubmatch(line); m != nil {
			// A backtick fence cannot have a backtick in its info string;
			// "```x``` text" is inline code, not the start of a block.
			if m[1][0] != '`' || !strings.Contains(m[2], "`") {
				fence = m[1]
				continue
			}
		}
		if m := todoLinePattern.FindStringSubmatch(line); m != nil {
			todos = append(todos, todoItem{line: index, checked: m[2] != " ", text: m[4]})
		}
	}
	return todos
}

// closesCodeFence reports whether a line ends a code block opened with
// marker: the same character, at least as many, and nothing else.
func closesCodeFence(line, marker string) bool {
	trimmed := strings.TrimSpace(line)
	return len(trimmed) >= len(marker) && strings.Trim(trimmed, marker[:1]) == ""
}

// toggleTodoLine returns the Markdown with the to-do on the given line
// checked or unchecked. ok is false when that line is not a to-do of the
// page (for example a look-alike line inside a code block).
func toggleTodoLine(markdown string, line int) (edited string, ok bool) {
	isTodo := false
	for _, todo := range parseTodos(markdown) {
		if todo.line == line {
			isTodo = true
			break
		}
	}
	if !isTodo {
		return markdown, false
	}

	lines := strings.Split(markdown, "\n")
	m := todoLinePattern.FindStringSubmatch(lines[line])
	mark := "x"
	if m[2] != " " {
		mark = " "
	}
	lines[line] = m[1] + mark + m[3] + m[4]
	return strings.Join(lines, "\n"), true
}

// pageHeader renders the title and properties shown above the page content.
func pageHeader(page *notionapi.Page) string {
	if page == nil {
		return ""
	}

	var parts []string
	for _, property := range notion.PageProperties(page) {
		switch {
		case property.Type == notionapi.PropertyTypeCheckbox:
			box := "☐"
			if property.Checked {
				box = "☑"
			}
			parts = append(parts, fmt.Sprintf("%s %s", box, escapeDisplay(property.Name)))
		case property.Text != "":
			parts = append(parts, fmt.Sprintf("**%s:** %s", escapeDisplay(property.Name), escapeDisplay(property.Text)))
		}
	}

	// Titles and values are the user's text, not Markdown: "*x*" must show
	// as typed rather than turn italic.
	header := "# " + escapeDisplay(notion.PageTitle(page))
	if len(parts) > 0 {
		header += "\n\n" + strings.Join(parts, "  ·  ")
	}
	return header
}

var (
	markdownLink   = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	emphasisMarker = regexp.MustCompile("\\*\\*|~~|`")
)

// escapeDisplay makes plain text safe to put into the Markdown shown in the
// viewer. Underscores are included because standard Markdown reads them as
// emphasis too.
func escapeDisplay(text string) string {
	return strings.ReplaceAll(notion.EscapeMarkdown(text), "_", `\_`)
}

// plainText reduces a line of Notion-flavored Markdown to readable text for
// lists, where formatting markers would only be noise.
func plainText(markdown string) string {
	text := notion.RenderableMarkdown(markdown)
	text = markdownLink.ReplaceAllString(text, "$1")
	text = emphasisMarker.ReplaceAllString(text, "")
	return strings.TrimSpace(notion.UnescapeMarkdown(text))
}

// StartProperties opens the list of properties that can be changed.
func (dp *DetailPage) StartProperties() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	databaseID := ""
	if dp.page != nil {
		databaseID = string(dp.page.Parent.DatabaseID)
	}
	if databaseID == "" {
		dp.setNotice("This page is not a database row, so it has no properties to change")
		return nil
	}
	if len(dp.editableProperties()) == 0 {
		dp.setNotice("This page has no status, select, date or checkbox properties to change")
		return nil
	}

	// The database says which options a status or select property offers.
	if dp.schema != nil && string(dp.schema.ID) == databaseID {
		dp.openProperties(0)
		return nil
	}
	dp.busy = true
	dp.setNotice("Loading properties...")
	pageID := dp.pageID
	client := dp.notionClient
	return func() tea.Msg {
		database, err := client.GetDatabase(context.Background(), databaseID)
		return schemaLoadedMsg{pageID: pageID, database: database, err: err}
	}
}

// StartTodos opens the list of the page's to-dos.
func (dp *DetailPage) StartTodos() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	if dp.truncated {
		dp.setNotice("Page is too large to change here: Notion returned only part of it")
		return nil
	}
	dp.todos = parseTodos(dp.markdown)
	if len(dp.todos) == 0 {
		dp.setNotice("This page has no to-dos")
		return nil
	}
	dp.picker = components.NewPicker("To-dos", todoPickerItems(dp.todos))
	dp.setMode(modeTodos)
	return nil
}

func todoPickerItems(todos []todoItem) []components.PickerItem {
	items := make([]components.PickerItem, 0, len(todos))
	for _, todo := range todos {
		box := "[ ]"
		if todo.checked {
			box = "[x]"
		}
		text := plainText(todo.text)
		if text == "" {
			text = "(empty)"
		}
		items = append(items, components.PickerItem{Label: box + " " + text})
	}
	return items
}

// editableProperties returns the properties the app can change.
func (dp *DetailPage) editableProperties() []notion.PropertyValue {
	var editable []notion.PropertyValue
	for _, property := range notion.PageProperties(dp.page) {
		if property.Editable() {
			editable = append(editable, property)
		}
	}
	return editable
}

// openProperties shows the property list with the cursor on a given row.
func (dp *DetailPage) openProperties(cursor int) {
	properties := dp.editableProperties()
	items := make([]components.PickerItem, 0, len(properties))
	for _, property := range properties {
		detail := property.Text
		if property.Type != notionapi.PropertyTypeCheckbox && detail == "" {
			detail = "—"
		}
		items = append(items, components.PickerItem{Label: property.Name, Detail: detail})
	}
	dp.picker = components.NewPicker("Properties", items)
	dp.picker.SetCursor(cursor)
	dp.propCursor = dp.picker.Cursor()
	dp.setMode(modeProperties)
}

// setMode switches what the page shows and the key help that goes with it.
func (dp *DetailPage) setMode(mode detailMode) {
	dp.mode = mode
	dp.notice = ""
	dp.inputErr = ""
	switch mode {
	case modeProperties:
		dp.statusBar.SetHelpText(propertiesHelpText)
	case modeOptions:
		dp.statusBar.SetHelpText(optionsHelpText)
	case modeDate:
		dp.statusBar.SetHelpText(dateHelpText)
	case modeTodos:
		dp.statusBar.SetHelpText(todosHelpText)
	case modeLinks:
		dp.statusBar.SetHelpText(linksHelpText)
	case modeSearch:
		dp.statusBar.SetHelpText(searchHelpText)
	case modeConfirm:
		dp.statusBar.SetHelpText(confirmHelpText)
	default:
		dp.statusBar.SetHelpText(detailHelpText)
	}
}

// updateMode handles a key while a mode other than the page view is open.
func (dp *DetailPage) updateMode(msg tea.KeyMsg) tea.Cmd {
	// A change is being saved: wait for it before accepting more input.
	if dp.busy {
		return nil
	}
	dp.inputErr = ""

	switch dp.mode {
	case modeProperties:
		return dp.updateProperties(msg)
	case modeOptions:
		return dp.updateOptions(msg)
	case modeDate:
		return dp.updateDate(msg)
	case modeTodos:
		return dp.updateTodos(msg)
	case modeLinks:
		return dp.updateLinks(msg)
	case modeSearch:
		return dp.updateSearch(msg)
	case modeConfirm:
		return dp.updateConfirm(msg)
	default:
		return nil
	}
}

// StartLinks lists what can be followed from the page: child pages,
// databases, mentions and web links.
func (dp *DetailPage) StartLinks() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	dp.links = notion.PageLinks(dp.markdown)
	if len(dp.links) == 0 {
		dp.setNotice("This page has no links to follow")
		return nil
	}
	items := make([]components.PickerItem, 0, len(dp.links))
	for _, link := range dp.links {
		title := plainText(link.Title)
		switch link.Kind {
		case notion.LinkPage:
			items = append(items, components.PickerItem{Label: "📄 " + title})
		case notion.LinkDatabase:
			items = append(items, components.PickerItem{Label: "🗂 " + title})
		default:
			items = append(items, components.PickerItem{Label: "↗ " + title, Detail: link.URL})
		}
	}
	dp.picker = components.NewPicker("Links", items)
	dp.setMode(modeLinks)
	return nil
}

func (dp *DetailPage) updateLinks(msg tea.KeyMsg) tea.Cmd {
	if dp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc", "q":
		dp.setMode(modeView)
		return nil
	case "enter":
		index := dp.picker.Cursor()
		if index >= len(dp.links) {
			return nil
		}
		link := dp.links[index]
		dp.setMode(modeView)
		if link.Kind == notion.LinkWeb {
			pageID := dp.pageID
			opener := dp.openURL
			return func() tea.Msg {
				return linkOpenedMsg{pageID: pageID, url: link.URL, err: opener(link.URL)}
			}
		}
		open := OpenLinkMsg{ID: link.ID, Database: link.Kind == notion.LinkDatabase}
		return func() tea.Msg { return open }
	}
	return nil
}

// StartSearch opens the prompt for finding text in the page.
func (dp *DetailPage) StartSearch() tea.Cmd {
	if !dp.ready() || dp.viewer == nil {
		return nil
	}
	input := textinput.New()
	input.Prompt = "Find: "
	input.CharLimit = 100
	input.SetValue(dp.lastSearch)
	input.CursorEnd()
	input.Focus()
	dp.searchInput = input
	dp.setMode(modeSearch)
	return nil
}

func (dp *DetailPage) updateSearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		dp.setMode(modeView)
		return nil
	case "enter":
		query := strings.TrimSpace(dp.searchInput.Value())
		dp.setMode(modeView)
		if query == "" {
			return nil
		}
		dp.lastSearch = query
		if total := dp.viewer.Find(query); total > 0 {
			dp.setNotice(fmt.Sprintf("Match 1 of %d for %q | n: next | N: previous", total, query))
		} else {
			dp.setNotice(fmt.Sprintf("No match for %q", query))
		}
		return nil
	}
	var cmd tea.Cmd
	dp.searchInput, cmd = dp.searchInput.Update(msg)
	return cmd
}

// findNext jumps to the next or previous match of the last search.
func (dp *DetailPage) findNext(delta int) {
	if dp.viewer == nil || dp.lastSearch == "" {
		dp.setNotice("Nothing searched yet: press / to find text")
		return
	}
	position, total := dp.viewer.FindNext(delta)
	if total == 0 {
		dp.setNotice(fmt.Sprintf("No match for %q", dp.lastSearch))
		return
	}
	dp.setNotice(fmt.Sprintf("Match %d of %d for %q | n: next | N: previous", position, total, dp.lastSearch))
}

// StartArchive asks before moving the page to Notion's trash.
func (dp *DetailPage) StartArchive() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	dp.setMode(modeConfirm)
	return nil
}

func (dp *DetailPage) updateConfirm(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		dp.busy = true
		dp.statusBar.SetSyncStatus(components.StatusSyncing)
		pageID := dp.pageID
		client := dp.notionClient
		return func() tea.Msg {
			_, err := client.UpdatePage(context.Background(), pageID, &notionapi.PageUpdateRequest{Archived: true})
			return archivedMsg{pageID: pageID, err: err}
		}
	case "n", "N", "esc", "q":
		dp.setMode(modeView)
	}
	return nil
}

func (dp *DetailPage) handleArchived(msg archivedMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	dp.busy = false
	dp.setMode(modeView)
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		dp.setNotice("Could not archive: " + msg.err.Error())
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)
	// The cached copy must not bring the page back
	if dp.cache != nil {
		_ = dp.cache.Delete(dp.pageID)
	}
	archived := PageArchivedMsg{ID: dp.pageID}
	return func() tea.Msg { return archived }
}

func (dp *DetailPage) handleLinkOpened(msg linkOpenedMsg) {
	if msg.pageID != dp.pageID {
		return
	}
	if msg.err != nil {
		dp.setNotice(fmt.Sprintf("Could not open the browser (%v). Link: %s", msg.err, msg.url))
		return
	}
	dp.setNotice("Opened in the browser")
}

func (dp *DetailPage) updateProperties(msg tea.KeyMsg) tea.Cmd {
	if dp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc", "q":
		dp.setMode(modeView)
		return nil
	case "enter", " ":
		properties := dp.editableProperties()
		index := dp.picker.Cursor()
		if index >= len(properties) {
			return nil
		}
		dp.propCursor = index
		dp.editing = properties[index]
		return dp.beginPropertyEdit()
	}
	return nil
}

// beginPropertyEdit opens the right input for the chosen property.
func (dp *DetailPage) beginPropertyEdit() tea.Cmd {
	property := dp.editing
	switch property.Type {
	case notionapi.PropertyTypeCheckbox:
		return dp.savePropertyCmd(notion.CheckboxUpdate(property.Name, !property.Checked))

	case notionapi.PropertyTypeStatus, notionapi.PropertyTypeSelect:
		options := notion.PropertyOptions(dp.schema, property.Name)
		if len(options) == 0 {
			dp.inputErr = fmt.Sprintf("%q has no options defined in Notion", property.Name)
			return nil
		}
		// A status always has a value; a select can be emptied.
		if property.Type == notionapi.PropertyTypeSelect {
			options = append([]string{clearOption}, options...)
		}
		items := make([]components.PickerItem, 0, len(options))
		current := 0
		for index, option := range options {
			items = append(items, components.PickerItem{Label: option})
			if option == property.Text {
				current = index
			}
		}
		dp.options = options
		dp.picker = components.NewPicker(property.Name, items)
		dp.picker.SetCursor(current)
		dp.setMode(modeOptions)
		return nil

	case notionapi.PropertyTypeDate:
		input := textinput.New()
		input.Placeholder = "YYYY-MM-DD"
		input.CharLimit = 10
		input.Prompt = property.Name + ": "
		// Only a single date can be typed here. A value with a time or an
		// end date is longer than a bare date; the prompt says that saving a
		// change will replace it.
		dp.dateCurrent = ""
		dp.dateComplex = len(property.Text) > len("2006-01-02")
		if len(property.Text) >= 10 {
			dp.dateCurrent = property.Text[:10]
			input.SetValue(dp.dateCurrent)
		}
		input.CursorEnd()
		input.Focus()
		dp.dateInput = input
		dp.setMode(modeDate)
		return nil
	}
	return nil
}

func (dp *DetailPage) updateOptions(msg tea.KeyMsg) tea.Cmd {
	if dp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc", "q":
		dp.openProperties(dp.propCursor)
		return nil
	case "enter", " ":
		index := dp.picker.Cursor()
		if index >= len(dp.options) {
			return nil
		}
		choice := dp.options[index]
		// Choosing the current value changes nothing: do not write it.
		if choice == dp.editing.Text || (choice == clearOption && dp.editing.Text == "") {
			dp.openProperties(dp.propCursor)
			return nil
		}
		if dp.editing.Type == notionapi.PropertyTypeStatus {
			return dp.savePropertyCmd(notion.StatusUpdate(dp.editing.Name, choice))
		}
		if choice == clearOption {
			choice = ""
		}
		return dp.savePropertyCmd(notion.SelectUpdate(dp.editing.Name, choice))
	}
	return nil
}

func (dp *DetailPage) updateDate(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		dp.openProperties(dp.propCursor)
		return nil
	case "enter":
		// Unchanged input writes nothing. This matters most for a date with
		// a time or end date, which a save would reduce to a single date.
		if strings.TrimSpace(dp.dateInput.Value()) == dp.dateCurrent {
			dp.openProperties(dp.propCursor)
			return nil
		}
		update, err := notion.DateUpdate(dp.editing.Name, dp.dateInput.Value())
		if err != nil {
			dp.inputErr = err.Error()
			return nil
		}
		return dp.savePropertyCmd(update)
	}
	var cmd tea.Cmd
	dp.dateInput, cmd = dp.dateInput.Update(msg)
	return cmd
}

func (dp *DetailPage) updateTodos(msg tea.KeyMsg) tea.Cmd {
	if dp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc", "q":
		dp.setMode(modeView)
		return nil
	case " ", "enter", "x":
		index := dp.picker.Cursor()
		if index >= len(dp.todos) {
			return nil
		}
		// The list must still describe the page. If the content changed
		// underneath it, a stored line number could point at another line.
		current := parseTodos(dp.markdown)
		if index >= len(current) || current[index] != dp.todos[index] {
			dp.syncTodos()
			dp.inputErr = "The page changed; the list was updated. Check the selection and try again"
			return nil
		}
		edited, ok := toggleTodoLine(dp.markdown, dp.todos[index].line)
		if !ok {
			return nil
		}
		dp.busy = true
		dp.statusBar.SetSyncStatus(components.StatusSyncing)

		pageID := dp.pageID
		client := dp.notionClient
		original := dp.markdown
		return func() tea.Msg {
			result, err := writePageMarkdown(context.Background(), client, pageID, original, edited)
			if err != nil {
				return todoSavedMsg{pageID: pageID, err: err}
			}
			markdown := result.Markdown
			if strings.TrimSpace(markdown) == "" {
				markdown = edited
			}
			return todoSavedMsg{pageID: pageID, markdown: markdown, truncated: result.Truncated}
		}
	}
	return nil
}

// savePropertyCmd writes one property and then reads the page back, so the
// page shows what Notion stored.
func (dp *DetailPage) savePropertyCmd(update notion.PropertyUpdate) tea.Cmd {
	dp.busy = true
	dp.statusBar.SetSyncStatus(components.StatusSyncing)

	pageID := dp.pageID
	client := dp.notionClient
	return func() tea.Msg {
		ctx := context.Background()
		if err := client.SetPageProperty(ctx, pageID, update); err != nil {
			return propertySavedMsg{pageID: pageID, err: err}
		}
		page, err := client.GetPage(ctx, pageID)
		if err != nil {
			return propertySavedMsg{pageID: pageID, err: fmt.Errorf("saved, but could not reload the page: %w", err)}
		}
		return propertySavedMsg{pageID: pageID, page: page}
	}
}

func (dp *DetailPage) handleSchemaLoaded(msg schemaLoadedMsg) {
	if msg.pageID != dp.pageID {
		return
	}
	dp.busy = false
	if msg.err != nil {
		dp.setNotice("Could not load properties: " + msg.err.Error())
		return
	}
	dp.schema = msg.database
	dp.clearNotice()
	dp.openProperties(0)
}

func (dp *DetailPage) handlePropertySaved(msg propertySavedMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	dp.busy = false
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		dp.inputErr = "Could not save: " + msg.err.Error()
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)
	dp.page = msg.page

	// Back to the property list, which now shows the new value, and redraw
	// the header behind it.
	dp.openProperties(dp.propCursor)
	return dp.showMarkdown(dp.markdown, dp.truncated)
}

func (dp *DetailPage) handleTodoSaved(msg todoSavedMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	dp.busy = false
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		if errors.Is(msg.err, errPageChanged) {
			dp.inputErr = "The page changed in Notion. Press esc, then r to reload it"
		} else {
			dp.inputErr = "Could not save: " + msg.err.Error()
		}
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)

	cursor := dp.picker.Cursor()
	dp.storeInCache(msg.markdown, msg.truncated)
	cmd := dp.showMarkdown(msg.markdown, msg.truncated)

	dp.syncTodos()
	dp.picker.SetCursor(cursor)
	return cmd
}

// syncTodos rebuilds the to-do list from the page's current content, closing
// the list if no to-dos are left.
func (dp *DetailPage) syncTodos() {
	dp.todos = parseTodos(dp.markdown)
	if len(dp.todos) == 0 {
		dp.setMode(modeView)
		return
	}
	dp.picker.SetItems(todoPickerItems(dp.todos))
}

// renderMode draws the open mode in place of the page view.
func (dp *DetailPage) renderMode() string {
	height := dp.height - 1 // Status bar
	if height < 1 {
		height = 1
	}
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	var footer []string
	if dp.busy {
		footer = append(footer, "", mutedStyle.Render("Saving..."))
	}
	if dp.inputErr != "" {
		footer = append(footer, "", errorStyle.Render(dp.inputErr))
	}

	bold := lipgloss.NewStyle().Bold(true)
	var body string
	switch dp.mode {
	case modeSearch:
		body = lipgloss.JoinVertical(lipgloss.Left, bold.Render("Find in page"), "", dp.searchInput.View())
	case modeConfirm:
		body = lipgloss.JoinVertical(lipgloss.Left,
			bold.Render(fmt.Sprintf("Archive %q?", dp.Title())), "",
			"The page moves to Notion's trash, where it can be restored.", "",
			"y: archive    n: keep")
	case modeDate:
		lines := []string{bold.Render("Change date"), "", dp.dateInput.View()}
		if dp.dateComplex {
			lines = append(lines, "", mutedStyle.Render(
				"Now: "+dp.editing.Text+". Saving a different date replaces the time and end date."))
		}
		body = lipgloss.JoinVertical(lipgloss.Left, lines...)
	default:
		body = dp.picker.View(dp.width-4, height-2-len(footer))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, append([]string{body}, footer...)...)
	return lipgloss.NewStyle().
		Width(dp.width).
		Height(height).
		MaxWidth(dp.width).
		MaxHeight(height).
		Padding(1, 2).
		Render(content)
}

// writePageMarkdown saves an edited page. Only the changed parts are sent,
// so untouched blocks keep their identity; the whole page is replaced only
// when the change cannot be expressed as separate edits, and only if the
// page is still what the edit started from.
func writePageMarkdown(ctx context.Context, client NotionClient, pageID, original, edited string) (*notion.PageMarkdown, error) {
	updates, ok := notion.BuildContentUpdates(original, edited)
	if ok && len(updates) == 0 {
		// Nothing changed: the page is already what was asked for.
		return &notion.PageMarkdown{ID: pageID, Markdown: original}, nil
	}
	if ok {
		// Edits locate their text in the page, so they fail on their own if
		// that text changed in Notion meanwhile.
		result, err := client.UpdatePageMarkdown(ctx, pageID, updates)
		if isNoMatchError(err) {
			return nil, errPageChanged
		}
		return result, err
	}

	// A full replace has no such protection: check first that the page is
	// still what the edit started from.
	current, err := client.GetPageMarkdown(ctx, pageID)
	if err != nil {
		return nil, err
	}
	if current.Truncated || notion.NormalizeMarkdown(current.Markdown) != notion.NormalizeMarkdown(original) {
		return nil, errPageChanged
	}
	return client.ReplacePageMarkdown(ctx, pageID, edited)
}
