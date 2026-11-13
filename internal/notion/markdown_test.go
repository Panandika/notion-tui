package notion

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

type recordedRequest struct {
	method  string
	path    string
	query   string
	version string
	auth    string
	body    map[string]any
}

// newMarkdownTestClient returns a client pointed at a test server, plus the
// requests the server received.
func newMarkdownTestClient(t *testing.T, handler func(w http.ResponseWriter, call int)) (*Client, *[]recordedRequest) {
	t.Helper()
	var requests []recordedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := recordedRequest{
			method:  r.Method,
			path:    r.URL.Path,
			query:   r.URL.RawQuery,
			version: r.Header.Get("Notion-Version"),
			auth:    r.Header.Get("Authorization"),
		}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			require.NoError(t, json.Unmarshal(raw, &rec.body))
		}
		requests = append(requests, rec)
		handler(w, len(requests))
	}))
	t.Cleanup(server.Close)

	client := NewClient("test-token")
	client.baseURL = server.URL
	client.limiter = rate.NewLimiter(rate.Inf, 1)
	return client, &requests
}

func TestGetPageMarkdown(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"object":"page_markdown","id":"p1","markdown":"# Hi","truncated":true,"unknown_block_ids":["b1"]}`))
	})

	page, err := client.GetPageMarkdown(context.Background(), "p1")

	require.NoError(t, err)
	assert.Equal(t, "# Hi", page.Markdown)
	assert.True(t, page.Truncated)
	assert.Equal(t, []string{"b1"}, page.UnknownBlockIDs)

	require.Len(t, *requests, 1)
	req := (*requests)[0]
	assert.Equal(t, http.MethodGet, req.method)
	assert.Equal(t, "/v1/pages/p1/markdown", req.path)
	assert.Equal(t, "markdown_version=v2", req.query)
	assert.Equal(t, markdownAPIVersion, req.version)
	assert.Equal(t, "Bearer test-token", req.auth)
}

func TestUpdatePageMarkdown_SendsContentUpdates(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"id":"p1","markdown":"new"}`))
	})

	_, err := client.UpdatePageMarkdown(context.Background(), "p1", []ContentUpdate{{OldStr: "old", NewStr: "new"}})

	require.NoError(t, err)
	req := (*requests)[0]
	assert.Equal(t, http.MethodPatch, req.method)
	assert.Equal(t, "update_content", req.body["type"])
	assert.Equal(t, "v2", req.body["markdown_version"])
	updates := req.body["update_content"].(map[string]any)["content_updates"].([]any)
	assert.Equal(t, map[string]any{"old_str": "old", "new_str": "new"}, updates[0])
}

func TestUpdatePageMarkdown_RejectsEmptyEdits(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(http.ResponseWriter, int) {})

	_, err := client.UpdatePageMarkdown(context.Background(), "p1", nil)

	assert.Error(t, err)
	assert.Empty(t, *requests)
}

func TestReplacePageMarkdown_NeverAllowsDeletingContent(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"id":"p1","markdown":"body"}`))
	})

	_, err := client.ReplacePageMarkdown(context.Background(), "p1", "body")

	require.NoError(t, err)
	replace := (*requests)[0].body["replace_content"].(map[string]any)
	assert.Equal(t, "body", replace["new_str"])
	assert.NotContains(t, replace, "allow_deleting_content", "child pages must never be deleted by a save")
}

func TestCreatePageMarkdown(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		_, _ = w.Write([]byte(`{"object":"page","id":"new-page","url":"https://notion.so/new-page"}`))
	})

	page, err := client.CreatePageMarkdown(context.Background(), PageParent{PageID: "parent-1"}, "# Title\n\nbody")

	require.NoError(t, err)
	assert.Equal(t, "new-page", page.ID)
	req := (*requests)[0]
	assert.Equal(t, http.MethodPost, req.method)
	assert.Equal(t, "/v1/pages", req.path)
	assert.Equal(t, map[string]any{"page_id": "parent-1"}, req.body["parent"])
	assert.Equal(t, "body", req.body["markdown"], "the heading is sent as the title, not as content")
	assert.Equal(t, "Title", titleOf(t, req.body))
}

func TestMarkdownCall_ReturnsAPIError(t *testing.T) {
	t.Parallel()

	client, _ := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"object":"error","status":404,"code":"object_not_found","message":"Could not find page"}`))
	})

	_, err := client.GetPageMarkdown(context.Background(), "missing")

	var apiErr *notionapi.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusNotFound, apiErr.Status)
	assert.Equal(t, "Could not find page", apiErr.Message)
}

func TestMarkdownCall_NonJSONErrorStillReportsStatus(t *testing.T) {
	t.Parallel()

	client, _ := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	})

	_, err := client.GetPageMarkdown(context.Background(), "p1")

	var apiErr *notionapi.Error
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, http.StatusBadGateway, apiErr.Status)
}

func TestMarkdownCall_RetriesAfterRateLimit(t *testing.T) {
	t.Parallel()

	client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, call int) {
		if call == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"object":"error","status":429,"code":"rate_limited","message":"slow down"}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"p1","markdown":"ok"}`))
	})

	page, err := client.GetPageMarkdown(context.Background(), "p1")

	require.NoError(t, err)
	assert.Equal(t, "ok", page.Markdown)
	assert.Len(t, *requests, 2)
}

func TestParseRetryAfterHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "seconds", value: "5", want: "5s"},
		{name: "missing falls back to one second", value: "", want: "1s"},
		{name: "garbage falls back to one second", value: "soon", want: "1s"},
		{name: "huge value is capped", value: "9999", want: "1m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseRetryAfterHeader(tt.value).String())
		})
	}
}

func TestPageTitle(t *testing.T) {
	t.Parallel()

	titled := func(parts ...string) *notionapi.Page {
		var rich []notionapi.RichText
		for _, part := range parts {
			rich = append(rich, notionapi.RichText{PlainText: part})
		}
		return &notionapi.Page{Properties: notionapi.Properties{"Name": &notionapi.TitleProperty{Title: rich}}}
	}

	tests := []struct {
		name string
		page *notionapi.Page
		want string
	}{
		{name: "joins every title segment", page: titled("Q3 ", "Plan"), want: "Q3 Plan"},
		{name: "empty title", page: titled(), want: "Untitled"},
		{name: "blank title", page: titled("  "), want: "Untitled"},
		{name: "nil page", page: nil, want: "Untitled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, PageTitle(tt.page))
		})
	}
}

// titleOf returns the title text of a create-page request body.
func titleOf(t *testing.T, body map[string]any) string {
	t.Helper()
	properties, ok := body["properties"].(map[string]any)
	if !ok {
		return ""
	}
	title := properties["title"].(map[string]any)["title"].([]any)
	require.Len(t, title, 1)
	return title[0].(map[string]any)["text"].(map[string]any)["content"].(string)
}

func TestSplitTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		markdown    string
		wantTitle   string
		wantContent string
	}{
		{name: "heading and body", markdown: "# Notes\n\nfirst line\n", wantTitle: "Notes", wantContent: "first line"},
		{name: "heading only", markdown: "# Notes\n", wantTitle: "Notes", wantContent: ""},
		{name: "blank lines before the heading", markdown: "\n\n# Notes\nbody", wantTitle: "Notes", wantContent: "body"},
		{name: "escaped characters are plain text in the title", markdown: `# Plan \[draft\] \*x\*`, wantTitle: "Plan [draft] *x*", wantContent: ""},
		{name: "no heading", markdown: "just text\nmore", wantTitle: "", wantContent: "just text\nmore"},
		{name: "second-level heading is content", markdown: "## Sub\nbody", wantTitle: "", wantContent: "## Sub\nbody"},
		{name: "empty heading is not a title", markdown: "# \nbody", wantTitle: "", wantContent: "# \nbody"},
		{name: "later headings stay in the content", markdown: "# One\n\n# Two\ntext", wantTitle: "One", wantContent: "# Two\ntext"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			title, content := SplitTitle(tt.markdown)
			assert.Equal(t, tt.wantTitle, title)
			assert.Equal(t, tt.wantContent, content)
		})
	}
}

func TestCreatePageMarkdown_TitleIsAlwaysExplicit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		markdown     string
		wantTitle    string
		wantMarkdown any // nil when no content is sent
	}{
		// Notion leaves a lone heading in the content and the title empty,
		// so the heading must be sent as the title.
		{name: "heading only", markdown: "# Meeting notes", wantTitle: "Meeting notes", wantMarkdown: nil},
		{name: "heading and body", markdown: "# Meeting notes\n\n- item", wantTitle: "Meeting notes", wantMarkdown: "- item"},
		{name: "no heading", markdown: "- item", wantTitle: "", wantMarkdown: "- item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client, requests := newMarkdownTestClient(t, func(w http.ResponseWriter, _ int) {
				_, _ = w.Write([]byte(`{"object":"page","id":"new"}`))
			})

			_, err := client.CreatePageMarkdown(context.Background(), PageParent{PageID: "p"}, tt.markdown)

			require.NoError(t, err)
			body := (*requests)[0].body
			assert.Equal(t, tt.wantTitle, titleOf(t, body))
			assert.Equal(t, tt.wantMarkdown, body["markdown"])
			if tt.wantMarkdown == nil {
				assert.NotContains(t, body, "markdown_version", "the version is only valid together with markdown")
			}
		})
	}
}

func TestEscapeUnescapeRoundTrip(t *testing.T) {
	t.Parallel()

	for _, text := range []string{"Plan [draft] *x* q?", `a\b`, "`x` $y$ a|b {c} 2^3 <tag>", "plain", ""} {
		assert.Equal(t, text, UnescapeMarkdown(EscapeMarkdown(text)))
	}
	assert.Equal(t, `keep \n as is`, UnescapeMarkdown(`keep \n as is`), "a backslash before an ordinary character is kept")
}
