package pages

import (
	"context"

	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/notion"
)

// NotionClient defines the interface for Notion API operations needed by pages.
// This consolidates all methods required across ListPage, DetailPage, and EditPage.
type NotionClient interface {
	// Page operations
	GetPage(ctx context.Context, id string) (*notionapi.Page, error)
	UpdatePage(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error)

	// Page content as Markdown: one call per page, nested content included
	GetPageMarkdown(ctx context.Context, id string) (*notion.PageMarkdown, error)
	UpdatePageMarkdown(ctx context.Context, id string, updates []notion.ContentUpdate) (*notion.PageMarkdown, error)
	ReplacePageMarkdown(ctx context.Context, id, markdown string) (*notion.PageMarkdown, error)
	CreatePageMarkdown(ctx context.Context, parentPageID, markdown string) (*notion.CreatedPage, error)

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
