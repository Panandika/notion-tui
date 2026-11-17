package testhelpers

import (
	"time"

	"github.com/jomei/notionapi"
)

// NewTestPage creates a test page with the given ID and title.
// Returns a valid notionapi.Page structure for hermetic testing.
func NewTestPage(id, title string) *notionapi.Page {
	now := time.Now()

	return &notionapi.Page{
		Object:         notionapi.ObjectTypePage,
		ID:             notionapi.ObjectID(id),
		CreatedTime:    now,
		LastEditedTime: now,
		Archived:       false,
		Properties: notionapi.Properties{
			"title": &notionapi.TitleProperty{
				Title: NewTestRichText(title),
			},
		},
		Parent: notionapi.Parent{
			Type: notionapi.ParentTypeDatabaseID,
		},
		URL: "https://www.notion.so/" + id,
	}
}

// NewTestRichText creates a slice of RichText from a plain text content string.
func NewTestRichText(content string) []notionapi.RichText {
	return []notionapi.RichText{
		{
			Type:      notionapi.ObjectTypeText,
			PlainText: content,
			Text: &notionapi.Text{
				Content: content,
			},
			Annotations: &notionapi.Annotations{
				Bold:          false,
				Italic:        false,
				Strikethrough: false,
				Underline:     false,
				Code:          false,
				Color:         notionapi.ColorDefault,
			},
		},
	}
}

// NewTestRichTextWithAnnotations creates a RichText with specific formatting.
type RichTextOptions struct {
	Bold          bool
	Italic        bool
	Strikethrough bool
	Underline     bool
	Code          bool
	Color         notionapi.Color
	Href          string
}

// NewTestRichTextAdvanced creates a RichText with custom annotations.
func NewTestRichTextAdvanced(content string, opts RichTextOptions) []notionapi.RichText {
	color := opts.Color
	if color == "" {
		color = notionapi.ColorDefault
	}

	return []notionapi.RichText{
		{
			Type:      notionapi.ObjectTypeText,
			PlainText: content,
			Href:      opts.Href,
			Text: &notionapi.Text{
				Content: content,
				Link:    nil,
			},
			Annotations: &notionapi.Annotations{
				Bold:          opts.Bold,
				Italic:        opts.Italic,
				Strikethrough: opts.Strikethrough,
				Underline:     opts.Underline,
				Code:          opts.Code,
				Color:         color,
			},
		},
	}
}

// NewTestDatabase creates a database query response with sample pages.
func NewTestDatabase(dbID string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object: notionapi.ObjectTypeList,
		Results: []notionapi.Page{
			*NewTestPage("page-1", "First Page"),
			*NewTestPage("page-2", "Second Page"),
			*NewTestPage("page-3", "Third Page"),
			*NewTestPage("page-4", "Fourth Page"),
			*NewTestPage("page-5", "Fifth Page"),
		},
		HasMore:    false,
		NextCursor: "",
	}
}

// NewTestDatabaseEmpty creates an empty database query response.
func NewTestDatabaseEmpty(dbID string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object:     notionapi.ObjectTypeList,
		Results:    []notionapi.Page{},
		HasMore:    false,
		NextCursor: "",
	}
}

// NewTestDatabaseWithCursor creates a database response with pagination cursor.
func NewTestDatabaseWithCursor(dbID string, cursor string) *notionapi.DatabaseQueryResponse {
	return &notionapi.DatabaseQueryResponse{
		Object: notionapi.ObjectTypeList,
		Results: []notionapi.Page{
			*NewTestPage("page-1", "Page One"),
			*NewTestPage("page-2", "Page Two"),
		},
		HasMore:    true,
		NextCursor: notionapi.Cursor(cursor),
	}
}
