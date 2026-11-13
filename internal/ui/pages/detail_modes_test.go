package pages

import (
	"context"
	"errors"
	"testing"

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

	fx.press(t, keyRune('p'), enterKey, enterKey)

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
