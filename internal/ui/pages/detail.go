package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// ViewerInterface defines the interface for the content viewer component.
// This interface allows parallel development and easy testing with mocks.
type ViewerInterface interface {
	Init() tea.Cmd
	Update(tea.Msg) (ViewerInterface, tea.Cmd)
	View() string
	SetBlocks([]notionapi.Block) tea.Cmd
	SetSize(width, height int)
}

// navigationMsg is emitted when the user requests navigation.
type navigationMsg struct {
	action string
	pageID string
}

// pageLoadedMsg is returned when page data is fetched.
type pageLoadedMsg struct {
	page   *notionapi.Page
	blocks []notionapi.Block
	err    error
}

// DetailPage displays a single Notion page with its content blocks.
// It fetches page metadata and blocks, with caching support.
type DetailPage struct {
	viewer       ViewerInterface
	statusBar    components.StatusBar
	pageID       string
	page         *notionapi.Page
	blocks       []notionapi.Block
	loading      bool
	err          error
	width        int
	height       int
	notionClient NotionClient
	cache        *cache.PageCache
}

// NewDetailPageInput contains the parameters for creating a DetailPage.
type NewDetailPageInput struct {
	Width        int
	Height       int
	Viewer       ViewerInterface
	NotionClient NotionClient
	Cache        *cache.PageCache
	PageID       string
}

// NewDetailPage creates a new DetailPage instance.
func NewDetailPage(input NewDetailPageInput) DetailPage {
	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("r: refresh | e: edit | esc: back | ?: help")

	viewerHeight := input.Height - 1 // Reserve 1 line for status bar
	if input.Viewer != nil {
		input.Viewer.SetSize(input.Width, viewerHeight)
	}

	return DetailPage{
		viewer:       input.Viewer,
		statusBar:    statusBar,
		pageID:       input.PageID,
		loading:      true,
		width:        input.Width,
		height:       input.Height,
		notionClient: input.NotionClient,
		cache:        input.Cache,
	}
}

// Init initializes the DetailPage and loads the page content.
func (dp *DetailPage) Init() tea.Cmd {
	// Call viewer init immediately if available
	if dp.viewer != nil {
		dp.viewer.Init()
	}

	return dp.fetchPageCmd()
}

// Update handles messages and updates the DetailPage state.
func (dp *DetailPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pageLoadedMsg:
		if msg.err != nil {
			dp.err = msg.err
			dp.loading = false
			dp.statusBar.SetSyncStatus(components.StatusError)
			return dp, nil
		}

		dp.page = msg.page
		dp.blocks = msg.blocks
		dp.loading = false
		dp.statusBar.SetSyncStatus(components.StatusSynced)

		// Pass blocks to viewer
		if dp.viewer != nil {
			return dp, dp.viewer.SetBlocks(msg.blocks)
		}
		return dp, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Refresh from API (bypass cache)
			return dp, dp.Refresh()

		case "e":
			// Navigate to edit mode
			return dp, func() tea.Msg {
				return navigationMsg{action: "edit", pageID: dp.pageID}
			}

		case "esc":
			// Navigate back
			return dp, func() tea.Msg {
				return navigationMsg{action: "back"}
			}
		}

	case tea.WindowSizeMsg:
		dp.width = msg.Width
		dp.height = msg.Height
		dp.statusBar.SetWidth(msg.Width)

		// Update viewer size (reserve 1 line for status bar)
		viewerHeight := msg.Height - 1
		if dp.viewer != nil {
			dp.viewer.SetSize(msg.Width, viewerHeight)
		}
		return dp, nil
	}

	// Delegate other messages to viewer
	if dp.viewer != nil {
		updatedViewer, cmd := dp.viewer.Update(msg)
		dp.viewer = updatedViewer
		return dp, cmd
	}

	return dp, nil
}

// View renders the DetailPage UI.
func (dp *DetailPage) View() string {
	if dp.loading {
		loadingText := "Loading page..."
		dp.statusBar.SetSyncStatus(components.StatusSyncing)
		statusContent := dp.statusBar.View()
		return lipgloss.JoinVertical(lipgloss.Left, loadingText, statusContent)
	}

	if dp.err != nil {
		errorText := fmt.Sprintf("Error: %v\nPress ESC to go back", dp.err)
		statusContent := dp.statusBar.View()
		return lipgloss.JoinVertical(lipgloss.Left, errorText, statusContent)
	}

	// Render viewer content and status bar
	var viewerContent string
	if dp.viewer != nil {
		viewerContent = dp.viewer.View()
	} else {
		viewerContent = "No viewer available"
	}

	statusContent := dp.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, viewerContent, statusContent)
}

// LoadPage loads a different page by ID.
func (dp *DetailPage) LoadPage(pageID string) tea.Cmd {
	dp.pageID = pageID
	dp.loading = true
	dp.err = nil
	dp.page = nil
	dp.blocks = nil
	return dp.fetchPageCmd()
}

// Refresh reloads the current page from the API, bypassing cache.
func (dp *DetailPage) Refresh() tea.Cmd {
	dp.loading = true
	dp.err = nil
	dp.statusBar.SetSyncStatus(components.StatusSyncing)
	return dp.fetchPageFromAPICmd()
}

// fetchPageCmd loads page data, trying cache first then falling back to API.
func (dp *DetailPage) fetchPageCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Try cache first
		if dp.cache != nil {
			if cached, err := dp.cache.Get(ctx, dp.pageID); err == nil {
				// Parse cached data
				var blocks []notionapi.Block
				cachedBytes, err := json.Marshal(cached)
				if err == nil {
					var response notionapi.GetChildrenResponse
					if err := json.Unmarshal(cachedBytes, &response); err == nil {
						blocks = response.Results

						// Also fetch page metadata (not cached)
						page, err := dp.notionClient.GetPage(ctx, dp.pageID)
						if err != nil {
							return pageLoadedMsg{err: fmt.Errorf("fetch page metadata: %w", err)}
						}

						return pageLoadedMsg{
							page:   page,
							blocks: blocks,
						}
					}
				}
			}
		}

		// Cache miss or error - fetch from API
		return dp.fetchPageFromAPI()
	}
}

// fetchPageFromAPICmd fetches page data directly from the API.
func (dp *DetailPage) fetchPageFromAPICmd() tea.Cmd {
	return func() tea.Msg {
		return dp.fetchPageFromAPI()
	}
}

// fetchPageFromAPI performs the actual API calls to fetch page and blocks.
func (dp *DetailPage) fetchPageFromAPI() tea.Msg {
	ctx := context.Background()

	// Fetch page metadata
	page, err := dp.notionClient.GetPage(ctx, dp.pageID)
	if err != nil {
		return pageLoadedMsg{err: fmt.Errorf("fetch page: %w", err)}
	}

	// Fetch blocks
	children, err := dp.notionClient.GetBlocks(ctx, dp.pageID, nil)
	if err != nil {
		return pageLoadedMsg{err: fmt.Errorf("fetch blocks: %w", err)}
	}

	// Cache the blocks
	if dp.cache != nil {
		if err := dp.cache.Set(ctx, cache.SetInput{
			PageID: dp.pageID,
			Data:   children,
			TTL:    time.Hour,
		}); err != nil {
			// Log error but don't fail the operation
			// In production, this would use structured logging
		}
	}

	return pageLoadedMsg{
		page:   page,
		blocks: children.Results,
	}
}

// PageID returns the current page ID.
func (dp *DetailPage) PageID() string {
	return dp.pageID
}

// Page returns the current page metadata.
func (dp *DetailPage) Page() *notionapi.Page {
	return dp.page
}

// Blocks returns the current blocks.
func (dp *DetailPage) Blocks() []notionapi.Block {
	return dp.blocks
}

// IsLoading returns whether the page is currently loading.
func (dp *DetailPage) IsLoading() bool {
	return dp.loading
}

// Error returns the current error, if any.
func (dp *DetailPage) Error() error {
	return dp.err
}
