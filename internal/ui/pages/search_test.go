package pages

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

func TestNewSearchPage(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	assert.Equal(t, 80, page.width)
	assert.Equal(t, 40, page.height)
	assert.Equal(t, "test-db", page.databaseID)
	assert.NotNil(t, page.notionClient)
	assert.False(t, page.searching)
	assert.Empty(t, page.results)
	assert.Empty(t, page.query)
}

func TestSearchPageInit(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	cmd := page.Init()
	assert.NotNil(t, cmd)
}

func TestSearchPageUpdate_WindowSize(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := page.Update(msg)
	updatedPage := updatedModel.(*SearchPage)

	assert.Equal(t, 100, updatedPage.width)
	assert.Equal(t, 50, updatedPage.height)
}

func TestSearchPageUpdate_EnterToSearch(t *testing.T) {
	// Mock QueryDatabase to return test pages
	mockResp := &notionapi.DatabaseQueryResponse{
		Results: []notionapi.Page{
			{
				ID: "page-1",
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "Test Page"},
						},
					},
				},
			},
		},
		HasMore: false,
	}

	client := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return mockResp, nil
		},
	}

	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	// Set input value
	page.input.SetValue("test")

	// Press enter to search
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := page.Update(msg)
	updatedPage := updatedModel.(*SearchPage)

	assert.True(t, updatedPage.searching)
	assert.NotNil(t, cmd)

	// Execute the search command
	searchMsg := cmd()
	assert.IsType(t, searchResultsMsg{}, searchMsg)

	resultsMsg := searchMsg.(searchResultsMsg)
	assert.NoError(t, resultsMsg.err)
	assert.Equal(t, "test", resultsMsg.query)
	assert.Len(t, resultsMsg.results, 1)
	assert.Equal(t, "Test Page", resultsMsg.results[0].Title)
	assert.Equal(t, "title", resultsMsg.results[0].MatchType)
}

func TestSearchPageUpdate_SearchResults(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.searching = true

	// Simulate receiving search results
	msg := searchResultsMsg{
		results: []SearchResult{
			{
				PageID:    "page-1",
				Title:     "Found Page",
				Snippet:   "This is a test",
				MatchType: "title",
			},
		},
		query: "test",
		err:   nil,
	}

	updatedModel, _ := page.Update(msg)
	updatedPage := updatedModel.(*SearchPage)

	assert.False(t, updatedPage.searching)
	assert.Len(t, updatedPage.results, 1)
	assert.Equal(t, "test", updatedPage.query)
	assert.NoError(t, updatedPage.err)
}

func TestSearchPageUpdate_SearchError(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.searching = true

	// Simulate receiving search error
	msg := searchResultsMsg{
		err: errors.New("API error"),
	}

	updatedModel, _ := page.Update(msg)
	updatedPage := updatedModel.(*SearchPage)

	assert.False(t, updatedPage.searching)
	assert.Error(t, updatedPage.err)
	assert.Equal(t, "API error", updatedPage.err.Error())
}

func TestSearchPageUpdate_NavigateToResult(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	// Set up results
	page.results = []SearchResult{
		{
			PageID:    "page-1",
			Title:     "Test Page",
			Snippet:   "snippet",
			MatchType: "title",
		},
	}
	page.updateResultsList()

	// Blur input to focus on results list
	page.input.Blur()

	// Press enter to navigate
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := page.Update(msg)

	assert.NotNil(t, cmd)

	// Execute the navigation command
	navMsg := cmd()
	assert.IsType(t, NavigationMsg{}, navMsg)

	navigationMsg := navMsg.(NavigationMsg)
	assert.Equal(t, "page-1", navigationMsg.PageID())
}

func TestSearchPageUpdate_TabToggleFocus(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	// Initially input is focused
	assert.True(t, page.input.Focused())

	// Press tab to blur input
	msg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := page.Update(msg)
	updatedPage := updatedModel.(*SearchPage)

	assert.False(t, updatedPage.input.Focused())

	// Press tab again to focus input
	updatedModel, _ = updatedPage.Update(msg)
	updatedPage = updatedModel.(*SearchPage)

	assert.True(t, updatedPage.input.Focused())
}

func TestSearchPage_SearchCmd_NilClient(t *testing.T) {
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: nil,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.input.SetValue("test")

	cmd := page.searchCmd()
	msg := cmd()

	assert.IsType(t, searchResultsMsg{}, msg)
	resultsMsg := msg.(searchResultsMsg)
	assert.Error(t, resultsMsg.err)
	assert.Contains(t, resultsMsg.err.Error(), "not initialized")
}

func TestSearchPage_SearchCmd_APIError(t *testing.T) {
	client := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return nil, errors.New("API error")
		},
	}

	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.input.SetValue("test")

	cmd := page.searchCmd()
	msg := cmd()

	assert.IsType(t, searchResultsMsg{}, msg)
	resultsMsg := msg.(searchResultsMsg)
	assert.Error(t, resultsMsg.err)
	assert.Contains(t, resultsMsg.err.Error(), "fetch pages")
}

func TestSearchPage_SearchCmd_MatchTitle(t *testing.T) {
	mockResp := &notionapi.DatabaseQueryResponse{
		Results: []notionapi.Page{
			{
				ID: "page-1",
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "Testing Page"},
						},
					},
				},
			},
			{
				ID: "page-2",
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "Another Page"},
						},
					},
				},
			},
		},
		HasMore: false,
	}

	client := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return mockResp, nil
		},
	}

	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.input.SetValue("testing")

	cmd := page.searchCmd()
	msg := cmd()

	resultsMsg := msg.(searchResultsMsg)
	assert.NoError(t, resultsMsg.err)
	assert.Len(t, resultsMsg.results, 1)
	assert.Equal(t, "Testing Page", resultsMsg.results[0].Title)
	assert.Equal(t, "page-1", resultsMsg.results[0].PageID)
	assert.Equal(t, "title", resultsMsg.results[0].MatchType)
}

func TestSearchPage_SearchCmd_MatchStatus(t *testing.T) {
	mockResp := &notionapi.DatabaseQueryResponse{
		Results: []notionapi.Page{
			{
				ID: "page-1",
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "My Page"},
						},
					},
					"Status": &notionapi.StatusProperty{
						Status: notionapi.Status{
							Name: "In Progress",
						},
					},
				},
			},
		},
		HasMore: false,
	}

	client := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return mockResp, nil
		},
	}

	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.input.SetValue("progress")

	cmd := page.searchCmd()
	msg := cmd()

	resultsMsg := msg.(searchResultsMsg)
	assert.NoError(t, resultsMsg.err)
	assert.Len(t, resultsMsg.results, 1)
	assert.Equal(t, "My Page", resultsMsg.results[0].Title)
	assert.Equal(t, "property", resultsMsg.results[0].MatchType)
	assert.Contains(t, resultsMsg.results[0].Snippet, "In Progress")
}

func TestSearchPage_SearchCmd_NoResults(t *testing.T) {
	mockResp := &notionapi.DatabaseQueryResponse{
		Results: []notionapi.Page{
			{
				ID: "page-1",
				Properties: notionapi.Properties{
					"Name": &notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{PlainText: "Test Page"},
						},
					},
				},
			},
		},
		HasMore: false,
	}

	client := &MockNotionClient{
		QueryDatabaseFunc: func(ctx context.Context, id string, req *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
			return mockResp, nil
		},
	}

	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.input.SetValue("nonexistent")

	cmd := page.searchCmd()
	msg := cmd()

	resultsMsg := msg.(searchResultsMsg)
	assert.NoError(t, resultsMsg.err)
	assert.Empty(t, resultsMsg.results)
}

func TestSearchPage_GenerateSnippet(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	tests := []struct {
		name          string
		text          string
		query         string
		shouldContain string // What the snippet should contain (accounting for case)
	}{
		{
			name:          "short text",
			text:          "Hello World",
			query:         "world",
			shouldContain: "World", // Original case preserved
		},
		{
			name:          "query at start",
			text:          "Testing is important for software quality and reliability",
			query:         "testing",
			shouldContain: "Testing", // Original case preserved
		},
		{
			name:          "query in middle",
			text:          "This is a very long text with the word important somewhere in the middle of it",
			query:         "important",
			shouldContain: "important",
		},
		{
			name:          "query at end",
			text:          "This is a text that ends with search",
			query:         "search",
			shouldContain: "search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snippet := page.generateSnippet(tt.text, tt.query)
			assert.Contains(t, snippet, tt.shouldContain)
		})
	}
}

func TestSearchPage_UpdateResultsList(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.results = []SearchResult{
		{PageID: "1", Title: "Page 1", Snippet: "snippet 1", MatchType: "title"},
		{PageID: "2", Title: "Page 2", Snippet: "snippet 2", MatchType: "property"},
	}

	page.updateResultsList()

	items := page.resultsList.Items()
	assert.Len(t, items, 2)
	assert.Equal(t, "Search Results (2)", page.resultsList.Title)
}

func TestSearchPage_View(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	// Test initial state
	view := page.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Search Pages")

	// Test with results
	page.results = []SearchResult{
		{PageID: "1", Title: "Test Page", Snippet: "snippet", MatchType: "title"},
	}
	page.query = "test"
	page.updateResultsList()

	view = page.View()
	assert.NotEmpty(t, view)

	// Test no results state
	page.results = []SearchResult{}
	page.query = "nosuchpage"
	page.updateResultsList()

	view = page.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No results found")

	// Test error state
	page.err = errors.New("test error")
	view = page.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Error")
}

func TestSearchPage_Getters(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	page.results = []SearchResult{{PageID: "1", Title: "Test"}}
	page.query = "test query"
	page.searching = true

	assert.Len(t, page.Results(), 1)
	assert.Equal(t, "test query", page.Query())
	assert.True(t, page.IsSearching())
}

func TestSearchResultItem_Methods(t *testing.T) {
	result := SearchResult{
		PageID:    "page-1",
		Title:     "Test Page",
		Snippet:   "This is a snippet",
		MatchType: "title",
	}

	item := searchResultItem{result: result}

	assert.Equal(t, "Test Page", item.Title())
	assert.Contains(t, item.Description(), "title")
	assert.Contains(t, item.Description(), "This is a snippet")
	assert.Equal(t, "Test Page", item.FilterValue())
}

// MockNotionClient is already defined in detail_test.go
// We'll use the same mock interface

// Ensure the client interface is properly implemented
func TestSearchPage_ClientInterface(t *testing.T) {
	client := &MockNotionClient{}
	page := NewSearchPage(NewSearchPageInput{
		Width:        80,
		Height:       40,
		NotionClient: client,
		Cache:        nil,
		DatabaseID:   "test-db",
	})

	assert.NotNil(t, page.notionClient)
}
