package pages

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// Page represents a Notion page in the UI.
type Page struct {
	ID        string
	Title     string
	Status    string
	UpdatedAt time.Time
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
	pages []Page
	err   error
}

// ListPage wraps the Sidebar component with page listing logic.
type ListPage struct {
	sidebar      components.Sidebar
	statusBar    components.StatusBar
	pageList     []Page
	selectedIdx  int
	loading      bool
	err          error
	width        int
	height       int
	notionClient NotionClient
	cache        *cache.PageCache
	databaseID   string
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
	// Create empty sidebar initially
	sidebar := components.NewSidebar(components.NewSidebarInput{
		Items:  []components.Item{},
		Width:  input.Width / 4,
		Height: input.Height - 2,
		Title:  "Pages",
	})

	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("r: refresh | ?: help")

	return ListPage{
		sidebar:      sidebar,
		statusBar:    statusBar,
		pageList:     []Page{},
		selectedIdx:  -1,
		loading:      true,
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
	return lp.fetchPagesCmd()
}

// Update handles messages and returns the updated model and command.
func (lp *ListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pagesLoadedMsg:
		lp.loading = false
		if msg.err != nil {
			lp.err = msg.err
			lp.statusBar.SetSyncStatus(components.StatusError)
			lp.statusBar.SetHelpText(fmt.Sprintf("Error: %v", msg.err))
			return lp, nil
		}

		lp.err = nil
		lp.pageList = msg.pages
		lp.updateSidebarItems()
		lp.statusBar.SetSyncStatus(components.StatusSynced)
		lp.statusBar.SetHelpText(fmt.Sprintf("%d pages | r: refresh", len(lp.pageList)))
		return lp, nil

	case components.ItemSelectedMsg:
		// Emit navigation message when an item is selected
		lp.selectedIdx = msg.Index
		return lp, func() tea.Msg {
			return NewNavigationMsg(msg.ID)
		}

	case tea.KeyMsg:
		if msg.String() == "r" {
			// Refresh page list
			lp.loading = true
			lp.statusBar.SetSyncStatus(components.StatusSyncing)
			lp.statusBar.SetHelpText("Refreshing...")
			return lp, lp.fetchPagesCmd()
		}

	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		lp.sidebar.SetSize(msg.Width/4, msg.Height-2)
		lp.statusBar.SetWidth(msg.Width)
		return lp, nil
	}

	// Update sidebar
	var cmd tea.Cmd
	lp.sidebar, cmd = lp.sidebar.Update(msg)
	return lp, cmd
}

// View renders the list page.
func (lp *ListPage) View() string {
	if lp.loading {
		loadingStyle := lipgloss.NewStyle().
			Width(lp.width).
			Height(lp.height-2).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color("#7C3AED"))

		main := loadingStyle.Render("Loading pages...")
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

	// Main content placeholder (sidebar is handled by root)
	mainStyle := lipgloss.NewStyle().
		Width(lp.width).
		Height(lp.height-2).
		Align(lipgloss.Center, lipgloss.Center).
		Foreground(lipgloss.Color("#6B7280"))

	main := mainStyle.Render("Select a page from the sidebar")
	status := lp.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, main, status)
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
			title := extractTitle(&p)
			status := extractStatus(&p)

			page := NewPage(
				string(p.ID),
				title,
				status,
				p.LastEditedTime,
			)
			pages = append(pages, page)
		}

		return pagesLoadedMsg{pages: pages}
	}
}

// updateSidebarItems converts the page list to sidebar items.
func (lp *ListPage) updateSidebarItems() {
	items := make([]components.Item, 0, len(lp.pageList))
	for _, page := range lp.pageList {
		desc := fmt.Sprintf("Updated: %s", formatTime(page.UpdatedAt))
		if page.Status != "" {
			desc = fmt.Sprintf("%s | %s", page.Status, desc)
		}

		item := components.NewItem(page.Title, desc, page.ID)
		items = append(items, item)
	}

	lp.sidebar.SetItems(items)
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
