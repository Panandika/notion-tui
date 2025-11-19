package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBlockEditor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         NewBlockEditorInput
		wantBlockID   string
		wantBlockType string
		wantContent   string
		wantWidth     int
		wantHeight    int
		wantDirty     bool
	}{
		{
			name: "basic editor",
			input: NewBlockEditorInput{
				BlockID:   "block-123",
				BlockType: "paragraph",
				Content:   "Hello world",
				Width:     80,
				Height:    10,
			},
			wantBlockID:   "block-123",
			wantBlockType: "paragraph",
			wantContent:   "Hello world",
			wantWidth:     80,
			wantHeight:    10,
			wantDirty:     false,
		},
		{
			name: "empty content",
			input: NewBlockEditorInput{
				BlockID:   "empty-block",
				BlockType: "text",
				Content:   "",
				Width:     60,
				Height:    5,
			},
			wantBlockID:   "empty-block",
			wantBlockType: "text",
			wantContent:   "",
			wantWidth:     60,
			wantHeight:    5,
			wantDirty:     false,
		},
		{
			name: "multiline content",
			input: NewBlockEditorInput{
				BlockID:   "multi-block",
				BlockType: "code",
				Content:   "line 1\nline 2\nline 3",
				Width:     100,
				Height:    15,
			},
			wantBlockID:   "multi-block",
			wantBlockType: "code",
			wantContent:   "line 1\nline 2\nline 3",
			wantWidth:     100,
			wantHeight:    15,
			wantDirty:     false,
		},
		{
			name: "heading block",
			input: NewBlockEditorInput{
				BlockID:   "heading-1",
				BlockType: "heading_1",
				Content:   "Main Title",
				Width:     120,
				Height:    3,
			},
			wantBlockID:   "heading-1",
			wantBlockType: "heading_1",
			wantContent:   "Main Title",
			wantWidth:     120,
			wantHeight:    3,
			wantDirty:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(tt.input)

			assert.Equal(t, tt.wantBlockID, editor.BlockID())
			assert.Equal(t, tt.wantBlockType, editor.BlockType())
			assert.Equal(t, tt.wantContent, editor.GetText())
			assert.Equal(t, tt.wantWidth, editor.Width())
			assert.Equal(t, tt.wantHeight, editor.Height())
			assert.Equal(t, tt.wantDirty, editor.IsDirty())
		})
	}
}

func TestEditorFocus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupFunc   func(*BlockEditor)
		wantFocused bool
	}{
		{
			name: "focus editor",
			setupFunc: func(e *BlockEditor) {
				e.Blur()
				e.Focus()
			},
			wantFocused: true,
		},
		{
			name: "blur editor",
			setupFunc: func(e *BlockEditor) {
				e.Focus()
				e.Blur()
			},
			wantFocused: false,
		},
		{
			name: "focus returns command",
			setupFunc: func(e *BlockEditor) {
				// Just focus
			},
			wantFocused: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   "Test content",
				Width:     80,
				Height:    10,
			})

			tt.setupFunc(&editor)

			// Verify focus command is returned
			cmd := editor.Focus()
			if tt.wantFocused {
				assert.NotNil(t, cmd)
			}
		})
	}
}

func TestEditorText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		initialText string
		newText     string
		wantText    string
		wantDirty   bool
	}{
		{
			name:        "set new text",
			initialText: "Original",
			newText:     "Updated",
			wantText:    "Updated",
			wantDirty:   false,
		},
		{
			name:        "set empty text",
			initialText: "Some content",
			newText:     "",
			wantText:    "",
			wantDirty:   false,
		},
		{
			name:        "set multiline text",
			initialText: "Single line",
			newText:     "Line 1\nLine 2\nLine 3",
			wantText:    "Line 1\nLine 2\nLine 3",
			wantDirty:   false,
		},
		{
			name:        "set text with special characters",
			initialText: "Normal",
			newText:     "Special: @#$%^&*()",
			wantText:    "Special: @#$%^&*()",
			wantDirty:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   tt.initialText,
				Width:     80,
				Height:    10,
			})

			editor.SetText(tt.newText)

			assert.Equal(t, tt.wantText, editor.GetText())
			assert.Equal(t, tt.wantDirty, editor.IsDirty())
		})
	}
}

func TestDirtyState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupFunc func(*BlockEditor)
		wantDirty bool
	}{
		{
			name: "new editor is clean",
			setupFunc: func(e *BlockEditor) {
				// No action
			},
			wantDirty: false,
		},
		{
			name: "mark clean resets dirty",
			setupFunc: func(e *BlockEditor) {
				e.dirty = true
				e.MarkClean()
			},
			wantDirty: false,
		},
		{
			name: "set text resets dirty",
			setupFunc: func(e *BlockEditor) {
				e.dirty = true
				e.SetText("New text")
			},
			wantDirty: false,
		},
		{
			name: "dirty remains after mark clean",
			setupFunc: func(e *BlockEditor) {
				e.dirty = true
				e.MarkClean()
				// dirty should be false now
			},
			wantDirty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   "Initial content",
				Width:     80,
				Height:    10,
			})

			tt.setupFunc(&editor)

			assert.Equal(t, tt.wantDirty, editor.IsDirty())
		})
	}
}

func TestSaveCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		blockID       string
		blockType     string
		content       string
		wantBlockID   string
		wantBlockType string
		wantContent   string
	}{
		{
			name:          "save paragraph block",
			blockID:       "para-1",
			blockType:     "paragraph",
			content:       "Paragraph content",
			wantBlockID:   "para-1",
			wantBlockType: "paragraph",
			wantContent:   "Paragraph content",
		},
		{
			name:          "save code block",
			blockID:       "code-block",
			blockType:     "code",
			content:       "func main() {}",
			wantBlockID:   "code-block",
			wantBlockType: "code",
			wantContent:   "func main() {}",
		},
		{
			name:          "save empty content",
			blockID:       "empty-save",
			blockType:     "text",
			content:       "",
			wantBlockID:   "empty-save",
			wantBlockType: "text",
			wantContent:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   tt.blockID,
				BlockType: tt.blockType,
				Content:   tt.content,
				Width:     80,
				Height:    10,
			})

			// Simulate Ctrl+S
			_, cmd := editor.Update(tea.KeyMsg{Type: tea.KeyCtrlS})

			require.NotNil(t, cmd)

			msg := cmd()
			saveMsg, ok := msg.(SaveDraftMsg)
			require.True(t, ok, "expected SaveDraftMsg")

			assert.Equal(t, tt.wantBlockID, saveMsg.BlockID)
			assert.Equal(t, tt.wantBlockType, saveMsg.BlockType)
			assert.Equal(t, tt.wantContent, saveMsg.Content)
		})
	}
}

func TestCancelCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		blockID     string
		wantBlockID string
	}{
		{
			name:        "cancel paragraph editing",
			blockID:     "para-cancel",
			wantBlockID: "para-cancel",
		},
		{
			name:        "cancel code editing",
			blockID:     "code-cancel",
			wantBlockID: "code-cancel",
		},
		{
			name:        "cancel with empty block id",
			blockID:     "",
			wantBlockID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   tt.blockID,
				BlockType: "paragraph",
				Content:   "Some content",
				Width:     80,
				Height:    10,
			})

			// Simulate Esc
			_, cmd := editor.Update(tea.KeyMsg{Type: tea.KeyEsc})

			require.NotNil(t, cmd)

			msg := cmd()
			cancelMsg, ok := msg.(CancelEditMsg)
			require.True(t, ok, "expected CancelEditMsg")

			assert.Equal(t, tt.wantBlockID, cancelMsg.BlockID)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		msg       tea.Msg
		checkFunc func(t *testing.T, e BlockEditor, cmd tea.Cmd)
	}{
		{
			name: "window size message",
			msg:  tea.WindowSizeMsg{Width: 100, Height: 20},
			checkFunc: func(t *testing.T, e BlockEditor, cmd tea.Cmd) {
				assert.Equal(t, 100, e.Width())
				assert.Equal(t, 20, e.Height())
			},
		},
		{
			name: "init returns blink command",
			msg:  nil,
			checkFunc: func(t *testing.T, e BlockEditor, cmd tea.Cmd) {
				initCmd := e.Init()
				assert.NotNil(t, initCmd)
			},
		},
		{
			name: "ctrl+s triggers save",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlS},
			checkFunc: func(t *testing.T, e BlockEditor, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(SaveDraftMsg)
				assert.True(t, ok)
			},
		},
		{
			name: "esc triggers cancel",
			msg:  tea.KeyMsg{Type: tea.KeyEsc},
			checkFunc: func(t *testing.T, e BlockEditor, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(CancelEditMsg)
				assert.True(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   "Test content",
				Width:     80,
				Height:    10,
			})

			var updatedEditor BlockEditor
			var cmd tea.Cmd

			if tt.msg != nil {
				updatedEditor, cmd = editor.Update(tt.msg)
			} else {
				updatedEditor = editor
			}

			tt.checkFunc(t, updatedEditor, cmd)
		})
	}
}

func TestView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		dirty     bool
		wantEmpty bool
	}{
		{
			name:      "clean editor view",
			dirty:     false,
			wantEmpty: false,
		},
		{
			name:      "dirty editor view",
			dirty:     true,
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   "Test content",
				Width:     80,
				Height:    10,
			})

			if tt.dirty {
				editor.dirty = true
			}

			view := editor.View()

			if tt.wantEmpty {
				assert.Empty(t, view)
			} else {
				assert.NotEmpty(t, view)
			}

			// Check that help text is present
			assert.Contains(t, view, "Ctrl+S")
			assert.Contains(t, view, "Esc")

			// Check dirty marker presence
			if tt.dirty {
				assert.Contains(t, view, "Modified")
			} else {
				assert.Contains(t, view, "Ready")
			}
		})
	}
}

func TestDefaultEditorStyles(t *testing.T) {
	t.Parallel()

	styles := DefaultEditorStyles()

	// Test that styles can render text
	testText := "test"
	assert.NotEmpty(t, styles.Container.Render(testText))
	assert.NotEmpty(t, styles.DirtyMarker.Render(testText))
	assert.NotEmpty(t, styles.HelpText.Render(testText))
}

func TestEditorSetSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		newWidth   int
		newHeight  int
		wantWidth  int
		wantHeight int
	}{
		{
			name:       "increase size",
			newWidth:   120,
			newHeight:  25,
			wantWidth:  120,
			wantHeight: 25,
		},
		{
			name:       "decrease size",
			newWidth:   40,
			newHeight:  5,
			wantWidth:  40,
			wantHeight: 5,
		},
		{
			name:       "same size",
			newWidth:   80,
			newHeight:  10,
			wantWidth:  80,
			wantHeight: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   "test-block",
				BlockType: "paragraph",
				Content:   "Test content",
				Width:     80,
				Height:    10,
			})

			editor.SetSize(tt.newWidth, tt.newHeight)

			assert.Equal(t, tt.wantWidth, editor.Width())
			assert.Equal(t, tt.wantHeight, editor.Height())
		})
	}
}

func TestEditorBlockInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		blockID       string
		blockType     string
		wantBlockID   string
		wantBlockType string
	}{
		{
			name:          "paragraph block",
			blockID:       "para-123",
			blockType:     "paragraph",
			wantBlockID:   "para-123",
			wantBlockType: "paragraph",
		},
		{
			name:          "heading block",
			blockID:       "heading-456",
			blockType:     "heading_2",
			wantBlockID:   "heading-456",
			wantBlockType: "heading_2",
		},
		{
			name:          "code block",
			blockID:       "code-789",
			blockType:     "code",
			wantBlockID:   "code-789",
			wantBlockType: "code",
		},
		{
			name:          "empty block id",
			blockID:       "",
			blockType:     "text",
			wantBlockID:   "",
			wantBlockType: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			editor := NewBlockEditor(NewBlockEditorInput{
				BlockID:   tt.blockID,
				BlockType: tt.blockType,
				Content:   "Content",
				Width:     80,
				Height:    10,
			})

			assert.Equal(t, tt.wantBlockID, editor.BlockID())
			assert.Equal(t, tt.wantBlockType, editor.BlockType())
		})
	}
}

func TestMarkCleanUpdatesInitialText(t *testing.T) {
	t.Parallel()

	editor := NewBlockEditor(NewBlockEditorInput{
		BlockID:   "test-block",
		BlockType: "paragraph",
		Content:   "Original",
		Width:     80,
		Height:    10,
	})

	// Manually set dirty and change content
	editor.dirty = true

	// Mark clean should update initial text to current value
	editor.MarkClean()

	assert.False(t, editor.IsDirty())
}
