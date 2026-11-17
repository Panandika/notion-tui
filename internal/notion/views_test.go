package notion

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListViews(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, call int) {
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"object":"list","results":[{"object":"view","id":"v1"},{"object":"view","id":"v2"},{"object":"view","id":"v3"}],"has_more":false,"next_cursor":null}`))
		case 2:
			_, _ = w.Write([]byte(`{"object":"view","id":"v1","name":"To Do","type":"list","filter":null}`))
		case 3:
			_, _ = w.Write([]byte(`{"object":"view","id":"v2","name":"New task","type":"form"}`))
		case 4:
			_, _ = w.Write([]byte(`{"object":"view","id":"v3","name":"","type":"board"}`))
		default:
			t.Errorf("unexpected request %d", call)
		}
	})

	views, err := client.ListViews(context.Background(), "db-1")
	require.NoError(t, err)

	assert.Equal(t, []View{
		{ID: "v1", Name: "To Do", Type: "list"},
		{ID: "v3", Name: "Untitled view", Type: "board"},
	}, views, "a form lists no rows and is left out")

	require.Len(t, *requests, 4)
	assert.Equal(t, "/v1/views", (*requests)[0].path)
	assert.Contains(t, (*requests)[0].query, "database_id=db-1")
	assert.Equal(t, markdownAPIVersion, (*requests)[0].version, "views need the newer API version")
	assert.Equal(t, "/v1/views/v1", (*requests)[1].path, "the list only holds IDs; names are asked for one by one")
}

func TestQueryView(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, call int) {
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"object":"view_query","id":"q1","view_id":"v1","total_count":3,
				"results":[{"object":"page","id":"row-b"},{"object":"page","id":"row-a"}],"next_cursor":"row-c","has_more":true}`))
		case 2:
			_, _ = w.Write([]byte(`{"object":"list","type":"page","results":[{"object":"page","id":"row-c"}],"next_cursor":null,"has_more":false}`))
		default:
			t.Errorf("unexpected request %d", call)
		}
	})
	client.SetReadOnly(true) // Running a view changes nothing

	first, err := client.QueryView(context.Background(), "v1")
	require.NoError(t, err)
	assert.Equal(t, &ViewRows{QueryID: "q1", RowIDs: []string{"row-b", "row-a"}, NextCursor: "row-c", HasMore: true}, first)

	next, err := client.ViewRowsAfter(context.Background(), "v1", first.QueryID, first.NextCursor)
	require.NoError(t, err)
	assert.Equal(t, &ViewRows{QueryID: "q1", RowIDs: []string{"row-c"}}, next)

	require.Len(t, *requests, 2)
	assert.Equal(t, http.MethodPost, (*requests)[0].method)
	assert.Equal(t, "/v1/views/v1/queries", (*requests)[0].path)
	assert.EqualValues(t, ViewPageSize, (*requests)[0].body["page_size"])
	assert.Equal(t, http.MethodGet, (*requests)[1].method)
	assert.Equal(t, "/v1/views/v1/queries/q1", (*requests)[1].path)
	assert.Contains(t, (*requests)[1].query, "start_cursor=row-c")
}
