package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// ViewerInterface is imported from components package.
// This type alias ensures compatibility with the viewer implementation.
type ViewerInterface = components.ViewerInterface

// EditBlockMsg is emitted when the user picks a block to edit.
type EditBlockMsg struct {
	PageID  string
	BlockID string
}

// editableBlock is one entry in the block picker shown before editing.
type editableBlock struct {
	id      string
	kind    string
	preview string
}

const (
	detailHelpText = "r: refresh | e: edit | esc: back | ?: help"
	selectHelpText = "↑/↓: select block | enter: edit | esc: cancel"
)

// pageLoadedMsg is returned when page data is fetched.
type pageLoadedMsg struct {
	pageID string // Page this result belongs to; empty matches any page
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

	// Block picker state (active after pressing "e").
	selecting bool
	editable  []editableBlock
	cursor    int
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
	statusBar.SetHelpText(detailHelpText)

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
		// A slow load of a previously opened page must not replace this one
		if msg.pageID != "" && msg.pageID != dp.pageID {
			return dp, nil
		}
		if msg.err != nil {
			dp.err = msg.err
			dp.loading = false
			dp.statusBar.SetSyncStatus(components.StatusError)
			return dp, nil
		}

		dp.page = msg.page
		dp.blocks = msg.blocks
		dp.loading = false
		dp.stopSelecting()
		dp.statusBar.SetSyncStatus(components.StatusSynced)

		// Pass blocks to viewer
		if dp.viewer != nil {
			return dp, dp.viewer.SetBlocks(msg.blocks)
		}
		return dp, nil

	case tea.KeyMsg:
		if dp.selecting {
			return dp, dp.updateSelecting(msg)
		}

		switch msg.String() {
		case "r":
			// Refresh from API (bypass cache)
			return dp, dp.Refresh()

		case "e":
			// Open the block picker; editing works on one block at a time
			dp.startSelecting()
			return dp, nil

		case "esc":
			// Navigate back (use exported BackNavigationMsg)
			return dp, func() tea.Msg {
				return BackNavigationMsg{}
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

	if dp.selecting {
		return lipgloss.JoinVertical(lipgloss.Left, dp.renderBlockPicker(), dp.statusBar.View())
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
	pageID := dp.pageID
	return func() tea.Msg {
		ctx := context.Background()

		// Try cache first
		if dp.cache != nil {
			if cached, err := dp.cache.Get(ctx, pageID); err == nil {
				// Parse cached data
				var blocks []notionapi.Block
				cachedBytes, err := json.Marshal(cached)
				if err == nil {
					var response notionapi.GetChildrenResponse
					if err := json.Unmarshal(cachedBytes, &response); err == nil {
						blocks = response.Results

						// Also fetch page metadata (not cached)
						page, err := dp.notionClient.GetPage(ctx, pageID)
						if err != nil {
							return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch page metadata: %w", err)}
						}

						return pageLoadedMsg{
							pageID: pageID,
							page:   page,
							blocks: blocks,
						}
					}
				}
			}
		}

		// Cache miss or error - fetch from API
		return dp.fetchPageFromAPI(pageID)
	}
}

// fetchPageFromAPICmd fetches page data directly from the API.
func (dp *DetailPage) fetchPageFromAPICmd() tea.Cmd {
	pageID := dp.pageID
	return func() tea.Msg {
		return dp.fetchPageFromAPI(pageID)
	}
}

// fetchPageFromAPI performs the actual API calls to fetch page and blocks.
// pageID is passed in rather than read from dp: this runs on another goroutine.
func (dp *DetailPage) fetchPageFromAPI(pageID string) tea.Msg {
	ctx := context.Background()

	// Fetch page metadata
	page, err := dp.notionClient.GetPage(ctx, pageID)
	if err != nil {
		return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch page: %w", err)}
	}

	// Fetch all blocks, including paginated and nested ones
	blocks, err := notion.FetchBlockTree(ctx, dp.notionClient, pageID, notion.DefaultMaxBlockDepth)
	if err != nil {
		return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch blocks: %w", err)}
	}
	children := &notionapi.GetChildrenResponse{
		Object:  notionapi.ObjectTypeList,
		Results: blocks,
	}

	// Cache the blocks. Caching is best effort: a failed write only means the
	// next load hits the API again, so the error is deliberately dropped.
	if dp.cache != nil {
		_ = dp.cache.Set(ctx, cache.SetInput{
			PageID: pageID,
			Data:   children,
			TTL:    time.Hour,
		})
	}

	return pageLoadedMsg{
		pageID: pageID,
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

// CapturesInput reports whether the page needs every key press,
// which is the case while the block picker is open.
func (dp *DetailPage) CapturesInput() bool {
	return dp.selecting
}

// startSelecting opens the block picker if the page has editable blocks.
func (dp *DetailPage) startSelecting() {
	if dp.loading || dp.err != nil {
		return
	}
	dp.editable = collectEditableBlocks(dp.blocks, 0)
	if len(dp.editable) == 0 {
		dp.statusBar.SetHelpText("No editable text blocks on this page | esc: back")
		return
	}
	dp.selecting = true
	dp.cursor = 0
	dp.statusBar.SetMode(components.ModeEdit)
	dp.statusBar.SetHelpText(selectHelpText)
}

// stopSelecting closes the block picker and restores the browse status bar.
func (dp *DetailPage) stopSelecting() {
	dp.selecting = false
	dp.editable = nil
	dp.cursor = 0
	dp.statusBar.SetMode(components.ModeBrowse)
	dp.statusBar.SetHelpText(detailHelpText)
}

// updateSelecting handles keys while the block picker is open.
func (dp *DetailPage) updateSelecting(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if dp.cursor > 0 {
			dp.cursor--
		}
	case "down", "j":
		if dp.cursor < len(dp.editable)-1 {
			dp.cursor++
		}
	case "home", "g":
		dp.cursor = 0
	case "end", "G":
		dp.cursor = len(dp.editable) - 1
	case "esc", "q":
		dp.stopSelecting()
	case "enter":
		if dp.cursor < 0 || dp.cursor >= len(dp.editable) {
			return nil
		}
		target := EditBlockMsg{PageID: dp.pageID, BlockID: dp.editable[dp.cursor].id}
		dp.stopSelecting()
		return func() tea.Msg { return target }
	}
	return nil
}

// renderBlockPicker renders the list of editable blocks with the cursor in view.
func (dp *DetailPage) renderBlockPicker() string {
	titleStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	// Reserve lines for the title, a blank line and the status bar.
	visible := dp.height - 3
	if visible < 1 {
		visible = 1
	}
	start := 0
	if dp.cursor >= visible {
		start = dp.cursor - visible + 1
	}
	end := start + visible
	if end > len(dp.editable) {
		end = len(dp.editable)
	}

	previewWidth := dp.width - 24
	if previewWidth < 10 {
		previewWidth = 10
	}

	lines := []string{
		titleStyle.Render(fmt.Sprintf("Select a block to edit (%d/%d)", dp.cursor+1, len(dp.editable))),
		"",
	}
	for i := start; i < end; i++ {
		item := dp.editable[i]
		preview := truncateRunes(item.preview, previewWidth)
		if preview == "" {
			preview = "(empty)"
		}
		line := fmt.Sprintf("%-14s %s", item.kind, preview)
		if i == dp.cursor {
			lines = append(lines, selectedStyle.Render("> "+line))
		} else {
			lines = append(lines, "  "+mutedStyle.Render(fmt.Sprintf("%-14s", item.kind))+" "+preview)
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// collectEditableBlocks flattens the block tree into the text blocks the
// editor can handle, in document order.
func collectEditableBlocks(blocks []notionapi.Block, depth int) []editableBlock {
	var out []editableBlock
	for _, block := range blocks {
		if block == nil {
			continue
		}
		if isEditableBlockType(block.GetType()) && block.GetID() != "" {
			preview := strings.ReplaceAll(extractBlockText(block), "\n", " ")
			out = append(out, editableBlock{
				id:      string(block.GetID()),
				kind:    strings.Repeat("  ", depth) + blockKindLabel(block.GetType()),
				preview: preview,
			})
		}
		out = append(out, collectEditableBlocks(notion.BlockChildren(block), depth+1)...)
	}
	return out
}

// isEditableBlockType reports whether the editor can load and save a block type.
func isEditableBlockType(blockType notionapi.BlockType) bool {
	switch blockType {
	case notionapi.BlockTypeParagraph,
		notionapi.BlockTypeHeading1,
		notionapi.BlockTypeHeading2,
		notionapi.BlockTypeHeading3,
		notionapi.BlockTypeBulletedListItem,
		notionapi.BlockTypeNumberedListItem,
		notionapi.BlockTypeToDo,
		notionapi.BlockTypeToggle,
		notionapi.BlockTypeCode,
		notionapi.BlockTypeQuote,
		notionapi.BlockTypeCallout:
		return true
	default:
		return false
	}
}

// blockKindLabel returns a short human-readable label for a block type.
func blockKindLabel(blockType notionapi.BlockType) string {
	switch blockType {
	case notionapi.BlockTypeParagraph:
		return "text"
	case notionapi.BlockTypeHeading1:
		return "h1"
	case notionapi.BlockTypeHeading2:
		return "h2"
	case notionapi.BlockTypeHeading3:
		return "h3"
	case notionapi.BlockTypeBulletedListItem:
		return "bullet"
	case notionapi.BlockTypeNumberedListItem:
		return "number"
	case notionapi.BlockTypeToDo:
		return "to-do"
	default:
		return string(blockType)
	}
}

// truncateRunes shortens text to at most max runes, adding an ellipsis when cut.
func truncateRunes(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}
