package pages

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/ui/components"
)

// blockLoadedMsg is sent when a block has been loaded from the API.
type blockLoadedMsg struct {
	block notionapi.Block
	text  string
	err   error
}

// blockSavedMsg is sent when a block has been saved to the API.
type blockSavedMsg struct {
	success bool
	err     error
}

// EditPage wraps the BlockEditor component with save logic for managing block editing.
type EditPage struct {
	editor       components.BlockEditor
	statusBar    components.StatusBar
	pageID       string
	blockID      string
	blockType    string
	originalText string
	loading      bool
	saving       bool
	saved        bool
	err          error
	width        int
	height       int
	notionClient NotionClient
}

// NewEditPageInput contains parameters for creating a new EditPage.
type NewEditPageInput struct {
	Width        int
	Height       int
	NotionClient NotionClient
	PageID       string
	BlockID      string
}

// NewEditPage creates a new EditPage instance with the given configuration.
func NewEditPage(input NewEditPageInput) EditPage {
	statusBar := components.NewStatusBar()
	statusBar.SetWidth(input.Width)
	statusBar.SetMode("Loading")
	statusBar.SetSyncStatus(components.StatusSynced)

	return EditPage{
		pageID:       input.PageID,
		blockID:      input.BlockID,
		loading:      true,
		width:        input.Width,
		height:       input.Height,
		notionClient: input.NotionClient,
		statusBar:    statusBar,
	}
}

// Init initializes the EditPage and loads the block content.
func (ep *EditPage) Init() tea.Cmd {
	return ep.loadBlockCmd()
}

// Update handles messages and returns the updated EditPage and command.
func (ep *EditPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ep.width = msg.Width
		ep.height = msg.Height
		ep.statusBar.SetWidth(msg.Width)
		if !ep.loading {
			editorHeight := msg.Height - lipgloss.Height(ep.statusBar.View()) - 4
			ep.editor.SetSize(msg.Width-4, editorHeight)
		}
		return ep, nil

	case blockLoadedMsg:
		if msg.err != nil {
			ep.loading = false
			ep.err = msg.err
			return ep, nil
		}

		ep.loading = false
		ep.blockType = string(msg.block.GetType())
		ep.originalText = msg.text

		// Create editor with loaded content
		editorHeight := ep.height - lipgloss.Height(ep.statusBar.View()) - 4
		ep.editor = components.NewBlockEditor(components.NewBlockEditorInput{
			BlockID:   ep.blockID,
			BlockType: ep.blockType,
			Content:   msg.text,
			Width:     ep.width - 4,
			Height:    editorHeight,
		})

		ep.statusBar.SetMode("Editing")
		return ep, ep.editor.Init()

	case components.SaveDraftMsg:
		// User pressed Ctrl+S in editor - save to Notion API
		if ep.saving {
			return ep, nil // Already saving
		}
		ep.saving = true
		ep.saved = false
		ep.statusBar.SetMode("Saving...")
		return ep, ep.saveCmd()

	case blockSavedMsg:
		ep.saving = false
		if msg.err != nil {
			ep.err = fmt.Errorf("save failed: %w", msg.err)
			ep.statusBar.SetMode("Error")
			return ep, nil
		}

		ep.saved = true
		ep.editor.MarkClean()
		ep.statusBar.SetMode("Saved!")

		// Clear "Saved!" message after a moment
		return ep, tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg {
			return components.SaveDraftMsg{} // Reuse as a "clear saved message" signal
		})

	case components.CancelEditMsg:
		// User pressed Esc - check if dirty and navigate back
		// In a real app, this would send a navigation message to parent
		// For now, we just mark that we want to exit
		if !ep.loading && ep.editor.IsDirty() {
			// TODO: Show confirmation dialog
			// For now, just allow cancel
		}
		return ep, tea.Quit

	case tea.KeyMsg:
		// Handle keys at page level before delegating to editor
		switch msg.String() {
		case "ctrl+s":
			if !ep.loading && !ep.saving {
				ep.saving = true
				ep.saved = false
				ep.statusBar.SetMode("Saving...")
				return ep, ep.saveCmd()
			}
			return ep, nil
		case "esc":
			// Check dirty state before exiting
			if !ep.loading && ep.editor.IsDirty() {
				// TODO: Show confirmation dialog
			}
			return ep, tea.Quit
		}
	}

	// Delegate to editor if loaded
	if !ep.loading {
		var cmd tea.Cmd
		ep.editor, cmd = ep.editor.Update(msg)

		// Update status bar based on dirty state
		if ep.editor.IsDirty() && !ep.saving {
			ep.statusBar.SetMode("Modified")
			ep.statusBar.SetHelpText("Ctrl+S: Save | Esc: Cancel")
		} else if !ep.saving && !ep.saved {
			ep.statusBar.SetMode("Editing")
			ep.statusBar.SetHelpText("Ctrl+S: Save | Esc: Cancel")
		}

		return ep, cmd
	}

	return ep, nil
}

// View renders the EditPage.
func (ep *EditPage) View() string {
	if ep.loading {
		return lipgloss.NewStyle().
			Width(ep.width).
			Height(ep.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render("Loading block...")
	}

	if ep.err != nil {
		errorView := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EF4444")).
			Bold(true).
			Render(fmt.Sprintf("Error: %v", ep.err))

		helpText := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6B7280")).
			Render("\n\nPress ESC to go back")

		return lipgloss.NewStyle().
			Width(ep.width).
			Height(ep.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render(errorView + helpText)
	}

	editorView := ep.editor.View()
	statusView := ep.statusBar.View()

	return lipgloss.JoinVertical(lipgloss.Left, editorView, statusView)
}

// LoadBlock loads a block's content for editing.
func (ep *EditPage) LoadBlock(blockID string) tea.Cmd {
	ep.blockID = blockID
	ep.loading = true
	return ep.loadBlockCmd()
}

// Save saves the current editor content to Notion API.
func (ep *EditPage) Save() tea.Cmd {
	if ep.saving || ep.loading {
		return nil
	}
	ep.saving = true
	ep.saved = false
	return ep.saveCmd()
}

// loadBlockCmd returns a command that loads a block from the API.
func (ep *EditPage) loadBlockCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		block, err := ep.notionClient.GetBlock(ctx, ep.blockID)
		if err != nil {
			return blockLoadedMsg{err: fmt.Errorf("load block: %w", err)}
		}

		// Extract text from block
		text := extractBlockText(block)

		return blockLoadedMsg{
			block: block,
			text:  text,
		}
	}
}

// saveCmd returns a command that saves the editor content to the API.
func (ep *EditPage) saveCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		newText := ep.editor.GetText()

		// Build update request based on block type
		req := buildBlockUpdateRequest(ep.blockType, newText)

		_, err := ep.notionClient.UpdateBlock(ctx, ep.blockID, req)
		if err != nil {
			return blockSavedMsg{err: fmt.Errorf("save block: %w", err)}
		}

		return blockSavedMsg{success: true}
	}
}

// extractBlockText extracts plain text content from a Notion block.
func extractBlockText(block notionapi.Block) string {
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		return richTextToPlainText(b.Paragraph.RichText)
	case *notionapi.Heading1Block:
		return richTextToPlainText(b.Heading1.RichText)
	case *notionapi.Heading2Block:
		return richTextToPlainText(b.Heading2.RichText)
	case *notionapi.Heading3Block:
		return richTextToPlainText(b.Heading3.RichText)
	case *notionapi.BulletedListItemBlock:
		return richTextToPlainText(b.BulletedListItem.RichText)
	case *notionapi.NumberedListItemBlock:
		return richTextToPlainText(b.NumberedListItem.RichText)
	case *notionapi.ToDoBlock:
		return richTextToPlainText(b.ToDo.RichText)
	case *notionapi.ToggleBlock:
		return richTextToPlainText(b.Toggle.RichText)
	case *notionapi.CodeBlock:
		return richTextToPlainText(b.Code.RichText)
	case *notionapi.QuoteBlock:
		return richTextToPlainText(b.Quote.RichText)
	case *notionapi.CalloutBlock:
		return richTextToPlainText(b.Callout.RichText)
	default:
		return ""
	}
}

// richTextToPlainText converts RichText array to plain text string.
func richTextToPlainText(richText []notionapi.RichText) string {
	if len(richText) == 0 {
		return ""
	}

	result := ""
	for _, rt := range richText {
		result += rt.PlainText
	}
	return result
}

// buildBlockUpdateRequest builds a BlockUpdateRequest for the given block type and text.
func buildBlockUpdateRequest(blockType, text string) *notionapi.BlockUpdateRequest {
	richText := []notionapi.RichText{
		{
			Type: notionapi.ObjectTypeText,
			Text: &notionapi.Text{
				Content: text,
			},
		},
	}

	req := &notionapi.BlockUpdateRequest{}

	switch blockType {
	case string(notionapi.BlockTypeParagraph):
		req.Paragraph = &notionapi.Paragraph{
			RichText: richText,
		}
	case string(notionapi.BlockTypeHeading1):
		req.Heading1 = &notionapi.Heading{
			RichText: richText,
		}
	case string(notionapi.BlockTypeHeading2):
		req.Heading2 = &notionapi.Heading{
			RichText: richText,
		}
	case string(notionapi.BlockTypeHeading3):
		req.Heading3 = &notionapi.Heading{
			RichText: richText,
		}
	case string(notionapi.BlockTypeBulletedListItem):
		req.BulletedListItem = &notionapi.ListItem{
			RichText: richText,
		}
	case string(notionapi.BlockTypeNumberedListItem):
		req.NumberedListItem = &notionapi.ListItem{
			RichText: richText,
		}
	case string(notionapi.BlockTypeToDo):
		req.ToDo = &notionapi.ToDo{
			RichText: richText,
		}
	case string(notionapi.BlockTypeToggle):
		req.Toggle = &notionapi.Toggle{
			RichText: richText,
		}
	case string(notionapi.BlockTypeCode):
		req.Code = &notionapi.Code{
			RichText: richText,
			Language: "plain text", // Default language
		}
	case string(notionapi.BlockTypeQuote):
		req.Quote = &notionapi.Quote{
			RichText: richText,
		}
	case string(notionapi.BlockTypeCallout):
		req.Callout = &notionapi.Callout{
			RichText: richText,
		}
	}

	return req
}
