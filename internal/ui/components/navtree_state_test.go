package components

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/notion"
)

func TestNavTree_ReloadKeepsOpenNodesAndSelection(t *testing.T) {
	t.Parallel()

	results := []notion.SearchResult{
		{ID: "a", Title: "Alpha", ObjectType: "page", ParentType: "workspace"},
		{ID: "a1", Title: "Alpha child", ObjectType: "page", ParentType: "page_id", ParentID: "a"},
		{ID: "b", Title: "Beta", ObjectType: "page", ParentType: "workspace"},
		{ID: "b1", Title: "Beta child", ObjectType: "page", ParentType: "page_id", ParentID: "b"},
	}
	visible := func(tree *NavTree) []string {
		var ids []string
		for _, node := range tree.Visible() {
			ids = append(ids, node.ID)
		}
		return ids
	}

	view := NewTreeView(NewTreeViewInput{Width: 30, Height: 20})
	view.SetTree(BuildNavTree(BuildNavTreeInput{Results: results}))
	old := view.Tree()
	old.CollapseAll()
	require.True(t, old.SelectByID("b"))
	old.Expand()
	require.True(t, old.SelectByID("b1"))
	require.Equal(t, []string{"a", "b", "b1"}, visible(old))

	// The same workspace again, with one page added
	results = append(results, notion.SearchResult{ID: "b2", Title: "Beta second", ObjectType: "page", ParentType: "page_id", ParentID: "b"})
	view.SetTree(BuildNavTree(BuildNavTreeInput{Results: results}))

	assert.Equal(t, []string{"a", "b", "b1", "b2"}, visible(view.Tree()), "the open node stays open and shows its new child")
	assert.Equal(t, "b1", view.Selected().ID, "the selection stays on the same item")

	// The selected item is gone after a reload: fall back to the top
	view.SetTree(BuildNavTree(BuildNavTreeInput{Results: results[:1]}))
	assert.Equal(t, "a", view.Selected().ID)
}
