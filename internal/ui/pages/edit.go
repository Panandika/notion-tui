package pages

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jomei/notionapi"

	"github.com/Panandika/notion-tui/internal/ui/components"
)

// blockLoadedMsg is sent when a block has been loaded from the API.
type blockLoadedMsg struct {
	block          notionapi.Block
	text           string
	lastEditedTime time.Time
	err            error
}

// blockSavedMsg is sent when a block has been saved to the API.
type blockSavedMsg struct {
	success        bool
	lastEditedTime time.Time
	err            error
	retryAttempt   int
}

// blockRefreshedMsg is sent when a block has been refreshed from the API.
type blockRefreshedMsg struct {
	block          notionapi.Block
	text           string
	lastEditedTime time.Time
	err            error
}

// retryDelayMsg is sent after a retry delay has elapsed.
type retryDelayMsg struct {
	attempt int
}

// clearSavedIndicatorMsg is sent to clear the "Saved!" indicator.
type clearSavedIndicatorMsg struct{}

// EditPage wraps the BlockEditor component with save logic for managing block editing.
type EditPage struct {
	editor           components.BlockEditor
	statusBar        components.StatusBar
	errorView        *components.ErrorView
	modal            *components.Modal
	pageID           string
	blockID          string
	blockType        string
	originalText     string
	lastEditedTime   time.Time
	loading          bool
	saving           bool
	saved            bool
	err              error
	width            int
	height           int
	notionClient     NotionClient
	showModal        bool
	showError        bool
	retryAttempt     int
	maxRetries       int
	pendingBlockType string // For block type transformation requests
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
		maxRetries:   3,
		showModal:    false,
		showError:    false,
	}
}

// Init initializes the EditPage and loads the block content.
func (ep *EditPage) Init() tea.Cmd {
	return ep.loadBlockCmd()
}

// Update handles messages and returns the updated EditPage and command.
func (ep *EditPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle modal if shown
	if ep.showModal && ep.modal != nil {
		switch msg := msg.(type) {
		case components.ModalResponseMsg:
			ep.showModal = false
			ep.modal = nil
			switch msg.Value {
			case "save":
				// Save and mark that we should quit after save
				ep.pendingBlockType = "quit_after_save" // Use as a flag
				ep.saving = true
				ep.saved = false
				ep.retryAttempt = 0
				ep.statusBar.SetMode("Saving...")
				ep.statusBar.SetSyncStatus(components.StatusSyncing)
				return ep, ep.saveCmd()
			case "discard":
				// Exit without saving
				return ep, tea.Quit
			case "cancel":
				// Continue editing
				return ep, nil
			}
		case components.ModalDismissMsg:
			ep.showModal = false
			ep.modal = nil
			return ep, nil
		default:
			var cmd tea.Cmd
			*ep.modal, cmd = ep.modal.Update(msg)
			return ep, cmd
		}
	}

	// Handle error view if shown
	if ep.showError && ep.errorView != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				// Retry save
				if ep.errorView.IsRetryable() {
					ep.showError = false
					ep.errorView = nil
					ep.err = nil
					ep.saving = true
					ep.statusBar.SetMode("Saving...")
					return ep, ep.saveCmd()
				}
			case "d", "esc":
				// Dismiss error
				ep.showError = false
				ep.errorView = nil
				ep.err = nil
				ep.statusBar.SetMode("Editing")
				return ep, nil
			}
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		ep.width = msg.Width
		ep.height = msg.Height
		ep.statusBar.SetWidth(msg.Width)
		if !ep.loading {
			editorHeight := msg.Height - lipgloss.Height(ep.statusBar.View()) - 4
			ep.editor.SetSize(msg.Width-4, editorHeight)
		}
		if ep.modal != nil {
			ep.modal.SetSize(msg.Width, msg.Height)
		}
		if ep.errorView != nil {
			ep.errorView.SetSize(msg.Width, msg.Height)
		}
		return ep, nil

	case blockLoadedMsg:
		if msg.err != nil {
			ep.loading = false
			ep.err = msg.err
			ep.showError = true
			ep.errorView = &components.ErrorView{}
			*ep.errorView = components.NewErrorView(components.NewErrorViewInput{
				Err:        msg.err,
				Width:      ep.width,
				Height:     ep.height,
				ShowBorder: true,
			})
			return ep, nil
		}

		ep.loading = false
		ep.blockType = string(msg.block.GetType())
		ep.originalText = msg.text
		ep.lastEditedTime = msg.lastEditedTime

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
		ep.statusBar.SetHelpText("Ctrl+S: Save | Ctrl+R: Refresh | Esc: Cancel")
		return ep, ep.editor.Init()

	case blockRefreshedMsg:
		if msg.err != nil {
			ep.err = msg.err
			ep.showError = true
			ep.errorView = &components.ErrorView{}
			*ep.errorView = components.NewErrorView(components.NewErrorViewInput{
				Err:        msg.err,
				Width:      ep.width,
				Height:     ep.height,
				ShowBorder: true,
			})
			return ep, nil
		}

		// Update with fresh content (discards local changes)
		ep.blockType = string(msg.block.GetType())
		ep.originalText = msg.text
		ep.lastEditedTime = msg.lastEditedTime
		ep.editor.SetText(msg.text)
		ep.statusBar.SetMode("Refreshed")

		// Clear refreshed indicator after a moment
		return ep, tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg {
			return clearSavedIndicatorMsg{}
		})

	case components.SaveDraftMsg:
		// User pressed Ctrl+S in editor - save to Notion API
		if ep.saving {
			return ep, nil // Already saving
		}
		ep.saving = true
		ep.saved = false
		ep.retryAttempt = 0
		ep.statusBar.SetMode("Saving...")
		ep.statusBar.SetSyncStatus(components.StatusSyncing)
		return ep, ep.saveCmd()

	case blockSavedMsg:
		ep.saving = false
		if msg.err != nil {
			// Check if we should retry
			if msg.retryAttempt < ep.maxRetries && ep.isTransientError(msg.err) {
				// Calculate exponential backoff delay
				delay := ep.calculateRetryDelay(msg.retryAttempt)
				ep.statusBar.SetMode(fmt.Sprintf("Retrying in %s... (%d/%d)", delay, msg.retryAttempt+1, ep.maxRetries))
				return ep, tea.Tick(delay, func(t time.Time) tea.Msg {
					return retryDelayMsg{attempt: msg.retryAttempt + 1}
				})
			}

			// Max retries exceeded or non-transient error - show error view
			ep.err = msg.err
			ep.showError = true
			ep.errorView = &components.ErrorView{}
			*ep.errorView = components.NewErrorView(components.NewErrorViewInput{
				Err:        msg.err,
				Width:      ep.width,
				Height:     ep.height,
				ShowBorder: true,
			})
			ep.statusBar.SetMode("Error")
			ep.statusBar.SetSyncStatus(components.StatusError)
			return ep, nil
		}

		// Save successful
		ep.saved = true
		ep.retryAttempt = 0
		ep.lastEditedTime = msg.lastEditedTime
		ep.originalText = ep.editor.GetText()
		ep.editor.MarkClean()
		ep.statusBar.SetMode("Saved!")
		ep.statusBar.SetSyncStatus(components.StatusSynced)
		ep.statusBar.UpdateSyncSuccess()

		// Check if we should quit after save (from modal "save and exit")
		if ep.pendingBlockType == "quit_after_save" {
			return ep, tea.Quit
		}

		// If we have a pending block type transformation, apply it now
		if ep.pendingBlockType != "" && ep.pendingBlockType != "quit_after_save" {
			ep.blockType = ep.pendingBlockType
			ep.pendingBlockType = ""
		} else if ep.pendingBlockType != "quit_after_save" {
			ep.pendingBlockType = ""
		}

		// Clear "Saved!" message after a moment
		return ep, tea.Tick(time.Millisecond*1500, func(t time.Time) tea.Msg {
			return clearSavedIndicatorMsg{}
		})

	case retryDelayMsg:
		// Retry delay has elapsed - attempt save again
		ep.retryAttempt = msg.attempt
		ep.saving = true
		ep.statusBar.SetMode("Saving...")
		ep.statusBar.SetSyncStatus(components.StatusSyncing)
		return ep, ep.saveCmd()

	case clearSavedIndicatorMsg:
		if !ep.editor.IsDirty() && !ep.saving {
			ep.statusBar.SetMode("Editing")
		}
		return ep, nil

	case components.CancelEditMsg:
		// User pressed Esc - check if dirty and navigate back
		if !ep.loading && ep.editor.IsDirty() {
			// Show confirmation modal
			ep.showModal = true
			modal := components.NewModal(components.NewModalInput{
				Title:   "Unsaved Changes",
				Message: "You have unsaved changes. What do you want to do?",
				Actions: []components.ModalAction{
					{Label: "Save", Key: "s", Value: "save"},
					{Label: "Discard", Key: "d", Value: "discard"},
					{Label: "Cancel", Key: "c", Value: "cancel"},
				},
				Width:  ep.width,
				Height: ep.height,
			})
			ep.modal = &modal
			return ep, nil
		}
		return ep, tea.Quit

	case tea.KeyMsg:
		// Handle keys at page level before delegating to editor
		switch msg.String() {
		case "ctrl+s":
			if !ep.loading && !ep.saving && !ep.showModal && !ep.showError {
				ep.saving = true
				ep.saved = false
				ep.retryAttempt = 0
				ep.statusBar.SetMode("Saving...")
				ep.statusBar.SetSyncStatus(components.StatusSyncing)
				return ep, ep.saveCmd()
			}
			return ep, nil

		case "ctrl+r":
			// Refresh block from Notion (discard local changes)
			if !ep.loading && !ep.saving && !ep.showModal && !ep.showError {
				ep.statusBar.SetMode("Refreshing...")
				return ep, ep.refreshBlockCmd()
			}
			return ep, nil

		case "esc":
			// Check dirty state before exiting
			if !ep.loading && ep.editor.IsDirty() && !ep.showModal && !ep.showError {
				// Show confirmation modal
				ep.showModal = true
				modal := components.NewModal(components.NewModalInput{
					Title:   "Unsaved Changes",
					Message: "You have unsaved changes. What do you want to do?",
					Actions: []components.ModalAction{
						{Label: "Save", Key: "s", Value: "save"},
						{Label: "Discard", Key: "d", Value: "discard"},
						{Label: "Cancel", Key: "c", Value: "cancel"},
					},
					Width:  ep.width,
					Height: ep.height,
				})
				ep.modal = &modal
				return ep, nil
			} else if !ep.showModal && !ep.showError {
				return ep, tea.Quit
			}
			return ep, nil

		// Block type transformation shortcuts
		case "ctrl+1":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeHeading1))
		case "ctrl+2":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeHeading2))
		case "ctrl+3":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeHeading3))
		case "ctrl+p":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeParagraph))
		case "ctrl+l":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeBulletedListItem))
		case "ctrl+o":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeNumberedListItem))
		case "ctrl+q":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeQuote))
		case "ctrl+k":
			return ep, ep.transformBlockType(string(notionapi.BlockTypeCode))
		}
	}

	// Delegate to editor if loaded and not showing modal/error
	if !ep.loading && !ep.showModal && !ep.showError {
		var cmd tea.Cmd
		ep.editor, cmd = ep.editor.Update(msg)

		// Update status bar based on dirty state
		if ep.editor.IsDirty() && !ep.saving {
			ep.statusBar.SetMode("Modified *")
			ep.statusBar.SetHelpText("Ctrl+S: Save | Ctrl+R: Refresh | Esc: Cancel")
		} else if !ep.saving && !ep.saved {
			ep.statusBar.SetMode("Editing")
			ep.statusBar.SetHelpText("Ctrl+S: Save | Ctrl+R: Refresh | Esc: Cancel")
		}

		return ep, cmd
	}

	return ep, nil
}

// View renders the EditPage.
func (ep *EditPage) View() string {
	// Show modal on top if active
	if ep.showModal && ep.modal != nil {
		// Render editor in background with modal overlay
		baseView := ep.renderBaseView()
		return lipgloss.Place(
			ep.width,
			ep.height,
			lipgloss.Center,
			lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Left, baseView, ep.modal.View()),
		)
	}

	// Show error view if active
	if ep.showError && ep.errorView != nil {
		return ep.errorView.View()
	}

	return ep.renderBaseView()
}

// renderBaseView renders the basic editor view.
func (ep *EditPage) renderBaseView() string {
	if ep.loading {
		return lipgloss.NewStyle().
			Width(ep.width).
			Height(ep.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render("Loading block...")
	}

	// If there's no editor yet (e.g., error during load), just show status bar
	var editorView string
	if ep.editor.BlockID() != "" {
		editorView = ep.editor.View()
	} else {
		editorView = lipgloss.NewStyle().
			Width(ep.width).
			Height(ep.height - 2).
			Render("")
	}

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

		// Extract text and last edited time from block
		text := extractBlockText(block)
		lastEdited := extractLastEditedTime(block)

		return blockLoadedMsg{
			block:          block,
			text:           text,
			lastEditedTime: lastEdited,
		}
	}
}

// refreshBlockCmd returns a command that refreshes a block from the API.
func (ep *EditPage) refreshBlockCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		block, err := ep.notionClient.GetBlock(ctx, ep.blockID)
		if err != nil {
			return blockRefreshedMsg{err: fmt.Errorf("refresh block: %w", err)}
		}

		// Extract text and last edited time from block
		text := extractBlockText(block)
		lastEdited := extractLastEditedTime(block)

		return blockRefreshedMsg{
			block:          block,
			text:           text,
			lastEditedTime: lastEdited,
		}
	}
}

// saveCmd returns a command that saves the editor content to the API.
func (ep *EditPage) saveCmd() tea.Cmd {
	retryAttempt := ep.retryAttempt
	return func() tea.Msg {
		ctx := context.Background()

		newText := ep.editor.GetText()

		// Determine block type to save (use pending if transformation requested)
		blockTypeToSave := ep.blockType
		if ep.pendingBlockType != "" {
			blockTypeToSave = ep.pendingBlockType
		}

		// Build update request based on block type
		req := buildBlockUpdateRequest(blockTypeToSave, newText)

		updatedBlock, err := ep.notionClient.UpdateBlock(ctx, ep.blockID, req)
		if err != nil {
			return blockSavedMsg{
				success:      false,
				err:          fmt.Errorf("save block: %w", err),
				retryAttempt: retryAttempt,
			}
		}

		// Extract last edited time from saved block
		lastEdited := extractLastEditedTime(updatedBlock)

		return blockSavedMsg{
			success:        true,
			lastEditedTime: lastEdited,
			retryAttempt:   retryAttempt,
		}
	}
}

// transformBlockType initiates a block type transformation.
func (ep *EditPage) transformBlockType(newBlockType string) tea.Cmd {
	if ep.loading || ep.saving || ep.showModal || ep.showError {
		return nil
	}

	// Mark the new block type as pending
	ep.pendingBlockType = newBlockType

	// Trigger save with the new block type
	ep.saving = true
	ep.saved = false
	ep.retryAttempt = 0
	ep.statusBar.SetMode(fmt.Sprintf("Converting to %s...", newBlockType))
	ep.statusBar.SetSyncStatus(components.StatusSyncing)

	return ep.saveCmd()
}

// isTransientError determines if an error is transient and should be retried.
func (ep *EditPage) isTransientError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network errors are transient
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "temporary failure") {
		return true
	}

	// Rate limit errors are transient
	if strings.Contains(errStr, "429") || strings.Contains(errStr, "rate limit") {
		return true
	}

	// Server errors (5xx) are transient
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	return false
}

// calculateRetryDelay calculates the retry delay using exponential backoff.
func (ep *EditPage) calculateRetryDelay(attempt int) time.Duration {
	// Exponential backoff: 1s, 2s, 4s
	baseDelay := time.Second
	delay := baseDelay * (1 << uint(attempt))

	// Cap at 10 seconds
	if delay > 10*time.Second {
		delay = 10 * time.Second
	}

	return delay
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

// extractLastEditedTime extracts the last edited time from a Notion block.
func extractLastEditedTime(block notionapi.Block) time.Time {
	// All block types have a LastEditedTime field via the embedded Block struct
	// The field is a *time.Time pointer, so we need to dereference it

	// Type assertion to access the common Block fields
	// Since all block types embed notionapi.Block, we can try to access it
	switch b := block.(type) {
	case *notionapi.ParagraphBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.Heading1Block:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.Heading2Block:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.Heading3Block:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.BulletedListItemBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.NumberedListItemBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.ToDoBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.ToggleBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.CodeBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.QuoteBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	case *notionapi.CalloutBlock:
		if b.LastEditedTime != nil {
			return time.Time(*b.LastEditedTime)
		}
	}
	return time.Now()
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
