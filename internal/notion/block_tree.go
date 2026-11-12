package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
)

// DefaultMaxBlockDepth bounds how deep nested blocks are fetched below a page.
// Each level costs one API call per parent block, so the limit keeps page loads bounded.
const DefaultMaxBlockDepth = 4

// blockPageSize is the Notion API maximum for block children.
const blockPageSize = 100

// BlockFetcher retrieves one page of child blocks for a page or block.
type BlockFetcher interface {
	GetBlocks(ctx context.Context, id string, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error)
}

// FetchBlockTree fetches every child block of rootID, following pagination, and
// attaches nested children to their parent blocks up to maxDepth levels.
// Child pages and child databases are not descended into: they are separate documents.
func FetchBlockTree(ctx context.Context, fetcher BlockFetcher, rootID string, maxDepth int) ([]notionapi.Block, error) {
	blocks, err := fetchAllChildren(ctx, fetcher, rootID)
	if err != nil {
		return nil, err
	}
	if maxDepth <= 1 {
		return blocks, nil
	}

	for _, block := range blocks {
		if !descendable(block) {
			continue
		}
		children, err := FetchBlockTree(ctx, fetcher, string(block.GetID()), maxDepth-1)
		if err != nil {
			return nil, err
		}
		SetBlockChildren(block, children)
	}
	return blocks, nil
}

// fetchAllChildren reads all pages of children for a single parent.
func fetchAllChildren(ctx context.Context, fetcher BlockFetcher, id string) ([]notionapi.Block, error) {
	var all []notionapi.Block
	var cursor notionapi.Cursor

	for {
		resp, err := fetcher.GetBlocks(ctx, id, &notionapi.Pagination{
			StartCursor: cursor,
			PageSize:    blockPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch children of %s: %w", id, err)
		}
		if resp == nil {
			return all, nil
		}
		all = append(all, resp.Results...)

		// Stop on the last page, or if the API reports more without a usable cursor.
		if !resp.HasMore || resp.NextCursor == "" || notionapi.Cursor(resp.NextCursor) == cursor {
			return all, nil
		}
		cursor = notionapi.Cursor(resp.NextCursor)
	}
}

// descendable reports whether a block's children belong to the same document.
func descendable(block notionapi.Block) bool {
	if block == nil || !block.GetHasChildren() || block.GetID() == "" {
		return false
	}
	switch block.GetType() {
	case notionapi.BlockTypeChildPage, notionapi.BlockTypeChildDatabase:
		return false
	default:
		return true
	}
}

// BlockChildren returns the nested children attached to a block, if any.
func BlockChildren(block notionapi.Block) []notionapi.Block {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return b.Paragraph.Children
	case *notionapi.Heading1Block:
		return b.Heading1.Children
	case *notionapi.Heading2Block:
		return b.Heading2.Children
	case *notionapi.Heading3Block:
		return b.Heading3.Children
	case *notionapi.CalloutBlock:
		return b.Callout.Children
	case *notionapi.QuoteBlock:
		return b.Quote.Children
	case *notionapi.BulletedListItemBlock:
		return b.BulletedListItem.Children
	case *notionapi.NumberedListItemBlock:
		return b.NumberedListItem.Children
	case *notionapi.ToDoBlock:
		return b.ToDo.Children
	case *notionapi.ToggleBlock:
		return b.Toggle.Children
	case *notionapi.TableBlock:
		return b.Table.Children
	case *notionapi.ColumnListBlock:
		return b.ColumnList.Children
	case *notionapi.ColumnBlock:
		return b.Column.Children
	case *notionapi.SyncedBlock:
		return b.SyncedBlock.Children
	case *notionapi.TemplateBlock:
		return b.Template.Children
	default:
		return nil
	}
}

// SetBlockChildren attaches nested children to a block.
// It returns false when the block type cannot hold children.
func SetBlockChildren(block notionapi.Block, children []notionapi.Block) bool {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		b.Paragraph.Children = children
	case *notionapi.Heading1Block:
		b.Heading1.Children = children
	case *notionapi.Heading2Block:
		b.Heading2.Children = children
	case *notionapi.Heading3Block:
		b.Heading3.Children = children
	case *notionapi.CalloutBlock:
		b.Callout.Children = children
	case *notionapi.QuoteBlock:
		b.Quote.Children = children
	case *notionapi.BulletedListItemBlock:
		b.BulletedListItem.Children = children
	case *notionapi.NumberedListItemBlock:
		b.NumberedListItem.Children = children
	case *notionapi.ToDoBlock:
		b.ToDo.Children = children
	case *notionapi.ToggleBlock:
		b.Toggle.Children = children
	case *notionapi.TableBlock:
		b.Table.Children = children
	case *notionapi.ColumnListBlock:
		b.ColumnList.Children = children
	case *notionapi.ColumnBlock:
		b.Column.Children = children
	case *notionapi.SyncedBlock:
		b.SyncedBlock.Children = children
	case *notionapi.TemplateBlock:
		b.Template.Children = children
	default:
		return false
	}
	return true
}
