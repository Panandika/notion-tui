package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

const (
	defaultBaseURL = "https://api.notion.com"

	// markdownAPIVersion is the Notion-Version sent to the page Markdown
	// endpoints. They are newer than the version the block library targets.
	markdownAPIVersion = "2026-03-11"

	// markdownFormat selects Notion-flavored Markdown ("v2"); "v1" is deprecated.
	markdownFormat = "v2"

	// maxRateLimitRetries bounds how often a rate-limited call is repeated.
	maxRateLimitRetries = 3

	// maxRetryAfter caps how long a single rate-limit wait may be.
	maxRetryAfter = 60 * time.Second

	// maxErrorBody bounds how much of an error response is read.
	maxErrorBody = 1 << 20
)

// PageMarkdown is a page's content as Notion-flavored Markdown.
type PageMarkdown struct {
	ID       string `json:"id"`
	Markdown string `json:"markdown"`
	// Truncated is true when the page was too large (or partly not shared)
	// and some content is only present as <unknown> placeholders.
	Truncated       bool              `json:"truncated"`
	UnknownBlockIDs []string          `json:"unknown_block_ids"`
	Warnings        []MarkdownWarning `json:"warnings,omitempty"`
}

// MarkdownWarning reports an edit that the API skipped.
type MarkdownWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ContentUpdate is one search-and-replace edit on a page's Markdown.
// OldStr must match exactly one place in the page.
type ContentUpdate struct {
	OldStr string `json:"old_str"`
	NewStr string `json:"new_str"`
}

// CreatedPage identifies a newly created page.
type CreatedPage struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// GetPageMarkdown returns the whole page, nested content included, in one call.
func (c *Client) GetPageMarkdown(ctx context.Context, id string) (*PageMarkdown, error) {
	query := url.Values{"markdown_version": {markdownFormat}}
	var out PageMarkdown
	if err := c.doJSON(ctx, http.MethodGet, markdownPath(id), query, nil, &out); err != nil {
		return nil, fmt.Errorf("get page markdown %s: %w", id, err)
	}
	return &out, nil
}

// UpdatePageMarkdown applies search-and-replace edits to a page. Content the
// edits do not touch keeps its blocks, so comments and child pages survive.
func (c *Client) UpdatePageMarkdown(ctx context.Context, id string, updates []ContentUpdate) (*PageMarkdown, error) {
	if err := c.writable(); err != nil {
		return nil, err
	}
	if len(updates) == 0 {
		return nil, errors.New("update page markdown: no edits given")
	}
	body := map[string]any{
		"markdown_version": markdownFormat,
		"type":             "update_content",
		"update_content":   map[string]any{"content_updates": updates},
	}
	var out PageMarkdown
	if err := c.doJSON(ctx, http.MethodPatch, markdownPath(id), nil, body, &out); err != nil {
		return nil, fmt.Errorf("update page markdown %s: %w", id, err)
	}
	return &out, nil
}

// ReplacePageMarkdown replaces a page's whole content. The API refuses to
// delete child pages or databases this way, which is the behavior we want.
func (c *Client) ReplacePageMarkdown(ctx context.Context, id, markdown string) (*PageMarkdown, error) {
	if err := c.writable(); err != nil {
		return nil, err
	}
	body := map[string]any{
		"markdown_version": markdownFormat,
		"type":             "replace_content",
		"replace_content":  map[string]any{"new_str": markdown},
	}
	var out PageMarkdown
	if err := c.doJSON(ctx, http.MethodPatch, markdownPath(id), nil, body, &out); err != nil {
		return nil, fmt.Errorf("replace page markdown %s: %w", id, err)
	}
	return &out, nil
}

// PageParent says where a new page goes: inside a page, or as a row of a
// database. Exactly one field is set.
type PageParent struct {
	PageID     string
	DatabaseID string
}

func (p PageParent) body() (map[string]any, error) {
	switch {
	case p.PageID != "" && p.DatabaseID == "":
		return map[string]any{"page_id": p.PageID}, nil
	case p.DatabaseID != "" && p.PageID == "":
		return map[string]any{"database_id": p.DatabaseID}, nil
	default:
		return nil, errors.New("a new page needs exactly one parent: a page or a database")
	}
}

// CreatePageMarkdown creates a page under parent from Markdown written by
// the user. A leading "# heading" is the page title (the title property, for
// a database row) and the rest is the content.
//
// The title is always sent explicitly. Notion only derives it from the
// heading when body text follows; a heading on its own would be left in the
// content and the page would have no title.
func (c *Client) CreatePageMarkdown(ctx context.Context, parent PageParent, markdown string) (*CreatedPage, error) {
	if err := c.writable(); err != nil {
		return nil, err
	}
	parentBody, err := parent.body()
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	body := map[string]any{"parent": parentBody}
	title, content := SplitTitle(markdown)
	if title != "" {
		body["properties"] = map[string]any{
			"title": map[string]any{"title": []map[string]any{{"text": map[string]any{"content": title}}}},
		}
	}
	if strings.TrimSpace(content) != "" {
		body["markdown"] = content
		body["markdown_version"] = markdownFormat
	}

	var out CreatedPage
	if err := c.doJSON(ctx, http.MethodPost, "/v1/pages", nil, body, &out); err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	return &out, nil
}

// SplitTitle separates a leading "# heading" from the rest of the Markdown.
// The title is returned as plain text; without such a heading it is empty
// and content is the whole text.
func SplitTitle(markdown string) (title, content string) {
	text := NormalizeMarkdown(markdown)
	trimmed := strings.TrimLeft(text, "\n")
	first, rest, _ := strings.Cut(trimmed, "\n")
	if !strings.HasPrefix(first, "# ") {
		return "", text
	}
	title = strings.TrimSpace(UnescapeMarkdown(strings.TrimPrefix(first, "# ")))
	if title == "" {
		return "", text
	}
	return title, strings.TrimLeft(rest, "\n")
}

func markdownPath(pageID string) string {
	return "/v1/pages/" + url.PathEscape(pageID) + "/markdown"
}

// doJSON sends one JSON request with rate limiting and decodes the response.
// A rate-limited call is repeated after the wait the server asks for.
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
	}

	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	for attempt := 0; ; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter wait: %w", err)
		}

		retryAfter, err := c.sendJSON(ctx, method, target, payload, out)
		if err == nil {
			return nil
		}
		if retryAfter <= 0 || attempt >= maxRateLimitRetries {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryAfter):
		}
	}
}

// sendJSON performs a single request. A positive retryAfter means the server
// rate limited the call and it may be repeated after that wait.
func (c *Client) sendJSON(ctx context.Context, method, target string, payload []byte, out any) (retryAfter time.Duration, err error) {
	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", markdownAPIVersion)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close() //nolint:errcheck // Nothing useful to do if closing a read body fails

	if resp.StatusCode >= http.StatusBadRequest {
		apiErr := decodeAPIError(resp)
		if resp.StatusCode == http.StatusTooManyRequests {
			return parseRetryAfterHeader(resp.Header.Get("Retry-After")), apiErr
		}
		return 0, apiErr
	}

	if out == nil {
		return 0, nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return 0, nil
}

// decodeAPIError turns an error response into the library's error type, so
// existing status-based handling (auth, not found, rate limit) applies.
func decodeAPIError(resp *http.Response) error {
	apiErr := &notionapi.Error{Status: resp.StatusCode}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if err := json.Unmarshal(raw, apiErr); err != nil || apiErr.Message == "" {
		apiErr.Message = fmt.Sprintf("notion API returned HTTP %d", resp.StatusCode)
	}
	apiErr.Status = resp.StatusCode
	return apiErr
}

// parseRetryAfterHeader reads a Retry-After value in seconds, bounded to a
// sane range. A missing or unreadable header falls back to one second.
func parseRetryAfterHeader(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 1 {
		return time.Second
	}
	wait := time.Duration(seconds) * time.Second
	if wait > maxRetryAfter {
		return maxRetryAfter
	}
	return wait
}
