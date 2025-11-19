package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// commandItem represents a single command in the palette.
type commandItem struct {
	name        string
	description string
	action      func() tea.Cmd
}

// Title returns the command's name for display.
func (c commandItem) Title() string {
	return c.name
}

// Description returns the command's description for display.
func (c commandItem) Description() string {
	return c.description
}

// FilterValue returns the value used for fuzzy filtering.
func (c commandItem) FilterValue() string {
	return c.name
}

// CommandExecutedMsg is sent when a command is executed.
type CommandExecutedMsg struct {
	CommandName string
}

// CommandPalette is a fuzzy-searchable command palette component.
type CommandPalette struct {
	list   list.Model
	isOpen bool
	width  int
	height int
	styles CommandPaletteStyles
}

// CommandPaletteStyles holds the styles for the command palette.
type CommandPaletteStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
}

// DefaultCommandPaletteStyles returns the default styles for the command palette.
func DefaultCommandPaletteStyles() CommandPaletteStyles {
	return CommandPaletteStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			MarginBottom(1),
	}
}

// NewCommandPalette creates a new command palette with built-in commands.
func NewCommandPalette() CommandPalette {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New([]list.Item{}, delegate, 60, 15)
	l.Title = "Command Palette"
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.DisableQuitKeybindings()

	styles := DefaultCommandPaletteStyles()
	l.Styles.Title = styles.Title

	palette := CommandPalette{
		list:   l,
		isOpen: false,
		width:  60,
		height: 15,
		styles: styles,
	}

	// Add built-in commands
	palette.addBuiltInCommands()

	return palette
}

// addBuiltInCommands adds the default built-in commands.
func (p *CommandPalette) addBuiltInCommands() {
	builtInCommands := []commandItem{
		{
			name:        "New Page",
			description: "Create a new Notion page",
			action:      func() tea.Cmd { return nil },
		},
		{
			name:        "Search All",
			description: "Search across all pages and content",
			action:      func() tea.Cmd { return nil },
		},
		{
			name:        "Refresh",
			description: "Refresh current view from Notion",
			action:      func() tea.Cmd { return nil },
		},
		{
			name:        "Quit",
			description: "Exit the application",
			action:      func() tea.Cmd { return tea.Quit },
		},
	}

	items := make([]list.Item, len(builtInCommands))
	for i, cmd := range builtInCommands {
		items[i] = cmd
	}
	p.list.SetItems(items)
}

// Init initializes the command palette component.
func (p CommandPalette) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated palette and command.
func (p CommandPalette) Update(msg tea.Msg) (CommandPalette, tea.Cmd) {
	if !p.isOpen {
		return p, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if !p.list.SettingFilter() {
				if item, ok := p.list.SelectedItem().(commandItem); ok {
					p.isOpen = false
					// Reset filter when closing
					p.list.ResetFilter()
					// Execute the command's action
					actionCmd := item.action()
					return p, tea.Batch(
						func() tea.Msg {
							return CommandExecutedMsg{CommandName: item.name}
						},
						actionCmd,
					)
				}
			}
		case "esc":
			p.isOpen = false
			p.list.ResetFilter()
			return p, nil
		}
	}

	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

// View renders the command palette.
func (p CommandPalette) View() string {
	if !p.isOpen {
		return ""
	}
	return p.styles.Container.Render(p.list.View())
}

// Open shows the command palette.
func (p *CommandPalette) Open() {
	p.isOpen = true
}

// Close hides the command palette.
func (p *CommandPalette) Close() {
	p.isOpen = false
	p.list.ResetFilter()
}

// IsOpen returns true if the palette is currently open.
func (p CommandPalette) IsOpen() bool {
	return p.isOpen
}

// AddCommand adds a custom command to the palette.
func (p *CommandPalette) AddCommand(name, desc string, action func() tea.Cmd) {
	newCmd := commandItem{
		name:        name,
		description: desc,
		action:      action,
	}

	currentItems := p.list.Items()
	newItems := make([]list.Item, len(currentItems)+1)
	copy(newItems, currentItems)
	newItems[len(currentItems)] = newCmd

	p.list.SetItems(newItems)
}

// SetSize updates the palette dimensions.
func (p *CommandPalette) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.list.SetSize(width, height)
}

// Width returns the palette width.
func (p CommandPalette) Width() int {
	return p.width
}

// Height returns the palette height.
func (p CommandPalette) Height() int {
	return p.height
}

// Commands returns the current commands in the palette.
func (p CommandPalette) Commands() []commandItem {
	listItems := p.list.Items()
	commands := make([]commandItem, 0, len(listItems))
	for _, li := range listItems {
		if cmd, ok := li.(commandItem); ok {
			commands = append(commands, cmd)
		}
	}
	return commands
}

// SelectedIndex returns the index of the currently selected command.
func (p CommandPalette) SelectedIndex() int {
	return p.list.Index()
}

// FilterState returns the current filter state.
func (p CommandPalette) FilterState() list.FilterState {
	return p.list.FilterState()
}

// IsFiltering returns true if the palette is currently filtering.
func (p CommandPalette) IsFiltering() bool {
	return p.list.SettingFilter()
}
