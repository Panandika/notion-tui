package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/config"
	"github.com/Panandika/notion-tui/internal/ui/components"
	"github.com/Panandika/notion-tui/internal/ui/pages"
)

func newActionTestModel(t *testing.T) AppModel {
	t.Helper()
	cfg := &config.Config{
		NotionToken: "test_token",
		DatabaseID:  "test_db_id",
		Databases:   []config.DatabaseConfig{{ID: "test_db_id", Name: "Test DB"}},
		CacheDir:    t.TempDir(),
	}
	model := NewModel(NewModelInput{Config: cfg, Cache: nil})
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	return updated.(AppModel)
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestModel_HelpScreen(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)

	updated, _ := m.Update(runeKey('?'))
	m = updated.(AppModel)

	require.True(t, m.showHelp)
	view := m.View()
	assert.Contains(t, view, "Keyboard shortcuts")
	assert.Contains(t, view, "command palette")
	assert.Contains(t, view, "edit the page")

	// Any key closes help and is not acted on: "q" must not quit here
	updated, cmd := m.Update(runeKey('q'))
	m = updated.(AppModel)

	assert.False(t, m.showHelp)
	assert.False(t, isQuit(cmd))
	assert.NotContains(t, m.View(), "Keyboard shortcuts")
}

func TestModel_HelpListsEveryPaletteCommand(t *testing.T) {
	t.Parallel()

	help := RenderHelp(120, 50)

	for _, word := range []string{"search", "switch database", "refresh", "new page", "export", "help"} {
		assert.Contains(t, help, word)
	}
}

func TestModel_PageActionsNeedAnOpenPage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		action string
	}{
		{name: "new page", action: "new-page"},
		{name: "export", action: "export"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := newActionTestModel(t)
			require.Equal(t, PageDashboard, m.currentPage)

			cmd := m.handleCommandExecution(components.CommandExecutedMsg{ActionType: tt.action})

			assert.Nil(t, cmd)
			assert.Contains(t, m.statusBar.HelpText(), "Open a page first")
		})
	}
}

func TestModel_PageActionsRunOnTheOpenPage(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	m.navigateToDetail("page-1")
	detail := m.pages[PageDetail].(*pages.DetailPage)
	// The page is still loading, so the action is refused by the page itself
	// rather than by the "open a page first" guard.
	require.True(t, detail.IsLoading())

	cmd := m.handleCommandExecution(components.CommandExecutedMsg{ActionType: "export"})

	assert.Nil(t, cmd)
	assert.NotContains(t, m.statusBar.HelpText(), "Open a page first")
}

func TestModel_NewPageShortcutOutsideAPageDoesNothingHarmful(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = updated.(AppModel)

	assert.False(t, isQuit(cmd))
	assert.Equal(t, PageDashboard, m.currentPage)
	assert.Contains(t, m.statusBar.HelpText(), "Open a page first")
}

func TestModel_PageCreatedOpensTheNewPage(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	m.navigateToDetail("parent-page")

	updated, cmd := m.Update(pages.PageCreatedMsg{ID: "new-page-id"})
	m = updated.(AppModel)

	assert.Equal(t, PageDetail, m.currentPage)
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	require.True(t, ok)
	assert.Equal(t, "new-page-id", detail.PageID())
	assert.NotNil(t, cmd, "the new page loads and the tree reloads")
}

func TestModel_SidebarFocusKeepsKeysAwayFromPage(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	m.navigateToDetail("page-1")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(AppModel)
	require.True(t, m.sidebarFocus)

	// "r" would refresh the detail page if it leaked through to the page.
	_, cmd := m.Update(runeKey('r'))

	assert.Nil(t, cmd)
}

func TestModel_PaletteTypingDoesNotQuit(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = updated.(AppModel)
	require.True(t, m.showPalette)

	_, cmd := m.Update(runeKey('q'))

	assert.False(t, isQuit(cmd))
}

// busyDetail returns a model whose open page has an edit in flight.
func busyDetail(t *testing.T) AppModel {
	t.Helper()
	m := newActionTestModel(t)
	m.navigateToDetail("page-1")
	detail := m.pages[PageDetail].(*pages.DetailPage)
	// Finish loading with an empty result, then start an edit: the page is
	// busy until the fetch-before-edit command is run, which this test
	// never does.
	loaded := pages.NewLoadedMsgForTest("page-1", "content")
	updated, _ := m.Update(loaded)
	m = updated.(AppModel)
	require.NotNil(t, detail.StartEdit())
	require.True(t, detail.Busy())
	return m
}

func TestModel_BusyPageHoldsNavigationAndQuit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  tea.Msg
	}{
		{name: "esc", msg: tea.KeyMsg{Type: tea.KeyEsc}},
		{name: "q", msg: runeKey('q')},
		{name: "open another page from the tree", msg: components.TreeNavigationMsg{ID: "other", ObjectType: "page"}},
		{name: "open a search result", msg: pages.SearchNavigationMsg{ID: "other", ObjectType: "page"}},
		{name: "back request", msg: pages.BackNavigationMsg{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := busyDetail(t)

			updated, cmd := m.Update(tt.msg)
			m = updated.(AppModel)

			assert.False(t, isQuit(cmd))
			assert.Equal(t, PageDetail, m.currentPage)
			assert.Equal(t, "page-1", m.pages[PageDetail].(*pages.DetailPage).PageID())
			assert.Contains(t, m.statusBar.HelpText(), "Still working")
		})
	}
}

func TestModel_CtrlCQuitsEvenWhenBusy(t *testing.T) {
	t.Parallel()

	m := busyDetail(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	assert.True(t, isQuit(cmd))
}

func TestModel_DetailResultsReachTheDetailPageFromAnyPage(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	m.navigateToDetail("page-1")
	// Another page comes to the front while the detail page still loads.
	m.currentPage = PageDashboard

	updated, _ := m.Update(pages.NewLoadedMsgForTest("page-1", "late content"))
	m = updated.(AppModel)

	detail := m.pages[PageDetail].(*pages.DetailPage)
	assert.False(t, detail.IsLoading())
	assert.Equal(t, "late content", detail.Markdown())
}
