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
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// SearchResult represents a single search result.
type SearchResult struct {
	PageID    string
	Title     string
	Snippet   string
	MatchType string // "title", "property", "content"
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
	matchIcon := "📄"
	if s.result.MatchType == "title" {
		matchIcon = "🔍"
	}
	return fmt.Sprintf("%s %s | %s", matchIcon, s.result.MatchType, s.result.Snippet)
}

// FilterValue returns the value used for fuzzy filtering.
func (s searchResultItem) FilterValue() string {
	return s.result.Title
}

// searchResultsMsg is sent when search results are fetched.
type searchResultsMsg struct {
	results []SearchResult
	query   string
	err     error
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
}

// NewSearchPage creates a new SearchPage instance.
func NewSearchPage(input NewSearchPageInput) SearchPage {
	// Create text input for search query
	ti := textinput.New()
	ti.Placeholder = "Search across all pages..."
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
	statusBar.SetHelpText("Type to search | Enter: navigate to page | ESC: back")

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
		sp.updateResultsList()

		helpText := fmt.Sprintf("%d results | Enter: navigate | ESC: back", len(sp.results))
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

			// If results list has selection, navigate to that page
			if len(sp.results) > 0 && !sp.input.Focused() {
				if item, ok := sp.resultsList.SelectedItem().(searchResultItem); ok {
					return sp, func() tea.Msg {
						return NewNavigationMsg(item.result.PageID)
					}
				}
			}

		case "tab":
			// Toggle focus between input and results
			if sp.input.Focused() {
				sp.input.Blur()
			} else {
				sp.input.Focus()
			}
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
	inputLabel := sp.styles.InputLabel.Render("Search Pages")
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

	return func() tea.Msg {
		ctx := context.Background()

		if sp.notionClient == nil {
			return searchResultsMsg{
				err: fmt.Errorf("notion client not initialized"),
			}
		}

		// Fetch all pages from database
		resp, err := sp.notionClient.QueryDatabase(ctx, sp.databaseID, nil)
		if err != nil {
			return searchResultsMsg{
				err: fmt.Errorf("fetch pages: %w", err),
			}
		}

		// Search through results
		results := make([]SearchResult, 0)
		queryLower := strings.ToLower(query)

		for _, p := range resp.Results {
			title := extractTitle(&p)
			titleLower := strings.ToLower(title)

			// Match on title
			if strings.Contains(titleLower, queryLower) {
				snippet := sp.generateSnippet(title, query)
				results = append(results, SearchResult{
					PageID:    string(p.ID),
					Title:     title,
					Snippet:   snippet,
					MatchType: "title",
				})
				continue
			}

			// Match on status property
			status := extractStatus(&p)
			if status != "" && strings.Contains(strings.ToLower(status), queryLower) {
				results = append(results, SearchResult{
					PageID:    string(p.ID),
					Title:     title,
					Snippet:   fmt.Sprintf("Status: %s", status),
					MatchType: "property",
				})
			}
		}

		return searchResultsMsg{
			results: results,
			query:   query,
		}
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
