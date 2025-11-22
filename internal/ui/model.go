package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
	"github.com/Panandika/notion-tui/internal/ui/pages"
)

// AppModel represents the root TUI orchestrator that manages pages and global components.
// It implements the Elm architecture pattern with page routing and navigation.
type AppModel struct {
	// Navigation
	currentPage PageID
	pages       map[PageID]tea.Model
	navigator   *Navigator

	// Components (always visible)
	sidebar    components.Sidebar
	statusBar  components.StatusBar
	cmdPalette components.CommandPalette

	// State
	width       int
	height      int
	showSidebar bool
	showPalette bool
	mode        ViewMode

	// Services
	notionClient *notion.Client
	cache        *cache.PageCache
	config       *config.Config

	// Data
	pageList     []pages.Page
	ready        bool
	err          error
	selectedPage *pages.Page
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

	// Initialize navigator
	nav := NewNavigator(NewNavigatorInput{
		InitialPage: PageList,
		MaxHistory:  DefaultMaxHistory,
	})

	// Initialize global components
	sidebar := components.NewSidebar(components.NewSidebarInput{
		Items:  []components.Item{},
		Width:  20, // Will be adjusted on first WindowSizeMsg
		Height: 20,
		Title:  "Pages",
	})

	statusBar := components.NewStatusBar()
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("? for help")

	cmdPalette := components.NewCommandPalette()

	return AppModel{
		currentPage:  PageList,
		pages:        make(map[PageID]tea.Model),
		navigator:    &nav,
		sidebar:      sidebar,
		statusBar:    statusBar,
		cmdPalette:   cmdPalette,
		width:        0,
		height:       0,
		showSidebar:  true,
		showPalette:  false,
		mode:         ViewModeBrowse,
		notionClient: notionClient,
		cache:        cacheInstance,
		config:       input.Config,
		pageList:     []pages.Page{},
		ready:        false,
		err:          nil,
		selectedPage: nil,
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
	)
}

// initializePages creates and registers all page instances.
func (m *AppModel) initializePages() {
	// Create ListPage
	listPage := pages.NewListPage(pages.NewListPageInput{
		Width:        m.width,
		Height:       m.height,
		NotionClient: m.notionClient,
		Cache:        m.cache,
		DatabaseID:   m.config.DatabaseID,
	})
	m.pages[PageList] = &listPage

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

		m.sidebar.SetSize(sidebarWidth, m.height-1)
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

	case tea.KeyMsg:
		// Handle mode-specific keys FIRST before global keys
		switch msg.String() {
		case "tab":
			// Toggle sidebar
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
			// Handle back navigation
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

	case components.ItemSelectedMsg:
		// User selected a page from sidebar
		pageID := msg.ID
		m.selectedPage = m.findPageByID(PageID(pageID))

		// Navigate to detail page
		return m, m.navigateToDetail(pageID)

	case components.CommandExecutedMsg:
		// Command palette executed a command
		m.showPalette = false
		// Handle command execution (future implementation)
		return m, nil
	}

	// Delegate to current page
	if page, ok := m.pages[m.currentPage]; ok {
		updatedPage, cmd := page.Update(msg)
		m.pages[m.currentPage] = updatedPage
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// Update sidebar if visible
	if m.showSidebar && m.currentPage == PageList {
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
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

	// For list page, compose with sidebar
	var mainContent string
	if m.currentPage == PageList && m.showSidebar {
		sidebarView := m.sidebar.View()
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
		// TODO: Show help overlay
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
	// Create or update detail page
	detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
		Width:        m.width,
		Height:       m.height,
		Viewer:       nil, // TODO: Add viewer when component is ready
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
			DatabaseID:   m.config.DatabaseID,
		})
		m.pages[pageID] = &listPage

	case PageDetail:
		// Detail page is created on-demand with specific page ID
		// This case shouldn't be hit normally
		detailPage := pages.NewDetailPage(pages.NewDetailPageInput{
			Width:        m.width,
			Height:       m.height,
			Viewer:       nil,
			NotionClient: m.notionClient,
			Cache:        m.cache,
			PageID:       "",
		})
		m.pages[pageID] = &detailPage

	case PageEdit:
		// Edit page is created on-demand with specific page ID
		// This case shouldn't be hit normally
		// TODO: Implement EditPage creation
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
