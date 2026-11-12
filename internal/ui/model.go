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

	case components.TreeNavigationMsg:
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

		// A page that captures input (text editor, block picker, focused
		// search box) receives every other key itself.
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
			// The palette navigates away, which would silently drop unsaved
			// text, so it stays closed while the editor is open.
			if m.currentPage == PageEdit {
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

		case "esc":
			// On search page, let it handle Esc first (e.g., to clear filter)
			// then fall through to page delegation which will handle back navigation
			if isSearchPage {
				break // Fall through to page delegation
			}
			// Handle back navigation for other pages
			if m.navigator.CanGoBack() {
				previousPage, ok := m.navigator.Back()
				if ok {
					m.currentPage = previousPage
					return m, nil
				}
			}
		}

	case pages.NavigationMsg:
		// Handle navigation to a new page
		return m, m.navigateTo(PageID(msg.PageID()))

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

	case pages.EditBlockMsg:
		// User picked a block to edit on the detail page
		return m, m.navigateToEdit(msg.PageID, msg.BlockID)

	case pages.EditFinishedMsg:
		// User left the editor: return to the page and reload it if it changed
		return m, m.finishEdit(msg)

	case components.CommandExecutedMsg:
		// Command palette executed a command
		m.showPalette = false
		return m, m.handleCommandExecution(msg)

	case pages.DatabaseSelectedMsg:
		// User selected a different database
		m.currentDBID = msg.DatabaseID
		// Refresh list page with new database
		return m, m.switchDatabase(msg.DatabaseID)

	case pages.SearchNavigationMsg:
		// User selected a search result (page or database)
		if msg.ObjectType == "database" {
			// Switch to this database and show its pages
			return m, m.switchDatabase(msg.ID)
		}
		// Navigate to page detail
		return m, m.navigateToDetail(msg.ID)

	case pages.BackNavigationMsg:
		// Handle back navigation request from search page
		if m.navigator.CanGoBack() {
			previousPage, ok := m.navigator.Back()
			if ok {
				m.currentPage = previousPage
				return m, nil
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
	case "ctrl+c", "q":
		return true, tea.Quit
	case "?":
		// Toggle help display
		m.showHelp = !m.showHelp
		if m.showHelp {
			m.statusBar.SetHelpText("Tab:focus tree | Ctrl+B:toggle tree | ←/→:expand | Enter:open | Ctrl+P:cmd | q:quit | ?:close")
		} else {
			m.statusBar.SetHelpText("? for help | Tab: focus tree")
		}
		return true, nil
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

// navigateToDetail navigates to the detail page for a specific Notion page.
func (m *AppModel) navigateToDetail(notionPageID string) tea.Cmd {
	// Create viewer for the detail page
	viewer := components.NewPageViewer(components.NewPageViewerInput{
		Width:  m.contentWidth(),
		Height: m.contentHeight() - 1, // Reserve 1 line for the page status bar
	})

	// Create or update detail page
	detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
		Width:        m.contentWidth(),
		Height:       m.contentHeight(),
		Viewer:       &viewer,
		NotionClient: m.notionClient,
		Cache:        m.cache,
		PageID:       notionPageID,
	})
	m.pages[PageDetail] = &detailPage

	// Navigate to detail page
	return m.navigateTo(PageDetail)
}

// navigateToEdit opens the editor for one block of a Notion page.
func (m *AppModel) navigateToEdit(notionPageID, blockID string) tea.Cmd {
	editPage := pages.NewEditPage(pages.NewEditPageInput{
		Width:        m.contentWidth(),
		Height:       m.contentHeight(),
		NotionClient: m.notionClient,
		PageID:       notionPageID,
		BlockID:      blockID,
	})
	m.pages[PageEdit] = &editPage

	// The editor needs the keyboard, so take focus away from the sidebar
	m.sidebarFocus = false
	m.treeView.SetFocused(false)
	m.mode = ViewModeEdit
	m.statusBar.SetMode(components.ModeEdit)

	return m.navigateTo(PageEdit)
}

// finishEdit leaves the editor and reloads the detail page if anything was saved.
func (m *AppModel) finishEdit(msg pages.EditFinishedMsg) tea.Cmd {
	m.mode = ViewModeBrowse
	m.statusBar.SetMode(components.ModeBrowse)
	delete(m.pages, PageEdit)

	if m.currentPage == PageEdit {
		previousPage, ok := m.navigator.Back()
		if !ok {
			previousPage = PageDashboard
		}
		m.currentPage = previousPage
	}

	if !msg.Saved {
		return nil
	}
	if detail, ok := m.pages[PageDetail].(*pages.DetailPage); ok && detail.PageID() == msg.PageID {
		return detail.Refresh()
	}
	return nil
}

// goBack navigates to the previous page in history.
func (m *AppModel) goBack() tea.Cmd {
	if previousPage, ok := m.navigator.Back(); ok {
		m.currentPage = previousPage
		return nil
	}
	return nil
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

	case PageEdit:
		// Edit page needs a page and block ID, so it is only created by
		// navigateToEdit. Nothing to create here.

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
		// Navigate to search page
		return m.navigateToSearch()

	case "switch-db":
		// Navigate to database list page
		return m.navigateToDatabaseList()

	case "refresh":
		// Refresh current page
		return m.refreshCurrentPage()

	case "new-page":
		// TODO: Implement new page creation
		return nil

	case "export":
		// TODO: Implement export functionality
		return nil

	default:
		return nil
	}
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
