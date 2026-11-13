package pages

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// Page represents a Notion page in the UI.
type Page struct {
	ID        string
	Title     string
	Status    string
	UpdatedAt time.Time
	// Summary is a one-line digest of the row's properties.
	Summary string
}

// NewPage creates a new Page instance.
func NewPage(id string, title string, status string, updatedAt time.Time) Page {
	return Page{
		ID:        id,
		Title:     title,
		Status:    status,
		UpdatedAt: updatedAt,
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

// pagesLoadedMsg is sent when pages are fetched from the database.
type pagesLoadedMsg struct {
	pages      []Page
	hasMore    bool
	nextCursor string
	err        error
}

// rowCreatedMsg is the result of creating a new row.
type rowCreatedMsg struct {
	id  string
	err error
}

// ListResultMsg marks the results of work a ListPage started. The root model
// delivers these to the list page even when another page is in front, so
// rows loaded in the background are not handed to the wrong page.
type ListResultMsg interface {
	listResult()
}

func (pagesLoadedMsg) listResult() {}
func (rowCreatedMsg) listResult()  {}

// rowSort is the order rows are shown in.
type rowSort int

const (
	sortNotion  rowSort = iota // As Notion returns them
	sortTitle                  // Title, A to Z
	sortUpdated                // Most recently edited first
)

func (s rowSort) String() string {
	switch s {
	case sortTitle:
		return "title"
	case sortUpdated:
		return "last edited"
	default:
		return "Notion order"
	}
}

// summaryProperties is how many property values a row shows under its title.
const summaryProperties = 3

// ListPage wraps the Sidebar component with page listing logic.
type ListPage struct {
	sidebar      components.Sidebar
	statusBar    components.StatusBar
	spinner      components.Spinner
	pageList     []Page
	selectedIdx  int
	loading      bool
	loadingMore  bool
	hasMore      bool
	nextCursor   string
	err          error
	width        int
	height       int
	notionClient NotionClient
	cache        *cache.PageCache
	databaseID   string

	sort      rowSort
	prompt    textinput.Model // Title prompt for a new row
	prompting bool
	creating  bool
	notice    string // Result of the last action, shown in the status bar
}

// NewListPageInput contains the parameters for creating a new ListPage.
type NewListPageInput struct {
	Width        int
	Height       int
	NotionClient NotionClient
	Cache        *cache.PageCache
	DatabaseID   string
}

// NewListPage creates a new ListPage instance.
func NewListPage(input NewListPageInput) ListPage {
	// The row list fills the page; it starts empty until rows are loaded
	sidebar := components.NewSidebar(components.NewSidebarInput{
		Items:  []components.Item{},
		Width:  input.Width,
		Height: listHeight(input.Height),
		Title:  "Pages",
	})

	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("/: search | r: refresh | ?: help")

	spinner := components.NewSpinner("Loading pages...")

	return ListPage{
		sidebar:      sidebar,
		statusBar:    statusBar,
		spinner:      spinner,
		pageList:     []Page{},
		selectedIdx:  -1,
		loading:      true,
		loadingMore:  false,
		hasMore:      false,
		nextCursor:   "",
		err:          nil,
		width:        input.Width,
		height:       input.Height,
		notionClient: input.NotionClient,
		cache:        input.Cache,
		databaseID:   input.DatabaseID,
	}
}

// Init fetches pages from the database on initialization.
func (lp *ListPage) Init() tea.Cmd {
	return tea.Batch(lp.spinner.Init(), lp.fetchPagesCmd())
}

// Update handles messages and returns the updated model and command.
func (lp *ListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pagesLoadedMsg:
		if msg.err != nil {
			lp.loading = false
			lp.loadingMore = false
			lp.err = msg.err
			lp.statusBar.SetSyncStatus(components.StatusError)
			lp.statusBar.SetHelpText(fmt.Sprintf("Error: %v", msg.err))
			return lp, nil
		}

		lp.err = nil

		// If loading more, append to existing pages; otherwise replace
		wasLoadingMore := lp.loadingMore
		lp.loading = false
		lp.loadingMore = false

		if wasLoadingMore && len(lp.pageList) > 0 && len(msg.pages) > 0 {
			lp.pageList = append(lp.pageList, msg.pages...)
		} else {
			lp.pageList = msg.pages
		}

		lp.hasMore = msg.hasMore
		lp.nextCursor = msg.nextCursor
		lp.updateSidebarItems()
		lp.statusBar.SetSyncStatus(components.StatusSynced)

		helpText := fmt.Sprintf("%d pages | r: refresh", len(lp.pageList))
		if lp.hasMore {
			helpText += " | m: load more"
		}
		lp.statusBar.SetHelpText(helpText)
		return lp, nil

	case components.ItemSelectedMsg:
		// Emit navigation message when an item is selected. The index in
		// the message is a position in the filtered, sorted list, so the row
		// is looked up by ID.
		lp.selectedIdx = -1
		for index := range lp.pageList {
			if lp.pageList[index].ID == msg.ID {
				lp.selectedIdx = index
				break
			}
		}
		return lp, func() tea.Msg {
			return NewNavigationMsg(msg.ID)
		}

	case rowCreatedMsg:
		lp.creating = false
		if msg.err != nil {
			lp.statusBar.SetSyncStatus(components.StatusError)
			lp.notice = "Could not create row: " + msg.err.Error()
			lp.updateStatusBarFromSidebar()
			return lp, nil
		}
		lp.statusBar.SetSyncStatus(components.StatusSynced)
		created := PageCreatedMsg{ID: msg.id}
		// Open the new row, and reload the list so it is there on return
		return lp, tea.Batch(func() tea.Msg { return created }, lp.fetchPagesCmd())

	case tea.KeyMsg:
		if lp.prompting {
			return lp, lp.updatePrompt(msg)
		}
		// While the filter is being typed, letters are filter text
		if lp.CapturesInput() {
			break
		}
		lp.notice = ""
		switch msg.String() {
		case "n":
			return lp, lp.startNewRow()

		case "s":
			lp.sort = (lp.sort + 1) % 3
			lp.updateSidebarItems()
			lp.updateStatusBarFromSidebar()
			return lp, nil

		case "r":
			// Refresh page list
			lp.loading = true
			lp.hasMore = false
			lp.nextCursor = ""
			lp.statusBar.SetSyncStatus(components.StatusSyncing)
			lp.statusBar.SetHelpText("Refreshing...")
			return lp, lp.fetchPagesCmd()

		case "m":
			// Load more pages if available
			if lp.hasMore && !lp.loadingMore {
				lp.loadingMore = true
				lp.spinner.SetMessage("Loading more pages...")
				lp.statusBar.SetSyncStatus(components.StatusSyncing)
				lp.statusBar.SetHelpText("Loading more pages...")
				return lp, lp.loadMoreCmd()
			}
		}

	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		lp.sidebar.SetSize(msg.Width, listHeight(msg.Height))
		lp.statusBar.SetWidth(msg.Width)
		return lp, nil
	}

	// Update spinner when loading
	var cmds []tea.Cmd
	if lp.loading || lp.loadingMore {
		var spinnerCmd tea.Cmd
		lp.spinner, spinnerCmd = lp.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Update sidebar
	var sidebarCmd tea.Cmd
	lp.sidebar, sidebarCmd = lp.sidebar.Update(msg)
	cmds = append(cmds, sidebarCmd)

	// Update status bar based on sidebar search state
	lp.updateStatusBarFromSidebar()

	return lp, tea.Batch(cmds...)
}

// View renders the list page.
func (lp *ListPage) View() string {
	if lp.loading {
		loadingStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-2).
			Align(lipgloss.Center, lipgloss.Center)

		spinnerView := lp.spinner.View()
		if lp.loadingMore {
			spinnerView = lp.spinner.View() + fmt.Sprintf(" (%d loaded)", len(lp.pageList))
		}

		main := loadingStyle.Render(spinnerView)
		status := lp.statusBar.View()

		return lipgloss.JoinVertical(lipgloss.Left, main, status)
	}

	if lp.err != nil {
		errorStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-2).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#EF4444"))

		main := errorStyle.Render(fmt.Sprintf("Error loading pages:\n%v", lp.err))
		status := lp.statusBar.View()

		return lipgloss.JoinVertical(lipgloss.Left, main, status)
	}

	status := lp.statusBar.View()

	if len(lp.pageList) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-1).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#6B7280"))
		return lipgloss.JoinVertical(lipgloss.Left, emptyStyle.Render("This database has no pages"), status)
	}

	// The database's rows: move with j/k, open with enter, filter with /.
	// The list has a minimum size of its own (title, counters, help), so it
	// is cut to the page: on a small terminal an oversized view would push
	// the top of the screen out of sight.
	rowsHeight := lp.height - 1
	var promptLine string
	if lp.prompting {
		rowsHeight--
		promptLine = lipgloss.NewStyle().MaxWidth(lp.width).Render(lp.prompt.View())
	}
	rows := lipgloss.NewStyle().
		MaxWidth(lp.width).
		MaxHeight(rowsHeight).
		Render(lp.sidebar.View())
	if lp.prompting {
		return lipgloss.JoinVertical(lipgloss.Left, rows, promptLine, status)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows, status)
}

// startNewRow opens the title prompt for a new row.
func (lp *ListPage) startNewRow() tea.Cmd {
	if lp.loading || lp.creating {
		return nil
	}
	input := textinput.New()
	input.Prompt = "New row title: "
	input.CharLimit = 200
	input.Focus()
	lp.prompt = input
	lp.prompting = true
	lp.statusBar.SetHelpText("enter: create | esc: cancel")
	return textinput.Blink
}

// updatePrompt handles a key while the new-row prompt is open.
func (lp *ListPage) updatePrompt(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		lp.prompting = false
		lp.updateStatusBarFromSidebar()
		return nil
	case "enter":
		title := strings.TrimSpace(lp.prompt.Value())
		lp.prompting = false
		if title == "" {
			lp.updateStatusBarFromSidebar()
			return nil
		}
		lp.creating = true
		lp.statusBar.SetSyncStatus(components.StatusSyncing)
		lp.statusBar.SetHelpText("Creating row...")

		client := lp.notionClient
		parent := notion.PageParent{DatabaseID: lp.databaseID}
		// The first heading of the Markdown becomes the row's title
		markdown := "# " + notion.EscapeMarkdown(title)
		return func() tea.Msg {
			created, err := client.CreatePageMarkdown(context.Background(), parent, markdown)
			if err != nil {
				return rowCreatedMsg{err: err}
			}
			return rowCreatedMsg{id: created.ID}
		}
	}
	var cmd tea.Cmd
	lp.prompt, cmd = lp.prompt.Update(msg)
	return cmd
}

// listChrome is the number of rows around the row list: the page status bar
// plus the list's own top and bottom padding.
const listChrome = 3

// listHeight returns the height available to the row list itself.
func listHeight(pageHeight int) int {
	height := pageHeight - listChrome
	if height < 1 {
		return 1
	}
	return height
}

// CapturesInput reports whether the row filter is being typed, in which
// case key presses are filter text rather than shortcuts.
func (lp *ListPage) CapturesInput() bool {
	return lp.prompting || lp.sidebar.FilterState() == list.Filtering
}

// SetPages updates the page list and sidebar items.
func (lp *ListPage) SetPages(pages []Page) {
	lp.pageList = pages
	lp.updateSidebarItems()
	lp.statusBar.SetHelpText(fmt.Sprintf("%d pages | r: refresh", len(lp.pageList)))
}

// SelectedPage returns the currently selected page or nil if none selected.
func (lp *ListPage) SelectedPage() *Page {
	if lp.selectedIdx < 0 || lp.selectedIdx >= len(lp.pageList) {
		return nil
	}
	return &lp.pageList[lp.selectedIdx]
}

// Refresh returns a command to refresh the page list.
func (lp *ListPage) Refresh() tea.Cmd {
	return lp.fetchPagesCmd()
}

// fetchPagesCmd returns a command that fetches pages from the database.
func (lp *ListPage) fetchPagesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if lp.notionClient == nil {
			return pagesLoadedMsg{
				err: fmt.Errorf("notion client not initialized"),
			}
		}

		resp, err := lp.notionClient.QueryDatabase(ctx, lp.databaseID, nil)
		if err != nil {
			return pagesLoadedMsg{
				err: fmt.Errorf("fetch pages: %w", err),
			}
		}

		pages := make([]Page, 0, len(resp.Results))
		for _, p := range resp.Results {
			pages = append(pages, rowFromPage(&p))
		}

		return pagesLoadedMsg{
			pages:      pages,
			hasMore:    resp.HasMore,
			nextCursor: string(resp.NextCursor),
		}
	}
}

// loadMoreCmd returns a command that fetches the next page of results.
func (lp *ListPage) loadMoreCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		if lp.notionClient == nil {
			return pagesLoadedMsg{
				err: fmt.Errorf("notion client not initialized"),
			}
		}

		if lp.nextCursor == "" {
			return pagesLoadedMsg{
				err: fmt.Errorf("no cursor available for pagination"),
			}
		}

		// Create pagination request with the next cursor
		req := &notionapi.DatabaseQueryRequest{
			StartCursor: notionapi.Cursor(lp.nextCursor),
		}

		resp, err := lp.notionClient.QueryDatabase(ctx, lp.databaseID, req)
		if err != nil {
			return pagesLoadedMsg{
				err: fmt.Errorf("fetch more pages: %w", err),
			}
		}

		pages := make([]Page, 0, len(resp.Results))
		for _, p := range resp.Results {
			pages = append(pages, rowFromPage(&p))
		}

		return pagesLoadedMsg{
			pages:      pages,
			hasMore:    resp.HasMore,
			nextCursor: string(resp.NextCursor),
		}
	}
}

// updateSidebarItems converts the page list to sidebar items.
func (lp *ListPage) updateSidebarItems() {
	rows := lp.sortedRows()
	items := make([]components.Item, 0, len(rows))
	for _, page := range rows {
		desc := fmt.Sprintf("Updated: %s", formatTime(page.UpdatedAt))
		switch {
		case page.Summary != "":
			desc = fmt.Sprintf("%s | %s", page.Summary, desc)
		case page.Status != "":
			desc = fmt.Sprintf("%s | %s", page.Status, desc)
		}

		item := components.NewItem(page.Title, desc, page.ID)
		items = append(items, item)
	}

	lp.sidebar.SetItems(items)
}

// sortedRows returns the rows in the chosen order without changing the
// order they were loaded in.
func (lp *ListPage) sortedRows() []Page {
	rows := append([]Page(nil), lp.pageList...)
	switch lp.sort {
	case sortTitle:
		sort.SliceStable(rows, func(i, j int) bool {
			return strings.ToLower(rows[i].Title) < strings.ToLower(rows[j].Title)
		})
	case sortUpdated:
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].UpdatedAt.After(rows[j].UpdatedAt)
		})
	}
	return rows
}

// rowFromPage builds a list row from a database page.
func rowFromPage(page *notionapi.Page) Page {
	row := NewPage(string(page.ID), notion.PageTitle(page), extractStatus(page), page.LastEditedTime)
	row.Summary = rowSummary(page)
	return row
}

// rowSummary lists a row's first few set properties: values alone for the
// self-explanatory kinds, "name: value" for the rest.
func rowSummary(page *notionapi.Page) string {
	var parts []string
	for _, property := range notion.PageProperties(page) {
		if len(parts) == summaryProperties {
			break
		}
		switch property.Type {
		case notionapi.PropertyTypeCheckbox:
			if property.Checked {
				parts = append(parts, "☑ "+property.Name)
			}
		case notionapi.PropertyTypeStatus, notionapi.PropertyTypeSelect, notionapi.PropertyTypeDate:
			if property.Text != "" {
				parts = append(parts, property.Text)
			}
		case notionapi.PropertyTypeCreatedTime, notionapi.PropertyTypeLastEditedTime,
			notionapi.PropertyTypeCreatedBy, notionapi.PropertyTypeLastEditedBy:
			// The row already shows when it was updated.
		default:
			if property.Text != "" {
				parts = append(parts, property.Name+": "+property.Text)
			}
		}
	}
	return strings.Join(parts, " · ")
}

// extractTitle extracts the title from a Notion page.
func extractTitle(page *notionapi.Page) string {
	if page == nil {
		return "Untitled"
	}

	// Try to find title property
	for _, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok && len(titleProp.Title) > 0 {
			return titleProp.Title[0].PlainText
		}
	}

	return "Untitled"
}

// extractStatus extracts the status from a Notion page if available.
func extractStatus(page *notionapi.Page) string {
	if page == nil {
		return ""
	}

	// Try to find status property
	for name, prop := range page.Properties {
		if name == "Status" || name == "status" {
			if statusProp, ok := prop.(*notionapi.StatusProperty); ok && statusProp.Status.Name != "" {
				return statusProp.Status.Name
			}
			if selectProp, ok := prop.(*notionapi.SelectProperty); ok && selectProp.Select.Name != "" {
				return selectProp.Select.Name
			}
		}
	}

	return ""
}

// formatTime formats a time as a relative string.
func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		return t.Format("Jan 2, 2006")
	}
}

// HasMore returns whether there are more pages to load.
func (lp *ListPage) HasMore() bool {
	return lp.hasMore
}

// NextCursor returns the cursor for the next page of results.
func (lp *ListPage) NextCursor() string {
	return lp.nextCursor
}

// IsLoadingMore returns whether the page is currently loading more results.
func (lp *ListPage) IsLoadingMore() bool {
	return lp.loadingMore
}

// updateStatusBarFromSidebar updates the status bar based on sidebar search state.
func (lp *ListPage) updateStatusBarFromSidebar() {
	switch {
	case lp.prompting || lp.creating:
		// The prompt and the create action set their own text.
	case lp.sidebar.IsFiltering():
		filter := lp.sidebar.FilterValue()
		visible := lp.sidebar.VisibleItemCount()
		total := len(lp.pageList)
		lp.statusBar.SetHelpText(fmt.Sprintf("Filter: %s | %d/%d pages | ESC: clear", filter, visible, total))
	case lp.notice != "":
		lp.statusBar.SetHelpText(lp.notice)
	case !lp.loading && !lp.loadingMore:
		helpText := fmt.Sprintf("%d pages | enter: open | n: new | s: sort (%s) | /: filter | r: refresh", len(lp.pageList), lp.sort)
		if lp.hasMore {
			helpText += " | m: load more"
		}
		lp.statusBar.SetHelpText(helpText)
	}
}
