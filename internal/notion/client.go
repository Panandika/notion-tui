package notion

import (
	"context"
	"fmt"
	"time"

	"github.com/jomei/notionapi"
	"golang.org/x/time/rate"
)

// SearchInput contains parameters for workspace search.
type SearchInput struct {
	Query       string
	Filter      string // "page", "database", or "" for all
	PageSize    int
	StartCursor string
}

// SearchResult represents a unified search result.
type SearchResult struct {
	ID         string
	Title      string
	ObjectType string // "page" or "database"
	LastEdited time.Time
	ParentType string // "workspace", "database_id", "page_id"
	ParentID   string
}

// SearchResponse wraps the search results with pagination.
type SearchResponse struct {
	Results    []SearchResult
	HasMore    bool
	NextCursor string
}

// Client wraps the Notion API client with rate limiting support.
type Client struct {
	api     *notionapi.Client
	limiter *rate.Limiter
}

// NewClient creates a new rate-limited Notion API client.
// Rate limit: 2.5 requests/second with burst of 3.
func NewClient(token string) *Client {
	return &Client{
		api:     notionapi.NewClient(notionapi.Token(token), notionapi.WithRetry(5)),
		limiter: rate.NewLimiter(rate.Limit(2.5), 3),
	}
}

// GetPage retrieves a page from Notion by ID.
func (c *Client) GetPage(ctx context.Context, id string) (*notionapi.Page, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	page, err := c.api.Page.Get(ctx, notionapi.PageID(id))
	if err != nil {
		return nil, fmt.Errorf("get page %s: %w", id, err)
	}
	return page, nil
}

// QueryDatabase queries a Notion database with optional filters and sorting.
func (c *Client) QueryDatabase(ctx context.Context, id string,
	req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	resp, err := c.api.Database.Query(ctx, notionapi.DatabaseID(id), req)
	if err != nil {
		return nil, fmt.Errorf("query database %s: %w", id, err)
	}
	return resp, nil
}

// GetBlocks retrieves child blocks of a page or block.
func (c *Client) GetBlocks(ctx context.Context, id string,
	pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	blocks, err := c.api.Block.GetChildren(ctx, notionapi.BlockID(id), pagination)
	if err != nil {
		return nil, fmt.Errorf("get blocks for %s: %w", id, err)
	}
	return blocks, nil
}

// AppendBlocks appends blocks to a page or block.
func (c *Client) AppendBlocks(ctx context.Context, id string,
	req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	resp, err := c.api.Block.AppendChildren(ctx, notionapi.BlockID(id), req)
	if err != nil {
		return nil, fmt.Errorf("append blocks to %s: %w", id, err)
	}
	return resp, nil
}

// UpdatePage updates a page's properties.
func (c *Client) UpdatePage(ctx context.Context, id string,
	req *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	page, err := c.api.Page.Update(ctx, notionapi.PageID(id), req)
	if err != nil {
		return nil, fmt.Errorf("update page %s: %w", id, err)
	}
	return page, nil
}

// DeleteBlock archives a block (Notion API soft-deletes via archive).
func (c *Client) DeleteBlock(ctx context.Context, id string) (notionapi.Block, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	block, err := c.api.Block.Delete(ctx, notionapi.BlockID(id))
	if err != nil {
		return nil, fmt.Errorf("delete block %s: %w", id, err)
	}
	return block, nil
}

// GetBlock retrieves a single block by ID.
func (c *Client) GetBlock(ctx context.Context, id string) (notionapi.Block, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	block, err := c.api.Block.Get(ctx, notionapi.BlockID(id))
	if err != nil {
		return nil, fmt.Errorf("get block %s: %w", id, err)
	}
	return block, nil
}

// UpdateBlock updates a block's properties.
func (c *Client) UpdateBlock(ctx context.Context, id string,
	req *notionapi.BlockUpdateRequest) (notionapi.Block, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	block, err := c.api.Block.Update(ctx, notionapi.BlockID(id), req)
	if err != nil {
		return nil, fmt.Errorf("update block %s: %w", id, err)
	}
	return block, nil
}

// Search performs a workspace-wide search using Notion Search API.
// When no filter is specified, it searches for both pages and databases
// by making separate API calls (due to library serialization constraints).
func (c *Client) Search(ctx context.Context, input SearchInput) (*SearchResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}

	// If a specific filter is provided, use it directly
	if input.Filter == "page" || input.Filter == "database" {
		return c.searchWithFilter(ctx, input, input.Filter)
	}

	// No filter specified - search for pages first
	// Note: The notionapi library's SearchFilter doesn't properly omit empty values,
	// so we must always specify a filter to avoid API validation errors.
	pageResp, err := c.searchWithFilter(ctx, input, "page")
	if err != nil {
		return nil, err
	}

	// Also search for databases
	dbInput := input
	dbInput.PageSize = input.PageSize / 2 // Split the page size
	if dbInput.PageSize < 10 {
		dbInput.PageSize = 10
	}
	dbResp, err := c.searchWithFilter(ctx, dbInput, "database")
	if err != nil {
		// If database search fails, still return page results
		return pageResp, nil
	}

	// Merge results: pages first, then databases
	combined := &SearchResponse{
		Results:    append(pageResp.Results, dbResp.Results...),
		HasMore:    pageResp.HasMore || dbResp.HasMore,
		NextCursor: pageResp.NextCursor, // Use page cursor for pagination
	}

	return combined, nil
}

// searchWithFilter performs a search with a specific object type filter.
func (c *Client) searchWithFilter(ctx context.Context, input SearchInput, filter string) (*SearchResponse, error) {
	req := &notionapi.SearchRequest{
		Query:    input.Query,
		PageSize: input.PageSize,
		Filter: notionapi.SearchFilter{
			Value:    filter,
			Property: "object",
		},
	}

	if input.StartCursor != "" {
		req.StartCursor = notionapi.Cursor(input.StartCursor)
	}

	resp, err := c.api.Search.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("search workspace: %w", err)
	}

	return c.convertSearchResponse(resp), nil
}

// convertSearchResponse transforms notionapi.SearchResponse to internal format.
func (c *Client) convertSearchResponse(resp *notionapi.SearchResponse) *SearchResponse {
	results := make([]SearchResult, 0, len(resp.Results))

	for _, obj := range resp.Results {
		switch v := obj.(type) {
		case *notionapi.Page:
			results = append(results, SearchResult{
				ID:         string(v.ID),
				Title:      extractPageTitle(v),
				ObjectType: "page",
				LastEdited: v.LastEditedTime,
				ParentType: getParentType(v.Parent),
				ParentID:   getParentID(v.Parent),
			})
		case *notionapi.Database:
			results = append(results, SearchResult{
				ID:         string(v.ID),
				Title:      extractDatabaseTitle(v),
				ObjectType: "database",
				LastEdited: v.LastEditedTime,
				ParentType: getDatabaseParentType(v.Parent),
				ParentID:   getDatabaseParentID(v.Parent),
			})
		}
	}

	return &SearchResponse{
		Results:    results,
		HasMore:    resp.HasMore,
		NextCursor: string(resp.NextCursor),
	}
}

// extractPageTitle gets the title from a page's properties.
func extractPageTitle(page *notionapi.Page) string {
	for _, prop := range page.Properties {
		if titleProp, ok := prop.(*notionapi.TitleProperty); ok {
			if len(titleProp.Title) > 0 {
				return titleProp.Title[0].PlainText
			}
		}
	}
	return "Untitled"
}

// extractDatabaseTitle gets the title from a database.
func extractDatabaseTitle(db *notionapi.Database) string {
	if len(db.Title) > 0 {
		return db.Title[0].PlainText
	}
	return "Untitled Database"
}

// getParentType returns the parent type for a page.
func getParentType(parent notionapi.Parent) string {
	switch parent.Type {
	case notionapi.ParentTypeWorkspace:
		return "workspace"
	case notionapi.ParentTypeDatabaseID:
		return "database_id"
	case notionapi.ParentTypePageID:
		return "page_id"
	case notionapi.ParentTypeBlockID:
		return "block_id"
	default:
		return "unknown"
	}
}

// getParentID returns the parent ID for a page.
func getParentID(parent notionapi.Parent) string {
	switch parent.Type {
	case notionapi.ParentTypeDatabaseID:
		return string(parent.DatabaseID)
	case notionapi.ParentTypePageID:
		return string(parent.PageID)
	case notionapi.ParentTypeBlockID:
		return string(parent.BlockID)
	default:
		return ""
	}
}

// getDatabaseParentType returns the parent type for a database.
func getDatabaseParentType(parent notionapi.Parent) string {
	return getParentType(parent)
}

// getDatabaseParentID returns the parent ID for a database.
func getDatabaseParentID(parent notionapi.Parent) string {
	return getParentID(parent)
}
