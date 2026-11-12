package notion

import (
	"context"
	"errors"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func txt(s string) []notionapi.RichText {
	return []notionapi.RichText{{PlainText: s}}
}

func basicBlock(id string, t notionapi.BlockType, hasChildren bool) notionapi.BasicBlock {
	return notionapi.BasicBlock{
		Object:      notionapi.ObjectTypeBlock,
		ID:          notionapi.BlockID(id),
		Type:        t,
		HasChildren: hasChildren,
	}
}

func para(id, s string) *notionapi.ParagraphBlock {
	return &notionapi.ParagraphBlock{
		BasicBlock: basicBlock(id, notionapi.BlockTypeParagraph, false),
		Paragraph:  notionapi.Paragraph{RichText: txt(s)},
	}
}

func bullet(id, s string, children ...notionapi.Block) *notionapi.BulletedListItemBlock {
	return &notionapi.BulletedListItemBlock{
		BasicBlock:       basicBlock(id, notionapi.BlockTypeBulletedListItem, len(children) > 0),
		BulletedListItem: notionapi.ListItem{RichText: txt(s), Children: children},
	}
}

// fakeFetcher serves canned pages of children keyed by parent ID and cursor.
type fakeFetcher struct {
	pages map[string]*notionapi.GetChildrenResponse
	calls []string
	err   error
}

func (f *fakeFetcher) GetBlocks(_ context.Context, id string, p *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	key := id
	if p != nil && p.StartCursor != "" {
		key = id + "@" + string(p.StartCursor)
	}
	f.calls = append(f.calls, key)
	if f.err != nil {
		return nil, f.err
	}
	if resp, ok := f.pages[key]; ok {
		return resp, nil
	}
	return &notionapi.GetChildrenResponse{}, nil
}

func TestFetchBlockTree_FollowsPagination(t *testing.T) {
	t.Parallel()

	f := &fakeFetcher{pages: map[string]*notionapi.GetChildrenResponse{
		"root":    {Results: notionapi.Blocks{para("a", "one")}, HasMore: true, NextCursor: "c1"},
		"root@c1": {Results: notionapi.Blocks{para("b", "two")}},
	}}

	blocks, err := FetchBlockTree(context.Background(), f, "root", DefaultMaxBlockDepth)

	require.NoError(t, err)
	assert.Len(t, blocks, 2)
	assert.Equal(t, []string{"root", "root@c1"}, f.calls)
}

func TestFetchBlockTree_AttachesNestedChildren(t *testing.T) {
	t.Parallel()

	toggle := &notionapi.ToggleBlock{
		BasicBlock: basicBlock("tog", notionapi.BlockTypeToggle, true),
		Toggle:     notionapi.Toggle{RichText: txt("More")},
	}
	childPage := &notionapi.ChildPageBlock{BasicBlock: basicBlock("sub", notionapi.BlockTypeChildPage, true)}
	f := &fakeFetcher{pages: map[string]*notionapi.GetChildrenResponse{
		"root": {Results: notionapi.Blocks{toggle, childPage}},
		"tog":  {Results: notionapi.Blocks{para("inner", "hidden text")}},
	}}

	blocks, err := FetchBlockTree(context.Background(), f, "root", DefaultMaxBlockDepth)

	require.NoError(t, err)
	require.Len(t, blocks, 2)
	assert.Len(t, BlockChildren(blocks[0]), 1)
	// Child pages are separate documents and must not be fetched.
	assert.Equal(t, []string{"root", "tog"}, f.calls)
}

func TestFetchBlockTree_RespectsMaxDepth(t *testing.T) {
	t.Parallel()

	f := &fakeFetcher{pages: map[string]*notionapi.GetChildrenResponse{
		"root": {Results: notionapi.Blocks{&notionapi.ToggleBlock{BasicBlock: basicBlock("l1", notionapi.BlockTypeToggle, true)}}},
		"l1":   {Results: notionapi.Blocks{&notionapi.ToggleBlock{BasicBlock: basicBlock("l2", notionapi.BlockTypeToggle, true)}}},
		"l2":   {Results: notionapi.Blocks{para("deep", "too deep")}},
	}}

	_, err := FetchBlockTree(context.Background(), f, "root", 2)

	require.NoError(t, err)
	assert.Equal(t, []string{"root", "l1"}, f.calls)
}

func TestFetchBlockTree_PropagatesError(t *testing.T) {
	t.Parallel()

	f := &fakeFetcher{err: errors.New("boom")}

	_, err := FetchBlockTree(context.Background(), f, "root", DefaultMaxBlockDepth)

	assert.ErrorContains(t, err, "boom")
}

func TestSetBlockChildren_UnsupportedType(t *testing.T) {
	t.Parallel()

	divider := &notionapi.DividerBlock{BasicBlock: basicBlock("d", notionapi.BlockTypeDivider, false)}

	assert.False(t, SetBlockChildren(divider, []notionapi.Block{para("x", "y")}))
	assert.Nil(t, BlockChildren(divider))
}

func TestConvertNestedBlocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		blocks   []notionapi.Block
		expected string
	}{
		{
			name: "toggle shows its children",
			blocks: []notionapi.Block{&notionapi.ToggleBlock{
				BasicBlock: basicBlock("t", notionapi.BlockTypeToggle, true),
				Toggle:     notionapi.Toggle{RichText: txt("Details"), Children: notionapi.Blocks{para("p", "hidden text")}},
			}},
			expected: "▸ Details\n\nhidden text",
		},
		{
			name: "column list flattens columns in order",
			blocks: []notionapi.Block{&notionapi.ColumnListBlock{
				BasicBlock: basicBlock("cl", notionapi.BlockTypeColumnList, true),
				ColumnList: notionapi.ColumnList{Children: notionapi.Blocks{
					&notionapi.ColumnBlock{Column: notionapi.Column{Children: notionapi.Blocks{para("l", "left")}}},
					&notionapi.ColumnBlock{Column: notionapi.Column{Children: notionapi.Blocks{para("r", "right")}}},
				}},
			}},
			expected: "left\n\nright",
		},
		{
			name:     "nested list items are indented",
			blocks:   []notionapi.Block{bullet("a", "parent", bullet("b", "child")), bullet("c", "sibling")},
			expected: "- parent\n    - child\n- sibling",
		},
		{
			name: "paragraph after list is separated by a blank line",
			blocks: []notionapi.Block{
				bullet("a", "item"),
				para("p", "text"),
			},
			expected: "- item\n\ntext",
		},
		{
			name: "quote children stay inside the quote",
			blocks: []notionapi.Block{&notionapi.QuoteBlock{
				BasicBlock: basicBlock("q", notionapi.BlockTypeQuote, true),
				Quote:      notionapi.Quote{RichText: txt("said"), Children: notionapi.Blocks{para("p", "more")}},
			}},
			expected: "> said\n>\n> more",
		},
		{
			name: "table renders rows with header separator",
			blocks: []notionapi.Block{&notionapi.TableBlock{
				BasicBlock: basicBlock("tb", notionapi.BlockTypeTableBlock, true),
				Table: notionapi.Table{TableWidth: 2, Children: notionapi.Blocks{
					&notionapi.TableRowBlock{TableRow: notionapi.TableRow{Cells: [][]notionapi.RichText{txt("Name"), txt("Qty")}}},
					&notionapi.TableRowBlock{TableRow: notionapi.TableRow{Cells: [][]notionapi.RichText{txt("a|b"), txt("2")}}},
				}},
			}},
			expected: "| Name | Qty |\n| --- | --- |\n| a\\|b | 2 |",
		},
		{
			name: "child page and database show their titles",
			blocks: []notionapi.Block{
				func() notionapi.Block {
					b := &notionapi.ChildPageBlock{BasicBlock: basicBlock("cp", notionapi.BlockTypeChildPage, false)}
					b.ChildPage.Title = "Notes"
					return b
				}(),
				func() notionapi.Block {
					b := &notionapi.ChildDatabaseBlock{BasicBlock: basicBlock("cd", notionapi.BlockTypeChildDatabase, false)}
					b.ChildDatabase.Title = "Tasks"
					return b
				}(),
			},
			expected: "📄 **Notes**\n\n🗂 **Tasks** (database)",
		},
		{
			name:     "unsupported block shows a placeholder",
			blocks:   []notionapi.Block{&notionapi.UnsupportedBlock{BasicBlock: basicBlock("u", notionapi.BlockTypeUnsupported, false)}},
			expected: "*[unsupported block]*",
		},
		{
			name: "embed uses caption as label",
			blocks: []notionapi.Block{&notionapi.EmbedBlock{
				BasicBlock: basicBlock("e", notionapi.BlockTypeEmbed, false),
				Embed:      notionapi.Embed{URL: "https://example.com", Caption: txt("Demo")},
			}},
			expected: "[Demo](https://example.com)",
		},
		{
			name: "numbered list restarts after a break",
			blocks: []notionapi.Block{
				&notionapi.NumberedListItemBlock{BasicBlock: basicBlock("n1", notionapi.BlockTypeNumberedListItem, false), NumberedListItem: notionapi.ListItem{RichText: txt("one")}},
				&notionapi.NumberedListItemBlock{BasicBlock: basicBlock("n2", notionapi.BlockTypeNumberedListItem, false), NumberedListItem: notionapi.ListItem{RichText: txt("two")}},
				para("p", "break"),
				&notionapi.NumberedListItemBlock{BasicBlock: basicBlock("n3", notionapi.BlockTypeNumberedListItem, false), NumberedListItem: notionapi.ListItem{RichText: txt("again")}},
			},
			expected: "1. one\n2. two\n\nbreak\n\n1. again",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := ConvertBlocksToMarkdown(tt.blocks)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertAnnotations_KeepsWhitespaceOutsideMarkers(t *testing.T) {
	t.Parallel()

	bold := &notionapi.Annotations{Bold: true}
	tests := []struct {
		name     string
		text     notionapi.RichText
		expected string
	}{
		{name: "trailing space", text: notionapi.RichText{PlainText: "type ", Annotations: bold}, expected: "**type** "},
		{name: "leading space", text: notionapi.RichText{PlainText: " Marketplace", Annotations: bold}, expected: " **Marketplace**"},
		{name: "whitespace only", text: notionapi.RichText{PlainText: "  ", Annotations: bold}, expected: "  "},
		{name: "linked text", text: notionapi.RichText{PlainText: "guides ", Href: "https://example.com"}, expected: "[guides](https://example.com) "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, convertAnnotations(tt.text))
		})
	}
}

func TestConvertAnnotations_InlineCodeKeepsItsSpaces(t *testing.T) {
	t.Parallel()

	code := notionapi.RichText{PlainText: " x ", Annotations: &notionapi.Annotations{Code: true}}

	assert.Equal(t, "` x `", convertAnnotations(code))
}

func TestConvertChildPage_EmptyTitle(t *testing.T) {
	t.Parallel()

	block := &notionapi.ChildPageBlock{BasicBlock: basicBlock("cp", notionapi.BlockTypeChildPage, false)}

	result, err := ConvertBlocksToMarkdown([]notionapi.Block{block})

	require.NoError(t, err)
	assert.Equal(t, "📄 **Untitled**", result)
}
