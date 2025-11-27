package pages

import (
	"context"

	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/jomei/notionapi"
)

// NotionClient defines the interface for Notion API operations needed by pages.
// This consolidates all methods required across ListPage, DetailPage, and EditPage.
type NotionClient interface {
	// Page operations
	GetPage(ctx context.Context, id string) (*notionapi.Page, error)
	UpdatePage(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error)

	// Block operations
	GetBlocks(ctx context.Context, id string, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error)
	AppendBlocks(ctx context.Context, id string, req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error)
	DeleteBlock(ctx context.Context, id string) (notionapi.Block, error)
	GetBlock(ctx context.Context, id string) (notionapi.Block, error)
	UpdateBlock(ctx context.Context, id string, req *notionapi.BlockUpdateRequest) (notionapi.Block, error)

	// Database operations
	QueryDatabase(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error)

	// Search operations
	Search(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error)
}
