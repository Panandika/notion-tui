package pages

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/testhelpers"
)

// mockViewer implements ViewerInterface for testing.
type mockViewer struct {
	markdown     string
	blocks       []notionapi.Block
	width        int
	height       int
	updateCalled int
	initCalled   bool
}

func newMockViewer() *mockViewer {
	return &mockViewer{}
}

func (m *mockViewer) Init() tea.Cmd {
	m.initCalled = true
	return nil
}

func (m *mockViewer) Update(tea.Msg) (ViewerInterface, tea.Cmd) {
	m.updateCalled++
	return m, nil
}

func (m *mockViewer) View() string {
	if m.markdown == "" {
		return "No content"
	}
	return "Viewer: " + m.markdown
}

func (m *mockViewer) SetBlocks(blocks []notionapi.Block) tea.Cmd {
	m.blocks = blocks
	return nil
}

func (m *mockViewer) SetMarkdown(markdown string) tea.Cmd {
	m.markdown = markdown
	return nil
}

func (m *mockViewer) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// fakeEditor stands in for the external editor: it rewrites the file the
// page hands it, then reports that the editor exited.
type fakeEditor struct {
	write   func(original string) string // nil leaves the file untouched
	exitErr error
	opened  []string
	seen    string // Content the editor was opened with
}

func (f *fakeEditor) run(path string, done func(error) tea.Msg) tea.Cmd {
	f.opened = append(f.opened, path)
	return func() tea.Msg {
		raw, err := os.ReadFile(path)
		if err != nil {
			return done(err)
		}
		f.seen = string(raw)
		if f.write != nil {
			if err := os.WriteFile(path, []byte(f.write(f.seen)), 0o600); err != nil {
				return done(err)
			}
		}
		return done(f.exitErr)
	}
}

type detailFixture struct {
	page   *DetailPage
	client *testhelpers.MockNotionClient
	viewer *mockViewer
	editor *fakeEditor
}

// newDetailFixture returns a detail page that has finished loading markdown.
func newDetailFixture(t *testing.T, markdown string) detailFixture {
	t.Helper()
	client := testhelpers.NewMockNotionClient()
	client.PageMarkdownToReturn = &notion.PageMarkdown{ID: "page-1", Markdown: markdown}
	client.PageToReturn = testhelpers.NewTestPage("page-1", "My Page")
	viewer := newMockViewer()
	editor := &fakeEditor{}

	dp := NewDetailPage(NewDetailPageInput{
		Width:        80,
		Height:       24,
		Viewer:       viewer,
		NotionClient: client,
		PageID:       "page-1",
		RunEditor:    editor.run,
		ExportDir:    t.TempDir(),
	})
	// The fake editor returns instantly; that must not look like a GUI
	// editor that failed to wait.
	dp.quickExit = 0
	fx := detailFixture{page: &dp, client: client, viewer: viewer, editor: editor}
	fx.drain(t, dp.Init())
	require.False(t, dp.IsLoading())
	require.NoError(t, dp.Error())
	return fx
}

// drain runs a command and feeds every resulting message back into the page
// until no command is left. Messages for the root model are returned.
func (fx detailFixture) drain(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	var escaped []tea.Msg
	for steps := 0; cmd != nil; steps++ {
		require.Less(t, steps, 20, "command chain did not settle")
		msg := cmd()
		if _, ok := msg.(PageCreatedMsg); ok {
			escaped = append(escaped, msg)
			return escaped
		}
		_, cmd = fx.page.Update(msg)
	}
	return escaped
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// mustCreateCache creates a cache for testing or fails the test.
func mustCreateCache(t *testing.T) *cache.PageCache {
	t.Helper()
	c, err := cache.NewPageCache(cache.NewPageCacheInput{Dir: t.TempDir()})
	require.NoError(t, err)
	return c
}

func TestDetailPage_LoadsPageAsMarkdown(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "Hello <b>world</b>")

	assert.Equal(t, "Hello <b>world</b>", fx.page.Markdown(), "source markdown is kept as Notion returned it")
	assert.Equal(t, "Hello **world**", fx.viewer.markdown, "viewer gets terminal markdown")
	assert.Equal(t, "My Page", fx.page.Title())
	assert.Equal(t, 0, fx.client.GetBlocksCallCount(), "nested blocks must not be fetched one by one")
	assert.True(t, fx.viewer.initCalled)
}

func TestDetailPage_LoadError(t *testing.T) {
	t.Parallel()

	client := testhelpers.NewMockNotionClient()
	client.ErrorToReturn = errors.New("boom")
	dp := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, Viewer: newMockViewer(), NotionClient: client, PageID: "page-1"})

	dp.Update(dp.Init()())

	assert.False(t, dp.IsLoading())
	assert.ErrorContains(t, dp.Error(), "boom")
	assert.Contains(t, dp.View(), "Error")
}

func TestDetailPage_TruncatedPageSaysSo(t *testing.T) {
	t.Parallel()

	client := testhelpers.NewMockNotionClient()
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "part", Truncated: true}
	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, Viewer: viewer, NotionClient: client, PageID: "page-1"})

	dp.Update(dp.Init()())

	assert.Contains(t, viewer.markdown, "too large to load completely")
}

func TestDetailPage_IgnoresLoadResultForAnotherPage(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "current")

	fx.page.Update(pageLoadedMsg{pageID: "some-other-page", markdown: "stale"})

	assert.Equal(t, "current", fx.page.Markdown())
}

func TestDetailPage_Cache(t *testing.T) {
	t.Parallel()

	pageCache := mustCreateCache(t)
	client := testhelpers.NewMockNotionClient()
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "from api"}
	open := func() (*DetailPage, *mockViewer) {
		viewer := newMockViewer()
		dp := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, Viewer: viewer, NotionClient: client, Cache: pageCache, PageID: "page-1"})
		dp.Update(dp.Init()())
		return &dp, viewer
	}

	_, firstViewer := open()
	require.Equal(t, "from api", firstViewer.markdown)
	require.Len(t, client.GetPageMarkdownCalls, 1)

	// Second open is served from the cache
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "changed in notion"}
	dp, secondViewer := open()
	assert.Equal(t, "from api", secondViewer.markdown)
	assert.Len(t, client.GetPageMarkdownCalls, 1, "cache hit must not call the content API")

	// Refresh bypasses the cache and updates it
	dp.Update(dp.Refresh()())
	assert.Equal(t, "changed in notion", dp.Markdown())
	_, thirdViewer := open()
	assert.Equal(t, "changed in notion", thirdViewer.markdown)
}

func TestDetailPage_IgnoresCacheEntriesInOldFormat(t *testing.T) {
	t.Parallel()

	pageCache := mustCreateCache(t)
	// What older versions cached: a block list, not markdown.
	require.NoError(t, pageCache.Set(context.Background(), cache.SetInput{
		PageID: "page-1",
		Data:   map[string]any{"object": "list", "results": []any{}},
	}))
	client := testhelpers.NewMockNotionClient()
	client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "fresh"}
	viewer := newMockViewer()
	dp := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, Viewer: viewer, NotionClient: client, Cache: pageCache, PageID: "page-1"})

	dp.Update(dp.Init()())

	assert.Equal(t, "fresh", viewer.markdown)
}

func TestDetailPage_Keys(t *testing.T) {
	t.Parallel()

	t.Run("esc asks to go back", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "text")

		_, cmd := fx.page.Update(tea.KeyMsg{Type: tea.KeyEsc})

		require.NotNil(t, cmd)
		_, ok := cmd().(BackNavigationMsg)
		assert.True(t, ok)
	})

	t.Run("r reloads from the API", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "text")
		before := len(fx.client.GetPageMarkdownCalls)

		_, cmd := fx.page.Update(keyRune('r'))

		require.NotNil(t, cmd)
		assert.True(t, fx.page.IsLoading())
		fx.drain(t, cmd)
		assert.Len(t, fx.client.GetPageMarkdownCalls, before+1)
	})

	t.Run("other keys scroll the viewer", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "text")
		before := fx.viewer.updateCalled

		fx.page.Update(keyRune('j'))

		assert.Equal(t, before+1, fx.viewer.updateCalled)
	})
}

func TestDetailPage_WindowResize(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")

	fx.page.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	assert.Equal(t, 120, fx.viewer.width)
	assert.Equal(t, 39, fx.viewer.height, "one line is reserved for the status bar")
}

func TestDetailPage_Edit_SendsOnlyTheChangedPart(t *testing.T) {
	t.Parallel()

	source := "# Plan\n\nfirst paragraph with <b>bold</b>\n\nsecond paragraph"
	fx := newDetailFixture(t, source)
	fx.editor.write = func(string) string {
		return "# Plan\n\nfirst paragraph with <b>bold</b>\n\nsecond paragraph, edited\n"
	}
	fx.client.UpdatePageMarkdownFunc = func(_ context.Context, id string, _ []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		return &notion.PageMarkdown{ID: id, Markdown: "# Plan\n\nfirst paragraph with <b>bold</b>\n\nsecond paragraph, edited"}, nil
	}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Equal(t, source, fx.editor.seen, "the editor opens the page exactly as Notion has it")
	require.Len(t, fx.client.UpdatePageMarkdownCalls, 1)
	updates := fx.client.UpdatePageMarkdownCalls[0].Updates
	require.Len(t, updates, 1)
	assert.Equal(t, "second paragraph", updates[0].OldStr)
	assert.Equal(t, "second paragraph, edited", updates[0].NewStr)
	assert.Empty(t, fx.client.ReplacePageMarkdownCalls, "a small edit must not rewrite the whole page")

	assert.Contains(t, fx.viewer.markdown, "second paragraph, edited")
	assert.Contains(t, fx.viewer.markdown, "**bold**", "untouched formatting is still there")
	assert.Equal(t, "Saved", fx.page.notice)
	assert.NoFileExists(t, fx.editor.opened[0], "temp file is removed after a successful save")
}

func TestDetailPage_Edit_FetchesFreshContentFirst(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "loaded earlier")
	fx.client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "changed in notion since"}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Equal(t, "changed in notion since", fx.editor.seen)
}

func TestDetailPage_Edit_NoChangesWritesNothing(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "unchanged text")
	// Editors commonly add a trailing newline; that is not a change.
	fx.editor.write = func(original string) string { return original + "\n" }

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Empty(t, fx.client.UpdatePageMarkdownCalls)
	assert.Empty(t, fx.client.ReplacePageMarkdownCalls)
	assert.Equal(t, "No changes", fx.page.notice)
	assert.False(t, fx.page.busy)
}

func TestDetailPage_Edit_WholePageRewriteUsesReplace(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "old")
	fx.editor.write = func(string) string { return "entirely new" }

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	require.Len(t, fx.client.ReplacePageMarkdownCalls, 1)
	assert.Equal(t, "entirely new", fx.client.ReplacePageMarkdownCalls[0].Markdown)
	assert.Empty(t, fx.client.UpdatePageMarkdownCalls)
}

func TestDetailPage_Edit_FailedSaveKeepsTheUsersText(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "line one\n\nline two")
	fx.editor.write = func(string) string { return "line one\n\nline two, with an hour of work" }
	fx.client.UpdatePageMarkdownFunc = func(context.Context, string, []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		return nil, errors.New("network down")
	}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	path := fx.editor.opened[0]
	kept, err := os.ReadFile(path)
	require.NoError(t, err, "the edited file must survive a failed save")
	assert.Contains(t, string(kept), "with an hour of work")
	assert.Contains(t, fx.page.notice, "Save failed")
	assert.Contains(t, fx.page.notice, path, "the user is told where their text is")
	assert.Equal(t, "line one\n\nline two", fx.page.Markdown(), "the view still shows what Notion has")
	assert.False(t, fx.page.busy)
}

func TestDetailPage_Edit_SkippedEditsKeepTheFile(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "a\n\nb")
	fx.editor.write = func(string) string { return "a\n\nb changed" }
	fx.client.UpdatePageMarkdownFunc = func(_ context.Context, id string, _ []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		return &notion.PageMarkdown{ID: id, Markdown: "a\n\nb", Warnings: []notion.MarkdownWarning{{Code: "no_match"}}}, nil
	}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.FileExists(t, fx.editor.opened[0])
	assert.Contains(t, fx.page.notice, "could not be placed")
}

func TestDetailPage_Edit_RefusesTruncatedPage(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")
	fx.client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "part", Truncated: true}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Empty(t, fx.editor.opened, "editing a partial page would write placeholders back")
	assert.Contains(t, fx.page.notice, "too large to edit")
	assert.False(t, fx.page.busy)
}

func TestDetailPage_Edit_EditorFailureSendsNothingButKeepsTheText(t *testing.T) {
	t.Parallel()

	// vim exits non-zero after ":wq" if any error message appeared.
	fx := newDetailFixture(t, "text")
	fx.editor.write = func(string) string { return "text the user saved before the editor failed" }
	fx.editor.exitErr = errors.New("exit status 1")

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Empty(t, fx.client.UpdatePageMarkdownCalls)
	assert.Empty(t, fx.client.ReplacePageMarkdownCalls)
	path := fx.editor.opened[0]
	kept, err := os.ReadFile(path)
	require.NoError(t, err, "edited text must survive an editor failure")
	assert.Contains(t, string(kept), "saved before the editor failed")
	assert.Contains(t, fx.page.notice, "Editor failed")
	assert.Contains(t, fx.page.notice, path)
	assert.False(t, fx.page.Busy())
}

func TestDetailPage_Edit_EditorFailureWithoutChangesCleansUp(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")
	fx.editor.exitErr = errors.New("exit status 1")

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.NoFileExists(t, fx.editor.opened[0])
	assert.Contains(t, fx.page.notice, "Nothing was changed")
}

func TestDetailPage_Edit_InstantReturnHintsAtWaitFlag(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")
	fx.page.quickExit = time.Hour // Treat the fake editor's instant return as too fast

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Contains(t, fx.page.notice, "--wait")
	assert.FileExists(t, fx.editor.opened[0], "a GUI editor may still have the file open")
	assert.False(t, fx.page.Busy())
}

func TestDetailPage_Edit_ReplaceRefusedWhenPageChangedMeanwhile(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "old")
	fx.editor.write = func(string) string {
		// Someone edits the page in Notion while the editor is open.
		fx.client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "old, plus a colleague's change"}
		return "entirely new"
	}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Empty(t, fx.client.ReplacePageMarkdownCalls, "a full replace would wipe the colleague's change")
	assert.Contains(t, fx.page.notice, "changed in Notion")
	assert.FileExists(t, fx.editor.opened[0])
}

func TestDetailPage_Edit_EmptySaveResponseReloadsThePage(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "a\n\nb")
	fx.editor.write = func(string) string {
		fx.client.PageMarkdownToReturn = &notion.PageMarkdown{Markdown: "a\n\nb changed"}
		return "a\n\nb changed"
	}
	fx.client.UpdatePageMarkdownFunc = func(_ context.Context, id string, _ []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		return &notion.PageMarkdown{ID: id}, nil // No markdown in the response
	}

	_, cmd := fx.page.Update(keyRune('e'))
	fx.drain(t, cmd)

	assert.Equal(t, "a\n\nb changed", fx.page.Markdown(), "an empty response must not blank the page")
}

func TestDetailPage_ResultsForAnotherPageAreIgnored(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")
	fx.page.busy = true

	for _, msg := range []tea.Msg{
		editorClosedMsg{pageID: "other"},
		editSavedMsg{pageID: "other", markdown: "foreign"},
		pageCreatedResultMsg{pageID: "other", id: "x"},
		exportDoneMsg{pageID: "other", path: "x.md"},
	} {
		_, cmd := fx.page.Update(msg)
		assert.Nil(t, cmd)
	}

	assert.Equal(t, "text", fx.page.Markdown())
	assert.True(t, fx.page.Busy(), "another page's result must not end this page's work")
}

func TestDetailPage_Export_TruncatedPageSaysPartial(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "part")
	fx.page.truncated = true

	fx.drain(t, fx.page.Export())

	assert.Contains(t, fx.page.notice, "partial")
}

func TestDetailPage_Edit_IgnoredWhileAnotherActionRuns(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")

	_, first := fx.page.Update(keyRune('e'))
	require.NotNil(t, first)
	_, second := fx.page.Update(keyRune('e'))

	assert.Nil(t, second, "a second edit must not start while the first is in flight")
}

func TestDetailPage_NewPage_CreatesChildOfCurrentPage(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "parent content")
	fx.editor.write = func(string) string { return "# Meeting notes\n\n- item" }

	escaped := fx.drain(t, fx.page.StartNewPage())

	assert.Equal(t, newPageTemplate, fx.editor.seen)
	require.Len(t, fx.client.CreatePageMarkdownCalls, 1)
	call := fx.client.CreatePageMarkdownCalls[0]
	assert.Equal(t, "page-1", call.ParentPageID)
	assert.Equal(t, "# Meeting notes\n\n- item", call.Markdown)
	require.Len(t, escaped, 1)
	assert.Equal(t, PageCreatedMsg{ID: "created-page-id"}, escaped[0])
}

func TestDetailPage_NewPage_UntouchedTemplateCreatesNothing(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "parent content")

	escaped := fx.drain(t, fx.page.StartNewPage())

	assert.Empty(t, fx.client.CreatePageMarkdownCalls)
	assert.Empty(t, escaped)
	assert.Contains(t, fx.page.notice, "cancelled")
}

func TestDetailPage_NewPage_FailureKeepsTheFile(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "parent content")
	fx.editor.write = func(string) string { return "# Draft\n\nlong text" }
	fx.client.CreatePageMarkdownFunc = func(context.Context, string, string) (*notion.CreatedPage, error) {
		return nil, errors.New("forbidden")
	}

	fx.drain(t, fx.page.StartNewPage())

	assert.FileExists(t, fx.editor.opened[0])
	assert.Contains(t, fx.page.notice, "Create failed")
}

func TestDetailPage_Export_WritesPlainMarkdown(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "Hello <b>world</b>\n\n<toggle>\nMore\n\ninside\n</toggle>")

	fx.drain(t, fx.page.Export())

	require.Contains(t, fx.page.notice, "Exported to ")
	path := filepath.Join(fx.page.exportDir, "my-page.md")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "# My Page\n\nHello **world**\n\n▸ More\n\n> inside\n", string(raw))
}

func TestDetailPage_Export_NeverOverwrites(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "new content")
	existing := filepath.Join(fx.page.exportDir, "my-page.md")
	require.NoError(t, os.WriteFile(existing, []byte("precious"), 0o600))

	fx.drain(t, fx.page.Export())

	kept, err := os.ReadFile(existing)
	require.NoError(t, err)
	assert.Equal(t, "precious", string(kept))
	assert.FileExists(t, filepath.Join(fx.page.exportDir, "my-page-1.md"))
}

func TestDetailPage_AnyKeyClearsTheNotice(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "text")
	fx.drain(t, fx.page.Export())
	require.NotEmpty(t, fx.page.notice)

	fx.page.Update(keyRune('j'))

	assert.Empty(t, fx.page.notice)
	assert.Equal(t, detailHelpText, fx.page.statusBar.HelpText())
}

func TestDetailPage_LongNoticeGetsItsOwnScreen(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "page body")
	longPath := "/var/folders/zz/some/very/long/temporary/directory/notion-tui-123456789/my-page.md"

	fx.page.setNotice("Save failed: network down. Your text is kept in " + longPath)

	view := fx.page.View()
	compact := strings.Join(strings.Fields(view), "")
	assert.Contains(t, compact, strings.ReplaceAll(longPath, " ", ""), "the whole path must be readable, wrapped if needed")
	assert.Contains(t, view, "press any key to continue")
	assert.NotContains(t, view, "Viewer: ", "the message replaces the page until acknowledged")

	fx.page.Update(keyRune('j'))

	assert.Contains(t, fx.page.View(), "Viewer: ")
}

func TestDetailPage_ShortNoticeStaysInStatusBar(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "page body")

	fx.page.setNotice("Saved")

	view := fx.page.View()
	assert.Contains(t, view, "Viewer: ")
	assert.Contains(t, view, "Saved")
}

func TestDetailPage_ViewStates(t *testing.T) {
	t.Parallel()

	client := testhelpers.NewMockNotionClient()
	dp := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, Viewer: newMockViewer(), NotionClient: client, PageID: "page-1"})
	assert.Contains(t, dp.View(), "Loading page...")

	dp.Update(dp.Init()())
	assert.Contains(t, dp.View(), "Viewer: ")

	withoutViewer := NewDetailPage(NewDetailPageInput{Width: 80, Height: 24, NotionClient: client, PageID: "page-1"})
	withoutViewer.Update(withoutViewer.Init()())
	assert.Contains(t, withoutViewer.View(), "No viewer available")
}

func TestFileSlug(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		title string
		want  string
	}{
		{name: "words", title: "My Page", want: "my-page"},
		{name: "punctuation collapses", title: "Q3: Plan / Budget (v2)", want: "q3-plan-budget-v2"},
		{name: "path separators cannot escape the directory", title: "../../etc/passwd", want: "etc-passwd"},
		{name: "no usable characters", title: "日本語", want: "page"},
		{name: "empty", title: "   ", want: "page"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, fileSlug(tt.title))
		})
	}

	long := fileSlug("a very long title that keeps going and going well past any sensible file name length limit")
	assert.LessOrEqual(t, len(long), 60)
}

func TestEditorParts(t *testing.T) {
	t.Parallel()

	env := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}
	tests := []struct {
		name string
		env  map[string]string
		want []string
	}{
		{name: "VISUAL wins over EDITOR", env: map[string]string{"VISUAL": "nvim", "EDITOR": "nano"}, want: []string{"nvim"}},
		{name: "EDITOR is used without VISUAL", env: map[string]string{"EDITOR": "nano"}, want: []string{"nano"}},
		{name: "arguments are kept", env: map[string]string{"EDITOR": "code --wait"}, want: []string{"code", "--wait"}},
		{name: "blank values fall back to vi", env: map[string]string{"VISUAL": "  ", "EDITOR": ""}, want: []string{"vi"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, editorParts(env(tt.env)))
		})
	}

	cmd := editorCommand("/tmp/page.md", env(map[string]string{"EDITOR": "code --wait"}))
	assert.Equal(t, []string{"code", "--wait", "/tmp/page.md"}, cmd.Args)
}
