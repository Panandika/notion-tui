package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
	"github.com/Panandika/notion-tui/internal/ui/pages"
)

// workspaceTreeMsg is sent when workspace tree data is fetched.
type workspaceTreeMsg struct {
	tree *components.NavTree
	err  error
}

// AppModel represents the root TUI orchestrator that manages pages and global components.
// It implements the Elm architecture pattern with page routing and navigation.
type AppModel struct {
	// Navigation
	currentPage PageID
	pages       map[PageID]tea.Model
	navigator   *Navigator

	// Components (always visible)
	treeView   components.TreeView
	statusBar  components.StatusBar
	cmdPalette components.CommandPalette

	// State
	width        int
	height       int
	showSidebar  bool
	sidebarFocus bool // Whether sidebar has keyboard focus
	showPalette  bool
	mode         ViewMode

	// Services
	notionClient *notion.Client
	cache        *cache.PageCache
	config       *config.Config

	// Data
	pageList     []pages.Page
	ready        bool
	err          error
	selectedPage *pages.Page
	currentDBID  string // Currently active database ID

	// Help state
	showHelp bool

	// Pages visited one after another in the page view, as Notion page IDs.
	// esc and "[" walk back through them, "]" forward again.
	backTrail    []string
	forwardTrail []string
}

// NewModelInput contains the parameters for creating a new AppModel.
type NewModelInput struct {
	Config *config.Config
	Cache  *cache.PageCache
}

// NewModel creates a new root TUI model with page orchestration.
// Initializes the navigator, components, and page registry.
func NewModel(input NewModelInput) AppModel {
	// Initialize services
	notionClient := notion.NewClient(input.Config.NotionToken)

	// Initialize cache if not provided
	cacheInstance := input.Cache
	if cacheInstance == nil {
		var err error
		cacheInstance, err = cache.NewPageCache(cache.NewPageCacheInput{
			Dir: input.Config.CacheDir,
		})
		if err != nil {
			// Fall back to no cache if initialization fails
			cacheInstance = nil
		}
	}

	// Determine initial page based on config
	// Start with Dashboard
	initialPage := PageDashboard

	// Initialize navigator
	nav := NewNavigator(NewNavigatorInput{
		InitialPage: initialPage,
		MaxHistory:  DefaultMaxHistory,
	})

	// Initialize tree view for navigation sidebar
	treeView := components.NewTreeView(components.NewTreeViewInput{
		Title:  "Workspace",
		Width:  25, // Will be adjusted on first WindowSizeMsg
		Height: 20,
	})

	statusBar := components.NewStatusBar()
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("? for help")

	cmdPalette := components.NewCommandPalette()

	return AppModel{
		currentPage:  initialPage,
		pages:        make(map[PageID]tea.Model),
		navigator:    &nav,
		treeView:     treeView,
		statusBar:    statusBar,
		cmdPalette:   cmdPalette,
		width:        0,
		height:       0,
		showSidebar:  true, // Always show sidebar by default
		sidebarFocus: false,
		showPalette:  false,
		mode:         ViewModeBrowse,
		notionClient: notionClient,
		cache:        cacheInstance,
		config:       input.Config,
		pageList:     []pages.Page{},
		ready:        false,
		err:          nil,
		selectedPage: nil,
		currentDBID:  input.Config.GetDatabaseID(),
	}
}

// Init initializes the AppModel and all pages.
// Returns commands to initialize pages and load initial data.
func (m AppModel) Init() tea.Cmd {
	// Initialize all pages
	m.initializePages()

	// Get init command from current page
	var pageInitCmd tea.Cmd
	if page, ok := m.pages[m.currentPage]; ok {
		pageInitCmd = page.Init()
	}

	return tea.Batch(
		pageInitCmd,
		m.cmdPalette.Init(),
		m.fetchWorkspaceTreeCmd(), // Fetch workspace tree on startup
	)
}

// fetchWorkspaceTreeCmd returns a command that fetches the workspace tree.
func (m *AppModel) fetchWorkspaceTreeCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Fetch all workspace items
		resp, err := m.notionClient.Search(ctx, notion.SearchInput{
			PageSize: 100,
		})
		if err != nil {
			return workspaceTreeMsg{err: fmt.Errorf("fetch workspace: %w", err)}
		}

		// Build tree from results
		tree := components.BuildNavTree(components.BuildNavTreeInput{
			Results: resp.Results,
		})

		return workspaceTreeMsg{tree: tree}
	}
}

// initializePages creates and registers all page instances.
func (m *AppModel) initializePages() {
	// If databases are configured, create ListPage
	if m.config.HasDatabases() {
		listPage := pages.NewListPage(pages.NewListPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.config.GetDatabaseID(),
		})
		m.pages[PageList] = &listPage
	}

	// If no databases, create workspace search page as initial view
	if !m.config.HasDatabases() {
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   "",
			Mode:         pages.SearchModeWorkspace,
		})
		m.pages[PageWorkspaceSearch] = &searchPage
	}

	// Create Dashboard page
	dashboardPage := pages.NewDashboardPage(pages.NewDashboardPageInput{
		Width:  m.contentWidth(),
		Height: m.contentHeight(),
		Config: m.config,
	})
	m.pages[PageDashboard] = &dashboardPage

	// DetailPage and EditPage will be created on-demand when navigating
	// This avoids creating pages with invalid state before we have page IDs
}

// Update handles all messages and orchestrates updates across pages and components.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update model dimensions
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update component sizes
		m.treeView.SetSize(m.sidebarWidth(), m.height-1)
		m.statusBar.SetWidth(m.width)

		// Pages draw next to the sidebar, so they get the content area size
		return m, m.resizePages()

	case workspaceTreeMsg:
		// Workspace tree data received
		if msg.err != nil {
			m.treeView.SetError(msg.err)
		} else {
			m.treeView.SetTree(msg.tree)
		}
		return m, nil

	case pages.DetailResultMsg:
		// Results of work the detail page started go to the detail page,
		// whichever page is in front.
		if page, ok := m.pages[PageDetail]; ok {
			updatedPage, cmd := page.Update(msg)
			m.pages[PageDetail] = updatedPage
			return m, cmd
		}
		return m, nil

	case pages.OpenLinkMsg:
		if m.pageBusy() {
			return m, nil
		}
		if msg.Database {
			return m, m.switchDatabase(msg.ID)
		}
		return m, m.navigateToDetail(msg.ID)

	case pages.PageArchivedMsg:
		// The page is gone: leave it, and reload what listed it
		m.statusBar.SetHelpText("Page archived")
		return m, tea.Batch(m.leaveArchivedPage(), m.fetchWorkspaceTreeCmd())

	case pages.ListResultMsg:
		// Rows loaded or created by the list page go to the list page,
		// whichever page is in front.
		if page, ok := m.pages[PageList]; ok {
			updatedPage, cmd := page.Update(msg)
			m.pages[PageList] = updatedPage
			return m, cmd
		}
		return m, nil

	case components.TreeNavigationMsg:
		if m.pageBusy() {
			return m, nil
		}
		// User selected an item from the tree
		if msg.ObjectType == "database" {
			// Switch to this database
			return m, m.switchDatabase(msg.ID)
		}
		// Navigate to page detail
		return m, m.navigateToDetail(msg.ID)

	case tea.KeyMsg:
		// Ctrl+C always quits, whatever has focus
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Check if we're on the search page - it needs special key handling
		isSearchPage := m.currentPage == PageWorkspaceSearch

		// The help screen covers everything; any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// A page that captures input (a focused search box) receives every
		// other key itself.
		capturing := m.pageCapturesInput()

		// Handle mode-specific keys FIRST before global keys
		switch msg.String() {
		case "tab":
			if capturing {
				break
			}
			// Toggle focus between sidebar and main content
			m.sidebarFocus = !m.sidebarFocus
			m.treeView.SetFocused(m.sidebarFocus)
			return m, nil

		case "ctrl+b":
			// Toggle sidebar visibility; the content area changes width with it
			m.showSidebar = !m.showSidebar
			if !m.showSidebar {
				m.sidebarFocus = false
				m.treeView.SetFocused(false)
			}
			return m, m.resizePages()

		case "ctrl+p":
			// Not over a property list, to-do list or prompt: a palette
			// command would change or leave the page underneath it.
			if m.pageInMode() {
				return m, nil
			}
			// Toggle command palette (works everywhere else)
			m.cmdPalette.Toggle()
			m.showPalette = m.cmdPalette.IsOpen()
			return m, nil
		}

		// If palette is open, only palette handles keys (except ctrl+p which is handled above).
		// This runs before global keys so typing "q" in the palette filters instead of quitting.
		if m.showPalette {
			var cmd tea.Cmd
			m.cmdPalette, cmd = m.cmdPalette.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// A capturing page gets the key and nothing else does
		if capturing {
			return m, m.updateCurrentPage(msg)
		}

		// Handle global keys
		if handled, cmd := m.handleGlobalKeys(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// With the sidebar focused, keys drive the tree only. Sending them to
		// the page as well would trigger page actions the user cannot see.
		if m.showSidebar && m.sidebarFocus && msg.String() != "esc" {
			var cmd tea.Cmd
			m.treeView, cmd = m.treeView.Update(msg)
			return m, cmd
		}

		// Handle other mode-specific keys
		switch msg.String() {

		case "[":
			if cmd, ok := m.trailBack(); ok {
				return m, cmd
			}

		case "]":
			if cmd, ok := m.trailForward(); ok {
				return m, cmd
			}

		case "esc":
			// On search page, let it handle Esc first (e.g., to clear filter)
			// then fall through to page delegation which will handle back navigation
			if isSearchPage {
				break // Fall through to page delegation
			}
			if m.pageBusy() {
				return m, nil
			}
			// With the tree focused and nowhere to go back to, esc must not
			// reach the page, where it would act unseen (clearing a filter).
			if m.showSidebar && m.sidebarFocus && !m.navigator.CanGoBack() {
				return m, nil
			}
			// In the page view, first walk back through the pages visited
			if cmd, ok := m.trailBack(); ok {
				return m, cmd
			}
			// Handle back navigation for other pages
			if m.navigator.CanGoBack() {
				previousPage, ok := m.navigator.Back()
				if ok {
					m.currentPage = previousPage
					return m, m.onReturnTo(previousPage)
				}
			}
		}

	case pages.NavigationMsg:
		// A database row was chosen on the list page: its ID is a Notion
		// page ID, so open that page
		if m.pageBusy() {
			return m, nil
		}
		return m, m.navigateToDetail(msg.PageID())

	case components.ContentLoadedMsg, components.ErrorMsg:
		// Rendered content belongs to the detail page's viewer. Deliver it
		// there even if the user has already moved on to the editor.
		if m.currentPage != PageDetail {
			if page, ok := m.pages[PageDetail]; ok {
				updatedPage, cmd := page.Update(msg)
				m.pages[PageDetail] = updatedPage
				return m, cmd
			}
		}

	case pages.PageCreatedMsg:
		// The tree gains an entry either way
		reloadTree := m.fetchWorkspaceTreeCmd()
		// Open the new page only if the user is still where it was created
		// and nothing there is mid-save; otherwise it must not take over.
		if !m.createdWhereUserIs(msg) || m.pageBusy() {
			return m, reloadTree
		}
		return m, tea.Batch(m.navigateToDetail(msg.ID), reloadTree)

	case components.CommandExecutedMsg:
		// Command palette executed a command
		m.showPalette = false
		return m, m.handleCommandExecution(msg)

	case pages.DatabaseSelectedMsg:
		if m.pageBusy() {
			return m, nil
		}
		// User selected a different database
		m.currentDBID = msg.DatabaseID
		// Refresh list page with new database
		return m, m.switchDatabase(msg.DatabaseID)

	case pages.SearchNavigationMsg:
		if m.pageBusy() {
			return m, nil
		}
		// User selected a search result (page or database)
		if msg.ObjectType == "database" {
			// Switch to this database and show its pages
			return m, m.switchDatabase(msg.ID)
		}
		// Navigate to page detail
		return m, m.navigateToDetail(msg.ID)

	case pages.BackNavigationMsg:
		if m.pageBusy() {
			return m, nil
		}
		// Handle back navigation request from search page
		if m.navigator.CanGoBack() {
			previousPage, ok := m.navigator.Back()
			if ok {
				m.currentPage = previousPage
				return m, m.onReturnTo(previousPage)
			}
		}
	}

	// Delegate to current page
	if cmd := m.updateCurrentPage(msg); cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Non-key messages still reach a focused tree view (key presses for the
	// tree were handled above and never get here).
	if _, isKey := msg.(tea.KeyMsg); !isKey && m.showSidebar && m.sidebarFocus {
		var cmd tea.Cmd
		m.treeView, cmd = m.treeView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update status bar
	// Status bar is read-only, no update needed

	// Update command palette
	if m.showPalette {
		var cmd tea.Cmd
		m.cmdPalette, cmd = m.cmdPalette.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the complete UI with all components and current page.
func (m AppModel) View() string {
	if !m.ready {
		return RenderInitializing()
	}

	if m.err != nil {
		return RenderError(m.err)
	}

	if m.showHelp {
		return RenderHelp(m.width, m.height)
	}

	// Get current page view
	var pageView string
	if page, ok := m.pages[m.currentPage]; ok {
		pageView = page.View()
	} else {
		pageView = fmt.Sprintf("Page '%s' not found", m.currentPage)
	}

	// Compose with sidebar if visible (on all pages)
	var mainContent string
	if m.showSidebar {
		sidebarView := m.treeView.View()
		mainContent = LayoutSidebarMain(LayoutSidebarMainInput{
			Sidebar:      sidebarView,
			Main:         pageView,
			SidebarWidth: m.sidebarWidth(),
			TotalWidth:   m.width,
			TotalHeight:  m.height - 1, // Reserve 1 line for status bar
		})
	} else {
		mainContent = pageView
	}

	// Add status bar at bottom
	statusView := m.statusBar.View()
	finalView := LayoutWithStatusBar(LayoutWithStatusBarInput{
		Content:   mainContent,
		StatusBar: statusView,
		Height:    m.height,
	})

	// Overlay command palette if visible
	if m.showPalette {
		paletteView := m.cmdPalette.View()
		finalView = LayoutCommandPalette(LayoutCommandPaletteInput{
			Background: finalView,
			Palette:    paletteView,
			Width:      m.width,
			Height:     m.height,
		})
	}

	return finalView
}

// sidebarWidth returns the sidebar width: a quarter of the terminal, clamped.
func (m *AppModel) sidebarWidth() int {
	width := m.width / 4
	if width < 20 {
		width = 20
	}
	if width > 40 {
		width = 40
	}
	return width
}

// contentWidth returns the width available to the current page.
func (m *AppModel) contentWidth() int {
	if !m.showSidebar {
		return m.width
	}
	width := m.width - m.sidebarWidth() - 1 // -1 for the sidebar border
	if width < 1 {
		width = 1
	}
	return width
}

// contentHeight returns the height available to the current page.
func (m *AppModel) contentHeight() int {
	height := m.height - 1 // Reserve 1 line for the global status bar
	if height < 1 {
		height = 1
	}
	return height
}

// resizePages tells every page the current content area size.
func (m *AppModel) resizePages() tea.Cmd {
	if m.width <= 0 || m.height <= 0 {
		return nil
	}
	size := tea.WindowSizeMsg{Width: m.contentWidth(), Height: m.contentHeight()}

	var cmds []tea.Cmd
	for pageID, page := range m.pages {
		updatedPage, cmd := page.Update(size)
		m.pages[pageID] = updatedPage
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// inputCapturer is implemented by pages that need raw key input,
// such as a text editor or a focused search box.
type inputCapturer interface {
	CapturesInput() bool
}

// pageCapturesInput reports whether the current page wants every key press.
// While the sidebar has focus the page is not the input target.
func (m *AppModel) pageCapturesInput() bool {
	if m.showSidebar && m.sidebarFocus {
		return false
	}
	page, ok := m.pages[m.currentPage].(inputCapturer)
	return ok && page.CapturesInput()
}

// updateCurrentPage forwards a message to the current page.
func (m *AppModel) updateCurrentPage(msg tea.Msg) tea.Cmd {
	page, ok := m.pages[m.currentPage]
	if !ok {
		return nil
	}
	updatedPage, cmd := page.Update(msg)
	m.pages[m.currentPage] = updatedPage
	return cmd
}

// handleGlobalKeys processes global keyboard shortcuts.
// Returns (handled, cmd) where handled indicates if the key was processed.
func (m *AppModel) handleGlobalKeys(msg tea.KeyMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return true, tea.Quit
	case "q":
		// Quitting mid-save would leave the outcome unknown; ctrl+c still
		// quits at once.
		if m.pageBusy() {
			return true, nil
		}
		return true, tea.Quit
	case "?":
		m.showHelp = true
		return true, nil
	case "ctrl+n":
		return true, m.onOpenPage("create a page inside it", (*pages.DetailPage).StartNewPage)
	default:
		return false, nil
	}
}

// navigateTo navigates to a specific page by ID.
// Creates the page if it doesn't exist yet.
func (m *AppModel) navigateTo(pageID PageID) tea.Cmd {
	// Record navigation
	m.navigator.NavigateTo(pageID)
	m.currentPage = pageID

	// Ensure page exists
	if _, ok := m.pages[pageID]; !ok {
		m.createPage(pageID)
	}

	// Initialize page if needed
	if page, ok := m.pages[pageID]; ok {
		return page.Init()
	}

	return nil
}

// navigateToDetail opens a Notion page in the page view. Coming from another
// page in the page view, the one being left is remembered so esc returns to
// it.
func (m *AppModel) navigateToDetail(notionPageID string) tea.Cmd {
	if current, ok := m.openPageID(); ok {
		if current == notionPageID {
			return nil
		}
		m.backTrail = append(m.backTrail, current)
		m.forwardTrail = nil
		return m.showPage(notionPageID)
	}

	// Entering the page view from elsewhere starts a new trail
	m.backTrail, m.forwardTrail = nil, nil
	m.installDetailPage(notionPageID)
	return m.navigateTo(PageDetail)
}

// openPageID returns the Notion page shown in the page view, if the page
// view is what is on screen.
func (m *AppModel) openPageID() (string, bool) {
	if m.currentPage != PageDetail {
		return "", false
	}
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	if !ok {
		return "", false
	}
	return detail.PageID(), true
}

// installDetailPage creates the page view for a Notion page.
func (m *AppModel) installDetailPage(notionPageID string) *pages.DetailPage {
	viewer := components.NewPageViewer(components.NewPageViewerInput{
		Width:  m.contentWidth(),
		Height: m.contentHeight() - 1, // Reserve 1 line for the page status bar
	})
	detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
		Width:        m.contentWidth(),
		Height:       m.contentHeight(),
		Viewer:       &viewer,
		NotionClient: m.notionClient,
		Cache:        m.cache,
		PageID:       notionPageID,
	})
	m.pages[PageDetail] = &detailPage
	return &detailPage
}

// showPage replaces the page in the page view without touching the screen
// history: the page view stays one entry there however many pages are read.
func (m *AppModel) showPage(notionPageID string) tea.Cmd {
	return m.installDetailPage(notionPageID).Init()
}

// trailBack returns to the page read before the current one. ok is false
// when the page view is not on screen or there is no earlier page.
func (m *AppModel) trailBack() (tea.Cmd, bool) {
	current, onPage := m.openPageID()
	if !onPage || len(m.backTrail) == 0 || m.pageBusy() {
		return nil, false
	}
	previous := m.backTrail[len(m.backTrail)-1]
	m.backTrail = m.backTrail[:len(m.backTrail)-1]
	m.forwardTrail = append(m.forwardTrail, current)
	return m.showPage(previous), true
}

// trailForward undoes trailBack.
func (m *AppModel) trailForward() (tea.Cmd, bool) {
	current, onPage := m.openPageID()
	if !onPage || len(m.forwardTrail) == 0 || m.pageBusy() {
		return nil, false
	}
	next := m.forwardTrail[len(m.forwardTrail)-1]
	m.forwardTrail = m.forwardTrail[:len(m.forwardTrail)-1]
	m.backTrail = append(m.backTrail, current)
	return m.showPage(next), true
}

// leaveArchivedPage moves off a page that no longer exists: to the page read
// before it, or else to the screen the page view was opened from.
func (m *AppModel) leaveArchivedPage() tea.Cmd {
	m.forwardTrail = nil
	if len(m.backTrail) > 0 {
		previous := m.backTrail[len(m.backTrail)-1]
		m.backTrail = m.backTrail[:len(m.backTrail)-1]
		return m.showPage(previous)
	}
	delete(m.pages, PageDetail)
	previousPage, ok := m.navigator.Back()
	if !ok {
		previousPage = PageDashboard
		m.navigator.Reset(PageDashboard)
	}
	m.currentPage = previousPage
	return m.onReturnTo(previousPage)
}

// goBack navigates to the previous page in history.
func (m *AppModel) goBack() {
	if previousPage, ok := m.navigator.Back(); ok {
		m.currentPage = previousPage
	}
}

// createPage creates a new page instance for the given page ID.
func (m *AppModel) createPage(pageID PageID) {
	switch pageID {
	case PageList:
		listPage := pages.NewListPage(pages.NewListPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
		})
		m.pages[pageID] = &listPage

	case PageDetail:
		// Detail page is created on-demand with specific page ID
		// This case shouldn't be hit normally
		viewer := components.NewPageViewer(components.NewPageViewerInput{
			Width:  m.contentWidth(),
			Height: m.contentHeight() - 1, // Reserve 1 line for the page status bar
		})
		detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			Viewer:       &viewer,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			PageID:       "",
		})
		m.pages[pageID] = &detailPage

	case PageSearch:
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
			Mode:         pages.SearchModeDatabase,
		})
		m.pages[pageID] = &searchPage

	case PageWorkspaceSearch:
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.contentWidth(),
			Height:       m.contentHeight(),
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
			Mode:         pages.SearchModeWorkspace,
		})
		m.pages[pageID] = &searchPage

	case PageDatabaseList:
		dbListPage := pages.NewDatabaseListPage(pages.NewDatabaseListPageInput{
			Width:       m.contentWidth(),
			Height:      m.contentHeight(),
			Databases:   m.config.Databases,
			DefaultDBID: m.currentDBID,
		})
		m.pages[pageID] = &dbListPage

	case PageDashboard:
		dashboardPage := pages.NewDashboardPage(pages.NewDashboardPageInput{
			Width:  m.contentWidth(),
			Height: m.contentHeight(),
			Config: m.config,
		})
		m.pages[pageID] = &dashboardPage
	}
}

// findPageByID finds a page in the page list by its ID.
func (m *AppModel) findPageByID(id PageID) *pages.Page {
	for i := range m.pageList {
		if m.pageList[i].ID == string(id) {
			return &m.pageList[i]
		}
	}
	return nil
}

// CurrentPage returns the currently active page ID.
func (m *AppModel) CurrentPage() PageID {
	return m.currentPage
}

// Navigator returns the navigation controller.
func (m *AppModel) Navigator() *Navigator {
	return m.navigator
}

// SetPageList updates the internal page list for sidebar rendering.
func (m *AppModel) SetPageList(pagesList []pages.Page) {
	m.pageList = pagesList
}

// handleCommandExecution routes command palette actions to appropriate handlers.
func (m *AppModel) handleCommandExecution(msg components.CommandExecutedMsg) tea.Cmd {
	switch msg.ActionType {
	case "search":
		if m.pageBusy() {
			return nil
		}
		// Navigate to search page
		return m.navigateToSearch()

	case "switch-db":
		if m.pageBusy() {
			return nil
		}
		// Navigate to database list page
		return m.navigateToDatabaseList()

	case "refresh":
		// Refresh current page
		return m.refreshCurrentPage()

	case "new-page":
		return m.onOpenPage("create a page inside it", (*pages.DetailPage).StartNewPage)

	case "export":
		return m.onOpenPage("export it", (*pages.DetailPage).Export)

	case "help":
		m.showHelp = true
		return nil

	default:
		return nil
	}
}

// modalPage is implemented by pages that can show a list or prompt in place
// of their content.
type modalPage interface {
	InMode() bool
}

// pageInMode reports whether the current page has such a list or prompt open.
func (m *AppModel) pageInMode() bool {
	page, ok := m.pages[m.currentPage].(modalPage)
	return ok && page.InMode()
}

// createdWhereUserIs reports whether the user is still on the page a new
// page was created from: the row list for a new row, the parent page for a
// new child page.
func (m *AppModel) createdWhereUserIs(msg pages.PageCreatedMsg) bool {
	if msg.ParentPageID == "" {
		return m.currentPage == PageList
	}
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	return ok && m.currentPage == PageDetail && detail.PageID() == msg.ParentPageID
}

// onReturnTo runs when going back lands on a page. A database's rows are
// reloaded, since the page just left may have changed one of them.
func (m *AppModel) onReturnTo(pageID PageID) tea.Cmd {
	if pageID != PageList {
		return nil
	}
	if list, ok := m.pages[PageList].(*pages.ListPage); ok {
		return list.Refresh()
	}
	return nil
}

// pageBusy reports whether the open page is in the middle of a save, create
// or export. Navigation and quitting wait for it: leaving now would hide
// the result, including the location of the user's text if the save fails.
func (m *AppModel) pageBusy() bool {
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	if !ok || !detail.Busy() {
		return false
	}
	m.statusBar.SetHelpText("Still working on the open page; ctrl+c quits anyway")
	return true
}

// onOpenPage runs an action on the page being viewed. Actions like "new
// page" and "export" need a page; without one the status bar says so.
func (m *AppModel) onOpenPage(purpose string, action func(*pages.DetailPage) tea.Cmd) tea.Cmd {
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	if !ok || m.currentPage != PageDetail {
		m.statusBar.SetHelpText("Open a page first to " + purpose)
		return nil
	}
	return action(detail)
}

// navigateToSearch navigates to the workspace search page.
func (m *AppModel) navigateToSearch() tea.Cmd {
	// Create or update search page with workspace mode
	searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
		Width:        m.contentWidth(),
		Height:       m.contentHeight(),
		NotionClient: m.notionClient,
		Cache:        m.cache,
		DatabaseID:   m.currentDBID,
		Mode:         pages.SearchModeWorkspace,
	})
	m.pages[PageWorkspaceSearch] = &searchPage

	// Navigate to workspace search page
	return m.navigateTo(PageWorkspaceSearch)
}

// navigateToDatabaseList navigates to the database list page.
func (m *AppModel) navigateToDatabaseList() tea.Cmd {
	// Create or update database list page
	dbListPage := pages.NewDatabaseListPage(pages.NewDatabaseListPageInput{
		Width:       m.contentWidth(),
		Height:      m.contentHeight(),
		Databases:   m.config.Databases,
		DefaultDBID: m.currentDBID,
	})
	m.pages[PageDatabaseList] = &dbListPage

	// Navigate to database list page
	return m.navigateTo(PageDatabaseList)
}

// refreshCurrentPage refreshes the current page.
func (m *AppModel) refreshCurrentPage() tea.Cmd {
	switch m.currentPage {
	case PageList:
		if page, ok := m.pages[PageList].(*pages.ListPage); ok {
			return page.Refresh()
		}
	case PageDetail:
		if page, ok := m.pages[PageDetail].(*pages.DetailPage); ok {
			return page.Refresh()
		}
	}
	return nil
}

// switchDatabase switches to a different database and refreshes the list page.
func (m *AppModel) switchDatabase(databaseID string) tea.Cmd {
	m.currentDBID = databaseID

	// Recreate list page with new database
	listPage := pages.NewListPage(pages.NewListPageInput{
		Width:        m.contentWidth(),
		Height:       m.contentHeight(),
		NotionClient: m.notionClient,
		Cache:        m.cache,
		DatabaseID:   databaseID,
	})
	m.pages[PageList] = &listPage

	// Navigate back to list page
	m.navigator.Reset(PageList)
	m.currentPage = PageList

	return listPage.Init()
}
