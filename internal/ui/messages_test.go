package ui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		message        string
		err            error
		expectedString string
	}{
		{
			name:           "Error with underlying error",
			message:        "failed to load",
			err:            errors.New("network timeout"),
			expectedString: "failed to load: network timeout",
		},
		{
			name:           "Error without underlying error",
			message:        "validation failed",
			err:            nil,
			expectedString: "validation failed",
		},
		{
			name:           "Empty message with error",
			message:        "",
			err:            errors.New("some error"),
			expectedString: ": some error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewErrorMsg(tt.message, tt.err)

			assert.Equal(t, tt.expectedString, msg.Error())
			assert.Equal(t, tt.err, msg.Unwrap())
		})
	}
}

func TestErrorMsgUnwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("original error")
	msg := NewErrorMsg("wrapped error", originalErr)

	// Test errors.Is
	assert.True(t, errors.Is(msg, originalErr))

	// Test Unwrap
	unwrapped := msg.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

func TestNavigationMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		pageID     string
		expectedID string
	}{
		{
			name:       "Valid page ID",
			pageID:     "page-123-abc",
			expectedID: "page-123-abc",
		},
		{
			name:       "Empty page ID",
			pageID:     "",
			expectedID: "",
		},
		{
			name:       "UUID format",
			pageID:     "550e8400-e29b-41d4-a716-446655440000",
			expectedID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewNavigationMsg(tt.pageID)

			assert.Equal(t, tt.expectedID, msg.PageID())
		})
	}
}

func TestLoadingMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		isLoading bool
	}{
		{
			name:      "Loading true",
			isLoading: true,
		},
		{
			name:      "Loading false",
			isLoading: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewLoadingMsg(tt.isLoading)

			assert.Equal(t, tt.isLoading, msg.IsLoading())
		})
	}
}

func TestContentLoadedMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		content         string
		err             error
		expectedContent string
		expectError     bool
	}{
		{
			name:            "Content loaded successfully",
			content:         "# Hello World\n\nThis is content.",
			err:             nil,
			expectedContent: "# Hello World\n\nThis is content.",
			expectError:     false,
		},
		{
			name:            "Content load failed",
			content:         "",
			err:             errors.New("failed to fetch content"),
			expectedContent: "",
			expectError:     true,
		},
		{
			name:            "Empty content without error",
			content:         "",
			err:             nil,
			expectedContent: "",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewContentLoadedMsg(tt.content, tt.err)

			assert.Equal(t, tt.expectedContent, msg.Content())
			if tt.expectError {
				assert.NotNil(t, msg.Err())
			} else {
				assert.Nil(t, msg.Err())
			}
		})
	}
}

func TestRefreshMsg(t *testing.T) {
	t.Parallel()

	msg := NewRefreshMsg()

	// RefreshMsg is a marker type, just ensure it can be created
	assert.NotNil(t, msg)
}

func TestItemSelectedMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		index         int
		id            string
		expectedIndex int
		expectedID    string
	}{
		{
			name:          "Valid selection",
			index:         5,
			id:            "item-abc-123",
			expectedIndex: 5,
			expectedID:    "item-abc-123",
		},
		{
			name:          "First item",
			index:         0,
			id:            "first-item",
			expectedIndex: 0,
			expectedID:    "first-item",
		},
		{
			name:          "Negative index",
			index:         -1,
			id:            "invalid",
			expectedIndex: -1,
			expectedID:    "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewItemSelectedMsg(tt.index, tt.id)

			assert.Equal(t, tt.expectedIndex, msg.Index())
			assert.Equal(t, tt.expectedID, msg.ID())
		})
	}
}

func TestSyncStatusMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		status         string
		expectedStatus string
	}{
		{
			name:           "Synced status",
			status:         "synced",
			expectedStatus: "synced",
		},
		{
			name:           "Syncing status",
			status:         "syncing",
			expectedStatus: "syncing",
		},
		{
			name:           "Offline status",
			status:         "offline",
			expectedStatus: "offline",
		},
		{
			name:           "Error status",
			status:         "error",
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewSyncStatusMsg(tt.status)

			assert.Equal(t, tt.expectedStatus, msg.Status())
		})
	}
}

func TestViewModeChangedMsg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mode         ViewMode
		expectedMode ViewMode
	}{
		{
			name:         "Browse mode",
			mode:         ViewModeBrowse,
			expectedMode: ViewModeBrowse,
		},
		{
			name:         "Edit mode",
			mode:         ViewModeEdit,
			expectedMode: ViewModeEdit,
		},
		{
			name:         "Command mode",
			mode:         ViewModeCommand,
			expectedMode: ViewModeCommand,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := NewViewModeChangedMsg(tt.mode)

			assert.Equal(t, tt.expectedMode, msg.Mode())
		})
	}
}
