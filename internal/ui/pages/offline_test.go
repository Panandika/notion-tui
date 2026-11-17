package pages

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/testhelpers"
)

// errNoNetwork is what a call returns when Notion cannot be reached.
var errNoNetwork = &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection refused")}

// offlineFixture opens a page once with a connection, so a copy is saved,
// and returns what is needed to open it again.
func offlineFixture(t *testing.T) (*testhelpers.MockNotionClient, func() (*DetailPage, *mockViewer)) {
	t.Helper()
	pageCache := mustCreateCache(t)
	client := testhelpers.NewMockNotionClient()
	client.PageToReturn = testhelpers.NewTestPage("page-1", "Budget 2026")
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "- [ ] saved content"}
	open := func() (*DetailPage, *mockViewer) {
		viewer := newMockViewer()
		dp := NewDetailPage(NewDetailPageInput{Width: 100, Height: 24, Viewer: viewer, NotionClient: client, Cache: pageCache, PageID: "page-1"})
		dp.Update(dp.Init()())
		return &dp, viewer
	}
	_, viewer := open()
	require.Contains(t, viewer.markdown, "saved content")
	return client, open
}

func TestDetailPage_OfflineShowsTheSavedCopy(t *testing.T) {
	t.Parallel()

	client, open := offlineFixture(t)
	client.ErrorToReturn = errNoNetwork

	dp, viewer := open()

	require.NoError(t, dp.Error(), "a saved copy is not an error")
	assert.True(t, dp.Offline())
	assert.Contains(t, viewer.markdown, "saved content")
	assert.Contains(t, viewer.markdown, "Budget 2026", "the title was saved with the content")
	assert.Equal(t, "Budget 2026", dp.Title())
	view := dp.View()
	assert.Contains(t, view, "OFFLINE")
	assert.Contains(t, view, "Offline: this is the copy saved")
}

func TestDetailPage_OfflineCopyCannotBeChanged(t *testing.T) {
	t.Parallel()

	client, open := offlineFixture(t)
	client.ErrorToReturn = errNoNetwork
	dp, _ := open()

	for _, key := range []rune{'e', 't', 'p', 'D'} {
		_, cmd := dp.Update(keyRune(key))
		assert.Nil(t, cmd, "%c starts nothing", key)
		assert.False(t, dp.InMode(), "%c opens nothing", key)
		assert.Contains(t, dp.View(), "Offline copy: reload with r", "%c", key)
	}
}

func TestDetailPage_ReloadLeavesOfflineWhenNotionIsBack(t *testing.T) {
	t.Parallel()

	client, open := offlineFixture(t)
	client.ErrorToReturn = errNoNetwork
	dp, _ := open()
	require.True(t, dp.Offline())

	// Still no connection: the copy stays on screen
	dp.Update(dp.Refresh()())
	assert.True(t, dp.Offline())
	require.NoError(t, dp.Error())

	client.ErrorToReturn = nil
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "newer content"}
	dp.Update(dp.Refresh()())

	assert.False(t, dp.Offline())
	assert.Equal(t, "newer content", dp.Markdown())
	assert.NotContains(t, dp.View(), "OFFLINE")
	_, cmd := dp.Update(keyRune('t'))
	_ = cmd
	assert.NotContains(t, dp.View(), "Offline copy", "changes are possible again")
}

func TestDetailPage_OnlyAMissingConnectionFallsBackToTheCopy(t *testing.T) {
	t.Parallel()

	t.Run("an answer from Notion that is an error stays an error", func(t *testing.T) {
		t.Parallel()
		client, open := offlineFixture(t)
		client.ErrorToReturn = errors.New("object_not_found: the page was deleted")

		dp, _ := open()

		assert.Error(t, dp.Error())
		assert.False(t, dp.Offline())
	})

	t.Run("no saved copy, no connection: the error is shown", func(t *testing.T) {
		t.Parallel()
		client := testhelpers.NewMockNotionClient()
		client.ErrorToReturn = errNoNetwork
		dp := NewDetailPage(NewDetailPageInput{Width: 100, Height: 24, Viewer: newMockViewer(), NotionClient: client, Cache: mustCreateCache(t), PageID: "never-opened"})

		dp.Update(dp.Init()())

		assert.Error(t, dp.Error())
		assert.False(t, dp.Offline())
	})
}

func TestListPage_OfflineShowsTheSavedRows(t *testing.T) {
	t.Parallel()

	pageCache, err := cache.NewPageCache(cache.NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)
	online := true
	client := &MockNotionClient{
		QueryDatabaseFunc: func(context.Context, string, *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			if !online {
				return nil, errNoNetwork
			}
			return &notionapi.DatabaseQueryResponse{Results: []notionapi.Page{newTestNotionPage("row-1", "Saved row", "Done")}}, nil
		},
		GetDatabaseFunc: func(context.Context, string) (*notionapi.Database, error) {
			return &notionapi.Database{Properties: notionapi.PropertyConfigs{"Done": &notionapi.CheckboxPropertyConfig{}}}, nil
		},
	}
	open := func() *ListPage {
		lp := NewListPage(NewListPageInput{Width: 110, Height: 30, NotionClient: client, Cache: pageCache, DatabaseID: "db-1"})
		feed(&lp, lp.fetchPagesCmd())
		return &lp
	}

	first := open()
	require.Contains(t, first.View(), "Saved row")

	online = false
	lp := open()

	view := lp.View()
	assert.Contains(t, view, "Saved row")
	assert.Contains(t, view, "OFFLINE")
	assert.Contains(t, view, "Offline: these are the rows saved")
	require.NoError(t, lp.err)

	t.Run("a filtered list is not answered from the unfiltered copy", func(t *testing.T) {
		online = true
		_, schema := lp.Update(listKey('p'))
		feed(lp, schema)
		lp.Update(listEnter())
		online = false
		_, filtered := lp.Update(listEnter())
		feed(lp, filtered)

		assert.Error(t, lp.err, "the saved rows are not the rows that match")
	})
}
