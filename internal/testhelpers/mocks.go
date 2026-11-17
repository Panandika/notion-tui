package testhelpers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/notion"
)

// MockNotionClient provides a mock implementation for testing Notion API interactions.
// It implements the common interface pattern used in the notion client.
type MockNotionClient struct {
	mu sync.Mutex

	// Configurable return values for each method
	GetPageFunc       func(ctx context.Context, id string) (*notionapi.Page, error)
	QueryDatabaseFunc func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error)
	UpdatePageFunc    func(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error)
	SearchFunc        func(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error)

	// Page Markdown behavior (see mocks_markdown.go)
	PageMarkdownToReturn     *notion.PageMarkdown
	GetPageMarkdownFunc      func(ctx context.Context, id string) (*notion.PageMarkdown, error)
	UpdatePageMarkdownFunc   func(ctx context.Context, id string, updates []notion.ContentUpdate) (*notion.PageMarkdown, error)
	ReplacePageMarkdownFunc  func(ctx context.Context, id, markdown string) (*notion.PageMarkdown, error)
	CreatePageMarkdownFunc   func(ctx context.Context, parent notion.PageParent, markdown string) (*notion.CreatedPage, error)
	DatabaseSchemaToReturn   *notionapi.Database
	CommentsToReturn         []notion.Comment
	ListCommentsCalls        int
	AddCommentCalls          []AddCommentCall
	AddCommentErr            error // Fails AddComment only
	GetDatabaseFunc          func(ctx context.Context, id string) (*notionapi.Database, error)
	SetPagePropertyFunc      func(ctx context.Context, pageID string, update notion.PropertyUpdate) error
	GetDatabaseCalls         []string
	SetPagePropertyCalls     []SetPagePropertyCall
	GetPageMarkdownCalls     []string
	UpdatePageMarkdownCalls  []UpdatePageMarkdownCall
	ReplacePageMarkdownCalls []ReplacePageMarkdownCall
	CreatePageMarkdownCalls  []CreatePageMarkdownCall

	// Call tracking for assertions
	GetPageCalls       []GetPageCall
	QueryDatabaseCalls []QueryDatabaseCall
	UpdatePageCalls    []UpdatePageCall
	SearchCalls        []SearchCall

	// Simple return values for common scenarios
	PageToReturn         *notionapi.Page
	DatabaseToReturn     *notionapi.DatabaseQueryResponse
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

// UpdatePageCall records a call to UpdatePage.
type UpdatePageCall struct {
	Ctx     context.Context
	ID      string
	Request *notionapi.PageUpdateRequest
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
		UpdatePageCalls:    make([]UpdatePageCall, 0),
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
	return len(m.GetPageCalls) + len(m.QueryDatabaseCalls) + len(m.UpdatePageCalls) +
		len(m.SearchCalls)
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

// UpdatePageCallCount returns the number of calls to UpdatePage.
func (m *MockNotionClient) UpdatePageCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.UpdatePageCalls)
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

// LastUpdatePageCall returns the most recent UpdatePage call, or nil if none.
func (m *MockNotionClient) LastUpdatePageCall() *UpdatePageCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.UpdatePageCalls) == 0 {
		return nil
	}
	return &m.UpdatePageCalls[len(m.UpdatePageCalls)-1]
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
	m.UpdatePageCalls = make([]UpdatePageCall, 0)
	m.SearchCalls = make([]SearchCall, 0)

	m.GetPageFunc = nil
	m.QueryDatabaseFunc = nil
	m.UpdatePageFunc = nil
	m.SearchFunc = nil

	m.PageToReturn = nil
	m.DatabaseToReturn = nil
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
