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
	CreatePageMarkdown(ctx context.Context, parent notion.PageParent, markdown string) (*notion.CreatedPage, error)

	// Comments on a page
	ListComments(ctx context.Context, pageID string) ([]notion.Comment, error)
	AddComment(ctx context.Context, pageID, threadID, text string) error

	// Page properties
	GetDatabase(ctx context.Context, id string) (*notionapi.Database, error)
	SetPageProperty(ctx context.Context, pageID string, update notion.PropertyUpdate) error

	// Database operations
	QueryDatabase(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error)

	// Views saved on a database
	ListViews(ctx context.Context, databaseID string) ([]notion.View, error)
	QueryView(ctx context.Context, viewID string) (*notion.ViewRows, error)
	ViewRowsAfter(ctx context.Context, viewID, queryID, cursor string) (*notion.ViewRows, error)

	// Search operations
	Search(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error)
}
