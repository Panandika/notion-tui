package components

import (
	"sort"

	"github.com/Panandika/notion-tui/internal/notion"
)

// NavNode represents a node in the navigation tree.
type NavNode struct {
	ID         string
	Title      string
	ObjectType string // "database" or "page"
	ParentID   string
	ParentType string // "workspace", "database_id", "page_id"
	Children   []*NavNode
	Expanded   bool
	Depth      int
}

// HasChildren returns true if the node has children.
func (n *NavNode) HasChildren() bool {
	return len(n.Children) > 0
}

// NavTree manages the hierarchical navigation structure.
type NavTree struct {
	roots       []*NavNode
	nodeMap     map[string]*NavNode
	selectedIdx int
	visible     []*NavNode // Cached flattened visible nodes
}

// NewNavTree creates a new empty navigation tree.
func NewNavTree() *NavTree {
	return &NavTree{
		roots:       make([]*NavNode, 0),
		nodeMap:     make(map[string]*NavNode),
		selectedIdx: 0,
		visible:     make([]*NavNode, 0),
	}
}

// BuildNavTreeInput contains parameters for building a NavTree.
type BuildNavTreeInput struct {
	Results []notion.SearchResult
}

// BuildNavTree constructs a navigation tree from flat search results.
func BuildNavTree(input BuildNavTreeInput) *NavTree {
	tree := NewNavTree()

	// First pass: create all nodes
	for _, r := range input.Results {
		node := &NavNode{
			ID:         r.ID,
			Title:      r.Title,
			ObjectType: r.ObjectType,
			ParentID:   r.ParentID,
			ParentType: r.ParentType,
			Children:   make([]*NavNode, 0),
			Expanded:   false,
			Depth:      0,
		}
		tree.nodeMap[r.ID] = node
	}

	// Second pass: build parent-child relationships
	for _, node := range tree.nodeMap {
		if node.ParentType == "workspace" || node.ParentID == "" {
			// Root-level item
			node.Depth = 0
			tree.roots = append(tree.roots, node)
		} else {
			// Find parent and add as child
			if parent, ok := tree.nodeMap[node.ParentID]; ok {
				node.Depth = parent.Depth + 1
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not in tree (possibly not fetched), treat as root
				node.Depth = 0
				tree.roots = append(tree.roots, node)
			}
		}
	}

	// Sort roots and children alphabetically, databases first
	tree.sortNodes(tree.roots)
	for _, node := range tree.nodeMap {
		if len(node.Children) > 0 {
			tree.sortNodes(node.Children)
		}
	}

	// Expand root databases by default for better UX
	for _, root := range tree.roots {
		if root.ObjectType == "database" {
			root.Expanded = true
		}
	}

	// Build initial visible list
	tree.rebuildVisible()

	return tree
}

// sortNodes sorts nodes: databases first, then alphabetically by title.
func (t *NavTree) sortNodes(nodes []*NavNode) {
	sort.Slice(nodes, func(i, j int) bool {
		// Databases come before pages
		if nodes[i].ObjectType != nodes[j].ObjectType {
			return nodes[i].ObjectType == "database"
		}
		// Alphabetical by title
		return nodes[i].Title < nodes[j].Title
	})
}

// rebuildVisible rebuilds the flattened visible nodes list.
func (t *NavTree) rebuildVisible() {
	t.visible = make([]*NavNode, 0)
	for _, root := range t.roots {
		t.flattenNode(root)
	}
}

// flattenNode recursively adds visible nodes to the visible list.
func (t *NavTree) flattenNode(node *NavNode) {
	t.visible = append(t.visible, node)
	if node.Expanded {
		for _, child := range node.Children {
			t.flattenNode(child)
		}
	}
}

// Visible returns the list of currently visible nodes.
func (t *NavTree) Visible() []*NavNode {
	return t.visible
}

// SelectedIndex returns the current selection index.
func (t *NavTree) SelectedIndex() int {
	return t.selectedIdx
}

// Selected returns the currently selected node, or nil if none.
func (t *NavTree) Selected() *NavNode {
	if t.selectedIdx >= 0 && t.selectedIdx < len(t.visible) {
		return t.visible[t.selectedIdx]
	}
	return nil
}

// MoveUp moves selection up by one.
func (t *NavTree) MoveUp() {
	if t.selectedIdx > 0 {
		t.selectedIdx--
	}
}

// MoveDown moves selection down by one.
func (t *NavTree) MoveDown() {
	if t.selectedIdx < len(t.visible)-1 {
		t.selectedIdx++
	}
}

// Toggle expands or collapses the selected node.
func (t *NavTree) Toggle() {
	if node := t.Selected(); node != nil && node.HasChildren() {
		node.Expanded = !node.Expanded
		t.rebuildVisible()
	}
}

// Expand expands the selected node if it has children.
func (t *NavTree) Expand() bool {
	if node := t.Selected(); node != nil && node.HasChildren() && !node.Expanded {
		node.Expanded = true
		t.rebuildVisible()
		return true
	}
	return false
}

// Collapse collapses the selected node, or moves to parent if already collapsed.
func (t *NavTree) Collapse() bool {
	node := t.Selected()
	if node == nil {
		return false
	}

	// If expanded, collapse
	if node.Expanded && node.HasChildren() {
		node.Expanded = false
		t.rebuildVisible()
		return true
	}

	// If collapsed or no children, move to parent
	if node.ParentID != "" {
		for i, n := range t.visible {
			if n.ID == node.ParentID {
				t.selectedIdx = i
				return true
			}
		}
	}

	return false
}

// SelectByID selects the node with the given ID.
func (t *NavTree) SelectByID(id string) bool {
	for i, node := range t.visible {
		if node.ID == id {
			t.selectedIdx = i
			return true
		}
	}
	return false
}

// Roots returns the root nodes.
func (t *NavTree) Roots() []*NavNode {
	return t.roots
}

// NodeCount returns the total number of nodes in the tree.
func (t *NavTree) NodeCount() int {
	return len(t.nodeMap)
}

// VisibleCount returns the number of currently visible nodes.
func (t *NavTree) VisibleCount() int {
	return len(t.visible)
}

// IsEmpty returns true if the tree has no nodes.
func (t *NavTree) IsEmpty() bool {
	return len(t.roots) == 0
}

// ExpandAll expands all nodes in the tree.
func (t *NavTree) ExpandAll() {
	for _, node := range t.nodeMap {
		if node.HasChildren() {
			node.Expanded = true
		}
	}
	t.rebuildVisible()
}

// CollapseAll collapses all nodes in the tree.
func (t *NavTree) CollapseAll() {
	for _, node := range t.nodeMap {
		node.Expanded = false
	}
	t.rebuildVisible()
	t.selectedIdx = 0
}
