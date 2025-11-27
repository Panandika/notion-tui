package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/notion"
)

// ContentLoadedMsg contains loaded content or error.
type ContentLoadedMsg struct {
	content string
	err     error
}

// Content returns the loaded content.
func (c ContentLoadedMsg) Content() string {
	return c.content
}

// Err returns any error that occurred.
func (c ContentLoadedMsg) Err() error {
	return c.err
}

// ErrorMsg represents an error that occurred during operation.
type ErrorMsg struct {
	message string
	err     error
}

// Error implements the error interface for ErrorMsg.
func (e ErrorMsg) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.message, e.err)
	}
	return e.message
}

// PageViewer displays page content with markdown rendering and scrolling.
// Uses viewport for smooth scrolling and glamour for markdown styling.
type PageViewer struct {
	viewport viewport.Model
	content  string
	pageID   string
	blocks   []notionapi.Block
	ready    bool
	loading  bool
	err      error
	width    int
	height   int
}

// NewPageViewerInput contains parameters for creating a new PageViewer.
// Following CS-5: input struct for functions with multiple parameters.
type NewPageViewerInput struct {
	Width  int
	Height int
}

// NewPageViewer creates a new PageViewer component with the given dimensions.
// The viewport is initialized but not yet ready until content is loaded.
func NewPageViewer(input NewPageViewerInput) PageViewer {
	vp := viewport.New(input.Width, input.Height)
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	return PageViewer{
		viewport: vp,
		width:    input.Width,
		height:   input.Height,
		ready:    false,
		loading:  false,
	}
}

// Init initializes the PageViewer component.
// Returns nil as no initial command is needed.
func (pv PageViewer) Init() tea.Cmd {
	return nil
}

// ViewerInterface defines the interface for content viewer components.
// This interface allows for different viewer implementations and easy testing.
type ViewerInterface interface {
	Init() tea.Cmd
	Update(tea.Msg) (ViewerInterface, tea.Cmd)
	View() string
	SetBlocks([]notionapi.Block) tea.Cmd
	SetSize(width, height int)
}

// Update handles messages and updates the PageViewer state.
// Processes content loading, errors, window resize, and keyboard/mouse events.
func (pv *PageViewer) Update(msg tea.Msg) (ViewerInterface, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case ContentLoadedMsg:
		if msg.Err() != nil {
			pv.err = msg.Err()
			pv.loading = false
			return pv, nil
		}

		pv.content = msg.Content()
		pv.viewport.SetContent(msg.Content())
		pv.ready = true
		pv.loading = false
		return pv, nil

	case ErrorMsg:
		pv.err = msg
		pv.loading = false
		return pv, nil

	case tea.WindowSizeMsg:
		pv.width = msg.Width
		pv.height = msg.Height
		pv.viewport.Width = msg.Width
		pv.viewport.Height = msg.Height
		if pv.ready {
			pv.viewport.SetContent(pv.content)
		}
		return pv, nil

	case tea.KeyMsg:
		pv.viewport, cmd = pv.viewport.Update(msg)
		return pv, cmd

	case tea.MouseMsg:
		pv.viewport, cmd = pv.viewport.Update(msg)
		return pv, cmd
	}

	return pv, nil
}

// View renders the PageViewer component.
// Shows loading state, error state, or the viewport with content.
func (pv PageViewer) View() string {
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Padding(1, 2).
		Width(pv.width).
		Height(pv.height)

	if pv.loading {
		return mutedStyle.Render("Loading content...")
	}

	if pv.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", pv.err))
	}

	if !pv.ready {
		return mutedStyle.Render("No content loaded")
	}

	return boxStyle.Render(pv.viewport.View())
}

// SetBlocks converts Notion blocks to markdown and renders them.
// Returns a command that performs the conversion asynchronously.
// Following the Bubble Tea pattern of non-blocking Update().
func (pv PageViewer) SetBlocks(blocks []notionapi.Block) tea.Cmd {
	pv.blocks = blocks
	pv.loading = true
	pv.err = nil

	return func() tea.Msg {
		// Convert blocks to markdown
		markdown, err := notion.ConvertBlocksToMarkdown(blocks)
		if err != nil {
			return ErrorMsg{message: "failed to convert blocks", err: err}
		}

		// Render with glamour using dark theme
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(pv.width-4), // Account for padding
		)
		if err != nil {
			return ErrorMsg{message: "failed to create renderer", err: err}
		}

		rendered, err := renderer.Render(markdown)
		if err != nil {
			return ErrorMsg{message: "failed to render markdown", err: err}
		}

		return ContentLoadedMsg{content: rendered, err: nil}
	}
}

// SetContent directly sets the viewport content.
// Use this for pre-rendered content, or SetBlocks for Notion blocks.
func (pv *PageViewer) SetContent(content string) {
	pv.content = content
	pv.viewport.SetContent(content)
	pv.ready = true
	pv.loading = false
	pv.err = nil
}

// SetSize updates the viewport dimensions.
// Useful for responsive layout changes.
func (pv *PageViewer) SetSize(width, height int) {
	pv.width = width
	pv.height = height
	pv.viewport.Width = width
	pv.viewport.Height = height
	if pv.ready {
		pv.viewport.SetContent(pv.content)
	}
}

// Content returns the current content string.
func (pv PageViewer) Content() string {
	return pv.content
}

// IsReady returns true if the viewer has content loaded and is ready to display.
func (pv PageViewer) IsReady() bool {
	return pv.ready
}

// IsLoading returns true if content is currently being loaded.
func (pv PageViewer) IsLoading() bool {
	return pv.loading
}

// Error returns any error that occurred during content loading.
func (pv PageViewer) Error() error {
	return pv.err
}

// PageID returns the current page ID being viewed.
func (pv PageViewer) PageID() string {
	return pv.pageID
}

// SetPageID sets the page ID for tracking purposes.
func (pv *PageViewer) SetPageID(pageID string) {
	pv.pageID = pageID
}

// ScrollPercent returns the current scroll percentage (0-100).
func (pv PageViewer) ScrollPercent() float64 {
	return pv.viewport.ScrollPercent()
}

// AtTop returns true if the viewport is scrolled to the top.
func (pv PageViewer) AtTop() bool {
	return pv.viewport.AtTop()
}

// AtBottom returns true if the viewport is scrolled to the bottom.
func (pv PageViewer) AtBottom() bool {
	return pv.viewport.AtBottom()
}
