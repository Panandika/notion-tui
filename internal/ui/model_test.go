package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/pages"
)

func TestNewModel(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		DefaultDatabase: "test_db_id",
		Debug:           false,
		CacheDir:        "/tmp/test-cache",
	}

	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	assert.Equal(t, "test_token", model.config.NotionToken)
	assert.Equal(t, "test_db_id", model.config.DatabaseID)
	assert.NotNil(t, model.notionClient)
	assert.NotNil(t, model.cache)
	assert.NotNil(t, model.navigator)
	assert.Equal(t, PageList, model.currentPage)
	assert.False(t, model.ready)
	assert.True(t, model.showSidebar)
	assert.False(t, model.showPalette)
	assert.Equal(t, ViewModeBrowse, model.mode)
}

func TestNewModel_NoDatabases(t *testing.T) {
	// Test model creation without databases - should start with workspace search
	cfg := &config.Config{
		NotionToken: "test_token",
		Debug:       false,
		CacheDir:    "/tmp/test-cache",
	}

	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	assert.Equal(t, PageWorkspaceSearch, model.currentPage)
	assert.False(t, model.showSidebar)
	assert.NotNil(t, model.notionClient)
}

func TestModelUpdate_WindowSize(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Send window size message
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(windowMsg)
	m := updatedModel.(AppModel)

	assert.Equal(t, 80, m.width)
	assert.Equal(t, 24, m.height)
	assert.True(t, m.ready)
}

func TestModelUpdate_GlobalKeys(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Test quit with 'q'
	qMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(qMsg)

	// Should return a quit command
	assert.NotNil(t, cmd)

	// Test ctrl+c
	ctrlCMsg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd = model.Update(ctrlCMsg)
	assert.NotNil(t, cmd)
}

func TestModelUpdate_ToggleSidebar(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})
	model.ready = true

	// Initially sidebar is shown
	assert.True(t, model.showSidebar)

	// Toggle sidebar with Tab
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ := model.Update(tabMsg)
	m := updatedModel.(AppModel)
	assert.False(t, m.showSidebar)

	// Toggle again
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(AppModel)
	assert.True(t, m.showSidebar)
}

func TestModelUpdate_TogglePalette(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})
	model.ready = true

	// Initially palette is hidden
	assert.False(t, model.showPalette)

	// Toggle palette with Ctrl+P
	ctrlPMsg := tea.KeyMsg{Type: tea.KeyCtrlP}
	updatedModel, _ := model.Update(ctrlPMsg)
	m := updatedModel.(AppModel)
	assert.True(t, m.showPalette)

	// Toggle again
	updatedModel, _ = m.Update(ctrlPMsg)
	m = updatedModel.(AppModel)
	assert.False(t, m.showPalette)
}

func TestModelView(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Before ready
	view := model.View()
	assert.Contains(t, view, "Initializing")

	// After ready (need to set dimensions)
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(windowMsg)
	m := updatedModel.(AppModel)

	view = m.View()
	assert.NotEmpty(t, view)
}

func TestModelInit(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Init should return a command
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestModelPageRegistry(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Initialize pages
	model.initializePages()

	// Check that ListPage is registered
	assert.NotNil(t, model.pages[PageList])

	// Current page should be ListPage
	assert.Equal(t, PageList, model.currentPage)
}

func TestModelNavigation(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})
	model.width = 80
	model.height = 24
	model.ready = true

	// Check initial state
	assert.Equal(t, PageList, model.currentPage)
	assert.False(t, model.navigator.CanGoBack())

	// Navigate to detail page
	model.navigateToDetail("test-page-id")
	assert.Equal(t, PageDetail, model.currentPage)
	assert.True(t, model.navigator.CanGoBack())

	// Navigate back
	model.goBack()
	assert.Equal(t, PageList, model.currentPage)
	assert.False(t, model.navigator.CanGoBack())
}

func TestModelNavigator(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	nav := model.Navigator()
	assert.NotNil(t, nav)
	assert.Equal(t, PageList, nav.CurrentPage())
}

func TestModelCurrentPage(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	assert.Equal(t, PageList, model.CurrentPage())
}

func TestModelFindPageByID(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	// Add some test pages
	model.pageList = []pages.Page{
		{ID: "page-1", Title: "Test Page 1"},
		{ID: "page-2", Title: "Test Page 2"},
		{ID: "page-3", Title: "Test Page 3"},
	}

	// Find existing page
	found := model.findPageByID("page-2")
	assert.NotNil(t, found)
	assert.Equal(t, "Test Page 2", found.Title)

	// Find non-existing page
	notFound := model.findPageByID("page-99")
	assert.Nil(t, notFound)
}

func TestModelSetPageList(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})

	pagesList := []pages.Page{
		{ID: "page-1", Title: "Test Page 1"},
		{ID: "page-2", Title: "Test Page 2"},
	}

	model.SetPageList(pagesList)
	assert.Equal(t, 2, len(model.pageList))
	assert.Equal(t, "Test Page 1", model.pageList[0].Title)
}

func TestModelNavigationMsg(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})
	model.width = 80
	model.height = 24
	model.ready = true

	// Create navigation message
	navMsg := pages.NewNavigationMsg("test-page-id")

	updatedModel, _ := model.Update(navMsg)
	m := updatedModel.(AppModel)

	// Should have navigated to the page
	assert.Equal(t, PageID("test-page-id"), m.currentPage)
}

func TestModelBackNavigation(t *testing.T) {
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases: []config.DatabaseConfig{
			{ID: "test_db_id", Name: "Test DB"},
		},
		CacheDir: "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{
		Config: cfg,
		Cache:  nil,
	})
	model.width = 80
	model.height = 24
	model.ready = true

	// Navigate to a page
	model.navigateTo(PageDetail)
	assert.Equal(t, PageDetail, model.currentPage)

	// Press Esc to go back
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ := model.Update(escMsg)
	m := updatedModel.(AppModel)

	// Should have navigated back to list
	assert.Equal(t, PageList, m.currentPage)
}
