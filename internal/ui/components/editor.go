package components

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SaveDraftMsg is sent when the user requests to save the draft (Ctrl+S).
type SaveDraftMsg struct {
	BlockID   string
	BlockType string
	Content   string
}

// CancelEditMsg is sent when the user cancels editing (Esc).
type CancelEditMsg struct {
	BlockID string
}

// BlockEditor is a textarea-based block editing component.
type BlockEditor struct {
	textarea    textarea.Model
	blockID     string
	blockType   string
	dirty       bool
	styles      EditorStyles
	width       int
	height      int
	initialText string
}

// EditorStyles holds the styles for the editor.
type EditorStyles struct {
	Container   lipgloss.Style
	DirtyMarker lipgloss.Style
	HelpText    lipgloss.Style
}

// DefaultEditorStyles returns the default styles for the editor.
func DefaultEditorStyles() EditorStyles {
	return EditorStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7C3AED")).
			Padding(1, 2),
		DirtyMarker: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Bold(true),
		HelpText: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Italic(true),
	}
}

// NewBlockEditorInput contains options for creating a new block editor.
type NewBlockEditorInput struct {
	BlockID   string
	BlockType string
	Content   string
	Width     int
	Height    int
}

// NewBlockEditor creates a new block editor component.
func NewBlockEditor(input NewBlockEditorInput) BlockEditor {
	ta := textarea.New()
	ta.Placeholder = "Enter text..."
	ta.Focus()
	ta.CharLimit = 0 // No character limit
	ta.SetWidth(input.Width)
	ta.SetHeight(input.Height)
	ta.SetValue(input.Content)
	ta.ShowLineNumbers = true

	// Style the textarea
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(lipgloss.Color("#374151"))
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F3F4F6"))
	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6B7280"))

	return BlockEditor{
		textarea:    ta,
		blockID:     input.BlockID,
		blockType:   input.BlockType,
		dirty:       false,
		styles:      DefaultEditorStyles(),
		width:       input.Width,
		height:      input.Height,
		initialText: input.Content,
	}
}

// Init initializes the block editor component.
func (e BlockEditor) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages and returns the updated editor and command.
func (e BlockEditor) Update(msg tea.Msg) (BlockEditor, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			// Save the draft
			return e, func() tea.Msg {
				return SaveDraftMsg{
					BlockID:   e.blockID,
					BlockType: e.blockType,
					Content:   e.textarea.Value(),
				}
			}
		case "esc":
			// Cancel editing
			return e, func() tea.Msg {
				return CancelEditMsg{
					BlockID: e.blockID,
				}
			}
		}

	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		e.textarea.SetWidth(msg.Width)
		e.textarea.SetHeight(msg.Height)
	}

	// Update the textarea
	e.textarea, cmd = e.textarea.Update(msg)

	// Check if content has changed to update dirty state
	if e.textarea.Value() != e.initialText {
		e.dirty = true
	}

	return e, cmd
}

// View renders the editor.
func (e BlockEditor) View() string {
	var header string
	if e.dirty {
		header = e.styles.DirtyMarker.Render("* Modified")
	} else {
		header = "Ready"
	}

	helpText := e.styles.HelpText.Render("Ctrl+S: Save | Esc: Cancel")

	content := e.textarea.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		e.styles.Container.Render(content),
		helpText,
	)
}

// Focus sets focus on the editor textarea.
func (e *BlockEditor) Focus() tea.Cmd {
	return e.textarea.Focus()
}

// Blur removes focus from the editor textarea.
func (e *BlockEditor) Blur() {
	e.textarea.Blur()
}

// GetText returns the current text content of the editor.
func (e BlockEditor) GetText() string {
	return e.textarea.Value()
}

// SetText sets the text content of the editor.
func (e *BlockEditor) SetText(text string) {
	e.textarea.SetValue(text)
	e.initialText = text
	e.dirty = false
}

// IsDirty returns true if the editor content has been modified.
func (e BlockEditor) IsDirty() bool {
	return e.dirty
}

// MarkClean marks the editor as not dirty.
func (e *BlockEditor) MarkClean() {
	e.dirty = false
	e.initialText = e.textarea.Value()
}

// BlockID returns the block ID being edited.
func (e BlockEditor) BlockID() string {
	return e.blockID
}

// BlockType returns the type of block being edited.
func (e BlockEditor) BlockType() string {
	return e.blockType
}

// Width returns the editor width.
func (e BlockEditor) Width() int {
	return e.width
}

// Height returns the editor height.
func (e BlockEditor) Height() int {
	return e.height
}

// SetSize updates the editor dimensions.
func (e *BlockEditor) SetSize(width, height int) {
	e.width = width
	e.height = height
	e.textarea.SetWidth(width)
	e.textarea.SetHeight(height)
}
