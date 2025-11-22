package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"
)

func TestHandleGlobalKeys(t *testing.T) {
	tests := []struct {
		name     string
		keyStr   string
		wantQuit bool
		wantCmd  bool
	}{
		{
			name:     "quit with ctrl+c",
			keyStr:   "ctrl+c",
			wantQuit: true,
			wantCmd:  true,
		},
		{
			name:     "quit with q",
			keyStr:   "q",
			wantQuit: true,
			wantCmd:  true,
		},
		{
			name:     "help with ?",
			keyStr:   "?",
			wantQuit: false,
			wantCmd:  true,
		},
		{
			name:     "unknown key not handled",
			keyStr:   "x",
			wantQuit: false,
			wantCmd:  false,
		},
		{
			name:     "letter not quit",
			keyStr:   "h",
			wantQuit: false,
			wantCmd:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.keyStr)}
			handled, cmd := HandleGlobalKeys(msg)

			assert.Equal(t, tt.wantCmd, handled)
			if tt.wantQuit {
				assert.NotNil(t, cmd)
			} else if tt.wantCmd {
				// Help case
				assert.Nil(t, cmd)
			}
		})
	}
}

func TestHandleNavigationKeys(t *testing.T) {
	tests := []struct {
		name       string
		keyStr     string
		cursor     int
		pageCount  int
		wantCursor int
		wantHandle bool
	}{
		{
			name:       "up key decrements cursor",
			keyStr:     "up",
			cursor:     2,
			pageCount:  5,
			wantCursor: 1,
			wantHandle: true,
		},
		{
			name:       "k key decrements cursor",
			keyStr:     "k",
			cursor:     2,
			pageCount:  5,
			wantCursor: 1,
			wantHandle: true,
		},
		{
			name:       "up at start stays at 0",
			keyStr:     "up",
			cursor:     0,
			pageCount:  5,
			wantCursor: 0,
			wantHandle: true,
		},
		{
			name:       "down key increments cursor",
			keyStr:     "down",
			cursor:     2,
			pageCount:  5,
			wantCursor: 3,
			wantHandle: true,
		},
		{
			name:       "j key increments cursor",
			keyStr:     "j",
			cursor:     2,
			pageCount:  5,
			wantCursor: 3,
			wantHandle: true,
		},
		{
			name:       "down at end stays at last",
			keyStr:     "down",
			cursor:     4,
			pageCount:  5,
			wantCursor: 4,
			wantHandle: true,
		},
		{
			name:       "unknown key not handled",
			keyStr:     "x",
			cursor:     2,
			pageCount:  5,
			wantCursor: 2,
			wantHandle: false,
		},
		{
			name:       "r key not navigation",
			keyStr:     "r",
			cursor:     2,
			pageCount:  5,
			wantCursor: 2,
			wantHandle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.keyStr)}
			cursor, handled := HandleNavigationKeys(msg, tt.cursor, tt.pageCount)

			assert.Equal(t, tt.wantCursor, cursor)
			assert.Equal(t, tt.wantHandle, handled)
		})
	}
}

func TestExtractPageTitle(t *testing.T) {
	t.Run("handles page with empty properties", func(t *testing.T) {
		// Create a minimal page with no properties
		page := notionapi.Page{
			Properties: notionapi.Properties{},
		}

		// Should not panic and should return a string
		title := ExtractPageTitle(page)
		assert.IsType(t, "", title)
	})
}
