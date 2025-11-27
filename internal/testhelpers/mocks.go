package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/jomei/notionapi"
)

// MockNotionClient provides a mock implementation for testing Notion API interactions.
// It implements the common interface pattern used in the notion client.
type MockNotionClient struct {
	mu sync.Mutex

	// Configurable return values for each method
	GetPageFunc       func(ctx context.Context, id string) (*notionapi.Page, error)
	QueryDatabaseFunc func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error)
	GetBlocksFunc     func(ctx context.Context, id string, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error)
	GetBlockFunc      func(ctx context.Context, id string) (notionapi.Block, error)
	UpdatePageFunc    func(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error)
	UpdateBlockFunc   func(ctx context.Context, id string, req *notionapi.BlockUpdateRequest) (notionapi.Block, error)
	AppendBlocksFunc  func(ctx context.Context, id string, req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error)
	DeleteBlockFunc   func(ctx context.Context, id string) (notionapi.Block, error)
	SearchFunc        func(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error)

	// Call tracking for assertions
	GetPageCalls       []GetPageCall
	QueryDatabaseCalls []QueryDatabaseCall
	GetBlocksCalls     []GetBlocksCall
	GetBlockCalls      []GetBlockCall
	UpdatePageCalls    []UpdatePageCall
	UpdateBlockCalls   []UpdateBlockCall
	AppendBlocksCalls  []AppendBlocksCall
	DeleteBlockCalls   []DeleteBlockCall
	SearchCalls        []SearchCall

	// Simple return values for common scenarios
	PageToReturn         *notionapi.Page
	DatabaseToReturn     *notionapi.DatabaseQueryResponse
	BlocksToReturn       *notionapi.GetChildrenResponse
	BlockToReturn        notionapi.Block
	AppendResponseReturn *notionapi.AppendBlockChildrenResponse
	DeletedBlockReturn   notionapi.Block
	SearchResponseReturn *notion.SearchResponse
	ErrorToReturn        error
}

// GetPageCall records a call to GetPage.
type GetPageCall struct {
	Ctx context.Context
	ID  string
}

// QueryDatabaseCall records a call to QueryDatabase.
type QueryDatabaseCall struct {
	Ctx     context.Context
	ID      string
	Request *notionapi.DatabaseQueryRequest
}

// GetBlocksCall records a call to GetBlocks.
type GetBlocksCall struct {
	Ctx        context.Context
	ID         string
	Pagination *notionapi.Pagination
}

// GetBlockCall records a call to GetBlock.
type GetBlockCall struct {
	Ctx context.Context
	ID  string
}

// UpdatePageCall records a call to UpdatePage.
type UpdatePageCall struct {
	Ctx     context.Context
	ID      string
	Request *notionapi.PageUpdateRequest
}

// UpdateBlockCall records a call to UpdateBlock.
type UpdateBlockCall struct {
	Ctx     context.Context
	ID      string
	Request *notionapi.BlockUpdateRequest
}

// AppendBlocksCall records a call to AppendBlocks.
type AppendBlocksCall struct {
	Ctx     context.Context
	ID      string
	Request *notionapi.AppendBlockChildrenRequest
}

// DeleteBlockCall records a call to DeleteBlock.
type DeleteBlockCall struct {
	Ctx context.Context
	ID  string
}

// SearchCall records a call to Search.
type SearchCall struct {
	Ctx   context.Context
	Input notion.SearchInput
}

// NewMockNotionClient creates a new MockNotionClient with default no-op behavior.
func NewMockNotionClient() *MockNotionClient {
	return &MockNotionClient{
		GetPageCalls:       make([]GetPageCall, 0),
		QueryDatabaseCalls: make([]QueryDatabaseCall, 0),
		GetBlocksCalls:     make([]GetBlocksCall, 0),
		GetBlockCalls:      make([]GetBlockCall, 0),
		UpdatePageCalls:    make([]UpdatePageCall, 0),
		UpdateBlockCalls:   make([]UpdateBlockCall, 0),
		AppendBlocksCalls:  make([]AppendBlocksCall, 0),
		DeleteBlockCalls:   make([]DeleteBlockCall, 0),
		SearchCalls:        make([]SearchCall, 0),
	}
}

// GetPage retrieves a page by ID. Returns configured values or error.
func (m *MockNotionClient) GetPage(ctx context.Context, id string) (*notionapi.Page, error) {
	m.mu.Lock()
	m.GetPageCalls = append(m.GetPageCalls, GetPageCall{Ctx: ctx, ID: id})
	m.mu.Unlock()

	if m.GetPageFunc != nil {
		return m.GetPageFunc(ctx, id)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.PageToReturn != nil {
		return m.PageToReturn, nil
	}

	return NewTestPage(id, "Test Page"), nil
}

// QueryDatabase queries a database. Returns configured values or error.
func (m *MockNotionClient) QueryDatabase(ctx context.Context, id string,
	req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	m.mu.Lock()
	m.QueryDatabaseCalls = append(m.QueryDatabaseCalls, QueryDatabaseCall{
		Ctx:     ctx,
		ID:      id,
		Request: req,
	})
	m.mu.Unlock()

	if m.QueryDatabaseFunc != nil {
		return m.QueryDatabaseFunc(ctx, id, req)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.DatabaseToReturn != nil {
		return m.DatabaseToReturn, nil
	}

	return NewTestDatabase(id), nil
}

// GetBlocks retrieves child blocks of a page or block.
func (m *MockNotionClient) GetBlocks(ctx context.Context, id string,
	pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	m.mu.Lock()
	m.GetBlocksCalls = append(m.GetBlocksCalls, GetBlocksCall{
		Ctx:        ctx,
		ID:         id,
		Pagination: pagination,
	})
	m.mu.Unlock()

	if m.GetBlocksFunc != nil {
		return m.GetBlocksFunc(ctx, id, pagination)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.BlocksToReturn != nil {
		return m.BlocksToReturn, nil
	}

	blocks := NewTestBlockList(3)
	return NewGetChildrenResponse(blocks), nil
}

// GetBlock retrieves a single block by ID.
func (m *MockNotionClient) GetBlock(ctx context.Context, id string) (notionapi.Block, error) {
	m.mu.Lock()
	m.GetBlockCalls = append(m.GetBlockCalls, GetBlockCall{
		Ctx: ctx,
		ID:  id,
	})
	m.mu.Unlock()

	if m.GetBlockFunc != nil {
		return m.GetBlockFunc(ctx, id)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.BlockToReturn != nil {
		return m.BlockToReturn, nil
	}

	return NewParagraphBlock("Test block content"), nil
}

// UpdatePage updates a page's properties.
func (m *MockNotionClient) UpdatePage(ctx context.Context, id string,
	req *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	m.mu.Lock()
	m.UpdatePageCalls = append(m.UpdatePageCalls, UpdatePageCall{
		Ctx:     ctx,
		ID:      id,
		Request: req,
	})
	m.mu.Unlock()

	if m.UpdatePageFunc != nil {
		return m.UpdatePageFunc(ctx, id, req)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.PageToReturn != nil {
		return m.PageToReturn, nil
	}

	return NewTestPage(id, "Updated Page"), nil
}

// UpdateBlock updates a block's content.
func (m *MockNotionClient) UpdateBlock(ctx context.Context, id string,
	req *notionapi.BlockUpdateRequest) (notionapi.Block, error) {
	m.mu.Lock()
	m.UpdateBlockCalls = append(m.UpdateBlockCalls, UpdateBlockCall{
		Ctx:     ctx,
		ID:      id,
		Request: req,
	})
	m.mu.Unlock()

	if m.UpdateBlockFunc != nil {
		return m.UpdateBlockFunc(ctx, id, req)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.BlockToReturn != nil {
		return m.BlockToReturn, nil
	}

	return NewParagraphBlock("Updated block content"), nil
}

// AppendBlocks appends blocks to a page or block.
func (m *MockNotionClient) AppendBlocks(ctx context.Context, id string,
	req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error) {
	m.mu.Lock()
	m.AppendBlocksCalls = append(m.AppendBlocksCalls, AppendBlocksCall{
		Ctx:     ctx,
		ID:      id,
		Request: req,
	})
	m.mu.Unlock()

	if m.AppendBlocksFunc != nil {
		return m.AppendBlocksFunc(ctx, id, req)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.AppendResponseReturn != nil {
		return m.AppendResponseReturn, nil
	}

	return &notionapi.AppendBlockChildrenResponse{
		Object:  notionapi.ObjectTypeList,
		Results: req.Children,
	}, nil
}

// DeleteBlock archives a block.
func (m *MockNotionClient) DeleteBlock(ctx context.Context, id string) (notionapi.Block, error) {
	m.mu.Lock()
	m.DeleteBlockCalls = append(m.DeleteBlockCalls, DeleteBlockCall{
		Ctx: ctx,
		ID:  id,
	})
	m.mu.Unlock()

	if m.DeleteBlockFunc != nil {
		return m.DeleteBlockFunc(ctx, id)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.DeletedBlockReturn != nil {
		return m.DeletedBlockReturn, nil
	}

	return NewParagraphBlock("Deleted block"), nil
}

// Search performs a workspace-wide search.
func (m *MockNotionClient) Search(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error) {
	m.mu.Lock()
	m.SearchCalls = append(m.SearchCalls, SearchCall{
		Ctx:   ctx,
		Input: input,
	})
	m.mu.Unlock()

	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, input)
	}

	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}

	if m.SearchResponseReturn != nil {
		return m.SearchResponseReturn, nil
	}

	// Default mock response
	return &notion.SearchResponse{
		Results: []notion.SearchResult{
			{ID: "page-1", Title: "Test Page", ObjectType: "page", LastEdited: time.Now()},
			{ID: "db-1", Title: "Test Database", ObjectType: "database", LastEdited: time.Now()},
		},
		HasMore: false,
	}, nil
}

// CallCount returns the total number of calls made to all methods.
func (m *MockNotionClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.GetPageCalls) + len(m.QueryDatabaseCalls) + len(m.GetBlocksCalls) +
		len(m.GetBlockCalls) + len(m.UpdatePageCalls) + len(m.UpdateBlockCalls) +
		len(m.AppendBlocksCalls) + len(m.DeleteBlockCalls) + len(m.SearchCalls)
}

// GetPageCallCount returns the number of calls to GetPage.
func (m *MockNotionClient) GetPageCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.GetPageCalls)
}

// QueryDatabaseCallCount returns the number of calls to QueryDatabase.
func (m *MockNotionClient) QueryDatabaseCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.QueryDatabaseCalls)
}

// GetBlocksCallCount returns the number of calls to GetBlocks.
func (m *MockNotionClient) GetBlocksCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.GetBlocksCalls)
}

// GetBlockCallCount returns the number of calls to GetBlock.
func (m *MockNotionClient) GetBlockCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.GetBlockCalls)
}

// UpdatePageCallCount returns the number of calls to UpdatePage.
func (m *MockNotionClient) UpdatePageCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.UpdatePageCalls)
}

// UpdateBlockCallCount returns the number of calls to UpdateBlock.
func (m *MockNotionClient) UpdateBlockCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.UpdateBlockCalls)
}

// AppendBlocksCallCount returns the number of calls to AppendBlocks.
func (m *MockNotionClient) AppendBlocksCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.AppendBlocksCalls)
}

// DeleteBlockCallCount returns the number of calls to DeleteBlock.
func (m *MockNotionClient) DeleteBlockCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.DeleteBlockCalls)
}

// SearchCallCount returns the number of calls to Search.
func (m *MockNotionClient) SearchCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.SearchCalls)
}

// LastGetPageCall returns the most recent GetPage call, or nil if none.
func (m *MockNotionClient) LastGetPageCall() *GetPageCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.GetPageCalls) == 0 {
		return nil
	}
	return &m.GetPageCalls[len(m.GetPageCalls)-1]
}

// LastQueryDatabaseCall returns the most recent QueryDatabase call, or nil if none.
func (m *MockNotionClient) LastQueryDatabaseCall() *QueryDatabaseCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.QueryDatabaseCalls) == 0 {
		return nil
	}
	return &m.QueryDatabaseCalls[len(m.QueryDatabaseCalls)-1]
}

// LastGetBlocksCall returns the most recent GetBlocks call, or nil if none.
func (m *MockNotionClient) LastGetBlocksCall() *GetBlocksCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.GetBlocksCalls) == 0 {
		return nil
	}
	return &m.GetBlocksCalls[len(m.GetBlocksCalls)-1]
}

// LastGetBlockCall returns the most recent GetBlock call, or nil if none.
func (m *MockNotionClient) LastGetBlockCall() *GetBlockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.GetBlockCalls) == 0 {
		return nil
	}
	return &m.GetBlockCalls[len(m.GetBlockCalls)-1]
}

// LastUpdatePageCall returns the most recent UpdatePage call, or nil if none.
func (m *MockNotionClient) LastUpdatePageCall() *UpdatePageCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.UpdatePageCalls) == 0 {
		return nil
	}
	return &m.UpdatePageCalls[len(m.UpdatePageCalls)-1]
}

// LastUpdateBlockCall returns the most recent UpdateBlock call, or nil if none.
func (m *MockNotionClient) LastUpdateBlockCall() *UpdateBlockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.UpdateBlockCalls) == 0 {
		return nil
	}
	return &m.UpdateBlockCalls[len(m.UpdateBlockCalls)-1]
}

// LastAppendBlocksCall returns the most recent AppendBlocks call, or nil if none.
func (m *MockNotionClient) LastAppendBlocksCall() *AppendBlocksCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.AppendBlocksCalls) == 0 {
		return nil
	}
	return &m.AppendBlocksCalls[len(m.AppendBlocksCalls)-1]
}

// LastDeleteBlockCall returns the most recent DeleteBlock call, or nil if none.
func (m *MockNotionClient) LastDeleteBlockCall() *DeleteBlockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.DeleteBlockCalls) == 0 {
		return nil
	}
	return &m.DeleteBlockCalls[len(m.DeleteBlockCalls)-1]
}

// LastSearchCall returns the most recent Search call, or nil if none.
func (m *MockNotionClient) LastSearchCall() *SearchCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.SearchCalls) == 0 {
		return nil
	}
	return &m.SearchCalls[len(m.SearchCalls)-1]
}

// Reset clears all recorded calls and resets return values.
func (m *MockNotionClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetPageCalls = make([]GetPageCall, 0)
	m.QueryDatabaseCalls = make([]QueryDatabaseCall, 0)
	m.GetBlocksCalls = make([]GetBlocksCall, 0)
	m.GetBlockCalls = make([]GetBlockCall, 0)
	m.UpdatePageCalls = make([]UpdatePageCall, 0)
	m.UpdateBlockCalls = make([]UpdateBlockCall, 0)
	m.AppendBlocksCalls = make([]AppendBlocksCall, 0)
	m.DeleteBlockCalls = make([]DeleteBlockCall, 0)
	m.SearchCalls = make([]SearchCall, 0)

	m.GetPageFunc = nil
	m.QueryDatabaseFunc = nil
	m.GetBlocksFunc = nil
	m.GetBlockFunc = nil
	m.UpdatePageFunc = nil
	m.UpdateBlockFunc = nil
	m.AppendBlocksFunc = nil
	m.DeleteBlockFunc = nil
	m.SearchFunc = nil

	m.PageToReturn = nil
	m.DatabaseToReturn = nil
	m.BlocksToReturn = nil
	m.BlockToReturn = nil
	m.AppendResponseReturn = nil
	m.DeletedBlockReturn = nil
	m.SearchResponseReturn = nil
	m.ErrorToReturn = nil
}

// WithError configures the mock to return an error for all methods.
// Returns the mock for chaining.
func (m *MockNotionClient) WithError(err error) *MockNotionClient {
	m.ErrorToReturn = err
	return m
}

// WithPage configures the mock to return a specific page.
// Returns the mock for chaining.
func (m *MockNotionClient) WithPage(page *notionapi.Page) *MockNotionClient {
	m.PageToReturn = page
	return m
}

// WithDatabase configures the mock to return a specific database response.
// Returns the mock for chaining.
func (m *MockNotionClient) WithDatabase(db *notionapi.DatabaseQueryResponse) *MockNotionClient {
	m.DatabaseToReturn = db
	return m
}

// WithBlocks configures the mock to return specific blocks.
// Returns the mock for chaining.
func (m *MockNotionClient) WithBlocks(resp *notionapi.GetChildrenResponse) *MockNotionClient {
	m.BlocksToReturn = resp
	return m
}

// WithBlock configures the mock to return a specific block.
// Returns the mock for chaining.
func (m *MockNotionClient) WithBlock(block notionapi.Block) *MockNotionClient {
	m.BlockToReturn = block
	return m
}

// ErrNotFound is a sentinel error for testing not found scenarios.
var ErrNotFound = fmt.Errorf("resource not found")

// ErrUnauthorized is a sentinel error for testing unauthorized scenarios.
var ErrUnauthorized = fmt.Errorf("unauthorized access")

// ErrRateLimited is a sentinel error for testing rate limit scenarios.
var ErrRateLimited = fmt.Errorf("rate limited")

// ErrConflict is a sentinel error for testing conflict scenarios.
var ErrConflict = fmt.Errorf("resource conflict")

// ErrServerError is a sentinel error for testing server error scenarios.
var ErrServerError = fmt.Errorf("internal server error")
