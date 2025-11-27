package pages

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// SearchMode defines the search scope.
type SearchMode string

const (
	// SearchModeDatabase searches within the current database only.
	SearchModeDatabase SearchMode = "database"
	// SearchModeWorkspace searches across the entire workspace.
	SearchModeWorkspace SearchMode = "workspace"
)

// SearchResult represents a single search result.
type SearchResult struct {
	PageID     string
	Title      string
	Snippet    string
	MatchType  string // "title", "property", "content"
	ObjectType string // "page" or "database" (for workspace search)
}

// searchResultItem wraps SearchResult for use in bubbles/list.
type searchResultItem struct {
	result SearchResult
}

// Title returns the result's title for display.
func (s searchResultItem) Title() string {
	return s.result.Title
}

// Description returns the result's description for display.
func (s searchResultItem) Description() string {
	// Determine icon based on object type and match type
	typeIcon := "📄"
	if s.result.ObjectType == "database" {
		typeIcon = "📊"
	} else if s.result.MatchType == "title" {
		typeIcon = "🔍"
	}

	// For workspace search, show object type
	if s.result.ObjectType != "" {
		return fmt.Sprintf("%s [%s] %s", typeIcon, strings.ToUpper(s.result.ObjectType[:1])+s.result.ObjectType[1:], s.result.Snippet)
	}

	return fmt.Sprintf("%s %s | %s", typeIcon, s.result.MatchType, s.result.Snippet)
}

// FilterValue returns the value used for fuzzy filtering.
func (s searchResultItem) FilterValue() string {
	return s.result.Title
}

// searchResultsMsg is sent when search results are fetched.
type searchResultsMsg struct {
	results    []SearchResult
	query      string
	err        error
	hasMore    bool
	nextCursor string
}

// SearchNavigationMsg is sent when a search result is selected for navigation.
type SearchNavigationMsg struct {
	ID         string
	ObjectType string // "page" or "database"
}

// PageID returns the page/database ID for navigation.
func (m SearchNavigationMsg) PageID() string {
	return m.ID
}

// SearchPage is a page component for cross-page search.
type SearchPage struct {
	input        textinput.Model
	resultsList  list.Model
	statusBar    components.StatusBar
	spinner      components.Spinner
	results      []SearchResult
	query        string
	searching    bool
	err          error
	width        int
	height       int
	notionClient NotionClient
	cache        *cache.PageCache
	databaseID   string
	styles       SearchPageStyles
	mode         SearchMode // database or workspace
	hasMore      bool       // pagination: more results available
	nextCursor   string     // pagination: cursor for next page
}

// SearchPageStyles holds the styles for the search page.
type SearchPageStyles struct {
	Container    lipgloss.Style
	InputLabel   lipgloss.Style
	ResultsTitle lipgloss.Style
	NoResults    lipgloss.Style
	Error        lipgloss.Style
	Highlight    lipgloss.Style
}

// DefaultSearchPageStyles returns the default styles for the search page.
func DefaultSearchPageStyles() SearchPageStyles {
	return SearchPageStyles{
		Container: lipgloss.NewStyle().
			Padding(1, 2),
		InputLabel: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			MarginBottom(1),
		ResultsTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true).
			MarginTop(1).
			MarginBottom(1),
		NoResults: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true),
		Highlight: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FBBF24")).
			Bold(true),
	}
}

// NewSearchPageInput contains the parameters for creating a new SearchPage.
type NewSearchPageInput struct {
	Width        int
	Height       int
	NotionClient NotionClient
	Cache        *cache.PageCache
	DatabaseID   string
	Mode         SearchMode // Default: SearchModeWorkspace
}

// NewSearchPage creates a new SearchPage instance.
func NewSearchPage(input NewSearchPageInput) SearchPage {
	// Default to workspace mode
	mode := input.Mode
	if mode == "" {
		mode = SearchModeWorkspace
	}

	// Create text input for search query
	ti := textinput.New()
	if mode == SearchModeWorkspace {
		ti.Placeholder = "Search workspace (pages & databases)..."
	} else {
		ti.Placeholder = "Search in current database..."
	}
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = input.Width - 10

	// Create results list
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	resultsList := list.New([]list.Item{}, delegate, input.Width-4, input.Height-10)
	resultsList.Title = "Search Results"
	resultsList.SetShowStatusBar(false)
	resultsList.SetShowHelp(false)
	resultsList.DisableQuitKeybindings()

	// Create status bar
	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)

	helpText := "Type to search | Enter: select | Tab: toggle mode | ESC: back"
	statusBar.SetHelpText(helpText)

	spinner := components.NewSpinner("Searching...")

	styles := DefaultSearchPageStyles()
	resultsList.Styles.Title = styles.ResultsTitle

	return SearchPage{
		input:        ti,
		resultsList:  resultsList,
		statusBar:    statusBar,
		spinner:      spinner,
		results:      []SearchResult{},
		query:        "",
		searching:    false,
		err:          nil,
		width:        input.Width,
		height:       input.Height,
		notionClient: input.NotionClient,
		cache:        input.Cache,
		databaseID:   input.DatabaseID,
		styles:       styles,
		mode:         mode,
		hasMore:      false,
		nextCursor:   "",
	}
}

// Init initializes the search page component.
func (sp *SearchPage) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and returns the updated model and command.
func (sp *SearchPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case searchResultsMsg:
		sp.searching = false
		if msg.err != nil {
			sp.err = msg.err
			sp.statusBar.SetSyncStatus(components.StatusError)
			sp.statusBar.SetHelpText(fmt.Sprintf("Error: %v", msg.err))
			return sp, nil
		}

		sp.err = nil
		sp.results = msg.results
		sp.query = msg.query
		sp.hasMore = msg.hasMore
		sp.nextCursor = msg.nextCursor
		sp.updateResultsList()

		modeLabel := "workspace"
		if sp.mode == SearchModeDatabase {
			modeLabel = "database"
		}
		helpText := fmt.Sprintf("%d results (%s) | Enter: select | Tab: toggle mode | ESC: back", len(sp.results), modeLabel)
		if sp.hasMore {
			helpText = fmt.Sprintf("%d+ results (%s) | Enter: select | Tab: toggle mode | ESC: back", len(sp.results), modeLabel)
		}
		sp.statusBar.SetHelpText(helpText)
		sp.statusBar.SetSyncStatus(components.StatusSynced)
		return sp, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// If search input is focused and has text, perform search
			if sp.input.Focused() && sp.input.Value() != "" {
				sp.searching = true
				sp.statusBar.SetSyncStatus(components.StatusSyncing)
				sp.statusBar.SetHelpText("Searching...")
				return sp, sp.searchCmd()
			}

			// If results list has selection, navigate to that page/database
			if len(sp.results) > 0 && !sp.input.Focused() {
				if item, ok := sp.resultsList.SelectedItem().(searchResultItem); ok {
					return sp, func() tea.Msg {
						return SearchNavigationMsg{
							ID:         item.result.PageID,
							ObjectType: item.result.ObjectType,
						}
					}
				}
			}

		case "tab":
			// Toggle between input focus and results, or toggle search mode
			if sp.input.Focused() && len(sp.results) > 0 {
				// If we have results, move focus to results list
				sp.input.Blur()
			} else if !sp.input.Focused() {
				// If in results, toggle mode and refocus input
				sp.toggleMode()
				sp.input.Focus()
			} else {
				// Toggle search mode
				sp.toggleMode()
			}
			return sp, nil

		case "ctrl+t":
			// Explicit mode toggle
			sp.toggleMode()
			return sp, nil

		case "esc":
			// If filtering in results list, clear filter
			if sp.resultsList.SettingFilter() {
				sp.resultsList.ResetFilter()
				return sp, nil
			}
			// Otherwise, let parent handle (go back)
		}

	case tea.WindowSizeMsg:
		sp.width = msg.Width
		sp.height = msg.Height
		sp.input.Width = msg.Width - 10
		sp.resultsList.SetSize(msg.Width-4, msg.Height-10)
		sp.statusBar.SetWidth(msg.Width)
		return sp, nil
	}

	// Update spinner when searching
	if sp.searching {
		var spinnerCmd tea.Cmd
		sp.spinner, spinnerCmd = sp.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	// Update input
	if sp.input.Focused() {
		var inputCmd tea.Cmd
		sp.input, inputCmd = sp.input.Update(msg)
		cmds = append(cmds, inputCmd)
	} else {
		// Update results list only when not focused on input
		var listCmd tea.Cmd
		sp.resultsList, listCmd = sp.resultsList.Update(msg)
		cmds = append(cmds, listCmd)
	}

	return sp, tea.Batch(cmds...)
}

// toggleMode switches between database and workspace search modes.
func (sp *SearchPage) toggleMode() {
	if sp.mode == SearchModeWorkspace {
		sp.mode = SearchModeDatabase
		sp.input.Placeholder = "Search in current database..."
	} else {
		sp.mode = SearchModeWorkspace
		sp.input.Placeholder = "Search workspace (pages & databases)..."
	}

	// Clear results when switching modes
	sp.results = []SearchResult{}
	sp.query = ""
	sp.hasMore = false
	sp.nextCursor = ""
	sp.updateResultsList()

	modeLabel := "workspace"
	if sp.mode == SearchModeDatabase {
		modeLabel = "database"
	}
	sp.statusBar.SetHelpText(fmt.Sprintf("Mode: %s | Type to search | Tab: toggle mode | ESC: back", modeLabel))
}

// View renders the search page.
func (sp *SearchPage) View() string {
	if sp.searching {
		loadingStyle := lipgloss.NewStyle().
			Width(sp.width).
			Height(sp.height-2).
			Align(lipgloss.Center, lipgloss.Center)

		main := loadingStyle.Render(sp.spinner.View())
		status := sp.statusBar.View()

		return lipgloss.JoinVertical(lipgloss.Left, main, status)
	}

	// Build search input section
	modeIndicator := "Workspace"
	if sp.mode == SearchModeDatabase {
		modeIndicator = "Database"
	}
	inputLabel := sp.styles.InputLabel.Render(fmt.Sprintf("Search (%s)", modeIndicator))
	inputView := sp.input.View()
	inputSection := lipgloss.JoinVertical(lipgloss.Left, inputLabel, inputView)

	// Build results section
	var resultsSection string
	if sp.err != nil {
		resultsSection = sp.styles.Error.Render(fmt.Sprintf("Error: %v", sp.err))
	} else if len(sp.results) == 0 && sp.query != "" {
		resultsSection = sp.styles.NoResults.Render("No results found")
	} else if len(sp.results) > 0 {
		resultsSection = sp.resultsList.View()
	} else {
		resultsSection = sp.styles.NoResults.Render("Type to search across all pages")
	}

	// Combine sections
	mainContent := sp.styles.Container.Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			inputSection,
			"",
			resultsSection,
		),
	)

	// Add status bar
	statusView := sp.statusBar.View()
	finalView := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusView)

	return finalView
}

// searchCmd returns a command that performs the search operation.
func (sp *SearchPage) searchCmd() tea.Cmd {
	query := sp.input.Value()
	mode := sp.mode
	databaseID := sp.databaseID

	return func() tea.Msg {
		ctx := context.Background()

		if sp.notionClient == nil {
			return searchResultsMsg{
				err: fmt.Errorf("notion client not initialized"),
			}
		}

		// Use workspace search or database search based on mode
		if mode == SearchModeWorkspace {
			return sp.searchWorkspace(ctx, query)
		}

		return sp.searchDatabase(ctx, query, databaseID)
	}
}

// searchWorkspace performs a workspace-wide search using the Notion Search API.
func (sp *SearchPage) searchWorkspace(ctx context.Context, query string) searchResultsMsg {
	resp, err := sp.notionClient.Search(ctx, notion.SearchInput{
		Query:    query,
		PageSize: 20,
	})
	if err != nil {
		return searchResultsMsg{
			err: fmt.Errorf("workspace search: %w", err),
		}
	}

	results := make([]SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		snippet := ""
		if r.ParentType == "workspace" {
			snippet = "Root level"
		} else if r.ParentType == "database_id" {
			snippet = "In database"
		} else if r.ParentType == "page_id" {
			snippet = "Child page"
		}

		results = append(results, SearchResult{
			PageID:     r.ID,
			Title:      r.Title,
			Snippet:    snippet,
			MatchType:  "title",
			ObjectType: r.ObjectType,
		})
	}

	return searchResultsMsg{
		results:    results,
		query:      query,
		hasMore:    resp.HasMore,
		nextCursor: resp.NextCursor,
	}
}

// searchDatabase performs a search within the current database.
func (sp *SearchPage) searchDatabase(ctx context.Context, query, databaseID string) searchResultsMsg {
	if databaseID == "" {
		return searchResultsMsg{
			err: fmt.Errorf("no database configured - use workspace search mode"),
		}
	}

	resp, err := sp.notionClient.QueryDatabase(ctx, databaseID, nil)
	if err != nil {
		return searchResultsMsg{
			err: fmt.Errorf("fetch pages: %w", err),
		}
	}

	results := make([]SearchResult, 0)
	queryLower := strings.ToLower(query)

	for _, p := range resp.Results {
		title := extractTitle(&p)
		titleLower := strings.ToLower(title)

		// Match on title
		if strings.Contains(titleLower, queryLower) {
			snippet := sp.generateSnippet(title, query)
			results = append(results, SearchResult{
				PageID:     string(p.ID),
				Title:      title,
				Snippet:    snippet,
				MatchType:  "title",
				ObjectType: "page",
			})
			continue
		}

		// Match on status property
		status := extractStatus(&p)
		if status != "" && strings.Contains(strings.ToLower(status), queryLower) {
			results = append(results, SearchResult{
				PageID:     string(p.ID),
				Title:      title,
				Snippet:    fmt.Sprintf("Status: %s", status),
				MatchType:  "property",
				ObjectType: "page",
			})
		}
	}

	return searchResultsMsg{
		results: results,
		query:   query,
	}
}

// generateSnippet creates a highlighted snippet showing the match.
func (sp *SearchPage) generateSnippet(text, query string) string {
	queryLower := strings.ToLower(query)
	textLower := strings.ToLower(text)

	index := strings.Index(textLower, queryLower)
	if index == -1 {
		// Fallback if not found
		if len(text) > 50 {
			return text[:50] + "..."
		}
		return text
	}

	// Calculate snippet window
	start := index - 20
	if start < 0 {
		start = 0
	}

	end := index + len(query) + 30
	if end > len(text) {
		end = len(text)
	}

	snippet := text[start:end]

	// Add ellipsis
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(text) {
		snippet = snippet + "..."
	}

	return snippet
}

// updateResultsList converts search results to list items.
func (sp *SearchPage) updateResultsList() {
	items := make([]list.Item, 0, len(sp.results))
	for _, result := range sp.results {
		items = append(items, searchResultItem{result: result})
	}
	sp.resultsList.SetItems(items)

	// Update title with count
	sp.resultsList.Title = fmt.Sprintf("Search Results (%d)", len(sp.results))
}

// Results returns the current search results.
func (sp *SearchPage) Results() []SearchResult {
	return sp.results
}

// Query returns the current search query.
func (sp *SearchPage) Query() string {
	return sp.query
}

// IsSearching returns whether a search is in progress.
func (sp *SearchPage) IsSearching() bool {
	return sp.searching
}

// Mode returns the current search mode.
func (sp *SearchPage) Mode() SearchMode {
	return sp.mode
}

// SetMode sets the search mode.
func (sp *SearchPage) SetMode(mode SearchMode) {
	sp.mode = mode
	if mode == SearchModeWorkspace {
		sp.input.Placeholder = "Search workspace (pages & databases)..."
	} else {
		sp.input.Placeholder = "Search in current database..."
	}
}
