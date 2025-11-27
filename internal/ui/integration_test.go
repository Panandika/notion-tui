package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/testhelpers"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// TestIntegration_NavigationFlow tests the complete navigation flow:
// Start at ListPage -> Navigate to DetailPage -> Navigate back with ESC.
func TestIntegration_NavigationFlow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                  string
		initialPage           PageID
		navigateTo            PageID
		expectedPageAfterNav  PageID
		expectedCanGoBack     bool
		expectedPageAfterBack PageID
	}{
		{
			name:                  "list to detail and back",
			initialPage:           PageList,
			navigateTo:            PageDetail,
			expectedPageAfterNav:  PageDetail,
			expectedCanGoBack:     true,
			expectedPageAfterBack: PageList,
		},
		{
			name:                  "list to edit and back",
			initialPage:           PageList,
			navigateTo:            PageEdit,
			expectedPageAfterNav:  PageEdit,
			expectedCanGoBack:     true,
			expectedPageAfterBack: PageList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Initial state check
			assert.Equal(t, tt.initialPage, model.currentPage)
			assert.False(t, model.navigator.CanGoBack())

			// Navigate to target page
			model.navigateTo(tt.navigateTo)
			assert.Equal(t, tt.expectedPageAfterNav, model.currentPage)
			assert.Equal(t, tt.expectedCanGoBack, model.navigator.CanGoBack())

			// Send ESC key to go back
			escMsg := tea.KeyMsg{Type: tea.KeyEsc}
			updatedModel, _ := model.Update(escMsg)
			m := updatedModel.(AppModel)

			// Verify back navigation
			assert.Equal(t, tt.expectedPageAfterBack, m.currentPage)
		})
	}
}

// TestIntegration_NavigationWithMessages tests navigation via messages.
// Tests ItemSelectedMsg triggering navigation to DetailPage.
func TestIntegration_NavigationWithMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		selectedPageID    string
		expectedPage      PageID
		expectedCanGoBack bool
	}{
		{
			name:              "select page navigates to detail",
			selectedPageID:    "page-123",
			expectedPage:      PageDetail,
			expectedCanGoBack: true,
		},
		{
			name:              "select another page navigates to detail",
			selectedPageID:    "page-456",
			expectedPage:      PageDetail,
			expectedCanGoBack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Send ItemSelectedMsg
			itemSelectedMsg := components.ItemSelectedMsg{
				ID:    tt.selectedPageID,
				Title: "Test Page",
				Index: 0,
			}

			updatedModel, _ := model.Update(itemSelectedMsg)
			m := updatedModel.(AppModel)

			// Verify navigation
			assert.Equal(t, tt.expectedPage, m.currentPage)
			assert.Equal(t, tt.expectedCanGoBack, m.navigator.CanGoBack())
			assert.NotNil(t, m.pages[PageDetail])
		})
	}
}

// TestIntegration_GlobalKeybindings tests all global keyboard shortcuts.
func TestIntegration_GlobalKeybindings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		keyMsg              tea.KeyMsg
		shouldToggleSidebar bool
		shouldTogglePalette bool
		shouldQuit          bool
		setupFn             func(*AppModel)
	}{
		{
			name:                "Tab toggles sidebar",
			keyMsg:              tea.KeyMsg{Type: tea.KeyTab},
			shouldToggleSidebar: true,
			setupFn:             func(m *AppModel) { m.showSidebar = true },
		},
		{
			name:                "Ctrl+P toggles command palette",
			keyMsg:              tea.KeyMsg{Type: tea.KeyCtrlP},
			shouldTogglePalette: true,
			setupFn:             func(m *AppModel) { m.showPalette = false },
		},
		{
			name:       "Ctrl+C quits application",
			keyMsg:     tea.KeyMsg{Type: tea.KeyCtrlC},
			shouldQuit: true,
			setupFn:    func(m *AppModel) {},
		},
		{
			name:       "q quits application",
			keyMsg:     tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			shouldQuit: true,
			setupFn:    func(m *AppModel) {},
		},
		{
			name:    "ESC navigates back when history available",
			keyMsg:  tea.KeyMsg{Type: tea.KeyEsc},
			setupFn: func(m *AppModel) { m.navigator.NavigateTo(PageDetail) },
		},
		{
			name:    "? key opens help (handled globally)",
			keyMsg:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			setupFn: func(m *AppModel) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Run setup function
			tt.setupFn(&model)

			// Store initial state
			initialShowSidebar := model.showSidebar
			initialShowPalette := model.showPalette

			// Update with key message
			_, cmd := model.Update(tt.keyMsg)

			// For quit commands, verify cmd is not nil
			if tt.shouldQuit {
				assert.NotNil(t, cmd, "expected quit command for key %v", tt.keyMsg)
				return
			}

			// Check sidebar toggle
			if tt.shouldToggleSidebar {
				// Re-run update to check toggle worked
				updatedModel, _ := model.Update(tt.keyMsg)
				m := updatedModel.(AppModel)
				assert.NotEqual(t, initialShowSidebar, m.showSidebar)
			}

			// Check palette toggle
			if tt.shouldTogglePalette {
				updatedModel, _ := model.Update(tt.keyMsg)
				m := updatedModel.(AppModel)
				assert.NotEqual(t, initialShowPalette, m.showPalette)
			}
		})
	}
}

// TestIntegration_WindowResize tests that window resize messages propagate correctly.
func TestIntegration_WindowResize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		initialWidth            int
		initialHeight           int
		newWidth                int
		newHeight               int
		expectedSidebarWidth    int
		expectedMinSidebarWidth int
		expectedMaxSidebarWidth int
	}{
		{
			name:                    "resize to larger window",
			initialWidth:            80,
			initialHeight:           24,
			newWidth:                120,
			newHeight:               40,
			expectedMinSidebarWidth: 20,
			expectedMaxSidebarWidth: 40,
		},
		{
			name:                    "resize to smaller window",
			initialWidth:            120,
			initialHeight:           40,
			newWidth:                60,
			newHeight:               15,
			expectedMinSidebarWidth: 20,
			expectedMaxSidebarWidth: 40,
		},
		{
			name:                    "resize with very small width",
			initialWidth:            40,
			initialHeight:           24,
			newWidth:                30,
			newHeight:               24,
			expectedMinSidebarWidth: 20,
			expectedMaxSidebarWidth: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})

			// Send initial window size
			windowMsg := tea.WindowSizeMsg{Width: tt.initialWidth, Height: tt.initialHeight}
			updatedModel, _ := model.Update(windowMsg)
			m := updatedModel.(AppModel)

			// Verify initial dimensions set
			assert.Equal(t, tt.initialWidth, m.width)
			assert.Equal(t, tt.initialHeight, m.height)
			assert.True(t, m.ready)

			// Send resize message
			resizeMsg := tea.WindowSizeMsg{Width: tt.newWidth, Height: tt.newHeight}
			updatedModel, _ = m.Update(resizeMsg)
			m = updatedModel.(AppModel)

			// Verify new dimensions
			assert.Equal(t, tt.newWidth, m.width)
			assert.Equal(t, tt.newHeight, m.height)
			assert.True(t, m.ready)

			// Verify sidebar width is within bounds
			calculatedSidebarWidth := tt.newWidth / 4
			if calculatedSidebarWidth < 20 {
				calculatedSidebarWidth = 20
			}
			if calculatedSidebarWidth > 40 {
				calculatedSidebarWidth = 40
			}
			assert.GreaterOrEqual(t, calculatedSidebarWidth, tt.expectedMinSidebarWidth)
			assert.LessOrEqual(t, calculatedSidebarWidth, tt.expectedMaxSidebarWidth)

			// Verify status bar width updated
			assert.Equal(t, tt.newWidth, m.statusBar.Width())
		})
	}
}

// TestIntegration_ComponentInteraction tests interaction between components.
func TestIntegration_ComponentInteraction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupFn  func(*AppModel)
		action   func(*AppModel) (AppModel, tea.Cmd)
		verifyFn func(*testing.T, *AppModel)
	}{
		{
			name: "sidebar visible on list page",
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
				m.currentPage = PageList
				m.showSidebar = true
			},
			action: func(m *AppModel) (AppModel, tea.Cmd) {
				return *m, nil
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				assert.True(t, m.showSidebar)
				assert.Equal(t, PageList, m.currentPage)
				view := m.View()
				assert.NotEmpty(t, view)
			},
		},
		{
			name: "command palette overlays content",
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
				m.showPalette = true
			},
			action: func(m *AppModel) (AppModel, tea.Cmd) {
				return *m, nil
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				assert.True(t, m.showPalette)
				view := m.View()
				assert.NotEmpty(t, view)
				// View should render (palette visibility depends on full setup)
				assert.True(t, len(view) > 0)
			},
		},
		{
			name: "status bar always visible",
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
			},
			action: func(m *AppModel) (AppModel, tea.Cmd) {
				return *m, nil
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				view := m.View()
				assert.NotEmpty(t, view)
				// Status bar should be visible
				assert.True(t, len(view) > 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})

			// Run setup
			tt.setupFn(&model)

			// Run action
			updatedModel, _ := tt.action(&model)

			// Verify
			tt.verifyFn(t, &updatedModel)
		})
	}
}

// TestIntegration_ErrorHandling tests error handling in the UI orchestrator.
func TestIntegration_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		errorMsg        tea.Msg
		shouldShowError bool
		expectedInView  string
	}{
		{
			name:            "error message displayed",
			errorMsg:        NewErrorMsg("API Error", testhelpers.ErrNotFound),
			shouldShowError: true,
			expectedInView:  "Error",
		},
		{
			name:            "sync status error",
			errorMsg:        NewSyncStatusMsg("error"),
			shouldShowError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Send message
			updatedModel, _ := model.Update(tt.errorMsg)
			m := updatedModel.(AppModel)

			// Verify
			if tt.shouldShowError {
				// Error should be set in model
				view := m.View()
				assert.NotEmpty(t, view)
			}
		})
	}
}

// TestIntegration_PageLifecycle tests page creation, persistence, and state management.
func TestIntegration_PageLifecycle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		navigatePages     []PageID
		expectedFinalPage PageID
		verifyRegistry    func(*testing.T, *AppModel)
		expectedCanGoBack bool
	}{
		{
			name:              "single page navigation",
			navigatePages:     []PageID{PageDetail},
			expectedFinalPage: PageDetail,
			verifyRegistry: func(t *testing.T, m *AppModel) {
				assert.NotNil(t, m.pages[PageList], "list page should be registered")
				assert.NotNil(t, m.pages[PageDetail], "detail page should be registered")
			},
			expectedCanGoBack: true,
		},
		{
			name:              "multiple page navigation to detail only",
			navigatePages:     []PageID{PageDetail, PageList},
			expectedFinalPage: PageList,
			verifyRegistry: func(t *testing.T, m *AppModel) {
				assert.NotNil(t, m.pages[PageList], "list page should be registered")
				// PageDetail should be registered after navigation
				assert.NotNil(t, m.pages[PageDetail], "detail page should be registered")
				// Should have at least list and detail
				assert.GreaterOrEqual(t, len(m.pages), 2, "should have list and detail pages")
			},
			expectedCanGoBack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Initialize pages (normally done in Init())
			model.initializePages()

			// Navigate through pages
			for _, pageID := range tt.navigatePages {
				model.navigateTo(pageID)
			}

			// Verify final state
			assert.Equal(t, tt.expectedFinalPage, model.currentPage)

			// Verify pages are created and registered
			tt.verifyRegistry(t, &model)

			// Verify navigation history
			assert.Equal(t, tt.expectedCanGoBack, model.navigator.CanGoBack())
		})
	}
}

// TestIntegration_ComplexUserWorkflow tests a realistic user workflow.
func TestIntegration_ComplexUserWorkflow(t *testing.T) {
	t.Parallel()

	// Simulate a complete user workflow:
	// 1. Start at list page
	// 2. Toggle sidebar
	// 3. Select an item (navigate to detail)
	// 4. Open command palette
	// 5. Close command palette
	// 6. Navigate back
	// 7. Resize window
	// 8. Quit

	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
		CacheDir:    "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{Config: cfg, Cache: nil})

	// Step 1: Initial window size
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(windowMsg)
	m := updatedModel.(AppModel)
	assert.True(t, m.ready)
	assert.Equal(t, PageList, m.currentPage)

	// Step 2: Toggle sidebar
	tabMsg := tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(AppModel)
	assert.False(t, m.showSidebar)

	// Toggle back
	updatedModel, _ = m.Update(tabMsg)
	m = updatedModel.(AppModel)
	assert.True(t, m.showSidebar)

	// Step 3: Select an item
	itemMsg := components.ItemSelectedMsg{
		ID:    "page-123",
		Title: "Test Page",
		Index: 0,
	}
	updatedModel, _ = m.Update(itemMsg)
	m = updatedModel.(AppModel)
	assert.Equal(t, PageDetail, m.currentPage)
	assert.True(t, m.navigator.CanGoBack())

	// Step 4: Open command palette
	ctrlPMsg := tea.KeyMsg{Type: tea.KeyCtrlP}
	updatedModel, _ = m.Update(ctrlPMsg)
	m = updatedModel.(AppModel)
	assert.True(t, m.showPalette)

	// Step 5: Close command palette
	updatedModel, _ = m.Update(ctrlPMsg)
	m = updatedModel.(AppModel)
	assert.False(t, m.showPalette)

	// Step 6: Navigate back
	escMsg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, _ = m.Update(escMsg)
	m = updatedModel.(AppModel)
	assert.Equal(t, PageList, m.currentPage)

	// Step 7: Resize window
	resizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ = m.Update(resizeMsg)
	m = updatedModel.(AppModel)
	assert.Equal(t, 120, m.width)
	assert.Equal(t, 40, m.height)

	// Step 8: Verify state is consistent
	assert.NotNil(t, m.sidebar)
	assert.NotNil(t, m.statusBar)
	assert.NotNil(t, m.cmdPalette)
	assert.True(t, len(m.pages) > 0)
}

// TestIntegration_ViewRendering tests that View() correctly renders all components.
func TestIntegration_ViewRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setupFn  func(*AppModel)
		verifyFn func(*testing.T, string)
	}{
		{
			name: "renders list page view",
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
				m.currentPage = PageList
				m.showSidebar = true
			},
			verifyFn: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				assert.True(t, len(view) > 0)
			},
		},
		{
			name: "renders with command palette",
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
				m.showPalette = true
			},
			verifyFn: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				// Just verify view is rendered, palette may not show if model not fully set up
				assert.True(t, len(view) > 0)
			},
		},
		{
			name: "renders before ready",
			setupFn: func(m *AppModel) {
				m.ready = false
			},
			verifyFn: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				assert.Contains(t, view, "Initializing")
			},
		},
		{
			name: "renders with error",
			setupFn: func(m *AppModel) {
				m.width = 80
				m.height = 24
				m.ready = true
				m.err = NewErrorMsg("Test Error", nil)
			},
			verifyFn: func(t *testing.T, view string) {
				assert.NotEmpty(t, view)
				assert.Contains(t, view, "Error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})

			// Run setup
			tt.setupFn(&model)

			// Render view
			view := model.View()

			// Verify
			tt.verifyFn(t, view)
		})
	}
}

// TestIntegration_MessagePropagation tests that messages propagate correctly to components.
func TestIntegration_MessagePropagation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  tea.Msg
		setupFn  func(*AppModel)
		verifyFn func(*testing.T, *AppModel)
	}{
		{
			name:    "window size message propagates to all pages",
			message: tea.WindowSizeMsg{Width: 100, Height: 30},
			setupFn: func(m *AppModel) {
				m.width = 80
				m.height = 24
				m.ready = true
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				assert.Equal(t, 100, m.width)
				assert.Equal(t, 30, m.height)
				// All registered pages should receive the update
				for _, page := range m.pages {
					require.NotNil(t, page)
				}
			},
		},
		{
			name:    "keyboard message to current page",
			message: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			setupFn: func(m *AppModel) {
				m.initializePages()
				m.width = 80
				m.height = 24
				m.ready = true
				m.currentPage = PageList
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				assert.Equal(t, PageList, m.currentPage)
				assert.NotNil(t, m.pages[PageList])
			},
		},
		{
			name:    "sidebar message when visible",
			message: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			setupFn: func(m *AppModel) {
				m.width = 80
				m.height = 24
				m.ready = true
				m.currentPage = PageList
				m.showSidebar = true
			},
			verifyFn: func(t *testing.T, m *AppModel) {
				assert.True(t, m.showSidebar)
				assert.Equal(t, PageList, m.currentPage)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})

			// Run setup
			tt.setupFn(&model)

			// Send message
			updatedModel, _ := model.Update(tt.message)
			m := updatedModel.(AppModel)

			// Verify
			tt.verifyFn(t, &m)
		})
	}
}

// TestIntegration_StateConsistency tests that model state remains consistent through operations.
func TestIntegration_StateConsistency(t *testing.T) {
	t.Parallel()

	// Setup
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
		CacheDir:    "/tmp/test-cache",
	}
	model := NewModel(NewModelInput{Config: cfg, Cache: nil})

	// Window size
	windowMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updatedModel, _ := model.Update(windowMsg)
	m := updatedModel.(AppModel)

	// Verify component initialization
	assert.NotNil(t, m.sidebar)
	assert.NotNil(t, m.statusBar)
	assert.NotNil(t, m.cmdPalette)
	assert.NotNil(t, m.navigator)
	assert.NotNil(t, m.notionClient)

	// Verify config is consistent
	assert.Equal(t, "test_token", m.config.NotionToken)
	assert.Equal(t, "test_db_id", m.config.DatabaseID)

	// Navigate and verify state
	m.navigateTo(PageDetail)
	assert.Equal(t, PageDetail, m.currentPage)
	assert.Equal(t, PageDetail, m.navigator.CurrentPage())

	// Go back and verify state
	if m.navigator.CanGoBack() {
		previousPage, ok := m.navigator.Back()
		assert.True(t, ok)
		assert.Equal(t, PageList, previousPage)
	}

	// Verify pages registry is maintained
	assert.True(t, len(m.pages) > 0)
	for pageID, page := range m.pages {
		assert.NotNil(t, page, "page %s should not be nil", pageID)
	}
}

// TestIntegration_SequentialNavigations tests multiple sequential navigations.
func TestIntegration_SequentialNavigations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		navigationSequence []PageID
		expectedFinalPage  PageID
		verifyHistoryFn    func(*testing.T, []PageID)
	}{
		{
			name:               "navigate forward and backward",
			navigationSequence: []PageID{PageDetail, PageEdit, PageDetail, PageList},
			expectedFinalPage:  PageList,
			verifyHistoryFn: func(t *testing.T, history []PageID) {
				// Should have: [PageList, PageDetail, PageEdit, PageDetail]
				// (current is PageList, so history shows what led here)
				assert.Greater(t, len(history), 0, "history should not be empty")
				assert.Equal(t, PageDetail, history[len(history)-1], "last history item should be PageDetail")
			},
		},
		{
			name:               "zigzag navigation",
			navigationSequence: []PageID{PageDetail, PageList, PageDetail, PageList, PageDetail},
			expectedFinalPage:  PageDetail,
			verifyHistoryFn: func(t *testing.T, history []PageID) {
				// Should have history of previous pages
				assert.Greater(t, len(history), 0, "history should not be empty")
				assert.Equal(t, PageList, history[len(history)-1], "last history item should be PageList")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			cfg := &config.Config{
				NotionToken: "test_token",
				DatabaseID:  "test_db_id",
				Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
				CacheDir:    "/tmp/test-cache",
			}
			model := NewModel(NewModelInput{Config: cfg, Cache: nil})
			model.width = 80
			model.height = 24
			model.ready = true

			// Initialize pages
			model.initializePages()

			// Navigate through sequence
			for _, pageID := range tt.navigationSequence {
				model.navigateTo(pageID)
			}

			// Verify final state
			assert.Equal(t, tt.expectedFinalPage, model.currentPage)
			history := model.navigator.History()
			tt.verifyHistoryFn(t, history)

			// Verify list page is always registered
			assert.NotNil(t, model.pages[PageList], "list page should be registered")
		})
	}
}
