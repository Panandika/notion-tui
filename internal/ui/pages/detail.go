package pages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/cache"
	"github.com/Panandika/notion-tui/internal/notion"
	"github.com/Panandika/notion-tui/internal/ui/components"
)

// ViewerInterface is imported from components package.
// This type alias ensures compatibility with the viewer implementation.
type ViewerInterface = components.ViewerInterface

const (
	detailHelpText = "e: edit | r: refresh | esc: back | ?: help"

	// cacheFormat marks cache entries written by this version of the page
	// loader. Entries without it (older block caches) are ignored.
	cacheFormat = "markdown-v2"

	// newPageTemplate is what the editor opens with for a new page. The
	// first heading becomes the page title.
	newPageTemplate = "# Untitled\n\n"

	truncatedNotice = "*This page is too large to load completely. Some content is not shown.*"
)

// PageCreatedMsg is emitted after a new page was created in Notion.
type PageCreatedMsg struct {
	ID string
}

// pageLoadedMsg is returned when page data is fetched.
type pageLoadedMsg struct {
	pageID    string // Page this result belongs to; empty matches any page
	page      *notionapi.Page
	markdown  string
	truncated bool
	err       error
}

// editSourceMsg carries the page's current Markdown, fetched right before
// the editor opens so the user edits what is in Notion now.
type editSourceMsg struct {
	pageID    string
	markdown  string
	truncated bool
	err       error
}

// editorClosedMsg is sent when the external editor exits.
type editorClosedMsg struct {
	pageID string
	err    error
}

// editSavedMsg is the result of writing an edited page back to Notion.
type editSavedMsg struct {
	pageID    string
	markdown  string // Page content after the save, as returned by the API
	truncated bool
	skipped   int // Edits the API could not place
	err       error
}

// errPageChanged means the page was modified in Notion while it was open in
// the editor, so replacing its whole content would discard that change.
var errPageChanged = errors.New("the page changed in Notion while you were editing")

// pageCreatedResultMsg is the result of creating a new page.
type pageCreatedResultMsg struct {
	pageID string // Parent page the request was made from
	id     string
	err    error
}

// exportDoneMsg is the result of exporting the page to a file.
type exportDoneMsg struct {
	pageID    string
	path      string
	truncated bool
	err       error
}

// DetailResultMsg marks the results of work a DetailPage started. The root
// model delivers these to the detail page even when another page is in
// front, so a finished save is never reported to the wrong page and lost.
type DetailResultMsg interface {
	detailResult()
}

func (pageLoadedMsg) detailResult()        {}
func (editSourceMsg) detailResult()        {}
func (editorClosedMsg) detailResult()      {}
func (editSavedMsg) detailResult()         {}
func (pageCreatedResultMsg) detailResult() {}
func (exportDoneMsg) detailResult()        {}

// cachedPage is the cache representation of a loaded page.
type cachedPage struct {
	Format    string `json:"format"`
	Markdown  string `json:"markdown"`
	Truncated bool   `json:"truncated"`
}

// editSession tracks one run of the external editor.
type editSession struct {
	dir      string    // Private temp directory holding the file
	path     string    // File the editor works on
	original string    // Content the editor started with
	newPage  bool      // The file becomes a new child page instead of an edit
	opened   time.Time // When the editor was started
}

// quickExitThreshold is how fast an editor must return, without changing the
// file, to look like a GUI editor that did not wait for the window to close.
const quickExitThreshold = time.Second

// EditorRunner opens path in an editor and reports back when it exits.
// The default suspends the TUI and runs $VISUAL or $EDITOR.
type EditorRunner func(path string, done func(error) tea.Msg) tea.Cmd

// DetailPage displays a single Notion page and lets the user edit it in
// their own editor. Content is read and written as Markdown.
type DetailPage struct {
	viewer       ViewerInterface
	statusBar    components.StatusBar
	pageID       string
	page         *notionapi.Page
	markdown     string // Notion-flavored Markdown as returned by the API
	truncated    bool
	loading      bool
	err          error
	width        int
	height       int
	notionClient NotionClient
	cache        *cache.PageCache

	runEditor EditorRunner
	exportDir string
	session   *editSession
	quickExit time.Duration // See quickExitThreshold
	busy      bool          // An edit, save, create or export is in progress
	notice    string        // One-off message shown in the status bar
}

// NewDetailPageInput contains the parameters for creating a DetailPage.
type NewDetailPageInput struct {
	Width        int
	Height       int
	Viewer       ViewerInterface
	NotionClient NotionClient
	Cache        *cache.PageCache
	PageID       string
	// RunEditor overrides how the editor is launched; nil uses $VISUAL/$EDITOR.
	RunEditor EditorRunner
	// ExportDir is where exports are written; empty means the working directory.
	ExportDir string
}

// NewDetailPage creates a new DetailPage instance.
func NewDetailPage(input NewDetailPageInput) DetailPage {
	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode(components.ModeBrowse)
	statusBar.SetSyncStatus(components.StatusSynced)
	statusBar.SetHelpText(detailHelpText)

	viewerHeight := input.Height - 1 // Reserve 1 line for status bar
	if input.Viewer != nil {
		input.Viewer.SetSize(input.Width, viewerHeight)
	}

	runEditor := input.RunEditor
	if runEditor == nil {
		runEditor = runExternalEditor
	}

	return DetailPage{
		viewer:       input.Viewer,
		statusBar:    statusBar,
		pageID:       input.PageID,
		loading:      true,
		width:        input.Width,
		height:       input.Height,
		notionClient: input.NotionClient,
		cache:        input.Cache,
		runEditor:    runEditor,
		exportDir:    input.ExportDir,
		quickExit:    quickExitThreshold,
	}
}

// Init initializes the DetailPage and loads the page content.
func (dp *DetailPage) Init() tea.Cmd {
	// Call viewer init immediately if available
	if dp.viewer != nil {
		dp.viewer.Init()
	}

	return dp.fetchPageCmd()
}

// Update handles messages and updates the DetailPage state.
func (dp *DetailPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pageLoadedMsg:
		// A slow load of a previously opened page must not replace this one
		if msg.pageID != "" && msg.pageID != dp.pageID {
			return dp, nil
		}
		if msg.err != nil {
			dp.err = msg.err
			dp.loading = false
			dp.statusBar.SetSyncStatus(components.StatusError)
			return dp, nil
		}

		dp.page = msg.page
		dp.loading = false
		dp.statusBar.SetSyncStatus(components.StatusSynced)
		return dp, dp.showMarkdown(msg.markdown, msg.truncated)

	case editSourceMsg:
		return dp, dp.handleEditSource(msg)

	case editorClosedMsg:
		return dp, dp.handleEditorClosed(msg)

	case editSavedMsg:
		return dp, dp.handleEditSaved(msg)

	case pageCreatedResultMsg:
		return dp, dp.handlePageCreated(msg)

	case exportDoneMsg:
		if msg.pageID != dp.pageID {
			return dp, nil
		}
		dp.busy = false
		switch {
		case msg.err != nil:
			dp.setNotice("Export failed: " + msg.err.Error())
		case msg.truncated:
			dp.setNotice("Exported to " + msg.path + " (partial: the page is too large for Notion to return in full)")
		default:
			dp.setNotice("Exported to " + msg.path)
		}
		return dp, nil

	case tea.KeyMsg:
		// Any key acknowledges the last notice
		dp.clearNotice()

		switch msg.String() {
		case "r":
			// Refresh from API (bypass cache)
			return dp, dp.Refresh()

		case "e":
			return dp, dp.StartEdit()

		case "esc":
			// Navigate back (use exported BackNavigationMsg)
			return dp, func() tea.Msg {
				return BackNavigationMsg{}
			}
		}

	case tea.WindowSizeMsg:
		dp.width = msg.Width
		dp.height = msg.Height
		dp.statusBar.SetWidth(msg.Width)

		// Update viewer size (reserve 1 line for status bar)
		viewerHeight := msg.Height - 1
		if dp.viewer != nil {
			dp.viewer.SetSize(msg.Width, viewerHeight)
		}
		return dp, nil
	}

	// Delegate other messages to viewer
	if dp.viewer != nil {
		updatedViewer, cmd := dp.viewer.Update(msg)
		dp.viewer = updatedViewer
		return dp, cmd
	}

	return dp, nil
}

// View renders the DetailPage UI.
func (dp *DetailPage) View() string {
	if dp.loading {
		loadingText := "Loading page..."
		dp.statusBar.SetSyncStatus(components.StatusSyncing)
		statusContent := dp.statusBar.View()
		return lipgloss.JoinVertical(lipgloss.Left, loadingText, statusContent)
	}

	if dp.err != nil {
		errorText := fmt.Sprintf("Error: %v\nPress ESC to go back", dp.err)
		statusContent := dp.statusBar.View()
		return lipgloss.JoinVertical(lipgloss.Left, errorText, statusContent)
	}

	// A message too long for the status bar gets the whole page until the
	// next key press, so file paths and errors are never cut off.
	if dp.noticeNeedsScreen() {
		return lipgloss.JoinVertical(lipgloss.Left, dp.renderNoticeScreen(), dp.statusBar.View())
	}

	// Render viewer content and status bar
	var viewerContent string
	if dp.viewer != nil {
		viewerContent = dp.viewer.View()
	} else {
		viewerContent = "No viewer available"
	}

	statusContent := dp.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, viewerContent, statusContent)
}

// Refresh reloads the current page from the API, bypassing cache.
func (dp *DetailPage) Refresh() tea.Cmd {
	dp.loading = true
	dp.err = nil
	dp.statusBar.SetSyncStatus(components.StatusSyncing)
	return dp.fetchPageFromAPICmd()
}

// StartEdit opens the page in the user's editor. The content is fetched
// fresh first, so the edit starts from what Notion holds right now.
func (dp *DetailPage) StartEdit() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	dp.busy = true
	dp.setNotice("Opening editor...")

	pageID := dp.pageID
	client := dp.notionClient
	return func() tea.Msg {
		page, err := client.GetPageMarkdown(context.Background(), pageID)
		if err != nil {
			return editSourceMsg{pageID: pageID, err: err}
		}
		return editSourceMsg{pageID: pageID, markdown: page.Markdown, truncated: page.Truncated}
	}
}

// StartNewPage opens the editor for a new page created inside this page.
func (dp *DetailPage) StartNewPage() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	session, err := newEditSession("new-page", newPageTemplate)
	if err != nil {
		dp.setNotice("Could not prepare editor: " + err.Error())
		return nil
	}
	session.newPage = true
	return dp.openEditor(session)
}

// Export writes the page as plain Markdown into the export directory.
func (dp *DetailPage) Export() tea.Cmd {
	if !dp.ready() {
		return nil
	}
	dp.busy = true
	dp.setNotice("Exporting...")

	pageID := dp.pageID
	title := dp.Title()
	body := notion.RenderableMarkdown(dp.markdown)
	truncated := dp.truncated
	dir := dp.exportDir
	return func() tea.Msg {
		path, err := writeExport(dir, title, body)
		return exportDoneMsg{pageID: pageID, path: path, truncated: truncated, err: err}
	}
}

// ready reports whether the page is loaded and no other action is running.
func (dp *DetailPage) ready() bool {
	return !dp.loading && dp.err == nil && !dp.busy
}

// handleEditSource writes the fetched Markdown to a temp file and opens it.
func (dp *DetailPage) handleEditSource(msg editSourceMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	if msg.err != nil {
		dp.busy = false
		dp.setNotice("Could not load page for editing: " + msg.err.Error())
		return nil
	}
	if msg.truncated {
		// Saving a partial page would write the placeholders back.
		dp.busy = false
		dp.setNotice("Page is too large to edit here: Notion returned only part of it")
		return nil
	}

	session, err := newEditSession(dp.Title(), msg.markdown)
	if err != nil {
		dp.busy = false
		dp.setNotice("Could not prepare editor: " + err.Error())
		return nil
	}
	return dp.openEditor(session)
}

// openEditor hands the session's file to the editor.
func (dp *DetailPage) openEditor(session *editSession) tea.Cmd {
	session.opened = time.Now()
	dp.session = session
	dp.busy = true
	dp.clearNotice()
	pageID := dp.pageID
	return dp.runEditor(session.path, func(err error) tea.Msg {
		return editorClosedMsg{pageID: pageID, err: err}
	})
}

// handleEditorClosed reads the edited file and saves it if it changed.
func (dp *DetailPage) handleEditorClosed(msg editorClosedMsg) tea.Cmd {
	session := dp.session
	if session == nil || msg.pageID != dp.pageID {
		// Not this page's editor run: leave any work in progress alone.
		return nil
	}

	raw, readErr := os.ReadFile(session.path)
	edited := notion.NormalizeMarkdown(string(raw))
	unchanged := readErr == nil &&
		(edited == notion.NormalizeMarkdown(session.original) || (session.newPage && strings.TrimSpace(edited) == ""))

	if msg.err != nil || readErr != nil {
		// Editors can exit non-zero after saving (vim does after any error
		// message). Nothing is sent in that case, but text the user wrote
		// must not be deleted.
		reason := "Editor failed"
		cause := msg.err
		if cause == nil {
			reason, cause = "Could not read edited file", readErr
		}
		if unchanged {
			dp.endSession(true)
			dp.setNotice(fmt.Sprintf("%s: %v. Nothing was changed", reason, cause))
		} else {
			dp.setNotice(fmt.Sprintf("%s: %v. Nothing was sent to Notion. Your text is kept in %s", reason, cause, session.path))
			dp.endSession(false)
		}
		return nil
	}

	if unchanged {
		// An editor that returns at once without changes is usually a GUI
		// editor that did not wait. Its window still has the file open, so
		// the file stays.
		if time.Since(session.opened) < dp.quickExit {
			dp.setNotice(fmt.Sprintf("Editor returned immediately without changes. If it opens a window, set a wait flag (for example EDITOR=\"code --wait\"). File: %s", session.path))
			dp.endSession(false)
			return nil
		}
		dp.endSession(true)
		if session.newPage {
			dp.setNotice("New page cancelled: nothing was written")
		} else {
			dp.setNotice("No changes")
		}
		return nil
	}

	dp.setNotice("Saving...")
	dp.statusBar.SetSyncStatus(components.StatusSyncing)
	if session.newPage {
		return dp.createPageCmd(edited)
	}
	return dp.savePageCmd(session.original, edited)
}

// savePageCmd writes the edited page. Only the changed parts are sent, so
// untouched blocks keep their identity; a rewrite of the whole page is used
// when the change cannot be expressed as separate edits.
func (dp *DetailPage) savePageCmd(original, edited string) tea.Cmd {
	pageID := dp.pageID
	client := dp.notionClient
	return func() tea.Msg {
		ctx := context.Background()

		updates, ok := notion.BuildContentUpdates(original, edited)
		var (
			result *notion.PageMarkdown
			err    error
		)
		if ok {
			// Edits locate their text in the page, so they fail on their own
			// if that text changed in Notion meanwhile.
			result, err = client.UpdatePageMarkdown(ctx, pageID, updates)
		} else {
			// A full replace has no such protection: check first that the
			// page is still what the editor started from.
			current, fetchErr := client.GetPageMarkdown(ctx, pageID)
			switch {
			case fetchErr != nil:
				err = fetchErr
			case current.Truncated || notion.NormalizeMarkdown(current.Markdown) != notion.NormalizeMarkdown(original):
				err = errPageChanged
			default:
				result, err = client.ReplacePageMarkdown(ctx, pageID, edited)
			}
		}
		if err != nil {
			return editSavedMsg{pageID: pageID, err: err}
		}
		return editSavedMsg{
			pageID:    pageID,
			markdown:  result.Markdown,
			truncated: result.Truncated,
			skipped:   len(result.Warnings),
		}
	}
}

// createPageCmd creates a new child page from the editor's content.
func (dp *DetailPage) createPageCmd(markdown string) tea.Cmd {
	parentID := dp.pageID
	client := dp.notionClient
	return func() tea.Msg {
		created, err := client.CreatePageMarkdown(context.Background(), parentID, markdown)
		if err != nil {
			return pageCreatedResultMsg{pageID: parentID, err: err}
		}
		return pageCreatedResultMsg{pageID: parentID, id: created.ID}
	}
}

// handleEditSaved shows the saved content, or keeps the user's file on failure.
func (dp *DetailPage) handleEditSaved(msg editSavedMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		dp.setNotice(fmt.Sprintf("Save failed: %v. Your text is kept in %s", msg.err, dp.keptFile()))
		dp.endSession(false)
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)

	if msg.skipped > 0 {
		dp.setNotice(fmt.Sprintf("Saved, but %d edit(s) could not be placed. Your text is kept in %s", msg.skipped, dp.keptFile()))
		dp.endSession(false)
	} else {
		dp.setNotice("Saved")
		dp.endSession(true)
	}

	// The response normally carries the page after the save. Without it,
	// load the page again rather than showing (and caching) an empty one.
	if strings.TrimSpace(msg.markdown) == "" {
		return dp.fetchPageFromAPICmd()
	}
	dp.storeInCache(msg.markdown, msg.truncated)
	return dp.showMarkdown(msg.markdown, msg.truncated)
}

// handlePageCreated reports the new page to the root model.
func (dp *DetailPage) handlePageCreated(msg pageCreatedResultMsg) tea.Cmd {
	if msg.pageID != dp.pageID {
		return nil
	}
	if msg.err != nil {
		dp.statusBar.SetSyncStatus(components.StatusError)
		dp.setNotice(fmt.Sprintf("Create failed: %v. Your text is kept in %s", msg.err, dp.keptFile()))
		dp.endSession(false)
		return nil
	}
	dp.statusBar.SetSyncStatus(components.StatusSynced)
	dp.endSession(true)
	dp.setNotice("Page created")

	created := PageCreatedMsg{ID: msg.id}
	return func() tea.Msg { return created }
}

// keptFile returns the path of the current session's file, for messages.
func (dp *DetailPage) keptFile() string {
	if dp.session == nil {
		return "(no file)"
	}
	return dp.session.path
}

// endSession finishes the current edit. With removeFile false the temp file
// stays on disk so the user's text is not lost after a failed save.
func (dp *DetailPage) endSession(removeFile bool) {
	if dp.session != nil && removeFile {
		// Best effort: a leftover temp directory is harmless.
		_ = os.RemoveAll(dp.session.dir)
	}
	dp.session = nil
	dp.busy = false
}

// showMarkdown stores the page content and renders it in the viewer.
func (dp *DetailPage) showMarkdown(markdown string, truncated bool) tea.Cmd {
	dp.markdown = markdown
	dp.truncated = truncated
	if dp.viewer == nil {
		return nil
	}

	display := notion.RenderableMarkdown(markdown)
	if truncated {
		display = strings.TrimSpace(display + "\n\n" + truncatedNotice)
	}
	return dp.viewer.SetMarkdown(display)
}

// statusBarReserve is the room the status bar needs for mode and sync state.
const statusBarReserve = 30

// setNotice shows a one-off message: in the status bar when it fits, on its
// own screen when it does not.
func (dp *DetailPage) setNotice(text string) {
	dp.notice = text
	if dp.noticeNeedsScreen() {
		dp.statusBar.SetHelpText("press any key to continue")
		return
	}
	dp.statusBar.SetHelpText(text)
}

// noticeNeedsScreen reports whether the current notice is too long for the
// status bar.
func (dp *DetailPage) noticeNeedsScreen() bool {
	return dp.notice != "" && lipgloss.Width(dp.notice) > dp.width-statusBarReserve
}

// renderNoticeScreen renders the notice wrapped to the page width.
func (dp *DetailPage) renderNoticeScreen() string {
	width := dp.width - 4
	if width < 20 {
		width = 20
	}
	height := dp.height - 1 // Status bar
	if height < 1 {
		height = 1
	}
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 2).
		Render(dp.notice)
}

// clearNotice restores the regular key help.
func (dp *DetailPage) clearNotice() {
	if dp.notice == "" {
		return
	}
	dp.notice = ""
	dp.statusBar.SetHelpText(detailHelpText)
}

// fetchPageCmd loads page data, trying cache first then falling back to API.
func (dp *DetailPage) fetchPageCmd() tea.Cmd {
	pageID := dp.pageID
	return func() tea.Msg {
		ctx := context.Background()

		if cached, ok := dp.readCache(ctx, pageID); ok {
			// Page metadata is not cached
			page, err := dp.notionClient.GetPage(ctx, pageID)
			if err != nil {
				return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch page metadata: %w", err)}
			}
			return pageLoadedMsg{
				pageID:    pageID,
				page:      page,
				markdown:  cached.Markdown,
				truncated: cached.Truncated,
			}
		}

		// Cache miss or error - fetch from API
		return dp.fetchPageFromAPI(pageID)
	}
}

// fetchPageFromAPICmd fetches page data directly from the API.
func (dp *DetailPage) fetchPageFromAPICmd() tea.Cmd {
	pageID := dp.pageID
	return func() tea.Msg {
		return dp.fetchPageFromAPI(pageID)
	}
}

// fetchPageFromAPI performs the actual API calls to fetch the page.
// pageID is passed in rather than read from dp: this runs on another goroutine.
func (dp *DetailPage) fetchPageFromAPI(pageID string) tea.Msg {
	ctx := context.Background()

	// Fetch page metadata
	page, err := dp.notionClient.GetPage(ctx, pageID)
	if err != nil {
		return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch page: %w", err)}
	}

	// One call returns the whole page, nested content included
	content, err := dp.notionClient.GetPageMarkdown(ctx, pageID)
	if err != nil {
		return pageLoadedMsg{pageID: pageID, err: fmt.Errorf("fetch content: %w", err)}
	}

	dp.writeCache(ctx, pageID, content.Markdown, content.Truncated)

	return pageLoadedMsg{
		pageID:    pageID,
		page:      page,
		markdown:  content.Markdown,
		truncated: content.Truncated,
	}
}

// readCache returns the cached content of a page, if a usable entry exists.
func (dp *DetailPage) readCache(ctx context.Context, pageID string) (cachedPage, bool) {
	if dp.cache == nil {
		return cachedPage{}, false
	}
	raw, err := dp.cache.Get(ctx, pageID)
	if err != nil {
		return cachedPage{}, false
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return cachedPage{}, false
	}
	var entry cachedPage
	if err := json.Unmarshal(encoded, &entry); err != nil || entry.Format != cacheFormat {
		return cachedPage{}, false
	}
	return entry, true
}

// writeCache stores page content. Caching is best effort: a failed write
// only means the next load hits the API again, so the error is dropped.
func (dp *DetailPage) writeCache(ctx context.Context, pageID, markdown string, truncated bool) {
	if dp.cache == nil {
		return
	}
	_ = dp.cache.Set(ctx, cache.SetInput{
		PageID: pageID,
		Data:   cachedPage{Format: cacheFormat, Markdown: markdown, Truncated: truncated},
		TTL:    time.Hour,
	})
}

// storeInCache updates the cache for the current page after a save.
func (dp *DetailPage) storeInCache(markdown string, truncated bool) {
	dp.writeCache(context.Background(), dp.pageID, markdown, truncated)
}

// Busy reports whether an edit, save, create or export is in progress. The
// root model keeps the page in front until it finishes, so its result is
// seen and the editor file is not orphaned.
func (dp *DetailPage) Busy() bool {
	return dp.busy
}

// CapturesInput reports whether the page needs every key press. The detail
// page never does: while the editor runs, the TUI itself is suspended.
func (dp *DetailPage) CapturesInput() bool {
	return false
}

// PageID returns the current page ID.
func (dp *DetailPage) PageID() string {
	return dp.pageID
}

// Page returns the current page metadata.
func (dp *DetailPage) Page() *notionapi.Page {
	return dp.page
}

// Markdown returns the page content as Notion-flavored Markdown.
func (dp *DetailPage) Markdown() string {
	return dp.markdown
}

// Title returns the page title, or "Untitled".
func (dp *DetailPage) Title() string {
	return notion.PageTitle(dp.page)
}

// IsLoading returns whether the page is currently loading.
func (dp *DetailPage) IsLoading() bool {
	return dp.loading
}

// Error returns the current error, if any.
func (dp *DetailPage) Error() error {
	return dp.err
}

// newEditSession writes content to a file in a private temp directory.
func newEditSession(title, content string) (*editSession, error) {
	dir, err := os.MkdirTemp("", "notion-tui-")
	if err != nil {
		return nil, fmt.Errorf("create temp directory: %w", err)
	}
	path := filepath.Join(dir, fileSlug(title)+".md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("write temp file: %w", err)
	}
	return &editSession{dir: dir, path: path, original: content}, nil
}

// writeExport writes "# title" plus body to a new file in dir and returns its
// path. An existing file is never overwritten; a numbered name is used instead.
func writeExport(dir, title, body string) (string, error) {
	if dir == "" {
		dir = "."
	}
	content := "# " + title + "\n\n" + body + "\n"

	slug := fileSlug(title)
	for attempt := 0; attempt < 1000; attempt++ {
		name := slug + ".md"
		if attempt > 0 {
			name = fmt.Sprintf("%s-%d.md", slug, attempt)
		}
		path := filepath.Join(dir, name)

		// O_EXCL makes "create only if absent" atomic.
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644) //nolint:gosec // An export is a normal user document, not a secret
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("create %s: %w", path, err)
		}
		_, writeErr := file.WriteString(content)
		closeErr := file.Close()
		if writeErr != nil {
			return "", fmt.Errorf("write %s: %w", path, writeErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close %s: %w", path, closeErr)
		}
		return path, nil
	}
	return "", fmt.Errorf("no free file name for %q in %s", slug, dir)
}

// fileSlug turns a page title into a safe file name without extension.
func fileSlug(title string) string {
	const maxLength = 60

	var sb strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			sb.WriteRune(r)
			lastDash = false
		case !lastDash:
			sb.WriteByte('-')
			lastDash = true
		}
		if sb.Len() >= maxLength {
			break
		}
	}

	slug := strings.Trim(sb.String(), "-")
	if slug == "" {
		return "page"
	}
	return slug
}
