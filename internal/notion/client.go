package notion

import (
	"context"
	"fmt"

	"github.com/jomei/notionapi"
	"golang.org/x/time/rate"
)

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
