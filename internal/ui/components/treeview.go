package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TreeViewStyles holds the styles for the tree view.
type TreeViewStyles struct {
	Container    lipgloss.Style
	Title        lipgloss.Style
	Node         lipgloss.Style
	SelectedNode lipgloss.Style
	DatabaseIcon lipgloss.Style
	PageIcon     lipgloss.Style
	Indent       lipgloss.Style
	ExpandIcon   lipgloss.Style
}

// DefaultTreeViewStyles returns the default styles for the tree view.
func DefaultTreeViewStyles() TreeViewStyles {
	return TreeViewStyles{
		Container: lipgloss.NewStyle().
			Padding(1, 1),
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7C3AED")).
			Bold(true).
			MarginBottom(1),
		Node: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")),
		SelectedNode: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981")).
			Bold(true),
		DatabaseIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")),
		PageIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#60A5FA")),
		Indent: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4B5563")),
		ExpandIcon: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF")),
	}
}

// TreeNavigationMsg is sent when user wants to navigate to a node.
type TreeNavigationMsg struct {
	ID         string
	ObjectType string // "database" or "page"
}

// TreeView is a component that renders a navigation tree.
type TreeView struct {
	tree    *NavTree
	title   string
	width   int
	height  int
	styles  TreeViewStyles
	focused bool
	loading bool
	err     error
}

// NewTreeViewInput contains parameters for creating a TreeView.
type NewTreeViewInput struct {
	Title  string
	Width  int
	Height int
}

// NewTreeView creates a new tree view component.
func NewTreeView(input NewTreeViewInput) TreeView {
	title := input.Title
	if title == "" {
		title = "Workspace"
	}

	return TreeView{
		tree:    NewNavTree(),
		title:   title,
		width:   input.Width,
		height:  input.Height,
		styles:  DefaultTreeViewStyles(),
		focused: false,
		loading: true,
		err:     nil,
	}
}

// Init initializes the tree view.
func (tv TreeView) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns the updated tree view.
func (tv TreeView) Update(msg tea.Msg) (TreeView, tea.Cmd) {
	if !tv.focused || tv.tree == nil {
		return tv, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			tv.tree.MoveUp()
		case "down", "j":
			tv.tree.MoveDown()
		case "left", "h":
			tv.tree.Collapse()
		case "right", "l":
			tv.tree.Expand()
		case "enter":
			// If has children and not expanded, expand first
			if node := tv.tree.Selected(); node != nil {
				if node.HasChildren() && !node.Expanded {
					tv.tree.Expand()
				} else {
					// Navigate to the selected item
					return tv, func() tea.Msg {
						return TreeNavigationMsg{
							ID:         node.ID,
							ObjectType: node.ObjectType,
						}
					}
				}
			}
		case " ":
			// Space toggles expand/collapse
			tv.tree.Toggle()
		}
	}

	return tv, nil
}

// View renders the tree view.
func (tv TreeView) View() string {
	var b strings.Builder

	// Title
	titleStr := tv.styles.Title.Render(tv.title)
	b.WriteString(titleStr)
	b.WriteString("\n")

	if tv.loading {
		b.WriteString(tv.styles.Node.Render("Loading..."))
		return tv.styles.Container.Width(tv.width).Height(tv.height).Render(b.String())
	}

	if tv.err != nil {
		b.WriteString(tv.styles.Node.Render("Error loading tree"))
		return tv.styles.Container.Width(tv.width).Height(tv.height).Render(b.String())
	}

	if tv.tree == nil || tv.tree.IsEmpty() {
		b.WriteString(tv.styles.Node.Render("No items"))
		return tv.styles.Container.Width(tv.width).Height(tv.height).Render(b.String())
	}

	// Calculate visible area (account for title and padding)
	visibleHeight := tv.height - 4
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Get visible nodes
	visible := tv.tree.Visible()
	selectedIdx := tv.tree.SelectedIndex()

	// Calculate scroll offset to keep selection visible
	startIdx := 0
	if selectedIdx >= visibleHeight {
		startIdx = selectedIdx - visibleHeight + 1
	}

	endIdx := startIdx + visibleHeight
	if endIdx > len(visible) {
		endIdx = len(visible)
	}

	// Render visible nodes
	for i := startIdx; i < endIdx; i++ {
		node := visible[i]
		line := tv.renderNode(node, i == selectedIdx)
		b.WriteString(line)
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	return tv.styles.Container.Width(tv.width).Height(tv.height).Render(b.String())
}

// renderNode renders a single tree node.
func (tv TreeView) renderNode(node *NavNode, selected bool) string {
	var b strings.Builder

	// Indentation
	indent := strings.Repeat("  ", node.Depth)
	b.WriteString(tv.styles.Indent.Render(indent))

	// Expand/collapse icon
	if node.HasChildren() {
		if node.Expanded {
			b.WriteString(tv.styles.ExpandIcon.Render("v "))
		} else {
			b.WriteString(tv.styles.ExpandIcon.Render("> "))
		}
	} else {
		b.WriteString("  ")
	}

	// Object icon
	if node.ObjectType == "database" {
		b.WriteString(tv.styles.DatabaseIcon.Render("# "))
	} else {
		b.WriteString(tv.styles.PageIcon.Render("- "))
	}

	// Title (truncate if too long)
	maxTitleWidth := tv.width - (node.Depth*2 + 6)
	if maxTitleWidth < 5 {
		maxTitleWidth = 5
	}
	title := node.Title
	if len(title) > maxTitleWidth {
		title = title[:maxTitleWidth-3] + "..."
	}

	// Apply style based on selection
	if selected && tv.focused {
		b.WriteString(tv.styles.SelectedNode.Render(title))
	} else {
		b.WriteString(tv.styles.Node.Render(title))
	}

	return b.String()
}

// SetTree updates the tree data.
func (tv *TreeView) SetTree(tree *NavTree) {
	tv.tree = tree
	tv.loading = false
	tv.err = nil
}

// SetLoading sets the loading state.
func (tv *TreeView) SetLoading(loading bool) {
	tv.loading = loading
}

// SetError sets an error state.
func (tv *TreeView) SetError(err error) {
	tv.err = err
	tv.loading = false
}

// SetFocused sets whether the tree view has focus.
func (tv *TreeView) SetFocused(focused bool) {
	tv.focused = focused
}

// IsFocused returns whether the tree view has focus.
func (tv TreeView) IsFocused() bool {
	return tv.focused
}

// SetSize updates the tree view dimensions.
func (tv *TreeView) SetSize(width, height int) {
	tv.width = width
	tv.height = height
}

// Tree returns the underlying NavTree.
func (tv TreeView) Tree() *NavTree {
	return tv.tree
}

// Selected returns the currently selected node.
func (tv TreeView) Selected() *NavNode {
	if tv.tree == nil {
		return nil
	}
	return tv.tree.Selected()
}

// SelectByID selects a node by its ID.
func (tv *TreeView) SelectByID(id string) bool {
	if tv.tree == nil {
		return false
	}
	return tv.tree.SelectByID(id)
}
