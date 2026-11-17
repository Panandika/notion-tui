package notion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxComments bounds how many comments of one page are loaded.
const maxComments = 500

// unknownAuthor stands in for a person whose name Notion would not give,
// usually because the connection may not read user information.
const unknownAuthor = "Someone"

// Comment is one open comment on a page.
type Comment struct {
	ID string
	// ThreadID is shared by a comment and the replies to it.
	ThreadID string
	Author   string
	Text     string
	Created  time.Time
}

type commentObject struct {
	ID           string    `json:"id"`
	DiscussionID string    `json:"discussion_id"`
	CreatedTime  time.Time `json:"created_time"`
	CreatedBy    struct {
		ID string `json:"id"`
	} `json:"created_by"`
	RichText []struct {
		PlainText string `json:"plain_text"`
	} `json:"rich_text"`
	DisplayName struct {
		ResolvedName string `json:"resolved_name"`
	} `json:"display_name"`
}

func (o commentObject) text() string {
	var sb strings.Builder
	for _, part := range o.RichText {
		sb.WriteString(part.PlainText)
	}
	return sb.String()
}

// ListComments returns the open comments on a page itself, oldest first,
// with the replies of a thread next to each other. Notion's API leaves out
// resolved comments, and comments attached to a block inside the page.
func (c *Client) ListComments(ctx context.Context, pageID string) ([]Comment, error) {
	var objects []commentObject
	cursor := ""
	for len(objects) < maxComments {
		query := url.Values{"block_id": {pageID}, "page_size": {"100"}}
		if cursor != "" {
			query.Set("start_cursor", cursor)
		}
		var out struct {
			Results    []commentObject `json:"results"`
			HasMore    bool            `json:"has_more"`
			NextCursor string          `json:"next_cursor"`
		}
		if err := c.doJSON(ctx, http.MethodGet, "/v1/comments", query, nil, &out); err != nil {
			return nil, fmt.Errorf("list comments of %s: %w", pageID, err)
		}
		objects = append(objects, out.Results...)
		if !out.HasMore || out.NextCursor == "" {
			break
		}
		cursor = out.NextCursor
	}

	if len(objects) > maxComments {
		objects = objects[:maxComments]
	}

	// Threads in the order their first comment appears; the API already
	// returns comments oldest first.
	var threads []string
	byThread := map[string][]Comment{}
	unknown := map[string]bool{} // People whose name could not be looked up this time
	for _, object := range objects {
		if _, seen := byThread[object.DiscussionID]; !seen {
			threads = append(threads, object.DiscussionID)
		}
		byThread[object.DiscussionID] = append(byThread[object.DiscussionID], Comment{
			ID:       object.ID,
			ThreadID: object.DiscussionID,
			Author:   c.commentAuthor(ctx, object, unknown),
			Text:     object.text(),
			Created:  object.CreatedTime,
		})
	}
	comments := make([]Comment, 0, len(objects))
	for _, thread := range threads {
		comments = append(comments, byThread[thread]...)
	}
	return comments, nil
}

// MaxComments is the most comments ListComments returns for one page.
const MaxComments = maxComments

// commentAuthor names who wrote a comment. Notion gives the name for
// comments written through a connection; for people it gives only an ID,
// which is looked up once and remembered. A lookup that fails is not
// remembered beyond this list (unknown), so a passing network error does
// not hide the name for good.
func (c *Client) commentAuthor(ctx context.Context, object commentObject, unknown map[string]bool) string {
	if name := strings.TrimSpace(object.DisplayName.ResolvedName); name != "" {
		return name
	}
	id := object.CreatedBy.ID
	if id == "" {
		return "Unknown"
	}

	c.userMu.Lock()
	name, known := c.userNames[id]
	c.userMu.Unlock()
	if known {
		return name
	}
	if unknown[id] {
		return unknownAuthor
	}

	var user struct {
		Name string `json:"name"`
	}
	// A failed lookup must not fail the comment list: the text matters more
	err := c.doJSON(ctx, http.MethodGet, "/v1/users/"+url.PathEscape(id), nil, nil, &user)
	if name = strings.TrimSpace(user.Name); err != nil || name == "" {
		unknown[id] = true
		return unknownAuthor
	}

	c.userMu.Lock()
	if c.userNames == nil {
		c.userNames = map[string]string{}
	}
	c.userNames[id] = name
	c.userMu.Unlock()
	return name
}

// AddComment writes a comment on a page, or a reply when threadID names an
// existing thread.
func (c *Client) AddComment(ctx context.Context, pageID, threadID, text string) error {
	if err := c.writable(); err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return errors.New("add comment: the comment is empty")
	}
	body := map[string]any{
		"rich_text": []map[string]any{{"text": map[string]any{"content": text}}},
	}
	if threadID != "" {
		body["discussion_id"] = threadID
	} else {
		body["parent"] = map[string]any{"page_id": pageID}
	}
	if err := c.doJSON(ctx, http.MethodPost, "/v1/comments", nil, body, nil); err != nil {
		return fmt.Errorf("add comment: %w", err)
	}
	return nil
}
