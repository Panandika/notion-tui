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
			Width:        m.width,
			Height:       m.height,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.config.GetDatabaseID(),
		})
		m.pages[PageList] = &listPage
	}

	// If no databases, create workspace search page as initial view
	if !m.config.HasDatabases() {
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.width,
			Height:       m.height,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   "",
			Mode:         pages.SearchModeWorkspace,
		})
		m.pages[PageWorkspaceSearch] = &searchPage
	}

	// Create Dashboard page
	dashboardPage := pages.NewDashboardPage(pages.NewDashboardPageInput{
		Width:  m.width,
		Height: m.height,
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
		sidebarWidth := m.width / 4
		if sidebarWidth < 20 {
			sidebarWidth = 20
		}
		if sidebarWidth > 40 {
			sidebarWidth = 40
		}

		m.treeView.SetSize(sidebarWidth, m.height-1)
		m.statusBar.SetWidth(m.width)

		// Update all pages with window size
		for pageID, page := range m.pages {
			updatedPage, cmd := page.Update(msg)
			m.pages[pageID] = updatedPage
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		return m, tea.Batch(cmds...)

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
		// Check if we're on the search page - it needs special key handling
		isSearchPage := m.currentPage == PageWorkspaceSearch

		// Handle mode-specific keys FIRST before global keys
		switch msg.String() {
		case "tab":
			// Toggle focus between sidebar and main content
			m.sidebarFocus = !m.sidebarFocus
			m.treeView.SetFocused(m.sidebarFocus)
			return m, nil

		case "ctrl+b":
			// Toggle sidebar visibility
			m.showSidebar = !m.showSidebar
			return m, nil

		case "ctrl+p":
			// Toggle command palette (works everywhere)
			m.cmdPalette.Toggle()
			m.showPalette = m.cmdPalette.IsOpen()
			return m, nil
		}

		// Handle global keys
		if handled, cmd := m.handleGlobalKeys(msg); handled {
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// If palette is open, only palette handles keys (except ctrl+p which is handled above)
		if m.showPalette {
			var cmd tea.Cmd
			m.cmdPalette, cmd = m.cmdPalette.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
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
	if page, ok := m.pages[m.currentPage]; ok {
		updatedPage, cmd := page.Update(msg)
		m.pages[m.currentPage] = updatedPage
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update tree view if sidebar is visible and focused
	if m.showSidebar && m.sidebarFocus {
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
			SidebarWidth: m.width / 4,
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
		Width:  m.width,
		Height: m.height - 2, // Reserve space for status bar
	})

	// Create or update detail page
	detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
		Width:        m.width,
		Height:       m.height,
		Viewer:       &viewer,
		NotionClient: m.notionClient,
		Cache:        m.cache,
		PageID:       notionPageID,
	})
	m.pages[PageDetail] = &detailPage

	// Navigate to detail page
	return m.navigateTo(PageDetail)
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
			Width:        m.width,
			Height:       m.height,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
		})
		m.pages[pageID] = &listPage

	case PageDetail:
		// Detail page is created on-demand with specific page ID
		// This case shouldn't be hit normally
		viewer := components.NewPageViewer(components.NewPageViewerInput{
			Width:  m.width,
			Height: m.height - 2,
		})
		detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
			Width:        m.width,
			Height:       m.height,
			Viewer:       &viewer,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			PageID:       "",
		})
		m.pages[pageID] = &detailPage

	case PageEdit:
		// Edit page is created on-demand with specific page ID
		// This case shouldn't be hit normally
		// TODO: Implement EditPage creation

	case PageSearch:
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.width,
			Height:       m.height,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
			Mode:         pages.SearchModeDatabase,
		})
		m.pages[pageID] = &searchPage

	case PageWorkspaceSearch:
		searchPage := pages.NewSearchPage(pages.NewSearchPageInput{
			Width:        m.width,
			Height:       m.height,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			DatabaseID:   m.currentDBID,
			Mode:         pages.SearchModeWorkspace,
		})
		m.pages[pageID] = &searchPage

	case PageDatabaseList:
		dbListPage := pages.NewDatabaseListPage(pages.NewDatabaseListPageInput{
			Width:       m.width,
			Height:      m.height,
			Databases:   m.config.Databases,
			DefaultDBID: m.currentDBID,
		})
		m.pages[pageID] = &dbListPage

	case PageDashboard:
		dashboardPage := pages.NewDashboardPage(pages.NewDashboardPageInput{
			Width:  m.width,
			Height: m.height,
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
		Width:        m.width,
		Height:       m.height,
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
		Width:       m.width,
		Height:      m.height,
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
		Width:        m.width,
		Height:       m.height,
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
