package pages

import (
	"context"
	"errors"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/testhelpers"
)

// mockViewer implements ViewerInterface for testing.
type mockViewer struct {
	blocks       []notionapi.Block
	width        int
	height       int
	updateCalled int
	initCalled   bool
}

func newMockViewer() *mockViewer {
	return &mockViewer{
		blocks: []notionapi.Block{},
	}
}

func (m *mockViewer) Init() tea.Cmd {
	m.initCalled = true
	return nil
}

func (m *mockViewer) Update(msg tea.Msg) (ViewerInterface, tea.Cmd) {
	m.updateCalled++
	return m, nil
}

func (m *mockViewer) View() string {
	if len(m.blocks) == 0 {
		return "No content"
	}
	return "Viewer content with blocks"
}

func (m *mockViewer) SetBlocks(blocks []notionapi.Block) tea.Cmd {
	m.blocks = blocks
	return nil
}

func (m *mockViewer) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func TestNewDetailPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      NewDetailPageInput
		wantWidth  int
		wantHeight int
		wantPageID string
	}{
		{
			name: "basic initialization",
			input: NewDetailPageInput{
				Width:        80,
				Height:       24,
				Viewer:       newMockViewer(),
				NotionClient: testhelpers.NewMockNotionClient(),
				PageID:       "page-123",
			},
			wantWidth:  80,
			wantHeight: 24,
			wantPageID: "page-123",
		},
		{
			name: "with cache",
			input: NewDetailPageInput{
				Width:        100,
				Height:       40,
				Viewer:       newMockViewer(),
				NotionClient: testhelpers.NewMockNotionClient(),
				Cache:        mustCreateCache(t),
				PageID:       "page-456",
			},
			wantWidth:  100,
			wantHeight: 40,
			wantPageID: "page-456",
		},
		{
			name: "minimal setup",
			input: NewDetailPageInput{
				Width:        40,
				Height:       10,
				NotionClient: testhelpers.NewMockNotionClient(),
				PageID:       "page-minimal",
			},
			wantWidth:  40,
			wantHeight: 10,
			wantPageID: "page-minimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dp := NewDetailPage(tt.input)

			assert.Equal(t, tt.wantPageID, dp.PageID())
			assert.True(t, dp.IsLoading())
			assert.Nil(t, dp.Error())
			assert.Nil(t, dp.Page())
			assert.Nil(t, dp.Blocks())
		})
	}
}

func TestDetailPageInit(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testhelpers.NewTestPage("page-1", "Test Page")
	mockClient.BlocksToReturn = testhelpers.NewGetChildrenResponse(
		testhelpers.NewTestBlockList(3),
	)

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-1",
	})

	cmd := dp.Init()
	require.NotNil(t, cmd)

	// Init should return a batch command and call viewer.Init()
	assert.True(t, viewer.initCalled)
}

func TestDetailPageLoadSuccess(t *testing.T) {
	t.Parallel()

	testBlocks := testhelpers.NewTestBlockList(5)
	testPage := testhelpers.NewTestPage("page-success", "Success Page")

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testPage
	mockClient.BlocksToReturn = testhelpers.NewGetChildrenResponse(testBlocks)

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-success",
	})

	// Execute the fetch command
	cmd := dp.fetchPageCmd()
	msg := cmd()

	// Should return pageLoadedMsg
	loadedMsg, ok := msg.(pageLoadedMsg)
	require.True(t, ok)
	assert.NoError(t, loadedMsg.err)
	assert.NotNil(t, loadedMsg.page)
	assert.Equal(t, 5, len(loadedMsg.blocks))

	// Update with the loaded message
	updated, _ := dp.Update(loadedMsg)
	updatedDP := updated.(*DetailPage)
	dp = *updatedDP

	assert.False(t, dp.IsLoading())
	assert.NoError(t, dp.Error())
	assert.NotNil(t, dp.Page())
	assert.Equal(t, 5, len(dp.Blocks()))
	assert.Equal(t, 5, len(viewer.blocks))
}

func TestDetailPageLoadError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("API error")
	mockClient := testhelpers.NewMockNotionClient()
	mockClient.ErrorToReturn = testErr

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-error",
	})

	// Execute the fetch command
	cmd := dp.fetchPageCmd()
	msg := cmd()

	// Should return pageLoadedMsg with error
	loadedMsg, ok := msg.(pageLoadedMsg)
	require.True(t, ok)
	assert.Error(t, loadedMsg.err)

	// Update with the error message
	updated, _ := dp.Update(loadedMsg)
	updatedDP := updated.(*DetailPage)
	dp = *updatedDP

	assert.False(t, dp.IsLoading())
	assert.Error(t, dp.Error())
	assert.Nil(t, dp.Page())
}

func TestDetailPageCacheHit(t *testing.T) {
	t.Parallel()

	testBlocks := testhelpers.NewTestBlockList(3)
	testPage := testhelpers.NewTestPage("page-cached", "Cached Page")
	pageID := "page-cached"

	// Setup cache with data
	testCache := mustCreateCache(t)
	ctx := context.Background()
	err := testCache.Set(ctx, cache.SetInput{
		PageID: pageID,
		Data: notionapi.GetChildrenResponse{
			Object:  notionapi.ObjectTypeList,
			Results: testBlocks,
		},
		TTL: time.Hour,
	})
	require.NoError(t, err)

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testPage

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		Cache:        testCache,
		PageID:       pageID,
	})

	// Execute fetch - should hit cache
	cmd := dp.fetchPageCmd()
	msg := cmd()

	loadedMsg, ok := msg.(pageLoadedMsg)
	require.True(t, ok)
	assert.NoError(t, loadedMsg.err)
	assert.Equal(t, 3, len(loadedMsg.blocks))

	// Should still call GetPage for metadata
	assert.Equal(t, 1, mockClient.GetPageCallCount())
}

func TestDetailPageCacheMiss(t *testing.T) {
	t.Parallel()

	testBlocks := testhelpers.NewTestBlockList(4)
	testPage := testhelpers.NewTestPage("page-miss", "Miss Page")

	testCache := mustCreateCache(t)

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testPage
	mockClient.BlocksToReturn = testhelpers.NewGetChildrenResponse(testBlocks)

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		Cache:        testCache,
		PageID:       "page-miss",
	})

	// Execute fetch - cache miss
	cmd := dp.fetchPageCmd()
	msg := cmd()

	loadedMsg, ok := msg.(pageLoadedMsg)
	require.True(t, ok)
	assert.NoError(t, loadedMsg.err)
	assert.Equal(t, 4, len(loadedMsg.blocks))

	// Should call both GetPage and GetBlocks
	assert.Equal(t, 1, mockClient.GetPageCallCount())
	assert.Equal(t, 1, mockClient.GetBlocksCallCount())
}

func TestDetailPageRefresh(t *testing.T) {
	t.Parallel()

	testBlocks := testhelpers.NewTestBlockList(2)
	testPage := testhelpers.NewTestPage("page-refresh", "Refresh Page")

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testPage
	mockClient.BlocksToReturn = testhelpers.NewGetChildrenResponse(testBlocks)

	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-refresh",
	})

	// Perform refresh
	cmd := dp.Refresh()
	require.NotNil(t, cmd)

	msg := cmd()
	loadedMsg, ok := msg.(pageLoadedMsg)
	require.True(t, ok)
	assert.NoError(t, loadedMsg.err)

	// Should bypass cache and call API directly
	assert.Equal(t, 1, mockClient.GetPageCallCount())
	assert.Equal(t, 1, mockClient.GetBlocksCallCount())
}

func TestDetailPageKeyHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		key            string
		wantNavMsg     bool
		navAction      string
		wantBackNavMsg bool
	}{
		{
			name:       "r key triggers refresh",
			key:        "r",
			wantNavMsg: false,
		},
		{
			name:       "e key navigates to edit",
			key:        "e",
			wantNavMsg: true,
			navAction:  "edit",
		},
		{
			name:           "esc key navigates back",
			key:            "esc",
			wantBackNavMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := testhelpers.NewMockNotionClient()
			viewer := newMockViewer()

			dp := NewDetailPage(NewDetailPageInput{
				Width:        80,
				Height:       24,
				Viewer:       viewer,
				NotionClient: mockClient,
				PageID:       "page-keys",
			})

			// Set to loaded state
			dp.loading = false

			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			_, cmd := dp.Update(keyMsg)

			if tt.wantNavMsg {
				require.NotNil(t, cmd)
				msg := cmd()
				navMsg, ok := msg.(navigationMsg)
				require.True(t, ok)
				assert.Equal(t, tt.navAction, navMsg.action)
			}

			if tt.wantBackNavMsg {
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(BackNavigationMsg)
				require.True(t, ok)
			}
		})
	}
}

func TestDetailPageWindowResize(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	viewer := newMockViewer()

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-resize",
	})

	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updated, _ := dp.Update(resizeMsg)
	updatedDP := updated.(*DetailPage)
	dp = *updatedDP

	// Viewer should be resized (height - 1 for status bar)
	assert.Equal(t, 120, viewer.width)
	assert.Equal(t, 39, viewer.height)
}

func TestDetailPageViewStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		setupFunc    func(*DetailPage)
		wantContains string
	}{
		{
			name: "loading state",
			setupFunc: func(dp *DetailPage) {
				dp.loading = true
			},
			wantContains: "Loading",
		},
		{
			name: "error state",
			setupFunc: func(dp *DetailPage) {
				dp.loading = false
				dp.err = errors.New("test error")
			},
			wantContains: "Error",
		},
		{
			name: "loaded state",
			setupFunc: func(dp *DetailPage) {
				dp.loading = false
				dp.err = nil
			},
			wantContains: "Viewer content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := testhelpers.NewMockNotionClient()
			viewer := newMockViewer()
			viewer.blocks = testhelpers.NewTestBlockList(1)

			dp := NewDetailPage(NewDetailPageInput{
				Width:        80,
				Height:       24,
				Viewer:       viewer,
				NotionClient: mockClient,
				PageID:       "page-view",
			})

			tt.setupFunc(&dp)

			view := dp.View()
			assert.Contains(t, view, tt.wantContains)
		})
	}
}

func TestDetailPageLoadPage(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	viewer := newMockViewer()

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-old",
	})

	// Load a different page
	cmd := dp.LoadPage("page-new")
	require.NotNil(t, cmd)

	assert.Equal(t, "page-new", dp.PageID())
	assert.True(t, dp.IsLoading())
	assert.Nil(t, dp.Error())
}

func TestDetailPageWithoutViewer(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	mockClient.PageToReturn = testhelpers.NewTestPage("page-no-viewer", "No Viewer")
	mockClient.BlocksToReturn = testhelpers.NewGetChildrenResponse(
		testhelpers.NewTestBlockList(2),
	)

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       nil, // No viewer
		NotionClient: mockClient,
		PageID:       "page-no-viewer",
	})

	// Execute load
	cmd := dp.fetchPageCmd()
	msg := cmd()
	loadedMsg := msg.(pageLoadedMsg)

	updated, _ := dp.Update(loadedMsg)
	updatedDP := updated.(*DetailPage)
	dp = *updatedDP

	// Should still work without viewer
	assert.False(t, dp.IsLoading())
	assert.NoError(t, dp.Error())

	view := dp.View()
	assert.Contains(t, view, "No viewer available")
}

func TestDetailPageMessageDelegation(t *testing.T) {
	t.Parallel()

	mockClient := testhelpers.NewMockNotionClient()
	viewer := newMockViewer()

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-delegate",
	})

	// Set to loaded state
	dp.loading = false

	// Send arbitrary message
	type customMsg struct{}
	_, _ = dp.Update(customMsg{})

	// Viewer should receive the update
	assert.Greater(t, viewer.updateCalled, 0)
}

func TestDetailPageGetters(t *testing.T) {
	t.Parallel()

	testPage := testhelpers.NewTestPage("page-getters", "Test Page")
	testBlocks := testhelpers.NewTestBlockList(3)

	mockClient := testhelpers.NewMockNotionClient()
	viewer := newMockViewer()

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-getters",
	})

	// Set internal state
	dp.page = testPage
	dp.blocks = testBlocks
	dp.loading = false
	dp.err = nil

	assert.Equal(t, "page-getters", dp.PageID())
	assert.Equal(t, testPage, dp.Page())
	assert.Equal(t, testBlocks, dp.Blocks())
	assert.False(t, dp.IsLoading())
	assert.NoError(t, dp.Error())
}

func TestDetailPageWithError(t *testing.T) {
	t.Parallel()

	testErr := testhelpers.ErrNotFound

	mockClient := testhelpers.NewMockNotionClient()
	viewer := newMockViewer()

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: mockClient,
		PageID:       "page-error-test",
	})

	dp.err = testErr
	dp.loading = false

	assert.Equal(t, testErr, dp.Error())

	view := dp.View()
	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "ESC")
}

// mustCreateCache creates a cache for testing or fails the test.
func mustCreateCache(t *testing.T) *cache.PageCache {
	t.Helper()
	c, err := cache.NewPageCache(cache.NewPageCacheInput{
		Dir: t.TempDir(),
	})
	require.NoError(t, err)
	return c
}
