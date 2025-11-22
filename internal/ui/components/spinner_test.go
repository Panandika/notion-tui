package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewSpinner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "creates spinner with message",
			message: "Loading...",
		},
		{
			name:    "creates spinner without message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpinner(tt.message)
			assert.Equal(t, tt.message, s.Message())
		})
	}
}

func TestSpinner_Init(t *testing.T) {
	t.Parallel()

	s := NewSpinner("Loading...")
	cmd := s.Init()
	assert.NotNil(t, cmd, "Init should return a tick command")
}

func TestSpinner_Update(t *testing.T) {
	t.Parallel()

	s := NewSpinner("Loading...")

	// Test with a window size message
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	updated, _ := s.Update(msg)

	assert.NotNil(t, updated)
	assert.Equal(t, "Loading...", updated.Message())
}

func TestSpinner_View(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		message     string
		expectEmpty bool
	}{
		{
			name:        "renders with message",
			message:     "Loading pages...",
			expectEmpty: false,
		},
		{
			name:        "renders without message",
			message:     "",
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSpinner(tt.message)
			view := s.View()

			if tt.expectEmpty {
				assert.Empty(t, view)
			} else {
				assert.NotEmpty(t, view)
				if tt.message != "" {
					assert.Contains(t, view, tt.message)
				}
			}
		})
	}
}

func TestSpinner_SetMessage(t *testing.T) {
	t.Parallel()

	s := NewSpinner("Initial message")
	assert.Equal(t, "Initial message", s.Message())

	s.SetMessage("Updated message")
	assert.Equal(t, "Updated message", s.Message())

	view := s.View()
	assert.Contains(t, view, "Updated message")
}
