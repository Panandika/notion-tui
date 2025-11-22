package pages

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"

	"github.com/Panandika/notion-tui/internal/cache"
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

	// Execute the command
	msg := cmd()
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
			expectMatch: "Select a page from the sidebar",
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
