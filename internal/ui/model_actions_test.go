package ui

import (
	"testing"
	"time"

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

func TestModel_DatabaseRowOpensThatPage(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)

	// The list page reports the chosen row by its Notion page ID
	updated, cmd := m.Update(pages.NewNavigationMsg("4169a22b-64ff-803c-af88-e10404de525f"))
	m = updated.(AppModel)

	require.Equal(t, PageDetail, m.currentPage)
	detail, ok := m.pages[PageDetail].(*pages.DetailPage)
	require.True(t, ok)
	assert.Equal(t, "4169a22b-64ff-803c-af88-e10404de525f", detail.PageID())
	assert.NotNil(t, cmd, "the page starts loading")
	assert.NotContains(t, m.View(), "not found")
}

func TestModel_PaletteSearchOpensWorkspaceSearch(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)

	updated, _ := m.Update(components.CommandExecutedMsg{ActionType: "search"})
	m = updated.(AppModel)

	assert.Equal(t, PageWorkspaceSearch, m.currentPage)
	assert.Contains(t, m.View(), "Search (Workspace)")
	assert.True(t, m.navigator.CanGoBack())

	// The search box has focus, so typed letters are text, not shortcuts
	updated, cmd := m.Update(runeKey('q'))
	m = updated.(AppModel)
	assert.False(t, isQuit(cmd))
	assert.Equal(t, PageWorkspaceSearch, m.currentPage)
}

func TestModel_SearchResultRouting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectType string
		wantPage   PageID
	}{
		{name: "page result opens the page", objectType: "page", wantPage: PageDetail},
		{name: "database result opens its row list", objectType: "database", wantPage: PageList},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newActionTestModel(t)
			updated, _ := m.Update(components.CommandExecutedMsg{ActionType: "search"})
			m = updated.(AppModel)

			updated, cmd := m.Update(pages.SearchNavigationMsg{ID: "target-id", ObjectType: tt.objectType})
			m = updated.(AppModel)

			assert.Equal(t, tt.wantPage, m.currentPage)
			assert.NotNil(t, cmd)
			if tt.objectType == "database" {
				assert.Equal(t, "target-id", m.currentDBID)
			}
		})
	}
}

func TestModel_SwitchDatabaseFlow(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)

	updated, _ := m.Update(components.CommandExecutedMsg{ActionType: "switch-db"})
	m = updated.(AppModel)
	require.Equal(t, PageDatabaseList, m.currentPage)
	assert.Contains(t, m.View(), "Test DB")

	updated, cmd := m.Update(pages.DatabaseSelectedMsg{DatabaseID: "other-db"})
	m = updated.(AppModel)

	assert.Equal(t, PageList, m.currentPage)
	assert.Equal(t, "other-db", m.currentDBID)
	assert.NotNil(t, cmd, "the new database's rows start loading")
	assert.False(t, m.navigator.CanGoBack(), "a database switch starts a fresh history")
}

func TestModel_RefreshCommand(t *testing.T) {
	t.Parallel()

	t.Run("on a page reloads it", func(t *testing.T) {
		t.Parallel()
		m := newActionTestModel(t)
		m.navigateToDetail("page-1")
		updated, _ := m.Update(pages.NewLoadedMsgForTest("page-1", "content"))
		m = updated.(AppModel)

		cmd := m.handleCommandExecution(components.CommandExecutedMsg{ActionType: "refresh"})

		assert.NotNil(t, cmd)
		assert.True(t, m.pages[PageDetail].(*pages.DetailPage).IsLoading())
	})

	t.Run("on the dashboard does nothing", func(t *testing.T) {
		t.Parallel()
		m := newActionTestModel(t)

		assert.Nil(t, m.handleCommandExecution(components.CommandExecutedMsg{ActionType: "refresh"}))
	})
}

// listModel returns a model showing a database's rows.
func listModel(t *testing.T) AppModel {
	t.Helper()
	m := newActionTestModel(t)
	updated, _ := m.Update(pages.DatabaseSelectedMsg{DatabaseID: "db"})
	m = updated.(AppModel)
	require.Equal(t, PageList, m.currentPage)
	list := m.pages[PageList].(*pages.ListPage)
	list.SetPages([]pages.Page{pages.NewPage("row-1", "Quarterly report", "", time.Now())})
	return m
}

func TestModel_RowFilterTypingIsNotGlobalShortcuts(t *testing.T) {
	t.Parallel()

	m := listModel(t)
	updated, _ := m.Update(runeKey('/'))
	m = updated.(AppModel)
	list := m.pages[PageList].(*pages.ListPage)
	require.True(t, list.CapturesInput())

	for _, key := range []tea.KeyMsg{runeKey('q'), runeKey('?'), {Type: tea.KeyTab}} {
		updated, cmd := m.Update(key)
		m = updated.(AppModel)
		assert.False(t, isQuit(cmd), "%q must be filter text", key.String())
	}

	assert.Equal(t, PageList, m.currentPage)
	assert.False(t, m.showHelp)
	assert.False(t, m.sidebarFocus)
}

func TestModel_EscWithTreeFocusedDoesNotReachTheList(t *testing.T) {
	t.Parallel()

	m := listModel(t)
	// Apply a filter, then move focus to the tree
	for _, key := range []tea.KeyMsg{runeKey('/'), runeKey('Q'), {Type: tea.KeyEnter}, {Type: tea.KeyTab}} {
		updated, _ := m.Update(key)
		m = updated.(AppModel)
	}
	require.True(t, m.sidebarFocus)
	view := m.pages[PageList].(*pages.ListPage).View()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)

	assert.Equal(t, view, m.pages[PageList].(*pages.ListPage).View(), "esc in the tree must not change the row list")
}

func TestModel_ListResultsReachTheListFromAnyPage(t *testing.T) {
	t.Parallel()

	m := listModel(t)
	// A row is opened while the list reloads in the background
	updated, _ := m.Update(pages.NewNavigationMsg("row-1"))
	m = updated.(AppModel)
	require.Equal(t, PageDetail, m.currentPage)

	updated, _ = m.Update(pages.NewRowsLoadedMsgForTest(pages.NewPage("row-9", "Loaded in background", "", time.Now())))
	m = updated.(AppModel)

	assert.Contains(t, m.pages[PageList].(*pages.ListPage).View(), "Loaded in background")
	assert.Equal(t, PageDetail, m.currentPage, "the open page keeps the screen")
}

func TestModel_ReturningToRowsReloadsThem(t *testing.T) {
	t.Parallel()

	m := listModel(t)
	updated, _ := m.Update(pages.NewNavigationMsg("row-1"))
	m = updated.(AppModel)
	updated, _ = m.Update(pages.NewLoadedMsgForTest("row-1", "content"))
	m = updated.(AppModel)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)

	require.Equal(t, PageList, m.currentPage)
	assert.NotNil(t, cmd, "a property changed on the row must show in the list")
}

func TestModel_PageModesCaptureGlobalKeys(t *testing.T) {
	t.Parallel()

	m := newActionTestModel(t)
	m.navigateToDetail("page-1")
	updated, _ := m.Update(pages.NewLoadedMsgForTest("page-1", "- [ ] a task\n\ntext"))
	m = updated.(AppModel)

	// "t" opens the to-do list; from then on q, ? and tab belong to it
	updated, _ = m.Update(runeKey('t'))
	m = updated.(AppModel)
	detail := m.pages[PageDetail].(*pages.DetailPage)
	require.True(t, detail.CapturesInput())

	for _, key := range []tea.KeyMsg{runeKey('?'), {Type: tea.KeyTab}} {
		updated, cmd := m.Update(key)
		m = updated.(AppModel)
		assert.False(t, isQuit(cmd))
	}
	assert.False(t, m.showHelp)
	assert.False(t, m.sidebarFocus)
	assert.True(t, detail.CapturesInput())

	// esc closes the list rather than leaving the page
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(AppModel)
	assert.Equal(t, PageDetail, m.currentPage)
	assert.False(t, detail.CapturesInput())
}
