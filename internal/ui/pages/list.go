package pages

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/keymap"
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
	// more marks the next page of an earlier result ("load more") rather
	// than a fresh first page. epoch is the reload the rows belong to.
	more  bool
	epoch int
	// offline marks rows saved earlier, shown because Notion could not be
	// reached; age is how long ago they were saved.
	offline bool
	age     time.Duration
	// Set for the rows of a view: the result set later pages come from,
	// and the rows already known, so they need not be fetched again.
	viewQueryID string
	viewKnown   map[string]Page
	skipped     int // Rows of the view that could not be read
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

// loadCounter numbers every load of rows in the app. A row list is rebuilt
// when the user switches database, so numbers of its own would start over
// and a slow answer for the old list could pass for one of the new.
var loadCounter atomic.Int64

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
	readOnly     bool
	offline      bool // Showing saved rows: Notion could not be reached
	keys         *keymap.Map

	sort      rowSort
	epoch     int             // Counts reloads, so a late "load more" of an older list is dropped
	prompt    textinput.Model // Title prompt for a new row
	prompting bool
	creating  bool
	notice    string // Result of the last action, shown in the status bar

	// Filtering rows by a property value
	filter        notion.RowFilter        // Applied by Notion to every load
	filterStage   filterStage             // Which picker is open, if any
	filterChoices []notion.FilterProperty // What the open picker chooses from
	filterProp    int                     // Property chosen in the first picker
	picker        components.Picker
	now           func() time.Time // Clock for date conditions

	// Showing the rows of a view saved in Notion, instead of a filter
	view        notion.View     // The view in effect; no ID means none
	viewChoices []notion.View   // What the view picker chooses from
	viewQueryID string          // Notion's result set for the view, for "load more"
	viewKnown   map[string]Page // Rows already loaded, by ID; replaced, never changed
}

// filterStage is a step of choosing a property filter.
type filterStage int

const (
	filterOff       filterStage = iota
	filterLoading               // Waiting for the database's properties
	filterProperty              // Choosing the property
	filterCondition             // Choosing the condition for it
	viewLoading                 // Waiting for the database's views
	viewChoosing                // Choosing a view
)

// filterSchemaMsg carries the properties rows can be filtered by.
type filterSchemaMsg struct {
	databaseID string
	properties []notion.FilterProperty
	err        error
}

func (filterSchemaMsg) listResult() {}

// viewsLoadedMsg carries the views saved on the database.
type viewsLoadedMsg struct {
	databaseID string
	views      []notion.View
	err        error
}

func (viewsLoadedMsg) listResult() {}

// NewListPageInput contains the parameters for creating a new ListPage.
type NewListPageInput struct {
	Width        int
	Height       int
	NotionClient NotionClient
	Cache        *cache.PageCache
	DatabaseID   string
	// ReadOnly turns off creating rows.
	ReadOnly bool
	// Keys names the keys in hints; nil means the default keys.
	Keys *keymap.Map
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
	statusBar.SetHelpText(input.Keys.Label("{row_filter}: filter | {row_reload}: refresh | {help}: help"))

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
		readOnly:     input.ReadOnly,
		now:          time.Now,
		keys:         input.Keys,
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
		// Rows of a list that was reloaded since (another filter, a
		// refresh) are out of date, whichever answer arrives last
		if msg.epoch != lp.epoch {
			return lp, nil
		}
		if msg.err != nil && msg.more {
			// The rows on screen are still good: say what went wrong and
			// leave them. "Load more" can be tried again.
			lp.loadingMore = false
			lp.statusBar.SetSyncStatus(components.StatusError)
			lp.notice = "Could not load more rows: " + msg.err.Error()
			if errors.Is(msg.err, errViewExpired) {
				lp.hasMore = false
				lp.notice = lp.keys.Label("Notion keeps a view's rows for 15 minutes: reload with {row_reload} to load more")
			}
			lp.updateStatusBarFromSidebar()
			return lp, nil
		}
		if msg.err != nil {
			lp.loading = false
			lp.loadingMore = false
			lp.err = msg.err
			lp.statusBar.SetSyncStatus(components.StatusError)
			if lp.filterStage == filterOff {
				// An open picker keeps its own hints; the error is on
				// screen as soon as it closes
				lp.statusBar.SetHelpText(fmt.Sprintf("Error: %v", msg.err))
			}
			return lp, nil
		}

		lp.err = nil
		lp.loading = false
		lp.loadingMore = false

		if msg.more {
			lp.pageList = append(lp.pageList, msg.pages...)
		} else {
			lp.pageList = msg.pages
		}

		lp.hasMore = msg.hasMore
		lp.nextCursor = msg.nextCursor
		if !msg.more {
			lp.viewQueryID = msg.viewQueryID
		}
		if !msg.more || msg.viewKnown != nil {
			lp.viewKnown = msg.viewKnown
		}
		lp.updateSidebarItems()
		lp.statusBar.SetSyncStatus(components.StatusSynced)
		lp.statusBar.SetMode(components.ModeBrowse)
		if lp.offline && !msg.offline {
			lp.notice = "" // "Offline: ..." is no longer true
		}
		lp.offline = msg.offline
		if msg.offline {
			lp.statusBar.SetMode(components.ModeOffline)
			lp.notice = lp.keys.Label("Offline: these are the rows saved " + components.HumanAge(msg.age) + " ago | {row_reload}: try again")
		}
		if msg.skipped > 0 {
			lp.notice = fmt.Sprintf("%d rows of the view could not be read and are left out", msg.skipped)
		}
		lp.updateStatusBarFromSidebar()
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

	case filterSchemaMsg:
		if msg.databaseID != lp.databaseID || lp.filterStage != filterLoading {
			return lp, nil // The user moved on before the answer came
		}
		lp.filterStage = filterOff
		switch {
		case msg.err != nil:
			lp.notice = "Could not load the properties: " + msg.err.Error()
		case len(msg.properties) == 0:
			lp.notice = "No status, select, checkbox or date property to filter by"
		default:
			lp.filterChoices = msg.properties
			lp.openPropertyPicker()
			return lp, nil
		}
		lp.updateStatusBarFromSidebar()
		return lp, nil

	case viewsLoadedMsg:
		if msg.databaseID != lp.databaseID || lp.filterStage != viewLoading {
			return lp, nil // The user moved on before the answer came
		}
		lp.filterStage = filterOff
		switch {
		case msg.err != nil:
			lp.notice = "Could not load the views: " + msg.err.Error()
		case len(msg.views) == 0:
			lp.notice = "This database has no views that list its rows"
		default:
			lp.viewChoices = msg.views
			lp.openViewPicker()
			return lp, nil
		}
		lp.updateStatusBarFromSidebar()
		return lp, nil

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
		if lp.filterStage != filterOff {
			return lp, lp.updateFilterPicker(msg)
		}
		// While the filter is being typed, letters are filter text
		if lp.CapturesInput() {
			break
		}
		lp.notice = ""
		switch msg.String() {
		case "n":
			return lp, lp.startNewRow()

		case "p":
			return lp, lp.startPropertyFilter()

		case "P":
			if (lp.filter.Active() || lp.view.ID != "") && !lp.loading {
				return lp, lp.applyFilter(notion.RowFilter{})
			}
			return lp, nil

		case "v":
			return lp, lp.startViews()

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
	// An open picker takes every key, so it must always be the thing shown
	if lp.filterStage == filterProperty || lp.filterStage == filterCondition || lp.filterStage == viewChoosing {
		return lipgloss.JoinVertical(lipgloss.Left, lp.picker.View(lp.width, lp.height-1), lp.statusBar.View())
	}

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

		text := fmt.Sprintf("Error loading pages:\n%v", lp.err)
		if lp.filter.Active() {
			// The filter may be the cause (an option renamed since)
			text += fmt.Sprintf("\n\nRows are limited to: %s.", lp.filter.Label) +
				lp.keys.Label(" Press {row_clear_filter} to show all rows")
		}
		if lp.view.ID != "" {
			text += fmt.Sprintf("\n\nShowing the view %s.", lp.view.Name) +
				lp.keys.Label(" Press {row_reload} to try again, {row_clear_filter} to show all rows")
		}
		main := errorStyle.Render(text)
		status := lp.statusBar.View()

		return lipgloss.JoinVertical(lipgloss.Left, main, status)
	}

	status := lp.statusBar.View()

	if len(lp.pageList) == 0 && lp.filter.Active() {
		emptyStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-1).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#6B7280"))
		text := fmt.Sprintf("No rows where %s.", lp.filter.Label) +
			lp.keys.Label(" Press {row_clear_filter} to show all rows")
		return lipgloss.JoinVertical(lipgloss.Left, emptyStyle.Render(text), status)
	}

	if len(lp.pageList) == 0 && lp.view.ID != "" && !lp.prompting {
		emptyStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-1).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#6B7280"))
		text := fmt.Sprintf("The view %s has no rows.", lp.view.Name) +
			lp.keys.Label(" Press {row_clear_filter} to show all rows")
		return lipgloss.JoinVertical(lipgloss.Left, emptyStyle.Render(text), status)
	}

	if len(lp.pageList) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-1).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#6B7280"))
		empty := emptyStyle.Render(lp.keys.Label("This database has no pages. Press {row_new} to add one"))
		if lp.prompting {
			// The prompt must be visible here too, or the user types blind
			emptyStyle = emptyStyle.Height(lp.height - 2)
			prompt := lipgloss.NewStyle().MaxWidth(lp.width).Render(lp.prompt.View())
			return lipgloss.JoinVertical(lipgloss.Left, emptyStyle.Render("This database has no pages"), prompt, status)
		}
		return lipgloss.JoinVertical(lipgloss.Left, empty, status)
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
	if lp.readOnly {
		lp.notice = ReadOnlyNotice
		lp.statusBar.SetHelpText(ReadOnlyNotice)
		return nil
	}
	if lp.offline {
		lp.notice = lp.keys.Label("Offline: reload with {row_reload} before adding a row")
		lp.statusBar.SetHelpText(lp.notice)
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
// InMode reports whether the new-row prompt is open. The root model keeps
// the command palette closed then.
func (lp *ListPage) InMode() bool {
	return lp.prompting || lp.filterStage != filterOff
}

func (lp *ListPage) CapturesInput() bool {
	return lp.prompting || lp.filterStage != filterOff || lp.sidebar.FilterState() == list.Filtering
}

// startPropertyFilter loads the database's properties and then offers
// them to filter the rows by.
func (lp *ListPage) startPropertyFilter() tea.Cmd {
	if lp.loading || lp.creating || lp.notionClient == nil {
		return nil
	}
	lp.filterStage = filterLoading
	lp.statusBar.SetHelpText("Loading properties... | esc: cancel")

	client, databaseID := lp.notionClient, lp.databaseID
	return func() tea.Msg {
		database, err := client.GetDatabase(context.Background(), databaseID)
		if err != nil {
			return filterSchemaMsg{databaseID: databaseID, err: err}
		}
		return filterSchemaMsg{databaseID: databaseID, properties: notion.FilterProperties(database)}
	}
}

func (lp *ListPage) openPropertyPicker() {
	items := make([]components.PickerItem, len(lp.filterChoices))
	for i, property := range lp.filterChoices {
		items[i] = components.PickerItem{Label: property.Name, Detail: string(property.Type)}
	}
	lp.picker = components.NewPicker("Show only rows where...", items)
	lp.picker.SetCursor(lp.filterProp)
	lp.filterStage = filterProperty
	lp.statusBar.SetHelpText("↑/↓: move | enter: choose | esc: cancel")
}

func (lp *ListPage) openConditionPicker() {
	property := lp.filterChoices[lp.filterProp]
	items := make([]components.PickerItem, len(property.Choices))
	cursor := 0
	for i, choice := range property.Choices {
		items[i] = components.PickerItem{Label: choice.Choice}
		if choice.Label == lp.filter.Label {
			cursor = i
		}
	}
	lp.picker = components.NewPicker(property.Name+"...", items)
	lp.picker.SetCursor(cursor)
	lp.filterStage = filterCondition
	lp.statusBar.SetHelpText("↑/↓: move | enter: apply | esc: back")
}

// updateFilterPicker handles a key while a filter picker is open.
func (lp *ListPage) updateFilterPicker(msg tea.KeyMsg) tea.Cmd {
	if lp.filterStage == filterLoading || lp.filterStage == viewLoading {
		if msg.String() == "esc" {
			lp.filterStage = filterOff
			lp.updateStatusBarFromSidebar()
		}
		return nil
	}
	if lp.picker.HandleKey(msg) {
		return nil
	}
	switch msg.String() {
	case "esc":
		if lp.filterStage == filterCondition {
			lp.openPropertyPicker()
			return nil
		}
		lp.filterStage = filterOff
		lp.updateStatusBarFromSidebar()
	case "enter":
		if lp.filterStage == filterProperty {
			lp.filterProp = lp.picker.Cursor()
			lp.openConditionPicker()
			return nil
		}
		if lp.filterStage == viewChoosing {
			lp.filterStage = filterOff
			return lp.applyView(lp.viewChoices[lp.picker.Cursor()])
		}
		lp.filterStage = filterOff
		return lp.applyFilter(lp.filterChoices[lp.filterProp].Choices[lp.picker.Cursor()])
	}
	return nil
}

// applyFilter reloads the rows with a filter, or without one for the zero
// value.
func (lp *ListPage) applyFilter(filter notion.RowFilter) tea.Cmd {
	lp.filter = filter
	lp.view = notion.View{} // A view and a filter of ours do not combine
	// The heading says it too: the status bar drops its text when narrow
	title := "Pages"
	if filter.Active() {
		title = "Pages where " + filter.Label
	}
	lp.sidebar.SetTitle(title)
	lp.loading = true
	lp.hasMore = false
	lp.statusBar.SetSyncStatus(components.StatusSyncing)
	lp.statusBar.SetHelpText("Loading rows...")
	return lp.fetchPagesCmd()
}

// startViews loads the views saved on the database and then offers them.
func (lp *ListPage) startViews() tea.Cmd {
	if lp.loading || lp.creating || lp.notionClient == nil {
		return nil
	}
	lp.filterStage = viewLoading
	lp.statusBar.SetHelpText("Loading views... | esc: cancel")

	client, databaseID := lp.notionClient, lp.databaseID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), viewTimeout)
		defer cancel()
		views, err := client.ListViews(ctx, databaseID)
		for i := range views {
			// Names are the user's text, and are drawn as they are
			views[i].Name = oneLine(views[i].Name)
			if views[i].Name == "" {
				views[i].Name = "Untitled view"
			}
			views[i].Type = oneLine(views[i].Type)
		}
		return viewsLoadedMsg{databaseID: databaseID, views: views, err: err}
	}
}

func (lp *ListPage) openViewPicker() {
	items := make([]components.PickerItem, len(lp.viewChoices))
	cursor := 0
	for i, view := range lp.viewChoices {
		items[i] = components.PickerItem{Label: view.Name, Detail: view.Type}
		if view.ID == lp.view.ID {
			cursor = i
		}
	}
	lp.picker = components.NewPicker("Show the rows of the view...", items)
	lp.picker.SetCursor(cursor)
	lp.filterStage = viewChoosing
	lp.statusBar.SetHelpText("↑/↓: move | enter: show | esc: cancel")
}

// applyView reloads the rows as a view saved in Notion shows them: the
// rows it filters to, in its order.
func (lp *ListPage) applyView(view notion.View) tea.Cmd {
	lp.view = view
	lp.filter = notion.RowFilter{}
	lp.offline = false // A view is never shown from a saved copy
	lp.sidebar.SetTitle("Pages in the view " + view.Name)
	lp.loading = true
	lp.hasMore = false
	lp.statusBar.SetSyncStatus(components.StatusSyncing)
	lp.statusBar.SetHelpText("Loading the view...")
	return lp.fetchPagesCmd()
}

// ShownView returns the Notion view whose rows are shown; its ID is empty when
// none is.
func (lp *ListPage) ShownView() notion.View {
	return lp.view
}

// viewTimeout bounds loading a view or the list of views. It is longer
// than for other reads: rows a view names that are not among the
// database's first rows are fetched one by one.
const viewTimeout = 90 * time.Second

// rowKey makes a row ID comparable however it is written.
func rowKey(id string) string {
	return strings.ToLower(strings.ReplaceAll(id, "-", ""))
}

// viewRowsCmd loads the first page of the rows of the view in effect.
func (lp *ListPage) viewRowsCmd(epoch int) tea.Cmd {
	client, databaseID, view := lp.notionClient, lp.databaseID, lp.view
	return func() tea.Msg {
		if client == nil {
			return pagesLoadedMsg{epoch: epoch, err: fmt.Errorf("notion client not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), viewTimeout)
		defer cancel()

		result, err := client.QueryView(ctx, view.ID)
		if err != nil {
			return pagesLoadedMsg{epoch: epoch, err: fmt.Errorf("fetch pages: %w", err)}
		}

		// The view only names its rows. Most are among the database's
		// first rows, which one request brings; the rest are fetched one
		// by one.
		known := map[string]Page{}
		if len(result.RowIDs) > 0 {
			resp, err := client.QueryDatabase(ctx, databaseID, nil)
			if err != nil {
				return pagesLoadedMsg{epoch: epoch, err: fmt.Errorf("fetch pages: %w", err)}
			}
			for i := range resp.Results {
				row := rowFromPage(&resp.Results[i])
				known[rowKey(row.ID)] = row
			}
		}
		rows, known, skipped, err := viewRows(ctx, client, result.RowIDs, known)
		if err != nil {
			return pagesLoadedMsg{epoch: epoch, err: fmt.Errorf("fetch pages: %w", err)}
		}
		return pagesLoadedMsg{
			epoch: epoch, pages: rows, skipped: skipped,
			// Later pages are asked for by the result's ID
			hasMore: result.HasMore && result.NextCursor != "" && result.QueryID != "", nextCursor: result.NextCursor,
			viewQueryID: result.QueryID, viewKnown: known,
		}
	}
}

// viewRows turns the row IDs of a view into rows, in the same order. A
// row not in known is fetched; the returned map is known plus those rows,
// as a new map, since known may be in use elsewhere. A row Notion will
// not hand out (not shared with the connection, or gone since the view
// ran) is left out and counted in skipped.
func viewRows(ctx context.Context, client NotionClient, ids []string, known map[string]Page) (rows []Page, all map[string]Page, skipped int, err error) {
	all = make(map[string]Page, len(known)+len(ids))
	for key, row := range known {
		all[key] = row
	}
	rows = make([]Page, 0, len(ids))
	for _, id := range ids {
		row, ok := all[rowKey(id)]
		if !ok {
			page, err := client.GetPage(ctx, id)
			var apiErr *notionapi.Error
			if errors.As(err, &apiErr) && (apiErr.Status == http.StatusNotFound || apiErr.Status == http.StatusForbidden) {
				skipped++
				continue
			}
			if err != nil {
				return nil, nil, 0, err
			}
			row = rowFromPage(page)
			all[rowKey(id)] = row
		}
		rows = append(rows, row)
	}
	return rows, all, skipped, nil
}

// errViewExpired says Notion no longer has the rows it prepared for a view.
var errViewExpired = errors.New("the rows Notion prepared for this view have expired")

// viewMoreCmd loads the next page of the rows of the view in effect.
func (lp *ListPage) viewMoreCmd() tea.Cmd {
	client, view, epoch := lp.notionClient, lp.view, lp.epoch
	queryID, cursor, known := lp.viewQueryID, lp.nextCursor, lp.viewKnown
	return func() tea.Msg {
		if client == nil {
			return pagesLoadedMsg{more: true, epoch: epoch, err: fmt.Errorf("notion client not initialized")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), viewTimeout)
		defer cancel()

		result, err := client.ViewRowsAfter(ctx, view.ID, queryID, cursor)
		if err != nil {
			var apiErr *notionapi.Error
			if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
				// Notion keeps the rows of a view for 15 minutes
				err = errViewExpired
			}
			return pagesLoadedMsg{more: true, epoch: epoch, err: fmt.Errorf("fetch more pages: %w", err)}
		}
		rows, known, skipped, err := viewRows(ctx, client, result.RowIDs, known)
		if err != nil {
			return pagesLoadedMsg{more: true, epoch: epoch, err: fmt.Errorf("fetch more pages: %w", err)}
		}
		return pagesLoadedMsg{
			more: true, epoch: epoch, pages: rows, skipped: skipped, viewKnown: known,
			hasMore: result.HasMore && result.NextCursor != "", nextCursor: result.NextCursor,
		}
	}
}

// Offline reports whether the rows on screen are a saved copy, shown
// because Notion could not be reached.
func (lp *ListPage) Offline() bool {
	return lp.offline
}

// Filter returns the property filter in effect.
func (lp *ListPage) Filter() notion.RowFilter {
	return lp.filter
}

// SetPages updates the page list and sidebar items.
func (lp *ListPage) SetPages(pages []Page) {
	lp.pageList = pages
	lp.updateSidebarItems()
	lp.statusBar.SetHelpText(fmt.Sprintf("%d pages | %s: refresh", len(lp.pageList), lp.keys.Key("row_reload")))
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
	// A fresh first page starts a new list: anything still loading for the
	// old one no longer applies.
	lp.epoch = int(loadCounter.Add(1))
	lp.loadingMore = false
	lp.nextCursor = ""
	filter, epoch, now := lp.filter, lp.epoch, lp.now
	if lp.view.ID != "" {
		return lp.viewRowsCmd(epoch)
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), notion.ReadTimeout)
		defer cancel()

		if lp.notionClient == nil {
			return pagesLoadedMsg{
				epoch: epoch,
				err:   fmt.Errorf("notion client not initialized"),
			}
		}

		resp, err := lp.notionClient.QueryDatabase(ctx, lp.databaseID, filter.Query("", now()))
		if err != nil {
			// Without a connection, the rows saved earlier are better than
			// an error. Only the unfiltered list is saved.
			// (read with its own context: the load's may be what ran out)
			if saved, age, ok := lp.savedRows(context.Background()); ok && notion.IsOffline(err) && !filter.Active() {
				return pagesLoadedMsg{epoch: epoch, pages: saved, offline: true, age: age}
			}
			return pagesLoadedMsg{
				epoch: epoch,
				err:   fmt.Errorf("fetch pages: %w", err),
			}
		}

		pages := make([]Page, 0, len(resp.Results))
		for _, p := range resp.Results {
			pages = append(pages, rowFromPage(&p))
		}

		if !filter.Active() {
			lp.saveRows(ctx, pages)
		}

		return pagesLoadedMsg{
			epoch:      epoch,
			pages:      pages,
			hasMore:    resp.HasMore,
			nextCursor: string(resp.NextCursor),
		}
	}
}

// savedRowsFormat marks cache entries holding a database's rows.
const savedRowsFormat = "rows-v1"

// savedRowList is the first page of a database's rows as kept in the
// cache for reading without a connection.
type savedRowList struct {
	Format string `json:"format"`
	Rows   []Page `json:"rows"`
}

func (lp *ListPage) rowsCacheKey() string {
	return "rows:" + lp.databaseID
}

func (lp *ListPage) saveRows(ctx context.Context, rows []Page) {
	if lp.cache == nil {
		return
	}
	_ = lp.cache.Set(ctx, cache.SetInput{PageID: lp.rowsCacheKey(), Data: savedRowList{Format: savedRowsFormat, Rows: rows}})
}

func (lp *ListPage) savedRows(ctx context.Context) (rows []Page, age time.Duration, ok bool) {
	if lp.cache == nil {
		return nil, 0, false
	}
	var saved savedRowList
	age, _, err := lp.cache.Load(ctx, lp.rowsCacheKey(), &saved)
	if err != nil || saved.Format != savedRowsFormat {
		return nil, 0, false
	}
	return saved.Rows, age, true
}

// loadMoreCmd returns a command that fetches the next page of results.
func (lp *ListPage) loadMoreCmd() tea.Cmd {
	if lp.view.ID != "" {
		return lp.viewMoreCmd()
	}
	epoch := lp.epoch
	cursor := lp.nextCursor
	filter, now := lp.filter, lp.now
	return func() tea.Msg {
		ctx := context.Background()

		if lp.notionClient == nil {
			return pagesLoadedMsg{
				more:  true,
				epoch: epoch,
				err:   fmt.Errorf("notion client not initialized"),
			}
		}

		if cursor == "" {
			return pagesLoadedMsg{
				more:  true,
				epoch: epoch,
				err:   fmt.Errorf("no cursor available for pagination"),
			}
		}

		// The next page of the same, possibly filtered, list
		req := filter.Query(cursor, now())

		resp, err := lp.notionClient.QueryDatabase(ctx, lp.databaseID, req)
		if err != nil {
			return pagesLoadedMsg{
				more:  true,
				epoch: epoch,
				err:   fmt.Errorf("fetch more pages: %w", err),
			}
		}

		pages := make([]Page, 0, len(resp.Results))
		for _, p := range resp.Results {
			pages = append(pages, rowFromPage(&p))
		}

		return pagesLoadedMsg{
			more:       true,
			epoch:      epoch,
			pages:      pages,
			hasMore:    resp.HasMore,
			nextCursor: string(resp.NextCursor),
		}
	}
}

// updateSidebarItems converts the page list to sidebar items.
func (lp *ListPage) updateSidebarItems() {
	// Keep the cursor on the same row when the list is reloaded or resorted
	selected := lp.sidebar.SelectedID()
	defer func() { lp.sidebar.SelectID(selected) }()

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
	case lp.prompting || lp.creating || lp.filterStage != filterOff:
		// The prompt, the create action and the pickers set their own text.
	case lp.sidebar.IsFiltering():
		filter := lp.sidebar.FilterValue()
		visible := lp.sidebar.VisibleItemCount()
		total := len(lp.pageList)
		lp.statusBar.SetHelpText(fmt.Sprintf("Filter: %s | %d/%d pages | ESC: clear", filter, visible, total))
	case lp.notice != "":
		lp.statusBar.SetHelpText(lp.notice)
	case !lp.loading && !lp.loadingMore:
		helpText := fmt.Sprintf("%d pages | enter: open | ", len(lp.pageList)) +
			lp.keys.Label(fmt.Sprintf("{row_new}: new | {row_sort}: sort (%s) | {row_filter}: filter | {row_property_filter}: by property | {row_view}: views | {row_reload}: refresh", lp.sort))
		if lp.filter.Active() {
			// The filter comes first: it explains why rows are missing.
			// Its label holds property and option names, which are the
			// user's, so only the hints go through Label.
			helpText = fmt.Sprintf("%d pages where %s", len(lp.pageList), lp.filter.Label) +
				lp.keys.Label(" | {row_clear_filter}: show all | {row_property_filter}: change | enter: open")
		}
		if lp.view.ID != "" {
			// The view's name is the user's text, so only the hints go
			// through Label.
			helpText = fmt.Sprintf("%d pages in the view %s", len(lp.pageList), lp.view.Name) +
				lp.keys.Label(" | {row_clear_filter}: show all | {row_view}: other view | enter: open")
		}
		if lp.hasMore {
			helpText += lp.keys.Label(" | {row_more}: load more")
		}
		lp.statusBar.SetHelpText(helpText)
	}
}
