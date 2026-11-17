package notion

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListComments(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, call int) {
		switch call {
		case 1:
			_, _ = w.Write([]byte(`{"results":[
				{"id":"c1","discussion_id":"t1","created_time":"2026-11-17T04:50:00.000Z","created_by":{"id":"u1"},
				 "rich_text":[{"plain_text":"first "},{"plain_text":"comment"}],"display_name":{"type":"integration","resolved_name":"notion-tui"}},
				{"id":"c2","discussion_id":"t2","created_time":"2026-11-17T05:00:00.000Z","created_by":{"id":"u2"},"rich_text":[{"plain_text":"other thread"}]}
			],"has_more":true,"next_cursor":"cur"}`))
		case 2:
			_, _ = w.Write([]byte(`{"results":[
				{"id":"c3","discussion_id":"t1","created_time":"2026-11-17T06:00:00.000Z","created_by":{"id":"u2"},"rich_text":[{"plain_text":"a reply"}]}
			],"has_more":false,"next_cursor":null}`))
		case 3:
			_, _ = w.Write([]byte(`{"object":"user","id":"u2","name":"Ada Lovelace"}`))
		default:
			t.Errorf("unexpected request %d", call)
		}
	})

	comments, err := client.ListComments(context.Background(), "page-1")
	require.NoError(t, err)

	var got []string
	for _, comment := range comments {
		got = append(got, comment.ThreadID+" "+comment.Author+": "+comment.Text)
	}
	assert.Equal(t, []string{
		"t1 notion-tui: first comment",
		"t1 Ada Lovelace: a reply",
		"t2 Ada Lovelace: other thread",
	}, got, "replies follow the comment that opened their thread; a person is looked up once")

	require.Len(t, *requests, 3)
	assert.Equal(t, "/v1/comments", (*requests)[0].path)
	assert.Contains(t, (*requests)[0].query, "block_id=page-1")
	assert.Contains(t, (*requests)[1].query, "start_cursor=cur")
	assert.Equal(t, "/v1/users/u2", (*requests)[2].path, "asked once for two comments by the same person")
}

func TestListComments_AuthorLookupFailureKeepsTheComment(t *testing.T) {
	t.Parallel()

	client, _ := newMarkdownTestClient(t, func(w http.ResponseWriter, call int) {
		if call == 1 {
			_, _ = w.Write([]byte(`{"results":[{"id":"c1","discussion_id":"t1","created_by":{"id":"u9"},"rich_text":[{"plain_text":"hello"}]}],"has_more":false}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"object":"error","status":403,"code":"restricted_resource","message":"no user access"}`))
	})

	comments, err := client.ListComments(context.Background(), "page-1")

	require.NoError(t, err)
	require.Len(t, comments, 1)
	assert.Equal(t, "Someone", comments[0].Author)
	assert.Equal(t, "hello", comments[0].Text)
}

func TestAddComment(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"object":"comment","id":"c"}`))
	})
	ctx := context.Background()

	require.NoError(t, client.AddComment(ctx, "page-1", "", "  on the page  "))
	require.NoError(t, client.AddComment(ctx, "page-1", "thread-7", "a reply"))
	require.Len(t, *requests, 2)

	first := (*requests)[0]
	assert.Equal(t, http.MethodPost, first.method)
	assert.Equal(t, "/v1/comments", first.path)
	assert.Equal(t, map[string]any{"page_id": "page-1"}, first.body["parent"])
	assert.NotContains(t, first.body, "discussion_id")
	assert.Equal(t, []any{map[string]any{"text": map[string]any{"content": "on the page"}}}, first.body["rich_text"])

	second := (*requests)[1]
	assert.Equal(t, "thread-7", second.body["discussion_id"])
	assert.NotContains(t, second.body, "parent", "a reply names the thread, not the page")

	err := client.AddComment(ctx, "page-1", "", "   ")
	assert.ErrorContains(t, err, "empty")

	client.SetReadOnly(true)
	assert.ErrorIs(t, client.AddComment(ctx, "page-1", "", "blocked"), ErrReadOnly)
	assert.Len(t, *requests, 2, "nothing more reached Notion")
	assert.True(t, strings.HasPrefix(first.auth, "Bearer "))
}
