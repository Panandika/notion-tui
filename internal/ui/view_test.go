package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderInitializing(t *testing.T) {
	result := RenderInitializing()
	assert.Contains(t, result, "Initializing")
	assert.Contains(t, result, "\n")
}

func TestRenderError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantText  string
		wantEmpty bool
	}{
		{
			name:      "nil error returns empty",
			err:       nil,
			wantText:  "",
			wantEmpty: true,
		},
		{
			name:      "non-nil error returns error text",
			err:       assert.AnError,
			wantText:  "Error:",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderError(tt.err)
			if tt.wantEmpty {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tt.wantText)
			}
		})
	}
}

func TestRenderEmptyPageList(t *testing.T) {
	result := RenderEmptyPageList()
	assert.Contains(t, result, "No pages found")
	assert.Contains(t, result, "refresh")
	assert.Contains(t, result, "quit")
}

func TestRenderPageList(t *testing.T) {
	tests := []struct {
		name      string
		pages     []string
		cursor    int
		wantLines []string
	}{
		{
			name:   "render single page",
			pages:  []string{"Page 1"},
			cursor: 0,
			wantLines: []string{
				"Notion Pages",
				"> Page 1",
				"(1/1)",
			},
		},
		{
			name:   "render multiple pages with cursor",
			pages:  []string{"Page 1", "Page 2", "Page 3"},
			cursor: 1,
			wantLines: []string{
				"Notion Pages",
				" Page 1",
				"> Page 2",
				" Page 3",
				"(2/3)",
			},
		},
		{
			name:   "cursor at start",
			pages:  []string{"A", "B", "C"},
			cursor: 0,
			wantLines: []string{
				"> A",
				"(1/3)",
			},
		},
		{
			name:   "cursor at end",
			pages:  []string{"A", "B", "C"},
			cursor: 2,
			wantLines: []string{
				"> C",
				"(3/3)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderPageList(tt.pages, tt.cursor)

			// Verify header
			assert.Contains(t, result, "Notion Pages")

			// Verify all wanted lines
			for _, wantLine := range tt.wantLines {
				assert.Contains(t, result, wantLine, "expected line not found: %s", wantLine)
			}

			// Verify help text
			assert.Contains(t, result, "navigate")
			assert.Contains(t, result, "refresh")
			assert.Contains(t, result, "quit")
		})
	}
}

func TestRenderViewState(t *testing.T) {
	tests := []struct {
		name     string
		ready    bool
		err      error
		pages    []string
		cursor   int
		wantText string
	}{
		{
			name:     "not ready shows initializing",
			ready:    false,
			err:      nil,
			pages:    []string{},
			cursor:   0,
			wantText: "Initializing",
		},
		{
			name:     "error shows error message",
			ready:    true,
			err:      assert.AnError,
			pages:    []string{},
			cursor:   0,
			wantText: "Error:",
		},
		{
			name:     "empty pages list shows no pages message",
			ready:    true,
			err:      nil,
			pages:    []string{},
			cursor:   0,
			wantText: "No pages found",
		},
		{
			name:     "ready with pages shows list",
			ready:    true,
			err:      nil,
			pages:    []string{"Page 1", "Page 2"},
			cursor:   0,
			wantText: "Notion Pages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderViewState(tt.ready, tt.err, tt.pages, tt.cursor)
			assert.Contains(t, result, tt.wantText)
		})
	}
}
