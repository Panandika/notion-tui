package pages

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Panandika/notion-tui/internal/notion"
)

// rowPage builds a database row with one property of each editable kind.
func rowPage(status, size string, done bool) *notionapi.Page {
	return &notionapi.Page{
		Parent: notionapi.Parent{Type: notionapi.ParentTypeDatabaseID, DatabaseID: "db-1"},
		Properties: notionapi.Properties{
			"Name":   &notionapi.TitleProperty{Type: notionapi.PropertyTypeTitle, Title: []notionapi.RichText{{PlainText: "Ship it"}}},
			"Status": &notionapi.StatusProperty{Type: notionapi.PropertyTypeStatus, Status: notionapi.Status{Name: status}},
			"Size":   &notionapi.SelectProperty{Type: notionapi.PropertyTypeSelect, Select: notionapi.Option{Name: size}},
			"Due":    &notionapi.DateProperty{Type: notionapi.PropertyTypeDate},
			"Done":   &notionapi.CheckboxProperty{Type: notionapi.PropertyTypeCheckbox, Checkbox: done},
			"Owner":  &notionapi.PeopleProperty{Type: notionapi.PropertyTypePeople, People: []notionapi.User{{Name: "Ada"}}},
		},
	}
}

func rowSchema() *notionapi.Database {
	return &notionapi.Database{ID: "db-1", Properties: notionapi.PropertyConfigs{
		"Status": &notionapi.StatusPropertyConfig{Status: notionapi.StatusConfig{Options: []notionapi.Option{{Name: "Not started"}, {Name: "In progress"}, {Name: "Done"}}}},
		"Size":   &notionapi.SelectPropertyConfig{Select: notionapi.Select{Options: []notionapi.Option{{Name: "S"}, {Name: "L"}}}},
	}}
}

// newRowFixture returns a loaded detail page showing a database row.
func newRowFixture(t *testing.T, markdown string) detailFixture {
	t.Helper()
	fx := newDetailFixture(t, markdown)
	fx.client.PageToReturn = rowPage("Not started", "", false)
	fx.client.DatabaseSchemaToReturn = rowSchema()
	fx.drain(t, fx.page.Refresh())
	require.Equal(t, "Ship it", fx.page.Title())
	return fx
}

func (fx detailFixture) press(t *testing.T, keys ...tea.KeyMsg) {
	t.Helper()
	for _, key := range keys {
		_, cmd := fx.page.Update(key)
		fx.drain(t, cmd)
	}
}

var (
	enterKey = tea.KeyMsg{Type: tea.KeyEnter}
	escKey   = tea.KeyMsg{Type: tea.KeyEsc}
	spaceKey = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
)

func lastPropertyUpdate(t *testing.T, fx detailFixture) notion.PropertyUpdate {
	t.Helper()
	require.NotEmpty(t, fx.client.SetPagePropertyCalls)
	call := fx.client.SetPagePropertyCalls[len(fx.client.SetPagePropertyCalls)-1]
	assert.Equal(t, "page-1", call.PageID)
	return call.Update
}

func TestPageHeader(t *testing.T) {
	t.Parallel()

	page := rowPage("In progress", "", true)

	header := pageHeader(page)

	assert.Equal(t, "# Ship it\n\n**Status:** In progress  ·  ☑ Done  ·  **Owner:** Ada", header,
		"title, then set properties; an unset select and date are left out")
	assert.Empty(t, pageHeader(nil))
}

func TestDetailPage_ShowsTitleAndPropertiesAboveContent(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body text")

	assert.Equal(t, "# Ship it\n\n**Status:** Not started  ·  ☐ Done  ·  **Owner:** Ada\n\n---\n\nbody text", fx.viewer.markdown)
}

func TestDetailPage_Properties_ChangeStatus(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")
	fx.client.SetPagePropertyFunc = func(context.Context, string, notion.PropertyUpdate) error {
		fx.client.PageToReturn = rowPage("Done", "", false)
		return nil
	}

	fx.press(t, keyRune('p'))
	require.True(t, fx.page.CapturesInput())
	view := fx.page.View()
	assert.Contains(t, view, "Properties")
	assert.Contains(t, view, "Status")
	assert.NotContains(t, view, "Owner", "people cannot be changed here, so they are not offered")

	// Status is the first property: open its options, pick the last one
	fx.press(t, enterKey)
	assert.Contains(t, fx.page.View(), "In progress", "options come from the database")
	fx.press(t, keyRune('G'), enterKey)

	assert.Equal(t, notion.StatusUpdate("Status", "Done"), lastPropertyUpdate(t, fx))
	assert.Equal(t, modeProperties, fx.page.mode, "back at the property list after saving")
	assert.Contains(t, fx.page.View(), "Done")
	assert.Contains(t, fx.viewer.markdown, "**Status:** Done", "the page header follows")
	assert.False(t, fx.page.Busy())

	fx.press(t, escKey)
	assert.False(t, fx.page.CapturesInput())
}

func TestDetailPage_Properties_OptionsOpenOnCurrentValue(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")
	fx.client.PageToReturn = rowPage("In progress", "", false)
	fx.drain(t, fx.page.Refresh())

	fx.press(t, keyRune('p'), enterKey)

	assert.Equal(t, 1, fx.page.picker.Cursor(), "the cursor starts on the current status")
}

func TestDetailPage_Properties_SelectCanBeSetAndCleared(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")
	fx.client.SetPagePropertyFunc = func(context.Context, string, notion.PropertyUpdate) error {
		fx.client.PageToReturn = rowPage("Not started", "L", false)
		return nil
	}

	// Size is the second property
	fx.press(t, keyRune('p'), keyRune('j'), enterKey)
	assert.Contains(t, fx.page.View(), clearOption)
	fx.press(t, keyRune('G'), enterKey)
	assert.Equal(t, notion.SelectUpdate("Size", "L"), lastPropertyUpdate(t, fx))

	fx.press(t, enterKey, keyRune('g'), enterKey)
	assert.Equal(t, notion.SelectUpdate("Size", ""), lastPropertyUpdate(t, fx), "the first entry clears the select")
}

func TestDetailPage_Properties_CheckboxTogglesAtOnce(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")

	// Order: Status, Size, Due, Done
	fx.press(t, keyRune('p'), keyRune('G'), spaceKey)

	assert.Equal(t, notion.CheckboxUpdate("Done", true), lastPropertyUpdate(t, fx))
}

func TestDetailPage_Properties_Date(t *testing.T) {
	t.Parallel()

	typeText := func(fx detailFixture, text string) {
		for _, r := range text {
			fx.page.Update(keyRune(r))
		}
	}

	t.Run("valid date is saved without a time", func(t *testing.T) {
		t.Parallel()
		fx := newRowFixture(t, "body")
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)
		require.Equal(t, modeDate, fx.page.mode)

		typeText(fx, "2026-12-01")
		fx.press(t, enterKey)

		want, err := notion.DateUpdate("Due", "2026-12-01")
		require.NoError(t, err)
		assert.Equal(t, want, lastPropertyUpdate(t, fx))
	})

	t.Run("invalid date is refused and can be corrected", func(t *testing.T) {
		t.Parallel()
		fx := newRowFixture(t, "body")
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)

		typeText(fx, "tomorrow")
		fx.press(t, enterKey)

		assert.Empty(t, fx.client.SetPagePropertyCalls, "nothing is sent for an invalid date")
		assert.Equal(t, modeDate, fx.page.mode, "the prompt stays open")
		assert.Contains(t, fx.page.View(), "YYYY-MM-DD")
	})

	t.Run("letters typed in the prompt are text, not shortcuts", func(t *testing.T) {
		t.Parallel()
		fx := newRowFixture(t, "body")
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)

		typeText(fx, "q")

		assert.Equal(t, modeDate, fx.page.mode)
		assert.Equal(t, "q", fx.page.dateInput.Value())
	})

	t.Run("esc returns to the property list without saving", func(t *testing.T) {
		t.Parallel()
		fx := newRowFixture(t, "body")
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)
		typeText(fx, "2026-12-01")

		fx.press(t, escKey)

		assert.Equal(t, modeProperties, fx.page.mode)
		assert.Equal(t, 2, fx.page.picker.Cursor(), "back on the row that was being changed")
		assert.Empty(t, fx.client.SetPagePropertyCalls)
	})
}

func TestDetailPage_Properties_SaveFailureIsShownAndRecoverable(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")
	fx.client.SetPagePropertyFunc = func(context.Context, string, notion.PropertyUpdate) error {
		return errors.New("forbidden")
	}

	fx.press(t, keyRune('p'), enterKey, keyRune('j'), enterKey)

	assert.Contains(t, fx.page.View(), "forbidden")
	assert.False(t, fx.page.Busy(), "the page must not stay locked after a failed save")
	assert.Contains(t, fx.viewer.markdown, "**Status:** Not started", "the header still shows what Notion has")

	fx.press(t, escKey, escKey)
	assert.False(t, fx.page.CapturesInput())
}

func TestDetailPage_Properties_NotOfferedOutsideDatabases(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "body") // A plain page, not a row

	fx.press(t, keyRune('p'))

	assert.False(t, fx.page.CapturesInput())
	assert.Contains(t, fx.page.notice, "not a database row")
	assert.Empty(t, fx.client.GetDatabaseCalls)
}

func TestDetailPage_Properties_SchemaIsFetchedOnce(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")

	fx.press(t, keyRune('p'), escKey, keyRune('p'))

	assert.Len(t, fx.client.GetDatabaseCalls, 1)
}

func TestParseTodos(t *testing.T) {
	t.Parallel()

	markdown := "intro\n- [ ] first\n  - [x] nested <b>bold</b>\n- plain bullet\n```\n- [ ] inside code\n```\n- [X] capital\n- [ ] "

	todos := parseTodos(markdown)

	require.Len(t, todos, 4, "plain bullets and code blocks are not to-dos")
	assert.Equal(t, todoItem{line: 1, checked: false, text: "first"}, todos[0])
	assert.Equal(t, todoItem{line: 2, checked: true, text: "nested <b>bold</b>"}, todos[1])
	assert.True(t, todos[2].checked)
	assert.Equal(t, 8, todos[3].line)
}

func TestToggleTodoLine(t *testing.T) {
	t.Parallel()

	markdown := "- [ ] first\n  - [x] nested\ntext"

	checked, ok := toggleTodoLine(markdown, 0)
	require.True(t, ok)
	assert.Equal(t, "- [x] first\n  - [x] nested\ntext", checked)

	unchecked, ok := toggleTodoLine(markdown, 1)
	require.True(t, ok)
	assert.Equal(t, "- [ ] first\n  - [ ] nested\ntext", unchecked, "indentation is kept")

	for _, line := range []int{2, -1, 99} {
		same, ok := toggleTodoLine(markdown, line)
		assert.False(t, ok)
		assert.Equal(t, markdown, same)
	}
}

func TestDetailPage_Todos_SpaceTogglesOnlyThatLine(t *testing.T) {
	t.Parallel()

	source := "# Plan\n\n- [ ] write\n- [ ] review\n\nnotes with <b>bold</b>"
	fx := newDetailFixture(t, source)
	fx.client.UpdatePageMarkdownFunc = func(_ context.Context, id string, _ []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		return &notion.PageMarkdown{ID: id, Markdown: "# Plan\n\n- [ ] write\n- [x] review\n\nnotes with <b>bold</b>"}, nil
	}

	fx.press(t, keyRune('t'))
	require.True(t, fx.page.CapturesInput())
	assert.Contains(t, fx.page.View(), "[ ] review")

	fx.press(t, keyRune('j'), spaceKey)

	require.Len(t, fx.client.UpdatePageMarkdownCalls, 1)
	updates := fx.client.UpdatePageMarkdownCalls[0].Updates
	require.Len(t, updates, 1)
	assert.Equal(t, "- [ ] review", updates[0].OldStr)
	assert.Equal(t, "- [x] review", updates[0].NewStr)
	assert.Empty(t, fx.client.ReplacePageMarkdownCalls)

	assert.Contains(t, fx.page.View(), "[x] review", "the list shows the new state")
	assert.Equal(t, 1, fx.page.picker.Cursor(), "the cursor stays on the toggled to-do")
	assert.Contains(t, fx.page.Markdown(), "- [x] review")
	assert.False(t, fx.page.Busy())
}

func TestDetailPage_Todos_IdenticalLinesStillToggleTheRightOne(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "- [ ] \n- [ ] \n- [ ] \nend")
	var sent []notion.ContentUpdate
	fx.client.UpdatePageMarkdownFunc = func(_ context.Context, id string, updates []notion.ContentUpdate) (*notion.PageMarkdown, error) {
		sent = updates
		return &notion.PageMarkdown{ID: id}, nil
	}

	fx.press(t, keyRune('t'), keyRune('j'), spaceKey)

	result, ok := notion.ReplayContentUpdates("- [ ] \n- [ ] \n- [ ] \nend", sent)
	require.True(t, ok, "the edit must be unambiguous for the API")
	assert.Equal(t, "- [ ] \n- [x] \n- [ ] \nend", result)
}

func TestDetailPage_Todos_ConflictAndErrors(t *testing.T) {
	t.Parallel()

	t.Run("page changed in Notion", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "- [ ] a\n\ntext")
		fx.client.UpdatePageMarkdownFunc = func(context.Context, string, []notion.ContentUpdate) (*notion.PageMarkdown, error) {
			return nil, &notionapi.Error{Status: 400, Code: "validation_error", Message: "No matches found for - [ ] a."}
		}

		fx.press(t, keyRune('t'), spaceKey)

		assert.Contains(t, fx.page.View(), "changed in Notion")
		assert.Contains(t, fx.page.Markdown(), "- [ ] a", "local state is not changed on failure")
		assert.False(t, fx.page.Busy())
	})

	t.Run("no to-dos", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "just text")

		fx.press(t, keyRune('t'))

		assert.False(t, fx.page.CapturesInput())
		assert.Contains(t, fx.page.notice, "no to-dos")
	})

	t.Run("truncated page", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "- [ ] a")
		fx.page.truncated = true

		fx.press(t, keyRune('t'))

		assert.False(t, fx.page.CapturesInput())
		assert.Contains(t, fx.page.notice, "too large")
	})
}

func TestDetailPage_Modes_IgnoreKeysWhileSaving(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "- [ ] a\n- [ ] b\n\ntext")

	fx.press(t, keyRune('t'))
	_, first := fx.page.Update(spaceKey)
	require.NotNil(t, first)
	require.True(t, fx.page.Busy())

	_, second := fx.page.Update(spaceKey)
	_, escaped := fx.page.Update(escKey)

	assert.Nil(t, second, "a second toggle must wait for the first")
	assert.Nil(t, escaped)
	assert.Equal(t, modeTodos, fx.page.mode)
}

func TestPageHeader_UserTextIsNotReadAsMarkdown(t *testing.T) {
	t.Parallel()

	page := &notionapi.Page{Properties: notionapi.Properties{
		"Name":  &notionapi.TitleProperty{Type: notionapi.PropertyTypeTitle, Title: []notionapi.RichText{{PlainText: "Plan [draft] *x* a_b"}}},
		"Notes": &notionapi.RichTextProperty{Type: notionapi.PropertyTypeRichText, RichText: []notionapi.RichText{{PlainText: "see <tag> & *star*"}}},
	}}

	header := pageHeader(page)

	assert.Equal(t, "# Plan \\[draft\\] \\*x\\* a\\_b\n\n**Notes:** see \\<tag\\> & \\*star\\*", header)
}

func TestPlainText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     string
	}{
		{name: "notion inline tags", markdown: "Click <b>type </b><b><code>/</code></b> to see", want: "Click type / to see"},
		{name: "links keep their label", markdown: "read the [guides](https://example.com) first", want: "read the guides first"},
		{name: "escaped characters are shown as typed", markdown: `finish \[draft\]`, want: "finish [draft]"},
		{name: "strikethrough", markdown: "<s>old</s> new", want: "old new"},
		{name: "empty", markdown: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, plainText(tt.markdown))
		})
	}
}

func TestParseTodos_CodeFences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     []int // Lines that are to-dos
	}{
		{name: "tilde fence", markdown: "~~~\n- [ ] code\n~~~\n- [ ] real", want: []int{3}},
		{name: "longer fence holds a shorter one", markdown: "````\n```\n- [ ] code\n```\n````\n- [ ] real", want: []int{5}},
		{name: "backtick block holds tildes", markdown: "```\n~~~\n- [ ] code\n```\n- [ ] real", want: []int{4}},
		{name: "indented fence", markdown: "  ```\n  - [ ] code\n  ```\n- [ ] real", want: []int{3}},
		{name: "inline code at line start is not a fence", markdown: "```x``` text\n- [ ] real", want: []int{1}},
		{name: "fence with a language", markdown: "```go\n- [ ] code\n```\n- [ ] real", want: []int{3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var lines []int
			for _, todo := range parseTodos(tt.markdown) {
				lines = append(lines, todo.line)
			}
			assert.Equal(t, tt.want, lines)
		})
	}
}

func TestToggleTodoLine_RefusesLinesInsideCode(t *testing.T) {
	t.Parallel()

	markdown := "```\n- [ ] looks like a to-do\n```\n- [ ] real"

	same, ok := toggleTodoLine(markdown, 1)

	assert.False(t, ok)
	assert.Equal(t, markdown, same)
}

func TestDetailPage_Todos_StaleListNeverTogglesAnotherLine(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "- [ ] first\n- [ ] second")
	fx.press(t, keyRune('t'), keyRune('j')) // cursor on "second"

	// The content changes underneath the open list: a line was added above
	fx.page.Update(pageLoadedMsg{pageID: "page-1", page: fx.page.Page(), markdown: "new intro\n- [ ] first\n- [ ] second"})
	fx.page.todos = []todoItem{{line: 0, text: "first"}, {line: 1, text: "second"}} // as if the list had not followed

	fx.press(t, spaceKey)

	assert.Empty(t, fx.client.UpdatePageMarkdownCalls, "a stale line number must not be used")
	assert.Empty(t, fx.client.ReplacePageMarkdownCalls)
	assert.Contains(t, fx.page.View(), "The page changed")
	assert.Equal(t, []todoItem{{line: 1, text: "first"}, {line: 2, text: "second"}}, fx.page.todos, "the list now matches the page")
}

func TestDetailPage_Todos_ListFollowsAReload(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "- [ ] first")
	fx.press(t, keyRune('t'))

	fx.page.Update(pageLoadedMsg{pageID: "page-1", page: fx.page.Page(), markdown: "intro\n- [x] first\n- [ ] added"})

	assert.Equal(t, []todoItem{{line: 1, checked: true, text: "first"}, {line: 2, text: "added"}}, fx.page.todos)
}

func TestDetailPage_RefreshWaitsWhileAListIsOpen(t *testing.T) {
	t.Parallel()

	fx := newDetailFixture(t, "- [ ] first")
	fx.press(t, keyRune('t'))

	assert.Nil(t, fx.page.Refresh(), "reloading would replace the content the list was built from")
	assert.False(t, fx.page.IsLoading())
	assert.Nil(t, fx.page.Export())
	assert.Nil(t, fx.page.StartNewPage())
	assert.Nil(t, fx.page.StartEdit())
}

func timedRowPage(t *testing.T) *notionapi.Page {
	t.Helper()
	page := rowPage("Not started", "", false)
	start, err := time.Parse(time.RFC3339, "2026-11-12T10:00:00Z")
	require.NoError(t, err)
	end, err := time.Parse("2006-01-02", "2026-11-14")
	require.NoError(t, err)
	startDate, endDate := notionapi.Date(start), notionapi.Date(end)
	page.Properties["Due"] = &notionapi.DateProperty{Type: notionapi.PropertyTypeDate, Date: &notionapi.DateObject{Start: &startDate, End: &endDate}}
	return page
}

func TestDetailPage_Properties_DateWithTimeOrRangeIsNotDroppedSilently(t *testing.T) {
	t.Parallel()

	open := func(t *testing.T) detailFixture {
		fx := newRowFixture(t, "body")
		fx.client.PageToReturn = timedRowPage(t)
		fx.drain(t, fx.page.Refresh())
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)
		require.Equal(t, modeDate, fx.page.mode)
		return fx
	}

	t.Run("enter without changes writes nothing", func(t *testing.T) {
		t.Parallel()
		fx := open(t)

		fx.press(t, enterKey)

		assert.Empty(t, fx.client.SetPagePropertyCalls, "the time and end date must survive just looking at the prompt")
		assert.Equal(t, modeProperties, fx.page.mode)
	})

	t.Run("the prompt says what a change will replace", func(t *testing.T) {
		t.Parallel()
		fx := open(t)

		// The notice may wrap, so compare with whitespace collapsed
		view := strings.Join(strings.Fields(fx.page.View()), " ")

		assert.Contains(t, view, "2026-11-12 10:00 → 2026-11-14")
		assert.Contains(t, view, "replaces the time and end date")
	})

	t.Run("a plain date gets no such warning", func(t *testing.T) {
		t.Parallel()
		fx := newRowFixture(t, "body")
		fx.press(t, keyRune('p'), keyRune('j'), keyRune('j'), enterKey)

		assert.NotContains(t, fx.page.View(), "replaces the time")
	})
}

func TestDetailPage_Properties_ChoosingTheCurrentValueWritesNothing(t *testing.T) {
	t.Parallel()

	fx := newRowFixture(t, "body")

	fx.press(t, keyRune('p'), enterKey, enterKey) // status: cursor starts on the current option

	assert.Empty(t, fx.client.SetPagePropertyCalls)
	assert.Equal(t, modeProperties, fx.page.mode)
}

const (
	linkedPageID = "4169a22b64ff806c9c42d2caf2d80d06"
	linkedDBID   = "b2f9a22b64ff832981390180e2c765dd"
)

func linksMarkdown() string {
	return "See the [guide](https://example.com/guide).\n" +
		"<page url=\"https://app.notion.com/p/" + linkedPageID + "\">Sub page</page>\n" +
		"<database url=\"https://app.notion.com/p/" + linkedDBID + "\">\nTasks\n</database>"
}

func TestDetailPage_Links(t *testing.T) {
	t.Parallel()

	t.Run("a child page opens in the app", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, linksMarkdown())

		fx.press(t, keyRune('o'))
		view := fx.page.View()
		assert.Contains(t, view, "guide")
		assert.Contains(t, view, "Sub page")
		assert.Contains(t, view, "Tasks")

		_, cmd := fx.page.Update(keyRune('j'))
		assert.Nil(t, cmd)
		_, cmd = fx.page.Update(enterKey)
		require.NotNil(t, cmd)
		assert.Equal(t, OpenLinkMsg{ID: linkedPageID}, cmd())
		assert.False(t, fx.page.CapturesInput())
	})

	t.Run("a database opens as a database", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, linksMarkdown())

		fx.press(t, keyRune('o'), keyRune('G'))
		_, cmd := fx.page.Update(enterKey)

		require.NotNil(t, cmd)
		assert.Equal(t, OpenLinkMsg{ID: linkedDBID, Database: true}, cmd())
	})

	t.Run("a web link goes to the browser", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, linksMarkdown())
		var opened []string
		fx.page.openURL = func(url string) error { opened = append(opened, url); return nil }

		fx.press(t, keyRune('o'), enterKey)

		assert.Equal(t, []string{"https://example.com/guide"}, opened)
		assert.Contains(t, fx.page.notice, "Opened in the browser")
	})

	t.Run("a browser failure shows the link so it can be copied", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, linksMarkdown())
		fx.page.openURL = func(string) error { return errors.New("no browser") }

		fx.press(t, keyRune('o'), enterKey)

		assert.Contains(t, fx.page.notice, "https://example.com/guide")
	})

	t.Run("no links", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "plain text")

		fx.press(t, keyRune('o'))

		assert.False(t, fx.page.CapturesInput())
		assert.Contains(t, fx.page.notice, "no links")
	})
}

func TestOpenInBrowser_RefusesAnythingButWebLinks(t *testing.T) {
	t.Parallel()

	for _, target := range []string{"file:///etc/passwd", "javascript:alert(1)", "mailto:a@b.c", "-h", "https://", ""} {
		assert.Error(t, openInBrowser(target), "target %q", target)
	}
}

func TestDetailPage_FindInPage(t *testing.T) {
	t.Parallel()

	typeText := func(fx detailFixture, text string) {
		for _, r := range text {
			fx.page.Update(keyRune(r))
		}
	}

	t.Run("search, then step through matches", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "budget one\n\nbudget two")

		fx.press(t, keyRune('/'))
		require.True(t, fx.page.CapturesInput())
		// Letters that are shortcuts elsewhere are search text here
		typeText(fx, "budget")
		fx.press(t, enterKey)

		assert.Equal(t, "budget", fx.viewer.lastFind)
		assert.Contains(t, fx.page.notice, "Match 1 of 2")
		assert.False(t, fx.page.CapturesInput())

		fx.page.Update(keyRune('n'))
		fx.page.Update(keyRune('N'))
		assert.Equal(t, []int{1, -1}, fx.viewer.findNextDeltas)
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "some text")

		fx.press(t, keyRune('/'))
		typeText(fx, "zzz")
		fx.press(t, enterKey)

		assert.Contains(t, fx.page.notice, `No match for "zzz"`)
	})

	t.Run("n before any search explains itself", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "some text")

		fx.page.Update(keyRune('n'))

		assert.Contains(t, fx.page.notice, "press / to find")
		assert.Empty(t, fx.viewer.findNextDeltas)
	})

	t.Run("esc cancels without searching", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "some text")

		fx.press(t, keyRune('/'))
		typeText(fx, "abc")
		fx.press(t, escKey)

		assert.Zero(t, fx.viewer.findCalls)
		assert.False(t, fx.page.CapturesInput())
	})
}

func TestDetailPage_Archive(t *testing.T) {
	t.Parallel()

	t.Run("asks first, and y archives", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "content")

		fx.press(t, keyRune('D'))
		require.True(t, fx.page.CapturesInput())
		view := fx.page.View()
		assert.Contains(t, view, `Archive "My Page"?`)
		assert.Contains(t, view, "can be restored")
		assert.Zero(t, fx.client.UpdatePageCallCount(), "nothing happens before the answer")

		_, cmd := fx.page.Update(keyRune('y'))
		require.NotNil(t, cmd)
		_, cmd = fx.page.Update(cmd())

		call := fx.client.LastUpdatePageCall()
		require.NotNil(t, call)
		assert.Equal(t, "page-1", call.ID)
		assert.True(t, call.Request.Archived)
		require.NotNil(t, cmd)
		assert.Equal(t, PageArchivedMsg{ID: "page-1"}, cmd())
	})

	t.Run("anything but y keeps the page", func(t *testing.T) {
		t.Parallel()
		for _, key := range []tea.KeyMsg{keyRune('n'), escKey, enterKey, keyRune('x')} {
			fx := newDetailFixture(t, "content")
			fx.press(t, keyRune('D'), key)
			assert.Zero(t, fx.client.UpdatePageCallCount(), "key %q must not archive", key.String())
		}
	})

	t.Run("failure keeps the page open and says why", func(t *testing.T) {
		t.Parallel()
		fx := newDetailFixture(t, "content")
		fx.client.UpdatePageFunc = func(context.Context, string, *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
			return nil, errors.New("forbidden")
		}

		fx.press(t, keyRune('D'), keyRune('y'))

		assert.Contains(t, fx.page.notice, "forbidden")
		assert.False(t, fx.page.Busy())
		assert.False(t, fx.page.CapturesInput())
	})
}
