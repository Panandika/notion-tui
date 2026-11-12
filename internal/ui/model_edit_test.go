package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/pages"
)

func newEditTestModel(t *testing.T) AppModel {
	t.Helper()
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
		CacheDir:    t.TempDir(),
	}
	model := NewModel(NewModelInput{Config: cfg, Cache: nil})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return updated.(AppModel)
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestModel_EditBlockMsgOpensEditor(t *testing.T) {
	t.Parallel()

	m := newEditTestModel(t)
	m.navigateToDetail("page-1")
	m.sidebarFocus = true

	updated, cmd := m.Update(pages.EditBlockMsg{PageID: "page-1", BlockID: "block-1"})
	m = updated.(AppModel)

	assert.Equal(t, PageEdit, m.currentPage)
	assert.NotNil(t, m.pages[PageEdit])
	assert.NotNil(t, cmd, "editor must start loading the block")
	assert.False(t, m.sidebarFocus, "editor needs keyboard focus")
	assert.Equal(t, ViewModeEdit, m.mode)
}

func TestModel_EditorKeysDoNotTriggerGlobalShortcuts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		key      tea.KeyMsg
		wantQuit bool
	}{
		{name: "q is text, not quit", key: keyRune('q'), wantQuit: false},
		{name: "question mark is text, not help", key: keyRune('?'), wantQuit: false},
		{name: "tab stays in the editor", key: tea.KeyMsg{Type: tea.KeyTab}, wantQuit: false},
		{name: "ctrl+c still quits", key: tea.KeyMsg{Type: tea.KeyCtrlC}, wantQuit: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newEditTestModel(t)
			m.navigateToDetail("page-1")
			updated, _ := m.Update(pages.EditBlockMsg{PageID: "page-1", BlockID: "block-1"})
			m = updated.(AppModel)

			updated, cmd := m.Update(tt.key)
			m = updated.(AppModel)

			assert.Equal(t, tt.wantQuit, isQuit(cmd))
			assert.Equal(t, PageEdit, m.currentPage)
			assert.False(t, m.sidebarFocus)
			assert.False(t, m.showHelp)
		})
	}
}

func TestModel_EditFinishedReturnsToDetail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		saved       bool
		wantRefresh bool
	}{
		{name: "nothing saved keeps the page as is", saved: false, wantRefresh: false},
		{name: "saved change reloads the page", saved: true, wantRefresh: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newEditTestModel(t)
			m.navigateToDetail("page-1")
			updated, _ := m.Update(pages.EditBlockMsg{PageID: "page-1", BlockID: "block-1"})
			m = updated.(AppModel)

			updated, cmd := m.Update(pages.EditFinishedMsg{PageID: "page-1", Saved: tt.saved})
			m = updated.(AppModel)

			require.Equal(t, PageDetail, m.currentPage)
			assert.Nil(t, m.pages[PageEdit], "finished editor must be discarded")
			assert.Equal(t, ViewModeBrowse, m.mode)
			assert.Equal(t, tt.wantRefresh, cmd != nil)
			assert.False(t, isQuit(cmd), "leaving the editor must not quit the app")
		})
	}
}

func TestModel_SidebarFocusKeepsKeysAwayFromPage(t *testing.T) {
	t.Parallel()

	m := newEditTestModel(t)
	m.navigateToDetail("page-1")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	require.True(t, m.sidebarFocus)

	// "r" would refresh the detail page if it leaked through to the page.
	_, cmd := m.Update(keyRune('r'))

	assert.Nil(t, cmd)
}

func TestModel_PaletteTypingDoesNotQuit(t *testing.T) {
	t.Parallel()

	m := newEditTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(AppModel)
	require.True(t, m.showPalette)

	_, cmd := m.Update(keyRune('q'))

	assert.False(t, isQuit(cmd))
}

func TestModel_PaletteStaysClosedWhileEditing(t *testing.T) {
	t.Parallel()

	m := newEditTestModel(t)
	m.navigateToDetail("page-1")
	updated, _ := m.Update(pages.EditBlockMsg{PageID: "page-1", BlockID: "block-1"})
	m = updated.(AppModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(AppModel)

	assert.False(t, m.showPalette, "palette navigation would drop unsaved text")
	assert.Equal(t, PageEdit, m.currentPage)
}
