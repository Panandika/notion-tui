package notion

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dateOf(t *testing.T, value string) *notionapi.Date {
	t.Helper()
	layout := dateLayout
	if len(value) > len(dateLayout) {
		layout = time.RFC3339
	}
	parsed, err := time.Parse(layout, value)
	require.NoError(t, err)
	date := notionapi.Date(parsed)
	return &date
}

func TestPageProperties(t *testing.T) {
	t.Parallel()

	prefix := "TASK"
	page := &notionapi.Page{Properties: notionapi.Properties{
		"Name":     &notionapi.TitleProperty{Type: notionapi.PropertyTypeTitle, Title: []notionapi.RichText{{PlainText: "Row"}}},
		"Status":   &notionapi.StatusProperty{Type: notionapi.PropertyTypeStatus, Status: notionapi.Status{Name: "In progress"}},
		"Priority": &notionapi.SelectProperty{Type: notionapi.PropertyTypeSelect},
		"Due":      &notionapi.DateProperty{Type: notionapi.PropertyTypeDate, Date: &notionapi.DateObject{Start: dateOf(t, "2026-11-12")}},
		"Done":     &notionapi.CheckboxProperty{Type: notionapi.PropertyTypeCheckbox, Checkbox: true},
		"Tags":     &notionapi.MultiSelectProperty{Type: notionapi.PropertyTypeMultiSelect, MultiSelect: []notionapi.Option{{Name: "a"}, {Name: "b"}}},
		"Owner":    &notionapi.PeopleProperty{Type: notionapi.PropertyTypePeople, People: []notionapi.User{{Name: "Ada"}, {Name: "Lin"}}},
		"ID":       &notionapi.UniqueIDProperty{Type: notionapi.PropertyTypeUniqueID, UniqueID: notionapi.UniqueID{Prefix: &prefix, Number: 7}},
		"Notes":    &notionapi.RichTextProperty{Type: notionapi.PropertyTypeRichText, RichText: []notionapi.RichText{{PlainText: "hello"}}},
	}}

	values := PageProperties(page)

	byName := map[string]PropertyValue{}
	var order []string
	for _, value := range values {
		byName[value.Name] = value
		order = append(order, value.Name)
	}
	assert.NotContains(t, byName, "Name", "the title is shown as the page heading, not as a property")
	assert.Equal(t, []string{"Status", "Priority", "Due", "Done", "ID", "Notes", "Owner", "Tags"}, order,
		"actionable properties first, the rest alphabetical")
	assert.Equal(t, "In progress", byName["Status"].Text)
	assert.Empty(t, byName["Priority"].Text, "an unset select has no text")
	assert.Equal(t, "2026-11-12", byName["Due"].Text)
	assert.True(t, byName["Done"].Checked)
	assert.Equal(t, "a, b", byName["Tags"].Text)
	assert.Equal(t, "Ada, Lin", byName["Owner"].Text)
	assert.Equal(t, "TASK-7", byName["ID"].Text)
	assert.Equal(t, "hello", byName["Notes"].Text)

	assert.True(t, byName["Status"].Editable())
	assert.True(t, byName["Priority"].Editable())
	assert.True(t, byName["Due"].Editable())
	assert.True(t, byName["Done"].Editable())
	assert.False(t, byName["Owner"].Editable())
	assert.False(t, byName["Tags"].Editable())

	assert.Nil(t, PageProperties(nil))
}

func TestFormatDateRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date *notionapi.DateObject
		want string
	}{
		{name: "unset", date: nil, want: ""},
		{name: "date only", date: &notionapi.DateObject{Start: dateOf(t, "2026-11-12")}, want: "2026-11-12"},
		{name: "with time", date: &notionapi.DateObject{Start: dateOf(t, "2026-11-12T09:30:00Z")}, want: "2026-11-12 09:30"},
		{name: "range", date: &notionapi.DateObject{Start: dateOf(t, "2026-11-12"), End: dateOf(t, "2026-11-14")}, want: "2026-11-12 → 2026-11-14"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, formatDateRange(tt.date))
		})
	}
}

func TestPropertyOptions(t *testing.T) {
	t.Parallel()

	database := &notionapi.Database{Properties: notionapi.PropertyConfigs{
		"Status": &notionapi.StatusPropertyConfig{Status: notionapi.StatusConfig{Options: []notionapi.Option{{Name: "Not started"}, {Name: "Done"}}}},
		"Size":   &notionapi.SelectPropertyConfig{Select: notionapi.Select{Options: []notionapi.Option{{Name: "S"}, {Name: "L"}}}},
		"Due":    &notionapi.DatePropertyConfig{},
	}}

	assert.Equal(t, []string{"Not started", "Done"}, PropertyOptions(database, "Status"), "database order is kept")
	assert.Equal(t, []string{"S", "L"}, PropertyOptions(database, "Size"))
	assert.Empty(t, PropertyOptions(database, "Due"))
	assert.Empty(t, PropertyOptions(database, "Missing"))
	assert.Empty(t, PropertyOptions(nil, "Status"))
}

func TestDateUpdate_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{name: "plain date", text: "2026-12-01"},
		{name: "spaces around", text: "  2026-12-01 "},
		{name: "empty clears", text: ""},
		{name: "wrong order", text: "01-12-2026", wantErr: true},
		{name: "impossible date", text: "2026-02-30", wantErr: true},
		{name: "words", text: "tomorrow", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := DateUpdate("Due", tt.text)
			assert.Equal(t, tt.wantErr, err != nil, "err=%v", err)
		})
	}
}

func TestSetPageProperty_RequestBodies(t *testing.T) {
	t.Parallel()

	date, err := DateUpdate("Due", "2026-12-01")
	require.NoError(t, err)
	cleared, err := DateUpdate("Due", "")
	require.NoError(t, err)

	tests := []struct {
		name   string
		update PropertyUpdate
		want   map[string]any
	}{
		{name: "status", update: StatusUpdate("Status", "Done"), want: map[string]any{"Status": map[string]any{"status": map[string]any{"name": "Done"}}}},
		{name: "select", update: SelectUpdate("Size", "L"), want: map[string]any{"Size": map[string]any{"select": map[string]any{"name": "L"}}}},
		{name: "select cleared", update: SelectUpdate("Size", ""), want: map[string]any{"Size": map[string]any{"select": nil}}},
		{name: "checkbox", update: CheckboxUpdate("Done", true), want: map[string]any{"Done": map[string]any{"checkbox": true}}},
		// A bare date, never a timestamp: Notion would show "12:00 AM" otherwise.
		{name: "date without time", update: date, want: map[string]any{"Due": map[string]any{"date": map[string]any{"start": "2026-12-01"}}}},
		{name: "date cleared", update: cleared, want: map[string]any{"Due": map[string]any{"date": nil}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
				_, _ = w.Write([]byte(`{"object":"page","id":"p1"}`))
			})

			require.NoError(t, client.SetPageProperty(context.Background(), "p1", tt.update))

			req := (*requests)[0]
			assert.Equal(t, http.MethodPatch, req.method)
			assert.Equal(t, "/v1/pages/p1", req.path)
			assert.Equal(t, tt.want, req.body["properties"])
			assert.NotContains(t, req.body, "archived", "a property update must not touch the archived flag")
		})
	}
}

func TestSetPageProperty_RejectsEmptyUpdate(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(http.ResponseWriter, int) {})

	assert.Error(t, client.SetPageProperty(context.Background(), "p1", PropertyUpdate{}))
	assert.Empty(t, *requests)
}

func TestCreatePageMarkdown_Parents(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		parent  PageParent
		want    map[string]any
		wantErr bool
	}{
		{name: "inside a page", parent: PageParent{PageID: "p"}, want: map[string]any{"page_id": "p"}},
		{name: "row of a database", parent: PageParent{DatabaseID: "d"}, want: map[string]any{"database_id": "d"}},
		{name: "no parent", parent: PageParent{}, wantErr: true},
		{name: "two parents", parent: PageParent{PageID: "p", DatabaseID: "d"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
				_, _ = w.Write([]byte(`{"object":"page","id":"new"}`))
			})

			_, err := client.CreatePageMarkdown(context.Background(), tt.parent, "# T")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, *requests)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, (*requests)[0].body["parent"])
		})
	}
}
