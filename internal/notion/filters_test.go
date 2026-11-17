package notion

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func filterTestDatabase() *notionapi.Database {
	options := func(names ...string) []notionapi.Option {
		out := make([]notionapi.Option, len(names))
		for i, name := range names {
			out[i] = notionapi.Option{Name: name}
		}
		return out
	}
	return &notionapi.Database{Properties: notionapi.PropertyConfigs{
		"Name":   &notionapi.TitlePropertyConfig{},
		"Notes":  &notionapi.RichTextPropertyConfig{},
		"Due":    &notionapi.DatePropertyConfig{},
		"Done":   &notionapi.CheckboxPropertyConfig{},
		"Tags":   &notionapi.MultiSelectPropertyConfig{MultiSelect: notionapi.Select{Options: options("home", "work")}},
		"Size":   &notionapi.SelectPropertyConfig{Select: notionapi.Select{Options: options("S", "L")}},
		"Status": &notionapi.StatusPropertyConfig{Status: notionapi.StatusConfig{Options: options("Todo", "Done")}},
	}}
}

// requestJSON is what Notion would receive for the first page of rows.
func requestJSON(t *testing.T, filter RowFilter) string {
	t.Helper()
	raw, err := json.Marshal(filter.Query(""))
	require.NoError(t, err)
	return string(raw)
}

func findChoice(t *testing.T, properties []FilterProperty, label string) RowFilter {
	t.Helper()
	for _, property := range properties {
		for _, choice := range property.Choices {
			if choice.Label == label {
				return choice
			}
		}
	}
	require.Failf(t, "no such choice", "%q", label)
	return RowFilter{}
}

func TestFilterProperties(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 11, 17, 15, 30, 0, 0, time.FixedZone("WITA", 8*3600))
	properties := FilterProperties(filterTestDatabase(), now)

	var names []string
	for _, property := range properties {
		names = append(names, property.Name)
	}
	assert.Equal(t, []string{"Status", "Size", "Due", "Done", "Tags"}, names, "title and text are left out; status first")

	var statusChoices []string
	for _, choice := range properties[0].Choices {
		statusChoices = append(statusChoices, choice.Choice)
	}
	assert.Equal(t, []string{"is Todo", "is Done", "is empty"}, statusChoices, "options in the database's order")

	tests := []struct {
		label string
		want  string
	}{
		{"Status is Done", `{"filter":{"property":"Status","status":{"equals":"Done"}}}`},
		{"Status is empty", `{"filter":{"property":"Status","status":{"is_empty":true}}}`},
		{"Size is L", `{"filter":{"property":"Size","select":{"equals":"L"}}}`},
		{"Tags has work", `{"filter":{"property":"Tags","multi_select":{"contains":"work"}}}`},
		{"Done is checked", `{"filter":{"property":"Done","checkbox":{"equals":true}}}`},
		{"Done is unchecked", `{"filter":{"property":"Done","checkbox":{"does_not_equal":true}}}`},
		{"Due is before today", `{"filter":{"property":"Due","date":{"before":"2026-11-17T00:00:00+08:00"}}}`},
		{"Due is today", `{"filter":{"and":[{"property":"Due","date":{"on_or_after":"2026-11-17T00:00:00+08:00"}},{"property":"Due","date":{"before":"2026-11-18T00:00:00+08:00"}}]}}`},
		{"Due is in the next 7 days", `{"filter":{"and":[{"property":"Due","date":{"on_or_after":"2026-11-17T00:00:00+08:00"}},{"property":"Due","date":{"before":"2026-11-24T00:00:00+08:00"}}]}}`},
		{"Due is after today", `{"filter":{"property":"Due","date":{"on_or_after":"2026-11-18T00:00:00+08:00"}}}`},
		{"Due is set", `{"filter":{"property":"Due","date":{"is_not_empty":true}}}`},
		{"Due is empty", `{"filter":{"property":"Due","date":{"is_empty":true}}}`},
	}
	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			t.Parallel()
			assert.JSONEq(t, tt.want, requestJSON(t, findChoice(t, properties, tt.label)))
		})
	}
}

func TestFilterProperties_NothingToFilterBy(t *testing.T) {
	t.Parallel()

	assert.Empty(t, FilterProperties(nil, time.Now()))
	assert.Empty(t, FilterProperties(&notionapi.Database{Properties: notionapi.PropertyConfigs{"Name": &notionapi.TitlePropertyConfig{}}}, time.Now()))
}

func TestRowFilter_Query(t *testing.T) {
	t.Parallel()

	var none RowFilter
	assert.False(t, none.Active())
	assert.Nil(t, none.Query(""), "no filter and no cursor: the plain first page")
	assert.Equal(t, notionapi.Cursor("c1"), none.Query("c1").StartCursor)

	done := findChoice(t, FilterProperties(filterTestDatabase(), time.Now()), "Status is Done")
	assert.True(t, done.Active())
	next := done.Query("c2")
	assert.Equal(t, notionapi.Cursor("c2"), next.StartCursor)
	assert.NotNil(t, next.Filter, "the next page keeps the filter")
}
