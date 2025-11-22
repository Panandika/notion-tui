package pages

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// databaseItem represents a single database in the list.
type databaseItem struct {
	db        config.DatabaseConfig
	isDefault bool
}

// Title returns the database's title for display.
func (d databaseItem) Title() string {
	icon := d.db.Icon
	if icon == "" {
		icon = "📄"
	}
	title := fmt.Sprintf("%s %s", icon, d.db.Name)
	if d.isDefault {
		title += " ✓"
	}
	return title
}

// Description returns the database's description for display.
func (d databaseItem) Description() string {
	desc := fmt.Sprintf("ID: %s", d.db.ID)
	if d.isDefault {
		desc += " (default)"
	}
	return desc
}

// FilterValue returns the value used for fuzzy filtering.
func (d databaseItem) FilterValue() string {
	return d.db.Name
}

// DatabaseSelectedMsg is sent when a database is selected.
type DatabaseSelectedMsg struct {
	DatabaseID string
	Database   config.DatabaseConfig
}

// DatabaseListPage is a page component for selecting databases.
type DatabaseListPage struct {
	list         list.Model
	statusBar    components.StatusBar
	databases    []config.DatabaseConfig
	defaultDBID  string
	selectedDBID string
	width        int
	height       int
	styles       DatabaseListPageStyles
}

// DatabaseListPageStyles holds the styles for the database list page.
type DatabaseListPageStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Default   lipgloss.Style
}

// DefaultDatabaseListPageStyles returns the default styles.
func DefaultDatabaseListPageStyles() DatabaseListPageStyles {
	return DatabaseListPageStyles{
		Container: lipgloss.NewStyle().
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			MarginBottom(1),
		Default: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true),
	}
}

// NewDatabaseListPageInput contains the parameters for creating a new DatabaseListPage.
type NewDatabaseListPageInput struct {
	Width       int
	Height      int
	Databases   []config.DatabaseConfig
	DefaultDBID string
}

// NewDatabaseListPage creates a new DatabaseListPage instance.
func NewDatabaseListPage(input NewDatabaseListPageInput) DatabaseListPage {
	// Create list items
	items := make([]list.Item, 0, len(input.Databases))
	for _, db := range input.Databases {
		items = append(items, databaseItem{
			db:        db,
			isDefault: db.ID == input.DefaultDBID,
		})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, input.Width-4, input.Height-6)
	l.Title = "Select Database"
	l.SetShowStatusBar(false)
	l.SetShowHelp(true)
	l.DisableQuitKeybindings()

	// Create status bar
	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText("Enter: select database | ESC: back")

	styles := DefaultDatabaseListPageStyles()
	l.Styles.Title = styles.Title

	return DatabaseListPage{
		list:         l,
		statusBar:    statusBar,
		databases:    input.Databases,
		defaultDBID:  input.DefaultDBID,
		selectedDBID: input.DefaultDBID,
		width:        input.Width,
		height:       input.Height,
		styles:       styles,
	}
}

// Init initializes the database list page component.
func (dlp *DatabaseListPage) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated model and command.
func (dlp *DatabaseListPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// Select database
			if item, ok := dlp.list.SelectedItem().(databaseItem); ok {
				dlp.selectedDBID = item.db.ID
				return dlp, func() tea.Msg {
					return DatabaseSelectedMsg{
						DatabaseID: item.db.ID,
						Database:   item.db,
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		dlp.width = msg.Width
		dlp.height = msg.Height
		dlp.list.SetSize(msg.Width-4, msg.Height-6)
		dlp.statusBar.SetWidth(msg.Width)
		return dlp, nil
	}

	// Update list
	var cmd tea.Cmd
	dlp.list, cmd = dlp.list.Update(msg)

	return dlp, cmd
}

// View renders the database list page.
func (dlp *DatabaseListPage) View() string {
	// Build main content
	listView := dlp.list.View()

	mainContent := dlp.styles.Container.Render(listView)

	// Add status bar
	statusView := dlp.statusBar.View()
	finalView := lipgloss.JoinVertical(lipgloss.Left, mainContent, statusView)

	return finalView
}

// SetDatabases updates the database list.
func (dlp *DatabaseListPage) SetDatabases(databases []config.DatabaseConfig, defaultDBID string) {
	dlp.databases = databases
	dlp.defaultDBID = defaultDBID

	items := make([]list.Item, 0, len(databases))
	for _, db := range databases {
		items = append(items, databaseItem{
			db:        db,
			isDefault: db.ID == defaultDBID,
		})
	}

	dlp.list.SetItems(items)
}

// SelectedDatabaseID returns the currently selected database ID.
func (dlp *DatabaseListPage) SelectedDatabaseID() string {
	return dlp.selectedDBID
}

// Databases returns the list of databases.
func (dlp *DatabaseListPage) Databases() []config.DatabaseConfig {
	return dlp.databases
}
