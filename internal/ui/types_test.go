package ui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPageID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		pageID         PageID
		expectedString string
		expectedEmpty  bool
	}{
		{
			name:           "Valid page ID",
			pageID:         PageID("page-123-abc"),
			expectedString: "page-123-abc",
			expectedEmpty:  false,
		},
		{
			name:           "Empty page ID",
			pageID:         PageID(""),
			expectedString: "",
			expectedEmpty:  true,
		},
		{
			name:           "UUID format",
			pageID:         PageID("550e8400-e29b-41d4-a716-446655440000"),
			expectedString: "550e8400-e29b-41d4-a716-446655440000",
			expectedEmpty:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedString, tt.pageID.String())
			assert.Equal(t, tt.expectedEmpty, tt.pageID.IsEmpty())
		})
	}
}

func TestViewMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		mode           ViewMode
		expectedString string
		expectedValid  bool
	}{
		{
			name:           "Browse mode",
			mode:           ViewModeBrowse,
			expectedString: "browse",
			expectedValid:  true,
		},
		{
			name:           "Edit mode",
			mode:           ViewModeEdit,
			expectedString: "edit",
			expectedValid:  true,
		},
		{
			name:           "Command mode",
			mode:           ViewModeCommand,
			expectedString: "command",
			expectedValid:  true,
		},
		{
			name:           "Invalid mode",
			mode:           ViewMode("invalid"),
			expectedString: "invalid",
			expectedValid:  false,
		},
		{
			name:           "Empty mode",
			mode:           ViewMode(""),
			expectedString: "",
			expectedValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedString, tt.mode.String())
			assert.Equal(t, tt.expectedValid, tt.mode.IsValid())
		})
	}
}

func TestSyncStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		status          SyncStatus
		expectedString  string
		expectedValid   bool
		expectedDisplay string
	}{
		{
			name:            "Synced status",
			status:          SyncStatusSynced,
			expectedString:  "synced",
			expectedValid:   true,
			expectedDisplay: "Synced",
		},
		{
			name:            "Syncing status",
			status:          SyncStatusSyncing,
			expectedString:  "syncing",
			expectedValid:   true,
			expectedDisplay: "Syncing...",
		},
		{
			name:            "Offline status",
			status:          SyncStatusOffline,
			expectedString:  "offline",
			expectedValid:   true,
			expectedDisplay: "Offline",
		},
		{
			name:            "Error status",
			status:          SyncStatusError,
			expectedString:  "error",
			expectedValid:   true,
			expectedDisplay: "Sync Error",
		},
		{
			name:            "Invalid status",
			status:          SyncStatus("unknown"),
			expectedString:  "unknown",
			expectedValid:   false,
			expectedDisplay: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedString, tt.status.String())
			assert.Equal(t, tt.expectedValid, tt.status.IsValid())
			assert.Equal(t, tt.expectedDisplay, tt.status.DisplayText())
		})
	}
}

func TestPage(t *testing.T) {
	t.Parallel()

	now := time.Now()

	tests := []struct {
		name          string
		page          Page
		expectedEmpty bool
	}{
		{
			name: "Valid page",
			page: NewPage(
				PageID("page-123"),
				"Test Page",
				"Published",
				now,
			),
			expectedEmpty: false,
		},
		{
			name: "Page with empty ID",
			page: NewPage(
				PageID(""),
				"Untitled",
				"Draft",
				now,
			),
			expectedEmpty: true,
		},
		{
			name: "Page with empty title",
			page: NewPage(
				PageID("page-456"),
				"",
				"Draft",
				now,
			),
			expectedEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedEmpty, tt.page.IsEmpty())
		})
	}
}

func TestNewPage(t *testing.T) {
	t.Parallel()

	id := PageID("test-id-123")
	title := "My Test Page"
	status := "Active"
	updatedAt := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	page := NewPage(id, title, status, updatedAt)

	assert.Equal(t, id, page.ID)
	assert.Equal(t, title, page.Title)
	assert.Equal(t, status, page.Status)
	assert.Equal(t, updatedAt, page.UpdatedAt)
}

func TestBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		block         Block
		expectedEmpty bool
	}{
		{
			name:          "Valid block",
			block:         NewBlock("block-123", "paragraph", "This is content"),
			expectedEmpty: false,
		},
		{
			name:          "Block with empty ID",
			block:         NewBlock("", "heading", "Title"),
			expectedEmpty: true,
		},
		{
			name:          "Block with empty content",
			block:         NewBlock("block-456", "divider", ""),
			expectedEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expectedEmpty, tt.block.IsEmpty())
		})
	}
}

func TestNewBlock(t *testing.T) {
	t.Parallel()

	id := "block-abc-123"
	blockType := "paragraph"
	content := "This is a paragraph of text."

	block := NewBlock(id, blockType, content)

	assert.Equal(t, id, block.ID)
	assert.Equal(t, blockType, block.Type)
	assert.Equal(t, content, block.Content)
}

func TestViewModeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ViewMode("browse"), ViewModeBrowse)
	assert.Equal(t, ViewMode("edit"), ViewModeEdit)
	assert.Equal(t, ViewMode("command"), ViewModeCommand)
}

func TestSyncStatusConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SyncStatus("synced"), SyncStatusSynced)
	assert.Equal(t, SyncStatus("syncing"), SyncStatusSyncing)
	assert.Equal(t, SyncStatus("offline"), SyncStatusOffline)
	assert.Equal(t, SyncStatus("error"), SyncStatusError)
}
