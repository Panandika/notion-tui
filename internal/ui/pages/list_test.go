package pages

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// MockNotionClient is a mock implementation of the NotionClient interface.
type MockNotionClient struct {
	QueryDatabaseFunc func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error)
	GetPageFunc       func(ctx context.Context, id string) (*notionapi.Page, error)
	GetBlocksFunc     func(ctx context.Context, id string, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error)
	GetBlockFunc      func(ctx context.Context, id string) (notionapi.Block, error)
	UpdatePageFunc    func(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error)
	UpdateBlockFunc   func(ctx context.Context, id string, req *notionapi.BlockUpdateRequest) (notionapi.Block, error)
	AppendBlocksFunc  func(ctx context.Context, id string, req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error)
	DeleteBlockFunc   func(ctx context.Context, id string) (notionapi.Block, error)
	SearchFunc        func(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error)
}

func (m *MockNotionClient) QueryDatabase(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	if m.QueryDatabaseFunc != nil {
		return m.QueryDatabaseFunc(ctx, id, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) GetPage(ctx context.Context, id string) (*notionapi.Page, error) {
	if m.GetPageFunc != nil {
		return m.GetPageFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) GetBlocks(ctx context.Context, id string, pagination *notionapi.Pagination) (*notionapi.GetChildrenResponse, error) {
	if m.GetBlocksFunc != nil {
		return m.GetBlocksFunc(ctx, id, pagination)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) GetBlock(ctx context.Context, id string) (notionapi.Block, error) {
	if m.GetBlockFunc != nil {
		return m.GetBlockFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) UpdatePage(ctx context.Context, id string, req *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	if m.UpdatePageFunc != nil {
		return m.UpdatePageFunc(ctx, id, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) UpdateBlock(ctx context.Context, id string, req *notionapi.BlockUpdateRequest) (notionapi.Block, error) {
	if m.UpdateBlockFunc != nil {
		return m.UpdateBlockFunc(ctx, id, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) AppendBlocks(ctx context.Context, id string, req *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error) {
	if m.AppendBlocksFunc != nil {
		return m.AppendBlocksFunc(ctx, id, req)
	}
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) DeleteBlock(ctx context.Context, id string) (notionapi.Block, error) {
	if m.DeleteBlockFunc != nil {
		return m.DeleteBlockFunc(ctx, id)
	}
	return nil, errors.New("not implemented")
}

// The list page never touches page Markdown; these only satisfy the interface.
func (m *MockNotionClient) GetPageMarkdown(context.Context, string) (*notion.PageMarkdown, error) {
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) UpdatePageMarkdown(context.Context, string, []notion.ContentUpdate) (*notion.PageMarkdown, error) {
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) ReplacePageMarkdown(context.Context, string, string) (*notion.PageMarkdown, error) {
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) CreatePageMarkdown(context.Context, string, string) (*notion.CreatedPage, error) {
	return nil, errors.New("not implemented")
}

func (m *MockNotionClient) Search(ctx context.Context, input notion.SearchInput) (*notion.SearchResponse, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, input)
	}
	return &notion.SearchResponse{}, nil
}

// newTestPage creates a test page for testing.
func newTestPage(id, title, status string) Page {
	return Page{
		ID:        id,
		Title:     title,
		Status:    status,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}
}

// newTestNotionPage creates a test Notion page for testing.
func newTestNotionPage(id, title, status string) notionapi.Page {
	page := notionapi.Page{
		ID:             notionapi.ObjectID(id),
		LastEditedTime: time.Now().Add(-1 * time.Hour),
		Properties: notionapi.Properties{
			"Name": &notionapi.TitleProperty{
				ID:   "title",
				Type: notionapi.PropertyTypeTitle,
				Title: []notionapi.RichText{
					{
						Type:      notionapi.ObjectTypeText,
						PlainText: title,
					},
				},
			},
		},
	}

	if status != "" {
		page.Properties["Status"] = &notionapi.SelectProperty{
			ID:   "status",
			Type: notionapi.PropertyTypeSelect,
			Select: notionapi.Option{
				Name: status,
			},
		}
	}

	return page
}

func TestNewListPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input NewListPageInput
	}{
		{
			name: "creates list page with default values",
			input: NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: &MockNotionClient{},
				Cache:        nil,
				DatabaseID:   "test-db-123",
			},
		},
		{
			name: "creates list page with cache",
			input: NewListPageInput{
				Width:        100,
				Height:       30,
				NotionClient: &MockNotionClient{},
				Cache:        &cache.PageCache{},
				DatabaseID:   "test-db-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewListPage(tt.input)

			assert.Equal(t, tt.input.Width, lp.width)
			assert.Equal(t, tt.input.Height, lp.height)
			assert.Equal(t, tt.input.DatabaseID, lp.databaseID)
			assert.Equal(t, tt.input.NotionClient, lp.notionClient)
			assert.True(t, lp.loading)
			assert.Nil(t, lp.err)
			assert.Empty(t, lp.pageList)
			assert.Equal(t, -1, lp.selectedIdx)
		})
	}
}

func TestListPage_Init(t *testing.T) {
	t.Parallel()

	mockClient := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Test Page 1", "Draft"),
					newTestNotionPage("page-2", "Test Page 2", "Published"),
				},
			}, nil
		},
	}

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		DatabaseID:   "test-db",
	})

	cmd := lp.Init()
	assert.NotNil(t, cmd, "Init should return a command")

	// Init returns a batch, so we test the fetch directly
	fetchCmd := lp.fetchPagesCmd()
	msg := fetchCmd()
	loadedMsg, ok := msg.(pagesLoadedMsg)
	assert.True(t, ok, "Command should return pagesLoadedMsg")
	assert.NoError(t, loadedMsg.err)
	assert.Len(t, loadedMsg.pages, 2)
}

func TestListPage_Update_PagesLoaded(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		msg           pagesLoadedMsg
		expectLoading bool
		expectError   bool
		expectPages   int
	}{
		{
			name: "successfully loads pages",
			msg: pagesLoadedMsg{
				pages: []Page{
					newTestPage("page-1", "Page 1", "Draft"),
					newTestPage("page-2", "Page 2", "Published"),
				},
			},
			expectLoading: false,
			expectError:   false,
			expectPages:   2,
		},
		{
			name: "handles error loading pages",
			msg: pagesLoadedMsg{
				err: errors.New("database not found"),
			},
			expectLoading: false,
			expectError:   true,
			expectPages:   0,
		},
		{
			name: "handles empty page list",
			msg: pagesLoadedMsg{
				pages: []Page{},
			},
			expectLoading: false,
			expectError:   false,
			expectPages:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewListPage(NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: &MockNotionClient{},
				DatabaseID:   "test-db",
			})
			lp.loading = true

			model, _ := lp.Update(tt.msg)
			updatedLP := model.(*ListPage)
			updated := *updatedLP

			assert.Equal(t, tt.expectLoading, updated.loading)
			if tt.expectError {
				assert.NotNil(t, updated.err)
			} else {
				assert.Nil(t, updated.err)
			}
			assert.Len(t, updated.pageList, tt.expectPages)
		})
	}
}

func TestListPage_Update_ItemSelected(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: &MockNotionClient{},
		DatabaseID:   "test-db",
	})
	lp.pageList = []Page{
		newTestPage("page-1", "Page 1", "Draft"),
		newTestPage("page-2", "Page 2", "Published"),
	}

	msg := components.ItemSelectedMsg{
		ID:    "page-2",
		Title: "Page 2",
		Index: 1,
	}

	model, cmd := lp.Update(msg)
	updatedLP := model.(*ListPage)
	updated := *updatedLP

	assert.Equal(t, 1, updated.selectedIdx)
	assert.NotNil(t, cmd)

	// Execute the command to get navigation message
	navMsg := cmd()
	nav, ok := navMsg.(NavigationMsg)
	assert.True(t, ok)
	assert.Equal(t, "page-2", nav.PageID())
}

func TestListPage_Update_RefreshKey(t *testing.T) {
	t.Parallel()

	mockClient := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Refreshed Page", "Draft"),
				},
			}, nil
		},
	}

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		DatabaseID:   "test-db",
	})
	lp.loading = false

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	model, cmd := lp.Update(msg)
	updatedLP := model.(*ListPage)
	updated := *updatedLP

	assert.True(t, updated.loading)
	assert.NotNil(t, cmd)

	// Execute the refresh command
	loadedMsg := cmd().(pagesLoadedMsg)
	assert.NoError(t, loadedMsg.err)
	assert.Len(t, loadedMsg.pages, 1)
	assert.Equal(t, "Refreshed Page", loadedMsg.pages[0].Title)
}

func TestListPage_Update_WindowSize(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: &MockNotionClient{},
		DatabaseID:   "test-db",
	})

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	model, _ := lp.Update(msg)
	updatedLP := model.(*ListPage)
	updated := *updatedLP

	assert.Equal(t, 120, updated.width)
	assert.Equal(t, 40, updated.height)
}

func TestListPage_View(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		loading     bool
		err         error
		expectMatch string
	}{
		{
			name:        "shows loading state",
			loading:     true,
			err:         nil,
			expectMatch: "Loading pages...",
		},
		{
			name:        "shows error state",
			loading:     false,
			err:         errors.New("database error"),
			expectMatch: "Error loading pages",
		},
		{
			name:        "shows normal state",
			loading:     false,
			err:         nil,
			expectMatch: "This database has no pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewListPage(NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: &MockNotionClient{},
				DatabaseID:   "test-db",
			})
			lp.loading = tt.loading
			lp.err = tt.err

			view := lp.View()
			assert.Contains(t, view, tt.expectMatch)
		})
	}
}

func TestListPage_SetPages(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: &MockNotionClient{},
		DatabaseID:   "test-db",
	})

	pages := []Page{
		newTestPage("page-1", "Page 1", "Draft"),
		newTestPage("page-2", "Page 2", "Published"),
		newTestPage("page-3", "Page 3", "Archived"),
	}

	lp.SetPages(pages)

	assert.Len(t, lp.pageList, 3)
	assert.Len(t, lp.sidebar.Items(), 3)

	// Verify sidebar items match pages
	items := lp.sidebar.Items()
	for i, item := range items {
		assert.Equal(t, pages[i].Title, item.Title())
		assert.Equal(t, pages[i].ID, item.ID())
	}
}

func TestListPage_SelectedPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		selectedIdx int
		expectNil   bool
		expectTitle string
	}{
		{
			name:        "returns selected page",
			selectedIdx: 1,
			expectNil:   false,
			expectTitle: "Page 2",
		},
		{
			name:        "returns nil for invalid index",
			selectedIdx: -1,
			expectNil:   true,
		},
		{
			name:        "returns nil for out of range index",
			selectedIdx: 10,
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewListPage(NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: &MockNotionClient{},
				DatabaseID:   "test-db",
			})
			lp.pageList = []Page{
				newTestPage("page-1", "Page 1", "Draft"),
				newTestPage("page-2", "Page 2", "Published"),
			}
			lp.selectedIdx = tt.selectedIdx

			page := lp.SelectedPage()

			if tt.expectNil {
				assert.Nil(t, page)
			} else {
				assert.NotNil(t, page)
				assert.Equal(t, tt.expectTitle, page.Title)
			}
		})
	}
}

func TestListPage_FetchPagesCmd_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		mockClient  NotionClient
		expectError string
	}{
		{
			name:        "handles nil client",
			mockClient:  nil,
			expectError: "notion client not initialized",
		},
		{
			name: "handles API error",
			mockClient: &MockNotionClient{
				QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
					return nil, errors.New("API request failed")
				},
			},
			expectError: "fetch pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewListPage(NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: tt.mockClient,
				DatabaseID:   "test-db",
			})

			cmd := lp.fetchPagesCmd()
			msg := cmd()
			loadedMsg, ok := msg.(pagesLoadedMsg)

			assert.True(t, ok)
			assert.Error(t, loadedMsg.err)
			assert.Contains(t, loadedMsg.err.Error(), tt.expectError)
		})
	}
}

func TestExtractTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		page        *notionapi.Page
		expectTitle string
	}{
		{
			name:        "returns Untitled for nil page",
			page:        nil,
			expectTitle: "Untitled",
		},
		{
			name: "extracts title from title property",
			page: &notionapi.Page{
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "My Page Title"},
						},
					},
				},
			},
			expectTitle: "My Page Title",
		},
		{
			name: "returns Untitled for page without title",
			page: &notionapi.Page{
				Properties: notionapi.Properties{},
			},
			expectTitle: "Untitled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title := extractTitle(tt.page)
			assert.Equal(t, tt.expectTitle, title)
		})
	}
}

func TestExtractStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		page         *notionapi.Page
		expectStatus string
	}{
		{
			name:         "returns empty for nil page",
			page:         nil,
			expectStatus: "",
		},
		{
			name: "extracts status from status property",
			page: &notionapi.Page{
				Properties: notionapi.Properties{
					"Status": &notionapi.StatusProperty{
						Status: notionapi.Status{
							Name: "In Progress",
						},
					},
				},
			},
			expectStatus: "In Progress",
		},
		{
			name: "extracts status from select property",
			page: &notionapi.Page{
				Properties: notionapi.Properties{
					"status": &notionapi.SelectProperty{
						Select: notionapi.Option{
							Name: "Done",
						},
					},
				},
			},
			expectStatus: "Done",
		},
		{
			name: "returns empty for page without status",
			page: &notionapi.Page{
				Properties: notionapi.Properties{},
			},
			expectStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := extractStatus(tt.page)
			assert.Equal(t, tt.expectStatus, status)
		})
	}
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name       string
		time       time.Time
		expectText string
	}{
		{
			name:       "just now",
			time:       now.Add(-30 * time.Second),
			expectText: "just now",
		},
		{
			name:       "minutes ago",
			time:       now.Add(-5 * time.Minute),
			expectText: "5 minutes ago",
		},
		{
			name:       "hours ago",
			time:       now.Add(-3 * time.Hour),
			expectText: "3 hours ago",
		},
		{
			name:       "yesterday",
			time:       now.Add(-24 * time.Hour),
			expectText: "yesterday",
		},
		{
			name:       "days ago",
			time:       now.Add(-3 * 24 * time.Hour),
			expectText: "3 days ago",
		},
		{
			name:       "weeks ago",
			time:       now.Add(-2 * 7 * 24 * time.Hour),
			expectText: "2 weeks ago",
		},
		{
			name:       "formatted date for old",
			time:       now.Add(-40 * 24 * time.Hour),
			expectText: now.Add(-40 * 24 * time.Hour).Format("Jan 2, 2006"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.time)
			assert.Equal(t, tt.expectText, result)
		})
	}
}

func TestListPage_Pagination(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		firstResponse    *notionapi.DatabaseQueryResponse
		secondResponse   *notionapi.DatabaseQueryResponse
		expectFirstCount int
		expectTotalCount int
		expectHasMore    bool
		expectNextCursor string
	}{
		{
			name: "handles pagination with more pages",
			firstResponse: &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Page 1", "Draft"),
					newTestNotionPage("page-2", "Page 2", "Draft"),
				},
				HasMore:    true,
				NextCursor: "cursor-123",
			},
			secondResponse: &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-3", "Page 3", "Draft"),
					newTestNotionPage("page-4", "Page 4", "Draft"),
				},
				HasMore:    false,
				NextCursor: "",
			},
			expectFirstCount: 2,
			expectTotalCount: 4,
			expectHasMore:    false,
			expectNextCursor: "",
		},
		{
			name: "handles single page with no more results",
			firstResponse: &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Page 1", "Draft"),
				},
				HasMore:    false,
				NextCursor: "",
			},
			expectFirstCount: 1,
			expectTotalCount: 1,
			expectHasMore:    false,
			expectNextCursor: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockClient := &MockNotionClient{
				QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
					callCount++
					if callCount == 1 {
						return tt.firstResponse, nil
					}
					return tt.secondResponse, nil
				},
			}

			lp := NewListPage(NewListPageInput{
				Width:        80,
				Height:       24,
				NotionClient: mockClient,
				DatabaseID:   "test-db",
			})

			// Initial load
			fetchCmd := lp.fetchPagesCmd()
			msg := fetchCmd().(pagesLoadedMsg)
			model, _ := lp.Update(msg)
			updatedLP := model.(*ListPage)
			lp = *updatedLP

			assert.Len(t, lp.pageList, tt.expectFirstCount)
			assert.Equal(t, tt.firstResponse.HasMore, lp.HasMore())
			assert.Equal(t, string(tt.firstResponse.NextCursor), lp.NextCursor())

			// Load more if available
			if tt.secondResponse != nil && tt.firstResponse.HasMore {
				lp.loadingMore = true
				loadMoreCmd := lp.loadMoreCmd()
				moreMsg := loadMoreCmd().(pagesLoadedMsg)
				model, _ = lp.Update(moreMsg)
				updatedLP = model.(*ListPage)
				lp = *updatedLP

				assert.Len(t, lp.pageList, tt.expectTotalCount)
				assert.Equal(t, tt.expectHasMore, lp.HasMore())
				assert.Equal(t, tt.expectNextCursor, lp.NextCursor())
			}
		})
	}
}

func TestListPage_LoadMoreKey(t *testing.T) {
	t.Parallel()

	mockClient := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			if req != nil && req.StartCursor != "" {
				// Second page
				return &notionapi.DatabaseQueryResponse{
					Results: []notionapi.Page{
						newTestNotionPage("page-3", "Page 3", "Draft"),
					},
					HasMore:    false,
					NextCursor: "",
				}, nil
			}
			// First page
			return &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Page 1", "Draft"),
					newTestNotionPage("page-2", "Page 2", "Draft"),
				},
				HasMore:    true,
				NextCursor: "cursor-abc",
			}, nil
		},
	}

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		DatabaseID:   "test-db",
	})

	// Initial load - need to extract the fetch command from the batch
	// Since Init returns a batch of spinner.Tick and fetchPagesCmd,
	// we'll call fetchPagesCmd directly for testing
	fetchCmd := lp.fetchPagesCmd()
	msg := fetchCmd().(pagesLoadedMsg)
	model, _ := lp.Update(msg)
	updatedLP := model.(*ListPage)
	lp = *updatedLP

	assert.Len(t, lp.pageList, 2)
	assert.True(t, lp.HasMore())

	// Press 'm' to load more
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}}
	model, _ = lp.Update(keyMsg)
	updatedLP = model.(*ListPage)
	lp = *updatedLP

	assert.True(t, lp.IsLoadingMore())

	// Execute load more command directly
	loadMoreCmd := lp.loadMoreCmd()
	moreMsg := loadMoreCmd().(pagesLoadedMsg)
	model, _ = lp.Update(moreMsg)
	updatedLP = model.(*ListPage)
	lp = *updatedLP

	assert.Len(t, lp.pageList, 3)
	assert.False(t, lp.HasMore())
	assert.False(t, lp.IsLoadingMore())
}

func TestListPage_LoadMore_NoCursor(t *testing.T) {
	t.Parallel()

	mockClient := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Page 1", "Draft"),
				},
				HasMore:    false,
				NextCursor: "",
			}, nil
		},
	}

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		DatabaseID:   "test-db",
	})
	lp.hasMore = true
	lp.nextCursor = ""
	lp.loadingMore = true

	cmd := lp.loadMoreCmd()
	msg := cmd().(pagesLoadedMsg)

	assert.Error(t, msg.err)
	assert.Contains(t, msg.err.Error(), "no cursor available")
}

func TestListPage_PaginationView(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: &MockNotionClient{},
		DatabaseID:   "test-db",
	})
	lp.loading = true
	lp.loadingMore = true
	lp.spinner.SetMessage("Loading more pages...")
	lp.pageList = []Page{
		newTestPage("page-1", "Page 1", "Draft"),
		newTestPage("page-2", "Page 2", "Draft"),
	}

	view := lp.View()
	// The view will contain the spinner with the message
	assert.Contains(t, view, "(2 loaded)")
}

func TestListPage_RefreshClearsPagination(t *testing.T) {
	t.Parallel()

	mockClient := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return &notionapi.DatabaseQueryResponse{
				Results: []notionapi.Page{
					newTestNotionPage("page-1", "Fresh Page", "Draft"),
				},
				HasMore:    false,
				NextCursor: "",
			}, nil
		},
	}

	lp := NewListPage(NewListPageInput{
		Width:        80,
		Height:       24,
		NotionClient: mockClient,
		DatabaseID:   "test-db",
	})
	lp.loading = false
	lp.hasMore = true
	lp.nextCursor = "old-cursor"

	// Refresh
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	model, _ := lp.Update(keyMsg)
	updatedLP := model.(*ListPage)
	lp = *updatedLP

	assert.True(t, lp.loading)
	assert.False(t, lp.HasMore())
	assert.Equal(t, "", lp.NextCursor())

	// Execute refresh command directly
	refreshCmd := lp.fetchPagesCmd()
	refreshMsg := refreshCmd().(pagesLoadedMsg)
	model, _ = lp.Update(refreshMsg)
	updatedLP = model.(*ListPage)
	lp = *updatedLP

	assert.False(t, lp.HasMore())
	assert.Equal(t, "", lp.NextCursor())
}

func TestListPage_ShowsDatabaseRows(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{Width: 80, Height: 24, NotionClient: &MockNotionClient{}, DatabaseID: "db"})
	lp.Update(pagesLoadedMsg{pages: []Page{
		NewPage("row-1", "Write report", "In progress", time.Now()),
		NewPage("row-2", "Book flights", "To do", time.Now()),
	}})

	view := lp.View()

	assert.Contains(t, view, "Write report", "rows must be visible on the page itself")
	assert.Contains(t, view, "Book flights")
	assert.NotContains(t, view, "Select a page from the sidebar")
}

func TestListPage_EnterOpensTheSelectedRow(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{Width: 80, Height: 24, NotionClient: &MockNotionClient{}, DatabaseID: "db"})
	lp.Update(pagesLoadedMsg{pages: []Page{
		NewPage("row-1", "Write report", "", time.Now()),
		NewPage("row-2", "Book flights", "", time.Now()),
	}})

	lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_, cmd := lp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	selected := firstMsgOfType[components.ItemSelectedMsg](t, cmd)

	_, cmd = lp.Update(selected)
	require.NotNil(t, cmd)
	nav, ok := cmd().(NavigationMsg)
	require.True(t, ok)
	assert.Equal(t, "row-2", nav.PageID())
}

func TestListPage_FilterTypingIsNotShortcuts(t *testing.T) {
	t.Parallel()

	client := &MockNotionClient{}
	queries := 0
	client.QueryDatabaseFunc = func(context.Context, string, *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
		queries++
		return &notionapi.DatabaseQueryResponse{}, nil
	}
	lp := NewListPage(NewListPageInput{Width: 80, Height: 24, NotionClient: client, DatabaseID: "db"})
	lp.Update(pagesLoadedMsg{pages: []Page{NewPage("row-1", "Write report", "", time.Now())}})
	require.False(t, lp.CapturesInput())

	lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.True(t, lp.CapturesInput(), "typing a filter must capture the keyboard")

	// "r" is the refresh shortcut; while filtering it is just a letter
	_, cmd := lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	runAll(cmd)

	assert.Zero(t, queries, "typing r in the filter must not refresh")
	assert.False(t, lp.loading)
}

// firstMsgOfType runs a command (unpacking batches) and returns the first
// message of type T it produces.
func firstMsgOfType[T any](t *testing.T, cmd tea.Cmd) T {
	t.Helper()
	for _, msg := range collectMsgs(cmd) {
		if typed, ok := msg.(T); ok {
			return typed
		}
	}
	var zero T
	t.Fatalf("no message of type %T produced", zero)
	return zero
}

// collectMsgs runs a command and returns its messages, unpacking batches.
func collectMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, inner := range batch {
			out = append(out, collectMsgs(inner)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func runAll(cmd tea.Cmd) {
	collectMsgs(cmd)
}

func TestListPage_FitsItsSize(t *testing.T) {
	t.Parallel()

	var rows []Page
	for i := 0; i < 40; i++ {
		rows = append(rows, NewPage("row", "A row title that is reasonably long", "To do", time.Now()))
	}

	for _, size := range []struct{ width, height int }{
		{80, 24}, {80, 14}, {80, 10}, {80, 6}, {30, 24}, {20, 8},
	} {
		for _, filtering := range []bool{false, true} {
			name := fmt.Sprintf("%dx%d filtering=%v", size.width, size.height, filtering)
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				lp := NewListPage(NewListPageInput{Width: size.width, Height: size.height, NotionClient: &MockNotionClient{}, DatabaseID: "db"})
				lp.Update(pagesLoadedMsg{pages: rows})
				if filtering {
					lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
				}

				view := lp.View()

				// A taller or wider view pushes the top of the screen out of sight.
				assert.LessOrEqual(t, lipgloss.Height(view), size.height)
				assert.LessOrEqual(t, lipgloss.Width(view), size.width)
			})
		}
	}
}

func TestListPage_TitleVisibleAtNormalSize(t *testing.T) {
	t.Parallel()

	lp := NewListPage(NewListPageInput{Width: 80, Height: 24, NotionClient: &MockNotionClient{}, DatabaseID: "db"})
	lp.Update(pagesLoadedMsg{pages: []Page{NewPage("row", "A row", "To do", time.Now())}})

	view := lp.View()

	assert.Contains(t, view, "Pages")
	assert.NotContains(t, view, "? more", "? belongs to the app's help, not the list's")
}
