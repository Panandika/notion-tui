package testhelpers

import (
	"context"

	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/notion"
)

// UpdatePageMarkdownCall records a call to UpdatePageMarkdown.
type UpdatePageMarkdownCall struct {
	ID      string
	Updates []notion.ContentUpdate
}

// ReplacePageMarkdownCall records a call to ReplacePageMarkdown.
type ReplacePageMarkdownCall struct {
	ID       string
	Markdown string
}

// CreatePageMarkdownCall records a call to CreatePageMarkdown.
type CreatePageMarkdownCall struct {
	Parent   notion.PageParent
	Markdown string
}

// SetPagePropertyCall records a call to SetPageProperty.
type SetPagePropertyCall struct {
	PageID string
	Update notion.PropertyUpdate
}

// GetPageMarkdown returns the configured page Markdown, or a small default page.
func (m *MockNotionClient) GetPageMarkdown(ctx context.Context, id string) (*notion.PageMarkdown, error) {
	m.mu.Lock()
	m.GetPageMarkdownCalls = append(m.GetPageMarkdownCalls, id)
	m.mu.Unlock()

	if m.GetPageMarkdownFunc != nil {
		return m.GetPageMarkdownFunc(ctx, id)
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	if m.PageMarkdownToReturn != nil {
		return m.PageMarkdownToReturn, nil
	}
	return &notion.PageMarkdown{ID: id, Markdown: "Test page content"}, nil
}

// UpdatePageMarkdown records the edits and echoes an empty page back.
func (m *MockNotionClient) UpdatePageMarkdown(ctx context.Context, id string, updates []notion.ContentUpdate) (*notion.PageMarkdown, error) {
	m.mu.Lock()
	m.UpdatePageMarkdownCalls = append(m.UpdatePageMarkdownCalls, UpdatePageMarkdownCall{ID: id, Updates: updates})
	m.mu.Unlock()

	if m.UpdatePageMarkdownFunc != nil {
		return m.UpdatePageMarkdownFunc(ctx, id, updates)
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return &notion.PageMarkdown{ID: id}, nil
}

// ReplacePageMarkdown records the new content and returns it as the page.
func (m *MockNotionClient) ReplacePageMarkdown(ctx context.Context, id, markdown string) (*notion.PageMarkdown, error) {
	m.mu.Lock()
	m.ReplacePageMarkdownCalls = append(m.ReplacePageMarkdownCalls, ReplacePageMarkdownCall{ID: id, Markdown: markdown})
	m.mu.Unlock()

	if m.ReplacePageMarkdownFunc != nil {
		return m.ReplacePageMarkdownFunc(ctx, id, markdown)
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return &notion.PageMarkdown{ID: id, Markdown: markdown}, nil
}

// CreatePageMarkdown records the request and returns a fixed new page ID.
func (m *MockNotionClient) CreatePageMarkdown(ctx context.Context, parent notion.PageParent, markdown string) (*notion.CreatedPage, error) {
	m.mu.Lock()
	m.CreatePageMarkdownCalls = append(m.CreatePageMarkdownCalls, CreatePageMarkdownCall{Parent: parent, Markdown: markdown})
	m.mu.Unlock()

	if m.CreatePageMarkdownFunc != nil {
		return m.CreatePageMarkdownFunc(ctx, parent, markdown)
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	return &notion.CreatedPage{ID: "created-page-id"}, nil
}

// GetDatabase returns the configured database, or an empty one.
func (m *MockNotionClient) GetDatabase(ctx context.Context, id string) (*notionapi.Database, error) {
	m.mu.Lock()
	m.GetDatabaseCalls = append(m.GetDatabaseCalls, id)
	m.mu.Unlock()

	if m.GetDatabaseFunc != nil {
		return m.GetDatabaseFunc(ctx, id)
	}
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	if m.DatabaseSchemaToReturn != nil {
		return m.DatabaseSchemaToReturn, nil
	}
	return &notionapi.Database{ID: notionapi.ObjectID(id)}, nil
}

// SetPageProperty records the update.
func (m *MockNotionClient) SetPageProperty(ctx context.Context, pageID string, update notion.PropertyUpdate) error {
	m.mu.Lock()
	m.SetPagePropertyCalls = append(m.SetPagePropertyCalls, SetPagePropertyCall{PageID: pageID, Update: update})
	m.mu.Unlock()

	if m.SetPagePropertyFunc != nil {
		return m.SetPagePropertyFunc(ctx, pageID, update)
	}
	return m.ErrorToReturn
}
