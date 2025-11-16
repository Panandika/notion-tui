package ui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/notion"
)

// Model represents the root TUI model.
type Model struct {
	config       *config.Config
	notionClient *notion.Client
	pages        []string
	pageIDs      []string
	cursor       int
	width        int
	height       int
	ready        bool
	err          error
}

// NewModel creates a new root TUI model.
func NewModel(cfg *config.Config) Model {
	return Model{
		config:       cfg,
		notionClient: notion.NewClient(cfg.NotionToken),
		pages:        []string{},
		pageIDs:      []string{},
		cursor:       0,
		ready:        false,
	}
}

// Init returns the initial command (fetch pages from database).
func (m Model) Init() tea.Cmd {
	if m.config.DatabaseID == "" {
		return func() tea.Msg {
			return pagesLoadedMsg{
				titles: []string{"Error: no database ID configured"},
				ids:    []string{},
				err:    fmt.Errorf("database ID not set in config"),
			}
		}
	}
	return fetchPagesCmd(m.notionClient, m.config.DatabaseID)
}

// Update handles all messages and returns updated model and command.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.pages)-1 {
				m.cursor++
			}
		case "r":
			return m, fetchPagesCmd(m.notionClient, m.config.DatabaseID)
		}

	case pagesLoadedMsg:
		m.pages = msg.titles
		m.pageIDs = msg.ids
		m.err = msg.err
		if m.cursor >= len(m.pages) && len(m.pages) > 0 {
			m.cursor = len(m.pages) - 1
		}
	}

	return m, nil
}

// View returns the rendered UI.
func (m Model) View() string {
	if !m.ready {
		return "Initializing...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if len(m.pages) == 0 {
		return "No pages found in database.\n(r: refresh, q: quit)\n"
	}

	s := "Notion Pages\n\n"
	for i, page := range m.pages {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, page)
	}
	s += fmt.Sprintf("\n(%d/%d)\n", m.cursor+1, len(m.pages))
	s += "(↑/↓ or k/j: navigate, r: refresh, q: quit)\n"
	return s
}

// pagesLoadedMsg is sent when pages are loaded from the database.
type pagesLoadedMsg struct {
	titles []string
	ids    []string
	err    error
}

// fetchPagesCmd fetches pages from the Notion database.
func fetchPagesCmd(client *notion.Client, dbID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		resp, err := client.QueryDatabase(ctx, dbID, nil)
		if err != nil {
			return pagesLoadedMsg{err: err}
		}

		titles := make([]string, len(resp.Results))
		ids := make([]string, len(resp.Results))

		for i, page := range resp.Results {
			ids[i] = page.ID.String()
			title := extractTitle(page)
			if title == "" {
				title = "(Untitled)"
			}
			titles[i] = title
		}

		return pagesLoadedMsg{titles: titles, ids: ids}
	}
}

// extractTitle extracts the title from a Notion page.
func extractTitle(page notionapi.Page) string {
	// Try to find a Title property
	for key, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			if len(titleProp.Title) > 0 {
				return titleProp.Title[0].PlainText
			}
		}
		// Some databases use "Name" instead of "Title"
		if key == "Name" || key == "Title" {
			if titleProp, ok := prop.(*notionapi.RichTextProperty); ok {
				if len(titleProp.RichText) > 0 {
					return titleProp.RichText[0].PlainText
				}
			}
		}
	}

	// Fallback to page ID
	return page.ID.String()
}
