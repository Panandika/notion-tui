package pages

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

const notionLogo = `
⠀⠀⠐⢶⣶⣶⣶⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣄⡀⠀⠀⠀⠀
⢰⣄⠀⠀⠙⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠿⠿⠿⠿⠿⠿⠗⠀⠀⠀
⢸⣿⣷⣄⠀⠀⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⣀⣀⡀
⢸⣿⣿⣿⡇⠀⢠⣶⣶⣶⣶⣶⣶⣶⣶⣶⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠿⠛⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣏⣀⠀⠀⠀⠀⠻⣿⣿⣷⣶⠀⠀⣾⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⠀⠀⠀⠀⠀⠙⣿⣿⣿⠀⠀⣿⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⠀⠀⣷⡀⠀⠀⠈⢿⣿⠀⠀⣿⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⠀⠀⣿⣿⣄⠀⠀⠀⢻⠀⠀⣿⣿⣿⡇
⢸⣿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⠀⠀⣿⣿⣿⣆⠀⠀⠀⠀⠀⣿⣿⣿⡇
⠈⢿⣿⣿⡇⠀⢸⣿⣿⣿⣿⣿⠀⠀⠿⢿⣿⣿⣧⡀⠀⠀⠀⣿⣿⣿⡇
⠀⠀⠻⣿⡇⠀⢸⣿⣿⣿⣯⣤⣤⣤⣤⣾⣿⣿⣿⣷⣦⣴⣶⣿⣿⣿⡇
⠀⠀⠀⠙⠇⠀⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠿⠿⠃
⠀⠀⠀⠀⠀⠀⠈⠉⠉⠉⠉⠉⠉⠉⠉⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀⠀
`

// DashboardPage represents the main dashboard.
type DashboardPage struct {
	width       int
	height      int
	config      *config.Config
	selectedIdx int
	menuItems   []dashboardItem
}

type dashboardItem struct {
	label      string
	actionType string // "search", "switch-db", "open-db"
	targetID   string // database ID for "open-db"
}

// NewDashboardPageInput contains parameters for creating a new DashboardPage.
type NewDashboardPageInput struct {
	Width  int
	Height int
	Config *config.Config
}

// NewDashboardPage creates a new DashboardPage instance.
func NewDashboardPage(input NewDashboardPageInput) DashboardPage {
	items := []dashboardItem{
		{label: "Search Workspace", actionType: "search"},
		{label: "Switch Database", actionType: "switch-db"},
	}

	// Add configured databases to the menu
	if input.Config != nil && len(input.Config.Databases) > 0 {
		for _, db := range input.Config.Databases {
			label := fmt.Sprintf("Open %s", db.Name)
			if db.Icon != "" {
				label = fmt.Sprintf("Open %s %s", db.Icon, db.Name)
			}
			items = append(items, dashboardItem{
				label:      label,
				actionType: "open-db",
				targetID:   db.ID,
			})
		}
	}

	return DashboardPage{
		width:       input.Width,
		height:      input.Height,
		config:      input.Config,
		selectedIdx: 0,
		menuItems:   items,
	}
}

// Init initializes the dashboard page.
func (d *DashboardPage) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated model and command.
func (d *DashboardPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.selectedIdx > 0 {
				d.selectedIdx--
			}
		case "down", "j":
			if d.selectedIdx < len(d.menuItems)-1 {
				d.selectedIdx++
			}
		case "enter":
			return d, d.executeAction()
		}

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
	}
	return d, nil
}

// executeAction performs the action for the selected item.
func (d *DashboardPage) executeAction() tea.Cmd {
	item := d.menuItems[d.selectedIdx]
	switch item.actionType {
	case "search":
		return func() tea.Msg {
			return components.CommandExecutedMsg{ActionType: "search"}
		}
	case "switch-db":
		return func() tea.Msg {
			return components.CommandExecutedMsg{ActionType: "switch-db"}
		}
	case "open-db":
		var targetDB config.DatabaseConfig
		if d.config != nil {
			for _, db := range d.config.Databases {
				if db.ID == item.targetID {
					targetDB = db
					break
				}
			}
		}
		return func() tea.Msg {
			return DatabaseSelectedMsg{
				DatabaseID: item.targetID,
				Database:   targetDB,
			}
		}
	}
	return nil
}

// View renders the dashboard page.
func (d *DashboardPage) View() string {
	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")). // Purple-ish
		Bold(true).
		MarginBottom(1)

	welcomeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true).
		MarginBottom(2)

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(2)

	selectedItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true).
		PaddingLeft(2).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("63"))

	var menuView strings.Builder
	for i, item := range d.menuItems {
		if i == d.selectedIdx {
			menuView.WriteString(selectedItemStyle.Render(item.label) + "\n")
		} else {
			menuView.WriteString(itemStyle.Render(item.label) + "\n")
		}
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logoStyle.Render(notionLogo),
		welcomeStyle.Render("Welcome to Notion TUI"),
		lipgloss.NewStyle().MarginTop(1).Render(menuView.String()),
	)

	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
