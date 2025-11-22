package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Item represents a single item in the sidebar list.
type Item struct {
	title string
	desc  string
	id    string
}

// NewItem creates a new sidebar item.
func NewItem(title, desc, id string) Item {
	return Item{
		title: title,
		desc:  desc,
		id:    id,
	}
}

// Title returns the item's title for display.
func (i Item) Title() string {
	return i.title
}

// Description returns the item's description for display.
func (i Item) Description() string {
	return i.desc
}

// FilterValue returns the value used for fuzzy filtering.
func (i Item) FilterValue() string {
	return i.title
}

// ID returns the item's unique identifier.
func (i Item) ID() string {
	return i.id
}

// ItemSelectedMsg is sent when an item is selected via Enter key.
type ItemSelectedMsg struct {
	ID    string
	Title string
	Index int
}

// Sidebar is a list-based sidebar component with fuzzy search.
type Sidebar struct {
	list       list.Model
	width      int
	height     int
	selectedID string
	styles     SidebarStyles
}

// SidebarStyles holds the styles for the sidebar.
type SidebarStyles struct {
	List       lipgloss.Style
	Title      lipgloss.Style
	Item       lipgloss.Style
	SelectedID lipgloss.Style
}

// DefaultSidebarStyles returns the default styles for the sidebar.
func DefaultSidebarStyles() SidebarStyles {
	return SidebarStyles{
		List: lipgloss.NewStyle().
			Padding(1, 0),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			MarginBottom(1),
		Item: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F3F4F6")),
		SelectedID: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true),
	}
}

// NewSidebarInput contains options for creating a new sidebar.
type NewSidebarInput struct {
	Items  []Item
	Width  int
	Height int
	Title  string
}

// NewSidebar creates a new sidebar component.
func NewSidebar(input NewSidebarInput) Sidebar {
	items := make([]list.Item, len(input.Items))
	for i, item := range input.Items {
		items[i] = item
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	l := list.New(items, delegate, input.Width, input.Height)
	l.Title = input.Title
	if l.Title == "" {
		l.Title = "Pages"
	}
	l.SetFilteringEnabled(true)
	l.SetShowStatusBar(true)
	l.SetShowHelp(true)
	l.DisableQuitKeybindings()

	styles := DefaultSidebarStyles()
	l.Styles.Title = styles.Title

	sidebar := Sidebar{
		list:   l,
		width:  input.Width,
		height: input.Height,
		styles: styles,
	}

	if len(input.Items) > 0 {
		sidebar.selectedID = input.Items[0].id
	}

	return sidebar
}

// Init initializes the sidebar component.
func (s Sidebar) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated sidebar and command.
func (s Sidebar) Update(msg tea.Msg) (Sidebar, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Note: '/' key will be handled by the list itself to activate filtering
		// We just need to ensure filtering is enabled (done in NewSidebar)

		if msg.String() == "enter" && !s.list.SettingFilter() {
			if item, ok := s.list.SelectedItem().(Item); ok {
				s.selectedID = item.id
				return s, func() tea.Msg {
					return ItemSelectedMsg{
						ID:    item.id,
						Title: item.title,
						Index: s.list.Index(),
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.list.SetSize(msg.Width, msg.Height)
	}

	s.list, cmd = s.list.Update(msg)

	if selectedItem, ok := s.list.SelectedItem().(Item); ok {
		s.selectedID = selectedItem.id
	}

	return s, cmd
}

// View renders the sidebar.
func (s Sidebar) View() string {
	return s.styles.List.Render(s.list.View())
}

// SetItems replaces all items in the sidebar.
func (s *Sidebar) SetItems(items []Item) {
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}
	s.list.SetItems(listItems)

	if len(items) > 0 {
		if selectedItem, ok := s.list.SelectedItem().(Item); ok {
			s.selectedID = selectedItem.id
		} else {
			s.selectedID = items[0].id
		}
	} else {
		s.selectedID = ""
	}
}

// SelectedID returns the ID of the currently selected item.
func (s Sidebar) SelectedID() string {
	return s.selectedID
}

// SelectedIndex returns the index of the currently selected item.
func (s Sidebar) SelectedIndex() int {
	return s.list.Index()
}

// SetSize updates the sidebar dimensions.
func (s *Sidebar) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.list.SetSize(width, height)
}

// Width returns the sidebar width.
func (s Sidebar) Width() int {
	return s.width
}

// Height returns the sidebar height.
func (s Sidebar) Height() int {
	return s.height
}

// Items returns the current items in the sidebar.
func (s Sidebar) Items() []Item {
	listItems := s.list.Items()
	items := make([]Item, 0, len(listItems))
	for _, li := range listItems {
		if item, ok := li.(Item); ok {
			items = append(items, item)
		}
	}
	return items
}

// FilterState returns the current filter state.
func (s Sidebar) FilterState() list.FilterState {
	return s.list.FilterState()
}

// IsFiltering returns true if the sidebar is currently filtering.
func (s Sidebar) IsFiltering() bool {
	return s.list.SettingFilter()
}

// FilterValue returns the current filter value.
func (s Sidebar) FilterValue() string {
	return s.list.FilterValue()
}

// VisibleItemCount returns the number of items after filtering.
func (s Sidebar) VisibleItemCount() int {
	return len(s.list.VisibleItems())
}
