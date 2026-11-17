package notion

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	// maxViews bounds how many views of one database are offered. Each
	// costs a request, as the list itself only holds their IDs.
	maxViews = 30

	// ViewPageSize is how many rows of a view are asked for at a time. A
	// view only says which rows it holds, and a row not already known has
	// to be fetched on its own, so the page is kept small.
	ViewPageSize = 25
)

// View is a view saved on a database in Notion: a named filter and order.
type View struct {
	ID   string
	Name string
	// Type is the layout: table, board, list, calendar and so on.
	Type string
}

// ViewRows is one page of the rows a view shows, in the view's order.
// Notion does not hand out the rows themselves, only which they are.
type ViewRows struct {
	// QueryID names the result set, for fetching its later pages. Notion
	// keeps it for 15 minutes.
	QueryID    string
	RowIDs     []string
	NextCursor string
	HasMore    bool
}

// viewTypesWithoutRows are layouts that do not list a database's rows.
var viewTypesWithoutRows = map[string]bool{"form": true, "chart": true, "dashboard": true}

// ListViews returns the views saved on a database that list its rows, in
// Notion's order. What a view filters by is not available: Notion's API
// applies it but does not tell.
func (c *Client) ListViews(ctx context.Context, databaseID string) ([]View, error) {
	var ids []string
	cursor := ""
	for len(ids) < maxViews {
		query := url.Values{"database_id": {databaseID}, "page_size": {"100"}}
		if cursor != "" {
			query.Set("start_cursor", cursor)
		}
		var out struct {
			Results []struct {
				ID string `json:"id"`
			} `json:"results"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		}
		if err := c.doJSON(ctx, http.MethodGet, "/v1/views", query, nil, &out); err != nil {
			return nil, fmt.Errorf("list views of %s: %w", databaseID, err)
		}
		for _, result := range out.Results {
			ids = append(ids, result.ID)
		}
		if !out.HasMore || out.NextCursor == "" {
			break
		}
		cursor = out.NextCursor
	}
	if len(ids) > maxViews {
		ids = ids[:maxViews]
	}

	views := make([]View, 0, len(ids))
	for _, id := range ids {
		var out struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}
		if err := c.doJSON(ctx, http.MethodGet, "/v1/views/"+url.PathEscape(id), nil, nil, &out); err != nil {
			return nil, fmt.Errorf("get view %s: %w", id, err)
		}
		if viewTypesWithoutRows[out.Type] {
			continue
		}
		if out.Name == "" {
			out.Name = "Untitled view"
		}
		views = append(views, View{ID: id, Name: out.Name, Type: out.Type})
	}
	return views, nil
}

// viewRowsResponse is the part shared by the answer to running a view and
// to asking for a later page of its results.
type viewRowsResponse struct {
	ID      string `json:"id"`
	Results []struct {
		ID string `json:"id"`
	} `json:"results"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor"`
}

func (r viewRowsResponse) rows(queryID string) *ViewRows {
	rows := &ViewRows{QueryID: queryID, HasMore: r.HasMore, NextCursor: r.NextCursor}
	for _, result := range r.Results {
		rows.RowIDs = append(rows.RowIDs, result.ID)
	}
	return rows
}

// QueryView runs a view and returns the first page of its rows. Running a
// view changes nothing in the workspace, so it is allowed in read-only
// mode.
func (c *Client) QueryView(ctx context.Context, viewID string) (*ViewRows, error) {
	body := map[string]any{"page_size": ViewPageSize}
	var out viewRowsResponse
	if err := c.doJSON(ctx, http.MethodPost, viewQueriesPath(viewID), nil, body, &out); err != nil {
		return nil, fmt.Errorf("query view %s: %w", viewID, err)
	}
	return out.rows(out.ID), nil
}

// ViewRowsAfter returns the page of a view's rows that follows cursor, from
// the result set QueryView started. Notion answers "not found" once the
// result set has expired.
func (c *Client) ViewRowsAfter(ctx context.Context, viewID, queryID, cursor string) (*ViewRows, error) {
	query := url.Values{"start_cursor": {cursor}, "page_size": {strconv.Itoa(ViewPageSize)}}
	path := viewQueriesPath(viewID) + "/" + url.PathEscape(queryID)
	var out viewRowsResponse
	if err := c.doJSON(ctx, http.MethodGet, path, query, nil, &out); err != nil {
		return nil, fmt.Errorf("more rows of view %s: %w", viewID, err)
	}
	return out.rows(queryID), nil
}

func viewQueriesPath(viewID string) string {
	return "/v1/views/" + url.PathEscape(viewID) + "/queries"
}
