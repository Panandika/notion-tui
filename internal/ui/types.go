package ui

import (
	"time"
)

// PageID represents a unique identifier for a Notion page.
type PageID string

// String returns the string representation of PageID.
func (p PageID) String() string {
	return string(p)
}

// IsEmpty returns true if the PageID is empty.
func (p PageID) IsEmpty() bool {
	return p == ""
}

// ViewMode represents the current view mode of the application.
type ViewMode string

const (
	// ViewModeBrowse is the default browsing mode.
	ViewModeBrowse ViewMode = "browse"
	// ViewModeEdit is the editing mode.
	ViewModeEdit ViewMode = "edit"
	// ViewModeCommand is the command palette mode.
	ViewModeCommand ViewMode = "command"
)

// String returns the string representation of ViewMode.
func (v ViewMode) String() string {
	return string(v)
}

// IsValid returns true if the ViewMode is a valid mode.
func (v ViewMode) IsValid() bool {
	switch v {
	case ViewModeBrowse, ViewModeEdit, ViewModeCommand:
		return true
	default:
		return false
	}
}

// SyncStatus represents the synchronization status with Notion.
type SyncStatus string

const (
	// SyncStatusSynced indicates content is synced with Notion.
	SyncStatusSynced SyncStatus = "synced"
	// SyncStatusSyncing indicates content is currently syncing.
	SyncStatusSyncing SyncStatus = "syncing"
	// SyncStatusOffline indicates offline mode.
	SyncStatusOffline SyncStatus = "offline"
	// SyncStatusError indicates a sync error occurred.
	SyncStatusError SyncStatus = "error"
)

// String returns the string representation of SyncStatus.
func (s SyncStatus) String() string {
	return string(s)
}

// IsValid returns true if the SyncStatus is a valid status.
func (s SyncStatus) IsValid() bool {
	switch s {
	case SyncStatusSynced, SyncStatusSyncing, SyncStatusOffline, SyncStatusError:
		return true
	default:
		return false
	}
}

// DisplayText returns a human-readable text for the status.
func (s SyncStatus) DisplayText() string {
	switch s {
	case SyncStatusSynced:
		return "Synced"
	case SyncStatusSyncing:
		return "Syncing..."
	case SyncStatusOffline:
		return "Offline"
	case SyncStatusError:
		return "Sync Error"
	default:
		return "Unknown"
	}
}

// Page represents a Notion page in the UI.
type Page struct {
	ID        PageID
	Title     string
	Status    string
	UpdatedAt time.Time
}

// NewPage creates a new Page instance.
func NewPage(id PageID, title string, status string, updatedAt time.Time) Page {
	return Page{
		ID:        id,
		Title:     title,
		Status:    status,
		UpdatedAt: updatedAt,
	}
}

// IsEmpty returns true if the Page has no ID.
func (p Page) IsEmpty() bool {
	return p.ID.IsEmpty()
}

// Block represents a content block in a Notion page.
type Block struct {
	ID      string
	Type    string
	Content string
}

// NewBlock creates a new Block instance.
func NewBlock(id string, blockType string, content string) Block {
	return Block{
		ID:      id,
		Type:    blockType,
		Content: content,
	}
}

// IsEmpty returns true if the Block has no ID.
func (b Block) IsEmpty() bool {
	return b.ID == ""
}
